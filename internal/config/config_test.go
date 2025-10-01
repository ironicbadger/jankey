package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ironicbadger/jankey/internal/models"
)

func TestGetDefaultConfig(t *testing.T) {
	cfg := GetDefaultConfig()

	if cfg.OAuth.PassPathClientID == "" {
		t.Error("Default config should have OAuth client ID path")
	}

	if cfg.OAuth.PassPathClientSecret == "" {
		t.Error("Default config should have OAuth client secret path")
	}

	// Tags are optional for API key mode (default), so empty is fine
	if cfg.AuthKeyDefaults.Tags == nil {
		t.Error("Default config tags should not be nil (can be empty slice)")
	}

	if cfg.AuthKeyDefaults.ExpiryDays <= 0 || cfg.AuthKeyDefaults.ExpiryDays > 90 {
		t.Errorf("Default expiry days should be between 1 and 90, got %d", cfg.AuthKeyDefaults.ExpiryDays)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *models.Config
		wantError bool
	}{
		{
			name:      "valid config",
			config:    GetDefaultConfig(),
			wantError: false,
		},
		{
			name: "missing client ID",
			config: &models.Config{
				OAuth: models.OAuthConfig{
					PassPathClientID:     "",
					PassPathClientSecret: "test/secret",
				},
				AuthKeyDefaults: models.AuthKeyDefaults{
					ExpiryDays: 7,
					Tags:       []string{"tag:test"},
				},
			},
			wantError: true,
		},
		{
			name: "valid config with API key",
			config: &models.Config{
				APIKey: models.APIKeyConfig{
					PassPathAPIKey: "test/api-key",
				},
				OAuth: models.OAuthConfig{
					PassPathClientID:     "",
					PassPathClientSecret: "",
				},
				AuthKeyDefaults: models.AuthKeyDefaults{
					ExpiryDays: 7,
					Tags:       []string{},
				},
			},
			wantError: false,
		},
		{
			name: "invalid tag format",
			config: &models.Config{
				OAuth: models.OAuthConfig{
					PassPathClientID:     "test/id",
					PassPathClientSecret: "test/secret",
				},
				AuthKeyDefaults: models.AuthKeyDefaults{
					ExpiryDays: 7,
					Tags:       []string{"invalid"},
				},
			},
			wantError: true,
		},
		{
			name: "invalid expiry days",
			config: &models.Config{
				OAuth: models.OAuthConfig{
					PassPathClientID:     "test/id",
					PassPathClientSecret: "test/secret",
				},
				AuthKeyDefaults: models.AuthKeyDefaults{
					ExpiryDays: 100,
					Tags:       []string{"tag:test"},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("validateConfig() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "jankey-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.yaml")

	// Save config
	cfg := GetDefaultConfig()
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify
	if loadedCfg.OAuth.PassPathClientID != cfg.OAuth.PassPathClientID {
		t.Errorf("Loaded config client ID = %v, want %v", loadedCfg.OAuth.PassPathClientID, cfg.OAuth.PassPathClientID)
	}

	if len(loadedCfg.AuthKeyDefaults.Tags) != len(cfg.AuthKeyDefaults.Tags) {
		t.Errorf("Loaded config tags length = %d, want %d", len(loadedCfg.AuthKeyDefaults.Tags), len(cfg.AuthKeyDefaults.Tags))
	}
}
