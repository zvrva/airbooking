# Airbooking

## API роуты
Базовый префикс: `/api/v1`.

- `GET /api/v1/flights` — список рейсов. Ответ: массив объектов с полями `id`, `from_airport`, `to_airport`, `departure_time`, `arrival_time`, `total_seats`, `available_seats`, `price_cents`, `created_at`, `updated_at`.
- `GET /api/v1/flights/{id}` — детали рейса по идентификатору.
- `POST /api/v1/bookings` — создать бронь. Тело запроса:
  ```json
  {
    "flight_id": 1,
    "seat_number": 12,
    "email": "user@example.com"
  }
  ```
  Успех `201 Created`. Ответ: `token`, `status` (`PENDING|CONFIRMED|CANCELLED|EXPIRED`), `expires_at` (RFC3339), `flight_id`, `seat_number`, `email`.
- `PUT /api/v1/bookings/{token}` — подтвердить бронь по токену до истечения `expires_at`. Ответ как выше, статус сменится на `CONFIRMED`.
- `DELETE /api/v1/bookings/{token}` — отменить бронь. Ответ как выше, статус станет `CANCELLED` (если уже отменена/просрочена — вернётся текущее состояние).

## Структура папок
- `cmd/app` — HTTP API сервис (Gin), инициализирует зависимости и поднимает сервер.
- `cmd/worker` — фоновые задачи: истечение броней и обработка уведомлений.
- `api` — HTTP-обработчики для рейсов и бронирований.
- `config` — код загрузки конфигурации YAML.
- `internal/domain` — бизнес-структуры (`Flight`, `Booking`, статусы).
- `internal/repository` — работа с Postgres (flights, bookings).
- `internal/service` — бизнес-логика: кеширование рейсов, блокировки мест, управление статусами брони, публикация событий.
- `internal/cache` — Redis (кеш рейсов, блокировки мест).
- `internal/kafka` — продюсер/консьюмер событий бронирования.
- `internal/email` — заглушка отправки писем.
- `scripts/001_init.sql` — первичная схема БД.
- `config.yaml` — пример конфигурации (HTTP, Postgres, Redis, Kafka, TTL).
- `docker-compose.yaml` — инфраструктура (Postgres, Redis, Kafka, Kafka UI) + сервисы `app` и `worker`.

## Как всё работает
- `cmd/app`: загружает `config.yaml`, создаёт подключения к Postgres и Redis, поднимает Kafka продюсер. Регистрирует Gin-маршруты `/api/v1/flights` и `/api/v1/bookings`.
- Рейсы: `FlightService` сначала пытается прочитать кеш из Redis, иначе берёт из Postgres и кладёт в кеш на `booking.flights_cache_ttl_seconds`.
- Создание брони: `BookingService.CreateBooking` ставит Redis-lock на пару (рейс, место) на `booking.hold_ttl_minutes`, генерирует токен, пытается атомарно уменьшить `available_seats` и создать запись со статусом `PENDING` в одной транзакции Postgres, публикует событие в Kafka (`booking_events_topic` и, если задан, `notifications_topic`).
- Подтверждение/отмена: по токену обновляется статус (`CONFIRMED` или `CANCELLED`), освобождается место и блокировка в Redis, публикуется событие. Отмена дополнительно возвращает место через репозиторий.
- Истечение: `BookingService.ExpirePendingBookings` помечает просроченные `PENDING` брони как `EXPIRED`, возвращает места и публикует события.
- `cmd/worker`: каждые `worker.expiration_sweep_minutes` вызывает `ExpirePendingBookings`, а Kafka-консьюмер читает `notifications_topic` и через `internal/email` (пока заглушка) выводит уведомления.
- Конфигурация: YAML описывает адрес HTTP, подключение к Postgres/Redis, Kafka брокеры и топики, TTL удержания и подтверждения брони. Путь к файлу можно задать `CONFIG_PATH`.
- Инфраструктура: `docker-compose.yaml` поднимает Postgres+Redis+Kafka+Kafka UI и два Go-контейнера (`app` и `worker`). Схема БД из `scripts/001_init.sql` загружается в Postgres (выполните скрипт после старта БД).

### Быстрый старт (docker compose)
1. `docker-compose up -d --build` — поднимает инфраструктуру, API и воркер (использует `config.yaml` в корне).
2. Инициализируйте БД: `docker-compose exec postgres psql -U app -d airbooking -f /scripts/001_init.sql`.
3. API доступно на `http://localhost:8080/api/v1`, Kafka UI — `http://localhost:8081`.
