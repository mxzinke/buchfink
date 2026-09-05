package invoice

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
)

// vatCategoryCode maps a Steuerfall to the EN 16931 / UNTDID 5305 category code
// a ZUGFeRD invoice must carry. Sending "S" for an exempt intra-community supply
// makes the document formally wrong even though the amounts add up.
func vatCategoryCode(t domain.TaxTreatment) string {
	switch t {
	case domain.TaxTreatmentIntraCommunitySupply:
		return "K" // Innergemeinschaftliche Lieferung
	case domain.TaxTreatmentExport:
		return "G" // Ausfuhr, steuerfrei
	case domain.TaxTreatmentReverseChargeSupply:
		return "AE" // Steuerschuldnerschaft des Leistungsempfängers
	case domain.TaxTreatmentExempt:
		return "E" // Steuerbefreit
	case domain.TaxTreatmentNotTaxable:
		return "O" // Nicht steuerbar
	case domain.TaxTreatmentZeroRated:
		// Nullsteuersatz nach § 12 Abs. 3 UStG: steuerpflichtig zum Satz null,
		// nicht steuerfrei. "S" mit 0,00 % wäre nach BR-S-05 formal falsch, und
		// die Rechnung ließe sich gar nicht erst ausstellen.
		return "Z"
	default:
		return "S" // Regelbesteuerung
	}
}

// exemptionReason returns the legal reference EN 16931 requires whenever the
// category code is not "S".
func exemptionReason(t domain.TaxTreatment) string {
	switch t {
	case domain.TaxTreatmentIntraCommunitySupply:
		return "Steuerfreie innergemeinschaftliche Lieferung nach § 4 Nr. 1 Buchst. b i. V. m. § 6a UStG"
	case domain.TaxTreatmentExport:
		return "Steuerfreie Ausfuhrlieferung nach § 4 Nr. 1 Buchst. a i. V. m. § 6 UStG"
	case domain.TaxTreatmentReverseChargeSupply:
		return "Steuerschuldnerschaft des Leistungsempfängers nach § 13b UStG"
	case domain.TaxTreatmentExempt:
		return "Steuerfreier Umsatz nach § 4 UStG"
	case domain.TaxTreatmentNotTaxable:
		return "Nicht steuerbarer Umsatz"
	default:
		return ""
	}
}

// typstDate renders an ISO date as a Typst datetime call, falling back to the
// current day only when the invoice carries no date at all — which the invoice
// validation prevents.
func typstDate(iso string) string {
	var y, m, d int
	if n, err := fmt.Sscanf(iso, "%4d-%2d-%2d", &y, &m, &d); err != nil || n != 3 {
		return "auto"
	}
	return fmt.Sprintf("datetime(year: %d, month: %d, day: %d)", y, m, d)
}

func quantityString(milli int64) string {
	neg := ""
	if milli < 0 {
		neg = "-"
		milli = -milli
	}
	return fmt.Sprintf("%s%d.%03d", neg, milli/1000, milli%1000)
}

func ratePercent(r domain.TaxRate) string {
	return fmt.Sprintf("%d.%02d", int(r)/100, int(r)%100)
}

// GenerateZUGFeRDXML creates a Factur-X / ZUGFeRD 2.2 (EN 16931) XML invoice.
//
// Sie bleibt als Name bestehen, ist aber nur noch der Regelfall von
// RenderInvoiceXML: derselbe Datensatz über das semantische Modell, mit dem
// ZUGFeRD-Profil als Kennung. Die frühere XML-Vorlage aus Formatstrings ist
// entfallen — siehe cii.go.
func GenerateZUGFeRDXML(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact) (string, error) {
	xml, _, err := RenderInvoiceXML(inv, seller, buyer, domain.EInvoiceProfileZUGFeRD)
	if err != nil {
		return "", err
	}
	return xml, nil
}

