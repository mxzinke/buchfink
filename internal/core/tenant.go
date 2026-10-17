// Package core ist der Buchungskern hinter Oberfläche und MCP-Server: ein
// offener Mandant mit genau einem Satz Dienste, eine Schreibsperre je
// Mandant und eine Transaktion je Vorgang. Das Paket kennt kein Wails.
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/repository"
	"gorm.io/gorm"
)

// closeTimeout ist die längste Wartezeit beim Schließen auf laufende
// Vorgänge. Danach wird trotzdem geschlossen; ein hängender Aufrufer bekommt
// dann einen Datenbankfehler statt eines offenen Pools für immer.
var closeTimeout = 60 * time.Second

// Options beschreibt einen zu öffnenden Mandanten.
type Options struct {
	TenantID   string
	TenantName string
	// Generation zählt die Mandantenwechsel des Hosts. Ein Aufrufer, der sich
	// eine Generation merkt, erkennt daran einen Wechsel.
	Generation uint64
	// Renderer setzt Rechnungen und Abschlüsse. Ohne Angabe bekommt der
	// Mandant einen eigenen, der mit ihm geschlossen wird.
	Renderer *invoice.Renderer
	// Examiner liefert den Stand des Prüfermodus. Ohne Angabe gilt er nie.
	Examiner func() ExaminerState
	// BackupDir liefert den eingerichteten Sicherungsordner.
	BackupDir func() string
	// Rates holt Referenzkurse. Ohne Angabe die EZB.
	Rates  domain.CurrencyFetcher
	Logger *slog.Logger
}

func (o Options) backupDir() string {
	if o.BackupDir == nil {
		return ""
	}
	return o.BackupDir()
}

// ExaminerState ist der Stand des Prüfermodus eines Mandanten.
type ExaminerState struct {
	Active bool
	Until  string
	Reason string
}

// Tenant ist ein offener Mandant.
type Tenant struct {
	info        TenantInfo
	gen         uint64
	db          *gorm.DB
	reader      *gorm.DB
	tx          domain.TxRunner
	svc         *Services
	renderer    *invoice.Renderer
	ownRenderer bool
	examiner    func() ExaminerState
	backupDir   func() string
	logger      *slog.Logger

	// startYear ist das Geschäftsjahr, mit dem die Ansicht nach dem Öffnen
	// beginnt. Die Dienste kennen es nicht.
	startYear int

	write    writeLock
	leases   leases
	backupMu sync.Mutex

	closeOnce sync.Once
	closeErr  error
}

// Open öffnet die Mandantendatei im Datenordner, verdrahtet die Dienste und
// legt fehlende Geschäftsjahre an.
func Open(ctx context.Context, dataDir string, o Options) (*Tenant, error) {
	db, err := repository.InitTenantDB(dataDir)
	if err != nil {
		return nil, fmt.Errorf("die Mandantendatenbank in %s ließ sich nicht öffnen: %w", dataDir, err)
	}
	reader, err := repository.OpenTenantReader(dataDir)
	if err != nil {
		_ = repository.CloseDB(db)
		return nil, err
	}

	logger := o.Logger
	if logger == nil {
		logger = slog.Default()
	}
	renderer, own := o.Renderer, false
	if renderer == nil {
		renderer, own = invoice.NewRenderer(), true
	}
	info := TenantInfo{ID: o.TenantID, Name: o.TenantName, DataDir: dataDir}

	t := &Tenant{
		info: info, gen: o.Generation,
		db: db, reader: reader, tx: repository.NewTxRunner(db),
		svc:      wire(db, info, renderer, o),
		renderer: renderer, ownRenderer: own,
		examiner: o.Examiner, backupDir: o.BackupDir, logger: logger,
		write: newWriteLock(),
	}

	t.svc.Backup.SetRecorder(t.recordInWrite)
	t.startYear = startYear(ctx, repository.NewSettingsRepository(db), time.Now())

	// Das Geschäftsjahr der Unternehmensdaten besteht auch ohne Buchung.
	// Scheitert das Anlegen, bleibt der Mandant benutzbar; die Ansicht
	// „Jahresabschluss" legt das Jahr beim ersten Aufruf an.
	system := actor.WithChannel(ctx, actor.ChannelSystem)
	if err := t.Write(system, func(ctx context.Context, s *Services) error {
		return s.Closing.EnsureFiscalYears(ctx, t.startYear)
	}, ExaminerExempt()); err != nil {
		logger.Warn("Geschäftsjahre konnten nicht angelegt werden", "mandant", info.ID, "fehler", err)
	}
	return t, nil
}

// startYear ist das Geschäftsjahr der Unternehmensdaten, sonst das des
// heutigen Tages nach Beginnmonat.
func startYear(ctx context.Context, settings domain.SettingsRepository, now time.Time) int {
	if s, err := settings.GetCompanySettings(ctx); err == nil && s != nil {
		if s.FiscalYear > 0 {
			return s.FiscalYear
		}
		return domain.GetFiscalYearForDate(now.Format("2006-01-02"), s.FiscalYearStartMonth)
	}
	return now.Year()
}

