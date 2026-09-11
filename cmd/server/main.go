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
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/THEcanon001/turnero/internal/appointment"
	"github.com/THEcanon001/turnero/internal/auth"
	"github.com/THEcanon001/turnero/internal/billing"
	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/notification"
	"github.com/THEcanon001/turnero/internal/platform/config"
	"github.com/THEcanon001/turnero/internal/platform/database"
	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/provider"
	"github.com/THEcanon001/turnero/internal/qr"
	"github.com/THEcanon001/turnero/internal/schedule"
	"github.com/THEcanon001/turnero/internal/search"
	"github.com/THEcanon001/turnero/internal/worker"
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
	employeeRepo := employee.NewRepository(pool)
	scheduleRepo := schedule.NewRepository(pool)
	authRepo := auth.NewRepository(pool)

	appointmentRepo := appointment.NewRepository(pool)
	appointmentService := appointment.NewService(appointmentRepo, scheduleRepo)

	authService := auth.NewService(providerRepo, employeeRepo, authRepo, jwtManager, jwtCfg)
	authHandler := auth.NewHandler(authService, cfg.Google.ClientID)
	providerHandler := provider.NewHandler(providerRepo)
	providerHandler.SetStatsProvider(&statsAdapter{appointmentRepo: appointmentRepo})
	employeeHandler := employee.NewHandler(employeeRepo)
	employeeHandler.SetAppointmentCanceller(appointmentRepo)
	scheduleHandler := schedule.NewHandler(scheduleRepo, employeeRepo)
	scheduleHandler.SetAppointmentCanceller(appointmentRepo)
	appointmentHandler := appointment.NewHandler(appointmentService, appointmentRepo, providerRepo, employeeRepo)
	searchHandler := search.NewHandler(pool)

	qrService := qr.NewService(cfg.QR.BaseURL, cfg.QR.OutputDir)
	qrHandler := qr.NewHandler(qrService, providerRepo)

	notifRepo := notification.NewRepository(pool)
	notifService := notification.NewService(notifRepo, "", cfg.Notification.FCMAPIKey)
	notifHandler := notification.NewHandler(notifRepo)
	notifWorker := notification.NewWorker(pool, notifService)

	// Billing
	billingRepo := billing.NewRepository(pool)
	billingService := billing.NewService(billingRepo)
	billingHandler := billing.NewHandler(billingService, billingRepo)
	billingWorker := billing.NewWorker(pool, billingService, billingRepo, &notifAdapter{notifService: notifService})
	appointmentHandler.SetBillingRecorder(&billingAdapter{billingService: billingService})

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Metrics())
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.CORS([]string{"*"})) // Restrict in production
	r.Use(middleware.RateLimit(100, 200))

	// Operational endpoints (restrict /metrics via Nginx in production)
	r.Get("/health", healthHandler(pool))
	r.Handle("/metrics", promhttp.Handler())

	// Auth routes (public)
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/login/employee", authHandler.LoginEmployee)
		r.Post("/google", authHandler.GoogleLogin)
		r.Post("/join", authHandler.Join)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Public routes (no auth)
	r.Get("/v1/search", searchHandler.Search)
	r.Get("/v1/providers/{slug}", appointmentHandler.GetProviderProfile)
	r.Get("/v1/providers/{slug}/employees/{employeeId}/slots", appointmentHandler.GetSlots)
	r.Post("/v1/appointments", appointmentHandler.Create)
	r.Get("/v1/appointments/{id}", appointmentHandler.GetByID)
	r.Post("/v1/appointments/{id}/cancel", appointmentHandler.Cancel)
	r.Post("/v1/appointments/{id}/reschedule", appointmentHandler.Reschedule)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtManager))

		// Provider profile
		r.Get("/v1/provider/me", providerHandler.GetMe)
		r.Put("/v1/provider/me", providerHandler.UpdateMe)
		r.Get("/v1/provider/me/qr", qrHandler.GetQR)
		r.Get("/v1/provider/me/stats", providerHandler.GetStats)

		// Services CRUD
		r.Post("/v1/services", providerHandler.CreateService)
		r.Get("/v1/services", providerHandler.ListServices)
		r.Get("/v1/services/{id}", providerHandler.GetService)
		r.Put("/v1/services/{id}", providerHandler.UpdateService)
		r.Delete("/v1/services/{id}", providerHandler.DeleteService)

		// Employees CRUD (admin only)
		r.Route("/v1/employees", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/", employeeHandler.List)
			r.Post("/", employeeHandler.Create)
			r.Get("/invitations", employeeHandler.ListInvitations)
			r.Post("/invitations", employeeHandler.CreateInvitation)
			r.Get("/{id}", employeeHandler.GetByID)
			r.Put("/{id}", employeeHandler.Update)
			r.Delete("/{id}", employeeHandler.Delete)
			r.Put("/{id}/services", employeeHandler.AssignServices)
		})

		// Schedules
		r.Put("/v1/employees/{employeeId}/schedules", scheduleHandler.SetSchedule)
		r.Get("/v1/employees/{employeeId}/schedules", scheduleHandler.GetSchedule)
		r.Post("/v1/employees/{employeeId}/schedule-exceptions", scheduleHandler.AddException)
		r.Get("/v1/employees/{employeeId}/schedule-exceptions", scheduleHandler.ListExceptions)
		r.Delete("/v1/schedule-exceptions/{id}", scheduleHandler.DeleteException)

		// Push tokens
		r.Post("/v1/push-tokens", notifHandler.RegisterToken)
		r.Delete("/v1/push-tokens", notifHandler.DeregisterToken)

		// Billing
		r.Get("/v1/billing/usage", billingHandler.GetCurrentUsage)
		r.Get("/v1/billing/history", billingHandler.GetUsageHistory)
		r.Get("/v1/billing/transactions", billingHandler.GetTransactions)

		// Appointments (provider management)
		r.Get("/v1/appointments", appointmentHandler.ListByProvider)
		r.Put("/v1/appointments/{id}/status", appointmentHandler.UpdateStatus)
		r.Post("/v1/appointments/{id}/provider-cancel", appointmentHandler.ProviderCancel)
		r.Put("/v1/appointments/{id}/reassign", appointmentHandler.Reassign)
		r.Post("/v1/appointments/walk-in", appointmentHandler.WalkIn)
	})

	// Start metrics worker (every 5 minutes)
	metricsWorker := worker.NewWorker(pool)
	workerStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				workerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := metricsWorker.Run(workerCtx); err != nil {
					logger.Error("metrics worker failed", slog.String("error", err.Error()))
				}
				cancel()
			case <-workerStop:
				return
			}
		}
	}()

	// Start reminder worker (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				workerCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				if err := notifWorker.RunReminders(workerCtx); err != nil {
					logger.Error("reminder worker failed", slog.String("error", err.Error()))
				}
				cancel()
			case <-workerStop:
				return
			}
		}
	}()

	// Start monthly billing worker (checks every hour, runs cycle on 1st of month)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		var lastRunMonth time.Month
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				if now.Day() == 1 && now.Month() != lastRunMonth {
					workerCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					if err := billingWorker.RunMonthlyCycle(workerCtx); err != nil {
						logger.Error("billing monthly cycle failed", slog.String("error", err.Error()))
					}
					if err := billingWorker.RunInvoiceNotifications(workerCtx); err != nil {
						logger.Error("billing invoice notifications failed", slog.String("error", err.Error()))
					}
					cancel()
					lastRunMonth = now.Month()
				}
			case <-workerStop:
				return
			}
		}
	}()

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

	close(workerStop)
	logger.Info("shutting down server")
	return srv.Shutdown(shutdownCtx)
}

