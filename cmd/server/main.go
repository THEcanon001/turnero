package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/THEcanon001/turnero/internal/auth"
	"github.com/THEcanon001/turnero/internal/platform/config"
	"github.com/THEcanon001/turnero/internal/platform/database"
	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/provider"
	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Database
	pool, err := database.New(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	logger.Info("database connected")

	// Run migrations
	if err := database.Migrate(cfg.Database.DSN()); err != nil {
		return err
	}
	logger.Info("migrations applied")

	// Dependencies
	jwtCfg := tjwt.Config{
		Secret:             cfg.JWT.Secret,
		AccessTokenExpiry:  cfg.JWT.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.JWT.RefreshTokenExpiry,
		Issuer:             cfg.JWT.Issuer,
	}
	jwtManager := tjwt.NewManager(jwtCfg)

	providerRepo := provider.NewRepository(pool)
	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(providerRepo, authRepo, jwtManager, jwtCfg)
	authHandler := auth.NewHandler(authService)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.CORS([]string{"*"})) // Restrict in production

	// Health endpoint
	r.Get("/health", healthHandler(pool))

	// Auth routes (public)
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Authenticated routes (example placeholder)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))
		// Provider routes will be added in Phase 2+
	})

	// Server
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", slog.String("addr", cfg.Server.Addr()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	logger.Info("shutting down server")
	return srv.Shutdown(shutdownCtx)
}

type dbPinger interface {
	Ping(ctx context.Context) error
}

func healthHandler(db dbPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		dbStatus := "ok"

		if err := db.Ping(r.Context()); err != nil {
			status = "degraded"
			dbStatus = "error"
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":   status,
			"database": dbStatus,
		})
	}
}
