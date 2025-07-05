package server

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/k2glyph/notification-service/internal/store" // Uncommented for integration
)

// --- Structs for API Responses (matching frontend expectations) ---

// SummaryData matches GET /api/stats/summary
type SummaryData struct {
	TotalNotifications    int     `json:"totalNotifications"`
	FailedNotifications   int     `json:"failedNotifications"`
	NotificationsInQueue  int     `json:"notificationsInQueue"`
	AverageDeliveryTime float64 `json:"averageDeliveryTime"` // in seconds
	SuccessRate         float64 `json:"successRate"`         // percentage
}

// ChannelStatData matches items in GET /api/stats/channels
type ChannelStatData struct {
	ChannelName         string  `json:"channelName"`
	TotalNotifications  int     `json:"totalNotifications"`
	FailedNotifications int     `json:"failedNotifications"`
	SuccessRate         float64 `json:"successRate"`
}

// NotificationEntry matches items in GET /api/notifications
type NotificationEntry struct {
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"` // ISO 8601 format "2023-10-26T10:30:00Z"
	Channel       string `json:"channel"`
	Status        string `json:"status"` // e.g., "success", "fail", "pending"
	Destination   string `json:"destination"`
	RetryStatus   string `json:"retryStatus"` // e.g., "not_attempted", "attempted", "failed"
	ErrorMessage  string `json:"errorMessage,omitempty"` // Only for failed notifications
}

// PaginatedNotificationsResponse matches GET /api/notifications
type PaginatedNotificationsResponse struct {
	Notifications []NotificationEntry `json:"notifications"`
	TotalItems    int                 `json:"totalItems"`
	TotalPages    int                 `json:"totalPages"`
	CurrentPage   int                 `json:"currentPage"`
}

// TimeSeriesPoint matches points in time series data
type TimeSeriesPoint struct {
	Date  string `json:"date,omitempty"` // YYYY-MM-DD for daily/weekly
	Week  string `json:"week,omitempty"` // YYYY-Www for weekly
	Count int    `json:"count,omitempty"`
	Timestamp string `json:"timestamp,omitempty"` // ISO 8601 for queue size
	Size      int    `json:"size,omitempty"`
	FailedCount int  `json:"failedCount,omitempty"`
}

// TimeSeriesData matches GET /api/stats/timeseries
type TimeSeriesData struct {
	DailySent     []TimeSeriesPoint `json:"dailySent"`
	WeeklySent    []TimeSeriesPoint `json:"weeklySent"`
	FailureTrends []TimeSeriesPoint `json:"failureTrends"` // daily failure counts
	QueueSize     []TimeSeriesPoint `json:"queueSize"`     // timestamped queue size
}

// FailedNotificationDetail matches items in GET /api/notifications/failed
// This is essentially NotificationEntry with a guaranteed ErrorMessage.
// For simplicity, we can reuse NotificationEntry and ensure ErrorMessage is populated.

