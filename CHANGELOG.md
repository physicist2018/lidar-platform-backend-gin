# Changelog

All notable changes to this project will be documented in this file.

## [2.4.0] — 2026-06-13

### Added

- **Новый endpoint** `POST /results/{stage}/data` — получение stage0-данных (distance, 2D data, time).
- **Поле `FileID` в `ProcessedSignal`** — для связывания с файлом и получения времени измерения.
- **Новый use case** `GetStage0DataUseCase` — фильтрация по wavelength, polarization, device_id, временному диапазону.
- **Time-фильтр в `GetByProcessingRunIDFiltered`** — JOIN с `lidar_files` для временной селекции профилей.
- **`FileID` теперь копируется в glued-профили (DeviceID="BG")** из template-профиля при склейке.

### Fixed

- **`ExperimentDataSource.Update`** — title больше не перетирается пустой строкой при частичных обновлениях (например, staged → uploading в preprocess).

## [2.3.0] — 2026-06-12

### Removed

- **Отключены ручки**: `POST /experiments/{id}/prepare`, `POST /experiments/{id}/glue`, `GET /prepared/{id}`, `GET /tasks/{taskID}`.
- Из контроллера удалены поля `PrepareExperimentUC`, `VisualizePreparedExperimentUC`, `GluePreparedExperimentUC`, `TaskStore`.
- Из `config/app.go` убрана инициализация use case'ов prepared experiment и task store.
- Из README удалены секции Prepared Experiments, Tasks, упоминания prepare/glue.

### Fixed

- `docker-compose.yml`: asynqmon теперь использует `--redis-addr=` вместо переменной окружения `ASYNQMON_REDIS_ADDR`, которая игнорировалась образом.

## [2.2.0] — 2026-06-12

### Added

- **Миграция**: `cmd/migrate/main.go` — GORM AutoMigrate для всех сущностей.
- **README**: секция `## Команды`.

### Fixed

- **`docker-compose.yml`**: исправлен `command` для `asynqmon` (убрана строковая конкатенация `&&`, разделены команды в `entrypoint`).

## [2.1.0] — 2026-06-12

### Added

- **Кеш (cache-aside)**: добавлен пакет `utils/cache` — дженерик-обёртка над Redis с блокирующим TTL, сериализацией JSON и фолбэком на источник. Реализован для `ExperimentRepository.FindByID`.
- **`CHANGELOG.md`** — начальное ведение.

## [2.0.0] — 2026-06-12

### Added

- **`internal/domain/processing`** — универсальная архитектура для алгоритмов обработки (Processor + Registry).
- **Stage0 (`stage0`)** — первый алгоритм обработки: фон (avgtail/medtail/file), crop, glue.
- **Таблицы БД** — `processing_runs`, `processed_signals`.
- **Async Pipeline** — Asynq-задачи: `TypeProcess("task:process")` + handler для запуска алгоритмов.
- **API**: `POST /experiments/{id}/process` — запуск алгоритма (JSON: `algorithm`, `params`). `GET /processing/{id}` — статус.
- **DTO** — `ProcessExperimentBody`, `ProcessingRunResponse`.
- **Обратная связь** — клиент отправляет `POST /experiments/{id}/process`, получает `201 Created` с `id` processing run. Статус — `GET /processing/{id}`.
- **`cmd/worker`** — Asynq worker (mux: prepare, glue, visualize, process).

## [1.9.0] — 2026-06-12

### Changed

- **Удалён `golang/slog`** → полный переход на Logrus во всех слоях с `controller.Helper` → `logrus.Logger`.
- **Логи** теперь пишутся через Logrus внутри asynq-хендлеров (handleProcess, prepare, glue, visualize).
- **Slog-логгер** заменён на Logrus через `logrus.WithFields(logrus.Fields{})`.
- **Замена** `go.uber.org/zap` на Logrus (убрана зависимость).
- **`internal/infrastructure/queue`** — полный переход на Logrus (asynq handler, task_store).
- **Удалён** `internal/infrastructure/task` (logrus → logrus).
- **Чистка** go.mod от неиспользуемых зависимостей (zap, slog, golang.org/x/exp и др.)

## [1.8.0] — 2026-06-12

### Added

