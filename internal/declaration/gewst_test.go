package declaration

import (
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
)

func tradeCommon(year int, kstInputs []Input, inputs ...Input) TradeCommon {
	base := []Input{
		confirmed("gewst.betriebsstaetten", Bool(false)),
		confirmed("gewst.verlust_vorjahr", Cents(0)),
	}
	return TradeCommon{Common: common(year, append(base, inputs...)...), CorporateInputs: kstInputs}
}

// Beispiel der Referenz zur Gewerbesteuer (buchfink-steuererklaerungen):
// Gewinn 120.000 €, Zinsen 8.000 €, Leasing Pkw 6.000 €, Büromiete
// 24.000 €, befristete Lizenz 10.000 €. Summe der Finanzierungsanteile
// 23.700 € unter 200.000 €, keine Hinzurechnung. Messbetrag 4.200 €,
// Gewerbesteuer bei 400 % 16.800 €.
func TestTradeTaxExample(t *testing.T) {
	l := Ledger{}
	book(l, "7320", eur(8000))
	book(l, "6560", eur(6000))
	book(l, "6310", eur(24000))
	book(l, "6837", eur(10000))
	data := CorporateData{Ledger: l, NetIncome: eur(120000), BalanceResult: eur(120000), PriorKnown: true, TradeTaxRate: 400}
	d, err := BuildTradeTax(tradeCommon(2025, settled()), data)
	if err != nil {
		t.Fatal(err)
	}
	noInternalFindings(t, d)
	wantAmount(t, d, "gewst.gewinn", eur(120000))
	wantAmount(t, d, "gewst.8_1a", eur(8000))
	wantAmount(t, d, "gewst.8_1d", eur(6000))
	wantAmount(t, d, "gewst.8_1e", eur(24000))
	wantAmount(t, d, "gewst.8_1f", eur(10000))
	wantAmount(t, d, "gewst.8_1_summe", eur(23700))
	wantAmount(t, d, "gewst.8_1", 0)
	wantAmount(t, d, "gewst.k.ertrag_gerundet", eur(120000))
	wantAmount(t, d, "gewst.k.messbetrag", eur(4200))
	wantAmount(t, d, "gewst.k.steuer", eur(16800))
	if len(d.Blocking()) != 0 {
		t.Errorf("blockierende Befunde: %+v", d.Blocking())
	}
	if l := line(t, d, "gewst.8_1e"); len(l.Origin.Accounts) != 1 || l.Origin.Accounts[0].Den != 2 || l.Origin.Accounts[0].Amount != eur(12000) {
		t.Errorf("Herkunft der Miete ohne Anteil: %+v", l.Origin)
	}
}

// § 8 Nr. 1 GewStG: ein Viertel der Summe, soweit sie 200.000 € übersteigt.
func TestTradeTaxFinancingAboveAllowance(t *testing.T) {
	l := Ledger{}
	book(l, "7320", eur(300000))
	data := CorporateData{Ledger: l, NetIncome: eur(500000), BalanceResult: eur(500000), PriorKnown: true, TradeTaxRate: 400}
	d, err := BuildTradeTax(tradeCommon(2025, settled()), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "gewst.8_1", eur(25000))
	wantAmount(t, d, "gewst.k.ertrag", eur(525000))
	if !hasFinding(d, "hinzurechnung_freibetrag_nah") {
		t.Errorf("Summe über dem Freibetrag ohne Hinweis")
	}
}

// Beispiel der Referenz zur Gewerbesteuer auf Beteiligungserträge: Tochter A
// zu 100 % seit Jahren, Dividende 100.000 €: im Gewerbeertrag bleiben die
// 5 %, Gewerbesteuer 700 €.
func TestTradeTaxSubsidiaryDividend(t *testing.T) {
	kst := append(settled(confirmed("kst.gk.8b_streubesitz", Cents(0))), confirmedWithholding(eur(25000), eur(1375))...)
	d, err := BuildTradeTax(tradeCommon(2025, kst, confirmed("gewst.8_5", Cents(0))), subsidiaryDividend())
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "gewst.gewinn", eur(5000))
	wantAmount(t, d, "gewst.8_5", 0)
	wantAmount(t, d, "gewst.k.messbetrag", eur(175))
	wantAmount(t, d, "gewst.k.steuer", eur(700))
	if len(d.Blocking()) != 0 {
		t.Errorf("blockierende Befunde: %+v", d.Blocking())
	}
}

