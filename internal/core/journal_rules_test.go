package core

import (
	"context"
	"reflect"
	"testing"
)

func TestJournalRuleRegistration(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Regelprüfung GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Read(uiCtx, func(_ context.Context, s *Services) error {
		want := []string{
			"accrual_taxation", "entry", "entertainment", "accounts", "collective_accounts",
			"lawful_tax_disclosure", "tax_lines", "manual_tax_lines", "year_not_adopted",
			"period_open", "receipt_header",
		}
		if got := s.Journal.EntryRuleNames(); !reflect.DeepEqual(got, want) {
			t.Errorf("Journalregeln = %v, erwartet %v", got, want)
		}
		want = []string{"opening_balances", "vat_settlement"}
		if got := s.Journal.TaxAccountExceptionNames(); !reflect.DeepEqual(got, want) {
			t.Errorf("Steuerkontenausnahmen = %v, erwartet %v", got, want)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
