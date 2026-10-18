package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

// -------------------------------------------------------------------------
// Die Freistellungsbescheinigung steht in der Aufgabenliste
// -------------------------------------------------------------------------

// Die Warnung 30 Tage vor Ablauf erreicht die Aufgabenliste. Eine Frist, die
// keine Liste zeigt, sieht niemand. Sie ist keine Pflicht der Gesellschaft
// gegenüber einer Behörde und steht deshalb nicht im Pflichtenkalender.
func TestExpiringExemptionCertificateShowsUpAmongTheTasks(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	soon := env.vendor(t, "Bau Bald GmbH", "DE", "")
	soon.ExemptionCertificateNumber = "FS-2026-002"
	soon.ExemptionCertificateValidUntil = time.Now().AddDate(0, 0, 20).Format("2006-01-02")
	if err := env.contacts.SaveContact(ctx, soon); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}

	svc := NewTaskService(repository.NewSettingsRepository(env.db))
	svc.SetExemptionSource(env.contacts)
	list, err := svc.Tasks(ctx, TaskOptions{})
	if err != nil {
		t.Fatalf("Aufgabenliste: %v", err)
	}
	for _, task := range list.Upcoming {
		if task.Key != domain.TaskKeyExemption {
			continue
		}
		if task.DueDate != soon.ExemptionCertificateValidUntil {
			t.Errorf("Aufgabe zum %s — erwartet den Ablauftag %s",
				task.DueDate, soon.ExemptionCertificateValidUntil)
		}
		if !strings.Contains(task.Reference, "§ 48b") {
			t.Errorf("Fundstelle %q — erwartet § 48b EStG", task.Reference)
		}
		return
	}
	t.Fatalf("die ablaufende Freistellungsbescheinigung fehlt in der Aufgabenliste: %+v", list)
}

// -------------------------------------------------------------------------
// Freistellungsbescheinigung nach § 48b EStG
// -------------------------------------------------------------------------

// Die Bescheinigung wird mit ihrer Frist überwacht: 30 Tage vorher steht sie in
// der Warnliste, danach als abgelaufen. Wer sie erst am Ablauftag erfährt, hat
// bei der nächsten Zahlung schon einbehalten müssen.
func TestExemptionCertificateWarnsBeforeItExpires(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	valid := env.vendor(t, "Bau Weit GmbH", "DE", "")
	valid.ExemptionCertificateNumber = "FS-2026-001"
	valid.ExemptionCertificateValidUntil = "2027-06-30"
	if err := env.contacts.SaveContact(ctx, valid); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}
	soon := env.vendor(t, "Bau Bald GmbH", "DE", "")
	soon.ExemptionCertificateNumber = "FS-2026-002"
	soon.ExemptionCertificateValidUntil = "2026-04-10"
	if err := env.contacts.SaveContact(ctx, soon); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}
	expired := env.vendor(t, "Bau Vorbei GmbH", "DE", "")
	expired.ExemptionCertificateNumber = "FS-2025-003"
	expired.ExemptionCertificateValidUntil = "2026-02-01"
	if err := env.contacts.SaveContact(ctx, expired); err != nil {
		t.Fatalf("Lieferant: %v", err)
	}

	warnings, err := env.contacts.ExemptionCertificateWarnings(ctx, "2026-03-20")
	if err != nil {
		t.Fatalf("Warnliste: %v", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("%d Warnungen — erwartet die abgelaufene und die bald ablaufende: %+v",
			len(warnings), warnings)
	}
	if warnings[0].State != "expired" || warnings[0].ContactID != expired.ID {
		t.Errorf("erste Warnung %+v — die abgelaufene steht vorn", warnings[0])
	}
	if warnings[1].State != "expiring" || warnings[1].ContactID != soon.ID {
		t.Errorf("zweite Warnung %+v — erwartet die in 21 Tagen ablaufende", warnings[1])
	}
	for _, w := range warnings {
		if !strings.Contains(w.Note, "§ 48 EStG") {
			t.Errorf("die Warnung muss die Folge nennen: %s", w.Note)
		}
	}

	// Ein Kontakt ohne Bescheinigung taucht nicht auf: das Feld wird nur geführt,
	// wo es eine gibt.
	if got := valid.ExemptionCertificateState("2026-03-20"); got != "valid" {
		t.Errorf("Zustand %q — die Bescheinigung läuft noch über ein Jahr", got)
	}
	plain := env.vendor(t, "Ohne Bescheinigung GmbH", "DE", "")
	if got := plain.ExemptionCertificateState("2026-03-20"); got != "" {
		t.Errorf("Zustand %q — ohne erfasste Bescheinigung gibt es keinen", got)
	}
}
