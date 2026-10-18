package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// TaxProvisionReference ist die Belegnummer der Buchung, an der der Baustein
// „Steuerrückstellung" erkennt, ob er gebucht ist.
func TaxProvisionReference(year int) string { return fmt.Sprintf("STR %d", year) }

// TaxProvisionOptions sind die Angaben zur Steuerrückstellung, die nicht in den
// Büchern stehen. Sie stammen aus Bescheiden und Beschlüssen; der Eigenbeleg
// hält sie mit der Rechnung fest.
type TaxProvisionOptions struct {
	// TradeTaxRatePercent ist der Hebesatz laut Gewerbesteuerbescheid; null
	// heißt: der Hebesatz aus den Abschluss-Einstellungen.
	TradeTaxRatePercent int64 `json:"tradeTaxRatePercent,omitempty"`
	// LossCarryforwardCorporate ist der verbleibende Verlustvortrag zur
	// Körperschaftsteuer zum Schluss des Vorjahres (§ 10d Abs. 4 EStG),
	// LossCarryforwardTrade der vortragsfähige Gewerbeverlust (§ 10a Satz 6
	// GewStG), jeweils laut Feststellungsbescheid.
	LossCarryforwardCorporate domain.Cents `json:"lossCarryforwardCorporate"`
	LossCarryforwardTrade     domain.Cents `json:"lossCarryforwardTrade"`
	// ExemptDividends ersetzt die steuerfreien Bezüge nach § 8b Abs. 1 KStG,
	// die sonst aus den Konten 7005, 7006, 7015 und 7104 kommen. Bezüge auf
	// 7014 und 7103 gelten ohne diese Angabe als steuerpflichtig (§ 8b Abs. 4
	// KStG).
	ExemptDividends *domain.Cents `json:"exemptDividends,omitempty"`
	// IncomeAdjustments sind Korrekturen außerhalb der Bücher, die
	// Körperschaft- und Gewerbesteuer treffen: zuzurechnen eine verdeckte
	// Gewinnausschüttung (§ 8 Abs. 3 Satz 2 KStG) oder eine Geldbuße auf einem
	// gewöhnlichen Aufwandskonto, abzuziehen eine als Ertrag gebuchte verdeckte
	// Einlage (§ 8 Abs. 3 Satz 3 KStG).
	IncomeAdjustments domain.Cents `json:"incomeAdjustments"`
	// TradeAdditions und TradeDeductions sind Hinzurechnungen (§ 8 GewStG) und
	// Kürzungen (§ 9 GewStG).
	TradeAdditions  domain.Cents `json:"tradeAdditions"`
	TradeDeductions domain.Cents `json:"tradeDeductions"`
	// SkipWithholdingCredit rechnet die Kapitalertragsteuer nicht an, etwa
	// solange die Steuerbescheinigung fehlt (§ 36 Abs. 2 Nr. 2 Satz 2 EStG).
	SkipWithholdingCredit bool `json:"skipWithholdingCredit"`
}

// TaxAdjustmentRow ist ein Konto, dessen Saldo die Bemessungsgrundlage anders
// trifft als das Ergebnis.
type TaxAdjustmentRow struct {
	Account   string                       `json:"account"`
	Name      string                       `json:"name"`
	Kind      accounting.TaxAdjustmentKind `json:"kind"`
	Reference string                       `json:"reference"`
	// Amount ist der Saldo in der Richtung seiner Wirkung: ein Aufwand im
	// Soll, ein Ertrag im Haben, jeweils positiv.
	Amount domain.Cents `json:"amount"`
}

