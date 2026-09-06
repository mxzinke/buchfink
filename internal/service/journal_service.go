package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
)

// JournalService is the single write path into the journal.
//
// Every booking in the system goes through Post, which enforces the rules that
// make the journal usable as accounting evidence: balanced entries, existing and
// bookable accounts, tax accounts reserved for the tax automation, gapless
// numbering, an unbroken hash chain and respect for committed periods.
type JournalService struct {
	journalRepo        domain.JournalRepository
	accountRepo        domain.AccountRepository
	contactRepo        domain.ContactRepository
	auditRepo          domain.AuditRepository
	settingsRepo       domain.SettingsRepository
	festschreibungRepo domain.FestschreibungRepository
	fiscalYearRepo     domain.FiscalYearRepository
	// receiptRepo ist der Beleg, auf den eine Buchung verweist. Optional:
	// ohne ihn bucht der Dienst wie zuvor, nur ohne die Kopfdatenprüfung.
	receiptRepo domain.ReceiptRepository

	hashChain   *accounting.HashChain
	taxResolver domain.TaxResolver

	chart      *accounting.Chart
	fiscalYear int
}

// NewJournalService wires the journal write path.
func NewJournalService(
	journalRepo domain.JournalRepository,
	accountRepo domain.AccountRepository,
	contactRepo domain.ContactRepository,
	auditRepo domain.AuditRepository,
	settingsRepo domain.SettingsRepository,
	fiscalYear int,
) *JournalService {
	return &JournalService{
		journalRepo:  journalRepo,
		accountRepo:  accountRepo,
		contactRepo:  contactRepo,
		auditRepo:    auditRepo,
		settingsRepo: settingsRepo,
		hashChain:    accounting.NewHashChain(),
		taxResolver:  accounting.NewSKR04TaxResolver(),
		fiscalYear:   fiscalYear,
	}
}

// SetFestschreibungRepo wires period-commitment enforcement. Optional.
func (s *JournalService) SetFestschreibungRepo(r domain.FestschreibungRepository) {
	s.festschreibungRepo = r
}

// SetFiscalYearRepo wires the Abschlussstand of the fiscal years. Optional.
func (s *JournalService) SetFiscalYearRepo(r domain.FiscalYearRepository) { s.fiscalYearRepo = r }

// SetReceiptRepo hängt die Belegablage an. Mit ihr prüft Post die Kopfdaten des
// Belegs, auf den eine Buchung verweist. Optional.
func (s *JournalService) SetReceiptRepo(r domain.ReceiptRepository) { s.receiptRepo = r }

// SetFiscalYear updates the active fiscal year filter.
func (s *JournalService) SetFiscalYear(year int) { s.fiscalYear = year }

// FiscalYear returns the active fiscal year.
func (s *JournalService) FiscalYear() int { return s.fiscalYear }

// Chart returns the cached chart of accounts resolver.
func (s *JournalService) Chart(ctx context.Context) (*accounting.Chart, error) {
	if s.chart != nil {
		return s.chart, nil
	}
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Kontenplan konnte nicht geladen werden: %w", err)
	}
	s.chart = accounting.NewChart(accounts)
	return s.chart, nil
}

// InvalidateChart verwirft den zwischengespeicherten Kontenplan.
//
// Der Cache hält den Kontenplan für die Laufzeit des Programms fest, weil jede
// Buchung ihn braucht. Er ist damit aber auch der Grund, warum ein neu
// angelegtes oder gesperrtes eigenes Konto (BEL-06 K2) im laufenden Betrieb
// nicht ankäme: die Buchung auf das neue Konto scheiterte als „im SKR04 nicht
// vorhanden", die Sperre eines Kontos griffe erst nach dem Neustart. Wer den
// Kontenplan ändert, sagt es deshalb hier.
func (s *JournalService) InvalidateChart() { s.chart = nil }

// TaxResolver exposes the SKR04 tax resolution used by the posting rules.
func (s *JournalService) TaxResolver() domain.TaxResolver { return s.taxResolver }

