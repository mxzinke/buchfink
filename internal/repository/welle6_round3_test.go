package repository

import (
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Altbestand bekommt seine Aufbewahrungsfrist nachgetragen.
//
// Klasse und Löschdatum werden seit Welle 6 beim Ablegen gespeichert. Belege
// und Anlagendokumente aus der Zeit davor trügen ohne diesen Lauf leere
// Fristen, obwohl beide aus Belegart und Entstehungsjahr genauso zu rechnen
// sind wie bei jedem neuen Objekt — und der Bericht über abgelaufene Objekte,
// der über die Löschung entscheidet, sähe sie überhaupt nicht.
func TestBackfillFillsRetentionOfLegacyObjects(t *testing.T) {
	db := welle6DB(t)

	// Zwei Belege und ein Anlagendokument, wie sie vor Welle 6 entstanden sind:
	// ohne Klasse und ohne Fristende. Geschrieben wird über die Spalten, weil
	// genau dieser Zustand nachgestellt werden soll.
	legacy := []domain.Receipt{
		{
			FiscalYear: 2026, ReceiptNumber: "ER-2026-9001",
			Direction: domain.DirectionIncoming, Status: domain.ReceiptStatusSealed,
			Kind: domain.ReceiptKindInvoice, ReceiptHash: "a",
			DocumentDate: "2026-03-01",
		},
		{
			// Ohne Belegdatum zählt das Geschäftsjahr.
			FiscalYear: 2024, ReceiptNumber: "ER-2024-9002",
			Direction: domain.DirectionIncoming, Status: domain.ReceiptStatusSealed,
			Kind: domain.ReceiptKindInvoice, ReceiptHash: "b",
		},
	}
	for i := range legacy {
		if err := db.Create(&legacy[i]).Error; err != nil {
			t.Fatalf("Altbeleg anlegen: %v", err)
		}
		if err := db.Model(&domain.Receipt{}).Where("id = ?", legacy[i].ID).
			Updates(map[string]any{"retention_class": "", "retention_until": ""}).Error; err != nil {
			t.Fatalf("Altzustand herstellen: %v", err)
		}
	}
	document := domain.AssetDocument{
		AssetID: 1, Kind: domain.AssetDocContract, FileName: "vertrag.pdf",
		MimeType: "application/pdf", Size: 12, SHA256: "c", StoredPath: "dokumente/c",
		DocumentDate: "2026-01-10",
	}
	if err := db.Create(&document).Error; err != nil {
		t.Fatalf("Altdokument anlegen: %v", err)
	}

	if err := BackfillRetention(db); err != nil {
		t.Fatalf("der Nachtrag ist fehlgeschlagen: %v", err)
	}

	// Rechnung 2026: Buchungsbeleg, acht Jahre ab Schluss 2026.
	var first domain.Receipt
	if err := db.First(&first, legacy[0].ID).Error; err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if first.RetentionClass != domain.RetentionClassVouchers || first.RetentionUntil != "2034-12-31" {
		t.Errorf("Beleg 2026: Klasse %q bis %q, erwartet %q bis 2034-12-31",
			first.RetentionClass, first.RetentionUntil, domain.RetentionClassVouchers)
	}

	// Rechnung 2024: dieselbe Klasse, aber acht Jahre ab Schluss 2024 — die
	// Verkürzung des Vierten Bürokratieentlastungsgesetzes wirkt auf die am
	// 1.1.2025 noch laufende Frist.
	var second domain.Receipt
	if err := db.First(&second, legacy[1].ID).Error; err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if second.RetentionUntil != "2032-12-31" {
		t.Errorf("Beleg 2024: Fristende %q, erwartet 2032-12-31", second.RetentionUntil)
	}

	var stored domain.AssetDocument
	if err := db.First(&stored, document.ID).Error; err != nil {
		t.Fatalf("Dokument lesen: %v", err)
	}
	if stored.RetentionClass != domain.RetentionClassBooks || stored.RetentionUntil != "2036-12-31" {
		t.Errorf("Anlagendokument: Klasse %q bis %q, erwartet %q bis 2036-12-31 (Organisationsunterlage)",
			stored.RetentionClass, stored.RetentionUntil, domain.RetentionClassBooks)
	}

	// Ein zweiter Lauf lässt die gesetzten Fristen unangetastet: er sucht nur
	// die leeren. Eine einmal geltende Frist ist eine Tatsache über das Objekt
	// und darf sich nicht mit dem Programmstand ändern.
	if err := db.Model(&domain.Receipt{}).Where("id = ?", legacy[0].ID).
		Update("retention_until", "2099-12-31").Error; err != nil {
		t.Fatalf("Frist setzen: %v", err)
	}
	if err := BackfillRetention(db); err != nil {
		t.Fatalf("der zweite Nachtrag ist fehlgeschlagen: %v", err)
	}
	if err := db.First(&first, legacy[0].ID).Error; err != nil {
		t.Fatalf("Beleg lesen: %v", err)
	}
	if first.RetentionUntil != "2099-12-31" {
		t.Errorf("der zweite Lauf hat eine vorhandene Frist überschrieben: %q", first.RetentionUntil)
	}
}
