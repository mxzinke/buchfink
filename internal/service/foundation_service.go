package service

import (
	"context"
	"fmt"
	"strings"

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
	serviceClock
	auditor[uint]
	foundationRepo domain.FoundationRepository
	accountRepo    domain.AccountRepository
	journalRepo    domain.JournalRepository
	settingsRepo   domain.SettingsRepository
	journalSvc     *JournalService
	// yearAligner zieht den Beginn des Gründungsjahres auf die Beurkundung nach.
	// Optional: ohne ihn bleibt die Gründung erfasst und der Zeitraum, wie er war.
	yearAligner FoundingYearAligner
	// statements und renderer tragen die Eröffnungsbilanz: die Gliederung auf
	// den Beurkundungstag und ihren Satz. Beide dürfen fehlen; dann fehlt der
	// Eröffnungsbilanz eine Stufe und sie sagt, welche.
	statements StatementAtSource
	renderer   DocumentRenderer
	// receipts bindet den Beschluss an die Einforderung.
	receipts closingReceiptFiler
}

// NewFoundationService creates the service for the Gründung.
func NewFoundationService(
	foundationRepo domain.FoundationRepository,
	accountRepo domain.AccountRepository,
	journalRepo domain.JournalRepository,
	settingsRepo domain.SettingsRepository,
	journalSvc *JournalService,
	auditRepo domain.AuditRepository,
) *FoundationService {
	return &FoundationService{
		foundationRepo: foundationRepo,
		accountRepo:    accountRepo,
		journalRepo:    journalRepo,
		settingsRepo:   settingsRepo,
		journalSvc:     journalSvc,
		auditor:        auditor[uint]{auditRepo: auditRepo, entity: "GRUENDUNG"},
	}
}

// FoundingYearAligner setzt den Beginn des Gründungsjahres auf die Beurkundung.
//
// Ein Ausschnitt und kein Verweis auf den Abschlussdienst: gebraucht wird die
// eine Wirkung, nicht die Fähigkeit, einen Abschluss aufzustellen.
type FoundingYearAligner interface {
	AlignFoundingYear(ctx context.Context) error
}

// SetFoundingYearAligner koppelt das Geschäftsjahr an die Gründung. Ohne sie
// wird die Gründung erfasst und der Zeitraum bleibt, wie er war.
func (s *FoundationService) SetFoundingYearAligner(a FoundingYearAligner) { s.yearAligner = a }

// FoundationState is everything the Gründungsansicht needs, in one call.
//
// Zusammen und nicht einzeln, weil die Teile voneinander abhängen: die Regeln
// folgen aus der Rechtsform, die Unterbilanz aus ihr, dem Beurkundungsdatum und
// dem Journal. Mehrere Aufrufe könnten einen Stand zeigen, den es so nie gab.
type FoundationState struct {
	// HasFoundation ist falsch, solange keine Gründung erfasst wurde.
	HasFoundation bool `json:"hasFoundation"`

	LegalForm  string                     `json:"legalForm"`
	Rules      accounting.FoundationRules `json:"rules"`
	Foundation *domain.Foundation         `json:"foundation,omitempty"`
	Stage      domain.FoundationStage     `json:"stage"`

	Anmeldung   *domain.AnmeldungCheck `json:"anmeldung,omitempty"`
	Unterbilanz *domain.Unterbilanz    `json:"unterbilanz,omitempty"`

	// PostingsBooked sagt, ob die Gründungsbuchungen schon im Journal stehen.
	PostingsBooked bool `json:"postingsBooked"`
}

// foundationRulesOf liefert die Kapitalaufbringungsregeln der Rechtsform.
func foundationRulesOf(legalForm string) (accounting.FoundationRules, error) {
	rules, ok := accounting.FoundationRulesFor(legalForm)
	if !ok {
		return rules, fmt.Errorf("für die Rechtsform %q sind keine Gründungsregeln hinterlegt", legalForm)
	}
	return rules, nil
}

// GetState assembles the Gründungsansicht.
func (s *FoundationService) GetState(ctx context.Context) (*FoundationState, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}

	rules, err := foundationRulesOf(settings.LegalForm)
	if err != nil {
		return nil, err
	}
	state := &FoundationState{
		LegalForm: settings.LegalForm,
		Rules:     rules,
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

	booked, err := s.postingsBooked(ctx)
	if err != nil {
		return nil, err
	}
	state.PostingsBooked = booked

	return state, nil
}

