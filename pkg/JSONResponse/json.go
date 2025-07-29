package jsonresponse

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

var (
	ErrNotFound      = errors.New("requested resource not found")
	ErrInvalidInput  = errors.New("invalid input provided")
	ErrInternalError = errors.New("internal server error")
)

type AppError struct {
	Code    int    `json:"-"`     // HTTP Status Code
	Message string `json:"error"` // User-friendly message
	Err     error  `json:"-"`     // Internal error (for logging)
}

func (e *AppError) Error() string {
	return e.Message
}

// Wrap error with additional context.
func WrapError(err error, message string, code int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// JSONResponse sends a JSON response with the given status code and data.
// Optional headers can be provided to set additional response headers.
func WriteResponse(w http.ResponseWriter, statusCode int, data interface{}, headers ...map[string]string) {
	w.Header().Set("Content-Type", "application/json")

	// Set optional headers if provided
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		http.Error(w, `{"error": "Failed to encode JSON response"}`, http.StatusInternalServerError)
	}
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *AppError

	if errors.As(err, &appErr) {
		slog.Error("Error handling request", "error", appErr.Err, "message", appErr.Message)
		WriteResponse(w, appErr.Code, map[string]string{"error": appErr.Message})
	} else {
		slog.Error("Unknown error occurred", "error", err)
		WriteResponse(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
	}
}
