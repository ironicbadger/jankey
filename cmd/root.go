package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ironicbadger/jankey/internal/apikey"
	"github.com/ironicbadger/jankey/internal/config"
	"github.com/ironicbadger/jankey/internal/oauth"
	"github.com/ironicbadger/jankey/internal/pass"
	"github.com/ironicbadger/jankey/internal/tailscale"
	"github.com/ironicbadger/jankey/pkg/models"
)

var (
	// Flags
	cfgFile     string
	verbose     bool
	jsonOutput  bool
	ephemeral   bool
	reusable    bool
	expiryDays  int
	tags        string
	description string
	initConfig  bool
	useOAuth    bool
)

var rootCmd = &cobra.Command{
	Use:   "jankey",
	Short: "Generate Tailscale authentication keys",
	Long: `Jankey - A CLI tool to generate Tailscale authentication keys programmatically
using API keys (default) or OAuth client credentials stored securely in pass.

The tool supports multiple output modes and can be configured via a YAML
configuration file or command-line flags.`,
	RunE: runGenerate,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initializeConfig)

	// Persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/jankey/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show API interactions and debug info")

	// Command flags
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "run interactive configuration wizard")
	rootCmd.Flags().BoolVar(&useOAuth, "use-oauth", false, "use OAuth authentication instead of API key")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output as JSON with metadata")
	rootCmd.Flags().BoolVarP(&ephemeral, "ephemeral", "e", false, "make key ephemeral (device auto-removed when offline)")
	rootCmd.Flags().BoolVarP(&reusable, "reusable", "r", false, "make key reusable (can authenticate multiple devices)")
	rootCmd.Flags().BoolP("preauthorized", "p", true, "pre-authorize device (skip approval if enabled)")
	rootCmd.Flags().Bool("no-preauthorized", false, "disable pre-authorization")
	rootCmd.Flags().IntVar(&expiryDays, "expiry-days", 0, "set key expiry in days (1-90, 0 for config default)")
	rootCmd.Flags().StringVar(&tags, "tags", "", "comma-separated list of tags (overrides config, required for OAuth)")
	rootCmd.Flags().StringVar(&description, "description", "", "description for the auth key")
}

func initializeConfig() {
	// This is called before command execution
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// If --init flag is set, run interactive wizard
	if initConfig {
		return runInitWizard()
	}

	// Get config path
	configPath := cfgFile
	if configPath == "" {
		var err error
		configPath, err = config.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}
	}

	// Load configuration
	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize pass client
	var passClient *pass.Client
	if pass.IsInstalled() {
		passClient, err = pass.New()
		if err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	var authKeyResp *models.AuthKeyResponse

	// Choose authentication method
	if useOAuth {
		// Use OAuth authentication
		authKeyResp, err = generateWithOAuth(cmd, cfg, passClient)
		if err != nil {
			return err
		}
	} else {
		// Default: Use API key authentication
		authKeyResp, err = generateWithAPIKey(cmd, cfg, passClient)
		if err != nil {
			return err
		}
	}

	// Output result
	return outputAuthKey(authKeyResp)
}

func generateWithAPIKey(cmd *cobra.Command, cfg *models.Config, passClient *pass.Client) (*models.AuthKeyResponse, error) {
	// Get API key
	apiKeyValue, err := pass.GetFromPassOrEnv(passClient, cfg.APIKey.PassPathAPIKey, "TS_API_KEY")
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w\n\nRun with --init to configure credentials or set TS_API_KEY environment variable", err)
	}

	// Create API key client
	apiClient := apikey.New(apiKeyValue, verbose)

	// Validate API key
	if err := apiClient.ValidateAPIKey(); err != nil {
		return nil, fmt.Errorf("API key validation failed: %w", err)
	}

	// Build auth key options
	opts := buildAPIKeyOptions(cmd, cfg)

	// Generate auth key
	authKeyResp, err := apiClient.CreateAuthKey(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth key: %w", err)
	}

	return authKeyResp, nil
}

func generateWithOAuth(cmd *cobra.Command, cfg *models.Config, passClient *pass.Client) (*models.AuthKeyResponse, error) {
	// Get OAuth credentials
	clientID, err := pass.GetFromPassOrEnv(passClient, cfg.OAuth.PassPathClientID, "TS_OAUTH_CLIENT_ID")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth client ID: %w\n\nRun with --init to configure credentials", err)
	}

	clientSecret, err := pass.GetFromPassOrEnv(passClient, cfg.OAuth.PassPathClientSecret, "TS_OAUTH_CLIENT_SECRET")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth client secret: %w\n\nRun with --init to configure credentials", err)
	}

	// Get OAuth access token
	oauthClient := oauth.New(clientID, clientSecret, verbose)
	accessToken, err := oauthClient.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth access token: %w", err)
	}

	// Build auth key options (OAuth requires tags)
	opts := buildOAuthOptions(cmd, cfg)

	// Create Tailscale client and generate auth key
	tsClient := tailscale.New(accessToken, verbose)
	authKeyResp, err := tsClient.CreateAuthKey(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth key: %w", err)
	}

	return authKeyResp, nil
}

