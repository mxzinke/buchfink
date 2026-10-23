# Abgrenzung zum Stichtag

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Grundsatz
2. Aktive Rechnungsabgrenzung
3. Passive Rechnungsabgrenzung
4. Kleine Beträge
5. Ausstehende Rechnungen und Erträge
6. Disagio
7. Folgejahr

## 1. Grundsatz

Aufwendungen und Erträge gehören in das Jahr, dem sie wirtschaftlich
zuzurechnen sind, unabhängig von der Zahlung ([§ 252 Abs. 1 Nr. 5 HGB][hgb252];
steuerlich [§ 5 Abs. 5 Satz 1 EStG][estg5]).

| Fall | Zahlung | Aufwand oder Ertrag | Posten |
|---|---|---|---|
| transitorisch | im alten Jahr | für eine bestimmte Zeit danach | Rechnungsabgrenzungsposten ([§ 250 HGB][hgb250]) |
| antizipativ | im neuen Jahr | im alten Jahr | sonstige Forderung oder Verbindlichkeit, bei ungewisser Höhe Rückstellung |

## 2. Aktive Rechnungsabgrenzung

- Ausgaben vor dem Stichtag, soweit sie Aufwand für eine kalendermäßig
  bestimmte Zeit nach dem Stichtag sind ([§ 250 Abs. 1 HGB][hgb250]), etwa
  Versicherung, Abonnement, Wartung, Miete oder Kfz-Steuer im Voraus.
- Such im letzten Quartal nach Aufwand, dessen Leistungszeitraum über den
  Stichtag reicht. Betrag: netto mal Zeitanteil nach dem Stichtag, nach
  Monaten oder taggenau, für alle Posten gleich.
- Buchung: Soll 1900, Haben das ursprüngliche Aufwandskonto;
  `abschlussbuchung` mit `art: abgrenzung`, `abgrenzung.art: aktiv` und
  `quell_buchung_id`.

## 3. Passive Rechnungsabgrenzung

- Einnahmen vor dem Stichtag, soweit sie Ertrag für eine bestimmte Zeit nach
  dem Stichtag sind ([§ 250 Abs. 2 HGB][hgb250]), etwa Jahreslizenz oder
  Wartung im Voraus.
- Steht die Vorauszahlung als
  [erhaltene Anzahlung](../rechnungen/anzahlungen.md) in Buchfink, entfällt der
  Posten; steht sie als Erlös, grenze den Nettoanteil ab.
- Buchung: Soll Erlöskonto, Haben 3900, ohne Steuerzeile; `abschlussbuchung`
  mit `art: abgrenzung` und `abgrenzung.art: passiv`.

## 4. Kleine Beträge

- Steuerlich darf der Posten unterbleiben, wenn die einzelne Ausgabe oder
  Einnahme 800 € nicht übersteigt; das Wahlrecht gilt einheitlich
  ([§ 5 Abs. 5 Satz 2 EStG][estg5] mit [§ 6 Abs. 2 Satz 1 EStG][estg6]).
  § 250 HGB kennt keine Grenze.
- Wertungsfrage: Leg dem Menschen Anzahl und Summe der Posten bis 800 € vor;
  er entscheidet einheitlich für beide Rechte. Halte die Entscheidung in der
  [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten) fest
  und wende sie im Folgejahr wieder an ([§ 246 Abs. 3 HGB][hgb246]).

## 5. Ausstehende Rechnungen und Erträge

- Leistung bezogen, Rechnung fehlt: Betrag sicher (Vertrag, Angebot,
  Pauschale): Soll Aufwandskonto, Haben 3500, netto, ohne Vorsteuer; Betrag
  unsicher: [Rückstellung](rueckstellungen.md). Die Rechnung buchst du später
  als [Eingangsbeleg](../belege/ablauf.md) und löst die Abgrenzung gegen
  dasselbe Aufwandskonto auf.
- Zinsen für das alte Jahr, im neuen belastet: Soll Zinsaufwand, Haben 3500
  ([bank/darlehen.md](../bank/darlehen.md#4-bankdarlehen-konten)).
- Ertrag im alten Jahr, Zahlung im neuen (Zinsgutschrift,
  Versicherungserstattung): Soll 1300, Haben Ertragskonto.
- Eigene Leistung im alten Jahr, Rechnung im neuen: Steht der Erlös im neuen
  Jahr (`buchungen_suchen`), gehört er ins alte (§ 252 Abs. 1 Nr. 4, 5 HGB);
  leg dem Menschen die Abgrenzung mit Konto vor. Die Umsatzsteuer ordnet
  Buchfink nach dem Leistungsdatum zu.

## 6. Disagio

- Ein aktiviertes Disagio (1940; Ansatz:
  [bank/darlehen.md](../bank/darlehen.md#6-disagio-und-bearbeitungsentgelt))
  schreibst du planmäßig über die Laufzeit ab
  ([§ 250 Abs. 3 Satz 2 HGB][hgb250]); schlag Laufzeit oder Zinsbindung vor.
  `abschlussbuchung` mit `art: abgrenzung`, `abgrenzung.art: disagio`: Soll
  7323, Haben 1940, Jahresanteil.
- Den gesonderten Ausweis oder die Anhangangabe ([§ 268 Abs. 6 HGB][hgb268])
  brauchen kleine Gesellschaften nicht ([§ 274a Nr. 3 HGB][hgb274a]).

## 7. Folgejahr

Abgrenzungen außerhalb von Buchfinks Kartei löst Buchfink nicht auf. Trag sie
in die [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten)
ein und buche nach dem Saldenvortrag die Auflösung mit `buchung_erfassen`
(Rückfrage):

- aktiv: Soll Aufwandskonto, Haben 1900, zum Jahresbeginn oder über die
  Laufzeit;
- passiv: Soll 3900, Haben Erlöskonto;
- sonstige Verbindlichkeit: Soll 3500, Haben Aufwandskonto, sobald die
  Rechnung gebucht ist.

Fertig, wenn 1900, 3900 und die Abgrenzungen auf 3500 und 1300 aus dem Vorjahr
aufgelöst sind.

[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
[hgb250]: https://www.gesetze-im-internet.de/hgb/__250.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[hgb274a]: https://www.gesetze-im-internet.de/hgb/__274a.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
