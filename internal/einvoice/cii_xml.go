package einvoice

import "encoding/xml"

// The CII wire structures.
//
// They exist only to be mapped onto the semantic model and back; nothing outside
// this file should reach into them. Every element is namespace qualified,
// because CII reuses the same local names across namespaces — `ID` alone is
// ambiguous, and a parser that matches on the local name will happily read a
// line identifier into a document identifier.

const (
	nsRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	nsRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	nsUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"
	nsQDT = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
)

type ciiDocument struct {
	XMLName xml.Name    `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 CrossIndustryInvoice"`
	Context ciiContext  `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 ExchangedDocumentContext"`
	Doc     ciiExchange `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 ExchangedDocument"`
	Trade   ciiTrade    `xml:"urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100 SupplyChainTradeTransaction"`
}

type ciiContext struct {
	BusinessProcess struct {
		ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BusinessProcessSpecifiedDocumentContextParameter"`
	Guideline struct {
		ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GuidelineSpecifiedDocumentContextParameter"`
}

type ciiExchange struct {
	ID        string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	TypeCode  string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
	IssueDate ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IssueDateTime"`
	Notes     []struct {
		Content     string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Content"`
		SubjectCode string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SubjectCode"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IncludedNote"`
}

type ciiDateTime struct {
	String struct {
		Value  string `xml:",chardata"`
		Format string `xml:"format,attr"`
	} `xml:"urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100 DateTimeString"`
}

func (d ciiDateTime) date() Date { return NewDateFromFormat(d.String.Value, d.String.Format) }

type ciiTrade struct {
	Lines      []ciiLine     `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IncludedSupplyChainTradeLineItem"`
	Agreement  ciiAgreement  `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeAgreement"`
	Delivery   ciiDelivery   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeDelivery"`
	Settlement ciiSettlement `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableHeaderTradeSettlement"`
}

type ciiAgreement struct {
	BuyerReference    string                `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BuyerReference"`
	Seller            ciiParty              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SellerTradeParty"`
	Buyer             ciiParty              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BuyerTradeParty"`
	TaxRepresentative *ciiParty             `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SellerTaxRepresentativeTradeParty"`
	SellerOrder       ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SellerOrderReferencedDocument"`
	BuyerOrder        ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BuyerOrderReferencedDocument"`
	Contract          ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ContractReferencedDocument"`
	Additional        []ciiSupportingDoc    `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AdditionalReferencedDocument"`
	Project           ciiProcuringProject   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedProcuringProject"`
}

type ciiProcuringProject struct {
	ID   string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	Name string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
}

type ciiReferencedDocument struct {
	IssuerAssignedID string       `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IssuerAssignedID"`
	LineID           string       `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineID"`
	TypeCode         string       `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
	ReferenceTypeID  string       `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReferenceTypeCode"`
	FormattedIssue   ciiFormatted `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 FormattedIssueDateTime"`
}

type ciiFormatted struct {
	String struct {
		Value  string `xml:",chardata"`
		Format string `xml:"format,attr"`
	} `xml:"urn:un:unece:uncefact:data:standard:QualifiedDataType:100 DateTimeString"`
}

func (f ciiFormatted) date() Date { return NewDateFromFormat(f.String.Value, f.String.Format) }

// ciiSupportingDoc carries BG-24, and doubles as the carrier of BT-17 and
// BT-18 — CII routes the tender reference and the object identifier through the
// same element, told apart by their TypeCode.
type ciiSupportingDoc struct {
	IssuerAssignedID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IssuerAssignedID"`
	URIID            string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 URIID"`
	TypeCode         string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
	Name             string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
	ReferenceTypeID  string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReferenceTypeCode"`
	Binary           struct {
		Value    string `xml:",chardata"`
		MimeCode string `xml:"mimeCode,attr"`
		Filename string `xml:"filename,attr"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AttachmentBinaryObject"`
}

type ciiParty struct {
	ID       []string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	GlobalID []struct {
		Value    string `xml:",chardata"`
		SchemeID string `xml:"schemeID,attr"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GlobalID"`
	Name        string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
	Description string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Description"`
	LegalOrg    *struct {
		ID struct {
			Value    string `xml:",chardata"`
			SchemeID string `xml:"schemeID,attr"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
		TradingName string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TradingBusinessName"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLegalOrganization"`
	Contact *struct {
		PersonName string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PersonName"`
		Department string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DepartmentName"`
		Phone      struct {
			Number string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CompleteNumber"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TelephoneUniversalCommunication"`
		Email struct {
			URIID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 URIID"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 EmailURIUniversalCommunication"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DefinedTradeContact"`
	Address *ciiAddress `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PostalTradeAddress"`
	URI     *struct {
		URIID struct {
			Value    string `xml:",chardata"`
			SchemeID string `xml:"schemeID,attr"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 URIID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 URIUniversalCommunication"`
	TaxRegistrations []struct {
		ID struct {
			Value    string `xml:",chardata"`
			SchemeID string `xml:"schemeID,attr"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTaxRegistration"`
}

type ciiAddress struct {
	PostCode    string   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PostcodeCode"`
	LineOne     string   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineOne"`
	LineTwo     string   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTwo"`
	LineThree   string   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineThree"`
	CityName    string   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CityName"`
	CountryID   string   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CountryID"`
	Subdivision []string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CountrySubDivisionName"`
}

type ciiDelivery struct {
	ShipTo *ciiParty `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ShipToTradeParty"`
	Event  struct {
		Occurrence ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 OccurrenceDateTime"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ActualDeliverySupplyChainEvent"`
	DespatchAdvice  ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DespatchAdviceReferencedDocument"`
	ReceivingAdvice ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReceivingAdviceReferencedDocument"`
}

type ciiSettlement struct {
	PaymentReference  string               `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PaymentReference"`
	TaxCurrency       string               `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxCurrencyCode"`
	Currency          string               `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 InvoiceCurrencyCode"`
	Payee             *ciiParty            `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PayeeTradeParty"`
	PaymentMeans      []ciiPaymentMeans    `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementPaymentMeans"`
	Taxes             []ciiTradeTax        `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeTax"`
	Period            *ciiPeriod           `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BillingSpecifiedPeriod"`
	AllowancesCharges []ciiAllowanceCharge `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeAllowanceCharge"`
	PaymentTerms      []struct {
		Description string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Description"`
		DueDate     ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DueDateDateTime"`
		MandateID   string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DirectDebitMandateID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradePaymentTerms"`
	Summation  ciiSummation `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementHeaderMonetarySummation"`
	InvoiceRef []struct {
		IssuerAssignedID string       `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IssuerAssignedID"`
		FormattedIssue   ciiFormatted `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 FormattedIssueDateTime"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 InvoiceReferencedDocument"`
	AccountingReference struct {
		ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReceivableSpecifiedTradeAccountingAccount"`
}

type ciiPaymentMeans struct {
	TypeCode    string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
	Information string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Information"`
	Card        *struct {
		ID         string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
		HolderName string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CardholderName"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeSettlementFinancialCard"`
	PayerAccount *struct {
		IBAN string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IBANID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PayerPartyDebtorFinancialAccount"`
	PayeeAccount *struct {
		IBAN          string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IBANID"`
		AccountName   string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AccountName"`
		ProprietaryID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ProprietaryID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PayeePartyCreditorFinancialAccount"`
	PayeeInstitution *struct {
		BIC string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BICID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 PayeeSpecifiedCreditorFinancialInstitution"`
}

type ciiPeriod struct {
	Start ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 StartDateTime"`
	End   ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 EndDateTime"`
}

type ciiTradeTax struct {
	CalculatedAmount    string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CalculatedAmount"`
	TypeCode            string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TypeCode"`
	ExemptionReason     string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ExemptionReason"`
	BasisAmount         string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BasisAmount"`
	CategoryCode        string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CategoryCode"`
	ExemptionReasonCode string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ExemptionReasonCode"`
	TaxPointDate        ciiDateTime `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxPointDate"`
	DueDateTypeCode     string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DueDateTypeCode"`
	RateApplicable      string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 RateApplicablePercent"`
}

type ciiAllowanceCharge struct {
	Indicator struct {
		Value string `xml:"urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100 Indicator"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeIndicator"`
	CalculationPercent string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CalculationPercent"`
	BasisAmount        string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BasisAmount"`
	ActualAmount       string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ActualAmount"`
	ReasonCode         string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReasonCode"`
	Reason             string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Reason"`
	CategoryTradeTax   ciiTradeTax `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 CategoryTradeTax"`
}

func (a ciiAllowanceCharge) isCharge() bool {
	v := a.Indicator.Value
	return v == "true" || v == "1"
}

type ciiSummation struct {
	LineTotal      string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTotalAmount"`
	ChargeTotal    string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeTotalAmount"`
	AllowanceTotal string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AllowanceTotalAmount"`
	TaxBasisTotal  string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxBasisTotalAmount"`
	TaxTotals      []ciiCurrencyAmount `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TaxTotalAmount"`
	RoundingAmount string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 RoundingAmount"`
	GrandTotal     string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GrandTotalAmount"`
	TotalPrepaid   string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 TotalPrepaidAmount"`
	DuePayable     string              `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DuePayableAmount"`
}

// ciiCurrencyAmount is an amount that says which currency it is in. The tax
// total appears twice on an invoice accounted in a second currency (BT-110 and
// BT-111), in the same element — only the attribute tells them apart.
type ciiCurrencyAmount struct {
	Value      string `xml:",chardata"`
	CurrencyID string `xml:"currencyID,attr"`
}

type ciiLine struct {
	Document struct {
		LineID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineID"`
		Notes  []struct {
			Content string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Content"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 IncludedNote"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AssociatedDocumentLineDocument"`
	Product    ciiProduct        `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeProduct"`
	Agreement  ciiLineAgreement  `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeAgreement"`
	Delivery   ciiLineDelivery   `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeDelivery"`
	Settlement ciiLineSettlement `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedLineTradeSettlement"`
}

type ciiProduct struct {
	ID       string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SellerAssignedID"`
	BuyerID  string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BuyerAssignedID"`
	GlobalID struct {
		Value    string `xml:",chardata"`
		SchemeID string `xml:"schemeID,attr"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GlobalID"`
	Name        string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Name"`
	Description string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Description"`
	Attributes  []struct {
		Description string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Description"`
		Value       string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 Value"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableProductCharacteristic"`
	Classifications []struct {
		ClassCode struct {
			Value         string `xml:",chardata"`
			ListID        string `xml:"listID,attr"`
			ListVersionID string `xml:"listVersionID,attr"`
		} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ClassCode"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 DesignatedProductClassification"`
	Origin struct {
		ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 OriginTradeCountry"`
}

type ciiLineAgreement struct {
	BuyerOrder ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BuyerOrderReferencedDocument"`
	GrossPrice struct {
		ChargeAmount     string               `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeAmount"`
		BasisQuantity    ciiQuantity          `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BasisQuantity"`
		AllowanceCharges []ciiAllowanceCharge `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AppliedTradeAllowanceCharge"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 GrossPriceProductTradePrice"`
	NetPrice struct {
		ChargeAmount  string      `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ChargeAmount"`
		BasisQuantity ciiQuantity `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BasisQuantity"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 NetPriceProductTradePrice"`
}

type ciiQuantity struct {
	Value    string `xml:",chardata"`
	UnitCode string `xml:"unitCode,attr"`
}

type ciiLineDelivery struct {
	BilledQuantity ciiQuantity `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BilledQuantity"`
}

type ciiLineSettlement struct {
	Tax               ciiTradeTax          `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ApplicableTradeTax"`
	Period            *ciiPeriod           `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 BillingSpecifiedPeriod"`
	AllowancesCharges []ciiAllowanceCharge `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeAllowanceCharge"`
	Summation         struct {
		LineTotal string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 LineTotalAmount"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 SpecifiedTradeSettlementLineMonetarySummation"`
	AccountingReference struct {
		ID string `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ID"`
	} `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 ReceivableSpecifiedTradeAccountingAccount"`
	ObjectReference ciiReferencedDocument `xml:"urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100 AdditionalReferencedDocument"`
}
