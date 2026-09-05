package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// ClosingService führt das Geschäftsjahr als Entität und trägt die Salden ins
// Folgejahr vor.
//
// Der Grund für beides ist derselbe Satz: § 252 Abs. 1 Nr. 1 HGB verlangt, dass
// die Eröffnungsbilanz des Geschäftsjahres mit der Schlussbilanz des Vorjahres
// übereinstimmt. Bisher entstanden alle Salden ausschließlich aus den Buchungen
// des jeweiligen Jahres — ein Bankkonto begann jedes Jahr bei null, und die
// Bilanzidentität war nicht bloß ungeprüft, sie war verletzt.
//
// Geschrieben wird wie überall über den JournalService: ein Saldenvortrag ist
// eine Buchung wie jede andere, mit Nummer, Hash und Festschreibungsprüfung.
type ClosingService struct {
	fiscalYearRepo     domain.FiscalYearRepository
	journalRepo        domain.JournalRepository
	accountRepo        domain.AccountRepository
	contactRepo        domain.ContactRepository
	allocationRepo     domain.PaymentAllocationRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository
	foundationRepo     domain.FoundationRepository
	auditRepo          domain.AuditRepository
	journalSvc         *JournalService
	// accruals und notes sind die Kopplungen dieser Welle: die Auflösung der
	// Rechnungsabgrenzung hängt am Saldenvortrag, die Anhangtexte hängen an der
	// Jahresanlage. Beide sind optional — ohne sie arbeitet der Abschluss wie
	// zuvor.
	accruals   AccrualCarrier
	notes      NotesCopier
	fiscalYear int
}

// NewClosingService wires the Jahresabschluss.
func NewClosingService(
	fiscalYearRepo domain.FiscalYearRepository,
	journalRepo domain.JournalRepository,
	accountRepo domain.AccountRepository,
	contactRepo domain.ContactRepository,
	allocationRepo domain.PaymentAllocationRepository,
	settingsRepo domain.SettingsRepository,
	festschreibungRepo domain.FestschreibungRepository,
	auditRepo domain.AuditRepository,
	journalSvc *JournalService,
	fiscalYear int,
) *ClosingService {
	return &ClosingService{
		fiscalYearRepo:     fiscalYearRepo,
		journalRepo:        journalRepo,
		accountRepo:        accountRepo,
		contactRepo:        contactRepo,
		allocationRepo:     allocationRepo,
		settingsRepo:       settingsRepo,
		festschreibungRepo: festschreibungRepo,
		auditRepo:          auditRepo,
		journalSvc:         journalSvc,
		fiscalYear:         fiscalYear,
	}
}

// SetFoundationRepo koppelt die Gründung an. Sie entscheidet über das
// Rumpfgeschäftsjahr: eine Gesellschaft, die im März beurkundet wurde, hat kein
// Geschäftsjahr, das im Januar begonnen hätte.
func (s *ClosingService) SetFoundationRepo(r domain.FoundationRepository) { s.foundationRepo = r }

// AccrualCarrier ist der Ausschnitt der Rechnungsabgrenzung, den der
// Saldenvortrag braucht.
//
// Die Auflösung eines Abgrenzungspostens gehört in das Folgejahr und wird dort
// am ersten Tag gebucht. Sie an den Vortrag zu hängen ist kein Zufall: der
// Vortrag ist der eine Vorgang, der das neue Jahr eröffnet, und ohne diesen
// Anstoß bliebe die Auflösung bis zum nächsten Jahresabschluss liegen — also
// bis zu dem Zeitpunkt, an dem sie längst gebucht sein müsste.
type AccrualCarrier interface {
	PendingReleases(ctx context.Context, toYear int) ([]AccrualReleaseDue, error)
	ReleaseInto(ctx context.Context, toYear int) ([]domain.JournalEntry, error)
}

// SetAccrualCarrier koppelt die Rechnungsabgrenzung an den Saldenvortrag.
func (s *ClosingService) SetAccrualCarrier(c AccrualCarrier) { s.accruals = c }

// NotesCopier übernimmt die Anhangtexte des Vorjahres in ein neu angelegtes
// Geschäftsjahr.
type NotesCopier interface {
	CopyNotesInto(ctx context.Context, toYear int) (int, error)
}

// SetNotesCopier koppelt die Anhangtexte an die Jahresanlage.
func (s *ClosingService) SetNotesCopier(c NotesCopier) { s.notes = c }

// SetFiscalYear updates the active fiscal year.
func (s *ClosingService) SetFiscalYear(year int) { s.fiscalYear = year }

// -------------------------------------------------------------
// Geschäftsjahre
// -------------------------------------------------------------

// FiscalYears liefert alle erfassten Geschäftsjahre.
func (s *ClosingService) FiscalYears(ctx context.Context) ([]domain.FiscalYear, error) {
	return s.fiscalYearRepo.FindAll(ctx)
}

// EnsureFiscalYears legt für jedes Jahr, das im Journal vorkommt, die Entität
// an, falls sie fehlt.
//
// Das ist die Migration bestehender Datenbanken. Bis hierher war das
// Geschäftsjahr nur eine Zahl an der Buchung; die Zeiträume dazu lassen sich
// nachträglich nur aus zwei Angaben gewinnen — dem Beginn des Geschäftsjahres
// aus den Unternehmensdaten und, für das Gründungsjahr, dem Beurkundungsdatum.
// Vorhandene Einträge bleiben unberührt: ein von Hand berichtigter Zeitraum darf
// beim nächsten Start nicht wieder überschrieben werden.
func (s *ClosingService) EnsureFiscalYears(ctx context.Context) error {
	years := map[int]bool{}
	if s.fiscalYear > 0 {
		years[s.fiscalYear] = true
	}
	journalYears, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return fmt.Errorf("die Geschäftsjahre des Journals konnten nicht gelesen werden: %w", err)
	}
	for _, y := range journalYears {
		if y > 0 {
			years[y] = true
		}
	}

	ordered := make([]int, 0, len(years))
	for y := range years {
		ordered = append(ordered, y)
	}
	sort.Ints(ordered)

	for _, y := range ordered {
		existing, err := s.fiscalYearRepo.FindByYear(ctx, y)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		fy := s.derive(ctx, y)
		if err := fy.Validate(); err != nil {
			return err
		}
		if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
			return fmt.Errorf("das Geschäftsjahr %d konnte nicht angelegt werden: %w", y, err)
		}
		s.audit(ctx, domain.AuditActionCreate, y, fmt.Sprintf(
			"Geschäftsjahr %d angelegt (%s bis %s%s)", y, fy.StartDate, fy.EndDate, shortSuffix(fy)))
	}
	return nil
}

