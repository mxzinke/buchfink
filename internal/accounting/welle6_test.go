package accounting

import (
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// --- Aufbewahrungsfristen -------------------------------------------------

func TestRetentionForClassifiesAndDates(t *testing.T) {
	cases := []struct {
		name         string
		kind         domain.RetentionKind
		year         int
		wantClass    domain.RetentionClass
		wantYears    int
		wantEnd      string
		wantDeletion string
	}{
		{
			// Das Vierte Bürokratieentlastungsgesetz: acht Jahre für den Beleg
			// aus 2025, gerechnet ab dem Schluss des Entstehungsjahres.
			name: "Beleg 2025", kind: domain.RetentionKindReceiptInvoice, year: 2025,
			wantClass: domain.RetentionClassVouchers, wantYears: 8,
			wantEnd: "2033-12-31", wantDeletion: "2034-01-01",
		},
		{
			// Das Journal ist ein Handelsbuch: zehn Jahre.
			name: "Journal 2025", kind: domain.RetentionKindJournal, year: 2025,
			wantClass: domain.RetentionClassBooks, wantYears: 10,
			wantEnd: "2035-12-31", wantDeletion: "2036-01-01",
		},
		{
			// Die Verkürzung wirkt auf laufende Fristen: der Beleg aus 2024 war
			// am 1.1.2025 noch aufzubewahren und bekommt deshalb acht Jahre.
			name: "Beleg 2024", kind: domain.RetentionKindReceiptInvoice, year: 2024,
			wantClass: domain.RetentionClassVouchers, wantYears: 8,
			wantEnd: "2032-12-31", wantDeletion: "2033-01-01",
		},
		{
			// Wessen Zehnjahresfrist vor dem 1.1.2025 abgelaufen war, bleibt bei
			// zehn Jahren: rückwirkend verkürzen ändert an einer beendeten
			// Aufbewahrung nichts, und die Auskunft soll die damals geltende
			// Frist nennen.
			name: "Beleg 2013", kind: domain.RetentionKindReceiptInvoice, year: 2013,
			wantClass: domain.RetentionClassVouchers, wantYears: 10,
			wantEnd: "2023-12-31", wantDeletion: "2024-01-01",
		},
		{
			name: "Handelsbrief 2025", kind: domain.RetentionKindReceiptLetter, year: 2025,
			wantClass: domain.RetentionClassLetters, wantYears: 6,
			wantEnd: "2031-12-31", wantDeletion: "2032-01-01",
		},
		{
			// Der Kontoauszug ist Buchungsbeleg.
			name: "Kontoauszug 2025", kind: domain.RetentionKindReceiptStatement, year: 2025,
			wantClass: domain.RetentionClassVouchers, wantYears: 8,
			wantEnd: "2033-12-31", wantDeletion: "2034-01-01",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := RetentionFor(tc.kind, tc.year)
			if info.Class != tc.wantClass {
				t.Errorf("Klasse: %q, erwartet %q", info.Class, tc.wantClass)
			}
			if info.Years != tc.wantYears {
				t.Errorf("Frist: %d Jahre, erwartet %d", info.Years, tc.wantYears)
			}
			if info.RetentionEnd != tc.wantEnd {
				t.Errorf("Fristende: %q, erwartet %q", info.RetentionEnd, tc.wantEnd)
			}
			if info.EarliestDeletion != tc.wantDeletion {
				t.Errorf("frühestes Löschdatum: %q, erwartet %q", info.EarliestDeletion, tc.wantDeletion)
			}
			if info.LegalBasis == "" {
				t.Error("zu jeder Frist gehört ihre Rechtsgrundlage")
			}
		})
	}
}

