package accounting

import (
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// accountsWith baut Konten des echten SKR04-Katalogs mit vorgegebenen Salden.
// Positiv heißt Sollsaldo — dieselbe Richtung, in der die Gliederung rechnet.
func accountsWith(t *testing.T, balances map[string]domain.Cents) []domain.Account {
	t.Helper()
	chart := NewChart(DefaultSKR04Accounts())
	out := make([]domain.Account, 0, len(balances))
	for number, value := range balances {
		acc, ok := chart.Lookup(number)
		if !ok {
			t.Fatalf("Konto %s steht nicht im SKR04-Katalog", number)
		}
		if value >= 0 {
			acc.DebitSum = value
		} else {
			acc.CreditSum = -value
		}
		out = append(out, acc)
	}
	return out
}

func lineAmount(t *testing.T, stmt *domain.Statement, key string) domain.Cents {
	t.Helper()
	line := stmt.Line(key)
	if line == nil {
		t.Fatalf("die Gliederung kennt die Position %s nicht", key)
	}
	return line.Amount
}

// Jede Position des Katalogs muss ein Ziel haben. Fehlt eines, verschwindet ein
// Konto beim Aufstellen der Bilanz — und zwar lautlos.
func TestEverySKR04PositionHasATarget(t *testing.T) {
	cat, err := GetSKR04Catalog()
	if err != nil {
		t.Fatalf("SKR04-Katalog: %v", err)
	}
	if len(cat.Positions) == 0 {
		t.Fatal("der Katalog enthält keine Positionen")
	}

	for _, position := range cat.Positions {
		target, ok := positionTargets[position.ID]
		if !ok {
			t.Errorf("die SKR04-Position %q (%s, %q) hat keine Gliederungsposition",
				position.ID, position.HGBCode, position.Name)
			continue
		}
		if _, known := lineIndex[target.Key]; !known {
			t.Errorf("die SKR04-Position %q zeigt auf die unbekannte Gliederungsposition %q",
				position.ID, target.Key)
		}
		if target.AltKey != "" {
			if _, known := lineIndex[target.AltKey]; !known {
				t.Errorf("die SKR04-Position %q nennt die unbekannte Gegenposition %q",
					position.ID, target.AltKey)
			}
		}
	}

	if PositionCount() != len(cat.Positions) {
		t.Errorf("die Zuordnungstabelle führt %d Positionen, der Katalog %d",
			PositionCount(), len(cat.Positions))
	}
}

// Die Gliederung selbst muss in sich stimmen: jede Unterposition braucht eine
// Oberposition, und die Ebenen dürfen nicht springen.
func TestStatementTemplateIsConsistent(t *testing.T) {
	for _, def := range allLines {
		if def.Parent == "" {
			if def.Level != 1 {
				t.Errorf("%s steht auf Ebene %d, hat aber keine Oberposition", def.Key, def.Level)
			}
			continue
		}
		parent, ok := lineIndex[def.Parent]
		if !ok {
			t.Errorf("%s verweist auf die unbekannte Oberposition %s", def.Key, def.Parent)
			continue
		}
		if parent.Level >= def.Level {
			t.Errorf("%s (Ebene %d) steht unter %s (Ebene %d)",
				def.Key, def.Level, parent.Key, parent.Level)
		}
		if parent.Section != def.Section {
			t.Errorf("%s steht im Abschnitt %s, seine Oberposition %s im Abschnitt %s",
				def.Key, def.Section, parent.Key, parent.Section)
		}
	}
}

// Das S/H-Konto: dieselben Konten sind Forderung oder Verbindlichkeit, je
// nachdem, wie sie stehen. Die Vorsteuer im Haben ist eine Verbindlichkeit.
func TestSignDecidesTheSideOfDualAccounts(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"0440": 4_000_000,  // Maschinen im Soll
		"1800": -3_000_000, // Bank im Haben
		"1400": -1_000_000, // Vorsteuer im Haben
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}

	if got := lineAmount(t, stmt, "passiva.C.2"); got != 3_000_000 {
		t.Errorf("die Bank im Haben gehört unter die Verbindlichkeiten gegenüber Kreditinstituten, "+
			"dort stehen aber %s €", got)
	}
	if got := lineAmount(t, stmt, "aktiva.B.IV"); got != 0 {
		t.Errorf("unter den flüssigen Mitteln stehen %s €, obwohl das Bankkonto im Haben steht", got)
	}
	if got := lineAmount(t, stmt, "passiva.C.8"); got != 1_000_000 {
		t.Errorf("die Vorsteuer im Haben gehört unter die sonstigen Verbindlichkeiten, "+
			"dort stehen aber %s €", got)
	}
	if got := lineAmount(t, stmt, "aktiva.B.II.4"); got != 0 {
		t.Errorf("unter den sonstigen Vermögensgegenständen stehen %s €, obwohl die Vorsteuer im Haben steht", got)
	}

	switched := map[string]string{}
	for _, s := range stmt.Assignment.SignSwitches {
		switched[s.Account] = s.To
	}
	for account, want := range map[string]string{"1800": "passiva.C.2", "1400": "passiva.C.8"} {
		if switched[account] != want {
			t.Errorf("der Zuordnungsbericht meldet für Konto %s die Position %q, erwartet %q",
				account, switched[account], want)
		}
	}
}

