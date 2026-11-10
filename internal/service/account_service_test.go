package service

import (
	"context"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/repository"
)

// Das eigene Konto ist sofort bebuchbar, und die Sperre wirkt sofort
// (BEL-06 K2, Entscheidung 4).
//
// Der Buchungsweg hält den Kontenplan zwischengespeichert. Ohne die
// Invalidierung wäre das neue Konto für ihn „im SKR04 nicht vorhanden" und ein
// gesperrtes weiter bebuchbar — beides erst nach einem Neustart richtig, und
// das ist keine Kontenpflege.
func TestCustomAccountIsPostableRightAfterItIsCreated(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Eine erste Buchung füllt den Zwischenspeicher des Kontenplans. Ohne sie
	// prüfte der Test einen Kontenplan, der ohnehin frisch gelesen wird.
	if _, err := env.journal.Post(ctx, simpleEntry("6300", domain.AccountBank, 1000)); err != nil {
		t.Fatalf("erste Buchung: %v", err)
	}

	accounts := NewAccountService(
		repository.NewAccountRepository(env.db), repository.NewAuditRepository(env.db))
	accounts.SetChartInvalidator(env.journal)

	const position = "guv.guv_8.sonstige_betriebliche_aufwendungen"
	if _, err := accounts.CreateCustom(ctx, CustomAccountRequest{
		Number: "6011", Name: "Fremdleistungen Fotografie", HGBPosition: position,
	}); err != nil {
		t.Fatalf("eigenes Konto: %v", err)
	}

	if _, err := env.journal.Post(ctx, simpleEntry("6011", domain.AccountBank, 2500)); err != nil {
		t.Fatalf("auf das eigene Konto muss sofort gebucht werden können: %v", err)
	}

	if _, err := accounts.SetBlocked(ctx, "6011", true, "Aufwandsart entfällt"); err != nil {
		t.Fatalf("Konto sperren: %v", err)
	}
	blocked := simpleEntry("6011", domain.AccountBank, 2500)
	blocked.Description = "Buchung nach der Sperre"
	if _, err := env.journal.Post(ctx, blocked); err == nil {
		t.Error("ein gesperrtes Konto nimmt keine Buchung mehr auf")
	}

	// Und die Freigabe wirkt ebenso sofort: eine Sperre, die sich nicht
	// zurücknehmen ließe, wäre eine Löschung.
	if _, err := accounts.SetBlocked(ctx, "6011", false, ""); err != nil {
		t.Fatalf("Konto freigeben: %v", err)
	}
	again := simpleEntry("6011", domain.AccountBank, 2500)
	again.Description = "Buchung nach der Freigabe"
	if _, err := env.journal.Post(ctx, again); err != nil {
		t.Errorf("nach der Freigabe ist das Konto wieder bebuchbar: %v", err)
	}
}

