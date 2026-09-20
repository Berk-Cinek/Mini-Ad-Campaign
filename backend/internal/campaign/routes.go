package campaign

import "net/http"

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /campaigns", h.Create)
	mux.HandleFunc("GET /campaigns", h.List)
	mux.HandleFunc("GET /campaigns/{id}", h.Get)
	mux.HandleFunc("PATCH /campaigns/{id}", h.Update)
	mux.HandleFunc("DELETE /campaigns/{id}", h.Delete)
	mux.HandleFunc("POST /campaigns/{id}/pause", h.Pause)
	mux.HandleFunc("POST /campaigns/{id}/resume", h.Resume)
	mux.HandleFunc("POST /campaigns/{id}/end", h.End)
	mux.HandleFunc("POST /impression/{id}", h.Impression)
	mux.HandleFunc("GET /stats/{id}", h.Stats)
}
