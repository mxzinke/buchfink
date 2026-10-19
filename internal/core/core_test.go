package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

var (
	uiCtx     = actor.WithChannel(context.Background(), actor.ChannelUI)
	systemCtx = actor.WithChannel(context.Background(), actor.ChannelSystem)
)

func agentCtx(t *testing.T) context.Context {
	t.Helper()
	c, err := actor.Agent("claude-desktop")
	if err != nil {
		t.Fatal(err)
	}
	return actor.WithChannel(context.Background(), c)
}

// newHost legt einen Host mit eigener Konfiguration im Testordner an. Er
// erreicht kein Netz.
func newHost(t *testing.T) *Host {
	t.Helper()
	return newHostWith(t, OfflineNetwork())
}

// newHostWith legt einen Host mit eigenen Netzwegen an.
func newHostWith(t *testing.T, network Network) *Host {
	t.Helper()
	base := t.TempDir()
	h, err := NewHost(context.Background(), HostOptions{
		Config:  repository.NewAppConfigRepository(filepath.Join(base, "config")),
		BaseDir: base,
		Network: network,
		Clock:   &fakeClock{now: coreTestNow()},
	})
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	t.Cleanup(func() { _ = h.Close(context.Background()) })
	return h
}

// newTenant legt einen Mandanten an und macht ihn zum offenen.
func newTenant(t *testing.T, h *Host, name string) *domain.TenantConfig {
	t.Helper()
	tc, err := h.CreateTenant(uiCtx, name, filepath.Join(t.TempDir(), "daten"),
		domain.CompanySettings{CompanyName: name, FiscalYear: 2026})
	if err != nil {
		t.Fatalf("Mandant %s anlegen: %v", name, err)
	}
	return tc
}

func openTenant(t *testing.T, h *Host) *Tenant {
	t.Helper()
	tenant, release, err := h.Acquire(context.Background())
	if err != nil {
		t.Fatalf("offener Mandant: %v", err)
	}
	release()
	return tenant
}

// receiptAndEntry legt einen Beleg an und liefert eine Handbuchung darauf.
func receiptAndEntry(ctx context.Context, s *Services) (service.ManualEntryRequest, error) {
	receipt, err := s.Receipts.File(ctx, service.FileReceiptRequest{
		Direction: domain.DirectionIncoming, FiscalYear: 2026,
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Büromarkt GmbH", GrossAmount: 10000,
		Files: []service.NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf",
			Content: []byte(fmt.Sprintf("%%PDF-1.4 Testbeleg %d", time.Now().UnixNano())),
		}},
	})
	if err != nil {
		return service.ManualEntryRequest{}, err
	}
	return service.ManualEntryRequest{ReceiptID: receipt.ID, Entry: domain.JournalEntry{
		BookingDate: "2026-03-01", Description: "Testbuchung",
		Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentNotTaxable,
		Lines: []domain.JournalLine{
			{Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}}, nil
}

func post(ctx context.Context, s *Services) (*domain.JournalEntry, error) {
	req, err := receiptAndEntry(ctx, s)
	if err != nil {
		return nil, err
	}
	return s.Posting.PostManualEntry(ctx, req)
}

func countEntries(t *testing.T, tenant *Tenant) int {
	t.Helper()
	n := 0
	if err := tenant.Read(systemCtx, func(ctx context.Context, s *Services) error {
		entries, err := s.Repos.Journal.FindAll(ctx, 2026)
		n = len(entries)
		return err
	}); err != nil {
		t.Fatalf("Journal lesen: %v", err)
	}
	return n
}

// Open verdrahtet jeden Dienst; ein vergessener stünde erst im Betrieb als
// nil da.
func TestOpenWiresEveryService(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	value := reflect.ValueOf(tenant.svc).Elem()
	for i := 0; i < value.NumField(); i++ {
		field := value.Type().Field(i)
		if field.Type.Kind() == reflect.Ptr && value.Field(i).IsNil() {
			t.Errorf("Dienst %s ist nicht verdrahtet", field.Name)
		}
	}
	repos := reflect.ValueOf(tenant.svc.Repos)
	for i := 0; i < repos.NumField(); i++ {
		if repos.Field(i).IsNil() {
			t.Errorf("Repository %s ist nicht verdrahtet", repos.Type().Field(i).Name)
		}
	}
	if tenant.StartYear() != 2026 {
		t.Errorf("Startjahr %d, erwartet 2026 aus den Unternehmensdaten", tenant.StartYear())
	}
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		years, err := s.Closing.FiscalYears(ctx)
		if err == nil && len(years) != 1 {
			return fmt.Errorf("Geschäftsjahre %+v, erwartet 2026", years)
		}
		return err
	}); err != nil {
		t.Error(err)
	}
}

