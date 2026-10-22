// Package ops beschreibt Freigabestufen und den Vorbehalt der App.
// Die Durchsetzung bleibt im Kern und in den Diensten.
package ops

import (
	"slices"
	"strings"
)

type Tier string

const (
	Read    Tier = "lesen"
	Direct  Tier = "direkt"
	Confirm Tier = "rueckfrage"
	AppOnly Tier = "nur App"
)

type Enforcement string

const (
	Channel    Enforcement = "Erfassungsweg ui im Kern oder Dienst"
	Structural Enforcement = "strukturell: private API der Bridge"
)

// Operation ordnet einen Vorgang ein. Guards nennt die Schutzstellen im Kern
// und in den Diensten; Bridge nennt die ausschließlich vorbehaltenen Methoden.
// Bedingte Schutzstellen (etwa bestätigte Erklärungsangaben) ordnen nicht die
// ganze Methode als AppOnly ein.
type Operation struct {
	Name        string
	Tier        Tier
	Description string
	Enforcement Enforcement
	Guards      []string
	Bridge      []string
}

func Define(name string, tier Tier) Operation {
	if name == "" || (tier != Read && tier != Direct && tier != Confirm && tier != AppOnly) {
		panic("ungültige Operationsdefinition: " + name)
	}
	return Operation{Name: name, Tier: tier}
}

var tools = []Operation{
	Define("abschluss_status", Read),
	Define("abschlussbuchung", Confirm),
	Define("anlagen", Read),
	Define("anlass_erfassen", Confirm),
	Define("aufgaben", Read),
	Define("auswertung", Read),
	Define("bankumsaetze", Read),
	Define("bankumsatz_buchen", Confirm),
	Define("bankumsatz_zuordnen", Direct),
	Define("beleg_ablegen", Direct),
	Define("beleg_buchen", Direct),
	Define("beleg_buchen_mit_rueckfrage", Confirm),
	Define("beleg_datei_exportieren", Direct),
	Define("beleg_lesen", Read),
	Define("beleg_verwerfen", Confirm),
	Define("belege_suchen", Read),
	Define("buchung_erfassen", Confirm),
	Define("buchung_lesen", Read),
	Define("buchung_stornieren", Confirm),
	Define("buchungen_suchen", Read),
	Define("darlehen_erfassen", Confirm),
	Define("ebilanz_exportieren", Direct),
	Define("eingang", Read),
	Define("ergebnisverwendung_buchen", Confirm),
	Define("erklaerung", Read),
	Define("erklaerung_eingabe", Confirm),
	Define("erklaerung_exportieren", Direct),
	Define("fristen", Read),
	Define("kontakt_anlegen", Direct),
	Define("kontakte_suchen", Read),
	Define("konten", Read),
	Define("kontoauszug_importieren", Direct),
	Define("lohnjournal_buchen", Confirm),
	Define("mahnung_vermerken", Confirm),
	Define("offene_posten", Read),
	Define("pflicht_vorbereitet", Direct),
	Define("rechnung_ausstellen", Confirm),
	Define("rechnung_vorschau", Read),
	Define("stammdaten_vorschlagen", Direct),
	Define("standardgruppe_vorschlagen", Direct),
	Define("status", Read),
	Define("verfahrensdoku_text", Confirm),
}

var catalog = buildCatalog()

func buildCatalog() map[string]Operation {
	out := make(map[string]Operation, len(tools)+len(reserved))
	for _, entries := range [][]Operation{tools, reserved} {
		for _, op := range entries {
			if _, exists := out[op.Name]; exists {
				panic("doppelter Vorgang: " + op.Name)
			}
			out[op.Name] = op
		}
	}
	return out
}

func clone(op Operation) Operation {
	op.Guards = slices.Clone(op.Guards)
	op.Bridge = slices.Clone(op.Bridge)
	return op
}

func Lookup(name string) (Operation, bool) {
	op, ok := catalog[name]
	return clone(op), ok
}

func All() []Operation {
	out := make([]Operation, 0, len(catalog))
	for _, op := range catalog {
		out = append(out, clone(op))
	}
	slices.SortFunc(out, func(a, b Operation) int { return strings.Compare(a.Name, b.Name) })
	return out
}

func ForBridge(method string) (Operation, bool) {
	for _, op := range reserved {
		if slices.Contains(op.Bridge, method) {
			return clone(op), true
		}
	}
	return Operation{}, false
}
