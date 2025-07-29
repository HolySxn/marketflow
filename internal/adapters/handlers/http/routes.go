package http

import (
	"marketflow/internal/adapters/handlers/http/handler"
	"net/http"
)

func addRoutes(mux *http.ServeMux, marketHandler *handler.MarketHandler) {
	mux.HandleFunc("GET /prices/latest/{symbol}", marketHandler.GetLatestPrice)
	mux.HandleFunc("GET /prices/latest/{symbol}/{exchange}}", marketHandler.GetLatestPrice)
}
