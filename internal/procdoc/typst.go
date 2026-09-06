package procdoc

import (
	"fmt"
	"strings"
	"time"
)

// PDFFileName ist der Dateiname des gesetzten Dokuments.
func PDFFileName(company, version string) string {
	return strings.TrimSuffix(FileName(company, version), ".md") + ".pdf"
}

// Typst übersetzt die erzeugte Verfahrensdokumentation in eine Typst-Vorlage.
//
// Übersetzt wird das erzeugte Markdown und nicht ein zweites Mal die Eingabe.
// Das ist die ganze Absicht: gäbe es neben der Markdown-Vorlage eine
// Typst-Vorlage, wären es zwei Fassungen desselben Dokuments, die
// auseinanderlaufen — und die Frage, welche gilt, wäre nicht mehr zu
// beantworten. So ist das PDF der Satz genau des Textes, der auch als Markdown
// im Belegspeicher liegt; beide sind aus einer Quelle.
//
// Der Umfang der Übersetzung ist der Umfang der Vorlage: Überschriften,
// Absätze, Aufzählungen, nummerierte Schritte, Tabellen, fetter Text und
// Textstellen in fester Breite. Was die Vorlage nicht verwendet, kann sie auch
// nicht erzeugen; unbekannte Zeilen laufen als Absatz durch, statt verloren zu
// gehen.
func Typst(markdown, title string, date time.Time) string {
	var b strings.Builder
	b.WriteString(`#set page(paper: "a4", margin: (x: 2cm, y: 2cm), numbering: "1 / 1")
#set text(font: ("Manrope", "Helvetica", "sans-serif"), size: 9.5pt, lang: "de")
#set par(justify: false, leading: 0.65em)
#show heading: set block(above: 1.4em, below: 0.7em)
#set heading(numbering: none)

`)
	fmt.Fprintf(&b, "#set document(title: %s, date: %s)\n\n",
		typstString(title), typstDateTime(date))

	lines := strings.Split(markdown, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			b.WriteString("\n")

		case strings.HasPrefix(trimmed, "|"):
			// Eine Tabelle reicht so weit, wie Zeilen mit Strich beginnen.
			var rows []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				rows = append(rows, strings.TrimSpace(lines[i]))
				i++
			}
			i--
			writeTypstTable(&b, rows)

		case strings.HasPrefix(trimmed, "#"):
			level := 0
			for level < len(trimmed) && trimmed[level] == '#' {
				level++
			}
			text := strings.TrimSpace(trimmed[level:])
			b.WriteString(strings.Repeat("=", level) + " " + inlineTypst(text) + "\n")

		case strings.HasPrefix(trimmed, "- "):
			// Die Einrückung bleibt erhalten: sie trägt in Typst wie in
			// Markdown die Schachtelung der Aufzählung.
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			b.WriteString(indent + "- " + inlineTypst(strings.TrimPrefix(trimmed, "- ")) + "\n")

		case isOrderedItem(trimmed):
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			b.WriteString(indent + "+ " + inlineTypst(afterOrderedMarker(trimmed)) + "\n")

		default:
			// Ein Absatz, der mit einem Zeichen beginnt, das Typst am
			// Zeilenanfang deutet, bekommt einen Schutz davor — sonst würde aus
			// einem Satz eine Überschrift oder eine Aufzählung.
			text := inlineTypst(trimmed)
			if strings.HasPrefix(text, "=") || strings.HasPrefix(text, "+") ||
				strings.HasPrefix(text, "/") {
				text = `\` + text
			}
			b.WriteString(text + "\n")
		}
	}
	return b.String()
}

// writeTypstTable setzt eine Markdown-Tabelle. Die Trennzeile aus Strichen
// entfällt — sie ist eine Markdown-Eigenheit und trägt keinen Inhalt.
func writeTypstTable(b *strings.Builder, rows []string) {
	cells := make([][]string, 0, len(rows))
	for _, row := range rows {
		fields := splitTableRow(row)
		if isSeparatorRow(fields) {
			continue
		}
		cells = append(cells, fields)
	}
	if len(cells) == 0 {
		return
	}
	columns := 0
	for _, row := range cells {
		if len(row) > columns {
			columns = len(row)
		}
	}

	fmt.Fprintf(b, "#table(\n  columns: %d,\n  stroke: (x, y) => (top: if y <= 1 { 0.5pt } else { 0pt }, bottom: 0.5pt),\n  inset: 5pt,\n", columns)
	for rowIndex, row := range cells {
		b.WriteString("  ")
		for column := 0; column < columns; column++ {
			value := ""
			if column < len(row) {
				value = row[column]
			}
			content := inlineTypst(value)
			// Die Kopfzeile wird fett gesetzt; sie ist die einzige Zeile, die
			// beschreibt statt zu behaupten.
			if rowIndex == 0 {
				content = "*" + content + "*"
			}
			fmt.Fprintf(b, "[%s], ", content)
		}
		b.WriteString("\n")
	}
	b.WriteString(")\n")
}

func splitTableRow(row string) []string {
	row = strings.TrimPrefix(row, "|")
	row = strings.TrimSuffix(row, "|")
	fields := strings.Split(row, "|")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}

// isSeparatorRow erkennt die Trennzeile `| --- | --- |`.
func isSeparatorRow(fields []string) bool {
	for _, field := range fields {
		if strings.Trim(field, "-: ") != "" {
			return false
		}
	}
	return len(fields) > 0
}

func isOrderedItem(line string) bool {
	digits := 0
	for digits < len(line) && line[digits] >= '0' && line[digits] <= '9' {
		digits++
	}
	return digits > 0 && strings.HasPrefix(line[digits:], ". ")
}

func afterOrderedMarker(line string) string {
	if index := strings.Index(line, ". "); index >= 0 {
		return line[index+2:]
	}
	return line
}

// inlineTypst übersetzt eine Zeile Fließtext.
//
// Erst wird alles geschützt, was Typst als Auszeichnung läse, dann werden die
// beiden Markdown-Auszeichnungen der Vorlage wieder eingesetzt: **fett** und
// `feste Breite`. Die Reihenfolge ist wesentlich — würde zuerst ausgezeichnet
// und dann geschützt, entkäme das Sternchen der Auszeichnung dem Schutz nicht,
// sondern der Schutz der Auszeichnung.
func inlineTypst(text string) string {
	var out strings.Builder
	rest := text
	for {
		bold := strings.Index(rest, "**")
		code := strings.Index(rest, "`")
		switch {
		case bold >= 0 && (code < 0 || bold < code):
			end := strings.Index(rest[bold+2:], "**")
			if end < 0 {
				out.WriteString(typstText(rest))
				return out.String()
			}
			out.WriteString(typstText(rest[:bold]))
			out.WriteString("*" + typstText(rest[bold+2:bold+2+end]) + "*")
			rest = rest[bold+2+end+2:]

		case code >= 0:
			end := strings.Index(rest[code+1:], "`")
			if end < 0 {
				out.WriteString(typstText(rest))
				return out.String()
			}
			out.WriteString(typstText(rest[:code]))
			// Der Rohtext steht in einem Kasten mit fester Breite; sein Inhalt
			// wird nicht geschützt, weil Typst ihn ohnehin nicht deutet.
			out.WriteString("#raw(" + typstString(rest[code+1:code+1+end]) + ")")
			rest = rest[code+1+end+1:]

		default:
			out.WriteString(typstText(rest))
			return out.String()
		}
	}
}

// typstText schützt, was Typst sonst als Auszeichnung läse.
func typstText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`, `#`, `\#`, `[`, `\[`, `]`, `\]`,
		`*`, `\*`, `_`, `\_`, `$`, `\$`, `@`, `\@`, `<`, `\<`, `>`, `\>`,
		"`", "\\`",
	)
	return replacer.Replace(value)
}

func typstString(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

func typstDateTime(t time.Time) string {
	if t.IsZero() {
		return "auto"
	}
	t = t.UTC()
	return fmt.Sprintf("datetime(year: %d, month: %d, day: %d)", t.Year(), int(t.Month()), t.Day())
}
