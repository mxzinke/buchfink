# Buchfink – Architektur und Bedienkonzept

Status: verbindlich für die Produktionsreife
Letzte Aktualisierung: 2026-09-04

Dieses Dokument beschreibt, wie Buchfink gebaut ist und wie es sich anfühlen
soll, wenn jemand ohne Buchhaltungskenntnis damit die Bücher einer kleinen
Kapitalgesellschaft führt. Es ergänzt zwei andere Dokumente: der
[Anforderungskatalog](anforderungskatalog.md) sagt, was das Gesetz verlangt und
wie weit Buchfink das erfüllt; das [Design-Konzept](design-konzept.md) sagt,
wie die Oberfläche aussieht. Hier steht, warum die Software so geschnitten ist
und welche Entscheidungen dahinterstehen.

---

## 1. Wer Buchfink bedient

Die Person am Rechner ist Gründerin oder Geschäftsführer einer UG oder GmbH.
Sie hat ein Bankkonto, stellt Rechnungen, bekommt Rechnungen, hat vielleicht
ein paar Anlagegüter und muss viermal im Jahr eine Umsatzsteuer-Voranmeldung
abgeben. Sie kennt die Wörter Soll und Haben, weiß aber nicht, was sie
bedeuten, und will das auch nicht lernen. Sie will drei Dinge: nichts
vergessen, nichts falsch machen, und am Jahresende einen Abschluss haben, den
ein Steuerberater oder das Finanzamt ohne Rückfrage annimmt.

Daraus folgt die Leitfrage für jede Funktion: **Was muss die Person heute
tun, und woran erkennt sie, dass es erledigt ist?** Buchhaltungsobjekte
(Konten, Buchungssätze, Steuerschlüssel) bleiben vorhanden und sichtbar, aber
sie sind die zweite Ebene. Die erste Ebene ist eine Aufgabenliste.

---

## 2. Grundentscheidungen

Die folgenden Entscheidungen sind bewusst getroffen und gelten bis auf
Widerruf. Jede hat eine Wirkung auf den Anforderungskatalog, die dort mit dem
Status `SCOPE` markiert ist.

