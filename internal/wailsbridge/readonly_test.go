package wailsbridge

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/currency"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Prüfermodus muss an der Bridge greifen und nicht in der Oberfläche.
//
// Die Bridge ist über die Wails-Laufzeit auch ohne die Oberfläche erreichbar;
// eine Sperre, die nur Knöpfe ausblendet, sperrt nichts. Geprüft wird deshalb
// zweierlei: dass eine schreibende Methode tatsächlich abweist, und dass keine
// exportierte Methode ohne Einordnung durchrutscht.

// Mit aktivem Prüfermodus weist eine schreibende Methode ab — und ohne ihn
// bucht dieselbe Methode. Die zweite Hälfte gehört dazu: ohne sie bestünde der
// Test auch dann, wenn das Buchen aus einem anderen Grund scheitert.
func TestReadOnlyModeRefusesWritingMethods(t *testing.T) {
	b := testBridge(t)

	if _, err := postManualEntry(b, manualEntryRequest(t, b)); err != nil {
		t.Fatalf("ohne Prüfermodus muss die Buchung durchgehen: %v", err)
	}
	refused := manualEntryRequest(t, b)

	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2022 bis 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}

	_, err := postManualEntry(b, refused)
	if err == nil {
		t.Fatal("im Prüfermodus darf keine Buchung entstehen")
	}
	if !strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}
	if !strings.Contains(err.Error(), "Betriebsprüfung 2022 bis 2024") {
		t.Errorf("die Meldung nennt den erfassten Grund nicht: %v", err)
	}

	if err := b.UpdateCompanySettings(domain.CompanySettings{CompanyName: "Andere GmbH"}); err == nil {
		t.Error("im Prüfermodus dürfen auch die Stammdaten nicht geändert werden")
	}

	// Lesen bleibt möglich — sonst wäre der Modus für einen Prüfer wertlos.
	if _, err := b.GetJournalEntries(); err != nil {
		t.Errorf("im Prüfermodus muss gelesen werden können: %v", err)
	}
	if _, err := b.VerifyIntegrity(); err != nil {
		t.Errorf("im Prüfermodus muss die Integrität geprüft werden können: %v", err)
	}
}

// Ein- und Ausschalten brauchen Datum und Grund und stehen im
// Änderungsprotokoll. Ohne den Eintrag ließe sich später nicht sagen, wer den
// Bestand wann eingefroren hat.
func TestReadOnlyModeRequiresDateAndReasonAndIsLogged(t *testing.T) {
	b := testBridge(t)
	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	if _, err := b.EnableReadOnly("", "Grund"); err == nil {
		t.Error("ohne Datum darf der Prüfermodus nicht eingeschaltet werden")
	}
	if _, err := b.EnableReadOnly("30.06.2026", "Grund"); err == nil {
		t.Error("ein unlesbares Datum muss abgewiesen werden")
	}
	if _, err := b.EnableReadOnly(time.Now().AddDate(0, 0, -1).Format("2006-01-02"), "Grund"); err == nil {
		t.Error("ein Ende in der Vergangenheit muss abgewiesen werden")
	}
	if _, err := b.EnableReadOnly(until, "   "); err == nil {
		t.Error("ohne Grund darf der Prüfermodus nicht eingeschaltet werden")
	}

	cfg, err := b.EnableReadOnly(until, "Betriebsprüfung")
	if err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}
	if !cfg.ReadOnly || cfg.ReadOnlyUntil != until || cfg.ReadOnlyReason != "Betriebsprüfung" {
		t.Errorf("die Konfiguration gibt den Zustand nicht wieder: %+v", cfg)
	}
	if got := b.GetAppConfig(); !got.ReadOnly {
		t.Error("GetAppConfig meldet den Prüfermodus nicht")
	}

	if _, err := b.DisableReadOnly(""); err == nil {
		t.Error("ohne Grund darf der Prüfermodus nicht beendet werden")
	}
	cfg, err = b.DisableReadOnly("Prüfung abgeschlossen")
	if err != nil {
		t.Fatalf("Prüfermodus beenden: %v", err)
	}
	if cfg.ReadOnly {
		t.Error("nach dem Beenden darf der Modus nicht mehr gelten")
	}

	logs := auditEntries(t, b, domain.AuditFilter{})
	var on, off bool
	for _, entry := range logs {
		if entry.EntityType != "READ_ONLY" {
			continue
		}
		if strings.Contains(entry.Details, "eingeschaltet") {
			on = true
		}
		if strings.Contains(entry.Details, "beendet") {
			off = true
		}
	}
	if !on || !off {
		t.Errorf("Start (%v) und Ende (%v) des Prüfermodus stehen nicht im Protokoll", on, off)
	}
}

