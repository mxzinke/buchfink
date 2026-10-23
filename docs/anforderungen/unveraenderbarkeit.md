# C. Unveränderbarkeit und Protokollierung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Datensicherheit](../nutzung/datensicherheit.md)

## UNV-01 Unveränderbarkeit der Aufzeichnung

MUSS · § 239 Abs. 3 HGB, § 146 Abs. 4 AO, GoBD Rz 107 bis 112

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Festgeschriebene Buchungen technisch gegen Ändern und Löschen gesichert | erfüllt | Das Journal kennt nur Anhängen; Hash-Kette je Geschäftsjahr (`accounting.HashChain`), Beleg-Hash an jeder Buchung (`accounting.ReceiptHash`). Grenze: keine Datenbanktrigger, die Datei selbst ist nicht verschlüsselt; eine Änderung von außen wird erkannt, nicht verhindert |
| Eine Manipulation wird benannt, mit erwartetem und tatsächlichem Prüfwert | erfüllt | `domain.IntegrityBreak` |
| Integritätsprüfung aufrufbar und protokolliert | erfüllt | `service.JournalService.VerifyIntegrity` |
| Export enthält die Integritätsmerkmale | erfüllt | `journal.csv` mit Vorgänger- und Eigenhash; die kanonische Form steht in der Feldbeschreibung |

## UNV-02 Festschreibung

MUSS · § 146 Abs. 4 AO, GoBD Rz 107

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Festschreibung mindestens monatlich, zusätzlich von Hand | erfüllt | `core.Tenant.CommitPeriod` in lückenloser Reihenfolge, nur in der App; Pflicht `festschreibung` im Pflichtenkalender |
| Festschreibungszeitpunkt je Buchung gespeichert und exportiert | erfüllt | `domain.JournalEntry.CommittedAt`; ein RFC-3161-Zeitstempel sichert den Stand (`timestamp.RequestToken`) |
| Keine Rücknahme der Festschreibung | erfüllt | `domain.Festschreibung` hat keinen Löschweg |
| Entwürfe außerhalb von Bilanz, GuV und Meldungen | erfüllt | eine Buchung entsteht erst mit `service.JournalService.Post`; Rechnungsentwürfe sind nicht gebucht |

## UNV-03 Änderungsprotokoll

MUSS · § 146 Abs. 4 AO, GoBD Rz 58 f., 109

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Stammdatenänderungen mit Vorher, Nachher, Zeitpunkt und Bearbeiter | erfüllt | `domain.AuditLogEntry` mit Erfassungsweg und Programmfassung |
| Zeitabhängige Parameter versioniert, Auswertung nutzt den damals gültigen | erfüllt | `accounting.TaxParametersFor`, `accounting.SizeThresholdsFor`, `accounting.CorporateTaxPermilleFor` |
| Protokoll mit derselben Frist, maschinell auswertbar | erfüllt | `aenderungsprotokoll.csv` in jeder Datenüberlassung |
| Protokoll nicht änderbar | erfüllt | nur Anhängen; jeder Eintrag hängt am Hash seines Vorgängers (`accounting.AuditChain`) |

## UNV-04 Berechtigungen und Funktionstrennung

MUSS · § 146 Abs. 4 AO, GoBD Rz 100 bis 103

Buchfink hat keine Benutzerkonten. Getrennt wird nach dem Erfassungsweg: Mensch
in der App, Hintergrundlauf oder verbundene KI-App.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Rollen für Erfassen, Freigeben, Festschreiben, Stammdaten | teilweise | Keine Personenrollen. Verbundene Apps erhalten je Mandant die Rechte Lesen oder Buchen. Festschreibung, Abschlussstände, Übermittlungen, Pflichtbestätigungen, Unternehmensdaten und Prüfermodus sind dem Menschen in der App vorbehalten (`core.AppOnly`, `service.requireHuman`) |
| Vergabe von Rechten protokolliert | erfüllt | Zulassung und Entzug einer App stehen im Änderungsprotokoll des Mandanten |
| Jede Buchung, Freigabe und Festschreibung trägt Bearbeiter und Erfassungsweg | erfüllt | `actor.Actor` (Benutzer@Rechner), `actor.RecordedChannel` (`ui`, `system`, `agent:<name>`) an Buchung und Protokolleintrag |
| Bericht über die aktuelle Rechtevergabe | teilweise | `wailsbridge.BuchfinkBridge.GetConnectedApps` listet die verbundenen Apps mit ihren Rechten; kein Export |

## UNV-05 Internes Kontrollsystem

SOLL · GoBD Rz 100 bis 102

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Regelbasierte Prüfung vor Buchung und Periodenabschluss | erfüllt | Prüfungen in `service.JournalService.Post`; Prüfregeln in `service.CheckService` |
| Jeder Kontrolllauf mit Zeitpunkt, Umfang und Befunden gespeichert | erfüllt | `domain.CheckRun` |
| Übergangene Befunde mit Begründung und Bearbeiter protokolliert | erfüllt | `service.CheckService.EnsureCommittable` |

## UNV-06 Programmfassung und Versionsnachweis

MUSS · GoBD Rz 154, § 147 Abs. 1 Nr. 1 AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jede Buchung speichert die Programmfassung | erfüllt | `domain.JournalEntry.AppVersion` |
| Versionshistorie mit Datum und Änderungen, exportierbar | erfüllt | `changelog.Entries`, eingebettet aus `internal/changelog/CHANGELOG.md` |
| Update ändert keine festgeschriebenen Daten, Migrationen protokolliert | erfüllt | `repository.ApplyMigrations`; eine Datei mit neuerer Schemaversion wird abgewiesen |
