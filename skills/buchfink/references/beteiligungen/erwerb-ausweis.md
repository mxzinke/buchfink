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

- **Anlagevermögen:** nur Gegenstände, die dauernd dem Geschäftsbetrieb
  dienen sollen ([§ 247 Abs. 2 HGB][hgb247]); alles andere ist
  Umlaufvermögen.
- **Beteiligung:** Anteile, die durch eine dauernde Verbindung dem eigenen
  Geschäftsbetrieb dienen sollen; vermutet ab mehr als einem Fünftel des
  Nennkapitals ([§ 271 Abs. 1 Satz 1, 3 HGB][hgb271]). Genossenschaftsanteile
  gelten nicht als Beteiligung (Satz 5).
- **Verbundenes Unternehmen:** Tochter nach § 290 HGB
  ([§ 271 Abs. 2 HGB][hgb271]), stets bei Mehrheit der Stimmrechte
  ([§ 290 Abs. 2 Nr. 1 HGB][hgb290]).

## 2. Konto wählen

| Anteil | Konto (SKR04) |
|---|---|
| Kapitalgesellschaft, Mehrheit der Stimmrechte (Tochter) | 0804 Anteile an verbundenen Unternehmen, Kapitalgesellschaften |
| Kapitalgesellschaft, dauernde Verbindung ohne Mehrheit | 0850 Beteiligungen an Kapitalgesellschaften |
| Aktien als Daueranlage, keine Beteiligung | 0910 Wertpapiere mit Gewinnbeteiligungsansprüchen, die dem Teileinkünfteverfahren unterliegen |
| Investmentfonds als Daueranlage | 0900 Wertpapiere des Anlagevermögens |
| Festverzinsliche Wertpapiere als Daueranlage | 0920 Festverzinsliche Wertpapiere; Zinsen auf 7010 |
| Aktien, Fonds zur kurzfristigen Anlage | 1510 Sonstige Wertpapiere |
| Anteile an verbundenen Unternehmen im Umlaufvermögen | 1500 |
| Personengesellschaft (GmbH & Co. KG, OHG) | 0803, 0860; Grenze: Buchfink bildet Mitunternehmerschaften nicht ab |
| Genossenschaftsanteile (eG) | 0980 Genossenschaftsanteile zum langfristigen Verbleib; die Rückvergütung bucht `kapitalertrag` nicht: Konto und Steuerfolge mit dem Menschen klären, dann von Hand buchen |

