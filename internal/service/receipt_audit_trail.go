package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Prüfvermerk am Eingangsbeleg (RECH-08).
//
// „Geprüft gegen Bestellung 4711 vom 3. März, Ware vollständig eingegangen" —
// ein Satz, der beim Buchen zehn Sekunden kostet und Jahre später die einzige
// Auskunft darüber ist, ob jemand die Rechnung inhaltlich angesehen hat. Er
// steht außerhalb des Beleg-Hashes (siehe domain.Receipt) und ist deshalb auch
// am gebuchten Beleg noch nachzutragen; jede Änderung geht ins
// Änderungsprotokoll.

// SaveServiceProof schreibt den Leistungsnachweis an einen Beleg.
//
// date ist der Tag der Prüfung; leer heißt heute. Ein leerer Text löscht den
// Vermerk — das muss möglich sein, weil ein falsch erfasster Vermerk sonst
// stehen bliebe; das Protokoll hält beide Stände fest.
func (s *ReceiptService) SaveServiceProof(
	ctx context.Context, receiptID uint, proof, date string,
) (*domain.Receipt, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if receipt.Direction != domain.DirectionIncoming {
		return nil, fmt.Errorf(
			"Beleg %s ist ein Ausgangsbeleg. Der Leistungsnachweis gehört an den Eingangsbeleg — geprüft wird, was jemand anderes in Rechnung stellt",
			receipt.ReceiptNumber)
	}
	proof = strings.TrimSpace(proof)
	if proof != "" && date == "" {
		date = todayLocal()
	}
	if proof == "" {
		date = ""
	}
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return nil, fmt.Errorf("%q ist kein gültiges Datum (erwartet JJJJ-MM-TT)", date)
		}
	}

	if err := s.receiptRepo.SaveAuditTrail(ctx, receiptID, receipt.OrderReference, proof, date); err != nil {
		return nil, fmt.Errorf("der Leistungsnachweis konnte nicht gespeichert werden: %w", err)
	}
	updated, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "RECEIPT", fmt.Sprintf("%d", receiptID),
			fmt.Sprintf("Leistungsnachweis zu Beleg %s erfasst", updated.ReceiptNumber),
			map[string]string{"serviceProof": receipt.ServiceProof, "serviceProofAt": receipt.ServiceProofAt},
			map[string]string{"serviceProof": proof, "serviceProofAt": date})
	}
	return updated, nil
}

// SaveOrderReference schreibt den Bestellbezug an einen Beleg.
//
// Nicht auf der Bridge: die Bestellnummer kommt aus dem strukturierten Teil der
// E-Rechnung (BT-13) und wird beim Lesen übernommen. Von Hand nachzutragen ist
// sie über denselben Weg wie die übrigen Kopfdaten.
func (s *ReceiptService) SaveOrderReference(
	ctx context.Context, receiptID uint, reference string,
) (*domain.Receipt, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	reference = strings.TrimSpace(reference)
	if reference == receipt.OrderReference {
		return receipt, nil
	}
	if err := s.receiptRepo.SaveAuditTrail(
		ctx, receiptID, reference, receipt.ServiceProof, receipt.ServiceProofAt); err != nil {
		return nil, fmt.Errorf("der Bestellbezug konnte nicht gespeichert werden: %w", err)
	}
	updated, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	// Ins Änderungsprotokoll, so wie der Leistungsnachweis: beide Felder stehen
	// außerhalb des Beleg-Hashes, und was sie unveränderbar macht, ist allein
	// das Protokoll. Ein Bestellbezug, der sich still ändern ließe, entwertete
	// den Prüfpfad, den er tragen soll.
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "RECEIPT", fmt.Sprintf("%d", receiptID),
			fmt.Sprintf("Bestellbezug zu Beleg %s geändert", updated.ReceiptNumber),
			map[string]string{"orderReference": receipt.OrderReference},
			map[string]string{"orderReference": reference})
	}
	return updated, nil
}
