// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"context"
	"time"
)

// AuditAction defines the type of event logged in the GoBD audit trail.
type AuditAction string

const (
	AuditActionCreate         AuditAction = "CREATE"
	AuditActionUpdate         AuditAction = "UPDATE"
	AuditActionStorno         AuditAction = "STORNO"
	AuditActionImport         AuditAction = "IMPORT"
	AuditActionIntegrityCheck AuditAction = "INTEGRITY_CHECK"
	AuditActionExport         AuditAction = "EXPORT"
)

// AuditLogEntry represents an immutable entry in the GoBD compliance audit trail.
type AuditLogEntry struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	Timestamp    time.Time   `gorm:"not null;index" json:"timestamp"`
	Action       AuditAction `gorm:"size:30;not null;index" json:"action"`
	EntityType   string      `gorm:"size:50;not null;index" json:"entityType"` // "BOOKING", "ACCOUNT", "BANK_TX", "INVOICE", "SETTINGS"
	EntityID     string      `gorm:"size:50;not null" json:"entityId"`
	Details      string      `gorm:"type:text;not null" json:"details"`
	PreviousHash string      `gorm:"size:64" json:"previousHash,omitempty"`
	EntryHash    string      `gorm:"size:64" json:"entryHash,omitempty"`

	// TODO: Add cryptographic signature per audit entry (e.g. Ed25519) if required for extended certifications
}

// AuditRepository defines persistence operations for audit logs.
type AuditRepository interface {
	Log(ctx context.Context, action AuditAction, entityType, entityID, details string) error
	FindAll(ctx context.Context, limit int) ([]AuditLogEntry, error)
	Count(ctx context.Context) (int64, error)
}
