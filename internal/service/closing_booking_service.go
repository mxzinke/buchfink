package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// ClosingBookingService führt die Abschlussbausteine, die keine eigene Kartei
// haben: Umsatzsteuer-Verrechnung, Steuerrückstellung, Wertberichtigung auf
// Forderungen und unfertige Leistungen.
//
// Sie stehen zusammen, weil sie dasselbe Muster teilen: sie lesen Kontensalden
// und Angaben, rechnen daraus Beträge, zeigen sie als Vorschau und buchen sie
// erst auf Freigabe, mit einem Eigenbeleg über die Rechnung. Was sie
// unterscheidet, ist nur die Rechnung dazwischen.
type ClosingBookingService struct {
	auditor[int]
	provisionRepo domain.ProvisionRepository
	journalRepo   domain.JournalRepository
	journalSvc    *JournalService
	settingsRepo  domain.SettingsRepository
	periods       PeriodSource
	receipts      closingReceiptFiler
	// openItems und revenue tragen die Wertberichtigung auf Forderungen: die
	// offenen Posten zum Stichtag und die Umsätze der Vorjahre für die Quote.
	openItems openItemSource
	revenue   RevenueSource
}

// NewClosingBookingService verdrahtet die Bausteine.
func NewClosingBookingService(
	provisionRepo domain.ProvisionRepository,
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	periods PeriodSource,
) *ClosingBookingService {
	return &ClosingBookingService{
		provisionRepo: provisionRepo, journalRepo: journalRepo,
		journalSvc: journalSvc, settingsRepo: settingsRepo, auditor: auditor[int]{auditRepo: auditRepo},
		periods: periods,
	}
}

// SetReceiptService gibt dem Dienst den Belegspeicher für die Eigenbelege.
func (s *ClosingBookingService) SetReceiptService(r closingReceiptFiler) { s.receipts = r }

// SetOpenItemSource gibt der Wertberichtigung die offenen Posten zum Stichtag.
func (s *ClosingBookingService) SetOpenItemSource(src openItemSource) { s.openItems = src }

// SetRevenueSource gibt der Wertberichtigung die Umsatzerlöse der Vorjahre.
func (s *ClosingBookingService) SetRevenueSource(src RevenueSource) { s.revenue = src }

// -------------------------------------------------------------------------
// Umsatzsteuer-Verrechnung
// -------------------------------------------------------------------------

// VatSettlementReference ist die Belegnummer der Verrechnungsbuchung. Über sie
// erkennt der Assistent, ob der Schritt getan ist.
func VatSettlementReference(year int) string { return accounting.VatSettlementReference(year) }

// VatSettlementRow ist ein Steuerkonto mit seinem Jahressaldo.
type VatSettlementRow struct {
	Account     string `json:"account"`
	AccountName string `json:"accountName"`
	// Balance ist der Saldo in Soll-Richtung: Vorsteuer positiv, Umsatzsteuer
	// negativ.
	Balance domain.Cents `json:"balance"`
}

// VatSettlement ist die Vorschau der Jahresverrechnung.
type VatSettlement struct {
	FiscalYear int                `json:"fiscalYear"`
	Cutoff     string             `json:"cutoff"`
	Rows       []VatSettlementRow `json:"rows"`
	InputTax   domain.Cents       `json:"inputTax"`
	OutputTax  domain.Cents       `json:"outputTax"`
	Prepaid    domain.Cents       `json:"prepaid"`
	// Payable ist die Zahllast (auf 3841), Refund die Erstattung (auf 1420).
	// Immer nur eines von beiden.
	Payable     domain.Cents         `json:"payable"`
	Refund      domain.Cents         `json:"refund"`
	Lines       []domain.JournalLine `json:"lines"`
	BookingDate string               `json:"bookingDate"`
	Explanation string               `json:"explanation"`
}

