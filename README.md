# RSSHub

CLI-приложение для агрегации RSS-лент с автоматическим фоновым обновлением и хранением в PostgreSQL. Проект демонстрирует работу с конкурентностью, Worker Pool паттерном и управлением жизненным циклом горутин.

## 🚀 Возможности

- **Агрегация RSS**: автоматический сбор статей из множества RSS-источников
- **Worker Pool**: параллельная обработка лент с настраиваемым количеством воркеров
- **Динамическая конфигурация**: изменение интервала и количества воркеров без перезапуска
- **PostgreSQL хранилище**: структурированное хранение лент и статей
- **CLI интерфейс**: удобное управление через команды терминала
- **Docker Compose**: простое развертывание всей инфраструктуры
- **Graceful Shutdown**: корректное завершение всех горутин

## 🛠️ Технологии

- **Язык**: Go (только стандартная библиотека + pgx/v5)
- **База данных**: PostgreSQL
- **Контейнеризация**: Docker Compose
- **Паттерны**: Worker Pool, Ticker, Context cancellation
- **XML Parsing**: encoding/xml

## 📦 Установка

```bash
# Клонировать репозиторий
git clone https://github.com/ebairamo/rsshub.git
cd rsshub

# Собрать проект
go build -o rsshub .
```

### Docker Compose

```bash
# Запустить PostgreSQL
docker-compose up -d
```

## 🎯 Использование

### Настройка окружения

Создайте файл `.env`:

```env
# CLI App
CLI_APP_TIMER_INTERVAL=3m
CLI_APP_WORKERS_COUNT=3

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=changeme
POSTGRES_DBNAME=rsshub
```

### Основные команды

#### Добавить RSS ленту

```bash
./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
```

#### Запустить фоновую агрегацию

```bash
# В первом терминале
./rsshub fetch
# Вывод: The background process for fetching feeds has started (interval = 3 minutes, workers = 3)
```

#### Изменить интервал обновления

```bash
# Во втором терминале (пока fetch работает)
./rsshub set-interval 2m
# Вывод: Interval of fetching feeds changed from 3 minutes to 2 minutes
```

#### Изменить количество воркеров

```bash
./rsshub set-workers 5
# Вывод: Number of workers changed from 3 to 5
```

#### Показать список лент

```bash
# Показать все ленты
./rsshub list

# Показать последние 5 лент
./rsshub list --num 5
```

**Пример вывода:**
```
# Available RSS Feeds

1. Name: tech-crunch
   URL: https://techcrunch.com/feed/
   Added: 2025-06-10 15:34

2. Name: hacker-news
   URL: https://news.ycombinator.com/rss
   Added: 2025-06-10 15:37

3. Name: bbc-world
   URL: http://feeds.bbci.co.uk/news/world/rss.xml
   Added: 2025-06-11 09:15
```

#### Показать статьи из ленты

```bash
# Показать 5 последних статей
./rsshub articles --feed-name "tech-crunch" --num 5

# По умолчанию показывается 3 статьи
./rsshub articles --feed-name "hacker-news"
```

**Пример вывода:**
```
Feed: tech-crunch

1. [2025-06-18] Apple announces new M4 chips for MacBook Pro
   https://techcrunch.com/apple-announces-m4/

2. [2025-06-17] OpenAI launches GPT-5 with multimodal capabilities
   https://techcrunch.com/openai-launches-gpt-5/

3. [2025-06-16] Google unveils new privacy tools at I/O 2025
   https://techcrunch.com/google-privacy-io-2025/
```

#### Удалить ленту

```bash
./rsshub delete --name "tech-crunch"
```

#### Справка

```bash
./rsshub --help
```

## 🏗️ Архитектура

### Worker Pool Pattern

```
                    ┌─────────────┐
                    │   Ticker    │
                    │  (3 min)    │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Fetcher   │
                    │  Get N feeds│
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │    Jobs Channel         │
              │  (Feed URLs queue)      │
              └────────────┬────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
    ┌──────────┐     ┌──────────┐    ┌──────────┐
    │ Worker 1 │     │ Worker 2 │    │ Worker N │
    └──────────┘     └──────────┘    └──────────┘
          │                │                │
          └────────────────┼────────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ PostgreSQL  │
                    └─────────────┘
```

