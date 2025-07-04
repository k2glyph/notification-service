package server

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/k2glyph/notification-service/internal/store" // Import for store.Notification type
)

const uiDefaultLimit = 50
const uiDefaultOffset = 0

// handleUINotifications handles requests for the UI notifications page.
// It fetches notifications from the store and renders them using an HTML template.
func (s *Server) handleUINotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = uiDefaultLimit
	}
	if limit > 200 { // Safety max limit
		limit = 200
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = uiDefaultOffset
	}

	notifications, totalCount, err := s.store.ListNotifications(r.Context(), limit, offset)
	if err != nil {
		log.Printf("Error fetching notifications for UI: %v", err)
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	prevOffset := offset - limit
	// prevOffset will be < 0 if on the first page or if limit makes it so. Template handles this.

	nextOffset := offset + limit
	// nextOffset will be >= totalCount if on the last page. Template handles this.


	funcMap := template.FuncMap{
		"mulf": func(a, b int) int { return a * b },
		"sub":  func(a, b int) int { return a - b },
		"add":  func(a, b int) int { return a + b },
		"payloadSummary": func(payload []byte) string {
			const maxLen = 100
			var jsData map[string]interface{}
			if json.Unmarshal(payload, &jsData) == nil {
				prettyPayload, err := json.MarshalIndent(jsData, "", "  ")
				if err == nil {
					if len(prettyPayload) > maxLen+20 { // +20 for ellipsis and some buffer
						return string(prettyPayload[:maxLen]) + "..."
					}
					return string(prettyPayload)
				}
			}
			if len(payload) > maxLen {
				return string(payload[:maxLen]) + "..."
			}
			return string(payload)
		},
		"formatTimePtr": func(t *time.Time) string {
			if t == nil {
				return "N/A"
			}
			return t.In(time.Local).Format("2006-01-02 15:04:05") // Use local time for display
		},
		"formatTime": func(t time.Time) string {
			return t.In(time.Local).Format("2006-01-02 15:04:05") // Use local time for display
		},
		"ptrToString": func(s *string) string {
			if s == nil {
				return "N/A"
			}
			return *s
		},
		"ptrToStringEllipsis": func(s *string, maxLen int) string {
			if s == nil {
				return "N/A"
			}
			if len(*s) > maxLen {
				return (*s)[:maxLen] + "..."
			}
			return *s
		},
	}

	// Using an absolute path or path relative to a known root is safer.
	// For now, assuming execution from project root. Embedding is better for production.
	tmplPath := filepath.Join("web", "ui", "templates", "notifications.html")

	// The name given to New() must be the base name of the template file if not using named templates within a single file.
	// If notifications.html defines {{define "base"}} then use "base". Here, it's the filename.
	tmpl, err := template.New(filepath.Base(tmplPath)).Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		log.Printf("Error parsing UI template '%s': %v", tmplPath, err)
		http.Error(w, "Internal server error (template parse)", http.StatusInternalServerError)
		return
	}

	currentPage := 0
	totalPages := 0
	if limit > 0 { // Avoid division by zero if limit somehow became 0
		currentPage = (offset / limit) + 1
		if totalCount > 0 {
			totalPages = (totalCount + limit - 1) / limit
		} else {
			totalPages = 1 // Show 1 page even if no results
		}
	} else {
		currentPage = 1
		totalPages = 1
	}


	data := struct {
		Notifications []store.Notification
		TotalCount    int
		Limit         int
		Offset        int
		PrevOffset    int
		NextOffset    int
		CurrentPage   int
		TotalPages    int
	}{
		Notifications: notifications,
		TotalCount:    totalCount,
		Limit:         limit,
		Offset:        offset,
		PrevOffset:    prevOffset,
		NextOffset:    nextOffset,
		CurrentPage:   currentPage,
		TotalPages:    totalPages,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing UI template: %v", err)
		// http.Error already sent or headers written, so just log.
	}
}

// handleUIRedirect redirects the root path ("/") to the notifications UI page.
func (s *Server) handleUIRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/ui/notifications", http.StatusFound)
		return
	}
	// For any other path not explicitly handled, this will result in a 404 if this is the last matching handler.
	http.NotFound(w, r)
}
