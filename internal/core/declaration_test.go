package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
)

// Die drei Jahreserklärungen lassen sich aus einem offenen Mandanten rechnen;
// jede Zeile des Katalogs hat eine Rechnung, und das Ergebnis folgt den
// Büchern.
func TestDeclarationsFromTheBooks(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		_, err := post(ctx, s)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range declaration.Kinds() {
		d, err := tenant.Declaration(agentCtx(t), kind, 2026)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		for _, f := range d.Findings {
			if strings.HasPrefix(f.Code, "intern_") {
				t.Errorf("%s: %s", kind, f.Text)
			}
		}
		if d.FiguresHash == "" || d.Status != declaration.StatusDraft {
			t.Errorf("%s: Hash %q, Stand %s", kind, d.FiguresHash, d.Status)
		}
	}
	kst, err := tenant.Declaration(uiCtx, declaration.KindCorporateTax, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if got := kst.Amount("kst.gk.z11"); got != -10000 {
		t.Errorf("Jahresergebnis %s €, erwartet −100,00 € aus der Testbuchung", got)
	}
	if _, err := tenant.Declaration(uiCtx, declaration.KindCorporateTax, 2019); err == nil {
		t.Errorf("Erklärung für ein Jahr ohne Geschäftsjahr")
	}
}

// Ein Agent setzt Angaben mit Rückfrage. Bei Zeilen, die dem Menschen
// vorbehalten sind, bleibt es ein Vorschlag; bestätigen kann nur die App.
func TestDeclarationInputsByChannel(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)

	art, err := tenant.SetDeclarationInput(agent, service.InputRequest{
		Kind: declaration.KindVatYear, Year: 2026, Key: "ust.z7.art", Value: declaration.Text("Softwareentwicklung")})
	if err != nil {
		t.Fatal(err)
	}
	if !art.Confirmed || art.Channel != "agent:claude-desktop" {
		t.Errorf("Angabe des Agenten zu seiner Zeile: %+v", art)
	}
	loss, err := tenant.SetDeclarationInput(agent, service.InputRequest{
		Kind: declaration.KindCorporateTax, Year: 2026, Key: "kst.v.z14", Value: declaration.Cents(500000),
		Source: declaration.InputFromNotice})
	if err != nil {
		t.Fatal(err)
	}
	if loss.Confirmed {
		t.Errorf("Verlustvortrag vom Agenten gilt ohne Bestätigung")
	}
	d, err := tenant.Declaration(agent, declaration.KindCorporateTax, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if l, _ := d.Line("kst.v.z14"); l.Status != declaration.StatusProposed || l.Value.Amount() != 500000 {
		t.Errorf("Vorschlag in der Erklärung: %+v", l)
	}

	if _, err := tenant.ConfirmDeclarationInput(agent, declaration.KindCorporateTax, 2026, "kst.v.z14"); !errors.Is(err, ErrAppOnly) {
		t.Errorf("Agent bestätigt selbst: %v", err)
	}
	if _, err := tenant.ConfirmDeclarationInput(uiCtx, declaration.KindCorporateTax, 2026, "kst.v.z14"); err != nil {
		t.Fatal(err)
	}
	d, _ = tenant.Declaration(agent, declaration.KindCorporateTax, 2026)
	if l, _ := d.Line("kst.v.z14"); l.Status != declaration.StatusInput {
		t.Errorf("bestätigte Angabe: Stand %s", l.Status)
	}
	if err := tenant.DeleteDeclarationInput(agent, declaration.KindCorporateTax, 2026, "kst.v.z14"); err == nil {
		t.Errorf("Agent nimmt eine bestätigte Angabe des Menschen zurück")
	}

	for name, req := range map[string]service.InputRequest{
		"Zeile aus den Büchern": {Kind: declaration.KindVatYear, Year: 2026, Key: "ust.z22.kz177", Value: declaration.Cents(100)},
		"unbekannte Zeile":      {Kind: declaration.KindVatYear, Year: 2026, Key: "ust.z999", Value: declaration.Cents(100)},
		"Text statt Betrag":     {Kind: declaration.KindCorporateTax, Year: 2026, Key: "kst.v.z14", Value: declaration.Text("viel")},
		"Cent in vollen Euro":   {Kind: declaration.KindVatYear, Year: 2026, Key: "ust.z23.kz178", Value: declaration.Cents(150)},
	} {
		if _, err := tenant.SetDeclarationInput(uiCtx, req); err == nil {
			t.Errorf("%s wurde angenommen", name)
		}
	}

	var log []domain.AuditLogEntry
	if err := tenant.Read(uiCtx, func(ctx context.Context, s *Services) (err error) {
		log, err = s.Repos.Audit.FindFiltered(ctx, 50, domain.AuditFilter{EntityType: "DECLARATION_INPUT"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(log) < 3 {
		t.Errorf("Protokoll der Angaben: %d Einträge, erwartet mindestens 3", len(log))
	}
}

// Der Agent exportiert; die Übermittlung erfasst nur die App, mit
// Transferticket, Datum und dem Hash der übermittelten Zahlen.
func TestDeclarationExportAndSubmission(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	agent := agentCtx(t)
	dir := t.TempDir()

	exp, err := tenant.ExportDeclaration(agent, declaration.KindVatYear, 2026, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exp.Path)
	if err != nil {
		t.Fatal(err)
	}
	var file declaration.Declaration
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	if file.Schema != declaration.Schema || file.FiguresHash != exp.FiguresHash || file.Status != declaration.StatusExported {
		t.Errorf("Exportdatei: Schema %s, Hash %s, Stand %s", file.Schema, file.FiguresHash, file.Status)
	}
	d, _ := tenant.Declaration(agent, declaration.KindVatYear, 2026)
	if d.Status != declaration.StatusExported {
		t.Errorf("Stand nach dem Export: %s", d.Status)
	}

	today := time.Now().Format(domain.DateLayout)
	req := service.SubmissionRequest{Kind: declaration.KindVatYear, Year: 2026, SubmittedAt: today,
		TransferTicket: "et9K2LX4QZ", FiguresHash: exp.FiguresHash}
	if _, err := tenant.SubmitDeclaration(agent, req); !errors.Is(err, ErrAppOnly) {
		t.Fatalf("Agent erfasst die Übermittlung: %v", err)
	}
	if exp.Blocking > 0 {
		if _, err := tenant.SubmitDeclaration(uiCtx, req); err == nil {
			t.Fatalf("Übermittlung trotz blockierender Befunde ohne Begründung erfasst")
		}
		req.OverrideReason = "In Mein ELSTER ergänzt und geprüft"
	}
	bad := req
	bad.FiguresHash = strings.Repeat("0", 64)
	if _, err := tenant.SubmitDeclaration(uiCtx, bad); err == nil {
		t.Errorf("Übermittlung mit fremdem Zahlen-Hash erfasst")
	}
	rec, err := tenant.SubmitDeclaration(uiCtx, req)
	if err != nil {
		t.Fatal(err)
	}
	if rec.TransferTicket != "et9K2LX4QZ" || rec.SubmittedAt != today {
		t.Errorf("Übermittlung: %+v", rec)
	}
	d, _ = tenant.Declaration(agent, declaration.KindVatYear, 2026)
	if d.Status != declaration.StatusSubmitted || d.Submission == nil || d.Submission.TransferTicket != "et9K2LX4QZ" {
		t.Errorf("Stand nach der Übermittlung: %s %+v", d.Status, d.Submission)
	}

	// Der Pflichtenkalender nimmt die Übermittlung als Nachweis.
	var o *domain.Obligation
	if err := tenant.Read(agent, func(ctx context.Context, s *Services) (err error) {
		o, err = s.Obligations.Get(ctx, string(domain.ObligationAnnualVatReturn)+".2026", today)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.ObligationDone || o.Proof == nil || !o.Proof.Automatic ||
		o.Proof.Kind != domain.EvidenceTransferTicket || o.Proof.Reference != "et9K2LX4QZ" || o.Proof.Date != today {
		t.Errorf("Pflicht nach der Übermittlung: %s %+v", o.Status, o.Proof)
	}
}

// Im Prüfermodus bleibt der Export möglich; Angaben nimmt die Buchführung
// nicht auf.
func TestDeclarationsInExaminerMode(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	until := time.Now().AddDate(0, 0, 10).Format(domain.DateLayout)
	if _, err := tenant.EnableExaminer(uiCtx, until, "Außenprüfung"); err != nil {
		t.Fatal(err)
	}
	if _, err := tenant.ExportDeclaration(uiCtx, declaration.KindCorporateTax, 2026, t.TempDir()); err != nil {
		t.Errorf("Export im Prüfermodus: %v", err)
	}
	_, err := tenant.SetDeclarationInput(uiCtx, service.InputRequest{
		Kind: declaration.KindVatYear, Year: 2026, Key: "ust.z7.art", Value: declaration.Text("Beratung")})
	if err == nil {
		t.Errorf("Angabe im Prüfermodus angenommen")
	}
}

// Die Umsatzsteuererklärung übernimmt das Kennziffernblatt des Kalenderjahres.
func TestVatYearDeclarationFromTheJournal(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Ventures GmbH")
	tenant := openTenant(t, h)
	entry := func(date string, lines ...domain.JournalLine) *domain.JournalEntry {
		return &domain.JournalEntry{BookingDate: date, DocumentDate: date, ServiceDateFrom: date, ServiceDateTo: date,
			Description: "Test", Source: domain.EntrySourceManual, TaxTreatment: domain.TaxTreatmentDomestic, Lines: lines}
	}
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		if _, err := s.Journal.Post(ctx, entry("2026-02-10",
			domain.JournalLine{Side: domain.SideDebit, Account: domain.AccountBank, Amount: 1_190_000},
			domain.JournalLine{Side: domain.SideCredit, Account: "4400", Amount: 1_000_000},
			domain.JournalLine{Side: domain.SideCredit, Account: domain.AccountUmsatzsteuer19, Amount: 190_000, TaxKey: "UST19", TaxBase: 1_000_000},
		)); err != nil {
			return err
		}
		_, err := s.Journal.Post(ctx, entry("2026-03-01",
			domain.JournalLine{Side: domain.SideDebit, Account: "6815", Amount: 10_000},
			domain.JournalLine{Side: domain.SideDebit, Account: domain.AccountVorsteuer19, Amount: 1_900, TaxKey: "VST19", TaxBase: 10_000},
			domain.JournalLine{Side: domain.SideCredit, Account: domain.AccountBank, Amount: 11_900},
		))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	d, err := tenant.Declaration(uiCtx, declaration.KindVatYear, 2026)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]domain.Cents{
		"ust.z22.kz177": 1_000_000, "ust.z22.steuer": 190_000, "ust.z79.kz320": 1_900,
		"ust.z118.kz816": 188_100, "ust.z119": 0, "ust.z120.kz820": 188_100,
	} {
		if got := d.Amount(key); got != want {
			t.Errorf("%s: %s €, erwartet %s €", key, got, want)
		}
	}
	if l, _ := d.Line("ust.z22.kz177"); len(l.Origin.EntryIDs) != 1 {
		t.Errorf("Zeile 22 ohne Buchung als Herkunft: %+v", l.Origin)
	}
}

// Bezüge, die über die Anlagenkartei mit einem Anteil verknüpft sind, gehen
// in § 8b KStG ein, auch wenn sie auf 7000 statt auf einem §-8b-Konto stehen.
// Die im Jahr erworbene Tochter zeigt die zwei Stichtage: bei der
// Körperschaftsteuer wirkt der Erwerb auf den Jahresbeginn zurück, bei der
// Gewerbesteuer nicht (§ 8 Nr. 5 GewStG).
func TestDeclarationsUseTheAssetRegister(t *testing.T) {
	h := newHost(t)
	newTenant(t, h, "Pfennig Holding GmbH")
	tenant := openTenant(t, h)
	if err := tenant.Write(uiCtx, func(ctx context.Context, s *Services) error {
		asset, err := s.Assets.Save(ctx, &domain.FixedAsset{
			Name: "Tochter A GmbH", Class: domain.AssetClassFinancial, Account: "0850",
			AcquisitionDate: "2026-02-01", AcquisitionCost: 2_500_000, Method: domain.DepreciationNone,
			TaxPrivileged: true, HoldingPermille: 1000,
		})
		if err != nil {
			return err
		}
		_, err = s.Assets.BookAssetIncome(ctx, service.AssetIncomeRequest{
			AssetID: asset.ID, Date: "2026-05-15", Amount: 1_000_000,
			Settlement: service.SettlementPaid, PaymentAccount: domain.AccountBank, WithholdingTax: 263_750,
		})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	kst, err := tenant.Declaration(uiCtx, declaration.KindCorporateTax, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if got := kst.Amount("kst.gk.8b_bezuege"); got != 1_000_000 {
		t.Errorf("Bezüge nach § 8b: %s €, erwartet 10.000 €", got)
	}
	if l, _ := kst.Line("kst.gk.8b_streubesitz"); l.Status != declaration.StatusProposed || l.Value.Amount() != 0 {
		t.Errorf("Streubesitz bei 100 %%: %+v", l)
	}
	if got := kst.Amount("kst.gk.8b_5_bezuege"); got != 50_000 {
		t.Errorf("5 %% nicht abziehbar: %s €", got)
	}
	if got := kst.Amount("kst.gk.kst"); got != 263_750 {
		t.Errorf("Kapitalertragsteuer hinzugerechnet: %s €", got)
	}
	found := false
	for _, f := range kst.Findings {
		found = found || f.Code == "bezuege_ohne_8b_konto"
	}
	if !found {
		t.Errorf("Hinweis auf das Konto 7000 fehlt")
	}

	gewst, err := tenant.Declaration(uiCtx, declaration.KindTradeTax, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if l, _ := gewst.Line("gewst.8_5"); l.Value.Amount() != 950_000 {
		t.Errorf("Hinzurechnung § 8 Nr. 5 für die im Jahr erworbene Tochter: %+v", l)
	}
}
