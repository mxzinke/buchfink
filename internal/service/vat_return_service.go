package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
)

// ProgramVersion steht an jeder übermittelten Anmeldung.
//
// Die Zuordnung Steuerfall → Kennziffer kann sich ändern — der amtliche Vordruck
// ändert sich fast jedes Jahr. Ohne die Fassung, die gerechnet hat, ließe sich
// eine alte Anmeldung später nicht mehr nachvollziehen. Sie kommt deshalb aus
// dem Bau (internal/buildinfo) und trägt den Regelstand mit; ein fester Name
// änderte sich nie und sagte damit nichts.
func ProgramVersion() string {
	return buildinfo.Program(accounting.PostingRuleVersion)
}

// VatReturnService erzeugt, speichert und bestätigt Umsatzsteuer-Voranmeldungen.
//
// Buchfink übermittelt nicht selbst. Es erzeugt das Kennziffernblatt, der
// Anwender gibt es in Mein ELSTER ein und bestätigt die Übermittlung mit dem
// Transferticket. Die Entität ist danach unveränderlich — sie *ist* das
// Übermittlungsprotokoll, und ein Protokoll, das sich ändern lässt, ist keines.
type VatReturnService struct {
	journalRepo        domain.JournalRepository
	receiptRepo        domain.ReceiptRepository
	contactRepo        domain.ContactRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository
	returnRepo         domain.VatReturnRepository
	auditRepo          domain.AuditRepository
	fiscalYear         int
}

