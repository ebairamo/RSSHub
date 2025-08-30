package models

import (
	"time"
)

// Обновите структуру Feed в models.go
type Feed struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Name      string    `db:"name"`
	URL       string    `db:"url"`
}

// Обновите структуру Article в models.go
type Article struct {
	ID          int       `db:"id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Title       string    `db:"title"`
	Link        string    `db:"link"`
	PublishedAt time.Time `db:"published_at"`
	Description string    `db:"description"`
	FeedID      int       `db:"feed_id"`
}

type RSS struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}