// Save writes the Gründung after checking what only this service can check.
func (s *FoundationService) Save(ctx context.Context, f *domain.Foundation) (*domain.Foundation, error) {
	if err := requireHuman(ctx, "Die Erfassung der Gründung"); err != nil {
		return nil, err
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.Foundation, error) {
		return s.save(ctx, f)
	})
}

func (s *FoundationService) save(ctx context.Context, f *domain.Foundation) (*domain.Foundation, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := foundationRulesOf(settings.LegalForm)
	if err != nil {
		return nil, err
	}
	if err := validateFoundation(f, rules); err != nil {
		return nil, err
	}

	existing, err := s.foundationRepo.Get(ctx)
	if err == nil && existing != nil {
		f.ID = existing.ID
	}
	if err := s.foundationRepo.Save(ctx, f); err != nil {
		return nil, err
	}
	if err := s.audit(ctx, domain.AuditActionUpdate, f.ID, fmt.Sprintf(
		"Gründung erfasst: Beurkundung %s, Stammkapital %s €, %d Gesellschafter",
		f.NotarizedOn, f.ShareCapital, len(f.Shareholders))); err != nil {
		return nil, err
	}
	if err := s.adoptFoundingSettings(ctx, existing, f); err != nil {
		return nil, err
	}

	// Aus der Beurkundung folgt der Beginn des ersten Geschäftsjahres: eine
	// Gesellschaft, die im März entstanden ist, hat kein Geschäftsjahr, das im
	// Januar begonnen hätte. Scheitert das — weil der Zeitraum schon
	// festgeschrieben oder bebucht ist —, bleibt die Gründung trotzdem
	// gespeichert: sie ist die Tatsache, das Geschäftsjahr die Folge daraus.
	if s.yearAligner != nil {
		if err := s.yearAligner.AlignFoundingYear(ctx); err != nil {
			if err := s.audit(ctx, domain.AuditActionUpdate, f.ID, fmt.Sprintf(
				"Der Beginn des Gründungsjahres konnte nicht auf die Beurkundung gesetzt werden: %v", err)); err != nil {
				return nil, err
			}
		}
	}
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
		if err := validateUncalled(sh, rules); err != nil {
			return err
		}
	}

	if sum := f.SubscribedCapital(); sum != f.ShareCapital {
		return fmt.Errorf(
			"die übernommenen Geschäftsanteile ergeben %s €, das Stammkapital beträgt %s €. "+
				"Beides muss übereinstimmen (§ 5 Abs. 3 Satz 2 GmbHG)",
			sum, f.ShareCapital)
	}
	if called, required := f.ShareCapital-f.UncalledCapital(), rules.RequiredPaidIn(f); called < required {
		return fmt.Errorf(
			"eingefordert sind %s €; vor der Anmeldung müssen %s € eingezahlt sein (%s). "+
				"Der nicht eingeforderte Teil ist zu groß", called, required, rules.Reference)
	}
	return nil
}

