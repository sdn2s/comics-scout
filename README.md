# Comics Scout

Поисковик по комиксам [XKCD](https://xkcd.com).
Пользователь вводит фразу, она нормализуется (стемминг + стоп-слова), затем по индексу слов
ищутся подходящие комиксы. Админка позволяет запустить индексацию (`update`) и очистить
базу (`drop`).

## Архитектура

| Сервис   | Назначение                                                          | Порт  |
|----------|---------------------------------------------------------------------|-------|
| `web`    | HTML-фронтенд (Go-шаблоны, JWT в HttpOnly-cookie)                   | 28084 |
| `api`    | REST-шлюз, аутентификация, rate-limiter, проксирует в gRPC          | 28080 |
| `words`  | Стемминг и нормализация слов (gRPC)                                 | 28081 |
| `update` | Тянет XKCD, кладёт описания в Postgres, рассылает события в NATS    | 28082 |
| `search` | Поиск по индексу слов; подписан на NATS для обновления индекса      | 28083 |

Инфраструктура: Postgres (хранение комиксов и слов), NATS (события индексации),
pgAdmin (доступен на `:18888`).

## Технологии

**Бэкенд**
- Go 1.25, стандартный `net/http` + `http.ServeMux` с методами роутинга (Go 1.22+)
- gRPC + protobuf для общения между сервисами
- NATS как брокер событий обновления индекса
- PostgreSQL + `golang-migrate` для миграций
- `golang-jwt/jwt/v5` для JWT-сессий
- `cleanenv` для конфигов (yaml + env-override)
- `log/slog` для структурированного логирования
- `kljensen/snowball` для стемминга

**Тесты и инструменты**
- `testify` для модульных и интеграционных тестов
- `go-sqlmock` для моков БД
- `protoc` + `protolint`, `golangci-lint`

**Инфраструктура**
- Docker Compose, отдельный Dockerfile на каждый сервис
- Multi-stage сборка (`golang:1.25` → `alpine:3.20`)

## Запуск

Требуется Docker.

```bash
make up           # собрать и поднять весь кластер
make down         # остановить
make clean        # остановить и удалить тома
make test         # поднять кластер и прогнать интеграционные тесты
make unit         # юнит-тесты
```

После `make up` открыть:

- UI поиска: <http://localhost:28084>
- Админка: <http://localhost:28084/admin> (логин `admin` / `password`)
- pgAdmin: <http://localhost:18888> (`admin@test.com` / `password`)

База стартует пустой — зайдите в админку и нажмите **Запустить update**, чтобы `update`-сервис
стянул XKCD и построил индекс.

## Структура

```
comics-scout/
├── compose.yaml               # оркестрация всего кластера
├── Makefile                   # обёртки над docker compose и тестами
├── search-services/
│   ├── api/                   # REST-шлюз + middleware (auth, rate-limit)
│   ├── search/                # gRPC-сервис поиска по индексу
│   ├── update/                # парсер XKCD + индексатор
│   ├── words/                 # нормализация слов (стемминг)
│   ├── web/                   # HTML-фронтенд (templates/, api_client.go)
│   ├── proto/                 # .proto файлы и сгенерированный код
│   ├── closers/               # утилита для graceful close
│   └── Dockerfile.{api,search,update,words,web}
└── tests/                     # интеграционные тесты, запускаются в контейнере
```
