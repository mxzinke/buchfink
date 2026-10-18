package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// JournalService is the single write path into the journal.
//
// Every booking in the system goes through Post, which enforces the rules that
// make the journal usable as accounting evidence: balanced entries, existing and
// bookable accounts, tax accounts reserved for the tax automation, gapless
// numbering, an unbroken hash chain and respect for committed periods.
type JournalService struct {
	auditor[uint]
	periods      *PeriodService
	settingsRepo domain.SettingsRepository
	journalRepo  domain.JournalRepository
	accountRepo  domain.AccountRepository
	contactRepo  domain.ContactRepository
	// receiptRepo liefert Belege für den Hashstempel und die lesende Regelsicht.
	receiptRepo domain.ReceiptRepository
	// bankRepo hält fest, welcher Bankumsatz gebucht ist (journal_bank.go).
	bankRepo domain.BankRepository

	// tx ist die Transaktionsklammer aller Vorgänge, die über das Journal
	// schreiben (siehe InTx).
	tx domain.TxRunner

	hashChain            *accounting.HashChain
	taxResolver          domain.TaxResolver
	ruleExtensions       []domain.EntryRule
	taxAccountExceptions []domain.TaxAccountException

	// chart ist der zwischengespeicherte Kontenplan im festgeschriebenen
	// Stand der Datenbank. Füllen dürfen ihn nur Schreibvorgänge ohne eigene
	// Kontenänderung: ein Lesevorgang sieht womöglich einen älteren Stand, und
	// ein Schreibvorgang, der Konten geändert hat (chartPending), einen noch
	// nicht festgeschriebenen. Beide bauen sich den Plan selbst. chartMu ordnet
	// Aufbau, Verwerfen und Abschluss.
	chart        atomic.Pointer[accounting.Chart]
	chartMu      sync.Mutex
	chartPending bool
}

// NewJournalService wires the journal write path.
func NewJournalService(
	journalRepo domain.JournalRepository,
	accountRepo domain.AccountRepository,
	contactRepo domain.ContactRepository,
	auditRepo domain.AuditRepository,
	settingsRepo domain.SettingsRepository,
	tx domain.TxRunner,
) *JournalService {
	return &JournalService{
		journalRepo:  journalRepo,
		accountRepo:  accountRepo,
		contactRepo:  contactRepo,
		auditor:      auditor[uint]{auditRepo: auditRepo, entity: "JOURNAL"},
		settingsRepo: settingsRepo,
		periods:      newPeriodService(journalRepo, settingsRepo, auditRepo, tx),
		tx:           tx,
		hashChain:    accounting.NewHashChain(),
		taxResolver:  accounting.NewSKR04TaxResolver(),
	}
}

// SetFestschreibungRepo wires period-commitment enforcement. Optional.
func (s *JournalService) SetFestschreibungRepo(r domain.FestschreibungRepository) {
	s.periods.festschreibungRepo = r
}

// SetFiscalYearRepo wires the Abschlussstand of the fiscal years. Optional.
func (s *JournalService) SetFiscalYearRepo(r domain.FiscalYearRepository) {
	s.periods.fiscalYearRepo = r
}

// CommittedThrough ist der jüngste Stichtag einer Festschreibung, leer ohne
// Festschreibung.
func (s *JournalService) CommittedThrough(ctx context.Context) (string, error) {
	return s.periods.CommittedThrough(ctx)
}

// SetPeriods verbindet Journal und Abschluss mit derselben Periodenverwaltung.
func (s *JournalService) SetPeriods(periods *PeriodService) {
	if periods == nil {
		panic("Periodenverwaltung fehlt")
	}
	s.periods = periods
}

// SetReceiptRepo hängt die Belegablage an. Mit ihr prüft Post die Kopfdaten des
// Belegs, auf den eine Buchung verweist. Optional.
func (s *JournalService) SetReceiptRepo(r domain.ReceiptRepository) { s.receiptRepo = r }

