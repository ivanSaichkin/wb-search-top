package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ivanSaichkin/wb-search-top/internal/domain/ports/usecases"
)

type Handler struct {
	searchUC   usecases.SearchUseCase
	stopListUC usecases.StoplistUseCase
}

func NewHandler(searchUC usecases.SearchUseCase, stopListUC usecases.StoplistUseCase) *Handler {
	return &Handler{
		searchUC:   searchUC,
		stopListUC: stopListUC,
	}
}

// регистрирует маршруты в стандартном мультиплексоре
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/top", h.GetTop)
	mux.HandleFunc("POST /api/v1/stoplist", h.AddStopWord)
	mux.HandleFunc("DELETE /api/v1/stoplist", h.RemoveStopWord)
}

func (h *Handler) GetTop(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	top, err := h.searchUC.GetFilteredTop(r.Context(), limit)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(top)
}

func (h *Handler) AddStopWord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Word string `json:"word"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.stopListUC.AddWord(r.Context(), req.Word); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) RemoveStopWord(w http.ResponseWriter, r *http.Request) {
	word := r.URL.Query().Get("word")
	if word == "" {
		http.Error(w, "Word parameter is required", http.StatusBadRequest)
		return
	}

	if err := h.stopListUC.RemoveWord(r.Context(), word); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
