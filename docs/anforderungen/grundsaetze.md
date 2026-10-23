# A. Buchführungspflicht und Grundsätze

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Belege und Buchungen](../fachkonzepte/beleg-buchungsflow.md)

## GOB-01 Doppelte Buchführung und Bilanzierung

MUSS · §§ 238, 242 HGB, § 13 Abs. 3 GmbHG, § 140 AO. Die GmbH und die UG sind
Formkaufleute und buchen doppelt, unabhängig von Umsatz und Gewinn.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jeder Buchungssatz hat Soll und Haben, Sollsumme gleich Habensumme ohne Toleranz | erfüllt | `domain.JournalEntry.IsBalanced` |
| Ein Buchungssatz ohne Ausgleich lässt sich nicht speichern | erfüllt | `service.JournalService.Post` ist der einzige Schreibweg |
| Summen- und Saldenliste zu jedem Stichtag | erfüllt | `service.AccountingService.GetSuSaOverviewAt` |
| Bilanz und GuV aus den Kontensalden, ohne Nacherfassung | erfüllt | `accounting.BuildStatement`; Eröffnungswerte als gebuchter Saldenvortrag |

## GOB-02 Nachvollziehbarkeit für einen sachverständigen Dritten

MUSS · § 238 Abs. 1 S. 2 HGB, § 145 Abs. 1 AO, GoBD Rz 30 ff.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Von jedem Wert in Bilanz oder GuV in höchstens vier Schritten zu Buchung und Beleg | erfüllt | `task check:clicks` zählt den Weg an der laufenden Oberfläche ([Prüfszenario](../entwicklung/pruefszenarien.md)) |
| Vom Beleg zu seinen Buchungen und Konten | erfüllt | `service.AuditTrailService`, `wailsbridge.BuchfinkBridge.GetEntriesForReceipt` |
| Bezeichnungen im Klartext oder als Schlüsselverzeichnis | erfüllt | `schluesselverzeichnis.csv` in jeder Datenüberlassung |
| Durchlauf mit einer fachkundigen Person ohne Produktkenntnis | teilweise | Gemessen werden die Klicks; ein Durchlauf mit einer Person ist nicht belegt |

## GOB-03 Vollständig, richtig, zeitgerecht, geordnet

MUSS · § 239 Abs. 2 HGB, § 146 Abs. 1 S. 1 AO, GoBD Rz 36 ff.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Lücken in Beleg- und Buchungsnummern im Prüfbericht | erfüllt | Prüfregel `number_gap` in `service.CheckService` |
| Nicht gebuchte Belege in eigener Liste, blockieren den Periodenabschluss | erfüllt | Prüfregel `receipt_unbooked` |
| Plausibilitätsprüfung vor dem Periodenabschluss | erfüllt | `service.CheckService.EnsureCommittable`, erster Schritt von `service.MonthCloseService` |
| Reproduzierbare Sortierung von Journal und Kontoblatt | erfüllt | Journalabfragen im Paket `repository` ordnen nach der Buchungs-ID |

## GOB-04 Sprache, Abkürzungen, Währung

MUSS · §§ 239 Abs. 1, 244 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Schlüsselverzeichnis aller Codes als Datei | erfüllt | `export.KeyDirectoryTable`, Teil der Datenüberlassung |
| Bilanz, GuV und Anhang deutsch und in Euro | erfüllt | `service.StatementService.ExportPDF`, `service.StatementService.ExportCSV` |
| Hauptbuch in Euro, Fremdwährung zusätzlich | erfüllt | `domain.JournalLine.ForeignAmount`, Kurs und Kursquelle am Buchungskopf |

## GOB-05 Einzelaufzeichnung

MUSS · § 146 Abs. 1 S. 1 AO, GoBD Rz 39 ff.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Sammelbuchung verweist auf gespeicherte Einzelposten mit gleicher Frist | erfüllt | Zahlungszuordnungen, Tabelle `zahlungszuordnungen` der Überlassung |
| Einzelposten erreichbar und maschinell auswertbar | erfüllt | `service.PaymentService.Allocations`; dieselben Zeilen in der Datenüberlassung |
| Je Vorgang Datum, Betrag, Steuer, Leistung, Partner, Belegverweis | erfüllt | Belegweg; Handbuchung über `service.PostingService.PostManualEntry` und die Journalregel `manual_tax_lines` in `internal/taxrules` |

## GOB-06 Geschäftsjahr und Periodenabgrenzung

MUSS · § 240 Abs. 2 S. 2 HGB, § 252 Abs. 1 Nr. 5 HGB, § 4a EStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Geschäftsjahr höchstens zwölf Monate, Rumpfjahr möglich | erfüllt | `domain.FiscalYear`, `service.ClosingService.AlignFoundingYear` |
| Buchungsdatum getrennt vom Belegdatum, Periode nach Buchungsdatum | erfüllt | vier Datumsfelder an `domain.JournalEntry` |
| Buchung in ein festgestelltes Jahr nur nach begründeter Wiedereröffnung | erfüllt | `service.ClosingService.ReopenFiscalYear` |
| Abgrenzungen lösen sich im Folgejahr selbst auf | erfüllt | `service.AccrualService.Book`, `service.AccrualService.ReleaseInto` |
