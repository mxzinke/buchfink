package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/goldentest"
	"github.com/buchfink/buchfink/internal/repository"
)

type settlementTaxFixture struct {
	name      string
	treatment domain.TaxTreatment
	rate      domain.TaxRate
	direction domain.Direction
	account   string
}

var settlementTaxFixtures = []settlementTaxFixture{
	{"eingang_inland19", domain.TaxTreatmentDomestic, domain.TaxRateStandard, domain.DirectionIncoming, "5906"},
	{"eingang_inland7", domain.TaxTreatmentDomestic, domain.TaxRateReduced, domain.DirectionIncoming, "5906"},
	{"ausgang_inland19", domain.TaxTreatmentDomestic, domain.TaxRateStandard, domain.DirectionOutgoing, "4400"},
	{"ausgang_inland7", domain.TaxTreatmentDomestic, domain.TaxRateReduced, domain.DirectionOutgoing, "4300"},
	{"13b_abs1", domain.TaxTreatmentReverseChargeEU, domain.TaxRateStandard, domain.DirectionIncoming, "5923"},
	{"13b_abs2", domain.TaxTreatmentReverseCharge, domain.TaxRateStandard, domain.DirectionIncoming, "5920"},
	{"ig_erwerb", domain.TaxTreatmentIntraCommunityAcquisition, domain.TaxRateStandard, domain.DirectionIncoming, "5425"},
	{"eingang_steuerfrei", domain.TaxTreatmentExempt, domain.TaxRateNone, domain.DirectionIncoming, "5900"},
	{"ausgang_steuerfrei", domain.TaxTreatmentExempt, domain.TaxRateNone, domain.DirectionOutgoing, "4100"},
	{"eingang_nicht_steuerbar", domain.TaxTreatmentNotTaxable, domain.TaxRateNone, domain.DirectionIncoming, "5900"},
	{"ausgang_nicht_steuerbar", domain.TaxTreatmentNotTaxable, domain.TaxRateNone, domain.DirectionOutgoing, "4338"},
	{"13b_leistender", domain.TaxTreatmentReverseChargeSupply, domain.TaxRateNone, domain.DirectionOutgoing, "4336"},
	{"nullsteuersatz", domain.TaxTreatmentZeroRated, domain.TaxRateNone, domain.DirectionIncoming, "5900"},
}

func settlementInvoice(t *testing.T, env *testEnv, tc settlementTaxFixture) (*domain.JournalEntry, domain.Cents) {
	t.Helper()
	contact := env.vendor(t, "Lieferant GmbH", "DE", "")
	side := domain.SideDebit
	if tc.direction == domain.DirectionOutgoing {
		contact = env.customer(t, "Kunde GmbH", "DE", "")
		side = domain.SideCredit
	}
	const net domain.Cents = 100000
	gross := net
	if tc.treatment == domain.TaxTreatmentDomestic {
		gross += tc.rate.Tax(net)
	}
	legs, err := env.journal.TaxResolver().Resolve(tc.direction, tc.treatment, tc.rate, net)
	if err != nil {
		t.Fatal(err)
	}
	lines := []domain.JournalLine{{Side: side, Account: tc.account, Amount: net}}
	for _, leg := range legs {
		lines = append(lines, taxLegLine(leg))
	}
	lines = append(lines, domain.JournalLine{
		Side: side.Opposite(), Account: contact.LedgerAccount, ContactID: &contact.ID, Amount: gross,
	})
	entry, err := env.journal.Post(context.Background(), &domain.JournalEntry{
		BookingDate: "2026-03-01", DocumentDate: "2026-03-01",
		ServiceDateFrom: "2026-03-01", ServiceDateTo: "2026-03-01",
		Description: "Rechnung", DocumentNumber: "RE-4711", Source: domain.EntrySourceManual,
		TaxTreatment: tc.treatment, ContactID: &contact.ID, Lines: lines,
	})
	if err != nil {
		t.Fatal(err)
	}
	return entry, gross
}

