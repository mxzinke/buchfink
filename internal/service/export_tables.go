package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/export"
)

// Die Tabellen der Datenüberlassung.
//
// Jede Spalte trägt ihre Erläuterung mit — nicht in einer Dokumentation
// nebenan, sondern hier, wo auch der Wert entsteht. Die Feldbeschreibung wird
// daraus erzeugt; eine Spalte ohne Erläuterung fällt beim Lesen der
// Tabellendefinition auf und nicht erst dem Prüfer.
const (
	tableJournal        = "journal"
	tableEntertainment  = "bewirtungen"
	tableAllocations    = "zahlungszuordnungen"
	tableAccounts       = "konten"
	tableBalances       = "salden"
	tableContacts       = "kontakte"
	tableOpenItems      = "offene_posten"
	tableAssets         = "anlagen"
	tableAssetMovements = "anlagen_bewegungen"
	tableTaxKeys        = "steuerschluessel"
	tableKeyDirectory   = export.KeyDirectoryTable
	tableAuditLog       = "aenderungsprotokoll"
	tableReceipts       = "belege"
	tableDocuments      = "dokumente"
	tableVatReturns     = "voranmeldungen"
	tableCommits        = "festschreibungen"
	tableCheckRuns      = "pruefläufe"
)

func alphaField(name, description string) export.Field {
	return export.Field{Name: name, Type: export.FieldAlphaNumeric, Description: description}
}

func numField(name, description string) export.Field {
	return export.Field{Name: name, Type: export.FieldNumeric, Description: description}
}

func dateField(name, description string) export.Field {
	return export.Field{Name: name, Type: export.FieldDate, Description: description}
}

func newTable(name, fileName, description string, fields ...export.Field) export.Table {
	return export.Table{
		Name: name, FileName: fileName, Description: description,
		Fields: fields, Rows: make([][]string, 0),
	}
}

// --- journal ---------------------------------------------------------------

// journalTable ist die Kerntabelle: eine Zeile je Buchungszeile, mit dem Kopf
// der Buchung daneben.
//
// Kopf und Zeilen verbunden und nicht in zwei Tabellen: eine Prüfsoftware
// verknüpft zwar, aber die Grundfrage „welche Konten hat dieser Vorgang
// bewegt" soll ohne Verknüpfung zu beantworten sein. Die Wiederholung der
// Kopfdaten kostet Platz und spart einen Arbeitsschritt bei jeder Auswertung.
func journalTable(d *exportData) (export.Table, error) {
	t := newTable(tableJournal, "journal.csv",
		"Alle Buchungen des Geschäftsjahres, eine Zeile je Buchungszeile. Die Kopfdaten wiederholen sich über die Zeilen einer Buchung.",
		alphaField("Buchungsnummer", "Fortlaufende, lückenlose Nummer der Buchung im Geschäftsjahr."),
		numField("Buchung_ID", "Interne Kennung der Buchung; Verknüpfungsschlüssel der übrigen Tabellen."),
		numField("Geschaeftsjahr", "Geschäftsjahr, dem die Buchung zugeordnet ist."),
		dateField("Buchungsdatum", "Tag, der über die Periode entscheidet."),
		dateField("Belegdatum", "Datum des zugrunde liegenden Belegs (Rechnungsdatum)."),
		dateField("Leistungsbeginn", "Beginn der Leistung (§ 14 Abs. 4 Nr. 6 UStG)."),
		dateField("Leistungsende", "Ende der Leistung; gleich dem Beginn bei einer Zeitpunktleistung."),
		dateField("Valuta", "Wertstellung; nur bei Zahlungsbuchungen belegt."),
		alphaField("Buchungstext", "Beschreibung des Geschäftsvorfalls."),
		alphaField("Quelle", "Teil des Systems, der die Buchung erzeugt hat; siehe Schlüsselverzeichnis, Kategorie „Quelle“."),
		alphaField("Buchungsart", "normal oder reversal (Generalumkehr); siehe Schlüsselverzeichnis."),
		alphaField("Steuerfall", "Umsatzsteuerlicher Sachverhalt; siehe Schlüsselverzeichnis, Kategorie „Steuerfall“."),
		numField("Storno_von_ID", "Kennung der Buchung, die diese Generalumkehr aufhebt; leer sonst."),
		alphaField("Storno_Grund", "Begründung der Generalumkehr."),
		alphaField("Belegnummer", "Belegfeld: Nummer des Belegs, unter der er abgelegt ist."),
		alphaField("Beleg_SHA256", "Prüfsumme über die geordnete Dateiliste des Belegs."),
		numField("Kontakt_ID", "Geschäftspartner der Buchung; verweist auf kontakte.Kontakt_ID."),
		numField("Bankumsatz_ID", "Interne Kennung des zugeordneten Bankumsatzes."),
		alphaField("Waehrung", "Währung des Belegs nach ISO 4217."),
		numField("Kurs_Millionstel", "Umrechnungskurs in Millionsteln; 1000000 bedeutet Euro."),
		alphaField("Kursquelle", "Herkunft des Kurses."),
		dateField("Kursdatum", "Tag, für den der Kurs gilt."),
		alphaField("Regelversion", "Fassung der Kontierungsregeln, nach der gebucht wurde."),
		alphaField("Erfassungszeitpunkt_UTC", "Zeitpunkt der Erfassung nach RFC 3339 in UTC; Bestandteil der Hash-Chain."),
		alphaField("Vorgaengerhash", "Eigenhash der vorhergehenden Buchung desselben Geschäftsjahres."),
		alphaField("Eigenhash", "SHA-256 über die kanonische Form der Buchung; siehe Abschnitt „Die Hash-Chain nachrechnen“."),
		dateField("Festgeschrieben_am", "Tag der Festschreibung des Zeitraums, in den die Buchung fällt; leer, solange sie nicht festgeschrieben ist."),
		numField("Zeilennummer", "Position der Zeile innerhalb der Buchung."),
		alphaField("Seite", "S für Soll, H für Haben."),
		alphaField("Konto", "Sachkonto (vier Stellen) oder Personenkonto (fünf Stellen)."),
		alphaField("Kontoname", "Bezeichnung des Kontos zum Zeitpunkt des Exports."),
		numField("Betrag", "Betrag der Zeile in Euro. Bei einer Generalumkehr negativ."),
		numField("Betrag_Cent", "Derselbe Betrag in ganzzahligen Cent; dieser Wert geht in die Hash-Chain ein."),
		alphaField("Steuerschluessel", "Schlüssel der Steuerzeile; siehe Tabelle steuerschluessel."),
		numField("Bemessungsgrundlage", "Entgelt, aus dem die Steuer dieser Zeile gerechnet wurde, in Euro."),
		numField("Bemessungsgrundlage_Cent", "Dieselbe Bemessungsgrundlage in Cent; dieser Wert geht in die Hash-Chain ein."),
		numField("Zeile_Kontakt_ID", "Geschäftspartner der Zeile; belegt auf Personenkonten."),
		alphaField("Zeilentext", "Text der einzelnen Zeile."),
	)

	for i := range d.entries {
		e := &d.entries[i]
		lines := sortedLines(e)
		committed := d.committedOn(e)
		for _, l := range lines {
			err := t.AddRow(
				e.EntryNumber,
				export.Uint(e.ID),
				export.Int(e.FiscalYear),
				e.BookingDate,
				e.DocumentDate,
				e.ServiceDateFrom,
				e.ServiceDateTo,
				e.ValueDate,
				e.Description,
				string(e.Source),
				string(e.Kind),
				string(e.TaxTreatment),
				export.OptUint(e.ReversalOfID),
				e.ReversalReason,
				e.DocumentNumber,
				e.ReceiptHash,
				export.OptUint(e.ContactID),
				export.OptUint(e.BankTxID),
				e.Currency,
				export.Int64(e.ExchangeRateMicros),
				e.ExchangeRateSource,
				e.ExchangeRateDate,
				e.PostingRuleVersion,
				e.CreatedAt.UTC().Format(time.RFC3339),
				e.PreviousHash,
				e.EntryHash,
				committed,
				export.Int(l.Position),
				string(l.Side),
				l.Account,
				d.accountName(l.Account),
				export.Amount(l.Amount),
				export.Int64(int64(l.Amount)),
				l.TaxKey,
				export.Amount(l.TaxBase),
				export.Int64(int64(l.TaxBase)),
				export.OptUint(l.ContactID),
				l.Text,
			)
			if err != nil {
				return t, err
			}
		}
	}
	return t, nil
}