// Jeder Einstieg nennt seinen Erfassungsweg; ohne ihn geht nichts.
func TestEveryEntryNeedsAChannel(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	noop := func(context.Context, *Services) error { return nil }
	if err := h.Write(context.Background(), noop); !errors.Is(err, ErrNoChannel) {
		t.Errorf("Write ohne Erfassungsweg: %v", err)
	}
	if err := h.Read(context.Background(), noop); !errors.Is(err, ErrNoChannel) {
		t.Errorf("Read ohne Erfassungsweg: %v", err)
	}
	if _, err := h.Switch(context.Background(), "x"); !errors.Is(err, ErrNoChannel) {
		t.Errorf("Wechsel ohne Erfassungsweg: %v", err)
	}

	// Ein unbekannter Weg käme so an Buchung und Protokoll; er wird abgewiesen.
	unknown := actor.WithChannel(context.Background(), actor.Channel("mcp:claude"))
	if err := h.Write(unknown, noop); !errors.Is(err, ErrUnknownChannel) {
		t.Errorf("Write mit unbekanntem Erfassungsweg: %v", err)
	}
	if err := h.Read(unknown, noop); !errors.Is(err, ErrUnknownChannel) {
		t.Errorf("Read mit unbekanntem Erfassungsweg: %v", err)
	}
	if _, err := h.Switch(unknown, "x"); !errors.Is(err, ErrUnknownChannel) {
		t.Errorf("Wechsel mit unbekanntem Erfassungsweg: %v", err)
	}
}

// Die Repositories, die Bridge und MCP-Server in die Hand bekommen, lesen nur.
// Festschreibung, Prüfermodus, Abschlussstand und Einstellungen ändern sich
// allein über Kern und Dienste mit Prüfung und Protokoll.
func TestReposOnlyRead(t *testing.T) {
	writing := []string{"Save", "Create", "Update", "Set", "Delete", "Mark", "LogChange", "Append", "Upsert"}
	repos := reflect.TypeOf(Repos{})
	for i := 0; i < repos.NumField(); i++ {
		field := repos.Field(i)
		for m := 0; m < field.Type.NumMethod(); m++ {
			name := field.Type.Method(m).Name
			for _, prefix := range writing {
				if strings.HasPrefix(name, prefix) {
					t.Errorf("Repos.%s bietet %s an", field.Name, name)
				}
			}
		}
	}
}

// Ein Vorgang ist eine Transaktion: scheitert er nach der Buchung, steht sie
// nicht im Journal — und der Erfassungsweg steht an jeder Buchung.
func TestWriteIsOneTransaction(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	failure := errors.New("Abbruch nach der Buchung")
	err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := post(ctx, s); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("Fehler %v, erwartet den Abbruch", err)
	}
	if n := countEntries(t, tenant); n != 0 {
		t.Fatalf("%d Buchungen nach dem Abbruch, erwartet keine", n)
	}

	var created *domain.JournalEntry
	if err := tenant.Write(agentCtx(t), func(ctx context.Context, s *Services) (err error) {
		created, err = post(ctx, s)
		return err
	}); err != nil {
		t.Fatalf("Buchung: %v", err)
	}
	if created.Channel != "agent:claude-desktop" {
		t.Errorf("die Buchung trägt den Weg %q", created.Channel)
	}
	if n := countEntries(t, tenant); n != 1 {
		t.Errorf("%d Buchungen, erwartet 1", n)
	}
}

