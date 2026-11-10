package domain

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/buchfink/buchfink/internal/calendar"
)

// Die Jahreserklärungen, die ein Steuerberater übernehmen kann. Die Werte
// entsprechen der Art einer Erklärung im MCP-Werkzeug `erklaerung`.
const (
	DeclarationCorporateTax = "kst"
	DeclarationTradeTax     = "gewst"
	DeclarationAnnualVat    = "ust_jahr"
	DeclarationEBilanz      = "ebilanz"
)

// AllDeclarations sind die Erklärungen in der Reihenfolge der Auswahl.
func AllDeclarations() []string {
	return []string{DeclarationCorporateTax, DeclarationTradeTax, DeclarationAnnualVat, DeclarationEBilanz}
}

// TaxAdvisor beschreibt die Beauftragung eines Steuerberaters mit der
// Erstellung von Jahreserklärungen (§ 149 Abs. 3 AO).
type TaxAdvisor struct {
	Engaged bool `json:"engaged"`
	// SinceYear ist der erste Besteuerungszeitraum, den der Berater erstellt;
	// 0 heißt: alle.
	SinceYear int `json:"sinceYear"`
	// Declarations sind die übernommenen Erklärungen; leer heißt: alle.
	Declarations []string `json:"declarations"`
}

// Prepares meldet, ob der Berater die Erklärung für das Jahr erstellt.
func (a TaxAdvisor) Prepares(declaration string, year int) bool {
	if !a.Engaged || (a.SinceYear > 0 && year < a.SinceYear) {
		return false
	}
	if len(a.Declarations) == 0 {
		return true
	}
	for _, d := range a.Declarations {
		if d == declaration {
			return true
		}
	}
	return false
}

// Validate prüft die übernommenen Erklärungen.
func (a TaxAdvisor) Validate() error {
	for _, d := range a.Declarations {
		known := false
		for _, k := range AllDeclarations() {
			known = known || d == k
		}
		if !known {
			return fmt.Errorf("unbekannte Erklärung %q für den Steuerberater (erwartet %s)",
				d, strings.Join(AllDeclarations(), ", "))
		}
	}
	if a.SinceYear < 0 {
		return fmt.Errorf("das Jahr der Beauftragung kann nicht negativ sein")
	}
	return nil
}

// StreetLine ist Straße und Hausnummer in einer Zeile.
func (s *CompanySettings) StreetLine() string {
	return strings.TrimSpace(strings.TrimSpace(s.Street) + " " + strings.TrimSpace(s.HouseNumber))
}

var houseNumberPattern = regexp.MustCompile(`^(.*\D)\s+(\d+\s?[a-zA-Z]?(?:\s?[-/]\s?\d+\s?[a-zA-Z]?)?)$`)

// SplitStreet trennt eine Anschriftzeile in Straße und Hausnummer:
// „Hauptstraße 12a" → „Hauptstraße", „12a". Ein Postfach oder eine Zeile ohne
// Hausnummer am Ende bleibt ganz Straße.
func SplitStreet(line string) (street, houseNumber string) {
	line = strings.Join(strings.Fields(line), " ")
	if strings.HasPrefix(strings.ToLower(line), "postfach") {
		return line, ""
	}
	m := houseNumberPattern.FindStringSubmatch(line)
	if m == nil {
		return line, ""
	}
	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
}

// NormalizeAddress trennt die Hausnummer von der Straße, wenn sie noch in
// der Straße steht und das Feld Hausnummer leer ist.
func (s *CompanySettings) NormalizeAddress() {
	s.Street = strings.TrimSpace(s.Street)
	s.HouseNumber = strings.TrimSpace(s.HouseNumber)
	if s.HouseNumber == "" {
		s.Street, s.HouseNumber = SplitStreet(s.Street)
	}
}

// ResolveTaxNumberElster setzt TaxNumberElster aus Steuernummer und Land;
// passt beides nicht zusammen, bleibt das Feld leer.
func (s *CompanySettings) ResolveTaxNumberElster() {
	s.TaxNumberElster, _ = ElsterTaxNumber(s.TaxNumber, s.FederalState)
}

// validateObligationProfile prüft Land, Steuernummer und Steuerberater.
func (s *CompanySettings) validateObligationProfile() error {
	state := strings.ToUpper(strings.TrimSpace(s.FederalState))
	if state != "" && !calendar.ValidState(state) {
		return fmt.Errorf("unbekanntes Bundesland %q (erwartet ein Kürzel wie BY oder NW)", s.FederalState)
	}
	if _, err := ElsterTaxNumber(s.TaxNumber, state); err != nil {
		return fmt.Errorf("Steuernummer: %w", err)
	}
	return s.TaxAdvisor.Validate()
}

// Unsupported nennt, warum Buchfink für diese Gesellschaft keinen Abschluss
// und keine Steuerrückstellung rechnet; nil, wenn nichts dagegen spricht.
func (s *CompanySettings) Unsupported() error {
	switch {
	case s.NonProfit:
		return fmt.Errorf("die Gesellschaft ist als gemeinnützig gekennzeichnet. Steuerbefreiung " +
			"(§ 5 Abs. 1 Nr. 9 KStG), Sphärentrennung und Mittelverwendung (§§ 51 ff. AO) bildet " +
			"Buchfink nicht ab; Abschluss und Steuerrückstellung gehören zu einem Steuerberater")
	case s.InLiquidation:
		return fmt.Errorf("die Gesellschaft ist als in Liquidation gekennzeichnet. " +
			"Liquidationseröffnungsbilanz und Abwicklungsabschlüsse (§ 71 GmbHG) und der " +
			"Besteuerungszeitraum der Abwicklung (§ 11 KStG) bildet Buchfink nicht ab; Abschluss und " +
			"Steuerrückstellung gehören zu einem Steuerberater")
	case s.FiscalUnity:
		return fmt.Errorf("die Gesellschaft ist als Organgesellschaft gekennzeichnet. Ihr Einkommen " +
			"wird dem Organträger zugerechnet (§§ 14, 17 KStG); Gewinnabführung und Steuerumlage bildet " +
			"Buchfink nicht ab. Abschluss und Steuerrückstellung gehören zu einem Steuerberater")
	}
	return nil
}
