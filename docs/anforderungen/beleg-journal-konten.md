# B. Beleg, Journal, Konten

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Belege und Buchungen](../fachkonzepte/beleg-buchungsflow.md)

## BEL-01 Keine Buchung ohne Beleg

MUSS · § 238 Abs. 1 S. 3 HGB, § 145 Abs. 1 AO, GoBD Rz 61 ff.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jede Buchung verweist auf einen Beleg oder Eigenbeleg | erfüllt | `service.PostingService`, `service.PostingService.PostManualEntry` |
| Eigenbeleg gekennzeichnet, mit Aussteller, Datum, Betrag, Grund und erfassender Person | erfüllt | `service.ReceiptService.CreateSelfIssued` |
| Buchungen ohne Beleg im Prüfbericht, verhindern die Festschreibung | erfüllt | Prüfregel `entry_without_receipt` |
| Belegverweis in beide Richtungen | erfüllt | `domain.JournalEntry.ReceiptID`, `wailsbridge.BuchfinkBridge.GetEntriesForReceipt` |

## BEL-02 Belegangaben und Belegnummer

MUSS · GoBD Rz 64, 71 bis 77

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Je Beleg Nummer, Datum, Aussteller, Betrag mit Währung, Steuer, Betreff, Erfassungszeitpunkt | erfüllt | `domain.Receipt`, `domain.ReceiptHeader` |
| Belegnummer vom System vergeben, eindeutig, unveränderlich | erfüllt | Vergabe in der Transaktion, die den Beleg schreibt |
| Fehlende Pflichtangaben blockieren und werden einzeln benannt | erfüllt | `domain.Receipt.ValidateHeader` |
| Nummernkreis je Geschäftsjahr und Belegart, ohne Doppelvergabe | erfüllt | `domain.NumberRange` |

## BEL-03 Belegsicherung

MUSS · GoBD Rz 67 bis 70

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Eingang unveränderlich gespeichert, mit Zeitstempel, Quelle und Prüfsumme | erfüllt | `receiptstore.Store.Put` legt unter SHA-256 ab |
| Kein Löschen; fehlerhafte Belege werden mit Begründung verworfen und bleiben sichtbar | erfüllt | `service.ReceiptService.Discard` |
| Keine halb gespeicherten Zustände | erfüllt | Datei zuerst atomar, dann die Transaktion (`service.ReceiptService.File`) |

## BEL-04 Zeitgerechte Erfassung und Festschreibung

MUSS · § 146 Abs. 1 AO, GoBD Rz 45 bis 52

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Bericht über Belege, die länger als die Erfassungsfrist liegen | erfüllt | Prüfregel `receipt_overdue`, Frist `domain.CompanySettings.ReceiptCaptureDays` (ohne Einstellung zehn Tage) |
| Erinnerung an die Festschreibung des Vormonats, Nachfrist einstellbar | erfüllt | Pflicht `festschreibung` im Pflichtenkalender, `domain.CompanySettings.CommitGraceDays` |
| Abstand zwischen Belegdatum, Erfassung und Festschreibung je Buchung auswertbar | erfüllt | `domain.JournalEntry.CommittedAt`, `domain.JournalEntry.FestschreibungID` |

## BEL-05 Journal

MUSS · § 239 Abs. 2 HGB, § 146 Abs. 1 AO, GoBD Rz 94 bis 99

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jede Buchung genau einmal, chronologisch, lückenlos nummeriert | erfüllt | Nummer und Kettenkopf in einer Transaktion |
| Nummer spätestens bei der Festschreibung, danach unveränderlich | erfüllt | vergeben bei der Erfassung, Teil des Hashs (`accounting.HashChain`) |
| Journal für jeden Zeitraum als Bericht und Datei | erfüllt | Journalfilter der App mit Zeitraum und CSV-Datei (`service.AccountingService.FilterEntriesCSV`); das Journal im Spaltenaufbau der Datenüberlassung (`service.ExportService.ExportJournalCSV`) erreicht weder App noch MCP |
| Storno als eigene Buchung | erfüllt | `service.JournalService.ReverseOn` |

## BEL-06 Konten und Kontenrahmen

MUSS · § 238 HGB, GoBD Rz 96 bis 99

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Standardkontenrahmen, je Mandant erweiterbar | teilweise | SKR04 2026 (`accounting/skr04_2026.json`); eigene Konten legt `service.AccountService.CreateCustom` an, ohne Erfassungsweg in App oder MCP |
| Jedes Konto einer Position nach §§ 266, 275 HGB und der E-Bilanz zugeordnet | erfüllt | `accounting/statement_mapping.go`, `ebilanz.ElementForAccount` |
| Konten ohne Zuordnung werden vor dem Abschluss gemeldet | erfüllt | Prüfregel `account_unmapped`; die E-Bilanz blockiert |
| Kontensalden gleich Journalsummen | erfüllt | Salden und Summen aus denselben Verkehrszahlen (`service.AccountingService`) |

## BEL-07 Kontokorrent und offene Posten

MUSS · §§ 238, 240 HGB, GoBD Rz 49

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Personenkonten rollen auf die Sammelkonten auf | erfüllt | Verdichtung auf 1200/3300; direkte Buchung auf das Sammelkonto wird abgewiesen |
| Offene Posten je Partner zu jedem Stichtag, mit Fälligkeit und Altersstruktur | erfüllt | `service.PaymentService.OpenItemsAt`, `accounting.AgeOpenItems` |
| Restlaufzeiten für §§ 268 Abs. 4, 5 und 285 Nr. 1 HGB | erfüllt | Restlaufzeitbänder in `accounting.AgeOpenItems` |
| Teilzahlung, Skonto, Ausbuchung je Posten dokumentiert | teilweise | Teilzahlung und Skonto mit `service.PaymentService.Settle`; die Ausbuchung bucht `service.PaymentService.WriteOffOpenItem`, ohne Erfassungsweg in App oder MCP |

## BEL-08 Ersetzendes Scannen

MUSS* · GoBD Rz 130, 136 bis 141

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Scanprotokoll, Vollständigkeitsprüfung, Fehlerprotokoll, Organisationsanweisung | nicht im Umfang | Buchfink bietet das Verfahren nicht an. Der Eingangsweg `scan` nennt nur die Herkunft; das Papier ist aufzubewahren |
| Keine Bildbearbeitung nach dem Buchen | erfüllt | gebuchte Belege sind versiegelt (`service.ReceiptService.Seal`), Dateien inhaltsadressiert |

## BEL-09 Storno und Korrektur

MUSS · § 239 Abs. 3 HGB, GoBD Rz 58 bis 59

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Korrektur als Storno und Neubuchung, das Original bleibt | erfüllt | `service.JournalService.CorrectEntry` |
| Storno, Neubuchung und Original in beide Richtungen verknüpft | erfüllt | `domain.JournalEntry.ReversalOfID`, `domain.JournalEntry.CorrectsEntryID` |
| Stornogrund ist Pflicht | erfüllt | `service.JournalService.ReverseOn`; der Grund geht in den Hash |
| Änderungen vor der Festschreibung mit Vorher, Nachher, Zeit und Bearbeiter | erfüllt | `domain.AuditLogEntry` |
