package domain

// TenantConfig represents a single company / client workspace. Field encryption
// is provisioned transparently: the envelope keyfile lives in DataDir and the
// wrapping secret in the OS keychain, so no key path needs to be stored here.
type TenantConfig struct {
	ID        string `json:"id"`        // Unique identifier
	Name      string `json:"name"`      // Display name / Company Name
	DataDir   string `json:"dataDir"`   // Directory where buchfink.sqlite, keyfile and /belege are stored
	CreatedAt string `json:"createdAt"` // Creation timestamp (RFC3339)
}

// AppConfig represents global persistent application preferences across tenants and fiscal years.
type AppConfig struct {
	Tenants        []TenantConfig `json:"tenants"`
	ActiveTenantID string         `json:"activeTenantId"`
	DataDir        string         `json:"dataDir"`        // Active tenant data dir
	IsConfigured   bool           `json:"isConfigured"`   // False on initial launch -> triggers Setup Assistant
	LastFiscalYear int            `json:"lastFiscalYear"` // Last active fiscal year filter (e.g. 2026)
}

// AppConfigRepository defines persistence for local application configuration.
type AppConfigRepository interface {
	Load() (*AppConfig, error)
	Save(cfg *AppConfig) error
}

