package update

import (
	"strconv"
	"strings"
)

// version ist eine Fassung nach dem Muster X.Y.Z oder X.Y.Z-vorab, wie
// scripts/release.sh sie vergibt.
type version struct {
	core [3]int
	pre  string
}

// parseVersion liest „0.4.0", „v0.4.0" und „0.4.0-rc.1". Ein Arbeitsstand
// („dev", „dev+abc123") ist keine Fassung.
func parseVersion(s string) (version, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	core, pre, _ := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return version{}, false
	}
	var v version
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || part == "" {
			return version{}, false
		}
		v.core[i] = n
	}
	v.pre = pre
	return v, true
}

// newer meldet, ob v nach other kommt. Eine Vorabversion steht vor ihrer
// Fassung: 0.4.0-rc.1 < 0.4.0.
func (v version) newer(other version) bool {
	for i := range v.core {
		if v.core[i] != other.core[i] {
			return v.core[i] > other.core[i]
		}
	}
	switch {
	case v.pre == other.pre:
		return false
	case v.pre == "":
		return true
	case other.pre == "":
		return false
	}
	return comparePre(v.pre, other.pre) > 0
}

// comparePre vergleicht die Teile einer Vorabkennung: Zahlen nach ihrem Wert,
// alles andere als Text; die kürzere Kennung steht vorn.
func comparePre(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		an, aerr := strconv.Atoi(as[i])
		bn, berr := strconv.Atoi(bs[i])
		switch {
		case aerr == nil && berr == nil:
			if an != bn {
				return an - bn
			}
		case aerr == nil:
			return -1
		case berr == nil:
			return 1
		default:
			if c := strings.Compare(as[i], bs[i]); c != 0 {
				return c
			}
		}
	}
	return len(as) - len(bs)
}
