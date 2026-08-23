package invoice

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Ein Umsatz zum Nullsteuersatz nach § 12 Abs. 3 UStG ist steuerpflichtig zum
// Satz null, nicht steuerfrei. UNTDID 5305 hat dafür den Code "Z". Mit "S" und
// 0,00 % wäre die Rechnung nach BR-S-05 formal falsch — und die Selbstprüfung
// im Generator ließe sie gar nicht erst entstehen.
func TestZeroRatedInvoiceCarriesCategoryZ(t *testing.T) {
	inv, seller, buyer := sampleInvoice()
	inv.TaxTreatment = domain.TaxTreatmentZeroRated
	for i := range inv.Items {
		inv.Items[i].TaxRate = domain.TaxRateNone
	}
	inv.Recalculate()

	xml, err := GenerateZUGFeRDXML(inv, seller, buyer)
	if err != nil {
		t.Fatalf("die Rechnung ließ sich nicht erzeugen: %v", err)
	}
	if !strings.Contains(xml, "<ram:CategoryCode>Z</ram:CategoryCode>") {
		t.Error("der Datensatz trägt nicht den Kategoriecode Z")
	}
	if strings.Contains(xml, "<ram:CategoryCode>S</ram:CategoryCode>") {
		t.Error("der Datensatz trägt weiterhin den Kategoriecode S")
	}
}
