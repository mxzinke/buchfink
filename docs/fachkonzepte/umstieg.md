# Umstieg aus einem Vorsystem

[Fachkonzepte](README.md) · [Gründung und Eröffnung](gruendung.md) ·
[Anlagen](anlagenverwaltung.md)

Code: `service.TakeoverService` (`service/takeover_service.go`,
`takeover_booking.go`, `takeover_reconciliation.go`),
`service/opening_balance.go`, `accounting/afa.go`
(`AfAPlan.TakeoverYear`), `mcpserver/eroeffnung.go`,
`wailsbridge/takeover_service.go`,
`frontend/src/pages/company/TakeoverBlock.tsx`.

Ein Unternehmen, das seine Bücher bisher in einem anderen Programm geführt
hat, beginnt in Buchfink mit den Beständen der Schlussbilanz des Vorjahres. Die
Wertansätze zu Beginn des Geschäftsjahres müssen mit denen dieser Schlussbilanz
übereinstimmen ([§ 252 Abs. 1 Nr. 1 HGB][hgb252]). Arbeitsschritte und Quellen
für den Agenten stehen in
[`skills/buchfink/references/gruendung/umstieg.md`](../../skills/buchfink/references/gruendung/umstieg.md).

## Der Umstieg als Stammdatum

Ob ein Unternehmen mit bestehenden Büchern beginnt und zu welchem Tag, legt der
Mensch fest: in der Einrichtung mit „Umstieg“ oder unter Unternehmen ›
Stammdaten im Block „Umstieg“ (`TakeoverService.SetDate`, Bridge
`SetTakeoverDate`, Einstellung `takeover_date`). Der Vorgang
`umstieg_festlegen` ist der App vorbehalten. Ohne festgelegten Umstieg nimmt
der Dienst keine Bestände aus einem Vorsystem an; ein Agent bucht die
Eröffnung, entscheidet aber nicht, ob es eine gibt.

Der Dienst nimmt den Tag an, wenn

- er der erste Tag eines Geschäftsjahres ist,
- keine Gründung erfasst ist und das Gründungsdatum der Unternehmensdaten vor
  ihm liegt,
- kein früheres Geschäftsjahr Buchungen hat.

Gründung und Umstieg schließen sich aus: `FoundationService.Save` lehnt ab,
sobald ein Umstieg festgelegt ist, und der Pflichtenkalender bildet für einen
Umsteiger keine Gründungspflichten. Ändern oder zurücknehmen lässt sich der
Tag, solange keine Eröffnung gebucht ist.

## Eröffnung

`TakeoverService.Book` bucht aus einer Eingabe drei Teile in einer Transaktion.
Die App erfasst sie im Block „Umstieg“, ein Agent mit `eroeffnung_buchen` nach
Rückfrage. Beleg ist die abgelegte Schlussbilanz oder Summen- und Saldenliste
des Vorsystems; ihr Hash steht in jeder Eröffnungsbuchung, und sie ist danach
versiegelt.

| Teil | Buchung |
|---|---|
| Sachkonten | eine Buchung mit allen Bestandskonten, Ausgleich auf **9000** |
| Offene Posten | je Rechnung eine Buchung auf das Personenkonto gegen **9008** (Debitoren) oder **9009** (Kreditoren), mit Kontakt, Rechnungsnummer, Rechnungsdatum, Fälligkeit und dem im Vorsystem erklärten Steuersatz |
| Anlagen | je Anlagegut ein Eintrag im Anlagenverzeichnis mit historischen Anschaffungskosten und einer Bewegung `takeover` über die bis zum Umstieg gebuchten Abschreibungen |

Die Sammelkonten **1200** und **3300** stehen in der Eingabe als Kontrollsumme
und werden nicht gebucht. Der Dienst bucht nur, wenn

- Aktiva und Passiva gleich sind, die Vortragskonten 9000, 9008 und 9009 sich
  also aufheben,
- die offenen Posten je Seite den Bestand ihres Sammelkontos ergeben,
- die Buchwerte der Anlagen je Anlagekonto dessen Bestand ergeben.

Die Vorschau (`TakeoverService.Preview`, `nur_vorschau`) rechnet dasselbe und
nennt jede Abweichung als Differenz.

Übernommene Posten speichern den Steuerfall der Rechnung, ohne erneut Steuer
zu buchen. Unterstützt sind inländische Rechnungen mit 19 oder 7 Prozent und
nicht steuerbare Posten. Fehlt der Steuersatz, sind vollständige Zahlungen
möglich; Skonto und Forderungsausfall weist Buchfink ab. Der Saldenvortrag
nimmt offene übernommene Posten wie jede Rechnung ins Folgejahr mit.