// Jede exportierte Methode der Bridge ist eingeordnet: entweder steht sie in
// der Leseliste, oder sie schreibt über write/change und damit über
// Tenant.Write, das den Prüfermodus prüft.
//
// Der Test liest den Quelltext, weil Go keine Möglichkeit bietet, zur Laufzeit
// festzustellen, ob eine Methode eine andere aufruft. Ohne ihn fiele eine neu
// hinzugefügte schreibende Methode niemandem auf — außer dem Prüfer, dem sie
// unter der Hand die Bücher ändert.
func TestEveryBridgeMethodIsClassified(t *testing.T) {
	methods := bridgeMethods(t)
	if len(methods) < 100 {
		t.Fatalf("nur %d Bridge-Methoden gefunden — der Test liest den Quelltext offenbar nicht mehr richtig",
			len(methods))
	}

	for name, m := range methods {
		switch {
		case m.guarded && readOnlyAllowed[name]:
			t.Errorf("%s.%s steht in der Leseliste und schreibt trotzdem über write/change — eines von beidem ist falsch",
				m.file, name)
		case !m.guarded && !readOnlyAllowed[name]:
			t.Errorf("%s.%s ist nicht eingeordnet: entweder gehört die Methode in readOnlyAllowed, oder sie schreibt über write/change",
				m.file, name)
		}
	}

	// Und umgekehrt: kein Eintrag in der Leseliste, den es nicht gibt. Ein
	// solcher Eintrag hielte einen Tippfehler am Leben.
	for name := range readOnlyAllowed {
		if _, ok := methods[name]; !ok {
			t.Errorf("die Leseliste nennt %s — eine Methode dieses Namens gibt es nicht", name)
		}
	}
}

// Übermittlungsbestätigung, Aufstellen/Feststellen/Offenlegen und die
// Stammdaten des Unternehmens sind der App vorbehalten. Der Kern weist sie für
// jeden anderen Erfassungsweg ab, wenn die Bridge sie als solche kennzeichnet;
// die Dienste tun es zusätzlich selbst. Die Festschreibung ist ein Vorgang des
// Kerns und prüft den Erfassungsweg dort (core.Tenant.CommitPeriod).
func TestAppOnlyOperationsAreMarked(t *testing.T) {
	methods := bridgeMethods(t)
	for _, name := range []string{
		"ConfirmVatReturnSubmitted", "ConfirmZMSubmitted",
		"SetFiscalYearStatus", "ReopenFiscalYear", "UpdateCompanySettings",
		"ConfigureBankAccounts", "SetInvoiceBankAccount", "BookAppropriation",
		"RegisterShareholderLoan", "CorrectShareholderLoan",
	} {
		m, ok := methods[name]
		if !ok {
			t.Errorf("die Methode %s gibt es nicht mehr", name)
			continue
		}
		if !m.appOnly {
			t.Errorf("%s.%s schreibt ohne core.AppOnly()", m.file, name)
		}
	}
}

// bridgeMethod ist eine exportierte Methode der Bridge mit der Auskunft, ob
// sie im Prüfermodus abgewiesen wird und ob sie der App vorbehalten ist.
type bridgeMethod struct {
	file    string
	guarded bool
	appOnly bool
}

// bridgeMethods liest den Quelltext des Pakets als Syntaxbaum.
//
// Als Baum und nicht als Text: eine Textsuche fände den Aufruf auch in einem
// Kommentar, und ein „// TODO write(…)" über einer schreibenden Methode
// genügte dann für die Einordnung.
func bridgeMethods(t *testing.T) map[string]bridgeMethod {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("Paketordner lesen: %v", err)
	}

	out := map[string]bridgeMethod{}
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name == nil || !fn.Name.IsExported() {
					continue
				}
				if !isBridgeReceiver(fn.Recv) {
					continue
				}
				guarded, appOnly := writeCalls(fn)
				out[fn.Name.Name] = bridgeMethod{file: filepath.Base(name), guarded: guarded, appOnly: appOnly}
			}
		}
	}
	return out
}

