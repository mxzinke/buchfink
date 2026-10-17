package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
)

// HostOptions beschreibt den Host eines Prozesses.
type HostOptions struct {
	// Config ist die Ablage der App-Konfiguration mit der Mandantenliste.
	Config domain.AppConfigRepository
	// BaseDir nimmt neue Mandanten ohne eigene Ordnerangabe auf
	// (BaseDir/tenants/<kennung>/data).
	BaseDir string
	// Rates holt Referenzkurse; ohne Angabe die EZB.
	Rates domain.CurrencyFetcher
	// WarmRenderer übersetzt das Satzprogramm beim Start im Hintergrund, damit
	// die erste Rechnung nicht darauf wartet.
	WarmRenderer bool
	Logger       *slog.Logger
}

// Host hält die Mandantenliste und genau einen offenen Mandanten. Ein Prozess
// ist der einzige Schreiber je Mandant; Bridge und MCP-Server teilen sich den
// Host und damit dieselben Dienste.
//
// Sperr-Reihenfolge: lifecycle → Schreibsperre eines Mandanten → mu. mu wird
// nur kurz für Konfiguration und Zeiger gehalten.
type Host struct {
	opts     HostOptions
	cfgRepo  domain.AppConfigRepository
	renderer *invoice.Renderer
	logger   *slog.Logger

	// lifecycle ordnet Öffnen, Wechseln, Anlegen und Entfernen.
	lifecycle sync.Mutex

	mu      sync.RWMutex
	cfg     domain.AppConfig
	current *Tenant
	gen     uint64

	closing sync.WaitGroup
}

// NewHost lädt die Konfiguration und öffnet den aktiven Mandanten.
func NewHost(ctx context.Context, o HostOptions) (*Host, error) {
	if o.Config == nil {
		return nil, fmt.Errorf("ohne Ablage der Konfiguration lässt sich kein Host einrichten")
	}
	cfg, err := o.Config.Load()
	if err != nil {
		return nil, fmt.Errorf("die Konfiguration ließ sich nicht laden: %w", err)
	}
	cfg.EnsureLists()
	logger := o.Logger
	if logger == nil {
		logger = slog.Default()
	}
	h := &Host{opts: o, cfgRepo: o.Config, cfg: *cfg, renderer: invoice.NewRenderer(), logger: logger}
	if o.WarmRenderer {
		go func(r *invoice.Renderer) { _ = r.Warm(context.Background()) }(h.renderer)
	}

	if !cfg.IsConfigured || len(cfg.Tenants) == 0 {
		return h, nil
	}
	active := cfg.ActiveTenant()
	if active == nil {
		active = &cfg.Tenants[0]
	}
	if _, err := h.switchTo(systemContext(ctx), *active); err != nil {
		_ = h.renderer.Close(ctx)
		return nil, err
	}
	return h, nil
}

// systemContext setzt den Erfassungsweg system, wo der Aufrufer keinen nennt.
func systemContext(ctx context.Context) context.Context {
	if _, ok := actor.ChannelOf(ctx); ok {
		return ctx
	}
	return actor.WithChannel(ctx, actor.ChannelSystem)
}

// requireApp behält die Mandantenverwaltung der Bedienung in der App vor.
func requireApp(ctx context.Context) error {
	channel, ok := actor.ChannelOf(ctx)
	if !ok {
		return ErrNoChannel
	}
	if channel != actor.ChannelUI {
		return ErrAppOnly
	}
	return nil
}

// Close schließt den offenen Mandanten, wartet auf Mandanten, die nach einem
// Wechsel noch schließen, und gibt das Satzprogramm frei.
func (h *Host) Close(ctx context.Context) error {
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()
	h.mu.Lock()
	current := h.current
	h.current = nil
	h.mu.Unlock()
	var errs []error
	if current != nil {
		errs = append(errs, current.Close(ctx))
	}
	h.closing.Wait()
	errs = append(errs, h.renderer.Close(ctx))
	return errors.Join(errs...)
}

// Generation zählt die Mandantenwechsel.
func (h *Host) Generation() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.gen
}

// Acquire liefert den offenen Mandanten mit einer Lease. Solange die Lease
// läuft, schließt ein Wechsel den Mandanten nicht; release gibt sie zurück.
func (h *Host) Acquire(ctx context.Context) (*Tenant, func(), error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.current == nil {
		return nil, nil, ErrNoTenant
	}
	release, err := h.current.leases.acquire()
	if err != nil {
		return nil, nil, err
	}
	return h.current, release, nil
}

