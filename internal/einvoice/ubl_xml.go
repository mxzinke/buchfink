package einvoice

import "encoding/xml"

// The UBL wire structures.
//
// UBL is the other syntax EN 16931 is defined for, and in practice the more
// common one outside Germany: Peppol BIS Billing leads with it, and XRechnung
// accepts it alongside CII. A reader that only speaks CII will one day be
// handed an invoice it cannot open, and the recipient still has to book it.
//
// Invoice and CreditNote are separate root elements with separate namespaces
// and a handful of renamed fields. They are read into one structure here,
// because at the semantic level they are the same document — the difference is
// the type code (BT-3), which decides the sign of the booking, not the shape of
// the data.

const (
	nsUBLInvoice    = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	nsUBLCreditNote = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
	nsCAC           = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	nsCBC           = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
)

// ublValue is a text value with the attributes UBL hangs off it.
type ublValue struct {
	Value         string `xml:",chardata"`
	SchemeID      string `xml:"schemeID,attr"`
	ListID        string `xml:"listID,attr"`
	ListVersionID string `xml:"listVersionID,attr"`
	UnitCode      string `xml:"unitCode,attr"`
	CurrencyID    string `xml:"currencyID,attr"`
	MimeCode      string `xml:"mimeCode,attr"`
	Filename      string `xml:"filename,attr"`
	Name          string `xml:"name,attr"`
}

func (u ublValue) trimmed() string { return trim(u.Value) }

