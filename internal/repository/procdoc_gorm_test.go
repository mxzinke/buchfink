package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Fassungen der Verfahrensdokumentation --------------------------------

func TestProcedureDocumentationCountsPerDay(t *testing.T) {
	db := welle6DB(t)
	repo := NewProcedureDocumentationRepository(db)
	ctx := context.Background()

	for _, version := range []string{"2026-09-05-1", "2026-09-05-2", "2026-09-06-1"} {
		if err := repo.Create(ctx, &domain.ProcedureDocumentation{
			Version: version, FiscalYear: 2026, FileName: version + ".md",
		}); err != nil {
			t.Fatalf("Fassung %s konnte nicht abgelegt werden: %v", version, err)
		}
	}

	count, err := repo.CountForDay(ctx, "2026-09-05")
	if err != nil {
		t.Fatalf("die Zählung ist fehlgeschlagen: %v", err)
	}
	if count != 2 {
		t.Errorf("%d Fassungen am 5.9., erwartet 2", count)
	}

	all, _ := repo.FindAll(ctx)
	if len(all) != 3 {
		t.Errorf("%d Fassungen insgesamt, erwartet 3", len(all))
	}
}