| Entscheidung | Begründung | Folge |
|---|---|---|
| **Einzelplatz, ein Bearbeiter.** Buchfink läuft auf einem Rechner, ohne Benutzerverwaltung. | Die Zielgruppe hat keine Buchhaltungsabteilung. Rollen und Freigabestufen wären für eine Person Theater. | Jede Buchung trägt die Bearbeiterkennung (Betriebssystem-Benutzer und Rechnername). Funktionstrennung nach GoBD Rz 100 ff. wird in der Verfahrensdokumentation als „ein Bearbeiter, Kontrolle durch Steuerberater und Abschlussprüfung" beschrieben. Ein Prüfer bekommt den Datenträger (Z3), ergänzt um einen schreibgeschützten Prüfermodus (Z1). |
| **Local-First, Speicherort Inland.** Alle Daten liegen in einem Ordner auf dem Rechner der Anwenderin. | Kein Cloud-Zwang, keine Auftragsverarbeitung, keine Verlagerung nach § 146 Abs. 2a AO. | Sicherung und Wiederherstellung müssen Teil der Software sein. Der Speicherort wird in der Verfahrensdokumentation als Inland dokumentiert; wer den Ordner in eine ausländische Cloud synchronisiert, wird beim Einrichten darauf hingewiesen. |
| **Keine ERiC-Anbindung.** Buchfink übermittelt nichts selbst an die Finanzverwaltung. | ERiC ist eine proprietäre C-Bibliothek mit eigenen Lizenzbedingungen; ihre Einbindung würde den Build und die Lizenz des Projekts verändern. | Umsatzsteuer-Voranmeldung, Zusammenfassende Meldung und E-Bilanz entstehen als Datei bzw. als Kennziffernblatt zum Übertragen in Mein ELSTER. Das Übermittlungsprotokoll (Datum, Transferticket) wird nach der Übermittlung erfasst und ist danach unveränderlich. |
| **SKR04, Einheitsbilanz.** Ein Kontenrahmen, ein Wertansatz. | Kleine Kapitalgesellschaften stellen in der Praxis eine Einheitsbilanz auf. Zwei Bewertungskreise verdoppeln jede Erfassungsmaske. | Wo das Steuerrecht zwingend abweicht (Sonderabschreibung § 7g EStG), führt Buchfink den steuerlichen Wert am Anlagegut mit und erzeugt daraus das Verzeichnis nach § 5 Abs. 1 S. 2 EStG und die Überleitungsrechnung. Latente Steuern (§ 274 HGB) entfallen für kleine Gesellschaften nach § 274a HGB; ab mittelgroß verweist Buchfink an den Steuerberater. |
| **Steuerfälle sind eine geschlossene Liste.** | Jeder Steuerfall, den die Software kennt, muss vollständig richtig sein: Buchung, Rechnungstext, Voranmeldung, Meldung. Ein halb unterstützter Fall ist gefährlicher als ein fehlender. | Unterstützt: Inland 19 %, 7 %, 0 %, steuerfrei, innergemeinschaftlicher Erwerb, innergemeinschaftliche Lieferung, Reverse Charge als Empfänger (§ 13b Abs. 2 Nr. 1 UStG) und als Leistender (§ 3a Abs. 2 UStG), Ausfuhr. Ausgeschlossen: Kleinunternehmer, Differenzbesteuerung, Reiseleistungen, OSS/IOSS, Konsignationslager, Dreiecksgeschäft, Bauleistungen nach § 13b Abs. 2 Nr. 4 UStG. Die Oberfläche sagt bei einem ausgeschlossenen Fall, dass Buchfink ihn nicht abbildet. |
| **Keine Kasse, kein Lager, kein Lohn.** | Bargeschäft löst die KassenSichV aus, Lager braucht Inventur, Lohn ist ein eigenes Rechtsgebiet. | Das Kassenkonto 1600 bleibt bebuchbar für Auslagen und Verauslagungen, ein Kassenbuch gibt es nicht. Vorräte werden zum Stichtag als Inventurwert erfasst und als Bestandsveränderung gebucht. Lohn kommt als Sammelbuchung aus dem Lohnjournal des Lohnbüros herein. |
| **Kapitalgesellschaften zuerst.** | Der Gründungsweg, die Kapitalaufbringung, die Größenklassen und die Offenlegung sind für UG, GmbH und AG gebaut. Personenhandelsgesellschaften brauchen Kapitalkonten je Gesellschafter, Entnahmen und Einlagen. | Die Rechtsformen KG, OHG und e.K. bleiben wählbar, tragen aber in der Oberfläche den Hinweis „Kapitalkonten und Entnahmen sind in dieser Fassung nicht abgebildet". |

---

## 3. Die Kette, an der alles hängt

Der Anforderungskatalog beschreibt die Kette vom Geschäftsvorfall bis zur
Offenlegung. In Buchfink hat jede Station eine Entität und einen Dienst.

```mermaid
flowchart LR
    A["Bankumsatz<br/><small>BankTransaction</small>"] --> D
    B["Beleg<br/><small>Receipt, SHA256 je Datei</small>"] --> D
    C["Ausgangsrechnung<br/><small>Invoice, ZUGFeRD</small>"] --> D
    D["Buchung<br/><small>JournalEntry, Hash-Chain</small>"] --> E["Konten<br/><small>Account, PositionID</small>"]
    E --> F["Bilanz und GuV<br/><small>§§ 266, 275 HGB</small>"]
    F --> G["E-Bilanz<br/><small>XBRL, Taxonomie als Ressource</small>"]
    F --> H["Offenlegung<br/><small>Unternehmensregister</small>"]
    D -. Festschreibung, RFC 3161 .-> I["Unveränderbarkeit"]
    B -. Fristenklasse, Löschsperre .-> J["Aufbewahrung"]
    E -. Z3-Export, Prüfermodus .-> K["Datenzugriff"]
```

Was heute trägt und was in dieser Runde dazukommt, steht je Station im
Anforderungskatalog. Die Architekturentscheidung ist, **dass jede Station aus
den Buchungen abgeleitet wird und nichts daneben erfasst wird**: die Bilanz
liest Kontensalden, die E-Bilanz liest die Bilanz, die Voranmeldung liest die
Steuerzeilen der Buchungen. Es gibt keine zweite Datenquelle für eine Zahl.

---

## 4. Schichten

