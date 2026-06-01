# Changelog

All notable changes to this project will be documented in this file.

## [0.2.3] — 2026-06-01

### Added

- **POST /experiments/{id}/prepare** — подготовка данных эксперимента: вычитание фона и обрезка по высоте.
  - Новый домен `PreparedExperiment` (ID, ExperimentID, CropAlt, BGRType, BGRAlt, PathToData, Status).
  - Статусная машина: `staged → removebgr → cropping → done | failed`.
  - Три стратегии вычитания фона: `file` (поэлементное вычитание из BGR-файла), `avgTail` (среднее), `medTail` (медиана).
  - Обрезка по `cropAlt` через `LicelPack.SetMaxDist`. Результат сохраняется в Minio: `experiments/{id}/processed/dats.zip`.
  - Воркерный пул: скачивает данные из Minio, обрабатывает, выгружает обратно.
- **Minio.DownloadFile** — загрузка объектов из Minio на диск.
- **PreparedExperiment DB entity** — GORM-сущность `prepared_experiments` с внешним ключом на `experiments`.
- **PreparedExperiment domain** — полный Clean Architecture стек (entity, datasource, repository, usecase, controller, route, DTO, mapper).

## [0.2.2] — 2026-05-31

### Changed

- **licelfile v2.1.2 → v2.1.4** — обновлена библиотека парсинга licel-файлов.
  `pack.StartTime` и `pack.StopTime` теперь возвращают min/max таймстемпов по всей пачке автоматически,
  что позволило убрать ручную итерацию по `pack.Data` при извлечении `MeasurementStartTime` и
  `MeasurementStopTime`.

## [0.2.1] — 2026-05-31

### Fixed

- **307 Temporary Redirect при POST-запросах** — маршруты для базовых путей (`/users`, `/experiments`) были зарегистрированы
  с конечным слешем `/` (например, `GET "/"`), из-за чего Gin с `RedirectTrailingSlash` делал 307-редирект
  с `/users` на `/users/`. Для POST-запросов это приводило к потере тела запроса.
  Исправлено: все базовые маршруты переписаны с `"/"` на `""` (пустая строка) — теперь запросы обрабатываются
  напрямую.
- **CLEAN_ARCH_SKILL.md** — шаблонный `SetupOrderRoutes` обновлён аналогичным образом, чтобы новые доменные
  фичи не наследовали этот баг.

## [0.2.0] — 2026-05-31

### Added

- **Experiment entity** — хранение лидарных измерений с асинхронным препроцессингом:
  - `entity.Experiment` с полями `ID`, `Title`, `Comments`, `MeasurementStartTime`, `MeasurementStopTime`,
    `LicelZipPath`, `LicelBgrPath`, `MeteoFilePath`, `Status`, `ErrorMsg`.
  - Статусная машина: `staged → uploading → done | failed` с валидацией переходов (`ValidateTransition`).
  - `POST /experiments` (multipart: `title`, `licelZip`, `licelBgr`, `meteoFile`) — создаёт эксперимент со статусом `staged`,
    немедленно возвращает `201`, асинхронно обрабатывает через worker pool.
  - `GET /experiments` — пагинированный список с фильтрацией по `status` и `title`.
  - `GET /experiments/:id` — получение одного эксперимента.
- **Worker pool** (`internal/utils/worker/pool.go`):
  - Канальный пул горутин, размер задаётся `MAX_WORKERS` (default=4).
  - `Start()`, `Submit(task)`, graceful `Shutdown()`.
- **Minio storage** (`internal/infrastructure/storage/minio.go`):
  - Клиент для S3-совместимого хранилища (Minio).
  - `UploadFile()`, авто-создание bucket при старте.
  - Конфигурация: `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`.
- **licelfile интеграция** (`github.com/physicist2018/licelfile/v2` v2.1.2):
  - Парсинг ZIP-архива licel-файлов в горутине для извлечения `MeasurementStartTime` (минимальное) и `MeasurementStopTime` (максимальное) по всем файлам в пачке.