func TestRetentionEndAndEarliestDeletionAreOneDayApart(t *testing.T) {
	info := RetentionFor(domain.RetentionKindReceiptInvoice, 2025)

	// Am letzten Tag der Frist ist noch aufzubewahren.
	if info.RetentionEnd != "2033-12-31" {
		t.Errorf("letzter Aufbewahrungstag %s, erwartet 2033-12-31", info.RetentionEnd)
	}
	if info.EarliestDeletion != "2034-01-01" {
		t.Errorf("frühestes Löschdatum %s, erwartet 2034-01-01", info.EarliestDeletion)
	}
}

func TestDeletionConceptCoversEveryClass(t *testing.T) {
	concept := DeletionConcept()
	if len(concept) == 0 {
		t.Fatal("das Löschkonzept ist leer")
	}
	seen := map[domain.RetentionClass]bool{}
	for _, row := range concept {
		if row.Category == "" || row.LegalBasis == "" || row.Years == 0 {
			t.Errorf("unvollständige Zeile des Löschkonzepts: %+v", row)
		}
		seen[row.Class] = true
	}
	for _, class := range domain.AllRetentionClasses() {
		if !seen[class] {
			t.Errorf("das Löschkonzept nennt die Klasse %q nicht", class)
		}
	}
}

func TestBlockedContactAnswerNamesTheNorms(t *testing.T) {
	answer := BlockedContactAnswer("Meier GmbH")
	for _, needle := range []string{"Meier GmbH", "§ 257 HGB", "§ 147 AO", "Art. 17 Abs. 3 Buchst. b DSGVO"} {
		if !strings.Contains(answer, needle) {
			t.Errorf("die Antwort an die betroffene Person nennt %q nicht:\n%s", needle, answer)
		}
	}
}

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

// --- Kette des Änderungsprotokolls ----------------------------------------

func auditEntry(id uint, action domain.AuditAction, details string) domain.AuditLogEntry {
	return domain.AuditLogEntry{
		ID:         id,
		Timestamp:  time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(id) * time.Minute),
		Action:     action,
		EntityType: "CONTACT",
		EntityID:   "7",
		Details:    details,
		Actor:      "buchhalterin@arbeitsplatz",
		Channel:    "ui",
		AppVersion: "1.2.3",
	}
}

// chained verkettet die Einträge so, wie das Repository es tut.
func chained(entries []domain.AuditLogEntry) []domain.AuditLogEntry {
	chain := NewAuditChain()
	prev := domain.GenesisHash
	for i := range entries {
		entries[i].PreviousHash = prev
		entries[i].EntryHash = chain.CalculateHash(&entries[i], prev)
		prev = entries[i].EntryHash
	}
	return entries
}

func TestAuditChainValidAfterSeveralEntries(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	result := NewAuditChain().Verify(entries)
	if !result.IsValid {
		t.Fatalf("die unveränderte Kette muss gültig sein: %s", result.Message)
	}
	if result.CheckedEntries != 3 {
		t.Errorf("%d geprüfte Einträge, erwartet 3", result.CheckedEntries)
	}
	if len(result.Breaks) != 0 {
		t.Errorf("keine Brüche erwartet, bekommen %d", len(result.Breaks))
	}
}

