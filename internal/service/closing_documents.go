package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Eigenbeleg der Abschlussbuchung.
//
// GoBD Rz. 61 lässt keine Buchung ohne Beleg zu, und eine Abschlussbuchung hat
// von außen keinen: die Rechnungsabgrenzung, die Rückstellung und die
// Steuerschätzung entstehen aus einer Rechnung, die Buchfink selbst anstellt.
// Also erzeugt Buchfink den Beleg — die Rechnung als Datei, abgelegt im
// Belegspeicher wie jeder andere Beleg, mit der Belegart „Eigenbeleg".
//
// Die Datei ist JSON und kein PDF. Ein PDF wäre hübscher; JSON ist prüfbar: die
// Zahlen stehen als Zahlen darin, und wer die Buchung nachrechnen will, muss
// sie nicht aus einem Layout zurückgewinnen. Der Beleg trägt zusätzlich einen
// erklärenden Text, damit er auch ohne Buchfink lesbar bleibt.

// closingVoucher ist der Inhalt eines Abschluss-Eigenbelegs.
type closingVoucher struct {
	Kind        string `json:"art"`
	FiscalYear  int    `json:"geschaeftsjahr"`
	Date        string `json:"buchungsdatum"`
	Description string `json:"bezeichnung"`
	// Explanation sagt in einem Satz, worauf die Buchung beruht.
	Explanation string `json:"erlaeuterung"`
	// Calculation ist die Rechnung selbst: die Größen, aus denen der Betrag
	// entstanden ist.
	Calculation any                  `json:"berechnung"`
	Lines       []domain.JournalLine `json:"buchungssatz"`
}

// closingReceiptFiler ist der Ausschnitt des Belegdienstes, den die
// Abschlussbausteine brauchen. Als Schnittstelle und nicht als Zeiger, damit
// ein Baustein auch ohne Belegspeicher arbeitet — im Test etwa, wo kein
// Verzeichnis eingerichtet ist.
type closingReceiptFiler interface {
	File(ctx context.Context, req FileReceiptRequest) (*domain.Receipt, error)
	Seal(ctx context.Context, receiptID, entryID uint) error
	Discard(ctx context.Context, receiptID uint, reason string) error
	// Get prüft einen mitgegebenen Belegverweis. Ein Baustein, der eine
	// Inventurliste verlangt, muss auch nachsehen können, ob es sie gibt —
	// sonst genügt jede Zahl ungleich null.
	Get(ctx context.Context, id uint) (*domain.Receipt, error)
}

// selfIssuedVoucher legt den Eigenbeleg ab und hängt ihn an die Buchung.
//
// Scheitert das Ablegen, scheitert der Baustein: eine Abschlussbuchung ohne
// Beleg zu schreiben und den Fehler zu verschweigen wäre die schlechtere
// Antwort — sie fiele erst im Prüflauf auf, dann aber als Befund gegen den
// Anwender.
func selfIssuedVoucher(
	ctx context.Context, filer closingReceiptFiler, fiscalYear int, voucher closingVoucher,
) (*domain.Receipt, error) {
	if filer == nil {
		return nil, nil
	}
	content, err := json.MarshalIndent(voucher, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("der Eigenbeleg konnte nicht erzeugt werden: %w", err)
	}
	receipt, err := filer.File(ctx, FileReceiptRequest{
		Direction:   domain.DirectionIncoming,
		FiscalYear:  fiscalYear,
		Kind:        domain.ReceiptKindSelfIssued,
		ReceivedAt:  voucher.Date,
		ReceivedVia: domain.ReceivedViaSelfIssued,
		Files: []NewFile{{
			Role:     domain.ReceiptRoleOriginal,
			Content:  content,
			FileName: fmt.Sprintf("abschluss-%s-%s.json", voucher.Kind, voucher.Date),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("der Eigenbeleg zur Abschlussbuchung konnte nicht abgelegt werden: %w", err)
	}
	return receipt, nil
}

// postWithVoucher schreibt die Buchung und räumt den Eigenbeleg auf, wenn sie
// scheitert.
//
// Der Beleg muss vor der Buchung entstehen: sein Hash geht in den Hash der
// Buchung ein, nachträglich gesetzt wäre er nicht mehr gedeckt. Scheitert die
// Buchung dann — an der Periodensperre, an einem festgestellten Jahr, an einem
// unbekannten Konto —, bliebe ohne diesen Weg ein unversiegelter Eigenbeleg im
// Belegspeicher zurück, der auf nichts verweist. Gelöscht wird er nicht: die
// GoBD kennen kein Löschen, sondern das Verwerfen mit Grund.
func postWithVoucher(
	ctx context.Context, journalSvc *JournalService, filer closingReceiptFiler,
	entry *domain.JournalEntry, receipt *domain.Receipt,
) (*domain.JournalEntry, error) {
	created, err := journalSvc.Post(ctx, entry)
	if err != nil {
		if receipt != nil && filer != nil {
			_ = filer.Discard(ctx, receipt.ID, fmt.Sprintf(
				"Die Abschlussbuchung %q ist nicht zustande gekommen: %v", entry.Description, err))
		}
		return nil, err
	}
	if receipt != nil && filer != nil {
		_ = filer.Seal(ctx, receipt.ID, created.ID)
	}
	return created, nil
}

// attachVoucher trägt den Belegverweis in die Buchung ein, bevor sie geschrieben
// wird. Der Verweis muss vor dem Schreiben stehen: der Belegverweis geht in den
// Hash der Buchung ein, nachträglich gesetzt wäre er nicht mehr gedeckt.
func attachVoucher(entry *domain.JournalEntry, receipt *domain.Receipt) {
	if receipt == nil {
		return
	}
	id := receipt.ID
	entry.ReceiptID = &id
	entry.ReceiptHash = receipt.ReceiptHash
	if entry.DocumentNumber == "" {
		entry.DocumentNumber = receipt.ReceiptNumber
	}
}