### Preprocessing flow (goroutine)

1. `status → uploading`
2. `licelformat.NewLicelPackFromZip(tempDir/licel.zip)` → ищет min(MeasurementStartTime) и max(MeasurementStopTime)
3. `minio.Upload(experiments/{id}/source/licel.zip)`
4. `minio.Upload(experiments/{id}/source/bgr.dat)`
5. `minio.Upload(experiments/{id}/source/meteo.dat)`
6. `status → done` (запись таймстемпов и путей в Minio)
7. `os.RemoveAll(tempDir)`

При любой ошибке → `status → failed` + `error_msg`.

## [0.1.0] — 2026-05-30

### Added

- **Project scaffold**: Clean Architecture directory layout (`cmd`, `internal`, `pkg`, `test`, `docs`).
- **Config**: Viper-based `.env` loader with `SERVER_ADDRESS`, `DB_SOURCE`, `REDIS_*`, `CACHE_TTL_DEFAULT`, `JWT_SECRET`, `JWT_EXPIRATION`.
- **Infrastructure**:
  - `db/postgres.go` — GORM + PostgreSQL connection with ping health-check.
  - `db/redis.go` — Redis 7 client with OTel instrumentation and ping health-check.
  - `docker-compose.yml` — PostgreSQL 15 Alpine, Redis 7 Alpine, app service.
  - `Dockerfile` — Multi-stage build (golang:1.23-alpine → alpine:3.19).
- **Utils**:
  - `pagination` — Generic `Pagination[T]` struct with offset calculation.
  - `response` — `AppError` wrapper for internal error propagation.
  - `hash` — bcrypt password hashing (`hash.Password`, `hash.CheckPassword`).
  - `auth` — JWT HS256 token generation/validation (`GenerateToken`, `ParseToken`).
- **Domain — User entity**:
  - `entity.User` with `ID`, `Name`, `Email`, `Role` (admin/guest/manager), `Password`.
  - `entity.UserFilter` for list filtering.
  - `HidePassword()` for safe JSON serialization.
  - Domain errors: `ErrEmailAlreadyExists`, `ErrInvalidCredentials`, `ErrAdminRequired`.
- **User CRUD** (full 12-step Clean Architecture flow):
  - Repository interface (`FindAll`, `FindByID`, `FindByEmail`, `Create`, `Update`, `Delete`).
  - 5 use-case interfaces + implementations (GetAll, GetByID, Create, Update, Delete).
  - GORM entity `UserEntity` with soft-delete.
  - DataSource with dynamic WHERE, ILIKE, offset pagination.
  - Redis cache-aside repository implementation.
  - Mappers: Entity ↔ Domain, Domain → DTO (non-nil slices).
  - Request/Response DTOs with `binding` validation.
- **Auth**:
  - `LoginUseCase` — email/password validation, JWT generation.
  - `AuthMiddleware` — Bearer token extraction and validation.
  - `AdminOnly` middleware — role-based access control for POST/PUT/DELETE `/users`.
  - Self-deletion guard in `DELETE /users/:id`.
- **Routing**:
  - `POST /auth/login` — public.
  - `GET /users`, `GET /users/:id` — authenticated.
  - `POST /users`, `PUT /users/:id`, `DELETE /users/:id` — admin-only.
- **Migration**: `cmd/migrate` — `AutoMigrate` for `UserEntity` (creates `users` table).
- **Seeder**: `cmd/seeder` — idempotent admin user creation (`admin@lidar-platform.io` / `admin123`).
- **Swagger**:
  - Full annotations on all endpoints (`@Summary`, `@Param`, `@Success`, `@Failure`, `@Security`).
  - Error response model `dto.ErrorResponse`.
  - Swagger UI at `/swagger/index.html`.
  - `updateDocs.sh` helper script.
- **Observability**:
  - OpenTelemetry spans in every use-case (Gin + Redis).
  - Structured JSON logging with `logrus` (operation, duration, error).
