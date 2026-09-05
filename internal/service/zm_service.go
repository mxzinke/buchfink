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

// ZMService erzeugt, speichert und bestätigt die Zusammenfassenden Meldungen
// (§ 18a UStG).
//
// Wie bei der Voranmeldung übermittelt Buchfink nicht selbst: es erzeugt die
// Datei im Format des BZSt-Online-Portals, der Anwender lädt sie dort hoch und
// bestätigt die Übermittlung. Und wie dort hängt die Bestätigung an der
// Festschreibung: eine übermittelte Meldung muss sich auf einen unveränderlichen
// Stand stützen, sonst entfernt sie sich nach der Bestätigung stillschweigend
// vom Journal. Dazu kommen die Stammdaten — eine Meldung ohne USt-IdNr. des
// Abnehmers ist keine Meldung.
type ZMService struct {
	journalRepo        domain.JournalRepository
	contactRepo        domain.ContactRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository
	zmRepo             domain.ZMReturnRepository
	vatReturnRepo      domain.VatReturnRepository
	auditRepo          domain.AuditRepository
	fiscalYear         int
}

// NewZMService wires the Zusammenfassende Meldung.
func NewZMService(
	journalRepo domain.JournalRepository,
	contactRepo domain.ContactRepository,
	settingsRepo domain.SettingsRepository,
	festschreibungRepo domain.FestschreibungRepository,
	zmRepo domain.ZMReturnRepository,
	vatReturnRepo domain.VatReturnRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *ZMService {
	return &ZMService{
		journalRepo:        journalRepo,
		contactRepo:        contactRepo,
		settingsRepo:       settingsRepo,
		festschreibungRepo: festschreibungRepo,
		zmRepo:             zmRepo,
		vatReturnRepo:      vatReturnRepo,
		auditRepo:          auditRepo,
		fiscalYear:         fiscalYear,
	}
}

// SetFiscalYear updates the active fiscal year.
func (s *ZMService) SetFiscalYear(year int) { s.fiscalYear = year }

// ZMPeriodStatus ist ein Meldezeitraum mit Fälligkeit und Stand.
type ZMPeriodStatus struct {
	accounting.VatPeriod
	DueDate  string                 `json:"dueDate"`
	Status   domain.VatReturnStatus `json:"status"`
	ReturnID uint                   `json:"returnId,omitempty"`
	// Committed meldet, ob der Meldezeitraum festgeschrieben ist. Die
	// Bestätigung setzt das voraus (ensureCommitted) — ohne das Feld erführe es
	// die Oberfläche erst, wenn der Anwender den Dialog schon ausgefüllt hat.
	Committed   bool         `json:"committed"`
	Total       domain.Cents `json:"total"`
	SubmittedAt string       `json:"submittedAt,omitempty"`
	IsOverdue   bool         `json:"isOverdue"`
}

// Periods liefert die Meldezeiträume eines Jahres. Ihre Länge folgt aus den
// Umsätzen, nicht aus einer Einstellung (§ 18a Abs. 1 UStG).
func (s *ZMService) Periods(ctx context.Context, year int) ([]ZMPeriodStatus, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	movements, err := s.movements(ctx, year)
	if err != nil {
		return nil, err
	}
	periods := accounting.ZMPeriodsOfYear(year, movements)

	saved, err := s.zmRepo.FindByFiscalYear(ctx, year)
	if err != nil {
		return nil, err
	}
	latest := map[string]*domain.ZMReturn{}
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
	out := make([]ZMPeriodStatus, 0, len(periods))
	for _, p := range periods {
		st := ZMPeriodStatus{
			VatPeriod: p,
			DueDate:   accounting.ZMDueDate(p),
			Status:    domain.VatReturnDraft,
			Committed: cutoff != "" && cutoff >= p.To,
		}
		for _, m := range movements {
			if m.Date >= p.From && m.Date <= p.To {
				st.Total += m.Amount
			}
		}
		if r := latest[p.Key]; r != nil {
			st.Status = r.Status
			st.ReturnID = r.ID
			st.SubmittedAt = r.SubmittedAt
		}
		st.IsOverdue = st.Status != domain.VatReturnSubmitted && st.Total != 0 && st.DueDate < today
		out = append(out, st)
	}
	return out, nil
}

// Draft rechnet die Meldung eines Zeitraums neu, ohne sie zu speichern.
func (s *ZMService) Draft(ctx context.Context, periodKey string) (*domain.ZMReturn, error) {
	period, err := accounting.ParseVatPeriodKey(periodKey)
	if err != nil {
		return nil, err
	}
	return s.build(ctx, period)
}

// Save legt die Meldung als Entwurf ab oder schreibt einen bestehenden Entwurf
// fort.
//
// Wie bei der Voranmeldung entsteht für einen bereits übermittelten
// Meldezeitraum kein zweiter Entwurf: er wäre eine zweite Erstmeldung desselben
// Zeitraums. Geändert wird über eine berichtigte Meldung (§ 18a Abs. 10 UStG).
func (s *ZMService) Save(ctx context.Context, periodKey string) (*domain.ZMReturn, error) {
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
				"für den Zeitraum %s ist am %s bereits eine Zusammenfassende Meldung übermittelt "+
					"(Transferticket %s). Lege stattdessen eine berichtigte Meldung an",
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
		for i := range fresh.Lines {
			fresh.Lines[i].ID = 0
			fresh.Lines[i].ZMReturnID = fresh.ID
		}
		if err := s.zmRepo.Update(ctx, fresh); err != nil {
			return nil, fmt.Errorf("die Zusammenfassende Meldung konnte nicht gespeichert werden: %w", err)
		}
	} else if err := s.zmRepo.Create(ctx, fresh); err != nil {
		return nil, fmt.Errorf("die Zusammenfassende Meldung konnte nicht gespeichert werden: %w", err)
	}

	s.audit(ctx, domain.AuditActionCreate, fresh.ID, fmt.Sprintf(
		"Zusammenfassende Meldung %s als Entwurf gespeichert (%d Zeilen, %s €)",
		fresh.PeriodKey, len(fresh.Lines), fresh.TotalSupplies+fresh.TotalServices))
	return fresh, nil
}

