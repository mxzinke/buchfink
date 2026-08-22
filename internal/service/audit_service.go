// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package service

import (
	"context"

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
