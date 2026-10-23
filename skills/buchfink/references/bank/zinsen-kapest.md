# Zinsen, Kapitalertragsteuer, Solidaritätszuschlag

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG und SolZG am 17.10.2026
gelesen. Sätze mit Stand Oktober 2026; prüfe sie für andere Jahre am Gesetz.

Dividenden und Fonds: [beteiligungen/dividenden.md](../beteiligungen/dividenden.md).

## Inhalt

1. Zinserträge der GmbH
2. Kapitalertragsteuer und Solidaritätszuschlag
3. Buchung
4. Anrechnung und Steuerbescheinigung
5. Sonderfälle

## 1. Zinserträge der GmbH

- Die Bank behält bei der GmbH die volle Kapitalertragsteuer ein: Ihre
  Einkünfte sind gewerblich ([§ 8 Abs. 2 KStG][kstg8]), ohne
  Sparer-Pauschbetrag und Freistellungsauftrag ([§ 20 Abs. 8, 9 EStG][estg20],
  [§ 44a Abs. 1, 2 EStG][estg44a]).
- Zinsen sind Ertrag des Zeitraums, auf den sie entfallen; Zinsen für das alte
  Jahr, die erst im neuen gutgeschrieben werden, grenzt
  [jahresabschluss/abgrenzung.md](../jahresabschluss/abgrenzung.md) ab.

## 2. Kapitalertragsteuer und Solidaritätszuschlag

| Was | Regel | Fundstelle |
|---|---|---|
| Kapitalertragsteuer | 25 % des Kapitalertrags, bemessen ohne jeden Abzug | [§ 43a Abs. 1 Satz 1 Nr. 1, Abs. 2 Satz 1 EStG][estg43a] |
| Solidaritätszuschlag | 5,5 % der Kapitalertragsteuer; Bruchteile eines Cents bleiben außer Ansatz | [§ 3 Abs. 1 Nr. 5][solzg3], [§ 4 Satz 1, 3 SolZG][solzg4] |
| Abzug auf Zinsen | wenn der Schuldner ein inländisches Kreditinstitut ist | [§ 43 Abs. 1 Satz 1 Nr. 7 Buchst. b EStG][estg43] |

Sätze und Bemessung gelten auch für Dividenden
([beteiligungen/dividenden.md](../beteiligungen/dividenden.md#2-steuerabzug))
und die eigene Ausschüttung
([gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md#3-steuerabzug)).
Bei der GmbH mindert die Kapitalertragsteuer als Steuer vom Einkommen das
Einkommen nicht ([§ 43 Abs. 1 Satz 1 EStG][estg43], [§ 10 Nr. 2 KStG][kstg10]).

Beispiel: Zinsen 100,00 €; Kapitalertragsteuer 25,00 €; Solidaritätszuschlag
1,375 €, also 1,37 €; Gutschrift 73,63 €.

## 3. Buchung

| Zeile | Konto (SKR04) | Beispiel |
|---|---|---|
| Zinsertrag brutto | 7110 Sonstige Zinserträge | 100,00 |
| Kapitalertragsteuer | 7630 Kapitalertragsteuer 25 % | −25,00 |
| Solidaritätszuschlag | 7633 Anrechenbarer Solidaritätszuschlag auf Kapitalertragsteuer 25 % | −1,37 |

`bankumsatz_buchen` mit diesen drei Zeilen `konto`. Ob der Zinsertrag als nicht
steuerbar oder steuerfrei geführt wird, wirkt nur auf die Voranmeldung; klär es
beim ersten Mal mit dem Menschen (Wertungsfrage) und bleib dabei. Eine eigene
Zinsabrechnung der Bank legst du ab; sonst ist der Kontoauszug der Beleg.

## 4. Anrechnung und Steuerbescheinigung

Auf die Körperschaftsteuer angerechnet wird nur mit Steuerbescheinigung
([steuererklaerungen/kst.md, Abschnitt 9](../steuererklaerungen/kst.md#9-steuer-rundung-anrechnung-vorauszahlungen),
dort auch der Abgleich von 7630 und 7633). Die Bank stellt sie auf Verlangen
aus ([§ 45a Abs. 2 EStG][estg45a]); erinnere den Menschen, sie anzufordern und
abzulegen.

## 5. Sonderfälle

- **Ausländische Bank:** kein deutscher Abzug (Abschnitt 2, Nr. 7 Buchst. b).
  Ausländische Quellensteuer legst du vor (Anrechnung,
  Doppelbesteuerungsabkommen: Steuerberater).
- **Zinsen auf Steuererstattungen:** 7105 bis 7107;
  [steuerzahlungen.md](steuerzahlungen.md#4-säumniszuschläge-zinsen-verspätungszuschläge).
- **Zinsen von Gesellschafter oder Tochter:**
  [darlehen.md](darlehen.md#10-zinsen-an-und-von-gesellschaftern),
  [beteiligungen/erwerb-ausweis.md](../beteiligungen/erwerb-ausweis.md).

[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