// Read liest im offenen Mandanten (siehe Tenant.Read).
func (h *Host) Read(ctx context.Context, fn func(context.Context, *Services) error) error {
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return t.Read(ctx, fn)
}

// Write schreibt im offenen Mandanten (siehe Tenant.Write).
func (h *Host) Write(ctx context.Context, fn func(context.Context, *Services) error, opts ...WriteOption) error {
	t, release, err := h.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return t.Write(ctx, fn, opts...)
}

// Config liefert die Konfiguration als Kopie, die Spiegelfelder des aktiven
// Mandanten auf den heutigen Tag gerechnet.
func (h *Host) Config() domain.AppConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.configLocked()
}

func (h *Host) configLocked() domain.AppConfig {
	cfg := h.cfg
	cfg.Tenants = append(make([]domain.TenantConfig, 0, len(h.cfg.Tenants)), h.cfg.Tenants...)
	cfg.SyncActiveTenant(today())
	cfg.ProgramVersion = buildinfo.Version
	return cfg
}

// ActiveTenant liefert den Eintrag des aktiven Mandanten oder nil.
func (h *Host) ActiveTenant() *domain.TenantConfig {
	cfg := h.Config()
	if active := cfg.ActiveTenant(); active != nil {
		return active
	}
	if len(cfg.Tenants) > 0 {
		return &cfg.Tenants[0]
	}
	return nil
}

// updateConfig ändert die Konfiguration und speichert sie. Scheitert das
// Speichern, bleibt der vorige Stand im Speicher.
func (h *Host) updateConfig(change func(cfg *domain.AppConfig) error) (domain.AppConfig, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	next := h.cfg
	next.Tenants = append([]domain.TenantConfig(nil), h.cfg.Tenants...)
	if err := change(&next); err != nil {
		return h.configLocked(), err
	}
	next.EnsureLists()
	if err := h.cfgRepo.Save(&next); err != nil {
		return h.configLocked(), fmt.Errorf("die Konfiguration ließ sich nicht speichern: %w", err)
	}
	h.cfg = next
	return h.configLocked(), nil
}

// updateTenantConfig ändert den Eintrag eines Mandanten.
func (h *Host) updateTenantConfig(id string, change func(t *domain.TenantConfig) error) (domain.AppConfig, error) {
	return h.updateConfig(func(cfg *domain.AppConfig) error {
		for i := range cfg.Tenants {
			if cfg.Tenants[i].ID == id {
				return change(&cfg.Tenants[i])
			}
		}
		return fmt.Errorf("der Mandant %s steht nicht in der Mandantenliste", id)
	})
}

// SetLastFiscalYear merkt sich das zuletzt angesehene Geschäftsjahr.
func (h *Host) SetLastFiscalYear(year int) error {
	_, err := h.updateConfig(func(cfg *domain.AppConfig) error {
		cfg.LastFiscalYear = year
		return nil
	})
	return err
}

// RenameTenant ändert den Anzeigenamen eines Mandanten in der Liste.
func (h *Host) RenameTenant(id, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	_, err := h.updateTenantConfig(id, func(t *domain.TenantConfig) error {
		t.Name = name
		return nil
	})
	return err
}

func (h *Host) tenantConfig(id string) (domain.TenantConfig, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, t := range h.cfg.Tenants {
		if t.ID == id {
			return t, true
		}
	}
	return domain.TenantConfig{}, false
}

// examinerOf liest den Prüfermodus eines Mandanten aus der Konfiguration.
func (h *Host) examinerOf(id string) ExaminerState {
	tc, ok := h.tenantConfig(id)
	if !ok {
		return ExaminerState{}
	}
	return ExaminerState{Active: tc.ReadOnlyActiveOn(today()), Until: tc.ReadOnlyUntil, Reason: tc.ReadOnlyReason}
}

func (h *Host) backupDirOf(id string) string {
	tc, _ := h.tenantConfig(id)
	return tc.BackupDir
}

// open öffnet einen Mandanten der Liste unter der nächsten Generation. Der
// Aufrufer hält lifecycle.
func (h *Host) open(ctx context.Context, tc domain.TenantConfig, initialYear int) (*Tenant, error) {
	h.mu.RLock()
	gen := h.gen + 1
	if initialYear == 0 {
		initialYear = h.cfg.LastFiscalYear
	}
	h.mu.RUnlock()
	id := tc.ID
	return Open(ctx, tc.DataDir, Options{
		TenantID: id, TenantName: tc.Name, Generation: gen, InitialYear: initialYear,
		Renderer:  h.renderer,
		Examiner:  func() ExaminerState { return h.examinerOf(id) },
		BackupDir: func() string { return h.backupDirOf(id) },
		Rates:     h.opts.Rates,
		Logger:    h.logger,
	})
}

