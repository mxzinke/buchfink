package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// PayrollService bucht das Lohnjournal eines externen Lohnbüros als
// Monatssumme und stimmt die Lohnkonten ab. Buchfink rechnet keine Löhne ab;
// Lohnsteuer-Anmeldung und Beitragsnachweise kommen vom Lohnbüro. Die Pflichten
// bleiben beim Arbeitgeber: Einbehalt, Anmeldung und Abführung der Lohnsteuer
// (§§ 38 Abs. 3, 41a EStG) und Zahlung des Gesamtsozialversicherungsbeitrags
// (§ 28e Abs. 1 SGB IV).
type PayrollService struct {
	runs        domain.PayrollRepository
	journalRepo domain.JournalRepository
	journalSvc  *JournalService
	posting     *PostingService
	auditRepo   domain.AuditRepository
}

// NewPayrollService verdrahtet den Dienst.
func NewPayrollService(
	runs domain.PayrollRepository, journalRepo domain.JournalRepository,
	journalSvc *JournalService, posting *PostingService, auditRepo domain.AuditRepository,
) *PayrollService {
	return &PayrollService{runs: runs, journalRepo: journalRepo, journalSvc: journalSvc, posting: posting, auditRepo: auditRepo}
}

// PayrollRequest sind die Summen eines Lohnjournals für einen
// Abrechnungsmonat.
type PayrollRequest struct {
	// Month ist der Abrechnungsmonat (JJJJ-MM), für den der Lohn gezahlt wird.
	Month string `json:"month"`
	// ReceiptID ist das abgelegte Lohnjournal; es ist der Beleg der Buchung.
	ReceiptID uint `json:"receiptId"`
	// BookingDate ist leer der letzte Tag des Monats.
	BookingDate string `json:"bookingDate,omitempty"`

	// Brutto, getrennt nach Konten.
	Salaries                    domain.Cents `json:"salaries"`                    // 6020 Gehälter
	ShareholderDirectorSalaries domain.Cents `json:"shareholderDirectorSalaries"` // 6024 GF-Gehälter der Gesellschafter
	DirectorSalaries            domain.Cents `json:"directorSalaries"`            // 6027 GF-Gehälter
	MinijobWages                domain.Cents `json:"minijobWages"`                // 6035 Löhne für Minijobs

	// Abzüge beim Arbeitnehmer.
	EmployeeSocialSecurity domain.Cents `json:"employeeSocialSecurity"`
	WageTax                domain.Cents `json:"wageTax"`
	ChurchTax              domain.Cents `json:"churchTax"`
	SolidaritySurcharge    domain.Cents `json:"solidaritySurcharge"`
	// NetPay ist der Nettoverdienst vor Einbehalten wie Vorschuss oder
	// Pfändung.
	NetPay domain.Cents `json:"netPay"`

	// Kosten des Arbeitgebers.
	EmployerSocialSecurity domain.Cents `json:"employerSocialSecurity"` // 6110
	// Levies sind die Umlagen U1, U2 und die Insolvenzgeldumlage (§ 7 AAG,
	// § 358 SGB III).
	Levies domain.Cents `json:"levies"` // 6110
	// MinijobContributions sind die Pauschalbeiträge an die Minijob-Zentrale.
	MinijobContributions domain.Cents `json:"minijobContributions"` // 6171
	// MinijobFlatTax ist die einheitliche Pauschsteuer, die an die
	// Minijob-Zentrale geht (§ 40a Abs. 2, 6 EStG).
	MinijobFlatTax domain.Cents `json:"minijobFlatTax"` // 6036
	// FlatRateWageTax ist die pauschale Lohnsteuer für Arbeitnehmer, die der
	// Arbeitgeber schuldet (§ 40 Abs. 3 EStG).
	FlatRateWageTax domain.Cents `json:"flatRateWageTax"` // 6039
	// ShareholderDirectorFlatRateTax ist dieselbe für Gesellschafter-
	// Geschäftsführer.
	ShareholderDirectorFlatRateTax domain.Cents `json:"shareholderDirectorFlatRateTax"` // 6037

	// NetToClearing bucht den Nettolohn auf das Lohn- und
	// Gehaltsverrechnungskonto 3790 statt auf 3720.
	NetToClearing bool `json:"netToClearing,omitempty"`
	ClosingPeriod bool `json:"closingPeriod,omitempty"`
}

