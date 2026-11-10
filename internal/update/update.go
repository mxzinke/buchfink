// Package update hält Buchfink aktuell: es fragt GitHub nach dem neuesten
// Release, lädt die Datei für dieses Betriebssystem, prüft sie gegen die
// Prüfsummen des Release und setzt sie an die Stelle der laufenden Fassung.
//
// Gesucht wird von selbst, installiert nur auf Anweisung des Menschen in der
// App: die Programmfassung steht an jeder Buchung und in jedem Export
// (internal/buildinfo), ein Wechsel soll kein Nebeneffekt sein. Ein
// Arbeitsstand ohne Fassung („dev") sucht und installiert nichts.
package update

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Phase ist der Stand der Aktualisierung.
type Phase string

const (
	// PhaseIdle: nichts gesucht oder die laufende Fassung ist die neueste.
	PhaseIdle        Phase = "idle"
	PhaseChecking    Phase = "checking"
	PhaseAvailable   Phase = "available"
	PhaseDownloading Phase = "downloading"
	// PhaseInstalled: die neue Fassung liegt an ihrem Platz und läuft nach dem
	// nächsten Start.
	PhaseInstalled Phase = "installed"
)

// State ist, was die Oberfläche über die Aktualisierung zeigt.
type State struct {
	Current string `json:"current"`
	// Supported ist false bei einem Arbeitsstand ohne Fassung.
	Supported bool     `json:"supported"`
	Phase     Phase    `json:"phase"`
	Release   *Release `json:"release,omitempty"`
	CheckedAt string   `json:"checkedAt,omitempty"`
	// Error ist der Grund, aus dem die letzte Suche oder Installation
	// scheiterte.
	Error string `json:"error,omitempty"`
	// Progress ist der Anteil des Downloads in Prozent.
	Progress int `json:"progress"`
	// ManualReason sagt, warum Buchfink die Fassung nicht selbst installieren
	// kann; der Mensch lädt sie dann von der Seite des Release.
	ManualReason string `json:"manualReason,omitempty"`
}

// Service sucht nach neuen Fassungen und installiert sie.
type Service struct {
	current      string
	goos, goarch string
	endpoint     string
	client       *http.Client
	locate       func() (Installation, error)
	now          func() time.Time
	logger       *slog.Logger

	// busy ordnet Suche und Installation: es läuft immer nur eine.
	busy  sync.Mutex
	mu    sync.Mutex
	state State
}

// New richtet die Aktualisierung für die laufende Fassung ein.
func New(current string, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	_, supported := parseVersion(current)
	return &Service{
		current: current, goos: runtime.GOOS, goarch: runtime.GOARCH,
		endpoint: LatestReleaseURL, client: newClient(),
		locate: func() (Installation, error) { return Locate(runtime.GOOS) },
		now:    time.Now, logger: logger,
		state: State{Current: current, Supported: supported, Phase: PhaseIdle},
	}
}

// State liefert den Stand.
func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Service) set(change func(*State)) State {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(&s.state)
	return s.state
}

// CleanupLeftovers entfernt beim Start, was ein früheres Update neben dem
// Programm zurückgelassen hat.
func (s *Service) CleanupLeftovers() {
	if !s.State().Supported {
		return
	}
	if installation, err := s.locate(); err == nil {
		installation.Cleanup()
	}
}

