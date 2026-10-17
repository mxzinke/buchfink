package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// ObligationService ist der Pflichtenkalender: jede Pflicht der Gesellschaft
// mit Fälligkeit, Stand und Nachweis.
//
// Die Instanzen entstehen bei jedem Aufruf aus Stammdaten und Buchführung.
// Erledigt ist, was die Daten belegen (Transferticket, Festschreibung,
// Abschlussstand, gebuchte Zahlung); was Buchfink nicht sieht, bestätigt der
// Mensch in der App mit Nachweis. Ein Agent darf eine Pflicht nur als
// vorbereitet melden.
type ObligationService struct {
	settings domain.SettingsRepository
	repo     domain.ObligationRepository
	audit    domain.AuditRepository

	vat        ObligationVatSource
	vatTickets ObligationVatTickets
	zm         ObligationZMSource
	zmTickets  ObligationZMTickets
	commits    domain.FestschreibungRepository
	years      ObligationYearSource
	sizes      ObligationSizeSource
	decisions  ObligationAppropriationSource
	journal    ObligationJournalSource
	returns    ObligationDeclarationSource

	now func() time.Time
}

// ObligationVatSource liefert die Voranmeldungszeiträume eines Kalenderjahres.
type ObligationVatSource interface {
	Periods(ctx context.Context, year int) ([]VatPeriodStatus, error)
	PeriodType(ctx context.Context) domain.VatPeriodType
}

// ObligationVatTickets liefert die Voranmeldungen eines Zeitraums.
type ObligationVatTickets interface {
	FindByPeriod(ctx context.Context, periodKey string) ([]domain.VatReturn, error)
}

// ObligationZMSource liefert die Meldezeiträume der Zusammenfassenden Meldung.
type ObligationZMSource interface {
	Periods(ctx context.Context, year int) ([]ZMPeriodStatus, error)
}

// ObligationZMTickets liefert die Zusammenfassenden Meldungen eines Zeitraums.
type ObligationZMTickets interface {
	FindByPeriod(ctx context.Context, periodKey string) ([]domain.ZMReturn, error)
}

// ObligationYearSource liefert die angelegten Geschäftsjahre.
type ObligationYearSource interface {
	FindAll(ctx context.Context) ([]domain.FiscalYear, error)
}

// ObligationSizeSource beurteilt die Größenklasse eines Geschäftsjahres.
type ObligationSizeSource interface {
	SizeClassFor(ctx context.Context, year int) (*domain.SizeClass, error)
}

// ObligationAppropriationSource liefert den Ergebnisverwendungsbeschluss.
type ObligationAppropriationSource interface {
	Appropriation(ctx context.Context, year int) (*domain.Appropriation, error)
}

// ObligationJournalSource liest ein Kontoblatt über Geschäftsjahre hinweg.
type ObligationJournalSource interface {
	FindByAccountRange(ctx context.Context, account, from, to string) ([]domain.JournalEntry, error)
}

// ObligationDeclarationSource liefert die festgehaltenen Stände einer
// Jahressteuererklärung, älteste zuerst.
type ObligationDeclarationSource interface {
	FindRecords(ctx context.Context, kind string, year int) ([]domain.DeclarationRecord, error)
}

// NewObligationService baut den Pflichtenkalender über Stammdaten, Ablage und
// Protokoll; die Quellen werden einzeln angehängt.
func NewObligationService(
	settings domain.SettingsRepository, repo domain.ObligationRepository, audit domain.AuditRepository,
) *ObligationService {
	return &ObligationService{settings: settings, repo: repo, audit: audit, now: time.Now}
}

// SetVatSources hängt Voranmeldungen und ihre Transfertickets an.
func (s *ObligationService) SetVatSources(src ObligationVatSource, tickets ObligationVatTickets) {
	s.vat, s.vatTickets = src, tickets
}

// SetZMSources hängt die Zusammenfassenden Meldungen an.
func (s *ObligationService) SetZMSources(src ObligationZMSource, tickets ObligationZMTickets) {
	s.zm, s.zmTickets = src, tickets
}

// SetCommitments hängt die Festschreibungen an.
func (s *ObligationService) SetCommitments(repo domain.FestschreibungRepository) { s.commits = repo }

