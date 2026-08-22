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
			Period struct {
				Start ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 StartDateTime"`
				End   ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 EndDateTime"`
			} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BillingSpecifiedPeriod"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeDelivery"`

		Settlement struct {
			Currency string        `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 InvoiceCurrencyCode"`
			Taxes    []ciiTradeTax `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeTax"`
			// Nachlässe und Zuschläge auf Dokumentebene (BG-20, BG-21). Buchfink
			// wertet sie nicht aus, muss aber wissen, ob welche da sind: sie
			// verschieben die Summenregeln, und eine Abweichung wäre dann
			// zulässig statt fehlerhaft.
			AllowancesCharges []ciiAllowanceCharge `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeAllowanceCharge"`
			Terms             struct {
				DueDate ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DueDateDateTime"`
			} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradePaymentTerms"`
			Summation struct {
				LineTotal     string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTotalAmount"`
				TaxBasisTotal string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxBasisTotalAmount"`
				TaxTotal      string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxTotalAmount"`
				GrandTotal    string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GrandTotalAmount"`
				DuePayable    string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DuePayableAmount"`
			} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementHeaderMonetarySummation"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeSettlement"`
	} `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 SupplyChainTradeTransaction"`
}

type ciiParty struct {
	Name string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
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

type ciiDateTime struct {
	Value string `xml:"urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100 DateTimeString"`
}

type ciiAllowanceCharge struct {
	// ChargeIndicator false = Nachlass, true = Zuschlag.
	Indicator struct {
		Value string `xml:"urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100 Indicator"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeIndicator"`
	ActualAmount string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ActualAmount"`
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
		Tax       ciiTradeTax `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeTax"`
		Summation struct {
			LineTotal string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTotalAmount"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementLineMonetarySummation"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeSettlement"`
}

// HasAllowancesOrCharges reports whether the invoice carries allowances or
// charges at document level. Buchfink does not evaluate them, so the sum rules
// only hold in their absence.
func (c *CIIInvoice) HasAllowancesOrCharges() bool {
	return len(c.Transaction.Settlement.AllowancesCharges) > 0
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