Eine zweite Eröffnung verdoppelte die Bestände und wird abgewiesen. Nach dem
Storno aller Eröffnungsbuchungen lässt sich die Eröffnung mit demselben Beleg
neu buchen; die übernommenen Anlagen bleiben dabei im Verzeichnis und werden
mit der neuen Buchung verbunden.

## Anlagen fortführen

Beispiel: Pkw, angeschafft am 15.01.2024 für 30.000,00 €, Nutzungsdauer 72
Monate, im Vorsystem bis 31.12.2025 mit 10.000,00 € abgeschrieben, Umstieg zum
01.01.2026.

| | Betrag |
|---|---|
| Zugangsbewegung 2024 (Anschaffungskosten) | 30.000,00 |
| Bewegung `takeover` 2025 (aufgelaufene Abschreibung) | 10.000,00 |
| Buchwert zum Umstieg, Bestand auf **0520** | 20.000,00 |
| AfA 2026: 20.000,00 auf 48 Restmonate | 5.000,00 |

Der Abschreibungsplan lässt die Jahre des Vorsystems aus und beginnt im
Umstiegsjahr mit dem übernommenen Buchwert (`AfAPlan.TakeoverYear`,
`TakeoverDepreciation`). Linear und degressiv verteilt er ihn auf die
Restnutzungsdauer; bei Gebäude, Elektrofahrzeug und Sammelposten läuft der
Satz auf die Anschaffungskosten weiter. Maßgeblich ist der Stand des
Vorsystems: es kann anders gerundet oder außerplanmäßig abgeschrieben haben.
Der AfA-Lauf meldet für ein übernommenes Anlagegut keine fehlenden Vorjahre,
und der Anlagenspiegel zeigt Anschaffungskosten und kumulierte Abschreibungen
zu Beginn des ersten Jahres getrennt.

## Abstimmung

`TakeoverService.Reconciliation` stellt die nicht stornierten
Eröffnungsbuchungen der erfassten Schlussbilanz gegenüber (Einstellung
`takeover_statement`):

- je Sachkonto Bestand laut Schlussbilanz, gebuchter Bestand und Differenz;
  die Sammelkonten zeigen die Summe der übernommenen Personenkonten,
- Aktiva und Passiva der Schlussbilanz und der Buchungen, dazu der Saldo der
  Vortragskonten,
- Nebenbücher: offene Posten gegen ihr Sammelkonto, Anlagen je Anlagekonto
  gegen dessen gebuchten Bestand.

Die App zeigt die Abstimmung im Block „Umstieg“, ein Agent liest sie mit
`auswertung` (`art: umstieg`). Der Prüflauf führt jede Differenz und eine
fehlende Eröffnung als sperrenden Befund `takeover_opening`; die
Festschreibung eines Monats ab dem Umstieg scheitert daran, bis die Eröffnung
aufgeht oder der Mensch den Befund mit Begründung übergeht.

## Grenzen

- **Umstieg im laufenden Geschäftsjahr.** Der Dienst nimmt nur Bestandskonten
  zum ersten Tag eines Geschäftsjahres an. Salden der Aufwands- und
  Ertragskonten und bereits abgegebene Voranmeldungen übernimmt er nicht.
- **Vorjahreszahlen.** Bilanz und GuV des ersten Jahres zeigen kein Vorjahr;
  [§ 265 Abs. 2 HGB][hgb265] verlangt die Beträge des Vorjahres zu jedem
  Posten. Eine Eingabe dafür gibt es nicht.
- **Steuerliche Vorträge.** Verlustvorträge übernimmt die Eröffnung nicht. Die
  Steuerrückstellung nimmt sie als Angabe entgegen
  ([Jahresabschluss](jahresabschluss.md)).
- **Anlagen.** Nicht übernommen werden sofort abgezogene geringwertige
  Wirtschaftsgüter, die Sonderabschreibung nach § 7g EStG, Stückzahlen von
  Wertpapieren und das Verzeichnis nach § 15a UStG. Nach einem Storno der
  Eröffnung lassen sich die übernommenen Anlagen nicht neu erfassen.
- **Rückstellungen und Abgrenzungen** stehen als Kontobestand, nicht im
  Rückstellungs- oder Abgrenzungsspiegel.
- **Offene Posten** in Fremdwährung und Anzahlungen übernimmt der Dienst
  nicht.
- **Altdaten.** Buchfink liest keine Bewegungsdaten des Vorsystems ein. Das
  Vorsystem bleibt für den Datenzugriff aufzubewahren
  ([§ 147 Abs. 6 AO][ao147]); den Tag der Umstellung hält die Einstellung
  „Umstellung aus einem Altsystem“ für die Verfahrensdokumentation fest.

[ao147]: https://www.gesetze-im-internet.de/ao_1977/__147.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb265]: https://www.gesetze-im-internet.de/hgb/__265.html
