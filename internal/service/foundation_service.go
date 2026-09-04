package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// FoundationService begleitet die Gründung einer Kapitalgesellschaft von der
// Beurkundung bis zur Eintragung.
//
// Der Kern ist eine Rechnung, keine Erfassung: Die Zahlen der Unterbilanz stehen
// bereits im Journal, seit der ersten Notarrechnung. Was fehlte, war jemand, der
// sie zusammenzählt und der Gesellschafterin sagt, dass sie am Tag der
// Eintragung 3.000 € nachschießen wird.
type FoundationService struct {
	foundationRepo domain.FoundationRepository
	accountRepo    domain.AccountRepository
	journalRepo    domain.JournalRepository
	settingsRepo   domain.SettingsRepository
	journalSvc     *JournalService
	auditRepo      domain.AuditRepository
	fiscalYear     int
}

// NewFoundationService creates the Gründungsbegleitung.
func NewFoundationService(
	foundationRepo domain.FoundationRepository,
	accountRepo domain.AccountRepository,
	journalRepo domain.JournalRepository,
	settingsRepo domain.SettingsRepository,
	journalSvc *JournalService,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *FoundationService {
	return &FoundationService{
		foundationRepo: foundationRepo,
		accountRepo:    accountRepo,
		journalRepo:    journalRepo,
		settingsRepo:   settingsRepo,
		journalSvc:     journalSvc,
		auditRepo:      auditRepo,
		fiscalYear:     fiscalYear,
	}
}

// SetFiscalYear updates the active fiscal year.
func (s *FoundationService) SetFiscalYear(year int) { s.fiscalYear = year }

// FoundationState is everything the Gründungsansicht needs, in one call.
//
// Zusammen und nicht einzeln, weil die Teile voneinander abhängen: die Regeln
// folgen aus der Rechtsform, die Fristen aus dem Beurkundungsdatum, die
// Unterbilanz aus beidem und dem Journal. Vier Aufrufe könnten einen Stand
// zeigen, den es so nie gab.
type FoundationState struct {
	// Applies ist falsch, wenn die Rechtsform des Mandanten keine
	// Kapitalgesellschaft ist. Dann gibt es keinen Gründungsweg, und die
	// übrigen Felder sind leer.
	Applies bool `json:"applies"`
	// HasFoundation ist falsch, solange keine Gründung erfasst wurde.
	HasFoundation bool `json:"hasFoundation"`

	LegalForm  string                     `json:"legalForm"`
	Rules      accounting.FoundationRules `json:"rules"`
	Foundation *domain.Foundation         `json:"foundation,omitempty"`
	Stage      domain.FoundationStage     `json:"stage"`

	Anmeldung   *domain.AnmeldungCheck  `json:"anmeldung,omitempty"`
	Unterbilanz *domain.Unterbilanz     `json:"unterbilanz,omitempty"`
	Duties      []domain.FoundationDuty `json:"duties"`

	// PostingsBooked sagt, ob die Gründungsbuchungen schon im Journal stehen.
	PostingsBooked bool `json:"postingsBooked"`
}

// GetState assembles the Gründungsansicht.
func (s *FoundationService) GetState(ctx context.Context) (*FoundationState, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}

	rules, applies := accounting.FoundationRulesFor(settings.LegalForm)
	state := &FoundationState{
		Applies:   applies,
		LegalForm: settings.LegalForm,
		Rules:     rules,
		Duties:    []domain.FoundationDuty{},
	}
	if !applies {
		return state, nil
	}

	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return state, nil
	}

	state.HasFoundation = true
	state.Foundation = f
	state.Stage = f.Stage()

	check := s.anmeldungCheck(f, rules)
	state.Anmeldung = &check

	unterbilanz, err := s.Unterbilanz(ctx, f)
	if err != nil {
		return nil, err
	}
	state.Unterbilanz = unterbilanz

	tasks, err := s.foundationRepo.Tasks(ctx, f.ID)
	if err != nil {
		return nil, err
	}
	done := make(map[string]string, len(tasks))
	for _, t := range tasks {
		done[t.Key] = t.DoneOn
	}
	state.Duties = accounting.FoundationDuties(f, rules, done)

	booked, err := s.postingsBooked(ctx)
	if err != nil {
		return nil, err
	}
	state.PostingsBooked = booked

	return state, nil
}