// activate macht t zum offenen Mandanten und schließt den bisherigen im
// Hintergrund, sobald dessen laufende Vorgänge fertig sind.
func (h *Host) activate(t *Tenant) {
	h.mu.Lock()
	previous := h.current
	h.current = t
	if t != nil {
		h.gen = t.Generation()
	}
	h.mu.Unlock()
	if previous != nil && previous != t {
		// Ab hier nimmt der vorige Mandant nichts Neues mehr an; was läuft,
		// läuft zu Ende.
		previous.leases.stop()
		h.closing.Add(1)
		go func() {
			defer h.closing.Done()
			if err := previous.Close(context.Background()); err != nil {
				h.logger.Warn("Mandant ließ sich nicht sauber schließen", "mandant", previous.ID(), "fehler", err)
			}
		}()
	}
}

// Switch wechselt zum Mandanten mit der Kennung. Nur die App wechselt.
func (h *Host) Switch(ctx context.Context, tenantID string) (*Tenant, error) {
	if err := requireApp(ctx); err != nil {
		return nil, err
	}
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()
	tc, ok := h.tenantConfig(tenantID)
	if !ok {
		return nil, fmt.Errorf("der Mandant %s steht nicht in der Mandantenliste", tenantID)
	}
	return h.switchTo(ctx, tc)
}

// switchTo öffnet tc, merkt ihn als aktiven Mandanten und schließt den
// bisherigen. Der Aufrufer hält lifecycle oder ist NewHost.
func (h *Host) switchTo(ctx context.Context, tc domain.TenantConfig) (*Tenant, error) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()
	if current != nil && current.ID() == tc.ID {
		return current, nil
	}
	t, err := h.open(ctx, tc, 0)
	if err != nil {
		return nil, err
	}
	if _, err := h.updateConfig(func(cfg *domain.AppConfig) error {
		cfg.ActiveTenantID = tc.ID
		cfg.DataDir = tc.DataDir
		cfg.LastFiscalYear = t.FiscalYear()
		return nil
	}); err != nil {
		// Der Wechsel gilt trotzdem; beim nächsten Start öffnet der vorige
		// Mandant.
		h.logger.Warn("aktiver Mandant ließ sich nicht speichern", "mandant", tc.ID, "fehler", err)
	}
	h.activate(t)
	return t, nil
}

