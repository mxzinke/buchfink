# Jahresabschluss

[Fachkonzepte](README.md) · [Pflichten und Erklärungen](pflichten-und-erklaerungen.md) ·
[Anlagen](anlagenverwaltung.md) · [Rechnungsabgrenzung](rechnungsabgrenzung.md)

Anforderungen: GOB-06, BEW-01 bis BEW-09, JAB-01 bis JAB-09.

## Abschlussperiode

Abschlussbuchungen lassen sich erst rechnen, wenn das Jahr gebucht ist. Deshalb
bleibt nach der Festschreibung des letzten Monats eine Abschlussperiode offen,
der „13. Monat“, bis das ganze Jahr festgeschrieben ist. Eine Buchung der
Abschlussperiode trägt den Bilanzstichtag; Abschlussbuchungen und AfA zum
Stichtag gehören von selbst dazu, eine Handbuchung auf Antrag
(`JournalEntry.ClosingPeriod`). Das Kennzeichen geht in die Hash-Kette und in
die Datenüberlassung ein. Eine gewöhnliche Buchung zum Stichtag nimmt der
festgeschriebene Dezember nicht mehr an.

## Bausteine

`domain.AllClosingSteps` legt 15 Bausteine in fester Reihenfolge fest. Die
Reihenfolge ist fachlich: die Steuerrückstellung braucht das Ergebnis, und das
steht erst nach Abgrenzung und Rückstellungen.

1. Abschreibungen
2. Wertaufholung prüfen ([§ 253 Abs. 5 HGB][hgb253])
3. Fremdwährungsbewertung ([§ 256a HGB][hgb256a])
4. Wertberichtigung auf Forderungen
5. Unfertige Leistungen
6. Rechnungsabgrenzung
7. Rückstellungen
8. Umsatzsteuer-Verrechnung
9. Vorsteuerberichtigung nach § 15a UStG
10. Steuerrückstellung
11. Prüfbericht
12. Bilanz und GuV
13. Aufstellen und Feststellen
14. E-Bilanz und Offenlegung
15. Vortrag und Ergebnisverwendung

Ein Baustein ist offen, erledigt oder übersprungen. Überspringen verlangt einen
Grund, weil ein Abschluss ohne Rückstellungen eine Aussage ist und kein
Versehen. Ein übersprungener Baustein lässt sich mit Grund wieder öffnen, bis
die Jahresfestschreibung ([§ 146 Abs. 4 AO][ao146]) oder die Feststellung
([§ 42a GmbHG][gmbhg42a]) den Weg schließt.

## Wertberichtigung auf Forderungen

Forderungen stehen mit dem niedrigeren beizulegenden Wert in der Bilanz
([§ 253 Abs. 4 Satz 2 HGB][hgb253]): einzeln, wo ein Ausfall droht, pauschal
für das allgemeine Risiko der übrigen. Bemessen wird der Nettobetrag, weil die
Umsatzsteuer erst mit der Uneinbringlichkeit berichtigt wird (§ 17 Abs. 2 Nr. 1
UStG). Die Forderung bleibt auf dem Personenkonto stehen; die Bilanz mindert
die Forderungen um die Wertberichtigung (`accounting/receivable_allowance.go`).

| | Bestand | Einstellung | Herabsetzung |
|---|---|---|---|
| Einzelwertberichtigung | **1246** bis 1 Jahr, **1247** darüber | **6923** | **4923** |
| Pauschalwertberichtigung | **1248** bis 1 Jahr, **1249** darüber | **6920** | **4920** |

## Unfertige Leistungen

Ein Projekt, das am Stichtag begonnen, aber weder abgenommen noch als
Teilleistung abgerechnet ist, hat noch keinen Erlös ([§ 252 Abs. 1 Nr. 4
HGB][hgb252]). Seine bisherigen Kosten werden zu Herstellungskosten aktiviert,
höchstens mit dem Wert, der sich aus dem Auftrag ergibt; ein Gewinnanteil gehört
nie dazu (`accounting/work_in_progress.go`). Gebucht wird **1080** Unfertige
Leistungen an **4815** Bestandsveränderungen, im Folgejahr aufgelöst. Erhaltene
Anzahlungen bleiben unter den Verbindlichkeiten oder werden offen abgesetzt
(**1190**, § 268 Abs. 5 Satz 2 HGB).

## Steuerrückstellung

Buchfink rechnet Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer
auf das Ergebnis des Jahres, abzüglich Vorauszahlungen und anrechenbarer
Kapitalertragsteuer (`service/closing_tax_provision.go`). Eine Überzahlung ist
ein Erstattungsanspruch.

- Der Körperschaftsteuersatz folgt dem Veranlagungszeitraum:
  15 % bis 2027, danach ein Punkt weniger je Jahr bis 10 % ab 2032
  ([§ 23 Abs. 1 KStG][kstg23]).
- Den Hebesatz nennt der Gewerbesteuerbescheid oder die Abschluss-Einstellung.
  Fehlt er, rechnet Buchfink nicht.
- Verlustvorträge, steuerfreie Bezüge nach § 8b KStG, Korrekturen außerhalb der
  Bücher wie eine verdeckte Gewinnausschüttung, Hinzurechnungen und Kürzungen
  der Gewerbesteuer sind Angaben zur Rechnung. Sie stehen mit der Rechnung in
  einem Eigenbeleg.

## Gesetzliche Rücklage der UG

Die UG stellt ein Viertel des um einen Verlustvortrag geminderten
Jahresüberschusses in die gesetzliche Rücklage ein ([§ 5a Abs. 3
GmbHG][gmbhg5a]). Buchfink bucht sie im Abschlussjahr über ein Konto der
Ergebnisverwendung, ohne den steuerlichen Gewinn zu mindern. Vor der
Jahresfestschreibung muss sie gebucht sein.

## Aufstellen, Feststellen, Offenlegen

Das Geschäftsjahr durchläuft die Stände offen, aufgestellt, festgestellt und
offengelegt. Jeden Wechsel nimmt der Mensch in der App vor. Ab „festgestellt“
nimmt das Jahr keine Buchung mehr an; die Feststellung lässt sich mit Grund
zurücknehmen, ebenfalls nur in der App. Die Fristen für Aufstellung und
Feststellung richten sich nach der Größenklasse ([§ 267 HGB][hgb267], zwei
aufeinanderfolgende Stichtage): sechs und elf Monate für kleine und
Kleinstgesellschaften, sonst drei und acht. Kleine Gesellschaften legen Bilanz
und Anhang offen, Kleinstgesellschaften nur die Bilanz und können sie
hinterlegen.

Der Beschluss über die Ergebnisverwendung wird in der App erfasst. Eine
Ausschüttung löst die Pflicht zur Kapitalertragsteuer-Anmeldung aus. Der
Saldenvortrag bringt Bestandskonten und offene Posten ins Folgejahr und weist
eine Differenz aus.

[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb256a]: https://www.gesetze-im-internet.de/hgb/__256a.html
[hgb267]: https://www.gesetze-im-internet.de/hgb/__267.html
[ao146]: https://www.gesetze-im-internet.de/ao_1977/__146.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[kstg23]: https://www.gesetze-im-internet.de/kstg_1977/__23.html
