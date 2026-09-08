package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
)

// Das Datenblatt zum Fragebogen zur steuerlichen Erfassung.
//
// Der Fragebogen wird über Mein ELSTER übermittelt (§ 138 Abs. 1b AO); Buchfink
// übermittelt nichts selbst — ERiC bleibt außerhalb des Funktionsumfangs. Was
// Buchfink kann, ist die Angaben zusammenstellen, die dort abgefragt werden und
// die es bereits kennt: Firma, Sitz, Rechtsform, Beurkundung, Stammkapital,
// Gesellschafter, Bankverbindung.
//
// Zum Abtippen und nicht zum Übermitteln. Das ist keine Verlegenheitslösung: den
// Fragebogen füllt aus, wer die Verantwortung dafür trägt, und die Hälfte seiner
// Felder — voraussichtliche Umsätze, Beschäftigte, Betriebseröffnung — steht in
// keinem Konto.

// FragebogenSheet ist das Datenblatt vor seiner Ablage.
type FragebogenSheet struct {
	// Rows sind die Angaben, je Zeile eine Frage des Fragebogens.
	Rows []FragebogenRow `json:"rows"`
	// Open nennt die Felder, die der Fragebogen verlangt und die Buchfink nicht
	// kennt — damit niemand das Datenblatt für vollständig hält.
	Open []string `json:"open"`
	// DocumentID ist die abgelegte Fassung, falls es schon eine gibt.
	DocumentID uint   `json:"documentId,omitempty"`
	FiledOn    string `json:"filedOn,omitempty"`
}

// FragebogenRow ist eine Angabe des Datenblatts.
type FragebogenRow struct {
	Section string `json:"section"`
	Label   string `json:"label"`
	Value   string `json:"value"`
	// Missing sagt, dass die Angabe in den Stammdaten fehlt.
	Missing bool `json:"missing"`
}

// Fragebogen stellt das Datenblatt zusammen, ohne es abzulegen.
func (s *FoundationService) Fragebogen(ctx context.Context) (*FragebogenSheet, error) {
	f, err := s.foundationRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if f == nil || len(f.NotarizedOn) != 10 {
		return nil, fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, err
	}

	sheet := &FragebogenSheet{Rows: []FragebogenRow{}, Open: []string{}}
	add := func(section, label, value string) {
		sheet.Rows = append(sheet.Rows, FragebogenRow{
			Section: section, Label: label,
			Value: strings.TrimSpace(value), Missing: strings.TrimSpace(value) == "",
		})
	}

	const unternehmen = "Unternehmen"
	add(unternehmen, "Firma", cfg.FirmName())
	add(unternehmen, "Rechtsform", cfg.LegalForm)
	add(unternehmen, "Sitz", cfg.Seat)
	add(unternehmen, "Anschrift", strings.TrimSpace(cfg.Street+", "+cfg.ZipCity))
	if f.IsRegistered() {
		add(unternehmen, "Registergericht", f.RegisterCourt)
		add(unternehmen, "Registernummer", f.RegisterNumber)
	}

	const gruendung = "Gründung"
	add(gruendung, "Tag der Beurkundung", germanDate(f.NotarizedOn))
	if f.IsRegistered() {
		add(gruendung, "Tag der Eintragung", germanDate(f.RegisteredOn))
	}
	add(gruendung, "Stammkapital", f.ShareCapital.String()+" €")
	add(gruendung, "Davon geleistet", f.PaidInCapital().String()+" €")

	const beteiligte = "Beteiligte"
	for _, sh := range f.Shareholders {
		add(beteiligte, sh.Name, fmt.Sprintf("%s € (%s), geleistet %s €",
			sh.ShareCapital, sh.Kind.Label(), sh.PaidIn))
	}

	const steuer = "Steuerliche Angaben"
	add(steuer, "Finanzamt", cfg.TaxOffice)
	// Die Steuernummer ist das Ergebnis des Fragebogens und nicht seine Angabe.
	// Sie hier als Lücke zu führen wäre die Frage nach dem, was man gerade
	// beantragt.
	if strings.TrimSpace(cfg.VatID) != "" {
		add(steuer, "USt-IdNr.", cfg.VatID)
	}
	year := domain.GetFiscalYearForDate(f.NotarizedOn, cfg.FiscalYearStartMonth)
	add(steuer, "Voranmeldungszeitraum",
		vatPeriodLabel(accounting.RecommendedVatPeriod(year)))

	const bank = "Bankverbindung"
	add(bank, "Kontoinhaber", cfg.FirmName())
	add(bank, "IBAN", cfg.IBAN)
	add(bank, "BIC", cfg.BIC)
	add(bank, "Kreditinstitut", cfg.BankName)

	// Was der Fragebogen verlangt und Buchfink nicht wissen kann. Ohne diese
	// Liste läse sich das Datenblatt wie eine vollständige Antwort.
	sheet.Open = []string{
		"Voraussichtliche Umsätze und Gewinne des Gründungsjahres und des Folgejahres",
		"Tag der Betriebseröffnung und Art der ausgeübten Tätigkeit",
		"Zahl der Beschäftigten und ob Lohnsteuer anzumelden ist",
		"Ob die Kleinunternehmerregelung des § 19 UStG in Anspruch genommen wird",
		"Empfangsvollmacht und steuerlicher Berater, soweit vorhanden",
		"SEPA-Lastschriftmandat für das Finanzamt",
	}

	if s.documents != nil {
		if filed, err := s.documents.ForDuty(ctx, DutyKeyFragebogen); err == nil && len(filed) > 0 {
			latest := filed[len(filed)-1]
			sheet.DocumentID = latest.ID
			sheet.FiledOn = latest.CreatedAt.Format("2006-01-02")
		}
	}
	return sheet, nil
}

