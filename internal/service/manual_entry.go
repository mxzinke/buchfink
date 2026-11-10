package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Handbuchung mit Beleg (BEL-01 K2, GOB-05 K1).
//
// „Keine Buchung ohne Beleg" galt in Buchfink bisher überall außer da, wo es am
// leichtesten zu unterlaufen war: die von Hand erfasste Buchung nahm jeden
// Buchungssatz an, auch ohne einen Beleg dahinter. Das ist der Mangel, den
// § 146 Abs. 1 AO und GoBD Rz. 61 meinen, und er fällt in einer Prüfung sofort
// auf — der Prüflauf und der Bericht melden jede Buchung ohne Belegverweis.
//
// Ab jetzt verlangt der Handbuchungsweg einen Beleg, und er bietet den zweiten
// Weg gleich mit an: gibt es keinen fremden, entsteht im selben Vorgang ein
// Eigenbeleg. Beides in einer Transaktion, denn die Alternative wäre die
// schlechteste von allen — ein Eigenbeleg im Speicher, dessen Buchung
// gescheitert ist, also der ungebuchte Beleg, den der Prüflauf meldet.

// ManualEntryRequest ist die Eingabe des Handbuchungswegs.
type ManualEntryRequest struct {
	// Entry ist der Buchungssatz, wie ihn die Maske zusammengestellt hat.
	Entry domain.JournalEntry `json:"entry"`
	// ReceiptID ist der bereits abgelegte Beleg. Alternative zu SelfIssued.
	ReceiptID uint `json:"receiptId,omitempty"`
	// SelfIssued sind die Angaben eines Eigenbelegs, der in demselben Vorgang
	// entsteht. Alternative zu ReceiptID.
	SelfIssued *SelfIssuedReceiptRequest `json:"selfIssued,omitempty"`
}

// PostManualEntry bucht einen von Hand erfassten Buchungssatz.
func (s *PostingService) PostManualEntry(
	ctx context.Context, req ManualEntryRequest,
) (*domain.JournalEntry, error) {
	entry := req.Entry
	entry.Lines = append([]domain.JournalLine(nil), req.Entry.Lines...)
	// Die Quelle ist keine Eingabe, sondern die Herkunft der Buchung: über
	// diesen Weg entsteht eine Handbuchung und nichts anderes. Ein von außen
	// mitgeschicktes „closing" umginge sonst den Schutz der Steuerkonten.
	entry.Source = domain.EntrySourceManual

	receiptID := req.ReceiptID
	if receiptID == 0 && entry.ReceiptID != nil {
		receiptID = *entry.ReceiptID
	}
	if receiptID == 0 && req.SelfIssued == nil {
		return nil, fmt.Errorf(
			"zu einer Buchung gehört ein Beleg (§ 146 Abs. 1 AO, GoBD Rz. 61). Nötig ist der abgelegte " +
				"Beleg zu diesem Vorgang — oder ein Eigenbeleg, wenn es keinen fremden gibt")
	}
	if receiptID != 0 && req.SelfIssued != nil {
		return nil, fmt.Errorf(
			"die Buchung verweist auf einen abgelegten Beleg und soll zugleich einen Eigenbeleg " +
				"erzeugen. Zu einem Geschäftsvorfall gehört ein Beleg — entscheide dich für einen der " +
				"beiden Wege")
	}
	if s.journalSvc != nil {
		if err := s.journalSvc.checkRulesAt(ctx, domain.EntryRulesManual, &entry, -1); err != nil {
			return nil, err
		}
	}
	if err := requireShareholderContacts(entry.Lines); err != nil {
		return nil, err
	}
	if s.journalSvc == nil {
		return nil, fmt.Errorf("der Buchungsweg ist nicht eingerichtet")
	}

	// Erst prüfen, dann den Eigenbeleg erzeugen: eine Buchung, die an einem
	// gesperrten Zeitraum oder einem unbekannten Konto scheitert, darf keinen
	// Beleg im Speicher hinterlassen. Die Vorprüfung läuft ohne den Belegbezug,
	// weil es ihn dann noch nicht gibt — die Kopfdatenprüfung des Belegs holt
	// Post danach nach.
	if err := s.journalSvc.ValidatePostable(ctx, &entry); err != nil {
		return nil, err
	}

	var created *domain.JournalEntry
	// Der Eigenbeleg dieses Vorgangs, gemerkt für den Fehlerfall: seine Zeilen
	// rollt die Transaktion zurück, seine Datei liegt schon im Belegspeicher.
	var selfIssued *domain.Receipt
	err := s.runInTx(ctx, func(ctx context.Context) error {
		receipt, err := s.manualReceipt(ctx, req, receiptID)
		if err != nil {
			return err
		}
		if req.SelfIssued != nil {
			selfIssued = receipt
		}
		entry.ReceiptID = &receipt.ID
		entry.ReceiptHash = receipt.ReceiptHash
		if entry.DocumentNumber == "" {
			entry.DocumentNumber = receipt.ReceiptNumber
		}
		posted, err := s.journalSvc.Post(ctx, &entry)
		if err != nil {
			return err
		}
		created = posted
		// Das Versiegeln gehört in dieselbe Klammer: ein Beleg, der als offen
		// zurückbliebe, nähme später eine weitere Datei auf und änderte damit
		// seinen Hash — den die geschriebene Buchung schon hat.
		if s.receiptSvc != nil {
			if err := s.receiptSvc.Seal(ctx, receipt.ID, posted.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// Die zurückgerollte Anlage hinterlässt sonst das PDF ohne Beleg.
		if selfIssued != nil && s.receiptSvc != nil {
			s.receiptSvc.DropRolledBackFiles(ctx, selfIssued)
		}
		return nil, err
	}
	return created, nil
}

// manualReceipt liefert den Beleg der Handbuchung: den abgelegten oder den
// Eigenbeleg, der jetzt entsteht.
func (s *PostingService) manualReceipt(
	ctx context.Context, req ManualEntryRequest, receiptID uint,
) (*domain.Receipt, error) {
	if s.receiptSvc == nil {
		return nil, fmt.Errorf("die Belegablage ist nicht eingerichtet")
	}
	if req.SelfIssued != nil {
		self := *req.SelfIssued
		if self.DocumentDate == "" {
			self.DocumentDate = firstNonEmptyString(req.Entry.DocumentDate, req.Entry.BookingDate)
		}
		if self.GrossAmount == 0 {
			self.GrossAmount = req.Entry.GrossAmount()
		}
		if self.FiscalYear == 0 {
			self.FiscalYear = req.Entry.FiscalYear
		}
		return s.receiptSvc.CreateSelfIssued(ctx, self)
	}
	receipt, err := s.receiptSvc.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if receipt.Status == domain.ReceiptStatusDiscarded {
		return nil, fmt.Errorf(
			"Beleg %s wurde verworfen und hat keine Buchung", receipt.ReceiptNumber)
	}
	return receipt, nil
}

// firstNonEmptyString liefert den ersten belegten Wert.
func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
