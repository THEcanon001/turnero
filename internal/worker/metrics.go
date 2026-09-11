package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	activeProviders = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "turnero_active_providers",
		Help: "Number of active providers.",
	})

	activeEmployees = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "turnero_active_employees",
		Help: "Number of active employees.",
	})

	appointmentsToday = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "turnero_appointments_today",
		Help: "Number of appointments today by status.",
	}, []string{"status"})

	billingUnpaidTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "turnero_billing_unpaid_total",
		Help: "Total unpaid billing amount across all providers.",
	})
)

func init() {
	prometheus.MustRegister(activeProviders, activeEmployees, appointmentsToday, billingUnpaidTotal)
}

// Worker collects business metrics periodically.
type Worker struct {
	pool *pgxpool.Pool
}

// NewWorker creates a new metrics worker.
func NewWorker(pool *pgxpool.Pool) *Worker {
	return &Worker{pool: pool}
}

// Run collects all business metrics.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.collectProviders(ctx); err != nil {
		slog.Error("worker.metrics: collectProviders", slog.String("error", err.Error()))
	}
	if err := w.collectEmployees(ctx); err != nil {
		slog.Error("worker.metrics: collectEmployees", slog.String("error", err.Error()))
	}
	if err := w.collectAppointmentsToday(ctx); err != nil {
		slog.Error("worker.metrics: collectAppointmentsToday", slog.String("error", err.Error()))
	}
	if err := w.collectBillingUnpaid(ctx); err != nil {
		slog.Error("worker.metrics: collectBillingUnpaid", slog.String("error", err.Error()))
	}
	return nil
}

func (w *Worker) collectProviders(ctx context.Context) error {
	var count int
	err := w.pool.QueryRow(ctx, `SELECT COUNT(*) FROM providers WHERE is_active = true`).Scan(&count)
	if err != nil {
		return fmt.Errorf("worker.metrics: collectProviders: %w", err)
	}
	activeProviders.Set(float64(count))
	return nil
}

func (w *Worker) collectEmployees(ctx context.Context) error {
	var count int
	err := w.pool.QueryRow(ctx, `SELECT COUNT(*) FROM employees WHERE is_active = true`).Scan(&count)
	if err != nil {
		return fmt.Errorf("worker.metrics: collectEmployees: %w", err)
	}
	activeEmployees.Set(float64(count))
	return nil
}

func (w *Worker) collectAppointmentsToday(ctx context.Context) error {
	rows, err := w.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM appointments
		WHERE date = CURRENT_DATE
		GROUP BY status`)
	if err != nil {
		return fmt.Errorf("worker.metrics: collectAppointmentsToday: %w", err)
	}
	defer rows.Close()

	// Reset all statuses to zero first
	appointmentsToday.WithLabelValues("confirmed").Set(0)
	appointmentsToday.WithLabelValues("completed").Set(0)
	appointmentsToday.WithLabelValues("cancelled").Set(0)
	appointmentsToday.WithLabelValues("no_show").Set(0)

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return fmt.Errorf("worker.metrics: collectAppointmentsToday: scan: %w", err)
		}
		appointmentsToday.WithLabelValues(status).Set(float64(count))
	}
	return rows.Err()
}

func (w *Worker) collectBillingUnpaid(ctx context.Context) error {
	var total float64
	err := w.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_due), 0)
		FROM billing_usage
		WHERE is_paid = false AND amount_due > 0`).Scan(&total)
	if err != nil {
		return fmt.Errorf("worker.metrics: collectBillingUnpaid: %w", err)
	}
	billingUnpaidTotal.Set(total)
	return nil
}
