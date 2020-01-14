package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k2glyph/notification-service/internal/services/email"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/queue/memory"
	"github.com/k2glyph/notification-service/internal/queue/redis"
	"github.com/k2glyph/notification-service/internal/server"
	"github.com/k2glyph/notification-service/internal/services/slack"
)

var apiAddr = flag.String("api-addr", ":8080", "API address to listen to")
var slackWebhookURL = os.Getenv("slackWebhookURL")
var redisURL = os.Getenv("redisURL")
var smtpHost = os.Getenv("smtpHost")
var smtpPort = os.Getenv("smtpPort")
var smtpUsername = os.Getenv("smtpUsername")
var smtpPassword = os.Getenv("smtpPassword")
var smtpFrom = os.Getenv("smtpFrom")

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	var qf queue.QueueFactory
	if redisURL != "" {
		log.Println("Using Redis queue at", redisURL)
		qf = redis.NewQueueFactory(redisURL)
	} else {
		log.Println("Using non-persistent in-memory queue")
		qf = memory.MemoryQueueFactory{}

	}
	s := server.NewServer(*apiAddr, qf)
	if slackWebhookURL != "" {
		slack, err := slack.NewSlack(slackWebhookURL)
		if err != nil {
			log.Fatal("Error setting up slack service:", err)
		}
		s.AddService(slack)
	}
	if smtpHost != "" {
		email, err := email.NewEmail(smtpFrom, smtpUsername, smtpPassword, smtpHost, smtpPort)
		if err != nil {
			log.Fatal("Error setting up email service:", err)
		}
		s.AddService(email)
	}
	go func() {
		err := s.Serve()
		if err != nil {
			log.Fatal("Error serving:", err)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)
	log.Println("Exiting")
}
