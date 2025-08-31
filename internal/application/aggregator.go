package application

import (
"context"
"fmt"
"rsshub/internal/domain"
"sync"
"time"
)

// RSSAggregator реализует интерфейс domain.Aggregator
type RSSAggregator struct {
repo        domain.FeedRepository
parser      domain.RSSParser
interval    time.Duration
workerCount int

ticker  *time.Ticker
jobCh   chan int
done    chan struct{}
running bool
mu      sync.Mutex
}

// NewRSSAggregator создает новый экземпляр RSSAggregator
func NewRSSAggregator(repo domain.FeedRepository, parser domain.RSSParser, interval time.Duration, workerCount int) *RSSAggregator {
return &RSSAggregator{
repo:        repo,
parser:      parser,
interval:    interval,
workerCount: workerCount,
done:        make(chan struct{}),
}
}

// Start запускает агрегатор
func (a *RSSAggregator) Start(ctx context.Context) error {
a.mu.Lock()
defer a.mu.Unlock()

if a.running {
return fmt.Errorf("агрегатор уже запущен")
}

// Запускаем тикер
a.ticker = time.NewTicker(a.interval)
a.running = true
a.jobCh = make(chan int, a.workerCount)

// Запускаем воркеров
for i := 0; i < a.workerCount; i++ {
go a.worker(ctx, i)
}

// Запускаем основной цикл обработки
go func() {
// Сразу запускаем первую обработку
a.processFeeds(ctx)

for {
select {
case <-a.ticker.C:
a.processFeeds(ctx)
case <-a.done:
return
case <-ctx.Done():
return
}
}
}()

return nil
}

// Stop останавливает агрегатор
func (a *RSSAggregator) Stop() error {
a.mu.Lock()
defer a.mu.Unlock()

if !a.running {
return nil
}

// Останавливаем тикер
if a.ticker != nil {
a.ticker.Stop()
}

// Сигнализируем о завершении
close(a.done)

// Закрываем канал задач после того, как все горутины завершатся
time.Sleep(100 * time.Millisecond)
close(a.jobCh)

a.running = false
return nil
}

// SetInterval изменяет интервал обновления
func (a *RSSAggregator) SetInterval(d time.Duration) {
a.mu.Lock()
defer a.mu.Unlock()

a.interval = d

// Если агрегатор запущен, обновляем тикер
if a.running && a.ticker != nil {
a.ticker.Reset(d)
}
}

// Resize изменяет количество воркеров
func (a *RSSAggregator) Resize(workers int) error {
a.mu.Lock()
defer a.mu.Unlock()

if workers <= 0 {
return fmt.Errorf("количество работников должно быть положительным")
}

// Если агрегатор не запущен, просто обновляем значение
if !a.running {
a.workerCount = workers
return nil
}

// Если нужно увеличить количество воркеров
if workers > a.workerCount {
// Запускаем дополнительных воркеров
for i := a.workerCount; i < workers; i++ {
go a.worker(context.Background(), i)
}
}

// Обновляем размер канала задач
a.workerCount = workers

return nil
}

// IsRunning возвращает статус агрегатора
func (a *RSSAggregator) IsRunning() bool {
a.mu.Lock()
defer a.mu.Unlock()
return a.running
}

// worker обрабатывает задачи из канала
func (a *RSSAggregator) worker(ctx context.Context, id int) {
for {
select {
case feedID, ok := <-a.jobCh:
if !ok {
return
}
a.processFeed(ctx, id, feedID)
case <-a.done:
return
case <-ctx.Done():
return
}
}
}

// processFeeds получает и обрабатывает каналы
func (a *RSSAggregator) processFeeds(ctx context.Context) {
// Получаем список каналов для обновления
feeds, err := a.repo.GetOutdatedFeeds(ctx, a.workerCount)
if err != nil {
fmt.Printf("Ошибка получения устаревших каналов: %v\n", err)
return
}

fmt.Printf("Получено %d каналов для обновления\n", len(feeds))

// Отправляем задачи воркерам
for _, feed := range feeds {
select {
case a.jobCh <- feed.ID:
// Задача отправлена
case <-a.done:
return
case <-ctx.Done():
return
}
}
}

// processFeed обрабатывает один канал
func (a *RSSAggregator) processFeed(ctx context.Context, workerID, feedID int) {
// Получаем информацию о канале
var feed domain.Feed
err := a.repo.DB().QueryRowContext(ctx,
"SELECT id, name, url FROM feeds WHERE id = $1", feedID).
Scan(&feed.ID, &feed.Name, &feed.URL)
if err != nil {
fmt.Printf("Воркер %d: ошибка получения информации о канале %d: %v\n",
workerID, feedID, err)
return
}

fmt.Printf("Воркер %d: обработка канала %s (%s)\n", workerID, feed.Name, feed.URL)

// Получаем RSS
rssFeed, err := a.parser.ParseFeed(feed.URL)
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
parsed, err := a.parsePubDate(item.PubDate)
if err == nil {
pubDate = parsed
}
}

// Создаем объект статьи
article := &domain.Article{
Title:       item.Title,
Link:        item.Link,
PublishedAt: pubDate,
Description: item.Description,
FeedID:      feedID,
}

// Добавляем статью в БД
err = a.repo.(domain.ArticleRepository).AddArticle(ctx, article)
if err != nil {
fmt.Printf("Воркер %d: ошибка добавления статьи %s: %v\n",
workerID, item.Title, err)
continue
}
}

// Обновляем время последнего обновления канала
err = a.repo.UpdateFeedTimestamp(ctx, feedID)
if err != nil {
fmt.Printf("Воркер %d: ошибка обновления времени канала %s: %v\n",
workerID, feed.Name, err)
}

fmt.Printf("Воркер %d: канал %s обработан, найдено %d статей\n",
workerID, feed.Name, len(rssFeed.Channel.Items))
}

// parsePubDate парсит дату публикации
func (a *RSSAggregator) parsePubDate(pubDate string) (time.Time, error) {
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
