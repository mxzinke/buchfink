package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/invoice"
	"github.com/buchfink/buchfink/internal/repository"
)

// statements wires the Jahresabschluss-Auswertung on the shared test
// environment.
func (e *testEnv) statements(t *testing.T) *StatementService {
	t.Helper()
	return NewStatementService(
		e.accounting,
		e.closing(t),
		repository.NewSettingsRepository(e.db),
		repository.NewAuditRepository(e.db),
		e.fiscalYear,
	)
}

func (e *testEnv) post(t *testing.T, date, debit, credit string, amount domain.Cents) {
	t.Helper()
	if _, err := e.journal.Post(context.Background(), datedEntry(date, debit, credit, amount)); err != nil {
		t.Fatalf("Buchung %s %s an %s: %v", date, debit, credit, err)
	}
}

// stubOpenItems ist eine feste Liste offener Posten. Die Restlaufzeitregel des
// § 268 Abs. 4 und 5 HGB hängt allein an Fälligkeit und Stichtag; sie an einer
// gebuchten Rechnung zu prüfen hieße, den Zahlungsverkehr mitzuprüfen.
//
// Dass der Abschluss die Stichtagssicht und nicht die heutige OP-Liste fragt,
// kann diese Liste nicht zeigen — dafür steht der Test weiter unten, der eine
// Rechnung bucht und sie im Folgejahr bezahlt.
type stubOpenItems []domain.OpenItem

func (s stubOpenItems) OpenItemsAt(context.Context, string) ([]domain.OpenItem, error) {
	return []domain.OpenItem(s), nil
}

// Die Vorjahresspalte des § 265 Abs. 2 HGB kommt aus den Buchungen des
// Vorjahres, nicht aus einer zweiten Abfrage desselben Jahres.
func TestStatementCarriesThePriorYearColumn(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2025-06-01", "1800", "2900", 5_000_000)
	env.post(t, "2026-06-01", "1800", "4400", 3_000_000)

	fs, err := env.statements(t).Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}

	if !fs.Statement.HasPrior {
		t.Fatal("die Bilanz meldet keine Vorjahresspalte, obwohl 2025 gebucht wurde")
	}
	if fs.Statement.PriorYear != 2025 {
		t.Errorf("die Vorjahresspalte gilt dem Jahr %d, erwartet 2025", fs.Statement.PriorYear)
	}
	if fs.Statement.TotalAssets != 3_000_000 {
		t.Errorf("die Bilanzsumme beträgt %s €, erwartet 30.000,00 €", fs.Statement.TotalAssets)
	}
	if fs.Statement.TotalAssetsPrior != 5_000_000 {
		t.Errorf("die Bilanzsumme des Vorjahres beträgt %s €, erwartet 50.000,00 €",
			fs.Statement.TotalAssetsPrior)
	}
	bank := fs.Statement.Line("aktiva.B.IV")
	if bank == nil || bank.PriorAmount != 5_000_000 {
		t.Errorf("die flüssigen Mittel tragen keine Vorjahreszahl: %+v", bank)
	}
	if fs.Statement.NetIncome != 3_000_000 {
		t.Errorf("das Jahresergebnis beträgt %s €, erwartet 30.000,00 €", fs.Statement.NetIncome)
	}
}

// Ohne Vorjahr bleibt die Spalte leer. Eine Spalte aus lauter Nullen läse sich
// wie eine Aussage über das Vorjahr.
func TestStatementLeavesThePriorColumnEmptyWithoutABookedPriorYear(t *testing.T) {
	env := newTestEnv(t)
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	fs, err := env.statements(t).Build(context.Background(), 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if fs.Statement.HasPrior {
		t.Error("die Bilanz meldet eine Vorjahresspalte, obwohl 2025 nicht gebucht wurde")
	}
}

// Der Kopf trägt die Pflichtangaben des § 264 Abs. 1a HGB — und benennt, was
// davon fehlt.
func TestStatementHeaderNamesTheMissingMandatoryData(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	fs, err := env.statements(t).Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if fs.Header.Reference != "§ 264 Abs. 1a HGB" {
		t.Errorf("der Kopf nennt %q als Fundstelle", fs.Header.Reference)
	}
	missing := strings.Join(fs.Header.Missing, ", ")
	for _, want := range []string{"Sitz", "Registergericht", "Registernummer"} {
		if !strings.Contains(missing, want) {
			t.Errorf("der Kopf meldet die fehlende Angabe %q nicht (gemeldet: %q)", want, missing)
		}
	}

	settings := repository.NewSettingsRepository(env.db)
	cfg, err := settings.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.Seat = "München"
	cfg.RegisterCourt = "Amtsgericht München"
	cfg.RegisterNumber = "HRB 123456"
	if err := settings.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Unternehmensdaten speichern: %v", err)
	}

	fs, err = env.statements(t).Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if len(fs.Header.Missing) != 0 {
		t.Errorf("der Kopf meldet noch %v als fehlend", fs.Header.Missing)
	}
	if fs.Header.RegisterNumber != "HRB 123456" {
		t.Errorf("die Registernummer im Kopf lautet %q", fs.Header.RegisterNumber)
	}
}

