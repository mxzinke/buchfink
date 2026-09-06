package actor

import "testing"

// Die Bearbeiterkennung hat immer dieselbe Form.
//
// Sie steht an jeder Buchung, an jedem Protokolleintrag und in jedem Export.
// Eine Kennung, die je nach Rechner einmal `anna@buero-pc` und einmal nur
// `anna` lautet, ließe sich später nicht mehr auswerten: derselbe Bearbeiter
// erschiene als zwei, und aus welchem Grund der Rechnername fehlt, stünde
// nirgends. Fehlt eine der beiden Angaben, tritt „unbekannt" an ihre Stelle.
func TestActorKeepsItsFormWhenAPartIsMissing(t *testing.T) {
	cases := []struct {
		name string
		user string
		host string
		want string
	}{
		{"beide bekannt", "anna", "buero-pc", "anna@buero-pc"},
		{"ohne Rechnername", "anna", "", "anna@" + Unknown},
		{"ohne Benutzer", "", "buero-pc", Unknown + "@buero-pc"},
		{"nichts bekannt", "", "", Unknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := format(c.user, c.host); got != c.want {
				t.Errorf("Kennung %q, erwartet %q", got, c.want)
			}
		})
	}
}

// Die ermittelte Kennung trägt die Form, die überall dokumentiert ist.
func TestActorIsUserAtHost(t *testing.T) {
	got := Actor()
	if got == "" {
		t.Fatal("die Bearbeiterkennung darf nicht leer sein")
	}
	if got == Unknown {
		// Weder Benutzer noch Rechner zu ermitteln ist erlaubt; dann steht der
		// Ersatzwert allein, und die Form gilt für ihn nicht.
		return
	}
	at := 0
	for _, r := range got {
		if r == '@' {
			at++
		}
	}
	if at != 1 {
		t.Errorf("die Kennung %q hat nicht die Form <Benutzer>@<Rechner>", got)
	}
}
