package xblive

import "time"

// DeviceCodeResponse represents the response from the device code flow
type DeviceCodeResponse struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// TokenResponse represents an OAuth token response
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// XboxUserTokenRequest represents a request for an Xbox user token
type XboxUserTokenRequest struct {
	RelyingParty string                         `json:"RelyingParty"`
	TokenType    string                         `json:"TokenType"`
	Properties   XboxUserTokenRequestProperties `json:"Properties"`
}

// XboxUserTokenRequestProperties contains properties for user token request
type XboxUserTokenRequestProperties struct {
	AuthMethod string `json:"AuthMethod"`
	SiteName   string `json:"SiteName"`
	RpsTicket  string `json:"RpsTicket"`
}

// XboxUserTokenResponse represents the response from user token endpoint
type XboxUserTokenResponse struct {
	IssueInstant  time.Time                  `json:"IssueInstant"`
	NotAfter      time.Time                  `json:"NotAfter"`
	Token         string                     `json:"Token"`
	DisplayClaims XboxUserTokenDisplayClaims `json:"DisplayClaims"`
}

// XboxUserTokenDisplayClaims contains the user hash
type XboxUserTokenDisplayClaims struct {
	Xui []map[string]interface{} `json:"xui"`
}

// XSTSTokenRequest represents a request for an XSTS token
type XSTSTokenRequest struct {
	RelyingParty string                     `json:"RelyingParty"`
	TokenType    string                     `json:"TokenType"`
	Properties   XSTSTokenRequestProperties `json:"Properties"`
}

// XSTSTokenRequestProperties contains properties for XSTS token request
type XSTSTokenRequestProperties struct {
	UserTokens []string `json:"UserTokens"`
	SandboxId  string   `json:"SandboxId"`
}

// XSTSTokenResponse represents the response from XSTS token endpoint
type XSTSTokenResponse struct {
	IssueInstant  time.Time              `json:"IssueInstant"`
	NotAfter      time.Time              `json:"NotAfter"`
	Token         string                 `json:"Token"`
	DisplayClaims XSTSTokenDisplayClaims `json:"DisplayClaims"`
}

// XSTSTokenDisplayClaims contains the user hash
type XSTSTokenDisplayClaims struct {
	Xui []map[string]interface{} `json:"xui"`
}

// SearchResponse represents the response from people search endpoint
type SearchResponse struct {
	People []*Profile `json:"people"`
}

// Profile represents an Xbox Live user profile
type Profile struct {
	XUID                 string `json:"xuid"`
	Gamertag             string `json:"gamertag"`
	ModernGamertag       string `json:"modernGamertag"`
	ModernGamertagSuffix string `json:"modernGamertagSuffix"`
	UniqueModernGamertag string `json:"uniqueModernGamertag"`
	GamerScore           string `json:"gamerScore"` // API returns as string
	PresenceState        string `json:"presenceState"`
	PresenceText         string `json:"presenceText"`
	IsFavorite           bool   `json:"isFavorite"`
	IsFollowingCaller    bool   `json:"isFollowingCaller"`
	IsFollowedByCaller   bool   `json:"isFollowedByCaller"`
	IsIdentityShared     bool   `json:"isIdentityShared"`
	DisplayPicRaw        string `json:"displayPicRaw"`
	RealName             string `json:"realName"`
	AccountTier          string `json:"accountTier"`
}

// CachedTokens represents cached authentication tokens
type CachedTokens struct {
	AccessToken       string    `json:"access_token"`
	RefreshToken      string    `json:"refresh_token"`
	AccessTokenExpiry time.Time `json:"access_token_expiry"`
	UserToken         string    `json:"user_token"`
	UserTokenExpiry   time.Time `json:"user_token_expiry"`
	XSTSToken         string    `json:"xsts_token"`
	XSTSTokenExpiry   time.Time `json:"xsts_token_expiry"`
	UserHash          string    `json:"user_hash"`
}

// XboxErrorResponse represents an error response from Xbox services
type XboxErrorResponse struct {
	Identity string `json:"Identity"`
	XErr     int64  `json:"XErr"`
	Message  string `json:"Message"`
	Redirect string `json:"Redirect"`
}