- **Ручка `POST /experiments/{id}/process`**: стартует алгоритм `stage0` (долгая операция — фон, crop, glue). Создаёт `ProcessingRun`.
- **`GET /processing/{id}`**: возвращает статус `ProcessingRun`.
- **Удалена ручка `POST /experiments/{id}/glue`** (заменена на `POST /experiments/{id}/process`).
- **Удалена ручка `GET /glue/{id}`** (заменена на `GET /processing/{id}`).
- **`GlueParam`**, **`GlueResult`** — перенесены в старые entity (пока закомментированы).
- **Удалён use case `GlueExperimentUseCase`**, `GluePreparedExperimentUseCase`.
- **Удалён старый контроллер `GlueController`**.
- **`ProcessingRun`** — новая сущность в `entity/`, `repository/`, `datasource/`, `persistance/`.
- **`ProcessedSignal`** — новая сущность для хранения обработанных сигналов.
- **Удалены неиспользуемые файлы**: `glue_controller.go`, `glue_experiment_use_case_impl.go`, `glue_prepared_experiment_use_case_impl.go`, `glue_prepared_experiment_use_case.go`.

## [1.7.0] — 2026-06-12

### Added

- **`visualize_prepared_experiment_use_case.go`** — заглушка use case для визуализации подготовленного эксперимента.
- **`renderer.go`** — интерфейс рендерера графиков (`Renderer` + `PlotlyRenderer`, `SvgRenderer`, `PngRenderer`).
- **`ExperimentChart`** — новая сущность для хранения сгенерированных графиков.
- **`ChartType`** — enum-тип для `range_corrected`, `range_squared_corrected`, `background`, `glue`.

## [1.6.0] — 2026-06-11

### Added

- **`glue_channels`** — новый endpoint `POST /experiments/{id}/glue`.
- **batch вставка** для `processed_signals` через `BatchCreate`.
- **Loglevel** через `viper` (`LOG_LEVEL`).
- **swagger**: авто-документация через `swaggo/swag`, `docs/` на `http://localhost:8080/swagger/index.html`.
- **Update** для `ProcessingRun` с помощью `UpdateStatus`.
- **`GET /experiments/{id}/channels`** — получение каналов эксперимента (wavelength, polarization, photon/analog).
- **`golang/slog`** — временный логгер (замена на Logrus в 1.9.0).

## [1.5.0] — 2026-06-11

### Changed

- **`GET /experiments`**: ответ содержит `measurement_start_time`, `measurement_stop_time`.
- **`GET /experiments/{id}`**: теперь возвращает не только `status`, но и `measurement_start_time`, `measurement_stop_time`.
- **MinIO**: в `experiments` сохраняется `licel_zip_path`, `licel_bgr_path`, `meteo_file_path`.
- **Эксперимент** после preprocess содержит `status: "done"`, заполненные `measurement_start_time`, `measurement_stop_time`, `available_channels`.
- **Подготовка данных** (preprocess) — загрузка всех файлов в Minio до установки статуса `"done"`.
- **Удалён `test_upload.dat`** из dockerdata.
- **`.gitignore`**: добавлен `dockerdata/*`.

### Fixed

- **Preprocess goroutine**: URL-ы загруженных файлов корректно обновляются.
- **Ошибка `no rows in result set`** при первом создании из-за попытки достать `measurement_start_time` у новой записи.
- **Cascade delete**: при удалении записи из `experiments` удаляются зависимые `lidar_packs`, `lidar_files`, `lidar_profiles`.

## [1.3.1] — 2026-06-05

### Fixed

- **Лишние обновления** в data source не должны перезаписывать поля нулевыми значениями.

### Changed

- **Logger** заменён на `golang/slog` (временное решение, полный переход на Logrus в 1.9.0).

## [1.3.0] — 2026-06-05

### Added

- **POST /experiments/{id}/process** — асинхронный run любого алгоритма.
- **Asynq-клиент**: создание задач с уникальным ID.
- **Algolia-подобная архивация**: очистка ключей при создании эксперимента.

### Changed

- **Controller** переписан на `CreateExperimentUseCase`.
- **Preprocessing** перемещён в `asynq` task.

## [1.2.0] — 2026-06-05

### Added

- **Обработка**: `POST /experiments/{id}/glue`.
- **Очередь задач**: asynq + handler.
- **Логлайн**: `ProcessingRun`.
- **API**: `POST /experiments/{id}/prepare` — подготовка данных.
- **Asynq-воркер**: `cmd/worker/main.go`.

### Changed

- **Две доменные сущности** разделены: `Experiment` (метаданные) и `LidarPack` (измерения).

## [0.3.5] — 2026-06-02

### Added

- **Поддержка `BgrFileID`**: при создании эксперимента BGR-файл загружается и сохраняется как `LidarPack` с `PackType="bgr"`, его ID записывается в `experiments.bgr_file_id`.

### Changed

