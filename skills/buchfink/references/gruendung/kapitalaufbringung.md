# Kapitalaufbringung

Rechtsstand: 17. Oktober 2026. Wortlaut von GmbHG und HGB am 17.10.2026
gelesen; Musterprotokoll aus Anlage 1 zum GmbHG. Konten nach dem SKR04 in
Buchfink.

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
| UG (haftungsbeschränkt) | unter 25.000 €; jeder Geschäftsanteil lautet auf volle Euro, also mindestens 1 € | [§ 5a Abs. 1 GmbHG][gmbhg5a], [§ 5 Abs. 2 GmbHG][gmbhg5] |

Stand: Oktober 2026. Erhöht eine UG ihr Stammkapital auf 25.000 € oder mehr,
gelten die Sonderregeln der UG nicht mehr; die Firma darf sie behalten
([§ 5a Abs. 5 GmbHG][gmbhg5a]).

## 2. Was vor der Anmeldung eingezahlt sein muss

**GmbH** ([§ 7 Abs. 2, 3 GmbHG][gmbhg7]):

- Auf jeden Geschäftsanteil mit Geldeinlage mindestens ein Viertel des
  Nennbetrags.
- Insgesamt müssen die eingezahlten Geldeinlagen plus der Nennbetrag der
  Sacheinlagen die Hälfte des Mindeststammkapitals erreichen, also 12.500 €.
  Maßgebend ist das Mindeststammkapital, nicht das vereinbarte.
- Sacheinlagen sind vor der Anmeldung vollständig zu bewirken.
- Eingezahlt wird nach dem Verhältnis der Geldeinlagen
  ([§ 19 Abs. 1 GmbHG][gmbhg19]).

**UG**: Das Stammkapital ist vor der Anmeldung voll einzuzahlen;
Sacheinlagen sind ausgeschlossen ([§ 5a Abs. 2 GmbHG][gmbhg5a]).

**Musterprotokoll**: Die Einlage ist in Geld zu erbringen, sofort in voller
Höhe oder zu 50 % sofort und im Übrigen nach Beschluss; bei der UG ist nur die
volle Einzahlung zulässig ([Anlage 1 GmbHG][gmbhganl1], Nr. 3 mit Hinweis 3).

Beispiele, rechnerisch:

| Fall | Viertel je Anteil | Untergrenze gesamt | vor der Anmeldung einzuzahlen |
|---|---|---|---|
| GmbH 25.000 €, ein Gesellschafter | 6.250 € | 12.500 € | 12.500 € |
| GmbH 25.000 €, Anteile 15.000 € und 10.000 € | 3.750 € und 2.500 € | 12.500 € | 12.500 €, nach Verhältnis 7.500 € und 5.000 € |
| GmbH 100.000 €, vier Anteile zu 25.000 € | je 6.250 € | 12.500 € | 25.000 € |
| UG 1.000 € | – | – | 1.000 € |

Stand: Oktober 2026. Buchfink rechnet dieselbe Regel
(`FoundationRules.RequiredPaidIn`): Viertel je Anteil, mindestens 12.500 €,
Sacheinlagen mit vollem Nennbetrag.

Den Rest fordern die Gesellschafter durch Beschluss ein
([§ 46 Nr. 2 GmbHG][gmbhg46]). Von der Einlagepflicht kann kein Gesellschafter
befreit werden; aufrechnen darf er nur in engen Grenzen
([§ 19 Abs. 2 GmbHG][gmbhg19]). Der Anspruch verjährt in zehn Jahren ab
Entstehung ([§ 19 Abs. 6 Satz 1 GmbHG][gmbhg19]).

## 3. Buchungen

| Vorgang | Soll | Haben | Werkzeug |
|---|---|---|---|
| Zeichnung, eingeforderter Teil (Beurkundungstag) | 1298 Ausstehende Einlagen auf das gezeichnete Kapital, eingefordert | 2900 Gezeichnetes Kapital | `buchung_erfassen` (Rückfrage) |
| Zeichnung, nicht eingeforderter Teil | 2910 Ausstehende Einlagen, nicht eingefordert | 2900 Gezeichnetes Kapital | `buchung_erfassen` (Rückfrage) |
| Einzahlung | 1800 Bank | 1298 | `bankumsatz_buchen` oder `buchung_erfassen` |
| spätere Einforderung | 1298 | 2910 | `buchung_erfassen` mit Beschluss als Beleg |

Ausweis: Nicht eingeforderte ausstehende Einlagen werden vom gezeichneten
Kapital offen abgesetzt; eingeforderte, noch nicht eingezahlte stehen gesondert
unter den Forderungen ([§ 272 Abs. 1 HGB][hgb272]).

Rückfrage-Beispiel:

