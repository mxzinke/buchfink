package domain

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Die Anlegerstellung folgt aus der Rechtsform, wo diese sie hergibt. Das ist
// der Regelfall — und er kommt ohne eine zweite Frage aus.
func TestInvestorTypeFollowsTheLegalForm(t *testing.T) {
	cases := []struct {
		form string
		want InvestorType
	}{
		{"GmbH", InvestorCorporate},
		{"UG (haftungsbeschränkt)", InvestorCorporate},
		{"AG", InvestorCorporate},
		{"eG", InvestorCorporate},
		{"Einzelunternehmen", InvestorIndividualBusiness},
		{"Eingetragener Kaufmann (e. K.)", InvestorIndividualBusiness},
		{"Freiberufliche Praxis", InvestorIndividualBusiness},
		// Aus einer Personengesellschaft folgt sie nicht: § 20 Abs. 3a InvStG
		// stellt auf den Gesellschafter ab.
		{"GbR", InvestorUnknown},
		{"OHG", InvestorUnknown},
		{"KG", InvestorUnknown},
		{"GmbH & Co. KG", InvestorUnknown},
		{"Partnerschaftsgesellschaft", InvestorUnknown},
		{"Sonstige", InvestorUnknown},
		// Eine Rechtsform, die nicht im Katalog steht, entscheidet nichts.
		{"Limited", InvestorUnknown},
		{"", InvestorUnknown},
	}
	for _, c := range cases {
		settings := &CompanySettings{LegalForm: c.form}
		got, reason := settings.InvestorTypeOrDerived()
		if got != c.want {
			t.Errorf("%q ergibt %q — erwartet %q", c.form, got, c.want)
		}
		if reason == "" {
			t.Errorf("%q kommt ohne Begründung", c.form)
		}
	}
}

// Wo die Rechtsform nicht entscheidet — und wo sie es falsch täte, etwa bei
// einem Versicherungsunternehmen nach § 20 Abs. 1 Satz 4 InvStG — schlägt die
// ausdrückliche Festlegung sie.
func TestExplicitChoiceBeatsTheLegalForm(t *testing.T) {
	settings := &CompanySettings{LegalForm: "GmbH & Co. KG", InvestorOverride: InvestorIndividualBusiness}
	got, reason := settings.InvestorTypeOrDerived()
	if got != InvestorIndividualBusiness {
		t.Errorf("die Festlegung wurde nicht übernommen: %q", got)
	}
	if reason == "" {
		t.Error("auch die Festlegung braucht ihre Begründung")
	}

	// Eine AG ist eine Körperschaft — ein Versicherungsunternehmen fällt
	// trotzdem auf den Grundsatz zurück.
	insurer := &CompanySettings{LegalForm: "AG", InvestorOverride: InvestorBasic}
	if got, _ := insurer.InvestorTypeOrDerived(); got != InvestorBasic {
		t.Errorf("die Ausnahme des § 20 Abs. 1 Satz 4 InvStG wurde überschrieben: %q", got)
	}
}

// Jeder Katalogeintrag trägt eine Begründung, und jede abgeleitete
// Anlegerstellung ist eine gültige.
func TestLegalFormCatalogIsComplete(t *testing.T) {
	for _, form := range LegalFormCatalog() {
		if form.Name == "" || form.Note == "" {
			t.Errorf("Katalogeintrag ohne Namen oder Begründung: %+v", form)
		}
		if form.Investor != InvestorUnknown && !form.Investor.Valid() {
			t.Errorf("%q leitet auf eine unbekannte Anlegerstellung %q ab", form.Name, form.Investor)
		}
	}
}

// Die Ersteinrichtung bietet Rechtsformen zur Auswahl an, bevor ein Mandant
// existiert — ihre Liste steht deshalb im Frontend. Weicht eine Schreibweise
// vom Katalog ab, fiele sie später aus der Ableitung, ohne dass es auffiele.
func TestSetupAssistantOffersOnlyCatalogForms(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(
		"..", "..", "frontend", "src", "components", "SetupAssistantScreen.tsx"))
	if err != nil {
		t.Skipf("die Ersteinrichtung ist von hier nicht lesbar: %v", err)
	}
	block := regexp.MustCompile(`(?s)const LEGAL_FORMS = \[(.*?)\];`).FindSubmatch(source)
	if block == nil {
		t.Fatal("in SetupAssistantScreen.tsx steht keine Liste LEGAL_FORMS mehr")
	}
	offered := regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(string(block[1]), -1)
	if len(offered) == 0 {
		t.Fatal("die Liste der Rechtsformen ist leer")
	}
	for _, match := range offered {
		if _, ok := LookupLegalForm(match[1]); !ok {
			t.Errorf("die Ersteinrichtung bietet %q an, der Katalog kennt die Schreibweise nicht",
				match[1])
		}
	}
}
