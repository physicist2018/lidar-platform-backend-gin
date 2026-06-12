# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0] — 2026-06-12

### Added

- **Processing pipeline** — новый архитектурный слой для запуска алгоритмов обработки данных экспериментов.
- **Единый endpoint** `POST /experiments/{id}/process` — принимает `{"algorithm": "stage0", "params": {...}}`, создаёт `ProcessingRun` и запускает асинхронную обработку через Asynq.
- **Processor registry** — расширяемый реестр алгоритмов (`internal/domain/processing/`), каждый алгоритм реализует интерфейс `Processor`.
- **Stage0 (`stage0`)** — первый алгоритм обработки:
  - Фон: `avgtail` (mean хвоста), `medtail` (median хвоста), `file` (вычитание BGR файла).
  - **Crop**: обрезка профилей по высоте (`crop.crop_from`).
  - **Glue**: склейка analog/digital каналов для указанных длин волн, с масштабированием к `analog` или `digital`. Создаёт новый профиль с `DeviceID="BG"` (оригинальные каналы не изменяются).
- **Таблицы БД** — `processing_runs` (запуски алгоритмов), `processed_signals` (результаты обработки сигналов).
- **Новые endpoints:** `GET /processing/{id}` — статус обработки.
- **Методы загрузки профилей** в `LidarPackDataSource` — `GetProfilesByExperimentID`, `GetProfilesByFileID` для доступа к сигналам из БД.

## [1.9.0] — 2026-06-12

### Changed

- **Extracted `Float64Slice`** from `utils/meteo` into `utils/gorm/datatypes` — reusable `[]float64` ↔ `bytea` SQL type with `sql.Scanner`/`driver.Valuer`.
- **`LidarProfileEntity.Signal`** migrated from `[]byte` to `datatypes.Float64Slice` — signal is now a GORM-native `Float64Slice` instead of manual `[]byte` + `Float64sToBytes()`/`BytesToFloat64s()` helpers.
- **Removed** `utils/licel/signal.go` — `Float64sToBytes`/`BytesToFloat64s` replaced by `datatypes.Float64Slice` scanning/valuing.

## [1.8.0] — 2026-06-12

### Added

- **Meteo data support**:
  - **New table `meteo_records`** — stores all meteo levels for an experiment as binary `bytea` arrays in PostgreSQL.
  - **`Float64Slice` custom SQL type** — implements `sql.Scanner` / `driver.Valuer` for `[]float64` ↔ `bytea` (little-endian 8-byte per element).
  - **`meteo.ParseMeteoFile`** parser — reads `meteo.dat` files extracting PRES, HGHT, TEMP (required) and RELH, MIXR, DRCT, SKNT (optional).
  - **`meteo.StandardAtmosphere`** — generates ISA standard atmosphere (0–25 km, 100 m step) when meteo file is missing.
  - **`ExperimentEntity.MeteoID`** — nullable FK to `meteo_records.id`.
  - **Automatic fallback** in `preprocess()`: if `meteo.dat` is absent or unparseable, standard atmosphere is used gracefully.

## [1.7.0] — 2026-06-12

### Added

- **`LidarPackEntity.PackType` field** (`data` / `bgr`) to distinguish data archives from background (BGR) files.
- **`ExperimentEntity.LidarPackID`** — nullable FK to `lidar_packs.id` referencing the data pack.
- **`ExperimentEntity.BgrFileID`** — nullable FK to `lidar_files.id` referencing the background file.
- **`licel.FromLicelFile`** converter — builds a `LidarPack` with `PackType="bgr"` from a single `licelformat.LicelFile`.
- **Background file handling in `CreateExperimentUseCaseImpl.preprocess`**:
  - BGR file is parsed via `licelformat.LoadLicelFile` as a full licel file with profiles.
  - Saved as a separate `LidarPack` with `PackType="bgr"` via existing `SavePack`.
  - `LidarPackID` and `BgrFileID` are persisted on the experiment.
- **Backward compatibility** — `LicelZipPath` / `LicelBgrPath` string fields are preserved.

## [1.6.0] — 2026-06-11

### Added

