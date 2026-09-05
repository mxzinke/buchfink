package accounting

import (
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die Vorratskonten des SKR04 und ihre Gegenkonten.
//
// § 275 Abs. 2 HGB weist die Bestandsveränderung an zwei verschiedenen Stellen
// aus: die der fertigen und unfertigen Erzeugnisse als Nr. 2 unmittelbar hinter
// den Umsatzerlösen, die der Roh-, Hilfs- und Betriebsstoffe und der Waren als
// Teil des Materialaufwands (Nr. 5). Das eine ist eine Ertrags-, das andere eine
// Aufwandsposition — welches Gegenkonto die Buchung trägt, folgt deshalb aus dem
// Bestandskonto und ist keine Wahl.

// InventoryGroup fasst die Vorratskonten zu Gruppen mit gemeinsamem Gegenkonto
// zusammen.
type InventoryGroup struct {
	Label         string
	From, To      int
	ChangeAccount string
}

// inventoryGroups sind die Bereiche der Kontenklasse 1, die Vorräte tragen.
// Die geleisteten Anzahlungen auf Vorräte (1180 ff.) gehören nicht dazu: sie
// sind eine Vorleistung und kein Bestand, und ihre Veränderung ist keine
// Bestandsveränderung.
var inventoryGroups = []InventoryGroup{
	{"Roh-, Hilfs- und Betriebsstoffe", 1000, 1039, domain.AccountBestandRHBWaren},
	{"Unfertige Erzeugnisse und Leistungen", 1040, 1049, domain.AccountBestandUnfertige},
	{"Unfertige Erzeugnisse", 1050, 1079, domain.AccountBestandUnfertige},
	{"Unfertige Leistungen", 1080, 1089, domain.AccountBestandUnfertigeLeist},
	{"In Ausführung befindliche Aufträge", 1090, 1099, domain.AccountBestandUnfertigeLeist},
	{"Fertige Erzeugnisse und Waren", 1100, 1109, domain.AccountBestandFertige},
	{"Fertige Erzeugnisse", 1110, 1139, domain.AccountBestandFertige},
	{"Waren", 1140, 1179, domain.AccountBestandRHBWaren},
}

// InventoryGroups liefert die Bereiche für die Oberfläche.
func InventoryGroups() []InventoryGroup {
	out := make([]InventoryGroup, len(inventoryGroups))
	copy(out, inventoryGroups)
	return out
}

// IsInventoryAccount meldet, ob ein Konto ein Vorratsbestandskonto ist.
func IsInventoryAccount(account string) bool {
	_, err := InventoryChangeAccount(account)
	return err == nil
}

// InventoryChangeAccount nennt das Gegenkonto der Bestandsveränderung.
func InventoryChangeAccount(account string) (string, error) {
	n, err := strconv.Atoi(account)
	if err != nil {
		return "", fmt.Errorf("%q ist keine Kontonummer", account)
	}
	for _, g := range inventoryGroups {
		if n >= g.From && n <= g.To {
			return g.ChangeAccount, nil
		}
	}
	return "", fmt.Errorf(
		"Konto %s ist kein Vorratskonto. Vorräte liegen im SKR04 zwischen 1000 und 1179", account)
}

// InventoryGroupLabel benennt die Gruppe eines Vorratskontos.
func InventoryGroupLabel(account string) string {
	n, err := strconv.Atoi(account)
	if err != nil {
		return ""
	}
	for _, g := range inventoryGroups {
		if n >= g.From && n <= g.To {
			return g.Label
		}
	}
	return ""
}
