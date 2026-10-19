# F. Umsatzsteuer, Aufzeichnung und Meldungen

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Pflichten und Erklärungen](../fachkonzepte/pflichten-und-erklaerungen.md)

Buchfink führt die Sollversteuerung. Warenlieferungen ins Ausland,
Fernverkäufe und die Kleinunternehmerregelung für den eigenen Mandanten sind
nicht im Umfang ([Umfang](../nutzung/umfang.md)).

## UST-01 Aufzeichnungspflichten

MUSS · § 22 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Je Voranmeldungszeitraum Entgelte nach Steuersätzen, steuerfreie Umsätze, Anzahlungen, unentgeltliche Wertabgaben ableitbar | teilweise | Kennziffern des Vordrucks USt 1 A aus den Steuerschlüsseln (`accounting/ustva.go`); Option nach § 9 UStG und Steuerfälle außerhalb des Umfangs fehlen |
| Bei Istversteuerung vereinnahmte Entgelte | nicht im Umfang | Die Istversteuerung wird beim Buchen abgewiesen (`service.JournalService.Post`) |
| Verzeichnis nach § 15a UStG gesondert | erfüllt | `service.InputTaxService.Register`; Konsignationslager sind nicht im Umfang |
| Jede Kennziffer bis zur Buchung auflösbar | erfüllt | Buchungen je Kennziffer an der Zeile von `domain.VatReturn` |

## UST-02 Soll- und Istversteuerung

MUSS · §§ 13, 16, 20 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Besteuerungsart je Mandant einstellbar, Wechsel zum Jahreswechsel, Umsatzgrenze überwacht | nicht im Umfang | Nur Sollversteuerung |
| Bei Sollversteuerung steuert das Leistungsdatum den Zeitraum | erfüllt | `accounting.VatPeriodFor` |
| Anzahlungen versteuert im Zeitraum der Vereinnahmung | erfüllt | Mindest-Istversteuerung in `accounting.VatPeriodFor` |

## UST-03 Umsatzsteuer-Voranmeldung

MUSS · § 18 UStG, §§ 46 bis 48 UStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Zeitraum aus der Vorjahressteuer vorgeschlagen, überschreibbar | erfüllt | `service.VatReturnService.SuggestPeriodType` |
| Zeitraum im Gründungsjahr und im Folgejahr nach § 18 Abs. 2 Sätze 4 und 6 UStG | teilweise | `accounting.RecommendedVatPeriod`: ab 2027 monatlich; 2021 bis 2026 im Folgejahr aus der umgerechneten Steuer des Gründungsjahres. Die voraussichtliche Steuer des Gründungsjahres und Vorratsgesellschaften (Satz 5) kennt Buchfink nicht |
| Amtlicher Datensatz per Datenfernübertragung oder Export für Mein ELSTER | teilweise | Kennziffernblatt nach Vordruck 2026 und CSV (`service.VatReturnService.ExportCSV`) zum Übertragen in Mein ELSTER; keine eigene Übermittlung |
| Übermittlung mit Zeitpunkt, Transferticket und Status gespeichert | erfüllt | `service.VatReturnService.ConfirmSubmitted`, nur in der App |
| Dauerfristverlängerung mit Sondervorauszahlung und Anrechnung | teilweise | Beides trägt der Mensch in der App ein; die letzte Voranmeldung des Jahres rechnet die Sondervorauszahlung in Kennziffer 39 an. Den Vorschlag des Betrags rechnet `service.VatReturnService.SuggestedSpecialPrepayment`, weder App noch MCP rufen ihn ab |
| Berichtigte Anmeldung gekennzeichnet und verknüpft | erfüllt | `service.VatReturnService.CreateCorrection` |
| Übermittlung erst nach Festschreibung des Zeitraums | erfüllt | Bestätigung setzt den festgeschriebenen Zeitraum und den aktuellen Journalstand voraus |

## UST-04 Zusammenfassende Meldung

MUSS* · § 18a UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| ZM aus den Buchungen | erfüllt | `accounting.ZMMovements`, sonstige Leistungen nach § 3a Abs. 2 UStG (Meldeart S) |
| Lieferungen, Dreiecksgeschäfte, 50.000-Euro-Grenze und Monatsmeldung | nicht im Umfang | Keine Warenlieferungen ins übrige Gemeinschaftsgebiet |
| Sonstige Leistungen quartalsweise | erfüllt | `accounting.ZMPeriodsOfYear` |
| Summen stimmen mit Kennziffer 21 der Voranmeldung überein, Abweichung angezeigt | erfüllt | Abstimmung in `service.ZMService` |
| Berichtigung möglich und protokolliert | teilweise | Übermittlung protokolliert (`service.ZMService.ConfirmSubmitted`); die berichtigte Meldung legt `service.ZMService.CreateCorrection` an, ohne Erfassungsweg in App oder MCP |