// SetFiscalYearSource hängt die Geschäftsjahre an.
func (s *ObligationService) SetFiscalYearSource(src ObligationYearSource) { s.years = src }

// SetSizeSource hängt die Größenklasse an; ohne sie gelten die Fristen der
// großen Gesellschaft.
func (s *ObligationService) SetSizeSource(src ObligationSizeSource) { s.sizes = src }

// SetAppropriationSource hängt die Ergebnisverwendung an.
func (s *ObligationService) SetAppropriationSource(src ObligationAppropriationSource) {
	s.decisions = src
}

// SetJournalSource hängt das Journal an: Vorauszahlungen und Lohnjournal.
func (s *ObligationService) SetJournalSource(src ObligationJournalSource) { s.journal = src }

// SetDeclarationSource hängt die Stände der Jahressteuererklärungen an.
func (s *ObligationService) SetDeclarationSource(src ObligationDeclarationSource) { s.returns = src }

// SetClock ersetzt die Uhr, gegen die überfällig gerechnet wird.
func (s *ObligationService) SetClock(now func() time.Time) { s.now = now }

// Catalog liefert die Pflichtarten.
func (s *ObligationService) Catalog() []domain.ObligationType { return ObligationTypes() }

// ObligationQuery wählt Pflichten aus.
type ObligationQuery struct {
	// From und To begrenzen die Fälligkeit (JJJJ-MM-TT, einschließlich);
	// leer heißt offen.
	From string
	To   string
	// FiscalYear wählt die Pflichten eines Jahres; 0 heißt alle.
	FiscalYear int
	Statuses   []domain.ObligationStatus
	Kinds      []domain.ObligationKind
	// Today ist der Stichtag für „überfällig"; leer heißt heute.
	Today string
}

// List liefert die Pflichten nach Fälligkeit, die früheste zuerst.
func (s *ObligationService) List(ctx context.Context, q ObligationQuery) ([]domain.Obligation, error) {
	for _, iso := range []struct{ label, value string }{{"von", q.From}, {"bis", q.To}, {"Stichtag", q.Today}} {
		if iso.value != "" {
			if err := domain.CheckDate("Fälligkeit "+iso.label, iso.value); err != nil {
				return nil, err
			}
		}
	}
	for _, st := range q.Statuses {
		if !st.Valid() {
			return nil, fmt.Errorf("unbekannter Stand %q (erwartet open, prepared, done, overdue, waived)", st)
		}
	}
	for _, k := range q.Kinds {
		if _, ok := obligationType(k); !ok {
			return nil, fmt.Errorf("unbekannte Pflichtart %q", k)
		}
	}
	run, err := s.begin(ctx, q.Today)
	if err != nil {
		return nil, err
	}
	from, to := run.span(q)
	run.dueFrom, run.dueTo, run.yearFilter = q.From, q.To, q.FiscalYear
	all, err := run.generate(from, to, q.Kinds)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Obligation, 0, len(all))
	for _, o := range all {
		if q.FiscalYear != 0 && o.FiscalYear != q.FiscalYear {
			continue
		}
		if (q.From != "" && o.DueDate < q.From) || (q.To != "" && o.DueDate > q.To) {
			continue
		}
		if len(q.Statuses) > 0 && !containsStatus(q.Statuses, o.Status) {
			continue
		}
		out = append(out, o)
	}
	sortObligations(out)
	return out, nil
}

// Next liefert die nächsten Pflichten, die noch etwas verlangen: überfällige
// zuerst, dann nach Fälligkeit. limit ≤ 0 heißt alle.
func (s *ObligationService) Next(ctx context.Context, today string, limit int) ([]domain.Obligation, error) {
	list, err := s.List(ctx, ObligationQuery{
		Today:    today,
		Statuses: []domain.ObligationStatus{domain.ObligationOverdue, domain.ObligationOpen, domain.ObligationPrepared},
	})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return list, nil
}

