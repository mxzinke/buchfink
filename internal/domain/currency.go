package domain

import (
	"context"
	"fmt"
	"time"
)

// RateScale ist der Nenner eines Umrechnungskurses: Kurse werden als ganze Zahl
// in Millionsteln geführt.
//
// Eine Fließkommazahl wäre der naheliegende Typ und der falsche. Der Kurs geht
// in jede Umrechnung ein, die Umrechnungen werden summiert, und die Summe muss
// derselbe Betrag sein, den auch der Buchungssatz trägt — mit Fließkomma driftet
// sie. Die Buchung führt den Kurs aus demselben Grund als ExchangeRateMicros.
const RateScale = 1_000_000

// ExchangeRate ist ein Umrechnungskurs zu einem Tag.
//
// Der Kurs ist datiert und historisch: § 16 Abs. 6 UStG und § 256a HGB fragen
// nach dem Kurs eines bestimmten Tages, nicht nach dem heutigen. Die frühere
// Fassung dieses Typs hatte die Währung als einzigen Schlüssel und konnte
// deshalb nur einen Kurs je Währung halten — sie war eine Zwischenablage und
// keine Historie.
type ExchangeRate struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// Currency und Date bilden zusammen den fachlichen Schlüssel.
	Currency string `gorm:"size:3;not null;index:idx_exchange_rate_day,unique" json:"currency"`
	Date     string `gorm:"size:10;not null;index:idx_exchange_rate_day,unique" json:"date"`

	// RateMicros ist der Kurs in Millionsteln: 1 EUR = RateMicros / 1.000.000
	// Einheiten der Fremdwährung.
	RateMicros int64 `gorm:"not null" json:"rateMicros"`
	// Source nennt die Herkunft — „EZB-Referenzkurs", „manuell erfasst" samt
	// Begründung. Ein Kurs ohne Quelle ist eine Behauptung: er entscheidet über
	// den Aufwand, und wer ihn später prüft, muss wissen, woher er stammt.
	Source string `gorm:"size:255;not null" json:"source"`
	// Manual kennzeichnet einen von Hand erfassten Kurs. Er ist zulässig — ohne
	// Netz gibt es keinen anderen —, aber er ist als solcher erkennbar.
	Manual bool `gorm:"not null;default:false" json:"manual"`

	CreatedAt time.Time `json:"createdAt"`
}

// Rate liefert den Kurs als Dezimalzahl für die Anzeige.
func (r ExchangeRate) Rate() float64 { return float64(r.RateMicros) / float64(RateScale) }

