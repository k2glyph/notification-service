package store

import (
	"context"
	"time"
)

// Notification represents a notification record in the database.
type Notification struct {
	ID             string    `json:"id"`
	ServiceID      string    `json:"service_id"`
	RecipientInfo  *string   `json:"recipient_info"` // Pointer to handle NULL
	Payload        []byte    `json:"payload"`        // Stored as JSONB, handled as []byte
	Status         string    `json:"status"`
	Attempts       int       `json:"attempts"`
	LastAttemptAt  *time.Time `json:"last_attempt_at"` // Pointer to handle NULL
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ErrorMessage   *string   `json:"error_message"`   // Pointer to handle NULL
}

// Store defines the interface for database operations.
// The Connect method is removed as connection is handled by the StoreFactory's NewStore method.
type Store interface {
	Close() error
	RecordNotification(ctx context.Context, serviceID string, payload []byte, recipientInfo string) (notificationID string, err error)
	UpdateNotificationStatus(ctx context.Context, notificationID string, status string, attempts int, errorMessage string) error
	GetNotification(ctx context.Context, notificationID string) (*Notification, error)
	ListNotifications(ctx context.Context, limit int, offset int) ([]Notification, int, error) // Returns notifications, total count, error
}

// StoreFactory defines the interface for creating Store instances.
type StoreFactory interface {
	NewStore(connectionString string) (Store, error)
}

// Ensure postgresStore (defined in postgres_store.go) implements Store (compile-time check)
// and mysqlStore (defined in mysql_store.go) implements Store.
// These checks will remain in their respective concrete implementation files.
// We need to refer to the concrete type if it's not defined in this file,
// or we can rely on the assignment in NewPostgresStore.
// For a clean interface file, it's often better to not have concrete types here.
// The check `var _ Store = (*postgresStore)(nil)` should be in postgres_store.go
// if postgresStore is not defined here.
// However, since NewPostgresStore returns Store, the compiler will check
// that NewPostgresStoreConcrete() returns a type compatible with Store.
