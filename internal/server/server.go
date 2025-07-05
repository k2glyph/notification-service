package server

import (
	"context"
	"log"
	"net/http"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/services"
	"github.com/k2glyph/notification-service/internal/store"
)

// Server ..
type Server struct {
	server       *http.Server
	shuttingDown bool
	queueFactory queue.QueueFactory
	store        store.Store
	workers      map[string]*worker
}

// Serve ...
func (s *Server) Serve() (err error) {
	log.Println("Notification Service Started at port", s.server.Addr)
	err = s.server.ListenAndServe()
	if s.shuttingDown {
		err = nil
	}
	return
}

// NewServer ...
func NewServer(addr string, qf queue.QueueFactory, st store.Store) (s *Server) {
	mux := http.NewServeMux()
	h := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	s = &Server{
		server:       h,
		queueFactory: qf,
		store:        st,
		workers:      make(map[string]*worker),
	}

	// API for sending notifications
	mux.HandleFunc("/api/push/", s.handlePush)

	// Dashboard UI static files
	// Ensure the path is relative to the execution directory or use an absolute path.
	// Assuming execution from project root.
	// TODO: The path "./web/dashboard/" seems incorrect based on the project structure.
	// It should likely be "./web/build/" or "./web/public/" if the frontend is served directly,
	// or this entire block might be handled differently if a reverse proxy is used in docker-compose.
	// For now, leaving as is, but this will need verification for actual deployment.
	dashboardFS := http.FileServer(http.Dir("./web/build/")) // Assuming frontend build output is in web/build
	mux.Handle("/", dashboardFS)                             // Serve frontend from root

	// New API Endpoints for Dashboard (matching frontend client)
	mux.HandleFunc("/api/stats/summary", s.apiGetStatsSummary)
	mux.HandleFunc("/api/stats/channels", s.apiGetStatsChannels)
	mux.HandleFunc("/api/notifications", s.apiGetNotifications)
	mux.HandleFunc("/api/notifications/failed", s.apiGetFailedNotifications)
	mux.HandleFunc("/api/stats/timeseries", s.apiGetStatsTimeSeries)
	// Note: The POST /api/notifications/:id/retry path needs careful handling with ServeMux
	// as ServeMux doesn't directly support path parameters in the middle like /api/notifications/ID/retry.
	// A common pattern is to handle /api/notifications/ and then parse the ID and "retry" suffix internally,
	// or use a router that supports path parameters.
	// For now, we'll make a specific path and extract from there, or use a prefix match.
	// Let's use a prefix match for now and check the method and path suffix in the handler.
	mux.HandleFunc("/api/notifications/", s.handleNotificationsActions) // Will differentiate GET for list/ID and POST for retry

	mux.HandleFunc("/api/stats/stream", s.apiGetStatsStream) // SSE endpoint

	return s
}

// Shutdown ...
func (s *Server) Shutdown(ctx context.Context) (err error) {
	s.shuttingDown = true
	s.server.Shutdown(ctx)
	if err = s.server.Shutdown(ctx); err != nil {
		log.Printf("Error Shutting down notification service %v\n", err)
		return
	}
	log.Println("Notification Service Stopped")
	return
}

// AddService ...
func (s *Server) AddService(pp services.PushService) (err error) {
	log.Printf("Initializing %s service", pp)
	q, err := s.queueFactory.NewQueue(pp.ID())
	if err != nil {
		return
	}
	// Pass the store to the worker, so it can pass it to the service or use it directly
	w, err := newWorker(pp, q, s.store)
	if err != nil {
		return
	}
	go w.serve(s) // s is a FeedbackCollector, maybe the store should be part of it or passed differently
	s.workers[pp.ID()] = w
	return
}
