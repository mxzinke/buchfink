package taxrules

import (
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// TaxAccountExceptions erlaubt Bestandsbuchungen ohne Steuerschlüssel.
// Der Saldenvortrag erhält die Bilanzidentität (§ 252 Abs. 1 Nr. 1 HGB).
// Die Jahresverrechnung stellt die Steuerkonten auf null und weist den Saldo
// als Forderung oder Verbindlichkeit aus. Beide buchen Bestände, keine Umsätze;
// die Umsatzsteuer-Auswertung lässt Eröffnungs- und Abschlussbuchungen aus.
func TaxAccountExceptions() []domain.TaxAccountException {
	return []domain.TaxAccountException{
		{Name: "opening_balances", AppliesToReversal: true, Allows: func(e *domain.JournalEntry, _ string) bool { return e.Source == domain.EntrySourceOpening }},
		{Name: "vat_settlement", AppliesToReversal: true, Allows: taxSettlement},
	}
}

// isVatSettlementReference erkennt die Belegnummer einer Jahresverrechnung.
//
// Geprüft wird die Form und nicht das Jahr des Eintrags: die Generalumkehr einer
// Verrechnung hat deren Belegnummer, liegt aber im Geschäftsjahr ihrer
// Erstellung. Mit einem Vergleich auf das Jahr des Eintrags ließe sich eine
// Verrechnung nicht mehr stornieren.
func isVatSettlementReference(documentNumber string) bool {
	year, err := strconv.Atoi(strings.TrimPrefix(documentNumber, "USTV "))
	if err != nil {
		return false
	}
	return documentNumber == accounting.VatSettlementReference(year)
}

// taxSettlement erlaubt ausschließlich die Konten der Jahresverrechnung.
// Die Quelle wird beim Handbuchungsweg auf manual gesetzt; die Belegnummer
// allein kann deshalb keine Ausnahme freischalten.
func taxSettlement(e *domain.JournalEntry, account string) bool {
	if e.Source != domain.EntrySourceClosing || !isVatSettlementReference(e.DocumentNumber) {
		return false
	}
	for _, settlement := range accounting.VatSettlementAccounts() {
		if settlement == account {
			return true
		}
	}
	return false
}
