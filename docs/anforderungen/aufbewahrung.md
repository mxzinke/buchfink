# D. Aufbewahrung und Archivierung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Datensicherheit](../nutzung/datensicherheit.md)

Buchfink löscht keine Buchführungsdaten. Die Fristen werden berechnet und
angezeigt; über das Löschen nach Fristablauf entscheidet das Unternehmen
außerhalb von Buchfink.

## ARC-01 Aufbewahrungsfristen

MUSS · § 257 Abs. 4 HGB, § 147 Abs. 3 AO, § 14b Abs. 1 UStG, Art. 95 EGHGB, Art. 97 § 19a EGAO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jedes Objekt hat eine Fristenklasse und ein frühestes Löschdatum | erfüllt | `accounting.RetentionFor`, gesetzt beim Ablegen eines Belegs |
| Klasse aus der Belegart abgeleitet, Verlängerung mit Grund und Protokoll | erfüllt | `domain.RetentionKindOf`, `service.ReceiptService.OverrideRetention` (nur nach oben) |
| Löschen vor Fristablauf ausgeschlossen | erfüllt | Buchfink kennt keinen Löschweg für Buchführungsdaten |
| Fristen ohne Codeänderung anpassbar | teilweise | Fristen als datierte Regeln in `accounting/retention_rules.json`; eine neue Fassung kommt mit einem Programmupdate |

## ARC-02 Fristbeginn und Ablaufhemmung

MUSS · § 257 Abs. 5 HGB, § 147 Abs. 3 S. 5 AO, §§ 169, 170 AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Fristbeginn am Schluss des Entstehungsjahres | erfüllt | `accounting.RetentionFor`, `domain.EarliestDeletionAfter` |
| Aussetzung der Frist bei Außenprüfung oder Rechtsbehelf | nicht im Umfang | Ohne Löschweg gibt es nichts aufzuhalten; die Ablaufhemmung beachtet das Unternehmen beim Löschen außerhalb von Buchfink |

## ARC-03 Originalformat und maschinelle Auswertbarkeit

MUSS · § 147 Abs. 2 AO, GoBD Rz 131 bis 135

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Eingehende Dateien im Original, zusätzlich zu jeder Ansicht | erfüllt | genau eine Datei in der Rolle `original` (`domain.Receipt.ValidateStructure`) |
| Strukturierte Formate bleiben strukturiert | erfüllt | `service.EInvoiceService.ExtractStructuredPart` legt das XML als eigene Datei ab; CAMT-Datei als Kontoauszugsbeleg (`core.Tenant.ImportBankStatement`, `service.BankService.FileStatement`) |
| Dateiname, Format, Zeitpunkt, Quelle und Prüfsumme gespeichert | erfüllt | `domain.ReceiptFile` |
| Abgeleitete Dateien gekennzeichnet und mit dem Original verknüpft | erfüllt | `domain.Receipt.DisplayFile` bevorzugt das Original |

## ARC-04 Lesbarmachung

MUSS · § 239 Abs. 4 HGB, § 147 Abs. 5 AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jede Unterlage am Bildschirm anzeigbar und als Datei ausgebbar | erfüllt | `service.ReceiptService.Content` mit Prüfsummenkontrolle, `wailsbridge.BuchfinkBridge.SaveReceiptFileAs` |
| Vollständiger Export eines Geschäftsjahres mit Index und Feldbeschreibung | erfüllt | `service.ExportService.ExportArchive` |
| Export ohne die Anwendung lesbar, auf fremdem System getestet | teilweise | CSV nach RFC 4180, `index.xml` gegen die mitgelieferte DTD, Feldbeschreibung im Klartext; ein Test auf fremdem System ist nicht belegt |

## ARC-05 Systemwechsel und Auslagerung

MUSS · § 147 Abs. 6 S. 5 AO, GoBD Rz 142 bis 144

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Migrationsprotokoll mit Quelle, Ziel, Umfang je Objektart, Zeitpunkt | erfüllt | `core.Host.recordMigration` beim Anmelden einer übernommenen oder wiederhergestellten Datei |
| Abstimmung von Salden und Zahlen vor und nach der Migration | teilweise | Zählung je Objektart und Soll-Haben-Probe nach der Übernahme; einen Vergleich mit den Zahlen des Altsystems gibt es nicht |
| Übernommene Datensätze gekennzeichnet, mit Herkunft im Altsystem | erfüllt | `domain.JournalEntry.LegacyRef` aus der Eröffnungsbilanz des Umsteigers |
| Umstellungszeitpunkt hinterlegt, Fünfjahresfrist berechnet | erfüllt | `wailsbridge.BuchfinkBridge.SetSystemChangeDate`, `accounting.SystemChangeNote` |

## ARC-06 Speicherort und Verlagerung ins Ausland

MUSS · § 146 Abs. 2 bis 2c AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Speicherort dokumentiert und einsehbar | erfüllt | Datenordner in der App und in der Verfahrensdokumentation |
| Verlagerung mit Hinweis auf Anzeige- oder Bewilligungspflicht | teilweise | Kein Umzugsweg. Liegt der Datenordner in einem bekannten Cloud-Ordner, weist `domain.CloudFolderWarning` auf § 146 Abs. 2a AO hin |
| Ortsbindung von Sicherungen, Betriebsvertrag, Datenzugriff aus dem Inland | nicht im Umfang | Kein Betreiber; die Daten liegen als Datei beim Unternehmen |

## ARC-07 Aufbewahrung von E-Rechnungen

MUSS · § 14b Abs. 1 UStG, § 14 Abs. 3 UStG, BMF-Schreiben vom 15.10.2025 Rn. 60

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| XML byteidentisch, Hybrid vollständig samt eingebettetem XML | erfüllt | Original bleibt `original`, das herausgelöste XML kommt als `structured` dazu |
| Prüfsumme beim Eingang, Prüfung bei jedem Abruf | erfüllt | `service.ReceiptService.Content` |
| Validierungsbericht mit der Rechnung gespeichert | erfüllt | `domain.ReceiptValidation` |
| Bildansicht ergänzt das Original | erfüllt | gebucht wird aus dem strukturierten Teil |

## ARC-08 Verfügbarkeit und Wiederherstellbarkeit

MUSS · § 239 Abs. 4 HGB, § 147 Abs. 5 AO, GoBD Rz 103 ff., Art. 32 Abs. 1 lit. b und c DSGVO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Automatische Sicherung, Ergebnis je Lauf protokolliert | erfüllt | `core.Host.RunDueBackup` alle 24 Stunden und beim Beenden, `domain.BackupRun` |
| Mindestens jährlich ein Wiederherstellungstest, dokumentiert | teilweise | `service.BackupService.VerifyBackup` testet und protokolliert; die Aufgabenliste meldet fehlende und gescheiterte Sicherungen, erinnert aber nicht an den jährlichen Test |
| Nach der Wiederherstellung bestätigt die Integritätsprüfung den Bestand | erfüllt | `core.Host.RestoreBackup` prüft Hash-Ketten und Belegdateien und protokolliert das Ergebnis |
| Aufbewahrung der Sicherungen deckt die Fristen | teilweise | Sicherungsordner außerhalb des Datenordners, jeder Lauf eine neue Datei; wie lange die Dateien liegen, entscheidet das Unternehmen |
