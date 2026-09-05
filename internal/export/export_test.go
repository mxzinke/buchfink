package export

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"regexp"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func sampleDataset() *Dataset {
	journal := Table{
		Name: "journal", FileName: "journal.csv",
		Description: "Alle Buchungen",
		Fields: []Field{
			{Name: "Buchungsnummer", Type: FieldAlphaNumeric, Length: 20, Description: "Nummer der Buchung"},
			{Name: "Buchungsdatum", Type: FieldDate, Description: "Tag der Buchung"},
			{Name: "Betrag", Type: FieldNumeric, Description: "Betrag in Euro"},
			{Name: "Buchungstext", Type: FieldAlphaNumeric, Description: "Beschreibung"},
		},
		Rows: [][]string{
			{"2026-000001", "2026-03-01", "119.00", `Miete "März"; Halle 2`},
			{"2026-000002", "2026-03-02", "-42.50", "Zeile eins\nZeile zwei"},
			{"2026-000003", "2026-03-03", "0.00", " führendes Leerzeichen"},
		},
	}
	keys := Table{
		Name: KeyDirectoryTable, FileName: "schluesselverzeichnis.csv",
		Description: "Codes im Klartext",
		Fields: []Field{
			{Name: "Kategorie", Type: FieldAlphaNumeric, Description: "Bereich"},
			{Name: "Schluessel", Type: FieldAlphaNumeric, Description: "Code"},
			{Name: "Bedeutung", Type: FieldAlphaNumeric, Description: "Klartext"},
			{Name: "Erlaeuterung", Type: FieldAlphaNumeric, Description: "Erklärung"},
		},
		Rows: [][]string{{"Buchungsseite", "S", "Soll", "Pipe | im Text"}},
	}
	return &Dataset{
		TenantName:       `Pfennig & Söhne "GmbH"`,
		SupplierLocation: "Hauptstraße 1, 80331 München",
		FiscalYear:       2026,
		From:             "2026-01-01", To: "2026-12-31",
		CreatedAt: "2026-09-05T10:00:00Z", ProgramVersion: "0.3.0-dev",
		Tables: []Table{journal, keys},
	}
}

// --- CSV -------------------------------------------------------------------

// Eine CSV, die sich nicht wieder einlesen lässt, ist keine Datenüberlassung.
// Geprüft wird deshalb nicht die Schreibweise, sondern der Rundlauf: was
// hineingegeben wurde, muss Feld für Feld wieder herauskommen.
func TestRenderCSVRoundTrips(t *testing.T) {
	table := sampleDataset().Tables[0]
	got := parseCSV(t, string(RenderCSV(table)))

	if len(got) != len(table.Rows)+1 {
		t.Fatalf("erwartet %d Zeilen (Kopf plus Daten), gelesen %d", len(table.Rows)+1, len(got))
	}
	for i, name := range table.FieldNames() {
		if got[0][i] != name {
			t.Errorf("Kopfzeile Spalte %d: %q statt %q", i, got[0][i], name)
		}
	}
	for r, want := range table.Rows {
		for c, value := range want {
			if got[r+1][c] != value {
				t.Errorf("Zeile %d Spalte %d: %q statt %q", r+1, c, got[r+1][c], value)
			}
		}
	}
}

// Ein Wert mit Semikolon, Anführungszeichen oder Zeilenumbruch muss in
// Anführungszeichen stehen; ein harmloser Wert nicht. Ohne die zweite Hälfte
// wäre die Regel „alles quoten" und der Test bestünde ohne Aussage.
func TestRenderCSVQuotesOnlyWhereNeeded(t *testing.T) {
	table := Table{
		Fields: []Field{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}},
		Rows:   [][]string{{"harmlos", "mit;Semikolon", `mit"Anfuehrung`, " Rand "}},
	}
	line := strings.Split(string(RenderCSV(table)), RecordDelimiter)[1]

	if !strings.HasPrefix(line, "harmlos;") {
		t.Errorf("ein Wert ohne Sonderzeichen darf nicht in Anführungszeichen stehen: %q", line)
	}
	if !strings.Contains(line, `"mit;Semikolon"`) {
		t.Errorf("ein Wert mit Semikolon muss eingefasst sein: %q", line)
	}
	if !strings.Contains(line, `"mit""Anfuehrung"`) {
		t.Errorf("ein enthaltenes Anführungszeichen muss verdoppelt werden: %q", line)
	}
	if !strings.Contains(line, `" Rand "`) {
		t.Errorf("Randleerzeichen müssen eingefasst werden, sonst gehen sie verloren: %q", line)
	}
}