// isBridgeReceiver meldet, ob der Empfänger *BuchfinkBridge ist.
func isBridgeReceiver(recv *ast.FieldList) bool {
	if recv == nil || len(recv.List) != 1 {
		return false
	}
	star, ok := recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "BuchfinkBridge"
}

// writeCalls sucht die Aufrufe von write, change, prepared und operate im
// Rumpf. guarded ist ein Aufruf ohne core.ExaminerExempt(), appOnly einer mit
// core.AppOnly(). operate ruft einen Vorgang des Kerns, der selbst über
// Tenant.Write schreibt.
func writeCalls(fn *ast.FuncDecl) (guarded, appOnly bool) {
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isWriteHelper(call.Fun) {
			return true
		}
		exempt := false
		for _, arg := range call.Args {
			switch optionName(arg) {
			case "ExaminerExempt":
				exempt = true
			case "AppOnly":
				appOnly = true
			}
		}
		if !exempt {
			guarded = true
		}
		return true
	})
	return guarded, appOnly
}

func isWriteHelper(fun ast.Expr) bool {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name == "write" || f.Name == "operate" || f.Name == "prepared"
	case *ast.IndexExpr:
		return isWriteHelper(f.X)
	case *ast.SelectorExpr:
		return f.Sel.Name == "change"
	}
	return false
}

// optionName ist der Name einer Option der Form core.Name().
func optionName(arg ast.Expr) string {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "core" {
		return ""
	}
	return sel.Sel.Name
}

// Ein Kommentar ist kein Aufruf. Ohne diese Probe bestünde der Test oben auch
// dann, wenn jemand den Schreibweg auskommentiert und den Hinweis stehen lässt.
func TestClassificationIgnoresCommentedOutGuards(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "beispiel.go", `package wailsbridge

func (b *BuchfinkBridge) Beispiel() error {
	// TODO: das gehört in b.change(…)
	return nil
}

func (b *BuchfinkBridge) Ausgenommen() error {
	return b.change(nil, core.ExaminerExempt(), core.AppOnly())
}
`, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("Beispiel übersetzen: %v", err)
	}
	if guarded, _ := writeCalls(file.Decls[0].(*ast.FuncDecl)); guarded {
		t.Error("ein auskommentierter Aufruf gilt als Schreibweg — damit ließe sich jede Sperre umgehen")
	}
	guarded, appOnly := writeCalls(file.Decls[1].(*ast.FuncDecl))
	if guarded || !appOnly {
		t.Errorf("ausgenommener Schreibweg falsch gelesen: geprüft %v, nur App %v", guarded, appOnly)
	}
}

// GetAppConfig ist die eine Quelle, aus der die Oberfläche den Zustand liest.
func TestGetAppConfigReportsTheCurrentState(t *testing.T) {
	b := testBridge(t)
	if got := b.GetAppConfig(); got.ProgramVersion == "" || got.ReadOnly {
		t.Errorf("Konfiguration ohne Programmfassung oder mit Prüfermodus: %+v", got)
	}
	until := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung"); err != nil {
		t.Fatal(err)
	}
	if cfg := b.GetAppConfig(); !cfg.ReadOnly || cfg.ReadOnlyReason != "Betriebsprüfung" {
		t.Errorf("GetAppConfig meldet den laufenden Prüfermodus nicht: %+v", cfg)
	}
}

