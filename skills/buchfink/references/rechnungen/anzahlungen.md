# Anzahlungen, Abschläge, Schlussrechnung

Rechtsstand: 17. Oktober 2026.

Anzahlungen an Lieferanten bucht [belege/ablauf.md](../belege/ablauf.md); diese
Referenz betrifft Geld, das Kunden vor der Leistung zahlen.

## Inhalt

1. Wann die Steuer entsteht
2. Teilleistung oder Anzahlung
3. Abschlagsrechnung
4. Wie Buchfink bucht
5. Schlussrechnung und Restrechnung
6. Sonderfälle
7. Bilanz und Verzug

## 1. Wann die Steuer entsteht

- Wird Entgelt vor der Leistung vereinnahmt, entsteht die Steuer mit Ablauf des
  Voranmeldungszeitraums der Vereinnahmung, auch bei Sollversteuerung
  ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG][ustg13],
  [UStAE 13.5 Abs. 1][ustae135]). Man nennt das Mindest-Istversteuerung.
- Voraussetzung: Die künftige Leistung steht bei der Zahlung genau fest
  ([UStAE 13.5 Abs. 3][ustae135]). Fehlt eine konkrete Vereinbarung, kann die
  Zahlung ein Darlehen sein. Wertungsfrage, frag den Menschen.
- Ist absehbar, dass die Leistung steuerfrei oder nicht steuerbar ist (etwa an
  ein Unternehmen im EU-Ausland), entsteht keine deutsche Steuer auf die
  Anzahlung ([UStAE 13.5 Abs. 4][ustae135]).

## 2. Teilleistung oder Anzahlung

- Teilleistungen liegen vor, wenn für bestimmte Teile einer wirtschaftlich
  teilbaren Leistung das Entgelt gesondert vereinbart ist
  ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 2, 3 UStG][ustg13]). Jeder Teil ist mit
  seiner Vollendung ausgeführt und wird als normale Rechnung abgerechnet
  ([UStAE 13.4][ustae134]).
- Eine Abschlagszahlung auf ein Gesamtwerk ohne gesondert vereinbarte Teile ist
  eine Anzahlung.
- Wertungsfrage: Ob Meilensteine, Sprints oder Monatspauschalen Teilleistungen
  sind, folgt aus dem Vertrag. Lies ihn mit dem Menschen. Teilleistung heißt:
  `rechnung_ausstellen` mit `art: rechnung`. Anzahlung heißt:
  Abschlagsrechnung im Rechnungsverbund (`art: abschlag`).

## 3. Abschlagsrechnung

- Sie ist eine vollwertige Rechnung mit allen Pflichtangaben
  ([§ 14 Abs. 5 Satz 1 UStG][ustg14]). Auch die E-Rechnungspflicht an ein
  inländisches Unternehmen gilt für sie
  ([e-rechnung-ausgang.md, Abschnitt 2](e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist)).
- Sie zeigt, dass eine Anzahlung abgerechnet wird, z. B. durch den
  voraussichtlichen Leistungszeitpunkt ([UStAE 14.8 Abs. 1][ustae148]).
