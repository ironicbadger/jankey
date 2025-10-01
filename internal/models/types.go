package models

import "time"

// Config represents the application configuration
type Config struct {
	APIKey           APIKeyConfig     `yaml:"api_key"`
	OAuth            OAuthConfig      `yaml:"oauth"`
	AuthKeyDefaults  AuthKeyDefaults  `yaml:"auth_key_defaults"`
}

// APIKeyConfig holds API key settings
type APIKeyConfig struct {
	PassPathAPIKey string `yaml:"pass_path_api_key"`
}

// OAuthConfig holds OAuth client credential paths
type OAuthConfig struct {
	PassPathClientID     string `yaml:"pass_path_client_id"`
	PassPathClientSecret string `yaml:"pass_path_client_secret"`
}

// AuthKeyDefaults holds default settings for auth key generation
type AuthKeyDefaults struct {
	Ephemeral      bool     `yaml:"ephemeral"`
	Reusable       bool     `yaml:"reusable"`
	Preauthorized  bool     `yaml:"preauthorized"`
	ExpiryDays     int      `yaml:"expiry_days"`
	Tags           []string `yaml:"tags"`
}

// OAuthTokenResponse represents the OAuth token response from Tailscale
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// AuthKeyRequest represents the request to create an auth key
type AuthKeyRequest struct {
	Capabilities   Capabilities `json:"capabilities"`
	ExpirySeconds  int64        `json:"expirySeconds"`
	Description    string       `json:"description,omitempty"`
}

// Capabilities defines the auth key capabilities
type Capabilities struct {
	Devices DeviceCapabilities `json:"devices"`
}

// DeviceCapabilities defines device creation capabilities
type DeviceCapabilities struct {
	Create DeviceCreateCapabilities `json:"create"`
}

// DeviceCreateCapabilities defines the settings for device creation
type DeviceCreateCapabilities struct {
	Reusable      bool     `json:"reusable"`
	Ephemeral     bool     `json:"ephemeral"`
	Preauthorized bool     `json:"preauthorized"`
	Tags          []string `json:"tags"`
}

// AuthKeyResponse represents the response from Tailscale when creating an auth key
type AuthKeyResponse struct {
	ID           string       `json:"id"`
	Key          string       `json:"key"`
	Created      time.Time    `json:"created"`
	Expires      time.Time    `json:"expires"`
	Capabilities Capabilities `json:"capabilities"`
}

// AuthKeyOutput represents the JSON output format
type AuthKeyOutput struct {
	Key          string                   `json:"key"`
	ID           string                   `json:"id"`
	Created      string                   `json:"created"`
	Expires      string                   `json:"expires"`
	Capabilities AuthKeyOutputCapabilities `json:"capabilities"`
	Tags         []string                 `json:"tags"`
}

// AuthKeyOutputCapabilities simplified capabilities for output
type AuthKeyOutputCapabilities struct {
	Ephemeral     bool `json:"ephemeral"`
	Reusable      bool `json:"reusable"`
	Preauthorized bool `json:"preauthorized"`
}
