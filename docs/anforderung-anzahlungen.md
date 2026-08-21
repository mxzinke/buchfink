# Buchfink – Anzahlungen & Rechnungsverbund

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-21
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft.

## 1. Warum das kein normaler Rechnungsfall ist

Bei Anzahlungen entsteht die Umsatzsteuer **mit der Vereinnahmung des Entgelts**, nicht
mit der Leistung (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG). Das gilt auch bei
Sollversteuerung und ist die Ausnahme, die den Fall vom übrigen Flow trennt: überall
sonst entsteht die Steuer mit der Rechnung, hier mit dem Geldeingang.

Der zweite Unterschied ist die Schlussrechnung. Sie muss die bereits vereinnahmten
Teilentgelte **und die darauf entfallenden Steuerbeträge absetzen** (§ 14 Abs. 5 Satz 2
UStG). Wer das vergisst, weist die Steuer zweimal aus – einmal in der
Anzahlungsrechnung, einmal in der Schlussrechnung – und **schuldet den Mehrbetrag nach
§ 14c Abs. 1 UStG**, auch wenn er ihn nie erhalten hat. Das ist der teuerste Fehler in
diesem ganzen Themenkomplex, und er entsteht durch bloßes Weglassen.

Daraus folgt die zentrale Anforderung: **die Schlussrechnung darf nicht ohne die
Verrechnung der Anzahlungen erstellbar sein.** Nicht als Warnung, sondern als
Konstruktion.

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

Erhaltene Anzahlungen auf Bestellungen dürfen offen von den Vorräten abgesetzt werden
(§ 268 Abs. 5 Satz 2 HGB) statt unter den Verbindlichkeiten zu stehen. Das ist ein
Wahlrecht mit sichtbarer Wirkung auf die Bilanzsumme und muss einmal je Mandant
entschieden und dann durchgehalten werden – ein Wechsel wäre ein Stetigkeitsbruch nach
§ 246 Abs. 3 HGB.

Da Buchfink Warenbestand in v1 nicht führt, ist der Verbindlichkeitsausweis der
naheliegende Standard. Die Entscheidung gehört trotzdem in die Stammdaten, damit sie
dokumentiert ist.

### Anzahlungsrechnung als Dokument

Eine Anzahlungsrechnung ist eine vollwertige Rechnung nach § 14 UStG: eigene
fortlaufende Nummer, alle Pflichtangaben, ausgewiesene Steuer. Sie ist kein Vorabbeleg
und keine Proforma-Rechnung. Sie läuft also durch denselben Nummernkreis wie jede
andere Ausgangsrechnung.

Der Unterschied liegt nur darin, **wann** gebucht wird: bei der normalen Rechnung beim
Ausstellen, bei der Anzahlungsrechnung beim Zahlungseingang.

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
- Die Schlussrechnung setzt **alle** vereinnahmten Anzahlungen des Verbunds ab, nicht
  nur die vom Nutzer ausgewählten.
- Ein Verbund mit Schlussrechnung ist abgeschlossen und nimmt keine weiteren
  Abschläge auf.
- Eine stornierte Anzahlungsrechnung fällt aus der Verrechnung heraus.

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
  den Vorräten (§ 268 Abs. 5 HGB)? Vorschlag: Verbindlichkeit, weil Vorräte in v1
  ohnehin nicht geführt werden.
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
