package accounting

import (
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Wertberichtigung auf Forderungen zum Stichtag.
//
// Forderungen sind mit dem niedrigeren beizulegenden Wert anzusetzen (§ 253
// Abs. 4 Satz 2 HGB): einzeln, wo ein Ausfall droht (Einzelwertberichtigung),
// pauschal für das allgemeine Ausfallrisiko der übrigen
// (Pauschalwertberichtigung). Bemessen wird der Nettobetrag: die Umsatzsteuer
// wird erst berichtigt, wenn das Entgelt uneinbringlich ist (§ 17 Abs. 2 Nr. 1
// UStG), bis dahin kommt sie im Ausfall zurück. Die Forderung selbst bleibt auf
// dem Personenkonto stehen.

// Konten der Wertberichtigung (SKR04).
const (
	AccountEWBShortTerm = "1246" // Einzelwertberichtigung, Restlaufzeit bis 1 Jahr
	AccountEWBLongTerm  = "1247" // Einzelwertberichtigung, Restlaufzeit über 1 Jahr
	AccountPWBShortTerm = "1248" // Pauschalwertberichtigung, Restlaufzeit bis 1 Jahr
	AccountPWBLongTerm  = "1249" // Pauschalwertberichtigung, Restlaufzeit über 1 Jahr
	AccountEWBExpense   = "6923" // Einstellung in die Einzelwertberichtigung
	AccountPWBExpense   = "6920" // Einstellung in die Pauschalwertberichtigung
	AccountEWBIncome    = "4923" // Erträge aus der Herabsetzung der Einzelwertberichtigung
	AccountPWBIncome    = "4920" // Erträge aus der Herabsetzung der Pauschalwertberichtigung
)

// AllowanceAccounts sind die vier Bestandskonten der Wertberichtigung.
func AllowanceAccounts() []string {
	return []string{AccountEWBShortTerm, AccountEWBLongTerm, AccountPWBShortTerm, AccountPWBLongTerm}
}

// ReceivableLossAccounts sind die Konten der Forderungsverluste, aus denen sich
// die eigene Ausfallquote ergibt.
func ReceivableLossAccounts() []string {
	return []string{"6930", "6931", "6932", "6933", "6936", "6938", "6280", "6281", "6286"}
}

// SpecificAllowance ist die Einzelwertberichtigung eines offenen Postens: der
// erwartete Ausfall in Prozent des Nettobetrags und warum.
type SpecificAllowance struct {
	EntryID uint   `json:"entryId"`
	Percent int64  `json:"percent"`
	Reason  string `json:"reason"`
}

// AllowanceInput sind die offenen Posten zum Stichtag und die Entscheidungen
// über sie.
type AllowanceInput struct {
	Cutoff string
	Items  []domain.OpenItem
	// Specific sind die Einzelwertberichtigungen je Posten.
	Specific []SpecificAllowance
	// BandPercents setzt einen Ausfall je Altersband für die Posten, die keine
	// eigene Angabe haben.
	BandPercents map[domain.AgingBucketKey]int64
	// GeneralPermille ist die Quote der Pauschalwertberichtigung in Promille
	// auf die übrigen Forderungen.
	GeneralPermille int64
}

// AllowanceItem ist eine Forderung mit ihrer Bewertung.
type AllowanceItem struct {
	EntryID        uint                  `json:"entryId"`
	EntryNumber    string                `json:"entryNumber"`
	DocumentNumber string                `json:"documentNumber"`
	ContactID      uint                  `json:"contactId"`
	ContactName    string                `json:"contactName"`
	DueDate        string                `json:"dueDate"`
	Band           domain.AgingBucketKey `json:"band"`
	OverdueDays    int                   `json:"overdueDays"`
	// OpenAmount ist der offene Bruttobetrag, NetAmount die Bemessung.
	OpenAmount domain.Cents `json:"openAmount"`
	NetAmount  domain.Cents `json:"netAmount"`
	// LongTerm heißt: Restlaufzeit über ein Jahr.
	LongTerm bool `json:"longTerm"`
	// Candidate ist der Vorschlag aus der Altersstruktur: über 60 Tage
	// überfällig, das Ausfallrisiko ist zu prüfen. Hint sagt warum.
	Candidate bool   `json:"candidate"`
	Hint      string `json:"hint,omitempty"`
	// Percent und Amount sind die Einzelwertberichtigung.
	Percent int64        `json:"percent"`
	Amount  domain.Cents `json:"amount"`
	Reason  string       `json:"reason,omitempty"`
}

// AllowanceResult sind die Sollbestände der vier Wertberichtigungskonten.
type AllowanceResult struct {
	Items []AllowanceItem `json:"items"`
	// SpecificShort und SpecificLong sind die Einzelwertberichtigungen nach
	// Restlaufzeit (1246, 1247).
	SpecificShort domain.Cents `json:"specificShort"`
	SpecificLong  domain.Cents `json:"specificLong"`
	// GeneralBase ist der Nettobetrag der nicht einzeln berichtigten
	// Forderungen; GeneralShort und GeneralLong die Pauschalwertberichtigung
	// darauf (1248, 1249).
	GeneralBase     domain.Cents `json:"generalBase"`
	GeneralPermille int64        `json:"generalPermille"`
	GeneralShort    domain.Cents `json:"generalShort"`
	GeneralLong     domain.Cents `json:"generalLong"`
}

// Targets sind die Sollbestände je Konto, als Habensaldo positiv.
func (r *AllowanceResult) Targets() map[string]domain.Cents {
	return map[string]domain.Cents{
		AccountEWBShortTerm: r.SpecificShort, AccountEWBLongTerm: r.SpecificLong,
		AccountPWBShortTerm: r.GeneralShort, AccountPWBLongTerm: r.GeneralLong,
	}
}

// CalculateAllowance bewertet die Forderungen aus Lieferungen und Leistungen.
func CalculateAllowance(in AllowanceInput) (*AllowanceResult, error) {
	if len(in.Cutoff) != 10 {
		return nil, fmt.Errorf("die Wertberichtigung braucht den Stichtag")
	}
	if in.GeneralPermille < 0 || in.GeneralPermille > 1000 {
		return nil, fmt.Errorf("die Quote der Pauschalwertberichtigung liegt zwischen 0 und 1000 ‰, nicht bei %d ‰",
			in.GeneralPermille)
	}
	for band, percent := range in.BandPercents {
		if percent < 0 || percent > 100 {
			return nil, fmt.Errorf("der Ausfall im Band %s liegt zwischen 0 und 100 %%, nicht bei %d %%", band, percent)
		}
	}
	specific := make(map[uint]SpecificAllowance, len(in.Specific))
	for _, s := range in.Specific {
		if s.Percent < 0 || s.Percent > 100 {
			return nil, fmt.Errorf("der erwartete Ausfall liegt zwischen 0 und 100 %%, nicht bei %d %%", s.Percent)
		}
		specific[s.EntryID] = s
	}

	out := &AllowanceResult{Items: make([]AllowanceItem, 0), GeneralPermille: in.GeneralPermille}
	oneYear := addYears(in.Cutoff, 1)
	var baseShort, baseLong domain.Cents
	for _, item := range in.Items {
		if item.ContactType != domain.ContactTypeCustomer || item.OpenAmount <= 0 || item.IsCredit() ||
			item.EntryID == 0 || item.Source == domain.OpenItemSourceAdvance {
			continue
		}
		row := AllowanceItem{
			EntryID: item.EntryID, EntryNumber: item.EntryNumber, DocumentNumber: item.DocumentNumber,
			ContactID: item.ContactID, ContactName: item.ContactName, DueDate: item.DueDate,
			Band: agingKey(item.DueDate, in.Cutoff), OpenAmount: item.OpenAmount,
			NetAmount: netOfOpenItem(item),
			LongTerm:  item.DueDate != "" && item.DueDate > oneYear,
		}
		if item.DueDate != "" && item.DueDate < in.Cutoff {
			row.OverdueDays = daysBetween(item.DueDate, in.Cutoff)
		}
		switch row.Band {
		case domain.AgingOver90:
			row.Candidate = true
			row.Hint = fmt.Sprintf("seit %d Tagen überfällig: Ausfallrisiko prüfen (Mahnstand, "+
				"Zahlungsfähigkeit, Bestreiten, Insolvenz)", row.OverdueDays)
		case domain.AgingUpTo90:
			row.Candidate = true
			row.Hint = fmt.Sprintf("seit %d Tagen überfällig: Mahnstand prüfen", row.OverdueDays)
		}
		if s, ok := specific[item.EntryID]; ok {
			row.Percent, row.Reason = s.Percent, s.Reason
			delete(specific, item.EntryID)
		} else if percent, ok := in.BandPercents[row.Band]; ok {
			row.Percent = percent
			row.Reason = fmt.Sprintf("Ausfall %d %% für das Altersband", percent)
		}
		if row.Percent > 0 {
			row.Amount = domain.MulRound(row.NetAmount, row.Percent, 100)
			if row.LongTerm {
				out.SpecificLong += row.Amount
			} else {
				out.SpecificShort += row.Amount
			}
		} else if row.LongTerm {
			baseLong += row.NetAmount
		} else {
			baseShort += row.NetAmount
		}
		out.Items = append(out.Items, row)
	}
	for id := range specific {
		return nil, fmt.Errorf("die Buchung %d trägt zum %s keine offene Forderung; einzeln berichtigen "+
			"lässt sich nur ein offener Posten", id, in.Cutoff)
	}
	out.GeneralBase = baseShort + baseLong
	out.GeneralShort = domain.MulRound(baseShort, in.GeneralPermille, 1000)
	out.GeneralLong = domain.MulRound(baseLong, in.GeneralPermille, 1000)

	sort.SliceStable(out.Items, func(i, j int) bool {
		if out.Items[i].OverdueDays != out.Items[j].OverdueDays {
			return out.Items[i].OverdueDays > out.Items[j].OverdueDays
		}
		return out.Items[i].EntryNumber < out.Items[j].EntryNumber
	})
	return out, nil
}

// netOfOpenItem ist der Nettobetrag einer offenen Forderung. Nur eine im
// Inland steuerpflichtige Leistung trägt die Umsatzsteuer im offenen Betrag.
func netOfOpenItem(item domain.OpenItem) domain.Cents {
	if item.TaxTreatment != domain.TaxTreatmentDomestic || item.TaxRate == 0 {
		return item.OpenAmount
	}
	return item.TaxRate.NetFromGross(item.OpenAmount)
}

// LossRatePermille ist die eigene Ausfallquote: Forderungsverluste im
// Verhältnis zu den Umsatzerlösen derselben Jahre, in Promille. Ohne Umsatz gibt
// es keine Quote.
func LossRatePermille(losses, revenue domain.Cents) int64 {
	if revenue <= 0 || losses <= 0 {
		return 0
	}
	return int64(domain.MulRound(losses, 1000, int64(revenue)))
}
