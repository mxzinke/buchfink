# Ablauf: Monatsabschluss

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Reihenfolge: **Prüflauf → Festschreibung → Übertragung in Mein ELSTER →
Transferticket**; das Transferticket nimmt Buchfink erst nach der
Festschreibung an. Du prüfst die Zahlen und legst sie als Tabelle vor; der
Mensch überträgt und sendet.

## Inhalt

1. Pflichten und Fristen
2. Ablauf
3. In Mein ELSTER übertragen
4. Entscheidungsregeln
5. Nur der Mensch
6. Stolperfallen

## 1. Pflichten und Fristen

`fristen` nennt Fälligkeit, Stand und `anker` je Pflicht (Voranmeldung, ZM,
Sondervorauszahlung, Monatsfestschreibung); Frist, Weg, Nachweis und Folgen
stehen unter diesem Anker in [grundlagen/pflichten.md](../grundlagen/pflichten.md).

Die Fälligkeit folgt aus Voranmeldungszeitraum (`status` →
`mandant.ust_zeitraum`, Regeln in
[ustva.md](ustva.md#2-voranmeldungszeitraum)), Dauerfristverlängerung und
Bundesland; diese Stammdaten schlägst du mit `stammdaten_vorschlagen` vor.

Beginne gleich nach Monatsende. Den letzten Monat eines
Voranmeldungszeitraums muss der Mensch spätestens am Fälligkeitstag der
Voranmeldung festschreiben; `fristen` nennt für die Monatsfestschreibung ein
späteres Datum.

## 2. Ablauf

1. **Zeitraum bestimmen.** `status`, `aufgaben` (mit `geschaeftsjahr`),
   `fristen`. Nimm den ältesten Monat nach `mandant.festgeschrieben_bis`. Bei
   vierteljährlicher Voranmeldung schließt du jeden Monat einzeln ab; die
   Voranmeldung folgt im dritten. Fertig, wenn Monat (`JJJJ-MM`),
   Voranmeldungszeitraum (`JJJJ-MM` oder `JJJJ-Q1` …) und Fälligkeit
   feststehen.
2. **Vollständig machen.**
   - Eingang: `eingang`, auf Anweisung übernehmen →
     [belege/ablauf.md](../belege/ablauf.md), CAMT →
     [bank/ablauf.md](../bank/ablauf.md);
   - ungebuchte Belege: `belege_suchen` (`status: abgelegt`,
     `bis: <Monatsende>`) → [belege/ablauf.md](../belege/ablauf.md);
   - Kontoauszüge bis Monatsende, jeder Umsatz erledigt: `bankumsaetze`
     (`status: offen`, `bis: <Monatsende>`) → [bank/ablauf.md](../bank/ablauf.md);
   - je erbrachter Leistung eine Ausgangsrechnung; frag nach Leistungen ohne
     Rechnung → [rechnungen/ablauf.md](../rechnungen/ablauf.md);
   - Lohnjournal → [lohn/ablauf.md](../lohn/ablauf.md); Kreditkarte →
     [bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md).

   Die Steuer einer Leistung gehört in ihren Zeitraum, auch ohne Rechnung
   ([ustva.md](ustva.md#1-pflicht-und-form)). Fertig, wenn beide Listen leer
   sind oder jeder Rest als Frage beim Menschen liegt.
3. **Konten abstimmen.** `auswertung` (`art: susa`, dann `art: kontoblatt`
   mit `konto`, jeweils `bis: <Monatsende>`; Bilanz und GuV ebenso zum
   Stichtag). Bankkonten gleich Endsaldo des Auszugs; 1460
   Geldtransit und 1370 Durchlaufende Posten auf null; 3610 nach
   [bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md), Lohn
   nach [lohn/abstimmung.md](../lohn/abstimmung.md), Gesellschafter nach
   [gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md).
   Fertig, wenn jede Abweichung erklärt ist oder als Rückfrage vorliegt.
4. **Prüflauf.** `auswertung` (`art: pruefbericht`, `bis: <Monatsende>`) nach
   [festschreibung.md](festschreibung.md). Behebe blockierende Befunde an der
   Ursache; nenne Hinweise, die Zahlen berühren. Fertig, wenn keiner bleibt
   oder der Mensch jeden verbliebenen übergehen will.
5. **Voranmeldung prüfen** (letzter Monat des Zeitraums). `erklaerung`
   (`art: ustva`, `zeitraum`), Abgleich nach
   [ustva.md, Abschnitt 5](ustva.md#5-abgleich-vor-der-freigabe). Fertig, wenn
   der Mensch Kennziffern und Zahllast bestätigt hat.
6. **ZM prüfen**, wenn Kennziffer 21 im Quartal belegt ist. `erklaerung`
   (`art: zm`, `zeitraum: <Quartal>`), Abgleich nach
   [zm.md, Abschnitt 3](zm.md#3-inhalt-und-abgleich). Fertig, wenn die Summe
   der Meldezeilen Kennziffer 21 gleicht und jede USt-IdNr. bestätigt ist.
7. **Festschreiben lassen (nur App).** Nenne Zeitraum, Ergebnis des Prüflaufs
   und den Wortlaut der Begründung für übergangene Befunde. Fertig, wenn
   `status` `mandant.festgeschrieben_bis` am oder nach dem Monatsende zeigt.
8. **Vorbereitet melden.** `erklaerung` erneut; der `zahlen_hash` muss dem aus
   Schritt 5 gleichen. Melde jede Pflicht mit `pflicht_vorbereitet`
   (`schluessel` aus `fristen`, `notiz`). Fertig, wenn der Hash passt und
   `fristen` die Pflicht als vorbereitet führt.
9. **In Mein ELSTER übertragen** nach Abschnitt 3. Fertig, wenn der Mensch
   gesendet hat und Transferticket und Zeitpunkt vorliegen.
10. **Transferticket erfassen lassen (nur App).** Fertig, wenn `erklaerung`
    `status: uebermittelt` und `fristen` die Pflicht als erledigt zeigt.
11. **Zahlung** bis zur Fälligkeit; Buchung nach
    [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart).
    Ein Überschuss wirkt erst mit Zustimmung des Finanzamts
    ([ustva.md](ustva.md#1-pflicht-und-form)). Fertig, wenn die Zahlung
    gebucht oder vom Menschen terminiert ist.

## 3. In Mein ELSTER übertragen

- Weg: Pflichtenkalender [ustva](../grundlagen/pflichten.md#ustva),
  [zm](../grundlagen/pflichten.md#zm); Durchgang und unklares Senden:
  [steuererklaerungen/abgabe.md, Abschnitte 4 und 5](../steuererklaerungen/abgabe.md#4-eingabe-in-mein-elster).
- Tabelle: Steuernummer, Jahr, Zeitraum, je belegte Kennzahl der Wert,
  Kennzahl 10 ja oder nein, Zahllast; Feldregeln in
  [ustva.md, Abschnitt 6](ustva.md#6-eingabe-in-mein-elster).
- Übermittelt ein Steuerberater, bekommt er die Tabelle; sein Transferticket
  erfasst der Mensch in der App.

## 4. Entscheidungsregeln

- Blockierender Befund kurz vor der Frist: zuerst beheben. Sonst Optionen
  nennen: (a) übergehen mit Begründung, wenn er die Steuer nicht berührt
  ([festschreibung.md](festschreibung.md#2-befunde-übergehen)); (b) verspätete
  Abgabe mit den Folgen aus
  [grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva).
- Fehler in einem übermittelten Zeitraum: erst [berichtigung.md](berichtigung.md)
  lesen, dann stornieren.

## 5. Nur der Mensch

Zusätzlich zu [SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app):
blockierende Befunde übergehen; in Mein ELSTER oder im BZSt-Online-Portal
senden, auch Dauerfristverlängerung und Sondervorauszahlung
([dauerfristverlaengerung.md](dauerfristverlaengerung.md)).

## 6. Stolperfallen

- Buchfink ordnet Umsätze nach Leistungsdatum zu, Vorsteuer nach dem spätesten
  Datum aus Leistung, Rechnung und Eingang. Vergleiche Kennziffern über Konten
  und Belegdaten, nie über das Buchungsdatum.
- Ein Storno zählt im Zeitraum der Ursprungsbuchung, der Storno einer
  Ausgangsrechnung im laufenden
  ([berichtigung.md](berichtigung.md#3-ablauf-in-buchfink)); Minderungen nach
  § 17 UStG im Zeitraum der Änderung
  ([berichtigung.md](berichtigung.md#1-berichtigung-oder-laufender-zeitraum)).
- Dezember: Anrechnung der Sondervorauszahlung prüfen
  ([dauerfristverlaengerung.md](dauerfristverlaengerung.md#4-anrechnung-im-dezember)).
- Fälligkeiten nur aus `fristen`; sie hängen an den Feiertagen des Bundeslands
  ([grundlagen/pflichten.md](../grundlagen/pflichten.md#fristende-zahlung-verlängerung)).
