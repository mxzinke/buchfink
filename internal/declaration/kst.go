package declaration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// CorporateData sind die Größen der Körperschaftsteuererklärung aus den
// Büchern. Die Gewerbesteuererklärung baut auf ihnen auf.
type CorporateData struct {
	Ledger Ledger
	// NetIncome ist der Jahresüberschuss der Gewinn- und Verlustrechnung
	// (§ 275 Abs. 2 Nr. 17 HGB), BalanceResult der Posten A.V der Bilanz.
	NetIncome     domain.Cents
	BalanceResult domain.Cents
	// Revenue sind die Umsatzerlöse (§ 275 Abs. 2 Nr. 1 HGB), Wages die Löhne
	// und Gehälter (Nr. 6 Buchst. a).
	Revenue domain.Cents
	Wages   domain.Cents
	// PriorKnown: Eigenkapital und Nennkapital zum Schluss des Vorjahres sind
	// aus den Büchern bekannt (Vorjahr gebucht oder Saldenvortrag).
	PriorKnown      bool
	PriorEquity     domain.Cents
	PriorSubscribed domain.Cents
	// TaxBalanceEffect ist der Unterschied des Eigenkapitals laut Steuerbilanz
	// zur Handelsbilanz am Schluss des Jahres, PriorTaxBalanceEffect am Schluss
	// des Vorjahres (Überleitung, § 60 Abs. 2 EStDV).
	TaxBalanceEffect      domain.Cents
	PriorTaxBalanceEffect domain.Cents
	TaxBalanceNotes       []string
	// TaxBalanceUnknown nennt den Grund, wenn die Überleitung fehlt.
	TaxBalanceUnknown string
	// Distributions sind die Gewinnverwendungsbeschlüsse mit Ausschüttung.
	Distributions []Distribution
	// Holdings sind die Anteile an Kapitalgesellschaften aus der Anlagenkartei.
	Holdings []Holding
	// Funds sind die Investmentanteile aus der Anlagenkartei.
	Funds []Fund
	// Founded: die Steuerpflicht beginnt in diesem Wirtschaftsjahr.
	Founded bool
	// TradeTaxRate ist der Hebesatz in Prozent; 0 heißt: nicht eingetragen.
	TradeTaxRate int64
}

// Distribution ist eine beschlossene Gewinnausschüttung.
type Distribution struct {
	ForYear      int
	DecisionDate string
	Amount       domain.Cents
}

// Holding ist ein Anteil an einer Kapitalgesellschaft mit den Bezügen des
// Jahres, die über die Anlagenkartei mit ihm verknüpft sind.
type Holding struct {
	AssetID uint
	Name    string
	// QuotaPermille ist die Beteiligungsquote der Kartei. Sie trägt keinen
	// Stichtag; null heißt: nicht erfasst.
	QuotaPermille int
	AcquiredOn    string
	DisposedOn    string
	// Dividends sind alle über die Kartei verknüpften Bezüge des Jahres,
	// OtherAccounts der Teil davon, der nicht auf einem §-8b-Konto gebucht ist
	// (etwa 7000).
	Dividends     domain.Cents
	OtherAccounts domain.Cents
	EntryIDs      []uint
}

// Fund ist ein Investmentanteil mit seinen steuerlichen Größen im Jahr.
type Fund struct {
	AssetID uint
	Name    string
	// ExemptPermille ist die Teilfreistellung nach § 20 InvStG für die GmbH;
	// Determined meldet, ob sie bestimmt ist.
	ExemptPermille int
	Determined     bool
	Income         domain.Cents
	Vorabpauschale domain.Cents
	// DisposedInYear: im Jahr ganz oder teilweise veräußert.
	DisposedInYear bool
	EntryIDs       []uint
}

// Konten des SKR04 für die außerbilanziellen Korrekturen.
var (
	accountsCorporateTax = []string{"7600", "7603", "7604", "7630"}
	accountsSolidarity   = []string{"7608", "7609", "7607", "7633"}
	accountsTradeTax     = []string{"7610", "7641", "7643"}
	accountsAncillary    = []string{"6437", "7302", "7304", "7306", "7308"}
	accountsGifts        = []string{"6620", "6621", "6622"}
	accountsDonations    = []string{"6390", "6391", "6392", "6393", "6394", "6395", "6397", "6398"}
	accountsDeductible   = []string{"6391", "6392", "6393", "6395", "6397", "6398"}
	accountsVGA          = []string{"6067", "6073", "6313", "6833", "7316", "7317"}
	accountsDividends    = []string{"7005", "7006", "7014", "7015", "7103", "7104"}
	accountsSaleGain     = []string{"4852", "4858", "4906", "4904"}
	accountsWriteUpShare = []string{"4914"}
	accountsWriteUpLoan  = []string{"4913", "4916"}
	accountsShareLoss    = []string{"6892", "6898", "6858", "7204", "7214"}
	accountsLoanWriteOff = []string{"7200", "7201"}
	accountsCapitalRes   = []string{"2920", "2925", "2927", "2928"}
	accountsNonDedVAT    = []string{"6860", "6865", "6871"}
)

// corporate ist das Ergebnis der Einkommensermittlung, das die
// Gewerbesteuererklärung übernimmt.
type corporate struct {
	// income ist das Einkommen vor Verlustabzug (nach Zuwendungen).
	income domain.Cents
	// donationsDeducted sind die abgezogenen Zuwendungen.
	donationsDeducted domain.Cents
	// dividendsFree sind die nach § 8b Abs. 1 KStG außer Ansatz bleibenden
	// Bezüge, fivePercent die 5 % darauf.
	dividendsFree domain.Cents
	fivePercent   domain.Cents
	// partialExemption ist die Teilfreistellung nach § 20 InvStG.
	partialExemption domain.Cents
	// tradeStreubesitz sind die § 8b-freien Bezüge aus Beteiligungen unter
	// 15 % zu Beginn des Erhebungszeitraums (§ 8 Nr. 5 GewStG), soweit aus der
	// Kartei ableitbar; tradeStreubesitzKnown meldet, ob das gelang.
	tradeStreubesitz      domain.Cents
	tradeStreubesitzKnown bool
	tradeStreubesitzNote  string
	revenue, wages        domain.Cents
	// keys verweisen auf die Zeilen, aus denen die Werte stammen.
	incomeKey string
}

