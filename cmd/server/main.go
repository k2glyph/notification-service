package main

import (
	"flag"
	"net/http"
	"gitlab.com/k2glyph/notification-service/internal/server"
)

var apiAddr = flag.String("api-addr", ":8322", "API address to listen to")

func main() {
	s := server.NewServer()
	// r := mux.NewRouter()
	// api := r.PathPrefix("/api/v1").Subrouter()
	// api.HandleFunc("", get).Methods(http.MethodGet)
	// api.HandleFunc("", post).Methods(http.MethodPost)
	// api.HandleFunc("", put).Methods(http.MethodPut)
	// api.HandleFunc("", delete).Methods(http.MethodDelete)

	// api.HandleFunc("/user/{userID}/comment/{commentID}", params).Methods(http.MethodGet)

	// fmt.Println("Welcome to notification service")
	// log.Fatal(http.ListenAndServe(":8080", r))

}