// Lesen schreibt nicht. Was es doch versucht, scheitert an der Verbindung.
func TestReadCannotWrite(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		return tenant.repos.settings.Set(ctx, "company_name", "Geändert GmbH")
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "readonly database") {
		t.Fatalf("Schreiben im Lesevorgang: %v", err)
	}
	err = tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		return tenant.Write(ctx, func(context.Context, *Services) error { return nil })
	})
	if !errors.Is(err, ErrWriteInRead) {
		t.Errorf("ein Write im Lesevorgang: %v", err)
	}
	// Der Zugriff wird trotzdem protokolliert (QUE-02).
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		return s.Repos.Audit.Log(ctx, domain.AuditActionExport, "JOURNAL", "2026", "Journal herausgegeben")
	}); err != nil {
		t.Errorf("Protokolleintrag im Lesevorgang: %v", err)
	}
}

// Festschreibung, Übermittlungsbestätigung, Abschlussstatus, Stammdaten und
// Prüfermodus bleiben der App vorbehalten — auch gegen einen Hintergrundlauf.
func TestAppOnlyWritesRefuseEveryOtherChannel(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	noop := func(context.Context, *Services) error { return nil }

	for name, ctx := range map[string]context.Context{"Agent": agentCtx(t), "Programmlauf": systemCtx} {
		if err := tenant.Write(ctx, noop, AppOnly()); !errors.Is(err, ErrAppOnly) {
			t.Errorf("%s: %v, erwartet ErrAppOnly", name, err)
		}
	}
	if err := tenant.Write(uiCtx, noop, AppOnly()); err != nil {
		t.Errorf("die App wird abgewiesen: %v", err)
	}
	until := coreTestNow().AddDate(0, 0, 10).Format("2006-01-02")
	if _, err := h.EnableExaminer(agentCtx(t), until, "Prüfung"); !errors.Is(err, ErrAppOnly) {
		t.Errorf("Prüfermodus durch einen Agenten: %v", err)
	}
	if _, err := h.CreateTenant(agentCtx(t), "Zweite GmbH", "", domain.CompanySettings{}); !errors.Is(err, ErrAppOnly) {
		t.Errorf("Mandant durch einen Agenten angelegt: %v", err)
	}
	if err := h.RemoveTenant(agentCtx(t), tenant.ID()); !errors.Is(err, ErrAppOnly) {
		t.Errorf("Mandant durch einen Agenten entfernt: %v", err)
	}
}

// Der Prüfermodus sperrt jeden Schreibvorgang außer den ausgenommenen, auch
// einen Programmlauf: ein buchender Hintergrundlauf darf die Bücher während
// der Prüfung so wenig ändern wie die Oberfläche.
func TestExaminerModeBlocksWrites(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	noop := func(context.Context, *Services) error { return nil }

	until := coreTestNow().AddDate(0, 0, 10).Format("2006-01-02")
	if _, err := h.EnableExaminer(uiCtx, until, "Betriebsprüfung 2024"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}
	for name, ctx := range map[string]context.Context{"App": uiCtx, "Agent": agentCtx(t), "Programmlauf": systemCtx} {
		err := tenant.Write(ctx, noop)
		if err == nil || !strings.Contains(err.Error(), "Betriebsprüfung 2024") {
			t.Errorf("%s schreibt im Prüfermodus: %v", name, err)
		}
	}
	if _, err := bookOn(systemCtx, tenant, "2026-03-01"); err == nil {
		t.Error("ein Programmlauf bucht im Prüfermodus")
	}
	if err := tenant.Write(systemCtx, noop, ExaminerExempt()); err != nil {
		t.Errorf("ausgenommener Programmlauf abgewiesen: %v", err)
	}
	if err := tenant.Write(uiCtx, noop, ExaminerExempt()); err != nil {
		t.Errorf("ausgenommener Vorgang abgewiesen: %v", err)
	}
	if err := tenant.Read(agentCtx(t), noop); err != nil {
		t.Errorf("Lesen im Prüfermodus: %v", err)
	}
	if _, err := tenant.DisableExaminer(agentCtx(t), "Agent will buchen"); !errors.Is(err, ErrAppOnly) {
		t.Errorf("ein Agent beendet den Prüfermodus: %v", err)
	}
	if _, err := h.DisableExaminer(uiCtx, "Prüfung beendet"); err != nil {
		t.Fatalf("Prüfermodus beenden: %v", err)
	}
	if err := tenant.Write(uiCtx, noop); err != nil {
		t.Errorf("nach dem Prüfermodus: %v", err)
	}
}

