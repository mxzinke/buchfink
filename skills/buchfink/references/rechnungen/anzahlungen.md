# Anzahlungen, Abschläge, Schlussrechnung

Rechtsstand: 17. Oktober 2026.

Geld, das Kunden vor der Leistung zahlen. Anzahlungen an Lieferanten:
[belege/ablauf.md](../belege/ablauf.md).

## Inhalt

1. Wann die Steuer entsteht
2. Teilleistung oder Anzahlung
3. Abschlagsrechnung
4. Wie Buchfink bucht
5. Schlussrechnung und Restrechnung
6. Sonderfälle
7. Bilanz und Verzug

## 1. Wann die Steuer entsteht

- Vor der Leistung vereinnahmtes Entgelt: Steuer mit Ablauf des
  Voranmeldungszeitraums der Vereinnahmung, auch bei Sollversteuerung
  (Mindest-Istversteuerung, [§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG][ustg13],
  [UStAE 13.5 Abs. 1][ustae135]).
- Voraussetzung: Die Leistung steht bei Zahlung fest
  ([UStAE 13.5 Abs. 3][ustae135]). Ohne konkrete Vereinbarung kann es ein
  Darlehen sein; frag den Menschen.
- Ist die Leistung absehbar steuerfrei oder nicht steuerbar (etwa an ein
  EU-Unternehmen), entsteht auf die Anzahlung keine deutsche Steuer
  ([UStAE 13.5 Abs. 4][ustae135]).

## 2. Teilleistung oder Anzahlung

- Teilleistung: Eine wirtschaftlich teilbare Leistung wird in Teilen
  geschuldet und bewirkt, das Entgelt für bestimmte Teile ist gesondert
  vereinbart ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 2, 3 UStG][ustg13],
  [UStAE 13.4 Satz 1, 2][ustae134]). Die Leistung muss kontinuierlich oder
  wiederkehrend sein; eine einmalige Leistung gegen Raten ist keine
  Teilleistung ([UStAE 13.4 Satz 4, 5][ustae134]). Jeder Teil:
  `art: rechnung`.
- Abschlag auf ein Gesamtwerk ohne gesondert vereinbarte Teile: Anzahlung,
  `art: abschlag` im Rechnungsverbund.
- Wertungsfrage: Ob Meilensteine, Sprints oder Monatspauschalen Teilleistungen
  sind, folgt aus dem Vertrag. Lies ihn mit dem Menschen.

## 3. Abschlagsrechnung

- Vollwertige Rechnung mit allen Pflichtangaben
  ([§ 14 Abs. 5 Satz 1 UStG][ustg14]), auch E-Rechnungspflicht
  ([e-rechnung-ausgang.md, Abschnitt 2](e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist)).
