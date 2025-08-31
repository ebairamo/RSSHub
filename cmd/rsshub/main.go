package main

import (
"context"
"flag"
"fmt"
"os"
"os/signal"
"rsshub/internal/adapters/parser"
"rsshub/internal/adapters/storage"
"rsshub/internal/application"
"rsshub/internal/domain"
"syscall"
"time"
)

var (
// Параметры по умолчанию
defaultInterval    = 3 * time.Minute
defaultWorkerCount = 3
dbConnectionString = "postgres://postgres:changeme@localhost:5432/rsshub?sslmode=disable"
)

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
fmt.Println("Необходимо указать url")
urlCmd.PrintDefaults()
os.Exit(1)
}

rssParser := parser.NewRSSParser()
feed, err := rssParser.ParseFeed(*url)
if err != nil {
fmt.Println("Ошибка при парсинге:", err)
os.Exit(1)
}

fmt.Printf("Канал: %s\n", feed.Channel.Title)
fmt.Printf("Описание: %s\n", feed.Channel.Description)
fmt.Printf("Ссылка: %s\n", feed.Channel.Link)
fmt.Printf("Количество статей: %d\n\n", len(feed.Channel.Items))

for i, item := range feed.Channel.Items {
if i == 5 {
break
}
fmt.Printf("%d. %s\n", i+1, item.Description)
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
// Создаем IPC менеджер
ipcManager := application.NewIPCManager()

// Проверяем, не запущен ли уже процесс
state, err := ipcManager.LoadState()
if err != nil {
fmt.Printf("Ошибка проверки состояния: %v\n", err)
os.Exit(1)
}

if state.Running {
fmt.Println("Фоновый процесс уже запущен")
return
}

// Создаем репозиторий
repo, err := storage.NewPostgresRepository(dbConnectionString)
if err != nil {
fmt.Println("Ошибка подключения к БД:", err)
return
}

// Создаем парсер
rssParser := parser.NewRSSParser()

// Создаем агрегатор
aggregator := application.NewRSSAggregator(repo, rssParser, defaultInterval, defaultWorkerCount)

// Запускаем агрегатор
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

err = aggregator.Start(ctx)
if err != nil {
fmt.Printf("Ошибка запуска агрегатора: %v\n", err)
return
}

// Сохраняем состояние агрегатора
state = &domain.AggregatorState{
Running:     true,
Interval:    defaultInterval,
WorkerCount: defaultWorkerCount,
PID:         os.Getpid(),
}

err = ipcManager.SaveState(state)
if err != nil {
fmt.Printf("Ошибка сохранения состояния: %v\n", err)
// Продолжаем работу, даже если не удалось сохранить состояние
}

fmt.Printf("Запущен фоновый процесс для получения каналов (интервал = %v, количество рабочих процессов = %d)\n",
defaultInterval, defaultWorkerCount)
fmt.Println("Для остановки процесса нажмите Ctrl+C")

// Ожидаем сигнал завершения
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

for {
sig := <-sigCh
switch sig {
case syscall.SIGINT, syscall.SIGTERM:
// Останавливаем агрегатор
aggregator.Stop()
// Удаляем файл состояния
os.Remove(application.StatePath)
fmt.Println("Изящное завершение работы: агрегатор остановлен")
return
case syscall.SIGUSR1:
// Получаем новый интервал из файла
newState, err := ipcManager.LoadState()
if err != nil {
fmt.Printf("Ошибка загрузки состояния: %v\n", err)
continue
}
oldInterval := state.Interval
aggregator.SetInterval(newState.Interval)
state.Interval = newState.Interval
fmt.Printf("Интервал получения данных изменился с %v на %v\n", oldInterval, newState.Interval)
case syscall.SIGUSR2:
// Получаем новое количество воркеров из файла
newState, err := ipcManager.LoadState()
if err != nil {
fmt.Printf("Ошибка загрузки состояния: %v\n", err)
continue
}
oldCount := state.WorkerCount
err = aggregator.Resize(newState.WorkerCount)
if err != nil {
fmt.Printf("Ошибка изменения количества рабочих процессов: %v\n", err)
continue
}
state.WorkerCount = newState.WorkerCount
fmt.Printf("Количество рабочих процессов изменилось с %d на %d\n", oldCount, newState.WorkerCount)
}
}
}