// Chart liefert den Kontenplan, aus dem Zwischenspeicher, wo er gilt.
func (s *JournalService) Chart(ctx context.Context) (*accounting.Chart, error) {
	if c := s.chart.Load(); c != nil {
		return c, nil
	}
	if repository.IsReadOnly(ctx) {
		return s.loadChart(ctx)
	}
	s.chartMu.Lock()
	defer s.chartMu.Unlock()
	if c := s.chart.Load(); c != nil {
		return c, nil
	}
	c, err := s.loadChart(ctx)
	if err != nil {
		return nil, err
	}
	if !s.chartPending {
		s.chart.Store(c)
	}
	return c, nil
}

func (s *JournalService) loadChart(ctx context.Context) (*accounting.Chart, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Kontenplan konnte nicht geladen werden: %w", err)
	}
	return accounting.NewChart(accounts), nil
}

// InvalidateChart verwirft den Kontenplan nach einer Kontenänderung. In einer
// Transaktion bleibt er verworfen, bis sie endet (SettleChart): bis dahin
// gilt die Änderung nur in ihr.
func (s *JournalService) InvalidateChart(ctx context.Context) {
	s.chartMu.Lock()
	defer s.chartMu.Unlock()
	s.chart.Store(nil)
	if repository.InTx(ctx) {
		s.chartPending = true
	}
}

// SettleChart schließt eine Transaktion für den Kontenplan ab, gleich ob sie
// festgeschrieben oder zurückgerollt wurde. Hat sie Konten geändert, baut der
// nächste Schreibvorgang den Plan neu auf. Der Kern ruft es, solange er die
// Schreibsperre noch hält.
func (s *JournalService) SettleChart() {
	s.chartMu.Lock()
	defer s.chartMu.Unlock()
	if s.chartPending {
		s.chart.Store(nil)
		s.chartPending = false
	}
}

// TaxResolver exposes the SKR04 tax resolution used by the posting rules.
func (s *JournalService) TaxResolver() domain.TaxResolver { return s.taxResolver }

// postMode sagt, auf welchem Weg eine Buchung entsteht. Generalumkehr und
// Verweis auf eine ersetzte Buchung setzt nur der Dienst selbst.
type postMode struct {
	reversal   bool
	correctsID *uint
}

// Post validates and appends a journal entry.
//
// Kennung, Nummer, Hashes, Bearbeiter, Erfassungsweg, Programmfassung,
// Regelstand und Festschreibungsstempel setzt der Dienst; was der Aufrufer dort mitgibt, wird
// verworfen. Das Geschäftsjahr folgt aus dem Buchungsdatum, eine abweichende
// Angabe wird abgewiesen. Eine Generalumkehr entsteht nur über Reverse.
func (s *JournalService) Post(ctx context.Context, entry *domain.JournalEntry) (*domain.JournalEntry, error) {
	return s.post(ctx, entry, postMode{})
}

func (s *JournalService) post(ctx context.Context, entry *domain.JournalEntry, mode postMode) (*domain.JournalEntry, error) {
	return atomically(ctx, s, func(ctx context.Context) (*domain.JournalEntry, error) {
		if err := s.prepare(ctx, entry, mode); err != nil {
			return nil, err
		}
		if err := s.check(ctx, entry); err != nil {
			return nil, err
		}
		bankTx, err := s.claimBankTx(ctx, entry)
		if err != nil {
			return nil, err
		}

		entry.CreatedAt = time.Now().UTC()

		if err := s.journalRepo.Append(ctx, entry, s.hashChain.CalculateHash); err != nil {
			return nil, fmt.Errorf("Buchung konnte nicht gespeichert werden: %w", err)
		}
		if err := s.settleBankTx(ctx, entry, bankTx); err != nil {
			return nil, err
		}

		if err := s.audit(ctx, domain.AuditActionCreate, entry.ID, fmt.Sprintf(
			"Buchung %s: %s, %s € (GJ %d, %d Zeilen)",
			entry.EntryNumber, entry.Description, entry.GrossAmount(), entry.FiscalYear, len(entry.Lines),
		)); err != nil {
			return nil, err
		}
		return entry, nil
	})
}

