package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
)

// Die Klärungsliste der Eingangsbelege (RECH-02 K5, RECH-07 K2).
//
// Sie führt zusammen, was bisher an zwei Orten stand: die Regelverstöße, die
// die E-Rechnungsprüfung beim Einlesen am Beleg hinterlassen hat, und die
// Pflichtangaben des § 14 Abs. 4 UStG, die der Buchungsweg gegen die Stammdaten
// hält. Wer eine Rechnung beanstanden will, braucht beides in einer Liste —
// und je Befund die Regel, die Norm und die Folge für den Vorsteuerabzug.
//
// Die Klasse eines Befunds folgt aus seiner Herkunft und wird nicht geraten:
// ein unlesbarer Datensatz ist ein Formatfehler, eine verletzte BR-Regel ein
// Geschäftsregelfehler, eine fehlende Pflichtangabe ein Inhaltsfehler. Der
// Unterschied ist keine Feinheit — nur der dritte kostet den Vorsteuerabzug.

// Die Sätze über die Folge für den Vorsteuerabzug.
//
// Sie stehen als Konstanten, weil sie an jedem Befund ihrer Klasse gleich
// lauten müssen: dieselbe Klasse mit zwei verschiedenen Folgen wäre eine
// Auskunft, die sich selbst widerspricht.
const (
	effectFormat = "Der Datensatz ist nicht lesbar. Damit ist die Rechnung keine E-Rechnung im Sinne " +
		"des § 14 Abs. 1 Satz 3 UStG, sondern eine sonstige Rechnung; für den Vorsteuerabzug kommt es " +
		"dann auf die Pflichtangaben im lesbaren Teil an. Lass den Lieferanten den Datensatz neu senden."
	effectBusinessRule = "Der Datensatz verletzt eine Regel der EN 16931 bzw. der deutschen " +
		"Ausprägung. Der Vorsteuerabzug hängt nicht unmittelbar daran, solange die Pflichtangaben " +
		"der §§ 14, 14a UStG vorhanden sind — die Rechnung ist aber nicht normgerecht, und Buchfink " +
		"kann aus ihr nicht bestätigen, dass sie es ist."
	effectBusinessRuleWarning = "Die Norm stuft diese Regel selbst als Hinweis ein. Für den " +
		"Vorsteuerabzug folgt daraus nichts; die Angabe fehlt oder ist unüblich."
	effectContent = "Es fehlt eine Pflichtangabe der §§ 14, 14a UStG. Ohne sie ist der Vorsteuerabzug " +
		"nicht zulässig (§ 15 Abs. 1 Satz 1 Nr. 1 UStG). Ergänze die Angabe in den Stammdaten oder " +
		"lass die Rechnung berichtigen — die Berichtigung wirkt auf den Zeitpunkt der Ausstellung zurück."
)

// ReceiptCheckContext sind die Daten neben dem Beleg, gegen die die
// Pflichtangaben des § 14 Abs. 4 UStG geprüft werden.
//
// Alle drei Felder dürfen fehlen, und jedes Fehlen hat eine eigene Bedeutung:
// ohne Kontakt bleiben die Prüfungen gegen die Stammdaten des Ausstellers aus
// (und das steht als eigener Befund in der Liste), ohne eigene
// Unternehmensdaten die Prüfung des Leistungsempfängers, ohne strukturierten
// Datensatz die Prüfungen, die nur ein Datensatz beantworten kann —
// Leistungsdatum und Entgeltaufschlüsselung.
type ReceiptCheckContext struct {
	// Contact ist der Aussteller aus den Stammdaten, soweit er zu finden war.
	Contact *domain.Contact
	// Company sind die eigenen Unternehmensdaten (der Leistungsempfänger).
	Company *domain.CompanySettings
	// Invoice ist der strukturierte Rechnungsdatensatz, soweit der Beleg einen
	// hat und er lesbar ist.
	Invoice *einvoice.Invoice
}

// SetContactSource hängt die Stammdaten der Geschäftspartner an.
//
// Freiwillig wie die übrigen Anschlüsse: ohne sie arbeitet die Klärungsliste
// weiter und sagt, dass sie den Aussteller nicht gegen die Stammdaten halten
// konnte, statt sein Fehlen zu verschweigen.
func (s *ReceiptService) SetContactSource(repo domain.ContactRepository) { s.contacts = repo }

