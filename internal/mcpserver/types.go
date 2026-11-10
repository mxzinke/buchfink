package mcpserver

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/google/jsonschema-go/jsonschema"
)

// Die Werte, die Werkzeuge austauschen: Beträge als Zeichenkette in Euro mit
// Punkt, Datum als JJJJ-MM-TT, Buchungsgruppen, Steuerfälle und Sätze als
// feste Listen. Das Schema zeigt dem Agenten die gültigen Werte.

// Betrag ist ein Betrag in Euro mit Punkt und höchstens zwei Nachkommastellen,
// etwa "119.00" oder "-35.7".
type Betrag string

// Datum ist ein Kalendertag JJJJ-MM-TT.
type Datum string

// Konto ist eine Kontonummer des SKR04 oder ein Personenkonto.
type Konto string

// Buchungsgruppe ist eine fachliche Gruppe für Eingangsbelege.
type Buchungsgruppe string

// Erloesgruppe ist eine fachliche Gruppe für Ausgangsrechnungen.
type Erloesgruppe string

// Steuerfall ist der umsatzsteuerliche Fall eines Belegs.
type Steuerfall string

// Satz ist ein Umsatzsteuersatz in Prozent.
type Satz string

// Zinssatz ist ein Zinssatz in Prozent im Jahr mit Punkt, etwa "3.5".
type Zinssatz string

var (
	betragPattern   = regexp.MustCompile(`^-?[0-9]{1,12}(\.[0-9]{1,2})?$`)
	datumPattern    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	zinssatzPattern = regexp.MustCompile(`^[0-9]{1,3}(\.[0-9]{1,2})?$`)
)

func schemaOptions() *jsonschema.ForOptions {
	return &jsonschema.ForOptions{TypeSchemas: typeSchemas}
}

// outputSchemaOptions: Ausgaben sind Zeichenketten, wo Eingaben ein Muster
// oder eine Werteliste haben.
func outputSchemaOptions() *jsonschema.ForOptions {
	plain := map[reflect.Type]*jsonschema.Schema{}
	for t := range typeSchemas {
		plain[t] = &jsonschema.Schema{Type: "string"}
	}
	return &jsonschema.ForOptions{TypeSchemas: plain}
}

var typeSchemas = map[reflect.Type]*jsonschema.Schema{
	reflect.TypeFor[Betrag](): {Type: "string", Pattern: betragPattern.String(), Description: `Euro, z. B. "119.00"`},
	reflect.TypeFor[Datum]():  {Type: "string", Pattern: datumPattern.String()},
	reflect.TypeFor[Konto]():  {Type: "string", Pattern: `^[0-9]{4,5}$`, Description: "SKR04-Konto oder Personenkonto"},
	reflect.TypeFor[Buchungsgruppe](): {Type: "string", Enum: groupKeys(domain.DirectionIncoming),
		Description: "Buchungsgruppe für Eingangsbelege; Bedeutung über konten mit art=buchungsgruppen"},
	reflect.TypeFor[Erloesgruppe](): {Type: "string", Enum: groupKeys(domain.DirectionOutgoing),
		Description: "Erlösgruppe für Ausgangsrechnungen"},
	reflect.TypeFor[Steuerfall](): {Type: "string", Enum: treatmentKeys(),
		Description: "Steuerfall; Bedeutung über konten mit art=steuerfaelle. Regelfall: domestic"},
	reflect.TypeFor[Satz](): {Type: "string", Enum: []any{"19", "7", "0"}, Description: "Umsatzsteuersatz in Prozent"},
	reflect.TypeFor[Zinssatz](): {Type: "string", Pattern: zinssatzPattern.String(),
		Description: `Prozent im Jahr, z. B. "3.5"`},
}

func groupKeys(dir domain.Direction) []any {
	var out []any
	for _, g := range accounting.PostingGroups(dir) {
		out = append(out, g.Key)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].(string) < out[j].(string) })
	return out
}

func treatmentKeys() []any {
	var out []any
	for _, t := range domain.AllTaxTreatments() {
		out = append(out, string(t))
	}
	return out
}

// inputError ist ein Eingabefehler mit dem Feld, an dem er hängt.
func inputError(field, format string, args ...any) *ToolError {
	return &ToolError{Code: CodeInvalidInput, Nachricht: field + ": " + fmt.Sprintf(format, args...)}
}

// cents liest einen Betrag; leer ist null.
func (b Betrag) cents(field string) (domain.Cents, error) {
	if b == "" {
		return 0, nil
	}
	if !betragPattern.MatchString(string(b)) {
		return 0, inputError(field, "%q ist kein Betrag (erwartet z. B. \"119.00\")", b)
	}
	c, err := domain.ParseCents(string(b))
	if err != nil {
		return 0, inputError(field, "%v", err)
	}
	return c, nil
}

// required liest einen Pflichtbetrag.
func (b Betrag) required(field string) (domain.Cents, error) {
	if b == "" {
		return 0, inputError(field, "der Betrag fehlt")
	}
	return b.cents(field)
}

func betrag(c domain.Cents) Betrag { return Betrag(c.Decimal()) }

func optBetrag(c domain.Cents) *Betrag {
	b := betrag(c)
	return &b
}

// basisPoints liest einen Zinssatz in Hundertstel Prozent: "3.5" ist 350.
func (z Zinssatz) basisPoints(field string) (int, error) {
	if !zinssatzPattern.MatchString(string(z)) {
		return 0, inputError(field, "%q ist kein Zinssatz (erwartet Prozent, z. B. \"3.5\")", z)
	}
	c, err := domain.ParseCents(string(z))
	if err != nil {
		return 0, inputError(field, "%v", err)
	}
	return int(c), nil
}

// zinssatz zeigt Hundertstel Prozent als Zinssatz.
func zinssatz(bp int) string { return fmt.Sprintf("%d.%02d", bp/100, bp%100) }

func (d Datum) check(field string) (string, error) {
	if d == "" {
		return "", nil
	}
	if err := domain.CheckDate(field, string(d)); err != nil {
		return "", inputError(field, "%v", err)
	}
	return string(d), nil
}

func (s Satz) rate() domain.TaxRate {
	switch s {
	case "19":
		return domain.TaxRateStandard
	case "7":
		return domain.TaxRateReduced
	}
	return domain.TaxRateNone
}

func satzOf(r domain.TaxRate) Satz {
	switch r {
	case domain.TaxRateStandard:
		return "19"
	case domain.TaxRateReduced:
		return "7"
	case domain.TaxRateNone:
		return "0"
	}
	return Satz(fmt.Sprintf("%d.%02d", r/100, r%100))
}

// yearOr ist das angegebene Geschäftsjahr, sonst das des heutigen Tages.
func yearOr(ctx context.Context, c *Call, year int) (int, error) {
	if year > 0 {
		return year, nil
	}
	return c.Tenant.CurrentFiscalYear(ctx)
}

// orEmpty macht aus nil eine leere Liste: Clients prüfen die Ausgabe gegen
// das Schema, und dort ist eine Liste kein null.
func orEmpty[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}

// ptr gibt einen Zeiger auf v, nil für den Nullwert.
func ptrIf[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