// Das Vorzeichen entscheidet auch andersherum: die Umsatzsteuer im Soll ist
// eine Forderung.
func TestDebitBalanceOnALiabilityAccountMovesToTheAssetSide(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"3820": 500_000,  // Umsatzsteuer-Vorauszahlungen im Soll
		"1800": -500_000, // Gegenkonto
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if got := lineAmount(t, stmt, "aktiva.B.II.4"); got != 500_000 {
		t.Errorf("die Umsatzsteuer-Vorauszahlung im Soll gehört unter die sonstigen "+
			"Vermögensgegenstände, dort stehen aber %s €", got)
	}
}

// Die Bilanz muss aufgehen, und das Jahresergebnis steht auf A.V.
func TestBalanceSheetBalancesAndCarriesTheResult(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"1800": 10_000_000, // Bank
		"2900": -2_500_000, // Gezeichnetes Kapital
		"4400": -7_500_000, // Erlöse 19 %
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}

	if stmt.TotalAssets != stmt.TotalLiabilities {
		t.Fatalf("Aktiva %s €, Passiva %s €", stmt.TotalAssets, stmt.TotalLiabilities)
	}
	if stmt.TotalAssets != 10_000_000 {
		t.Errorf("die Bilanzsumme beträgt %s €, erwartet 100.000,00 €", stmt.TotalAssets)
	}
	if stmt.NetIncome != 7_500_000 {
		t.Errorf("das Jahresergebnis beträgt %s €, erwartet 75.000,00 €", stmt.NetIncome)
	}
	if got := lineAmount(t, stmt, "passiva.A.V"); got != 7_500_000 {
		t.Errorf("auf A.V stehen %s €, erwartet das Jahresergebnis von 75.000,00 €", got)
	}
	if got := lineAmount(t, stmt, "passiva.A.I"); got != 2_500_000 {
		t.Errorf("das gezeichnete Kapital beträgt %s €, erwartet 25.000,00 €", got)
	}
	if stmt.Revenue != 7_500_000 {
		t.Errorf("die Umsatzerlöse betragen %s €, erwartet 75.000,00 €", stmt.Revenue)
	}
}

// Eine Bilanz, deren Seiten verschieden sind, ist keine Bilanz. Die Funktion
// muss die Differenz nennen, nicht bloß scheitern.
func TestBuildStatementRefusesAnUnbalancedSheet(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{"1800": 10_000_000})
	// Ein Saldo auf dem Vortragskonto der Klasse 9 gehört in keine Bilanzseite —
	// genau der Fall, den ein unvollständiger Saldenvortrag hinterlässt.
	accounts = append(accounts, accountsWith(t, map[string]domain.Cents{"9000": -10_000_000})...)

	_, err := BuildStatement(accounts, nil, domain.DepthFull)
	if err == nil {
		t.Fatal("eine Bilanz, die nicht aufgeht, darf nicht entstehen")
	}
	for _, want := range []string{"geht nicht auf", "Differenz", "statistischen Konten"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("die Fehlermeldung nennt %q nicht: %v", want, err)
		}
	}
}

