# Buchfink – Aufbau der E-Rechnungs-Module

Stand: 22.08.2026

## 1. Warum geschichtet

XRechnung und ZUGFeRD sind keine eigenen Formate. XRechnung ist eine **CIUS** –
eine Nutzungsspezifikation, die EN 16931 einengt: optionale Felder werden
verpflichtend, Codelisten werden gekürzt, Kombinationen verboten. ZUGFeRD ist
eine Reihe von **Profilen** über derselben Norm plus eine Zustellform, nämlich
das PDF/A-3 mit eingebettetem Datensatz.

Die Norm selbst ist so gebaut: ihre Geschäftsregeln stehen in einem abstrakten
Regelsatz über dem semantischen Modell, und CII und UBL liefern nur die
Bindungen. Diesem Aufbau zu folgen kostet nichts und trägt zweierlei:

- **Eine Rechnung wird gleich beurteilt, egal in welcher Schreibweise sie
  ankommt.** Nachgewiesen, nicht behauptet: jedes offizielle UBL-Beispiel wird
  durch das Modell nach CII geschrieben und bekommt dasselbe Urteil.
- **Die Schichten können nicht auseinanderlaufen.** Sie sehen dieselbe gelesene
  Rechnung wie die Normregeln, und der Kern beurteilt einen Beleg genau einmal,
  gleichgültig wie viele Schichten greifen.

## 2. Die Pakete

```
internal/einvoice              das semantische Modell und EN 16931
├── model.go, date.go,         alle Geschäftsbegriffe der Norm
│   decimal.go                 Beträge, wie das Dokument sie schreibt
├── cii_*.go                   CII lesen und schreiben
├── ubl_*.go                   UBL lesen
├── validate*.go               alle 223 Geschäftsregeln
├── pdf.go                     hybrides PDF, Datensatz herausholen
├── xrechnung/                 die deutsche CIUS (BR-DE-*)
├── zugferd/                   Profile und der Vertrag fürs hybride PDF
└── ruleset/                   wählt die Schichten aus, die das Dokument angibt
```

Der Aufruf, den ein Empfangspfad braucht:

```go
inv, err := einvoice.ParseAny(data)   // XML in beiden Syntaxen oder hybrides PDF
result := ruleset.Validate(inv)       // Norm plus alles, was das Dokument angibt
what := ruleset.Describe(inv)         // Syntax, Profil, Rechnungsart
```

Keine Abhängigkeit zum übrigen Buchfink. Das Modul bringt seine eigene
Zahlenschicht mit, weil Beträge in einer Rechnung nicht alle zweistellig sind:
Einzelpreise (BT-146) und Steuersätze (BT-119) tragen legitim mehr
Nachkommastellen, und ein Satz von 8,375 %, vorher auf zwei Stellen gerundet,
liegt je Position ein bis zwei Cent daneben.

## 3. Was jede Schicht prüft

| Schicht | Umfang | Nachgewiesen an |
|---|---|---|
| **EN 16931** | 223 von 223 Geschäftsregeln | 15 CII- und 20 UBL-Beispiele fehlerfrei; 191 Regeln gegen die Regelsuite der Norm bestätigt, die übrigen 32 über eigene Tests |
| **XRechnung** | 41 von 59 Regeln, mit den Schweregraden der Spezifikation | neun Testinstanzen von KoSIT – eine gültig, acht ungültig; alle neun Urteile stimmen überein |
| **ZUGFeRD** | Profilerkennung und Profiltreue | die offiziellen Beispiele nennen ausschließlich erkannte Profile |

Was **nicht** geprüft wird, steht abfragbar im Code: `einvoice.RulesUnchecked()`
und `xrechnung.UncheckedRules()` nennen jede Regel einzeln, letzteres mit
Begründung. Es gibt bewusst keinen Wert, der „vollständig geprüft" bedeutet.

Die drei Gruppen, die bei XRechnung offen bleiben:

- **Die Extension** (BR-DEX-02, -03, -09 bis -15) betrifft Geschäftsbegriffe,
  die EN 16931 nicht kennt: Unterpositionen und Zahlungen durch Dritte. Sie
  stehen nicht im semantischen Modell.