// ValidatePostable prüft eine Buchung, ohne sie zu schreiben.
//
// Gedacht für Vorgänge aus mehreren Buchungen, die nicht zur Hälfte geschehen
// dürfen. Geprüft wird eine Kopie mit denselben Regeln wie in Post.
func (s *JournalService) ValidatePostable(ctx context.Context, entry *domain.JournalEntry) error {
	return s.validatePostable(ctx, entry, postMode{})
}

func (s *JournalService) validatePostable(ctx context.Context, entry *domain.JournalEntry, mode postMode) error {
	probe := *entry
	probe.Lines = append([]domain.JournalLine(nil), entry.Lines...)
	probe.Gifts = append([]domain.GiftRecord(nil), entry.Gifts...)
	if entry.Entertainment != nil {
		d := *entry.Entertainment
		probe.Entertainment = &d
	}
	if err := s.prepare(ctx, &probe, mode); err != nil {
		return err
	}
	return s.check(ctx, &probe)
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
// Rückdatiert wird trotzdem nicht in eine geschlossene Periode: die Periodensperre
// weist jedes Datum bis zur letzten Festschreibung ab, und der Aufrufer, der ein
// Datum vorgibt, muss sich am ersten offenen Tag orientieren.
//
// Und das vorgegebene Datum bleibt auf Eröffnungsbuchungen beschränkt: für jede
// andere Buchung wäre es ein Weg, eine Korrektur in einen abgelaufenen, nur noch
// nicht festgeschriebenen Zeitraum zurückzudatieren — das, was der Storno
// auf „heute" verhindert.
func (s *JournalService) ReverseOn(ctx context.Context, entryID uint, reason, date string) (*domain.JournalEntry, error) {
	return atomically(ctx, s, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.reverseOn(ctx, entryID, reason, date)
	})
}

func (s *JournalService) reverseOn(ctx context.Context, entryID uint, reason, date string) (*domain.JournalEntry, error) {
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
				"nur für den Saldenvortrag. Storniert wird ohne Datumsangabe – die Korrektur hat "+
				"dann den Tag ihrer Erstellung", original.EntryNumber)
	}
	// Und auch beim Saldenvortrag ist das Datum nicht frei: die Rücknahme
	// gehört in das Geschäftsjahr der Ursprungsbuchung. Landete sie in einem
	// anderen Jahr, stünde der Vortrag im einen Jahr doppelt und im anderen
	// eine Umkehr ohne Gegenstück — die Eröffnungsbilanz beider Jahre wäre
	// falsch.
	if date != "" {
		if target := s.periods.FiscalYearOf(ctx, date); target != original.FiscalYear {
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

	created, err := s.post(ctx, reversal, postMode{reversal: true})
	if err != nil {
		return nil, err
	}

	if err := s.audit(ctx, domain.AuditActionStorno, created.ID, fmt.Sprintf(
		"Generalumkehr %s storniert Buchung %s (Grund: %s)",
		created.EntryNumber, original.EntryNumber, reason,
	)); err != nil {
		return nil, err
	}
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

	byYear := make(map[int][]domain.JournalEntry, len(years))
	for _, year := range years {
		entries, err := s.journalRepo.FindAll(ctx, year)
		if err != nil {
			return domain.IntegrityCheckResult{}, err
		}
		byYear[year] = entries
	}

	result := s.hashChain.VerifyYears(byYear)
	if err := s.verifyReceiptHashes(ctx, years, byYear, &result); err != nil {
		return domain.IntegrityCheckResult{}, err
	}

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
		if err := s.auditRepo.Log(ctx, domain.AuditActionIntegrityCheck, "HASH_CHAIN",
			entity, result.Message); err != nil {
			return domain.IntegrityCheckResult{}, fmt.Errorf("die Prüfung ließ sich nicht protokollieren: %w", err)
		}
	}
	return result, nil
}