// Schreiber warten aufeinander; ein Wartender kann über seinen Kontext
// abbrechen, und Leser warten nicht.
func TestWritersQueueAndReadersDoNot(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	inside, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
			if _, err := post(ctx, s); err != nil {
				return err
			}
			close(inside)
			<-release
			return nil
		})
	}()
	<-inside

	waiting, cancel := context.WithTimeout(uiCtx, 50*time.Millisecond)
	defer cancel()
	if err := tenant.Write(waiting, func(context.Context, *Services) error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("ein zweiter Schreiber kam an der Sperre vorbei: %v", err)
	}

	start := time.Now()
	if n := countEntries(t, tenant); n != 0 {
		t.Errorf("der Leser sieht %d Buchungen aus einer offenen Transaktion", n)
	}
	if time.Since(start) > time.Second {
		t.Errorf("der Leser wartete %s auf den Schreiber", time.Since(start))
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Schreiber: %v", err)
	}
	if n := countEntries(t, tenant); n != 1 {
		t.Errorf("%d Buchungen nach dem Schreiber, erwartet 1", n)
	}
}

// Viele Leser und Schreiber zugleich, die Leser in verschiedenen
// Geschäftsjahren. Unter `go test -race` belegt der Test, dass Dienste und
// Kontenplan das aushalten; das Jahr des einen Lesers verschiebt nichts beim
// anderen, weil kein Dienst ein Jahr hält.
func TestConcurrentReadersWritersInDifferentYears(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 4; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			errs <- tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
				_, err := post(ctx, s)
				return err
			})
		}()
		go func() {
			defer wg.Done()
			errs <- tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
				_, err := s.Accounting.GetSuSaOverview(ctx, 2026)
				return err
			})
		}()
		go func() {
			defer wg.Done()
			errs <- tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
				susa, err := s.Accounting.GetSuSaOverview(ctx, 2025)
				if err == nil && (susa.FiscalYear != 2025 || susa.TotalDebit != 0) {
					return fmt.Errorf("SuSa 2025 zeigt Buchungen aus 2026: %+v", susa)
				}
				return err
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	if n := countEntries(t, tenant); n != 4 {
		t.Errorf("%d Buchungen, erwartet 4", n)
	}
}

// Der Wechsel schließt den vorigen Mandanten. Fünfzig Wechsel lassen keine
// Dateien offen zurück.
func TestSwitchClosesThePreviousTenant(t *testing.T) {
	if _, err := os.ReadDir("/proc/self/fd"); err != nil {
		t.Skip("ohne /proc lassen sich offene Dateien nicht zählen")
	}
	h := newHost(t)
	first := newTenant(t, h, "Erste GmbH")
	second := newTenant(t, h, "Zweite GmbH")
	gen := h.Generation()

	openFiles := func() int {
		h.closing.Wait()
		entries, _ := os.ReadDir("/proc/self/fd")
		return len(entries)
	}
	if _, err := h.Switch(uiCtx, first.ID); err != nil {
		t.Fatal(err)
	}
	before := openFiles()
	for i := 0; i < 50; i++ {
		id := second.ID
		if i%2 == 1 {
			id = first.ID
		}
		if _, err := h.Switch(uiCtx, id); err != nil {
			t.Fatalf("Wechsel %d: %v", i, err)
		}
	}
	if after := openFiles(); after > before+4 {
		t.Errorf("offene Dateien: %d vor, %d nach 50 Wechseln", before, after)
	}
	if got := h.Generation(); got != gen+51 {
		t.Errorf("Generation %d nach 51 Wechseln ab %d", got, gen)
	}
	if active := h.ActiveTenant(); active == nil || active.ID != first.ID {
		t.Errorf("aktiv ist %+v, erwartet %s", active, first.ID)
	}
}

