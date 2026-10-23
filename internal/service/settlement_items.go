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

// openItemReader berechnet offene Posten aus Buchungen, Ausgleichen und Kontakten.
// Er benötigt keine Zahlungsbuchungen, Anlagen oder Gesellschafterdarlehen.
type openItemReader struct {
	serviceClock
	journalRepo    domain.JournalRepository
	allocationRepo domain.PaymentAllocationRepository
	contactRepo    domain.ContactRepository
}

func (s *openItemReader) OpenItemsAt(ctx context.Context, cutoff string) ([]domain.OpenItem, error) {
	return s.OpenItems(ctx, cutoff)
}

// allocations löst Sammelzahlungen in ihre Einzelzuordnungen auf (GoBD Rz. 36).
func (s *openItemReader) allocations(ctx context.Context, paymentEntryID uint) ([]domain.PaymentAllocationDetail, error) {
	allocations, err := s.allocationRepo.FindByPayment(ctx, paymentEntryID)
	if err != nil {
		return nil, fmt.Errorf("die Zahlungszuordnungen konnten nicht gelesen werden: %w", err)
	}

	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]domain.Contact, len(contacts))
	for _, c := range contacts {
		byID[c.ID] = c
	}

	out := make([]domain.PaymentAllocationDetail, 0, len(allocations))
	for _, a := range allocations {
		detail := domain.PaymentAllocationDetail{PaymentAllocation: a}
		if c, ok := byID[a.ContactID]; ok {
			detail.ContactName = c.LegalName()
			detail.ContactType = c.Type
			detail.LedgerAccount = c.LedgerAccount
		}
		// Der zugeordnete Posten kann aus einem anderen Geschäftsjahr stammen.
		if item, err := s.journalRepo.FindByID(ctx, a.OpenItemEntryID); err == nil && item != nil {
			detail.OpenItemEntryNumber = item.EntryNumber
			detail.DocumentNumber = item.DocumentNumber
			detail.DocumentDate = item.DocumentDate
			detail.Description = item.Description
		}
		out = append(out, detail)
	}
	return out, nil
}

// OpenItems liest den Stand aller Geschäftsjahre. Ohne Stichtag zählt jeder
// Ausgleich, mit Stichtag begrenzt das Buchungsdatum Posten und Zahlungen.
func (s *openItemReader) OpenItems(ctx context.Context, cutoff string) ([]domain.OpenItem, error) {
	var entries []domain.JournalEntry
	var settled map[uint]domain.Cents
	var err error
	if cutoff == "" {
		entries, err = s.journalRepo.FindOpenItemCandidates(ctx, 0)
	} else {
		entries, err = s.journalRepo.FindOpenItemCandidatesAt(ctx, cutoff)
	}
	if err != nil {
		return nil, err
	}
	if cutoff == "" {
		settled, err = s.allocationRepo.SettledByOpenItem(ctx)
	} else {
		settled, err = s.allocationRepo.SettledByOpenItemAt(ctx, cutoff)
	}
	if err != nil {
		return nil, err
	}
	return s.openItemsFrom(ctx, entries, settled)
}

