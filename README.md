# xblive - Xbox Live API Go Client

A Go client library for the Xbox Live API that converts gamertags to XUIDs, retrieves player profiles, and authenticates with Minecraft Java Edition.

## Features

- **OAuth 2.0 Device Code Flow**: Full authentication flow with Microsoft Entra ID
- **Token Caching**: Automatic token caching and refresh
- **Gamertag to XUID Conversion**: Convert single or multiple gamertags to XUIDs
- **Profile Lookup**: Retrieve detailed Xbox Live player profiles
- **Minecraft Java Edition Authentication**: Full auth chain for Minecraft Java Edition
- **Example CLI Tool**: Complete command-line tool demonstrating usage

## Installation

```bash
go get github.com/tadhunt/xblive
```

## Prerequisites

### Microsoft Entra ID Application

You need to register an application in Microsoft Entra ID (formerly Azure AD).

#### Basic Setup (Xbox Live API only)

1. Go to the [Azure Portal](https://portal.azure.com/)
   - Create a free Azure account if needed (requires a credit card)
2. Navigate to **Microsoft Entra ID** → **App registrations** → **New registration**
3. Register your application:
   - **Name**: Choose any name (e.g., "Xbox Live Client")
   - **Supported account types**: Select **"Personal Microsoft accounts only"**
   - **Redirect URI**: Select **"Public client/native (mobile & desktop)"** and enter `http://localhost`
4. After registration, copy the **Application (client) ID**
5. Go to **Authentication** → Enable **"Allow public client flows"** → **Save**

#### Additional Setup for Minecraft Java Edition Authentication

To use the Minecraft authentication features, your app registration needs additional configuration:

1. In your app registration, go to **Authentication**
2. Under **Advanced settings**, ensure these are configured:
   - **Allow public client flows**: Yes
   - **Enable the following mobile and desktop flows**: Yes
3. Under **Platform configurations**, click **Add a platform**:
   - Select **Mobile and desktop applications**
   - Check the box for `https://login.microsoftonline.com/common/oauth2/nativeclient`
   - Also add custom redirect URI: `http://localhost`
4. Go to **API permissions** → **Add a permission**:
   - Select **APIs my organization uses**
   - Search for **"Xbox Live"** and select **Xbox Live API**
   - Select **Delegated permissions**
   - Check **XboxLive.signin** and **XboxLive.offline_access**
   - Click **Add permissions**
5. Click **Grant admin consent** (if available) or sign in to consent on first use

#### Alternative: Use a Public Client ID

For testing purposes, you can use the Prism Launcher client ID which is already configured for Minecraft authentication:

```bash
export XBLIVE_CLIENT_ID='c36a9fb6-4f2a-41ff-90bd-ae7cc92031eb'
```

**Note**: Using third-party client IDs is fine for testing but not recommended for production applications, as you depend on their app registration remaining active.

## Quick Start

### Using the Library

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tadhunt/xblive"
)