// Ein laufender Schreibvorgang wird beim Wechsel fertig; der alte Mandant
// schließt erst danach.
func TestSwitchWaitsForARunningWrite(t *testing.T) {
	h := newHost(t)
	first := newTenant(t, h, "Erste GmbH")
	second := newTenant(t, h, "Zweite GmbH")
	if _, err := h.Switch(uiCtx, first.ID); err != nil {
		t.Fatal(err)
	}
	old := openTenant(t, h)

	inside, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- h.Write(uiCtx, func(ctx context.Context, s *Services) error {
			if _, err := post(ctx, s); err != nil {
				return err
			}
			close(inside)
			<-release
			return nil
		})
	}()
	<-inside

	if _, err := h.Switch(uiCtx, second.ID); err != nil {
		t.Fatalf("Wechsel während eines Schreibvorgangs: %v", err)
	}
	if got := openTenant(t, h); got.ID() != second.ID {
		t.Fatalf("offen ist %s, erwartet %s", got.ID(), second.ID)
	}
	if err := old.Write(uiCtx, func(context.Context, *Services) error { return nil }); !errors.Is(err, ErrTenantClosing) {
		t.Errorf("der alte Mandant nimmt nach dem Wechsel neue Vorgänge an: %v", err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("der laufende Schreibvorgang scheiterte am Wechsel: %v", err)
	}
	h.closing.Wait()
	sqlDB, err := old.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Ping(); err == nil {
		t.Error("die Datenbank des alten Mandanten ist nach dem Wechsel noch offen")
	}

	reopened, err := Open(systemCtx, old.DataDir(), Options{TenantID: first.ID})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close(context.Background())
	if n := countEntries(t, reopened); n != 1 {
		t.Errorf("die Buchung des laufenden Vorgangs fehlt: %d Buchungen", n)
	}
}

