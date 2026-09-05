package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Fremdwährung im laufenden Geschäft (BEW-10).
//
// Ein Beleg in Fremdwährung hat zwei Kurse und nicht einen. Der Tageskurs der
// Europäischen Zentralbank bewertet den Geschäftsvorfall — den Aufwand, den
// Erlös, die Forderung, die Verbindlichkeit. Der monatliche Durchschnittskurs
// des Bundesministeriums der Finanzen bewertet die Bemessungsgrundlage der
// Umsatzsteuer (§ 16 Abs. 6 UStG). Beide gelten für denselben Beleg, und die
// Differenz zwischen ihnen ist Kursaufwand oder Kursertrag — 6880 bzw. 4840.
//
// Und es gibt keinen Rückfallkurs. Vor dieser Änderung reichte der Belegweg die
// Währung an den Buchungskopf durch, wo der Kurs still auf 1,000000 stand: eine
// Buchung über 1.000 USD stand mit 1.000 € in den Büchern, in richtiger
// Größenordnung und mit falschem Wert. Wo kein Kurs zu holen ist, kommt jetzt
// ein Fehler; der Anwender trägt den Kurs mit seiner Quelle ein
// (CurrencyService.SaveRate).

// currencyConverter ist der Ausschnitt des Kursdienstes, den die Buchungswege
// brauchen.
type currencyConverter interface {
	Convert(ctx context.Context, currency, date string, foreign domain.Cents) (*Conversion, error)
}

// SetCurrencyConverter koppelt den Kursdienst an den Belegweg. Ohne ihn ist ein
// Beleg in Fremdwährung nicht buchbar — und das ist richtig so: er wäre sonst
// einer mit geratenem Kurs.
func (s *PostingService) SetCurrencyConverter(c currencyConverter) { s.currency = c }

// fxContext ist die Umrechnung eines Belegs.
type fxContext struct {
	currency string
	conv     *Conversion
}

// rateMicros ist der Tageskurs in Millionsteln.
func (f *fxContext) rateMicros() int64 { return f.conv.Rate.RateMicros }

// toEuro rechnet einen Fremdbetrag mit dem Tageskurs in Euro um.
func (f *fxContext) toEuro(foreign domain.Cents) domain.Cents {
	return domain.ConvertToEuro(foreign, f.rateMicros())
}

// toForeign rechnet einen Eurobetrag mit dem Tageskurs zurück.
//
// Das Ergebnis ist eine Ableitung und keine Eingabe — es steht deshalb nur an
// den Zeilen, deren Fremdbetrag sich nicht aus dem Beleg selbst ergibt: an den
// Steuerzeilen und an der Gegenzeile. Die Positionszeilen tragen den Betrag, den
// der Anwender erfasst hat.
func (f *fxContext) toForeign(euro domain.Cents) domain.Cents {
	return domain.MulRound(euro, f.rateMicros(), domain.RateScale)
}

// vatBase rechnet eine Bemessungsgrundlage vom Tageskurs auf den
// Umsatzsteuer-Durchschnittskurs um.
//
// Liegt für den Monat kein amtlicher Durchschnittskurs vor, bleibt es nach
// § 16 Abs. 6 Satz 1 UStG beim Tageskurs — dann gibt diese Rechnung den Wert
// unverändert zurück, und es entsteht keine Kursdifferenz.
func (f *fxContext) vatBase(euroAtDailyRate domain.Cents) domain.Cents {
	if f.conv.VatRate == nil || f.conv.VatRate.RateMicros <= 0 {
		return euroAtDailyRate
	}
	return domain.MulRound(euroAtDailyRate, f.rateMicros(), f.conv.VatRate.RateMicros)
}

// head trägt Kurs, Quelle und Kurstag in den Buchungskopf.
func (f *fxContext) head(entry *domain.JournalEntry) {
	entry.Currency = f.currency
	entry.ExchangeRateMicros = f.rateMicros()
	entry.ExchangeRateSource = f.conv.Rate.Source
	entry.ExchangeRateDate = f.conv.Rate.Date
}

// splitToEuro rechnet die Positionsbeträge um.
//
// Gerundet wird je Position, und der Rest fällt auf die letzte: die Summe der
// einzeln gerundeten Beträge weicht sonst von der Umrechnung der Gesamtsumme ab,
// und der Beleg stimmte um einen Cent nicht mit der Rechnung überein, die
// daneben liegt.
func (f *fxContext) splitToEuro(foreign []domain.Cents) []domain.Cents {
	out := make([]domain.Cents, len(foreign))
	var total, assigned domain.Cents
	for _, v := range foreign {
		total += v
	}
	euroTotal := f.toEuro(total)
	for i, v := range foreign {
		if i == len(foreign)-1 {
			out[i] = euroTotal - assigned
			break
		}
		out[i] = f.toEuro(v)
		assigned += out[i]
	}
	return out
}

