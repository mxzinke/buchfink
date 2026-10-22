# Gewinnausschüttung und Kapitalertragsteuer

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Voraussetzungen
2. Grenzen der Ausschüttung
3. Steuerabzug
4. Zufluss, Anmeldung, Abführung
5. Buchungen in Buchfink
6. Was Buchfink nicht rechnet
7. Vorabausschüttung

## 1. Voraussetzungen

- Ausgeschüttet wird nur auf einen Beschluss der Gesellschafter über die
  Ergebnisverwendung: Form in
  [jahresabschluss/aufstellen-feststellen-offenlegen.md, Abschnitt 3](../jahresabschluss/aufstellen-feststellen-offenlegen.md#3-feststellen),
  Frist in [grundlagen/pflichten.md#ergebnisverwendung](../grundlagen/pflichten.md#ergebnisverwendung),
  Verteilung in
  [jahresabschluss/ergebnisverwendung-ug.md, Abschnitt 3](../jahresabschluss/ergebnisverwendung-ug.md#3-ergebnisverwendung).
- Der Auszahlungstag im Beschluss bestimmt den Zufluss (Abschnitt 4).
- Lege Feststellungs- und Verwendungsbeschluss mit `beleg_ablegen`
  (`art: sonstiges`) ab.

## 2. Grenzen der Ausschüttung

- **Kapitalerhaltung** ([Grundregel](ablauf.md#3-grundregeln)): Bleibt das
  Eigenkapital nach der Auszahlung unter dem Stammkapital (Bilanz zum letzten
  Stichtag und aktuell, `auswertung` mit `art: bilanz`), halte an und
  empfiehl rechtliche Beratung.
- **UG:** Die gesetzliche Rücklage ist nicht ausschüttbar
  ([jahresabschluss/ergebnisverwendung-ug.md, Abschnitt 1](../jahresabschluss/ergebnisverwendung-ug.md#1-gesetzliche-rücklage-der-ug)).
- **Zahlungsfähigkeit** ([Grundregel](ablauf.md#3-grundregeln)): Nach der
  Auszahlung müssen alle fälligen Verbindlichkeiten bezahlbar sein
  (`offene_posten`, Bankstände).

## 3. Steuerabzug

| Größe | Wert | Grundlage |
|---|---|---|
| Kapitalertragsteuer mit Solidaritätszuschlag | Abzug auf Gewinnanteile; Satz und Bemessung in [bank/zinsen-kapest.md](../bank/zinsen-kapest.md#2-kapitalertragsteuer-und-solidaritätszuschlag) | [§ 43 Abs. 1 Satz 1 Nr. 1 EStG][estg43] |
| Kirchensteuer | Satz nach Landesrecht; mindert die Kapitalertragsteuer um 25 % der Kirchensteuer | [§ 43a Abs. 1 Satz 2 EStG][estg43a], [§ 51a Abs. 2c EStG][estg51a] |

Beispiel ohne Kirchensteuer: Ausschüttung 10.000,00 € → Kapitalertragsteuer
2.500,00 €, Solidaritätszuschlag 137,50 €, Auszahlung 7.362,50 €.

- Der Abzug gilt auch gegenüber einer Kapitalgesellschaft als Gesellschafterin
  ([§ 43 Abs. 1 Satz 3 EStG][estg43]).
- Die GmbH behält für Rechnung des Gesellschafters ein
  ([§ 44 Abs. 1 Satz 3 EStG][estg44]) und haftet für die Steuer
  ([§ 44 Abs. 5 Satz 1 EStG][estg44]); der Geschäftsführer haftet bei grober
  Fahrlässigkeit ([§ 69 AO][ao69]).
- Kirchensteuer: Für natürliche Personen fragt die GmbH die
  Kirchensteuerpflicht beim BZSt ab, bei Begründung der Beteiligung und
  jährlich vom 1. September bis 31. Oktober zum Stichtag 31. August
  ([§ 51a Abs. 2c Satz 1 Nr. 3 EStG][estg51a]); erinnere im September. Ob das
  ohne geplante Ausschüttung nötig ist, klärt der Mensch mit dem
  Steuerberater.
- Freistellung: Mit einer Nichtveranlagungs-Bescheinigung des Gesellschafters
  unterbleibt der Abzug ([§ 44a Abs. 1 Satz 4, Abs. 2 Satz 1 Nr. 2 EStG][estg44a]).
  Ein Freistellungsauftrag genügt für Gewinnanteile aus GmbH-Anteilen nicht
  ([§ 44a Abs. 1 Satz 1 Nr. 1, 2 EStG][estg44a]).
- Ausländische Gesellschafter, EU-Muttergesellschaften: Grenze,
  Steuerberater.

## 4. Zufluss, Anmeldung, Abführung

- **Zufluss** am Auszahlungstag laut Beschluss, sonst am Tag nach der
  Beschlussfassung ([§ 44 Abs. 2 Satz 1, 2 EStG][estg44]).
- **Abführung** bei Gewinnausschüttungen im Zeitpunkt des Zuflusses
  ([§ 44 Abs. 1 Satz 5 Halbsatz 2 EStG][estg44]); „bis zum 10. des
  Folgemonats“ gilt nur für andere Kapitalerträge.
- **Anmeldung** innerhalb derselben Frist elektronisch
  ([§ 45a Abs. 1 Satz 1 EStG][estg45a]), in Mein ELSTER als
  „Kapitalertragsteuer-Anmeldung nach Einkommensteuergesetz“
  ([ELSTER-Formularliste][elster]). Auch ohne Abzug ist anzumelden und der
  Grund anzugeben ([§ 45a Abs. 1 Satz 2, 3 EStG][estg45a]).
- **Rundung:** Einbehalt auf den Cent ([§ 44 Abs. 1 Satz 3 EStG][estg44]),
  Solidaritätszuschlag ohne Bruchteile eines Cents ([§ 4 Satz 3 SolZG][solzg4]).
  Die zu einem Zeitpunkt abzuführende Kapitalertragsteuer wird auf volle Euro
  abgerundet ([§ 44 Abs. 1 Satz 6 EStG][estg44]); der Rest bleibt auf 3760.
- **Bekannte Grenze:** Offen lässt der Wortlaut, und Buchfink beantwortet
  nicht:
  1. wem der Abrundungsrest auf 3760 zusteht;
  2. ob die Abrundung auch für den Solidaritätszuschlag gilt
     ([§ 1 Abs. 2 Satz 1 SolZG][solzg1] verweist auf das EStG);
  3. ob je Gesellschafter zu rechnen ist. Buchfink rechnet auf die ganze
     Ausschüttung; je Gesellschafter kann die Summe um Cent abweichen.

  Lege die Frage vor, die den Fall betrifft, und rate zur steuerlichen
  Beratung. Den Rest auf 3760 buchst du erst nach der Antwort mit
  `buchung_erfassen` aus.
- **Säumnis:** Säumniszuschlag nach
  [grundlagen/pflichten.md](../grundlagen/pflichten.md#fristende-zahlung-verlängerung).

Bereite vor: Beschlussdatum, Zuflusstag, je Gesellschafter Bruttobetrag und
Abzüge, Summen für die Anmeldung.

## 5. Buchungen in Buchfink

| Schritt | Buchung (SKR04) | Werkzeug |
|---|---|---|
| Ergebnisverwendung beschlossen | 2970 Gewinnvortrag vor Verwendung an 3519 Verbindlichkeiten gegenüber Gesellschaftern für offene Ausschüttungen (netto) und 3760 Verbindlichkeiten aus Einbehaltungen (KapESt, SolZ) | `ergebnisverwendung_buchen` |
| Auszahlung an Gesellschafter | 3519 an Bank | [bank/ablauf.md](../bank/ablauf.md) |
| Zahlung an das Finanzamt | 3760 an Bank | [bank/ablauf.md](../bank/ablauf.md) |

Fertig, wenn 3519 für diese Ausschüttung auf null steht, 3760 bis auf den
Abrundungsrest (`auswertung`, `art: kontoblatt`), und die Anmeldung mit
Übermittlungsprotokoll als Beleg abgelegt ist.

## 6. Was Buchfink nicht rechnet

- Kirchensteuer und die Minderung der Kapitalertragsteuer durch sie.
- Abstandnahme bei Nichtveranlagungs-Bescheinigung.
- **Einlagenrückgewähr:** Übersteigt eine Leistung den ausschüttbaren Gewinn,
  mindert sie das steuerliche Einlagekonto ([§ 27 Abs. 1 Satz 3, 5 KStG][kstg27]);
  insoweit ist sie beim Gesellschafter keine Einnahme
  ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]) und braucht eine Bescheinigung
  nach [§ 27 Abs. 3 KStG][kstg27].

Dann stimmen die Beträge auf 3519 und 3760 nicht. Lege die Abweichung vor;
die Korrektur buchst du nach Klärung mit dem Steuerberater mit
`buchung_erfassen` (Rückfrage) und dem Beschluss als Beleg.

## 7. Vorabausschüttung

Eine Ausschüttung vor Feststellung des Abschlusses braucht einen Beschluss der
Gesellschafter und unterliegt [§ 30 GmbHG][gmbhg30] und dem
Kapitalertragsteuerabzug wie jede Ausschüttung. Buchfink hat dafür keinen
Ablauf. Grenze: Steuerberater.

[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44]: https://www.gesetze-im-internet.de/estg/__44.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[estg51a]: https://www.gesetze-im-internet.de/estg/__51a.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[ao69]: https://www.gesetze-im-internet.de/ao_1977/__69.html
[elster]: https://www.elster.de/eportal/formulare-leistungen/alleformulare
[solzg1]: https://www.gesetze-im-internet.de/solzg_1995/__1.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
