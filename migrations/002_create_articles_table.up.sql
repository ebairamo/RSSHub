
CREATE TABLE IF NOT EXISTS articles (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL,
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    published_at TIMESTAMP NOT NULL,
    description TEXT,
    feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
);


ALTER TABLE articles ADD CONSTRAINT articles_link_key UNIQUE (link);