// Beteiligung B zu 12 % seit Jahren, Dividende 10.000 €: bei der
// Körperschaftsteuer zu 95 % frei, bei der Gewerbesteuer 9.500 €
// hinzuzurechnen (§ 8 Nr. 5 GewStG); Gewerbesteuer 1.400 €.
func TestTradeTaxHoldingBelowFifteenPercent(t *testing.T) {
	l := Ledger{}
	book(l, "7005", -eur(10000))
	book(l, "7630", eur(2500))
	book(l, "7633", eur(137, 50))
	data := CorporateData{Ledger: l, NetIncome: eur(7362, 50), BalanceResult: eur(7362, 50), PriorKnown: true, TradeTaxRate: 400,
		Holdings: []Holding{{Name: "Beteiligung B GmbH", QuotaPermille: 120, AcquiredOn: "2019-01-01", Dividends: eur(10000)}}}
	kst := append(settled(confirmed("kst.gk.8b_streubesitz", Cents(0))), confirmedWithholding(eur(2500), eur(137, 50))...)
	d, err := BuildTradeTax(tradeCommon(2025, kst), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "gewst.gewinn", eur(500))
	if l := line(t, d, "gewst.8_5"); l.Status != StatusProposed || l.Value.Amount() != eur(9500) {
		t.Errorf("§ 8 Nr. 5 aus der Kartei: %+v", l)
	}
	wantAmount(t, d, "gewst.k.ertrag_gerundet", eur(10000))
	wantAmount(t, d, "gewst.k.steuer", eur(1400))
}

