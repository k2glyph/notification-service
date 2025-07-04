package server

import (
	"context"
	"log"
	"net/http"

	"github.com/k2glyph/notification-service/internal/queue"
	"github.com/k2glyph/notification-service/internal/services"
)

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
		store:        st, // Corrected: store assignment only once
		workers:      make(map[string]*worker),
	}
	mux.HandleFunc("/api/push/", s.handlePush)
	mux.HandleFunc("/ui/notifications", s.handleUINotifications) // New UI route
	mux.HandleFunc("/", s.handleUIRedirect)                     // Redirect root to UI

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
