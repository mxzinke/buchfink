package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Abschlussbaustein „Wertberichtigung auf Forderungen" (§ 253 Abs. 4 HGB).
//
// Gebucht wird die Veränderung der vier Wertberichtigungskonten auf ihren
// Sollbestand zum Stichtag: was hinzukommt, über 6923 und 6920, was entfällt,
// über 4923 und 4920. Eine Einzelwertberichtigung, deren Grund im Folgejahr
// entfallen ist (Zahlung, Ausbuchung), löst so der nächste Abschluss auf.

// ReceivableAllowanceReference ist die Belegnummer der Buchung.
func ReceivableAllowanceReference(year int) string { return fmt.Sprintf("WB %d", year) }

// ReceivableAllowanceRequest sind die Entscheidungen über die Forderungen.
type ReceivableAllowanceRequest struct {
	FiscalYear int `json:"fiscalYear"`
	// Specific sind die Einzelwertberichtigungen je offenem Posten.
	Specific []accounting.SpecificAllowance `json:"specific"`
	// BandPercents setzt einen Ausfall je Altersband für Posten ohne eigene
	// Angabe.
	BandPercents map[domain.AgingBucketKey]int64 `json:"bandPercents,omitempty"`
	// GeneralPermille ist die Quote der Pauschalwertberichtigung.
	GeneralPermille int64  `json:"generalPermille"`
	Reason          string `json:"reason"`
}

// AllowanceAccountRow ist ein Wertberichtigungskonto vor und nach der Buchung,
// als Habensaldo.
type AllowanceAccountRow struct {
	Account string       `json:"account"`
	Name    string       `json:"name"`
	Current domain.Cents `json:"current"`
	Target  domain.Cents `json:"target"`
	Change  domain.Cents `json:"change"`
}

// LossYear sind Forderungsverluste und Umsatzerlöse eines Vorjahres.
type LossYear struct {
	Year    int          `json:"year"`
	Losses  domain.Cents `json:"losses"`
	Revenue domain.Cents `json:"revenue"`
}

// ReceivableAllowancePreview ist der Vorschlag mit Buchungssatz.
type ReceivableAllowancePreview struct {
	FiscalYear int    `json:"fiscalYear"`
	Cutoff     string `json:"cutoff"`
	accounting.AllowanceResult
	// ProposedPermille ist die eigene Ausfallquote der bis zu drei Vorjahre,
	// der belegbare Vorschlag für die Pauschalwertberichtigung.
	ProposedPermille int64                 `json:"proposedPermille"`
	LossHistory      []LossYear            `json:"lossHistory"`
	Accounts         []AllowanceAccountRow `json:"accounts"`
	Lines            []domain.JournalLine  `json:"lines"`
	Explanation      string                `json:"explanation"`
	Notes            []string              `json:"notes"`
}

