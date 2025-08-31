package storage

import (
"context"
"database/sql"
"fmt"
"os"
"path/filepath"
"rsshub/internal/domain"
"sort"
"time"

_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// PostgresRepository реализует интерфейсы domain.FeedRepository и domain.ArticleRepository
type PostgresRepository struct {
db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(connStr string) (*PostgresRepository, error) {
db, err := sql.Open("postgres", connStr)
if err != nil {
return nil, err
}

err = db.Ping()
if err != nil {
db.Close()
return nil, err
}

return &PostgresRepository{db: db}, nil
}

// DB возвращает ссылку на соединение с базой данных
func (r *PostgresRepository) DB() *sql.DB {
return r.db
}

// Close закрывает соединение с базой данных
func (r *PostgresRepository) Close() error {
if r.db != nil {
return r.db.Close()
}
return nil
}

// RunMigrations выполняет SQL миграции для создания или обновления таблиц
func (r *PostgresRepository) RunMigrations(migrationsDir string) error {
// Проверка существования директории
_, err := os.Stat(migrationsDir)
if err != nil {
if os.IsNotExist(err) {
return fmt.Errorf("директория миграций не существует: %s", migrationsDir)
}
return fmt.Errorf("ошибка проверки директории миграций: %w", err)
}

// Получение списка файлов миграций
files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
if err != nil {
return fmt.Errorf("ошибка поиска файлов миграций: %w", err)
}

if len(files) == 0 {
fmt.Println("Миграции не найдены")
return nil
}

// Сортировка файлов по имени
sort.Strings(files)

// Создание таблицы migrations
_, err = r.db.Exec(`
        CREATE TABLE IF NOT EXISTS migrations (
            name TEXT PRIMARY KEY,
            applied_at TIMESTAMP NOT NULL DEFAULT NOW()
        )
    `)
if err != nil {
return fmt.Errorf("ошибка создания таблицы migrations: %w", err)
}

// Получение списка уже выполненных миграций
rows, err := r.db.Query("SELECT name FROM migrations")
if err != nil {
return fmt.Errorf("ошибка получения списка выполненных миграций: %w", err)
}
defer rows.Close()

appliedMigrations := make(map[string]bool)

for rows.Next() {
var name string
if err := rows.Scan(&name); err != nil {
return fmt.Errorf("ошибка чтения имени миграции: %w", err)
}
appliedMigrations[name] = true
}

if err := rows.Err(); err != nil {
return fmt.Errorf("ошибка при итерации по результатам запроса: %w", err)
}

// Выполнение миграций
for _, file := range files {
// Получение имени файла без пути
fileName := filepath.Base(file)

// Проверка, была ли миграция уже применена
if appliedMigrations[fileName] {
fmt.Printf("Миграция %s уже применена\n", fileName)
continue
}

// Чтение содержимого файла
content, err := os.ReadFile(file)
if err != nil {
return fmt.Errorf("ошибка чтения файла %s: %w", file, err)
}

// Начало транзакции
tx, err := r.db.Begin()
if err != nil {
return fmt.Errorf("ошибка начала транзакции: %w", err)
}

// Выполнение SQL-запроса из файла
_, err = tx.Exec(string(content))
if err != nil {
tx.Rollback()
return fmt.Errorf("ошибка выполнения миграции %s: %w", fileName, err)
}

// Запись информации о миграции
_, err = tx.Exec("INSERT INTO migrations (name) VALUES ($1)", fileName)
if err != nil {
tx.Rollback()
return fmt.Errorf("ошибка записи информации о миграции %s: %w", fileName, err)
}

// Фиксация транзакции
if err := tx.Commit(); err != nil {
return fmt.Errorf("ошибка фиксации транзакции: %w", err)
}

fmt.Printf("Миграция %s успешно применена\n", fileName)
}

return nil
}

// AddFeed добавляет новый канал в базу данных
func (r *PostgresRepository) AddFeed(ctx context.Context, feed *domain.Feed) error {
tx, err := r.db.BeginTx(ctx, nil)
if err != nil {
return err
}

query := `
INSERT INTO feeds (created_at, updated_at, name, url)
VALUES (NOW(), NOW(), $1, $2)
`

_, err = tx.ExecContext(ctx, query, feed.Name, feed.URL)
if err != nil {
tx.Rollback()
return err
}

return tx.Commit()
}

// GetFeedByName возвращает канал по имени
func (r *PostgresRepository) GetFeedByName(ctx context.Context, name string) (*domain.Feed, error) {
query := `
SELECT id, created_at, updated_at, name, url
FROM feeds
WHERE name = $1
`

var feed domain.Feed
err := r.db.QueryRowContext(ctx, query, name).Scan(
&feed.ID,
&feed.CreatedAt,
&feed.UpdatedAt,
&feed.Name,
&feed.URL,
)
if err != nil {
return nil, err
}

return &feed, nil
}

// ListFeeds возвращает список каналов с ограничением по количеству
func (r *PostgresRepository) ListFeeds(ctx context.Context, limit int) ([]*domain.Feed, error) {
query := `
SELECT id, created_at, updated_at, name, url
FROM feeds
ORDER BY created_at DESC
LIMIT $1
`
rows, err := r.db.QueryContext(ctx, query, limit)
if err != nil {
return nil, err
}
defer rows.Close()

var results []*domain.Feed
for rows.Next() {
var feed domain.Feed
err = rows.Scan(
&feed.ID,
&feed.CreatedAt,
&feed.UpdatedAt,
&feed.Name,
&feed.URL,
)
if err != nil {
return nil, err
}
results = append(results, &feed)
}

if err := rows.Err(); err != nil {
return nil, fmt.Errorf("error iterating rows: %w", err)
}

return results, nil
}

// DeleteFeed удаляет канал по имени
func (r *PostgresRepository) DeleteFeed(ctx context.Context, name string) error {
query := `
DELETE FROM feeds WHERE name = $1
`
result, err := r.db.ExecContext(ctx, query, name)
if err != nil {
return err
}

rowsAffected, err := result.RowsAffected()
if err != nil {
return err
}

if rowsAffected == 0 {
return fmt.Errorf("канал с именем '%s' не найден", name)
}

return nil
}

// GetOutdatedFeeds получает каналы, которые давно не обновлялись
func (r *PostgresRepository) GetOutdatedFeeds(ctx context.Context, count int) ([]*domain.Feed, error) {
query := `
        SELECT id, created_at, updated_at, name, url
        FROM feeds
        ORDER BY updated_at ASC
        LIMIT $1
    `

rows, err := r.db.QueryContext(ctx, query, count)
if err != nil {
return nil, err
}
defer rows.Close()

var feeds []*domain.Feed
for rows.Next() {
var feed domain.Feed
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

// UpdateFeedTimestamp обновляет время последнего обновления канала
func (r *PostgresRepository) UpdateFeedTimestamp(ctx context.Context, feedID int) error {
query := `
UPDATE feeds SET updated_at = NOW() WHERE id = $1
`
_, err := r.db.ExecContext(ctx, query, feedID)
return err
}

// AddArticle добавляет новую статью в базу данных
func (r *PostgresRepository) AddArticle(ctx context.Context, article *domain.Article) error {
// Проверка входных данных
if article == nil {
return fmt.Errorf("статья не может быть nil")
}

if article.Title == "" || article.Link == "" || article.FeedID == 0 {
return fmt.Errorf("необходимо указать title, link и feed_id")
}

// Используем текущее время, если время публикации не указано
publishedAt := article.PublishedAt
if publishedAt.IsZero() {
publishedAt = time.Now()
}

// SQL-запрос для вставки статьи
query := `
        INSERT INTO articles (
            created_at, updated_at, title, link, published_at, description, feed_id
        ) VALUES (
            NOW(), NOW(), $1, $2, $3, $4, $5
        ) ON CONFLICT (link) DO NOTHING
    `

// Выполнение запроса
result, err := r.db.ExecContext(
ctx,
query,
article.Title,
article.Link,
publishedAt,
article.Description,
article.FeedID,
)

if err != nil {
return fmt.Errorf("ошибка добавления статьи: %w", err)
}

// Проверка, была ли статья действительно добавлена
rowsAffected, err := result.RowsAffected()
if err != nil {
return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
}

if rowsAffected == 0 {
// Статья уже существует, но это не ошибка
return nil
}

return nil
}

// GetArticlesByFeed возвращает статьи канала
func (r *PostgresRepository) GetArticlesByFeed(ctx context.Context, feedName string, limit int) ([]*domain.Article, error) {
// Запрос для получения статей канала без учета регистра
query := `
        SELECT a.id, a.created_at, a.updated_at, a.title, a.link, a.published_at, a.description, a.feed_id
        FROM articles a
        JOIN feeds f ON a.feed_id = f.id
        WHERE LOWER(f.name) = LOWER($1)
        ORDER BY a.published_at DESC
        LIMIT $2
    `

// Выполнение запроса
rows, err := r.db.QueryContext(ctx, query, feedName, limit)
if err != nil {
return nil, fmt.Errorf("ошибка запроса статей: %w", err)
}
defer rows.Close()

// Обработка результатов
var articles []*domain.Article
for rows.Next() {
article := &domain.Article{}
err := rows.Scan(
&article.ID,
&article.CreatedAt,
&article.UpdatedAt,
&article.Title,
&article.Link,
&article.PublishedAt,
&article.Description,
&article.FeedID,
)
if err != nil {
return nil, fmt.Errorf("ошибка сканирования статьи: %w", err)
}
articles = append(articles, article)
}

// Проверка ошибок после цикла
if err := rows.Err(); err != nil {
return nil, fmt.Errorf("ошибка при итерации по статьям: %w", err)
}

return articles, nil
}
