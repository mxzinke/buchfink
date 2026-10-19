# Fremdwährung und Auslagen

Rechtsstand: 17. Oktober 2026. Wortlaut des UStG am 17.10.2026 gelesen.
Werkzeugstand: Buchfink 0.3.0.

Kartenbelege und die Kreditkartenabrechnung regelt
[bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md).

## Inhalt

1. Beleg in Fremdwährung
2. Auslage eines Gesellschafters
3. Auslage eines Arbeitnehmers

## 1. Beleg in Fremdwährung

**Was der Beleg zeigen muss.** Währung und, soweit umgerechnet, den Kurs
([GoBD Rz. 77][gobd]).

**Wie Buchfink umrechnet.**

- Aufwand und Verbindlichkeit: EZB-Referenzkurs des Belegdatums. Ohne Kurs
  bucht Buchfink nicht (Fehler `fachliche_regel`, bei fehlendem Netz mit
  `wiederholbar: true`); einen Kurs rät Buchfink nie.
- Umsatzsteuer und Vorsteuer: Durchschnittskurs, den das BMF für den Monat
  bekannt gibt, in dem die Leistung ausgeführt oder das Entgelt vor der
  Ausführung vereinnahmt wird ([§ 16 Abs. 6 Satz 1 UStG][ustg16]). Den Tageskurs
  nur, wenn das Finanzamt ihn gestattet (Satz 3). Buchfink nimmt den Monat des
  Leistungsendes (`leistung_bis`, sonst `leistung_von`). Wird der Beleg mit der
  Zahlung gebucht (`bezahlt`, `kreditkarte`, `auslage`, Anzahlung) und liegt
  der Buchungstag davor, nimmt es den Monat der Zahlung. Ist der Beleg vor der
  Leistung bezahlt, aber `offen` gebucht, sieht Buchfink das nicht; sag es dem
  Menschen.
- Die BMF-Kurse lassen sich in Buchfink weder in der App noch über ein
  Werkzeug erfassen. Ohne sie rechnet Buchfink die Steuer mit dem
  EZB-Tageskurs des Belegdatums, und der Hinweis der Vorschau sagt es. Das
  trägt § 16 Abs. 6 UStG nur, wenn das Finanzamt den Tageskurs gestattet;
  frag den Menschen danach und lege sonst die Abweichung zum BMF-Kurs des
  Monats vor (Wertungsfrage, Steuerberater).
- Die Differenz zwischen beiden Kursen bucht Buchfink als Kursaufwand (6880)
  oder Kursertrag (4840).

**Was du übergibst.** In `beleg_buchen_mit_rueckfrage`:

- `waehrung` (ISO-Code, z. B. `USD`),
- `positionen[].netto` in der Fremdwährung, so wie auf dem Beleg,
- `fremdbetrag` als Endsumme des Belegs; Buchfink prüft die Summe der
  Positionen dagegen.

Zeige dem Menschen vor dem Ja den Kurs und den Eurobetrag aus der Vorschau.

**Später.** Die Kursdifferenz bei Zahlung steht in
[bank/skonto-differenzen.md](../bank/skonto-differenzen.md#4-kursdifferenz), die
Bewertung offener Fremdwährungsposten zum Stichtag in
[jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

**Bankkonto in Fremdwährung.** Der Kontoauszugsimport lehnt Umsätze in
Fremdwährung ab ([bank/ablauf.md](../bank/ablauf.md#6-wenn-der-import-ablehnt)).

## 2. Auslage eines Gesellschafters

Der Gesellschafter hat eine Ausgabe der GmbH privat bezahlt und bekommt sie
erstattet.

1. Prüfe, ob die Ausgabe betrieblich veranlasst ist. Zweifel: Wertungsfrage,
   vorlegen; die Folgen (verdeckte Gewinnausschüttung) stehen in
   [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).
2. Prüfe den Leistungsempfänger: Die Rechnung muss die GmbH nennen, sonst
   trägt sie keine Vorsteuer der GmbH. Ab welchem Betrag das gilt und wie der
   Lieferant berichtigt, steht in
   [pflichtangaben.md](pflichtangaben.md#2-die-zehn-pflichtangaben)
   (Nr. 1, Erleichterung bis 250 € in Abschnitt 3, Berichtigung in
   Abschnitt 5).
3. Den Kontakt des Gesellschafters findest du mit `kontakte_suchen`
   (`text`: sein Name). Fehlt er, legst du ihn mit `kontakt_anlegen` an
   (`typ: lieferant`, Name genau wie in der Gesellschafterliste der
   Stammdaten): Buchfink erkennt Gesellschafter und Geschäftsführer am Namen
   und lehnt andere Gläubiger einer Auslage ab.
4. Buche mit `beleg_buchen_mit_rueckfrage`, `bezahlung: auslage` und
   `gesellschafter_kontakt_id`. Buchfink bucht die Erstattungsschuld gegenüber
   dem Gesellschafter (3640), bei einem Geschäftsführer ohne Anteil auf 3500.
   Eine Auslage ist nie Routine.
5. Die Erstattung (Überweisung an den Gesellschafter) ordnest du nach
   [bank/zuordnung.md](../bank/zuordnung.md) zu. Fertig, wenn der Saldo für
   diese Auslage null ist. Verrechnung und Ausweis stehen in
   [gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md).

## 3. Auslage eines Arbeitnehmers

Reisekostenabrechnung und Erstattung an Arbeitnehmer laufen außerhalb von
Buchfink ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
Liegt eine einzelne Rechnung vor, die ein Arbeitnehmer bezahlt hat, leg den
Fall vor; `bezahlung: auslage` gilt nur für Gesellschafter.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
