// Prüft, dass jede exportierte Methode eines Dienstes in internal/service einen
// Aufrufer außerhalb der Tests hat.
//
// Ein Dienst ohne Aufrufer ist eine Fähigkeit, die niemand erreicht: weder die
// App über die Bridge noch ein Agent über MCP, Kern oder Policy noch ein
// anderer Dienst. Sie veraltet unbemerkt, weil nur ihre Tests sie ausführen,
// und täuscht im Code einen Weg vor, den es nicht gibt.
//
// Dienste sind die exportierten Typen mit der Endung „Service“; gezählt wird
// ihre Methodenmenge samt eingebetteter Typen aus internal/service. Aufgerufen
// ist eine Methode, wenn eine Datei des Moduls außer den Tests sie auswählt,
// unmittelbar oder über eine Schnittstelle, die der Dienst erfüllt. Die
// Typprüfung läuft über go/types: Namen allein trennten `Get` des einen
// Dienstes nicht von `Get` des anderen.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const servicePackage = "/internal/service"

// Gründe der Ausnahmen, die für mehrere Methoden gelten.
const (
	testHook  = "Prüfzugang für die Verdrahtungstests in internal/core; Tests anderer Pakete sehen nur Exportiertes"
	testSetup = "Setter für einen begrenzten Testaufbau; die Verdrahtung nimmt SetPeriods"
)

