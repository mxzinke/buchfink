# Ablauf: Monatsabschluss

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Der Monatsabschluss macht einen Zeitraum vollständig, prüft ihn, lässt ihn
festschreiben und bereitet die Umsatzsteuer-Meldungen vor. Die Reihenfolge ist
fest: **Prüflauf → Festschreibung → Übertragung in Mein ELSTER →
Transferticket**. Buchfink nimmt das Transferticket erst an, wenn der Zeitraum
festgeschrieben ist und die Zahlen dem heutigen Stand entsprechen. Was gemeldet
wird, ist damit unveränderbar.

Buchfink übermittelt nicht. Es zeigt Voranmeldung und ZM formularnah mit
Kennzahlen (Eingabehilfe). Der Mensch überträgt die Werte in Mein ELSTER, die
ZM auch im BZSt-Online-Portal, und sendet; du prüfst die Zahlen, legst sie als
Tabelle vor und gehst die Eingabe mit ihm durch.

| Thema | Referenz |
|---|---|
| Belege buchen | [belege/ablauf.md](../belege/ablauf.md) |
| Vorsteuer, Steuerfälle der Eingangsseite | [belege/vorsteuer.md](../belege/vorsteuer.md), [belege/steuerfaelle.md](../belege/steuerfaelle.md) |
| Bankumsätze | [bank/ablauf.md](../bank/ablauf.md) |
| Zahlung der Zahllast, Sondervorauszahlung buchen | [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart) |
| Ausgangsrechnungen, USt-IdNr. der Kunden | [rechnungen/ablauf.md](../rechnungen/ablauf.md), [rechnungen/ausland.md](../rechnungen/ausland.md) |
| Lohnkonten | [lohn/abstimmung.md](../lohn/abstimmung.md) |
| Umsatzsteuer-Jahreserklärung | [steuererklaerungen/ust-jahr.md](../steuererklaerungen/ust-jahr.md) |
| Storno-Ablauf | [grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren) |

## Inhalt

1. Pflichten und Fristen
2. Ablauf
3. In Mein ELSTER übertragen
4. Entscheidungsregeln
5. Nur der Mensch
6. Stolperfallen

## 1. Pflichten und Fristen

`fristen` nennt für jeden Zeitraum Fälligkeit und Stand dieser Pflichten. Frist,
Weg, Nachweis und Folgen stehen je Pflicht im Pflichtenkalender:

