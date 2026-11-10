package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// keepPostingGroup hält die Standard-Buchungsgruppe aus SaveContact heraus.
// Ein Formular, das die Felder nicht kennt, löschte sonst die Bestätigung;
// ein Agent setzte sie. Beim Anlegen nimmt SaveContact einen Vorschlag an und
// eine Gruppe nur aus der App, dann als bestätigt.
func keepPostingGroup(ctx context.Context, c, before *domain.Contact) {
	if before != nil {
		c.DefaultPostingGroup = before.DefaultPostingGroup
		c.DefaultPostingGroupConfirmedAt = before.DefaultPostingGroupConfirmedAt
		c.DefaultPostingGroupConfirmedBy = before.DefaultPostingGroupConfirmedBy
		c.ProposedPostingGroup = before.ProposedPostingGroup
		c.ProposedPostingGroupNote = before.ProposedPostingGroupNote
		c.ProposedPostingGroupBy = before.ProposedPostingGroupBy
		return
	}
	if c.DefaultPostingGroup != "" && requireHuman(ctx, "") == nil && validIncomingGroup(c.DefaultPostingGroup) == nil {
		c.DefaultPostingGroupConfirmedAt = time.Now().UTC().Format(time.RFC3339)
		c.DefaultPostingGroupConfirmedBy = actor.Actor()
	} else {
		c.DefaultPostingGroup, c.DefaultPostingGroupConfirmedAt, c.DefaultPostingGroupConfirmedBy = "", "", ""
	}
	if c.ProposedPostingGroup != "" && validIncomingGroup(c.ProposedPostingGroup) == nil {
		c.ProposedPostingGroupBy = string(actor.RecordedChannel(ctx))
	} else {
		c.ProposedPostingGroup, c.ProposedPostingGroupNote, c.ProposedPostingGroupBy = "", "", ""
	}
}

func validIncomingGroup(key string) error {
	group, err := accounting.LookupPostingGroup(key)
	if err != nil {
		return err
	}
	if group.Direction != domain.DirectionIncoming {
		return fmt.Errorf("die Buchungsgruppe %q gehört zu Ausgangsrechnungen, nicht zu Eingangsbelegen", key)
	}
	return nil
}

// ProposeDefaultPostingGroup hält den Vorschlag einer Standard-Buchungsgruppe
// fest. Er ändert nichts an der Buchung: bis ein Mensch ihn in der App
// bestätigt, bucht ein Agent die Belege des Lieferanten nur mit Rückfrage.
func (s *ContactService) ProposeDefaultPostingGroup(
	ctx context.Context, contactID uint, group, note string,
) (*domain.Contact, error) {
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("Geschäftspartner %d nicht gefunden: %w", contactID, err)
	}
	if contact.Type != domain.ContactTypeVendor {
		return nil, fmt.Errorf("%s ist kein Lieferant; eine Standard-Buchungsgruppe gilt für Eingangsrechnungen", contact.Name)
	}
	group = strings.TrimSpace(group)
	if err := validIncomingGroup(group); err != nil {
		return nil, err
	}
	before := *contact
	contact.ProposedPostingGroup = group
	contact.ProposedPostingGroupNote = strings.TrimSpace(note)
	contact.ProposedPostingGroupBy = string(actor.RecordedChannel(ctx))
	if err := s.contactRepo.Save(ctx, contact); err != nil {
		return nil, err
	}
	details := fmt.Sprintf("Standard-Buchungsgruppe %s für %s vorgeschlagen über %s", group, contact.Name, contact.ProposedPostingGroupBy)
	if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "CONTACT", fmt.Sprintf("%d", contact.ID),
		details, &before, contact); err != nil {
		return nil, err
	}
	return contact, nil
}

// ConfirmDefaultPostingGroup bestätigt die Standard-Buchungsgruppe eines
// Lieferanten; eine leere Gruppe hebt sie auf. Nur in der App: an der
// Bestätigung hängt, ob ein Agent ohne Rückfrage bucht.
func (s *ContactService) ConfirmDefaultPostingGroup(
	ctx context.Context, contactID uint, group string,
) (*domain.Contact, error) {
	if err := requireHuman(ctx, "Das Bestätigen einer Standard-Buchungsgruppe"); err != nil {
		return nil, err
	}
	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("Geschäftspartner %d nicht gefunden: %w", contactID, err)
	}
	group = strings.TrimSpace(group)
	if group != "" {
		if contact.Type != domain.ContactTypeVendor {
			return nil, fmt.Errorf("%s ist kein Lieferant; eine Standard-Buchungsgruppe gilt für Eingangsrechnungen", contact.Name)
		}
		if err := validIncomingGroup(group); err != nil {
			return nil, err
		}
	}
	before := *contact
	contact.DefaultPostingGroup = group
	contact.DefaultPostingGroupConfirmedAt, contact.DefaultPostingGroupConfirmedBy = "", ""
	details := fmt.Sprintf("Standard-Buchungsgruppe von %s aufgehoben", contact.Name)
	if group != "" {
		contact.DefaultPostingGroupConfirmedAt = time.Now().UTC().Format(time.RFC3339)
		contact.DefaultPostingGroupConfirmedBy = actor.Actor()
		details = fmt.Sprintf("Standard-Buchungsgruppe %s für %s bestätigt", group, contact.Name)
	}
	contact.ProposedPostingGroup, contact.ProposedPostingGroupNote, contact.ProposedPostingGroupBy = "", "", ""
	if err := s.contactRepo.Save(ctx, contact); err != nil {
		return nil, err
	}
	if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "CONTACT", fmt.Sprintf("%d", contact.ID),
		details, &before, contact); err != nil {
		return nil, err
	}
	return contact, nil
}
