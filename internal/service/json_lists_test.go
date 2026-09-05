package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kein `null` in den Listen, die an die Oberfläche gehen.
//
// Die Ansicht liest die Listen ohne Umweg — `figures.length`,
// `lateEntries.length`, `findings.length`. Ein nicht belegter Go-Slice wird in
// JSON zu `null`, und `null.length` wirft im Render einen TypeError; ohne
// ErrorBoundary nimmt das den ganzen Baum mit. Betroffen wäre jeweils der
// Regelfall: der Zeitraum ohne Nachtrag, die Meldung ohne Befund, der saubere
// Prüflauf. Die Zusage wird deshalb an der Ausgabe geprüft und nicht an den
// Feldern.
func assertNoNullLists(t *testing.T, label string, v any, keys ...string) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("%s als JSON: %v", label, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("%s zurücklesen: %v", label, err)
	}
	for _, key := range keys {
		value, ok := decoded[key]
		if !ok {
			t.Errorf("%s: das Feld %q fehlt in der Ausgabe — die Oberfläche läse `undefined`", label, key)
			continue
		}
		if value == nil {
			t.Errorf("%s: %q ist `null`, erwartet eine leere Liste:\n%s", label, key, raw)
			continue
		}
		if _, ok := value.([]any); !ok {
			t.Errorf("%s: %q ist keine Liste (%T)", label, key, value)
		}
	}
}

// Die leere Voranmeldung: kein Umsatz, kein Nachtrag.
func TestVatReturnMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	draft, err := env.vatReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	assertNoNullLists(t, "Voranmeldung (Entwurf)", draft, "figures", "lateEntries")

	saved, err := env.vatReturns(t).Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	// Auch aus der Datenbank zurückgelesen: gespeichert wird die Liste als JSON,
	// und eine gespeicherte „null" käme sonst als `null` wieder heraus.
	loaded, err := env.vatReturns(t).List(ctx, 2026)
	if err != nil || len(loaded) == 0 {
		t.Fatalf("gespeicherte Anmeldungen lesen: %v (%d)", err, len(loaded))
	}
	if loaded[0].ID != saved.ID {
		t.Fatalf("erwartet die gespeicherte Anmeldung %d, erhalten %d", saved.ID, loaded[0].ID)
	}
	assertNoNullLists(t, "Voranmeldung (gespeichert)", loaded[0], "figures", "lateEntries")
}

// Die leere Zusammenfassende Meldung: keine Zeile, kein Befund, kein Nachtrag.
func TestZMReturnMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	draft, err := env.zmReturns(t).Draft(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Entwurf: %v", err)
	}
	assertNoNullLists(t, "Zusammenfassende Meldung (Entwurf)", draft, "lines", "lateEntries", "findings")

	saved, err := env.zmReturns(t).Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("speichern: %v", err)
	}
	assertNoNullLists(t, "Zusammenfassende Meldung (gespeichert)", saved, "lines", "lateEntries", "findings")
}

// Der saubere Prüflauf — genau der Fall, in dem der Festschreibungsdialog
// `findings.length` liest.
func TestCheckRunMarshalsEmptyFindingsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	preview, err := env.checks(t).Preview(ctx, CheckRequest{CutoffDate: "2026-01-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if len(preview.Findings) != 0 {
		t.Fatalf("für diesen Fall wird ein Lauf ohne Befund erwartet: %+v", preview.Findings)
	}
	assertNoNullLists(t, "Prüflauf (Vorschau)", preview, "findings")

	run, err := env.checks(t).Run(ctx, CheckRequest{CutoffDate: "2026-01-31", PeriodType: "month"})
	if err != nil {
		t.Fatalf("Prüflauf: %v", err)
	}
	assertNoNullLists(t, "Prüflauf (gespeichert)", run, "findings")

	stored, err := env.checks(t).Runs(ctx, 2026)
	if err != nil || len(stored) == 0 {
		t.Fatalf("gespeicherte Läufe lesen: %v (%d)", err, len(stored))
	}
	assertNoNullLists(t, "Prüflauf (zurückgelesen)", stored[0], "findings")
}

// Die Fristenliste ist selbst die Liste: leer statt `null`, damit die Ansicht
// darüber laufen kann.
func TestDeadlineListMarshalsEmptyNotNull(t *testing.T) {
	env := newTestEnv(t)
	deadlines, err := env.deadlines(t).Deadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Termine: %v", err)
	}
	if deadlines == nil {
		t.Fatal("die Fristenliste darf nicht nil sein")
	}
	raw, err := json.Marshal(deadlines)
	if err != nil {
		t.Fatalf("Termine als JSON: %v", err)
	}
	if string(raw) == "null" {
		t.Error("die Fristenliste kommt als `null` in der Oberfläche an")
	}
}

