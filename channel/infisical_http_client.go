package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// InfisicalHTTPClient implements InfisicalClient by calling the Infisical REST
// API directly (Universal Auth machine identity). This avoids the
// github.com/infisical/go-sdk dependency while using the same stable v3 API.
//
// Auth flow: POST /api/v1/auth/universal-auth/login → short-lived access token.
// Token is refreshed automatically when it expires.
type InfisicalHTTPClient struct {
	siteURL      string // e.g. "https://app.infisical.com"
	clientID     string
	clientSecret string
	projectID    string
	httpClient   *http.Client

	mu        sync.Mutex
	token     string
	tokenExp  time.Time
}

// NewInfisicalHTTPClient constructs an HTTP client for the Infisical API.
// siteURL defaults to "https://app.infisical.com" when empty.
func NewInfisicalHTTPClient(siteURL, clientID, clientSecret, projectID string) *InfisicalHTTPClient {
	if siteURL == "" {
		siteURL = "https://app.infisical.com"
	}
	return &InfisicalHTTPClient{
		siteURL:      siteURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		projectID:    projectID,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// authTokenResponse is the JSON shape returned by the Universal Auth login endpoint.
type authTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"` // seconds
}

// ensureToken fetches or refreshes the access token. Caller must not hold mu.
func (c *InfisicalHTTPClient) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check inside lock; leave 30 s buffer before expiry.
	if c.token != "" && time.Now().Add(30*time.Second).Before(c.tokenExp) {
		return c.token, nil
	}

	body, _ := json.Marshal(map[string]string{
		"clientId":     c.clientID,
		"clientSecret": c.clientSecret,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.siteURL+"/api/v1/auth/universal-auth/login",
		bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("infisical: build auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("infisical: auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("infisical: auth HTTP %d: %s", resp.StatusCode, b)
	}

	var tok authTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("infisical: decode auth response: %w", err)
	}

	c.token = tok.AccessToken
	if tok.ExpiresIn > 0 {
		c.tokenExp = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	} else {
		c.tokenExp = time.Now().Add(1 * time.Hour) // safe default
	}

	return c.token, nil
}

// getSecretResponse is the JSON shape for GET /api/v3/secrets/raw/{name}.
type getSecretResponse struct {
	Secret struct {
		SecretValue string `json:"secretValue"`
	} `json:"secret"`
}

// GetSecret implements InfisicalClient.
func (c *InfisicalHTTPClient) GetSecret(ctx context.Context, key, environment, secretPath string) (string, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/api/v3/secrets/raw/%s", c.siteURL, url.PathEscape(key))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("infisical: build get request: %w", err)
	}
	q := req.URL.Query()
	q.Set("workspaceId", c.projectID)
	q.Set("environment", environment)
	q.Set("secretPath", secretPath)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("infisical: get secret %q: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("infisical: secret %q not found", key)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("infisical: get secret %q HTTP %d: %s", key, resp.StatusCode, b)
	}

	var payload getSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("infisical: decode get secret response: %w", err)
	}

	return payload.Secret.SecretValue, nil
}

// listSecretsResponse is the JSON shape for GET /api/v3/secrets/raw.
type listSecretsResponse struct {
	Secrets []struct {
		SecretKey string `json:"secretKey"`
	} `json:"secrets"`
}

// ListSecrets implements InfisicalClient.
func (c *InfisicalHTTPClient) ListSecrets(ctx context.Context, environment, secretPath string) ([]string, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.siteURL+"/api/v3/secrets/raw", nil)
	if err != nil {
		return nil, fmt.Errorf("infisical: build list request: %w", err)
	}
	q := req.URL.Query()
	q.Set("workspaceId", c.projectID)
	q.Set("environment", environment)
	q.Set("secretPath", secretPath)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("infisical: list secrets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("infisical: list secrets HTTP %d: %s", resp.StatusCode, b)
	}

	var payload listSecretsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("infisical: decode list secrets response: %w", err)
	}

	keys := make([]string, 0, len(payload.Secrets))
	for _, s := range payload.Secrets {
		keys = append(keys, s.SecretKey)
	}
	return keys, nil
}
