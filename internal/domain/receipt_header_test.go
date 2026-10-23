package domain

import (
	"strings"
	"testing"
)

// --- Belegkopfdaten -------------------------------------------------------

func bookableReceipt() *Receipt {
	return &Receipt{
		ReceiptNumber: "ER-2026-0001",
		FiscalYear:    2026,
		Direction:     DirectionIncoming,
		Status:        ReceiptStatusFiled,
		Kind:          ReceiptKindInvoice,
		DocumentDate:  "2026-03-01",
		IssuerName:    "Lieferant GmbH",
		GrossAmount:   11900,
		TaxAmount:     1900,
		Currency:      "EUR",
		Files: []ReceiptFile{{
			Position: 1, Role: ReceiptRoleOriginal, FileName: "rechnung.pdf",
			MimeType: "application/pdf", Size: 1024,
			SHA256:     strings.Repeat("a", 64),
			StoredPath: "belege/2026/eingang/aaa.pdf",
		}},
	}
}

func TestValidateBookableDemandsHeaderFields(t *testing.T) {
	receipt := bookableReceipt()
	if err := receipt.ValidateBookable(); err != nil {
		t.Fatalf("ein vollständiger Beleg muss buchbar sein: %v", err)
	}

	withoutDate := bookableReceipt()
	withoutDate.DocumentDate = ""
	if err := withoutDate.ValidateBookable(); err == nil {
		t.Error("ohne Belegdatum darf nicht gebucht werden — die zeitgerechte Erfassung wäre nicht zu beurteilen")
	}

	withoutIssuer := bookableReceipt()
	withoutIssuer.IssuerName = ""
	if err := withoutIssuer.ValidateBookable(); err == nil {
		t.Error("ohne Aussteller darf nicht gebucht werden (§ 14 Abs. 4 Nr. 1 UStG)")
	}

	withoutAmount := bookableReceipt()
	withoutAmount.GrossAmount = 0
	if err := withoutAmount.ValidateBookable(); err == nil {
		t.Error("ohne Betrag darf nicht gebucht werden")
	}
}

func TestValidateBookableHasOtherRulesForStatementsAndSelfIssued(t *testing.T) {
	// Ein Kontoauszug hat keinen Aussteller im Sinne einer Rechnung und keinen
	// einzelnen Betrag: er braucht nur seine Einordnung in der Zeit.
	statement := bookableReceipt()
	statement.Kind = ReceiptKindStatement
	statement.IssuerName = ""
	statement.GrossAmount = 0
	if err := statement.ValidateBookable(); err != nil {
		t.Errorf("ein Kontoauszug braucht nur das Belegdatum: %v", err)
	}

	// Der Eigenbeleg hat keinen fremden Aussteller, aber einen Anlass.
	selfIssued := bookableReceipt()
	selfIssued.Kind = ReceiptKindSelfIssued
	selfIssued.IssuerName = ""
	if err := selfIssued.ValidateBookable(); err == nil {
		t.Error("ein Eigenbeleg ohne Betreff nennt seinen Anlass nicht und darf nicht gebucht werden")
	}
	selfIssued.Subject = "Trinkgeld Taxifahrt Kundentermin"
	if err := selfIssued.ValidateBookable(); err != nil {
		t.Errorf("ein Eigenbeleg mit Betreff und Betrag ist buchbar: %v", err)
	}
}

func TestOtherDocumentsNeedDateAndSubjectInsteadOfIssuerAndAmount(t *testing.T) {
	// Die Schlussbilanz des Altsystems, der Gesellschafterbeschluss:
	// Dokumente, die gebucht werden, aber keine
	// Rechnung sind. § 14 Abs. 4 Nr. 1 UStG gilt für sie nicht — sie haben
	// keinen leistenden Unternehmer und oft keinen einzelnen Betrag.
	for _, kind := range []ReceiptKind{ReceiptKindOther, ReceiptKindLetter} {
		document := bookableReceipt()
		document.Kind = kind
		document.IssuerName = ""
		document.GrossAmount = 0

		if err := document.ValidateBookable(); err == nil {
			t.Errorf("%s: ohne Betreff ist nicht zu erkennen, was das Dokument belegt", kind.Label())
		} else if strings.Contains(err.Error(), "§ 14 Abs. 4 Nr. 1 UStG") {
			t.Errorf("%s: die Rechnungspflichtangabe gilt hier nicht: %v", kind.Label(), err)
		}

		document.Subject = "Schlussbilanz zum 31.12.2025"
		if err := document.ValidateBookable(); err != nil {
			t.Errorf("%s mit Belegdatum und Betreff ist buchbar: %v", kind.Label(), err)
		}
	}

	// Die Rechnung bleibt, was sie war: ohne Aussteller und Betrag nicht
	// buchbar.
	invoice := bookableReceipt()
	invoice.IssuerName = ""
	if err := invoice.ValidateBookable(); err == nil {
		t.Error("eine Rechnung ohne Aussteller darf nicht gebucht werden (§ 14 Abs. 4 Nr. 1 UStG)")
	}
}

func TestLetterIsNotBooked(t *testing.T) {
	// Ein Handelsbrief belegt eine Abrede, keinen Geschäftsvorfall. Meldete der
	// Prüflauf ihn als „abgelegt, aber nicht gebucht", stünde er dort für immer.
	if ReceiptKindLetter.RequiresBooking() {
		t.Error("ein Handelsbrief wird nicht gebucht")
	}
	if !ReceiptKindInvoice.RequiresBooking() {
		t.Error("eine Rechnung ist zu buchen")
	}
	if RetentionKindOf(ReceiptKindLetter) != RetentionKindReceiptLetter {
		t.Error("der Handelsbrief muss seine eigene Aufbewahrungsart haben — sechs statt acht Jahre")
	}
}
