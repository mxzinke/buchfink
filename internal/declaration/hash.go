package declaration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// FiguresHash ist der SHA-256 über die kanonische Liste aller Zeilen
// („schluessel=wert“ je Zeile, in der Reihenfolge des Katalogs) mit Art, Jahr
// und Vordruckjahr davor. Er ändert sich mit jedem Wert, nicht mit Hinweisen,
// Befunden oder dem Stand.
func FiguresHash(d *Declaration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "schema=%s\nart=%s\njahr=%d\nvordruckjahr=%d\n", Schema, d.Kind, d.Period.Year, d.FormYear)
	for _, f := range d.Forms {
		for _, s := range f.Sections {
			for _, l := range s.Lines {
				fmt.Fprintf(&b, "%s=%s\n", l.Key, l.Value.Canonical())
			}
		}
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
