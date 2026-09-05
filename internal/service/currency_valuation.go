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

// Die Stichtagsbewertung der Fremdwährungsposten (§ 256a HGB).
//
// Auf Fremdwährung lautende Vermögensgegenstände und Verbindlichkeiten sind zum
// Abschlussstichtag mit dem Devisenkassamittelkurs umzurechnen. Satz 2 nimmt
// davon die Posten mit einer Restlaufzeit von einem Jahr oder weniger aus:
// bei ihnen gelten das Anschaffungskostenprinzip und das Realisationsprinzip
// nicht, ein Kursgewinn wird also erfolgswirksam. Bei längeren Laufzeiten bleibt
// es beim Imparitätsprinzip — der Verlust wird gebucht, der Gewinn nicht.
//
// Die Bewertung wird am ersten Tag des Folgejahres wieder aufgelöst. Sie ist
// eine Aussage über einen Stichtag und keine über den Posten; ohne die Auflösung
// stünde der bewertete Betrag im neuen Jahr weiter und der Zahlungsausgleich
// ergäbe eine Kursdifferenz, die es nie gab.
//
// Gebucht wird die Auflösung vom Saldenvortrag (ReverseInto), wie die Auflösung
// der Rechnungsabgrenzung: der Vortrag ist der Vorgang, der das neue Jahr
// eröffnet, und er fragt dabei das Journal statt ein Merkzeichen — damit ist
// jeder erneute Vortrag zugleich der Nachholweg für eine Auflösung, die beim
// ersten Mal ausgeblieben ist.

// SetOpenItemSource koppelt die Zahlungsverwaltung an den Kursdienst.
func (s *CurrencyService) SetOpenItemSource(src openItemSource) { s.openItems = src }

// SetJournalService koppelt den Buchungsweg an den Kursdienst.
func (s *CurrencyService) SetJournalService(j *JournalService) { s.journalSvc = j }

// SetClosingService koppelt den Abschluss an den Kursdienst — er kennt die
// Stichtage.
func (s *CurrencyService) SetClosingService(c *ClosingService) { s.closingSvc = c }

// PreviewCurrencyValuation rechnet die Stichtagsbewertung, ohne sie zu buchen.
func (s *CurrencyService) PreviewCurrencyValuation(ctx context.Context, year int) (*ForeignCurrencyValuation, error) {
	valuation, _, err := s.buildValuation(ctx, year)
	return valuation, err
}

