CREATE TABLE IF NOT EXISTS airports (
    code VARCHAR(10) PRIMARY KEY,
    name TEXT NOT NULL,
    city TEXT,
    country TEXT
);

CREATE TABLE IF NOT EXISTS flights (
    id SERIAL PRIMARY KEY,
    from_airport VARCHAR(10) NOT NULL REFERENCES airports(code),
    to_airport VARCHAR(10) NOT NULL REFERENCES airports(code),
    departure_time TIMESTAMPTZ NOT NULL,
    arrival_time TIMESTAMPTZ NOT NULL,
    total_seats INT NOT NULL,
    available_seats INT NOT NULL,
    price_cents BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bookings (
    id SERIAL PRIMARY KEY,
    flight_id INT NOT NULL REFERENCES flights(id) ON DELETE CASCADE,
    seat_number INT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_flight_seat ON bookings (flight_id, seat_number);

CREATE UNIQUE INDEX idx_bookings_flight_seat_active 
ON bookings(flight_id, seat_number) 
WHERE status IN ('pending', 'confirmed');
