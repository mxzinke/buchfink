package wailsbridge

import (
	"context"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
)

// AgentRules sind die Regeln für verbundene Apps: wie gespeichert (null und
// leer heißen Voreinstellung) und wie sie gelten. Den Eingang stellt
// SetInboxFolder ein.
type AgentRules struct {
	Stored            domain.AgentSettings `json:"stored"`
	DirectMaxGross    domain.Cents         `json:"directMaxGross"`
	DirectMinBookings int                  `json:"directMinBookings"`
	ExtraReadFolders  []string             `json:"extraReadFolders"`
	ExportFolder      string               `json:"exportFolder"`
}

func agentRules(s *core.Services, stored *domain.AgentSettings) *AgentRules {
	limits := policy.LimitsFrom(stored)
	folders := policy.FoldersFrom(stored, "", s.Tenant.DataDir)
	out := &AgentRules{Stored: *stored, DirectMaxGross: limits.MaxGross, DirectMinBookings: limits.MinBookings,
		ExportFolder: folders.Export}
	out.Stored.ExtraReadFolders, _ = emptyList(stored.ExtraReadFolders, nil)
	out.ExtraReadFolders, _ = emptyList(folders.Read, nil)
	return out
}

// GetAgentRules liefert Direktgrenze, Mindestzahl früherer Buchungen sowie
// Lese- und Exportordner der verbundenen Apps.
func (b *BuchfinkBridge) GetAgentRules() (*AgentRules, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*AgentRules, error) {
		stored, err := s.Settings.AgentSettings(ctx)
		if err != nil {
			return nil, err
		}
		return agentRules(s, stored), nil
	})
}

// SaveAgentRules ändert die Regeln für verbundene Apps. Null und eine leere
// Zeichenkette stehen für die Voreinstellung, eine leere Liste für keinen
// weiteren Leseordner. Ein Leseordner darf keinen Datenordner und keine
// Sicherung berühren.
func (b *BuchfinkBridge) SaveAgentRules(in domain.AgentSettings) (*AgentRules, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*AgentRules, error) {
		protected := policy.ProtectedDirs(b.host.Config(), s.Tenant.DataDir)
		if err := policy.CheckFolders("", in.ExtraReadFolders, s.Tenant.DataDir, protected); err != nil {
			return nil, err
		}
		stored, err := s.Settings.SaveAgentSettings(ctx, in)
		if err != nil {
			return nil, err
		}
		return agentRules(s, stored), nil
	}, core.AppOnly())
}

// ConfirmDefaultPostingGroup bestätigt die Standard-Buchungsgruppe eines
// Lieferanten, oft den Vorschlag eines Agenten; eine leere Gruppe hebt sie
// auf. Erst mit ihr bucht ein Agent E-Rechnungen des Lieferanten ohne
// Rückfrage.
func (b *BuchfinkBridge) ConfirmDefaultPostingGroup(contactID uint, group string) (*domain.Contact, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Contact, error) {
		return s.Contacts.ConfirmDefaultPostingGroup(ctx, contactID, group)
	}, core.AppOnly())
}
