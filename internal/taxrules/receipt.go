package taxrules

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// ensureReceiptHeader verlangt die Kopfdaten des Belegs, auf den die Buchung
// verweist.
//
// Die Kopfdaten sind beim Ablegen freiwillig und beim Buchen Pflicht (BEL-02).
// „Beim Buchen" heißt: hier — Post ist der einzige Schreibweg ins Journal, und
// eine Regel, die nur im Dialog „Beleg buchen" gilt, ließe jeden anderen Weg
// (Abschlussbuchung, Anlagenzugang, Zahlungsausgleich) an ihr vorbei.
//
// Geprüft wird ValidateHeader und nicht ValidateBookable: die dort zusätzlich
// verlangte Ansehbarkeit gilt dem Beleg, den ein Mensch vor sich hat, und
// nicht dem maschinell erzeugten Eigenbeleg einer Abschlussbuchung.
//
// Die Generalumkehr ist in der Regelliste ausgenommen (GoBD Rz. 58):
// die Kopfdatenpflicht darf die Korrektur vorhandener Buchungen nicht sperren.
func ensureReceiptHeader(ctx context.Context, entry *domain.JournalEntry, view domain.EntryRuleView) error {
	if entry.ReceiptID == nil {
		return nil
	}
	receipt, err := view.Receipt(ctx, *entry.ReceiptID)
	if err != nil || receipt == nil {
		return nil
	}
	// Ein Beleg, der nicht mehr offen ist, ist bereits gebucht (versiegelt) oder
	// verworfen. Seine Kopfdaten sind beim Versiegeln geprüft worden
	// (ReceiptService.Seal) und lassen sich nicht mehr nachtragen, weil
	// SaveHeader den offenen Beleg verlangt; eine verschärfte Prüfung darf ihn
	// nicht nachträglich sperren.
	if receipt.Status != domain.ReceiptStatusFiled {
		return nil
	}
	return receipt.ValidateHeader()
}
