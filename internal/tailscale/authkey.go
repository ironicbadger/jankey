package tailscale

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ironicbadger/jankey/internal/models"
)

const (
	TailscaleAuthKeyURL = "https://api.tailscale.com/api/v2/tailnet/-/keys"
)

// AuthKey represents a Tailscale auth key
type AuthKey struct {
	ID          string    `json:"id"`
	Created     time.Time `json:"created"`
	Expires     time.Time `json:"expires"`
	Description string    `json:"description"`
}

// Client represents a Tailscale API client
type Client struct {
	accessToken string
	httpClient  *http.Client
	verbose     bool
}

// New creates a new Tailscale API client
func New(accessToken string, verbose bool) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		verbose: verbose,
	}
}

// AuthKeyOptions holds options for creating an auth key
type AuthKeyOptions struct {
	Ephemeral     bool
	Reusable      bool
	Preauthorized bool
	ExpiryDays    int
	Tags          []string
	Description   string
}

// CreateAuthKey creates a new Tailscale auth key
func (c *Client) CreateAuthKey(opts AuthKeyOptions) (*models.AuthKeyResponse, error) {
	// Validate tags
	if len(opts.Tags) == 0 {
		return nil, fmt.Errorf("tags are required by Tailscale API (must specify at least one tag)")
	}

	// Calculate expiry seconds
	var expirySeconds int64
	if opts.ExpiryDays > 0 {
		expirySeconds = int64(opts.ExpiryDays * 24 * 60 * 60)
	} else {
		// 0 means maximum (90 days)
		expirySeconds = 0
	}

	// Build request
	reqBody := models.AuthKeyRequest{
		Capabilities: models.Capabilities{
			Devices: models.DeviceCapabilities{
				Create: models.DeviceCreateCapabilities{
					Reusable:      opts.Reusable,
					Ephemeral:     opts.Ephemeral,
					Preauthorized: opts.Preauthorized,
					Tags:          opts.Tags,
				},
			},
		},
		ExpirySeconds: expirySeconds,
		Description:   opts.Description,
	}

	// Marshal request body
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal auth key request: %w", err)
	}

	if c.verbose {
		fmt.Println("\n→ Creating Tailscale auth key...")
		fmt.Printf("  URL: %s\n", TailscaleAuthKeyURL)
		fmt.Printf("  Request body:\n%s\n", c.formatJSON(jsonData))
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", TailscaleAuthKeyURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth key request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.executeWithRetry(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth key response: %w", err)
	}

	if c.verbose {
		fmt.Printf("  Response status: %d\n", resp.StatusCode)
		fmt.Printf("  Response body:\n%s\n", c.formatJSON(body))
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleAPIError(resp.StatusCode, body)
	}

	// Parse response
	var authKeyResp models.AuthKeyResponse
	if err := json.Unmarshal(body, &authKeyResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth key response: %w", err)
	}

	if c.verbose {
		fmt.Printf("✓ Auth key created successfully\n")
		fmt.Printf("  ID: %s\n", authKeyResp.ID)
		fmt.Printf("  Expires: %s\n", authKeyResp.Expires.Format(time.RFC3339))
	}

	return &authKeyResp, nil
}

// ListAuthKeys lists all auth keys for the tailnet
func (c *Client) ListAuthKeys() ([]AuthKey, error) {
	if c.verbose {
		fmt.Println("\n→ Listing auth keys...")
	}

	req, err := http.NewRequest("GET", TailscaleAuthKeyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.executeWithRetry(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read list response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleAPIError(resp.StatusCode, body)
	}

	var listResp struct {
		Keys []AuthKey `json:"keys"`
	}

	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	if c.verbose {
		fmt.Printf("✓ Found %d auth key(s)\n", len(listResp.Keys))
	}

	return listResp.Keys, nil
}

// DeleteAuthKey deletes an auth key by ID
func (c *Client) DeleteAuthKey(keyID string) error {
	deleteURL := fmt.Sprintf("%s/%s", TailscaleAuthKeyURL, keyID)

	if c.verbose {
		fmt.Printf("\n→ Deleting auth key %s...\n", keyID)
	}

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.executeWithRetry(req, 3)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return c.handleAPIError(resp.StatusCode, body)
	}

	if c.verbose {
		fmt.Printf("✓ Auth key %s deleted\n", keyID)
	}

	return nil
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

// handleAPIError formats Tailscale API errors
func (c *Client) handleAPIError(statusCode int, body []byte) error {
	var errorMsg string

	// Try to parse error response
	var errorResp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errorResp.Message != "" {
			errorMsg = errorResp.Message
		} else if errorResp.Error != "" {
			errorMsg = errorResp.Error
		}
	}

	if errorMsg == "" {
		errorMsg = string(body)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("API token invalid (401): %s\n\nThe OAuth access token may have expired or is invalid", errorMsg)
	case http.StatusForbidden:
		return fmt.Errorf("access forbidden (403): %s\n\nEnsure your OAuth client has the required permissions", errorMsg)
	case http.StatusBadRequest:
		if contains(errorMsg, "capability") {
			return fmt.Errorf("invalid request (400): %s\n\nThis may be due to missing or invalid tags in the request", errorMsg)
		}
		return fmt.Errorf("invalid request (400): %s", errorMsg)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited (429): %s\n\nPlease wait before retrying", errorMsg)
	default:
		return fmt.Errorf("API request failed (%d): %s", statusCode, errorMsg)
	}
}

// formatJSON formats JSON for pretty printing
func (c *Client) formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "  ", "  "); err != nil {
		return string(data)
	}
	return prettyJSON.String()
}

// isNetworkError checks if an error is network-related (retryable)
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "timeout") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "no such host") ||
		contains(errStr, "temporary failure")
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
