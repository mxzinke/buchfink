package domain

import (
	"fmt"
	"strings"
)

// Die 13-stellige Steuernummer im vereinheitlichten Bundesschema, wie ELSTER,
// ERiC-Programme und die E-Bilanz sie verlangen. Die Länder vergeben ihre
// Steuernummern in eigenen Schemata (F = Finanzamt, B = Bezirk,
// U = Unterscheidungsnummer, P = Prüfziffer); das Bundesschema setzt die
// Landesnummer vor die Finanzamtsnummer und fügt an fünfter Stelle eine 0 ein.
// Tabelle nach dem Handbuch zu myebilanz 2.3.1, Abschnitt 15.1 (dort aus
// Wikipedia „Steuernummer" übernommen), mit dessen Beispielen in den Tests.
type taxNumberScheme struct {
	// Digits ist die Länge der Landesnummer ohne Trennzeichen.
	digits int
	// Leading ist die feste erste Ziffer der Landesnummer, leer wenn keine.
	leading string
	// Prefix steht im Bundesschema vor der Finanzamtsnummer.
	prefix string
	// DropLeading lässt die feste erste Ziffer weg (Hessen: 0FF → 26FF).
	dropLeading bool
}

var taxNumberSchemes = map[string]taxNumberScheme{
	"BW": {digits: 10, prefix: "28"},
	"BY": {digits: 11, prefix: "9"},
	"BE": {digits: 10, prefix: "11"},
	"BB": {digits: 11, leading: "0", prefix: "3"},
	"HB": {digits: 10, prefix: "24"},
	"HH": {digits: 10, prefix: "22"},
	"HE": {digits: 11, leading: "0", prefix: "26", dropLeading: true},
	"MV": {digits: 11, leading: "0", prefix: "4"},
	"NI": {digits: 10, prefix: "23"},
	"NW": {digits: 11, prefix: "5"},
	"RP": {digits: 10, prefix: "27"},
	"SL": {digits: 11, leading: "0", prefix: "1"},
	"SN": {digits: 11, leading: "2", prefix: "3"},
	"ST": {digits: 11, leading: "1", prefix: "3"},
	"SH": {digits: 10, prefix: "21"},
	"TH": {digits: 11, leading: "1", prefix: "4"},
}

// ElsterTaxNumber rechnet die Steuernummer eines Landes in das 13-stellige
// Bundesschema um. Steht sie schon im Bundesschema, wird sie geprüft und
// zurückgegeben. Leere Steuernummer oder leeres Land ergeben "" ohne Fehler.
func ElsterTaxNumber(taxNumber, state string) (string, error) {
	digits := onlyDigits(taxNumber)
	state = strings.ToUpper(strings.TrimSpace(state))
	if digits == "" || state == "" {
		return "", nil
	}
	scheme, ok := taxNumberSchemes[state]
	if !ok {
		return "", fmt.Errorf("unbekanntes Bundesland %q", state)
	}
	if len(digits) == 13 {
		if !strings.HasPrefix(digits, scheme.prefix) || digits[4] != '0' {
			return "", fmt.Errorf("die 13-stellige Steuernummer %s passt nicht zum Bundesland %s "+
				"(erwartet Beginn %s und an fünfter Stelle eine 0)", digits, state, scheme.prefix)
		}
		return digits, nil
	}
	// Hessen schreibt die führende 0 nicht immer mit.
	if scheme.dropLeading && len(digits) == scheme.digits-1 {
		digits = scheme.leading + digits
	}
	if len(digits) != scheme.digits {
		return "", fmt.Errorf("eine Steuernummer aus %s hat %d Ziffern, eingetragen sind %d (%s)",
			state, scheme.digits, len(digits), strings.TrimSpace(taxNumber))
	}
	if scheme.leading != "" && !strings.HasPrefix(digits, scheme.leading) {
		return "", fmt.Errorf("eine Steuernummer aus %s beginnt mit %s, eingetragen ist %s",
			state, scheme.leading, strings.TrimSpace(taxNumber))
	}
	rest := digits
	if scheme.dropLeading {
		rest = digits[1:]
	}
	// Die Finanzamtsnummer ist im Bundesschema dreistellig nach dem
	// Länderpräfix (vierstellig mit ihm), danach folgt die eingefügte 0.
	officeLen := 4 - len(scheme.prefix)
	out := scheme.prefix + rest[:officeLen] + "0" + rest[officeLen:]
	if len(out) != 13 {
		return "", fmt.Errorf("die Steuernummer %s ließ sich nicht umrechnen", strings.TrimSpace(taxNumber))
	}
	return out, nil
}

// TaxOfficeNumber ist die vierstellige Bundesfinanzamtsnummer: die ersten vier
// Ziffern der 13-stelligen Steuernummer.
func TaxOfficeNumber(elster string) string {
	if len(elster) != 13 {
		return ""
	}
	return elster[:4]
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