// Die Restlaufzeiten nach § 268 Abs. 4 und 5 HGB entstehen aus Fälligkeit und
// Stichtag.
func TestMaturitiesSplitOpenItemsByRemainingTerm(t *testing.T) {
	env := newTestEnv(t)
	svc := env.statements(t)
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	svc.SetOpenItemSource(stubOpenItems{
		// Forderung, fällig im Folgejahr: bis ein Jahr.
		{ContactType: domain.ContactTypeCustomer, DocumentDate: "2026-11-01",
			DueDate: "2027-01-15", OpenAmount: 100_000},
		// Forderung mit Restlaufzeit über einem Jahr.
		{ContactType: domain.ContactTypeCustomer, DocumentDate: "2026-11-01",
			DueDate: "2028-06-30", OpenAmount: 200_000},
		// Verbindlichkeit über fünf Jahren.
		{ContactType: domain.ContactTypeVendor, DocumentDate: "2026-02-01",
			DueDate: "2032-12-31", OpenAmount: 400_000},
		// Verbindlichkeit ohne Fälligkeit: keine Restlaufzeit.
		{ContactType: domain.ContactTypeVendor, DocumentDate: "2026-02-01",
			OpenAmount: 50_000},
		// Nach dem Stichtag entstanden: stand am Stichtag nicht offen.
		{ContactType: domain.ContactTypeVendor, DocumentDate: "2027-02-01",
			DueDate: "2027-03-01", OpenAmount: 900_000},
	})

	fs, err := svc.Build(context.Background(), 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if fs.Maturities.ClosingDate != "2026-12-31" {
		t.Fatalf("die Restlaufzeiten sind auf den %s bezogen, erwartet den 31.12.2026",
			fs.Maturities.ClosingDate)
	}

	rows := map[string]domain.MaturityRow{}
	for _, row := range fs.Maturities.Rows {
		rows[row.Key] = row
	}
	receivables := rows["receivables"]
	if receivables.UpToOneYear != 100_000 || receivables.OverOneYear != 200_000 {
		t.Errorf("Forderungen: bis ein Jahr %s €, über ein Jahr %s € — erwartet 1.000,00 € und 2.000,00 €",
			receivables.UpToOneYear, receivables.OverOneYear)
	}
	liabilities := rows["liabilities"]
	if liabilities.OverFiveYears != 400_000 {
		t.Errorf("Verbindlichkeiten über fünf Jahre: %s €, erwartet 4.000,00 €", liabilities.OverFiveYears)
	}
	if liabilities.OverOneYear != 400_000 {
		t.Errorf("eine Verbindlichkeit über fünf Jahre läuft auch über ein Jahr; ausgewiesen sind %s €",
			liabilities.OverOneYear)
	}
	if liabilities.Undated != 50_000 {
		t.Errorf("die Verbindlichkeit ohne Fälligkeit steht mit %s € unter „ohne Fälligkeit\", erwartet 500,00 €",
			liabilities.Undated)
	}
	if liabilities.Total != 450_000 {
		t.Errorf("die Verbindlichkeiten summieren sich auf %s €, erwartet 4.500,00 € — "+
			"der nach dem Stichtag entstandene Posten gehört nicht dazu", liabilities.Total)
	}
}

// Die Restlaufzeiten stehen auf dem Stichtag, nicht auf dem heutigen Stand.
//
// Eine Rechnung aus 2026, die im Januar 2027 bezahlt wird, war am 31.12.2026
// offen und gehört in die Angaben unter der Bilanz des Jahres 2026. Die
// operative OP-Liste kennt sie nicht mehr; die Angabe für ein abgeschlossenes
// Jahr schrumpfte sonst mit jeder Zahlung, die danach gebucht wird.
func TestMaturitiesUseTheOpenItemsAsOfTheClosingDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	payments := env.payments(t)
	vendor := env.vendor(t, "Lieferant", "DE", "")
	invoice := env.openPayable(t, vendor.ID, 100_000, domain.TaxRateStandard)

	if _, err := payments.Settle(ctx, PaymentRequest{
		PaymentAccount: domain.AccountBank,
		PaymentDate:    "2027-01-15",
		Allocations:    []AllocationRequest{{OpenItemEntryID: invoice.ID, SettledAmount: 119_000}},
	}); err != nil {
		t.Fatalf("Zahlung im Folgejahr: %v", err)
	}

	// Heute ist nichts mehr offen — das ist die operative Sicht, und sie ist
	// richtig.
	today, err := payments.OpenItems(ctx)
	if err != nil {
		t.Fatalf("offene Posten: %v", err)
	}
	if len(today) != 0 {
		t.Fatalf("nach der Zahlung sind %d Posten offen, erwartet keinen", len(today))
	}

	svc := env.statements(t)
	svc.SetOpenItemSource(payments)
	fs, err := svc.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}

	var liabilities domain.MaturityRow
	for _, row := range fs.Maturities.Rows {
		if row.Key == "liabilities" {
			liabilities = row
		}
	}
	if liabilities.Total != 119_000 {
		t.Errorf("die Verbindlichkeiten zum 31.12.2026 betragen %s €, erwartet 1.190,00 € — "+
			"die Zahlung vom 15.01.2027 gehört nicht in den Abschluss 2026", liabilities.Total)
	}
	if liabilities.UpToOneYear != 119_000 {
		t.Errorf("mit einer Restlaufzeit bis zu einem Jahr sind %s € ausgewiesen, erwartet 1.190,00 €",
			liabilities.UpToOneYear)
	}
}

