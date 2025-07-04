package email

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/smtp"
	"time"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/services"
	"github.com/k2glyph/notification-service/internal/store" // Added store
)

func NewEmail(from string, username string, password string, host string, port string) (email *Email, err error) {
	email = &Email{
		smtpHost:     host,
		smtpPort:     port,
		smtpUsername: username,
		smtpPassword: password,
		smtpFrom:     from,
		transport: &http.Transport{
			MaxIdleConns:    5,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return
}
func (email *Email) ID() string {
	return "email"
}

func (email *Email) String() string {
	return "EMAIL"
}
func (email *Email) push(msg emailMessage) (done, retry bool) {
	mime := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
	msgBody := "From: " + email.smtpFrom + "\n" + "To: " + msg.To + "\n" + "Subject:" + msg.Subject + " \n\n" + msg.Body
	err := smtp.SendMail(email.smtpHost+":"+email.smtpPort,
		smtp.PlainAuth("", email.smtpUsername, email.smtpPassword, email.smtpHost),
		email.smtpFrom, []string{msg.To}, []byte(mime+"\n"+msgBody))
	if err != nil {
		log.Println(email, "Error not able to send email", err)
		return false, true
	}
	return true, false
}
func (email *Email) remove(q queue.Queue, qm queue.QueuedMessage) {
	if err := q.Remove(qm); err != nil { // This remove might be optional if DB state is the source of truth
		log.Println(email, "error removing from the queue:", err)
	}
}

// ServeClient processes messages from the queue for Email
func (email *Email) ServeClient(ctx context.Context, q queue.Queue, s store.Store) (err error) {
	defer func() {
		email.wg.Done()
	}()

	currentAttempts := 0 // Local attempt counter

	for ctx.Err() == nil {
		queuedMsgInterface, err := q.Get(ctx)
		if err != nil {
			if ctx.Err() != nil { // Context cancelled
				return nil
			}
			log.Println(email, "Error reading from queue", err)
			time.Sleep(1 * time.Second)
			continue
		}

		rawMsgBytes := queuedMsgInterface.Message()
		qnm, err := services.ParseQueuedMessage(rawMsgBytes)
		if err != nil {
			log.Printf("%s: Error parsing QueuedNotificationMessage: %v. Raw: %s. Discarding.", email, err, string(rawMsgBytes))
			email.remove(q, queuedMsgInterface)
			continue
		}

		currentAttempts = 1 // Reset for new message
		s.UpdateNotificationStatus(ctx, qnm.NotificationID, "processing", currentAttempts, "")

		var servicePayload emailMessage
		if err := json.Unmarshal(qnm.Payload, &servicePayload); err != nil {
			log.Printf("%s: Error unmarshaling Email payload for NotifID %s: %v. Payload: %s", email, qnm.NotificationID, err, string(qnm.Payload))
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "failed_parse", currentAttempts, err.Error())
			email.remove(q, queuedMsgInterface)
			continue
		}

		// Ensure 'To' field is populated, either from payload or RecipientInfo
		if servicePayload.To == "" {
			if qnm.RecipientInfo == "" {
				log.Printf("%s: Missing 'To' address in payload and RecipientInfo for NotifID %s.", email, qnm.NotificationID)
				s.UpdateNotificationStatus(ctx, qnm.NotificationID, "failed_validation", currentAttempts, "Missing recipient 'To' address")
				email.remove(q, queuedMsgInterface)
				continue
			}
			servicePayload.To = qnm.RecipientInfo
		}


		done, retry := email.push(servicePayload)

		if done {
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "sent", currentAttempts, "")
			email.remove(q, queuedMsgInterface)
		} else if retry {
			currentAttempts++
			log.Printf("%s: Retrying NotifID %s (local attempts: %d)", email, qnm.NotificationID, currentAttempts)
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "retry_queued", currentAttempts, "Service requested retry (e.g. SMTP error)")

			if errRequeue := q.Requeue(queuedMsgInterface); errRequeue != nil {
				log.Printf("%s: Error requeuing NotifID %s: %v", email, qnm.NotificationID, errRequeue)
				s.UpdateNotificationStatus(ctx, qnm.NotificationID, "failed_requeue", currentAttempts, errRequeue.Error())
				email.remove(q, queuedMsgInterface)
			}
		} else { // Not done, not retry (should not happen with current email.push logic, as it always returns retry=true on error)
			log.Printf("%s: Message NotifID %s failed by service and not retryable (unexpected).", email, qnm.NotificationID)
			s.UpdateNotificationStatus(ctx, qnm.NotificationID, "failed", currentAttempts, "Failed by service (unexpected non-retry)")
			email.remove(q, queuedMsgInterface)
		}
	}
	return nil
}

// Serve starts the Email service worker pool.
// It now accepts a store.Store instance.
func (email *Email) Serve(ctx context.Context, q queue.Queue, s store.Store, fc services.FeedbackCollector) (err error) {
	numWorkers := 4 // Configurable
	for i := 0; i < numWorkers; i++ {
		email.wg.Add(1)
		go email.ServeClient(ctx, q, s)
	}
	log.Println(email, "Worker started with", numWorkers, "goroutines")
	email.wg.Wait()
	log.Println(email, "Worker Finished")
	return
}
