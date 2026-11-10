package taxrules

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

func validateCollectiveAccount(_ context.Context, e *domain.JournalEntry, view domain.EntryRuleView) error {
	i := view.AccountLine()
	l := e.Lines[i]
	// Offene Posten gehören auf das Personenkonto des Geschäftspartners.
	// Eine Buchung direkt auf das Sammelkonto stünde zwar in der Bilanz,
	// aber in keiner OPOS-Liste.
	if kind, ok := domain.CollectiveAccounts()[l.Account]; ok {
		partner := "Kunden"
		if kind == domain.ContactTypeVendor {
			partner = "Lieferanten"
		}
		return fmt.Errorf(
			"Zeile %d: Konto %s ist das Sammelkonto für die Bilanz und wird nicht direkt bebucht. "+
				"Der offene Posten gehört auf das Personenkonto des %s – die Bilanzposition verdichtet sich daraus",
			i+1, l.Account, partner,
		)
	}
	return nil
}
