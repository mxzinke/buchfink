# E. Ausgangsrechnungen und E-Rechnung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[E-Rechnung](../fachkonzepte/e-rechnung.md)

## RECH-01 Rechnungspflicht und Ausstellungsfrist

MUSS · § 14 Abs. 2 UStG, § 14a Abs. 1 und 3 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Leistungen ohne Rechnung überwacht, Sechsmonatsfrist und Frist zum 15. des Folgemonats gemeldet | nicht im Umfang | Buchfink führt keine Aufträge; eine Rechnung entsteht erst beim Ausstellen |
| Leistungsdatum ist Pflicht | teilweise | `domain.Invoice.Validate` verlangt es; `service.InvoiceService.Issue` setzt ein leeres Leistungsdatum vorher auf das Rechnungsdatum |

## RECH-02 Pflichtangaben

MUSS · § 14 Abs. 4 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Jede Rechnung enthält die Pflichtangaben des § 14 Abs. 4 UStG | erfüllt | `domain.Invoice.ValidateParties` |
| Bei Anzahlungen steht der Zeitpunkt der Vereinnahmung | erfüllt | Abschlagsrechnung, [Anzahlungen](../fachkonzepte/anzahlungen.md) |
| Bei Gutschriften steht „Gutschrift“ | nicht im Umfang | Buchfink stellt keine Abrechnungsgutschrift nach § 14 Abs. 2 S. 2 UStG aus; Storno und Berichtigung tragen das Wort nicht |
| Ohne vollständige Angaben keine Ausstellung, das fehlende Feld wird benannt | erfüllt | `service.InvoiceService.Issue` |
| Gleiche Prüfung bei Eingangsrechnungen, mit Klärungsliste | erfüllt | `service.EInvoiceService.Validate`, Befunde am Beleg |

## RECH-03 Fortlaufende Rechnungsnummer

MUSS · § 14 Abs. 4 S. 1 Nr. 4 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Vergabe systemseitig, in der Transaktion, auch nebenläufig lückenlos | erfüllt | Nummer, Rechnung, Dokument und Buchung in einer Transaktion (`service.InvoiceService.Issue`) |
| Nummernkreise je Jahr und Art, Systematik einstellbar | erfüllt | `domain.NumberRange` |
| Eine vergebene Nummer wird nie wieder vergeben, auch nicht nach Storno | erfüllt | eindeutiger Index; das Storno bekommt eine eigene Nummer |
| Bericht über Lücken mit Grund | erfüllt | `service.InvoiceService.NumberGaps`, `service.InvoiceService.RecordNumberGapReason` |

## RECH-04 Pflichthinweise in Sonderfällen

MUSS* · § 14a UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Der Steuerfall setzt den Pflichthinweis selbst | erfüllt | Hinweise für Steuerschuldnerschaft des Leistungsempfängers, steuerfreien und nicht steuerbaren Umsatz im Paket `invoice` |
| Bei Reverse Charge kein gesonderter Steuerausweis | erfüllt | `domain.TaxTreatment.MayShowTax` |
| Hinweistexte mehrsprachig | nicht im Umfang | Rechnungen sind deutsch |
| Reverse Charge ohne Hinweis oder ohne USt-IdNr. des Empfängers lässt sich nicht ausstellen | erfüllt | Hinweis nicht abschaltbar; `domain.Invoice.ValidateParties` verlangt die USt-IdNr. |

## RECH-05 Kleinbetragsrechnung und Kleinunternehmer

MUSS* · §§ 33, 34, 34a UStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Erleichterung bis 250 Euro brutto, in den Ausnahmefällen gesperrt | erfüllt | `service.InvoiceService.Issue` prüft Grenze und Steuerfall; nur für den steuerpflichtigen Inlandsumsatz |
| Grenze datiert und zum Rechnungsdatum gelesen | erfüllt | `accounting.TaxParametersFor` |
| Reduzierte Mindestangaben | teilweise | Hinweis auf § 33 UStDV, keine Anschrift nötig, Ausgabe als PDF zulässig; ein Empfänger bleibt Pflicht, einen Barverkauf gibt es nicht |
| Kleinunternehmerrechnung | nicht im Umfang | Der eigene Mandant kann kein Kleinunternehmer sein; bei Lieferanten wird der Status geführt |
| Eingehende Kleinbetragsrechnungen und Fahrausweise geprüft | erfüllt | Belegprüfung nach §§ 33, 34 UStDV beim Buchen |

## RECH-06 E-Rechnung erzeugen

