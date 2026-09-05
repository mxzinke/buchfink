package ebilanz

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die XBRL-Instanz der E-Bilanz (§ 5b EStG).
//
// Sie entsteht aus der Gliederung nach §§ 266, 275 HGB und aus der
// eingebetteten Taxonomie-Ressource — nicht mehr aus einer Kontentabelle im
// Code. Der Unterschied ist kein technischer: die alte Tabelle kannte rund
// neunzig Konten, alle übrigen landeten still auf „bs.other". Eine Instanz, die
// so entsteht, ist formal vollständig und inhaltlich unbrauchbar, und niemand
// hätte es gemerkt. Jetzt entscheidet dieselbe Zuordnung wie in der Bilanz, und
// was sie nicht einordnen kann, verhindert die Erzeugung.
//
// Geschrieben wird mit encoding/xml statt mit fmt.Sprintf: die Escaping-Regeln
// für Attribute und Textknoten sind damit die des XML-Kodierers und nicht die
// eines Formatstrings, der sie zufällig auch kennt.

// node ist ein Element der Instanz. Der Namensraum steht als Präfix im lokalen
// Namen — so, wie die Taxonomie ihn schreibt und wie ihn die Prüfprogramme des
// Fiskus erwarten; die Deklaration steht einmal am Wurzelelement.
type node struct {
	Name     string
	Attrs    []xml.Attr
	Value    string
	Children []node
}

// MarshalXML schreibt das Element mit seinem Präfix.
func (n node) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	start := xml.StartElement{Name: xml.Name{Local: n.Name}, Attr: n.Attrs}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if n.Value != "" {
		if err := e.EncodeToken(xml.CharData(n.Value)); err != nil {
			return err
		}
	}
	for _, child := range n.Children {
		if err := e.Encode(child); err != nil {
			return err
		}
	}
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

func attr(name, value string) xml.Attr {
	return xml.Attr{Name: xml.Name{Local: name}, Value: value}
}

// fact ist eine Zahl mit Kontext, Einheit und Genauigkeit.
func fact(element, context string, amount domain.Cents) node {
	return node{
		Name:  element,
		Attrs: []xml.Attr{attr("contextRef", context), attr("unitRef", "EUR"), attr("decimals", "2")},
		Value: amount.Decimal(),
	}
}

func text(element, context, value string) node {
	return node{
		Name:  element,
		Attrs: []xml.Attr{attr("contextRef", context)},
		Value: value,
	}
}

// InstanceInput ist alles, was in die Instanz eingeht.
type InstanceInput struct {
	Settings  *domain.CompanySettings
	Statement *domain.Statement
	// Accounts sind die Konten des Geschäftsjahres mit ihren Salden; der
	// Kontennachweis führt sie unverdichtet auf.
	Accounts       []domain.Account
	Anlagenspiegel *domain.Anlagenspiegel

	FiscalYear     int
	StartDate      string
	EndDate        string
	PriorStartDate string
	PriorEndDate   string
}

const (
	contextInstant       = "ctx_instant"
	contextInstantPrior  = "ctx_instant_prior"
	contextDuration      = "ctx_duration"
	contextDurationPrior = "ctx_duration_prior"
)

