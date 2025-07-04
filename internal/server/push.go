package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	service := strings.TrimPrefix(r.URL.Path, "/api/push/")
	wrk, ok := s.workers[service]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
		return
	}

	"context"
	"encoding/json"

	"github.com/k2glyph/notification-service/internal/queue"
)

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	serviceID := strings.TrimPrefix(r.URL.Path, "/api/push/") // Renamed 'service' to 'serviceID' for clarity
	wrk, ok := s.workers[serviceID]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", 405)
		return
	}

	requestBodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Attempt to extract recipient information from the payload.
	// This is a placeholder: actual extraction logic might be more complex
	// or rely on specific fields in the JSON payload (e.g., "to", "channel").
	var payloadMap map[string]interface{}
	var recipientInfo string
	if json.Unmarshal(requestBodyBytes, &payloadMap) == nil {
		if recipient, found := payloadMap["to"]; found { // Common for email
			if rStr, ok := recipient.(string); ok {
				recipientInfo = rStr
			}
		} else if channel, found := payloadMap["channel"]; found { // Common for Slack
			if cStr, ok := channel.(string); ok {
				recipientInfo = cStr
			}
		}
	}
	if recipientInfo == "" {
		log.Printf("Recipient info not found or not a string in payload for service %s. Proceeding without it.", serviceID)
		// Depending on requirements, this could be an error.
	}


	// Record the notification attempt in the database
	notificationID, err := s.store.RecordNotification(context.Background(), serviceID, requestBodyBytes, recipientInfo)
	if err != nil {
		log.Printf("Error recording notification for service %s: %v", serviceID, err)
		http.Error(w, "Failed to record notification", http.StatusInternalServerError)
		return
	}

	// Prepare the message for the queue
	queuedMsg := queue.QueuedNotificationMessage{
		NotificationID: notificationID,
		ServiceID:      serviceID,
		RecipientInfo:  recipientInfo,
		Payload:        requestBodyBytes, // This is the original payload for the target service
	}

	marshaledQueuedMsg, err := json.Marshal(queuedMsg)
	if err != nil {
		log.Printf("Error marshaling queue message for notification ID %s: %v", notificationID, err)
		// Potentially update DB status to "failed_to_queue" here
		s.store.UpdateNotificationStatus(context.Background(), notificationID, "failed_to_queue", 0, err.Error())
		http.Error(w, "Failed to prepare message for queue", http.StatusInternalServerError)
		return
	}

	err = wrk.push(marshaledQueuedMsg)
	if err != nil {
		log.Printf("Error pushing message to queue for notification ID %s: %v", notificationID, err)
		// Potentially update DB status to "failed_to_queue" here
		s.store.UpdateNotificationStatus(context.Background(), notificationID, "failed_to_queue", 0, err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "OK")
}
