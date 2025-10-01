# Quick Start Guide

This guide will get you up and running with the Tailscale Auth Key Generator in 5 minutes.

## Prerequisites

- Go 1.21+ installed
- Tailscale account
- `pass` (optional, for secure credential storage)

## Step 1: Build the Tool

```bash
# Clone or navigate to the project directory
cd jankey

# Build the binary
make build

# Or use go build directly
go build -o jankey
```

## Step 2: Create OAuth Client

1. Visit: https://login.tailscale.com/admin/settings/oauth
2. Click "Generate OAuth Client"
3. Set the following:
   - **Description**: "Auth Key Generator"
   - **Scopes**: Select `devices` (or `auth_keys` if available)
4. Save your **Client ID** and **Client Secret**

## Step 3: Configure Tags in ACL

Tags are **required** by Tailscale. Add them to your ACL:

1. Visit: https://login.tailscale.com/admin/acls
2. Add tag definitions:

```json
{
  "tagOwners": {
    "tag:container": ["autogroup:admin"]
  }
}
```

3. Save your ACL

## Step 4: Run Interactive Setup

```bash
./jankey --init
```

Follow the prompts to:
- Configure credential storage (pass or environment variables)
- Set default auth key options
- Configure tags

### Option A: Using Pass (Recommended)

If you have `pass` installed, store credentials:

```bash
pass insert tailscale/oauth-client-id
# Enter your OAuth client ID

pass insert tailscale/oauth-client-secret
# Enter your OAuth client secret
```

### Option B: Using Environment Variables

```bash
export TS_OAUTH_CLIENT_ID="your-client-id"
export TS_OAUTH_CLIENT_SECRET="your-client-secret"
```

## Step 5: Generate Your First Auth Key

```bash
./jankey
```

Output:
```
tskey-auth-k5aBcDeFgH1JkLmNoPqRsTuVwXyZ
```

## Common Use Cases

### Generate Key for Docker Compose

```bash
export TS_AUTHKEY=$(./jankey --tags tag:docker)
docker compose up -d
```

### Generate Ephemeral Key

```bash
./jankey --ephemeral
```

### Generate Reusable Key

```bash
./jankey --reusable --expiry-days 30
```

### Get JSON Output

```bash
./jankey --json
```

### Verbose Mode (Debugging)

```bash
./jankey --verbose
```

## Installation

Install to system path:

```bash
make install
# Or manually:
sudo mv jankey /usr/local/bin/
```

Now you can run from anywhere:

```bash
jankey
```

## Troubleshooting

### "pass not found"

Install pass:
```bash
# macOS
brew install pass

# Ubuntu/Debian
sudo apt install pass
```

Or use environment variables instead.

### "OAuth credentials invalid"

Verify:
1. Client ID and secret are correct
2. OAuth client has `devices` scope
3. Credentials are stored correctly in pass or environment

### "tags cannot be empty"

Configure tags:
```bash
./jankey --init
```

Or specify via command line:
```bash
./jankey --tags tag:container
```

## Next Steps

- Read the full [README.md](README.md) for all features
- Check the [specification](CLAUDE.md) for implementation details
- View example config: [config.example.yaml](config.example.yaml)

## Quick Reference

```bash
# Help
./jankey --help

# Interactive setup
./jankey --init

# Basic generation
./jankey

# Custom options
./jankey --ephemeral --reusable --tags tag:docker,tag:prod

# JSON output
./jankey --json

# Verbose mode
./jankey --verbose
```

## Configuration File

Default location: `~/.config/jankey/config.yaml`

Example:
```yaml
oauth:
  pass_path_client_id: "tailscale/oauth-client-id"
  pass_path_client_secret: "tailscale/oauth-client-secret"

auth_key_defaults:
  ephemeral: false
  reusable: false
  preauthorized: true
  expiry_days: 7
  tags:
    - "tag:container"
```

## Support

- GitHub Issues: https://github.com/ironicbadger/jankey/issues
- Tailscale Docs: https://tailscale.com/kb/
