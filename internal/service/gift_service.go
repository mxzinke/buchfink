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

// GiftService führt die nicht abziehbaren Betriebsausgaben (§ 4 Abs. 5 EStG).
//
// Er liest die Aufzeichnungen aus dem Journal und schreibt nichts als
// Umbuchungen. Das ist Absicht: die Aufzeichnung zum Geschenk ist Teil der
// Buchung und von der Hashkette gedeckt, und eine Kartei daneben, die dasselbe
// noch einmal führte, wäre die zweite Wahrheit, die beim ersten Storno von der
// ersten abweicht.
type GiftService struct {
	journalRepo  domain.JournalRepository
	journalSvc   *JournalService
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	fiscalYear   int
	// txRunner klammert Storno und Neubuchung einer Umbuchung. Fehlt er, laufen
	// beide nacheinander — die Vorprüfung darunter hält den halben Zustand auch
	// dann fern, sie kann ihn nur nicht mehr zurückrollen.
	txRunner domain.TxRunner
}

// NewGiftService wires die Auswertung der nicht abziehbaren Betriebsausgaben.
func NewGiftService(
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *GiftService {
	return &GiftService{
		journalRepo: journalRepo, journalSvc: journalSvc,
		settingsRepo: settingsRepo, auditRepo: auditRepo, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *GiftService) SetFiscalYear(year int) { s.fiscalYear = year }

// SetTxRunner koppelt die Transaktionsklammer an die Umbuchung.
func (s *GiftService) SetTxRunner(r domain.TxRunner) { s.txRunner = r }

// runInTx führt fn in einer Transaktion aus, wo eine eingerichtet ist.
func (s *GiftService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.RunInTx(ctx, fn)
}

// giftEntry ist eine Aufzeichnung samt ihrer Buchung und der Frage, ob die
// Buchung noch steht.
type giftEntry struct {
	entry    *domain.JournalEntry
	record   domain.GiftRecord
	reversed bool
}

// GiftsInYear liefert die Geschenke eines Wirtschaftsjahres, die noch stehen.
//
// Stornierte Buchungen fallen heraus: eine Generalumkehr nimmt den Vorgang
// zurück, und ein zurückgenommenes Geschenk zählt nicht gegen die Freigrenze.
// Ohne diesen Filter hielte eine korrigierte Fehlbuchung den Empfänger dauerhaft
// über der Grenze.
func (s *GiftService) GiftsInYear(ctx context.Context, fiscalYear int) ([]domain.GiftRecord, error) {
	entries, err := s.giftEntries(ctx, fiscalYear)
	if err != nil {
		return nil, err
	}
	out := make([]domain.GiftRecord, 0, len(entries))
	for _, e := range entries {
		if e.reversed {
			continue
		}
		out = append(out, e.record)
	}
	return out, nil
}

// giftEntries sammelt die Aufzeichnungen eines Jahres mit ihren Buchungen.
func (s *GiftService) giftEntries(ctx context.Context, fiscalYear int) ([]giftEntry, error) {
	if s.journalRepo == nil {
		return nil, fmt.Errorf("das Journal ist nicht eingerichtet")
	}
	entries, err := s.journalRepo.FindAll(ctx, fiscalYear)
	if err != nil {
		return nil, err
	}
	// Welche Buchung eine Generalumkehr zurückgenommen hat, steht an der
	// Generalumkehr. Einmal eingesammelt statt je Geschenk einzeln gefragt.
	reversed := map[uint]bool{}
	for i := range entries {
		if entries[i].Kind == domain.EntryKindReversal && entries[i].ReversalOfID != nil {
			reversed[*entries[i].ReversalOfID] = true
		}
	}

	out := make([]giftEntry, 0, 8)
	for i := range entries {
		entry := &entries[i]
		// Die Generalumkehr trägt die Aufzeichnung mit — sie korrigiert die
		// Buchung, nicht das Geschenk —, gezählt wird sie aber nicht: sonst stünde
		// jedes stornierte Geschenk zweimal in der Kartei, einmal mit und einmal
		// ohne Betrag, und die Freigrenze liefe über beide.
		if entry.Kind == domain.EntryKindReversal {
			continue
		}
		for _, record := range entry.Gifts {
			out = append(out, giftEntry{entry: entry, record: record, reversed: reversed[entry.ID]})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].record.Date != out[j].record.Date {
			return out[i].record.Date < out[j].record.Date
		}
		return out[i].entry.EntryNumber < out[j].entry.EntryNumber
	})
	return out, nil
}

// -------------------------------------------------------------------------
// Bericht über die nicht abziehbaren Betriebsausgaben
// -------------------------------------------------------------------------

// NonDeductibleCategoryRow ist eine Kategorie im Bericht.
type NonDeductibleCategoryRow struct {
	accounting.NonDeductibleCategory
	Deductible    domain.Cents `json:"deductibleAmount"`
	NonDeductible domain.Cents `json:"nonDeductibleAmount"`
	Total         domain.Cents `json:"total"`
	Count         int          `json:"count"`
}

// GiftRecipientRow ist ein Empfänger im Bericht.
type GiftRecipientRow struct {
	RecipientKey  string       `json:"recipientKey"`
	RecipientName string       `json:"recipientName"`
	ContactID     uint         `json:"contactId,omitempty"`
	Total         domain.Cents `json:"total"`
	// OverLimit sagt, ob die Freigrenze gerissen ist.
	OverLimit bool `json:"overLimit"`
	// ToRebook sind die Buchungen, die noch abziehbar stehen, obwohl der
	// Empfänger die Freigrenze überschritten hat.
	ToRebook []GiftBookingRow `json:"toRebook"`
	Bookings []GiftBookingRow `json:"bookings"`
	Note     string           `json:"note"`
}

// GiftBookingRow ist eine einzelne Geschenkbuchung.
type GiftBookingRow struct {
	EntryID     uint         `json:"entryId"`
	EntryNumber string       `json:"entryNumber"`
	Date        string       `json:"date"`
	NetAmount   domain.Cents `json:"netAmount"`
	Account     string       `json:"account"`
	Deductible  bool         `json:"deductible"`
	Occasion    string       `json:"occasion,omitempty"`
}

// NonDeductibleReport ist der Bericht eines Geschäftsjahres.
type NonDeductibleReport struct {
	FiscalYear int                        `json:"fiscalYear"`
	Limit      domain.Cents               `json:"giftLimit"`
	Categories []NonDeductibleCategoryRow `json:"categories"`
	Recipients []GiftRecipientRow         `json:"recipients"`
	Note       string                     `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *NonDeductibleReport) EnsureLists() {
	if r.Categories == nil {
		r.Categories = make([]NonDeductibleCategoryRow, 0)
	}
	if r.Recipients == nil {
		r.Recipients = make([]GiftRecipientRow, 0)
	}
}

// NonDeductibleReport stellt die beschränkt abziehbaren Betriebsausgaben eines
// Jahres zusammen.
//
// Zwei Sichten, weil zwei Fragen gestellt werden: die Kategorien beantworten
// „wie viel davon ist steuerlich abziehbar" — die Zahl, die in die Überleitung
// zur Steuerbilanz geht. Die Empfänger beantworten „wo ist die Freigrenze
// gerissen" — die Frage, aus der eine Umbuchung folgt.
func (s *GiftService) NonDeductibleReport(ctx context.Context, year int) (*NonDeductibleReport, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	out := &NonDeductibleReport{FiscalYear: year}
	out.EnsureLists()

	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return nil, err
	}

	// 1. Die Kategorien: gezählt wird über die Konten, nicht über die
	// Aufzeichnungen. Eine Bewirtung trägt keine Geschenkaufzeichnung, und eine
	// Buchung aus der Zeit vor dieser Welle trägt gar keine — auf den Konten
	// steht sie trotzdem.
	//
	// Die Generalumkehr wird mitgezählt und nicht übergangen: sie trägt
	// negative Beträge auf derselben Seite und stellt die Verkehrszahlen des
	// Kontos damit auf null. Sie zu überspringen ließe eine zurückgenommene
	// Buchung im Bericht stehen — und genau das passiert nach einer Umbuchung.
	sums := map[string]domain.Cents{}
	counts := map[string]int{}
	for i := range entries {
		for _, line := range entries[i].Lines {
			if line.Side != domain.SideDebit {
				continue
			}
			sums[line.Account] += line.Amount
			if line.Amount > 0 {
				counts[line.Account]++
			}
		}
	}
	for _, category := range accounting.NonDeductibleCategories() {
		row := NonDeductibleCategoryRow{NonDeductibleCategory: category}
		if category.DeductibleAccount != "" {
			row.Deductible = sums[category.DeductibleAccount]
			row.Count += counts[category.DeductibleAccount]
		}
		if category.NonDeductibleAccount != "" {
			row.NonDeductible = sums[category.NonDeductibleAccount]
			row.Count += counts[category.NonDeductibleAccount]
		}
		row.Total = row.Deductible + row.NonDeductible
		out.Categories = append(out.Categories, row)
	}

	// 2. Die Empfänger und ihre Freigrenze.
	gifts, err := s.giftEntries(ctx, year)
	if err != nil {
		return nil, err
	}
	limit, err := s.giftLimit(ctx, year)
	if err != nil {
		return nil, err
	}
	out.Limit = limit

	byRecipient := map[string]*GiftRecipientRow{}
	order := make([]string, 0, len(gifts))
	for i := range gifts {
		g := &gifts[i]
		if g.reversed {
			continue
		}
		key := g.record.RecipientKey()
		row, ok := byRecipient[key]
		if !ok {
			row = &GiftRecipientRow{
				RecipientKey:  key,
				RecipientName: g.record.RecipientName,
				Bookings:      make([]GiftBookingRow, 0, 2),
				ToRebook:      make([]GiftBookingRow, 0),
			}
			if g.record.RecipientContactID != nil {
				row.ContactID = *g.record.RecipientContactID
			}
			byRecipient[key] = row
			order = append(order, key)
		}
		row.Total += g.record.NetAmount
		row.Bookings = append(row.Bookings, GiftBookingRow{
			EntryID: g.entry.ID, EntryNumber: g.entry.EntryNumber, Date: g.record.Date,
			NetAmount: g.record.NetAmount, Account: g.record.Account,
			Deductible: g.record.Deductible(), Occasion: g.record.Occasion,
		})
	}

	for _, key := range order {
		row := byRecipient[key]
		row.OverLimit = row.Total > limit
		switch {
		case !row.OverLimit:
			row.Note = fmt.Sprintf(
				"%s € von %s € der Freigrenze ausgeschöpft.", row.Total, limit)
		default:
			for _, b := range row.Bookings {
				if b.Deductible {
					row.ToRebook = append(row.ToRebook, b)
				}
			}
			if len(row.ToRebook) == 0 {
				row.Note = fmt.Sprintf(
					"Die Freigrenze von %s € ist mit %s € überschritten. Alle Geschenke an diesen "+
						"Empfänger stehen bereits auf dem nicht abziehbaren Konto.", limit, row.Total)
			} else {
				row.Note = fmt.Sprintf(
					"Die Freigrenze von %s € ist mit %s € überschritten. Damit sind sämtliche "+
						"Geschenke an diesen Empfänger nicht abziehbar (§ 4 Abs. 5 Satz 1 Nr. 1 EStG) "+
						"und der Vorsteuerabzug entfällt (§ 15 Abs. 1a UStG). %d Buchung(en) stehen "+
						"noch als abziehbar und sind umzubuchen.",
					limit, row.Total, len(row.ToRebook))
			}
		}
		out.Recipients = append(out.Recipients, *row)
	}
	sort.SliceStable(out.Recipients, func(i, j int) bool {
		if out.Recipients[i].OverLimit != out.Recipients[j].OverLimit {
			return out.Recipients[i].OverLimit
		}
		return out.Recipients[i].RecipientName < out.Recipients[j].RecipientName
	})

	out.Note = fmt.Sprintf(
		"Die Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG beträgt im Wirtschaftsjahr %d %s € netto je "+
			"Empfänger. Sie ist eine Freigrenze und kein Freibetrag: mit dem ersten Cent darüber ist "+
			"nicht der übersteigende Teil nicht abziehbar, sondern der gesamte Betrag.",
		year, limit)
	return out, nil
}

// giftLimit liefert die Freigrenze des Wirtschaftsjahres.
func (s *GiftService) giftLimit(ctx context.Context, year int) (domain.Cents, error) {
	// Am ersten Tag des Wirtschaftsjahres gemessen: die Grenze gilt für
	// Wirtschaftsjahre, die nach einem Stichtag beginnen.
	month := 1
	if s.settingsRepo != nil {
		if settings, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && settings != nil {
			if settings.FiscalYearStartMonth >= 1 && settings.FiscalYearStartMonth <= 12 {
				month = settings.FiscalYearStartMonth
			}
		}
	}
	params, err := accounting.TaxParametersFor(fmt.Sprintf("%04d-%02d-01", year, month))
	if err != nil {
		return 0, err
	}
	return params.GiftDeductibleLimit, nil
}

// -------------------------------------------------------------------------
// Umbuchung
// -------------------------------------------------------------------------

// RebookGiftsRequest bucht die noch abziehbar stehenden Geschenke eines
// Empfängers auf das nicht abziehbare Konto um.
type RebookGiftsRequest struct {
	FiscalYear   int    `json:"fiscalYear"`
	RecipientKey string `json:"recipientKey"`
	// Date ist der Tag der Umbuchung. Leer heißt: der Tag der Korrektur, und
	// ein anderer als dieser wird abgewiesen — Storno und Neubuchung gehören in
	// denselben Zeitraum.
	Date string `json:"date,omitempty"`
	// Reason ist Pflicht: eine Umbuchung ohne Grund ist im Journal später nicht
	// mehr erklärbar.
	Reason string `json:"reason"`
}

// GiftRebooking ist das Ergebnis einer Umbuchung.
type GiftRebooking struct {
	RecipientName string   `json:"recipientName"`
	Reversals     []string `json:"reversals"`
	Rebookings    []string `json:"rebookings"`
	Note          string   `json:"note"`
}

// RebookGiftsForRecipient nimmt die abziehbar gebuchten Geschenke eines
// Empfängers zurück und bucht sie ohne Vorsteuerabzug neu.
//
// Zwei Buchungen je Geschenk und keine „Korrektur": die ursprüngliche Buchung
// wird durch eine Generalumkehr zurückgenommen, und die neue steht daneben. Eine
// nachträglich veränderte Buchung wäre die eine Sache, die § 146 Abs. 4 AO
// ausschließt — und die Hashkette würde es merken.
func (s *GiftService) RebookGiftsForRecipient(
	ctx context.Context, req RebookGiftsRequest,
) (*GiftRebooking, error) {
	if s.journalSvc == nil {
		return nil, fmt.Errorf("der Buchungsweg ist nicht eingerichtet")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf(
			"zur Umbuchung gehört ihr Grund. Er steht später als einzige Erklärung im Journal")
	}
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}

	gifts, err := s.giftEntries(ctx, year)
	if err != nil {
		return nil, err
	}
	limit, err := s.giftLimit(ctx, year)
	if err != nil {
		return nil, err
	}

	var total domain.Cents
	name := ""
	pending := make([]giftRebookingPlan, 0, 2)
	// Je Buchung ein Vorgang und nicht je Aufzeichnung: die Generalumkehr nimmt
	// die ganze Buchung zurück. Zwei Geschenke an denselben Empfänger auf einem
	// Beleg wären sonst zwei Stornos derselben Buchung — der zweite liefe ins
	// Leere, und die Umbuchung bräche mitten in der Liste ab.
	at := map[uint]int{}
	for _, g := range gifts {
		if g.reversed || g.record.RecipientKey() != req.RecipientKey {
			continue
		}
		total += g.record.NetAmount
		name = g.record.RecipientName
		if !g.record.Deductible() {
			continue
		}
		if i, ok := at[g.entry.ID]; ok {
			pending[i].records = append(pending[i].records, g.record)
			continue
		}
		at[g.entry.ID] = len(pending)
		pending = append(pending, giftRebookingPlan{
			entry: g.entry, records: []domain.GiftRecord{g.record}})
	}
	if name == "" {
		return nil, fmt.Errorf("zu %q sind im Geschäftsjahr %d keine Geschenke erfasst",
			req.RecipientKey, year)
	}
	if total <= limit {
		return nil, fmt.Errorf(
			"an %s sind im Geschäftsjahr %d %s € verschenkt worden; die Freigrenze liegt bei %s €. "+
				"Solange sie nicht überschritten ist, bleiben die Geschenke abziehbar — es gibt nichts "+
				"umzubuchen", name, year, total, limit)
	}
	if len(pending) == 0 {
		return nil, fmt.Errorf(
			"an %s steht kein Geschenk mehr als abziehbar; es ist bereits alles umgebucht", name)
	}

	// Storno und Neubuchung tragen denselben Tag, und der ist der Tag der
	// Korrektur.
	//
	// Vorher datierte die Generalumkehr auf heute und die Neubuchung auf das
	// Datum der ursprünglichen Buchung. Lag das in einer festgeschriebenen
	// Periode oder einem festgestellten Jahr, ging die Umkehr durch und die
	// Neubuchung nicht — der Aufwand war danach ganz aus den Büchern, und die
	// Schleife brach mitten in der Liste ab.
	//
	// Ein anderer Tag als der der Korrektur steht nicht zur Wahl, und das ist
	// keine Einschränkung dieses Bausteins: die Generalumkehr trägt den Tag
	// ihrer Erstellung (JournalService.ReverseOn lässt ein vorgegebenes Datum
	// nur für den Saldenvortrag zu), und eine Neubuchung an einem anderen Tag
	// wäre die Hälfte des Vorgangs in einem anderen Zeitraum — mit allem, was
	// daran hängt: Voranmeldung, Festschreibung, Abschlussstand.
	today := time.Now().Format("2006-01-02")
	date := strings.TrimSpace(req.Date)
	if date == "" {
		date = today
	}
	if date != today {
		return nil, fmt.Errorf(
			"die Umbuchung wird zum %s gebucht und nicht zum %s. Sie besteht aus einer Generalumkehr "+
				"und einer Neubuchung, und die Generalumkehr trägt immer den Tag ihrer Erstellung "+
				"(§ 239 Abs. 3 HGB); ein eigenes Datum für die Neubuchung risse den Vorgang in zwei "+
				"Zeiträume", germanDay(today), germanDay(date))
	}

	// Und geprüft wird alles, bevor das erste Mal geschrieben wird: eine
	// Umbuchung, die zur Hälfte gelingt, ist schlechter als eine, die gar nicht
	// beginnt.
	planned := make([]*domain.JournalEntry, 0, len(pending))
	for _, g := range pending {
		entry, err := s.buildRebooking(g, req, date)
		if err != nil {
			return nil, err
		}
		if err := s.journalSvc.ValidatePostable(ctx, entry); err != nil {
			return nil, fmt.Errorf(
				"die Neubuchung zu %s ließe sich zum %s nicht schreiben; die Umbuchung wird deshalb "+
					"gar nicht erst begonnen: %w", g.entry.EntryNumber, germanDay(date), err)
		}
		planned = append(planned, entry)
	}

	out := &GiftRebooking{
		RecipientName: name,
		Reversals:     make([]string, 0, len(pending)),
		Rebookings:    make([]string, 0, len(pending)),
	}
	err = s.runInTx(ctx, func(ctx context.Context) error {
		out.Reversals = out.Reversals[:0]
		out.Rebookings = out.Rebookings[:0]
		for i, g := range pending {
			reversal, err := s.journalSvc.Reverse(ctx, g.entry.ID, fmt.Sprintf(
				"Freigrenze für Geschenke an %s überschritten: %s", name, req.Reason))
			if err != nil {
				return fmt.Errorf(
					"die Buchung %s ließ sich nicht zurücknehmen: %w", g.entry.EntryNumber, err)
			}
			// Gebucht wird auf den Tag der Generalumkehr und nicht auf den
			// vorgeprüften: über Mitternacht sind das zwei verschiedene, und die
			// beiden Hälften des Vorgangs gehören zusammen.
			planned[i].BookingDate = reversal.BookingDate
			rebooked, err := s.journalSvc.Post(ctx, planned[i])
			if err != nil {
				return fmt.Errorf(
					"die Neubuchung zu %s ließ sich nicht schreiben: %w", g.entry.EntryNumber, err)
			}
			out.Reversals = append(out.Reversals, reversal.EntryNumber)
			out.Rebookings = append(out.Rebookings, rebooked.EntryNumber)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out.Note = fmt.Sprintf(
		"An %s sind im Geschäftsjahr %d %s € verschenkt worden und damit mehr als die Freigrenze von "+
			"%s €. %d Buchung(en) wurden zurückgenommen und ohne Vorsteuerabzug neu gebucht "+
			"(§ 4 Abs. 5 Satz 1 Nr. 1 EStG, § 15 Abs. 1a UStG).",
		name, year, total, limit, len(pending))
	s.audit(ctx, fmt.Sprintf("Geschenke an %s umgebucht: %s", name, out.Note))
	return out, nil
}

// giftRebookingPlan ist eine Buchung mit den Geschenken eines Empfängers, die
// aus ihr umzubuchen sind.
type giftRebookingPlan struct {
	entry   *domain.JournalEntry
	records []domain.GiftRecord
}

// buildRebooking stellt die Neubuchung einer Geschenkbuchung zusammen, ohne sie
// zu schreiben.
//
// Die Neubuchung ist die ursprüngliche Buchung mit genau einer Änderung: die
// Zeilen der betroffenen Geschenke stehen auf dem nicht abziehbaren Konto und
// tragen die Vorsteuer, die auf sie entfällt, im Aufwand (§ 15 Abs. 1a UStG,
// § 9b Abs. 1 EStG). Alles andere bleibt, wie es gebucht war.
//
// Das ist der Unterschied zur ersten Fassung, die alle Sollzeilen der Buchung
// auf das nicht abziehbare Konto zusammenzog: ein Eingangsbeleg über ein
// Geschenk (40 €) und Bürobedarf (100 €) verlor damit den Abzug für den
// Bürobedarf, und die Aufzeichnungen der übrigen Empfänger desselben Belegs
// fielen mit dem Storno aus der Kartei — „zehn Präsentkörbe an zehn Empfänger
// sind ein Beleg und eine Buchung, aber zehn Aufzeichnungen".
//
// Wo sich die Zeile eines Geschenks nicht eindeutig zuordnen lässt, wird die
// Umbuchung abgewiesen statt geraten: eine falsch zusammengezogene Buchung ist
// schlechter als eine, die von Hand gemacht werden muss.
func (s *GiftService) buildRebooking(
	plan giftRebookingPlan, req RebookGiftsRequest, date string,
) (*domain.JournalEntry, error) {
	entry := plan.entry

	// 1. Die Vorsteuerzeile der Buchung. Aus ihr kommt der Anteil, der auf die
	// Geschenke entfällt; sie ist die einzige Stelle, an der Buchfink den
	// Steuersatz einer Zeile noch findet (Bemessungsgrundlage und Betrag).
	taxIdx := -1
	for i, l := range entry.Lines {
		if l.Side != domain.SideDebit {
			continue
		}
		if reverseChargeInputTaxKey(l.TaxKey) {
			// Beim innergemeinschaftlichen Erwerb und beim Reverse Charge stehen
			// Steuerschuld und Vorsteuer aus derselben Bemessungsgrundlage
			// nebeneinander. Die Vorsteuer zu kürzen, ohne die Steuerschuld zu
			// berühren, risse die Buchung auseinander — dieselbe Grenze, an der
			// schon der Belegweg den Ausschluss des § 15 Abs. 1a UStG abweist.
			return nil, fmt.Errorf(
				"die Buchung %s ist ein innergemeinschaftlicher Erwerb oder eine Leistung nach "+
					"§ 13b UStG. Dort steht die Vorsteuer neben der Steuerschuld aus derselben "+
					"Bemessungsgrundlage, und § 15 Abs. 1a UStG nimmt nur den Abzug — diese "+
					"Aufteilung rechnet Buchfink nicht. Buche den Beleg von Hand um",
				entry.EntryNumber)
		}
		if !deductibleInputTaxKey(l.TaxKey) {
			continue
		}
		if taxIdx >= 0 {
			return nil, fmt.Errorf(
				"die Buchung %s trägt mehrere Vorsteuerzeilen (verschiedene Steuersätze). Welcher "+
					"Anteil auf das Geschenk entfällt, ist ihr nicht mehr zu entnehmen — buche diesen "+
					"Beleg von Hand um", entry.EntryNumber)
		}
		taxIdx = i
	}

	// 2. Zu jeder Aufzeichnung ihre Aufwandszeile und die Vorsteuer darauf.
	matched := make(map[int]domain.GiftRecord, len(plan.records))
	taxOf := make(map[int]domain.Cents, len(plan.records))
	accountOf := make(map[int]string, len(plan.records))
	order := make([]int, 0, len(plan.records))
	flip := make(map[uint]string, len(plan.records))
	var giftNet, giftTax domain.Cents
	for _, record := range plan.records {
		category, ok := accounting.NonDeductibleCategoryForAccount(record.Account)
		if !ok || category.NonDeductibleAccount == "" {
			return nil, fmt.Errorf(
				"zu Konto %s ist keine nicht abziehbare Gegenposition hinterlegt", record.Account)
		}
		idx := -1
		for i, l := range entry.Lines {
			if _, taken := matched[i]; taken {
				continue
			}
			if l.Side == domain.SideDebit && l.Account == record.Account && l.Amount == record.NetAmount {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, fmt.Errorf(
				"in der Buchung %s ist die Zeile zum Geschenk an %s (%s € auf Konto %s) nicht "+
					"eindeutig zu finden. Eine Umbuchung nähme sonst fremden Aufwand mit — buche "+
					"diesen Beleg von Hand um",
				entry.EntryNumber, record.RecipientName, record.NetAmount, record.Account)
		}
		matched[idx] = record
		accountOf[idx] = category.NonDeductibleAccount
		order = append(order, idx)
		flip[record.ID] = category.NonDeductibleAccount
		giftNet += record.NetAmount

		if taxIdx >= 0 {
			tax := entry.Lines[taxIdx]
			if tax.TaxBase <= 0 || record.NetAmount > tax.TaxBase {
				return nil, fmt.Errorf(
					"die Vorsteuerzeile der Buchung %s trägt keine Bemessungsgrundlage, aus der sich "+
						"der Anteil des Geschenks rechnen ließe — buche diesen Beleg von Hand um",
					entry.EntryNumber)
			}
			share := domain.MulRound(tax.Amount, int64(record.NetAmount), int64(tax.TaxBase))
			taxOf[idx] = share
			giftTax += share
		}
	}
	if len(order) == 0 {
		return nil, fmt.Errorf(
			"die Buchung %s trägt keine Aufzeichnung, die sich umbuchen ließe", entry.EntryNumber)
	}
	// Der Rundungsrest landet auf der letzten Zeile. Tragen die Geschenke die
	// ganze Bemessungsgrundlage, gehört ihnen die ganze Vorsteuerzeile — sonst
	// bliebe sie mit einem Cent stehen und behauptete einen Abzug, den es nicht
	// mehr gibt.
	if taxIdx >= 0 {
		tax := entry.Lines[taxIdx]
		last := order[len(order)-1]
		if giftNet == tax.TaxBase || giftTax > tax.Amount {
			taxOf[last] += tax.Amount - giftTax
			giftTax = tax.Amount
		}
	}

	lines := make([]domain.JournalLine, 0, len(entry.Lines))
	for i, l := range entry.Lines {
		l.ID, l.EntryID, l.Position = 0, 0, 0
		if record, ok := matched[i]; ok {
			l.Account = accountOf[i]
			l.Amount = record.NetAmount + taxOf[i]
			l.InputTaxShare = domain.InputTaxExcluded
			l.Text = fmt.Sprintf("Geschenk an %s, nicht abziehbar", record.RecipientName)
			lines = append(lines, l)
			continue
		}
		if i == taxIdx {
			l.Amount -= giftTax
			l.TaxBase -= giftNet
			// Bleibt nichts übrig, fällt die Zeile weg: eine Vorsteuerzeile über
			// null Euro stünde in der Voranmeldung als Abzug ohne Betrag.
			if l.Amount <= 0 {
				continue
			}
		}
		lines = append(lines, l)
	}

	// Die Aufzeichnungen der Buchung gehen vollständig mit — auch die der
	// anderen Empfänger. Das Original ist nach dem Storno zurückgenommen; was
	// hier fehlte, fehlte danach in der Kartei.
	records := make([]domain.GiftRecord, 0, len(entry.Gifts))
	for _, r := range entry.Gifts {
		if account, ok := flip[r.ID]; ok {
			r.NonDeductible = true
			r.Account = account
		}
		r.ID, r.EntryID = 0, 0
		records = append(records, r)
	}

	return &domain.JournalEntry{
		BookingDate: date, DocumentDate: entry.DocumentDate,
		ServiceDateFrom: entry.ServiceDateFrom, ServiceDateTo: entry.ServiceDateTo,
		Description: fmt.Sprintf("Geschenk an %s ohne Abzug (Freigrenze überschritten): %s",
			plan.records[0].RecipientName, req.Reason),
		Source:             domain.EntrySourceManual,
		DocumentNumber:     entry.DocumentNumber,
		ReceiptID:          entry.ReceiptID,
		ReceiptHash:        entry.ReceiptHash,
		TaxTreatment:       entry.TaxTreatment,
		ContactID:          entry.ContactID,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
		Gifts:              records,
	}, nil
}

// deductibleInputTaxKey meldet, ob ein Steuerschlüssel eine gezogene Vorsteuer
// trägt. Die Berichtigung nach § 15a UStG hat einen eigenen Schlüssel und ist
// keine Zeile eines Belegs; sie bleibt außen vor.
func deductibleInputTaxKey(key string) bool {
	return key == "VST19" || key == "VST7"
}

// reverseChargeInputTaxKey meldet die Vorsteuer, die neben einer Steuerschuld
// aus derselben Bemessungsgrundlage steht.
func reverseChargeInputTaxKey(key string) bool {
	switch key {
	case "IG19_VST", "IG7_VST", "RC19_VST", "RC7_VST":
		return true
	}
	return false
}

func (s *GiftService) audit(ctx context.Context, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "GIFT", "", details)
}
