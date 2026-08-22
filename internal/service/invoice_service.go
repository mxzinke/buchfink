package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
)

// InvoiceService manages outgoing invoices, their booking and the ZUGFeRD /
// Typst rendering.
type InvoiceService struct {
	invoiceRepo  domain.InvoiceRepository
	contactRepo  domain.ContactRepository
	settingsRepo domain.SettingsRepository
	numberRepo   domain.NumberRangeRepository
	postingSvc   *PostingService
	auditRepo    domain.AuditRepository
}

// NewInvoiceService creates the invoice service.
func NewInvoiceService(
	invoiceRepo domain.InvoiceRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	numberRepo domain.NumberRangeRepository,
	postingSvc *PostingService,
	auditRepo domain.AuditRepository,
) *InvoiceService {
	return &InvoiceService{
		invoiceRepo:  invoiceRepo,
		contactRepo:  contactRepo,
		settingsRepo: settingsRepo,
		numberRepo:   numberRepo,
		postingSvc:   postingSvc,
		auditRepo:    auditRepo,
	}
}

// GetInvoices returns the invoices of a fiscal year.
func (s *InvoiceService) GetInvoices(ctx context.Context, fiscalYear int) ([]domain.Invoice, error) {
	return s.invoiceRepo.FindAll(ctx, fiscalYear)
}

// Issue finalises an invoice: it assigns the consecutive invoice number, books
// the receivable and stores the document.
//
// Issuing and booking are one step. § 14 Abs. 4 Nr. 4 UStG requires the number
// to be unique and consecutive, and GoBD requires the transaction to be recorded
// when it happens — so there is no state in which an invoice exists on paper but
// not in the journal.
func (s *InvoiceService) Issue(ctx context.Context, inv *domain.Invoice) error {
	if inv.Status == domain.InvoiceStatusIssued || inv.JournalEntryID != nil {
		return fmt.Errorf("Rechnung %s ist bereits ausgestellt und gebucht", inv.InvoiceNumber)
	}

	if inv.Date == "" {
		inv.Date = time.Now().Format("2006-01-02")
	}
	if inv.ServiceDateFrom == "" {
		inv.ServiceDateFrom = inv.Date
	}
	if inv.ServiceDateTo == "" {
		inv.ServiceDateTo = inv.ServiceDateFrom
	}
	if inv.Currency == "" {
		inv.Currency = "EUR"
	}
	if inv.TaxTreatment == "" {
		inv.TaxTreatment = domain.TaxTreatmentDomestic
	}
	if inv.FiscalYear == 0 {
		inv.FiscalYear = domain.GetFiscalYearForDate(inv.Date, s.fiscalYearStartMonth(ctx))
	}

	contact, err := s.contactRepo.FindByID(ctx, inv.ContactID)
	if err != nil {
		return fmt.Errorf("Rechnungsempfänger konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeCustomer {
		return fmt.Errorf("%s ist als Lieferant angelegt und kann keine Ausgangsrechnung erhalten", contact.Name)
	}
	inv.ContactName = contact.Name

	if inv.DueDate == "" {
		days := contact.PaymentTermsDays
		if days <= 0 {
			days = 14
		}
		if t, err := time.Parse("2006-01-02", inv.Date); err == nil {
			inv.DueDate = t.AddDate(0, 0, days).Format("2006-01-02")
		} else {
			inv.DueDate = inv.Date
		}
	}

	for i := range inv.Items {
		if inv.Items[i].Position == 0 {
			inv.Items[i].Position = i + 1
		}
	}
	inv.Recalculate()

	if err := inv.Validate(); err != nil {
		return err
	}
	if err := s.validateTaxTreatment(inv, contact); err != nil {
		return err
	}

	if inv.InvoiceNumber == "" {
		seq, err := s.numberRepo.Allocate(ctx, domain.NumberRangeInvoice, inv.FiscalYear)
		if err != nil {
			return fmt.Errorf("Rechnungsnummer konnte nicht vergeben werden: %w", err)
		}
		inv.InvoiceNumber = domain.FormatInvoiceNumber(inv.FiscalYear, seq)
	}

	entry, err := s.postingSvc.PostOutgoingInvoice(ctx, inv, contact)
	if err != nil {
		return err
	}
	inv.JournalEntryID = &entry.ID
	inv.Status = domain.InvoiceStatusIssued

	if err := s.invoiceRepo.Save(ctx, inv); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionCreate, "INVOICE", fmt.Sprintf("%d", inv.ID),
			fmt.Sprintf("Rechnung %s über %s %s an %s ausgestellt und als %s gebucht",
				inv.InvoiceNumber, inv.GrossAmount, inv.Currency, contact.Name, entry.EntryNumber))
	}
	return nil
}

