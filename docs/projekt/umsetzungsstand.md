# Umsetzungsstand

[Dokumentation](../README.md) · [Roadmap](roadmap.md) ·
[Umfang und Grenzen](../nutzung/umfang.md)

Stand: Version 0.3, Vorschau. Diese Seite beschreibt, was gebaut ist und wo es
im Code steht. Sie ist keine Freigabe für den produktiven Einsatz. Welche Fälle
Buchfink fachlich abdeckt, steht unter [Umfang und Grenzen](../nutzung/umfang.md),
einzelne Kriterien im [Anforderungskatalog](../anforderungen/README.md).

## Aufbau

| Baustein | Stand | Fundstelle |
|---|---|---|
| Kern | Ein offener Mandant mit einem Satz Dienste, Schreibsperre je Mandant, eine Transaktion je Vorgang, Prüfermodus in der Mandantendatenbank, Erfassungsweg `ui`, `system` oder `agent:<kurzname>` an jedem Vorgang, Vorgänge nur für die App, Sperre des Datenordners | `internal/core` (`Host`, `Tenant.Read`, `Tenant.Write`, `AppOnly`) |
| MCP-Server | Streamable HTTP auf `127.0.0.1:7465`, Port in der App änderbar, eigener Autorisierungsserver mit OAuth 2.1, PKCE und dynamischer Registrierung, Zustimmung nur im Fenster der App, Liste verbundener Apps mit Entzug, Idempotenzschlüssel für schreibende Werkzeuge, 41 Werkzeuge in drei Freigabestufen | `internal/mcpserver`, `internal/mcpserver/oauth`, [MCP-Server](../entwicklung/mcp.md) |
| Brücke für Claude Desktop | stdio-Programm ohne Wails, meldet sich selbst per OAuth an, als Erweiterung `.mcpb` | `cmd/buchfink-mcp`, `packaging/claude-desktop` |
| Oberfläche | fünf Bereiche: Übersicht, Bücher, Abschlüsse, Unternehmen, Einstellungen; Einrichtung in zwei Schritten (Unternehmen mit Ablage und Eingang, KI-App verbinden); Zustimmungsdialog für Apps; Dateien per Drag & Drop in den Eingang; die offene Ansicht lädt nach jeder Änderung im Mandanten nach | `frontend/src/pages/`, `frontend/src/components/SetupAssistantScreen.tsx`, `frontend/src/live.tsx`, [Gestaltung](../gestaltung/README.md) |

## Fachfunktionen

