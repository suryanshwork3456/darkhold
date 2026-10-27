package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"darkhold/internal/auth"
	"darkhold/internal/config"
	"darkhold/internal/db"
	"darkhold/internal/httpx"
	"darkhold/internal/items"
	"darkhold/internal/wallet"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	// stores and handlers
	authH := auth.NewHandler(auth.NewStore(database), cfg.JWTSecret, cfg.TokenTTL)
	itemsH := items.NewHandler(items.NewStore(database))
	walletH := wallet.NewHandler(wallet.NewStore(database))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api", func(r chi.Router) {
		// public routes
		authH.Routes(r) // POST /auth/signup, POST /auth/login
		itemsH.Routes(r) // GET /items, GET /items/{id}

		// protected routes (need Authorization: Bearer <token>)
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(cfg.JWTSecret))
			r.Get("/auth/me", authH.Me)
			walletH.Routes(r) // /me/wallet, /me/holdings, /me/ledger, /me/deposit
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// run the server in the background so main can wait for Ctrl+C
	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server stopped")
}