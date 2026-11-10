package wailsbridge

import (
	"slices"
	"sync"
	"time"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventChanged trägt ein ChangeEvent: in einem Mandanten hat sich etwas
// geändert, gleich ob in der App, durch einen Agenten oder im Hintergrund.
// Die Oberfläche lädt dann neu, was sie zeigt.
const EventChanged = "buchfink:geaendert"

// changeInterval ist der kürzeste Abstand zweier Ereignisse: ein Agent, der
// zwanzig Belege ablegt, löst kein Neuladen je Beleg aus.
const changeInterval = 250 * time.Millisecond

// ChangeEvent fasst die Vorgänge eines Mandanten seit dem letzten Ereignis
// zusammen.
type ChangeEvent struct {
	TenantID string `json:"tenantId"`
	// Channels sind die Erfassungswege (ui, system, agent:<kurzname>).
	Channels []string `json:"channels"`
	// Kinds sind die groben Arten: buchungen, belege, bank, stammdaten,
	// abschluss, eingang.
	Kinds []string `json:"kinds"`
}

// changeBatcher bündelt die Meldungen des Kerns und gibt sie höchstens alle
// interval weiter.
type changeBatcher struct {
	interval time.Duration
	emit     func(ChangeEvent)

	mu      sync.Mutex
	order   []string
	pending map[string]*ChangeEvent
	timer   *time.Timer
}

func newChangeBatcher(interval time.Duration, emit func(ChangeEvent)) *changeBatcher {
	return &changeBatcher{interval: interval, emit: emit, pending: map[string]*ChangeEvent{}}
}

// add nimmt eine Meldung des Kerns an; sie blockiert nicht.
func (c *changeBatcher) add(ch core.Change) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ev, ok := c.pending[ch.TenantID]
	if !ok {
		ev = &ChangeEvent{TenantID: ch.TenantID, Channels: []string{}, Kinds: []string{}}
		c.pending[ch.TenantID] = ev
		c.order = append(c.order, ch.TenantID)
	}
	if channel := string(ch.Channel); channel != "" && !slices.Contains(ev.Channels, channel) {
		ev.Channels = append(ev.Channels, channel)
	}
	for _, k := range ch.Kinds {
		if !slices.Contains(ev.Kinds, string(k)) {
			ev.Kinds = append(ev.Kinds, string(k))
		}
	}
	if c.timer == nil {
		c.timer = time.AfterFunc(c.interval, c.flush)
	}
}

func (c *changeBatcher) flush() {
	c.mu.Lock()
	events := make([]ChangeEvent, 0, len(c.order))
	for _, id := range c.order {
		ev := c.pending[id]
		slices.Sort(ev.Channels)
		slices.Sort(ev.Kinds)
		events = append(events, *ev)
	}
	c.order, c.pending, c.timer = nil, map[string]*ChangeEvent{}, nil
	c.mu.Unlock()
	for _, ev := range events {
		c.emit(ev)
	}
}

// emitToWindow reicht ein Ereignis an die Oberfläche.
func emitToWindow(ev ChangeEvent) {
	if app := application.Get(); app != nil {
		app.Event.Emit(EventChanged, ev)
	}
}