| Vorgang | Stand | Fundstelle |
|---|---|---|
| Eingang | ein Eingang je Mandant, im Datenordner oder außerhalb; Liste mit Art jeder Datei; nach der Ablage nach `Übernommen/<JJJJ-MM>/`; weitere Leseordner und Exportordner; Warnung, wenn die Ablage in einem Sync-Ordner liegt | `internal/inbox`, `internal/core/policy`, `internal/syncdir`, `wailsbridge.AttachFileDrop` |
| Belege | Ablage im Original unter SHA-256, Kopfdaten, Pflichtangaben je Belegart, Eigenbeleg, Beleg-Hash in der Buchung | `service.ReceiptService`, `ReceiptService.CreateSelfIssued`, `internal/receiptstore` |
| E-Rechnung empfangen | ZUGFeRD, Factur-X und XRechnung in CII und UBL lesen, gegen EN 16931 und XRechnung prüfen, Buchungsvorschlag | `internal/einvoice`, `internal/invoice/reader.go`, [Modulaufbau](../entwicklung/e-rechnung.md) |
| Buchen und korrigieren | ein Schreibweg ins Journal, Steuerzeilen aus dem Steuerfall, Konten nach Rechtsform, Storno per Generalumkehr | `service.JournalService.Post`, `service.PostingService`, `accounting.SKR04TaxResolver` |
| Zahlungswege | CAMT.053-Import, Zuordnung zu offenen Posten mit Teilzahlung und Skonto, Aufteilung eines Umsatzes, Firmenkreditkarte (3610), Auslage, Gesellschafterdarlehen | `internal/bank`, `service.PaymentService`, `service/bank_split.go`, `service.ShareholderService` |
| Lohnjournal | Monatssummen eines externen Lohnbüros buchen und abstimmen | `service.PayrollService` |
| Ausgangsrechnungen | Nummer, Buchung und Dokument in einer Transaktion, ZUGFeRD und XRechnung (CII), Storno und Berichtigung, Anzahlungen mit Schlussrechnung, Verzugszinsen | `service.InvoiceService`, `service/advance_service.go`, `service/overdue_service.go`, `internal/invoice` |
| Umsatzsteuer | Voranmeldung nach Vordruck USt 1 A 2026 mit Drill-down, Dauerfristverlängerung, Sondervorauszahlung, Zusammenfassende Meldung, Mindest-Istversteuerung, § 13b nach Absätzen, Vorsteuerberichtigung § 15a, USt-IdNr.-Bestätigung | `accounting/ustva.go`, `accounting/vat_prepayment.go`, `service.VatReturnService`, `service.ZMService`, `internal/vatid` |
| Anlagen | Kartei, AfA linear und degressiv, GWG, Sammelposten, § 7g, außerplanmäßige Abschreibung und Wertaufholung, Abgang | `service.AssetService`, `accounting/afa.go` |
| Monatsabschluss | Prüflauf, Festschreibung mit Zeitstempel, Voranmeldung als Eingabehilfe | `service.CheckService`, `core.Tenant.CommitPeriod`, `service.MonthCloseService` |
| Jahresabschluss | 15 Bausteine in fester Reihenfolge, Abschlussperiode zwischen Dezember- und Jahresfestschreibung, Wertberichtigung auf Forderungen, unfertige Leistungen, Steuerrückstellung mit Hebesatz, Verlustvortrag und KSt-Satz nach § 23 KStG, UG-Rücklage, Statuskette aufgestellt, festgestellt, offengelegt, Saldenvortrag | `domain.AllClosingSteps`, `service.ClosingStepsService`, `service/closing_tax_provision.go`, `service/closing_receivables.go`, `service/closing_work_in_progress.go` |
| Bilanz und GuV | Gliederung nach §§ 266, 275 HGB mit Vorjahr, Größenklasse, Anhangangaben | `service.StatementService`, `accounting/statement.go`, `accounting/groessenklasse.go` |
| Steuererklärungen | KSt 1 mit Anlagen, GewSt 1 A und USt 2 A formularnah mit Herkunft je Wert, Eingaben mit Bestätigung, Export als JSON (`buchfink.erklaerung/1`), Übermittlung mit Transferticket | `internal/declaration`, `service.DeclarationService`, `core.Tenant.ExportDeclaration` |
| E-Bilanz | XBRL nach Taxonomie 6.9 und 6.10 mit Kontennachweis, Überleitung, Eröffnungsbilanz, Befunden vor dem Export; Summen- und Saldenliste mit Taxonomie-Position je Konto für eBilanz+ | `internal/ebilanz`, `ebilanz.SuSaCSV`, `service.EBilanzService` |
| Pflichtenkalender | 21 Pflichtarten mit Rechtsgrundlage, Frist und Nachweis; Fristen nach steuerlicher Beratung, Dauerfristverlängerung, Wochenende und Feiertagen des Bundeslands; Stand aus den Daten oder bestätigt in der App | `service/obligation_catalog.go`, `service/obligation_rules.go`, `service.ObligationService`, `internal/calendar` |
| Gründung | Zeichnung, Einforderung und Einzahlung des Stammkapitals, Halbeinzahlung, Eröffnungsbilanz | `service.FoundationService`, `service/foundation_capital.go`, `service/opening_balance.go` |
| Nachweise | Hash-Ketten für Journal und Protokoll, Änderungsprotokoll mit Vorher und Nachher, Datenüberlassung (Z3), Belegarchiv, Prüferpaket, Verfahrensdokumentation | `accounting/journalhash.go`, `accounting/audithash.go`, `service.ExportService`, `internal/export`, `internal/procdoc` |
| Sicherung | ZIP mit Prüfsummen, automatisch bei eingerichtetem Ziel, Wiederherstellungstest, Wiederherstellung mit Prüfung | `service.BackupService`, `core.Host.RunDueBackup`, `core.Host.RestoreBackup` |

