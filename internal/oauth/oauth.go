package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ironicbadger/jankey/pkg/models"
)

const (
	TailscaleOAuthURL = "https://api.tailscale.com/api/v2/oauth/token"
)

// Client represents an OAuth client for Tailscale API
type Client struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	verbose      bool
}

// New creates a new OAuth client
func New(clientID, clientSecret string, verbose bool) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		verbose: verbose,
	}
}

// GetAccessToken exchanges OAuth credentials for an access token
func (c *Client) GetAccessToken() (string, error) {
	// Prepare form data
	formData := url.Values{}
	formData.Set("client_id", c.clientID)
	formData.Set("client_secret", c.clientSecret)
	formData.Set("grant_type", "client_credentials")

	if c.verbose {
		fmt.Println("→ Requesting OAuth access token from Tailscale API...")
		fmt.Printf("  URL: %s\n", TailscaleOAuthURL)
		fmt.Printf("  Client ID: %s\n", c.redactClientID())
	}

	// Create request
	req, err := http.NewRequest("POST", TailscaleOAuthURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create OAuth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request with retry logic
	resp, err := c.executeWithRetry(req, 3)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OAuth response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", c.handleOAuthError(resp.StatusCode, body)
	}

	// Parse response
	var tokenResp models.OAuthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse OAuth response: %w", err)
	}

	if c.verbose {
		fmt.Printf("✓ OAuth access token obtained (expires in %d seconds)\n", tokenResp.ExpiresIn)
	}

	return tokenResp.AccessToken, nil
}

// executeWithRetry executes an HTTP request with exponential backoff retry
func (c *Client) executeWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(1<<uint(attempt-1)) * time.Second
			if c.verbose {
				fmt.Printf("  Retry attempt %d/%d after %v...\n", attempt, maxRetries, waitTime)
			}
			time.Sleep(waitTime)
		}

		resp, err = c.httpClient.Do(req)
		if err == nil {
			return resp, nil
		}

		// Don't retry on non-network errors
		if !isNetworkError(err) {
			break
		}

		if c.verbose {
			fmt.Printf("  Network error: %v\n", err)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

// handleOAuthError formats OAuth API errors
func (c *Client) handleOAuthError(statusCode int, body []byte) error {
	var errorMsg string

	// Try to parse error response
	var errorResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errorResp.Error != "" {
			errorMsg = errorResp.Error
			if errorResp.ErrorDescription != "" {
				errorMsg += ": " + errorResp.ErrorDescription
			}
		} else if errorResp.Message != "" {
			errorMsg = errorResp.Message
		}
	}

	if errorMsg == "" {
		errorMsg = string(body)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("OAuth credentials invalid (401): %s\n\nPlease check your OAuth client ID and secret.\nSee: https://tailscale.com/kb/1215/oauth-clients", errorMsg)
	case http.StatusForbidden:
		return fmt.Errorf("OAuth access forbidden (403): %s\n\nEnsure your OAuth client has the required scopes (auth_keys or devices:write)", errorMsg)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited (429): %s\n\nPlease wait before retrying", errorMsg)
	default:
		return fmt.Errorf("OAuth request failed (%d): %s", statusCode, errorMsg)
	}
}

// redactClientID returns a redacted version of the client ID for logging
func (c *Client) redactClientID() string {
	if len(c.clientID) <= 8 {
		return "****"
	}
	return c.clientID[:4] + "****" + c.clientID[len(c.clientID)-4:]
}

// isNetworkError checks if an error is network-related (retryable)
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common network errors
	errStr := err.Error()
	return contains(errStr, "timeout") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "no such host") ||
		contains(errStr, "temporary failure")
}

// contains is a simple case-insensitive string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
