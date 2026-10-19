package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/calendar"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Konten, an denen Buchfink gezahlte Vorauszahlungen und das gebuchte
// Lohnjournal erkennt (SKR04).
const (
	accountSpecialPrepayment = "3830"
	accountPayrollTax        = "3730"
	accountCorporateTax      = "7600"
	accountSolidarityTax     = "7608"
)

// Die Grenzen des Lohnsteuer-Anmeldungszeitraums (§ 41a Abs. 2 Satz 2 EStG).
const (
	payrollYearlyLimit    domain.Cents = 108_000
	payrollQuarterlyLimit domain.Cents = 500_000
)

// prepaymentWindowDays: eine Zahlung zählt für den Vorauszahlungstermin, wenn
// sie bis zu so vielen Tagen davor oder danach gebucht ist. Die Termine liegen
// drei Monate auseinander; die Fenster überschneiden sich nicht.
const prepaymentWindowDays = 44

// obligationRun hält, was ein Aufruf des Kalenders liest.
type obligationRun struct {
	s     *ObligationService
	ctx   context.Context
	today string
	cfg   *domain.CompanySettings
	state string
	// start ist der erste Tag der Gesellschaft in Buchfink: Beginn des ersten
	// Geschäftsjahres oder, wenn früher, die Gründung.
	start string
	// foundedHere: die Gründung liegt im Zeitraum der Bücher; nur dann
	// stehen die Gründungspflichten im Kalender.
	foundedHere bool
	years       []domain.FiscalYear
	records     map[string]domain.ObligationRecord
	events      []domain.ObligationEvent
	sizes       map[int]*domain.SizeClass
	commits     map[int][]domain.Festschreibung
	ledgers     map[string][]domain.JournalEntry
	ledgerFrom  string
	// dueFrom, dueTo und yearFilter sind die Grenzen der Abfrage. Was sicher
	// außerhalb liegt, wird nicht gebildet: die Größenklasse kostet je
	// Geschäftsjahr einen Abschluss.
	dueFrom, dueTo string
	yearFilter     int
}

// outside meldet, ob alle Fälligkeiten zwischen earliest und latest außerhalb
// der Abfrage liegen.
func (r *obligationRun) outside(year int, earliest, latest time.Time) bool {
	if r.yearFilter != 0 && year != r.yearFilter {
		return true
	}
	return (r.dueFrom != "" && calendar.Format(latest) < r.dueFrom) ||
		(r.dueTo != "" && calendar.Format(earliest) > r.dueTo)
}

func (s *ObligationService) begin(ctx context.Context, today string) (*obligationRun, error) {
	if today == "" {
		today = s.today()
	}
	r := &obligationRun{
		s: s, ctx: ctx, today: today,
		records: map[string]domain.ObligationRecord{},
		sizes:   map[int]*domain.SizeClass{}, commits: map[int][]domain.Festschreibung{},
		ledgers: map[string][]domain.JournalEntry{},
	}
	r.cfg = &domain.CompanySettings{FiscalYearStartMonth: 1}
	if s.settings != nil {
		cfg, err := s.settings.GetCompanySettings(ctx)
		if err != nil {
			return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
		}
		r.cfg = cfg
	}
	r.state = r.cfg.FederalState
	if s.repo != nil {
		recs, err := s.repo.FindRecords(ctx)
		if err != nil {
			return nil, fmt.Errorf("die bestätigten Pflichten konnten nicht gelesen werden: %w", err)
		}
		for _, rec := range recs {
			r.records[rec.Key] = rec
		}
		if r.events, err = s.repo.FindEvents(ctx); err != nil {
			return nil, fmt.Errorf("die Anlässe konnten nicht gelesen werden: %w", err)
		}
	}
	if s.years != nil {
		years, err := s.years.FindAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("die Geschäftsjahre konnten nicht gelesen werden: %w", err)
		}
		sort.Slice(years, func(i, j int) bool { return years[i].Year < years[j].Year })
		r.years = years
	}
	firstYear := ""
	for _, fy := range r.years {
		if fy.StartDate != "" && (firstYear == "" || fy.StartDate < firstYear) {
			firstYear = fy.StartDate
		}
	}
	r.start = firstYear
	if founded := r.cfg.FoundedOn; founded != "" && (r.start == "" || founded < r.start) {
		r.start = founded
	}
	r.foundedHere = r.cfg.FoundedOn != "" && (firstYear == "" || r.cfg.FoundedOn >= addDays(firstYear, -366))
	return r, nil
}

// span sind die Jahre, für die der Kalender Pflichten bildet, damit alle
// Fälligkeiten der Abfrage darin liegen. Eine Erklärung mit Steuerberater
// wird im zweiten Folgejahr fällig; deshalb zwei Jahre vor der unteren Grenze.
func (r *obligationRun) span(q ObligationQuery) (int, int) {
	todayYear := yearOfDate(r.today)
	from, to := todayYear-3, todayYear+1
	if q.FiscalYear != 0 {
		from, to = q.FiscalYear-1, q.FiscalYear+1
	} else {
		if q.From != "" {
			from = yearOfDate(q.From) - 2
		}
		if q.To != "" {
			to = yearOfDate(q.To)
		}
	}
	if startYear := yearOfDate(r.start); startYear > 0 && from < startYear {
		from = startYear
	}
	if to < from {
		to = from
	}
	return from, to
}

// keySpan sind die Jahre, in denen die Pflicht mit diesem Zeitraum entsteht.
// Ein Zeitraum, der mit einem Jahr beginnt, liegt höchstens ein Jahr daneben;
// Anlässe und Gründung reichen vom ersten Jahr der Gesellschaft bis zwei Jahre
// voraus. Bei den Pflichten eines Geschäftsjahres ist das Jahr des Schlüssels
// das Geschäftsjahr.
func (r *obligationRun) keySpan(kind domain.ObligationKind, period string) (int, int) {
	first, last := yearOfDate(r.start), yearOfDate(r.today)+2
	if first == 0 {
		first = yearOfDate(r.today) - 3
	}
	year := yearOfDate(period)
	if year == 0 {
		return first, last
	}
	switch kind {
	case domain.ObligationClosingBookings, domain.ObligationStatementPrep, domain.ObligationAdoption,
		domain.ObligationAppropriation, domain.ObligationDisclosure, domain.ObligationEBilanz,
		domain.ObligationWithholdingTax:
		r.yearFilter = year
	}
	return max(year-1, first), min(year+1, last)
}

