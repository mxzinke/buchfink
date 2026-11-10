package service

import (
	"context"
	"testing"
)

func TestPostStampsVersionAndActor(t *testing.T) {
	env := newTestEnv(t)

	entry, err := env.journal.Post(context.Background(), simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("die Buchung ist fehlgeschlagen: %v", err)
	}
	if entry.AppVersion == "" {
		t.Error("jede Buchung hat die Programmfassung (UNV-06)")
	}
	if entry.Actor == "" {
		t.Error("jede Buchung hat die Bearbeiterkennung (UNV-04)")
	}
	if entry.PostingRuleVersion == "" {
		t.Error("auch eine von Hand erfasste Buchung hat den Regelstand")
	}
}
