# Orientierung im Code

[Dokumentation](../README.md) · [Entwicklung](README.md) ·
[Architektur](architektur.md)

Buchfink ist eine Wails-v3-Anwendung mit Go-Backend und einer Oberfläche aus
React und TypeScript. SQLite speichert die Daten je Mandant. Die Oberfläche
nutzt Base UI, Tailwind CSS und Lucide; Typst setzt Rechnungen und Berichte
über WebAssembly.

## Wo eine Änderung beginnt

| Aufgabe | Fundstelle |
|---|---|
| Werkzeug für Agenten | [internal/mcpserver/](../../internal/mcpserver), [MCP-Server](mcp.md) |
| Dienstverdrahtung und Adaptergrenze | [core/wire.go](../../internal/core/wire.go), [Graph und Setter-Inventur](../../internal/core/wire_graph_test.go), [MCP-Fassade](../../internal/core/agent_services.go), [Fassadenwächter](../../internal/mcpserver/agent_services_test.go) |
| Anmeldung und Zustimmung verbundener Apps | [internal/mcpserver/oauth/](../../internal/mcpserver/oauth), [internal/wailsbridge/mcp_service.go](../../internal/wailsbridge/mcp_service.go) |
| Brücke für Claude Desktop | [cmd/buchfink-mcp/](../../cmd/buchfink-mcp), [packaging/claude-desktop/](../../packaging/claude-desktop) |
| Was ein Agent ohne Rückfrage bucht, welche Pfade er lesen darf, Exportordner | [internal/core/policy/](../../internal/core/policy), [MCP-Server](mcp.md#regeln-für-direkt) |
| Eingang: Dateien listen, Art erkennen, nach `Übernommen/` verschieben, Dateien aus dem Fenster aufnehmen | [internal/inbox/](../../internal/inbox), [internal/core/policy/settle.go](../../internal/core/policy/settle.go), [internal/wailsbridge/inbox_service.go](../../internal/wailsbridge/inbox_service.go), [MCP-Server](mcp.md#eingang-leseordner-und-exportordner) |
| Warnung, wenn die Ablage in einem Sync-Ordner liegt (Einrichtung, Einstellungen, Verfahrensdokumentation) | [internal/syncdir/](../../internal/syncdir) |
| Ereignis nach einer Änderung, Nachladen der offenen Ansicht | [internal/core/changes.go](../../internal/core/changes.go), [internal/wailsbridge/changes.go](../../internal/wailsbridge/changes.go), [frontend/src/live.tsx](../../frontend/src/live.tsx), [MCP-Server](mcp.md#live-aktualisierung) |
| Freigabestufe und Vorbehalt der App: Katalog und Schutzstellen | [internal/ops/](../../internal/ops), [core/agent_test.go](../../internal/core/agent_test.go), [core/operation_contract_test.go](../../internal/core/operation_contract_test.go), [Freigabestufen](mcp.md#freigabestufen) |
| Schreibsperre, Transaktion, Prüfermodus und Durchsetzung der App-Vorbehalte | [internal/core/](../../internal/core), [service/app_only.go](../../internal/service/app_only.go) |
| Journal: feste Kernregeln, Regelreihenfolge, Steuerregeln und Ausnahmen | [service/journal_rules.go](../../internal/service/journal_rules.go), [service/journal_service.go](../../internal/service/journal_service.go), [domain/entry_rule.go](../../internal/domain/entry_rule.go), [taxrules](../../internal/taxrules), Registrierung in [core/wire.go](../../internal/core/wire.go) |
| Geschäftsjahr, Rumpfjahr, Periodensperre, Festschreibung und Feststellung | [service/periods.go](../../internal/service/periods.go), [periods_calendar.go](../../internal/service/periods_calendar.go), [periods_commit.go](../../internal/service/periods_commit.go), [periods_adoption.go](../../internal/service/periods_adoption.go); Vorprüfungen in [core/wire.go](../../internal/core/wire.go), Zeitstempelablauf in [core/commit.go](../../internal/core/commit.go) |
| Buchungsfolgen: Bankzuordnung und Freigabe beim Storno | [domain/entry_effect.go](../../internal/domain/entry_effect.go), [service/journal_effects.go](../../internal/service/journal_effects.go), [bankeffects](../../internal/bankeffects), Registrierung in [core/wire.go](../../internal/core/wire.go) |
| Offene Posten, Stichtagsstand und Altersstruktur | [settlement_items.go](../../internal/service/settlement_items.go); Dienstfassade in [payment_service.go](../../internal/service/payment_service.go) |
| Zahlungsausgleich, Differenzen und Anlagenminderung, Bankaufteilung, Ausbuchung | [settlement.go](../../internal/service/settlement.go), [settlement_difference.go](../../internal/service/settlement_difference.go), [settlement_split.go](../../internal/service/settlement_split.go), [settlement_writeoff.go](../../internal/service/settlement_writeoff.go) |
| Anwendungsfall | [internal/service/](../../internal/service) |
| Rechenregel, Steuerschlüssel, Kontenrahmen | [internal/accounting/](../../internal/accounting) |
| Steuerfall, Skonto, Steuerentstehung, UStVA- und ZM-Zuordnung | [TaxCase-Katalog](../../internal/accounting/taxcase.go), [Zeitpunkt](../../internal/accounting/taxcase_timing.go), [Melderegeln](../../internal/accounting/taxcase_reporting.go) |
| Entität, Invariante, Repository-Schnittstelle | [internal/domain/](../../internal/domain) |
| Speicherung und Migration | [internal/repository/](../../internal/repository) |
| Ansicht oder Bedienablauf | [frontend/src/](../../frontend/src), [Gestaltung](../gestaltung/README.md) |
| Aufruf aus der Oberfläche ins Backend | [frontend/src/services/](../../frontend/src/services), [internal/wailsbridge/](../../internal/wailsbridge) |
| App-Start, Build, Pakete | [main.go](../../main.go), [build/](../../build), [Taskfile.yml](../../Taskfile.yml) |

## Fachmodule

| Thema | Paket | Hintergrund |
|---|---|---|
| Pflichtenkalender | `service/obligation_*.go`, [calendar](../../internal/calendar) (Feiertage, Fristenden) | [Pflichten und Erklärungen](../fachkonzepte/pflichten-und-erklaerungen.md) |
| Steuererklärungen KSt, GewSt, USt 2 A | [declaration](../../internal/declaration) mit Zeilenkatalog in `catalog/` | [Pflichten und Erklärungen](../fachkonzepte/pflichten-und-erklaerungen.md) |
| E-Bilanz als XBRL und als Summen- und Saldenliste | [ebilanz](../../internal/ebilanz) mit Taxonomie-Auszug in `taxonomy/` | [Pflichten und Erklärungen](../fachkonzepte/pflichten-und-erklaerungen.md#e-bilanz) |
| Bankimport | [bank](../../internal/bank) | [Zahlungswege](../fachkonzepte/zahlungswege.md) |
| E-Rechnung lesen, prüfen, schreiben | [einvoice](../../internal/einvoice), [invoice](../../internal/invoice) | [Modulaufbau](e-rechnung.md) |
| Belegablage | [receiptstore](../../internal/receiptstore) | [Belege und Buchungen](../fachkonzepte/beleg-buchungsflow.md) |
| Datenüberlassung, Verfahrensdokumentation | [export](../../internal/export), [procdoc](../../internal/procdoc) | [Anforderungen I](../anforderungen/betriebspruefung.md) |
| Externe Dienste | [currency](../../internal/currency), [timestamp](../../internal/timestamp), [vatid](../../internal/vatid) | [Datensicherheit](../nutzung/datensicherheit.md) |
| Bearbeiter, Erfassungsweg, Programmfassung | [actor](../../internal/actor), [buildinfo](../../internal/buildinfo), [changelog](../../internal/changelog) | [Architektur](architektur.md#3-kern) |

## Werkzeuge

| Ort | Inhalt |
|---|---|
| [scripts/](../../scripts) | Prüfskripte für `task check`, Erzeugung von Taxonomie-Auszug, SKR04-Positionen und Drittkomponentenhinweisen |
| [scripts/site-screenshots/](../../scripts/site-screenshots/README.md) | Screenshots der Webseite und Klickweg-Prüfung mit Beispieldaten |
| [website/](../../website/README.md) | Projektseite |
| [skills/](../../skills/) | Skills für Endnutzer, eine Quelle für Plugin und Skill-Pakete |
| [plugins/buchfink-buchhaltung/](../../plugins/buchfink-buchhaltung/) | Claude-Plugin mit MCP-Server und Skills |
| [.claude-plugin/marketplace.json](../../.claude-plugin/marketplace.json) | Marketplace `buchfink` für Claude Code |
