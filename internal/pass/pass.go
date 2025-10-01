package pass

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client represents a pass password manager client
type Client struct {
	passPath string
}

// New creates a new pass client
func New() (*Client, error) {
	// Check if pass is installed
	passPath, err := exec.LookPath("pass")
	if err != nil {
		return nil, fmt.Errorf("pass not found: please install pass (https://www.passwordstore.org/)")
	}

	return &Client{passPath: passPath}, nil
}

// Get retrieves a secret from pass
func (c *Client) Get(path string) (string, error) {
	cmd := exec.Command(c.passPath, "show", path)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "is not in the password store") {
			return "", fmt.Errorf("secret not found at '%s'", path)
		}
		return "", fmt.Errorf("failed to retrieve secret from pass: %s", stderrStr)
	}

	secret := strings.TrimSpace(stdout.String())
	if secret == "" {
		return "", fmt.Errorf("secret at '%s' is empty", path)
	}

	return secret, nil
}

// Insert adds a secret to pass
func (c *Client) Insert(path, value string) error {
	cmd := exec.Command(c.passPath, "insert", "-m", path)

	var stdin bytes.Buffer
	stdin.WriteString(value)
	cmd.Stdin = &stdin

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to insert secret into pass: %s", stderr.String())
	}

	return nil
}

// Exists checks if a secret exists at the given path
func (c *Client) Exists(path string) bool {
	_, err := c.Get(path)
	return err == nil
}

// IsInstalled checks if pass is installed and available
func IsInstalled() bool {
	_, err := exec.LookPath("pass")
	return err == nil
}

// GetFromPassOrEnv attempts to get a value from pass first, then falls back to environment variable
func GetFromPassOrEnv(passClient *Client, passPath, envVar string) (string, error) {
	// Try pass first if client is available
	if passClient != nil {
		value, err := passClient.Get(passPath)
		if err == nil {
			return value, nil
		}
		// If pass fails for any reason other than "not found", we might want to log it
	}

	// Fall back to environment variable
	value := os.Getenv(envVar)
	if value == "" {
		if passClient != nil {
			return "", fmt.Errorf("secret not found in pass at '%s' or environment variable '%s'", passPath, envVar)
		}
		return "", fmt.Errorf("pass not available and environment variable '%s' not set", envVar)
	}

	return value, nil
}
