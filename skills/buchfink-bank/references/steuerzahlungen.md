# Zahlungen an und von Finanzamt und Gemeinde

Rechtsstand: 17. Oktober 2026. Wortlaut von AO, UStG, UStDV, EStG und KStG am
17.10.2026 gelesen. Konten aus dem SKR04 und der Buchungslogik von Buchfink.

Wie hoch eine Steuer ist, bestimmen buchfink-monatsabschluss (Umsatzsteuer) und
buchfink-steuererklaerungen (KSt, GewSt). Hier wird die Zahlung gebucht.

## Inhalt

1. Erkennen und belegen
2. Konten je Steuerart
3. Nachzahlung und Erstattung für Vorjahre
4. Säumniszuschläge, Zinsen, Verspätungszuschläge
5. Fälligkeit und Zahlungstag
6. Verrechnung durch das Finanzamt

## 1. Erkennen und belegen

- Zahlungen an das Finanzamt tragen Steuernummer, Steuerart und Zeitraum im
  Verwendungszweck. Gewerbesteuer zahlt die GmbH an die Gemeinde.
- Beleg ist die Anmeldung (Voranmeldung) oder der Bescheid. Lege einen Bescheid
  mit `beleg_ablegen` ab, bevor du die Zahlung buchst; den Abgleich mit der
  Erklärung macht buchfink-steuererklaerungen.
- Ein Umsatz mit Steuer und Nebenleistung (Zinsen, Säumniszuschlag) bekommt je
  Bestandteil eine Zeile. Die Aufteilung steht im Bescheid oder in der
  Kontoübersicht des Finanzamts; frag danach, wenn sie fehlt.

## 2. Konten je Steuerart

| Zahlung | Konto | Hinweis |
|---|---|---|
| Umsatzsteuer-Vorauszahlung laut Voranmeldung, auch Erstattung | 3820 Umsatzsteuer-Vorauszahlungen | Zahlung im Soll, Erstattung im Haben |
| Sondervorauszahlung bei Dauerfristverlängerung | 3830 Umsatzsteuer-Vorauszahlungen 1/11 | Pflicht und Höhe: buchfink-monatsabschluss |
| Umsatzsteuer-Abschlusszahlung für das Vorjahr | 3841 Umsatzsteuerverbindlichkeiten Vorjahr | Buchfink bucht die Jahreszahllast dorthin |
| Umsatzsteuer-Erstattung für das Vorjahr | 1420 | Buchfink bucht die Jahreserstattung dorthin |
| Körperschaftsteuer-Vorauszahlung laufendes Jahr | 7600 Körperschaftsteuer | nicht abziehbar ([§ 10 Nr. 2 KStG][kstg10]) |
| Solidaritätszuschlag-Vorauszahlung | 7608 Solidaritätszuschlag | nicht abziehbar |
| Gewerbesteuer-Vorauszahlung | 7610 Gewerbesteuer | keine Betriebsausgabe ([§ 4 Abs. 5b EStG][estg4]) |
| Lohnsteuer, Sozialversicherung | Konten aus buchfink-lohn | – |
| Kapitalertragsteuer auf eine eigene Ausschüttung | buchfink-gesellschafter | – |

Lege jede Steuerzahlung als Rückfrage vor (`bankumsatz_buchen` mit einer
Zeile `konto` je Bestandteil). Direkt zuordnen lässt sich nur eine Zahlung auf
einen offenen Posten mit `bankumsatz_zuordnen`.

## 3. Nachzahlung und Erstattung für Vorjahre

Für das Vorjahr ist meist eine Steuerrückstellung gebucht
(buchfink-jahresabschluss). Die Zahlung verbraucht sie; nur der Unterschied
wirkt im laufenden Jahr.

| Steuer | Rückstellung | Mehr als zurückgestellt | Weniger oder Erstattung |
|---|---|---|---|
| Körperschaftsteuer | 3040 | 7603 Körperschaftsteuer für Vorjahre | 7604 Körperschaftsteuererstattungen für Vorjahre |
| Solidaritätszuschlag | 3020 (Sammelkonto) | 7609 Solidaritätszuschlag für Vorjahre | 7607 Solidaritätszuschlagerstattungen für Vorjahre |
| Gewerbesteuer | 3035 | 7641 Gewerbesteuernachzahlungen und -erstattungen für Vorjahre | 7641; Auflösung einer zu hohen Rückstellung 7643 |

