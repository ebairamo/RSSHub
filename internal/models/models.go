package models

import (
	"time"
)

// Feed представляет RSS-канал
type Feed struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Name      string    `db:"name"`
	URL       string    `db:"url"`
}

// Article представляет статью из RSS-канала
type Article struct {
	ID          string    `db:"id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Title       string    `db:"title"`
	Link        string    `db:"link"`
	PublishedAt time.Time `db:"published_at"`
	Description string    `db:"description"`
	FeedID      string    `db:"feed_id"`
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
