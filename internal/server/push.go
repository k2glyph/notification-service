package server

import (
	"net/http"
	"strings"
)

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	service := strings.TrimPrefix(r.URL.Path, "/api/push/")
	// s.
	// w.Header().Set("Content-Type", "application/json")
	// pathParams := mux.Vars(r)
	// userID := -1
	// var err error
	// if val, ok := pathParams["userID"]; ok {
	// 	userID, err = strconv.Atoi(val)
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		w.Write([]byte(`{"message":"need a number"}`))
	// 		return
	// 	}
	// }
	// commentID := -1
	// if val, ok := pathParams["commentID"]; ok {
	// 	commentID, err = strconv.Atoi(val)
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		w.Write([]byte(`{"message":"need a number"}`))
	// 		return
	// 	}
	// }
	// query := r.URL.Query()
	// location := query.Get("location")
	// w.Write([]byte(fmt.Sprintf(`{"userID": %d, "commentID": %d, "location": "%s" }`, userID, commentID, location)))
}
