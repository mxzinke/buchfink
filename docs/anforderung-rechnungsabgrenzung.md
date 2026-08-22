# Buchfink – Rechnungsabgrenzung

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-22
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind am **22.08.2026** gegen den
> Gesetzestext auf gesetze-im-internet.de verifiziert; die Fundstellen stehen in
> [Abschnitt 8](#8-quellen).

## 1. Worum es geht

Die Rechnungsabgrenzung sorgt dafür, dass Aufwand und Ertrag in dem Jahr stehen, zu dem
sie gehören – und nicht in dem, in dem das Geld geflossen ist. Die Versicherungsprämie,
die im Dezember für das kommende Jahr gezahlt wird, ist Aufwand des kommenden Jahres.

| Fall | § 250 HGB | Konto |
|---|---|---|
| **Aktive RAP** – Ausgabe vor dem Stichtag, Aufwand danach | Abs. 1 | **1900** Aktive Rechnungsabgrenzung |
| **Passive RAP** – Einnahme vor dem Stichtag, Ertrag danach | Abs. 2 | **3900** Passive Rechnungsabgrenzung |

Beispiel Versicherung, 1.200 € für zwölf Monate ab 1. Dezember:

| Schritt | Buchung |
|---|---|
| Zahlung im Dezember | SOLL **6400** Versicherungen 1.200,00 · HABEN **1800** Bank 1.200,00 |
| Abgrenzung zum 31.12. | SOLL **1900** ARAP 1.100,00 · HABEN **6400** Versicherungen 1.100,00 |
| Auflösung im Folgejahr | SOLL **6400** Versicherungen 1.100,00 · HABEN **1900** ARAP 1.100,00 |

Im alten Jahr bleibt ein Monat Aufwand stehen, elf Monate wandern ins neue.

## 2. Die Abgrenzung gilt nicht für alles

§ 250 HGB verlangt einen Rechnungsabgrenzungsposten nur, soweit die Ausgabe „Aufwand
für eine **bestimmte Zeit** nach diesem Tag" darstellt. Das ist die entscheidende
Einschränkung und der häufigste Fehler:

- **Zeitraumbezogen** – Miete, Versicherung, Wartungsvertrag, Lizenz, Zeitschriftenabo:
  echter RAP.
- **Nicht zeitraumbezogen** – eine Anzahlung auf eine Lieferung, ein Vorschuss, eine
  Kaution: **kein** RAP, sondern sonstige Forderung (**1300** Sonstige
  Vermögensgegenstände) bzw. sonstige Verbindlichkeit (**3500** Sonstige
  Verbindlichkeiten).

Die Software kann das nicht am Betrag erkennen, aber sie hat die Information bereits:
**der Leistungszeitraum steht an jeder Buchung.** Ein Leistungszeitraum, der über den
Bilanzstichtag hinausreicht, ist genau das Signal für einen abgrenzungspflichtigen
Sachverhalt. Nur ein Zeitraum – ein Zeitpunkt reicht nicht.

Das ist der Grund, warum die vier Datumsfelder von Anfang an erfasst werden, obwohl die
Abgrenzung selbst noch nicht gebucht wird: **die Information ist später nicht mehr
rekonstruierbar**, wenn sie jetzt nicht erfasst wird.

## 3. Was Buchfink daraus machen kann

Der Kern ist eine Auswertung, keine neue Erfassung:

1. Zum Bilanzstichtag alle Buchungen suchen, deren Leistungszeitraum über den Stichtag
   hinausreicht.
2. Je Buchung den abzugrenzenden Anteil taggenau oder monatsgenau errechnen.
3. Die Abgrenzungsbuchungen als Vorschlag mit Vorschau anzeigen.
4. Nach Freigabe buchen – und die **Auflösung im Folgejahr** gleich mit vormerken.

Punkt 4 ist der, den man leicht vergisst: eine gebildete Abgrenzung, die nie aufgelöst
wird, verfälscht das Folgejahr genauso wie die fehlende Abgrenzung das alte. Die
Auflösung gehört an die Buchung gekoppelt, nicht an das Gedächtnis des Nutzers.

Wie bei der AfA ist die **jährliche Festschreibung der Auslöser**: vor dem Sperren des
Jahres prüft Buchfink, ob abgrenzungspflichtige Sachverhalte offen sind.

## 4. Berechnung

**Monatsgenau oder taggenau?** Handelsrechtlich ist beides vertretbar, solange es
stetig angewandt wird. Taggenau ist genauer, monatsgenau ist üblicher und leichter
nachvollziehbar. Die Wahl gehört in die Stammdaten und muss dann durchgehalten werden.

Gerechnet wird auf ganze Cent, einmal gerundet – wie überall sonst im System. Bei
monatsgenauer Rechnung: abzugrenzender Anteil = Betrag × verbleibende Monate ÷
Gesamtmonate.

## 5. Wesentlichkeitsgrenze

Handelsrechtlich gibt es keine Bagatellgrenze; jeder abgrenzungspflichtige Sachverhalt
ist abzugrenzen. Steuerlich darf der Ansatz unterbleiben, wenn die einzelne Ausgabe oder
Einnahme den Betrag des § 6 Abs. 2 Satz 1 EStG nicht übersteigt – also 800 €
(§ 5 Abs. 5 Satz 2 EStG). Das ist ein Wahlrecht, keine Pflicht.

Praktisch braucht es trotzdem eine konfigurierbare Schwelle, unterhalb derer Buchfink
nicht vorschlägt – sonst produziert die Prüfung dreißig Vorschläge über je vier Euro und
wird weggeklickt. Der Wert gehört in die Stammdaten, die Entscheidung ins Protokoll.

## 6. Offene Entscheidungen

Das ist der kleinste der offenen Punkte; im Kern hängt alles an einer Frage.

- **Vorschlagen oder nur anzeigen?** Erzeugt Buchfink Abgrenzungsbuchungen auf Freigabe
  (wie bei der AfA), oder listet es nur die Sachverhalte auf und überlässt die Buchung
  dem Nutzer? *Vorschlag: erzeugen, mit Vorschau und Freigabe – konsistent zur AfA.*
- **Monatsgenau oder taggenau?** *Vorschlag: monatsgenau als Standard, konfigurierbar.*
- **Wesentlichkeitsgrenze:** Standardwert und ob sie überhaupt vorbelegt wird.
- **Auflösung:** automatisch zum 1. Januar des Folgejahres buchen, oder ebenfalls als
  Vorschlag beim ersten Öffnen des neuen Jahres?
- **Abgrenzung ohne Buchungsbezug:** Manche Abgrenzungen entstehen nicht aus einer
  einzelnen Buchung. Braucht es einen manuellen Weg, oder reicht der Vorschlagsweg?

## 7. Abhängigkeiten

- Der **Festschreibungs-Workflow** ruft die Prüfung vor der Jahressperre auf – dieselbe
  Stelle wie die AfA-Prüfung. Beide sollten dort gemeinsam erscheinen.
- Die Abgrenzung greift auf den **Leistungszeitraum** zu, der bereits an jeder Buchung
  steht. Es sind keine Modelländerungen am Journal nötig.
- Ein **Jahreswechsel-Ablauf** muss die Auflösungen ins neue Jahr tragen.

## 8. Quellen

Stand der Prüfung: 22.08.2026, Volltexte über gesetze-im-internet.de.

| Aussage im Dokument | Fundstelle | Link |
|---|---|---|
| Aktiver RAP: Ausgabe vor dem Stichtag, soweit Aufwand für eine bestimmte Zeit danach | § 250 Abs. 1 HGB | [hgb/__250.html](https://www.gesetze-im-internet.de/hgb/__250.html) |
| Passiver RAP: Einnahme vor dem Stichtag, soweit Ertrag für eine bestimmte Zeit danach | § 250 Abs. 2 HGB | dito |
| Wortlaut „für eine bestimmte Zeit nach diesem Tag" als Abgrenzungskriterium | § 250 Abs. 1 und 2 HGB | dito |
| Steuerliches Wahlrecht, den Ansatz bis zur Grenze des § 6 Abs. 2 Satz 1 EStG zu unterlassen | § 5 Abs. 5 Satz 2 EStG | [estg/__5.html](https://www.gesetze-im-internet.de/estg/__5.html) |
| Betragsgrenze dieses Wahlrechts: 800 € | § 6 Abs. 2 Satz 1 EStG (Verweisziel) | [estg/__6.html](https://www.gesetze-im-internet.de/estg/__6.html) |

**Zur Formulierung „Handelsrechtlich gibt es keine Bagatellgrenze":** das ist eine
Aussage über das Fehlen einer Norm, nicht über eine Norm. § 250 HGB kennt keine
Wertgrenze; in der Praxis wird der allgemeine Wesentlichkeitsgrundsatz bemüht,
der aber nicht kodifiziert ist. Das steuerliche Wahlrecht in § 5 Abs. 5 Satz 2
EStG ist demgegenüber ausdrücklich geregelt und gilt nur steuerlich.

**Monatsgenau oder taggenau** ist keine Gesetzesfrage. § 250 HGB sagt nur „für
eine bestimmte Zeit" und schreibt kein Verteilungsverfahren vor. Die Bindung an
eine einmal gewählte Methode folgt aus dem Stetigkeitsgebot des § 252 Abs. 1
Nr. 6 HGB ([hgb/__252.html](https://www.gesetze-im-internet.de/hgb/__252.html)),
das die Beibehaltung der Bewertungsmethoden verlangt.
