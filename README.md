# ironicbadger/jankey

> This tool is not officially endorsed or supported by Tailscale. Use at your own risk.

A standalone CLI tool written in Go that generates Tailscale authentication keys programmatically. The tool integrates with `pass` (the standard Unix password manager) for secure credential storage and supports flexible output modes for automation workflows.

The name `jankey` is a play on words for "gen key" and "janky".

## Features

- 🔐 **Secure credential storage** via `pass` or environment variables
- 🔑 **API key authentication** (default, recommended) - simpler setup, no tags required
- 🔐 **OAuth authentication** (advanced) - requires tags, more complex setup
- 🏷️ **Tag support** (optional for API keys, required for OAuth)
- ⚙️ **Configurable defaults** via YAML configuration
- 📤 **Multiple output modes** (plain text, JSON, verbose)
- 🔄 **Automatic retry logic** with exponential backoff
- 🎯 **Interactive configuration wizard** for easy setup

## Installation

### From Source

```bash
git clone https://github.com/ironicbadger/jankey.git
cd jankey
go build -o jankey
sudo mv jankey /usr/local/bin/
```

### Prerequisites

- Go 1.21 or later
- `pass` (optional, but recommended for credential storage)
- Tailscale account with API key (or OAuth client credentials for advanced usage)

## Quick Start

### 1. Get a Tailscale API Key (Recommended)

1. Visit: https://login.tailscale.com/admin/settings/keys
2. Generate a new API key
3. Save the key securely
4. **Note**: API keys expire after 90 days and must be regenerated

### 2. Run Interactive Setup

This step is optional and command line flags may be used instead if you prefer.

```bash
jankey --init
```

This will guide you through:
- Choosing authentication method (API key or OAuth)
- Credential storage configuration
- Default auth key settings
- Tag configuration (optional for API keys)
- Configuration file creation

### 3. Generate an Auth Key automatically

```bash
jankey
```

Output:
```
tskey-auth-exampleoutput
```

The tool will place it onto your clipboard as a kind courtesy if this tool is run on macOS.

## Usage

### Basic Usage

```bash
# Generate auth key with config defaults (API key mode)
jankey

# Generate with custom tags (optional for API keys)
jankey --tags tag:docker,tag:production

# Generate ephemeral, reusable key
jankey --ephemeral --reusable

# Generate with custom expiry (14 days)
jankey --expiry-days 14

# Use OAuth instead of API key (advanced)
jankey --use-oauth --tags tag:docker

# Output as JSON
jankey --json

# Verbose output (show API interactions)
jankey --verbose
```

### Command-Line Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | | Path to config file | `~/.config/jankey/config.yaml` |
| `--init` | | Run interactive configuration wizard | - |
| `--use-oauth` | | Use OAuth instead of API key (advanced) | `false` |
| `--verbose` | `-v` | Show API interactions and debug info | `false` |
| `--json` | | Output as JSON with metadata | `false` |
| `--ephemeral` | `-e` | Make key ephemeral (device auto-removed when offline) | `false` |
| `--reusable` | `-r` | Make key reusable (can authenticate multiple devices) | `false` |
| `--preauthorized` | `-p` | Pre-authorize device (skip approval if enabled) | `true` |
| `--no-preauthorized` | | Disable pre-authorization | - |
| `--expiry-days` | | Set key expiry in days (1-90) | From config |
| `--tags` | | Comma-separated list of tags (optional for API key, REQUIRED for OAuth) | From config |
| `--description` | | Description for the auth key | Auto-generated |

### Docker Compose Integration

Generate an auth key and use it in Docker Compose:

```bash
# Using API key (default)
export TS_AUTHKEY=$(./jankey)
docker compose up -d

# Or with tags (required by OAuth)
export TS_AUTHKEY=$(./jankey --tags tag:docker)
docker compose up -d
```

Then use the env var we just created in your `docker-compose.yml`:

```yaml
services:
  app:
    image: myapp:latest

  tailscale:
    image: tailscale/tailscale:latest
    environment:
      - TS_AUTHKEY=${TS_AUTHKEY}
      - TS_STATE_DIR=/var/lib/tailscale
    volumes:
      - tailscale-state:/var/lib/tailscale

volumes:
  tailscale-state:
```

## Configuration

### Configuration File

Default location: `~/.config/jankey/config.yaml`

Example configuration:

```yaml
# API Key configuration (default auth method)
api_key:
  pass_path_api_key: "tailscale/api-key"

# OAuth configuration (use with --use-oauth flag)
oauth:
  pass_path_client_id: "tailscale/oauth-client-id"
  pass_path_client_secret: "tailscale/oauth-client-secret"

auth_key_defaults:
  ephemeral: false
  reusable: false
  preauthorized: true
  expiry_days: 7
  tags: []  # Optional for API key, required for OAuth
```

### Credential Storage

#### Option 1: Pass (Recommended)

Store credentials in `pass`:

**For API Key (default):**
```bash
pass insert tailscale/api-key
```

**For OAuth (advanced):**
```bash
pass insert tailscale/oauth-client-id
pass insert tailscale/oauth-client-secret
```

#### Option 2: Environment Variables

**For API Key (default):**
```bash
export TS_API_KEY="your-api-key"
```

**For OAuth (advanced):**
```bash
export TS_OAUTH_CLIENT_ID="your-client-id"
export TS_OAUTH_CLIENT_SECRET="your-client-secret"
```

The tool will try `pass` first, then fall back to environment variables.

## Output Modes

### Default (stdout)

```bash
$ jankey
tskey-auth-examplekey
```

Perfect for piping or variable assignment:

```bash
TS_AUTHKEY=$(jankey)
```

### JSON Output

```bash
$ jankey --json
{
  "key": "tskey-auth-examplekey",
  "id": "k5aBcD",
  "created": "2025-09-30T10:30:00Z",
  "expires": "2025-10-07T10:30:00Z",
  "capabilities": {
    "ephemeral": false,
    "reusable": false,
    "preauthorized": true
  },
  "tags": ["tag:container", "tag:docker"]
}
```

### Verbose Output

```bash
$ jankey --verbose
→ Validating Tailscale API key...
  API Key: tskey-api****xyz9
✓ API key validated

→ Creating Tailscale auth key...
  URL: https://api.tailscale.com/api/v2/tailnet/-/keys
  Request body:
  {
    "capabilities": {
      "devices": {
        "create": {
          "reusable": false,
          "ephemeral": false,
          "preauthorized": true,
          "tags": []
        }
      }
    },
    "expirySeconds": 604800,
    "description": "Generated by jankey"
  }
  Response status: 200
✓ Auth key created successfully
  ID: k5aBcD
  Expires: 2025-10-07T10:30:00Z
tskey-auth-k5aBcDeFgH1JkLmNoPqRsTuVwXyZ
```

## Error Handling

The tool provides detailed error messages with suggestions:

```bash
$ jankey
Error: API key is invalid or expired (401)

Please regenerate your API key at: https://login.tailscale.com/admin/settings/keys
Note: API keys expire after 90 days
```

Common errors:
- **API key invalid or expired**: Regenerate at https://login.tailscale.com/admin/settings/keys (API keys expire every 90 days)
- **OAuth credentials invalid** (when using `--use-oauth`): Check your client ID and secret
- **Tags not specified** (OAuth mode only): Configure tags in your config file or use `--tags`
- **Access forbidden**: Ensure proper permissions
- **Pass not installed**: Install `pass` or use environment variables

## Authentication Methods

### API Key (Default, Recommended)

**Benefits:**
- Simpler setup - no OAuth client required
- No mandatory tags requirement
- Direct API access

**Limitations:**
- API keys expire after 90 days and must be regenerated

**Setup:**
1. Visit: https://login.tailscale.com/admin/settings/keys
2. Generate a new API key
3. Store in pass: `pass insert tailscale/api-key`
4. Or use environment variable: `export TS_API_KEY="your-key"`

### OAuth (Advanced)

**Benefits:**
- Access token refreshed automatically
- More granular permissions control

**Limitations:**
- **Tags are mandatory** for OAuth-generated auth keys
- More complex setup process

**Setup:**
1. Visit: https://login.tailscale.com/admin/settings/oauth
2. Create a new OAuth client
3. Required scopes: `auth_keys` or `devices:write`
4. Define tags in your Tailscale ACL (required)
5. Store credentials in pass or environment variables
6. Use with `--use-oauth` flag

## Tag Support

Tags are **optional for API key mode** and **required for OAuth mode**.

### When to Use Tags

Tags are used for:
- Access Control List (ACL) management
- Device organization and grouping
- Permission and routing policies

### Common Tag Examples

- `tag:container` - For containerized applications
- `tag:docker` - For Docker deployments
- `tag:ci` - For CI/CD systems
- `tag:services` - For service accounts
- `tag:ephemeral` - For temporary workloads