func buildAPIKeyOptions(cmd *cobra.Command, cfg *models.Config) apikey.AuthKeyOptions {
	opts := apikey.AuthKeyOptions{
		Ephemeral:     cfg.AuthKeyDefaults.Ephemeral,
		Reusable:      cfg.AuthKeyDefaults.Reusable,
		Preauthorized: cfg.AuthKeyDefaults.Preauthorized,
		ExpiryDays:    cfg.AuthKeyDefaults.ExpiryDays,
		Tags:          cfg.AuthKeyDefaults.Tags,
		Description:   "Generated by jankey",
	}

	// Override with command-line flags
	if ephemeral {
		opts.Ephemeral = true
	}

	if reusable {
		opts.Reusable = true
	}

	// Handle preauthorized flag
	if cmd.Flags().Changed("preauthorized") {
		opts.Preauthorized = true
	}
	if cmd.Flags().Changed("no-preauthorized") {
		opts.Preauthorized = false
	}

	if expiryDays > 0 {
		opts.ExpiryDays = expiryDays
	}

	if tags != "" {
		opts.Tags = parseTags(tags)
	}

	if description != "" {
		opts.Description = description
	}

	return opts
}

func buildOAuthOptions(cmd *cobra.Command, cfg *models.Config) tailscale.AuthKeyOptions {
	opts := tailscale.AuthKeyOptions{
		Ephemeral:     cfg.AuthKeyDefaults.Ephemeral,
		Reusable:      cfg.AuthKeyDefaults.Reusable,
		Preauthorized: cfg.AuthKeyDefaults.Preauthorized,
		ExpiryDays:    cfg.AuthKeyDefaults.ExpiryDays,
		Tags:          cfg.AuthKeyDefaults.Tags,
		Description:   "Generated by jankey",
	}

	// OAuth requires tags - ensure we have at least one
	if len(opts.Tags) == 0 {
		opts.Tags = []string{"tag:container"}
	}

	// Override with command-line flags
	if ephemeral {
		opts.Ephemeral = true
	}

	if reusable {
		opts.Reusable = true
	}

	// Handle preauthorized flag
	if cmd.Flags().Changed("preauthorized") {
		opts.Preauthorized = true
	}
	if cmd.Flags().Changed("no-preauthorized") {
		opts.Preauthorized = false
	}

	if expiryDays > 0 {
		opts.ExpiryDays = expiryDays
	}

	if tags != "" {
		opts.Tags = parseTags(tags)
	}

	if description != "" {
		opts.Description = description
	}

	return opts
}

func parseTags(tagString string) []string {
	parts := strings.Split(tagString, ",")
	result := make([]string, 0, len(parts))

	for _, tag := range parts {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			// Add "tag:" prefix if not present
			if !strings.HasPrefix(tag, "tag:") {
				tag = "tag:" + tag
			}
			result = append(result, tag)
		}
	}

	return result
}

func outputAuthKey(resp *models.AuthKeyResponse) error {
	if jsonOutput {
		output := models.AuthKeyOutput{
			Key:     resp.Key,
			ID:      resp.ID,
			Created: resp.Created.Format("2006-01-02T15:04:05Z"),
			Expires: resp.Expires.Format("2006-01-02T15:04:05Z"),
			Capabilities: models.AuthKeyOutputCapabilities{
				Ephemeral:     resp.Capabilities.Devices.Create.Ephemeral,
				Reusable:      resp.Capabilities.Devices.Create.Reusable,
				Preauthorized: resp.Capabilities.Devices.Create.Preauthorized,
			},
			Tags: resp.Capabilities.Devices.Create.Tags,
		}

		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON output: %w", err)
		}

		fmt.Println(string(jsonData))
	} else {
		// Simple stdout output - just the key
		fmt.Println(resp.Key)

		// Copy to clipboard on macOS
		if runtime.GOOS == "darwin" {
			if err := copyToClipboard(resp.Key); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to copy to clipboard: %v\n", err)
				}
			} else if verbose {
				fmt.Fprintf(os.Stderr, "Auth key copied to clipboard\n")
			}
		}
	}

	return nil
}

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := stdin.Write([]byte(text)); err != nil {
		return err
	}

	if err := stdin.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}