// Das eigene Konto trägt seine Gliederungsposition — und damit erscheint es in
// Bilanz und GuV und in der E-Bilanz (BEL-06 K2).
func TestCustomAccountCarriesItsPositionIntoTheStatement(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	svc := NewAccountService(
		repository.NewAccountRepository(env.db), repository.NewAuditRepository(env.db))

	const position = "guv.guv_8.sonstige_betriebliche_aufwendungen"
	account, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "6011", Name: "Fremdleistungen Fotografie", HGBPosition: position,
	})
	if err != nil {
		t.Fatalf("eigenes Konto: %v", err)
	}
	if !account.IsCustom || !account.IsActive {
		t.Error("das angelegte Konto ist ein eigenes und ist bebuchbar")
	}
	if account.PositionID != position || account.HGBCode != "GuV.8" {
		t.Errorf("Position %q / %q — erwartet die gewählte", account.PositionID, account.HGBCode)
	}
	if account.Type != domain.AccountTypeExpense {
		t.Errorf("Kontoart %q — die Position sagt Aufwand", account.Type)
	}

	// Eine belegte Nummer wird nicht überschrieben: das zerstörte die
	// Zuordnung des Kontenrahmens.
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "6300", Name: "Doppelt", HGBPosition: position,
	}); err == nil {
		t.Error("eine belegte Kontonummer darf nicht ein zweites Mal vergeben werden")
	}
	// Ohne Position gibt es kein Konto: es fehlte im Abschluss.
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "6012", Name: "Ohne Position",
	}); err == nil {
		t.Error("ein Konto ohne Gliederungsposition erschiene in keinem Abschluss")
	}
	// Die Klasse 9 bleibt den Vortrags- und statistischen Konten vorbehalten.
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "9999", Name: "Statistisch", HGBPosition: position,
	}); err == nil {
		t.Error("die Klasse 9 ist den Vortrags- und statistischen Konten vorbehalten")
	}
	// Die Klasse 0 ist die des Anlagevermögens: mit einer Position des
	// Anlagevermögens ist ein eigenes Anlagenkonto anlegbar, mit einer
	// Aufwandsposition nicht.
	const assetPosition = "bilanz.aktiva_a_ii.technische_anlagen_und_maschinen"
	asset, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "0499", Name: "Prüfstand", HGBPosition: assetPosition,
	})
	if err != nil {
		t.Fatalf("eigenes Anlagenkonto: %v", err)
	}
	if asset.PositionID != assetPosition {
		t.Errorf("Position %q — erwartet die des Anlagevermögens", asset.PositionID)
	}
	if _, err := svc.CreateCustom(ctx, CustomAccountRequest{
		Number: "0498", Name: "Aufwand in der Anlagenklasse", HGBPosition: position,
	}); err == nil {
		t.Error("eine Aufwandsposition erlaubt kein Konto der Klasse 0")
	}

	// Gebucht wird darauf wie auf jedes andere Konto, und es erscheint in der Bilanz.
	account.DebitSum = 250_000
	bank, err := repository.NewAccountRepository(env.db).FindByNumber(ctx, domain.AccountBank)
	if err != nil {
		t.Fatalf("Bankkonto: %v", err)
	}
	bank.CreditSum = 250_000
	stmt, err := accounting.BuildStatement(
		[]domain.Account{*account, *bank}, nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz und GuV: %v", err)
	}
	found := false
	for _, line := range stmt.Income {
		for _, a := range line.Accounts {
			if a.Number == "6011" {
				found = true
			}
		}
	}
	if !found {
		t.Error("das eigene Konto steht in keiner Position der GuV")
	}

	// Und die E-Bilanz bildet es ab: der Kontennachweis führt es unter einer
	// Position der GuV.
	res, err := ebilanz.Generate(ebilanz.Input{
		Settings:  companySettingsFor(t, env),
		Statement: stmt,
		Accounts:  []domain.Account{*account, *bank},
		Kind:      ebilanz.KindAnnual, FiscalYearBegin: "2026-01-01", FiscalYearEnd: "2026-12-31",
	})
	if err != nil {
		t.Fatalf("E-Bilanz: %v", err)
	}
	placed := false
	for _, row := range res.Report.Rows {
		if row.Account == "6011" {
			placed = row.Element != "" && row.Finding == ""
		}
	}
	if !placed {
		t.Error("die E-Bilanz ordnet das eigene Konto keiner Position zu")
	}
	for _, row := range res.Report.Blocking {
		if strings.Contains(row.Account, "6011") {
			t.Errorf("das eigene Konto gilt als nicht zuordenbar: %+v", row)
		}
	}

	// Gesperrt wird mit Grund, und ein Konto des SKR04 überhaupt nicht.
	if _, err := svc.SetBlocked(ctx, "6011", true, ""); err == nil {
		t.Error("zum Sperren gehört eine Begründung")
	}
	blocked, err := svc.SetBlocked(ctx, "6011", true, "Leistung wird nicht mehr eingekauft")
	if err != nil {
		t.Fatalf("sperren: %v", err)
	}
	if blocked.IsActive {
		t.Error("das gesperrte Konto ist nicht mehr bebuchbar")
	}
	if _, err := svc.SetBlocked(ctx, "6300", true, "Grund"); err == nil {
		t.Error("ein Konto des SKR04 wird nicht gesperrt")
	}
}

// companySettingsFor liest die Unternehmensdaten der Testumgebung.
func companySettingsFor(t *testing.T, env *testEnv) *domain.CompanySettings {
	t.Helper()
	cfg, err := repository.NewSettingsRepository(env.db).
		GetCompanySettings(context.Background())
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	return cfg
}
