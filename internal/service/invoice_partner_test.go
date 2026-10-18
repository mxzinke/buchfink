package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

func TestInvoicePartnerValidation(t *testing.T) {
	for _, row := range []struct {
		treatment               domain.TaxTreatment
		allowed, vatID, confirm bool
	}{
		{domain.TaxTreatmentDomestic, true, false, false},
		{domain.TaxTreatmentReverseChargeSupply, true, true, true},
		{domain.TaxTreatmentZeroRated, true, false, false},
		{domain.TaxTreatmentExempt, true, false, false},
		{domain.TaxTreatmentNotTaxable, true, false, false},
		{domain.TaxTreatmentReverseChargeEU, false, false, false},
		{domain.TaxTreatmentReverseCharge, false, false, false},
		{domain.TaxTreatmentIntraCommunityAcquisition, false, false, false},
		{"unbekannt", false, false, false},
		{"", false, false, false},
	} {
		for _, country := range []string{"DE", "FR", "US", ""} {
			for _, vatID := range []string{"", "FR12345678901"} {
				t.Run(fmt.Sprintf("%s/%s/%s", row.treatment, country, vatID), func(t *testing.T) {
					inv := &domain.Invoice{TaxTreatment: row.treatment}
					contact := &domain.Contact{Name: "Partner", CountryCode: country, VatID: vatID}
					var want string
					if !row.allowed {
						want = fmt.Sprintf("den Steuerfall %q gibt es für Ausgangsrechnungen nicht", row.treatment)
					} else if row.vatID && vatID == "" {
						want = "für eine Rechnung nach § 13b UStG braucht Partner eine USt-IdNr."
					}
					s := &InvoiceService{}
					assertPartnerError(t, s.validateTaxTreatment(inv, contact), want)
					if s.needsVatIDConfirmation(inv, contact) {
						t.Fatal("ohne Bestätigungsdienst darf keine Abfrage erforderlich sein")
					}
					confirmer := &recordingVatIDConfirmer{}
					s.SetVatIDConfirmer(confirmer)
					wantConfirm := row.confirm && country == "FR"
					if got := s.needsVatIDConfirmation(inv, contact); got != wantConfirm {
						t.Fatalf("Bestätigung erforderlich: %t, erwartet %t", got, wantConfirm)
					}
					inv.VatIDOverrideReason = "Bestätigung liegt schriftlich vor"
					if err := s.ensureVatIDConfirmed(context.Background(), inv, contact); err != nil {
						t.Fatal(err)
					}
					if (confirmer.contact != nil) != wantConfirm || wantConfirm && (confirmer.contact != contact || confirmer.reason != inv.VatIDOverrideReason) {
						t.Fatalf("Bestätigungsaufruf: %+v", confirmer)
					}
				})
			}
		}
	}
}

type recordingVatIDConfirmer struct {
	contact *domain.Contact
	reason  string
}

func (c *recordingVatIDConfirmer) EnsureConfirmed(_ context.Context, contact *domain.Contact, reason string) error {
	c.contact, c.reason = contact, reason
	return nil
}

func (*recordingVatIDConfirmer) NeedsCheck(context.Context, *domain.Contact) (bool, error) {
	return true, nil
}

func TestIncomingPartnerValidation(t *testing.T) {
	for _, row := range []struct {
		treatment            domain.TaxTreatment
		country, vatID, want string
	}{
		{domain.TaxTreatmentIntraCommunityAcquisition, "DE", "", "ein innergemeinschaftlicher Erwerb setzt einen Lieferanten in einem anderen EU-Land voraus, Partner ist in \"DE\" erfasst"},
		{domain.TaxTreatmentIntraCommunityAcquisition, "US", "US123", "ein innergemeinschaftlicher Erwerb setzt einen Lieferanten in einem anderen EU-Land voraus, Partner ist in \"US\" erfasst"},
		{domain.TaxTreatmentIntraCommunityAcquisition, "FR", "", "für einen innergemeinschaftlichen Erwerb braucht Partner eine USt-IdNr."},
		{domain.TaxTreatmentIntraCommunityAcquisition, "FR", "FR123", ""},
		{domain.TaxTreatmentReverseChargeEU, "DE", "", "§ 13b Abs. 1 UStG gilt nur für die sonstige Leistung eines Unternehmers aus einem anderen EU-Staat, Partner ist in \"DE\" erfasst. Für einen Lieferanten aus dem Drittland gilt § 13b Abs. 2 UStG"},
		{domain.TaxTreatmentReverseChargeEU, "US", "US123", "§ 13b Abs. 1 UStG gilt nur für die sonstige Leistung eines Unternehmers aus einem anderen EU-Staat, Partner ist in \"US\" erfasst. Für einen Lieferanten aus dem Drittland gilt § 13b Abs. 2 UStG"},
		{domain.TaxTreatmentReverseChargeEU, "FR", "", "für § 13b UStG braucht der ausländische Lieferant Partner eine USt-IdNr., die seine Unternehmereigenschaft belegt"},
		{domain.TaxTreatmentReverseChargeEU, "FR", "FR123", ""},
		{domain.TaxTreatmentReverseCharge, "FR", "", "für § 13b UStG braucht der ausländische Lieferant Partner eine USt-IdNr., die seine Unternehmereigenschaft belegt"},
		{domain.TaxTreatmentReverseCharge, "FR", "FR123", ""},
		{domain.TaxTreatmentReverseCharge, "DE", "", ""},
		{domain.TaxTreatmentReverseCharge, "US", "", ""},
		{domain.TaxTreatmentReverseCharge, "", "", ""},
		{domain.TaxTreatmentDomestic, "FR", "", ""},
		{domain.TaxTreatmentReverseChargeSupply, "FR", "", ""},
		{domain.TaxTreatmentZeroRated, "FR", "", ""},
		{domain.TaxTreatmentExempt, "FR", "", ""},
		{domain.TaxTreatmentNotTaxable, "FR", "", ""},
		{"unbekannt", "FR", "", ""},
		{"", "FR", "", ""},
	} {
		t.Run(fmt.Sprintf("%s/%s/%s", row.treatment, row.country, row.vatID), func(t *testing.T) {
			contact := &domain.Contact{Name: "Partner", CountryCode: row.country, VatID: row.vatID}
			assertPartnerError(t, validateIncomingTreatment(row.treatment, contact), row.want)
		})
	}
}

func assertPartnerError(t *testing.T, err error, want string) {
	t.Helper()
	got := ""
	if err != nil {
		got = err.Error()
	}
	if got != want {
		t.Fatalf("Partnerprüfung: %q, erwartet %q", got, want)
	}
}