// GenerateEBilanzXBRL erzeugt die Instanz.
//
// Sie lehnt ab, wenn der Zuordnungsbericht blockierende Befunde hat. Der
// Bericht ist Rückgabewert und nicht nur Nebenwirkung: die Oberfläche zeigt ihn
// vor dem Export, und der Fehlerfall soll dieselbe Liste liefern wie der
// Erfolgsfall.
func GenerateEBilanzXBRL(in InstanceInput) (string, *MappingReport, error) {
	if in.Settings == nil {
		return "", nil, fmt.Errorf("ohne die Unternehmensdaten lässt sich keine E-Bilanz erzeugen")
	}
	if in.Statement == nil {
		return "", nil, fmt.Errorf("ohne Bilanz und Gewinn- und Verlustrechnung lässt sich keine E-Bilanz erzeugen")
	}
	tax, err := LoadTaxonomy()
	if err != nil {
		return "", nil, err
	}

	report, err := BuildMappingReport(in.FiscalYear, in.Statement, in.Accounts)
	if err != nil {
		return "", nil, err
	}
	if err := report.BlockingError(); err != nil {
		return "", report, err
	}

	root := node{Name: "xbrli:xbrl"}
	for _, prefix := range sortedPrefixes(tax.Namespaces) {
		root.Attrs = append(root.Attrs, attr("xmlns:"+prefix, tax.Namespaces[prefix]))
	}

	identifier := in.Settings.TaxNumber
	root.Children = append(root.Children,
		instantContext(contextInstant, identifier, in.EndDate),
		durationContext(contextDuration, identifier, in.StartDate, in.EndDate),
	)
	if in.Statement.HasPrior && in.PriorEndDate != "" {
		root.Children = append(root.Children,
			instantContext(contextInstantPrior, identifier, in.PriorEndDate),
			durationContext(contextDurationPrior, identifier, in.PriorStartDate, in.PriorEndDate),
		)
	}
	root.Children = append(root.Children, node{
		Name:     "xbrli:unit",
		Attrs:    []xml.Attr{attr("id", "EUR")},
		Children: []node{{Name: "xbrli:measure", Value: "iso4217:EUR"}},
	})

	root.Children = append(root.Children, companyData(in)...)
	root.Children = append(root.Children, statementFacts(in.Statement)...)
	root.Children = append(root.Children, proofOfAccounts(report)...)
	root.Children = append(root.Children, fixedAssetMovements(in.Anlagenspiegel, report)...)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(instanceComment(tax))
	buf.WriteString("\n")

	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "\t")
	if err := encoder.Encode(root); err != nil {
		return "", report, fmt.Errorf("die XBRL-Instanz konnte nicht geschrieben werden: %w", err)
	}
	if err := encoder.Flush(); err != nil {
		return "", report, err
	}
	buf.WriteString("\n")
	return buf.String(), report, nil
}

// instanceComment schreibt den Vorbehalt in die Datei selbst. Wer sie in fünf
// Jahren öffnet, hat die Aktennotiz nicht mehr, in der er stand.
func instanceComment(tax *Taxonomy) string {
	return fmt.Sprintf(
		"<!--\n\tErzeugt von Buchfink aus der Gliederung nach §§ 266, 275 HGB.\n"+
			"\tTaxonomie: HGB %s vom %s.\n\t%s\n-->",
		tax.Version, tax.Date, tax.Note)
}

func sortedPrefixes(namespaces map[string]string) []string {
	out := make([]string, 0, len(namespaces))
	for prefix := range namespaces {
		out = append(out, prefix)
	}
	sort.Strings(out)
	return out
}

func entity(identifier string) node {
	return node{
		Name: "xbrli:entity",
		Children: []node{{
			Name:  "xbrli:identifier",
			Attrs: []xml.Attr{attr("scheme", "http://www.steuerliche-identifikationsnummer.de")},
			Value: identifier,
		}},
	}
}

func instantContext(id, identifier, date string) node {
	return node{
		Name:  "xbrli:context",
		Attrs: []xml.Attr{attr("id", id)},
		Children: []node{
			entity(identifier),
			{Name: "xbrli:period", Children: []node{{Name: "xbrli:instant", Value: date}}},
		},
	}
}

func durationContext(id, identifier, start, end string) node {
	return node{
		Name:  "xbrli:context",
		Attrs: []xml.Attr{attr("id", id)},
		Children: []node{
			entity(identifier),
			{Name: "xbrli:period", Children: []node{
				{Name: "xbrli:startDate", Value: start},
				{Name: "xbrli:endDate", Value: end},
			}},
		},
	}
}