// Post validates and appends a journal entry.
func (s *JournalService) Post(ctx context.Context, entry *domain.JournalEntry) (*domain.JournalEntry, error) {
	s.applyDefaults(ctx, entry)

	// Die Generalumkehr läuft durch: eine Regel über künftige Buchungen darf
	// nicht die Korrektur vorhandener verhindern. Wer einen Mandanten auf
	// Istversteuerung stehen hat, muss seine falschen Buchungen stornieren
	// können — sonst schließt die Prüfung ihn in dem Zustand ein, den sie
	// beanstandet.
	if entry.Kind != domain.EntryKindReversal {
		if err := s.ensureAccrualTaxation(ctx); err != nil {
			return nil, err
		}
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}
	if d := entry.Entertainment; d != nil {
		if err := d.Validate(); err != nil {
			return nil, err
		}
	}
	if err := s.validateAccounts(ctx, entry); err != nil {
		return nil, err
	}
	if err := s.ensureLawfulTaxDisclosure(entry); err != nil {
		return nil, err
	}
	// Dieselbe Pflichtprüfung wie in ValidatePostable: Post ist der einzige
	// Schreibweg, und eine Invariante, die nur die Vorprüfung kennt, ist keine.
	if entry.Source == domain.EntrySourceManual {
		if err := ValidateManualTaxLines(entry); err != nil {
			return nil, err
		}
	}
	// Der Abschlussstand steht vor der Festschreibung: ein festgestelltes Jahr
	// ist immer auch festgeschrieben, und von den beiden Meldungen ist die über
	// die Feststellung die weiterführende — sie nennt die Rücksetzung, während
	// die Festschreibung nur auf den Storno verweist, der hier nicht hilft.
	if err := s.ensureYearNotAdopted(ctx, entry); err != nil {
		return nil, err
	}
	if err := s.ensurePeriodOpen(ctx, entry); err != nil {
		return nil, err
	}
	if err := s.ensureReceiptHeader(ctx, entry); err != nil {
		return nil, err
	}

	entry.CreatedAt = time.Now().UTC()

	if err := s.journalRepo.Append(ctx, entry, s.hashChain.CalculateHash); err != nil {
		return nil, fmt.Errorf("Buchung konnte nicht gespeichert werden: %w", err)
	}

	s.audit(ctx, domain.AuditActionCreate, entry.ID, fmt.Sprintf(
		"Buchung %s: %s, %s € (GJ %d, %d Zeilen)",
		entry.EntryNumber, entry.Description, entry.GrossAmount(), entry.FiscalYear, len(entry.Lines),
	))

	return entry, nil
}

// ensureReceiptHeader verlangt die Kopfdaten des Belegs, auf den die Buchung
// verweist.
//
// Die Kopfdaten sind beim Ablegen freiwillig und beim Buchen Pflicht (BEL-02).
// „Beim Buchen" heißt: hier — Post ist der einzige Schreibweg ins Journal, und
// eine Regel, die nur im Dialog „Beleg buchen" gilt, ließe jeden anderen Weg
// (Abschlussbuchung, Anlagenzugang, Zahlungsausgleich) an ihr vorbei.
//
// Geprüft wird ValidateHeader und nicht ValidateBookable: die dort zusätzlich
// verlangte Ansehbarkeit gilt dem Beleg, den ein Mensch vor sich hat, und
// nicht dem maschinell erzeugten Eigenbeleg einer Abschlussbuchung.
//
// Ein Verweis auf einen Beleg, den es nicht gibt, hält die Buchung nicht auf:
// er ist ein anderer Fehler als ein fehlendes Belegdatum, und ihn hier zu
// melden, verdeckte ihn.
//
// Zwei Fälle sind ausgenommen, und beide aus demselben Grund: die Kopfdatenpflicht
// gilt dem Buchen eines noch offenen Belegs und darf keine Buchung sperren, die
// sich nicht mehr in Ordnung bringen lässt.
func (s *JournalService) ensureReceiptHeader(ctx context.Context, entry *domain.JournalEntry) error {
	if s.receiptRepo == nil || entry.ReceiptID == nil {
		return nil
	}
	// Die Generalumkehr läuft durch — dieselbe Erwägung wie bei der
	// Istversteuerung weiter oben in Post. Der Storno kopiert den Beleg der
	// Ursprungsbuchung; hinge er an den Kopfdaten, ließe sich eine Altbuchung
	// auf einen Beleg ohne Kopfdaten nicht mehr zurücknehmen, obwohl gerade der
	// Storno der Weg ist, den GoBD Rz. 58 für die Korrektur vorsieht. Eine
	// Regel über künftige Aufzeichnungen darf die Korrektur vorhandener nicht
	// verhindern.
	if entry.Kind == domain.EntryKindReversal {
		return nil
	}
	receipt, err := s.receiptRepo.FindByID(ctx, *entry.ReceiptID)
	if err != nil || receipt == nil {
		return nil
	}
	// Ein Beleg, der nicht mehr offen ist, ist bereits gebucht (versiegelt) oder
	// verworfen. Beim Versiegeln sind die Kopfdaten geprüft worden
	// (ReceiptService.Seal); lagen damals keine vor — Altbestand aus der Zeit
	// vor BEL-02 —, lassen sie sich auch nicht mehr nachtragen, weil SaveHeader
	// den offenen Beleg verlangt. Jede weitere Buchung auf denselben Beleg hier
	// abzuweisen, sperrte den Altbestand ein, ohne einen Weg heraus zu lassen.
	if receipt.Status != domain.ReceiptStatusFiled {
		return nil
	}
	return receipt.ValidateHeader()
}

