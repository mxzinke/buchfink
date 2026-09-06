package domain

// AuditChainBreak ist ein einzelner Bruch der Protokollkette.
//
// Wie bei der Journalkette stehen erwarteter und tatsächlicher Hash daneben:
// „Eintrag 42 ist gebrochen" ist außerhalb von Buchfink nicht nachrechenbar,
// und ein Prüfer, der die Kanonisierung aus der Feldbeschreibung kennt, soll
// selbst nachrechnen können, welche Seite recht hat.
type AuditChainBreak struct {
	EntryID      uint                 `json:"entryId"`
	Reason       IntegrityBreakReason `json:"reason"`
	ExpectedHash string               `json:"expectedHash"`
	ActualHash   string               `json:"actualHash"`
	Message      string               `json:"message"`
}

// AuditChainResult ist das Ergebnis der Prüfung des Änderungsprotokolls.
//
// Das Protokoll ist eine einzige Kette und nicht — wie das Journal — eine je
// Geschäftsjahr: es hält Vorgänge fest, die keinem Jahr zugeordnet sind (ein
// Mandantenimport, eine Sicherung, eine Einstellungsänderung), und ein Schnitt
// zum Jahreswechsel wäre die Stelle, an der sich unbemerkt etwas entfernen
// ließe.
type AuditChainResult struct {
	IsValid      bool `json:"isValid"`
	TotalEntries int  `json:"totalEntries"`
	// CheckedEntries ist die Zahl der nachgerechneten Einträge. Die Prüfung
	// läuft nach einem Bruch weiter, damit nicht die erste Änderung jede
	// spätere verdeckt; das Feld sagt also, wie viel geprüft wurde.
	CheckedEntries   int               `json:"checkedEntries"`
	FirstBrokenID    *uint             `json:"firstBrokenId,omitempty"`
	Breaks           []AuditChainBreak `json:"breaks"`
	LastVerifiedHash string            `json:"lastVerifiedHash"`
	CheckedAt        string            `json:"checkedAt"`
	Message          string            `json:"message"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
//
// Das Ergebnis geht als JSON an die Oberfläche; ein nicht belegter Slice würde
// dort zu `null`, und `breaks.length` bräche ausgerechnet im Regelfall — dem
// unversehrten Protokoll.
func (r *AuditChainResult) EnsureLists() {
	if r.Breaks == nil {
		r.Breaks = make([]AuditChainBreak, 0)
	}
}
