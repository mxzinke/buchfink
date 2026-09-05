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

	"github.com/buchfink/buchfink/internal/currency"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// Der Prüfermodus muss an der Bridge greifen und nicht in der Oberfläche.
//
// Die Bridge ist über die Wails-Laufzeit auch ohne die Oberfläche erreichbar;
// eine Sperre, die nur Knöpfe ausblendet, sperrt nichts. Geprüft wird deshalb
// zweierlei: dass eine schreibende Methode tatsächlich abweist, und dass keine
// exportierte Methode ohne Einordnung durchrutscht.

// testBridge baut eine Bridge auf einem Wegwerf-Datenordner, ohne die
// Wails-Anwendung zu starten.
func testBridge(t *testing.T) *BuchfinkBridge {
	t.Helper()
	dataDir := t.TempDir()

	db, err := repository.InitTenantDB(dataDir)
	if err != nil {
		t.Fatalf("Mandantendatenbank: %v", err)
	}
	tenant := domain.TenantConfig{
		ID: "tenant_test", Name: "Pfennig Ventures GmbH", DataDir: dataDir,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	b := &BuchfinkBridge{
		appCfgRepo: repository.NewAppConfigRepository(filepath.Join(t.TempDir(), "config")),
		appConfig: domain.AppConfig{
			Tenants: []domain.TenantConfig{tenant}, ActiveTenantID: tenant.ID,
			DataDir: dataDir, IsConfigured: true, LastFiscalYear: 2026,
		},
		dataDir:     dataDir,
		currentYear: 2026,
		db:          db,
	}
	b.journalRepo = repository.NewJournalRepository(db)
	b.accountRepo = repository.NewAccountRepository(db)
	b.contactRepo = repository.NewContactRepository(db)
	b.auditRepo = repository.NewAuditRepository(db)
	b.settingsRepo = repository.NewSettingsRepository(db)
	b.journalSvc = service.NewJournalService(
		b.journalRepo, b.accountRepo, b.contactRepo, b.auditRepo, b.settingsRepo, 2026)
	return b
}

func manualEntry() domain.JournalEntry {
	return domain.JournalEntry{
		BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		Description: "Testbuchung", Source: domain.EntrySourceManual,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
}

// Mit aktivem Prüfermodus weist eine schreibende Methode ab — und ohne ihn
// bucht dieselbe Methode. Die zweite Hälfte gehört dazu: ohne sie bestünde der
// Test auch dann, wenn das Buchen aus einem anderen Grund scheitert.
func TestReadOnlyModeRefusesWritingMethods(t *testing.T) {
	b := testBridge(t)

	if _, err := b.PostJournalEntry(manualEntry()); err != nil {
		t.Fatalf("ohne Prüfermodus muss die Buchung durchgehen: %v", err)
	}

	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2022 bis 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}

	_, err := b.PostJournalEntry(manualEntry())
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

// Der Modus endet von selbst. Ein Datum in der Vergangenheit sperrt nichts mehr
// — sonst müsste jemand daran denken, ihn auszuschalten, und niemand tut das.
func TestReadOnlyModeExpires(t *testing.T) {
	b := testBridge(t)

	tenant := b.activeTenantLocked()
	tenant.ReadOnlyUntil = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	tenant.ReadOnlyReason = "abgelaufene Prüfung"

	if _, err := b.PostJournalEntry(manualEntry()); err != nil {
		t.Errorf("nach dem Ablaufdatum muss wieder gebucht werden können: %v", err)
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

	logs, err := b.auditRepo.FindAll(context.Background(), 0)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
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
// der Leseliste oder sie ruft ensureWritable.
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
			t.Errorf("%s.%s steht in der Leseliste und ruft trotzdem ensureWritable — eines von beidem ist falsch",
				m.file, name)
		case !m.guarded && !readOnlyAllowed[name]:
			t.Errorf("%s.%s ist nicht eingeordnet: entweder gehört die Methode in readOnlyAllowed, oder sie muss ensureWritable rufen",
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

// bridgeMethod ist eine exportierte Methode der Bridge mit der Auskunft, ob sie
// ensureWritable ruft.
type bridgeMethod struct {
	file    string
	guarded bool
}

// bridgeMethods liest den Quelltext des Pakets als Syntaxbaum.
//
// Als Baum und nicht als Text: eine Textsuche nach „b.ensureWritable()" fände
// den Aufruf auch in einem Kommentar. Ein „// TODO b.ensureWritable()" über
// einer schreibenden Methode genügte dann für die Einordnung, ohne dass je eine
// Prüfung liefe — und der Test, der das verhindern soll, hielte sie für
// erledigt. Der Syntaxbaum kennt keine Kommentare.
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
				out[fn.Name.Name] = bridgeMethod{
					file:    filepath.Base(name),
					guarded: callsEnsureWritable(fn),
				}
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

// callsEnsureWritable sucht den tatsächlichen Aufruf im Rumpf.
func callsEnsureWritable(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "ensureWritable" {
			return true
		}
		found = true
		return false
	})
	return found
}

// Ein Kommentar ist kein Aufruf. Ohne diese Probe bestünde der Test oben auch
// dann, wenn jemand die Prüfung auskommentiert und den Hinweis stehen lässt.
func TestClassificationIgnoresCommentedOutGuards(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "beispiel.go", `package wailsbridge

func (b *BuchfinkBridge) Beispiel() error {
	// TODO: hier fehlt noch b.ensureWritable()
	return nil
}
`, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("Beispiel übersetzen: %v", err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if callsEnsureWritable(fn) {
		t.Error("ein auskommentierter Aufruf gilt als Prüfung — damit ließe sich jede Sperre umgehen")
	}
}

// GetAppConfig ist die eine Quelle, aus der die Oberfläche den Zustand liest.
// Sie muss ihn zum heutigen Tag melden und nicht so, wie er beim letzten
// Speichern war — sonst zeigt die Oberfläche ein Banner und blendet Knöpfe aus,
// während die Bridge längst wieder schreiben lässt.
func TestGetAppConfigReportsTheCurrentState(t *testing.T) {
	b := testBridge(t)

	if got := b.GetAppConfig(); got.ProgramVersion == "" {
		t.Error("GetAppConfig nennt die Programmversion nicht")
	}

	// Ein Prüfermodus, der gestern abgelaufen ist: die Bridge lässt wieder
	// buchen, also darf die Konfiguration ihn nicht mehr melden.
	tenant := b.activeTenantLocked()
	tenant.ReadOnlyUntil = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	tenant.ReadOnlyReason = "abgelaufene Prüfung"
	b.appConfig.ReadOnly = true

	cfg := b.GetAppConfig()
	if cfg.ReadOnly {
		t.Error("GetAppConfig meldet den Prüfermodus, obwohl er abgelaufen ist")
	}
	if _, err := b.PostJournalEntry(manualEntry()); err != nil {
		t.Errorf("die Bridge lässt nicht buchen, obwohl der Modus abgelaufen ist: %v", err)
	}

	// Und andersherum: ein laufender Modus wird gemeldet, auch wenn seit dem
	// letzten Speichern nichts nachgezogen wurde.
	tenant.ReadOnlyUntil = time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	b.appConfig.ReadOnly = false
	if cfg := b.GetAppConfig(); !cfg.ReadOnly || cfg.ReadOnlyReason != "abgelaufene Prüfung" {
		t.Errorf("GetAppConfig meldet den laufenden Prüfermodus nicht: %+v", cfg)
	}
}

// Im Prüfermodus darf der geprüfte Mandant nicht gelöscht werden: mit seinem
// Schlüsselbund-Geheimnis wären die eingefrorenen Daten unzugänglich.
func TestDeleteTenantIsRefusedForTheAuditedTenant(t *testing.T) {
	b := testBridge(t)
	until := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := b.EnableReadOnly(until, "Betriebsprüfung 2022 bis 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}

	err := b.DeleteTenant(b.appConfig.ActiveTenantID)
	if err == nil {
		t.Fatal("der geprüfte Mandant wurde im Prüfermodus gelöscht")
	}
	if !strings.Contains(err.Error(), "Prüfermodus") {
		t.Errorf("die Meldung nennt den Grund nicht: %v", err)
	}
	if len(b.appConfig.Tenants) != 1 {
		t.Errorf("der Mandant steht nicht mehr in der Konfiguration: %d Einträge", len(b.appConfig.Tenants))
	}

	// Ein anderer Mandant darf gehen — der Modus friert einen Bestand ein und
	// nicht die Mandantenverwaltung.
	b.appConfig.Tenants = append(b.appConfig.Tenants, domain.TenantConfig{
		ID: "tenant_zweiter", Name: "Zweiter Mandant", DataDir: t.TempDir(),
	})
	if err := b.DeleteTenant("tenant_zweiter"); err != nil {
		t.Errorf("ein anderer Mandant ließ sich im Prüfermodus nicht entfernen: %v", err)
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
	b := testBridge(t)
	ctx := context.Background()
	b.exchangeRateRepo = repository.NewExchangeRateRepository(b.db)
	// Ein Kursdienst, dessen Adresse ins Leere zeigt: würde geholt, käme ein
	// Fehler — und genau das soll im Prüfermodus nicht geschehen.
	b.currencySvc = service.NewCurrencyService(
		b.exchangeRateRepo, currency.New("http://127.0.0.1:1"), b.auditRepo)
	b.currencySvc.SetJournalRepo(b.journalRepo)
	b.currencySvc.SetJournalService(b.journalSvc)

	if err := b.exchangeRateRepo.SaveRate(ctx, &domain.ExchangeRate{
		Currency: "USD", Date: "2026-12-31", RateMicros: 1_085_000,
		Source: "EZB-Referenzkurs",
	}); err != nil {
		t.Fatalf("Kurs anlegen: %v", err)
	}

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
