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
7. Befunde in Buchfink und was du tust

## 1. Was eine Rechnung ist

- Rechnung ist jedes Dokument, mit dem über eine Leistung abgerechnet wird,
  gleich wie es heißt ([§ 14 Abs. 1 Satz 1 UStG][ustg14]).
- Sie ist E-Rechnung (strukturiertes Format nach EN 16931 oder vereinbart und
  interoperabel) oder sonstige Rechnung (Papier oder anderes elektronisches
  Format) ([§ 14 Abs. 1 Satz 2 bis 6 UStG][ustg14]).
- Gutschrift heißt im Umsatzsteuerrecht: Der Leistungsempfänger rechnet ab, nach
  vorheriger Vereinbarung ([§ 14 Abs. 2 Satz 5 UStG][ustg14]). Ein Dokument
  „Gutschrift“ über einen Preisnachlass ist eine Rechnungskorrektur. Frag den
  Menschen, welcher Fall vorliegt.
- Echtheit, Unversehrtheit und Lesbarkeit sichert ein innerbetriebliches
  Kontrollverfahren mit Prüfpfad zwischen Rechnung und Leistung
  ([§ 14 Abs. 3 Satz 4 und 5 UStG][ustg14]). In Buchfink ist das der
  Leistungsnachweis ab der eingestellten Grenze (Vorgabe 1.000 € brutto).
  Den Vermerk gibt der Mensch; du trägst ihn nur ein (`leistungsnachweis`,
  `leistungsnachweis_am`).

## 2. Die zehn Pflichtangaben

[§ 14 Abs. 4 Satz 1 UStG][ustg14]:

| Nr. | Angabe | Prüfhinweis |
|---|---|---|
| 1 | vollständiger Name und Anschrift von Lieferant und Leistungsempfänger | Empfänger ist die GmbH mit ihrer Firma, nicht der Geschäftsführer persönlich |
| 2 | Steuernummer oder USt-IdNr. des Lieferanten | eine von beiden genügt |
| 3 | Ausstellungsdatum | wird zum Belegdatum |
| 4 | fortlaufende, einmalige Rechnungsnummer | Grundlage der Dublettenprüfung |
| 5 | Menge und handelsübliche Bezeichnung der Gegenstände oder Umfang und Art der Leistung | „Beratung“ ohne Bezug ist dünn; frag nach Angebot oder Leistungsnachweis |
| 6 | Zeitpunkt der Leistung; bei Anzahlung Zeitpunkt der Vereinnahmung, wenn bekannt und abweichend | wird `leistung_von` / `leistung_bis` |
| 7 | Entgelt nach Steuersätzen und Befreiungen aufgeschlüsselt, vorab vereinbarte Minderungen (Skonto, Bonus) | Netto je Satz in die Positionen |
| 8 | Steuersatz und Steuerbetrag oder Hinweis auf die Steuerbefreiung | bei § 13b statt dessen Hinweis nach Nr. 7 unten |
| 9 | Hinweis auf die Aufbewahrungspflicht des Leistungsempfängers in den Fällen des § 14b Abs. 1 Satz 5 | betrifft Grundstücksleistungen an Nichtunternehmer, für die GmbH selten |
| 10 | Angabe „Gutschrift“, wenn der Leistungsempfänger abrechnet | – |

Nennt eine Rechnung über 250 € den Gesellschafter oder Geschäftsführer
persönlich statt der GmbH, trägt sie keine Vorsteuer der GmbH. Lass sie auf
die GmbH berichtigen (Abschnitt 5).

Zusätzlich in besonderen Fällen ([§ 14a UStG][ustg14a]):

- Leistung, für die die GmbH die Steuer nach § 13b schuldet: Angabe
  „Steuerschuldnerschaft des Leistungsempfängers“, kein Steuerausweis
  ([§ 14a Abs. 5 UStG][ustg14a]). Fehlt der Hinweis, schuldet die GmbH die Steuer
  trotzdem; § 13b knüpft an die Leistung, nicht an die Rechnung.
- Lieferant ohne Sitz und Betriebsstätte im Inland, GmbH schuldet nach § 13b:
  Für die Rechnung gelten die Vorschriften des Staates des Lieferanten
  ([§ 14 Abs. 7 UStG][ustg14]). Prüfe trotzdem, dass die GmbH als Empfängerin
  genannt ist. Buchfink verlangt am Kontakt eines EU-Lieferanten die USt-IdNr.

## 3. Erleichterungen

| Fall | Was genügt | Fundstelle |
|---|---|---|
| Gesamtbetrag bis 250 € | Name und Anschrift des Lieferanten, Ausstellungsdatum, Menge und Art der Leistung, Brutto in einer Summe und Steuersatz oder Hinweis auf Befreiung. Kein Empfänger, keine Nummer, keine Steuernummer | [§ 33 Satz 1 UStDV][ustdv33] |
| Fahrausweis für Personenbeförderung | Beförderer mit Anschrift, Ausstellungsdatum, Brutto in einer Summe, Steuersatz (entfällt bei 7 % nach § 12 Abs. 2 Nr. 10 UStG) | [§ 34 Abs. 1 UStDV][ustdv34] |
| Kleinunternehmer | Name und Anschrift beider Seiten, Steuernummer, USt-IdNr. oder Kleinunternehmer-IdNr., Datum, Leistung, Entgelt mit Hinweis auf § 19 UStG | [§ 34a UStDV][ustdv34a] |

