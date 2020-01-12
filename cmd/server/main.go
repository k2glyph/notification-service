package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/queue/memory"
	"github.com/k2glyph/notification-service/internal/server"
	"github.com/k2glyph/notification-service/internal/services/slack"
)

var apiAddr = flag.String("api-addr", ":8080", "API address to listen to")

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	var qf queue.QueueFactory
	qf = memory.MemoryQueueFactory{}
	s := server.NewServer(*apiAddr, qf)
	slack, err := slack.NewSlack("")
	if err != nil {
		log.Fatal("Error setting up slack service:", err)
	}
	s.AddService(slack)
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
