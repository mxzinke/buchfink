package declaration

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// taxParams sind die gesetzlichen Werte des Veranlagungs- oder
// Erhebungszeitraums: Steuersätze, Messzahl, Verlustabzug, Höchstbeträge der
// Zuwendungen und Mindesthebesatz. Sie kommen aus derselben datierten Tabelle
// wie Buchungen und Steuerrückstellung, mit dem Jahresende als Stichtag (§ 7
// Abs. 4 Satz 2 KStG).
func taxParams(year int) (accounting.TaxParameters, error) {
	return accounting.TaxParametersFor(fmt.Sprintf("%04d-12-31", year))
}

// financingAllowance ist der Freibetrag der Hinzurechnung nach § 8 Nr. 1
// GewStG.
const financingAllowance domain.Cents = 200_000_00

// lossDeduction ist der abziehbare Verlustvortrag bei einem positiven
// Gesamtbetrag: bis zum Sockel voll, darüber mit dem Anteil (§ 10d Abs. 2
// Satz 1 EStG, § 10a GewStG).
func lossDeduction(carryForward, income, base domain.Cents, permille int64) domain.Cents {
	if carryForward <= 0 || income <= 0 {
		return 0
	}
	limit := income
	if income > base {
		limit = base + domain.MulRound(income-base, permille, 1000)
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
