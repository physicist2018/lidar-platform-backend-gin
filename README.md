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
│   ├── controller/      # Gin-контроллеры
│   ├── middleware/       # Auth, AdminOnly
│   └── route/           # Регистрация роутов
├── domain/
│   ├── entity/          # Бизнес-модели
│   ├── repository/      # Интерфейсы репозиториев
│   └── usecase/         # Use-case интерфейсы + реализация
├── infrastructure/
│   ├── datasource/      # GORM entities, persistance, cache
│   ├── db/              # PostgreSQL, Redis подключения
│   └── repository/      # Реализация cache-aside репозиториев
└── utils/
    ├── auth/            # JWT (generate, parse)
    ├── hash/            # bcrypt
    ├── mapper/          # Entity ↔ Domain ↔ DTO
    ├── pagination/      # Дженерик-пагинация
    └── response/        # AppError

pkg/dto/                 # Публичные DTO (запросы/ответы)
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
