package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Voranmeldungszeitraum aus der Vorjahressteuer (UST-03 K1).
//
// § 18 Abs. 2 UStG knüpft den Zeitraum an die Steuer des vorangegangenen
// Kalenderjahres: Regelfall ist das Vierteljahr; über 9.000 € ist monatlich
// anzumelden, unter 2.000 € kann das Finanzamt von der Abgabe befreien. Die
// Beträge gelten seit dem 1.1.2025 (vorher 7.500 € und 1.000 €).
//
// Buchfink schlägt den Zeitraum vor und stellt ihn nicht um. Das folgt aus der
// Rechtslage, nicht aus Vorsicht: die Umstellung folgt aus dem Gesetz, aber
// die Befreiung von der Abgabe ist eine Entscheidung des Finanzamts, und ein
// Programm, das den Zeitraum im Januar still ändert, ändert die Fälligkeiten
// eines ganzen Jahres, ohne dass jemand davon erfährt.

const (
	// VatMonthlyThreshold ist die Grenze, über der monatlich anzumelden ist
	// (§ 18 Abs. 2 Satz 2 UStG): mehr als 9.000 € Steuer im Vorjahr.
	VatMonthlyThreshold = accounting.VatMonthlyThreshold
	// VatExemptionThreshold ist die Grenze, unter der das Finanzamt von der
	// Abgabe der Voranmeldungen befreien kann (§ 18 Abs. 2 Satz 3 UStG):
	// nicht mehr als 2.000 € Steuer im Vorjahr.
	VatExemptionThreshold = domain.Cents(200_000)
)

// VatPeriodProposal ist der Vorschlag für den Voranmeldungszeitraum.
type VatPeriodProposal struct {
	// Year ist das Jahr, für das der Zeitraum gilt, BasedOnYear das Jahr, aus
	// dessen Steuer er folgt.
	Year        int `json:"year"`
	BasedOnYear int `json:"basedOnYear"`
	// PriorYearTax ist die Summe der Kennziffer 83 des Vorjahres.
	PriorYearTax domain.Cents `json:"priorYearTax"`
	// Current ist der eingestellte Zeitraum, Proposed der vorgeschlagene.
	Current  domain.VatPeriodType `json:"current"`
	Proposed domain.VatPeriodType `json:"proposed"`
	// Changes meldet, ob der Vorschlag vom eingestellten Zeitraum abweicht.
	Changes bool `json:"changes"`
	// Complete sagt, ob für jeden Zeitraum des Vorjahres eine übermittelte
	// Anmeldung vorliegt. Fehlt eine, ist die Summe zu niedrig — und mit ihr
	// der Vorschlag.
	Complete       bool   `json:"complete"`
	MissingPeriods int    `json:"missingPeriods"`
	Reference      string `json:"reference"`
	Note           string `json:"note"`
}

// SuggestPeriodType leitet den Voranmeldungszeitraum aus der Steuer des
// Vorjahres ab.
func (s *VatReturnService) SuggestPeriodType(ctx context.Context, year int) (*VatPeriodProposal, error) {
	if err := requireYear(year); err != nil {
		return nil, err
	}
	prior := year - 1
	out := &VatPeriodProposal{
		Year: year, BasedOnYear: prior,
		Current:   s.PeriodType(ctx),
		Reference: "§ 18 Abs. 2 UStG",
	}

	founded := s.foundedOn(ctx)
	foundingYear := yearOfDate(founded)
	_, founding := accounting.RecommendedVatPeriod(foundingYear, year, nil)
	// Eine Gründung hat noch keinen festgelegten Zeitraum; gerade dann braucht
	// sie den Vorschlag.
	if !out.Current.Valid() && !(founding && out.Current == domain.VatPeriodUnknown) {
		return nil, nil
	}
	if founding && year == foundingYear {
		return foundingYearProposal(out, foundingYear), nil
	}
	saved, err := s.returnRepo.FindByFiscalYear(ctx, prior)
	if err != nil {
		return nil, fmt.Errorf(
			"die Voranmeldungen des Jahres %d konnten nicht gelesen werden: %w", prior, err)
	}
	// Je Zeitraum die jüngste übermittelte Anmeldung: eine Berichtigung tritt
	// an die Stelle der berichtigten.
	latest := map[string]*domain.VatReturn{}
	for i := range saved {
		r := &saved[i]
		if r.Status != domain.VatReturnSubmitted {
			continue
		}
		if cur, ok := latest[r.PeriodKey]; !ok || r.ID > cur.ID {
			latest[r.PeriodKey] = r
		}
	}
	for _, r := range latest {
		out.PriorYearTax += r.Payable
	}

	// Gezählt wird gegen die Zeiträume, in denen im Vorjahr tatsächlich
	// angemeldet wurde. Wer 2025 monatlich anmeldete, hat zwölf Anmeldungen —
	// gegen vier Quartale gemessen sähe das nach acht überzähligen aus.
	periods := accounting.VatPeriodsOfYear(prior, periodTypeOfKeys(latest))
	for _, p := range periods {
		// Vor der Gründung gab es nichts anzumelden.
		if founding && p.To < founded {
			continue
		}
		if latest[p.Key] == nil {
			out.MissingPeriods++
		}
	}
	out.Complete = len(latest) > 0 && out.MissingPeriods == 0

	if founding {
		return followUpYearProposal(out, foundingYear, founded, len(latest) > 0), nil
	}

	switch {
	case len(latest) == 0:
		out.Proposed = out.Current
		out.Note = fmt.Sprintf(
			"Für %d ist keine übermittelte Voranmeldung erfasst. Ohne die Steuer des Vorjahres "+
				"lässt sich der Zeitraum nicht ableiten.", prior)
		return out, nil
	case out.PriorYearTax > VatMonthlyThreshold:
		out.Proposed = domain.VatPeriodMonth
		out.Note = fmt.Sprintf(
			"Die Steuer des Jahres %d betrug %s € und damit mehr als %s €. Die Voranmeldungen sind "+
				"monatlich abzugeben (§ 18 Abs. 2 Satz 2 UStG).",
			prior, out.PriorYearTax, VatMonthlyThreshold)
	case out.PriorYearTax <= VatExemptionThreshold:
		// Der Vorschlag bleibt das Vierteljahr: die Befreiung nach § 18 Abs. 2
		// Satz 3 UStG spricht das Finanzamt aus, sie tritt nicht von selbst
		// ein. Ein Vorschlag „jährlich" ließe den Anwender die Anmeldungen
		// einstellen, bevor er den Bescheid hat.
		out.Proposed = domain.VatPeriodQuarter
		out.Note = fmt.Sprintf(
			"Die Steuer des Jahres %d betrug %s € und damit nicht mehr als %s €. Das Finanzamt kann "+
				"von der Abgabe der Voranmeldungen befreien (§ 18 Abs. 2 Satz 3 UStG) — die Befreiung "+
				"spricht es aus, sie tritt nicht von selbst ein. Bis dahin bleibt es beim Vierteljahr.",
			prior, out.PriorYearTax, VatExemptionThreshold)
	default:
		out.Proposed = domain.VatPeriodQuarter
		out.Note = fmt.Sprintf(
			"Die Steuer des Jahres %d betrug %s €. Damit bleibt es beim Vierteljahr als "+
				"Voranmeldungszeitraum (§ 18 Abs. 2 Satz 1 UStG).", prior, out.PriorYearTax)
	}
	out.Changes = out.Proposed != out.Current
	if !out.Complete {
		out.Note += fmt.Sprintf(
			" Für %d Zeiträume des Jahres %d liegt keine übermittelte Anmeldung vor; die Summe ist "+
				"deshalb möglicherweise zu niedrig.", out.MissingPeriods, prior)
	}
	return out, nil
}

