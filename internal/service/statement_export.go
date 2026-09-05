package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ausgabe des Jahresabschlusses als Datei.
//
// Beide Formate entstehen aus derselben Gliederung wie die Ansicht. Das ist der
// Punkt: eine PDF-Bilanz, die anders zustande kommt als die Bilanz auf dem
// Schirm, ist ein zweiter Abschluss — und dann ist offen, welcher gilt.

// ExportCSV schreibt die Gliederung als CSV mit Semikolon, wie es die deutsche
// Tabellenkalkulation erwartet.
func (s *StatementService) ExportCSV(ctx context.Context, year int, depth domain.StatementDepth) (string, error) {
	// Ohne ausdrückliche Tiefe wird voll gegliedert, und nicht in der Tiefe der
	// Größenklasse. Die CSV ist der Datenexport und nicht das offenzulegende
	// Dokument: eine Tabelle, die für eine Kleinstgesellschaft nur die
	// Buchstabenzeilen enthält, taugt zum Weiterrechnen nicht. Das PDF bleibt
	// bei der Tiefe der Offenlegung.
	if depth == "" {
		depth = domain.DepthFull
	}
	fs, err := s.Build(ctx, year, depth)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	// Die Kopfzeilen tragen die Pflichtangaben mit: eine Gliederung ohne die
	// Firma, zu der sie gehört, ist ein Zahlenblock.
	writeRow(&b, "Jahresabschluss", fs.Header.CompanyName, fs.Header.LegalForm, "", "", "", "")
	writeRow(&b, "Sitz", fs.Header.Seat, fs.Header.RegisterCourt, fs.Header.RegisterNumber, "", "", "")
	writeRow(&b, "Geschäftsjahr", fmt.Sprintf("%d", fs.Header.FiscalYear),
		fs.Header.StartDate, fs.Header.ClosingDate, "", "", "")
	writeRow(&b, "Größenklasse", fs.SizeClass.Class.Label(), fs.SizeClass.Class.Reference(),
		fs.Statement.Depth.Label(), "", "", "")
	b.WriteString("\n")

	writeRow(&b, "Abschnitt", "Schlüssel", "Ordnungszahl", "Bezeichnung", "Ebene",
		fmt.Sprintf("%d", fs.Header.FiscalYear), "Vorjahr", "Entfällt")

	for _, group := range [][]domain.StatementLine{
		fs.Statement.Assets, fs.Statement.Liabilities, fs.Statement.Income, fs.Statement.Statistical,
	} {
		for _, line := range group {
			// Die leeren Posten des § 265 Abs. 8 HGB bleiben in der Tabelle
			// stehen und tragen den Merker: die CSV ist der Datenexport, und
			// wer eine Zahl sucht, soll die Zeile finden, auch wenn sie null
			// ist. Welche Zeilen das Dokument weglässt, sagt die Spalte —
			// dieselbe Regel wie in Ansicht und PDF, nur nicht angewendet.
			writeRow(&b, string(line.Section), line.Key, line.Ordinal, line.Label,
				fmt.Sprintf("%d", line.Level), germanAmount(line.Amount), germanAmount(line.PriorAmount),
				omittedMark(line.Omitted))
		}
	}

	b.WriteString("\n")
	writeRow(&b, "aktiva", "summe.aktiva", "", "Summe Aktiva", "0",
		germanAmount(fs.Statement.TotalAssets), germanAmount(fs.Statement.TotalAssetsPrior), "")
	writeRow(&b, "passiva", "summe.passiva", "", "Summe Passiva", "0",
		germanAmount(fs.Statement.TotalLiabilities), germanAmount(fs.Statement.TotalLiabilitiesPrior), "")
	writeRow(&b, "guv", "jahresergebnis", "", "Jahresüberschuss/Jahresfehlbetrag", "0",
		germanAmount(fs.Statement.NetIncome), germanAmount(fs.Statement.NetIncomePrior), "")

	for _, row := range fs.Maturities.Rows {
		writeRow(&b, "restlaufzeiten", row.Key, "", row.Label, "0",
			germanAmount(row.Total), "", "")
		writeRow(&b, "restlaufzeiten", row.Key+".bis1", "", row.Label+" — bis ein Jahr", "1",
			germanAmount(row.UpToOneYear), "", "")
		writeRow(&b, "restlaufzeiten", row.Key+".ueber1", "", row.Label+" — über ein Jahr", "1",
			germanAmount(row.OverOneYear), "", "")
		writeRow(&b, "restlaufzeiten", row.Key+".ueber5", "", row.Label+" — über fünf Jahre", "1",
			germanAmount(row.OverFiveYears), "", "")
		writeRow(&b, "restlaufzeiten", row.Key+".ohne", "", row.Label+" — ohne Fälligkeit", "1",
			germanAmount(row.Undated), "", "")
	}

	s.logExport(ctx, year, "CSV")
	return b.String(), nil
}

