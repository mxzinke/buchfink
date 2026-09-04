# Buchfink – Anzahlungen & Rechnungsverbund

Gesetzliche Grundlage: [Anforderungskatalog](anforderungskatalog.md), UST-02,
RECH-02, RECH-03, RECH-10, BEL-07

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-22
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Die gesetzlichen Anforderungen samt Fundstellen stehen im
> Anforderungskatalog unter UST-02, RECH-02, RECH-03 und RECH-10.

## 1. Warum das kein normaler Rechnungsfall ist

Die gesetzlichen Anforderungen stehen im Anforderungskatalog: die Entstehung der
Steuer mit der Vereinnahmung unter UST-02, die Anzahlungsrechnung und ihre
Pflichtangaben unter RECH-02 und RECH-03, die Folge eines doppelten Steuerausweises
unter RECH-10.

Zwei Punkte trennen den Fall vom übrigen Flow. Erstens entsteht die Steuer mit dem
Geldeingang statt mit der Rechnung – auch bei Sollversteuerung. Zweitens muss die
Schlussrechnung die bereits vereinnahmten Teilentgelte und die darauf entfallenden
Steuerbeträge absetzen, aber nur, soweit über sie Anzahlungsrechnungen ausgestellt
wurden. Die Absetzungspflicht hängt also an der **ausgestellten
Anzahlungsrechnung**, nicht am Geldeingang: wer eine Anzahlung ohne Rechnung
vereinnahmt hat, muss in der Endrechnung nichts absetzen, weil dort auch nichts
doppelt ausgewiesen wurde. Wer sie dagegen vergisst, obwohl eine
Anzahlungsrechnung existiert, weist die Steuer zweimal aus und schuldet den
Mehrbetrag – der teuerste Fehler dieses Themenkomplexes, und er entsteht durch
bloßes Weglassen.

Daraus folgt die zentrale Anforderung: **die Schlussrechnung darf nicht ohne die
Verrechnung der berechneten Anzahlungen erstellbar sein.** Nicht als Warnung,
sondern als Konstruktion. Der Verbund muss dafür zwei Dinge auseinanderhalten, die
in der Praxis gern zusammenfallen: *vereinnahmt* (Geld ist da) und *berechnet*
(eine Anzahlungsrechnung wurde ausgestellt). Nur die Schnittmenge ist
absetzungspflichtig.

## 2. Erhaltene Anzahlungen (Ausgangsseite)

### Ablauf

| Schritt | Vorgang | Buchung |
|---|---|---|
| 1 | Anzahlungsrechnung stellen | *keine* – die Steuer entsteht erst mit der Zahlung |
| 2 | Zahlung geht ein | SOLL **1800** Bank · HABEN **3272** Erhaltene, versteuerte Anzahlungen 19 % USt + HABEN **3806** Umsatzsteuer 19 % |
| 3 | Leistung erbracht, Schlussrechnung | SOLL **Debitorenkonto** (Restbetrag) + SOLL **3272** + SOLL **3806** · HABEN **4400** Erlöse (Gesamtbetrag netto) + HABEN **3806** Umsatzsteuer (auf den Gesamtbetrag) |

Schritt 3 sieht umständlich aus, ist aber genau die geforderte Absetzung: die
Anzahlung wird aufgelöst, die darauf entfallende Steuer zurückgenommen, und die
Gesamtleistung wird in voller Höhe als Erlös erfasst.

### Konten

| Zweck | SKR04 |
|---|---|
| Erhaltene, versteuerte Anzahlungen 19 % USt | **3272** |
| Erhaltene, versteuerte Anzahlungen 7 % USt | **3260** |
| Erhaltene Anzahlungen auf Bestellungen (Verbindlichkeiten) | **3250** |
| Erhaltene Anzahlungen, offen von den Vorräten abgesetzt | **1190** |
| Restlaufzeitgliederung | **3280** / **3284** / **3285** |

### Bilanzausweis

Erhaltene Anzahlungen auf Bestellungen sind unter den Verbindlichkeiten gesondert
auszuweisen, „soweit Anzahlungen auf Vorräte nicht von dem Posten ‚Vorräte' offen
abgesetzt werden" (§ 268 Abs. 5 Satz 2 HGB). Das Wahlrecht betrifft also nur
Anzahlungen auf **Vorräte**; für alles andere bleibt es beim
Verbindlichkeitsausweis. Es hat sichtbare Wirkung auf die Bilanzsumme und muss
einmal je Mandant entschieden und dann durchgehalten werden – ein Wechsel wäre ein
Bruch der Darstellungsstetigkeit nach § 265 Abs. 1 HGB.