// List liefert die gespeicherten Meldungen eines Jahres.
func (s *ZMService) List(ctx context.Context, year int) ([]domain.ZMReturn, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	return s.zmRepo.FindByFiscalYear(ctx, year)
}

// ConfirmSubmitted bestätigt die Übermittlung an das Bundeszentralamt.
func (s *ZMService) ConfirmSubmitted(ctx context.Context, id uint, date, ticket, note string) (*domain.ZMReturn, error) {
	rec, err := s.zmRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("die Zusammenfassende Meldung %d wurde nicht gefunden: %w", id, err)
	}
	if rec.Status == domain.VatReturnSubmitted {
		return nil, fmt.Errorf(
			"die Zusammenfassende Meldung %s ist bereits am %s als übermittelt bestätigt (Transferticket %s). "+
				"Eine Änderung geschieht über eine berichtigte Meldung", rec.PeriodKey, rec.SubmittedAt, rec.TransferTicket)
	}
	if len(date) != 10 {
		return nil, fmt.Errorf("zur Bestätigung gehört das Datum der Übermittlung (erwartet JJJJ-MM-TT)")
	}
	if strings.TrimSpace(ticket) == "" {
		return nil, fmt.Errorf("zur Bestätigung gehört das Transferticket aus dem BZSt-Online-Portal")
	}
	if err := s.ensureCommitted(ctx, rec); err != nil {
		return nil, err
	}
	if err := s.ensureFirstOrCorrection(ctx, rec); err != nil {
		return nil, err
	}

	// Der Stand von heute entscheidet: eine Meldung, an der eine USt-IdNr.
	// fehlt, wurde so nicht übermittelt — das Portal nimmt sie nicht an.
	period, err := accounting.ParseVatPeriodKey(rec.PeriodKey)
	if err != nil {
		return nil, err
	}
	current, err := s.build(ctx, period)
	if err != nil {
		return nil, err
	}
	if len(current.Findings) > 0 {
		return nil, fmt.Errorf(
			"die Zusammenfassende Meldung %s ist unvollständig und kann nicht als übermittelt bestätigt werden: %s",
			rec.PeriodKey, strings.Join(current.Findings, "; "))
	}

	rec.Status = domain.VatReturnSubmitted
	rec.SubmittedAt = date
	rec.TransferTicket = strings.TrimSpace(ticket)
	rec.SubmissionNote = note
	if err := s.zmRepo.Update(ctx, rec); err != nil {
		return nil, fmt.Errorf("die Bestätigung konnte nicht gespeichert werden: %w", err)
	}

	// Statuswechsel, nicht Ausgabe — siehe VatReturnService.ConfirmSubmitted.
	s.audit(ctx, domain.AuditActionUpdate, rec.ID, fmt.Sprintf(
		"Zusammenfassende Meldung %s am %s übermittelt (Transferticket %s)", rec.PeriodKey, date, rec.TransferTicket))
	return rec, nil
}