// generate bildet die Pflichten der Jahre from bis to.
func (r *obligationRun) generate(from, to int, kinds []domain.ObligationKind) ([]domain.Obligation, error) {
	want := func(k domain.ObligationKind) bool {
		if len(kinds) == 0 {
			return true
		}
		for _, w := range kinds {
			if w == k {
				return true
			}
		}
		return false
	}
	var out []domain.Obligation
	add := func(o domain.Obligation, proof *domain.ObligationEvidence) {
		r.finish(&o, proof)
		out = append(out, o)
	}
	if to-from > 30 {
		return nil, fmt.Errorf("der Zeitraum %d bis %d ist zu lang für den Pflichtenkalender", from, to)
	}
	r.ledgerFrom = fmt.Sprintf("%04d-01-01", from-1)

	generators := r.generators(want, add)
	for first := 0; first < len(generators); {
		next := first + 1
		for next < len(generators) && generators[next].axis == generators[first].axis {
			next++
		}
		// Je Zeitraum laufen alle benachbarten Generatoren derselben Achse.
		for _, period := range r.generatorPeriods(generators[first].axis, from, to) {
			for _, generator := range generators[first:next] {
				if !generator.selected(want) {
					continue
				}
				if err := generator.generate(period); err != nil {
					return nil, err
				}
			}
		}
		first = next
	}
	return out, nil
}

// base baut eine Instanz aus dem Katalog.
func (r *obligationRun) base(
	kind domain.ObligationKind, periodKey, label, from, to string, fiscalYear int,
) domain.Obligation {
	t, _ := obligationType(kind)
	return domain.Obligation{
		Key: string(kind) + "." + periodKey, Kind: kind,
		Title: t.Title + " " + label, Description: t.Description, Basis: t.Basis,
		Channel: t.Channel, Responsibility: t.Responsibility, Evidence: t.Evidence, SkillRef: t.SkillRef,
		PeriodKey: periodKey, PeriodLabel: label, PeriodFrom: from, PeriodTo: to, FiscalYear: fiscalYear,
		Hints: []string{},
	}
}

// setDue setzt das gesetzliche Fristende und, wo § 108 Abs. 3 AO gilt, das
// verschobene.
func (r *obligationRun) setDue(o *domain.Obligation, statutory time.Time) {
	o.StatutoryDate = calendar.Format(statutory)
	o.DueDate = o.StatutoryDate
	t, _ := obligationType(o.Kind)
	if !t.ShiftsToWorkday {
		return
	}
	due, reason := calendar.Shift(statutory, r.state)
	o.DueDate, o.ShiftReason = calendar.Format(due), reason
	if reason != "" {
		o.Hints = append(o.Hints, fmt.Sprintf(
			"Das gesetzliche Fristende am %s ist kein Werktag (%s); die Frist endet am %s (§ 108 Abs. 3 AO).",
			domain.GermanDate(o.StatutoryDate), reason, domain.GermanDate(o.DueDate)))
	}
}

// finish setzt den Stand aus Datennachweis und gespeicherter Entscheidung.
func (r *obligationRun) finish(o *domain.Obligation, proof *domain.ObligationEvidence) {
	rec, has := r.records[o.Key]
	if has && rec.PreparedAt != nil {
		o.Preparation = &domain.ObligationPreparation{
			At: rec.PreparedAt.UTC().Format(time.RFC3339), Note: rec.PreparedNote,
			Channel: rec.PreparedChannel, Actor: rec.PreparedActor,
		}
	}
	manual := has && o.Evidence != domain.EvidenceFromData
	switch {
	case proof != nil:
		o.Status, o.Proof = domain.ObligationDone, proof
	case manual && rec.Status == domain.ObligationDone:
		o.Status = domain.ObligationDone
		o.Proof = &domain.ObligationEvidence{
			Date: rec.EvidenceDate, Kind: rec.EvidenceKind, Reference: rec.EvidenceReference,
			Note: rec.EvidenceNote, Channel: rec.DecidedChannel, Actor: rec.DecidedActor,
		}
	case manual && rec.Status == domain.ObligationWaived:
		o.Status, o.WaivedReason = domain.ObligationWaived, rec.WaivedReason
	case o.DueDate != "" && o.DueDate < r.today:
		o.Status = domain.ObligationOverdue
	case o.Preparation != nil:
		o.Status = domain.ObligationPrepared
	default:
		o.Status = domain.ObligationOpen
	}
}

func (r *obligationRun) vatPeriodType() domain.VatPeriodType {
	if r.s.vat != nil {
		return r.s.vat.PeriodType(r.ctx)
	}
	return vatPeriodTypeOf(r.cfg)
}

func (r *obligationRun) beforeStart(periodTo string) bool {
	return r.start != "" && periodTo < r.start
}

// ---------------------------------------------------------------------------
// Umsatzsteuer

func (r *obligationRun) vatReturns(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	pt := r.vatPeriodType()
	if r.s.vat == nil || (pt != domain.VatPeriodMonth && pt != domain.VatPeriodQuarter) ||
		r.outside(year, calendar.Date(year, time.February, 10), calendar.Date(year+1, time.March, 15)) {
		return nil
	}
	periods, err := r.s.vat.Periods(r.ctx, year)
	if err != nil {
		return fmt.Errorf("die Voranmeldungszeiträume %d konnten nicht gelesen werden: %w", year, err)
	}
	for _, p := range periods {
		if r.beforeStart(p.To) {
			continue
		}
		o := r.base(domain.ObligationVatReturn, p.Key, p.Label, p.From, p.To, p.Year)
		statutory, ok := accounting.VatStatutoryDate(p.VatPeriod, r.cfg.PermanentExtension)
		if !ok {
			continue
		}
		r.setDue(&o, statutory)
		if r.cfg.PermanentExtension {
			o.Hints = append(o.Hints, "Mit Dauerfristverlängerung einen Monat später (§ 46 UStDV).")
		}
		var proof *domain.ObligationEvidence
		if p.Status == domain.VatReturnSubmitted {
			proof = &domain.ObligationEvidence{
				Date: p.SubmittedAt, Kind: domain.EvidenceTransferTicket,
				Reference: r.vatTicket(p.Key), Automatic: true,
			}
		}
		add(o, proof)
	}
	return nil
}

func (r *obligationRun) vatTicket(periodKey string) string {
	if r.s.vatTickets == nil {
		return ""
	}
	returns, err := r.s.vatTickets.FindByPeriod(r.ctx, periodKey)
	if err != nil {
		return ""
	}
	for _, ret := range returns {
		if ret.Status == domain.VatReturnSubmitted && ret.TransferTicket != "" {
			return ret.TransferTicket
		}
	}
	return ""
}

func (r *obligationRun) ecSalesLists(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	if r.s.zm == nil || r.outside(year, calendar.Date(year, time.April, 25), calendar.Date(year+1, time.January, 31)) {
		return nil
	}
	periods, err := r.s.zm.Periods(r.ctx, year)
	if err != nil {
		return fmt.Errorf("die Meldezeiträume %d konnten nicht gelesen werden: %w", year, err)
	}
	for _, p := range periods {
		// Ohne sonstige Leistung ins übrige Gemeinschaftsgebiet keine Meldung
		// (§ 18a Abs. 2 Satz 1 UStG).
		if (p.Total == 0 && p.ReturnID == 0) || r.beforeStart(p.To) {
			continue
		}
		o := r.base(domain.ObligationECSalesList, p.Key, p.Label, p.From, p.To, p.Year)
		statutory, ok := accounting.ZMStatutoryDate(p.VatPeriod)
		if !ok {
			continue
		}
		r.setDue(&o, statutory)
		var proof *domain.ObligationEvidence
		if p.Status == domain.VatReturnSubmitted {
			proof = &domain.ObligationEvidence{
				Date: p.SubmittedAt, Kind: domain.EvidenceTransferTicket,
				Reference: r.zmTicket(p.Key), Automatic: true,
			}
		}
		add(o, proof)
	}
	return nil
}

