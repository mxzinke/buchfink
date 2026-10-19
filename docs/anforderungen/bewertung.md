# G. Bewertung, Anlagen, Fremdwährung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Anlagen](../fachkonzepte/anlagenverwaltung.md) ·
[Jahresabschluss](../fachkonzepte/jahresabschluss.md)

Buchfink führt eine Einheitsbilanz: ein Wertansatz, abweichende steuerliche
Werte nur, wo das Gesetz sie erzwingt oder ein steuerliches Wahlrecht genutzt
wird, und dann am Gegenstand mitgeführt.

## BEW-01 Bewertungsgrundsätze

MUSS · § 252 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Eröffnungswerte gleich Schlusswerte des Vorjahres | erfüllt | `service.ClosingService.CarryForward`, Abweichung in `service.ClosingService.CarryForwardState` |
| Bewertungsmethoden je Bilanzposition hinterlegt, Änderung begründet | teilweise | Methode je Anlagegut; Verfahren der Rechnungsabgrenzung als Einstellung; Anhangtext Bilanzierungs- und Bewertungsmethoden (`domain.NotesSectionMethods`). Keine Begründungspflicht beim Wechsel |
| Wertansatz je Wirtschaftsgut, Sammelbewertung gekennzeichnet | teilweise | Anlagen einzeln, Sammelposten gekennzeichnet; Vorräte sind nicht im Umfang |
| Bericht über geänderte Bewertungsmethoden | offen | – |

## BEW-02 Anschaffungs- und Herstellungskosten

MUSS · § 255 HGB, § 6 Abs. 1 Nr. 1a EStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Nebenkosten und nachträgliche Kosten zuordenbar, ändern die Basis ab Zuordnung | erfüllt | `service.AssetService.RecordCostAdjustment` |
| Preisminderungen mindern die Anschaffungskosten | erfüllt | eigene Bewegungsart mit Notiz |
| Pflicht- und Wahlbestandteile der Herstellungskosten getrennt | offen | nur ein Gesamtbetrag |
| Anschaffungsnahe Herstellungskosten bei Gebäuden (15 % in drei Jahren) | nicht im Umfang | Erhaltungsaufwand am Anlagegut ist nicht abgebildet |
| Handels- und steuerrechtlicher Wert parallel, wo sie abweichen | teilweise | Sonderabschreibung nach § 7g Abs. 5 EStG am Anlagegut; sonst Einheitsbilanz |

## BEW-03 Anlagenbuchhaltung und Anlagenspiegel

MUSS · § 253 Abs. 3 HGB, § 284 Abs. 3 HGB, § 5b Abs. 1 EStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Je Wirtschaftsgut Bezeichnung, Nummer, Datum, Kosten, Nutzungsdauer, Methode, Abschreibungen, Buchwert, Abgang | erfüllt | `domain.FixedAsset`; Abschreibung und Buchwert aus den Bewegungen |
| Anlagenspiegel mit allen Spalten und Vorjahresvergleich | teilweise | `service.AssetService.Anlagenspiegel`; ein vollständiger Vorjahresspiegel fehlt |
| Handels- und steuerrechtlicher Anlagenspiegel getrennt | nicht im Umfang | Einheitsbilanz; § 7g über das Verzeichnis nach BEW-06 |
| Anlagenspiegel in der Struktur der E-Bilanz | offen | Die E-Bilanz enthält noch keinen Anlagenspiegel (Pflicht ab Wirtschaftsjahr 2028) |
| Anlagen im Bau und Anzahlungen als eigene Position, umbuchbar | erfüllt | `service.AssetService.Transfer` |

## BEW-04 Abschreibungen