func errorMessages(result einvoice.Result) []string {
	var out []string
	for _, f := range result.Findings {
		if f.Severity == einvoice.SeverityFatal {
			out = append(out, f.Message)
		}
	}
	return out
}

func countryOrDE(code string) string {
	if code == "" {
		return "DE"
	}
	return code
}

// GenerateTypstTemplate renders the invoice as Typst markup.
//
// Das PDF ist das, was der Empfänger liest. Alles, was § 14 Abs. 4 UStG als
// Pflichtangabe nennt, muss deshalb hier stehen und nicht nur im XML: die
// vollständige Anschrift beider Seiten, Steuernummer oder USt-IdNr. des
// Ausstellers, die USt-IdNr. des Empfängers, wo sie hingehört, der
// Leistungszeitpunkt, die vereinbarte Entgeltminderung — und bei einer
// Korrektur der Bezug auf die Rechnung, die sie berichtigt.
func GenerateTypstTemplate(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact) string {
	return typstTemplate(inv, seller, buyer, true)
}

// GeneratePlainTypstTemplate renders the same document without the attached
// record.
//
// Es ist die sonstige Rechnung des § 14 Abs. 1 Satz 4 UStG: ein PDF ohne
// strukturierten Teil, zulässig nur innerhalb der Übergangsfrist des
// § 27 Abs. 38 UStG. Sie entsteht aus derselben Vorlage — ein zweites Layout
// wäre eine zweite Stelle, an der eine Pflichtangabe fehlen könnte.
func GeneratePlainTypstTemplate(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact) string {
	return typstTemplate(inv, seller, buyer, false)
}

