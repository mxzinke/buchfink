package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Abschlussbaustein „Unfertige Leistungen“.
//
// Am Stichtag wird der Bestand der laufenden Projekte aktiviert (1080 an 4815)
// und, wenn gewählt, die erhaltene Anzahlung von den Vorräten offen abgesetzt
// (Anzahlungskonto an 1190). Am ersten Tag des Folgejahres wird beides mit dem
// Saldenvortrag aufgelöst: die Bestandsminderung steht dem Erlös gegenüber, den
// die Schlussrechnung bringt, und die Schlussrechnung findet die Anzahlung auf
// ihrem Konto.

// WorkInProgressReference ist die Belegnummer der Buchung und ihrer Auflösung.
func WorkInProgressReference(year int) string { return fmt.Sprintf("UL %d", year) }

// WorkInProgressRequest sind die Projekte am Stichtag.
type WorkInProgressRequest struct {
	FiscalYear int                                `json:"fiscalYear"`
	Projects   []accounting.WorkInProgressProject `json:"projects"`
	// DeductAdvances setzt die erhaltenen Anzahlungen von den Vorräten offen
	// ab; sonst bleiben sie unter den Verbindlichkeiten (§ 268 Abs. 5 Satz 2
	// HGB).
	DeductAdvances bool   `json:"deductAdvances"`
	Reason         string `json:"reason"`
}

// WorkInProgressPreview ist die Bewertung mit Buchungssatz.
type WorkInProgressPreview struct {
	FiscalYear   int    `json:"fiscalYear"`
	Cutoff       string `json:"cutoff"`
	ReversalDate string `json:"reversalDate"`
	accounting.WorkInProgressResult
	DeductAdvances bool                 `json:"deductAdvances"`
	Lines          []domain.JournalLine `json:"lines"`
	Explanation    string               `json:"explanation"`
	Notes          []string             `json:"notes"`
}

// PreviewWorkInProgress bewertet die laufenden Projekte zum Stichtag.
func (s *ClosingBookingService) PreviewWorkInProgress(
	ctx context.Context, req WorkInProgressRequest,
) (*WorkInProgressPreview, error) {
	if err := requireYear(req.FiscalYear); err != nil {
		return nil, err
	}
	fy, err := s.closingSvc.PeriodOf(ctx, req.FiscalYear)
	if err != nil {
		return nil, err
	}
	reversal, err := dayAfterCutoff(fy.EndDate)
	if err != nil {
		return nil, err
	}
	result, err := accounting.CalculateWorkInProgress(req.Projects, req.DeductAdvances)
	if err != nil {
		return nil, err
	}
	out := &WorkInProgressPreview{
		FiscalYear: req.FiscalYear, Cutoff: fy.EndDate, ReversalDate: reversal,
		WorkInProgressResult: *result, DeductAdvances: req.DeductAdvances,
		Lines: make([]domain.JournalLine, 0, 4), Notes: make([]string, 0, 3),
	}
	if result.Total > 0 {
		out.Lines = append(out.Lines,
			domain.JournalLine{Side: domain.SideDebit, Account: accounting.AccountUnfertigeLeistungen,
				Amount: result.Total, Text: "Unfertige Leistungen"},
			domain.JournalLine{Side: domain.SideCredit, Account: accounting.AccountBestandsveraenderungUL,
				Amount: result.Total, Text: "Bestandsveränderung unfertige Leistungen"})
	}
	if len(result.AdvanceOffsets) > 0 {
		turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, req.FiscalYear, fy.EndDate)
		if err != nil {
			return nil, err
		}
		accounts := make([]string, 0, len(result.AdvanceOffsets))
		for account := range result.AdvanceOffsets {
			accounts = append(accounts, account)
		}
		sort.Strings(accounts)
		for _, account := range accounts {
			amount := result.AdvanceOffsets[account]
			if amount == 0 {
				continue
			}
			t := turnovers[account]
			if held := t.Credit - t.Debit; amount > held {
				return nil, fmt.Errorf(
					"auf %s stehen zum %s erhaltene Anzahlungen über %s €; abzusetzen sind %s €. "+
						"Abgesetzt werden kann nur, was als Anzahlung gebucht ist",
					account, germanDate(fy.EndDate), held, amount)
			}
			out.Lines = append(out.Lines,
				domain.JournalLine{Side: domain.SideDebit, Account: account, Amount: amount,
					Text: "Erhaltene Anzahlungen, von den Vorräten abgesetzt"},
				domain.JournalLine{Side: domain.SideCredit, Account: accounting.AccountAnzahlungenVonVorraetenOffen,
					Amount: amount, Text: "Erhaltene Anzahlungen auf Bestellungen"})
		}
	}
	if result.ExpectedLosses > 0 {
		out.Notes = append(out.Notes, fmt.Sprintf(
			"Drohende Verluste über den Bestand hinaus: %s €. Dafür ist eine Rückstellung für drohende "+
				"Verluste zu bilden (§ 249 Abs. 1 Satz 1 HGB); steuerlich ist sie nicht zulässig "+
				"(§ 5 Abs. 4a EStG).", result.ExpectedLosses))
	}
	for _, p := range result.Projects {
		if p.WriteDown > 0 {
			out.Notes = append(out.Notes, fmt.Sprintf(
				"%s: auf den beizulegenden Wert abgeschrieben (%s € unter den Herstellungskosten, "+
					"§ 253 Abs. 4 Satz 2 HGB).", p.Name, p.WriteDown))
		}
	}
	out.Notes = append(out.Notes, fmt.Sprintf(
		"Die Auflösung zum %s bucht der Saldenvortrag in das Geschäftsjahr %d.",
		germanDate(reversal), req.FiscalYear+1))
	out.Explanation = fmt.Sprintf(
		"%d Projekte zu Herstellungskosten (§ 255 Abs. 2 HGB), verlustfrei bewertet: Bestand %s €.",
		len(result.Projects), result.Total)
	return out, nil
}