// BuildCorporateTax rechnet die Körperschaftsteuererklärung.
func BuildCorporateTax(common Common, data CorporateData) (*Declaration, error) {
	cat, exact, err := CatalogFor(KindCorporateTax, common.Year)
	if err != nil {
		return nil, err
	}
	c := newCalc(cat, common.Inputs)
	c.corporateHeader(common)
	c.corporateIncome(common, data)
	c.contributionAccount(common, data)
	c.corporateControl(common, data)
	c.corporateChecks(common, data)
	c.attachments()
	return finish(KindCorporateTax, common, cat, exact, c), nil
}

// attachments nennt die Anlagen, die mit der KSt 1 abzugeben sind.
func (c *calc) attachments() {
	names := []string{}
	for _, f := range c.cat.Forms {
		if f.ID == "KSt1" {
			continue
		}
		if state, ok := c.forms[f.ID]; ok && state.required == RequirementMandatory {
			names = append(names, strings.TrimSuffix(strings.SplitN(f.Title, " (", 2)[0], " "))
		}
	}
	c.books("kst.kst1.anlagen", Text(strings.Join(names, ", ")), Origin{Kind: OriginComputation,
		Text: "Anlagen, die nach den Zahlen des Jahres abzugeben sind"})
}

func (c *calc) corporateHeader(common Common) {
	s := subjectOf(common.Settings)
	c.textLine("kst.kst1.name", firmName(common.Settings), "Der Firmenname fehlt in den Unternehmensdaten.")
	c.textLine("kst.kst1.rechtsform", legalFormText(s.LegalForm), "Die Rechtsform fehlt in den Unternehmensdaten.")
	c.taxNumberLine("kst.kst1.steuernummer", s)
	c.resolve("kst.kst1.wid", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}})
	address := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(s.Address.Street + " " + s.Address.HouseNumber),
		strings.TrimSpace(s.Address.PostalCode + " " + s.Address.City),
	}, ", "))
	if s.Address.Street == "" || s.Address.City == "" {
		address = ""
	}
	c.textLine("kst.kst1.anschrift", address, "Die Anschrift fehlt in den Unternehmensdaten.")
	period := Origin{Kind: OriginSettings, Text: "Geschäftsjahr"}
	c.books("kst.kst1.wj_von", Date(common.Period.From), period)
	c.books("kst.kst1.wj_bis", Date(common.Period.To), period)
	c.books("kst.kst1.z10", Bool(false), Origin{Kind: OriginSettings},
		"Eine steuerbefreite Körperschaft bildet Buchfink nicht ab.")
	advisor := common.Settings.TaxAdvisor.Prepares(string(KindCorporateTax), common.Year)
	c.books("kst.kst1.berater", Bool(advisor), Origin{Kind: OriginSettings, Text: "Steuerliche Beratung laut Unternehmensdaten"})
}