// CreateFiscalYear legt das Folgejahr an: es beginnt am Tag nach dem Ende des
// Vorjahres und dauert zwölf Monate.
func (s *ClosingService) CreateFiscalYear(ctx context.Context, year int) (*domain.FiscalYear, error) {
	if year <= 0 {
		return nil, fmt.Errorf("das Geschäftsjahr braucht eine Jahreszahl")
	}
	existing, err := s.fiscalYearRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	fy := s.derive(ctx, year)
	// Das Vorjahr entscheidet über den Beginn: nach einem Rumpfgeschäftsjahr
	// oder einer Umstellung ist der abgeleitete Beginn nicht der richtige, wohl
	// aber der Tag nach dem Ende des Vorjahres — sonst entstünde eine Lücke oder
	// eine Überschneidung, in der Buchungen zu keinem Jahr gehören.
	if prev, err := s.fiscalYearRepo.FindByYear(ctx, year-1); err == nil && prev != nil {
		start := nextDay(prev.EndDate)
		fy = domain.NewFiscalYear(year, start, lastDayOfTwelveMonths(start))
	}
	if err := fy.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return nil, fmt.Errorf("das Geschäftsjahr %d konnte nicht angelegt werden: %w", year, err)
	}
	s.audit(ctx, domain.AuditActionCreate, year, fmt.Sprintf(
		"Geschäftsjahr %d angelegt (%s bis %s%s)", year, fy.StartDate, fy.EndDate, shortSuffix(fy)))
	// Die Anhangtexte des Vorjahres werden als Vorlage übernommen. Die
	// Bilanzierungs- und Bewertungsmethoden ändern sich selten, und ein leerer
	// Anhang führt in der Praxis dazu, dass die Angaben schlicht fehlen.
	// Scheitert das, bleibt das Jahr trotzdem angelegt: eine fehlende Vorlage ist
	// kein Grund, kein Geschäftsjahr zu haben.
	if s.notes != nil {
		if _, err := s.notes.CopyNotesInto(ctx, year); err != nil {
			s.audit(ctx, domain.AuditActionUpdate, year, fmt.Sprintf(
				"Die Anhangtexte des Vorjahres konnten nicht übernommen werden: %v", err))
		}
	}
	return fy, nil
}

// YearOf liefert das Geschäftsjahr und legt es an, falls es noch fehlt.
func (s *ClosingService) YearOf(ctx context.Context, year int) (*domain.FiscalYear, error) {
	fy, err := s.fiscalYearRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	if fy != nil {
		return fy, nil
	}
	derived := s.derive(ctx, year)
	if err := derived.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, derived); err != nil {
		return nil, fmt.Errorf("das Geschäftsjahr %d konnte nicht angelegt werden: %w", year, err)
	}
	// Auch das nebenbei angelegte Geschäftsjahr steht im Protokoll: wer später
	// wissen will, woher ein Zeitraum stammt, findet ihn sonst nur dann, wenn er
	// über die Jahresanlage entstanden ist (Entscheidung 8).
	s.audit(ctx, domain.AuditActionCreate, year, fmt.Sprintf(
		"Geschäftsjahr %d angelegt (%s bis %s%s)", year, derived.StartDate, derived.EndDate, shortSuffix(derived)))
	return derived, nil
}

// yearOrDerived liefert das Geschäftsjahr, ohne es anzulegen.
//
// Die Vorschau darf nichts erzeugen: wer den Abschlussstand von 2026 ansieht,
// fragt damit auch nach dem Vortrag ins Jahr 2027 — und fände es danach in der
// Jahresauswahl wieder, ohne es angelegt zu haben.
func (s *ClosingService) yearOrDerived(ctx context.Context, year int) (*domain.FiscalYear, error) {
	fy, err := s.fiscalYearRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	if fy != nil {
		return fy, nil
	}
	derived := s.derive(ctx, year)
	if err := derived.Validate(); err != nil {
		return nil, err
	}
	return derived, nil
}

// derive baut den Zeitraum eines Geschäftsjahres aus den Unternehmensdaten.
func (s *ClosingService) derive(ctx context.Context, year int) *domain.FiscalYear {
	startMonth := s.fiscalYearStartMonth(ctx)
	start := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	startDate := start.Format("2006-01-02")
	endDate := lastDayOfTwelveMonths(startDate)

	// Das Gründungsjahr beginnt mit der Beurkundung, nicht am Kalenderanfang:
	// vorher gab es die Gesellschaft nicht, und ein Jahr, das vor ihrer
	// Entstehung beginnt, hätte einen Zeitraum ohne Bücher.
	if s.foundationRepo != nil {
		if f, err := s.foundationRepo.Get(ctx); err == nil && f != nil && len(f.NotarizedOn) == 10 {
			if f.NotarizedOn > startDate && f.NotarizedOn <= endDate {
				startDate = f.NotarizedOn
			}
		}
	}
	return domain.NewFiscalYear(year, startDate, endDate)
}

func (s *ClosingService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FiscalYearStartMonth <= 0 || cfg.FiscalYearStartMonth > 12 {
		return 1
	}
	return cfg.FiscalYearStartMonth
}

// -------------------------------------------------------------
// Abschlussstand
// -------------------------------------------------------------

// ClosingState ist alles, was die Abschlussansicht eines Jahres braucht.
type ClosingState struct {
	Year       int               `json:"year"`
	FiscalYear domain.FiscalYear `json:"fiscalYear"`
	// NetIncome ist das Jahresergebnis: Erträge minus Aufwendungen der
	// GuV-Konten. Eine abgeleitete Größe, keine gebuchte — bei SKR04 werden die
	// Erfolgskonten nicht über ein Abschlusskonto geschlossen.
	NetIncome domain.Cents `json:"netIncome"`

	// HasYearCommitment sagt, ob für das Jahr eine Jahres-Festschreibung
	// vorliegt. Ohne sie lässt sich der Abschluss nicht feststellen.
	HasYearCommitment bool   `json:"hasYearCommitment"`
	CommittedUntil    string `json:"committedUntil,omitempty"`

	NextYear int `json:"nextYear"`
	// CarriedForward sagt, ob ins Folgejahr bereits vorgetragen wurde,
	// CarryForwardCurrent, ob dieser Vortrag noch zu den Salden dieses Jahres
	// passt. Auseinanderfallen können sie, sobald hier nachträglich gebucht wird.
	CarriedForward      bool       `json:"carriedForward"`
	CarryForwardCurrent bool       `json:"carryForwardCurrent"`
	CarriedForwardAt    *time.Time `json:"carriedForwardAt,omitempty"`

	// NextStatus ist der Schritt, der als nächster ansteht; leer, wenn das Jahr
	// offengelegt ist.
	NextStatus domain.FiscalYearStatus `json:"nextStatus,omitempty"`
	// CanAdopt sagt, ob der nächste Schritt jetzt gegangen werden kann.
	CanAdopt bool   `json:"canAdopt"`
	Blocker  string `json:"blocker,omitempty"`
}

// ClosingStateFor stellt den Abschlussstand eines Jahres zusammen.
func (s *ClosingService) ClosingStateFor(ctx context.Context, year int) (*ClosingState, error) {
	fy, err := s.YearOf(ctx, year)
	if err != nil {
		return nil, err
	}

	state := &ClosingState{Year: year, FiscalYear: *fy, NextYear: year + 1}

	turnovers, err := s.journalRepo.AccountTurnovers(ctx, year)
	if err != nil {
		return nil, err
	}
	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}
	state.NetIncome = netIncomeOf(turnovers, chart)

	state.HasYearCommitment, state.CommittedUntil, err = s.yearCommitment(ctx, year)
	if err != nil {
		return nil, err
	}

	if next, err := s.fiscalYearRepo.FindByYear(ctx, year+1); err == nil && next != nil {
		state.CarriedForwardAt = next.CarriedForwardAt
	}
	if preview, err := s.CarryForwardState(ctx, year+1); err == nil {
		state.CarriedForward = preview.AlreadyCarried
		state.CarryForwardCurrent = preview.AlreadyCarried && !preview.NeedsCorrection
	}

	state.NextStatus, state.CanAdopt, state.Blocker = s.nextStep(fy, state.HasYearCommitment)
	return state, nil
}

