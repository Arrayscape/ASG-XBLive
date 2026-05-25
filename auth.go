package xblive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// MSAL OAuth endpoints (Microsoft Entra ID / Azure AD)
	msalDeviceCodeEndpoint = "https://login.microsoftonline.com/consumers/oauth2/v2.0/devicecode"
	msalTokenEndpoint      = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

	// Live OAuth endpoints (Xbox Live / login.live.com)
	liveDeviceCodeEndpoint = "https://login.live.com/oauth20_connect.srf"
	liveTokenEndpoint      = "https://login.live.com/oauth20_token.srf"

	// Xbox endpoints
	userAuthEndpoint = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuthEndpoint = "https://xsts.auth.xboxlive.com/xsts/authorize"

	// OAuth scopes
	msalScopes = "Xboxlive.signin Xboxlive.offline_access"
	liveScopes = "service::user.auth.xboxlive.com::MBI_SSL"
)

// getDeviceCodeEndpoint returns the device code endpoint for the configured auth flow
func (c *Client) getDeviceCodeEndpoint() string {
	if c.authFlow == AuthFlowLive {
		return liveDeviceCodeEndpoint
	}
	return msalDeviceCodeEndpoint
}

// getTokenEndpoint returns the token endpoint for the configured auth flow
func (c *Client) getTokenEndpoint() string {
	if c.authFlow == AuthFlowLive {
		return liveTokenEndpoint
	}
	return msalTokenEndpoint
}

// getScopes returns the OAuth scopes for the configured auth flow
func (c *Client) getScopes() string {
	if c.authFlow == AuthFlowLive {
		return liveScopes
	}
	return msalScopes
}

// authenticateDeviceCode performs the device code OAuth flow inline, blocking
// until the user completes the flow in their browser (or the device code
// expires). Built on top of the public StartDeviceCodeFlow + PollDeviceCode
// primitives so blocking and stateless callers share the same wire protocol.
func (c *Client) authenticateDeviceCode(ctx context.Context) error {
	deviceCode, err := c.StartDeviceCodeFlow(ctx)
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}

	// Notify caller of device code via callback or print to stdout
	if c.deviceCodeCallback != nil {
		c.deviceCodeCallback(*deviceCode)
	} else {
		fmt.Printf("\n")
		fmt.Printf("To sign in, use a web browser to open the page:\n")
		fmt.Printf("    %s\n", deviceCode.VerificationURI)
		fmt.Printf("\n")
		fmt.Printf("And enter the code:\n")
		fmt.Printf("    %s\n", deviceCode.UserCode)
		fmt.Printf("\n")
	}

	// Poll until the user completes the flow upstream, the code expires, or
	// the user declines. Honors the server-supplied Interval (with a small
	// floor to defend against a zero/negative value).
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("device code expired")
			}
			result, err := c.PollDeviceCode(ctx, deviceCode.DeviceCode)
			if err != nil {
				return fmt.Errorf("poll device code: %w", err)
			}
			switch result {
			case PollResultSuccess:
				if c.deviceCodeCallback == nil {
					fmt.Printf("Authentication successful!\n\n")
				}
				return nil
			case PollResultPending:
				// Keep polling.
			case PollResultExpired:
				return fmt.Errorf("device code expired")
			case PollResultDeclined:
				return fmt.Errorf("user declined authorization")
			default:
				return fmt.Errorf("unexpected poll result %v", result)
			}
		}
	}
}

// requestDeviceCode requests a device code from Microsoft
func (c *Client) requestDeviceCode(ctx context.Context) (*DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("scope", c.getScopes())
	if c.authFlow == AuthFlowLive {
		data.Set("response_type", "device_code")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.getDeviceCodeEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code request failed: %s - %s", resp.Status, string(body))
	}

	var deviceCode DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCode); err != nil {
		return nil, err
	}

	return &deviceCode, nil
}

// StartDeviceCodeFlow requests a fresh device code from Microsoft and returns
// it. The caller is responsible for displaying UserCode + VerificationURI to
// the user, and for driving PollDeviceCode until completion. No state is
// retained on the Client between calls — the returned DeviceCodeResponse
// carries everything the caller needs to persist across requests (the
// stateless, multi-replica-safe usage pattern).
//
// To continue using the blocking flow, call Authenticate() instead; it's
// built on top of this primitive.
func (c *Client) StartDeviceCodeFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	return c.requestDeviceCode(ctx)
}

// PollResult is the outcome of a single PollDeviceCode call. The terminal
// values are Success, Expired, and Declined; Pending means the caller should
// wait (at least DeviceCodeResponse.Interval seconds) and try again.
type PollResult int

