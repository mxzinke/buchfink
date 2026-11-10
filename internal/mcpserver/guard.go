package mcpserver

import (
	"net/http"
	"strings"

	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
)

// Der Server lauscht nur auf 127.0.0.1. Eine Webseite im Browser erreicht ihn
// trotzdem: über DNS-Rebinding (ein fremder Name, der auf 127.0.0.1 zeigt)
// oder über eine Anfrage, die der Browser im Namen der Seite stellt. Gegen
// das erste hilft der Host-Kopf, gegen das zweite Origin und Sec-Fetch-Site.

// guardHost weist jede Anfrage ab, deren Host nicht dieser Server ist.
func guardHost(as *oauth.Server, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !as.HostAllowed(r.Host) {
			http.Error(w, "Unzulässiger Host", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// private lässt keine Anfrage durch, die eine fremde Seite ausgelöst hat.
// Apps senden weder Origin noch Sec-Fetch-Site; ein vom Betriebssystem
// geöffneter Browser sendet Sec-Fetch-Site: none. Es gibt kein CORS.
func private(as *oauth.Server, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && !sameOrigin(as, origin) {
			http.Error(w, "Anfragen fremder Seiten sind nicht erlaubt", http.StatusForbidden)
			return
		}
		switch r.Header.Get("Sec-Fetch-Site") {
		case "", "none", "same-origin":
		default:
			http.Error(w, "Anfragen fremder Seiten sind nicht erlaubt", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(as *oauth.Server, origin string) bool {
	host, ok := strings.CutPrefix(strings.ToLower(origin), "http://")
	return ok && as.HostAllowed(host)
}

// getOnly lässt nur GET durch. Der ServeMux bedient HEAD über die GET-Route;
// ein HEAD auf /authorize legte sonst eine Anfrage vor, eines auf
// /authorize/warten holte den Ausgang ab.
func getOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Methode nicht erlaubt", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// public sind die Metadaten: öffentlich und für jede Seite lesbar
// (RFC 9728 §3.1, RFC 8414 §3).
func public(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, MCP-Protocol-Version")
		switch r.Method {
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet, http.MethodHead:
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Allow", "GET, OPTIONS")
			http.Error(w, "Methode nicht erlaubt", http.StatusMethodNotAllowed)
		}
	})
}
