package wailsbridge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/changelog"
	"github.com/buchfink/buchfink/internal/update"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// updater ist die Aktualisierung des Programms. Sie gehört zum Rechner und
// nicht zu einem Mandanten; ein Werkzeug für Agenten gibt es dafür nicht.
type updater struct {
	once    sync.Once
	service *update.Service
}

func (b *BuchfinkBridge) updateService() *update.Service {
	b.updates.once.Do(func() { b.updates.service = update.New(buildinfo.Version, nil) })
	return b.updates.service
}

// startUpdates räumt die Reste einer früheren Aktualisierung weg und sucht im
// Hintergrund nach neuen Fassungen, solange die Suche eingeschaltet ist.
func (b *BuchfinkBridge) startUpdates() {
	service := b.updateService()
	service.CleanupLeftovers()
	go service.Run(context.Background(), func() bool { return !b.host.Config().UpdateCheckOff })
}

// UpdateStatus ist der Stand der Aktualisierung mit der Einstellung, ob
// Buchfink von selbst sucht.
type UpdateStatus struct {
	update.State
	AutoCheck bool `json:"autoCheck"`
	// ReleaseNotes ist der Text des Release in den Teilen der
	// Änderungshistorie.
	ReleaseNotes *changelog.Entry `json:"releaseNotes,omitempty"`
}

func (b *BuchfinkBridge) updateStatus(state update.State) UpdateStatus {
	status := UpdateStatus{State: state, AutoCheck: !b.host.Config().UpdateCheckOff}
	if release := state.Release; release != nil {
		date, _, _ := strings.Cut(release.PublishedAt, "T")
		notes := changelog.EntryOf("v"+release.Version, date, release.Notes)
		status.ReleaseNotes = &notes
	}
	return status
}

// GetUpdateStatus liefert den Stand der Aktualisierung, ohne zu suchen.
func (b *BuchfinkBridge) GetUpdateStatus() UpdateStatus {
	return b.updateStatus(b.updateService().State())
}

// CheckForUpdate fragt GitHub nach dem neuesten Release.
func (b *BuchfinkBridge) CheckForUpdate() UpdateStatus {
	return b.updateStatus(b.updateService().Check(context.Background()))
}

// SetUpdateCheck schaltet die Suche nach neuen Fassungen beim Start und
// danach täglich ein oder aus.
func (b *BuchfinkBridge) SetUpdateCheck(enabled bool) (UpdateStatus, error) {
	err := b.host.SetUpdateCheck(uiContext(), enabled)
	return b.GetUpdateStatus(), err
}

// InstallUpdate lädt die gefundene Fassung, prüft sie gegen die Prüfsummen
// des Release und setzt sie an die Stelle der laufenden. Sie läuft nach dem
// nächsten Start (RestartAfterUpdate).
func (b *BuchfinkBridge) InstallUpdate() (UpdateStatus, error) {
	state, err := b.updateService().Install(context.Background())
	return b.updateStatus(state), err
}

// RestartAfterUpdate startet die installierte Fassung und beendet die
// laufende App. Beim Beenden sichert der Kern wie bei jedem Schließen
// (ServiceShutdown).
func (b *BuchfinkBridge) RestartAfterUpdate() error {
	app := application.Get()
	if app == nil {
		return fmt.Errorf("die App lässt sich von hier aus nicht neu starten")
	}
	if err := b.updateService().Relaunch(); err != nil {
		return err
	}
	app.Quit()
	return nil
}