// NewVatReturnService wires the Voranmeldung.
func NewVatReturnService(
	journalRepo domain.JournalRepository,
	receiptRepo domain.ReceiptRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	festschreibungRepo domain.FestschreibungRepository,
	returnRepo domain.VatReturnRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *VatReturnService {
	return &VatReturnService{
		journalRepo:        journalRepo,
		receiptRepo:        receiptRepo,
		contactRepo:        contactRepo,
		settingsRepo:       settingsRepo,
		festschreibungRepo: festschreibungRepo,
		returnRepo:         returnRepo,
		auditRepo:          auditRepo,
		fiscalYear:         fiscalYear,
	}
}

// SetFiscalYear updates the active fiscal year.
func (s *VatReturnService) SetFiscalYear(year int) { s.fiscalYear = year }

// VatPeriodStatus ist ein Zeitraum mit dem Stand seiner Anmeldung.
type VatPeriodStatus struct {
	accounting.VatPeriod
	DueDate string                 `json:"dueDate"`
	Status  domain.VatReturnStatus `json:"status"`
	// ReturnID ist die gespeicherte Anmeldung, falls es eine gibt.
	ReturnID uint `json:"returnId,omitempty"`
	// Committed meldet, ob der Zeitraum festgeschrieben ist — ohne
	// Festschreibung ist die Bestätigung gesperrt.
	Committed   bool         `json:"committed"`
	Payable     domain.Cents `json:"payable"`
	SubmittedAt string       `json:"submittedAt,omitempty"`
	IsOverdue   bool         `json:"isOverdue"`
}

// PeriodType ist der Voranmeldungszeitraum des Mandanten (§ 18 Abs. 2 UStG).
func (s *VatReturnService) PeriodType(ctx context.Context) domain.VatPeriodType {
	if s.settingsRepo == nil {
		return domain.VatPeriodQuarter
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return domain.VatPeriodQuarter
	}
	return vatPeriodTypeOf(cfg)
}

// vatPeriodTypeOf liest den Voranmeldungszeitraum aus den Unternehmensdaten.
//
// Die Regel steht an einer Stelle, weil drei Dienste dieselbe Frage stellen —
// Voranmeldung, Prüflauf und Fristenliste. Ein unbekannter oder leerer Wert gilt
// als Quartal: das ist der Regelfall des § 18 Abs. 2 Satz 1 UStG, und er mahnt
// niemanden zu einer monatlichen Abgabe, die er nicht schuldet.
func vatPeriodTypeOf(cfg *domain.CompanySettings) domain.VatPeriodType {
	if cfg == nil {
		return domain.VatPeriodQuarter
	}
	t := domain.VatPeriodType(cfg.VatPeriod)
	if !t.Valid() {
		return domain.VatPeriodQuarter
	}
	return t
}

// Periods liefert die Zeiträume eines Jahres mit Fälligkeit und Stand.
func (s *VatReturnService) Periods(ctx context.Context, year int) ([]VatPeriodStatus, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	cfg, err := s.companySettings(ctx)
	if err != nil {
		return nil, err
	}
	periods := accounting.VatPeriodsOfYear(year, s.PeriodType(ctx))

	saved, err := s.returnRepo.FindByFiscalYear(ctx, year)
	if err != nil {
		return nil, err
	}
	// Die jüngste Anmeldung eines Zeitraums gewinnt: eine Berichtigung tritt an
	// die Stelle der berichtigten Anmeldung.
	latest := map[string]*domain.VatReturn{}
	for i := range saved {
		r := &saved[i]
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}

	cutoff := ""
	if s.festschreibungRepo != nil {
		cutoff, _ = s.festschreibungRepo.LatestCutoff(ctx, year)
	}
	today := time.Now().Format("2006-01-02")

	out := make([]VatPeriodStatus, 0, len(periods))
	for _, p := range periods {
		st := VatPeriodStatus{
			VatPeriod: p,
			DueDate:   accounting.VatDueDate(p, cfg.PermanentExtension),
			Status:    domain.VatReturnDraft,
			Committed: cutoff != "" && cutoff >= p.To,
		}
		if r := latest[p.Key]; r != nil {
			st.Status = r.Status
			st.ReturnID = r.ID
			st.Payable = r.Payable
			st.SubmittedAt = r.SubmittedAt
		}
		st.IsOverdue = st.Status != domain.VatReturnSubmitted && st.DueDate != "" && st.DueDate < today
		out = append(out, st)
	}
	return out, nil
}

// Draft rechnet das Kennziffernblatt eines Zeitraums neu, ohne es zu speichern.
//
// Liegt für den Zeitraum bereits eine berichtigte Anmeldung als Entwurf vor,
// trägt auch das neu gerechnete Blatt die Kennziffer 10. Ohne diese Übernahme
// wäre die Berichtigung auf dem Blatt von einer Erstanmeldung nicht zu
// unterscheiden — dabei ist Kennziffer 10 genau das Merkmal, an dem das
// Finanzamt erkennt, dass die Anmeldung eine frühere ersetzt (§ 153 AO).
func (s *VatReturnService) Draft(ctx context.Context, periodKey string) (*domain.VatReturn, error) {
	period, err := accounting.ParseVatPeriodKey(periodKey)
	if err != nil {
		return nil, err
	}
	ret, err := s.build(ctx, period)
	if err != nil {
		return nil, err
	}
	existing, err := s.openDraft(ctx, period.Key)
	if err != nil {
		return nil, err
	}
	// Die Nachtragsliste bleibt stehen: sie führt Buchungen *früherer*
	// übermittelter Zeiträume, und die berichtigt dieses Blatt nicht mit.
	if existing != nil && existing.IsCorrection {
		ret.IsCorrection = true
		ret.CorrectsID = existing.CorrectsID
	}
	return ret, nil
}

// Save legt das Blatt als Entwurf ab oder schreibt einen bestehenden Entwurf
// fort. Eine bestätigte Anmeldung wird nicht überschrieben.
//
// Ist der Zeitraum bereits übermittelt, entsteht hier *kein* zweiter Entwurf.
// Er wäre eine zweite Erstanmeldung desselben Zeitraums — ohne Kennziffer 10,
// ohne Bezug auf die erste, und das Übermittlungsprotokoll führte für einen
// Zeitraum zwei Originalanmeldungen. Geändert wird eine übermittelte Anmeldung
// nur über eine Berichtigung (§ 18 Abs. 1 UStG i. V. m. § 153 AO), und die legt
// CreateCorrection an.
func (s *VatReturnService) Save(ctx context.Context, periodKey string) (*domain.VatReturn, error) {
	period, err := accounting.ParseVatPeriodKey(periodKey)
	if err != nil {
		return nil, err
	}

	existing, err := s.openDraft(ctx, period.Key)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		submitted, err := s.latestSubmitted(ctx, period.Key)
		if err != nil {
			return nil, err
		}
		if submitted != nil {
			return nil, fmt.Errorf(
				"für den Zeitraum %s ist am %s bereits eine Voranmeldung übermittelt (Transferticket %s). "+
					"Ein zweiter Entwurf wäre eine zweite Erstanmeldung desselben Zeitraums — lege stattdessen "+
					"eine berichtigte Anmeldung an (Kennziffer 10)",
				period.Key, submitted.SubmittedAt, submitted.TransferTicket)
		}
	}

	fresh, err := s.build(ctx, period)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		fresh.ID = existing.ID
		fresh.CreatedAt = existing.CreatedAt
		fresh.IsCorrection = existing.IsCorrection
		fresh.CorrectsID = existing.CorrectsID
		if err := s.returnRepo.Update(ctx, fresh); err != nil {
			return nil, fmt.Errorf("die Voranmeldung konnte nicht gespeichert werden: %w", err)
		}
	} else if err := s.returnRepo.Create(ctx, fresh); err != nil {
		return nil, fmt.Errorf("die Voranmeldung konnte nicht gespeichert werden: %w", err)
	}

	s.audit(ctx, domain.AuditActionCreate, fresh.ID, fmt.Sprintf(
		"Umsatzsteuer-Voranmeldung %s als Entwurf gespeichert (Zahllast %s €)", fresh.PeriodKey, fresh.Payable))
	return fresh, nil
}

