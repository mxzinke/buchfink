package core

import "errors"

var (
	// ErrNoTenant: es ist kein Mandant geöffnet.
	ErrNoTenant = errors.New("es ist kein Mandant geöffnet")
	// ErrTenantClosing: der Mandant wird gerade geschlossen, etwa nach einem
	// Wechsel in der App.
	ErrTenantClosing = errors.New("der Mandant wird gerade geschlossen; bitte den Vorgang wiederholen")
	// ErrNoChannel: ein Einstieg in den Kern hat keinen Erfassungsweg gesetzt.
	ErrNoChannel = errors.New("der Vorgang nennt keinen Erfassungsweg (ui, system oder agent)")
	// ErrAppOnly: der Vorgang ist der Bedienung in der App vorbehalten.
	ErrAppOnly = errors.New("dieser Vorgang ist der Bedienung in der App vorbehalten")
	// ErrWriteInRead: ein Lesevorgang wollte schreiben.
	ErrWriteInRead = errors.New("ein Lesevorgang darf nicht schreiben")
)
