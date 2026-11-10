# Folgejahr: Auflösung nach dem Saldenvortrag

Rechtsstand: 17. Oktober 2026. Ablauf: [ablauf.md](ablauf.md#3-ablauf), Schritt 16.

Nach dem Saldenvortrag des Menschen arbeitest du die
[Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten) ab.
Was Buchfink aus seinen Karteien nicht selbst auflöst, buchst du nach
Rückfrage mit `buchung_erfassen`.

## Inhalt

1. Abgrenzungen
2. Rückstellungen
3. Wertberichtigungen

## Inhalt

1. Abgrenzungen
2. Rückstellungen
3. Wertberichtigungen

## Abgrenzungen

Abgrenzungen außerhalb von Buchfinks Kartei löst Buchfink nicht auf:

- aktiv: Soll Aufwandskonto, Haben 1900, zum Jahresbeginn oder über die
  Laufzeit;
- passiv: Soll 3900, Haben Erlöskonto;
- sonstige Verbindlichkeit: Soll 3500, Haben Aufwandskonto, sobald die
  Rechnung gebucht ist.

Fertig, wenn 1900, 3900 und die Abgrenzungen auf 3500 und 1300 aus dem Vorjahr
aufgelöst sind.

## Rückstellungen

- Die Rechnung im Folgejahr buchst du als
  [Eingangsbeleg](../belege/ablauf.md); `beleg_buchen_mit_rueckfrage` mit
  `rueckstellung_id` verbraucht eine Rückstellung aus Buchfinks Kartei. Sonst
  löst du sie mit `buchung_erfassen` gegen dasselbe Aufwandskonto auf, bis zur
  Höhe der Rechnung.
- Fiel weniger an: Rest Soll Rückstellung, Haben 4930; fiel mehr an, ist der
  Rest Aufwand des neuen Jahres.
- Einen entfallenen Rest löst `abschlussbuchung` mit `art: rueckstellung`,
  `bewegung: aufloesung`, Kennung, Datum und begründetem Wegfall auf.
  Zuerst Vorschau, dann nach Rückfrage buchen.
  [Fortführung und Satznachweis](rueckstellungen.md#5-buchen).
- Aufgelöst wird nur bei entfallenem Grund ([§ 249 Abs. 2 Satz 2 HGB][hgb249]);
  prüfe jede bestehende Rückstellung zu jedem Stichtag (Konten 3070 bis
  3096).

## Wertberichtigungen

- Einzelwertberichtigung: Bei Ausfall löst `forderung_ausbuchen` die
  zugeordnete Wertberichtigung auf; nicht noch einmal von Hand auflösen
  ([rechnungen/forderungsausfall.md](../rechnungen/forderungsausfall.md)).
  Bei Zahlung passt der nächste Abschluss den Bestand an.
- Pauschalwertberichtigung: zum nächsten Stichtag neu berechnen; die Differenz
  geht auf 6920 oder 4920.

Steuerbescheid und Steuerrückstellung: [steuern-im-abschluss.md](steuern-im-abschluss.md#5-folgejahr).

[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
