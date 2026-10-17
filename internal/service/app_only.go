package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/actor"
)

// refuseAgent weist einen Vorgang ab, den nur ein Mensch in der App auslöst:
// Übermittlungsbestätigung, Aufstellen/Feststellen/Offenlegen, Stammdaten des
// Unternehmens und seine Bankkonten. Der Kern behält sie zusätzlich der App vor
// (core.AppOnly); diese Prüfung hält auch dann, wenn ein Aufrufer das vergisst.
func refuseAgent(ctx context.Context, what string) error {
	if channel := actor.RecordedChannel(ctx); channel.IsAgent() {
		return fmt.Errorf("%s ist der Bedienung in der App vorbehalten; der verbundene Agent %q kann das nicht",
			what, channel.AgentSlug())
	}
	return nil
}
