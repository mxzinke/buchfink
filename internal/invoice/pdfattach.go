package invoice

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// maxAttachmentBytes bounds what is pulled out of a PDF. An e-invoice XML is
// kilobytes; anything approaching this is not one.
const maxAttachmentBytes = 32 << 20

// facturXNames are the file names the standards prescribe for the embedded
// invoice data. Preferring them over "the first XML attachment" matters: a
// hybrid invoice may legitimately carry other attachments, such as a timesheet.
var facturXNames = []string{
	"factur-x.xml",        // Factur-X / ZUGFeRD 2.x
	"zugferd-invoice.xml", // ZUGFeRD 1.0
	"xrechnung.xml",
	"order-x.xml",
}

// EmbeddedFile is one attachment pulled out of a PDF.
type EmbeddedFile struct {
	Name string
	Data []byte
}

// ExtractEmbeddedInvoice returns the structured invoice data embedded in a hybrid
// PDF.
//
// Reading the PDF structure properly rather than scanning for XML is not
// pedantry: real invoices arrive from every ERP on the market, and most of them
// write object streams and cross-reference streams that a naive scan cannot
// follow. The input tax deduction hangs on this file, so it has to be the right
// one — found through /Names /EmbeddedFiles, not guessed.
func ExtractEmbeddedInvoice(pdf []byte) (*EmbeddedFile, error) {
	files, err := extractAttachments(pdf)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("das PDF enthält keinen eingebetteten Rechnungsdatensatz — es ist eine sonstige Rechnung, keine E-Rechnung")
	}

	// Erst der genormte Name, dann irgendein XML.
	for _, want := range facturXNames {
		for i := range files {
			if strings.EqualFold(files[i].Name, want) {
				return &files[i], nil
			}
		}
	}
	for i := range files {
		if strings.HasSuffix(strings.ToLower(files[i].Name), ".xml") || looksLikeXML(files[i].Data) {
			return &files[i], nil
		}
	}
	return nil, fmt.Errorf("das PDF enthält Anhänge, aber keinen Rechnungsdatensatz")
}

func extractAttachments(pdf []byte) ([]EmbeddedFile, error) {
	conf := model.NewDefaultConfiguration()
	// Real-world invoices are frequently not strictly conformant. Refusing to
	// read one over a formal defect would block the very documents the law makes
	// mandatory to receive.
	conf.ValidationMode = model.ValidationRelaxed

	attachments, err := api.ExtractAttachmentsRaw(bytes.NewReader(pdf), "", nil, conf)
	if err != nil {
		// "no attachments available" ist kein Lesefehler, sondern die Antwort:
		// das PDF hat keine. Ein gewöhnliches Rechnungs-PDF ist der häufigste
		// Fall überhaupt und darf nicht wie ein kaputtes Dokument aussehen.
		if strings.Contains(err.Error(), "no attachments") {
			return nil, nil
		}
		return nil, fmt.Errorf("das PDF konnte nicht gelesen werden: %w", err)
	}

	out := make([]EmbeddedFile, 0, len(attachments))
	for _, a := range attachments {
		if a.Reader == nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(a.Reader, maxAttachmentBytes+1))
		if err != nil {
			return nil, fmt.Errorf("der Anhang %s konnte nicht gelesen werden: %w", a.ID, err)
		}
		if len(data) > maxAttachmentBytes {
			return nil, fmt.Errorf("der Anhang %s ist zu groß für einen Rechnungsdatensatz", a.ID)
		}
		out = append(out, EmbeddedFile{Name: a.ID, Data: data})
	}
	return out, nil
}

func looksLikeXML(data []byte) bool {
	// Ein UTF-8-BOM am Anfang ist verbreitet und kein Fehler.
	trimmed := bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	trimmed = bytes.TrimLeft(trimmed, " \t\r\n")
	return bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<"))
}

// IsPDF reports whether the bytes start with a PDF header.
func IsPDF(data []byte) bool { return bytes.HasPrefix(data, []byte("%PDF-")) }
