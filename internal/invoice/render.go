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
	once     sync.Once
	compiler *typst.Compiler
	initErr  error
}

// NewRenderer returns a renderer. The WASM module is compiled on first use and
// then reused: that compilation takes several seconds, so the instance is meant
// to be long-lived, and start-up stays free of a cost most sessions never incur.
func NewRenderer() *Renderer { return &Renderer{} }

// Warm compiles the WASM module ahead of the first invoice, so the wait does not
// land on the user pressing "Rechnung ausstellen".
func (r *Renderer) Warm(ctx context.Context) error { return r.ensure(ctx) }

// Close releases the WASM instance.
func (r *Renderer) Close(ctx context.Context) error {
	if r.compiler == nil {
		return nil
	}
	return r.compiler.Close(ctx)
}

func (r *Renderer) ensure(ctx context.Context) error {
	r.once.Do(func() {
		r.compiler, r.initErr = typst.NewCompiler(ctx)
	})
	return r.initErr
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
	if err := r.ensure(ctx); err != nil {
		return nil, fmt.Errorf("der PDF-Renderer konnte nicht gestartet werden: %w", err)
	}

	pdf, err := r.compiler.Compile(ctx, typst.CompileRequest{
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
