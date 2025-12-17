package httpx

import (
	"net/http"
)

// Handler is a function that handles an HTTP request.
type Handler func(w http.ResponseWriter, r *http.Request)

// NewHandler creates a new HTTP handler.
func NewHandler(handler Handler) http.Handler {
	return http.HandlerFunc(handler)
}