// Gross ist das Brutto aller Beschäftigten.
func (r PayrollRequest) Gross() domain.Cents {
	return r.Salaries + r.ShareholderDirectorSalaries + r.DirectorSalaries + r.MinijobWages
}

// wageTaxes ist, was an das Betriebsstättenfinanzamt geht.
func (r PayrollRequest) wageTaxes() domain.Cents {
	return r.WageTax + r.ChurchTax + r.SolidaritySurcharge + r.FlatRateWageTax + r.ShareholderDirectorFlatRateTax
}

// registeredWageTax ist die Lohnsteuer, die der Arbeitgeber anmeldet:
// einbehalten und pauschal übernommen, ohne Kirchensteuer und
// Solidaritätszuschlag (§ 41a Abs. 1 Satz 1 Nr. 1, Abs. 2 EStG).
func (r PayrollRequest) registeredWageTax() domain.Cents {
	return r.WageTax + r.FlatRateWageTax + r.ShareholderDirectorFlatRateTax
}

// socialSecurity ist, was an die Einzugsstellen und die Minijob-Zentrale geht.
func (r PayrollRequest) socialSecurity() domain.Cents {
	return r.EmployeeSocialSecurity + r.EmployerSocialSecurity + r.Levies + r.MinijobContributions + r.MinijobFlatTax
}

// validate prüft die Summen: keine negativen Beträge, ein Brutto über null und
// die Netto-Probe Brutto − Sozialversicherung Arbeitnehmer − Lohnsteuer −
// Kirchensteuer − Solidaritätszuschlag = Netto.
func (r PayrollRequest) validate() error {
	if _, err := time.Parse("2006-01", r.Month); err != nil || len(r.Month) != 7 {
		return fmt.Errorf("der Abrechnungsmonat %q ist kein Monat (erwartet JJJJ-MM)", r.Month)
	}
	if r.ReceiptID == 0 {
		return fmt.Errorf("zur Lohnbuchung gehört das abgelegte Lohnjournal als Beleg (§ 146 Abs. 1 AO, GoBD Rz. 61)")
	}
	for _, f := range []struct {
		label string
		value domain.Cents
	}{
		{"Gehälter", r.Salaries}, {"Geschäftsführergehälter der Gesellschafter", r.ShareholderDirectorSalaries},
		{"Geschäftsführergehälter", r.DirectorSalaries}, {"Löhne für Minijobs", r.MinijobWages},
		{"Sozialversicherung Arbeitnehmer", r.EmployeeSocialSecurity}, {"Lohnsteuer", r.WageTax},
		{"Kirchensteuer", r.ChurchTax}, {"Solidaritätszuschlag", r.SolidaritySurcharge}, {"Netto", r.NetPay},
		{"Sozialversicherung Arbeitgeber", r.EmployerSocialSecurity}, {"Umlagen", r.Levies},
		{"Pauschalbeiträge Minijobs", r.MinijobContributions}, {"Pauschsteuer Minijobs", r.MinijobFlatTax},
		{"pauschale Lohnsteuer", r.FlatRateWageTax},
		{"pauschale Lohnsteuer Gesellschafter-Geschäftsführer", r.ShareholderDirectorFlatRateTax},
	} {
		if f.value < 0 {
			return fmt.Errorf("%s ist negativ (%s €); eine Korrektur des Lohnbüros wird storniert und neu gebucht",
				f.label, f.value)
		}
	}
	if r.Gross() <= 0 {
		return fmt.Errorf("das Lohnjournal hat kein Brutto")
	}
	expected := r.Gross() - r.EmployeeSocialSecurity - r.WageTax - r.ChurchTax - r.SolidaritySurcharge
	if expected != r.NetPay {
		return fmt.Errorf(
			"die Netto-Probe geht nicht auf: Brutto %s € − Sozialversicherung %s € − Lohnsteuer %s € − "+
				"Kirchensteuer %s € − Solidaritätszuschlag %s € = %s €, angegeben sind %s € Netto (Differenz %s €). "+
				"Vorschuss, Pfändung, Sachbezug oder vermögenswirksame Leistung stehen nicht in den Summen und "+
				"werden nach dem Lohnjournal umgebucht",
			r.Gross(), r.EmployeeSocialSecurity, r.WageTax, r.ChurchTax, r.SolidaritySurcharge,
			expected, r.NetPay, r.NetPay-expected)
	}
	return nil
}

