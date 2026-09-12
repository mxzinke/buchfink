package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
)

// Die Eröffnungsbilanz auf den Tag der Beurkundung.
//
// § 242 Abs. 1 HGB verlangt sie zu Beginn des Handelsgewerbes — bei einer
// Kapitalgesellschaft also auf den Tag, an dem der Gesellschaftsvertrag
// beurkundet wurde, nicht auf den Tag der Eintragung. Sie ist der eine
// Gründungsvorgang, den Buchfink vollständig aus den eigenen Zahlen kann: an
// diesem Tag steht in den Büchern die Zeichnung des Stammkapitals und, soweit
// schon geflossen, die Einlage.
//
// Aufgestellt und nicht gebucht: die Eröffnungsbilanz ist eine Aufstellung über
// den Stand, kein Geschäftsvorfall. Die Buchungen, aus denen sie entsteht,
// stehen bereits im Journal.

// OpeningBalanceSheet ist die Eröffnungsbilanz mit allem, was ihr Kopf braucht.
type OpeningBalanceSheet struct {
	// AsOf ist der Stichtag: der Tag der Beurkundung.
	AsOf      string                 `json:"asOf"`
	Header    domain.StatementHeader `json:"header"`
	Statement domain.Statement       `json:"statement"`
	// Balances sagt, ob Aktiva und Passiva übereinstimmen. Tun sie es nicht,
	// fehlt eine Buchung — und das ist ein Befund und kein Rundungsfehler.
	Balances bool         `json:"balances"`
	Assets   domain.Cents `json:"assets"`
	Equity   domain.Cents `json:"equity"`
	// Findings nennen, was der Aufstellung entgegensteht — je Befund ein Satz.
	Findings []string `json:"findings"`
	// DocumentID ist die abgelegte Fassung, falls es schon eine gibt.
	DocumentID uint   `json:"documentId,omitempty"`
	FiledOn    string `json:"filedOn,omitempty"`
}

// StatementAtSource ist der Ausschnitt des Abschlusses, den die Eröffnungsbilanz
// braucht: eine Gliederung auf einen Stichtag und der Kopf dazu.
type StatementAtSource interface {
	StatementAt(ctx context.Context, cutoff string, depth domain.StatementDepth) (*domain.Statement, error)
	Header(ctx context.Context, year int) (domain.StatementHeader, error)
}

// SetStatementSource koppelt die Gliederung an die Gründung.
func (s *FoundationService) SetStatementSource(src StatementAtSource) { s.statements = src }

// SetDocumentStore koppelt die Dokumentenablage an die Gründung. Ohne sie
// entsteht die Eröffnungsbilanz als Vorschau und wird nicht abgelegt.
func (s *FoundationService) SetDocumentStore(svc *DocumentService) { s.documents = svc }

// SetRenderer koppelt den Satz an die Gründung. Ohne ihn gibt es die
// Eröffnungsbilanz als Zahlenwerk, aber nicht als Dokument.
func (s *FoundationService) SetRenderer(r DocumentRenderer) { s.renderer = r }

