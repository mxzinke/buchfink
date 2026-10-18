package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// catalog ist der Werkzeugkatalog mit seiner Stufe, wie ihn
// docs/entwicklung/mcp.md führt.
var catalog = map[string]Tier{
	"status": TierRead, "aufgaben": TierRead, "fristen": TierRead, "konten": TierRead,
	"kontakte_suchen": TierRead, "belege_suchen": TierRead, "beleg_lesen": TierRead,
	"buchungen_suchen": TierRead, "buchung_lesen": TierRead, "offene_posten": TierRead,
	"bankumsaetze": TierRead, "auswertung": TierRead, "anlagen": TierRead, "abschluss_status": TierRead,
	"erklaerung": TierRead, "rechnung_vorschau": TierRead,

	"beleg_ablegen": TierDirect, "kontakt_anlegen": TierDirect, "kontoauszug_importieren": TierDirect,
	"beleg_buchen": TierDirect, "bankumsatz_zuordnen": TierDirect, "erklaerung_exportieren": TierDirect,
	"ebilanz_exportieren": TierDirect, "pflicht_vorbereitet": TierDirect, "standardgruppe_vorschlagen": TierDirect,
	"beleg_datei_exportieren": TierDirect,

	"beleg_buchen_mit_rueckfrage": TierConfirm, "bankumsatz_buchen": TierConfirm, "buchung_erfassen": TierConfirm,
	"buchung_stornieren": TierConfirm, "lohnjournal_buchen": TierConfirm, "rechnung_ausstellen": TierConfirm,
	"mahnung_vermerken": TierConfirm, "abschlussbuchung": TierConfirm, "erklaerung_eingabe": TierConfirm,
	"beleg_verwerfen": TierConfirm, "anlass_erfassen": TierConfirm, "darlehen_erfassen": TierConfirm,
	"ergebnisverwendung_buchen": TierConfirm, "verfahrensdoku_text": TierConfirm,
}

// writeFixtures sind gültige Mindesteingaben der schreibenden Werkzeuge ohne
// mandant_id. Gültig heißt: sie bestehen das Eingabeschema; was danach
// fachlich passiert, prüfen die Szenarien.
var writeFixtures = map[string]map[string]any{
	"beleg_ablegen":               {"eingang_am": "2026-05-04", "dateien": []any{map[string]any{"pfad": "/nicht/da.pdf"}}},
	"kontakt_anlegen":             {"typ": "lieferant", "name": "Matrix GmbH"},
	"kontoauszug_importieren":     {"pfad": "/nicht/da.xml"},
	"beleg_buchen":                bookArgs(1, 1, "2026-05-04", "100.00"),
	"bankumsatz_zuordnen":         {"bankumsatz_id": 1, "buchung_id": 1},
	"erklaerung_exportieren":      {"art": "kst", "jahr": 2026},
	"ebilanz_exportieren":         {"jahr": 2026},
	"beleg_datei_exportieren":     {"beleg_id": 1},
	"pflicht_vorbereitet":         {"schluessel": "ustva.2026-Q1", "notiz": "Kennzahlen geprüft"},
	"standardgruppe_vorschlagen":  {"kontakt_id": 1, "buchungsgruppe": "software", "begruendung": "drei Rechnungen"},
	"beleg_buchen_mit_rueckfrage": bookArgs(1, 1, "2026-05-04", "100.00"),
	"bankumsatz_buchen":           {"bankumsatz_id": 1, "ignorieren": map[string]any{"grund": "privat"}},
	"buchung_erfassen": {"idempotenz_schluessel": "matrix-erfassen", "buchungsdatum": "2026-05-04", "beschreibung": "Umbuchung", "steuerfall": "not_taxable",
		"zeilen": []any{map[string]any{"konto": "1360", "seite": "S", "betrag": "1.00"},
			map[string]any{"konto": "1800", "seite": "H", "betrag": "1.00"}}},
	"buchung_stornieren":  {"buchung_id": 1, "grund": "Fehlbuchung"},
	"lohnjournal_buchen":  {"monat": "2026-05", "beleg_id": 1, "netto": "1.00"},
	"rechnung_ausstellen": {"idempotenz_schluessel": "matrix-rechnung"},
	"mahnung_vermerken":   {"buchung_id": 1, "datum": "2026-05-04", "idempotenz_schluessel": "matrix-mahnung"},
	"abschlussbuchung":    {"art": "ust_verrechnung", "nur_vorschau": true, "idempotenz_schluessel": "matrix-abschluss"},
	"erklaerung_eingabe":  {"art": "kst", "jahr": 2026, "schluessel": "kst.x", "wert": map[string]any{"text": "y"}},
	"beleg_verwerfen":     {"beleg_id": 1, "grund": "Dublette"},
	"anlass_erfassen":     {"art": "gesellschafterwechsel", "datum": "2026-05-04", "idempotenz_schluessel": "matrix-anlass"},
	"darlehen_erfassen": {"kontakt_id": 1, "richtung": "an_gesellschaft", "betrag": "1.00", "zinssatz": "3",
		"beginn": "2026-05-04", "idempotenz_schluessel": "matrix-darlehen"},
	"ergebnisverwendung_buchen": {"geschaeftsjahr": 2025, "beschluss_am": "2026-05-04", "nur_vorschau": true},
	"verfahrensdoku_text":       {},
}

