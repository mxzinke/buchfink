package declaration

import (
	"encoding/json"
	"strings"
	"testing"
)

// Jeder Katalog lädt, und jede seiner Zeilen hat eine Rechnung: eine Zeile
// ohne Rechnung stünde in der Anzeige leer, ohne dass jemand es merkt.
func TestCatalogsAreCompleteAndComputed(t *testing.T) {
	for _, kind := range Kinds() {
		cat, exact, err := CatalogFor(kind, 2025)
		if err != nil || !exact {
			t.Fatalf("%s: Katalog 2025 fehlt (%v)", kind, err)
		}
		if len(cat.Keys()) == 0 {
			t.Fatalf("%s: Katalog ohne Zeilen", kind)
		}
		var d *Declaration
		switch kind {
		case KindVatYear:
			d, err = BuildVatYear(common(2025), VatYearData{Ledger: Ledger{}})
		case KindCorporateTax:
			d, err = BuildCorporateTax(common(2025), CorporateData{Ledger: Ledger{}})
		case KindTradeTax:
			d, err = BuildTradeTax(TradeCommon{Common: common(2025)}, CorporateData{Ledger: Ledger{}})
		}
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		noInternalFindings(t, d)
		for _, key := range cat.Keys() {
			if _, ok := d.Line(key); !ok {
				t.Errorf("%s: Zeile %s fehlt in der Erklärung", kind, key)
			}
		}
	}
}

// Eine Zeilennummer steht nur, wo eine amtliche Quelle sie nennt; der
// Prüfvermerk sagt, ob sie am Vordruck selbst geprüft ist.
func TestCatalogRowsCarryTheirSource(t *testing.T) {
	for _, kind := range Kinds() {
		cat, _, err := CatalogFor(kind, 2025)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range cat.Forms {
			for _, s := range f.Sections {
				for _, l := range s.Lines {
					switch kind {
					case KindVatYear:
						if !l.Check.OnForm || l.Row == "" {
							t.Errorf("%s: USt 2 A ist am Vordruck geprüft, Zeile %q ohne Vermerk", l.Key, l.Row)
						}
					default:
						if l.Check.OnForm {
							t.Errorf("%s: Vordrucke KSt und GewSt 2025 liegen nicht vor, trotzdem als geprüft vermerkt", l.Key)
						}
						if l.Row != "" && !strings.Contains(l.Check.Source, "Zeile") {
							t.Errorf("%s: Zeile %s ohne Fundstelle im Prüfvermerk (%s)", l.Key, l.Row, l.Check.Source)
						}
					}
				}
			}
		}
	}
}

func TestFiguresHashFollowsTheValues(t *testing.T) {
	a, _ := BuildVatYear(common(2025), vatYear())
	b, _ := BuildVatYear(common(2025, proposed("ust.z7.art", Text("Software"))), vatYear())
	c, _ := BuildVatYear(common(2025), vatYear())
	if a.FiguresHash != c.FiguresHash {
		t.Errorf("gleiche Zahlen, verschiedener Hash")
	}
	if a.FiguresHash == b.FiguresHash {
		t.Errorf("geänderter Wert, gleicher Hash")
	}
	if len(a.FiguresHash) != 64 {
		t.Errorf("Hash %q ist kein SHA-256", a.FiguresHash)
	}
}

// Der Export nennt das Schema und jeden Betrag zusätzlich als Zeichenkette.
func TestExportJSON(t *testing.T) {
	d, _ := BuildVatYear(common(2025), vatYear())
	data, err := ExportJSON(d)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{`"schema": "buchfink.erklaerung/1"`, `"betrag": "14000.00"`, `"zahlen_hash": "` + d.FiguresHash} {
		if !strings.Contains(text, want) {
			t.Errorf("Export ohne %s", want)
		}
	}
	var back Declaration
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if FiguresHash(&back) != d.FiguresHash {
		t.Errorf("Export und Wiedereinlesen ergeben einen anderen Hash")
	}
}

// Die Übersicht ist die erste Ebene der Anzeige: Ergebnis, Vordrucke und was
// zu klären ist; View klappt einen Vordruck auf.
func TestOverviewAndView(t *testing.T) {
	d, _ := BuildCorporateTax(common(2025), subsidiaryDividend())
	o := d.Overview()
	if len(o.Result) == 0 || o.Result[1].Key != "kst.zve.zve" {
		t.Errorf("Ergebnis: %+v", o.Result)
	}
	if o.Complete || len(o.Pending) == 0 || o.Blocking == 0 {
		t.Errorf("offene Zeilen fehlen in der Übersicht: %+v", o.Pending)
	}
	for _, p := range o.Pending {
		if p.Status != StatusOpen && p.Status != StatusProposed {
			t.Errorf("erledigte Zeile unter zu klären: %+v", p)
		}
	}
	v := d.View(ViewOptions{Form: "GK", Pending: true})
	if len(v.Forms) != 1 || v.Forms[0].ID != "GK" {
		t.Fatalf("View: %d Vordrucke", len(v.Forms))
	}
	for _, s := range v.Forms[0].Sections {
		for _, l := range s.Lines {
			if l.Status != StatusOpen && l.Status != StatusProposed {
				t.Errorf("View zeigt %s mit Stand %s", l.Key, l.Status)
			}
		}
	}
	if v.FiguresHash != d.FiguresHash {
		t.Errorf("die Ansicht ändert den Hash")
	}
	all := d.View(ViewOptions{Applicable: true})
	for _, f := range all.Forms {
		if f.Required == RequirementNone {
			t.Errorf("entfallender Vordruck %s gezeigt", f.ID)
		}
	}
}