// OpeningBalance stellt die Eröffnungsbilanz auf, ohne sie abzulegen.
func (s *FoundationService) OpeningBalance(ctx context.Context) (*OpeningBalanceSheet, error) {
	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if f == nil || len(f.NotarizedOn) != 10 {
		return nil, fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}
	if s.statements == nil {
		return nil, fmt.Errorf("die Gliederung ist nicht verfügbar")
	}

	stmt, err := s.statements.StatementAt(ctx, f.NotarizedOn, domain.DepthFull)
	if err != nil {
		return nil, err
	}

	out := &OpeningBalanceSheet{
		AsOf: f.NotarizedOn, Statement: *stmt, Findings: []string{},
	}

	// Der Kopf des Geschäftsjahres, in das die Beurkundung fällt — Firma, Sitz,
	// Register. Der Stichtag wird darin auf den Beurkundungstag gesetzt: der
	// Abschlussstichtag des Jahres ist nicht der Stichtag dieser Aufstellung.
	if header, err := s.statements.Header(ctx, stmt.FiscalYear); err == nil {
		header.ClosingDate = f.NotarizedOn
		header.StartDate = f.NotarizedOn
		header.Reference = "§ 242 Abs. 1 HGB"
		out.Header = header
		for _, missing := range header.Missing {
			// Registergericht und -nummer fehlen vor der Eintragung
			// notwendigerweise. Sie hier als Mangel zu führen hieße, nach etwas
			// zu fragen, das es noch nicht geben kann.
			if !f.IsRegistered() && (missing == "Registergericht" || missing == "Registernummer") {
				continue
			}
			out.Findings = append(out.Findings, fmt.Sprintf(
				"Die Pflichtangabe %q fehlt in den Unternehmensdaten (§ 264 Abs. 1a HGB).", missing))
		}
	} else {
		return nil, fmt.Errorf("Unternehmensangaben für die Eröffnungsbilanz lesen: %w", err)
	}

	out.Assets = sumStatement(stmt.Assets)
	out.Equity = sumStatement(stmt.Liabilities)
	out.Balances = out.Assets == out.Equity
	if !out.Balances {
		out.Findings = append(out.Findings, fmt.Sprintf(
			"Aktiva %s € und Passiva %s € gehen auseinander. Die Eröffnungsbilanz entsteht aus den "+
				"Buchungen bis zum %s — solange sie nicht aufgeht, fehlt eine davon.",
			out.Assets, out.Equity, germanDate(f.NotarizedOn)))
	}
	if out.Assets == 0 {
		out.Findings = append(out.Findings, fmt.Sprintf(
			"Zum %s steht keine Buchung in den Büchern. Die Zeichnung des Stammkapitals gehört auf "+
				"den Tag der Beurkundung — buchen Sie sie über „Gründung buchen“.",
			germanDate(f.NotarizedOn)))
	}

	entries, err := s.journalRepo.FindByAccount(ctx, domain.AccountGezeichnetesKapital, 0)
	if err != nil {
		return nil, err
	}
	var capital domain.Cents
	for _, entry := range entries {
		if entry.BookingDate <= f.NotarizedOn {
			for _, line := range entry.Lines {
				if line.Account == domain.AccountGezeichnetesKapital {
					if line.Side == domain.SideCredit {
						capital += line.Amount
					} else {
						capital -= line.Amount
					}
				}
			}
		}
	}
	if capital != f.ShareCapital {
		out.Findings = append(out.Findings, fmt.Sprintf("Das gezeichnete Kapital zum Stichtag beträgt %s €; laut Gründung sind %s € zu buchen. Bitte die Gründungsbuchungen vervollständigen.", capital, f.ShareCapital))
	}
	// The contribution form records money already paid at the opening date.
	// A balanced capital subscription alone must not hide a missing payment.
	contributions, err := s.journalRepo.FindByAccount(ctx, domain.AccountAusstehendeEinlagenGeford, 0)
	if err != nil {
		return nil, err
	}
	contributed := capital
	for _, entry := range contributions {
		if entry.BookingDate > f.NotarizedOn {
			continue
		}
		for _, line := range entry.Lines {
			if line.Account != domain.AccountAusstehendeEinlagenGeford {
				continue
			}
			if line.Side == domain.SideCredit {
				contributed += line.Amount
			} else {
				contributed -= line.Amount
			}
		}
	}
	if contributed < f.PaidInCapital() {
		out.Findings = append(out.Findings, fmt.Sprintf("Laut Gründung wurden %s € eingezahlt; davon sind zum Stichtag nur %s € als Einlage gebucht. Bitte die Einzahlung vervollständigen.", f.PaidInCapital(), contributed))
	}

	if s.documents != nil {
		if filed, err := s.documents.ForDuty(ctx, DutyKeyEroeffnungsbilanz); err == nil && len(filed) > 0 {
			// Die zuletzt abgelegte Fassung. Eine neue ersetzt die alte nicht:
			// beide bleiben in der Ablage, und welche gilt, sagt ihr Datum.
			latest := filed[len(filed)-1]
			out.DocumentID = latest.ID
			out.FiledOn = latest.CreatedAt.In(time.Local).Format("2006-01-02")
		}
	}
	return out, nil
}

// sumStatement addiert die obersten Posten einer Seite.
//
// Nur Ebene eins: die Buchstaben des § 266 HGB tragen bereits die Summen ihrer
// Unterposten, und wer alle Ebenen addierte, zählte jeden Betrag mehrfach.
func sumStatement(lines []domain.StatementLine) domain.Cents {
	var sum domain.Cents
	for _, line := range lines {
		if line.Level == 1 && !line.IsSubtotal {
			sum += line.Amount
		}
	}
	return sum
}

