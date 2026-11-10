# Prüfszenarien

[Dokumentation](../README.md) · [Entwicklung](README.md)

Diese Seite nennt wiederholbare Prüfwege und was sie belegen. Ergebnisse,
Aufnahmen und Exporte bleiben lokal unter `.cache/` und gehören nicht ins
Repository.

## Agent und Kern

Die Sperren des Kerns sind die wichtigste Zusage gegenüber einem Agenten. Sie
sind als Go-Tests geschrieben und laufen mit `task test`.

| Was | Test |
|---|---|
| Ein Agent kann nichts ausführen, was der App vorbehalten ist | `core.TestAgentsAreRefusedForHumanOnlyOperations`, `service.TestAgentsCannotTriggerAppOnlyOperations` |
| Ein Agent bucht, was sich per Storno korrigieren lässt, auf jedem Konto: Kapital, Rücklagen, Gesellschafterkonten, Darlehen, Ergebnisverwendung, Organisationstexte | `service.TestAgentsBookWhatAStornoCorrects`, `mcpserver.TestShareholderLoanTool`, `mcpserver.TestAppropriationTool` |
| Ein Agent liest nur aus Eingang und Leseordnern, nie aus Datenordnern und Sicherungen | `policy.TestReadFileStaysInsideInboxAndReadFolders`, `policy.TestReadFolderNeverReachesAnotherTenant` |
| Nach der Ablage geht die Datei aus dem Eingang nach `Übernommen/<JJJJ-MM>/`, auch aus der App | `mcpserver.TestFilingSettlesTheInbox`, `policy.TestSettleInboxMovesWhatIsArchived`, `wailsbridge.TestStatementImportSettlesTheInbox` |
| Jede Änderung erreicht das Fenster, gebündelt je Mandant | `core.TestWriteReportsEachChange`, `wailsbridge.TestChangesReachTheWindowBundled` |
| Ein Datenordner hat genau einen Schreiber; ein Mandantenwechsel wartet auf laufende Vorgänge | `internal/core/single_writer_test.go` |
| Prüfermodus und Festschreibung halten auch für wartende Schreiber und nach geänderten Stammdaten | `internal/core/period_lock_test.go` |
| Ein Agent teilt einen Bankumsatz genau einmal auf, auch bei Wiederholung | `core.TestAgentSplitsABankTransactionOnce` |
| Erklärungen: Eingaben je Erfassungsweg, Export und Übermittlung | `internal/core/declaration_test.go` |
| Anmeldung, Zustimmung, Tokens, Entzug | `internal/mcpserver/oauth/oauth_test.go` |
| Werkzeugrahmen: Mandant, Rechte, Idempotenz, Fehlercodes | `internal/mcpserver/tools_test.go`, `internal/mcpserver/server_test.go` |
| Brücke: App nicht geöffnet, Zustimmung abgelehnt, erneute Anmeldung | `cmd/buchfink-mcp/bridge_test.go` |

## Einen Client von Hand verbinden

1. Buchfink im Entwicklungsbetrieb starten, bei Bedarf auf einem eigenen Port:
   `BUCHFINK_MCP_PORT=7466 task dev`.
2. Einen Testmandanten mit erfundenen Daten in einem leeren Ordner anlegen.
3. Einen Client nach der [Anleitung](../nutzung/ki-agenten/README.md) mit
   `http://127.0.0.1:7466/mcp` verbinden.
4. Im Browser und im Fenster von Buchfink denselben Abgleichcode prüfen und
   zulassen.
5. Den Agenten `status` aufrufen lassen. Erwartet: `bereit: true` und der Name
   des Testmandanten.
6. Eine erfundene Rechnung ins Fenster von Buchfink ziehen und dem Agenten
   sagen: „Buche die Belege im Eingang.“ Erwartet: `eingang` listet die Datei,
   nach `beleg_ablegen` liegt sie unter `Eingang/Übernommen/<JJJJ-MM>/`, und
   die Übersicht zeigt den Beleg ohne Neuladen.
