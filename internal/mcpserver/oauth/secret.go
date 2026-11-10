package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

// Präfixe machen ein Token in Logs und Dateien erkennbar, ohne etwas über
// seinen Inhalt zu verraten.
const (
	prefixAccess  = "bfa_"
	prefixRefresh = "bfr_"
	prefixCode    = "bfc_"
)

// newSecret liefert 256 zufällige Bit, base64url, mit Präfix.
func newSecret(prefix string) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("Zufallsquelle nicht verfügbar: %v", err))
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b)
}

// hashSecret ist die gespeicherte Form eines Codes oder Tokens.
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// equalSecret vergleicht in konstanter Zeit.
func equalSecret(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// pkceS256 ist BASE64URL(SHA256(verifier)) nach RFC 7636 §4.2.
func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// newUserCode ist der Abgleichcode, den Browser und App gleichzeitig zeigen.
func newUserCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		panic(fmt.Sprintf("Zufallsquelle nicht verfügbar: %v", err))
	}
	s := fmt.Sprintf("%06d", n.Int64())
	return s[:3] + "-" + s[3:]
}

const (
	maxNameRunes = 80
	maxSlugLen   = 32
)

// sanitizeName bereinigt den Namen, den eine App bei der Registrierung
// angibt: keine Steuerzeichen, höchstens 80 Zeichen. Er ist eine Angabe der
// App und wird in der App auch so gekennzeichnet.
func sanitizeName(name string) string {
	var b strings.Builder
	n := 0
	for _, r := range name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if n == maxNameRunes {
			break
		}
		b.WriteRune(r)
		n++
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if out == "" {
		return "Unbenannte App"
	}
	return out
}

// slugify bildet den Kurznamen im Erfassungsweg: a–z, 0–9 und Bindestrich.
func slugify(name string) string {
	replacer := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "Ä", "ae", "Ö", "oe", "Ü", "ue")
	name = strings.ToLower(replacer.Replace(name))
	var b strings.Builder
	dash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
		if b.Len() >= maxSlugLen {
			break
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > maxSlugLen {
		slug = strings.TrimRight(slug[:maxSlugLen], "-")
	}
	if slug == "" {
		return "agent"
	}
	return slug
}
