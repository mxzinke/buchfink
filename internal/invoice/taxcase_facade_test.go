package invoice

import (
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

func TestTaxCaseFacadeMatchesInvoiceCategories(t *testing.T) {
	for _, treatment := range domain.AllTaxTreatments() {
		if got, want := accounting.TaxCaseFor(treatment).InvoiceCategory(), vatCategoryCode(treatment); got != want {
			t.Errorf("Kategorie %s: Fassade=%q, Rechnung=%q", treatment, got, want)
		}
	}
}
