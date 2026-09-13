package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
)

func TestFoundationConditionalDutyCanBeReopened(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())

	if err := svc.CompleteDuty(ctx, "stammdaten", "2026-02-01", foundationNotApplicable); err == nil {
		t.Fatal("Stammdaten dürfen nicht als nicht zutreffend markiert werden")
	}
	if err := svc.CompleteDuty(ctx, "betriebsnummer", "2026-02-01", foundationNotApplicable); err != nil {
		t.Fatal(err)
	}
	state, err := env.foundations(t).GetState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, duty := range state.Duties {
		if duty.Key == "betriebsnummer" {
			found = true
			if !duty.IsNotApplicable || !duty.IsDone || duty.DoneOn != "2026-02-01" {
				t.Fatalf("Status fehlt nach erneutem Laden: %+v", duty)
			}
		}
	}
	if !found || state.Guide.Done != 1 {
		t.Fatalf("Fortschritt: %+v", state.Guide)
	}

	if err := svc.CompleteDuty(ctx, "betriebsnummer", "", ""); err != nil {
		t.Fatal(err)
	}
	state, err = svc.GetState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, duty := range state.Duties {
		if duty.Key == "betriebsnummer" && (duty.IsDone || duty.IsNotApplicable) {
			t.Fatalf("Aufgabe bleibt abgeschlossen: %+v", duty)
		}
	}
	if state.Guide.Done != 0 {
		t.Fatalf("Fortschritt nach Wiederöffnen: %+v", state.Guide)
	}
}

func TestFoundationFragebogenUsesCurrentSettingsAndDetectsMissingAddress(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())
	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ street, city, period, label string }{
		{"", "", "unknown", "Meldepflicht noch nicht geklärt"},
		{"Teststraße 1", "", "none", "keine regelmäßigen Umsatzsteuererklärungen"},
		{"Teststraße 1", "12345 Teststadt", "month", "monatlich"},
	} {
		cfg.Street, cfg.VatPeriod = tc.street, tc.period
		cfg.PostalCode, cfg.City = domain.SplitLegacyPostalLine(tc.city)
		if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
			t.Fatal(err)
		}
		sheet, err := svc.Fragebogen(ctx)
		if err != nil {
			t.Fatal(err)
		}
		rows := map[string]FragebogenRow{}
		for _, row := range sheet.Rows {
			rows[row.Label] = row
		}
		for label, expected := range map[string]string{"Straße und Hausnummer": tc.street, "Postleitzahl und Ort": tc.city, "Voranmeldungszeitraum": tc.label} {
			row, ok := rows[label]
			if !ok || row.Value != expected || row.Missing != (expected == "") {
				t.Errorf("%s: %+v, erwartet %q", label, row, expected)
			}
		}
		if text := fragebogenMarkdown(sheet); strings.Contains(text, "| , |") {
			t.Fatal("Leere Anschrift wird als Komma ausgegeben")
		}
	}
}