func (c *calc) corporateIncome(common Common, data CorporateData) *corporate {
	l := data.Ledger
	out := &corporate{revenue: data.Revenue, wages: data.Wages}

	// Bilanzielles Ergebnis.
	c.books("kst.gk.z11", Cents(data.NetIncome), Origin{Kind: OriginStatement,
		Formula: "Jahresüberschuss/Jahresfehlbetrag (§ 275 Abs. 2 Nr. 17 HGB) ohne Ergebnisverwendung"})
	if data.NetIncome != data.BalanceResult {
		c.finding(SeverityBlocking, "ergebnis_ungleich_bilanz", "kst.gk.z11",
			fmt.Sprintf("Der Jahresüberschuss der Gewinn- und Verlustrechnung (%s €) weicht vom Posten A.V der Bilanz (%s €) ab.",
				data.NetIncome, data.BalanceResult), "§ 266 Abs. 3, § 275 Abs. 2 HGB")
	}
	stb := data.TaxBalanceEffect - data.PriorTaxBalanceEffect
	stbOrigin := Origin{Kind: OriginComputation,
		Formula: "Unterschied Steuer- zu Handelsbilanz am Schluss des Jahres abzüglich des Unterschieds am Schluss des Vorjahres",
		Text:    strings.Join(data.TaxBalanceNotes, " ")}
	switch {
	case data.TaxBalanceUnknown != "":
		f := c.resolve("kst.gk.stb", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{data.TaxBalanceUnknown}})
		if f.status == StatusOpen {
			c.finding(SeverityBlocking, "ueberleitung_fehlt", "kst.gk.stb", data.TaxBalanceUnknown, "§ 60 Abs. 2 EStDV")
		}
	case stb == 0:
		c.books("kst.gk.stb", Cents(0), stbOrigin, "Die Überleitung zeigt keine Abweichung, die sich im Jahr auswirkt.")
	default:
		c.propose("kst.gk.stb", Cents(stb), stbOrigin,
			"Abweichungen der Steuerbilanz sind eine Wertungsfrage; im Zweifel mit einem Steuerberater klären.")
	}

	// Nicht abziehbare Steuern und Aufwendungen.
	accountLine := func(key string, accounts []string) domain.Cents {
		amount, o := l.expense(full(accounts...)...)
		c.books(key, Cents(amount), o)
		return amount
	}
	accountLine("kst.gk.kst", accountsCorporateTax)
	accountLine("kst.gk.z82", accountsSolidarity)
	accountLine("kst.gk.gewst", accountsTradeTax)
	if amount, o := l.expense(full(accountsAncillary...)...); amount != 0 {
		c.propose("kst.gk.z88", Cents(amount), o,
			"Erstattungen früher festgesetzter Nachzahlungszinsen gegenrechnen, Erstattungszinsen nach § 233a AO nicht.")
	} else {
		c.books("kst.gk.z88", Cents(0), o)
	}
	accountLine("kst.gk.geschenke", accountsGifts)
	accountLine("kst.gk.bewirtung", []string{"6644"})
	accountLine("kst.gk.repraesentation", []string{"6645"})
	accountLine("kst.gk.nab", []string{"6642"})
	nabNotes := []string{"Geldbußen, Ordnungs- und Verwarnungsgelder und Geldstrafen haben kein eigenes Konto; sind welche gebucht, hier eintragen."}
	if l.hasBalance(accountsNonDedVAT...) {
		nabNotes = append(nabNotes, "Auf den Konten der nicht abziehbaren Vorsteuer (6860 ff.) sind Beträge gebucht. Soweit sie auf Aufwendungen nach § 4 Abs. 5 Satz 1 Nr. 1 bis 4 oder 7 EStG entfallen, sind sie hier hinzuzurechnen (§ 10 Nr. 2 KStG).")
	}
	c.propose("kst.gk.nab_weitere", Cents(0), Origin{Kind: OriginNone}, nabNotes...)
	accountLine("kst.gk.aufsichtsrat", []string{"6875"})
	accountLine("kst.gk.spenden", accountsDonations)

	vgaNotes := []string{}
	for _, a := range accountsVGA {
		if t, ok := l[a]; ok && t.Net() != 0 {
			vgaNotes = append(vgaNotes, fmt.Sprintf("%s %s: %s €", a, t.Name, t.Net()))
		}
	}
	if len(vgaNotes) > 0 {
		c.resolve("kst.gk.vga", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: []string{
			"Auf Konten für Leistungen an Gesellschafter ist gebucht. Prüfen, ob sie fremdüblich sind: " + strings.Join(vgaNotes, "; ") + "."}})
		c.finding(SeverityWarning, "vga_pruefen", "kst.gk.vga",
			"Wertungsfrage: Leistungen an Gesellschafter auf "+strings.Join(vgaNotes, "; ")+" auf verdeckte Gewinnausschüttungen prüfen.",
			"§ 8 Abs. 3 Satz 2 KStG")
	} else {
		c.propose("kst.gk.vga", Cents(0), Origin{Kind: OriginNone},
			"Auf den typischen Konten für Leistungen an Gesellschafter ist nichts gebucht. Eine verdeckte Gewinnausschüttung kann trotzdem vorliegen; nur der Mensch entscheidet.")
	}
	c.propose("kst.gk.einlage", Cents(0), Origin{Kind: OriginNone},
		"Eine verdeckte Einlage, die als Ertrag gebucht ist (etwa ein Forderungsverzicht eines Gesellschafters), mindert das Einkommen.")
	c.resolve("kst.gk.z81", noneFigure("Eine Forschungszulage bucht Buchfink nicht gesondert; ist eine als Ertrag erfasst, mit negativem Vorzeichen eintragen."))
	c.resolve("kst.gk.z93", noneFigure())

	// § 8b KStG.
	c.participations(common, data, out)

	// Investmentanteile.
	c.investments(data, out)

	// Summe der Einkünfte.
	add := []string{"kst.gk.z11", "kst.gk.stb", "kst.gk.kst", "kst.gk.z82", "kst.gk.gewst", "kst.gk.z88",
		"kst.gk.geschenke", "kst.gk.bewirtung", "kst.gk.repraesentation", "kst.gk.nab", "kst.gk.nab_weitere",
		"kst.gk.aufsichtsrat", "kst.gk.spenden", "kst.gk.vga", "kst.gk.z81", "kst.gk.z93",
		"kst.gk.8b_5_bezuege", "kst.gk.8b_5_gewinne", "kst.gk.8b_minderungen", "kst.gk.8b_darlehen",
		"kst.gk.inv_vorabpauschale"}
	sub := []string{"kst.gk.einlage", "kst.gk.8b_frei", "kst.gk.8b_gewinne", "kst.gk.8b_zuschreibung_darlehen",
		"kst.gk.inv_teilfreistellung"}
	var total domain.Cents
	for _, k := range add {
		total += c.amount(k)
	}
	for _, k := range sub {
		total -= c.amount(k)
	}
	c.books("kst.gk.summe", Cents(total), fromLines(
		"Jahresüberschuss + Hinzurechnungen − Kürzungen", append(add, sub...)...))

	// Anlage ZVE und Anlage Z.
	c.books("kst.zve.einkuenfte", Cents(total), fromLines("Übertrag", "kst.gk.summe"))
	c.books("kst.zve.z32", Cents(total), fromLines("Einkommen vor Zuwendungen und Verlustabzug", "kst.zve.einkuenfte"))
	deducted := c.donations(common, data, total)
	out.donationsDeducted = deducted
	income := total - deducted
	c.books("kst.zve.gesamtbetrag", Cents(income), fromLines("Zeile 32 − abziehbare Zuwendungen", "kst.zve.z32", "kst.zve.zuwendungen"))
	out.income = income
	out.incomeKey = "kst.zve.gesamtbetrag"
	c.losses(common, data, income)
	c.books("kst.zve.z69", Cents(0), Origin{Kind: OriginNone},
		"Der Freibetrag steht einer Körperschaft nicht zu, deren Ausschüttungen beim Empfänger Einnahmen nach § 20 Abs. 1 Nr. 1 EStG sind.")
	zve := income - c.amount("kst.zve.verlustabzug")
	c.books("kst.zve.zve", Cents(zve), fromLines("Gesamtbetrag − Verlustabzug − Freibetrag",
		"kst.zve.gesamtbetrag", "kst.zve.verlustabzug", "kst.zve.z69"))

	// Anlage WA.
	c.withholding(data)
	c.resolve("kst.wa.z11e", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: []string{
		"Wertungsfrage: Wurden innerhalb von fünf Jahren mehr als 50 % der Anteile an einen Erwerber übertragen? Dann gehen nicht genutzte Verluste unter."}})
	if yes, ok := c.flag("kst.wa.z11e"); ok && yes {
		c.finding(SeverityBlocking, "schaedlicher_beteiligungserwerb", "kst.wa.z11e",
			"Ein schädlicher Beteiligungserwerb ist angegeben. Den Verlustuntergang und seine Ausnahmen rechnet Buchfink nicht; bitte mit einem Steuerberater klären.",
			"§ 8c KStG")
	}
	c.books("kst.wa.z12", Bool(false), Origin{Kind: OriginNone}, "Den fortführungsgebundenen Verlustvortrag bildet Buchfink nicht ab.")
	c.books("kst.wa.z29", Bool(false), Origin{Kind: OriginNone}, "Nur für Konzernobergesellschaften multinationaler Unternehmensgruppen.")
	c.resolve("kst.wa.z40", noneFigure())

	c.form("KSt1", RequirementMandatory, "§ 31 Abs. 1a KStG")
	c.form("GK", RequirementMandatory, "Gewinn aus Gewerbebetrieb")
	c.form("ZVE", RequirementMandatory, "Ermittlung des zu versteuernden Einkommens")
	c.form("WA", RequirementMandatory, "Anrechnung, § 8c KStG")
	c.form("KSt1F", RequirementMandatory, "§ 27 Abs. 2 Satz 4 KStG: jährlich, auch bei Bestand null")
	c.form("Kontrolle", RequirementControl, "")
	return out
}