Welche Rückstellung besteht, zeigt `auswertung` mit `art: kontoblatt` und
`konto: 3040` (bzw. 3035, 3020). Den Vorschlag legst du vor; ob die Rückstellung damit erledigt
ist, entscheidet der Mensch.

## 4. Säumniszuschläge, Zinsen, Verspätungszuschläge

Alle drei sind steuerliche Nebenleistungen ([§ 3 Abs. 4 AO][ao3]).

| Nebenleistung | Höhe | Fundstelle |
|---|---|---|
| Säumniszuschlag | 1 % des auf 50 € abgerundeten Rückstands je angefangenem Monat | [§ 240 Abs. 1 Satz 1 AO][ao240] |
| Zinsen auf Nachzahlung und Erstattung (§ 233a AO) | 0,15 % je vollem Monat; Zinslauf beginnt 15 Monate nach Ablauf des Entstehungsjahres | [§ 238 Abs. 1a][ao238], [§ 233a Abs. 2 Satz 1 AO][ao233a] |
| Verspätungszuschlag | nach Bescheid | [§ 152 AO][ao152]; Regeln in buchfink-steuererklaerungen |

Stand: Oktober 2026.

Abziehbarkeit und Konto:

| Nebenleistung auf | Abziehbar? | Konto |
|---|---|---|
| Körperschaftsteuer, Solidaritätszuschlag | nein ([§ 10 Nr. 2 KStG][kstg10], letzter Halbsatz) | Säumniszuschlag 7304; Zinsen § 233a 7308; Verspätungszuschlag 6437 |
| Gewerbesteuer | nein ([§ 4 Abs. 5b EStG][estg4]) | Säumniszuschlag 7302; Zinsen § 233a 7308; Verspätungszuschlag 6437 |
| andere Steuern, etwa Umsatzsteuer | keiner der beiden Ausschlüsse greift | Säumniszuschlag 7303; Zinsen § 233a 7305; Verspätungszuschlag 6436 |

Erstattungszinsen nach § 233a AO bucht das SKR04 je nach steuerlicher
Behandlung auf 7105, 7106 oder 7107. Welches Konto passt, ist eine Wertungsfrage;
frag den Menschen und verweise auf buchfink-steuererklaerungen.

## 5. Fälligkeit und Zahlungstag

- Umsatzsteuer-Vorauszahlung: fällig am zehnten Tag nach Ablauf des
  Voranmeldungszeitraums ([§ 18 Abs. 1 Satz 4 UStG][ustg18]); mit
  Dauerfristverlängerung einen Monat später ([§ 46 UStDV][ustdv46]). Stand:
  Oktober 2026. Die Termine aller Steuerzahlungen des Mandanten nennt
  `fristen`.
- An der Buchung auf dem Konto der Steuerart (Abschnitt 2) erkennt `fristen`
  Sondervorauszahlung und Vorauszahlungen als bezahlt. Ordne jede Zahlung
  deshalb dem richtigen Konto und Jahr zu.
- Als entrichtet gilt eine Überweisung am Tag der Gutschrift beim Finanzamt,
  eine Zahlung per Lastschriftmandat am Fälligkeitstag
  ([§ 224 Abs. 2 Nr. 2, 3 AO][ao224]).
- Bis zu drei Tage Säumnis kosten keinen Zuschlag; das gilt nicht für Bar- und
  Scheckzahlung ([§ 240 Abs. 3 AO][ao240]).
- Fällt dir bei der Zuordnung auf, dass eine Steuer verspätet gezahlt wurde,
  sag es dem Menschen; ein Säumniszuschlag folgt meist mit späterer Post.

## 6. Verrechnung durch das Finanzamt

Verrechnet das Finanzamt eine Erstattung mit einer anderen Steuerschuld
(Umbuchungsmitteilung), fließt kein Geld. Lege die Mitteilung ab und buche die
Umbuchung zwischen den Steuerkonten mit `buchung_erfassen` (Rückfrage, Beleg ist
die Mitteilung). Danach stimmt der Bankumsatz der Restzahlung wieder mit einem
einzelnen Konto überein.

[ao3]: https://www.gesetze-im-internet.de/ao_1977/__3.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao224]: https://www.gesetze-im-internet.de/ao_1977/__224.html
[ao233a]: https://www.gesetze-im-internet.de/ao_1977/__233a.html
[ao238]: https://www.gesetze-im-internet.de/ao_1977/__238.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustdv46]: https://www.gesetze-im-internet.de/ustdv_1980/__46.html
