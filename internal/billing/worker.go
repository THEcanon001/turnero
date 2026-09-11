package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationSender abstracts push notification delivery to avoid import cycles.
type NotificationSender interface {
	SendToProvider(ctx context.Context, providerID uuid.UUID, msg NotificationMessage) error
}

// NotificationMessage is a simplified message struct to avoid importing notification package.
type NotificationMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// Worker runs periodic billing tasks.
type Worker struct {
	pool          *pgxpool.Pool
	service       *Service
	repo          *Repository
	notifService  NotificationSender
}

// NewWorker creates a new billing worker.
func NewWorker(pool *pgxpool.Pool, service *Service, repo *Repository, notifService NotificationSender) *Worker {
	return &Worker{
		pool:         pool,
		service:      service,
		repo:         repo,
		notifService: notifService,
	}
}

// RunMonthlyCycle ensures billing_usage rows exist for the current month for all providers
// and recalculates totals from the appointments table.
func (w *Worker) RunMonthlyCycle(ctx context.Context) error {
	now := time.Now()

	providerIDs, err := w.repo.ListAllProviderIDs(ctx)
	if err != nil {
		return fmt.Errorf("billing.worker: RunMonthlyCycle: %w", err)
	}

	var processed int
	for _, pid := range providerIDs {
		usage, err := w.repo.GetOrCreateUsage(ctx, pid, now)
		if err != nil {
			slog.Error("billing.worker: RunMonthlyCycle: get or create usage",
				slog.String("provider", pid.String()),
				slog.String("error", err.Error()))
			continue
		}

		// Recalculate from actual appointment data
		completed, err := w.repo.CountCompletedForMonth(ctx, pid, now)
		if err != nil {
			slog.Error("billing.worker: RunMonthlyCycle: count completed",
				slog.String("provider", pid.String()),
				slog.String("error", err.Error()))
			continue
		}

		if err := w.repo.SyncUsageTotals(ctx, usage.ID, completed); err != nil {
			slog.Error("billing.worker: RunMonthlyCycle: sync totals",
				slog.String("provider", pid.String()),
				slog.String("error", err.Error()))
			continue
		}

		processed++
	}

	slog.Info("billing.worker: RunMonthlyCycle completed",
		slog.Int("providers_processed", processed),
		slog.Int("providers_total", len(providerIDs)))
	return nil
}

// RunInvoiceNotifications sends push notifications to providers with outstanding amounts.
func (w *Worker) RunInvoiceNotifications(ctx context.Context) error {
	if w.notifService == nil {
		return nil
	}

	usages, err := w.repo.ListUnpaidWithAmountDue(ctx)
	if err != nil {
		return fmt.Errorf("billing.worker: RunInvoiceNotifications: %w", err)
	}

	var sent int
	for _, u := range usages {
		msg := NotificationMessage{
			Title: "Factura pendiente",
			Body:  fmt.Sprintf("Tienes un saldo pendiente de $%.2f %s del mes %s", u.AmountDue, Currency, u.Month.Format("2006-01")),
			Data: map[string]string{
				"type":     "billing_invoice",
				"month":    u.Month.Format("2006-01"),
				"amount":   fmt.Sprintf("%.2f", u.AmountDue),
				"currency": Currency,
			},
		}

		if err := w.notifService.SendToProvider(ctx, u.ProviderID, msg); err != nil {
			slog.Error("billing.worker: RunInvoiceNotifications: send",
				slog.String("provider", u.ProviderID.String()),
				slog.String("error", err.Error()))
			continue
		}
		sent++
	}

	slog.Info("billing.worker: RunInvoiceNotifications completed",
		slog.Int("notifications_sent", sent),
		slog.Int("unpaid_total", len(usages)))
	return nil
}