// Get liefert eine Pflicht mit Stand und Nachweis.
func (s *ObligationService) Get(ctx context.Context, key, today string) (*domain.Obligation, error) {
	kind, period, ok := strings.Cut(key, ".")
	if !ok {
		return nil, fmt.Errorf("unbekannte Pflicht %q (erwartet <pflichtart>.<zeitraum>)", key)
	}
	if _, known := obligationType(domain.ObligationKind(kind)); !known {
		return nil, fmt.Errorf("unbekannte Pflichtart %q", kind)
	}
	run, err := s.begin(ctx, today)
	if err != nil {
		return nil, err
	}
	from, to := run.keySpan(domain.ObligationKind(kind), period)
	all, err := run.generate(from, to, []domain.ObligationKind{domain.ObligationKind(kind)})
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].Key == key {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("die Pflicht %q gibt es für diesen Mandanten nicht", key)
}

// ConfirmInput ist der Nachweis einer Bestätigung.
type ConfirmInput struct {
	Date      string              `json:"date"`
	Kind      domain.EvidenceKind `json:"kind"`
	Reference string              `json:"reference"`
	Note      string              `json:"note"`
}

// Confirm bestätigt eine Pflicht als erledigt. Der Mensch bestätigt in der
// App; die Bestätigung steht mit Nachweis im Protokoll.
func (s *ObligationService) Confirm(ctx context.Context, key string, in ConfirmInput) (*domain.Obligation, error) {
	if err := requireHuman(ctx, "Die Bestätigung einer Pflicht"); err != nil {
		return nil, err
	}
	o, err := s.Get(ctx, key, "")
	if err != nil {
		return nil, err
	}
	if err := manualDecisionAllowed(o, "bestätigt"); err != nil {
		return nil, err
	}
	if o.Status == domain.ObligationDone {
		return nil, fmt.Errorf("%s ist bereits erledigt", o.Title)
	}
	in.Reference, in.Note = strings.TrimSpace(in.Reference), strings.TrimSpace(in.Note)
	if in.Kind == "" {
		in.Kind = domain.EvidenceConfirmation
	}
	if !in.Kind.Valid() {
		return nil, fmt.Errorf("unbekannte Nachweisart %q", in.Kind)
	}
	if err := domain.CheckDate("Datum des Nachweises", in.Date); err != nil {
		return nil, err
	}
	if in.Date > s.today() {
		return nil, fmt.Errorf("der Nachweis vom %s liegt in der Zukunft", domain.GermanDate(in.Date))
	}
	switch in.Kind {
	case domain.EvidenceTransferTicket, domain.EvidenceFileNumber, domain.EvidenceOrderNumber:
		if in.Reference == "" {
			return nil, fmt.Errorf("zum Nachweis %s gehört die Nummer (Transferticket, Aktenzeichen oder Auftragsnummer)",
				evidenceLabel(in.Kind))
		}
	}
	rec, err := s.record(ctx, o)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	rec.Status = domain.ObligationDone
	rec.EvidenceDate, rec.EvidenceKind = in.Date, in.Kind
	rec.EvidenceReference, rec.EvidenceNote = in.Reference, in.Note
	rec.WaivedReason = ""
	rec.DecidedChannel, rec.DecidedActor, rec.DecidedAt = string(actor.RecordedChannel(ctx)), actor.Actor(), &now
	if err := s.save(ctx, rec); err != nil {
		return nil, err
	}
	details := fmt.Sprintf("%s (%s) als erledigt bestätigt: %s am %s", o.Title, o.PeriodLabel,
		evidenceLabel(in.Kind), domain.GermanDate(in.Date))
	if in.Reference != "" {
		details += ", " + in.Reference
	}
	if err := logAudit(ctx, s.audit, domain.AuditActionUpdate, "OBLIGATION", key, details); err != nil {
		return nil, err
	}
	return s.Get(ctx, key, "")
}