// PreviewReceivableAllowance bewertet die offenen Forderungen zum Stichtag und
// schlägt aus ihrer Altersstruktur die Kandidaten der Einzelwertberichtigung
// und aus den eigenen Ausfällen die Quote der Pauschalwertberichtigung vor.
func (s *ClosingBookingService) PreviewReceivableAllowance(
	ctx context.Context, req ReceivableAllowanceRequest,
) (*ReceivableAllowancePreview, error) {
	if err := requireYear(req.FiscalYear); err != nil {
		return nil, err
	}
	if s.openItems == nil {
		return nil, fmt.Errorf("die offenen Posten sind für die Wertberichtigung nicht verfügbar")
	}
	fy, err := s.periods.PeriodOf(ctx, req.FiscalYear)
	if err != nil {
		return nil, err
	}
	items, err := s.openItems.OpenItemsAt(ctx, fy.EndDate)
	if err != nil {
		return nil, err
	}
	result, err := accounting.CalculateAllowance(accounting.AllowanceInput{
		Cutoff: fy.EndDate, Items: items, Specific: req.Specific,
		BandPercents: req.BandPercents, GeneralPermille: req.GeneralPermille,
	})
	if err != nil {
		return nil, err
	}
	current, err := s.carriedBalances(ctx, req.FiscalYear, fy.EndDate, accounting.AllowanceAccounts())
	if err != nil {
		return nil, err
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	out := &ReceivableAllowancePreview{
		FiscalYear: req.FiscalYear, Cutoff: fy.EndDate, AllowanceResult: *result,
		LossHistory: make([]LossYear, 0, 3), Accounts: make([]AllowanceAccountRow, 0, 4),
		Lines: make([]domain.JournalLine, 0, 4), Notes: make([]string, 0, 4),
	}
	targets := result.Targets()
	for _, account := range accounting.AllowanceAccounts() {
		row := AllowanceAccountRow{
			Account: account, Name: chart.Name(account),
			Current: current[account], Target: targets[account],
		}
		row.Change = row.Target - row.Current
		out.Accounts = append(out.Accounts, row)
		if row.Change == 0 {
			continue
		}
		expense, income := accounting.AccountPWBExpense, accounting.AccountPWBIncome
		if account == accounting.AccountEWBShortTerm || account == accounting.AccountEWBLongTerm {
			expense, income = accounting.AccountEWBExpense, accounting.AccountEWBIncome
		}
		if row.Change > 0 {
			out.Lines = append(out.Lines,
				domain.JournalLine{Side: domain.SideDebit, Account: expense, Amount: row.Change,
					Text: "Einstellung in die Wertberichtigung"},
				domain.JournalLine{Side: domain.SideCredit, Account: account, Amount: row.Change,
					Text: "Wertberichtigung auf Forderungen"})
		} else {
			out.Lines = append(out.Lines,
				domain.JournalLine{Side: domain.SideDebit, Account: account, Amount: -row.Change,
					Text: "Herabsetzung der Wertberichtigung"},
				domain.JournalLine{Side: domain.SideCredit, Account: income, Amount: -row.Change,
					Text: "Ertrag aus der Herabsetzung der Wertberichtigung"})
		}
	}

	if err := s.lossHistory(ctx, req.FiscalYear, out); err != nil {
		return nil, err
	}
	candidates := 0
	for _, item := range result.Items {
		if item.Candidate && item.Percent == 0 {
			candidates++
		}
	}
	if candidates > 0 {
		out.Notes = append(out.Notes, fmt.Sprintf(
			"%d überfällige Forderungen ohne Einzelwertberichtigung: der erwartete Ausfall ist je Posten "+
				"zu entscheiden (§ 252 Abs. 1 Nr. 3 und 4 HGB); Zahlungen bis zur Aufstellung zeigen, dass "+
				"am Stichtag kein Ausfallrisiko in dieser Höhe bestand.", candidates))
	}
	out.Notes = append(out.Notes,
		"Bemessen wird der Nettobetrag; die Umsatzsteuer wird erst bei Uneinbringlichkeit berichtigt "+
			"(§ 17 Abs. 2 Nr. 1 UStG). Die Forderung bleibt auf dem Personenkonto stehen.",
		"Steuerlich gilt die Wertberichtigung nur bei voraussichtlich dauernder Wertminderung "+
			"(§ 6 Abs. 1 Nr. 2 Satz 2 EStG); die Quote der Pauschalwertberichtigung muss sich aus eigenen "+
			"Ausfällen belegen lassen.")
	out.Explanation = fmt.Sprintf(
		"Einzelwertberichtigung %s € (Restlaufzeit bis ein Jahr) und %s € (darüber); "+
			"Pauschalwertberichtigung %d ‰ auf %s € netto = %s € und %s €.",
		result.SpecificShort, result.SpecificLong, result.GeneralPermille, result.GeneralBase,
		result.GeneralShort, result.GeneralLong)
	return out, nil
}

// lossHistory trägt die Forderungsverluste und Umsätze der bis zu drei
// Vorjahre ein und rechnet daraus die vorgeschlagene Quote.
func (s *ClosingBookingService) lossHistory(ctx context.Context, year int, out *ReceivableAllowancePreview) error {
	var losses, revenue domain.Cents
	for y := year - 3; y < year; y++ {
		if y <= 0 {
			continue
		}
		count, err := s.journalRepo.Count(ctx, y)
		if err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		turnovers, err := s.journalRepo.AccountTurnovers(ctx, y)
		if err != nil {
			return err
		}
		row := LossYear{Year: y}
		for _, account := range accounting.ReceivableLossAccounts() {
			t := turnovers[account]
			row.Losses += t.Debit - t.Credit
		}
		if s.revenue != nil {
			if row.Revenue, err = s.revenue.RevenueOf(ctx, y); err != nil {
				return err
			}
		}
		losses += row.Losses
		revenue += row.Revenue
		out.LossHistory = append(out.LossHistory, row)
	}
	out.ProposedPermille = accounting.LossRatePermille(losses, revenue)
	if out.ProposedPermille == 0 {
		out.Notes = append(out.Notes,
			"Aus den Vorjahren ergibt sich keine eigene Ausfallquote; eine Pauschalwertberichtigung lässt "+
				"sich so nicht belegen.")
	}
	return nil
}

// carriedBalances sind die Habensalden der Konten zum Stichtag. Haben die
// Konten zum Schluss des Vorjahres einen anderen Bestand, als vorgetragen
// wurde, fehlt der Saldenvortrag: die Veränderung wäre dann gegen einen
// falschen Anfangsbestand gerechnet.
func (s *ClosingBookingService) carriedBalances(
	ctx context.Context, year int, cutoff string, accounts []string,
) (map[string]domain.Cents, error) {
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, cutoff)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Cents, len(accounts))
	for _, account := range accounts {
		t := turnovers[account]
		out[account] = t.Credit - t.Debit
	}

	prior := year - 1
	count, err := s.journalRepo.Count(ctx, prior)
	if err != nil || count == 0 || prior <= 0 {
		return out, err
	}
	priorPeriod, err := s.periods.PeriodOf(ctx, prior)
	if err != nil {
		return nil, err
	}
	closing, err := s.journalRepo.AccountTurnoversUntil(ctx, prior, priorPeriod.EndDate)
	if err != nil {
		return nil, err
	}
	carried, _, err := carriedSoFar(ctx, s.journalRepo, year)
	if err != nil {
		return nil, err
	}
	var gaps []string
	for _, account := range accounts {
		t := closing[account]
		if closed, opened := t.Credit-t.Debit, -carried[account]; closed != opened {
			gaps = append(gaps, fmt.Sprintf("%s schließt %d mit %s €, vorgetragen sind %s €",
				account, prior, closed, opened))
		}
	}
	if len(gaps) > 0 {
		return nil, fmt.Errorf(
			"der Saldenvortrag nach %d fehlt oder ist nicht aktuell (%s). Gebucht wird die Veränderung "+
				"gegen den Bestand; zuerst den Saldenvortrag ausführen", year, strings.Join(gaps, "; "))
	}
	return out, nil
}

