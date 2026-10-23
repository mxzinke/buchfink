# Gewerbesteuer auf Beteiligungserträge

Rechtsstand: 17. Oktober 2026. Wortlaut von GewStG, KStG und InvStG am
17.10.2026 auf gesetze-im-internet.de gelesen, dazu die ELSTER-Anleitung zur
Gewerbesteuererklärung 2025.

Rechenweg, Messzahl, Hebesatz und Abgabe regelt
[steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md). Buchfink
rechnet die Beteiligungszeilen in `erklaerung` (`art: gewst`), du prüfst sie.

## Inhalt

1. Ausgangspunkt
2. Dividenden nach Quote
3. Ausländische Gesellschaften
4. Verluste und Teilwertabschreibungen
5. Finanzierungskosten der Beteiligung
6. Investmentfonds
7. Angaben für die Erklärung

## 1. Ausgangspunkt

Gewerbeertrag ist der nach dem KStG ermittelte Gewinn mit Hinzurechnungen und
Kürzungen
([steuererklaerungen/gewst.md, Abschnitt 2](../steuererklaerungen/gewst.md#2-rechenweg));
was § 8b KStG herausnimmt, fehlt schon im Ausgangswert. Veräußerungsgewinne
und Teilwertabschreibungen folgen der Körperschaftsteuer; § 8 Nr. 5 erfasst
nur Gewinnanteile.

## 2. Dividenden nach Quote

| Quote | Körperschaftsteuer | Gewerbesteuer |
|---|---|---|
| unter 10 % zu Beginn des Kalenderjahres | Dividende voll angesetzt ([steuerfreistellung.md, Abschnitt 3](steuerfreistellung.md#3-streubesitz-unter-10-)) | bleibt voll; keine Hinzurechnung |
| mindestens 10 %, aber zu Beginn des Erhebungszeitraums unter 15 % | 95 % frei ([steuerfreistellung.md, Abschnitt 2](steuerfreistellung.md#2-dividenden-und-andere-bezüge)) | Hinzurechnung nach [§ 8 Nr. 5 GewStG][gewstg8] |
| zu Beginn des Erhebungszeitraums mindestens 15 %, inländische nicht steuerbefreite Kapitalgesellschaft | 95 % frei | keine Hinzurechnung; die 5 % bleiben im Gewerbeertrag ([§ 9 Nr. 2a Satz 4 GewStG][gewstg9]) |

- **§ 8 Nr. 5 GewStG** rechnet die nach § 8b Abs. 1 KStG außer Ansatz
  bleibenden Gewinnanteile hinzu, soweit § 9 Nr. 2a oder 7 nicht greift,
  abzüglich der nach § 8b Abs. 5 KStG nicht abziehbaren 5 %. Bei 100.000 €
  Dividende: 95.000 € Hinzurechnung, die Dividende ist damit voll
  gewerbesteuerpflichtig.
- **§ 9 Nr. 2a GewStG** (Satz 1): Weil die 5 % nach Satz 4 keine
  Gewinnanteile sind, bleibt bei der GmbH nichts zu kürzen; die Wirkung ist,
  dass § 8 Nr. 5 nicht greift.
- **Stichtage.** § 8b Abs. 4 Satz 6 KStG lässt einen Erwerb von mindestens
  10 % auf den Beginn des Kalenderjahres zurückwirken
  ([steuerfreistellung.md, Abschnitt 3](steuerfreistellung.md#3-streubesitz-unter-10-));
  § 9 Nr. 2a GewStG nicht. Die Dividende einer im Laufe des Jahres erworbenen
  Tochter ist im Erwerbsjahr bei der Körperschaftsteuer zu 95 % frei und bei
  der Gewerbesteuer hinzuzurechnen.

## 3. Ausländische Gesellschaften

- Kürzung bei mindestens 15 % zu Beginn des Erhebungszeitraums an einer
  Kapitalgesellschaft mit Sitz und Geschäftsleitung im Ausland
  ([§ 9 Nr. 7 GewStG][gewstg9]); eine niedrigere Mindestbeteiligung nach
  Doppelbesteuerungsabkommen geht vor ([§ 9 Nr. 8 GewStG][gewstg9]).
- Als Betriebsausgabe abgezogene ausländische Steuern auf gekürzte Gewinne
  werden hinzugerechnet ([§ 8 Nr. 12 GewStG][gewstg8]).
- Ausländische Beteiligungen: Steuerberater.

## 4. Verluste und Teilwertabschreibungen

[§ 8 Nr. 10 GewStG][gewstg8] rechnet ausschüttungsbedingte
Teilwertabschreibungen und Veräußerungsverluste hinzu, soweit sie auf nach
§ 9 Nr. 2a, 7 oder 8 gekürzte Ausschüttungen zurückgehen. Beim Anteil an einer
Kapitalgesellschaft hat § 8b Abs. 3 Satz 3 KStG die Minderung schon aus dem
Gewinn genommen; sie wird nicht ein zweites Mal hinzugerechnet
([steuererklaerungen/gewst.md, Abschnitt 3](../steuererklaerungen/gewst.md#3-hinzurechnungen)).

## 5. Finanzierungskosten der Beteiligung

- Zinsen für den Kaufpreis sind Entgelte für Schulden
  ([§ 8 Nr. 1 Buchst. a GewStG][gewstg8];
  [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)).
- Aufwendungen in unmittelbarem Zusammenhang mit Gewinnanteilen mindern den
  Kürzungsbetrag nach § 9 Nr. 2a; insoweit gilt § 8 Nr. 1 nicht
  ([§ 9 Nr. 2a Satz 3 GewStG][gewstg9]).
- Finanzierungszinsen einer Holding bei § 8b-freier Dividende:
  Wertungsfrage, Steuerberater.

## 6. Investmentfonds

Die Teilfreistellung zählt beim Gewerbeertrag nur zur Hälfte
([§ 20 Abs. 5 InvStG][invstg20]); Sätze je Fondsart in
[investmentfonds.md, Abschnitt 4](investmentfonds.md#4-teilfreistellung).
GewSt 1 A Zeile 48 nimmt die gesamten Teilfreistellungen nach §§ 20 und 21
InvStG aus unmittelbaren Beteiligungen auf; die Korrektur nach § 20 Abs. 5
rechnet das Finanzamt ([ELSTER-Anleitung GewSt 2025][elstergewst]). Buchfink
führt Zeile 48 mit dem vollen Betrag und rechnet die Hälfte nur in den
Gewerbeertrag.

## 7. Angaben für die Erklärung

- `erklaerung` (`art: gewst`) führt § 8 Nr. 5 und § 9 Nr. 2a als eigene
  Zeilen; den Vorschlag zu § 8 Nr. 5 rechnet Buchfink aus Quote und
  Erwerbsdatum der Kartei. Bestätige ihn nach Rückfrage, wenn die Quote zu
  Beginn des Erhebungszeitraums laut Gesellschafterliste stimmt.
- Belegt ist nur Zeile 48. Für § 8 Nr. 5 und § 9 Nr. 2a, 7 folgt der Mensch
  der Ausfüllhilfe in Mein ELSTER.

[elstergewst]: https://www.elster.de/eportal/helpGlobal?themaGlobal=help_gewst_ufa_20_2025
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
[gewstg9]: https://www.gesetze-im-internet.de/gewstg/__9.html
[invstg20]: https://www.gesetze-im-internet.de/invstg_2018/__20.html
