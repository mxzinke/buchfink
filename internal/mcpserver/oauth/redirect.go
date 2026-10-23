package oauth

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Rücksprungadressen sind nur Loopback-Adressen von Apps auf diesem Rechner
// (RFC 8252 §7.3). Der Port wechselt mit jeder Anmeldung und zählt beim
// Abgleich nicht; Schema, Host, Pfad und Query müssen genau stimmen.

// checkRedirectURI prüft eine Rücksprungadresse bei der Registrierung.
func checkRedirectURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%q ist keine gültige Adresse", raw)
	}
	if u.Scheme != "http" {
		return fmt.Errorf("%q: nur http-Rücksprünge auf diesen Rechner sind erlaubt", raw)
	}
	if u.User != nil {
		return fmt.Errorf("%q: Benutzerangaben in der Adresse sind nicht erlaubt", raw)
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("%q: ein Fragment ist nicht erlaubt", raw)
	}
	if !isLoopbackHost(u.Hostname()) {
		return fmt.Errorf("%q: erlaubt sind nur 127.0.0.1, [::1] und localhost", raw)
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return fmt.Errorf("%q: ungültiger Port", raw)
		}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "127.0.0.1", "::1", "localhost":
		return true
	}
	return false
}

// matchRedirect sucht die registrierte Adresse, zu der eine angefragte
// passt. Der Port bleibt außer Betracht.
func matchRedirect(registered []string, requested string) bool {
	req, err := url.Parse(requested)
	if err != nil || checkRedirectURI(requested) != nil {
		return false
	}
	for _, r := range registered {
		reg, err := url.Parse(r)
		if err != nil {
			continue
		}
		if reg.Scheme == req.Scheme &&
			strings.EqualFold(reg.Hostname(), req.Hostname()) &&
			reg.EscapedPath() == req.EscapedPath() &&
			reg.RawQuery == req.RawQuery {
			return true
		}
	}
	return false
}

// redirectHostPort ist das, was die App als Rücksprungziel anzeigt.
func redirectHostPort(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Port() == "" {
		return u.Hostname()
	}
	return net.JoinHostPort(u.Hostname(), u.Port())
}
