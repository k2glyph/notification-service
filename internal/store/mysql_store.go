package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time" // Required for db.SetConnMaxLifetime, etc.

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/google/uuid"          // For generating UUIDs
)

// mysqlStore implements the Store interface for MySQL.
type mysqlStore struct {
	db *sql.DB
}

// Ensure mysqlStore implements Store (compile-time check)
var _ Store = (*mysqlStore)(nil)

// MySQLStoreFactory implements the StoreFactory interface for MySQL.
type MySQLStoreFactory struct{}

// NewMySQLStoreFactory creates a new MySQLStoreFactory.
// This function will be called from main.go
func NewMySQLStoreFactory() StoreFactory {
	return &MySQLStoreFactory{}
}

// NewStore creates and connects a new mysqlStore instance.
func (f *MySQLStoreFactory) NewStore(connectionString string) (Store, error) {
	s := &mysqlStore{} // Create an instance of the store

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	// Configure connection pool settings (optional, but recommended)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Ping the database to verify the connection.
	if err = db.PingContext(context.Background()); err != nil {
		db.Close() // Close the db if ping fails
		return nil, err
	}
	s.db = db

	log.Println("Successfully connected to MySQL.")
	return s, nil
}

// Ensure MySQLStoreFactory implements StoreFactory (compile-time check)
var _ StoreFactory = (*MySQLStoreFactory)(nil)

// Close closes the database connection. This is part of the Store interface.
func (s *mysqlStore) Close() error {
	if s.db != nil {
		log.Println("MySQL connection pool closed.")
		return s.db.Close()
	}
	return nil
}

// RecordNotification inserts a new notification record into the database.
// It generates a UUID for the id.
func (s *mysqlStore) RecordNotification(ctx context.Context, serviceID string, payload []byte, recipientInfo string) (string, error) {
	newID := uuid.New().String()

	if !json.Valid(payload) {
		log.Printf("Warning: MySQL store: payload for service %s is not valid JSON: %s", serviceID, string(payload))
	}

	query := `
		INSERT INTO notifications (id, service_id, payload, recipient_info, status, created_at, updated_at, attempts)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP(6), CURRENT_TIMESTAMP(6), 0)`

	var recipientSQL sql.NullString
	if recipientInfo != "" {
		recipientSQL.String = recipientInfo
		recipientSQL.Valid = true
	}

	_, err := s.db.ExecContext(ctx, query, newID, serviceID, payload, recipientSQL, "queued")
	if err != nil {
		return "", err
	}
	return newID, nil
}

// UpdateNotificationStatus updates the status, attempts, and error message of an existing notification.
func (s *mysqlStore) UpdateNotificationStatus(ctx context.Context, notificationID string, status string, attempts int, errorMessage string) error {
	query := `
		UPDATE notifications
		SET status = ?, attempts = ?, error_message = ?, last_attempt_at = CURRENT_TIMESTAMP(6)
		WHERE id = ?`
		// updated_at is handled by ON UPDATE CURRENT_TIMESTAMP(6) in MySQL schema

	var errMsgSQL sql.NullString
	if errorMessage != "" {
		errMsgSQL.String = errorMessage
		errMsgSQL.Valid = true
	}

	_, err := s.db.ExecContext(ctx, query, status, attempts, errMsgSQL, notificationID)
	return err
}

// GetNotification retrieves a notification by its ID.
func (s *mysqlStore) GetNotification(ctx context.Context, notificationID string) (*Notification, error) {
	query := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications
		WHERE id = ?`

	row := s.db.QueryRowContext(ctx, query, notificationID)
	n := &Notification{}
	var payloadBytes []byte
	var recipientInfoSQL sql.NullString
	var lastAttemptAtSQL sql.NullTime
	var errorMsgSQL sql.NullString

	err := row.Scan(
		&n.ID,
		&n.ServiceID,
		&recipientInfoSQL,
		&payloadBytes,
		&n.Status,
		&n.Attempts,
		&lastAttemptAtSQL,
		&n.CreatedAt,
		&n.UpdatedAt,
		&errorMsgSQL,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	n.Payload = payloadBytes
	if recipientInfoSQL.Valid {
		n.RecipientInfo = &recipientInfoSQL.String
	}
	if lastAttemptAtSQL.Valid {
		n.LastAttemptAt = &lastAttemptAtSQL.Time
	}
	if errorMsgSQL.Valid {
		n.ErrorMessage = &errorMsgSQL.String
	}

	return n, nil
}

// ListNotifications retrieves a paginated list of notifications from MySQL, ordered by creation date descending.
// It also returns the total count of notifications.
func (s *mysqlStore) ListNotifications(ctx context.Context, limit int, offset int) ([]Notification, int, error) {
	countQuery := `SELECT COUNT(*) FROM notifications`
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	if totalCount == 0 || offset >= totalCount {
		return []Notification{}, totalCount, nil
	}

	listQuery := `
		SELECT id, service_id, recipient_info, payload, status, attempts, last_attempt_at, created_at, updated_at, error_message
		FROM notifications
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, listQuery, limit, offset)
	if err != nil {
		return nil, totalCount, err
	}
	defer rows.Close()

	notifications := make([]Notification, 0, limit)
	for rows.Next() {
		n := Notification{}
		var payloadBytes []byte
		var recipientInfoSQL sql.NullString
		var lastAttemptAtSQL sql.NullTime
		var errorMsgSQL sql.NullString

		err := rows.Scan(
			&n.ID,
			&n.ServiceID,
			&recipientInfoSQL,
			&payloadBytes,
			&n.Status,
			&n.Attempts,
			&lastAttemptAtSQL,
			&n.CreatedAt,
			&n.UpdatedAt,
			&errorMsgSQL,
		)
		if err != nil {
			log.Printf("Error scanning notification row in ListNotifications (MySQL): %v", err)
			return nil, totalCount, err
		}
		n.Payload = payloadBytes
		if recipientInfoSQL.Valid {
			n.RecipientInfo = &recipientInfoSQL.String
		}
		if lastAttemptAtSQL.Valid {
			n.LastAttemptAt = &lastAttemptAtSQL.Time
		}
		if errorMsgSQL.Valid {
			n.ErrorMessage = &errorMsgSQL.String
		}
		notifications = append(notifications, n)
	}

	if err = rows.Err(); err != nil {
		return nil, totalCount, err
	}

	return notifications, totalCount, nil
}
