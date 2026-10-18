# Fremdwährung, Auslagen, Kreditkartenbelege

Rechtsstand: 17. Oktober 2026. Wortlaut von UStG und HGB am 17.10.2026 gelesen.
Werkzeugfelder nach Buchfink 0.3.0.

## Inhalt

1. Beleg in Fremdwährung
2. Auslage eines Gesellschafters
3. Auslage eines Arbeitnehmers
4. Kreditkartenbeleg

## 1. Beleg in Fremdwährung

**Was der Beleg zeigen muss.** Währung und, soweit umgerechnet, den Kurs
([GoBD Rz. 77][gobd]).

**Wie Buchfink umrechnet.**

- Aufwand und Verbindlichkeit: EZB-Referenzkurs des Belegdatums. Ohne Kurs
  bucht Buchfink nicht (Fehler `fachliche_regel`, bei fehlendem Netz mit
  `wiederholbar: true`); einen Kurs rät Buchfink nie.
- Umsatzsteuer und Vorsteuer: Monatsdurchschnittskurs, den das BMF für den
  Monat der Leistung bekannt gibt ([§ 16 Abs. 6 Satz 1 UStG][ustg16]). Das
  Finanzamt kann den Tageskurs gestatten (Satz 3). Buchfink nimmt den Kurs des
  Monats des Belegdatums; liegt die Leistung in einem anderen Monat, sag es dem
  Menschen. Fehlt der amtliche Monatskurs, nimmt Buchfink den Tageskurs und
  meldet das.
- Die Differenz zwischen beiden Kursen bucht Buchfink als Kursaufwand (6880)
  oder Kursertrag (4840).

**Was du übergibst.** In `beleg_buchen_mit_rueckfrage`:

- `waehrung` (ISO-Code, z. B. `USD`),
- `positionen[].netto` in der Fremdwährung, so wie auf dem Beleg,
- `fremdbetrag` als Endsumme des Belegs; Buchfink prüft die Summe der
  Positionen dagegen.

Zeige dem Menschen vor dem Ja den Kurs und den Eurobetrag aus der Vorschau.

**Später.** Die Kursdifferenz bei Zahlung bucht buchfink-bank. Die Bewertung
offener Fremdwährungsposten zum Stichtag
([§ 256a HGB][hgb256a]) macht buchfink-jahresabschluss.

**Bankkonto in Fremdwährung.** Der Kontoauszugsimport lehnt Umsätze in
Fremdwährung ab; siehe buchfink-bank.

## 2. Auslage eines Gesellschafters

Der Gesellschafter hat eine Ausgabe der GmbH privat bezahlt und bekommt sie
erstattet.

1. Prüfe, ob die Ausgabe betrieblich veranlasst ist. Zweifel: Wertungsfrage,
   vorlegen; die Folgen (verdeckte Gewinnausschüttung) regelt
   buchfink-gesellschafter.
2. Prüfe den Leistungsempfänger: Über 250 € brutto muss die Rechnung die GmbH
   nennen ([§ 14 Abs. 4 Satz 1 Nr. 1 UStG][ustg14]); sonst trägt sie keine
   Vorsteuer der GmbH. Lass sie auf die GmbH berichtigen
   ([§ 31 Abs. 5 UStDV][ustdv31]). Bis 250 € ist kein Empfänger nötig
   ([§ 33 UStDV][ustdv33]).
3. Den Kontakt des Gesellschafters findest du mit `kontakte_suchen`
   (`text`: sein Name). Fehlt er, legst du ihn mit `kontakt_anlegen` an
   (`typ: lieferant`, Name genau wie in der Gesellschafterliste der
   Stammdaten): Buchfink erkennt Gesellschafter und Geschäftsführer am Namen
   und lehnt andere Gläubiger einer Auslage ab.
4. Buche mit `beleg_buchen_mit_rueckfrage`, `bezahlung: auslage` und
   `gesellschafter_kontakt_id`. Buchfink bucht die Erstattungsschuld gegenüber
   dem Gesellschafter (3640), bei einem Geschäftsführer ohne Anteil auf 3500.
   Eine Auslage ist nie Routine.
5. Die Erstattung (Überweisung an den Gesellschafter) ordnet buchfink-bank zu.
   Verrechnung, Darlehen und Ausweis regelt buchfink-gesellschafter.

## 3. Auslage eines Arbeitnehmers

Reisekostenabrechnung und Erstattung an Arbeitnehmer laufen außerhalb von
Buchfink (buchfink-grundlagen, Grenzen). Liegt eine einzelne Rechnung vor, die
ein Arbeitnehmer bezahlt hat, leg den Fall vor; `bezahlung: auslage` gilt
nur für Gesellschafter.

## 4. Kreditkartenbeleg

**Was ein Beleg ist.** Der Kartenbeleg (Slip) und die Zeile der Monatsabrechnung
weisen nur die Zahlung nach. Für die Vorsteuer braucht die GmbH die Rechnung des
Händlers ([§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG][ustg15]). Bis 250 € genügt
eine Kleinbetragsrechnung, etwa ein Kassenbon mit den Angaben nach
[§ 33 UStDV][ustdv33].

**Buchen.** Lege die Rechnung ab und buche sie mit `bezahlung: kreditkarte`. Buchfink bucht gegen das Konto 3610 Kreditkartenabrechnung statt
gegen den Lieferanten. Kreditkarte ist mit `beleg_buchen` als Routine möglich,
wenn alle Bedingungen für „direkt“ erfüllt sind.

**Ohne Rechnung.** Frag den Menschen nach der Rechnung (Kundenkonto des
Anbieters, E-Mail). Gibt es keine, erfasst er einen Eigenbeleg mit Grund
(buchfink-grundlagen); Vorsteuer entfällt.

**Fremdwährung auf der Karte.** Rechnung in Fremdwährung wie Abschnitt 1. Der
Betrag, den die Kartenabrechnung in Euro belastet, weicht vom Kurs ab; die
Differenz klärt buchfink-bank beim Ausgleich der Abrechnung.

**Monatsabrechnung.** Den Ausgleich des Kontos 3610 mit der Lastschrift und
Gebühren der Karte regelt buchfink-bank.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[hgb256a]: https://www.gesetze-im-internet.de/hgb/__256a.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