```mermaid
flowchart TB
    subgraph Frontend["Frontend (React, TypeScript)"]
        P["pages/*"] --> API["services/api.ts"]
    end
    API --> BR["wailsbridge<br/>Aufrufbare Oberfläche, Mandant, Geschäftsjahr"]
    subgraph Backend["Backend (Go)"]
        BR --> SVC["service<br/>Anwendungsfälle: Buchen, Zahlen, Abschließen, Exportieren"]
        SVC --> ACC["accounting<br/>Regeln: SKR04, Steuerschlüssel, AfA, Hash"]
        SVC --> DOM["domain<br/>Entitäten, Invarianten, Repository-Schnittstellen"]
        SVC --> REP["repository<br/>GORM, SQLite, Feldverschlüsselung"]
        SVC --> FMT["Formatmodule<br/>einvoice, invoice, ebilanz, bank, export"]
    end
    REP --> DB[("buchfink.sqlite je Mandant")]
    FMT --> FS[("belege/, dokumente/, export/")]
```

Regeln, die diese Schichtung stützen:

1. **Ein Schreibweg ins Journal.** `service.JournalService.Post` ist die einzige
   Stelle, an der eine Buchung entsteht. Alle Anwendungsfälle (Beleg buchen,
   Zahlung zuordnen, AfA-Lauf, Saldenvortrag, Abschlussbuchung) bauen einen
   `JournalEntry` und geben ihn dorthin. Dort sitzen Ausgleich, Kontenprüfung,
   Periodensperre, Nummernvergabe und Hash-Chain.
2. **Regeln ohne Datenbank.** Das Paket `accounting` rechnet: Steuerschlüssel,
   AfA-Pläne, Wertgrenzen, Größenklassen, Bilanzgliederung. Es kennt keine
   Repositories und ist vollständig durch Tabellentests abgedeckt.
3. **Zeitabhängige Parameter als datierte Regelsätze.** Wertgrenzen, AfA-Sätze,
   Größenklassen-Schwellen, Basiszinssatz und Aufbewahrungsfristen liegen in
   `accounting/tax_params.go` als Tabellen mit Gültigkeitsbeginn. Eine
   Gesetzesänderung ist eine neue Zeile, keine Codeänderung.
4. **Formatmodule sind austauschbar.** Export (Z3, DATEV, Archiv), E-Bilanz und
   E-Rechnung lesen ein neutrales Datenmodell und schreiben ein Format. Die
   künftige Buchführungsdatenschnittstelle nach § 147b AO wird ein weiteres
   Modul, kein Umbau.
5. **Die Bridge hält den Zustand, die Dienste sind zustandsarm.** Mandant und
   aktives Geschäftsjahr leben in `wailsbridge`; die Dienste bekommen das Jahr
   gesetzt und filtern danach.

---

## 5. Datenmodell: was dazukommt

Das bestehende Modell (Buchung, Beleg, Kontakt, Rechnung, Anlage,
Festschreibung, Audit-Log) bleibt. Für die Produktionsreife kommen wenige
Entitäten dazu, jede mit einem klaren Grund.

```mermaid
erDiagram
    FiscalYear ||--o{ JournalEntry : "enthält"
    FiscalYear ||--o| FiscalYear : "Vortrag aus Vorjahr"
    FiscalYear ||--o{ Festschreibung : "wird festgeschrieben"
    FiscalYear ||--o| ClosingStatus : "Entwurf, aufgestellt, festgestellt, offengelegt"
    JournalEntry ||--o{ JournalLine : "Soll/Haben"
    JournalEntry }o--o| Receipt : "Belegverweis"
    Receipt ||--o{ ReceiptFile : "Original, Struktur, Darstellung"
    Receipt ||--|| RetentionClass : "6, 8 oder 10 Jahre"
    Accrual ||--o{ JournalEntry : "Bildung und Auflösung"
    Provision ||--o{ JournalEntry : "Bildung, Verbrauch, Auflösung"
    VatReturn ||--o{ JournalEntry : "Kennziffern aus Steuerzeilen"
    VatReturn ||--o| SubmissionRecord : "Transferticket"
    AuditLogEntry }o--|| AuditLogEntry : "Vorgänger-Hash"
    Check ||--o{ Finding : "Prüfbericht"
```

