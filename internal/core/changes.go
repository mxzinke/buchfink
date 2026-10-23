package core

import (
	"context"
	"slices"
	"sync"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// ChangeKind ist die grobe Art einer Änderung, nach der die Oberfläche neu
// lädt.
type ChangeKind string

const (
	ChangeBookings   ChangeKind = "buchungen"
	ChangeReceipts   ChangeKind = "belege"
	ChangeBank       ChangeKind = "bank"
	ChangeMasterData ChangeKind = "stammdaten"
	ChangeClosing    ChangeKind = "abschluss"
	ChangeInbox      ChangeKind = "eingang"
)

// Change meldet einen gelungenen Vorgang: nach jedem Tenant.Write, gleich auf
// welchem Erfassungsweg, und nach einer Änderung im Eingang. Kinds ist leer,
// wenn der Vorgang nichts protokolliert hat.
type Change struct {
	TenantID string
	Channel  actor.Channel
	Kinds    []ChangeKind
}

// changeKinds ordnet die Objekte des Änderungsprotokolls einer Art zu. Jede
// Änderung steht im Protokoll; was ein Vorgang geändert hat, liest der Kern
// deshalb dort ab und nicht an jedem Aufrufer.
var changeKinds = map[string]ChangeKind{
	"JOURNAL": ChangeBookings, "JOURNAL_ENTRY": ChangeBookings, "OPEN_ITEM": ChangeBookings,
	"DUNNING_NOTE": ChangeBookings, "INVOICE": ChangeBookings, "INVOICE_GROUP": ChangeBookings,
	"NUMBER_GAP": ChangeBookings, "GIFT": ChangeBookings, "CURRENCY": ChangeBookings,
	"INPUT_TAX_CORRECTION": ChangeBookings, "ANLAGE": ChangeBookings, "SHAREHOLDER_LOAN": ChangeBookings,

	"RECEIPT": ChangeReceipts, "RECEIPT_FILES": ChangeReceipts,

	"BANK_TX": ChangeBank, "BANK_RULE": ChangeBank, "BANK_ACCOUNT": ChangeBank,

	"SETTINGS": ChangeMasterData, "CONTACT": ChangeMasterData, "ACCOUNT": ChangeMasterData,
	"VAT_ID_CHECK": ChangeMasterData, "PROCEDURE_DOC": ChangeMasterData, "BACKUP_DIR": ChangeMasterData,
	domain.AuditEntityReadOnly: ChangeMasterData,
	"MIGRATION":                ChangeMasterData, "RESTORE": ChangeMasterData,
	"STAMMDATEN_VORSCHLAG": ChangeMasterData,

	"FISCAL_YEAR": ChangeClosing, "JAHRESABSCHLUSS": ChangeClosing, "CLOSING_STEP": ChangeClosing,
	"ACCRUAL": ChangeClosing, "PROVISION": ChangeClosing, "TAX_PROVISION": ChangeClosing,
	"RECEIVABLE_ALLOWANCE": ChangeClosing, "WORK_IN_PROGRESS": ChangeClosing, "APPROPRIATION": ChangeClosing,
	"NOTES": ChangeClosing, "VAT_SETTLEMENT": ChangeClosing, "FESTSCHREIBUNG": ChangeClosing,
	"VAT_RETURN": ChangeClosing, "ZM_RETURN": ChangeClosing, "DECLARATION": ChangeClosing,
	"DECLARATION_INPUT": ChangeClosing, "OBLIGATION": ChangeClosing, "OBLIGATION_EVENT": ChangeClosing,
	"CHECK_RUN": ChangeClosing, "GRUENDUNG": ChangeClosing,
}

// changeSet sammelt die Arten eines Schreibvorgangs.
type changeSet struct {
	mu    sync.Mutex
	kinds []ChangeKind
}

type changeSetKey struct{}

func withChangeSet(ctx context.Context, cs *changeSet) context.Context {
	return context.WithValue(ctx, changeSetKey{}, cs)
}

// noteChange vermerkt einen Protokolleintrag im laufenden Schreibvorgang.
// Ausgaben und Prüfläufe ändern nichts und zählen nicht.
func noteChange(ctx context.Context, action domain.AuditAction, entityType string) {
	cs, _ := ctx.Value(changeSetKey{}).(*changeSet)
	if cs == nil || action == domain.AuditActionExport || action == domain.AuditActionIntegrityCheck {
		return
	}
	kind, ok := changeKinds[entityType]
	if !ok {
		return
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if !slices.Contains(cs.kinds, kind) {
		cs.kinds = append(cs.kinds, kind)
	}
}

func (cs *changeSet) list() []ChangeKind {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	out := append([]ChangeKind{}, cs.kinds...)
	slices.Sort(out)
	return out
}

// recordingAudit ist das Änderungsprotokoll, das jeden geschriebenen Eintrag
// dem laufenden Schreibvorgang meldet.
type recordingAudit struct {
	domain.AuditRepository
}

func (r recordingAudit) Log(ctx context.Context, action domain.AuditAction, entityType, entityID, details string) error {
	if err := r.AuditRepository.Log(ctx, action, entityType, entityID, details); err != nil {
		return err
	}
	noteChange(ctx, action, entityType)
	return nil
}

func (r recordingAudit) LogChange(
	ctx context.Context, action domain.AuditAction, entityType, entityID, details string, before, after any,
) error {
	if err := r.AuditRepository.LogChange(ctx, action, entityType, entityID, details, before, after); err != nil {
		return err
	}
	noteChange(ctx, action, entityType)
	return nil
}

// Notify meldet eine Änderung, die nicht über Tenant.Write lief, etwa eine
// Datei, die in den Eingang kam oder aus ihm nach Übernommen/ ging.
func (t *Tenant) Notify(ctx context.Context, kinds ...ChangeKind) {
	channel, err := channelOf(ctx)
	if err != nil {
		return
	}
	t.emit(Change{TenantID: t.ID(), Channel: channel, Kinds: append([]ChangeKind{}, kinds...)})
}

func (t *Tenant) emit(c Change) {
	if t.onChange != nil {
		t.onChange(c)
	}
}
