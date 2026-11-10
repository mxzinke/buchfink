package accounting

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Altersstruktur und Restlaufzeiten ------------------------------------

func agingItem(kind domain.ContactType, due string, open domain.Cents) domain.OpenItem {
	return domain.OpenItem{
		ContactType: kind, DueDate: due, OpenAmount: open,
		DocumentDate: "2026-01-01", GrossAmount: open,
	}
}

func TestAgeOpenItemsSortsIntoBands(t *testing.T) {
	cutoff := "2026-06-30"
	items := []domain.OpenItem{
		agingItem(domain.ContactTypeCustomer, "2026-07-15", 10000), // nicht fällig
		agingItem(domain.ContactTypeCustomer, "2026-06-30", 20000), // am Stichtag fällig, noch nicht überfällig
		agingItem(domain.ContactTypeCustomer, "2026-06-01", 30000), // 29 Tage
		agingItem(domain.ContactTypeCustomer, "2026-05-01", 40000), // 60 Tage
		agingItem(domain.ContactTypeCustomer, "2026-04-01", 50000), // 90 Tage
		agingItem(domain.ContactTypeCustomer, "2026-01-01", 60000), // 180 Tage
		agingItem(domain.ContactTypeCustomer, "", 70000),           // ohne Fälligkeit
		agingItem(domain.ContactTypeVendor, "2026-08-01", 15000),   // Verbindlichkeit
	}

	result := AgeOpenItems(items, cutoff)
	if len(result.Sides) != 2 {
		t.Fatalf("erwartet zwei Seiten, bekommen %d", len(result.Sides))
	}

	receivables := result.Sides[0]
	if receivables.Items != 7 {
		t.Errorf("Forderungen: %d Posten, erwartet 7", receivables.Items)
	}
	if receivables.Total != 280000 {
		t.Errorf("Forderungen: Summe %d, erwartet 280000", receivables.Total)
	}

	want := map[domain.AgingBucketKey]domain.Cents{
		domain.AgingNotDue:      30000, // 10000 + 20000
		domain.AgingUpTo30:      30000,
		domain.AgingUpTo60:      40000,
		domain.AgingUpTo90:      50000,
		domain.AgingOver90:      60000,
		domain.AgingWithoutDate: 70000,
	}
	for _, bucket := range receivables.Buckets {
		if bucket.Amount != want[bucket.Key] {
			t.Errorf("Band %q: %d, erwartet %d", bucket.Key, bucket.Amount, want[bucket.Key])
		}
	}

	payables := result.Sides[1]
	if payables.Total != 15000 || payables.Items != 1 {
		t.Errorf("Verbindlichkeiten: %d in %d Posten, erwartet 15000 in 1", payables.Total, payables.Items)
	}
}

func TestAgeOpenItemsSplitsRemainingTerms(t *testing.T) {
	cutoff := "2026-12-31"
	items := []domain.OpenItem{
		agingItem(domain.ContactTypeVendor, "2027-06-30", 10000), // bis 1 Jahr
		agingItem(domain.ContactTypeVendor, "2029-01-01", 20000), // über 1 bis 5 Jahre
		agingItem(domain.ContactTypeVendor, "2032-06-30", 30000), // über 5 Jahre
		agingItem(domain.ContactTypeVendor, "2026-01-01", 40000), // überfällig: sofort fällig
	}

	result := AgeOpenItems(items, cutoff)
	want := map[domain.MaturityBandKey]domain.Cents{
		domain.MaturityUpToOneYear:   50000, // 10000 + der überfällige Posten
		domain.MaturityOneToFive:     20000,
		domain.MaturityOverFiveYears: 30000,
		domain.MaturityUndated:       0,
	}
	for _, band := range result.Sides[1].Maturities {
		if band.Amount != want[band.Key] {
			t.Errorf("Restlaufzeit %q: %d, erwartet %d", band.Key, band.Amount, want[band.Key])
		}
	}
}

func TestAgeOpenItemsIgnoresSettledItems(t *testing.T) {
	// Ein ausgeglichener Posten steht in der Liste, ist aber nicht überfällig:
	// er ist bezahlt. Zählte er mit, meldete die Altersstruktur eine Forderung,
	// die es nicht mehr gibt.
	items := []domain.OpenItem{
		agingItem(domain.ContactTypeCustomer, "2026-01-01", 0),
		agingItem(domain.ContactTypeCustomer, "2026-01-01", 5000),
	}
	result := AgeOpenItems(items, "2026-06-30")
	if result.Sides[0].Items != 1 || result.Sides[0].Total != 5000 {
		t.Errorf("nur der offene Posten zählt, bekommen %d Posten über %d",
			result.Sides[0].Items, result.Sides[0].Total)
	}
}
