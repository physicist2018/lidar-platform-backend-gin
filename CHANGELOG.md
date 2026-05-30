# Changelog

All notable changes to this project will be documented in this file.

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
