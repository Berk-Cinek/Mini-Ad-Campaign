package httpx

import (
	"errors"
	"log"
	"net/http"
)

type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func Invalid(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: msg}
}

func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: msg}
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		WriteJSON(w, appErr.Status, map[string]string{"error": appErr.Message})
		return
	}
	log.Printf("unexpected error: %v", err)
	WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
