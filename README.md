# Lidar Platform Backend

REST API для платформы LiDAR — управление пользователями, обработка облаков точек, телеметрия устройств.

## Стек

| Слой | Технология |
|---|---|
| HTTP-роутер | Gin |
| ORM | GORM + PostgreSQL 15 |
| Кеш | Redis 7 (cache-aside) |
| Трейсинг | OpenTelemetry (Gin + Redis) |
| Логи | Logrus (structured JSON) |
| Аутентификация | JWT (HS256, bcrypt-пароли) |
| Документация | Swagger (swaggo) |
| Конфигурация | Viper (`.env`) |
| Контейнеризация | Docker + docker-compose |

## Архитектура

```
Delivery (Gin) → Domain (pure Go) ← Infrastructure (GORM, Redis)
                       ↑
                   pkg/dto
```

Чистая архитектура (Clean Architecture) с 12‑шаговым процессом добавления новых доменных сущностей. Подробности — в [`.agent/workflows/CLEAN_ARCH_SKILL.md`](.agent/workflows/CLEAN_ARCH_SKILL.md).

## Быстрый старт

```bash
# 1. Инфраструктура
docker-compose up -d postgres redis

# 2. Миграция БД
go run ./cmd/migrate

# 3. Сидирование (создаёт admin‑пользователя)
go run ./cmd/seeder

# 4. Запуск сервера
go run ./cmd/app
```

Сервер поднимается на `http://localhost:8080`.

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
| `GET` | `/experiments/:id` | Любая | Получить один (со статусом и путями к файлам) |
| `GET` | `/experiments/:id/channels` | Любая | Список каналов эксперимента (`wavelen`, `polarization`, `isPhoton`, `isActive`) |
| `POST` | `/experiments` | **admin** | Создать (multipart: `title`, `licelZip`, `licelBgr`, `meteoFile`) |
| `POST` | `/experiments/:id/prepare` | **admin, manager** | Подготовка данных (JSON: `crop_alt`, `bgr_type`, `bgr_alt`) |

### Prepared Experiments (требуется аутентификация)

| Метод | Путь | Роль | Описание |
|---|---|---|---|
| `GET` | `/prepared/:id/:wavelen/:photon/:polarization/:action` | **admin, manager** | Визуализация: возвращает `{"url"}` — presigned URL на график в MinIO (`?type=svg|json|png&formula=raw|rangecorr|lograngecorr&regenerate=true`) |

> **GET /prepared/:id/:wavelen/:photon/:polarization/:action** — `:action` = `image` (heatmap: X=время, Y=дистанция) или `profile` (усреднённый XY-график). `:wavelen` — длина волны (например `532`), `:photon` — `0` (аналоговый) или `1` (фотонный). `type=svg` (по умолчанию) — SVG-изображение, `type=png` — PNG, `type=json` — Plotly JSON. `formula=raw` — сырой сигнал P, `rangecorr` — P×r², `lograngecorr` — log₁₀(P×r²). `?regenerate=true` — принудительная перерисовка в обход кеша. Ответ — `{"url": "https://minio/..."}`, presigned URL действителен 1 час.

> **POST /experiments** — возвращает `201` сразу со статусом `staged`. Препроцессинг (парсинг licel zip, загрузка в Minio) выполняется асинхронно в worker pool. Статус обновляется: `staged → uploading → done|failed`.

> **POST /experiments/:id/prepare** — асинхронный пайплайн: вычитание фона (`file`/`avgTail`/`medTail`) → обрезка по высоте → загрузка в Minio (`experiments/{id}/processed/dats.zip`). Статус: `staged → removebgr → cropping → done|failed`.

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
├── app/main.go          # HTTP-сервер
├── migrate/main.go      # AutoMigrate (GORM)
└── seeder/main.go       # Admin user seed

internal/
├── config/              # Viper config + DI composition root
├── delivery/http/
│   ├── controller/      # Gin-контроллеры (user, experiment)
│   ├── middleware/       # Auth, AdminOnly
│   └── route/           # Регистрация роутов
├── domain/
│   ├── entity/          # Бизнес-модели (User, Experiment)
│   ├── repository/      # Интерфейсы репозиториев
│   └── usecase/         # Use-case интерфейсы + реализация
├── infrastructure/
│   ├── datasource/      # GORM entities, persistance, cache
│   ├── db/              # PostgreSQL, Redis подключения
│   ├── repository/      # Реализация cache-aside репозиториев
│   └── storage/         # Minio/S3 клиент
└── utils/
    ├── auth/            # JWT (generate, parse)
    ├── hash/            # bcrypt
    ├── mapper/          # Entity ↔ Domain ↔ DTO
    ├── pagination/      # Дженерик-пагинация
    ├── response/        # AppError
    └── worker/          # Worker pool (MAX_WORKERS)

pkg/
├── dto/                 # Публичные DTO (запросы/ответы)
└── visualize/           # Рендеринг графиков (SVG, PNG, Plotly JSON)
docs/                    # Swagger (авто-генерируется)
test/                    # Unit, integration, k6
```

## Команды

```bash
# Сборка
go build ./...

# Линтинг
go vet ./...

# Генерация Swagger
./updateDocs.sh

# Запуск миграций
go run ./cmd/migrate

# Сидирование
go run ./cmd/seeder
```

## Лицензия

MIT


## Планы на будущее
[ ] Gaceful shutdown
[ ] Обработка данных
