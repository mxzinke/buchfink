package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// CurrencyService führt die Umrechnungskurse.
//
// Er tut zwei Dinge, die auseinandergehalten werden müssen. Der Tageskurs der
// Europäischen Zentralbank bewertet den Geschäftsvorfall — den Aufwand, die
// Forderung, den Bankbestand. Der monatliche Durchschnittskurs des
// Bundesministeriums der Finanzen bewertet die Bemessungsgrundlage der
// Umsatzsteuer (§ 16 Abs. 6 UStG). Beide gelten für denselben Beleg, und die
// Differenz zwischen ihnen ist Kursaufwand oder Kursertrag — kein Rundungsfehler
// und keine Nachlässigkeit.
//
// Und er rät nie. Fehlt ein Kurs und ist das Netz nicht da, kommt ein Fehler und
// kein Wert; der Anwender trägt den Kurs mit seiner Quelle ein.
type CurrencyService struct {
	rates      domain.ExchangeRateRepository
	fetcher    domain.CurrencyFetcher
	journal    domain.JournalRepository
	auditLog   domain.AuditRepository
	openItems  openItemSource
	journalSvc *JournalService
	closingSvc *ClosingService
	// storedOnly schaltet den Dienst auf reines Lesen: kein Abruf beim
	// Kursdienst, kein Speichern. Für den Prüfermodus — siehe ReadOnly.
	storedOnly bool
}

// SettingExchangeRateEndpoint ist der Schlüssel der einstellbaren Adresse des
// Kursdienstes — das Gegenstück zu SettingVatIDEndpoint.
//
// Als Konstante und nicht als Zeichenkette an der einen Stelle, an der sie
// gelesen wird: die Oberfläche stellt beide Adressen ein, und ein Schlüssel, den
// sie raten muss, geht irgendwann an der Einstellung vorbei — ohne dass es
// jemandem auffiele, weil ein fehlender Wert die Voreinstellung bedeutet.
const SettingExchangeRateEndpoint = "exchange_rate_endpoint"

// ReadOnly liefert denselben Dienst, der keinen Kurs mehr holt und keinen mehr
// speichert.
//
// Er ist der Weg, im Prüfermodus eine Stichtagsbewertung anzusehen. Sie braucht
// den Kurs des Stichtags, und der wird sonst beim ersten Ansehen geholt und
// abgelegt — ein Schreibvorgang, den der Prüfermodus zu Recht abweist. Für ein
// abgeschlossenes Jahr stehen die Kurse längst in der Historie; fehlt einer,
// sagt der Dienst das, statt still einen anderen zu nehmen.
func (s *CurrencyService) ReadOnly() *CurrencyService {
	if s == nil {
		return nil
	}
	out := *s
	out.storedOnly = true
	return &out
}

// NewCurrencyService wires den Kursdienst.
func NewCurrencyService(
	rates domain.ExchangeRateRepository,
	fetcher domain.CurrencyFetcher,
	auditLog domain.AuditRepository,
) *CurrencyService {
	return &CurrencyService{rates: rates, fetcher: fetcher, auditLog: auditLog}
}

// SetJournalRepo gibt dem Dienst das Journal für die Stichtagsbewertung.
func (s *CurrencyService) SetJournalRepo(r domain.JournalRepository) { s.journal = r }

// -------------------------------------------------------------------------
// Tageskurse
// -------------------------------------------------------------------------

