# Dokumentationsprüfung vom 11. September 2026

Geprüft wurden die öffentliche Beschreibung, README, Umsetzungsstand,
Dokumentstatus, Hilfekomponenten und ausgewählte Widersprüche zum Code.
Die Prüfung umfasst keine erneute fachliche Abnahme aller 349 Kriterien des
Anforderungskatalogs.

## Berichtigte Aussagen

| Bisherige Aussage | Befund und Berichtigung | Beleg |
|---|---|---|
| Laufende Buchhaltung und Jahresabschluss seien vollständig fertig | Implementierung, Einschränkungen und Projektziel werden getrennt beschrieben. Ein vollständiger Einreichungs- und Offenlegungsablauf fehlt. | `internal/service/closing_steps_service.go`, `internal/ebilanz/taxonomy_6.9.json`, Anforderungskatalog JAB-03 bis JAB-09 |
| Die E-Bilanz sei eine Datei zur direkten Einreichung oder zum Hochladen in Mein ELSTER | Die Zuordnung ist ungeprüft. Eine einreichungsfertige Datei oder ein beliebiger XBRL-Upload wird nicht zugesichert. | `internal/ebilanz/taxonomy_6.9.json`, durchgehend `verified: false` |
| Andere bilanzierende Rechtsformen seien ebenso unterstützt | Der Schwerpunkt liegt auf Kapitalgesellschaften. Die Auswahl einer Rechtsform ersetzt keine Implementierung ihrer Kapitalkonten und Sonderfälle. | `frontend/src/components/SetupAssistantScreen.tsx` |
| Angaben aus jeder PDF-Rechnung würden automatisch gelesen | Automatische strukturierte Daten kommen aus E-Rechnungen. Sonstige Belege benötigen manuelle Angaben. | `internal/service/einvoice_service.go`, `frontend/src/pages/ReceiptsPage.tsx` |
| Skontokorrekturen erreichten die Voranmeldung nicht | `vatMovements` wertet die Skontoschlüssel aus und berichtigt Bemessungsgrundlage sowie Steuer. UST-01 bleibt wegen anderer offener Steuerfälle teilweise erfüllt. | `internal/accounting/ustva.go`, `TestGrantedSkontoReducesTheReportedBase`, `TestSkontoOnReverseChargeCorrectsBothLegs` |
| Ein Anhang werde nicht ausgegeben | Freitexte, Rückstellungsspiegel und Überleitung werden zusammengestellt und ausgegeben. Die größenabhängige Auswahl bleibt unvollständig. Das Sprach-/Währungskriterium wurde auf erfüllt gesetzt. | `StatementService.notesFor`, `ExportCSV`, `statementTypst`, `writeTypstNotes`, `TestFinancialStatementCarriesTheNotes` |
| Anzahlungen und Rechnungsabgrenzung seien noch nicht implementiert | Die Statuszeilen der Fachkonzepte waren veraltet. Sie sind jetzt als Entwurfsgrundlagen mit Verweis auf die Implementierung gekennzeichnet. | `internal/service/advance_service.go`, `accrual_service.go` |
| Bankzuordnung schlage bewusst keine Kontierung vor | Offene Rechnungen und gelernte Bankregeln liefern Vorschläge, die der Anwender bestätigt. | `frontend/src/pages/BankImportPage.tsx`, `internal/service/bank_service.go` |
| Sicherung erfolge täglich oder beim Beenden höchstens einmal pro Tag | Automatische Aufrufe erfolgen beim Start und Beenden. Beim Beenden zählen auch Änderungen seit der letzten Sicherung; es gibt keinen täglichen Zeitgeber. | `internal/wailsbridge/backup_service.go`, `runDueBackup`, `internal/service/backup_service.go` |
| Unverschlüsselte Originalbelege seien wegen der GoBD vorgeschrieben | Unverschlüsselte Belege sind die aktuelle Speicherentscheidung. Die Pflicht zur Originalerhaltung ist kein allgemeines Verschlüsselungsverbot. | `internal/receiptstore/store.go`, `internal/repository/encryption.go` |
| Datenbankverschlüsselung verhindere jede Neuberechnung der Hashkette | Feldverschlüsselung und Hashprüfung haben unterschiedliche Grenzen. Das Sicherheitskonzept enthält keine solche Garantie mehr. | `internal/accounting/journalhash.go`, `internal/security/crypto.go` |
| Node ab 20 genüge | Die installierte Vite-Version verlangt Node 20 ab 20.19 oder Node ab 22.12. README und Installationsseite nennen die passende Untergrenze. | `frontend/package-lock.json`, Vite `engines.node` |
| Fragezeichen seien ausführliche Popover, teils Tooltip/Popover/Dialog | Ein gemeinsamer Hilfebaustein verwendet kurze Tooltips und einen getrennten Button „Mehr erfahren“ für Details. Die Dokumente beschreiben dasselbe Verhalten. | `frontend/src/components/ui/Help.tsx`, `Field.tsx`, `Section.tsx` |
| Gesetzesverweise seien nur Text oder eigene sichtbare Tabellenspalten | Normen in Hilfedialogen sind verlinkt. Sichtbare Rechtsgrundlagen wurden in Details verlegt. Handlungsrelevante Befunde bleiben sichtbar. | `frontend/src/components/ui/LegalText.tsx`, betroffene Seiten |
| Zukünftige Schnittstellen seien TODO-Kommentare im Datenmodell | Kommentare beschreiben die vorhandenen Grenzen. Geplante Erweiterungen stehen getrennt in der Roadmap, einschließlich des noch nicht vorhandenen lokalen MCP-Servers. | `internal/domain/`, `docs/roadmap.md` |

