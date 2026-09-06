package repository

import (
	"context"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Jahresdatensatz gehört zum Jahr. Bleibt er nach dem Löschen stehen, führt
// die Jahresliste ein Geschäftsjahr mit dem Abschlussstand „festgestellt“
// weiter, dessen Buchungen, Belege und Abschlüsse es nicht mehr gibt — eine
// Aussage über einen Abschluss ohne Grundlage.
func TestDeleteFiscalYearAlsoRemovesTheYearRecord(t *testing.T) {
	db := welle6DB(t)
	retention := NewRetentionRepository(db)
	ctx := context.Background()

	for _, year := range []domain.FiscalYear{
		{Year: 2015, StartDate: "2015-01-01", EndDate: "2015-12-31",
			Status: domain.FiscalYearAdopted, AverageEmployees: 4},
		{Year: 2026, StartDate: "2026-01-01", EndDate: "2026-12-31",
			Status: domain.FiscalYearOpen},
	} {
		if err := db.Create(&year).Error; err != nil {
			t.Fatalf("Geschäftsjahr %d anlegen: %v", year.Year, err)
		}
	}

	if _, _, err := retention.DeleteFiscalYear(ctx, 2015); err != nil {
		t.Fatalf("das Löschen ist fehlgeschlagen: %v", err)
	}

	var gone int64
	if err := db.Model(&domain.FiscalYear{}).Where("year = ?", 2015).Count(&gone).Error; err != nil {
		t.Fatalf("Geschäftsjahre zählen: %v", err)
	}
	if gone != 0 {
		t.Error("der Jahresdatensatz des gelöschten Geschäftsjahres steht noch in der Datei")
	}

	var kept domain.FiscalYear
	if err := db.First(&kept, "year = ?", 2026).Error; err != nil {
		t.Fatalf("das laufende Geschäftsjahr wurde mitgelöscht: %v", err)
	}
}

// Die Fristenübersicht fragt nach Jahren mit aufzubewahrenden Objekten und
// nicht nach Jahren mit Buchungen: ein Jahr, in dem nur Belege abgelegt wurden,
// hat dieselbe Aufbewahrungsfrist und muss in der Übersicht stehen.
func TestFiscalYearsWithObjectsSeesMoreThanTheJournal(t *testing.T) {
	db := welle6DB(t)
	journal := NewJournalRepository(db)
	retention := NewRetentionRepository(db)
	ctx := context.Background()

	welle6Entry(t, journal, "2026-01-10")

	// 2015 hat nur einen Beleg, 2018 nur eine Voranmeldung.
	receipt := &domain.Receipt{
		ReceiptNumber: "ER-2015-0001", FiscalYear: 2015,
		Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindLetter,
		Status: domain.ReceiptStatusFiled, ReceivedAt: "2015-03-01",
	}
	if err := db.Create(receipt).Error; err != nil {
		t.Fatalf("Beleg anlegen: %v", err)
	}
	vat := &domain.VatReturn{
		FiscalYear: 2018, PeriodKey: "2018-01", PeriodType: domain.VatPeriodMonth,
		PeriodFrom: "2018-01-01", PeriodTo: "2018-01-31",
	}
	if err := db.Create(vat).Error; err != nil {
		t.Fatalf("Voranmeldung anlegen: %v", err)
	}

	years, err := retention.FiscalYearsWithObjects(ctx)
	if err != nil {
		t.Fatalf("die Geschäftsjahre ließen sich nicht bestimmen: %v", err)
	}
	want := []int{2015, 2018, 2026}
	if len(years) != len(want) {
		t.Fatalf("%v gemeldet, erwartet %v", years, want)
	}
	for i, year := range want {
		if years[i] != year {
			t.Fatalf("%v gemeldet, erwartet %v (aufsteigend sortiert)", years, want)
		}
	}
}