// RateAt liefert den Kurs eines Tages.
//
// Der Weg ist immer derselbe: erst die eigene Historie zum Tag, dann der
// Kursdienst, dann — und nur dann — der jüngste ältere Kurs aus der Historie,
// deutlich als solcher benannt. Ein Kurs vom Vortag ist eine Näherung, die man
// bewusst nimmt; einer, den man nicht als solche erkennt, ist eine Fälschung.
func (s *CurrencyService) RateAt(ctx context.Context, currency, date string) (*domain.ExchangeRate, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" || currency == "EUR" {
		return nil, fmt.Errorf("der Euro ist die Buchwährung; für ihn gibt es keinen Umrechnungskurs")
	}
	if len(date) != 10 {
		return nil, fmt.Errorf("das Kursdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if s.rates == nil {
		return nil, fmt.Errorf("die Kurshistorie ist nicht eingerichtet")
	}

	if stored, err := s.rates.FindRate(ctx, currency, date); err != nil {
		return nil, err
	} else if stored != nil {
		return stored, nil
	}

	var fetchErr error
	if s.fetcher != nil && !s.storedOnly {
		micros, actual, source, err := s.fetcher.RateAt(ctx, currency, date)
		if err == nil {
			// Gespeichert wird unter dem angefragten Tag *und*, wo er abweicht,
			// unter dem Tag der Feststellung. Der erste macht den Beleg vom
			// Sonntag buchbar, der zweite hält die Historie richtig.
			rate := &domain.ExchangeRate{
				Currency: currency, Date: date, RateMicros: micros, Source: source,
			}
			if err := s.rates.SaveRate(ctx, rate); err != nil {
				return nil, fmt.Errorf("der Kurs ließ sich nicht speichern: %w", err)
			}
			if actual != "" && actual != date {
				_ = s.rates.SaveRate(ctx, &domain.ExchangeRate{
					Currency: currency, Date: actual, RateMicros: micros, Source: source,
				})
			}
			return rate, nil
		}
		fetchErr = err
	}

	if earlier, err := s.rates.FindRateOnOrBefore(ctx, currency, date); err == nil && earlier != nil {
		out := *earlier
		out.Date = date
		out.Source = fmt.Sprintf(
			"Näherung: zuletzt festgestellter Kurs vom %s (%s)", earlier.Date, earlier.Source)
		return &out, nil
	}

	if fetchErr != nil {
		return nil, fetchErr
	}
	if s.storedOnly {
		return nil, fmt.Errorf(
			"für %s liegt zum %s kein gespeicherter Kurs vor. Solange der Prüfermodus läuft, holt "+
				"Buchfink keinen nach — ein geholter Kurs wäre eine Änderung am Datenbestand",
			currency, date)
	}
	return nil, fmt.Errorf(
		"für %s liegt zum %s kein Kurs vor und es ist kein Kursdienst eingerichtet. Trage den Kurs "+
			"mit seiner Quelle von Hand ein — Buchfink rät keinen", currency, date)
}

// SaveRate nimmt einen von Hand erfassten Kurs auf.
func (s *CurrencyService) SaveRate(ctx context.Context, rate domain.ExchangeRate) (*domain.ExchangeRate, error) {
	if s.rates == nil {
		return nil, fmt.Errorf("die Kurshistorie ist nicht eingerichtet")
	}
	rate.Currency = strings.ToUpper(strings.TrimSpace(rate.Currency))
	rate.Manual = true
	if strings.TrimSpace(rate.Source) == "" {
		return nil, fmt.Errorf(
			"zu einem von Hand erfassten Kurs gehört seine Quelle — etwa „EZB-Referenzkurs, " +
				"abgelesen am 03.03.2026\". Ohne sie ist der Kurs später nicht prüfbar")
	}
	if err := rate.Validate(); err != nil {
		return nil, err
	}
	if err := s.rates.SaveRate(ctx, &rate); err != nil {
		return nil, fmt.Errorf("der Kurs ließ sich nicht speichern: %w", err)
	}
	s.audit(ctx, domain.AuditActionCreate, fmt.Sprintf(
		"Kurs %s zum %s von Hand erfasst: %.6f (%s)",
		rate.Currency, rate.Date, rate.Rate(), rate.Source))
	return &rate, nil
}