func typstTemplate(inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact, attach bool) string {
	var rows strings.Builder
	for i := range inv.Items {
		item := &inv.Items[i]
		rows.WriteString(fmt.Sprintf("  [%d], [%s], [%s %s], [%s €], [%s €],\n",
			item.Position,
			typstEscape(item.Description),
			strings.TrimSuffix(strings.TrimRight(quantityString(item.QuantityMilli), "0"), "."),
			typstEscape(domain.UnitLabel(item.Unit)),
			item.UnitPrice.String(),
			item.TotalNet().String(),
		))
	}

	var totals strings.Builder
	totals.WriteString("      [Nettobetrag:], [" + inv.NetAmount.String() + " €],\n")
	if inv.TaxTreatment == domain.TaxTreatmentDomestic {
		for _, g := range inv.TaxGroups() {
			if g.Tax == 0 {
				continue
			}
			totals.WriteString(fmt.Sprintf("      [zzgl. %s USt:], [%s €],\n", g.Rate.Label(), g.Tax.String()))
		}
	}
	totals.WriteString(`      [*Gesamtbetrag:*], [*#text(size: 12pt, fill: rgb("#d97706"))[` + inv.GrossAmount.String() + ` €]*],` + "\n")
	// Die verrechneten Anzahlungen stehen unter dem Gesamtbetrag und nicht
	// darin: § 14 Abs. 5 Satz 2 UStG verlangt, dass die Schlussrechnung die
	// vereinnahmten Teilentgelte und die darauf entfallende Steuer absetzt —
	// sichtbar, nicht schon im Gesamtbetrag verrechnet.
	if inv.PrepaidAmount != 0 {
		totals.WriteString(fmt.Sprintf("      [abzüglich Anzahlungen:], [%s €],\n", (-inv.PrepaidAmount).String()))
		totals.WriteString(fmt.Sprintf("      [*Zahlbetrag:*], [*%s €*],\n", inv.OpenAmount().String()))
	}

	// Every non-standard Steuerfall needs its legal reference printed on the
	// invoice; § 14 Abs. 4 Nr. 8 UStG requires the reason for an exemption.
	var notes []string
	if reason := exemptionReason(inv.TaxTreatment); reason != "" {
		notes = append(notes, reason)
	}
	if ref := correctionNote(inv); ref != "" {
		notes = append(notes, ref)
	}
	if terms := inv.Terms.Note(inv.Date); terms != "" {
		notes = append(notes, terms)
	}
	if inv.SmallAmount {
		note := "Kleinbetragsrechnung nach § 33 UStDV"
		// Ohne Empfänger geht sie als PDF hinaus. Der Hinweis steht auf dem
		// Dokument, weil der Empfänger sonst eine E-Rechnung erwartet, die es zu
		// diesem Vorgang nicht gibt und nicht geben muss.
		if inv.ContactID == 0 {
			note += " — von der Pflicht zur E-Rechnung ausgenommen"
		}
		notes = append(notes, note)
	}
	if inv.ResolvedKind() == domain.InvoiceKindAdvance && inv.PaymentReceivedAt != "" {
		notes = append(notes, "Zeitpunkt der Vereinnahmung: "+domain.GermanDate(inv.PaymentReceivedAt))
	}
	var note string
	for _, n := range notes {
		note += fmt.Sprintf("\n#v(0.3cm)\n#text(size: 8.5pt, fill: rgb(\"#78716c\"), style: \"italic\")[%s]", typstEscape(n))
	}

	// PDF/A demands a document date, and the embedded file needs both a mime type
	// and a description — Typst refuses to emit a-3b without them, which is
	// exactly the check ZUGFeRD needs. The relationship must be "alternative":
	// the PDF and the XML are two renderings of one invoice, and for the BASIC
	// and EN-16931 profiles anything else is not legally valid in Germany.
	docDate := typstDate(inv.Date)
	heading := strings.ToUpper(inv.ResolvedKind().Label())

	embed := ""
	if attach {
		embed = `#pdf.embed(
  "factur-x.xml",
  bytes(sys.inputs.zugferd_xml),
  relationship: "alternative",
  mime-type: "text/xml",
  description: "Factur-X / ZUGFeRD invoice data (EN 16931)",
)
`
	}

	return fmt.Sprintf(`#set document(title: "%s %s", author: %q, date: %s)
%s#set page(paper: "a4", margin: (x: 2cm, y: 2.5cm))
#set text(font: "Manrope", size: 10pt, fill: rgb("#1c1917"))

#grid(
  columns: (1fr, 1fr),
  [
    #text(size: 8pt, fill: rgb("#78716c"))[%s · %s · %s]\
    #v(0.5cm)
    *#text(size: 11pt)[%s]*\
    %s
  ],
  [
    #align(right)[
      #text(size: 16pt, weight: "bold", fill: rgb("#d97706"))[%s]\
      #v(0.2cm)
      #text(size: 10pt, weight: "bold")[Nr. %s]\
      Rechnungsdatum: %s\
      %s
      Fällig bis: %s
    ]
  ]
)

#v(1.5cm)

#table(
  columns: (auto, 1fr, auto, auto, auto),
  align: (center, left, right, right, right),
  stroke: (x, y) => if y == 0 { (bottom: 1.5pt + rgb("#d97706")) } else { (bottom: 0.5pt + rgb("#e7e5e4")) },
  table.header([*Pos*], [*Bezeichnung*], [*Menge*], [*Einzelpreis*], [*Gesamt*]),
%s)

#v(0.5cm)
#align(right)[
  #block(width: 60%%)[
    #grid(
      columns: (1fr, auto),
      row-gutter: 0.3cm,
%s    )
  ]
]%s

#v(2cm)
#line(length: 100%%, stroke: 0.5pt + rgb("#e7e5e4"))
#text(size: 8pt, fill: rgb("#78716c"))[
  Bankverbindung: %s · IBAN: %s · BIC: %s\
  Steuernummer: %s · USt-IdNr.: %s
]
`,
		typstEscape(inv.ResolvedKind().Label()), typstEscape(inv.InvoiceNumber),
		// %q inside a Typst string literal: the markup escaping of typstEscape
		// would land as visible backslashes in the PDF metadata.
		seller.CompanyName,
		docDate,
		embed,
		typstEscape(seller.CompanyName), typstEscape(seller.Street), typstEscape(seller.ZipCity),
		typstEscape(buyerName(inv, buyer)), buyerAddressBlock(inv, buyer),
		typstEscape(heading),
		typstEscape(inv.InvoiceNumber), domain.GermanDate(inv.Date),
		servicePeriodLine(inv),
		domain.GermanDate(inv.DueDate),
		rows.String(),
		totals.String(),
		note,
		typstEscape(seller.BankName), typstEscape(seller.IBAN), typstEscape(seller.BIC),
		typstEscape(seller.TaxNumber), typstEscape(seller.VatID),
	)
}