func (r *obligationRun) zmTicket(periodKey string) string {
	if r.s.zmTickets == nil {
		return ""
	}
	returns, err := r.s.zmTickets.FindByPeriod(r.ctx, periodKey)
	if err != nil {
		return ""
	}
	for _, ret := range returns {
		if ret.Status == domain.VatReturnSubmitted && ret.TransferTicket != "" {
			return ret.TransferTicket
		}
	}
	return ""
}

// specialPrepayment gibt es nur bei monatlicher Voranmeldung mit
// Dauerfristverlängerung (§ 47 Abs. 1 UStDV); dem Vierteljahreszahler wird die
// Verlängerung ohne sie gewährt.
func (r *obligationRun) specialPrepayment(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	if !r.cfg.PermanentExtension || r.vatPeriodType() != domain.VatPeriodMonth {
		return nil
	}
	// Wer im Jahr erst beginnt, meldet sie mit dem ersten Antrag an (§ 48 Abs. 1
	// Satz 4 UStDV); der 10. Februar gilt dann nicht.
	if r.beforeStart(fmt.Sprintf("%d-02-10", year)) {
		return nil
	}
	o := r.base(domain.ObligationSpecialPrepayment, fmt.Sprintf("%d", year), fmt.Sprintf("%d", year),
		fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year), year)
	r.setDue(&o, calendar.Date(year, time.February, 10))
	if r.cfg.SpecialPrepayment > 0 {
		o.Hints = append(o.Hints, fmt.Sprintf("Erfasster Betrag: %s €.", r.cfg.SpecialPrepayment))
	}
	booked, err := r.postings(accountSpecialPrepayment, domain.SideDebit)
	if err != nil {
		return err
	}
	add(o, bookingProof(payments(booked), fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year)))
	return nil
}

// ---------------------------------------------------------------------------
// Vorauszahlungen und Lohnsteuer

func (r *obligationRun) corporatePrepayments(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	return r.prepayments(year, domain.ObligationCorporatePrepay, domain.EventCorporatePrepaymentNotice,
		[]string{accountCorporateTax, accountSolidarityTax}, 10, time.March, add)
}

func (r *obligationRun) tradePrepayments(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	return r.prepayments(year, domain.ObligationTradePrepay, domain.EventTradePrepaymentNotice,
		[]string{domain.AccountGewerbesteuer}, 15, time.February, add)
}

// prepayments bildet die vier Vorauszahlungstermine eines Jahres. Sie stehen
// im Kalender, wenn ein Vorauszahlungsbescheid erfasst ist oder im Vorjahr
// oder im Jahr schon Vorauszahlungen gebucht sind: ihre Höhe steht nur im
// Bescheid. Ein Zahltermin ist ein Fristende und verschiebt sich nach § 108
// Abs. 3 AO (AEAO zu § 108 Nr. 1); bei der Gewerbesteuer über § 1 Abs. 2 Nr. 3
// AO.
func (r *obligationRun) prepayments(
	year int, kind domain.ObligationKind, notice domain.ObligationEventKind, accounts []string,
	day int, firstMonth time.Month, add func(domain.Obligation, *domain.ObligationEvidence),
) error {
	if r.start != "" && year < yearOfDate(r.start) {
		return nil
	}
	var paid []ledgerPosting
	for _, account := range accounts {
		lines, err := r.postings(account, domain.SideDebit)
		if err != nil {
			return err
		}
		paid = append(paid, lines...)
	}
	sort.SliceStable(paid, func(i, j int) bool { return paid[i].date < paid[j].date })
	history := bookingProof(paid, fmt.Sprintf("%d-01-01", year-1), fmt.Sprintf("%d-12-31", year)) != nil
	for q := 0; q < 4; q++ {
		due := calendar.Date(year, firstMonth+time.Month(3*q), day)
		dueISO := calendar.Format(due)
		if r.beforeStart(dueISO) || (!history && !r.noticeBefore(notice, dueISO)) {
			continue
		}
		quarter := (int(due.Month())-1)/3 + 1
		qFrom := calendar.Date(year, time.Month(quarter*3-2), 1)
		o := r.base(kind, fmt.Sprintf("%d-Q%d", year, quarter), domain.GermanDate(dueISO),
			calendar.Format(qFrom), calendar.Format(calendar.EndOfMonth(year, time.Month(quarter*3))), year)
		r.setDue(&o, due)
		add(o, bookingProof(payments(paid), addDays(dueISO, -prepaymentWindowDays), addDays(dueISO, prepaymentWindowDays)))
	}
	return nil
}

func (r *obligationRun) noticeBefore(kind domain.ObligationEventKind, date string) bool {
	for _, ev := range r.events {
		if ev.Kind == kind && ev.Date <= date {
			return true
		}
	}
	return false
}

// payrollTax bildet die Lohnsteuer-Anmeldungen, sobald ein Lohnjournal gebucht
// ist. Das gebuchte Lohnjournal bereitet einen Zeitraum vor; erledigt ist er
// erst mit der Bestätigung der Übermittlung durch den Menschen, weil das
// Journal weder die Anmeldung noch die Abführung belegt (§ 41a Abs. 1 EStG).
// Ohne Lohnjournal steht ein Zeitraum im Kalender, solange er zwischen erstem
// und letztem gebuchtem Lohnjournal liegt oder das letzte keine drei Monate
// zurückliegt.
func (r *obligationRun) payrollTax(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	payroll, err := r.postings(accountPayrollTax, domain.SideCredit)
	if err != nil || len(payroll) == 0 {
		return err
	}
	first, last := payroll[0].date, payroll[len(payroll)-1].date
	active := last >= addDays(r.today, -92)
	tax, err := r.wageTaxByMonth(payroll)
	if err != nil {
		return err
	}
	pt, basis := r.payrollPeriodType(year, tax)
	for _, p := range accounting.VatPeriodsOfYear(year, pt) {
		journal := bookingProof(payroll, p.From, p.To)
		expected := p.To >= first && (p.From <= last || (active && p.From <= r.today))
		if journal == nil && !expected {
			continue
		}
		o := r.base(domain.ObligationPayrollTax, p.Key, p.Label, p.From, p.To, year)
		statutory, ok := accounting.VatStatutoryDate(p, false)
		if !ok {
			continue
		}
		r.setDue(&o, statutory)
		o.Hints = append(o.Hints, basis)
		if journal != nil {
			o.Preparation = &domain.ObligationPreparation{At: journal.Date,
				Note: journal.Reference + ": Lohnjournal gebucht. Anmeldung und Abführung übernimmt meist der " +
					"Lohnanbieter; erledigt mit dem Transferticket oder seiner Bestätigung.",
				Channel: string(actor.ChannelSystem)}
		}
		add(o, nil)
	}
	return nil
}