func TestAuditChainDetectsAlteredEntry(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "Betrag 100 € auf 200 € geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	// Wer eine Buchung ändert, ändert danach die Protokollzeile. Genau das
	// findet die Kette.
	entries[1].Details = "Betrag unverändert"

	result := NewAuditChain().Verify(entries)
	if result.IsValid {
		t.Fatal("ein nachträglich geänderter Protokolleintrag muss auffallen")
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != 2 {
		t.Errorf("der erste Bruch muss Eintrag 2 sein, bekommen %v", result.FirstBrokenID)
	}
	if len(result.Breaks) != 1 || result.Breaks[0].Reason != domain.IntegrityBreakContent {
		t.Errorf("erwartet genau einen Inhaltsbruch, bekommen %+v", result.Breaks)
	}
}

func TestAuditChainDetectsRemovedEntry(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	// Der mittlere Eintrag wird entfernt — der Fall, für den es die Kette gibt.
	without := []domain.AuditLogEntry{entries[0], entries[2]}

	result := NewAuditChain().Verify(without)
	if result.IsValid {
		t.Fatal("ein entfernter Protokolleintrag muss die Kette brechen")
	}
	if len(result.Breaks) == 0 || result.Breaks[0].Reason != domain.IntegrityBreakLinkage {
		t.Errorf("erwartet einen Verkettungsbruch, bekommen %+v", result.Breaks)
	}
}

func TestAuditChainReportsEntryWithoutHash(t *testing.T) {
	entries := chained([]domain.AuditLogEntry{
		auditEntry(1, domain.AuditActionCreate, "angelegt"),
		auditEntry(2, domain.AuditActionUpdate, "geändert"),
		auditEntry(3, domain.AuditActionExport, "exportiert"),
	})

	// Wer den Eigenhash leert, nimmt den Eintrag nicht aus der Kette.
	entries[1].EntryHash = ""

	result := NewAuditChain().Verify(entries)
	if result.IsValid {
		t.Fatal("ein Eintrag ohne Eigenhash muss als Bruch auffallen")
	}
	if result.FirstBrokenID == nil || *result.FirstBrokenID != 2 {
		t.Errorf("der erste Bruch muss Eintrag 2 sein, bekommen %v", result.FirstBrokenID)
	}
	if result.CheckedEntries != 3 {
		t.Errorf("%d geprüfte Einträge, erwartet 3", result.CheckedEntries)
	}
}

func TestAuditChainCoversActorAndVersion(t *testing.T) {
	// Bearbeiterkennung und Programmfassung sind der Kern der Nachweispflicht.
	// Wären sie nicht gedeckt, ließen sie sich nachträglich austauschen.
	base := chained([]domain.AuditLogEntry{auditEntry(1, domain.AuditActionUpdate, "geändert")})

	tampered := base[0]
	tampered.Actor = "jemand.anderes@rechner"
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("eine geänderte Bearbeiterkennung muss den Hash ändern")
	}

	tampered = base[0]
	tampered.Channel = "agent:claude-desktop"
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("ein geänderter Erfassungsweg muss den Hash ändern")
	}

	tampered = base[0]
	tampered.AppVersion = "9.9.9"
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("eine geänderte Programmfassung muss den Hash ändern")
	}

	tampered = base[0]
	tampered.Before = `{"name":"anders"}`
	if NewAuditChain().CalculateHash(&tampered, tampered.PreviousHash) == base[0].EntryHash {
		t.Error("ein geändertes Vorher muss den Hash ändern")
	}
}

// --- Kanonische Form der Buchung ------------------------------------------

func sampleEntry() *domain.JournalEntry {
	return &domain.JournalEntry{
		EntryNumber:     "2025-000001",
		FiscalYear:      2025,
		BookingDate:     "2025-03-01",
		DocumentDate:    "2025-03-01",
		ServiceDateFrom: "2025-03-01",
		ServiceDateTo:   "2025-03-01",
		Description:     "Bürobedarf",
		Source:          domain.EntrySourceManual,
		Kind:            domain.EntryKindNormal,
		Currency:        "EUR",
		AppVersion:      "1.2.3",
		Actor:           "buchhalterin@arbeitsplatz",
		Channel:         "ui",
		CreatedAt:       time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC),
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
}

// Es gibt genau eine kanonische Form: jedes Feld steht in jeder Buchung, auch
// leer oder null.
func TestCanonicalizeWritesEveryFieldForEveryEntry(t *testing.T) {
	entry := sampleEntry()
	entry.AppVersion = ""
	entry.Actor = ""
	entry.Channel = ""
	form := string(canonicalize(entry, domain.GenesisHash))

	for _, field := range []string{
		"app_version:0:\n", "actor:0:\n", "channel:0:\n", "legacy_ref:0:\n", "due_date:0:\n",
		"line_input_tax_share:1:0\n", "line_foreign_amount:1:0\n",
		"entertainment:1:0\n", "gifts:1:0\n",
	} {
		if !strings.Contains(form, "\n"+field) {
			t.Errorf("die kanonische Form enthält %q nicht", strings.TrimSuffix(field, "\n"))
		}
	}
}

