package accounting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Zusammenfassende Meldung nach § 18a UStG.
//
// Sie ist keine zweite Voranmeldung, sondern eine Meldung an das
// Bundeszentralamt für Steuern, mit der die EU-Staaten ihre Umsätze abgleichen.
// Dieselben Umsätze stehen in beiden: die ig. Lieferungen in Kennziffer 41 der
// Voranmeldung und in der Meldeart „L", die sonstigen Leistungen in Kennziffer
// 21 und in der Meldeart „S". Gehen die beiden auseinander, fragt das Finanzamt
// nach — deshalb rechnet Buchfink die Abstimmung mit aus, statt sie zu erwarten.

// ZMThreshold ist die Grenze des § 18a Abs. 1 Satz 2 UStG: übersteigt die Summe
// der innergemeinschaftlichen Lieferungen und Dreiecksgeschäfte im laufenden
// oder in einem der vier vorangegangenen Quartale 50.000 Euro, ist monatlich zu
// melden.
const ZMThreshold = domain.Cents(5_000_000)

// ZMRecipient sind die Stammdaten des Empfängers, soweit die Meldung sie
// braucht.
type ZMRecipient struct {
	Name        string
	CountryCode string
	VatID       string
	IsEU        bool
}

// ZMMovement ist ein einzelner meldepflichtiger Umsatz.
type ZMMovement struct {
	EntryID     uint
	EntryNumber string
	Date        string
	Kind        domain.ZMLineKind
	ContactID   uint
	Amount      domain.Cents
}

// ZMSource ist die Herkunft der Meldung.
type ZMSource struct {
	Entries []domain.JournalEntry
	// Recipient liefert die Stammdaten eines Geschäftspartners.
	Recipient func(contactID uint) ZMRecipient
}

// ZMMovements zerlegt die Buchungen in meldepflichtige Umsätze.
//
// Zugeordnet wird nach dem Leistungsdatum wie bei den Ausgangsumsätzen der
// Voranmeldung — sonst stünde derselbe Umsatz in den beiden Meldungen in
// verschiedenen Zeiträumen, und der Abgleich des Bundeszentralamts liefe genau
// darauf hinaus.
func ZMMovements(src ZMSource) []ZMMovement {
	treatments := RevenueTreatments()
	out := make([]ZMMovement, 0, len(src.Entries))

	for i := range src.Entries {
		entry := &src.Entries[i]
		if entry.Source == domain.EntrySourceOpening {
			continue
		}
		var contactID uint
		if entry.ContactID != nil {
			contactID = *entry.ContactID
		}
		recipient := ZMRecipient{}
		if src.Recipient != nil && contactID != 0 {
			recipient = src.Recipient(contactID)
		}

		for _, line := range entry.Lines {
			if line.TaxKey != "" {
				continue
			}
			var kind domain.ZMLineKind
			switch treatments[line.Account] {
			case domain.TaxTreatmentIntraCommunitySupply:
				kind = domain.ZMKindSupply
			case domain.TaxTreatmentReverseChargeSupply:
				// Nur Leistungen an einen Empfänger im übrigen
				// Gemeinschaftsgebiet sind zu melden. Die Leistung an ein
				// Drittland ist zwar auch nicht steuerbar, geht das
				// Bundeszentralamt aber nichts an.
				if !recipient.IsEU {
					continue
				}
				kind = domain.ZMKindService
			default:
				continue
			}

			amount := line.Amount
			if line.Side == domain.SideDebit {
				amount = -amount
			}
			out = append(out, ZMMovement{
				EntryID:     entry.ID,
				EntryNumber: entry.EntryNumber,
				Date:        VatPeriodFor(entry, line, ""),
				Kind:        kind,
				ContactID:   contactID,
				Amount:      amount,
			})
		}
	}
	return out
}