- **Normalised storage of LicelPack data in PostgreSQL** (`lidar_packs`, `lidar_files`, `lidar_profiles` tables).
  - After parsing a Licel archive, the full hierarchy (pack → files → profiles) is saved into three linked GORM entities via a single transaction.
  - `LidarProfile.Signal` stored as `bytea` (LittleEndian `float64` array via `internal/utils/licel/signal.go`).
  - New domain entities: `internal/domain/entity/lidar_pack.go` (`LidarPack`, `LidarFile`, `LidarProfile`).
  - New DataSource: `internal/infrastructure/datasource/persistance/implementation/lidar_pack_datasource_impl.go` (`LidarPackDataSourceImpl.SavePack`).
  - New Repository: `internal/infrastructure/repository/lidar_pack_repository_impl.go`.
  - Converter: `internal/utils/licel/converter.go` (`licelformat.LicelPack` → `entity.LidarPack`).
  - Wiring in `internal/config/app.go` — `LidarPackRepositoryImpl` injected into `CreateExperimentUseCaseImpl`.
  - Auto-migration includes all three new entities (`cmd/migrate/main.go`).
- **No changes to MinIO upload or downstream handlers** — prepare/visualize/glue continue to work with the zip archive as before.

## [1.5.0] — 2026-06-11

### Changed

- **Full migration from Echo v5 to Chi v5** (`github.com/go-chi/chi/v5`).
  - Replaced `*echo.Echo` → `*chi.Mux`, all 9+ handler files migrated from `func(c *echo.Context) error` to standard `http.HandlerFunc`.
  - Body binding: `c.Bind()` → `json.NewDecoder(r.Body).Decode()`.
  - Path params: `c.Param()` → `chi.URLParam()`.
  - Query params: `c.QueryParam()` → `r.URL.Query().Get()`.
  - JSON responses: `c.JSON()` → new `response.JSON()` / `response.Error()` helpers.
  - Context values: `c.Set()/c.Get()` → `context.WithValue()` / `r.Context().Value()`.
  - Middleware: `echo.MiddlewareFunc` → `func(http.Handler) http.Handler`.
  - Server startup: `echo.Start()` → `http.ListenAndServe()`.
  - Route patterns: `:id` → `{id}`.
  - Removed dependencies: `github.com/labstack/echo/v5`, `github.com/labstack/echo-opentelemetry`.
  - Added `internal/delivery/http/controller/helpers.go` (`parseUint`, `parseInt`).
  - Added `internal/delivery/http/response/response.go` (JSON helpers).

### Fixed

- **Swagger UI default spec URL**: replaced stock Swagger UI `index.html` (pointing to `petstore.swagger.io`) with a custom version that loads our local `/swagger/swagger.json`.
  - Custom `index.html` loads Swagger UI from CDN (`unpkg.com/swagger-ui-dist@5`).
  - Removed `github.com/swaggo/files/v2` dependency.

- **Middleware ordering panic**: all middleware (including custom panic recovery) is now registered before routes, as required by Chi.

## [1.3.1] — 2026-06-05

### Fixed

- **`internal/infrastructure/queue/handlers.go`** — `handleVisualize` теперь принимает статусы `done stage 1` и `done stage 2` как валидные для визуализации. Ранее требовался только `done`, из-за чего после glue (`done stage 2`) визуализация падала с ошибкой "not ready".

### Changed

- **`docs/API.md`** — полное обновление документации: все три асинхронные задачи (prepare, glue, visualize), task polling (`GET /tasks/:taskID`), asynqmon, новая статусная машина.

## [1.3.0] — 2026-06-05

### Added

- **Asynq worker** — добавлено приложение `cmd/worker` на базе `github.com/hibiken/asynq` для асинхронной обработки долгих операций.
  - Все задачи (prepare, glue, visualize) вынесены из процесса API-сервера в отдельный воркер.
  - API `GET /prepared/:id` теперь возвращает `task_id` для polling, а результат доступен по `GET /tasks/:taskID`.
- **Polling-эндпоинт** — `GET /tasks/:taskID` (auth required) возвращает статус асинхронной задачи (`processing`, `done`, `failed`) и presigned URL результата.
- **Asynqmon** — в `docker-compose.yml` добавлен сервис `asynqmon` (порт 8090) для мониторинга очереди.
- **TaskStore** — Redis-based хранение результатов асинхронных задач для polling.

### Changed

