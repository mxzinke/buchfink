package einvoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// DetectSyntax reports which EN 16931 syntax a document is written in.
//
// It reads the root element rather than guessing from the file name or from a
// substring search. A ZUGFeRD file called invoice.xml and a UBL file called
// invoice.xml look the same from outside, and a supplier is free to name
// neither.
func DetectSyntax(data []byte) (Syntax, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charsetReader
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return SyntaxUnknown, fmt.Errorf("die Datei enthält kein XML-Wurzelelement")
		}
		if err != nil {
			return SyntaxUnknown, fmt.Errorf("das XML konnte nicht gelesen werden: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Space {
		case nsRSM:
			return SyntaxCII, nil
		case nsUBLInvoice, nsUBLCreditNote:
			return SyntaxUBL, nil
		}
		return SyntaxUnknown, fmt.Errorf(
			"das Wurzelelement %q gehört weder zu einer CII- noch zu einer UBL-Rechnung",
			start.Name.Local)
	}
}

// Parse reads an invoice in whichever EN 16931 syntax it is written in.
//
// This is the entry point a caller wants: what arrives in the post is a file,
// not a syntax, and which of the two it is should not be the recipient's
// problem.
func Parse(data []byte) (*Invoice, error) {
	syntax, err := DetectSyntax(data)
	if err != nil {
		return nil, err
	}
	switch syntax {
	case SyntaxCII:
		return ParseCII(data)
	case SyntaxUBL:
		return ParseUBL(data)
	default:
		return nil, fmt.Errorf("die Syntax der Datei ist unbekannt")
	}
}