- **Syntaxregeln** (BR-TMP-3 bis -5) begrenzen die Wiederholung von Elementen im
  XML, nicht im Modell.
- **Die Clean-Vehicles-Regeln** (BR-DE-CVD-*) gehören zu einem Vergabeszenario,
  das sich dem Dokument nicht ansehen lässt. Sie laufen über
  `xrechnung.WithCleanVehicles()`, wenn der Aufrufer weiß, dass es zutrifft.

## 4. Was das Modul nicht tut

**Es bucht nicht.** Aus einer Rechnung eine Buchung zu machen verlangt
Entscheidungen, die keine Formatfrage sind. Die wichtigste ist die Perspektive:
der Steuerkategoriecode steht aus Sicht des **Ausstellers** im Dokument. „K" ist
beim Lieferanten eine innergemeinschaftliche Lieferung und beim Empfänger ein
innergemeinschaftlicher Erwerb; „AE" wird zu § 13b. Wer den Code eins zu eins
übernimmt, bucht den halben Vorgang.

Ebenso wenig entscheidet es über den **Rechnungstyp** (BT-3). Das Modul sagt,
was das Dokument ist – `einvoice.KindCreditNote`, `KindCorrection`,
`KindPrepayment` –, aber was daraus folgt, ist eine Buchungsfrage.

## 5. Die Naht zum Buchungspfad (offen)

Heute hängt `internal/service/einvoice_service.go` noch am alten Prüfer in
`internal/invoice`. Dort liegen ein zweiter, kleinerer EN-16931-Prüfer und eine
eigene CII-Struktur; beide sind als abgelöst gekennzeichnet. **Neue Regeln
gehören ins Modul.**

Der zweite Schritt hängt sie um, und zwar über eine Schnittstelle, die der
Buchungspfad selbst besitzt – nicht über die Typen des Moduls:

```go
// internal/service — der Verbraucher beschreibt, was er braucht
type EInvoiceReader interface {
    Read(data []byte) (*IncomingInvoice, error)
}
```

Damit lässt sich der Buchungscode ohne das E-Rechnungsmodul testen: ein
Prüfling liefert eine `IncomingInvoice` von Hand, und die Buchungsregeln werden
gegen sie geprüft, ohne dass je ein XML entsteht. Umgekehrt bleibt das Modul
prüfbar, ohne dass ein Konto im Spiel ist.

Was dabei mitzunehmen ist:

- **BT-3 auswerten.** `Propose` liest den Rechnungstyp heute nicht. Eine
  Gutschrift (381) trägt positive Beträge und sagt nur dort, was sie ist – als
  gewöhnliche Rechnung gelesen dreht sie das Vorzeichen der Vorsteuer und
  eröffnet einen offenen Posten, wo einer zu schließen wäre.
- **BG-24 ablegen.** Eingebettete Unterlagen gehören als Belegdatei mit der
  Rolle `attachment` an den Beleg; die Rolle gibt es bereits.
- **BG-3 führen.** Der Rechnungsbezug ist die Klammer zwischen Korrektur und
  Original und zwischen Schlussrechnung und Anzahlungen.

## 6. Herkunft und Lizenzen

| Artefakt | Lizenz | Verwendung |
|---|---|---|
| ConnectingEurope/eInvoicing-EN16931 | EUPL-1.2 | nur Regelkennungen, Begriffsnummern und Codewerte – Tatsachen über die Norm. Die Dateien liegen nicht im Repository; `task test:en16931` holt sie. |
| itplr-kosit/xrechnung-schematron | Apache-2.0 | Regelkennungen, Schweregrade, Begriffsnummern |
| itplr-kosit/validator-configuration-xrechnung | Apache-2.0 | Testinstanzen und die zugesicherten Urteile; `task test:xrechnung` holt sie |
| ZUGFeRD/mustangproject | Apache-2.0 | die Profiltabelle, maschinell aus dem Referenz-Schematron abgeleitet |

Buchfink steht unter MIT. Regeltexte aus dem EUPL-Artefakt sind deshalb nicht
übernommen; die Meldungstexte sind durchgehend eigene.
