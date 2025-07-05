package handlers

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, data any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, message string, status int) error {
	type ErrorResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	errResponse := ErrorResponse{Success: false, Message: message}

	return writeJSON(w, errResponse, status)
}
