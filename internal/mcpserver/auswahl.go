package mcpserver

import (
	"reflect"
	"regexp"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/google/jsonschema-go/jsonschema"
)

// Auswahlfelder haben eine feste Werteliste; das Schema nennt sie, und ein
// unbekannter Wert scheitert schon an der Schemaprüfung. Die Werte sind
// deutsch. Wo ein Wert auf einen Schlüssel des Kerns zeigt, übersetzt die
// Tabelle daneben.

type (
	KontenArt        string
	Richtung         string
	AuswertungArt    string
	Gliederung       string
	BelegStatus      string
	BelegArt         string
	DateiRolle       string
	EingangWeg       string
	BezahlungDirekt  string
	Bezahlung        string
	AnzahlungZiel    string
	Buchungsseite    string
	BankStatus       string
	DifferenzArt     string
	KontaktTyp       string
	PostenRichtung   string
	PflichtStatus    string
	PflichtArt       string
	AnlassArt        string
	RechnungArt      string
	AbschlussArt     string
	AbgrenzungArt    string
	RueckstellungArt string
	AnlageKlasse     string
	Fondsart         string
	AfaMethode       string
	ErklaerungArt    string
	JahrArt          string
	EBilanzFormat    string
	Bilanzart        string
	Quelle           string
	DarlehenRichtung string
	DarlehenArt      string
	Zinsmethode      string
	Monat            string
	Zeitraum         string
)

type choice struct {
	values []string
	desc   string
}

var choices = map[reflect.Type]choice{
	reflect.TypeFor[KontenArt]():       {[]string{"suche", "buchungsgruppen", "steuerfaelle"}, "suche: Konten des SKR04; buchungsgruppen; steuerfaelle"},
	reflect.TypeFor[Richtung]():        {[]string{"eingang", "ausgang"}, "eingang oder ausgang"},
	reflect.TypeFor[AuswertungArt]():   {[]string{"susa", "kontoblatt", "bilanz", "guv", "ust", "pruefbericht", "darlehen", "unterbilanz"}, "Art der Auswertung"},
	reflect.TypeFor[Gliederung]():      {[]string{"voll", "kurz", "buchstaben"}, "Tiefe der Gliederung; Vorgabe voll"},
	reflect.TypeFor[BelegStatus]():     {[]string{"abgelegt", "gebucht", "verworfen"}, "abgelegt (noch nicht gebucht), gebucht oder verworfen"},
	reflect.TypeFor[BelegArt]():        {choiceKeys(receiptKinds), "Belegart"},
	reflect.TypeFor[DateiRolle]():      {[]string{"original", "anhang", "darstellung"}, "Rolle der Datei"},
	reflect.TypeFor[EingangWeg]():      {[]string{"email", "scan", "upload"}, "Weg des Eingangs"},
	reflect.TypeFor[BezahlungDirekt](): {[]string{"offen", "kreditkarte"}, "offen (Verbindlichkeit) oder kreditkarte (Firmenkreditkarte 3610)"},
	reflect.TypeFor[Bezahlung]():       {choiceKeys(settlements), "offen, bezahlt (mit zahlungskonto), kreditkarte oder auslage (ein Gesellschafter hat privat bezahlt)"},
	reflect.TypeFor[AnzahlungZiel]():   {choiceKeys(advanceTargets), "der Beleg ist eine geleistete Anzahlung auf Vorräte, Sachanlagen oder immaterielle Werte"},
	reflect.TypeFor[Buchungsseite]():   {[]string{"S", "H"}, "S (Soll) oder H (Haben)"},
	reflect.TypeFor[BankStatus]():      {[]string{"offen", "zugeordnet", "ignoriert", "alle"}, "Stand der Umsätze; Vorgabe offen"},
	reflect.TypeFor[DifferenzArt]():    {choiceKeys(differenceKinds), "Art der Differenz zum offenen Posten"},
	reflect.TypeFor[KontaktTyp]():      {[]string{"kunde", "lieferant"}, "kunde oder lieferant"},
	reflect.TypeFor[PostenRichtung]():  {[]string{"forderung", "verbindlichkeit", "alle"}, "Vorgabe alle"},
	reflect.TypeFor[PflichtStatus]():   {choiceKeys(statusFromLabel), "Stand der Pflicht"},
	reflect.TypeFor[PflichtArt]():      {obligationKindValues(), "Pflichtart wie in fristen; die Anker aus pflichten.md gelten auch"},
	reflect.TypeFor[AnlassArt]():       {eventKindValues(), "Art des Anlasses"},
	reflect.TypeFor[RechnungArt]():     {[]string{"rechnung", "abschlag", "schlussrechnung"}, "Vorgabe rechnung"},
	reflect.TypeFor[AbschlussArt]():    {choiceKeys(closingRuns), "Schritt des Abschlusses; die Angaben stehen im gleichnamigen Feld"},
	reflect.TypeFor[AbgrenzungArt]():   {choiceKeys(accrualKinds), "aktiv (aktiver RAP), passiv (passiver RAP) oder disagio"},
	reflect.TypeFor[RueckstellungArt](): {choiceKeys(provisionKinds),
		"Art der Rückstellung; Steuerrückstellungen über art=steuerrueckstellung"},
	reflect.TypeFor[AnlageKlasse]():  {choiceKeys(assetClasses), "sachanlage, immateriell oder finanzanlage"},
	reflect.TypeFor[Fondsart]():      {choiceKeys(fundClasses), "Fondsart nach § 2 InvStG"},
	reflect.TypeFor[AfaMethode]():    {choiceKeys(depreciationMethods), "Abschreibungsmethode"},
	reflect.TypeFor[ErklaerungArt](): {[]string{"ustva", "zm", "ust_jahr", "kst", "gewst"}, "Art der Anmeldung oder Erklärung"},
	reflect.TypeFor[JahrArt]():       {choiceKeys(declarationKinds), "Jahreserklärung"},
	reflect.TypeFor[EBilanzFormat](): {[]string{"xbrl", "susa_csv"},
		"xbrl (Vorgabe): die Instanz zur Übermittlung; susa_csv: Summen- und Saldenliste mit Taxonomie-Position je Konto, etwa für eBilanz+"},
	reflect.TypeFor[Bilanzart](): {choiceKeys(balanceKinds),
		"jahresabschluss (Vorgabe): Bilanz und GuV des Geschäftsjahres; eroeffnungsbilanz: Bilanz auf den Gründungstag, nur im Geschäftsjahr der Gründung"},
	reflect.TypeFor[Quelle](): {choiceKeys(inputSources), "Herkunft der Angabe; Vorgabe mensch"},
	reflect.TypeFor[DarlehenRichtung](): {choiceKeys(loanDirections),
		"an_gesellschaft: der Gesellschafter leiht der GmbH; an_gesellschafter: die GmbH leiht ihm"},
	reflect.TypeFor[Zinsmethode](): {choiceKeys(dayCounts), "Zinsmethode; Vorgabe act/365"},
	reflect.TypeFor[DarlehenArt](): {[]string{"vertrag", "zinsen"},
		"vertrag (Vorgabe): Vertrag anlegen oder berichtigen; zinsen: Zinsen eines erfassten Darlehens buchen"},
}

