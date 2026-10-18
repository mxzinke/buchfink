package wailsbridge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/inbox"
)

const testIBAN = "DE89370400440532013000"

const testStatement = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02"><BkToCstmrStmt><Stmt>
<Acct><Id><IBAN>` + testIBAN + `</IBAN></Id></Acct>
<Ntry><Amt Ccy="EUR">10.00</Amt><CdtDbtInd>CRDT</CdtDbtInd><BookgDt><Dt>2026-04-12</Dt></BookgDt><ValDt><Dt>2026-04-12</Dt></ValDt>
<NtryDtls><TxDtls><Refs><EndToEndId>E2E-1</EndToEndId></Refs><RltdPties><Dbtr><Nm>Hausbank</Nm></Dbtr></RltdPties>
<RmtInf><Ustrd>Zinsen</Ustrd></RmtInf></TxDtls></NtryDtls></Ntry></Stmt></BkToCstmrStmt></Document>`

func writeTestFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func tenantDataDir(t *testing.T, b *BuchfinkBridge) string {
	t.Helper()
	var dir string
	testRead(t, b, func(ctx context.Context, s *core.Services) error {
		dir = s.Tenant.DataDir
		return nil
	})
	return dir
}

// Der Eingang in der App: zeigen, hineinziehen, ablegen und aufräumen.
func TestInboxInTheApp(t *testing.T) {
	b := testBridge(t)
	data := tenantDataDir(t, b)
	defaultInbox := filepath.Join(data, policy.InboxDir)

	info, err := b.GetInbox()
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != defaultInbox || !info.Default || info.Count != 0 || info.ReadFolders == nil ||
		info.ExportFolder != filepath.Join(data, policy.ExportDir) {
		t.Fatalf("Voreinstellung: %+v", info)
	}
	if _, err := os.Stat(defaultInbox); err != nil {
		t.Fatalf("der Eingang fehlt: %v", err)
	}

	src := t.TempDir()
	pdf := writeTestFile(t, filepath.Join(src, "rechnung.pdf"), "%PDF-1.4 Rechnung Papier AG")
	txt := writeTestFile(t, filepath.Join(src, "notiz.txt"), "Notiz")
	added, err := b.AddToInbox([]string{pdf, txt})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(added.Names, []string{"rechnung.pdf"}) || len(added.Skipped) != 1 || added.Skipped[0].Name != "notiz.txt" {
		t.Fatalf("hineingezogen: %+v", added)
	}
	list, err := b.ListInbox()
	if err != nil || len(list) != 1 || list[0].Kind != "beleg" || list[0].ReceiptID != nil ||
		list[0].Path != filepath.Join(defaultInbox, "rechnung.pdf") {
		t.Fatalf("Eingang: %+v %v", list, err)
	}
	if info, _ = b.GetInbox(); info.Count != 1 {
		t.Errorf("Zahl der Dateien: %+v", info)
	}

	// Beleg aus Datei: die Datei aus dem Eingang geht danach nach Übernommen/.
	receipt, err := b.FileIncomingReceipt(time.Now().Format("2006-01-02"), "upload",
		[]ReceiptFileInput{{Path: list[0].Path, Role: string(domain.ReceiptRoleOriginal)}})
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(defaultInbox, inbox.DoneDir, time.Now().Format("2006-01"), "rechnung.pdf")
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("nicht nach Übernommen verschoben: %v", err)
	}
	if list, _ = b.ListInbox(); len(list) != 0 {
		t.Errorf("Eingang nach der Ablage: %+v", list)
	}

	// Dieselbe Datei ein zweites Mal hineingezogen: Buchfink kennt sie schon.
	if _, err := b.AddToInbox([]string{pdf}); err != nil {
		t.Fatal(err)
	}
	if list, _ = b.ListInbox(); len(list) != 1 || list[0].ReceiptID == nil || *list[0].ReceiptID != receipt.ID {
		t.Errorf("bereits abgelegt: %+v", list)
	}

	if err := b.OpenInbox(); err == nil {
		t.Error("ohne Anwendung lässt sich kein Dateimanager öffnen")
	}
}

// Den Eingang wählt der Mensch; Datenordner und Sicherungen nimmt die Bridge
// nicht an. Der Ordner wird angelegt.
func TestInboxFolderIsChosenInTheApp(t *testing.T) {
	b := testBridge(t)
	data := tenantDataDir(t, b)
	custom := filepath.Join(t.TempDir(), "Scans", "Eingang")
	info, err := b.SetInboxFolder(custom)
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != custom || info.Default {
		t.Errorf("eigener Eingang: %+v", info)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("nicht angelegt: %v", err)
	}
	for _, refused := range []string{data, filepath.Join(data, "receipts"), filepath.Dir(data), "relativ"} {
		if _, err := b.SetInboxFolder(refused); err == nil {
			t.Errorf("Eingang %s angenommen", refused)
		}
	}
	// Geprüft, angelegt und gespeichert wird derselbe Pfad: „daten “ ist ein
	// Geschwisterordner des Datenordners, nicht der Datenordner selbst.
	spaced := data + " "
	if info, err = b.SetInboxFolder(spaced); err != nil || info.Path != spaced {
		t.Errorf("Ordner mit Leerzeichen am Ende: %+v %v", info, err)
	}
	if st, err := os.Stat(spaced); err != nil || !st.IsDir() {
		t.Errorf("nicht angelegt: %v", err)
	}
	// Ein Leseordner im Eingang und ein Eingang um einen Leseordner gehen nicht.
	if _, err := b.SaveAgentRules(domain.AgentSettings{ExtraReadFolders: []string{filepath.Join(custom, "Verträge")}}); err != nil {
		t.Fatalf("Leseordner, solange der Eingang woanders liegt: %v", err)
	}
	if _, err := b.SetInboxFolder(custom); err == nil {
		t.Error("Eingang um einen Leseordner angenommen")
	}
	if _, err := b.SaveAgentRules(domain.AgentSettings{ExtraReadFolders: []string{filepath.Join(spaced, "Verträge")}}); err == nil {
		t.Error("Leseordner im Eingang angenommen")
	}
	if info, err = b.SetInboxFolder(""); err != nil || info.Path != filepath.Join(data, policy.InboxDir) || !info.Default {
		t.Errorf("zurück zur Voreinstellung: %+v %v", info, err)
	}
}

// Ein Kontoauszug aus dem Eingang geht nach dem Import nach Übernommen/.
func TestStatementImportSettlesTheInbox(t *testing.T) {
	b := testBridge(t)
	data := tenantDataDir(t, b)
	if err := b.ConfigureBankAccounts([]domain.BankAccount{{IBAN: testIBAN, Name: "Geschäftskonto", LedgerAccount: "1800", Currency: "EUR"}}); err != nil {
		t.Fatal(err)
	}
	path := writeTestFile(t, filepath.Join(data, policy.InboxDir, "auszug.xml"), testStatement)
	if n, err := b.ImportCAMT053File(path, ""); err != nil || n != 1 {
		t.Fatalf("Import: %d %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(data, policy.InboxDir, inbox.DoneDir, time.Now().Format("2006-01"), "auszug.xml")); err != nil {
		t.Errorf("nicht verschoben: %v", err)
	}
	// Ein Auszug von anderswo bleibt, wo er ist.
	elsewhere := writeTestFile(t, filepath.Join(t.TempDir(), "auszug.xml"), testStatement)
	if _, err := b.ImportCAMT053File(elsewhere, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(elsewhere); err != nil {
		t.Errorf("Datei außerhalb des Eingangs bewegt: %v", err)
	}
}

// Die Einrichtung nimmt den Eingang mit; leer ist er <Datenordner>/Eingang.
func TestSetupCreatesTheInbox(t *testing.T) {
	b := newBridge(t)
	data := filepath.Join(t.TempDir(), "mandant")
	custom := filepath.Join(t.TempDir(), "Scanner")
	if err := b.SetupApplication(data, custom, domain.CompanySettings{CompanyName: "Pfennig GmbH", FiscalYear: 2026}); err != nil {
		t.Fatal(err)
	}
	info, err := b.GetInbox()
	if err != nil || info.Path != custom {
		t.Fatalf("Eingang nach der Einrichtung: %+v %v", info, err)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("nicht angelegt: %v", err)
	}

	other := newBridge(t)
	dir := filepath.Join(t.TempDir(), "mandant")
	if err := other.SetupApplication(dir, filepath.Join(dir, "Belege"), domain.CompanySettings{CompanyName: "X GmbH", FiscalYear: 2026}); err == nil {
		t.Error("ein Eingang im Datenordner außer dem voreingestellten wurde angenommen")
	}
	if err := other.SetupApplication(dir, "", domain.CompanySettings{CompanyName: "X GmbH", FiscalYear: 2026}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, policy.InboxDir)); err != nil {
		t.Errorf("voreingestellter Eingang fehlt: %v", err)
	}
}

// Ob ein Ordner synchronisiert wird, sagt CheckFolder; ein lokaler bleibt
// ohne Hinweis.
func TestCheckFolderWarnsOnlyForSyncedFolders(t *testing.T) {
	b := newBridge(t)
	local := t.TempDir()
	if got := b.CheckFolder(local); got.Sync != "" || got.Warning != "" {
		t.Errorf("lokal: %+v", got)
	}
	synced := filepath.Join(t.TempDir(), "Sync")
	if err := os.MkdirAll(filepath.Join(synced, ".stfolder"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := b.CheckFolder(filepath.Join(synced, "Buchfink")); got.Sync != "Syncthing" || got.Warning == "" {
		t.Errorf("Syncthing: %+v", got)
	}
}

// Ein Drop auf die Ablagezone kommt in den Eingang; daneben geschieht nichts.
func TestDropOnTheWindowFillsTheInbox(t *testing.T) {
	b := testBridge(t)
	file := writeTestFile(t, filepath.Join(t.TempDir(), "scan.pdf"), "%PDF-1.4 Scan")
	var reports []InboxAdded
	report := func(a InboxAdded) { reports = append(reports, a) }
	handleDrop(b, []string{file}, false, report)
	handleDrop(b, nil, true, report)
	if len(reports) != 0 {
		t.Fatalf("Drop neben die Ablagezone: %+v", reports)
	}
	handleDrop(b, []string{file}, true, report)
	if len(reports) != 1 || !slices.Equal(reports[0].Names, []string{"scan.pdf"}) || reports[0].Error != "" {
		t.Fatalf("Drop: %+v", reports)
	}
	if list, _ := b.ListInbox(); len(list) != 1 {
		t.Errorf("Eingang: %+v", list)
	}
	closed := newBridge(t)
	handleDrop(closed, []string{file}, true, report)
	if len(reports) != 2 || reports[1].Error == "" {
		t.Errorf("ohne Mandant: %+v", reports)
	}
}

// Jeder gelungene Vorgang erreicht die Oberfläche als gebündeltes Ereignis
// mit Mandant, Erfassungsweg und Art; ein gescheiterter nicht.
func TestChangesReachTheWindowBundled(t *testing.T) {
	flushed := make(chan ChangeEvent, 64)
	batcher := newChangeBatcher(200*time.Millisecond, func(ev ChangeEvent) { flushed <- ev })
	// drain sammelt Ereignisse, bis eine Weile keines mehr kommt.
	drain := func() []ChangeEvent {
		var out []ChangeEvent
		for {
			select {
			case ev := <-flushed:
				out = append(out, ev)
			case <-time.After(500 * time.Millisecond):
				return out
			}
		}
	}
	b := testBridge(t, func(o *core.HostOptions) { o.OnChange = batcher.add })
	if len(drain()) == 0 {
		t.Fatal("die Einrichtung hat nichts gemeldet")
	}

	for i := 0; i < 3; i++ {
		fileTestReceipt(t, b)
	}
	if _, err := b.AddToInbox([]string{writeTestFile(t, filepath.Join(t.TempDir(), "x.pdf"), "%PDF-1.4 x")}); err != nil {
		t.Fatal(err)
	}
	if err := b.change(func(ctx context.Context, s *core.Services) error { return errors.New("abgebrochen") }); err == nil {
		t.Fatal("der Vorgang hätte scheitern sollen")
	}
	events := drain()
	if len(events) == 0 || len(events) >= 4 {
		t.Fatalf("%d Ereignisse für vier Vorgänge: %+v", len(events), events)
	}
	kinds, channels := []string{}, []string{}
	for _, ev := range events {
		if ev.TenantID != b.host.ActiveTenant().ID {
			t.Errorf("Mandant: %+v", ev)
		}
		kinds, channels = append(kinds, ev.Kinds...), append(channels, ev.Channels...)
	}
	slices.Sort(kinds)
	slices.Sort(channels)
	if !slices.Equal(slices.Compact(kinds), []string{"belege", "eingang"}) || !slices.Equal(slices.Compact(channels), []string{"ui"}) {
		t.Errorf("Arten %v, Wege %v", kinds, channels)
	}

	if err := b.change(func(ctx context.Context, s *core.Services) error { return errors.New("abgebrochen") }); err == nil {
		t.Fatal("der Vorgang hätte scheitern sollen")
	}
	if events := drain(); len(events) != 0 {
		t.Errorf("Ereignis ohne gelungenen Vorgang: %+v", events)
	}
}

// Meldungen aus einem Fenster gehen zusammen hinaus, je Mandant eine;
// danach beginnt ein neues Fenster.
func TestChangeBatcherMergesPerTenant(t *testing.T) {
	var mu sync.Mutex
	var events []ChangeEvent
	done := make(chan struct{}, 4)
	c := newChangeBatcher(20*time.Millisecond, func(ev ChangeEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
		done <- struct{}{}
	})
	c.add(core.Change{TenantID: "a", Channel: "agent:claude-code", Kinds: []core.ChangeKind{core.ChangeBookings}})
	c.add(core.Change{TenantID: "b", Channel: "ui", Kinds: []core.ChangeKind{core.ChangeMasterData}})
	c.add(core.Change{TenantID: "a", Channel: "ui", Kinds: []core.ChangeKind{core.ChangeReceipts, core.ChangeBookings}})
	c.add(core.Change{TenantID: "a", Channel: "system"})
	<-done
	<-done
	c.add(core.Change{TenantID: "a", Channel: "ui", Kinds: []core.ChangeKind{core.ChangeInbox}})
	<-done
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 3 {
		t.Fatalf("Ereignisse: %+v", events)
	}
	if a := events[0]; a.TenantID != "a" || !slices.Equal(a.Channels, []string{"agent:claude-code", "system", "ui"}) ||
		!slices.Equal(a.Kinds, []string{"belege", "buchungen"}) {
		t.Errorf("Mandant a: %+v", a)
	}
	if b := events[1]; b.TenantID != "b" || !slices.Equal(b.Kinds, []string{"stammdaten"}) {
		t.Errorf("Mandant b: %+v", b)
	}
	if next := events[2]; !slices.Equal(next.Kinds, []string{"eingang"}) {
		t.Errorf("nächstes Fenster: %+v", next)
	}
}
