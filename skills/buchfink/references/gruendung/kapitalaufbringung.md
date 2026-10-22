# Kapitalaufbringung

Rechtsstand: 17. Oktober 2026. Konten nach dem SKR04 in Buchfink.

## Inhalt

1. Stammkapital
2. Was vor der Anmeldung eingezahlt sein muss
3. Buchungen
4. Sacheinlage
5. Verdeckte Sacheinlage und Hin- und Herzahlen
6. Unterbilanz am Eintragungstag

## 1. Stammkapital

| Rechtsform | Stammkapital | Grundlage |
|---|---|---|
| GmbH | mindestens 25.000 € | [§ 5 Abs. 1 GmbHG][gmbhg5] |
| UG (haftungsbeschränkt) | unter 25.000 €; Geschäftsanteile auf volle Euro, also mindestens 1 € | [§ 5a Abs. 1 GmbHG][gmbhg5a], [§ 5 Abs. 2 GmbHG][gmbhg5] |

Erhöht eine UG auf 25.000 € oder mehr, entfallen die Sonderregeln der UG; die
Firma darf sie behalten ([§ 5a Abs. 5 GmbHG][gmbhg5a]).

## 2. Was vor der Anmeldung eingezahlt sein muss

**GmbH** ([§ 7 Abs. 2, 3 GmbHG][gmbhg7]):

- Auf jeden Geschäftsanteil mit Geldeinlage mindestens ein Viertel des
  Nennbetrags.
- Eingezahlte Geldeinlagen plus Nennbetrag der Sacheinlagen erreichen
  mindestens 12.500 €, die Hälfte des Mindeststammkapitals (nicht des
  vereinbarten).
- Sacheinlagen sind vor der Anmeldung voll zu bewirken.
- Eingezahlt wird nach dem Verhältnis der Geldeinlagen
  ([§ 19 Abs. 1 GmbHG][gmbhg19]).

Beispiel: 100.000 € in vier Anteilen zu 25.000 € verlangen je 6.250 €, also
25.000 €; 25.000 € in Anteilen zu 15.000 € und 10.000 € verlangen 12.500 €,
nach Verhältnis 7.500 € und 5.000 €.

**UG**: Stammkapital vor der Anmeldung voll einzahlen; Sacheinlagen sind
ausgeschlossen ([§ 5a Abs. 2 GmbHG][gmbhg5a]).

**Musterprotokoll**: nur Geldeinlage, sofort voll oder zu 50 % sofort und im
Übrigen nach Beschluss; die UG zahlt voll ein
([Anlage 1 GmbHG][gmbhganl1], Nr. 3 mit Hinweis 3).

Den Rest fordern die Gesellschafter durch Beschluss ein
([§ 46 Nr. 2 GmbHG][gmbhg46]). Kein Gesellschafter kann von der
Einlagepflicht befreit werden; aufrechnen darf er nur in engen Grenzen
([§ 19 Abs. 2 GmbHG][gmbhg19]).

## 3. Buchungen

| Vorgang | Soll | Haben | Beleg |
|---|---|---|---|
| Zeichnung, eingeforderter Teil (Beurkundungstag) | 1298 Ausstehende Einlagen, eingefordert | 2900 Gezeichnetes Kapital | Gesellschaftsvertrag |
| Zeichnung, nicht eingeforderter Teil | 2910 Ausstehende Einlagen, nicht eingefordert | 2900 | Gesellschaftsvertrag |
| Einzahlung | 1800 Bank | 1298 | Kontoauszug |
| spätere Einforderung | 1298 | 2910 | Beschluss der Gesellschafter |

Ausweis: Nicht eingeforderte ausstehende Einlagen setzt die Bilanz offen vom
gezeichneten Kapital ab; eingeforderte, noch nicht eingezahlte stehen
gesondert unter den Forderungen ([§ 272 Abs. 1 HGB][hgb272]).

## 4. Sacheinlage

- Gegenstand und Nennbetrag stehen im Vertrag, dazu ein Sachgründungsbericht
  ([§ 5 Abs. 4 GmbHG][gmbhg5]); Verträge und Wertnachweise gehören zur
  Anmeldung ([§ 8 Abs. 1 Nr. 4, 5 GmbHG][gmbhg8]).
- Bleibt der Wert bei der Anmeldung unter dem Nennbetrag, zahlt der
  Gesellschafter die Differenz in Geld ([§ 9 GmbHG][gmbhg9]).
- Buchung nach Rückfrage: Konto des Gegenstands an 1298, Wert laut Nachweis.
- Wertungsfrage: Bewertung, Umsatzsteuer auf die Einbringung, Wert über dem
  Nennbetrag. Empfiehl einen Steuerberater.

## 5. Verdeckte Sacheinlage und Hin- und Herzahlen

- **Verdeckte Sacheinlage:** Kauft die GmbH nach einer Abrede bei der
  Übernahme mit der Geldeinlage einen Gegenstand des Gesellschafters, bleibt
  die Einlagepflicht bestehen. Der Wert wird erst nach der Eintragung
  angerechnet; die Beweislast trägt der Gesellschafter
  ([§ 19 Abs. 4 GmbHG][gmbhg19]).
- **Hin- und Herzahlen:** Fließt die Einlage nach vorheriger Abrede zurück
  (etwa als Darlehen), befreit sie nur bei vollwertigem, jederzeit fälligem
  oder fristlos kündbarem Rückgewähranspruch, der in der Anmeldung angegeben
  ist ([§ 19 Abs. 5 GmbHG][gmbhg19]).
- Darlehen an Gesellschafter: [bank/darlehen.md](../bank/darlehen.md).

## 6. Unterbilanz am Eintragungstag

- Ist das Stammkapital bei der Eintragung ganz oder teilweise verbraucht,
  haften die Gesellschafter anteilig für die Differenz zum Reinvermögen
  (Unterbilanzhaftung, Richterrecht: BGH vom 09.03.1981, II ZR 54/80,
  BGHZ 80, 129).
- `auswertung` (`art: unterbilanz`) rechnet bis zur Eintragung auf heute
  (`endgueltig: false`), danach auf den Eintragungstag. `gedeckt` ist das
  Minimum aus Gründungsaufwand laut Satzung und `unterdeckung`; `haftung` ist
  der Rest, verteilt nach Geschäftsanteilen. Die App zeigt dieselbe Rechnung
  unter Unternehmen › Stammdaten.
- Prüfauftrag: Buchfink unterstellt, dass die Unterdeckung bis zur Höhe der
  Klausel aus Gründungsaufwand stammt
  ([gruendungskosten.md, Abschnitt 1](gruendungskosten.md#1-bilanz-kein-aktivposten)).
  Stell dem Menschen `unterdeckung`, `gedeckt` und den bis zur Eintragung
  gebuchten Gründungsaufwand nebeneinander. Ob die Unterdeckung aus
  Gründungsaufwand oder aus Anlaufverlust stammt und ob der Gründungsaufwand
  die Haftung mindert, beurteilt er, im Zweifel mit einem Berater
  (Wertungsfrage).
- Ist `haftung` größer als null, nenne Betrag und Anteile und empfiehl Notar
  oder Steuerberater.

[gmbhg5]: https://www.gesetze-im-internet.de/gmbhg/__5.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg8]: https://www.gesetze-im-internet.de/gmbhg/__8.html
[gmbhg9]: https://www.gesetze-im-internet.de/gmbhg/__9.html
[gmbhg19]: https://www.gesetze-im-internet.de/gmbhg/__19.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhganl1]: https://www.gesetze-im-internet.de/normengrafiken/bgbl1_2021/j3364_0010.pdf
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
