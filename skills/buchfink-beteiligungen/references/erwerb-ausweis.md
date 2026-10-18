# Erwerb und Ausweis von Finanzanlagen

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, EStG, GmbHG und AO am
17.10.2026 auf gesetze-im-internet.de gelesen. Konten nach dem SKR04 und dem
Anlagenkatalog von Buchfink.

## Inhalt

1. Begriffe
2. Konto wählen
3. Anschaffungskosten
4. Erwerb buchen
5. Gründung, Kapitalerhöhung, Einlage in eine Tochter
6. Darlehen an eine Tochter
7. Pflichten, die aus der Beteiligung folgen

## 1. Begriffe

- **Anlagevermögen** sind nur Gegenstände, die dauernd dem Geschäftsbetrieb
  dienen sollen ([§ 247 Abs. 2 HGB][hgb247]). Alles andere ist
  Umlaufvermögen.
- **Beteiligungen** sind Anteile an anderen Unternehmen, die dem eigenen
  Geschäftsbetrieb durch eine dauernde Verbindung dienen sollen. Eine
  Beteiligung wird vermutet, wenn die Anteile ein Fünftel des Nennkapitals
  überschreiten ([§ 271 Abs. 1 Satz 1, 3 HGB][hgb271]).
- **Verbundene Unternehmen** sind Mutter- und Tochterunternehmen nach § 290
  HGB ([§ 271 Abs. 2 HGB][hgb271]). Beherrschender Einfluss besteht stets bei
  der Mehrheit der Stimmrechte ([§ 290 Abs. 2 Nr. 1 HGB][hgb290]).
- **Bilanzposten:** Finanzanlagen gliedern sich in Anteile an verbundenen
  Unternehmen, Ausleihungen an verbundene Unternehmen, Beteiligungen,
  Ausleihungen an Unternehmen mit Beteiligungsverhältnis, Wertpapiere des
  Anlagevermögens und sonstige Ausleihungen ([§ 266 Abs. 2 A. III. HGB][hgb266]).
  Im Umlaufvermögen stehen Anteile an verbundenen Unternehmen und sonstige
  Wertpapiere (B. III.). Eine kleine GmbH zeigt nur die Posten mit Buchstaben
  und römischen Zahlen ([§ 266 Abs. 1 Satz 3 HGB][hgb266]); die Konten trennen
  trotzdem fein, weil GuV und Überleitung es brauchen.

## 2. Konto wählen

| Anteil | Konto (SKR04) | Dividenden auf |
|---|---|---|
| Kapitalgesellschaft, Mehrheit der Stimmrechte (Tochter) | 0804 Anteile an verbundenen Unternehmen, Kapitalgesellschaften | 7006 |
| Kapitalgesellschaft, dauernde Verbindung ohne Mehrheit | 0850 Beteiligungen an Kapitalgesellschaften | 7005 |
| Aktien als Daueranlage, keine Beteiligung | 0910 Wertpapiere mit Gewinnbeteiligungsansprüchen, die dem Teileinkünfteverfahren unterliegen | 7014 |
| Investmentfonds als Daueranlage | 0900 Wertpapiere des Anlagevermögens | 7010 |
| Festverzinsliche Wertpapiere als Daueranlage | 0920 Festverzinsliche Wertpapiere | 7010 |
| Aktien, Fonds zur kurzfristigen Anlage | 1510 Sonstige Wertpapiere | 7103 (Aktien), 7100 (Fonds) |
| Anteile an verbundenen Unternehmen im Umlaufvermögen | 1500 | 7104 |
| Personengesellschaft (GmbH & Co. KG, OHG) | 0803, 0860 | Grenze: Buchfink bildet Mitunternehmerschaften nicht ab |

Die Bezeichnungen der Dividendenkonten nennen § 8b KStG; die Konten 7005,
7006, 7014, 7015, 7103 und 7104 tragen im SKR04 die Kennzeichen für
Gewerbe- und Körperschaftsteuer. Buchfink selbst schlägt für Anteile das
Konto 7000 vor, für Wertpapiere 7010 und für unbekannte Konten 7100
(`AssetIncomeAccount`). Weiche davon ab, wenn du über das Werkzeug buchst.

## 3. Anschaffungskosten

- Anschaffungskosten sind die Aufwendungen, um den Gegenstand zu erwerben,
  einschließlich der Nebenkosten und der nachträglichen Anschaffungskosten;
  einzeln zuordenbare Minderungen sind abzusetzen
  ([§ 255 Abs. 1 HGB][hgb255]).