// companyData ist das GCD-Modul: die Stammdaten des Unternehmens einschließlich
// der Pflichtangaben des § 264 Abs. 1a HGB.
func companyData(in InstanceInput) []node {
	s := in.Settings
	nodes := []node{
		text("de-gcd:genInfo.company.id.name", contextDuration, s.CompanyName),
		text("de-gcd:genInfo.company.id.legalForm", contextDuration, s.LegalForm),
		text("de-gcd:genInfo.company.id.taxNumber", contextDuration, s.TaxNumber),
		text("de-gcd:genInfo.company.id.vatId", contextDuration, s.VatID),
	}
	if s.Seat != "" {
		nodes = append(nodes, text("de-gcd:genInfo.company.id.location.city", contextDuration, s.Seat))
	}
	if s.RegisterCourt != "" {
		nodes = append(nodes, text("de-gcd:genInfo.company.id.register.court", contextDuration, s.RegisterCourt))
	}
	if s.RegisterNumber != "" {
		nodes = append(nodes, text("de-gcd:genInfo.company.id.register.number", contextDuration, s.RegisterNumber))
	}
	return append(nodes,
		text("de-gcd:genInfo.report.period.fiscalYearBegin", contextDuration, in.StartDate),
		text("de-gcd:genInfo.report.period.fiscalYearEnd", contextDuration, in.EndDate),
		text("de-gcd:genInfo.report.accountingStandard", contextDuration, "HGB / Steuerrecht"),
		text("de-gcd:genInfo.report.accountScheme", contextDuration, "SKR04"),
	)
}

// statementFacts schreibt Bilanz und Gewinn- und Verlustrechnung, jede Position
// mit ihrem Wert und — wenn es ein Vorjahr gibt — mit dem zweiten Kontext.
func statementFacts(stmt *domain.Statement) []node {
	var nodes []node
	emitValue := func(key, current, prior string, amount, priorAmount domain.Cents) {
		element, ok := ElementFor(key)
		if !ok {
			return
		}
		nodes = append(nodes, fact(element.Element, current, amount))
		if stmt.HasPrior && prior != "" {
			nodes = append(nodes, fact(element.Element, prior, priorAmount))
		}
	}
	emit := func(lines []domain.StatementLine, current, prior string) {
		for _, line := range lines {
			amount, priorAmount := line.Amount, line.PriorAmount
			// Die Gliederung führt Aufwendungen als negativen Beitrag zum
			// Ergebnis, die Taxonomie erwartet sie als positiven Betrag: ein
			// Materialaufwand von 10.000 Euro steht in der Instanz als 10000.00,
			// nicht als -10000.00. Zwischensummen und Erträge bleiben, wie sie
			// sind.
			if accounting.IsExpensePosition(line.Key) {
				amount, priorAmount = -amount, -priorAmount
			}
			emitValue(line.Key, current, prior, amount, priorAmount)
		}
	}
	emit(stmt.Assets, contextInstant, contextInstantPrior)
	// Die Bilanzsumme ist kein Posten des § 266 HGB und steht deshalb in keiner
	// Zeile der Gliederung — in der Instanz muss sie trotzdem stehen: sie ist
	// die Zahl, gegen die das Prüfprogramm des Fiskus die Seiten abgleicht, und
	// ohne sie bliebe der Abgleich dem Leser überlassen.
	emitValue("aktiva", contextInstant, contextInstantPrior, stmt.TotalAssets, stmt.TotalAssetsPrior)
	emit(stmt.Liabilities, contextInstant, contextInstantPrior)
	emitValue("passiva", contextInstant, contextInstantPrior, stmt.TotalLiabilities, stmt.TotalLiabilitiesPrior)
	emit(stmt.Income, contextDuration, contextDurationPrior)
	return nodes
}

// proofOfAccounts ist der Kontennachweis: jedes Konto mit Saldo, unverdichtet,
// jetzt mit seiner Gliederungsposition und dem Taxonomie-Element.
func proofOfAccounts(report *MappingReport) []node {
	var nodes []node
	for _, row := range report.Rows {
		nodes = append(nodes, node{
			Name:  "de-gaap-ci:accountAuditProof",
			Attrs: []xml.Attr{attr("contextRef", contextDuration)},
			Children: []node{
				{Name: "de-gaap-ci:accountNumber", Value: row.Account},
				{Name: "de-gaap-ci:accountLabel", Value: row.Name},
				{Name: "de-gaap-ci:accountPosition", Value: row.PositionLabel},
				{Name: "de-gaap-ci:accountTaxonomyPosition", Value: row.Element},
				fact("de-gaap-ci:accountBalance", contextDuration, row.Balance),
			},
		})
	}
	return nodes
}

