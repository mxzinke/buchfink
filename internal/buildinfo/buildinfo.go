// Package buildinfo hält fest, welche Fassung des Programms läuft.
//
// Sie steht an allem, was Buchfink nach außen abgibt — allen voran an der
// übermittelten Umsatzsteuer-Voranmeldung. Die Zuordnung Steuerfall →
// Kennziffer folgt einem amtlichen Vordruck, der sich fast jedes Jahr ändert;
// ohne die Fassung, die gerechnet hat, ließe sich eine alte Anmeldung später
// nicht mehr nachvollziehen (GoBD Rz. 34, Verfahrensdokumentation).
package buildinfo

import "fmt"

// Version ist die Fassung des Programms.
//
// Sie ist eine Variable und keine Konstante, damit der Bau sie setzen kann:
//
//	go build -ldflags "-X github.com/buchfink/buchfink/internal/buildinfo.Version=1.2.3"
//
// Ohne das steht hier die Fassung des Arbeitsstands. Ein Name, der sich nicht
// mit dem Bau ändert — eine Wellen- oder Meilensteinbezeichnung etwa —, wäre
// keine Version, sondern eine Behauptung.
var Version = "0.3.0-dev"

// Program ist die Fassung mit dem Regelstand, unter dem gebucht wurde.
//
// Beide gehören zusammen: die Programmfassung sagt, welcher Code gerechnet hat,
// der Regelstand, welche Kontierungs- und Steuerregeln dabei galten. Getrennt
// beantwortet keine von beiden die Frage, wie eine Zahl zustande kam.
func Program(postingRuleVersion string) string {
	return fmt.Sprintf("buchfink %s/Regeln %s", Version, postingRuleVersion)
}
