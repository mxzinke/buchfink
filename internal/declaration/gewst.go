package declaration

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Konten des SKR04 für die Hinzurechnungen nach § 8 Nr. 1 GewStG.
var (
	accountsDebtInterest = []string{"7300", "7309", "7310", "7316", "7317", "7318", "7319", "7320",
		"7325", "7326", "7329", "7330", "7339"}
	accountsTaxInterest   = []string{"7305", "7306", "7308", "7311", "7360", "7361", "7362", "7363", "7364", "7365", "7366"}
	accountsAnnuities     = []string{"7327"}
	accountsSilentPartner = []string{"7399"}
	accountsMovableRent   = []string{"6498", "6560", "6833", "6835", "6836", "6838", "6840"}
	accountsElectricRent  = []string{"6565"}
	accountsRealRent      = []string{"6310", "6313", "6315", "6316", "6317", "6550"}
	accountsLicences      = []string{"6837"}
)

// TradeCommon sind die Angaben der Gewerbesteuererklärung; die Einkommens-
// ermittlung übernimmt sie aus der Körperschaftsteuererklärung desselben
// Jahres mit deren Eingaben.
type TradeCommon struct {
	Common
	CorporateInputs []Input
}

// BuildTradeTax rechnet die Gewerbesteuererklärung.
func BuildTradeTax(common TradeCommon, data CorporateData) (*Declaration, error) {
	cat, exact, err := CatalogFor(KindTradeTax, common.Year)
	if err != nil {
		return nil, err
	}
	kstCat, _, err := CatalogFor(KindCorporateTax, common.Year)
	if err != nil {
		return nil, err
	}
	kc := newCalc(kstCat, common.CorporateInputs)
	corp := kc.corporateIncome(common.Common, data)

	c := newCalc(cat, common.Inputs)
	s := subjectOf(common.Settings)
	c.textLine("gewst.name", firmName(common.Settings), "Der Firmenname fehlt in den Unternehmensdaten.")
	c.taxNumberLine("gewst.steuernummer", s)
	c.textLine("gewst.rechtsform", legalFormText(s.LegalForm), "Die Rechtsform fehlt in den Unternehmensdaten.")
	c.resolve("gewst.betriebsstaetten", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: []string{
		"Wertungsfrage: Unterhält die Gesellschaft Betriebsstätten in mehreren Gemeinden, etwa ein Homeoffice des Geschäftsführers in einer anderen Gemeinde?"}})
	if yes, ok := c.flag("gewst.betriebsstaetten"); ok && yes {
		c.finding(SeverityBlocking, "zerlegung", "gewst.betriebsstaetten",
			"Bei Betriebsstätten in mehreren Gemeinden ist zusätzlich die Zerlegungserklärung (GewSt 1 D) abzugeben. Die Zerlegung rechnet Buchfink nicht; bitte mit einem Steuerberater klären.",
			"§ 28 GewStG, § 14a Satz 1 GewStG")
	}
	c.resolve("gewst.z21", figure{value: Bool(false), status: StatusAuto, origin: Origin{Kind: OriginNone}})

	pending := pendingCorporate(kc)
	if len(pending) > 0 {
		c.finding(SeverityBlocking, "kst_nicht_vollstaendig", "gewst.gewinn",
			"Der Gewinn stammt aus der Körperschaftsteuererklärung, in der noch offen oder vorgeschlagen ist: "+strings.Join(pending, "; ")+".",
			"§ 7 Satz 1 GewStG")
	}
	kstOrigin := Origin{Kind: OriginComputation, Text: "Körperschaftsteuererklärung desselben Jahres"}
	c.books("gewst.gewinn", Cents(corp.income), withFormula(kstOrigin, "Einkommen vor Verlustabzug laut Anlage ZVE (nach Abzug der Zuwendungen)"))
	c.books("gewst.z39", Cents(kc.amount("kst.gk.summe")), withFormula(kstOrigin, "Summe der Einkünfte laut Anlage GK und ZVE"),
		"Für die Fälle, die Buchfink bucht, ist an der Summe der Einkünfte nichts zu korrigieren.")
	c.resolve("gewst.z40", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}})
	c.books("gewst.z48", Cents(corp.partialExemption), withFormula(kstOrigin, "Teilfreistellung laut Anlage GK"))

	c.financing(data.Ledger)
	switch {
	case corp.dividendsFree == 0:
		c.resolve("gewst.8_5", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	case corp.tradeStreubesitzKnown:
		c.resolve("gewst.8_5", figure{value: Cents(corp.tradeStreubesitz), status: StatusProposed,
			origin: Origin{Kind: OriginAssets, Formula: "§ 8b-freie Bezüge aus Beteiligungen unter 15 % zu Beginn des Erhebungszeitraums, abzüglich 5 %"},
			notes:  []string{corp.tradeStreubesitzNote, "Ein Erwerb im Laufe des Jahres wirkt bei der Gewerbesteuer nicht auf den Beginn zurück."}})
	default:
		c.resolve("gewst.8_5", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: []string{corp.tradeStreubesitzNote}})
	}
	c.books("gewst.8_9", Cents(corp.donationsDeducted), withFormula(kstOrigin, "abziehbare Zuwendungen laut Anlage ZVE"))
	c.resolve("gewst.8_10", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})

	if tax, o := data.Ledger.expense(full("7680")...); tax != 0 {
		c.propose("gewst.9_1", Cents(tax), o, "Nur die Grundsteuer für Grundbesitz im Betriebsvermögen der Gesellschaft; umgelegte Grundsteuer einer Miete gehört nicht dazu.")
	} else {
		c.resolve("gewst.9_1", noneFigure())
	}
	if corp.dividendsFree != 0 {
		c.resolve("gewst.9_2a", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	} else {
		c.resolve("gewst.9_2a", noneFigure())
	}

	// § 9 Nr. 5 GewStG: Höchstbetrag aus dem um § 8 Nr. 9 erhöhten Gewinn
	// oder 4 Promille der Umsätze, Löhne und Gehälter.
	base := corp.income + c.amount("gewst.8_9")
	capIncome := domain.Cents(0)
	if base > 0 {
		capIncome = domain.MulRound(base, donationPercent, 100)
	}
	capTurnover := domain.MulRound(kc.amount("kst.z.z10")+kc.amount("kst.z.z11"), donationPermille, 1000)
	limit := capIncome
	if capTurnover > limit {
		limit = capTurnover
	}
	given := kc.amount("kst.z.z5")
	cut := given
	if cut > limit {
		cut = limit
	}
	c.books("gewst.9_5", Cents(cut), Origin{Kind: OriginComputation, FromLines: []string{"gewst.gewinn", "gewst.8_9"},
		Formula: fmt.Sprintf("Zuwendungen des Jahres (%s €), höchstens der höhere Betrag aus 20 %% des um § 8 Nr. 9 erhöhten Gewinns (%s €) und 4 Promille der Umsätze, Löhne und Gehälter (%s €)",
			given, capIncome, capTurnover)})

	before := corp.income + c.amount("gewst.8_1") + c.amount("gewst.8_5") + c.amount("gewst.8_9") + c.amount("gewst.8_10") -
		c.amount("gewst.9_1") - c.amount("gewst.9_2a") - c.amount("gewst.9_5") +
		corp.partialExemption/2

	if data.Founded {
		c.resolve("gewst.verlust_vorjahr", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
			notes: []string{"Im ersten Erhebungszeitraum gibt es keinen Gewerbeverlust aus Vorjahren."}})
	} else {
		c.resolve("gewst.verlust_vorjahr", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{"Vortragsfähigen Gewerbeverlust aus dem Feststellungsbescheid des Vorjahres übernehmen; ohne Verlust null."}})
	}
	carried := c.amount("gewst.verlust_vorjahr")
	deduction := lossDeduction(carried, before, tradeLossPermille)
	beforeKeys := []string{"gewst.gewinn", "gewst.8_1", "gewst.8_5", "gewst.8_9", "gewst.8_10", "gewst.9_1", "gewst.9_2a", "gewst.9_5", "gewst.z48"}
	c.books("gewst.verlust_abzug", Cents(deduction), fromLines(
		fmt.Sprintf("Gewerbeverlust bis 1 Mio. € voll, darüber %s des übersteigenden Gewerbeertrags", percentText(tradeLossPermille)),
		append([]string{"gewst.verlust_vorjahr"}, beforeKeys...)...))
	loss := domain.Cents(0)
	if before < 0 {
		loss = -before
	}
	c.books("gewst.verlust_ende", Cents(carried-deduction+loss), fromLines("Vortrag − Abzug + Fehlbetrag des Jahres",
		"gewst.verlust_vorjahr", "gewst.verlust_abzug"))
	c.none("gewst.z100", Value{}, "Tonnagegewinn betrifft Handelsschiffe.")

	c.tradeControl(common.Common, data, before, deduction, beforeKeys)
	if common.Basis.ClosingState == string(domain.FiscalYearOpen) {
		c.finding(SeverityBlocking, "abschluss_offen", "",
			"Der Jahresabschluss ist noch nicht aufgestellt. Die Erklärung baut auf dem aufgestellten Abschluss auf.", "§ 7 GewStG")
	}
	c.form("GewSt1A", RequirementMandatory, "§ 14a GewStG")
	c.form("Kontrolle", RequirementControl, "")
	return finish(KindTradeTax, common.Common, cat, exact, c), nil
}