- Typische Nebenkosten: Notargebühren für den Kaufvertrag, Bankgebühren und
  Provisionen beim Wertpapierkauf.
- Wertungsfrage: Gehören Beratungs- und Prüfungskosten vor der
  Kaufentscheidung zu den Anschaffungskosten oder sind sie Aufwand? Lege das
  mit Empfehlung Steuerberater vor; bei kleinen Beträgen entscheidet der
  Mensch.
- Abziehbare Vorsteuer gehört nicht zu den Anschaffungskosten. Ob die Holding
  sie abziehen darf, regelt die Referenz zur Umsatzsteuer der Holding (Liste
  in SKILL.md).
- Finanzanlagen nutzen sich nicht ab und werden nicht planmäßig
  abgeschrieben ([§ 253 Abs. 3 Satz 1 HGB][hgb253]: nur bei zeitlich
  begrenzter Nutzung).

## 4. Erwerb buchen

1. **Belege:** Für GmbH-Anteile den notariellen Abtretungsvertrag
   ([§ 15 Abs. 3, 4 GmbHG][gmbhg15]); für Wertpapiere die Kaufabrechnung der
   Bank; bei Gründung die Gründungsurkunde. Lege sie mit `beleg_ablegen` ab.
2. **Rückfrage** mit Konto aus Abschnitt 2, Anschaffungskosten und
   Gegenkonto.
3. **Buchung:**
   - Kaufpreis über die Bank: `bankumsatz_buchen`, Zeile `konto` auf das
     Anlagekonto. Mehrere Zeilen, wenn die Abrechnung
     Kurswert und Nebenkosten getrennt zeigt; beide gehören auf das
     Anlagekonto.
   - Kaufpreis noch offen oder an den Verkäufer direkt: `buchung_erfassen`
     mit `beleg_id`, Soll Anlagekonto, Haben Verbindlichkeit.
4. **Kartei:** Das Anlagegut entsteht aus der Zugangsbuchung, nach
   Rückfrage mit `abschlussbuchung` (`art: anlage_aktivieren`,
   `klasse: finanzanlage`). Die Felder der Finanzanlage bildet Buchfink noch
   nicht ab, weder als Werkzeug noch in der App: Kennung (ISIN, WKN oder
   Handelsregisternummer), Beteiligungsquote, Kennzeichen „Anteil an einer
   Kapitalgesellschaft“ (für GmbH- und AG-Anteile, nie für Fonds), Fondsart
   (nur Fonds), Stückzahl, Notierungswährung. Halte sie in der
   Beteiligungsliste fest.
5. **Beteiligungsliste** fortschreiben (SKILL.md, Schritt 1).

Fertig, wenn Buchung, Anlagegut und Beteiligungsliste übereinstimmen.

## 5. Gründung, Kapitalerhöhung, Einlage in eine Tochter

- **Gründung einer Tochter-GmbH:** Die übernommene Stammeinlage (und ein
  Aufgeld) sind Anschaffungskosten. Buchung nach Einzahlung: Anlagekonto an
  Bank. Die Tochter ist ein eigener Mandant; ihre Gründung bucht
  buchfink-gruendung.
- **Zuzahlung in die Kapitalrücklage der Tochter, Kapitalerhöhung:**
  nachträgliche Anschaffungskosten ([§ 255 Abs. 1 Satz 2 HGB][hgb255]).
  Buchung: Anlagekonto an Bank. Beleg: Beschluss oder schriftliche Erklärung.
- **Verdeckte Einlage** (die Holding überträgt der Tochter ein Wirtschaftsgut
  ohne angemessenes Entgelt, aus Gründen des Gesellschaftsverhältnisses):
  Die Anschaffungskosten der Beteiligung erhöhen sich um den Teilwert des
  eingelegten Wirtschaftsguts ([§ 6 Abs. 6 Satz 2 EStG][estg6]). Die
  verdeckte Einlage eines Anteils gilt als Veräußerung
  ([§ 8b Abs. 2 Satz 6 KStG][kstg8b]). Wertungsfrage, Steuerberater.
- **Verzicht auf eine Forderung gegen die Tochter:** Ob und in welcher Höhe
  eine Einlage vorliegt, hängt von der Werthaltigkeit ab. Wertungsfrage,
  Steuerberater; gebucht wird erst nach der Klärung.

## 6. Darlehen an eine Tochter

