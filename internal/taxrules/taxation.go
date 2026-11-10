package taxrules

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Buchungsweg setzt Sollversteuerung voraus (§ 16 Abs. 1 Satz 1 UStG).
// Istversteuerung braucht einen Buchungsweg bei Zahlungseingang (§ 13 Abs. 1
// Nr. 1 Buchst. b UStG). Die Generalumkehr bleibt über die Regelliste möglich.
func ensureAccrualTaxation(ctx context.Context, _ *domain.JournalEntry, view domain.EntryRuleView) error {
	taxation := view.TaxationType(ctx)
	if taxation == "" || strings.EqualFold(taxation, "SOLL") {
		return nil
	}
	// Kein Verweis auf eine Einstellung: die Besteuerungsart ist in der
	// Oberfläche nur ablesbar, nicht änderbar. Eine Meldung, die auf einen
	// Schalter zeigt, den es nicht gibt, hilft niemandem weiter.
	return fmt.Errorf(
		"Für diesen Mandanten ist die Istversteuerung hinterlegt. Buchfink unterstützt derzeit nur die Sollversteuerung (§ 16 Abs. 1 Satz 1 UStG): bei Istversteuerung entsteht die Steuer erst mit der Vereinnahmung des Entgelts (§ 13 Abs. 1 Nr. 1 Buchst. b UStG), und die Buchungen sähen anders aus. Stornos bestehender Buchungen bleiben möglich")
}