// participations rechnet § 8b KStG aus den Konten und der Anlagenkartei.
func (c *calc) participations(common Common, data CorporateData, out *corporate) {
	l := data.Ledger
	dividends, divOrigin := l.income(full(accountsDividends...)...)
	var other domain.Cents
	names := []string{}
	for _, h := range data.Holdings {
		if h.OtherAccounts == 0 {
			continue
		}
		other += h.OtherAccounts
		names = append(names, h.Name)
		divOrigin.EntryIDs = append(divOrigin.EntryIDs, h.EntryIDs...)
	}
	if other != 0 {
		dividends += other
		divOrigin.Text = fmt.Sprintf("Davon %s € aus Anteilen der Anlagenkartei auf anderen Ertragskonten (%s).",
			other, strings.Join(names, ", "))
		c.finding(SeverityHint, "bezuege_ohne_8b_konto", "kst.gk.8b_bezuege",
			fmt.Sprintf("Bezüge aus %s sind nicht auf einem Konto nach § 8b KStG (7005, 7006, 7014, 7015, 7103, 7104) gebucht; sie sind trotzdem berücksichtigt.",
				strings.Join(names, ", ")), "§ 8b Abs. 1 KStG")
	}
	c.books("kst.gk.8b_bezuege", Cents(dividends), divOrigin)

	streubesitz, known, note, trade, tradeKnown, tradeNote := classifyHoldings(common.Year, common.Period, dividends, data.Holdings)
	holdingOrigin := Origin{Kind: OriginAssets, Formula: "Bezüge je Anteil laut Anlagenkartei mit der dort erfassten Quote"}
	switch {
	case dividends == 0:
		c.resolve("kst.gk.8b_streubesitz", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	case known:
		c.resolve("kst.gk.8b_streubesitz", figure{value: Cents(streubesitz), status: StatusProposed, origin: holdingOrigin,
			notes: []string{note}})
	default:
		c.resolve("kst.gk.8b_streubesitz", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: []string{note}})
	}
	streu := c.amount("kst.gk.8b_streubesitz")
	free := dividends - streu
	if free < 0 {
		free = 0
	}
	c.books("kst.gk.8b_frei", Cents(free), fromLines("Bezüge − steuerpflichtige Bezüge aus Streubesitz",
		"kst.gk.8b_bezuege", "kst.gk.8b_streubesitz"))
	five := domain.MulRound(free, 5, 100)
	c.books("kst.gk.8b_5_bezuege", Cents(five), fromLines("5 % der außer Ansatz bleibenden Bezüge", "kst.gk.8b_frei"))
	out.dividendsFree = free
	out.fivePercent = five
	out.tradeStreubesitz, out.tradeStreubesitzKnown, out.tradeStreubesitzNote = trade, tradeKnown, tradeNote

	gain, gainOrigin := l.income(full(append(append([]string{}, accountsSaleGain...), accountsWriteUpShare...)...)...)
	costs, costOrigin := l.expense(full("6857")...)
	gainOrigin.Accounts = append(gainOrigin.Accounts, negateParts(costOrigin.Accounts)...)
	gainOrigin.EntryIDs = append(gainOrigin.EntryIDs, costOrigin.EntryIDs...)
	gain -= costs
	loss, lossOrigin := l.expense(full(accountsShareLoss...)...)
	if gain < 0 {
		// Übersteigen die Veräußerungskosten den Erlös, ist es eine
		// Gewinnminderung nach § 8b Abs. 3 Satz 3 KStG.
		loss -= gain
		lossOrigin.Accounts = append(lossOrigin.Accounts, gainOrigin.Accounts...)
		gain = 0
		gainOrigin = Origin{Kind: OriginNone}
	}
	c.books("kst.gk.8b_gewinne", Cents(gain), gainOrigin)
	c.books("kst.gk.8b_5_gewinne", Cents(domain.MulRound(gain, 5, 100)),
		fromLines("5 % der Gewinne nach § 8b Abs. 2 KStG", "kst.gk.8b_gewinne"))
	c.books("kst.gk.8b_minderungen", Cents(loss), lossOrigin)
	if gain != 0 {
		c.finding(SeverityHint, "8b_teilwertabschreibung_frueher", "kst.gk.8b_gewinne",
			"Wurde ein veräußerter Anteil früher steuerwirksam auf den niedrigeren Teilwert abgeschrieben, bleibt der Gewinn insoweit steuerpflichtig.",
			"§ 8b Abs. 2 Satz 4 KStG")
	}

	if writeOff, o := l.expense(full(accountsLoanWriteOff...)...); writeOff != 0 {
		c.propose("kst.gk.8b_darlehen", Cents(0), o, fmt.Sprintf(
			"Auf Finanzanlagen sind %s € abgeschrieben. Betrifft das ein Darlehen an eine Gesellschaft, an der die GmbH zu mehr als einem Viertel beteiligt ist oder war, ist die Minderung hinzuzurechnen, es sei denn, ein fremder Dritter hätte das Darlehen ebenso gewährt.", writeOff))
	} else {
		c.resolve("kst.gk.8b_darlehen", noneFigure())
	}
	if writeUp, o := l.income(full(accountsWriteUpLoan...)...); writeUp != 0 {
		c.propose("kst.gk.8b_zuschreibung_darlehen", Cents(writeUp), o,
			"Betrifft die Zuschreibung einen Anteil statt einer Darlehensforderung, gehört sie zu den Gewinnen nach § 8b Abs. 2 KStG (mit 5 % nicht abziehbar).")
	} else {
		c.resolve("kst.gk.8b_zuschreibung_darlehen", noneFigure())
	}
}

