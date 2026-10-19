---
name: buchfink-monatsabschluss
description: "Monat oder Quartal in Buchfink abschließen: vollständig machen, Konten abstimmen, Prüflauf, Festschreibung durch den Menschen, Umsatzsteuer-Voranmeldung und Zusammenfassende Meldung prüfen und als Eingabehilfe für Mein ELSTER vorlegen, Transferticket in der App. Lade den Skill beim Monats- oder Quartalsabschluss; wenn fristen eine Voranmeldung, ZM, Sondervorauszahlung oder Monatsfestschreibung meldet; wenn eine Voranmeldung oder ZM vorbereitet, in Mein ELSTER übertragen oder berichtigt werden soll; bei Fragen zu Voranmeldungszeitraum, Dauerfristverlängerung oder Sondervorauszahlung; bei Prüfbericht und blockierenden Befunden vor der Festschreibung; wenn ein Transferticket zu erfassen ist; wenn ein Fehler in einem schon übermittelten Zeitraum auffällt. Belege und Bankumsätze buchen buchfink-belege und buchfink-bank, die Umsatzsteuer-Jahreserklärung regelt buchfink-steuererklaerungen."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem zugelassenen Mandanten. Senden kann nur der Mensch: in Mein ELSTER, die ZM auch im BZSt-Online-Portal."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Monatsabschluss

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Der Monatsabschluss macht einen Zeitraum vollständig, prüft ihn, lässt ihn
festschreiben und bereitet die Umsatzsteuer-Meldungen vor. Die Reihenfolge ist
fest: **Prüflauf → Festschreibung → Übertragung in Mein ELSTER →
Transferticket**. Buchfink nimmt das Transferticket erst an, wenn der Zeitraum
festgeschrieben ist und die Zahlen dem heutigen Stand entsprechen. Was gemeldet
wird, ist damit unveränderbar.

Buchfink übermittelt nicht. Es zeigt Voranmeldung und ZM formularnah mit
Kennzahlen (Eingabehilfe). Der Mensch überträgt die Werte in Mein ELSTER und
sendet; du prüfst die Zahlen, legst sie als Tabelle vor und gehst die Eingabe
mit ihm durch.

| Thema | Skill |
|---|---|
| Belege, Vorsteuer, Steuerfälle der Eingangsseite | buchfink-belege |
| Bankumsätze, Zahlung der Zahllast, Sondervorauszahlung buchen | buchfink-bank |
| Ausgangsrechnungen, USt-IdNr. der Kunden | buchfink-rechnungen |
| Lohnkonten | buchfink-lohn |
| Umsatzsteuer-Jahreserklärung | buchfink-steuererklaerungen |
| Storno-Ablauf | buchfink-grundlagen |

## 1. Pflichten und Fristen

`fristen` nennt für jeden Zeitraum Fälligkeit und Stand dieser Pflichten. Frist,
Weg, Nachweis und Folgen stehen je Pflicht in buchfink-grundlagen:

| Pflicht | Abschnitt |
|---|---|
| Umsatzsteuer-Voranmeldung | [grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva) |
| Zusammenfassende Meldung | [grundlagen/pflichten.md#zm](../grundlagen/pflichten.md#zm) |
| Sondervorauszahlung | [grundlagen/pflichten.md#sondervorauszahlung](../grundlagen/pflichten.md#sondervorauszahlung) |
| Monatsfestschreibung | [grundlagen/pflichten.md#monatsfestschreibung](../grundlagen/pflichten.md#monatsfestschreibung) |

Buchfink rechnet die Fälligkeit aus Voranmeldungszeitraum
(`status` → `mandant.ust_zeitraum`), Dauerfristverlängerung und Bundesland;
diese Stammdaten stellt der Mensch in der App ein. Regeln für den Zeitraum:
[ustva.md](ustva.md). Mit Steuerberater gelten dieselben
Fristen.

Beginne gleich nach Monatsende. Dann bleibt Zeit, Befunde zu klären, bevor die
Frist läuft.

Den letzten Monat eines Voranmeldungszeitraums muss der Mensch spätestens am
Fälligkeitstag der Voranmeldung festschreiben: Du legst nur festgeschriebene
Zahlen vor, und Buchfink nimmt das Transferticket erst nach der Festschreibung
an. `fristen` nennt für die Monatsfestschreibung ein späteres Datum; plane
diesen Monat nach der Fälligkeit der Voranmeldung.

## 2. Ablauf

1. **Zeitraum bestimmen.** Rufe `status`, `aufgaben` (mit `geschaeftsjahr`)
   und `fristen` auf. Nimm den ältesten Monat nach
   `mandant.festgeschrieben_bis`. Bei
   vierteljährlicher Voranmeldung schließt du jeden Monat einzeln ab; die
   Voranmeldung folgt im dritten Monat. Fertig, wenn Monat (`JJJJ-MM`),
   Voranmeldungszeitraum (`JJJJ-MM` oder `JJJJ-Q1` …) und die Fälligkeit aus
   `fristen` feststehen.
2. **Vollständig machen.** Arbeite mit den Fach-Skills ab:
   - Dateien im Eingang: `eingang`; übernimm sie auf Anweisung des Menschen
     → buchfink-belege, CAMT-Dateien → buchfink-bank;
   - abgelegte, ungebuchte Belege: `belege_suchen` (`status: abgelegt`,
     `bis: <Monatsende>`) → buchfink-belege;
   - Kontoauszüge bis Monatsende importiert, jeder Umsatz erledigt:
     `bankumsaetze` (`status: offen`, `bis: <Monatsende>`) → buchfink-bank;
   - für jede im Monat erbrachte Leistung eine Ausgangsrechnung; frag den
     Menschen nach Leistungen ohne Rechnung → buchfink-rechnungen;
   - Lohnjournal → buchfink-lohn; Kreditkartenabrechnung → buchfink-bank.

   Die Steuer entsteht mit Ablauf des Zeitraums, in dem die Leistung erbracht
   wurde, bei Anzahlungen mit Ablauf des Zeitraums der Vereinnahmung
   ([§ 13 Abs. 1 Nr. 1 Buchst. a UStG][ustg13]). Eine fehlende Rechnung
   verschiebt die Steuer nicht. Fertig, wenn beide Listen leer sind oder jeder
   Rest als offene Frage beim Menschen liegt.
3. **Konten abstimmen.** `auswertung` (`art: kontoblatt`, `konto`,
   `bis: <Monatsende>`) je Konto. Prüfe: jedes Bankkonto gleich dem Endsaldo
   des Auszugs;
   1460 Geldtransit und 1370 Durchlaufende Posten auf null; 3610
   Kreditkartenabrechnung, Lohnverbindlichkeiten und Verrechnungskonten der
   Gesellschafter nach ihren Skills. Fertig, wenn jede Abweichung erklärt ist
   oder als Rückfrage vorliegt.
4. **Prüflauf.** `auswertung` (`art: pruefbericht`, `bis: <Monatsende>`). Behebe jeden blockierenden Befund an der Ursache. Lies die
   Hinweise und nenne dem Menschen die, die Zahlen berühren. Fertig, wenn kein
   blockierender Befund bleibt oder der Mensch für jeden verbliebenen
   entschieden hat, ihn bei der Festschreibung mit Begründung zu übergehen.
   Regeln: [festschreibung.md](festschreibung.md).
5. **Voranmeldung prüfen** (letzter Monat des Voranmeldungszeitraums).
   `erklaerung` (`art: ustva`, `zeitraum: <Zeitraum>`). Prüfe `befunde`, jede
   Kennzahl gegen die Konten dahinter, die `zahllast` und gemeldete Nachträge
   zu früheren Zeiträumen. Fertig, wenn der Mensch die Tabelle Kennziffer → Wert
   mit Zahllast gesehen und bestätigt hat. Abgleich:
   [ustva.md](ustva.md).
6. **ZM prüfen**, wenn Kennziffer 21 im Quartal belegt ist.
   `erklaerung` (`art: zm`, `zeitraum: <Quartal>`). Summe der Meldezeilen
   gleich Kennziffer 21 der Voranmeldungen des Quartals, jede USt-IdNr.
   bestätigt. Fertig, wenn beides stimmt. [zm.md](zm.md).
7. **Festschreiben lassen (nur App).** Nenne dem Menschen Zeitraum, Ergebnis
   des Prüflaufs und den Wortlaut der Begründung für übergangene Befunde. Er
   schreibt in der App fest. Fertig, wenn `status` ein
   `mandant.festgeschrieben_bis` am oder nach dem Monatsende zeigt.
8. **Vorbereitet melden.** Rufe `erklaerung` noch einmal auf; der
   `zahlen_hash` muss dem aus Schritt 5 gleichen. Eine CSV-Datei der ZM
   bildet Buchfink noch nicht ab; der Mensch überträgt die Zeilen aus deiner
   Tabelle. Melde jede
   Pflicht mit `pflicht_vorbereitet` (`schluessel` aus `fristen`, `notiz`)
   ([grundlagen/pflichten.md](../grundlagen/pflichten.md#arbeiten-mit-fristen)).
   Fertig, wenn der Hash passt und `fristen` die Pflicht als vorbereitet
   führt.
9. **In Mein ELSTER übertragen** nach Abschnitt 3. Fertig, wenn der Mensch
   gesendet hat und Transferticket und Zeitpunkt vorliegen.
10. **Transferticket erfassen lassen (nur App).** Nenne dem Menschen Zeitraum,
    Zeitpunkt und Transferticket. Fertig, wenn `erklaerung` den `status`
    `uebermittelt` zeigt und `fristen` die Pflicht als erledigt führt.
11. **Zahlung.** Die Zahllast ist am Fristtag fällig
    ([§ 18 Abs. 1 Satz 4 UStG][ustg18]). Erinnere an Überweisung oder
    Lastschrift; den Bankumsatz ordnet buchfink-bank zu. Ein Überschuss wird
    erst mit Zustimmung des Finanzamts wirksam ([§ 168 Satz 2 AO][ao168]).
    Fertig, wenn die Zahlung gebucht oder vom Menschen terminiert ist.

## 3. In Mein ELSTER übertragen

Das Senden ist unumkehrbar; es ist Sache des Menschen. Er sendet erst, wenn
die Werte in Mein ELSTER mit deiner Tabelle übereinstimmen.

1. **Tabelle vorlegen.** Steuernummer, Jahr, Zeitraum, je belegte Kennzahl der
   Wert, Kennzahl 10 (berichtigte Anmeldung) ja oder nein, Zahllast. Dieselbe
   Ansicht zeigt die App formularnah.
2. **Formular öffnen.** Der Mensch meldet sich mit dem Organisationszertifikat
   der GmbH an und öffnet das Formular (Weg in
   [grundlagen/pflichten.md#ustva](../grundlagen/pflichten.md#ustva)).
3. **Eintragen.** Er trägt Zeitraum und Kennzahlen ein; du gehst die Tabelle
   Zeile für Zeile mit ihm durch. Hat dein Client Zugriff auf seinen Browser
   und bittet er darum, trägst du die Werte ins geöffnete Formular ein;
   Anmeldung und Senden bleiben bei ihm.
4. **Prüfen.** Er lässt Mein ELSTER prüfen und vergleicht die berechnete
   Zahllast mit der Tabelle. Jede Abweichung, auch um Cent, klärt ihr vor dem
   Senden.
5. **Senden.** Er sendet und nennt dir Zeitpunkt und Transferticket aus dem
   Übermittlungsprotokoll. Er speichert das Protokoll als PDF in den Eingang;
   du legst es auf seine Anweisung mit `beleg_ablegen` ab.

Feldregeln (volle Euro, Kennzahl 500):
[ustva.md](ustva.md), Abschnitt 6. Die ZM geht über
Mein ELSTER oder das BZSt-Online-Portal:
[zm.md](zm.md), Abschnitt 4.

Endet ein Senden unklar (Abbruch, Zeitüberschreitung, Fehler nach dem Senden),
bitte den Menschen, in Mein ELSTER nachzusehen, ob die Anmeldung angekommen
ist, bevor er erneut sendet. Eine zweite Erstanmeldung für denselben Zeitraum
lehnt Buchfink ab.

Übermittelt ein Steuerberater die Voranmeldung, gib ihm die Tabelle.
Das Transferticket, das er nennt, erfasst der Mensch in der App.

## 4. Entscheidungsregeln

- Lege nur festgeschriebene Zahlen zur Übertragung vor, deren `zahlen_hash`
  dem aktuellen Stand entspricht. Ändert sich danach eine Buchung, rechne neu.
- Weicht die Zahllast in Mein ELSTER auch nur um Cent ab, nenne die
  Abweichung und ihre Ursache (etwa Rundung); gesendet wird erst nach Klärung.
- Wertungsfrage: Blockierender Befund kurz vor der Frist. Vorrang hat die
  Behebung. Sonst nenne die Optionen: (a) der Mensch übergeht den Befund mit
  Begründung, wenn er die Steuer nicht berührt; (b) verspätete Abgabe. Folgen:
  Verspätungszuschlag nach Ermessen ([§ 152 Abs. 1, 8 AO][ao152]),
  Säumniszuschlag bei verspäteter Zahlung ([§ 240 AO][ao240]), Geldbuße bei
  vorsätzlich nicht rechtzeitiger Entrichtung
  ([§ 26a Abs. 1, 3 UStG][ustg26a]).
- Ein Fehler in einem übermittelten Zeitraum: lies
  [berichtigung.md](berichtigung.md), bevor du etwas
  stornierst.
- Ergänzende Angaben zur Anmeldung (ab 2026 Kennzahl 500 statt 23) setzt du
  nur nach Rückfrage; sie lösen eine personelle Prüfung aus
  ([ustva.md](ustva.md)).

## 5. Nur der Mensch

- Zeitraum festschreiben (App).
- In Mein ELSTER oder im BZSt-Online-Portal anmelden, prüfen und senden.
- Transferticket erfassen (App).
- Antrag auf Dauerfristverlängerung und Anmeldung der Sondervorauszahlung in
  Mein ELSTER; Voranmeldungszeitraum und Dauerfristverlängerung in der App
  ([dauerfristverlaengerung.md](dauerfristverlaengerung.md)).
- Einen blockierenden Befund mit Begründung übergehen (App, bei der
  Festschreibung).

## 6. Stolperfallen

- Buchfink ordnet Umsätze nach dem Leistungsdatum zu, Vorsteuer nach dem
  spätesten Datum aus Leistung, Rechnung und Eingang. Vergleiche Kennziffern
  deshalb über die Konten und die Daten der Belege, nicht über das
  Buchungsdatum.
- Ein Storno übernimmt Beleg- und Leistungsdatum der Ursprungsbuchung. Die
  Steuer daraus erscheint als Nachtrag zum alten Zeitraum. Nur der Storno einer
  Ausgangsrechnung wirkt im laufenden Zeitraum.
- Minderungen nach § 17 UStG (Skonto, Rabatt, Ausfall) gehören in den
  Zeitraum der Änderung ([§ 17 Abs. 1 Satz 8 UStG][ustg17],
  [§ 18 Abs. 1 Satz 3 UStG][ustg18]); der alte Zeitraum bleibt.
- Die Sondervorauszahlung wird in der letzten Voranmeldung des Jahres
  angerechnet (Kennziffer 39, [§ 48 Abs. 4 UStDV][ustdv48]). Prüfe sie im
  Dezember.
- Nimm Fälligkeiten aus `fristen`. Buchfink verschiebt nach
  [§ 108 Abs. 3 AO][ao108] mit den Feiertagen des Bundeslands aus den
  Stammdaten; stimmt das Bundesland nicht, bitte den Menschen um die Korrektur
  in der App.
- Der Steuerfall der Buchung entscheidet zwischen Kennziffer 46/47
  (`reverse_charge_eu`) und 84/85 (`reverse_charge`). Eine Warnung
  `reverse_charge_eu_supplier` klärst du vor der Abgabe
  ([ustva.md](ustva.md), Abschnitt 4).
- Für die ZM gilt die Dauerfristverlängerung nicht.
- Zertifikatsdatei und Passwort für ELSTER bedient nur der Mensch. Lass sie dir
  nie nennen oder schicken.

## Referenzen

- [ustva.md](ustva.md): Lies bei Fragen zu Pflicht,
  Voranmeldungszeitraum, Kennzahlen, Abgleich, Rundung, Zuschlägen und vor der
  Eingabe in Mein ELSTER.
- [zm.md](zm.md): Lies, sobald Leistungen an
  Unternehmer im EU-Ausland im Zeitraum liegen.
- [dauerfristverlaengerung.md](dauerfristverlaengerung.md):
  Lies bei Antrag, Sondervorauszahlung, Wechsel des Voranmeldungszeitraums und
  im Januar.
- [festschreibung.md](festschreibung.md): Lies bei
  Befunden des Prüflaufs und bei Fragen, was nach der Festschreibung gesperrt
  ist.
- [berichtigung.md](berichtigung.md): Lies, wenn ein
  Fehler einen schon übermittelten Zeitraum betrifft.

[ustg13]: https://www.gesetze-im-internet.de/ustg_1980/__13.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
[ustg26a]: https://www.gesetze-im-internet.de/ustg_1980/__26a.html
[ustdv48]: https://www.gesetze-im-internet.de/ustdv_1980/__48.html
[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao152]: https://www.gesetze-im-internet.de/ao_1977/__152.html
[ao168]: https://www.gesetze-im-internet.de/ao_1977/__168.html
[ao240]: https://www.gesetze-im-internet.de/ao_1977/__240.html
