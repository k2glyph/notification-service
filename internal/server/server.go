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
	mux.Handle("/", dashboardFS) // Serve frontend from root

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

	// Old API Endpoints for Dashboard (commented out)
	/*
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
	*/

	// Redirects for old paths are no longer strictly necessary if the frontend is served from root
	// and handles its own routing. The old /dashboard/ prefix for static assets is also removed.
	// The handleRootRedirect might be simplified or removed if frontend handles all non-API routes.
	// For now, any path not matched by API handlers will be passed to dashboardFS.
	// If dashboardFS is correctly serving an SPA from web/build, this should work.
	// mux.HandleFunc("/", s.handleRootRedirect) // This is now covered by dashboardFS
	// mux.HandleFunc("/ui/notifications", s.handleRootRedirect) // Also covered

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
