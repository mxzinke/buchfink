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
	// claimed: die Zustimmung ist angenommen, der Code entsteht gerade.
	claimed bool
	// delivered: der Browser hat den Rücksprung abgeholt.
	delivered bool
	created   time.Time
	timer     *time.Timer
}

// pendingSet hält die offenen Anfragen im Speicher. Sie überleben keinen
// Neustart; der Browser zeigt dann „abgelaufen".
type pendingSet struct {
	mu    sync.Mutex
	items map[string]*pending
	max   int
}

func newPendingSet(max int) *pendingSet {
	return &pendingSet{items: map[string]*pending{}, max: max}
}

// add nimmt eine Anfrage auf. Eine noch offene Anfrage derselben App wird
// abgelöst; mehr als max offene gibt es nicht.
func (p *pendingSet) add(item *pending, now time.Time) (superseded []*pending, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, other := range p.items {
		if isDone(other) && (other.delivered || now.Sub(other.created) > 2*time.Hour) {
			delete(p.items, id)
		}
	}
	open := 0
	for _, other := range p.items {
		if isDone(other) {
			continue
		}
		if other.req.ClientID == item.req.ClientID {
			superseded = append(superseded, other)
			continue
		}
		open++
	}
	if open >= p.max {
		return nil, ErrTooManyRequests
	}
	p.items[item.req.ID] = item
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
	item.approved, item.code, item.reason = approved, code, reason
	if item.timer != nil {
		item.timer.Stop()
	}
	close(item.done)
	return true
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