// ValidatePostable prüft eine Buchung, ohne sie zu schreiben.
//
// Gedacht für Vorgänge, die aus mehreren Buchungen bestehen und nicht zur Hälfte
// geschehen dürfen — allen voran der Korrekturvortrag, der erst den alten
// Saldenvortrag zurücknimmt und dann den neuen bucht. Scheiterte dort die zweite
// Buchung, stünde das Zieljahr mit zurückgenommenem Altvortrag und halbem
// Neuvortrag da, also mit einer Eröffnungsbilanz, die es so nie gab. Geprüft
// wird deshalb vorher, was sich vorher prüfen lässt: Konten, Abschlussstand,
// Periodensperre und die Kopfdaten des Belegs. Die Buchung selbst bleibt
// unverändert; gearbeitet wird auf einer Kopie.
//
// Die Kopfdatenprüfung gehört ausdrücklich dazu: ohne sie bestünde eine
// Neubuchung auf einen Beleg ohne Kopfdaten die Vorprüfung, der Storno würde
// geschrieben, und erst Post scheiterte — also der halb ausgeführte
// Vorgang, den diese Methode verhindern soll. Was Post abweist, muss sie
// abweisen, sonst prüft sie etwas anderes als das, was danach geschieht.
func (s *JournalService) ValidatePostable(ctx context.Context, entry *domain.JournalEntry) error {
	probe := *entry
	probe.Lines = append([]domain.JournalLine(nil), entry.Lines...)
	s.applyDefaults(ctx, &probe)

	if probe.Kind != domain.EntryKindReversal {
		if err := s.ensureAccrualTaxation(ctx); err != nil {
			return err
		}
	}
	if err := probe.Validate(); err != nil {
		return err
	}
	if d := probe.Entertainment; d != nil {
		if err := d.Validate(); err != nil {
			return err
		}
	}
	if err := s.validateAccounts(ctx, &probe); err != nil {
		return err
	}
	if err := s.ensureLawfulTaxDisclosure(&probe); err != nil {
		return err
	}
	// Steuerfall, Steuerschlüssel und Bemessungsgrundlage sind an einer
	// Handbuchung Pflicht (Welle 8, Entscheidung 1).
	//
	// Hier und nicht nur im Vorbau PostManualEntry: die Regel gilt für die
	// Aufzeichnung und nicht für eine Maske. Eine Buchung, die über einen
	// anderen Weg mit Source „manual" hereinkäme, liefe sonst an der
	// Umsatzsteuer-Auswertung vorbei — ohne Schlüssel keine Kennziffer, ohne
	// Bemessungsgrundlage keine Zeile 81. Die programmseitigen Buchungen
	// haben eine andere Quelle oder den Steuerfall; die Generalumkehr nimmt
	// ValidateManualTaxLines selbst aus.
	if probe.Source == domain.EntrySourceManual {
		if err := ValidateManualTaxLines(&probe); err != nil {
			return err
		}
	}
	if err := s.ensureYearNotAdopted(ctx, &probe); err != nil {
		return err
	}
	if err := s.ensurePeriodOpen(ctx, &probe); err != nil {
		return err
	}
	return s.ensureReceiptHeader(ctx, &probe)
}

// Reverse cancels a booking by Generalumkehr: the same accounts on the same
// sides with negated amounts.
//
// A side-swapped Storno would also produce a zero balance, but it inflates the
// Verkehrszahlen of every account involved — a 1.000 € expense corrected that
// way leaves the account showing 1.000 € Soll and 1.000 € Haben. The Summen- und
// Saldenliste and the VAT figures derived from turnover would then be wrong. The
// Generalumkehr returns the turnover to zero and is what DATEV records as "GU".
func (s *JournalService) Reverse(ctx context.Context, entryID uint, reason string) (*domain.JournalEntry, error) {
	return s.ReverseOn(ctx, entryID, reason, "")
}