// exceptions sind Methoden ohne Aufrufer außerhalb der Tests, die bleiben, mit
// ihrem Grund. Außer den Prüfzugängen ist es Fachlogik mit Tests, die weder
// die App noch ein Werkzeug erreicht; der Grund nennt, was deshalb fehlt.
var exceptions = map[string]string{
	"JournalService.EntryEffectNames":         testHook,
	"JournalService.TaxAccountExceptionNames": testHook,
	"PeriodService.PreCommitCheckNames":       testHook,
	"JournalService.SetFestschreibungRepo":    testSetup,
	"JournalService.SetFiscalYearRepo":        testSetup,

	"AccountService.CreateCustom":       "eigene Konten im freien Bereich des SKR04 lassen sich nicht anlegen",
	"AccountService.AvailablePositions": "Bilanzposten für ein eigenes Konto; gehört zu CreateCustom",
	"AccountService.SetBlocked":         "eigene Konten lassen sich nicht sperren",

	"AccrualService.Propose": "keine Vorschläge zur Rechnungsabgrenzung; der Agent sucht die Buchungen selbst",
	"AccrualService.Report":  "kein Abgrenzungsspiegel zum Stichtag",

	"AppropriationService.SaveNotesText": "Anhangtexte lassen sich nicht erfassen; mit leerem Pflichtabschnitt scheitert das Aufstellen",

	"AssetService.BookImpairment":            "keine außerplanmäßige Abschreibung",
	"AssetService.BookWriteUp":               "keine Wertaufholung nach § 253 Abs. 5 HGB",
	"AssetService.ConfirmImpairmentPersists": "die Beibehaltung einer außerplanmäßigen Abschreibung lässt sich nicht bestätigen",
	"AssetService.Transfer":                  "eine Anlage im Bau lässt sich nicht fertigstellen",
	"AssetService.BookCurrencyValuation":     "keine Stichtagsbewertung eines Anlageguts in Fremdwährung",
	"AssetService.PreviewDisposal":           "Vorschau zu Dispose",
	"AssetService.Dispose":                   "kein Abgang: ein verkauftes Anlagegut bleibt in der Kartei",

	"ClosingService.SetPriorYearRevenue": "der vorbelegte Vorjahresumsatz (§ 27 Abs. 38 UStG) lässt sich nicht berichtigen",

	"CurrencyService.SaveRate": "kein Kurs von Hand mit Quelle, obwohl die Meldung bei fehlendem Kurs darauf verweist",
	"CurrencyService.Rates":    "keine Liste der gespeicherten Tageskurse; gehört zu SaveRate",

	"ExportService.ExportJournalCSV": "kein Journal eines Zeitraums als CSV; die Datenüberlassung enthält ganze Jahre",

	"FoundationService.BookPostings":  "Zeichnung aus den Stammdaten mit Schutz vor doppelter Buchung; der Agent bucht sie mit buchung_erfassen ohne diesen Schutz",
	"FoundationService.CallInCapital": "Einforderung mit Prüfung gegen den nicht eingeforderten Teil; der Agent bucht sie mit buchung_erfassen ohne diese Prüfung",

	"GiftService.RebookGiftsForRecipient": "Geschenke über der Freigrenze lassen sich nicht umbuchen, obwohl der Bericht sie nennt",

	"InputTaxService.SaveUsage": "kein Verwendungsanteil nach § 15a UStG; der Abschlussschritt verweist auf die App, die ihn nur zeigt",
	"InputTaxService.BookYear":  "keine Vorsteuerberichtigung nach § 15a UStG; gehört zu SaveUsage",
	"InputTaxService.Close":     "ein Eintrag nach § 15a UStG lässt sich nicht vorzeitig schließen",

	"InvoiceService.CorrectInvoice":   "keine Stornorechnung und keine Berichtigung einer Ausgangsrechnung",
	"InvoiceService.MarkSent":         "kein Versandvermerk einer Ausgangsrechnung",
	"InvoiceService.RefundAdvance":    "keine Rückzahlung einer erhaltenen Anzahlung",
	"InvoiceService.OpenAdvanceItems": "offene Abschlagsrechnungen fehlen in offene_posten und in den Bankvorschlägen",
	"InvoiceService.GetInvoiceGroups": "kein Stand eines Rechnungsverbunds",

	"JournalService.BookOpeningBalance":    "keine Eröffnungsbuchungen eines Umsteigers",
	"JournalService.PreviewOpeningBalance": "Vorschau zu BookOpeningBalance",

	"PaymentService.WriteOffOpenItem": "kein Forderungsausfall mit Berichtigung nach § 17 UStG",

	"PostingService.OpenVendorAdvances": "die Kennungen geleisteter Anzahlungen, die beleg_buchen_mit_rueckfrage verrechnet, liefert kein Werkzeug",

	"ProvisionService.BookIncrease":           "keine Zuführung zu einer gebildeten Rückstellung",
	"ProvisionService.BookUnwinding":          "keine Aufzinsung einer Rückstellung",
	"ProvisionService.Settle":                 "keine Auflösung einer Rückstellung",
	"ProvisionService.ImportDiscountRatesCSV": "keine Abzinsungssätze der Bundesbank; Rückstellungen bleiben unabgezinst",
	"ProvisionService.DiscountRates":          "gehört zu ImportDiscountRatesCSV",
	"ProvisionService.DiscountRateMonths":     "gehört zu ImportDiscountRatesCSV",

	"ReceiptService.RemoveFile":        "eine Datei lässt sich von einem offenen Beleg nicht entfernen",
	"ReceiptService.OverrideRetention": "die Aufbewahrungsfrist eines Belegs lässt sich nicht verlängern",

	"ShareholderService.Balances": "keine Salden je Gesellschafter",

	"VatReturnService.SuggestedSpecialPrepayment": "kein Vorschlag zur Sondervorauszahlung",
	"VatIDService.Checks":                         "kein Verlauf der Bestätigungsanfragen einer USt-IdNr.",
	"ZMService.CreateCorrection":                  "keine berichtigte ZM, obwohl das Speichern einer übermittelten Meldung darauf verweist",
	"ZMService.ExportCSV":                         "keine ZM als CSV für das BZSt-Online-Portal",
}

type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	CgoFiles   []string
	Export     string
}

type loader struct {
	module   string
	fset     *token.FileSet
	listed   map[string]*listedPackage
	checked  map[string]*types.Package
	compiled types.Importer
	// selections sind alle Methodenauswahlen außerhalb der Tests.
	selections []*types.Selection
}