Da Buchfink Warenbestand in v1 nicht führt, ist der Verbindlichkeitsausweis der
naheliegende Standard. Die Entscheidung gehört trotzdem in die Stammdaten, damit sie
dokumentiert ist.

### Anzahlungsrechnung als Dokument

Eine Anzahlungsrechnung ist eine vollwertige Rechnung nach § 14 UStG: eigene
fortlaufende Nummer, alle Pflichtangaben, ausgewiesene Steuer. § 14 Abs. 5 Satz 1
UStG stellt das ausdrücklich klar – für vereinnahmte Teilentgelte „gelten die
Absätze 1 bis 4 sinngemäß". Sie ist also kein Vorabbeleg und keine
Proforma-Rechnung und läuft durch denselben Nummernkreis wie jede andere
Ausgangsrechnung.

Eine Pflichtangabe verschiebt sich allerdings: statt des Leistungszeitpunkts ist
der **Zeitpunkt der Vereinnahmung** anzugeben, sofern er feststeht und nicht mit
dem Rechnungsdatum übereinstimmt (§ 14 Abs. 4 Nr. 6 UStG). Bei einer
Anzahlungsrechnung, die vor dem Geldeingang gestellt wird, steht er noch nicht
fest – dann entfällt die Angabe. Buchfink muss das Feld also optional führen und
darf es nicht wie bei der normalen Rechnung erzwingen.

Der zweite Unterschied liegt darin, **wann** gebucht wird: bei der normalen
Rechnung beim Ausstellen, bei der Anzahlungsrechnung beim Zahlungseingang.

## 3. Geleistete Anzahlungen (Eingangsseite)

Spiegelbildlich, aber mit einem Unterschied: die Vorsteuer ist bereits mit der Zahlung
abziehbar, sofern eine ordnungsgemäße Rechnung vorliegt.

| Schritt | Buchung |
|---|---|
| Anzahlung geleistet | SOLL Anzahlungskonto + SOLL **1406** Vorsteuer · HABEN **1800** Bank |
| Schlussrechnung | SOLL Aufwand/Anlage (Gesamt) + SOLL **1406** (Rest) · HABEN Anzahlungskonto + HABEN **1406** (Rückbuchung) + HABEN **Kreditorenkonto** (Rest) |

Das Anzahlungskonto richtet sich nach der Bilanzposition, für die angezahlt wurde:

| Zweck | SKR04 |
|---|---|
| Geleistete Anzahlungen auf Vorräte | **1180** |
| Geleistete Anzahlungen auf immaterielle Vermögensgegenstände | **0170** |
| Geleistete Anzahlungen und Anlagen im Bau | **0700** |
| Anzahlungen auf andere Anlagen, BGA | **0795** |

Das ist kein Detail: eine Anzahlung auf eine Maschine gehört ins Anlagevermögen, eine
auf Handelsware ins Umlaufvermögen. Die Software muss also wissen, **wofür** angezahlt
wird, und kann das nicht aus dem Betrag ableiten.

## 4. Rechnungsverbund

Der Verbund ist ein eigener Entity, nicht nur eine Referenz zwischen Rechnungen. Er
hält zusammen, was fachlich ein Vorgang ist:

- den vereinbarten **Gesamtbetrag** des Auftrags,
- die **Abschlagsrechnungen** mit ihrem jeweiligen Zahlungsstand,
- die **Schlussrechnung** mit der Verrechnung,
- den **Fortschritt**: wie viel ist vereinbart, abgerechnet, vereinnahmt, offen.

Ohne den Verbund lässt sich die zentrale Frage nicht beantworten – „welche Anzahlungen
muss diese Schlussrechnung absetzen?" –, und genau daran hängt § 14c.

### Invarianten

- Die Summe der Abschlagsrechnungen darf den Gesamtbetrag nicht überschreiten.
- Die Schlussrechnung setzt **alle** berechneten und vereinnahmten Anzahlungen des
  Verbunds ab, nicht nur die vom Nutzer ausgewählten.
- Ein Verbund mit Schlussrechnung ist abgeschlossen und nimmt keine weiteren
  Abschläge auf.
