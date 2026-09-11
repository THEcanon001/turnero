package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Service sends push notifications via FCM HTTP v1 API.
type Service struct {
	repo       *Repository
	fcmURL     string
	fcmAPIKey  string
	httpClient *http.Client
}

// NewService creates a new notification service.
func NewService(repo *Repository, fcmURL, fcmAPIKey string) *Service {
	if fcmURL == "" {
		fcmURL = "https://fcm.googleapis.com/fcm/send"
	}
	return &Service{
		repo:      repo,
		fcmURL:    fcmURL,
		fcmAPIKey: fcmAPIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendToProvider sends a push notification to all provider devices.
func (s *Service) SendToProvider(ctx context.Context, providerID uuid.UUID, msg Message) error {
	tokens, err := s.repo.GetActiveTokensByProvider(ctx, providerID)
	if err != nil {
		return fmt.Errorf("notification.service: %w", err)
	}
	return s.sendToTokens(ctx, tokens, msg)
}

// SendToEmployee sends a push notification to all employee devices.
func (s *Service) SendToEmployee(ctx context.Context, employeeID uuid.UUID, msg Message) error {
	tokens, err := s.repo.GetActiveTokensByEmployee(ctx, employeeID)
	if err != nil {
		return fmt.Errorf("notification.service: %w", err)
	}
	return s.sendToTokens(ctx, tokens, msg)
}

func (s *Service) sendToTokens(ctx context.Context, tokens []PushToken, msg Message) error {
	if s.fcmAPIKey == "" {
		slog.Warn("notification.service: FCM API key not configured, skipping push")
		return nil
	}

	for _, t := range tokens {
		if err := s.sendFCM(ctx, t.Token, msg); err != nil {
			slog.Error("notification.service: send failed",
				slog.String("token_id", t.ID.String()),
				slog.String("error", err.Error()))
			// Deactivate invalid tokens
			if isInvalidTokenError(err) {
				_ = s.repo.DeactivateToken(ctx, t.Token)
			}
		}
	}
	return nil
}

func (s *Service) sendFCM(ctx context.Context, token string, msg Message) error {
	payload := map[string]any{
		"to": token,
		"notification": map[string]string{
			"title": msg.Title,
			"body":  msg.Body,
		},
		"data": msg.Data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("notification.service: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.fcmURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("notification.service: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+s.fcmAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notification.service: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification.service: FCM returned status %d", resp.StatusCode)
	}

	return nil
}

func isInvalidTokenError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "NotRegistered") || contains(msg, "InvalidRegistration") || contains(msg, "status 401")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
