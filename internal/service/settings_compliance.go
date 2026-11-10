package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// SystemChangeHint nennt Umstellungsdatum und Frist für den Zugriff auf das Altsystem.
type SystemChangeHint struct{ Date, Note string }

func (s *SettingsService) SystemChangeHint(ctx context.Context) (SystemChangeHint, error) {
	date, err := s.settingsRepo.Get(ctx, domain.SettingSystemChangeDate)
	if err != nil {
		return SystemChangeHint{}, err
	}
	return SystemChangeHint{Date: date, Note: accounting.SystemChangeNote(date, "")}, nil
}
