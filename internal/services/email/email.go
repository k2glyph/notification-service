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
	if err := q.Remove(qm); err != nil {
		log.Println(email, "error removing from the queue:", err)
	}
}

// Serve Client
func (email *Email) ServeClient(ctx context.Context, q queue.Queue) (err error) {
	defer func() {
		email.wg.Done()
	}()
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println(email, "Error reading from queue", err)
		}
		msg := qm.Message()
		var emailMsg emailMessage
		if err := json.Unmarshal(msg, &emailMsg); err != nil {
			log.Println(email, "Error parsing", err)
		}
		done, _ := email.push(emailMsg)
		if done {
			email.remove(q, qm)
		}
	}
	return
}
func (email *Email) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	for i := 0; i < 4; i++ {
		go email.ServeClient(ctx, q)
		email.wg.Add(1)
	}
	log.Println(email, "Worker started")
	email.wg.Wait()
	log.Println(email, "Worker Finished")
	return
}
