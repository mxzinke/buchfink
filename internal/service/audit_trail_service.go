package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
)

// Der Prüfpfad eines Belegs (RECH-08).
//
// Die Frage einer Betriebsprüfung lautet nicht „gibt es den Beleg", sondern
// „führt der Beleg zur Buchung, die Buchung zur Zahlung und die Zahlung zum
// Kontoauszug". Jede dieser Verbindungen steht in Buchfink schon in den Daten —
// sie stand nur nirgends zusammen. Der Prüfpfad ist genau diese Kette, in einer
// Ansicht und in einer Datei (GoBD Rz. 36: progressive und retrograde
// Prüfbarkeit).

// AuditTrailStage benennt die Stufe der Kette.
type AuditTrailStage string

const (
	TrailStageReceipt AuditTrailStage = "receipt"
	TrailStageBooking AuditTrailStage = "booking"
	TrailStagePayment AuditTrailStage = "payment"
	TrailStageBank    AuditTrailStage = "bank"
)

// AuditTrailStep ist eine Stufe des Prüfpfads.
type AuditTrailStep struct {
	Stage     AuditTrailStage `json:"stage"`
	Title     string          `json:"title"`
	Date      string          `json:"date"`
	Reference string          `json:"reference"`
	Amount    domain.Cents    `json:"amount"`
	Detail    string          `json:"detail"`
}

// AuditTrail ist der Prüfpfad eines Belegs.
type AuditTrail struct {
	ReceiptID      uint         `json:"receiptId"`
	ReceiptNumber  string       `json:"receiptNumber"`
	Direction      string       `json:"direction"`
	DocumentDate   string       `json:"documentDate"`
	IssuerName     string       `json:"issuerName"`
	GrossAmount    domain.Cents `json:"grossAmount"`
	OrderReference string       `json:"orderReference,omitempty"`
	ServiceProof   string       `json:"serviceProof,omitempty"`
	ServiceProofAt string       `json:"serviceProofAt,omitempty"`

	Steps []AuditTrailStep `json:"steps"`
	// Note sagt, wo die Kette endet: ein Beleg ohne Buchung oder eine Rechnung
	// ohne Zahlung ist kein Fehler, aber eine Auskunft.
	Note string `json:"note"`
}

// EnsureLists ersetzt eine nicht belegte Schrittliste durch eine leere.
func (t *AuditTrail) EnsureLists() {
	if t.Steps == nil {
		t.Steps = make([]AuditTrailStep, 0)
	}
}

// AuditTrailService baut den Prüfpfad und gibt ihn aus.
type AuditTrailService struct {
	receiptRepo    domain.ReceiptRepository
	journalRepo    domain.JournalRepository
	allocationRepo domain.PaymentAllocationRepository
	bankRepo       domain.BankRepository
	auditRepo      domain.AuditRepository
	renderer       DocumentRenderer
}

// NewAuditTrailService wires the Prüfpfad.
func NewAuditTrailService(
	receiptRepo domain.ReceiptRepository,
	journalRepo domain.JournalRepository,
	allocationRepo domain.PaymentAllocationRepository,
	bankRepo domain.BankRepository,
	auditRepo domain.AuditRepository,
) *AuditTrailService {
	return &AuditTrailService{
		receiptRepo: receiptRepo, journalRepo: journalRepo,
		allocationRepo: allocationRepo, bankRepo: bankRepo, auditRepo: auditRepo,
	}
}

// SetRenderer hängt den Dokumentensetzer für die PDF-Ausgabe an.
func (s *AuditTrailService) SetRenderer(r DocumentRenderer) { s.renderer = r }

