# Unfertige Leistungen

Rechtsstand: 17. Oktober 2026.

Abschläge, Anzahlungen und Schlussrechnung:
[rechnungen/anzahlungen.md](../rechnungen/anzahlungen.md).

## Inhalt

1. Wann der Posten entsteht
2. Bewertung
3. Verlustprojekte
4. Erhaltene Anzahlungen
5. Buchen
6. Daten beschaffen

## 1. Wann der Posten entsteht

- Beim Werkvertrag ist der Erlös mit der Abnahme realisiert, bei Teilabnahme
  je Teil ([§ 640][bgb640], [§ 641 Abs. 1 BGB][bgb641];
  [Realisationsprinzip](ablauf.md#4-entscheidungsregeln)). Für ein am Stichtag
  begonnenes, weder abgenommenes noch abgerechnetes Projekt aktivierst du die
  angefallenen Kosten als unfertige Leistung
  ([§ 266 Abs. 2 B. I. 2. HGB][hgb266]).
- Dienstverträge mit laufender Abrechnung (Stunden, Pauschale) bilden in der
  Regel keinen Bestand; nicht abgerechnete Stunden des alten Jahres sind Erlös
  des alten Jahres ([abgrenzung.md](abgrenzung.md#5-ausstehende-rechnungen-und-erträge)).
- Wertungsfrage: abgenommen, Teilleistung abgerechnet oder Dienstvertrag? Frag
  nach Vertrag, Abnahmeprotokoll und Abrechnungsweise.

## 2. Bewertung

Zu Herstellungskosten ([§ 255 Abs. 2 HGB][hgb255]), höchstens zum niedrigeren
beizulegenden Wert (Abschnitt 3):

| Bestandteil | Behandlung |
|---|---|
| Einzelkosten: Fremdleistungen, Material, Löhne für Projektstunden | Pflicht (Satz 2) |
| angemessene, durch die Leistung veranlasste Gemeinkosten einschließlich anteiliger Abschreibung | Pflicht (Satz 2) |
| allgemeine Verwaltung, soziale Einrichtungen, freiwillige Sozialleistungen, Altersversorgung | Wahlrecht (Satz 3); steuerlich einheitlich mit dem Handelsabschluss ([§ 6 Abs. 1 Nr. 1b EStG][estg6]) |
| Vertriebskosten, Forschung | verboten (Satz 4) |
| Fremdkapitalzinsen | nur wenn für die Herstellung finanziert ([§ 255 Abs. 3 HGB][hgb255]) |
| Gewinnanteil | nie |

- Die gewählte Methode gilt in den Folgejahren weiter
  ([Stetigkeit](ablauf.md#4-entscheidungsregeln)).
- Wertungsfragen: Gemeinkostenschlüssel; Anteil des Geschäftsführergehalts
  bei eigener Projektarbeit. Leg Rechenweg und Annahmen offen.

## 3. Verlustprojekte

- Übersteigen bisherige und noch erwartete Kosten den Erlös, schreibst du den
  Bestand auf den niedrigeren Wert ab ([§ 253 Abs. 4 Satz 2 HGB][hgb253]).
- Für einen darüber hinausgehenden Verlust ist eine Rückstellung für drohende
  Verluste zu bilden ([§ 249 Abs. 1 Satz 1 HGB][hgb249]); steuerlich ist sie
  unzulässig ([rueckstellungen.md](rueckstellungen.md#4-abzinsung)) und kommt
  in die [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten).

## 4. Erhaltene Anzahlungen

Erhaltene Anzahlungen stehen unter den Verbindlichkeiten oder offen
abgesetzt von den Vorräten ([§ 268 Abs. 5 Satz 2 HGB][hgb268]). Buchfink führt
sie auf eigenen Konten; verrechne sie nicht mit dem Bestand.

## 5. Buchen

- `abschlussbuchung` mit `art: unfertige_leistungen`: Erhöhung Soll 1080,
  Haben 4815; Minderung umgekehrt. Zu jedem Stichtag passt du den Bestand je
  Projekt an; ist das Projekt abgerechnet, sinkt er auf null.
- `grund` und `notiz` je Projekt: Vertrag, Stand am Stichtag,
  Kostenaufstellung, Gemeinkostensatz, Begründung des Werts.

## 6. Daten beschaffen

Buchfink führt keine Projektkostenrechnung. Frag den Menschen nach laufenden
Projekten mit Vertrag und Abnahmestand, Stunden je Beschäftigtem und Projekt
und erwarteten Restkosten bei Verlustrisiko; Fremdleistungen findest du mit
`buchungen_suchen`.

Fertig, wenn jedes laufende Projekt bewertet oder vom Menschen als unwesentlich
festgehalten ist.

[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[hgb266]: https://www.gesetze-im-internet.de/hgb/__266.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[bgb640]: https://www.gesetze-im-internet.de/bgb/__640.html
[bgb641]: https://www.gesetze-im-internet.de/bgb/__641.html
