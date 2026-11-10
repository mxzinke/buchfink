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
| MCP-Server | Streamable HTTP auf `127.0.0.1:7465`, Port in der App änderbar, eigener Autorisierungsserver mit OAuth 2.1, PKCE und dynamischer Registrierung, Zustimmung nur im Fenster der App, Liste verbundener Apps mit Entzug, Idempotenzschlüssel für schreibende Werkzeuge, 42 Werkzeuge in drei Freigabestufen | `internal/mcpserver`, `internal/mcpserver/oauth`, [MCP-Server](../entwicklung/mcp.md) |
| Brücke für Claude Desktop | stdio-Programm ohne Wails, meldet sich selbst per OAuth an, als Erweiterung `.mcpb` | `cmd/buchfink-mcp`, `packaging/claude-desktop` |
| Aktualisierung | Suche nach dem neuesten Release beim Start und täglich, abschaltbar; Installation auf einen Klick in der App mit Prüfsumme, Tausch im Installationsordner und Neustart | `internal/update`, `wailsbridge.InstallUpdate` |
| Oberfläche | fünf Bereiche: Übersicht, Bücher, Abschlüsse, Unternehmen, Einstellungen; Einrichtung in zwei Schritten (Unternehmen mit Ablage und Eingang, KI-App verbinden); Zustimmungsdialog für Apps; Dateien per Drag & Drop in den Eingang; die offene Ansicht lädt nach jeder Änderung im Mandanten nach | `frontend/src/pages/`, `frontend/src/components/SetupAssistantScreen.tsx`, `frontend/src/live.tsx`, [Gestaltung](../gestaltung/README.md) |

## Fachfunktionen