### Setting Up Tags (OAuth mode only)

1. Define tags in your Tailscale ACL:
   ```json
   {
     "tagOwners": {
       "tag:container": ["autogroup:admin"]
     }
   }
   ```

2. Configure default tags in your config file:
   ```yaml
   auth_key_defaults:
     tags:
       - "tag:container"
   ```

3. Or specify tags via command line:
   ```bash
   jankey --use-oauth --tags tag:container,tag:docker
   ```

## API Reference

### Auth Key Creation (API Key)

```http
POST https://api.tailscale.com/api/v2/tailnet/-/keys
Authorization: Basic <base64(api_key:)>
Content-Type: application/json

{
  "capabilities": {
    "devices": {
      "create": {
        "reusable": false,
        "ephemeral": false,
        "preauthorized": true,
        "tags": []
      }
    }
  },
  "expirySeconds": 604800,
  "description": "Generated by jankey"
}
```

### OAuth Token Exchange (OAuth mode only)

```http
POST https://api.tailscale.com/api/v2/oauth/token
Content-Type: application/x-www-form-urlencoded

client_id=<client_id>&client_secret=<client_secret>&grant_type=client_credentials
```

Response:
```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### Auth Key Creation (OAuth)

```http
POST https://api.tailscale.com/api/v2/tailnet/-/keys
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "capabilities": {
    "devices": {
      "create": {
        "reusable": false,
        "ephemeral": false,
        "preauthorized": true,
        "tags": ["tag:container"]
      }
    }
  },
  "expirySeconds": 604800,
  "description": "Generated by jankey"
}
```

## Security Considerations

- ✅ API keys and OAuth credentials stored securely in `pass` (encrypted GPG)
- ✅ API keys validated before use
- ✅ OAuth access tokens are short-lived (1 hour expiry)
- ✅ Credentials never logged or printed in full (even in verbose mode)
- ✅ Generated auth keys are ephemeral (not stored by the tool)
- ✅ Automatic retry with exponential backoff for network resilience
- ⚠️ API keys expire after 90 days - set a reminder to regenerate

## Troubleshooting

### "pass not found"

Install `pass`:
```bash
# macOS
brew install pass

# Ubuntu/Debian
sudo apt install pass

# Arch
sudo pacman -S pass
```

Or use environment variables instead.

### "secret not found in pass"

Store your credentials:

**For API key (default):**
```bash
pass insert tailscale/api-key
```

**For OAuth (advanced):**
```bash
pass insert tailscale/oauth-client-id
pass insert tailscale/oauth-client-secret
```

### "API key is invalid or expired"

API keys expire after 90 days. Regenerate your key:
1. Visit https://login.tailscale.com/admin/settings/keys
2. Generate a new API key
3. Update in pass: `pass insert tailscale/api-key`

### "tags cannot be empty" (OAuth mode only)

When using OAuth (`--use-oauth`), tags are required:
```bash
jankey --use-oauth --tags tag:container
```

Or configure default tags in your config file.

### "OAuth credentials invalid" (OAuth mode only)

Verify your OAuth client:
1. Check the client ID and secret are correct
2. Ensure the OAuth client has required scopes
3. Verify the credentials are properly stored in `pass` or environment variables

## Development

### Building from Source

```bash
git clone https://github.com/ironicbadger/jankey.git
cd jankey
go mod download
go build -o jankey
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
jankey/
├── cmd/                    # CLI commands
│   ├── root.go            # Main command
│   └── init.go            # Interactive wizard
├── internal/
│   ├── config/            # Configuration management
│   ├── oauth/             # OAuth client
│   ├── pass/              # Pass integration
│   └── tailscale/         # Tailscale API client
├── pkg/
│   └── models/            # Data models
├── main.go                # Entry point
├── config.example.yaml    # Example configuration
└── README.md
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Resources

- [Tailscale API Keys](https://login.tailscale.com/admin/settings/keys) - Generate API keys (recommended)
- [Tailscale OAuth Clients](https://tailscale.com/kb/1215/oauth-clients) - OAuth setup (advanced)
- [Tailscale ACL Tags](https://tailscale.com/kb/1068/acl-tags) - Tag configuration
- [Pass Password Manager](https://www.passwordstore.org/) - Secure credential storage
- [Tailscale API Documentation](https://tailscale.com/api) - API reference

## Support

For issues and questions:
- GitHub Issues: https://github.com/ironicbadger/jankey/issues
- Tailscale Support: https://tailscale.com/contact/support