// MarkPrepared meldet eine Pflicht als vorbereitet. Das darf auch ein
// verbundener Agent; erledigt meldet er nie.
func (s *ObligationService) MarkPrepared(ctx context.Context, key, note string) (*domain.Obligation, error) {
	if channel, ok := actor.ChannelOf(ctx); ok && channel == actor.ChannelSystem {
		return nil, fmt.Errorf("vorbereitet meldet ein Mensch in der App oder ein verbundener Agent, kein Hintergrundlauf")
	}
	o, err := s.Get(ctx, key, "")
	if err != nil {
		return nil, err
	}
	if o.Status.Closed() {
		return nil, fmt.Errorf("%s steht schon auf „%s“", o.Title, o.Status.Label())
	}
	rec, err := s.record(ctx, o)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	rec.PreparedNote = strings.TrimSpace(note)
	rec.PreparedChannel, rec.PreparedActor, rec.PreparedAt = string(actor.RecordedChannel(ctx)), actor.Actor(), &now
	if err := s.save(ctx, rec); err != nil {
		return nil, err
	}
	details := fmt.Sprintf("%s (%s) als vorbereitet gemeldet über %s", o.Title, o.PeriodLabel, actor.RecordedChannel(ctx))
	if rec.PreparedNote != "" {
		details += ": " + rec.PreparedNote
	}
	if err := logAudit(ctx, s.audit, domain.AuditActionUpdate, "OBLIGATION", key, details); err != nil {
		return nil, err
	}
	return s.Get(ctx, key, "")
}

// Waive kennzeichnet eine Pflicht als entfallen, mit Grund.
func (s *ObligationService) Waive(ctx context.Context, key, reason string) (*domain.Obligation, error) {
	if err := requireHuman(ctx, "Das Kennzeichnen einer Pflicht als entfallen"); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("zum Entfallen einer Pflicht gehört der Grund")
	}
	o, err := s.Get(ctx, key, "")
	if err != nil {
		return nil, err
	}
	if err := manualDecisionAllowed(o, "als entfallen gekennzeichnet"); err != nil {
		return nil, err
	}
	if o.Status.Closed() {
		return nil, fmt.Errorf("%s steht schon auf „%s“", o.Title, o.Status.Label())
	}
	rec, err := s.record(ctx, o)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	rec.Status, rec.WaivedReason = domain.ObligationWaived, reason
	rec.EvidenceDate, rec.EvidenceKind, rec.EvidenceReference, rec.EvidenceNote = "", "", "", ""
	rec.DecidedChannel, rec.DecidedActor, rec.DecidedAt = string(actor.RecordedChannel(ctx)), actor.Actor(), &now
	if err := s.save(ctx, rec); err != nil {
		return nil, err
	}
	if err := logAudit(ctx, s.audit, domain.AuditActionUpdate, "OBLIGATION", key,
		fmt.Sprintf("%s (%s) entfällt: %s", o.Title, o.PeriodLabel, reason)); err != nil {
		return nil, err
	}
	return s.Get(ctx, key, "")
}

// Reopen nimmt eine Bestätigung oder ein Entfallen zurück, mit Grund.
func (s *ObligationService) Reopen(ctx context.Context, key, reason string) (*domain.Obligation, error) {
	if err := requireHuman(ctx, "Das Zurücknehmen einer Bestätigung"); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("zum Zurücknehmen gehört der Grund")
	}
	rec, err := s.repoFind(ctx, key)
	if err != nil {
		return nil, err
	}
	if rec == nil || !rec.Status.Closed() {
		return nil, fmt.Errorf("die Pflicht %q ist weder bestätigt noch als entfallen gekennzeichnet", key)
	}
	before := rec.Status
	rec.Status, rec.WaivedReason = "", ""
	rec.EvidenceDate, rec.EvidenceKind, rec.EvidenceReference, rec.EvidenceNote = "", "", "", ""
	rec.DecidedChannel, rec.DecidedActor, rec.DecidedAt = "", "", nil
	if err := s.save(ctx, rec); err != nil {
		return nil, err
	}
	if err := logAudit(ctx, s.audit, domain.AuditActionUpdate, "OBLIGATION", key,
		fmt.Sprintf("Stand „%s“ zurückgenommen: %s", before.Label(), reason)); err != nil {
		return nil, err
	}
	return s.Get(ctx, key, "")
}

