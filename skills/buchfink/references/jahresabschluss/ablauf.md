# Ablauf: Jahresabschluss

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Der Jahresabschluss einer GmbH besteht aus Bilanz, Gewinn- und
Verlustrechnung und Anhang ([§ 242 Abs. 3][hgb242], [§ 264 Abs. 1 Satz 1 HGB][hgb264]).
Die Geschäftsführung stellt ihn auf, die Gesellschafter stellen ihn fest, die
Geschäftsführung legt ihn offen; du stimmst ab, bereitest Abschlussbuchungen
vor, prüfst die Zahlen und entwirfst die Anhangangaben.

## Inhalt

1. Pflichten und Fristen
2. Weg der Abschlussbuchungen
3. Ablauf
4. Entscheidungsregeln
5. Nur der Mensch
6. Stolperfallen

## 1. Pflichten und Fristen

`fristen` nennt Fälligkeit und Stand jedes Schritts; Buchfink wählt die Frist
nach der Größenklasse des Jahres. Frist, Weg, Nachweis und Folgen stehen je
Pflicht im Pflichtenkalender:

| Schritt | Wer | Pflicht |
|---|---|---|
| Inventar und Abschlussbuchungen | du, nach Ja | [bilanzstichtag](../grundlagen/pflichten.md#bilanzstichtag) |
| Aufstellen | Geschäftsführung | [aufstellung](../grundlagen/pflichten.md#aufstellung) |
| Feststellen | Gesellschafter | [feststellung](../grundlagen/pflichten.md#feststellung) |
| Ergebnisverwendung beschließen; du buchst sie | Gesellschafter | [ergebnisverwendung](../grundlagen/pflichten.md#ergebnisverwendung) |
| Offenlegen | Geschäftsführung | [offenlegung](../grundlagen/pflichten.md#offenlegung) |

Wie das Ende dieser Fristen aus HGB und GmbHG fällt, steht unter
[Fristende](../grundlagen/pflichten.md#fristende-zahlung-verlängerung). Die
Größenklasse bestimmst du in Schritt 1; Merkmale, Kleinstgesellschaft und
reine Holding: [groessenklassen-anhang.md](groessenklassen-anhang.md#1-merkmale).

Die Fristen gelten mit und ohne Steuerberater
([Pflichtenkalender](../grundlagen/pflichten.md#arbeiten-mit-fristen)).
Erstellt ein Berater den Abschluss, gib ihm Saldenliste, Nachweise und die
Liste Abschlussposten; unterzeichnen, feststellen und offenlegen bleibt Sache
der Gesellschaft.

Angrenzende Themen:

| Thema | Referenz |
|---|---|
| Körperschaftsteuer, Gewerbesteuer, steuerliche Überleitung | [steuererklaerungen/kst.md](../steuererklaerungen/kst.md), [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md) |
| E-Bilanz | [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md) |
| Beteiligungen, Wertpapiere, Fonds zum Stichtag | [beteiligungen/bewertung-veraeusserung.md](../beteiligungen/bewertung-veraeusserung.md) |
| Ausschüttung | [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md) |
| Posten gegenüber Gesellschaftern | [gesellschafter/ausweis.md](../gesellschafter/ausweis.md) |
| Eröffnungsbilanz, Rumpfgeschäftsjahr | [gruendung/eroeffnungsbilanz.md](../gruendung/eroeffnungsbilanz.md), [gruendung/rumpfjahr.md](../gruendung/rumpfjahr.md) |
| Dezember und Voranmeldung | [monatsabschluss/ablauf.md](../monatsabschluss/ablauf.md) |

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

Vorschau, Felder je `art`, Karteien, Eigenbeleg, Idempotenz und die Liste
**Abschlussposten** für Umbuchungen und steuerliche Abweichungen:
[abschlussbuchungen.md](abschlussbuchungen.md).

## 3. Ablauf

1. **Vorbereiten.** `status` (Geschäftsjahr, `stand`, `festgeschrieben_bis`)
   und `abschluss_status`.
   Bestimme die Größenklasse aus Bilanzsumme und Umsatz dieses und des
   Vorjahres (`auswertung`, `art: bilanz` und `guv`) und der Arbeitnehmerzahl
   (frag den Menschen). Fertig, wenn die Größenklasse feststeht und du die
   Fälligkeiten aus `fristen` genannt hast.
   [groessenklassen-anhang.md](groessenklassen-anhang.md).
2. **Jahr vollständig.** Alle Monate nach dem
   [Monatsabschluss](../monatsabschluss/ablauf.md), einschließlich der letzten
   Voranmeldung. Leistungen des alten Jahres, deren Rechnung noch fehlt,
   grenzt du ab (Schritt 5). Fertig, wenn jeder Monat festgeschrieben oder ohne
   blockierenden Befund ist.
3. **Inventar und Abstimmung.** Stelle das Inventar zum Stichtag auf und weise
   jedes Bestandskonto mit Saldo nach: Bank, offene Posten, Darlehen, Steuer-
   und Lohnkonten, Gesellschafterkonten, Anlagen. Fertig, wenn jeder Saldo
   einen Nachweis hat oder als Rückfrage offen ist.
   [abschlussbuchungen.md](abschlussbuchungen.md#1-inventar-und-abstimmung).
4. **Anlagen.** `anlagen` zeigt Verzeichnis, fällige AfA und
   Aktivierungskandidaten. Aktiviere offene Kandidaten, dann
   `abschlussbuchung` mit `art: afa_lauf`. Prüfe, ob eine außerplanmäßige
   Abschreibung oder Wertaufholung nötig ist, und leg sie dem Menschen vor.
   Fertig, wenn der Prüflauf kein `depreciation_missing` mehr meldet.
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
   [steuern-im-abschluss.md](steuern-im-abschluss.md#1-umsatzsteuer-verrechnung).
10. **Steuerrückstellung.** Erst wenn alle übrigen Buchungen stehen:
    Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer des Jahres aus
    den Erklärungen ([steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
    [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)), abzüglich
    Vorauszahlungen des Jahres; `abschlussbuchung` mit
    `art: steuerrueckstellung`. Fertig, wenn die Rückstellung gebucht ist und
    der Rechenweg in `grund` steht.
    [steuern-im-abschluss.md](steuern-im-abschluss.md#3-steuerrückstellung-betrag).
11. **UG-Rücklage** (nur UG). `abschlussbuchung` mit `art: ug_ruecklage`
    rechnet und bucht sie; bei der UG schon vor der Festschreibung des
    Dezembers. Fertig, wenn 7765 an 2930 gebucht ist.
    [ergebnisverwendung-ug.md](ergebnisverwendung-ug.md).
12. **Prüfen.** `auswertung` (`art: pruefbericht`, dann `bilanz`, `guv`).
    Bilanz geht auf, keine blockierenden Befunde, Abweichungen zum Vorjahr
    erklärt. Prüfe die
    [Warnzeichen](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest)
    (Hälfte des Stammkapitals verloren, nicht durch Eigenkapital gedeckter
    Fehlbetrag). Fertig, wenn der Mensch Bilanz und GuV mit deinen
    Erläuterungen gesehen hat.
13. **Anhang.** Entwirf die Angaben nach Größenklasse. Fertig, wenn der Mensch
    einen Entwurf hat.
    [groessenklassen-anhang.md](groessenklassen-anhang.md).
14. **Aufstellen (nur App).** Die Geschäftsführung unterzeichnet mit Datum und
    setzt den Stand `aufgestellt`. Danach schreibt der Mensch das Jahr fest;
    Buchfink verlangt das vor der Feststellung, und danach nimmt das Jahr auch
    in der Abschlussperiode keine Buchung mehr an. Fertig, wenn `status` den
    Stand und `mandant.festgeschrieben_bis` das Jahresende zeigt.
    [aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#2-aufstellen).
15. **Feststellen.** Die Geschäftsführung legt vor; die Gesellschafter
    beschließen Feststellung und Ergebnisverwendung. Lege den Beschluss mit
    `beleg_ablegen` ab. Der Mensch setzt `festgestellt` in der App. Fertig, wenn
    `status` den Stand zeigt.
    [aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#3-feststellen).
16. **Folgejahr.** Der Mensch führt den Saldenvortrag in der App aus (noch
    kein Werkzeug) und wiederholt ihn nach späten Abschlussbuchungen; Buchfink
    meldet Differenzen.
    Du buchst die Ergebnisverwendung nach dem Ja mit
    `ergebnisverwendung_buchen` und arbeitest die Liste Abschlussposten ab.
    Fertig, wenn der Vortrag ohne Differenz steht und die Liste leer ist.
    [ergebnisverwendung-ug.md](ergebnisverwendung-ug.md#4-buchen-im-folgejahr).
17. **Offenlegen.** Der Mensch übermittelt an das Unternehmensregister und
    setzt `offengelegt` mit dem Beleg aus „Meine Daten“ der Plattform.
    Registrierung, Identifizierung und Upload:
    [grundlagen/pflichten.md#offenlegung](../grundlagen/pflichten.md#offenlegung).
    Fertig, wenn `status` den Stand zeigt.
    [aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#4-offenlegen).

## 4. Entscheidungsregeln

- Wertaufhellung: Risiken, die am Stichtag bestanden und bis zur Aufstellung
  bekannt werden, gehören in den Abschluss; Gewinne erst, wenn sie am Stichtag
  realisiert sind ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]).
- Jede Rückstellung braucht einen Zweck aus dem Gesetz
  ([rueckstellungen.md](rueckstellungen.md#1-wann-eine-rückstellung-zu-bilden-ist)).
- Bewertungsmethoden des Vorjahres beibehalten ([§ 252 Abs. 1 Nr. 6 HGB][hgb252]).
  Abweichungen nur in begründeten Ausnahmefällen (Abs. 2), mit Anhangangabe
  ([groessenklassen-anhang.md](groessenklassen-anhang.md#4-anhang-einer-kleinen-gmbh)).
- Buchfink führt eine Bilanz für Handels- und Steuerrecht
  ([Grenzen](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
  Steuerliche Abweichungen (Abzinsung, Drohverlust, Sonderabschreibung) buchst
  du nicht; sie kommen in die
  [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten).
- Wertungsfragen, die der Mensch entscheidet: Höhe einer Rückstellung,
  Nutzungsdauer, dauernde Wertminderung, Ausfallrisiko einer Forderung,
  Fertigstellungsgrad und Gemeinkosten, Ansatzwahlrechte, Wesentlichkeit
  kleiner Beträge. Nenne Grundlage, Optionen und Wirkung auf Ergebnis und
  Steuer ([Rückfrage](../../SKILL.md#3-den-menschen-fragen)).
- Mittelgroße oder große Gesellschaft, Konzern, Abschlussprüfung: anhalten,
  [Grenzen](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet).

## 5. Nur der Mensch

Neben den Vorbehalten aus der
[Liste in SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app) (Aufstellen,
Jahr festschreiben, Feststellen, Offenlegen, Wiedereröffnen):

- Arbeitnehmerzahl und Stammdaten des Geschäftsjahres in der App.
- Den Abschluss unterzeichnen.
- Gesellschafterbeschlüsse fassen und unterschreiben.
- Saldenbestätigungen bei Banken, Kunden, Lieferanten anfordern.

Ohne Werkzeug und noch nicht abgebildet: [Abschnitt 2](#2-weg-der-abschlussbuchungen).

## 6. Stolperfallen

- Meldet eine Abschlussbuchung `fachliche_regel` mit festgeschriebenem
  Zeitraum, ist das Jahr schon festgeschrieben. Halte an und frag den
  Menschen. Buche nie ins neue Jahr, um auszuweichen.
- Vor jeder Abschlussbuchung prüfst du, ob es sie schon gibt; ändern heißt
  stornieren und neu buchen ([abschlussbuchungen.md](abschlussbuchungen.md#2-abschlussbuchung-mit-abschlussbuchung)).
- Die Steuerrückstellung kommt zuletzt; ein zweiter Rechengang ist unnötig
  ([steuern-im-abschluss.md](steuern-im-abschluss.md#3-steuerrückstellung-betrag)).
- Bei einer UG verlangt Buchfink die Rücklage schon für die Festschreibung des
  Dezembers ([ergebnisverwendung-ug.md](ergebnisverwendung-ug.md#2-rücklage-und-festschreibung-in-buchfink)).
- Buchungen in ein aufgestelltes oder festgestelltes Jahr:
  [Geschäftsjahr und Zeiträume](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume).
- Die Eröffnungsbilanz muss der Schlussbilanz entsprechen
  ([§ 252 Abs. 1 Nr. 1 HGB][hgb252]). Korrigiere nie über Vortragskonten.

[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