// fixedAssetMovements ist die Entwicklung des Anlagevermögens (§ 284 Abs. 3
// HGB).
//
// Der Anlagenspiegel ist keine zweite Buchung, sondern die Auswertung der
// Kartei — aber eine, die das Journal allein nicht liefern könnte: Zugänge,
// Abgänge und kumulierte Abschreibungen eines vor Jahren angeschafften
// Wirtschaftsguts stehen nur dort. Genau deshalb gehört sie in den
// Kontennachweis: die Bilanz zeigt einen Buchwert, und erst der Spiegel zeigt,
// woraus er entstanden ist.
func fixedAssetMovements(spiegel *domain.Anlagenspiegel, report *MappingReport) []node {
	if spiegel == nil || len(spiegel.Rows) == 0 {
		return nil
	}
	elements := make(map[string]string, len(report.Rows))
	for _, row := range report.Rows {
		elements[row.Account] = row.Element
	}

	build := func(row domain.AnlagenspiegelRow, element, key, label string) node {
		position := elements[row.Account]
		if position == "" {
			// Die Klassensummen tragen kein Konto; sie stehen unter der
			// Sammelposition des Anlagevermögens.
			if e, ok := ElementFor("aktiva.A"); ok {
				position = e.Element
			}
		}
		return node{
			Name:  "de-gaap-ci:" + element,
			Attrs: []xml.Attr{attr("contextRef", contextDuration)},
			Children: []node{
				{Name: "de-gaap-ci:position", Value: key},
				{Name: "de-gaap-ci:positionLabel", Value: label},
				{Name: "de-gaap-ci:taxonomyPosition", Value: position},
				fact("de-gaap-ci:histCost.begin", contextDuration, row.CostOpening),
				fact("de-gaap-ci:histCost.addition", contextDuration, row.Additions),
				fact("de-gaap-ci:histCost.disposal", contextDuration, row.Disposals),
				fact("de-gaap-ci:histCost.transfer", contextDuration, row.Transfers),
				fact("de-gaap-ci:histCost.end", contextDuration, row.CostClosing),
				fact("de-gaap-ci:deprec.begin", contextDuration, row.DepreciationOpening),
				fact("de-gaap-ci:deprec.currentYear", contextDuration, row.DepreciationYear),
				fact("de-gaap-ci:deprec.writeUp", contextDuration, row.WriteUpsYear),
				fact("de-gaap-ci:deprec.disposal", contextDuration, row.DepreciationDisposal),
				fact("de-gaap-ci:deprec.transfer", contextDuration, row.DepreciationTransfer),
				fact("de-gaap-ci:deprec.end", contextDuration, row.DepreciationClosing),
				fact("de-gaap-ci:netBookValue.begin", contextDuration, row.BookValueOpening),
				fact("de-gaap-ci:netBookValue.end", contextDuration, row.BookValueClosing),
			},
		}
	}

	var nodes []node
	for _, row := range spiegel.Rows {
		nodes = append(nodes, build(row, "fixedAssetsMovement", row.Account, row.AccountName))
	}
	// Die drei Blöcke des § 266 Abs. 2 A HGB und die Gesamtsumme. Sie stehen
	// hier, weil die Bilanz sie so ausweist — nachrechnen soll sie niemand
	// müssen, der den Nachweis liest.
	for _, total := range spiegel.ClassTotals {
		nodes = append(nodes, build(total, "fixedAssetsMovementSubtotal", string(total.Class), total.AccountName))
	}
	nodes = append(nodes, build(spiegel.Totals, "fixedAssetsMovementTotal", "total", spiegel.Totals.AccountName))
	return nodes
}

// StatementCoverage meldet die Gliederungspositionen, für die die Taxonomie
// kein Element kennt. Der Test wacht damit darüber, dass die Ressource zur
// Gliederung passt; ohne sie fiele eine neue Zeile erst beim Export auf.
func StatementCoverage() []string {
	var missing []string
	for _, line := range accounting.StatementLines() {
		switch line.Key {
		case "aktiva.X", "passiva.X", "guv.X", "statistisch":
			// Diese vier sind keine Posten des Gesetzes, sondern Befunde. Sie
			// haben in der Taxonomie nichts zu suchen; ein Konto, das dort
			// landet, blockiert den Export ohnehin.
			continue
		}
		if _, ok := ElementFor(line.Key); !ok {
			missing = append(missing, line.Key)
		}
	}
	return missing
}
