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
| Ein Agent kann nichts ausführen, was der App vorbehalten ist | `core.TestAgentsAreRefusedForHumanOnlyOperations` |
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
6. Einen Vorgang anstoßen, der der App vorbehalten ist, etwa die
   Festschreibung. Erwartet: kein Werkzeug dafür, oder der Fehler `nur_in_app`.
7. In den Einstellungen unter „Verbundene Apps“ den Zugriff entziehen.
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
| E-Bilanz | Zuordnung der Konten, Befunde, Taxonomie nach Wirtschaftsjahr | `internal/ebilanz/cases_test.go`, `version_test.go`; Schema mit `task test:ebilanz-xsd` |
| Pflichtenkalender | Fristen mit Beratung, Dauerfristverlängerung und Feiertagen | `internal/service/obligation_service_test.go`, `internal/calendar/holidays_test.go` |
| Rechnungsdokumente | Pflichtangaben in PDF und XML, Bezüge bei Storno und Berichtigung | `internal/invoice/business_letters_test.go`, `cii_content_test.go` |
| Sicherung | Sicherung, Wiederherstellungstest, beschädigte Dateien werden erkannt | `internal/service/backup_service_test.go` |

## Vom Abschluss zum Beleg

Ein sachverständiger Dritter muss sich ohne Erklärung zurechtfinden (GOB-02).
`task check:clicks` startet die Oberfläche mit den Beispieldaten aus
`scripts/site-screenshots/mock-bridge.ts` und geht von einer Bilanzposition über
Konto und Buchung zum Beleg. Mehr als vier Klicks sind ein Befund, keine
Anpassung der Grenze.

## Was nicht belegt ist

Nicht belegt sind: eine Annahme der E-Bilanz durch ERiC, die Annahme durch das
Unternehmensregister, ein Import der Datenüberlassung in eine Prüfsoftware, eine
Wiederherstellung auf einem zweiten Rechner und eine Ende-zu-Ende-Anmeldung
jedes Clients unter jedem Betriebssystem. Automatische Tests belegen keine
vollständige Gesetzeskonformität.
