package accounting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// AuditChain verkettet und prüft das Änderungsprotokoll.
//
// Das Protokoll ist der Nachweis, dass an den Aufzeichnungen nichts unbemerkt
// geändert wurde (GoBD Rz. 34, § 146 Abs. 4 AO). Es selbst lag bisher
// ungeschützt in einer Tabelle: wer eine Buchung ändert, entfernt danach die
// Protokollzeile, und der Nachweis war der Nachweis von nichts. Die Kette
// schließt diese Lücke mit demselben Mittel wie das Journal — jeder Eintrag
// hat den Hash seines Vorgängers, und ein entfernter Eintrag bricht sie.
type AuditChain struct{}

// NewAuditChain liefert die Kettenimplementierung des Protokolls.
func NewAuditChain() *AuditChain { return &AuditChain{} }

// canonicalizeAudit rendert einen Protokolleintrag in eine eindeutige
// Bytefolge.
//
// Die ID steht bewusst nicht darin: sie wird von der Datenbank vergeben, und
// eine Kopie der Daten in eine neue Datei — der Weg jedes Mandantenumzugs —
// vergäbe andere. Was den Eintrag ausmacht, ist sein Inhalt und seine Stellung
// in der Kette, und beides steht hier.
func canonicalizeAudit(e *domain.AuditLogEntry, prevHash string) []byte {
	var w canonicalWriter
	w.put("prev", prevHash)
	w.put("timestamp", e.Timestamp.UTC().Format(time.RFC3339Nano))
	w.put("action", string(e.Action))
	w.put("entity_type", e.EntityType)
	w.put("entity_id", e.EntityID)
	w.put("details", e.Details)
	w.put("before", e.Before)
	w.put("after", e.After)
	w.put("actor", e.Actor)
	w.put("app_version", e.AppVersion)
	return w.bytes()
}

// CalculateHash liefert den SHA256-Abdruck, der den Eintrag an seinen Vorgänger
// bindet.
func (c *AuditChain) CalculateHash(e *domain.AuditLogEntry, prevHash string) string {
	sum := sha256.Sum256(canonicalizeAudit(e, prevHash))
	return hex.EncodeToString(sum[:])
}

// Verify läuft die Einträge in Protokollreihenfolge ab und prüft beides: die
// Bindung an den Vorgänger und dass jeder Eintrag noch auf seinen gespeicherten
// Eigenhash rechnet.
//
// entries müssen aufsteigend nach ID sortiert sein — in der Reihenfolge, in der
// sie geschrieben wurden.
//
// Einträge ohne Eigenhash sind Altbestand aus der Zeit vor der Kette. Sie
// werden gezählt, aber nicht als Bruch gemeldet: sie wurden nie verkettet, und
// eine Meldung „gebrochen" wäre eine Behauptung über eine Manipulation, die es
// nicht gab. Die Kette beginnt beim ersten Eintrag, der einen Hash hat.
func (c *AuditChain) Verify(entries []domain.AuditLogEntry) domain.AuditChainResult {
	result := domain.AuditChainResult{
		IsValid:          true,
		TotalEntries:     len(entries),
		LastVerifiedHash: domain.GenesisHash,
		Breaks:           make([]domain.AuditChainBreak, 0),
	}

	expectedPrev := domain.GenesisHash
	for i := range entries {
		entry := &entries[i]
		if entry.EntryHash == "" {
			continue
		}
		result.CheckedEntries++

		if entry.PreviousHash != expectedPrev {
			result.IsValid = false
			if result.FirstBrokenID == nil {
				id := entry.ID
				result.FirstBrokenID = &id
			}
			result.Breaks = append(result.Breaks, domain.AuditChainBreak{
				EntryID:      entry.ID,
				Reason:       domain.IntegrityBreakLinkage,
				ExpectedHash: expectedPrev,
				ActualHash:   entry.PreviousHash,
				Message: fmt.Sprintf(
					"Protokolleintrag %d: Die Verkettung zum vorherigen Eintrag weicht ab. Ein Eintrag wurde nachträglich eingefügt oder entfernt.",
					entry.ID),
			})
		}

		if computed := c.CalculateHash(entry, entry.PreviousHash); computed != entry.EntryHash {
			result.IsValid = false
			if result.FirstBrokenID == nil {
				id := entry.ID
				result.FirstBrokenID = &id
			}
			result.Breaks = append(result.Breaks, domain.AuditChainBreak{
				EntryID:      entry.ID,
				Reason:       domain.IntegrityBreakContent,
				ExpectedHash: computed,
				ActualHash:   entry.EntryHash,
				Message: fmt.Sprintf(
					"Protokolleintrag %d: Der Eintrag wurde nach dem Schreiben verändert.",
					entry.ID),
			})
		}

		// Weiter geht es mit dem gespeicherten Eigenhash — sonst schleppte ein
		// einzelner Bruch sich durch das ganze restliche Protokoll.
		expectedPrev = entry.EntryHash
	}

	result.LastVerifiedHash = expectedPrev
	result.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	result.Message = auditChainMessage(result)
	result.EnsureLists()
	return result
}

func auditChainMessage(r domain.AuditChainResult) string {
	switch {
	case r.CheckedEntries == 0:
		return "Das Änderungsprotokoll enthält noch keine verketteten Einträge."
	case r.IsValid && r.CheckedEntries < r.TotalEntries:
		return fmt.Sprintf(
			"Alle %d verketteten Protokolleinträge sind unverändert. %d ältere Einträge stammen aus der Zeit vor der Verkettung und haben keinen Hash.",
			r.CheckedEntries, r.TotalEntries-r.CheckedEntries)
	case r.IsValid:
		return fmt.Sprintf("Alle %d Protokolleinträge sind vollständig und unverändert.", r.CheckedEntries)
	case len(r.Breaks) == 1:
		return "Eine Unstimmigkeit gefunden: " + r.Breaks[0].Message
	default:
		return fmt.Sprintf("%d Unstimmigkeiten gefunden. Die erste: %s", len(r.Breaks), r.Breaks[0].Message)
	}
}
