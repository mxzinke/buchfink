package accounting

import (
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Kette des Änderungsprotokolls ----------------------------------------

func auditEntry(id uint, action domain.AuditAction, details string) domain.AuditLogEntry {
	return domain.AuditLogEntry{
		ID:         id,
		Timestamp:  time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(id) * time.Minute),
		Action:     action,
		EntityType: "CONTACT",
		EntityID:   "7",
		Details:    details,
		Actor:      "buchhalterin@arbeitsplatz",
		Channel:    "ui",
		AppVersion: "1.2.3",
	}
}

// chained verkettet die Einträge so, wie das Repository es tut.
func chained(entries []domain.AuditLogEntry) []domain.AuditLogEntry {
	chain := NewAuditChain()
	prev := domain.GenesisHash
	for i := range entries {
		entries[i].PreviousHash = prev
		entries[i].EntryHash = chain.CalculateHash(&entries[i], prev)
		prev = entries[i].EntryHash
	}
	return entries
}

func TestAuditChainValidAfterSeveralEntries(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	result := NewAuditChain().Verify(entries)
	if !result.IsValid {
		t.Fatalf("die unveränderte Kette muss gültig sein: %s", result.Message)
	}
	if result.CheckedEntries != 3 {
		t.Errorf("%d geprüfte Einträge, erwartet 3", result.CheckedEntries)
	}
	if len(result.Breaks) != 0 {
		t.Errorf("keine Brüche erwartet, bekommen %d", len(result.Breaks))
	}
}

func TestAuditChainDetectsAlteredEntry(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "Betrag 100 € auf 200 € geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	// Wer eine Buchung ändert, ändert danach die Protokollzeile. Genau das
	// findet die Kette.
	entries[1].Details = "Betrag unverändert"

	result := NewAuditChain().Verify(entries)
	if result.IsValid {
		t.Fatal("ein nachträglich geänderter Protokolleintrag muss auffallen")
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != 2 {
		t.Errorf("der erste Bruch muss Eintrag 2 sein, bekommen %v", result.FirstBrokenID)
	}
	if len(result.Breaks) != 1 || result.Breaks[0].Reason != domain.IntegrityBreakContent {
		t.Errorf("erwartet genau einen Inhaltsbruch, bekommen %+v", result.Breaks)
	}
}

func TestAuditChainDetectsRemovedEntry(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	// Der mittlere Eintrag wird entfernt — der Fall, für den es die Kette gibt.
	without := []domain.AuditLogEntry{entries[0], entries[2]}

	result := NewAuditChain().Verify(without)
	if result.IsValid {
		t.Fatal("ein entfernter Protokolleintrag muss die Kette brechen")
	}
	if len(result.Breaks) == 0 || result.Breaks[0].Reason != domain.IntegrityBreakLinkage {
		t.Errorf("erwartet einen Verkettungsbruch, bekommen %+v", result.Breaks)
	}
}

func TestAuditChainReportsEntryWithoutHash(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	// Wer den Eigenhash leert, nimmt den Eintrag nicht aus der Kette.
	entries[1].EntryHash = ""

	result := NewAuditChain().Verify(entries)
	if result.IsValid {
		t.Fatal("ein Eintrag ohne Eigenhash muss als Bruch auffallen")
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != 2 {
		t.Errorf("der erste Bruch muss Eintrag 2 sein, bekommen %v", result.FirstBrokenID)
	}
	if result.CheckedEntries != 3 {
		t.Errorf("%d geprüfte Einträge, erwartet 3", result.CheckedEntries)
	}
}

func TestAuditChainCoversActorAndVersion(t *testing.T) {
	// Bearbeiterkennung und Programmfassung sind der Kern der Nachweispflicht.
	// Wären sie nicht gedeckt, ließen sie sich nachträglich austauschen.
	base := chained([]domain.AuditLogEntry{auditEntry(1, domain.AuditActionUpdate, "geändert")})

	tampered := base[0]
	tampered.Actor = "jemand.anderes@rechner"
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("eine geänderte Bearbeiterkennung muss den Hash ändern")
	}

	tampered = base[0]
	tampered.Channel = "agent:claude-desktop"
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("ein geänderter Erfassungsweg muss den Hash ändern")
	}

	tampered = base[0]
	tampered.AppVersion = "9.9.9"
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("eine geänderte Programmfassung muss den Hash ändern")
	}

	tampered = base[0]
	tampered.Before = `{"name":"anders"}`
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("ein geändertes Vorher muss den Hash ändern")
	}
}
