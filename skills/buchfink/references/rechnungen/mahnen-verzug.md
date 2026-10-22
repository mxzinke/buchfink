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
| 30 Tage | 30 Tage nach Fälligkeit und Zugang der Rechnung; bei Verbrauchern nur mit Hinweis in der Rechnung | [§ 286 Abs. 3 Satz 1 BGB][bgb286] |
| Verweigerung | Kunde verweigert ernsthaft und endgültig | [§ 286 Abs. 2 Nr. 3 BGB][bgb286] |

- Ist der Zugang unsicher, kommt ein Unternehmer spätestens 30 Tage nach
  Fälligkeit und Empfang der Leistung in Verzug
  ([§ 286 Abs. 3 Satz 2 BGB][bgb286]).
- Kein Verzug ohne Vertretenmüssen, etwa bei fehlerhafter Rechnung
  ([§ 286 Abs. 4 BGB][bgb286]).
- Wertungsfrage: Ein Zahlungsziel ist nur „nach dem Kalender bestimmt“, wenn es
  vereinbart wurde (Vertrag, AGB); steht es nur auf der Rechnung, im Zweifel
  mahnen.
- Fehlt der Verbraucherhinweis auf einer älteren Rechnung, beginnt Verzug erst
  mit der Mahnung.

## 2. Verzugszinsen

- Basiszinssatz plus 5 Prozentpunkte; bei Entgeltforderungen ohne Verbraucher
  plus 9 ([§ 288 Abs. 1, 2 BGB][bgb288]). Der Basiszinssatz ändert sich zum
  1. Januar und 1. Juli; maßgeblich ist die Bekanntmachung der Bundesbank
  ([§ 247 BGB][bgb247]).

| gültig ab | Basiszinssatz |
|---|---|
| 01.01.2025 | 2,27 % |
| 01.07.2025 | 1,27 % |
| 01.01.2026 | 1,27 % |
| 01.07.2026 | 1,52 % |

Quelle: [Deutsche Bundesbank][bbk], abgerufen am 17.10.2026. Ab 01.01.2027 neu
nachsehen.

- Höhere Zinsen aus Vertrag bleiben möglich ([§ 288 Abs. 3 BGB][bgb288]); ein
  vorab vereinbarter Ausschluss ist gegenüber Unternehmern unwirksam
  ([§ 288 Abs. 6 BGB][bgb288]).

## 3. Pauschale und weitere Kosten

- Gegenüber Nichtverbrauchern zusätzlich 40 € Pauschale, auch bei
  Abschlagszahlungen, je Forderung einmal, anrechenbar auf
  Rechtsverfolgungskosten ([§ 288 Abs. 5 BGB][bgb288]).
- Weitere Schäden (Inkasso, Anwalt, [§ 288 Abs. 4 BGB][bgb288]) klärt der
  Mensch mit einem Anwalt.
- Mahngebühren nur, wenn sie in AGB oder Vertrag vereinbart sind.

## 4. Rechenweg

`offene_posten` liefert je Posten `verzug_ab`, `verzugszinsen` bis zum
`stichtag`, `pauschale` und den Mahnstand. Übernimm diese Werte und nenne
Verzugsbeginn, Zinssatz und Zeitraum. Selbst rechnest du nur, wenn ein Wert
fehlt (Verzug aus einer Abrede außerhalb von Buchfink) oder bestritten ist: ab
dem Tag nach Verzugsbeginn ([§ 187 Abs. 1 BGB][bgb187] entsprechend), je
Abschnitt gleichen Basiszinssatzes offener Betrag × Zinssatz × Tage ÷ 365, auf
Cent gerundet; nach Teilzahlungen mit dem Rest.

Beispiel: 1.190,00 € B2B, 30 Tage zu 10,52 %: 1.190,00 × 0,1052 × 30 ÷ 365 =
10,29 €, dazu 40 € Pauschale.

## 5. Mahnschreiben entwerfen

Inhalt: Rechnungsnummer, Rechnungsdatum, Betrag, Fälligkeit, offener Betrag;
Zahlungsbitte mit Datum; bei Verzug Zinsen mit Satz und Zeitraum und
Pauschale, jeweils mit Fundstelle; Bankverbindung aus den Stammdaten.

Stufen: Erinnerung, Mahnung mit Frist, letzte Mahnung mit Hinweis auf das
gerichtliche Mahnverfahren. Ton und Stufe bestimmt der Mensch. Versand und
Vermerk: [ablauf.md, Abschnitt 7](ablauf.md#7-überfällige-forderungen-und-mahnung).

## 6. Buchen und Umsatzsteuer

- Gebucht wird erst der Zahlungseingang; eine nicht gezahlte Zinsforderung
  bleibt außerhalb der Bücher.
- Verzugszinsen, Mahngebühren und Kosten des Mahnverfahrens sind
  Schadensersatz ohne Umsatzsteuer ([UStAE 1.3 Abs. 6][ustae13]).
- SKR04: Verzugszinsen 7110 Sonstige Zinserträge; Pauschale und Mahngebühren
  4970 Versicherungsentschädigungen und Schadenersatzleistungen. Aufteilung am
  Bankumsatz mit `bankumsatz_buchen` ([bank/zuordnung.md](../bank/zuordnung.md)).

## 7. Verjährung

- Drei Jahre ([§ 195 BGB][bgb195]) ab Schluss des Jahres, in dem der Anspruch
  entstand und die GmbH von Anspruch und Schuldner wusste
  ([§ 199 Abs. 1 BGB][bgb199]): eine Forderung aus 2023 verjährt mit Ablauf
  des 31.12.2026.
- Nenne beim Monatsabschluss Forderungen, deren Verjährung im laufenden Jahr
  endet. Hemmung (Mahnbescheid, Klage) klärt der Mensch mit einem Anwalt.

[bgb187]: https://www.gesetze-im-internet.de/bgb/__187.html
[bgb195]: https://www.gesetze-im-internet.de/bgb/__195.html
[bgb199]: https://www.gesetze-im-internet.de/bgb/__199.html
[bgb247]: https://www.gesetze-im-internet.de/bgb/__247.html
[bgb286]: https://www.gesetze-im-internet.de/bgb/__286.html
[bgb288]: https://www.gesetze-im-internet.de/bgb/__288.html
[bbk]: https://www.bundesbank.de/de/bundesbank/organisation/agb-und-regelungen/basiszinssatz-607820
[ustae13]: https://www.steuerschroeder.de/steuergesetze/ustae/1.3
