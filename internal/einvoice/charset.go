package einvoice

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
)

// charsetReader lets the XML decoder read documents that are not UTF-8.
//
// The standard library refuses anything but UTF-8 outright, and refusing is the
// wrong answer here: an invoice declared as windows-1252 is a perfectly valid
// invoice that a supplier's system produced, and the recipient still has to
// book it. The declared encoding is honoured; only an encoding nobody can name
// is an error.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	name := strings.ToLower(strings.TrimSpace(charset))
	switch name {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return input, nil
	case "iso-8859-1", "latin1", "latin-1", "iso8859-1", "iso_8859-1":
		// Windows-1252 is a superset of Latin-1 and what documents labelled
		// Latin-1 almost always actually contain — the difference is the range
		// where the Euro sign lives, which on an invoice matters.
		return charmap.Windows1252.NewDecoder().Reader(input), nil
	}
	encoding, err := htmlindex.Get(name)
	if err != nil {
		return nil, fmt.Errorf("die Zeichenkodierung %q ist unbekannt", charset)
	}
	return encoding.NewDecoder().Reader(input), nil
}