// Die Größenklasse folgt aus den Merkmalen des Abschlusses, und die
// Arbeitnehmerzahl kommt aus dem Geschäftsjahr.
//
// Der Aufbau ist so gewählt, dass die Arbeitnehmerzahl entscheidet: die
// Umsatzerlöse liegen über der Schwelle der Kleinstgesellschaft, die
// Bilanzsumme darunter. Solange nur ein Merkmal überschritten ist, bleibt es
// bei der kleinsten Klasse — erst das zweite kippt sie (§ 267 Abs. 1 HGB).
func TestSizeClassUsesTheStatementAndTheRecordedHeadcount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	// 20 Mio € Umsatz, davon 19,7 Mio € Aufwand: Bilanzsumme 300.000 €.
	env.post(t, "2026-06-01", "1800", "4400", 2_000_000_000)
	env.post(t, "2026-07-01", "6300", "1800", 1_970_000_000)

	closing := env.closing(t)
	svc := NewStatementService(env.accounting, closing,
		repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db), 2026)

	sizeClass, err := svc.SizeClassFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Größenklasse: %v", err)
	}
	if sizeClass.Class != domain.SizeMicro {
		t.Errorf("die Größenklasse lautet %s, erwartet die Kleinstkapitalgesellschaft",
			sizeClass.Class.Label())
	}
	if sizeClass.Obligations.Depth != domain.DepthLetters {
		t.Errorf("die Gliederungstiefe lautet %s, erwartet die Buchstabengliederung",
			sizeClass.Obligations.Depth)
	}
	if sizeClass.Criteria.BalanceSheetTotal != 30_000_000 {
		t.Errorf("die Beurteilung rechnet mit der Bilanzsumme %s €, erwartet 300.000,00 €",
			sizeClass.Criteria.BalanceSheetTotal)
	}

	if _, err := closing.SetAverageEmployees(ctx, 2026, 20); err != nil {
		t.Fatalf("Arbeitnehmerzahl: %v", err)
	}
	sizeClass, err = svc.SizeClassFor(ctx, 2026)
	if err != nil {
		t.Fatalf("Größenklasse: %v", err)
	}
	if sizeClass.Class != domain.SizeSmall {
		t.Errorf("bei 20 Arbeitnehmern lautet die Klasse %s, erwartet die kleine Kapitalgesellschaft",
			sizeClass.Class.Label())
	}
	if sizeClass.Criteria.Employees != 20 {
		t.Errorf("die Beurteilung rechnet mit %d Arbeitnehmern", sizeClass.Criteria.Employees)
	}
	if sizeClass.Obligations.Depth != domain.DepthShort {
		t.Errorf("die Gliederungstiefe lautet %s, erwartet die verkürzte Gliederung",
			sizeClass.Obligations.Depth)
	}
}

