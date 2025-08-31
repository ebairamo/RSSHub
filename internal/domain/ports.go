package domain

import (
"context"
"database/sql"
"time"
)

// FeedRepository определяет интерфейс для работы с хранилищем каналов
type FeedRepository interface {
AddFeed(ctx context.Context, feed *Feed) error
GetFeedByName(ctx context.Context, name string) (*Feed, error)
ListFeeds(ctx context.Context, limit int) ([]*Feed, error)
DeleteFeed(ctx context.Context, name string) error
GetOutdatedFeeds(ctx context.Context, count int) ([]*Feed, error)
UpdateFeedTimestamp(ctx context.Context, feedID int) error
Close() error
DB() *sql.DB
}

// ArticleRepository определяет интерфейс для работы с хранилищем статей
type ArticleRepository interface {
AddArticle(ctx context.Context, article *Article) error
GetArticlesByFeed(ctx context.Context, feedName string, limit int) ([]*Article, error)
}

// RSSParser определяет интерфейс для парсинга RSS
type RSSParser interface {
ParseFeed(url string) (*RSS, error)
}

// Aggregator определяет интерфейс для работы с агрегатором
type Aggregator interface {
Start(ctx context.Context) error
Stop() error
SetInterval(d time.Duration)
Resize(workers int) error
IsRunning() bool
}

// IPCManager определяет интерфейс для межпроцессного взаимодействия
type IPCManager interface {
SaveState(state *AggregatorState) error
LoadState() (*AggregatorState, error)
SignalProcess(state *AggregatorState, signal int) error
IsProcessRunning(pid int) bool
}
