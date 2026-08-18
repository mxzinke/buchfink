package domain

// TenantConfig represents a single company / client workspace.
type TenantConfig struct {
	ID          string `json:"id"`          // Unique identifier
	Name        string `json:"name"`        // Display name / Company Name
	DataDir     string `json:"dataDir"`     // Directory where buchfink.sqlite and /belege are stored
	CertPath    string `json:"certPath"`    // Path to the local certificate/key file
	HasPassword bool   `json:"hasPassword"` // Whether certificate/access key is password-protected
	CreatedAt   string `json:"createdAt"`   // Creation timestamp (RFC3339)
}

// AppConfig represents global persistent application preferences across tenants and fiscal years.
type AppConfig struct {
	Tenants        []TenantConfig `json:"tenants"`
	ActiveTenantID string         `json:"activeTenantId"`
	DataDir        string         `json:"dataDir"`        // Active tenant data dir
	CertPath       string         `json:"certPath"`       // Active tenant cert path
	HasPassword    bool           `json:"hasPassword"`    // Active tenant password flag
	IsConfigured   bool           `json:"isConfigured"`   // False on initial launch -> triggers Setup Assistant
	LastFiscalYear int            `json:"lastFiscalYear"` // Last active fiscal year filter (e.g. 2026)
}

// AppConfigRepository defines persistence for local application configuration.
type AppConfigRepository interface {
	Load() (*AppConfig, error)
	Save(cfg *AppConfig) error
}