- **Preprocess** теперь сохраняет BGR-пак в БД.
- **`SetFailed`** — при неудаче выставляет `ErrorMsg`.

## [0.3.4] — 2026-06-02

### Added

- **ID сырого профиля** в `ExperimentalProfileModel.ID`.
- **`FindByID` для `prepared_experiments`**.

### Changed

- **Конвертер**: перенос данных из `licelformat.LicelPack` в `LidarPack` (entity) через `FromLicelPack()`.
- **DTO**: поля `TimeFrom`, `TimeTo` опциональны через `*string` → `*time.Time`.
- **REST**: путь `GET /experiments/{id}` теперь содержит `prepared_experiment_id`.

## [0.3.3] — 2026-06-02

### Changed

- **Логгер** вынесен в App struct.
- **Данные переведены на новую модель**: LidarPack/LidarFile/LidarProfile.
- **Удалены старые модели** `ChannelAvail`, `ExperimentalProfile`, `ExperimentalProfileModel`.
- **Предыдущий функционал (preprocessing)** адаптирован под новую model.

## [0.3.2] — 2026-06-01

### Added

- **LidarPack**: новая структура для хранения данных измерений.
- **Handler**: `GET /prepared/{id}` возвращает детали `prepared_experiments` (id, experiment_id, crop_alt, bgr_type, bgr_alt).

### Changed

- **API** `GET /experiments/{id}` теперь возвращает `measurement_start_time` и `measurement_stop_time`.
- **Preprocess** заполняет time-range при сохранении.

## [0.3.1] — 2026-06-01

### Added

- **MeteoDataSource + MeteoRepository**: сохранение, загрузка метео-данных.
- **Preprocessing**: теперь включает распаковку архива licel, парсинг, конвертацию метео-файла, загрузку в MinIO.
- **Поддержка fallback**: если метео-файл не приложен — используется стандартная атмосфера.

## [0.3.0] — 2026-06-01

### Added

- **`AvailableChannels`**: автоматическое извлечение каналов измерений из licel формата.

### Changed

- **Модель `Experiment`**: добавлен временной стейт `measurement_start_time`, `measurement_stop_time`.
- **Entity-папка**: перенос в `internal/domain/entity`.

### Fixed

- **Миграция**: теперь триггерится один раз при старте.

## [0.2.3] — 2026-06-01

### Added

- **Метео-данные**: парсер `meteo.csv`, сущность `MeteoRecord`.
- **`PreparedExperiment`**: готовый к визуализации срез.
- **Парсер Licel `PreprocessZip`**: визуализация аналоговых/фотонных каналов (plotly).
- **Хранение BGR**: отдельный файл BGR сохраняется и привязывается к эксперименту.

## [0.2.2] — 2026-05-31

### Changed

- **Пароль теперь хранится в bcrypt** (вместо bcrypt) — исправление опечатки в описании.
- **Поле `Status` в `experiments`** — теперь enum через pg.

## [0.2.1] — 2026-05-31

### Fixed

- **Сравнение bcrypt-хэшей** при логине (проблема с `rows` vs `row`).
- **Ошибка `pq: invalid byte sequence`** при SQL-запросах (настройка `TimeZone=Asia/Vladivostok`).
- **Тип `users.role`**: `VARCHAR(20)` добавлен явно.
- **Дубликат user** при логине — убрана мелкая ошибка с конфликтом имён.
- **Тип `model.Users`**: `Role` теперь `string`.

## [0.2.0] — 2026-05-31

### Added

- **Autovivification**: автоматическое создание таблиц через AutoMigrate, если их нет.
- **JWT-аутентификация**: логин, регистрация, refresh.
- **RBAC**: роли `admin`, `manager`, `guest` с middleware `AdminOnly`, `AdminOrManager`.
- **Эксперименты**: CRUD (admin — создание/удаление, остальные — чтение).
- **Фильтрация**: `GET /experiments` с пагинацией, сортировкой, фильтром по статусу/названию.
- **Пагинация**: дженерик-пагинация для всех списков.
- **Preprocessing flow (goroutine)**: фоновая обработка в go-рутине с обновлением статуса. Неблокирующая для клиента.

## [0.1.0] — 2026-05-30

### Added

- **Базовый CRUD Users**: create, read, update, delete (soft-delete).
- **PostgreSQL**: GORM, настройка соединения через Viper.
- **Viper конфиг**: загрузка из `.env`.
- **Chi v5 роутер**: с middleware.
- **Clean Architecture**: слои delivery, domain, infrastructure.
- **Пагинация**: для списков.
- **Логирование**: начальная настройка логгера (Logrus).
