package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// takeoverFixture ist die Schlussbilanz zum 31.12.2025 eines Umsteigers:
//
//	Pkw (0520)              20.000,00   Gezeichnetes Kapital (2900)  25.000,00
//	Büroeinrichtung (0650)   4.200,00   Gewinnvortrag (2970)          9.405,00
//	Forderungen (1200)       1.785,00   Rückstellungen (3095)         2.500,00
//	Bank (1800)             15.000,00   Verbindlichkeiten (3300)      4.080,00
//	                        40.985,00                                40.985,00
//
// Der Pkw kostete am 15.01.2024 30.000 € (72 Monate, 10.000 € abgeschrieben),
// die Büroeinrichtung am 10.01.2023 6.000 € (120 Monate, 1.800 € abgeschrieben).
func takeoverFixture(receiptID, customerID, vendorID uint) service.TakeoverRequest {
	standard := domain.TaxRateStandard
	return service.TakeoverRequest{
		OpeningBalanceRequest: service.OpeningBalanceRequest{
			ReceiptID: &receiptID, LegacySystem: "Vorgängerprogramm",
			Accounts: []service.OpeningBalanceLine{
				{Account: "0520", Side: domain.SideDebit, Amount: 2_000_000},
				{Account: "0650", Side: domain.SideDebit, Amount: 420_000},
				{Account: domain.AccountForderungenLuL, Side: domain.SideDebit, Amount: 178_500},
				{Account: domain.AccountBank, Side: domain.SideDebit, Amount: 1_500_000},
				{Account: "2900", Side: domain.SideCredit, Amount: 2_500_000},
				{Account: domain.AccountGewinnvortrag, Side: domain.SideCredit, Amount: 940_500},
				{Account: "3095", Side: domain.SideCredit, Amount: 250_000},
				{Account: domain.AccountVerbindlichkeitenLuL, Side: domain.SideCredit, Amount: 408_000},
			},
			Receivables: []service.OpenOpeningItem{
				{
					ContactID: customerID, Amount: 119_000, DocumentNumber: "RE-2025-0098",
					DocumentDate: "2025-12-10", DueDate: "2026-01-09",
					TaxRate: &standard, TaxTreatment: domain.TaxTreatmentDomestic,
				},
				{ContactID: customerID, Amount: 59_500, DocumentNumber: "RE-2025-0099", DocumentDate: "2025-12-20"},
			},
			Payables: []service.OpenOpeningItem{
				{ContactID: vendorID, Amount: 408_000, DocumentNumber: "ER-4711", DocumentDate: "2025-12-15"},
			},
		},
		Assets: []service.TakeoverAsset{
			{
				Name: "Pkw Kombi", Account: "0520", AcquisitionDate: "2024-01-15", AcquisitionCost: 3_000_000,
				UsefulLifeMonths: 72, AccumulatedDepreciation: 1_000_000, LegacyRef: "INV-0007",
			},
			{
				Name: "Büromöbel", Account: "0650", AcquisitionDate: "2023-01-10", AcquisitionCost: 600_000,
				UsefulLifeMonths: 120, AccumulatedDepreciation: 180_000,
			},
		},
	}
}

