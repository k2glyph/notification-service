CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(36) PRIMARY KEY,
    service_id TEXT NOT NULL,
    recipient_info TEXT,
    payload JSON, -- Using JSON type, ensure MySQL version supports it (5.7.8+)
    status TEXT NOT NULL,
    attempts INTEGER DEFAULT 0,
    last_attempt_at DATETIME(6), -- DATETIME with microsecond precision
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    error_message TEXT
);

-- Note on UUIDs:
-- MySQL doesn't have a native UUID type that auto-generates like PostgreSQL's uuid_generate_v4().
-- The application will be responsible for generating UUIDs (e.g., using Google's UUID library or similar)
-- and inserting them as strings into the VARCHAR(36) `id` column.

-- Note on TEXT vs VARCHAR:
-- TEXT is used for service_id, recipient_info, status, error_message for flexibility.
-- If there are known max lengths, VARCHAR could be used for minor performance benefits on some MySQL versions/configs.

-- Note on Timezones:
-- DATETIME stores point-in-time values without explicit timezone information.
-- It assumes the timezone of the MySQL server session.
-- If strict timezone handling across different server/client timezones is critical,
-- consider storing as UTC and converting in the application, or using TIMESTAMP
-- with careful understanding of MySQL's TIMESTAMP behavior regarding timezones.
-- For this application, DATETIME(6) with CURRENT_TIMESTAMP(6) should be acceptable if
-- all app servers and DB server operate effectively in UTC or a consistent timezone.
-- The `parseTime=true` in the DSN for Go driver helps in converting DB time types to Go's time.Time.