- **Dockerfile** — multi-stage сборка: теперь билдятся оба бинарника (`server` и `worker`).
- **`internal/config/app.go`** — добавлена инициализация `queue.Client` и `queue.TaskStore`; use case'ы переключены на asynq.
- **`internal/domain/usecase/implementation/prepare`** — вместо `workerPool.Submit()` отправляет задачу в asynq.
- **`internal/domain/usecase/implementation/glue`** — вместо `workerPool.Submit()` отправляет задачу в asynq.
- **`internal/domain/usecase/implementation/visualize`** — полностью переписан: теперь только отправляет задачу в asynq (вместо синхронного выполнения).
- **Интерфейс `VisualizePreparedExperimentUseCase`** — возвращает `*AsyncTaskInfo{TaskID}` вместо `(string, error)`.

## [1.2.0] — 2026-06-05

### Added

- **Параметр `glued` в визуализацию** — `GET /prepared/:id?wavelen=...&photon=...&polarization=...&action=...&glued=0|1`.
  - `glued=0` (default) — профили выбираются как раньше (не-склеенные).
  - `glued=1` — выбираются только склеенные профили (DeviceID=BG) для заданной длины волны.
- **Параметр `polarization` в склейку** — `POST /experiments/:id/glue` теперь принимает опциональный `{"polarization": "..."}`.
  - Передаётся в `licelformat.LicelPack.Glue(wvl, h1, h2, polarization)` — склейка выполняется только для профилей указанной поляризации.
- **Поле `Glued` в `ExperimentChart`** — кеш визуализации различает glued/non-glued запросы.

### Changed

- **Маршрут визуализации** — `GET /prepared/{id}/{wavelen}/{photon}/{polarization}/{action}` → `GET /prepared/{id}`. Все параметры (кроме `id`) — query-параметры.
- **Параметры визуализации** — `photon` теперь опционален (default `0`), `polarization` default `o`, `type` default `png` вместо `svg`.
- Обновлена зависимость `github.com/physicist2018/licelfile/v2` до `v2.4.4`.

## [0.3.5] — 2026-06-02

### Added

- **Кеширование графиков визуализации** — эндпоинт `GET /prepared/:id/:wavelen/:photon/:polarization/:action` теперь сохраняет сгенерированные графики в MinIO и запоминает их в БД.
  - Новая модель `ExperimentChart` (chartType, formula, wavelen, polarization, isPhoton, pathToObject).
  - Таблица `experiment_charts` с уникальностью по `(experiment_id, chart_type, formula, wavelen, polarization, is_photon)`.
  - Query-параметр `?regenerate=true` — принудительная перерисовка. По умолчанию (`false`) — ищет кеш в БД, возвращает presigned URL без перегенерации.
  - MinIO: новые методы `UploadBytes` (загрузка `[]byte`) и `PresignedGetObject` (presigned URL на 1 час).

### Changed

- **`photon` в URI** — с `bool` на `int8`: `0` = analog, `1` = photon (поддержка `2` на будущее).
- **Ответ эндпоинта визуализации** — вместо raw-контента (SVG/PNG/JSON) теперь возвращается `{"url": "https://..."}` — presigned URL на объект в MinIO.
- **Сигнатура `VisualizePreparedExperimentUseCase.Execute`** — добавлены `isPhoton int8`, `regenerate bool`; возвращает `(string, error)`.
- **DI** — `ExperimentChartDataSource` → `ExperimentChartRepository` → `VisualizePreparedExperimentUseCaseImpl`.
- **AutoMigrate** — добавлена таблица `experiment_charts`.

## [0.3.4] — 2026-06-02

### Added

- **GET /experiments/{id}/channels** — список измерительных каналов эксперимента.
  - Ответ: `{ "channels": [{ "wavelen": 355.0, "polarization": "parallel", "isPhoton": 0, "isActive": 1 }, ...] }`.
  - Каналы извлекаются при препроцессинге из заголовков licel-профилей: `LicelProfile.Wavelength`, `.Polarization`, `.Photon`, `.Active`.
  - `isActive = 0`, если хотя бы один профиль канала имеет `Active=false`.
  - Хранятся в колонке `available_channels` таблицы `experiments` (тип: `jsonb`).

### Changed

- **Experiment** — новое поле `AvailableChannels []ExperimentChannel` на всех слоях (entity, DB, DTO).
- **datasource/ExperimentEntity** — добавлена колонка `AvailableChannels datatypes.JSON` (gorm `jsonb`).
- **create_experiment_use_case_impl** — `preprocess()` теперь сохраняет каналы при финальном `Update`.

## [0.3.3] — 2026-06-02

### Changed