// readFixtures sind gültige Mindesteingaben der lesenden Werkzeuge.
var readFixtures = map[string]map[string]any{
	"auswertung":    {"art": "susa"},
	"konten":        {"art": "suche"},
	"beleg_lesen":   {"beleg_id": 1},
	"buchung_lesen": {"buchung_id": 1},
	"erklaerung":    {"art": "ustva", "zeitraum": "2026-Q1"},
	"rechnung_vorschau": {"kontakt_id": 1, "leistung_von": "2026-05-04",
		"positionen": []any{map[string]any{"text": "Beratung", "menge": "1", "einzelpreis": "100.00", "satz": "19"}}},
}

func fixture(base map[string]any, tenant string) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	if tenant != "" {
		out["mandant_id"] = tenant
	}
	return out
}

func listTools(t *testing.T, cs *mcp.ClientSession) map[string]*mcp.Tool {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		if tool.Name != "test_lieferant" {
			out[tool.Name] = tool
		}
	}
	return out
}

// Der Katalog ist vollständig, jede Stufe hat ihre Annotationen, und kein
// Werkzeug verlangt eine zweite Abfrage durch den Client.
func TestCatalogTiersAndAnnotations(t *testing.T) {
	h := newHarness(t)
	tools := listTools(t, h.mustSession(h.login("Claude Code", "").access))
	specs := map[string]Spec{}
	for _, s := range h.srv.Toolbox().Specs() {
		specs[s.Name] = s
	}
	if len(tools) != len(catalog) {
		t.Errorf("%d Werkzeuge, im Katalog stehen %d", len(tools), len(catalog))
	}
	for name, tier := range catalog {
		tool, ok := tools[name]
		if !ok {
			t.Errorf("%s fehlt", name)
			continue
		}
		if specs[name].Tier != tier {
			t.Errorf("%s hat die Stufe %s, erwartet %s", name, specs[name].Tier, tier)
		}
		a := tool.Annotations
		if a == nil || a.Title == "" || tool.Title == "" || len(tool.Description) < 80 {
			t.Errorf("%s: Titel oder Beschreibung fehlen: %+v", name, tool)
			continue
		}
		if a.OpenWorldHint == nil || *a.OpenWorldHint {
			t.Errorf("%s: openWorldHint muss false sein", name)
		}
		if _, ok := tool.Meta["anthropic/requiresUserInteraction"]; ok {
			t.Errorf("%s fordert eine zweite Abfrage durch den Client an", name)
		}
		schema, _ := json.Marshal(tool.InputSchema)
		switch tier {
		case TierRead:
			if !a.ReadOnlyHint || a.DestructiveHint != nil {
				t.Errorf("%s: lesend, aber %+v", name, a)
			}
		case TierDirect, TierConfirm:
			if a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint != (tier == TierConfirm) {
				t.Errorf("%s: Stufe %s, aber %+v", name, tier, a)
			}
			if !strings.Contains(string(schema), `"mandant_id"`) || !slices.Contains(requiredOf(tool), "mandant_id") {
				t.Errorf("%s verlangt mandant_id nicht: %s", name, schema)
			}
		}
	}
	// Was nur in der App geht, hat kein Werkzeug.
	for name := range tools {
		for _, appOnly := range []string{"festschreib", "uebermittl", "transferticket", "erledigt", "aufstell",
			"feststell", "offenleg", "stammdaten", "gesellschafter", "bankkonto", "bestaetig",
			"pruefermodus", "zulass", "entzieh", "regeln"} {
			if strings.Contains(name, appOnly) {
				t.Errorf("%s klingt nach einem Vorgang, der der App vorbehalten ist", name)
			}
		}
	}
}