func negateParts(parts []AccountPart) []AccountPart {
	out := make([]AccountPart, 0, len(parts))
	for _, p := range parts {
		p.Amount = -p.Amount
		out = append(out, p)
	}
	return out
}

// classifyHoldings teilt die Bezüge nach den Stichtagen von § 8b Abs. 4 KStG
// (Beginn des Kalenderjahres, Erwerb ab 10 % rückbezogen) und § 9 Nr. 2a
// GewStG (Beginn des Erhebungszeitraums, ohne Rückbeziehung). Gelingt das
// nicht, weil Bezüge keinem Anteil zugeordnet sind oder eine Quote fehlt,
// meldet sie es.
func classifyHoldings(year int, period Period, dividends domain.Cents, holdings []Holding) (
	streubesitz domain.Cents, known bool, note string,
	trade domain.Cents, tradeKnown bool, tradeNote string,
) {
	if dividends == 0 {
		return 0, true, "", 0, true, ""
	}
	start := fmt.Sprintf("%d-01-01", year)
	var linked domain.Cents
	missing := []string{}
	parts := []string{}
	for _, h := range holdings {
		if h.Dividends == 0 {
			continue
		}
		linked += h.Dividends
		if h.QuotaPermille <= 0 {
			missing = append(missing, h.Name)
			continue
		}
		acquiredInYear := h.AcquiredOn >= start
		// § 8b Abs. 4 KStG: Quote zu Beginn des Kalenderjahres; ein Erwerb von
		// mindestens 10 % gilt als zu Beginn erfolgt.
		if h.QuotaPermille < 100 {
			streubesitz += h.Dividends
		}
		// § 9 Nr. 2a GewStG: Quote zu Beginn des Erhebungszeitraums; ein
		// Erwerb im Jahr zählt nicht.
		if h.QuotaPermille >= 100 && (acquiredInYear || h.QuotaPermille < 150) {
			trade += h.Dividends - domain.MulRound(h.Dividends, 5, 100)
		}
		parts = append(parts, fmt.Sprintf("%s: %s € bei %s Quote%s", h.Name, h.Dividends,
			percentText(int64(h.QuotaPermille)), map[bool]string{true: ", erworben " + h.AcquiredOn, false: ""}[acquiredInYear]))
	}
	switch {
	case linked != dividends:
		msg := fmt.Sprintf("Von %s € Bezügen sind %s € keinem Anteil der Anlagenkartei zugeordnet. "+
			"Die Quote zu Beginn des Kalenderjahres je Beteiligung erfragen (Gesellschafterliste).", dividends, dividends-linked)
		return 0, false, msg, 0, false, msg
	case len(missing) > 0:
		msg := fmt.Sprintf("Für %s ist keine Beteiligungsquote erfasst. Die Quote zu Beginn des Kalenderjahres erfragen.",
			strings.Join(missing, ", "))
		return 0, false, msg, 0, false, msg
	}
	sort.Strings(parts)
	base := "Aus der Anlagenkartei (die Quote dort trägt keinen Stichtag; Quote zu Beginn des Kalenderjahres bestätigen): " +
		strings.Join(parts, "; ") + "."
	return streubesitz, true, base, trade, true, base
}

// investments rechnet Vorabpauschale und Teilfreistellung aus der Kartei.
func (c *calc) investments(data CorporateData, out *corporate) {
	var vp, exempt domain.Cents
	ids := []uint{}
	notes := []string{}
	undetermined := []string{}
	disposed := []string{}
	for _, f := range data.Funds {
		vp += f.Vorabpauschale
		ids = append(ids, f.EntryIDs...)
		if f.DisposedInYear {
			disposed = append(disposed, f.Name)
		}
		base := f.Income + f.Vorabpauschale
		if base == 0 {
			continue
		}
		if !f.Determined {
			undetermined = append(undetermined, f.Name)
			continue
		}
		part := domain.MulRound(base, int64(f.ExemptPermille), 1000)
		exempt += part
		notes = append(notes, fmt.Sprintf("%s: %s × %s = %s €", f.Name, base, percentText(int64(f.ExemptPermille)), part))
	}
	o := Origin{Kind: OriginAssets, EntryIDs: ids, Formula: "Investmentanteile laut Anlagenkartei"}
	if vp != 0 {
		c.propose("kst.gk.inv_vorabpauschale", Cents(vp), o,
			"Wertungsfrage: Die Vorabpauschale gilt am ersten Werktag des Folgejahres als zugeflossen (§ 18 Abs. 3 InvStG); angesetzt ist sie im Wirtschaftsjahr dieses Tages.")
	} else {
		c.resolve("kst.gk.inv_vorabpauschale", noneFigure())
	}
	switch {
	case len(undetermined) > 0:
		c.resolve("kst.gk.inv_teilfreistellung", figure{status: StatusOpen, origin: o, notes: []string{
			"Für " + strings.Join(undetermined, ", ") + " fehlt die Fondsart oder die Anlegerstellung."}})
	case len(disposed) > 0:
		c.resolve("kst.gk.inv_teilfreistellung", figure{status: StatusOpen, origin: o, notes: []string{
			"Im Jahr veräußert: " + strings.Join(disposed, ", ") + ". Die Teilfreistellung des Veräußerungsgewinns (abzüglich der versteuerten Vorabpauschalen) ergänzen."}})
	case exempt != 0:
		c.propose("kst.gk.inv_teilfreistellung", Cents(exempt), o, notes...)
	default:
		c.resolve("kst.gk.inv_teilfreistellung", noneFigure())
	}
	out.partialExemption = c.amount("kst.gk.inv_teilfreistellung")
}

