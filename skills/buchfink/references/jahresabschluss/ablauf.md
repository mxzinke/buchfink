# Ablauf: Jahresabschluss

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Der Jahresabschluss der GmbH besteht aus Bilanz, Gewinn- und Verlustrechnung
und Anhang ([§ 242 Abs. 3][hgb242], [§ 264 Abs. 1 Satz 1 HGB][hgb264]). Du
stimmst ab, bereitest die Abschlussbuchungen vor, prüfst die Zahlen und
entwirfst den Anhang.

## Inhalt

1. Pflichten und Fristen
2. Weg der Abschlussbuchungen
3. Ablauf
4. Entscheidungsregeln
5. Nur der Mensch

## 1. Pflichten und Fristen

`fristen` nennt Fälligkeit und Stand jedes Schritts; Frist, Weg, Nachweis und
Folgen stehen im Pflichtenkalender:

| Schritt | Wer | Pflicht |
|---|---|---|
| Inventar und Abschlussbuchungen | du, nach Ja | [bilanzstichtag](../grundlagen/pflichten.md#bilanzstichtag) |
| Aufstellen | Geschäftsführung | [aufstellung](../grundlagen/pflichten.md#aufstellung) |
| Feststellen | Gesellschafter | [feststellung](../grundlagen/pflichten.md#feststellung) |
| Ergebnisverwendung beschließen; du buchst sie | Gesellschafter | [ergebnisverwendung](../grundlagen/pflichten.md#ergebnisverwendung) |
| Offenlegen | Geschäftsführung | [offenlegung](../grundlagen/pflichten.md#offenlegung) |

Die Fristen gelten auch mit Steuerberater. Erstellt er den Abschluss, gib ihm
Saldenliste, Nachweise und die Liste Abschlussposten.

## 2. Weg der Abschlussbuchungen

Abschlussbuchungen haben den Stichtag als Buchungsdatum; Buchfink nimmt sie
an, bis das Jahr festgeschrieben ist. `abschluss_status` nennt je Schritt das
Werkzeug oder `nur_app`. Vorgehen, Umbuchungen und die Liste
**Abschlussposten**: [abschlussbuchungen.md](abschlussbuchungen.md).

## 3. Ablauf

1. **Vorbereiten.** `status`, `abschluss_status`; Größenklasse nach
   [groessenklassen-anhang.md](groessenklassen-anhang.md#1-merkmale). Fertig,
   wenn die Größenklasse feststeht und du die Fälligkeiten aus `fristen`
   genannt hast.
2. **Jahr vollständig** nach dem
   [Monatsabschluss](../monatsabschluss/ablauf.md). Fertig, wenn jeder Monat festgeschrieben oder ohne
   blockierenden Befund ist.
3. **Inventar und Abstimmung**
   ([abschlussbuchungen.md](abschlussbuchungen.md#1-inventar-und-abstimmung)).
   Fertig, wenn jeder Saldo einen Nachweis hat oder als Rückfrage offen ist.
4. **Anlagen** ([abschreibungen.md](abschreibungen.md); Finanzanlagen:
   [beteiligungen/bewertung-veraeusserung.md](../beteiligungen/bewertung-veraeusserung.md)).
   Fertig, wenn der Prüflauf kein `depreciation_missing` meldet und jede
   außerplanmäßige Abschreibung oder Wertaufholung dem Menschen vorliegt.
5. **Abgrenzen** ([abgrenzung.md](abgrenzung.md)). Fertig, wenn jede Zahlung
   über den Stichtag und jede Leistung ohne Rechnung geprüft ist.
6. **Rückstellungen** ([rueckstellungen.md](rueckstellungen.md)). Fertig, wenn
   jede Rückstellung mit Grund, Berechnung und Restlaufzeit gebucht oder
   verworfen ist.
7. **Forderungen und Fremdwährung** ([forderungen.md](forderungen.md)).
   Fertig, wenn jede überfällige Forderung bewertet ist.
8. **Unfertige Leistungen** ([unfertige-leistungen.md](unfertige-leistungen.md)).
   Fertig, wenn jedes laufende Projekt bewertet oder als unwesentlich vermerkt
   ist.
9. **Umsatzsteuer-Verrechnung** nach der letzten Voranmeldung
   ([steuern-im-abschluss.md](steuern-im-abschluss.md#1-umsatzsteuer-verrechnung)).
   Fertig, wenn die Umsatzsteuer- und Vorsteuerkonten null zeigen.
10. **Steuerrückstellung**, erst wenn alle übrigen Buchungen stehen
    ([steuern-im-abschluss.md](steuern-im-abschluss.md#3-steuerrückstellung-betrag)).
    Fertig, wenn sie gebucht ist und der Rechenweg in `grund` steht.
11. **UG-Rücklage** (nur UG, schon vor der Festschreibung des Dezembers;
    [ergebnisverwendung-ug.md](ergebnisverwendung-ug.md)). Fertig, wenn 7765
    an 2930 gebucht ist.
12. **Prüfen.** `auswertung` (`art: pruefbericht`, `bilanz`, `guv`) und
    [Warnzeichen](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest).
    Fertig, wenn die Bilanz aufgeht, kein blockierender Befund bleibt und der
    Mensch Bilanz und GuV mit deinen Erläuterungen zum Vorjahr gesehen hat.
13. **Anhang** nach Größenklasse entwerfen
    ([groessenklassen-anhang.md](groessenklassen-anhang.md)). Fertig, wenn der
    Mensch einen Entwurf hat.
14. **Aufstellen (nur App).** Die Geschäftsführung unterzeichnet und setzt
    `aufgestellt`, dann schreibt der Mensch das Jahr fest
    ([aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#2-aufstellen)).
    Fertig, wenn `status` den Stand und `mandant.festgeschrieben_bis` das
    Jahresende zeigt.
15. **Feststellen.** Leg den Beschluss der Gesellschafter über Feststellung
    und Ergebnisverwendung mit `beleg_ablegen` (`art: sonstiges`) ab; der
    Mensch setzt `festgestellt`
    ([aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#3-feststellen)).
    Fertig, wenn `status` den Stand zeigt.
16. **Folgejahr.** Nach dem Saldenvortrag des Menschen (nach späten
    Abschlussbuchungen wiederholt) buchst du die Ergebnisverwendung mit
    `ergebnisverwendung_buchen`
    ([ergebnisverwendung-ug.md](ergebnisverwendung-ug.md#4-buchen-im-folgejahr))
    und arbeitest die Liste Abschlussposten ab. Fertig, wenn der Vortrag ohne
    Differenz steht und die Liste leer ist.
17. **Offenlegen.** Der Mensch übermittelt an das Unternehmensregister und
    setzt `offengelegt` mit dem Beleg aus „Meine Daten“ der Plattform
    ([aufstellen-feststellen-offenlegen.md](aufstellen-feststellen-offenlegen.md#4-offenlegen)).
    Fertig, wenn `status` den Stand zeigt.

## 4. Entscheidungsregeln

- **Wertaufhellung:** Risiken, die am Stichtag bestanden und bis zur
  Aufstellung bekannt werden, gehören in den Abschluss; Gewinne erst, wenn sie
  am Stichtag realisiert sind ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]).
- Rückstellungen nur für die Zwecke des § 249 HGB
  ([rueckstellungen.md](rueckstellungen.md#1-wann-eine-rückstellung-zu-bilden-ist)).
- **Stetigkeit:** Bewertungsmethoden des Vorjahres beibehalten; Abweichung nur
  in begründeten Ausnahmefällen, mit Anhangangabe
  ([§ 252 Abs. 1 Nr. 6, Abs. 2 HGB][hgb252];
  [groessenklassen-anhang.md](groessenklassen-anhang.md#4-anhang-einer-kleinen-gmbh)).
- **Bilanzidentität:** Die Eröffnungsbilanz entspricht der Schlussbilanz des
  Vorjahres ([§ 252 Abs. 1 Nr. 1 HGB][hgb252]). Korrigiere nie über
  Vortragskonten.
- Meldet eine Abschlussbuchung, dass der Zeitraum festgeschrieben ist, halte an
  und frag den Menschen. Buche nie ins neue Jahr, um auszuweichen.
- Buchfink führt eine Bilanz für Handels- und Steuerrecht
  ([Grenzen](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).
  Steuerliche Abweichungen (Abzinsung, Drohverlust, Sonderabschreibung) buchst
  du nicht; sie kommen in die
  [Liste Abschlussposten](abschlussbuchungen.md#4-liste-abschlussposten).
- Wertungsfragen (Höhe einer Rückstellung, Nutzungsdauer, dauernde
  Wertminderung, Ausfallrisiko, Gemeinkosten, Ansatzwahlrechte, Wesentlichkeit)
  entscheidet der Mensch; nenne Grundlage, Optionen und Wirkung auf Ergebnis
  und Steuer ([Rückfrage](../../SKILL.md#3-den-menschen-fragen)).
- Mittelgroße oder große Gesellschaft, Konzern, Abschlussprüfung: anhalten
  ([Grenzen](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet)).

## 5. Nur der Mensch

Neben den [Vorbehalten aus SKILL.md](../../SKILL.md#4-nur-der-mensch-in-der-app):
Arbeitnehmerzahl und Stammdaten des Geschäftsjahres eintragen, den Abschluss
unterzeichnen, Gesellschafterbeschlüsse fassen, Saldenbestätigungen anfordern,
den Saldenvortrag ausführen.

[hgb242]: https://www.gesetze-im-internet.de/hgb/__242.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb264]: https://www.gesetze-im-internet.de/hgb/__264.html
