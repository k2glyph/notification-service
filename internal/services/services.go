package services

import (
	"context"
	"encoding/json"
	"fmt"

	"log" // Added for potential logging in services

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/store" // Added store
)

type PushService interface {
	fmt.Stringer
	ID() string
	// Serve now accepts a store.Store instance
	Serve(ctx context.Context, q queue.Queue, s store.Store, fc FeedbackCollector) error
}

// FeedbackCollector might become less relevant if all feedback is through the store,
// or it could be used for other types of feedback to the core server.
// For now, it's kept as is.
type FeedbackCollector interface {
}

// Helper function to parse the QueuedNotificationMessage
// This can be used by service implementations.
func ParseQueuedMessage(data []byte) (*queue.QueuedNotificationMessage, error) {
	var qnm queue.QueuedNotificationMessage
	if err := json.Unmarshal(data, &qnm); err != nil {
		log.Printf("Error unmarshaling QueuedNotificationMessage: %v, data: %s", err, string(data))
		return nil, err
	}
	// Basic validation
	if qnm.NotificationID == "" || qnm.ServiceID == "" || len(qnm.Payload) == 0 {
		log.Printf("Invalid QueuedNotificationMessage: missing NotificationID, ServiceID, or Payload. Data: %+v", qnm)
		return nil, fmt.Errorf("invalid QueuedNotificationMessage: missing NotificationID, ServiceID, or Payload")
	}
	return &qnm, nil
}