| Entität | Zweck | Katalog |
|---|---|---|
| **FiscalYear** | Das Geschäftsjahr wird eine Entität mit Beginn, Ende, Rumpfjahr-Kennzeichen, Vortragsstand und Abschlussstatus. Bisher war es nur ein Filterfeld an der Buchung. | GOB-06, JAB-04, JAB-09 |
| **Saldenvortrag** | Eigene Buchung mit Quelle `opening`, Gegenkonto 9000/9008/9009, wiederholbar durch Storno und Neuvortrag. Personenkonten werden je offenem Posten vorgetragen. | JAB-09, BEW-01 |
| **Abschlussbuchungen** | Erfolgskonten auf das Jahresergebnis, Umsatzsteuer-Verrechnung, Steuerrückstellung, Ergebnisverwendung. Alle mit Quelle `closing`, alle im Journal. | JAB-01, JAB-04 |
| **Accrual (Rechnungsabgrenzung)** | Start, Ende, Betrag, Konto; monatliche Auflösung als Buchung. | BEW-08 |
| **Provision (Rückstellung)** | Art, Erfüllungsbetrag, Restlaufzeit, Abzinsung aus einer pflegbaren Zinstabelle, Verbrauch und Auflösung mit Begründung. | BEW-07 |
| **VatReturn** | Ein Voranmeldungszeitraum mit allen Kennziffern, abgeleitet aus den Steuerzeilen, mit Übermittlungsprotokoll und Berichtigungskette. | UST-03, UST-01 |
| **RetentionClass am Beleg** | Fristenklasse aus der Belegart, Fristbeginn 31.12. des Entstehungsjahres, frühestes Löschdatum, Aufbewahrungs-Hold. | ARC-01, ARC-02 |
| **AuditLog mit Vorher/Nachher und Kette** | Stammdatenänderungen als Diff, jeder Eintrag verkettet, Bearbeiterkennung und Programmversion an jeder Buchung. | UNV-03, UNV-04, UNV-06 |
| **Check und Finding** | Ergebnisse der Prüfläufe (Plausibilität, Zeitgerechtigkeit, Doppelbelege) mit Zeitpunkt und Umfang, übergangene Befunde mit Begründung. | GOB-03, BEL-04, UNV-05 |

---

## 6. Bedienkonzept: der Jahreslauf als Rückgrat

Das Menü bleibt (Übersicht, Buchhaltung, Stammdaten, Auswertungen, Verwaltung).
Was sich ändert, ist die Übersicht: sie wird zur Aufgabenliste, und die beiden
Abschlussvorgänge werden geführte Wege.

```mermaid
flowchart LR
    T["Täglich<br/>Bankumsätze zuordnen<br/>Belege ablegen<br/>Rechnungen schreiben"] --> M
    M["Monatlich<br/>1 Prüfbericht<br/>2 Voranmeldung<br/>3 Festschreiben"] --> J
    J["Jährlich<br/>1 Prüfbericht<br/>2 AfA, Abgrenzung, Rückstellung<br/>3 Bilanz und GuV<br/>4 Aufstellen, feststellen<br/>5 E-Bilanz, Offenlegung<br/>6 Saldenvortrag"] --> T
```

Zwei Ansichten sind mit den Wellen 5b und 5c dazugekommen, weil ihr Vorgang
nicht in einen Dialog passt: „Anzahlungen" führt den Rechnungsverbund mit
Abschlägen, Vereinnahmung und Schlussrechnung, und „Nebenpflichten" bündelt
das Verzeichnis nach § 15a UStG, die USt-IdNr.-Bestätigungen, den Belegnachweis,
die Berichte zu nicht abziehbaren Betriebsausgaben und die Kurse. Die
Rechnungs- und Belegdialoge verweisen dorthin, statt die Vorgänge zu
verdoppeln.

### 6.1 Die Übersicht ist eine Aufgabenliste

Beim Start sieht die Anwenderin keine Kennzahlen, sondern was ansteht, in
dieser Reihenfolge:

1. **Überfällig.** Voranmeldung nicht bestätigt, Vormonat nicht
   festgeschrieben (GoBD-Frist Folgemonat), Belege älter als zehn Tage ohne
   Buchung, Rechnungen ohne Zahlung über Zahlungsziel.
2. **Offen.** Bankumsätze ohne Zuordnung, Belege ohne Buchung, Prüfbefunde
   aus dem letzten Prüflauf.
3. **Demnächst.** Fristen der nächsten dreißig Tage: Voranmeldung,
   Offenlegung, Aufstellung, ablaufende Dokumente am Anlagegut.

