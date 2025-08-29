package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"rsshub/internal/models"
	"rsshub/internal/parser"
	"rsshub/internal/storage"
	"syscall"
)

// Другие необходимые импорты

func main() {
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
	for _, feedd := range feeded {
		fmt.Println(feedd)
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

// Функция для команды fetch
func runFetch() {
	// Не принимаем параметры URL

	// Выводим сообщение о запуске
	fmt.Println("Запущен фоновый процесс для получения каналов (интервал = 3 минуты, количество рабочих процессов = 3)")

	// Заглушка для будущей реализации
	fmt.Println("Имитация работы фонового процесса...")
	fmt.Println("Для остановки процесса нажмите Ctrl+C")

	// Простая имитация бесконечного цикла с возможностью прерывания
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Изящное завершение работы: агрегатор остановлен")
}

// Функция для команды add
func runAdd() {
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)

	name := addCmd.String("name", "", "Название RSS-канала")
	url := addCmd.String("url", "", "URL RSS-канала")

	addCmd.Parse(os.Args[2:])

	if *name == "" || *url == "" {
		fmt.Println("Необходимо указать name и url", *name, *url)
		addCmd.PrintDefaults()
		os.Exit(1)
	}
	feed := &models.Feed{
		Name: *name,
		URL:  *url,
	}

	fmt.Printf("Добавлен новый URL %s с именем %s\n", *url, *name)
	repo, err := storage.NewPostgresRepository("postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable")
	if err != nil {
		fmt.Println("Ошибка создания бд", err)
		return
	}
	defer repo.Close()
	ctx := context.Background()
	err = repo.AddFeed(ctx, feed)
	if err != nil {
		fmt.Println(err)
	}
	// 6. Здесь будет код для добавления канала в БД
}

// Функция для команды set-interval
func runSetInterval() {

	intervalCmd := flag.NewFlagSet("set-interval", flag.ExitOnError)
	intervalFlag := intervalCmd.Duration("interval", 3, "Интервал между RSS")
	intervalCmd.Parse(os.Args[2:])

	if *intervalFlag <= 0 {
		fmt.Println("Укажите время ожидания")
		intervalCmd.PrintDefaults()
		os.Exit(1)
	}
	fmt.Printf("Интервал получения данных изменился с 3 на %d", *intervalFlag)

	// 1. Создайте набор флагов для команды set-interval
	// Используйте flag.NewFlagSet()

	// 2. Определите флаг для нового интервала
	// Например, intervalFlag := setIntervalCmd.Duration()

	// 3. Распарсите флаги
	// Используйте setIntervalCmd.Parse()

	// 4. Проверьте, что указан интервал
	// Если нет, выведите ошибку и справку по команде

	// 5. Выведите сообщение об изменении интервала
	// Например: "Интервал получения данных изменился с X на Y"

	// 6. Здесь будет код для изменения интервала в работающем агрегаторе
}

// Функция для команды set-workers
func runSetWorkers() {

	workersCmd := flag.NewFlagSet("set-workers", flag.ExitOnError)

	workersFlag := workersCmd.Int("count", 3, "Количество работников")

	workersCmd.Parse(os.Args[2:])

	if *workersFlag <= 0 {
		fmt.Println("Укажите количество работников")
		workersCmd.PrintDefaults()
		os.Exit(1)
	}
	fmt.Printf("Количество рабочих процессов изменилось с 3 на %d\n", *workersFlag)
	// 1. Создайте набор флагов для команды set-workers
	// Используйте flag.NewFlagSet()

	// 2. Определите флаг для нового количества воркеров
	// Например, workersFlag := setWorkersCmd.Int()

	// 3. Распарсите флаги
	// Используйте setWorkersCmd.Parse()

	// 4. Проверьте, что указано количество воркеров
	// Если нет, выведите ошибку и справку по команде

	// 5. Выведите сообщение об изменении количества воркеров
	// Например: "Количество рабочих процессов изменилось с X на Y"

	// 6. Здесь будет код для изменения количества воркеров в работающем агрегаторе
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

	// 1. Создайте набор флагов для команды list
	// Используйте flag.NewFlagSet()

	// 2. Определите флаг для ограничения количества выводимых каналов
	// Например, numFlag := listCmd.Int()

	// 3. Распарсите флаги
	// Используйте listCmd.Parse()

	// 4. Выведите заголовок списка каналов
	// Например: "# Доступные RSS-каналы"

	// 5. Здесь будет код для получения списка каналов из БД

	// 6. Выведите информацию о каждом канале
	// Например: "1. Название: NAME\n URL: URL\n Добавлено: DATE"
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
	repo.DeleteFeed(ctx, *name)
	// 6. Здесь будет код для удаления канала из БД
}

// Функция для команды articles
func runArticles() {

	articlesCmd := flag.NewFlagSet("articles", flag.ExitOnError)
	feedNameFlag := articlesCmd.String("feed-name", "", "Название RSS канала")
	numFlag := articlesCmd.Int("num", 5, "Лимит статей")

	articlesCmd.Parse(os.Args[2:])

	if *feedNameFlag == "" {
		articlesCmd.PrintDefaults()
		os.Exit(1)
	}
	fmt.Printf("Источник: %s %d \n", *feedNameFlag, *numFlag)

	fmt.Println("1. [2025-06-18] Apple анонсирует новые чипы M4 для MacBook Pro")
	fmt.Println("   https://techcrunch.com/apple-announces-m4/")
	fmt.Println("")
	fmt.Println("2. [2025-06-17] OpenAI запускает GPT-5 с мультимодальными возможностями")
	fmt.Println("   https://techcrunch.com/openai-launches-gpt-5/")
	fmt.Println("")
	fmt.Println("3. [2025-06-16] Google представляет новые инструменты для обеспечения конфиденциальности")
	fmt.Println("   https://techcrunch.com/google-privacy-io-2025/")

	// 1. Создайте набор флагов для команды articles
	// Используйте flag.NewFlagSet()

	// 2. Определите флаги для имени канала и количества статей
	// Например, feedNameFlag := articlesCmd.String(), numFlag := articlesCmd.Int()

	// 3. Распарсите флаги
	// Используйте articlesCmd.Parse()

	// 4. Проверьте, что указано имя канала
	// Если нет, выведите ошибку и справку по команде

	// 5. Выведите заголовок списка статей
	// Например: "Источник: FEED_NAME"

	// 6. Здесь будет код для получения статей из БД

	// 7. Выведите информацию о каждой статье
	// Например: "1. [DATE] TITLE\n URL"
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