// donations rechnet Anlage Z: Höchstbetrag, Abzug und Vortrag.
func (c *calc) donations(common Common, data CorporateData, income domain.Cents) domain.Cents {
	given, o := data.Ledger.expense(full(accountsDeductible...)...)
	c.books("kst.z.z5", Cents(given), o)
	if data.Founded {
		c.resolve("kst.z.vortrag", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
			notes: []string{"Im ersten Jahr der Steuerpflicht gibt es keinen Vortrag."}})
	} else {
		c.resolve("kst.z.vortrag", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
			notes: []string{"Ohne festgestellten Zuwendungsvortrag null. Stellt der Bescheid des Vorjahres einen Vortrag fest, hier eintragen."}})
	}
	carried := c.amount("kst.z.vortrag")
	relevant := given+carried > 0
	if relevant {
		c.propose("kst.z.z10", Cents(data.Revenue), Origin{Kind: OriginStatement, Formula: "Umsatzerlöse (§ 275 Abs. 2 Nr. 1 HGB)"},
			"Ob nicht steuerbare Umsätze im Ausland zu den gesamten Umsätzen zählen, ist nicht belegt.")
	} else {
		c.resolve("kst.z.z10", figure{value: Cents(data.Revenue), status: StatusAuto,
			origin: Origin{Kind: OriginStatement, Formula: "Umsatzerlöse (§ 275 Abs. 2 Nr. 1 HGB)"}})
	}
	c.books("kst.z.z11", Cents(data.Wages), Origin{Kind: OriginStatement, Formula: "Löhne und Gehälter (§ 275 Abs. 2 Nr. 6 Buchst. a HGB)"})
	capIncome := domain.Cents(0)
	if income > 0 {
		capIncome = domain.MulRound(income, donationPercent, 100)
	}
	capTurnover := domain.MulRound(c.amount("kst.z.z10")+c.amount("kst.z.z11"), donationPermille, 1000)
	limit := capIncome
	if capTurnover > limit {
		limit = capTurnover
	}
	c.books("kst.z.z7", Cents(limit), fromLines(
		fmt.Sprintf("höherer Betrag aus 20 %% von Zeile 32 (%s €) und 4 Promille der Zeilen 10 und 11 (%s €)", capIncome, capTurnover),
		"kst.zve.z32", "kst.z.z10", "kst.z.z11"))
	deductible := given + carried
	if deductible > limit {
		deductible = limit
	}
	c.books("kst.zve.zuwendungen", Cents(deductible), fromLines("geringerer Betrag aus Zuwendungen mit Vortrag und Höchstbetrag",
		"kst.z.z5", "kst.z.vortrag", "kst.z.z7"))
	c.books("kst.z.vortrag_ende", Cents(given+carried-deductible), fromLines("Zuwendungen + Vortrag − Abzug",
		"kst.z.z5", "kst.z.vortrag", "kst.zve.zuwendungen"))
	if given+carried > limit {
		c.finding(SeverityHint, "zuwendungen_ueber_hoechstbetrag", "kst.z.z7",
			fmt.Sprintf("Die Zuwendungen übersteigen den Höchstbetrag; %s € werden vorgetragen.", given+carried-limit),
			"§ 9 Abs. 1 Nr. 2 Satz 9 KStG")
	}
	if relevant {
		c.form("Z", RequirementMandatory, "Zuwendungen nach § 9 Abs. 1 Nr. 2 KStG")
	} else {
		c.form("Z", RequirementNone, "Keine Zuwendungen und kein Vortrag")
	}
	if l := data.Ledger; l.hasBalance("6394") {
		c.finding(SeverityHint, "parteispenden", "kst.gk.spenden",
			"Zuwendungen an politische Parteien sind hinzugerechnet und nicht abziehbar.", "§ 4 Abs. 6 EStG, § 9 Abs. 1 Nr. 2 KStG")
	}
	return deductible
}

// losses rechnet Anlage Verluste und den Verlustabzug.
func (c *calc) losses(common Common, data CorporateData, income domain.Cents) {
	if data.Founded {
		c.resolve("kst.v.z14", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
			notes: []string{"Im ersten Jahr der Steuerpflicht gibt es keinen Verlustvortrag."}})
	} else {
		c.resolve("kst.v.z14", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{"Verbleibenden Verlustvortrag aus dem Feststellungsbescheid des Vorjahres übernehmen; ohne Verlust null."}})
	}
	carried := c.amount("kst.v.z14")
	permille := corporateLossPermille(common.Year)
	deduction := lossDeduction(carried, income, permille)
	c.books("kst.zve.verlustabzug", Cents(deduction), fromLines(
		fmt.Sprintf("Verlustvortrag bis 1 Mio. € voll, darüber %s des übersteigenden Gesamtbetrags", percentText(permille)),
		"kst.v.z14", "kst.zve.gesamtbetrag"))
	loss := domain.Cents(0)
	if income < 0 {
		loss = -income
		c.resolve("kst.v.z20", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}, notes: []string{
			"Wertungsfrage: Ohne Eintrag trägt das Finanzamt den Verlust in größtmöglicher Höhe in die zwei Vorjahre zurück; „0“ verzichtet auf den Rücktrag."}})
		c.finding(SeverityHint, "verlustruecktrag", "kst.v.z20",
			fmt.Sprintf("Das Jahr schließt mit einem Verlust von %s €. Rücktrag oder Vortrag ist eine Entscheidung des Menschen.", loss),
			"§ 10d Abs. 1 EStG")
	} else {
		c.resolve("kst.v.z20", noneFigure("Kein Verlust im Jahr."))
	}
	carryBack := c.amount("kst.v.z20")
	end := carried - deduction + loss - carryBack
	if end < 0 {
		end = 0
	}
	c.books("kst.v.ende", Cents(end), fromLines("Anfangsbestand − Abzug + Verlust des Jahres − Rücktrag",
		"kst.v.z14", "kst.zve.verlustabzug", "kst.zve.gesamtbetrag", "kst.v.z20"))
	switch {
	case carried > 0 || loss > 0:
		c.form("Verluste", RequirementMandatory, "§ 10d Abs. 4 EStG")
	case data.Founded:
		c.form("Verluste", RequirementNone, "Kein Verlust und kein Vortrag")
	default:
		c.form("Verluste", RequirementIfNeeded, "Nur bei Verlust oder festgestelltem Verlustvortrag")
	}
}

