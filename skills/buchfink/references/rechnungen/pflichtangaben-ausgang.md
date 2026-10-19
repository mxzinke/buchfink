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

- Was eine Rechnung ist:
  [belege/pflichtangaben.md, Abschnitt 1](../belege/pflichtangaben.md#1-was-eine-rechnung-ist).
- Pflicht binnen sechs Monaten nach der Leistung bei Leistungen an ein anderes
  Unternehmen für dessen Unternehmen und an juristische Personen, außer bei
  steuerfreien Umsätzen nach § 4 Nr. 8 bis 29 UStG
  ([§ 14 Abs. 2 Satz 2 Nr. 1, 2 UStG][ustg14]).
- An Verbraucher besteht keine Pflicht nach dem UStG (außer bei Leistungen im
  Zusammenhang mit einem Grundstück, [§ 14 Abs. 2 Satz 2 Nr. 3 UStG][ustg14]).
  Eine Rechnung ist trotzdem der Beleg für die Erlösbuchung
  ([§ 238 Abs. 1 HGB][hgb238]); stelle sie deshalb immer aus.
- Bei Anzahlungen gelten dieselben Regeln sinngemäß
  ([§ 14 Abs. 5 Satz 1 UStG][ustg14]); Frist sechs Monate ab Vereinnahmung
  ([UStAE 14.1 Abs. 3 Satz 4 i. d. F. vom 15.10.2025][bmf2025]).

## 2. Pflichtangaben und woher Buchfink sie nimmt

Pflichtangaben nach [§ 14 Abs. 4 Satz 1 UStG][ustg14]:

| Nr. | Angabe | Quelle in Buchfink | Dein Teil |
|---|---|---|---|
| 1 | Vollständiger Name und Anschrift von Unternehmen und Kunde | Unternehmensstammdaten, Kontakt | Kontakt vollständig anlegen; Stammdaten pflegt der Mensch in der App |
| 2 | Steuernummer oder USt-IdNr. des Unternehmens | Unternehmensstammdaten | fehlt sie, meldet `rechnung_vorschau` es; der Mensch ergänzt in der App |
| 3 | Ausstellungsdatum | `rechnungsdatum` | Tag der Ausstellung, nicht rückdatieren |
| 4 | Fortlaufende, einmalige Rechnungsnummer | Nummernkreis | Buchfink vergibt sie beim Ausstellen |
| 5 | Menge und Art der Leistung, Umfang und Art der sonstigen Leistung | `positionen[].text`, `menge`, `einheit` | so genau, dass die Leistung eindeutig feststeht |
| 6 | Zeitpunkt der Leistung; bei Anzahlung Tag der Vereinnahmung, wenn bekannt und abweichend vom Rechnungsdatum | `leistung_von`, `leistung_bis` | Abschnitt 4 |
| 7 | Entgelt nach Steuersätzen und Befreiungen aufgeschlüsselt, jede im Voraus vereinbarte Minderung | Positionen, Zahlungsbedingungen | Abschnitt 5 |
| 8 | Steuersatz und Steuerbetrag oder Hinweis auf die Befreiung | `satz`, `steuerfall` | Steuerfall nach [ablauf.md, Abschnitt 3](ablauf.md#3-steuerfall) |
| 9 | Hinweis auf die Aufbewahrungspflicht des Empfängers (§ 14b Abs. 1 Satz 5 UStG) | – | nur bei Leistungen im Zusammenhang mit einem Grundstück an Nichtunternehmer; frag den Menschen |
| 10 | Angabe „Gutschrift“ | – | Buchfink stellt keine Gutschriften aus |

- Mehrere Dokumente, Abkürzungen und der Kalendermonat als Leistungszeitpunkt
  ([§ 31 Abs. 1 bis 4 UStDV][ustdv31]):
  [belege/pflichtangaben.md, Abschnitt 4](../belege/pflichtangaben.md#4-mehrere-dokumente-abkürzungen-leistungsmonat).
  Bei der E-Rechnung gehören alle Pflichtangaben in den strukturierten Teil
  ([e-rechnung-ausgang.md, Abschnitt 5](e-rechnung-ausgang.md#5-inhalt-des-strukturierten-teils)).
- Fehlt eine Pflichtangabe, kann der Kunde die Vorsteuer nicht abziehen
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]). Rechne mit Rückfragen und
  berichtige nach [storno-berichtigung.md](storno-berichtigung.md).

## 3. Kleinbetragsrechnung

- Was eine Rechnung bis 250 € Gesamtbetrag braucht und wann die Erleichterung
  nicht gilt ([§ 33 UStDV][ustdv33]):
  [belege/pflichtangaben.md, Abschnitt 3](../belege/pflichtangaben.md#3-erleichterungen).
  Die Grenze steht in der UStDV und kann sich durch Verordnung ändern.
- `rechnung_ausstellen` erzeugt immer die vollständige Rechnung. Eine
  Kleinbetragsrechnung brauchst du deshalb nicht; sie ist nur für die
  E-Rechnungspflicht von Bedeutung
  ([e-rechnung-ausgang.md, Abschnitt 2](e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist)).

## 4. Leistungszeitpunkt

- Pflichtangabe auch dann, wenn er mit dem Rechnungsdatum übereinstimmt
  ([§ 14 Abs. 4 Satz 1 Nr. 6 UStG][ustg14]).
- Sonstige Leistungen sind mit ihrer Vollendung ausgeführt; zeitlich begrenzte
  Dauerleistungen mit Ende des Rechtsverhältnisses, wenn keine Teilleistungen
  vereinbart sind ([UStAE 13.1 Abs. 3][ustae131]).
- Der Leistungszeitpunkt bestimmt, in welchem Voranmeldungszeitraum die
  Steuer entsteht
  ([monatsabschluss/ustva.md, Abschnitt 1](../monatsabschluss/ustva.md#1-pflicht-und-form));
  für Anzahlungen gilt [anzahlungen.md, Abschnitt 1](anzahlungen.md#1-wann-die-steuer-entsteht).
- Wertungsfrage bei Projekten: Abnahme, Übergabe, Go-live oder Ende des
  Leistungszeitraums. Das Datum entscheidet über den Voranmeldungszeitraum der
  Steuer. Frag den Menschen, was vereinbart und was geschehen ist.

## 5. Entgelt, Skonto, Zahlungsziel

- Ein im Voraus vereinbarter Skonto ist Pflichtangabe
  ([§ 14 Abs. 4 Satz 1 Nr. 7 UStG][ustg14]). Ein Hinweis wie „2 % Skonto bei
  Zahlung binnen 10 Tagen“ genügt. `rechnung_ausstellen` nimmt ihn als
  `skonto_promille` (20 für 2 %) und `skonto_tage`, wenn der Mensch Skonto
  gewährt.
- Ein Skontoabzug ändert die Bemessungsgrundlage erst, wenn der Kunde ihn
  zieht ([§ 17 Abs. 1 UStG][ustg17]); berichtigt wird die Rechnung dann nicht
  ([storno-berichtigung.md, Abschnitt 2](storno-berichtigung.md#2-welcher-weg-für-welchen-fehler)).
  Den Abzug bucht [bank/skonto-differenzen.md](../bank/skonto-differenzen.md)
  am Zahlungseingang.
- `zahlungsziel_tage`: Fälligkeit ab Rechnungsdatum. Eine vereinbarte
  Zahlungsfrist von mehr als 60 Tagen nach Empfang der Leistung ist gegenüber
  einem Unternehmen nur wirksam, wenn sie ausdrücklich getroffen und nicht grob
  unbillig ist ([§ 271a Abs. 1 BGB][bgb271a]). Das gilt nicht für Abschläge
  ([§ 271a Abs. 5 Nr. 1 BGB][bgb271a]) und nicht gegenüber Verbrauchern
  ([§ 271a Abs. 5 BGB][bgb271a]).

## 6. Hinweise in Sonderfällen

| Fall | Hinweis auf der Rechnung | Grundlage |
|---|---|---|
| Leistung an Unternehmen im EU-Ausland, Kunde schuldet die Steuer | „Steuerschuldnerschaft des Leistungsempfängers“, USt-IdNr. beider Seiten | [§ 14a Abs. 1 UStG][ustg14a] |
| Inländische Leistung nach § 13b Abs. 2 UStG an Unternehmer | „Steuerschuldnerschaft des Leistungsempfängers“, keine Steuer | [§ 14a Abs. 5 UStG][ustg14a]; Bauleistungen bildet Buchfink nicht ab |
| Steuerfreie Leistung | Hinweis auf die Steuerbefreiung | [§ 14 Abs. 4 Satz 1 Nr. 8 UStG][ustg14] |
| Rechnung an Verbraucher | Hinweis auf den Verzug 30 Tage nach Fälligkeit und Zugang; Buchfink druckt ihn bei `privatperson: true` | [mahnen-verzug.md, Abschnitt 1](mahnen-verzug.md#1-fälligkeit-und-verzug) |
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