// payrollTaxes ist die Lohnsteuer je Abrechnungsmonat (JJJJ-MM). fromLedger
// nennt die Monate, deren Betrag aus dem Konto 3730 stammt, das auch
// Kirchensteuer und Solidaritätszuschlag trägt.
type payrollTaxes struct {
	byMonth    map[string]domain.Cents
	fromLedger map[string]bool
}

// source nennt die Herkunft der Beträge der Monate von from bis to.
func (t payrollTaxes) source(from, to string) string {
	for m := range t.fromLedger {
		if m >= from && m <= to {
			return "laut Konto 3730 (mit Kirchensteuer und Solidaritätszuschlag)"
		}
	}
	return "laut Lohnjournal"
}

// wageTaxByMonth liest die anzumeldende Lohnsteuer aus den gebuchten
// Lohnjournalen. Für Monate ohne Lohnjournal, etwa bei Lohnbuchung von Hand,
// nimmt sie die Haben-Buchungen auf 3730.
func (r *obligationRun) wageTaxByMonth(payroll []ledgerPosting) (payrollTaxes, error) {
	out := payrollTaxes{byMonth: map[string]domain.Cents{}, fromLedger: map[string]bool{}}
	runEntries := map[uint]bool{}
	if r.s.payroll != nil {
		runs, err := r.s.payroll.FindAll(r.ctx)
		if err != nil {
			return out, fmt.Errorf("die Lohnjournale konnten nicht gelesen werden: %w", err)
		}
		reversed := map[uint]bool{}
		for _, e := range r.ledgers[accountPayrollTax] {
			if e.Kind == domain.EntryKindReversal && e.ReversalOfID != nil {
				reversed[*e.ReversalOfID] = true
			}
		}
		for _, run := range runs {
			runEntries[run.EntryID] = true
			if !reversed[run.EntryID] {
				out.byMonth[run.Month] += run.WageTax
			}
		}
	}
	for _, p := range payroll {
		if runEntries[p.entryID] {
			continue
		}
		month := p.date[:7]
		out.byMonth[month] += p.amount
		out.fromLedger[month] = true
	}
	return out, nil
}

// payrollPeriodType bestimmt den Anmeldungszeitraum nach § 41a Abs. 2 EStG
// aus der Lohnsteuer des Vorjahres: bis 1.080 € das Kalenderjahr, bis 5.000 €
// das Vierteljahr, sonst der Monat. Die lohnsteuerliche Betriebsstätte
// (§ 41 Abs. 2 EStG) besteht ab dem ersten Monat mit Lohn. Bestand sie nicht
// das ganze Vorjahr, wird dessen Lohnsteuer auf ein Jahr umgerechnet (Satz 3);
// bestand sie im Vorjahr noch nicht, zählt die auf ein Jahr umgerechnete
// Lohnsteuer des ersten vollen Kalendermonats (Satz 4).
func (r *obligationRun) payrollPeriodType(year int, tax payrollTaxes) (domain.VatPeriodType, string) {
	prevFrom, prevTo := fmt.Sprintf("%d-01-01", year-1), fmt.Sprintf("%d-12-31", year-1)
	months := make([]string, 0, len(tax.byMonth))
	for m := range tax.byMonth {
		months = append(months, m)
	}
	sort.Strings(months)
	if len(months) == 0 {
		return domain.VatPeriodMonth, "Anmeldungszeitraum Kalendermonat: es ist keine Lohnsteuer gebucht (§ 41a Abs. 2 Satz 1 EStG)."
	}
	opening := months[0] + "-01"
	if r.start > opening && r.start[:7] == months[0] {
		opening = r.start
	}

	var sum domain.Cents
	var how string
	switch {
	case opening <= prevFrom:
		for m, amount := range tax.byMonth {
			if m >= prevFrom[:7] && m <= prevTo[:7] {
				sum += amount
			}
		}
		how = fmt.Sprintf("Lohnsteuer im Vorjahr %s € %s (§ 41a Abs. 2 Satz 2 EStG)",
			sum, tax.source(prevFrom[:7], prevTo[:7]))
	case opening <= prevTo:
		var booked domain.Cents
		for m, amount := range tax.byMonth {
			if m <= prevTo[:7] {
				booked += amount
			}
		}
		existed := 13 - int(monthOf(opening))
		sum = booked * 12 / domain.Cents(existed)
		how = fmt.Sprintf("Lohnsteuer im Vorjahr %s € in %d Monaten %s, auf ein Jahr umgerechnet %s € (§ 41a Abs. 2 Satz 3 EStG)",
			booked, existed, tax.source(prevFrom[:7], prevTo[:7]), sum)
	default:
		full := opening[:7]
		if opening[8:] != "01" {
			full = addMonthsISO(opening, 1)[:7]
		}
		amount, ok := tax.byMonth[full]
		if !ok {
			return domain.VatPeriodMonth, fmt.Sprintf("Anmeldungszeitraum vorläufig Kalendermonat: maßgebend ist die Lohnsteuer "+
				"des ersten vollen Kalendermonats %s/%s, für den noch kein Lohnjournal gebucht ist (§ 41a Abs. 2 Satz 4 EStG).",
				full[5:7], full[:4])
		}
		sum = amount * 12
		how = fmt.Sprintf("Lohnsteuer im ersten vollen Kalendermonat %s/%s %s € %s, auf ein Jahr umgerechnet %s € (§ 41a Abs. 2 Satz 4 EStG)",
			full[5:7], full[:4], amount, tax.source(full, full), sum)
	}
	switch {
	case sum <= payrollYearlyLimit:
		return domain.VatPeriodYear, "Anmeldungszeitraum Kalenderjahr: " + how + ", nicht mehr als 1.080 €."
	case sum <= payrollQuarterlyLimit:
		return domain.VatPeriodQuarter, "Anmeldungszeitraum Kalendervierteljahr: " + how + ", nicht mehr als 5.000 €."
	}
	return domain.VatPeriodMonth, "Anmeldungszeitraum Kalendermonat: " + how + "."
}

func monthOf(iso string) time.Month {
	t, err := calendar.Parse(iso)
	if err != nil {
		return time.January
	}
	return t.Month()
}

// ---------------------------------------------------------------------------
// Jahreserklärungen