// Der geprüfte Mandant darf aus der Liste: der Prüfermodus steht in seiner
// Datenbank und gilt wieder, sobald der Ordner wieder geöffnet wird.
func TestDeleteTenantKeepsTheExaminerModeInTheData(t *testing.T) {
	b := testBridge(t)
	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2022 bis 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}
	cfg := b.GetAppConfig()
	audited := cfg.ActiveTenant()
	if err := b.DeleteTenant(audited.ID); err != nil {
		t.Fatalf("Mandant aus der Liste nehmen: %v", err)
	}
	if b.GetAppConfig().ReadOnly {
		t.Error("ohne offenen Mandanten meldet die Konfiguration einen Prüfermodus")
	}
	if err := b.LoadExistingDatabase(filepath.Join(audited.DataDir, "buchfink.sqlite")); err != nil {
		t.Fatalf("wieder öffnen: %v", err)
	}
	if got := b.GetAppConfig(); !got.ReadOnly || got.ReadOnlyUntil != until {
		t.Errorf("nach dem Wiederöffnen fehlt der Prüfermodus: %+v", got)
	}
	if err := b.UpdateCompanySettings(domain.CompanySettings{CompanyName: "Andere GmbH"}); err == nil {
		t.Error("der wieder geöffnete Mandant nimmt im Prüfermodus Änderungen an")
	}
}

// Der Prüfermodus sperrt die Vorschau der Stichtagsbewertung nicht.
//
// Sie ist eine Auswertung, und der Prüfermodus ist der Modus, in dem
// ausgewertet wird. Weil sie einen fehlenden Kurs sonst holt und ablegt, stand
// sie eine Zeit lang hinter ensureWritable — und ein Prüfer konnte die
// Bewertung eines abgeschlossenen Jahres nicht einmal ansehen, obwohl alle
// Kurse dazu längst in der Historie stehen. Jetzt antwortet dort der nur
// lesende Kursdienst: er nimmt, was gespeichert ist, und holt nichts nach.
func TestReadOnlyModeAnswersTheExchangeRateFromHistory(t *testing.T) {
	// Ein Kursdienst, dessen Adresse ins Leere zeigt: würde geholt, käme ein
	// Fehler — und genau das soll im Prüfermodus nicht geschehen.
	b := testBridge(t, func(o *core.HostOptions) { o.Network.Rates = currency.New("http://127.0.0.1:1") })
	testWrite(t, b, func(ctx context.Context, s *core.Services) error {
		_, err := s.Currency.SaveRate(ctx, domain.ExchangeRate{
			Currency: "USD", Date: "2026-12-31", RateMicros: 1_085_000,
			Source: "EZB-Referenzkurs",
		})
		return err
	})

	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2022 bis 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}

	rate, err := b.GetExchangeRate("USD", "2026-12-31")
	if err != nil {
		t.Fatalf("im Prüfermodus muss der gespeicherte Kurs lesbar sein: %v", err)
	}
	if rate.RateMicros != 1_085_000 {
		t.Errorf("Kurs %d Millionstel — erwartet den gespeicherten", rate.RateMicros)
	}

	// Und wo keiner gespeichert ist, sagt der Dienst genau das, statt still
	// einen zu holen und abzulegen.
	if _, err := b.GetExchangeRate("CHF", "2026-12-31"); err == nil {
		t.Error("ohne gespeicherten Kurs darf im Prüfermodus keiner geholt werden")
	} else if !strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("die Meldung muss den Grund nennen: %v", err)
	}
}

// Ein Lesevorgang darf einen geholten Kurs ablegen: er ist kein
// Buchungsinhalt und muss beim nächsten Mal derselbe sein.
func TestReadingAnExchangeRateKeepsTheFetchedRate(t *testing.T) {
	b := testBridge(t, func(o *core.HostOptions) { o.Network.Rates = fixedRate{micros: 1_100_000} })
	rate, err := b.GetExchangeRate("USD", "2026-06-30")
	if err != nil {
		t.Fatalf("Kurs holen: %v", err)
	}
	if rate.RateMicros != 1_100_000 {
		t.Errorf("Kurs %d, erwartet den geholten", rate.RateMicros)
	}
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		stored, err := s.Repos.ExchangeRates.FindRate(ctx, "USD", "2026-06-30")
		if err == nil && stored == nil {
			t.Error("der geholte Kurs wurde nicht abgelegt")
		}
		return err
	})
}

type fixedRate struct{ micros int64 }

func (f fixedRate) RateAt(_ context.Context, _, date string) (int64, string, string, error) {
	return f.micros, date, "Testkurs", nil
}
