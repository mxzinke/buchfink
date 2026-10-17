package declaration

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// Gesetzliche Größen der Erklärungen, Wortlaut gelesen am 17.10.2026.

// corporateTaxPermille ist der Körperschaftsteuersatz des
// Veranlagungszeitraums (§ 23 Abs. 1 KStG).
func corporateTaxPermille(year int) int64 {
	switch {
	case year <= 2027:
		return 150
	case year == 2028:
		return 140
	case year == 2029:
		return 130
	case year == 2030:
		return 120
	case year == 2031:
		return 110
	default:
		return 100
	}
}

// solidarityPermille ist der Solidaritätszuschlag (§ 4 Satz 1 SolZG).
const solidarityPermille = 55

// tradeTaxBasePermille ist die Steuermesszahl (§ 11 Abs. 2 GewStG).
const tradeTaxBasePermille = 35

// lossCap ist der Sockel, bis zu dem ein Verlustvortrag unbeschränkt abgezogen
// wird (§ 10d Abs. 2 Satz 1 EStG, § 10a Satz 1 GewStG).
const lossCap domain.Cents = 1_000_000_00

// corporateLossPermille ist der Anteil über dem Sockel, der beim
// Verlustvortrag abziehbar ist. 70 % ab dem Veranlagungszeitraum 2024
// (§ 10d Abs. 2 Satz 1 EStG i. V. m. § 52 Abs. 18b Satz 3 EStG), vorher 60 %.
func corporateLossPermille(year int) int64 {
	if year >= 2024 {
		return 700
	}
	return 600
}

// tradeLossPermille ist der Anteil über dem Sockel beim Gewerbeverlust
// (§ 10a Satz 2 GewStG).
const tradeLossPermille = 600

// financingAllowance ist der Freibetrag der Hinzurechnung nach § 8 Nr. 1
// GewStG.
const financingAllowance domain.Cents = 200_000_00

// donationPercent und donationPermille sind die Höchstbeträge für Zuwendungen
// (§ 9 Abs. 1 Nr. 2 Satz 1 KStG, § 9 Nr. 5 Satz 1 GewStG).
const (
	donationPercent  = 20
	donationPermille = 4
)

// minimumMultiplier ist der Mindesthebesatz (§ 16 Abs. 4 Satz 2 GewStG):
// 200 %, ab dem Erhebungszeitraum 2027 280 % (§ 36 Abs. 5b GewStG).
func minimumMultiplier(year int) int64 {
	if year >= 2027 {
		return 280
	}
	return 200
}

// lossDeduction ist der abziehbare Verlustvortrag bei einem positiven
// Gesamtbetrag: bis zum Sockel voll, darüber mit dem Anteil.
func lossDeduction(carryForward, income domain.Cents, permille int64) domain.Cents {
	if carryForward <= 0 || income <= 0 {
		return 0
	}
	limit := income
	if income > lossCap {
		limit = lossCap + domain.MulRound(income-lossCap, permille, 1000)
	}
	if carryForward < limit {
		return carryForward
	}
	return limit
}

// truncEuro schneidet die Cent ab (zugunsten bei einer Steuerschuld).
func truncEuro(c domain.Cents) domain.Cents { return c - c%100 }

// ceilEuro rundet auf volle Euro auf (§ 36 Abs. 3 EStG).
func ceilEuro(c domain.Cents) domain.Cents {
	if c <= 0 {
		return c - c%100
	}
	if c%100 == 0 {
		return c
	}
	return c - c%100 + 100
}

// percentText schreibt Promille als Prozentangabe: 150 → „15 %“, 55 → „5,5 %“.
func percentText(permille int64) string {
	if permille%10 == 0 {
		return fmt.Sprintf("%d %%", permille/10)
	}
	return fmt.Sprintf("%d,%d %%", permille/10, permille%10)
}
