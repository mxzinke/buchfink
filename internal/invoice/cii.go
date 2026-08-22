// ACHTUNG: Dieser Prüfer ist abgelöst.
//
// Die EN-16931-Prüfung liegt in `internal/einvoice`. Sie läuft auf einem
// semantischen Modell statt auf CII-Structs, deckt alle 223 Geschäftsregeln ab
// statt 170, liest neben CII auch UBL, und XRechnung und ZUGFeRD sitzen als
// Schichten darüber.
//
// Was hier steht, hängt nur noch am Buchungspfad (`internal/service`), der
// weiterhin die CIIInvoice-Struktur verwendet. Das Umhängen ist der zweite
// Schritt und für sich zu machen — bis dahin gilt: **neue Regeln kommen ins
// Modul, nicht hierher.** Zwei Prüfer im Baum sind genau die Stelle, an der
// jemand den falschen bearbeitet.

package invoice

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// The CII namespaces of ZUGFeRD / Factur-X. Matching on them rather than on the
// local element name matters: a document can bind the prefixes to anything, and
// two of the names below appear in several namespaces.
const (
	nsRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	nsRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	nsUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
)

// EInvoiceFormat is the structured format a document was recognised as.
type EInvoiceFormat string

const (
	FormatUnknown EInvoiceFormat = ""
	// FormatCII is UN/CEFACT Cross Industry Invoice, the syntax behind both
	// ZUGFeRD/Factur-X and the CII variant of XRechnung.
	FormatCII EInvoiceFormat = "cii"
)

// Known guideline identifiers. The profile decides whether a document counts as
// an E-Rechnung at all: ZUGFeRD MINIMUM and BASIC WL do not contain a complete
// invoice and are therefore no E-Rechnung in the sense of the law
// (UStAE 14.1 Abs. 14 Satz 4).
const (
	profileMinimum = "urn:factur-x.eu:1p0:minimum"
	profileBasicWL = "urn:factur-x.eu:1p0:basicwl"
)