// lines baut die Summenbuchung des Lohnjournals.
func (r PayrollRequest) lines() []domain.JournalLine {
	label := r.Month[5:7] + "/" + r.Month[0:4]
	netAccount := domain.AccountVerbLohnGehalt
	if r.NetToClearing {
		netAccount = domain.AccountLohnGehaltVerrechnung
	}
	var out []domain.JournalLine
	add := func(side domain.Side, account string, amount domain.Cents, text string) {
		if amount != 0 {
			out = append(out, domain.JournalLine{Side: side, Account: account, Amount: amount, Text: text + " " + label})
		}
	}
	add(domain.SideDebit, domain.AccountGehaelter, r.Salaries, "Gehälter")
	add(domain.SideDebit, domain.AccountGFGehaelterGesellschafter, r.ShareholderDirectorSalaries,
		"Geschäftsführergehälter Gesellschafter")
	add(domain.SideDebit, domain.AccountGFGehaelter, r.DirectorSalaries, "Geschäftsführergehälter")
	add(domain.SideDebit, domain.AccountLoehneMinijobs, r.MinijobWages, "Löhne Minijobs")
	add(domain.SideDebit, domain.AccountGesetzlicheSozialeAufw, r.EmployerSocialSecurity, "Sozialversicherung Arbeitgeberanteil")
	add(domain.SideDebit, domain.AccountGesetzlicheSozialeAufw, r.Levies, "Umlagen")
	add(domain.SideDebit, domain.AccountSozialeAbgabenMinijobs, r.MinijobContributions, "Pauschalbeiträge Minijobs")
	add(domain.SideDebit, domain.AccountPauschsteuerMinijobs, r.MinijobFlatTax, "Pauschsteuer Minijobs")
	add(domain.SideDebit, domain.AccountPauschsteuerArbeitnehmer, r.FlatRateWageTax, "Pauschale Lohnsteuer")
	add(domain.SideDebit, domain.AccountPauschsteuerGesellschafter, r.ShareholderDirectorFlatRateTax,
		"Pauschale Lohnsteuer Gesellschafter-Geschäftsführer")

	add(domain.SideCredit, netAccount, r.NetPay, "Nettolöhne")
	add(domain.SideCredit, domain.AccountVerbLohnKirchensteuer, r.WageTax, "Lohnsteuer")
	add(domain.SideCredit, domain.AccountVerbLohnKirchensteuer, r.ChurchTax, "Kirchensteuer")
	add(domain.SideCredit, domain.AccountVerbLohnKirchensteuer, r.SolidaritySurcharge, "Solidaritätszuschlag")
	add(domain.SideCredit, domain.AccountVerbLohnKirchensteuer, r.FlatRateWageTax+r.ShareholderDirectorFlatRateTax,
		"Pauschale Lohnsteuer")
	add(domain.SideCredit, domain.AccountVerbSozialeSicherheit, r.EmployeeSocialSecurity, "Sozialversicherung Arbeitnehmeranteil")
	add(domain.SideCredit, domain.AccountVerbSozialeSicherheit, r.EmployerSocialSecurity, "Sozialversicherung Arbeitgeberanteil")
	add(domain.SideCredit, domain.AccountVerbSozialeSicherheit, r.Levies, "Umlagen")
	add(domain.SideCredit, domain.AccountVerbSozialeSicherheit, r.MinijobContributions+r.MinijobFlatTax, "Minijob-Zentrale")
	return out
}

