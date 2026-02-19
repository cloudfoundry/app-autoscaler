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
)

// CFOauth2HTTPClient is an OAuth2 HTTP client that uses Basic auth header
// for the token request, which is required by CF's "cf" UAA client.
// This is necessary because the go-log-cache library sends client credentials
// in the request body, but the "cf" client requires Basic auth header.
type CFOauth2HTTPClient struct {
	oauth2URL         string
	clientID          string
	clientSecret      string
	username          string
	password          string
	skipSSLValidation bool

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
	return &CFOauth2HTTPClient{
		oauth2URL:         oauth2URL,
		clientID:          clientID,
		clientSecret:      clientSecret,
		username:          username,
		password:          password,
		skipSSLValidation: skipSSLValidation,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// #nosec G402
					InsecureSkipVerify: skipSSLValidation,
				},
			},
		},
	}
}

// Do implements the HTTPClient interface for go-log-cache.
// It adds the Bearer token to the request and handles 401 responses by refreshing the token.
func (c *CFOauth2HTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.RLock()
	token := c.token
	expiresAt := c.expiresAt
	c.mu.RUnlock()

	// Check if token is missing or expired
	if token == "" || time.Now().After(expiresAt) {
		var err error
		token, err = c.refreshToken()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If we get 401, try to refresh the token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		token, err = c.refreshToken()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		return c.httpClient.Do(req)
	}

	return resp, nil
}

func (c *CFOauth2HTTPClient) refreshToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tokenURL := c.oauth2URL
	if !strings.HasSuffix(tokenURL, "/oauth/token") {
		tokenURL = strings.TrimSuffix(tokenURL, "/") + "/oauth/token"
	}

	// Build form data for password grant
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Use Basic auth header for client credentials (required by CF's "cf" client)
	basicAuth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+basicAuth)

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