const (
	// PollResultPending: the user hasn't completed the flow upstream yet.
	// Wait and poll again. Returned for the OAuth "authorization_pending"
	// and "slow_down" error codes (caller may want to increase its poll
	// interval on slow_down).
	PollResultPending PollResult = iota

	// PollResultSuccess: the user completed the flow and the access/refresh
	// tokens have been written to the client's cache. Caller can proceed
	// with the usual token-derivation flow (GetMinecraftJavaAuth, etc.).
	PollResultSuccess

	// PollResultExpired: the device code expired before the user completed
	// the flow. Caller must restart with a new StartDeviceCodeFlow.
	PollResultExpired

	// PollResultDeclined: the user explicitly denied consent at the
	// verification URI. Caller must restart with a new StartDeviceCodeFlow.
	PollResultDeclined
)

// String returns a human-readable name for the result. Useful for logs.
func (r PollResult) String() string {
	switch r {
	case PollResultPending:
		return "pending"
	case PollResultSuccess:
		return "success"
	case PollResultExpired:
		return "expired"
	case PollResultDeclined:
		return "declined"
	default:
		return fmt.Sprintf("PollResult(%d)", int(r))
	}
}

// PollDeviceCode attempts a single token exchange for the given device code.
// Idempotent and stateless — safe to call across requests or processes (the
// device code is the only context needed). On PollResultSuccess, the access
// and refresh tokens are persisted to the client's cache.
//
// Transient transport errors (network, parse) are returned as a non-nil
// error and the caller should retry. OAuth-level negative responses are
// surfaced via the PollResult enum, not as errors.
func (c *Client) PollDeviceCode(ctx context.Context, deviceCode string) (PollResult, error) {
	token, oauthErr, err := c.tryGetTokenOAuth(ctx, deviceCode)
	if err != nil {
		return PollResultPending, err
	}
	if oauthErr != "" {
		switch oauthErr {
		case "authorization_pending", "slow_down":
			return PollResultPending, nil
		case "expired_token", "code_expired":
			return PollResultExpired, nil
		case "access_denied", "authorization_declined":
			return PollResultDeclined, nil
		default:
			// Unknown OAuth error — bubble as transport error so the caller
			// can decide whether to retry or surface to the operator.
			return PollResultPending, fmt.Errorf("oauth error: %s", oauthErr)
		}
	}

	// Success. Persist tokens for the rest of the xblive flow.
	notAfter := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	if err := c.cache.SetAccessToken(ctx, token.AccessToken, notAfter); err != nil {
		return PollResultPending, fmt.Errorf("cache access token: %w", err)
	}
	if err := c.cache.SetRefreshToken(ctx, token.RefreshToken); err != nil {
		return PollResultPending, fmt.Errorf("cache refresh token: %w", err)
	}
	return PollResultSuccess, nil
}

// tryGetTokenOAuth attempts to exchange a device code for an access token.
//
// Returns one of three states (only one is set):
//   - token != nil, oauthErr == "", err == nil           → success
//   - token == nil, oauthErr != "", err == nil           → OAuth-level negative
//     response (e.g. "authorization_pending", "expired_token", "access_denied").
//     Caller maps these to PollResult values.
//   - token == nil, oauthErr == "", err != nil           → transport/parse failure.
//     Caller should treat as transient and retry.
//
// Separating oauthErr from err lets callers distinguish "the upstream
// protocol said no" from "the request never reached the upstream" without
// string-matching error messages.
func (c *Client) tryGetTokenOAuth(ctx context.Context, deviceCode string) (*TokenResponse, string, error) {
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	tokenEndpoint := c.getTokenEndpoint()
	// Live flow requires client_id as query parameter too
	if c.authFlow == AuthFlowLive {
		tokenEndpoint = tokenEndpoint + "?client_id=" + c.clientID
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
			return nil, errorResp.Error, nil
		}
		return nil, "", fmt.Errorf("token request failed: %s - %s", resp.Status, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, "", err
	}
	return &token, "", nil
}

// refreshAccessToken refreshes the access token using the refresh token
func (c *Client) refreshAccessToken(ctx context.Context) error {
	refreshToken, ok := c.cache.GetRefreshToken(ctx)
	if !ok {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.clientID)
	data.Set("refresh_token", refreshToken)
	data.Set("scope", c.getScopes())

	req, err := http.NewRequestWithContext(ctx, "POST", c.getTokenEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return err
	}

	// Cache the new tokens
	notAfter := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	if err := c.cache.SetAccessToken(ctx, token.AccessToken, notAfter); err != nil {
		return err
	}
	if token.RefreshToken != "" {
		if err := c.cache.SetRefreshToken(ctx, token.RefreshToken); err != nil {
			return err
		}
	}

	return nil
}