// buildValuation sammelt die Fremdwährungsposten und bewertet sie.
func (s *CurrencyService) buildValuation(
	ctx context.Context, year int,
) (*ForeignCurrencyValuation, []domain.JournalLine, error) {
	if s.closingSvc == nil || s.openItems == nil || s.journal == nil {
		return nil, nil, fmt.Errorf("die Fremdwährungsbewertung ist nicht eingerichtet")
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, nil, err
	}
	cutoff := fy.EndDate
	reversal, err := dayAfterCutoff(cutoff)
	if err != nil {
		return nil, nil, err
	}

	out := &ForeignCurrencyValuation{
		FiscalYear:   year,
		Cutoff:       cutoff,
		ReversalDate: reversal,
		Items:        make([]ForeignCurrencyValuationItem, 0),
	}

	items, err := s.openItems.OpenItemsAt(ctx, cutoff)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.journal.FindOpenItemCandidatesAt(ctx, cutoff)
	if err != nil {
		return nil, nil, err
	}
	byEntry := make(map[uint]*domain.JournalEntry, len(entries))
	for i := range entries {
		byEntry[entries[i].ID] = &entries[i]
	}

	var lines []domain.JournalLine
	for i := range items {
		item := &items[i]
		entry := byEntry[item.EntryID]
		if entry == nil || entry.Currency == "" || entry.Currency == "EUR" {
			continue
		}
		if item.OpenAmount == 0 {
			continue
		}
		if entry.ExchangeRateMicros <= 0 {
			return nil, nil, fmt.Errorf(
				"die Buchung %s lautet auf %s, trägt aber keinen Umrechnungskurs. Ohne ihn lässt sich "+
					"der Fremdwährungsbetrag des offenen Postens nicht bestimmen",
				entry.EntryNumber, entry.Currency)
		}

		rate, err := s.RateAt(ctx, entry.Currency, cutoff)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"für die Bewertung des offenen Postens %s zum %s: %w",
				item.DocumentNumber, cutoff, err)
		}

		// Der Fremdwährungsbetrag des offenen Postens ergibt sich aus dem noch
		// offenen Eurobetrag und dem Kurs, mit dem gebucht wurde. Er wird nicht
		// gespeichert, weil eine Teilzahlung ihn ändert und ein gespeicherter
		// Wert dann still falsch stünde.
		foreign := domain.MulRound(item.OpenAmount, entry.ExchangeRateMicros, domain.RateScale)
		valued := domain.ConvertToEuro(foreign, rate.RateMicros)

		row := ForeignCurrencyValuationItem{
			Kind:    "open_item",
			EntryID: entry.ID, EntryNumber: entry.EntryNumber,
			Account: item.LedgerAccount, ContactID: item.ContactID,
			Description:   fmt.Sprintf("%s %s", item.DocumentNumber, item.ContactName),
			Currency:      entry.Currency,
			DueDate:       item.DueDate,
			ForeignAmount: foreign,
			BookValue:     item.OpenAmount,
			ValueAtCutoff: valued,
			Difference:    valued - item.OpenAmount,
			RateMicros:    rate.RateMicros,
		}
		if row.Difference == 0 {
			row.Reason = "Der Stichtagskurs entspricht dem Buchkurs; es ist nichts zu bewerten."
			out.Items = append(out.Items, row)
			continue
		}

		receivable := item.ContactType == domain.ContactTypeCustomer
		row.Gain = (receivable && row.Difference > 0) || (!receivable && row.Difference < 0)
		row.Amount = row.Difference.Abs()

		shortTerm := currencyShortTerm(item.DueDate, cutoff)
		switch {
		case !row.Gain:
			row.Recognised = true
			row.Reason = "Kursverlust. Er wird in jedem Fall erfasst — das Imparitätsprinzip des " +
				"§ 252 Abs. 1 Nr. 4 HGB kennt keine Restlaufzeit."
		case shortTerm:
			row.Recognised = true
			row.Reason = "Kursgewinn bei einer Restlaufzeit von höchstens einem Jahr. § 256a Satz 2 HGB " +
				"nimmt diese Posten vom Anschaffungskosten- und vom Realisationsprinzip aus; der Gewinn " +
				"ist deshalb erfolgswirksam."
		default:
			row.Reason = "Kursgewinn bei einer Restlaufzeit von mehr als einem Jahr. Er wird nicht " +
				"gebucht: die Ausnahme des § 256a Satz 2 HGB gilt für ihn nicht, und das " +
				"Realisationsprinzip lässt einen unrealisierten Gewinn nicht zu."
		}

		if row.Recognised {
			if row.Gain {
				out.TotalGain += row.Amount
				lines = append(lines,
					domain.JournalLine{Side: domain.SideDebit, Account: row.Account,
						ContactID: contactRef(row.ContactID), Amount: row.Amount, Text: row.Description},
					domain.JournalLine{Side: domain.SideCredit, Account: accounting.CurrencyGainAccount,
						Amount: row.Amount, Text: row.Description})
			} else {
				out.TotalLoss += row.Amount
				lines = append(lines,
					domain.JournalLine{Side: domain.SideDebit, Account: accounting.CurrencyLossAccount,
						Amount: row.Amount, Text: row.Description},
					domain.JournalLine{Side: domain.SideCredit, Account: row.Account,
						ContactID: contactRef(row.ContactID), Amount: row.Amount, Text: row.Description})
			}
		}
		out.Items = append(out.Items, row)
	}

	// Die Bankkonten in Fremdwährung gehören ebenso dazu: § 256a HGB nennt „auf
	// Fremdwährung lautende Vermögensgegenstände und Verbindlichkeiten" und meint
	// damit nicht nur die offenen Posten. Ein Dollarkonto, das zum Stichtag nicht
	// umgerechnet wird, steht mit dem Kurs des Tages in der Bilanz, an dem zuletzt
	// etwas darauf gebucht wurde.
	bankLines, err := s.bankValuation(ctx, cutoff, out)
	if err != nil {
		return nil, nil, err
	}
	lines = append(lines, bankLines...)

	sort.SliceStable(out.Items, func(i, j int) bool {
		if out.Items[i].Kind != out.Items[j].Kind {
			return out.Items[i].Kind < out.Items[j].Kind
		}
		return out.Items[i].EntryNumber < out.Items[j].EntryNumber
	})
	out.Note = fmt.Sprintf(
		"Bewertet wird zum %s mit dem Kurs des Stichtags. Buchfink nimmt dafür den EZB-Referenzkurs "+
			"als Näherung an den Devisenkassamittelkurs, den § 256a HGB nennt; die beiden weichen im "+
			"Regelfall nur um die Geld-Brief-Spanne voneinander ab. Die Bewertung wird am %s wieder "+
			"aufgelöst — sie gilt dem Stichtag und nicht dem Posten.",
		germanDay(cutoff), germanDay(reversal))
	return out, lines, nil
}

