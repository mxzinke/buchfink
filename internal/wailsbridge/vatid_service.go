package wailsbridge

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/service"
)

// -------------------------------------------------------------------------
// Bestätigung der USt-IdNr.
// -------------------------------------------------------------------------

// GetVatIDStatus beantwortet ohne Netzaufruf, ob eine frische Bestätigung
// vorliegt.
func (b *BuchfinkBridge) GetVatIDStatus(contactID uint) (*service.VatIDStatus, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.VatIDStatus, error) {
		contact, err := s.Repos.Contacts.FindByID(ctx, contactID)
		if err != nil {
			return nil, fmt.Errorf("der Geschäftspartner konnte nicht geladen werden: %w", err)
		}
		return s.VatIDs.Status(ctx, contact)
	})
}