// validateUncalled prüft den nicht eingeforderten Teil eines Geschäftsanteils.
// Eingefordert sein muss mindestens, was vor der Anmeldung einzuzahlen ist;
// eine Sacheinlage ist vollständig zu bewirken, bei der UG das ganze
// Stammkapital einzuzahlen.
func validateUncalled(sh *domain.Shareholder, rules accounting.FoundationRules) error {
	if sh.Uncalled < 0 {
		return fmt.Errorf("%s: der nicht eingeforderte Teil kann nicht negativ sein", sh.Name)
	}
	if sh.Uncalled > 0 {
		switch {
		case sh.Kind == domain.ContributionInKind:
			return fmt.Errorf(
				"%s: eine Sacheinlage ist vor der Anmeldung vollständig zu bewirken (§ 7 Abs. 3 GmbHG); "+
					"ein nicht eingeforderter Teil ist bei ihr nicht möglich", sh.Name)
		case rules.PaidInFloorIsFullCapital:
			return fmt.Errorf(
				"%s: bei der %s ist das Stammkapital vor der Anmeldung voll einzuzahlen (%s); "+
					"nicht eingefordert kann nichts bleiben", sh.Name, rules.LegalForm, rules.Reference)
		case sh.CalledIn() < rules.RequiredPerShare(*sh):
			return fmt.Errorf(
				"%s: eingefordert sind %s €, vor der Anmeldung einzuzahlen sind mindestens %s € (%s)",
				sh.Name, sh.CalledIn(), rules.RequiredPerShare(*sh), rules.Reference)
		}
	}
	if sh.PaidIn > sh.CalledIn() {
		return fmt.Errorf(
			"%s: geleistet sind %s €, eingefordert aber nur %s €. Wird mehr eingezahlt, haben die "+
				"Gesellschafter den Rest eingefordert (§ 46 Nr. 2 GmbHG); der nicht eingeforderte Teil "+
				"ist dann kleiner", sh.Name, sh.PaidIn, sh.CalledIn())
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
		asOf = s.today()
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
// Über alle Geschäftsjahre, nicht über das aktive: Die Vorbelastungshaftung hängt
// an ihrem Stichtag und nicht an einem Geschäftsjahr. Beurkundung im November,
// Eintragung im Februar ist der Regelfall — das Handelsregister braucht Wochen.
// Auf das aktive Jahr eingeschränkt zählte die Rechnung im Februar nur die
// Buchungen des neuen Jahres, und weil die Zeichnung des Stammkapitals im alten
// steht, wiese sie näherungsweise das volle Stammkapital als Unterbilanz aus.
//
// Der fehlende Saldenvortrag stört dabei nicht: Bis zur Eintragung sind Bewegung
// und Bestand dasselbe, denn vor der Beurkundung gab es keine Buchung.
func (s *FoundationService) netAssetsUntil(ctx context.Context, until string) (domain.Cents, domain.Cents, error) {
	entries, err := s.journalRepo.FindByBookingDateRange(ctx, 0, "", until)
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
				// gezeichneten Kapital auf der Passivseite (§ 272 Abs. 1 Satz 2
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
// Gebucht wird die Zeichnung am Tag der Beurkundung:
//
//	SOLL 1298 Ausstehende Einlagen, eingefordert       (eingeforderter Teil)
//	SOLL 2910 Ausstehende Einlagen, nicht eingefordert (Rest laut Vertrag)
//	HABEN 2900 Gezeichnetes Kapital
//
// Der eingeforderte, noch nicht eingezahlte Betrag ist eine Forderung, der
// nicht eingeforderte wird vom gezeichneten Kapital offen abgesetzt (§ 272
// Abs. 1 Satz 2 HGB). Die Einzahlung bucht der Bankumsatz: der Geldeingang auf
// dem Geschäftskonto wird dem Konto 1298 zugeordnet. Eine Einzahlung am Tag der
// Beurkundung ohne Kontoauszug stünde neben dem späteren Umsatz ein zweites
// Mal im Journal.
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

	uncalled := f.UncalledCapital()
	called := f.ShareCapital - uncalled
	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: domain.AccountAusstehendeEinlagenGeford, Amount: called},
	}
	if uncalled > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: domain.AccountAusstehendeEinlagenOffen, Amount: uncalled,
		})
	}
	lines = append(lines, domain.JournalLine{
		Side: domain.SideCredit, Account: domain.AccountGezeichnetesKapital, Amount: f.ShareCapital,
	})
	preview.Postings = append(preview.Postings, FoundationPosting{
		Title:       "Zeichnung des Stammkapitals",
		Date:        f.NotarizedOn,
		Description: "Übernahme der Geschäftsanteile laut Gesellschaftsvertrag",
		Reference:   "§ 272 Abs. 1 HGB",
		Amount:      f.ShareCapital,
		Lines:       lines,
	})
	if uncalled > 0 {
		preview.Skipped = append(preview.Skipped, fmt.Sprintf(
			"Nicht eingefordert sind %s €. Fordern die Gesellschafter den Rest ein (§ 46 Nr. 2 GmbHG), "+
				"bucht die Einforderung 1298 an 2910.", uncalled))
	}

	for _, sh := range f.Shareholders {
		if sh.Kind == domain.ContributionInKind {
			preview.Skipped = append(preview.Skipped, fmt.Sprintf(
				"%s leistet eine Sacheinlage über %s €. Sie gehört auf das Konto des "+
					"eingebrachten Gegenstands an 1298 und ist deshalb von Hand zu buchen.",
				sh.Name, sh.ShareCapital))
			continue
		}
		if sh.PaidIn > 0 {
			preview.Skipped = append(preview.Skipped, fmt.Sprintf(
				"Die Einzahlung von %s (%s €) bucht der Bankumsatz: den Geldeingang auf dem "+
					"Geschäftskonto dem Konto 1298 zuordnen.", sh.Name, sh.PaidIn))
		}
	}

	for _, p := range preview.Postings {
		preview.Total += p.Amount
	}
	return preview, nil
}