// CreateCorrection legt eine berichtigte Meldung an (§ 18a Abs. 10 UStG).
func (s *ZMService) CreateCorrection(ctx context.Context, periodKey string) (*domain.ZMReturn, error) {
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
			"für den Zeitraum %s ist keine übermittelte Zusammenfassende Meldung erfasst", period.Key)
	}
	fresh, err := s.build(ctx, period)
	if err != nil {
		return nil, err
	}
	fresh.IsCorrection = true
	fresh.CorrectsID = &submitted.ID
	// Die Berichtigung meldet den Zeitraum vollständig neu; was vorher ein
	// Nachtrag zu *diesem* Zeitraum war, steht jetzt an seinem Platz.
	fresh.LateEntries = make([]domain.ZMLateEntry, 0)
	if err := s.zmRepo.Create(ctx, fresh); err != nil {
		return nil, fmt.Errorf("die berichtigte Meldung konnte nicht gespeichert werden: %w", err)
	}
	s.audit(ctx, domain.AuditActionUpdate, fresh.ID, fmt.Sprintf(
		"Berichtigte Zusammenfassende Meldung %s angelegt (berichtigt Meldung %d)", fresh.PeriodKey, submitted.ID))
	return fresh, nil
}

// ExportCSV liefert die Meldung im Spaltenformat des BZSt-Online-Portals:
// Länderkennzeichen, USt-IdNr., Betrag in vollen Euro, Meldeart.
func (s *ZMService) ExportCSV(ctx context.Context, id uint) (string, error) {
	rec, err := s.zmRepo.FindByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("die Zusammenfassende Meldung %d wurde nicht gefunden: %w", id, err)
	}
	var b strings.Builder
	b.WriteString("Laenderkennzeichen;USt-IdNr.;Betrag;Art\n")
	for _, line := range rec.Lines {
		// Gemeldet wird in vollen Euro; die Cent gehören in die Buchführung,
		// nicht in die Meldung.
		fmt.Fprintf(&b, "%s;%s;%d;%s\n",
			line.CountryCode, strings.TrimPrefix(line.VatID, line.CountryCode),
			int64(accounting.TruncToEuro(line.Amount))/100, line.Kind)
	}
	s.audit(ctx, domain.AuditActionExport, rec.ID, fmt.Sprintf(
		"Zusammenfassende Meldung %s als Datei ausgegeben", rec.PeriodKey))
	return b.String(), nil
}

// -------------------------------------------------------------
// Innenleben
// -------------------------------------------------------------