## Quellen für überarbeitete Rechtsverweise

- [§ 5b EStG](https://www.gesetze-im-internet.de/estg/__5b.html) zur elektronischen Bilanzübermittlung.
- [§ 146 AO](https://www.gesetze-im-internet.de/ao_1977/__146.html) zu Aufzeichnungen und elektronischen Büchern im EU-Ausland beziehungsweise in Drittstaaten. Der Einrichtungshinweis unterscheidet jetzt Absätze 2a und 2b.
- [§ 248 HGB](https://www.gesetze-im-internet.de/hgb/__248.html) zum Ansatzverbot für Gründungsaufwendungen.
- [BGH, II ZR 65/04](https://juris.bundesgerichtshof.de/cgi-bin/rechtsprechung/document.py?Art=en&Blank=1.pdf&Datum=2006-1-16&Gericht=bgh&anz=6&nr=35702&pos=3) als Rechtsprechungsquelle im Gründungsdialog.
- [GoBD im amtlichen Handbuch](https://amtliche-handbuecher.bundesfinanzministerium.de/ao/2025/Anhaenge/BMF-Schreiben-und-gleichlautende-Laendererlasse/Anhang-33/anhang-33.html) für die verlinkten Verwaltungsregeln.

## Grenzen und Pflege

Ältere Zeilenverweise im Anforderungskatalog können durch Umbauten verschoben
sein. Bei den berichtigten Punkten werden Funktionen und Tests genannt.
Verbliebene Fachkonzepte enthalten Entwurfsentscheidungen; ihr Inhalt darf nicht
ungeprüft als aktuelle Funktionszusage auf die Webseite übernommen werden.

Backend-Meldungen und fachliche Detailtexte sind nicht sämtlich rechtlich neu
bewertet worden. Automatische Quellenlinks ersetzen keine Prüfung der Aussage.
Bei unbekannten Zitierweisen und Quellen müssen konkrete Links ergänzt werden.
Ein menschenlesbarer Kurztext für neue Hilfe ist Pflicht; die Prüfung über
`task check:text` unterstützt diese Regel.

## Technische Prüfung

Der Frontend-Build, die Prüfungen für Design-Tokens, Hilfetexte, Bridge-Bindings
und Abschlussnavigation sowie `go vet` laufen durch. Auch die Tests der
Wails-Bridge sind erfolgreich. Der Browserlauf prüft
Tooltips, Tastaturfokus, Quellenlinks, Escape, Fokusrückgabe und verschachtelte
Hilfedialoge. Alle 20 Webseiten-Screenshots wurden aus der aktuellen Oberfläche
neu erstellt. Die sieben Webseiten wurden einschließlich Dialogen, lokalen
Links und mobiler Startseite geprüft.

Die Go-Testsuite hat einen vorbestehenden Fehler:
`TestFileOpeningBalanceRefusesAnUnbalancedSheet` in
`internal/service/foundation_opening_test.go:133` erwartet die Ablehnung einer
Eröffnungsbilanz, erhält aber keinen Fehler. Derselbe Test scheitert auch in
einer separaten Kopie des unveränderten Git-Ausgangsstands. Das belegt eine
abweichende Testerwartung; die fachliche Ursache ist separat zu klären. Die
Buchungslogik wurde im Rahmen dieser Textüberarbeitung nicht geändert.

`task` ist in der Prüfungsumgebung nicht installiert. Die im Taskfile genannten
Prüfbefehle wurden direkt ausgeführt. Vite meldet weiterhin die vorhandene
Warnung zur Größe des JavaScript-Bundles.