func requiredOf(tool *mcp.Tool) []string {
	raw, _ := json.Marshal(tool.InputSchema)
	var s struct {
		Required []string `json:"required"`
	}
	_ = json.Unmarshal(raw, &s)
	return s.Required
}

// docs/entwicklung/mcp.md ist die Quelle für die Skills: jede Zeile der
// Tabelle ist ein Werkzeug mit seiner Stufe und umgekehrt.
func TestDocsListEveryTool(t *testing.T) {
	raw, err := os.ReadFile("../../docs/entwicklung/mcp.md")
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\| (lesen|direkt|rueckfrage) \\|")
	documented := map[string]Tier{}
	for _, m := range row.FindAllStringSubmatch(string(raw), -1) {
		documented[m[1]] = Tier(m[2])
	}
	for name, tier := range catalog {
		if documented[name] != tier {
			t.Errorf("mcp.md: %s steht als %q, erwartet %q", name, documented[name], tier)
		}
	}
	for name := range documented {
		if _, ok := catalog[name]; !ok {
			t.Errorf("mcp.md nennt %s, das es nicht gibt", name)
		}
	}
	for _, code := range []string{CodeNoTenant, CodeTenantNotGranted, CodeTenantSwitched, CodeMissingScope,
		CodeExaminer, CodeAppOnly, CodeApprovalNeeded, CodePathNotAllowed, CodeFileInvalid, CodeKeyReused,
		CodeBusy, CodeInvalidInput, CodeNotFound, CodeRule, CodeInternal} {
		if !strings.Contains(string(raw), "| `"+code+"` |") {
			t.Errorf("mcp.md erklärt den Fehlercode %s nicht", code)
		}
	}
}

// Jedes schreibende Werkzeug prüft Mandant, Recht und Prüfermodus, bevor es
// irgendetwas tut.
func TestWriteToolsGuardTheTenant(t *testing.T) {
	h := newHarness(t)
	full := h.mustSession(h.login("Claude Code", "").access)
	readOnly := h.mustSession(h.login("Nur lesen", "buchfink:lesen").access)
	specs := map[string]Spec{}
	for _, s := range h.srv.Toolbox().Specs() {
		specs[s.Name] = s
	}
	for name, tier := range catalog {
		if tier == TierRead {
			continue
		}
		base, ok := writeFixtures[name]
		if !ok {
			t.Errorf("%s hat keine Mindesteingabe in writeFixtures", name)
			continue
		}
		if e := toolErrorOf(t, callTool(t, full, name, fixture(base, "fremder-mandant"))); e.Code != CodeTenantSwitched {
			t.Errorf("%s mit fremder mandant_id: %+v", name, e)
		}
		if e := toolErrorOf(t, callTool(t, readOnly, name, fixture(base, h.tenant.ID))); e.Code != CodeMissingScope {
			t.Errorf("%s mit Lese-Token: %+v", name, e)
		}
	}
	until := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	for name, base := range writeFixtures {
		res := callTool(t, full, name, fixture(base, h.tenant.ID))
		blocked := res.IsError && toolErrorOf(t, res).Code == CodeExaminer
		if specs[name].ExaminerExempt == blocked {
			t.Errorf("%s im Prüfermodus: %s (ausgenommen: %v)", name, resultText(res), specs[name].ExaminerExempt)
		}
	}
	if n := h.countVendors("Matrix GmbH"); n != 0 {
		t.Error("ein abgewiesener Aufruf hat geschrieben")
	}
}