// BookCurrencyValuation bucht die Stichtagsbewertung.
//
// Nur sie — die Auflösung folgt mit dem Saldenvortrag ins Folgejahr, wie bei der
// Rechnungsabgrenzung (ClosingService.CarryForward → ReverseInto). Vorher wurden
// beide in einem Zug geschrieben, und das war der schlechtere Weg: scheiterte
// die zweite Buchung — weil das Folgejahr noch nicht lief oder festgestellt war
// —, stand die Bewertung ohne ihre Auflösung, und ein erneuter Lauf wurde mit
// „bereits gebucht" abgewiesen. Es gab keinen Weg zurück.
//
// Jetzt gibt es ihn, und er ist derselbe wie überall: der Vortrag ins neue Jahr
// bucht, was ins neue Jahr gehört, und er fragt dabei das Journal — läuft er ein
// zweites Mal, holt er die fehlende Auflösung nach.
func (s *CurrencyService) BookCurrencyValuation(ctx context.Context, year int) (*ForeignCurrencyValuation, error) {
	if s.journalSvc == nil {
		return nil, fmt.Errorf("der Buchungsweg ist nicht eingerichtet")
	}
	valuation, lines, err := s.buildValuation(ctx, year)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		valuation.Note = "Zum Stichtag steht kein offener Posten in Fremdwährung offen, der zu bewerten " +
			"wäre. " + valuation.Note
		return valuation, nil
	}

	document := foreignCurrencyDocument(year)
	existing, err := s.standingValuationEntries(ctx, year, document)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf(
			"die Fremdwährungsbewertung des Geschäftsjahres %d ist mit der Buchung %s bereits "+
				"gebucht. Nimm sie zurück, bevor du sie neu rechnest",
			year, existing[0].EntryNumber)
	}

	entry := &domain.JournalEntry{
		BookingDate: valuation.Cutoff, DocumentDate: valuation.Cutoff,
		ServiceDateFrom: valuation.Cutoff, ServiceDateTo: valuation.Cutoff,
		Description: fmt.Sprintf("Fremdwährungsbewertung zum %s (§ 256a HGB)",
			germanDay(valuation.Cutoff)),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     document,
		TaxTreatment:       domain.TaxTreatmentNotTaxable,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	posted, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}
	valuation.EntryNumber = posted.EntryNumber
	valuation.Note = fmt.Sprintf(
		"Die Auflösung zum %s wird mit dem Saldenvortrag in das Geschäftsjahr %d gebucht — wie die "+
			"Auflösung der Rechnungsabgrenzung. ", germanDay(valuation.ReversalDate), year+1) +
		valuation.Note

	s.audit(ctx, domain.AuditActionCreate, fmt.Sprintf(
		"Fremdwährungsbewertung %d gebucht: %s € Ertrag, %s € Aufwand (%s); Auflösung zum %s folgt "+
			"mit dem Saldenvortrag",
		year, valuation.TotalGain, valuation.TotalLoss, posted.EntryNumber, valuation.ReversalDate))
	return valuation, nil
}

