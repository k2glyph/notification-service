// package server

// import (
// 	"fmt"
// 	"net/http"
// 	"strconv"

// 	"github.com/gorilla/mux"
// )

// // func handlePush(w http.ResponseWriter, r *http.Request) {
// // 	w.Header().Set("Content-Type", "application/json")
// // 	pathParams := mux.Vars(r)
// // 	userID := -1
// // 	var err error
// // 	if val, ok := pathParams["userID"]; ok {
// // 		userID, err = strconv.Atoi(val)
// // 		if err != nil {
// // 			w.WriteHeader(http.StatusInternalServerError)
// // 			w.Write([]byte(`{"message":"need a number"}`))
// // 			return
// // 		}
// // 	}
// // 	commentID := -1
// // 	if val, ok := pathParams["commentID"]; ok {
// // 		commentID, err = strconv.Atoi(val)
// // 		if err != nil {
// // 			w.WriteHeader(http.StatusInternalServerError)
// // 			w.Write([]byte(`{"message":"need a number"}`))
// // 			return
// // 		}
// // 	}
// // 	query := r.URL.Query()
// // 	location := query.Get("location")
// // 	w.Write([]byte(fmt.Sprintf(`{"userID": %d, "commentID": %d, "location": "%s" }`, userID, commentID, location)))
// // }

// func get(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte(`{ "message": "GET method called" }`))
// }
// func post(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusCreated)
// 	w.Write([]byte(`{ "message": "POST method called" }`))
// }
// func put(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusAccepted)
// 	w.Write([]byte(`{ "message": "PUT method called" }`))
// }
// func delete(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte(`{ "message": "DELETE method called" }`))
// }
// func notfound(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusNotFound)
// 	w.Write([]byte(`{ "message": "Not Found" }`))
// }
