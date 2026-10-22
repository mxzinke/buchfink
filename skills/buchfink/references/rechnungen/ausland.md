# Kunden im Ausland

Rechtsstand: 17. Oktober 2026.

Gilt für sonstige Leistungen. Innergemeinschaftliche Lieferung, Ausfuhr und
Rechnungen in fremder Währung bildet Buchfink nicht ab
(`rechnung_ausstellen` kennt nur Euro); sprich die Grenze an und empfiehl
einen Steuerberater.

## Inhalt

1. Erst den Ort der Leistung bestimmen
2. Unternehmen im übrigen EU-Gebiet
3. Unternehmen im Drittland
4. Verbraucher im Ausland
5. Ausnahmen vom Grundsatz

## 1. Erst den Ort der Leistung bestimmen

Deutsche Umsatzsteuer fällt nur an, wenn der Ort der Leistung im Inland liegt.

| Kunde | Grundsatz | Fundstelle |
|---|---|---|
| Unternehmer, Leistung für sein Unternehmen | Ort, von dem aus der Kunde sein Unternehmen betreibt | [§ 3a Abs. 2 Satz 1 UStG][ustg3a] |
| Juristische Person ohne Unternehmen, aber mit USt-IdNr. | wie Unternehmer | [§ 3a Abs. 2 Satz 3 UStG][ustg3a] |
| Verbraucher | Ort, von dem aus die GmbH ihr Unternehmen betreibt | [§ 3a Abs. 1 UStG][ustg3a] |

Abschnitt 5 geht dem Grundsatz vor.

## 2. Unternehmen im übrigen EU-Gebiet

- Ort beim Kunden, der Kunde schuldet die Steuer:
  `steuerfall: reverse_charge_supply`. Hinweis und beide USt-IdNrn. setzt
  Buchfink ([pflichtangaben-ausgang.md, Abschnitt 6](pflichtangaben-ausgang.md#6-hinweise-in-sonderfällen)).
- Rechnung bis zum 15. des Folgemonats ([§ 14a Abs. 1 Satz 2 UStG][ustg14a]).
- Unternehmereigenschaft: Verwendet der Kunde seine USt-IdNr. und ist sie samt
  Name und Anschrift beim BZSt bestätigt, gilt er als Unternehmer
  ([§ 18e Nr. 1 UStG][ustg18e], [UStAE 3a.2 Abs. 9 Satz 4, 5][ustae3a2]).
  Buchfink fragt die Bestätigung vor dem Ausstellen ab; bei ungültiger Nummer
  fragt der Mensch beim Kunden nach.
- Bei Leistungen, die wahrscheinlich privat genutzt werden, braucht es
  zusätzlich eine Erklärung des Kunden zur unternehmerischen Verwendung
  ([UStAE 3a.2 Abs. 11a][ustae3a2]).
- Meldung: Voranmeldung Kennzahl 21
  ([monatsabschluss/ustva.md](../monatsabschluss/ustva.md)) und ZM
  ([monatsabschluss/zm.md](../monatsabschluss/zm.md)). Keine E-Rechnungspflicht
  ([e-rechnung-ausgang.md, Abschnitt 2](e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist)).

## 3. Unternehmen im Drittland

- Ort beim Kunden, nicht steuerbar ([§ 3a Abs. 2 UStG][ustg3a]):
  `steuerfall: not_taxable`.
- Nachweis der Unternehmereigenschaft: Bescheinigung einer Behörde des
  Sitzstaates oder ein anderer Nachweis ([UStAE 3a.2 Abs. 11][ustae3a2]). Frag,
  welcher vorliegt, und lass ihn als Beleg ablegen.
- Pflichten im Staat des Kunden prüft Buchfink nicht; weise darauf hin.

## 4. Verbraucher im Ausland

| Leistung | Ort | `steuerfall` |
|---|---|---|
| Beratung, Agentur, Projektarbeit an Verbraucher in der EU | Inland ([§ 3a Abs. 1 UStG][ustg3a]) | `domestic` |
| Katalogleistung nach [§ 3a Abs. 4 Satz 2 UStG][ustg3a] (u. a. Beratung, Werbung, Datenverarbeitung, Überlassung von Rechten und Informationen, Personalgestellung) an Verbraucher im Drittland | Wohnsitz des Kunden, nicht steuerbar | `not_taxable` |
| Andere Leistung an Verbraucher im Drittland | Inland | `domestic` |
| Telekommunikation, Rundfunk, elektronische Leistung (SaaS, Download, App) an Verbraucher in der EU | Wohnsitz des Kunden ([§ 3a Abs. 5 Satz 1, 2 UStG][ustg3a]) | Grenze, siehe unten |

Elektronische Leistungen an EU-Verbraucher bleiben im Inland steuerbar,
solange die Summe dieser Leistungen und der Fernverkäufe in andere EU-Staaten
im Vorjahr und im laufenden Jahr 10.000 € nicht übersteigt
([§ 3a Abs. 5 Satz 3 UStG][ustg3a]). Darüber schuldet die GmbH die Steuer des
Wohnsitzstaates, meist über OSS ([§ 18j UStG][ustg18j]); OSS bildet Buchfink
nicht ab ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
Rechne mit `auswertung` und `buchungen_suchen` nach, wie nah die Schwelle ist,
und empfiehl beim Überschreiten einen Steuerberater.

## 5. Ausnahmen vom Grundsatz

| Leistung | Ort | Fundstelle |
|---|---|---|
| Leistung im Zusammenhang mit einem Grundstück (z. B. Planung, Bauleitung, Vermittlung) | Lage des Grundstücks | [§ 3a Abs. 3 Nr. 1 UStG][ustg3a] |
| Eintritt zu Veranstaltungen, Messen an Unternehmer | Ort der Veranstaltung; virtuelle Teilnahme nach Abs. 2 | [§ 3a Abs. 3 Nr. 5 UStG][ustg3a] |
| Kulturelle, unterrichtende, unterhaltende Leistungen an Verbraucher | Ort der Tätigkeit; Streaming beim Kunden | [§ 3a Abs. 3 Nr. 3 UStG][ustg3a] |
| Vermittlung an Verbraucher | Ort des vermittelten Umsatzes | [§ 3a Abs. 3 Nr. 4 UStG][ustg3a] |

Liegt der Ort danach im Ausland, entsteht dort Steuer, die Buchfink nicht
abbildet; leg den Fall dem Menschen als Grenze vor.

[ustg3a]: https://www.gesetze-im-internet.de/ustg_1980/__3a.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg18e]: https://www.gesetze-im-internet.de/ustg_1980/__18e.html
[ustg18j]: https://www.gesetze-im-internet.de/ustg_1980/__18j.html
[ustae3a2]: https://www.steuerschroeder.de/steuergesetze/ustae/3a.2