func main() {
	root, err := moduleRoot()
	if err != nil {
		fail(err)
	}
	l, err := newLoader(root)
	if err != nil {
		fail(err)
	}
	for path := range l.listed {
		if strings.HasPrefix(path, l.module) && !strings.HasPrefix(path, l.module+"/scripts/") {
			if _, err := l.check(path); err != nil {
				fail(err)
			}
		}
	}
	excepted := map[string]bool{}
	findings, count := l.unused(excepted)
	stale := staleExceptions(l, excepted)
	for _, f := range findings {
		fmt.Println(f)
	}
	for _, s := range stale {
		fmt.Println(s)
	}
	if len(findings) > 0 || len(stale) > 0 {
		fmt.Printf("\nDienste ohne Aufrufer: %d Befunde. Anschließen, löschen oder mit Grund in die Ausnahmen von %s.\n",
			len(findings)+len(stale), filepath.Join("scripts", "check_service_callers", "main.go"))
		os.Exit(1)
	}
	fmt.Printf("Dienste ohne Aufrufer: %d Methoden geprüft, jede hat einen Aufrufer außerhalb der Tests; %d Ausnahmen.\n",
		count, len(exceptions))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "check_service_callers:", err)
	os.Exit(2)
}

func moduleRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	mod := strings.TrimSpace(string(out))
	if mod == "" || mod == os.DevNull {
		return "", errors.New("kein Go-Modul gefunden")
	}
	return filepath.Dir(mod), nil
}

// newLoader liest Pakete und Exportdaten über `go list`. Pakete außerhalb des
// Moduls kommen aus den Exportdaten des Compilers; scheitert eines am Bau, etwa
// Wails ohne GTK, bleiben seine Typen unbekannt, und die Prüfung läuft weiter.
func newLoader(root string) (*loader, error) {
	cmd := exec.Command("go", "list", "-e", "-export", "-deps",
		"-json=ImportPath,Dir,GoFiles,CgoFiles,Export,Module", "./...")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// Mit -e meldet go list Pakete, die nicht bauen, im Ergebnis und endet
	// trotzdem mit einem Fehler; ohne Ausgabe ist nichts zu prüfen.
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("go list: %w\n%s", err, stderr.String())
	}
	l := &loader{
		fset:    token.NewFileSet(),
		listed:  map[string]*listedPackage{},
		checked: map[string]*types.Package{},
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p struct {
			listedPackage
			Module *struct {
				Path string
				Main bool
			}
		}
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("go list lesen: %w", err)
		}
		pkg := p.listedPackage
		l.listed[pkg.ImportPath] = &pkg
		if p.Module != nil && p.Module.Main {
			l.module = p.Module.Path
		}
	}
	if l.module == "" {
		return nil, errors.New("go list nennt kein Hauptmodul")
	}
	l.compiled = importer.ForCompiler(l.fset, "gc", func(path string) (io.ReadCloser, error) {
		p, ok := l.listed[path]
		if !ok || p.Export == "" {
			return nil, fmt.Errorf("keine Exportdaten für %s", path)
		}
		return os.Open(p.Export)
	})
	return l, nil
}

func (l *loader) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if strings.HasPrefix(path, l.module) {
		return l.check(path)
	}
	return l.compiled.Import(path)
}