// ReverseInto bucht die Auflösung der Bewertung des Vorjahres.
//
// Aufgerufen wird sie vom Saldenvortrag. Sie liest den Zustand aus dem Journal
// und führt kein eigenes Merkzeichen: gefragt wird, ob die Bewertungsbuchung des
// Vorjahres steht und ob im Zieljahr schon eine Auflösung dazu liegt. Damit ist
// jeder erneute Vortrag zugleich der Nachholweg — ein Zustand „bewertet, aber
// nicht aufgelöst", aus dem es keinen Ausweg gäbe, kann nicht entstehen.
func (s *CurrencyService) ReverseInto(ctx context.Context, toYear int) ([]domain.JournalEntry, error) {
	created := make([]domain.JournalEntry, 0, 1)
	if s.journalSvc == nil || s.journal == nil || toYear <= 1 {
		return created, nil
	}
	fromYear := toYear - 1
	document := foreignCurrencyDocument(fromYear)

	// Eine stornierte Bewertung wird nicht aufgelöst: die Generalumkehr hat sie
	// bereits zurückgenommen.
	valuations, err := s.standingValuationEntries(ctx, fromYear, document)
	if err != nil {
		return created, err
	}
	if len(valuations) == 0 {
		return created, nil
	}
	existing, err := s.standingValuationEntries(ctx, toYear, document)
	if err != nil {
		return created, err
	}
	if len(existing) > 0 {
		return created, nil
	}

	reversalDate, err := dayAfterCutoff(valuations[0].BookingDate)
	if err != nil {
		return created, err
	}
	for i := range valuations {
		valuation := &valuations[i]
		lines := make([]domain.JournalLine, 0, len(valuation.Lines))
		for _, l := range valuation.Lines {
			lines = append(lines, domain.JournalLine{
				Side: l.Side.Opposite(), Account: l.Account, ContactID: l.ContactID,
				Amount: l.Amount, Text: l.Text,
			})
		}
		if len(lines) == 0 {
			continue
		}
		posted, err := s.journalSvc.Post(ctx, &domain.JournalEntry{
			BookingDate: reversalDate, DocumentDate: reversalDate,
			ServiceDateFrom: reversalDate, ServiceDateTo: reversalDate,
			Description: fmt.Sprintf("Auflösung der Fremdwährungsbewertung zum %s",
				germanDay(valuation.BookingDate)),
			Source:             domain.EntrySourceOpening,
			DocumentNumber:     document,
			TaxTreatment:       domain.TaxTreatmentNotTaxable,
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              lines,
		})
		if err != nil {
			return created, fmt.Errorf(
				"die Bewertung %s ließ sich zum %s nicht auflösen. Ohne die Auflösung steht der "+
					"bewertete Betrag im neuen Jahr weiter, und der Zahlungsausgleich ergäbe eine "+
					"Kursdifferenz, die es nie gab: %w",
				valuation.EntryNumber, germanDay(reversalDate), err)
		}
		created = append(created, *posted)
	}
	if len(created) > 0 {
		s.audit(ctx, domain.AuditActionCreate, fmt.Sprintf(
			"Fremdwährungsbewertung %d zum %s aufgelöst: %d Buchung(en)",
			fromYear, reversalDate, len(created)))
	}
	return created, nil
}

// standingValuationEntries sind die Bewertungsbuchungen eines Jahres, die noch
// stehen — die stornierten sind heraus.
//
// Gefragt wird über FindReversalOf und nicht über die Generalumkehren desselben
// Geschäftsjahres: der Storno trägt den Tag seiner Erstellung, und die
// Generalumkehr einer Bewertung zum 31.12. liegt fast immer im Folgejahr. Wer
// nur das Jahr der Bewertung durchsähe, fände sie nie.
//
// Ohne diese Frage wäre der Weg zurück eine Sackgasse: BookCurrencyValuation
// wies mit „bereits gebucht — nimm sie zurück" ab, die Schrittliste meldete
// „storniert, offen", und die Bewertung ließ sich für dieses Jahr nie wieder
// buchen.
func (s *CurrencyService) standingValuationEntries(
	ctx context.Context, year int, document string,
) ([]domain.JournalEntry, error) {
	entries, err := s.findValuationEntries(ctx, year, document)
	if err != nil {
		return nil, err
	}
	index := newReversalIndex(s.journal)
	out := make([]domain.JournalEntry, 0, len(entries))
	for i := range entries {
		voided, err := index.reversed(ctx, &entries[i].ID)
		if err != nil {
			return nil, err
		}
		if voided {
			continue
		}
		out = append(out, entries[i])
	}
	return out, nil
}