// Die Gliederungstiefe ändert die Zeilen, nicht die Summen.
func TestDepthsAgreeOnTheTotals(t *testing.T) {
	balances := map[string]domain.Cents{
		"0440": 4_000_000,
		"0640": 1_000_000,
		"1200": 2_000_000,
		"1800": 3_000_000,
		"2900": -2_500_000,
		"3300": -1_500_000,
		"4400": -8_000_000,
		"6300": 2_000_000,
	}

	var reference *domain.Statement
	for _, depth := range []domain.StatementDepth{domain.DepthFull, domain.DepthShort, domain.DepthLetters} {
		stmt, err := BuildStatement(accountsWith(t, balances), nil, depth)
		if err != nil {
			t.Fatalf("Bilanz in der Tiefe %s: %v", depth, err)
		}
		if reference == nil {
			reference = stmt
			continue
		}
		if stmt.TotalAssets != reference.TotalAssets || stmt.TotalLiabilities != reference.TotalLiabilities {
			t.Errorf("die Tiefe %s ergibt Aktiva %s € / Passiva %s €, die Vollgliederung %s € / %s €",
				depth, stmt.TotalAssets, stmt.TotalLiabilities,
				reference.TotalAssets, reference.TotalLiabilities)
		}
		if stmt.NetIncome != reference.NetIncome {
			t.Errorf("die Tiefe %s ergibt das Jahresergebnis %s €, die Vollgliederung %s €",
				depth, stmt.NetIncome, reference.NetIncome)
		}
		// Das Anlagevermögen muss in jeder Tiefe denselben Wert tragen, auch
		// wenn seine Unterpositionen nicht mehr erscheinen.
		if got, want := lineAmount(t, stmt, "aktiva.A"), lineAmount(t, reference, "aktiva.A"); got != want {
			t.Errorf("in der Tiefe %s steht das Anlagevermögen auf %s €, in der Vollgliederung auf %s €",
				depth, got, want)
		}
		if stmt.Line("aktiva.A.II.2") != nil {
			t.Errorf("die Tiefe %s zeigt die arabische Ziffer aktiva.A.II.2, obwohl sie das nicht darf", depth)
		}
	}
}

// In jeder Tiefe muss der Weg von der Zeile zum Konto offen bleiben: die
// Konten der weggefallenen Unterpositionen hängen an der Zeile, die noch da ist.
func TestAccountsStayReachableAtEveryDepth(t *testing.T) {
	balances := map[string]domain.Cents{
		"0440": 4_000_000,
		"1800": -4_000_000,
	}
	stmt, err := BuildStatement(accountsWith(t, balances), nil, domain.DepthLetters)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	line := stmt.Line("aktiva.A")
	if line == nil {
		t.Fatal("die Buchstabengliederung kennt das Anlagevermögen nicht")
	}
	found := false
	for _, acc := range line.Accounts {
		if acc.Number == "0440" {
			found = true
		}
	}
	if !found {
		t.Error("das Konto 0440 ist in der Buchstabengliederung unter A. Anlagevermögen nicht erreichbar")
	}
}