Dividendenkonten: [dividenden.md, Abschnitt 3](dividenden.md#3-buchung);
Fondserträge: [investmentfonds.md, Abschnitt 2](investmentfonds.md#2-ausweis-und-buchung).

## 3. Anschaffungskosten

- Kaufpreis mit Nebenkosten und nachträglichen Anschaffungskosten, abzüglich
  einzeln zuordenbarer Minderungen ([§ 255 Abs. 1 HGB][hgb255]); Nebenkosten
  sind etwa Notar beim Kaufvertrag, Bankgebühren und Provisionen.
- Abziehbare Vorsteuer gehört nicht dazu
  ([holding-umsatzsteuer.md, Abschnitt 3](holding-umsatzsteuer.md#3-vorsteuer-aus-kosten-der-beteiligung)).
- Wertungsfrage: Beratungs- und Prüfungskosten vor der Kaufentscheidung,
  Anschaffungskosten oder Aufwand? Empfehlung Steuerberater; bei kleinen
  Beträgen entscheidet der Mensch.
- Keine planmäßige Abschreibung; sie gilt nur bei zeitlich begrenzter
  Nutzung ([§ 253 Abs. 3 Satz 1 HGB][hgb253]).

## 4. Erwerb buchen

1. **Belege** mit `beleg_ablegen`: notarieller Abtretungsvertrag bei
   GmbH-Anteilen ([§ 15 Abs. 3, 4 GmbHG][gmbhg15]), Kaufabrechnung der Bank,
   bei Gründung die Gründungsurkunde.
2. **Rückfrage** mit Konto aus Abschnitt 2, Anschaffungskosten und
   Gegenkonto.
3. **Buchung:**
   - Kaufpreis über die Bank: `bankumsatz_buchen`, Zeile `konto` auf das
     Anlagekonto; Kurswert und Nebenkosten bei getrennter Abrechnung in
     mehreren Zeilen, beide auf das Anlagekonto.
   - Kaufpreis offen oder direkt an den Verkäufer: `buchung_erfassen` mit
     `beleg_id`, Soll Anlagekonto, Haben Verbindlichkeit.
4. **Kartei und Beteiligungsliste** nach
   [ablauf.md, Abschnitt 2](ablauf.md#2-ablauf-für-jeden-vorgang), Schritte 1
   und 6.

Fertig, wenn Buchung, Anlagegut und Beteiligungsliste übereinstimmen.

## 5. Gründung, Kapitalerhöhung, Einlage in eine Tochter

- **Gründung einer Tochter-GmbH:** Stammeinlage und Aufgeld sind
  Anschaffungskosten; nach Einzahlung Anlagekonto an Bank. Die Tochter ist
  ein eigener Mandant ([gruendung/ablauf.md](../gruendung/ablauf.md)).
- **Zuzahlung in die Kapitalrücklage, Kapitalerhöhung:** nachträgliche
  Anschaffungskosten ([§ 255 Abs. 1 Satz 2 HGB][hgb255]); Anlagekonto an
  Bank, Beleg Beschluss oder schriftliche Erklärung.
- **Verdeckte Einlage** eines Wirtschaftsguts in die Tochter: Die
  Anschaffungskosten der Beteiligung erhöhen sich um dessen Teilwert
  ([§ 6 Abs. 6 Satz 2 EStG][estg6]); bei einem Anteil gilt sie als
  Veräußerung
  ([steuerfreistellung.md, Abschnitt 4](steuerfreistellung.md#4-gewinne-aus-der-veräußerung)).
  Wertungsfrage, Steuerberater.
- **Verzicht auf eine Forderung gegen die Tochter:** Einlage nach
  Werthaltigkeit; Wertungsfrage, Steuerberater, gebucht wird erst danach.

## 6. Darlehen an eine Tochter

- **Ausweis:** dauernd dienend als Ausleihung an verbundene Unternehmen
  (0814) oder an Unternehmen mit Beteiligungsverhältnis (0885), kurzfristig
  als Forderung (1260 bis 1265 bzw. 1280 bis 1285).
- **Beleg:** schriftlicher Vertrag mit Betrag, Zins, Laufzeit, Tilgung,
  Sicherheiten. Auszahlung: Ausleihung an Bank.
- **Zinsen** auf 7012 (verbundene Unternehmen) oder 7011, ohne Umsatzsteuer
  ([holding-umsatzsteuer.md, Abschnitt 5](holding-umsatzsteuer.md#5-darlehen-an-tochtergesellschaften)).
- **Ausfall:** steuerlich
  [steuerfreistellung.md, Abschnitt 6](steuerfreistellung.md#6-darlehen-an-die-tochter),
  Bewertung
  [bewertung-veraeusserung.md, Abschnitt 4](bewertung-veraeusserung.md#4-darlehen-an-die-tochter).
  Zur Fremdüblichkeit lass den Menschen Zins und Sicherheiten im Vertrag
  begründen.

## 7. Pflichten, die aus der Beteiligung folgen

- **Größenklasse:** Eine reine Holding ist keine Kleinstkapitalgesellschaft
  ([jahresabschluss/groessenklassen-anhang.md, Abschnitt 1](../jahresabschluss/groessenklassen-anhang.md#1-merkmale)).
- **Anhang:** Beteiligungsangaben nach [§ 285 Nr. 11 HGB][hgb285]; kleine
  Kapitalgesellschaften sind befreit ([§ 288 Abs. 1 Nr. 1 HGB][hgb288]).
- **Konzernabschluss** bei beherrschendem Einfluss
  ([§ 290 Abs. 1, 2 HGB][hgb290]), größenabhängig befreit
  ([§ 293 HGB][hgb293]): Buchfink erstellt keinen
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet));
  die Pflicht prüft der Steuerberater.
- **Ausländische Beteiligungen:** Erwerb oder Veräußerung von Anteilen an
  einer Körperschaft mit Sitz und Geschäftsleitung im Ausland ist
  mitzuteilen, wenn damit mindestens 10 % erreicht werden oder die
  Anschaffungskosten aller solchen Beteiligungen 150.000 € übersteigen
  ([§ 138 Abs. 2 Satz 1 Nr. 3 AO][ao138]); mit der
  Körperschaftsteuererklärung, spätestens 14 Monate nach Ablauf des Jahres
  ([§ 138 Abs. 5 AO][ao138]).
- **Verlustuntergang bei der Tochter:** Wechseln innerhalb von fünf Jahren
  mehr als 50 % ihrer Anteile, gehen ihre nicht genutzten Verluste unter
  ([steuererklaerungen/kst.md, Abschnitt 8](../steuererklaerungen/kst.md#8-verlustabzug)).
  Weise bei Kauf und Verkauf darauf hin und empfiehl einen Steuerberater.

[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[gmbhg15]: https://www.gesetze-im-internet.de/gmbhg/__15.html
[hgb247]: https://www.gesetze-im-internet.de/hgb/__247.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[hgb271]: https://www.gesetze-im-internet.de/hgb/__271.html
[hgb285]: https://www.gesetze-im-internet.de/hgb/__285.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[hgb290]: https://www.gesetze-im-internet.de/hgb/__290.html
[hgb293]: https://www.gesetze-im-internet.de/hgb/__293.html
