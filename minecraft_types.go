package xblive

// MinecraftAuthRequest is the request body for login_with_xbox
type MinecraftAuthRequest struct {
	IdentityToken string `json:"identityToken"` // XBL3.0 x={userHash};{xstsToken}
}

// MinecraftAuthResponse is the Minecraft access token response
type MinecraftAuthResponse struct {
	Username    string   `json:"username"` // UUID without dashes
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	Roles       []string `json:"roles"`
}

// MinecraftProfile is a player profile from the Minecraft API
type MinecraftProfile struct {
	ID             string          `json:"id"`   // UUID without dashes
	Name           string          `json:"name"` // In-game username
	Skins          []MinecraftSkin `json:"skins"`
	Capes          []MinecraftCape `json:"capes"`
	ProfileActions map[string]any  `json:"profileActions"`
}

// MinecraftSkin represents a player skin
type MinecraftSkin struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	URL        string `json:"url"`
	TextureKey string `json:"textureKey"`
	Variant    string `json:"variant"`
}

// MinecraftCape represents a player cape
type MinecraftCape struct {
	ID    string `json:"id"`
	State string `json:"state"`
	URL   string `json:"url"`
	Alias string `json:"alias"`
}

// MinecraftEntitlements contains game ownership information
type MinecraftEntitlements struct {
	Items []MinecraftEntitlement `json:"items"`
}

// MinecraftEntitlement represents a single entitlement
type MinecraftEntitlement struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// MinecraftJavaAuth contains everything needed for Minecraft Java authentication
type MinecraftJavaAuth struct {
	Token        string
	Profile      *MinecraftProfile
	Entitlements *MinecraftEntitlements
}