// BookReceivableAllowance bucht die Wertberichtigung.
func (s *ClosingBookingService) BookReceivableAllowance(
	ctx context.Context, req ReceivableAllowanceRequest,
) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.bookReceivableAllowance(ctx, req)
	})
}

func (s *ClosingBookingService) bookReceivableAllowance(
	ctx context.Context, req ReceivableAllowanceRequest,
) (*domain.JournalEntry, error) {
	preview, err := s.PreviewReceivableAllowance(ctx, req)
	if err != nil {
		return nil, err
	}
	reference := ReceivableAllowanceReference(preview.FiscalYear)
	if err := refuseSecondClosingEntry(ctx, s.journalRepo, preview.FiscalYear, reference,
		"Die Wertberichtigung auf Forderungen"); err != nil {
		return nil, err
	}
	if len(preview.Lines) == 0 {
		return nil, fmt.Errorf(
			"die Wertberichtigungen stehen zum %s bereits in dieser Höhe; es gibt nichts zu buchen",
			germanDate(preview.Cutoff))
	}
	for _, spec := range req.Specific {
		if spec.Percent > 0 && strings.TrimSpace(spec.Reason) == "" {
			return nil, fmt.Errorf(
				"die Einzelwertberichtigung der Buchung %d braucht einen Grund: er belegt das Ausfallrisiko",
				spec.EntryID)
		}
	}
	explanation := preview.Explanation
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		explanation = reason + " " + explanation
	}
	entry := &domain.JournalEntry{
		BookingDate: preview.Cutoff, DocumentDate: preview.Cutoff,
		ServiceDateFrom: preview.Cutoff, ServiceDateTo: preview.Cutoff,
		Description:        fmt.Sprintf("Wertberichtigung auf Forderungen %d (§ 253 Abs. 4 HGB)", preview.FiscalYear),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     reference,
		TaxTreatment:       domain.TaxTreatmentNotTaxable,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              preview.Lines,
	}
	receipt, err := selfIssuedVoucher(ctx, s.receipts, preview.FiscalYear, closingVoucher{
		Kind: "wertberichtigung", FiscalYear: preview.FiscalYear, Date: preview.Cutoff,
		Description: entry.Description, Explanation: explanation,
		Calculation: struct {
			Request  ReceivableAllowanceRequest `json:"angaben"`
			Result   accounting.AllowanceResult `json:"bewertung"`
			Accounts []AllowanceAccountRow      `json:"konten"`
			History  []LossYear                 `json:"ausfaelle_vorjahre"`
		}{req, preview.AllowanceResult, preview.Accounts, preview.LossHistory},
		Lines: preview.Lines,
	})
	if err != nil {
		return nil, err
	}
	attachVoucher(entry, receipt)
	entry.DocumentNumber = reference
	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}
	if err := s.auditEntity(ctx, domain.AuditActionCreate, "RECEIVABLE_ALLOWANCE", preview.FiscalYear, fmt.Sprintf(
		"Wertberichtigung auf Forderungen %d (%s): EWB %s € und %s €, PWB %s € und %s €",
		preview.FiscalYear, created.EntryNumber, preview.SpecificShort, preview.SpecificLong,
		preview.GeneralShort, preview.GeneralLong)); err != nil {
		return nil, err
	}
	return created, nil
}
