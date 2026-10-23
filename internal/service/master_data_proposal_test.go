package service

import (
	"reflect"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Jedes Feld, das ein Agent vorschlagen kann, steht in der Vorschau. Fehlte
// eines, übernähme der Mensch eine Änderung, die er nicht gesehen hat.
func TestEveryProposableFieldShowsInThePreview(t *testing.T) {
	special := map[string]any{
		"RegisteredOn": "2026-08-24", "FederalState": "HH", "VatPeriod": "month", "TaxationType": "IST",
		"TaxAdvisor":   domain.TaxAdvisor{Engaged: true},
		"Shareholders": []domain.CompanyShareholder{{Name: "Ole Brandt", ShareCapital: 1_000_000}},
	}
	fields := reflect.TypeFor[domain.CompanyChanges]()
	for i := range fields.NumField() {
		field := fields.Field(i)
		var changes domain.CompanyChanges
		target := reflect.ValueOf(&changes).Elem().Field(i)
		value := reflect.New(field.Type.Elem())
		switch v := special[field.Name]; {
		case v != nil:
			value.Elem().Set(reflect.ValueOf(v))
		case field.Type.Elem().Kind() == reflect.String:
			value.Elem().SetString("geändert")
		case field.Type.Elem().Kind() == reflect.Bool:
			value.Elem().SetBool(true)
		case field.Type.Elem().Kind() == reflect.Int64:
			value.Elem().SetInt(12_345)
		default:
			t.Fatalf("%s: kein Testwert für %s", field.Name, field.Type)
		}
		target.Set(value)
		before := &domain.CompanySettings{}
		after := *before
		changes.ApplyTo(&after)
		if lines := companyLines(before, &after); len(lines) != 1 {
			t.Errorf("%s: %d Zeilen in der Vorschau, erwartet eine", field.Name, len(lines))
		}
	}
}

// Ohne Angabe im Vorschlag bleibt das Konto für Rechnungen, wo es ist; gibt es
// keines, wird es das erste neue Konto. Wechselt es, zeigt die Vorschau auch
// das Konto, das die Markierung verliert.
func TestProposedInvoiceAccount(t *testing.T) {
	old := domain.BankAccount{IBAN: "DE02120300000000202051", Name: "Alt", LedgerAccount: "1800", IsInvoiceAccount: true}
	fresh := domain.BankAccount{IBAN: "DE42200411550742138800", Name: "Neu"}

	accounts, err := resolveBankAccounts([]domain.BankAccount{fresh}, []domain.BankAccount{old})
	if err != nil || accounts[0].LedgerAccount != "1810" {
		t.Fatalf("Buchungskonto: %+v %v", accounts, err)
	}
	if target, current := invoiceAccount(accounts, []domain.BankAccount{old}); target != old.IBAN || current != old.IBAN {
		t.Fatalf("ohne Angabe: %s %s", target, current)
	}
	if target, _ := invoiceAccount(accounts, nil); target != fresh.IBAN {
		t.Fatalf("erstes Konto: %s", target)
	}

	accounts[0].IsInvoiceAccount = true
	lines := bankLines(accounts, []domain.BankAccount{old})
	if len(lines) != 2 || lines[1].Before != "Alt · Konto 1800 · Konto für Rechnungen" || lines[1].After != "Alt · Konto 1800" {
		t.Fatalf("Wechsel des Rechnungskontos: %+v", lines)
	}
}
