package accounting

import (
	"strconv"
	"strings"
)

// canonicalWriter renders values into an unambiguous byte sequence for hashing.
//
// Each field is written as name, byte length and value. The explicit length makes
// the encoding injection-proof: no value can be crafted to look like a field
// boundary, which a plain delimiter-joined string would allow. A file called
// "invoice\nrole:8:original" cannot impersonate a second file.
//
// Both the journal chain and the Beleg-Hash use this one implementation. Two
// copies of a canonicalisation are two chances to change one and not the other.
type canonicalWriter struct {
	b strings.Builder
}

func (w *canonicalWriter) put(name, value string) {
	w.b.WriteString(name)
	w.b.WriteByte(':')
	w.b.WriteString(strconv.Itoa(len(value)))
	w.b.WriteByte(':')
	w.b.WriteString(value)
	w.b.WriteByte('\n')
}

func (w *canonicalWriter) putInt(name string, value int64) {
	w.put(name, strconv.FormatInt(value, 10))
}

func (w *canonicalWriter) bytes() []byte { return []byte(w.b.String()) }