### Компоненты

**Ticker**
- Периодический таймер (по умолчанию 3 минуты)
- Динамически изменяемый интервал
- Graceful shutdown через context

**Worker Pool**
- N горутин для параллельной обработки
- Масштабируемый размер (можно изменять на лету)
- Чтение из общего канала jobs

**Fetcher**
- Получает N самых неактуальных лент из БД
- Парсит RSS XML
- Отправляет задачи в jobs channel

## 💾 База данных

### Таблица feeds

Хранит метаданные о RSS лентах.

```sql
CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    name TEXT UNIQUE,
    url TEXT
);
```

### Таблица articles

Хранит все загруженные статьи.

```sql
CREATE TABLE articles (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    title TEXT,
    link TEXT,
    published_at TIMESTAMP,
    description TEXT,
    feed_id UUID REFERENCES feeds(id)
);
```

## 📝 Структура RSS

Приложение парсит RSS feeds следующей структуры:

```xml
<rss version="2.0">
  <channel>
    <title>RSS Feed Example</title>
    <link>https://www.example.com</link>
    <description>This is an example RSS feed</description>
    <item>
      <title>First Article</title>
      <link>https://www.example.com/article1</link>
      <description>Article content...</description>
      <pubDate>Mon, 06 Sep 2021 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>
```

## 🔄 Workflow

### Типичный сценарий использования

**Терминал 1 - Агрегатор:**
```bash
$ ./rsshub fetch
The background process for fetching feeds has started (interval = 3 minutes, workers = 3)

# Для остановки: Ctrl+C
Graceful shutdown: aggregator stopped
```

**Терминал 2 - Управление:**
```bash
# Добавить ленты
$ ./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
$ ./rsshub add --name "hacker-news" --url "https://news.ycombinator.com/rss"

# Изменить настройки
$ ./rsshub set-interval 2m
Interval of fetching feeds changed from 3 minutes to 2 minutes

$ ./rsshub set-workers 5
Number of workers changed from 3 to 5

# Просмотр результатов
$ ./rsshub list
$ ./rsshub articles --feed-name "tech-crunch" --num 5
```

## 🔧 Технические детали

### Конкурентность

**Проблемы и решения:**

| Проблема | Решение |
|----------|---------|
| Data Race | `sync.Mutex` или `atomic` для общих переменных |
| Goroutine Leaks | Использование `context.Context` для отмены |
| Дубликаты Ticker | Остановка старого ticker перед созданием нового |
| Закрытие канала дважды | Канал закрывается только одной горутиной |
| Deadlock | Воркеры всегда читают из канала jobs |

### Graceful Shutdown

Приложение корректно завершает работу:
1. При нажатии Ctrl+C
2. При получении сигнала SIGINT/SIGTERM
3. Отменяет context для всех горутин
4. Ожидает завершения всех воркеров
5. Закрывает соединение с БД

## 📚 Рекомендуемые RSS ленты

```bash
# Технологии
./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
./rsshub add --name "hacker-news" --url "https://news.ycombinator.com/rss"
./rsshub add --name "the-verge" --url "https://www.theverge.com/rss/index.xml"
./rsshub add --name "ars-technica" --url "http://feeds.arstechnica.com/arstechnica/index"

# Новости
./rsshub add --name "bbc-world" --url "https://feeds.bbci.co.uk/news/world/rss.xml"
./rsshub add --name "un-news" --url "https://news.un.org/feed/subscribe/ru/news/all/rss.xml"
```

## ⚠️ Важные предупреждения

**Не создавайте DoS атаку!**
- Не делайте слишком много запросов слишком быстро
- Используйте разумный интервал (минимум 1 минута)
- Выводите логи каждого запроса
- Будьте готовы быстро остановить программу (Ctrl+C)

## 🎓 Цели обучения

Этот проект демонстрирует:
- Работу с XML и RSS форматами
- Конкурентность и каналы в Go
- Worker Pool паттерн
- Управление жизненным циклом горутин
- Работу с PostgreSQL
- Docker Compose для развертывания
- Context cancellation для graceful shutdown
- Race condition prevention