// Eine im Jahr erworbene Tochter: bei der Körperschaftsteuer wirkt der Erwerb
// auf den Jahresbeginn zurück, bei der Gewerbesteuer nicht.
func TestTradeTaxHoldingAcquiredDuringTheYear(t *testing.T) {
	l := Ledger{}
	book(l, "7006", -eur(10000))
	data := CorporateData{Ledger: l, NetIncome: eur(10000), BalanceResult: eur(10000), PriorKnown: true, TradeTaxRate: 400,
		Holdings: []Holding{{Name: "Neue Tochter GmbH", QuotaPermille: 1000, AcquiredOn: "2025-06-01", Dividends: eur(10000)}}}
	kd, err := BuildCorporateTax(common(2025, settled()...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, kd, "kst.gk.8b_streubesitz", 0)
	td, err := BuildTradeTax(tradeCommon(2025, settled()), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, td, "gewst.8_5", eur(9500))
}

// § 8 Nr. 9 und § 9 Nr. 5 GewStG: die abgezogenen Spenden werden
// hinzugerechnet und im Rahmen des Höchstbetrags wieder gekürzt.
func TestTradeTaxDonations(t *testing.T) {
	l := Ledger{}
	book(l, "6393", eur(30000))
	data := CorporateData{Ledger: l, NetIncome: eur(70000), BalanceResult: eur(70000), PriorKnown: true,
		Revenue: eur(1000000), Wages: eur(500000), TradeTaxRate: 400}
	d, err := BuildTradeTax(tradeCommon(2025, settled(confirmed("kst.z.z10", Cents(eur(1000000))))), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "gewst.gewinn", eur(80000))
	wantAmount(t, d, "gewst.8_9", eur(20000))
	wantAmount(t, d, "gewst.9_5", eur(20000))
	wantAmount(t, d, "gewst.k.ertrag", eur(80000))
}

// § 10a GewStG: bis 1 Mio. € voll, darüber 60 %.
func TestTradeTaxLossCarryForward(t *testing.T) {
	data := CorporateData{Ledger: Ledger{}, NetIncome: eur(2000000), BalanceResult: eur(2000000), PriorKnown: true, TradeTaxRate: 400}
	tc := tradeCommon(2025, settled())
	tc.Inputs[1] = confirmed("gewst.verlust_vorjahr", Cents(eur(3000000)))
	d, err := BuildTradeTax(tc, data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "gewst.verlust_abzug", eur(1600000))
	wantAmount(t, d, "gewst.verlust_ende", eur(1400000))
	wantAmount(t, d, "gewst.k.ertrag", eur(400000))
}

// Investmentfonds: Zeile 48 trägt die ganze Teilfreistellung, die
// Kontrollrechnung rechnet die Hälfte wieder hinzu (§ 20 Abs. 5 InvStG).
func TestTradeTaxPartialExemption(t *testing.T) {
	l := Ledger{}
	book(l, "7010", -eur(1000))
	data := CorporateData{Ledger: l, NetIncome: eur(1000), BalanceResult: eur(1000), PriorKnown: true, TradeTaxRate: 400,
		Funds: []Fund{{Name: "Aktien-ETF", ExemptPermille: 800, Determined: true, Income: eur(1000)}}}
	kst := settled(confirmed("kst.gk.inv_teilfreistellung", Cents(eur(800))))
	kd, err := BuildCorporateTax(common(2025, kst...), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, kd, "kst.gk.summe", eur(200))
	td, err := BuildTradeTax(tradeCommon(2025, kst), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, td, "gewst.z48", eur(800))
	wantAmount(t, td, "gewst.k.ertrag", eur(600))
}

func TestTradeTaxChecks(t *testing.T) {
	data := CorporateData{Ledger: Ledger{}, NetIncome: eur(1000), BalanceResult: eur(1000), PriorKnown: true}
	tc := tradeCommon(2025, nil, confirmed("gewst.betriebsstaetten", Bool(true)))
	d, err := BuildTradeTax(tc, data)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"zerlegung", "kst_nicht_vollstaendig", "hebesatz_fehlt"} {
		if !hasFinding(d, code) {
			t.Errorf("Befund %s fehlt", code)
		}
	}
	data.TradeTaxRate = 250
	d, err = BuildTradeTax(tradeCommon(2027, settled()), data)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(d, "hebesatz_unter_minimum") {
		t.Errorf("Hebesatz 250 %% ab 2027 ohne Befund")
	}
}

// Übersteigen die Vorauszahlungen die Gewerbesteuer, steht ein
// Erstattungsanspruch auf 1435 statt einer Rückstellung; die Kontrollrechnung
// gleicht beides ab. Gewinn 120.000 €, Steuer 16.800 €, Vorauszahlungen
// 20.000 €: Erstattung 3.200 €.
func TestTradeTaxProvisionCheckCountsTheRefund(t *testing.T) {
	l := Ledger{}
	book(l, "7610", eur(16800))
	book(l, accounting.AccountGewerbesteuerForderung, eur(3200))
	data := CorporateData{Ledger: l, NetIncome: eur(103200), BalanceResult: eur(103200), PriorKnown: true, TradeTaxRate: 400}
	// Die Vorauszahlungen sind laufende Buchungen, die Erstattung die
	// Abschlussbuchung, die 7610 um 3.200 € mindert.
	prepaid := l["7610"]
	prepaid.Debit, prepaid.Credit, prepaid.ClosingCredit = eur(20000), eur(3200), eur(3200)
	l["7610"] = prepaid
	d, err := BuildTradeTax(tradeCommon(2025, settled()), data)
	if err != nil {
		t.Fatal(err)
	}
	wantAmount(t, d, "gewst.k.steuer", eur(16800))
	wantAmount(t, d, "gewst.k.ergebnis", -eur(3200))
	wantAmount(t, d, "gewst.k.rueckstellung", -eur(3200))
	if hasFinding(d, "rueckstellung_abweichung") {
		t.Errorf("Abweichung gemeldet, obwohl der Erstattungsanspruch gebucht ist: %+v", d.Findings)
	}
}