## UST-05 Reverse Charge

MUSS* · § 13b UStG, § 14a Abs. 5 UStG, § 22 Abs. 2 Nr. 8 UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Steuer und Vorsteuer aus einem Steuerschlüssel | erfüllt | Steuerfälle `reverse_charge` und `reverse_charge_eu` (`domain.TaxTreatment`), Steuerschlüssel in `accounting/tax_skr04.go` |
| Freistellungsbescheinigung mit Gültigkeit, Meldung vor Ablauf | erfüllt | `domain.Contact.ExemptionCertificateState`; die Aufgabenliste nennt ablaufende Bescheinigungen |
| Betragsgrenze von 5.000 Euro nach § 13b Abs. 2 Nr. 10 und 11 UStG | nicht im Umfang | Diese Fälle (Mobilfunkgeräte, Metalle) bildet Buchfink nicht ab |
| Umsätze nach § 13b UStG getrennt aufgezeichnet und in den richtigen Kennziffern | erfüllt | Kennziffern 46/47 und 84/85 getrennt nach Abs. 1 und Abs. 2 |

## UST-06 Innergemeinschaftliche Lieferung und USt-IdNr.

MUSS* · §§ 4 Nr. 1 lit. b, 6a, 18e UStG, §§ 17a bis 17d UStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Qualifizierte Bestätigungsabfrage beim BZSt, Ergebnis dauerhaft gespeichert | erfüllt | `vatid.Client.Check`, `domain.VatIDCheck` |
| Prüfung vor jeder Leistung an Unternehmer im übrigen Gemeinschaftsgebiet | erfüllt | `service.VatIDService.EnsureConfirmed` beim Ausstellen |
| Fehlende oder negative Bestätigung blockiert oder verlangt eine begründete Übersteuerung | erfüllt | Prüfregel `vat_id_unconfirmed` nennt die übersteuerten Rechnungen |
| Belegnachweis und Bericht zu steuerfreien Lieferungen | nicht im Umfang | Keine innergemeinschaftlichen Lieferungen |

## UST-07 Vorsteuerabzug und Vorsteuerberichtigung

MUSS · §§ 15, 15a UStG, § 44 UStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Vorsteuerabzug an eine geprüfte Rechnung gebunden | erfüllt | Prüfung der Rechnungsangaben beim Buchen (`service.PostingService`) |
| Gemischt genutzte Güter mit Aufteilungsmaßstab und Begründung | erfüllt | abziehbarer Anteil in Promille je Position, Maßstab Pflicht |
| Verzeichnis berichtigungspflichtiger Wirtschaftsgüter | teilweise | `service.InputTaxService.Register`, in der App sichtbar; den Verwendungsanteil der Folgejahre nimmt `service.InputTaxService.SaveUsage` auf, ohne Erfassungsweg in App oder MCP |
| Bagatellgrenzen des § 44 UStDV | erfüllt | `accounting.AssessInputTaxCorrection` |
| Berichtigung fließt in Voranmeldung und Jahreserklärung | teilweise | Die App zeigt den Berichtigungsbetrag des Jahres (`service.InputTaxService.Year`); die Buchung rechnet `service.InputTaxService.BookYear`, ohne Erfassungsweg in App oder MCP |

## UST-08 OSS und IOSS

MUSS* · §§ 18i, 18j, 18k UStG, § 3c UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Fernverkäufe je Bestimmungsland, Lieferschwelle, OSS-Meldung, Aufzeichnung | nicht im Umfang | Kein Fernverkauf an Verbraucher; nur die deutschen Steuersätze |

## UST-09 Kleinunternehmerregelung

MUSS* · §§ 19, 19a UStG, § 34a UStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Grenzen überwacht, Wechsel zur Regelbesteuerung, Rechnung nach § 34a UStDV, Verzicht nach § 19 Abs. 3 UStG | nicht im Umfang | Der eigene Mandant ist nie Kleinunternehmer; bei Lieferanten führt `domain.Contact.IsSmallBusiness` den Status |