// UTF-8 ohne BOM: eine Prüfsoftware, die die BOM nicht erwartet, fände den
// ersten Spaltennamen nicht wieder.
func TestRenderCSVHasNoByteOrderMark(t *testing.T) {
	out := RenderCSV(sampleDataset().Tables[0])
	if strings.HasPrefix(string(out), "\ufeff") {
		t.Error("die CSV trägt eine BOM")
	}
	if !strings.HasPrefix(string(out), "Buchungsnummer;") {
		t.Errorf("die Datei beginnt nicht mit der Kopfzeile: %q", string(out[:20]))
	}
}

// --- index.xml -------------------------------------------------------------

func TestIndexXMLIsWellFormed(t *testing.T) {
	data := RenderIndexXML(sampleDataset())
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = true
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("index.xml ist nicht wohlgeformt: %v\n%s", err, data)
		}
	}
	if !strings.Contains(string(data), `<!DOCTYPE DataSet SYSTEM "`+DTDFileName+`">`) {
		t.Error("index.xml nennt die mitgelieferte Grammatik nicht im DOCTYPE")
	}
}

// Der Firmenname kommt aus den Stammdaten und darf ein „&" enthalten. Ohne
// Maskierung wäre die Beschreibungsdatei damit unlesbar — und mit ihr die ganze
// Überlassung.
func TestIndexXMLEscapesTenantName(t *testing.T) {
	data := string(RenderIndexXML(sampleDataset()))
	if strings.Contains(data, `Pfennig & Söhne`) {
		t.Error("das kaufmännische Und wurde nicht maskiert")
	}
	if !strings.Contains(data, "Pfennig &amp; Söhne &quot;GmbH&quot;") {
		t.Errorf("der Mandantenname steht nicht maskiert in index.xml:\n%s", data)
	}
}

// Location ist der Sitz des Datenlieferanten. Stünde dort der Programmname,
// ließe sich der Datenträger nicht mehr einem Betrieb zuordnen — und die
// Programmfassung gehört ohnehin in den Kommentar.
func TestIndexXMLNamesTheSupplierLocation(t *testing.T) {
	d := sampleDataset()
	root := parseTree(t, string(RenderIndexXML(d)))
	supplier := findAll(root, "DataSupplier")
	if len(supplier) != 1 {
		t.Fatalf("index.xml nennt %d Datenlieferanten", len(supplier))
	}
	if got := text(supplier[0], "Location"); got != d.SupplierLocation {
		t.Errorf("Location = %q, erwartet den Sitz %q", got, d.SupplierLocation)
	}
	if !strings.Contains(text(supplier[0], "Comment"), d.ProgramVersion) {
		t.Errorf("die Programmfassung steht nicht im Comment: %q", text(supplier[0], "Comment"))
	}

	// Ohne hinterlegten Sitz bleibt das Element nicht leer, sondern sagt es.
	d.SupplierLocation = ""
	empty := parseTree(t, string(RenderIndexXML(d)))
	if got := text(findAll(empty, "DataSupplier")[0], "Location"); got == "" {
		t.Error("ohne hinterlegten Sitz bleibt Location leer statt einen Hinweis zu tragen")
	}
}

// Die mitgelieferte Datei muss der amtliche Text sein und nicht eine
// Nachbildung unter amtlichem Namen.
//
// Die Prüfsumme steht fest, weil die Grammatik feststeht: sie ist vorgegeben
// und wird nicht gepflegt. Fiele einem später ein, sie zu erweitern, damit eine
// index.xml durchgeht, schlüge dieser Test an — zu ändern wäre dann die
// index.xml. Geprüft wird zusätzlich am Inhalt und nicht nur an der Summe,
// damit die Meldung sagt, was fehlt.
func TestShippedDTDIsTheOfficialText(t *testing.T) {
	sum := sha256.Sum256(DTD())
	if got := hex.EncodeToString(sum[:]); got != DTDSHA256 {
		t.Errorf("die mitgelieferte %s hat die Prüfsumme %s statt %s — sie ist nicht mehr der amtliche Text",
			DTDFileName, got, DTDSHA256)
	}
	// Die drei Inhaltsmodelle, an denen sich die amtliche Fassung von einer
	// selbst geschriebenen Teilmenge unterscheidet: die Stellung der
	// Kodierungsangabe in Table, MaxLength als Geschwister von AlphaNumeric und
	// AlphaNumeric als leeres Element.
	for _, want := range []string{
		"<!ELEMENT Table (URL, Name?, Description?, Validity?, (ANSI | Macintosh | OEM | UTF16 | UTF7 | UTF8)?, (DecimalSymbol, DigitGroupingSymbol)?, SkipNumBytes?, Range?, Epoch?, (VariableLength | FixedLength))>",
		"<!ELEMENT VariableColumn (Name, Description?, (Numeric | (AlphaNumeric, MaxLength?) | Date), Map*)>",
		"<!ELEMENT AlphaNumeric EMPTY>",
		"<!ELEMENT DataSupplier (Name, Location, Comment)>",
	} {
		if !strings.Contains(string(DTD()), want) {
			t.Errorf("die mitgelieferte %s enthält die amtliche Deklaration nicht:\n%s", DTDFileName, want)
		}
	}
}