// Die Zweijahresregel über drei Stichtage, durch den Dienst hindurch.
//
// § 267 Abs. 4 Satz 1 HGB vergleicht den Stichtag mit dem Vorjahr — und was
// gilt, wenn beide auseinandergehen, ist die *wirksame* Klasse des Vorjahres.
// Der Dienst muss dafür so weit zurück beurteilen, bis zwei aufeinanderfolgende
// Stichtage übereinstimmen; eine einzelne Vorjahresbeurteilung genügt nicht.
//
// Aufbau: die Umsatzerlöse bleiben null (immer unter der Schwelle), die
// Arbeitnehmerzahl liegt über jeder Schwelle. Damit entscheidet die Bilanzsumme
// allein, und sie lässt sich mit einer einzigen Buchung je Jahr setzen.
func TestSizeClassKeepsTheEffectiveClassOverThreeClosingDates(t *testing.T) {
	// 5 Mio € Bilanzsumme: kleine Gesellschaft. 10 Mio €: mittelgroß.
	const smallTotal = domain.Cents(500_000_000)
	const mediumTotal = domain.Cents(1_000_000_000)

	build := func(t *testing.T, totals map[int]domain.Cents) *domain.SizeClass {
		t.Helper()
		env := newTestEnv(t)
		ctx := context.Background()
		closing := env.closing(t)
		for year := 2024; year <= 2026; year++ {
			env.post(t, fmt.Sprintf("%d-06-01", year), "1800", "2900", totals[year])
			if _, err := closing.SetAverageEmployees(ctx, year, 300); err != nil {
				t.Fatalf("Arbeitnehmerzahl %d: %v", year, err)
			}
		}
		svc := NewStatementService(env.accounting, closing,
			repository.NewSettingsRepository(env.db), repository.NewAuditRepository(env.db), 2026)
		sizeClass, err := svc.SizeClassFor(ctx, 2026)
		if err != nil {
			t.Fatalf("Größenklasse: %v", err)
		}
		return sizeClass
	}

	back := build(t, map[int]domain.Cents{2024: smallTotal, 2025: mediumTotal, 2026: smallTotal})
	if back.Class != domain.SizeSmall {
		t.Errorf("klein/mittelgroß/klein ergibt %s, erwartet die kleine Gesellschaft — "+
			"mittelgroß war die Gesellschaft nie an zwei aufeinanderfolgenden Stichtagen",
			back.Class.Label())
	}
	if len(back.History) != 3 {
		t.Errorf("die Beurteilung reicht über %d Stichtage, erwartet drei", len(back.History))
	}

	up := build(t, map[int]domain.Cents{2024: smallTotal, 2025: mediumTotal, 2026: mediumTotal})
	if up.Class != domain.SizeMedium {
		t.Errorf("klein/mittelgroß/mittelgroß ergibt %s, erwartet die mittelgroße Gesellschaft",
			up.Class.Label())
	}
	if !up.Obligations.AuditRequired {
		t.Error("die mittelgroße Gesellschaft ist nach § 316 Abs. 1 HGB prüfungspflichtig")
	}
}