// Lesende Werkzeuge verlangen die Zulassung für den geöffneten Mandanten und
// das Recht zu lesen; im Prüfermodus lesen sie weiter.
func TestReadToolsNeedGrantAndScope(t *testing.T) {
	h := newHarness(t)
	full := h.mustSession(h.login("Claude Code", "").access)
	writeOnly := h.mustSession(h.login("Nur buchen", "buchfink:buchen").access)
	until := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := h.host.EnableExaminer(uiCtx, until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	for name, tier := range catalog {
		if tier != TierRead || name == "status" {
			continue
		}
		in := fixture(readFixtures[name], "")
		if res := callTool(t, full, name, in); res.IsError && toolErrorOf(t, res).Code == CodeExaminer {
			t.Errorf("%s liest im Prüfermodus nicht: %s", name, resultText(res))
		}
		if e := toolErrorOf(t, callTool(t, writeOnly, name, in)); e.Code != CodeMissingScope {
			t.Errorf("%s ohne Leserecht: %+v", name, e)
		}
	}
	h.createTenant("Andere GmbH")
	for name, tier := range catalog {
		if tier != TierRead || name == "status" {
			continue
		}
		if e := toolErrorOf(t, callTool(t, full, name, fixture(readFixtures[name], ""))); e.Code != CodeTenantNotGranted {
			t.Errorf("%s im nicht freigegebenen Mandanten: %+v", name, e)
		}
	}
}

// Was nur der Mensch in der App tut, hat kein Werkzeug: weder unter eigenem
// Namen noch als Eingabe eines anderen Werkzeugs.
func TestAppOnlyOperationsHaveNoTool(t *testing.T) {
	h := newHarness(t)
	tools := listTools(t, h.mustSession(h.login("Claude Code", "").access))
	appOnly := []string{"festschreib", "uebermitt", "transferticket", "erledigt", "aufstell", "feststell", "offenleg",
		"stammdaten", "bankkonto", "berater", "pruefermodus", "zulass", "entzieh", "direktgrenze", "mindestbuchungen",
		"bestaetig"}
	for name, tool := range tools {
		schema, _ := json.Marshal(tool.InputSchema)
		var in struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		_ = json.Unmarshal(schema, &in)
		for _, word := range appOnly {
			if strings.Contains(name, word) {
				t.Errorf("Werkzeug %s für einen Vorgang der App", name)
			}
			for prop := range in.Properties {
				if strings.Contains(prop, word) {
					t.Errorf("%s nimmt %s", name, prop)
				}
			}
		}
	}
}

// Jedes Werkzeug, das in einem Vorgang schreibt, nimmt einen
// Idempotenzschlüssel; die Werkzeuge aus mehreren Schritten nehmen keinen.
// Wer keinen natürlichen Schlüssel hat, verlangt ihn.
func TestWriteToolsTakeAnIdempotencyKey(t *testing.T) {
	h := newHarness(t)
	tools := listTools(t, h.mustSession(h.login("Claude Code", "").access))
	flows := map[string]bool{"kontoauszug_importieren": true, "erklaerung_exportieren": true, "ebilanz_exportieren": true,
		"beleg_datei_exportieren": true}
	keyed := map[string]bool{"buchung_erfassen": true, "rechnung_ausstellen": true, "abschlussbuchung": true,
		"mahnung_vermerken": true, "anlass_erfassen": true, "darlehen_erfassen": true}
	for name, tier := range catalog {
		if tier == TierRead {
			continue
		}
		schema, _ := json.Marshal(tools[name].InputSchema)
		if has := strings.Contains(string(schema), `"idempotenz_schluessel"`); has == flows[name] {
			t.Errorf("%s: Idempotenzschlüssel %v, mehrschrittig %v", name, has, flows[name])
		}
		if required := slices.Contains(requiredOf(tools[name]), "idempotenz_schluessel"); required != keyed[name] {
			t.Errorf("%s: Schlüssel Pflicht %v, erwartet %v", name, required, keyed[name])
		}
	}
}

// Der Werkzeugkatalog liegt in jedem Gespräch im Kontext des Agenten. Er
// bleibt deshalb knapp: keine veröffentlichten Ausgabeschemas, kurze
// Beschreibungen der wiederkehrenden Felder.
func TestCatalogStaysSmall(t *testing.T) {
	h := newHarness(t)
	tools := listTools(t, h.mustSession(h.login("Claude Code", "").access))
	total := 0
	for name, tool := range tools {
		raw, _ := json.Marshal(tool)
		total += len(raw)
		if tool.OutputSchema != nil {
			t.Errorf("%s veröffentlicht ein Ausgabeschema", name)
		}
	}
	if total > 80_000 {
		t.Errorf("der Katalog hat %d Bytes", total)
	}
}