## 🙏 Автор задания

**@trech**
- [Email](mailto:trech@example.com)
- [GitHub](https://github.com/trech)
- [LinkedIn](https://www.linkedin.com/in/trech/)

---

*Проект выполнен в рамках обучения в ALEM School*
*Групповой проект с: @akairamb*

# RSSHub

CLI application for RSS feed aggregation with automatic background updates and PostgreSQL storage. This project demonstrates concurrency, Worker Pool pattern, and goroutine lifecycle management.

## 🚀 Features

- **RSS Aggregation**: automatic article collection from multiple RSS sources
- **Worker Pool**: parallel feed processing with configurable worker count
- **Dynamic Configuration**: change interval and worker count without restart
- **PostgreSQL Storage**: structured storage of feeds and articles
- **CLI Interface**: convenient terminal-based management
- **Docker Compose**: simple infrastructure deployment
- **Graceful Shutdown**: proper termination of all goroutines

## 🛠️ Tech Stack

- **Language**: Go (standard library only + pgx/v5)
- **Database**: PostgreSQL
- **Containerization**: Docker Compose
- **Patterns**: Worker Pool, Ticker, Context cancellation
- **XML Parsing**: encoding/xml

## 📦 Installation

```bash
# Clone repository
git clone https://github.com/ebairamo/rsshub.git
cd rsshub

# Build project
go build -o rsshub .
```

### Docker Compose

```bash
# Start PostgreSQL
docker-compose up -d
```

## 🎯 Usage

### Environment Setup

Create `.env` file:

```env
# CLI App
CLI_APP_TIMER_INTERVAL=3m
CLI_APP_WORKERS_COUNT=3

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=changeme
POSTGRES_DBNAME=rsshub
```

### Main Commands

#### Add RSS Feed

```bash
./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
```

#### Start Background Aggregation

```bash
# In first terminal
./rsshub fetch
# Output: The background process for fetching feeds has started (interval = 3 minutes, workers = 3)
```

#### Change Update Interval

```bash
# In second terminal (while fetch is running)
./rsshub set-interval 2m
# Output: Interval of fetching feeds changed from 3 minutes to 2 minutes
```

#### Change Worker Count

```bash
./rsshub set-workers 5
# Output: Number of workers changed from 3 to 5
```

#### Show Feed List

```bash
# Show all feeds
./rsshub list

# Show last 5 feeds
./rsshub list --num 5
```

**Example output:**
```
# Available RSS Feeds

1. Name: tech-crunch
   URL: https://techcrunch.com/feed/
   Added: 2025-06-10 15:34

2. Name: hacker-news
   URL: https://news.ycombinator.com/rss
   Added: 2025-06-10 15:37

3. Name: bbc-world
   URL: http://feeds.bbci.co.uk/news/world/rss.xml
   Added: 2025-06-11 09:15
```

#### Show Feed Articles

```bash
# Show last 5 articles
./rsshub articles --feed-name "tech-crunch" --num 5

# Default shows 3 articles
./rsshub articles --feed-name "hacker-news"
```

**Example output:**
```
Feed: tech-crunch

1. [2025-06-18] Apple announces new M4 chips for MacBook Pro
   https://techcrunch.com/apple-announces-m4/

2. [2025-06-17] OpenAI launches GPT-5 with multimodal capabilities
   https://techcrunch.com/openai-launches-gpt-5/

3. [2025-06-16] Google unveils new privacy tools at I/O 2025
   https://techcrunch.com/google-privacy-io-2025/
```

#### Delete Feed

```bash
./rsshub delete --name "tech-crunch"
```

#### Help

```bash
./rsshub --help
```

## 🏗️ Architecture

### Worker Pool Pattern

```
                    ┌─────────────┐
                    │   Ticker    │
                    │  (3 min)    │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Fetcher   │
                    │  Get N feeds│
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │    Jobs Channel         │
              │  (Feed URLs queue)      │
              └────────────┬────────────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
    ┌──────────┐     ┌──────────┐    ┌──────────┐
    │ Worker 1 │     │ Worker 2 │    │ Worker N │
    └──────────┘     └──────────┘    └──────────┘
          │                │                │
          └────────────────┼────────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ PostgreSQL  │
                    └─────────────┘
```

### Components

**Ticker**
- Periodic timer (default 3 minutes)
- Dynamically changeable interval
- Graceful shutdown via context

**Worker Pool**
- N goroutines for parallel processing
- Scalable size (changeable on-the-fly)
- Reading from shared jobs channel

**Fetcher**
- Gets N most outdated feeds from DB
- Parses RSS XML
- Sends tasks to jobs channel

## 💾 Database

### feeds Table

Stores RSS feed metadata.

```sql
CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    name TEXT UNIQUE,
    url TEXT
);
```

### articles Table

Stores all fetched articles.

```sql
CREATE TABLE articles (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    title TEXT,
    link TEXT,
    published_at TIMESTAMP,
    description TEXT,
    feed_id UUID REFERENCES feeds(id)
);
```

## 📝 RSS Structure

The application parses RSS feeds with the following structure:

```xml
<rss version="2.0">
  <channel>
    <title>RSS Feed Example</title>
    <link>https://www.example.com</link>
    <description>This is an example RSS feed</description>
    <item>
      <title>First Article</title>
      <link>https://www.example.com/article1</link>
      <description>Article content...</description>
      <pubDate>Mon, 06 Sep 2021 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>
```

## 🔄 Workflow

### Typical Usage Scenario

**Terminal 1 - Aggregator:**
```bash
$ ./rsshub fetch
The background process for fetching feeds has started (interval = 3 minutes, workers = 3)

# To stop: Ctrl+C
Graceful shutdown: aggregator stopped
```

**Terminal 2 - Management:**
```bash
# Add feeds
$ ./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
$ ./rsshub add --name "hacker-news" --url "https://news.ycombinator.com/rss"

# Change settings
$ ./rsshub set-interval 2m
Interval of fetching feeds changed from 3 minutes to 2 minutes

$ ./rsshub set-workers 5
Number of workers changed from 3 to 5

# View results
$ ./rsshub list
$ ./rsshub articles --feed-name "tech-crunch" --num 5
```

## 🔧 Technical Details

### Concurrency

**Problems and Solutions:**

| Problem | Solution |
|---------|----------|
| Data Race | `sync.Mutex` or `atomic` for shared variables |
| Goroutine Leaks | Use `context.Context` for cancellation |
| Duplicate Tickers | Stop old ticker before creating new one |
| Closing Channel Twice | Channel closed by only one goroutine |
| Deadlock | Workers always read from jobs channel |

### Graceful Shutdown

Application properly terminates:
1. On Ctrl+C press
2. On SIGINT/SIGTERM signal
3. Cancels context for all goroutines
4. Waits for all workers to finish
5. Closes DB connection

## 📚 Recommended RSS Feeds

```bash
# Technology
./rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
./rsshub add --name "hacker-news" --url "https://news.ycombinator.com/rss"
./rsshub add --name "the-verge" --url "https://www.theverge.com/rss/index.xml"
./rsshub add --name "ars-technica" --url "http://feeds.arstechnica.com/arstechnica/index"

# News
./rsshub add --name "bbc-world" --url "https://feeds.bbci.co.uk/news/world/rss.xml"
./rsshub add --name "un-news" --url "https://news.un.org/feed/subscribe/en/news/all/rss.xml"
```

## ⚠️ Important Warnings

**Don't DoS the servers!**
- Don't make too many requests too quickly
- Use reasonable interval (minimum 1 minute)
- Log every request
- Be ready to quickly stop the program (Ctrl+C)

## 🎓 Learning Objectives

This project demonstrates:
- Working with XML and RSS formats
- Concurrency and channels in Go
- Worker Pool pattern
- Goroutine lifecycle management
- Working with PostgreSQL
- Docker Compose deployment
- Context cancellation for graceful shutdown
- Race condition prevention

## 🙏 Project Author

**@trech**
- [Email](mailto:trech@example.com)
- [GitHub](https://github.com/trech)
- [LinkedIn](https://www.linkedin.com/in/trech/)

---

*Project completed as part of ALEM School curriculum*
*Group project with: @akairamb*