Jede Zeile hat drei Teile: was zu tun ist, warum (ein Satz mit der Norm hinter
dem Erklärzeichen) und einen Knopf, der an die richtige Stelle führt. Die
Kennzahlen (Bankguthaben, Ergebnis, offene Posten) rutschen darunter.

### 6.2 Monatsabschluss in drei Schritten

Der Monatsabschluss ist ein Dialog mit drei Schritten, von der Fristenseite
und von der Aufgabenliste aus erreichbar.

| Schritt | Was Buchfink tut | Was die Anwenderin tut |
|---|---|---|
| 1 Prüfbericht | Läuft die Plausibilitätsprüfung: unausgeglichene Konten, Buchungen ohne Beleg, Interimskonten, offene Bankumsätze, Doppelbelege. | Klärt Befunde oder übergeht sie mit Begründung. |
| 2 Festschreiben | Schließt den Monat ab, holt den Zeitstempel. Danach ändert sich das Kennziffernblatt nicht mehr. | Bestätigt. |
| 3 Voranmeldung | Zeigt alle Kennziffern des Vordrucks mit Drill-down bis zur Buchung, erzeugt das Kennziffernblatt als Datei. | Trägt die Werte in Mein ELSTER ein, erfasst danach Datum und Transferticket. |

Die Reihenfolge ist bewusst: Was gemeldet wird, muss vorher unveränderlich
sein, sonst weicht die Buchführung später von der Meldung ab. Ein Monat ohne
Voranmeldung (Zeitraum Quartal) endet nach Schritt 2; am Quartalsende folgt
Schritt 3 über die drei Monate.

### 6.3 Jahresabschluss als geführter Weg

Der Jahresabschluss ist die Stelle, an der ein Nicht-Buchhalter bisher
aussteigt. Buchfink führt in sechs Stationen, jede mit einer Vorschau der
Buchungen, die entstehen, und einem Zurück.

1. **Prüfbericht** wie im Monat, zusätzlich: Konten ohne Bilanzposition,
   Anlagen ohne AfA, offene Posten älter als ein Jahr (Wertberichtigung?).
2. **Abschlussbuchungen** in vorgegebener Reihenfolge, jede optional
   überspringbar mit Hinweis: AfA-Lauf, Rechnungsabgrenzung, Rückstellungen,
   Inventurwert der Vorräte, Umsatzsteuer-Verrechnung, Steuerrückstellung.
3. **Bilanz und GuV** nach §§ 266, 275 HGB in der Gliederung der
   Größenklasse, mit Vorjahresspalte, als Datei.
4. **Aufstellen und feststellen.** Statuswechsel mit Datum; der
   Feststellungsbeschluss wird als Dokument angehängt. Ab „festgestellt" ist
   das Jahr gesperrt.
5. **E-Bilanz und Offenlegung.** XBRL-Datei erzeugen, Übermittlung
   bestätigen; Offenlegungsumfang aus der Größenklasse, Frist und Nachweis.
6. **Saldenvortrag** ins neue Jahr, mit Ergebnisverwendung.

### 6.4 Sprache

Die Oberfläche spricht in Vorgängen, nicht in Konten. Ein Bankumsatz ist
„Geld erhalten" oder „Geld bezahlt", ein Beleg ist „Rechnung vom Lieferanten",
eine Abgrenzung ist „Kosten, die ins nächste Jahr gehören". Der Buchungssatz
mit Soll und Haben steht in der Vorschau jedes Vorgangs und im Journal, damit
der Steuerberater ihn sieht und die Anwenderin ihn lernen kann, wenn sie will.
Das Design-Konzept regelt die drei Stufen der Erklärung (Tooltip, Popover,
Dialog); jede gesetzliche Prüfung nennt in der zweiten Stufe die Norm.

### 6.5 Das Prüferpaket

Ein Knopf unter „Sicherheit & Protokoll" erzeugt in einem Ordner alles, was
eine Betriebsprüfung verlangt: den Z3-Export nach dem Beschreibungsstandard
(Datendateien plus `index.xml`), den Archivexport der Belege mit Index, das
Integritätsprotokoll, das Schlüsselverzeichnis und die
Verfahrensdokumentation in der Version, die zu den Daten gehört. Der
Prüfermodus schaltet die Oberfläche schreibgeschützt und protokolliert den
Zugriff.