// buyerName is what the address block is headed with. Without a contact — the
// Barverkauf of a Kleinbetragsrechnung — it says so instead of leaving a gap.
func buyerName(inv *domain.Invoice, buyer *domain.Contact) string {
	if buyer != nil && buyer.Name != "" {
		return buyer.Name
	}
	if inv.ContactName != "" {
		return inv.ContactName
	}
	return "Barverkauf"
}

// buyerAddressBlock renders the recipient below their name: street, postal code
// and city each on their own line, and the USt-IdNr. where § 14a UStG puts it
// on the document.
func buyerAddressBlock(inv *domain.Invoice, buyer *domain.Contact) string {
	if buyer == nil {
		return ""
	}
	street, postalCode, city := buyer.PostalAddress()
	var b strings.Builder
	if street != "" {
		b.WriteString(typstEscape(street) + " \\\n    ")
	}
	if postalCode != "" || city != "" {
		b.WriteString(typstEscape(strings.TrimSpace(postalCode+" "+city)) + " \\\n    ")
	}
	if buyer.VatID != "" {
		b.WriteString("USt-IdNr.: " + typstEscape(buyer.VatID))
	}
	return strings.TrimSuffix(b.String(), " \\\n    ")
}

// servicePeriodLine states the Leistungszeitpunkt (§ 14 Abs. 4 Nr. 6 UStG). A
// Abschlagsrechnung before the money came in has none to state — then the line
// stays away rather than repeating the invoice date as if it were one.
func servicePeriodLine(inv *domain.Invoice) string {
	if inv.ResolvedKind() == domain.InvoiceKindAdvance {
		if inv.PaymentReceivedAt == "" {
			return ""
		}
		return "Vereinnahmung: " + domain.GermanDate(inv.PaymentReceivedAt) + "\\\n      "
	}
	if inv.ServiceDateFrom == inv.ServiceDateTo {
		return "Leistungsdatum: " + domain.GermanDate(inv.ServiceDateTo) + "\\\n      "
	}
	return "Leistungszeitraum: " + domain.GermanDate(inv.ServiceDateFrom) + " – " +
		domain.GermanDate(inv.ServiceDateTo) + "\\\n      "
}

// correctionNote names the document this one refers to.
func correctionNote(inv *domain.Invoice) string {
	if inv.CorrectsInvoiceNumber == "" {
		return ""
	}
	date := ""
	if inv.CorrectsInvoiceDate != "" {
		date = " vom " + domain.GermanDate(inv.CorrectsInvoiceDate)
	}
	switch inv.ResolvedKind() {
	case domain.InvoiceKindCancellation:
		return "Storno zu Rechnung " + inv.CorrectsInvoiceNumber + date
	case domain.InvoiceKindCorrection:
		return "Berichtigung der Rechnung " + inv.CorrectsInvoiceNumber + date
	default:
		return "Bezug: Rechnung " + inv.CorrectsInvoiceNumber + date
	}
}

func typstEscape(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "[", "\\[", "]", "\\]",
		"#", "\\#", "$", "\\$", "*", "\\*", "_", "\\_", "@", "\\@",
		"\n", " \\\n",
	)
	return replacer.Replace(s)
}
