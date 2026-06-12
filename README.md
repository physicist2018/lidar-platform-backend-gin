# Lidar Platform Backend

REST API для платформы LiDAR — управление пользователями, обработка облаков точек, телеметрия устройств.

## Стек

| Слой | Технология |
|---|---|
| HTTP-роутер | Chi v5 |
| ORM | GORM + PostgreSQL 15 |
| Кеш | Redis 7 (cache-aside) |
| Очередь задач | Asynq (Redis-backed) |
| Трейсинг | OpenTelemetry (Redis) |
| Логи | Logrus (structured JSON, replaces slog) |
| Аутентификация | JWT (HS256, bcrypt-пароли) |
| Документация | Swagger (swaggo), API.md |
| Конфигурация | Viper (`.env`) |
| Контейнеризация | Docker + docker-compose |

## Архитектура

```
Delivery (Chi v5) → Domain (pure Go) ← Infrastructure (GORM, Redis)
                       ↑
                   pkg/dto
```

Чистая архитектура (Clean Architecture) с 12‑шаговым процессом добавления новых доменных сущностей. Подробности — в [`.agent/workflows/CLEAN_ARCH_SKILL.md`](.agent/workflows/CLEAN_ARCH_SKILL.md).

## Быстрый старт

```bash
# 1. Инфраструктура (PostgreSQL, Redis, MinIO)
docker-compose up -d postgres redis

# 2. Миграция БД
go run ./cmd/migrate

# 3. Сидирование (создаёт admin‑пользователя)
go run ./cmd/seeder

# 4. Запуск сервера
go run ./cmd/app

# 5. (Опционально) Запуск воркера для асинхронных задач
go run ./cmd/worker

Сервер поднимается на `http://localhost:8080`, воркер запускает asynq-обработчики.

### Docker Compose (полный запуск)

```bash
docker-compose up -d
```

Поднимает все сервисы: сервер, воркер, asynqmon (мониторинг очереди на порту 8090), PostgreSQL, Redis, MinIO.

## Admin по умолчанию

| Поле | Значение |
|---|---|
| Email | `admin@lidar-platform.io` |
| Password | `admin123` |
| Role | `admin` |

> 🔒 Смените пароль в production!

## API Endpoints

### Auth (публичный)

| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/auth/login` | Логин → JWT токен |

### Users (требуется аутентификация)

| Метод | Путь | Роль | Описание |
|---|---|---|---|
| `GET` | `/users` | Любая | Список с пагинацией / фильтрацией |
| `GET` | `/users/:id` | Любая | Получить одного |
| `POST` | `/users` | **admin** | Создать |
| `PUT` | `/users/:id` | **admin** | Обновить |
| `DELETE` | `/users/:id` | **admin** | Удалить (soft-delete) |

### Experiments (требуется аутентификация)

| Метод | Путь | Роль | Описание |
|---|---|---|---|
| `GET` | `/experiments` | Любая | Список с пагинацией / фильтрацией (`status`, `title`) |
| `GET` | `/experiments/:id` | Любая | Получить один (со статусом, путями к файлам и ID пакета/файла фона) |
| `GET` | `/experiments/:id/channels` | Любая | Список каналов эксперимента (`wavelen`, `polarization`, `isPhoton`, `isActive`) |
| `POST` | `/experiments` | **admin** | Создать (multipart: `title`, `licelZip`, `licelBgr`, `meteoFile`) |
| `POST` | `/experiments/:id/prepare` | **admin, manager** | Подготовка данных (JSON: `crop_alt`, `bgr_type`, `bgr_alt`). Асинхронно — статус по `GET /experiments/:id` |
| `POST` | `/experiments/:id/glue` | **admin, manager** | Склейка каналов (JSON: `wavelengths`, `polarization`, `h1`, `h2`). Асинхронно (`202 Accepted`) |
| `POST` | `/experiments/:id/process` | **admin, manager** | Запуск алгоритма обработки (JSON: `algorithm`, `params`). Асинхронно — статус по `GET /processing/{id}` |

### Prepared Experiments (требуется аутентификация)

| Метод | Путь | Роль | Описание |
|---|---|---|---|
| `GET` | `/prepared/:id` | **admin, manager** | Визуализация (async): возвращает `202` с `task_id` для polling. Query params: `?wavelen=...&polarization=...&action=...&glued=0|1&type=png|svg|json&formula=...&regenerate=true` |

### Tasks (требуется аутентификация)

| Метод | Путь | Роль | Описание |
|---|---|---|---|
| `GET` | `/tasks/:taskID` | Любая | Polling: возвращает статус задачи (`pending`, `processing`, `done`, `failed`) и presigned URL при готовности |
| `GET` | `/processing/:id` | **admin, manager** | Статус запуска алгоритма обработки

> **GET /prepared/:id** — все параметры query:
> - `wavelen` (float64, required) — длина волны, например `532`
> - `photon` (int) — `0` (аналоговый, default) или `1` (фотонный); игнорируется при `glued=1`
> - `polarization` (string) — поляризация (default: `o`)
> - `action` (string, required) — `image` (heatmap: X=время, Y=дистанция) или `profile` (усреднённый XY-график)
> - `glued` (int) — `0` (не-склеенные, default) или `1` (склеенные профили DeviceID=BG)
> - `type` (string) — `png` (default), `svg`, `json`
> - `formula` (string) — `raw` (default), `rangecorr`, `lograngecorr`
> - `regenerate` (bool) — принудительная перерисовка в обход кеша
> Ответ: `{"task_id": "...", "status": "accepted"}` — опросить готовность через `GET /tasks/:taskID`.
>
> **GET /tasks/:taskID** — ответы:
> - `{"task_id": "...", "status": "pending"}` — в очереди
> - `{"task_id": "...", "status": "processing"}` — выполняется
> - `{"task_id": "...", "status": "done", "url": "..."}` — готов, presigned URL действителен 1 час
> - `{"task_id": "...", "status": "failed", "error": "..."}` — ошибка

> **POST /experiments** — возвращает `201` сразу со статусом `staged`. Препроцессинг (парсинг licel zip, загрузка в Minio) выполняется асинхронно в worker pool. Статус: `staged → uploading → done|failed`.

> **POST /experiments/:id/prepare** — асинхронный пайплайн (asynq): вычитание фона (`file`/`avgTail`/`medTail`) → обрезка по высоте → загрузка в Minio (`experiments/{id}/prepared/licel-prepared.zip`). Статус prepared: `staged → removebgr → cropping → done stage 1 → (glue) → done stage 2 → done|failed`.

> **POST /experiments/:id/glue** — асинхронный пайплайн (asynq): склейка каналов для указанных длин волн → перезапись zip → статус `done stage 2`. Ответ: `202 Accepted`.

> **POST /experiments/{id}/process** — единый endpoint для запуска алгоритмов обработки. Параметры:
> - `algorithm` (string, required) — имя алгоритма: `"stage0"`.
> - `params` (object, required) — параметры алгоритма. Для `stage0`:
>   - `crop.crop_from` — высота обрезки профилей (в метрах), данные выше удаляются. `0` — без обрезки.
>   - `background.type` — `"file"`, `"avgtail"`, `"medtail"`.
>   - `background.bgr_from` — высота начала хвоста для tail-based (в метрах).
>   - `glue` — массив объектов `{"wavelength", "polarization", "r0", "r1", "scale_to"}`. Создаёт новый склеенный профиль с `DeviceID="BG"`.
> Порядок: фон → crop → glue. Статус `ProcessingRun`: `staged → processing → done|failed`. Статус по `GET /processing/{id}`.

> **GET /prepared/:id** — асинхронная визуализация (asynq): возвращает `202 Accepted` с `task_id`. Результат доступен через `GET /tasks/:taskID` (polling).

### Swagger UI

```
http://localhost:8080/swagger/index.html
```

## Переменные окружения

```env
SERVER_ADDRESS=0.0.0.0:8080
DB_SOURCE=postgres://user:pass@localhost:5432/lidar_platform?sslmode=disable
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
CACHE_TTL_DEFAULT=15m
JWT_SECRET=change-me-in-production-use-a-256-bit-random-key
JWT_EXPIRATION=24h
MAX_WORKERS=4
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=lidar-experiments
MINIO_USE_SSL=false
```

## Структура проекта

```
cmd/
├── app/main.go            # HTTP-сервер
├── worker/main.go         # Asynq worker (асинхронные задачи)
├── migrate/main.go        # AutoMigrate (GORM)
└── seeder/main.go         # Admin user seed

internal/
├── config/                # Viper config + DI composition root
├── delivery/http/
│   ├── controller/        # HTTP-контроллеры (user, experiment)
│   ├── middleware/         # Auth, AdminOnly
│   └── route/             # Регистрация роутов
├── domain/
│   ├── entity/            # Бизнес-модели (User, Experiment)
│   ├── repository/        # Интерфейсы репозиториев
│   └── usecase/           # Use-case интерфейсы + реализация
├── infrastructure/
│   ├── datasource/        # GORM entities, persistance, cache
│   ├── db/                # PostgreSQL, Redis подключения
│   ├── queue/             # Asynq: tasks, client, handlers, task_store
│   ├── repository/        # Реализация cache-aside репозиториев
│   └── storage/           # Minio/S3 клиент
├── utils/
│   ├── auth/              # JWT (generate, parse)
│   ├── hash/              # bcrypt
│   ├── licel/             # Converter LicelPack → domain entities
│   ├── gorm/datatypes/    # Custom GORM types (Float64Slice ↔ bytea)
│   ├── mapper/            # Entity ↔ Domain ↔ DTO
│   ├── pagination/        # Дженерик-пагинация
│   ├── response/          # AppError
│   └── worker/            # Worker pool (legacy, только CreateExperiment)

pkg/
├── dto/                   # Публичные DTO (запросы/ответы)
└── visualize/             # Рендеринг графиков (SVG, PNG, Plotly JSON)

docs/                      # Swagger (авто-генерируется), API.md
test/                      # Unit, integration, k6
```

## Команды

```bash
# Сборка всех бинарников (сервер + воркер)
go build ./...
go build -o ./server ./cmd/app
go build -o ./worker ./cmd/worker

# Линтинг
go vet ./...

# Генерация Swagger
./updateDocs.sh

# Запуск миграций
go run ./cmd/migrate

# Сидирование
go run ./cmd/seeder

# Запуск воркера отдельно
go run ./cmd/worker
```

## Лицензия

MIT


## Планы на будущее