7. Einen Vorgang anstoßen, der der App vorbehalten ist, etwa die
   Festschreibung. Erwartet: kein Werkzeug dafür, oder der Fehler `nur_in_app`.
8. In den Einstellungen unter „Apps verbinden“ den Zugriff entziehen.
   Erwartet: der nächste Aufruf scheitert.

## Fachliche Abläufe

| Ablauf | Erwartung | Test |
|---|---|---|
| Teilzahlung und Skonto | Auf 1.190 € werden 500 € gezahlt, 690 € bleiben offen; Skonto mindert Entgelt und Umsatzsteuer | `internal/service/payment_service_test.go`, `internal/accounting/ustva_test.go` |
| Mindest-Istversteuerung | Ein im Januar bezahltes Jahresabo erscheint in der Voranmeldung für Januar | `internal/accounting/vat_prepayment_test.go` |
| Firmenkreditkarte, Auslage, Gesellschafterdarlehen, Aufteilung, Lohnjournal | Zahlungswege mit ihren Gegenkonten, Darlehen mit Zins, Lohnsummen abgestimmt | `internal/service/payment_ways_test.go`, `shareholder_service_test.go`, `bank_split_test.go`, `payroll_service_test.go` |
| Gründung mit Halbeinzahlung | Ausstehende Einlage und Einforderung richtig getrennt | `internal/service/foundation_capital_test.go` |
| Jahresabschluss | Wertberichtigung, unfertige Leistungen, Steuerrückstellung, UG-Rücklage | `internal/service/closing_receivables_test.go`, `closing_work_in_progress_test.go`, `closing_tax_provision_test.go`, `ug_reserve_test.go` |
| Erklärungen | KSt, GewSt und USt 2 A aus den Büchern, § 13b nach Absätzen | `internal/declaration/*_test.go`, `core.TestVatYearDeclarationSplitsReverseCharge` |
| E-Bilanz | Zuordnung der Konten, Befunde, Taxonomie nach Wirtschaftsjahr; die Saldenliste hat dieselben Konten und Salden wie die Instanz und summiert zu null | `internal/ebilanz/cases_test.go`, `version_test.go`, `susa_test.go`; Schema mit `task test:ebilanz-xsd` |
| Sync-Erkennung | iCloud Drive, OneDrive, Google Drive, Dropbox, Nextcloud/ownCloud und Syncthing nach Ort und Markerdatei | `internal/syncdir/syncdir_test.go` |
| Pflichtenkalender | Fristen mit Beratung, Dauerfristverlängerung und Feiertagen | `internal/service/obligation_service_test.go`, `internal/calendar/holidays_test.go` |
| Rechnungsdokumente | Pflichtangaben in PDF und XML, Bezüge bei Storno und Berichtigung | `internal/invoice/business_letters_test.go`, `cii_content_test.go` |
| Sicherung | Sicherung, Wiederherstellungstest, beschädigte Dateien werden erkannt | `internal/service/backup_service_test.go` |

## Testmandanten mit einem Agenten

Drei erfundene Unternehmen prüfen Buchfink mit einem verbundenen Agenten vom
Beleg bis zur Festschreibung. Der Generator unter
[`scripts/test-tenants/`](../../scripts/test-tenants/README.md) erzeugt
Belege, E-Rechnungen und Kontoauszüge. Er schreibt zu jedem Mandanten eine
README mit Stammdaten, erwarteten Buchungssätzen und Kontrollwerten.

