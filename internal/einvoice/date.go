package einvoice

import (
	"fmt"
	"strings"
	"time"
)

// Date is a calendar date from an invoice.
//
// EN 16931 knows only whole days — there is no time and no zone anywhere in the
// model. The two syntaxes write them differently (CII uses YYYYMMDD with a
// format code, UBL uses ISO YYYY-MM-DD), so the model normalises to ISO and
// keeps what was written alongside it. Keeping the original matters for a
// document that states something unparseable: reporting "das Datum %q ist
// unlesbar" needs the string the supplier actually sent, not an empty value.
type Date struct {
	iso string // YYYY-MM-DD, leer wenn unlesbar oder nicht angegeben
	raw string // wie im Dokument geschrieben
}

// NewDate takes an ISO date, as UBL writes it.
func NewDate(raw string) Date {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Date{}
	}
	if _, err := time.Parse("2006-01-02", raw); err != nil {
		return Date{raw: raw}
	}
	return Date{iso: raw, raw: raw}
}

// NewDateFromFormat takes a date in one of the UN/CEFACT date formats, as CII
// writes it. EN 16931 permits only format 102 (YYYYMMDD); anything else is kept
// as written so the reader can say what arrived.
func NewDateFromFormat(raw, format string) Date {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Date{}
	}
	if strings.TrimSpace(format) != "102" {
		return Date{raw: raw}
	}
	t, err := time.Parse("20060102", raw)
	if err != nil {
		return Date{raw: raw}
	}
	return Date{iso: t.Format("2006-01-02"), raw: raw}
}

// Present reports whether the document stated a date at all.
func (d Date) Present() bool { return d.raw != "" }

// Valid reports whether the stated date could be read as a calendar date.
func (d Date) Valid() bool { return d.iso != "" }

// ISO returns the date as YYYY-MM-DD, or the empty string if it is absent or
// unreadable.
func (d Date) ISO() string { return d.iso }

// Raw returns the date as the document wrote it.
func (d Date) Raw() string { return d.raw }

// String returns the ISO form where there is one, otherwise what was written.
func (d Date) String() string {
	if d.iso != "" {
		return d.iso
	}
	return d.raw
}

// CII returns the date in UN/CEFACT format 102, for writing a CII document.
func (d Date) CII() string {
	if d.iso == "" {
		return ""
	}
	return strings.ReplaceAll(d.iso, "-", "")
}

// Before reports whether d lies strictly before other. Both have to be readable;
// if either is not, the result is false and no rule fires on it — an unreadable
// date is reported by its own rule, not by an ordering one.
func (d Date) Before(other Date) bool {
	if d.iso == "" || other.iso == "" {
		return false
	}
	return d.iso < other.iso
}

// After reports whether d lies strictly after other.
func (d Date) After(other Date) bool { return other.Before(d) }

// DateFromTime builds a Date from a Go time value.
func DateFromTime(t time.Time) Date {
	iso := t.Format("2006-01-02")
	return Date{iso: iso, raw: iso}
}

// DateFromGerman parses a date written as DD.MM.YYYY, the form Buchfink stores
// and displays.
func DateFromGerman(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Date{}, nil
	}
	t, err := time.Parse("02.01.2006", s)
	if err != nil {
		return Date{}, fmt.Errorf("Datum %q ist nicht im Format TT.MM.JJJJ", s)
	}
	return DateFromTime(t), nil
}

// German renders the date as DD.MM.YYYY.
func (d Date) German() string {
	if d.iso == "" {
		return d.raw
	}
	t, err := time.Parse("2006-01-02", d.iso)
	if err != nil {
		return d.raw
	}
	return t.Format("02.01.2006")
}
