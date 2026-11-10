# Ausgangsrechnung stornieren oder berichtigen

Rechtsstand: 17. Oktober 2026.

Rechnungsdokument und Umsatzsteuer. Die Buchung korrigiert
[grundlagen/arbeitsweise.md, Abschnitt 11](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren).

## Inhalt

1. Was Buchfink anbietet
2. Welcher Weg für welchen Fehler
3. Falsch ausgewiesene Umsatzsteuer
4. Wirkung auf die Voranmeldung
5. Ablauf

## 1. Was Buchfink anbietet

Buchfink stellt Stornorechnungen und Berichtigungen von Euro-Rechnungen
in der App und über `rechnung_ausstellen` aus. `rechnung_vorschau` zeigt die
Buchungswirkung mit `art` und `ursprung_id`. Eine Berichtigung braucht zusätzlich
den vollständigen richtigen Rechnungsinhalt. Reine Textänderungen buchen nichts;
Betragsänderungen buchen nur die Differenz.

| Dokument | Inhalt |
|---|---|
| **Stornorechnung** | hebt die Rechnung vollständig auf, mit Verweis auf sie; im lesbaren Teil negative Beträge, im XML Typ 381 mit positiven Beträgen |
| **Rechnungskorrektur** | ersetzt den Inhalt vollständig, mit Verweis auf die berichtigte Rechnung |

- Beide bekommen eine eigene Nummer; die ursprüngliche wird nie
  wiederverwendet.
- Nenn eine Stornorechnung nie Gutschrift; Gutschrift ist die Abrechnung durch
  den Kunden ([§ 14 Abs. 2 Satz 5 UStG][ustg14]).

## 2. Welcher Weg für welchen Fehler

| Fehler | Weg | Grundlage |
|---|---|---|
| Pflichtangabe fehlt oder ist falsch (Anschrift, Leistungsdatum, Beschreibung, USt-IdNr.) | Rechnungskorrektur; ein Dokument mit spezifischem, eindeutigem Verweis auf die Rechnung genügt | [§ 31 Abs. 5 UStDV][ustdv31] |
| Falscher Kunde | Stornorechnung, neue Rechnung an den richtigen Kunden | [§ 14 Abs. 4 Satz 1 Nr. 1 UStG][ustg14] |
| Leistungsumfang oder -inhalt ändert sich | Rechnungskorrektur | [UStAE 14.11 Abs. 4 Satz 3][bmf2025] |
| Skonto, Nachlass wegen Mangel ohne Änderung der Leistung, Rückgängigmachung | keine Rechnungsberichtigung; Minderung der Bemessungsgrundlage ([bank/skonto-differenzen.md](../bank/skonto-differenzen.md)) | [§ 17 Abs. 1, Abs. 2 Nr. 3 UStG][ustg17], [UStAE 14.11 Abs. 4 Satz 1, 2][bmf2025] |
| Leistung findet nicht statt | Stornorechnung | [§ 17 Abs. 2 Nr. 2 UStG][ustg17] |
| Falscher Steuersatz oder Steuerfall | Abschnitt 3 | [§ 14c UStG][ustg14c] |

E-Rechnung: [e-rechnung-ausgang.md, Abschnitt 8](e-rechnung-ausgang.md#8-berichtigung-und-aufbewahrung).

## 3. Falsch ausgewiesene Umsatzsteuer

Meist durch falschen Steuerfall oder Satz.

- **Zu hoch** (19 % statt 7 %, Steuer an ein EU-Unternehmen): Die GmbH schuldet
  den Mehrbetrag ([§ 14c Abs. 1 Satz 1 UStG][ustg14c]). Berichtigt sie den
  Steuerbetrag gegenüber dem Kunden, mindert sich die Schuld im Zeitraum der
  Berichtigung ([§ 14c Abs. 1 Satz 2 UStG][ustg14c], [§ 17 Abs. 1 UStG][ustg17]).
- **Unberechtigt** (keine Leistung, Doppel mit abweichendem Inhalt): Der
  ausgewiesene Betrag wird geschuldet ([§ 14c Abs. 2 Satz 1 UStG][ustg14c]);
  Berichtigung erst nach Rückgängigmachung der Vorsteuer beim Kunden, auf
  Antrag mit Zustimmung des Finanzamts ([§ 14c Abs. 2 Satz 3 bis 5 UStG][ustg14c]).
  Fall für den Steuerberater.
- **Zu niedrig:** Die gesetzliche Steuer steckt im gezahlten Betrag
  ([§ 10 Abs. 1 Satz 2 UStG][ustg10]); der Kunde zieht nur die ausgewiesene ab.
  Ob nachgefordert wird, entscheidet der Mensch; berichtigt wird mit
  Rechnungskorrektur.

## 4. Wirkung auf die Voranmeldung

- Der Storno einer Ausgangsrechnung wirkt im laufenden Voranmeldungszeitraum
  ([grundlagen/arbeitsweise.md, Abschnitt 11](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)),
  Minderungen nach § 17 UStG im Zeitraum der Änderung
  ([monatsabschluss/berichtigung.md, Abschnitt 1](../monatsabschluss/berichtigung.md#1-berichtigung-oder-laufender-zeitraum)).
- War die ursprüngliche Voranmeldung schon falsch (falscher Satz, falscher
  Monat), frag, ob sie zu berichtigen ist ([§ 153 AO][ao153]);
  Ablauf: [monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md).

## 5. Ablauf

1. Rechnung finden: `buchungen_suchen` oder `offene_posten`. Fertig, wenn
   Nummer, Datum, Buchung und Zahlungsstand feststehen.
2. Fehler nach Abschnitt 2 und 3 einordnen. Fertig, wenn der Weg feststeht.
3. Vorlage an den Menschen: Fehler, Weg, Fundstelle, Vorschau, Wirkung auf
   Voranmeldung und offenen Posten. Erst nach seinem Ja `rechnung_ausstellen`
   mit `art: storno` oder `berichtigung`, `ursprung_id` und `grund` aufrufen.
4. Dateien aus dem Exportordner versenden. Ursprung und Folgedokument bleiben
   sichtbar miteinander verbunden.

Eine aktive Forderungsausbuchung zuerst stornieren. Ein bezahlter Storno lässt
ein Kundenguthaben zurück, das verrechnet oder erstattet werden kann.
Vereinnahmte Abschläge erst nach Rückzahlung stornieren; unbezahlte Abschläge
ersetzt die Berichtigung ohne Buchung im selben Verbund.
Eine Schlussrechnung behält ihre abgesetzten Anzahlungen.

[ustg10]: https://www.gesetze-im-internet.de/ustg_1980/__10.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14c]: https://www.gesetze-im-internet.de/ustg_1980/__14c.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ao153]: https://www.gesetze-im-internet.de/ao_1977/__153.html
[bmf2025]: https://www.bundesfinanzministerium.de/Content/DE/Downloads/BMF_Schreiben/Steuerarten/Umsatzsteuer/Umsatzsteuer-Anwendungserlass/2025-10-15-einfuehrung-obligatorische-e-rechnung.html