var patterns = map[reflect.Type]*jsonschema.Schema{
	reflect.TypeFor[Monat](): {Type: "string", Pattern: `^[0-9]{4}-(0[1-9]|1[0-2])$`, Description: "Monat JJJJ-MM"},
	reflect.TypeFor[Zeitraum](): {Type: "string", Pattern: zeitraumPattern.String(),
		Description: "ustva und zm: 2026-03 oder 2026-Q1, auch der Schlüssel aus fristen (ustva.2026-Q1); Jahreserklärungen: 2026"},
}

var zeitraumPattern = regexp.MustCompile(`^((ustva|zm)\.)?[0-9]{4}(-(0[1-9]|1[0-2]|Q[1-4]))?$`)

func init() {
	for t, c := range choices {
		enum := make([]any, 0, len(c.values))
		for _, v := range c.values {
			enum = append(enum, v)
		}
		typeSchemas[t] = &jsonschema.Schema{Type: "string", Enum: enum, Description: c.desc}
	}
	for t, s := range patterns {
		typeSchemas[t] = s
	}
}

func choiceKeys[K ~string, V any](m map[K]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, string(k))
	}
	sort.Strings(out)
	return out
}

// pick übersetzt einen Auswahlwert; leer liefert den Nullwert.
func pick[K ~string, V any](field string, k K, m map[K]V) (V, error) {
	var zero V
	if k == "" {
		return zero, nil
	}
	v, ok := m[k]
	if !ok {
		return zero, inputError(field, "%q ist kein gültiger Wert (%v)", k, choiceKeys(m))
	}
	return v, nil
}

// label übersetzt einen Schlüssel des Kerns zurück; unbekannte bleiben, wie
// sie sind.
func label[K ~string, V comparable](m map[K]V, v V) string {
	for k, x := range m {
		if x == v {
			return string(k)
		}
	}
	return ""
}