func TestCanonicalizeCoversEveryField(t *testing.T) {
	chain := NewHashChain()
	base := chain.CalculateHash(sampleEntry(), domain.GenesisHash)

	contact := uint(7)
	changes := map[string]func(e *domain.JournalEntry){
		"Programmfassung":    func(e *domain.JournalEntry) { e.AppVersion = "9.9.9" },
		"Bearbeiter":         func(e *domain.JournalEntry) { e.Actor = "jemand.anderes@rechner" },
		"Erfassungsweg":      func(e *domain.JournalEntry) { e.Channel = "agent:claude-desktop" },
		"Herkunftskennung":   func(e *domain.JournalEntry) { e.LegacyRef = "ALT-4711" },
		"Fälligkeit":         func(e *domain.JournalEntry) { e.DueDate = "2025-04-01" },
		"Vorsteuerschlüssel": func(e *domain.JournalEntry) { e.Lines[0].InputTaxShare = 500 },
		"Fremdbetrag":        func(e *domain.JournalEntry) { e.Lines[0].ForeignAmount = 10870 },
		"Geschenk": func(e *domain.JournalEntry) {
			e.Gifts = []domain.GiftRecord{{
				FiscalYear: 2025, Date: "2025-03-01", RecipientContactID: &contact,
				RecipientName: "Kunde GmbH", NetAmount: 3000, Account: "6610",
			}}
		},
	}
	for name, change := range changes {
		entry := sampleEntry()
		change(entry)
		if chain.CalculateHash(entry, domain.GenesisHash) == base {
			t.Errorf("%s muss in der kanonischen Form gedeckt sein", name)
		}
	}
}

// --- Beleg-Hash ------------------------------------------------------------

func sampleReceipt() *domain.Receipt {
	return &domain.Receipt{
		ReceiptNumber: "ER-2025-0001",
		FiscalYear:    2025,
		Direction:     domain.DirectionIncoming,
		Kind:          domain.ReceiptKindInvoice,
		Status:        domain.ReceiptStatusFiled,
		ReceivedAt:    "2025-03-01",
		ReceivedVia:   domain.ReceivedViaEmail,
		DocumentDate:  "2025-03-01",
		IssuerName:    "Lieferant GmbH",
		GrossAmount:   11900,
		TaxAmount:     1900,
		Currency:      "EUR",
		Subject:       "Bürobedarf",
		Files: []domain.ReceiptFile{{
			Position:   1,
			Role:       domain.ReceiptRoleOriginal,
			FileName:   "a.pdf",
			MimeType:   "application/pdf",
			SHA256:     strings.Repeat("aa", 32),
			Size:       3,
			StoredPath: "x/aa",
		}},
	}
}

func TestReceiptHashCoversHeader(t *testing.T) {
	hashed := ReceiptHash(sampleReceipt())

	changes := map[string]func(r *domain.Receipt){
		"Belegdatum": func(r *domain.Receipt) { r.DocumentDate = "" },
		"Betrag":     func(r *domain.Receipt) { r.GrossAmount = 11901 },
		"Steuer":     func(r *domain.Receipt) { r.TaxAmount = 1901 },
		"Aussteller": func(r *domain.Receipt) { r.IssuerName = "Jemand anderes GmbH" },
		"Betreff":    func(r *domain.Receipt) { r.Subject = "etwas ganz anderes" },
		"Belegart":   func(r *domain.Receipt) { r.Kind = domain.ReceiptKindLetter },
		"Datei":      func(r *domain.Receipt) { r.Files[0].SHA256 = strings.Repeat("bb", 32) },
	}
	for name, change := range changes {
		receipt := sampleReceipt()
		change(receipt)
		if ReceiptHash(receipt) == hashed {
			t.Errorf("%s muss im Beleg-Hash gedeckt sein", name)
		}
	}
}