// Die Entitäten sagen die leeren Listen selbst zu — an ihnen hängt die Zusage,
// nicht an den Stellen, die sie erzeugen.
func TestEnsureListsReplacesNilWithEmpty(t *testing.T) {
	var vat domain.VatReturn
	vat.EnsureLists()
	assertNoNullLists(t, "leere Voranmeldung", vat, "figures", "lateEntries")

	var zm domain.ZMReturn
	zm.EnsureLists()
	assertNoNullLists(t, "leere Zusammenfassende Meldung", zm, "lines", "lateEntries", "findings")

	var run domain.CheckRun
	run.EnsureLists()
	assertNoNullLists(t, "leerer Prüflauf", run, "findings")

	// Gegenprobe: ohne die Zusage steht dort `null`.
	raw, err := json.Marshal(domain.VatReturn{})
	if err != nil {
		t.Fatalf("Gegenprobe: %v", err)
	}
	if !strings.Contains(string(raw), `"figures":null`) {
		t.Errorf("die Gegenprobe trägt nicht mehr — der Test prüft dann nichts mehr: %s", raw)
	}
}

// assertNoNilSlices sucht in der Ausgabe eines Dienstes nach nicht belegten
// Slices.
//
// Die Zusage gilt für alles, was als JSON an die Oberfläche geht: ein nil-Slice
// wird dort zu `null`, und `null.length` oder `null.map` nimmt im Render den
// ganzen Baum mit. Geprüft wird über die Struktur und nicht über einzelne
// Feldnamen — die Bausteine des Abschlusses tragen zu viele Listen, als dass
// eine Aufzählung vollständig bliebe.
//
// Ausgenommen sind Felder mit `omitempty`: sie stehen bei nil gar nicht in der
// Ausgabe, und die Ansicht liest sie ausdrücklich als optional. Sie zu belegen
// hieße nur, jede Zeile der Gliederung um eine leere Liste zu verlängern.
func assertNoNilSlices(t *testing.T, label string, value any) {
	t.Helper()
	walkForNilSlices(t, label, reflect.ValueOf(value))
}

func walkForNilSlices(t *testing.T, path string, v reflect.Value) {
	t.Helper()
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return
		}
		walkForNilSlices(t, path, v.Elem())
	case reflect.Slice:
		if v.IsNil() {
			t.Errorf("%s ist nil — an der Oberfläche käme dort `null` an", path)
			return
		}
		for i := 0; i < v.Len(); i++ {
			walkForNilSlices(t, path+"[]", v.Index(i))
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			walkForNilSlices(t, path+"."+key.String(), v.MapIndex(key))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			tag := field.Tag.Get("json")
			if tag == "-" || strings.Contains(tag, ",omitempty") {
				continue
			}
			walkForNilSlices(t, path+"."+field.Name, v.Field(i))
		}
	}
}

// Die Auswertungen des leeren Mandanten. Er ist der Zustand nach der
// Einrichtung — und genau der Bildschirm, den ein neuer Anwender zuerst sieht.
func TestServiceOutputsHaveNoNilLists(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	entries, err := env.accounting.GetEntries(ctx)
	if err != nil {
		t.Fatalf("Journal: %v", err)
	}
	assertNoNilSlices(t, "Journal", entries)

	all, err := env.accounting.GetAllEntries(ctx)
	if err != nil {
		t.Fatalf("Journal über alle Jahre: %v", err)
	}
	assertNoNilSlices(t, "Journal über alle Jahre", all)

	susa, err := env.accounting.GetSuSaOverview(ctx)
	if err != nil {
		t.Fatalf("Summen- und Saldenliste: %v", err)
	}
	assertNoNilSlices(t, "Summen- und Saldenliste", susa)

	ledger, err := env.accounting.GetAccountLedger(ctx, domain.AccountBank)
	if err != nil {
		t.Fatalf("Kontoblatt: %v", err)
	}
	assertNoNilSlices(t, "Kontoblatt", ledger)

	items, err := env.payments(t).OpenItems(ctx)
	if err != nil {
		t.Fatalf("Offene Posten: %v", err)
	}
	assertNoNilSlices(t, "Offene Posten", items)

	statement, err := env.statements(t).Build(ctx, env.fiscalYear, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz und GuV: %v", err)
	}
	assertNoNilSlices(t, "Bilanz und GuV", statement)

	assets := env.assets(t)
	list, err := assets.List(ctx, "")
	if err != nil {
		t.Fatalf("Anlagenverzeichnis: %v", err)
	}
	assertNoNilSlices(t, "Anlagenverzeichnis", list)

	spiegel, err := assets.Anlagenspiegel(ctx)
	if err != nil {
		t.Fatalf("Anlagenspiegel: %v", err)
	}
	assertNoNilSlices(t, "Anlagenspiegel", spiegel)

	run, err := assets.Run(ctx)
	if err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	assertNoNilSlices(t, "Abschreibungslauf", run)

	transactions, err := env.banking(t).GetTransactions(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Bankumsätze: %v", err)
	}
	assertNoNilSlices(t, "Bankumsätze", transactions)

	invoices, err := env.invoices(t).GetInvoices(ctx, env.fiscalYear)
	if err != nil {
		t.Fatalf("Ausgangsrechnungen: %v", err)
	}
	assertNoNilSlices(t, "Ausgangsrechnungen", invoices)
}

