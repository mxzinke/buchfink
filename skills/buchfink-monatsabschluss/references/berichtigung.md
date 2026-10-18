# Fehler in einem übermittelten Zeitraum

Rechtsstand: 17. Oktober 2026. Wortlaut von § 153, § 164, § 168 AO und
§§ 17, 18, 18a, 18b UStG am 17.10.2026 gelesen.

## Inhalt

1. Berichtigung oder laufender Zeitraum
2. Pflicht und Recht zur Berichtigung
3. Ablauf in Buchfink
4. Nach der Jahreserklärung

## 1. Berichtigung oder laufender Zeitraum

| Vorgang | Zeitraum | Fundstelle |
|---|---|---|
| Entgelt ändert sich später (Skonto, Rabatt, Gutschrift, Ausfall, Rückgängigmachung) | Zeitraum der Änderung; der alte Zeitraum bleibt | [§ 17 Abs. 1 Satz 8 UStG][ustg17], [§ 18 Abs. 1 Satz 3 UStG][ustg18] |
| Fehler in der ursprünglichen Erfassung: falscher Satz, falscher Steuerfall, vergessene Rechnung, Leistung im falschen Monat, Vorsteuer ohne ordnungsgemäße Rechnung | der ursprüngliche Zeitraum wird berichtigt | [§ 153 AO][ao153], [§ 164 Abs. 2 AO][ao164] |
| Falsche Angaben zu Kennziffer 21 | Voranmeldung unverzüglich berichtigen, ZM binnen eines Monats | [§ 18b Satz 5 UStG][ustg18b], [§ 18a Abs. 10 UStG][ustg18a] |

Wertungsfrage: Ob ein Fehler oder eine spätere Änderung vorliegt, ist im
Einzelfall zu prüfen (etwa bei berichtigten Eingangsrechnungen). Lege dem
Menschen beide Lesarten mit Folge vor; im Zweifel Steuerberater.

## 2. Pflicht und Recht zur Berichtigung

- Erkennt die GmbH vor Ablauf der Festsetzungsfrist, dass eine Erklärung
  unrichtig ist und es dadurch zu einer Steuerverkürzung kommen kann, muss sie
  das unverzüglich anzeigen und richtigstellen ([§ 153 Abs. 1 AO][ao153]).
- War die Steuer zu hoch, darf sie die Änderung beantragen: Die Voranmeldung
  steht einer Festsetzung unter Vorbehalt der Nachprüfung gleich
  ([§ 168 Satz 1 AO][ao168]), die jederzeit geändert werden kann
  ([§ 164 Abs. 2 AO][ao164]). Eine Herabsetzung wirkt erst mit Zustimmung des
  Finanzamts (§ 168 Satz 2 AO).
- Sieht ein Fehler nach Absicht aus oder ist er groß, gilt
  buchfink-grundlagen (Grenzen, Warnzeichen).

## 3. Ablauf in Buchfink

1. **Fehler belegen.** Finde die Buchung (`buchungen_suchen`), nenne Zeitraum,
   Kennziffer und Betrag der Auswirkung.
2. **Korrigieren.** Storno und Neubuchung nach buchfink-grundlagen
   (Rückfrage). Der Storno trägt das heutige Datum, übernimmt aber Belegdatum
   und Leistungszeitraum der Ursprungsbuchung. Buchfink ordnet seine Steuer
   deshalb dem ursprünglichen Zeitraum zu und zeigt sie dort als Nachtrag.
   Ausnahme: Der Storno einer Ausgangsrechnung wirkt im Zeitraum des Stornos
   (§ 17 Abs. 1 Satz 8 UStG); dafür gibt es keine Berichtigung.
3. **Auswirkung zeigen.** `erklaerung` (`art: ustva`, `zeitraum: <alter
   Zeitraum>`) zeigt Nachträge und die neu gerechneten Kennziffern. Leg dem
   Menschen alt, neu und Differenz vor.
4. **Entscheidung.** Der Mensch entscheidet, ob berichtigt wird. Bei
   Steuerverkürzung ist das keine Wahl (§ 153 AO).
5. **Berichtigte Anmeldung.** Buchfink rechnet den ganzen Zeitraum neu, setzt
   Kennziffer 10 („berichtigte Anmeldung“) und verknüpft sie mit der
   übermittelten. Was zu tun ist, nennt `naechster_schritt` in `erklaerung`;
   legt die App die berichtigte Anmeldung an, tut das der Mensch. Danach
   Übermittlung und Transferticket wie im SKILL.md, Schritte 8 bis 10.
6. **ZM.** Betrifft der Fehler Kennziffer 21, folgt die berichtigte ZM.
7. **Zahlung.** Nachzahlung oder Erstattung ordnet buchfink-bank zu. Den
   Zahlungstermin legt der Mensch fest.

Fertig, wenn der Mensch entschieden hat und, falls berichtigt wird, das
Transferticket der berichtigten Anmeldung in Buchfink steht.

## 4. Nach der Jahreserklärung

Ist die Umsatzsteuer-Jahreserklärung für das Jahr schon übermittelt, betrifft
der Fehler auch sie. Ihre Berichtigung regelt buchfink-steuererklaerungen
([§ 18 Abs. 3 UStG][ustg18], [§ 153 AO][ao153]). Ob die Voranmeldung daneben
noch berichtigt wird, klärt der Mensch mit dem Finanzamt.

[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustg18a]: https://www.gesetze-im-internet.de/ustg_1980/__18a.html
[ustg18b]: https://www.gesetze-im-internet.de/ustg_1980/__18b.html
[ao153]: https://www.gesetze-im-internet.de/ao_1977/__153.html
[ao164]: https://www.gesetze-im-internet.de/ao_1977/__164.html
[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