MUSS · § 253 Abs. 3 HGB, § 7 EStG, AfA-Tabellen

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Methode je Wirtschaftsgut wählbar | teilweise | linear, degressiv, Sammelposten, Sofortabzug, Staffel nach § 7 Abs. 2a EStG, Gebäudesatz; keine Leistungsabschreibung |
| Befristete Regeln datiert, ohne Codeänderung | teilweise | `accounting/afa_rules.json`; neue Regeln kommen mit einem Programmupdate |
| Wechsel von degressiv zu linear dokumentiert | teilweise | automatisch im günstigsten Jahr, vermerkt im Plan, nicht wählbar |
| Unterschiedliche Nutzungsdauer und Methode in Handels- und Steuerbilanz | teilweise | nur die Sonderabschreibung nach § 7g Abs. 5 EStG als steuerlicher Wert |
| Außerplanmäßige Abschreibung und Zuschreibung mit Begründung, Bericht zur Wertaufholung | erfüllt | `service.AssetService.BookImpairment`, `service.AssetService.BookWriteUp`, `service.AssetService.WriteUpReport` |
| AfA-Tabellenwerte als Vorschlag mit Begründungsfeld | teilweise | Vorschlagswerte für einen Teil der Anlagekonten (`accounting/asset_accounts.go`) |

## BEW-05 Geringwertige Wirtschaftsgüter und Sammelposten

MUSS · § 6 Abs. 2, Abs. 2a EStG, R 6.13 EStR

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Wertklasse aus dem Nettobetrag, Vorschlag mit Begründung | erfüllt | `accounting.ClassifyAcquisition` |
| Wertgrenzen und Auflösungsdauer datiert | teilweise | `accounting/afa_rules.json`; Änderungen mit einem Programmupdate |
| Verzeichnis für Wirtschaftsgüter über 250 Euro netto | erfüllt | auch der Sofortabzug bleibt im Anlagenverzeichnis |
| Wahlrecht je Wirtschaftsjahr einheitlich | erfüllt | `service.AssetService.PoolConsistency` |
| Abgang aus dem Sammelposten mindert ihn nicht | erfüllt | `service.AssetService.Dispose` weist den Abgang ab |

## BEW-06 Verzeichnis steuerlicher Wahlrechte

MUSS · § 5 Abs. 1 S. 2 und 3 EStG, § 60 EStDV

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Verzeichnis je Wirtschaftsgut mit Tag, Kosten, Vorschrift und Abschreibungen | erfüllt | `service.TaxRegisterService` |
| Eintrag entsteht, sobald die Werte abweichen | erfüllt | jedes Anlagegut mit Sonderabschreibungssatz |
| Zu jedem Stichtag als Bericht und Datei | erfüllt | `service.TaxRegisterService.RegisterCSV` |
| Aufbewahrt wie die Bücher, übermittelt mit der E-Bilanz | teilweise | Die Überleitung geht in die E-Bilanz; das Verzeichnis selbst geht nicht mit |

## BEW-07 Rückstellungen

MUSS · §§ 249, 253 Abs. 1 und 2 HGB, § 6 Abs. 1 Nr. 3a EStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Rückstellungsarten nach § 249 HGB | erfüllt | `domain.Provision`; Steuerrückstellung als eigener Abschlussbaustein (`service.ClosingBookingService.BookTaxProvision`) |
| Erfüllungsbetrag, Restlaufzeit, Zinssatz und Barwert je Rückstellung | erfüllt | `service.ProvisionService.Preview` |
| Abzinsungssätze der Bundesbank pflegbar | erfüllt | `domain.DiscountRate` |
| Handels- und steuerrechtlicher Wert parallel | erfüllt | steuerlich 5,5 % nach § 6 Abs. 1 Nr. 3a Buchst. e EStG; die Differenz geht in die Überleitung |
| Auflösung nur mit Grund | erfüllt | `service.ProvisionService.BookRelease` |

## BEW-08 Rechnungsabgrenzung

MUSS · § 250 HGB, § 252 Abs. 1 Nr. 5 HGB, § 5 Abs. 5 EStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Aktive und passive Posten als eigene Bilanzpositionen | erfüllt | 1900 und 3900 (`domain.Accrual`) |
| Beginn, Ende, Verteilung, automatische Auflösung | erfüllt | `accounting.AccrualReleasePlanFor` |
| Bestand zu jedem Stichtag mit Restlaufzeit | erfüllt | `service.AccrualService.Report` |
| Disagio als eigener Fall | erfüllt | eigene Art an `domain.Accrual` |