// sortedLines liefert die Zeilen in derselben Reihenfolge, in der sie in die
// Hash-Chain eingehen. Ohne diese Sortierung ließe sich der Eigenhash aus der
// Datei nicht nachrechnen.
func sortedLines(e *domain.JournalEntry) []domain.JournalLine {
	lines := make([]domain.JournalLine, len(e.Lines))
	copy(lines, e.Lines)
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Position < lines[j].Position })
	return lines
}

// entertainmentTable trägt die Aufzeichnung des § 4 Abs. 5 Satz 1 Nr. 2 EStG.
//
// Sie steht in einer eigenen Tabelle und nicht im Journal, weil sie an der
// Buchung hängt und nicht an der Zeile: im Journal stünde sie so oft, wie die
// Buchung Zeilen hat. Für das Nachrechnen der Hash-Chain wird sie gebraucht.
func entertainmentTable(d *exportData) (export.Table, error) {
	t := newTable(tableEntertainment, "bewirtungen.csv",
		"Aufzeichnungen zu Bewirtungsaufwendungen (§ 4 Abs. 5 Satz 1 Nr. 2 EStG). Sie sind Bestandteil der kanonischen Form der Buchung.",
		numField("Buchung_ID", "Buchung, zu der die Aufzeichnung gehört."),
		alphaField("Buchungsnummer", "Nummer derselben Buchung."),
		alphaField("Ort", "Ort der Bewirtung."),
		dateField("Tag", "Tag der Bewirtung."),
		alphaField("Teilnehmer", "Bewirtete Personen."),
		alphaField("Anlass", "Anlass der Bewirtung."),
	)
	for i := range d.entries {
		e := &d.entries[i]
		if e.Entertainment == nil {
			continue
		}
		if err := t.AddRow(
			export.Uint(e.ID), e.EntryNumber,
			e.Entertainment.Place, e.Entertainment.Day,
			e.Entertainment.Participants, e.Entertainment.Occasion,
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

// allocationsTable ist die Einzelpostenliste zu jeder Zahlung.
//
// Eine Sammelüberweisung gleicht drei Rechnungen aus; aus dem Journal allein
// ist danach nicht mehr zu sehen, welche. Erst diese Tabelle macht die Zahlung
// nachvollziehbar.
func allocationsTable(d *exportData) (export.Table, error) {
	t := newTable(tableAllocations, "zahlungszuordnungen.csv",
		"Zuordnung von Zahlungen zu offenen Posten. Eine Zahlung kann mehrere Posten ausgleichen und ein Posten durch mehrere Zahlungen ausgeglichen werden.",
		numField("Zuordnung_ID", "Interne Kennung der Zuordnung."),
		numField("Posten_Buchung_ID", "Buchung, die den offenen Posten begründet hat."),
		alphaField("Posten_Buchungsnummer", "Nummer dieser Buchung."),
		numField("Zahlung_Buchung_ID", "Buchung der Zahlung."),
		alphaField("Zahlung_Buchungsnummer", "Nummer der Zahlungsbuchung."),
		numField("Bankumsatz_ID", "Zugeordneter Bankumsatz, sofern vorhanden."),
		numField("Kontakt_ID", "Geschäftspartner des Postens."),
		numField("Ausgleichsbetrag", "Betrag, um den der offene Posten sinkt, in Euro."),
		numField("Zahlbetrag", "Betrag, der tatsächlich über das Geldkonto geflossen ist, in Euro."),
		alphaField("Differenzart", "Grund der Abweichung; siehe Schlüsselverzeichnis, Kategorie „Differenzart“."),
		numField("Differenzbetrag", "Höhe der Abweichung in Euro."),
	)
	for _, a := range d.allocations {
		if err := t.AddRow(
			export.Uint(a.ID),
			export.Uint(a.OpenItemEntryID), d.entryNumber(a.OpenItemEntryID),
			export.Uint(a.PaymentEntryID), d.entryNumber(a.PaymentEntryID),
			export.OptUint(a.BankTxID),
			export.Uint(a.ContactID),
			export.Amount(a.SettledAmount),
			export.Amount(a.CashAmount),
			string(a.DifferenceKind),
			export.Amount(a.DifferenceAmount),
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

// --- Stammdaten ------------------------------------------------------------

func accountsTable(d *exportData) (export.Table, error) {
	t := newTable(tableAccounts, "konten.csv",
		"Der Kontenrahmen mit der Zuordnung zur Bilanz- und GuV-Gliederung.",
		alphaField("Konto", "Kontonummer."),
		alphaField("Name", "Bezeichnung des Kontos."),
		alphaField("Typ", "Kontoart; siehe Schlüsselverzeichnis, Kategorie „Kontoart“."),
		numField("Kontenklasse", "Klasse 0 bis 9 des Kontenrahmens."),
		alphaField("Kontenklasse_Name", "Bezeichnung der Kontenklasse."),
		alphaField("Kategorie", "Grobe Zuordnung, etwa „Umlaufvermögen“."),
		alphaField("Unterkategorie", "Feinere Zuordnung innerhalb der Kategorie."),
		alphaField("HGB_Position", "Posten der Bilanz oder GuV nach HGB, etwa „Aktiva.B.IV“."),
		alphaField("Gliederungsposition", "Schlüssel der Gliederungsposition in Buchfink."),
		alphaField("Posten", "Wortlaut des Postens der Gliederung."),
		alphaField("Taxonomie_Element", "Element der HGB-Taxonomie, unter dem der Posten in die E-Bilanz geht; leer, wo die Gliederungsposition keinem Element zugeordnet ist."),
		alphaField("Bilanzseite", "Aktiva, Passiva, GuV oder Statistisch."),
		alphaField("Abschlussart", "Bilanz, GuV oder Statistisch."),
		numField("Steuersatz", "Hinterlegter Steuersatz in Prozent; 0.00, wo keiner hinterlegt ist."),
		numField("Aktiv", "1, wenn das Konto bebucht werden darf, sonst 0."),
	)
	for _, a := range d.accounts {
		if a.IsRange {
			// Ein Bereichskonto ist kein Konto, sondern eine Kurzschreibweise
			// für zehn. Es kann nicht bebucht werden und gehört deshalb nicht
			// in die Liste der Konten.
			continue
		}
		if err := t.AddRow(
			a.Number, a.Name, string(a.Type),
			export.Int(a.Kontenklasse), a.KontenklasseName,
			a.Category, a.Subcategory,
			a.HGBCode, a.PositionID, a.Posten, taxonomyElement(a),
			a.BalanceSide, a.StatementType,
			fmt.Sprintf("%.2f", a.TaxRate*100),
			export.Bool(a.IsActive),
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

// taxonomyElement liefert das Element der HGB-Taxonomie zu einem Konto.
//
// Der Weg geht über die Gliederungsposition und nicht unmittelbar über die
// SKR04-Position: die Taxonomie kennt Schlüssel wie „aktiva.B.IV", die Konten
// tragen Schlüssel wie „bilanz.aktiva_b_iv.kassenbestand…". Dazwischen steht
// dieselbe Übersetzung, die auch die E-Bilanz benutzt — zwei Wege zu demselben
// Element wären zwei Gelegenheiten, verschiedene Auskünfte zu geben.
//
// Leer, wo es keine Zuordnung gibt: eine erfundene Angabe wäre schlimmer als
// eine fehlende, denn der Prüfer sähe eine E-Bilanz-Zuordnung, die es nicht gibt.
func taxonomyElement(account domain.Account) string {
	key, ok := accounting.StatementKeyForAccount(account)
	if !ok {
		return ""
	}
	element, ok := ebilanz.ElementFor(key)
	if !ok {
		return ""
	}
	return element.Element
}

// balancesTable ist die Summen- und Saldenliste zum Stichtag.
//
// Der Anfangsbestand wird aus den Vortragsbuchungen gewonnen und nicht aus dem
// Vorjahr gelesen: maßgeblich ist, was in diesem Geschäftsjahr gebucht wurde.
// Steht kein Vortrag in den Büchern, ist der Anfangsbestand null — und das ist
// die richtige Auskunft, nicht eine aus dem Vorjahr geborgte.
func balancesTable(d *exportData) (export.Table, error) {
	t := newTable(tableBalances, "salden.csv",
		"Verkehrszahlen und Salden je Konto zum Ende des Geschäftsjahres.",
		alphaField("Konto", "Kontonummer."),
		alphaField("Name", "Bezeichnung des Kontos."),
		dateField("Stichtag", "Tag, bis zu dem gerechnet wurde."),
		numField("Anfangsbestand", "Saldo aus den Vortragsbuchungen des Geschäftsjahres, in Euro."),
		numField("Soll", "Summe der Sollbuchungen ohne Vortrag, in Euro."),
		numField("Haben", "Summe der Habenbuchungen ohne Vortrag, in Euro."),
		numField("Schlusssaldo", "Anfangsbestand zuzüglich Soll abzüglich Haben, in Euro. Ein Habensaldo ist negativ."),
		numField("Buchungen", "Anzahl der Buchungszeilen auf diesem Konto."),
	)

	accounts := make([]string, 0, len(d.balances))
	for account := range d.balances {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)

	for _, account := range accounts {
		b := d.balances[account]
		if err := t.AddRow(
			account, d.accountName(account), d.to,
			export.Amount(b.opening),
			export.Amount(b.debit),
			export.Amount(b.credit),
			export.Amount(b.opening+b.debit-b.credit),
			export.Int(b.count),
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

func contactsTable(d *exportData) (export.Table, error) {
	t := newTable(tableContacts, "kontakte.csv",
		"Debitoren und Kreditoren mit ihren Personenkonten. Personenbezogene Angaben stehen im Klartext, wie es § 147 Abs. 6 AO für die Datenüberlassung verlangt.",
		numField("Kontakt_ID", "Interne Kennung; Verknüpfungsschlüssel des Journals."),
		alphaField("Personenkonto", "Debitoren- (10000–69999) oder Kreditorenkonto (70000–99999)."),
		alphaField("Art", "customer für Debitoren, vendor für Kreditoren; siehe Schlüsselverzeichnis."),
		alphaField("Name", "Name des Geschäftspartners."),
		alphaField("Firma", "Firmenname, sofern abweichend."),
		alphaField("Anschrift", "Anschrift des Geschäftspartners."),
		alphaField("Land", "Ländercode nach ISO 3166-1 alpha-2."),
		alphaField("USt_IdNr", "Umsatzsteuer-Identifikationsnummer."),
		alphaField("Steuernummer", "Steuernummer des Geschäftspartners."),
		alphaField("Email", "Kontaktadresse."),
		alphaField("IBAN", "Bankverbindung."),
		numField("Zahlungsziel_Tage", "Vereinbartes Zahlungsziel in Tagen."),
		alphaField("Sammelkonto", "Konto, auf dem die Posten dieses Partners in der Bilanz zusammengefasst werden."),
	)
	for i := range d.contacts {
		c := &d.contacts[i]
		if err := t.AddRow(
			export.Uint(c.ID), c.LedgerAccount, string(c.Type),
			c.Name, c.Company, c.Address, c.CountryCode,
			c.VatID, c.TaxID, c.Email, c.IBAN,
			export.Int(c.PaymentTermsDays), c.CollectiveAccount(),
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

func openItemsTable(d *exportData) (export.Table, error) {
	t := newTable(tableOpenItems, "offene_posten.csv",
		"Forderungen und Verbindlichkeiten, die zum Stichtag noch offen waren.",
		dateField("Stichtag", "Tag, zu dem die Posten ermittelt wurden."),
		numField("Buchung_ID", "Buchung, die den Posten begründet."),
		alphaField("Buchungsnummer", "Nummer dieser Buchung."),
		numField("Kontakt_ID", "Geschäftspartner."),
		alphaField("Kontakt_Name", "Name des Geschäftspartners."),
		alphaField("Personenkonto", "Konto, auf dem der Posten steht."),
		alphaField("Belegnummer", "Belegfeld der zugrunde liegenden Rechnung."),
		dateField("Belegdatum", "Datum der Rechnung."),
		dateField("Faelligkeit", "Tag, an dem der Posten fällig wird."),
		numField("Bruttobetrag", "Ursprünglicher Betrag des Postens in Euro."),
		numField("Ausgeglichen", "Bis zum Stichtag ausgeglichener Anteil in Euro."),
		numField("Offen", "Zum Stichtag offener Rest in Euro."),
		numField("Steuersatz", "Steuersatz der Rechnung in Prozent."),
		alphaField("Steuerfall", "Steuerfall der Rechnung; siehe Schlüsselverzeichnis."),
	)
	for i := range d.openItems {
		o := &d.openItems[i]
		if err := t.AddRow(
			d.to,
			export.Uint(o.EntryID), o.EntryNumber,
			export.Uint(o.ContactID), o.ContactName, o.LedgerAccount,
			o.DocumentNumber, o.DocumentDate, o.DueDate,
			export.Amount(o.GrossAmount), export.Amount(o.SettledAmount), export.Amount(o.OpenAmount),
			export.Rate(o.TaxRate), string(o.TaxTreatment),
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

// --- Anlagen ---------------------------------------------------------------

func assetsTable(d *exportData) (export.Table, error) {
	t := newTable(tableAssets, "anlagen.csv",
		"Die Anlagenkartei: Stammdaten der Wirtschaftsgüter des Anlagevermögens.",
		numField("Anlage_ID", "Interne Kennung."),
		alphaField("Inventarnummer", "Nummer, unter der das Wirtschaftsgut geführt wird."),
		alphaField("Bezeichnung", "Name des Wirtschaftsguts."),
		alphaField("Beschreibung", "Ergänzende Beschreibung."),
		alphaField("Klasse", "Anlagenklasse; siehe Schlüsselverzeichnis, Kategorie „Anlagenklasse“."),
		alphaField("Anlagekonto", "Bestandskonto der Klasse 0."),
		alphaField("AfA_Konto", "Aufwandskonto der planmäßigen Abschreibung."),
		dateField("Anschaffungsdatum", "Tag der Anschaffung oder Herstellung."),
		dateField("Betriebsbereit_ab", "Tag, ab dem abgeschrieben wird; leer bedeutet: ab der Anschaffung."),
		numField("Anschaffungskosten", "Zugangswert in Euro."),
		alphaField("AfA_Methode", "Abschreibungsverfahren; siehe Schlüsselverzeichnis."),
		numField("Nutzungsdauer_Monate", "Betriebsgewöhnliche Nutzungsdauer in Monaten."),
		numField("Sammelposten_Jahr", "Wirtschaftsjahr des Sammelpostens nach § 6 Abs. 2a EStG; 0 sonst."),
		numField("Sonder_AfA_Promille", "In Anspruch genommene Sonderabschreibung nach § 7g Abs. 5 EStG in Promille."),
		numField("Sonder_AfA_Jahre", "Zahl der Jahre, auf die die Sonderabschreibung verteilt wird."),
		numField("Kontakt_ID", "Lieferant des Wirtschaftsguts."),
		numField("Zugangsbuchung_ID", "Buchung des Zugangs."),
		dateField("Abgangsdatum", "Tag des Abgangs; leer, solange das Gut im Bestand ist."),
		alphaField("Abgangsart", "Art des Abgangs; siehe Schlüsselverzeichnis."),
		numField("Abgangserloes", "Erlös aus dem Abgang in Euro."),
		numField("Abgangsbuchung_ID", "Buchung des Abgangs."),
		alphaField("Notizen", "Freitext zum Wirtschaftsgut."),
	)
	for i := range d.assets {
		a := &d.assets[i]
		if err := t.AddRow(
			export.Uint(a.ID), a.InventoryNumber, a.Name, a.Description,
			string(a.Class), a.Account, a.DepreciationAccount,
			a.AcquisitionDate, a.InServiceDate,
			export.Amount(a.AcquisitionCost),
			string(a.Method), export.Int(a.UsefulLifeMonths), export.Int(a.PoolYear),
			export.Int(a.SpecialPermille), export.Int(a.SpecialYears),
			export.OptUint(a.ContactID), export.OptUint(a.AcquisitionEntryID),
			a.DisposalDate, string(a.DisposalKind), export.Amount(a.DisposalProceeds),
			export.OptUint(a.DisposalEntryID), a.Notes,
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

func assetMovementsTable(d *exportData) (export.Table, error) {
	t := newTable(tableAssetMovements, "anlagen_bewegungen.csv",
		"Zugänge, Abschreibungen, Zuschreibungen und Abgänge je Wirtschaftsgut. Aus ihnen ergibt sich der Anlagenspiegel.",
		numField("Bewegung_ID", "Interne Kennung."),
		numField("Anlage_ID", "Wirtschaftsgut, zu dem die Bewegung gehört."),
		alphaField("Inventarnummer", "Nummer desselben Wirtschaftsguts."),
		alphaField("Art", "Bewegungsart; siehe Schlüsselverzeichnis, Kategorie „Anlagenbewegung“."),
		dateField("Datum", "Tag der Bewegung."),
		numField("Geschaeftsjahr", "Geschäftsjahr, in dem die Bewegung ausgewiesen wird."),
		alphaField("Konto", "Anlagekonto, dem die Bewegung zugeordnet ist."),
		numField("AHK_Veraenderung", "Veränderung der Anschaffungs- und Herstellungskosten in Euro."),
		numField("AfA_Veraenderung", "Veränderung der kumulierten Abschreibungen in Euro."),
		numField("Steuerbetrag", "Betrag, der nur steuerlich zählt, etwa die Vorabpauschale, in Euro."),
		numField("Buchung_ID", "Buchung, die die Bewegung trägt."),
		alphaField("Begruendung", "Begründung, bei außerplanmäßigen Vorgängen Pflicht."),
	)
	for i := range d.assets {
		a := &d.assets[i]
		for j := range a.Movements {
			m := &a.Movements[j]
			account := m.Account
			if account == "" {
				account = a.Account
			}
			if err := t.AddRow(
				export.Uint(m.ID), export.Uint(a.ID), a.InventoryNumber,
				string(m.Kind), m.Date, export.Int(m.FiscalYear), account,
				export.Amount(m.CostAmount), export.Amount(m.DepreciationAmount),
				export.Amount(m.TaxAmount),
				export.OptUint(m.JournalEntryID), m.Note,
			); err != nil {
				return t, err
			}
		}
	}
	return t, nil
}

// --- Verzeichnisse ---------------------------------------------------------

func taxKeysTable() (export.Table, error) {
	t := newTable(tableTaxKeys, "steuerschluessel.csv",
		"Die Steuerschlüssel des Journals mit Konto, Steuerfall, Satz und der Kennziffer der Umsatzsteuer-Voranmeldung.",
		alphaField("Schluessel", "Wert der Spalte Steuerschluessel im Journal."),
		alphaField("Bedeutung", "Klartext des Schlüssels."),
		alphaField("Konto", "Steuerkonto, auf das die Zeile bucht."),
		alphaField("Richtung", "incoming für Eingangs-, outgoing für Ausgangsbelege."),
		alphaField("Steuerfall", "Steuerfall, aus dem der Schlüssel entsteht."),
		numField("Satz", "Steuersatz in Prozent."),
		alphaField("Seite", "S für Soll, H für Haben."),
		alphaField("UStVA_Kennziffer", "Kennziffer des Vordrucks USt 1 A."),
		alphaField("UStVA_Zeile", "Wortlaut dieser Zeile des Vordrucks."),
	)
	for _, k := range accounting.TaxKeyCatalog() {
		if err := t.AddRow(
			k.Key, k.Label, k.Account, string(k.Direction), string(k.Treatment),
			export.Rate(k.Rate), string(k.Side), k.VatCode, k.VatCodeLabel,
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

// keyDirectoryTable ist das Schlüsselverzeichnis nach GoBD Rz. 95.
//
// Es ist die eine Quelle: dieselbe Tabelle geht in die Überlassung, in die
// Feldbeschreibung und in den Einzelexport „Schlüsselverzeichnis“. Drei Listen
// desselben Inhalts wären drei Gelegenheiten, einen Code zu vergessen.
func keyDirectoryTable() (export.Table, error) {
	t := newTable(tableKeyDirectory, "schluesselverzeichnis.csv",
		"Alle in den Tabellen verwendeten Codes mit ihrer Bedeutung im Klartext (GoBD Rz. 95).",
		alphaField("Kategorie", "Bereich, in dem der Code vorkommt."),
		alphaField("Schluessel", "Der Code, wie er in den Daten steht."),
		alphaField("Bedeutung", "Bedeutung im Klartext."),
		alphaField("Erlaeuterung", "Ergänzende Erklärung, sofern nötig."),
	)

	add := func(category, key, label, hint string) error {
		return t.AddRow(category, key, label, hint)
	}

	for _, info := range domain.TaxTreatments(domain.DirectionIncoming) {
		if err := add("Steuerfall (Eingang)", string(info.Treatment), info.Label, info.Hint); err != nil {
			return t, err
		}
	}
	for _, info := range domain.TaxTreatments(domain.DirectionOutgoing) {
		if err := add("Steuerfall (Ausgang)", string(info.Treatment), info.Label, info.Hint); err != nil {
			return t, err
		}
	}
	for _, g := range accounting.PostingGroups("") {
		if err := add("Buchungsgruppe", g.Key, g.Label, g.Hint); err != nil {
			return t, err
		}
	}
	for _, k := range accounting.TaxKeyCatalog() {
		if err := add("Steuerschlüssel", k.Key, k.Label, "Kennziffer "+k.VatCode); err != nil {
			return t, err
		}
	}
	for _, r := range []struct{ key, label, hint string }{
		{string(domain.ReceiptRoleOriginal), "Empfangene Originaldatei", "Die Datei in der Form, in der sie eingegangen ist (GoBD Rz. 131)."},
		{string(domain.ReceiptRoleStructured), "Strukturierter Rechnungsdatensatz", "Der maschinenlesbare Teil einer E-Rechnung."},
		{string(domain.ReceiptRoleRendering), "Erzeugte Darstellung", "Eine von Buchfink erzeugte, ansehbare Fassung."},
		{string(domain.ReceiptRoleAttachment), "Anlage", "Eigenbeleg, Teilnehmerliste, Lieferschein oder Zahlungsnachweis."},
	} {
		if err := add("Belegrolle", r.key, r.label, r.hint); err != nil {
			return t, err
		}
	}
	for _, s := range []struct{ key, label, hint string }{
		{string(domain.ReceiptStatusFiled), "Abgelegt", "Der Beleg liegt vor, ist aber noch nicht gebucht."},
		{string(domain.ReceiptStatusSealed), "Gebucht", "Der Beleg ist gebucht; seine Dateiliste ist festgeschrieben."},
		{string(domain.ReceiptStatusDiscarded), "Verworfen", "Der Beleg wird nicht gebucht, bleibt aber erhalten."},
	} {
		if err := add("Belegstatus", s.key, s.label, s.hint); err != nil {
			return t, err
		}
	}
	for _, k := range domain.AllReceiptKinds() {
		hint := "Muss gebucht werden."
		if !k.RequiresBooking() {
			hint = "Trägt selbst keine Buchung."
		}
		if err := add("Belegart", string(k), k.Label(), hint); err != nil {
			return t, err
		}
	}
	for _, v := range []struct{ key, label string }{
		{domain.ReceivedViaUpload, "Datei vom Rechner abgelegt"},
		{domain.ReceivedViaEmail, "Per E-Mail empfangen"},
		{domain.ReceivedViaScan, "Papier eingescannt"},
		{domain.ReceivedViaSelfIssued, "Von Buchfink selbst erzeugt"},
	} {
		if err := add("Eingangsweg", v.key, v.label, ""); err != nil {
			return t, err
		}
	}
	for _, s := range []struct{ key, label string }{
		{string(domain.EntrySourceManual), "Von Hand im Journal erfasst"},
		{string(domain.EntrySourceReceipt), "Aus einem Eingangsbeleg"},
		{string(domain.EntrySourceInvoice), "Aus einer Ausgangsrechnung"},
		{string(domain.EntrySourcePayment), "Zahlung oder Ausgleich eines offenen Postens"},
		{string(domain.EntrySourceOpening), "Eröffnungsbilanz oder Saldenvortrag"},
		{string(domain.EntrySourceDepreciation), "Abschreibung"},
		{string(domain.EntrySourceClosing), "Abschlussbuchung"},
	} {
		if err := add("Quelle", s.key, s.label, ""); err != nil {
			return t, err
		}
	}
	if err := add("Buchungsart", string(domain.EntryKindNormal), "Ursprüngliche Buchung", ""); err != nil {
		return t, err
	}
	if err := add("Buchungsart", string(domain.EntryKindReversal), "Generalumkehr",
		"Dieselben Konten und Seiten wie das Original, mit negativen Beträgen — die Verkehrszahlen der berührten Konten gehen dadurch auf null zurück."); err != nil {
		return t, err
	}
	if err := add("Buchungsseite", string(domain.SideDebit), "Soll", ""); err != nil {
		return t, err
	}
	if err := add("Buchungsseite", string(domain.SideCredit), "Haben", ""); err != nil {
		return t, err
	}
	for _, k := range domain.DifferenceKinds() {
		if err := add("Differenzart", string(k.Kind), k.Label, k.Hint); err != nil {
			return t, err
		}
	}
	for _, k := range domain.AllAssetMovementKinds() {
		if err := add("Anlagenbewegung", string(k), k.Label(), ""); err != nil {
			return t, err
		}
	}
	for _, k := range domain.AllAssetDocumentKinds() {
		if err := add("Anlagendokument", string(k), k.Label(), ""); err != nil {
			return t, err
		}
	}
	for _, c := range domain.AllAssetClasses() {
		if err := add("Anlagenklasse", string(c), c.Label(), ""); err != nil {
			return t, err
		}
	}
	for _, m := range domain.AllDepreciationMethods() {
		if err := add("Abschreibungsverfahren", string(m), m.Label(), ""); err != nil {
			return t, err
		}
	}
	for _, k := range domain.AllDisposalKinds() {
		if err := add("Abgangsart", string(k), k.Label(), ""); err != nil {
			return t, err
		}
	}
	for _, s := range domain.AllFiscalYearStatuses() {
		if err := add("Abschlussstatus", string(s), s.Label(), ""); err != nil {
			return t, err
		}
	}
	for _, ct := range []struct{ key, label string }{
		{string(domain.ContactTypeCustomer), "Debitor (Kunde)"},
		{string(domain.ContactTypeVendor), "Kreditor (Lieferant)"},
	} {
		if err := add("Kontaktart", ct.key, ct.label, ""); err != nil {
			return t, err
		}
	}
	for _, at := range []struct{ key, label string }{
		{string(domain.AccountTypeAsset), "Aktivkonto"},
		{string(domain.AccountTypeLiability), "Passivkonto"},
		{string(domain.AccountTypeEquity), "Eigenkapitalkonto"},
		{string(domain.AccountTypeRevenue), "Ertragskonto"},
		{string(domain.AccountTypeExpense), "Aufwandskonto"},
		{string(domain.AccountTypeStatistical), "Statistisches Konto"},
	} {
		if err := add("Kontoart", at.key, at.label, ""); err != nil {
			return t, err
		}
	}
	for _, s := range []struct{ key, label string }{
		{string(domain.MatchStatusUnmatched), "Bankumsatz noch nicht zugeordnet"},
		{string(domain.MatchStatusMatched), "Bankumsatz gebucht"},
		{string(domain.MatchStatusIgnored), "Bankumsatz bewusst nicht gebucht"},
	} {
		if err := add("Bankabgleich", s.key, s.label, ""); err != nil {
			return t, err
		}
	}
	for _, s := range []struct{ key, label string }{
		{string(domain.AuditActionCreate), "Anlegen"},
		{string(domain.AuditActionUpdate), "Ändern"},
		{string(domain.AuditActionStorno), "Stornieren"},
		{string(domain.AuditActionImport), "Import"},
		{string(domain.AuditActionIntegrityCheck), "Integritätsprüfung"},
		{string(domain.AuditActionExport), "Export"},
	} {
		if err := add("Protokollart", s.key, s.label, ""); err != nil {
			return t, err
		}
	}
	for _, s := range []struct{ key, label, hint string }{
		{string(domain.CheckBlocking), "Blockierender Befund", "Verhindert die Festschreibung, solange er nicht behoben oder mit Begründung übergangen wird."},
		{string(domain.CheckWarning), "Hinweis", "Die Festschreibung geht durch."},
	} {
		if err := add("Befundgewicht", s.key, s.label, s.hint); err != nil {
			return t, err
		}
	}
	return t, nil
}

// --- Nachweise -------------------------------------------------------------

func auditLogTable(d *exportData) (export.Table, error) {
	t := newTable(tableAuditLog, "aenderungsprotokoll.csv",
		"Das Änderungsprotokoll: wer wann was getan hat (GoBD Rz. 34 ff.).",
		numField("Protokoll_ID", "Fortlaufende Kennung."),
		alphaField("Zeitpunkt", "Zeitpunkt des Vorgangs nach RFC 3339."),
		alphaField("Art", "Art des Vorgangs; siehe Schlüsselverzeichnis, Kategorie „Protokollart“."),
		alphaField("Objektart", "Betroffene Art von Objekt."),
		alphaField("Objekt_ID", "Kennung des betroffenen Objekts."),
		alphaField("Einzelheiten", "Beschreibung des Vorgangs."),
	)
	for i := range d.auditLog {
		a := &d.auditLog[i]
		if err := t.AddRow(
			export.Uint(a.ID), a.Timestamp.UTC().Format(time.RFC3339),
			string(a.Action), a.EntityType, a.EntityID, a.Details,
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

func receiptsTable(d *exportData) (export.Table, error) {
	t := newTable(tableReceipts, "belege.csv",
		"Die abgelegten Belege mit ihren Dateien. Eine Zeile je Datei; die Spalte Pfad_im_Export ist nur im Archivexport belegt.",
		numField("Beleg_ID", "Interne Kennung des Belegs."),
		alphaField("Belegnummer", "Nummer, unter der der Beleg geführt wird."),
		numField("Geschaeftsjahr", "Geschäftsjahr des Belegs."),
		alphaField("Richtung", "incoming für Eingangs-, outgoing für Ausgangsbelege."),
		alphaField("Belegart", "Art des Belegs; siehe Schlüsselverzeichnis, Kategorie „Belegart“."),
		alphaField("Status", "Stand des Belegs; siehe Schlüsselverzeichnis, Kategorie „Belegstatus“."),
		dateField("Eingang", "Tag, an dem der Beleg eingegangen ist."),
		alphaField("Eingangsweg", "Weg, auf dem der Beleg eingegangen ist."),
		alphaField("Beleg_SHA256", "Prüfsumme über die geordnete Dateiliste."),
		numField("Buchung_ID", "Buchung, mit der der Beleg gebucht wurde."),
		alphaField("Buchungsnummer", "Nummer dieser Buchung."),
		numField("Datei_Position", "Reihenfolge der Datei innerhalb des Belegs."),
		alphaField("Rolle", "Rolle der Datei; siehe Schlüsselverzeichnis, Kategorie „Belegrolle“."),
		alphaField("Dateiname", "Name, unter dem die Datei empfangen wurde."),
		alphaField("Dateityp", "MIME-Typ der Datei."),
		numField("Groesse_Bytes", "Größe der Datei in Bytes."),
		alphaField("Datei_SHA256", "Prüfsumme des Dateiinhalts."),
		numField("Abgeleitet", "1, wenn Buchfink die Datei aus einer anderen erzeugt hat, sonst 0."),
		alphaField("Pfad_im_Export", "Pfad der Datei innerhalb dieses Exports; leer, wenn keine Dateien beiliegen."),
	)
	for i := range d.receipts {
		r := &d.receipts[i]
		entryNumber := ""
		if r.JournalEntryID != nil {
			entryNumber = d.entryNumber(*r.JournalEntryID)
		}
		for j := range r.Files {
			f := &r.Files[j]
			if err := t.AddRow(
				export.Uint(r.ID), r.ReceiptNumber, export.Int(r.FiscalYear),
				string(r.Direction), string(r.Kind), string(r.Status),
				r.ReceivedAt, r.ReceivedVia, r.ReceiptHash,
				export.OptUint(r.JournalEntryID), entryNumber,
				export.Int(f.Position), string(f.Role), f.FileName, f.MimeType,
				export.Int64(f.Size), f.SHA256, export.Bool(f.Derived),
				d.receiptFilePaths[f.ID],
			); err != nil {
				return t, err
			}
		}
	}
	return t, nil
}

// documentsTable führt die Anlagendokumente mit ihren Prüfsummen auf.
//
// Sie stehen sonst nur in export.json, und die ist eine Metadatei über den
// Datenträger, keine Tabelle der Überlassung: ein Prüfer, der die Verträge zu
// den Wirtschaftsgütern sucht, findet sie dort nicht mit der Inventarnummer
// verbunden.
func documentsTable(d *exportData) (export.Table, error) {
	t := newTable(tableDocuments, "dokumente.csv",
		"Verträge, Gutachten und Zulassungen zu den Wirtschaftsgütern des Anlagevermögens. Die Spalte Pfad_im_Export ist nur im Archivexport belegt.",
		numField("Dokument_ID", "Interne Kennung des Dokuments."),
		numField("Anlage_ID", "Wirtschaftsgut, zu dem das Dokument gehört."),
		alphaField("Inventarnummer", "Nummer desselben Wirtschaftsguts."),
		alphaField("Art", "Art des Dokuments; siehe Schlüsselverzeichnis, Kategorie „Anlagendokument“."),
		alphaField("Dateiname", "Name, unter dem die Datei abgelegt wurde."),
		alphaField("Dateityp", "MIME-Typ der Datei."),
		numField("Groesse_Bytes", "Größe der Datei in Bytes."),
		alphaField("SHA256", "Prüfsumme des Dateiinhalts."),
		dateField("Gueltig_bis", "Tag, bis zu dem das Dokument gilt; leer, wo keine Frist besteht."),
		alphaField("Notiz", "Freitext zum Dokument."),
		alphaField("Pfad_im_Export", "Pfad der Datei innerhalb dieses Exports; leer, wenn keine Dateien beiliegen."),
	)
	for i := range d.documents {
		doc := &d.documents[i]
		if err := t.AddRow(
			export.Uint(doc.ID), export.Uint(doc.AssetID), d.inventoryNumber(doc.AssetID),
			string(doc.Kind), doc.FileName, doc.MimeType,
			export.Int64(doc.Size), doc.SHA256,
			doc.ValidUntil, doc.Note,
			d.documentFilePaths[doc.ID],
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

func vatReturnsTable(d *exportData) (export.Table, error) {
	t := newTable(tableVatReturns, "voranmeldungen.csv",
		"Die Umsatzsteuer-Voranmeldungen des Geschäftsjahres, eine Zeile je Kennziffer mit einem Wert.",
		numField("Anmeldung_ID", "Interne Kennung der Anmeldung."),
		alphaField("Zeitraum", "Schlüssel des Zeitraums, etwa 2026-03 oder 2026-Q1."),
		alphaField("Zeitraumart", "month, quarter oder year."),
		dateField("Von", "Erster Tag des Zeitraums."),
		dateField("Bis", "Letzter Tag des Zeitraums."),
		numField("Berichtigung", "1, wenn es sich um eine berichtigte Anmeldung handelt (Kennziffer 10), sonst 0."),
		alphaField("Status", "Stand der Anmeldung."),
		dateField("Uebermittelt_am", "Tag der Übermittlung."),
		alphaField("Transferticket", "Ticket der Übermittlung."),
		alphaField("Programmversion", "Fassung des Programms, die das Blatt gerechnet hat."),
		alphaField("Kennziffer", "Kennziffer des Vordrucks USt 1 A."),
		alphaField("Kennziffer_Text", "Wortlaut der Zeile des Vordrucks."),
		numField("Bemessungsgrundlage", "Bemessungsgrundlage in Euro, in vollen Euro abgerundet."),
		numField("Steuer", "Steuerbetrag in Euro."),
	)
	for i := range d.vatReturns {
		v := &d.vatReturns[i]
		for _, line := range v.Figures {
			if line.Base == 0 && line.Tax == 0 {
				// Der Vordruck kennt über hundert Kennziffern; die leeren
				// gehören auf das Blatt, aber nicht in die Datenüberlassung —
				// dort blähten sie die Tabelle um das Zwanzigfache auf, ohne
				// eine Auskunft zu tragen.
				continue
			}
			if err := t.AddRow(
				export.Uint(v.ID), v.PeriodKey, string(v.PeriodType),
				v.PeriodFrom, v.PeriodTo, export.Bool(v.IsCorrection),
				string(v.Status), v.SubmittedAt, v.TransferTicket, v.ProgramVersion,
				line.Code, line.Label,
				export.Amount(line.Base), export.Amount(line.Tax),
			); err != nil {
				return t, err
			}
		}
	}
	return t, nil
}

func commitsTable(d *exportData) (export.Table, error) {
	t := newTable(tableCommits, "festschreibungen.csv",
		"Die Festschreibungen: bis zu welchem Tag der Zeitraum geschlossen ist und welcher Kettenkopf dabei festgehalten wurde.",
		numField("Festschreibung_ID", "Interne Kennung."),
		numField("Geschaeftsjahr", "Geschäftsjahr der Festschreibung."),
		alphaField("Zeitraumart", "month, quarter oder year."),
		alphaField("Zeitraum", "Bezeichnung des Zeitraums."),
		dateField("Stichtag", "Letzter Tag, der festgeschrieben ist."),
		alphaField("Kettenkopf", "Eigenhash der letzten Buchung zum Zeitpunkt der Festschreibung."),
		numField("Buchungen", "Zahl der zu diesem Zeitpunkt erfassten Buchungen."),
		alphaField("Zeitstempel_Status", "confirmed, wenn ein qualifizierter Zeitstempel vorliegt."),
		alphaField("Zeitstempel_Dienst", "Name des Zeitstempeldienstes."),
		alphaField("Zeitstempel_Zeit", "Zeitpunkt des Zeitstempels nach RFC 3339."),
		alphaField("Erstellt_am", "Zeitpunkt der Festschreibung nach RFC 3339."),
	)
	for i := range d.commits {
		c := &d.commits[i]
		genTime := ""
		if c.TSAGenTime != nil {
			genTime = c.TSAGenTime.UTC().Format(time.RFC3339)
		}
		if err := t.AddRow(
			export.Uint(c.ID), export.Int(c.FiscalYear), c.PeriodType, c.PeriodLabel,
			c.CutoffDate, c.ChainHead, export.Int(c.EntryCount),
			c.TimestampStatus, c.TSAName, genTime,
			c.CreatedAt.UTC().Format(time.RFC3339),
		); err != nil {
			return t, err
		}
	}
	return t, nil
}

func checkRunsTable(d *exportData) (export.Table, error) {
	t := newTable(tableCheckRuns, "prueflaeufe.csv",
		"Die Prüfläufe vor den Festschreibungen mit ihren Befunden (internes Kontrollsystem, GoBD Rz. 100 ff.).",
		numField("Lauf_ID", "Interne Kennung des Laufs."),
		numField("Geschaeftsjahr", "Geschäftsjahr des Laufs."),
		dateField("Stichtag", "Tag, bis zu dem geprüft wurde."),
		alphaField("Zeitraumart", "Anlass des Laufs: month, quarter, year oder leer."),
		numField("Geprueft_Buchungen", "Zahl der geprüften Buchungen."),
		numField("Geprueft_Belege", "Zahl der geprüften Belege."),
		numField("Geprueft_Bankumsaetze", "Zahl der geprüften Bankumsätze."),
		alphaField("Uebergehungsgrund", "Begründung, mit der ein blockierender Befund übergangen wurde."),
		alphaField("Erstellt_am", "Zeitpunkt des Laufs nach RFC 3339."),
		alphaField("Regel", "Schlüssel der verletzten Regel."),
		alphaField("Gewicht", "blocking oder warning; siehe Schlüsselverzeichnis."),
		alphaField("Objektart", "Art des betroffenen Objekts."),
		alphaField("Objekt_ID", "Kennung des betroffenen Objekts."),
		alphaField("Befund", "Wortlaut des Befunds."),
		alphaField("Fundstelle", "Rechtliche Fundstelle des Befunds."),
	)
	for i := range d.checkRuns {
		r := &d.checkRuns[i]
		head := []string{
			export.Uint(r.ID), export.Int(r.FiscalYear), r.CutoffDate, r.PeriodType,
			export.Int(r.CheckedEntries), export.Int(r.CheckedReceipts), export.Int(r.CheckedBankTx),
			r.OverrideReason, r.CreatedAt.UTC().Format(time.RFC3339),
		}
		if len(r.Findings) == 0 {
			if err := t.AddRow(append(append([]string{}, head...), "", "", "", "", "", "")...); err != nil {
				return t, err
			}
			continue
		}
		for _, f := range r.Findings {
			row := append(append([]string{}, head...),
				f.Rule, string(f.Severity), f.ObjectType, f.ObjectID, f.Message, f.Reference)
			if err := t.AddRow(row...); err != nil {
				return t, err
			}
		}
	}
	return t, nil
}
