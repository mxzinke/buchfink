package xrechnung

import "github.com/buchfink/buchfink/internal/einvoice"

// The rule inventory of the German CIUS.
//
// XRechnung narrows EN 16931: it makes optional fields mandatory, restricts
// code lists and forbids combinations the standard would allow. Every rule
// carries the severity KoSIT gives it — a good third are warnings, and treating
// them as errors would block invoices that public authorities accept.
//
// Only identifiers, severities and business term numbers are kept; the wording
// of the findings is Buchfink's own and lives in ruleset.go. KoSIT's texts are
// Apache-2.0 and could be reused — they are written for a validator report,
// though, not for somebody deciding whether to book the invoice.
//
// Erzeugt aus: itplr-kosit/xrechnung-schematron (Apache-2.0).

var xrechnungRules = map[string]einvoice.RuleInfo{
	"BR-DE-1":        {Terms: []string{"BG-16"}, Severity: einvoice.SeverityFatal},
	"BR-DE-10":       {Terms: []string{"BG-15", "BT-77"}, Severity: einvoice.SeverityFatal},
	"BR-DE-11":       {Terms: []string{"BG-15", "BT-78"}, Severity: einvoice.SeverityFatal},
	"BR-DE-14":       {Terms: []string{"BT-119"}, Severity: einvoice.SeverityFatal},
	"BR-DE-15":       {Terms: []string{"BT-10"}, Severity: einvoice.SeverityFatal},
	"BR-DE-16":       {Terms: []string{"BG-11", "BT-31", "BT-32"}, Severity: einvoice.SeverityFatal},
	"BR-DE-17":       {Terms: []string{"BT-3"}, Severity: einvoice.SeverityWarning},
	"BR-DE-18":       {Terms: []string{"BT-115", "BT-20"}, Severity: einvoice.SeverityFatal},
	"BR-DE-19":       {Terms: []string{"BT-81", "BT-84"}, Severity: einvoice.SeverityWarning},
	"BR-DE-2":        {Terms: []string{"BG-6"}, Severity: einvoice.SeverityFatal},
	"BR-DE-20":       {Terms: []string{"BT-81", "BT-91"}, Severity: einvoice.SeverityWarning},
	"BR-DE-21":       {Terms: []string{"BT-24"}, Severity: einvoice.SeverityWarning},
	"BR-DE-22":       {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DE-23-a":     {Terms: []string{"BG-17", "BT-81"}, Severity: einvoice.SeverityFatal},
	"BR-DE-23-b":     {Terms: []string{"BG-18", "BG-19", "BT-81"}, Severity: einvoice.SeverityFatal},
	"BR-DE-24-a":     {Terms: []string{"BG-18", "BT-81"}, Severity: einvoice.SeverityFatal},
	"BR-DE-24-b":     {Terms: []string{"BG-17", "BG-19", "BT-81"}, Severity: einvoice.SeverityFatal},
	"BR-DE-25-a":     {Terms: []string{"BG-19", "BT-81"}, Severity: einvoice.SeverityFatal},
	"BR-DE-25-b":     {Terms: []string{"BG-17", "BG-18", "BT-81"}, Severity: einvoice.SeverityFatal},
	"BR-DE-26":       {Terms: []string{"BG-3", "BT-3"}, Severity: einvoice.SeverityWarning},
	"BR-DE-27":       {Terms: []string{"BT-42"}, Severity: einvoice.SeverityWarning},
	"BR-DE-28":       {Terms: []string{"BT-43"}, Severity: einvoice.SeverityWarning},
	"BR-DE-3":        {Terms: []string{"BT-37"}, Severity: einvoice.SeverityFatal},
	"BR-DE-30":       {Terms: []string{"BG-19", "BT-90"}, Severity: einvoice.SeverityFatal},
	"BR-DE-31":       {Terms: []string{"BG-19", "BT-91"}, Severity: einvoice.SeverityFatal},
	"BR-DE-4":        {Terms: []string{"BT-38"}, Severity: einvoice.SeverityFatal},
	"BR-DE-5":        {Terms: []string{"BT-41"}, Severity: einvoice.SeverityFatal},
	"BR-DE-6":        {Terms: []string{"BT-42"}, Severity: einvoice.SeverityFatal},
	"BR-DE-7":        {Terms: []string{"BT-43"}, Severity: einvoice.SeverityFatal},
	"BR-DE-8":        {Terms: []string{"BT-52"}, Severity: einvoice.SeverityFatal},
	"BR-DE-9":        {Terms: []string{"BT-53"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-01":   {Terms: []string{"BT-12"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-02":   {Terms: []string{"BT-17"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-03":   {Terms: []string{"BG-25", "BT-158", "BT-160"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-04":   {Terms: []string{"BT-158"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-05":   {Terms: []string{"BG-32", "BT-160", "BT-161"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-06-a": {Terms: []string{"BT-158", "BT-160"}, Severity: einvoice.SeverityFatal},
	"BR-DE-CVD-06-b": {Terms: []string{"BT-158", "BT-160"}, Severity: einvoice.SeverityFatal},
	"BR-DE-TMP-32":   {Terms: []string{"BG-14", "BG-26", "BT-72"}, Severity: einvoice.SeverityInfo},
	"BR-DEX-01":      {Terms: []string{"BT-125"}, Severity: einvoice.SeverityFatal},
	"BR-DEX-02":      {Terms: []string{"BG-25", "BT-131"}, Severity: einvoice.SeverityWarning},
	"BR-DEX-03":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-04":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-05":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-06":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-07":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-08":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-09":      {Terms: []string{"BT-112", "BT-113", "BT-114", "BT-115"}, Severity: einvoice.SeverityFatal},
	"BR-DEX-10":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-11":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-12":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-13":      {Terms: nil, Severity: einvoice.SeverityFatal},
	"BR-DEX-14":      {Terms: []string{"BT-5"}, Severity: einvoice.SeverityFatal},
	"BR-DEX-15":      {Terms: nil, Severity: einvoice.SeverityWarning},
	"BR-TMP-2":       {Terms: []string{"BT-124"}, Severity: einvoice.SeverityWarning},
	"BR-TMP-3":       {Terms: []string{"BT-149", "BT-150"}, Severity: einvoice.SeverityFatal},
	"BR-TMP-4":       {Terms: []string{"BG-24", "BT-123"}, Severity: einvoice.SeverityWarning},
	"BR-TMP-5":       {Terms: []string{"BG-24", "BT-125"}, Severity: einvoice.SeverityWarning},
	"BR-TMP-CVD-01":  {Terms: []string{"BT-158"}, Severity: einvoice.SeverityFatal},
}