// nextStep benennt den nächsten Schritt und, falls er noch nicht gegangen werden
// kann, was ihm entgegensteht.
func (s *ClosingService) nextStep(fy *domain.FiscalYear, hasYearCommitment bool) (domain.FiscalYearStatus, bool, string) {
	switch fy.Status {
	case domain.FiscalYearOpen:
		return domain.FiscalYearPrepared, true, ""
	case domain.FiscalYearPrepared:
		if !hasYearCommitment {
			return domain.FiscalYearAdopted, false, fmt.Sprintf(
				"Für %d fehlt die Jahres-Festschreibung. Ein Abschluss, dessen Buchungen noch änderbar "+
					"sind, kann nicht festgestellt werden.", fy.Year)
		}
		return domain.FiscalYearAdopted, true, ""
	case domain.FiscalYearAdopted:
		return domain.FiscalYearDisclosed, true, ""
	}
	return "", false, ""
}

// yearCommitment sucht die Festschreibung des ganzen Jahres.
func (s *ClosingService) yearCommitment(ctx context.Context, year int) (bool, string, error) {
	if s.festschreibungRepo == nil {
		return false, "", nil
	}
	records, err := s.festschreibungRepo.FindByFiscalYear(ctx, year)
	if err != nil {
		return false, "", fmt.Errorf("die Festschreibungen des Jahres %d konnten nicht gelesen werden: %w", year, err)
	}
	var latest string
	found := false
	for _, rec := range records {
		if rec.CutoffDate > latest {
			latest = rec.CutoffDate
		}
		if rec.PeriodType == "year" {
			found = true
		}
	}
	return found, latest, nil
}

// SetFiscalYearStatus schaltet den Abschluss einen Schritt weiter.
//
// Immer nur einen: die Feststellung setzt die Aufstellung voraus (§ 264 Abs. 1,
// § 42a Abs. 1 GmbHG), die Offenlegung die Feststellung (§ 325 Abs. 1 HGB). Ein
// Sprung über einen Schritt hinweg wäre kein schnellerer Weg, sondern eine
// Behauptung über einen Vorgang, den es nicht gab.
func (s *ClosingService) SetFiscalYearStatus(
	ctx context.Context, year int, status domain.FiscalYearStatus, date, note string,
) (*domain.FiscalYear, error) {
	fy, err := s.YearOf(ctx, year)
	if err != nil {
		return nil, err
	}
	if !status.Valid() || status == domain.FiscalYearOpen {
		return nil, fmt.Errorf(
			"unbekannter Abschlussstand %q; ein Jahr wird über die Rücksetzung wieder geöffnet", status)
	}
	if status == fy.Status {
		return nil, fmt.Errorf("das Geschäftsjahr %d steht bereits auf %q", year, status.Label())
	}
	if !isNextStatus(fy.Status, status) {
		return nil, fmt.Errorf(
			"das Geschäftsjahr %d steht auf %q; der nächste Schritt ist %q und nicht %q",
			year, fy.Status.Label(), nextStatus(fy.Status).Label(), status.Label())
	}
	if len(date) != 10 {
		return nil, fmt.Errorf("das Datum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("%q ist kein Datum (erwartet JJJJ-MM-TT)", date)
	}
	if date < fy.EndDate {
		return nil, fmt.Errorf(
			"der %s liegt vor dem Ende des Geschäftsjahres am %s; ein Abschluss lässt sich nicht "+
				"aufstellen, bevor das Jahr vorbei ist", date, fy.EndDate)
	}

	switch status {
	case domain.FiscalYearPrepared:
		fy.PreparedOn = date
	case domain.FiscalYearAdopted:
		hasYear, _, err := s.yearCommitment(ctx, year)
		if err != nil {
			return nil, err
		}
		// Die Feststellung ohne Festschreibung wäre ein Beschluss über Zahlen,
		// die sich danach noch ändern lassen.
		if !hasYear {
			return nil, fmt.Errorf(
				"für %d liegt keine Jahres-Festschreibung vor. Schreibe das Geschäftsjahr zuerst fest – "+
					"erst dann steht fest, worüber die Gesellschafter beschließen (§ 42a Abs. 2 GmbHG)", year)
		}
		if date < fy.PreparedOn {
			return nil, fmt.Errorf(
				"die Feststellung am %s läge vor der Aufstellung am %s", date, fy.PreparedOn)
		}
		fy.AdoptedOn = date
		fy.AdoptionNote = strings.TrimSpace(note)
	case domain.FiscalYearDisclosed:
		if date < fy.AdoptedOn {
			return nil, fmt.Errorf(
				"die Offenlegung am %s läge vor der Feststellung am %s", date, fy.AdoptedOn)
		}
		fy.DisclosedOn = date
	}
	fy.Status = status

	if err := fy.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return nil, fmt.Errorf("der Abschlussstand konnte nicht gespeichert werden: %w", err)
	}

	details := fmt.Sprintf("Geschäftsjahr %d: %s am %s", year, status.Label(), date)
	if n := strings.TrimSpace(note); n != "" {
		details += " (" + n + ")"
	}
	s.audit(ctx, domain.AuditActionUpdate, year, details)
	return fy, nil
}

// ReopenFiscalYear nimmt die Feststellung zurück.
//
// Der Grund ist Pflicht und landet im Protokoll: die Rücksetzung ist der einzige
// Weg, in ein festgestelltes Jahr wieder zu buchen, und wer sie geht, muss sagen
// warum. Die Festschreibungen bleiben, wie sie sind — sie sind der Nachweis, dass
// die Buchungen bis dahin unverändert vorlagen, und dieser Nachweis wird durch
// eine spätere Entscheidung nicht falsch.
func (s *ClosingService) ReopenFiscalYear(ctx context.Context, year int, reason string) (*domain.FiscalYear, error) {
	fy, err := s.YearOf(ctx, year)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("für die Rücksetzung des Jahresabschlusses ist ein Grund anzugeben")
	}
	if fy.Status == domain.FiscalYearOpen || fy.Status == domain.FiscalYearPrepared {
		return nil, fmt.Errorf(
			"das Geschäftsjahr %d ist nicht festgestellt (Stand: %s); es gibt nichts zurückzusetzen",
			year, fy.Status.Label())
	}

	previous := fy.Status
	fy.Status = domain.FiscalYearPrepared
	fy.AdoptedOn = ""
	fy.AdoptionNote = ""
	fy.DisclosedOn = ""

	if err := fy.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return nil, fmt.Errorf("die Rücksetzung konnte nicht gespeichert werden: %w", err)
	}

	s.audit(ctx, domain.AuditActionUpdate, year, fmt.Sprintf(
		"Geschäftsjahr %d von %q auf %q zurückgesetzt (Grund: %s)",
		year, previous.Label(), domain.FiscalYearPrepared.Label(), strings.TrimSpace(reason)))
	return fy, nil
}

func isNextStatus(current, target domain.FiscalYearStatus) bool {
	return nextStatus(current) == target
}

func nextStatus(current domain.FiscalYearStatus) domain.FiscalYearStatus {
	switch current {
	case domain.FiscalYearOpen:
		return domain.FiscalYearPrepared
	case domain.FiscalYearPrepared:
		return domain.FiscalYearAdopted
	case domain.FiscalYearAdopted:
		return domain.FiscalYearDisclosed
	}
	return ""
}

// -------------------------------------------------------------
// Saldenvortrag
// -------------------------------------------------------------

// CarryForwardKind unterscheidet die drei Vortragsarten. Jede ist eine eigene
// Buchung gegen ein eigenes Gegenkonto, so wie DATEV es führt.
type CarryForwardKind string

