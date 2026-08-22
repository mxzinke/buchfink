package xrechnung

import "strings"

// ValidIBAN reports whether a string is a well-formed IBAN.
//
// XRechnung requires one wherever SEPA is the payment means (BR-DE-19, -20),
// and the check is worth doing properly rather than by length: the checksum
// catches a transposed pair of digits, which is the mistake that actually
// happens and the one that sends a payment to a stranger.
//
// The method is ISO 7064 mod-97-10: move the first four characters to the end,
// replace every letter by its position in the alphabet plus nine, and read the
// result as one number — a valid IBAN leaves remainder 1.
func ValidIBAN(s string) bool {
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == ' ' {
			return -1
		}
		return r
	}, strings.ToUpper(strings.TrimSpace(s)))

	// Two letters for the country, two check digits, then at least one more —
	// and no IBAN anywhere is longer than 34 characters.
	if len(compact) < 5 || len(compact) > 34 {
		return false
	}
	if !isLetter(compact[0]) || !isLetter(compact[1]) || !isDigit(compact[2]) || !isDigit(compact[3]) {
		return false
	}

	rearranged := compact[4:] + compact[:4]
	remainder := 0
	for i := 0; i < len(rearranged); i++ {
		c := rearranged[i]
		switch {
		case isDigit(c):
			remainder = (remainder*10 + int(c-'0')) % 97
		case isLetter(c):
			// Der Buchstabenwert ist zweistellig, also zwei Schritte.
			value := int(c-'A') + 10
			remainder = (remainder*100 + value) % 97
		default:
			return false
		}
	}
	return remainder == 1
}

func isDigit(c byte) bool  { return c >= '0' && c <= '9' }
func isLetter(c byte) bool { return c >= 'A' && c <= 'Z' }
