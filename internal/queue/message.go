package queue

// QueuedNotificationMessage is the structure for messages put onto the internal queues.
// It includes the NotificationID from the database and the original payload.
type QueuedNotificationMessage struct {
	NotificationID string `json:"notification_id"`
	ServiceID      string `json:"service_id"`      // e.g. "slack", "email"
	RecipientInfo  string `json:"recipient_info"`  // e.g. Slack channel, email address
	Payload        []byte `json:"payload"`         // Original payload intended for the service
}