func (s *openItemReader) openItemsFrom(
	ctx context.Context, entries []domain.JournalEntry, settled map[uint]domain.Cents,
) ([]domain.OpenItem, error) {
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]domain.Contact, len(contacts))
	for _, c := range contacts {
		byID[c.ID] = c
	}

	// Leere Listen werden auch in JSON als Listen ausgegeben.
	items := make([]domain.OpenItem, 0, len(entries))
	for i := range entries {
		entry := &entries[i]
		// Zahlungen erzeugen keine eigenen offenen Posten.
		if entry.Source == domain.EntrySourcePayment {
			continue
		}
		// Beim Saldenvortrag bleibt die ursprüngliche Rechnung der offene
		// Posten. Die Eröffnungsbilanz eines Umsteigers begründet dagegen eigene.
		if entry.Source == domain.EntrySourceOpening &&
			entry.DocumentNumber == carryForwardReference(entry.FiscalYear) {
			continue
		}

		line, ok := ledgerLine(entry)
		if !ok || entry.ContactID == nil {
			continue
		}
		contact, ok := byID[*entry.ContactID]
		if !ok {
			continue
		}

		gross := line.Amount
		alreadySettled := settled[entry.ID]
		open := gross - alreadySettled
		if open == 0 {
			continue
		}

		// Ein leerer Steuerfall kennzeichnet eine nicht bestimmbare Steuerkorrektur.
		rate, rateKnown := documentTaxRate(entry)
		treatment := entry.TaxTreatment
		switch {
		case !rateKnown:
			// Eine Steuerzeile, die zu keinem bekannten Satz passt. Dann ist auch
			// der Steuerfall nicht bestimmbar.
			treatment = ""
		case treatment == "":
			// Ohne gespeicherten Steuerfall entscheidet die vorhandene Steuerzeile.
			treatment = domain.TaxTreatmentDomestic
			if rate == domain.TaxRateNone {
				treatment = domain.TaxTreatmentNotTaxable
			}
		}

		items = append(items, domain.OpenItem{
			Source:         domain.OpenItemSourceJournal,
			EntryID:        entry.ID,
			EntryNumber:    entry.EntryNumber,
			ContactID:      contact.ID,
			ContactName:    contact.LegalName(),
			ContactType:    contact.Type,
			LedgerAccount:  line.Account,
			DocumentNumber: entry.DocumentNumber,
			DocumentDate:   entry.DocumentDate,
			DueDate:        dueDate(entry, contact),
			DueFromTerms:   entry.DueDate == "",
			GrossAmount:    gross,
			SettledAmount:  alreadySettled,
			OpenAmount:     open,
			Side:           line.Side,
			Currency:       foreignCurrency(entry.Currency),
			TaxRate:        rate,
			TaxTreatment:   treatment,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].DueDate != items[j].DueDate {
			return items[i].DueDate < items[j].DueDate
		}
		return items[i].EntryNumber < items[j].EntryNumber
	})
	return items, nil
}

func ledgerLine(entry *domain.JournalEntry) (domain.JournalLine, bool) {
	for _, l := range entry.Lines {
		if domain.IsLedgerAccount(l.Account) {
			return l, true
		}
	}
	return domain.JournalLine{}, false
}

// documentTaxRate trennt steuerlose Belege von nicht lesbaren Steuerzeilen.
func documentTaxRate(entry *domain.JournalEntry) (domain.TaxRate, bool) {
	for _, l := range entry.Lines {
		if l.TaxKey == "" || l.TaxBase == 0 {
			continue
		}
		// Der Satz folgt aus Bemessungsgrundlage und Steuerbetrag. Ein Cent
		// Abstand bleibt zulässig: bei Bruttobeträgen ist die Steuer aus dem
		// Bruttobetrag herausgerechnet (§ 35 UStDV).
		switch {
		case (domain.TaxRateStandard.Tax(l.TaxBase) - l.Amount).Abs() <= 1:
			return domain.TaxRateStandard, true
		case (domain.TaxRateReduced.Tax(l.TaxBase) - l.Amount).Abs() <= 1:
			return domain.TaxRateReduced, true
		}
		return domain.TaxRateNone, false
	}
	return domain.TaxRateNone, true
}

func dueDate(entry *domain.JournalEntry, contact domain.Contact) string {
	// Eine vereinbarte Fälligkeit geht dem Zahlungsziel des Kontakts vor.
	if entry.DueDate != "" {
		return entry.DueDate
	}
	days := contact.PaymentTermsDays
	if days <= 0 {
		days = 14
	}
	t, err := time.Parse("2006-01-02", entry.DocumentDate)
	if err != nil {
		return entry.DocumentDate
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

// ageOpenItems berechnet die Altersstruktur einer Auswahl offener Posten.
func ageOpenItems(items []domain.OpenItem, cutoff string) *domain.OpenItemsAging {
	aging := accounting.AgeOpenItems(items, cutoff)
	aging.EnsureLists()
	return &aging
}

// openItemsAging liefert die Altersstruktur zum Stichtag; leer heißt heute.
func (s *openItemReader) openItemsAging(ctx context.Context, cutoff string) (*domain.OpenItemsAging, error) {
	if cutoff == "" {
		cutoff = s.today()
	}
	items, err := s.OpenItems(ctx, cutoff)
	if err != nil {
		return nil, err
	}
	return ageOpenItems(items, cutoff), nil
}

// foreignCurrency ist die Währung einer Buchung, leer bei Euro.
func foreignCurrency(code string) string {
	if code = strings.ToUpper(strings.TrimSpace(code)); code == "EUR" {
		return ""
	}
	return code
}