// Die Staffel des § 275 Abs. 2 HGB in der Fassung nach dem BilRUG: Nr. 15 ist
// das Ergebnis nach Steuern, Nr. 17 der Jahresüberschuss.
func TestIncomeStatementSubtotals(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"4400": -10_000_000, // Umsatzerlöse
		"6300": 3_000_000,   // sonstige betriebliche Aufwendungen
		"7600": 1_000_000,   // Körperschaftsteuer
		"7650": 500_000,     // sonstige Betriebssteuern
		"1800": 5_500_000,   // Bank als Gegenkonto
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Gewinn- und Verlustrechnung: %v", err)
	}

	for _, want := range []struct {
		key    string
		amount domain.Cents
	}{
		{"guv.1", 10_000_000},
		{"guv.8", -3_000_000},
		{"guv.14", -1_000_000},
		{"guv.15", 6_000_000},
		{"guv.16", -500_000},
		{"guv.17", 5_500_000},
	} {
		if got := lineAmount(t, stmt, want.key); got != want.amount {
			t.Errorf("%s (%s) steht auf %s €, erwartet %s €",
				want.key, LineLabel(want.key), got, want.amount)
		}
	}
	if stmt.NetIncome != 5_500_000 {
		t.Errorf("das Jahresergebnis beträgt %s €, erwartet 55.000,00 €", stmt.NetIncome)
	}
}

// Die Vorjahresspalte des § 265 Abs. 2 HGB steht neben derselben Zeile — und
// zwar dort, wo der Posten im Vorjahr stand.
func TestPriorYearColumnFollowsItsOwnSign(t *testing.T) {
	current := accountsWith(t, map[string]domain.Cents{
		"1800": 4_000_000,
		"0440": 1_000_000,
		"2900": -5_000_000,
	})
	prior := accountsWith(t, map[string]domain.Cents{
		"1800": -1_000_000, // im Vorjahr im Haben: eine Verbindlichkeit
		"0440": 6_000_000,
		"2900": -5_000_000,
	})

	stmt, err := BuildStatement(current, prior, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if !stmt.HasPrior {
		t.Error("die Bilanz meldet keine Vorjahresspalte, obwohl Vorjahressalden übergeben wurden")
	}
	bank := stmt.Line("aktiva.B.IV")
	if bank.Amount != 4_000_000 || bank.PriorAmount != 0 {
		t.Errorf("die flüssigen Mittel stehen auf %s € (Vorjahr %s €), erwartet 40.000,00 € und 0,00 €",
			bank.Amount, bank.PriorAmount)
	}
	credit := stmt.Line("passiva.C.2")
	if credit.Amount != 0 || credit.PriorAmount != 1_000_000 {
		t.Errorf("die Verbindlichkeiten gegenüber Kreditinstituten stehen auf %s € (Vorjahr %s €), "+
			"erwartet 0,00 € und 10.000,00 €", credit.Amount, credit.PriorAmount)
	}
	if stmt.TotalAssetsPrior != 6_000_000 {
		t.Errorf("die Bilanzsumme des Vorjahres beträgt %s €, erwartet 60.000,00 €", stmt.TotalAssetsPrior)
	}
}

// Ein Konto ohne Gliederungsposition darf nicht verschwinden: es steht in der
// Position „Nicht zugeordnet" und im Zuordnungsbericht.
func TestUnassignedAccountIsReported(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{"1800": -1_000_000})
	accounts = append(accounts, domain.Account{
		Number: "1234", Name: "Konto ohne Position", Type: domain.AccountTypeAsset,
		PositionID: "bilanz.aktiva_x.gibt_es_nicht", DebitSum: 1_000_000,
	})

	stmt, err := BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if len(stmt.Assignment.Unassigned) != 1 || stmt.Assignment.Unassigned[0].Number != "1234" {
		t.Fatalf("der Zuordnungsbericht meldet %d nicht zugeordnete Konten, erwartet das Konto 1234",
			len(stmt.Assignment.Unassigned))
	}
	if !stmt.Assignment.HasFindings() {
		t.Error("der Zuordnungsbericht meldet keine Befunde, obwohl ein Konto ohne Position vorliegt")
	}
	if got := lineAmount(t, stmt, "aktiva.X"); got != 1_000_000 {
		t.Errorf("in der Position „Nicht zugeordnet\" stehen %s €, erwartet 10.000,00 €", got)
	}
}

