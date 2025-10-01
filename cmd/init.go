package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ironicbadger/jankey/internal/config"
	"github.com/ironicbadger/jankey/internal/pass"
)

func runInitWizard() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Jankey - Tailscale Auth Key Generator - Config Wizard        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This wizard will help you set up the configuration for generating")
	fmt.Println("Tailscale authentication keys using API keys or OAuth.")
	fmt.Println()

	// Get config path
	configPath := cfgFile
	if configPath == "" {
		var err error
		configPath, err = config.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}
	}

	// Check if config already exists
	if config.ConfigExists(configPath) {
		fmt.Printf("⚠  Config file already exists at: %s\n\n", configPath)
		if !promptYesNo(reader, "Do you want to overwrite it?", false) {
			fmt.Println("Configuration wizard cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Step 1: Choose authentication method
	fmt.Println("Step 1: Authentication Method")
	fmt.Println("─────────────────────────────────")
	fmt.Println()
	fmt.Println("Jankey supports two authentication methods:")
	fmt.Println("  1. API Key (recommended, simpler, no tags required)")
	fmt.Println("  2. OAuth (advanced, requires tags)")
	fmt.Println()
	fmt.Println("API keys expire after 90 days and must be regenerated.")
	fmt.Println("Generate API keys at: https://login.tailscale.com/admin/settings/keys")
	fmt.Println()

	useAPIKey := promptYesNo(reader, "Do you want to use API key authentication?", true)
	fmt.Println()

	// Step 2: Check pass installation
	fmt.Println("Step 2: Credential Storage")
	fmt.Println("─────────────────────────────")
	fmt.Println()

	usePass := false
	passClient, err := pass.New()
	if err != nil {
		fmt.Println("⚠  Pass (password store) is not installed or not available.")
		fmt.Println("   Pass is recommended for secure credential storage.")
		fmt.Println("   Install: https://www.passwordstore.org/")
		fmt.Println()
		if useAPIKey {
			fmt.Println("You can use the TS_API_KEY environment variable instead.")
		} else {
			fmt.Println("You can use environment variables instead:")
			fmt.Println("  - TS_OAUTH_CLIENT_ID")
			fmt.Println("  - TS_OAUTH_CLIENT_SECRET")
		}
		fmt.Println()
	} else {
		fmt.Println("✓ Pass is installed and available")
		fmt.Println()
		usePass = promptYesNo(reader, "Do you want to store credentials in pass?", true)
		fmt.Println()
	}

	cfg := config.GetDefaultConfig()

	// Step 3: Configure credentials
	if useAPIKey {
		fmt.Println("Step 3: API Key Configuration")
		fmt.Println("──────────────────────────────")
		fmt.Println()
		fmt.Println("Generate an API key at:")
		fmt.Println("  https://login.tailscale.com/admin/settings/keys")
		fmt.Println()
		fmt.Println("⚠  API keys expire after 90 days and will need to be regenerated.")
		fmt.Println()

		if usePass {
			fmt.Printf("Enter the pass path for API key [%s]: ", cfg.APIKey.PassPathAPIKey)
			apiKeyPath := readLine(reader)
			if apiKeyPath != "" {
				cfg.APIKey.PassPathAPIKey = apiKeyPath
			}

			fmt.Println()
			if promptYesNo(reader, "Do you want to store the API key in pass now?", true) {
				fmt.Print("Enter API key: ")
				apiKey := readLine(reader)

				if err := passClient.Insert(cfg.APIKey.PassPathAPIKey, apiKey); err != nil {
					fmt.Printf("Warning: failed to store API key in pass: %v\n", err)
				} else {
					fmt.Println("✓ API key stored in pass")
				}
			}
		} else {
			fmt.Println("Remember to set this environment variable:")
			fmt.Println("  export TS_API_KEY='your-api-key'")
		}
	} else {
		fmt.Println("Step 3: OAuth Client Credentials")
		fmt.Println("─────────────────────────────────────")
		fmt.Println()
		fmt.Println("You need to create an OAuth client in the Tailscale admin console:")
		fmt.Println("  https://login.tailscale.com/admin/settings/oauth")
		fmt.Println()
		fmt.Println("Required scopes:")
		fmt.Println("  - auth_keys (or devices:write)")
		fmt.Println()

		if usePass {
			fmt.Printf("Enter the pass path for OAuth client ID [%s]: ", cfg.OAuth.PassPathClientID)
			clientIDPath := readLine(reader)
			if clientIDPath != "" {
				cfg.OAuth.PassPathClientID = clientIDPath
			}

			fmt.Printf("Enter the pass path for OAuth client secret [%s]: ", cfg.OAuth.PassPathClientSecret)
			clientSecretPath := readLine(reader)
			if clientSecretPath != "" {
				cfg.OAuth.PassPathClientSecret = clientSecretPath
			}

			fmt.Println()
			if promptYesNo(reader, "Do you want to store the credentials in pass now?", true) {
				fmt.Print("Enter OAuth client ID: ")
				clientID := readLine(reader)
				fmt.Print("Enter OAuth client secret: ")
				clientSecret := readLine(reader)

				if err := passClient.Insert(cfg.OAuth.PassPathClientID, clientID); err != nil {
					fmt.Printf("Warning: failed to store client ID in pass: %v\n", err)
				} else {
					fmt.Println("✓ Client ID stored in pass")
				}

				if err := passClient.Insert(cfg.OAuth.PassPathClientSecret, clientSecret); err != nil {
					fmt.Printf("Warning: failed to store client secret in pass: %v\n", err)
				} else {
					fmt.Println("✓ Client secret stored in pass")
				}
			}
		} else {
			fmt.Println("Remember to set these environment variables:")
			fmt.Println("  export TS_OAUTH_CLIENT_ID='your-client-id'")
			fmt.Println("  export TS_OAUTH_CLIENT_SECRET='your-client-secret'")
		}
	}
	fmt.Println()

	// Step 4: Auth Key Defaults
	fmt.Println("Step 4: Auth Key Default Settings")
	fmt.Println("──────────────────────────────────────")
	fmt.Println()

	cfg.AuthKeyDefaults.Ephemeral = promptYesNo(reader, "Make keys ephemeral by default (device removed when offline)?", false)
	cfg.AuthKeyDefaults.Reusable = promptYesNo(reader, "Make keys reusable by default (can auth multiple devices)?", false)
	cfg.AuthKeyDefaults.Preauthorized = promptYesNo(reader, "Pre-authorize devices by default (skip manual approval)?", true)

	fmt.Printf("Default key expiry in days (1-90) [%d]: ", cfg.AuthKeyDefaults.ExpiryDays)
	expiryStr := readLine(reader)
	if expiryStr != "" {
		expiry, err := strconv.Atoi(expiryStr)
		if err != nil || expiry < 1 || expiry > 90 {
			fmt.Println("⚠  Invalid expiry, using default: 7 days")
		} else {
			cfg.AuthKeyDefaults.ExpiryDays = expiry
		}
	}
	fmt.Println()

	// Step 5: Tags (optional for API key, required for OAuth)
	if !useAPIKey {
		fmt.Println("Step 5: Tags (REQUIRED for OAuth)")
		fmt.Println("──────────────────────────────────")
		fmt.Println()
		fmt.Println("⚠  IMPORTANT: Tailscale requires at least one tag for OAuth-generated auth keys.")
		fmt.Println()
		fmt.Println("Tags are used for ACL (Access Control List) management and must be")
		fmt.Println("defined in your Tailscale ACL configuration first.")
		fmt.Println()
		fmt.Println("Common examples:")
		fmt.Println("  - tag:container    (for containerized applications)")
		fmt.Println("  - tag:docker       (for Docker deployments)")
		fmt.Println("  - tag:ci           (for CI/CD systems)")
		fmt.Println("  - tag:services     (for service accounts)")
		fmt.Println()
		fmt.Println("Learn more: https://tailscale.com/kb/1068/acl-tags")
		fmt.Println()

		fmt.Print("Enter default tags (comma-separated): ")
		tagsInput := readLine(reader)
		if tagsInput != "" {
			cfg.AuthKeyDefaults.Tags = parseTags(tagsInput)
		}

		// Validate tags for OAuth
		if len(cfg.AuthKeyDefaults.Tags) == 0 {
			fmt.Println("⚠  No tags specified. Using default: tag:container")
			cfg.AuthKeyDefaults.Tags = []string{"tag:container"}
		}

		fmt.Println()
		fmt.Println("Tags configured:", strings.Join(cfg.AuthKeyDefaults.Tags, ", "))
		fmt.Println()
	} else {
		fmt.Println("Step 5: Tags (Optional for API key)")
		fmt.Println("────────────────────────────────────")
		fmt.Println()
		fmt.Println("Tags are optional when using API keys.")
		fmt.Println("You can specify tags via command-line flags if needed.")
		fmt.Println()

		if promptYesNo(reader, "Do you want to configure default tags?", false) {
			fmt.Print("Enter default tags (comma-separated): ")
			tagsInput := readLine(reader)
			if tagsInput != "" {
				cfg.AuthKeyDefaults.Tags = parseTags(tagsInput)
				fmt.Println("Tags configured:", strings.Join(cfg.AuthKeyDefaults.Tags, ", "))
			}
		} else {
			cfg.AuthKeyDefaults.Tags = []string{}
		}
		fmt.Println()
	}

	// Step 6: Save configuration
	fmt.Println("Step 6: Save Configuration")
	fmt.Println("──────────────────────────")
	fmt.Println()
	fmt.Printf("Configuration will be saved to: %s\n", configPath)
	fmt.Println()

	if !promptYesNo(reader, "Save configuration?", true) {
		fmt.Println("Configuration wizard cancelled.")
		return nil
	}

	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Configuration saved successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	if useAPIKey {
		fmt.Println("  1. Ensure your API key is properly stored in pass or TS_API_KEY")
		fmt.Println("  2. Run 'jankey' to generate your first auth key")
	} else {
		fmt.Println("  1. Ensure your OAuth credentials are properly stored")
		fmt.Println("  2. Verify your Tailscale ACL includes the configured tags")
		fmt.Println("  3. Run 'jankey --use-oauth' to generate your first auth key")
	}
	fmt.Println()

	return nil
}

func promptYesNo(reader *bufio.Reader, prompt string, defaultYes bool) bool {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s [%s]: ", prompt, defaultStr)
	response := strings.ToLower(strings.TrimSpace(readLine(reader)))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
