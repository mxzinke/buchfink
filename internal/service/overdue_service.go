package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Überfällige Kundenforderungen und was der Verzug kostet (§§ 286, 288 BGB).
//
// Der Dienst rechnet und bucht nicht: Verzugszinsen und Pauschale sind erst mit
// der Zahlung Ertrag. Wer sie vorher einbuchte, wiese eine Forderung aus, die in
// den meisten Fällen nie fließt, und müsste sie später wieder abschreiben.

// OverdueItem ist ein überfälliger Posten mit seiner Verzugsrechnung.
type OverdueItem struct {
	EntryID        uint         `json:"entryId"`
	DocumentNumber string       `json:"documentNumber"`
	DocumentDate   string       `json:"documentDate"`
	DueDate        string       `json:"dueDate"`
	OpenAmount     domain.Cents `json:"openAmount"`
	// DaysOverdue ist der Abstand zur Fälligkeit, DefaultFrom der erste Tag des
	// Verzugs (dreißig Tage danach), InterestDays die verzinsten Tage.
	//
	// Verzinst wird der heute offene Betrag über den ganzen Verzugszeitraum.
	// Eine Teilzahlung mitten im Verzug bleibt damit außer Betracht; die
	// Zinsforderung ist dann eher zu niedrig als zu hoch, und der Hinweis sagt
	// es.
	DaysOverdue  int          `json:"daysOverdue"`
	DefaultFrom  string       `json:"defaultFrom"`
	InterestDays int          `json:"interestDays"`
	Interest     domain.Cents `json:"interest"`
	// LumpSum ist die Pauschale des § 288 Abs. 5 BGB für diese Forderung.
	LumpSum domain.Cents `json:"lumpSum"`
	// LastReminder ist der Zugang der letzten Mahnung, ReminderLevel ihre
	// Stufe; leer ohne Mahnvermerk.
	LastReminder  string `json:"lastReminder,omitempty"`
	ReminderLevel int    `json:"reminderLevel,omitempty"`
	// Note sagt, was an diesem Posten nicht gerechnet werden konnte. Zinsen,
	// die still mit null ausgewiesen werden, führten zu einer zu niedrigen
	// Forderung, und niemand sähe, woran es lag.
	Note string `json:"note,omitempty"`
}

// OverdueClaim sind die überfälligen Posten eines Kunden.
type OverdueClaim struct {
	ContactID   uint   `json:"contactId"`
	ContactName string `json:"contactName"`
	// IsConsumer entscheidet über Zinssatz und Pauschale. Er folgt aus den
	// Stammdaten (Contact.IsPrivate): ein Geschäftspartner, der kein
	// Unternehmer ist, ist der Verbraucher des § 13 BGB.
	IsConsumer bool `json:"isConsumer"`
	// Cutoff ist der Stichtag der Rechnung.
	Cutoff string `json:"cutoff"`

	Items []OverdueItem `json:"items"`

	Principal domain.Cents `json:"principal"`
	Interest  domain.Cents `json:"interest"`
	LumpSum   domain.Cents `json:"lumpSum"`
	Total     domain.Cents `json:"total"`

	Note string `json:"note"`
}

// EnsureLists ersetzt eine nicht belegte Postenliste durch eine leere.
func (c *OverdueClaim) EnsureLists() {
	if c.Items == nil {
		c.Items = make([]OverdueItem, 0)
	}
}

// OverdueService rechnet die überfälligen Kundenforderungen.
type OverdueService struct {
	openItems   LiveOpenItemSource
	contactRepo domain.ContactRepository
	// invoiceRepo beantwortet, ob eine Rechnung gegenüber einem Verbraucher auf
	// den Verzug nach dreißig Tagen hingewiesen hat (§ 286 Abs. 3 Satz 1
	// Halbsatz 2 BGB).
	invoiceRepo domain.InvoiceRepository
	// dunning sind die Mahnvermerke; ohne sie rechnet der Dienst wie vor
	// jeder Mahnung.
	dunning   domain.DunningRepository
	auditRepo domain.AuditRepository
}

// NewOverdueService wires die Verzugsrechnung.
func NewOverdueService(openItems LiveOpenItemSource, contactRepo domain.ContactRepository) *OverdueService {
	return &OverdueService{openItems: openItems, contactRepo: contactRepo}
}

// SetInvoiceSource hängt die Rechnungen an: sie tragen das Kennzeichen, ob der
// Verzugshinweis an den Verbraucher gedruckt wurde.
func (s *OverdueService) SetInvoiceSource(r domain.InvoiceRepository) { s.invoiceRepo = r }