// withholding setzt die anrechenbaren Abzugsteuern; angerechnet wird nur mit
// Steuerbescheinigung.
func (c *calc) withholding(data CorporateData) {
	for _, item := range []struct{ key, account string }{{"kst.wa.kapest", "7630"}, {"kst.wa.solz", "7633"}} {
		amount, o := data.Ledger.expense(full(item.account)...)
		if amount == 0 {
			c.resolve(item.key, figure{value: Cents(0), status: StatusAuto, origin: o})
			continue
		}
		c.propose(item.key, Cents(amount), o, "Nur anzurechnen, soweit eine Steuerbescheinigung vorliegt; sie als Beleg ablegen.")
	}
}

// contributionAccount rechnet die Anlage KSt 1 F.
func (c *calc) contributionAccount(common Common, data CorporateData) {
	var paid domain.Cents
	count := 0
	dates := []string{}
	for _, d := range data.Distributions {
		if d.Amount == 0 || d.DecisionDate < common.Period.From || d.DecisionDate > common.Period.To {
			continue
		}
		paid += d.Amount
		count++
		dates = append(dates, fmt.Sprintf("Beschluss vom %s über das Ergebnis %d: %s €", d.DecisionDate, d.ForYear, d.Amount))
	}
	vga := c.amount("kst.gk.vga")
	if vga > 0 {
		count++
	}
	c.resolve("kst.f.leistungen", figure{value: Cents(paid + vga), status: StatusAuto,
		origin: Origin{Kind: OriginComputation, FromLines: []string{"kst.gk.vga"},
			Formula: "Ausschüttungen laut Gewinnverwendungsbeschlüssen im Wirtschaftsjahr + verdeckte Gewinnausschüttungen",
			Text:    strings.Join(dates, "; ")},
		notes: []string{"Maßgeblich ist der Abfluss im Wirtschaftsjahr; gezählt sind die Beschlüsse mit Datum im Wirtschaftsjahr."}})
	c.books("kst.f.z13", Bool(count > 1), fromLines("mehr als eine Leistung", "kst.f.leistungen"))

	equityOrigin := Origin{Kind: OriginStatement, Formula: "Eigenkapital zum Schluss des Vorjahres + Unterschied der Steuerbilanz"}
	switch {
	case data.Founded:
		c.resolve("kst.f.ek_vorjahr", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
			notes: []string{"Im ersten Wirtschaftsjahr gibt es kein Vorjahr."}})
		c.resolve("kst.f.z18", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone}})
	case data.PriorKnown:
		c.resolve("kst.f.ek_vorjahr", figure{value: Cents(data.PriorEquity + data.PriorTaxBalanceEffect), status: StatusAuto, origin: equityOrigin})
		c.resolve("kst.f.z18", figure{value: Cents(data.PriorSubscribed), status: StatusAuto,
			origin: Origin{Kind: OriginStatement, Formula: "Gezeichnetes Kapital (Posten A.I) zum Schluss des Vorjahres"}})
	default:
		c.resolve("kst.f.ek_vorjahr", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{"Die Bücher enthalten das Vorjahr nicht. Eigenkapital laut Steuerbilanz des Vorjahres eintragen."}})
		c.resolve("kst.f.z18", figure{status: StatusOpen, origin: Origin{Kind: OriginNone}})
	}
	if data.Founded {
		c.resolve("kst.f.z40", figure{value: Cents(0), status: StatusAuto, origin: Origin{Kind: OriginNone},
			notes: []string{"Bei Gründung beginnt das Einlagekonto mit null; ein Aufgeld steht bei den Einlagen des Jahres."}})
		c.resolve("kst.f.bestand_vorjahr", noneFigure("Im ersten Wirtschaftsjahr gibt es keinen festgestellten Bestand."))
	} else {
		c.resolve("kst.f.z40", noneFigure())
		c.resolve("kst.f.bestand_vorjahr", figure{status: StatusOpen, origin: Origin{Kind: OriginNone},
			notes: []string{"Bestand aus dem Feststellungsbescheid zum Schluss des Vorjahres übernehmen."}})
	}
	opening := c.amount("kst.f.z40") + c.amount("kst.f.bestand_vorjahr")
	distributable := c.amount("kst.f.ek_vorjahr") - c.amount("kst.f.z18") - c.amount("kst.f.bestand_vorjahr")
	c.books("kst.f.ausschuettbar", Cents(distributable), fromLines("Eigenkapital − Nennkapital − Bestand des Einlagekontos",
		"kst.f.ek_vorjahr", "kst.f.z18", "kst.f.bestand_vorjahr"))

	contributions, o := data.Ledger.income(full(accountsCapitalRes...)...)
	if contributions != 0 {
		c.propose("kst.f.einlagen", Cents(contributions), o,
			"Zugänge auf den Konten der Kapitalrücklage; eine verdeckte Einlage ohne Buchung auf diesen Konten ergänzen.")
	} else {
		c.resolve("kst.f.einlagen", figure{value: Cents(0), status: StatusAuto, origin: o})
	}
	if contributions < 0 {
		c.finding(SeverityBlocking, "kapitalruecklage_entnommen", "kst.f.einlagen",
			"Die Kapitalrücklage wurde im Jahr gemindert. Eine Einlagenrückgewähr bildet Buchfink nicht ab; bitte mit einem Steuerberater klären.",
			"§ 27 KStG")
	}
	c.resolve("kst.f.z91", noneFigure())
	c.resolve("kst.f.z95", noneFigure())
	usable := distributable
	if usable < 0 {
		usable = 0
	}
	use := c.amount("kst.f.leistungen") - usable
	if use < 0 {
		use = 0
	}
	c.books("kst.f.verwendung", Cents(use), fromLines("Leistungen − ausschüttbarer Gewinn, nicht unter null",
		"kst.f.leistungen", "kst.f.ausschuettbar"))
	if use > 0 {
		c.finding(SeverityBlocking, "einlagenrueckgewaehr", "kst.f.verwendung",
			fmt.Sprintf("Die Leistungen des Jahres übersteigen den ausschüttbaren Gewinn um %s €. Das ist eine Einlagenrückgewähr; sie braucht eine Bescheinigung nach § 27 Abs. 3 KStG und gehört zu einem Steuerberater.", use),
			"§ 27 Abs. 1 Satz 3 KStG")
	}
	end := opening + c.amount("kst.f.einlagen") + c.amount("kst.f.z91") + c.amount("kst.f.z95") - use
	c.books("kst.f.bestand_ende", Cents(end), fromLines("Anfangsbestand + Einlagen − Verwendung",
		"kst.f.z40", "kst.f.bestand_vorjahr", "kst.f.einlagen", "kst.f.z91", "kst.f.z95", "kst.f.verwendung"))
}