// Rates liefert die Kurshistorie einer Währung in einem Zeitraum.
func (s *CurrencyService) Rates(ctx context.Context, currency, from, to string) ([]domain.ExchangeRate, error) {
	if s.rates == nil {
		return []domain.ExchangeRate{}, nil
	}
	return s.rates.FindRates(ctx, strings.ToUpper(strings.TrimSpace(currency)), from, to)
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Durchschnittskurse (§ 16 Abs. 6 UStG)
// -------------------------------------------------------------------------

// VatRateFor liefert den Durchschnittskurs eines Monats oder nil.
//
// Nil ist ein gültiges Ergebnis und kein Fehler: liegt kein amtlicher
// Durchschnittskurs vor, wird nach § 16 Abs. 6 Satz 1 UStG mit dem Tageskurs
// gerechnet. Der Aufrufer sagt das dann dazu.
func (s *CurrencyService) VatRateFor(ctx context.Context, currency, date string) (*domain.VatExchangeRate, error) {
	if s.rates == nil {
		return nil, nil
	}
	month := domain.MonthOf(date)
	if month == "" {
		return nil, fmt.Errorf("ohne Datum lässt sich der Monat des Durchschnittskurses nicht bestimmen")
	}
	return s.rates.FindVatRate(ctx, strings.ToUpper(strings.TrimSpace(currency)), month)
}

// VatRates liefert die erfassten Durchschnittskurse eines Zeitraums.
func (s *CurrencyService) VatRates(ctx context.Context, from, to string) ([]domain.VatExchangeRate, error) {
	if s.rates == nil {
		return []domain.VatExchangeRate{}, nil
	}
	return s.rates.FindVatRates(ctx, from, to)
}

// SaveVatRate nimmt einen Durchschnittskurs auf.
func (s *CurrencyService) SaveVatRate(ctx context.Context, rate domain.VatExchangeRate) (*domain.VatExchangeRate, error) {
	if s.rates == nil {
		return nil, fmt.Errorf("die Kurstabelle ist nicht eingerichtet")
	}
	rate.Currency = strings.ToUpper(strings.TrimSpace(rate.Currency))
	if strings.TrimSpace(rate.Source) == "" {
		rate.Source = "BMF-Umsatzsteuer-Umrechnungskurse"
	}
	if err := rate.Validate(); err != nil {
		return nil, err
	}
	if err := s.rates.SaveVatRate(ctx, &rate); err != nil {
		return nil, fmt.Errorf("der Durchschnittskurs ließ sich nicht speichern: %w", err)
	}
	return &rate, nil
}

// VatRateImport ist das Ergebnis eines CSV-Imports.
type VatRateImport struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Problems []string `json:"problems"`
}

// ImportVatRatesCSV liest die Umsatzsteuer-Umrechnungskurse aus einer CSV-Datei.
//
// Erwartet werden drei Spalten: Monat (JJJJ-MM), Währung, Kurs. Eine Kopfzeile
// wird erkannt und übergangen. Das Format ist absichtlich das einfachste
// mögliche — die amtliche Veröffentlichung ist ein PDF, und was daraus wird,
// tippt oder kopiert jemand. Ein Importer, der ein Amtsformat zu parsen
// versucht, das jedes Jahr anders aussieht, ist ein Importer, der jedes Jahr
// bricht.
func (s *CurrencyService) ImportVatRatesCSV(ctx context.Context, path string) (*VatRateImport, error) {
	if s.rates == nil {
		return nil, fmt.Errorf("die Kurstabelle ist nicht eingerichtet")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("die Datei %s ließ sich nicht öffnen: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	// Semikolon ist die Trennung, die aus einer deutschen Tabellenkalkulation
	// kommt; das Komma wird unten aufgefangen, wo eine Zeile nur ein Feld hat.
	reader.Comma = ';'

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("die Datei %s ist keine lesbare CSV: %w", path, err)
	}

	out := &VatRateImport{Problems: make([]string, 0)}
	for i, row := range rows {
		if len(row) == 1 && strings.Contains(row[0], ",") {
			row = strings.Split(row[0], ",")
		}
		if len(row) < 3 {
			if strings.TrimSpace(strings.Join(row, "")) != "" {
				out.Skipped++
				out.Problems = append(out.Problems, fmt.Sprintf(
					"Zeile %d: erwartet werden Monat, Währung und Kurs", i+1))
			}
			continue
		}
		month := strings.TrimSpace(row[0])
		currency := strings.ToUpper(strings.TrimSpace(row[1]))
		rateText := strings.TrimSpace(row[2])
		micros, err := parseExchangeRateMicros(rateText)
		if err != nil {
			// Die Kopfzeile fällt genau hier durch, ohne dass sie erkannt werden
			// müsste: „Kurs" ist keine Zahl.
			if i == 0 {
				continue
			}
			out.Skipped++
			out.Problems = append(out.Problems, fmt.Sprintf("Zeile %d: %v", i+1, err))
			continue
		}
		rate := domain.VatExchangeRate{
			Month: month, Currency: currency, RateMicros: micros,
			Source: "BMF-Umsatzsteuer-Umrechnungskurse, importiert aus " + path,
		}
		if err := rate.Validate(); err != nil {
			out.Skipped++
			out.Problems = append(out.Problems, fmt.Sprintf("Zeile %d: %v", i+1, err))
			continue
		}
		if err := s.rates.SaveVatRate(ctx, &rate); err != nil {
			out.Skipped++
			out.Problems = append(out.Problems, fmt.Sprintf("Zeile %d: %v", i+1, err))
			continue
		}
		out.Imported++
	}
	s.audit(ctx, domain.AuditActionImport, fmt.Sprintf(
		"Umsatzsteuer-Umrechnungskurse importiert: %d übernommen, %d übergangen (%s)",
		out.Imported, out.Skipped, path))
	return out, nil
}

// parseExchangeRateMicros liest einen Kurs in deutscher oder englischer Schreibweise.
func parseExchangeRateMicros(text string) (int64, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(text), " ", "")
	// Deutsche Schreibweise: Punkt trennt Tausender, Komma die Nachkommastellen.
	if strings.Contains(cleaned, ",") {
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("%q ist kein Kurs", text)
	}
	if value <= 0 {
		return 0, fmt.Errorf("der Kurs %q muss größer als null sein", text)
	}
	return int64(value*float64(domain.RateScale) + 0.5), nil
}

