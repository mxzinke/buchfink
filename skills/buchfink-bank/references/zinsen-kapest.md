# Zinsen, Kapitalertragsteuer, Solidaritätszuschlag

Rechtsstand: 17. Oktober 2026. Wortlaut von EStG, KStG und SolZG am 17.10.2026
gelesen. Sätze mit Stand Oktober 2026; prüfe sie für andere Jahre am Gesetz.

Dividenden, Fonds und Wertpapiere regelt buchfink-beteiligungen. Zinsen des
Finanzamts und Darlehenszinsen haben eigene Referenzen (Liste in SKILL.md).

## Inhalt

1. Zinserträge der GmbH
2. Kapitalertragsteuer und Solidaritätszuschlag
3. Buchung
4. Anrechnung und Steuerbescheinigung
5. Sonderfälle

## 1. Zinserträge der GmbH

- Alle Einkünfte einer GmbH sind Einkünfte aus Gewerbebetrieb
  ([§ 8 Abs. 2 KStG][kstg8]); Kapitalerträge werden diesen zugerechnet
  ([§ 20 Abs. 8 Satz 1 EStG][estg20]).
- Den Sparer-Pauschbetrag gibt es nur bei Einkünften aus Kapitalvermögen
  ([§ 20 Abs. 9 EStG][estg20]). Ein Freistellungsauftrag setzt ihn voraus
  ([§ 44a Abs. 1, 2 EStG][estg44a]). Die Bank behält deshalb bei der GmbH
  regelmäßig die volle Kapitalertragsteuer ein.
- Zinsen sind Ertrag des Zeitraums, auf den sie entfallen
  ([§ 252 Abs. 1 Nr. 5 HGB][hgb252]). Zinsen für das alte Jahr, die erst im
  neuen gutgeschrieben werden, grenzt buchfink-jahresabschluss ab.

## 2. Kapitalertragsteuer und Solidaritätszuschlag

| Was | Satz | Fundstelle |
|---|---|---|
| Kapitalertragsteuer auf Zinsen, wenn der Schuldner ein inländisches Kreditinstitut ist | 25 % des Ertrags | [§ 43 Abs. 1 Satz 1 Nr. 7 Buchst. b][estg43], [§ 43a Abs. 1 Satz 1 Nr. 1 EStG][estg43a] |
| Solidaritätszuschlag | 5,5 % der Kapitalertragsteuer; Bruchteile eines Cents bleiben außer Ansatz | [§ 3 Abs. 1 Nr. 5][solzg3], [§ 4 Satz 1, 3 SolZG][solzg4] |

- Kapitalertragsteuer ist Einkommensteuer, die durch Abzug erhoben wird
  ([§ 43 Abs. 1 Satz 1 EStG][estg43]). Bei der GmbH ist sie eine Steuer vom
  Einkommen und mindert das Einkommen nicht ([§ 10 Nr. 2 KStG][kstg10]).
- Beispiel: Zinsen 100,00 €; Kapitalertragsteuer 25,00 €; Solidaritätszuschlag
  1,375 €, also 1,37 €; Gutschrift 73,63 €.

## 3. Buchung

Konten aus dem SKR04:

| Zeile | Konto | Betrag im Beispiel |
|---|---|---|
| Zinsertrag brutto | 7110 Sonstige Zinserträge | 100,00 |
| Kapitalertragsteuer | 7630 Kapitalertragsteuer 25 % | 25,00 (Abzug) |
| Solidaritätszuschlag | 7633 Anrechenbarer Solidaritätszuschlag auf Kapitalertragsteuer 25 % | 1,37 (Abzug) |
| Bank | Bankkonto | 73,63 |

Werkzeug: `bankumsatz_buchen` mit drei Zeilen `konto`: Zinsertrag mit
positivem `betrag`, Kapitalertragsteuer und Solidaritätszuschlag als Abzüge
mit negativem `betrag`. Keine der Zeilen trägt Umsatzsteuer. Ob der Zinsertrag als nicht
steuerbar oder als steuerfrei geführt wird, beeinflusst nur die Meldung in der
Voranmeldung; klär das beim ersten Mal mit dem Menschen (Wertungsfrage) und
bleib dann dabei.

Steht die Abrechnung der Bank (Zinsabrechnung) als eigenes Dokument zur
Verfügung, lege sie aus dem Eingang mit `beleg_ablegen` ab; sonst ist der
Kontoauszug der Beleg.

## 4. Anrechnung und Steuerbescheinigung

- Die durch Abzug erhobene Steuer wird auf die Körperschaftsteuer angerechnet
  ([§ 31 Abs. 1 KStG][kstg31] mit [§ 36 Abs. 2 Nr. 2 EStG][estg36]); der
  Solidaritätszuschlag entsprechend ([§ 1 Abs. 2 SolZG][solzg1]).
- Ohne Steuerbescheinigung der Bank keine Anrechnung
  ([§ 36 Abs. 2 Nr. 2 Satz 2 EStG][estg36]). Die Bank stellt sie auf Verlangen
  aus ([§ 45a Abs. 2 EStG][estg45a]). Erinnere den Menschen, sie anzufordern und
  abzulegen.
- Die Summe auf 7630 und 7633 muss am Jahresende zur Bescheinigung passen.
  Abgleich und Erklärung: buchfink-steuererklaerungen.

## 5. Sonderfälle

- **Ausländische Bank:** Kapitalertragsteuer fällt nach § 43 Abs. 1 Satz 1
  Nr. 7 Buchst. b nur bei inländischen Kreditinstituten an. Zinsen einer
  ausländischen Bank kommen meist ohne deutschen Abzug; vielleicht mit
  ausländischer Quellensteuer. Leg ausländische Quellensteuer vor (Anrechnung,
  Doppelbesteuerungsabkommen: Steuerberater).
- **Zinsen auf Steuererstattungen:** SKR04 7105 bis 7107; Regel bei den
  Steuerzahlungen (SKILL.md).
- **Zinsen von Gesellschafter oder Tochter:** buchfink-gesellschafter bzw.
  buchfink-beteiligungen.
- **Gewerbesteuer:** Hinzurechnungen betreffen Zinsaufwand, nicht Zinsertrag
  ([§ 8 Nr. 1 Buchst. a GewStG][gewstg8]).

[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[solzg1]: https://www.gesetze-im-internet.de/solzg_1995/__1.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
