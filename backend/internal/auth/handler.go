package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"darkhold/internal/httpx"
)

type Handler struct {
	store  *Store
	secret string
	ttl    time.Duration
}

func NewHandler(store *Store, secret string, ttl time.Duration) *Handler {
	return &Handler{store: store, secret: secret, ttl: ttl}
}

type authResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// POST /auth/signup  {"email","password","name"}
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	name := strings.TrimSpace(body.Name)
	switch {
	case !strings.Contains(email, "@"):
		httpx.WriteError(w, http.StatusBadRequest, "valid email required")
		return
	case name == "":
		httpx.WriteError(w, http.StatusBadRequest, "name required")
		return
	case len(body.Password) < 8 || len(body.Password) > 72: // bcrypt limit is 72 bytes
		httpx.WriteError(w, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}

	hash, err := HashPassword(body.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not create account")
		return
	}

	user, err := h.store.CreateUser(r.Context(), email, hash, name)
	if errors.Is(err, ErrEmailTaken) {
		httpx.WriteError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not create account")
		return
	}

	h.respondWithToken(w, http.StatusCreated, user)
}

// POST /auth/login  {"email","password"}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.ReadJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	user, err := h.store.GetUserByEmail(r.Context(), email)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !CheckPassword(user.PasswordHash, body.Password)) {
		// same message for both cases, so attackers can't find valid emails
		httpx.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "login failed")
		return
	}

	h.respondWithToken(w, http.StatusOK, user)
}

// GET /auth/me  (protected)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "not logged in")
		return
	}
	user, err := h.store.GetUserByID(r.Context(), uid)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusUnauthorized, "user no longer exists")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not load user")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) respondWithToken(w http.ResponseWriter, status int, u User) {
	token, err := NewToken(u.ID, h.secret, h.ttl)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	httpx.WriteJSON(w, status, authResponse{Token: token, User: u})
}

// Routes registers the public auth routes.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/auth/signup", h.Signup)
	r.Post("/auth/login", h.Login)
}