// Check fragt nach dem neuesten Release. Eine installierte, noch nicht
// gestartete Fassung bleibt stehen.
func (s *Service) Check(ctx context.Context) State {
	if st := s.State(); !st.Supported || st.Phase == PhaseInstalled {
		return st
	}
	if !s.busy.TryLock() {
		return s.State()
	}
	defer s.busy.Unlock()

	s.set(func(st *State) { st.Phase, st.Error = PhaseChecking, "" })
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	release, err := s.latest(ctx)
	checkedAt := s.now().UTC().Format(time.RFC3339)
	if err != nil {
		s.logger.Info("Suche nach einer neuen Fassung gescheitert", "fehler", err)
		return s.set(func(st *State) {
			st.Phase, st.Error, st.CheckedAt = PhaseIdle, err.Error(), checkedAt
			if st.Release != nil {
				st.Phase = PhaseAvailable
			}
		})
	}
	current, _ := parseVersion(s.current)
	found, _ := parseVersion(release.Version)
	if !found.newer(current) {
		return s.set(func(st *State) {
			st.Phase, st.Release, st.CheckedAt, st.ManualReason = PhaseIdle, nil, checkedAt, ""
		})
	}
	reason := s.manualReason(release)
	return s.set(func(st *State) {
		st.Phase, st.Release, st.CheckedAt, st.ManualReason = PhaseAvailable, release, checkedAt, reason
	})
}

// manualReason nennt, was eine Installation durch Buchfink verhindert.
func (s *Service) manualReason(release *Release) string {
	if release.asset.url == "" {
		return fmt.Sprintf("Für dieses System (%s/%s) enthält das Release keine Datei.", s.goos, s.goarch)
	}
	installation, err := s.locate()
	if err != nil {
		return err.Error()
	}
	if err := installation.checkWritable(); err != nil {
		return err.Error() + ". Ersetzen Sie das Programm dort von Hand."
	}
	return ""
}

// Install lädt die gefundene Fassung, prüft sie und setzt sie an die Stelle
// der laufenden. Danach läuft sie mit dem nächsten Start.
func (s *Service) Install(ctx context.Context) (State, error) {
	if !s.busy.TryLock() {
		return s.State(), fmt.Errorf("eine Aktualisierung läuft bereits")
	}
	defer s.busy.Unlock()

	st := s.State()
	if st.Phase != PhaseAvailable || st.Release == nil {
		return st, fmt.Errorf("es liegt keine neue Fassung zur Installation vor")
	}
	if st.ManualReason != "" {
		return st, fmt.Errorf("%s", st.ManualReason)
	}
	release := st.Release
	s.set(func(st *State) { st.Phase, st.Progress, st.Error = PhaseDownloading, 0, "" })

	err := s.install(ctx, release)
	if err != nil {
		s.logger.Warn("Aktualisierung gescheitert", "fassung", release.Version, "fehler", err)
		return s.set(func(st *State) { st.Phase, st.Error = PhaseAvailable, err.Error() }), err
	}
	s.logger.Info("Aktualisierung installiert", "fassung", release.Version)
	return s.set(func(st *State) { st.Phase, st.Progress = PhaseInstalled, 100 }), nil
}

func (s *Service) install(ctx context.Context, release *Release) error {
	installation, err := s.locate()
	if err != nil {
		return err
	}
	installation.Cleanup()
	defer installation.removeStages()
	archive, err := s.download(ctx, release, installation.dir(), func(done int64) {
		if release.asset.size <= 0 {
			return
		}
		percent := int(done * 100 / release.asset.size)
		s.set(func(st *State) { st.Progress = min(percent, 99) })
	})
	if err != nil {
		return err
	}
	staged, err := installation.stage(archive, release.asset.name)
	if err != nil {
		return err
	}
	return installation.swap(staged)
}

// checkTimeout begrenzt eine Suche.
const checkTimeout = 30 * time.Second

// checkInterval ist der Abstand zweier Suchen, solange die App läuft.
const checkInterval = 24 * time.Hour

// startDelay lässt den Start der App vorbei, bevor zum ersten Mal gesucht
// wird.
const startDelay = 20 * time.Second

// Run sucht kurz nach dem Start und danach täglich, solange enabled es
// erlaubt, bis ctx endet.
func (s *Service) Run(ctx context.Context, enabled func() bool) {
	if !s.State().Supported {
		return
	}
	timer := time.NewTimer(startDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if enabled() {
			s.Check(ctx)
		}
		timer.Reset(checkInterval)
	}
}