// CIIInvoice is the part of a Cross Industry Invoice Buchfink reads.
//
// It is deliberately a plain transcription of the document, not a booking. The
// mapping into a Buchungsvorschlag happens separately, because that step needs
// judgement — most of all about the perspective of the VAT category code.
type CIIInvoice struct {
	XMLName xml.Name `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 CrossIndustryInvoice"`

	Context struct {
		Guideline struct {
			ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GuidelineSpecifiedDocumentContextParameter"`
	} `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 ExchangedDocumentContext"`

	Document struct {
		ID        string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
		TypeCode  string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
		IssueDate ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IssueDateTime"`
	} `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 ExchangedDocument"`

	Transaction struct {
		Lines []ciiLine `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IncludedSupplyChainTradeLineItem"`

		Agreement struct {
			Seller ciiParty `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SellerTradeParty"`
			Buyer  ciiParty `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BuyerTradeParty"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeAgreement"`

		Delivery struct {
			// ShipTo carries the Bestimmungsland (BT-80), which BR-IC-12 requires
			// on an intra-community supply.
			ShipTo *ciiParty `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ShipToTradeParty"`
			Event  struct {
				Occurrence ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 OccurrenceDateTime"`
			} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ActualDeliverySupplyChainEvent"`
			// Period is a pointer for the same reason the address is: BR-CO-19
			// asks whether a stated Abrechnungszeitraum has a beginning or an
			// end, which only means something if "no period" and "an empty
			// period" are distinguishable.
			Period *ciiPeriod `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BillingSpecifiedPeriod"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeDelivery"`

		Settlement struct {
			Currency string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 InvoiceCurrencyCode"`
			// TaxCurrency (BT-6) is the currency the VAT is accounted in when it
			// differs from the invoice currency.
			TaxCurrency string        `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxCurrencyCode"`
			Taxes       []ciiTradeTax `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeTax"`
			// Nachlässe und Zuschläge auf Dokumentebene (BG-20, BG-21). Sie
			// verschieben jede Summenregel: ohne sie ist der Nettogesamtbetrag
			// die Summe der Positionen, mit ihnen nicht mehr. Wer sie nicht
			// liest, kann die Summen nur noch vermuten.
			AllowancesCharges []ciiAllowanceCharge `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeAllowanceCharge"`
			Terms             struct {
				DueDate ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DueDateDateTime"`
			} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradePaymentTerms"`
			Summation struct {
				LineTotal      string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTotalAmount"`
				AllowanceTotal string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AllowanceTotalAmount"`
				ChargeTotal    string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeTotalAmount"`
				TaxBasisTotal  string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxBasisTotalAmount"`
				// TaxTotals holds BT-110 and, where the VAT is accounted in a
				// second currency, BT-111. Both are written as the same element
				// and only the currencyID tells them apart — reading just one of
				// them picks whichever came last, which on a Danish invoice with
				// a euro VAT total is the wrong number.
				TaxTotals      []ciiAmount `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxTotalAmount"`
				GrandTotal     string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GrandTotalAmount"`
				TotalPrepaid   string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TotalPrepaidAmount"`
				RoundingAmount string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 RoundingAmount"`
				DuePayable     string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DuePayableAmount"`
			} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementHeaderMonetarySummation"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeSettlement"`
	} `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 SupplyChainTradeTransaction"`
}

type ciiParty struct {
	Name string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
	// ID, GlobalID and LegalOrganization carry the identifiers BR-CO-26 asks
	// for (BT-29, BT-30). They are also what Buchfink matches a Kontakt on.
	ID                string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	GlobalID          string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GlobalID"`
	LegalOrganization struct {
		ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLegalOrganization"`
	// Address is a pointer because BR-08 and BR-10 ask whether the address
	// element exists at all, not whether any particular field in it is filled.
	// An address consisting of nothing but a country code is valid — BR-09 is
	// the rule that requires the country, and it is a separate one.
	Address       *ciiAddress          `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PostalTradeAddress"`
	Registrations []ciiTaxRegistration `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTaxRegistration"`
}

// VatID returns the partner's USt-IdNr., which carries schemeID "VA".
func (p ciiParty) VatID() string { return p.registration("VA") }

// TaxNumber returns the partner's Steuernummer, schemeID "FC".
func (p ciiParty) TaxNumber() string { return p.registration("FC") }

func (p ciiParty) registration(scheme string) string {
	for _, r := range p.Registrations {
		if strings.EqualFold(r.ID.SchemeID, scheme) {
			return strings.TrimSpace(r.ID.Value)
		}
	}
	return ""
}

type ciiTaxRegistration struct {
	ID struct {
		Value    string `xml:",chardata"`
		SchemeID string `xml:"schemeID,attr"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
}

type ciiAddress struct {
	LineOne   string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineOne"`
	PostCode  string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PostcodeCode"`
	CityName  string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CityName"`
	CountryID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CountryID"`
}

// CountryCode returns the party's country, or the empty string if no address is
// present at all.
func (p ciiParty) CountryCode() string {
	if p.Address == nil {
		return ""
	}
	return strings.TrimSpace(p.Address.CountryID)
}

// ciiAmount is a monetary value together with the currency it is stated in.
type ciiAmount struct {
	Value      string `xml:",chardata"`
	CurrencyID string `xml:"currencyID,attr"`
}

type ciiPeriod struct {
	Start ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 StartDateTime"`
	End   ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 EndDateTime"`
}

type ciiDateTime struct {
	Value string `xml:"urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100 DateTimeString"`
}

type ciiAllowanceCharge struct {
	// ChargeIndicator false = Nachlass, true = Zuschlag. Beide stehen im
	// selben Element; allein dieses Flag trennt einen Abzug von einem Aufschlag,
	// und mit ihm kehrt sich das Vorzeichen jeder Summenregel um.
	Indicator struct {
		Value string `xml:"urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100 Indicator"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeIndicator"`
	ActualAmount string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ActualAmount"`
	BasisAmount  string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BasisAmount"`
	Reason       string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Reason"`
	ReasonCode   string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReasonCode"`
	// CategoryTax trägt Steuerkategorie und Satz des Nachlasses (BT-95/BT-96
	// bzw. BT-102/BT-103). Ein Nachlass gehört damit in dieselbe Steuergruppe
	// wie die Positionen, die er mindert.
	CategoryTax ciiTradeTax `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CategoryTradeTax"`
}

// IsCharge reports whether the entry is a Zuschlag rather than a Nachlass.
func (a ciiAllowanceCharge) IsCharge() bool {
	switch strings.ToLower(strings.TrimSpace(a.Indicator.Value)) {
	case "true", "1":
		return true
	}
	return false
}

// HasReason reports whether a reason or a reason code is given, which BR-33,
// BR-38, BR-42 and BR-44 require — one of the two, not both.
func (a ciiAllowanceCharge) HasReason() bool {
	return strings.TrimSpace(a.Reason) != "" || strings.TrimSpace(a.ReasonCode) != ""
}

type ciiTradeTax struct {
	CalculatedAmount    string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CalculatedAmount"`
	TypeCode            string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
	ExemptionReason     string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ExemptionReason"`
	ExemptionReasonCode string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ExemptionReasonCode"`
	BasisAmount         string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BasisAmount"`
	CategoryCode        string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CategoryCode"`
	RatePercent         string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 RateApplicablePercent"`
}

type ciiLine struct {
	Document struct {
		LineID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AssociatedDocumentLineDocument"`
	Product struct {
		Name string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeProduct"`
	Agreement struct {
		NetPrice struct {
			ChargeAmount string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeAmount"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 NetPriceProductTradePrice"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeAgreement"`
	Delivery struct {
		BilledQuantity struct {
			Value    string `xml:",chardata"`
			UnitCode string `xml:"unitCode,attr"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BilledQuantity"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeDelivery"`
	Settlement struct {
		Tax ciiTradeTax `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeTax"`
		// Nachlässe und Zuschläge auf Positionsebene (BG-27, BG-28).
		AllowancesCharges []ciiAllowanceCharge `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeAllowanceCharge"`
		Summation         struct {
			LineTotal string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTotalAmount"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementLineMonetarySummation"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeSettlement"`
}

// TaxTotal returns the total VAT amount in the invoice currency (BT-110).
func (c *CIIInvoice) TaxTotal() ciiAmount {
	return pickAmount(c.Transaction.Settlement.Summation.TaxTotals, c.Currency())
}

// TaxTotalInAccountingCurrency returns the total VAT in the VAT accounting
// currency (BT-111), or an empty amount if the invoice states only one.
func (c *CIIInvoice) TaxTotalInAccountingCurrency() ciiAmount {
	totals := c.Transaction.Settlement.Summation.TaxTotals
	if len(totals) < 2 {
		return ciiAmount{}
	}
	invoiceCurrency := strings.ToUpper(strings.TrimSpace(c.Currency()))
	for _, a := range totals {
		if strings.ToUpper(strings.TrimSpace(a.CurrencyID)) != invoiceCurrency {
			return a
		}
	}
	return ciiAmount{}
}

func pickAmount(amounts []ciiAmount, currency string) ciiAmount {
	if len(amounts) == 0 {
		return ciiAmount{}
	}
	want := strings.ToUpper(strings.TrimSpace(currency))
	for _, a := range amounts {
		if strings.ToUpper(strings.TrimSpace(a.CurrencyID)) == want {
			return a
		}
	}
	return amounts[0]
}

// DocumentAllowances returns the Nachlässe on document level (BG-20).
func (c *CIIInvoice) DocumentAllowances() []ciiAllowanceCharge {
	return splitAllowancesCharges(c.Transaction.Settlement.AllowancesCharges, false)
}

// DocumentCharges returns the Zuschläge on document level (BG-21).
func (c *CIIInvoice) DocumentCharges() []ciiAllowanceCharge {
	return splitAllowancesCharges(c.Transaction.Settlement.AllowancesCharges, true)
}

func splitAllowancesCharges(all []ciiAllowanceCharge, wantCharge bool) []ciiAllowanceCharge {
	var out []ciiAllowanceCharge
	for _, a := range all {
		if a.IsCharge() == wantCharge {
			out = append(out, a)
		}
	}
	return out
}

// ParseCII reads a Cross Industry Invoice.
func ParseCII(data []byte) (*CIIInvoice, error) {
	var doc CIIInvoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("der strukturierte Rechnungsdatensatz konnte nicht gelesen werden: %w", err)
	}
	if doc.Document.ID == "" && len(doc.Transaction.Settlement.Taxes) == 0 {
		return nil, fmt.Errorf("die Datei enthält keinen erkennbaren Rechnungsdatensatz nach EN 16931")
	}
	return &doc, nil
}

// Profile returns the guideline identifier the document declares.
func (c *CIIInvoice) Profile() string {
	return strings.TrimSpace(c.Context.Guideline.ID)
}

// EnsureUsableProfile rejects the two ZUGFeRD profiles that are not an
// E-Rechnung in the sense of the law.
//
// MINIMUM and BASIC WL carry no complete invoice — no line items, and in MINIMUM
// not even the tax breakdown. Accepting them would mean booking input tax from a
// document that legally is not an invoice.
func (c *CIIInvoice) EnsureUsableProfile() error {
	switch strings.ToLower(c.Profile()) {
	case profileMinimum:
		return fmt.Errorf("das Profil ZUGFeRD MINIMUM enthält keine vollständige Rechnung und ist keine E-Rechnung im Sinne des Gesetzes (UStAE 14.1 Abs. 14 Satz 4)")
	case profileBasicWL:
		return fmt.Errorf("das Profil ZUGFeRD BASIC WL enthält keine Rechnungspositionen und ist keine E-Rechnung im Sinne des Gesetzes (UStAE 14.1 Abs. 14 Satz 4)")
	}
	return nil
}

// IssueDate returns the Belegdatum as an ISO date.
func (c *CIIInvoice) IssueDate() string { return isoFromCompact(c.Document.IssueDate.Value) }

// DeliveryDate returns the Leistungsdatum as an ISO date.
func (c *CIIInvoice) DeliveryDate() string {
	return isoFromCompact(c.Transaction.Delivery.Event.Occurrence.Value)
}

// DueDate returns the Fälligkeitsdatum as an ISO date.
func (c *CIIInvoice) DueDate() string {
	return isoFromCompact(c.Transaction.Settlement.Terms.DueDate.Value)
}

// Currency returns the invoice currency, defaulting to EUR.
func (c *CIIInvoice) Currency() string {
	if v := strings.TrimSpace(c.Transaction.Settlement.Currency); v != "" {
		return v
	}
	return "EUR"
}

// GrandTotal is the gross amount of the invoice.
func (c *CIIInvoice) GrandTotal() (domain.Cents, error) {
	return domain.ParseCents(c.Transaction.Settlement.Summation.GrandTotal)
}

// isoFromCompact converts the "102" date format (YYYYMMDD) used throughout CII.
// Anything else is passed through: a document may carry an ISO date already, and
// guessing at an unknown format would be worse than showing it unchanged.
func isoFromCompact(v string) string {
	v = strings.TrimSpace(v)
	if len(v) != 8 {
		return v
	}
	for _, c := range v {
		if c < '0' || c > '9' {
			return v
		}
	}
	return v[0:4] + "-" + v[4:6] + "-" + v[6:8]
}