// SpecialPrepaymentSuggestion ist der Vorschlag für die Sondervorauszahlung
// eines Jahres (§ 47 Abs. 1 UStDV).
type SpecialPrepaymentSuggestion struct {
	// Year ist das Jahr, für das die Dauerfristverlängerung gilt; BasedOnYear
	// das Vorjahr, aus dem gerechnet wird.
	Year        int `json:"year"`
	BasedOnYear int `json:"basedOnYear"`
	// Amount ist der Vorschlag: ein Elftel der Summe, auf volle Euro
	// abgerundet. Der Vordruck USt 1 H kennt keine Cent.
	Amount domain.Cents `json:"amount"`
	// PrepaymentSum ist die Summe der Vorauszahlungen des Vorjahres — vor
	// Anrechnung der damaligen Sondervorauszahlung.
	PrepaymentSum domain.Cents `json:"prepaymentSum"`
	// Periods sind die übermittelten Anmeldungen, aus denen die Summe entstand.
	// Ohne sie ist der Vorschlag eine Zahl, die niemand nachrechnen kann.
	Periods []SpecialPrepaymentPeriod `json:"periods"`
	// Applicable meldet, ob der Mandant überhaupt eine Sondervorauszahlung
	// schuldet — sie setzt den monatlichen Voranmeldungszeitraum voraus
	// (§ 47 Abs. 1 UStDV). Ohne dieses Merkmal wäre ein Vorschlag von null Euro
	// nicht von „nichts zu berechnen" zu unterscheiden.
	Applicable bool `json:"applicable"`
	// Complete meldet, ob für jeden Zeitraum des Vorjahres eine übermittelte
	// Anmeldung vorliegt.
	Complete bool `json:"complete"`
	// Account ist das Konto, auf das die Zahlung gebucht wird (SKR04 3830
	// „Umsatzsteuer-Vorauszahlungen 1/11").
	Account string `json:"account"`
	// Note sagt, was der Vorschlag wert ist — und wo er es nicht ist.
	Note string `json:"note"`
}

// SpecialPrepaymentPeriod ist eine Anmeldung des Vorjahres im Vorschlag.
type SpecialPrepaymentPeriod struct {
	PeriodKey   string `json:"periodKey"`
	PeriodLabel string `json:"periodLabel"`
	ReturnID    uint   `json:"returnId"`
	SubmittedAt string `json:"submittedAt"`
	// Prepayment ist die Vorauszahlung des Zeitraums (Kennziffer 83) zuzüglich
	// der dort angerechneten Sondervorauszahlung (Kennziffer 39).
	Prepayment domain.Cents `json:"prepayment"`
}