| Pflicht | Abschnitt |
|---|---|
| Umsatzsteuer-Voranmeldung | [grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva) |
| Zusammenfassende Meldung | [grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm) |
| Sondervorauszahlung | [grundlagen/pflichten.md#sondervorauszahlung](../grundlagen/pflichten.md#sondervorauszahlung) |
| Monatsfestschreibung | [grundlagen/pflichten.md#monatsfestschreibung](../grundlagen/pflichten.md#monatsfestschreibung) |

Buchfink rechnet die Fälligkeit aus Voranmeldungszeitraum
(`status` → `mandant.ust_zeitraum`), Dauerfristverlängerung und Bundesland;
diese Stammdaten stellt der Mensch in der App ein. Regeln für den Zeitraum:
[ustva.md](ustva.md#2-voranmeldungszeitraum).

Beginne gleich nach Monatsende. Dann bleibt Zeit, Befunde zu klären, bevor die
Frist läuft.

Plane den letzten Monat eines Voranmeldungszeitraums nach der Fälligkeit der
Voranmeldung: Der Mensch muss ihn spätestens an diesem Tag festschreiben;
`fristen` nennt für die Monatsfestschreibung ein späteres Datum
([grundlagen/pflichten.md#monatsfestschreibung](../grundlagen/pflichten.md#monatsfestschreibung)).

## 2. Ablauf

1. **Zeitraum bestimmen.** Rufe `status`, `aufgaben` (mit `geschaeftsjahr`)
   und `fristen` auf. Nimm den ältesten Monat nach
   `mandant.festgeschrieben_bis`. Bei
   vierteljährlicher Voranmeldung schließt du jeden Monat einzeln ab; die
   Voranmeldung folgt im dritten Monat. Fertig, wenn Monat (`JJJJ-MM`),
   Voranmeldungszeitraum (`JJJJ-MM` oder `JJJJ-Q1` …) und die Fälligkeit aus
   `fristen` feststehen.
2. **Vollständig machen.** Arbeite die Bereiche ab:
   - Dateien im Eingang: `eingang`; übernimm sie auf Anweisung des Menschen
     → [belege/ablauf.md](../belege/ablauf.md), CAMT-Dateien →
     [bank/ablauf.md](../bank/ablauf.md);
   - abgelegte, ungebuchte Belege: `belege_suchen` (`status: abgelegt`,
     `bis: <Monatsende>`) → [belege/ablauf.md](../belege/ablauf.md);
   - Kontoauszüge bis Monatsende importiert, jeder Umsatz erledigt:
     `bankumsaetze` (`status: offen`, `bis: <Monatsende>`) →
     [bank/ablauf.md](../bank/ablauf.md);
   - für jede im Monat erbrachte Leistung eine Ausgangsrechnung; frag den
     Menschen nach Leistungen ohne Rechnung →
     [rechnungen/ablauf.md](../rechnungen/ablauf.md);
   - Lohnjournal → [lohn/ablauf.md](../lohn/ablauf.md);
     Kreditkartenabrechnung →
     [bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md).

   Die Steuer einer Leistung gehört in ihren Zeitraum, auch wenn die Rechnung
   fehlt ([ustva.md](ustva.md#1-pflicht-und-form)). Fertig, wenn beide Listen
   leer sind oder jeder Rest als offene Frage beim Menschen liegt.
3. **Konten abstimmen.** `auswertung` (`art: kontoblatt`, `konto`,
   `bis: <Monatsende>`) je Konto. Prüfe: jedes Bankkonto gleich dem Endsaldo
   des Auszugs; 1460 Geldtransit und 1370 Durchlaufende Posten auf null; 3610
   Kreditkartenabrechnung nach
   [bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md),
   Lohnverbindlichkeiten nach [lohn/abstimmung.md](../lohn/abstimmung.md),
   Verrechnungskonten der Gesellschafter nach
   [gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md).
   Fertig, wenn jede Abweichung erklärt ist oder als Rückfrage vorliegt.
4. **Prüflauf.** `auswertung` (`art: pruefbericht`, `bis: <Monatsende>`).
   Behebe jeden blockierenden Befund an der Ursache. Lies die Hinweise und
   nenne dem Menschen die, die Zahlen berühren. Fertig, wenn kein
   blockierender Befund bleibt oder der Mensch für jeden verbliebenen
   entschieden hat, ihn bei der Festschreibung mit Begründung zu übergehen.
   Regeln: [festschreibung.md](festschreibung.md).
5. **Voranmeldung prüfen** (letzter Monat des Voranmeldungszeitraums).
   `erklaerung` (`art: ustva`, `zeitraum: <Zeitraum>`). Prüfe `befunde`, jede
   Kennzahl gegen die Konten dahinter, die `zahllast` und gemeldete Nachträge
   zu früheren Zeiträumen. Fertig, wenn der Mensch die Tabelle Kennziffer → Wert
   mit Zahllast gesehen und bestätigt hat. Abgleich:
   [ustva.md](ustva.md#5-abgleich-vor-der-freigabe).
6. **ZM prüfen**, wenn Kennziffer 21 im Quartal belegt ist.
   `erklaerung` (`art: zm`, `zeitraum: <Quartal>`). Summe der Meldezeilen
   gleich Kennziffer 21 der Voranmeldungen des Quartals, jede USt-IdNr.
   bestätigt. Fertig, wenn beides stimmt. [zm.md](zm.md#3-inhalt-und-abgleich).
7. **Festschreiben lassen (nur App).** Nenne dem Menschen Zeitraum, Ergebnis
   des Prüflaufs und den Wortlaut der Begründung für übergangene Befunde. Er
   schreibt in der App fest. Fertig, wenn `status` ein
   `mandant.festgeschrieben_bis` am oder nach dem Monatsende zeigt.
8. **Vorbereitet melden.** Rufe `erklaerung` noch einmal auf; der
   `zahlen_hash` muss dem aus Schritt 5 gleichen. Melde jede
   Pflicht mit `pflicht_vorbereitet` (`schluessel` aus `fristen`, `notiz`)
   ([grundlagen/pflichten.md](../grundlagen/pflichten.md#arbeiten-mit-fristen)).
   Fertig, wenn der Hash passt und `fristen` die Pflicht als vorbereitet
   führt.
9. **In Mein ELSTER übertragen** nach Abschnitt 3. Fertig, wenn der Mensch
   gesendet hat und Transferticket und Zeitpunkt vorliegen.
10. **Transferticket erfassen lassen (nur App).** Nenne dem Menschen Zeitraum,
    Zeitpunkt und Transferticket. Fertig, wenn `erklaerung` den `status`
    `uebermittelt` zeigt und `fristen` die Pflicht als erledigt führt.
11. **Zahlung.** Erinnere an Überweisung oder Lastschrift bis zur Fälligkeit
    ([grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva)). Den
    Bankumsatz buchst du nach
    [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart).
    Ein Überschuss wirkt erst mit Zustimmung des Finanzamts
    ([ustva.md](ustva.md#1-pflicht-und-form)). Fertig, wenn die Zahlung
    gebucht oder vom Menschen terminiert ist.

## 3. In Mein ELSTER übertragen

Das Senden ist unumkehrbar; es ist Sache des Menschen. Anmelden, Eintragen,
Prüfen, Senden und Ablegen des Protokolls gehen wie bei den Jahreserklärungen
([steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md#4-eingabe-in-mein-elster)).
Für die Voranmeldung gilt dazu:

1. **Tabelle.** Steuernummer, Jahr, Zeitraum, je belegte Kennzahl der Wert,
   Kennzahl 10 (berichtigte Anmeldung) ja oder nein, Zahllast. Dieselbe
   Ansicht zeigt die App formularnah.
2. **Formular.** Weg in
   [grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva),
   Feldregeln (volle Euro, Kennzahl 500) in
   [ustva.md](ustva.md#6-eingabe-in-mein-elster).
3. **ZM.** Sie geht über Mein ELSTER oder das BZSt-Online-Portal
   ([zm.md](zm.md#4-übermittlung)).

Eine zweite Erstanmeldung für denselben Zeitraum lehnt Buchfink ab. Endet ein
Senden unklar, gilt
[steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md#5-nach-dem-senden).

Übermittelt ein Steuerberater die Voranmeldung, gib ihm die Tabelle.
Das Transferticket, das er nennt, erfasst der Mensch in der App.

## 4. Entscheidungsregeln

- Lege nur festgeschriebene Zahlen zur Übertragung vor, deren `zahlen_hash`
  dem aktuellen Stand entspricht. Ändert sich danach eine Buchung, rechne neu.
- Wertungsfrage: Blockierender Befund kurz vor der Frist. Vorrang hat die
  Behebung. Sonst nenne die Optionen: (a) der Mensch übergeht den Befund mit
  Begründung, wenn er die Steuer nicht berührt
  ([festschreibung.md](festschreibung.md#3-befunde-übergehen)); (b) verspätete
  Abgabe mit den Folgen aus
  [grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva).
- Ein Fehler in einem übermittelten Zeitraum: lies
  [berichtigung.md](berichtigung.md), bevor du etwas stornierst.

## 5. Nur der Mensch

Zu den Vorbehalten in
[SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app) gehören hier:

- Zeitraum festschreiben und dabei einen blockierenden Befund mit Begründung
  übergehen (App).
- In Mein ELSTER oder im BZSt-Online-Portal anmelden, prüfen und senden;
  danach das Transferticket in der App erfassen.
- Antrag auf Dauerfristverlängerung und Anmeldung der Sondervorauszahlung in
  Mein ELSTER; Voranmeldungszeitraum und Dauerfristverlängerung in der App
  ([dauerfristverlaengerung.md](dauerfristverlaengerung.md)).

## 6. Stolperfallen

- Buchfink ordnet Umsätze nach dem Leistungsdatum zu, Vorsteuer nach dem
  spätesten Datum aus Leistung, Rechnung und Eingang. Vergleiche Kennziffern
  deshalb über die Konten und die Daten der Belege, nicht über das
  Buchungsdatum.
- Ein Storno zählt im Zeitraum der Ursprungsbuchung, der Storno einer
  Ausgangsrechnung im laufenden
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren),
  [berichtigung.md](berichtigung.md#3-ablauf-in-buchfink)).
- Minderungen nach § 17 UStG (Skonto, Rabatt, Ausfall) gehören in den
  Zeitraum der Änderung
  ([berichtigung.md](berichtigung.md#1-berichtigung-oder-laufender-zeitraum)).
- Prüfe im Dezember die Anrechnung der Sondervorauszahlung
  ([dauerfristverlaengerung.md](dauerfristverlaengerung.md#4-anrechnung-im-dezember)).
- Nimm Fälligkeiten aus `fristen`. Buchfink verschiebt sie mit den
  Feiertagen des Bundeslands aus den Stammdaten
  ([grundlagen/pflichten.md](../grundlagen/pflichten.md#fristende-zahlung-verlängerung));
  stimmt das Bundesland nicht, bitte den Menschen um die Korrektur in der App.
- Zertifikatsdatei und Passwort für ELSTER bedient nur der Mensch
  ([steuererklaerungen/abgabe.md](../steuererklaerungen/abgabe.md#2-organisationszertifikat)).