// prepare füllt die Vorgaben und setzt, was nur der Dienst bestimmt.
// verifyReceiptHashes hält den Beleg-Hash jeder Buchung gegen den abgelegten
// Beleg. Die Journalkette deckt den Hash, den die Buchung trägt; erst dieser
// Vergleich zeigt, ob der Beleg selbst — Dateiliste und Kopfdaten — noch der
// ist, auf den gebucht wurde.
func (s *JournalService) verifyReceiptHashes(
	ctx context.Context, years []int, byYear map[int][]domain.JournalEntry, result *domain.IntegrityCheckResult,
) error {
	if s.receiptRepo == nil {
		return nil
	}
	actual := map[uint]string{}
	numbers := map[uint]string{}
	broken := 0
	for _, year := range years {
		for _, e := range byYear[year] {
			if e.ReceiptID == nil {
				continue
			}
			id := *e.ReceiptID
			if _, ok := actual[id]; !ok {
				receipt, err := s.receiptRepo.FindByID(ctx, id)
				if err != nil && !errors.Is(err, domain.ErrNotFound) {
					return fmt.Errorf("der Beleg %d ließ sich nicht lesen: %w", id, err)
				}
				if receipt != nil {
					actual[id] = accounting.ReceiptHash(receipt)
					numbers[id] = receipt.ReceiptNumber
				} else {
					actual[id] = ""
				}
			}
			if actual[id] == e.ReceiptHash {
				continue
			}
			message := fmt.Sprintf("Der Beleg %s stimmt nicht mehr mit dem Beleg-Hash der Buchung %s überein.",
				numbers[id], e.EntryNumber)
			if actual[id] == "" {
				message = fmt.Sprintf("Der Beleg %d der Buchung %s fehlt.", id, e.EntryNumber)
			}
			result.Breaks = append(result.Breaks, domain.IntegrityBreak{
				FiscalYear: e.FiscalYear, EntryID: e.ID, EntryNumber: e.EntryNumber,
				Reason: domain.IntegrityBreakReceipt, ExpectedHash: e.ReceiptHash, ActualHash: actual[id],
				Message: message,
			})
			broken++
		}
	}
	if broken > 0 {
		result.IsValid = false
		result.Message += fmt.Sprintf(" %d Buchung(en) verweisen auf einen veränderten oder fehlenden Beleg.", broken)
	}
	return nil
}

