# Arbeitsweise des Buchfink-MCP-Servers

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0 (Oktober 2026).

Was ein Werkzeug tut und verlangt, sagen Beschreibung und Eingabeschema;
Ergebnisse und Fehler nennen den nächsten Schritt. Hier steht, was dort fehlt.
Fehlt ein hier genanntes Werkzeug, sag dem Menschen, welcher Schritt in der App
bleibt.

## Inhalt

1. Konventionen
2. Felder von `status`
3. Werkzeuge
4. Wann der Server „direkt“ zulässt
5. Idempotenz
6. Eingang, Ablage, Exportordner
7. Fehler
8. Erfassungsweg und Protokoll
9. Vorgänge ohne Werkzeug
10. Geschäftsjahr und Zeiträume
11. Eine Buchung korrigieren

## 1. Konventionen

Formate, Auswahlwerte und Pflichtfelder prüft das Eingabeschema.

## 2. Felder von `status`

Die Ausgabe nennt ihre Felder selbst. Über
[SKILL.md, Abschnitt 1](../../SKILL.md#1-sitzung-beginnen) hinaus brauchst du
`direktregeln` (Abschnitt 4), `verbindung.erfassungsweg` (Abschnitt 8) und
`stammdaten_vorschlaege`: wie der Mensch über deine Vorschläge entschieden hat,
mit dem Grund einer Ablehnung in `vermerk`.

## 3. Werkzeuge

Maßgeblich ist die Werkzeugliste des Servers; Stufen:
[SKILL.md, Abschnitt 2](../../SKILL.md#2-stufen-der-werkzeuge).

## 4. Wann der Server „direkt“ zulässt

Ob ein Vorgang Routine ist, entscheidet Buchfink: `beleg_lesen` und
`bankumsaetze` nennen die `einstufung`, `freigabe_noetig` in `gruende` jede
verletzte Regel. Die Schwellen (`direktregeln`) und die Standardgruppe eines
Lieferanten bestätigt nur der Mensch in der App. Den ersten Kontoauszug eines
Kontos liest der Mensch ein (`nur_app`).

## 5. Idempotenz

Sprechender Schlüssel je Vorgang (`beleg-142-buchen`). Nach einer
Zeitüberschreitung wiederholst du mit demselben Schlüssel und denselben
Angaben.

## 6. Eingang, Ablage, Exportordner

- Was du aus dem Eingang übernimmst, bestimmt der Mensch („arbeite den Eingang
  ab“, „buche die Rechnung von X“). Liegt ein Beleg nur im Chat, bitte ihn, das
  Original in den Eingang zu speichern (bei Claude Desktop: Chat-Anhang in den
  Eingang oder ins Buchfink-Fenster ziehen).
- Legst du selbst eine Datei in den Eingang, dann nur das Original, das der
  Mensch dir als Beleg gegeben hat, Byte für Byte; Buchfink prüft die Herkunft
  nicht.
- Die Ablage ist endgültig: Auch ein verworfener Beleg bleibt zehn Jahre im
  Archiv ([aufbewahrung.md](aufbewahrung.md)).

## 7. Fehler

Folge `naechster_schritt` und `werkzeug` aus dem Fehler. Bei
`fachliche_regel` legst du dem Menschen die `nachricht` vor und klärst den
Grund (etwa einen festgeschriebenen Zeitraum) mit ihm, statt ihn über ein
anderes Werkzeug zu umgehen; mit `wiederholbar: true` versuchst du es später
erneut. `intern` ist ein Programmfehler: Informiere den Menschen und wiederhole
nicht.

## 8. Erfassungsweg und Protokoll

Buchfink speichert an jeder Buchung, jedem Beleg und jeder Protokollzeile den
Erfassungsweg (`status.verbindung.erfassungsweg`); der Prüfer sieht ihn.
Schreib `beschreibung`, `grund` und `text` so, dass ein Prüfer ohne Rückfrage
versteht, was gebucht wurde und warum
([gobd.md, Abschnitt 2](gobd.md#2-nachvollziehbarkeit)).

## 9. Vorgänge ohne Werkzeug

Was sich per Storno korrigieren lässt, buchst du nach Rückfrage selbst, auch
Kapital-, Rücklagen- und Gesellschafterkonten, Gründungsbuchungen, Darlehen
und die beschlossene Ergebnisverwendung. Privatkonten hat die GmbH als
juristische Person nicht ([§ 13 Abs. 1 GmbHG][gmbhg13]); Geld zwischen GmbH
und Gesellschafter fließt über Verrechnungskonto, Darlehen oder Ausschüttung
([gesellschafter/ablauf.md](../gesellschafter/ablauf.md)).

Was nur der Mensch in der App erledigt, steht in
[SKILL.md, Abschnitt 4](../../SKILL.md#4-nur-der-mensch-in-der-app).

Kein Vorbehalt, aber ohne Werkzeug, deshalb in der App: einen vorhandenen
Kontakt ändern (USt-IdNr., IBAN, E-Rechnungsformat, Leitweg-ID), den
Saldenvortrag ausführen, den Zahlungseingang auf eine ausgestellte
Abschlagsrechnung zuordnen.

Noch nicht abgebildet, weder als Werkzeug noch in der App: Stornorechnung und
Rechnungskorrektur, Forderung ausbuchen, Versand vermerken, Rückzahlung von
Abschlagsrechnungen, Pflege der Anlagenkartei (Kennung, Quote,
Kennzeichen, Stückzahl; Abgang, Fertigstellung einer Anlage im Bau,
außerplanmäßige Abschreibung, Zuschreibung, Währung), Verwendungsanteil nach
§ 15a UStG, Eröffnungsbuchungen eines Umsteigers, berichtigte ZM und
CSV-Datei der ZM, die Texte des Anhangs zum Jahresabschluss, Zuführung,
Aufzinsung und Auflösung einer gebildeten Rückstellung samt
Abzinsungssätzen, Löschen nach Ablauf der Aufbewahrungsfrist. Sag dem Menschen, dass Buchfink den Vorgang
noch nicht kann, leg Sachverhalt, Grundlage und die nötige Buchung vor und
weiche nicht auf `buchung_erfassen` aus, wo Buchfink ein Dokument oder eine
Kartei mitführt.

Außerhalb von Buchfink, nur der Mensch: Senden in Mein ELSTER, im
BZSt-Online-Portal oder in einem Online-Dienst für die E-Bilanz; Zahlungen
freigeben; Unterschriften und Beschlüsse.

## 10. Geschäftsjahr und Zeiträume

- Gib `geschaeftsjahr` immer ausdrücklich mit und leite es aus dem Datum des
  Vorgangs ab, nie aus dem Jahr, das der Mensch gerade in der App ansieht.
- Ein Geschäftsjahr dauert höchstens zwölf Monate
  ([§ 240 Abs. 2 Satz 2 HGB][hgb240]). Ein abweichendes Geschäftsjahr wählt
  die GmbH bei der Gründung; eine spätere Umstellung wirkt steuerlich nur im
  Einvernehmen mit dem Finanzamt
  ([gruendung/rumpfjahr.md, Abschnitt 1](../gruendung/rumpfjahr.md#1-was-ein-rumpfgeschäftsjahr-ist)).
  Beides stellt der Mensch in der App ein.
- Der Abschluss durchläuft die Stände offen → aufgestellt → festgestellt →
  offengelegt (`geschaeftsjahre[].stand` in `status`); jeden Schritt setzt der
  Mensch in der App ([§ 264 Abs. 1 HGB][hgb264], [§ 42a GmbHG][gmbhg42a],
  [§ 325 HGB][hgb325]). In ein offenes Vorjahr buchst du wie gewohnt, in ein
  aufgestelltes nur nach Rückfrage, weil jede Buchung den aufgestellten
  Abschluss ändert. Ein festgestelltes Jahr nimmt keine Buchung mehr an.

Zeitpunkt eines Vorgangs:

| Was | Wann | Fundstelle |
|---|---|---|
| Ertrag und Forderung | mit der Leistung, nicht mit der Rechnung | [§ 252 Abs. 1 Nr. 4 HGB][hgb252] |
| Aufwand | im Geschäftsjahr, in dem die Leistung bezogen wurde, unabhängig von Rechnung und Zahlung; fehlt am Stichtag die Rechnung, Verbindlichkeit oder Rückstellung | [§ 252 Abs. 1 Nr. 5 HGB][hgb252] |
| Umsatzsteuer | mit Ablauf des Voranmeldungszeitraums der Leistung, bei Anzahlung der Vereinnahmung | [§ 13 Abs. 1 Nr. 1 Buchst. a UStG][ustg13] |
| Vorsteuer | wenn Leistung und Rechnung vorliegen; maßgeblich ist `eingang_am` | [§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15] |
| Fälligkeit | am vereinbarten Tag laut Beleg (`faellig_am`) | [§ 271 BGB][bgb271] |

## 11. Eine Buchung korrigieren

Eine Buchung wird nie geändert. Korrigiert wird durch Storno und Neubuchung
([gobd.md, Abschnitt 7](gobd.md#7-unveränderbarkeit-storno-festschreibung)).

1. Finde die Buchung mit `buchungen_suchen` und lies sie mit `buchung_lesen`.
2. Lege vor: was falsch ist, die Stornobuchung und die Neubuchung mit Zeilen.
3. Nach dem Ja: `buchung_stornieren` mit `buchung_id`, `grund` und
   `neubuchung`.
4. Buchungsdatum des Storno ist der Tag der Korrektur; Beleg- und
   Leistungsdatum übernimmt er, umsatzsteuerlich zählt also der alte Zeitraum
   (Nachtrag), beim Storno einer Ausgangsrechnung der laufende. Ist für den alten Zeitraum
   eine Voranmeldung schon übermittelt, frag den Menschen, ob sie berichtigt
   werden muss
   ([monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md)).

Ein festgeschriebener Zeitraum bleibt unberührt.

[gmbhg13]: https://www.gesetze-im-internet.de/gmbhg/__13.html
[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[bgb271]: https://www.gesetze-im-internet.de/bgb/__271.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb325]: https://www.gesetze-im-internet.de/hgb/__325.html
[gmbhg42a]: https://www.gesetze-im-internet.de/gmbhg/__42a.html
