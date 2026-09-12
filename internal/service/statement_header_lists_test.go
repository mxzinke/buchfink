package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/repository"
)

func TestStatementWithCompleteCompanyDataHasEmptyMissingList(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	repo := repository.NewSettingsRepository(env.db)
	cfg, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.CompanyName, cfg.Seat = "Beispiel GmbH", "Berlin"
	cfg.RegisterCourt, cfg.RegisterNumber = "Amtsgericht Charlottenburg", "HRB 123456 B"
	if err := repo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	header, err := env.statements(t).Header(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"missing":[]`) {
		t.Fatalf("complete company data must return an empty list, not null: %s", data)
	}
}