func runAdd() {
addCmd := flag.NewFlagSet("add", flag.ExitOnError)

name := addCmd.String("name", "", "Название RSS-канала")
url := addCmd.String("url", "", "URL RSS-канала")

addCmd.Parse(os.Args[2:])

if *name == "" || *url == "" {
fmt.Printf("Необходимо указать name и url\n")
addCmd.PrintDefaults()
os.Exit(1)
}

// Создание репозитория
repo, err := storage.NewPostgresRepository(dbConnectionString)
if err != nil {
fmt.Println("Ошибка создания бд", err)
return
}
defer repo.Close()

// Создание объекта канала
feed := &domain.Feed{
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

// Создаем IPC менеджер
ipcManager := application.NewIPCManager()

// Проверка, запущен ли агрегатор
state, err := ipcManager.LoadState()
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
oldInterval := state.Interval
state.Interval = duration
err = ipcManager.SaveState(state)
if err != nil {
fmt.Printf("Ошибка сохранения состояния: %v\n", err)
os.Exit(1)
}

// Отправляем сигнал процессу
err = ipcManager.SignalProcess(state, int(syscall.SIGUSR1))
if err != nil {
fmt.Printf("Ошибка отправки сигнала: %v\n", err)
os.Exit(1)
}

fmt.Printf("Интервал получения данных изменился с %v на %v\n", oldInterval, duration)
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

// Создаем IPC менеджер
ipcManager := application.NewIPCManager()

// Проверка, запущен ли агрегатор
state, err := ipcManager.LoadState()
if err != nil {
fmt.Printf("Ошибка проверки состояния: %v\n", err)
os.Exit(1)
}

if !state.Running {
fmt.Println("Ошибка: фоновый процесс не запущен. Сначала запустите команду 'fetch'")
os.Exit(1)
}

// Сохраняем новое количество воркеров в файл состояния
oldCount := state.WorkerCount
state.WorkerCount = *countFlag
err = ipcManager.SaveState(state)
if err != nil {
fmt.Printf("Ошибка сохранения состояния: %v\n", err)
os.Exit(1)
}

// Отправляем сигнал процессу
err = ipcManager.SignalProcess(state, int(syscall.SIGUSR2))
if err != nil {
fmt.Printf("Ошибка отправки сигнала: %v\n", err)
os.Exit(1)
}

fmt.Printf("Количество рабочих процессов изменилось с %d на %d\n", oldCount, *countFlag)
}

// Функция для команды list
func runList() {
listCmd := flag.NewFlagSet("list", flag.ExitOnError)
numFlag := listCmd.Int("num", 10, "Количество каналов для отображения")
listCmd.Parse(os.Args[2:])

repo, err := storage.NewPostgresRepository(dbConnectionString)
if err != nil {
fmt.Println("Ошибка создания бд", err)
return
}
defer repo.Close()

ctx := context.Background()
feeds, err := repo.ListFeeds(ctx, *numFlag)
if err != nil {
fmt.Println(err)
return
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
fmt.Println("Необходимо указать название канала с помощью флага --name")
deleteCmd.PrintDefaults()
os.Exit(1)
}

repo, err := storage.NewPostgresRepository(dbConnectionString)
if err != nil {
fmt.Println("Ошибка создания бд", err)
return
}
defer repo.Close()

ctx := context.Background()
err = repo.DeleteFeed(ctx, *name)
if err != nil {
fmt.Printf("Ошибка: %v\n", err)
os.Exit(1)
}

fmt.Printf("Канал с именем '%s' успешно удален\n", *name)
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
repo, err := storage.NewPostgresRepository(dbConnectionString)
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

// Цикл вывода статей с правильной нумерацией
for i, article := range articles {
fmt.Printf("%d. [%s] %s\n", i+1, article.PublishedAt.Format("2006-01-02"), article.Title)
fmt.Printf("   %s\n\n", article.Link)
}
}

func runDBTest() {
repo, err := storage.NewPostgresRepository(dbConnectionString)
if err != nil {
fmt.Println("Ошибка создания бд", err)
return
}
defer repo.Close()

// Тестовый запрос
var version string
err = repo.DB().QueryRow("SELECT version()").Scan(&version)
if err != nil {
fmt.Printf("Ошибка запроса: %v\n", err)
return
}

fmt.Printf("Успешное подключение! Версия PostgreSQL: %s\n", version)
err = repo.RunMigrations("./migrations")
if err != nil {
fmt.Println(err)
} else {
fmt.Println("Миграции успешно выполнены")
}
}
