package domain

// AppConfig represents global persistent application preferences across fiscal years.
type AppConfig struct {
	DataDir        string `json:"dataDir"`        // Directory where *.sqlite and /belege are stored
	CertPath       string `json:"certPath"`       // Path to the local certificate/key file
	HasPassword    bool   `json:"hasPassword"`    // Whether certificate/access key is password-protected
	IsConfigured   bool   `json:"isConfigured"`   // False on initial launch -> triggers Setup Assistant
	LastFiscalYear int    `json:"lastFiscalYear"` // Last active fiscal year (e.g. 2026)
}

// AppConfigRepository defines persistence for local application configuration.
type AppConfigRepository interface {
	Load() (*AppConfig, error)
	Save(cfg *AppConfig) error
}