type dbPinger interface {
	Ping(ctx context.Context) error
}

// statsAdapter bridges the appointment.Repository to the provider.StatsProvider interface.
type statsAdapter struct {
	appointmentRepo *appointment.Repository
}

func (a *statsAdapter) GetStats(ctx context.Context, providerID uuid.UUID, fromDate, toDate string) (*provider.StatsResult, error) {
	s, err := a.appointmentRepo.GetStats(ctx, providerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	return &provider.StatsResult{
		Total:     s.Total,
		Confirmed: s.Confirmed,
		Completed: s.Completed,
		Cancelled: s.Cancelled,
		NoShow:    s.NoShow,
	}, nil
}

func (a *statsAdapter) GetStatsByEmployee(ctx context.Context, providerID uuid.UUID, fromDate, toDate string) ([]provider.EmployeeStatsResult, error) {
	es, err := a.appointmentRepo.GetStatsByEmployee(ctx, providerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	result := make([]provider.EmployeeStatsResult, len(es))
	for i, e := range es {
		result[i] = provider.EmployeeStatsResult{
			EmployeeID:   e.EmployeeID,
			EmployeeName: e.EmployeeName,
			Stats: provider.StatsResult{
				Total:     e.Stats.Total,
				Confirmed: e.Stats.Confirmed,
				Completed: e.Stats.Completed,
				Cancelled: e.Stats.Cancelled,
				NoShow:    e.Stats.NoShow,
			},
		}
	}
	return result, nil
}

// billingAdapter bridges billing.Service to appointment.BillingRecorder interface.
type billingAdapter struct {
	billingService *billing.Service
}

func (a *billingAdapter) RecordCompletion(ctx context.Context, providerID uuid.UUID) (bool, error) {
	_, exceeded, err := a.billingService.RecordCompletion(ctx, providerID)
	return exceeded, err
}

// notifAdapter bridges notification.Service to billing.NotificationSender interface.
type notifAdapter struct {
	notifService *notification.Service
}

func (a *notifAdapter) SendToProvider(ctx context.Context, providerID uuid.UUID, msg billing.NotificationMessage) error {
	return a.notifService.SendToProvider(ctx, providerID, notification.Message{
		Title: msg.Title,
		Body:  msg.Body,
		Data:  msg.Data,
	})
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
