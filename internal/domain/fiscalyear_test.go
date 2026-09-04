package domain

import (
	"strings"
	"testing"
)

// Ein Geschäftsjahr darf zwölf Monate nicht überschreiten (§ 240 Abs. 2 Satz 2
// HGB). Das Rumpfgeschäftsjahr darf kürzer sein.
func TestFiscalYearValidateLength(t *testing.T) {
	cases := []struct {
		name       string
		start, end string
		wantErr    bool
		wantShort  bool
	}{
		{"Kalenderjahr", "2026-01-01", "2026-12-31", false, false},
		{"abweichendes Geschäftsjahr", "2026-07-01", "2027-06-30", false, false},
		{"Rumpfgeschäftsjahr ab Beurkundung", "2026-03-15", "2026-12-31", false, true},
		{"ein Tag zu lang", "2026-01-01", "2027-01-01", true, false},
		{"anderthalb Jahre", "2026-01-01", "2027-06-30", true, false},
		{"Ende vor Beginn", "2026-12-31", "2026-01-01", true, false},
	}

	for _, c := range cases {
		fy := NewFiscalYear(2026, c.start, c.end)
		err := fy.Validate()
		if c.wantErr && err == nil {
			t.Errorf("%s: %s bis %s hätte abgelehnt werden müssen", c.name, c.start, c.end)
			continue
		}
		if !c.wantErr && err != nil {
			t.Errorf("%s: %s bis %s wurde abgelehnt: %v", c.name, c.start, c.end, err)
			continue
		}
		if c.wantErr {
			continue
		}
		if fy.IsShort != c.wantShort {
			t.Errorf("%s: Rumpfgeschäftsjahr = %v, erwartet %v", c.name, fy.IsShort, c.wantShort)
		}
	}
}

// Die Zwölfmonatsgrenze muss in der Meldung stehen; sonst weiß niemand, warum
// der Zeitraum abgelehnt wurde.
func TestFiscalYearLengthErrorNamesTheRule(t *testing.T) {
	fy := NewFiscalYear(2026, "2026-01-01", "2027-01-01")
	err := fy.Validate()
	if err == nil {
		t.Fatal("ein Zeitraum von mehr als zwölf Monaten muss abgelehnt werden")
	}
	if !strings.Contains(err.Error(), "zwölf Monate") {
		t.Errorf("die Meldung sollte die Zwölfmonatsgrenze benennen, lautet aber: %v", err)
	}
}

// Zu jedem erreichten Abschlussschritt gehört sein Datum.
func TestFiscalYearStatusNeedsItsDate(t *testing.T) {
	fy := NewFiscalYear(2026, "2026-01-01", "2026-12-31")

	fy.Status = FiscalYearPrepared
	if err := fy.Validate(); err == nil {
		t.Error("„aufgestellt\" ohne Datum der Aufstellung muss abgelehnt werden")
	}

	fy.PreparedOn = "2027-03-31"
	if err := fy.Validate(); err != nil {
		t.Errorf("„aufgestellt\" mit Datum wurde abgelehnt: %v", err)
	}

	fy.Status = FiscalYearAdopted
	if err := fy.Validate(); err == nil {
		t.Error("„festgestellt\" ohne Datum der Feststellung muss abgelehnt werden")
	}

	fy.AdoptedOn = "2027-06-30"
	if err := fy.Validate(); err != nil {
		t.Errorf("„festgestellt\" mit Datum wurde abgelehnt: %v", err)
	}
	if !fy.IsAdopted() {
		t.Error("ein festgestelltes Jahr muss als festgestellt gelten")
	}

	fy.Status = FiscalYearStatus("erledigt")
	if err := fy.Validate(); err == nil {
		t.Error("ein unbekannter Abschlussstand muss abgelehnt werden")
	}
}

// Das Ergebnis geht auf den Gewinnvortrag, der Verlust auf den Verlustvortrag.
func TestResultCarryForwardAccount(t *testing.T) {
	if got := ResultCarryForwardAccount(100000); got != AccountGewinnvortrag {
		t.Errorf("Gewinn gehört auf %s, erhalten %s", AccountGewinnvortrag, got)
	}
	if got := ResultCarryForwardAccount(-100000); got != AccountVerlustvortrag {
		t.Errorf("Verlust gehört auf %s, erhalten %s", AccountVerlustvortrag, got)
	}
	if got := ResultCarryForwardAccount(0); got != AccountGewinnvortrag {
		t.Errorf("ein Ergebnis von null gehört auf %s, erhalten %s", AccountGewinnvortrag, got)
	}
}

// Die Vortragskonten selbst werden nie vorgetragen.
func TestIsCarryForwardAccount(t *testing.T) {
	for _, account := range []string{AccountSaldenvortraegeSachkonten, AccountSaldenvortraegeDebitoren, AccountSaldenvortraegeKreditoren} {
		if !IsCarryForwardAccount(account) {
			t.Errorf("%s ist ein Vortragskonto", account)
		}
	}
	for _, account := range []string{AccountBank, AccountGewinnvortrag, "9090"} {
		if IsCarryForwardAccount(account) {
			t.Errorf("%s ist kein Vortragskonto", account)
		}
	}
}
