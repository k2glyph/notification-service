package slack

import (
	"net/http"
	"sync"
)

type slackMessage struct {
	Text        string `json:"text"`
	Channel     string `json:"channel"`
	Attachments []struct {
		Fallback  string `json:"fallback"`
		Pretext   string `json:"pretext"`
		Title     string `json:"title"`
		TitleLink string `json:"title_link"`
		Text      string `json:"text"`
		Color     string `json:"color"`
	} `json:"attachments"`
}
type Slack struct {
	transport  *http.Transport
	wg         sync.WaitGroup
	webhookUrl string
}
