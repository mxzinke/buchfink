# Pflichtangaben einer Eingangsrechnung

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und UStDV am 17.10.2026 auf
gesetze-im-internet.de gelesen.

## Inhalt

1. Was eine Rechnung ist
2. Die zehn Pflichtangaben
3. Erleichterungen
4. Mehrere Dokumente, Abkürzungen, Leistungsmonat
5. Fehlende oder falsche Angaben
6. Falscher Steuerausweis

## 1. Was eine Rechnung ist

- Jedes Dokument, mit dem über eine Leistung abgerechnet wird
  ([§ 14 Abs. 1 Satz 1 UStG][ustg14]); E-Rechnung oder sonstige Rechnung:
  [e-rechnung-empfang.md](e-rechnung-empfang.md#1-begriffe).
- Eine Gutschrift im Sinne des UStG stellt der Leistungsempfänger nach
  Vereinbarung aus ([§ 14 Abs. 2 Satz 5 UStG][ustg14]); ein Dokument
  „Gutschrift“ über einen Preisnachlass ist eine Rechnungskorrektur. Frag,
  welcher Fall vorliegt.
- Das Kontrollverfahren mit Prüfpfad zwischen Rechnung und Leistung
  ([§ 14 Abs. 3 Satz 4, 5 UStG][ustg14]) ist in Buchfink der
  Leistungsnachweis ab der eingestellten Grenze. Den Vermerk gibt der Mensch.

## 2. Die zehn Pflichtangaben

Liste nach [§ 14 Abs. 4 Satz 1 UStG][ustg14]:
[rechnungen/pflichtangaben-ausgang.md, Abschnitt 2](../rechnungen/pflichtangaben-ausgang.md#2-pflichtangaben-und-woher-buchfink-sie-nimmt).
Beim Empfang prüfst du:

| Nr. | Prüfhinweis |
|---|---|
| 1 | Empfänger ist die GmbH. Nennt eine Rechnung über 250 € Gesellschafter oder Geschäftsführer persönlich, gibt sie der GmbH keine Vorsteuer; berichtigen lassen |
| 2 | Steuernummer oder USt-IdNr. des Lieferanten |
| 3, 4 | Ausstellungsdatum wird Belegdatum; Rechnungsnummer dient der Dublettenprüfung |
| 5 | „Beratung“ ohne Bezug ist dünn; frag nach Angebot oder Leistungsnachweis |
| 6 | Leistungszeitpunkt wird `leistung_von` / `leistung_bis` (Abschnitt 4) |
| 7 | Netto je Steuersatz in die Positionen; vereinbarte Minderungen wie Skonto stehen auf der Rechnung |
| 8 | bei § 13b statt Steuer der Hinweis „Steuerschuldnerschaft des Leistungsempfängers“ ([§ 14a Abs. 5 UStG][ustg14a]); fehlt er, gilt § 13b trotzdem |
| 9 | Grundstücksleistung an Nichtunternehmer; selten |
| 10 | „Gutschrift“, wenn der Leistungsempfänger abrechnet |

Für einen Lieferanten ohne Sitz und Betriebsstätte im Inland gelten bei § 13b
die Rechnungsvorschriften seines Staates ([§ 14 Abs. 7 UStG][ustg14]); die
GmbH muss trotzdem Empfängerin sein.

## 3. Erleichterungen

| Fall | Was genügt | Fundstelle |
|---|---|---|
| Gesamtbetrag bis 250 € | Name und Anschrift des Lieferanten, Ausstellungsdatum, Menge und Art der Leistung, Brutto in einer Summe und Steuersatz oder Hinweis auf Befreiung | [§ 33 Satz 1 UStDV][ustdv33] |
| Fahrausweis für Personenbeförderung | Beförderer mit Anschrift, Ausstellungsdatum, Brutto in einer Summe, Steuersatz (entfällt bei 7 % nach § 12 Abs. 2 Nr. 10 UStG) | [§ 34 Abs. 1 UStDV][ustdv34] |
| Kleinunternehmer | Name und Anschrift beider Seiten, Steuernummer, USt-IdNr. oder Kleinunternehmer-IdNr., Datum, Leistung, Entgelt mit Hinweis auf § 19 UStG | [§ 34a UStDV][ustdv34a] |

Kein Kleinbetrag bei Fernverkauf, innergemeinschaftlicher Lieferung und § 13b
([§ 33 Satz 3 UStDV][ustdv33]).

## 4. Mehrere Dokumente, Abkürzungen, Leistungsmonat

Nach [§ 31 UStDV][ustdv31]:

- Mehrere Dokumente: Eines enthält Entgelt und Steuer und bezeichnet die
  übrigen (Abs. 1); leg alle Teile als einen Beleg ab (`original`, `anhang`).
  Abkürzungen und Schlüssel sind zulässig, wenn ihre Bedeutung beiden Seiten
  vorliegt (Abs. 2, 3).
- Als Leistungszeitpunkt genügt der Kalendermonat (Abs. 4): `leistung_von` am
  Ersten, `leistung_bis` am Letzten des Monats. Fehlt der Leistungszeitpunkt,
  frag nach und lass berichtigen.

## 5. Fehlende oder falsche Angaben

- Der Lieferant berichtigt mit einem Dokument, das sich spezifisch und
  eindeutig auf die Rechnung bezieht, in der Form des § 14
  ([§ 31 Abs. 5 UStDV][ustdv31]).
- Ob die Vorsteuer dann auf den ursprünglichen Zeitraum zurückwirkt, ist eine
  Wertungsfrage; nenne die Optionen, bei großem Betrag einen Steuerberater.
  Ersatz einer sonstigen Rechnung durch eine E-Rechnung:
  [e-rechnung-empfang.md](e-rechnung-empfang.md#5-folgen-für-die-vorsteuer).
- In Buchfink bleibt der Beleg abgelegt, bis die Berichtigung kommt; sie wird
  `anhang` desselben Belegs, ist er schon gebucht, ein eigener Beleg.

## 6. Falscher Steuerausweis

- Zu hoch (etwa 19 % statt 7 %): Der Lieferant schuldet den Mehrbetrag
  ([§ 14c Abs. 1 UStG][ustg14c]); abziehbar ist nur die gesetzlich geschuldete
  Steuer ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]). Berichtigen lassen; bis
  dahin nicht buchen oder nach Entscheidung des Menschen mit der richtigen
  Steuer.
- Unberechtigt (Kleinunternehmer, steuerfreie Leistung): Der Lieferant
  schuldet den Betrag ([§ 14c Abs. 2 UStG][ustg14c]); für die GmbH keine
  Vorsteuer. Vorlegen.
- Zu niedrig: abziehbar höchstens der ausgewiesene Betrag; berichtigen lassen.
- Ausländische Umsatzsteuer:
  [steuerfaelle.md](steuerfaelle.md#7-ausländische-umsatzsteuer-auf-der-rechnung).

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg14c]: https://www.gesetze-im-internet.de/ustg_1980/__14c.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[ustdv34]: https://www.gesetze-im-internet.de/ustdv_1980/__34.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
