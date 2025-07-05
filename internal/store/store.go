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

	// --- Methods for Dashboard ---
	GetDashboardGlobalStats(ctx context.Context, dateRangeStart, dateRangeEnd *time.Time) (*DashboardGlobalStats, error)
	GetDashboardChannelStats(ctx context.Context, dateRangeStart, dateRangeEnd *time.Time) ([]DashboardChannelStat, error)
	ListDashboardActivity(ctx context.Context, params DashboardActivityParams) ([]Notification, int, error)
	GetDashboardVolumeTrend(ctx context.Context, period string, startDate, endDate time.Time) ([]DashboardHistoricalPoint, error)
	GetDashboardFailureRateTrend(ctx context.Context, period string, startDate, endDate time.Time) ([]DashboardHistoricalPoint, error)
	GetDashboardQueueSizeTrend(ctx context.Context, period string, startDate, endDate time.Time) ([]DashboardHistoricalPoint, error)
	GetDashboardFailureReasons(ctx context.Context, dateRangeStart, dateRangeEnd *time.Time, limit int) ([]DashboardFailureReason, error)
}

// --- Structs for Dashboard Data ---
type DashboardGlobalStats struct {
	TotalSent            int64   `json:"totalSent"`
	TotalFailed          int64   `json:"totalFailed"`
	NotificationsInQueue int64   `json:"notificationsInQueue"`
	AvgDeliveryTimeSec   float64 `json:"avgDeliveryTimeSec"`
	SuccessRate          float64 `json:"successRate"`
}

type DashboardChannelStat struct {
	Channel     string  `json:"channel"`
	Total       int64   `json:"total"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"successRate"`
}

type DashboardActivityParams struct {
	Limit         int
	Offset        int
	StartDate     *time.Time
	EndDate       *time.Time
	ChannelFilter string
	StatusFilter  string
	SearchTerm    string
}

type DashboardHistoricalPoint struct {
	Date      string `json:"date"` // YYYY-MM-DD or YYYY-MM-DD HH:00
	Volume    int64  `json:"volume"`
	Failures  int64  `json:"failures,omitempty"`
	QueueSize int64  `json:"queueSize,omitempty"`
}

type DashboardFailureReason struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
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
