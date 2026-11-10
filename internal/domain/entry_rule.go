package domain

import "context"

// EntryRulePosition legt die Einhängepunkte zwischen den festen Journalregeln fest.
type EntryRulePosition uint8

const (
	EntryRulesBeforeValidation EntryRulePosition = iota
	EntryRulesAfterValidation
	// EntryRulesAfterAccount läuft nach der Kernprüfung jeder einzelnen Sachkontozeile.
	EntryRulesAfterAccount
	EntryRulesAfterAccounts
	// EntryRulesManual läuft auch vor der Belegerzeugung einer Handbuchung.
	EntryRulesManual
	EntryRulesAfterPeriods
)

// EntryRule prüft eine Buchung im Kontext des Aufrufers.
// Innerhalb einer Position gilt die Registrierungsreihenfolge. Regeln ändern
// weder die Buchung noch gespeicherte Daten und greifen nicht auf das Netz zu.
type EntryRule struct {
	Name              string
	Position          EntryRulePosition
	AppliesToReversal bool
	Check             func(context.Context, *JournalEntry, EntryRuleView) error
}

// EntryRuleView gibt Regeln nur die benötigten Lesezugriffe. AccountLine ist
// der nullbasierte Zeilenindex bei EntryRulesAfterAccount, sonst -1.
type EntryRuleView interface {
	TaxationType(context.Context) string
	Receipt(context.Context, uint) (*Receipt, error)
	AccountLine() int
}

// TaxAccountException erlaubt eine Bestandsbuchung ohne Steuerschlüssel.
// Die Prüfung, ob das Konto bebuchbar ist, bleibt davon unberührt.
type TaxAccountException struct {
	Name              string
	AppliesToReversal bool
	Allows            func(*JournalEntry, string) bool
}
