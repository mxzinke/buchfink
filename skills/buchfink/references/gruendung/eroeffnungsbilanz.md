# Eröffnungsbilanz

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, AO, EStG und GmbHG am
17.10.2026 gelesen.

## Inhalt

1. Pflicht
2. Stichtag in Buchfink
3. Inhalt bei der Gründung
4. Prüfen
5. Unterzeichnen, aufbewahren, übermitteln
6. Bestehende GmbH wechselt zu Buchfink

## 1. Pflicht

- Die GmbH gilt als Handelsgesellschaft ([§ 13 Abs. 3 GmbHG][gmbhg13]); die
  Vorschriften für Kaufleute gelten für sie ([§ 6 Abs. 1 HGB][hgb6]).
- Der Kaufmann stellt zu Beginn seines Handelsgewerbes eine Eröffnungsbilanz
  auf; für sie gelten die Vorschriften über die Bilanz des Jahresabschlusses
  entsprechend ([§ 242 Abs. 1 HGB][hgb242]). Zu Beginn ist auch ein Inventar
  aufzustellen ([§ 240 Abs. 1 HGB][hgb240]).

## 2. Stichtag in Buchfink

Buchfink beginnt das erste Geschäftsjahr am Tag der Beurkundung und stellt die
Eröffnungsbilanz auf diesen Tag. An diesem Tag stehen in den Büchern die
Zeichnung des Stammkapitals und, soweit schon geflossen, die Einzahlung.

Wertungsfrage: Wann das Handelsgewerbe beginnt, ist im Gesetz nicht als Datum
bestimmt. Begann die Geschäftstätigkeit erkennbar früher oder später, lege das
dem Menschen vor; den Beginn des Geschäftsjahres ändert nur er in der App.

## 3. Inhalt bei der Gründung

Beispiel GmbH mit 25.000 € Stammkapital, voll eingefordert, noch nichts
eingezahlt:

| Aktiva | | Passiva | |
|---|---|---|---|
| B.II Eingeforderte, noch ausstehende Kapitaleinlagen (1298) | 25.000 € | A.I Gezeichnetes Kapital (2900) | 25.000 € |

Ist die Hälfte nicht eingefordert, wird sie vom gezeichneten Kapital offen
abgesetzt; auf der Passivseite steht dann „Eingefordertes Kapital“ 12.500 €
und auf der Aktivseite die eingeforderte Einlage 12.500 €
([§ 272 Abs. 1 HGB][hgb272]).

Gründungsaufwand erscheint nicht als Aktivposten
([gruendungskosten.md, Abschnitt 1](gruendungskosten.md#1-bilanz-kein-aktivposten)).

## 4. Prüfen

1. Salden zum Beurkundungstag: `auswertung` (`art: kontoblatt`, `konto`,
   `bis: <Beurkundungstag>`) für jedes Konto der Gründungsbuchungen. Stehen
   im Geschäftsjahr nur diese Buchungen, zeigt `auswertung` (`art: bilanz`)
   dasselbe auf einen Blick.
2. Aktiva gleich Passiva. Geht sie nicht auf, fehlt eine Buchung; suche sie
   mit `buchungen_suchen` (`von` und `bis`: Beurkundungstag).
3. Nur Gründungsbuchungen am Stichtag: Zeichnung, gegebenenfalls Einzahlung.
4. Stammkapital laut Bilanz gleich Stammkapital laut Vertrag und `status`.

Fertig, wenn alle vier Punkte erfüllt sind und der Mensch die Bilanz gesehen
hat.

## 5. Unterzeichnen, aufbewahren, übermitteln

- Der Jahresabschluss ist vom Kaufmann mit Datum zu unterzeichnen
  ([§ 245 HGB][hgb245]); über [§ 242 Abs. 1 Satz 2 HGB][hgb242] gilt das
  entsprechend für die Eröffnungsbilanz. Für die GmbH unterzeichnen die
  Geschäftsführer. Das tut der Mensch.
- Die Eröffnungsbilanz ist zehn Jahre aufzubewahren
  ([§ 147 Abs. 1 Nr. 1, Abs. 3 AO][ao147]) und darf nicht nur als Wiedergabe
  auf einem Bild- oder Datenträger aufbewahrt werden
  ([§ 147 Abs. 2 AO][ao147], [§ 257 Abs. 3 HGB][hgb257]). Lege die
  unterzeichnete Fassung mit `beleg_ablegen` ab und erinnere den Menschen an
  das Original.
- Bei Eröffnung des Betriebs ist auch die Eröffnungsbilanz elektronisch an das
  Finanzamt zu übermitteln ([§ 5b Abs. 1 Satz 4 EStG][estg5b]). Taxonomie und
  Übermittlungsweg regelt [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md). `ebilanz_exportieren`
  schreibt die Bilanz eines Geschäftsjahres; für die Eröffnungsbilanz gibt es
  noch keinen Export. Der Mensch erfasst sie im Online-Dienst, oder ein
  Steuerberater übermittelt sie.

## 6. Bestehende GmbH wechselt zu Buchfink

Das ist keine Gründung. Die Eröffnungsbuchungen eines Umsteigers bildet
Buchfink noch nicht ab, weder als Werkzeug noch in der App. Gebucht würden
Sachkonten gegen 9000 Saldenvorträge, offene Posten einzeln gegen 9008
(Debitoren) und 9009 (Kreditoren), mit der Schlussbilanz des Altsystems als
Beleg. Sag das dem Menschen, bevor er Daten überträgt.

Dein Teil:

1. Schlussbilanz und Saldenliste des Altsystems mit `beleg_ablegen` ablegen.
2. Liste der offenen Forderungen und Verbindlichkeiten mit Belegnummer,
   Belegdatum, Fälligkeit und Kontakt vorbereiten; fehlende Kontakte mit
   `kontakt_anlegen` anlegen.
3. Sobald die Buchungen möglich sind, prüfen: `auswertung` (`art: susa`) zum
   ersten Tag gleich der Schlussbilanz; die Salden von 9000, 9008 und 9009
   ergeben zusammen null.

[ao147]: https://www.gesetze-im-internet.de/ao_1977/__147.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[gmbhg13]: https://www.gesetze-im-internet.de/gmbhg/__13.html
[hgb6]: https://www.gesetze-im-internet.de/hgb/__6.html
[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb257]: https://www.gesetze-im-internet.de/hgb/__257.html
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
