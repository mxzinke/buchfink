# Buchfink – Aufbau der E-Rechnungs-Module

Gesetzliche Grundlage: [Anforderungskatalog](anforderungskatalog.md), RECH-06, RECH-07

Stand: 22.08.2026

Dieses Dokument beschreibt den Modulschnitt, nicht das Recht: was beim Erzeugen und
beim Empfang einer E-Rechnung zu prüfen ist, steht im Anforderungskatalog unter
RECH-06 und RECH-07.

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

## 5. Die Naht zum Buchungspfad

Der Buchungspfad kennt die Typen des Moduls nicht. Er besitzt eine eigene
Schnittstelle und ein eigenes Datenmodell:

```go
// internal/service — der Verbraucher beschreibt, was er braucht
type EInvoiceReader interface {
    Read(data []byte) (*domain.IncomingInvoice, error)
    ValidateOnly(data []byte) (domain.ReceiptValidation, error)
}
```

`domain.IncomingInvoice` trägt die gut zwölf Felder, aus denen eine Buchung
entsteht — nicht die 160 Geschäftsbegriffe der Norm. Übersetzt wird an einer
einzigen Stelle, `internal/invoice/reader.go`. Die Richtung der Abhängigkeiten:

```
internal/domain      ← kennt niemanden
internal/service     ← kennt domain und invoice, aber nicht einvoice
internal/invoice     ← der Adapter: kennt beides
internal/einvoice    ← kennt nur sich selbst
```

Was das einbringt, steht in `internal/service/einvoice_seam_test.go`: in dieser
Datei kommt kein XML vor. Der Buchungsvorschlag, die Umkehr des Steuerfalls,
die Zuordnung des Lieferanten, die Ablage der Anlagen — alles geprüft an einer
`IncomingInvoice`, die von Hand hingeschrieben ist. Umgekehrt kommt in
`internal/einvoice` kein Konto vor.

Die Umkehr der Perspektive steht in `internal/domain`, nicht im Leser: sie ist
eine steuerliche Entscheidung, keine Formatfrage. `"K"` heißt beim Lieferanten
innergemeinschaftliche Lieferung und bei uns innergemeinschaftlicher Erwerb —
wer den Code übernimmt, bucht den halben Vorgang. Das lässt sich jetzt prüfen,
ohne dass irgendwo ein Parser läuft.

### Was dabei behoben wurde

- **Der Rechnungstyp wird ausgewertet.** Eine Gutschrift trägt positive Beträge
  und sagt nur in BT-3, was sie ist. Bisher wurde sie als Eingangsrechnung
  vorgeschlagen — mit umgekehrtem Vorzeichen der Vorsteuer und einem neuen
  offenen Posten, wo einer zu schließen wäre. Jetzt wird der Vorschlag
  verweigert, und die Meldung nennt den Fall beim Namen. Dasselbe gilt für
  Korrektur, Anzahlungs- und Abschlagsrechnung sowie das Gutschriftverfahren.
- **Mitgeschickte Unterlagen landen am Beleg.** Was im Datensatz eingebettet
  ankommt (BG-24), wird als Belegdatei mit der Rolle `attachment` abgelegt. Ein
  Stundenzettel zur Rechnung ist Aufbewahrungsgegenstand wie die Rechnung.
- **Der Rechnungsbezug wird geführt.** BG-3 steht im Vorschlag und ist die
  Klammer zwischen Korrektur und Original.
- **Die Prüfung ist die aus dem Modul.** Der zweite, kleinere Prüfer in
  `internal/invoice` ist gelöscht; ein empfangener Beleg wird gegen alle 223
  Regeln der Norm gehalten, und bei einer XRechnung zusätzlich gegen die
  deutsche Ausprägung.

### Was offen bleibt

Gutschrift, Korrektur und Anzahlungsrechnung werden erkannt und benannt, aber
nicht gebucht. Das ist kein Versehen: jede von ihnen ist ein anderer
Geschäftsvorfall. Eine Gutschrift mindert Aufwand und Vorsteuer und verrechnet
sich gegen die ursprüngliche Rechnung; eine Anzahlungsrechnung wird erst mit
der Zahlung steuerwirksam und in der Schlussrechnung wieder abgesetzt
(§ 14 Abs. 5 UStG, siehe `anforderung-anzahlungen.md`). Sie brauchen einen
eigenen Buchungsweg, keinen Sonderfall im vorhandenen.

## 6. Herkunft und Lizenzen

| Artefakt | Lizenz | Verwendung |
|---|---|---|
| ConnectingEurope/eInvoicing-EN16931 | EUPL-1.2 | Regelkennungen, Begriffsnummern, Codewerte; Beispielrechnungen und Regeldateien liegen unter `internal/einvoice/testdata/` |
| itplr-kosit/xrechnung-schematron | Apache-2.0 | Regelkennungen, Schweregrade, Begriffsnummern |
| itplr-kosit/validator-configuration-xrechnung | Apache-2.0 | Testinstanzen und die zugesicherten Urteile, unter `internal/einvoice/xrechnung/testdata/` |
| ZUGFeRD/mustangproject | Apache-2.0 | die Profiltabelle, maschinell aus dem Referenz-Schematron abgeleitet |

Buchfink steht seit dem Wechsel selbst unter EUPL-1.2 – derselben Lizenz wie
das Validierungsartefakt. Das Prüfmaterial liegt deshalb im Repository, und die
Prüfungen laufen ohne Netz und ohne Vorbereitung. Die Meldungstexte sind
trotzdem durchgehend eigene: die Texte der Norm sind für einen Prüfbericht
geschrieben, nicht für jemanden, der entscheiden muss, ob er die Rechnung
verbucht.

Alle vier Quellen stehen mit Fundstelle und Lizenz in `THIRD-PARTY-NOTICES.md`
bzw. – für das Prüfmaterial – in `internal/einvoice/testdata/README.md`.