// TaxProvisionPreview ist der Vorschlag für die Steuerrückstellung.
type TaxProvisionPreview struct {
	FiscalYear int    `json:"fiscalYear"`
	Cutoff     string `json:"cutoff"`

	accounting.TaxProvisionResult
	Input   accounting.TaxProvisionInput `json:"input"`
	Options TaxProvisionOptions          `json:"options"`
	// TradeTaxRateSource sagt, woher der Hebesatz stammt: „einstellungen"
	// oder „eingabe".
	TradeTaxRateSource string `json:"tradeTaxRateSource"`
	// Adjustments sind die Konten mit steuerlicher Korrektur.
	Adjustments []TaxAdjustmentRow `json:"adjustments"`
	// PortfolioDividends sind Bezüge auf 7014 und 7103, die ohne Angabe als
	// steuerpflichtig gelten.
	PortfolioDividends domain.Cents `json:"portfolioDividends"`

	Lines       []domain.JournalLine `json:"lines"`
	Explanation string               `json:"explanation"`
	// Warning sagt ausdrücklich, dass die Rechnung eine Schätzung ist.
	Warning string `json:"warning"`
	// Notes nennen, was der Mensch prüfen oder angeben muss.
	Notes []string `json:"notes"`
}

// PreviewTaxProvision rechnet Körperschaftsteuer, Solidaritätszuschlag und
// Gewerbesteuer auf das Ergebnis des Jahres und stellt sie den Vorauszahlungen
// und der anrechenbaren Kapitalertragsteuer gegenüber.
func (s *ClosingBookingService) PreviewTaxProvision(
	ctx context.Context, year int, opts TaxProvisionOptions,
) (*TaxProvisionPreview, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	if err := ensureSupportedCompany(ctx, s.settingsRepo); err != nil {
		return nil, err
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	rate, rateSource, err := s.tradeTaxRate(ctx, opts)
	if err != nil {
		return nil, err
	}
	if opts.ExemptDividends != nil && *opts.ExemptDividends < 0 {
		return nil, fmt.Errorf("steuerfreie Bezüge können nicht negativ sein (%s €)", *opts.ExemptDividends)
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, fy.EndDate)
	if err != nil {
		return nil, err
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	preview := &TaxProvisionPreview{
		FiscalYear: year, Cutoff: fy.EndDate, Options: opts, TradeTaxRateSource: rateSource,
		Adjustments: make([]TaxAdjustmentRow, 0), Lines: make([]domain.JournalLine, 0, 6),
		Notes: make([]string, 0),
	}
	var profit, nonDeductible, dividends, gains, donations, turnoverAndWages domain.Cents
	for account, turnover := range turnovers {
		acc, ok := chart.Lookup(account)
		if !ok {
			continue
		}
		// Die Ertragsteuern gehören nicht in ihre eigene Bemessungsgrundlage —
		// weder als Aufwand noch als Erstattung. Vorauszahlungen und
		// Anrechnung liest taxPrepayments aus den Buchungen.
		if accounting.IsIncomeTaxAccount(account) || accounting.IsResultAppropriation(acc) {
			continue
		}
		if acc.Type != domain.AccountTypeRevenue && acc.Type != domain.AccountTypeExpense {
			continue
		}
		balance := turnover.Debit - turnover.Credit
		profit -= balance
		if accounting.IsRevenueAccount(acc) {
			turnoverAndWages -= balance
		} else if accounting.IsWageAccount(acc) {
			turnoverAndWages += balance
		}
		adj, ok := accounting.TaxAdjustmentFor(account)
		if !ok || balance == 0 {
			continue
		}
		row := TaxAdjustmentRow{Account: account, Name: acc.Name, Kind: adj.Kind, Reference: adj.Reference, Amount: balance}
		switch adj.Kind {
		case accounting.TaxAdjustmentNonDeductible:
			nonDeductible += balance
		case accounting.TaxAdjustmentDonation:
			donations += balance
		case accounting.TaxAdjustmentExemptDividend:
			row.Amount = -balance
			dividends -= balance
		case accounting.TaxAdjustmentPortfolioDividend:
			row.Amount = -balance
			preview.PortfolioDividends -= balance
		case accounting.TaxAdjustmentExemptGain:
			row.Amount = -balance
			gains -= balance
		}
		preview.Adjustments = append(preview.Adjustments, row)
	}
	sort.Slice(preview.Adjustments, func(i, j int) bool {
		return preview.Adjustments[i].Account < preview.Adjustments[j].Account
	})
	// Ein Verlust auf den Konten des § 8b Abs. 2 KStG ist eine Gewinnminderung
	// nach Abs. 3 Satz 3 und nicht abziehbar.
	if gains < 0 {
		nonDeductible += -gains
		gains = 0
	}
	if dividends < 0 {
		dividends = 0
	}
	if opts.ExemptDividends != nil {
		dividends = *opts.ExemptDividends
	}
	if donations < 0 {
		donations = 0
	}

	paid, err := s.taxPrepayments(ctx, year)
	if err != nil {
		return nil, err
	}
	input := accounting.TaxProvisionInput{
		ProfitBeforeTax: profit, NonDeductible: nonDeductible + opts.IncomeAdjustments,
		ExemptDividends: dividends, ExemptGains: gains,
		Donations: donations, TurnoverAndWages: turnoverAndWages,
		LossCarryforwardCorporate: opts.LossCarryforwardCorporate,
		LossCarryforwardTrade:     opts.LossCarryforwardTrade,
		TradeAdditions:            opts.TradeAdditions, TradeDeductions: opts.TradeDeductions,
		TradeTaxRatePercent: rate,
		PrepaidCorporate:    paid.corporate, PrepaidSolidarity: paid.solidarity, PrepaidTrade: paid.trade,
		Date: fy.EndDate,
	}
	if !opts.SkipWithholdingCredit {
		input.CreditableWithholding = paid.withholding
		input.CreditableSolidarity = paid.withholdingSolidarity
	}
	result, err := accounting.CalculateTaxProvision(input)
	if err != nil {
		return nil, err
	}
	preview.TaxProvisionResult = *result
	preview.Input = input
	preview.Lines = taxProvisionLines(result)
	preview.Notes = taxProvisionNotes(preview, paid, opts)
	preview.Warning = "Das ist eine Schätzung und keine Steuererklärung. Hinzurechnungen und Kürzungen der " +
		"Gewerbesteuer (§§ 8, 9 GewStG), verdeckte Gewinnausschüttungen und Verlustvorträge stehen nicht in " +
		"den Büchern; sie gehen nur ein, soweit sie angegeben sind. Der Betrag ist änderbar."
	preview.Explanation = fmt.Sprintf(
		"Ergebnis vor Ertragsteuern %s € zuzüglich nicht abziehbarer Aufwendungen %s €, abzüglich steuerfreier "+
			"Beteiligungserträge %s € zuzüglich 5 %% davon (%s €), abzüglich abziehbarer Zuwendungen %s € und "+
			"Verlustvortrag %s € ergibt ein zu versteuerndes Einkommen von %s €. Darauf %s: Körperschaftsteuer "+
			"%s €, Solidaritätszuschlag %s €, Gewerbesteuer %s € (Gewerbeertrag %s €, Messbetrag %s €). "+
			"Abzüglich Vorauszahlungen (%s €, %s €, %s €) und anrechenbarer Kapitalertragsteuer mit "+
			"Solidaritätszuschlag (%s €, %s €) bleiben %s € Körperschaftsteuer- und %s € "+
			"Gewerbesteuerrückstellung; Erstattungsansprüche %s € und %s €.",
		profit, input.NonDeductible, result.ExemptIncome, result.ExemptAddBack, result.DonationsDeducted,
		result.LossDeducted, result.TaxableIncome, result.RatesUsed,
		result.CorporateTax, result.Solidarity, result.TradeTax, result.TradeIncome, result.TradeBase,
		input.PrepaidCorporate, input.PrepaidSolidarity, input.PrepaidTrade,
		accounting.CreditableWithholding(input.CreditableWithholding), input.CreditableSolidarity,
		result.IncomeProvision, result.TradeProvision, result.IncomeRefund, result.TradeRefund)
	return preview, nil
}

// taxProvisionNotes nennt, was die Schätzung annimmt und was zu prüfen ist.
func taxProvisionNotes(preview *TaxProvisionPreview, paid taxPayments, opts TaxProvisionOptions) []string {
	notes := make([]string, 0, 6)
	if opts.ExemptDividends == nil && preview.Input.ExemptDividends > 0 {
		notes = append(notes, fmt.Sprintf(
			"Die Bezüge auf 7005, 7006, 7015 und 7104 (%s €) gelten als steuerfrei nach § 8b Abs. 1 KStG; "+
				"das setzt eine Beteiligung von mindestens 10 %% zu Beginn des Kalenderjahres voraus "+
				"(§ 8b Abs. 4 KStG). Für die Gewerbesteuer bleibt es dabei nur bei mindestens 15 %% zu "+
				"Beginn des Erhebungszeitraums (§ 9 Nr. 2a GewStG); sonst sind sie als Hinzurechnung nach "+
				"§ 8 Nr. 5 GewStG anzugeben.", preview.Input.ExemptDividends))
	}
	if opts.ExemptDividends == nil && preview.PortfolioDividends > 0 {
		notes = append(notes, fmt.Sprintf(
			"Die Bezüge auf 7014 und 7103 (%s €) gelten als steuerpflichtig, weil die Beteiligung zu Beginn "+
				"des Kalenderjahres unter 10 %% gelegen haben kann (§ 8b Abs. 4 KStG). Lag sie darüber, sind "+
				"die steuerfreien Bezüge anzugeben.", preview.PortfolioDividends))
	}
	if !opts.SkipWithholdingCredit && (paid.withholding != 0 || paid.withholdingSolidarity != 0) {
		notes = append(notes, fmt.Sprintf(
			"Angerechnet sind Kapitalertragsteuer %s € (auf volle Euro aufgerundet, § 36 Abs. 3 EStG) und "+
				"Solidaritätszuschlag %s €. Das setzt die Steuerbescheinigung voraus (§ 36 Abs. 2 Nr. 2 Satz 2 EStG).",
			accounting.CreditableWithholding(paid.withholding), paid.withholdingSolidarity))
	}
	if preview.DonationsCarried > 0 {
		notes = append(notes, fmt.Sprintf(
			"Zuwendungen über dem Höchstbetrag (%s €) werden vorgetragen (§ 9 Abs. 1 Satz 1 Nr. 2 KStG).",
			preview.DonationsCarried))
	}
	if preview.IncomeBeforeLoss < 0 {
		notes = append(notes, "Einen Verlustrücktrag in die Vorjahre (§ 10d Abs. 1 EStG) rechnet die "+
			"Schätzung nicht; er wird in der Erklärung beantragt.")
	}
	if preview.LossRemaining > 0 || preview.TradeLossRemaining > 0 {
		notes = append(notes, fmt.Sprintf(
			"Verbleibender Verlustvortrag nach diesem Jahr: Körperschaftsteuer %s €, Gewerbesteuer %s €.",
			preview.LossRemaining, preview.TradeLossRemaining))
	}
	return notes
}

// taxPayments sind die Vorauszahlungen und Abzugsteuern eines Jahres.
type taxPayments struct {
	corporate, solidarity, trade       domain.Cents
	withholding, withholdingSolidarity domain.Cents
}

// taxPrepayments liest die Vorauszahlungen für das Jahr und die einbehaltene
// Kapitalertragsteuer aus den Buchungen.
//
// Nur die Konten der laufenden Steuern zählen: 7600, 7608 und 7610. Nach- und
// Erstattungen für Vorjahre stehen auf 7603, 7604, 7607, 7609, 7641 und 7643
// und gehören nicht in die Abrechnung dieses Jahres. Gelesen wird die Herkunft
// der Buchung und nicht der Kontensaldo: auf 7600 ff. steht am Jahresende auch
// der Aufwand aus der Steuerrückstellung, die dieser Baustein selbst gebucht
// hat, und über den Saldo gerechnet minderte sie sich selbst.
func (s *ClosingBookingService) taxPrepayments(ctx context.Context, year int) (taxPayments, error) {
	var out taxPayments
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return out, fmt.Errorf(
			"die Buchungen des Geschäftsjahres %d konnten nicht gelesen werden: %w", year, err)
	}
	for i := range entries {
		entry := &entries[i]
		if entry.Source == domain.EntrySourceClosing {
			continue
		}
		for _, line := range entry.Lines {
			amount := line.Amount
			if line.Side == domain.SideCredit {
				amount = -amount
			}
			switch line.Account {
			case domain.AccountKoerperschaftsteuer:
				out.corporate += amount
			case domain.AccountSolidaritaetszuschlag:
				out.solidarity += amount
			case domain.AccountGewerbesteuer:
				out.trade += amount
			case accounting.AccountKapitalertragsteuer25:
				out.withholding += amount
			case accounting.AccountSoliKapitalertragsteuer:
				out.withholdingSolidarity += amount
			}
		}
	}
	return out, nil
}

// taxProvisionLines baut den Buchungssatz: die Steuerarten laufen auf eigene
// Aufwands-, Rückstellungs- und Forderungskonten, weil § 4 Abs. 5b EStG die
// Gewerbesteuer vom Abzug ausnimmt und die Überleitung sie getrennt braucht.
// Eine Überzahlung ist am Stichtag ein Erstattungsanspruch (§ 246 Abs. 1 HGB):
// Körperschaftsteuer und Solidaritätszuschlag auf 1450, Gewerbesteuer auf 1435.
func taxProvisionLines(result *accounting.TaxProvisionResult) []domain.JournalLine {
	lines := make([]domain.JournalLine, 0, 6)
	corporate, solidarity := accounting.SplitIncomeTax(result.CorporateDue, result.SolidarityDue)
	switch {
	case result.IncomeProvision > 0:
		if corporate > 0 {
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: domain.AccountKoerperschaftsteuer,
				Amount: corporate, Text: "Körperschaftsteuer",
			})
		}
		if solidarity > 0 {
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: domain.AccountSolidaritaetszuschlag,
				Amount: solidarity, Text: "Solidaritätszuschlag",
			})
		}
		lines = append(lines, domain.JournalLine{
			Side: domain.SideCredit, Account: domain.AccountRueckstellungKoerperschaft,
			Amount: result.IncomeProvision, Text: "Steuerrückstellung",
		})
	case result.IncomeRefund > 0:
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: accounting.AccountKoerperschaftsteuerForderung,
			Amount: result.IncomeRefund, Text: "Erstattungsanspruch Körperschaftsteuer und Solidaritätszuschlag",
		})
		if corporate < 0 {
			lines = append(lines, domain.JournalLine{
				Side: domain.SideCredit, Account: domain.AccountKoerperschaftsteuer,
				Amount: -corporate, Text: "Körperschaftsteuer, Überzahlung",
			})
		}
		if solidarity < 0 {
			lines = append(lines, domain.JournalLine{
				Side: domain.SideCredit, Account: domain.AccountSolidaritaetszuschlag,
				Amount: -solidarity, Text: "Solidaritätszuschlag, Überzahlung",
			})
		}
	}
	switch {
	case result.TradeProvision > 0:
		lines = append(lines,
			domain.JournalLine{
				Side: domain.SideDebit, Account: domain.AccountGewerbesteuer,
				Amount: result.TradeProvision, Text: "Gewerbesteuer",
			},
			domain.JournalLine{
				Side: domain.SideCredit, Account: domain.AccountRueckstellungGewerbesteuer,
				Amount: result.TradeProvision, Text: "Gewerbesteuerrückstellung",
			},
		)
	case result.TradeRefund > 0:
		lines = append(lines,
			domain.JournalLine{
				Side: domain.SideDebit, Account: accounting.AccountGewerbesteuerForderung,
				Amount: result.TradeRefund, Text: "Erstattungsanspruch Gewerbesteuer",
			},
			domain.JournalLine{
				Side: domain.SideCredit, Account: domain.AccountGewerbesteuer,
				Amount: result.TradeRefund, Text: "Gewerbesteuer, Überzahlung",
			},
		)
	}
	return lines
}