// PreviewVatSettlement rechnet die Jahresverrechnung der Umsatzsteuer.
//
// Am Bilanzstichtag stehen auf den Steuerkonten die Summen des ganzen Jahres:
// die abziehbare Vorsteuer im Soll, die geschuldete Umsatzsteuer im Haben, die
// geleisteten Vorauszahlungen im Soll. Was übrigbleibt, ist die Zahllast oder
// die Erstattung der Jahreserklärung — und genau sie gehört in die Bilanz, nicht
// die drei Salden nebeneinander.
func (s *ClosingBookingService) PreviewVatSettlement(ctx context.Context, year int) (*VatSettlement, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	fy, err := s.periods.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, fy.EndDate)
	if err != nil {
		return nil, err
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	out := &VatSettlement{
		FiscalYear: year, Cutoff: fy.EndDate, BookingDate: fy.EndDate,
		Rows: make([]VatSettlementRow, 0), Lines: make([]domain.JournalLine, 0),
	}
	var total domain.Cents
	for _, account := range accounting.VatSettlementAccounts() {
		turnover, ok := turnovers[account]
		if !ok {
			continue
		}
		balance := turnover.Debit - turnover.Credit
		if balance == 0 {
			continue
		}
		out.Rows = append(out.Rows, VatSettlementRow{
			Account: account, AccountName: chart.Name(account), Balance: balance,
		})
		switch account {
		case domain.AccountUmsatzsteuerVorauszahlungen, domain.AccountSondervorauszahlung:
			out.Prepaid += balance
		default:
			if balance > 0 {
				out.InputTax += balance
			} else {
				out.OutputTax += -balance
			}
		}
		total += balance
		// Jedes Konto wird auf null gestellt: die Gegenseite seines Saldos.
		side := domain.SideCredit
		amount := balance
		if balance < 0 {
			side, amount = domain.SideDebit, -balance
		}
		out.Lines = append(out.Lines, domain.JournalLine{
			Side: side, Account: account, Amount: amount,
			Text: fmt.Sprintf("Jahresverrechnung %d", year),
		})
	}

	switch {
	case total < 0:
		out.Payable = -total
		out.Lines = append(out.Lines, domain.JournalLine{
			Side: domain.SideCredit, Account: domain.AccountUmsatzsteuerVorjahr,
			Amount: out.Payable, Text: fmt.Sprintf("Umsatzsteuer-Zahllast %d", year),
		})
		out.Explanation = fmt.Sprintf(
			"Umsatzsteuer %s € abzüglich Vorsteuer %s € und Vorauszahlungen %s € ergibt eine Zahllast "+
				"von %s €. Sie steht als Verbindlichkeit auf %s.",
			out.OutputTax, out.InputTax, out.Prepaid, out.Payable, domain.AccountUmsatzsteuerVorjahr)
	case total > 0:
		out.Refund = total
		out.Lines = append(out.Lines, domain.JournalLine{
			Side: domain.SideDebit, Account: domain.AccountUmsatzsteuerforderung,
			Amount: out.Refund, Text: fmt.Sprintf("Umsatzsteuer-Erstattung %d", year),
		})
		out.Explanation = fmt.Sprintf(
			"Vorsteuer %s € und Vorauszahlungen %s € übersteigen die Umsatzsteuer von %s € um %s €. "+
				"Der Betrag steht als Forderung auf %s.",
			out.InputTax, out.Prepaid, out.OutputTax, out.Refund, domain.AccountUmsatzsteuerforderung)
	default:
		out.Explanation = "Die Steuerkonten gleichen sich aus; es bleibt weder Zahllast noch Erstattung."
	}
	return out, nil
}

// BookVatSettlement bucht die Jahresverrechnung.
func (s *ClosingBookingService) BookVatSettlement(
	ctx context.Context, year int,
) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.bookVatSettlement(ctx, year)
	})
}

func (s *ClosingBookingService) bookVatSettlement(
	ctx context.Context, year int,
) (*domain.JournalEntry, error) {
	settlement, err := s.PreviewVatSettlement(ctx, year)
	if err != nil {
		return nil, err
	}
	if len(settlement.Lines) == 0 {
		return nil, fmt.Errorf(
			"im Geschäftsjahr %d steht auf keinem Steuerkonto ein Saldo; es gibt nichts zu verrechnen",
			settlement.FiscalYear)
	}
	entry := &domain.JournalEntry{
		BookingDate: settlement.BookingDate, DocumentDate: settlement.BookingDate,
		ServiceDateFrom: settlement.BookingDate, ServiceDateTo: settlement.BookingDate,
		Description:        fmt.Sprintf("Umsatzsteuer-Jahresverrechnung %d", settlement.FiscalYear),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     VatSettlementReference(settlement.FiscalYear),
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              settlement.Lines,
	}
	receipt, err := selfIssuedVoucher(ctx, s.receipts, settlement.FiscalYear, closingVoucher{
		Kind: "umsatzsteuer-verrechnung", FiscalYear: settlement.FiscalYear,
		Date: settlement.BookingDate, Description: entry.Description,
		Explanation: settlement.Explanation, Calculation: settlement, Lines: settlement.Lines,
	})
	if err != nil {
		return nil, err
	}
	// Die Belegnummer der Verrechnung bleibt die Kennung des Schrittes; der
	// Eigenbeleg hängt daneben.
	reference := entry.DocumentNumber
	attachVoucher(entry, receipt)
	entry.DocumentNumber = reference

	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}
	if err := s.auditEntity(ctx, domain.AuditActionCreate, "VAT_SETTLEMENT", settlement.FiscalYear, fmt.Sprintf(
		"Umsatzsteuer-Jahresverrechnung %d: Zahllast %s €, Erstattung %s €",
		settlement.FiscalYear, settlement.Payable, settlement.Refund)); err != nil {
		return nil, err
	}
	return created, nil
}