| Vorgang | Stand | Fundstelle |
|---|---|---|
| Eingang | ein Eingang je Mandant, im Datenordner oder außerhalb; Liste mit Art jeder Datei; nach der Ablage nach `Übernommen/<JJJJ-MM>/`; weitere Leseordner und Exportordner; Warnung, wenn die Ablage in einem Sync-Ordner liegt | `internal/inbox`, `internal/core/policy`, `internal/syncdir`, `wailsbridge.AttachFileDrop` |
| Belege | Ablage im Original unter SHA-256, Kopfdaten, Pflichtangaben je Belegart, Eigenbeleg, Beleg-Hash in der Buchung; Kleinbetragsrechnung und Fahrausweis mit Bruttobeträgen, Posten ohne Umsatzsteuer im selben Beleg, Gutschrift eines Lieferanten; ein Vertrag ohne eigene Buchung gehört zu seinem Datensatz, etwa einem Gesellschafterdarlehen | `service.ReceiptService`, `ReceiptService.CreateSelfIssued`, `service/posting_gross.go`, `ReceiptService.AssignToRecord`, `internal/receiptstore` |
| E-Rechnung empfangen | ZUGFeRD, Factur-X und XRechnung in CII und UBL lesen, gegen EN 16931 und XRechnung prüfen, Buchungsvorschlag für Rechnung und Gutschrift; eine reine XML ist ohne PDF buchbar, die App zeigt sie als Ansicht aus dem Datensatz | `internal/einvoice`, `internal/invoice/reader.go`, `EInvoiceService.View`, [Modulaufbau](../entwicklung/e-rechnung.md) |
| Buchen und korrigieren | ein Schreibweg ins Journal, Steuerzeilen aus dem Steuerfall, Konten nach Rechtsform, Storno per Generalumkehr | `service.JournalService.Post`, `service.PostingService`, `accounting.SKR04TaxResolver` |
| Zahlungswege | CAMT.053-Import, Zuordnung zu offenen Posten mit Teilzahlung, Skonto und Gutschriften, Aufteilung eines Umsatzes, Firmenkreditkarte (3610), Auslage, Gesellschafterdarlehen; Kursdifferenz beim Ausgleich eines Postens in Fremdwährung und bei der Kartenabrechnung auf 6880/4840 ohne eigene Angabe | `internal/bank`, `service.PaymentService`, `service/settlement_split.go`, `service/settlement_difference.go`, `service.ShareholderService` |
| Lohnjournal | Monatssummen eines externen Lohnbüros buchen und abstimmen | `service.PayrollService` |
| Ausgangsrechnungen | Stornorechnung und Berichtigung mit Vorschau in App und Werkzeug; Nummer, Buchung und Dokument in einer Transaktion, ZUGFeRD und XRechnung (CII), Abschlagsrechnungen, Verzugszinsen | `service.InvoiceService`, `service/advance_service.go`, `service/overdue_service.go`, `internal/invoice` |
| Umsatzsteuer | Voranmeldung nach Vordruck USt 1 A 2026 mit Drill-down, Dauerfristverlängerung, Sondervorauszahlung, Zusammenfassende Meldung, Mindest-Istversteuerung, § 13b nach Absätzen, Verzeichnis nach § 15a, USt-IdNr.-Bestätigung | `accounting/ustva.go`, `accounting/vat_prepayment.go`, `service.VatReturnService`, `service.ZMService`, `internal/vatid` |
| Anlagen | Beschreibende Stammdaten mit Grund und Änderungsprotokoll in der App pflegen; Stückzahlen nur vor Buchung und Folgebewegungen in offener Periode. Kartei, AfA linear und degressiv, GWG, Sammelposten, § 7g | `service.AssetService`, `accounting/afa.go` |
| Monatsabschluss | Prüflauf, Festschreibung mit Zeitstempel, Voranmeldung als Eingabehilfe | `service.CheckService`, `core.Tenant.CommitPeriod`, `service.MonthCloseService` |
| Jahresabschluss | 15 Bausteine in fester Reihenfolge, Abschlussperiode zwischen Dezember- und Jahresfestschreibung, Wertberichtigung auf Forderungen, unfertige Leistungen, Steuerrückstellung mit Hebesatz, Verlustvortrag und KSt-Satz nach § 23 KStG, UG-Rücklage, Statuskette aufgestellt, festgestellt, offengelegt, Saldenvortrag | `domain.AllClosingSteps`, `service.ClosingStepsService`, `service/closing_tax_provision.go`, `service/closing_receivables.go`, `service/closing_work_in_progress.go` |
| Bilanz und GuV | Gliederung nach §§ 266, 275 HGB mit Vorjahr, auch zu einem Stichtag im Jahr, Größenklasse, Anhangangaben | `service.StatementService`, `accounting/statement.go`, `accounting/groessenklasse.go` |
| Steuererklärungen | KSt 1 mit Anlagen, GewSt 1 A und USt 2 A formularnah mit Herkunft je Wert, Eingaben mit Bestätigung, Export als JSON (`buchfink.erklaerung/1`), Übermittlung mit Transferticket | `internal/declaration`, `service.DeclarationService`, `core.Tenant.ExportDeclaration` |
| E-Bilanz | XBRL nach Taxonomie 6.9 und 6.10 mit Kontennachweis, Überleitung, Befunden vor dem Export; Eröffnungsbilanz einer Gründung in Buchfink in der App und über `ebilanz_exportieren`; Summen- und Saldenliste mit Taxonomie-Position je Konto für eBilanz+ | `internal/ebilanz`, `ebilanz.SuSaCSV`, `service.EBilanzService` (`EBilanzOptions.Kind`) |
| Pflichtenkalender | 23 Pflichtarten mit Rechtsgrundlage, Frist und Nachweis, darunter Aufstellung und E-Bilanz der Eröffnungsbilanz einer Gründung; Fristen nach steuerlicher Beratung, Dauerfristverlängerung, Wochenende und Feiertagen des Bundeslands; Stand aus den Daten oder bestätigt in der App | `service/obligation_catalog.go`, `service/obligation_rules.go`, `service.ObligationService`, `internal/calendar` |
| Gründung | Zeichnung, Einforderung und Einzahlung des Stammkapitals, Halbeinzahlung, Eröffnungsbilanz als PDF zum Unterzeichnen in der App, auf denselben Stichtag wie ihre E-Bilanz | `service.FoundationService`, `service/foundation_capital.go`, `service/foundation_opening.go`, `service/opening_balance.go` |
| Nachweise | Hash-Ketten für Journal und Protokoll, Änderungsprotokoll mit Vorher und Nachher, Datenüberlassung (Z3), Belegarchiv, Prüferpaket, Verfahrensdokumentation | `accounting/journalhash.go`, `accounting/audithash.go`, `service.ExportService`, `internal/export`, `internal/procdoc` |
| Sicherung | ZIP mit Prüfsummen, automatisch bei eingerichtetem Ziel, Wiederherstellungstest, Wiederherstellung mit Prüfung | `service.BackupService`, `core.Host.RunDueBackup`, `core.Host.RestoreBackup` |

## Forderungen ausbuchen