// ReverseOn is the Generalumkehr with an explicit correction date; an empty date
// means today, which is what Reverse passes.
//
// Es gibt genau einen Grund, das Datum vorzugeben, und der ist der
// Korrekturvortrag. Ein Saldenvortrag steht auf dem ersten Tag des neuen Jahres;
// seine Rücknahme muss in demselben Jahr landen, sonst trägt das neue Jahr den
// alten Vortrag weiter und den neuen dazu — die Eröffnungsbilanz wäre doppelt
// gebucht. Auf dem Weg über „heute" wäre das nur so lange richtig, wie die
// Korrektur im selben Jahr geschieht; im Januar darauf wäre sie es nicht mehr.
//
// Rückdatiert wird trotzdem nicht in eine geschlossene Periode: ensurePeriodOpen
// weist jedes Datum bis zur letzten Festschreibung ab, und der Aufrufer, der ein
// Datum vorgibt, muss sich am ersten offenen Tag orientieren.
//
// Und das vorgegebene Datum bleibt auf Eröffnungsbuchungen beschränkt: für jede
// andere Buchung wäre es ein Weg, eine Korrektur in einen abgelaufenen, nur noch
// nicht festgeschriebenen Zeitraum zurückzudatieren — das, was der Storno
// auf „heute" verhindert.
func (s *JournalService) ReverseOn(ctx context.Context, entryID uint, reason, date string) (*domain.JournalEntry, error) {
	original, err := s.journalRepo.FindByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("Buchung %d wurde nicht gefunden: %w", entryID, err)
	}
	if original.Kind == domain.EntryKindReversal {
		return nil, fmt.Errorf("Buchung %s ist selbst eine Generalumkehr und kann nicht erneut storniert werden", original.EntryNumber)
	}
	if date != "" && original.Source != domain.EntrySourceOpening {
		return nil, fmt.Errorf(
			"Buchung %s ist keine Eröffnungsbuchung; eine Generalumkehr mit vorgegebenem Datum gibt es "+
				"nur für den Saldenvortrag. Storniere die Buchung ohne Datumsangabe – die Korrektur hat "+
				"dann den Tag ihrer Erstellung", original.EntryNumber)
	}
	// Und auch beim Saldenvortrag ist das Datum nicht frei: die Rücknahme
	// gehört in das Geschäftsjahr der Ursprungsbuchung. Landete sie in einem
	// anderen Jahr, stünde der Vortrag im einen Jahr doppelt und im anderen
	// eine Umkehr ohne Gegenstück — die Eröffnungsbilanz beider Jahre wäre
	// falsch.
	if date != "" {
		if target := domain.GetFiscalYearForDate(date, s.fiscalYearStartMonth(ctx)); target != original.FiscalYear {
			return nil, fmt.Errorf(
				"das Korrekturdatum %s fällt in das Geschäftsjahr %d; die Generalumkehr zur "+
					"Eröffnungsbuchung %s gehört in das Geschäftsjahr %d",
				date, target, original.EntryNumber, original.FiscalYear)
		}
	}

	existing, err := s.journalRepo.FindReversalOf(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("bestehende Stornos zu Buchung %d konnten nicht geprüft werden: %w", entryID, err)
	}
	if existing != nil {
		return nil, fmt.Errorf("Buchung %s wurde bereits durch %s storniert", original.EntryNumber, existing.EntryNumber)
	}
	if reason == "" {
		return nil, fmt.Errorf("für eine Stornierung ist ein Grund anzugeben")
	}

	// The correction is dated at the time of correction, never backdated into
	// the original period: that is what keeps a committed period untouched.
	today := date
	if today == "" {
		today = todayLocal()
	}

	lines := make([]domain.JournalLine, 0, len(original.Lines))
	for _, l := range original.Lines {
		lines = append(lines, domain.JournalLine{
			Position:  l.Position,
			Side:      l.Side, // unchanged — this is what makes it a Generalumkehr
			Account:   l.Account,
			ContactID: l.ContactID,
			Amount:    -l.Amount,
			TaxKey:    l.TaxKey,
			TaxBase:   -l.TaxBase,
			Text:      l.Text,
		})
	}

	reversal := &domain.JournalEntry{
		FiscalYear:         domain.GetFiscalYearForDate(today, s.fiscalYearStartMonth(ctx)),
		BookingDate:        today,
		DocumentDate:       original.DocumentDate,
		ServiceDateFrom:    original.ServiceDateFrom,
		ServiceDateTo:      original.ServiceDateTo,
		ValueDate:          original.ValueDate,
		Description:        fmt.Sprintf("Storno zu %s: %s", original.EntryNumber, original.Description),
		Source:             original.Source,
		DocumentNumber:     original.DocumentNumber,
		ReceiptID:          original.ReceiptID,
		ReceiptHash:        original.ReceiptHash,
		TaxTreatment:       original.TaxTreatment,
		ContactID:          original.ContactID,
		BankTxID:           original.BankTxID,
		Kind:               domain.EntryKindReversal,
		ReversalOfID:       &original.ID,
		ReversalReason:     reason,
		Currency:           original.Currency,
		ExchangeRateMicros: original.ExchangeRateMicros,
		ExchangeRateSource: original.ExchangeRateSource,
		ExchangeRateDate:   original.ExchangeRateDate,
		PostingRuleVersion: original.PostingRuleVersion,
		Lines:              lines,
	}
	// The Generalumkehr points at the same Beleg and carries the same
	// Aufzeichnung: it corrects the booking, it does not undo the meal.
	if d := original.Entertainment; d != nil {
		reversal.Entertainment = &domain.EntertainmentDetail{
			Place: d.Place, Day: d.Day, Participants: d.Participants, Occasion: d.Occasion,
		}
	}
	// Für das Geschenk gilt dasselbe: § 4 Abs. 7 EStG verlangt die Aufzeichnung
	// zur Aufwendung, und die Umkehr ist eine Buchung über dieselbe Aufwendung.
	// Sie ohne den Empfänger zu schreiben hieße, eine Buchung zu hinterlassen,
	// die aus sich heraus nicht mehr erklärt, was sie zurücknimmt. Gezählt wird
	// sie deshalb trotzdem nicht — die Freigrenze richtet sich nach den Normalbuchungen.
	for _, g := range original.Gifts {
		var recipient *uint
		if g.RecipientContactID != nil {
			id := *g.RecipientContactID
			recipient = &id
		}
		reversal.Gifts = append(reversal.Gifts, domain.GiftRecord{
			RecipientContactID: recipient,
			RecipientName:      g.RecipientName,
			Occasion:           g.Occasion,
			Date:               g.Date,
			NetAmount:          g.NetAmount,
			Account:            g.Account,
			NonDeductible:      g.NonDeductible,
			FiscalYear:         g.FiscalYear,
		})
	}

	created, err := s.Post(ctx, reversal)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, domain.AuditActionStorno, created.ID, fmt.Sprintf(
		"Generalumkehr %s storniert Buchung %s (Grund: %s)",
		created.EntryNumber, original.EntryNumber, reason,
	))

	return created, nil
}