MUSS / TERMIN · § 14 Abs. 1 S. 3 bis 6 UStG, § 27 Abs. 38 UStG, EN 16931, BMF-Schreiben vom 15.10.2024 und 15.10.2025

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| XRechnung und ZUGFeRD ohne MINIMUM und BASIC-WL, Format je Empfänger | teilweise | ZUGFeRD EN 16931 und XRechnung in CII, Format am Kontakt (`domain.EInvoiceProfile`); XRechnung in UBL wird nur gelesen |
| Validierung gegen Schema und Geschäftsregeln, Bericht gespeichert | erfüllt | die erzeugte Datei wird wieder eingelesen und gegen EN 16931 und XRechnung geprüft (Paket `einvoice`) |
| Alle Pflichtangaben im strukturierten Teil | erfüllt | Datensatz aus dem semantischen Modell, Einheiten nach UN/ECE Rec. 20 |
| Bei Hybridformaten stimmen XML und Bild überein | erfüllt | `invoice.VerifyEmbeddedRecord` vor der Ablage |
| Vorjahresumsatz über 800.000 Euro steuert die Übergangsregel | erfüllt | `accounting.EInvoiceIssueTransitionFor` |
| Übertragungsweg, Versandzeitpunkt, Empfangsnachweis | nicht im Umfang | Kein Peppol, EDI oder Portal. Der Versand wird an der Rechnung vermerkt (`service.InvoiceService.MarkSent`) |

## RECH-07 E-Rechnung empfangen und prüfen

MUSS · § 14 Abs. 1 UStG, § 15 Abs. 1 UStG, BMF-Schreiben vom 15.10.2025 Rn. 6a, 6b, 35a

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| XRechnung und ZUGFeRD lesen, Buchungsvorschlag erzeugen | erfüllt | `invoice.Reader.Read` (CII und UBL, XML und Hybrid-PDF), `service.EInvoiceService.Propose`; MINIMUM und BASIC WL werden mit Grund abgewiesen |
| Befunde getrennt nach Format-, Geschäftsregel- und Inhaltsfehlern, Ergebnis gespeichert | erfüllt | `domain.ReceiptValidation` |
| Fehlerhafte Rechnungen archiviert und in einer Klärungsliste | erfüllt | Ablage vor jeder Auswertung |
| Zurückweisen ohne Archivierung ausgeschlossen | erfüllt | Verwerfen verlangt einen Grund, der Beleg bleibt (`service.ReceiptService.Discard`) |
| Vorsteuerabzug erst nach fehlerfreier Prüfung oder begründeter Übersteuerung | erfüllt | `service.PostingService` hält die Buchung an; `service.ReceiptService.SaveInputTaxOverride` |

## RECH-08 Echtheit, Unversehrtheit, Lesbarkeit

MUSS · § 14 Abs. 3 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Abgleich der Eingangsrechnung mit Bestellung oder Leistung, Ergebnis gespeichert | erfüllt | Bestellbezug (BT-13) und Leistungsnachweis am Beleg (`service.ReceiptService.SaveServiceProof`); ab der eingestellten Grenze Pflicht, Prüfregel `service_proof_missing` |
| Prüfpfad Rechnung, Leistung, Zahlung in beide Richtungen und exportierbar | erfüllt | `service.AuditTrailService`; `belege.csv` und `zahlungszuordnungen.csv` in der Datenüberlassung |
| Prüfsumme beim Eingang, Prüfung bei jedem Abruf | erfüllt | `service.ReceiptService.Content` |
| Kontrollverfahren in der Verfahrensdokumentation | erfüllt | Abschnitte Belegfluss und internes Kontrollsystem (`procdoc.Render`) |

## RECH-09 Rechnungsberichtigung

MUSS · § 14 Abs. 6 UStG, § 31 Abs. 5 UStDV, § 14c UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Berichtigung als neues Dokument mit Nummer und Datum der Ursprungsrechnung | erfüllt | `service.InvoiceService.CancelWithDocument`, `service.InvoiceService.CorrectInvoice` |
| Ursprungsrechnung bleibt unverändert | erfüllt | nur Status und Verweis ändern sich |
| Berichtigung im selben strukturierten Format | erfüllt | Format der Ursprungsrechnung |
| Storno allein oder Storno mit neuer Rechnung, verkettet | erfüllt | wie oben |
| Steuerliche Wirkung im richtigen Zeitraum | erfüllt | Storno auf den Korrekturtag datiert, `accounting.VatPeriodFor` |

## RECH-10 Unrichtiger und unberechtigter Steuerausweis

MUSS · § 14c UStG, § 22 Abs. 2 Nr. 4 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Risikofälle nach § 14c UStG vor der Ausstellung erkannt | erfüllt | `service.ensureNoUnlawfulTax` |
| Eigenes Konto und eigener Steuerschlüssel | erfüllt | Konto 3851, Steuerschlüssel `accounting.TaxKeyUnlawful` |
| Betrag fließt in die Voranmeldung | erfüllt | Kennziffer 69 |
