package mcpserver

import (
	"errors"
	"fmt"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestNotFoundErrorMapping(t *testing.T) {
	for _, err := range []error{
		domain.ErrNotFound,
		fmt.Errorf("Beleg lesen: %w", domain.ErrNotFound),
	} {
		t.Run(err.Error(), func(t *testing.T) {
			got := toolError(err)
			if got.Code != "nicht_gefunden" || got.Nachricht != "Eine der angegebenen Kennungen gibt es nicht: "+err.Error() {
				t.Fatalf("Fehlerabbildung: %+v", got)
			}
			got = toolError(notFound("beleg_id", 42, err))
			if got.Code != "nicht_gefunden" || got.Nachricht != "beleg_id 42 gibt es nicht" || got.Werkzeug != "belege_suchen" {
				t.Fatalf("Feldfehler: %+v", got)
			}
		})
	}
	other := errors.New("Lesen fehlgeschlagen")
	if got := notFound("beleg_id", 42, other); got != other {
		t.Fatalf("anderer Fehler wurde ersetzt: %v", got)
	}
	if got := notFound("beleg_id", 42, nil); got != nil {
		t.Fatalf("erfolgreiche Abfrage: %v", got)
	}
}