// Helper function to parse date range query parameters
func parseDateRangeParams(r *http.Request) (*time.Time, *time.Time, error) {
	fromStr := r.URL.Query().Get("fromDate") // Match frontend query param
	toStr := r.URL.Query().Get("toDate")     // Match frontend query param

	var fromDate, toDate *time.Time
	var err error

	if fromStr != "" {
		// Try parsing RFC3339 first, then YYYY-MM-DD
		t, errParse := time.Parse(time.RFC3339, fromStr)
		if errParse != nil {
			t, errParse = time.Parse("2006-01-02", fromStr)
			if errParse != nil {
				return nil, nil, errParse
			}
		}
		fromDate = &t
	}
	if toStr != "" {
		t, errParse := time.Parse(time.RFC3339, toStr)
		if errParse != nil {
			t, errParse = time.Parse("2006-01-02", toStr)
			if errParse != nil {
				return nil, nil, errParse
			}
			// Adjust to end of day for "to" date if only date is provided
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		toDate = &t
	}
	return fromDate, toDate, err
}

// --- API Handlers ---

func (s *Server) apiGetStatsSummary(w http.ResponseWriter, r *http.Request) {
	// Date range parsing is optional for summary, depends on if store method uses it.
	// Assuming GetDashboardGlobalStats can take nil for full range.
	fromDate, toDate, err := parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
		return
	}

	storeStats, err := s.store.GetDashboardGlobalStats(r.Context(), fromDate, toDate)
	if err != nil {
		log.Printf("Error getting dashboard global stats from store: %v", err)
		http.Error(w, "Failed to fetch global stats", http.StatusInternalServerError)
		return
	}

	if storeStats == nil { // Handle case where store returns nil (e.g. no data)
		storeStats = &store.DashboardGlobalStats{} // Default to zero values
	}

	resp := SummaryData{
		TotalNotifications:    int(storeStats.TotalSent),
		FailedNotifications:   int(storeStats.TotalFailed),
		NotificationsInQueue:  int(storeStats.NotificationsInQueue),
		AverageDeliveryTime: storeStats.AvgDeliveryTimeSec,
		SuccessRate:         storeStats.SuccessRate,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) apiGetStatsChannels(w http.ResponseWriter, r *http.Request) {
	fromDate, toDate, err := parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
		return
	}

	storeChannelStats, err := s.store.GetDashboardChannelStats(r.Context(), fromDate, toDate)
	if err != nil {
		log.Printf("Error getting dashboard channel stats from store: %v", err)
		http.Error(w, "Failed to fetch channel stats", http.StatusInternalServerError)
		return
	}

	resp := make([]ChannelStatData, len(storeChannelStats))
	for i, scs := range storeChannelStats {
		resp[i] = ChannelStatData{
			ChannelName:         scs.Channel,
			TotalNotifications:  int(scs.Total),
			FailedNotifications: int(scs.Failed),
			SuccessRate:         scs.SuccessRate,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func mapStoreNotificationToEntry(n store.Notification) NotificationEntry {
	dest := ""
	if n.RecipientInfo != nil {
		dest = *n.RecipientInfo
	}

	errMsg := ""
	if n.Status == "fail" && n.ErrorMessage != nil {
		errMsg = *n.ErrorMessage
	}

	retryStatus := "not_attempted"
	if n.Attempts > 0 {
		if n.Status == "fail" {
			retryStatus = "failed" // Could also be 'pending_retry' if that's a status we use
		} else if n.Status == "success" {
			retryStatus = "succeeded" // Or could still be 'not_attempted' if it succeeded on first try
		} else {
			retryStatus = "attempted"
		}
	}
	if n.Status == "pending_retry" { // Assuming this status is set by retry logic
		retryStatus = "pending"
	}


	return NotificationEntry{
		ID:            n.ID,
		Timestamp:     n.CreatedAt.Format(time.RFC3339),
		Channel:       n.ServiceID, // Assuming ServiceID is the channel identifier
		Status:        n.Status,
		Destination:   dest,
		RetryStatus:   retryStatus,
		ErrorMessage:  errMsg,
	}
}

func (s *Server) apiGetNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	params := store.DashboardActivityParams{
		Limit:         limit,
		Offset:        (page - 1) * limit,
		StatusFilter:  q.Get("status"),
		ChannelFilter: q.Get("channel"),
		SearchTerm:    q.Get("search"),
	}

	var err error
	params.StartDate, params.EndDate, err = parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format for 'fromDate' or 'toDate'. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
		return
	}

	storeNotifications, totalItems, err := s.store.ListDashboardActivity(r.Context(), params)
	if err != nil {
		log.Printf("Error listing dashboard activity from store: %v", err)
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	notifications := make([]NotificationEntry, len(storeNotifications))
	for i, n := range storeNotifications {
		notifications[i] = mapStoreNotificationToEntry(n)
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(limit)))
	}

	resp := PaginatedNotificationsResponse{
		Notifications: notifications,
		TotalItems:    totalItems,
		TotalPages:    totalPages,
		CurrentPage:   page,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) apiGetStatsTimeSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	// Define default date ranges for trends
	dailyRangeStart := now.AddDate(0, 0, -29) // Last 30 days
	weeklyRangeStart := now.AddDate(0, 0, - (12*7 -1)) // Last 12 weeks (approx 83 days)
	failureTrendRangeStart := dailyRangeStart // Same as daily for failures
	queueSizeRangeStart := now.Add(-24 * time.Hour) // Last 24 hours for queue size

	// Allow overriding with query parameters if needed (not implemented here for brevity)
	// fromDate, toDate, _ := parseDateRangeParams(r)
	// if fromDate != nil { dailyRangeStart = *fromDate /* adapt for others */ }
	// if toDate != nil { now = *toDate /* adapt for others */ }


	resp := TimeSeriesData{
		DailySent:     []TimeSeriesPoint{},
		WeeklySent:    []TimeSeriesPoint{},
		FailureTrends: []TimeSeriesPoint{},
		QueueSize:     []TimeSeriesPoint{},
	}
	var err error

	// Daily Sent
	dailyStorePoints, err := s.store.GetDashboardVolumeTrend(ctx, "daily", dailyRangeStart, now)
	if err != nil {
		log.Printf("Error getting daily volume trend: %v", err)
		// Decide if to return partial data or full error. For now, continue and return what we have.
	} else {
		for _, p := range dailyStorePoints {
			resp.DailySent = append(resp.DailySent, TimeSeriesPoint{Date: p.Date, Count: int(p.Volume)})
		}
	}

	// Weekly Sent - Aggregate from daily data for last 12 weeks
	// Fetch daily data for the last 12 weeks (84 days)
	dailyForWeeklyAgg, err := s.store.GetDashboardVolumeTrend(ctx, "daily", weeklyRangeStart, now)
	if err != nil {
		log.Printf("Error getting daily volume trend for weekly aggregation: %v", err)
	} else {
		weeklyMap := make(map[string]int) // Key: "YYYY-Www"
		for _, p := range dailyForWeeklyAgg {
			t, parseErr := time.Parse("2006-01-02", p.Date)
			if parseErr != nil {
				continue
			}
			year, weekNum := t.ISOWeek()
			weekKey := fmt.Sprintf("%d-W%02d", year, weekNum)
			weeklyMap[weekKey] += int(p.Volume)
		}
		for weekKey, count := range weeklyMap {
			resp.WeeklySent = append(resp.WeeklySent, TimeSeriesPoint{Week: weekKey, Count: count})
		}
		// Sort weekly data for consistency (optional, depends on frontend needs)
		// sort.Slice(resp.WeeklySent, func(i, j int) bool { return resp.WeeklySent[i].Week < resp.WeeklySent[j].Week })
	}


	// Failure Trends (Daily)
	failureStorePoints, err := s.store.GetDashboardFailureRateTrend(ctx, "daily", failureTrendRangeStart, now)
	if err != nil {
		log.Printf("Error getting failure rate trend: %v", err)
	} else {
		for _, p := range failureStorePoints {
			resp.FailureTrends = append(resp.FailureTrends, TimeSeriesPoint{Date: p.Date, FailedCount: int(p.Failures)})
		}
	}

	// Queue Size Trend (e.g., hourly for last 24h)
	// Store's GetDashboardQueueSizeTrend takes "period" which can be "daily" or "hourly"
	// For a more granular view like the mock data (every 5 min), the store method might need adjustment
	// or this needs to be fetched from a different source (e.g. live queue metrics)
	// For now, using "hourly" for the last 24 hours.
	queueStorePoints, err := s.store.GetDashboardQueueSizeTrend(ctx, "hourly", queueSizeRangeStart, now)
	if err != nil {
		log.Printf("Error getting queue size trend: %v", err)
	} else {
		for _, p := range queueStorePoints {
			// p.Date here is likely YYYY-MM-DD HH:00, which is fine for hourly.
			// If more granular (like actual timestamp), adjust TimeSeriesPoint.Timestamp
			resp.QueueSize = append(resp.QueueSize, TimeSeriesPoint{Timestamp: p.Date, Size: int(p.QueueSize)})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) apiGetFailedNotifications(w http.ResponseWriter, r *http.Request) {
	// For simplicity, fetch all failed notifications for now, not paginated.
	// Could add pagination similar to apiGetNotifications if lists get very long.
	// Max limit can be set if performance becomes an issue.
	limit := 0 // 0 might mean unlimited in some store implementations, or use a high number.
	// Let's use a high number for now, e.g. 500, to avoid accidental unlimited queries.
	// The frontend does not paginate this view.
	maxFailedLimit := 500


	params := store.DashboardActivityParams{
		Limit:        maxFailedLimit,
		Offset:       0,
		StatusFilter: "fail", // Hardcoded to 'fail'
	}

	// Optional: Allow date filtering for failed notifications too
	var err error
	params.StartDate, params.EndDate, err = parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format for 'fromDate' or 'toDate'. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
		return
	}


	storeNotifications, _, err := s.store.ListDashboardActivity(r.Context(), params)
	if err != nil {
		log.Printf("Error listing failed dashboard activity from store: %v", err)
		http.Error(w, "Failed to fetch failed notifications", http.StatusInternalServerError)
		return
	}

	failedNotifs := make([]NotificationEntry, len(storeNotifications))
	for i, n := range storeNotifications {
		failedNotifs[i] = mapStoreNotificationToEntry(n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(failedNotifs)
}

func (s *Server) apiPostRetryNotification(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	var notificationID string
	if len(pathParts) >= 4 && parts[0] == "api" && parts[1] == "notifications" && parts[3] == "retry" {
		notificationID = pathParts[2]
	} else { // Fallback for direct call if path structure is slightly different
		// This part might be redundant if handleNotificationsActions normalizes the call
		// but good for robustness if apiPostRetryNotification is called directly with /api/notifications/:id/retry
		if len(pathParts) < 4 {
			http.Error(w, "Invalid notification ID in URL", http.StatusBadRequest)
			return
		}
		notificationID = pathParts[3]
	}

	if notificationID == "" { // Should be caught by Split path logic, but double check
		http.Error(w, "Missing notification ID", http.StatusBadRequest)
		return
	}

	// 1. Fetch the notification by ID from the store.
	notification, err := s.store.GetNotification(r.Context(), notificationID)
	if err != nil {
		log.Printf("Error fetching notification %s for retry: %v", notificationID, err)
		http.Error(w, "Notification not found or error fetching it.", http.StatusNotFound)
		return
	}
	if notification == nil {
		http.Error(w, "Notification not found.", http.StatusNotFound)
		return
	}

	// 2. Check if it's actually in a retryable state (e.g., "fail").
	if notification.Status != "fail" {
		log.Printf("Notification %s is not in 'fail' state (current: %s). Cannot retry.", notificationID, notification.Status)
		http.Error(w, fmt.Sprintf("Notification is not in 'fail' state (current: %s). Cannot retry.", notification.Status), http.StatusBadRequest)
		return
	}

	// 3. Update its status (e.g., to "pending_retry" or "queued").
	// For now, setting to "pending_retry". The worker would then pick this up.
	// The number of attempts is not incremented here, the worker should do that upon actual attempt.
	// Error message is cleared as it's a new attempt.
	err = s.store.UpdateNotificationStatus(r.Context(), notificationID, "pending_retry", notification.Attempts, "")
	if err != nil {
		log.Printf("Error updating notification %s status to 'pending_retry': %v", notificationID, err)
		http.Error(w, "Failed to update notification status for retry.", http.StatusInternalServerError)
		return
	}

	log.Printf("Notification ID: %s marked for retry. Previous attempts: %d", notificationID, notification.Attempts)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"message": "Retry request accepted for notification " + notificationID})
}

// handleNotificationsActions is a multiplexer for /api/notifications/* routes
// It distinguishes between listing notifications, getting a specific one (not yet implemented), and retrying.
func (s *Server) handleNotificationsActions(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Example: /api/notifications/msg_123/retry
	parts := strings.Split(strings.Trim(path, "/"), "/") // {"api", "notifications", "msg_123", "retry"}

	if len(parts) == 2 && parts[0] == "api" && parts[1] == "notifications" && r.Method == http.MethodGet {
		// This is GET /api/notifications (list with query params)
		s.apiGetNotifications(w, r)
		return
	}

	// Check for /api/notifications/:id/retry
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "notifications" && parts[3] == "retry" {
		if r.Method == http.MethodPost {
			// Simulate extracting ID and calling the retry handler
			// The actual ID extraction for apiPostRetryNotification expects it to be the 3rd part of a 4-part path
			// when split from the root, which is parts[2] here.
			// The apiPostRetryNotification currently has its own path splitting logic.
			// For consistency, let's adjust apiPostRetryNotification or pass the ID directly.
			// For now, let's assume apiPostRetryNotification can handle the request as is if path is correct.
			// Or, more simply, call it directly with the extracted ID.

			// Re-routing r.URL.Path to be what apiPostRetryNotification expects if it does its own parsing:
			// r.URL.Path = fmt.Sprintf("/api/notifications/%s/retry", parts[2]) // This might be problematic
			// Instead, let's modify apiPostRetryNotification to accept ID as a parameter or simplify its parsing.
			// For now, let's call it, it will re-parse. This isn't ideal.
			// A better way:
			// notificationID := parts[2]
			// s.doRetryNotification(w, r, notificationID) // and apiPostRetryNotification becomes doRetryNotification
			s.apiPostRetryNotification(w, r) // Let apiPostRetryNotification re-parse for now
			return
		}
		http.Error(w, "Method not allowed for retry action.", http.StatusMethodNotAllowed)
		return
	}

	// Placeholder for GET /api/notifications/:id (get single notification) - Not explicitly requested but common
	// if len(parts) == 3 && parts[0] == "api" && parts[1] == "notifications" && r.Method == http.MethodGet {
	// 	notificationID := parts[2]
	// 	// Call a handler like s.apiGetNotificationByID(w, r, notificationID)
	// 	http.Error(w, "Get specific notification not yet implemented. ID: "+notificationID, http.StatusNotImplemented)
	// 	return
	// }

	// If it's GET /api/notifications with query parameters, it should be handled by apiGetNotifications
	// The check above `len(parts) == 2` handles the base path.
	// If there are more parts and it's not retry, it's an unknown path for now.
	http.NotFound(w, r)
}

// --- SSE Endpoint Handler ---

func (s *Server) apiGetStatsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Optional: For local dev if frontend is on different port

	// Notify client of connection
	fmt.Fprintf(w, "event: connection_established\ndata: %s\n\n", `{"message": "SSE connection established"}`)
	flusher.Flush()

	ticker := time.NewTicker(5 * time.Second) // Send updates every 5 seconds
	defer ticker.Stop()

	log.Println("SSE client connected")

	for {
		select {
		case <-r.Context().Done():
			log.Println("SSE client disconnected")
			return
		case t := <-ticker.C:
			// Send summary_update
			summaryData := getMockSummaryData()
			summaryJSON, _ := json.Marshal(summaryData)
			fmt.Fprintf(w, "event: summary_update\ndata: %s\n\n", string(summaryJSON))

			// Send channel_stats_update
			channelData := getMockChannelData()
			channelJSON, _ := json.Marshal(channelData)
			fmt.Fprintf(w, "event: channel_stats_update\ndata: %s\n\n", string(channelJSON))

			// Send queue_size_update
			// For queue_size_update, the spec asks for a single TimePoint object for queue size.
			// Example: event: queue_size_update\ndata: {"timestamp":"2023-10-27T10:20:00Z","size":22}\n\n
			queuePoint := TimeSeriesPoint{
				Timestamp: t.Format(time.RFC3339),
				Size:      20 + (int(t.Unix()) % 15), // Mock size fluctuation
			}
			queueJSON, _ := json.Marshal(queuePoint)
			fmt.Fprintf(w, "event: queue_size_update\ndata: %s\n\n", string(queueJSON))

			flusher.Flush()
			log.Println("SSE data sent")
		}
	}
}