// SuggestedSpecialPrepayment rechnet die Sondervorauszahlung aus den
// übermittelten Voranmeldungen des Vorjahres.
//
// § 47 Abs. 1 UStDV: ein Elftel der Summe der Vorauszahlungen für das
// vorangegangene Kalenderjahr. Maßgeblich ist die Vorauszahlung *vor* Anrechnung
// der damaligen Sondervorauszahlung — sonst minderte die Anrechnung des Vorjahres
// den Vorschlag für dieses Jahr, und die Sondervorauszahlung schrumpfte Jahr für
// Jahr, ohne dass die Umsätze es täten. Deshalb wird Kennziffer 39 der
// Dezember- bzw. Q4-Anmeldung wieder hinzugerechnet.
//
// Vorgeschlagen wird, nicht gesetzt: angemeldet und gezahlt wird außerhalb von
// Buchfink, und maßgeblich bleibt, was der Anwender angemeldet hat.
//
// Vorgeschlagen wird nur dem Monatszahler. § 47 Abs. 1 UStDV knüpft die
// Sondervorauszahlung an die monatliche Abgabe; wer vierteljährlich voranmeldet,
// erhält die Dauerfristverlängerung nach § 46 UStDV ohne sie. Ein Vorschlag an
// ihn wäre die Aufforderung, etwas anzumelden und zu zahlen, das er nicht
// schuldet.
func (s *VatReturnService) SuggestedSpecialPrepayment(ctx context.Context, year int) (*SpecialPrepaymentSuggestion, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	prior := year - 1
	out := &SpecialPrepaymentSuggestion{
		Year:        year,
		BasedOnYear: prior,
		Account:     domain.AccountSondervorauszahlung,
		Periods:     make([]SpecialPrepaymentPeriod, 0, 12),
	}

	if periodType := s.PeriodType(ctx); periodType != domain.VatPeriodMonth {
		out.Applicable = false
		out.Note = "Die Sondervorauszahlung gibt es nur bei monatlicher Voranmeldung (§ 47 Abs. 1 UStDV). " +
			"Bei vierteljährlicher Abgabe wird die Dauerfristverlängerung ohne Sondervorauszahlung " +
			"gewährt (§ 46 UStDV)."
		return out, nil
	}
	out.Applicable = true

	saved, err := s.returnRepo.FindByFiscalYear(ctx, prior)
	if err != nil {
		return nil, fmt.Errorf("die Voranmeldungen des Jahres %d konnten nicht gelesen werden: %w", prior, err)
	}
	// Je Zeitraum zählt die jüngste bestätigte Anmeldung: eine Berichtigung
	// tritt an die Stelle der berichtigten und meldet den Zeitraum neu.
	latest := map[string]*domain.VatReturn{}
	for i := range saved {
		r := &saved[i]
		if r.Status != domain.VatReturnSubmitted {
			continue
		}
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}

	periods := accounting.VatPeriodsOfYear(prior, s.PeriodType(ctx))
	labels := map[string]string{}
	for _, p := range periods {
		labels[p.Key] = p.Label
	}

	for key, r := range latest {
		// Die Vorauszahlung des Zeitraums vor Anrechnung der
		// Sondervorauszahlung.
		amount := r.Payable + r.Tax(accounting.VatCodeSpecialPrepayment)
		out.PrepaymentSum += amount
		out.Periods = append(out.Periods, SpecialPrepaymentPeriod{
			PeriodKey:   key,
			PeriodLabel: labels[key],
			ReturnID:    r.ID,
			SubmittedAt: r.SubmittedAt,
			Prepayment:  amount,
		})
	}
	sort.SliceStable(out.Periods, func(i, j int) bool { return out.Periods[i].PeriodKey < out.Periods[j].PeriodKey })

	missing := 0
	for _, p := range periods {
		if latest[p.Key] == nil {
			missing++
		}
	}
	out.Complete = len(latest) > 0 && missing == 0

	switch {
	case len(latest) == 0:
		out.Note = fmt.Sprintf(
			"Für %d ist keine übermittelte Voranmeldung erfasst. Die Sondervorauszahlung ist dann nach den "+
				"voraussichtlichen Vorauszahlungen des laufenden Jahres zu schätzen (§ 47 Abs. 3 UStDV).", prior)
		return out, nil

	case out.PrepaymentSum <= 0:
		out.Note = fmt.Sprintf(
			"Die Vorauszahlungen des Jahres %d ergeben in der Summe %s € — daraus folgt keine "+
				"Sondervorauszahlung.", prior, out.PrepaymentSum)
		return out, nil
	}

	// Ein Elftel, auf volle Euro abgerundet: der Vordruck USt 1 H nimmt keine
	// Cent an.
	out.Amount = accounting.TruncToEuro(out.PrepaymentSum / 11)
	out.Note = fmt.Sprintf(
		"Ein Elftel der Vorauszahlungen %d (%s €) aus %d übermittelten Anmeldungen.",
		prior, out.PrepaymentSum, len(out.Periods))
	if !out.Complete {
		out.Note += fmt.Sprintf(
			" Für %d der %d Zeiträume des Jahres %d liegt keine übermittelte Anmeldung vor — prüfe den "+
				"Vorschlag. Wurde die Tätigkeit nur einen Teil des Jahres ausgeübt, ist die Summe nach "+
				"§ 47 Abs. 2 UStDV auf ein volles Jahr hochzurechnen.", missing, len(periods), prior)
	}
	return out, nil
}

// List liefert die gespeicherten Anmeldungen eines Jahres.
func (s *VatReturnService) List(ctx context.Context, year int) ([]domain.VatReturn, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	return s.returnRepo.FindByFiscalYear(ctx, year)
}

