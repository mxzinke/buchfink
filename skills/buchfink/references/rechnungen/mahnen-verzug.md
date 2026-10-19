# Mahnen, Verzug, Verzugszinsen

Rechtsstand: 17. Oktober 2026. Basiszinssatz Stand 1. Juli 2026.

## Inhalt

1. Fälligkeit und Verzug
2. Verzugszinsen
3. Pauschale und weitere Kosten
4. Rechenweg
5. Mahnschreiben entwerfen
6. Buchen und Umsatzsteuer
7. Verjährung

## 1. Fälligkeit und Verzug

Fällig ist die Forderung zum Datum `faellig` aus `offene_posten`. Verzug tritt
ein:

| Weg | Voraussetzung | Grundlage |
|---|---|---|
| Mahnung | nach Fälligkeit gemahnt, Kunde zahlt nicht | [§ 286 Abs. 1 BGB][bgb286] |
| Kalenderdatum | Leistungszeit nach dem Kalender bestimmt oder ab einem Ereignis berechenbar | [§ 286 Abs. 2 Nr. 1, 2 BGB][bgb286] |
| 30 Tage | 30 Tage nach Fälligkeit und Zugang der Rechnung; gegenüber Verbrauchern nur mit besonderem Hinweis in der Rechnung | [§ 286 Abs. 3 Satz 1 BGB][bgb286] |
| Verweigerung | Kunde verweigert ernsthaft und endgültig | [§ 286 Abs. 2 Nr. 3 BGB][bgb286] |

- Ist der Zugang der Rechnung unsicher, kommt ein Unternehmer spätestens
  30 Tage nach Fälligkeit und Empfang der Leistung in Verzug
  ([§ 286 Abs. 3 Satz 2 BGB][bgb286]).
- Kein Verzug, solange der Kunde die Verzögerung nicht zu vertreten hat
  ([§ 286 Abs. 4 BGB][bgb286]), etwa weil die Rechnung fehlerhaft war.
- Wertungsfrage: Ob ein Zahlungsziel „nach dem Kalender bestimmt“ ist, hängt
  davon ab, ob es vereinbart wurde (Vertrag, AGB) oder nur auf der Rechnung
  steht. Im Zweifel: mahnen. Die Mahnung nach Fälligkeit begründet Verzug
  ([§ 286 Abs. 1 BGB][bgb286]).
