package oauth

import (
	"errors"
	"sync"
	"time"
)

// Zugestimmt wird nur im Fenster von Buchfink. Der Browser zeigt unter
// /authorize lediglich „Bitte in Buchfink bestätigen" mit einem Abgleichcode;
// die Entscheidung kommt über die Bridge (Resolve mit Einmal-Nonce) und nie
// über HTTP. Eine Zustimmungsseite im Browser ließe sich von jedem lokalen
// Prozess selbst bestätigen.

// ConsentRequest ist eine offene Anfrage, wie die App sie zeigt.
type ConsentRequest struct {
	ID    string `json:"id"`
	Nonce string `json:"nonce"`
	// ClientName ist die Angabe der App bei der Registrierung, nicht geprüft.
	ClientName         string    `json:"clientName"`
	ClientID           string    `json:"clientId"`
	ClientRegisteredAt time.Time `json:"clientRegisteredAt"`
	// KnownSince ist die erste Zustimmung zu dieser App, sonst null.
	KnownSince *time.Time `json:"knownSince"`
	// RedirectURI samt Port: wohin der Browser den Code bringt.
	RedirectURI string   `json:"redirectUri"`
	RedirectTo  string   `json:"redirectTo"`
	Scopes      []string `json:"scopes"`
	// TenantID und TenantName sind der Mandant, für den zugestimmt wird;
	// leer, wenn keiner geöffnet ist.
	TenantID   string    `json:"tenantId"`
	TenantName string    `json:"tenantName"`
	UserCode   string    `json:"userCode"`
	ExpiresAt  time.Time `json:"expiresAt"`
	// Replacing: die Anfrage löst eine offene derselben App ab. Der Broker
	// holt das Fenster dafür nicht erneut nach vorn.
	Replacing bool `json:"-"`
}

// ConsentBroker bringt Anfragen in die App. Present darf nicht blockieren;
// die Entscheidung kommt über Server.Resolve. Withdraw nimmt eine erledigte,
// ersetzte oder abgelaufene Anfrage wieder heraus.
type ConsentBroker interface {
	Present(r ConsentRequest)
	Withdraw(id string)
}

// Fehler beim Entscheiden einer Anfrage.
var (
	ErrUnknownRequest  = errors.New("die Anfrage ist unbekannt oder schon entschieden")
	ErrExpiredRequest  = errors.New("die Anfrage ist abgelaufen")
	ErrInvalidNonce    = errors.New("die Bestätigung passt nicht zur Anfrage")
	ErrTooManyRequests = errors.New("es warten schon zu viele Anmeldungen auf Bestätigung")
	ErrTooSoon         = errors.New("diese App hat gerade erst angefragt")
)

// Grenzen der offenen Anfragen. Jede Anfrage holt das Fenster nach vorn und
// belegt Speicher; /authorize erreicht jeder Prozess auf diesem Rechner.
const (
	// clientInterval ist der Mindestabstand zweier Anfragen derselben App.
	// Nach einer Zustimmung darf sie sofort wieder anfragen.
	clientInterval = 10 * time.Second
	// maxEntries deckelt alle Einträge, auch erledigte, nicht abgeholte.
	maxEntries = 32
)

// pending ist eine offene Anfrage mit ihrem Ausgang.
type pending struct {
	req ConsentRequest

	redirectURI string
	state       string
	challenge   string
	resource    string

	done chan struct{}
	// Nach done gesetzt:
	approved bool
	code     string
	reason   string
	doneAt   time.Time
	// claimed: die Zustimmung ist angenommen, der Code entsteht gerade.
	claimed bool
	// delivered: der Browser hat den Rücksprung abgeholt.
	delivered bool
	created   time.Time
	timer     *time.Timer
}

// pendingSet hält die Anfragen im Speicher. Sie überleben keinen Neustart;
// der Browser zeigt dann „abgelaufen". Ein erledigter Ausgang bleibt nur so
// lange, bis der Browser ihn abholt, höchstens keep.
type pendingSet struct {
	mu    sync.Mutex
	items map[string]*pending
	// last ist je App die jüngste Anfrage der letzten clientInterval.
	last  map[string]*pending
	max   int
	keep  time.Duration
	clock func() time.Time
}

func newPendingSet(max int, keep time.Duration, clock func() time.Time) *pendingSet {
	return &pendingSet{items: map[string]*pending{}, last: map[string]*pending{}, max: max, keep: keep, clock: clock}
}

// add nimmt eine Anfrage auf und löst eine offene derselben App ab; die
// abgelöste ist danach entschieden und entfernt. Mehr als max offene, mehr
// als maxEntries Einträge oder eine zweite Anfrage derselben App binnen
// clientInterval (außer nach einer Zustimmung) gibt es nicht.
func (p *pendingSet) add(item *pending) (superseded []string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := item.created
	for id, other := range p.items {
		if isDone(other) && (other.delivered || now.Sub(other.doneAt) > p.keep) {
			delete(p.items, id)
		}
	}
	for id, other := range p.last {
		if now.Sub(other.created) >= clientInterval {
			delete(p.last, id)
		}
	}
	if prev := p.last[item.req.ClientID]; prev != nil && !(isDone(prev) && prev.approved) {
		return nil, ErrTooSoon
	}
	var replace []*pending
	open := 0
	for _, other := range p.items {
		switch {
		case isDone(other):
		case other.req.ClientID == item.req.ClientID && !other.claimed:
			replace = append(replace, other)
		default:
			open++
		}
	}
	if open >= p.max || len(p.items)-len(replace) >= maxEntries {
		return nil, ErrTooManyRequests
	}
	for _, old := range replace {
		p.decide(old, false, "", "durch eine neue Anmeldung derselben App ersetzt")
		delete(p.items, old.req.ID)
		superseded = append(superseded, old.req.ID)
	}
	item.req.Replacing = len(replace) > 0
	p.items[item.req.ID] = item
	p.last[item.req.ClientID] = item
	return superseded, nil
}

func (p *pendingSet) get(id string) *pending {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.items[id]
}

// open liefert die unentschiedenen Anfragen.
func (p *pendingSet) open() []ConsentRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := []ConsentRequest{}
	for _, item := range p.items {
		if !isDone(item) {
			out = append(out, item.req)
		}
	}
	return out
}

// claim reserviert eine Anfrage für die Zustimmung, solange der Code
// entsteht. Ablauf und Ablösung warten nicht darauf, sondern lassen sie aus.
func (p *pendingSet) claim(item *pending) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if isDone(item) || item.claimed {
		return false
	}
	item.claimed = true
	return true
}

// finish entscheidet eine Anfrage, genau einmal. Eine reservierte Anfrage
// entscheidet nur, wer sie reserviert hat.
func (p *pendingSet) finish(item *pending, approved bool, code, reason string, claimer bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if isDone(item) || (item.claimed && !claimer) {
		return false
	}
	p.decide(item, approved, code, reason)
	return true
}

// decide setzt den Ausgang; p.mu ist gehalten.
func (p *pendingSet) decide(item *pending, approved bool, code, reason string) {
	item.approved, item.code, item.reason = approved, code, reason
	item.doneAt = p.clock()
	if item.timer != nil {
		item.timer.Stop()
	}
	close(item.done)
}

// take holt den Ausgang für den Rücksprung ab, genau einmal.
func (p *pendingSet) take(item *pending) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if item.delivered {
		return false
	}
	item.delivered = true
	return true
}

func isDone(item *pending) bool {
	select {
	case <-item.done:
		return true
	default:
		return false
	}
}
