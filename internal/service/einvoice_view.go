package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
)

// EInvoiceView ist die Ansicht einer E-Rechnung für die App. Sie gibt wieder,
// was die XML sagt, Beträge und Daten so, wie sie dort stehen; maßgeblich
// bleibt die XML selbst.
type EInvoiceView struct {
	Syntax         string `json:"syntax"`
	TypeCode       string `json:"typeCode"`
	TypeLabel      string `json:"typeLabel"`
	Number         string `json:"number"`
	IssueDate      string `json:"issueDate"`
	DueDate        string `json:"dueDate,omitempty"`
	Currency       string `json:"currency"`
	PeriodFrom     string `json:"periodFrom,omitempty"`
	PeriodTo       string `json:"periodTo,omitempty"`
	DeliveryDate   string `json:"deliveryDate,omitempty"`
	BuyerReference string `json:"buyerReference,omitempty"`
	OrderReference string `json:"orderReference,omitempty"`
	// PrecedingInvoices nennt die Rechnungen, auf die sich eine Gutschrift oder
	// Korrektur bezieht.
	PrecedingInvoices []string              `json:"precedingInvoices,omitempty"`
	Seller            EInvoiceViewParty     `json:"seller"`
	Buyer             EInvoiceViewParty     `json:"buyer"`
	Lines             []EInvoiceViewLine    `json:"lines"`
	VAT               []EInvoiceViewVAT     `json:"vat"`
	Totals            EInvoiceViewTotals    `json:"totals"`
	PaymentTerms      string                `json:"paymentTerms,omitempty"`
	Payment           []EInvoiceViewPayment `json:"payment,omitempty"`
	Notes             []string              `json:"notes,omitempty"`
	FileName          string                `json:"fileName"`
	Intact            bool                  `json:"intact"`
}

// EInvoiceViewParty ist Verkäufer oder Erwerber.
type EInvoiceViewParty struct {
	Name      string   `json:"name"`
	Address   []string `json:"address,omitempty"`
	VATID     string   `json:"vatId,omitempty"`
	TaxNumber string   `json:"taxNumber,omitempty"`
	Contact   string   `json:"contact,omitempty"`
}

// EInvoiceViewLine ist eine Rechnungsposition.
type EInvoiceViewLine struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Quantity    string `json:"quantity"`
	Unit        string `json:"unit"`
	NetPrice    string `json:"netPrice"`
	NetAmount   string `json:"netAmount"`
	VATRate     string `json:"vatRate"`
	VATCategory string `json:"vatCategory"`
	PeriodFrom  string `json:"periodFrom,omitempty"`
	PeriodTo    string `json:"periodTo,omitempty"`
}

// EInvoiceViewVAT ist eine Zeile der Umsatzsteueraufschlüsselung.
type EInvoiceViewVAT struct {
	Category        string `json:"category"`
	Rate            string `json:"rate"`
	TaxableAmount   string `json:"taxableAmount"`
	TaxAmount       string `json:"taxAmount"`
	ExemptionReason string `json:"exemptionReason,omitempty"`
}

// EInvoiceViewTotals sind die Dokumentsummen.
type EInvoiceViewTotals struct {
	LineTotal     string `json:"lineTotal"`
	TaxBasisTotal string `json:"taxBasisTotal"`
	TaxTotal      string `json:"taxTotal"`
	GrandTotal    string `json:"grandTotal"`
	Prepaid       string `json:"prepaid,omitempty"`
	DuePayable    string `json:"duePayable"`
}

// EInvoiceViewPayment ist eine Zahlungsanweisung.
type EInvoiceViewPayment struct {
	Means          string `json:"means"`
	IBAN           string `json:"iban,omitempty"`
	BIC            string `json:"bic,omitempty"`
	AccountName    string `json:"accountName,omitempty"`
	RemittanceInfo string `json:"remittanceInfo,omitempty"`
}

// View liest den Rechnungsdatensatz eines Belegs für die Anzeige.
func (s *EInvoiceService) View(ctx context.Context, receiptID uint) (*EInvoiceView, error) {
	receipt, err := s.receiptSvc.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	structured, ok := receipt.FileByRole(domain.ReceiptRoleStructured)
	if !ok {
		return nil, fmt.Errorf("Beleg %s hat keinen Rechnungsdatensatz", receipt.ReceiptNumber)
	}
	content, err := s.receiptSvc.Content(ctx, receiptID, structured.ID)
	if err != nil {
		return nil, err
	}
	inv, err := einvoice.ParseAny(content.Data)
	if err != nil {
		return nil, fmt.Errorf("der Rechnungsdatensatz von Beleg %s lässt sich nicht lesen: %w", receipt.ReceiptNumber, err)
	}
	view := eInvoiceViewOf(inv)
	view.FileName, view.Intact = structured.FileName, content.Intact
	return view, nil
}