// TaxProvisionRequest bucht die Steuerrückstellung. IncomeProvision und
// TradeProvision ersetzen, wenn einer von beiden größer als null ist, die
// gerechneten Rückstellungen; ein Erstattungsanspruch wird dann nicht gebucht.
type TaxProvisionRequest struct {
	FiscalYear      int                 `json:"fiscalYear"`
	Options         TaxProvisionOptions `json:"options"`
	IncomeProvision domain.Cents        `json:"incomeProvision"`
	TradeProvision  domain.Cents        `json:"tradeProvision"`
	Reason          string              `json:"reason"`
}

// BookTaxProvision bildet die Steuerrückstellungen und Erstattungsansprüche und
// bucht sie.
func (s *ClosingBookingService) BookTaxProvision(
	ctx context.Context, req TaxProvisionRequest,
) ([]domain.Provision, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) ([]domain.Provision, error) {
		return s.bookTaxProvision(ctx, req)
	})
}

func (s *ClosingBookingService) bookTaxProvision(
	ctx context.Context, req TaxProvisionRequest,
) ([]domain.Provision, error) {
	if req.IncomeProvision < 0 || req.TradeProvision < 0 {
		return nil, fmt.Errorf("eine geänderte Rückstellung kann nicht negativ sein")
	}
	preview, err := s.PreviewTaxProvision(ctx, req.FiscalYear, req.Options)
	if err != nil {
		return nil, err
	}
	// Zweimal gebucht wäre die Steuerrückstellung doppelt gebildet. Wer den
	// Betrag ändern will, storniert die erste Buchung (§ 239 Abs. 3 HGB); was
	// storniert ist, sperrt nicht mehr.
	if err := refuseSecondClosingEntry(ctx, s.journalRepo, preview.FiscalYear,
		TaxProvisionReference(preview.FiscalYear), "Die Steuerrückstellung"); err != nil {
		return nil, err
	}
	stored, err := s.provisionRepo.FindByYear(ctx, preview.FiscalYear)
	if err != nil {
		return nil, err
	}
	booked, err := liveProvisions(ctx, s.journalRepo, stored)
	if err != nil {
		return nil, err
	}
	for i := range booked {
		if booked[i].Kind.IsTax() {
			return nil, fmt.Errorf(
				"für das Geschäftsjahr %d besteht bereits eine Steuerrückstellung (%q über %s €). "+
					"Zuerst ist ihre Buchung zu stornieren; ein zweiter Lauf bildete sie ein zweites Mal",
				preview.FiscalYear, booked[i].Text, booked[i].SettlementAmount)
		}
	}

	result := preview.TaxProvisionResult
	if req.IncomeProvision > 0 || req.TradeProvision > 0 {
		// Geänderte Beträge: die Aufteilung auf Körperschaftsteuer und
		// Solidaritätszuschlag bleibt im Verhältnis der Rechnung.
		result.IncomeProvision, result.IncomeRefund = req.IncomeProvision, 0
		result.TradeProvision, result.TradeRefund = req.TradeProvision, 0
		result.CorporateDue, result.SolidarityDue = req.IncomeProvision, 0
		if total := result.CorporateTax + result.Solidarity; total > 0 {
			result.CorporateDue = domain.MulRound(req.IncomeProvision, int64(result.CorporateTax), int64(total))
			result.SolidarityDue = req.IncomeProvision - result.CorporateDue
		}
	}
	lines := taxProvisionLines(&result)
	if len(lines) == 0 {
		return nil, fmt.Errorf(
			"für das Geschäftsjahr %d ergibt sich weder eine Steuerrückstellung noch ein "+
				"Erstattungsanspruch: die Vorauszahlungen decken die Steuer genau",
			preview.FiscalYear)
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = preview.Explanation
	}
	entry := &domain.JournalEntry{
		BookingDate: preview.Cutoff, DocumentDate: preview.Cutoff,
		ServiceDateFrom: preview.Cutoff, ServiceDateTo: preview.Cutoff,
		Description:        fmt.Sprintf("Steuerrückstellung %d", preview.FiscalYear),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     TaxProvisionReference(preview.FiscalYear),
		TaxTreatment:       domain.TaxTreatmentNotTaxable,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	receipt, err := selfIssuedVoucher(ctx, s.receipts, preview.FiscalYear, closingVoucher{
		Kind: "steuerrueckstellung", FiscalYear: preview.FiscalYear, Date: preview.Cutoff,
		Description: entry.Description, Explanation: reason,
		Calculation: struct {
			Input   accounting.TaxProvisionInput  `json:"eingaben"`
			Options TaxProvisionOptions           `json:"angaben"`
			Result  accounting.TaxProvisionResult `json:"ergebnis"`
			Konten  []TaxAdjustmentRow            `json:"korrekturkonten"`
		}{preview.Input, req.Options, result, preview.Adjustments},
		Lines: lines,
	})
	if err != nil {
		return nil, err
	}
	reference := entry.DocumentNumber
	attachVoucher(entry, receipt)
	entry.DocumentNumber = reference

	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}

	provisions := make([]domain.Provision, 0, 2)
	for _, part := range []struct {
		kind   domain.ProvisionKind
		amount domain.Cents
		text   string
	}{
		{domain.ProvisionTaxIncome, result.IncomeProvision, "Körperschaftsteuer und Solidaritätszuschlag"},
		{domain.ProvisionTaxTrade, result.TradeProvision, "Gewerbesteuer"},
	} {
		if part.amount <= 0 {
			continue
		}
		balance, expense := accounting.ProvisionAccounts(part.kind)
		provision := &domain.Provision{
			FiscalYear: preview.FiscalYear, Kind: part.kind,
			Text:             fmt.Sprintf("%s %d", part.text, preview.FiscalYear),
			SettlementAmount: part.amount, DiscountedAmount: part.amount,
			// Steuern des abgelaufenen Jahres werden mit der Veranlagung fällig;
			// binnen eines Jahres wird nicht abgezinst (§ 253 Abs. 2 Satz 1 HGB).
			ExpectedDate:   preview.Cutoff,
			BalanceAccount: balance, ExpenseAccount: expense, Reason: reason,
		}
		if err := s.provisionRepo.Save(ctx, provision); err != nil {
			return nil, fmt.Errorf("die Rückstellung ließ sich nicht speichern: %w", err)
		}
		movement := &domain.ProvisionMovement{
			ProvisionID: provision.ID, Kind: domain.ProvisionFormation, Date: preview.Cutoff,
			FiscalYear: preview.FiscalYear, Amount: part.amount, Reason: reason,
			JournalEntryID: &created.ID,
		}
		if err := s.provisionRepo.AddMovement(ctx, movement); err != nil {
			return nil, err
		}
		provision.Movements = append(provision.Movements, *movement)
		provisions = append(provisions, *provision)
	}
	if err := s.auditEntity(ctx, domain.AuditActionCreate, "TAX_PROVISION", preview.FiscalYear, fmt.Sprintf(
		"Steuerrückstellung %d (%s): Körperschaftsteuer und Soli %s €, Gewerbesteuer %s €; "+
			"Erstattungsansprüche %s € und %s €",
		preview.FiscalYear, created.EntryNumber, result.IncomeProvision, result.TradeProvision,
		result.IncomeRefund, result.TradeRefund)); err != nil {
		return nil, err
	}
	return provisions, nil
}