// ConfirmSubmitted bestätigt die Übermittlung in Mein ELSTER.
//
// Zwei Bedingungen, und beide sind keine Förmlichkeit. Der Zeitraum muss
// festgeschrieben sein: was gemeldet ist, darf sich nicht mehr ändern, sonst
// weicht die nächste Auswertung von der Anmeldung ab. Und das Transferticket ist
// der einzige Nachweis, dass die Anmeldung angekommen ist.
func (s *VatReturnService) ConfirmSubmitted(ctx context.Context, id uint, date, ticket, note string) (*domain.VatReturn, error) {
	rec, err := s.returnRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("die Voranmeldung %d wurde nicht gefunden: %w", id, err)
	}
	if err := rec.ValidateSubmission(date, ticket); err != nil {
		return nil, err
	}
	if err := s.ensureCommitted(ctx, rec); err != nil {
		return nil, err
	}
	if err := s.ensureFirstOrCorrection(ctx, rec); err != nil {
		return nil, err
	}
	if err := s.ensureCurrent(ctx, rec); err != nil {
		return nil, err
	}

	rec.Status = domain.VatReturnSubmitted
	rec.SubmittedAt = date
	rec.TransferTicket = strings.TrimSpace(ticket)
	rec.SubmissionNote = note
	if err := s.returnRepo.Update(ctx, rec); err != nil {
		return nil, fmt.Errorf("die Bestätigung konnte nicht gespeichert werden: %w", err)
	}

	// Protokolliert wird ein Statuswechsel und keine Datei-Ausgabe: die
	// Bestätigung ändert die Entität (Entwurf → übermittelt), das ExportCSV
	// daneben gibt nur aus. Stünden beide unter EXPORT, wären sie im Protokoll
	// nur am Text zu unterscheiden — und wer nach dem Weg einer Anmeldung sucht,
	// filtert nach der Aktion.
	s.audit(ctx, domain.AuditActionUpdate, rec.ID, fmt.Sprintf(
		"Umsatzsteuer-Voranmeldung %s am %s übermittelt (Transferticket %s, Zahllast %s €)",
		rec.PeriodKey, date, rec.TransferTicket, rec.Payable))
	return rec, nil
}

// CreateCorrection legt eine berichtigte Anmeldung für einen bereits
// übermittelten Zeitraum an (Kennziffer 10).
//
// Sie ist eine vollständige Neuanmeldung des Zeitraums und keine Differenz: das
// Finanzamt ersetzt die alte durch die neue. Deshalb wird das Blatt aus dem
// heutigen Journalstand gerechnet — die Nachträge sind dann keine Nachträge mehr,
// sondern stehen an ihrem Platz.
func (s *VatReturnService) CreateCorrection(ctx context.Context, periodKey string) (*domain.VatReturn, error) {
	period, err := accounting.ParseVatPeriodKey(periodKey)
	if err != nil {
		return nil, err
	}
	submitted, err := s.latestSubmitted(ctx, period.Key)
	if err != nil {
		return nil, err
	}
	if submitted == nil {
		return nil, fmt.Errorf(
			"für den Zeitraum %s ist keine übermittelte Voranmeldung erfasst. Eine Berichtigung setzt voraus, "+
				"dass die ursprüngliche Anmeldung abgegeben wurde", period.Key)
	}

	fresh, err := s.build(ctx, period)
	if err != nil {
		return nil, err
	}
	fresh.IsCorrection = true
	fresh.CorrectsID = &submitted.ID
	// Die Berichtigung meldet den Zeitraum vollständig neu. Was vorher ein
	// Nachtrag war, ist jetzt Teil der Anmeldung — eine Nachtragsliste an ihr
	// wäre die Doppelzählung derselben Buchung.
	fresh.LateEntries = nil
	if err := s.returnRepo.Create(ctx, fresh); err != nil {
		return nil, fmt.Errorf("die Berichtigung konnte nicht gespeichert werden: %w", err)
	}

	s.audit(ctx, domain.AuditActionUpdate, fresh.ID, fmt.Sprintf(
		"Berichtigte Voranmeldung %s angelegt (berichtigt Anmeldung %d, Zahllast %s € statt %s €)",
		fresh.PeriodKey, submitted.ID, fresh.Payable, submitted.Payable))
	return fresh, nil
}