func eInvoiceViewOf(inv *einvoice.Invoice) *EInvoiceView {
	v := &EInvoiceView{
		Syntax: string(inv.Syntax), TypeCode: inv.TypeCode, TypeLabel: invoiceTypeLabel(inv.TypeCode),
		Number: inv.Number, IssueDate: inv.IssueDate.String(), DueDate: inv.DueDate.String(),
		Currency: inv.Currency, BuyerReference: inv.BuyerReference, OrderReference: inv.OrderReference,
		Seller: partyView(inv.Seller), Buyer: partyView(inv.Buyer),
		Lines: make([]EInvoiceViewLine, 0, len(inv.Lines)), VAT: make([]EInvoiceViewVAT, 0, len(inv.VATBreakdown)),
		PaymentTerms: inv.PaymentTermsNote,
		Totals: EInvoiceViewTotals{
			LineTotal: inv.Totals.LineTotal.String(), TaxBasisTotal: inv.Totals.TaxBasisTotal.String(),
			TaxTotal: inv.Totals.TaxTotal.String(), GrandTotal: inv.Totals.GrandTotal.String(),
			Prepaid: inv.Totals.PrepaidAmount.String(), DuePayable: inv.Totals.DuePayableAmount.String(),
		},
	}
	if inv.Period.Present() {
		v.PeriodFrom, v.PeriodTo = inv.Period.Start.String(), inv.Period.End.String()
	}
	if inv.Delivery != nil {
		v.DeliveryDate = inv.Delivery.Date.String()
	}
	for _, p := range inv.PrecedingInvoices {
		v.PrecedingInvoices = append(v.PrecedingInvoices, p.Number)
	}
	for _, l := range inv.Lines {
		line := EInvoiceViewLine{ID: l.ID, Name: l.Item.Name, Description: l.Item.Description,
			Quantity: l.Quantity.String(), Unit: unitLabel(l.UnitCode), NetPrice: l.Price.NetPrice.String(),
			NetAmount: l.NetAmount.String(), VATRate: l.VAT.Rate.String(), VATCategory: l.VAT.CategoryCode}
		if l.Period.Present() {
			line.PeriodFrom, line.PeriodTo = l.Period.Start.String(), l.Period.End.String()
		}
		v.Lines = append(v.Lines, line)
	}
	for _, b := range inv.VATBreakdown {
		reason := b.ExemptionReason
		if reason == "" {
			reason = b.ExemptionReasonCode
		}
		v.VAT = append(v.VAT, EInvoiceViewVAT{Category: b.CategoryCode, Rate: b.Rate.String(),
			TaxableAmount: b.TaxableAmount.String(), TaxAmount: b.TaxAmount.String(), ExemptionReason: reason})
	}
	for _, m := range inv.PaymentMeans {
		means := m.TypeText
		if means == "" {
			means = paymentMeansLabel(m.TypeCode)
		}
		if len(m.CreditTransfer) == 0 {
			v.Payment = append(v.Payment, EInvoiceViewPayment{Means: means, RemittanceInfo: m.RemittanceInfo})
		}
		for _, t := range m.CreditTransfer {
			v.Payment = append(v.Payment, EInvoiceViewPayment{Means: means, IBAN: t.AccountID, BIC: t.ProviderID,
				AccountName: t.AccountName, RemittanceInfo: m.RemittanceInfo})
		}
	}
	for _, n := range inv.Notes {
		if text := strings.TrimSpace(n.Text); text != "" {
			v.Notes = append(v.Notes, text)
		}
	}
	return v
}

func partyView(p einvoice.Party) EInvoiceViewParty {
	out := EInvoiceViewParty{Name: p.Name, VATID: p.VATIdentifier, TaxNumber: p.TaxRegistration}
	if a := p.Address; a != nil {
		for _, line := range []string{a.LineOne, a.LineTwo, a.LineThree,
			strings.TrimSpace(a.PostCode + " " + a.City), a.CountryCode} {
			if line = strings.TrimSpace(line); line != "" {
				out.Address = append(out.Address, line)
			}
		}
	}
	if c := p.Contact; c != nil {
		parts := []string{}
		for _, s := range []string{c.Name, c.Phone, c.Email} {
			if s = strings.TrimSpace(s); s != "" {
				parts = append(parts, s)
			}
		}
		out.Contact = strings.Join(parts, " · ")
	}
	return out
}

// invoiceTypeLabel benennt die häufigen Rechnungstypen (UNTDID 1001).
func invoiceTypeLabel(code string) string {
	switch code {
	case "380":
		return "Rechnung"
	case "381":
		return "Gutschrift"
	case "384":
		return "Rechnungskorrektur"
	case "389":
		return "Gutschrift im Gutschriftverfahren"
	case "326":
		return "Teilrechnung"
	case "875", "876", "877":
		return "Abschlags- oder Schlussrechnung"
	}
	return "Rechnung (Typ " + code + ")"
}

// unitLabel benennt die häufigen Mengeneinheiten (UN/ECE Rec 20).
func unitLabel(code string) string {
	labels := map[string]string{"C62": "Stück", "H87": "Stück", "XPP": "Stück", "HUR": "Stunde", "DAY": "Tag",
		"WEE": "Woche", "MON": "Monat", "ANN": "Jahr", "MIN": "Minute", "KGM": "kg", "MTR": "m", "LTR": "l",
		"LS": "pauschal", "MTK": "m²"}
	if label, ok := labels[code]; ok {
		return label
	}
	return code
}

// paymentMeansLabel benennt die häufigen Zahlungsmittel (UNTDID 4461).
func paymentMeansLabel(code string) string {
	switch code {
	case "58":
		return "SEPA-Überweisung"
	case "59":
		return "SEPA-Lastschrift"
	case "30":
		return "Überweisung"
	case "48", "54", "55":
		return "Karte"
	case "10":
		return "Bar"
	}
	return "Zahlungsmittel " + code
}
