package invoice

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	typst "github.com/varunbpatil/typst-go-wasm"

	"github.com/buchfink/buchfink/internal/domain"
)

// The invoice typeface is embedded rather than looked up on the system: a
// generated invoice has to look the same on every machine, and PDF/A requires
// every glyph used to be embedded in the file anyway.
//
// Manrope, SIL Open Font License 1.1 — see fonts/LICENSE.txt.
var (
	//go:embed fonts/Manrope-Regular.ttf
	fontRegular []byte
	//go:embed fonts/Manrope-SemiBold.ttf
	fontSemiBold []byte
	//go:embed fonts/Manrope-Bold.ttf
	fontBold []byte
)

// Renderer turns an invoice into a hybrid PDF/A-3b with the ZUGFeRD XML embedded.
//
// It runs the Typst compiler as WebAssembly in this process: no external binary,
// no CGO, nothing to install alongside the app. Typst produces the PDF/A-3b and
// attaches the XML in one step, so there is no post-processing pass that would
// have to write the Factur-X XMP extension schema by hand.
type Renderer struct {
	// mu guards the start-up. Warm runs in the background while the rest of the
	// application carries on, so the compiler handle has to be published under a
	// lock rather than read straight off the struct.
	mu       sync.Mutex
	started  bool
	closed   bool
	compiler *typst.Compiler
	initErr  error
}

// NewRenderer returns a renderer. The WASM module is compiled on first use and
// then reused: that compilation takes several seconds, so the instance is meant
// to be long-lived, and start-up stays free of a cost most sessions never incur.
func NewRenderer() *Renderer { return &Renderer{} }

// Warm compiles the WASM module ahead of the first invoice, so the wait does not
// land on the user pressing "Rechnung ausstellen".
func (r *Renderer) Warm(ctx context.Context) error {
	_, err := r.ensure(ctx)
	return err
}

// Close releases the WASM instance. It is idempotent, and safe to call while a
// Warm is still compiling: whoever gets the lock first wins, and a renderer that
// was closed before it started never compiles at all.
func (r *Renderer) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	compiler := r.compiler
	r.compiler = nil
	if compiler == nil {
		return nil
	}
	return compiler.Close(ctx)
}

// ensure hands back the compiler, starting it on first use.
//
// It returns the handle rather than leaving the caller to read the field: a
// concurrent Close clears it, and a caller that took the field afterwards would
// dereference nil.
func (r *Renderer) ensure(ctx context.Context) (*typst.Compiler, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, fmt.Errorf("der PDF-Renderer wurde bereits geschlossen")
	}
	if !r.started {
		r.started = true
		r.compiler, r.initErr = typst.NewCompiler(ctx)
	}
	return r.compiler, r.initErr
}

// RenderInvoicePDF produces the hybrid invoice document.
//
// The XML is passed through sys.inputs rather than as a file: the compiler's file
// resolver serves Typst sources only, and routing the payload through the
// template avoids having to escape a whole XML document into a Typst string
// literal.
func (r *Renderer) RenderInvoicePDF(ctx context.Context, inv *domain.Invoice, seller *domain.CompanySettings, buyer *domain.Contact, zugferdXML string) ([]byte, error) {
	if zugferdXML == "" {
		return nil, fmt.Errorf("ohne den strukturierten Rechnungsdatensatz lässt sich keine E-Rechnung erzeugen")
	}
	// Erst prüfen, dann den Compiler hochfahren: das Übersetzen des
	// WASM-Moduls kostet beim ersten Aufruf mehrere Sekunden.
	compiler, err := r.ensure(ctx)
	if err != nil {
		return nil, fmt.Errorf("der PDF-Renderer konnte nicht gestartet werden: %w", err)
	}

	pdf, err := compiler.Compile(ctx, typst.CompileRequest{
		Template: GenerateTypstTemplate(inv, seller, buyer),
		Data:     map[string]any{"zugferd_xml": zugferdXML},
		Fonts:    [][]byte{fontRegular, fontSemiBold, fontBold},
		PDFOpts: typst.PDFOptions{
			// PDF/A-3b is what lets the XML ride along as an associated file —
			// that is the whole basis of the hybrid format.
			Standards: []string{"a-3b"},
			// A stable identifier keeps two renderings of the same invoice
			// byte-comparable, which is what makes the "identical Mehrstück" of
			// GoBD Rz. 76 Abs. 2 more than a claim.
			Ident: inv.InvoiceNumber,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("die Rechnung %s konnte nicht als PDF erzeugt werden: %w", inv.InvoiceNumber, err)
	}
	return pdf, nil
}

// compilePlainForTest renders a document without an attachment. It exists so the
// extraction path can be tested against a PDF that is a sonstige Rechnung —
// which is what most incoming documents still are.
func (r *Renderer) compilePlainForTest(ctx context.Context) ([]byte, error) {
	compiler, err := r.ensure(ctx)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(ctx, typst.CompileRequest{
		Template: `#set document(title: "Rechnung", date: datetime(year: 2026, month: 8, day: 22))
= Rechnung ohne strukturierten Teil`,
		Fonts:   [][]byte{fontRegular},
		PDFOpts: typst.PDFOptions{Standards: []string{"a-3b"}},
	})
}

// RenderDocumentPDF übersetzt ein beliebiges Typst-Dokument.
//
// Sie steht neben RenderInvoicePDF, weil nicht jedes Dokument eine Rechnung
// ist: Bilanz und Gewinn- und Verlustrechnung brauchen denselben Compiler,
// dieselben eingebetteten Schriften und dieselbe reproduzierbare Ausgabe, aber
// keinen ZUGFeRD-Anhang — es hängt kein strukturierter Teil daran. PDF/A-3b
// bleibt trotzdem: ein Jahresabschluss ist zehn Jahre aufzubewahren (§ 257
// Abs. 4 HGB), und dafür ist die Schrifteneinbettung des Archivformats der
// Unterschied zwischen lesbar und unlesbar.
//
// ident macht zwei Läufe desselben Abschlusses byteweise vergleichbar; das ist
// es, was das „inhaltlich identische Mehrstück" der GoBD Rz. 76 Abs. 2 mehr als
// eine Behauptung sein lässt.
func (r *Renderer) RenderDocumentPDF(ctx context.Context, template, ident string) ([]byte, error) {
	if template == "" {
		return nil, fmt.Errorf("ohne Vorlage lässt sich kein PDF erzeugen")
	}
	compiler, err := r.ensure(ctx)
	if err != nil {
		return nil, fmt.Errorf("der PDF-Renderer konnte nicht gestartet werden: %w", err)
	}
	pdf, err := compiler.Compile(ctx, typst.CompileRequest{
		Template: template,
		Fonts:    [][]byte{fontRegular, fontSemiBold, fontBold},
		PDFOpts: typst.PDFOptions{
			Standards: []string{"a-3b"},
			Ident:     ident,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("das Dokument konnte nicht als PDF erzeugt werden: %w", err)
	}
	return pdf, nil
}
