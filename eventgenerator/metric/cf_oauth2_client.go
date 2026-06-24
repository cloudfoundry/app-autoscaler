package metric

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

// CFOauth2HTTPClient is an OAuth2 HTTP client that uses Basic auth header
// for the token request, which is required by CF's "cf" UAA client.
// This is necessary because the go-log-cache library sends client credentials
// in the request body, but the "cf" client requires Basic auth header.
type CFOauth2HTTPClient struct {
	tokenURL        string
	basicAuthHeader string
	username        string
	password        string

	httpClient *http.Client

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// NewCFOauth2HTTPClient creates a new OAuth2 HTTP client that is compatible
// with CF's "cf" UAA client by using Basic auth header for authentication.
func NewCFOauth2HTTPClient(oauth2URL, clientID, clientSecret, username, password string, skipSSLValidation bool) *CFOauth2HTTPClient {
	tokenURL := oauth2URL
	if !strings.HasSuffix(tokenURL, "/oauth/token") {
		tokenURL = strings.TrimSuffix(tokenURL, "/") + "/oauth/token"
	}

	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret))

	return &CFOauth2HTTPClient{
		tokenURL:        tokenURL,
		basicAuthHeader: basicAuthHeader,
		username:        username,
		password:        password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// #nosec G402 -- controlled by config, used for dev/test environments
					InsecureSkipVerify: skipSSLValidation,
				},
			},
		},
	}
}

// Do implements the HTTPClient interface for go-log-cache.
// It adds the Bearer token to the request and handles 401 responses by refreshing the token.
func (c *CFOauth2HTTPClient) Do(req *http.Request) (*http.Response, error) {
	token, err := c.getValidToken()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	// #nosec G704 -- URL comes from user-configured metrics endpoint
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we get 401, force refresh the token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		token, err = c.forceRefreshToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		// #nosec G704 -- URL comes from user-configured metrics endpoint
		return c.httpClient.Do(req)
	}

	return resp, nil
}

// getValidToken returns a valid token, refreshing it if necessary.
// Uses double-checked locking to prevent multiple concurrent token refreshes.
func (c *CFOauth2HTTPClient) getValidToken() (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expiresAt) {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	return c.refreshToken()
}

// forceRefreshToken refreshes the token after a 401 response.
// The rejectedToken parameter is the token that was rejected by the server.
// If another goroutine has already refreshed the token (i.e., the cached token
// differs from the rejected one), the new token is returned without re-fetching.
func (c *CFOauth2HTTPClient) forceRefreshToken(rejectedToken string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If the token has changed since we got the 401, another goroutine already refreshed it
	if c.token != rejectedToken {
		if c.token != "" && time.Now().Before(c.expiresAt) {
			return c.token, nil
		}
		// Cached token is invalid, refresh anyway
		return c.doRefreshToken()
	}

	return c.doRefreshToken()
}

func (c *CFOauth2HTTPClient) refreshToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: another goroutine may have refreshed the token while we waited for the lock
	if c.token != "" && time.Now().Before(c.expiresAt) {
		return c.token, nil
	}

	return c.doRefreshToken()
}

// doRefreshToken performs the actual token refresh. Must be called with lock held.
func (c *CFOauth2HTTPClient) doRefreshToken() (string, error) {
	data := url.Values{}
	data.Set("grant_type", models.GrantTypePassword)
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequest(http.MethodPost, c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Use Basic auth header for client credentials (required by CF's "cf" client)
	req.Header.Set("Authorization", c.basicAuthHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	c.token = tokenResp.AccessToken
	// Calculate expiration time with 30 second buffer for clock skew
	c.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-30) * time.Second)
	return c.token, nil
}
