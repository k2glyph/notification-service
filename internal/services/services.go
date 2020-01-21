package services

import (
	"context"
	"fmt"

	"github.com/k2glyph/notification-service/internal/queue"
)

type PushService interface {
	fmt.Stringer
	ID() string
	Serve(ctx context.Context, q queue.Queue, fc FeedbackCollector) error
}
type FeedbackCollector interface {
}