// Die Auffangpositionen werden gezählt: ein Abschluss, dessen Gewicht in den
// „sonstigen" Posten liegt, ist richtig, aber wenig aussagekräftig.
func TestFallbackPositionsAreCounted(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"1400": 500_000,  // sonstige Vermögensgegenstände
		"1800": -500_000, // Gegenkonto
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	found := false
	for _, fallback := range stmt.Assignment.Fallbacks {
		if fallback.Key == "aktiva.B.II.4" {
			found = true
			if fallback.Accounts != 1 || fallback.Amount != 500_000 {
				t.Errorf("die Auffangposition zählt %d Konten mit %s €, erwartet 1 Konto mit 5.000,00 €",
					fallback.Accounts, fallback.Amount)
			}
		}
	}
	if !found {
		t.Error("die Auffangposition „sonstige Vermögensgegenstände\" fehlt im Zuordnungsbericht")
	}
}

// Die Bilanzsumme des § 267 Abs. 4a HGB lässt die nicht eingeforderten
// ausstehenden Einlagen außen vor — sie stehen vor Buchstabe A.
func TestBalanceSheetTotalExcludesOutstandingContributions(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"1800": 2_500_000,  // Bank
		"0050": 2_500_000,  // Nicht eingeforderte ausstehende Einlagen
		"2900": -5_000_000, // Gezeichnetes Kapital
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if stmt.TotalAssets != 5_000_000 {
		t.Fatalf("die Summe der Aktivseite beträgt %s €, erwartet 50.000,00 €", stmt.TotalAssets)
	}
	if stmt.BalanceSheetTotal != 2_500_000 {
		t.Errorf("die Bilanzsumme nach § 267 Abs. 4a HGB beträgt %s €, erwartet 25.000,00 €",
			stmt.BalanceSheetTotal)
	}
}

