# Aufstellen, Feststellen, Offenlegen

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 243, 245, 264, 267a, 316, 325,
326, 335 HGB und §§ 42a, 46, 48 GmbHG am 17.10.2026 gelesen.

## Inhalt

1. Stände in Buchfink
2. Aufstellen
3. Feststellen
4. Offenlegen
5. Ordnungsgeld
6. Änderung nach der Feststellung

## 1. Stände in Buchfink

`offen` → `aufgestellt` → `festgestellt` → `offengelegt`. Jeden Schritt setzt
der Mensch in der App, nur in dieser Reihenfolge, mit Datum. Buchfink lehnt ab:

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
- Frist: drei Monate nach Ende des Geschäftsjahres; kleine Gesellschaften
  dürfen später aufstellen, wenn es einem ordnungsgemäßen Geschäftsgang
  entspricht, spätestens nach sechs Monaten (§ 264 Abs. 1 Satz 3, 4).
- Unterzeichnung unter Angabe des Datums ([§ 245 HGB][hgb245]). Bei mehreren
  Geschäftsführern frag, wer unterzeichnet.
- Dein Teil: Bilanz, GuV, Anhangentwurf und die Liste Abschlussposten
  übergeben; die Fristen nennen. Der Mensch unterzeichnet, speichert das
  unterschriebene Dokument in den Eingang (du legst es auf seine Anweisung mit
  `beleg_ablegen` ab) und setzt `aufgestellt`.
- Danach schreibt der Mensch das Jahr in der App fest.

## 3. Feststellen

- Die Geschäftsführung legt den Abschluss unverzüglich nach der Aufstellung den
  Gesellschaftern vor ([§ 42a Abs. 1 GmbHG][gmbhg42a]).
- Die Gesellschafter beschließen über Feststellung und Ergebnisverwendung
  ([§ 46 Nr. 1 GmbHG][gmbhg46]), spätestens nach acht Monaten, bei kleinen
  Gesellschaften nach elf Monaten; die Satzung kann die Frist nicht verlängern
  ([§ 42a Abs. 2 GmbHG][gmbhg42a]).
- Form des Beschlusses: [§ 48 GmbHG][gmbhg48]; beim Alleingesellschafter eine
  unverzüglich unterschriebene Niederschrift.
- Dein Teil: einen Beschlussentwurf vorbereiten (Datum, Feststellung des
  Abschlusses mit Bilanzsumme und Ergebnis, Ergebnisverwendung), nach dem Ja
  den unterschriebenen Beschluss mit `beleg_ablegen` ablegen. Der Mensch setzt
  `festgestellt` mit Datum und Bezug auf den Beschluss.
- Danach nimmt das Jahr keine Buchung mehr an (`fachliche_regel`). Die
  beschlossene Ergebnisverwendung buchst du im Folgejahr mit
  `ergebnisverwendung_buchen`.

Pflichtprüfung ab mittelgroß: Ohne Prüfung kann der Abschluss nicht
festgestellt werden ([§ 316 Abs. 1 HGB][hgb316]). Dann: buchfink-grundlagen,
Grenzen.

## 4. Offenlegen

- Die Geschäftsführung übermittelt den festgestellten Abschluss in deutscher
  Sprache elektronisch an die das Unternehmensregister führende Stelle
  ([§ 325 Abs. 1 HGB][hgb325]), spätestens ein Jahr nach dem Stichtag
  ([§ 325 Abs. 1a HGB][hgb325]).
