package declaration

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"sync"
)

//go:embed catalog/*.json
var catalogFiles embed.FS

// Catalog ist der Zeilenkatalog einer Erklärungsart für ein Vordruckjahr.
type Catalog struct {
	Kind     Kind          `json:"art"`
	FormYear int           `json:"vordruckjahr"`
	Version  string        `json:"version"`
	Title    string        `json:"titel"`
	Sources  []Ref         `json:"quellen"`
	Forms    []CatalogForm `json:"formulare"`
}

// CatalogForm ist ein Vordruck im Katalog.
type CatalogForm struct {
	ID       string           `json:"id"`
	Title    string           `json:"titel"`
	Source   string           `json:"quelle"`
	Sections []CatalogSection `json:"abschnitte"`
}

// CatalogSection ist ein Abschnitt im Katalog.
type CatalogSection struct {
	ID    string        `json:"id"`
	Title string        `json:"titel"`
	Lines []CatalogLine `json:"zeilen"`
}

// CatalogLine beschreibt eine Zeile, wie sie der Vordruck des Jahres führt.
type CatalogLine struct {
	Key      string         `json:"schluessel"`
	Row      string         `json:"zeile,omitempty"`
	Code     string         `json:"kz,omitempty"`
	Label    string         `json:"bezeichnung"`
	Unit     Unit           `json:"einheit"`
	Rounding Rounding       `json:"rundung,omitempty"`
	Owner    Responsibility `json:"zustaendig"`
	Input    bool           `json:"eingabe,omitempty"`
	Required bool           `json:"pflicht,omitempty"`
	Legal    []string       `json:"recht,omitempty"`
	Note     string         `json:"hinweis,omitempty"`
	Check    Verification   `json:"pruefvermerk"`
}

var (
	catalogOnce sync.Once
	catalogs    map[Kind][]*Catalog
	catalogErr  error
)

func loadCatalogs() {
	catalogs = map[Kind][]*Catalog{}
	entries, err := catalogFiles.ReadDir("catalog")
	if err != nil {
		catalogErr = err
		return
	}
	for _, e := range entries {
		data, err := catalogFiles.ReadFile(path.Join("catalog", e.Name()))
		if err != nil {
			catalogErr = err
			return
		}
		var c Catalog
		if err := json.Unmarshal(data, &c); err != nil {
			catalogErr = fmt.Errorf("Zeilenkatalog %s: %w", e.Name(), err)
			return
		}
		if err := c.validate(); err != nil {
			catalogErr = fmt.Errorf("Zeilenkatalog %s: %w", e.Name(), err)
			return
		}
		catalogs[c.Kind] = append(catalogs[c.Kind], &c)
	}
	for kind := range catalogs {
		sort.Slice(catalogs[kind], func(i, j int) bool {
			return catalogs[kind][i].FormYear < catalogs[kind][j].FormYear
		})
	}
}

// CatalogFor liefert den Katalog für ein Jahr: den des Jahres, sonst den
// nächstälteren, sonst den ältesten. exact meldet, ob er zum Jahr gehört.
func CatalogFor(kind Kind, year int) (c *Catalog, exact bool, err error) {
	catalogOnce.Do(loadCatalogs)
	if catalogErr != nil {
		return nil, false, catalogErr
	}
	list := catalogs[kind]
	if len(list) == 0 {
		return nil, false, fmt.Errorf("für %s gibt es keinen Zeilenkatalog", kind.Label())
	}
	c = list[0]
	for _, candidate := range list {
		if candidate.FormYear <= year {
			c = candidate
		}
	}
	return c, c.FormYear == year, nil
}

// Line sucht eine Zeile im Katalog.
func (c *Catalog) Line(key string) (CatalogLine, bool) {
	for _, f := range c.Forms {
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				if l.Key == key {
					return l, true
				}
			}
		}
	}
	return CatalogLine{}, false
}

// Keys sind alle Zeilenschlüssel in der Reihenfolge des Katalogs.
func (c *Catalog) Keys() []string {
	out := []string{}
	for _, f := range c.Forms {
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				out = append(out, l.Key)
			}
		}
	}
	return out
}

func (c *Catalog) source(id string) Ref {
	for _, s := range c.Sources {
		if s.ID == id {
			return s
		}
	}
	return Ref{ID: id}
}

func (c *Catalog) validate() error {
	if !c.Kind.Valid() {
		return fmt.Errorf("unbekannte Art %q", c.Kind)
	}
	if c.FormYear < 2000 || c.Version == "" {
		return fmt.Errorf("Vordruckjahr oder Version fehlt")
	}
	seen := map[string]bool{}
	for _, f := range c.Forms {
		if f.ID == "" || f.Title == "" {
			return fmt.Errorf("Vordruck ohne Kennung oder Titel")
		}
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				switch {
				case l.Key == "" || l.Label == "":
					return fmt.Errorf("Zeile ohne Schlüssel oder Bezeichnung in %s", f.ID)
				case seen[l.Key]:
					return fmt.Errorf("Schlüssel %s doppelt", l.Key)
				case l.Check.Source == "":
					return fmt.Errorf("Zeile %s ohne Quelle im Prüfvermerk", l.Key)
				}
				switch l.Unit {
				case UnitEuro, UnitCents, UnitText, UnitYesNo, UnitDate, UnitPercent, UnitKey:
				default:
					return fmt.Errorf("Zeile %s mit unbekannter Einheit %q", l.Key, l.Unit)
				}
				switch l.Owner {
				case OwnerBooks, OwnerAgent, OwnerHuman:
				default:
					return fmt.Errorf("Zeile %s ohne Zuständigkeit", l.Key)
				}
				seen[l.Key] = true
			}
		}
	}
	return nil
}