// Die Beschreibungsdatei muss gegen die amtliche Grammatik gehen: jedes Element
// ist dort deklariert, und die Reihenfolge seiner Kinder entspricht dem
// Inhaltsmodell. Eine index.xml, die die DTD verletzt, weist eine Prüfsoftware
// zurück — und mit ihr die ganze Überlassung.
func TestIndexXMLValidatesAgainstOfficialDTD(t *testing.T) {
	models := parseDTD(t, string(DTD()))
	root := parseTree(t, string(RenderIndexXML(sampleDataset())))
	validateAgainstDTD(t, models, root)

	// Und die Feldbeschreibung sagt, dass die beiliegende Grammatik die
	// amtliche ist. Ein Prüfer soll der Überlassung entnehmen können, wogegen
	// er validiert.
	if doc := RenderFieldDoc(sampleDataset()); !strings.Contains(string(doc), "amtliche Text des Beschreibungsstandards") {
		t.Error("die Feldbeschreibung weist die beiliegende Grammatik nicht als den amtlichen Text aus")
	}
}

// Die Stellen, an denen die frühere Fassung gegen die amtliche Grammatik verstieß.
// Sie stehen einzeln, weil der Ausdruck über die Kindernamen im Fehlerfall nur
// sagt, dass etwas nicht passt, und nicht, was.
func TestIndexXMLFollowsTheOfficialContentModel(t *testing.T) {
	root := parseTree(t, string(RenderIndexXML(sampleDataset())))
	table := findAll(root, "Table")[0]

	names := make([]string, 0, len(table.children))
	for _, c := range table.children {
		names = append(names, c.name)
	}
	utf8At, decimalAt := indexOf(names, "UTF8"), indexOf(names, "DecimalSymbol")
	if utf8At < 0 || decimalAt < 0 {
		t.Fatalf("Table führt %v — Kodierungsangabe oder Dezimaltrennzeichen fehlen", names)
	}
	if utf8At > decimalAt {
		t.Errorf("Table führt %v — die Kodierungsangabe steht hinter den Trennzeichen, "+
			"das Inhaltsmodell erwartet sie davor", names)
	}

	// MaxLength ist Geschwister von AlphaNumeric und nicht dessen Kind.
	column := findAll(root, "VariableColumn")[0]
	alpha := findAll(column, "AlphaNumeric")
	if len(alpha) != 1 {
		t.Fatalf("die erste Spalte beschreibt %d alphanumerische Typen", len(alpha))
	}
	if len(alpha[0].children) != 0 {
		t.Errorf("AlphaNumeric hat Kinder — im Standard ist es ein leeres Element")
	}
	if len(findAll(column, "MaxLength")) != 1 {
		t.Error("die Längenangabe MaxLength fehlt neben AlphaNumeric")
	}
}

func indexOf(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}

// Jede Spalte jeder Tabelle steht mit Namen und Typ in der Beschreibungsdatei.
// Ohne sie beschreibt die Datei eine andere Datenüberlassung als die, die
// danebenliegt.
func TestIndexXMLDescribesEveryColumn(t *testing.T) {
	d := sampleDataset()
	root := parseTree(t, string(RenderIndexXML(d)))

	tables := findAll(root, "Table")
	if len(tables) != len(d.Tables) {
		t.Fatalf("index.xml beschreibt %d Tabellen, das Dataset hat %d", len(tables), len(d.Tables))
	}
	for i, table := range tables {
		want := d.Tables[i]
		// Die Kodierungsangabe gehört an jede Tabelle. Fehlt sie, liest eine
		// Prüfsoftware die CSV als ANSI: aus „Bürobedarf" wird „BÃ¼robedarf",
		// und der Prüfer bekommt eine Überlassung, die er beanstanden muss.
		if len(findAll(table, "UTF8")) != 1 {
			t.Errorf("Tabelle %s: die Kodierungsangabe <UTF8/> fehlt in index.xml", want.Name)
		}
		if got := text(table, "URL"); got != want.FileName {
			t.Errorf("Tabelle %d: URL %q statt %q", i, got, want.FileName)
		}
		if got := text(table, "Name"); got != want.Name {
			t.Errorf("Tabelle %d: Name %q statt %q", i, got, want.Name)
		}
		columns := findAll(table, "VariableColumn")
		if len(columns) != len(want.Fields) {
			t.Fatalf("Tabelle %s: %d Spalten beschrieben, %d vorhanden",
				want.Name, len(columns), len(want.Fields))
		}
		for j, column := range columns {
			field := want.Fields[j]
			if got := text(column, "Name"); got != field.Name {
				t.Errorf("Tabelle %s Spalte %d: %q statt %q", want.Name, j, got, field.Name)
			}
			wantType := map[FieldType]string{
				FieldNumeric: "Numeric", FieldDate: "Date", FieldAlphaNumeric: "AlphaNumeric",
			}[field.Type]
			if len(findAll(column, wantType)) != 1 {
				t.Errorf("Tabelle %s Spalte %s: der Typ %s fehlt", want.Name, field.Name, wantType)
			}
		}
	}
}