// BookWorkInProgress bucht den Bestand der unfertigen Leistungen.
func (s *ClosingBookingService) BookWorkInProgress(
	ctx context.Context, req WorkInProgressRequest,
) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.bookWorkInProgress(ctx, req)
	})
}

func (s *ClosingBookingService) bookWorkInProgress(
	ctx context.Context, req WorkInProgressRequest,
) (*domain.JournalEntry, error) {
	preview, err := s.PreviewWorkInProgress(ctx, req)
	if err != nil {
		return nil, err
	}
	reference := WorkInProgressReference(preview.FiscalYear)
	if err := refuseSecondClosingEntry(ctx, s.journalRepo, preview.FiscalYear, reference,
		"Der Bestand der unfertigen Leistungen"); err != nil {
		return nil, err
	}
	if len(preview.Lines) == 0 {
		return nil, fmt.Errorf(
			"zum %s ist kein Bestand zu aktivieren. Läuft kein Projekt, wird der Schritt abgehakt",
			germanDate(preview.Cutoff))
	}
	explanation := preview.Explanation
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		explanation = reason + " " + explanation
	}
	entry := &domain.JournalEntry{
		BookingDate: preview.Cutoff, DocumentDate: preview.Cutoff,
		ServiceDateFrom: preview.Cutoff, ServiceDateTo: preview.Cutoff,
		Description:        fmt.Sprintf("Unfertige Leistungen %d (§ 255 Abs. 2 HGB)", preview.FiscalYear),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     reference,
		TaxTreatment:       domain.TaxTreatmentNotTaxable,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              preview.Lines,
	}
	receipt, err := selfIssuedVoucher(ctx, s.receipts, preview.FiscalYear, closingVoucher{
		Kind: "unfertige-leistungen", FiscalYear: preview.FiscalYear, Date: preview.Cutoff,
		Description: entry.Description, Explanation: explanation,
		Calculation: struct {
			Projects []accounting.WorkInProgressProject `json:"projekte"`
			Result   accounting.WorkInProgressResult    `json:"bewertung"`
			Deduct   bool                               `json:"anzahlungen_abgesetzt"`
		}{req.Projects, preview.WorkInProgressResult, req.DeductAdvances},
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
	if err := s.auditEntity(ctx, domain.AuditActionCreate, "WORK_IN_PROGRESS", preview.FiscalYear, fmt.Sprintf(
		"Unfertige Leistungen %d (%s): Bestand %s €, %d Projekte; Auflösung zum %s mit dem Saldenvortrag",
		preview.FiscalYear, created.EntryNumber, preview.Total, len(preview.Projects),
		preview.ReversalDate)); err != nil {
		return nil, err
	}
	return created, nil
}

// WorkInProgressReverser ist die Auflösung der unfertigen Leistungen, die der
// Saldenvortrag anstößt.
func (s *ClosingBookingService) WorkInProgressReverser() OpeningReverser {
	return workInProgressReverser{s}
}

type workInProgressReverser struct{ s *ClosingBookingService }

// ReverseInto bucht die Auflösung des Vorjahresbestands am ersten Tag des
// Zieljahres. Sie fragt das Journal: steht die Buchung des Vorjahres und fehlt
// die Auflösung, wird aufgelöst; ein erneuter Vortrag holt nach, was fehlt.
func (r workInProgressReverser) ReverseInto(ctx context.Context, toYear int) ([]domain.JournalEntry, error) {
	s := r.s
	created := make([]domain.JournalEntry, 0, 1)
	if toYear <= 1 {
		return created, nil
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) ([]domain.JournalEntry, error) {
		reference := WorkInProgressReference(toYear - 1)
		standing, err := standingClosingEntries(ctx, s.journalRepo, toYear-1, reference)
		if err != nil || len(standing) == 0 {
			return created, err
		}
		done, err := standingOpeningEntries(ctx, s.journalRepo, toYear, reference)
		if err != nil || len(done) > 0 {
			return created, err
		}
		for i := range standing {
			valuation := &standing[i]
			date, err := dayAfterCutoff(valuation.BookingDate)
			if err != nil {
				return nil, err
			}
			lines := make([]domain.JournalLine, 0, len(valuation.Lines))
			for _, l := range valuation.Lines {
				lines = append(lines, domain.JournalLine{
					Side: l.Side.Opposite(), Account: l.Account, Amount: l.Amount, Text: l.Text,
				})
			}
			posted, err := s.journalSvc.Post(ctx, &domain.JournalEntry{
				BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
				Description: fmt.Sprintf("Auflösung der unfertigen Leistungen zum %s",
					germanDate(valuation.BookingDate)),
				Source:             domain.EntrySourceOpening,
				DocumentNumber:     reference,
				TaxTreatment:       domain.TaxTreatmentNotTaxable,
				PostingRuleVersion: accounting.PostingRuleVersion,
				Lines:              lines,
			})
			if err != nil {
				return nil, fmt.Errorf("die unfertigen Leistungen %s ließen sich zum %s nicht auflösen: %w",
					valuation.EntryNumber, germanDate(date), err)
			}
			created = append(created, *posted)
		}
		if err := s.auditEntity(ctx, domain.AuditActionCreate, "WORK_IN_PROGRESS", toYear, fmt.Sprintf(
			"Unfertige Leistungen %d aufgelöst: %d Buchung(en)", toYear-1, len(created))); err != nil {
			return nil, err
		}
		return created, nil
	})
}
