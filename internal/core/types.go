package core

import "time"

// Category classifies providers for UI grouping and filtering.
type Category string

const (
	CategoryCoding       Category = "coding"       // Cursor, Copilot, Codex, Antigravity
	CategoryChat         Category = "chat"          // Claude, ChatGPT
	CategoryAPI          Category = "api"           // DeepSeek, OpenRouter
	CategoryCreative     Category = "creative"      // Midjourney, ElevenLabs
	CategoryProductivity Category = "productivity"  // Notion AI, Perplexity
	CategoryCustom       Category = "custom"        // Plugins
)

// Cap is a bitmask of provider capabilities.
type Cap uint32

const (
	CapQuota     Cap = 1 << iota // Reports remaining quota percentage
	CapBalance                   // Reports monetary balance
	CapReset                     // Reports reset timing
	CapModels                    // Reports per-model breakdown
	CapCost                      // Reports estimated cost
	CapMultiAcct                 // Supports multiple accounts
	CapTokens                    // Reports token-level usage
)

// Has reports whether the capability set includes the given capability.
func (c Cap) Has(flag Cap) bool { return c&flag != 0 }

// AuthType identifies the method of authentication.
type AuthType string

const (
	AuthAPIKey    AuthType = "api_key"     // Bearer token or API key header
	AuthOAuth     AuthType = "oauth"       // OAuth2 flow (PKCE or token file)
	AuthCookie    AuthType = "cookie"      // Browser cookie from local filesystem
	AuthCLIToken  AuthType = "cli_token"   // Token file from CLI tool
	AuthProcessRPC AuthType = "process_rpc" // Local process detection + IPC
)

// FieldType identifies the type of a configuration field.
type FieldType string

const (
	FieldString FieldType = "string"
	FieldSecret FieldType = "secret"  // Masked in UI, never synced
	FieldBool   FieldType = "bool"
	FieldInt    FieldType = "int"
	FieldSelect FieldType = "select"  // Dropdown with Options
)

// Credentials holds authentication material for a provider.
type Credentials struct {
	APIKey      string            // For AuthAPIKey
	AccessToken string            // For AuthOAuth, AuthCLIToken
	Cookie      string            // For AuthCookie
	Extra       map[string]string // Provider-specific extras
}

// AuthMethod describes one way to authenticate with a provider.
type AuthMethod struct {
	Type        AuthType  // How this method authenticates
	Priority    int       // Lower = try first
	Label       string    // "API Key", "OAuth Token", "Browser Cookie"
	ConfigKeys  []string  // Which config keys this method needs
}

// ConfigField describes one configuration setting for a provider.
// These drive auto-generation of the Settings UI.
type ConfigField struct {
	Key      string    // Config key (e.g., "deepseek_api_key")
	Label    string    // Display label (e.g., "API Key")
	Type     FieldType // Field type for validation + UI
	Required bool      // Whether this field must be set
	Default  string    // Default value (empty if none)
	EnvVar   string    // Environment variable to check (e.g., "DEEPSEEK_API_KEY")
	Hint     string    // Help text (e.g., "Get from platform.deepseek.com")
	Syncable bool      // Whether this value syncs to cloud (false for secrets)
	Options  []string  // For FieldSelect: allowed values
}

// Snapshot is the universal data structure returned by Provider.Fetch().
// Provider-specific data lives in Data (marshaled to data_json in the store).
type Snapshot struct {
	Provider   string  // Provider ID (e.g., "codex", "claude")
	AccountID  string  // FK to accounts table
	Email      string  // Account email
	OverallPct float64 // 0-100, primary health signal
	PlanTier   string  // "free", "pro", "enterprise", etc.
	CostUSD    float64 // Estimated cost since last snapshot

	ResetAt   *time.Time // Next quota reset (nil if unknown)
	ResetType string     // "5h", "7d", "monthly", "daily", "" if unknown

	Data   any // Provider-specific structured data → data_json
	Models any // Per-model breakdown → models_json

	CaptureMethod string // "auto", "manual", "import"
	CaptureSource string // "ls_poll", "oauth", "api_key", "cookie"
}
