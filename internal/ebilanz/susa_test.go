package ebilanz

import (
	"bytes"
	"encoding/csv"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

var (
	tupleRe   = regexp.MustCompile(`(?s)<de-gaap-ci:detailedInformation\.accountBalances>(.*?)</de-gaap-ci:detailedInformation\.accountBalances>`)
	tupleItem = func(name string) *regexp.Regexp {
		return regexp.MustCompile(`accountbalances\.` + name + `[^>]*>([^<]*)<`)
	}
	positionRe = tupleItem("positionName")
	numberRe   = tupleItem("accountNumber")
	amountRe   = tupleItem("amount")
	saldoRe    = regexp.MustCompile(`^-?[0-9]+,[0-9]{2}$`)
)

// cents liest „-1234,56“.
func cents(t *testing.T, s string) domain.Cents {
	t.Helper()
	v, err := strconv.ParseInt(strings.Replace(s, ",", "", 1), 10, 64)
	if err != nil {
		t.Fatalf("Saldo %q: %v", s, err)
	}
	return domain.Cents(v)
}

// accountProof liest den Kontennachweis der Instanz: Konto → Position und
// Betrag in Richtung der Position.
func accountProof(t *testing.T, instance []byte) map[string][2]string {
	t.Helper()
	out := map[string][2]string{}
	for _, m := range tupleRe.FindAllStringSubmatch(string(instance), -1) {
		number := numberRe.FindStringSubmatch(m[1])
		if number == nil || number[1] == "" {
			continue // Position ohne Konto (Leitfaden 6.10 B.12.1)
		}
		out[number[1]] = [2]string{strings.TrimPrefix(positionRe.FindStringSubmatch(m[1])[1], "de-gaap-ci:"),
			amountRe.FindStringSubmatch(m[1])[1]}
	}
	return out
}

// Die Saldenliste trägt dieselben Konten und Salden wie die Instanz, in
// Summe null, mit gültigen Positionen und im Format für eBilanz+.
func TestSuSaCSVMatchesTheInstance(t *testing.T) {
	for _, tc := range []struct {
		name     string
		balances map[string]domain.Cents
		year     string
	}{
		{"Einheitsbilanz 6.9", exampleBalances, "2025"},
		{"Ergebnisverwendung", withChanges(exampleBalances, map[string]domain.Cents{"2970": -400000, "7790": 500000, "1800": -100000}), "2025"},
		{"Fassung 6.10", exampleBalances, "2027"},
	} {
		in := exampleInput(t, tc.balances, tc.year)
		// Ein Konto ohne Saldo entfällt.
		catalog := map[string]domain.Account{}
		for _, acc := range accounting.DefaultSKR04Accounts() {
			catalog[acc.Number] = acc
		}
		in.Accounts = append(in.Accounts, withBalance(catalog["6815"], 0))
		if tc.name == "Ergebnisverwendung" {
			prior := domain.Cents(2900000)
			paid := domain.Cents(2500000)
			in.PriorEquity, in.PriorPaidIn = &prior, &paid
		}
		res := generateOK(t, in)
		data, err := SuSaCSV(res)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		if !bytes.HasPrefix(data, []byte("\xEF\xBB\xBF")) {
			t.Errorf("%s: ohne BOM", tc.name)
		}
		lines := strings.Split(strings.TrimSuffix(string(data[3:]), "\r\n"), "\r\n")
		if lines[0] != "Konto;Bezeichnung;Saldo;Taxonomie" {
			t.Errorf("%s: Kopfzeile %q", tc.name, lines[0])
		}
		for _, line := range lines {
			if strings.Contains(line, "\n") {
				t.Errorf("%s: Zeile ohne CRLF: %q", tc.name, line)
			}
		}
		r := csv.NewReader(bytes.NewReader(data[3:]))
		r.Comma = ';'
		records, err := r.ReadAll()
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		tax, err := LoadTaxonomy(res.Version.Version)
		if err != nil {
			t.Fatal(err)
		}
		rows := map[string][]string{}
		var sum domain.Cents
		for _, rec := range records[1:] {
			if len(rec) != 4 || !saldoRe.MatchString(rec[2]) {
				t.Errorf("%s: Zeile %q", tc.name, rec)
				continue
			}
			if _, ok := tax.Element(rec[3]); !ok {
				t.Errorf("%s: %s steht unter %q, das die Taxonomie %s nicht kennt", tc.name, rec[0], rec[3], tax.Version)
			}
			if cents(t, rec[2]) == 0 {
				t.Errorf("%s: Konto %s ohne Saldo", tc.name, rec[0])
			}
			rows[rec[0]] = rec
			sum += cents(t, rec[2])
		}
		if sum != 0 {
			t.Errorf("%s: Summe der Salden %d Cent", tc.name, sum)
		}
		if _, ok := rows["6815"]; ok {
			t.Errorf("%s: Konto ohne Saldo in der Liste", tc.name)
		}
		if len(rows) != len(tc.balances) {
			t.Errorf("%s: %d Konten, erwartet %d", tc.name, len(rows), len(tc.balances))
		}

		proof := accountProof(t, res.Instance)
		if len(proof) != len(rows) {
			t.Errorf("%s: Kontennachweis %d Konten, Liste %d", tc.name, len(proof), len(rows))
		}
		for account, p := range proof {
			row, ok := rows[account]
			if !ok {
				t.Errorf("%s: Konto %s der Instanz fehlt", tc.name, account)
				continue
			}
			if row[3] != p[0] {
				t.Errorf("%s: Konto %s unter %s, in der Instanz %s", tc.name, account, row[3], p[0])
			}
			// Die Instanz führt den Saldo in Richtung der Position.
			want := cents(t, row[2])
			if el, _ := tax.Element(p[0]); el.Balance == "credit" {
				want = -want
			}
			if got := cents(t, strings.Replace(p[1], ".", ",", 1)); got != want {
				t.Errorf("%s: Konto %s mit %d in der Instanz, %d in der Liste", tc.name, account, got, want)
			}
		}
	}
}

// Bezeichnungen mit Trennzeichen kommen in Anführungszeichen; blockierende
// Befunde verhindern die Liste wie die Instanz.
func TestSuSaCSVQuotesAndBlocks(t *testing.T) {
	res := &Result{Report: &MappingReport{CanExport: true, Rows: []MappingRow{
		{Account: "1800", Name: "Bank; Hausbank \"Nord\"", Balance: 123456, Element: "bs.ass.currAss.cashEquiv.bank"},
		{Account: "2900", Name: "Gezeichnetes Kapital", Balance: -123456, Element: "bs.eqLiab.equity.subscribed.corp"},
		{Account: "6815", Name: "Bürobedarf", Balance: 0, Element: "x"},
	}}}
	data, err := SuSaCSV(res)
	if err != nil {
		t.Fatal(err)
	}
	want := "\xEF\xBB\xBFKonto;Bezeichnung;Saldo;Taxonomie\r\n" +
		"1800;\"Bank; Hausbank \"\"Nord\"\"\";1234,56;bs.ass.currAss.cashEquiv.bank\r\n" +
		"2900;Gezeichnetes Kapital;-1234,56;bs.eqLiab.equity.subscribed.corp\r\n"
	if string(data) != want {
		t.Errorf("Liste:\n%q\nerwartet\n%q", data, want)
	}
	if decimalComma(-5) != "-0,05" || decimalComma(100) != "1,00" {
		t.Errorf("Beträge: %s %s", decimalComma(-5), decimalComma(100))
	}

	blocked := &Result{Report: &MappingReport{Blocking: []MappingRow{{Account: "9000", Finding: "Statistisches Konto"}}}}
	if _, err := SuSaCSV(blocked); err == nil {
		t.Error("Liste trotz blockierender Befunde")
	}
}

// Bei einer Handelsbilanz mit Überleitung enthält die Saldenliste die
// handelsrechtlichen Salden; der Bericht nennt die Abweichungen samt Position,
// und SuSaNote sagt, dass sie im Dienst nachzutragen sind. Eine Einheitsbilanz
// braucht keinen Hinweis.
func TestSuSaCSVNamesTheTransfer(t *testing.T) {
	plain := generateOK(t, exampleInput(t, exampleBalances, "2025"))
	if note := SuSaNote(plain.Report); note != "" || len(plain.Report.Transfers) != 0 {
		t.Errorf("Einheitsbilanz: %q %+v", note, plain.Report.Transfers)
	}
	in := exampleInput(t, exampleBalances, "2025")
	in.Adjustments = []Adjustment{{Reason: "Sonderabschreibung § 7g Abs. 5 EStG", Account: "0520", IncomeAccount: "6242",
		Current: -600000}}
	res := generateOK(t, in)
	if len(res.Report.Transfers) != 1 {
		t.Fatalf("Überleitung im Bericht: %+v", res.Report.Transfers)
	}
	tr := res.Report.Transfers[0]
	if tr.Account != "0520" || tr.Current != -600000 || tr.Element != "bs.ass.fixAss.tan.otherEquipm.passengerCars" {
		t.Errorf("Abweichung: %+v", tr)
	}
	if note := SuSaNote(res.Report); !strings.Contains(note, "Handelsbilanz") || !strings.Contains(note, "Überleitung") {
		t.Errorf("Hinweis: %q", note)
	}
	data, err := SuSaCSV(res)
	if err != nil {
		t.Fatal(err)
	}
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))))
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows[1:] {
		found = found || row[0] == "0520"
		if row[0] == "0520" && cents(t, row[2]) != exampleBalances["0520"] {
			t.Errorf("0520 mit %s statt dem handelsrechtlichen Saldo", row[2])
		}
	}
	if !found {
		t.Error("0520 fehlt in der Saldenliste")
	}
}
