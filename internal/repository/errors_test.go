package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

func TestMapErrorPreservesCauseAndMessage(t *testing.T) {
	for _, original := range []error{
		gorm.ErrRecordNotFound,
		fmt.Errorf("Nummernkreis JOURNAL konnte nicht gelesen werden: %w", gorm.ErrRecordNotFound),
	} {
		mapped := mapError(original)
		wrapped := fmt.Errorf("Buchung fehlgeschlagen: %w", mapped)
		if mapped.Error() != original.Error() || wrapped.Error() != "Buchung fehlgeschlagen: "+original.Error() {
			t.Fatalf("Fehlertext verändert: %v", wrapped)
		}
		if !errors.Is(wrapped, domain.ErrNotFound) || !errors.Is(wrapped, original) {
			t.Fatalf("Fehlerkennung oder Ursache verloren: %v", wrapped)
		}
		if errors.Unwrap(mapped) != original || mapError(mapped) != mapped {
			t.Fatalf("Fehlerkette verändert: %v", mapped)
		}
	}
	for _, original := range []error{nil, errors.New("Datenbank gesperrt"), domain.ErrNotFound} {
		if got := mapError(original); got != original {
			t.Errorf("anderer Fehler wurde ersetzt: %v", got)
		}
	}
}
