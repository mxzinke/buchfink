// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// ContactService manages debtor/creditor master data and their Personenkonten.
type ContactService struct {
	contactRepo domain.ContactRepository
	journalRepo domain.JournalRepository
	numberRepo  domain.NumberRangeRepository
	auditRepo   domain.AuditRepository
	fiscalYear  int
}

// NewContactService creates the master data service.
func NewContactService(
	contactRepo domain.ContactRepository,
	journalRepo domain.JournalRepository,
	numberRepo domain.NumberRangeRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		journalRepo: journalRepo,
		numberRepo:  numberRepo,
		auditRepo:   auditRepo,
		fiscalYear:  fiscalYear,
	}
}

// SetFiscalYear updates the year the open item balances are computed for.
func (s *ContactService) SetFiscalYear(year int) { s.fiscalYear = year }

// GetContacts returns all business partners with the balance of their
// Personenkonto (the open item total) filled in.
func (s *ContactService) GetContacts(ctx context.Context) ([]domain.Contact, error) {
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if s.journalRepo == nil {
		return contacts, nil
	}

	turnovers, err := s.journalRepo.AccountTurnovers(ctx, s.fiscalYear)
	if err != nil {
		return contacts, nil
	}

	for i := range contacts {
		t := turnovers[contacts[i].LedgerAccount]
		if contacts[i].Type == domain.ContactTypeCustomer {
			contacts[i].OpenAmount = t.Debit - t.Credit
		} else {
			contacts[i].OpenAmount = t.Credit - t.Debit
		}
	}
	return contacts, nil
}

// SaveContact stores a business partner, assigning a Personenkonto from the
// DATEV number range on first save.
func (s *ContactService) SaveContact(ctx context.Context, c *domain.Contact) error {
	if c.Type != domain.ContactTypeCustomer && c.Type != domain.ContactTypeVendor {
		return fmt.Errorf("Kontakttyp muss Kunde (Debitor) oder Lieferant (Kreditor) sein")
	}
	if c.Name == "" {
		return fmt.Errorf("Name des Geschäftspartners fehlt")
	}

	action := domain.AuditActionUpdate
	if c.ID == 0 {
		action = domain.AuditActionCreate
	}

	if c.LedgerAccount == "" {
		account, err := s.allocateLedgerAccount(ctx, c.Type)
		if err != nil {
			return err
		}
		c.LedgerAccount = account
	}

	if c.CountryCode == "" {
		c.CountryCode = "DE"
	}

	if err := s.contactRepo.Save(ctx, c); err != nil {
		return err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, action, "CONTACT", fmt.Sprintf("%d", c.ID),
			fmt.Sprintf("Geschäftspartner %s, Personenkonto %s (%s)", c.Name, c.LedgerAccount, c.Type))
	}
	return nil
}

// allocateLedgerAccount takes the next free number from the Debitoren or
// Kreditoren range. Numbers are never reused, so a deleted partner does not free
// its account for a different one — an old booking must stay attributable.
func (s *ContactService) allocateLedgerAccount(ctx context.Context, kind domain.ContactType) (string, error) {
	if s.numberRepo == nil {
		return "", fmt.Errorf("Nummernkreis für Personenkonten ist nicht verfügbar")
	}

	key := domain.NumberRangeDebitor
	if kind == domain.ContactTypeVendor {
		key = domain.NumberRangeCreditor
	}

	for attempt := 0; attempt < 100; attempt++ {
		seq, err := s.numberRepo.Allocate(ctx, key, 0)
		if err != nil {
			return "", err
		}
		account, err := domain.FormatLedgerAccount(kind, seq)
		if err != nil {
			return "", err
		}
		// Guard against a counter that fell behind manually assigned accounts.
		existing, err := s.contactRepo.FindByLedgerAccount(ctx, account)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return account, nil
		}
	}
	return "", fmt.Errorf("es konnte kein freies Personenkonto vergeben werden")
}

// DeleteContact removes a business partner that carries no bookings.
func (s *ContactService) DeleteContact(ctx context.Context, id uint) error {
	contact, err := s.contactRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if s.journalRepo != nil {
		entries, err := s.journalRepo.FindByAccount(ctx, contact.LedgerAccount, 0)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf(
				"%s hat %d Buchungen auf Personenkonto %s und kann nicht gelöscht werden",
				contact.Name, len(entries), contact.LedgerAccount,
			)
		}
	}

	if err := s.contactRepo.Delete(ctx, id); err != nil {
		return err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "CONTACT", fmt.Sprintf("%d", id),
			fmt.Sprintf("Geschäftspartner %s (Personenkonto %s) gelöscht", contact.Name, contact.LedgerAccount))
	}
	return nil
}
