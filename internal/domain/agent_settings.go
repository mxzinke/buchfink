package domain

// AgentSettings sind die Grenzen, in denen ein verbundener Agent ohne
// Rückfrage bucht, und die Ordner, aus denen er Belege liest und in die er
// exportiert. Nur der Mensch ändert sie in der App.
type AgentSettings struct {
	// DirectMaxGross ist die Direktgrenze: der höchste Bruttobetrag einer
	// E-Rechnung, die ein Agent ohne Rückfrage bucht. Null heißt Voreinstellung.
	DirectMaxGross Cents `json:"directMaxGross"`
	// DirectMinBookings ist die Zahl früherer Buchungen eines Lieferanten mit
	// seiner Standard-Buchungsgruppe, ab der ein Agent direkt bucht. Null heißt
	// Voreinstellung.
	DirectMinBookings int `json:"directMinBookings"`
	// ReceiptFolders sind die Belegordner. Nil heißt Voreinstellung (Eingang
	// im Datenordner), eine leere Liste: kein Ordner.
	ReceiptFolders []string `json:"receiptFolders"`
	// ExportFolder ist der Exportordner; leer heißt Voreinstellung (Export im
	// Datenordner).
	ExportFolder string `json:"exportFolder"`
}

// Voreinstellungen der Regeln für Agenten.
const (
	DefaultDirectMaxGross    Cents = 100000
	DefaultDirectMinBookings       = 3
)

// Schlüssel der Einstellungen für Agenten.
const (
	SettingAgentDirectMaxGross    = "agent.direct_max_gross"
	SettingAgentDirectMinBookings = "agent.direct_min_bookings"
	SettingAgentReceiptFolders    = "agent.receipt_folders"
	SettingAgentExportFolder      = "agent.export_folder"
)
