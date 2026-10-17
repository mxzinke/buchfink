package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/actor"
)

// requireHuman behält einen Vorgang dem Menschen in der App vor:
// Festschreibung, Übermittlungsbestätigung, Aufstellen/Feststellen/Offenlegen,
// Stammdaten des Unternehmens samt Gründung, Bankkonten und Organisation.
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
		return fmt.Errorf("%s ist der Bedienung in der App vorbehalten; der verbundene Agent %q kann das nicht",
			what, channel.AgentSlug())
	}
	return fmt.Errorf("%s ist der Bedienung in der App vorbehalten, nicht dem Erfassungsweg %q", what, channel)
}
