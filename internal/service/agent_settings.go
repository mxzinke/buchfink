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
// gespeichert sind: ein nicht gesetzter Wert bleibt null oder leer und steht
// damit für die Voreinstellung.
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
	out.ExtraReadFolders = []string{}
	if raw, ok, err := s.lookup(ctx, domain.SettingAgentExtraReadFolders); err != nil {
		return nil, err
	} else if ok {
		if err := json.Unmarshal([]byte(raw), &out.ExtraReadFolders); err != nil {
			return nil, fmt.Errorf("die Leseordner ließen sich nicht lesen: %w", err)
		}
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
		domain.SettingAgentExtraReadFolders:  "",
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
	folders := make([]string, 0, len(in.ExtraReadFolders))
	for _, f := range in.ExtraReadFolders {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !filepath.IsAbs(f) {
			return nil, fmt.Errorf("der Leseordner %q ist kein absoluter Pfad", f)
		}
		folders = append(folders, filepath.Clean(f))
	}
	if len(folders) > 0 {
		raw, err := json.Marshal(folders)
		if err != nil {
			return nil, err
		}
		values[domain.SettingAgentExtraReadFolders] = string(raw)
	}
	for _, key := range []string{
		domain.SettingAgentDirectMaxGross, domain.SettingAgentDirectMinBookings,
		domain.SettingAgentExtraReadFolders, domain.SettingAgentExportFolder,
	} {
		if err := s.setLogged(ctx, key, values[key], "Regel für verbundene Apps geändert: "+key); err != nil {
			return nil, err
		}
	}
	return s.AgentSettings(ctx)
}

// setLogged schreibt eine Einstellung mit Protokolleintrag, wenn sie sich
// ändert.
func (s *SettingsService) setLogged(ctx context.Context, key, value, details string) error {
	before, _, err := s.lookup(ctx, key)
	if err != nil {
		return err
	}
	if before == value {
		return nil
	}
	if err := s.settingsRepo.Set(ctx, key, value); err != nil {
		return err
	}
	return logAuditChange(ctx, s.auditRepo, domain.AuditActionUpdate, "SETTINGS", key, details,
		map[string]string{key: before}, map[string]string{key: value})
}

// InboxFolder ist der eingestellte Eingang; leer heißt Voreinstellung
// (<Datenordner>/Eingang).
func (s *SettingsService) InboxFolder(ctx context.Context) (string, error) {
	raw, _, err := s.lookup(ctx, domain.SettingInboxFolder)
	return strings.TrimSpace(raw), err
}

// SetInboxFolder stellt den Eingang ein; leer heißt Voreinstellung. Nur in der
// App: aus dem Eingang lesen verbundene Apps Belege und Kontoauszüge. Ob der
// Ordner einen geschützten umschließt, prüft der Aufrufer
// (policy.CheckFolders), der die Datenordner aller Mandanten kennt.
func (s *SettingsService) SetInboxFolder(ctx context.Context, folder string) (string, error) {
	if err := requireHuman(ctx, "Die Wahl des Eingangs"); err != nil {
		return "", err
	}
	folder = strings.TrimSpace(folder)
	if folder != "" {
		if !filepath.IsAbs(folder) {
			return "", fmt.Errorf("der Eingang %q ist kein absoluter Pfad", folder)
		}
		folder = filepath.Clean(folder)
	}
	if err := s.setLogged(ctx, domain.SettingInboxFolder, folder, "Eingang geändert"); err != nil {
		return "", err
	}
	return folder, nil
}
