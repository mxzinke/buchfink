package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/actor"
)

// requireHuman behält einen Vorgang dem Menschen in der App vor.
// internal/ops ordnet die Vorgänge ein; Tests gleichen die Schutzstellen ab.
//
// Zugelassen ist nur der Erfassungsweg ui. Ohne Erfassungsweg läuft der Dienst
// unterhalb des Kerns (Einrichtung eines Mandanten, Tests); der Kern lässt
// keinen Vorgang ohne Weg herein. Der Kern behält dieselben Vorgänge
// zusätzlich der App vor (core.AppOnly).
func requireHuman(ctx context.Context, what string) error {
	channel, ok := actor.ChannelOf(ctx)
	if !ok || channel == actor.ChannelUI {
		return nil
	}
	if channel.IsAgent() {
		return appOnlyError(fmt.Sprintf("%s ist der Bedienung in der App vorbehalten; der verbundene Agent %q kann das nicht",
			what, channel.AgentSlug()))
	}
	return appOnlyError(fmt.Sprintf("%s ist der Bedienung in der App vorbehalten, nicht dem Erfassungsweg %q", what, channel))
}

// appOnlyError nennt den Vorgang und ist zugleich actor.ErrAppOnly.
type appOnlyError string

func (e appOnlyError) Error() string { return string(e) }
func (e appOnlyError) Unwrap() error { return actor.ErrAppOnly }