// Neben den Journalzeilen bleiben Zuordnungen, Stichtagssicht, Bankstatus und
// das Protokoll Teil des Vertrags. Nur flüchtige Stempel werden normiert.
func settlementSnapshot(t *testing.T, env *testEnv, payments *PaymentService, auditStart int, bookErr error) map[string]any {
	t.Helper()
	ctx := context.Background()
	entries, err := env.journalRepo.FindAll(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	items, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	beforePayment, err := payments.OpenItemsAt(ctx, "2026-03-09")
	if err != nil {
		t.Fatal(err)
	}
	aging, err := payments.OpenItemsAging(ctx, "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	allocations, err := repository.NewPaymentAllocationRepository(env.db).FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	details := []domain.PaymentAllocationDetail{}
	for _, entry := range entries {
		if entry.Source == domain.EntrySourcePayment && entry.Kind == domain.EntryKindNormal {
			rows, err := payments.Allocations(ctx, entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			details = append(details, rows...)
		}
	}
	bank, err := repository.NewBankRepository(env.db).FindAll(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := repository.NewAuditRepository(env.db).FindAllAscending(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for i := range entries {
		entry := &entries[i]
		if entry.Kind != domain.EntryKindReversal {
			continue
		}
		if entry.BookingDate != testNow().Format(domain.DateLayout) {
			t.Fatalf("Stornotag %s, erwartet heute", entry.BookingDate)
		}
		for j := range logs {
			if logs[j].EntityType == "JOURNAL" && logs[j].EntityID == fmt.Sprint(entry.ID) {
				logs[j].Details = strings.ReplaceAll(logs[j].Details, entry.EntryNumber, "<Storno>")
				logs[j].Details = strings.ReplaceAll(logs[j].Details, fmt.Sprintf("GJ %d", entry.FiscalYear), "GJ <Stornojahr>")
			}
		}
		entry.BookingDate, entry.EntryNumber, entry.FiscalYear = "<Stornotag>", "<Storno>", 0
	}
	message := ""
	if bookErr != nil {
		message = bookErr.Error()
	}
	return map[string]any{
		"error": message, "journal": entries, "openItems": items, "beforePayment": beforePayment,
		"aging": aging, "allocations": allocations, "details": details, "bank": bank, "audit": logs[auditStart:],
	}
}

func assertSettlementGolden(t *testing.T, name string, result map[string]any) {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	normalizeBookedGolden(t, value)
	normalizeSettlementTimestamps(value)
	goldentest.Assert(t, "settlement/"+name, value)
}

func normalizeSettlementTimestamps(value any) {
	switch v := value.(type) {
	case []any:
		for _, child := range v {
			normalizeSettlementTimestamps(child)
		}
	case map[string]any:
		for key, child := range v {
			if key == "updatedAt" && child != "" {
				v[key] = "<updatedAt>"
			} else {
				normalizeSettlementTimestamps(child)
			}
		}
	}
}

func settlementAuditStart(t *testing.T, env *testEnv) int {
	t.Helper()
	logs, err := repository.NewAuditRepository(env.db).FindAllAscending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return len(logs)
}

func TestGoldenSettlementTaxCases(t *testing.T) {
	for _, tc := range settlementTaxFixtures {
		for _, skonto := range []bool{false, true} {
			name := fmt.Sprintf("%s/skonto_%t", tc.name, skonto)
			t.Run(name, func(t *testing.T) {
				env := newTestEnv(t)
				payments := env.paymentsWithBank(t)
				invoice, gross := settlementInvoice(t, env, tc)
				allocation := AllocationRequest{OpenItemEntryID: invoice.ID, SettledAmount: gross}
				if skonto {
					allocation.DifferenceKind, allocation.DifferenceAmount = domain.DifferenceSkonto, gross/50
				}
				cash := allocation.cashAmount(domain.SideDebit)
				if tc.direction == domain.DirectionIncoming {
					cash = -cash
				}
				id := env.bankLine(t, "2026-03-10", cash, "Zahlungspartner", "RE-4711")
				start := settlementAuditStart(t, env)
				ctx := actor.WithChannel(context.Background(), "agent:golden")
				_, err := payments.Settle(ctx, PaymentRequest{BankTxID: &id, Allocations: []AllocationRequest{allocation}})
				if err != nil {
					t.Fatal(err)
				}
				assertSettlementGolden(t, name, settlementSnapshot(t, env, payments, start, nil))
			})
		}
	}
}

func TestGoldenSettlementPaymentWays(t *testing.T) {
	for _, name := range []string{
		"teilzahlung", "ueberzahlung", "kursgewinn", "kursverlust", "anlage_skonto", "anlage_skonto_storno",
		"aufteilung_gebuehr", "aufteilung_skonto", "aufteilung_darlehen", "storno", "zuordnung_scheitert", "anlage_scheitert",
	} {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			payments := env.paymentsWithBank(t)
			tc := settlementTaxFixtures[0]
			if strings.HasPrefix(name, "anlage_") {
				tc.account = "0440"
			}
			var invoice *domain.JournalEntry
			var gross domain.Cents
			if strings.HasPrefix(name, "kurs") {
				contact := env.vendor(t, "Lieferant USA", "US", "")
				if name == "kursverlust" {
					contact = env.customer(t, "Kunde USA", "US", "")
				}
				invoice = env.foreignOpenItem(t, contact, "USD-4711", 100000, 1100000)
				gross = 100000
			} else {
				invoice, gross = settlementInvoice(t, env, tc)
			}
			allocation := AllocationRequest{OpenItemEntryID: invoice.ID, SettledAmount: gross}
			switch name {
			case "teilzahlung":
				allocation.SettledAmount = 50000
			case "ueberzahlung":
				allocation.SettledAmount = gross + 1
			case "kursgewinn", "kursverlust":
				allocation.DifferenceKind, allocation.DifferenceAmount = domain.DifferenceCurrency, 2500
			case "anlage_skonto", "anlage_skonto_storno", "anlage_scheitert", "aufteilung_skonto":
				allocation.DifferenceKind, allocation.DifferenceAmount = domain.DifferenceSkonto, gross/50
			}
			var assetID uint
			assets := env.assets(t)
			if strings.HasPrefix(name, "anlage_") {
				asset, err := assets.Save(context.Background(), &domain.FixedAsset{
					Name: "Maschine", Class: domain.AssetClassTangible, Account: "0440", DepreciationAccount: "6220",
					AcquisitionDate: "2026-03-01", AcquisitionCost: 100000, UsefulLifeMonths: 48,
					Method: domain.DepreciationLinear, AcquisitionEntryID: &invoice.ID,
				})
				if err != nil {
					t.Fatal(err)
				}
				assetID = asset.ID
				payments.SetAssetRegister(assets)
				if name == "anlage_scheitert" {
					payments.SetAssetRegister(env.assetsWithAudit(t, "ANLAGE"))
				}
			}
			cash := -allocation.cashAmount(domain.SideDebit)
			date := "2026-03-10"
			if strings.HasPrefix(name, "kurs") {
				date = "2026-12-10"
			}
			if name == "kursverlust" {
				cash = -cash
			}
			if strings.HasPrefix(name, "aufteilung_") {
				cash -= 250
			}
			id := env.bankLine(t, date, cash, "Zahlungspartner", "RE-4711")
			split := BankSplitRequest{BankTxID: id, Lines: []BankSplitLine{
				{OpenItemEntryID: invoice.ID, SettledAmount: gross, DifferenceKind: allocation.DifferenceKind, DifferenceAmount: allocation.DifferenceAmount},
				{Account: domain.AccountNebenkostenGeld, Amount: 250, Text: "Überweisungsentgelt"},
			}}
			if name == "aufteilung_darlehen" {
				env.withShareholders(t)
				person := env.person(t, "Mara Beispiel")
				loans := env.shareholders(t)
				loan, err := loans.RegisterLoan(context.Background(), LoanRequest{
					ContactID: person.ID, Direction: domain.LoanToCompany, Principal: 500000,
					InterestRateBP: 350, ContractDate: "2026-01-01", StartDate: "2026-01-01", MaturityDate: "2029-12-31",
				})
				if err != nil {
					t.Fatal(err)
				}
				payments.SetLoanSource(loans)
				split.Lines[0] = BankSplitLine{LoanID: loan.Loan.ID, Amount: gross, Text: "Tilgung"}
			}
			if name == "zuordnung_scheitert" {
				payments = NewPaymentService(env.journal, env.journalRepo,
					failingAllocations{repository.NewPaymentAllocationRepository(env.db)}, env.contactRepo, repository.NewBankRepository(env.db))
			}
			start := settlementAuditStart(t, env)
			ctx := actor.WithChannel(context.Background(), "agent:golden")
			var entry *domain.JournalEntry
			var err error
			if strings.HasPrefix(name, "aufteilung_") {
				entry, err = payments.BookBankSplit(ctx, split)
			} else {
				entry, err = payments.Settle(ctx, PaymentRequest{BankTxID: &id, Allocations: []AllocationRequest{allocation}})
			}
			wantError := name == "ueberzahlung" || name == "zuordnung_scheitert" || name == "anlage_scheitert"
			if (err != nil) != wantError {
				t.Fatalf("Fehler = %v, erwartet Fehler: %t", err, wantError)
			}
			result := settlementSnapshot(t, env, payments, start, err)
			if name == "storno" || name == "anlage_skonto_storno" {
				if _, err := env.journal.Reverse(ctx, entry.ID, "Zahlung berichtigen"); err != nil {
					t.Fatal(err)
				}
				result["reversed"] = settlementSnapshot(t, env, payments, start, nil)
			}
			if assetID != 0 {
				detail, err := assets.Get(ctx, 2026, assetID)
				if err != nil {
					t.Fatal(err)
				}
				result["assetCost"], result["assetMovements"] = detail.Asset.Cost, detail.Movements
			}
			assertSettlementGolden(t, name, result)
		})
	}
}

func TestGoldenSettlementWriteOff(t *testing.T) {
	for _, name := range []string{"inland19", "inland7", "steuerfrei", "nicht_steuerbar", "13b_leistender", "teilbetrag", "verbindlichkeit", "grund_fehlt", "protokoll_scheitert"} {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			payments := env.payments(t)
			payments.SetAuditRepo(repository.NewAuditRepository(env.db))
			tc := settlementTaxFixtures[2]
			for _, fixture := range settlementTaxFixtures {
				if fixture.name == "ausgang_"+name || fixture.name == name {
					tc = fixture
				}
			}
			if name == "verbindlichkeit" {
				tc = settlementTaxFixtures[0]
			}
			invoice, _ := settlementInvoice(t, env, tc)
			req := WriteOffRequest{OpenItemEntryID: invoice.ID, Date: "2026-03-10", Reason: "Insolvenz mangels Masse"}
			if name == "teilbetrag" {
				req.Amount = 3
			}
			if name == "grund_fehlt" {
				req.Reason = ""
			}
			if name == "protokoll_scheitert" {
				payments.SetAuditRepo(failingAudit{AuditRepository: repository.NewAuditRepository(env.db), trigger: "OPEN_ITEM"})
			}
			start := settlementAuditStart(t, env)
			ctx := actor.WithChannel(context.Background(), "agent:golden")
			_, err := payments.WriteOffOpenItem(ctx, req)
			wantError := name == "verbindlichkeit" || name == "grund_fehlt" || name == "protokoll_scheitert"
			if (err != nil) != wantError {
				t.Fatalf("Fehler = %v, erwartet Fehler: %t", err, wantError)
			}
			assertSettlementGolden(t, "ausbuchen/"+name, settlementSnapshot(t, env, payments, start, err))
		})
	}
}