| Mandant | Schwerpunkt | Zeitraum |
|---|---|---|
| 1 Nordlicht Digital GmbH | Gründung mit Halbeinzahlung und Einforderung, Auslagen, Foto eines Kassenbons, Ausgangsrechnungen, XRechnung an eine Behörde, § 13b, ausländische Umsatzsteuer | 03.08.–16.10.2026 |
| 2 Kranich Software GmbH | laufender Betrieb: Direktbuchung, reine XRechnung und UBL, Gutschrift, Dublette, geänderte Bankverbindung, Bewirtung, Geschenke, Skonto, Kreditkarte in USD, § 13b, innergemeinschaftlicher Erwerb, Lohn, Gesellschafterdarlehen, Dauerfristverlängerung | 01.06.–16.10.2026 |
| 3 Waldkauz Projekte UG | Projektgeschäft mit Abschlägen, Kundeninsolvenz, Mindest-Istversteuerung, Jahresabschluss mit Rückstellungen, UG-Rücklage und Ergebnisverwendung, Geschäftsjahr 01.10.–30.09. | 01.10.2025–16.10.2026 |

Ablauf je Mandant:

1. Testdaten erzeugen und in Buchfink einen Mandanten anlegen, Ablage und
   Eingang in leeren Ordnern außerhalb der Testdaten.
2. Den Agenten verbinden und die Stammdaten aus der README vorschlagen lassen
   (`stammdaten_vorschlagen`); in der App nach der Vorschau übernehmen.
3. Den ersten Kontoauszug liest der Mensch in der App ein.
4. Die Schrittordner der Reihe nach: Dateien ins Fenster ziehen, „Buche die
   Belege im Eingang.“ sagen, Rückfragen beantworten, den Monat in der App
   festschreiben.
5. Gegen die Kontrollwerte prüfen: Voranmeldung je Monat, Zusammenfassende
   Meldung, Bankbestand, Bilanz und GuV zum Stichtag (`auswertung` mit `bis`).

Die README beschreibt die fachlich erwartete Lösung. Weicht Buchfink ab, ist das
ein Befund zum Nachsehen, nicht automatisch ein Fehler der App.

| Mandant | Letzter Lauf | Ergebnis |
|---|---|---|
| 1 Nordlicht | Oktober 2026 auf `8cd1f9a` | August und September mit Zeitstempel festgeschrieben. Die Befunde sind behoben, ein Lauf auf dem behobenen Stand steht aus. |
| 2 Kranich | Oktober 2026 auf `8d4179f` | Juni bis August mit Zeitstempel festgeschrieben. Voranmeldungen Juni bis September und Zusammenfassende Meldung für das dritte Quartal stimmen mit den Kontrollwerten, der Bankbestand bis auf die damals nicht zuordenbare Gutschrift. Die Befunde sind behoben, ein Lauf auf dem behobenen Stand steht aus. |
| 3 Waldkauz | noch keiner | Der Lauf steht aus. Der Agent ordnet Zahlungen auf Abschlagsrechnungen zu und bucht die Forderung des insolventen Kunden aus; die Texte des Anhangs erfasst der Mensch in der App oder der Agent als Entwurf. |

## Vom Abschluss zum Beleg

Ein sachverständiger Dritter muss sich ohne Erklärung zurechtfinden (GOB-02).
`task check:clicks` startet die Oberfläche mit den Beispieldaten aus
`scripts/site-screenshots/mock-bridge.ts` und geht von einer Bilanzposition über
Konto und Buchung zum Beleg. Mehr als vier Klicks sind ein Befund, keine
Anpassung der Grenze.

## Was nicht belegt ist

Nicht belegt sind: ein Import der E-Bilanz als XBRL oder als Saldenliste in
ebilanzplus.de oder ebilanzonline.de und ihre Annahme durch die
Finanzverwaltung, die Annahme durch das Unternehmensregister, ein Import der
Datenüberlassung in eine Prüfsoftware, eine Wiederherstellung auf einem zweiten
Rechner, die sichtbare Ablagezone beim Ziehen von Dateien unter macOS und Linux
und eine Ende-zu-Ende-Anmeldung jedes Clients unter jedem Betriebssystem. Automatische Tests belegen keine
vollständige Gesetzeskonformität.
