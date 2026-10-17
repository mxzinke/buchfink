package core

import (
	"errors"

	"github.com/buchfink/buchfink/internal/actor"
)

var (
	// ErrNoTenant: es ist kein Mandant geöffnet.
	ErrNoTenant = errors.New("es ist kein Mandant geöffnet")
	// ErrTenantClosing: der Mandant wird gerade geschlossen, etwa nach einem
	// Wechsel in der App.
	ErrTenantClosing = errors.New("der Mandant wird gerade geschlossen; bitte den Vorgang wiederholen")
	// ErrNoChannel: ein Einstieg in den Kern hat keinen Erfassungsweg gesetzt.
	ErrNoChannel = errors.New("der Vorgang nennt keinen Erfassungsweg (ui, system oder agent)")
	// ErrUnknownChannel: der Erfassungsweg ist keiner der bekannten.
	ErrUnknownChannel = errors.New("unbekannter Erfassungsweg")
	// ErrAppOnly: der Vorgang ist der Bedienung in der App vorbehalten.
	ErrAppOnly = actor.ErrAppOnly
	// ErrForeignTenant: ein Vorgang kam mit der Transaktion eines anderen
	// Mandanten.
	ErrForeignTenant = errors.New("der Vorgang gehört zu einem anderen Mandanten")
	// ErrHostClosed: die App wird beendet.
	ErrHostClosed = errors.New("Buchfink wird beendet")
	// ErrExaminerActive: der Prüfermodus weist den Schreibvorgang ab.
	ErrExaminerActive = errors.New("der Prüfermodus ist aktiv")
	// ErrWriteInRead: ein Lesevorgang wollte schreiben.
	ErrWriteInRead = errors.New("ein Lesevorgang darf nicht schreiben")
)