```
[1] Beleg B-2026-0001 · Gesellschaftsvertrag · 25.000,00 € · 03.11.2026
Sachverhalt: Gründung Muster GmbH i. G., Stammkapital 25.000 €, voll eingefordert.
Vorschlag:   1298 Ausstehende Einlagen, eingefordert 25.000,00 an 2900 Gezeichnetes Kapital 25.000,00
Warum ich frage: Handbuchung; Gründungsbuchungen haben kein eigenes Werkzeug.
Bitte wähle: (a) so buchen  (b) Teil nicht eingefordert (2910)  (c) zurückstellen
```

## 4. Sacheinlage

- Gegenstand und Nennbetrag stehen im Vertrag; die Gesellschafter erstellen
  einen Sachgründungsbericht ([§ 5 Abs. 4 GmbHG][gmbhg5]). Verträge und
  Wertnachweise gehören zur Anmeldung ([§ 8 Abs. 1 Nr. 4, 5 GmbHG][gmbhg8]).
- Erreicht der Wert bei der Anmeldung den Nennbetrag nicht, zahlt der
  Gesellschafter die Differenz in Geld; der Anspruch verjährt zehn Jahre nach
  der Eintragung ([§ 9 GmbHG][gmbhg9]).
- Buchfink bucht keine Sacheinlage von selbst. Buche sie nur nach Rückfrage:
  Konto des eingebrachten Gegenstands an 1298, Wert laut Nachweis.
- Wertungsfrage: Bewertung, Umsatzsteuer auf die Einbringung und ein Wert
  über dem Nennbetrag. Empfiehl einen Steuerberater.

## 5. Verdeckte Sacheinlage und Hin- und Herzahlen

- **Verdeckte Sacheinlage:** Ist eine Geldeinlage wirtschaftlich und nach einer
  Abrede bei der Übernahme ganz oder teilweise eine Sacheinlage, etwa wenn die
  GmbH mit dem eingezahlten Geld einen Gegenstand des Gesellschafters kauft,
  bleibt die Einlagepflicht bestehen. Der Wert des Gegenstands wird angerechnet,
  aber erst nach der Eintragung; die Beweislast trägt der Gesellschafter
  ([§ 19 Abs. 4 GmbHG][gmbhg19]).
- **Hin- und Herzahlen:** Fließt die Einlage aufgrund einer vorher getroffenen
  Vereinbarung an den Gesellschafter zurück (etwa als Darlehen), befreit das
  nur, wenn ein vollwertiger, jederzeit fälliger oder fristlos kündbarer
  Rückgewähranspruch besteht und die Leistung in der Anmeldung angegeben wird
  ([§ 19 Abs. 5 GmbHG][gmbhg19]).
- Erkennst du kurz nach der Einzahlung eine Zahlung an einen Gesellschafter
  oder einen Kauf von ihm, halte an und lege den Fall vor. Darlehen an
  Gesellschafter regelt buchfink-gesellschafter.

## 6. Unterbilanz am Eintragungstag

- Ist das Stammkapital bei der Eintragung ganz oder teilweise verbraucht,
  haften die Gesellschafter nach der Rechtsprechung des BGH anteilig für die
  Differenz (Unterbilanz- oder Vorbelastungshaftung; grundlegend
  BGH vom 09.03.1981, II ZR 54/80, BGHZ 80, 129; Fundstelle über
  Sekundärquellen geprüft). Eine Norm im GmbHG gibt es dafür nicht.
- Prüfe zur Eintragung die Salden zum Eintragungstag: `auswertung`
  (`art: susa`) nennt die bebuchten Konten, `auswertung` (`art: kontoblatt`,
  `bis: <Eintragungstag>`) den Saldo je Konto. Reinvermögen = Vermögensgegenstände
  einschließlich ausstehender Einlagen minus Schulden. Liegt es unter dem
  Stammkapital, nenne dem Menschen die Differenz und empfiehl Notar oder
  Steuerberater.
- Gründungsaufwand mindert das Reinvermögen sofort, weil er nicht aktiviert
  werden darf ([§ 248 Abs. 1 Nr. 1 HGB][hgb248]). Wertungsfrage: Ob der im
  Vertrag übernommene Gründungsaufwand die Haftung mindert, beurteilt ein
  Berater.
- Buchfink kann die Unterbilanz aus den Gründungsangaben intern rechnen
  (`FoundationService.Unterbilanz`). Die App zeigt sie seit dem Umbau nicht
  mehr an, und ein Werkzeug dafür gibt es nicht; rechne deshalb selbst.

[gmbhg5]: https://www.gesetze-im-internet.de/gmbhg/__5.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg8]: https://www.gesetze-im-internet.de/gmbhg/__8.html
[gmbhg9]: https://www.gesetze-im-internet.de/gmbhg/__9.html
[gmbhg19]: https://www.gesetze-im-internet.de/gmbhg/__19.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhganl1]: https://www.gesetze-im-internet.de/normengrafiken/bgbl1_2021/j3364_0010.pdf
[hgb248]: https://www.gesetze-im-internet.de/hgb/__248.html
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
