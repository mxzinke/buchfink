// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Cents is a monetary amount in the smallest currency unit (Eurocent).
//
// Buchhaltung rechnet exakt. Fließkomma-Beträge driften bei Summierung, wodurch
// die Grundinvariante der doppelten Buchführung – Summe Soll gleich Summe Haben –
// nur noch mit einer Toleranz prüfbar wäre. Alle Beträge im System sind deshalb
// ganzzahlige Cent; gerundet wird ausschließlich an den definierten Stellen in
// diesem Paket.
type Cents int64

// ErrInvalidAmount is returned when a string cannot be parsed as a monetary amount.
var ErrInvalidAmount = errors.New("ungültiger Betrag")

// Euros returns the amount as a float. Only for display and for interfaces that
// cannot carry integers (JSON to the frontend, XBRL, ZUGFeRD). Never use the
// result for further arithmetic.
func (c Cents) Euros() float64 {
	return float64(c) / 100.0
}

// Abs returns the absolute amount.
func (c Cents) Abs() Cents {
	if c < 0 {
		return -c
	}
	return c
}

// String formats the amount in German notation without currency symbol,
// e.g. "1.234,56" or "-42,00".
func (c Cents) String() string {
	neg := c < 0
	v := c.Abs()

	whole := int64(v) / 100
	frac := int64(v) % 100

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString(groupThousands(whole))
	b.WriteByte(',')
	b.WriteString(fmt.Sprintf("%02d", frac))
	return b.String()
}

// Decimal formats the amount for machine-readable output (XBRL, ZUGFeRD, CSV):
// a plain decimal with a dot and exactly two places, e.g. "-1234.56". Derived
// from the integer directly, so the exported figure is bit-for-bit the booked
// one rather than a float rendering of it.
func (c Cents) Decimal() string {
	neg := c < 0
	v := int64(c.Abs())
	s := fmt.Sprintf("%d.%02d", v/100, v%100)
	if neg {
		return "-" + s
	}
	return s
}

func groupThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, digit := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, digit)
	}
	return string(out)
}

// CentsFromEuros converts a float amount to Cents using commercial rounding.
// Reserved for external inputs (bank files, ZUGFeRD XML, exchange rates) where
// the source itself is a float. Internal arithmetic never goes through here.
func CentsFromEuros(f float64) Cents {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return Cents(math.Round(f * 100))
}

// ParseCents reads a German or plain decimal amount ("1.234,56", "1234.56",
// "-42") into Cents. It rejects anything with more than two decimal places
// rather than silently truncating a value the user typed.
func ParseCents(s string) (Cents, error) {
	raw := strings.TrimSpace(s)
	raw = strings.NewReplacer("€", "", " ", "", " ", "").Replace(raw)
	if raw == "" {
		return 0, ErrInvalidAmount
	}

	neg := false
	switch raw[0] {
	case '-':
		neg = true
		raw = raw[1:]
	case '+':
		raw = raw[1:]
	}

	// German notation uses "." as thousands separator and "," as decimal mark.
	// Plain notation uses "." as decimal mark and no grouping.
	if strings.Contains(raw, ",") {
		raw = strings.ReplaceAll(raw, ".", "")
		raw = strings.Replace(raw, ",", ".", 1)
	}
	if strings.Count(raw, ".") > 1 {
		return 0, ErrInvalidAmount
	}

	whole, frac, hasFrac := strings.Cut(raw, ".")
	if whole == "" {
		whole = "0"
	}
	if len(frac) > 2 {
		return 0, fmt.Errorf("%w: höchstens zwei Nachkommastellen erlaubt", ErrInvalidAmount)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	if !hasFrac {
		frac = "00"
	}

	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}

	total := Cents(w*100 + f)
	if neg {
		total = -total
	}
	return total, nil
}

// SumCents adds up amounts. Exact by construction — no tolerance needed.
func SumCents(values ...Cents) Cents {
	var total Cents
	for _, v := range values {
		total += v
	}
	return total
}

// MulRound multiplies an amount by numerator/denominator and rounds the result
// commercially (kaufmännisches Runden, halbe Cent aufwärts vom Betrag weg).
// This is the single rounding point for tax and share calculations.
func MulRound(base Cents, numerator, denominator int64) Cents {
	if denominator == 0 {
		return 0
	}
	neg := false
	n := int64(base) * numerator
	d := denominator
	if (n < 0) != (d < 0) {
		neg = true
	}
	if n < 0 {
		n = -n
	}
	if d < 0 {
		d = -d
	}
	// Round half away from zero.
	res := (n + d/2) / d
	if neg {
		res = -res
	}
	return Cents(res)
}
