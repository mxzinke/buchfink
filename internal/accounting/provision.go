package accounting

import (
	"fmt"
	"math"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// ProvisionAccounts nennt das Bilanz- und das Aufwandskonto einer
// Rückstellungsart.
//
// Der Vorschlag folgt dem Kontenrahmen und ist änderbar: der SKR04 führt für
// jede Art des § 249 HGB ein eigenes Rückstellungskonto, aber der Aufwand
// entsteht dort, wo er ohne die Rückstellung entstanden wäre — die
// Abschlusskosten auf 6827, die Instandhaltung auf ihrem Reparaturkonto. Wer
// einen anderen Aufwand meint, ändert das Konto; das Bilanzkonto folgt der Art
// und bleibt.
func ProvisionAccounts(kind domain.ProvisionKind) (balance, expense string) {
	switch kind {
	case domain.ProvisionPendingLoss:
		return domain.AccountRueckstellungDrohverlust, domain.AccountSonstigerAufwand
	case domain.ProvisionDeferredMaintenance:
		return domain.AccountRueckstellungInstandhaltung, domain.AccountInstandhaltung
	case domain.ProvisionWarranty:
		return domain.AccountRueckstellungGewaehrleistung, domain.AccountAufwandGewaehrleistung
	case domain.ProvisionTaxIncome:
		return domain.AccountRueckstellungKoerperschaft, domain.AccountKoerperschaftsteuer
	case domain.ProvisionTaxTrade:
		return domain.AccountRueckstellungGewerbesteuer, domain.AccountGewerbesteuer
	case domain.ProvisionClosingCosts:
		return domain.AccountRueckstellungAbschluss, domain.AccountAbschlusskosten
	case domain.ProvisionRetentionCosts:
		return domain.AccountRueckstellungAufbewahrung, domain.AccountSonstigerAufwand
	case domain.ProvisionPersonnel:
		return domain.AccountRueckstellungPersonal, domain.AccountPersonalaufwandUrlaub
	case domain.ProvisionPension:
		return domain.AccountRueckstellungPensionen, domain.AccountSonstigerAufwand
	default:
		return domain.AccountRueckstellungSonstige, domain.AccountSonstigerAufwand
	}
}

// DiscountScale ist der Nenner der Zinssätze: sie stehen in Millionsteln, damit
// 1,49 % ohne Rundung gespeichert werden kann.
const DiscountScale = 1_000_000

// RemainingYears ist die Restlaufzeit in vollen Jahren, aufgerundet.
//
// § 253 Abs. 2 Satz 1 HGB verlangt den „ihrer Restlaufzeit entsprechenden
// durchschnittlichen Marktzinssatz", und die Bundesbank veröffentlicht ihn je
// vollem Jahr. Aufgerundet wird, weil die Tabelle keinen Satz für 2,4 Jahre
// kennt und die längere Laufzeit den vorsichtigeren — weil höheren — Satz
// trägt.
func RemainingYears(cutoff, due string) (int, error) {
	from, err := time.Parse("2006-01-02", cutoff)
	if err != nil {
		return 0, fmt.Errorf("der Stichtag %q ist kein Datum", cutoff)
	}
	to, err := time.Parse("2006-01-02", due)
	if err != nil {
		return 0, fmt.Errorf("der Erfüllungszeitpunkt %q ist kein Datum", due)
	}
	if !to.After(from) {
		return 0, nil
	}
	// Gezählt wird kalendarisch und nicht in 365-Tage-Schritten. Drei Jahre über
	// ein Schaltjahr sind 1096 Tage; durch 365 geteilt wären das drei Jahre und
	// ein Rest, aufgerundet vier — und damit der Satz einer Laufzeit, die es
	// nicht gibt. NeedsDiscounting rechnet aus demselben Grund mit AddDate.
	years := 0
	anchor := from
	for {
		next := anchor.AddDate(1, 0, 0)
		if next.After(to) {
			break
		}
		anchor = next
		years++
	}
	// Ein angefangenes Jahr zählt voll: die Tabelle der Bundesbank kennt keinen
	// Satz für 2,4 Jahre, und die längere Laufzeit trägt den vorsichtigeren —
	// weil höheren — Satz.
	if anchor.Before(to) {
		years++
	}
	return years, nil
}

// NeedsDiscounting meldet, ob abzuzinsen ist: erst ab einer Restlaufzeit von
// mehr als einem Jahr (§ 253 Abs. 2 Satz 1 HGB).
func NeedsDiscounting(cutoff, due string) bool {
	from, err1 := time.Parse("2006-01-02", cutoff)
	to, err2 := time.Parse("2006-01-02", due)
	if err1 != nil || err2 != nil {
		return false
	}
	return to.After(from.AddDate(1, 0, 0))
}

// PresentValue zinst einen Erfüllungsbetrag auf den Bilanzstichtag ab.
//
//	Barwert = Betrag / (1 + Zinssatz)^Jahre
//
// Gerechnet wird in Gleitkomma und am Ende kaufmännisch gerundet. Das ist hier
// vertretbar, wo es sonst nicht wäre: die Potenz lässt sich in Ganzzahlen nicht
// ohne eigene Festkommabibliothek bilden, und der Zinssatz selbst ist eine auf
// zwei Nachkommastellen veröffentlichte Größe — die Genauigkeit der Rechnung
// übersteigt die ihrer Eingangsgröße um ein Vielfaches.
func PresentValue(amount domain.Cents, years int, rateMicros int64) domain.Cents {
	if years <= 0 || rateMicros <= 0 || amount == 0 {
		return amount
	}
	rate := float64(rateMicros) / DiscountScale
	factor := math.Pow(1+rate, float64(years))
	value := float64(amount) / factor
	if value < 0 {
		return domain.Cents(value - 0.5)
	}
	return domain.Cents(value + 0.5)
}

// DiscountRateFor sucht in einer Zinstabelle den Satz für eine Restlaufzeit.
//
// Fehlt er, meldet die Funktion das und rät nicht. Ein interpolierter oder aus
// dem Nachbarjahr geliehener Satz sähe aus wie ein echter, und der Unterschied
// zwischen „abgezinst mit 1,49 %" und „nicht abgezinst, weil kein Satz
// hinterlegt ist" ist genau das, was der Anwender wissen muss.
// DiscountAverageFor nennt die Mittelungsdauer, mit der eine Rückstellungsart
// abzuzinsen ist.
//
// § 253 Abs. 2 Satz 1 HGB nennt den Durchschnitt der vergangenen sieben Jahre.
// Satz 2 macht davon eine Ausnahme für Altersversorgungsverpflichtungen: sie
// dürfen pauschal mit dem Satz einer Restlaufzeit von fünfzehn Jahren und —
// seit dem Gesetz vom 11. März 2016 — mit dem Durchschnitt der vergangenen zehn
// Jahre abgezinst werden. Die Bundesbank veröffentlicht beide Reihen, und die
// Zinstabelle hält beide; welche gilt, folgt aus der Art und nicht aus der
// Eingabe.
func DiscountAverageFor(kind domain.ProvisionKind) int {
	if kind == domain.ProvisionPension {
		return 10
	}
	return 7
}

func DiscountRateFor(rates []domain.DiscountRate, years, average int) (domain.DiscountRate, bool) {
	if years < 1 {
		return domain.DiscountRate{}, false
	}
	if years > 50 {
		years = 50
	}
	for _, r := range rates {
		if r.Years == years && r.Average == average {
			return r, true
		}
	}
	return domain.DiscountRate{}, false
}

// ProvisionMirrorRow und ProvisionMirror stehen in internal/domain, damit der
// Jahresabschluss den Spiegel als Teil des Anhangs tragen kann, ohne dass
// domain dieses Paket importieren müsste. Gebaut wird er hier — die Rechnung
// gehört zur Auswertung, nicht zum Datenmodell.
type (
	ProvisionMirrorRow = domain.ProvisionMirrorRow
	ProvisionMirror    = domain.ProvisionMirror
)

// BuildProvisionMirror verdichtet die Rückstellungen eines Jahres zum Spiegel.
func BuildProvisionMirror(provisions []domain.Provision, fiscalYear int) *ProvisionMirror {
	mirror := &ProvisionMirror{FiscalYear: fiscalYear, Rows: make([]ProvisionMirrorRow, 0, len(provisions))}
	index := map[domain.ProvisionKind]int{}

	for i := range provisions {
		p := &provisions[i]
		pos, ok := index[p.Kind]
		if !ok {
			balance, _ := ProvisionAccounts(p.Kind)
			mirror.Rows = append(mirror.Rows, ProvisionMirrorRow{
				Kind: p.Kind, Label: p.Kind.Label(), Account: balance,
			})
			pos = len(mirror.Rows) - 1
			index[p.Kind] = pos
		}
		row := &mirror.Rows[pos]
		row.Opening += p.BalanceAt(fiscalYear - 1)
		for _, m := range p.Movements {
			if m.FiscalYear != fiscalYear {
				continue
			}
			switch m.Kind {
			case domain.ProvisionFormation, domain.ProvisionIncrease:
				row.Additions += m.Amount
			case domain.ProvisionConsumption:
				row.Used += m.Amount
			case domain.ProvisionRelease:
				row.Released += m.Amount
			case domain.ProvisionUnwinding:
				row.Unwinding += m.Amount
			}
		}
		row.Closing = row.Opening + row.Additions + row.Unwinding - row.Used - row.Released
	}

	for _, row := range mirror.Rows {
		mirror.Total.Opening += row.Opening
		mirror.Total.Additions += row.Additions
		mirror.Total.Used += row.Used
		mirror.Total.Released += row.Released
		mirror.Total.Unwinding += row.Unwinding
		mirror.Total.Closing += row.Closing
	}
	mirror.Total.Label = "Summe"
	return mirror
}
