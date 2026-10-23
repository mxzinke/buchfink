package accounting

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Die unfertigen Leistungen zum Stichtag.
//
// Ein Projekt, das am Stichtag begonnen, aber weder abgenommen noch als
// Teilleistung abgerechnet ist, hat noch keinen Erlös (§ 252 Abs. 1 Nr. 4 HGB).
// Seine bisherigen Kosten werden zu Herstellungskosten aktiviert (§ 255 Abs. 2
// HGB), höchstens mit dem Wert, der sich aus dem Auftrag ergibt (§ 253 Abs. 4
// Satz 2 HGB: verlustfreie Bewertung). Der Gewinnanteil gehört nie dazu.

// Konten der unfertigen Leistungen (SKR04).
const (
	AccountUnfertigeLeistungen          = "1080" // Unfertige Leistungen (Bestand)
	AccountBestandsveraenderungUL       = "4815" // Bestandsveränderungen unfertige Leistungen
	AccountAnzahlungenVonVorraetenOffen = "1190" // Erhaltene Anzahlungen, von Vorräten offen abgesetzt
)

// advanceAccounts sind die Konten erhaltener Anzahlungen, die Buchfink führt
// (AdvanceAccountFor).
var advanceAccounts = map[string]bool{
	AccountErhalteneAnzahlungen:   true,
	AccountErhalteneAnzahlungen7:  true,
	AccountErhalteneAnzahlungen19: true,
}

// WorkInProgressProject ist ein am Stichtag laufendes Projekt.
type WorkInProgressProject struct {
	Name      string `json:"name"`
	ContactID uint   `json:"contactId,omitempty"`
	// DirectCosts sind die Einzelkosten: Fremdleistungen, Material und die
	// Löhne der Projektstunden; Overheads die angemessenen Gemeinkosten
	// einschließlich anteiliger Abschreibung. Beide sind anzusetzen (§ 255
	// Abs. 2 Satz 2 HGB).
	DirectCosts domain.Cents `json:"directCosts"`
	Overheads   domain.Cents `json:"overheads"`
	// AdministrativeCosts sind Kosten der allgemeinen Verwaltung und sozialer
	// Leistungen; sie dürfen einbezogen werden (Satz 3), wenn
	// IncludeAdministrative gesetzt ist. Vertriebs- und Forschungskosten
	// dürfen es nicht (Satz 4).
	AdministrativeCosts   domain.Cents `json:"administrativeCosts"`
	IncludeAdministrative bool         `json:"includeAdministrative"`
	// ContractValue ist der vereinbarte Nettoerlös, CostsToComplete die
	// erwarteten Kosten bis zur Fertigstellung. Mit ihnen wird verlustfrei
	// bewertet; ohne Auftragswert nicht.
	ContractValue   domain.Cents `json:"contractValue"`
	CostsToComplete domain.Cents `json:"costsToComplete"`
	// AdvancesReceived sind die erhaltenen Anzahlungen netto, AdvanceAccount
	// ihr Konto (3250, 3260 oder 3272; ohne Angabe 3272).
	AdvancesReceived domain.Cents `json:"advancesReceived"`
	AdvanceAccount   string       `json:"advanceAccount,omitempty"`
	Note             string       `json:"note,omitempty"`
}

// WorkInProgressValuation ist die Bewertung eines Projekts.
type WorkInProgressValuation struct {
	Name           string       `json:"name"`
	ContactID      uint         `json:"contactId,omitempty"`
	ProductionCost domain.Cents `json:"productionCost"`
	// NetRealizable ist Auftragswert abzüglich der Kosten bis zur
	// Fertigstellung; null ohne Auftragswert.
	NetRealizable domain.Cents `json:"netRealizable"`
	Value         domain.Cents `json:"value"`
	WriteDown     domain.Cents `json:"writeDown"`
	// ExpectedLoss ist der Verlust über den Bestand hinaus; für ihn ist eine
	// Rückstellung für drohende Verluste zu prüfen (§ 249 Abs. 1 Satz 1 HGB).
	ExpectedLoss domain.Cents `json:"expectedLoss"`
	// AdvanceOffset ist die von den Vorräten offen abgesetzte Anzahlung.
	AdvanceOffset  domain.Cents `json:"advanceOffset"`
	AdvanceAccount string       `json:"advanceAccount,omitempty"`
}

// WorkInProgressResult ist der Bestand der unfertigen Leistungen.
type WorkInProgressResult struct {
	Projects       []WorkInProgressValuation `json:"projects"`
	Total          domain.Cents              `json:"total"`
	ExpectedLosses domain.Cents              `json:"expectedLosses"`
	// AdvanceOffsets sind die offen abgesetzten Anzahlungen je Konto.
	AdvanceOffsets map[string]domain.Cents `json:"advanceOffsets"`
}

// CalculateWorkInProgress bewertet die Projekte. deductAdvances setzt die
// erhaltenen Anzahlungen von den Vorräten offen ab, höchstens bis zum Wert des
// Projekts (§ 268 Abs. 5 Satz 2 HGB); sonst bleiben sie unter den
// Verbindlichkeiten.
func CalculateWorkInProgress(projects []WorkInProgressProject, deductAdvances bool) (*WorkInProgressResult, error) {
	out := &WorkInProgressResult{
		Projects: make([]WorkInProgressValuation, 0, len(projects)), AdvanceOffsets: map[string]domain.Cents{},
	}
	for i, p := range projects {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return nil, fmt.Errorf("das %d. Projekt hat keinen Namen", i+1)
		}
		for _, v := range []domain.Cents{p.DirectCosts, p.Overheads, p.AdministrativeCosts,
			p.ContractValue, p.CostsToComplete, p.AdvancesReceived} {
			if v < 0 {
				return nil, fmt.Errorf("%s: Kosten, Auftragswert und Anzahlungen können nicht negativ sein", name)
			}
		}
		row := WorkInProgressValuation{Name: name, ContactID: p.ContactID}
		row.ProductionCost = p.DirectCosts + p.Overheads
		if p.IncludeAdministrative {
			row.ProductionCost += p.AdministrativeCosts
		}
		row.Value = row.ProductionCost
		if p.ContractValue > 0 {
			row.NetRealizable = p.ContractValue - p.CostsToComplete
			if row.NetRealizable < row.Value {
				row.Value = maxCents(row.NetRealizable, 0)
				row.WriteDown = row.ProductionCost - row.Value
			}
			if row.NetRealizable < 0 {
				row.ExpectedLoss = -row.NetRealizable
			}
		}
		if deductAdvances && p.AdvancesReceived > 0 {
			account := p.AdvanceAccount
			if account == "" {
				account = AccountErhalteneAnzahlungen19
			}
			if !advanceAccounts[account] {
				return nil, fmt.Errorf("%s: %s ist kein Konto erhaltener Anzahlungen (3250, 3260, 3272)",
					name, account)
			}
			row.AdvanceOffset = minCents(p.AdvancesReceived, row.Value)
			row.AdvanceAccount = account
			out.AdvanceOffsets[account] += row.AdvanceOffset
		}
		out.Total += row.Value
		out.ExpectedLosses += row.ExpectedLoss
		out.Projects = append(out.Projects, row)
	}
	sort.SliceStable(out.Projects, func(i, j int) bool { return out.Projects[i].Name < out.Projects[j].Name })
	return out, nil
}