// findValuationEntries sucht die Buchungen einer Bewertung im Journal.
func (s *CurrencyService) findValuationEntries(
	ctx context.Context, year int, document string,
) ([]domain.JournalEntry, error) {
	entries, err := s.journal.FindAll(ctx, year)
	if err != nil {
		return nil, err
	}
	out := make([]domain.JournalEntry, 0, 2)
	for i := range entries {
		if entries[i].DocumentNumber == document && entries[i].Kind != domain.EntryKindReversal {
			out = append(out, entries[i])
		}
	}
	return out, nil
}

// currencyShortTerm meldet, ob ein Posten am Stichtag eine Restlaufzeit von
// höchstens einem Jahr hat.
//
// Ein Posten ohne Fälligkeit ist kurzfristig und nicht langfristig: eine
// Forderung, für die kein Zahlungsziel vereinbart wurde, ist sofort fällig.
func currencyShortTerm(dueDate, cutoff string) bool {
	if dueDate == "" {
		return true
	}
	due, err1 := time.Parse("2006-01-02", dueDate)
	at, err2 := time.Parse("2006-01-02", cutoff)
	if err1 != nil || err2 != nil {
		return true
	}
	return !due.After(at.AddDate(1, 0, 0))
}

// dayAfterCutoff liefert den Folgetag eines ISO-Datums.
func dayAfterCutoff(date string) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", fmt.Errorf("%q ist kein Datum im Format JJJJ-MM-TT", date)
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02"), nil
}

// germanDay rendert ein ISO-Datum in der Form, die ein deutscher Leser erwartet.
func germanDay(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
}

// contactRef liefert den Zeiger auf eine Kontakt-Kennung, oder nil bei null.
func contactRef(id uint) *uint {
	if id == 0 {
		return nil
	}
	return &id
}

// audit schreibt einen Eintrag ins Protokoll.
func (s *CurrencyService) audit(ctx context.Context, action domain.AuditAction, details string) {
	if s.auditLog == nil {
		return
	}
	_ = s.auditLog.Log(ctx, action, "CURRENCY", "", details)
}

// -------------------------------------------------------------------------
// Bankkonten in Fremdwährung
// -------------------------------------------------------------------------

// foreignBankBalance ist der Bestand eines Zahlungsmittelkontos in einer
// Fremdwährung.
type foreignBankBalance struct {
	account   string
	currency  string
	bookValue domain.Cents
	foreign   domain.Cents
}

