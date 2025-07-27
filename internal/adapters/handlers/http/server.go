package http

import (
	"log/slog"
	"net/http"
)

func NewServer(
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux)

	var handler http.Handler = mux

	return handler
}