- Die Kleinbetragsregel gilt nicht für Leistungen nach §§ 3c, 6a und 13b UStG
  (Fernverkäufe, innergemeinschaftliche Lieferungen, Steuerschuld des
  Leistungsempfängers; [§ 33 Satz 3 UStDV][ustdv33]).
- Die Grenze von 250 € gilt seit 2017; davor 150 €. Buchfink führt sie
  datiert. Stand: Oktober 2026.
- Fahrausweise im grenzüberschreitenden Verkehr sind nur mit Bescheinigung über
  den Inlandsanteil eine Rechnung ([§ 34 Abs. 2 UStDV][ustdv34]).

## 4. Mehrere Dokumente, Abkürzungen, Leistungsmonat

- Eine Rechnung darf aus mehreren Dokumenten bestehen. In einem Dokument stehen
  Entgelt und Steuer; es bezeichnet die übrigen
  ([§ 31 Abs. 1 UStDV][ustdv31]). Lege alle Teile als einen Beleg ab
  (`original` und `anhang`).
- Name und Anschrift genügen, wenn sie sich eindeutig feststellen lassen
  ([§ 31 Abs. 2 UStDV][ustdv31]).
- Abkürzungen und Schlüssel für Nr. 1 und 5 sind zulässig, wenn ihre Bedeutung
  festgelegt ist und beiden Seiten vorliegt ([§ 31 Abs. 3 UStDV][ustdv31]).
- Als Leistungszeitpunkt genügt der Kalendermonat
  ([§ 31 Abs. 4 UStDV][ustdv31]). Setze dann `leistung_von` auf den Ersten und
  `leistung_bis` auf den Letzten des Monats.
- Steht kein Leistungsdatum auf der Rechnung, frag nach und fordere eine
  Berichtigung an; der Leistungszeitpunkt ist Pflichtangabe (Nr. 6).

## 5. Fehlende oder falsche Angaben

- Eine Rechnung kann berichtigt werden, wenn Angaben fehlen oder falsch sind.
  Der Lieferant übermittelt nur die fehlenden oder richtigen Angaben in einem
  Dokument, das sich spezifisch und eindeutig auf die Rechnung bezieht, in der
  Form des § 14 ([§ 31 Abs. 5 UStDV][ustdv31]).
- Die Berichtigung ist kein rückwirkendes Ereignis im Sinne von § 175 Abs. 1
  Satz 1 Nr. 2 und § 233a Abs. 2a AO ([§ 14 Abs. 4 Satz 4 UStG][ustg14]).
- Ob die Vorsteuer aus der berichtigten Rechnung auf den ursprünglichen
  Zeitraum zurückwirkt, ist eine Wertungsfrage mit Rechtsprechung dazu. Für den
  Ersatz einer sonstigen Rechnung durch eine E-Rechnung gilt
  [e-rechnung-empfang.md](e-rechnung-empfang.md#5-folgen-für-die-vorsteuer). In
  anderen Fällen nennst du dem Menschen die Optionen und empfiehlst bei großem
  Betrag einen Steuerberater.
- Ablauf in Buchfink: Beleg abgelegt lassen, Befund dem Menschen zeigen,
  Berichtigung anfordern lassen. Die Berichtigung legst du als `anhang` zum
  selben Beleg ab, solange er nicht gebucht ist; sonst als eigenen Beleg.

## 6. Falscher Steuerausweis

- Zu hoher Ausweis (z. B. 19 % statt 7 %): Der Lieferant schuldet den
  Mehrbetrag ([§ 14c Abs. 1 UStG][ustg14c]). Abziehbar ist nur die gesetzlich
  geschuldete Steuer ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]). Vorschlag:
  Rechnung berichtigen lassen; bis dahin nicht buchen oder nur mit der
  richtigen Steuer nach Entscheidung des Menschen.
- Steuer ausgewiesen, obwohl der Lieferant nicht berechtigt ist (z. B. als
  Kleinunternehmer oder für eine steuerfreie Leistung): Er schuldet den
  Betrag ([§ 14c Abs. 2 UStG][ustg14c]). Für die GmbH ist er keine
  abziehbare Vorsteuer. Wertungsfrage, vorlegen.
- Zu niedriger Ausweis: Abziehbar ist höchstens der ausgewiesene Betrag; lass
  berichtigen.
- Ausländische Umsatzsteuer ist keine abziehbare Vorsteuer; Vorgehen in
  [steuerfaelle.md](steuerfaelle.md#7-ausländische-umsatzsteuer-auf-der-rechnung).

## 7. Befunde in Buchfink

Befunde der E-Rechnungsprüfung, ihre Klassen und dein Vorgehen stehen in
[e-rechnung-empfang.md](e-rechnung-empfang.md#6-fehlerarten-und-befunde).

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14a]: https://www.gesetze-im-internet.de/ustg_1980/__14a.html
[ustg14c]: https://www.gesetze-im-internet.de/ustg_1980/__14c.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[ustdv34]: https://www.gesetze-im-internet.de/ustdv_1980/__34.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