// RecordEvent erfasst einen Anlass, der Pflichten auslöst, etwa einen
// Gesellschafterwechsel. Ein Anlass fügt nur Pflichten hinzu; auch ein Agent
// darf ihn melden.
func (s *ObligationService) RecordEvent(
	ctx context.Context, kind domain.ObligationEventKind, date, note string,
) (*domain.ObligationEvent, error) {
	if !kind.Valid() {
		return nil, fmt.Errorf("unbekannter Anlass %q", kind)
	}
	if err := domain.CheckDate("Tag des Anlasses", date); err != nil {
		return nil, err
	}
	if s.repo == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	ev := &domain.ObligationEvent{
		Kind: kind, Date: date, Note: strings.TrimSpace(note),
		Channel: string(actor.RecordedChannel(ctx)), Actor: actor.Actor(),
	}
	if err := s.repo.CreateEvent(ctx, ev); err != nil {
		return nil, fmt.Errorf("der Anlass ließ sich nicht speichern: %w", err)
	}
	if err := logAudit(ctx, s.audit, domain.AuditActionCreate, "OBLIGATION_EVENT", fmt.Sprintf("%d", ev.ID),
		fmt.Sprintf("Anlass %s am %s erfasst", kind, domain.GermanDate(date))); err != nil {
		return nil, err
	}
	return ev, nil
}

// Deadlines liefert die Pflichten eines Geschäftsjahres in der Form der
// Fristenseite.
func (s *ObligationService) Deadlines(ctx context.Context, year int) ([]domain.Deadline, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	list, err := s.List(ctx, ObligationQuery{FiscalYear: year})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Deadline, 0, len(list))
	for _, o := range list {
		d := domain.Deadline{
			Key: o.Key, Title: o.Title, DueDate: o.DueDate, Period: o.PeriodLabel,
			Reference: o.Basis, Description: o.Description, FiscalYear: o.FiscalYear,
			Status: string(o.Status), Confirmable: o.Evidence != domain.EvidenceFromData && !o.Status.Closed(),
		}
		if o.Status.Closed() {
			d.IsDone = true
			if o.Proof != nil {
				d.DoneOn = o.Proof.Date
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// ---------------------------------------------------------------------------

func (s *ObligationService) today() string { return s.now().Format(domain.DateLayout) }

func (s *ObligationService) repoFind(ctx context.Context, key string) (*domain.ObligationRecord, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	rec, err := s.repo.FindRecord(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("der Stand der Pflicht ließ sich nicht lesen: %w", err)
	}
	return rec, nil
}

func (s *ObligationService) record(ctx context.Context, o *domain.Obligation) (*domain.ObligationRecord, error) {
	rec, err := s.repoFind(ctx, o.Key)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		rec = &domain.ObligationRecord{Key: o.Key, Kind: o.Kind}
	}
	return rec, nil
}

func (s *ObligationService) save(ctx context.Context, rec *domain.ObligationRecord) error {
	if err := s.repo.SaveRecord(ctx, rec); err != nil {
		return fmt.Errorf("der Stand der Pflicht ließ sich nicht speichern: %w", err)
	}
	return nil
}

// manualDecisionAllowed weist Bestätigung und Entfallen bei Pflichten ab,
// deren Stand allein aus den Daten kommt.
func manualDecisionAllowed(o *domain.Obligation, what string) error {
	if o.Evidence != domain.EvidenceFromData {
		return nil
	}
	return fmt.Errorf("%s (%s) ergibt sich aus den Daten und wird nicht von Hand %s: %s",
		o.Title, o.PeriodLabel, what, o.Description)
}

func evidenceLabel(k domain.EvidenceKind) string {
	switch k {
	case domain.EvidenceTransferTicket:
		return "Transferticket"
	case domain.EvidenceFileNumber:
		return "Aktenzeichen"
	case domain.EvidenceOrderNumber:
		return "Auftragsnummer"
	case domain.EvidenceBooking:
		return "Buchung"
	case domain.EvidenceCommitment:
		return "Festschreibung"
	case domain.EvidenceResolution:
		return "Beschluss"
	case domain.EvidenceStatus:
		return "Abschlussstand"
	}
	return "Bestätigung"
}

func containsStatus(list []domain.ObligationStatus, st domain.ObligationStatus) bool {
	for _, s := range list {
		if s == st {
			return true
		}
	}
	return false
}

func sortObligations(list []domain.Obligation) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].DueDate != list[j].DueDate {
			return list[i].DueDate < list[j].DueDate
		}
		return list[i].Key < list[j].Key
	})
}
