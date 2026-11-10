package actor

import (
	"context"
	"testing"
)

func TestChannelRoundTrip(t *testing.T) {
	agent, err := Agent("claude-desktop")
	if err != nil {
		t.Fatalf("Kurzname abgewiesen: %v", err)
	}
	if !agent.IsAgent() || agent.AgentSlug() != "claude-desktop" {
		t.Errorf("Weg %q wird nicht als Agent erkannt", agent)
	}
	for _, c := range []Channel{ChannelUI, ChannelSystem, agent} {
		parsed, err := ParseChannel(string(c))
		if err != nil || parsed != c {
			t.Errorf("%q: gelesen %q, Fehler %v", c, parsed, err)
		}
	}
	if ChannelUI.IsAgent() || ChannelSystem.IsAgent() {
		t.Error("ui und system sind keine Agenten")
	}
}

// Der Kurzname steht in Hash-Kette, Export und Protokoll. Was dort nicht ohne
// Maskierung lesbar ist, wird abgewiesen.
func TestAgentSlugIsRestricted(t *testing.T) {
	for _, slug := range []string{"", "Claude", "a b", "agent:x", "-x", "x-", "ä", string(make([]byte, 41))} {
		if _, err := Agent(slug); err == nil {
			t.Errorf("Kurzname %q angenommen", slug)
		}
	}
	for _, s := range []string{"", "UI", "agent:", "mensch"} {
		if _, err := ParseChannel(s); err == nil {
			t.Errorf("Erfassungsweg %q angenommen", s)
		}
	}
}

func TestChannelInContext(t *testing.T) {
	ctx := context.Background()
	if _, ok := ChannelOf(ctx); ok {
		t.Error("ein leerer Kontext hat keinen Erfassungsweg")
	}
	if got := RecordedChannel(ctx); got != ChannelSystem {
		t.Errorf("ohne Angabe steht %q, erwartet system", got)
	}
	ctx = WithChannel(ctx, ChannelUI)
	if got, ok := ChannelOf(ctx); !ok || got != ChannelUI {
		t.Errorf("gelesen %q (%v), erwartet ui", got, ok)
	}
	if got := RecordedChannel(ctx); got != ChannelUI {
		t.Errorf("an der Buchung stünde %q, erwartet ui", got)
	}
}
