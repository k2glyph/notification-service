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
	dashboardFS := http.FileServer(http.Dir("./web/dashboard/"))
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", dashboardFS))

	// API Endpoints for Dashboard
	mux.HandleFunc("/api/dashboard/global-stats", s.handleGetGlobalStats)
	mux.HandleFunc("/api/dashboard/channel-stats", s.handleGetChannelStats)
	mux.HandleFunc("/api/dashboard/recent-activity", s.handleGetRecentActivity)
	mux.HandleFunc("/api/dashboard/historical-trends/volume", func(w http.ResponseWriter, r *http.Request) {
		s.handleGetHistoricalTrend(w, r, "volume")
	})
	mux.HandleFunc("/api/dashboard/historical-trends/failure-rate", func(w http.ResponseWriter, r *http.Request) {
		s.handleGetHistoricalTrend(w, r, "failureRate")
	})
	mux.HandleFunc("/api/dashboard/historical-trends/queue-size", func(w http.ResponseWriter, r *http.Request) {
		s.handleGetHistoricalTrend(w, r, "queueSize")
	})
	mux.HandleFunc("/api/dashboard/failure-reasons", s.handleGetFailureReasons)

	// Redirect root and old UI path to new dashboard
	// The handleRootRedirect function needs to be part of the Server struct if it's not already.
	// For now, assuming handleRootRedirect will be defined or adapted in ui.go or here.
	// If handleUIRedirect in ui.go is suitable, it can be reused or adapted.
	// Let's define a simple redirect here for clarity and update ui.go separately.
	mux.HandleFunc("/", s.handleRootRedirect)
	mux.HandleFunc("/ui/notifications", s.handleRootRedirect) // Explicitly redirect old path

	// mux.HandleFunc("/ui/notifications", s.handleUINotifications) // This line will be removed or handled by the new redirect.
	// The old s.handleUIRedirect might need to be removed if it conflicts or its logic is fully replaced.

	return s
}

// handleRootRedirect redirects to the new dashboard.
// This function can be part of Server struct methods.
func (s *Server) handleRootRedirect(w http.ResponseWriter, r *http.Request) {
	// Check if it's an API call or a specific known path that shouldn't be redirected.
	// For this simple case, any unhandled path "/" or "/ui/notifications" goes to dashboard.
	if r.URL.Path == "/" || r.URL.Path == "/ui/notifications" {
		http.Redirect(w, r, "/dashboard/", http.StatusFound)
		return
	}
	// If it's not the root or old UI path, and not handled by other mux rules, it's a 404.
	// However, http.ServeMux handles this by default if no other pattern matches.
	// If other specific non-dashboard non-API paths existed, they'd need their own handlers.
	// For now, if it's not "/" or "/ui/notifications", it might be a 404 unless another handler matches.
	// To ensure only specific paths are redirected and others 404 correctly,
	// this handler should ideally only be for "/" and "/ui/notifications".
	// Any other unhandled path will naturally 404 with ServeMux.
	// The current setup is fine as long as no other top-level paths are expected to be handled by a generic catch-all.
	http.NotFound(w, r) // Default for paths not matching / or /ui/notifications
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