func (s *JournalService) prepare(ctx context.Context, e *domain.JournalEntry, mode postMode) error {
	if mode.reversal {
		if e.Kind != domain.EntryKindReversal || e.ReversalOfID == nil {
			return fmt.Errorf("die Generalumkehr verweist nicht auf die stornierte Buchung")
		}
	} else {
		if e.Kind == domain.EntryKindReversal || e.ReversalOfID != nil {
			return fmt.Errorf(
				"eine Generalumkehr entsteht nur über den Storno einer vorhandenen Buchung, " +
					"nicht als eigene Buchung")
		}
		if e.Kind == "" {
			e.Kind = domain.EntryKindNormal
		}
		e.ReversalReason = ""
	}

	if e.BookingDate == "" {
		e.BookingDate = todayLocal()
	}
	if err := domain.CheckDate("Buchungsdatum", e.BookingDate); err != nil {
		return err
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
	year := s.FiscalYearOf(ctx, e.BookingDate)
	if e.FiscalYear != 0 && e.FiscalYear != year {
		return fmt.Errorf(
			"das Buchungsdatum %s gehört zum Geschäftsjahr %d, angegeben ist %d. "+
				"Das Geschäftsjahr folgt aus dem Buchungsdatum", e.BookingDate, year, e.FiscalYear)
	}
	e.FiscalYear = year
	if err := s.periods.assignClosingPeriod(ctx, e); err != nil {
		return err
	}

	e.ID = 0
	e.EntryNumber = ""
	e.PreviousHash = ""
	e.EntryHash = ""
	e.CreatedAt = time.Time{}
	e.CommittedAt = nil
	e.FestschreibungID = nil
	e.CorrectsEntryID = mode.correctsID
	e.PostingRuleVersion = accounting.PostingRuleVersion
	e.AppVersion = buildinfo.Version
	e.Actor = actor.Actor()
	e.Channel = string(actor.RecordedChannel(ctx))

	if err := s.stampReceiptHash(ctx, e, mode); err != nil {
		return err
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
	for i := range e.Lines {
		e.Lines[i].ID = 0
		e.Lines[i].EntryID = 0
		e.Lines[i].Position = i + 1
	}
	if e.Entertainment != nil {
		e.Entertainment.ID = 0
		e.Entertainment.EntryID = 0
	}
	for i := range e.Gifts {
		e.Gifts[i].ID = 0
		e.Gifts[i].EntryID = 0
	}
	return nil
}

// stampReceiptHash setzt den Beleg-Hash aus dem abgelegten Beleg. Eine Buchung
// mit Belegverweis trägt ihn immer, eine ohne nie; was der Aufrufer mitgibt,
// zählt nicht. Die Generalumkehr übernimmt den Hash der stornierten Buchung.
func (s *JournalService) stampReceiptHash(ctx context.Context, e *domain.JournalEntry, mode postMode) error {
	if mode.reversal {
		return nil
	}
	e.ReceiptHash = ""
	if e.ReceiptID == nil {
		return nil
	}
	if s.receiptRepo == nil {
		return fmt.Errorf("ohne Belegablage lässt sich der Beleg %d der Buchung nicht prüfen", *e.ReceiptID)
	}
	receipt, err := s.receiptRepo.FindByID(ctx, *e.ReceiptID)
	if err != nil || receipt == nil {
		return fmt.Errorf("der Beleg %d, auf den die Buchung verweist, existiert nicht", *e.ReceiptID)
	}
	if actual := accounting.ReceiptHash(receipt); actual != receipt.ReceiptHash {
		return fmt.Errorf(
			"der Beleg %s stimmt nicht mehr mit seinem Hash überein (gespeichert %s, berechnet %s) "+
				"und kann keine Buchung belegen", receipt.ReceiptNumber, receipt.ReceiptHash, actual)
	}
	e.ReceiptHash = receipt.ReceiptHash
	return nil
}

// FiscalYearOf ist das Geschäftsjahr, in das ein Datum fällt.
func (s *JournalService) FiscalYearOf(ctx context.Context, date string) int {
	return s.periods.FiscalYearOf(ctx, date)
}

func (s *JournalService) legalForm(ctx context.Context) string {
	if s.settingsRepo == nil {
		return ""
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.LegalForm
}

func (s *JournalService) validateAccounts(ctx context.Context, e *domain.JournalEntry) error {
	chart, err := s.Chart(ctx)
	if err != nil {
		return err
	}
	legalForm := s.legalForm(ctx)

	for i, l := range e.Lines {
		if domain.IsLedgerAccount(l.Account) {
			if err := s.validateLedgerAccount(ctx, l); err != nil {
				return fmt.Errorf("Zeile %d: %w", i+1, err)
			}
			continue
		}

		if err := chart.EnsurePostableFor(l.Account, legalForm); err != nil {
			return fmt.Errorf("Zeile %d: %w", i+1, err)
		}

		// Steuerkonten liefern die Beträge der Voranmeldung. Ohne TaxKey darf
		// nur eine registrierte Ausnahme ihre Bestände verändern.
		if s.taxResolver.IsTaxAccount(l.Account) && l.TaxKey == "" && !s.mayWriteTaxAccount(e, l.Account) {
			return fmt.Errorf(
				"Zeile %d: Konto %s ist ein Steuerkonto und darf nur über die Steuerautomatik bebucht werden",
				i+1, l.Account,
			)
		}

		if err := s.checkRulesAt(ctx, domain.EntryRulesAfterAccount, e, i); err != nil {
			return err
		}
	}

	return nil
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

func (s *JournalService) fiscalYearStartMonth(ctx context.Context) int {
	return s.periods.fiscalYearStartMonth(ctx)
}
