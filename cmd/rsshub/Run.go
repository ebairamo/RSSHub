package rsshub

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"rsshub/internal/models"
	"rsshub/internal/parser"
	"rsshub/internal/storage"
	"sync"
	"syscall"
	"time"
)

// Структура для хранения информации об агрегаторе
type Aggregator struct {
	repo        *storage.PostgresRepository
	interval    time.Duration
	workerCount int

	ticker  *time.Ticker
	jobCh   chan int
	done    chan struct{}
	running bool
	mu      sync.Mutex
}

// Состояние агрегатора для сохранения в файл
type AggregatorState struct {
	Running     bool          `json:"running"`
	Interval    time.Duration `json:"interval"`
	WorkerCount int           `json:"worker_count"`
	PID         int           `json:"pid"`
}

// Путь к файлу состояния
const statePath = "/tmp/rsshub_state.json"

// Глобальный экземпляр агрегатора
var aggregator *Aggregator

// Функции для работы с файлом состояния
func saveAggregatorState() error {
	aggregator.mu.Lock()
	defer aggregator.mu.Unlock()

	state := AggregatorState{
		Running:     aggregator.running,
		Interval:    aggregator.interval,
		WorkerCount: aggregator.workerCount,
		PID:         os.Getpid(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

func loadAggregatorState() (*AggregatorState, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AggregatorState{Running: false}, nil
		}
		return nil, err
	}

	var state AggregatorState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// Проверяем, существует ли процесс
	if state.Running {
		proc, err := os.FindProcess(state.PID)
		if err != nil {
			return &AggregatorState{Running: false}, nil
		}
		// На Unix нужно отправить сигнал 0, чтобы проверить существование процесса
		err = proc.Signal(syscall.Signal(0))
		if err != nil {
			// Процесс не существует
			return &AggregatorState{Running: false}, nil
		}
	}

	return &state, nil
}

// Функция для отправки сигнала запущенному процессу
func signalRunningProcess(signal syscall.Signal) error {
	state, err := loadAggregatorState()
	if err != nil {
		return err
	}

	if !state.Running {
		return fmt.Errorf("фоновый процесс не запущен")
	}

	proc, err := os.FindProcess(state.PID)
	if err != nil {
		return err
	}

	return proc.Signal(signal)
}

func Run() {
	if len(os.Args) < 2 {
		fmt.Println("Ошибка: команда не указана")
		printHelp()
		os.Exit(1)
	}
	comand := os.Args[1]
	fmt.Println(comand)

	switch comand {
	case "fetch":
		runFetch()

	case "add":
		runAdd()

	case "set-interval":
		runSetInterval()

	case "set-workers":
		runSetWorkers()

	case "list":
		runList()

	case "delete":
		runDelete()

	case "articles":
		runArticles()

	case "help":
		printHelp()
	case "url":
		runUrl()
	case "db":
		runDBTest()

	default:
		fmt.Printf("Ошибка: неизвестная команда '%s'\n", comand)
		printHelp()
		os.Exit(1)
	}
}

func runUrl() {
	urlCmd := flag.NewFlagSet("url", flag.ExitOnError)

	url := urlCmd.String("url", "", "URL RSS-канала")
	urlCmd.Parse(os.Args[2:])
	fmt.Printf("Парсинг URL: %s\n", *url)
	if *url == "" {
		fmt.Println("Необходимо указать name и url", *url)
		urlCmd.PrintDefaults()
		os.Exit(1)
	}
	feed, err := parser.ParseFeed(*url)
	if err != nil {
		fmt.Println("!!!!!!!")
	}
	fmt.Printf("Канал: %s\n", feed.Channel.Title)
	fmt.Printf("Описание: %s\n", feed.Channel.Description)
	fmt.Printf("Ссылка: %s\n", feed.Channel.Link)
	fmt.Printf("Количество статей: %d\n\n", len(feed.Channel.Items))
	feeded := feed.Channel.Items
	for i, feedd := range feeded {
		if i == 5 {
			break
		}
		i++
		fmt.Println(feedd.Description)
	}
}

// Функция для вывода справки
func printHelp() {
	fmt.Print(`$ ./rsshub --help

  Usage:
    rsshub COMMAND [OPTIONS]

  Common Commands:
       add             add new RSS feed
       set-interval    set RSS fetch interval
       set-workers     set number of workers
       list            list available RSS feeds
       delete          delete RSS feed
       articles        show latest articles
       fetch           starts the background process that periodically fetches and processes RSS feeds using a worker pool`)
}

// Функция для запуска команды fetch
func runFetch() {
	// Проверяем, не запущен ли уже процесс
	state, err := loadAggregatorState()
	if err != nil {
		fmt.Printf("Ошибка проверки состояния: %v\n", err)
		os.Exit(1)
	}

	if state.Running {
		fmt.Println("Фоновый процесс уже запущен")
		return
	}

	// Создаем репозиторий
	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Println("Ошибка подключения к БД:", err)
		return
	}

	// Устанавливаем значения по умолчанию
	interval := 3 * time.Minute
	workerCount := 3

	// Создаем агрегатор
	aggregator = &Aggregator{
		repo:        repo,
		interval:    interval,
		workerCount: workerCount,
		done:        make(chan struct{}),
		jobCh:       make(chan int, workerCount),
	}

	// Запускаем агрегатор
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = startAggregator(ctx)
	if err != nil {
		fmt.Printf("Ошибка запуска агрегатора: %v\n", err)
		return
	}

	// Сохраняем состояние агрегатора
	err = saveAggregatorState()
	if err != nil {
		fmt.Printf("Ошибка сохранения состояния: %v\n", err)
		// Продолжаем работу, даже если не удалось сохранить состояние
	}

	fmt.Printf("Запущен фоновый процесс для получения каналов (интервал = %v, количество рабочих процессов = %d)\n",
		interval, workerCount)
	fmt.Println("Для остановки процесса нажмите Ctrl+C")

	// Ожидаем сигнал завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			// Останавливаем агрегатор
			stopAggregator()
			// Удаляем файл состояния
			os.Remove(statePath)
			fmt.Println("Изящное завершение работы: агрегатор остановлен")
			return
		case syscall.SIGUSR1:
			// Получаем новый интервал из файла
			newState, err := loadAggregatorState()
			if err != nil {
				fmt.Printf("Ошибка загрузки состояния: %v\n", err)
				continue
			}
			oldInterval := aggregator.interval
			aggregator.SetInterval(newState.Interval)
			fmt.Printf("Интервал получения данных изменился с %v на %v\n", oldInterval, newState.Interval)
		case syscall.SIGUSR2:
			// Получаем новое количество воркеров из файла
			newState, err := loadAggregatorState()
			if err != nil {
				fmt.Printf("Ошибка загрузки состояния: %v\n", err)
				continue
			}
			oldCount := aggregator.workerCount
			err = aggregator.Resize(newState.WorkerCount)
			if err != nil {
				fmt.Printf("Ошибка изменения количества рабочих процессов: %v\n", err)
				continue
			}
			fmt.Printf("Количество рабочих процессов изменилось с %d на %d\n", oldCount, newState.WorkerCount)
		}
	}
}

