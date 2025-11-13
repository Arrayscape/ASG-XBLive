# xblive - Xbox Live API Go Client

A Go client library for the Xbox Live API that converts gamertags to XUIDs and retrieves player profiles.

## Features

- **OAuth 2.0 Device Code Flow**: Full authentication flow with Microsoft Entra ID
- **Token Caching**: Automatic token caching and refresh
- **Gamertag to XUID Conversion**: Convert single or multiple gamertags to XUIDs
- **Profile Lookup**: Retrieve detailed Xbox Live player profiles
- **Example CLI Tool**: Complete command-line tool demonstrating usage

## Installation

```bash
go get github.com/tadhunt/xblive
```

## Prerequisites

### Microsoft Entra ID Application

You need to register an application in Microsoft Entra ID (formerly Azure AD):

1. Go to the [Azure Portal](https://portal.azure.com/)
2. Navigate to "Microsoft Entra ID" → "App registrations" → "New registration"
3. Register your application:
   - Name: Choose any name (e.g., "Xbox Live Client")
   - Supported account types: "Personal Microsoft accounts only"
   - Redirect URI: Select "Public client/native (mobile & desktop)" and enter `http://localhost`
4. After registration, copy the **Application (client) ID**
5. Go to "Authentication" → Enable "Allow public client flows" → Save
6. Go to "API permissions" → "Add a permission" → "Xbox Live" → Add the following delegated permissions:
   - `Xboxlive.signin`
   - `Xboxlive.offline_access`
7. Grant admin consent for these permissions

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
    // Create a new client with your Microsoft Entra ID client ID
    client, err := xblive.New("your-client-id-here")
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

# Authenticate (one-time setup)
go run example/main.go auth

# Look up a single gamertag
go run example/main.go lookup MajorNelson

# Batch lookup multiple gamertags
go run example/main.go batch "Player1,Player2,Player3"

# Clear cached tokens (logout)
go run example/main.go logout
```

## API Reference

### Creating a Client

```go
client, err := xblive.New(clientID)
```

Creates a new Xbox Live API client with the default file-based token cache. The `clientID` is your Microsoft Entra ID application client ID.

### Creating a Client with Custom Cache

```go
cache, err := xblive.NewFileTokenCacheWithPath("/custom/path/tokens.json")
if err != nil {
    log.Fatal(err)
}

client, err := xblive.NewWithCache(clientID, cache)
```

Creates a client with a custom cache location or implementation.

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

### Clear Cache

```go
err := client.ClearCache()
```

Clears all cached authentication tokens.

## How It Works

The authentication flow follows these steps:

1. **Device Code Flow**: Requests a device code from Microsoft
2. **User Authentication**: User visits URL and enters code to authenticate
3. **Access Token**: Exchanges device code for Microsoft access token
4. **Xbox User Token**: Exchanges Microsoft token for Xbox user token
5. **XSTS Token**: Exchanges user token for Xbox Secure Token Service (XSTS) token
6. **API Calls**: Uses XSTS token to authenticate Xbox Live API requests

All tokens are cached in `~/.xblive/tokens.json` and automatically refreshed when they expire.

## Project Structure

```
xblive/
├── client.go       # Main client and public API
├── auth.go         # OAuth and token exchange logic
├── types.go        # Request/response types
├── cache.go        # Token caching
└── example/        # Example CLI tool
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

- Access token (Microsoft)
- Refresh token
- Xbox user token
- XSTS token
- User hash

The cache file is created with `0600` permissions (owner read/write only) for security.

### Custom Cache Implementations

You can implement your own token cache by implementing the `TokenCache` interface:

```go
type TokenCache interface {
    GetAccessToken() (string, bool)
    GetRefreshToken() (string, bool)
    GetUserToken() (string, bool)
    GetXSTSToken() (token string, userHash string, ok bool)
    SetAccessToken(token string, expiresIn int) error
    SetRefreshToken(token string) error
    SetUserToken(token string, notAfter time.Time) error
    SetXSTSToken(token string, userHash string, notAfter time.Time) error
    Clear() error
}
```

Example use cases:
- Store tokens in a database
- Use an in-memory cache for testing
- Integrate with a secrets management system
- Implement custom encryption

## References

- [Microsoft Entra ID Device Code Flow](https://learn.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-device-code)
- [Xbox Live REST API Reference](https://learn.microsoft.com/en-us/gaming/gdk/docs/reference/live/rest/atoc-xboxlivews-reference)
- [Converting Gamertag to XUID Guide](https://den.dev/blog/convert-gamertag-to-xuid/)

## License

MIT License

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
