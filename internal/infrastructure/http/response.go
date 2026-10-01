package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

type successResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type errorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type errorInfo struct {
	Code      string        `json:"code"`
	Status    int           `json:"status"`
	Path      string        `json:"path"`
	Timestamp string        `json:"timestamp"`
	Details   []errorDetail `json:"details,omitempty"`
}

type errorResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    any       `json:"data"`
	Error   errorInfo `json:"error"`
}

const (
	errCodeProfileNotFound = "PROFILE_NOT_FOUND"
	errCodeValidation      = "VALIDATION_ERROR"
	errCodeInvalidJSON     = "INVALID_JSON"
	errCodeInternal        = "INTERNAL_ERROR"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeSuccess(w http.ResponseWriter, status int, message string, data any) {
	writeJSON(w, status, successResponse{Success: true, Message: message, Data: data})
}

func writeError(w http.ResponseWriter, status int, message, code, path string, details ...errorDetail) {
	writeJSON(w, status, errorResponse{
		Success: false,
		Message: message,
		Data:    nil,
		Error: errorInfo{
			Code:      code,
			Status:    status,
			Path:      path,
			Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			Details:   details,
		},
	})
}