// Ein Konto mit Saldo, dem die Gliederung keine Position zuordnen kann, wird
// namentlich gemeldet — durch den Dienst hindurch, nicht nur in der reinen
// Gliederung.
func TestStatementReportsAnAccountWithoutAPosition(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	// Der Zuordnungstabelle die Position nehmen, statt ein erfundenes Konto
	// anzulegen: ein Konto außerhalb des Katalogs fiele in den Bereich einer
	// Katalogzeile und stünde dann zweimal in der Bilanz.
	accounts := repository.NewAccountRepository(env.db)
	bank, err := accounts.FindByNumber(ctx, "1800")
	if err != nil {
		t.Fatalf("Konto 1800: %v", err)
	}
	bank.PositionID = ""
	if err := accounts.Update(ctx, bank); err != nil {
		t.Fatalf("Konto 1800 speichern: %v", err)
	}

	fs, err := env.statements(t).Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	report := fs.Statement.Assignment
	if !report.HasFindings() {
		t.Fatal("der Zuordnungsbericht meldet nichts, obwohl ein Konto ohne Position gebucht ist")
	}
	found := false
	for _, acc := range report.Unassigned {
		if acc.Number == "1800" {
			found = true
			if acc.Amount != 1_000_000 {
				t.Errorf("das nicht zugeordnete Konto steht mit %s €, erwartet 10.000,00 €", acc.Amount)
			}
		}
	}
	if !found {
		t.Errorf("das Konto 1800 fehlt im Zuordnungsbericht: %+v", report.Unassigned)
	}
	// Es bleibt in der Bilanz sichtbar, damit sie aufgeht.
	if line := fs.Statement.Line("aktiva.X"); line == nil || line.Amount != 1_000_000 {
		t.Errorf("die Position „Nicht zugeordnet\" führt den Saldo nicht: %+v", line)
	}
}

// Ein regulärer Abschluss meldet keinen Befund.
//
// Der Zuordnungsbericht wäre wertlos, wenn er bei jedem Abschluss anschlüge:
// ein Aufwandskonto mit Habensaldo ist eine Erstattung und keine Auffälligkeit.
func TestAssignmentReportStaysQuietOnAnOrdinaryStatement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	env.post(t, "2026-07-01", "6300", "1800", 200_000)
	// Erstattung einer Betriebsausgabe: das Aufwandskonto steht im Haben.
	env.post(t, "2026-08-01", "1800", "6540", 50_000)
	// Verlustvortrag: die Position „Gewinnvortrag/Verlustvortrag" nennt beide
	// Richtungen ausdrücklich.
	env.post(t, "2026-08-02", "2978", "1800", 30_000)

	fs, err := env.statements(t).Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if fs.Statement.Assignment.HasFindings() {
		t.Errorf("der Zuordnungsbericht meldet einen Befund: nicht zugeordnet %+v, falsches Vorzeichen %+v",
			fs.Statement.Assignment.Unassigned, fs.Statement.Assignment.WrongSign)
	}
}

// Was den Abschluss verändert oder aus dem Haus gibt, steht im Protokoll:
// die Arbeitnehmerzahl als drittes Merkmal des § 267 Abs. 1 HGB und jede
// Ausgabe als Datei.
func TestHeadcountAndExportsAreAudited(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	if _, err := env.closing(t).SetAverageEmployees(ctx, 2026, 7); err != nil {
		t.Fatalf("Arbeitnehmerzahl: %v", err)
	}
	if _, err := env.statements(t).ExportCSV(ctx, 2026, domain.DepthFull); err != nil {
		t.Fatalf("CSV: %v", err)
	}

	entries, err := repository.NewAuditRepository(env.db).FindAll(ctx, 100)
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	var headcount, export bool
	for _, e := range entries {
		if e.EntityType == "FISCAL_YEAR" && strings.Contains(e.Details, "Arbeitnehmerzahl") {
			headcount = true
		}
		if e.EntityType == "JAHRESABSCHLUSS" && e.Action == domain.AuditActionExport {
			export = true
		}
	}
	if !headcount {
		t.Error("die Änderung der Arbeitnehmerzahl steht nicht im Protokoll")
	}
	if !export {
		t.Error("die Ausgabe des Jahresabschlusses steht nicht im Protokoll")
	}
}

// Ohne ausdrückliche Tiefe gilt die Tiefe der Größenklasse — das ist die Tiefe,
// in der offenzulegen ist.
func TestDepthDefaultsToTheSizeClass(t *testing.T) {
	env := newTestEnv(t)
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	fs, err := env.statements(t).Build(context.Background(), 2026, "")
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if fs.Statement.Depth != domain.DepthLetters {
		t.Errorf("die Gliederungstiefe lautet %s, erwartet die Buchstabengliederung", fs.Statement.Depth)
	}
	if fs.Statement.Line("aktiva.B.IV") != nil {
		t.Error("die Buchstabengliederung zeigt eine römische Ziffer")
	}
}