// BookPostings writes the founding bookings after the user released them.
func (s *FoundationService) BookPostings(ctx context.Context) ([]domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) ([]domain.JournalEntry, error) {
		return s.bookPostings(ctx)
	})
}

func (s *FoundationService) bookPostings(ctx context.Context) ([]domain.JournalEntry, error) {
	preview, err := s.PreviewPostings(ctx)
	if err != nil {
		return nil, err
	}
	if preview.AlreadyBooked {
		return nil, fmt.Errorf(
			"die Gründungsbuchungen stehen bereits im Journal. Eine Korrektur geschieht über den Storno")
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
			return nil, fmt.Errorf("%s konnte nicht gebucht werden: %w", p.Title, err)
		}
		created = append(created, *out)
	}
	return created, nil
}

// postingsBooked reports whether Gezeichnetes Kapital has already been booked.
//
// Über alle Jahre, aus demselben Grund wie die Unterbilanz: Die Zeichnung wird
// einmal gebucht, am Tag der Beurkundung. Auf das aktive Geschäftsjahr
// eingeschränkt meldete die Prüfung im Folgejahr „noch nicht gebucht" und böte
// die Zeichnung ein zweites Mal an.
func (s *FoundationService) postingsBooked(ctx context.Context) (bool, error) {
	entries, err := s.journalRepo.FindByAccount(ctx, domain.AccountGezeichnetesKapital, 0)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

// -------------------------------------------------------------
// Eintragung
// -------------------------------------------------------------

// Register records the entry in the Handelsregister — the end of the
// Vorgesellschaft.
//
// Ab hier besteht die Gesellschaft als juristische Person (§ 11 Abs. 1 GmbHG),
// die Handelndenhaftung des § 11 Abs. 2 GmbHG endet, und die Unterbilanz steht
// auf diesen Tag fest.
func (s *FoundationService) Register(ctx context.Context, date, court, number string) (*domain.Foundation, error) {
	if err := requireHuman(ctx, "Die Eintragung ins Handelsregister"); err != nil {
		return nil, err
	}
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.Foundation, error) {
		return s.register(ctx, date, court, number)
	})
}

func (s *FoundationService) register(ctx context.Context, date, court, number string) (*domain.Foundation, error) {
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

	// Registergericht und Registernummer sind zugleich Pflichtangaben im Kopf
	// jedes Jahresabschlusses (§ 264 Abs. 1a HGB). Sie hier zu übernehmen
	// erspart, dieselbe Tatsache ein zweites Mal einzutragen — überschrieben
	// wird nichts: was in den Einstellungen steht, hat jemand dort gewollt.
	if err := s.adoptRegisterSettings(ctx, f); err != nil {
		return nil, err
	}

	if err := s.audit(ctx, domain.AuditActionUpdate, f.ID, fmt.Sprintf(
		"Eintragung ins Handelsregister am %s: %s %s", date, f.RegisterCourt, f.RegisterNumber)); err != nil {
		return nil, err
	}
	return f, nil
}

// RegisterFromSettings übernimmt die Eintragung aus den Stammdaten. Ohne
// erfasste Gründung oder bei einer schon eingetragenen Gesellschaft ist nichts
// zu tun.
func (s *FoundationService) RegisterFromSettings(ctx context.Context, date, court, number string) error {
	return s.journalSvc.InTx(ctx, func(ctx context.Context) error {
		return s.registerFromSettings(ctx, date, court, number)
	})
}

func (s *FoundationService) registerFromSettings(ctx context.Context, date, court, number string) error {
	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return err
	}
	if f == nil || f.IsRegistered() {
		return nil
	}
	_, err = s.Register(ctx, date, court, number)
	return err
}