const (
	CarryForwardSachkonto CarryForwardKind = "sachkonto"
	CarryForwardDebitor   CarryForwardKind = "debitor"
	CarryForwardKreditor  CarryForwardKind = "kreditor"
)

// CarryForwardRow ist eine Zeile der Vortragsvorschau: ein Konto mit dem Wert,
// der im neuen Jahr darauf stehen muss, und dem, was schon darauf steht.
//
// Alle drei Beträge sind vorzeichenbehaftet in Soll-Richtung: ein positiver Wert
// ist ein Sollsaldo, ein negativer ein Habensaldo. Nur so ist die Differenz eine
// Zahl und nicht ein Paar aus Betrag und Seite.
type CarryForwardRow struct {
	Account string           `json:"account"`
	Name    string           `json:"name"`
	Kind    CarryForwardKind `json:"kind"`
	// ClosingBalance ist der Schlusssaldo des Vorjahres; auf dem Ergebniskonto
	// zusätzlich das Jahresergebnis (siehe IncludesNetIncome).
	ClosingBalance domain.Cents `json:"closingBalance"`
	Carried        domain.Cents `json:"carried"`
	Difference     domain.Cents `json:"difference"`
	// OpenItems ist die Zahl der offenen Posten hinter einem Personenkonto.
	OpenItems int `json:"openItems,omitempty"`
	// IncludesNetIncome kennzeichnet die Zeile des Ergebniskontos.
	IncludesNetIncome bool `json:"includesNetIncome,omitempty"`
}

// CarryForwardPreview ist der Stand des Saldenvortrags in ein Jahr.
type CarryForwardPreview struct {
	FromYear int `json:"fromYear"`
	ToYear   int `json:"toYear"`
	// BookingDate ist das Datum, auf das gebucht würde: der erste Tag des neuen
	// Jahres, oder der erste offene Tag, wenn dieser bereits festgeschrieben ist.
	BookingDate string `json:"bookingDate"`
	Deferred    bool   `json:"deferred"`

	Rows []CarryForwardRow `json:"rows"`
	// NetIncome ist das Jahresergebnis des Vorjahres, ResultAccount das Konto,
	// auf das es gebracht wird.
	NetIncome         domain.Cents `json:"netIncome"`
	ResultAccount     string       `json:"resultAccount"`
	ResultAccountName string       `json:"resultAccountName"`

	// AlreadyCarried sagt, ob im Zieljahr Vortragswerte stehen — gemessen an den
	// Summen und nicht an den Buchungen: eine Vortragsbuchung, die außerhalb des
	// Zieljahres storniert wurde, ist keine lebende Buchung mehr, ihr Wert steht
	// aber weiterhin auf den Konten des Zieljahres.
	AlreadyCarried  bool `json:"alreadyCarried"`
	NeedsCorrection bool `json:"needsCorrection"`
	// Irreversible sagt, dass Vortragswerte im Zieljahr stehen, zu denen es keine
	// zurücknehmbare Buchung mehr gibt. Ein Korrekturvortrag würde sie
	// verdoppeln, statt sie zu ersetzen.
	Irreversible bool `json:"irreversible,omitempty"`
	// PriorYearNotCarried sagt, dass es Buchungen aus Jahren vor dem Vorjahr
	// gibt, das Vorjahr selbst aber keinen Saldenvortrag trägt. Das erklärt eine
	// Differenz, die in der Summen- und Saldenliste des Vorjahres nicht zu
	// finden ist: die offenen Posten laufen jahresübergreifend, die
	// Sachkontensalden nicht.
	PriorYearNotCarried bool `json:"priorYearNotCarried,omitempty"`
	// Entries ist die Zahl der Buchungen, die ein Lauf erzeugt (höchstens drei:
	// Sachkonten, Debitoren, Kreditoren).
	Entries int `json:"entries"`

	// BalanceDifference ist die Probe auf die Bilanzidentität: Summe aller
	// Vortragswerte. Sie muss null sein — sonst stimmen Aktiva, Passiva und
	// Jahresergebnis des Vorjahres nicht zusammen, und ein Vortrag würde den
	// Fehler ins neue Jahr tragen.
	BalanceDifference domain.Cents `json:"balanceDifference"`
	IsBalanced        bool         `json:"isBalanced"`

	// AccrualReleases sind die Auflösungen der Rechnungsabgrenzung, die der
	// Vortrag im neuen Jahr gleich mitbucht. Sie stehen in der Vorschau, weil
	// der Vortrag sonst mehr täte, als er ankündigt.
	AccrualReleases []AccrualReleaseDue `json:"accrualReleases"`
}

// HasDifference sagt, ob ein Korrekturvortrag nötig ist.
func (p *CarryForwardPreview) HasDifference() bool { return p.NeedsCorrection }

// carriedItem ist ein offener Posten, wie er vorgetragen wird.
type carriedItem struct {
	account   string
	contactID uint
	text      string
	amount    domain.Cents // vorzeichenbehaftet in Soll-Richtung
}

// carryForwardPlan ist die Vorschau samt der Zeilen, aus denen gebucht wird.
type carryForwardPlan struct {
	preview  *CarryForwardPreview
	targets  map[string]domain.Cents // Konto → Vortragswert (Soll positiv)
	items    []carriedItem
	existing []domain.JournalEntry // bestehende, nicht stornierte Vortragsbuchungen
	// residual ist der Teil der vorgetragenen Summen, der zu keiner
	// zurücknehmbaren Buchung gehört (Konto → Wert in Soll-Richtung).
	residual map[string]domain.Cents
}

// CarryForwardState stellt den Vortragsstand in ein Geschäftsjahr zusammen: je
// Konto Schlusssaldo des Vorjahres, bereits vorgetragener Wert und Differenz.
func (s *ClosingService) CarryForwardState(ctx context.Context, toYear int) (*CarryForwardPreview, error) {
	plan, err := s.plan(ctx, toYear)
	if err != nil {
		return nil, err
	}
	plan.preview.AccrualReleases = make([]AccrualReleaseDue, 0)
	if s.accruals != nil {
		if due, err := s.accruals.PendingReleases(ctx, toYear); err == nil {
			plan.preview.AccrualReleases = due
		}
	}
	return plan.preview, nil
}

