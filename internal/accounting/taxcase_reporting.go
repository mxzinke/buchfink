package accounting

import (
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

type TaxPrepayment int

const (
	TaxPrepaymentNone TaxPrepayment = iota
	TaxPrepaymentOnPayment
	TaxPrepaymentWithInvoice
)

// TaxCaseVAT beschreibt Kennzahl, Vorzeichen und vorgezogene Steuerentstehung.
// HasBase verwendet die TaxBase, BaseFromAmount den Betrag der Erlöszeile.
type TaxCaseVAT struct {
	Code               string
	HasBase            bool
	BaseFromAmount     bool
	TaxSide            domain.Side
	Prepayment         TaxPrepayment
	LongTerm           bool
	BadDebtBase        bool
	AllowClosing       bool
	RecipientDependent bool
	ZMKind             domain.ZMLineKind
}

// ForRecipient ordnet Leistungen mit Steuerschuld des Empfängers zu:
// EU-Leistung in Kz 21 und ZM (§ 18b Satz 1 Nr. 2, § 18a UStG),
// Inland in Kz 60 (§ 13b Abs. 5 UStG), übrige Empfänger in Kz 45.
func (v TaxCaseVAT) ForRecipient(eu, domestic bool) TaxCaseVAT {
	if v.RecipientDependent {
		switch {
		case eu:
			v.Code, v.ZMKind = VatCodeEUServices, domain.ZMKindService
		case domestic:
			v.Code, v.ZMKind = VatCodeRecipientLiable, ""
		default:
			v.Code, v.ZMKind = VatCodeNotTaxable, ""
		}
	}
	return v
}

// VatCode beschreibt den Beitrag einer Steuer- oder Erlöszeile. Ohne Schlüssel
// bezeichnet c den Steuerfall des Erlöskontos; Aufwandskonten haben keinen Beitrag.
// SKONTO-Schlüssel berichtigen dieselbe Kennzahl (§ 17 Abs. 1 UStG).
func (c TaxCase) VatCode(key string) TaxCaseVAT {
	originalKey := key
	key = strings.TrimPrefix(key, TaxKeySkontoPrefix)
	v := TaxCaseVAT{}
	switch {
	case key == "":
		v.BaseFromAmount = true
		switch c.treatment {
		case domain.TaxTreatmentExempt:
			v.Code = VatCodeExemptNoDeduction
		case domain.TaxTreatmentZeroRated:
			v.Code, v.Prepayment = VatCodeZeroRate, TaxPrepaymentOnPayment
		case domain.TaxTreatmentReverseChargeSupply:
			v.Code, v.RecipientDependent = VatCodeNotTaxable, true
		case domain.TaxTreatmentNotTaxable:
			v.Code = VatCodeNotTaxable
		default:
			return TaxCaseVAT{}
		}
	case key == "UST19", key == "UST7":
		v.Code = VatCodeStandardRate
		if key == "UST7" {
			v.Code = VatCodeReducedRate
		}
		v.HasBase, v.TaxSide = true, domain.SideCredit
		v.Prepayment, v.BadDebtBase = TaxPrepaymentOnPayment, originalKey == key
	case key == TaxKeyUnlawful:
		// § 14c UStG: geschuldete Steuer ohne Bemessungsgrundlage im Vordruck.
		v.Code, v.TaxSide = VatCodeUnlawfulTax, domain.SideCredit
	case key == "IG19_UST", key == "IG7_UST":
		v.Code = VatCodeAcquisition19
		if key == "IG7_UST" {
			v.Code = VatCodeAcquisition7
		}
		v.HasBase, v.TaxSide = true, domain.SideCredit
	case isReverseChargeKey(key, TaxKeyReverseChargeEUPrefix, "_UST"):
		v.Code, v.HasBase, v.TaxSide = VatCodeReverseChargeEU, true, domain.SideCredit
		v.Prepayment, v.LongTerm = TaxPrepaymentOnPayment, true
	case isReverseChargeKey(key, TaxKeyReverseChargePrefix, "_UST"):
		v.Code, v.HasBase, v.TaxSide = VatCodeReverseChargeOther, true, domain.SideCredit
		v.Prepayment, v.LongTerm = TaxPrepaymentOnPayment, true
	case key == "VST19", key == "VST7":
		v.Code, v.TaxSide, v.Prepayment = VatCodeInputTax, domain.SideDebit, TaxPrepaymentWithInvoice
	case key == "IG19_VST", key == "IG7_VST":
		v.Code, v.TaxSide = VatCodeInputTaxAcquisition, domain.SideDebit
	case isReverseChargeKey(key, TaxKeyReverseChargeEUPrefix, "_VST"), isReverseChargeKey(key, TaxKeyReverseChargePrefix, "_VST"):
		v.Code, v.TaxSide = VatCodeInputTaxReverse, domain.SideDebit
		v.Prepayment, v.LongTerm = TaxPrepaymentOnPayment, true
	case key == TaxKeyInputTaxCorrection:
		// § 15a UStG: auch als Abschlussbuchung ein Beitrag zur Voranmeldung.
		v.Code, v.TaxSide, v.AllowClosing = VatCodeInputTaxCorrection, domain.SideDebit, originalKey == key
	}
	return v
}

// isReverseChargeKey erkennt die Schlüssel eines § 13b-Absatzes. Der Präfix
// „RC" ist in „RCEU" enthalten; geprüft wird deshalb der Satz dazwischen.
func isReverseChargeKey(key, prefix, suffix string) bool {
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return false
	}
	rate := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
	if rate == "" {
		return false
	}
	for _, r := range rate {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// RevenueTreatments ordnet jedes Erlöskonto seinem Steuerfall zu.
//
// Der Katalog der Buchungsgruppen liefert die Zuordnung; dazu kommen die
// steuerfreien Erlöskonten des SKR04, die keine Gruppe anbietet, eine
// Handbuchung aber erreicht. Die Tabelle steht einmal — eine zweite Kopie in der
// Auswertung driftete stillschweigend, und ein neuer Steuerfall fiele einfach
// aus der Voranmeldung heraus.
func RevenueTreatments() map[string]domain.TaxTreatment {
	out := RevenueAccountTreatments()
	for _, account := range []string{"4110", "4160", "4165"} {
		out[account] = domain.TaxTreatmentExempt
	}
	// Das gewährte Skonto auf eine Leistung mit Steuerschuld des Empfängers
	// mindert deren Bemessungsgrundlage in Kz 21, 45 oder 60 und in der
	// Zusammenfassenden Meldung (§ 17 Abs. 1 Satz 8, § 18a Abs. 7 Satz 2 UStG).
	out[ReverseChargeSupplySkontoAccount] = domain.TaxTreatmentReverseChargeSupply
	return out
}

// ReverseChargeSupplySkontoAccount ist das Skontokonto zum Erlöskonto 4337
// (SKR04: Gewährte Skonti aus Leistungen, für die der Leistungsempfänger die
// Umsatzsteuer nach § 13b UStG schuldet).
const ReverseChargeSupplySkontoAccount = "4741"