// CreateTenant legt einen Mandanten in einem leeren Ordner an und öffnet ihn.
// Scheitert ein Schritt, bleiben Ordner, Liste und offener Mandant, wie sie
// waren.
func (h *Host) CreateTenant(
	ctx context.Context, name, dataDir string, settings domain.CompanySettings,
) (*domain.TenantConfig, error) {
	if err := requireApp(ctx); err != nil {
		return nil, err
	}
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()

	if name == "" {
		name = settings.CompanyName
	}
	if name == "" {
		name = "Neuer Mandant"
	}
	id := newTenantID()
	if dataDir == "" {
		dataDir = filepath.Join(h.opts.BaseDir, "tenants", id, "data")
	}
	// Erneut geprüft: seit der Ordnerauswahl können Dateien hinzugekommen sein.
	dataDir, err := CheckTenantDirectory(dataDir)
	if err != nil {
		return nil, err
	}
	thisYear := time.Now().Year()
	year := settings.FiscalYear
	if year == 0 {
		year = thisYear
	}
	if year < 1900 || year > thisYear+1 {
		return nil, fmt.Errorf("bitte ein Geschäftsjahr zwischen 1900 und %d wählen", thisYear+1)
	}
	if settings.CompanyName == "" {
		settings.CompanyName = name
	}
	settings.FiscalYear = year

	complete := false
	defer func() {
		if !complete {
			// Der Ordner war leer oder fehlte; angelegt hat ihn dieser Vorgang.
			_ = os.RemoveAll(dataDir)
		}
	}()

	// Das Profil steht, bevor Dienste oder Geschäftsjahre entstehen: eine
	// neue Datenbank legt sonst das laufende Kalenderjahr an.
	db, err := repository.InitTenantDB(dataDir)
	if err != nil {
		return nil, err
	}
	err = service.NewSettingsService(repository.NewSettingsRepository(db), repository.NewAuditRepository(db)).
		UpdateCompanySettings(ctx, &settings)
	closeErr := repository.CloseDB(db)
	if err != nil {
		return nil, fmt.Errorf("Unternehmensdaten speichern: %w", err)
	}
	if closeErr != nil {
		return nil, closeErr
	}

	tc := domain.TenantConfig{ID: id, Name: name, DataDir: dataDir, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	t, err := h.open(ctx, tc, year)
	if err != nil {
		return nil, fmt.Errorf("Buchhaltung einrichten: %w", err)
	}
	if _, err := h.updateConfig(func(cfg *domain.AppConfig) error {
		cfg.Tenants = append(cfg.Tenants, tc)
		cfg.ActiveTenantID = id
		cfg.DataDir = dataDir
		cfg.IsConfigured = true
		cfg.LastFiscalYear = year
		return nil
	}); err != nil {
		_ = t.Close(ctx)
		return nil, fmt.Errorf("Einrichtung speichern: %w", err)
	}
	h.activate(t)
	complete = true
	return &tc, nil
}

// RegisterTenant meldet einen vorhandenen Datenordner unter einer neuen
// Kennung an, öffnet ihn und protokolliert die Übernahme. name ist der
// Anzeigename, den der Aufrufer schon kennt; leer kommt er aus den
// Unternehmensdaten oder dem Ordner.
func (h *Host) RegisterTenant(
	ctx context.Context, dbFile, name string, kind domain.MigrationKind,
) (*domain.TenantConfig, error) {
	if err := requireApp(ctx); err != nil {
		return nil, err
	}
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()
	tc, _, err := h.register(ctx, dbFile, name, kind)
	return tc, err
}

// register ist RegisterTenant ohne Sperre. Der Aufrufer hält lifecycle.
func (h *Host) register(
	ctx context.Context, dbFile, name string, kind domain.MigrationKind,
) (*domain.TenantConfig, *Tenant, error) {
	if _, err := os.Stat(dbFile); err != nil {
		return nil, nil, fmt.Errorf("die Datenbankdatei %s wurde nicht gefunden", dbFile)
	}
	dataDir := filepath.Dir(dbFile)
	// Absolut: ein relativer Pfad bräche, sobald das Arbeitsverzeichnis wechselt.
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}
	if name == "" {
		name = companyNameAt(dbFile)
	}
	if name == "" {
		name = filepath.Base(dataDir)
		if name == "data" || name == "." || name == "" {
			name = fmt.Sprintf("Mandant (%s)", filepath.Base(dbFile))
		}
	}

	tc := domain.TenantConfig{
		ID: newTenantID(), Name: name, DataDir: dataDir,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	t, err := h.open(ctx, tc, 0)
	if err != nil {
		return nil, nil, err
	}
	if _, err := h.updateConfig(func(cfg *domain.AppConfig) error {
		cfg.Tenants = append(cfg.Tenants, tc)
		cfg.ActiveTenantID = tc.ID
		cfg.DataDir = dataDir
		cfg.IsConfigured = true
		cfg.LastFiscalYear = t.FiscalYear()
		return nil
	}); err != nil {
		_ = t.Close(ctx)
		return nil, nil, err
	}
	h.activate(t)
	h.recordMigration(ctx, t, kind, dbFile)
	return &tc, t, nil
}

// companyNameAt liest den Firmennamen aus einer Mandantendatei, ohne sie zu
// verändern. Leer, wo er sich nicht lesen lässt.
func companyNameAt(dbFile string) string {
	db, err := repository.OpenReadOnlyDB(dbFile)
	if err != nil {
		return ""
	}
	defer func() { _ = repository.CloseDB(db) }()
	name, err := repository.NewSettingsRepository(db).Get(context.Background(), "company_name")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

// recordMigration zählt die übernommenen Objekte, prüft die Ketten und
// schreibt das Migrationsprotokoll (ARC-05). Jeder Weg herein geht durch
// register, deshalb wird nur hier protokolliert.
//
// Der Vorgang hält die Übernahme nicht auf: die Daten sind da, und ein
// gescheitertes Protokoll wäre kein Grund, sie wieder wegzunehmen.
func (h *Host) recordMigration(ctx context.Context, t *Tenant, kind domain.MigrationKind, source string) {
	err := t.Write(ctx, func(ctx context.Context, s *Services) error {
		counts, err := repository.CountForMigration(ctx, t.db)
		if err != nil {
			return s.Repos.Audit.Log(ctx, domain.AuditActionImport, "MIGRATION", t.ID(),
				fmt.Sprintf("Die Zählung der übernommenen Daten ist fehlgeschlagen: %v", err))
		}
		rec := &domain.MigrationRecord{Kind: kind, Source: source, Target: t.ID(), RunAt: time.Now().UTC()}
		rec.SetCounts(counts)
		// Die Kette wird sofort geprüft: eine Buchhaltung, die schon beim
		// Ankommen gebrochen ist, soll das sagen und nicht später so aussehen,
		// als sei sie hier gebrochen worden.
		if result, err := s.Journal.VerifyIntegrity(ctx); err == nil {
			rec.ChainValid = result.IsValid
			rec.ChainMessage = result.Message
		}
		if err := s.Repos.Migrations.CreateMigrationRecord(ctx, rec); err != nil {
			return err
		}
		balance := "Soll und Haben stimmen überein"
		if !counts.IsBalanced() {
			balance = fmt.Sprintf("Soll %s € und Haben %s € stimmen NICHT überein — die Übernahme ist unvollständig",
				counts.DebitTotal, counts.CreditTotal)
		}
		return s.Repos.Audit.Log(ctx, domain.AuditActionImport, "MIGRATION", t.ID(), fmt.Sprintf(
			"Datenübernahme (%s) aus %s: %d Buchungen mit %d Zeilen, %d Belege, %d Geschäftspartner, %d Konten, %d Rechnungen, %d Anlagegüter, %d Protokolleinträge. %s. Kette: %s",
			kind, filepath.Base(source), counts.JournalEntries, counts.JournalLines,
			counts.Receipts, counts.Contacts, counts.Accounts, counts.Invoices,
			counts.FixedAssets, counts.AuditEntries, balance, rec.ChainMessage))
	}, ExaminerExempt())
	if err != nil {
		h.logger.Warn("Übernahme ließ sich nicht protokollieren", "mandant", t.ID(), "fehler", err)
	}
}

// RemoveTenant nimmt einen Mandanten aus der Liste; die Daten bleiben liegen.
// Ein Mandant im Prüfermodus bleibt: mit dem Eintrag ginge die Sperre, und
// der Ordner ließe sich ohne sie wieder öffnen.
func (h *Host) RemoveTenant(ctx context.Context, tenantID string) error {
	if err := requireApp(ctx); err != nil {
		return err
	}
	h.lifecycle.Lock()
	defer h.lifecycle.Unlock()

	if err := examinerError(h.examinerOf(tenantID)); err != nil {
		return err
	}
	h.mu.RLock()
	wasOpen := h.current != nil && h.current.ID() == tenantID
	h.mu.RUnlock()

	cfg, err := h.updateConfig(func(cfg *domain.AppConfig) error {
		kept := make([]domain.TenantConfig, 0, len(cfg.Tenants))
		for _, t := range cfg.Tenants {
			if t.ID != tenantID {
				kept = append(kept, t)
			}
		}
		cfg.Tenants = kept
		if cfg.ActiveTenantID == tenantID {
			cfg.ActiveTenantID = ""
			if len(kept) > 0 {
				cfg.ActiveTenantID = kept[0].ID
			} else {
				cfg.IsConfigured = false
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !wasOpen {
		return nil
	}
	if next := cfg.ActiveTenant(); next != nil {
		if t, err := h.open(ctx, *next, 0); err == nil {
			h.activate(t)
			return nil
		} else {
			h.logger.Warn("nächster Mandant ließ sich nicht öffnen", "mandant", next.ID, "fehler", err)
		}
	}
	h.activate(nil)
	return nil
}

// newTenantID vergibt eine Kennung für einen neuen Eintrag der Mandantenliste.
func newTenantID() string {
	return fmt.Sprintf("tenant_%d", time.Now().UnixNano())
}

// CheckTenantDirectory prüft den Speicherort eines neuen Mandanten, ohne etwas
// anzulegen, und liefert ihn absolut. Er muss leer sein oder fehlen.
func CheckTenantDirectory(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", fmt.Errorf("bitte einen Ordner für die Buchungsdaten und Belege wählen")
	}
	if dataDir == "~" || strings.HasPrefix(dataDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("das Benutzerverzeichnis kann nicht ermittelt werden: %w", err)
		}
		dataDir = filepath.Join(homeDir, strings.TrimPrefix(dataDir, "~"))
	}
	absolute, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("der Datenordner kann nicht aufgelöst werden: %w", err)
	}
	entries, err := os.ReadDir(absolute)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("der Datenordner kann nicht gelesen werden: %w", err)
	}
	if len(entries) != 0 {
		return "", fmt.Errorf("der Datenordner ist nicht leer. Wählen Sie einen neuen, leeren Ordner oder öffnen Sie die vorhandene Buchhaltung")
	}
	return absolute, nil
}

func today() string { return time.Now().Format("2006-01-02") }
