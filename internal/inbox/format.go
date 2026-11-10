package inbox

import (
	"bytes"
	"path/filepath"
	"strings"
)

// MaxFileBytes ist die größte Datei, die Buchfink aus dem Eingang liest.
const MaxFileBytes = 50 << 20

// Format ist das Dateiformat nach Endung.
type Format string

const (
	FormatPDF   Format = "pdf"
	FormatImage Format = "bild"
	FormatXML   Format = "xml"
)

var extensions = map[string]Format{
	".pdf": FormatPDF, ".png": FormatImage, ".jpg": FormatImage, ".jpeg": FormatImage,
	".tif": FormatImage, ".tiff": FormatImage, ".webp": FormatImage, ".xml": FormatXML,
}

// FormatOf ist das Format einer Datei nach ihrer Endung; ok ist false für
// Endungen, die kein Beleg hat.
func FormatOf(name string) (Format, bool) {
	f, ok := extensions[strings.ToLower(filepath.Ext(name))]
	return f, ok
}

// ContentMatches prüft die ersten Bytes gegen das Format der Endung.
func ContentMatches(format Format, data []byte) bool {
	switch format {
	case FormatPDF:
		return bytes.HasPrefix(data, []byte("%PDF-"))
	case FormatImage:
		return bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) ||
			bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) ||
			bytes.HasPrefix(data, []byte("II*\x00")) || bytes.HasPrefix(data, []byte("MM\x00*")) ||
			(len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP")
	case FormatXML:
		trimmed := bytes.TrimLeft(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), " \t\r\n")
		return bytes.HasPrefix(trimmed, []byte("<"))
	}
	return false
}