// Запуск агрегатора
func startAggregator(ctx context.Context) error {
	aggregator.mu.Lock()
	defer aggregator.mu.Unlock()

	if aggregator.running {
		return fmt.Errorf("агрегатор уже запущен")
	}

	// Запускаем тикер
	aggregator.ticker = time.NewTicker(aggregator.interval)
	aggregator.running = true

	// Запускаем воркеров
	for i := 0; i < aggregator.workerCount; i++ {
		go worker(ctx, i)
	}

	// Запускаем основной цикл обработки
	go func() {
		// Сразу запускаем первую обработку
		processFeeds(ctx)

		for {
			select {
			case <-aggregator.ticker.C:
				processFeeds(ctx)
			case <-aggregator.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Остановка агрегатора
func stopAggregator() {
	aggregator.mu.Lock()
	defer aggregator.mu.Unlock()

	if !aggregator.running {
		return
	}

	// Останавливаем тикер
	if aggregator.ticker != nil {
		aggregator.ticker.Stop()
	}

	// Сигнализируем о завершении
	close(aggregator.done)

	// Закрываем канал задач после того, как все горутины завершатся
	time.Sleep(100 * time.Millisecond)
	close(aggregator.jobCh)

	aggregator.running = false
}

// Функция воркера
func worker(ctx context.Context, id int) {
	for {
		select {
		case feedID, ok := <-aggregator.jobCh:
			if !ok {
				return
			}
			processFeed(ctx, id, feedID)
		case <-aggregator.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Обработка всех каналов
func processFeeds(ctx context.Context) {
	// Получаем список каналов для обновления
	feeds, err := getOutdatedFeeds(ctx)
	if err != nil {
		fmt.Printf("Ошибка получения устаревших каналов: %v\n", err)
		return
	}

	fmt.Printf("Получено %d каналов для обновления\n", len(feeds))

	// Отправляем задачи воркерам
	for _, feed := range feeds {
		select {
		case aggregator.jobCh <- feed.ID:
			// Задача отправлена
		case <-aggregator.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Получение устаревших каналов
func getOutdatedFeeds(ctx context.Context) ([]*models.Feed, error) {
	query := `
        SELECT id, created_at, updated_at, name, url
        FROM feeds
        ORDER BY updated_at ASC
        LIMIT $1
    `

	rows, err := aggregator.repo.DB.QueryContext(ctx, query, aggregator.workerCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []*models.Feed
	for rows.Next() {
		var feed models.Feed
		err = rows.Scan(&feed.ID, &feed.CreatedAt, &feed.UpdatedAt, &feed.Name, &feed.URL)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, &feed)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return feeds, nil
}

// Обработка одного канала
func processFeed(ctx context.Context, workerID, feedID int) {
	// Получаем информацию о канале
	var feed models.Feed
	err := aggregator.repo.DB.QueryRowContext(ctx,
		"SELECT id, name, url FROM feeds WHERE id = $1", feedID).
		Scan(&feed.ID, &feed.Name, &feed.URL)
	if err != nil {
		fmt.Printf("Воркер %d: ошибка получения информации о канале %d: %v\n",
			workerID, feedID, err)
		return
	}

	fmt.Printf("Воркер %d: обработка канала %s (%s)\n", workerID, feed.Name, feed.URL)

	// Получаем RSS
	rssFeed, err := parser.ParseFeed(feed.URL)
	if err != nil {
		fmt.Printf("Воркер %d: ошибка парсинга канала %s: %v\n",
			workerID, feed.Name, err)
		return
	}

	// Обрабатываем статьи
	for _, item := range rssFeed.Channel.Items {
		// Парсим дату публикации
		pubDate := time.Now()
		if item.PubDate != "" {
			parsed, err := parsePubDate(item.PubDate)
			if err == nil {
				pubDate = parsed
			}
		}

		// Создаем объект статьи
		article := &models.Article{
			Title:       item.Title,
			Link:        item.Link,
			PublishedAt: pubDate,
			Description: item.Description,
			FeedID:      feedID,
		}

		// Добавляем статью в БД
		err = aggregator.repo.AddArticle(ctx, article)
		if err != nil {
			fmt.Printf("Воркер %d: ошибка добавления статьи %s: %v\n",
				workerID, item.Title, err)
			continue
		}
	}

	// Обновляем время последнего обновления канала
	_, err = aggregator.repo.DB.ExecContext(ctx,
		"UPDATE feeds SET updated_at = NOW() WHERE id = $1", feedID)
	if err != nil {
		fmt.Printf("Воркер %d: ошибка обновления времени канала %s: %v\n",
			workerID, feed.Name, err)
	}

	fmt.Printf("Воркер %d: канал %s обработан, найдено %d статей\n",
		workerID, feed.Name, len(rssFeed.Channel.Items))
}

// Парсинг даты публикации
func parsePubDate(pubDate string) (time.Time, error) {
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, pubDate); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("не удалось распарсить дату: %s", pubDate)
}

func runAdd() {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)

	name := addCmd.String("name", "", "Название RSS-канала")
	url := addCmd.String("url", "", "URL RSS-канала")

	addCmd.Parse(os.Args[2:])

	if *name == "" || *url == "" {
		fmt.Printf("Необходимо указать name и url %s %s\n", *name, *url)
		addCmd.PrintDefaults()
		os.Exit(1)
	}

	// Создание репозитория
	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Println("Ошибка создания бд", err)
		return
	}
	defer repo.Close()

	// Создание объекта канала
	feed := &models.Feed{
		Name:      *name,
		URL:       *url,
		UpdatedAt: time.Now(),
	}

	// Добавление канала в БД
	ctx := context.Background()
	err = repo.AddFeed(ctx, feed)
	if err != nil {
		fmt.Printf("Ошибка добавления в базу данных\n%v\n", err)
		return
	}

	// Всегда выводим сообщение об успешном добавлении
	fmt.Printf("Добавлен новый URL %s с именем %s\n", *url, *name)
}

func runSetInterval() {
	// Создание набора флагов
	intervalCmd := flag.NewFlagSet("set-interval", flag.ExitOnError)

	// Определение флага для интервала
	durationFlag := intervalCmd.String("duration", "3m", "Интервал получения данных (например, 2m, 30s)")

	// Парсинг флагов
	intervalCmd.Parse(os.Args[2:])

	// Проверка, указан ли интервал
	if *durationFlag == "" {
		fmt.Println("Необходимо указать интервал с помощью флага --duration")
		intervalCmd.PrintDefaults()
		os.Exit(1)
	}

	// Проверка, запущен ли агрегатор
	state, err := loadAggregatorState()
	if err != nil {
		fmt.Printf("Ошибка проверки состояния: %v\n", err)
		os.Exit(1)
	}

	if !state.Running {
		fmt.Println("Ошибка: фоновый процесс не запущен. Сначала запустите команду 'fetch'")
		os.Exit(1)
	}

	// Парсинг интервала
	duration, err := time.ParseDuration(*durationFlag)
	if err != nil {
		fmt.Printf("Ошибка парсинга интервала: %v\n", err)
		os.Exit(1)
	}

	// Сохраняем новый интервал в файл состояния
	state.Interval = duration
	data, err := json.Marshal(state)
	if err != nil {
		fmt.Printf("Ошибка сериализации состояния: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(statePath, data, 0644)
	if err != nil {
		fmt.Printf("Ошибка сохранения состояния: %v\n", err)
		os.Exit(1)
	}

	// Отправляем сигнал процессу
	err = signalRunningProcess(syscall.SIGUSR1)
	if err != nil {
		fmt.Printf("Ошибка отправки сигнала: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Интервал получения данных изменился на %v\n", duration)
}

// Добавляем в Aggregator методы для изменения параметров
func (a *Aggregator) SetInterval(newInterval time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.interval = newInterval

	// Если агрегатор запущен, обновляем тикер
	if a.running && a.ticker != nil {
		a.ticker.Reset(newInterval)
	}
}

func (a *Aggregator) Resize(newWorkerCount int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if newWorkerCount <= 0 {
		return fmt.Errorf("количество работников должно быть положительным")
	}

	// Если агрегатор не запущен, просто обновляем значение
	if !a.running {
		a.workerCount = newWorkerCount
		return nil
	}

	// Если нужно увеличить количество воркеров
	if newWorkerCount > a.workerCount {
		// Запускаем дополнительных воркеров
		for i := a.workerCount; i < newWorkerCount; i++ {
			go worker(context.Background(), i)
		}
	}

	// Обновляем размер канала задач
	a.workerCount = newWorkerCount

	return nil
}

func (a *Aggregator) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func runSetWorkers() {
	// Создание набора флагов
	workersCmd := flag.NewFlagSet("set-workers", flag.ExitOnError)

	// Определение флага для количества воркеров
	countFlag := workersCmd.Int("count", 3, "Количество рабочих процессов")

	// Парсинг флагов
	workersCmd.Parse(os.Args[2:])

	// Проверка, указано ли количество воркеров
	if *countFlag <= 0 {
		fmt.Println("Количество рабочих процессов должно быть положительным")
		workersCmd.PrintDefaults()
		os.Exit(1)
	}

	// Проверка, запущен ли агрегатор
	state, err := loadAggregatorState()
	if err != nil {
		fmt.Printf("Ошибка проверки состояния: %v\n", err)
		os.Exit(1)
	}

	if !state.Running {
		fmt.Println("Ошибка: фоновый процесс не запущен. Сначала запустите команду 'fetch'")
		os.Exit(1)
	}

	// Сохраняем новое количество воркеров в файл состояния
	state.WorkerCount = *countFlag
	data, err := json.Marshal(state)
	if err != nil {
		fmt.Printf("Ошибка сериализации состояния: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(statePath, data, 0644)
	if err != nil {
		fmt.Printf("Ошибка сохранения состояния: %v\n", err)
		os.Exit(1)
	}

	// Отправляем сигнал процессу
	err = signalRunningProcess(syscall.SIGUSR2)
	if err != nil {
		fmt.Printf("Ошибка отправки сигнала: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Количество рабочих процессов изменилось на %d\n", *countFlag)
}

// Функция для команды list
func runList() {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	limit := listCmd.Int("limit", 10, "")

	listCmd.Parse(os.Args[2:])

	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Println("Ошибка создания бд", err)
		return
	}
	defer repo.Close()
	ctx := context.Background()
	feeds, err := repo.ListFeeds(ctx, *limit)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("# Доступные RSS-каналы")
	for i, feed := range feeds {
		fmt.Printf("%d. Название: %s\n", i+1, feed.Name)
		fmt.Printf("   URL: %s\n", feed.URL)
		fmt.Printf("   Добавлено: %s\n\n", feed.CreatedAt.Format("2006-01-02 15:04"))
	}
}

// Функция для команды delete
func runDelete() {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)

	name := deleteCmd.String("name", "", "Название RSS-канала")

	deleteCmd.Parse(os.Args[2:])

	if *name == "" {
		deleteCmd.PrintDefaults()
		os.Exit(1)
	}

	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Println("Ошибка создания бд", err)
		return
	}
	defer repo.Close()
	ctx := context.Background()
	err = repo.DeleteFeed(ctx, *name)
	if err != nil {
		fmt.Printf("ошибка: канал с именем %s не найден\n", *name)
		os.Exit(1)
	}
}

func runArticles() {
	articlesCmd := flag.NewFlagSet("articles", flag.ExitOnError)
	feedNameFlag := articlesCmd.String("feed-name", "", "Название RSS канала")
	numFlag := articlesCmd.Int("num", 3, "Лимит статей")

	articlesCmd.Parse(os.Args[2:])

	if *feedNameFlag == "" {
		fmt.Println("Необходимо указать имя канала (--feed-name)")
		articlesCmd.PrintDefaults()
		os.Exit(1)
	}

	// Подключение к базе данных
	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Printf("Ошибка подключения к БД: %v\n", err)
		return
	}
	defer repo.Close()

	// Получение статей
	ctx := context.Background()
	articles, err := repo.GetArticlesByFeed(ctx, *feedNameFlag, *numFlag)
	if err != nil {
		fmt.Printf("Ошибка получения статей: %v\n", err)
		return
	}

	// Вывод заголовка
	fmt.Printf("Источник: %s\n\n", *feedNameFlag)

	// Вывод статей
	if len(articles) == 0 {
		fmt.Println("Статьи не найдены")
		return
	}

	// Исправленный цикл вывода статей с правильной нумерацией
	for i, article := range articles {
		fmt.Printf("%d. [%s] %s\n", i+1, article.PublishedAt.Format("2006-01-02"), article.Title)
		fmt.Printf("   %s\n\n", article.Link)
	}
}

func runDBTest() {
	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Println("Ошибка создания бд", err)
		return
	}
	defer repo.Close()

	// Тестовый запрос
	var version string

	err = repo.DB.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		fmt.Printf("Ошибка запроса: %v\n", err)
		return
	}

	fmt.Printf("Успешное подключение! Версия PostgreSQL: %s\n", version)
	err = repo.RunMigrations("./migrations")
	if err != nil {
		fmt.Println(err)
	}
}
