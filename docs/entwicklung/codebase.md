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
| Anmeldung und Zustimmung verbundener Apps | [internal/mcpserver/oauth/](../../internal/mcpserver/oauth), [internal/wailsbridge/mcp_service.go](../../internal/wailsbridge/mcp_service.go) |
| Brücke für Claude Desktop | [cmd/buchfink-mcp/](../../cmd/buchfink-mcp), [packaging/claude-desktop/](../../packaging/claude-desktop) |
| Schreibsperre, Transaktion, Prüfermodus, Vorbehalt der App | [internal/core/](../../internal/core) |
| Anwendungsfall | [internal/service/](../../internal/service) |
| Rechenregel, Steuerschlüssel, Kontenrahmen | [internal/accounting/](../../internal/accounting) |
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
| E-Bilanz | [ebilanz](../../internal/ebilanz) mit Taxonomie-Auszug in `taxonomy/` | [Pflichten und Erklärungen](../fachkonzepte/pflichten-und-erklaerungen.md) |
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