func (s *ClosingService) plan(ctx context.Context, toYear int) (*carryForwardPlan, error) {
	if toYear <= 1 {
		return nil, fmt.Errorf("das Zieljahr des Saldenvortrags fehlt")
	}
	fromYear := toYear - 1

	fyFrom, err := s.yearOrDerived(ctx, fromYear)
	if err != nil {
		return nil, err
	}
	fyTo, err := s.yearOrDerived(ctx, toYear)
	if err != nil {
		return nil, err
	}

	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnovers(ctx, fromYear)
	if err != nil {
		return nil, fmt.Errorf("die Salden des Geschäftsjahres %d konnten nicht gelesen werden: %w", fromYear, err)
	}

	targets := map[string]domain.Cents{}
	kinds := map[string]CarryForwardKind{}
	names := map[string]string{}

	for number, t := range turnovers {
		// Personenkonten laufen über die offenen Posten, nicht über den Saldo:
		// die Bilanz zeigt eine Summe, die Buchhaltung braucht die Posten, gegen
		// die im neuen Jahr die Zahlungen laufen.
		if domain.IsLedgerAccount(number) {
			continue
		}
		if domain.IsCarryForwardAccount(number) {
			continue
		}
		acc, ok := chart.Lookup(number)
		if !ok || acc.StatementType != "Bilanz" {
			continue
		}
		balance := t.Debit - t.Credit
		if balance == 0 {
			continue
		}
		targets[number] += balance
		kinds[number] = CarryForwardSachkonto
		names[number] = acc.Name
	}

	netIncome := netIncomeOf(turnovers, chart)
	resultAccount := domain.ResultCarryForwardAccount(netIncome)
	if netIncome != 0 {
		// Ein Gewinn steht im Haben, in Soll-Richtung also negativ.
		targets[resultAccount] += -netIncome
		kinds[resultAccount] = CarryForwardSachkonto
		if _, ok := names[resultAccount]; !ok {
			names[resultAccount] = chart.Name(resultAccount)
		}
	}

	items, err := s.openItemsAt(ctx, fyFrom.EndDate)
	if err != nil {
		return nil, err
	}
	ledgerNames, err := s.ledgerAccountNames(ctx)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, item := range items {
		targets[item.account] += item.amount
		kind := CarryForwardDebitor
		if t, ok := domain.LedgerAccountKind(item.account); ok && t == domain.ContactTypeVendor {
			kind = CarryForwardKreditor
		}
		kinds[item.account] = kind
		names[item.account] = ledgerNames[item.account]
		counts[item.account]++
	}

	carried, existing, err := s.carriedSoFar(ctx, toYear)
	if err != nil {
		return nil, err
	}
	residual := residualOf(carried, existing)
	for account := range carried {
		if _, ok := targets[account]; ok {
			continue
		}
		targets[account] = 0
		if domain.IsLedgerAccount(account) {
			kind := CarryForwardDebitor
			if t, ok := domain.LedgerAccountKind(account); ok && t == domain.ContactTypeVendor {
				kind = CarryForwardKreditor
			}
			kinds[account] = kind
			names[account] = ledgerNames[account]
			continue
		}
		kinds[account] = CarryForwardSachkonto
		names[account] = chart.Name(account)
	}

	rows := make([]CarryForwardRow, 0, len(targets))
	var balanceSum domain.Cents
	needsCorrection := false
	for account, target := range targets {
		if domain.IsCarryForwardAccount(account) {
			continue
		}
		name := names[account]
		if name == "" {
			name = account
		}
		row := CarryForwardRow{
			Account:           account,
			Name:              name,
			Kind:              kinds[account],
			ClosingBalance:    target,
			Carried:           carried[account],
			Difference:        target - carried[account],
			OpenItems:         counts[account],
			IncludesNetIncome: account == resultAccount && netIncome != 0,
		}
		balanceSum += target
		if row.Difference != 0 {
			needsCorrection = true
		}
		if row.ClosingBalance == 0 && row.Carried == 0 {
			continue
		}
		rows = append(rows, row)
	}
	sortCarryForwardRows(rows)

	bookingDate, deferred, err := s.carryForwardDate(ctx, fyTo)
	if err != nil {
		return nil, err
	}

	priorMissing, err := s.priorCarryForwardMissing(ctx, fromYear)
	if err != nil {
		return nil, err
	}

	preview := &CarryForwardPreview{
		FromYear:          fromYear,
		ToYear:            toYear,
		BookingDate:       bookingDate,
		Deferred:          deferred,
		Rows:              rows,
		NetIncome:         netIncome,
		ResultAccount:     resultAccount,
		ResultAccountName: chart.Name(resultAccount),
		// Gemessen wird an den Summen: sonst gälte ein Zieljahr als
		// unvorgetragen, sobald die Vortragsbuchung außerhalb des Jahres
		// storniert wurde — ihr Wert steht dann weiterhin auf den Konten, und
		// ein zweiter Lauf schriebe ihn ein zweites Mal hin.
		AlreadyCarried:      len(existing) > 0 || anyNonZero(carried),
		NeedsCorrection:     needsCorrection,
		Irreversible:        anyNonZero(residual),
		PriorYearNotCarried: priorMissing,
		BalanceDifference:   balanceSum,
		IsBalanced:          balanceSum == 0,
	}
	preview.Entries = countEntries(targets)

	return &carryForwardPlan{
		preview: preview, targets: targets, items: items, existing: existing, residual: residual,
	}, nil
}

// residualOf rechnet aus, welcher Teil der vorgetragenen Summen zu keiner
// zurücknehmbaren Buchung mehr gehört.
//
// Im Regelfall ist das nichts: was im Zieljahr steht, steht dort wegen der
// lebenden Vortragsbuchungen, und ein Korrekturvortrag nimmt sie zurück. Anders,
// wenn eine Vortragsbuchung außerhalb des Zieljahres storniert wurde — die
// Generalumkehr wirkt dann im Jahr ihres Datums, der Wert im Zieljahr bleibt
// stehen, und zurücknehmen lässt er sich nicht mehr.
func residualOf(carried map[string]domain.Cents, existing []domain.JournalEntry) map[string]domain.Cents {
	residual := map[string]domain.Cents{}
	for account, value := range carried {
		if value != 0 {
			residual[account] = value
		}
	}
	for i := range existing {
		for _, line := range existing[i].Lines {
			if line.Side == domain.SideDebit {
				residual[line.Account] -= line.Amount
			} else {
				residual[line.Account] += line.Amount
			}
		}
	}
	for account, value := range residual {
		if value == 0 {
			delete(residual, account)
		}
	}
	return residual
}

func anyNonZero(values map[string]domain.Cents) bool {
	for _, v := range values {
		if v != 0 {
			return true
		}
	}
	return false
}

// priorCarryForwardMissing sagt, ob das Vorjahr selbst noch auf seinen
// Saldenvortrag wartet.
//
// Das ist der häufigste Grund für eine Differenz, die sich in der Summen- und
// Saldenliste des Vorjahres nicht finden lässt: die offenen Posten laufen
// jahresübergreifend und stehen zum Stichtag da, die Sachkontensalden des
// Vorjahres kennen den alten Bestand aber nicht. Die Meldung soll dann auf den
// übersprungenen Vortrag zeigen und nicht auf eine Liste, die in sich aufgeht.
func (s *ClosingService) priorCarryForwardMissing(ctx context.Context, fromYear int) (bool, error) {
	years, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return false, fmt.Errorf("die Geschäftsjahre des Journals konnten nicht gelesen werden: %w", err)
	}
	earlier := false
	for _, y := range years {
		if y > 0 && y < fromYear {
			earlier = true
			break
		}
	}
	if !earlier {
		return false, nil
	}
	carried, existing, err := s.carriedSoFar(ctx, fromYear)
	if err != nil {
		return false, err
	}
	return len(existing) == 0 && !anyNonZero(carried), nil
}

// countEntries zählt, wie viele Buchungen ein Lauf erzeugt.
func countEntries(targets map[string]domain.Cents) int {
	var sach, debitor, kreditor bool
	for account, value := range targets {
		if value == 0 {
			continue
		}
		switch {
		case !domain.IsLedgerAccount(account):
			sach = true
		default:
			if t, ok := domain.LedgerAccountKind(account); ok && t == domain.ContactTypeVendor {
				kreditor = true
			} else {
				debitor = true
			}
		}
	}
	n := 0
	for _, present := range []bool{sach, debitor, kreditor} {
		if present {
			n++
		}
	}
	return n
}

