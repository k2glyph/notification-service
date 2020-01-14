package email

import (
	"net/http"
	"sync"
)

type emailMessage struct {
	To      string `json:"to"`
	Body    string `json:"body"`
	Subject string `json:"subject"`
}
type Email struct {
	transport    *http.Transport
	wg           sync.WaitGroup
	smtpHost     string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	smtpFrom     string
}