// Die Fristen kommen aus dem Backend, mit Datum und Norm.
func TestDeadlinesComeFromTheClosingDateAndTheSizeClass(t *testing.T) {
	env := newTestEnv(t)
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	deadlines, err := env.statements(t).Deadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("Fristen: %v", err)
	}
	if len(deadlines) != 2 {
		t.Fatalf("es entstehen %d Termine, erwartet Aufstellung und Offenlegung", len(deadlines))
	}
	if deadlines[0].DueDate != "2027-06-30" {
		t.Errorf("die Aufstellungsfrist endet am %s, erwartet den 30.06.2027", deadlines[0].DueDate)
	}
	if deadlines[1].DueDate != "2027-12-31" {
		t.Errorf("die Offenlegungsfrist endet am %s, erwartet den 31.12.2027", deadlines[1].DueDate)
	}
	for _, d := range deadlines {
		if d.Reference == "" {
			t.Errorf("der Termin %q nennt keine Norm", d.Title)
		}
	}
}

// Die CSV-Ausgabe enthält jede Position der Gliederung und die Summenzeilen.
func TestCSVContainsEveryLineOfTheStatement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	env.post(t, "2026-07-01", "6300", "1800", 200_000)

	svc := env.statements(t)
	csv, err := svc.ExportCSV(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	fs, err := svc.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}

	for _, group := range [][]domain.StatementLine{
		fs.Statement.Assets, fs.Statement.Liabilities, fs.Statement.Income, fs.Statement.Statistical,
	} {
		for _, line := range group {
			if !strings.Contains(csv, `"`+line.Key+`"`) {
				t.Errorf("die CSV-Ausgabe enthält die Position %s (%s) nicht", line.Key, line.Label)
			}
		}
	}
	for _, want := range []string{"summe.aktiva", "summe.passiva", "jahresergebnis", "restlaufzeiten"} {
		if !strings.Contains(csv, want) {
			t.Errorf("die CSV-Ausgabe enthält %q nicht", want)
		}
	}
	// Deutscher Zahlensatz: Dezimalkomma.
	if !strings.Contains(csv, `"8000,00"`) {
		t.Errorf("die CSV-Ausgabe schreibt das Jahresergebnis nicht als \"8000,00\":\n%s", csv)
	}
	if !strings.Contains(csv, ";") {
		t.Error("die CSV-Ausgabe trennt nicht mit Semikolon")
	}
}

// Ohne ausdrückliche Tiefe gliedert die CSV voll — auch für eine
// Kleinstgesellschaft, deren Bilanz nur auf Buchstabenebene offenzulegen ist.
//
// Die Bridge fragt ohne Tiefe. Bekäme sie die Tiefe der Größenklasse, enthielte
// die Tabelle einer Kleinstgesellschaft nur die Buchstabenzeilen — als
// Datenexport wäre sie damit unbrauchbar, und die Zahl, die jemand sucht, stünde
// nicht darin.
func TestCSVFallsBackToTheFullDepth(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)

	svc := env.statements(t)
	fs, err := svc.Build(ctx, 2026, "")
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}
	if fs.Statement.Depth != domain.DepthLetters {
		t.Fatalf("die Ansicht gliedert %s, erwartet die Buchstabengliederung der Kleinstgesellschaft",
			fs.Statement.Depth)
	}

	csv, err := svc.ExportCSV(ctx, 2026, "")
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	if !strings.Contains(csv, `"aktiva.B.IV"`) {
		t.Error("die CSV-Ausgabe ohne Tiefenangabe enthält die römischen Ziffern nicht")
	}
	if !strings.Contains(csv, `"guv.17"`) {
		t.Error("die CSV-Ausgabe ohne Tiefenangabe enthält die Staffel der GuV nicht vollständig")
	}
}

// Die PDF-Ausgabe entsteht über denselben Typst-Compiler wie die Rechnung.
func TestStatementPDFIsProduced(t *testing.T) {
	if testing.Short() {
		t.Skip("das Übersetzen des WASM-Moduls dauert mehrere Sekunden")
	}
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	env.post(t, "2026-07-01", "6300", "1800", 200_000)

	svc := env.statements(t)
	renderer := invoice.NewRenderer()
	defer func() { _ = renderer.Close(ctx) }()
	svc.SetRenderer(renderer)

	pdf, err := svc.ExportPDF(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF") {
		t.Fatalf("die Ausgabe beginnt mit %q, erwartet %%PDF", string(pdf[:min(8, len(pdf))]))
	}
}

