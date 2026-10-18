package ebilanz

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
)

// SuSaCSV ist die Summen- und Saldenliste zur E-Bilanz für Programme, die
// statt der XBRL-Instanz eine Saldenliste einlesen (eBilanz+: Import
// „Summen- und Saldenliste“ mit Taxonomie-Layout). Je Konto mit Saldo eine
// Zeile Konto;Bezeichnung;Saldo;Taxonomie: der Saldo mit Vorzeichen (Soll
// positiv, Haben negativ) und die Position der Taxonomie, unter der das Konto
// im Kontennachweis der Instanz steht. Es sind dieselben Konten und Salden wie
// in der Instanz; blockieren Befunde die Instanz, gibt es auch die Liste
// nicht. UTF-8 mit BOM, Semikolon, Dezimalkomma, CRLF.
func SuSaCSV(res *Result) ([]byte, error) {
	if res == nil || res.Report == nil {
		return nil, fmt.Errorf("die E-Bilanz fehlt")
	}
	if err := res.Report.BlockingError(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	w.UseCRLF = true
	if err := w.Write([]string{"Konto", "Bezeichnung", "Saldo", "Taxonomie"}); err != nil {
		return nil, err
	}
	for _, row := range res.Report.Rows {
		if row.Balance == 0 || row.Element == "" {
			continue
		}
		if err := w.Write([]string{row.Account, row.Name, decimalComma(row.Balance), row.Element}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decimalComma schreibt Cent als Betrag mit zwei Nachkommastellen und
// Dezimalkomma, ohne Tausendertrennzeichen.
func decimalComma(c domain.Cents) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d,%02d", sign, c/100, c%100)
}