// VerifyIntegrity rechnet die Hash-Chain jedes Geschäftsjahres nach.
//
// Alle Jahre und nicht nur das aktive: die Kette beginnt je Jahr neu, und eine
// Prüfung, die nur das laufende Jahr ansieht, meldet Unversehrtheit, während in
// einem festgeschriebenen Jahr eine Zeile verändert wurde. Genau dagegen steht
// die Kette (§ 146 Abs. 4 AO, GoBD Rz. 107) — sie muss also über den ganzen
// Aufbewahrungszeitraum geprüft werden können.
func (s *JournalService) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	years, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return domain.IntegrityCheckResult{}, err
	}
	if len(years) == 0 && s.fiscalYear > 0 {
		years = []int{s.fiscalYear}
	}

	byYear := make(map[int][]domain.JournalEntry, len(years))
	for _, year := range years {
		entries, err := s.journalRepo.FindAll(ctx, year)
		if err != nil {
			return domain.IntegrityCheckResult{}, err
		}
		byYear[year] = entries
	}

	result := s.hashChain.VerifyYears(byYear)

	// Die Protokollkette gehört in dieselbe Antwort wie die Journalkette.
	//
	// Beide beantworten die Frage nach der Unveränderbarkeit, und sie ist erst
	// beantwortet, wenn beide halten: wer eine Buchung ändert und danach den
	// Protokolleintrag darüber entfernt, hinterließe in einer Prüfung, die nur
	// das Journal nachrechnet, keine Spur. Ein gebrochenes Protokoll macht das
	// Gesamtergebnis deshalb ungültig.
	if s.auditRepo != nil {
		entries, err := s.auditRepo.FindAllAscending(ctx)
		if err != nil {
			return domain.IntegrityCheckResult{}, fmt.Errorf(
				"das Änderungsprotokoll konnte nicht gelesen werden: %w", err)
		}
		chain := accounting.NewAuditChain().Verify(entries)
		result.AuditChain = &chain
		if !chain.IsValid {
			result.IsValid = false
			result.Message += " " + chain.Message
		}
	}

	if s.auditRepo != nil {
		// Die Jahre mit Komma und nicht als Go-Wert einer Liste: „GJ_[2025
		// 2026]" steht so im Änderungsprotokoll und in aenderungsprotokoll.csv
		// und lässt sich dort weder lesen noch filtern.
		years := make([]string, 0, len(result.FiscalYears))
		for _, year := range result.FiscalYears {
			years = append(years, strconv.Itoa(year))
		}
		// Ohne ein einziges Geschäftsjahr — eine Datenbank ohne Buchungen —
		// stünde sonst das nackte „GJ_" im Protokoll und in
		// aenderungsprotokoll.csv. Eine Objektkennung, die nichts benennt, ist
		// später nicht mehr zu deuten.
		entity := "GJ_keine"
		if len(years) > 0 {
			entity = "GJ_" + strings.Join(years, ",")
		}
		_ = s.auditRepo.Log(ctx, domain.AuditActionIntegrityCheck, "HASH_CHAIN",
			entity, result.Message)
	}
	return result, nil
}

func (s *JournalService) applyDefaults(ctx context.Context, e *domain.JournalEntry) {
	if e.BookingDate == "" {
		e.BookingDate = todayLocal()
	}
	if e.DocumentDate == "" {
		e.DocumentDate = e.BookingDate
	}
	if e.ServiceDateFrom == "" {
		e.ServiceDateFrom = e.DocumentDate
	}
	if e.ServiceDateTo == "" {
		e.ServiceDateTo = e.ServiceDateFrom
	}
	if e.Kind == "" {
		e.Kind = domain.EntryKindNormal
	}
	// Programmfassung, Bearbeiterkennung und Regelstand stehen an jeder
	// Buchung, ohne dass ein Aufrufer daran denken muss.
	//
	// Vorher setzten nur die automatischen Wege den Regelstand; eine von Hand
	// erfasste Buchung trug ihn nicht, und ausgerechnet die, über die am
	// meisten gestritten wird, ließ sich nicht mehr erklären (UNV-06). Die drei
	// Felder gehören zu jeder Aufzeichnung und nicht zu einem Erfassungsweg.
	if e.PostingRuleVersion == "" {
		e.PostingRuleVersion = accounting.PostingRuleVersion
	}
	if e.AppVersion == "" {
		e.AppVersion = buildinfo.Version
	}
	if e.Actor == "" {
		e.Actor = actor.Actor()
	}
	if e.Source == "" {
		e.Source = domain.EntrySourceManual
	}
	if e.Currency == "" {
		e.Currency = "EUR"
	}
	if e.ExchangeRateMicros == 0 {
		e.ExchangeRateMicros = 1_000_000
	}
	if e.FiscalYear == 0 {
		e.FiscalYear = domain.GetFiscalYearForDate(e.BookingDate, s.fiscalYearStartMonth(ctx))
	}
	for i := range e.Lines {
		if e.Lines[i].Position == 0 {
			e.Lines[i].Position = i + 1
		}
	}
}

