# H. Jahresabschluss, E-Bilanz, Offenlegung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Jahresabschluss](../fachkonzepte/jahresabschluss.md) ·
[Pflichten und Erklärungen](../fachkonzepte/pflichten-und-erklaerungen.md)

## JAB-01 Aufstellung, Gliederung, Fristen

MUSS · §§ 242 bis 245, 264, 266, 275 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Bilanz nach § 266 HGB und GuV nach § 275 HGB, mit Vorjahr | erfüllt | `accounting.BuildStatement`, `service.StatementService.Build` |
| Gesamtkosten- und Umsatzkostenverfahren wählbar | nicht im Umfang | Nur Gesamtkostenverfahren |
| Gliederungstiefe nach Größenklasse | erfüllt | `accounting.ObligationsFor` |
| Pflichtangaben nach § 264 Abs. 1a HGB im Kopf | erfüllt | `service.StatementService.Header` benennt fehlende Angaben |
| Aufstellungsfrist je Größenklasse als Termin | erfüllt | Pflicht `aufstellung` im Pflichtenkalender |

## JAB-02 Größenklassen

MUSS · §§ 267, 267a HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Drei Merkmale je Stichtag, Klasse mit Zweijahresregel | erfüllt | `accounting.AssessSize`, `accounting.ClassifySize` |
| Schwellenwerte datiert | erfüllt | `accounting.SizeThresholdsFor` nach dem Beginn des Geschäftsjahres |
| Klassenwechsel angekündigt, mit Folgen | erfüllt | Prüfregel `size_class_change` |
| Kapitalmarktorientierung nach § 264d HGB | nicht im Umfang | – |

## JAB-03 Anhang und Lagebericht

MUSS · §§ 284 bis 289 HGB, §§ 274a, 288 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Aus den Daten: Anlagenspiegel, Restlaufzeiten, Rückstellungen, Haftungsverhältnisse, Verpflichtungen, Beteiligungen | teilweise | Aus den Daten: Anlagenspiegel (`service.AssetService.Anlagenspiegel`), Restlaufzeiten und Rückstellungsspiegel (`domain.FinancialStatement`). Haftungsverhältnisse, Verpflichtungen und Beteiligungen sind Freitext |
| Freitexte erfassbar und ins Folgejahr übernommen | erfüllt | `domain.NotesSection`, `service.AppropriationService.CopyNotesInto` |
| Anhangumfang nach Größenklasse | teilweise | `accounting.ObligationsFor` entscheidet über Anhang und Lagebericht; die Größenerleichterungen der einzelnen Angaben sind nicht abgebildet; einen Lagebericht erzeugt Buchfink nicht |
| Kleinstgesellschaft: Angaben unter der Bilanz | teilweise | eigener Abschnitt in Ansicht und PDF; ob die Angaben vollständig sind, prüft Buchfink nicht |

## JAB-04 Feststellung und Unterzeichnung

MUSS · § 245 HGB, § 42a GmbHG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Stand offen, aufgestellt, festgestellt, offengelegt, je Wechsel Datum, Person und Beschlussbezug | erfüllt | `domain.FiscalYear`, `service.ClosingService.SetFiscalYearStatus`, nur in der App |
| Nach der Feststellung keine Buchung mehr | erfüllt | `service.JournalService.Post` weist sie ab |
| Begründete Rücknahme | erfüllt | `service.ClosingService.ReopenFiscalYear` |
| Feststellungsbeschluss mit dem Abschluss verknüpft | teilweise | Beschlussbezug als Text am Geschäftsjahr; das Dokument selbst wird nicht abgelegt |
| Unterzeichneter Abschluss unveränderlich archiviert | offen | Bilanz und GuV entstehen als PDF (`service.StatementService.ExportPDF`), werden aber nicht abgelegt |

## JAB-05 E-Bilanz