// Trail baut den Prüfpfad eines Belegs.
func (s *AuditTrailService) Trail(ctx context.Context, receiptID uint) (*AuditTrail, error) {
	receipt, err := s.receiptRepo.FindByID(ctx, receiptID)
	if err != nil {
		return nil, fmt.Errorf("Beleg %d wurde nicht gefunden: %w", receiptID, err)
	}

	trail := &AuditTrail{
		ReceiptID: receipt.ID, ReceiptNumber: receipt.ReceiptNumber,
		Direction: string(receipt.Direction), DocumentDate: receipt.DocumentDate,
		IssuerName: receipt.IssuerName, GrossAmount: receipt.GrossAmount,
		OrderReference: receipt.OrderReference,
		ServiceProof:   receipt.ServiceProof, ServiceProofAt: receipt.ServiceProofAt,
	}
	trail.EnsureLists()

	detail := receipt.Subject
	if receipt.OrderReference != "" {
		detail = strings.TrimSpace(detail + " (Bestellung " + receipt.OrderReference + ")")
	}
	trail.Steps = append(trail.Steps, AuditTrailStep{
		Stage: TrailStageReceipt, Title: "Beleg", Date: receipt.DocumentDate,
		Reference: receipt.ReceiptNumber, Amount: receipt.GrossAmount,
		Detail: strings.TrimSpace(strings.Join([]string{receipt.IssuerName, detail}, " – ")),
	})

	if receipt.JournalEntryID == nil {
		trail.Note = "Der Beleg ist noch nicht gebucht; der Pfad endet hier."
		return trail, nil
	}

	entry, err := s.journalRepo.FindByID(ctx, *receipt.JournalEntryID)
	if err != nil || entry == nil {
		trail.Note = "Die Buchung zu diesem Beleg ließ sich nicht lesen."
		return trail, nil
	}
	trail.Steps = append(trail.Steps, AuditTrailStep{
		Stage: TrailStageBooking, Title: "Buchung", Date: entry.BookingDate,
		Reference: entry.EntryNumber, Amount: entryAmount(entry),
		Detail: entry.Description,
	})

	if s.allocationRepo == nil {
		return trail, nil
	}
	allocations, err := s.allocationRepo.FindByOpenItem(ctx, entry.ID)
	if err != nil {
		trail.Note = "Die Zahlungszuordnungen ließen sich nicht lesen."
		return trail, nil
	}
	if len(allocations) == 0 {
		trail.Note = "Zu der Buchung ist noch keine Zahlung zugeordnet; der Pfad endet hier."
		return trail, nil
	}

	for _, allocation := range allocations {
		payment, err := s.journalRepo.FindByID(ctx, allocation.PaymentEntryID)
		date, reference, description := "", "", ""
		if err == nil && payment != nil {
			date, reference, description = payment.BookingDate, payment.EntryNumber, payment.Description
		}
		difference := ""
		if allocation.DifferenceKind != "" && allocation.DifferenceKind != domain.DifferenceNone {
			difference = fmt.Sprintf(" (%s %s €)", allocation.DifferenceKind, allocation.DifferenceAmount)
		}
		trail.Steps = append(trail.Steps, AuditTrailStep{
			Stage: TrailStagePayment, Title: "Zahlung", Date: date,
			Reference: reference, Amount: allocation.SettledAmount,
			Detail: strings.TrimSpace(description + difference),
		})

		if allocation.BankTxID == nil || s.bankRepo == nil {
			continue
		}
		tx, err := s.bankRepo.FindByID(ctx, *allocation.BankTxID)
		if err != nil || tx == nil {
			continue
		}
		trail.Steps = append(trail.Steps, AuditTrailStep{
			Stage: TrailStageBank, Title: "Bankumsatz", Date: tx.BookingDate,
			Reference: tx.EndToEndID, Amount: tx.Amount,
			Detail: strings.TrimSpace(tx.CounterpartyName + " – " + tx.RemittanceInfo),
		})
	}
	if trail.Note == "" {
		trail.Note = "Der Pfad ist vollständig: Beleg, Buchung und Zahlung sind verbunden."
	}
	return trail, nil
}

// entryAmount ist die Summe der Sollseite einer Buchung — der Betrag, in dem
// sie im Journal steht.
func entryAmount(entry *domain.JournalEntry) domain.Cents {
	var sum domain.Cents
	for i := range entry.Lines {
		if entry.Lines[i].Side == domain.SideDebit {
			sum += entry.Lines[i].Amount
		}
	}
	return sum
}

