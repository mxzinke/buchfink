package domain

// TenantConfig represents a single company / client workspace. Field encryption
// is provisioned transparently: the envelope keyfile lives in DataDir and the
// wrapping secret in the OS keychain, so no key path needs to be stored here.
type TenantConfig struct {
	ID        string `json:"id"`        // Unique identifier
	Name      string `json:"name"`      // Display name / Company Name
	DataDir   string `json:"dataDir"`   // Directory where buchfink.sqlite, keyfile and /belege are stored
	CreatedAt string `json:"createdAt"` // Creation timestamp (RFC3339)

	// KeyID ist die Kennung, unter der das Geheimnis dieses Mandanten im
	// Schlüsselbund des Betriebssystems liegt. Leer heißt: dieselbe wie ID.
	//
	// Getrennt wird beides erst, wenn eine wiederhergestellte Sicherung neben
	// dem Mandanten steht, aus dem sie stammt: zwei Einträge der Mandantenliste
	// dürfen nicht dieselbe ID tragen (die Liste würde mehrdeutig), teilen sich
	// aber denselben Schlüssel — die Sicherung bringt die Schlüsseldatei des
	// Ursprungsmandanten mit, und das Geheimnis dazu liegt im Schlüsselbund
	// unter dessen Kennung.
	KeyID string `json:"keyId,omitempty"`

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

	// ReadOnlyUntil ist der letzte Tag des Prüfermodus (YYYY-MM-DD). Solange er
	// nicht vergangen ist, weist die Bridge jede schreibende Methode ab.
	//
	// Ein Datum und kein Schalter: ein Prüfermodus ohne Ende wird vergessen und
	// blockiert irgendwann die laufende Buchführung. Die Datenüberlassung nach
	// § 147 Abs. 6 AO ist auf die Prüfung befristet, der Modus also auch.
	ReadOnlyUntil string `json:"readOnlyUntil,omitempty"`
	// ReadOnlyReason ist der Grund, der beim Einschalten anzugeben ist. Er
	// steht auch im Änderungsprotokoll.
	ReadOnlyReason string `json:"readOnlyReason,omitempty"`
}

// VaultID liefert die Kennung, unter der der Schlüssel dieses Mandanten zu
// suchen ist. Jeder Zugriff auf den Schlüsselbund geht über sie und nicht über
// ID: sonst bliebe eine wiederhergestellte Sicherung gesperrt, obwohl ihr
// Geheimnis auf diesem Rechner liegt.
func (t *TenantConfig) VaultID() string {
	if t.KeyID != "" {
		return t.KeyID
	}
	return t.ID
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
	// der Mandantenliste heraussuchen.
	BackupDir      string `json:"backupDir"`
	LastBackupAt   string `json:"lastBackupAt"`
	ReadOnly       bool   `json:"readOnly"`
	ReadOnlyUntil  string `json:"readOnlyUntil"`
	ReadOnlyReason string `json:"readOnlyReason"`

	// ProgramVersion ist die Fassung des laufenden Programms. Sie steht in den
	// Einstellungen und in jedem Export.
	ProgramVersion string `json:"programVersion"`
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
func (c *AppConfig) SyncActiveTenant(today string) {
	t := c.ActiveTenant()
	if t == nil {
		c.BackupDir, c.LastBackupAt = "", ""
		c.ReadOnly, c.ReadOnlyUntil, c.ReadOnlyReason = false, "", ""
		return
	}
	c.DataDir = t.DataDir
	c.BackupDir = t.BackupDir
	c.LastBackupAt = t.LastBackupAt
	c.ReadOnlyUntil = t.ReadOnlyUntil
	c.ReadOnlyReason = t.ReadOnlyReason
	c.ReadOnly = t.ReadOnlyActiveOn(today)
}

// ReadOnlyActiveOn meldet, ob der Prüfermodus an diesem Tag noch gilt.
//
// Der letzte Tag zählt mit: „bis zum 30.06." heißt einschließlich. Ein
// abgelaufener Modus endet von selbst — sonst müsste jemand daran denken, ihn
// auszuschalten, und niemand tut das.
func (t *TenantConfig) ReadOnlyActiveOn(today string) bool {
	return t.ReadOnlyUntil != "" && today <= t.ReadOnlyUntil
}

// AppConfigRepository defines persistence for local application configuration.
type AppConfigRepository interface {
	Load() (*AppConfig, error)
	Save(cfg *AppConfig) error
}