// BookPayroll bucht die Summen eines Lohnjournals. Ein Monat wird einmal
// gebucht; ein berichtigtes Journal ersetzt die Buchung über Storno und
// Neubuchung.
func (s *PayrollService) BookPayroll(ctx context.Context, req PayrollRequest) (*domain.JournalEntry, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		active, err := s.activeRuns(ctx, req.Month)
		if err != nil {
			return nil, err
		}
		if len(active) > 0 {
			existing, err := s.journalRepo.FindByID(ctx, active[0].EntryID)
			number := fmt.Sprintf("%d", active[0].EntryID)
			if err == nil {
				number = existing.EntryNumber
			}
			return nil, fmt.Errorf(
				"das Lohnjournal %s ist bereits mit Buchung %s gebucht. Ein berichtigtes Journal ersetzt sie über "+
					"Storno und Neubuchung; bildet das Lohnbüro die Korrektur im Folgemonat ab, wird nur dieser gebucht",
				req.Month, number)
		}
		first, last, _, err := monthBounds(req.Month)
		if err != nil {
			return nil, err
		}
		bookingDate := req.BookingDate
		if bookingDate == "" {
			bookingDate = last
		}
		entry, err := s.posting.PostManualEntry(ctx, ManualEntryRequest{
			ReceiptID: req.ReceiptID,
			Entry: domain.JournalEntry{
				BookingDate:     bookingDate,
				DocumentDate:    bookingDate,
				ServiceDateFrom: first,
				ServiceDateTo:   last,
				Description:     fmt.Sprintf("Löhne und Gehälter %s/%s lt. Lohnjournal", req.Month[5:7], req.Month[0:4]),
				// Arbeitslohn ist kein Leistungsaustausch im Sinne des UStG.
				TaxTreatment:  domain.TaxTreatmentNotTaxable,
				ClosingPeriod: req.ClosingPeriod,
				Lines:         req.lines(),
			},
		})
		if err != nil {
			return nil, err
		}
		if err := s.runs.Save(ctx, &domain.PayrollRun{
			Month: req.Month, EntryID: entry.ID, ReceiptID: req.ReceiptID,
			Gross: req.Gross(), Net: req.NetPay, WageTaxes: req.wageTaxes(), WageTax: req.registeredWageTax(),
			SocialSecurity: req.socialSecurity(),
		}); err != nil {
			return nil, fmt.Errorf("das Lohnjournal ließ sich nicht vermerken: %w", err)
		}
		return entry, nil
	})
}

// activeRuns sind die gebuchten Lohnjournale eines Monats, deren Buchung nicht
// storniert ist.
func (s *PayrollService) activeRuns(ctx context.Context, month string) ([]domain.PayrollRun, error) {
	runs, err := s.runs.FindByMonth(ctx, month)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PayrollRun, 0, len(runs))
	for _, run := range runs {
		reversal, err := s.journalRepo.FindReversalOf(ctx, run.EntryID)
		if err != nil {
			return nil, err
		}
		if reversal == nil {
			out = append(out, run)
		}
	}
	return out, nil
}

// PayrollAccountStatus ist der Stand eines Lohnkontos zum Stichtag.
type PayrollAccountStatus struct {
	Account string `json:"account"`
	Name    string `json:"name"`
	// Balance ist der Saldo als Verbindlichkeit: positiv im Haben, negativ im
	// Soll.
	Balance domain.Cents `json:"balance"`
	// Expected ist, was zum Stichtag noch offen stehen darf.
	Expected domain.Cents `json:"expected"`
	Note     string       `json:"note"`
}