// adoptFoundingSettings trägt Gründungsdatum, Stammkapital und Gesellschafter in
// die Stammdaten ein, solange dort nichts steht oder noch der vorige Stand der
// Gründung. Was jemand in den Stammdaten davon abweichend gepflegt hat, bleibt.
func (s *FoundationService) adoptFoundingSettings(ctx context.Context, previous, f *domain.Foundation) error {
	if s.settingsRepo == nil {
		return nil
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil {
		return err
	}
	var before domain.Foundation
	if previous != nil {
		before = *previous
	}
	changed := false
	if (settings.FoundedOn == "" || settings.FoundedOn == before.NotarizedOn) && settings.FoundedOn != f.NotarizedOn {
		settings.FoundedOn = f.NotarizedOn
		changed = true
	}
	if (settings.ShareCapital == 0 || settings.ShareCapital == before.ShareCapital) && settings.ShareCapital != f.ShareCapital {
		settings.ShareCapital = f.ShareCapital
		changed = true
	}
	current := companyShareholders(f.Shareholders)
	if (len(settings.Shareholders) == 0 || sameShareholders(settings.Shareholders, companyShareholders(before.Shareholders))) &&
		!sameShareholders(settings.Shareholders, current) {
		settings.Shareholders = current
		changed = true
	}
	if !changed {
		return nil
	}
	if err := s.settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		return fmt.Errorf("die Unternehmensdaten ließen sich nicht aus der Gründung übernehmen: %w", err)
	}
	return logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", "COMPANY", fmt.Sprintf(
		"Aus der Gründung übernommen: Gründungsdatum %s, Stammkapital %s €, %d Gesellschafter",
		settings.FoundedOn, settings.ShareCapital, len(settings.Shareholders)))
}

func companyShareholders(shareholders []domain.Shareholder) []domain.CompanyShareholder {
	out := make([]domain.CompanyShareholder, 0, len(shareholders))
	for _, sh := range shareholders {
		out = append(out, domain.CompanyShareholder{Name: strings.TrimSpace(sh.Name), ShareCapital: sh.ShareCapital})
	}
	return out
}

func sameShareholders(a, b []domain.CompanyShareholder) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i].Name) != strings.TrimSpace(b[i].Name) || a[i].ShareCapital != b[i].ShareCapital {
			return false
		}
	}
	return true
}

// adoptRegisterSettings trägt Registergericht und Registernummer in die
// Unternehmensdaten nach. Die ausdrücklich erfasste Eintragung ist maßgeblich.
func (s *FoundationService) adoptRegisterSettings(ctx context.Context, f *domain.Foundation) error {
	if s.settingsRepo == nil {
		return nil
	}
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || settings == nil {
		return err
	}
	changed := false
	if settings.RegisteredOn != f.RegisteredOn {
		settings.RegisteredOn = f.RegisteredOn
		changed = true
	}
	if f.RegisterCourt != "" && settings.RegisterCourt != f.RegisterCourt {
		settings.RegisterCourt = f.RegisterCourt
		changed = true
	}
	if f.RegisterNumber != "" && settings.RegisterNumber != f.RegisterNumber {
		settings.RegisterNumber = f.RegisterNumber
		changed = true
	}
	// Der Sitz steht in den Gründungsdaten nicht eigens; der Ort der Anschrift
	// ist die nächste Tatsache, aus der er folgt. Der Sitz des § 264 Abs. 1a
	// Nr. 2 HGB ist die Gemeinde, deshalb ohne Postleitzahl.
	if seat := strings.TrimSpace(settings.City); settings.Seat == "" && seat != "" {
		settings.Seat = seat
		changed = true
	}
	if !changed {
		return nil
	}
	if err := s.settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		return fmt.Errorf("die Unternehmensdaten ließen sich nicht aus der Eintragung übernehmen: %w", err)
	}
	// Auch die abgeleitete Änderung der Unternehmensdaten gehört ins Protokoll;
	// sonst stünden im Kopf des Jahresabschlusses Angaben, die niemand gemacht hat.
	return logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", "COMPANY", fmt.Sprintf(
		"Aus der Handelsregistereintragung übernommen: Sitz %q, Registergericht %q, Registernummer %q",
		settings.Seat, settings.RegisterCourt, settings.RegisterNumber))
}