func main() {
    // Create a new client
    client, err := xblive.New(xblive.Config{
        ClientID: "your-client-id-here",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Authenticate (first time only - tokens are cached)
    if err := client.Authenticate(ctx); err != nil {
        log.Fatal(err)
    }

    // Convert a gamertag to XUID
    xuid, err := client.GamertagToXUID(ctx, "MajorNelson")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("XUID: %s\n", xuid)
}
```

### Using the CLI Tool

```bash
# Set your client ID
export XBLIVE_CLIENT_ID='your-client-id-here'

# Authenticate with Xbox Live (one-time setup)
./xblive auth

# Authenticate with Minecraft Java Edition (requires Xbox auth first)
./xblive mc-auth

# Look up a single gamertag
./xblive lookup MajorNelson

# Get full profile for a gamertag
./xblive profile MajorNelson

# Batch lookup multiple gamertags
./xblive batch "Player1,Player2,Player3"

# Clear cached tokens (logout)
./xblive logout
```

## API Reference

### Creating a Client

```go
client, err := xblive.New(xblive.Config{
    ClientID: "your-client-id",
})
```

Creates a new Xbox Live API client with the default file-based token cache.

**Config options:**
- `ClientID` (required) - Your Microsoft Entra ID application client ID
- `Cache` (optional) - Custom `TokenCache` implementation (defaults to file-based cache at `~/.xblive/tokens.json`)

### Creating a Client with Custom Cache

```go
// Custom cache location
cache, err := xblive.NewFileTokenCacheWithPath("/custom/path/tokens.json")
if err != nil {
    log.Fatal(err)
}

client, err := xblive.New(xblive.Config{
    ClientID: "your-client-id",
    Cache:    cache,
})
```

Or implement your own `TokenCache` interface.

### Authentication

```go
err := client.Authenticate(ctx)
```

Performs the OAuth 2.0 device code flow. This will:
1. Display a URL and code for the user
2. Wait for the user to complete authentication in their browser
3. Cache the tokens for future use

Tokens are automatically refreshed when they expire.

### Gamertag to XUID

```go
xuid, err := client.GamertagToXUID(ctx, "PlayerName")
```

Converts a single gamertag to its XUID (Xbox User ID).

### Batch Gamertag Lookup

```go
xuids, err := client.GamertagsToXUIDs(ctx, []string{"Player1", "Player2"})
```

Converts multiple gamertags to XUIDs in batch. Returns a `map[string]string` where keys are gamertags and values are XUIDs.

### Minecraft Java Edition Authentication

```go
auth, err := client.GetMinecraftJavaAuth(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Username: %s\n", auth.Profile.Name)
fmt.Printf("UUID: %s\n", auth.Profile.ID)
fmt.Printf("Token: %s\n", auth.Token)
```

Performs the complete Minecraft Java Edition authentication flow and returns:
- `Token` - Minecraft access token for API calls and server authentication
- `Profile` - Player profile including username, UUID, skins, and capes
- `Entitlements` - Game ownership information

**Note**: Requires Xbox Live authentication first. Call `client.Authenticate(ctx)` before using Minecraft features.

#### Individual Minecraft API Calls

```go
// Get just the Minecraft token
tokenResp, err := client.GetMinecraftToken(ctx)

// Get player profile (requires MC token)
profile, err := client.GetMinecraftProfile(ctx, tokenResp.AccessToken)

// Check game ownership (requires MC token)
entitlements, err := client.GetMinecraftEntitlements(ctx, tokenResp.AccessToken)
```

### Clear Cache

```go
err := client.ClearCache()
```

Clears all cached authentication tokens.

## How It Works

### Xbox Live Authentication Flow

1. **Device Code Flow**: Requests a device code from Microsoft
2. **User Authentication**: User visits URL and enters code to authenticate
3. **Access Token**: Exchanges device code for Microsoft access token
4. **Xbox User Token**: Exchanges Microsoft token for Xbox user token
5. **XSTS Token**: Exchanges user token for Xbox Secure Token Service (XSTS) token
6. **API Calls**: Uses XSTS token to authenticate Xbox Live API requests

### Minecraft Java Edition Authentication Flow

Building on Xbox Live authentication:

1. **Xbox User Token**: Same as above
2. **Minecraft XSTS Token**: Exchanges user token for XSTS token with Minecraft relying party (`rp://api.minecraftservices.com/`)
3. **Minecraft Token**: Exchanges XSTS token for Minecraft access token via `api.minecraftservices.com`
4. **Profile & Entitlements**: Fetches player profile and game ownership

All tokens are cached in `~/.xblive/tokens.json` and automatically refreshed when they expire.

## Project Structure

```
xblive/
├── client.go           # Main client and public API
├── auth.go             # OAuth and Xbox token exchange logic
├── minecraft_auth.go   # Minecraft Java Edition authentication
├── minecraft_types.go  # Minecraft API types
├── types.go            # Xbox Live request/response types
├── cache.go            # Token caching
└── cmd/                # CLI tool
    └── main.go
```

## Error Handling

The library returns descriptive errors for common scenarios:

- Authentication failures
- Network errors
- Token expiration (automatically handles refresh)
- Gamertag not found
- API rate limiting

## Token Cache

### Default File-Based Cache

Tokens are stored in `~/.xblive/tokens.json` with the following structure:

- Access token (Microsoft OAuth)
- Refresh token
- Xbox user token
- XSTS token (for Xbox Live API)
- Minecraft XSTS token (for Minecraft API)
- Minecraft access token
- User hash

The cache file is created with `0600` permissions (owner read/write only) for security.

### Custom Cache Implementations

You can implement your own token cache by implementing the `TokenCache` interface:

```go
type TokenCache interface {
    // Microsoft OAuth tokens
    GetAccessToken(ctx context.Context) (string, bool)
    SetAccessToken(ctx context.Context, token string, notAfter time.Time) error
    GetRefreshToken(ctx context.Context) (string, bool)
    SetRefreshToken(ctx context.Context, token string) error

    // Xbox Live tokens
    GetUserToken(ctx context.Context) (string, bool)
    SetUserToken(ctx context.Context, token string, notAfter time.Time) error
    GetXSTSToken(ctx context.Context) (token string, userHash string, ok bool)
    SetXSTSToken(ctx context.Context, token string, userHash string, notAfter time.Time) error

    // Minecraft tokens
    GetMinecraftXSTSToken(ctx context.Context) (token string, userHash string, ok bool)
    SetMinecraftXSTSToken(ctx context.Context, token string, userHash string, notAfter time.Time) error
    GetMinecraftToken(ctx context.Context) (token string, ok bool)
    SetMinecraftToken(ctx context.Context, token string, expiresIn int) error

    Clear(ctx context.Context) error
}
```

Example use cases:
- Store tokens in a database
- Use an in-memory cache for testing
- Integrate with a secrets management system
- Implement custom encryption

## References

- [Microsoft Entra ID Device Code Flow](https://learn.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-device-code)
- [Microsoft Entra App Registration](https://aka.ms/AppRegInfo)
- [Xbox Live REST API Reference](https://learn.microsoft.com/en-us/gaming/gdk/docs/reference/live/rest/atoc-xboxlivews-reference)
- [Minecraft Services API](https://wiki.vg/Microsoft_Authentication_Scheme)
- [Converting Gamertag to XUID Guide](https://den.dev/blog/convert-gamertag-to-xuid/)

## License

MIT License

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