## Bekannte Grenzen

### Erklärungen und Übermittlung

- Buchfink übermittelt nichts. Voranmeldung, Zusammenfassende Meldung und
  Jahreserklärungen sind Eingabehilfen; die Übermittlung bestätigt der Mensch
  in der App mit Transferticket.
- Der Zeilenkatalog der Jahreserklärungen liegt für das Vordruckjahr 2025 vor.
  Für andere Jahre nimmt Buchfink den nächstälteren Katalog und meldet das.
- Bei USt 2 A ist jede Zeile am amtlichen Vordruck geprüft. Bei KSt und GewSt
  ist es keine, alle tragen `am_vordruck_geprueft: false`; bei mehr als der
  Hälfte ist auch die Zeilennummer nicht belegt. Der Prüfvermerk jeder Zeile
  nennt ihre Quelle.
- Die E-Bilanz übermittelt der Mensch über einen E-Bilanz-Dienst wie
  ebilanzplus.de oder ebilanzonline.de. Die Instanzen prüft
  `task test:ebilanz-xsd` gegen die amtlichen Schemata 6.9 und 6.10. Ob ein
  Dienst die XBRL-Datei oder die Saldenliste annimmt und die Finanzverwaltung
  sie akzeptiert, ist nicht belegt. Die Saldenliste füllt im Dienst nur Bilanz
  und GuV.
- Offenlegung und Hinterlegung erfasst Buchfink als Stand mit Nachweis. Einen
  Datensatz für das Unternehmensregister erzeugt es nicht.

### Laufende Buchhaltung

- Bankimport nur für Euro-Konten und CAMT.053. Sammeleinträge mit mehreren
  Einzeltransaktionen weist Buchfink ab.
- XRechnungen stellt Buchfink im CII-Format aus. UBL liest es, schreibt es aber
  nicht.
- Gutschrift, Korrektur und Anzahlungsrechnung eines Lieferanten erkennt
  Buchfink beim Empfang einer E-Rechnung. Einen Buchungsvorschlag aus dem
  Datensatz macht es nur für gewöhnliche Rechnungen.
- Die Grenzen bei Rechtsform, Steuerfällen, Warenverkehr, Kasse und Lohn
  stehen unter [Umfang und Grenzen](../nutzung/umfang.md).

### Betrieb

- Releases sind nicht signiert.
- Keine Verschlüsselung und keine Benutzerverwaltung, siehe
  [Datensicherheit](../nutzung/datensicherheit.md).
- Der MCP-Server läuft nur, solange die App geöffnet ist. Ein belegter Port
  hält die App nicht auf; der Server bleibt dann aus, bis in den Einstellungen
  ein freier Port gewählt ist.
- Buchfink beobachtet den Eingang nicht. Was dort liegt, legt der Agent auf
  Anweisung ab.
- Die sichtbare Ablagezone beim Ziehen von Dateien ist nur mit den
  DOM-Ereignissen geprüft, die Wails auch unter Windows nutzt. Unter macOS und
  Linux schaltet die Laufzeit sie nativ ein; das ist ungeprüft. Dateien nimmt
  das ganze Fenster auf jedem System an.

## Was die Prüfungen aussagen

Automatische Tests belegen einzelne Abläufe, Rechenregeln und die Sperren des
Kerns, auch gegen Zugriffe eines Agenten. Sie belegen weder vollständige
Gesetzeskonformität noch, dass Menschen ohne Vorkenntnisse jeden Schritt
verstehen. Eine externe fachliche Prüfung hat nicht stattgefunden. Die
[Prüfszenarien](../entwicklung/pruefszenarien.md) beschreiben, was sich
wiederholbar prüfen lässt. Behobene Fehler je Version stehen in der
[Änderungshistorie](../../internal/changelog/CHANGELOG.md).
