package ebilanz

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed taxonomy_6.9.json
var taxonomyJSON []byte

// Taxonomy ist die eingebettete Zuordnung der Gliederung zu den Elementen der
// HGB-Taxonomie.
//
// Sie steht als Ressource und nicht als Tabelle im Code, weil sie mit der
// Taxonomie altert: das Bundesministerium der Finanzen gibt jedes Jahr eine
// neue Fassung heraus, und dann ist genau diese Datei auszutauschen — nicht der
// Übersetzer der Instanz.
type Taxonomy struct {
	Version    string            `json:"version"`
	Date       string            `json:"date"`
	Note       string            `json:"note"`
	Namespaces map[string]string `json:"namespaces"`
	Elements   []TaxonomyElement `json:"elements"`
}

// TaxonomyElement ordnet eine Gliederungsposition einem Taxonomie-Element zu.
type TaxonomyElement struct {
	// Key ist der Schlüssel der Gliederung, z. B. "aktiva.A.II.3".
	Key     string `json:"key"`
	Element string `json:"element"`
	// Verified sagt, ob der Elementname gegen die amtliche Taxonomie geprüft
	// wurde. Er steht auf false, solange das nicht geschehen ist — eine
	// unbelegte Zusicherung wäre schlimmer als das offene Eingeständnis, und
	// die Oberfläche zeigt die Spalte.
	Verified bool `json:"verified"`
}

var (
	taxonomyOnce sync.Once
	taxonomy     *Taxonomy
	taxonomyErr  error
	elementIndex map[string]TaxonomyElement
)

// LoadTaxonomy liest die eingebettete Taxonomie-Ressource.
func LoadTaxonomy() (*Taxonomy, error) {
	taxonomyOnce.Do(func() {
		var t Taxonomy
		if err := json.Unmarshal(taxonomyJSON, &t); err != nil {
			taxonomyErr = fmt.Errorf("die Taxonomie-Ressource konnte nicht gelesen werden: %w", err)
			return
		}
		if len(t.Elements) == 0 {
			taxonomyErr = fmt.Errorf("die Taxonomie-Ressource enthält keine Elemente")
			return
		}
		elementIndex = make(map[string]TaxonomyElement, len(t.Elements))
		for _, e := range t.Elements {
			elementIndex[e.Key] = e
		}
		taxonomy = &t
	})
	return taxonomy, taxonomyErr
}

// ElementFor liefert das Taxonomie-Element einer Gliederungsposition.
//
// Es gibt keinen Auffangwert. Bis hierher landete jedes unbekannte Konto still
// auf „bs.other"; die Instanz war dann formal vollständig und inhaltlich
// falsch, ohne dass es jemand bemerkt hätte. Ein fehlendes Element ist ein
// Befund und keine Randnotiz.
func ElementFor(key string) (TaxonomyElement, bool) {
	if _, err := LoadTaxonomy(); err != nil {
		return TaxonomyElement{}, false
	}
	e, ok := elementIndex[key]
	return e, ok
}
