package http_api

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Message *string `json:"message,omitempty"`
}

func HandleError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Message: &message})
}