// -------------------------------------------------------------------------
// Umrechnung eines Belegs
// -------------------------------------------------------------------------

// Conversion ist die Umrechnung eines Fremdwährungsbelegs.
type Conversion struct {
	Currency string `json:"currency"`
	Date     string `json:"date"`

	// Rate ist der Tageskurs, mit dem der Aufwand bewertet wird.
	Rate domain.ExchangeRate `json:"rate"`
	// VatRate ist der Durchschnittskurs, mit dem die Bemessungsgrundlage der
	// Umsatzsteuer gerechnet wird. Nil heißt: es liegt keiner vor, und es bleibt
	// beim Tageskurs.
	VatRate *domain.VatExchangeRate `json:"vatRate,omitempty"`

	// ForeignAmount ist der Betrag in Fremdwährung, Amount sein Gegenwert zum
	// Tageskurs, TaxBaseAmount sein Gegenwert zum Umsatzsteuerkurs.
	ForeignAmount domain.Cents `json:"foreignAmount"`
	Amount        domain.Cents `json:"amount"`
	TaxBaseAmount domain.Cents `json:"taxBaseAmount"`
	// Difference ist die Differenz zwischen beiden. Sie ist Kursaufwand oder
	// Kursertrag und geht auf 6880 bzw. 4840.
	Difference domain.Cents `json:"difference"`
	Note       string       `json:"note"`
}

// Convert rechnet einen Fremdwährungsbetrag zu einem Belegdatum um.
func (s *CurrencyService) Convert(
	ctx context.Context, currency, date string, foreign domain.Cents,
) (*Conversion, error) {
	rate, err := s.RateAt(ctx, currency, date)
	if err != nil {
		return nil, err
	}
	out := &Conversion{
		Currency:      strings.ToUpper(strings.TrimSpace(currency)),
		Date:          date,
		Rate:          *rate,
		ForeignAmount: foreign,
		Amount:        domain.ConvertToEuro(foreign, rate.RateMicros),
	}
	out.TaxBaseAmount = out.Amount

	vatRate, err := s.VatRateFor(ctx, currency, date)
	if err != nil {
		return nil, err
	}
	if vatRate != nil {
		out.VatRate = vatRate
		out.TaxBaseAmount = domain.ConvertToEuro(foreign, vatRate.RateMicros)
		out.Difference = out.TaxBaseAmount - out.Amount
		out.Note = fmt.Sprintf(
			"Der Aufwand wird mit dem Tageskurs vom %s bewertet, die Bemessungsgrundlage der "+
				"Umsatzsteuer mit dem Durchschnittskurs des Monats %s (§ 16 Abs. 6 UStG). Die Differenz "+
				"von %s € ist Kursaufwand oder Kursertrag.",
			rate.Date, vatRate.Month, out.Difference.Abs())
		return out, nil
	}
	out.Note = fmt.Sprintf(
		"Für %s liegt im Monat %s kein amtlicher Durchschnittskurs vor. Nach § 16 Abs. 6 Satz 1 UStG "+
			"wird dann mit dem Tageskurs gerechnet — auch für die Umsatzsteuer.",
		out.Currency, domain.MonthOf(date))
	return out, nil
}