// --- Feldbeschreibung ------------------------------------------------------

// Die Feldbeschreibung muss jede Spalte jeder Tabelle im Klartext nennen. Der
// Test vergleicht die Feldlisten, nicht eine Stichprobe: eine vergessene Spalte
// ist genau die, nach der später gefragt wird.
func TestFieldDocNamesEveryColumn(t *testing.T) {
	d := sampleDataset()
	doc := string(RenderFieldDoc(d))

	for _, table := range d.Tables {
		if !strings.Contains(doc, table.FileName) {
			t.Errorf("die Feldbeschreibung nennt die Datei %s nicht", table.FileName)
		}
		for _, f := range table.Fields {
			if !strings.Contains(doc, f.Name) {
				t.Errorf("Tabelle %s: die Spalte %s fehlt in der Feldbeschreibung", table.Name, f.Name)
			}
			if f.Description != "" && !strings.Contains(doc, escapeCell(f.Description)) {
				t.Errorf("Tabelle %s, Spalte %s: die Erläuterung fehlt", table.Name, f.Name)
			}
		}
	}
}

// Das Schlüsselverzeichnis steht im Klartext in der Feldbeschreibung — und zwar
// aus derselben Tabelle, die exportiert wird. Ein senkrechter Strich im Text
// darf die Markdown-Tabelle nicht zerlegen.
func TestFieldDocRendersKeyDirectory(t *testing.T) {
	doc := string(RenderFieldDoc(sampleDataset()))
	if !strings.Contains(doc, "## Schlüsselwerte im Klartext") {
		t.Fatal("der Abschnitt mit den Schlüsselwerten fehlt")
	}
	if !strings.Contains(doc, `Pipe \| im Text`) {
		t.Error("ein senkrechter Strich im Text wurde nicht entschärft")
	}
}

// Das Verfahren, mit dem sich die Kette nachrechnen lässt, muss beschrieben
// sein — sonst sind Vorgänger- und Eigenhash zwei Spalten, die niemand prüfen
// kann.
func TestFieldDocDescribesHashChain(t *testing.T) {
	doc := string(RenderFieldDoc(sampleDataset()))
	for _, want := range []string{
		"Die Hash-Chain nachrechnen",
		"Längenpräfix",
		"line_pos", "line_amount", "line_text",
		"created_at", "entertainment",
		"SHA-256",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("die Beschreibung der Kanonisierung nennt %q nicht", want)
		}
	}
}

// --- Formatierung ----------------------------------------------------------

func TestAmountFormatsWithDotAndTwoPlaces(t *testing.T) {
	cases := map[int64]string{0: "0.00", 5: "0.05", 119_00: "119.00", -42_50: "-42.50", 1_234_567: "12345.67"}
	for cents, want := range cases {
		if got := Amount(domain.Cents(cents)); got != want {
			t.Errorf("%d Cent: %q statt %q", cents, got, want)
		}
	}
}

func TestSafeNameStripsPathSeparators(t *testing.T) {
	cases := map[string]string{
		// Die Trennzeichen werden ersetzt, und der führende Punkt fällt weg:
		// eine Datei, die im Export mit einem Punkt beginnt, wäre auf jedem
		// Unix-System unsichtbar.
		"../../etc/passwd": "_.._etc_passwd",
		"RE 2026-0001":     "RE 2026-0001",
		`a:b*c?d"e<f>g|h`:  "a_b_c_d_e_f_g_h",
		"   ":              "unbenannt",
	}
	for in, want := range cases {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q, erwartet %q", in, got, want)
		}
	}
}

