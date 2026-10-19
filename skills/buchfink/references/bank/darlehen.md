# Darlehen: Bank und Gesellschafter

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, EStG, KStG, GewStG und UStG am
17.10.2026 gelesen. Werkzeugstand: Buchfink 0.3.0.

Darlehen an Tochtergesellschaften regelt
[beteiligungen/erwerb-ausweis.md](../beteiligungen/erwerb-ausweis.md). Wie ein
Gesellschafterdarlehen im Arbeitsablauf erfasst wird, steht in
[gesellschafter/ablauf.md](../gesellschafter/ablauf.md#4-darlehen).

## Inhalt

1. Vertrag und Belege
2. Ansatz und Ausweis
3. Zinsen: Umsatzsteuer, Gewerbesteuer, Stichtag
4. Bankdarlehen: Konten
5. Bankdarlehen: Auszahlung und Rate
6. Disagio und Bearbeitungsentgelt
7. Kontokorrent
8. Gesellschafter leiht der GmbH
9. GmbH leiht dem Gesellschafter
10. Zinsen an und von Gesellschaftern
11. Insolvenz, Nachrang, Rangrücktritt
12. Buchungen eines Gesellschafterdarlehens

## 1. Vertrag und Belege

- Ein Darlehen verpflichtet zur Auszahlung, zur Zinszahlung und zur
  Rückzahlung bei Fälligkeit; ohne bestimmte Laufzeit gilt eine Kündigungsfrist
  von drei Monaten ([§ 488 BGB][bgb488]).
- Lege den Darlehensvertrag (bei einem Bankdarlehen mit Tilgungsplan) mit
  `beleg_ablegen` ab, bevor die Auszahlung gebucht wird. Der Vertrag ist
  Ursprungsbeleg für alle Folgebuchungen ([GoBD Rz. 81][gobd]). Die jährliche
  Zins- und Saldenbestätigung der Bank legst du ebenfalls ab.
- Mit einem beherrschenden Gesellschafter braucht es eine klare, im Voraus
  geschlossene und tatsächlich durchgeführte Vereinbarung, sonst droht eine
  verdeckte Gewinnausschüttung
  ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md#1-was-eine-vga-ist)).
  Ist der Gesellschafter alleiniger Geschäftsführer einer Ein-Personen-GmbH,
  gelten Befreiung von § 181 BGB und Niederschrift
  ([gesellschafter/ablauf.md](../gesellschafter/ablauf.md#3-grundregeln)).
- Fehlt der Vertrag mit einem Gesellschafter, empfiehl dem Menschen, ihn
  schriftlich nachzuholen; die Wirkung „im Voraus“ heilt ein späterer Vertrag
  nicht. Wertungsfrage, Steuerberater.

## 2. Ansatz und Ausweis

- Darlehen werden mit dem Erfüllungsbetrag angesetzt
  ([§ 253 Abs. 1 Satz 2 HGB][hgb253]), also mit dem Betrag, der zurückzuzahlen
  ist.
- Je Darlehen ein Konto der passenden Restlaufzeit (Abschnitte 4, 8 und 9).
  Den Restlaufzeitvermerk in der Bilanz und die Umgliederung zum Stichtag regelt
  [jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## 3. Zinsen: Umsatzsteuer, Gewerbesteuer, Stichtag

- Zinsen für einen Kredit sind umsatzsteuerfrei
  ([§ 4 Nr. 8 Buchst. a UStG][ustg4], Steuerfall `exempt`); die Zinszeile
  trägt keine Vorsteuer. Zinsen an einen privaten Gesellschafter tragen keine
  Umsatzsteuer. Auszahlung und Tilgung sind kein Umsatz (`not_taxable` bei
  `buchung_erfassen`).
- **Gewerbesteuer:** Ein Viertel der Summe aus Entgelten für Schulden und
  weiteren Finanzierungsanteilen wird hinzugerechnet, soweit die Summe
  200.000 € übersteigt ([§ 8 Nr. 1 GewStG][gewstg8]). Stand: Oktober 2026.
  Rechnung in [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md).
  Buche Zinsen deshalb getrennt von Gebühren.
- **Zinsschranke:** Sie greift erst, wenn die Nettozinsaufwendungen 3 Mio. €
  erreichen ([§ 4h Abs. 2 Satz 1 Buchst. a EStG][estg4h],
  [§ 8a KStG][kstg8a]). Stand: Oktober 2026. Für die Zielgruppe von Buchfink
  selten; erreicht der Zinsaufwand diese Größenordnung, empfiehl einen
  Steuerberater.
- **Zinsen zum Stichtag:** Zinsen, die wirtschaftlich in das alte Jahr gehören,
  aber erst im neuen belastet werden, grenzt
  [jahresabschluss/abgrenzung.md](../jahresabschluss/abgrenzung.md) ab.

## 4. Bankdarlehen: Konten

Aus dem SKR04:

| Konto | Inhalt |
|---|---|
| 3150 | Verbindlichkeiten gegenüber Kreditinstituten |
| 3151 | dto. Restlaufzeit bis 1 Jahr |
| 3160 | dto. Restlaufzeit 1 bis 5 Jahre |
| 3170 | dto. Restlaufzeit größer 5 Jahre |
| 7310 | Zinsaufwendungen für kurzfristige Verbindlichkeiten |
| 7318 | Zinsen auf Kontokorrentkonten |
| 7320 | Zinsaufwendungen für langfristige Verbindlichkeiten |
| 1940 | Damnum/Disagio |
| 7323 | Abschreibungen auf ein Agio oder Disagio/Damnum zur Finanzierung |

Hat die GmbH mehrere Bankdarlehen, frag den Menschen, wie sie getrennt werden
sollen (Text, Konto).

## 5. Bankdarlehen: Auszahlung und Rate

- **Auszahlung:** Bankumsatz ist der Eingang des Auszahlungsbetrags. Werkzeug:
  `bankumsatz_buchen` mit einer Zeile `konto` (Darlehenskonto) über den
  Nennbetrag. Behält die Bank Disagio oder Entgelte ein, kommt je Abzug eine
  weitere Zeile mit negativem `betrag` dazu (Abschnitt 6).
- **Rate:** Teile jede Rate nach Tilgungsplan oder Kontoauszug in Tilgung
  (Darlehenskonto) und Zinsen (7320 bzw. 7310). Die Tilgung ist kein Aufwand.
- Fehlt die Aufteilung, frag nach dem Tilgungsplan. Rechne sie nicht selbst
  aus, außer der Mensch bittet darum und bestätigt das Ergebnis.
- Raten sind Rückfrage (`bankumsatz_buchen` mit je einer Zeile für Tilgung
  und Zinsen). Lege mehrere Raten gesammelt vor.

## 6. Disagio und Bearbeitungsentgelt

- Ist der Rückzahlungsbetrag höher als der Auszahlungsbetrag, darf der
  Unterschied als aktiver Rechnungsabgrenzungsposten angesetzt und über die
  Laufzeit planmäßig abgeschrieben werden ([§ 250 Abs. 3 HGB][hgb250]).
- Buchfink führt eine Einheitsbilanz
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
  Vorschlag: aktivieren (1940) und über die Laufzeit bzw. Zinsbindung auf 7323
  auflösen. Ob die Steuerbilanz das verlangt, bestätigt der Steuerberater; die
  Frage ist hier nicht am Wortlaut geprüft. Wertungsfrage, Rückfrage.
- Bearbeitungsentgelte der Bank: Frag, ob sie laufzeitabhängig sind. Die
  Einordnung (Aufwand oder Abgrenzung) ist eine Wertungsfrage.
- Die jährliche Auflösung ist eine Abschlussbuchung (`abschlussbuchung` mit
  `art: abgrenzung` und `abgrenzung.art: disagio`;
  [jahresabschluss/abgrenzung.md](../jahresabschluss/abgrenzung.md)).

## 7. Kontokorrent

- Ein Bankkonto mit Sollsaldo (Dispositionskredit) bleibt auf seinem
  Bankkonto. Zinsen dafür buchst du auf 7318.
- Ein dauerhaft negativer Saldo ist ein Warnzeichen für die Liquidität. Melde
  ihn dem Menschen
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest)).

## 8. Gesellschafter leiht der GmbH

- Konto nach vereinbarter Restlaufzeit: 3641 bis 1 Jahr, 3642 1 bis 5 Jahre,
  3643 über 5 Jahre (Verbindlichkeiten gegenüber GmbH-Gesellschaftern).
- Zins: Wertungsfrage. Ein Zins über dem, was ein fremder Darlehensgeber
  erhielte, ist im übersteigenden Teil vGA-verdächtig
  ([§ 8 Abs. 3 Satz 2 KStG][kstg8]). Ein niedriger Zins oder ein zinsloses
  Darlehen belastet die GmbH nicht.
- Die Rückzahlung eines Gesellschafterdarlehens fällt nicht unter das
  Auszahlungsverbot ([§ 30 Abs. 1 Satz 3 GmbHG][gmbhg30]). Insolvenzrisiken in
  Abschnitt 11.

## 9. GmbH leiht dem Gesellschafter

- Zulässig nur, wenn der Rückzahlungsanspruch vollwertig ist
  ([§ 30 Abs. 1 Satz 2 GmbHG][gmbhg30]). Wertungsfrage: Bonität des
  Gesellschafters, Sicherheit. Geschäftsführer, Prokuristen und
  Generalbevollmächtigte dürfen keinen Kredit aus dem Vermögen erhalten, das das
  Stammkapital deckt ([§ 43a GmbHG][gmbhg43a]).
- Zinslos oder zu niedrig verzinst: verhinderte Vermögensmehrung, vGA-Verdacht
  ([R 8.5 Abs. 1 KStR 2022][kstr85]).
- Konto: Soll das Darlehen dauernd dem Geschäftsbetrieb dienen, gehört es ins
  Anlagevermögen ([§ 247 Abs. 2 HGB][hgb247]): 0960 Ausleihungen an
  Gesellschafter. Sonst 1308 (Restlaufzeit bis 1 Jahr) oder 1309 (über 1 Jahr).
  Die Einordnung ist eine Wertungsfrage.

## 10. Zinsen an und von Gesellschaftern

- Zinsaufwand der GmbH: 7316 Zinsen für Gesellschafterdarlehen; bei einem
  Gesellschafter mit mehr als 25 % Beteiligung führt der SKR04 das Konto 7317
  (Kontenbezeichnung). Frag nach der Beteiligungsquote.
- Zinsen an die GmbH: Ertrag 7110 Sonstige Zinserträge.
- Kapitalertragsteuer: Auf Zinsen eines gewöhnlichen Darlehens behält die GmbH
  keine Kapitalertragsteuer ein; § 43 Abs. 1 Satz 1 Nr. 7 EStG erfasst nur
  Zinsen aus registrierten Anleihen und Zinsen, die Kreditinstitute schulden
  ([§ 43 Abs. 1 Satz 1 Nr. 7 EStG][estg43]). Gewinnabhängige Zinsen
  (partiarisches Darlehen) oder eine stille Beteiligung unterliegen dem Abzug
  ([§ 43 Abs. 1 Satz 1 Nr. 3, § 20 Abs. 1 Nr. 4 EStG][estg43]): Grenze,
  Steuerberater.
- Hinweis für den Gesellschafter: Zinsen an einen zu mindestens 10 %
  beteiligten Gesellschafter fallen nicht unter den gesonderten Steuertarif
  für Kapitalerträge ([§ 32d Abs. 2 Nr. 1 Buchst. b EStG][estg32d]). Seine
  Einkommensteuer bildet Buchfink nicht ab.

## 11. Insolvenz, Nachrang, Rangrücktritt

- Forderungen auf Rückgewähr eines Gesellschafterdarlehens sind im
  Insolvenzverfahren nachrangig ([§ 39 Abs. 1 Satz 1 Nr. 5 InsO][inso39]).
  Ausgenommen ist ein nicht geschäftsführender Gesellschafter mit höchstens
  10 % am Haftkapital ([§ 39 Abs. 5 InsO][inso39]).
- Anfechtbar sind eine Rückzahlung im letzten Jahr vor dem Insolvenzantrag und
  eine Sicherung in den letzten zehn Jahren
  ([§ 135 Abs. 1 InsO][inso135]). Weise vor jeder Tilgung auf diese Frist hin,
  wenn die Zahlen auf eine Krise deuten
  ([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest)).
- Ein vereinbarter Rangrücktritt hinter alle Forderungen nach § 39 Abs. 1
  Nr. 1 bis 5 InsO nimmt das Darlehen aus der Überschuldungsprüfung
  ([§ 19 Abs. 2 Satz 2 InsO][inso19]).
- Steuerlich: Ist die Verbindlichkeit nur aus künftigen Gewinnen oder Einnahmen
  zu erfüllen, darf sie in der Steuerbilanz erst angesetzt werden, wenn diese
  anfallen ([§ 5 Abs. 2a EStG][estg5]); die Ausbuchung kann einen Gewinn
  auslösen. Die Formulierung eines Rangrücktritts gehört zu Anwalt und
  Steuerberater. Buchfink führt eine Einheitsbilanz und bildet abweichende
  Steuerbilanzwerte hier nicht ab.

## 12. Buchungen eines Gesellschafterdarlehens

| Vorgang | Soll | Haben | Werkzeug |
|---|---|---|---|
| Gesellschafter zahlt Darlehen aus | Bank | 3641 bis 3643 | `bankumsatz_buchen`, Zeile `darlehen_id` |
| Zinsen fällig, noch nicht gezahlt | 7316 oder 7317 | 3640 | `darlehen_erfassen` mit `art: zinsen`, Eigenbeleg mit der Zinsrechnung |
| Zinszahlung | 3640 | Bank | `bankumsatz_buchen`, Zeile `konto` |
| Tilgung | 3641 bis 3643 | Bank | `bankumsatz_buchen`, Zeile `darlehen_id` |
| GmbH zahlt Darlehen an Gesellschafter aus | 0960, 1308 oder 1309 | Bank | `bankumsatz_buchen`, Zeile `darlehen_id` |
| Gesellschafter tilgt | Bank | 0960, 1308 oder 1309 | `bankumsatz_buchen`, Zeile `darlehen_id` |

Das Konto des Darlehens legt `darlehen_erfassen` aus dem Vertrag fest. Alle
Darlehensbuchungen sind Rückfrage.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[bgb488]: https://www.gesetze-im-internet.de/bgb/__488.html
[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg43a]: https://www.gesetze-im-internet.de/gmbhg/__43a.html
[hgb247]: https://www.gesetze-im-internet.de/hgb/__247.html
[hgb250]: https://www.gesetze-im-internet.de/hgb/__250.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[estg4h]: https://www.gesetze-im-internet.de/estg/__4h.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg32d]: https://www.gesetze-im-internet.de/estg/__32d.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg8a]: https://www.gesetze-im-internet.de/kstg_1977/__8a.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[inso19]: https://www.gesetze-im-internet.de/inso/__19.html
[inso39]: https://www.gesetze-im-internet.de/inso/__39.html
[inso135]: https://www.gesetze-im-internet.de/inso/__135.html
[kstr85]: https://www.steuerschroeder.de/steuergesetze/kstr/8.5
