package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Server struct {
	server       *http.Server
	shuttingDown bool
}

func (s *Server) Serve() (err error) {
	log.Println("Notification Service Started")
	err = s.server.ListenAndServe()
	if s.shuttingDown {
		err = nil
	}
	return
}
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
func handlePush(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathParams := mux.Vars(r)
	userID := -1
	var err error
	if val, ok := pathParams["userID"]; ok {
		userID, err = strconv.Atoi(val)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"need a number"}`))
			return
		}
	}
	commentID := -1
	if val, ok := pathParams["commentID"]; ok {
		commentID, err = strconv.Atoi(val)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"need a number"}`))
			return
		}
	}
	query := r.URL.Query()
	location := query.Get("location")
	w.Write([]byte(fmt.Sprintf(`{"userID": %d, "commentID": %d, "location": "%s" }`, userID, commentID, location)))
}
func NewServer(addr string) (s *Server) {
	mux := http.NewServeMux()
	h := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	s = &Server{
		server: h,
	}
	mux.HandleFunc("/api/push", handlePush)
	return s
}
