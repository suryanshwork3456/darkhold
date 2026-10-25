package items

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"darkhold/internal/httpx"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// GET /items?category=pass
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category != "" && category != "pass" && category != "merch" {
		httpx.WriteError(w, http.StatusBadRequest, "category must be 'pass' or 'merch'")
		return
	}

	list, err := h.store.ListItems(r.Context(), category)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load items")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// GET /items/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	it, err := h.store.GetItem(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "item not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load item")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, it)
}

// Routes registers public routes (no login needed).
func (h *Handler) Routes(r chi.Router) {
	r.Get("/items", h.List)
	r.Get("/items/{id}", h.Get)
}