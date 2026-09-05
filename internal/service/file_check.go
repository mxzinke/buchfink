package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// Der Belegprüflauf.
//
// Die Hash-Chain sichert die Buchungen und trägt vom Beleg nur dessen
// Prüfsumme. Ob die Datei auf der Platte noch die ist, die gebucht wurde, sagt
// sie nicht — dazu muss die Datei gelesen und neu gehasht werden. Genau das
// verlangt die Aufbewahrung: unveränderte, jederzeit lesbare Wiedergabe über
// zehn Jahre (§ 147 Abs. 1, Abs. 2 Nr. 1 AO, GoBD Rz. 110). Ein stiller
// Plattenfehler fällt sonst erst auf, wenn der Prüfer den Beleg sehen will.

// VerifyReceiptFiles prüft jede Belegdatei gegen ihre Prüfsumme.
func (s *ReceiptService) VerifyReceiptFiles(ctx context.Context) (*domain.FileCheckResult, error) {
	// Alle Geschäftsjahre: ein Beleg von vor drei Jahren muss genauso lesbar
	// sein wie der von gestern.
	receipts, err := s.receiptRepo.FindAll(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("die Belege konnten nicht gelesen werden: %w", err)
	}

	var documents []domain.AssetDocument
	if s.documents != nil {
		documents, err = s.documents.AllDocuments(ctx)
		if err != nil {
			return nil, fmt.Errorf("die Anlagendokumente konnten nicht gelesen werden: %w", err)
		}
	}

	result := checkReceiptFiles(receipts, documents, s.store)
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionIntegrityCheck, "RECEIPT_FILES", "",
			result.Message)
	}
	return result, nil
}

// DocumentSource ist der Ausschnitt der Anlagenkartei, den der Belegprüflauf
// braucht: alle Dokumente, ohne Rücksicht auf das Anlagegut, an dem sie hängen.
type DocumentSource interface {
	AllDocuments(ctx context.Context) ([]domain.AssetDocument, error)
}

// SetDocumentSource hängt die Anlagendokumente an den Prüflauf. Ohne sie prüft
// er nur die Belege — ein Vertrag zum Anlagegut ist aber genauso
// aufbewahrungspflichtig.
func (s *ReceiptService) SetDocumentSource(src DocumentSource) { s.documents = src }

// checkReceiptFiles ist der eigentliche Lauf. Er steht getrennt vom Dienst,
// weil ihn der Wiederherstellungstest auf einem entpackten Datenordner braucht,
// zu dem es keine Dienste gibt.
func checkReceiptFiles(
	receipts []domain.Receipt, documents []domain.AssetDocument, store *receiptstore.Store,
) *domain.FileCheckResult {
	result := &domain.FileCheckResult{
		Issues:    make([]domain.FileCheckIssue, 0),
		CheckedAt: time.Now().Format("02.01.2006 15:04:05"),
	}
	if store == nil {
		result.Message = "Es ist kein Belegspeicher eingerichtet."
		return result
	}

	check := func(kind, receiptNumber, fileName, path, sha string) {
		result.Checked++
		if !store.Exists(path) {
			result.Missing++
			result.Issues = append(result.Issues, domain.FileCheckIssue{
				Kind: kind, ReceiptNumber: receiptNumber, FileName: fileName, Path: path,
				Reason:  "missing",
				Message: fmt.Sprintf("Die Datei %s liegt nicht mehr im Datenordner.", fileName),
			})
			return
		}
		if err := store.Verify(path, sha); err != nil {
			result.Damaged++
			result.Issues = append(result.Issues, domain.FileCheckIssue{
				Kind: kind, ReceiptNumber: receiptNumber, FileName: fileName, Path: path,
				Reason:  "damaged",
				Message: fmt.Sprintf("Die Datei %s stimmt nicht mehr mit ihrer Prüfsumme überein.", fileName),
			})
			return
		}
		result.Intact++
	}

	for i := range receipts {
		r := &receipts[i]
		for j := range r.Files {
			f := &r.Files[j]
			check("receipt", r.ReceiptNumber, f.FileName, f.StoredPath, f.SHA256)
		}
	}
	for i := range documents {
		d := &documents[i]
		check("document", "", d.FileName, d.StoredPath, d.SHA256)
	}

	result.IsValid = result.Damaged == 0 && result.Missing == 0
	switch {
	case result.Checked == 0:
		result.Message = "Es sind keine Dateien abgelegt."
	case result.IsValid:
		result.Message = fmt.Sprintf("Alle %d Dateien sind unverändert lesbar.", result.Checked)
	default:
		result.Message = fmt.Sprintf(
			"%d von %d Dateien sind zu beanstanden: %d beschädigt, %d fehlen.",
			result.Damaged+result.Missing, result.Checked, result.Damaged, result.Missing)
	}
	return result
}