// writeRow schreibt eine CSV-Zeile. Semikolon trennt, Anführungszeichen
// umschließen jedes Feld — dann ist ein Semikolon in einer Postenbezeichnung
// kein Feldtrenner mehr.
func writeRow(b *strings.Builder, fields ...string) {
	for i, field := range fields {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteByte('"')
		b.WriteString(strings.ReplaceAll(field, `"`, `""`))
		b.WriteByte('"')
	}
	b.WriteString("\r\n")
}

// omittedMark schreibt den Merker des § 265 Abs. 8 HGB als Wort. Ein leeres
// Feld heißt: der Posten wird ausgewiesen.
func omittedMark(omitted bool) string {
	if omitted {
		return "ja"
	}
	return ""
}

// germanAmount schreibt den Betrag mit Dezimalkomma und ohne
// Tausendertrennung: gruppiert wäre er in der Tabellenkalkulation Text.
func germanAmount(c domain.Cents) string {
	return strings.Replace(c.Decimal(), ".", ",", 1)
}

// ExportPDF setzt Bilanz und Gewinn- und Verlustrechnung als Dokument.
func (s *StatementService) ExportPDF(ctx context.Context, year int, depth domain.StatementDepth) ([]byte, error) {
	if s.renderer == nil {
		return nil, fmt.Errorf("der PDF-Renderer ist nicht verfügbar")
	}
	fs, err := s.Build(ctx, year, depth)
	if err != nil {
		return nil, err
	}
	pdf, err := s.renderer.RenderDocumentPDF(ctx, statementTypst(fs),
		fmt.Sprintf("Jahresabschluss %d", fs.Header.FiscalYear))
	if err != nil {
		return nil, err
	}
	s.logExport(ctx, year, "PDF")
	return pdf, nil
}

func (s *StatementService) logExport(ctx context.Context, year int, format string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "JAHRESABSCHLUSS",
		fmt.Sprintf("%d", year),
		fmt.Sprintf("Bilanz und Gewinn- und Verlustrechnung %d als %s ausgegeben", year, format))
}

