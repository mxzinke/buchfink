# Abgrenzung zum Stichtag

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 246, 250, 252, 268, 274a HGB
und § 5 Abs. 5, § 6 Abs. 2 EStG am 17.10.2026 gelesen.

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
zuzurechnen sind, unabhängig von der Zahlung ([§ 252 Abs. 1 Nr. 5 HGB][hgb252]).
Zwei Fälle:

| Fall | Zahlung | Aufwand oder Ertrag | Posten |
|---|---|---|---|
| transitorisch | im alten Jahr | für eine bestimmte Zeit danach | Rechnungsabgrenzungsposten ([§ 250 HGB][hgb250]) |
| antizipativ | im neuen Jahr | im alten Jahr | sonstige Forderung oder Verbindlichkeit, bei ungewisser Höhe Rückstellung |

Steuerlich gilt dasselbe ([§ 5 Abs. 5 Satz 1 EStG][estg5]).

## 2. Aktive Rechnungsabgrenzung

- Ausgaben vor dem Stichtag, soweit sie Aufwand für eine bestimmte Zeit nach
  dem Stichtag sind ([§ 250 Abs. 1 HGB][hgb250]). Die Zeit muss kalendermäßig
  bestimmt sein.
- Typisch: Versicherung, Software-Abonnement, Wartungsvertrag, Miete,
  Kfz-Steuer, jeweils im Voraus für einen Zeitraum über den Stichtag gezahlt.
- Suche: `buchungen_suchen` im letzten Quartal nach Aufwandsbuchungen, deren
  Leistungszeitraum über den Stichtag reicht; `beleg_lesen` zeigt den Zeitraum.
- Betrag: Nettobetrag (die Vorsteuer ist schon abgezogen) mal Anteil der Zeit
  nach dem Stichtag. Rechne nach Monaten oder taggenau, für alle Posten gleich.
- Buchung: Soll 1900 Aktive Rechnungsabgrenzung, Haben das ursprüngliche
  Aufwandskonto. Werkzeug: `abschlussbuchung` mit `art: abgrenzung` und
  `abgrenzung` (`art: aktiv`, `text`, `gesamtbetrag`, `von`, `bis`, `konto`,
  `quell_buchung_id` der Ursprungsbuchung; `abzugrenzen` nur bei einem
  anderen als dem zeitanteiligen Betrag).

## 3. Passive Rechnungsabgrenzung

- Einnahmen vor dem Stichtag, soweit sie Ertrag für eine bestimmte Zeit nach
  dem Stichtag sind ([§ 250 Abs. 2 HGB][hgb250]).
- Typisch: Jahreslizenz, Wartung oder Abonnement, im Voraus berechnet und
  bezahlt.
- Steht die Vorauszahlung in Buchfink als erhaltene Anzahlung (Abschlag,
  buchfink-rechnungen), ist sie schon eine Verbindlichkeit; dann entfällt der
  Posten. Steht sie als Erlös, grenze den Anteil nach dem Stichtag ab.
- Die Umsatzsteuer auf Vorauszahlungen regelt buchfink-rechnungen
  (Mindest-Istversteuerung); die Abgrenzung betrifft nur den Nettoerlös.
- Buchung: Soll das Erlöskonto der Rechnung, Haben 3900 Passive
  Rechnungsabgrenzung. Keine Steuerzeile. Werkzeug: `abschlussbuchung` mit
  `art: abgrenzung` und `abgrenzung.art: passiv`.

## 4. Kleine Beträge

- Steuerrecht: Der Posten darf unterbleiben, wenn die einzelne Ausgabe oder
  Einnahme 800 € nicht übersteigt (Betrag des [§ 6 Abs. 2 Satz 1 EStG][estg6]);
  das Wahlrecht gilt einheitlich für alle Posten ([§ 5 Abs. 5 Satz 2 EStG][estg5]).
  Stand: Oktober 2026.
- Handelsrecht: § 250 HGB kennt keine Grenze.
- Wertungsfrage: ob kleine Posten im Abschluss unterbleiben. Buchfink führt eine
  Bilanz für beide Rechte. Leg dem Menschen Anzahl und Summe der Posten bis
  800 € vor; er entscheidet einheitlich. Halte die Entscheidung in der Liste
  Abschlussposten fest und wende sie im Folgejahr wieder an
  ([§ 246 Abs. 3 HGB][hgb246]).

## 5. Ausstehende Rechnungen und Erträge

- Leistung im alten Jahr bezogen, Rechnung fehlt: Ist der Betrag sicher
  (Vertrag, Angebot, Monatspauschale), Soll Aufwandskonto, Haben 3500 Sonstige
  Verbindlichkeiten, netto. Ist er unsicher, Rückstellung (SKILL.md,
  Schritt 6).
- Die Vorsteuer entsteht erst mit der Rechnung; sie gehört nicht in die
  Abgrenzung. Kommt die Rechnung, bucht buchfink-belege sie normal; du löst die
  Abgrenzung gegen dasselbe Aufwandskonto auf.
- Zinsen für das alte Jahr, die die Bank im neuen belastet: Soll Zinsaufwand,
  Haben 3500 (buchfink-bank, Darlehen).
- Ertrag im alten Jahr, Zahlung im neuen (z. B. Zinsgutschrift, Erstattung
  einer Versicherung): Soll 1300 Sonstige Vermögensgegenstände, Haben
  Ertragskonto.
- Eigene Leistung im alten Jahr erbracht, Rechnung erst im neuen: Prüfe nach dem
  Ausstellen mit `buchungen_suchen`, in welchem Jahr der Erlös gebucht ist.
  Steht er im neuen Jahr, gehört er nach § 252 Abs. 1 Nr. 4, 5 HGB ins alte.
  Leg dem Menschen die Abgrenzung vor; das Konto klärt ihr gemeinsam. Die
  Umsatzsteuer ordnet Buchfink nach dem Leistungsdatum dem alten Zeitraum zu
  (buchfink-monatsabschluss).

## 6. Disagio

- Ein aktiviertes Disagio (1940 Damnum/Disagio, gebucht nach buchfink-bank)
  wird planmäßig über die Laufzeit des Darlehens abgeschrieben
  ([§ 250 Abs. 3 Satz 2 HGB][hgb250]). Abschlussbuchung: Soll 7323, Haben 1940,
  Jahresanteil.
- Ausweis gesondert oder im Anhang ([§ 268 Abs. 6 HGB][hgb268]); kleine
  Gesellschaften sind davon befreit ([§ 274a Nr. 3 HGB][hgb274a]).

## 7. Folgejahr

Buchfink löst Abgrenzungen, die nicht über seine Kartei gebucht wurden, nicht
von selbst auf. Trag jeden Posten in die Liste Abschlussposten ein. Nach dem
Saldenvortrag buchst du die Auflösung mit `buchung_erfassen` (Rückfrage):

- aktiv: Soll Aufwandskonto, Haben 1900, zum Beginn des Jahres oder über die
  Laufzeit verteilt;
- passiv: Soll 3900, Haben Erlöskonto;
- sonstige Verbindlichkeit: Soll 3500, Haben Aufwandskonto, sobald die Rechnung
  gebucht ist.

Fertig, wenn 1900, 3900 und die Abgrenzungen auf 3500 und 1300 aus dem Vorjahr
aufgelöst sind.

[hgb246]: https://www.gesetze-im-internet.de/hgb/__246.html
[hgb250]: https://www.gesetze-im-internet.de/hgb/__250.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[hgb274a]: https://www.gesetze-im-internet.de/hgb/__274a.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