- Umfang ([§ 326 HGB][hgb326]):
  - kleine Gesellschaft: Bilanz und Anhang; der Anhang braucht die GuV
    betreffenden Angaben nicht;
  - Kleinstgesellschaft: nur die Bilanz; sie kann die dauerhafte Hinterlegung
    statt der Veröffentlichung verlangen, wenn sie mitteilt, dass sie zwei der
    drei Merkmale des § 267a Abs. 1 HGB an den maßgeblichen Stichtagen nicht
    überschreitet. Keine Kleinstkapitalgesellschaft ist eine Holding, deren
    einziger Zweck Erwerb, Verwaltung und Verwertung von Beteiligungen ist,
    ohne in die Verwaltung der Töchter einzugreifen; die Ausübung der
    Gesellschafterrechte zählt nicht ([§ 267a Abs. 3 Nr. 3 HGB][hgb267a]). Sie
    kann nicht hinterlegen. Wertungsfrage: Greift die Holding durch
    Geschäftsführung oder Dienstleistungen in die Verwaltung der Tochter ein?
    Dann kann sie Kleinstkapitalgesellschaft sein.
- Weg: die Publikations-Plattform des Unternehmensregisters
  ([publikations-plattform.de][pp]). Vor der ersten Übermittlung registriert
  und identifiziert sich jede Person, die übermittelt. Schritte, Entgelte und
  Nachweis: buchfink-grundlagen, references/pflichten.md, Abschnitt
  `offenlegung`. Buchfink übermittelt nicht selbst und erzeugt kein Format der
  Plattform; Bilanz und Anhang gibt `auswertung` aus, die Datei erstellt der
  Mensch.
- Wird der Abschluss geändert, ist auch die Änderung offenzulegen
  ([§ 325 Abs. 1b HGB][hgb325]).
- Dein Teil: Umfang nach Größenklasse nennen, Inhalt gegen den festgestellten
  Abschluss prüfen, Frist überwachen. Der Mensch übermittelt und setzt
  `offengelegt` mit Datum und Nachweis: dem Beleg der Veröffentlichung oder
  Hinterlegung aus „Meine Daten“ der Plattform.

## 5. Ordnungsgeld

- Wer nicht rechtzeitig offenlegt, gegen den führt das Bundesamt für Justiz ein
  Ordnungsgeldverfahren; es kann sich auch gegen die Gesellschaft richten
  ([§ 335 Abs. 1 HGB][hgb335]).
- Ordnungsgeld mindestens 2.500 €, höchstens 25.000 € (§ 335 Abs. 1 Satz 4).
- Mit der Androhung beginnt eine Frist von sechs Wochen, die Pflicht zu
  erfüllen oder Einspruch einzulegen (§ 335 Abs. 3).
- Erfüllt die Gesellschaft erst nach den sechs Wochen, setzt das Bundesamt
  herab: auf 500 € bei Hinterlegung einer Kleinstgesellschaft, auf 1.000 € bei
  einer kleinen Gesellschaft, sonst auf 2.500 €, bei nur geringfügiger
  Überschreitung niedriger (§ 335 Abs. 4 Satz 2).

Stand: Oktober 2026. Liegt eine Androhung vor, lege sie ab und nenne dem
Menschen sofort die Sechswochenfrist.

## 6. Änderung nach der Feststellung

- Ein festgestelltes Jahr öffnet nur der Mensch wieder (App, mit Grund); es
  steht dann wieder auf `aufgestellt`. Den geänderten Abschluss stellen die
  Gesellschafter erneut fest ([§ 46 Nr. 1 GmbHG][gmbhg46]); die Änderung ist
  offenzulegen ([§ 325 Abs. 1b HGB][hgb325]).
- Wertungsfrage: ob ein Fehler im alten Abschluss oder im laufenden Jahr
  korrigiert wird. Wesentliche Fehler, Steuerfolgen oder ein geänderter
  Bescheid: Steuerberater.

[hgb243]: https://www.gesetze-im-internet.de/hgb/__243.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb267a]: https://www.gesetze-im-internet.de/hgb/__267a.html
[hgb316]: https://www.gesetze-im-internet.de/hgb/__316.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[hgb326]: https://www.gesetze-im-internet.de/hgb/__326.html
[hgb335]: https://www.gesetze-im-internet.de/hgb/__335.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
[gmbhg46]: https://www.gesetze-im-internet.de/gmbhg/__46.html
[gmbhg48]: https://www.gesetze-im-internet.de/gmbhg/__48.html
[pp]: https://publikations-plattform.de