// Der Belegprüflauf ohne Befund. Genau der Regelfall wäre betroffen: die
// unversehrte Ablage.
func TestFileCheckResultMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)

	result, err := env.receipts.VerifyReceiptFiles(context.Background())
	if err != nil {
		t.Fatalf("Belegprüflauf: %v", err)
	}
	assertNoNullLists(t, "Belegprüflauf", result, "issues")
}

// Die Integritätsprüfung der unversehrten Buchführung.
func TestIntegrityResultMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000)); err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	result, err := env.journal.VerifyIntegrity(ctx)
	if err != nil {
		t.Fatalf("Integritätsprüfung: %v", err)
	}
	assertNoNullLists(t, "Integritätsprüfung", result, "breaks", "fiscalYears")
}

// Das Ergebnis eines Exports geht ebenfalls an die Oberfläche.
func TestExportResultMarshalsEmptyListsNotNull(t *testing.T) {
	env := newTestEnv(t)

	dir := filepath.Join(t.TempDir(), "z3")
	result, err := env.exports(t).ExportZ3(context.Background(), env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}
	assertNoNullLists(t, "Export", result, "tables", "files", "notes")
}

// Die Anlagenseite des leeren Mandanten: keine Frist läuft ab, kein Zugang
// wartet auf ein Anlagegut. Das ist der Regelfall — und genau er stand bisher
// als `null` in der Ausgabe.
func TestAssetListsAreEmptyNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := env.assets(t)

	expiring, err := svc.ExpiringDocuments(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("ablaufende Dokumente: %v", err)
	}
	assertNoNilSlices(t, "ablaufende Dokumente", expiring)

	candidates, err := svc.AcquisitionCandidates(ctx)
	if err != nil {
		t.Fatalf("Zugangskandidaten: %v", err)
	}
	assertNoNilSlices(t, "Zugangskandidaten", candidates)

	// Der AfA-Plan wird gerechnet, während die Maske noch gefüllt wird: ohne
	// Anschaffungskosten gibt es kein Jahr, aber weiterhin eine Liste.
	plan, err := svc.PreviewPlan(ctx, PlanRequest{
		AcquisitionDate: "2026-03-15", UsefulLifeMonths: 36,
		Method: domain.DepreciationLinear,
	})
	if err != nil {
		t.Fatalf("AfA-Vorschau: %v", err)
	}
	assertNoNilSlices(t, "AfA-Vorschau ohne Anschaffungskosten", plan)

	// Ein Anlagegut, dessen Klasse keine Erläuterung trägt, liefert trotzdem
	// eine Liste — die Ansicht läuft über sie.
	asset := env.machine(t, svc)
	detail, err := svc.Get(ctx, asset.ID)
	if err != nil {
		t.Fatalf("Anlagegut lesen: %v", err)
	}
	assertNoNilSlices(t, "Anlagegut", detail)
}

// Das Kontoblatt nennt je Zeile die Gegenkonten. Auch die Buchung, die für ein
// Konto keines hat, gehört als leere Liste in die Ausgabe.
func TestAccountLedgerCounterAccountsAreEmptyNotNull(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	if _, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 25_000)); err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	ledger, err := env.accounting.GetAccountLedger(ctx, "6815")
	if err != nil {
		t.Fatalf("Kontoblatt: %v", err)
	}
	if len(ledger.Rows) == 0 {
		t.Fatal("das Kontoblatt zeigt die gebuchte Zeile nicht")
	}
	assertNoNilSlices(t, "Kontoblatt", ledger)
}
