package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/k2glyph/notification-service/internal/services"
	"github.com/k2glyph/notification-service/internal/store" // Added store

	"github.com/k2glyph/notification-service/internal/queue"
)

func NewSlack(webhookUrl string) (slack *Slack, err error) {
	slack = &Slack{
		webhookUrl: webhookUrl,
		transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return
}
func (slack *Slack) ID() string {
	return "slack"
}

func (slack *Slack) String() string {
	return "SLACK"
}
func (slack *Slack) push(msg slackMessage) (done, retry bool) {
	slackBody, _ := json.Marshal(msg)
	req, err := http.NewRequest(http.MethodPost, slack.webhookUrl, bytes.NewBuffer(slackBody))
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return false, true
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(slack, "Error creating post request", err)
		return false, true
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		log.Println(slack, "rejected, status code:", resp.StatusCode)
		return true, false
	}
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		log.Println(slack, "upstream error, status code:", resp.StatusCode)
		return false, true
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		log.Println("Non-ok response return from slack")
		return false, true
	}
	return true, false
}
func (slack *Slack) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil { // This remove might be optional if DB state is the source of truth
		log.Println(slack, "error removing from the queue:", err)
	}
}

// ServeClient processes messages from the queue for Slack
func (slack *Slack) ServeClient(ctx context.Context, q queue.Queue, s store.Store) (err error) {
	defer func() {
		slack.wg.Done()
	}()

	currentAttempts := 0 // Local attempt counter for the current message being processed by this goroutine

	for ctx.Err() == nil {
		queuedMsgInterface, err := q.Get(ctx)
		if err != nil {
			if ctx.Err() != nil { // Context cancelled, normal shutdown
				return nil
			}
			log.Println(slack, "Error reading from queue", err)
			time.Sleep(1 * time.Second) // Avoid fast spin on queue error
			continue
		}

		rawMsgBytes := queuedMsgInterface.Message()
		qnm, err := services.ParseQueuedMessage(rawMsgBytes)
		if err != nil {
			log.Printf("%s: Error parsing QueuedNotificationMessage: %v. Raw: %s. Discarding.", slack, err, string(rawMsgBytes))
			slack.remove(q, queuedMsgInterface) // Malformed, cannot get DB ID.
			continue
		}

		// Fetch current attempts from DB for more robust counting if desired, for now using local
		// For simplicity, we'll use a local attempt counter that resets per message from queue.
		// A more robust system might fetch `notification.Attempts` from qnm.NotificationID via store.GetNotification
		// and increment that. Here, `currentAttempts` will reflect processing attempts by *this worker cycle*.
		currentAttempts = 1 // Reset for new message from queue.
		s.UpdateNotificationStatus(ctx, qnm.NotificationID, "processing", currentAttempts, "")

		var servicePayload slackMessage
		if err := json.Unmarshal(qnm.Payload, &servicePayload); err != nil {
			log.Printf("%s: Error unmarshaling Slack payload for NotifID %s: %v. Payload: %s", slack, qnm.NotificationID, err, string(qnm.Payload))
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "failed_parse", currentAttempts, err.Error())
			slack.remove(q, queuedMsgInterface) // Unrecoverable payload error for this service
			continue
		}

		if servicePayload.Channel == "" && qnm.RecipientInfo != "" {
			servicePayload.Channel = qnm.RecipientInfo
		}

		done, retry := slack.push(servicePayload)

		if done {
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "sent", currentAttempts, "")
			slack.remove(q, queuedMsgInterface)
		} else if retry {
			currentAttempts++ // Increment attempt for this processing cycle
			log.Printf("%s: Retrying NotifID %s (local attempts: %d)", slack, qnm.NotificationID, currentAttempts)
			// Update status to reflect it's queued for retry, and include current attempt count
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "retry_queued", currentAttempts, "Service requested retry")

			if errRequeue := q.Requeue(queuedMsgInterface); errRequeue != nil {
				log.Printf("%s: Error requeuing NotifID %s: %v", slack, qnm.NotificationID, errRequeue)
				s.UpdateNotificationStatus(ctx, qnm.NotificationID, "failed_requeue", currentAttempts, errRequeue.Error())
				slack.remove(q, queuedMsgInterface)
			}
		} else { // Not done, not retry (e.g. rejected by Slack 4xx)
			log.Printf("%s: Message NotifID %s rejected by service and not retryable.", slack, qnm.NotificationID)
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "rejected", currentAttempts, "Rejected by Slack API")
			slack.remove(q, queuedMsgInterface)
		}
	}
	return nil
}

// Serve starts the Slack service worker pool.
// It now accepts a store.Store instance.
func (slack *Slack) Serve(ctx context.Context, q queue.Queue, s store.Store, fc services.FeedbackCollector) (err error) {
	numWorkers := 4 // Configurable number of concurrent workers
	for i := 0; i < numWorkers; i++ {
		slack.wg.Add(1)
		go slack.ServeClient(ctx, q, s)
	}
	log.Println(slack, "Worker started with", numWorkers, "goroutines")
	slack.wg.Wait()
	log.Println(slack, "Worker Finished")
	return
}
