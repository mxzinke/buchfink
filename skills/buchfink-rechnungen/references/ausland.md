# Kunden im Ausland

Rechtsstand: 17. Oktober 2026. Gilt für sonstige Leistungen (Beratung,
Software, Agentur, Projekte). Warenlieferungen ins Ausland bildet dieser Skill
nicht ab; bei einer innergemeinschaftlichen Lieferung oder Ausfuhr sprich die
Grenze an und empfiehl einen Steuerberater.

## Inhalt

1. Erst den Ort der Leistung bestimmen
2. Unternehmen im übrigen EU-Gebiet
3. Unternehmen im Drittland
4. Verbraucher im Ausland
5. Ausnahmen vom Grundsatz
6. Prüfliste vor `rechnung_ausstellen`

## 1. Erst den Ort der Leistung bestimmen

Deutsche Umsatzsteuer fällt nur an, wenn der Ort der Leistung im Inland liegt.

| Kunde | Grundsatz | Fundstelle |
|---|---|---|
| Unternehmer, Leistung für sein Unternehmen | Ort, von dem aus der Kunde sein Unternehmen betreibt | [§ 3a Abs. 2 Satz 1 UStG][ustg3a] |
| Juristische Person ohne Unternehmen, aber mit USt-IdNr. | wie Unternehmer | [§ 3a Abs. 2 Satz 3 UStG][ustg3a] |
| Verbraucher | Ort, von dem aus die GmbH ihr Unternehmen betreibt | [§ 3a Abs. 1 UStG][ustg3a] |

Die Ausnahmen in Abschnitt 5 gehen dem Grundsatz vor.

## 2. Unternehmen im übrigen EU-Gebiet

- Ort beim Kunden. Der Kunde schuldet die Steuer in seinem Staat.
- `steuerfall: reverse_charge_supply`. Buchfink weist keine Steuer aus, setzt
  den Hinweis „Steuerschuldnerschaft des Leistungsempfängers“ und beide
  USt-IdNrn. ([§ 14a Abs. 1 Satz 1, 3 UStG][ustg14a]).
- Rechnung bis zum 15. des Monats nach der Leistung
  ([§ 14a Abs. 1 Satz 2 UStG][ustg14a]).
- Nachweis der Unternehmereigenschaft: Verwendet der Kunde für den Umsatz seine
  USt-IdNr. und hat das Unternehmen deren Gültigkeit samt Name und Anschrift
  beim BZSt bestätigen lassen, darf es von einem Unternehmer ausgehen
  ([§ 18e Nr. 1 UStG][ustg18e], [UStAE 3a.2 Abs. 9 Satz 4, 5][ustae3a2]).
  Buchfink fragt die Bestätigung vor dem Ausstellen ab. Lehnt es wegen einer
  ungültigen Nummer ab, lass den Menschen beim Kunden nachfragen.
- Bei Leistungen, die mit hoher Wahrscheinlichkeit privat genutzt werden,
  genügt die USt-IdNr. nicht; nötig ist eine Erklärung des Kunden zur
  unternehmerischen Verwendung ([UStAE 3a.2 Abs. 11a][ustae3a2]).
- Der Umsatz geht in die Voranmeldung (Kennzahl 21) und in die
  Zusammenfassende Meldung ([§ 18a UStG][ustg18a]); beides regelt
  buchfink-monatsabschluss.
- Eine E-Rechnung ist keine Pflicht, weil der Kunde nicht im Inland ansässig
  ist ([UStAE 14.1 Abs. 6 Satz 3 i. d. F. vom 15.10.2025][bmf2025]).

## 3. Unternehmen im Drittland

- Ort beim Kunden, in Deutschland nicht steuerbar
  ([§ 3a Abs. 2 UStG][ustg3a]).
- `steuerfall: not_taxable`. Buchfink kennzeichnet den Umsatz als nicht
  steuerbar; die Voranmeldung führt ihn unter den nicht steuerbaren Umsätzen.
- Nachweis der Unternehmereigenschaft: Bescheinigung einer Behörde des
  Sitzstaates, sonst ein anderer Nachweis nach Wahl des Unternehmens
  ([UStAE 3a.2 Abs. 11][ustae3a2]). Frag den Menschen, welcher Nachweis vorliegt,
  und lass ihn als Beleg ablegen.
- Welche Rechnungs- oder Steuerpflichten im Staat des Kunden gelten, prüft
  Buchfink nicht. Weise darauf hin.

