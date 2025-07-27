package handlers

import (
	"log/slog"
	"marketflow/internal/core/port"
	"net/http"
)

type MarketHandler struct {
	MarketService port.MarketServicePort
	logger        *slog.Logger
}

func NewMarketHandler(logger *slog.Logger, marketService port.MarketServicePort) *MarketHandler {
	return &MarketHandler{
		MarketService: marketService,
		logger:        logger,
	}
}

func GetLatest(w http.ResponseWriter, r *http.Request) {

}