func (s *ZMService) build(ctx context.Context, period accounting.VatPeriod) (*domain.ZMReturn, error) {
	movements, err := s.movements(ctx, period.Year)
	if err != nil {
		return nil, err
	}
	recipients, err := s.recipients(ctx)
	if err != nil {
		return nil, err
	}
	lookup := func(id uint) accounting.ZMRecipient { return recipients[id] }

	lines, findings := accounting.ZMLines(period, movements, lookup)
	rec := &domain.ZMReturn{
		FiscalYear: period.Year,
		PeriodType: period.Type,
		PeriodKey:  period.Key,
		PeriodFrom: period.From,
		PeriodTo:   period.To,
		Status:     domain.VatReturnDraft,
		DueDate:    accounting.ZMDueDate(period),
		Lines:      lines,
		Findings:   findings,
	}
	for _, line := range lines {
		switch line.Kind {
		case domain.ZMKindService:
			rec.TotalServices += line.Amount
		default:
			rec.TotalSupplies += line.Amount
		}
	}
	rec.LateEntries, err = s.lateEntries(ctx, period, movements)
	if err != nil {
		return nil, err
	}
	rec.Reconciliation = s.reconcile(ctx, period, rec, movements)
	rec.EnsureLists()
	return rec, nil
}

// lateEntries sammelt die Nachträge: meldepflichtige Umsätze, deren
// Meldezeitraum bereits übermittelt ist und die dort nicht gemeldet wurden.
//
// Erkannt wird ein Nachtrag an beidem — übermittelter Zeitraum *und* nicht
// gemeldete Buchung. Ohne die zweite Hälfte wäre jeder gemeldete Umsatz sein
// eigener Nachtrag.
func (s *ZMService) lateEntries(
	ctx context.Context, period accounting.VatPeriod, movements []accounting.ZMMovement,
) ([]domain.ZMLateEntry, error) {
	submitted, reported, err := s.submittedPeriods(ctx, period.Year)
	if err != nil {
		return nil, err
	}
	if len(submitted) == 0 {
		return nil, nil
	}
	recipients, err := s.recipients(ctx)
	if err != nil {
		return nil, err
	}
	// Die Meldezeiträume eines Jahres können verschieden lang sein — die
	// Schwelle des § 18a Abs. 1 Satz 2 UStG schaltet mitten im Jahr auf den
	// Monat um. Der Zeitraum einer Buchung ist deshalb der, der sie enthält,
	// und nicht der aus einer Formel.
	periods := accounting.ZMPeriodsOfYear(period.Year, movements)
	yearFrom := fmt.Sprintf("%d-01-01", period.Year)
	yearTo := fmt.Sprintf("%d-12-31", period.Year)

	var out []domain.ZMLateEntry
	for _, m := range movements {
		if m.Date < yearFrom || m.Date > yearTo {
			continue
		}
		if m.Date >= period.From && m.Date <= period.To {
			continue
		}
		p, ok := periodContaining(periods, m.Date)
		if !ok || !submitted[p.Key] || reported[p.Key][m.EntryID] {
			continue
		}
		out = append(out, domain.ZMLateEntry{
			EntryID:     m.EntryID,
			EntryNumber: m.EntryNumber,
			PeriodKey:   p.Key,
			Date:        m.Date,
			VatID:       recipients[m.ContactID].VatID,
			Kind:        m.Kind,
			Amount:      m.Amount,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].PeriodKey != out[j].PeriodKey {
			return out[i].PeriodKey < out[j].PeriodKey
		}
		return out[i].EntryNumber < out[j].EntryNumber
	})
	return out, nil
}

// submittedPeriods liefert die Meldezeiträume mit bestätigter Übermittlung und
// die Buchungen, die darin gemeldet wurden.
func (s *ZMService) submittedPeriods(ctx context.Context, year int) (map[string]bool, map[string]map[uint]bool, error) {
	saved, err := s.zmRepo.FindByFiscalYear(ctx, year)
	if err != nil {
		return nil, nil, err
	}
	// Je Zeitraum zählt die jüngste bestätigte Meldung: eine Berichtigung tritt
	// an die Stelle der berichtigten und meldet den Zeitraum vollständig neu.
	latest := map[string]*domain.ZMReturn{}
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
		for _, line := range r.Lines {
			for _, id := range line.EntryIDs {
				ids[id] = true
			}
		}
		reported[key] = ids
	}
	return submitted, reported, nil
}

