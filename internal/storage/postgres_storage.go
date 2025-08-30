package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"rsshub/internal/models"

	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// ConnectToDB устанавливает соединение с базой данных
func ConnectToDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// PostgresRepository структура для работы с PostgreSQL
type PostgresRepository struct {
	DB *sql.DB
	// Поле для хранения соединения с базой данных
	// Используйте *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(connStr string) (*PostgresRepository, error) {

	DB, err := ConnectToDB(connStr)
	if err != nil {
		return nil, err
	}
	// Создание репозитория
	repo := &PostgresRepository{
		DB: DB,
	}

	// Использование методов репозитория

	// 1. Установите соединение с БД через ConnectToDB

	// 2. Создайте новый экземпляр PostgresRepository

	// 3. Верните репозиторий и nil или nil и ошибку
	return repo, nil
}

// Close закрывает соединение с базой данных
func (r *PostgresRepository) Close() {

	// 1. Проверьте, что соединение не nil

	// 2. Закройте соединение
}

// RunMigrations выполняет SQL миграции для создания или обновления таблиц
func (r *PostgresRepository) RunMigrations(migrationsDir string) error {
	// 1. Проверка существования директории
	_, err := os.Stat(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("директория миграций не существует: %s", migrationsDir)
		}
		return fmt.Errorf("ошибка проверки директории миграций: %w", err)
	}

	// 2. Получение списка файлов миграций
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("ошибка поиска файлов миграций: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("Миграции не найдены")
		return nil
	}
	fmt.Println(files)
	// Сортировка файлов по имени
	sort.Strings(files)

	// 3. Создание таблицы migrations
	_, err = r.DB.Exec(`
        CREATE TABLE IF NOT EXISTS migrations (
            name TEXT PRIMARY KEY,
            applied_at TIMESTAMP NOT NULL DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы migrations: %w", err)
	}

	// 4. Получение списка уже выполненных миграций
	rows, err := r.DB.Query("SELECT name FROM migrations")
	if err != nil {
		return fmt.Errorf("ошибка получения списка выполненных миграций: %w", err)
	}
	fmt.Println(rows)
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

	// 5. Выполнение миграций
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
		fmt.Println(string(content))
		if err != nil {
			return fmt.Errorf("ошибка чтения файла %s: %w", file, err)
		}

		// Начало транзакции
		tx, err := r.DB.Begin()
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

	// 6. Возврат результата
	return nil
}

// AddFeed добавляет новый канал в базу данных
func (r *PostgresRepository) AddFeed(ctx context.Context, feed *models.Feed) error {
	tx, err := r.DB.Begin()
	if err != nil {
		fmt.Println("начала транзакции")
		return err
	}

	query := `
	INSERT INTO feeds (created_at, updated_at, name, url)
VALUES (NOW(), NOW(), $1, $2)
	`

	_, err = tx.Exec(query, feed.Name, feed.URL)
	if err != nil {
		fmt.Println("ошибка добавления в базу данных")
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// 1. Подготовьте SQL-запрос для вставки (INSERT INTO feeds)

	// 2. Выполните запрос с параметрами из feed

	// 3. Обработайте возможную ошибку (например, если имя канала не уникально)

	// 4. Верните nil или ошибку
	return nil
}

// GetFeedByName возвращает канал по имени
func (r *PostgresRepository) GetFeedByName(ctx context.Context, name string) (*models.Feed, error) {
	// 1. Подготовьте SQL-запрос для выборки (SELECT * FROM feeds WHERE name = ?)

	// 2. Выполните запрос с параметром name

	// 3. Проверьте, найдена ли запись

	// 4. Если запись найдена, создайте объект Feed из результата

	// 5. Верните feed и nil или nil и ошибку
	return nil, nil
}

// ListFeeds возвращает список каналов с ограничением по количеству
func (r *PostgresRepository) ListFeeds(ctx context.Context, limit int) ([]*models.Feed, error) {
	query := `
		SELECT id, created_at, updated_at, name,url
		FROM feeds
		LIMIT $1
		`
	rows, err := r.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*models.Feed
	for rows.Next() {
		var data models.Feed
		err = rows.Scan(
			&data.ID,
			&data.CreatedAt,
			&data.UpdatedAt,
			&data.Name,
			&data.URL,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &data)

	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	// 1. Подготовьте SQL-запрос для выборки с лимитом

	// 2. Выполните запрос

	// 3. Создайте слайс для результатов

	// 4. Пройдите по результатам и добавьте каждый канал в слайс

	// 5. Верните слайс и nil или nil и ошибку
	return results, nil
}

// DeleteFeed удаляет канал по имени
func (r *PostgresRepository) DeleteFeed(ctx context.Context, name string) error {
	query := `
		DELETE FROM feeds WHERE name = $1
	`
	result, err := r.DB.ExecContext(ctx, query, name)
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

	// 1. Подготовьте SQL-запрос для удаления (DELETE FROM feeds WHERE name = ?)

	// 2. Выполните запрос с параметром name

	// 3. Проверьте, была ли удалена запись

	// 4. Верните nil или ошибку
	return nil
}

// GetOutdatedFeeds получает каналы, которые давно не обновлялись
func (r *PostgresRepository) GetOutdatedFeeds(ctx context.Context, count int) ([]*models.Feed, error) {
	// 1. Подготовьте SQL-запрос для выборки каналов, отсортированных по updated_at

	// 2. Выполните запрос с параметром count

	// 3. Создайте слайс для результатов

	// 4. Пройдите по результатам и добавьте каждый канал в слайс

	// 5. Верните слайс и nil или nil и ошибку
	return nil, nil
}

// UpdateFeedTimestamp обновляет время последнего обновления канала
func (r *PostgresRepository) UpdateFeedTimestamp(ctx context.Context, feedID string) error {
	// 1. Подготовьте SQL-запрос для обновления (UPDATE feeds SET updated_at = ? WHERE id = ?)

	// 2. Выполните запрос с текущим временем и feedID

	// 3. Верните nil или ошибку
	return nil
}

// AddArticle добавляет новую статью в базу данных
func (r *PostgresRepository) AddArticle(ctx context.Context, article *models.Article) error {
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
	result, err := r.DB.ExecContext(
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

func (r *PostgresRepository) GetArticlesByFeed(ctx context.Context, feedName string, limit int) ([]*models.Article, error) {
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
	rows, err := r.DB.QueryContext(ctx, query, feedName, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса статей: %w", err)
	}
	defer rows.Close()

	// Обработка результатов
	var articles []*models.Article
	for rows.Next() {
		article := &models.Article{}
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