// annualReturnDeadline ist das gesetzliche Ende der Erklärungsfrist für ein
// Kalenderjahr: sieben Monate nach seinem Ablauf (§ 149 Abs. 2 Satz 1 AO), mit
// Steuerberater der letzte Tag des Februar des zweiten Folgejahres (§ 149
// Abs. 3 AO). Für 2019 bis 2024 gelten die Übergangsfristen des Art. 97 § 36
// Abs. 1 und 3 EGAO.
func annualReturnDeadline(year int, advisor bool) time.Time {
	if advisor {
		transitional := map[int]time.Time{
			2019: calendar.Date(2021, time.August, 31),
			2020: calendar.Date(2022, time.August, 31),
			2021: calendar.Date(2023, time.August, 31),
			2022: calendar.Date(2024, time.July, 31),
			2023: calendar.Date(2025, time.May, 31),
			2024: calendar.Date(2026, time.April, 30),
		}
		if d, ok := transitional[year]; ok {
			return d
		}
		return calendar.EndOfMonth(year+2, time.February)
	}
	months := 7
	switch year {
	case 2020, 2021:
		months = 10
	case 2022:
		months = 9
	case 2023:
		months = 8
	}
	return calendar.EndOfMonthsAfter(calendar.Date(year, time.December, 31), months)
}

func (r *obligationRun) annualReturn(
	kind domain.ObligationKind, declaration string,
) func(int, func(domain.Obligation, *domain.ObligationEvidence)) error {
	return func(year int, add func(domain.Obligation, *domain.ObligationEvidence)) error {
		if r.start != "" && year < yearOfDate(r.start) {
			return nil
		}
		if kind == domain.ObligationAnnualVatReturn && !r.vatPeriodType().Valid() {
			return nil
		}
		o := r.base(kind, fmt.Sprintf("%d", year), fmt.Sprintf("%d", year),
			fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-31", year), year)
		advisor := r.cfg.TaxAdvisor.Prepares(declaration, year)
		r.setDue(&o, annualReturnDeadline(year, advisor))
		r.returnHints(&o, year, advisor)
		if advisor {
			o.Responsibility = domain.ResponsibilityTaxAdvisor
		}
		proof, err := r.declarationProof(declaration, year)
		if err != nil {
			return err
		}
		add(o, proof)
		return nil
	}
}

// declarationProof belegt eine Jahreserklärung mit der jüngsten in der App
// erfassten Übermittlung und ihrem Transferticket.
func (r *obligationRun) declarationProof(kind string, year int) (*domain.ObligationEvidence, error) {
	if r.s.returns == nil {
		return nil, nil
	}
	records, err := r.s.returns.FindRecords(r.ctx, kind, year)
	if err != nil {
		return nil, fmt.Errorf("die Stände der Erklärung %d konnten nicht gelesen werden: %w", year, err)
	}
	for i := len(records) - 1; i >= 0; i-- {
		if rec := records[i]; rec.Status == string(declaration.StatusSubmitted) {
			return &domain.ObligationEvidence{
				Date: rec.SubmittedAt, Kind: domain.EvidenceTransferTicket,
				Reference: rec.TransferTicket, Automatic: true,
			}, nil
		}
	}
	return nil, nil
}

func (r *obligationRun) returnHints(o *domain.Obligation, year int, advisor bool) {
	if advisor {
		o.Hints = append(o.Hints, "Ein Steuerberater erstellt die Erklärung. Das Finanzamt kann sie vorab "+
			"anfordern; dann gilt die Frist der Anordnung, mindestens vier Monate nach ihrer Bekanntgabe "+
			"(§ 149 Abs. 4 AO).")
	}
	if year >= 2025 {
		limit := calendar.EndOfMonthsAfter(calendar.Date(year, time.December, 31), 14)
		o.Hints = append(o.Hints, fmt.Sprintf("Ohne Abgabe binnen 14 Monaten nach Ablauf des Kalenderjahres, "+
			"bis zum %s, setzt das Finanzamt einen Verspätungszuschlag fest (§ 152 Abs. 2 Nr. 1 AO).",
			domain.GermanDate(calendar.Format(limit))))
	}
}

// ---------------------------------------------------------------------------
// Geschäftsjahr: Festschreibung und Jahresabschluss

// fiscalYearsIn liefert die Geschäftsjahre from bis to; fehlt eines, wird es
// aus dem Beginnmonat abgeleitet.
func (r *obligationRun) fiscalYearsIn(from, to int) []domain.FiscalYear {
	known := map[int]domain.FiscalYear{}
	for _, fy := range r.years {
		known[fy.Year] = fy
	}
	horizon := addDays(r.today, 366)
	var out []domain.FiscalYear
	for year := from - 1; year <= to; year++ {
		fy, ok := known[year]
		if !ok {
			start := domain.FiscalYearFirstDay(year, r.cfg.FiscalYearStartMonth)
			t, err := calendar.Parse(start)
			if err != nil {
				continue
			}
			end := calendar.Format(t.AddDate(1, 0, -1))
			if r.beforeStart(end) {
				continue
			}
			if r.start != "" && start < r.start {
				start = r.start
			}
			fy = domain.FiscalYear{Year: year, StartDate: start, EndDate: end, Status: domain.FiscalYearOpen}
		}
		if fy.StartDate == "" || fy.EndDate == "" || fy.StartDate > horizon {
			continue
		}
		out = append(out, fy)
	}
	return out
}

func fiscalYearLabel(fy domain.FiscalYear) string {
	if yearOfDate(fy.StartDate) == yearOfDate(fy.EndDate) {
		return fmt.Sprintf("%d", fy.Year)
	}
	return fmt.Sprintf("%d/%d", yearOfDate(fy.StartDate), yearOfDate(fy.EndDate))
}

func (r *obligationRun) commitmentsOf(year int) ([]domain.Festschreibung, error) {
	if list, ok := r.commits[year]; ok {
		return list, nil
	}
	if r.s.commits == nil {
		return nil, nil
	}
	list, err := r.s.commits.FindByFiscalYear(r.ctx, year)
	if err != nil {
		return nil, fmt.Errorf("die Festschreibungen %d konnten nicht gelesen werden: %w", year, err)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CutoffDate < list[j].CutoffDate })
	r.commits[year] = list
	return list, nil
}

func (r *obligationRun) monthCommits(fy domain.FiscalYear, add func(domain.Obligation, *domain.ObligationEvidence)) error {
	start, errStart := calendar.Parse(fy.StartDate)
	end, errEnd := calendar.Parse(fy.EndDate)
	if errStart != nil || errEnd != nil || r.outside(fy.Year, calendar.EndOfMonthsAfter(start, 1),
		calendar.EndOfMonthsAfter(end, 2).AddDate(0, 0, r.cfg.CommitGraceDays)) {
		return nil
	}
	commitments, err := r.commitmentsOf(fy.Year)
	if err != nil {
		return err
	}
	for calendarYear := yearOfDate(fy.StartDate); calendarYear <= yearOfDate(fy.EndDate); calendarYear++ {
		for _, p := range accounting.VatPeriodsOfYear(calendarYear, domain.VatPeriodMonth) {
			if p.To < fy.StartDate || p.From > fy.EndDate || r.beforeStart(p.To) {
				continue
			}
			o := r.base(domain.ObligationMonthCommit, p.Key, p.Label, p.From, p.To, fy.Year)
			due, err := calendar.Parse(CommitDueDate(p, r.cfg))
			if err != nil {
				continue
			}
			r.setDue(&o, due)
			var proof *domain.ObligationEvidence
			for _, c := range commitments {
				if c.CutoffDate >= p.To {
					proof = &domain.ObligationEvidence{
						Date: c.CreatedAt.Local().Format(domain.DateLayout), Kind: domain.EvidenceCommitment,
						Reference: c.PeriodLabel, Automatic: true,
					}
					break
				}
			}
			add(o, proof)
		}
	}
	return nil
}