Uneinbringliche Forderungen lassen sich unter Bücher › Konten › Offene Posten
und mit `forderung_ausbuchen` nach Rückfrage ganz oder teilweise ausbuchen.
Grund, Nachweis und Eintrittsdatum sind Pflicht. Die Vorschau zeigt die
Steuerkorrektur und die Auflösung einer zugeordneten Einzelwertberichtigung.
Ein geführter Ablauf für spätere Zahlung auf ausgebuchte Forderungen fehlt.
Einzelwertberichtigungen ohne Forderungszuordnung sperren die Ausbuchung;
die Zuordnung muss fachlich geklärt werden.

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
- Die Eröffnungsbilanz als PDF gibt nur die App aus, kein Werkzeug.
- Eine Ausschüttung per Handbuchung prüft Buchfink nicht gegen die
  Feststellung und rechnet keine Kapitalertragsteuer; eine zurückgenommene
  Feststellung lässt die gebuchte Ergebnisverwendung stehen. Beides ist
  gewollt; den richtigen Weg beschreibt das
  [Fachkonzept Jahresabschluss](../fachkonzepte/jahresabschluss.md#aufstellen-feststellen-offenlegen).
- Kapitalertragsteuer auf eine Ausschüttung rechnet Buchfink auf die ganze
  Ausschüttung und nennt zur Abführung den auf volle Euro abgerundeten Betrag
  ([§ 44 Abs. 1 Satz 6 EStG](https://www.gesetze-im-internet.de/estg/__44.html)).
  Ungeklärt und nicht gebaut: wem der Abrundungsrest auf 3760 nach der
  Abführung zusteht, ob die Abrundung über
  [§ 1 Abs. 2 SolZG](https://www.gesetze-im-internet.de/solzg_1995/__1.html)
  auch für den Solidaritätszuschlag gilt und ob je Gesellschafter statt auf
  die ganze Ausschüttung zu rechnen ist. Den Rest bucht Buchfink nicht aus;
  die Antworten gehören zur steuerlichen Beratung.

### Laufende Buchhaltung

- Bankimport nur für Euro-Konten und CAMT.053. Sammeleinträge mit mehreren
  Einzeltransaktionen weist Buchfink ab.
- XRechnungen stellt Buchfink im CII-Format aus. UBL liest es, schreibt es aber
  nicht.
- Rechnungskorrektur und Anzahlungsrechnung eines Lieferanten erkennt
  Buchfink beim Empfang einer E-Rechnung. Einen Buchungsvorschlag aus dem
  Datensatz macht es nur für Rechnung und Gutschrift.
- Die Grenzen bei Rechtsform, Steuerfällen, Warenverkehr, Kasse und Lohn
  stehen unter [Umfang und Grenzen](../nutzung/umfang.md).

### Betrieb

- Releases sind nicht signiert. Die Aktualisierung in der App prüft die Datei
  gegen die Prüfsumme des Release, die von derselben Stelle kommt; sie schützt
  vor einem beschädigten Download, nicht vor einem gefälschten Release.
- Die Aktualisierung ist mit Tests an Dateien belegt (Programmdatei, Paket für
  Linux, App-Paket). Tausch und Neustart auf macOS und Windows selbst sind
  ungeprüft. Für Linux auf ARM und Windows auf ARM gibt es keine Datei; dort
  führt der Hinweis zur Releases-Seite.
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

Die ZM-Ansicht erkennt Änderungen an bestätigten Meldungen, zeigt alte und neue
Beträge und legt Berichtigungen mit Kenntnisdatum und Monatsfrist an. Erstmeldung
und Berichtigung lassen sich als BOP-CSV exportieren; die Übermittlung bestätigt
weiterhin der Mensch. Ohne Kenntnisdatum bleibt die datierte Berichtigungsfrist
offen. Die Erzeugung aus Buchungen umfasst weiterhin sonstige Leistungen.


## Rückstellungen fortführen

App und `abschlussbuchung` bieten Zuführung, Auflösung und Aufzinsung mit
Vorschau. `abschluss_status` liefert Bestände und Spiegel über die Jahre.
Verbrauch ist beim Buchen einer Eingangsrechnung mit Rückstellungszuordnung möglich.

Grenzen: Die automatische Aufzinsung verwendet den gespeicherten Satz.
Neubewertung bei geändertem Marktzins, gebrochene Laufzeiten und steuerliche
Ausnahmen sind gesondert zu prüfen. Einzelheiten und Quellen:
[Rückstellungen fortführen](../fachkonzepte/jahresabschluss.md#rückstellungen-fortführen).
