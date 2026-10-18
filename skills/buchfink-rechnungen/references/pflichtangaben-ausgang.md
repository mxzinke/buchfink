# Pflichtangaben einer Ausgangsrechnung

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Wann eine Rechnung Pflicht ist
2. Pflichtangaben und woher Buchfink sie nimmt
3. Kleinbetragsrechnung
4. Leistungszeitpunkt
5. Entgelt, Skonto, Zahlungsziel
6. Hinweise in Sonderfällen

## 1. Wann eine Rechnung Pflicht ist

- Rechnung ist jedes Dokument, mit dem über eine Leistung abgerechnet wird,
  gleich wie es heißt ([§ 14 Abs. 1 Satz 1 UStG][ustg14]).
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
| 8 | Steuersatz und Steuerbetrag oder Hinweis auf die Befreiung | `satz`, `steuerfall` | Steuerfall nach SKILL.md Abschnitt 3 |
| 9 | Hinweis auf die Aufbewahrungspflicht des Empfängers (§ 14b Abs. 1 Satz 5 UStG) | – | nur bei Leistungen im Zusammenhang mit einem Grundstück an Nichtunternehmer; frag den Menschen |
| 10 | Angabe „Gutschrift“ | – | Buchfink stellt keine Gutschriften aus |

- Name und Anschrift genügen, wenn sich beide eindeutig feststellen lassen;
  Abkürzungen und Symbole sind zulässig, wenn ihre Bedeutung festgelegt ist
  ([§ 31 Abs. 2, 3 UStDV][ustdv31]).
- Eine Rechnung kann aus mehreren Dokumenten bestehen, wenn eines davon Entgelt
  und Steuer zusammengefasst nennt und auf die anderen verweist
  ([§ 31 Abs. 1 UStDV][ustdv31]). Für die E-Rechnung gilt das nur eingeschränkt:
  alle Pflichtangaben gehören in den strukturierten Teil
  ([BMF 15.10.2025, Rn. 35][bmf2025]).
- Fehlt eine Pflichtangabe, kann der Kunde die Vorsteuer nicht abziehen
  ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]). Rechne mit Rückfragen und
  berichtige nach SKILL.md, Abschnitt 6.

## 3. Kleinbetragsrechnung

- Rechnungen bis 250 € Gesamtbetrag brauchen nur Name und Anschrift des
  leistenden Unternehmens, Ausstellungsdatum, Menge und Art der Leistung,
  Entgelt und Steuer in einer Summe sowie den Steuersatz
  ([§ 33 Satz 1 UStDV][ustdv33]). Stand: Oktober 2026; die Grenze steht in der
  UStDV und kann sich durch Verordnung ändern.
- Nicht zulässig bei Leistungen nach §§ 3c, 6a und 13b UStG
  ([§ 33 Satz 3 UStDV][ustdv33]).
- `rechnung_ausstellen` erzeugt immer die vollständige Rechnung. Eine
  Kleinbetragsrechnung brauchst du deshalb nicht; sie ist nur für die
  E-Rechnungspflicht von Bedeutung (SKILL.md, Abschnitt 4).

## 4. Leistungszeitpunkt

- Pflichtangabe auch dann, wenn er mit dem Rechnungsdatum übereinstimmt
  ([§ 14 Abs. 4 Satz 1 Nr. 6 UStG][ustg14]).
- Der Kalendermonat der Leistung genügt ([§ 31 Abs. 4 UStDV][ustdv31]).
- Sonstige Leistungen sind mit ihrer Vollendung ausgeführt; zeitlich begrenzte
  Dauerleistungen mit Ende des Rechtsverhältnisses, wenn keine Teilleistungen
  vereinbart sind ([UStAE 13.1 Abs. 3][ustae131]).
- Wertungsfrage bei Projekten: Abnahme, Übergabe, Go-live oder Ende des
  Leistungszeitraums. Das Datum entscheidet über den Voranmeldungszeitraum der
  Steuer ([§ 13 Abs. 1 Nr. 1 Buchst. a Satz 1 UStG][ustg13]). Frag den
  Menschen, was vereinbart und was geschehen ist.

## 5. Entgelt, Skonto, Zahlungsziel

- Ein im Voraus vereinbarter Skonto ist Pflichtangabe
  ([§ 14 Abs. 4 Satz 1 Nr. 7 UStG][ustg14]). Ein Hinweis wie „2 % Skonto bei
  Zahlung binnen 10 Tagen“ genügt. `rechnung_ausstellen` nimmt ihn als
  `skonto_promille` (20 für 2 %) und `skonto_tage`, wenn der Mensch Skonto
  gewährt.
- Ein Skontoabzug ändert die Bemessungsgrundlage erst, wenn der Kunde ihn
  zieht ([§ 17 Abs. 1 UStG][ustg17]); die Rechnung wird dann nicht berichtigt
  ([UStAE 14.11 Abs. 4 i. d. F. vom 15.10.2025][bmf2025]). Den Abzug bucht
  buchfink-bank am Zahlungseingang.
- `zahlungsziel_tage`: Fälligkeit ab Rechnungsdatum. Eine vereinbarte
  Zahlungsfrist von mehr als 60 Tagen nach Empfang der Leistung ist gegenüber
  einem Unternehmen nur wirksam, wenn sie ausdrücklich getroffen und nicht grob
  unbillig ist ([§ 271a Abs. 1 BGB][bgb271a]). Das gilt nicht für Abschläge und
  nicht gegenüber Verbrauchern ([§ 271a Abs. 5 BGB][bgb271a]).

## 6. Hinweise in Sonderfällen

| Fall | Hinweis auf der Rechnung | Grundlage |
|---|---|---|
| Leistung an Unternehmen im EU-Ausland, Kunde schuldet die Steuer | „Steuerschuldnerschaft des Leistungsempfängers“, USt-IdNr. beider Seiten | [§ 14a Abs. 1 UStG][ustg14a] |
| Inländische Leistung nach § 13b Abs. 2 UStG an Unternehmer | „Steuerschuldnerschaft des Leistungsempfängers“, keine Steuer | [§ 14a Abs. 5 UStG][ustg14a]; Bauleistungen bildet Buchfink nicht ab |
| Steuerfreie Leistung | Hinweis auf die Steuerbefreiung | [§ 14 Abs. 4 Satz 1 Nr. 8 UStG][ustg14] |
| Rechnung an Verbraucher | Verzug 30 Tage nach Fälligkeit und Zugang ohne Mahnung nur mit besonderem Hinweis | [§ 286 Abs. 3 Satz 1 BGB][bgb286]; Buchfink druckt ihn bei `privatperson: true` |
| Anzahlungsrechnung | erkennbar als Anzahlung, voraussichtlicher Leistungszeitpunkt | [UStAE 14.8 Abs. 1, 4][ustae148] |

[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[hgb238]: https://www.gesetze-im-internet.de/hgb/__238.html
[bgb271a]: https://www.gesetze-im-internet.de/bgb/__271a.html
[bgb286]: https://www.gesetze-im-internet.de/bgb/__286.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
[ustae131]: https://www.steuerschroeder.de/steuergesetze/ustae/13.1
[ustae148]: https://www.steuerschroeder.de/steuergesetze/ustae/14.8
