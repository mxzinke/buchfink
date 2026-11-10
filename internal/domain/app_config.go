package domain

// TenantConfig represents a single company / client workspace.
type TenantConfig struct {
	ID        string `json:"id"`        // Unique identifier
	Name      string `json:"name"`      // Display name / Company Name
	DataDir   string `json:"dataDir"`   // Directory where buchfink.sqlite and /belege are stored
	CreatedAt string `json:"createdAt"` // Creation timestamp (RFC3339)

	// BackupDir ist der Zielordner der Sicherung. Leer heißt: keine Sicherung
	// eingerichtet — die Aufgabenliste weist darauf hin, statt still nichts zu
	// tun.
	BackupDir string `json:"backupDir,omitempty"`
	// LastBackupAt ist der Zeitpunkt der letzten erfolgreichen Sicherung
	// (RFC3339). Er steht hier für die Anzeige: die Einstellungen und die
	// Aufgabenliste sollen sagen können, wann zuletzt gesichert wurde, ohne
	// dafür die Läufe aus der Mandantendatenbank zu lesen. Ob eine Sicherung
	// fällig ist, entscheidet BackupService.IsDue aus den Läufen selbst.
	LastBackupAt string `json:"lastBackupAt,omitempty"`
}

// AppConfig represents global persistent application preferences across tenants and fiscal years.
type AppConfig struct {
	Tenants        []TenantConfig `json:"tenants"`
	ActiveTenantID string         `json:"activeTenantId"`
	DataDir        string         `json:"dataDir"`        // Active tenant data dir
	IsConfigured   bool           `json:"isConfigured"`   // False on initial launch -> triggers Setup Assistant
	LastFiscalYear int            `json:"lastFiscalYear"` // Last active fiscal year filter (e.g. 2026)

	// Die folgenden Felder spiegeln den aktiven Mandanten, so wie DataDir es
	// schon tut. Die Oberfläche liest einen Zustand und muss ihn nicht erst aus
	// der Mandantenliste heraussuchen. Den Prüfermodus rechnet der Kern aus der
	// Mandantendatenbank; gespeichert wird er hier nicht.
	BackupDir      string `json:"backupDir"`
	LastBackupAt   string `json:"lastBackupAt"`
	ReadOnly       bool   `json:"readOnly"`
	ReadOnlyUntil  string `json:"readOnlyUntil"`
	ReadOnlyReason string `json:"readOnlyReason"`

	// ProgramVersion ist die Fassung des laufenden Programms. Sie steht in den
	// Einstellungen und in jedem Export.
	ProgramVersion string `json:"programVersion"`

	// MCPPort ist der Port des MCP-Servers für verbundene Apps; 0 heißt
	// Vorgabe (7465). Die Apps tragen die Adresse fest ein, deshalb weicht
	// Buchfink bei belegtem Port nicht aus.
	MCPPort int `json:"mcpPort,omitempty"`
}

// EnsureLists belegt die Listen, die als JSON an die Oberfläche gehen.
//
// Eine Konfiguration ohne Mandanten ist der Zustand vor der Einrichtung — genau
// der Bildschirm, den ein neuer Anwender zuerst sieht. Ein nicht belegter Slice
// käme dort als `null` an, und `null.map` nimmt im Render den ganzen Baum mit.
func (c *AppConfig) EnsureLists() {
	if c.Tenants == nil {
		c.Tenants = []TenantConfig{}
	}
}

// ActiveTenant liefert den aktiven Mandanten oder nil.
func (c *AppConfig) ActiveTenant() *TenantConfig {
	for i := range c.Tenants {
		if c.Tenants[i].ID == c.ActiveTenantID {
			return &c.Tenants[i]
		}
	}
	return nil
}

// SyncActiveTenant zieht die Felder des aktiven Mandanten nach oben.
//
// Sie werden gespiegelt und nicht dort gelesen, wo sie stehen, weil die
// Oberfläche eine Konfiguration bekommt und keine Suche darin durchführen soll.
// Der Mandant bleibt die Quelle: geschrieben wird immer dort.
func (c *AppConfig) SyncActiveTenant() {
	t := c.ActiveTenant()
	if t == nil {
		c.BackupDir, c.LastBackupAt = "", ""
		return
	}
	c.DataDir = t.DataDir
	c.BackupDir = t.BackupDir
	c.LastBackupAt = t.LastBackupAt
}

// AppConfigRepository defines persistence for local application configuration.
type AppConfigRepository interface {
	Load() (*AppConfig, error)
	Save(cfg *AppConfig) error
}
