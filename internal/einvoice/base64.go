package einvoice

import (
	"encoding/base64"
	"strings"
)

// decodeBase64 reads an embedded attachment.
//
// Whitespace is stripped first: both syntaxes allow the base64 payload to be
// wrapped across lines, and a decoder that chokes on a newline would drop the
// attachment of every invoice produced by a formatter. A payload that does not
// decode is returned as nothing rather than as garbage — a half-read PDF in the
// Belegablage is worse than a missing one, because it looks like a file.
func decodeBase64(raw string) []byte {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, raw)
	if cleaned == "" {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return nil
	}
	return decoded
}
