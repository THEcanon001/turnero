package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Worker runs periodic notification tasks.
type Worker struct {
	pool    *pgxpool.Pool
	service *Service
}

// NewWorker creates a new notification worker.
func NewWorker(pool *pgxpool.Pool, service *Service) *Worker {
	return &Worker{
		pool:    pool,
		service: service,
	}
}

// RunReminders checks for upcoming appointments and sends reminder push notifications.
// Should be called periodically (e.g., every 5 minutes).
func (w *Worker) RunReminders(ctx context.Context) error {
	// Find appointments in the next 24h or 2h that haven't had reminders sent
	query := `
		SELECT a.id, a.provider_id, a.employee_id, a.client_name, a.date::text, a.start_time::text
		FROM appointments a
		WHERE a.status = 'confirmed'
		AND a.reminder_sent = false
		AND (a.date + a.start_time::time) <= (now() + interval '24 hours')
		AND (a.date + a.start_time::time) > now()
		LIMIT 100`

	rows, err := w.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("notification.worker: query reminders: %w", err)
	}
	defer rows.Close()

	type reminder struct {
		id, providerID, employeeID string
		clientName, date, startTime string
	}

	var reminders []reminder
	for rows.Next() {
		var r reminder
		if err := rows.Scan(&r.id, &r.providerID, &r.employeeID, &r.clientName, &r.date, &r.startTime); err != nil {
			return fmt.Errorf("notification.worker: scan: %w", err)
		}
		reminders = append(reminders, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("notification.worker: rows: %w", err)
	}

	for _, r := range reminders {
		msg := Message{
			Title: "Recordatorio de turno",
			Body:  fmt.Sprintf("%s tiene turno el %s a las %s", r.clientName, r.date, r.startTime),
			Data: map[string]string{
				"appointment_id": r.id,
				"type":           "reminder",
			},
		}

		// Send to provider
		if err := w.service.sendToAll(ctx, r.providerID, r.employeeID, msg); err != nil {
			slog.Error("notification.worker: send reminder", slog.String("appointment", r.id), slog.String("error", err.Error()))
			continue
		}

		// Mark reminder as sent
		if _, err := w.pool.Exec(ctx, `UPDATE appointments SET reminder_sent = true WHERE id = $1::uuid`, r.id); err != nil {
			slog.Error("notification.worker: mark sent", slog.String("appointment", r.id), slog.String("error", err.Error()))
		}
	}

	slog.Info("notification.worker: reminders processed", slog.Int("count", len(reminders)))
	return nil
}

// RunWeeklySummary sends a weekly summary push to all providers.
// Should be called once a week (e.g., Monday 9 AM).
func (w *Worker) RunWeeklySummary(ctx context.Context) error {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -7).Format("2006-01-02")
	weekEnd := now.Format("2006-01-02")

	query := `
		SELECT p.id::text, p.name,
			COUNT(*) FILTER (WHERE a.status = 'completed') as completed,
			COUNT(*) FILTER (WHERE a.status = 'cancelled') as cancelled,
			COUNT(*) FILTER (WHERE a.status = 'no_show') as no_show
		FROM providers p
		LEFT JOIN appointments a ON a.provider_id = p.id
			AND a.date >= $1::date AND a.date < $2::date
		WHERE p.is_active = true
		GROUP BY p.id, p.name`

	rows, err := w.pool.Query(ctx, query, weekStart, weekEnd)
	if err != nil {
		return fmt.Errorf("notification.worker: query summary: %w", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var providerID, name string
		var completed, cancelled, noShow int
		if err := rows.Scan(&providerID, &name, &completed, &cancelled, &noShow); err != nil {
			slog.Error("notification.worker: scan summary", slog.String("error", err.Error()))
			continue
		}

		msg := Message{
			Title: "Resumen semanal",
			Body:  fmt.Sprintf("Completados: %d | Cancelados: %d | No-show: %d", completed, cancelled, noShow),
			Data: map[string]string{
				"type": "weekly_summary",
			},
		}

		if err := w.service.sendToAll(ctx, providerID, "", msg); err != nil {
			slog.Error("notification.worker: send summary", slog.String("provider", providerID), slog.String("error", err.Error()))
		}
		count++
	}

	slog.Info("notification.worker: weekly summaries sent", slog.Int("count", count))
	return nil
}

// sendToAll sends a notification to provider and optionally employee devices.
func (s *Service) sendToAll(ctx context.Context, providerID, employeeID string, msg Message) error {
	if providerID != "" {
		tokens, err := s.repo.queryTokens(ctx,
			`SELECT id, provider_id, employee_id, client_phone, token, platform, is_active, created_at, updated_at
			 FROM push_tokens WHERE provider_id = $1::uuid AND is_active = true`, providerID)
		if err == nil {
			_ = s.sendToTokens(ctx, tokens, msg)
		}
	}
	if employeeID != "" {
		tokens, err := s.repo.queryTokens(ctx,
			`SELECT id, provider_id, employee_id, client_phone, token, platform, is_active, created_at, updated_at
			 FROM push_tokens WHERE employee_id = $1::uuid AND is_active = true`, employeeID)
		if err == nil {
			_ = s.sendToTokens(ctx, tokens, msg)
		}
	}
	return nil
}