// foundingYearProposal ist der Vorschlag im Jahr der Gründung. Ohne die
// voraussichtliche Steuer, die Buchfink nicht kennt, bleibt es in den Jahren
// mit ausgesetzter Monatspflicht beim eingestellten Zeitraum.
func foundingYearProposal(out *VatPeriodProposal, foundingYear int) *VatPeriodProposal {
	advice, _ := accounting.RecommendedVatPeriod(foundingYear, out.Year, nil)
	out.BasedOnYear = out.Year
	out.Proposed = out.Current
	if advice.Period != "" {
		out.Proposed = advice.Period
	}
	out.Note = advice.Reason
	out.Changes = out.Proposed != out.Current
	return out
}

// followUpYearProposal ist der Vorschlag im Jahr nach der Gründung.
//
// § 18 Abs. 2 Satz 6 UStG verlangt, die Steuer des Gründungsjahres in eine
// Jahressteuer umzurechnen, und sagt nicht, wie. Buchfink rechnet mit vollen
// Monaten ab dem Gründungsmonat und nennt das in der Begründung.
func followUpYearProposal(out *VatPeriodProposal, foundingYear int, founded string, known bool) *VatPeriodProposal {
	var annual *domain.Cents
	months := 13 - int(monthOf(founded))
	if known && months >= 1 && months <= 12 {
		v := domain.MulRound(out.PriorYearTax, 12, int64(months))
		annual = &v
	}
	advice, _ := accounting.RecommendedVatPeriod(foundingYear, out.Year, annual)
	out.Proposed = out.Current
	if advice.Period != "" {
		out.Proposed = advice.Period
	}
	out.Note = advice.Reason
	if annual != nil && months < 12 {
		out.Note += fmt.Sprintf(" Umgerechnet hat Buchfink %s € aus %d Monaten ab dem Gründungsmonat.",
			out.PriorYearTax, months)
	}
	if known && !out.Complete {
		out.Note += fmt.Sprintf(" Für %d Zeiträume seit der Gründung liegt keine übermittelte Anmeldung "+
			"vor; die Summe ist deshalb möglicherweise zu niedrig.", out.MissingPeriods)
	}
	out.Changes = out.Proposed != out.Current
	return out
}

// foundedOn ist der Gründungstag aus den Unternehmensdaten oder leer.
func (s *VatReturnService) foundedOn(ctx context.Context) string {
	if s.settingsRepo == nil {
		return ""
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.FoundedOn
}

// periodTypeOfKeys erkennt am Zeitraumschlüssel, in welchem Rhythmus im Vorjahr
// angemeldet wurde. Ohne Anmeldungen gilt das Vierteljahr — der Regelfall des
// § 18 Abs. 2 Satz 1 UStG.
func periodTypeOfKeys(latest map[string]*domain.VatReturn) domain.VatPeriodType {
	for key := range latest {
		p, err := accounting.ParseVatPeriodKey(key)
		if err != nil {
			continue
		}
		return p.Type
	}
	return domain.VatPeriodQuarter
}