// statementTypst baut die Vorlage für das Dokument.
//
// Sie steht hier und nicht in internal/invoice, weil sie nichts über Rechnungen
// weiß: sie setzt eine Gliederung, deren Zeilen schon fertig gerechnet sind.
func statementTypst(fs *domain.FinancialStatement) string {
	var b strings.Builder
	b.WriteString(`#set page(paper: "a4", margin: (x: 2cm, y: 2cm))
#set text(font: ("Manrope", "Helvetica", "sans-serif"), size: 9pt, lang: "de")
#set par(justify: false)
#show heading: set block(above: 1.2em, below: 0.6em)

`)
	// PDF/A verlangt ein Dokumentdatum. Genommen wird der Abschlussstichtag und
	// nicht der Tag des Ausdrucks: zwei Ausdrucke desselben Abschlusses sollen
	// gleich sein, sonst wäre das „inhaltlich identische Mehrstück" der GoBD
	// Rz. 76 Abs. 2 vom Zufall des Druckzeitpunkts abhängig.
	fmt.Fprintf(&b, "#set document(title: %s, date: %s)\n\n",
		typstString(fmt.Sprintf("Jahresabschluss %d — %s", fs.Header.FiscalYear, fs.Header.CompanyName)),
		typstDate(fs.Header.ClosingDate))

	fmt.Fprintf(&b, "#text(size: 14pt, weight: \"bold\")[%s]\n\n", typstText(headerTitle(fs.Header)))
	fmt.Fprintf(&b, "#text(size: 9pt)[%s]\n\n", typstText(headerSubtitle(fs)))

	b.WriteString("= Bilanz\n\n")
	b.WriteString("== Aktiva\n\n")
	writeTypstTable(&b, fs.Statement.Assets, fs.Statement.HasPrior, fs.Header, "Summe Aktiva",
		fs.Statement.TotalAssets, fs.Statement.TotalAssetsPrior)
	b.WriteString("\n== Passiva\n\n")
	writeTypstTable(&b, fs.Statement.Liabilities, fs.Statement.HasPrior, fs.Header, "Summe Passiva",
		fs.Statement.TotalLiabilities, fs.Statement.TotalLiabilitiesPrior)

	b.WriteString("\n#pagebreak()\n= Gewinn- und Verlustrechnung\n\n")
	writeTypstTable(&b, fs.Statement.Income, fs.Statement.HasPrior, fs.Header, "",
		0, 0)

	b.WriteString("\n= Angaben unter der Bilanz\n\n")
	fmt.Fprintf(&b, "#text(size: 8pt)[%s]\n\n", typstText("Restlaufzeiten nach "+fs.Maturities.Reference))
	// „Über fünf Jahre" ist die Angabe, die § 268 Abs. 5 Satz 1 HGB neben der
	// bis zu einem Jahr ausdrücklich verlangt; „ohne Fälligkeit" steht daneben,
	// weil ein Posten ohne vereinbarte Frist sonst stillschweigend in einer der
	// Restlaufzeitspalten verschwände. Beide stehen in der Struktur — sie im
	// Dokument wegzulassen hieße, die Angabe zu kürzen.
	b.WriteString("#table(\n  columns: (1fr, auto, auto, auto, auto, auto),\n" +
		"  align: (left, right, right, right, right, right),\n  stroke: none,\n")
	b.WriteString("  table.hline(),\n")
	fmt.Fprintf(&b, "  [*Posten*], [*Gesamt*], [*bis 1 Jahr*], [*über 1 Jahr*], [*über 5 Jahre*], [*ohne Fälligkeit*],\n")
	b.WriteString("  table.hline(),\n")
	for _, row := range fs.Maturities.Rows {
		fmt.Fprintf(&b, "  [%s], [%s], [%s], [%s], [%s], [%s],\n",
			typstText(row.Label), typstText(row.Total.String()),
			typstText(row.UpToOneYear.String()), typstText(row.OverOneYear.String()),
			typstText(row.OverFiveYears.String()), typstText(row.Undated.String()))
	}
	b.WriteString("  table.hline(),\n)\n\n")

	if len(fs.Statement.Assignment.Unassigned) > 0 {
		b.WriteString("= Nicht zugeordnete Konten\n\n")
		for _, acc := range fs.Statement.Assignment.Unassigned {
			fmt.Fprintf(&b, "- %s\n", typstText(fmt.Sprintf("%s %s: %s €", acc.Number, acc.Name, acc.Amount)))
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "#text(size: 7pt)[%s]\n", typstText(fs.SizeClass.Reason))
	return b.String()
}

func headerTitle(h domain.StatementHeader) string {
	name := h.CompanyName
	if name == "" {
		name = "(Firma nicht erfasst)"
	}
	if h.LegalForm != "" && !strings.Contains(name, h.LegalForm) {
		name += " " + h.LegalForm
	}
	return name
}

func headerSubtitle(fs *domain.FinancialStatement) string {
	h := fs.Header
	parts := []string{fmt.Sprintf("Jahresabschluss zum %s", germanDate(h.ClosingDate))}
	if h.Seat != "" {
		parts = append(parts, "Sitz: "+h.Seat)
	}
	if h.RegisterCourt != "" || h.RegisterNumber != "" {
		parts = append(parts, strings.TrimSpace(h.RegisterCourt+" "+h.RegisterNumber))
	}
	parts = append(parts, h.Reference)
	parts = append(parts, fs.SizeClass.Class.Label()+" ("+fs.SizeClass.Class.Reference()+")")
	parts = append(parts, fs.Statement.Depth.Label())
	return strings.Join(parts, " · ")
}

// writeTypstTable setzt eine Gliederung als Tabelle mit Vorjahresspalte. Die
// Summenzeile bekommt die buchhalterische Doppellinie.
func writeTypstTable(
	b *strings.Builder, lines []domain.StatementLine, hasPrior bool,
	header domain.StatementHeader, totalLabel string, total, totalPrior domain.Cents,
) {
	columns := "(1fr, auto)"
	align := "(left, right)"
	if hasPrior {
		columns = "(1fr, auto, auto)"
		align = "(left, right, right)"
	}
	fmt.Fprintf(b, "#table(\n  columns: %s,\n  align: %s,\n  stroke: none,\n  inset: (x: 4pt, y: 3pt),\n", columns, align)
	b.WriteString("  table.hline(),\n")
	fmt.Fprintf(b, "  [*Posten*], [*%d*]", header.FiscalYear)
	if hasPrior {
		fmt.Fprintf(b, ", [*%d*]", header.PriorYear)
	}
	b.WriteString(",\n  table.hline(),\n")

	for _, line := range lines {
		// § 265 Abs. 8 HGB: der leere Posten entfällt. Der Merker kommt aus dem
		// Aufbau, damit das Dokument dieselben Zeilen zeigt wie die Ansicht.
		if line.Omitted {
			continue
		}
		label := strings.Repeat("#h(1em)", line.Level-1)
		if line.Ordinal != "" {
			label += typstText(line.Ordinal) + " "
		}
		label += typstText(line.Label)
		weight := ""
		if line.Level == 1 || line.IsSubtotal {
			weight = "*"
		}
		fmt.Fprintf(b, "  [%s], [%s%s%s]", label, weight, typstText(line.Amount.String()), weight)
		if hasPrior {
			fmt.Fprintf(b, ", [%s]", typstText(line.PriorAmount.String()))
		}
		b.WriteString(",\n")
	}

	if totalLabel != "" {
		// Die Doppellinie ist die buchhalterische Kennzeichnung der Summe.
		b.WriteString("  table.hline(),\n  table.hline(),\n")
		fmt.Fprintf(b, "  [*%s*], [*%s*]", typstText(totalLabel), typstText(total.String()))
		if hasPrior {
			fmt.Fprintf(b, ", [*%s*]", typstText(totalPrior.String()))
		}
		b.WriteString(",\n")
	}
	b.WriteString("  table.hline(),\n)\n")
}

// typstText entschärft die Zeichen, die Typst als Auszeichnung liest.
func typstText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`, `#`, `\#`, `[`, `\[`, `]`, `\]`,
		`*`, `\*`, `_`, `\_`, `$`, `\$`, `@`, `\@`, `<`, `\<`, `>`, `\>`,
	)
	return replacer.Replace(value)
}

// typstDate schreibt ein ISO-Datum als Typst-Datum. Ohne gültiges Datum bleibt
// es bei "auto" — dann fehlt die Angabe, statt eine falsche zu behaupten.
func typstDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return "auto"
	}
	return fmt.Sprintf("datetime(year: %d, month: %d, day: %d)", t.Year(), int(t.Month()), t.Day())
}

// typstString schreibt einen Wert als Typst-Zeichenkette.
func typstString(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}