func (r *obligationRun) sizeOf(year int) *domain.SizeClass {
	if size, ok := r.sizes[year]; ok {
		return size
	}
	var size *domain.SizeClass
	if r.s.sizes != nil {
		if got, err := r.s.sizes.SizeClassFor(r.ctx, year); err == nil {
			size = got
		}
	}
	r.sizes[year] = size
	return size
}

const ebilanzChannelHint = "Mein ELSTER führt kein Formular für die E-Bilanz; übermittelt wird sie über einen " +
	"E-Bilanz-Dienst wie ebilanzplus.de oder ebilanzonline.de oder vom Steuerberater."

// annualStatement bildet Abschlussbuchungen, Aufstellung, Feststellung,
// Ergebnisverwendung, Offenlegung, E-Bilanz und die Kapitalertragsteuer auf
// eine Ausschüttung. Die handelsrechtlichen Fristen verschieben sich nicht.
func (r *obligationRun) annualStatement(
	fy domain.FiscalYear, want func(domain.ObligationKind) bool,
	add func(domain.Obligation, *domain.ObligationEvidence),
) error {
	end, err := calendar.Parse(fy.EndDate)
	if err != nil {
		return nil
	}
	label := fiscalYearLabel(fy)
	key := fmt.Sprintf("%d", fy.Year)
	if r.yearFilter != 0 && fy.Year != r.yearFilter {
		return nil
	}
	commercial := (want(domain.ObligationClosingBookings) || want(domain.ObligationStatementPrep) ||
		want(domain.ObligationAdoption) || want(domain.ObligationAppropriation) || want(domain.ObligationDisclosure)) &&
		!r.outside(fy.Year, calendar.EndOfMonthsAfter(end, 3), calendar.AddMonths(end, 12))

	prepMonths, adoptMonths := 3, 8
	sizeHint := ""
	var size *domain.SizeClass
	if commercial {
		size = r.sizeOf(fy.Year)
		switch {
		case size == nil:
			sizeHint = "Die Größenklasse ließ sich nicht ermitteln; genannt sind die kürzeren Fristen der " +
				"mittelgroßen Gesellschaft."
		case size.Class == domain.SizeMicro || size.Class == domain.SizeSmall:
			prepMonths, adoptMonths = 6, 11
			if size.Obligations.PreparationMonths > 0 {
				prepMonths = size.Obligations.PreparationMonths
			}
		}
	}
	prepDue := calendar.EndOfMonthsAfter(end, prepMonths)

	if commercial && want(domain.ObligationClosingBookings) {
		o := r.base(domain.ObligationClosingBookings, key, label, fy.StartDate, fy.EndDate, fy.Year)
		o.Title = fmt.Sprintf("Abschlussbuchungen zum Bilanzstichtag %s", domain.GermanDate(fy.EndDate))
		r.setDue(&o, prepDue)
		commitments, err := r.commitmentsOf(fy.Year)
		if err != nil {
			return err
		}
		var proof *domain.ObligationEvidence
		for _, c := range commitments {
			if c.PeriodType == domain.PeriodTypeYear {
				proof = &domain.ObligationEvidence{
					Date: c.CreatedAt.Local().Format(domain.DateLayout), Kind: domain.EvidenceCommitment,
					Reference: c.PeriodLabel, Automatic: true,
				}
			}
		}
		if proof == nil && fy.PreparedOn != "" {
			proof = &domain.ObligationEvidence{Date: fy.PreparedOn, Kind: domain.EvidenceStatus, Automatic: true,
				Note: "Abschluss aufgestellt"}
		}
		add(o, proof)
	}
	if commercial && want(domain.ObligationStatementPrep) {
		o := r.base(domain.ObligationStatementPrep, key, label, fy.StartDate, fy.EndDate, fy.Year)
		r.setDue(&o, prepDue)
		if prepMonths == 6 {
			o.Hints = append(o.Hints, "Die sechs Monate gelten nur, wenn die spätere Aufstellung einem "+
				"ordnungsgemäßen Geschäftsgang entspricht; sonst endet die Frist nach drei Monaten "+
				"(§ 264 Abs. 1 Satz 3 und 4 HGB).")
		}
		if sizeHint != "" {
			o.Hints = append(o.Hints, sizeHint)
		}
		var proof *domain.ObligationEvidence
		if fy.PreparedOn != "" {
			proof = &domain.ObligationEvidence{Date: fy.PreparedOn, Kind: domain.EvidenceStatus, Automatic: true,
				Note: "Abschluss aufgestellt"}
		}
		add(o, proof)
	}
	adoptDue := calendar.EndOfMonthsAfter(end, adoptMonths)
	if commercial && want(domain.ObligationAdoption) {
		o := r.base(domain.ObligationAdoption, key, label, fy.StartDate, fy.EndDate, fy.Year)
		r.setDue(&o, adoptDue)
		if sizeHint != "" {
			o.Hints = append(o.Hints, sizeHint)
		}
		var proof *domain.ObligationEvidence
		if fy.AdoptedOn != "" {
			proof = &domain.ObligationEvidence{Date: fy.AdoptedOn, Kind: domain.EvidenceResolution,
				Reference: fy.AdoptionNote, Automatic: true}
		}
		add(o, proof)
	}
	var decision *domain.Appropriation
	if r.s.decisions != nil && (want(domain.ObligationAppropriation) || want(domain.ObligationWithholdingTax)) {
		if decision, err = r.s.decisions.Appropriation(r.ctx, fy.Year); err != nil {
			return fmt.Errorf("die Ergebnisverwendung %d konnte nicht gelesen werden: %w", fy.Year, err)
		}
	}
	if commercial && want(domain.ObligationAppropriation) {
		o := r.base(domain.ObligationAppropriation, key, label, fy.StartDate, fy.EndDate, fy.Year)
		r.setDue(&o, adoptDue)
		var proof *domain.ObligationEvidence
		if decision != nil {
			proof = &domain.ObligationEvidence{Date: decision.DecisionDate, Kind: domain.EvidenceResolution,
				Reference: truncate(decision.Text, 200), Automatic: true}
		}
		add(o, proof)
	}
	if commercial && want(domain.ObligationDisclosure) {
		o := r.base(domain.ObligationDisclosure, key, label, fy.StartDate, fy.EndDate, fy.Year)
		r.setDue(&o, calendar.AddMonths(end, 12))
		if size != nil && size.Obligations.DisclosureScope != "" {
			o.Hints = append(o.Hints, fmt.Sprintf("Umfang: %s (%s).",
				size.Obligations.DisclosureScope, size.Obligations.DisclosureScopeReference))
		}
		var proof *domain.ObligationEvidence
		if fy.DisclosedOn != "" {
			proof = &domain.ObligationEvidence{Date: fy.DisclosedOn, Kind: domain.EvidenceOrderNumber,
				Reference: fy.DisclosureNote, Automatic: true}
		}
		add(o, proof)
	}
	if want(domain.ObligationEBilanz) {
		// Die E-Bilanz gehört zur Körperschaftsteuererklärung des
		// Kalenderjahres, in dem das Wirtschaftsjahr endet (§ 7 Abs. 4 Satz 2
		// KStG), und wird mit ihr fällig.
		vz := yearOfDate(fy.EndDate)
		o := r.base(domain.ObligationEBilanz, key, label, fy.StartDate, fy.EndDate, fy.Year)
		r.setDue(&o, annualReturnDeadline(vz, r.cfg.TaxAdvisor.Prepares(domain.DeclarationCorporateTax, vz)))
		if r.cfg.TaxAdvisor.Prepares(domain.DeclarationEBilanz, vz) {
			o.Responsibility = domain.ResponsibilityTaxAdvisor
		}
		o.Hints = append(o.Hints, ebilanzChannelHint)
		add(o, nil)
	}
	if want(domain.ObligationWithholdingTax) && decision != nil && decision.Distribution > 0 {
		r.withholdingTax(fy, decision, add)
	}
	return nil
}

