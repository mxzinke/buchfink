# Zahlungen an und von Finanzamt und Gemeinde

Rechtsstand: 17. Oktober 2026. Wortlaut von AO, EStG und KStG am 17.10.2026
gelesen. Konten aus dem SKR04 und der Buchungslogik von Buchfink.

Hier wird die Zahlung gebucht; die Höhe der Steuer bestimmen
[monatsabschluss/ustva.md](../monatsabschluss/ustva.md),
[steuererklaerungen/kst.md](../steuererklaerungen/kst.md) und
[steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md).

## Inhalt

1. Erkennen und belegen
2. Konten je Steuerart
3. Nachzahlung und Erstattung für Vorjahre
4. Säumniszuschläge, Zinsen, Verspätungszuschläge
5. Fälligkeit und Zahlungstag
6. Verrechnung durch das Finanzamt

## 1. Erkennen und belegen

- Zahlungen an das Finanzamt nennen Steuernummer, Steuerart und Zeitraum im
  Verwendungszweck; Gewerbesteuer geht an die Gemeinde.
- Beleg ist die Anmeldung oder der Bescheid. Lege einen Bescheid ab, bevor du
  die Zahlung buchst; Abgleich mit der Erklärung:
  [steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md).
- Steuer und Nebenleistung (Zinsen, Säumniszuschlag) in einem Umsatz bekommen
  je eine Zeile. Die Aufteilung steht im Bescheid oder in der Kontoübersicht
  des Finanzamts; fehlt sie, frag.

## 2. Konten je Steuerart

| Zahlung | Konto | Hinweis |
|---|---|---|
| Umsatzsteuer-Vorauszahlung laut Voranmeldung, auch Erstattung | 3820 | Zahlung im Soll, Erstattung im Haben |
| Sondervorauszahlung bei Dauerfristverlängerung | 3830 | [monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md) |
| Letzte Voranmeldung des Vorjahres nach gebuchter Umsatzsteuer-Verrechnung; Abschlusszahlung für das Vorjahr | 3841 | dorthin bucht Buchfink Verrechnung und Jahreszahllast ([jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md#1-umsatzsteuer-verrechnung)) |
| Umsatzsteuer-Erstattung für das Vorjahr | 1420 | dorthin bucht Buchfink die Jahreserstattung |
| Körperschaftsteuer-, Solidaritätszuschlag-Vorauszahlung laufendes Jahr | 7600, 7608 | nicht abziehbar ([§ 10 Nr. 2 KStG][kstg10]) |
| Gewerbesteuer-Vorauszahlung | 7610 | keine Betriebsausgabe ([§ 4 Abs. 5b EStG][estg4]) |
| Lohnsteuer, Sozialversicherung | Konten aus [lohn/lohnjournal.md](../lohn/lohnjournal.md) | – |
| Kapitalertragsteuer auf eine eigene Ausschüttung | 3760 | [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md#5-buchungen-in-buchfink) |

Jede Steuerzahlung ist Rückfrage: `bankumsatz_buchen` mit einer Zeile `konto`
je Bestandteil.

## 3. Nachzahlung und Erstattung für Vorjahre

Die Zahlung verbraucht die Steuerrückstellung des Vorjahres
([jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md));
nur der Unterschied wirkt im laufenden Jahr.

| Steuer | Rückstellung | Mehr als zurückgestellt | Weniger oder Erstattung |
|---|---|---|---|
| Körperschaftsteuer | 3040 | 7603 | 7604 |
| Solidaritätszuschlag | 3040 | 7609 | 7607 |
| Gewerbesteuer | 3035 | 7641 | 7641; Auflösung einer zu hohen Rückstellung 7643 |

Die Rückstellung zeigt `auswertung` (`art: kontoblatt`, `konto: 3040` bzw.
3035). Ob sie damit erledigt ist, entscheidet der Mensch.

## 4. Säumniszuschläge, Zinsen, Verspätungszuschläge

Steuerliche Nebenleistungen ([§ 3 Abs. 4 AO][ao3]). Höhe: Säumniszuschlag
([§ 240 AO][ao240]) in
[grundlagen/pflichten.md](../grundlagen/pflichten.md#fristende-zahlung-verlängerung);
Zinsen ([§ 233a AO][ao233a]) in
[steuererklaerungen/fristen-zuschlaege.md, Abschnitt 5](../steuererklaerungen/fristen-zuschlaege.md#5-zinsen-und-säumniszuschläge);
Verspätungszuschlag ([§ 152 AO][ao152]) nach Bescheid,
[Abschnitt 4](../steuererklaerungen/fristen-zuschlaege.md#4-verspätungszuschlag).

| Nebenleistung auf | Abziehbar? | Konto |
|---|---|---|
| Körperschaftsteuer, Solidaritätszuschlag | nein ([§ 10 Nr. 2 KStG][kstg10], letzter Halbsatz) | Säumniszuschlag 7304; Zinsen § 233a 7308; Verspätungszuschlag 6437 |
| Gewerbesteuer | nein ([§ 4 Abs. 5b EStG][estg4]) | Säumniszuschlag 7302; Zinsen § 233a 7308; Verspätungszuschlag 6437 |
| andere Steuern, etwa Umsatzsteuer | ja, kein Ausschluss greift | Säumniszuschlag 7303; Zinsen § 233a 7305; Verspätungszuschlag 6436 |

Erstattungszinsen nach § 233a AO: 7105, 7106 oder 7107 je nach steuerlicher
Behandlung; die ist eine Wertungsfrage
([steuererklaerungen/kst.md, Abschnitt 2](../steuererklaerungen/kst.md#2-überleitung-zum-zu-versteuernden-einkommen)).

## 5. Fälligkeit und Zahlungstag

- Termine nennt `fristen`; wann eine Zahlung als entrichtet gilt:
  [Fristende, Zahlung, Verlängerung](../grundlagen/pflichten.md#fristende-zahlung-verlängerung).
- `fristen` erkennt Sondervorauszahlung und Vorauszahlungen an der Buchung auf
  dem Konto der Steuerart (Abschnitt 2) als bezahlt; ordne jede Zahlung dem
  richtigen Konto und Jahr zu.
- Fällt dir eine verspätete Zahlung auf, sag es dem Menschen; ein
  Säumniszuschlag folgt meist später.

## 6. Verrechnung durch das Finanzamt

Verrechnet das Finanzamt eine Erstattung mit einer anderen Steuerschuld
(Umbuchungsmitteilung), fließt kein Geld: Mitteilung ablegen, Umbuchung
zwischen den Steuerkonten mit `buchung_erfassen` (Rückfrage, Beleg ist die
Mitteilung). Danach passt die Restzahlung wieder zu einem Konto.

[ao3]: https://www.gesetze-im-internet.de/ao_1977/__3.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao233a]: https://www.gesetze-im-internet.de/ao_1977/__233a.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
