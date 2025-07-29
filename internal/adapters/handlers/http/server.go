package http

import (
	"log/slog"
	"marketflow/internal/adapters/handlers/http/handler"
	"net/http"
)

func NewServer(
	logger *slog.Logger,
	marketHandler *handler.MarketHandler,
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, marketHandler)

	var handler http.Handler = mux

	return handler
}