// bankValuation bewertet die Zahlungsmittelkonten in Fremdwährung zum Stichtag
// und hängt die Zeilen an die Bewertung.
//
// Ein Bankguthaben hat keine Restlaufzeit — es ist jederzeit fällig. Damit greift
// die Ausnahme des § 256a Satz 2 HGB immer, und Gewinn wie Verlust sind
// erfolgswirksam.
func (s *CurrencyService) bankValuation(
	ctx context.Context, cutoff string, out *ForeignCurrencyValuation,
) ([]domain.JournalLine, error) {
	balances, err := s.foreignBankBalances(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	var lines []domain.JournalLine
	for _, b := range balances {
		rate, err := s.RateAt(ctx, b.currency, cutoff)
		if err != nil {
			return nil, fmt.Errorf(
				"für die Bewertung des Kontos %s (%s) zum %s: %w", b.account, b.currency, cutoff, err)
		}
		valued := domain.ConvertToEuro(b.foreign, rate.RateMicros)
		row := ForeignCurrencyValuationItem{
			Kind:          "bank",
			EntryNumber:   b.account,
			Account:       b.account,
			Description:   fmt.Sprintf("Bankkonto %s in %s", b.account, b.currency),
			Currency:      b.currency,
			ForeignAmount: b.foreign,
			BookValue:     b.bookValue,
			ValueAtCutoff: valued,
			Difference:    valued - b.bookValue,
			RateMicros:    rate.RateMicros,
		}
		if row.Difference == 0 {
			row.Reason = "Der Stichtagskurs entspricht dem Buchkurs; es ist nichts zu bewerten."
			out.Items = append(out.Items, row)
			continue
		}
		row.Gain = row.Difference > 0
		row.Amount = row.Difference.Abs()
		row.Recognised = true
		if row.Gain {
			row.Reason = "Kursgewinn auf einem Fremdwährungskonto. Ein Guthaben ist jederzeit fällig " +
				"und damit stets kurzfristig; § 256a Satz 2 HGB nimmt es vom Anschaffungskosten- und " +
				"vom Realisationsprinzip aus, der Gewinn ist deshalb erfolgswirksam."
			out.TotalGain += row.Amount
			lines = append(lines,
				domain.JournalLine{Side: domain.SideDebit, Account: b.account,
					Amount: row.Amount, ForeignAmount: 0, Text: row.Description},
				domain.JournalLine{Side: domain.SideCredit, Account: accounting.CurrencyGainAccount,
					Amount: row.Amount, Text: row.Description})
		} else {
			row.Reason = "Kursverlust auf einem Fremdwährungskonto. Er wird in jedem Fall erfasst."
			out.TotalLoss += row.Amount
			lines = append(lines,
				domain.JournalLine{Side: domain.SideDebit, Account: accounting.CurrencyLossAccount,
					Amount: row.Amount, Text: row.Description},
				domain.JournalLine{Side: domain.SideCredit, Account: b.account,
					Amount: row.Amount, ForeignAmount: 0, Text: row.Description})
		}
		out.Items = append(out.Items, row)
	}
	return lines, nil
}

// foreignBankBalances rechnet je Zahlungsmittelkonto und Währung zusammen, was
// bis zum Stichtag darauf gebucht wurde.
//
// Gezählt werden die Fremdwährungsbuchungen aller Geschäftsjahre bis zum
// Stichtag — der Fremdwährungsbestand eines Kontos ist die Summe seiner
// Bewegungen und beginnt nicht mit dem Geschäftsjahr neu. Zwei Arten von
// Buchungen bleiben draußen: die Eröffnungsbuchungen, weil sie in Euro
// wiederholen, was die Bewegungen darunter schon tragen, und die Bewertungen
// selbst samt ihren Auflösungen — sie ändern den Eurobuchwert und nicht den
// Bestand, und wer sie mitzählte, bewertete die Bewertung des Vorjahres.
func (s *CurrencyService) foreignBankBalances(
	ctx context.Context, cutoff string,
) ([]foreignBankBalance, error) {
	years, err := s.journal.GetAvailableFiscalYears(ctx)
	if err != nil {
		return nil, err
	}
	byKey := map[string]*foreignBankBalance{}
	order := make([]string, 0, 4)
	for _, year := range years {
		entries, err := s.journal.FindAll(ctx, year)
		if err != nil {
			return nil, err
		}
		for i := range entries {
			entry := &entries[i]
			if entry.BookingDate > cutoff {
				continue
			}
			if entry.Currency == "" || entry.Currency == "EUR" {
				continue
			}
			if entry.Source == domain.EntrySourceOpening {
				continue
			}
			if strings.HasPrefix(entry.DocumentNumber, foreignCurrencyDocumentPrefix) {
				continue
			}
			for _, l := range entry.Lines {
				if !isLiquidAccount(l.Account) || l.ForeignAmount == 0 {
					continue
				}
				key := l.Account + "/" + entry.Currency
				balance, ok := byKey[key]
				if !ok {
					balance = &foreignBankBalance{account: l.Account, currency: entry.Currency}
					byKey[key] = balance
					order = append(order, key)
				}
				sign := domain.Cents(1)
				if l.Side == domain.SideCredit {
					sign = -1
				}
				balance.bookValue += sign * l.Amount
				balance.foreign += sign * l.ForeignAmount
			}
		}
	}
	sort.Strings(order)
	out := make([]foreignBankBalance, 0, len(order))
	for _, key := range order {
		if byKey[key].foreign == 0 {
			continue
		}
		out = append(out, *byKey[key])
	}
	return out, nil
}
