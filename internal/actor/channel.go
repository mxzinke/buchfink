package actor

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrAppOnly: der Vorgang ist der Bedienung in der App (Erfassungsweg ui)
// vorbehalten. Kern und Dienste melden ihn unter diesem Fehler, damit der
// MCP-Server ihn als solchen erkennt.
var ErrAppOnly = errors.New("dieser Vorgang ist der Bedienung in der App vorbehalten")

// Channel ist der Erfassungsweg: über welche Bedienung ein Vorgang in die
// Bücher kam. Der Bearbeiter (Actor) sagt, wer verantwortet; der Erfassungsweg,
// ob er selbst in der App gehandelt hat, ein Hintergrundlauf des Programms oder
// ein verbundener Agent.
//
// Ein Agent trägt seinen Kurznamen im Weg (`agent:<kurzname>`). Der Kurzname
// geht in die Hash-Ketten ein und wird deshalb nach der ersten Zustimmung nie
// geändert.
type Channel string

const (
	// ChannelUI ist die Bedienung in der App.
	ChannelUI Channel = "ui"
	// ChannelSystem ist ein Lauf des Programms ohne Bedienung: Sicherung,
	// nachgeholter Zeitstempel, Einrichtung eines Mandanten.
	ChannelSystem Channel = "system"

	agentPrefix  = "agent:"
	maxSlugBytes = 40
)

// Agent bildet den Erfassungsweg eines verbundenen Agenten.
func Agent(slug string) (Channel, error) {
	if err := checkSlug(slug); err != nil {
		return "", err
	}
	return Channel(agentPrefix + slug), nil
}

// ParseChannel prüft einen gespeicherten oder übergebenen Erfassungsweg.
func ParseChannel(s string) (Channel, error) {
	switch c := Channel(s); {
	case c == ChannelUI, c == ChannelSystem:
		return c, nil
	case strings.HasPrefix(s, agentPrefix):
		return Agent(strings.TrimPrefix(s, agentPrefix))
	default:
		return "", fmt.Errorf("unbekannter Erfassungsweg %q (erwartet ui, system oder agent:<kurzname>)", s)
	}
}

// IsAgent meldet, ob ein verbundener Agent gehandelt hat.
func (c Channel) IsAgent() bool { return strings.HasPrefix(string(c), agentPrefix) }

// AgentSlug ist der Kurzname des Agenten, leer bei anderen Wegen.
func (c Channel) AgentSlug() string {
	if !c.IsAgent() {
		return ""
	}
	return strings.TrimPrefix(string(c), agentPrefix)
}

// Kurznamen bestehen aus Kleinbuchstaben, Ziffern und Bindestrichen. Sie
// stehen in Export und Protokoll und sollen dort ohne Maskierung lesbar sein.
func checkSlug(slug string) error {
	if slug == "" || len(slug) > maxSlugBytes {
		return fmt.Errorf("der Kurzname eines Agenten hat 1 bis %d Zeichen, nicht %d", maxSlugBytes, len(slug))
	}
	for _, r := range slug {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return fmt.Errorf("der Kurzname %q enthält %q; erlaubt sind a–z, 0–9 und -", slug, r)
		}
	}
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return fmt.Errorf("der Kurzname %q beginnt oder endet mit einem Bindestrich", slug)
	}
	return nil
}

type channelKey struct{}

// WithChannel legt den Erfassungsweg in den Kontext. Jeder Einstieg in den
// Kern setzt ihn: die Bridge `ui`, Hintergrundläufe `system`, der MCP-Server
// den Weg des Agenten.
func WithChannel(ctx context.Context, c Channel) context.Context {
	return context.WithValue(ctx, channelKey{}, c)
}

// ChannelOf liest den Erfassungsweg aus dem Kontext.
func ChannelOf(ctx context.Context) (Channel, bool) {
	c, ok := ctx.Value(channelKey{}).(Channel)
	return c, ok && c != ""
}

// RecordedChannel ist der Weg, der an Buchung und Protokolleintrag steht.
// Ohne Angabe im Kontext ist es `system`: der Kern verlangt den Weg an jedem
// Einstieg, ohne ihn schreiben nur Programmteile unterhalb des Kerns, etwa das
// Anlegen einer Datenbank.
func RecordedChannel(ctx context.Context) Channel {
	if c, ok := ChannelOf(ctx); ok {
		return c
	}
	return ChannelSystem
}