// Findings baut die Beanstandungsliste eines Belegs.
//
// Sie lädt dazu, was die Pflichtangaben prüfbar macht: den Aussteller aus den
// Stammdaten, die eigenen Unternehmensdaten und den strukturierten Datensatz
// des Belegs. Was nicht zu laden ist, fehlt nicht stillschweigend — es wird
// zum Befund.
func (s *ReceiptService) Findings(ctx context.Context, receiptID uint) (*domain.ReceiptFindings, error) {
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	check := ReceiptCheckContext{
		Invoice: s.structuredRecord(ctx, receipt),
	}
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil {
			check.Company = cfg
		}
	}
	check.Contact = s.issuerContact(ctx, receipt, check.Invoice)
	return ClassifyReceiptFindings(receipt, check), nil
}

// structuredRecord liest den strukturierten Teil eines Belegs.
//
// Gelesen und nicht aus den Kopfdaten geraten: die Kopfdaten sind eine
// Abschrift, der Datensatz ist die Rechnung. Ein unlesbarer Datensatz ergibt
// nil — das ist ein Formatfehler, und den hat die Prüfung beim Einlesen bereits
// festgehalten.
func (s *ReceiptService) structuredRecord(ctx context.Context, receipt *domain.Receipt) *einvoice.Invoice {
	file, ok := receipt.FileByRole(domain.ReceiptRoleStructured)
	if !ok || s.store == nil {
		return nil
	}
	content, err := s.Content(ctx, receipt.ID, file.ID)
	if err != nil || content == nil {
		return nil
	}
	inv, err := einvoice.ParseAny(content.Data)
	if err != nil {
		return nil
	}
	return inv
}

// issuerContact sucht den Aussteller in den Stammdaten.
//
// Über die USt-IdNr. des Datensatzes und sonst über den Namen: die USt-IdNr.
// ist eindeutig, der Name ist es nicht — aber bei einem Papierscan ist er
// alles, was der Beleg über den Aussteller weiß. Kein Treffer heißt kein
// Kontakt und kein geratener: eine Prüfung gegen die falschen Stammdaten wäre
// schlechter als keine.
func (s *ReceiptService) issuerContact(
	ctx context.Context, receipt *domain.Receipt, inv *einvoice.Invoice,
) *domain.Contact {
	if s.contacts == nil {
		return nil
	}
	all, err := s.contacts.FindAll(ctx)
	if err != nil {
		return nil
	}
	vatID := ""
	if inv != nil {
		vatID = normalizedVatID(inv.Seller.VATIdentifier)
	}
	name := normalizedName(receipt.IssuerName)
	if name == "" && inv != nil {
		name = normalizedName(inv.Seller.Name)
	}
	for i := range all {
		if vatID != "" && normalizedVatID(all[i].VatID) == vatID {
			return &all[i]
		}
	}
	for i := range all {
		if name != "" && normalizedName(all[i].Name) == name {
			return &all[i]
		}
	}
	return nil
}

func normalizedVatID(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
}

func normalizedName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// ClassifyReceiptFindings ordnet die Befunde eines Belegs ihren Fehlerklassen zu.
//
// Eine reine Funktion über den Beleg und die mitgegebenen Stammdaten: sie liest
// die Regelverstöße, die beim Einlesen festgehalten wurden, und prüft die
// Pflichtangaben des § 14 Abs. 4 UStG. Die Regelverstöße rechnet sie nicht nach
// — die Liste soll zeigen, was befunden wurde, und nicht ein zweites
// Prüfergebnis daneben stellen, das vom ersten abweichen kann.
func ClassifyReceiptFindings(receipt *domain.Receipt, check ReceiptCheckContext) *domain.ReceiptFindings {
	out := &domain.ReceiptFindings{
		ReceiptID: receipt.ID, ReceiptNumber: receipt.ReceiptNumber,
		Checked: receipt.ValidatedAt != "",
	}
	byClass := map[domain.ValidationFindingClass][]domain.ValidationFinding{}
	add := func(f domain.ValidationFinding) {
		byClass[f.Class] = append(byClass[f.Class], f)
		out.Total++
		if f.Blocking {
			out.Blocking++
		}
	}

	if raw := strings.TrimSpace(receipt.ValidationFindings); raw != "" {
		var parsed []einvoice.Finding
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			// Ein unlesbarer Befundvorrat ist selbst ein Befund. Ihn zu
			// verschweigen hieße, eine leere Liste als „nichts gefunden" zu
			// zeigen, obwohl niemand nachgesehen hat.
			add(domain.ValidationFinding{
				Class: domain.ValidationClassFormat, Rule: "findings_unreadable",
				Severity: string(einvoice.SeverityFatal),
				Message: "Das gespeicherte Prüfergebnis dieses Belegs ist nicht lesbar. " +
					"Lies den Beleg erneut ein.",
				Norm: "GoBD Rz. 100 ff.", InputTaxEffect: effectFormat, Blocking: false,
			})
		}
		for _, f := range parsed {
			add(classifyRuleFinding(f))
		}
	}

	// Die Pflichtangaben, die Buchfink am Beleg selbst sieht. Sie stehen auch
	// dann in der Liste, wenn der Beleg keinen strukturierten Teil hat — gerade
	// dann: eine eingescannte Papierrechnung wird von keiner Regel geprüft, und
	// die Pflichtangaben gelten für sie genauso.
	for _, f := range contentFindingsOf(receipt, check) {
		add(f)
	}

	for _, class := range domain.AllValidationFindingClasses() {
		findings := byClass[class]
		if len(findings) == 0 {
			continue
		}
		out.Groups = append(out.Groups, domain.ValidationFindingGroup{
			Class: class, Label: class.Label(), Findings: findings,
		})
	}
	out.EnsureLists()
	return out
}

