# Eröffnungsbilanz

Rechtsstand: 17. Oktober 2026.

## Inhalt

1. Pflicht
2. Stichtag in Buchfink
3. Inhalt bei der Gründung
4. Prüfen
5. Unterzeichnen, aufbewahren, übermitteln
6. Bestehende GmbH wechselt zu Buchfink

## 1. Pflicht

Die GmbH ist Handelsgesellschaft ([§ 13 Abs. 3 GmbHG][gmbhg13],
[§ 6 Abs. 1 HGB][hgb6]). Zu Beginn des Handelsgewerbes stellt sie ein
Inventar und eine Eröffnungsbilanz auf; für diese gelten die Vorschriften über
die Bilanz des Jahresabschlusses entsprechend ([§ 240 Abs. 1 HGB][hgb240],
[§ 242 Abs. 1 HGB][hgb242]).

## 2. Stichtag in Buchfink

Buchfink stellt die Eröffnungsbilanz auf den Beurkundungstag, den Beginn des
ersten Geschäftsjahres. Wertungsfrage: Das Gesetz nennt kein Datum für den
Beginn des Handelsgewerbes. Begann die Tätigkeit erkennbar früher oder später,
lege das dem Menschen vor; den Beginn des Geschäftsjahres ändert nur er in der
App.

## 3. Inhalt bei der Gründung

- Aktiva B.II: eingeforderte, noch ausstehende Kapitaleinlagen (1298).
- Passiva A.I: gezeichnetes Kapital (2900); nicht eingeforderte Einlagen
  offen abgesetzt, der Rest als „Eingefordertes Kapital“
  ([§ 272 Abs. 1 HGB][hgb272]).
- Gründungsaufwand ist kein Aktivposten
  ([gruendungskosten.md, Abschnitt 1](gruendungskosten.md#1-bilanz-kein-aktivposten)).

## 4. Prüfen

1. Salden zum Beurkundungstag: `auswertung` (`art: kontoblatt`, `konto`,
   `bis: <Beurkundungstag>`) für jedes Konto der Gründungsbuchungen. Stehen
   im Geschäftsjahr nur diese Buchungen, genügt `auswertung` (`art: bilanz`).
2. Aktiva gleich Passiva. Sonst suche die fehlende Buchung mit
   `buchungen_suchen` (`von` und `bis`: Beurkundungstag).
3. Am Stichtag stehen nur Gründungsbuchungen: Zeichnung, gegebenenfalls
   Einzahlung.
4. Stammkapital laut Bilanz gleich Stammkapital laut Vertrag.

Fertig, wenn alle vier Punkte erfüllt sind und der Mensch die Bilanz gesehen
hat.

## 5. Unterzeichnen, aufbewahren, übermitteln

Sind die vier Punkte erfüllt, meldest du die Pflichten `eroeffnungsbilanz` und
`ebilanz-eroeffnung` aus `fristen` mit `pflicht_vorbereitet`. Das PDF zum
Unterzeichnen gibt nur die App aus. Unterschrift, Ablage und Übermittlung:
[eroeffnungsbilanz](../grundlagen/pflichten.md#eroeffnungsbilanz),
[ebilanz-eroeffnung](../grundlagen/pflichten.md#ebilanz-eroeffnung) und
[steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md);
Aufbewahrung des Originals:
[grundlagen/aufbewahrung.md](../grundlagen/aufbewahrung.md), Abschnitte 1
und 3.

## 6. Bestehende GmbH wechselt zu Buchfink

Eröffnungsbuchungen eines Umsteigers bildet Buchfink nicht ab; sag das dem
Menschen, bevor er Daten überträgt. Fachlich wären es Sachkonten gegen 9000
Saldenvorträge und offene Posten einzeln gegen 9008 (Debitoren) und 9009
(Kreditoren), Beleg ist die Schlussbilanz des Altsystems.

Dein Teil: Schlussbilanz und Saldenliste des Altsystems mit `beleg_ablegen`
ablegen, offene Forderungen und Verbindlichkeiten mit Belegnummer,
Belegdatum, Fälligkeit und Kontakt auflisten, fehlende Kontakte mit
`kontakt_anlegen` anlegen.

[gmbhg13]: https://www.gesetze-im-internet.de/gmbhg/__13.html
[hgb6]: https://www.gesetze-im-internet.de/hgb/__6.html
[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
