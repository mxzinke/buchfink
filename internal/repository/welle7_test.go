package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Ablagen der Welle 7: die gelernte Regel wird fortgeschrieben und nicht
// verdoppelt, der Basiszinssatz ersetzt seinen Stichtag, und die Mahnstufen
// überstehen das Speichern.

func TestBankRuleIsUpdatedInsteadOfDuplicated(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	repo := NewBankRuleRepository(db)

	first := &domain.BankRule{
		Pattern: "meier hausverwaltung | miete buero", Label: "Hausverwaltung Meier",
		CounterAccount: "6310", MoneyIn: false, LastUsedAt: "2026-03-01",
	}
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("Regel anlegen: %v", err)
	}
	// Dieselbe Sache, anderes Konto: der Anwender hat es sich anders überlegt.
	second := &domain.BankRule{
		Pattern: "meier hausverwaltung | miete buero", Label: "Hausverwaltung Meier",
		CounterAccount: "6315", MoneyIn: false, LastUsedAt: "2026-04-01",
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("Regel fortschreiben: %v", err)
	}

	rules, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Regeln lesen: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("erwartet eine Regel, erhalten %d", len(rules))
	}
	if rules[0].CounterAccount != "6315" {
		t.Errorf("Gegenkonto = %q, erwartet das zuletzt bestätigte 6315", rules[0].CounterAccount)
	}
	if rules[0].Hits != 2 {
		t.Errorf("Bestätigungen = %d, erwartet 2", rules[0].Hits)
	}
	if rules[0].LastUsedAt != "2026-04-01" {
		t.Errorf("zuletzt benutzt = %q, erwartet 2026-04-01", rules[0].LastUsedAt)
	}

	if err := repo.Delete(ctx, rules[0].ID); err != nil {
		t.Fatalf("Regel löschen: %v", err)
	}
	if remaining, err := repo.FindAll(ctx); err != nil || len(remaining) != 0 {
		t.Errorf("nach dem Löschen bleiben %d Regeln (%v)", len(remaining), err)
	}
}

func TestBaseRateIsReplacedPerValidFrom(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	repo := NewBaseRateRepository(db)

	if err := repo.Save(ctx, &domain.BaseRate{ValidFrom: "2026-07-01", BasisPoints: 150}); err != nil {
		t.Fatalf("Satz anlegen: %v", err)
	}
	// Die Berichtigung derselben Bekanntgabe ersetzt sie.
	if err := repo.Save(ctx, &domain.BaseRate{ValidFrom: "2026-07-01", BasisPoints: 152}); err != nil {
		t.Fatalf("Satz berichtigen: %v", err)
	}
	rates, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("Sätze lesen: %v", err)
	}
	if len(rates) != 1 || rates[0].BasisPoints != 152 {
		t.Fatalf("erwartet einen Satz mit 152, erhalten %+v", rates)
	}
	if err := repo.Save(ctx, &domain.BaseRate{BasisPoints: 100}); err == nil {
		t.Error("ein Satz ohne Stichtag muss abgewiesen werden")
	}
}

func TestDunningNoticeKeepsItsItems(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	repo := NewDunningRepository(db)

	notice := &domain.DunningNotice{
		FiscalYear: 2026, ContactID: 7, ContactName: "Saumselig GmbH",
		Level: 1, LevelLabel: "Zahlungserinnerung",
		NoticeDate: "2026-04-17", DueDate: "2026-05-01",
		PrincipalAmount: 1_000_000, InterestAmount: 12662, LumpSumAmount: 4000,
		TotalAmount: 1_016_662,
		Items: []domain.DunningNoticeItem{
			{OpenItemEntryID: 42, DocumentNumber: "RE-2026-0001", DueDate: "2026-01-31",
				OpenAmount: 1_000_000, DefaultFrom: "2026-03-03", InterestDays: 45,
				InterestAmount: 12662, Level: 1},
		},
	}
	if err := repo.Create(ctx, notice); err != nil {
		t.Fatalf("Schreiben ablegen: %v", err)
	}

	notices, err := repo.FindByContact(ctx, 7)
	if err != nil {
		t.Fatalf("Verlauf lesen: %v", err)
	}
	if len(notices) != 1 || len(notices[0].Items) != 1 {
		t.Fatalf("erwartet ein Schreiben mit einem Posten, erhalten %+v", notices)
	}

	levels, err := repo.LevelByOpenItem(ctx)
	if err != nil {
		t.Fatalf("Stufen je Posten: %v", err)
	}
	if levels[42] != 1 {
		t.Errorf("Stufe des Postens = %d, erwartet 1", levels[42])
	}
	last, err := repo.LastNoticeByOpenItem(ctx)
	if err != nil {
		t.Fatalf("letztes Schreiben je Posten: %v", err)
	}
	if last[42] != "2026-04-17" {
		t.Errorf("letztes Schreiben = %q, erwartet 2026-04-17", last[42])
	}

	// Ein anderer Kunde hat nichts davon.
	if others, err := repo.FindByContact(ctx, 8); err != nil || len(others) != 0 {
		t.Errorf("fremder Verlauf: %d Schreiben (%v)", len(others), err)
	}
}

