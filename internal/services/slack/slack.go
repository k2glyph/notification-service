package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/services"
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
func (slack *Slack) push(msg slackMessage, fc services.FeedbackCollector) (done, retry bool) {
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
	if err := q.Remove(qm); err != nil {
		log.Println(slack, "error removing from the queue:", err)
	}
}

// Serve Client
func (slack *Slack) ServeClient(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	defer func() {
		slack.wg.Done()
	}()
	for ctx.Err() == nil {
		qm, err := q.Get(ctx)
		if err != nil {
			log.Println(slack, "Error reading from queue", err)
		}
		msg := qm.Message()
		var slackMsg slackMessage
		if err := json.Unmarshal(msg, &slackMsg); err != nil {
			log.Println(slack, "Error parsing", err)
		}
		done, _ := slack.push(slackMsg, fc)
		if done {
			slack.remove(q, qm)
		}
	}
	return
}
func (slack *Slack) Serve(ctx context.Context, q queue.Queue, fc services.FeedbackCollector) (err error) {
	for i := 0; i < 4; i++ {
		go slack.ServeClient(ctx, q, fc)
		slack.wg.Add(1)
	}
	log.Println(slack, "Worker started")
	slack.wg.Wait()
	log.Println(slack, "Worker Finished")
	return
}