// StartYear ist das Geschäftsjahr, mit dem die Ansicht nach dem Öffnen
// beginnt: das der Unternehmensdaten, sonst das des heutigen Tages.
func (t *Tenant) StartYear() int { return t.startYear }

// Close wartet auf laufende Vorgänge und schließt die Datenbank. Danach
// liefert jeder Aufruf ErrTenantClosing.
func (t *Tenant) Close(ctx context.Context) error {
	t.closeOnce.Do(func() {
		if !t.leases.drain(closeTimeout) {
			t.logger.Warn("Mandant wird mit laufenden Vorgängen geschlossen", "mandant", t.info.ID)
		}
		errs := []error{repository.CloseDB(t.reader), repository.CloseDB(t.db)}
		if t.ownRenderer {
			errs = append(errs, t.renderer.Close(ctx))
		}
		t.closeErr = errors.Join(errs...)
	})
	return t.closeErr
}

// ID ist die Kennung des Mandanten in der Mandantenliste.
func (t *Tenant) ID() string { return t.info.ID }

// Name ist der Anzeigename beim Öffnen.
func (t *Tenant) Name() string { return t.info.Name }

// DataDir ist der Datenordner.
func (t *Tenant) DataDir() string { return t.info.DataDir }

// Generation ist der Stand des Mandantenwechsels, unter dem er geöffnet wurde.
func (t *Tenant) Generation() uint64 { return t.gen }

// Read führt fn in einer lesenden Transaktion aus, ohne Schreibsperre.
//
// Alle Repository-Aufrufe in fn sehen denselben Stand. Ein Schreibversuch
// scheitert an der Verbindung; einzig das Änderungsprotokoll schreibt, weil
// ein Zugriff auf personenbezogene Daten auch beim Lesen protokolliert wird.
func (t *Tenant) Read(ctx context.Context, fn func(context.Context, *Services) error) error {
	if _, ok := actor.ChannelOf(ctx); !ok {
		return ErrNoChannel
	}
	if inWriteOf(ctx, t) || repository.IsReadOnly(ctx) {
		return fn(ctx, t.svc)
	}
	release, err := t.leases.acquire()
	if err != nil {
		return err
	}
	defer release()
	return repository.RunReadOnly(ctx, t.reader, func(ctx context.Context) error {
		return fn(ctx, t.svc)
	})
}

// Write führt fn als einen Vorgang aus: unter der Schreibsperre des
// Mandanten, nach der Prüfung des Prüfermodus und in einer Transaktion.
// Scheitert fn, ist nichts geschrieben.
//
// Ein Write im Kontext eines laufenden Write desselben Mandanten gehört zu
// dessen Transaktion.
func (t *Tenant) Write(ctx context.Context, fn func(context.Context, *Services) error, opts ...WriteOption) error {
	var o writeOptions
	for _, opt := range opts {
		opt(&o)
	}
	channel, ok := actor.ChannelOf(ctx)
	if !ok {
		return ErrNoChannel
	}
	if o.appOnly && channel != actor.ChannelUI {
		return ErrAppOnly
	}
	if repository.IsReadOnly(ctx) {
		return ErrWriteInRead
	}
	if inWriteOf(ctx, t) {
		return fn(ctx, t.svc)
	}

	release, err := t.leases.acquire()
	if err != nil {
		return err
	}
	defer release()
	if err := t.write.lock(ctx); err != nil {
		return err
	}
	defer t.write.unlock()

	if !o.examinerExempt {
		if err := t.ensureWritable(); err != nil {
			return err
		}
	}
	ctx = markInWrite(ctx, t)
	if err := t.tx.RunInTx(ctx, func(ctx context.Context) error { return fn(ctx, t.svc) }); err != nil {
		// Der Kontenplan im Speicher kann ein Konto aus der zurückgerollten
		// Transaktion enthalten.
		t.svc.Journal.InvalidateChart()
		return err
	}
	return nil
}

// Examiner liefert den Stand des Prüfermodus.
func (t *Tenant) Examiner() ExaminerState {
	if t.examiner == nil {
		return ExaminerState{}
	}
	return t.examiner()
}

// ensureWritable weist Schreibvorgänge ab, solange der Prüfermodus gilt.
func (t *Tenant) ensureWritable() error { return examinerError(t.Examiner()) }

func examinerError(state ExaminerState) error {
	if !state.Active {
		return nil
	}
	message := fmt.Sprintf("Der Prüfermodus ist bis zum %s aktiv. Bis dahin nimmt die Buchführung keine Änderung auf",
		germanDate(state.Until))
	if strings.TrimSpace(state.Reason) != "" {
		message += fmt.Sprintf(" (Grund: %s)", state.Reason)
	}
	return errors.New(message + ".")
}

// germanDate macht aus 2026-06-30 den 30.06.2026.
func germanDate(iso string) string {
	parsed, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return parsed.Format("02.01.2006")
}
