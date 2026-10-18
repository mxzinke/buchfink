# Ausweis gegenüber Gesellschaftern im Abschluss

Rechtsstand: 17. Oktober 2026. Den Abschluss selbst regelt
buchfink-jahresabschluss; hier steht nur, was Gesellschafter betrifft.

## Regel

- Ausleihungen, Forderungen und Verbindlichkeiten gegenüber Gesellschaftern
  sind in der Regel gesondert auszuweisen oder im Anhang anzugeben; stehen sie
  unter anderen Posten, ist das zu vermerken ([§ 42 Abs. 3 GmbHG][gmbhg42]).
- Forderungen und Verbindlichkeiten werden nicht miteinander verrechnet
  ([§ 246 Abs. 2 Satz 1 HGB][hgb246]).
- Restlaufzeiten: Forderungen über einem Jahr und Verbindlichkeiten bis zu
  einem Jahr sind je Posten zu vermerken
  ([§ 268 Abs. 4 Satz 1, Abs. 5 Satz 1 HGB][hgb268]).

## Konten und Bilanzposten (SKR04-Katalog von Buchfink)

| Konto | Bezeichnung | Posten laut Katalog |
|---|---|---|
| 0960 | Ausleihungen an Gesellschafter | Finanzanlagen, Ausleihungen an Gesellschafter |
| 1250 | Forderungen aus Lieferungen und Leistungen gegen Gesellschafter | Forderungen gegen Gesellschafter |
| 1307, 1308, 1309 | Forderungen gegen GmbH-Gesellschafter (gesamt, bis 1 Jahr, über 1 Jahr) | Forderungen gegen Gesellschafter |
| 3340 | Verbindlichkeiten aus Lieferungen und Leistungen gegenüber Gesellschaftern | Verbindlichkeiten gegenüber Gesellschaftern |
| 3519 | Verbindlichkeiten gegenüber Gesellschaftern für offene Ausschüttungen | Verbindlichkeiten gegenüber Gesellschaftern |
| 3640 bis 3643 | Verbindlichkeiten gegenüber GmbH-Gesellschaftern (gesamt, bis 1, 1 bis 5, über 5 Jahre) | Verbindlichkeiten gegenüber Gesellschaftern |
| 2928 | Kapitalrücklage durch Zuzahlungen in das Eigenkapital | Eigenkapital, Kapitalrücklage |

Den Posten jedes Kontos liest du mit `konten` (`art: suche`) oder in der
Bilanz aus `auswertung`.

## Prüfung vor dem Abschluss

1. `auswertung` (`art: susa`, `geschaeftsjahr`): Salden aller Konten der Tabelle.
2. Je Gesellschafter: Stimmt der Saldo mit Vertrag, Kontoauszug des
   Gesellschafters oder seiner Saldenbestätigung überein? Frag den Menschen.
3. Liegt ein Saldo auf 1307 statt auf 1308 oder 1309 bzw. auf 3640 statt
   auf 3641 bis 3643, schlag die Umbuchung auf das Restlaufzeitkonto vor
   (`buchung_erfassen`, Rückfrage).
4. Rechnungen mit Gesellschaftern als Kunden oder Lieferanten laufen über ihr
   Personenkonto. Ob sie unter „Forderungen gegen Gesellschafter“ zu zeigen
   sind, klärt buchfink-jahresabschluss bei der Bilanz.

[gmbhg42]: https://www.gesetze-im-internet.de/gmbhg/__42.html
[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
