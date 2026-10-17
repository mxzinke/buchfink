package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

// AgentSettings liest die Einstellungen für verbundene Agenten so, wie sie
// gespeichert sind: ein nicht gesetzter Wert bleibt null, nil oder leer und
// steht damit für die Voreinstellung.
func (s *SettingsService) AgentSettings(ctx context.Context) (*domain.AgentSettings, error) {
	out := &domain.AgentSettings{}
	if raw, ok, err := s.lookup(ctx, domain.SettingAgentDirectMaxGross); err != nil {
		return nil, err
	} else if ok {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("die Direktgrenze %q ist keine Zahl", raw)
		}
		out.DirectMaxGross = domain.Cents(v)
	}
	if raw, ok, err := s.lookup(ctx, domain.SettingAgentDirectMinBookings); err != nil {
		return nil, err
	} else if ok {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("die Mindestzahl der Buchungen %q ist keine Zahl", raw)
		}
		out.DirectMinBookings = v
	}
	if raw, ok, err := s.lookup(ctx, domain.SettingAgentReceiptFolders); err != nil {
		return nil, err
	} else if ok {
		folders := []string{}
		if err := json.Unmarshal([]byte(raw), &folders); err != nil {
			return nil, fmt.Errorf("die Belegordner ließen sich nicht lesen: %w", err)
		}
		out.ReceiptFolders = folders
	}
	if raw, ok, err := s.lookup(ctx, domain.SettingAgentExportFolder); err != nil {
		return nil, err
	} else if ok {
		out.ExportFolder = raw
	}
	return out, nil
}

func (s *SettingsService) lookup(ctx context.Context, key string) (string, bool, error) {
	v, err := s.settingsRepo.Get(ctx, key)
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("die Einstellung %s ließ sich nicht lesen: %w", key, err)
	}
	return v, strings.TrimSpace(v) != "", nil
}

// SaveAgentSettings ändert die Einstellungen für verbundene Agenten. Nur in
// der App: von ihnen hängt ab, was ein Agent ohne Rückfrage bucht und welche
// Dateien er lesen darf.
func (s *SettingsService) SaveAgentSettings(ctx context.Context, in domain.AgentSettings) (*domain.AgentSettings, error) {
	if err := requireHuman(ctx, "Die Änderung der Regeln für verbundene Apps"); err != nil {
		return nil, err
	}
	if in.DirectMaxGross < 0 {
		return nil, fmt.Errorf("die Direktgrenze kann nicht negativ sein")
	}
	if in.DirectMinBookings < 0 {
		return nil, fmt.Errorf("die Mindestzahl der Buchungen kann nicht negativ sein")
	}
	values := map[string]string{
		domain.SettingAgentDirectMaxGross:    "",
		domain.SettingAgentDirectMinBookings: "",
		domain.SettingAgentExportFolder:      strings.TrimSpace(in.ExportFolder),
		domain.SettingAgentReceiptFolders:    "",
	}
	if in.DirectMaxGross > 0 {
		values[domain.SettingAgentDirectMaxGross] = strconv.FormatInt(int64(in.DirectMaxGross), 10)
	}
	if in.DirectMinBookings > 0 {
		values[domain.SettingAgentDirectMinBookings] = strconv.Itoa(in.DirectMinBookings)
	}
	if ef := values[domain.SettingAgentExportFolder]; ef != "" && !filepath.IsAbs(ef) {
		return nil, fmt.Errorf("der Exportordner %q ist kein absoluter Pfad", ef)
	}
	if in.ReceiptFolders != nil {
		folders := make([]string, 0, len(in.ReceiptFolders))
		for _, f := range in.ReceiptFolders {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if !filepath.IsAbs(f) {
				return nil, fmt.Errorf("der Belegordner %q ist kein absoluter Pfad", f)
			}
			folders = append(folders, filepath.Clean(f))
		}
		raw, err := json.Marshal(folders)
		if err != nil {
			return nil, err
		}
		values[domain.SettingAgentReceiptFolders] = string(raw)
	}
	for _, key := range []string{
		domain.SettingAgentDirectMaxGross, domain.SettingAgentDirectMinBookings,
		domain.SettingAgentReceiptFolders, domain.SettingAgentExportFolder,
	} {
		before, _, err := s.lookup(ctx, key)
		if err != nil {
			return nil, err
		}
		if before == values[key] {
			continue
		}
		if err := s.settingsRepo.Set(ctx, key, values[key]); err != nil {
			return nil, err
		}
		if err := logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", key,
			"Regel für verbundene Apps geändert: "+key,
			map[string]string{key: before}, map[string]string{key: values[key]}); err != nil {
			return nil, err
		}
	}
	return s.AgentSettings(ctx)
}