// check prüft ein Paket des Moduls aus dem Quelltext, ohne Tests. Alle Pakete
// des Moduls teilen so dieselben Typobjekte.
func (l *loader) check(path string) (*types.Package, error) {
	if pkg, ok := l.checked[path]; ok {
		return pkg, nil
	}
	listed, ok := l.listed[path]
	if !ok {
		return nil, fmt.Errorf("Paket %s fehlt in go list", path)
	}
	var files []*ast.File
	for _, name := range append(append([]string{}, listed.GoFiles...), listed.CgoFiles...) {
		file, err := parser.ParseFile(l.fset, filepath.Join(listed.Dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	info := &types.Info{Selections: map[*ast.SelectorExpr]*types.Selection{}}
	conf := types.Config{Importer: l, FakeImportC: true, Error: func(error) {}}
	pkg, _ := conf.Check(path, l.fset, files, info)
	l.checked[path] = pkg
	for _, sel := range info.Selections {
		if sel.Kind() != types.FieldVal {
			l.selections = append(l.selections, sel)
		}
	}
	return pkg, nil
}

// services liefert die Dienste: exportierte Typen mit der Endung „Service“.
func (l *loader) services() []*types.Named {
	pkg := l.checked[l.module+servicePackage]
	if pkg == nil {
		fail(fmt.Errorf("Paket %s nicht geprüft", l.module+servicePackage))
	}
	var out []*types.Named
	for _, name := range pkg.Scope().Names() {
		tn, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok || !tn.Exported() || !strings.HasSuffix(name, "Service") {
			continue
		}
		if named, ok := tn.Type().(*types.Named); ok {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct {
				out = append(out, named)
			}
		}
	}
	return out
}

// methodKey ist der Name einer Methode in Meldungen und Ausnahmen:
// Typ.Methode des deklarierenden Typs.
func methodKey(fn *types.Func) string {
	recv := fn.Type().(*types.Signature).Recv().Type()
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}
	if named, ok := recv.(*types.Named); ok {
		return named.Obj().Name() + "." + fn.Name()
	}
	return fn.Name()
}

// unused nennt jede Dienstmethode ohne Aufrufer, die keine Ausnahme ist, und
// merkt sich die Ausnahmen ohne Aufrufer in unused.
func (l *loader) unused(unused map[string]bool) ([]string, int) {
	used := map[*types.Func]bool{}
	type ifaceCall struct {
		iface *types.Interface
		name  string
	}
	var viaInterface []ifaceCall
	for _, sel := range l.selections {
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			continue
		}
		if iface, ok := sel.Recv().Underlying().(*types.Interface); ok {
			viaInterface = append(viaInterface, ifaceCall{iface, fn.Name()})
			continue
		}
		used[fn.Origin()] = true
	}

	servicePath := l.module + servicePackage
	methods := map[*types.Func][]*types.Named{}
	for _, svc := range l.services() {
		set := types.NewMethodSet(types.NewPointer(svc))
		for i := 0; i < set.Len(); i++ {
			fn, ok := set.At(i).Obj().(*types.Func)
			if !ok || !fn.Exported() || fn.Pkg() == nil || fn.Pkg().Path() != servicePath {
				continue
			}
			fn = fn.Origin()
			methods[fn] = append(methods[fn], svc)
		}
	}

	var findings []string
	for fn, owners := range methods {
		if used[fn] {
			continue
		}
		implemented := false
		for _, call := range viaInterface {
			if call.name != fn.Name() {
				continue
			}
			for _, svc := range owners {
				if types.Implements(types.NewPointer(svc), call.iface) {
					implemented = true
				}
			}
		}
		if implemented {
			continue
		}
		key := methodKey(fn)
		if _, ok := exceptions[key]; ok {
			unused[key] = true
			continue
		}
		pos := l.fset.Position(fn.Pos())
		rel, err := filepath.Rel(filepath.Dir(l.listed[servicePath].Dir), pos.Filename)
		if err != nil {
			rel = pos.Filename
		}
		findings = append(findings, fmt.Sprintf("internal/%s:%d: %s hat keinen Aufrufer außerhalb der Tests",
			filepath.ToSlash(rel), pos.Line, key))
	}
	sort.Strings(findings)
	return findings, len(methods)
}

// staleExceptions meldet Ausnahmen für Methoden, die es nicht mehr gibt oder die
// inzwischen einen Aufrufer haben.
func staleExceptions(l *loader, unused map[string]bool) []string {
	known := map[string]bool{}
	for _, svc := range l.services() {
		set := types.NewMethodSet(types.NewPointer(svc))
		for i := 0; i < set.Len(); i++ {
			if fn, ok := set.At(i).Obj().(*types.Func); ok {
				known[methodKey(fn.Origin())] = true
			}
		}
	}
	var out []string
	for key := range exceptions {
		switch {
		case !known[key]:
			out = append(out, fmt.Sprintf("Ausnahme %s: die Methode gibt es nicht mehr", key))
		case !unused[key]:
			out = append(out, fmt.Sprintf("Ausnahme %s: die Methode hat einen Aufrufer; den Eintrag entfernen", key))
		}
	}
	sort.Strings(out)
	return out
}
