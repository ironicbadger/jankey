package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ironicbadger/jankey/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir  = ".config/jankey"
	DefaultConfigFile = "config.yaml"
)

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFile), nil
}

// Load reads and parses the config file
func Load(configPath string) (*models.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s: run with --init to create one", configPath)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save writes the config to the specified path
func Save(config *models.Config, configPath string) error {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadOrDefault loads config or returns defaults if not found
func LoadOrDefault(configPath string) (*models.Config, error) {
	config, err := Load(configPath)
	if err != nil {
		// If config doesn't exist, return defaults
		if os.IsNotExist(err) || err.Error() == fmt.Sprintf("config file not found at %s: run with --init to create one", configPath) {
			return GetDefaultConfig(), nil
		}
		return nil, err
	}
	return config, nil
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *models.Config {
	return &models.Config{
		APIKey: models.APIKeyConfig{
			PassPathAPIKey: "tailscale/api-key",
		},
		OAuth: models.OAuthConfig{
			PassPathClientID:     "tailscale/oauth-client-id",
			PassPathClientSecret: "tailscale/oauth-client-secret",
		},
		AuthKeyDefaults: models.AuthKeyDefaults{
			Ephemeral:     false,
			Reusable:      false,
			Preauthorized: true,
			ExpiryDays:    7,
			Tags:          []string{},
		},
	}
}

// validateConfig checks if the config is valid
func validateConfig(config *models.Config) error {
	// At least one auth method must be configured
	hasAPIKey := config.APIKey.PassPathAPIKey != ""
	hasOAuth := config.OAuth.PassPathClientID != "" && config.OAuth.PassPathClientSecret != ""

	if !hasAPIKey && !hasOAuth {
		return fmt.Errorf("at least one authentication method must be configured (API key or OAuth)")
	}

	if config.AuthKeyDefaults.ExpiryDays < 1 || config.AuthKeyDefaults.ExpiryDays > 90 {
		return fmt.Errorf("auth_key_defaults.expiry_days must be between 1 and 90")
	}

	// Validate tag format if tags are provided
	for _, tag := range config.AuthKeyDefaults.Tags {
		if len(tag) < 5 || tag[:4] != "tag:" {
			return fmt.Errorf("invalid tag format '%s': tags must start with 'tag:'", tag)
		}
	}

	return nil
}

// ConfigExists checks if a config file exists at the given path
func ConfigExists(configPath string) bool {
	_, err := os.Stat(configPath)
	return err == nil
}