// carryForwardDate bestimmt das Buchungsdatum des Vortrags.
//
// Der erste Tag des neuen Jahres, außer er ist schon festgeschrieben. Das kommt
// beim Korrekturvortrag vor: bis dahin ist im neuen Jahr weitergebucht und
// vielleicht schon ein Monat festgeschrieben worden. Dann rückt der Vortrag auf
// den ersten offenen Tag — die Alternative wäre, die Festschreibung zu brechen.
func (s *ClosingService) carryForwardDate(ctx context.Context, fy *domain.FiscalYear) (string, bool, error) {
	date := fy.StartDate
	if s.festschreibungRepo == nil {
		return date, false, nil
	}
	cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, fy.Year)
	if err != nil {
		return "", false, fmt.Errorf("der Festschreibungsstand konnte nicht geprüft werden: %w", err)
	}
	if cutoff == "" || cutoff < date {
		return date, false, nil
	}
	return nextDay(cutoff), true, nil
}

// openItemsAt sammelt die offenen Posten zum Bilanzstichtag.
func (s *ClosingService) openItemsAt(ctx context.Context, cutoff string) ([]carriedItem, error) {
	entries, err := s.journalRepo.FindOpenItemCandidatesAt(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("die offenen Posten zum %s konnten nicht gelesen werden: %w", cutoff, err)
	}
	settled, err := s.allocationRepo.SettledByOpenItemAt(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("die Zahlungszuordnungen bis zum %s konnten nicht gelesen werden: %w", cutoff, err)
	}

	items := make([]carriedItem, 0, len(entries))
	for i := range entries {
		entry := &entries[i]
		// Der Vortrag des Vorjahres ist kein offener Posten, sondern seine
		// Wiederholung. Ihn erneut vorzutragen verdoppelte die Forderung.
		if entry.Source == domain.EntrySourceOpening {
			continue
		}
		line, ok := ledgerLine(entry)
		if !ok || entry.ContactID == nil {
			continue
		}
		// Der Ausgleich ist immer positiv erfasst und mindert den Posten,
		// gleich auf welcher Seite dieser steht.
		open := line.Amount - settled[entry.ID]
		if open == 0 {
			continue
		}
		amount := open
		if line.Side == domain.SideCredit {
			amount = -open
		}
		items = append(items, carriedItem{
			account:   line.Account,
			contactID: *entry.ContactID,
			text:      documentReference(entry),
			amount:    amount,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].account != items[j].account {
			return items[i].account < items[j].account
		}
		return items[i].text < items[j].text
	})
	return items, nil
}

// carriedSoFar liest, was im Zieljahr schon vorgetragen ist.
//
// Gezählt wird über alle Vortragsbuchungen des Jahres einschließlich ihrer
// Generalumkehr: deren negative Beträge heben die stornierte Buchung genau auf,
// und damit steht in der Summe, was tatsächlich auf den Konten liegt. Die
// zweite Rückgabe sind die Buchungen, die ein Korrekturvortrag zurücknehmen
// müsste — Stornos und bereits stornierte Buchungen gehören nicht dazu.
func (s *ClosingService) carriedSoFar(ctx context.Context, toYear int) (map[string]domain.Cents, []domain.JournalEntry, error) {
	entries, err := s.journalRepo.FindAll(ctx, toYear)
	if err != nil {
		return nil, nil, fmt.Errorf("die Buchungen des Geschäftsjahres %d konnten nicht gelesen werden: %w", toYear, err)
	}

	reference := carryForwardReference(toYear)
	carried := map[string]domain.Cents{}
	var live []domain.JournalEntry

	for i := range entries {
		entry := &entries[i]
		if entry.Source != domain.EntrySourceOpening || entry.DocumentNumber != reference {
			continue
		}
		for _, line := range entry.Lines {
			if line.Side == domain.SideDebit {
				carried[line.Account] += line.Amount
			} else {
				carried[line.Account] -= line.Amount
			}
		}
		if entry.Kind == domain.EntryKindReversal {
			continue
		}
		reversal, err := s.journalRepo.FindReversalOf(ctx, entry.ID)
		if err != nil {
			return nil, nil, err
		}
		if reversal == nil {
			live = append(live, *entry)
		}
	}
	return carried, live, nil
}

// CarryForward bucht den Saldenvortrag ins Zieljahr.
//
// Wiederholbar: ein erneuter Lauf nimmt die bestehenden Vortragsbuchungen per
// Generalumkehr zurück und bucht neu. Das ist der Normalfall und kein
// Ausnahmeweg — im Vorjahr wird nach dem ersten Vortrag noch gebucht, bis der
// Abschluss steht, und jede dieser Buchungen verschiebt einen Schlusssaldo.
func (s *ClosingService) CarryForward(ctx context.Context, toYear int) ([]domain.JournalEntry, error) {
	plan, err := s.plan(ctx, toYear)
	if err != nil {
		return nil, err
	}
	preview := plan.preview

	if !preview.IsBalanced {
		hint := "Prüfe die Summen- und Saldenliste des Vorjahres"
		if preview.PriorYearNotCarried {
			hint = fmt.Sprintf(
				"Im Geschäftsjahr %d steht kein Saldenvortrag, obwohl es Buchungen aus früheren Jahren "+
					"gibt: buche zuerst den Saldenvortrag %d → %d",
				preview.FromYear, preview.FromYear-1, preview.FromYear)
		}
		return nil, fmt.Errorf(
			"die Salden des Geschäftsjahres %d gehen um %s € nicht auf: Aktiva, Passiva und das "+
				"Jahresergebnis von %s € stimmen nicht zusammen. Ein Vortrag würde die Differenz ins "+
				"Geschäftsjahr %d tragen. %s",
			preview.FromYear, preview.BalanceDifference, preview.NetIncome, toYear, hint)
	}
	if preview.Entries == 0 && !preview.AlreadyCarried {
		return nil, fmt.Errorf("im Geschäftsjahr %d gibt es keine Salden, die vorzutragen wären", preview.FromYear)
	}
	if preview.AlreadyCarried && !preview.NeedsCorrection {
		return nil, fmt.Errorf(
			"der Saldenvortrag ins Geschäftsjahr %d ist auf dem aktuellen Stand; es gibt nichts zu buchen", toYear)
	}
	// Ein Korrekturvortrag ersetzt den alten; er setzt deshalb voraus, dass sich
	// der alte zurücknehmen lässt. Steht ein Wert im Zieljahr, zu dem es keine
	// stornierbare Buchung mehr gibt, würde neu zu buchen ihn verdoppeln.
	if preview.Irreversible {
		return nil, fmt.Errorf(
			"der Saldenvortrag ins Geschäftsjahr %d lässt sich nicht korrigieren: %s stehen auf den "+
				"Konten des Zieljahres, ohne dass es dazu eine zurücknehmbare Vortragsbuchung gibt – "+
				"die bestehende Vortragsbuchung wurde außerhalb des Geschäftsjahres %d storniert. Ein "+
				"neuer Vortrag würde diese Werte verdoppeln. Gleiche den stornierten Vortrag zuerst "+
				"innerhalb des Geschäftsjahres %d aus",
			toYear, residualDescription(plan.residual), toYear, toYear)
	}

	correction := preview.AlreadyCarried

	// Erst bauen und prüfen, dann schreiben: Rücknahme und Neuvortrag gehören
	// zusammen. Bräche der Lauf zwischen ihnen ab — ein Personenkonto ohne
	// Geschäftspartner, ein festgestelltes Zieljahr —, bliebe das Jahr mit
	// storniertem Altvortrag und halbem Neuvortrag zurück, also mit einer
	// Eröffnungsbilanz, die es so nie gab.
	prepared := make([]*domain.JournalEntry, 0, 3)
	for _, kind := range []CarryForwardKind{CarryForwardSachkonto, CarryForwardDebitor, CarryForwardKreditor} {
		entry := s.buildEntry(plan, kind, correction)
		if entry == nil {
			continue
		}
		if err := s.journalSvc.ValidatePostable(ctx, entry); err != nil {
			return nil, fmt.Errorf(
				"der Saldenvortrag ins Geschäftsjahr %d wurde nicht gebucht, weil eine seiner Buchungen "+
					"nicht durchgeht: %w", toYear, err)
		}
		prepared = append(prepared, entry)
	}

	for _, entry := range plan.existing {
		if _, err := s.journalSvc.ReverseOn(ctx, entry.ID, "Korrekturvortrag", preview.BookingDate); err != nil {
			return nil, fmt.Errorf(
				"der bestehende Saldenvortrag %s konnte nicht zurückgenommen werden: %w", entry.EntryNumber, err)
		}
	}

	created := make([]domain.JournalEntry, 0, len(prepared))
	for _, entry := range prepared {
		out, err := s.journalSvc.Post(ctx, entry)
		if err != nil {
			// Der Lauf ist hier zur Hälfte geschehen: das Protokoll muss sagen,
			// in welchem Zustand das Jahr zurückbleibt, damit die Wiederholung
			// keine Überraschung ist.
			s.audit(ctx, domain.AuditActionUpdate, toYear, fmt.Sprintf(
				"Saldenvortrag %d → %d abgebrochen: %d von %d Buchungen geschrieben, der bestehende "+
					"Vortrag ist bereits zurückgenommen. Das Geschäftsjahr %d hat bis zur Wiederholung "+
					"des Vortrags keine vollständige Eröffnungsbilanz (%v)",
				preview.FromYear, toYear, len(created), len(prepared), toYear, err))
			return created, fmt.Errorf("der Saldenvortrag konnte nicht gebucht werden: %w", err)
		}
		created = append(created, *out)
	}

	fyTo, err := s.YearOf(ctx, toYear)
	if err != nil {
		return created, err
	}
	now := time.Now().UTC()
	fyTo.CarriedForwardAt = &now
	if err := s.fiscalYearRepo.Save(ctx, fyTo); err != nil {
		return created, fmt.Errorf("der Vortragsstand konnte nicht gespeichert werden: %w", err)
	}

	label := "Saldenvortrag"
	if correction {
		label = "Korrekturvortrag"
	}
	s.audit(ctx, domain.AuditActionCreate, toYear, fmt.Sprintf(
		"%s %d → %d zum %s: %d Buchungen, Jahresergebnis %s € auf Konto %s",
		label, preview.FromYear, toYear, preview.BookingDate, len(created),
		preview.NetIncome, preview.ResultAccount))

	// Die Auflösung der Rechnungsabgrenzung folgt unmittelbar: der Posten des
	// Vorjahres wird am ersten Tag des neuen Jahres wieder Aufwand bzw. Ertrag.
	// Sie scheitert, ohne den Vortrag mitzureißen — der steht und ist richtig;
	// was fehlt, ist die Folgebuchung, und die Meldung sagt es.
	if s.accruals != nil {
		releases, err := s.accruals.ReleaseInto(ctx, toYear)
		created = append(created, releases...)
		if err != nil {
			return created, fmt.Errorf(
				"der Saldenvortrag ins Geschäftsjahr %d steht; die Auflösung der Rechnungsabgrenzung "+
					"ist aber unvollständig: %w", toYear, err)
		}
	}

	return created, nil
}

