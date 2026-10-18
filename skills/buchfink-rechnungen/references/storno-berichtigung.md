# Ausgangsrechnung stornieren oder berichtigen

Rechtsstand: 17. Oktober 2026. Wie eine Buchung korrigiert wird (Storno mit
heutigem Datum, Neubuchung), steht in buchfink-grundlagen. Hier geht es um das
Rechnungsdokument und die Umsatzsteuer.

## Inhalt

1. Was Buchfink anbietet
2. Welcher Weg für welchen Fehler
3. Falsch ausgewiesene Umsatzsteuer
4. Wirkung auf die Voranmeldung
5. Ablauf

## 1. Was Buchfink anbietet

| Dokument | Wirkung in Buchfink |
|---|---|
| **Stornorechnung** | hebt die Rechnung vollständig auf, mit negativen Beträgen und Verweis; die Forderungsbuchung wird storniert |
| **Rechnungskorrektur** | ersetzt den Inhalt vollständig, mit Verweis auf die berichtigte Rechnung; Buchfink storniert die alte Buchung und bucht neu |

- Beide bekommen eine eigene Nummer aus dem Nummernkreis. Die ursprüngliche
  Nummer wird nie wiederverwendet.
- Im Datensatz der E-Rechnung tragen beide den Typcode 384 (Korrekturrechnung).
- Buchfink bildet beide noch nicht ab, weder als Werkzeug noch in der App. Du
  bereitest Grund, neue Positionen und Text vor, damit sie ausgestellt werden
  können, sobald es den Weg gibt.
- Eine „Gutschrift“ im Sinne von [§ 14 Abs. 2 Satz 5 UStG][ustg14] ist die
  Abrechnung durch den Kunden. Nenn eine Stornorechnung deshalb nie Gutschrift.

## 2. Welcher Weg für welchen Fehler

| Fehler | Weg | Grundlage |
|---|---|---|
| Pflichtangabe fehlt oder ist falsch (Anschrift, Leistungsdatum, Beschreibung, USt-IdNr.) | Rechnungskorrektur; ein Dokument, das spezifisch und eindeutig auf die Rechnung verweist, genügt | [§ 31 Abs. 5 UStDV][ustdv31] |
| Falscher Kunde | Stornorechnung, neue Rechnung an den richtigen Kunden | [§ 14 Abs. 4 Satz 1 Nr. 1 UStG][ustg14] |
| Leistungsumfang oder -inhalt ändert sich | Rechnungskorrektur | [UStAE 14.11 Abs. 4 Satz 3][bmf2025] |
| Skonto, Nachlass wegen Mangel ohne Änderung der Leistung, Rückgängigmachung | keine Rechnungsberichtigung; Minderung der Bemessungsgrundlage | [§ 17 Abs. 1, Abs. 2 Nr. 3 UStG][ustg17], [UStAE 14.11 Abs. 4 Satz 1, 2][bmf2025] |
| Leistung findet nicht statt | Stornorechnung | [§ 17 Abs. 2 Nr. 2 UStG][ustg17] |
| Falscher Steuersatz oder Steuerfall | Abschnitt 3 | [§ 14c UStG][ustg14c] |

- Eine E-Rechnung wird wieder als E-Rechnung berichtigt; Angaben per E-Mail
  oder PDF reichen nicht ([UStAE 14.11 Abs. 1 Satz 7, 8 i. d. F. vom 15.10.2025][bmf2025]).
- Nachlässe wegen Mängeln und Skonto am Zahlungseingang bucht buchfink-bank.

## 3. Falsch ausgewiesene Umsatzsteuer

- **Zu hoch ausgewiesen** (z. B. 19 % statt 7 %, oder Steuer auf eine Leistung
  an ein EU-Unternehmen): Das Unternehmen schuldet auch den Mehrbetrag
  ([§ 14c Abs. 1 Satz 1 UStG][ustg14c]). Berichtigt es den Steuerbetrag
  gegenüber dem Kunden, mindert sich die Schuld wie bei einer Änderung der
  Bemessungsgrundlage, also im Zeitraum der Berichtigung
  ([§ 14c Abs. 1 Satz 2 UStG][ustg14c], [§ 17 Abs. 1 UStG][ustg17]).
- **Unberechtigt ausgewiesen** (keine Leistung, Scheinrechnung, Doppel mit
  abweichendem Inhalt): Der ausgewiesene Betrag wird geschuldet
  ([§ 14c Abs. 2 Satz 1 UStG][ustg14c]). Berichtigt werden darf erst, wenn der
  Kunde keine Vorsteuer gezogen oder sie zurückgezahlt hat, und nur nach
  schriftlichem Antrag und Zustimmung des Finanzamts
  ([§ 14c Abs. 2 Satz 3 bis 5 UStG][ustg14c]). Das ist ein Fall für den
  Steuerberater.
- **Zu niedrig ausgewiesen:** Die gesetzliche Steuer ist aus dem Betrag
  herauszurechnen, den der Kunde zahlt ([§ 10 Abs. 1 Satz 2 UStG][ustg10]).
  Der Kunde zieht nur die ausgewiesene Steuer ab. Ob nachgefordert wird,
  entscheidet der Mensch; berichtigt wird mit Rechnungskorrektur.
- Buchfink lässt einen Steuerausweis nur beim steuerpflichtigen Inlandsumsatz
  zu. Fälle nach § 14c entstehen deshalb meist durch einen falsch gewählten
  Steuerfall oder Satz.

## 4. Wirkung auf die Voranmeldung

- Der Storno trägt das heutige Datum und wirkt im laufenden
  Voranmeldungszeitraum (buchfink-grundlagen).
- Minderungen nach § 17 sind im Zeitraum der Änderung zu berücksichtigen
  ([§ 17 Abs. 1 Satz 8 UStG][ustg17]). Das passt zum Storno.
- War die ursprüngliche Voranmeldung schon falsch (z. B. falscher Satz bei
  richtiger Leistung, Leistung im falschen Monat erfasst), frag den Menschen, ob
  sie berichtigt werden muss ([§ 153 AO][ao153]); den Ablauf regelt
  buchfink-monatsabschluss.

## 5. Ablauf

1. Rechnung finden: `buchungen_suchen` (Text, Kontakt, Betrag) oder
   `offene_posten`. Fertig, wenn Nummer, Datum, Buchung und Zahlungsstand
   feststehen.
2. Fehler einordnen nach Abschnitt 2 und 3. Fertig, wenn der Weg feststeht.
3. Vorlage an den Menschen: Fehler, Weg, Fundstelle, neue Positionen, Wirkung
   auf Voranmeldung und offenen Posten.
4. Sag dem Menschen, dass Buchfink Stornorechnung und Rechnungskorrektur noch
   nicht ausstellt; die Rechnung bleibt bis dahin unverändert stehen. Fertig,
   wenn er Fehler, Weg und Folgen kennt.

[ustg10]: https://www.gesetze-im-internet.de/ustg_1980/__10.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14c]: https://www.gesetze-im-internet.de/ustg_1980/__14c.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ao153]: https://www.gesetze-im-internet.de/ao_1977/__153.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
