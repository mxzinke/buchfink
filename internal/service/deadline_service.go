package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// DeadlineService führt alle Termine eines Jahres an einer Stelle zusammen.
//
// Bis hierher rechnete die Ansicht die Fristen selbst und merkte sich die Haken
// im localStorage. Beides war falsch. Eine Frist ist eine Aussage über den
// Mandanten und nicht über den Browser, und „erledigt" ergibt sich aus den
// Daten: eine übermittelte Voranmeldung ist abgegeben, ein festgeschriebener
// Monat ist festgeschrieben. Ein Haken daneben könnte nur widersprechen.
//
// Ein Haken bleibt für das, was Buchfink nicht sieht — die
// Umsatzsteuer-Jahreserklärung etwa. Er steht in der Datenbank.
type DeadlineService struct {
	vatSvc       *VatReturnService
	zmSvc        *ZMService
	settingsRepo domain.SettingsRepository

	festschreibungRepo domain.FestschreibungRepository
	deadlineRepo       domain.DeadlineRepository
	auditRepo          domain.AuditRepository

	statements  StatementDeadlineSource
	foundations FoundationDeadlineSource

	fiscalYear int
}

// StatementDeadlineSource liefert die Termine des Jahresabschlusses aus Welle 2.
type StatementDeadlineSource interface {
	Deadlines(ctx context.Context, year int) ([]domain.Deadline, error)
}

// FoundationDeadlineSource liefert die Pflichten aus der Gründung.
type FoundationDeadlineSource interface {
	GetState(ctx context.Context) (*FoundationState, error)
}