// ExportCSV liefert das Kennziffernblatt als Datei zum Abtippen in Mein ELSTER.
//
// Ausgegeben werden nur die Kennziffern mit einem Wert. Eine Datei, die achtzig
// Nullen enthält, liest niemand — und in ELSTER ist eine getippte Null eine
// Angabe und keine Auslassung.
func (s *VatReturnService) ExportCSV(ctx context.Context, id uint) (string, error) {
	rec, err := s.returnRepo.FindByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("die Voranmeldung %d wurde nicht gefunden: %w", id, err)
	}

	var b strings.Builder
	b.WriteString("Kennziffer;Wert\n")
	if rec.IsCorrection {
		b.WriteString("10;1\n")
	}
	for _, line := range rec.Figures {
		if line.HasBase && line.Base != 0 {
			// Bemessungsgrundlagen trägt der Vordruck in vollen Euro.
			fmt.Fprintf(&b, "%s;%d\n", line.Code, int64(line.Base)/100)
		}
		if line.HasTax && (line.Tax != 0 || line.Code == accounting.VatCodePayable) {
			code := line.Code
			if line.TaxCode != "" {
				code = line.TaxCode
			}
			fmt.Fprintf(&b, "%s;%s\n", code, line.Tax.Decimal())
		}
	}

	s.audit(ctx, domain.AuditActionExport, rec.ID, fmt.Sprintf(
		"Umsatzsteuer-Voranmeldung %s als Datei ausgegeben", rec.PeriodKey))
	return b.String(), nil
}

// -------------------------------------------------------------
// Innenleben
// -------------------------------------------------------------

// build rechnet ein Blatt aus dem Journal.
func (s *VatReturnService) build(ctx context.Context, period accounting.VatPeriod) (*domain.VatReturn, error) {
	cfg, err := s.companySettings(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := s.entriesFor(ctx, period)
	if err != nil {
		return nil, err
	}
	submitted, reported, err := s.submittedPeriods(ctx, period)
	if err != nil {
		return nil, err
	}
	receivedAt, err := s.receiptDates(ctx, period.Year)
	if err != nil {
		return nil, err
	}
	euRecipients, err := s.euRecipients(ctx)
	if err != nil {
		return nil, err
	}

	source := accounting.VatReturnSource{
		Entries:    entries,
		ReceivedAt: func(receiptID uint) string { return receivedAt[receiptID] },
		EURecipient: func(contactID uint) bool {
			return euRecipients[contactID]
		},
		SubmittedPeriod: func(date string) bool {
			p, err := accounting.VatPeriodOf(date, period.Type)
			if err != nil {
				return false
			}
			return submitted[p.Key]
		},
		ReportedEntry: func(date string, entryID uint) bool {
			p, err := accounting.VatPeriodOf(date, period.Type)
			if err != nil {
				return false
			}
			return reported[p.Key][entryID]
		},
		SpecialPrepayment: s.specialPrepaymentFor(period, cfg),
	}

	ret := accounting.BuildVatReturn(period, source)
	ret.DueDate = accounting.VatDueDate(period, cfg.PermanentExtension)
	ret.ProgramVersion = ProgramVersion()
	return ret, nil
}

// specialPrepaymentFor rechnet die Sondervorauszahlung nur im letzten Zeitraum
// des Jahres an (§ 48 Abs. 4 UStDV) und nur, wenn die Dauerfristverlängerung
// besteht.
func (s *VatReturnService) specialPrepaymentFor(period accounting.VatPeriod, cfg *domain.CompanySettings) domain.Cents {
	if !cfg.PermanentExtension || cfg.SpecialPrepayment == 0 {
		return 0
	}
	// Nur der Monatszahler leistet eine Sondervorauszahlung: § 47 Abs. 1 UStDV
	// verlangt sie von den Unternehmern, die ihre Voranmeldungen monatlich
	// abzugeben haben. Der Vierteljahreszahler bekommt die
	// Dauerfristverlängerung nach § 46 UStDV ohne sie, der Jahreszahler gibt gar
	// keine Voranmeldung ab. Wer hier auch ihnen etwas anrechnete, minderte eine
	// Zahllast um einen Betrag, den niemand vorausgezahlt hat.
	if period.Type != domain.VatPeriodMonth || vatPeriodTypeOf(cfg) != domain.VatPeriodMonth {
		return 0
	}
	if period.To != fmt.Sprintf("%d-12-31", period.Year) {
		return 0
	}
	return cfg.SpecialPrepayment
}

// entriesFor liest die Buchungen, die in den Zeitraum hineinwirken können.
//
// Gelesen wird über drei Geschäftsjahre, ausgewählt wird über das Datum. Das
// Geschäftsjahr einer Buchung folgt ihrem Buchungsdatum, der
// Voranmeldungszeitraum dagegen dem Entstehen der Steuer — und die beiden fallen
// regelmäßig auseinander: die im Januar erfasste Dezemberleistung, die
// Generalumkehr aus dem Folgejahr, und bei abweichendem Wirtschaftsjahr
// (FiscalYearStartMonth ≠ 1) jeder Kalendermonat, der im anderen Geschäftsjahr
// liegt. Wer zu welchem Zeitraum gehört, entscheidet allein VatPeriodFor.
func (s *VatReturnService) entriesFor(ctx context.Context, period accounting.VatPeriod) ([]domain.JournalEntry, error) {
	entries, err := s.journalRepo.FindAll(ctx, period.Year)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen des Jahres %d konnten nicht gelesen werden: %w", period.Year, err)
	}
	for _, year := range []int{period.Year - 1, period.Year + 1} {
		if more, err := s.journalRepo.FindAll(ctx, year); err == nil {
			entries = append(entries, more...)
		}
	}
	return entries, nil
}

