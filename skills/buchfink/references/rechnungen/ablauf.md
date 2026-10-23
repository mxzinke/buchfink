# Ablauf: Ausgangsrechnungen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Von der Rechnung bis zum Geldeingang oder Ausfall. Zahlungseingänge:
[bank/zuordnung.md](../bank/zuordnung.md); Bewertung zum Stichtag:
[jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## Inhalt

1. Rechnung ausstellen
2. Nach dem Ausstellen
3. Steuerfall
4. E-Rechnung oder sonstige Rechnung
5. Abschläge, Anzahlungen, Schlussrechnung
6. Storno und Berichtigung
7. Überfällige Forderungen und Mahnung
8. Kunde zahlt nicht
9. Wertungsfragen
10. Nur der Mensch in der App
11. Stolperfallen

## 1. Rechnung ausstellen

1. **Kunde.** `kontakte_suchen`, sonst `kontakt_anlegen` mit vollständiger
   Anschrift und `land`; EU-Unternehmen mit `ust_id`, Verbraucher mit
   `privatperson: true` (nur dann druckt Buchfink den Verzugshinweis),
   Behörden mit `leitweg_id`. Fertig, wenn `kontakt_id`, Land, Unternehmer oder
   Verbraucher und bei EU-Kunden die USt-IdNr. feststehen.
2. **Leistung klären.** Frag nach Art, Umfang, Leistungsdatum oder -zeitraum,
   Menge, Preis, Steuersatz und Zahlungsziel. Rechnungsdatum ist der Tag der
   Ausstellung
   ([pflichtangaben-ausgang.md, Abschnitt 2](pflichtangaben-ausgang.md#2-pflichtangaben-und-woher-buchfink-sie-nimmt),
   Nr. 3). Fertig, wenn jede Position Text, Menge, Einzelpreis und Satz hat und
   `leistung_von` feststeht.
3. **Steuerfall** nach Abschnitt 3. Fertig, wenn `steuerfall` und `satz` je
   Position feststehen.
4. **Form** nach Abschnitt 4. Fertig, wenn feststeht, ob eine E-Rechnung Pflicht ist und ob der
   Kunde ein Format verlangt.
5. **Durchrechnen** mit `rechnung_vorschau` und denselben Angaben wie für
   `rechnung_ausstellen`. Fehlende Unternehmensdaten schlägst du mit
   `stammdaten_vorschlagen` vor. Fertig, wenn `hinweise` keinen Eintrag mit
   `code: pflichtangabe` hat und `befunde` leer ist.
6. **Rückfrage** mit Kunde, Positionen, Netto, Steuer, Brutto, Steuerfall,
   Leistungszeitraum, Fälligkeit, Form und jedem Hinweis der Vorschau.
7. **Ausstellen** mit `rechnung_ausstellen`. Fertig, wenn die Antwort
   `nummer`, `buchung_id` und `dateien` enthält.
8. **Versand.** Gib dem Menschen die Pfade aus `dateien` oder versende sie, wenn
   dein Client das kann und er es will. Eine E-Rechnung geht als unveränderte
   Datei hinaus.

Frist für die Rechnung: Inland
[pflichtangaben-ausgang.md, Abschnitt 1](pflichtangaben-ausgang.md#1-wann-eine-rechnung-pflicht-ist),
EU-Unternehmen [ausland.md, Abschnitt 2](ausland.md#2-unternehmen-im-übrigen-eu-gebiet).

## 2. Nach dem Ausstellen

- Buchungstag ist das Ende des Leistungszeitraums, nicht das Rechnungsdatum
  ([arbeitsweise.md, Abschnitt 10](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume)).
  Meldet die Vorschau `leistungstag_festgeschrieben`, bucht Buchfink zum
  Rechnungsdatum; ist die Voranmeldung des Leistungszeitraums schon
  übermittelt, sag dem Menschen, dass sie zu berichtigen ist
  ([monatsabschluss/berichtigung.md](../monatsabschluss/berichtigung.md)).
- Entstehung der Steuer:
  [pflichtangaben-ausgang.md, Abschnitt 4](pflichtangaben-ausgang.md#4-leistungszeitpunkt),
  bei Anzahlungen [anzahlungen.md, Abschnitt 1](anzahlungen.md#1-wann-die-steuer-entsteht).
- Prüfe mindestens beim Monatsabschluss `offene_posten` (`richtung: forderung`,
  `nur_ueberfaellig: true`).

## 3. Steuerfall

| Kunde | Leistung | `steuerfall` |
|---|---|---|
| Inland, Unternehmer oder Verbraucher | Regelfall | `domestic`, Satz 19 oder 7 |
| Unternehmer im übrigen EU-Gebiet mit bestätigter USt-IdNr. | sonstige Leistung, Ort beim Kunden | `reverse_charge_supply` |
| Unternehmer im Drittland, Unternehmereigenschaft nachgewiesen | sonstige Leistung, Ort beim Kunden | `not_taxable` |
| Verbraucher im EU-Ausland oder Drittland | jede Leistung | erst [ausland.md](ausland.md) lesen |
| Leistung im Zusammenhang mit einem Grundstück im Ausland | – | Grenze, [ausland.md, Abschnitt 5](ausland.md#5-ausnahmen-vom-grundsatz) |

- `ust_id_uebersteuerung_grund` (USt-IdNr. nicht bestätigt) setzt du nur mit
  einem Grund, den der Mensch nennt.
- Im Zweifel über den Steuerfall frag vor dem Ausstellen; Folgen eines falschen
  Ausweises:
  [storno-berichtigung.md, Abschnitt 3](storno-berichtigung.md#3-falsch-ausgewiesene-umsatzsteuer).
- `exempt` (§ 4 UStG) und `zero_rated` nur, wenn der Mensch die Befreiung
  benennt.

## 4. E-Rechnung oder sonstige Rechnung

[e-rechnung-ausgang.md](e-rechnung-ausgang.md), Regel für deine Wahl in
Abschnitt 3 dort.

## 5. Abschläge, Anzahlungen, Schlussrechnung

- Steuer, Buchung, Absetzung:
  [anzahlungen.md, Abschnitte 1, 4 und 5](anzahlungen.md#4-wie-buchfink-bucht).
- Abschlagsrechnung: `rechnung_ausstellen` mit `art: abschlag`, die erste mit
  `verbund` und `kontakt_id`, jede weitere mit der `verbund_id` der Antwort.
  Schlussrechnung: `art: schlussrechnung` mit `verbund_id` und `rechnung`.
- Den Zahlungseingang auf Abschläge bildet Buchfink noch nicht ab
  ([anzahlungen.md, Abschnitt 4](anzahlungen.md#4-wie-buchfink-bucht)); sag
  das vor der ersten Abschlagsrechnung eines Verbunds.

## 6. Storno und Berichtigung

- Eine ausgestellte Rechnung bleibt unverändert. Weg je Fehler:
  [storno-berichtigung.md](storno-berichtigung.md); E-Rechnung zusätzlich
  [e-rechnung-ausgang.md, Abschnitt 8](e-rechnung-ausgang.md#8-berichtigung-und-aufbewahrung).
- Stornorechnung und Rechnungskorrektur bildet Buchfink noch nicht ab
  ([storno-berichtigung.md, Abschnitt 1](storno-berichtigung.md#1-was-buchfink-anbietet)).
  Stell keine zweite Rechnung über dieselbe Leistung aus: Forderung und Steuer
  stünden doppelt in den Büchern.
- Skonto am Zahlungseingang: [bank/skonto-differenzen.md](../bank/skonto-differenzen.md).

## 7. Überfällige Forderungen und Mahnung

1. `offene_posten` mit `nur_ueberfaellig: true`. Fertig, wenn du je Kunde
   Posten, offenen Betrag, Fälligkeit, Verzugszinsen und Mahnstand kennst.
2. Prüfe mit `bankumsaetze` (Status `offen`), ob eine Zahlung nur noch nicht
   zugeordnet ist; dann ordne sie nach [bank/zuordnung.md](../bank/zuordnung.md)
   zu.
3. Kläre Verzug ([mahnen-verzug.md, Abschnitt 1](mahnen-verzug.md#1-fälligkeit-und-verzug)).
4. Entwirf das Mahnschreiben
   ([mahnen-verzug.md, Abschnitt 5](mahnen-verzug.md#5-mahnschreiben-entwerfen)).
5. Der Mensch prüft und versendet. Leg den Brief mit `beleg_ablegen` ab und
   halte den Zugang nach Rückfrage mit `mahnung_vermerken` fest. Gebucht wird
   erst der Zahlungseingang
   ([mahnen-verzug.md, Abschnitt 6](mahnen-verzug.md#6-buchen-und-umsatzsteuer)).

## 8. Kunde zahlt nicht

[forderungsausfall.md, Abschnitt 3](forderungsausfall.md#3-ablauf).

## 9. Wertungsfragen

Leistungszeitpunkt, der den Voranmeldungszeitraum bestimmt
([pflichtangaben-ausgang.md, Abschnitt 4](pflichtangaben-ausgang.md#4-leistungszeitpunkt));
Teilleistung oder Anzahlung
([anzahlungen.md, Abschnitt 2](anzahlungen.md#2-teilleistung-oder-anzahlung));
Unternehmer, der für sein Unternehmen bezieht
([ausland.md, Abschnitt 1](ausland.md#1-erst-den-ort-der-leistung-bestimmen));
Uneinbringlichkeit
([forderungsausfall.md, Abschnitt 1](forderungsausfall.md#1-zweifelhaft-oder-uneinbringlich));
Verzug bei strittigem Zahlungsziel oder Zugang
([mahnen-verzug.md, Abschnitt 1](mahnen-verzug.md#1-fälligkeit-und-verzug)).

## 10. Nur der Mensch in der App

- Unternehmensstammdaten (du schlägst sie vor) und Nummernkreise.
- E-Rechnungsformat oder Leitweg-ID eines vorhandenen Kontakts.
- Was Buchfink noch nicht abbildet:
  [grundlagen/arbeitsweise.md, Abschnitt 9](../grundlagen/arbeitsweise.md#9-vorgänge-ohne-werkzeug).

## 11. Stolperfallen

- Verzugszinsen und Mahngebühren sind kein Umsatz
  ([mahnen-verzug.md, Abschnitt 6](mahnen-verzug.md#6-buchen-und-umsatzsteuer)).
  Fordere sie im Mahnschreiben, nie mit `rechnung_ausstellen`.
