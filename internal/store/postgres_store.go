package store

import (
	"context"
	"encoding/json"
	"log"
	// "time" // Not directly used by this file after removing Connect from postgresStore

	"github.com/jackc/pgx/v4/pgxpool"
)

// postgresStore implements the Store interface for PostgreSQL.
type postgresStore struct {
	pool *pgxpool.Pool
}

// Ensure postgresStore implements Store (compile-time check)
var _ Store = (*postgresStore)(nil)

// PostgresStoreFactory implements the StoreFactory interface for PostgreSQL.
type PostgresStoreFactory struct{}

// NewPostgresStoreFactory creates a new PostgresStoreFactory.
// This function will be called from main.go
func NewPostgresStoreFactory() StoreFactory {
	return &PostgresStoreFactory{}
}

// NewStore creates and connects a new postgresStore instance.
func (f *PostgresStoreFactory) NewStore(connectionString string) (Store, error) {
	s := &postgresStore{} // Create an instance of the store

	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}

	// You might want to configure connection pool settings here
	// config.MaxConns = 10
	// config.MinConns = 2
	// config.MaxConnLifetime = time.Hour
	// config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	s.pool = pool

	// Ping the database to verify the connection.
	if err = s.pool.Ping(context.Background()); err != nil {
		s.pool.Close() // Close the pool if ping fails
		return nil, err
	}

	log.Println("Successfully connected to PostgreSQL.")
	return s, nil
}

// Ensure PostgresStoreFactory implements StoreFactory (compile-time check)
var _ StoreFactory = (*PostgresStoreFactory)(nil)


// Close closes the database connection pool. This is part of the Store interface.
func (s *postgresStore) Close() error {
	if s.pool != nil {
		s.pool.Close()
		log.Println("PostgreSQL connection pool closed.")
	}
	return nil
}

// RecordNotification inserts a new notification record into the database.
// It returns the UUID of the newly created record.
func (s *postgresStore) RecordNotification(ctx context.Context, serviceID string, payload []byte, recipientInfo string) (string, error) {
	var id string
	// Ensure payload is valid JSON before inserting into JSONB
	if !json.Valid(payload) {
		log.Printf("Warning: PostgreSQL store: payload for service %s is not valid JSON: %s", serviceID, string(payload))
	}

	query := `
		INSERT INTO notifications (service_id, payload, recipient_info, status, created_at, updated_at, attempts)
		VALUES ($1, $2, $3, $4, NOW(), NOW(), 0)
		RETURNING id`

	var recipientSQL *string // Use *string to handle potential NULL for recipient_info
	if recipientInfo != "" {
		recipientSQL = &recipientInfo
	}

	err := s.pool.QueryRow(ctx, query, serviceID, payload, recipientSQL, "queued").Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateNotificationStatus updates the status, attempts, and error message of an existing notification.
func (s *postgresStore) UpdateNotificationStatus(ctx context.Context, notificationID string, status string, attempts int, errorMessage string) error {
	query := `
		UPDATE notifications
		SET status = $1, attempts = $2, error_message = $3, last_attempt_at = NOW(), updated_at = NOW()
		WHERE id = $4`

	var errMsgSQL *string // Use *string to handle potential NULL for error_message
	if errorMessage != "" {
		errMsgSQL = &errorMessage
	}

	_, err := s.pool.Exec(ctx, query, status, attempts, errMsgSQL, notificationID)
	return err
}

// GetNotification retrieves a notification by its ID.
func (s *postgresStore) GetNotification(ctx context.Context, notificationID string) (*Notification, error) {
	query := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications
		WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, notificationID)
	n := &Notification{}
	var payloadBytes []byte

	err := row.Scan(
		&n.ID,
		&n.ServiceID,
		&n.RecipientInfo, // Directly scan into *string
		&payloadBytes,
		&n.Status,
		&n.Attempts,
		&n.LastAttemptAt,  // Directly scan into *time.Time
		&n.CreatedAt,
		&n.UpdatedAt,
		&n.ErrorMessage,   // Directly scan into *string
	)
	if err != nil {
		// pgx.ErrNoRows is handled correctly by returning err
		return nil, err
	}
	n.Payload = payloadBytes
	return n, nil
}

// ListNotifications retrieves a paginated list of notifications, ordered by creation date descending.
// It also returns the total count of notifications matching the criteria (ignoring pagination).
func (s *postgresStore) ListNotifications(ctx context.Context, limit int, offset int) ([]Notification, int, error) {
	countQuery := `SELECT COUNT(*) FROM notifications`
	var totalCount int
	err := s.pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	if totalCount == 0 || offset >= totalCount {
		return []Notification{}, totalCount, nil // No notifications or offset is out of bounds
	}

	listQuery := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, totalCount, err
	}
	defer rows.Close()

	notifications := make([]Notification, 0, limit) // Pre-allocate slice capacity
	for rows.Next() {
		n := Notification{}
		var payloadBytes []byte
		err := rows.Scan(
			&n.ID,
			&n.ServiceID,
			&n.RecipientInfo,
			&payloadBytes,
			&n.Status,
			&n.Attempts,
			&n.LastAttemptAt,
			&n.CreatedAt,
			&n.UpdatedAt,
			&n.ErrorMessage,
		)
		if err != nil {
			// Log individual row scan error, but try to continue if possible,
			// or return immediately depending on desired error handling.
			log.Printf("Error scanning notification row in ListNotifications (PostgreSQL): %v", err)
			// For simplicity, returning error here. A more robust app might collect valid rows.
			return nil, totalCount, err
		}
		n.Payload = payloadBytes
		notifications = append(notifications, n)
	}

	if err = rows.Err(); err != nil {
		return nil, totalCount, err
	}

	return notifications, totalCount, nil
}
