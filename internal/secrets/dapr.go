// Package secrets provides helpers for fetching secrets from Vault via the Dapr sidecar HTTP API.
// This avoids importing the Dapr Go SDK while still honoring the platform secret delivery contract:
// microservices receive all secrets through Dapr (never from K8s Secrets or env var passwords).
package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// GetDaprSecret fetches a secret from a Dapr secret store via the sidecar HTTP API.
//
//	daprAddr  — sidecar address, e.g. "localhost:3500"
//	storeName — Dapr component name, e.g. "vault-db" or "vault"
//	secretKey — secret path, e.g. "creds/profile-api-role" or "local/mathtrail-profile"
//
// Returns a map of key→value strings from the secret.
// For the vault-db (database engine) component: {"username": "...", "password": "..."}.
// For the vault (KV v2) component: {"redis-password": "...", "db-password": "..."}.
func GetDaprSecret(ctx context.Context, daprAddr, storeName, secretKey string) (map[string]string, error) {
	// URL-encode the secret key path so slashes don't break the URL routing.
	encodedKey := url.PathEscape(secretKey)
	apiURL := fmt.Sprintf("http://%s/v1.0/secrets/%s/%s", daprAddr, storeName, encodedKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("secrets: build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("secrets: dapr http api (%s/%s): %w", storeName, secretKey, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("secrets: dapr returned status %d for %s/%s", resp.StatusCode, storeName, secretKey)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("secrets: decode response for %s/%s: %w", storeName, secretKey, err)
	}
	return result, nil
}

// GetDaprSecretWithRetry wraps GetDaprSecret with linear backoff.
// The Dapr sidecar starts concurrently with the app; retrying handles the race.
func GetDaprSecretWithRetry(ctx context.Context, daprAddr, storeName, secretKey string, maxAttempts int) (map[string]string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := GetDaprSecret(ctx, daprAddr, storeName, secretKey)
		if err == nil {
			return result, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}
	return nil, fmt.Errorf("secrets: dapr unavailable after %d attempts: %w", maxAttempts, lastErr)
}
