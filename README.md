
- `cmd/app` — HTTP API сервис (Gin), инициализирует зависимости и поднимает сервер
- `cmd/worker` — фоновые задачи: истечение броней и обработка уведомлений
- `api` — HTTP-обработчики для рейсов и бронирований
- `internal/domain` — бизнес-структуры (`Flight`, `Booking`, статусы)
- `internal/repository` — работа с Postgres (flights, bookings)
- `internal/service` — бизнес-логика: кеширование рейсов, блокировки мест, управление статусами брони, публикация событий
- `internal/cache` — Redis (кеш рейсов, блокировки мест)
- `internal/kafka` — продюсер/консьюмер событий бронирования
- `internal/email` — заглушка отправки писем
- `scripts/001_init.sql` — БД


`docker-compose up -d --build`
`docker-compose exec postgres psql -U app -d airbooking -f /scripts/001_init.sql`


http://localhost:8081


curl -X POST "http://localhost:8080/api/v1/bookings" -H "Content-Type: application/ison" -d '{"flight_id": '4', "seat_number": 60, "email": "test@example.com"}'
curl -X PUT "http://localhost:8080/api/v1/bookings/" -H "Content-Type: application/json"
curl -X DELETE "http://localhost:8080/api/v1/bookings/" -H "Content-Type: application/json"


go test ./internal/service/... -v 
go test ./internal/service/... -cover