type ublDocument struct {
	XMLName xml.Name

	CustomizationID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CustomizationID"`
	ProfileID       string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ProfileID"`
	ID              string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
	IssueDate       string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 IssueDate"`
	DueDate         string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 DueDate"`
	// InvoiceTypeCode and CreditNoteTypeCode are the same business term (BT-3)
	// under two names; which one is present follows from the root element.
	InvoiceTypeCode    string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 InvoiceTypeCode"`
	CreditNoteTypeCode string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CreditNoteTypeCode"`
	Notes              []ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Note"`
	TaxPointDate       string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxPointDate"`
	DocumentCurrency   string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 DocumentCurrencyCode"`
	TaxCurrency        string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxCurrencyCode"`
	AccountingCost     string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 AccountingCost"`
	BuyerReference     string     `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 BuyerReference"`

	InvoicePeriod *ublPeriod `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 InvoicePeriod"`

	OrderReference *struct {
		ID           string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
		SalesOrderID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 SalesOrderID"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 OrderReference"`

	BillingReference []struct {
		InvoiceDocumentReference struct {
			ID        string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
			IssueDate string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 IssueDate"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 InvoiceDocumentReference"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 BillingReference"`

	DespatchDocumentReference   ublDocRef   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 DespatchDocumentReference"`
	ReceiptDocumentReference    ublDocRef   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 ReceiptDocumentReference"`
	OriginatorDocumentReference ublDocRef   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 OriginatorDocumentReference"`
	ContractDocumentReference   ublDocRef   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 ContractDocumentReference"`
	AdditionalDocumentReference []ublDocRef `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AdditionalDocumentReference"`
	ProjectReference            ublDocRef   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 ProjectReference"`

	AccountingSupplierParty struct {
		Party ublParty `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Party"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AccountingSupplierParty"`
	AccountingCustomerParty struct {
		Party ublParty `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Party"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AccountingCustomerParty"`
	PayeeParty             *ublParty `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PayeeParty"`
	TaxRepresentativeParty *ublParty `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxRepresentativeParty"`

	Delivery *struct {
		ActualDeliveryDate string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ActualDeliveryDate"`
		DeliveryLocation   *struct {
			ID      ublValue    `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
			Address *ublAddress `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Address"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 DeliveryLocation"`
		DeliveryParty *struct {
			PartyName []struct {
				Name string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Name"`
			} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PartyName"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 DeliveryParty"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Delivery"`

	PaymentMeans []struct {
		PaymentMeansCode ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PaymentMeansCode"`
		PaymentID        []string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PaymentID"`
		CardAccount      *struct {
			PrimaryAccountNumberID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PrimaryAccountNumberID"`
			HolderName             string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 HolderName"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 CardAccount"`
		PayeeFinancialAccount *struct {
			ID                         string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
			Name                       string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Name"`
			FinancialInstitutionBranch *struct {
				ID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
			} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 FinancialInstitutionBranch"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PayeeFinancialAccount"`
		PaymentMandate *struct {
			ID                    string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
			PayerFinancialAccount *struct {
				ID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
			} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PayerFinancialAccount"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PaymentMandate"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PaymentMeans"`

	PaymentTerms []struct {
		Note string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Note"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PaymentTerms"`

	AllowanceCharge []ublAllowanceCharge `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AllowanceCharge"`

	TaxTotal []struct {
		TaxAmount   ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxAmount"`
		TaxSubtotal []struct {
			TaxableAmount ublValue       `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxableAmount"`
			TaxAmount     ublValue       `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxAmount"`
			TaxCategory   ublTaxCategory `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxCategory"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxSubtotal"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxTotal"`

	LegalMonetaryTotal struct {
		LineExtensionAmount   string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 LineExtensionAmount"`
		TaxExclusiveAmount    string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxExclusiveAmount"`
		TaxInclusiveAmount    string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxInclusiveAmount"`
		AllowanceTotalAmount  string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 AllowanceTotalAmount"`
		ChargeTotalAmount     string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ChargeTotalAmount"`
		PrepaidAmount         string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PrepaidAmount"`
		PayableRoundingAmount string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PayableRoundingAmount"`
		PayableAmount         string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PayableAmount"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 LegalMonetaryTotal"`

	InvoiceLine    []ublLine `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 InvoiceLine"`
	CreditNoteLine []ublLine `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 CreditNoteLine"`
}

type ublDocRef struct {
	ID               ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
	DocumentTypeCode string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 DocumentTypeCode"`
	DocumentType     string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 DocumentType"`
	Description      string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 DocumentDescription"`
	Attachment       *struct {
		EmbeddedDocument  ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 EmbeddedDocumentBinaryObject"`
		ExternalReference *struct {
			URI string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 URI"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 ExternalReference"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Attachment"`
}

type ublPeriod struct {
	StartDate       string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 StartDate"`
	EndDate         string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 EndDate"`
	DescriptionCode string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 DescriptionCode"`
}

type ublParty struct {
	EndpointID          ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 EndpointID"`
	PartyIdentification []struct {
		ID ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PartyIdentification"`
	PartyName []struct {
		Name string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Name"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PartyName"`
	PostalAddress  *ublAddress `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PostalAddress"`
	PartyTaxScheme []struct {
		CompanyID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CompanyID"`
		TaxScheme struct {
			ID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxScheme"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PartyTaxScheme"`
	PartyLegalEntity []struct {
		RegistrationName string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 RegistrationName"`
		CompanyID        ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CompanyID"`
		CompanyLegalForm string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CompanyLegalForm"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 PartyLegalEntity"`
	Contact *struct {
		Name           string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Name"`
		Telephone      string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Telephone"`
		ElectronicMail string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ElectronicMail"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Contact"`
}

type ublAddress struct {
	StreetName           string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 StreetName"`
	AdditionalStreetName string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 AdditionalStreetName"`
	CityName             string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CityName"`
	PostalZone           string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PostalZone"`
	CountrySubentity     string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CountrySubentity"`
	AddressLine          []struct {
		Line string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Line"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AddressLine"`
	Country struct {
		IdentificationCode string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 IdentificationCode"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Country"`
}

type ublTaxCategory struct {
	ID                     ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
	Percent                string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Percent"`
	TaxExemptionReasonCode string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxExemptionReasonCode"`
	TaxExemptionReason     string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 TaxExemptionReason"`
	TaxScheme              struct {
		ID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxScheme"`
}

type ublAllowanceCharge struct {
	ChargeIndicator           string         `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ChargeIndicator"`
	AllowanceChargeReasonCode string         `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 AllowanceChargeReasonCode"`
	AllowanceChargeReason     string         `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 AllowanceChargeReason"`
	MultiplierFactor          string         `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 MultiplierFactorNumeric"`
	Amount                    string         `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Amount"`
	BaseAmount                string         `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 BaseAmount"`
	TaxCategory               ublTaxCategory `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 TaxCategory"`
}

func (a ublAllowanceCharge) isCharge() bool {
	return trim(a.ChargeIndicator) == "true" || trim(a.ChargeIndicator) == "1"
}

type ublLine struct {
	ID   string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
	Note []string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Note"`
	// InvoicedQuantity and CreditedQuantity are the same business term (BT-129).
	InvoicedQuantity    ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 InvoicedQuantity"`
	CreditedQuantity    ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 CreditedQuantity"`
	LineExtensionAmount string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 LineExtensionAmount"`
	AccountingCost      string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 AccountingCost"`

	InvoicePeriod      *ublPeriod `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 InvoicePeriod"`
	OrderLineReference *struct {
		LineID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 LineID"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 OrderLineReference"`
	DocumentReference []ublDocRef          `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 DocumentReference"`
	AllowanceCharge   []ublAllowanceCharge `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AllowanceCharge"`

	Item struct {
		Description              string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Description"`
		Name                     string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Name"`
		BuyersItemIdentification *struct {
			ID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 BuyersItemIdentification"`
		SellersItemIdentification *struct {
			ID string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 SellersItemIdentification"`
		StandardItemIdentification *struct {
			ID ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ID"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 StandardItemIdentification"`
		OriginCountry *struct {
			IdentificationCode string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 IdentificationCode"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 OriginCountry"`
		CommodityClassification []struct {
			ItemClassificationCode ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 ItemClassificationCode"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 CommodityClassification"`
		ClassifiedTaxCategory  []ublTaxCategory `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 ClassifiedTaxCategory"`
		AdditionalItemProperty []struct {
			Name  string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Name"`
			Value string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Value"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AdditionalItemProperty"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Item"`

	Price struct {
		PriceAmount     string   `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 PriceAmount"`
		BaseQuantity    ublValue `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 BaseQuantity"`
		AllowanceCharge []struct {
			Amount     string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 Amount"`
			BaseAmount string `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2 BaseAmount"`
		} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 AllowanceCharge"`
	} `xml:"urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2 Price"`
}