- Eine stornierte Anzahlungsrechnung fällt aus der Verrechnung heraus – mit ihr
  entfällt die Rechnung im Sinne des § 14 Abs. 5 Satz 2 UStG und damit der Grund
  für die Absetzung.

### Was der offene Posten anzeigt

Eine Anzahlungsrechnung erzeugt einen offenen Posten wie jede andere Rechnung – aber
erst mit ihrer Bezahlung entsteht die Buchung. Das passt nicht zur bestehenden
OPOS-Logik, die den offenen Posten aus der Buchung ableitet. Hier ist eine
Designentscheidung nötig (siehe unten).

## 5. Offene Entscheidungen

- **Wie entsteht der offene Posten einer Anzahlungsrechnung?** Bisher gilt: kein
  offener Posten ohne Buchung. Bei Anzahlungen wird aber erst mit der Zahlung gebucht.
  Zwei Wege: entweder erzeugt die Anzahlungsrechnung eine Merkposten-Buchung, oder die
  OPOS-Liste bekommt eine zweite Quelle. Ersteres bleibt bei einer Wahrheit, braucht
  aber ein statistisches Konto.
- **Bilanzausweis erhaltener Anzahlungen:** Verbindlichkeit oder offene Absetzung von
  den Vorräten (§ 268 Abs. 5 Satz 2 HGB)? Vorschlag: Verbindlichkeit – die offene
  Absetzung steht ohnehin nur für Anzahlungen auf Vorräte offen, und Vorräte
  führt v1 nicht.
- **Anzahlungskonto ableiten oder abfragen?** Die Bilanzposition, für die angezahlt
  wird, ist eine Nutzerangabe. Kann sie aus der fachlichen Gruppe des Auftrags folgen?
- **Teilschlussrechnungen** bei langlaufenden Aufträgen: in v1 abbilden oder auf
  Abschläge plus eine Schlussrechnung beschränken?
- **Zahlungsplan:** soll der Verbund die Abschläge vorplanen (Termine, Prozentsätze)
  oder nur nachhalten, was tatsächlich gestellt wurde?

## 6. Abhängigkeiten

- Der **Rechnungsflow** bekommt einen zweiten Buchungszeitpunkt (Zahlung statt
  Ausstellung) und muss das im Modell tragen.
- Die **OPOS-Logik** muss den Anzahlungsfall abbilden, ohne ihre Ableitung aus dem
  Journal aufzugeben.
- **ZUGFeRD** kennt eigene Dokumenttypen für Anzahlungs- und Schlussrechnung; die
  Verrechnung gehört ins XML.
- Die **Anlagenverwaltung** braucht geleistete Anzahlungen für Anlagen im Bau.
- Die **E-Rechnung** ([anforderung-e-rechnung.md](anforderung-e-rechnung.md)) gilt
  auch für Anzahlungsrechnungen im B2B-Inland – § 14 Abs. 5 Satz 1 UStG verweist
  auf die Absätze 1 bis 4 und damit auf die Formvorschrift des Abs. 1.

## 7. Anmerkungen zu den Fundstellen

Die Normen und ihre Fundstellen stehen im Anforderungskatalog unter UST-02,
RECH-02, RECH-03 und RECH-10. Zwei Punkte trägt der Katalog nicht:

**Präzisierung zur Entstehung der Steuer:** § 13 Abs. 1 Nr. 1 Buchst. a Satz 4
UStG lässt die Steuer nicht taggenau mit dem Geldeingang entstehen, sondern „mit
Ablauf des Voranmeldungszeitraums, in dem das Entgelt oder das Teilentgelt
vereinnahmt worden ist". Für die Kontierung macht das keinen Unterschied – die
Buchung trägt das Zahlungsdatum und fällt damit in den richtigen Zeitraum –, für
den Text der UStVA-Auswertung aber schon.

**Korrektur gegenüber einer früheren Fassung dieses Dokuments:** dort war der
Stetigkeitsbruch beim Wechsel des Anzahlungs-Ausweises auf § 246 Abs. 3 HGB
gestützt. Der regelt die **Ansatz**stetigkeit; Ausweis und Gliederung sind eine
Darstellungsfrage und stehen in § 265 Abs. 1 HGB. Ebenso fehlte die Bedingung des
§ 14 Abs. 5 Satz 2 UStG, und das Absetzungswahlrecht des § 268 Abs. 5 Satz 2 HGB
war zu weit beschrieben – es gilt nur für Anzahlungen auf Vorräte.
