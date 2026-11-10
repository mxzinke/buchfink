package accounting

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// VatSettlementReference bezeichnet die Jahresverrechnung der Umsatzsteuer.
func VatSettlementReference(year int) string { return fmt.Sprintf("USTV %d", year) }

// VatSettlementAccounts sind die Konten, die die Jahresverrechnung anfasst: die
// Steuerautomatik-Konten plus die Vorauszahlungskonten.
func VatSettlementAccounts() []string {
	return []string{
		domain.AccountVorsteuer, domain.AccountVorsteuer7, domain.AccountVorsteuer19,
		domain.AccountVorsteuerIG, domain.AccountVorsteuerIG19,
		domain.AccountVorsteuer13b, domain.AccountVorsteuer13b19,
		domain.AccountUmsatzsteuer, domain.AccountUmsatzsteuer7, domain.AccountUmsatzsteuer19,
		domain.AccountUmsatzsteuerIG, domain.AccountUmsatzsteuerIG19,
		domain.AccountUmsatzsteuer13b, domain.AccountUmsatzsteuer13b19,
		domain.AccountUmsatzsteuer14c,
		domain.AccountUmsatzsteuerVorauszahlungen, domain.AccountSondervorauszahlung,
	}
}
