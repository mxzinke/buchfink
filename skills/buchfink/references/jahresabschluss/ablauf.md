---
name: buchfink-jahresabschluss
description: "Jahresabschluss einer GmbH oder UG in Buchfink vorbereiten und begleiten, bis die Geschäftsführung ihn aufstellt, die Gesellschafter ihn feststellen und er offengelegt ist. Lade den Skill zum Jahresende und bei Jahresabschluss, Bilanz, GuV oder Anhang; bei Inventur und Abstimmung zum Stichtag; bei Rechnungsabgrenzung, ausstehenden Rechnungen, Rückstellung und Abzinsung; bei Abschreibungslauf, außerplanmäßiger Abschreibung, Wertaufholung oder Anlagenabgang; bei Wertberichtigung auf Forderungen, unfertigen Leistungen und Fremdwährung zum Stichtag; bei Umsatzsteuer-Verrechnung und Steuerrückstellung; bei UG-Rücklage, Ergebnisverwendung, Gewinn- oder Verlustvortrag, Saldenvortrag; bei Größenklasse und Ordnungsgeld; wenn fristen Bilanzstichtag, Aufstellung, Feststellung, Ergebnisverwendung oder Offenlegung meldet. Steuerberechnung und E-Bilanz regelt buchfink-steuererklaerungen, Ausschüttung buchfink-gesellschafter, Beteiligungen buchfink-beteiligungen."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem zugelassenen Mandanten. Für die Steuerrückstellung zusätzlich buchfink-steuererklaerungen."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Jahresabschluss

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Der Jahresabschluss einer GmbH besteht aus Bilanz, Gewinn- und
Verlustrechnung und Anhang ([§ 242 Abs. 3][hgb242], [§ 264 Abs. 1 Satz 1 HGB][hgb264]).
Die Geschäftsführung stellt ihn auf, die Gesellschafter stellen ihn fest, die
Geschäftsführung legt ihn offen. Du stimmst ab, bereitest Abschlussbuchungen
vor, prüfst die Zahlen und entwirfst die Anhangangaben. Jede Abschlussbuchung
ist eine Rückfrage.

| Thema | Skill |
|---|---|
| Körperschaftsteuer, Gewerbesteuer, E-Bilanz, steuerliche Überleitung | buchfink-steuererklaerungen |
| Beteiligungen, Wertpapiere, Fonds zum Stichtag | buchfink-beteiligungen |
| Ausschüttung, Posten gegenüber Gesellschaftern ([§ 42 Abs. 3 GmbHG][gmbhg42]) | buchfink-gesellschafter |
| Eröffnungsbilanz, Rumpfgeschäftsjahr | buchfink-gruendung |
| Dezember und Voranmeldung | buchfink-monatsabschluss |

## 1. Pflichten und Fristen

`fristen` nennt Fälligkeit und Stand jedes Schritts; Buchfink wählt die Frist
nach der Größenklasse des Jahres. Frist, Weg, Nachweis und Folgen stehen je
Pflicht in buchfink-grundlagen:

| Schritt | Wer | Pflicht |
|---|---|---|
| Inventar und Abschlussbuchungen | du, nach Ja | [bilanzstichtag](../grundlagen/pflichten.md#bilanzstichtag) |
| Aufstellen | Geschäftsführung | [aufstellung](../grundlagen/pflichten.md#aufstellung) |
| Feststellen | Gesellschafter | [feststellung](../grundlagen/pflichten.md#feststellung) |
| Ergebnisverwendung beschließen; du buchst sie | Gesellschafter | [ergebnisverwendung](../grundlagen/pflichten.md#ergebnisverwendung) |
| Offenlegen | Geschäftsführung | [offenlegung](../grundlagen/pflichten.md#offenlegung) |

Für diese Fristen aus HGB und GmbHG verschiebt sich das Ende nicht auf den
nächsten Werktag; plane mit dem Kalenderdatum. Kleinstkapitalgesellschaften
gelten als klein ([§ 267a Abs. 2 HGB][hgb267a]). Keine
Kleinstkapitalgesellschaft ist eine Holding, deren einziger Zweck Erwerb,
Verwaltung und Verwertung von Beteiligungen ist, ohne in die Verwaltung der
Töchter einzugreifen; die Ausübung der Gesellschafterrechte zählt nicht
([§ 267a Abs. 3 Nr. 3 HGB][hgb267a]). Sie kann nicht hinterlegen.
Wertungsfrage: Greift die Holding durch Geschäftsführung oder Dienstleistungen
ein? Die Größenklasse bestimmst du in Schritt 1.

Die Fristen gelten mit und ohne Steuerberater. Erstellt ein Berater den
Abschluss, gib ihm Saldenliste, Nachweise und die Liste Abschlussposten;
unterzeichnen, feststellen und offenlegen bleibt Sache der Gesellschaft.

## 2. Weg der Abschlussbuchungen

Abschlussbuchungen tragen den Stichtag als Buchungsdatum. Buchfink nimmt sie
in der Abschlussperiode auch nach der Festschreibung des letzten Monats an, bis
das Jahr festgeschrieben ist. `abschluss_status` zeigt die Schritte mit der
nächsten Aktion.

| Vorgang | Werkzeug (Rückfrage) |
|---|---|
| Abschreibungslauf | `abschlussbuchung` mit `art: afa_lauf` |
| Abgrenzung, Rückstellung, Wertberichtigung, unfertige Leistungen, Fremdwährung, Umsatzsteuer-Verrechnung, Steuerrückstellung, UG-Rücklage | `abschlussbuchung` mit der passenden `art` |
| Anlage aus einer Buchung aktivieren | `abschlussbuchung` mit `art: anlage_aktivieren` |
| Umbuchungen, etwa auf Restlaufzeitkonten | `buchung_erfassen` mit `abschlussperiode: true` |
| Ergebnisverwendung nach dem Beschluss | `ergebnisverwendung_buchen` |
| Saldenvortrag | ohne Werkzeug: der Mensch in der App |
| Außerplanmäßige Abschreibung, Zuschreibung, Anlagenabgang | noch nicht abgebildet; du lieferst dem Menschen Zahlen und Begründung |
| Aufstellen, Jahr festschreiben, Feststellen, Offenlegen, Wiedereröffnen | nur der Mensch in der App |

Rechne jeden Schritt zuerst mit `nur_vorschau: true` und lege Buchungssatz und
Erläuterung vor. `abschlussbuchung` führt Buchfinks Karteien mit
(Anlagenverzeichnis, Rückstellungsspiegel, Auflösungsplan der Abgrenzung).
Führe daneben eine Liste **Abschlussposten** für Umbuchungen mit
`buchung_erfassen` und für steuerliche Abweichungen: Posten, Konto, Betrag,
Auflösung im Folgejahr, steuerliche Abweichung. Sie geht an den Menschen, an
buchfink-steuererklaerungen und ins Folgejahr. Felder, Eigenbeleg,
Idempotenz: [abschlussbuchungen.md](abschlussbuchungen.md).

## 3. Ablauf

1. **Vorbereiten.** `status` (Geschäftsjahr, `stand`, `festgeschrieben_bis`)
   und `abschluss_status`.
   Bestimme die Größenklasse aus Bilanzsumme und Umsatz dieses und des
   Vorjahres (`auswertung`, `art: bilanz` und `guv`) und der Arbeitnehmerzahl
   (frag den Menschen). Fertig, wenn die Größenklasse feststeht und du die
   Fälligkeiten aus `fristen` genannt hast.
2. **Jahr vollständig.** Alle Monate nach buchfink-monatsabschluss, einschließlich
   der letzten Voranmeldung. Leistungen des alten Jahres, deren Rechnung noch
   fehlt, grenzt du ab (Schritt 5). Fertig, wenn jeder Monat festgeschrieben
   oder ohne blockierenden Befund ist.
3. **Inventar und Abstimmung.** Zum Stichtag ist ein Inventar aufzustellen
   ([§ 240 Abs. 2 HGB][hgb240]). Weise jedes Bestandskonto mit Saldo nach:
   Bank, offene Posten, Darlehen, Steuer- und Lohnkonten, Gesellschafterkonten,
   Anlagen. Fertig, wenn jeder Saldo einen Nachweis hat oder als Rückfrage
   offen ist. [abschlussbuchungen.md](abschlussbuchungen.md).
4. **Anlagen.** `anlagen` zeigt Verzeichnis, fällige AfA und
   Aktivierungskandidaten. Aktiviere offene Kandidaten, dann
   `abschlussbuchung` mit `art: afa_lauf`. Prüfe, ob eine außerplanmäßige
   Abschreibung oder Wertaufholung nötig ist, und leg sie dem Menschen vor.
   Beide und Abgänge bildet Buchfink noch nicht ab. Fertig, wenn der
   Prüflauf kein `depreciation_missing` mehr meldet.
   [abschreibungen.md](abschreibungen.md).
5. **Abgrenzen.** Rechnungsabgrenzungsposten, ausstehende Rechnungen,
   sonstige Forderungen. Fertig, wenn jede Zahlung über den Stichtag hinaus und
   jede Leistung ohne Rechnung geprüft ist.
   [abgrenzung.md](abgrenzung.md).
6. **Rückstellungen.** Ungewisse Verbindlichkeiten, drohende Verluste,
   Abschluss- und Aufbewahrungskosten, Urlaub. Fertig, wenn jede Rückstellung
   mit Grund, Berechnung und Restlaufzeit gebucht oder verworfen ist.
   [rueckstellungen.md](rueckstellungen.md).
7. **Forderungen und Fremdwährung.** Einzel- und Pauschalwertberichtigung,
   Umrechnung zum Stichtag. Fertig, wenn jede überfällige Forderung bewertet
   ist. [forderungen.md](forderungen.md).
8. **Unfertige Leistungen.** Begonnene, nicht abgerechnete Projekte zu
   Herstellungskosten. Fertig, wenn jedes laufende Projekt bewertet oder als
   unwesentlich vermerkt ist.
   [unfertige-leistungen.md](unfertige-leistungen.md).
9. **Umsatzsteuer-Verrechnung.** Nach der letzten Voranmeldung verrechnet
   `abschlussbuchung` mit `art: ust_verrechnung` die Umsatzsteuer- und
   Vorsteuerkonten auf 3841 oder 1420; du prüfst den Betrag in der Vorschau.
   Fertig, wenn diese Konten null zeigen.
   [steuern-im-abschluss.md](steuern-im-abschluss.md).
10. **Steuerrückstellung.** Erst wenn alle übrigen Buchungen stehen:
    Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer des Jahres nach
    buchfink-steuererklaerungen, abzüglich Vorauszahlungen des Jahres;
    `abschlussbuchung` mit `art: steuerrueckstellung`. Fertig, wenn die
    Rückstellung gebucht ist und der Rechenweg in `grund` steht.
    [steuern-im-abschluss.md](steuern-im-abschluss.md).
11. **UG-Rücklage** (nur UG). Ein Viertel des um einen Verlustvortrag
    geminderten Jahresüberschusses ([§ 5a Abs. 3 GmbHG][gmbhg5a]);
    `abschlussbuchung` mit `art: ug_ruecklage`. Fertig, wenn 7765 an 2930
    gebucht ist.
    [ergebnisverwendung-ug.md](ergebnisverwendung-ug.md).
12. **Prüfen.** `auswertung` (`art: pruefbericht`, dann `bilanz`, `guv`).
    Bilanz geht auf, keine blockierenden Befunde, Abweichungen zum Vorjahr
    erklärt. Prüfe die Warnzeichen aus buchfink-grundlagen (Hälfte des
    Stammkapitals verloren, nicht durch Eigenkapital gedeckter Fehlbetrag).
    Fertig, wenn der Mensch Bilanz und GuV mit deinen Erläuterungen gesehen hat.
13. **Anhang.** Entwirf die Angaben nach Größenklasse. Fertig, wenn der Mensch
    einen Entwurf hat.
    [groessenklassen-anhang.md](groessenklassen-anhang.md).
14. **Aufstellen (nur App).** Die Geschäftsführung unterzeichnet mit Datum
    ([§ 245 HGB][hgb245]) und setzt den Stand `aufgestellt`. Danach schreibt
    der Mensch das Jahr fest; Buchfink verlangt das vor der Feststellung, und
    danach nimmt das Jahr auch in der Abschlussperiode keine Buchung mehr an.
    Fertig, wenn `status` den Stand und `mandant.festgeschrieben_bis` das
    Jahresende zeigt.
15. **Feststellen.** Die Geschäftsführung legt vor; die Gesellschafter
    beschließen Feststellung und Ergebnisverwendung. Lege den Beschluss mit
    `beleg_ablegen` ab. Der Mensch setzt `festgestellt` in der App. Fertig, wenn
    `status` den Stand zeigt.
16. **Folgejahr.** Der Mensch führt den Saldenvortrag in der App aus (noch
    kein Werkzeug) und wiederholt ihn nach späten Abschlussbuchungen; Buchfink
    meldet Differenzen.
    Du buchst die Ergebnisverwendung nach dem Ja mit
    `ergebnisverwendung_buchen` und arbeitest die Liste Abschlussposten ab.
    Fertig, wenn der Vortrag ohne Differenz steht und die Liste leer ist.
17. **Offenlegen.** Der Mensch übermittelt an das Unternehmensregister und
    setzt `offengelegt` mit dem Beleg aus „Meine Daten“ der Plattform.
    Registrierung, Identifizierung und Upload:
    [grundlagen/pflichten.md#offenlegung](../grundlagen/pflichten.md#offenlegung).
    Fertig, wenn `status` den Stand zeigt.
    [aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md).

## 4. Entscheidungsregeln

- Wertaufhellung: Risiken, die am Stichtag bestanden und bis zur Aufstellung
  bekannt werden, gehören in den Abschluss; Gewinne erst, wenn sie am Stichtag
  realisiert sind ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]).
- Jede Rückstellung braucht einen Zweck aus [§ 249 Abs. 1 HGB][hgb249]; für
  andere Zwecke sind sie verboten (Abs. 2).
- Bewertungsmethoden des Vorjahres beibehalten ([§ 252 Abs. 1 Nr. 6 HGB][hgb252]).
  Abweichungen nur in begründeten Ausnahmefällen (Abs. 2), mit Anhangangabe.
- Buchfink führt eine Bilanz für Handels- und Steuerrecht. Steuerliche
  Abweichungen (Abzinsung, Drohverlust, Sonderabschreibung) bucht du nicht;
  sie kommen in die Liste Abschlussposten.
- Wertungsfragen, die der Mensch entscheidet: Höhe einer Rückstellung,
  Nutzungsdauer, dauernde Wertminderung, Ausfallrisiko einer Forderung,
  Fertigstellungsgrad und Gemeinkosten, Ansatzwahlrechte, Wesentlichkeit
  kleiner Beträge. Nenne Grundlage, Optionen und Wirkung auf Ergebnis und
  Steuer.
- Mittelgroße oder große Gesellschaft, Konzern, Abschlussprüfung: anhalten,
  buchfink-grundlagen (Grenzen).

## 5. Nur der Mensch

- Arbeitnehmerzahl und Stammdaten des Geschäftsjahres in der App.
- Unterzeichnen, Aufstellen, Jahr festschreiben, Feststellen, Offenlegen,
  Wiedereröffnen in der App.
- Gesellschafterbeschlüsse fassen und unterschreiben.
- Saldenbestätigungen bei Banken, Kunden, Lieferanten anfordern.

Ohne Werkzeug, deshalb in der App: Saldenvortrag. Noch nicht abgebildet:
Anlagenabgang, außerplanmäßige Abschreibung und Zuschreibung.

## 6. Stolperfallen

- Meldet eine Abschlussbuchung `fachliche_regel` mit festgeschriebenem
  Zeitraum, ist das Jahr schon festgeschrieben. Halte an und frag den
  Menschen. Buche nie ins neue Jahr, um auszuweichen.
- Doppelte Abschlussbuchungen: Prüfe vor jeder Buchung mit `buchungen_suchen`
  (Konto, Text, Zeitraum), ob es sie schon gibt. Ändern heißt stornieren und neu
  buchen.
- Die Steuerrückstellung kommt zuletzt. Körperschaftsteuer und Gewerbesteuer
  mindern das zu versteuernde Ergebnis nicht ([§ 10 Nr. 2 KStG][kstg10],
  [§ 4 Abs. 5b EStG][estg4]); ein zweiter Rechengang ist unnötig.
- Die UG-Rücklage buchst du im Abschluss des Gewinnjahres, vor dem
  Gesellschafterbeschluss. Ohne sie lässt Buchfink weder die Festschreibung bis
  zum Jahresende noch Aufstellen oder Feststellen zu
  ([ergebnisverwendung-ug.md](ergebnisverwendung-ug.md)).
- Nach der Aufstellung ändert jede Buchung ins Jahr den aufgestellten
  Abschluss: nur nach Rückfrage. Nach der Feststellung nimmt das Jahr keine
  Buchung mehr an (`fachliche_regel`).
- Die Eröffnungsbilanz muss der Schlussbilanz entsprechen
  ([§ 252 Abs. 1 Nr. 1 HGB][hgb252]). Korrigiere nie über Vortragskonten.

## Referenzen

- [abschlussbuchungen.md](abschlussbuchungen.md): Lies
  vor der ersten Abschlussbuchung und für Inventar und Abstimmung.
- [abschreibungen.md](abschreibungen.md): Lies bei
  Abschreibungslauf, Wertminderung, Zuschreibung, Abgang, Anlagenspiegel.
- [abgrenzung.md](abgrenzung.md): Lies bei Zahlungen
  über den Stichtag, Leistungen ohne Rechnung, Disagio.
- [rueckstellungen.md](rueckstellungen.md): Lies bei
  jeder Rückstellung, ihrer Abzinsung, ihrem Verbrauch oder ihrer Auflösung.
- [forderungen.md](forderungen.md): Lies bei
  überfälligen oder zweifelhaften Forderungen und Fremdwährungsposten.
- [unfertige-leistungen.md](unfertige-leistungen.md):
  Lies bei Projekten, die über den Stichtag laufen.
- [steuern-im-abschluss.md](steuern-im-abschluss.md):
  Lies bei Umsatzsteuer-Verrechnung und Steuerrückstellung.
- [ergebnisverwendung-ug.md](ergebnisverwendung-ug.md):
  Lies bei UG, Ergebnisverwendung, Vorträgen, Verlust.
- [groessenklassen-anhang.md](groessenklassen-anhang.md):
  Lies für Größenklasse, Gliederung, Anhang und Angaben unter der Bilanz.
- [aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md):
  Lies für Unterzeichnung, Beschluss, Offenlegung, Ordnungsgeld, Stand in
  Buchfink.

[hgb240]: https://www.gesetze-im-internet.de/hgb/__240.html
[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb245]: https://www.gesetze-im-internet.de/hgb/__245.html
[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
[hgb267a]: https://www.gesetze-im-internet.de/hgb/__267a.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg42]: https://www.gesetze-im-internet.de/gmbhg/__42.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
