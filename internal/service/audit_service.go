package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// AuditService retrieves GoBD compliance audit trail logs.
type AuditService struct {
	auditRepo domain.AuditRepository
}

func NewAuditService(auditRepo domain.AuditRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

func (s *AuditService) GetLogs(ctx context.Context, limit int) ([]domain.AuditLogEntry, error) {
	return s.auditRepo.FindAll(ctx, limit)
}

// GetLogsFiltered liefert das Protokoll eingeschränkt durch den Filter.
//
// Das Protokoll wächst mit jeder Buchung, jedem Export und jeder Sicherung.
// Ohne Filter müsste die Oberfläche alles laden und selbst suchen — und die
// Frage „wer hat diesen Kontakt geändert" wäre nur zu beantworten, indem man
// zehntausend Zeilen durchsieht.
func (s *AuditService) GetLogsFiltered(
	ctx context.Context, limit int, filter domain.AuditFilter,
) ([]domain.AuditLogEntry, error) {
	return s.auditRepo.FindFiltered(ctx, limit, filter)
}

// VerifyChain rechnet die Kette des Änderungsprotokolls nach.
//
// Das Protokoll ist der Nachweis, dass an den Aufzeichnungen nichts unbemerkt
// geändert wurde. Ohne diese Prüfung wäre es der Nachweis von nichts: wer eine
// Buchung ändert, entfernt danach die Protokollzeile. Gelesen wird in
// Schreibreihenfolge, weil die Kette in dieser Richtung läuft.
func (s *AuditService) VerifyChain(ctx context.Context) (domain.AuditChainResult, error) {
	entries, err := s.auditRepo.FindAllAscending(ctx)
	if err != nil {
		return domain.AuditChainResult{}, fmt.Errorf("das Änderungsprotokoll konnte nicht gelesen werden: %w", err)
	}
	return accounting.NewAuditChain().Verify(entries), nil
}