// NewDeadlineService wires the Fristenübersicht.
func NewDeadlineService(
	vatSvc *VatReturnService,
	zmSvc *ZMService,
	settingsRepo domain.SettingsRepository,
	festschreibungRepo domain.FestschreibungRepository,
	deadlineRepo domain.DeadlineRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *DeadlineService {
	return &DeadlineService{
		vatSvc:             vatSvc,
		zmSvc:              zmSvc,
		settingsRepo:       settingsRepo,
		festschreibungRepo: festschreibungRepo,
		deadlineRepo:       deadlineRepo,
		auditRepo:          auditRepo,
		fiscalYear:         fiscalYear,
	}
}

// SetStatementSource wires the Abschlusstermine.
func (s *DeadlineService) SetStatementSource(src StatementDeadlineSource) { s.statements = src }

// SetFoundationSource wires the Gründungspflichten.
func (s *DeadlineService) SetFoundationSource(src FoundationDeadlineSource) { s.foundations = src }

// SetFiscalYear updates the active fiscal year.
func (s *DeadlineService) SetFiscalYear(year int) { s.fiscalYear = year }

// Die Schlüssel der Termine. Sie landen in der Datenbank, sobald ein Termin von
// Hand abgehakt wird, und dürfen sich deshalb nicht mehr ändern.
const (
	DeadlineKeyVatReturn  = "ustva"
	DeadlineKeyZM         = "zm"
	DeadlineKeyPrepayment = "sondervorauszahlung"
	DeadlineKeyCommit     = "festschreibung"
	DeadlineKeyAnnualVat  = "ust.jahreserklaerung"
	DeadlineKeyFoundation = "gruendung"
)

// Deadlines liefert alle Termine eines Jahres, nach Datum sortiert.
func (s *DeadlineService) Deadlines(ctx context.Context, year int) ([]domain.Deadline, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}

	out := make([]domain.Deadline, 0, 32)
	out = append(out, s.vatDeadlines(ctx, year, cfg)...)
	out = append(out, s.zmDeadlines(ctx, year)...)
	out = append(out, s.prepaymentDeadline(ctx, year, cfg)...)
	out = append(out, s.commitDeadlines(ctx, year, cfg)...)
	out = append(out, s.annualVatDeadline(year)...)

	if s.statements != nil {
		if statementDeadlines, err := s.statements.Deadlines(ctx, year); err == nil {
			out = append(out, statementDeadlines...)
		}
	}
	if s.foundations != nil {
		out = append(out, s.foundationDeadlines(ctx, year)...)
	}

	// Der manuelle Haken zählt nur, wo er nicht aus den Daten kommt. Ein
	// abgehakter Termin, den die Daten als offen ausweisen, wäre eine
	// Selbstauskunft gegen den eigenen Bestand.
	done, err := s.manualMarks(ctx)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].IsDone {
			continue
		}
		if date, ok := done[out[i].Key]; ok {
			out[i].IsDone, out[i].DoneOn = true, date
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DueDate != out[j].DueDate {
			return out[i].DueDate < out[j].DueDate
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

// MarkDone setzt den Haken an einem Termin, der sich nicht aus den Daten ergibt.
func (s *DeadlineService) MarkDone(ctx context.Context, key, date string) error {
	if key == "" {
		return fmt.Errorf("zum Abhaken gehört der Termin")
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if s.deadlineRepo == nil {
		return fmt.Errorf("kein aktiver Mandant")
	}
	if err := s.deadlineRepo.Mark(ctx, key, date); err != nil {
		return fmt.Errorf("der Termin konnte nicht abgehakt werden: %w", err)
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "DEADLINE", key,
			fmt.Sprintf("Termin %s am %s als erledigt vermerkt", key, date))
	}
	return nil
}

// -------------------------------------------------------------
// Die einzelnen Terminarten
// -------------------------------------------------------------

func (s *DeadlineService) vatDeadlines(ctx context.Context, year int, cfg *domain.CompanySettings) []domain.Deadline {
	if s.vatSvc == nil {
		return nil
	}
	// Wer vom Voranmeldungsverfahren befreit ist (§ 18 Abs. 2 Satz 3 UStG),
	// gibt keine Voranmeldung ab, sondern nur die Jahreserklärung. Ein Termin
	// „Voranmeldung Jahr" wäre eine Pflicht, die es nicht gibt — und die
	// Jahreserklärung steht ohnehin schon in der Liste.
	if s.vatSvc.PeriodType(ctx) == domain.VatPeriodYear {
		return nil
	}
	periods, err := s.vatSvc.Periods(ctx, year)
	if err != nil {
		return nil
	}
	note := ""
	if cfg.PermanentExtension {
		note = " Die Dauerfristverlängerung verschiebt den Termin um einen Monat (§ 46 UStDV)."
	}

	out := make([]domain.Deadline, 0, len(periods))
	for _, p := range periods {
		d := domain.Deadline{
			Key:         fmt.Sprintf("%s.%s", DeadlineKeyVatReturn, p.Key),
			Title:       fmt.Sprintf("Umsatzsteuer-Voranmeldung %s", p.Label),
			DueDate:     p.DueDate,
			Period:      p.Label,
			Reference:   "§ 18 Abs. 1 UStG",
			FiscalYear:  year,
			Description: "Anmeldung in Mein ELSTER; danach die Übermittlung mit dem Transferticket bestätigen." + note,
		}
		if p.Status == domain.VatReturnSubmitted {
			d.IsDone, d.DoneOn = true, p.SubmittedAt
		}
		out = append(out, d)
	}
	return out
}

func (s *DeadlineService) zmDeadlines(ctx context.Context, year int) []domain.Deadline {
	if s.zmSvc == nil {
		return nil
	}
	periods, err := s.zmSvc.Periods(ctx, year)
	if err != nil {
		return nil
	}
	out := make([]domain.Deadline, 0, len(periods))
	for _, p := range periods {
		// Ein Zeitraum ohne innergemeinschaftliche Umsätze braucht keine Meldung
		// (§ 18a Abs. 1 Satz 1 UStG: „soweit … ausgeführt"), und ein Termin, der
		// nichts verlangt, gehört nicht in eine Fristenliste.
		if p.Total == 0 && p.ReturnID == 0 {
			continue
		}
		d := domain.Deadline{
			Key:         fmt.Sprintf("%s.%s", DeadlineKeyZM, p.Key),
			Title:       fmt.Sprintf("Zusammenfassende Meldung %s", p.Label),
			DueDate:     p.DueDate,
			Period:      p.Label,
			Reference:   "§ 18a Abs. 1 UStG",
			FiscalYear:  year,
			Description: "Übermittlung an das Bundeszentralamt für Steuern bis zum 25. Tag nach Ablauf des Meldezeitraums.",
		}
		if p.Status == domain.VatReturnSubmitted {
			d.IsDone, d.DoneOn = true, p.SubmittedAt
		}
		out = append(out, d)
	}
	return out
}

// prepaymentDeadline ist die Sondervorauszahlung: Anmeldung und Zahlung bis zum
// 10. Februar (§ 48 Abs. 1 und 2 UStDV).
//
// Sie gibt es nur mit Dauerfristverlängerung *und* nur bei monatlichem
// Voranmeldungszeitraum. § 47 Abs. 1 UStDV verlangt sie von den Unternehmern,
// die ihre Voranmeldungen monatlich abzugeben haben; dem Vierteljahreszahler
// wird die Dauerfristverlängerung nach § 46 UStDV ohne Sondervorauszahlung
// gewährt. Ein Termin an ihn wäre eine Pflicht, die es nicht gibt — und die
// schlimmste Sorte Frist ist die erfundene.
func (s *DeadlineService) prepaymentDeadline(ctx context.Context, year int, cfg *domain.CompanySettings) []domain.Deadline {
	if !cfg.PermanentExtension {
		return nil
	}
	if s.periodType(ctx, cfg) != domain.VatPeriodMonth {
		return nil
	}
	due := accounting.NextWorkday(time.Date(year, time.February, 10, 0, 0, 0, 0, time.UTC))
	return []domain.Deadline{{
		Key:        fmt.Sprintf("%s.%d", DeadlineKeyPrepayment, year),
		Title:      fmt.Sprintf("Sondervorauszahlung %d anmelden und zahlen", year),
		DueDate:    due.Format("2006-01-02"),
		Period:     fmt.Sprintf("Jahr %d", year),
		Reference:  "§ 47 Abs. 1, § 48 Abs. 1 und 2 UStDV",
		FiscalYear: year,
		Description: "Ein Elftel der Vorauszahlungen des Vorjahres. Sie wird in der letzten Voranmeldung " +
			"des Jahres unter Kennziffer 39 angerechnet.",
	}}
}

// Nicht in dieser Liste stehen die Vorauszahlungen auf die Ertragsteuern
// (10.03., 10.06., 10.09., 10.12. — § 37 Abs. 1 EStG, § 31 Abs. 1 KStG) und auf
// die Gewerbesteuer (15.02., 15.05., 15.08., 15.11. — § 19 Abs. 1 GewStG). Die
// Ansicht führte sie vor dieser Welle als feste Termine; das war eine Behauptung
// über den Mandanten, die Buchfink nicht belegen kann. Vorauszahlungen entstehen
// erst durch den Bescheid des Finanzamts bzw. der Gemeinde, ihre Höhe steht dort
// und nirgends in diesen Daten — und ein Termin, den niemand schuldet, steht auf
// der Fristenliste jedes Jahr viermal rot. Kommt der Bescheid als Stammdatum in
// Buchfink an, gehören die vier Termine wieder hierher, abgeleitet wie die
// Sondervorauszahlung aus der Dauerfristverlängerung.

// commitDeadlines sind die Festschreibungen der Monate. Erledigt ist, was
// festgeschrieben ist — das steht in den Daten und nicht in einem Haken.
//
// Die Frist ist das Ende des Folgemonats, verlängert um die Nachfrist aus den
// Einstellungen (`commit_grace_days`). Sie muss dieselbe sein, die der Prüflauf
// unter `commit_overdue` anlegt: stünde hier ein anderer Tag, meldete die
// Fristenliste einen Monat als offen, den der Prüfbericht noch nicht anmahnt —
// oder umgekehrt.
func (s *DeadlineService) commitDeadlines(ctx context.Context, year int, cfg *domain.CompanySettings) []domain.Deadline {
	if s.festschreibungRepo == nil {
		return nil
	}
	committed, err := s.festschreibungRepo.LatestCutoff(ctx, year)
	if err != nil {
		return nil
	}
	grace := 0
	if cfg != nil && cfg.CommitGraceDays > 0 {
		grace = cfg.CommitGraceDays
	}
	out := make([]domain.Deadline, 0, 12)
	for _, p := range accounting.VatPeriodsOfYear(year, domain.VatPeriodMonth) {
		d := domain.Deadline{
			Key:        fmt.Sprintf("%s.%s", DeadlineKeyCommit, p.Key),
			Title:      fmt.Sprintf("%s festschreiben", p.Label),
			DueDate:    addDays(endOfNextMonth(p.To), grace),
			Period:     p.Label,
			Reference:  "GoBD Rz. 107",
			FiscalYear: year,
			Description: "Nach der Festschreibung nimmt der Monat keine Buchung mehr auf; Korrekturen " +
				"geschehen über eine Generalumkehr im laufenden Zeitraum.",
		}
		if committed >= p.To {
			d.IsDone, d.DoneOn = true, committed
		}
		out = append(out, d)
	}
	return out
}

// annualVatDeadline ist die Umsatzsteuer-Jahreserklärung. Buchfink erzeugt sie
// nicht und sieht ihre Abgabe nicht — sie ist deshalb der Termin, der einen
// Haken von Hand braucht.
func (s *DeadlineService) annualVatDeadline(year int) []domain.Deadline {
	due := accounting.NextWorkday(time.Date(year+1, time.July, 31, 0, 0, 0, 0, time.UTC))
	return []domain.Deadline{{
		Key:        fmt.Sprintf("%s.%d", DeadlineKeyAnnualVat, year),
		Title:      fmt.Sprintf("Umsatzsteuer-Jahreserklärung %d abgeben", year),
		DueDate:    due.Format("2006-01-02"),
		Period:     fmt.Sprintf("Jahr %d", year),
		Reference:  "§ 18 Abs. 3 UStG, § 149 Abs. 2 AO",
		FiscalYear: year,
		Description: "Die Jahreserklärung entsteht außerhalb von Buchfink. Sie ist deshalb der einzige " +
			"Termin hier, der von Hand abgehakt wird.",
	}}
}

func (s *DeadlineService) foundationDeadlines(ctx context.Context, year int) []domain.Deadline {
	state, err := s.foundations.GetState(ctx)
	if err != nil || state == nil || !state.HasFoundation {
		return nil
	}
	out := make([]domain.Deadline, 0, len(state.Duties))
	for _, duty := range state.Duties {
		if duty.DueDate == "" {
			continue
		}
		out = append(out, domain.Deadline{
			Key:         fmt.Sprintf("%s.%s", DeadlineKeyFoundation, duty.Key),
			Title:       duty.Title,
			DueDate:     duty.DueDate,
			Period:      duty.Deadline,
			Reference:   duty.Reference,
			Description: duty.Description,
			FiscalYear:  year,
			IsDone:      duty.IsDone,
			DoneOn:      duty.DoneOn,
		})
	}
	return out
}

// periodType ist der Voranmeldungszeitraum des Mandanten. Gefragt wird der
// Voranmeldungsdienst, damit Fristenliste und Anmeldung denselben Zeitraum
// annehmen; ohne ihn genügen die schon gelesenen Unternehmensdaten.
func (s *DeadlineService) periodType(ctx context.Context, cfg *domain.CompanySettings) domain.VatPeriodType {
	if s.vatSvc != nil {
		return s.vatSvc.PeriodType(ctx)
	}
	return vatPeriodTypeOf(cfg)
}

func (s *DeadlineService) manualMarks(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	if s.deadlineRepo == nil {
		return out, nil
	}
	marks, err := s.deadlineRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("die abgehakten Termine konnten nicht gelesen werden: %w", err)
	}
	for _, m := range marks {
		out[m.Key] = m.DoneOn
	}
	return out, nil
}