// Kann die Konfiguration nicht gespeichert werden, bleibt alles, wie es war.
func TestCreateTenantRollsBackWhenTheConfigCannotBeSaved(t *testing.T) {
	base := t.TempDir()
	h, err := NewHost(context.Background(), HostOptions{Config: unsavableConfig{}, BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close(context.Background())

	dir := filepath.Join(base, "firma")
	if _, err := h.CreateTenant(uiCtx, "Fehlerfall", dir, domain.CompanySettings{FiscalYear: 2025}); err == nil {
		t.Fatal("angelegt, ohne die Konfiguration zu speichern")
	}
	if cfg := h.Config(); cfg.IsConfigured || len(cfg.Tenants) != 0 {
		t.Errorf("halber Mandant in der Konfiguration: %+v", cfg)
	}
	if _, _, err := h.Acquire(context.Background()); !errors.Is(err, ErrNoTenant) {
		t.Errorf("ein Mandant ist offen: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("der Datenordner blieb zurück: %v", err)
	}
}

// Ein neuer Host öffnet den zuletzt aktiven Mandanten wieder.
func TestNewHostReopensTheActiveTenant(t *testing.T) {
	base := t.TempDir()
	config := repository.NewAppConfigRepository(filepath.Join(base, "config"))
	h, err := NewHost(context.Background(), HostOptions{Config: config, BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	newTenant(t, h, "Erste GmbH")
	second := newTenant(t, h, "Zweite GmbH")
	if err := h.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	again, err := NewHost(context.Background(), HostOptions{Config: config, BaseDir: base})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close(context.Background())
	if got := openTenant(t, again); got.ID() != second.ID {
		t.Errorf("geöffnet ist %s, erwartet %s", got.ID(), second.ID)
	}
}

// unsavableConfig lädt eine leere Konfiguration und kann nicht speichern.
type unsavableConfig struct{}

func (unsavableConfig) Load() (*domain.AppConfig, error) { return &domain.AppConfig{}, nil }
func (unsavableConfig) Save(*domain.AppConfig) error {
	return errors.New("Datenträger schreibgeschützt")
}

// Der Prüfermodus endet von selbst: ein vergangenes Datum sperrt nichts mehr,
// und die Konfiguration meldet ihn nicht mehr — auch wenn seit dem Ablauf
// nichts gespeichert wurde.
func TestExaminerModeExpires(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	yesterday := coreTestNow().AddDate(0, 0, -1).Format("2006-01-02")
	tenant := openTenant(t, h)
	if tenant.ID() != tc.ID {
		t.Fatalf("offen ist %s", tenant.ID())
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		return tenant.repos.examiner.Create(ctx, &domain.ExaminerPeriod{
			From: coreTestNow().AddDate(0, 0, -10).Format("2006-01-02"), Until: yesterday,
			Reason: "abgelaufene Prüfung",
		})
	}, ExaminerExempt()); err != nil {
		t.Fatal(err)
	}
	if err := tenant.loadExaminer(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cfg := h.Config(); cfg.ReadOnly {
		t.Error("die Konfiguration meldet einen abgelaufenen Prüfermodus")
	}
	if err := h.Write(uiCtx, func(context.Context, *Services) error { return nil }); err != nil {
		t.Errorf("nach dem Ablauf muss wieder geschrieben werden können: %v", err)
	}
}

// Die automatische Sicherung ist ein Programmlauf: ihr Vermerk trägt den
// Erfassungsweg system und geht über einen Schreibvorgang.
func TestDueBackupRunsAsSystem(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	if _, err := h.SetBackupDir(uiCtx, t.TempDir()); err != nil {
		t.Fatalf("Sicherungsordner: %v", err)
	}
	h.RunDueBackup(context.Background(), false)

	var runs []domain.BackupRun
	var logged []domain.AuditLogEntry
	if err := h.Read(uiCtx, func(ctx context.Context, s *Services) error {
		var err error
		if runs, err = s.Backup.GetRuns(ctx, 5); err != nil {
			return err
		}
		logged, err = s.Repos.Audit.FindFiltered(ctx, 0, domain.AuditFilter{EntityType: "BACKUP"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || !runs[0].Success || runs[0].Kind != domain.BackupKindAutomatic {
		t.Fatalf("Läufe nach der fälligen Sicherung: %+v", runs)
	}
	if len(logged) != 1 || logged[0].Channel != string(actor.ChannelSystem) {
		t.Errorf("Vermerk der Sicherung: %+v", logged)
	}
	if h.Config().LastBackupAt == "" {
		t.Error("der Zeitpunkt der Sicherung fehlt in der Konfiguration")
	}
}

// Ein Lesevorgang, der einen Zugriff protokolliert, wartet auf einen
// laufenden Schreibvorgang, statt mit „database is locked" zu scheitern.
func TestAccessLogDuringAWriteWaits(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)

	inside, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
			if _, err := post(ctx, s); err != nil {
				return err
			}
			close(inside)
			<-release
			return nil
		})
	}()
	<-inside

	logged := make(chan error, 1)
	go func() {
		logged <- tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
			return s.Repos.Audit.Log(ctx, domain.AuditActionExport, "JOURNAL", "2026", "Journal herausgegeben")
		})
	}()
	time.Sleep(200 * time.Millisecond)
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Schreibvorgang: %v", err)
	}
	if err := <-logged; err != nil {
		t.Fatalf("der Protokolleintrag scheiterte am Schreibvorgang: %v", err)
	}

	var chain domain.AuditChainResult
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		var err error
		chain, err = s.Audit.VerifyChain(ctx)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !chain.IsValid {
		t.Errorf("Protokollkette nach parallelem Anhängen gebrochen: %s", chain.Message)
	}
}

// Der Prüfermodus steht in der Mandantendatenbank: er gilt nach dem
// Neuöffnen, nach dem Entfernen aus der Liste und in einer wiederhergestellten
// Sicherung weiter. In der App-Konfiguration stünde er nur auf diesem Rechner.
func TestExaminerModeTravelsWithTheDatabase(t *testing.T) {
	h := newHost(t)
	tc := newTenant(t, h, "Pfennig Ventures GmbH")
	backupDir := t.TempDir()
	if _, err := h.SetBackupDir(uiCtx, backupDir); err != nil {
		t.Fatal(err)
	}
	until := coreTestNow().AddDate(0, 0, 30).Format("2006-01-02")
	if _, err := h.EnableExaminer(uiCtx, until, "Betriebsprüfung 2023 bis 2025"); err != nil {
		t.Fatalf("Prüfermodus einschalten: %v", err)
	}
	run, err := h.RunBackup(uiCtx, domain.BackupKindManual)
	if err != nil {
		t.Fatalf("Sicherung im Prüfermodus: %v", err)
	}

	// Aus der Liste genommen und wieder angemeldet: die Sperre ist noch da.
	if err := h.RemoveTenant(uiCtx, tc.ID); err != nil {
		t.Fatalf("Mandant entfernen: %v", err)
	}
	if _, err := h.RegisterTenant(uiCtx, filepath.Join(tc.DataDir, "buchfink.sqlite"), "", domain.MigrationKindOpen); err != nil {
		t.Fatalf("wieder anmelden: %v", err)
	}
	if cfg := h.Config(); !cfg.ReadOnly || cfg.ReadOnlyUntil != until {
		t.Errorf("nach dem Wiederanmelden fehlt der Prüfermodus: %+v", cfg)
	}

	// Und in der wiederhergestellten Sicherung.
	if _, err := h.RestoreBackup(uiCtx, run.Target, filepath.Join(t.TempDir(), "zurueck")); err != nil {
		t.Fatalf("Wiederherstellen: %v", err)
	}
	restored := openTenant(t, h)
	if state := restored.Examiner(); !state.Active || state.Reason != "Betriebsprüfung 2023 bis 2025" {
		t.Errorf("die wiederhergestellte Sicherung kennt den Prüfermodus nicht: %+v", state)
	}
	if err := restored.Write(uiCtx, func(context.Context, *Services) error { return nil }); err == nil {
		t.Error("die wiederhergestellte Sicherung nimmt im Prüfermodus Änderungen an")
	}
}

// Ein- und Ausschalten stehen als Zeiträume in der Datenbank; ein neuer
// Zeitraum löst den laufenden ab, statt ihn zu überschreiben.
func TestExaminerPeriodsKeepTheirHistory(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	first := coreTestNow().AddDate(0, 0, 10).Format("2006-01-02")
	second := coreTestNow().AddDate(0, 0, 20).Format("2006-01-02")
	if _, err := tenant.EnableExaminer(uiCtx, first, "Prüfung"); err != nil {
		t.Fatal(err)
	}
	if state, err := tenant.EnableExaminer(uiCtx, second, "Prüfung verlängert"); err != nil || state.Until != second {
		t.Fatalf("Verlängern: %+v, %v", state, err)
	}
	if state, err := tenant.DisableExaminer(uiCtx, "Prüfung beendet"); err != nil || state.Active {
		t.Fatalf("Beenden: %+v, %v", state, err)
	}
	if _, err := tenant.DisableExaminer(uiCtx, "noch einmal"); err == nil {
		t.Error("ein nicht laufender Prüfermodus ließ sich beenden")
	}
	var periods []domain.ExaminerPeriod
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
		return tenant.reader.WithContext(ctx).Order("id").Find(&periods).Error
	}); err != nil {
		t.Fatal(err)
	}
	if len(periods) != 2 || periods[0].EndedOn == nil || periods[1].EndedOn == nil ||
		periods[1].EndReason != "Prüfung beendet" || periods[0].Channel != "ui" {
		t.Errorf("Verlauf der Zeiträume: %+v", periods)
	}
}

// Eine Sicherung im Eingang oder in einem Leseordner wäre für verbundene Apps
// erreichbar; der Sicherungsordner darf dort nicht liegen. Eine Sicherung um
// den Eingang herum geht.
func TestBackupDirStaysOutOfAgentFolders(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	dropbox := t.TempDir()
	scans := filepath.Join(dropbox, "Scans")
	read := t.TempDir()
	for _, d := range []string{filepath.Join(scans, "Sicherung"), filepath.Join(read, "Sicherung")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := s.Settings.SetInboxFolder(ctx, scans); err != nil {
			return err
		}
		_, err := s.Settings.SaveAgentSettings(ctx, domain.AgentSettings{ExtraReadFolders: []string{read}})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, refused := range []string{scans, filepath.Join(scans, "Sicherung"), filepath.Join(read, "Sicherung")} {
		if _, err := h.SetBackupDir(uiCtx, refused); err == nil {
			t.Errorf("Sicherung in %s angenommen", refused)
		}
	}
	if _, err := h.SetBackupDir(uiCtx, dropbox); err != nil {
		t.Errorf("Sicherung um den Eingang: %v", err)
	}
}

func coreTestNow() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }
func today() string          { return coreTestNow().Format(domain.DateLayout) }
