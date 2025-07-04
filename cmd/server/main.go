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
	"github.com/k2glyph/notification-service/internal/store"
)

var apiAddr = flag.String("api-addr", ":8080", "API address to listen to")
var databaseURL = os.Getenv("DATABASE_URL")     // e.g., "postgres://user:password@localhost:5432/notifications_db"
var databaseType = os.Getenv("DATABASE_TYPE") // e.g., "postgres"
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

	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set.")
	}

	if databaseType == "" {
		log.Println("DATABASE_TYPE not set, defaulting to postgres")
		databaseType = "postgres"
	}

	var factory store.StoreFactory
	switch databaseType {
	case "postgres":
		factory = store.NewPostgresStoreFactory()
	case "mysql":
		factory = store.NewMySQLStoreFactory()
	default:
		log.Fatalf("Unsupported DATABASE_TYPE: %s", databaseType)
	}

	dbStore, err := factory.NewStore(databaseURL)
	if err != nil {
		log.Fatalf("Error initializing store via factory (type: %s): %v", databaseType, err)
	}
	defer dbStore.Close()

	log.Println("Successfully initialized store with type:", databaseType)

	s := server.NewServer(*apiAddr, qf, dbStore)
	if slackWebhookURL != "" {
		slackService, slackErr := slack.NewSlack(slackWebhookURL) // Renamed to avoid conflict with err
		if slackErr != nil {
			log.Fatal("Error setting up slack service:", slackErr)
		}
		s.AddService(slackService)
	}
	if smtpHost != "" {
		emailService, emailErr := email.NewEmail(smtpFrom, smtpUsername, smtpPassword, smtpHost, smtpPort) // Renamed to avoid conflict with err
		if emailErr != nil {
			log.Fatal("Error setting up email service:", emailErr)
		}
		s.AddService(emailService)
	}
	go func() {
		serveErr := s.Serve() // Renamed to avoid conflict with err
		if serveErr != nil {
			log.Fatal("Error serving:", serveErr)
		}
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.Shutdown(ctx)
	log.Println("Exiting")
}
