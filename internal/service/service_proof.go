package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Leistungsnachweis am Eingangsbeleg (RECH-08).
//
// Ab der eingestellten Grenze (invoice_check_threshold, Voreinstellung
// 1.000 Euro) ist der Vermerk Pflicht, gegen welche Bestellung die Rechnung
// geprüft wurde. Pflicht heißt: der Beleg wird ohne ihn nicht gebucht. Die
// Grenze ist eine Festlegung des internen Kontrollsystems, und ein
// Kontrollschritt, den man übergehen kann, ist keiner — wer den Vermerk erst
// nach dem Buchen nachträgt, hat die Rechnung bezahlt, bevor jemand geprüft hat,
// ob die Leistung überhaupt erbracht wurde.
//
// Die Regel steht hier und nicht dreimal im Programm: der Prüflauf meldet den
// fehlenden Vermerk, die Aufgabenliste führt ihn, und der Buchungsweg weist ab.
// Alle drei müssen dieselbe Grenze und dieselben Ausnahmen kennen, sonst zeigt
// die Aufgabenliste eine Zeile, die das Buchen nicht verlangt — oder umgekehrt.

// serviceProofWarningCode kennzeichnet den Hinweis in der Buchungsvorschau. Die
// Maske macht das Feld ab der Grenze zum Pflichtfeld und erkennt den Fall daran.
const serviceProofWarningCode = "service_proof_required"

// receiptNeedsServiceProof meldet, ob ein Eingangsbeleg über der Grenze den
// Leistungsnachweis noch schuldet.
//
// Nur Rechnungen: ein Kontoauszug, ein Handelsbrief und ein sonstiges Dokument
// haben keine Bestellung, gegen die sie zu prüfen wären. Eine leere Belegart ist
// die Rechnung — so legt sie die Belegablage an.
func receiptNeedsServiceProof(receipt *domain.Receipt, threshold domain.Cents) bool {
	if receipt == nil || threshold <= 0 {
		return false
	}
	if receipt.Direction != domain.DirectionIncoming {
		return false
	}
	if receipt.Kind != "" && receipt.Kind != domain.ReceiptKindInvoice {
		return false
	}
	return receipt.GrossAmount >= threshold && receipt.ServiceProof == ""
}

// serviceProofMissingMessage ist der Satz, der den fehlenden Vermerk benennt.
// Er nennt die Grenze, weil sie einstellbar ist: „ab 1.000 €" ist eine Auskunft,
// „Pflichtfeld" wäre eine Behauptung ohne Maß.
func serviceProofMissingMessage(receipt *domain.Receipt, threshold domain.Cents) string {
	return fmt.Sprintf(
		"Beleg %s über %s € hat keinen Leistungsnachweis. Ab %s € verlangt die eigene Vorgabe "+
			"den Vermerk, gegen welche Bestellung geprüft wurde",
		receipt.ReceiptNumber, receipt.GrossAmount, threshold)
}

// invoiceCheckThreshold liest die Nachweisgrenze. Ohne Einstellungen null: dann
// gilt keine Grenze, und der Belegweg bucht wie zuvor.
func (s *PostingService) invoiceCheckThreshold(ctx context.Context) domain.Cents {
	if s.settingsRepo == nil {
		return 0
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return 0
	}
	return cfg.InvoiceCheckThreshold
}

// saveServiceProof schreibt einen mit der Buchung mitgeschickten Vermerk an den
// Beleg.
//
// Geschrieben wird über den Belegdienst: dort steht die Prüfung des Datums, dort
// entsteht der Protokolleintrag, und dort bleibt der Beleg-Hash unangetastet.
// Gerufen wird der Weg innerhalb der Transaktion der Buchung — ein Vorgang, der
// danach scheitert, hinterlässt sonst einen Vermerk ohne Buchung. Ein leeres
// Feld ändert nichts — der nachgetragene Vermerk läuft weiter über
// SaveServiceProof, und eine Buchung ohne Angabe darf einen schon erfassten
// Vermerk nicht löschen.
func (s *PostingService) saveServiceProof(
	ctx context.Context, built *incomingLines, req ReceiptRequest,
) error {
	if built == nil || built.receipt == nil || s.receiptSvc == nil {
		return nil
	}
	if strings.TrimSpace(req.ServiceProof) == "" {
		return nil
	}
	updated, err := s.receiptSvc.SaveServiceProof(
		ctx, built.receipt.ID, req.ServiceProof, req.ServiceProofAt)
	if err != nil {
		return err
	}
	built.receipt = updated
	return nil
}

// requireServiceProof hält die Buchung an, solange der Vermerk fehlt.
//
// Geprüft wird vor der Transaktion, geschrieben wird in ihr. Deshalb zählt hier
// auch der Vermerk, den die Maske mit der Buchung mitschickt: er steht noch
// nicht am Beleg, ist aber Teil desselben Vorgangs. Ob er ein gültiges Datum
// hat, entscheidet der Belegdienst beim Schreiben.
func (s *PostingService) requireServiceProof(
	ctx context.Context, receipt *domain.Receipt, req ReceiptRequest,
) error {
	if strings.TrimSpace(req.ServiceProof) != "" {
		return nil
	}
	threshold := s.invoiceCheckThreshold(ctx)
	if !receiptNeedsServiceProof(receipt, threshold) {
		return nil
	}
	return fmt.Errorf(
		"%s. Trage ihn am Beleg ein — etwa „geprüft gegen Bestellung 4711 vom 12.03.2026\" — und "+
			"buche dann; die Grenze steht in den Einstellungen",
		serviceProofMissingMessage(receipt, threshold))
}

// serviceProofNotice ist derselbe Sachverhalt in der Vorschau: die Maske soll
// das Feld verlangen, bevor jemand auf „Buchen" drückt, und nicht erst danach
// eine Fehlermeldung zeigen.
func (s *PostingService) serviceProofNotice(
	ctx context.Context, receipt *domain.Receipt,
) *PostingWarning {
	threshold := s.invoiceCheckThreshold(ctx)
	if !receiptNeedsServiceProof(receipt, threshold) {
		return nil
	}
	return &PostingWarning{
		Code:     serviceProofWarningCode,
		Severity: "warning",
		Title:    "Der Leistungsnachweis fehlt",
		Detail: fmt.Sprintf(
			"%s. Ohne ihn wird der Beleg nicht gebucht.",
			serviceProofMissingMessage(receipt, threshold)),
	}
}