func withFormula(o Origin, formula string) Origin {
	o.Formula = formula
	return o
}

// pendingCorporate nennt die offenen und vorgeschlagenen Zeilen der
// Einkommensermittlung.
func pendingCorporate(kc *calc) []string {
	out := []string{}
	for _, f := range kc.cat.Forms {
		switch f.ID {
		case "GK", "ZVE", "Verluste", "Z":
		default:
			continue
		}
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				fig, ok := kc.sheet[l.Key]
				if !ok {
					continue
				}
				switch {
				case fig.status == StatusProposed:
					out = append(out, l.Label+" (vorgeschlagen)")
				case fig.status == StatusOpen && l.Required:
					out = append(out, l.Label+" (offen)")
				}
			}
		}
	}
	return out
}

// financing rechnet die Hinzurechnung nach § 8 Nr. 1 GewStG.
func (c *calc) financing(l Ledger) {
	var total domain.Cents
	sumOrigin := Origin{Kind: OriginAccounts, FromLines: []string{}}
	line := func(key string, parts []share, note string) {
		var booked domain.Cents
		o := Origin{Kind: OriginAccounts}
		for _, p := range parts {
			t, ok := l[p.account]
			if !ok || t.Net() == 0 {
				continue
			}
			share := t.Net()
			if p.num != p.den {
				share = domain.MulRound(t.Net(), p.num, p.den)
			}
			booked += t.Net()
			total += share
			o.Accounts = append(o.Accounts, AccountPart{Account: p.account, Name: t.Name, Balance: t.Net(),
				Num: p.num, Den: p.den, Amount: share})
			o.EntryIDs = append(o.EntryIDs, t.EntryIDs...)
		}
		notes := []string{}
		if note != "" {
			notes = append(notes, note)
		}
		c.books(key, Cents(booked), o, notes...)
		sumOrigin.FromLines = append(sumOrigin.FromLines, key)
	}
	taxInterestNote := ""
	if l.hasBalance(accountsTaxInterest...) {
		taxInterestNote = "Auf Konten für Steuerzinsen und Aufzinsungen (7305 ff., 7360 ff.) ist gebucht; ob sie Entgelte für Schulden sind, ist zu prüfen."
	}
	line("gewst.8_1a", full(accountsDebtInterest...), taxInterestNote)
	line("gewst.8_1b", full(accountsAnnuities...), "")
	line("gewst.8_1c", full(accountsSilentPartner...), "")
	moveNote := ""
	if l.hasBalance("6250") {
		moveNote = "Auf Kaufleasing (6250) ist gebucht; ob ein Mietleasing vorliegt, ist zu prüfen."
	}
	line("gewst.8_1d", append(fraction(1, 5, accountsMovableRent...), fraction(1, 10, accountsElectricRent...)...), moveNote)
	line("gewst.8_1e", fraction(1, 2, accountsRealRent...), "Miet- und Pachtnebenkosten (6318) gehören nicht dazu.")
	line("gewst.8_1f", fraction(1, 4, accountsLicences...), "SaaS, Hosting und Coworking auf anderen Konten nach dem Vertrag einordnen: Miete, Lizenz oder Dienstleistung.")
	sumOrigin.Formula = "Entgelte für Schulden + Renten + stille Gesellschafter + 1/5 bewegliche Miete (Elektrofahrzeuge 1/10) + 1/2 unbewegliche Miete + 1/4 Lizenzen"
	c.books("gewst.8_1_summe", Cents(total), sumOrigin)
	add := domain.Cents(0)
	if total > financingAllowance {
		add = domain.MulRound(total-financingAllowance, 1, 4)
	}
	c.books("gewst.8_1", Cents(add), fromLines("ein Viertel der Summe, soweit sie 200.000 € übersteigt", "gewst.8_1_summe"))
	if total > 150_000_00 {
		c.finding(SeverityWarning, "hinzurechnung_freibetrag_nah", "gewst.8_1_summe",
			fmt.Sprintf("Die Summe der Finanzierungsanteile (%s €) liegt nahe am oder über dem Freibetrag von 200.000 €. Die Einordnung von Mieten, Lizenzen und Dienstleistungen prüfen.", total),
			"§ 8 Nr. 1 GewStG")
	}
}