### 6.6 Sicherung ohne Nachdenken

Buchfink sichert den Datenordner beim Beenden und einmal täglich in einen
gewählten Zielordner, protokolliert das Ergebnis und bietet beim Start eine
Wiederherstellung an, nach der die Integritätsprüfung läuft. Die Sicherung
ist eine ZIP-Datei mit Datenbank, Belegen, Dokumenten und Schlüsseldatei;
ihre Wiederherstellung ist ohne die Software möglich, weil das Format offen
ist.

---

## 7. Einordnung der offenen Anforderungen

Die Lückenanalyse im Anforderungskatalog ordnet jedes Kriterium einem Status
zu. Die offenen Punkte fallen in sieben Wellen. Die Reihenfolge folgt der
Frage, was einen Jahreslauf blockiert.

| Welle | Inhalt | Katalog | Warum in dieser Reihenfolge |
|---|---|---|---|
| 1 | Geschäftsjahr als Entität, Saldenvortrag, Abschluss der Erfolgskonten, Ergebnisverwendung, Abschlussstatus | GOB-06, JAB-04, JAB-09, BEW-01 | Ohne Vortrag endet die Buchhaltung nach einem Jahr. |
| 2 | Bilanz und GuV nach §§ 266, 275 HGB im Backend, Größenklassen, Vorjahresspalte, Ausgabe als Datei; E-Bilanz aus der Gliederung | JAB-01, JAB-02, JAB-03, JAB-05 | Erst mit der Gliederung gibt es eine Struktur, auf die E-Bilanz und Offenlegung zeigen. |
| 3 | Umsatzsteuer-Voranmeldung mit allen Kennziffern, Dauerfristverlängerung, Übermittlungsprotokoll, Zusammenfassende Meldung, Prüfberichte vor Festschreibung | UST-01, UST-03, UST-04, GOB-03, BEL-04, UNV-05 | Die Daten hängen an den Buchungen; die Meldung ist die Pflicht mit dem kürzesten Takt. |
| 4 | Z3-Export mit Beschreibungsstandard, Archivexport, Sicherung und Wiederherstellung, Prüfermodus | PRF-01, PRF-02, ARC-04, ARC-08 | Betriebsprüfung und Datenverlust sind die beiden Ereignisse, die eine Buchhaltung beenden. |
| 5a | Rechnungsabgrenzung, Rückstellungen, Inventurwert, Umsatzsteuer-Verrechnung, Steuerrückstellung, Verzeichnis nach § 5 EStG | BEW-07, BEW-08, BEW-09, JAB-06 | Für die Bilanz nicht verzichtbar, aber erst mit Welle 1 und 2 sinnvoll. |
| 5b | Rechnungsnummer in einer Transaktion, Pflichtangaben, XRechnung, Storno- und Korrekturbelege, Kleinbetrag, Anzahlungen als Rechnungsverbund, Ausbuchung | RECH-02 bis RECH-09, BEL-09, UST-02 | Die Ausgangsrechnung ist der häufigste Beleg; ihre Fehler wandern in jede Meldung. |
| 5c | Vorsteuerkopplung, Vorsteuerschlüssel, Verzeichnis nach § 15a UStG, USt-IdNr.-Bestätigung, Belegnachweis, Geschenke, Fremdwährung, Abschreibungsregeln als Ressource | UST-05 bis UST-07, RECH-07, BEW-03, BEW-10, BEW-12 | Nebenpflichten, die im laufenden Jahr anfallen und im Abschluss nicht mehr nachholbar sind. |
| 6 | Änderungsprotokoll mit Vorher/Nachher und Kette, Bearbeiterkennung, Programmversion je Buchung, Aufbewahrungsfristen und Holds, Verfahrensdokumentation | UNV-03, UNV-04, UNV-06, ARC-01, ARC-02, PRF-03 | Nachweispflichten, die ohne die ersten Wellen leer blieben. |
| 7 | Aufgabenliste, Monatsabschluss-Dialog, Jahresabschluss-Weg, Mahnwesen | Abschnitt 6, QUE-05 | Die Bedienung legt sich über die fertigen Funktionen. |

Welche Kriterien in welcher Welle liegen und was davon schon gebaut ist, steht
mit Fundstellen im [Anforderungskatalog](anforderungskatalog.md).
