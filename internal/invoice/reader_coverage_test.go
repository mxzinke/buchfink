package invoice

import (
	"testing"

	"github.com/buchfink/buchfink/internal/einvoice"
)

// Nur eine Rechnung nach reinem EN 16931 ist vollständig geprüft. XRechnung und
// Peppol BIS Billing 3.0 haben eigene Regeln (BR-DE-*, PEPPOL-EN16931-R*), die
// Buchfink nicht prüft.
func TestCoverageIsPartialWhereNationalRulesApply(t *testing.T) {
	for profile, want := range map[string]string{
		einvoice.ProfileEN16931:    "full",
		einvoice.ProfileXRechnung:  "partial",
		einvoice.ProfilePeppolBIS3: "partial",
	} {
		if got := coverage(&einvoice.Invoice{SpecificationID: profile}); got != want {
			t.Errorf("Prüfumfang für %s = %q, erwartet %q", profile, got, want)
		}
	}
}