- Den Hinweis nach § 286 Abs. 3 Satz 1 BGB druckt Buchfink auf Rechnungen an
  Verbraucher ([pflichtangaben-ausgang.md, Abschnitt 6](pflichtangaben-ausgang.md#6-hinweise-in-sonderfällen)).
  Fehlt er auf einer älteren Rechnung, beginnt Verzug erst mit der Mahnung.

## 2. Verzugszinsen

- Satz: Basiszinssatz plus 5 Prozentpunkte; bei Entgeltforderungen ohne
  Beteiligung eines Verbrauchers plus 9 Prozentpunkte
  ([§ 288 Abs. 1, 2 BGB][bgb288]).
- Der Basiszinssatz ändert sich zum 1. Januar und 1. Juli; die Bundesbank gibt
  ihn bekannt ([§ 247 BGB][bgb247]). Der Wert im Gesetzestext ist veraltet;
  maßgeblich ist die Bekanntmachung.

| gültig ab | Basiszinssatz | Verzugszins B2B (+9) | Verzugszins mit Verbraucher (+5) |
|---|---|---|---|
| 01.01.2025 | 2,27 % | 11,27 % | 7,27 % |
| 01.07.2025 | 1,27 % | 10,27 % | 6,27 % |
| 01.01.2026 | 1,27 % | 10,27 % | 6,27 % |
| 01.07.2026 | 1,52 % | 10,52 % | 6,52 % |

Quelle: [Deutsche Bundesbank, Basiszinssatz nach § 247 BGB][bbk], abgerufen am
17.10.2026. Ab 01.01.2027 neu nachsehen.

- Höhere Zinsen aus Vertrag oder anderem Grund bleiben möglich
  ([§ 288 Abs. 3 BGB][bgb288]). Ein im Voraus vereinbarter Ausschluss der
  Verzugszinsen ist gegenüber Unternehmern unwirksam
  ([§ 288 Abs. 6 BGB][bgb288]).

## 3. Pauschale und weitere Kosten

- Ist der Kunde kein Verbraucher, kann die GmbH bei Verzug zusätzlich 40 €
  Pauschale verlangen, auch bei Abschlagszahlungen
  ([§ 288 Abs. 5 Satz 1, 2 BGB][bgb288]). Je Forderung einmal; sie wird auf
  Rechtsverfolgungskosten angerechnet ([§ 288 Abs. 5 Satz 3 BGB][bgb288]).
- Weitere Schäden (z. B. Inkasso, Anwalt) sind möglich
  ([§ 288 Abs. 4 BGB][bgb288]); ob und in welcher Höhe, klärt der Mensch mit
  einem Anwalt.
- Mahngebühren ohne vertragliche Grundlage: Wertungsfrage; schlag sie nur vor,
  wenn der Mensch sie in AGB oder Vertrag vereinbart hat.

## 4. Rechenweg

`offene_posten` liefert je Posten `verzug_ab`, `verzugszinsen` bis zum
`stichtag`, `pauschale` und den Mahnstand. Übernimm diese Werte und nenne dem
Menschen die Grundlage (Verzugsbeginn, Zinssatz, Zeitraum). Selbst rechnest du
nur, wenn ein Wert fehlt, etwa weil der Verzug aus einer Abrede außerhalb von
Buchfink folgt, oder wenn der Kunde ihn bestreitet. Dann so:

1. Verzugsbeginn nach Abschnitt 1 bestimmen. Zinsen laufen ab dem Folgetag
   ([§ 187 Abs. 1 BGB][bgb187] entsprechend).
2. Zeitraum bis zum Stichtag in Abschnitte je Basiszinssatz teilen
   (Wechsel am 1. Januar und 1. Juli).
3. Je Abschnitt: offener Betrag × Zinssatz × Tage ÷ 365, auf Cent runden.
4. Summe der Abschnitte ist der Zins. Teilzahlungen im Verzug: ab dem
   Zahlungstag mit dem geringeren Betrag weiterrechnen.

Beispiel: 1.190,00 € offen, B2B, 30 Zinstage im zweiten Halbjahr 2026 zu
10,52 %: 1.190,00 × 0,1052 × 30 ÷ 365 = 10,29 €. Dazu 40 € Pauschale.

Der Tagesteiler 365 ist eine Rechenkonvention, die Buchfink auch intern nutzt.
Wertungsfrage, wenn der Kunde die Methode bestreitet.

## 5. Mahnschreiben entwerfen

Ein Mahnschreiben enthält:

- Rechnungsnummer, Rechnungsdatum, Betrag, Fälligkeit, offener Betrag.
- Bitte um Zahlung mit konkretem Datum.
- Bei Verzug: Zinsen mit Satz und Zeitraum, Pauschale, jeweils mit Fundstelle.
- Bankverbindung aus den Unternehmensstammdaten.

Stufen: freundliche Erinnerung, Mahnung mit Frist, letzte Mahnung mit Hinweis
auf gerichtliches Mahnverfahren. Ton und Stufe bestimmt der Mensch. Der Mensch
versendet; lege den versendeten Brief als Beleg ab (`beleg_ablegen`) und
halte den Zugang nach Rückfrage mit `mahnung_vermerken` fest, damit Mahnung
und Zugang nachweisbar sind.

## 6. Buchen und Umsatzsteuer

- Gebucht wird erst, wenn Geld kommt. Eine eingeforderte, nicht gezahlte
  Zinsforderung bleibt außerhalb der Bücher.
- Verzugszinsen, Mahngebühren und Kosten eines gerichtlichen Mahnverfahrens
  sind Schadensersatz, kein Entgelt; keine Umsatzsteuer
  ([UStAE 1.3 Abs. 6][ustae13]).
- Konten (SKR04): Verzugszinsen 7110 Sonstige Zinserträge; Pauschale,
  Mahngebühren 4970 Versicherungsentschädigungen und Schadenersatzleistungen.
  Die Zuordnung am Bankumsatz macht [bank/zuordnung.md](../bank/zuordnung.md)
  mit `bankumsatz_buchen` (Aufteilung in Posten und Konten).

## 7. Verjährung

- Regelfrist drei Jahre ([§ 195 BGB][bgb195]), Beginn mit dem Schluss des
  Jahres, in dem der Anspruch entstanden ist und die GmbH von Anspruch und
  Schuldner wusste ([§ 199 Abs. 1 BGB][bgb199]). Eine Rechnungsforderung aus
  2023 verjährt danach mit Ablauf des 31.12.2026.
- Weise den Menschen beim Monatsabschluss auf Forderungen hin, deren
  Verjährung im laufenden Jahr endet. Hemmung (z. B. durch Mahnbescheid oder
  Klage) klärt er mit einem Anwalt.

[bgb187]: https://www.gesetze-im-internet.de/bgb/__187.html
[bgb195]: https://www.gesetze-im-internet.de/bgb/__195.html
[bgb199]: https://www.gesetze-im-internet.de/bgb/__199.html
[bgb247]: https://www.gesetze-im-internet.de/bgb/__247.html
[bgb286]: https://www.gesetze-im-internet.de/bgb/__286.html
[bgb288]: https://www.gesetze-im-internet.de/bgb/__288.html
[bbk]: https://www.bundesbank.de/de/bundesbank/organisation/agb-und-regelungen/basiszinssatz-607820
[ustae13]: https://www.steuerschroeder.de/steuergesetze/ustae/1.3