// validateTaxTreatment blocks the combinations that would produce a formally
// wrong invoice — the ones a supplier only finds out about during an audit.
func (s *InvoiceService) validateTaxTreatment(inv *domain.Invoice, contact *domain.Contact) error {
	switch inv.TaxTreatment {
	case domain.TaxTreatmentIntraCommunitySupply:
		if !contact.IsEUCounterparty() {
			return fmt.Errorf("eine innergemeinschaftliche Lieferung setzt einen Empfänger in einem anderen EU-Land voraus, %s ist in %q erfasst", contact.Name, contact.CountryCode)
		}
		if contact.VatID == "" {
			return fmt.Errorf("für eine innergemeinschaftliche Lieferung braucht %s eine USt-IdNr. (§ 6a Abs. 1 Nr. 4 UStG)", contact.Name)
		}
	case domain.TaxTreatmentReverseChargeSupply:
		if contact.VatID == "" {
			return fmt.Errorf("für eine Rechnung nach § 13b UStG braucht %s eine USt-IdNr.", contact.Name)
		}
	case domain.TaxTreatmentExport:
		if contact.IsEUCounterparty() || contact.CountryCode == "DE" || contact.CountryCode == "" {
			return fmt.Errorf("eine Ausfuhrlieferung setzt einen Empfänger außerhalb der EU voraus, %s ist in %q erfasst", contact.Name, contact.CountryCode)
		}
	}
	return nil
}

// Preview computes what an invoice would book, without issuing it.
//
// It applies the same defaults Issue does, so the numbers the form shows are the
// numbers that will be booked. The invoice form used to compute net, tax and
// gross itself, in a second implementation of the rounding rules — this replaces
// it, because two implementations of a tax computation are one too many.
func (s *InvoiceService) Preview(ctx context.Context, inv *domain.Invoice) (*PostingPreview, error) {
	contact, err := s.contactRepo.FindByID(ctx, inv.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Rechnungsempfänger konnte nicht geladen werden: %w", err)
	}
	if contact.Type != domain.ContactTypeCustomer {
		return nil, fmt.Errorf("%s ist als Lieferant angelegt und kann keine Ausgangsrechnung erhalten", contact.Name)
	}

	draft := *inv
	if draft.TaxTreatment == "" {
		draft.TaxTreatment = domain.TaxTreatmentDomestic
	}
	for i := range draft.Items {
		if draft.Items[i].Position == 0 {
			draft.Items[i].Position = i + 1
		}
	}
	if err := s.validateTaxTreatment(&draft, contact); err != nil {
		return nil, err
	}

	return s.postingSvc.PreviewOutgoingInvoice(ctx, &draft, contact)
}

// Cancel reverses an issued invoice by Generalumkehr.
func (s *InvoiceService) Cancel(ctx context.Context, invoiceID uint, reason string) error {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if inv.Status == domain.InvoiceStatusCancelled {
		return fmt.Errorf("Rechnung %s ist bereits storniert", inv.InvoiceNumber)
	}
	if inv.JournalEntryID == nil {
		return fmt.Errorf("Rechnung %s ist nicht gebucht und kann nicht storniert werden", inv.InvoiceNumber)
	}

	if _, err := s.postingSvc.journalSvc.Reverse(ctx, *inv.JournalEntryID, reason); err != nil {
		return err
	}
	return s.invoiceRepo.UpdateStatus(ctx, invoiceID, domain.InvoiceStatusCancelled)
}

// GenerateZUGFeRDAndTypst produces the Factur-X / ZUGFeRD XML and the Typst
// source of an invoice.
func (s *InvoiceService) GenerateZUGFeRDAndTypst(ctx context.Context, invoiceID uint) (xml string, typst string, err error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return "", "", fmt.Errorf("Rechnung %d wurde nicht gefunden: %w", invoiceID, err)
	}

	seller, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return "", "", fmt.Errorf("Unternehmensdaten konnten nicht geladen werden: %w", err)
	}

	buyer, err := s.contactRepo.FindByID(ctx, inv.ContactID)
	if err != nil || buyer == nil {
		buyer = &domain.Contact{Name: inv.ContactName, CountryCode: "DE"}
	}

	xml, err = invoice.GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		return "", "", fmt.Errorf("ZUGFeRD-XML konnte nicht erzeugt werden: %w", err)
	}

	typst = invoice.GenerateTypstTemplate(inv, seller, buyer)
	// TODO: Compile Typst template to PDF using typst CLI or pure-Go typst compiler
	// TODO: Embed Factur-X / ZUGFeRD XML into PDF/A-3 as attachment

	return xml, typst, nil
}

func (s *InvoiceService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FiscalYearStartMonth <= 0 {
		return 1
	}
	return cfg.FiscalYearStartMonth
}