- Statt des Leistungszeitpunkts steht der voraussichtliche Zeitpunkt oder
  Kalendermonat der Leistung darin ([UStAE 14.8 Abs. 4 Satz 3 bis 5][ustae148]);
  zum Tag der Vereinnahmung siehe
  [pflichtangaben-ausgang.md, Abschnitt 2](pflichtangaben-ausgang.md#2-pflichtangaben-und-woher-buchfink-sie-nimmt),
  Nr. 6.
- Sie darf vor dem Geldeingang ausgestellt werden. Zahlt der Kunde weniger,
  entsteht Steuer nur auf den gezahlten Teil; zahlt er gar nicht, wird keine
  Steuer nach § 14c geschuldet ([UStAE 14.8 Abs. 2, Abs. 5 Satz 3 bis 5][ustae148]).

## 4. Wie Buchfink bucht

Buchfink führt Abschläge im Rechnungsverbund: vereinbarter Gesamtbetrag netto,
Abschlagsrechnungen, Zahlungsstand, Schlussrechnung.

| Schritt | Buchung (SKR04) |
|---|---|
| Abschlagsrechnung ausstellen | keine Buchung |
| Zahlung geht ein | Bank an 3272 Erhaltene, versteuerte Anzahlungen 19 % USt und 3806 Umsatzsteuer 19 % (bei 7 %: 3260, 3801) |
| Schlussrechnung | Debitor (Rest), 3272 und 3806 (Auflösung) an Erlöse (gesamt netto) und 3806 (Steuer auf das Gesamtentgelt) |

Abschlag und Schlussrechnung stellst du nach Rückfrage mit
`rechnung_ausstellen` aus; die Angaben stehen in
[ablauf.md, Abschnitt 5](ablauf.md#5-abschläge-anzahlungen-schlussrechnung).

Den Zahlungseingang auf einen Abschlag bildet Buchfink noch nicht ab, weder
als Werkzeug noch in der App: der offene Abschlag fehlt in `offene_posten` und
in den Bankvorschlägen. Solange ein Abschlag nicht vereinnahmt ist, lehnt
Buchfink die Schlussrechnung ab, und stornieren lässt sich die
Abschlagsrechnung auch nicht
([storno-berichtigung.md, Abschnitt 1](storno-berichtigung.md#1-was-buchfink-anbietet)).
Sag das dem Menschen, bevor du die erste Abschlagsrechnung eines Verbunds
ausstellst. Geht die Zahlung ein, weiche nicht auf `bankumsatz_buchen` oder
`buchung_erfassen` aus
([grundlagen/arbeitsweise.md, Abschnitt 9](../grundlagen/arbeitsweise.md#9-vorgänge-ohne-werkzeug)).

## 5. Schlussrechnung und Restrechnung

- Die Endrechnung setzt die vor der Leistung vereinnahmten Teilentgelte und die
  darauf entfallende Steuer ab, soweit über sie Rechnungen mit Steuerausweis
  ausgestellt wurden ([§ 14 Abs. 5 Satz 2 UStG][ustg14]). Es genügt der
  Gesamtbetrag der Anzahlungen und die Summe der Steuer
  ([UStAE 14.8 Abs. 7][ustae148]).
- Auch zurückgenommene oder widerrufene Anzahlungsrechnungen werden abgesetzt
  ([UStAE 14.8 Abs. 9][ustae148]).
- Fehlt die Absetzung ganz oder teilweise, schuldet das Unternehmen die zu viel
  ausgewiesene Steuer zusätzlich nach § 14c Abs. 1 UStG, bis es die Endrechnung
  berichtigt ([UStAE 14.8 Abs. 10][ustae148]).
- Statt der Endrechnung ist eine Restrechnung über das verbleibende Entgelt
  zulässig ([UStAE 14.8 Abs. 11][ustae148]). Buchfink erzeugt die
  Schlussrechnung mit Absetzung.
- E-Rechnung: Nach Verwaltungsauffassung lässt sich die Absetzung im
  strukturierten Teil derzeit nicht vollständig darstellen. Das BMF empfiehlt
  deshalb eine Restrechnung und beanstandet bis 31.12.2027 eine Endrechnung mit
  unstrukturiertem Anhang nicht ([BMF 15.10.2024, Rn. 48][bmf2024]). Buchfink
  gibt den Gesamtbetrag der Anzahlungen im Datensatz an (BT-113) und verweist auf
  die Abschlagsrechnungen. Ob das im Einzelfall genügt, ist offen; lege den
  Punkt dem Menschen vor, wenn der Kunde die Schlussrechnung beanstandet.
- Buchfink setzt in der Schlussrechnung alle berechneten und vereinnahmten
  Abschläge des Verbunds ab. Ein bezahlter Abschlag ohne Abschlagsrechnung fehlt
  dort; prüfe vor der Schlussrechnung mit dem Menschen, ob jede Zahlung im
  Verbund steht. `offene_posten` zeigt die Abschläge nicht (Abschnitt 4).

## 6. Sonderfälle

- **Anzahlung ohne Rechnung.** Die Steuer entsteht trotzdem mit der
  Vereinnahmung (Abschnitt 1). Frist für die Rechnung:
  [pflichtangaben-ausgang.md, Abschnitt 1](pflichtangaben-ausgang.md#1-wann-eine-rechnung-pflicht-ist).
  Weg: Du stellst die Abschlagsrechnung nachträglich im Verbund aus
  (`zahlung_erhalten_am`). Die Zahlung ihr zuzuordnen bildet Buchfink noch
  nicht ab (Abschnitt 4).
- **Auftrag fällt weg, Anzahlung wird zurückgezahlt.** Die Steuer auf die
  Anzahlung wird im Zeitraum der Rückzahlung berichtigt
  ([§ 17 Abs. 2 Nr. 2 UStG][ustg17], [UStAE 17.1 Abs. 7][ustae171]). Die
  Rückzahlung im Verbund bildet Buchfink noch nicht ab; sag das dem Menschen.
- **Leistung an Unternehmen im EU-Ausland.** Keine deutsche Steuer auf die
  Anzahlung (Abschnitt 1), Steuerfall wie die Leistung
  ([ablauf.md, Abschnitt 3](ablauf.md#3-steuerfall)).

## 7. Bilanz und Verzug

- Erhaltene Anzahlungen stehen unter den Verbindlichkeiten
  ([§ 266 Abs. 3 C. Nr. 3 HGB][hgb266], [§ 268 Abs. 5 Satz 2 HGB][hgb268]).
  Unfertige Leistungen zum Stichtag regelt
  [jahresabschluss/unfertige-leistungen.md](../jahresabschluss/unfertige-leistungen.md).
- Verzugspauschale bei Abschlagszahlungen:
  [mahnen-verzug.md, Abschnitt 3](mahnen-verzug.md#3-pauschale-und-weitere-kosten);
  die 60-Tage-Grenze für Zahlungsfristen bei Abschlägen:
  [pflichtangaben-ausgang.md, Abschnitt 5](pflichtangaben-ausgang.md#5-entgelt-skonto-zahlungsziel).

[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[hgb266]: https://www.gesetze-im-internet.de/hgb/__266.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[ustae134]: https://www.steuerschroeder.de/steuergesetze/ustae/13.4
[ustae135]: https://www.steuerschroeder.de/steuergesetze/ustae/13.5
[ustae148]: https://www.steuerschroeder.de/steuergesetze/ustae/14.8
[ustae171]: https://www.steuerschroeder.de/steuergesetze/ustae/17.1
