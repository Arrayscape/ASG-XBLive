package xblive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// Minecraft API endpoints
	minecraftAuthEndpoint         = "https://api.minecraftservices.com/authentication/login_with_xbox"
	minecraftProfileEndpoint      = "https://api.minecraftservices.com/minecraft/profile"
	minecraftEntitlementsEndpoint = "https://api.minecraftservices.com/entitlements/mcstore"

	// Relying party for Minecraft XSTS token
	minecraftRelyingParty = "rp://api.minecraftservices.com/"
)

// getXSTSTokenForMinecraft exchanges the Xbox user token for an XSTS token
// using the Minecraft relying party
func (c *Client) getXSTSTokenForMinecraft(ctx context.Context, userToken string) (*XSTSTokenResponse, error) {
	reqBody := XSTSTokenRequest{
		RelyingParty: minecraftRelyingParty,
		TokenType:    "JWT",
		Properties: XSTSTokenRequestProperties{
			UserTokens: []string{userToken},
			SandboxId:  "RETAIL",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", xstsAuthEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-xbl-contract-version", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		// Try to parse Xbox error response
		var xboxErr XboxErrorResponse
		if err := json.Unmarshal(body, &xboxErr); err == nil && xboxErr.XErr != 0 {
			return nil, formatXboxError(xboxErr)
		}

		return nil, fmt.Errorf("XSTS token request (Minecraft) failed: %s - %s", resp.Status, string(body))
	}

	var xstsToken XSTSTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&xstsToken); err != nil {
		return nil, err
	}

	return &xstsToken, nil
}

// ensureXSTSTokenForMinecraft ensures we have a valid XSTS token for Minecraft, refreshing if necessary
func (c *Client) ensureXSTSTokenForMinecraft(ctx context.Context) (string, string, error) {
	// Check if we have a valid cached Minecraft XSTS token
	if token, userHash, ok := c.cache.GetMinecraftXSTSToken(ctx); ok {
		return token, userHash, nil
	}

	// Check if we have a valid cached user token
	if userToken, ok := c.cache.GetUserToken(ctx); ok {
		// Exchange for Minecraft XSTS token
		xstsResp, err := c.getXSTSTokenForMinecraft(ctx, userToken)
		if err == nil {
			userHash := extractUserHash(xstsResp.DisplayClaims)
			if err := c.cache.SetMinecraftXSTSToken(ctx, xstsResp.Token, userHash, xstsResp.NotAfter); err != nil {
				return "", "", err
			}
			return xstsResp.Token, userHash, nil
		}
	}

	// Check if we have a valid cached access token
	accessToken, ok := c.cache.GetAccessToken(ctx)
	if !ok {
		// Try to refresh
		if err := c.refreshAccessToken(ctx); err != nil {
			return "", "", fmt.Errorf("not authenticated, please call Authenticate() first")
		}
		accessToken, ok = c.cache.GetAccessToken(ctx)
		if !ok {
			return "", "", fmt.Errorf("failed to obtain access token")
		}
	}

	// Exchange access token for user token
	userTokenResp, err := c.getXboxUserToken(ctx, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user token: %w", err)
	}

	if err := c.cache.SetUserToken(ctx, userTokenResp.Token, userTokenResp.NotAfter); err != nil {
		return "", "", err
	}

	// Exchange user token for Minecraft XSTS token
	xstsResp, err := c.getXSTSTokenForMinecraft(ctx, userTokenResp.Token)
	if err != nil {
		return "", "", fmt.Errorf("failed to get XSTS token for Minecraft: %w", err)
	}

	userHash := extractUserHash(xstsResp.DisplayClaims)
	if err := c.cache.SetMinecraftXSTSToken(ctx, xstsResp.Token, userHash, xstsResp.NotAfter); err != nil {
		return "", "", err
	}

	return xstsResp.Token, userHash, nil
}

// GetMinecraftToken exchanges an XSTS token for a Minecraft access token
func (c *Client) GetMinecraftToken(ctx context.Context) (*MinecraftAuthResponse, error) {
	// Check if we have a valid cached Minecraft token
	if token, ok := c.cache.GetMinecraftToken(ctx); ok {
		return &MinecraftAuthResponse{AccessToken: token}, nil
	}

	// Get XSTS token for Minecraft
	xstsToken, userHash, err := c.ensureXSTSTokenForMinecraft(ctx)
	if err != nil {
		return nil, err
	}

	// Build identity token in required format
	identityToken := fmt.Sprintf("XBL3.0 x=%s;%s", userHash, xstsToken)

	reqBody := MinecraftAuthRequest{
		IdentityToken: identityToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", minecraftAuthEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		//lint:ignore ST1005 Minecraft is a proper name
		return nil, fmt.Errorf("Minecraft auth request failed: %s - %s", resp.Status, string(body))
	}

	var authResp MinecraftAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	// Cache the Minecraft token
	// Minecraft tokens typically expire in 86400 seconds (24 hours)
	if err := c.cache.SetMinecraftToken(ctx, authResp.AccessToken, authResp.ExpiresIn); err != nil {
		return nil, err
	}

	return &authResp, nil
}

// GetMinecraftProfile retrieves the player's Minecraft profile
func (c *Client) GetMinecraftProfile(ctx context.Context, mcToken string) (*MinecraftProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", minecraftProfileEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+mcToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no Minecraft profile found - user may not own Minecraft Java Edition")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		//lint:ignore ST1005 Minecraft is a proper name
		return nil, fmt.Errorf("Minecraft profile request failed: %s - %s", resp.Status, string(body))
	}

	var profile MinecraftProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// GetMinecraftEntitlements retrieves the player's Minecraft entitlements (game ownership info)
func (c *Client) GetMinecraftEntitlements(ctx context.Context, mcToken string) (*MinecraftEntitlements, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", minecraftEntitlementsEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+mcToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		//lint:ignore ST1005 Minecraft is a proper name
		return nil, fmt.Errorf("Minecraft entitlements request failed: %s - %s", resp.Status, string(body))
	}

	var entitlements MinecraftEntitlements
	if err := json.NewDecoder(resp.Body).Decode(&entitlements); err != nil {
		return nil, err
	}

	return &entitlements, nil
}

// GetMinecraftJavaAuth performs the complete authentication flow for Minecraft Java Edition
// Returns the Minecraft access token, profile, and entitlements
func (c *Client) GetMinecraftJavaAuth(ctx context.Context) (*MinecraftJavaAuth, error) {
	// Get Minecraft token
	authResp, err := c.GetMinecraftToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Minecraft token: %w", err)
	}

	// Get profile
	profile, err := c.GetMinecraftProfile(ctx, authResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get Minecraft profile: %w", err)
	}

	// Get entitlements
	entitlements, err := c.GetMinecraftEntitlements(ctx, authResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get Minecraft entitlements: %w", err)
	}

	return &MinecraftJavaAuth{
		Token:        authResp.AccessToken,
		Profile:      profile,
		Entitlements: entitlements,
	}, nil
}