// consumerNoticePrinted meldet, ob die Rechnung hinter einem offenen Posten den
// Verzugshinweis aufwies.
//
// Beantwortet wird das nur für Rechnungen, die Buchfink selbst ausgestellt hat.
// Ein von Hand gebuchter Posten ist kein „ohne Hinweis": Buchfink kennt das
// Dokument nicht, es bleibt beim Verzug nach dreißig Tagen, und der Hinweis
// unter der Forderung sagt, worauf das beruht.
func (s *OverdueService) consumerNoticePrinted(ctx context.Context, documentNumber string) bool {
	if s.invoiceRepo == nil || strings.TrimSpace(documentNumber) == "" {
		return true
	}
	inv, err := s.invoiceRepo.FindByNumber(ctx, documentNumber)
	if err != nil || inv == nil {
		return true
	}
	return inv.ConsumerNoticePrinted
}

// Claims liefert je Kunde die überfälligen Posten mit Verzugszinsen und
// Pauschale zum Stichtag; leer heißt heute.
func (s *OverdueService) Claims(ctx context.Context, today string) ([]OverdueClaim, error) {
	if today == "" {
		today = todayLocal()
	}
	claims := make([]OverdueClaim, 0)
	if s.openItems == nil {
		return claims, nil
	}
	items, err := s.openItems.OpenItems(ctx)
	if err != nil {
		return nil, err
	}
	rates := accounting.DefaultBaseRates()
	reminders, err := s.reminders(ctx)
	if err != nil {
		return nil, err
	}

	byContact := map[uint]*OverdueClaim{}
	order := make([]uint, 0, 8)
	// notes sammelt, was je Kunde nicht gerechnet werden konnte.
	notes := map[uint][]string{}

	for i := range items {
		item := &items[i]
		if item.ContactType != domain.ContactTypeCustomer || item.OpenAmount <= 0 {
			continue
		}
		if item.DueDate == "" || item.DueDate >= today {
			continue
		}

		claim := byContact[item.ContactID]
		if claim == nil {
			// Ein Kunde, dessen Stammdaten sich nicht lesen lassen, fällt nicht
			// heraus: die Forderung besteht. Gerechnet wird dann mit dem
			// Verbrauchersatz ohne Pauschale, weil eine zu hohe Forderung der
			// teurere Fehler ist. Der Grund steht im Hinweis.
			isConsumer := true
			contact, err := s.contact(ctx, item.ContactID)
			if err == nil {
				isConsumer = contact.IsConsumer()
			} else {
				notes[item.ContactID] = append(notes[item.ContactID], fmt.Sprintf(
					"Die Stammdaten dieses Kunden ließen sich nicht lesen (%v); gerechnet ist "+
						"vorsichtshalber mit fünf Prozentpunkten und ohne Pauschale.", err))
			}
			claim = &OverdueClaim{
				ContactID: item.ContactID, ContactName: item.ContactName,
				IsConsumer: isConsumer, Cutoff: today,
			}
			claim.EnsureLists()
			byContact[item.ContactID] = claim
			order = append(order, item.ContactID)
		}

		row := OverdueItem{
			EntryID: item.EntryID, DocumentNumber: item.DocumentNumber,
			DocumentDate: item.DocumentDate, DueDate: item.DueDate,
			OpenAmount: item.OpenAmount, DaysOverdue: daysBetween(item.DueDate, today),
		}
		noticePrinted := true
		if claim.IsConsumer {
			noticePrinted = s.consumerNoticePrinted(ctx, item.DocumentNumber)
		}
		from, err := accounting.DefaultInterestStartFor(item.DueDate, claim.IsConsumer, noticePrinted)
		if last, effective := reminderOf(reminders[item.EntryID], item.DueDate); last != nil {
			row.LastReminder, row.ReminderLevel = last.Date, last.Level
			// Eine Mahnung nach Fälligkeit setzt in Verzug (§ 286 Abs. 1 BGB),
			// verzinst ab dem Folgetag ihres Zugangs; gegenüber einem
			// Verbraucher ohne Hinweis in der Rechnung ist sie der einzige Weg.
			if effective != "" && (err != nil || effective < from) {
				from, err = effective, nil
			}
		}
		if err != nil {
			// Ohne Verzugsbeginn keine Zinsen und keine Pauschale; beide setzen
			// den Verzug voraus.
			row.Note = fmt.Sprintf(
				"Zu %s ist der Verzugsbeginn nicht bestimmbar: %v", documentOf(item), err)
		} else {
			row.DefaultFrom = from
			inDefault := from <= today
			interest, err := accounting.DefaultInterest(
				item.OpenAmount, from, today, claim.IsConsumer, rates)
			if err != nil {
				row.Note = fmt.Sprintf(
					"Zu %s sind keine Verzugszinsen gerechnet: %v.", documentOf(item), err)
			} else {
				row.Interest = interest.Amount
				row.InterestDays = interest.Days
				if interest.UncoveredFrom != "" {
					row.Note = fmt.Sprintf(
						"Zu %s sind Verzugszinsen nur bis zum %s gerechnet: für die Zeit danach ist in "+
							"dieser Programmfassung kein Basiszinssatz hinterlegt. Eine neuere Fassung "+
							"von Buchfink bringt ihn mit.",
						documentOf(item), germanDay(accounting.BaseRatesCoveredUntil(rates)))
				}
			}
			// Die Pauschale richtet sich nach dem Verzug und nicht nach der
			// Zinsrechnung: ein fehlender Basiszinssatz lässt die Zinsen
			// ausfallen, den Verzug aber nicht.
			if !claim.IsConsumer && inDefault {
				row.LumpSum = accounting.DefaultInterestLumpSum
				if err != nil {
					row.Note += " Die Pauschale von 40 € ist trotzdem angesetzt: sie richtet sich nach " +
						"dem Verzug und nicht nach der Zinsrechnung."
				}
			}
		}

		claim.Items = append(claim.Items, row)
		claim.Principal += row.OpenAmount
		claim.Interest += row.Interest
		claim.LumpSum += row.LumpSum
	}

	for _, contactID := range order {
		claim := byContact[contactID]
		sort.SliceStable(claim.Items, func(i, j int) bool {
			return claim.Items[i].DueDate < claim.Items[j].DueDate
		})
		claim.Total = claim.Principal + claim.Interest + claim.LumpSum
		claim.Note = overdueNote(claim, notes[contactID])
		claims = append(claims, *claim)
	}
	// Die älteste Fälligkeit zuerst, dann der größte Betrag.
	sort.SliceStable(claims, func(i, j int) bool {
		if claims[i].Items[0].DueDate != claims[j].Items[0].DueDate {
			return claims[i].Items[0].DueDate < claims[j].Items[0].DueDate
		}
		return claims[i].Total > claims[j].Total
	})
	return claims, nil
}

