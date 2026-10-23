package domain

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Buchfink führt GmbH und UG (haftungsbeschränkt) und keine andere Rechtsform.
func TestLegalFormCatalogHoldsGmbHAndUG(t *testing.T) {
	catalog := LegalFormCatalog()
	if len(catalog) != 2 {
		t.Fatalf("erwartet zwei Rechtsformen, erhalten %d: %+v", len(catalog), catalog)
	}
	for _, name := range []string{LegalFormGmbH, LegalFormUG} {
		if !IsSupportedLegalForm(name) {
			t.Errorf("%q fehlt im Katalog", name)
		}
	}
	for _, name := range []string{"", "AG", "GbR", "Einzelunternehmen", "GmbH & Co. KG", "Limited"} {
		if IsSupportedLegalForm(name) {
			t.Errorf("%q darf nicht im Katalog stehen", name)
		}
	}
	for _, form := range catalog {
		if form.Note == "" || !form.Investor.Valid() {
			t.Errorf("Katalogeintrag ohne Begründung oder Anlegerstellung: %+v", form)
		}
	}
}

// Die Anlegerstellung folgt aus der Rechtsform: beide sind Körperschaften.
func TestInvestorTypeFollowsTheLegalForm(t *testing.T) {
	for _, form := range []string{LegalFormGmbH, LegalFormUG} {
		settings := &CompanySettings{LegalForm: form}
		got, reason := settings.InvestorTypeOrDerived()
		if got != InvestorCorporate {
			t.Errorf("%q ergibt %q — erwartet %q", form, got, InvestorCorporate)
		}
		if reason == "" {
			t.Errorf("%q kommt ohne Begründung", form)
		}
	}
	empty := &CompanySettings{}
	if got, reason := empty.InvestorTypeOrDerived(); got != InvestorUnknown || reason == "" {
		t.Errorf("ohne Rechtsform erwartet keine Anlegerstellung mit Begründung, erhalten %q (%q)", got, reason)
	}
}

// Eine GmbH ist eine Körperschaft — ein Versicherungsunternehmen fällt
// trotzdem auf den Grundsatz zurück (§ 20 Abs. 1 Satz 4 InvStG).
func TestExplicitChoiceBeatsTheLegalForm(t *testing.T) {
	insurer := &CompanySettings{LegalForm: LegalFormGmbH, InvestorOverride: InvestorBasic}
	got, reason := insurer.InvestorTypeOrDerived()
	if got != InvestorBasic {
		t.Errorf("die Ausnahme des § 20 Abs. 1 Satz 4 InvStG wurde überschrieben: %q", got)
	}
	if reason == "" {
		t.Error("auch die Festlegung braucht ihre Begründung")
	}
}

// Die Ersteinrichtung bietet Rechtsformen zur Auswahl an, bevor ein Mandant
// existiert — ihre Liste steht deshalb im Frontend (utils/companyLabels.ts). Weicht eine Schreibweise
// vom Katalog ab, ließe sich der Mandant nicht speichern.
func TestSetupAssistantOffersOnlyCatalogForms(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(
		"..", "..", "frontend", "src", "utils", "companyLabels.ts"))
	if err != nil {
		t.Skipf("die Ersteinrichtung ist von hier nicht lesbar: %v", err)
	}
	block := regexp.MustCompile(`(?s)const LEGAL_FORMS = \[(.*?)\];`).FindSubmatch(source)
	if block == nil {
		t.Fatal("in companyLabels.ts steht keine Liste LEGAL_FORMS mehr")
	}
	offered := regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(string(block[1]), -1)
	if len(offered) == 0 {
		t.Fatal("die Liste der Rechtsformen ist leer")
	}
	for _, match := range offered {
		if !IsSupportedLegalForm(match[1]) {
			t.Errorf("die Ersteinrichtung bietet %q an, der Katalog kennt die Schreibweise nicht",
				match[1])
		}
	}
}