// Vollständigkeit allein sagt nichts: die Tabelle darf jede Katalogposition
// kennen und sie trotzdem auf die falsche Zeile legen. Der SKR04-Katalog trägt
// bei mehreren Positionen einen um eine Zeile verschobenen Namen und hgb_code —
// deshalb prüft dieser Test den Inhalt, Konto für Konto, gegen die Zeile, in der
// das Konto nach §§ 266, 275 HGB auszuweisen ist.
func TestAccountsLandInTheRightPosition(t *testing.T) {
	chart := NewChart(DefaultSKR04Accounts())
	for _, want := range []struct {
		account string
		key     string
		why     string
	}{
		// Anlage- und Umlaufvermögen.
		{"0440", "aktiva.A.II.2", "Maschinen sind technische Anlagen und Maschinen"},
		{"0520", "aktiva.A.II.3", "der Pkw gehört zur Betriebs- und Geschäftsausstattung"},
		{"1600", "aktiva.B.IV", "die Kasse gehört zu den flüssigen Mitteln"},
		{"1900", "aktiva.C", "die aktive Rechnungsabgrenzung ist Posten C der Aktivseite"},
		{"1950", "aktiva.D", "aktive latente Steuern sind Posten D der Aktivseite"},
		// Eigenkapital: der Katalogname von 2963 ff. ist verschoben.
		{"2900", "passiva.A.I", "das gezeichnete Kapital ist Posten A.I"},
		{"2920", "passiva.A.II", "die Kapitalrücklage ist Posten A.II"},
		{"2960", "passiva.A.III.4", "andere Gewinnrücklagen sind Posten A.III.4"},
		{"2963", "passiva.A.III.4", "Gewinnrücklagen aus den Übergangsvorschriften des BilMoG sind Gewinnrücklagen, kein gezeichnetes Kapital"},
		{"3020", "passiva.B.2", "Steuerrückstellungen sind Posten B.2"},
		{"3150", "passiva.C.2", "Verbindlichkeiten gegenüber Kreditinstituten sind Posten C.2"},
		// Die Staffel: hier liegen die verschobenen Katalognamen dicht beieinander.
		{"4400", "guv.1", "Erlöse sind Umsatzerlöse"},
		{"4600", "guv.1", "unentgeltliche Wertabgaben gehören nach der DATEV-Zuordnung zu den Umsatzerlösen"},
		{"4700", "guv.1", "Erlösschmälerungen sind nach § 277 Abs. 1 HGB von den Umsatzerlösen abzusetzen"},
		{"4730", "guv.1", "gewährte Skonti sind Erlösschmälerungen"},
		{"4800", "guv.2", "Bestandsveränderungen fertiger Erzeugnisse sind Nummer 2"},
		{"4820", "guv.3", "andere aktivierte Eigenleistungen sind Nummer 3"},
		{"4830", "guv.4", "sonstige betriebliche Erträge sind Nummer 4"},
		{"4840", "guv.4", "Erträge aus der Währungsumrechnung sind keine Umsatzerlöse"},
		{"4845", "guv.4", "der Buchgewinn aus dem Verkauf von Sachanlagen ist kein Umsatzerlös"},
		{"5900", "guv.5.b", "Fremdleistungen sind Aufwendungen für bezogene Leistungen"},
		{"6020", "guv.6.a", "Gehälter sind Löhne und Gehälter"},
		{"6110", "guv.6.b", "gesetzliche soziale Aufwendungen sind soziale Abgaben"},
		{"6200", "guv.7.a", "Abschreibungen auf immaterielle Vermögensgegenstände sind Nummer 7 a), kein Personalaufwand"},
		{"6220", "guv.7.a", "Abschreibungen auf Sachanlagen sind Nummer 7 a)"},
		{"6270", "guv.7.b", "Abschreibungen auf Umlaufvermögen sind Nummer 7 b)"},
		{"6300", "guv.8", "sonstige betriebliche Aufwendungen sind Nummer 8"},
		{"7000", "guv.9", "Erträge aus Beteiligungen sind Nummer 9"},
		{"7010", "guv.10", "Erträge aus anderen Wertpapieren sind Nummer 10"},
		{"7100", "guv.11", "sonstige Zinserträge sind Nummer 11"},
		{"7200", "guv.12", "Abschreibungen auf Finanzanlagen sind Nummer 12"},
		{"7300", "guv.13", "Zinsaufwendungen sind Nummer 13"},
		{"7600", "guv.14", "die Körperschaftsteuer ist Nummer 14"},
		{"7650", "guv.16", "sonstige Betriebssteuern sind Nummer 16"},
	} {
		acc, ok := chart.Lookup(want.account)
		if !ok {
			t.Errorf("Konto %s steht nicht im SKR04-Katalog", want.account)
			continue
		}
		got, known := StatementKeyForAccount(acc)
		if !known {
			t.Errorf("Konto %s (%s) hat keine Gliederungsposition", want.account, acc.Name)
			continue
		}
		if got != want.key {
			t.Errorf("Konto %s (%s) steht unter %s (%s), erwartet %s (%s): %s",
				want.account, acc.Name, got, LineLabel(got), want.key, LineLabel(want.key), want.why)
		}
	}
}

// Erlösschmälerungen mindern die Umsatzerlöse — und damit auch das Merkmal der
// Größenklasse nach § 267 HGB. Der Sollsaldo darf nicht als negative
// Eigenleistung an anderer Stelle stehen.
func TestRevenueReductionsLowerTheRevenueLine(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"4400": -10_000_000, // Erlöse 19 %
		"4700": 1_000_000,   // Erlösschmälerungen im Soll
		"1200": 9_000_000,   // Forderungen als Gegenkonto
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Gewinn- und Verlustrechnung: %v", err)
	}
	if got := lineAmount(t, stmt, "guv.1"); got != 9_000_000 {
		t.Errorf("die Umsatzerlöse stehen auf %s €, erwartet 90.000,00 € nach Abzug der Erlösschmälerungen", got)
	}
	if stmt.Revenue != 9_000_000 {
		t.Errorf("das Merkmal Umsatzerlöse beträgt %s €, erwartet 90.000,00 €", stmt.Revenue)
	}
	if got := lineAmount(t, stmt, "guv.3"); got != 0 {
		t.Errorf("unter den aktivierten Eigenleistungen stehen %s €, erwartet 0,00 €", got)
	}
}