// withholdingTax: die Kapitalertragsteuer auf eine Ausschüttung ist im
// Zeitpunkt des Zuflusses anzumelden und abzuführen (§ 44 Abs. 1 Satz 5,
// § 45a Abs. 1 EStG). Ohne Auszahlungstag im Beschluss fließt der Ertrag am
// Tag nach der Beschlussfassung zu (§ 44 Abs. 2 Satz 2 EStG).
func (r *obligationRun) withholdingTax(
	fy domain.FiscalYear, decision *domain.Appropriation, add func(domain.Obligation, *domain.ObligationEvidence),
) {
	decided, err := calendar.Parse(decision.DecisionDate)
	if err != nil {
		return
	}
	o := r.base(domain.ObligationWithholdingTax, fmt.Sprintf("%d", fy.Year),
		"aus dem Ergebnis "+fiscalYearLabel(fy), decision.DecisionDate, decision.DecisionDate, fy.Year)
	r.setDue(&o, decided.AddDate(0, 0, 1))
	o.Hints = append(o.Hints,
		fmt.Sprintf("Laut Beschluss vom %s: Ausschüttung %s €, Kapitalertragsteuer %s €, Solidaritätszuschlag %s €.",
			domain.GermanDate(decision.DecisionDate), decision.Distribution, decision.WithholdingTax,
			decision.SolidarityOnWithholding),
		"Bestimmt der Beschluss einen Tag der Auszahlung, fließt der Ertrag an diesem Tag zu (§ 44 Abs. 2 Satz 1 EStG); "+
			"Buchfink rechnet mit dem Tag nach dem Beschluss.",
		"Die zu einem Zeitpunkt abzuführende Kapitalertragsteuer wird auf volle Euro abgerundet (§ 44 Abs. 1 Satz 6 EStG).")
	add(o, nil)
}

// ---------------------------------------------------------------------------
// Anlässe und Gründung

func (r *obligationRun) eventObligations(
	want func(domain.ObligationKind) bool, add func(domain.Obligation, *domain.ObligationEvidence),
) {
	for _, ev := range r.events {
		due, err := calendar.Parse(ev.Date)
		if err != nil {
			continue
		}
		key := fmt.Sprintf("ereignis-%d", ev.ID)
		label := fmt.Sprintf("(Veränderung vom %s)", domain.GermanDate(ev.Date))
		var kinds []domain.ObligationKind
		switch ev.Kind {
		case domain.EventShareholderChange:
			kinds = []domain.ObligationKind{domain.ObligationShareholderList, domain.ObligationTransparency}
		case domain.EventBeneficialOwnerChange:
			kinds = []domain.ObligationKind{domain.ObligationTransparency}
		}
		for _, kind := range kinds {
			if !want(kind) {
				continue
			}
			o := r.base(kind, key, label, ev.Date, ev.Date, due.Year())
			o.Immediate = true
			r.setDue(&o, due)
			if ev.Note != "" {
				o.Hints = append(o.Hints, "Anlass: "+ev.Note)
			}
			if kind == domain.ObligationTransparency {
				o.Hints = append(o.Hints, "Nötig, wenn sich die wirtschaftlich Berechtigten (mehr als 25 % der "+
					"Kapitalanteile oder Stimmrechte, § 3 Abs. 2 GwG) oder ihre Angaben ändern; sonst als "+
					"entfallen kennzeichnen.")
			}
			add(o, nil)
		}
	}
}

func (r *obligationRun) foundingObligations(
	want func(domain.ObligationKind) bool, add func(domain.Obligation, *domain.ObligationEvidence),
) {
	if !r.foundedHere {
		return
	}
	founded, err := calendar.Parse(r.cfg.FoundedOn)
	if err != nil {
		return
	}
	label := fmt.Sprintf("(Gründung am %s)", domain.GermanDate(r.cfg.FoundedOn))
	if want(domain.ObligationTaxRegistration) {
		o := r.base(domain.ObligationTaxRegistration, "gruendung", label, r.cfg.FoundedOn, r.cfg.FoundedOn, founded.Year())
		r.setDue(&o, calendar.AddMonths(founded, 1))
		o.Hints = append(o.Hints, "Buchfink rechnet den Monat ab der Beurkundung.")
		add(o, nil)
	}
	if want(domain.ObligationTradeRegistration) {
		o := r.base(domain.ObligationTradeRegistration, "gruendung", label, r.cfg.FoundedOn, r.cfg.FoundedOn, founded.Year())
		r.setDue(&o, founded)
		o.Hints = append(o.Hints, "Fällig mit dem Beginn des Betriebs; Buchfink setzt den Tag der Gründung an. "+
			"Bei rein freiberuflichem Gegenstand ist zu klären, ob ein Gewerbe vorliegt; sonst als entfallen kennzeichnen.")
		add(o, nil)
	}
	if want(domain.ObligationOpeningBalance) || want(domain.ObligationOpeningEBilanz) {
		r.openingBalance(founded, want, add)
	}
	if want(domain.ObligationTransparency) && r.cfg.RegisteredOn != "" {
		registered, err := calendar.Parse(r.cfg.RegisteredOn)
		if err != nil {
			return
		}
		o := r.base(domain.ObligationTransparency, "gruendung",
			fmt.Sprintf("(Eintragung am %s)", domain.GermanDate(r.cfg.RegisteredOn)),
			r.cfg.RegisteredOn, r.cfg.RegisteredOn, registered.Year())
		o.Immediate = true
		r.setDue(&o, registered)
		add(o, nil)
	}
}