// submittedPeriods sammelt die Zeiträume mit bestätigter Übermittlung und die
// Buchungen, die darin gemeldet wurden.
//
// Beides zusammen entscheidet über einen Nachtrag: er ist eine Buchung, die in
// einen übermittelten Zeitraum gehört und dort *nicht* gemeldet wurde. Ohne die
// zweite Hälfte wäre jede gemeldete Buchung ihr eigener Nachtrag.
func (s *VatReturnService) submittedPeriods(
	ctx context.Context, period accounting.VatPeriod,
) (map[string]bool, map[string]map[uint]bool, error) {
	saved, err := s.returnRepo.FindByFiscalYear(ctx, period.Year)
	if err != nil {
		return nil, nil, err
	}
	// Je Zeitraum zählt die jüngste bestätigte Anmeldung: eine Berichtigung
	// tritt an die Stelle der berichtigten und meldet den Zeitraum vollständig
	// neu.
	latest := map[string]*domain.VatReturn{}
	for i := range saved {
		r := &saved[i]
		if r.Status != domain.VatReturnSubmitted {
			continue
		}
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}

	submitted := map[string]bool{}
	reported := map[string]map[uint]bool{}
	for key, r := range latest {
		submitted[key] = true
		ids := map[uint]bool{}
		for _, line := range r.Figures {
			for _, id := range line.EntryIDs {
				ids[id] = true
			}
		}
		reported[key] = ids
	}
	return submitted, reported, nil
}

// receiptDates liefert den Belegeingang je Beleg. Er entscheidet mit über den
// Zeitraum des Vorsteuerabzugs (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG).
func (s *VatReturnService) receiptDates(ctx context.Context, year int) (map[uint]string, error) {
	out := map[uint]string{}
	if s.receiptRepo == nil {
		return out, nil
	}
	// Dieselben drei Jahre wie bei den Buchungen: ein Beleg des Vorjahres, der
	// zu einer Buchung dieses Zeitraums gehört, entscheidet über deren
	// Vorsteuerzeitraum mit.
	for _, y := range []int{year - 1, year, year + 1} {
		receipts, err := s.receiptRepo.FindAll(ctx, y)
		if err != nil {
			continue
		}
		for i := range receipts {
			if receipts[i].ReceivedAt != "" {
				out[receipts[i].ID] = receipts[i].ReceivedAt
			}
		}
	}
	return out, nil
}

// euRecipients sammelt die Geschäftspartner im übrigen Gemeinschaftsgebiet mit
// USt-IdNr. Ohne beides ist eine Leistung mit Steuerschuld des Empfängers kein
// Fall des § 18b Satz 1 Nr. 2 UStG.
func (s *VatReturnService) euRecipients(ctx context.Context) (map[uint]bool, error) {
	out := map[uint]bool{}
	if s.contactRepo == nil {
		return out, nil
	}
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return out, nil
	}
	for i := range contacts {
		c := &contacts[i]
		out[c.ID] = c.IsEUCounterparty() && strings.TrimSpace(c.VatID) != ""
	}
	return out, nil
}

// ensureCommitted verlangt die Festschreibung des Zeitraums vor der Bestätigung.
func (s *VatReturnService) ensureCommitted(ctx context.Context, rec *domain.VatReturn) error {
	if s.festschreibungRepo == nil {
		return nil
	}
	cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, rec.FiscalYear)
	if err != nil {
		return fmt.Errorf("der Festschreibungsstand konnte nicht geprüft werden: %w", err)
	}
	if cutoff == "" || cutoff < rec.PeriodTo {
		return fmt.Errorf(
			"der Zeitraum %s (bis %s) ist noch nicht festgeschrieben. Eine übermittelte Voranmeldung muss sich "+
				"auf einen unveränderlichen Stand stützen — schreibe den Zeitraum zuerst fest",
			rec.PeriodKey, rec.PeriodTo)
	}
	return nil
}

