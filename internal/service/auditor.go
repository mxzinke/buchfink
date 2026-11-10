package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

type auditor[ID int | uint | string] struct {
	auditRepo domain.AuditRepository
	entity    string
}

func (a auditor[ID]) audit(ctx context.Context, action domain.AuditAction, id ID, details string) error {
	return a.auditEntity(ctx, action, a.entity, id, details)
}

func (a auditor[ID]) auditEntity(ctx context.Context, action domain.AuditAction, entity string, id ID, details string) error {
	return logAudit(ctx, a.auditRepo, action, entity, fmt.Sprint(id), details)
}

func (a auditor[ID]) logChange(ctx context.Context, action domain.AuditAction, id ID, details string, before, after any) error {
	return logAuditChange(ctx, a.auditRepo, action, a.entity, fmt.Sprint(id), details, before, after)
}