// Ein Umstieg zum Geschäftsjahresbeginn von der Festlegung bis zum
// Saldenvortrag ins Folgejahr.
func TestTakeoverFromLegacySystem(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Umsteiger GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)

	var receipt *domain.Receipt
	customer := &domain.Contact{
		Type: domain.ContactTypeCustomer, Name: "Kunde GmbH", CountryCode: "DE",
		Street: "Kundenweg 2", PostalCode: "10115", City: "Berlin",
	}
	vendor := &domain.Contact{
		Type: domain.ContactTypeVendor, Name: "Lieferant GmbH", CountryCode: "DE",
		Street: "Lieferantenweg 3", PostalCode: "20095", City: "Hamburg", TaxID: "12/345/67890",
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) (err error) {
		if err = s.Contacts.SaveContact(ctx, customer); err != nil {
			return err
		}
		if err = s.Contacts.SaveContact(ctx, vendor); err != nil {
			return err
		}
		receipt, err = s.Receipts.File(ctx, service.FileReceiptRequest{
			Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindOther, DocumentDate: "2025-12-31",
			Subject: "Schlussbilanz zum 31.12.2025 aus dem Vorgängerprogramm",
			Files: []service.NewFile{{
				Role: domain.ReceiptRoleOriginal, FileName: "schlussbilanz.pdf",
				Content: []byte("%PDF-1.4 Schlussbilanz\n"),
			}},
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	req := takeoverFixture(receipt.ID, customer.ID, vendor.ID)
	book := func(ctx context.Context, req service.TakeoverRequest) (*service.TakeoverPreview, error) {
		var out *service.TakeoverPreview
		err := tenant.Write(ctx, func(ctx context.Context, s *Services) (err error) {
			out, err = s.Takeover.Book(ctx, req)
			return err
		})
		return out, err
	}
	findings := func() []domain.CheckFinding {
		t.Helper()
		var out []domain.CheckFinding
		if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) error {
			run, err := s.Checks.Preview(ctx, service.CheckRequest{
				FiscalYear: 2026, CutoffDate: "2026-01-31", PeriodType: "month",
			})
			if err != nil {
				return err
			}
			for _, f := range run.Findings {
				if f.Rule == domain.CheckRuleTakeoverOpening {
					out = append(out, f)
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return out
	}

	// Ohne festgelegten Umstieg nimmt der Dienst keine Bestände an, und
	// festlegen kann ihn nur der Mensch.
	if _, err := book(agent, req); err == nil || !strings.Contains(err.Error(), "kein Umstieg festgelegt") {
		t.Fatalf("Eröffnung ohne festgelegten Umstieg: %v", err)
	}
	setDate := func(ctx context.Context, date string) error {
		return tenant.Write(ctx, func(ctx context.Context, s *Services) error {
			_, err := s.Takeover.SetDate(ctx, date)
			return err
		})
	}
	if err := setDate(agent, "2026-01-01"); !errors.Is(err, ErrAppOnly) {
		t.Fatalf("ein Agent legt den Umstieg fest: %v", err)
	}
	if err := setDate(uiCtx, "2026-03-01"); err == nil ||
		!strings.Contains(err.Error(), "ersten Tag eines Geschäftsjahres") {
		t.Fatalf("Umstieg im laufenden Geschäftsjahr: %v", err)
	}
	if got := findings(); len(got) != 0 {
		t.Fatalf("ohne Umstieg meldet der Prüflauf eine Eröffnung: %+v", got)
	}
	if err := setDate(uiCtx, "2026-01-01"); err != nil {
		t.Fatal(err)
	}
	if got := findings(); len(got) != 1 || got[0].Severity != domain.CheckBlocking ||
		!strings.Contains(got[0].Message, "keine Eröffnung") {
		t.Fatalf("fehlende Eröffnung im Prüflauf: %+v", got)
	}

	// Eine Eröffnung, die nicht aufgeht, und ein Anlagenverzeichnis, das vom
	// Anlagekonto abweicht, werden nicht gebucht.
	unbalanced := takeoverFixture(receipt.ID, customer.ID, vendor.ID)
	unbalanced.Accounts[4].Amount -= 10_000
	if _, err := book(agent, unbalanced); err == nil || !strings.Contains(err.Error(), "unterscheiden sich um 100,00 €") {
		t.Fatalf("unausgeglichene Eröffnung: %v", err)
	}
	wrongAssets := takeoverFixture(receipt.ID, customer.ID, vendor.ID)
	wrongAssets.Assets[0].AccumulatedDepreciation = 900_000
	if _, err := book(agent, wrongAssets); err == nil ||
		!strings.Contains(err.Error(), "Konto 0520") || !strings.Contains(err.Error(), "Differenz 1.000,00 €") {
		t.Fatalf("Anlagen weichen vom Anlagekonto ab: %v", err)
	}
	wrongItems := takeoverFixture(receipt.ID, customer.ID, vendor.ID)
	wrongItems.Receivables = wrongItems.Receivables[:1]
	if _, err := book(agent, wrongItems); err == nil || !strings.Contains(err.Error(), "Sammelkonto 1200") {
		t.Fatalf("offene Posten weichen vom Sammelkonto ab: %v", err)
	}
	if n := countEntries(t, tenant); n != 0 {
		t.Fatalf("%d Buchungen nach den abgewiesenen Eröffnungen", n)
	}

	// Der Agent bucht die Eröffnung: eine Buchung der Sachkonten, je offenen
	// Posten eine, dazu die beiden Anlagen.
	booked, err := book(agent, req)
	if err != nil {
		t.Fatalf("Eröffnung buchen: %v", err)
	}
	if len(booked.Entries) != 4 || len(booked.Assets) != 2 || booked.DebitTotal != 4_098_500 ||
		booked.CreditTotal != 4_098_500 {
		t.Fatalf("Eröffnung: %d Buchungen, %d Anlagen, Aktiva %s, Passiva %s",
			len(booked.Entries), len(booked.Assets), booked.DebitTotal, booked.CreditTotal)
	}
	if booked.Assets[0].BookValue != 2_000_000 || booked.Assets[0].FirstYearDepreciation != 500_000 ||
		booked.Assets[0].InventoryNumber == "" {
		t.Errorf("Pkw nach der Übernahme: %+v", booked.Assets[0])
	}
	if _, err := book(agent, req); err == nil {
		t.Fatal("eine zweite Eröffnung verdoppelt die Bestände")
	}
	if got := findings(); len(got) != 0 {
		t.Errorf("Prüflauf nach der Eröffnung: %+v", got)
	}

	if err := tenant.Read(agent, func(ctx context.Context, s *Services) error {
		rec, err := s.Takeover.Reconciliation(ctx)
		if err != nil {
			return err
		}
		if !rec.Reconciled || len(rec.Differences) != 0 || rec.CarryForwardBalance != 0 ||
			rec.StatementAssets != 4_098_500 || rec.StatementLiabilities != 4_098_500 ||
			rec.BookedAssets != 4_098_500 || rec.BookedLiabilities != 4_098_500 {
			t.Errorf("Abstimmung: %+v", rec)
		}
		rows := map[string]service.TakeoverAccountRow{}
		for _, row := range rec.Accounts {
			rows[row.Account] = row
		}
		if len(rows) != 8 || rows["1200"].Booked != 178_500 || rows["3300"].Booked != -408_000 ||
			rows["0520"].Statement != 2_000_000 || rows["2900"].Booked != -2_500_000 {
			t.Errorf("Abstimmung je Konto: %+v", rec.Accounts)
		}
		subs := map[string]service.TakeoverSubLedgerRow{}
		for _, row := range rec.SubLedgers {
			subs[row.Kind+"/"+row.Account] = row
		}
		if sub := subs["receivables/1200"]; sub.Count != 2 || sub.SubLedger != 178_500 || sub.Difference != 0 {
			t.Errorf("Nebenbuch Forderungen: %+v", sub)
		}
		if sub := subs["assets/0520"]; sub.Count != 1 || sub.SubLedger != 2_000_000 || sub.Ledger != 2_000_000 {
			t.Errorf("Nebenbuch Anlagen: %+v", sub)
		}

		// Der Anlagenspiegel zeigt Anschaffungskosten und kumulierte
		// Abschreibungen getrennt.
		mirror, err := s.Assets.Anlagenspiegel(ctx, 2026)
		if err != nil {
			return err
		}
		if mirror.Totals.CostOpening != 3_600_000 || mirror.Totals.DepreciationOpening != 1_180_000 ||
			mirror.Totals.BookValueOpening != 2_420_000 {
			t.Errorf("Anlagenspiegel zu Beginn: %+v", mirror.Totals)
		}
		candidates, err := s.Assets.AcquisitionCandidates(ctx, 2026)
		if err != nil {
			return err
		}
		if len(candidates) != 0 {
			t.Errorf("die Eröffnung erscheint als Zugang ohne Anlagegut: %+v", candidates)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Zahlung auf einen übernommenen Posten und AfA des ersten Jahres.
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		items, err := s.Payments.OpenItems(ctx)
		if err != nil {
			return err
		}
		var itemID uint
		for _, item := range items {
			if item.DocumentNumber == "RE-2025-0098" {
				itemID = item.EntryID
			}
		}
		if len(items) != 3 || itemID == 0 {
			t.Fatalf("offene Posten nach der Eröffnung: %+v", items)
		}
		if _, err := s.Payments.Settle(ctx, service.PaymentRequest{
			PaymentAccount: domain.AccountBank, PaymentDate: "2026-01-09",
			Allocations: []service.AllocationRequest{{OpenItemEntryID: itemID, SettledAmount: 119_000}},
		}); err != nil {
			return err
		}
		if items, err = s.Payments.OpenItems(ctx); err != nil || len(items) != 2 {
			t.Errorf("offene Posten nach der Zahlung: %+v, %v", items, err)
		}

		run, err := s.Assets.Run(ctx, 2026)
		if err != nil {
			return err
		}
		if len(run.MissingPriorYears) != 0 || len(run.Due) != 2 || run.Total != 560_000 {
			t.Fatalf("AfA-Lauf 2026: fehlende Vorjahre %v, %d Anlagen, %s €",
				run.MissingPriorYears, len(run.Due), run.Total)
		}
		result, err := s.Assets.BookDepreciation(ctx, service.BookDepreciationRequest{FiscalYear: 2026})
		if err != nil {
			return err
		}
		if len(result.Entries) != 2 || result.Total != 560_000 {
			t.Errorf("AfA 2026: %d Buchungen über %s €", len(result.Entries), result.Total)
		}
		mirror, err := s.Assets.Anlagenspiegel(ctx, 2026)
		if err != nil {
			return err
		}
		if mirror.Totals.DepreciationYear != 560_000 || mirror.Totals.BookValueClosing != 1_860_000 {
			t.Errorf("Anlagenspiegel am Jahresende: %+v", mirror.Totals)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Jahresabschluss: die Bilanz geht auf, und der Saldenvortrag nimmt die
	// Bestände ins Folgejahr mit.
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		preview, err := s.Closing.CarryForwardState(ctx, 2027)
		if err != nil {
			return err
		}
		if !preview.IsBalanced || preview.NetIncome != -560_000 {
			t.Fatalf("Vortrag nach 2027: ausgeglichen %v, Ergebnis %s €", preview.IsBalanced, preview.NetIncome)
		}
		closing := map[string]domain.Cents{}
		for _, row := range preview.Rows {
			closing[row.Account] = row.ClosingBalance
		}
		if closing[domain.AccountBank] != 1_619_000 || closing["0520"] != 1_500_000 || closing["0650"] != 360_000 {
			t.Errorf("Schlusssalden 2026: %+v", closing)
		}
		created, err := s.Closing.CarryForward(ctx, 2027)
		if err != nil {
			return err
		}
		if len(created) == 0 {
			t.Error("kein Saldenvortrag gebucht")
		}
		rec, err := s.Takeover.Reconciliation(ctx)
		if err != nil {
			return err
		}
		if !rec.Reconciled {
			t.Errorf("Abstimmung nach dem Saldenvortrag: %v", rec.Differences)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Ein Umsteiger erfasst keine Gründung.
	err = tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Foundation.Save(ctx, "2026-01-10", takeoverTestFoundation())
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "Umstieg") {
		t.Errorf("Gründung neben dem Umstieg: %v", err)
	}
	if err := setDate(uiCtx, ""); err == nil || !strings.Contains(err.Error(), "storniert") {
		t.Errorf("Umstieg trotz gebuchter Eröffnung zurückgenommen: %v", err)
	}
}

func takeoverTestFoundation() *domain.Foundation {
	return &domain.Foundation{
		ShareCapital: 2_500_000,
		Shareholders: []domain.Shareholder{{Name: "Mara Beispiel", ShareCapital: 2_500_000, PaidIn: 1_250_000}},
	}
}

// Der Gründungsfall bleibt, wie er ist: die Gründung lässt sich erfassen, und
// ein Umstieg ist daneben ausgeschlossen.
func TestFoundationExcludesTakeover(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Gründer GmbH")
	tenant := openTenant(t, h)

	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		saved, err := s.Foundation.Save(ctx, "2026-01-10", takeoverTestFoundation())
		if err != nil {
			return err
		}
		if saved.ShareCapital != 2_500_000 {
			t.Errorf("Gründung: %+v", saved)
		}
		state, err := s.Takeover.State(ctx)
		if err != nil {
			return err
		}
		if state.Date != "" || state.Booked {
			t.Errorf("Umstieg einer gegründeten Gesellschaft: %+v", state)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Takeover.SetDate(ctx, "2027-01-01")
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "Gründung erfasst") {
		t.Errorf("Umstieg neben der Gründung: %v", err)
	}
}

// Eine fehlerhafte Eröffnung wird storniert und mit demselben Beleg neu
// gebucht. Die übernommenen Anlagen bleiben im Verzeichnis und gehören zur
// neuen Eröffnung.
func TestTakeoverIsCorrectedByReversalAndRebooking(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Umsteiger GmbH")
	tenant := openTenant(t, h)

	var req service.TakeoverRequest
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		customer := &domain.Contact{Type: domain.ContactTypeCustomer, Name: "Kunde GmbH", CountryCode: "DE"}
		vendor := &domain.Contact{Type: domain.ContactTypeVendor, Name: "Lieferant GmbH", CountryCode: "DE"}
		if err := s.Contacts.SaveContact(ctx, customer); err != nil {
			return err
		}
		if err := s.Contacts.SaveContact(ctx, vendor); err != nil {
			return err
		}
		receipt, err := s.Receipts.File(ctx, service.FileReceiptRequest{
			Direction: domain.DirectionIncoming, Kind: domain.ReceiptKindOther, DocumentDate: "2025-12-31",
			Subject: "Schlussbilanz zum 31.12.2025",
			Files: []service.NewFile{{
				Role: domain.ReceiptRoleOriginal, FileName: "schlussbilanz.pdf",
				Content: []byte("%PDF-1.4 Schlussbilanz\n"),
			}},
		})
		if err != nil {
			return err
		}
		if _, err := s.Takeover.SetDate(ctx, "2026-01-01"); err != nil {
			return err
		}
		req = takeoverFixture(receipt.ID, customer.ID, vendor.ID)
		booked, err := s.Takeover.Book(ctx, req)
		if err != nil {
			return err
		}
		for _, entry := range booked.Entries {
			if _, err := s.Journal.ReverseOn(ctx, entry.ID, "Schlussbilanz berichtigt", "2026-01-01"); err != nil {
				return err
			}
		}
		rec, err := s.Takeover.Reconciliation(ctx)
		if err != nil {
			return err
		}
		if rec.Booked || len(rec.Differences) != 1 {
			t.Errorf("Abstimmung nach dem Storno: %+v", rec)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := s.Takeover.Book(ctx, req)
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "übernommene Anlagen") {
		t.Fatalf("Anlagen ein zweites Mal übernommen: %v", err)
	}

	// Die berichtigte Schlussbilanz: 500 € mehr auf der Bank und im Gewinnvortrag.
	req.Assets = nil
	req.Accounts[3].Amount += 50_000
	req.Accounts[5].Amount += 50_000
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		booked, err := s.Takeover.Book(ctx, req)
		if err != nil {
			return err
		}
		if len(booked.Assets) != 2 || booked.DebitTotal != 4_148_500 {
			t.Errorf("erneute Eröffnung: %d Anlagen, Aktiva %s", len(booked.Assets), booked.DebitTotal)
		}
		rec, err := s.Takeover.Reconciliation(ctx)
		if err != nil {
			return err
		}
		if !rec.Reconciled || rec.BookedAssets != 4_148_500 {
			t.Errorf("Abstimmung nach der Berichtigung: %v, Aktiva %s", rec.Differences, rec.BookedAssets)
		}
		assets, err := s.Assets.List(ctx, 2026, "")
		if err != nil {
			return err
		}
		if len(assets) != 2 {
			t.Errorf("%d Anlagen nach der Berichtigung", len(assets))
		}
		candidates, err := s.Assets.AcquisitionCandidates(ctx, 2026)
		if err != nil {
			return err
		}
		if len(candidates) != 0 {
			t.Errorf("die neue Eröffnung erscheint als Zugang ohne Anlagegut: %+v", candidates)
		}
		run, err := s.Assets.Run(ctx, 2026)
		if err != nil {
			return err
		}
		if len(run.MissingPriorYears) != 0 || run.Total != 560_000 {
			t.Errorf("AfA-Lauf nach der Berichtigung: %v, %s €", run.MissingPriorYears, run.Total)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