// Validate prüft einen Kurs, bevor er gespeichert wird.
func (r *ExchangeRate) Validate() error {
	if len(r.Currency) != 3 {
		return fmt.Errorf("die Währung wird als dreistelliger ISO-Code angegeben, etwa USD oder CHF")
	}
	if r.Currency == "EUR" {
		return fmt.Errorf("der Euro ist die Buchwährung; für ihn gibt es keinen Umrechnungskurs")
	}
	if len(r.Date) != 10 {
		return fmt.Errorf("das Kursdatum fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if r.RateMicros <= 0 {
		return fmt.Errorf("der Kurs muss größer als null sein")
	}
	if r.Source == "" {
		return fmt.Errorf(
			"zum Kurs gehört seine Quelle. Ein Kurs ohne Herkunft lässt sich später nicht prüfen")
	}
	return nil
}

// VatExchangeRate ist ein Umrechnungskurs für die Umsatzsteuer.
//
// § 16 Abs. 6 UStG lässt die Umrechnung nach den Durchschnittskursen zu, die das
// Bundesministerium der Finanzen monatlich veröffentlicht — und in der Praxis
// ist das die Regel, weil sie die Bemessungsgrundlage über den Monat hinweg
// einheitlich hält. Der Kurs ist deshalb ein anderer als der Tageskurs, mit dem
// der Aufwand gebucht wird; die Differenz ist Kursaufwand oder Kursertrag und
// kein Fehler.
type VatExchangeRate struct {
	// Month ist der Monat im Format JJJJ-MM.
	Month    string `gorm:"primaryKey;size:7" json:"month"`
	Currency string `gorm:"primaryKey;size:3" json:"currency"`

	// RateMicros ist der Durchschnittskurs in Millionsteln.
	RateMicros int64 `gorm:"not null" json:"rateMicros"`
	// Source nennt die Fundstelle, etwa „BMF-Umsatzsteuer-Umrechnungskurse".
	Source string `gorm:"size:255;not null" json:"source"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// Rate liefert den Durchschnittskurs als Dezimalzahl für die Anzeige.
func (r VatExchangeRate) Rate() float64 { return float64(r.RateMicros) / float64(RateScale) }

// Validate prüft einen Durchschnittskurs.
func (r *VatExchangeRate) Validate() error {
	if len(r.Month) != 7 || r.Month[4] != '-' {
		return fmt.Errorf("der Monat wird als JJJJ-MM angegeben, etwa 2026-03")
	}
	if len(r.Currency) != 3 {
		return fmt.Errorf("die Währung wird als dreistelliger ISO-Code angegeben, etwa USD oder CHF")
	}
	if r.Currency == "EUR" {
		return fmt.Errorf("der Euro ist die Buchwährung; für ihn gibt es keinen Umrechnungskurs")
	}
	if r.RateMicros <= 0 {
		return fmt.Errorf("der Kurs muss größer als null sein")
	}
	if r.Source == "" {
		return fmt.Errorf("zum Durchschnittskurs gehört seine Fundstelle")
	}
	return nil
}

// MonthOf liefert den Monat eines ISO-Datums.
func MonthOf(date string) string {
	if len(date) < 7 {
		return ""
	}
	return date[:7]
}

// ConvertToEuro rechnet einen Fremdwährungsbetrag in Euro um.
//
// Gerundet wird einmal, kaufmännisch, und der Kurs ist ein Bruch: erst
// multiplizieren, dann teilen. Wer zuerst den Kurs in eine Dezimalzahl wandelt
// und danach multipliziert, verliert genau dort Genauigkeit, wo die Summe später
// gegen den Kontoauszug gehalten wird.
func ConvertToEuro(foreign Cents, rateMicros int64) Cents {
	if rateMicros <= 0 {
		return 0
	}
	return MulRound(foreign, RateScale, rateMicros)
}

// ExchangeRateRepository persistiert die Kurshistorie und die
// Umsatzsteuer-Durchschnittskurse.
type ExchangeRateRepository interface {
	// FindRate liefert den Kurs eines Tages oder nil.
	FindRate(ctx context.Context, currency, date string) (*ExchangeRate, error)
	// FindRateOnOrBefore liefert den jüngsten Kurs bis zu einem Tag.
	//
	// Die Europäische Zentralbank stellt an Wochenenden und Feiertagen keinen
	// Referenzkurs fest. Ein Beleg vom Sonntag hat trotzdem einen Kurs — den
	// zuletzt festgestellten. Ohne diesen Rückgriff wäre jede Rechnung vom
	// Wochenende unbuchbar.
	FindRateOnOrBefore(ctx context.Context, currency, date string) (*ExchangeRate, error)
	FindRates(ctx context.Context, currency, from, to string) ([]ExchangeRate, error)
	SaveRate(ctx context.Context, rate *ExchangeRate) error

	FindVatRate(ctx context.Context, currency, month string) (*VatExchangeRate, error)
	FindVatRates(ctx context.Context, from, to string) ([]VatExchangeRate, error)
	SaveVatRate(ctx context.Context, rate *VatExchangeRate) error
}

// CurrencyFetcher holt Kurse von einer externen Quelle.
type CurrencyFetcher interface {
	// RateAt liefert den Referenzkurs eines Tages als Millionstel, den Tag, für
	// den er tatsächlich festgestellt wurde, und die Quelle.
	//
	// Der tatsächliche Tag ist Teil der Antwort und keine Nebensache: die
	// Europäische Zentralbank stellt an Wochenenden und Feiertagen keinen Kurs
	// fest, und der Kurs eines Belegs vom Sonntag stammt vom Freitag. Wer das
	// nicht mitliefert, speichert einen Freitagskurs unter dem Sonntag.
	//
	// Es gibt keinen Rückfallwert. Ein geratener Kurs von 1,0 ist die
	// schlimmste aller Antworten: er sieht aus wie ein Ergebnis, bucht den
	// Fremdbetrag als Eurobetrag und fällt niemandem auf, bis der Abschluss
	// nicht mehr stimmt. Ohne Netz gibt der Anwender den Kurs mit seiner Quelle
	// selbst ein.
	RateAt(ctx context.Context, currency, date string) (rateMicros int64, actualDate, source string, err error)
}
