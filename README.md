# RSSHub
RSSHub
Цели обучения
Работа с форматами XML и RSS
Параллелизм и каналы
Пул работников
PostgreSQL
Создание докера
Абстрактный
В этом проекте вы создадите приложение с интерфейсом командной строки — RSSагрегатор каналов, который:

Предоставляет интерфейс командной строки (CLI)
Извлекает и анализирует RSS-каналы
Хранит статьи в PostgreSQL
Агрегирует RSS-каналы с помощью пула рабочих процессов в фоновом режиме
Это сервис, который собирает публикации из различных источников, предоставляющих RSS-каналы (новостные сайты, блоги, форумы). Он помогает пользователям оставаться в курсе событий, не посещая каждый сайт по отдельности.

Такой инструмент будет полезен журналистам, исследователям, аналитикам и всем, кто хочет быть в курсе интересующих их тем без лишнего шума. Такое приложение делает информацию более доступной и централизованной.

Контекст
Вы разрабатываете приложение с интерфейсом командной строки, которое периодически извлекает статьи из RSS-каналов, добавленных пользователями, и сохраняет их в PostgreSQL. Вы также реализуете механизм, отвечающий за фоновую параллельную обработку RSS-каналов. Все сервисы развертываются с помощью Docker Compose.

Общие критерии
Ваш код ДОЛЖЕН быть отформатирован в соответствии с gofumpt. В противном случае вы автоматически получите оценку 0.

Ваша программа ДОЛЖНА успешно компилироваться без ошибок.

Ваша программа НЕ ДОЛЖНА завершать работу неожиданно (например, из-за разыменования нулевого указателя, выхода индекса за пределы диапазона и т. д.). Если это произойдёт, вы получите оценку 0 во время защиты.

Внешние пакеты разрешены только для работы с PostgreSQL. Использование любых других внешних зависимостей приведёт к снижению оценки до 0.

Если во время запуска возникает ошибка (например, неверные аргументы командной строки), программа ДОЛЖНА:

Выход с ненулевым кодом состояния
Отобразите чёткое и понятное сообщение об ошибке
Проект ДОЛЖЕН успешно компилироваться с помощью следующей команды, запущенной из корневого каталога проекта:

$ go build -o rsshub .
Проект НЕ ДОЛЖЕН выдавать никаких ошибок при запуске с флагом -race:
$ go run -race main.go
Интервал ДОЛЖЕН устанавливаться динамически с помощью команды в терминале. В противном случае вы автоматически получите оценку 0.

Количество рабочих процессов ДОЛЖНО также настраиваться динамически с помощью команды в терминале. В противном случае вы автоматически получите оценку 0.

Обязательная Часть
Инфраструктура
Добавьте файл docker-compose.yml, в котором запущены следующие службы:

RSSHub — приложение с интерфейсом командной строки
PostgreSQL – для хранения статей
RSS-канал
Основная цель программы rsshub — получить RSS фид веб-сайта и сохранить его содержимое в структурированном формате в нашей базе данных. Это позволяет нам удобно отображать данные в интерфейсе командной строки.

RSS Расшифровывается как "Really Simple Syndication" — это способ получать свежий контент с веб-сайта в структурированном формате. Он широко используется в интернете: большинство сайтов, ориентированных на контент, предоставляют RSS фид.

Структура RSS-канала
RSS Это особая XML структура. Мы упростим задачу и сосредоточимся только на нескольких полях. Ниже приведён пример документов, которые необходимо проанализировать:

<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
<channel>
 <title>Пример RSS-канала</title>
 <link>https://www.example.com</link>
 <description>Это пример RSS-канала</description>
 <item>
 <title>Первая статья</title>
 <link>https://www.example.com/article1</link>
 <description>Это содержание первой статьи.</description>
 <pubDate>Пн, 6 сентября 2021 г., 12:00:00 по Гринвичу</pubDate>
 </item>
 <item>
 <title>Вторая статья</title>
 <link>https://www.example.com/article2</link>
 <description>Это содержание второй статьи.</description>
 <pubDate>Вт, 7 сентября 2021 г., 14:30:00 по Гринвичу</pubDate>
 </item>
</channel>
</rss>
Затем преобразуйте этот тип документа в такие структуры:

структура RSS-канал тип {
 Канал структура {
 Заголовок       строка    `xml:"title"`
		Ссылка        строка    `xml:"link"`
		Описание строка    `xml:"description"`
		Элемент []RSS-элемент `xml:"item"`
 } `xml:"канал"`
}

