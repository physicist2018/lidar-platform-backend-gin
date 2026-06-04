# LiDAR Platform API · Документация

> **Базовый URL:** `http://lidarbaclup.dvo.ru:18080`
>
> **Аутентификация:** JWT Bearer Token (HS256)
>
> **Content-Type:** `application/json` (кроме multipart-загрузок)

---

## Оглавление

1. [Аутентификация](#1-аутентификация)
2. [Пользователи (Users)](#2-пользователи-users)
3. [Эксперименты (Experiments)](#3-эксперименты-experiments)
4. [Подготовка данных (Prepare)](#4-подготовка-данных-prepare)
5. [Визуализация (Visualize)](#5-визуализация-visualize)
6. [Роли и доступ](#6-роли-и-доступ)
7. [Обработка ошибок](#7-обработка-ошибок)
8. [Рабочий процесс (Workflow)](#8-рабочий-процесс-workflow)

---

## 1. Аутентификация

### 1.1. Логин

Получение JWT-токена по email и паролю.

```
POST /auth/login
```

**Тело запроса:**

```json
{
  "email": "admin@lidar-platform.io",
  "password": "admin123"
}
```

**Успешный ответ `200`:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "Admin",
    "email": "admin@lidar-platform.io",
    "role": "admin"
  }
}
```

**Ошибки:**

| Код | Причина |
|-----|---------|
| `400` | Неверный формат запроса (email невалиден, пароль пустой) |
| `401` | Неверный email или пароль |

**Использование токена:** во всех остальных запросах передавать в заголовке:

```
Authorization: Bearer <token>
```

---

## 2. Пользователи (Users)

### 2.1. Список пользователей

```
GET /users
```

**Query-параметры:**

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|-------------|----------|
| `page` | int | `1` | Номер страницы (≥1) |
| `limit` | int | `10` | Элементов на странице (1–100) |
| `sort` | string | — | `asc` / `desc` (по ID) |
| `role` | string | — | Фильтр: `admin`, `guest`, `manager` |
| `name` | string | — | Частичное совпадение по имени (ILIKE) |
| `email` | string | — | Частичное совпадение по email (ILIKE) |

**Пример:** `GET /users?page=1&limit=20&role=admin`

**Ответ `200`:**

```json
{
  "data": [
    {
      "id": 1,
      "name": "Admin",
      "email": "admin@lidar-platform.io",
      "role": "admin"
    }
  ],
  "page": 1,
  "limit": 20,
  "total_items": 1,
  "total_pages": 1
}
```

### 2.2. Получить пользователя по ID

```
GET /users/:id
```

**Пример:** `GET /users/1`

**Ответ `200`:** см. `UserResponse` выше.

**Ошибка `404`:** пользователь не найден.

### 2.3. Создать пользователя

> 🔒 **Роль:** `admin`

```
POST /users
```

**Тело запроса:**

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "secret123",
  "role": "manager"
}
```

| Поле | Ограничения |
|------|-------------|
| `name` | 1–100 символов, обязательно |
| `email` | Валидный email, уникальный, обязательно |
| `password` | 6–255 символов, обязательно |
| `role` | `admin` / `guest` / `manager`, обязательно |

**Ответ `201`:** см. `UserResponse`.

**Ошибки:**

| Код | Причина |
|-----|---------|
| `400` | Валидация не пройдена |
| `403` | Не хватает прав (не admin) |
| `409` | Email уже существует |

### 2.4. Обновить пользователя

> 🔒 **Роль:** `admin`

```
PUT /users/:id
```

**Тело запроса:**

```json
{
  "name": "John Updated",
  "email": "john@example.com",
  "password": "",
  "role": "manager"
}
```

> Если `password` пустой — пароль **не меняется**.

**Ответ `200`:** см. `UserResponse`.

### 2.5. Удалить пользователя

> 🔒 **Роль:** `admin`

```
DELETE /users/:id
```

> ⚠️ Нельзя удалить самого себя (даже админу).

**Ответ:** `204 No Content` (успех), `403` (при попытке удалить себя).

---

## 3. Эксперименты (Experiments)

### 3.1. Создать эксперимент

> 🔒 **Роль:** `admin`

```
POST /experiments
Content-Type: multipart/form-data
```

**Поля формы:**

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| `title` | string | ✅ | Название эксперимента |
| `comments` | string | ❌ | Комментарий |
| `licelZip` | file | ✅ | ZIP-архив с licel-файлами измерений |
| `licelBgr` | file | ✅ | Файл фона (BGR) |
| `meteoFile` | file | ✅ | Файл метеоданных |

**cURL-пример:**

```bash
curl -X POST http://lidarbaclup.dvo.ru:18080/experiments \
  -H "Authorization: Bearer <token>" \
  -F "title=Test Experiment" \
  -F "comments=Night measurement" \
  -F "licelZip=@measurements.zip" \
  -F "licelBgr=@bgr.dat" \
  -F "meteoFile=@meteo.txt"
```

**Ответ `201`:**

```json
{
  "id": 42,
  "user_id": 1,
  "title": "Test Experiment",
  "comments": "Night measurement",
  "measurement_start_time": null,
  "measurement_stop_time": null,
  "licel_zip_path": "",
  "licel_bgr_path": "",
  "meteo_file_path": "",
  "status": "staged",
  "error_msg": null,
  "created_at": "2026-06-02T12:00:00Z",
  "updated_at": "2026-06-02T12:00:00Z"
}
```

> ⚡ Эксперимент создаётся сразу со статусом `staged`. Препроцессинг (парсинг licel zip, загрузка в MinIO, извлечение каналов) выполняется **асинхронно** в фоне.
>
> Статусная машина: `staged → uploading → done` (или `failed` при ошибке).

### 3.2. Список экспериментов

```
GET /experiments
```

**Query-параметры:**

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|-------------|----------|
| `page` | int | `1` | Номер страницы |
| `limit` | int | `10` | Элементов на странице |
| `sort` | string | — | `asc` / `desc` |
| `status` | string | — | Фильтр: `staged`, `uploading`, `done`, `failed` |
| `title` | string | — | Частичное совпадение (ILIKE) |

**Пример:** `GET /experiments?status=done&sort=asc`

**Ответ `200`:**

```json
{
  "data": [
    {
      "id": 42,
      "user_id": 1,
      "title": "Test Experiment",
      "comments": "Night measurement",
      "measurement_start_time": "2026-06-01T22:00:00Z",
      "measurement_stop_time": "2026-06-02T04:00:00Z",
      "licel_zip_path": "experiments/42/source/licel.zip",
      "licel_bgr_path": "experiments/42/source/bgr.dat",
      "meteo_file_path": "experiments/42/source/meteo.txt",
      "status": "done",
      "error_msg": null,
      "created_at": "2026-06-02T12:00:00Z",
      "updated_at": "2026-06-02T12:01:30Z"
    }
  ],
  "page": 1,
  "limit": 10,
  "total_items": 1,
  "total_pages": 1
}
```

### 3.3. Получить эксперимент по ID

```
GET /experiments/:id
```

**Ответ `200`:** см. структуру `ExperimentResponse` выше.

### 3.4. Каналы эксперимента

```
GET /experiments/:id/channels
```

Возвращает измерительные каналы, обнаруженные при препроцессинге licel-файлов. Каналы дедуплицированы по `(wavelen, polarization, isPhoton)`.

**Ответ `200`:**

```json
{
  "channels": [
    { "wavelen": 355.0, "polarization": "p(arallel)", "isPhoton": 0, "isActive": 1 },
    { "wavelen": 355.0, "polarization": "s(enkrecht)",    "isPhoton": 0, "isActive": 1 },
    { "wavelen": 355.0, "polarization": "o(no polaroid)", "isPhoton": 1, "isActive": 0 },
    { "wavelen": 1064.0,"polarization": "parallel", "isPhoton": 0, "isActive": 1 }
  ]
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `wavelen` | float64 | Длина волны (нм) |
| `polarization` | string | Поляризация (`p(arallel)`, `s(enkrecht)`, `o(no polaroid)` др.) |
| `isPhoton` | int | `0` = аналоговый, `1` = фотонный |
| `isActive` | int | `0` = канал неактивен (нет сигнала), `1` = активен |

---

## 4. Подготовка данных (Prepare)

> 🔒 **Роль:** `admin` или `manager`

Запускает асинхронный пайплайн: вычитание фона → обрезка по высоте → сохранение в MinIO.

```
POST /experiments/:id/prepare
```

**Тело запроса:**

```json
{
  "crop_alt": 15000.0,
  "bgr_type": "avgTail",
  "bgr_alt": 12000.0
}
```

| Поле | Тип | Обязательно | Описание |
|------|-----|-------------|----------|
| `crop_alt` | float64 | ✅ | Максимальная дистанция (м), ≥0 |
| `bgr_type` | string | ✅ | Стратегия вычитания фона |
| `bgr_alt` | float64 | условно | Высота хвоста для `avgTail`/`medTail`, >0 |

**`bgr_type` — стратегии:**

| Значение | Описание |
|----------|----------|
| `file` | Поэлементное вычитание из BGR-файла |
| `avgTail` | Среднее значение хвоста (требуется `bgr_alt > 0`) |
| `medTail` | Медианное значение хвоста (требуется `bgr_alt > 0`) |

**Ответ `201`:**

```json
{
  "id": 10,
  "user_id": 1,
  "experiment_id": 42,
  "crop_alt": 15000.0,
  "bgr_type": "avgTail",
  "bgr_alt": 12000.0,
  "path_to_data": "",
  "status": "staged",
  "error_msg": null
}
```

> ⚡ Статусная машина: `staged → removebgr → cropping → done` (или `failed`). После завершения `path_to_data` содержит путь к обработанному zip в MinIO.

**Мониторинг:** проверяйте статус через `GET /experiments/:id` (поле `status` эксперимента не меняется — следить через `GET` подготовленного эксперимента пока не реализован отдельный эндпоинт).

---

## 5. Визуализация (Visualize)

> 🔒 **Роль:** `admin` или `manager`

Генерирует heatmap или усреднённый профиль по подготовленным данным. Результат кешируется в MinIO и возвращается в виде presigned URL.

```
GET /prepared/:id/:wavelen/:photon/:polarization/:action
```

### Параметры пути

| Параметр | Тип | Пример | Описание |
|----------|-----|--------|----------|
| `:id` | uint | `10` | ID подготовленного эксперимента |
| `:wavelen` | float64 | `532` | Длина волны (нм) |
| `:photon` | int | `0` | `0` = аналоговый, `1` = фотонный |
| `:polarization` | string | `parallel` | Поляризация |
| `:action` | string | `image` | Тип визуализации: `image` или `profile` |

### Query-параметры

| Параметр | Тип | По умолчанию | Описание |
|----------|-----|-------------|----------|
| `type` | string | `svg` | Формат вывода: `svg`, `png`, `json` |
| `formula` | string | `raw` | Формула сигнала |
| `regenerate` | bool | `false` | `true` — перерисовать в обход кеша |

**`formula` — формулы сигнала:**

| Значение | Формула | Описание |
|----------|---------|----------|
| `raw` | P | Сырой сигнал |
| `rangecorr` | P × r² | Коррекция на расстояние |
| `lograngecorr` | log₁₀(P × r²) | Логарифмическая коррекция |

### Ответ `200`:

```json
{
  "url": "http://minio:9000/lidar-experiments/experiments/42/images/image-532.0-parallel-0-raw.svg?X-Amz-Algorithm=AWS4-HMAC-SHA256&..."
}
```

> ⏰ Presigned URL действителен **1 час**. После истечения нужно запросить заново.
>
> 💾 При `regenerate=false` (по умолчанию) график **не пересчитывается**, если уже был построен с такими параметрами — возвращается кешированный URL.

### Кеширование

Записи о построенных графиках хранятся в таблице `experiment_charts`. Ключ кеша: `(experiment_id, chart_type, formula, wavelen, polarization, is_photon)`. Файлы в MinIO: `experiments/{id}/images/{action}-{wavelen}-{polarization}-{photon}-{formula}.{ext}`.

### Примеры запросов

```bash
# Heatmap, SVG, raw signal (первый запрос — построит и закеширует)
curl "http://lidarbaclup.dvo.ru:18080/prepared/10/532/0/parallel/image?type=svg&formula=raw" \
  -H "Authorization: Bearer <token>"

# Profile, PNG, range-corrected
curl "http://lidarbaclup.dvo.ru:18080/prepared/10/355/1/cross/profile?type=png&formula=rangecorr" \
  -H "Authorization: Bearer <token>"

# Plotly JSON (для интерактивного графика во фронтенде)
curl "http://lidarbaclup.dvo.ru:18080/prepared/10/532/0/parallel/image?type=json&formula=lograngecorr" \
  -H "Authorization: Bearer <token>"

# Принудительная перерисовка
curl "http://lidarbaclup.dvo.ru:18080/prepared/10/532/0/parallel/image?type=svg&formula=raw&regenerate=true" \
  -H "Authorization: Bearer <token>"
```

---

## 6. Роли и доступ

| Роль | Описание | Доступ |
|------|----------|--------|
| `admin` | Администратор | Всё (пользователи, эксперименты, подготовка, визуализация) |
| `manager` | Оператор | Чтение экспериментов, подготовка, визуализация. **Не может** создавать эксперименты и управлять пользователями |
| `guest` | Гость | Только чтение (пользователи, эксперименты) |

### Матрица доступов

| Действие | guest | manager | admin |
|----------|:-----:|:-------:|:-----:|
| `POST /auth/login` | ✅ | ✅ | ✅ |
| `GET /users`, `GET /users/:id` | ✅ | ✅ | ✅ |
| `POST /users`, `PUT /users/:id`, `DELETE /users/:id` | ❌ | ❌ | ✅ |
| `GET /experiments`, `GET /experiments/:id`, `GET /experiments/:id/channels` | ✅ | ✅ | ✅ |
| `POST /experiments` | ❌ | ❌ | ✅ |
| `POST /experiments/:id/prepare` | ❌ | ✅ | ✅ |
| `GET /prepared/:id/...` | ❌ | ✅ | ✅ |

---

## 7. Обработка ошибок

Все ошибки возвращаются в формате:

```json
{
  "error": "Bad Request",
  "message": "title is required"
}
```

### Стандартные HTTP-коды

| Код | Описание |
|-----|----------|
| `200` | Успех |
| `201` | Создано |
| `204` | Успех без тела (например, DELETE) |
| `400` | Ошибка валидации запроса |
| `401` | Не аутентифицирован (отсутствует / истёк токен) |
| `403` | Недостаточно прав |
| `404` | Ресурс не найден |
| `409` | Конфликт (например, эксперимент не готов для подготовки) |
| `500` | Внутренняя ошибка сервера |

---

## 8. Рабочий процесс (Workflow)

Полный цикл работы с экспериментом:

```
1. POST /auth/login                          → получаем JWT-токен
2. POST /experiments                         → создаём эксперимент (status: staged)
   (multipart: licel.zip, bgr.dat, meteo.txt)
3. GET /experiments/:id                      → ждём status: done (~30 сек на препроцессинг)
4. GET /experiments/:id/channels             → смотрим доступные каналы
5. POST /experiments/:id/prepare             → запускаем обработку (вычитание фона+обрезка)
   { crop_alt: 15000, bgr_type: "avgTail", bgr_alt: 12000 }
6. GET /prepared/:prep_id/:wavelen/:photon/:polarization/image   → визуализация
   ?type=svg&formula=raw → получаем URL графика
```

**Статусы эксперимента:**

```
staged → uploading → done
                  → failed (+ error_msg)
```

**Статусы подготовки:**

```
staged → removebgr → cropping → done
                               → failed (+ error_msg)
```
