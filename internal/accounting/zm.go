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
// Buchfink meldet die sonstigen Leistungen nach § 3a Abs. 2 UStG an
// Unternehmer im übrigen Gemeinschaftsgebiet (Meldeart „S"). Dieselben Umsätze
// stehen in Kennziffer 21 der Voranmeldung. Gehen die beiden auseinander, fragt
// das Finanzamt nach — deshalb rechnet Buchfink die Abstimmung mit aus, statt
// sie zu erwarten.

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
			rule := TaxCaseFor(treatments[line.Account]).VatCode("").ForRecipient(recipient.IsEU, false)
			if rule.ZMKind == "" {
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
				ContactID:   contactID,
				Amount:      amount,
			})
		}
	}
	return out
}

// ZMPeriodsOfYear liefert die Meldezeiträume eines Jahres: die vier
// Kalendervierteljahre. Für sonstige Leistungen ist das Quartal der Meldezeitraum
// (§ 18a Abs. 1 Satz 1 UStG); die Umstellung auf den Monat knüpft § 18a Abs. 1
// Satz 2 UStG allein an Warenlieferungen.
func ZMPeriodsOfYear(year int) []VatPeriod {
	return VatPeriodsOfYear(year, domain.VatPeriodQuarter)
}

// ZMPeriodCovers meldet, ob ein Umsatz in den Meldezeitraum gehört.
func ZMPeriodCovers(period VatPeriod, m ZMMovement) bool {
	return m.Date >= period.From && m.Date <= period.To
}

// ZMPeriodFor liefert den Meldezeitraum, in dem ein Umsatz zu melden ist.
func ZMPeriodFor(periods []VatPeriod, m ZMMovement) (VatPeriod, bool) {
	for _, p := range periods {
		if ZMPeriodCovers(p, m) {
			return p, true
		}
	}
	return VatPeriod{}, false
}

// ZMLines fasst die Umsätze eines Zeitraums je USt-IdNr. zusammen und meldet die
// Befunde, die eine Übermittlung verhindern.
//
// Eine fehlende USt-IdNr. ist kein Formfehler: ohne sie ist die Meldung nicht
// abzugeben (§ 18a Abs. 7 UStG).
func ZMLines(period VatPeriod, movements []ZMMovement, recipient func(uint) ZMRecipient) ([]domain.ZMLine, []string) {
	service := TaxCaseFor(domain.TaxTreatmentReverseChargeSupply).VatCode("").ForRecipient(true, false)
	grouped := map[string]*domain.ZMLine{}
	order := make([]string, 0, len(movements))
	findings := make([]string, 0)
	reported := map[uint]bool{}

	for _, m := range movements {
		if !ZMPeriodCovers(period, m) {
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
						"(§ 18a Abs. 7 UStG)", name, m.EntryNumber))
			}
			continue
		}

		line := grouped[vatID]
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
				Kind:        service.ZMKind,
				ContactID:   m.ContactID,
				ContactName: info.Name,
			}
			grouped[vatID] = line
			order = append(order, vatID)
		}
		line.Amount += m.Amount
		line.EntryIDs = appendUnique(line.EntryIDs, m.EntryID)
	}

	sort.Strings(order)
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