// Save writes the Gründung after checking what only this service can check.
func (s *FoundationService) Save(ctx context.Context, f *domain.Foundation) (*domain.Foundation, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}
	rules, applies := accounting.FoundationRulesFor(settings.LegalForm)
	if !applies {
		return nil, fmt.Errorf(
			"der Gründungsweg gilt für Kapitalgesellschaften. Die Rechtsform des Mandanten ist %q — "+
				"eine Vorgesellschaft und damit eine Unterbilanzhaftung gibt es dort nicht",
			settings.LegalForm)
	}
	if err := validateFoundation(f, rules); err != nil {
		return nil, err
	}

	if existing, err := s.foundationRepo.Get(ctx); err == nil && existing != nil {
		f.ID = existing.ID
	}
	if err := s.foundationRepo.Save(ctx, f); err != nil {
		return nil, err
	}
	s.audit(ctx, domain.AuditActionUpdate, f.ID, fmt.Sprintf(
		"Gründung erfasst: Beurkundung %s, Stammkapital %s €, %d Gesellschafter",
		f.NotarizedOn, f.ShareCapital, len(f.Shareholders)))
	return f, nil
}

// validateFoundation checks what the Gesellschaftsvertrag itself has to satisfy.
func validateFoundation(f *domain.Foundation, rules accounting.FoundationRules) error {
	if len(f.NotarizedOn) != 10 {
		return fmt.Errorf("das Datum der Beurkundung fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if f.RegisteredOn != "" {
		if len(f.RegisteredOn) != 10 {
			return fmt.Errorf("das Datum der Eintragung ist unvollständig (erwartet JJJJ-MM-TT)")
		}
		if f.RegisteredOn < f.NotarizedOn {
			return fmt.Errorf(
				"die Eintragung (%s) kann nicht vor der Beurkundung (%s) liegen",
				f.RegisteredOn, f.NotarizedOn)
		}
	}
	if f.ShareCapital < rules.MinShareCapital {
		return fmt.Errorf(
			"das Stammkapital beträgt %s €. Für die Rechtsform %s sind mindestens %s € vorgeschrieben",
			f.ShareCapital, rules.LegalForm, rules.MinShareCapital)
	}
	if f.FoundationCostCap < 0 {
		return fmt.Errorf("der Gründungsaufwand laut Satzung kann nicht negativ sein")
	}
	if len(f.Shareholders) == 0 {
		return fmt.Errorf("ohne Gesellschafter gibt es kein gezeichnetes Kapital")
	}

	for i := range f.Shareholders {
		sh := &f.Shareholders[i]
		if strings.TrimSpace(sh.Name) == "" {
			return fmt.Errorf("der %d. Gesellschafter hat keinen Namen", i+1)
		}
		if sh.Kind == "" {
			sh.Kind = domain.ContributionCash
		}
		if !sh.Kind.Valid() {
			return fmt.Errorf("%s: die Art der Einlage ist unbekannt", sh.Name)
		}
		if rules.CashOnly && sh.Kind == domain.ContributionInKind {
			return fmt.Errorf(
				"%s: bei der %s sind Sacheinlagen ausgeschlossen (§ 5a Abs. 2 Satz 2 GmbHG)",
				sh.Name, rules.LegalForm)
		}
		if sh.ShareCapital <= 0 {
			return fmt.Errorf("%s: der übernommene Geschäftsanteil muss größer als null sein", sh.Name)
		}
		if sh.PaidIn < 0 {
			return fmt.Errorf("%s: die geleistete Einlage kann nicht negativ sein", sh.Name)
		}
		if sh.PaidIn > sh.ShareCapital {
			return fmt.Errorf(
				"%s: geleistet sind %s €, übernommen aber nur %s €. Mehr als den Geschäftsanteil "+
					"kann niemand einzahlen — ein Aufgeld gehört in die Kapitalrücklage (Konto 2920)",
				sh.Name, sh.PaidIn, sh.ShareCapital)
		}
	}

	if sum := f.SubscribedCapital(); sum != f.ShareCapital {
		return fmt.Errorf(
			"die übernommenen Geschäftsanteile ergeben %s €, das Stammkapital beträgt %s €. "+
				"Beides muss übereinstimmen (§ 5 Abs. 3 Satz 2 GmbHG)",
			sum, f.ShareCapital)
	}
	return nil
}

// -------------------------------------------------------------
// Anmeldung zum Handelsregister
// -------------------------------------------------------------

// anmeldungCheck answers whether enough has been contributed to file.
func (s *FoundationService) anmeldungCheck(
	f *domain.Foundation, rules accounting.FoundationRules,
) domain.AnmeldungCheck {
	check := domain.AnmeldungCheck{
		LegalForm:       rules.LegalForm,
		MinShareCapital: rules.MinShareCapital,
		ShareCapital:    f.ShareCapital,
		RequiredPaidIn:  rules.RequiredPaidIn(f),
		ActualPaidIn:    f.PaidInCapital(),
		Reference:       rules.Reference,
		Findings:        []string{},
	}

	for _, sh := range f.Shareholders {
		required := rules.RequiredPerShare(sh)
		row := domain.ShareholderCheck{
			ShareholderID:  sh.ID,
			Name:           sh.Name,
			Kind:           sh.Kind,
			ShareCapital:   sh.ShareCapital,
			RequiredPaidIn: required,
			PaidIn:         sh.PaidIn,
			IsSatisfied:    sh.PaidIn >= required,
		}
		check.Shareholders = append(check.Shareholders, row)
		if !row.IsSatisfied {
			check.Findings = append(check.Findings, fmt.Sprintf(
				"%s: geleistet sind %s €, vor der Anmeldung nötig sind %s € (%s).",
				sh.Name, sh.PaidIn, required, rules.Reference))
		}
	}

	if check.ActualPaidIn < check.RequiredPaidIn {
		check.Findings = append(check.Findings, fmt.Sprintf(
			"Zusammen sind %s € geleistet, die Anmeldung verlangt %s € (%s).",
			check.ActualPaidIn, check.RequiredPaidIn, rules.Reference))
	}

	check.IsSatisfied = len(check.Findings) == 0
	return check
}

// -------------------------------------------------------------
// Unterbilanz (Vorbelastungshaftung)
// -------------------------------------------------------------

// Unterbilanz computes the Vorbelastungsrechnung on the relevant Stichtag.
//
// Stichtag ist der Tag der Eintragung; fehlt er noch, rechnet Buchfink auf heute
// und sagt, dass die Zahl vorläufig ist. Eingefroren wird sie nicht: die
// Buchungen bis zum Eintragungstag hängen in der Hash-Chain, und was sich nicht
// mehr ändern kann, muss auch nicht zweimal gespeichert werden. Eine
// rückdatierte Buchung könnte das Ergebnis noch verschieben — davor schützt die
// Festschreibung des Zeitraums, nicht dieser Dienst.
func (s *FoundationService) Unterbilanz(ctx context.Context, f *domain.Foundation) (*domain.Unterbilanz, error) {
	asOf := f.RegisteredOn
	final := true
	if asOf == "" {
		asOf = time.Now().Format("2006-01-02")
		final = false
	}

	assets, liabilities, err := s.netAssetsUntil(ctx, asOf)
	if err != nil {
		return nil, err
	}

	u := &domain.Unterbilanz{
		AsOf:         asOf,
		IsFinal:      final,
		ShareCapital: f.ShareCapital,
		Assets:       assets,
		Liabilities:  liabilities,
		NetAssets:    assets - liabilities,
		Shares:       []domain.UnterbilanzShare{},
	}

	u.Shortfall = u.ShareCapital - u.NetAssets
	if u.Shortfall < 0 {
		u.Shortfall = 0
	}

	// Der satzungsmäßige Gründungsaufwand deckt die Unterdeckung, soweit er
	// reicht: was die Satzung der Gesellschaft auferlegt, ist zulässig von ihr
	// getragen und wird den Gesellschaftern nicht noch einmal berechnet.
	//
	// Buchfink unterscheidet dabei nicht, ob die Unterdeckung aus dem
	// Gründungsaufwand oder aus einem Anlaufverlust stammt — das ist eine
	// Zuordnung, die nur der Gründer treffen kann. Beide Zahlen stehen deshalb
	// nebeneinander im Ergebnis, statt zu einer verschmolzen zu werden.
	u.Covered = f.FoundationCostCap
	if u.Covered > u.Shortfall {
		u.Covered = u.Shortfall
	}
	u.Amount = u.Shortfall - u.Covered

	u.Shares = distributeUnterbilanz(u.Amount, f.Shareholders)
	return u, nil
}

// netAssetsUntil sums assets and debts from the journal up to a cutoff date.
//
// Der fehlende Saldenvortrag stört hier nicht: Eine Gründung liegt im ersten
// Geschäftsjahr, und dort sind Bewegung und Bestand dasselbe. Auf ein Folgejahr
// angewandt wäre die Rechnung falsch — die Vorbelastungshaftung endet aber mit
// der Eintragung, und die liegt in aller Regel im Gründungsjahr.
func (s *FoundationService) netAssetsUntil(ctx context.Context, until string) (domain.Cents, domain.Cents, error) {
	entries, err := s.journalRepo.FindByBookingDateRange(ctx, s.fiscalYear, "", until)
	if err != nil {
		return 0, 0, err
	}

	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return 0, 0, err
	}
	chart := accounting.NewChart(accounts)

	var assets, liabilities domain.Cents
	for _, e := range entries {
		for _, l := range e.Lines {
			acc, ok := chart.Lookup(collectAccount(l.Account))
			if !ok {
				continue
			}

			// Vorzeichen aus der Seite: Soll erhöht ein Aktivkonto, Haben ein
			// Passivkonto. Eine Generalumkehr trägt negative Beträge auf
			// derselben Seite und rechnet sich damit von selbst heraus.
			signed := l.Amount
			if l.Side == domain.SideCredit {
				signed = -signed
			}

			switch {
			case acc.Type == domain.AccountTypeAsset:
				assets += signed
			case l.Account == domain.AccountAusstehendeEinlagenOffen:
				// Die nicht eingeforderte Einlage steht als offene Absetzung vom
				// gezeichneten Kapital auf der Passivseite (§ 272 Abs. 1 Satz 3
				// HGB), ist der Sache nach aber eine Forderung gegen den
				// Gesellschafter. Sie gehört ins Reinvermögen, sonst zeigte die
				// Rechnung eine Unterbilanz in Höhe der noch nicht eingeforderten
				// Einlage — und die ist keine.
				assets += signed
			case acc.Type == domain.AccountTypeEquity:
				// Eigenkapital ist keine Schuld. Es steht auf der anderen Seite
				// der Rechnung: gegen das Stammkapital wird verglichen, nicht
				// mit ihm summiert.
			case acc.Type == domain.AccountTypeLiability:
				liabilities -= signed
			}
		}
	}
	return assets, liabilities, nil
}

