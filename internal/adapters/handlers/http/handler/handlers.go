package handler

import (
	"log/slog"
	"marketflow/internal/core/port"
	jsonresponse "marketflow/pkg/JSONResponse"
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

func (h *MarketHandler) GetLatestPrice(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")

	if symbol == "" {
		h.logger.Error("Symbol not provided in request")
		jsonresponse.WriteError(w, jsonresponse.WrapError(
			jsonresponse.ErrInvalidInput,
			"Symbol must be provided",
			http.StatusBadRequest,
		))
		return
	}

	price, err := h.MarketService.GetLatestPrice(r.Context(), "", symbol)
	if err != nil {
		h.logger.Error("Failed to get latest price", slog.Any("error", err))
		jsonresponse.WriteError(w, jsonresponse.WrapError(
			jsonresponse.ErrInternalError,
			"Failed to get latest price",
			http.StatusInternalServerError,
		))
		return
	}

	jsonresponse.WriteResponse(w, http.StatusOK, price)
	h.logger.Info("Successfully retrieved latest price", slog.String("symbol", symbol), slog.Float64("price", price.Price))
}
