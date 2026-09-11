package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo?id_token="

// VerifyGoogleToken validates a Google id_token and returns the claims.
// It checks that the token's audience matches the expected client ID to prevent
// token confusion attacks. Uses Google's tokeninfo endpoint for simplicity;
// for high-throughput production use, consider verifying JWT signature locally
// with Google's JWKS.
func VerifyGoogleToken(ctx context.Context, idToken, expectedClientID string) (*GoogleClaims, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleTokenInfoURL+idToken, nil)
	if err != nil {
		return nil, fmt.Errorf("auth.google: create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth.google: verify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth.google: invalid token (status %d)", resp.StatusCode)
	}

	var result struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Aud           string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("auth.google: decode response: %w", err)
	}

	if result.Sub == "" || result.Email == "" {
		return nil, fmt.Errorf("auth.google: missing required claims")
	}

	if result.EmailVerified != "true" {
		return nil, fmt.Errorf("auth.google: email not verified")
	}

	// Validate audience to prevent token confusion attacks
	if expectedClientID != "" && result.Aud != expectedClientID {
		return nil, fmt.Errorf("auth.google: audience mismatch")
	}

	return &GoogleClaims{
		Sub:   result.Sub,
		Email: result.Email,
		Name:  result.Name,
	}, nil
}