// tradeTaxRate ist der Hebesatz aus der Angabe, sonst aus den
// Abschluss-Einstellungen. Fehlt beides, wird nicht gerechnet: der Hebesatz ist
// eine Angabe der Gemeinde, und ein angenommener Wert ergäbe eine Rückstellung,
// die niemand bemerkt falsch hat.
func (s *ClosingBookingService) tradeTaxRate(ctx context.Context, opts TaxProvisionOptions) (int64, string, error) {
	if opts.TradeTaxRatePercent < 0 {
		return 0, "", fmt.Errorf("der Hebesatz kann nicht negativ sein")
	}
	if opts.TradeTaxRatePercent > 0 {
		return opts.TradeTaxRatePercent, "eingabe", nil
	}
	if s.settingsRepo != nil {
		value, err := s.settingsRepo.Get(ctx, domain.SettingTradeTaxRate)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, "", fmt.Errorf("der Hebesatz ließ sich nicht lesen: %w", err)
		}
		if rate, perr := strconv.ParseInt(strings.TrimSpace(value), 10, 64); perr == nil && rate > 0 {
			return rate, "einstellungen", nil
		}
	}
	return 0, "", fmt.Errorf(
		"der Gewerbesteuer-Hebesatz der Gemeinde fehlt. Er steht im Gewerbesteuerbescheid " +
			"(§ 16 GewStG) und wird in den Abschluss-Einstellungen hinterlegt oder mit der " +
			"Steuerrückstellung angegeben; ohne ihn rechnet Buchfink keine Gewerbesteuer")
}
