package accounting

import (
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// partnershipAccountRanges sind die Konten des SKR04, die nur Einzelunternehmen
// und Personengesellschaften führen: ausstehende Einlagen der Komplementäre und
// Kommanditisten, Ausleihungen, Forderungen und Verbindlichkeiten gegenüber
// persönlich haftenden Gesellschaftern und Kommanditisten, die Kapital- und
// Privatkonten (2000–2799) und die statistischen Kapitalkonten der
// Gesellschafter.
var partnershipAccountRanges = [][2]int{
	{50, 89},
	{962, 963},
	{1317, 1319},
	{2000, 2799},
	{3645, 3653},
	{9140, 9189},
	{9400, 9798},
	{9802, 9870},
	{9883, 9886},
}

// IsPartnershipAccount meldet, ob ein Konto nur Einzelunternehmen und
// Personengesellschaften gehört.
func IsPartnershipAccount(number string) bool {
	n, err := strconv.Atoi(number)
	if err != nil || len(number) != 4 {
		return false
	}
	for _, r := range partnershipAccountRanges {
		if n >= r[0] && n <= r[1] {
			return true
		}
	}
	return false
}

// EnsurePostableFor prüft ein Konto wie EnsurePostable und zusätzlich gegen die
// Rechtsform. Buchfink führt nur Kapitalgesellschaften; eine fehlende Angabe
// gilt deshalb als GmbH.
func (c *Chart) EnsurePostableFor(number, legalForm string) error {
	if err := c.EnsurePostable(number); err != nil {
		return err
	}
	if !IsPartnershipAccount(number) {
		return nil
	}
	if legalForm == "" {
		legalForm = domain.LegalFormGmbH
	}
	if !domain.IsSupportedLegalForm(legalForm) {
		return nil
	}
	return fmt.Errorf(
		"Konto %s (%s) gehört zu Einzelunternehmen und Personengesellschaften; eine %s hat keine "+
			"Privat- oder Kapitalkonten ihrer Gesellschafter. Entnimmt ein Gesellschafter Geld, ist das "+
			"eine Forderung gegen ihn (1307 Forderungen gegen GmbH-Gesellschafter); legt er Geld ein oder "+
			"gibt er ein Darlehen, eine Verbindlichkeit ihm gegenüber (3640 Verbindlichkeiten gegenüber "+
			"GmbH-Gesellschaftern) oder eine Kapitalrücklage (2920). Fließt ihm ohne angemessene "+
			"Gegenleistung ein Vorteil zu, kann das eine verdeckte Gewinnausschüttung sein "+
			"(§ 8 Abs. 3 Satz 2 KStG)",
		number, c.Name(number), legalForm)
}