var receiptKinds = map[BelegArt]domain.ReceiptKind{
	"rechnung": domain.ReceiptKindInvoice, "quittung": domain.ReceiptKindTicket, "kontoauszug": domain.ReceiptKindStatement,
	"eigenbeleg": domain.ReceiptKindSelfIssued, "brief": domain.ReceiptKindLetter, "sonstiges": domain.ReceiptKindOther,
}

var advanceTargets = map[AnzahlungZiel]accounting.AdvanceTarget{
	"vorraete": accounting.AdvanceTargetInventory, "sachanlage": accounting.AdvanceTargetTangible,
	"immateriell": accounting.AdvanceTargetIntangible,
}

var accrualKinds = map[AbgrenzungArt]domain.AccrualKind{
	"aktiv": domain.AccrualActive, "passiv": domain.AccrualPassive, "disagio": domain.AccrualDisagio,
}

var provisionKinds = map[RueckstellungArt]domain.ProvisionKind{
	"ungewisse_verbindlichkeit": domain.ProvisionUncertainLiability, "drohverlust": domain.ProvisionPendingLoss,
	"instandhaltung": domain.ProvisionDeferredMaintenance, "kulanz": domain.ProvisionWarranty,
	"abschlusskosten": domain.ProvisionClosingCosts, "aufbewahrung": domain.ProvisionRetentionCosts,
	"personal": domain.ProvisionPersonnel,
}

var assetClasses = map[AnlageKlasse]domain.AssetClass{
	"sachanlage": domain.AssetClassTangible, "immateriell": domain.AssetClassIntangible,
	"finanzanlage": domain.AssetClassFinancial,
}

var fundClasses = map[Fondsart]accounting.FundClass{
	"aktienfonds": accounting.FundEquity, "mischfonds": accounting.FundMixed,
	"immobilienfonds": accounting.FundRealEstate, "auslands_immobilienfonds": accounting.FundForeignRealEstate,
	"sonstiger_fonds": accounting.FundOther,
}

var depreciationMethods = map[AfaMethode]domain.DepreciationMethod{
	"linear": domain.DepreciationLinear, "degressiv": domain.DepreciationDegressive,
	"elektrofahrzeug": domain.DepreciationElectricVehicle, "gebaeude": domain.DepreciationBuildingLinear,
	"sammelposten": domain.DepreciationPool, "sofort": domain.DepreciationImmediate, "keine": domain.DepreciationNone,
}

var assetStatus = map[string]domain.AssetStatus{
	"im_bestand": domain.AssetStatusActive, "abgeschrieben": domain.AssetStatusFullyDepr,
	"abgegangen": domain.AssetStatusDisposed, "ohne_zugangsbuchung": domain.AssetStatusNotYetBooked,
	"afa_faellig": domain.AssetStatusDepreciateDue,
}

var suggestionKinds = map[string]service.SuggestionKind{
	"offener_posten": service.SuggestionOpenItem, "sammelzahlung": service.SuggestionCollective,
	"regel": service.SuggestionRule, "kreditkarte": service.SuggestionCreditCard,
}

var depths = map[Gliederung]domain.StatementDepth{
	"voll": domain.DepthFull, "kurz": domain.DepthShort, "buchstaben": domain.DepthLetters,
}

var loanDirections = map[DarlehenRichtung]domain.LoanDirection{
	"an_gesellschaft": domain.LoanToCompany, "an_gesellschafter": domain.LoanToShareholder,
}

var dayCounts = map[Zinsmethode]domain.DayCount{
	"act/365": domain.DayCountAct365, "act/360": domain.DayCountAct360, "30/360": domain.DayCount30E360,
}

var balanceKinds = map[Bilanzart]ebilanz.Kind{
	"jahresabschluss": ebilanz.KindAnnual, "eroeffnungsbilanz": ebilanz.KindOpening,
}

var inputSources = map[Quelle]declaration.InputSource{
	"bescheid": declaration.InputFromNotice, "beschluss": declaration.InputFromResolution,
	"beleg": declaration.InputFromReceipt, "mensch": declaration.InputFromPerson, "buecher": declaration.InputFromBooks,
}

func obligationKindValues() []string {
	seen := map[string]bool{}
	var out []string
	for _, k := range domain.ObligationKinds() {
		for _, v := range []string{string(k), k.Anchor()} {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	sort.Strings(out)
	return out
}

func eventKindValues() []string {
	return []string{string(domain.EventShareholderChange), string(domain.EventBeneficialOwnerChange),
		string(domain.EventCorporatePrepaymentNotice), string(domain.EventTradePrepaymentNotice)}
}