// Die Abschreibung ist Aufwand der Nummer 7, nicht des Personalaufwands. Der
// Unterschied entscheidet über das Taxonomie-Element der E-Bilanz.
func TestDepreciationIsNotStaffExpense(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"6200": 1_000_000, // Abschreibungen auf immaterielle Vermögensgegenstände
		"0135": -1_000_000,
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Gewinn- und Verlustrechnung: %v", err)
	}
	if got := lineAmount(t, stmt, "guv.7.a"); got != -1_000_000 {
		t.Errorf("die Abschreibungen stehen auf %s €, erwartet -10.000,00 €", got)
	}
	if got := lineAmount(t, stmt, "guv.6"); got != 0 {
		t.Errorf("unter dem Personalaufwand stehen %s €, obwohl nur abgeschrieben wurde", got)
	}
}

// Die nicht eingeforderten ausstehenden Einlagen und die erworbenen eigenen
// Anteile stehen von Gesetzes wegen mit Sollsaldo auf einer Passivposition
// (§ 272 Abs. 1 Satz 3, Abs. 1a HGB). Das ist der Normalfall und kein Befund.
func TestOpenDeductionsFromSubscribedCapitalAreNoFinding(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"2900": -2_500_000, // Gezeichnetes Kapital
		"2910": 500_000,    // davon nicht eingefordert, offen abgesetzt
		"2909": 100_000,    // erworbene eigene Anteile
		"1800": 1_900_000,  // Bank
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if len(stmt.Assignment.WrongSign) != 0 {
		t.Errorf("der Zuordnungsbericht meldet %d Befunde zum Vorzeichen, erwartet keinen: %+v",
			len(stmt.Assignment.WrongSign), stmt.Assignment.WrongSign)
	}
	if stmt.Assignment.HasFindings() {
		t.Error("der Zuordnungsbericht meldet Befunde, obwohl die offene Absetzung dem Gesetz entspricht")
	}
	if got := lineAmount(t, stmt, "passiva.A.I"); got != 1_900_000 {
		t.Errorf("das gezeichnete Kapital nach offener Absetzung beträgt %s €, erwartet 19.000,00 €", got)
	}
}

// Ein Erfolgskonto ohne Gliederungsposition gehört in die Staffel, nicht auf die
// Aktivseite: sonst wiche Nummer 17 von dem Betrag ab, der als Posten A.V in
// der Bilanz steht.
func TestUnassignedIncomeAccountStaysInTheIncomeStatement(t *testing.T) {
	accounts := accountsWith(t, map[string]domain.Cents{"1800": -1_000_000})
	accounts = append(accounts, domain.Account{
		Number: "5678", Name: "Aufwand ohne Position", Type: domain.AccountTypeExpense,
		StatementType: "GuV", PositionID: "guv.gibt_es_nicht", DebitSum: 1_000_000,
	})

	stmt, err := BuildStatement(accounts, nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}
	if got := lineAmount(t, stmt, "guv.X"); got != -1_000_000 {
		t.Errorf("in der Auffangzeile der Staffel stehen %s €, erwartet -10.000,00 €", got)
	}
	if got := lineAmount(t, stmt, "aktiva.X"); got != 0 {
		t.Errorf("auf der Aktivseite stehen %s € unter „Nicht zugeordnet\", obwohl es ein Erfolgskonto ist", got)
	}
	if stmt.NetIncome != -1_000_000 {
		t.Errorf("das Jahresergebnis beträgt %s €, erwartet -10.000,00 €", stmt.NetIncome)
	}
	if got := lineAmount(t, stmt, "passiva.A.V"); got != -1_000_000 {
		t.Errorf("auf A.V stehen %s €, erwartet dasselbe Jahresergebnis von -10.000,00 €", got)
	}
	if len(stmt.Assignment.Unassigned) != 1 || stmt.Assignment.Unassigned[0].Number != "5678" {
		t.Errorf("der Zuordnungsbericht meldet das Konto 5678 nicht: %+v", stmt.Assignment.Unassigned)
	}
}

