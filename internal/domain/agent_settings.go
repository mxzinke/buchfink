package domain

// AgentSettings sind die Grenzen, in denen ein verbundener Agent ohne
// Rückfrage bucht, und die Ordner, aus denen er neben dem Eingang liest und in
// die er exportiert. Nur der Mensch ändert sie in der App.
type AgentSettings struct {
	// DirectMaxGross ist die Direktgrenze: der höchste Bruttobetrag einer
	// E-Rechnung, die ein Agent ohne Rückfrage bucht. Null heißt Voreinstellung.
	DirectMaxGross Cents `json:"directMaxGross"`
	// DirectMinBookings ist die Zahl früherer Buchungen eines Lieferanten mit
	// seiner Standard-Buchungsgruppe, ab der ein Agent direkt bucht. Null heißt
	// Voreinstellung.
	DirectMinBookings int `json:"directMinBookings"`
	// ExtraReadFolders sind weitere Leseordner neben dem Eingang; leer heißt
	// keiner.
	ExtraReadFolders []string `json:"extraReadFolders"`
	// ExportFolder ist der Exportordner; leer heißt Voreinstellung (Export im
	// Datenordner).
	ExportFolder string `json:"exportFolder"`
}

// Voreinstellungen der Regeln für Agenten.
const (
	DefaultDirectMaxGross    Cents = 100000
	DefaultDirectMinBookings       = 3
)

// DefaultInboxDir ist der voreingestellte Eingang im Datenordner.
const DefaultInboxDir = "Eingang"

// Schlüssel der Einstellungen für Agenten und den Eingang.
const (
	SettingAgentDirectMaxGross    = "agent.direct_max_gross"
	SettingAgentDirectMinBookings = "agent.direct_min_bookings"
	SettingAgentExtraReadFolders  = "agent.extra_read_folders"
	SettingAgentExportFolder      = "agent.export_folder"
	// SettingInboxFolder ist der Eingang des Mandanten; leer heißt
	// <Datenordner>/Eingang.
	SettingInboxFolder = "inbox.folder"
)