// Der Zuordnungsbericht der E-Bilanz führt die Konten mit Saldo auf und lässt
// den Export zu, solange nichts fehlt.
func TestEBilanzMappingReportCoversTheBookedAccounts(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	env.post(t, "2026-07-01", "6300", "1800", 200_000)

	ebilanzSvc := NewEBilanzService(
		env.statements(t), repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db), 2026)

	report, err := ebilanzSvc.MappingReport(ctx, 2026)
	if err != nil {
		t.Fatalf("Zuordnungsbericht: %v", err)
	}
	if !report.CanExport {
		t.Fatalf("der Bericht blockiert den Export: %+v", report.Blocking)
	}
	accounts := map[string]bool{}
	for _, row := range report.Rows {
		accounts[row.Account] = true
		if row.Element == "" {
			t.Errorf("das Konto %s hat kein Taxonomie-Element", row.Account)
		}
	}
	// Die Erlöse stehen im Katalog als Bereich „4400-4409"; genau diese Zeile
	// trägt den gebuchten Umsatz.
	for _, want := range []string{"1800", "4400-4409", "6300"} {
		if !accounts[want] {
			t.Errorf("das gebuchte Konto %s fehlt im Kontennachweis", want)
		}
	}

	xbrl, err := ebilanzSvc.ExportXBRL(ctx, 2026)
	if err != nil {
		t.Fatalf("E-Bilanz: %v", err)
	}
	if !strings.Contains(xbrl, "de-gaap-ci:is.netSales") {
		t.Error("die Instanz enthält die Umsatzerlöse nicht")
	}
}

// Die Regel des § 265 Abs. 8 HGB steht im Aufbau. Das Dokument lässt den leeren
// Posten weg; die CSV behält ihn als Datenzeile und schreibt den Merker daneben,
// damit Ansicht, PDF und Tabelle dieselbe Regel zeigen.
func TestOmittedLinesAreLeftOutOfThePDFAndMarkedInTheCSV(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.post(t, "2026-06-01", "1800", "4400", 1_000_000)
	env.post(t, "2026-07-01", "6300", "1800", 200_000)

	svc := env.statements(t)
	fs, err := svc.Build(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("Jahresabschluss: %v", err)
	}

	// Ein Posten, auf den nichts gebucht wurde, und einer mit Betrag.
	empty := fs.Statement.Line("aktiva.A.I.1")
	if empty == nil || !empty.Omitted {
		t.Fatalf("die immateriellen Vermögensgegenstände tragen nichts und müssten entfallen: %+v", empty)
	}
	filled := fs.Statement.Line("aktiva.B.IV")
	if filled == nil || filled.Omitted {
		t.Fatalf("die flüssigen Mittel tragen einen Betrag und dürfen nicht entfallen: %+v", filled)
	}

	document := statementTypst(fs)
	if strings.Contains(document, typstText(empty.Label)) {
		t.Errorf("das Dokument enthält den entfallenden Posten %q (§ 265 Abs. 8 HGB)", empty.Label)
	}
	if !strings.Contains(document, typstText(filled.Label)) {
		t.Errorf("das Dokument enthält den Posten %q nicht", filled.Label)
	}

	csv, err := svc.ExportCSV(ctx, 2026, domain.DepthFull)
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	if !strings.Contains(csv, `"Entfällt"`) {
		t.Error("die CSV-Ausgabe nennt die Spalte „Entfällt\" nicht")
	}
	for _, line := range fs.Statement.Assets {
		want := `"` + line.Key + `";"` + line.Ordinal + `";"` + line.Label + `";"` +
			fmt.Sprintf("%d", line.Level) + `";"` + germanAmount(line.Amount) + `";"` +
			germanAmount(line.PriorAmount) + `";"` + omittedMark(line.Omitted) + `"`
		if !strings.Contains(csv, want) {
			t.Errorf("die CSV-Ausgabe führt die Position %s nicht mit dem Merker %q: erwartet %s",
				line.Key, omittedMark(line.Omitted), want)
		}
	}
}