// ensureFirstOrCorrection lässt für einen Zeitraum nur eine Erstanmeldung zu.
//
// Jede weitere Anmeldung desselben Zeitraums ist eine Berichtigung und trägt die
// Kennziffer 10. Zwei Originalanmeldungen für einen Zeitraum wären für das
// Finanzamt zwei Erklärungen, von denen keine die andere ersetzt — und im
// Übermittlungsprotokoll zwei Wahrheiten über denselben Monat.
func (s *VatReturnService) ensureFirstOrCorrection(ctx context.Context, rec *domain.VatReturn) error {
	if rec.IsCorrection {
		return nil
	}
	submitted, err := s.latestSubmitted(ctx, rec.PeriodKey)
	if err != nil {
		return err
	}
	if submitted == nil || submitted.ID == rec.ID {
		return nil
	}
	return fmt.Errorf(
		"für den Zeitraum %s ist am %s bereits eine Voranmeldung übermittelt (Transferticket %s). "+
			"Eine zweite Erstanmeldung desselben Zeitraums gibt es nicht — bestätige sie als berichtigte "+
			"Anmeldung (Kennziffer 10) oder lege eine Berichtigung an",
		rec.PeriodKey, submitted.SubmittedAt, submitted.TransferTicket)
}

// ensureCurrent verlangt, dass der Entwurf noch dem Journal entspricht.
//
// Der Entwurf ist eine Momentaufnahme. Wurde nach dem Speichern noch gebucht,
// stünden im Protokoll Kennziffern, die der Anwender so gar nicht übermittelt
// hat — und niemandem fiele es auf, weil das Protokoll seine eigene Quelle ist.
func (s *VatReturnService) ensureCurrent(ctx context.Context, rec *domain.VatReturn) error {
	period, err := accounting.ParseVatPeriodKey(rec.PeriodKey)
	if err != nil {
		return err
	}
	fresh, err := s.build(ctx, period)
	if err != nil {
		return err
	}
	if fresh.Payable == rec.Payable && sameFigures(fresh.Figures, rec.Figures) {
		return nil
	}
	return fmt.Errorf(
		"der gespeicherte Entwurf der Voranmeldung %s stimmt nicht mehr mit dem Journal überein "+
			"(Zahllast im Entwurf %s €, aus dem Journal %s €). Speichere ihn neu und bestätige dann die "+
			"Übermittlung — bestätigt wird, was übermittelt wurde",
		rec.PeriodKey, rec.Payable, fresh.Payable)
}

// sameFigures vergleicht zwei Kennziffernblätter in den Zahlen, die übermittelt
// werden. Der Drill-down bleibt außen vor: er ändert sich schon, wenn eine
// Buchung eine andere Nummer bekommt, ohne dass sich eine Kennziffer bewegt.
func sameFigures(a, b []domain.VatReturnLine) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Code != b[i].Code || a[i].Base != b[i].Base || a[i].Tax != b[i].Tax {
			return false
		}
	}
	return true
}

// openDraft liefert den noch nicht bestätigten Entwurf eines Zeitraums.
func (s *VatReturnService) openDraft(ctx context.Context, periodKey string) (*domain.VatReturn, error) {
	saved, err := s.returnRepo.FindByPeriod(ctx, periodKey)
	if err != nil {
		return nil, err
	}
	for i := range saved {
		if saved[i].Status == domain.VatReturnDraft {
			return &saved[i], nil
		}
	}
	return nil, nil
}

// latestSubmitted liefert die jüngste bestätigte Anmeldung eines Zeitraums.
func (s *VatReturnService) latestSubmitted(ctx context.Context, periodKey string) (*domain.VatReturn, error) {
	saved, err := s.returnRepo.FindByPeriod(ctx, periodKey)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(saved, func(i, j int) bool { return saved[i].ID > saved[j].ID })
	for i := range saved {
		if saved[i].Status == domain.VatReturnSubmitted {
			return &saved[i], nil
		}
	}
	return nil, nil
}

func (s *VatReturnService) companySettings(ctx context.Context) (*domain.CompanySettings, error) {
	if s.settingsRepo == nil {
		return &domain.CompanySettings{VatPeriod: "quarter", ReceiptCaptureDays: 10}, nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}
	return cfg, nil
}

func (s *VatReturnService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "VAT_RETURN", fmt.Sprintf("%d", id), details)
}