func TestFoundationDependenciesAndEmployeeChoice(t *testing.T) {
	env := newTestEnv(t)
	svc := env.foundations(t)
	ctx := context.Background()
	env.saveFoundation(t, svc, gmbhFoundation())
	setEmployees := func(value string) error {
		repo := repository.NewSettingsRepository(env.db)
		cfg, err := repo.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		cfg.Employees = value
		return repo.UpdateCompanySettings(ctx, cfg)
	}

	duty := func(key string) domain.FoundationDuty {
		t.Helper()
		state, err := svc.GetState(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range state.Duties {
			if d.Key == key {
				return d
			}
		}
		t.Fatalf("Aufgabe fehlt: %s", key)
		return domain.FoundationDuty{}
	}
	if d := duty("fragebogen"); !d.IsPending || !strings.Contains(d.WaitingFor, "Stammdaten") {
		t.Fatalf("Abhängigkeit fehlt: %+v", d)
	}
	if err := svc.SetDutyStatus(ctx, "fragebogen", "done"); err == nil {
		t.Fatal("gesperrte Aufgabe wurde abgeschlossen")
	}
	env.completeFoundationMasterData(t)
	if err := svc.SetDutyStatus(ctx, "stammdaten", "done"); err != nil {
		t.Fatal(err)
	}
	if duty("fragebogen").IsPending {
		t.Fatal("Fragebogen bleibt gesperrt")
	}
	if err := svc.SetDutyStatus(ctx, "ust_id", "deferred"); err == nil {
		t.Fatal("Zurückstellen darf nicht mehr angeboten werden")
	}
	if err := svc.SetDutyStatus(ctx, "ust_id", "skipped"); err != nil {
		t.Fatal(err)
	}
	if !duty("ust_id").IsNotApplicable {
		t.Fatal("Nicht zutreffend fehlt")
	}
	if err := svc.SetDutyStatus(ctx, "ust_id", "open"); err != nil {
		t.Fatal(err)
	}
	if d := duty("ust_id"); d.IsNotApplicable || !d.IsPending {
		t.Fatalf("Wiederöffnen: %+v", d)
	}
	for _, choice := range []string{"none", "yes"} {
		if err := setEmployees(choice); err != nil {
			t.Fatal(err)
		}
		for key := range foundationEmployeeDuties {
			d := duty(key)
			if (choice == "none") != d.IsNotApplicable {
				t.Fatalf("Beschäftigte %s, %s: %+v", choice, key, d)
			}
		}
	}
	if d := duty("betriebsnummer"); !d.IsPending || !strings.Contains(d.WaitingFor, "Unfallversicherung") {
		t.Fatalf("Unternehmensnummer fehlt als Voraussetzung: %+v", d)
	}
	if err := svc.SetDutyStatus(ctx, "unfallversicherung", "done"); err != nil {
		t.Fatal(err)
	}
	if duty("betriebsnummer").IsPending {
		t.Fatal("Betriebsnummer bleibt gesperrt")
	}
	if !duty("sozialversicherung").IsPending {
		t.Fatal("Sozialversicherung ohne Betriebsnummer freigegeben")
	}
	if err := svc.SetDutyStatus(ctx, "betriebsnummer", "done"); err != nil {
		t.Fatal(err)
	}
	if duty("sozialversicherung").IsPending {
		t.Fatal("Sozialversicherung bleibt gesperrt")
	}
	if err := svc.SetDutyStatus(ctx, "betriebsnummer", "open"); err != nil {
		t.Fatal(err)
	}
	if !duty("sozialversicherung").IsPending {
		t.Fatal("Abhängigkeit wird nach Wiederöffnen nicht neu geprüft")
	}
	if err := setEmployees("none"); err != nil {
		t.Fatal(err)
	}
	deadlines := env.deadlines(t)
	deadlines.SetFoundationSource(svc)
	list, err := deadlines.Deadlines(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := deadlineByKey(list, "gruendung.betriebsnummer"); ok {
		t.Fatal("Ohne Beschäftigte erscheint eine Betriebsnummer-Frist")
	}
}

func TestDocumentBrowserUploadPreservesContentsAndMetadata(t *testing.T) {
	env := newTestEnv(t)
	_, documents := openingEnv(t, env)
	ctx := context.Background()
	content := []byte("%PDF-1.4 Testnachweis")
	doc, err := documents.Attach(ctx, DocumentRequest{
		Kind: domain.DocBehoerde, DutyKey: "ihk", FileName: "IHK.pdf", DocumentDate: "2026-08-03",
		ContentBase64: base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.DocumentDate != "2026-08-03" || doc.Note != "" || doc.DutyKey != "ihk" || doc.GeneratedBy != "" {
		t.Fatalf("Metadaten: %+v", doc)
	}
	_, stored, err := documents.Content(ctx, doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, stored) {
		t.Fatal("Datei wurde verändert")
	}
	if _, err := documents.Attach(ctx, DocumentRequest{Kind: domain.DocBehoerde, FileName: "bad.pdf", ContentBase64: "not-base64"}); err == nil {
		t.Fatal("Ungültiger Upload akzeptiert")
	}
}