MUSS / TERMIN · § 5b EStG, § 60 EStDV, BMF-Schreiben zu den Taxonomien

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| XBRL nach der gültigen Taxonomie, übermittelt oder zur Übermittlung bereit | teilweise | `ebilanz.Generate` mit Taxonomie 6.9 und 6.10; Prüfung gegen die amtlichen Schemata mit `task test:ebilanz-xsd`. Daneben eine Summen- und Saldenliste mit derselben Zuordnung (`ebilanz.SuSaCSV`) für den Import in eBilanz+. Buchfink übermittelt nicht; eine Annahme durch einen E-Bilanz-Dienst und die Finanzverwaltung ist nicht belegt |
| Taxonomie als austauschbare Ressource | teilweise | eingebettete Auszüge in `internal/ebilanz/taxonomy/`; eine neue Fassung kommt mit einem Programmupdate |
| Jedes Konto einer Position zugeordnet, ungeklärte blockieren und werden benannt | erfüllt | `ebilanz.ElementForAccount`, Befunde in `ebilanz.MappingReport` |
| Kontennachweis unverdichtet mit Salden | erfüllt | Kontennachweis je Konto in der Instanz |
| Ab Wirtschaftsjahr 2028 Anlagenspiegel und Anlagenverzeichnis | offen | – |
| Auffangpositionen nur ersatzweise, im Bericht ausgewiesen | erfüllt | `ebilanz.MappingReport` |
| Übermittlung protokolliert | teilweise | Der Mensch bestätigt die Pflicht `e-bilanz` mit Datum und Referenz; der Verlauf steht im Änderungsprotokoll |

## JAB-06 Maßgeblichkeit und Überleitung

MUSS · § 5 Abs. 1 EStG, § 60 Abs. 2 EStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Handels- und Steuerbilanz aus demselben Buchungsstamm, Abweichungen je Position | erfüllt | `service.TaxRegisterService.Reconcile` |
| Überleitung mit Rechtsgrundlage je Abweichung | erfüllt | wie oben |
| Vollständige Steuerbilanz wahlweise | nicht im Umfang | Einheitsbilanz |
| Überleitung in der E-Bilanz | erfüllt | `ebilanz.Adjustment` |

## JAB-07 Offenlegung

MUSS · §§ 325 bis 329, 335 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Offenzulegender Umfang aus der Größenklasse | teilweise | `accounting.ObligationsFor` nennt Umfang und Norm; ein Auszug in diesem Umfang entsteht nicht |
| Datensatz im Format des Unternehmensregisters | offen | – |
| Zwölfmonatsfrist überwacht | erfüllt | Pflicht `offenlegung` im Pflichtenkalender |
| Hinterlegung für Kleinstgesellschaften wählbar | offen | – |
| Einreichungsnachweis archiviert | teilweise | Bestätigung der Pflicht mit Datum und Referenz; kein Dokument |

## JAB-08 Prüfungsfähigkeit

MUSS* · §§ 316 bis 324a HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Lesender, befristeter Prüferzugang | erfüllt | Prüfermodus mit Ablaufdatum und Grund (`core.Tenant.EnableExaminer`), gespeichert im Mandanten |
| Alle Bewegungsdaten eines Jahres maschinell auswertbar | erfüllt | `service.ExportService.ExportAuditPackage` |
| Summen- und Saldenliste zu jedem Stichtag, auch rückwirkend | erfüllt | `service.AccountingService.GetSuSaOverviewAt` |
| Saldenbestätigungen für Debitoren und Kreditoren | offen | – |
| Prüfungsbericht mit dem Abschluss verknüpft | offen | – |

## JAB-09 Jahreswechsel und Saldenvortrag

MUSS · § 252 Abs. 1 Nr. 1 HGB, § 242 Abs. 1 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Saldenvortrag als eigene Buchung mit Belegverweis | erfüllt | `service.ClosingService.CarryForward` gegen 9000, 9008 und 9009 |
| Wiederholbar ohne doppelte Werte | erfüllt | ein neuer Lauf storniert den alten |
| Änderung des Vorjahres nach dem Vortrag wird gemeldet, Korrektur angeboten | erfüllt | `service.ClosingService.CarryForwardState` |
| Personenkonten mit offenen Posten vorgetragen | erfüllt | einzelne Posten je Partner |
| Vorjahresergebnis auf das Ergebnisvortragskonto, verteilt nach Beschluss | erfüllt | `service.AppropriationService.BookAppropriation` |