// collectAccount folds a Personenkonto into its Sammelkonto so that an open item
// counts as Forderung or Verbindlichkeit — the Personenkonten themselves are not
// part of the chart of accounts.
func collectAccount(number string) string {
	if kind, ok := domain.LedgerAccountKind(number); ok {
		if kind == domain.ContactTypeCustomer {
			return domain.AccountForderungenLuL
		}
		return domain.AccountVerbindlichkeitenLuL
	}
	return number
}

// distributeUnterbilanz splits the liability by share of the Stammkapital.
//
// Der Rundungsrest geht an den größten Anteil. Irgendwohin muss er, sonst ergibt
// die Spalte nicht die Summe — und beim größten fällt ein Cent am wenigsten ins
// Gewicht.
func distributeUnterbilanz(amount domain.Cents, shareholders []domain.Shareholder) []domain.UnterbilanzShare {
	shares := make([]domain.UnterbilanzShare, 0, len(shareholders))
	var total domain.Cents
	for _, sh := range shareholders {
		total += sh.ShareCapital
	}

	largest, assigned := -1, domain.Cents(0)
	for i, sh := range shareholders {
		row := domain.UnterbilanzShare{
			ShareholderID: sh.ID,
			Name:          sh.Name,
			ShareCapital:  sh.ShareCapital,
		}
		if total > 0 && amount > 0 {
			row.Amount = domain.Cents(int64(amount) * int64(sh.ShareCapital) / int64(total))
			assigned += row.Amount
		}
		shares = append(shares, row)
		if largest < 0 || sh.ShareCapital > shareholders[largest].ShareCapital {
			largest = i
		}
	}

	if largest >= 0 && amount > 0 {
		shares[largest].Amount += amount - assigned
	}
	return shares
}

