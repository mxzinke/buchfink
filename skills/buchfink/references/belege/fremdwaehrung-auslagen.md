# Fremdwährung und Auslagen

Rechtsstand: 17. Oktober 2026. Wortlaut des UStG am 17.10.2026 gelesen.
Werkzeugstand: Buchfink 0.3.0.

## Inhalt

1. Beleg in Fremdwährung
2. Auslage eines Gesellschafters
3. Auslage eines Arbeitnehmers

## 1. Beleg in Fremdwährung

Der Beleg zeigt Währung und, soweit umgerechnet, den Kurs
([GoBD Rz. 77][gobd]).

- Aufwand und Verbindlichkeit rechnet Buchfink mit dem EZB-Referenzkurs des
  Belegdatums; einen Kurs von Hand bildet es nicht ab.
- Umsatzsteuer und Vorsteuer: BMF-Durchschnittskurs des Monats der Leistung
  oder der Vereinnahmung vor der Leistung ([§ 16 Abs. 6 Satz 1 UStG][ustg16]);
  Tageskurs nur mit Gestattung des Finanzamts (Satz 3). Ist der Beleg vor der
  Leistung bezahlt, aber `offen` gebucht, sieht Buchfink den Zahlungsmonat
  nicht; sag es dem Menschen.
- Die BMF-Kurse liest der Mensch in der App ein (Einstellungen › Erweitert ›
  „Umsatzsteuer-Umrechnungskurse“; CSV mit Monat `JJJJ-MM`, Währung, Kurs je
  Euro). Fehlt der Monatskurs, nimmt Buchfink den EZB-Tageskurs und die
  Vorschau sagt es: Bitte dann um die Kurse oder frag, ob das Finanzamt den
  Tageskurs gestattet hat.

Erfasse die Positionen in der Währung des Belegs und zeig vor dem Ja Kurs und
Eurobetrag aus der Vorschau. Kursdifferenz bei Zahlung:
[bank/skonto-differenzen.md](../bank/skonto-differenzen.md#4-kursdifferenz);
Stichtag: [jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md);
Kartenzahlung: [bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md).
Umsätze in Fremdwährung auf dem Bankkonto lehnt der Import ab
([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).

## 2. Auslage eines Gesellschafters

Der Gesellschafter hat eine Ausgabe der GmbH privat bezahlt und bekommt sie
erstattet.

1. Betriebliche Veranlassung prüfen; Zweifel sind Wertungsfrage, Folgen in
   [gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md).
2. Die Rechnung muss die GmbH nennen, sonst keine Vorsteuer
   ([pflichtangaben.md](pflichtangaben.md#2-die-zehn-pflichtangaben), Nr. 1).
3. Kontakt mit `kontakte_suchen` (`text`: Name); fehlt er, `kontakt_anlegen`
   mit `typ: lieferant` und dem Namen wie in der Gesellschafterliste der
   Stammdaten, denn daran erkennt Buchfink Gesellschafter und
   Geschäftsführer.
4. `beleg_buchen_mit_rueckfrage` mit `bezahlung: auslage`; die
   Erstattungsschuld steht auf 3640, bei einem Geschäftsführer ohne Anteil auf
   3500.
5. Die Erstattung ordnest du nach [bank/zuordnung.md](../bank/zuordnung.md) zu.
   Fertig, wenn der Saldo dieser Auslage null ist. Verrechnung und Ausweis:
   [gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md).

## 3. Auslage eines Arbeitnehmers

Reisekostenabrechnung und Erstattung an Arbeitnehmer liegen außerhalb von
Buchfink ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
Eine einzelne Rechnung, die ein Arbeitnehmer bezahlt hat, legst du vor;
`bezahlung: auslage` gilt nur für Gesellschafter.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