// classifyRuleFinding ordnet einen Befund der E-Rechnungsprüfung ein.
//
// Formatfehler sind die, an denen das Lesen selbst gescheitert ist; sie haben
// keine BR-Kennung, weil eine Geschäftsregel einen lesbaren Datensatz
// voraussetzt. Alles mit einer BR-Kennung ist ein Geschäftsregelfehler.
func classifyRuleFinding(f einvoice.Finding) domain.ValidationFinding {
	out := domain.ValidationFinding{
		Rule: f.Rule, Severity: string(f.Severity), Where: f.Where, Message: f.Message,
	}
	switch {
	case strings.HasPrefix(f.Rule, "BR-DE-"):
		out.Class = domain.ValidationClassBusinessRule
		out.Norm = "XRechnung (CIUS der EN 16931), Regel " + f.Rule
	case strings.HasPrefix(f.Rule, "BR-"):
		out.Class = domain.ValidationClassBusinessRule
		out.Norm = "EN 16931, Regel " + f.Rule
	default:
		out.Class = domain.ValidationClassFormat
		out.Norm = "§ 14 Abs. 1 Satz 6 UStG i. V. m. EN 16931 (Syntax und Schema)"
	}
	switch out.Class {
	case domain.ValidationClassFormat:
		out.InputTaxEffect = effectFormat
	default:
		out.InputTaxEffect = effectBusinessRule
		if f.Severity != einvoice.SeverityFatal {
			out.InputTaxEffect = effectBusinessRuleWarning
		}
	}
	// Blockierend ist, was die Rechnungsprüfung als Fehler zählt: derselbe
	// Maßstab, den der Buchungsweg anlegt, wenn er die Vorsteuer anhält.
	out.Blocking = f.Severity == einvoice.SeverityFatal
	return out
}

