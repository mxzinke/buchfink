package oauth

import (
	"html/template"
	"net/http"
)

// Die Seiten im Browser sind statisch: kein Skript, kein Formular, keine
// fremde Quelle. Zugestimmt wird nicht hier, sondern im Fenster von Buchfink.

var pageTemplate = template.Must(template.New("seite").Parse(`<!doctype html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .Refresh}}<meta http-equiv="refresh" content="{{.RefreshAfter}};url={{.Refresh}}">{{end}}
<title>{{.Title}} – Buchfink</title>
<style>
body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;background:#faf9f6;color:#1f1d1a;margin:0;display:flex;min-height:100vh;align-items:center;justify-content:center}
main{max-width:30rem;padding:2rem}
h1{font-size:1.4rem;font-weight:600;margin:0 0 1rem}
p{line-height:1.5;margin:0 0 1rem;color:#4a4640}
.code{font-size:2rem;letter-spacing:.15em;font-variant-numeric:tabular-nums;font-weight:600;color:#1f1d1a;margin:1.5rem 0}
.klein{font-size:.85rem;color:#7a756c}
</style>
</head>
<body><main>
<h1>{{.Title}}</h1>
{{range .Lines}}<p>{{.}}</p>{{end}}
{{if .Code}}<p class="code">{{.Code}}</p>{{end}}
{{if .Footer}}<p class="klein">{{.Footer}}</p>{{end}}
</main></body>
</html>`))

type page struct {
	Title        string
	Lines        []string
	Code         string
	Footer       string
	Refresh      string
	RefreshAfter int
}

func writePage(w http.ResponseWriter, status int, p page) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = pageTemplate.Execute(w, p)
}

// waitingPage nennt nur App und Abgleichcode. Den Mandanten zeigt erst der
// Dialog in der App: /authorize erreicht jeder Prozess auf diesem Rechner.
func waitingPage(item *pending, waitURL string) page {
	return page{
		Title: "Bitte in Buchfink bestätigen",
		Lines: []string{
			"„" + item.req.ClientName + "“ möchte auf Buchfink zugreifen. Wechseln Sie in das Fenster von Buchfink und vergleichen Sie dort diesen Code:",
		},
		Code:         item.req.UserCode,
		Footer:       "Diese Seite wartet auf Ihre Entscheidung und leitet danach selbst weiter.",
		Refresh:      waitURL,
		RefreshAfter: 0,
	}
}

func errorPage(w http.ResponseWriter, status int, title string, lines ...string) {
	writePage(w, status, page{Title: title, Lines: lines,
		Footer: "Sie können dieses Fenster schließen."})
}
