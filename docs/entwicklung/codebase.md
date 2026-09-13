# Orientierung im Code

[Dokumentation](../README.md) · [Entwicklungsumgebung](README.md)

Buchfink verbindet eine Wails-v3-Desktop-Anwendung mit einem Go-Backend und
einem Frontend aus React und TypeScript. SQLite speichert die Daten je
Unternehmen. Die Oberfläche nutzt Base UI, Tailwind CSS und Lucide; Typst
rendert Dokumente über WebAssembly.

## Wo eine Änderung beginnt

| Aufgabe | Fundstelle |
|---|---|
| Ansicht oder Bedienablauf ändern | [frontend/src/pages/](../../frontend/src/pages), [components/](../../frontend/src/components) |
| Gemeinsame UI-Bausteine anpassen | [components/ui/](../../frontend/src/components/ui), [Designkonzept](../gestaltung/README.md) |
| Frontend mit Go verbinden | [frontend/src/services/](../../frontend/src/services), [internal/wailsbridge/](../../internal/wailsbridge) |
| Anwendungsfall umsetzen | [internal/service/](../../internal/service) |
| Fachliche Typen und Rechenregeln ändern | [internal/domain/](../../internal/domain), [internal/accounting/](../../internal/accounting) |
| Speicherung oder Migration ändern | [internal/repository/](../../internal/repository) |
| App-Start oder Build anpassen | [main.go](../../main.go), [build/](../../build), [Taskfile.yml](../../Taskfile.yml) |

Die [Architektur und das Bedienkonzept](architektur.md) erläutern den
fachlichen Zuschnitt, die Schichten und den Jahreslauf in der Oberfläche.

## Fachmodule nachschlagen

| Thema | Pakete und Hintergrund |
|---|---|
| Bankimport | [bank](../../internal/bank) |
| E-Rechnungen und Dokumenterzeugung | [einvoice](../../internal/einvoice), [invoice](../../internal/invoice), [Modulaufbau](e-rechnung.md) |
| E-Bilanz | [ebilanz](../../internal/ebilanz), [bekannte Grenzen](../projekt/umsetzungsstand.md#jahresabschluss-und-einreichung) |
| Belegablage und Verschlüsselung | [receiptstore](../../internal/receiptstore), [security](../../internal/security), [Sicherheitskonzept](../nutzung/datensicherheit.md) |
| Datenüberlassung und Verfahrensdokumentation | [export](../../internal/export), [procdoc](../../internal/procdoc) |
| Externe Dienste | [currency](../../internal/currency), [timestamp](../../internal/timestamp), [vatid](../../internal/vatid) |
| Bearbeiter und Programmversion | [actor](../../internal/actor), [buildinfo](../../internal/buildinfo), [changelog](../../internal/changelog) |

Die zugehörigen Abläufe und Anforderungen findest du im
[fachlichen Themenverzeichnis](../fachkonzepte/README.md).

## Werkzeuge und Webseite

[scripts/](../../scripts) enthält Prüfungen und Erzeugungsskripte.
[Prüfszenarien](pruefszenarien.md) beschreiben die fachlichen und manuellen
Prüfwege. Die öffentliche Projektseite liegt unter [website/](../../website/README.md),
ihre Logos und weitere App-Grafiken unter [assets/](../../assets).