// contentFindingsOf prüft die Pflichtangaben des § 14 Abs. 4 UStG.
//
// Geprüft wird gegen dreierlei: die Kopfdaten des Belegs, die Stammdaten
// (Aussteller und eigenes Unternehmen) und den strukturierten Datensatz, soweit
// der Beleg einen hat. Der Papierscan wird von keiner BR-Regel geprüft — für
// ihn ist das hier die einzige Prüfung, die es gibt.
func contentFindingsOf(receipt *domain.Receipt, check ReceiptCheckContext) []domain.ValidationFinding {
	out := make([]domain.ValidationFinding, 0, 4)
	content := func(rule, message, norm string) domain.ValidationFinding {
		return domain.ValidationFinding{
			Class: domain.ValidationClassContent, Rule: rule,
			Severity: string(einvoice.SeverityFatal), Message: message, Norm: norm,
			InputTaxEffect: effectContent, Blocking: true,
		}
	}
	if receipt.Direction != domain.DirectionIncoming || !receipt.Kind.RequiresBooking() {
		return out
	}
	if receipt.DocumentDate == "" {
		out = append(out, content("content_document_date",
			"Das Ausstellungsdatum der Rechnung fehlt.", "§ 14 Abs. 4 Nr. 3 UStG"))
	}
	if strings.TrimSpace(receipt.IssuerName) == "" && receipt.Kind == domain.ReceiptKindInvoice {
		out = append(out, content("content_issuer_name",
			"Der vollständige Name des leistenden Unternehmers fehlt.", "§ 14 Abs. 4 Nr. 1 UStG"))
	}
	if receipt.GrossAmount != 0 && receipt.TaxAmount == 0 && receipt.Kind == domain.ReceiptKindInvoice {
		// Kein blockierender Befund: eine Rechnung ohne Steuerausweis ist der
		// Regelfall bei Kleinunternehmern, steuerfreien Umsätzen und § 13b.
		// Gemeldet wird sie trotzdem, weil aus ihr keine Vorsteuer folgt und
		// genau das die häufigste stille Falschbuchung ist.
		out = append(out, domain.ValidationFinding{
			Class: domain.ValidationClassContent, Rule: "content_tax_amount",
			Severity: string(einvoice.SeverityWarning),
			Message: fmt.Sprintf(
				"Für Beleg %s ist kein Steuerbetrag erfasst. Aus einer Rechnung ohne gesondert "+
					"ausgewiesene Steuer ist kein Vorsteuerabzug möglich.", receipt.ReceiptNumber),
			Norm: "§ 14 Abs. 4 Nr. 8 UStG, § 15 Abs. 1 Satz 1 Nr. 1 UStG",
			InputTaxEffect: "Ohne gesondert ausgewiesene Steuer gibt es keinen Vorsteuerabzug. " +
				"Ist der Umsatz steuerfrei oder schuldest du die Steuer nach § 13b UStG, ist das " +
				"richtig so — dann ist der Steuerfall der Buchung entsprechend zu wählen.",
			Blocking: false,
		})
	}
	if _, hasStructured := receipt.FileByRole(domain.ReceiptRoleStructured); !hasStructured &&
		receipt.Kind == domain.ReceiptKindInvoice {
		out = append(out, domain.ValidationFinding{
			Class: domain.ValidationClassFormat, Rule: "format_no_structured_part",
			Severity: string(einvoice.SeverityInfo),
			Message: "Der Beleg hat keinen strukturierten Rechnungsdatensatz; er ist eine " +
				"sonstige Rechnung.",
			Norm: "§ 14 Abs. 1 Satz 4, § 27 Abs. 38 UStG",
			InputTaxEffect: "Für den Vorsteuerabzug folgt daraus nichts. Die Übergangsregel des " +
				"§ 27 Abs. 38 UStG lässt die sonstige Rechnung noch zu; ab 2028 ist sie es nicht mehr.",
			Blocking: false,
		})
	}
	out = append(out, masterDataFindings(check)...)
	out = append(out, recordContentFindings(check.Invoice)...)
	return out
}

// masterDataFindings hält Aussteller und Leistungsempfänger gegen die
// Stammdaten (§ 14 Abs. 4 Nr. 1 und 2 UStG).
func masterDataFindings(check ReceiptCheckContext) []domain.ValidationFinding {
	out := make([]domain.ValidationFinding, 0, 4)
	if check.Contact == nil {
		// Ein fehlender Kontakt ist keine bestandene Prüfung. Ohne ihn sind
		// Anschrift und Steuernummer des Ausstellers nicht zu prüfen, und wer
		// die Liste liest, muss den Unterschied zwischen „geprüft und in
		// Ordnung" und „nicht geprüft" sehen.
		out = append(out, domain.ValidationFinding{
			Class: domain.ValidationClassContent, Rule: "content_issuer_unknown",
			Severity: string(einvoice.SeverityWarning),
			Message: "Zu diesem Beleg ist kein Kontakt hinterlegt. Anschrift sowie " +
				"Steuernummer oder USt-IdNr. des Ausstellers sind damit nicht gegen die " +
				"Stammdaten geprüft.",
			Norm: "§ 14 Abs. 4 Nr. 1 und 2 UStG",
			InputTaxEffect: "Ob die Rechnung die Angaben zum Aussteller hat, ist offen. " +
				"Lege den Aussteller als Kontakt an, dann prüft Buchfink die Angaben mit.",
			Blocking: false,
		})
	}
	defects := issuerMasterDataDefects(check.Contact)
	defects = append(defects, recipientMasterDataDefects(check.Company)...)
	for _, d := range defects {
		out = append(out, domain.ValidationFinding{
			Class: domain.ValidationClassContent, Rule: d.Code,
			Severity: string(einvoice.SeverityFatal),
			Message:  d.Title + ": Es fehlt " + d.Detail,
			Norm:     d.Norm, InputTaxEffect: effectContent, Blocking: true,
		})
	}
	return out
}