// periodContaining sucht den Meldezeitraum, in den ein Datum fällt.
func periodContaining(periods []accounting.VatPeriod, date string) (accounting.VatPeriod, bool) {
	for _, p := range periods {
		if date >= p.From && date <= p.To {
			return p, true
		}
	}
	return accounting.VatPeriod{}, false
}

// reconcile stellt die Summen der Meldung den Kennziffern 41 und 21 der
// Voranmeldungen desselben Zeitraums gegenüber.
//
// Verglichen wird in vollen Euro, weil beide Meldungen in vollen Euro abgegeben
// werden — ein Cent Unterschied wäre eine Abweichung, die es auf dem Papier
// nicht gibt.
func (s *ZMService) reconcile(
	ctx context.Context, period accounting.VatPeriod, rec *domain.ZMReturn, movements []accounting.ZMMovement,
) *domain.ZMReconciliation {
	out := &domain.ZMReconciliation{
		ScopeKey:   period.Key,
		ScopeLabel: period.Label,
		SuppliesZM: accounting.TruncToEuro(rec.TotalSupplies),
		ServicesZM: accounting.TruncToEuro(rec.TotalServices),
	}
	if s.vatReturnRepo == nil {
		return out
	}
	returns, err := s.vatReturnRepo.FindByFiscalYear(ctx, period.Year)
	if err != nil {
		return out
	}
	// Je Zeitraum zählt die jüngste Anmeldung: eine Berichtigung tritt an die
	// Stelle der berichtigten.
	latest := map[string]*domain.VatReturn{}
	for i := range returns {
		r := &returns[i]
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}

	// Der Regelfall: die Anmeldungen liegen im Meldezeitraum (gleich lang oder
	// kürzer — Monatsanmeldungen in einem ZM-Quartal).
	found := false
	for _, r := range latest {
		if r.PeriodFrom < period.From || r.PeriodTo > period.To {
			continue
		}
		found = true
		out.VatReturnsFound++
		out.SuppliesVat += r.Base(accounting.VatCodeIntraCommunitySupply)
		out.ServicesVat += r.Base(accounting.VatCodeEUServices)
	}
	if found {
		return out
	}

	// Sonst der umgekehrte Fall: die Meldung ist kürzer als die Anmeldung —
	// monatliche ZM neben vierteljährlicher Voranmeldung. Verglichen wird dann
	// auf der Ebene der Anmeldung, gegen die Summe der ZM-Monate des Quartals.
	for _, r := range latest {
		if r.PeriodFrom > period.From || r.PeriodTo < period.To {
			continue
		}
		out.VatReturnsFound = 1
		out.ScopeKey = r.PeriodKey
		out.ScopeLabel = r.PeriodKey
		if p, err := accounting.ParseVatPeriodKey(r.PeriodKey); err == nil {
			out.ScopeLabel = p.Label
		}
		out.SuppliesVat = r.Base(accounting.VatCodeIntraCommunitySupply)
		out.ServicesVat = r.Base(accounting.VatCodeEUServices)

		var supplies, services domain.Cents
		for _, m := range movements {
			if m.Date < r.PeriodFrom || m.Date > r.PeriodTo {
				continue
			}
			if m.Kind == domain.ZMKindService {
				services += m.Amount
			} else {
				supplies += m.Amount
			}
		}
		out.SuppliesZM = accounting.TruncToEuro(supplies)
		out.ServicesZM = accounting.TruncToEuro(services)
		break
	}
	return out
}