- **Рефакторинг визуализации** — код отрисовки графиков (SVG, PNG, Plotly JSON) вынесен из
  `internal/domain/usecase/implementation/visualize_prepared_experiment_use_case_impl.go` в
  новый пакет `pkg/visualize/`.
  - Файл use-case сократился с ~1100 до ~293 строк и теперь содержит только бизнес-логику:
    загрузку данных, парсинг, подготовку данных и маршрутизацию в рендереры.
  - Пакет `pkg/visualize/` состоит из 8 файлов с чистыми standalone-функциями, не зависит от
    `internal/` и может переиспользоваться или тестироваться изолированно:
    - `result.go` — тип `Result` (ContentType + Body).
    - `utils.go` — `FormatTimeHHMM`, `MinInt`, `Percentile`, `ApplyFormula`, `HeatmapColor`.
    - `draw.go` — `DrawDashedLineH`, `DrawDashedLineV`, `LoadFont`.
    - `plotly.go` — Plotly-структуры + `HeatmapToPlotly` / `ProfileToPlotly`.
    - `heatmap_svg.go` — `HeatmapToSVG`.
    - `heatmap_png.go` — `HeatmapToPNG`.
    - `profile_svg.go` — `ProfileToSVG`.
    - `profile_png.go` — `ProfileToPNG`.
  - Контракты и публичное API не изменены.

## [0.3.2] — 2026-06-01

### Added

- **PNG-формат для визуализации** — новый `type=png` для `/image` и `/profile`. Генерирует PNG с помощью библиотеки `fogleman/gg` (чистый Go, без CGO). Heatmap PNG включает сетку, colorbar с тиками, подписи осей.
- **Сетка (grid lines) на heatmap SVG** — пунктирные горизонтальные и вертикальные линии для улучшения читаемости.

### Changed

- **Локальное время на оси X** — `formatTimeHHMM` теперь возвращает локальное время сервера вместо UTC.
- **Colorbar тики в SVG** — вместо двух крайних значений (min/max) теперь рисуется 5+ равномерных тиков с засечками и подписями слева от цветовой шкалы.
- **Heatmap SVG: персентильное масштабирование цветовой шкалы** — вместо абсолютного min/max для границ цвета теперь используются 5-й и 95-й персентили. Это обрезает выбросы и улучшает контрастность изображения.

## [0.3.1] — 2026-06-01

### Added

- **GET /prepared/:id/:wavelen/:photon/:polarization/:action** — визуализация подготовленных данных эксперимента.
  - `:action` = `image` → heatmap (X=время HH:MM, Y=дистанция м, цвет=интенсивность).
  - `:action` = `profile` → усреднённый XY-профиль (X=дистанция, Y=интенсивность).
  - Query-параметр `?type=` → `svg` (по умолчанию, `Content-Type: image/svg+xml`) или `json` (Plotly-совместимый JSON).
  - Query-параметр `?formula=` → `raw` (сырой сигнал P, по умолчанию), `rangecorr` (P × r²), `lograngecorr` (log₁₀(P × r²)).
  - Доступ: **admin, manager**.
  - Внутренняя логика: скачивание подготовленного zip из Minio, парсинг `LicelPack`, фильтрация профилей по `(isPhoton, wavelength, polarization)`, трансформация сигнала, генерация SVG/JSON.
  - SVG: встроенная генерация без внешних зависимостей. Heatmap с цветовой шкалой blue→cyan→green→yellow→red. Profile с полигональной линией и сеткой.
  - Plotly JSON: heatmap-трейс с `colorscale: Jet` или scatter-трейс `mode: lines` с hover-подсказками.

## [0.3.0] — 2026-06-01

### Added

- **UserID в Experiment и PreparedExperiment** — идентификация автора.
  - Поле `user_id` добавлено в `Experiment` и `PreparedExperiment` на всех слоях (entity, DB, DTO, mapper).
  - `POST /experiments` и `POST /experiments/{id}/prepare` теперь сохраняют `userID` из JWT claims.
- **AdminOrManager middleware** — роут `POST /experiments/{id}/prepare` теперь доступен админам и менеджерам.

### Changed

- **create_experiment_use_case_impl.go** — сигнатура `Execute()` теперь принимает `userID uint`.
- **prepare_experiment_use_case_impl.go** — сигнатура `Execute()` теперь принимает `userID uint`.

### Fixed

- **AutoMigrate с существующими строками** — колонка `user_id` в `experiments` и `prepared_experiments` теперь имеет `DEFAULT 1`, чтобы существующие строки не ломали `NOT NULL` constraint при миграции.

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
