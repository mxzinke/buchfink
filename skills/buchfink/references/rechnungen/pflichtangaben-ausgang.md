# Pflichtangaben einer Ausgangsrechnung

Rechtsstand: 17. Oktober 2026.

Die Eingangsseite prüft [belege/pflichtangaben.md](../belege/pflichtangaben.md).

## Inhalt

1. Wann eine Rechnung Pflicht ist
2. Pflichtangaben und woher Buchfink sie nimmt
3. Kleinbetragsrechnung
4. Leistungszeitpunkt
5. Entgelt, Skonto, Zahlungsziel
6. Hinweise in Sonderfällen

## 1. Wann eine Rechnung Pflicht ist

- Binnen sechs Monaten nach der Leistung an ein Unternehmen für dessen
  Unternehmen oder an eine juristische Person, außer bei Umsätzen nach § 4
  Nr. 8 bis 29 UStG ([§ 14 Abs. 2 Satz 2 Nr. 1, 2 UStG][ustg14]).
- An Verbraucher keine Pflicht nach dem UStG (außer Grundstücksleistungen,
  [§ 14 Abs. 2 Satz 2 Nr. 3 UStG][ustg14]). Stelle die Rechnung trotzdem aus;
  sie ist der Beleg der Erlösbuchung ([§ 238 Abs. 1 HGB][hgb238]).
- Anzahlungen: dieselben Regeln ([§ 14 Abs. 5 Satz 1 UStG][ustg14]), Frist
  sechs Monate ab Vereinnahmung
  ([UStAE 14.1 Abs. 3 Satz 4 i. d. F. vom 15.10.2025][bmf2025]).

## 2. Pflichtangaben und woher Buchfink sie nimmt

[§ 14 Abs. 4 Satz 1 UStG][ustg14]:

| Nr. | Angabe | In Buchfink |
|---|---|---|
| 1 | Vollständiger Name und Anschrift von Unternehmen und Kunde | Stammdaten, Kontakt |
| 2 | Steuernummer oder USt-IdNr. des Unternehmens | Stammdaten; fehlt sie, meldet es `rechnung_vorschau` |
| 3 | Ausstellungsdatum | `rechnungsdatum`: Tag der Ausstellung, nie rückdatieren |
| 4 | Fortlaufende, einmalige Rechnungsnummer | vergibt Buchfink aus dem Nummernkreis |
| 5 | Menge und Art der Lieferung, Umfang und Art der sonstigen Leistung | `positionen[].text`, `menge`, `einheit`; Leistung eindeutig beschreiben |
| 6 | Zeitpunkt der Leistung; bei Anzahlung Tag der Vereinnahmung, wenn bekannt und abweichend vom Rechnungsdatum | `leistung_von`, `leistung_bis` (Abschnitt 4) |
| 7 | Entgelt nach Steuersätzen und Befreiungen, jede im Voraus vereinbarte Minderung | Positionen, Zahlungsbedingungen (Abschnitt 5) |
| 8 | Steuersatz und Steuerbetrag oder Hinweis auf die Befreiung | `satz`, `steuerfall` ([ablauf.md, Abschnitt 3](ablauf.md#3-steuerfall)) |
| 9 | Hinweis auf die Aufbewahrungspflicht des Empfängers (§ 14b Abs. 1 Satz 5 UStG) | nur bei Grundstücksleistungen an Nichtunternehmer; Mensch fragen |
| 10 | Angabe „Gutschrift“ | Buchfink stellt keine Gutschriften aus |

- Mehrere Dokumente, Abkürzungen, Kalendermonat als Leistungszeitpunkt
  ([§ 31 Abs. 1 bis 4 UStDV][ustdv31]):
  [belege/pflichtangaben.md, Abschnitt 4](../belege/pflichtangaben.md#4-mehrere-dokumente-abkürzungen-leistungsmonat).
  E-Rechnung: alle Angaben im strukturierten Teil
  ([e-rechnung-ausgang.md, Abschnitt 5](e-rechnung-ausgang.md#5-inhalt-des-strukturierten-teils)).
- Fehlt eine Pflichtangabe, verliert der Kunde den Vorsteuerabzug
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]); Berichtigung nach
  [storno-berichtigung.md](storno-berichtigung.md).

## 3. Kleinbetragsrechnung

- Bis 250 € Gesamtbetrag ([§ 33 UStDV][ustdv33]):
  [belege/pflichtangaben.md, Abschnitt 3](../belege/pflichtangaben.md#3-erleichterungen).
- `rechnung_ausstellen` erzeugt immer die vollständige Rechnung. Die Grenze
  zählt nur für die E-Rechnungspflicht
  ([e-rechnung-ausgang.md, Abschnitt 2](e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist)).

## 4. Leistungszeitpunkt

- Pflichtangabe auch bei Gleichheit mit dem Rechnungsdatum
  ([§ 14 Abs. 4 Satz 1 Nr. 6 UStG][ustg14]).
- Sonstige Leistungen sind mit Vollendung ausgeführt, Dauerleistungen ohne
  vereinbarte Teilleistungen mit Ende des Rechtsverhältnisses
  ([UStAE 13.1 Abs. 3][ustae131]).
- Er bestimmt den Voranmeldungszeitraum der Steuer
  ([monatsabschluss/ustva.md, Abschnitt 1](../monatsabschluss/ustva.md#1-pflicht-und-form));
  Anzahlungen: [anzahlungen.md, Abschnitt 1](anzahlungen.md#1-wann-die-steuer-entsteht).
- Wertungsfrage bei Projekten: Abnahme, Übergabe, Go-live oder Ende des
  Zeitraums. Frag, was vereinbart und was geschehen ist.

## 5. Entgelt, Skonto, Zahlungsziel

- Im Voraus vereinbarter Skonto ist Pflichtangabe
  ([§ 14 Abs. 4 Satz 1 Nr. 7 UStG][ustg14]); „2 % Skonto bei Zahlung binnen
  10 Tagen“ genügt (`skonto_promille: 20`, `skonto_tage: 10`).
- Gezogener Skonto mindert die Bemessungsgrundlage ([§ 17 Abs. 1 UStG][ustg17])
  ohne Rechnungsberichtigung
  ([storno-berichtigung.md, Abschnitt 2](storno-berichtigung.md#2-welcher-weg-für-welchen-fehler));
  Buchung: [bank/skonto-differenzen.md](../bank/skonto-differenzen.md).
- `zahlungsziel_tage` zählt ab Rechnungsdatum. Mehr als 60 Tage nach Empfang
  der Leistung wirken gegenüber Unternehmen nur ausdrücklich vereinbart und
  nicht grob unbillig ([§ 271a Abs. 1 BGB][bgb271a]); ausgenommen Abschläge
  und Verbraucher ([§ 271a Abs. 5 BGB][bgb271a]).

## 6. Hinweise in Sonderfällen

| Fall | Hinweis auf der Rechnung | Grundlage |
|---|---|---|
| Leistung an Unternehmen im EU-Ausland, Kunde schuldet die Steuer | „Steuerschuldnerschaft des Leistungsempfängers“, USt-IdNr. beider Seiten | [§ 14a Abs. 1 UStG][ustg14a] |
| Inländische Leistung nach § 13b Abs. 2 UStG an Unternehmer | „Steuerschuldnerschaft des Leistungsempfängers“, keine Steuer | [§ 14a Abs. 5 UStG][ustg14a]; Bauleistungen bildet Buchfink nicht ab |
| Steuerfreie Leistung | Hinweis auf die Befreiung | [§ 14 Abs. 4 Satz 1 Nr. 8 UStG][ustg14] |
| Rechnung an Verbraucher | Verzug 30 Tage nach Fälligkeit und Zugang; Buchfink druckt ihn bei `privatperson: true` | [mahnen-verzug.md, Abschnitt 1](mahnen-verzug.md#1-fälligkeit-und-verzug) |
| Anzahlungsrechnung | Kennzeichnung als Anzahlung | [anzahlungen.md, Abschnitt 3](anzahlungen.md#3-abschlagsrechnung) |

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[hgb238]: https://www.gesetze-im-internet.de/hgb/__238.html
[bgb271a]: https://www.gesetze-im-internet.de/bgb/__271a.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
[ustae131]: https://www.steuerschroeder.de/steuergesetze/ustae/13.1