тип RSS-элемент структура {
 Заголовок       строка `xml:"title"`
	Ссылка        строка `xml:"link"`
	Описание строка `xml:"description"`
	Дата публикации     строка `xml:"pubDate"`
}
If there are additional fields in the XML, the parser will simply ignore them. If some fields are missing, they will retain their default (zero) values.

Accordingly, you will need to implement an RSS parser that performs an HTTP request using the feed URL stored in the database.

Periodic Feed Aggregation
Feeds are essentially lists of publications. Each publication is a separate web page. The main goal of the rsshub program is to fetch actual publications using the URLs from RSS feeds and store them in the database. This allows us to display them nicely in the CLI.

You need to create a mechanism that regularly retrieves feed URLs from the database, compares them with new articles from the RSS feeds, and saves any changes to the database. Priority should be given to feeds that have not been updated for a long time or have never been updated.

This mechanism must run in the background at a specified interval. The default interval should be 3 minutes. This interval should also be configurable using a CLI command. The command must be able to change the interval while the application is running, without stopping the background aggregation process.

To improve performance, implement a worker pool for parallel processing of multiple articles:

On each timer tick, retrieve the N most outdated or never-updated feeds from the database.

Distribute them across the worker pool (goroutines).

Each worker:

Downloads the feed by its URL
Parses new articles
Saves them to Postgres
The application should be able to change the ticker interval and size of workers without restarting the application.

Default behavior:

Default settings must be defined in the application configuration
Default interval: 3 minutes
Default number of workers: 3
Mechanism requirements:

The ticker must be controllable: it should be possible to stop and restart it with a new interval
The worker pool must be scalable: workers can be stopped or added dynamically
What to Avoid

Problem	How to Avoid
❌ Data Race	Use sync.Mutex or atomic when accessing shared variables (e.g., n, interval)
❌ Goroutine Leaks	All spawned goroutines (ticker, workers) must be terminated using context.Context or a done channel
❌ Duplicate Tickers	When calling SetInterval(), always stop the old ticker before starting a new one
❌ Closing Channel Twice	Channels (e.g., jobs) must be closed by only one goroutine
❌ Panic on ticker.Reset()	Never call Reset() on a stopped ticker
❌ Unconsumed jobs Channel	Workers must read from the jobs channel; otherwise, writing to the channel will cause a deadlock
What Needs to Be Implemented

Structures for the ticker and worker pool
Workers — read from the jobs channel and process articles
The jobs channel must be created and closed correctly
Using context cancellation to implement graceful shutdown
The code should be structured as a service (in internal/)
The interface is below:
type YourAggregator interface {
    Start(ctx context.Context) error           // Starts background feed polling
    Stop() error                               // Graceful shutdown

    SetInterval(d time.Duration)               // Dynamically changes fetch interval
    Resize(workers int) error                  // Dynamically resizes worker pool
    ...
}
WARNING
Do NOT DoS the servers you're fetching feeds from. Anytime you write code that makes a request to a third party server you should be sure that you are not making too many requests too quickly. That's why I recommend printing to the console for each request, and being ready with a quick Ctrl+C to stop the program if you see something going wrong.

DoS attack - read more.

WARNING
Implement the following commands
Start background fetching
Starts the background process that periodically fetches and processes RSS feeds using a worker pool.

rsshub fetch
This command launches:

The RSS fetcher loop (ticker-based)
The worker pool for concurrent feed parsing and storage
After executing this command, the application must log a confirmation message like:

The background process for fetching feeds has started (interval = 3 minutes, workers = 3)
Important: Only one instance of the background process can be running at a time. If you try to start it again while it’s already active, the application must log:

Фоновый процесс уже запущен
Это сделано для того, чтобы несколько сборщиков данных не работали одновременно и не дублировали работу.

Добавить новый RSS-канал
Команда добавляет новый RSS-канал в базу данных PostgreSQL.

rsshub add --name "tech-crunch" --url "https://techcrunch.com/feed/"
Установите Интервал выборки по RSS
Динамически изменяет частоту фоновой загрузки RSS-каналов.

rsshub установить интервал 2 минуты
Пример: получать данные каждые 2 минуты.

После выполнения этой команды приложение должно вывести подтверждающее сообщение:

Интервал получения данных изменился с 3 минут на 2 минуты
Эта команда работает только в том случае, если команда rsshub fetch запущена в другом терминале
Эта команда обновляет интервал тикера без перезапуска приложения.
Установленное количество работников
Динамически изменяет размер фонового пула рабочих процессов, который одновременно обрабатывает RSS-каналы.

rsshub set-workers 5
Пример: запустите 5 параллельных процессов.

После выполнения этой команды приложение должно вывести подтверждающее сообщение:

Количество работников увеличилось с 3 до 5
Эта команда работает только в том случае, если команда rsshub fetch запущена в другом терминале
Это изменение вступает в силу немедленно, без перезапуска приложения или прерывания текущего цикла выборки.
Список доступных RSS-каналов
Команда отображает RSS-каналы, хранящиеся в базе данных PostgreSQL.

rsshub list --num 5
Показывает 5 последних добавленных каналов. Без --num отображаются все каналы.

Выходной формат:

# Доступные RSS-каналы

1. Название: tech-crunch
 URL: https://techcrunch.com/feed/
 Добавлено: 2025-06-10 15:34

2. Название: hacker-news
 URL: https://news.ycombinator.com/rss
 Добавлено: 2025-06-10 15:37

3. Название: bbc-world
 URL: http://feeds.bbci.co.uk/news/world/rss.xml
 Добавлено: 11 июня 2025 г. в 09:15

4. Название: the-verge
 URL: https://www.theverge.com/rss/index.xml
 Добавлено: 12 июня 2025 г. в 13:50

5. Название: ars-technica
 URL: http://feeds.arstechnica.com/arstechnica/index
 Добавлено: 13 июня 2025 г. в 08:25
Удалить RSS-канал
Команда удаляет фид из базы данных PostgreSQL по имени.

rsshub удалить --имя "tech-crunch"
Пример: удалите ленту TechCrunch из хранилища.

Показывать последние статьи
Команда показывает последние статьи из PostgreSQL по названию канала.

rsshub articles --feed-name "tech-crunch" --num 5
Показывает 5 последних статей из указанной ленты. По умолчанию отображается 3 статьи, если --num не указано.

Выходной формат:

Источник: tech-crunch

1. [2025-06-18] Apple анонсирует новые чипы M4 для MacBook Pro
 https://techcrunch.com/apple-announces-m4/

2. [2025-06-17] OpenAI запускает GPT-5 с мультимодальными возможностями
 https://techcrunch.com/openai-launches-gpt-5/

3. [16.06.2025] Google представляет новые инструменты для обеспечения конфиденциальности на конференции I/O 2025
 https://techcrunch.com/google-privacy-io-2025/

4. [15.06.2025] TikTok представляет платформу для разработчиков для интеграции
 https://techcrunch.com/tiktok-developer-platform/

5. [14.06.2025] В Microsoft Teams появилась функция обобщения информации о совещаниях с помощью ИИ
 https://techcrunch.com/microsoft-teams-ai-summary/
Показать справку CLI
Команда выводит инструкции по использованию и описания всех доступных команд.

rsshub - справка
Пример Использования:

$ ./rsshub --help

 Использование:
 rsshub КОМАНДА [ПАРАМЕТРЫ]

 Основные команды:
 add — добавление нового RSS-канала
 set-interval установить интервал получения RSS-каналов
 set-workers установить количество рабочих процессов
 list — список доступных RSS-каналов
 delete — удаление RSS-канала
 articles — отображение последних статей
 fetch — запуск фонового процесса, который периодически получает и обрабатывает RSS-каналы с помощью пула рабочих процессов
Как корректно остановить агрегатор
Чтобы остановить работающий фоновый процесс:

Нажмите Ctrl+C в терминале, где выполняется команда rsshub fetch

или

Отправьте сигнал завершения (например, SIGINT, SIGTERM) из другого процесса

При завершении работы приложение:

Отмените фоновый контекст
Плавно остановите таймер и всех работников
Запишите сообщение с подтверждением:
Изящное завершение работы: агрегатор остановлен
Пример рабочего процесса
В одном терминале:

# Запустите агрегатор
$ ./rsshub fetch
$ Запущен фоновый процесс для получения каналов (интервал = 3 минуты, количество рабочих процессов = 3)


# Чтобы остановить: нажмите Ctrl+C
$ Изящное завершение работы: агрегатор остановлен
В другом терминале: измените настройки

$ ./rsshub set-interval --duration 2m
$ Интервал получения данных изменился с 3 минут на 2 минуты