- **Ausweis:** Ausleihungen an verbundene Unternehmen
  (0814 Kapitalgesellschaften) oder an Unternehmen mit Beteiligungsverhältnis
  (0885), wenn sie dauernd dienen sollen; kurzfristige Darlehen als Forderung
  (1260 bis 1265 bzw. 1280 bis 1285).
- **Beleg:** schriftlicher Darlehensvertrag mit Betrag, Zins, Laufzeit,
  Tilgung und Sicherheiten. Buchung der Auszahlung: Ausleihung an Bank.
- **Zinsen:** Ertrag auf 7012 (verbundene Unternehmen) oder 7011. Die
  Kreditgewährung ist umsatzsteuerfrei
  ([§ 4 Nr. 8 Buchst. a UStG][ustg4]); die Folgen für die Vorsteuer stehen in
  der Referenz zur Umsatzsteuer der Holding (Liste in SKILL.md). Buche die
  Zinsen ohne Umsatzsteuer.
- **Ausfall:** Teilwertabschreibungen auf Darlehen eines zu mehr als einem
  Viertel beteiligten Gesellschafters sind grundsätzlich nicht abziehbar
  ([§ 8b Abs. 3 Satz 4 KStG][kstg8b]); Ausnahme bei Fremdüblichkeit
  (Satz 7). Einzelheiten in der Referenz zu Bewertung und Veräußerung (Liste
  in SKILL.md).
- Wertungsfrage: Sind Zins und Sicherheiten fremdüblich? Nur dann greift die
  Ausnahme. Lass den Menschen die Gründe im Vertrag festhalten.

## 7. Pflichten, die aus der Beteiligung folgen

- **Größenklasse:** Eine Gesellschaft, deren einziger Zweck Erwerb,
  Verwaltung und Verwertung von Beteiligungen ist, ohne in deren Verwaltung
  einzugreifen, ist keine Kleinstkapitalgesellschaft
  ([§ 267a Abs. 3 Nr. 3 HGB][hgb267a]). buchfink-jahresabschluss bestimmt die
  Klasse.
- **Anhang:** Angaben zu Beteiligungen (Name, Sitz, Anteil, Eigenkapital,
  Ergebnis) verlangt [§ 285 Nr. 11 HGB][hgb285]; kleine Kapitalgesellschaften
  sind davon befreit ([§ 288 Abs. 1 Nr. 1 HGB][hgb288]).
- **Konzernabschluss:** Kann die GmbH auf eine Tochter beherrschenden
  Einfluss ausüben, ist sie Mutterunternehmen
  ([§ 290 Abs. 1, 2 HGB][hgb290]); größenabhängige Befreiungen regelt
  [§ 293 HGB][hgb293]. Buchfink erstellt keinen Konzernabschluss
  (buchfink-grundlagen, Grenzen). Lass die Pflicht vom Steuerberater prüfen.
- **Ausländische Beteiligungen:** Erwerb oder Veräußerung von Anteilen an
  einer Körperschaft mit Sitz und Geschäftsleitung im Ausland ist dem
  Finanzamt mitzuteilen, wenn damit mindestens 10 % erreicht werden oder die
  Anschaffungskosten aller solchen Beteiligungen 150.000 € übersteigen
  ([§ 138 Abs. 2 Satz 1 Nr. 3 AO][ao138]); die Mitteilung geht mit der
  Körperschaftsteuererklärung, spätestens 14 Monate nach Ablauf des Jahres
  ([§ 138 Abs. 5 AO][ao138]). Stand: Oktober 2026.
- **Verlustuntergang bei der Tochter:** Erwirbt die GmbH innerhalb von fünf
  Jahren mehr als 50 % der Anteile, gehen nicht genutzte Verluste der Tochter
  unter ([§ 8c Abs. 1 Satz 1 KStG][kstg8c]). Weise beim Kauf darauf hin.

[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[gmbhg15]: https://www.gesetze-im-internet.de/gmbhg/__15.html
[hgb247]: https://www.gesetze-im-internet.de/hgb/__247.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[hgb266]: https://www.gesetze-im-internet.de/hgb/__266.html
[hgb267a]: https://www.gesetze-im-internet.de/hgb/__267a.html
[hgb271]: https://www.gesetze-im-internet.de/hgb/__271.html
[hgb285]: https://www.gesetze-im-internet.de/hgb/__285.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[hgb290]: https://www.gesetze-im-internet.de/hgb/__290.html
[hgb293]: https://www.gesetze-im-internet.de/hgb/__293.html
[kstg8b]: https://www.gesetze-im-internet.de/kstg_1977/__8b.html
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
