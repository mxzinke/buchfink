package declaration

import (
	"fmt"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// € baut einen Betrag aus Euro und Cent.
func eur(euros int64, cents ...int64) domain.Cents {
	c := domain.Cents(euros * 100)
	if len(cents) > 0 {
		if euros < 0 {
			c -= domain.Cents(cents[0])
		} else {
			c += domain.Cents(cents[0])
		}
	}
	return c
}

// book trägt eine Jahresbewegung als Sollsaldo ein: positiv Aufwand oder
// Aktivzugang, negativ Ertrag oder Passivzugang.
func book(l Ledger, account string, net domain.Cents) {
	t := l[account]
	t.Name = "Konto " + account
	if net >= 0 {
		t.Debit += net
	} else {
		t.Credit += -net
	}
	t.EntryIDs = append(t.EntryIDs, uint(len(l)+1))
	l[account] = t
}

func settings() domain.CompanySettings {
	return domain.CompanySettings{
		CompanyName: "Muster Software GmbH", LegalForm: domain.LegalFormGmbH,
		TaxNumber: "21/815/08150", FederalState: "BE", TaxNumberElster: "1121081508150",
		Street: "Musterstraße", HouseNumber: "1", PostalCode: "10115", City: "Berlin", CountryCode: "DE",
		FoundedOn: "2019-03-01", TaxationType: "SOLL", VatPeriod: "quarter",
	}
}

func common(year int, inputs ...Input) Common {
	return Common{
		Year:     year,
		Period:   Period{Year: year, From: fmt.Sprintf("%d-01-01", year), To: fmt.Sprintf("%d-12-31", year)},
		Settings: settings(),
		Inputs:   inputs,
		Basis:    Basis{ClosingState: string(domain.FiscalYearPrepared)},
	}
}

func confirmed(key string, v Value) Input {
	return Input{Key: key, Value: v, Source: InputFromPerson, Channel: "ui", Confirmed: true}
}

func proposed(key string, v Value) Input {
	return Input{Key: key, Value: v, Source: InputFromPerson, Channel: "agent:claude"}
}

func amount(t *testing.T, d *Declaration, key string) domain.Cents {
	t.Helper()
	l, ok := d.Line(key)
	if !ok {
		t.Fatalf("Zeile %s fehlt", key)
	}
	return l.Value.Amount()
}

func line(t *testing.T, d *Declaration, key string) *Line {
	t.Helper()
	l, ok := d.Line(key)
	if !ok {
		t.Fatalf("Zeile %s fehlt", key)
	}
	return l
}

func wantAmount(t *testing.T, d *Declaration, key string, want domain.Cents) {
	t.Helper()
	if got := amount(t, d, key); got != want {
		t.Errorf("%s: %s €, erwartet %s €", key, got, want)
	}
}

func hasFinding(d *Declaration, code string) bool {
	for _, f := range d.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func noInternalFindings(t *testing.T, d *Declaration) {
	t.Helper()
	for _, f := range d.Findings {
		if f.Code == "intern_zeile_ohne_rechnung" || f.Code == "intern_zeile_unbekannt" {
			t.Errorf("interner Befund: %s %s", f.LineKey, f.Text)
		}
	}
}