// corporateControl rechnet die Steuer zur Kontrolle des Bescheids und der
// Rückstellung.
func (c *calc) corporateControl(common Common, data CorporateData) {
	rate := corporateTaxPermille(common.Year)
	c.books("kst.k.satz", Text(percentText(rate)), Origin{Kind: OriginComputation, Text: "§ 23 Abs. 1 KStG"})
	zve := c.amount("kst.zve.zve")
	tax := domain.Cents(0)
	if zve > 0 {
		tax = truncEuro(domain.Cents(int64(zve) * rate / 1000))
	}
	c.books("kst.k.kst", Cents(tax), fromLines(fmt.Sprintf("%s des zu versteuernden Einkommens, Cent abgeschnitten", percentText(rate)), "kst.zve.zve"))
	solz := domain.Cents(int64(tax) * solidarityPermille / 1000)
	c.books("kst.k.solz", Cents(solz), fromLines("5,5 % der Körperschaftsteuer, Bruchteile eines Cents abgeschnitten", "kst.k.kst"))
	prepaid, o := data.Ledger.sum(full("7600"), true, AccountTotals.NetBeforeClosing)
	c.books("kst.k.vz_kst", Cents(prepaid), o)
	prepaidSolz, so := data.Ledger.sum(full("7608"), true, AccountTotals.NetBeforeClosing)
	c.books("kst.k.vz_solz", Cents(prepaidSolz), so)
	kapest := ceilEuro(c.amount("kst.wa.kapest"))
	c.books("kst.k.kapest", Cents(kapest), fromLines("auf volle Euro aufgerundet", "kst.wa.kapest"))
	c.books("kst.k.solz_kapest", Cents(c.amount("kst.wa.solz")), fromLines("Übertrag", "kst.wa.solz"))
	result := tax + solz - prepaid - prepaidSolz - kapest - c.amount("kst.wa.solz")
	c.books("kst.k.ergebnis", Cents(result), fromLines("Steuer − Vorauszahlungen − Anrechnung",
		"kst.k.kst", "kst.k.solz", "kst.k.vz_kst", "kst.k.vz_solz", "kst.k.kapest", "kst.k.solz_kapest"))
	provision := -data.Ledger["3040"].EndBalance()
	c.books("kst.k.rueckstellung", Cents(provision), Origin{Kind: OriginAccounts, Accounts: []AccountPart{{
		Account: "3040", Name: data.Ledger["3040"].Name, Balance: provision, Num: 1, Den: 1, Amount: provision}}})
	expected := tax + solz - prepaid - prepaidSolz
	if expected < 0 {
		expected = 0
	}
	if diff := provision - expected; diff.Abs() > 100 {
		c.finding(SeverityWarning, "rueckstellung_abweichung", "kst.k.rueckstellung",
			fmt.Sprintf("Die gebuchte Körperschaftsteuerrückstellung (%s €) weicht von der Kontrollrechnung (%s € Steuer und Solidaritätszuschlag nach Vorauszahlungen) um %s € ab.",
				provision, expected, diff), "§ 249 Abs. 1 HGB")
	}
}

// corporateChecks sind die Plausibilitätsprüfungen und Grenzen.
func (c *calc) corporateChecks(common Common, data CorporateData) {
	if common.Basis.ClosingState == string(domain.FiscalYearOpen) {
		c.finding(SeverityBlocking, "abschluss_offen", "",
			"Der Jahresabschluss ist noch nicht aufgestellt. Die Erklärung baut auf dem aufgestellten Abschluss auf.", "§ 5b EStG, § 60 EStDV")
	}
	interest, _ := data.Ledger.expense(full("7300", "7309", "7310", "7316", "7317", "7318", "7319", "7320", "7325", "7326", "7329", "7330", "7339")...)
	interestIncome, _ := data.Ledger.income(full("7100", "7110", "7120")...)
	if interest-interestIncome >= 3_000_000_00 {
		c.finding(SeverityBlocking, "zinsschranke", "",
			"Der Zinsaufwand übersteigt den Zinsertrag um mindestens 3 Mio. €. Die Zinsschranke rechnet Buchfink nicht; bitte mit einem Steuerberater klären.",
			"§ 4h EStG, § 8a KStG")
	}
}