// documentOf benennt den Posten in einem Hinweis.
func documentOf(item *domain.OpenItem) string {
	if number := strings.TrimSpace(item.DocumentNumber); number != "" {
		return "Beleg " + number
	}
	if number := strings.TrimSpace(item.EntryNumber); number != "" {
		return "Buchung " + number
	}
	return "diesem Posten"
}

// overdueNote ist der Hinweis unter der Forderung eines Kunden.
func overdueNote(c *OverdueClaim, extra []string) string {
	parts := []string{
		"Verzugszinsen und Pauschale werden nicht gebucht: sie sind erst mit der Zahlung Ertrag.",
	}
	if c.IsConsumer {
		parts = append(parts,
			"Der Kunde ist Verbraucher: Verzugszinsen fünf Prozentpunkte über dem Basiszinssatz, "+
				"keine Pauschale (§ 288 Abs. 1 und 5 BGB).",
			"Gegenüber einem Verbraucher setzen die Zinsen voraus, dass die Rechnung auf den "+
				"Verzugseintritt dreißig Tage nach Fälligkeit hingewiesen hat "+
				"(§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB) — fehlt der Hinweis, beginnt der Verzug "+
				"erst mit der Mahnung. Bei den in Buchfink ausgestellten Rechnungen ist das "+
				"berücksichtigt; bei einem anderen Posten ist der Text der Rechnung zu prüfen.")
	} else {
		parts = append(parts,
			"Der Kunde ist kein Verbraucher: Verzugszinsen neun Prozentpunkte über dem Basiszinssatz, "+
				"Pauschale 40 Euro je Forderung (§ 288 Abs. 2 und 5 BGB).")
	}
	for _, item := range c.Items {
		if item.InterestDays > 0 {
			parts = append(parts,
				"Die Zinsen sind auf den heute offenen Betrag gerechnet; eine Teilzahlung während "+
					"des Verzugs bleibt außer Betracht — die Forderung ist dann eher zu niedrig.")
			break
		}
	}
	seen := map[string]bool{}
	for _, item := range c.Items {
		if item.Note == "" || seen[item.Note] {
			continue
		}
		seen[item.Note] = true
		parts = append(parts, item.Note)
	}
	for _, note := range extra {
		if note == "" || seen[note] {
			continue
		}
		seen[note] = true
		parts = append(parts, note)
	}
	return strings.Join(parts, " ")
}

func (s *OverdueService) contact(ctx context.Context, id uint) (*domain.Contact, error) {
	if s.contactRepo == nil {
		return &domain.Contact{ID: id}, nil
	}
	return s.contactRepo.FindByID(ctx, id)
}

// daysBetween zählt die Tage zwischen zwei ISO-Daten; ein unlesbares Datum
// ergibt null.
func daysBetween(from, to string) int {
	a, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0
	}
	b, err := time.Parse("2006-01-02", to)
	if err != nil {
		return 0
	}
	return int(b.Sub(a).Hours() / 24)
}
