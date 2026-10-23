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

// percentText schreibt Promille als Prozentangabe: 150 → „15 %“, 55 → „5,5 %“.
func percentText(permille int64) string {
	if permille%10 == 0 {
		return fmt.Sprintf("%d %%", permille/10)
	}
	return fmt.Sprintf("%d,%d %%", permille/10, permille%10)
}