// buildEntry baut die Buchung einer Vortragsart, oder nil, wenn es nichts zu
// buchen gibt. Das Gegenkonto nimmt die Summe auf, damit die Buchung in sich
// ausgeglichen ist — dass die drei Gegenkonten zusammen null ergeben, ist die
// Bilanzidentität und wurde vorher geprüft.
func (s *ClosingService) buildEntry(plan *carryForwardPlan, kind CarryForwardKind, correction bool) *domain.JournalEntry {
	preview := plan.preview
	var lines []domain.JournalLine
	var sum domain.Cents
	var counterAccount string

	switch kind {
	case CarryForwardSachkonto:
		counterAccount = domain.AccountSaldenvortraegeSachkonten
		accounts := make([]string, 0, len(plan.targets))
		for account, value := range plan.targets {
			if value == 0 || domain.IsLedgerAccount(account) {
				continue
			}
			accounts = append(accounts, account)
		}
		sort.Strings(accounts)
		for _, account := range accounts {
			value := plan.targets[account]
			text := "Saldenvortrag"
			if account == preview.ResultAccount && preview.NetIncome != 0 {
				text = fmt.Sprintf("Jahresergebnis %d und Saldenvortrag", preview.FromYear)
			}
			lines = append(lines, carryLine(account, value, nil, text))
			sum += value
		}
	case CarryForwardDebitor, CarryForwardKreditor:
		counterAccount = domain.AccountSaldenvortraegeDebitoren
		wanted := domain.ContactTypeCustomer
		if kind == CarryForwardKreditor {
			counterAccount = domain.AccountSaldenvortraegeKreditoren
			wanted = domain.ContactTypeVendor
		}
		for _, item := range plan.items {
			t, ok := domain.LedgerAccountKind(item.account)
			if !ok || t != wanted {
				continue
			}
			contactID := item.contactID
			lines = append(lines, carryLine(item.account, item.amount, &contactID, item.text))
			sum += item.amount
		}
	}

	if len(lines) == 0 {
		return nil
	}
	// Das Gegenkonto entfällt, wenn die Vortragsart für sich aufgeht. Bei den
	// Sachkonten ist das der Regelfall, sobald es keine offenen Posten gibt:
	// Aktiva, Passiva und Jahresergebnis gleichen sich dann schon untereinander
	// aus, und eine Zeile über null wäre keine Buchung.
	if sum != 0 {
		lines = append(lines, carryLine(counterAccount, -sum, nil, "Saldenvortrag"))
	}

	date := preview.BookingDate
	return &domain.JournalEntry{
		BookingDate:        date,
		DocumentDate:       date,
		ServiceDateFrom:    date,
		ServiceDateTo:      date,
		Description:        carryForwardDescription(preview, kind, correction),
		Source:             domain.EntrySourceOpening,
		DocumentNumber:     carryForwardReference(preview.ToYear),
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
}

func carryForwardReference(toYear int) string { return fmt.Sprintf("SV %d", toYear) }

// residualDescription benennt die Konten, auf denen ein nicht mehr
// zurücknehmbarer Vortragswert steht — höchstens drei, danach die Zahl der
// übrigen: eine Fehlermeldung soll den Einstieg zeigen, nicht die ganze Bilanz.
func residualDescription(residual map[string]domain.Cents) string {
	accounts := make([]string, 0, len(residual))
	for account := range residual {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)

	parts := make([]string, 0, 3)
	for i, account := range accounts {
		if i == 3 {
			parts = append(parts, fmt.Sprintf("und %d weitere Konten", len(accounts)-3))
			break
		}
		parts = append(parts, fmt.Sprintf("%s € auf Konto %s", residual[account], account))
	}
	return strings.Join(parts, ", ")
}

func carryForwardDescription(preview *CarryForwardPreview, kind CarryForwardKind, correction bool) string {
	label := "Saldenvortrag"
	if correction {
		label = "Korrekturvortrag"
	}
	part := "Sachkonten"
	switch kind {
	case CarryForwardDebitor:
		part = "Debitoren"
	case CarryForwardKreditor:
		part = "Kreditoren"
	}
	text := fmt.Sprintf("%s %s %d aus %d", label, part, preview.ToYear, preview.FromYear)
	if preview.Deferred {
		text += fmt.Sprintf(" – gebucht zum %s, weil der Jahresbeginn bereits festgeschrieben ist",
			germanDate(preview.BookingDate))
	}
	return text
}

// carryLine macht aus einem vorzeichenbehafteten Wert eine Buchungszeile.
func carryLine(account string, value domain.Cents, contactID *uint, text string) domain.JournalLine {
	line := domain.JournalLine{Account: account, ContactID: contactID, Text: text}
	if value >= 0 {
		line.Side = domain.SideDebit
		line.Amount = value
		return line
	}
	line.Side = domain.SideCredit
	line.Amount = -value
	return line
}

// -------------------------------------------------------------
// Hilfen
// -------------------------------------------------------------

// netIncomeOf rechnet das Jahresergebnis aus den Verkehrszahlen: Erträge minus
// Aufwendungen aller GuV-Konten.
//
// Es wird nicht gebucht. Bei SKR04 werden die Erfolgskonten nicht über ein
// Abschlusskonto geschlossen; das Ergebnis ist eine Rechnung über die Konten und
// erscheint erst im Folgejahr als Buchung — im Saldenvortrag auf dem
// Gewinn-/Verlustvortrag.
func netIncomeOf(turnovers map[string]domain.AccountTurnover, chart *accounting.Chart) domain.Cents {
	var result domain.Cents
	for number, t := range turnovers {
		if domain.IsLedgerAccount(number) {
			continue
		}
		acc, ok := chart.Lookup(number)
		if !ok || acc.StatementType != "GuV" {
			continue
		}
		result += t.Credit - t.Debit
	}
	return result
}

func (s *ClosingService) chart(ctx context.Context) (*accounting.Chart, error) {
	if s.journalSvc != nil {
		return s.journalSvc.Chart(ctx)
	}
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Kontenplan konnte nicht geladen werden: %w", err)
	}
	return accounting.NewChart(accounts), nil
}

