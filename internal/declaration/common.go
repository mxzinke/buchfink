package declaration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Common sind die Angaben, die jede Erklärung braucht.
type Common struct {
	// Year ist der Veranlagungs- oder Erhebungszeitraum.
	Year int
	// Period ist das Wirtschaftsjahr, das im Jahr endet; bei der Umsatzsteuer
	// das Kalenderjahr (§ 16 Abs. 1 Satz 2 UStG).
	Period   Period
	Settings domain.CompanySettings
	Inputs   []Input
	Basis    Basis
	// Records sind die gespeicherten Stände dieser Erklärung, älteste zuerst.
	Records []RecordState
}

// RecordState ist ein gespeicherter Stand: ein Export oder eine Übermittlung.
type RecordState struct {
	Status      Status
	FiguresHash string
	Submission  *Submission
}

// Ledger sind die Konten des Geschäftsjahres mit ihren Bewegungen.
type Ledger map[string]AccountTotals

// AccountTotals sind die Summen eines Kontos im Geschäftsjahr.
type AccountTotals struct {
	Name string
	// OpeningDebit und OpeningCredit stammen aus dem Saldenvortrag.
	OpeningDebit  domain.Cents
	OpeningCredit domain.Cents
	// Debit und Credit sind die Bewegungen des Jahres ohne Saldenvortrag,
	// einschließlich der Abschlussbuchungen.
	Debit  domain.Cents
	Credit domain.Cents
	// ClosingDebit und ClosingCredit sind der Anteil der Abschlussbuchungen.
	ClosingDebit  domain.Cents
	ClosingCredit domain.Cents
	EntryIDs      []uint
}

// Net ist die Bewegung des Jahres als Sollsaldo.
func (t AccountTotals) Net() domain.Cents { return t.Debit - t.Credit }

// NetBeforeClosing ist die Bewegung ohne Abschlussbuchungen.
func (t AccountTotals) NetBeforeClosing() domain.Cents {
	return (t.Debit - t.ClosingDebit) - (t.Credit - t.ClosingCredit)
}

// EndBalance ist der Sollsaldo zum Schluss des Jahres.
func (t AccountTotals) EndBalance() domain.Cents {
	return t.OpeningDebit - t.OpeningCredit + t.Debit - t.Credit
}

// AddLine trägt eine Buchungszeile in die Summen ein.
func (l Ledger) AddLine(entry *domain.JournalEntry, line domain.JournalLine, name string) {
	t := l[line.Account]
	if t.Name == "" {
		t.Name = name
	}
	opening := entry.Source == domain.EntrySourceOpening
	closing := entry.Source == domain.EntrySourceClosing
	switch {
	case opening && line.Side == domain.SideDebit:
		t.OpeningDebit += line.Amount
	case opening:
		t.OpeningCredit += line.Amount
	case line.Side == domain.SideDebit:
		t.Debit += line.Amount
		if closing {
			t.ClosingDebit += line.Amount
		}
	default:
		t.Credit += line.Amount
		if closing {
			t.ClosingCredit += line.Amount
		}
	}
	if !opening && (len(t.EntryIDs) == 0 || t.EntryIDs[len(t.EntryIDs)-1] != entry.ID) {
		t.EntryIDs = append(t.EntryIDs, entry.ID)
	}
	l[line.Account] = t
}

// share ist ein Konto mit dem Bruchteil, mit dem es in eine Zeile eingeht.
type share struct {
	account  string
	num, den int64
}

func full(accounts ...string) []share {
	out := make([]share, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, share{account: a, num: 1, den: 1})
	}
	return out
}

func fraction(num, den int64, accounts ...string) []share {
	out := make([]share, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, share{account: a, num: num, den: den})
	}
	return out
}

// sum rechnet die Beiträge der Konten. expense: Aufwand zählt positiv (Soll),
// sonst Ertrag positiv (Haben). Konten ohne Bewegung stehen nicht in der
// Herkunft.
func (l Ledger) sum(shares []share, expense bool, net func(AccountTotals) domain.Cents) (domain.Cents, Origin) {
	if net == nil {
		net = AccountTotals.Net
	}
	origin := Origin{Kind: OriginAccounts}
	var total domain.Cents
	ids := map[uint]bool{}
	for _, s := range shares {
		t, ok := l[s.account]
		if !ok {
			continue
		}
		balance := net(t)
		if !expense {
			balance = -balance
		}
		if balance == 0 {
			continue
		}
		amount := balance
		if s.num != s.den {
			amount = domain.MulRound(balance, s.num, s.den)
		}
		total += amount
		origin.Accounts = append(origin.Accounts, AccountPart{
			Account: s.account, Name: t.Name, Balance: balance, Num: s.num, Den: s.den, Amount: amount,
		})
		for _, id := range t.EntryIDs {
			ids[id] = true
		}
	}
	for id := range ids {
		origin.EntryIDs = append(origin.EntryIDs, id)
	}
	sort.Slice(origin.EntryIDs, func(i, j int) bool { return origin.EntryIDs[i] < origin.EntryIDs[j] })
	return total, origin
}

// expense ist die Summe von Aufwandskonten (Sollsaldo positiv).
func (l Ledger) expense(shares ...share) (domain.Cents, Origin) { return l.sum(shares, true, nil) }

// income ist die Summe von Ertragskonten (Habensaldo positiv).
func (l Ledger) income(shares ...share) (domain.Cents, Origin) { return l.sum(shares, false, nil) }