- Sie weist aus, dass eine Anzahlung abgerechnet wird, und nennt den
  voraussichtlichen Zeitpunkt oder Kalendermonat der Leistung
  ([UStAE 14.8 Abs. 1, Abs. 4 Satz 3 bis 5][ustae148]); Tag der Vereinnahmung:
  [pflichtangaben-ausgang.md, Abschnitt 2](pflichtangaben-ausgang.md#2-pflichtangaben-und-woher-buchfink-sie-nimmt),
  Nr. 6.
- Ausstellen vor dem Geldeingang ist zulässig. Steuer entsteht nur auf das
  Gezahlte; zahlt der Kunde nicht, entsteht keine Steuer nach § 14c UStG
  ([UStAE 14.8 Abs. 2, Abs. 5 Satz 3 bis 5][ustae148]).

## 4. Wie Buchfink bucht

| Schritt | Buchung (SKR04) |
|---|---|
| Abschlagsrechnung ausstellen | keine Buchung |
| Zahlung geht ein | Bank an 3272 Erhaltene, versteuerte Anzahlungen 19 % USt und 3806 Umsatzsteuer 19 % (bei 7 %: 3260, 3801) |
| Schlussrechnung | Debitor (Rest), 3272 und 3806 (Auflösung) an Erlöse (gesamt netto) und 3806 (Steuer auf das Gesamtentgelt) |

Aufruf: [ablauf.md, Abschnitt 5](ablauf.md#5-abschläge-anzahlungen-schlussrechnung).

Offene Abschläge stehen in `offene_posten` mit `abschlag_id` und in den
Bankvorschlägen. Gib `abschlag_id` statt `buchung_id` an: bei Routine mit
`bankumsatz_zuordnen`, bei Teilzahlung oder Aufteilung nach Rückfrage mit
`bankumsatz_buchen`. `ausgleich` nennt den gezahlten Teilbetrag. Buchfink
berechnet Anzahlung und Steuer selbst.

## 5. Schlussrechnung und Restrechnung

- Buchfink setzt in der Schlussrechnung alle berechneten und vereinnahmten
  Abschläge des Verbunds mit dem vereinnahmten Teilbetrag und dessen Steuer ab
  ([§ 14 Abs. 5 Satz 2 UStG][ustg14], [UStAE 14.8 Abs. 7][ustae148]). Prüfe
  vorher mit dem Menschen, ob jede Zahlung im Verbund steht: Fehlt eine
  Absetzung, wird die zu viel ausgewiesene Steuer nach § 14c Abs. 1 UStG
  geschuldet ([UStAE 14.8 Abs. 10][ustae148]).
- Eine Restrechnung über das verbleibende Entgelt ist zulässig
  ([UStAE 14.8 Abs. 11][ustae148]); Buchfink erzeugt die Endrechnung.
- E-Rechnung: Das BMF empfiehlt eine Restrechnung und beanstandet bis
  31.12.2027 eine Endrechnung mit unstrukturiertem Anhang nicht
  ([BMF 15.10.2024, Rn. 48][bmf2024]). Beanstandet der Kunde die Endrechnung,
  leg es dem Menschen vor.

## 6. Sonderfälle

- **Anzahlung ohne Rechnung.** Steuer entsteht trotzdem (Abschnitt 1); Frist:
  [pflichtangaben-ausgang.md, Abschnitt 1](pflichtangaben-ausgang.md#1-wann-eine-rechnung-pflicht-ist).
  Stell die Abschlagsrechnung nachträglich im Verbund aus
  (`zahlung_erhalten_am`); ordne anschließend die Zahlung zu (Abschnitt 4).
- **Auftrag fällt weg, Anzahlung zurück.** Steuer im Zeitraum der Rückzahlung
  berichtigen ([§ 17 Abs. 2 Nr. 2 UStG][ustg17], [BFH V R 34/09, Leitsatz 1][bfh3409]).
  Frag, ob die Leistung nicht ausgeführt wurde und das Geld tatsächlich
  zurückgeflossen ist. Nach dem Ja: `bankumsatz_buchen` für den Bankabgang mit
  `abschlag_id`, `ausgleich` und Grund in `text`. Die Rechnung bleibt danach
  offen und muss bei weggefallenem Auftrag storniert werden. Zurückgezahlte
  Beträge werden in der Schlussrechnung nicht abgesetzt. Bereits verrechnete
  Abschläge erfordern zuerst eine Berichtigung der Schlussrechnung.
- **EU-Unternehmen.** Keine deutsche Steuer auf die Anzahlung, Steuerfall wie
  die Leistung ([ablauf.md, Abschnitt 3](ablauf.md#3-steuerfall)).

## 7. Bilanz und Verzug

- Erhaltene Anzahlungen stehen unter den Verbindlichkeiten
  ([§ 266 Abs. 3 C. Nr. 3 HGB][hgb266], [§ 268 Abs. 5 Satz 2 HGB][hgb268]);
  unfertige Leistungen:
  [jahresabschluss/unfertige-leistungen.md](../jahresabschluss/unfertige-leistungen.md).
- Abschläge im Verzug: [mahnen-verzug.md, Abschnitt 3](mahnen-verzug.md#3-pauschale-und-weitere-kosten),
  [pflichtangaben-ausgang.md, Abschnitt 5](pflichtangaben-ausgang.md#5-entgelt-skonto-zahlungsziel).

[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[hgb266]: https://www.gesetze-im-internet.de/hgb/__266.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[ustae134]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/Umsatzsteuer-Anwendungserlass-aktuell.html
[ustae135]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/Umsatzsteuer-Anwendungserlass-aktuell.html
[ustae148]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/Umsatzsteuer-Anwendungserlass-aktuell.html
[ustae171]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/Umsatzsteuer-Anwendungserlass-aktuell.html

[bfh3409]: https://www.bundesfinanzhof.de/de/entscheidung/entscheidungen-online/detail/pdf/STRE201010349?type=1646225765