## 4. Verbraucher im Ausland

| Leistung | Ort | `steuerfall` |
|---|---|---|
| Beratung, Agentur, Projektarbeit an Verbraucher in der EU | Inland ([§ 3a Abs. 1 UStG][ustg3a]) | `domestic` |
| Katalogleistung nach [§ 3a Abs. 4 Satz 2 UStG][ustg3a] (u. a. Beratung, Werbung, Datenverarbeitung, Überlassung von Rechten und Informationen, Personalgestellung) an Verbraucher im Drittland | Wohnsitz des Kunden, nicht steuerbar | `not_taxable` |
| Andere Leistung an Verbraucher im Drittland | Inland | `domestic` |
| Telekommunikation, Rundfunk, auf elektronischem Weg erbrachte Leistung (SaaS, Download, App) an Verbraucher in der EU | Wohnsitz des Kunden ([§ 3a Abs. 5 Satz 1, 2 UStG][ustg3a]) | Grenze, siehe unten |

Elektronische Leistungen an Verbraucher in anderen EU-Staaten:

- Der Ort bleibt im Inland, solange das Unternehmen nur in Deutschland ansässig
  ist und die Summe dieser Leistungen und der Fernverkäufe in andere EU-Staaten
  im Vorjahr 10.000 € nicht überschritten hat und im laufenden Jahr nicht
  überschreitet ([§ 3a Abs. 5 Satz 3 UStG][ustg3a]). Stand: Oktober 2026. Das
  Unternehmen kann darauf verzichten und bleibt zwei Jahre gebunden
  ([§ 3a Abs. 5 Satz 4, 5 UStG][ustg3a]).
- Über der Schwelle ist die Steuer des Wohnsitzstaates geschuldet, meist über
  das OSS-Verfahren ([§ 18j UStG][ustg18j]). Buchfink bildet OSS nicht ab
  (buchfink-grundlagen, Grenzen). Rechne mit dem Menschen nach
  `auswertung` und `buchungen_suchen`, wie nah die Schwelle ist, und empfiehl
  bei Überschreiten einen Steuerberater.

## 5. Ausnahmen vom Grundsatz

| Leistung | Ort | Fundstelle |
|---|---|---|
| Leistung im Zusammenhang mit einem Grundstück (z. B. Planung, Bauleitung, Vermittlung) | Lage des Grundstücks | [§ 3a Abs. 3 Nr. 1 UStG][ustg3a] |
| Eintritt zu Veranstaltungen, Messen an Unternehmer | Ort der Veranstaltung; virtuelle Teilnahme nach Abs. 2 | [§ 3a Abs. 3 Nr. 5 UStG][ustg3a] |
| Kulturelle, unterrichtende, unterhaltende Leistungen an Verbraucher | Ort der Tätigkeit; Streaming beim Kunden | [§ 3a Abs. 3 Nr. 3 UStG][ustg3a] |
| Vermittlung an Verbraucher | Ort des vermittelten Umsatzes | [§ 3a Abs. 3 Nr. 4 UStG][ustg3a] |

Liegt der Ort danach im Ausland, entsteht dort Steuer, die Buchfink nicht
abbildet. Lege den Fall dem Menschen als Grenze vor.

## 6. Prüfliste vor `rechnung_ausstellen`

- Land, Unternehmer oder Verbraucher und USt-IdNr. des Kontakts stimmen;
  Buchfink bestätigt die USt-IdNr. beim Ausstellen.
- Leistung ist eine sonstige Leistung und fällt unter keine Ausnahme aus
  Abschnitt 5.
- Steuerfall nach der Tabelle in SKILL.md Abschnitt 3 gewählt.
- Rechnungsdatum liegt spätestens am 15. des Folgemonats (EU-Unternehmen).
- `rechnung_ausstellen` kennt nur Euro. Verlangt der Kunde eine andere
  Währung, sprich die Grenze an.

[ustg3a]: https://www.gesetze-im-internet.de/ustg_1980/__3a.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg18a]: https://www.gesetze-im-internet.de/ustg_1980/__18a.html
[ustg18e]: https://www.gesetze-im-internet.de/ustg_1980/__18e.html
[ustg18j]: https://www.gesetze-im-internet.de/ustg_1980/__18j.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
[ustae3a2]: https://www.steuerschroeder.de/steuergesetze/ustae/3a.2