/*
// Old handlers - can be removed or adapted further if any specific logic needs to be preserved.
// For now, they are commented out as new handlers above are intended to replace them.

func (s *Server) handleGetGlobalStats(w http.ResponseWriter, r *http.Request) {
	fromDate, toDate, err := parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD.", http.StatusBadRequest)
		return
	}

	stats, err := s.store.GetDashboardGlobalStats(r.Context(), fromDate, toDate)
	if err != nil {
		log.Printf("Error getting dashboard global stats: %v", err)
		http.Error(w, "Failed to fetch global stats", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleGetChannelStats(w http.ResponseWriter, r *http.Request) {
	fromDate, toDate, err := parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD.", http.StatusBadRequest)
		return
	}
	stats, err := s.store.GetDashboardChannelStats(r.Context(), fromDate, toDate)
	if err != nil {
		log.Printf("Error getting dashboard channel stats: %v", err)
		http.Error(w, "Failed to fetch channel stats", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleGetRecentActivity(w http.ResponseWriter, r *http.Request) {
	params := store.DashboardActivityParams{}
	q := r.URL.Query()

	limitStr := q.Get("limit")
	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		params.Limit = val
	} else {
		params.Limit = 50 // Default limit
	}
	if params.Limit > 200 { params.Limit = 200 } // Max limit

	offsetStr := q.Get("offset")
	if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
		params.Offset = val
	} else {
		params.Offset = 0 // Default offset
	}

	fromDate, toDate, err := parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format for 'from' or 'to'. Use YYYY-MM-DD.", http.StatusBadRequest)
		return
	}
	params.StartDate = fromDate
	params.EndDate = toDate

	params.ChannelFilter = q.Get("channel")
	params.StatusFilter = q.Get("status")
	params.SearchTerm = q.Get("search")

	activity, totalCount, err := s.store.ListDashboardActivity(r.Context(), params)
	if err != nil {
		log.Printf("Error getting dashboard recent activity: %v", err)
		http.Error(w, "Failed to fetch recent activity", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Notifications []store.Notification `json:"notifications"`
		TotalCount    int                `json:"totalCount"`
	}{
		Notifications: activity,
		TotalCount:    totalCount,
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetHistoricalTrend(w http.ResponseWriter, r *http.Request, trendType string) {
	q := r.URL.Query()
	period := q.Get("period") // "daily", "hourly" (weekly can be derived or specific)
	if period == "" {
		period = "daily" // Default period
	}

	// Default to last 7 days if no specific dates are given
	endDate := time.Now()
	startDateStr := q.Get("startDate")
	endDateStr := q.Get("endDate")

	var startDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid startDate format. Use YYYY-MM-DD.", http.StatusBadRequest)
			return
		}
	} else {
		startDate = endDate.AddDate(0,0,-7) // Default to 7 days ago
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid endDate format. Use YYYY-MM-DD.", http.StatusBadRequest)
			return
		}
		endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second) // End of day
	}


	var data []store.DashboardHistoricalPoint
	switch trendType {
	case "volume":
		data, err = s.store.GetDashboardVolumeTrend(r.Context(), period, startDate, endDate)
	case "failureRate":
		data, err = s.store.GetDashboardFailureRateTrend(r.Context(), period, startDate, endDate)
	case "queueSize":
		data, err = s.store.GetDashboardQueueSizeTrend(r.Context(), period, startDate, endDate)
	default:
		http.Error(w, "Invalid trend type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error getting dashboard %s trend: %v", trendType, err)
		http.Error(w, "Failed to fetch trend data", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(data)
}


func (s *Server) handleGetFailureReasons(w http.ResponseWriter, r *http.Request) {
	fromDate, toDate, err := parseDateRangeParams(r)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD.", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // Default limit
	if val, errConv := strconv.Atoi(limitStr); errConv == nil && val > 0 {
		limit = val
	}
	if limit > 50 { limit = 50} // Max limit


	reasons, err := s.store.GetDashboardFailureReasons(r.Context(), fromDate, toDate, limit)
	if err != nil {
		log.Printf("Error getting dashboard failure reasons: %v", err)
		http.Error(w, "Failed to fetch failure reasons", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(reasons)
}
*/
