package wallet

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"darkhold/internal/auth"
	"darkhold/internal/httpx"
)

// Demo deposit cap: 1,00,000 rupees in paise.
const maxDeposit int64 = 10_000_000

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// userID reads the logged-in user from the context (set by auth middleware).
func userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "not logged in")
	}
	return id, ok
}

// GET /me/wallet
func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(w, r)
	if !ok {
		return
	}
	wl, err := h.store.GetWallet(r.Context(), uid)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "wallet not found")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load wallet")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, wl)
}

// GET /me/holdings
func (h *Handler) GetHoldings(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(w, r)
	if !ok {
		return
	}
	list, err := h.store.GetHoldings(r.Context(), uid)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load holdings")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// GET /me/ledger?limit=50
func (h *Handler) GetLedger(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(w, r)
	if !ok {
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = min(n, 200)
	}

	list, err := h.store.GetLedger(r.Context(), uid, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load ledger")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// POST /me/deposit   body: {"amount": 500000}  (paise)
func (h *Handler) Deposit(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(w, r)
	if !ok {
		return
	}

	var body struct {
		Amount int64 `json:"amount"`
	}
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Amount <= 0 || body.Amount > maxDeposit {
		httpx.WriteError(w, http.StatusBadRequest, "amount out of range")
		return
	}

	if err := h.store.Deposit(r.Context(), uid, body.Amount); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "deposit failed")
		return
	}

	wl, err := h.store.GetWallet(r.Context(), uid)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load wallet")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, wl)
}

// Routes must be mounted behind the auth middleware.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/me/wallet", h.GetWallet)
	r.Get("/me/holdings", h.GetHoldings)
	r.Get("/me/ledger", h.GetLedger)
	r.Post("/me/deposit", h.Deposit)
}