// -------------------------------------------------------------
// Gründungsbuchungen
// -------------------------------------------------------------

// FoundationPosting is one proposed booking, shown before it is written.
type FoundationPosting struct {
	Title       string               `json:"title"`
	Date        string               `json:"date"`
	Description string               `json:"description"`
	Reference   string               `json:"reference"`
	Lines       []domain.JournalLine `json:"lines"`
	Amount      domain.Cents         `json:"amount"`
}

// FoundationPostingPreview is the Gründungsbuchungslauf before the release.
//
// Vorschlag mit Vorschau, gebucht auf Freigabe — wie beim Abschreibungslauf.
// Buchungen, die eine Anwendung von sich aus schreibt, sind in einer
// GoBD-Buchhaltung die schlechtere Hälfte der Bequemlichkeit.
type FoundationPostingPreview struct {
	Postings      []FoundationPosting `json:"postings"`
	Total         domain.Cents        `json:"total"`
	AlreadyBooked bool                `json:"alreadyBooked"`
	// Skipped nennt, was nicht gebucht wird, und warum.
	Skipped []string `json:"skipped"`
}

// PreviewPostings builds the founding bookings without writing them.
//
// Zwei Sätze, mehr ist es nicht:
//
//	Zeichnung   SOLL 1298 Ausstehende Einlagen, eingefordert
//	            HABEN 2900 Gezeichnetes Kapital
//	Einzahlung  SOLL 1800 Bank
//	            HABEN 1298 Ausstehende Einlagen, eingefordert
//
// Gebucht wird das volle Stammkapital als eingefordert. Ein nicht eingeforderter
// Teil (Konto 2910) ist eine Entscheidung der Gesellschafter und keine, die aus
// den erfassten Daten folgt; wer ihn braucht, bucht ihn von Hand — die
// Unterbilanzrechnung liest das Konto trotzdem.
func (s *FoundationService) PreviewPostings(ctx context.Context) (*FoundationPostingPreview, error) {
	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}

	booked, err := s.postingsBooked(ctx)
	if err != nil {
		return nil, err
	}

	preview := &FoundationPostingPreview{
		AlreadyBooked: booked,
		Postings:      []FoundationPosting{},
		Skipped:       []string{},
	}
	if booked {
		return preview, nil
	}

	preview.Postings = append(preview.Postings, FoundationPosting{
		Title:       "Zeichnung des Stammkapitals",
		Date:        f.NotarizedOn,
		Description: "Übernahme der Geschäftsanteile laut Gesellschaftsvertrag",
		Reference:   "§ 272 Abs. 1 HGB",
		Amount:      f.ShareCapital,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: domain.AccountAusstehendeEinlagenGeford, Amount: f.ShareCapital},
			{Side: domain.SideCredit, Account: domain.AccountGezeichnetesKapital, Amount: f.ShareCapital},
		},
	})

	for _, sh := range f.Shareholders {
		if sh.Kind == domain.ContributionInKind {
			preview.Skipped = append(preview.Skipped, fmt.Sprintf(
				"%s leistet eine Sacheinlage über %s €. Sie gehört auf das Konto des "+
					"eingebrachten Gegenstands und ist deshalb von Hand zu buchen.",
				sh.Name, sh.ShareCapital))
			continue
		}
		if sh.PaidIn <= 0 {
			continue
		}
		preview.Postings = append(preview.Postings, FoundationPosting{
			Title:       "Einzahlung " + sh.Name,
			Date:        f.NotarizedOn,
			Description: fmt.Sprintf("Bareinlage %s auf das Geschäftskonto", sh.Name),
			Reference:   "§ 7 Abs. 2 GmbHG",
			Amount:      sh.PaidIn,
			Lines: []domain.JournalLine{
				{Side: domain.SideDebit, Account: domain.AccountBank, Amount: sh.PaidIn},
				{Side: domain.SideCredit, Account: domain.AccountAusstehendeEinlagenGeford, Amount: sh.PaidIn},
			},
		})
	}

	for _, p := range preview.Postings {
		preview.Total += p.Amount
	}
	return preview, nil
}

