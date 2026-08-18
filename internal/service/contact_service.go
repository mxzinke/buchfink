package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// ContactService manages debtor/creditor master data.
type ContactService struct {
	contactRepo domain.ContactRepository
	auditRepo   domain.AuditRepository
}

func NewContactService(
	contactRepo domain.ContactRepository,
	auditRepo domain.AuditRepository,
) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		auditRepo:   auditRepo,
	}
}

func (s *ContactService) GetContacts(ctx context.Context) ([]domain.Contact, error) {
	return s.contactRepo.FindAll(ctx)
}

func (s *ContactService) SaveContact(ctx context.Context, c *domain.Contact) error {
	action := domain.AuditActionCreate
	if c.ID != 0 {
		action = domain.AuditActionUpdate
	}

	if err := s.contactRepo.Save(ctx, c); err != nil {
		return err
	}

	_ = s.auditRepo.Log(
		ctx,
		action,
		"CONTACT",
		fmt.Sprintf("%d", c.ID),
		fmt.Sprintf("Kontakt %s (%s, %s)", c.Name, c.Number, c.Type),
	)

	return nil
}
