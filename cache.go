package xblive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TokenCache manages cached authentication tokens
type TokenCache struct {
	filePath string
	tokens   *CachedTokens
}

// NewTokenCache creates a new token cache
func NewTokenCache() (*TokenCache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".xblive")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	filePath := filepath.Join(cacheDir, "tokens.json")
	cache := &TokenCache{
		filePath: filePath,
		tokens:   &CachedTokens{},
	}

	// Try to load existing tokens
	_ = cache.load()

	return cache, nil
}

// load reads tokens from disk
func (c *TokenCache) load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cached tokens yet
		}
		return fmt.Errorf("failed to read token cache: %w", err)
	}

	if err := json.Unmarshal(data, c.tokens); err != nil {
		return fmt.Errorf("failed to parse token cache: %w", err)
	}

	return nil
}

// save writes tokens to disk
func (c *TokenCache) save() error {
	data, err := json.MarshalIndent(c.tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	if err := os.WriteFile(c.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token cache: %w", err)
	}

	return nil
}

// GetAccessToken returns the cached access token if valid
func (c *TokenCache) GetAccessToken() (string, bool) {
	if c.tokens.AccessToken == "" {
		return "", false
	}
	if time.Now().After(c.tokens.AccessTokenExpiry) {
		return "", false
	}
	return c.tokens.AccessToken, true
}

// GetRefreshToken returns the cached refresh token
func (c *TokenCache) GetRefreshToken() (string, bool) {
	if c.tokens.RefreshToken == "" {
		return "", false
	}
	return c.tokens.RefreshToken, true
}

// GetUserToken returns the cached user token if valid
func (c *TokenCache) GetUserToken() (string, bool) {
	if c.tokens.UserToken == "" {
		return "", false
	}
	if time.Now().After(c.tokens.UserTokenExpiry) {
		return "", false
	}
	return c.tokens.UserToken, true
}

// GetXSTSToken returns the cached XSTS token and user hash if valid
func (c *TokenCache) GetXSTSToken() (token string, userHash string, ok bool) {
	if c.tokens.XSTSToken == "" || c.tokens.UserHash == "" {
		return "", "", false
	}
	if time.Now().After(c.tokens.XSTSTokenExpiry) {
		return "", "", false
	}
	return c.tokens.XSTSToken, c.tokens.UserHash, true
}

// SetAccessToken stores the access token
func (c *TokenCache) SetAccessToken(token string, expiresIn int) error {
	c.tokens.AccessToken = token
	c.tokens.AccessTokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return c.save()
}

// SetRefreshToken stores the refresh token
func (c *TokenCache) SetRefreshToken(token string) error {
	c.tokens.RefreshToken = token
	return c.save()
}

// SetUserToken stores the user token
func (c *TokenCache) SetUserToken(token string, notAfter time.Time) error {
	c.tokens.UserToken = token
	c.tokens.UserTokenExpiry = notAfter
	return c.save()
}

// SetXSTSToken stores the XSTS token and user hash
func (c *TokenCache) SetXSTSToken(token string, userHash string, notAfter time.Time) error {
	c.tokens.XSTSToken = token
	c.tokens.UserHash = userHash
	c.tokens.XSTSTokenExpiry = notAfter
	return c.save()
}

// Clear removes all cached tokens
func (c *TokenCache) Clear() error {
	c.tokens = &CachedTokens{}
	if err := os.Remove(c.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token cache: %w", err)
	}
	return nil
}