// BookPostings writes the founding bookings after the user released them.
func (s *FoundationService) BookPostings(ctx context.Context) ([]domain.JournalEntry, error) {
	preview, err := s.PreviewPostings(ctx)
	if err != nil {
		return nil, err
	}
	if preview.AlreadyBooked {
		return nil, fmt.Errorf(
			"die Gründungsbuchungen stehen bereits im Journal. Eine Korrektur läuft über den Storno")
	}
	if len(preview.Postings) == 0 {
		return nil, fmt.Errorf("es gibt nichts zu buchen")
	}

	created := make([]domain.JournalEntry, 0, len(preview.Postings))
	for _, p := range preview.Postings {
		entry := &domain.JournalEntry{
			BookingDate:        p.Date,
			DocumentDate:       p.Date,
			ServiceDateFrom:    p.Date,
			ServiceDateTo:      p.Date,
			Description:        p.Description,
			Source:             domain.EntrySourceOpening,
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              p.Lines,
		}
		out, err := s.journalSvc.Post(ctx, entry)
		if err != nil {
			return created, fmt.Errorf("%s konnte nicht gebucht werden: %w", p.Title, err)
		}
		created = append(created, *out)
	}
	return created, nil
}

// postingsBooked reports whether Gezeichnetes Kapital has already been booked.
func (s *FoundationService) postingsBooked(ctx context.Context) (bool, error) {
	entries, err := s.journalRepo.FindByAccount(ctx, domain.AccountGezeichnetesKapital, s.fiscalYear)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

// -------------------------------------------------------------
// Eintragung und Pflichten
// -------------------------------------------------------------

// Register records the entry in the Handelsregister — the end of the
// Vorgesellschaft.
//
// Ab hier besteht die Gesellschaft als juristische Person (§ 11 Abs. 1 GmbHG),
// die Handelndenhaftung des § 11 Abs. 2 GmbHG endet, und die Unterbilanz steht
// auf diesen Tag fest.
func (s *FoundationService) Register(ctx context.Context, date, court, number string) (*domain.Foundation, error) {
	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}
	if f.IsRegistered() {
		return nil, fmt.Errorf("die Gesellschaft ist seit dem %s eingetragen", f.RegisteredOn)
	}
	if len(date) != 10 {
		return nil, fmt.Errorf("das Datum der Eintragung fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if date < f.NotarizedOn {
		return nil, fmt.Errorf(
			"die Eintragung (%s) kann nicht vor der Beurkundung (%s) liegen", date, f.NotarizedOn)
	}
	if strings.TrimSpace(number) == "" {
		return nil, fmt.Errorf("die Registernummer fehlt")
	}

	f.RegisteredOn = date
	f.RegisterCourt = strings.TrimSpace(court)
	f.RegisterNumber = strings.TrimSpace(number)
	if err := s.foundationRepo.Save(ctx, f); err != nil {
		return nil, err
	}

	// Die Anmeldung ist mit der Eintragung erledigt; sie noch als offene Frist
	// zu führen wäre eine Nachfrage nach etwas, das nachweislich geschehen ist.
	_ = s.foundationRepo.CompleteTask(ctx, &domain.FoundationTask{
		FoundationID: f.ID,
		Key:          accounting.DutyHandelsregister,
		DoneOn:       date,
		Note:         strings.TrimSpace(court + " " + number),
	})

	s.audit(ctx, domain.AuditActionUpdate, f.ID, fmt.Sprintf(
		"Eintragung ins Handelsregister am %s: %s %s", date, f.RegisterCourt, f.RegisterNumber))
	return f, nil
}

// CompleteDuty records a fulfilled Gründungspflicht, or takes it back when the
// date is empty.
func (s *FoundationService) CompleteDuty(ctx context.Context, key, doneOn, note string) error {
	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return err
	}
	if f == nil {
		return fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}
	if doneOn == "" {
		return s.foundationRepo.ClearTask(ctx, f.ID, key)
	}
	if len(doneOn) != 10 {
		return fmt.Errorf("das Datum der Erledigung ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	return s.foundationRepo.CompleteTask(ctx, &domain.FoundationTask{
		FoundationID: f.ID,
		Key:          key,
		DoneOn:       doneOn,
		Note:         note,
	})
}

func (s *FoundationService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "GRUENDUNG", fmt.Sprintf("%d", id), details)
}