// ExportCSV schreibt den Prüfpfad als CSV und liefert den Pfad.
func (s *AuditTrailService) ExportCSV(ctx context.Context, receiptID uint, path string) (string, error) {
	trail, err := s.Trail(ctx, receiptID)
	if err != nil {
		return "", err
	}
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("die Datei %s konnte nicht angelegt werden: %w", path, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	// Semikolon wie in den übrigen Ausgaben: die Tabellenkalkulation im
	// deutschsprachigen Raum erwartet es, und ein Komma trennt hier die
	// Nachkommastellen.
	writer.Comma = ';'
	rows := [][]string{
		{"Stufe", "Bezeichnung", "Datum", "Beleg_oder_Nummer", "Betrag", "Erläuterung"},
	}
	for _, step := range trail.Steps {
		rows = append(rows, []string{
			string(step.Stage), step.Title, step.Date, step.Reference,
			step.Amount.Decimal(), step.Detail,
		})
	}
	if err := writer.WriteAll(rows); err != nil {
		return "", fmt.Errorf("der Prüfpfad konnte nicht geschrieben werden: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	s.log(ctx, trail, path)
	return path, nil
}

// ExportPDF setzt den Prüfpfad als PDF und liefert den Pfad.
func (s *AuditTrailService) ExportPDF(ctx context.Context, receiptID uint, path string) (string, error) {
	if s.renderer == nil {
		return "", fmt.Errorf("der Dokumentensetzer ist nicht verfügbar; der Prüfpfad lässt sich als CSV ausgeben")
	}
	trail, err := s.Trail(ctx, receiptID)
	if err != nil {
		return "", err
	}
	title := fmt.Sprintf("Prüfpfad zu Beleg %s", trail.ReceiptNumber)
	pdf, err := s.renderer.RenderDocumentPDF(ctx,
		procdoc.Typst(auditTrailMarkdown(trail), title, time.Now()), title)
	if err != nil {
		return "", fmt.Errorf("der Prüfpfad konnte nicht gesetzt werden: %w", err)
	}
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		return "", fmt.Errorf("die Datei %s konnte nicht geschrieben werden: %w", path, err)
	}
	s.log(ctx, trail, path)
	return path, nil
}

// auditTrailMarkdown schreibt den Prüfpfad als Text.
func auditTrailMarkdown(trail *AuditTrail) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Prüfpfad zu Beleg %s\n\n", trail.ReceiptNumber)
	fmt.Fprintf(&b, "Aussteller: %s\n\n", orDash(trail.IssuerName))
	fmt.Fprintf(&b, "Belegdatum: %s, Bruttobetrag %s €\n\n",
		germanDate(trail.DocumentDate), trail.GrossAmount)
	if trail.OrderReference != "" {
		fmt.Fprintf(&b, "Bestellbezug: %s\n\n", trail.OrderReference)
	}
	if trail.ServiceProof != "" {
		fmt.Fprintf(&b, "Leistungsnachweis (%s): %s\n\n",
			germanDate(trail.ServiceProofAt), trail.ServiceProof)
	}

	b.WriteString("| Stufe | Datum | Nummer | Betrag | Erläuterung |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, step := range trail.Steps {
		fmt.Fprintf(&b, "| %s | %s | %s | %s € | %s |\n",
			step.Title, germanDate(step.Date), orDash(step.Reference), step.Amount, orDash(step.Detail))
	}
	b.WriteString("\n")
	if trail.Note != "" {
		fmt.Fprintf(&b, "%s\n", trail.Note)
	}
	return b.String()
}

func (s *AuditTrailService) log(ctx context.Context, trail *AuditTrail, path string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "RECEIPT",
		fmt.Sprintf("%d", trail.ReceiptID),
		fmt.Sprintf("Prüfpfad zu Beleg %s ausgegeben (%d Stufen) nach %s",
			trail.ReceiptNumber, len(trail.Steps), path))
}
