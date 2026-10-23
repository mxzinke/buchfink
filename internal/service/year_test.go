package service

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kein Dienst hält ein Geschäftsjahr. Das Jahr ist eine Angabe des Aufrufs;
// ein Feld im Dienst wäre ein Jahr, das gerade jemand ansieht, und zwei
// Aufrufer mit verschiedenen Jahren (Oberfläche, Agent) sähen sich gegenseitig
// das falsche.
func TestNoServiceHoldsAFiscalYear(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			spec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				for _, ident := range field.Names {
					if strings.EqualFold(ident.Name, "fiscalYear") && !ident.IsExported() {
						t.Errorf("%s: %s hält ein Geschäftsjahr im Feld %s", fset.Position(ident.Pos()), spec.Name.Name, ident.Name)
					}
				}
			}
			return true
		})
	}
}

// Ein Aufruf ohne Jahr wird abgewiesen, statt still ein Jahr anzunehmen.
func TestCallsWithoutYearAreRefused(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	if _, err := env.accounting.GetEntries(ctx, 0); !errors.Is(err, domain.ErrFiscalYearRequired) {
		t.Errorf("Journal ohne Jahr: %v", err)
	}
	if _, err := env.receipts.List(ctx, 0, ""); !errors.Is(err, domain.ErrFiscalYearRequired) {
		t.Errorf("Belege ohne Jahr: %v", err)
	}
	if _, err := NewVatService(env.journalRepo).Summary(ctx, 0, "", ""); !errors.Is(err, domain.ErrFiscalYearRequired) {
		t.Errorf("Umsatzsteuer ohne Jahr: %v", err)
	}
}

// Derselbe Dienst beantwortet zwei Jahre nebeneinander, ohne dass das eine das
// andere verschiebt.
func TestOneServiceAnswersTwoYears(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.bookOn(t, "2025-11-10", "6815", "1800", 5000)
	env.bookOn(t, "2026-02-10", "6815", "1800", 3000)

	old, err := env.accounting.GetEntries(ctx, 2025)
	if err != nil {
		t.Fatal(err)
	}
	current, err := env.accounting.GetEntries(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != 1 || old[0].BookingDate != "2025-11-10" {
		t.Errorf("2025: %+v", old)
	}
	if len(current) != 1 || current[0].BookingDate != "2026-02-10" {
		t.Errorf("2026: %+v", current)
	}
}

// Ein Beleg gehört in das Geschäftsjahr seines Belegdatums, ohne Belegdatum in
// das seines Eingangs — nie in das Jahr, das jemand gerade ansieht.
func TestReceiptTakesTheYearFromItsDates(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	file := func(document, received string) *domain.Receipt {
		t.Helper()
		receipt, err := env.receipts.File(ctx, FileReceiptRequest{
			Direction: domain.DirectionIncoming, DocumentDate: document, ReceivedAt: received,
			Files: []NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "beleg.pdf",
				Content: []byte("%PDF-1.4 " + document + received)}},
		})
		if err != nil {
			t.Fatalf("Beleg %s/%s: %v", document, received, err)
		}
		return receipt
	}
	if got := file("2025-12-30", "2026-01-04").FiscalYear; got != 2025 {
		t.Errorf("Beleg vom 30.12.2025: Geschäftsjahr %d, erwartet 2025", got)
	}
	if got := file("", "2026-01-04").FiscalYear; got != 2026 {
		t.Errorf("Beleg ohne Datum, eingegangen 2026: Geschäftsjahr %d", got)
	}
	if _, err := env.receipts.File(ctx, FileReceiptRequest{
		Direction: domain.DirectionIncoming, DocumentDate: "30.12.2025",
		Files: []NewFile{{Role: domain.ReceiptRoleOriginal, FileName: "x.pdf", Content: []byte("%PDF-1.4 x")}},
	}); err == nil {
		t.Error("ein unlesbares Belegdatum wurde angenommen")
	}

	// Ein abweichendes Geschäftsjahr ab Juli: der März 2026 gehört zu 2025.
	settings, err := env.receipts.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	settings.FiscalYearStartMonth = 7
	if err := env.receipts.settingsRepo.UpdateCompanySettings(ctx, settings); err != nil {
		t.Fatal(err)
	}
	if got := file("2026-03-01", "").FiscalYear; got != 2025 {
		t.Errorf("März 2026 bei Beginn im Juli: Geschäftsjahr %d, erwartet 2025", got)
	}
}

// Ein Rechnungsverbund gehört in das Jahr seines Auftrags.
func TestInvoiceGroupTakesTheYearOfItsOrder(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	customer := env.customer(t, "Kunde GmbH", "DE", "")
	svc := env.invoicesWired(t)
	group, err := svc.CreateInvoiceGroup(ctx, AdvanceGroupRequest{
		Date: "2025-12-01", ContactID: customer.ID, Title: "Umbau",
		TotalNet: 100000, TaxRate: domain.TaxRateStandard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if group.FiscalYear != 2025 {
		t.Errorf("Verbund vom 01.12.2025 im Geschäftsjahr %d", group.FiscalYear)
	}
}
