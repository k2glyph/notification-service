package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/k2glyph/notification-service/internal/store"
)

// Helper function to parse date range query parameters
func parseDateRangeParams(r *http.Request) (*time.Time, *time.Time, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var fromDate, toDate *time.Time
	var err error

	if fromStr != "" {
		t, errParse := time.Parse("2006-01-02", fromStr)
		if errParse != nil {
			return nil, nil, errParse
		}
		fromDate = &t
	}
	if toStr != "" {
		t, errParse := time.Parse("2006-01-02", toStr)
		if errParse != nil {
			return nil, nil, errParse
		}
		// Adjust to end of day for "to" date
		t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		toDate = &t
	}
	return fromDate, toDate, err
}

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
