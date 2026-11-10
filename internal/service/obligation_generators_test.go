package service

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestObligationGeneratorsCoverCatalog(t *testing.T) {
	run := &obligationRun{}
	registered := map[domain.ObligationKind]bool{}
	for _, generator := range run.generators(func(domain.ObligationKind) bool { return true }, nil) {
		if len(generator.kinds) == 0 {
			t.Error("Pflichtgenerator ohne Pflichtarten")
		}
		for _, kind := range generator.kinds {
			if _, ok := obligationType(kind); !ok {
				t.Errorf("Pflichtart %q fehlt im Katalog", kind)
			}
			registered[kind] = true
		}
	}
	for _, definition := range ObligationTypes() {
		if !registered[definition.Kind] {
			t.Errorf("Pflichtart %q hat keinen Generator", definition.Kind)
		}
	}
}
