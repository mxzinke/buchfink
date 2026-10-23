package service

import (
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// expenseLines bildet Aufwand und nicht abziehbare Anteile auf getrennten
// Konten ab (§ 4 Abs. 5 EStG), damit die Steuerbilanz sie unterscheiden kann.
func (s *PostingService) expenseLines(
	r resolvedPosition, nonDeductibleTax domain.Cents,
	treatment domain.TaxTreatment, documentDate string,
) ([]domain.JournalLine, accounting.DeductibleQuota, domain.Cents, error) {
	p := r.position

	// Was von der Vorsteuer nicht abziehbar ist, gehört zum Aufwand
	// (§ 9b Abs. 1 EStG). Der Betrag kommt von außen: er wird je
	// Steuersatzgruppe gerechnet und nicht je Position — siehe
	// nonDeductibleTaxByPosition.
	taxableNet := taxableNetOf(r)
	// Der Anteil steht nur an der Zeile, wo er nicht der volle ist. Der
	// Ausschluss des § 15 Abs. 1a UStG bekommt seinen eigenen Wert: null
	// Promille abziehbar ist etwas anderes als „kein Vorsteuerschlüssel", und
	// die Zahl null bedeutet an dieser Stelle das Zweite.
	share := 0
	switch {
	case r.permille == 0:
		share = domain.InputTaxExcluded
	case r.permille < 1000:
		share = int(r.permille)
	}
	text := appendText(p.Text, r.note)

	if r.account != "" {
		// Ein unmittelbar gewähltes Konto durchläuft weiter die Prüfungen des
		// Journals; es ist der Notausgang für Fälle, die der Gruppenkatalog nicht
		// abdeckt — und für das Geschenk über der Freigrenze, dessen Konto der
		// Belegweg selbst gesetzt hat.
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: r.account, Amount: p.Net + nonDeductibleTax,
			InputTaxShare: share, Text: text,
		}}, "", taxableNet, nil
	}
	if !r.hasGroup {
		return nil, "", 0, fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group := r.group

	account := group.ResolveAccount(treatment, p.TaxRate)
	if group.NonDeductibleAccount == "" || group.DeductibleQuota == "" {
		return []domain.JournalLine{{
			Side: domain.SideDebit, Account: account, Amount: p.Net + nonDeductibleTax,
			InputTaxShare: share, Text: text,
		}}, "", taxableNet, nil
	}

	params, err := accounting.TaxParametersFor(documentDate)
	if err != nil {
		return nil, "", 0, err
	}
	permille := deductiblePermille(group.DeductibleQuota, params)
	if permille <= 0 || permille >= 1000 {
		return nil, "", 0, fmt.Errorf("für %q ist kein abziehbarer Anteil hinterlegt", group.Label)
	}

	base := p.Net + nonDeductibleTax
	deductible := domain.MulRound(base, permille, 1000)
	// Der Rest ergibt sich als Differenz und wird nicht ein zweites Mal
	// gerundet — sonst summierten sich die beiden Zeilen an einem Cent vorbei.
	nonDeductible := base - deductible

	lines := []domain.JournalLine{
		{Side: domain.SideDebit, Account: account, Amount: deductible,
			InputTaxShare: share, Text: text},
	}
	if nonDeductible > 0 {
		lines = append(lines, domain.JournalLine{
			Side: domain.SideDebit, Account: group.NonDeductibleAccount,
			Amount: nonDeductible, InputTaxShare: share, Text: text,
		})
	}
	return lines, group.DeductibleQuota, taxableNet, nil
}

// preparedPosition ist eine Belegposition, nachdem Umrechnung, Konto und
// abziehbarer Vorsteueranteil feststehen und bevor ihre Zeilen entstehen.
type preparedPosition struct {
	// position hat den Nettobetrag bereits in Euro; foreignNet den Betrag der
	// Fremdwährung, aus dem er entstanden ist.
	position   ReceiptPosition
	foreignNet domain.Cents
	// advance kennzeichnet die Position eines Anzahlungsbelegs. Sie geht nicht
	// durch die fachlichen Gruppen und hat keinen geteilten Vorsteuerabzug.
	advance  bool
	resolved resolvedPosition
}

// taxableNetOf liefert den Teil des Nettobetrags, auf den Vorsteuer gezogen
// wird — bei ungeteiltem Abzug der ganze.
func taxableNetOf(r resolvedPosition) domain.Cents {
	if r.permille >= 1000 {
		return r.position.Net
	}
	return domain.MulRound(r.position.Net, r.permille, 1000)
}

// nonDeductibleTaxByPosition verteilt die nicht abziehbare Vorsteuer des
// geteilten Abzugs auf die Positionen.
//
// Gerundet wird einmal je Steuersatzgruppe — genau wie die Steuerzeile, die aus
// derselben Gruppe entsteht (§ 9b Abs. 1 EStG macht den nicht abziehbaren Teil
// zum Aufwand, sagt aber nichts über die Reihenfolge der Rundung). Wird je
// Position gerundet, gehen Sollzeilen und Steuerzeile an unterschiedlichen Cents
// vorbei: die Summe der Buchung weicht dann um ein bis zwei Cent vom
// Bruttobetrag der Rechnung ab, die Gegenzeile gleicht das still aus, und die
// Verbindlichkeit gegenüber dem Lieferanten stimmt mit seiner Rechnung nicht
// mehr überein. Der Rest der Gruppe liegt auf ihrer letzten geteilten Position —
// dieselbe Regel wie beim Ausgleich der Umrechnung.
func nonDeductibleTaxByPosition(prepared []preparedPosition, grossTax map[domain.TaxRate]domain.Cents) []domain.Cents {
	out := make([]domain.Cents, len(prepared))
	netByRate := map[domain.TaxRate]domain.Cents{}
	taxableByRate := map[domain.TaxRate]domain.Cents{}
	assignedByRate := map[domain.TaxRate]domain.Cents{}
	lastSplit := map[domain.TaxRate]int{}

	for i, pp := range prepared {
		if pp.advance || pp.position.NotTaxableReason != "" {
			continue
		}
		rate := pp.position.TaxRate
		taxable := taxableNetOf(pp.resolved)
		netByRate[rate] += pp.position.Net
		taxableByRate[rate] += taxable
		if pp.resolved.permille >= 1000 {
			continue
		}
		// Die Aufteilung je Position ist der Vorschlag; sie hält die Verteilung
		// nah an den Beträgen, aus denen sie stammt.
		out[i] = rate.Tax(pp.position.Net) - rate.Tax(taxable)
		assignedByRate[rate] += out[i]
		lastSplit[rate] = i
	}

	for rate, idx := range lastSplit {
		// Was auf die Gruppe insgesamt entfällt: die Differenz der beiden einmal
		// gerundeten Steuerbeträge. Der Unterschied zur Summe der Vorschläge
		// liegt auf der letzten geteilten Position.
		owed := rate.Tax(netByRate[rate])
		if tax, ok := grossTax[rate]; ok {
			owed = tax
		}
		total := owed - rate.Tax(taxableByRate[rate])
		out[idx] += total - assignedByRate[rate]
	}
	return out
}

// appendText hängt einen Zusatz an den Buchungstext.
func appendText(text, addition string) string {
	switch {
	case addition == "":
		return text
	case text == "":
		return addition
	default:
		return text + " — " + addition
	}
}

func deductiblePermille(quota accounting.DeductibleQuota, params accounting.TaxParameters) int64 {
	switch quota {
	case accounting.QuotaEntertainment:
		return params.EntertainmentDeductiblePermille
	default:
		return 0
	}
}
