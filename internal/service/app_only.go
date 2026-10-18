package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
)

// requireHuman behält einen Vorgang dem Menschen in der App vor:
// Festschreibung, Übermittlungsbestätigung, Aufstellen/Feststellen/Offenlegen,
// Beschluss über die Ergebnisverwendung, Stammdaten des Unternehmens samt
// Gründung, Gesellschafterdarlehen, Bankkonten und Organisation.
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

// Konten, die ein Agent nicht von Hand bebucht: sie haben einen Fachweg, und
// eine Handbuchung daran vorbei ließe dessen Folgen aus. Eine Ausschüttung
// per Handbuchung hätte keinen Beschluss, keine Kapitalertragsteuer und keine
// Pflicht zur Anmeldung; ein Gesellschafterdarlehen ohne Vertrag keine
// Zinsprüfung.
func agentGuardedAccount(account string) (string, bool) {
	switch {
	case len(account) == 4 && account >= "2000" && account <= "2999":
		return "Kapital, Rücklagen und Ergebnisvortrag ändern sich nur über Kapitalmaßnahme oder Ergebnisverwendung in der App", true
	case account == domain.AccountAusschuettung || account == domain.AccountKapitalertragsteuer:
		return "Ausschüttung und Kapitalertragsteuer bucht die Ergebnisverwendung in der App", true
	case account == domain.AccountAusleihungenGesellschafter,
		account >= domain.AccountForderungenGesellschafter && account <= domain.AccountForderungenGesellschafterUeber1,
		account >= domain.AccountVerbindlichkeitenGesellschafter && account <= domain.AccountVerbGesellschafterUeber5:
		return "Gesellschafterdarlehen bucht ein Agent nur mit Bezug auf den Darlehensvertrag (bankumsatz_buchen mit darlehen_id); den Vertrag legt der Mensch in der App an", true
	}
	return "", false
}

// requireFachweg weist die Handbuchung eines Agenten auf Konten mit eigenem
// Fachweg ab.
func requireFachweg(ctx context.Context, e *domain.JournalEntry) error {
	channel, ok := actor.ChannelOf(ctx)
	if !ok || !channel.IsAgent() {
		return nil
	}
	for _, l := range e.Lines {
		if why, guarded := agentGuardedAccount(l.Account); guarded {
			return appOnlyError(fmt.Sprintf("Konto %s bucht der verbundene Agent %q nicht von Hand: %s",
				l.Account, channel.AgentSlug(), why))
		}
	}
	return nil
}

// requireFachwegForPayment ist dieselbe Sperre für eine Zeile der Aufteilung
// eines Bankumsatzes. Die Zahlung einer beschlossenen Ausschüttung oder der
// einbehaltenen Kapitalertragsteuer bleibt erlaubt: sie gleicht aus, was die
// Ergebnisverwendung gebucht hat.
func requireFachwegForPayment(ctx context.Context, account string) error {
	channel, ok := actor.ChannelOf(ctx)
	if !ok || !channel.IsAgent() || account == domain.AccountAusschuettung || account == domain.AccountKapitalertragsteuer {
		return nil
	}
	if why, guarded := agentGuardedAccount(account); guarded {
		return appOnlyError(fmt.Sprintf("Konto %s bucht der verbundene Agent %q nicht ohne Fachweg: %s",
			account, channel.AgentSlug(), why))
	}
	return nil
}