// -------------------------------------------------------------------------
// Stichtagsbewertung (§ 256a HGB)
// -------------------------------------------------------------------------

// ForeignCurrencyValuationItem ist ein zu bewertender Posten in Fremdwährung.
type ForeignCurrencyValuationItem struct {
	// Kind trennt die beiden Arten, die § 256a HGB gleich behandelt und die die
	// Ansicht auseinanderhalten muss: „open_item" ist eine Forderung oder
	// Verbindlichkeit, „bank" ein Guthaben auf einem Fremdwährungskonto.
	Kind          string       `json:"kind"`
	EntryID       uint         `json:"entryId,omitempty"`
	EntryNumber   string       `json:"entryNumber"`
	Account       string       `json:"account"`
	ContactID     uint         `json:"contactId,omitempty"`
	Description   string       `json:"description"`
	Currency      string       `json:"currency"`
	DueDate       string       `json:"dueDate,omitempty"`
	ForeignAmount domain.Cents `json:"foreignAmount"`
	// BookValue ist der bisher gebuchte Eurobetrag, ValueAtCutoff sein Wert zum
	// Stichtagskurs.
	BookValue     domain.Cents `json:"bookValue"`
	ValueAtCutoff domain.Cents `json:"valueAtCutoff"`
	// Difference ist die Wertänderung des Postens: positiv, wo der Posten mehr
	// wert ist als gebucht.
	Difference domain.Cents `json:"difference"`
	// Gain sagt, ob die Änderung erfolgswirksam als Ertrag wirkt. Bei einer
	// Verbindlichkeit ist ein höherer Eurowert ein Aufwand, bei einer Forderung
	// ein Ertrag — dieselbe Zahl, entgegengesetzte Bedeutung.
	Gain bool `json:"gain"`
	// Recognised sagt, ob die Änderung gebucht wird. Ein Gewinn aus einem Posten
	// mit mehr als einem Jahr Restlaufzeit wird nicht gebucht: § 256a Satz 2 HGB
	// nimmt nur die kurzfristigen vom Realisationsprinzip aus.
	Recognised bool         `json:"recognised"`
	Amount     domain.Cents `json:"amount"`
	Reason     string       `json:"reason"`
	// RateMicros ist der Stichtagskurs, mit dem gerechnet wurde.
	RateMicros int64 `json:"rateMicros"`
}

// ForeignCurrencyValuation ist die Stichtagsbewertung eines Geschäftsjahres.
type ForeignCurrencyValuation struct {
	FiscalYear int    `json:"fiscalYear"`
	Cutoff     string `json:"cutoff"`
	// ReversalDate ist der Tag, an dem die Bewertung wieder aufgelöst wird: der
	// erste Tag des Folgejahres.
	ReversalDate string                         `json:"reversalDate"`
	Items        []ForeignCurrencyValuationItem `json:"items"`
	// TotalGain und TotalLoss sind die gebuchten Beträge.
	TotalGain domain.Cents `json:"totalGain"`
	TotalLoss domain.Cents `json:"totalLoss"`
	Note      string       `json:"note"`
	// EntryNumber und ReversalEntryNumber sind nach dem Buchen belegt.
	EntryNumber         string `json:"entryNumber,omitempty"`
	ReversalEntryNumber string `json:"reversalEntryNumber,omitempty"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (v *ForeignCurrencyValuation) EnsureLists() {
	if v.Items == nil {
		v.Items = make([]ForeignCurrencyValuationItem, 0)
	}
}

// foreignCurrencyDocument ist die Belegnummer, unter der die Bewertung und
// ihre Auflösung im Journal stehen. Sie ist der Faden, an dem die Auflösung die
// Bewertung wiederfindet.
func foreignCurrencyDocument(fiscalYear int) string {
	return foreignCurrencyDocumentPrefix + fmt.Sprintf("%d", fiscalYear)
}

// foreignCurrencyDocumentPrefix erkennt die Buchungen der Bewertung wieder — die
// Bestandsrechnung der Fremdwährungskonten muss sie überspringen.
const foreignCurrencyDocumentPrefix = "FX-"

// openItemSource ist der Ausschnitt der Zahlungsverwaltung, den die
// Stichtagsbewertung braucht: welche Posten waren am Stichtag offen.
type openItemSource interface {
	OpenItemsAt(ctx context.Context, cutoff string) ([]domain.OpenItem, error)
}