// hasBalance meldet, ob eines der Konten im Jahr bewegt wurde.
func (l Ledger) hasBalance(accounts ...string) bool {
	for _, a := range accounts {
		if t, ok := l[a]; ok && t.Net() != 0 {
			return true
		}
	}
	return false
}

// subjectOf liest den Steuerpflichtigen aus den Stammdaten.
func subjectOf(s domain.CompanySettings) Subject {
	country := s.ResolvedCountryCode()
	return Subject{
		Name:            strings.TrimSpace(s.CompanyName),
		LegalForm:       s.LegalForm,
		TaxNumber:       strings.TrimSpace(s.TaxNumber),
		TaxNumberElster: s.TaxNumberElster,
		TaxOfficeNumber: domain.TaxOfficeNumber(s.TaxNumberElster),
		TaxOffice:       s.TaxOffice,
		FederalState:    s.FederalState,
		VatID:           s.VatID,
		Address: Address{
			Street: s.Street, HouseNumber: s.HouseNumber, Addition: s.AddressAddition,
			PostalCode: s.PostalCode, City: s.City, Country: country,
		},
	}
}

// taxNumberLine setzt die Steuernummer: die des Landes, die ELSTER-Form als
// Hinweis. Ohne Steuernummer bleibt die Zeile offen.
func (c *calc) taxNumberLine(key string, s Subject) {
	if s.TaxNumber == "" {
		c.open(key, "Die Steuernummer fehlt in den Unternehmensdaten.")
		return
	}
	notes := []string{}
	if s.TaxNumberElster != "" {
		notes = append(notes, fmt.Sprintf("Im 13-stelligen ELSTER-Format: %s (Finanzamtsnummer %s).",
			s.TaxNumberElster, s.TaxOfficeNumber))
	} else {
		notes = append(notes, "Ohne Bundesland in den Unternehmensdaten lässt sich die 13-stellige ELSTER-Steuernummer nicht ableiten.")
	}
	c.books(key, Text(s.TaxNumber), Origin{Kind: OriginSettings, Text: "Unternehmensdaten"}, notes...)
}

// textLine setzt eine Angabe aus den Stammdaten; leer bleibt sie offen.
func (c *calc) textLine(key, value, missing string) {
	value = strings.TrimSpace(value)
	if value == "" {
		c.open(key, missing)
		return
	}
	c.books(key, Text(value), Origin{Kind: OriginSettings, Text: "Unternehmensdaten"})
}

// legalFormText ist die Rechtsform, wie Mein ELSTER sie zur Auswahl anbietet.
func legalFormText(form string) string {
	switch form {
	case domain.LegalFormGmbH:
		return "Gesellschaft mit beschränkter Haftung"
	case domain.LegalFormUG:
		return "Unternehmergesellschaft (haftungsbeschränkt)"
	}
	return form
}

// finish setzt die Erklärung zusammen und bestimmt Zahlen-Hash und Stand.
func finish(kind Kind, common Common, cat *Catalog, exact bool, c *calc) *Declaration {
	if !exact {
		c.finding(SeverityWarning, "vordruck_nicht_geprueft", "",
			fmt.Sprintf("Für %d liegt kein geprüfter Zeilenkatalog vor; Zeilen und Kennzahlen stammen aus dem Vordruck %d. "+
				"Im Formular des Jahres in Mein ELSTER nach der Bezeichnung suchen.", common.Year, cat.FormYear), "")
	}
	if err := common.Settings.Unsupported(); err != nil {
		c.finding(SeverityBlocking, "nicht_unterstuetzt", "", err.Error(), "")
	}
	forms := c.assemble()
	if !exact {
		for i := range forms {
			for j := range forms[i].Sections {
				for k := range forms[i].Sections[j].Lines {
					forms[i].Sections[j].Lines[k].Check.OnForm = false
				}
			}
		}
	}
	basis := common.Basis
	basis.CatalogVersion = cat.Version
	d := &Declaration{
		Schema: Schema, Kind: kind, Title: fmt.Sprintf("%s %d", kind.Label(), common.Year),
		Period: common.Period, FormYear: cat.FormYear, Subject: subjectOf(common.Settings),
		Forms: forms, Inputs: append([]Input{}, common.Inputs...), Findings: c.findings,
		Status: StatusDraft, Basis: basis,
	}
	d.FiguresHash = FiguresHash(d)
	d.applyRecords(common.Records)
	sortFindings(d.Findings)
	return d
}

// applyRecords leitet den Stand aus den gespeicherten Exporten und
// Übermittlungen ab.
func (d *Declaration) applyRecords(records []RecordState) {
	var lastSubmission *Submission
	exported := false
	for _, r := range records {
		if r.Status == StatusSubmitted && r.Submission != nil {
			s := *r.Submission
			lastSubmission = &s
		}
		if r.Status == StatusExported && r.FiguresHash == d.FiguresHash {
			exported = true
		}
	}
	d.Submission = lastSubmission
	switch {
	case lastSubmission != nil && lastSubmission.FiguresHash == d.FiguresHash:
		d.Status = StatusSubmitted
	case exported:
		d.Status = StatusExported
	}
	if lastSubmission != nil && lastSubmission.FiguresHash != d.FiguresHash {
		d.Findings = append(d.Findings, Finding{
			Severity: SeverityWarning, Code: "seit_uebermittlung_geaendert",
			Text: fmt.Sprintf("Die Zahlen weichen von der am %s übermittelten Erklärung (Transferticket %s) ab. "+
				"Ist die übermittelte Erklärung unrichtig, ist sie zu berichtigen.", lastSubmission.At, lastSubmission.TransferTicket),
			Legal: "§ 153 Abs. 1 AO",
		})
	}
}