// FileFragebogen setzt das Datenblatt und legt es in der Dokumentenablage ab.
func (s *FoundationService) FileFragebogen(ctx context.Context) (*domain.Document, error) {
	sheet, err := s.Fragebogen(ctx)
	if err != nil {
		return nil, err
	}
	if s.documents == nil {
		return nil, fmt.Errorf("die Dokumentenablage ist nicht verfügbar")
	}
	if s.renderer == nil {
		return nil, fmt.Errorf("der Satz ist nicht verfügbar: das Datenblatt kann nicht gesetzt werden")
	}
	f, err := s.foundationRepo.Get(ctx)
	if err != nil || f == nil {
		return nil, fmt.Errorf("für diesen Mandanten ist keine Gründung erfasst")
	}

	today := todayLocal()
	title := "Datenblatt zum Fragebogen zur steuerlichen Erfassung"
	pdf, err := s.renderer.RenderDocumentPDF(ctx,
		procdoc.Typst(fragebogenMarkdown(sheet), title, parseDayOrNow(today)),
		"fragebogen-datenblatt-"+today)
	if err != nil {
		return nil, fmt.Errorf("das Datenblatt konnte nicht gesetzt werden: %w", err)
	}

	doc, err := s.documents.Attach(ctx, DocumentRequest{
		Kind:         domain.DocSteuerlicheErfassung,
		Title:        title,
		DocumentDate: today,
		DutyKey:      DutyKeyFragebogen,
		FileName:     "Fragebogen_Datenblatt_" + today + ".pdf",
		Content:      pdf,
		Note: "Zusammenstellung der Angaben aus Buchfink. Übermittelt wird der Fragebogen " +
			"über Mein ELSTER.",
	})
	if err != nil {
		return nil, err
	}
	s.audit(ctx, domain.AuditActionCreate, doc.ID,
		"Datenblatt zum Fragebogen zur steuerlichen Erfassung erzeugt und abgelegt")
	return doc, nil
}

// fragebogenMarkdown ist der Text des Datenblatts.
func fragebogenMarkdown(sheet *FragebogenSheet) string {
	var b strings.Builder
	b.WriteString("# Fragebogen zur steuerlichen Erfassung\n\n")
	b.WriteString("## Datenblatt\n\n")
	b.WriteString(
		"Die folgenden Angaben stammen aus Buchfink. Der Fragebogen selbst wird über Mein " +
			"ELSTER übermittelt (§ 138 Abs. 1b AO); dieses Blatt ist die Vorlage zum Übertragen " +
			"und keine Übermittlung.\n\n")

	section := ""
	for _, row := range sheet.Rows {
		if row.Section != section {
			if section != "" {
				b.WriteString("\n")
			}
			section = row.Section
			fmt.Fprintf(&b, "## %s\n\n", section)
			b.WriteString("| Angabe | Inhalt |\n| --- | --- |\n")
		}
		value := row.Value
		if row.Missing {
			value = "— nicht erfasst —"
		}
		fmt.Fprintf(&b, "| %s | %s |\n", row.Label, value)
	}
	b.WriteString("\n")

	if len(sheet.Open) > 0 {
		b.WriteString("## Was der Fragebogen außerdem verlangt\n\n")
		b.WriteString(
			"Diese Angaben stehen in keinem Konto und sind im Fragebogen selbst zu machen:\n\n")
		for _, open := range sheet.Open {
			fmt.Fprintf(&b, "- %s\n", open)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Hinweis\n\n")
	b.WriteString(
		"Die Aufnahme der Tätigkeit ist dem Finanzamt innerhalb eines Monats anzuzeigen " +
			"(§ 138 Abs. 1b und Abs. 4 AO). Bei einer Kapitalgesellschaft beginnt sie mit der " +
			"Beurkundung des Gesellschaftsvertrags und nicht erst mit der Eintragung: die " +
			"Vorgesellschaft ist bereits Körperschaftsteuersubjekt. Aus dem Fragebogen folgt die " +
			"Steuernummer, ohne die keine Rechnung mit Steuerausweis möglich ist.\n")
	return b.String()
}