func (s *ClosingService) ledgerAccountNames(ctx context.Context) (map[string]string, error) {
	names := map[string]string{}
	if s.contactRepo == nil {
		return names, nil
	}
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range contacts {
		names[c.LedgerAccount] = c.Name
	}
	return names, nil
}

func (s *ClosingService) audit(ctx context.Context, action domain.AuditAction, year int, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "FISCAL_YEAR", fmt.Sprintf("%d", year), details)
}

// documentReference benennt einen offenen Posten so, wie er im Kontoblatt des
// neuen Jahres wiederzufinden sein muss: Belegnummer und Belegdatum.
func documentReference(entry *domain.JournalEntry) string {
	number := entry.DocumentNumber
	if number == "" {
		number = entry.EntryNumber
	}
	if entry.DocumentDate == "" {
		return number
	}
	return fmt.Sprintf("%s vom %s", number, germanDate(entry.DocumentDate))
}

func sortCarryForwardRows(rows []CarryForwardRow) {
	rank := map[CarryForwardKind]int{CarryForwardSachkonto: 0, CarryForwardDebitor: 1, CarryForwardKreditor: 2}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rank[rows[i].Kind] < rank[rows[j].Kind]
		}
		return rows[i].Account < rows[j].Account
	})
}

func shortSuffix(fy *domain.FiscalYear) string {
	if fy.IsShort {
		return ", Rumpfgeschäftsjahr"
	}
	return ""
}

// nextDay liefert den Folgetag eines Datums im Format JJJJ-MM-TT.
func nextDay(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}

// lastDayOfTwelveMonths liefert den letzten Tag eines vollen Geschäftsjahres,
// das am gegebenen Tag beginnt.
func lastDayOfTwelveMonths(start string) string {
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return start
	}
	return t.AddDate(0, 12, 0).AddDate(0, 0, -1).Format("2006-01-02")
}

func germanDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("02.01.2006")
}

// SetAverageEmployees hält die durchschnittliche Arbeitnehmerzahl eines
// Geschäftsjahres fest.
//
// Sie ist das dritte Merkmal des § 267 Abs. 1 HGB und lässt sich aus der
// Buchführung nicht ableiten: § 267 Abs. 5 HGB bildet den Durchschnitt aus vier
// Quartalsstichtagen, und wer an ihnen beschäftigt war, steht in keinem Konto.
// Sie gehört ans Geschäftsjahr und nicht in die Stammdaten, weil sie sich von
// Jahr zu Jahr ändert — und weil die Größenklasse eines abgeschlossenen Jahres
// sonst mit dem heutigen Personalstand berechnet würde.
func (s *ClosingService) SetAverageEmployees(ctx context.Context, year, count int) (*domain.FiscalYear, error) {
	if count < 0 {
		return nil, fmt.Errorf("die durchschnittliche Arbeitnehmerzahl kann nicht negativ sein")
	}
	fy, err := s.YearOf(ctx, year)
	if err != nil {
		return nil, err
	}
	// Ab der Feststellung steht der Abschluss. An der Arbeitnehmerzahl hängt
	// über die Größenklasse die Gliederungstiefe und der Umfang der
	// Offenlegung — sie danach zu ändern hieße, einen festgestellten Abschluss
	// nachträglich anders auszuweisen.
	if fy.IsAdopted() {
		return nil, fmt.Errorf(
			"das Geschäftsjahr %d ist %s; die Arbeitnehmerzahl bestimmt über die Größenklasse die "+
				"Gliederung des Abschlusses. Nehmen Sie die Feststellung zurück, um sie zu ändern",
			year, fy.Status.Label())
	}
	if fy.AverageEmployees == count {
		return fy, nil
	}
	previous := fy.AverageEmployees
	fy.AverageEmployees = count
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return nil, err
	}
	s.audit(ctx, domain.AuditActionUpdate, year, fmt.Sprintf(
		"Durchschnittliche Arbeitnehmerzahl des Geschäftsjahres %d von %d auf %d gesetzt (§ 267 Abs. 5 HGB)",
		year, previous, count))
	return fy, nil
}

// PeriodOf liefert den Zeitraum eines Geschäftsjahres, ohne ihn anzulegen.
//
// Die Bilanz fragt danach, und eine Ansicht darf nichts erzeugen: wer den
// Abschluss von 2026 ansieht, fragt damit auch nach dem Vorjahr — und fände es
// danach in der Jahresauswahl wieder, ohne es angelegt zu haben.
func (s *ClosingService) PeriodOf(ctx context.Context, year int) (*domain.FiscalYear, error) {
	return s.yearOrDerived(ctx, year)
}

// EarliestFiscalYear ist das früheste erfasste Geschäftsjahr, oder 0, wenn es
// keines gibt. Die Größenklasse braucht es für § 267 Abs. 4 Satz 2 HGB: der
// erste Abschluss nach der Gründung wird nach seinem eigenen Stichtag beurteilt.
func (s *ClosingService) EarliestFiscalYear(ctx context.Context) (int, error) {
	years, err := s.fiscalYearRepo.FindAll(ctx)
	if err != nil {
		return 0, err
	}
	earliest := 0
	for _, fy := range years {
		if earliest == 0 || fy.Year < earliest {
			earliest = fy.Year
		}
	}
	journalYears, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return earliest, nil
	}
	for _, y := range journalYears {
		if y > 0 && (earliest == 0 || y < earliest) {
			earliest = y
		}
	}
	return earliest, nil
}
