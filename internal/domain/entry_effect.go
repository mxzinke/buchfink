package domain

import "context"

// EntryEffect schreibt Buchungsfolgen in der Transaktion des Journals.
// Die Funktionen laufen in Registrierungsreihenfolge, ohne Netz und ohne die
// Buchung zu ändern. Ihre Abhängigkeiten erhalten sie als schmale Schreib-
// und Lesesichten beim Verdrahten.
type EntryEffect struct {
	Name string
	// BeforeAppend läuft nach allen Regeln, nur bei Normalbuchungen.
	BeforeAppend func(context.Context, *JournalEntry) error
	// AfterPost läuft nach Append, vor dem Journalprotokoll.
	AfterPost func(context.Context, *JournalEntry) error
	// AfterReverse ersetzt AfterPost bei Generalumkehr und erhält Original und Umkehr.
	AfterReverse func(context.Context, *JournalEntry, *JournalEntry) error
}