// PayrollReconciliation ist die Abstimmung der Lohnkonten zum Stichtag.
type PayrollReconciliation struct {
	Cutoff   string                 `json:"cutoff"`
	Month    string                 `json:"month"`
	Booked   bool                   `json:"booked"`
	Accounts []PayrollAccountStatus `json:"accounts"`
	Findings []domain.CheckFinding  `json:"findings"`
}

// Reconcile stimmt die Lohnkonten zum Stichtag ab. Offen stehen dürfen der
// Nettolohn und die Lohnsteuer des Stichtagsmonats, bis zum Zehnten auch die
// des Vormonats: die Lohnsteuer ist bis zum zehnten Tag nach Ablauf des
// Anmeldungszeitraums anzumelden und abzuführen (§ 41a Abs. 1 EStG). Die Sozialversicherungsbeiträge sind am drittletzten
// Bankarbeitstag des Beschäftigungsmonats fällig (§ 23 Abs. 1 Satz 2 SGB IV)
// und stehen zum Monatsende auf null.
func (s *PayrollService) Reconcile(ctx context.Context, cutoff string) (*PayrollReconciliation, error) {
	if err := domain.CheckDate("Stichtag", cutoff); err != nil {
		return nil, err
	}
	month := cutoff[:7]
	result := &PayrollReconciliation{Cutoff: cutoff, Month: month,
		Accounts: make([]PayrollAccountStatus, 0, 6), Findings: make([]domain.CheckFinding, 0)}

	// Bis zum Zehnten darf auch der Vormonat offen stehen: Lohnsteuer bis zur
	// Fälligkeit, Nettolöhne bis zur Auszahlung zum Monatswechsel.
	months := []string{month}
	if cutoff[8:] <= "10" {
		months = append(months, addMonthsISO(cutoff, -1)[:7])
	}
	var net, taxes domain.Cents
	for i, m := range months {
		runs, err := s.activeRuns(ctx, m)
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			net += run.Net
			taxes += run.WageTaxes
		}
		if i == 0 {
			result.Booked = len(runs) > 0
		}
	}

	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, s.journalSvc.FiscalYearOf(ctx, cutoff), cutoff)
	if err != nil {
		return nil, fmt.Errorf("die Lohnkonten ließen sich nicht lesen: %w", err)
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}
	balance := func(account string) domain.Cents {
		t := turnovers[account]
		return t.Credit - t.Debit
	}
	finding := func(account, message, reference string) {
		result.Findings = append(result.Findings, domain.CheckFinding{
			Rule: domain.CheckRulePayrollAccounts, Severity: domain.CheckWarning,
			ObjectType: "ACCOUNT", ObjectID: account, ObjectName: chart.Name(account),
			Message: message, Reference: reference,
		})
	}

	netBalance := balance(domain.AccountVerbLohnGehalt) + balance(domain.AccountLohnGehaltVerrechnung)
	for _, st := range []PayrollAccountStatus{
		{Account: domain.AccountVerbLohnGehalt, Expected: net,
			Note: "Nettolöhne; nach der Auszahlung null, offen höchstens der Stichtagsmonat"},
		{Account: domain.AccountLohnGehaltVerrechnung, Note: "Lohn- und Gehaltsverrechnung"},
		{Account: domain.AccountVerbEinbehaltungen, Note: "Pfändungen und Sparbeträge bis zur Weiterleitung"},
		{Account: domain.AccountVerbLohnKirchensteuer, Expected: taxes,
			Note: "Lohnsteuer des Anmeldungszeitraums bis zur Zahlung am 10. des Folgemonats"},
		{Account: domain.AccountVerbSozialeSicherheit, Note: "Beiträge; zum Monatsende null"},
		{Account: domain.AccountBeitragsschuldSV, Note: "voraussichtliche Beitragsschuld"},
	} {
		st.Name = chart.Name(st.Account)
		st.Balance = balance(st.Account)
		result.Accounts = append(result.Accounts, st)
	}

	if netBalance > net {
		finding(domain.AccountVerbLohnGehalt, fmt.Sprintf(
			"Auf den Lohnkonten stehen %s € Nettolohn offen, aus dem Lohnjournal %s stammen %s €. Nettolöhne "+
				"früherer Monate sind nicht ausgezahlt oder die Auszahlung ist nicht zugeordnet; bei Vorschuss, "+
				"Pfändung oder Sachbezug fehlt die Umbuchung nach dem Lohnjournal", netBalance, month, net),
			"§ 41 EStG, GoBD Rz. 36")
	}
	if netBalance < 0 {
		finding(domain.AccountVerbLohnGehalt, fmt.Sprintf(
			"Auf den Lohnkonten steht ein Sollsaldo von %s €: mehr ausgezahlt als gebucht. Eine Auszahlung ist "+
				"doppelt zugeordnet, ein Vorschuss steht auf dem Lohnkonto statt auf %s, oder das Lohnjournal fehlt",
			-netBalance, domain.AccountForderungenPersonal), "GoBD Rz. 36")
	}
	if tax := balance(domain.AccountVerbLohnKirchensteuer); tax > taxes {
		finding(domain.AccountVerbLohnKirchensteuer, fmt.Sprintf(
			"Auf %s stehen %s € offen, aus dem Lohnjournal %s stammen %s €. Lohnsteuer früherer Zeiträume ist "+
				"nicht abgeführt; bei vierteljährlicher oder jährlicher Anmeldung (§ 41a Abs. 2 EStG) kann mehr offen "+
				"sein. Für nicht abgeführte Lohnsteuer haftet der Arbeitgeber",
			domain.AccountVerbLohnKirchensteuer, tax, month, taxes), "§ 41a Abs. 1 EStG, § 42d Abs. 1 EStG")
	} else if tax < 0 {
		finding(domain.AccountVerbLohnKirchensteuer, fmt.Sprintf(
			"Auf %s steht ein Sollsaldo von %s €: mehr Lohnsteuer gezahlt als gebucht",
			domain.AccountVerbLohnKirchensteuer, -tax), "GoBD Rz. 36")
	}
	if sv := balance(domain.AccountVerbSozialeSicherheit); sv > 0 {
		finding(domain.AccountVerbSozialeSicherheit, fmt.Sprintf(
			"Auf %s stehen %s € Sozialversicherungsbeiträge offen. Sie sind am drittletzten Bankarbeitstag des "+
				"Beschäftigungsmonats fällig; ein Rest ist im Folgemonat zu zahlen. Arbeitnehmeranteile "+
				"vorzuenthalten ist strafbar", domain.AccountVerbSozialeSicherheit, sv),
			"§ 23 Abs. 1 Satz 2 SGB IV, § 266a Abs. 1 StGB")
	} else if sv < 0 && !result.Booked {
		finding(domain.AccountVerbSozialeSicherheit, fmt.Sprintf(
			"Auf %s steht ein Sollsaldo von %s €: Beiträge sind gezahlt, das Lohnjournal %s ist nicht gebucht",
			domain.AccountVerbSozialeSicherheit, -sv, month), "§ 23 Abs. 1 Satz 2 SGB IV")
	}
	return result, nil
}

// Findings liefert die Befunde der Lohnkonten für den Prüflauf. Ohne gebuchtes
// Lohnjournal und ohne Bewegung auf den Lohnkonten gibt es keine.
func (s *PayrollService) Findings(ctx context.Context, cutoff string) ([]domain.CheckFinding, error) {
	all, err := s.runs.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, nil
	}
	r, err := s.Reconcile(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	return r.Findings, nil
}