// ensureAccrualTaxation refuses to book while the company is set to
// Istversteuerung.
//
// The whole flow — record the invoice, book it at once, settle it later —
// presumes taxation on agreed consideration (§ 16 Abs. 1 Satz 1 UStG). Under
// Istversteuerung the tax only arises with the receipt of payment
// (§ 13 Abs. 1 Nr. 1 Buchst. b UStG) and the bookings would look different. The
// setting has existed since the setup wizard shipped, but nothing ever checked
// it — so the option quietly produced wrong VAT returns. Refusing is the honest
// behaviour until the second booking path exists.
func (s *JournalService) ensureAccrualTaxation(ctx context.Context) error {
	if s.settingsRepo == nil {
		return nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.TaxationType == "" {
		return nil
	}
	if strings.EqualFold(cfg.TaxationType, "SOLL") {
		return nil
	}
	// Kein Verweis auf eine Einstellung: die Besteuerungsart ist in der
	// Oberfläche nur ablesbar, nicht änderbar. Eine Meldung, die auf einen
	// Schalter zeigt, den es nicht gibt, hilft niemandem weiter.
	return fmt.Errorf(
		"Für diesen Mandanten ist die Istversteuerung hinterlegt. Buchfink unterstützt derzeit nur die Sollversteuerung (§ 16 Abs. 1 Satz 1 UStG): bei Istversteuerung entsteht die Steuer erst mit der Vereinnahmung des Entgelts (§ 13 Abs. 1 Nr. 1 Buchst. b UStG), und die Buchungen sähen anders aus. Stornos bestehender Buchungen bleiben möglich")
}

func (s *JournalService) fiscalYearStartMonth(ctx context.Context) int {
	if s.settingsRepo == nil {
		return 1
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FiscalYearStartMonth <= 0 {
		return 1
	}
	return cfg.FiscalYearStartMonth
}

// validateAccounts rejects bookings that reference accounts which cannot carry
// them. Without this check the journal accepts numbers that exist nowhere in the
// chart of accounts, and the resulting balances silently omit them.
// ensureLawfulTaxDisclosure hält § 14c UStG auch auf dem Handbuchungsweg.
//
// Der Rechnungsdienst weist eine Rechnung zurück, die zu einem Steuerfall ohne
// Steuerpflicht Umsatzsteuer ausweist. Ohne dieselbe Regel hier bliebe die
// Handbuchung offen: eine innergemeinschaftliche Lieferung mit einer UST19-Zeile
// stünde in Kennziffer 41 *und* in 81, und in der Zusammenfassenden Meldung
// stünde ein steuerfreier Umsatz, den die Voranmeldung als steuerpflichtig
// führt. Wer die Steuer tatsächlich schuldet, weil die Rechnung außerhalb von
// Buchfink entstanden ist, bucht sie mit dem Schlüssel UST14C in Kennziffer 69.
//
// Die Generalumkehr läuft durch: eine Regel über künftige Buchungen darf die
// Rücknahme vorhandener nicht verhindern.
//
// Fehlt der Steuerfall an der Buchung, entscheiden die Erlöskonten. Eine
// Handbuchung hat ihn nicht zwingend — sie kommt ohne Buchungsgruppe zustande
// —, und ohne diese Ableitung stünde der Fall offen, den die Regel
// verhindern soll: Konto 4125 und eine UST19-Zeile in einer Buchung, gemeldet in
// Kennziffer 41 *und* in 81.
func (s *JournalService) ensureLawfulTaxDisclosure(e *domain.JournalEntry) error {
	if e.Kind == domain.EntryKindReversal {
		return nil
	}
	treatment := e.TaxTreatment
	derived := false
	if treatment == "" {
		treatment = revenueTreatmentOf(e)
		derived = treatment != ""
	}
	if treatment == "" || treatment.MayShowTax() {
		return nil
	}
	for i, l := range e.Lines {
		if !accounting.IsDomesticOutputTaxKey(l.TaxKey) {
			continue
		}
		source := fmt.Sprintf("der Steuerfall %q", treatment)
		if derived {
			source = fmt.Sprintf("das Erlöskonto der Buchung führt den Steuerfall %q und", treatment)
		}
		return fmt.Errorf(
			"Zeile %d weist %s € Umsatzsteuer aus, %s lässt aber keine entstehen. "+
				"Ein solcher Ausweis wird nach § 14c UStG trotzdem geschuldet – buche ihn mit dem "+
				"Steuerschlüssel %s (Kennziffer 69) oder wähle den steuerpflichtigen Inlandsumsatz",
			i+1, l.Amount, source, accounting.TaxKeyUnlawful)
	}
	return nil
}

// revenueTreatmentOf leitet den Steuerfall einer Buchung aus ihren Erlöskonten
// ab.
//
// Abgeleitet wird nur, wenn *jede* Erlöszeile der Buchung einen Steuerfall ohne
// Steuerpflicht hat. Eine Buchung, die daneben einen steuerpflichtigen
// Inlandsumsatz enthält, darf Umsatzsteuer ausweisen — sie gehört dann zu diesem
// Erlös, und ein Verbot träfe die richtige Buchung.
func revenueTreatmentOf(e *domain.JournalEntry) domain.TaxTreatment {
	treatments := accounting.RevenueTreatments()
	taxable := accounting.TaxableRevenueAccounts()
	found := domain.TaxTreatment("")
	for _, l := range e.Lines {
		if taxable[l.Account] {
			return ""
		}
		if t, ok := treatments[l.Account]; ok && t != "" && !t.MayShowTax() && found == "" {
			found = t
		}
	}
	return found
}

func (s *JournalService) validateAccounts(ctx context.Context, e *domain.JournalEntry) error {
	chart, err := s.Chart(ctx)
	if err != nil {
		return err
	}

	for i, l := range e.Lines {
		if domain.IsLedgerAccount(l.Account) {
			if err := s.validateLedgerAccount(ctx, l); err != nil {
				return fmt.Errorf("Zeile %d: %w", i+1, err)
			}
			continue
		}

		if err := chart.EnsurePostable(l.Account); err != nil {
			return fmt.Errorf("Zeile %d: %w", i+1, err)
		}

		// Tax accounts carry the figures of the Umsatzsteuer-Voranmeldung. They
		// may only be written by the tax automation, which stamps a TaxKey on
		// the line it generates.
		//
		// Der Saldenvortrag ist die Ausnahme, und er ist keine Umgehung: Konten
		// wie die abziehbare Vorsteuer sind Bilanzkonten und tragen zum
		// Bilanzstichtag einen Bestand, der ins neue Jahr gehört (§ 252 Abs. 1
		// Nr. 1 HGB). Vorgetragen wird der Saldo, nicht ein Umsatz — die Zeile
		// hat deshalb bewusst keinen Steuerschlüssel, und die
		// Umsatzsteuer-Auswertung lässt Eröffnungsbuchungen aus.
		//
		// Die zweite Ausnahme ist die Umsatzsteuer-Jahresverrechnung: sie stellt
		// die Steuerkonten zum Bilanzstichtag auf null und bringt den Saldo auf
		// die Verbindlichkeit bzw. Forderung, die der Jahreserklärung
		// entspricht. Auch sie bucht keinen Umsatz, sondern einen Bestand —
		// deshalb hat sie keinen Steuerschlüssel, und die Auswertungen lassen
		// Abschlussbuchungen genauso aus wie den Vortrag.
		//
		// Die Ausnahme gilt ausdrücklich nur für diese eine Buchung und nur für
		// die Konten, die sie auf null stellt: jeder andere Abschlussbaustein
		// bucht auf anwendergewählte Konten, und ein Steuerkonto darunter würde
		// die Voranmeldung von der Jahreserklärung trennen.
		if s.taxResolver.IsTaxAccount(l.Account) && l.TaxKey == "" && !mayWriteTaxAccount(e, l.Account) {
			return fmt.Errorf(
				"Zeile %d: Konto %s ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden",
				i+1, l.Account,
			)
		}

		// Offene Posten gehören auf das Personenkonto des Geschäftspartners.
		// Eine Buchung direkt auf das Sammelkonto stünde zwar in der Bilanz,
		// aber in keiner OPOS-Liste.
		if kind, ok := domain.CollectiveAccounts()[l.Account]; ok {
			partner := "Kunden"
			if kind == domain.ContactTypeVendor {
				partner = "Lieferanten"
			}
			return fmt.Errorf(
				"Zeile %d: Konto %s ist das Sammelkonto für die Bilanz und wird nicht direkt bebucht. "+
					"Buche den offenen Posten auf das Personenkonto des %s – die Bilanzposition verdichtet sich daraus",
				i+1, l.Account, partner,
			)
		}
	}

	return nil
}

// isVatSettlementReference erkennt die Belegnummer einer Jahresverrechnung.
//
// Geprüft wird die Form und nicht das Jahr des Eintrags: die Generalumkehr einer
// Verrechnung hat deren Belegnummer, liegt aber im Geschäftsjahr ihrer
// Erstellung. Mit einem Vergleich auf das Jahr des Eintrags ließe sich eine
// Verrechnung nicht mehr stornieren.
func isVatSettlementReference(documentNumber string) bool {
	year, err := strconv.Atoi(strings.TrimPrefix(documentNumber, "USTV "))
	if err != nil {
		return false
	}
	return documentNumber == VatSettlementReference(year)
}

// mayWriteTaxAccount meldet, ob eine Zeile auf einem Steuerkonto ohne
// Steuerschlüssel ausnahmsweise zulässig ist.
//
// Der Saldenvortrag darf jedes Steuerkonto anfassen — er trägt Bestände vor.
// Die Abschlussbuchung darf es nur als Umsatzsteuer-Jahresverrechnung, erkennbar
// an ihrer Belegnummer, und auch dann nur auf den Konten, die sie auf null
// stellt. Die Belegnummer allein ist kein Schlüssel, den jemand von außen
// setzen könnte: die Bridge normiert die Quelle jeder Handbuchung auf manual,
// und ohne Quelle „closing" greift diese Ausnahme nicht.
func mayWriteTaxAccount(e *domain.JournalEntry, account string) bool {
	if e.Source == domain.EntrySourceOpening {
		return true
	}
	if e.Source != domain.EntrySourceClosing {
		return false
	}
	if !isVatSettlementReference(e.DocumentNumber) {
		return false
	}
	for _, settlement := range settlementAccounts() {
		if settlement == account {
			return true
		}
	}
	return false
}

func (s *JournalService) validateLedgerAccount(ctx context.Context, l domain.JournalLine) error {
	if s.contactRepo == nil {
		return fmt.Errorf("Personenkonto %s kann ohne Stammdaten nicht geprüft werden", l.Account)
	}
	contact, err := s.contactRepo.FindByLedgerAccount(ctx, l.Account)
	if err != nil || contact == nil {
		return fmt.Errorf("Personenkonto %s gehört zu keinem angelegten Geschäftspartner", l.Account)
	}
	if l.ContactID != nil && *l.ContactID != contact.ID {
		return fmt.Errorf("Personenkonto %s gehört zu %s, die Buchung verweist aber auf einen anderen Geschäftspartner", l.Account, contact.Name)
	}
	return nil
}

// ensurePeriodOpen blocks bookings backdated into a committed period. A
// correction is dated at correction time and therefore never lands in one.
func (s *JournalService) ensurePeriodOpen(ctx context.Context, e *domain.JournalEntry) error {
	if s.festschreibungRepo == nil {
		return nil
	}
	cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, e.FiscalYear)
	if err != nil {
		return fmt.Errorf("Festschreibungsstand konnte nicht geprüft werden: %w", err)
	}
	if cutoff != "" && e.BookingDate <= cutoff {
		return fmt.Errorf(
			"Der Zeitraum bis %s ist festgeschrieben. Eine Buchung zum %s ist nicht mehr möglich – bitte über eine Stornierung korrigieren",
			cutoff, e.BookingDate,
		)
	}
	return nil
}

// ensureYearNotAdopted blocks every booking into a fiscal year whose annual
// accounts have been adopted.
//
// Mit der Feststellung durch die Gesellschafter (§ 42a Abs. 2 GmbHG) ist der
// Abschluss verbindlich: er ist die Grundlage des Ergebnisverwendungsbeschlusses
// und der Steuererklärung. Eine Buchung danach änderte Zahlen, die beschlossen
// und weitergereicht sind, ohne dass jemand davon erführe. Deshalb sperrt die
// Feststellung das Jahr — und deshalb nennt die Meldung den Weg zurück, statt
// nur nein zu sagen: die Rücksetzung ist möglich, sie ist nur eine Entscheidung
// und kein Nebeneffekt einer Buchung.
//
// Die Generalumkehr ist nicht ausgenommen. Sie hat das Datum ihrer eigenen
// Erstellung und landet damit im laufenden Jahr; nur eine ausdrücklich in das
// festgestellte Jahr datierte Korrektur fällt hierunter, und die soll auffallen.
func (s *JournalService) ensureYearNotAdopted(ctx context.Context, e *domain.JournalEntry) error {
	if s.fiscalYearRepo == nil {
		return nil
	}
	fy, err := s.fiscalYearRepo.FindByYear(ctx, e.FiscalYear)
	if err != nil {
		return fmt.Errorf("der Abschlussstand des Geschäftsjahres %d konnte nicht geprüft werden: %w", e.FiscalYear, err)
	}
	if fy == nil || !fy.IsAdopted() {
		return nil
	}
	return fmt.Errorf(
		"Der Jahresabschluss %d ist am %s festgestellt (§ 42a Abs. 2 GmbHG). Eine Buchung zum %s ist "+
			"deshalb nicht mehr möglich. Wenn der Abschluss geändert werden muss, setze das Geschäftsjahr "+
			"zunächst unter „Jahresabschluss\" mit Angabe des Grundes zurück",
		fy.Year, fy.AdoptedOn, e.BookingDate)
}

func (s *JournalService) audit(ctx context.Context, action domain.AuditAction, id uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "JOURNAL", fmt.Sprintf("%d", id), details)
}