// movements liest die meldepflichtigen Umsätze des Jahres und der vier
// vorangegangenen Quartale. Die Rückschau braucht die Schwellenregel des
// § 18a Abs. 1 Satz 2 UStG.
//
// Gelesen wird über drei Geschäftsjahre, ausgewählt wird über das
// Leistungsdatum: das Geschäftsjahr folgt dem Buchungsdatum, der Meldezeitraum
// der Leistung. Die im Januar erfasste Dezemberlieferung und — bei abweichendem
// Wirtschaftsjahr — jeder Kalendermonat aus dem anderen Geschäftsjahr fehlten
// sonst.
func (s *ZMService) movements(ctx context.Context, year int) ([]accounting.ZMMovement, error) {
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("die Buchungen des Jahres %d konnten nicht gelesen werden: %w", year, err)
	}
	for _, y := range []int{year - 1, year + 1} {
		if more, err := s.journalRepo.FindAll(ctx, y); err == nil {
			entries = append(entries, more...)
		}
	}
	recipients, err := s.recipients(ctx)
	if err != nil {
		return nil, err
	}
	return accounting.ZMMovements(accounting.ZMSource{
		Entries:   entries,
		Recipient: func(id uint) accounting.ZMRecipient { return recipients[id] },
	}), nil
}

func (s *ZMService) recipients(ctx context.Context) (map[uint]accounting.ZMRecipient, error) {
	out := map[uint]accounting.ZMRecipient{}
	if s.contactRepo == nil {
		return out, nil
	}
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return out, fmt.Errorf("die Geschäftspartner konnten nicht gelesen werden: %w", err)
	}
	for i := range contacts {
		c := &contacts[i]
		out[c.ID] = accounting.ZMRecipient{
			Name:        c.Name,
			CountryCode: c.CountryCode,
			VatID:       c.VatID,
			IsEU:        c.IsEUCounterparty(),
		}
	}
	return out, nil
}

// ensureCommitted verlangt die Festschreibung des Meldezeitraums vor der
// Bestätigung — dieselbe Bedingung wie bei der Voranmeldung.
//
// Ohne sie könnte sich eine bestätigte Meldung nach der Bestätigung durch
// weitere Buchungen im Zeitraum vom Journal entfernen, und nichts wiese darauf
// hin: die Meldung ist ihr eigenes Protokoll.
func (s *ZMService) ensureCommitted(ctx context.Context, rec *domain.ZMReturn) error {
	if s.festschreibungRepo == nil {
		return nil
	}
	cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, rec.FiscalYear)
	if err != nil {
		return fmt.Errorf("der Festschreibungsstand konnte nicht geprüft werden: %w", err)
	}
	if cutoff == "" || cutoff < rec.PeriodTo {
		return fmt.Errorf(
			"der Meldezeitraum %s (bis %s) ist noch nicht festgeschrieben. Eine übermittelte Meldung muss "+
				"sich auf einen unveränderlichen Stand stützen — schreibe den Zeitraum zuerst fest",
			rec.PeriodKey, rec.PeriodTo)
	}
	return nil
}

// ensureFirstOrCorrection lässt je Meldezeitraum nur eine Erstmeldung zu; jede
// weitere ist eine Berichtigung.
func (s *ZMService) ensureFirstOrCorrection(ctx context.Context, rec *domain.ZMReturn) error {
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
		"für den Zeitraum %s ist am %s bereits eine Zusammenfassende Meldung übermittelt (Transferticket %s). "+
			"Eine zweite Erstmeldung desselben Zeitraums gibt es nicht — bestätige sie als berichtigte Meldung",
		rec.PeriodKey, submitted.SubmittedAt, submitted.TransferTicket)
}

func (s *ZMService) openDraft(ctx context.Context, periodKey string) (*domain.ZMReturn, error) {
	saved, err := s.zmRepo.FindByPeriod(ctx, periodKey)
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

func (s *ZMService) latestSubmitted(ctx context.Context, periodKey string) (*domain.ZMReturn, error) {
	saved, err := s.zmRepo.FindByPeriod(ctx, periodKey)
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

func (s *ZMService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "ZM_RETURN", fmt.Sprintf("%d", id), details)
}