$ ./rsshub set-workers --count 4
$ Количество рабочих процессов изменилось с 3 до 5
База данных
PostgreSQL
🗂 Таблица каналов (feeds) Содержит метаданные о каждом канале RSS, добавленном в систему.

Поле	Тип	Описание
ID	UUID (PK)	Уникальный идентификатор для канала
созданный_ат	ВРЕМЕННАЯ МЕТКА	Когда был добавлен корм
обновленный_ат	ВРЕМЕННАЯ МЕТКА	Когда лента обновлялась в последний раз
Имя	ТЕКСТ (уникальный)	Человекочитаемое название канала
url -адрес	ТЕКСТ	URL-адрес RSS-канала
📝 Эта таблица используется для отслеживания всех RSS-источников, которые отслеживает агрегатор. URL-адрес используется для получения данных, а название — для ссылки на каналы с помощью команд интерфейса командной строки.

📰 Таблица статей (articles) Содержит все статьи, извлечённые из различных RSS-каналов.

Поле	Тип	Описание
ID	UUID (PK)	Уникальный идентификатор для статьи
созданный_ат	ВРЕМЕННАЯ МЕТКА	Когда статья хранилась
обновленный_ат	ВРЕМЕННАЯ МЕТКА	Когда статья была изменена в последний раз
Название	ТЕКСТ	Название статьи
Ссылка	ТЕКСТ	Канонический URL-адрес статьи
опубликованный файл_at	ВРЕМЕННАЯ МЕТКА	Отметка времени первоначальной публикации
Описание	ТЕКСТ	Краткое описание или аннотация из RSS-канала
feed_id ( идентификатор канала )	UUID	Внешний ключ, ссылающийся на feeds.id
📝 В этой таблице собраны все полученные статьи и ссылки на соответствующие им RSS-каналы. Она обеспечивает дедупликацию и поддерживает запросы к последним публикациям в каждом канале.

Это стандартная инструкция для таблиц. Вам лучше добавить поля.

Миграции
Миграции — это набор файлов с версиями, которые описывают изменения в схеме базы данных (DDL): создание таблиц, изменение столбцов, добавление индексов и т. д.

Пример:

migrations/
├── create_feeds_table.up.sql
└── create_feeds_table.down.sql
*.up.sql — Используемый SQL
*.down.sql — SQL-запрос, который был отменён
create_feeds_table.up.sql:

каналы ТАБЛИЦА СОЗДАТЬ (
 //TODO
);
create_feeds_table.down.sql:

ТАБЛИЦА УДАЛИТЬ ЕСЛИ СУЩЕСТВУЕТ feeds;
Начальная настройка
Конфигурация
# Приложение CLI
CLI_APP_TIMER_INTERVAL=3m
CLI_APP_WORKERS_COUNT=3

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=changem
POSTGRES_DBNAME=rsshub
Создание докера
Запуски:

RSSHub (приложение CLI)
PostgreSQL (порт: 5432)
укажите имя пользователя, пароль и базу данных env
Рекомендации от Автора
Начните с одного RSS-канала и команды CLI для его получения
Реализуйте механизм агрегации с помощью фонового таймера
Добавьте пул рабочих процессов для параллельной обработки данных
Используйте PostgreSQL для хранения данных и настройте миграцию баз данных
Создайте файл docker-compose.yml для локальной разработки
В завершение протестируйте логику работы приложения
Вот несколько RSS-каналов для начала:

TechCrunch: https://techcrunch.com/feed/
Новости хакеров: https://news.ycombinator.com/rss
Новости ООН: https://news.un.org/feed/subscribe/ru/news/all/rss.xml
Новости Би - би - си: https://feeds.bbci.co.uk/news/world/rss.xml
Ars Technica ( Арс Техника ): http://feeds.arstechnica.com/arstechnica/index
На Грани: https://www.theverge.com/rss/index.xml
Необязательно (приятно иметь)
Для вашего собственного развития (Будущие возможности):

Реализуйте веб-API для внешнего доступа
Добавьте Telegram-бота, чтобы уведомлять пользователей о новых статьях
Поддержка
Всегда сложно понять, с чего начать. Попробуйте разбить задачу на более мелкие части, каждая из которых решает одну конкретную проблему. Затем объедините и расширьте их — в итоге у вас будет готовое решение.

Удачи вам и приятного времяпрепровождения :3

Автор
Этот проект был создан:

@trech

Контакты:

Электронная почта
GitHub
Discord
LinkedIn