// --- Hilfen ----------------------------------------------------------------

// parseCSV liest eine Datei nach RFC 4180 mit Semikolon als Trennzeichen. Der
// Leser ist bewusst eigener Code und nicht encoding/csv: geprüft werden soll
// die Datei, nicht die Übereinstimmung zweier Bibliotheken.
func parseCSV(t *testing.T, data string) [][]string {
	t.Helper()
	var rows [][]string
	var row []string
	var field strings.Builder
	inQuotes := false

	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case inQuotes && c == '"' && i+1 < len(data) && data[i+1] == '"':
			field.WriteByte('"')
			i++
		case c == '"':
			inQuotes = !inQuotes
		case !inQuotes && c == ';':
			row = append(row, field.String())
			field.Reset()
		case !inQuotes && c == '\r' && i+1 < len(data) && data[i+1] == '\n':
			row = append(row, field.String())
			field.Reset()
			rows = append(rows, row)
			row = nil
			i++
		default:
			field.WriteByte(c)
		}
	}
	if field.Len() > 0 || len(row) > 0 {
		t.Fatalf("die Datei endet nicht mit einem vollständigen Datensatz")
	}
	return rows
}

// node ist ein Element des geparsten XML-Baums.
type node struct {
	name     string
	content  string
	children []*node
}

func parseTree(t *testing.T, data string) *node {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(data))
	var stack []*node
	var root *node
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch tok := token.(type) {
		case xml.StartElement:
			n := &node{name: tok.Name.Local}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].content += string(tok)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		t.Fatal("index.xml hat kein Wurzelelement")
	}
	return root
}

func findAll(n *node, name string) []*node {
	var out []*node
	for _, c := range n.children {
		if c.name == name {
			out = append(out, c)
		}
		out = append(out, findAll(c, name)...)
	}
	return out
}

func text(n *node, name string) string {
	for _, c := range n.children {
		if c.name == name {
			return strings.TrimSpace(c.content)
		}
	}
	return ""
}

// parseDTD liest die Inhaltsmodelle aus den ELEMENT-Deklarationen.
var (
	elementDecl = regexp.MustCompile(`(?s)<!ELEMENT\s+(\w+)\s+\((.*?)\)>`)
	// Ein leeres Element wird ohne Klammern deklariert. Ohne diese zweite
	// Zeile hielte der Test <UTF8/> für undeklariert und beanstandete genau
	// die Angabe, die die Kodierung nennt.
	emptyElementDecl = regexp.MustCompile(`<!ELEMENT\s+(\w+)\s+EMPTY\s*>`)
)

func parseDTD(t *testing.T, dtd string) map[string]*regexp.Regexp {
	t.Helper()
	models := map[string]*regexp.Regexp{}
	for _, m := range emptyElementDecl.FindAllStringSubmatch(dtd, -1) {
		models[m[1]] = regexp.MustCompile(`^$`)
	}
	for _, m := range elementDecl.FindAllStringSubmatch(dtd, -1) {
		name, model := m[1], strings.Join(strings.Fields(m[2]), "")
		if model == "#PCDATA" {
			models[name] = regexp.MustCompile(`^$`)
			continue
		}
		// Das Inhaltsmodell wird zu einem Ausdruck über die Kindernamen: jeder
		// Name wird zu einem Muster, Komma verschwindet, Klammern und
		// Häufigkeitszeichen bleiben stehen.
		pattern := regexp.MustCompile(`\w+`).ReplaceAllString(model, "(?:$0\x00)")
		pattern = strings.ReplaceAll(pattern, ",", "")
		models[name] = regexp.MustCompile("^" + pattern + "$")
	}
	if len(models) == 0 {
		t.Fatal("die DTD enthält keine ELEMENT-Deklaration")
	}
	return models
}

func validateAgainstDTD(t *testing.T, models map[string]*regexp.Regexp, n *node) {
	t.Helper()
	model, ok := models[n.name]
	if !ok {
		t.Errorf("das Element %s ist in %s nicht deklariert", n.name, DTDFileName)
		return
	}
	var children strings.Builder
	for _, c := range n.children {
		children.WriteString(c.name)
		children.WriteByte(0)
	}
	if !model.MatchString(children.String()) {
		t.Errorf("das Element %s hat die Kinder %s — das Inhaltsmodell der DTD lässt das nicht zu",
			n.name, strings.ReplaceAll(children.String(), "\x00", " "))
	}
	for _, c := range n.children {
		validateAgainstDTD(t, models, c)
	}
}
