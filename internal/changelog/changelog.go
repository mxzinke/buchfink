// Package changelog hält die Versionshistorie des Programms.
//
// Sie ist eingebettet und nicht daneben abgelegt: eine Historie, die als Datei
// neben dem Programm liegt, fehlt in dem Moment, in dem sie gebraucht wird —
// bei der Prüfung einer Sicherung auf einem anderen Rechner. UNV-06 verlangt,
// dass sich zu jeder Programmfassung feststellen lässt, was sie geändert hat;
// dafür muss die Historie mit dem Programm reisen.
package changelog

import _ "embed"

//go:embed CHANGELOG.md
var markdown string

// Markdown liefert die Änderungshistorie als Markdown.
func Markdown() string { return markdown }