// ZMPeriodsOfYear liefert die Meldezeiträume eines Jahres.
//
// Der Regelfall ist das Quartal (§ 18a Abs. 1 Satz 1 UStG). Übersteigt die Summe
// der ig. Lieferungen und Dreiecksgeschäfte im laufenden oder in einem der vier
// vorangegangenen Quartale 50.000 Euro, wird monatlich gemeldet (Satz 2). Das
// Quartal, in dem die Grenze überschritten wird, wird selbst schon monatlich
// gemeldet — Satz 2 verlangt die Meldungen für die bereits abgelaufenen Monate
// dieses Quartals bis zum 25. Tag nach dem Monat der Überschreitung.
//
// movements sind die Umsätze des Jahres *und* der vier vorangegangenen Quartale;
// ohne sie ließe sich die Rückschau nicht führen.
func ZMPeriodsOfYear(year int, movements []ZMMovement) []VatPeriod {
	totals := map[string]domain.Cents{}
	for _, m := range movements {
		// Nur Lieferungen und Dreiecksgeschäfte zählen für die Grenze; die
		// sonstigen Leistungen nennt § 18a Abs. 1 Satz 2 UStG nicht.
		if m.Kind != domain.ZMKindSupply && m.Kind != domain.ZMKindTriangular {
			continue
		}
		p, err := VatPeriodOf(m.Date, domain.VatPeriodQuarter)
		if err != nil {
			continue
		}
		totals[p.Key] += m.Amount
	}

	out := make([]VatPeriod, 0, 12)
	for q := 1; q <= 4; q++ {
		monthly := false
		// Das laufende Quartal und die vier davor.
		for back := 0; back <= 4; back++ {
			qy, qq := year, q-back
			for qq <= 0 {
				qq += 4
				qy--
			}
			if totals[fmt.Sprintf("%d-Q%d", qy, qq)] > ZMThreshold {
				monthly = true
				break
			}
		}
		if !monthly {
			out = append(out, quarterPeriod(year, q))
			continue
		}
		for m := q*3 - 2; m <= q*3; m++ {
			out = append(out, monthPeriod(year, m))
		}
	}
	return out
}

// ZMLines fasst die Umsätze eines Zeitraums je USt-IdNr. und Meldeart zusammen
// und meldet die Befunde, die eine Übermittlung verhindern.
//
// Eine fehlende USt-IdNr. ist kein Formfehler: ohne sie ist die Meldung nicht
// abzugeben und die Steuerbefreiung der Lieferung nach § 6a Abs. 1 Satz 1 Nr. 4
// UStG nicht belegt.
func ZMLines(period VatPeriod, movements []ZMMovement, recipient func(uint) ZMRecipient) ([]domain.ZMLine, []string) {
	type key struct {
		vatID string
		kind  domain.ZMLineKind
	}
	grouped := map[key]*domain.ZMLine{}
	order := make([]key, 0, len(movements))
	findings := make([]string, 0)
	reported := map[uint]bool{}

	for _, m := range movements {
		if m.Date < period.From || m.Date > period.To {
			continue
		}
		info := ZMRecipient{}
		if recipient != nil && m.ContactID != 0 {
			info = recipient(m.ContactID)
		}
		vatID := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(info.VatID), " ", ""))
		if vatID == "" {
			if !reported[m.ContactID] {
				reported[m.ContactID] = true
				name := info.Name
				if name == "" {
					name = fmt.Sprintf("Geschäftspartner %d", m.ContactID)
				}
				findings = append(findings, fmt.Sprintf(
					"%s hat keine USt-IdNr. Ohne sie kann der Umsatz aus %s nicht gemeldet werden "+
						"(§ 18a Abs. 7 UStG); die Steuerbefreiung der Lieferung setzt sie ebenfalls voraus "+
						"(§ 6a Abs. 1 Satz 1 Nr. 4 UStG)", name, m.EntryNumber))
			}
			continue
		}

		k := key{vatID: vatID, kind: m.Kind}
		line := grouped[k]
		if line == nil {
			// Das Länderkennzeichen kommt aus der USt-IdNr. und nicht aus dem
			// Land des Kontakts. Beide gehen auseinander: Griechenland meldet
			// unter „EL" bei Landeskennung „GR", Nordirland unter „XI" bei „GB".
			// Das BZSt-Portal erwartet das Präfix der USt-IdNr. — stünde dort das
			// Land des Kontakts, meldete die Zeile „GR;EL123456789" statt
			// „EL;123456789", und die Datei wäre unbrauchbar. Das Land des
			// Kontakts bleibt der Rückfall, wenn die USt-IdNr. fehlt.
			country := ""
			if len(vatID) >= 2 {
				country = vatID[:2]
			}
			if country == "" {
				country = info.CountryCode
			}
			line = &domain.ZMLine{
				CountryCode: country,
				VatID:       vatID,
				Kind:        m.Kind,
				ContactID:   m.ContactID,
				ContactName: info.Name,
			}
			grouped[k] = line
			order = append(order, k)
		}
		line.Amount += m.Amount
		line.EntryIDs = appendUnique(line.EntryIDs, m.EntryID)
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].vatID != order[j].vatID {
			return order[i].vatID < order[j].vatID
		}
		return order[i].kind < order[j].kind
	})
	lines := make([]domain.ZMLine, 0, len(order))
	for _, k := range order {
		lines = append(lines, *grouped[k])
	}
	sort.Strings(findings)
	return lines, findings
}

func appendUnique(ids []uint, id uint) []uint {
	if id == 0 {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
