# Aufstellen, Feststellen, Offenlegen

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 243, 245, 264, 267a, 316, 325,
326, 335 HGB und §§ 42a, 46, 48 GmbHG am 17.10.2026 gelesen. Frist, Weg,
Nachweis und Folgen jedes Schritts stehen im
[Pflichtenkalender](../grundlagen/pflichten.md#aufstellung); hier steht, was
darüber hinaus gilt.

## Inhalt

1. Stände in Buchfink
2. Aufstellen
3. Feststellen
4. Offenlegen
5. Ordnungsgeld
6. Änderung nach der Feststellung

## 1. Stände in Buchfink

Die Stände `offen` → `aufgestellt` → `festgestellt` → `offengelegt`
([Geschäftsjahr und Zeiträume](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume))
setzt der Mensch in der App, nur in dieser Reihenfolge, mit Datum. Buchfink
lehnt ab:

- ein Datum vor dem Ende des Geschäftsjahres; eine Feststellung vor der
  Aufstellung; eine Offenlegung vor der Feststellung;
- Aufstellen oder Feststellen, solange bei einer UG die gesetzliche Rücklage
  fehlt oder Bilanz und GuV nicht stimmig sind;
- die Feststellung ohne Festschreibung des ganzen Jahres;
- die Offenlegung ohne Übermittlungsnachweis oder Auftragsnummer.

Prüfe nach jedem Schritt mit `status` (`geschaeftsjahre[].stand`).

## 2. Aufstellen

- Die Geschäftsführung stellt den Jahresabschluss mit Anhang auf
  ([§ 264 Abs. 1 Satz 1 HGB][hgb264]) nach den Grundsätzen ordnungsmäßiger
  Buchführung, klar und übersichtlich ([§ 243 Abs. 1, 2 HGB][hgb243]).
- Frist, Weg und Nachweis:
  [grundlagen/pflichten.md#aufstellung](../grundlagen/pflichten.md#aufstellung).
- Unterzeichnung unter Angabe des Datums ([§ 245 HGB][hgb245]). Bei mehreren
  Geschäftsführern frag, wer unterzeichnet.
- Danach schreibt der Mensch das Jahr in der App fest.

## 3. Feststellen

- Die Geschäftsführung legt den Abschluss unverzüglich nach der Aufstellung den
  Gesellschaftern vor ([§ 42a Abs. 1 GmbHG][gmbhg42a]).
- Die Gesellschafter beschließen über Feststellung und Ergebnisverwendung
  ([§ 46 Nr. 1 GmbHG][gmbhg46]). Frist, Weg und Nachweis:
  [grundlagen/pflichten.md#feststellung](../grundlagen/pflichten.md#feststellung).
- Form des Beschlusses: Versammlung, Textform mit Zustimmung aller oder beim
  Alleingesellschafter eine unverzüglich unterschriebene Niederschrift
  ([§ 48 GmbHG][gmbhg48]).
- Danach nimmt das Jahr keine Buchung mehr an
  ([Geschäftsjahr und Zeiträume](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume)).
  Die beschlossene Ergebnisverwendung buchst du im Folgejahr mit
  `ergebnisverwendung_buchen`
  ([ergebnisverwendung-ug.md](ergebnisverwendung-ug.md#4-buchen-im-folgejahr)).

Ab mittelgroß braucht die Feststellung eine Pflichtprüfung
([groessenklassen-anhang.md](groessenklassen-anhang.md#3-folgen-der-größenklasse));
dann [Grenzen](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet).

## 4. Offenlegen

- Die Geschäftsführung übermittelt den festgestellten Abschluss in deutscher
  Sprache elektronisch an die das Unternehmensregister führende Stelle
  ([§ 325 Abs. 1 HGB][hgb325]). Frist, Weg über die Publikations-Plattform,
  Nachweis und Ordnungsgeld:
  [grundlagen/pflichten.md#offenlegung](../grundlagen/pflichten.md#offenlegung).
- Umfang ([§ 326 HGB][hgb326]):
  - kleine Gesellschaft: Bilanz und Anhang; der Anhang braucht die GuV
    betreffenden Angaben nicht;
  - Kleinstgesellschaft: nur die Bilanz; sie kann die dauerhafte Hinterlegung
    statt der Veröffentlichung verlangen, wenn sie mitteilt, dass sie zwei der
    drei Merkmale des § 267a Abs. 1 HGB an den maßgeblichen Stichtagen nicht
    überschreitet. Eine reine Holding ist keine Kleinstgesellschaft und kann
    nicht hinterlegen
    ([groessenklassen-anhang.md](groessenklassen-anhang.md#1-merkmale)).
- Dein Teil: Umfang nach Größenklasse nennen, Inhalt gegen den festgestellten
  Abschluss prüfen, Frist überwachen. Die Datei für die Plattform erstellt der
  Mensch aus Bilanz und Anhang (`auswertung`).

## 5. Ordnungsgeld

Verfahren, Höhe, Sechswochenfrist und Wiedereinsetzung stehen unter „Bei
Versäumnis“ in [grundlagen/pflichten.md#offenlegung](../grundlagen/pflichten.md#offenlegung).

## 6. Änderung nach der Feststellung

- Ein festgestelltes Jahr öffnet nur der Mensch wieder (App, mit Grund); es
  steht dann wieder auf `aufgestellt`. Den geänderten Abschluss stellen die
  Gesellschafter erneut fest ([§ 46 Nr. 1 GmbHG][gmbhg46]); die Änderung ist
  offenzulegen ([grundlagen/pflichten.md#offenlegung](../grundlagen/pflichten.md#offenlegung)).
- Wertungsfrage: ob ein Fehler im alten Abschluss oder im laufenden Jahr
  korrigiert wird. Wesentliche Fehler, Steuerfolgen oder ein geänderter
  Bescheid: Steuerberater.

[hgb243]: https://www.gesetze-im-internet.de/hgb/__243.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[hgb326]: https://www.gesetze-im-internet.de/hgb/__326.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhg48]: https://www.gesetze-im-internet.de/gmbhg/__48.html