// prepareCurrency baut die Umrechnung eines Eingangsbelegs, oder nil, wo in Euro
// gebucht wird.
//
// Die Positionsbeträge eines Fremdwährungsbelegs sind die der Fremdwährung: der
// Anwender tippt ab, was auf der Rechnung steht, und rechnet nicht selbst um.
// req.ForeignAmount ist die Kontrollsumme dazu — die Endsumme der Rechnung.
// Stimmt sie nicht mit den Positionen überein, ist irgendetwas falsch
// abgetippt, und das fällt hier auf und nicht in der Bilanz.
func (s *PostingService) prepareCurrency(ctx context.Context, req ReceiptRequest) (*fxContext, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Currency))
	if code == "" || code == "EUR" {
		if req.ForeignAmount != 0 {
			return nil, fmt.Errorf(
				"zu einem Fremdbetrag gehört die Währung, auf die er lautet")
		}
		return nil, nil
	}
	var total domain.Cents
	for _, p := range req.Positions {
		total += p.Net
	}
	if req.ForeignAmount != 0 && req.ForeignAmount != total {
		return nil, fmt.Errorf(
			"die Positionen ergeben zusammen %s %s, die Endsumme des Belegs lautet aber auf %s %s",
			total, code, req.ForeignAmount, code)
	}
	if total <= 0 {
		return nil, fmt.Errorf("der Beleg hat einen Gesamtbetrag von null")
	}
	if s.currency == nil {
		return nil, fmt.Errorf(
			"der Kursdienst ist nicht eingerichtet. Ein Beleg in %s ließe sich nur mit einem "+
				"geratenen Kurs buchen, und geraten wird keiner", code)
	}
	conv, err := s.currency.Convert(ctx, code, req.DocumentDate, total)
	if err != nil {
		return nil, fmt.Errorf(
			"für den Beleg in %s vom %s: %w", code, req.DocumentDate, err)
	}
	return &fxContext{currency: code, conv: conv}, nil
}

// taxLinesInCurrency baut die Steuerzeilen eines Fremdwährungsbelegs samt der
// Kursdifferenz.
//
// Gebucht wird die Steuer aus dem Durchschnittskurs des Monats — sie ist die,
// die in der Voranmeldung steht (§ 16 Abs. 6 UStG). Was der Lieferant
// tatsächlich fordert, folgt dem Tageskurs; die Differenz zwischen beiden ist
// weder Rundungsfehler noch Nachlässigkeit, sondern Kursaufwand oder
// Kursertrag, und sie bekommt ihre eigene Zeile.
func (s *PostingService) taxLinesInCurrency(
	dir domain.Direction, treatment domain.TaxTreatment,
	netByRate map[domain.TaxRate]domain.Cents, fx *fxContext,
) ([]domain.JournalLine, error) {
	if fx == nil {
		return s.taxLines(dir, treatment, netByRate)
	}

	vatByRate := make(map[domain.TaxRate]domain.Cents, len(netByRate))
	for rate, net := range netByRate {
		vatByRate[rate] = fx.vatBase(net)
	}
	booked, err := s.taxLines(dir, treatment, vatByRate)
	if err != nil {
		return nil, err
	}
	atDailyRate, err := s.taxLines(dir, treatment, netByRate)
	if err != nil {
		return nil, err
	}

	out := make([]domain.JournalLine, 0, len(booked)+1)
	for _, l := range booked {
		l.ForeignAmount = fx.toForeign(l.Amount)
		out = append(out, l)
	}

	// Die Gegenzeile ergibt sich aus dem, was die Buchung zum Ausgleich braucht.
	// Zu zahlen ist aber der Betrag zum Tageskurs — die Zeile hier trägt genau
	// den Unterschied, damit die Gegenzeile stimmt und die Steuer trotzdem die
	// des Durchschnittskurses bleibt.
	difference := signedSum(atDailyRate) - signedSum(booked)
	if difference == 0 {
		return out, nil
	}
	line := domain.JournalLine{
		Side: domain.SideDebit, Account: accounting.CurrencyLossAccount, Amount: difference,
		Text: currencyDifferenceText(fx),
	}
	if difference < 0 {
		line = domain.JournalLine{
			Side: domain.SideCredit, Account: accounting.CurrencyGainAccount, Amount: -difference,
			Text: currencyDifferenceText(fx),
		}
	}
	return append(out, line), nil
}

// currencyDifferenceText sagt in der Zeile selbst, woher der Betrag kommt.
func currencyDifferenceText(fx *fxContext) string {
	month := ""
	if fx.conv.VatRate != nil {
		month = fx.conv.VatRate.Month
	}
	return fmt.Sprintf(
		"Kursdifferenz Tageskurs/Umsatzsteuerkurs %s %s (§ 16 Abs. 6 UStG)", fx.currency, month)
}

// signedSum rechnet Soll positiv und Haben negativ — die Zahl, gegen die die
// Gegenzeile ausgeglichen wird.
func signedSum(lines []domain.JournalLine) domain.Cents {
	var out domain.Cents
	for _, l := range lines {
		if l.Side == domain.SideDebit {
			out += l.Amount
		} else {
			out -= l.Amount
		}
	}
	return out
}

// spreadForeign verteilt den Fremdbetrag einer Position auf die Zeilen, die aus
// ihr entstanden sind — der Rest auf die letzte, damit die Summe stimmt.
func spreadForeign(lines []domain.JournalLine, foreign domain.Cents) {
	if len(lines) == 0 || foreign == 0 {
		return
	}
	var total domain.Cents
	for _, l := range lines {
		total += l.Amount
	}
	if total <= 0 {
		lines[0].ForeignAmount = foreign
		return
	}
	assigned := domain.Cents(0)
	for i := range lines {
		if i == len(lines)-1 {
			lines[i].ForeignAmount = foreign - assigned
			break
		}
		lines[i].ForeignAmount = domain.MulRound(foreign, int64(lines[i].Amount), int64(total))
		assigned += lines[i].ForeignAmount
	}
}