// § 265 Abs. 8 HGB: ein Posten, der in beiden Jahren keinen Betrag trägt, darf
// entfallen. Die Entscheidung fällt beim Aufstellen und nicht in der Ansicht —
// nur so zeigen Bildschirm, PDF und CSV dieselben Zeilen.
func TestEmptyPositionsAreMarkedAsOmitted(t *testing.T) {
	current := accountsWith(t, map[string]domain.Cents{
		"1800": 4_000_000,  // Bank
		"2900": -4_000_000, // Gezeichnetes Kapital
	})
	prior := accountsWith(t, map[string]domain.Cents{
		"0440": 1_000_000,  // im Vorjahr eine Maschine
		"1800": -1_000_000, // im Vorjahr im Haben: Verbindlichkeit gegenüber Kreditinstituten
	})

	stmt, err := BuildStatement(current, prior, domain.DepthFull)
	if err != nil {
		t.Fatalf("Bilanz: %v", err)
	}

	omitted := func(key string) bool {
		line := stmt.Line(key)
		if line == nil {
			t.Fatalf("die Gliederung kennt die Position %s nicht", key)
		}
		return line.Omitted
	}

	if omitted("aktiva.B.IV") {
		t.Error("die flüssigen Mittel tragen einen Betrag und dürfen nicht entfallen")
	}
	// Nur das Vorjahr trägt hier etwas. Der Posten bleibt: die Vorjahresspalte
	// ist Teil des Abschlusses, und ohne die Zeile stünde ihre Zahl nirgends.
	if omitted("passiva.C.2") {
		t.Error("ein Posten mit Vorjahresbetrag darf nicht entfallen")
	}
	if !omitted("aktiva.A.I.1") {
		t.Error("ein Posten ohne Betrag in beiden Jahren muss als entfallen gekennzeichnet sein")
	}
	// Die Zwischensummen der Staffel bleiben stehen, auch wenn das Ergebnis
	// null ist: § 275 Abs. 2 HGB nennt sie, und ein Abschluss ohne die Zeile
	// „Jahresüberschuss/Jahresfehlbetrag" wäre unvollständig.
	for _, key := range []string{"guv.15", "guv.17"} {
		if omitted(key) {
			t.Errorf("die Zwischensumme %s (%s) darf nicht entfallen", key, LineLabel(key))
		}
	}
}

// Eine Obergruppe, deren Unterposten sich gegenseitig aufheben, trägt in der
// Summe null — verschwinden darf sie trotzdem nicht, sonst wären die Beträge
// darunter nicht mehr erreichbar.
func TestGroupWithOffsettingChildrenStays(t *testing.T) {
	stmt, err := BuildStatement(accountsWith(t, map[string]domain.Cents{
		"6220": 1_000_000,  // Abschreibungen auf Sachanlagen (Nummer 7 a)
		"6270": -1_000_000, // Zuschreibung im Umlaufvermögen (Nummer 7 b)
		"1800": 1_000_000,  // Bank
		"3150": -1_000_000, // Verbindlichkeiten gegenüber Kreditinstituten
	}), nil, domain.DepthFull)
	if err != nil {
		t.Fatalf("Gewinn- und Verlustrechnung: %v", err)
	}

	group := stmt.Line("guv.7")
	if group == nil {
		t.Fatal("die Gliederung kennt die Position guv.7 nicht")
	}
	if group.Amount != 0 {
		t.Fatalf("die Abschreibungen stehen auf %s €, erwartet 0,00 € — die Unterposten heben sich auf",
			group.Amount)
	}
	if group.Omitted {
		t.Error("die Obergruppe darf nicht entfallen, solange unter ihr Beträge stehen")
	}
	for _, key := range []string{"guv.7.a", "guv.7.b"} {
		if stmt.Line(key).Omitted {
			t.Errorf("der Unterposten %s trägt einen Betrag und darf nicht entfallen", key)
		}
	}
}