// Die Mahnstufen und die Nachweisgrenze überstehen das Speichern; ein leeres
// Feld bleibt die Voreinstellung und schaltet nichts stumm ab.
func TestSettingsKeepDunningLevelsAndCheckThreshold(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	repo := NewSettingsRepository(db)

	defaults, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if len(defaults.DunningLevels) != 3 {
		t.Errorf("erwartet drei voreingestellte Mahnstufen, erhalten %d", len(defaults.DunningLevels))
	}
	if defaults.InvoiceCheckThreshold != 100_000 {
		t.Errorf("Nachweisgrenze = %s €, erwartet 1.000,00", defaults.InvoiceCheckThreshold)
	}

	if err := repo.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
		DunningLevels: []domain.DunningLevel{
			{Label: "Zweite Erinnerung", DaysAfterDue: 30, Fee: 750},
			{Label: "Erste Erinnerung", DaysAfterDue: 10, Fee: 0},
		},
	}); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}

	saved, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if len(saved.DunningLevels) != 2 {
		t.Fatalf("erwartet zwei Mahnstufen, erhalten %d", len(saved.DunningLevels))
	}
	// Sortiert nach Abstand und neu durchnummeriert.
	if saved.DunningLevels[0].Label != "Erste Erinnerung" || saved.DunningLevels[0].Level != 1 {
		t.Errorf("erste Stufe = %+v, erwartet die nach zehn Tagen", saved.DunningLevels[0])
	}
	if saved.DunningLevels[1].Level != 2 || saved.DunningLevels[1].Fee != 750 {
		t.Errorf("zweite Stufe = %+v", saved.DunningLevels[1])
	}
	// Das leere Feld der Grenze wird zur Voreinstellung und nicht zu null.
	if saved.InvoiceCheckThreshold != 100_000 {
		t.Errorf("Nachweisgrenze = %s €, erwartet die Voreinstellung 1.000,00",
			saved.InvoiceCheckThreshold)
	}
}

// Ein Formular, das Mahnstufen und Nachweisgrenze nicht kennt, ändert sie nicht.
//
// Die Einstellungsseite schickt beim Speichern den ganzen Satz Felder. Ein
// Dialog, der nur die Anschrift ändert, schickt für die übrigen Felder den
// Nullwert — und würde eine gepflegte Stufenfolge auf die Voreinstellung
// zurücksetzen, ohne dass jemand das wollte.
func TestSettingsDoNotResetMaintainedValues(t *testing.T) {
	db, err := InitInMemoryDB()
	if err != nil {
		t.Fatalf("Testdatenbank: %v", err)
	}
	ctx := context.Background()
	repo := NewSettingsRepository(db)

	if err := repo.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026,
		InvoiceCheckThreshold: 500_000,
		DunningLevels: []domain.DunningLevel{
			{Label: "Einzige Erinnerung", DaysAfterDue: 14, Fee: 250},
		},
	}); err != nil {
		t.Fatalf("Einstellungen pflegen: %v", err)
	}

	// Ein zweiter Aufruf ohne die beiden Felder.
	if err := repo.UpdateCompanySettings(ctx, &domain.CompanySettings{
		CompanyName: "Pfennig Ventures GmbH", FiscalYear: 2026, Street: "Hauptstraße 1",
	}); err != nil {
		t.Fatalf("Einstellungen speichern: %v", err)
	}

	saved, err := repo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Einstellungen lesen: %v", err)
	}
	if saved.InvoiceCheckThreshold != 500_000 {
		t.Errorf("Nachweisgrenze = %s €, erwartet die gepflegten 5.000,00", saved.InvoiceCheckThreshold)
	}
	if len(saved.DunningLevels) != 1 || saved.DunningLevels[0].Label != "Einzige Erinnerung" {
		t.Errorf("die gepflegte Stufenfolge ist verloren: %+v", saved.DunningLevels)
	}
	if saved.Street != "Hauptstraße 1" {
		t.Errorf("die geänderte Anschrift fehlt: %q", saved.Street)
	}
}