// tradeControl rechnet Messbetrag und Steuer zur Kontrolle.
func (c *calc) tradeControl(common Common, data CorporateData, before, deduction domain.Cents, keys []string) {
	ertrag := before - deduction
	c.books("gewst.k.ertrag", Cents(ertrag), fromLines("Gewinn + Hinzurechnungen − Kürzungen + halbe Teilfreistellung − Gewerbeverlust",
		append(keys, "gewst.verlust_abzug")...),
		"Die Hälfte der Teilfreistellung ist bei der Gewerbesteuer wieder hinzuzurechnen (§ 20 Abs. 5 InvStG).")
	rounded := domain.Cents(0)
	if ertrag > 0 {
		rounded = ertrag - ertrag%10000
	}
	c.books("gewst.k.ertrag_gerundet", Cents(rounded), fromLines("auf volle 100 € abgerundet", "gewst.k.ertrag"))
	base := domain.MulRound(rounded, tradeTaxBasePermille, 1000)
	c.books("gewst.k.messbetrag", Cents(base), fromLines("3,5 % des abgerundeten Gewerbeertrags", "gewst.k.ertrag_gerundet"))
	if data.TradeTaxRate <= 0 {
		c.open("gewst.k.hebesatz", "Der Hebesatz der Gemeinde fehlt in den Einstellungen.")
		c.finding(SeverityWarning, "hebesatz_fehlt", "gewst.k.hebesatz",
			"Ohne Hebesatz rechnet Buchfink keine Gewerbesteuer; er steht im Bescheid der Gemeinde.", "§ 16 GewStG")
	} else {
		c.books("gewst.k.hebesatz", Text(fmt.Sprintf("%d %%", data.TradeTaxRate)), Origin{Kind: OriginSettings, Text: "Einstellungen zum Abschluss"})
		if min := minimumMultiplier(common.Year); data.TradeTaxRate < min {
			c.finding(SeverityWarning, "hebesatz_unter_minimum", "gewst.k.hebesatz",
				fmt.Sprintf("Der Hebesatz von %d %% liegt unter dem Mindesthebesatz von %d %%.", data.TradeTaxRate, min),
				"§ 16 Abs. 4 Satz 2 GewStG")
		}
	}
	tax := domain.MulRound(base, data.TradeTaxRate, 100)
	c.books("gewst.k.steuer", Cents(tax), fromLines("Messbetrag × Hebesatz", "gewst.k.messbetrag", "gewst.k.hebesatz"))
	prepaid, o := data.Ledger.sum(full("7610"), true, AccountTotals.NetBeforeClosing)
	c.books("gewst.k.vorauszahlungen", Cents(prepaid), o)
	c.books("gewst.k.ergebnis", Cents(tax-prepaid), fromLines("Steuer − Vorauszahlungen", "gewst.k.steuer", "gewst.k.vorauszahlungen"))
	provision := -data.Ledger["3035"].EndBalance()
	c.books("gewst.k.rueckstellung", Cents(provision), Origin{Kind: OriginAccounts, Accounts: []AccountPart{{
		Account: "3035", Name: data.Ledger["3035"].Name, Balance: provision, Num: 1, Den: 1, Amount: provision}}})
	expected := tax - prepaid
	if expected < 0 {
		expected = 0
	}
	if data.TradeTaxRate > 0 {
		if diff := provision - expected; diff.Abs() > 100 {
			c.finding(SeverityWarning, "rueckstellung_abweichung", "gewst.k.rueckstellung",
				fmt.Sprintf("Die gebuchte Gewerbesteuerrückstellung (%s €) weicht von der Kontrollrechnung (%s € nach Vorauszahlungen) um %s € ab.",
					provision, expected, diff), "§ 249 Abs. 1 HGB")
		}
	}
}