// openingBalance bildet Aufstellung und E-Bilanz der Eröffnungsbilanz auf den
// Tag der Beurkundung. Beide gehören zum Geschäftsjahr, in das die Gründung
// fällt.
//
// Die Eröffnungsbilanz ist zu Beginn des Handelsgewerbes aufzustellen
// (§ 242 Abs. 1 Satz 1 HGB), nach Satz 2 i. V. m. § 243 Abs. 3 HGB in der
// einem ordnungsmäßigen Geschäftsgang entsprechenden Zeit. Eine feste Frist
// gibt es nicht; Buchfink setzt in Anlehnung an § 264 Abs. 1 Satz 3 HGB drei
// Monate an. Als handelsrechtliche Frist verschiebt sie sich nicht.
//
// Ihre E-Bilanz verlangt § 5b Abs. 1 Satz 4 EStG ohne eigene Frist. Buchfink
// setzt die Frist der E-Bilanz des ersten Geschäftsjahres an, wie in
// annualStatement aus der Körperschaftsteuererklärung des Kalenderjahres, in
// dem es endet.
func (r *obligationRun) openingBalance(
	founded time.Time, want func(domain.ObligationKind) bool,
	add func(domain.Obligation, *domain.ObligationEvidence),
) {
	fy, ok := r.foundingYear()
	if !ok {
		return
	}
	label := "zum " + domain.GermanDate(r.cfg.FoundedOn)
	if want(domain.ObligationOpeningBalance) {
		o := r.base(domain.ObligationOpeningBalance, "gruendung", label, r.cfg.FoundedOn, r.cfg.FoundedOn, fy.Year)
		r.setDue(&o, calendar.AddMonths(founded, 3))
		o.Hints = append(o.Hints,
			"Eine feste Frist nennt das Gesetz nicht: aufzustellen ist in der einem ordnungsmäßigen Geschäftsgang "+
				"entsprechenden Zeit (§ 242 Abs. 1 Satz 2, § 243 Abs. 3 HGB). Buchfink setzt in Anlehnung an "+
				"§ 264 Abs. 1 Satz 3 HGB drei Monate nach der Beurkundung an.",
			"Die Geschäftsführer unterzeichnen sie mit Datum (§ 245 HGB). Aufzubewahren ist sie zehn Jahre "+
				"(§ 147 Abs. 1 Nr. 1, Abs. 3 AO; § 257 HGB).")
		add(o, nil)
	}
	if want(domain.ObligationOpeningEBilanz) {
		vz := yearOfDate(fy.EndDate)
		o := r.base(domain.ObligationOpeningEBilanz, "gruendung", label, r.cfg.FoundedOn, r.cfg.FoundedOn, fy.Year)
		r.setDue(&o, annualReturnDeadline(vz, r.cfg.TaxAdvisor.Prepares(domain.DeclarationCorporateTax, vz)))
		if r.cfg.TaxAdvisor.Prepares(domain.DeclarationEBilanz, vz) {
			o.Responsibility = domain.ResponsibilityTaxAdvisor
		}
		o.Hints = append(o.Hints,
			fmt.Sprintf("Eine eigene Frist nennt das Gesetz nicht (§ 5b Abs. 1 Satz 4 EStG). Buchfink setzt die Frist "+
				"der E-Bilanz für das Geschäftsjahr %s an; das ist die späteste vertretbare Annahme.", fiscalYearLabel(fy)),
			ebilanzChannelHint)
		add(o, nil)
	}
}

// foundingYear ist das Geschäftsjahr, in das die Gründung fällt.
func (r *obligationRun) foundingYear() (domain.FiscalYear, bool) {
	year := domain.GetFiscalYearForDate(r.cfg.FoundedOn, r.cfg.FiscalYearStartMonth)
	for _, fy := range r.fiscalYearsIn(year, year) {
		if fy.Year == year {
			return fy, true
		}
	}
	return domain.FiscalYear{}, false
}

// ---------------------------------------------------------------------------
// Journal

// ledgerPosting ist eine Zeile auf einem Konto, aufsteigend nach Datum.
type ledgerPosting struct {
	date    string
	number  string
	amount  domain.Cents
	entryID uint
	// paid: die Buchung beruht auf einem Bankumsatz oder stammt nicht von
	// einem Agenten. Nur sie belegt eine Zahlung.
	paid bool
}

// postings liest die Buchungen einer Seite eines Kontos ab ledgerFrom.
// Stornierte Buchungen, Stornos, Eröffnungs- und Abschlussbuchungen zählen
// nicht: auf 7600 ff. steht zum Jahresende auch die Steuerrückstellung.
func (r *obligationRun) postings(account string, side domain.Side) ([]ledgerPosting, error) {
	if r.s.journal == nil {
		return nil, nil
	}
	entries, ok := r.ledgers[account]
	if !ok {
		var err error
		entries, err = r.s.journal.FindByAccountRange(r.ctx, account, r.ledgerFrom, "")
		if err != nil {
			return nil, fmt.Errorf("das Konto %s konnte nicht gelesen werden: %w", account, err)
		}
		r.ledgers[account] = entries
	}
	reversed := map[uint]bool{}
	for _, e := range entries {
		if e.Kind == domain.EntryKindReversal && e.ReversalOfID != nil {
			reversed[*e.ReversalOfID] = true
		}
	}
	var out []ledgerPosting
	for _, e := range entries {
		if e.Kind == domain.EntryKindReversal || reversed[e.ID] ||
			e.Source == domain.EntrySourceClosing || e.Source == domain.EntrySourceOpening {
			continue
		}
		for _, line := range e.Lines {
			if line.Account == account && line.Side == side && line.Amount > 0 {
				out = append(out, ledgerPosting{date: e.BookingDate, number: e.EntryNumber, amount: line.Amount, entryID: e.ID,
					paid: e.BankTxID != nil || !actor.Channel(e.Channel).IsAgent()})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].date < out[j].date })
	return out, nil
}

// payments sind die Buchungen, die eine Zahlung belegen. Eine Handbuchung
// eines Agenten („laut Bescheid gezahlt") ist keine: sie setzte eine
// versäumte Vorauszahlung auf erledigt, ohne dass Geld floss.
func payments(postings []ledgerPosting) []ledgerPosting {
	out := make([]ledgerPosting, 0, len(postings))
	for _, p := range postings {
		if p.paid {
			out = append(out, p)
		}
	}
	return out
}

// bookingProof ist der Nachweis aus der ersten Buchung im Zeitraum.
func bookingProof(postings []ledgerPosting, from, to string) *domain.ObligationEvidence {
	for _, p := range postings {
		if p.date >= from && p.date <= to {
			return &domain.ObligationEvidence{
				Date: p.date, Kind: domain.EvidenceBooking, Automatic: true,
				Reference: strings.TrimSpace("Buchung " + p.number),
			}
		}
	}
	return nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