// getXboxUserToken exchanges the Microsoft access token for an Xbox user token
func (c *Client) getXboxUserToken(ctx context.Context, accessToken string) (*XboxUserTokenResponse, error) {
	// Live flow uses "t=" prefix, MSAL uses "d=" prefix
	rpsTicketPrefix := "d="
	if c.authFlow == AuthFlowLive {
		rpsTicketPrefix = "t="
	}

	reqBody := XboxUserTokenRequest{
		RelyingParty: "http://auth.xboxlive.com",
		TokenType:    "JWT",
		Properties: XboxUserTokenRequestProperties{
			AuthMethod: "RPS",
			SiteName:   "user.auth.xboxlive.com",
			RpsTicket:  rpsTicketPrefix + accessToken,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", userAuthEndpoint, bytes.NewBuffer(jsonData))
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
		return nil, fmt.Errorf("user token request failed: %s - %s", resp.Status, string(body))
	}

	var userToken XboxUserTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&userToken); err != nil {
		return nil, err
	}

	return &userToken, nil
}

// getXSTSToken exchanges the Xbox user token for an XSTS token
func (c *Client) getXSTSToken(ctx context.Context, userToken string) (*XSTSTokenResponse, error) {
	reqBody := XSTSTokenRequest{
		RelyingParty: "http://xboxlive.com",
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

		return nil, fmt.Errorf("XSTS token request failed: %s - %s", resp.Status, string(body))
	}

	var xstsToken XSTSTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&xstsToken); err != nil {
		return nil, err
	}

	return &xstsToken, nil
}

// formatXboxError formats an Xbox error response into a user-friendly message
func formatXboxError(err XboxErrorResponse) error {
	switch err.XErr {
	case 2148916233: // 0x8015DC0B
		return fmt.Errorf("no Xbox account found: the Microsoft account you authenticated with doesn't have an Xbox Live profile. Create one at https://www.xbox.com/")
	case 2148916235: // 0x8015DC0D
		//lint:ignore ST1005 Xbox Live is a proper name
		return fmt.Errorf("Xbox Live is not available in your country/region")
	case 2148916236, 2148916237: // 0x8015DC0E, 0x8015DC0F
		return fmt.Errorf("the account needs adult verification. Please verify your account at https://account.microsoft.com/")
	case 2148916238: // 0x8015DC10
		return fmt.Errorf("the account is a child account and cannot proceed unless the parent consents")
	default:
		if err.Message != "" {
			//lint:ignore ST1005 Xbox is a proper name
			return fmt.Errorf("Xbox error %d: %s", err.XErr, err.Message)
		}
		//lint:ignore ST1005 Xbox is a proper name
		return fmt.Errorf("Xbox error code: %d (0x%X)", err.XErr, err.XErr)
	}
}

// ensureXSTSToken ensures we have a valid XSTS token, refreshing if necessary
func (c *Client) ensureXSTSToken(ctx context.Context) (string, string, error) {
	// Check if we have a valid cached XSTS token
	if token, userHash, ok := c.cache.GetXSTSToken(ctx); ok {
		return token, userHash, nil
	}

	// Check if we have a valid cached user token
	if userToken, ok := c.cache.GetUserToken(ctx); ok {
		// Exchange for XSTS token
		xstsResp, err := c.getXSTSToken(ctx, userToken)
		if err == nil {
			userHash := extractUserHash(xstsResp.DisplayClaims)
			if err := c.cache.SetXSTSToken(ctx, xstsResp.Token, userHash, xstsResp.NotAfter); err != nil {
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

	// Exchange user token for XSTS token
	xstsResp, err := c.getXSTSToken(ctx, userTokenResp.Token)
	if err != nil {
		return "", "", fmt.Errorf("failed to get XSTS token: %w", err)
	}

	userHash := extractUserHash(xstsResp.DisplayClaims)
	if err := c.cache.SetXSTSToken(ctx, xstsResp.Token, userHash, xstsResp.NotAfter); err != nil {
		return "", "", err
	}

	return xstsResp.Token, userHash, nil
}

// extractUserHash extracts the user hash from display claims
func extractUserHash(claims XSTSTokenDisplayClaims) string {
	if len(claims.Xui) > 0 {
		if uhs, ok := claims.Xui[0]["uhs"].(string); ok {
			return uhs
		}
	}
	return ""
}