// recordContentFindings prüft die Angaben, die nur der strukturierte Datensatz
// beantwortet: Leistungsdatum und Entgeltaufschlüsselung.
//
// Nur mit Datensatz: was auf einem Papierscan steht, liest Buchfink nicht, und
// eine Beanstandung „das Leistungsdatum fehlt" gegen ein Dokument, das niemand
// gelesen hat, wäre eine Behauptung. Der Buchungsweg fragt das Leistungsdatum
// ohnehin ab (domain.JournalEntry.ServiceDateFrom, Pflichtfeld).
func recordContentFindings(inv *einvoice.Invoice) []domain.ValidationFinding {
	out := make([]domain.ValidationFinding, 0, 2)
	if inv == nil {
		return out
	}
	// Das Leistungsdatum (§ 14 Abs. 4 Nr. 6 UStG): das tatsächliche
	// Lieferdatum (BT-72), ersatzweise der Rechnungszeitraum (BG-14) oder der
	// Steuerzeitpunkt (BT-7). Fehlt alles drei, sagt die Rechnung nicht, wann
	// geleistet wurde.
	hasPeriod := inv.Period != nil && (inv.Period.Start.Present() || inv.Period.End.Present())
	hasDelivery := inv.Delivery != nil && inv.Delivery.Date.Present()
	if !hasDelivery && !hasPeriod && !inv.TaxPointDate.Present() {
		out = append(out, domain.ValidationFinding{
			Class: domain.ValidationClassContent, Rule: "content_service_date",
			Severity: string(einvoice.SeverityFatal),
			Message: "Der Rechnungsdatensatz nennt weder ein Lieferdatum noch einen " +
				"Leistungszeitraum. Der Zeitpunkt der Leistung ist Pflichtangabe.",
			Norm: "§ 14 Abs. 4 Nr. 6 UStG", InputTaxEffect: effectContent, Blocking: true,
		})
	}
	// Die Entgeltaufschlüsselung (§ 14 Abs. 4 Nr. 7 und 8 UStG): das Entgelt,
	// nach Steuersätzen aufgeschlüsselt, und die darauf entfallende Steuer.
	if len(inv.VATBreakdown) == 0 {
		out = append(out, domain.ValidationFinding{
			Class: domain.ValidationClassContent, Rule: "content_tax_breakdown",
			Severity: string(einvoice.SeverityFatal),
			Message: "Der Rechnungsdatensatz schlüsselt das Entgelt nicht nach Steuersätzen " +
				"auf; der Steuerbetrag ist damit nicht zuzuordnen.",
			Norm: "§ 14 Abs. 4 Nr. 7 und 8 UStG", InputTaxEffect: effectContent, Blocking: true,
		})
	}
	if net, netOK := centsOf(inv.Totals.TaxBasisTotal); netOK {
		tax, taxOK := centsOf(inv.Totals.TaxTotal)
		gross, grossOK := centsOf(inv.Totals.GrandTotal)
		if taxOK && grossOK && net+tax != gross {
			out = append(out, domain.ValidationFinding{
				Class: domain.ValidationClassContent, Rule: "content_amount_breakdown",
				Severity: string(einvoice.SeverityFatal),
				Message: fmt.Sprintf(
					"Entgelt %s und Steuer %s ergeben nicht den Rechnungsbetrag %s.",
					inv.Totals.TaxBasisTotal.String(), inv.Totals.TaxTotal.String(),
					inv.Totals.GrandTotal.String()),
				Norm: "§ 14 Abs. 4 Nr. 7 und 8 UStG", InputTaxEffect: effectContent, Blocking: true,
			})
		}
	}
	return out
}

// centsOf liest einen Betrag des Datensatzes in Cent. Ein fehlender oder
// unlesbarer Betrag ergibt false — gerechnet wird nur mit dem, was dasteht.
func centsOf(a einvoice.Amount) (domain.Cents, bool) {
	if !a.Present() {
		return 0, false
	}
	value, err := a.Cents()
	if err != nil {
		return 0, false
	}
	return domain.Cents(value), true
}