// FileOpeningBalance stellt die Eröffnungsbilanz auf, setzt sie und legt sie in
// der Dokumentenablage ab.
//
// Abgelegt und nicht nur heruntergeladen: die Eröffnungsbilanz ist eine
// Unterlage des Unternehmens nach § 147 Abs. 1 Nr. 1 AO, und sie wird zehn Jahre
// gebraucht. Wer sie nur speichert, hat sie in drei Jahren nicht mehr.
func (s *FoundationService) FileOpeningBalance(ctx context.Context) (*domain.Document, error) {
	sheet, err := s.OpeningBalance(ctx)
	if err != nil {
		return nil, err
	}
	if !sheet.Balances {
		return nil, fmt.Errorf(
			"die Eröffnungsbilanz geht nicht auf: Aktiva %s €, Passiva %s €. Eine Bilanz, die nicht "+
				"aufgeht, ist keine", sheet.Assets, sheet.Equity)
	}
	if len(sheet.Findings) > 0 {
		return nil, fmt.Errorf("die Eröffnungsbilanz ist noch unvollständig: %s", strings.Join(sheet.Findings, " "))
	}
	if s.documents == nil {
		return nil, fmt.Errorf("die Dokumentenablage ist nicht verfügbar")
	}
	if s.renderer == nil {
		return nil, fmt.Errorf("der Satz ist nicht verfügbar: die Eröffnungsbilanz kann nicht gesetzt werden")
	}

	markdown := openingBalanceMarkdown(sheet)
	title := "Eröffnungsbilanz " + germanDate(sheet.AsOf)
	pdf, err := s.renderer.RenderDocumentPDF(ctx,
		procdoc.Typst(markdown, title, parseDayOrNow(sheet.AsOf)), "eroeffnungsbilanz-"+sheet.AsOf)
	if err != nil {
		return nil, fmt.Errorf("die Eröffnungsbilanz konnte nicht gesetzt werden: %w", err)
	}

	doc, err := s.documents.Attach(ctx, DocumentRequest{
		Kind:         domain.DocEroeffnungsbilanz,
		Title:        title,
		DocumentDate: sheet.AsOf,
		DutyKey:      DutyKeyEroeffnungsbilanz,
		GeneratedBy:  "eroeffnungsbilanz",
		FileName:     "Eroeffnungsbilanz_" + sheet.AsOf + ".pdf",
		Content:      pdf,
	})
	if err != nil {
		return nil, err
	}
	// Die Pflicht ist mit der Aufstellung erledigt. Sie danach noch als offen zu
	// führen wäre eine Nachfrage nach etwas, das in der Ablage liegt.
	if err := s.CompleteDuty(ctx, DutyKeyEroeffnungsbilanz, todayLocal(),
		"Aufgestellt und abgelegt: "+doc.FileName); err != nil {
		return doc, fmt.Errorf("Dokument abgelegt, aber Erledigungsvermerk nicht gespeichert: %w", err)
	}
	s.audit(ctx, domain.AuditActionCreate, doc.ID, fmt.Sprintf(
		"Eröffnungsbilanz auf den %s aufgestellt und abgelegt (§ 242 Abs. 1 HGB)", sheet.AsOf))
	return doc, nil
}

// openingBalanceMarkdown ist der Text der Eröffnungsbilanz.
//
// Als Markdown und nicht unmittelbar als Typst — derselbe Weg wie beim
// Eigenbeleg, beim Mahnschreiben und bei der Verfahrensdokumentation.
func openingBalanceMarkdown(sheet *OpeningBalanceSheet) string {
	var b strings.Builder
	b.WriteString("# Eröffnungsbilanz\n\n")

	h := sheet.Header
	name := h.FirmName
	if name == "" {
		name = h.CompanyName
	}
	fmt.Fprintf(&b, "**%s**\n\n", name)

	b.WriteString("| Angabe | Inhalt |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| Stichtag | %s |\n", germanDate(sheet.AsOf))
	fmt.Fprintf(&b, "| Aufgestellt am | %s |\n", germanDate(todayLocal()))
	if h.Seat != "" {
		fmt.Fprintf(&b, "| Sitz | %s |\n", h.Seat)
	}
	if h.RegisterCourt != "" || h.RegisterNumber != "" {
		fmt.Fprintf(&b, "| Register | %s |\n", strings.TrimSpace(h.RegisterCourt+" "+h.RegisterNumber))
	}
	fmt.Fprintf(&b, "| Rechtsgrundlage | § 242 Abs. 1 HGB |\n")
	b.WriteString("\n")

	writeOpeningSide(&b, "Aktiva", sheet.Statement.Assets, sheet.Assets)
	writeOpeningSide(&b, "Passiva", sheet.Statement.Liabilities, sheet.Equity)

	b.WriteString("## Hinweis\n\n")
	b.WriteString(
		"Die Eröffnungsbilanz ist zu Beginn des Handelsgewerbes aufzustellen (§ 242 Abs. 1 HGB). " +
			"Bei einer Kapitalgesellschaft ist das der Tag der notariellen Beurkundung des " +
			"Gesellschaftsvertrags: mit ihm entsteht die Vorgesellschaft, und mit ihr die " +
			"Buchführungspflicht.\n\n")
	b.WriteString(
		"Die Zahlen stammen aus den Buchungen bis zu diesem Tag. Aufwendungen für die Gründung " +
			"dürfen nicht aktiviert werden (§ 248 Abs. 1 Nr. 1 HGB); sie erscheinen deshalb in " +
			"keiner Position dieser Aufstellung.\n")
	return b.String()
}

// writeOpeningSide schreibt eine Seite der Bilanz als Tabelle.
//
// Nur die Posten mit Betrag: § 265 Abs. 8 HGB lässt einen Posten entfallen, der
// in beiden Jahren keinen trägt, und eine Eröffnungsbilanz besteht aus wenigen
// Zeilen. Die volle Gliederung mit lauter Nullen wäre vier Seiten Papier für
// zwei Zahlen.
func writeOpeningSide(b *strings.Builder, title string, lines []domain.StatementLine, total domain.Cents) {
	fmt.Fprintf(b, "## %s\n\n", title)
	b.WriteString("| Position | Betrag |\n| --- | --- |\n")
	for _, line := range lines {
		if line.Amount == 0 || line.IsSubtotal {
			continue
		}
		label := strings.TrimSpace(line.Ordinal + " " + line.Label)
		fmt.Fprintf(b, "| %s | %s € |\n", label, line.Amount)
	}
	fmt.Fprintf(b, "| **Summe %s** | **%s €** |\n\n", title, total)
}
