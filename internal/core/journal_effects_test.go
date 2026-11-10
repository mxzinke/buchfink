package core

import (
	"context"
	"reflect"
	"testing"
)

func TestJournalEffectRegistration(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Buchungsfolgen GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Read(uiCtx, func(_ context.Context, s *Services) error {
		want := []string{"bank_transaction"}
		if got := s.Journal.EntryEffectNames(); !reflect.DeepEqual(got, want) {
			t.Errorf("Buchungseffekte = %v, erwartet %v", got, want)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