## BEW-09 Inventar und Vorratsbewertung

MUSS · §§ 240, 241, 256 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Inventar zum Stichtag, mit dem Bilanzansatz abgestimmt | teilweise | Anlagenverzeichnis, offene Posten zum Stichtag und unfertige Leistungen (`service.ClosingBookingService.BookWorkInProgress`); ein Gesamtinventar entsteht außerhalb |
| Festwert, Gruppenbewertung, Lifo/Fifo, Niederstwert im Umlaufvermögen, verlegte Inventur | nicht im Umfang | Keine Vorräte |

## BEW-10 Fremdwährung

MUSS* · §§ 244, 256a HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Originalbetrag, Währung, Kurs, Quelle und Kursdatum je Buchung | erfüllt | `domain.JournalLine.ForeignAmount`; Teil des Hashs |
| Kurse aus nachvollziehbarer Quelle, gespeichert | erfüllt | EZB-Referenzkurs (`currency` Paket), `service.CurrencyService.RateAt` |
| Stichtagsbewertung, realisiert und unrealisiert getrennt | erfüllt | `service.CurrencyService.PreviewCurrencyValuation`, `service.CurrencyService.BookCurrencyValuation` |
| Sonderregel bis ein Jahr Restlaufzeit | erfüllt | Restlaufzeit am Zahlungsziel |
| Umrechnung nach § 16 Abs. 6 UStG je Beleg | teilweise | Durchschnittskurs des Monats, in dem die Leistung ausgeführt ist, bei Zahlung vor der Ausführung des Zahlungsmonats (`service.vatRateDateOf`, `service.CurrencyService.VatRateFor`, `service.CurrencyService.ImportVatRatesCSV`). Die BMF-Kurse liest der Mensch in der App als CSV-Datei ein (Einstellungen › Erweitert, Bridge `ImportVatExchangeRates`), ein Werkzeug dafür gibt es nicht; ohne Kurs des Monats rechnet Buchfink mit dem Tageskurs und sagt es. Eine Zahlung vor der Leistung erkennt Buchfink nur, wenn der Beleg mit ihr gebucht wird |

## BEW-11 Latente Steuern

MUSS* · §§ 274, 274a Nr. 4 HGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Differenzen, Steuersatz, Verlustvorträge, Anhangangaben | nicht im Umfang | Einheitsbilanz; kleine Kapitalgesellschaften sind befreit |
| Befreiung über die Größenklasse gesteuert | teilweise | `accounting.ClassifySize` bestimmt die Klasse, die Prüfregel `size_class_change` meldet einen Wechsel; latente Steuern für mittelgroße Gesellschaften rechnet Buchfink nicht |

## BEW-12 Nicht abziehbare Betriebsausgaben

MUSS · § 4 Abs. 5 und Abs. 7 EStG, § 15 Abs. 1a UStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Je Kategorie eigenes Konto | erfüllt | Buchungsgruppen in `accounting/posting_groups.go` |
| Geschenke je Empfänger und Jahr, Freigrenze 50 Euro | erfüllt | Aufzeichnung nach § 4 Abs. 7 EStG an der Buchung (`domain.GiftRecord`) |
| Bewirtung 70 / 30, Vorsteuer voll | erfüllt | datierter Anteil in `accounting.TaxParametersFor` |
| Bewirtungsbeleg mit Anlass und Teilnehmern | erfüllt | Angaben an der Buchung, Teil des Hashs |
| Bericht je Kategorie | erfüllt | `service.GiftService.NonDeductibleReport` |

## BEW-13 Schuldzinsenabzug

MUSS* · § 4 Abs. 4a EStG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Entnahmen, Einlagen, Überentnahme, Sockelbetrag, Investitionsdarlehen | nicht im Umfang | Die Vorschrift gilt für Einzelunternehmen und Personengesellschaften; Buchfink führt nur GmbH und UG |
