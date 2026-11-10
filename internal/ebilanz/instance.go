package ebilanz

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Kontexte und Einheiten der Instanz (Leitfaden B.8). Vorjahreswerte
// übermittelt Buchfink nicht; die Instanz kennt nur das aktuelle Jahr.
const (
	contextInstant  = "I-AKTJAHR"
	contextDuration = "D-AKTJAHR"
	unitEUR         = "EUR"
	identifierRZF   = "http://www.rzf-nrw.de/Steuernummer"

	nsXBRLI   = "http://www.xbrl.org/2003/instance"
	nsLink    = "http://www.xbrl.org/2003/linkbase"
	nsXLink   = "http://www.w3.org/1999/xlink"
	nsXSI     = "http://www.w3.org/2001/XMLSchema-instance"
	nsISO4217 = "http://www.xbrl.org/2003/iso4217"
)

// node ist ein Element der Instanz mit Präfix im Namen, wie die Taxonomie
// es schreibt.
type node struct {
	name     string
	attrs    []xml.Attr
	text     string
	children []*node
}

func attr(name, value string) xml.Attr { return xml.Attr{Name: xml.Name{Local: name}, Value: value} }

// item ist ein Wert mit Kontext; ein monetärer Wert mit Einheit und zwei
// Nachkommastellen.
func (b *instanceBuilder) item(name string, value string) *node {
	el, _ := b.tax.Element(name)
	n := &node{name: b.tax.QName(name), attrs: []xml.Attr{attr("contextRef", b.contextOf(el))}, text: value}
	if el != nil && el.Monetary() {
		n.attrs = append(n.attrs, attr("unitRef", unitEUR), attr("decimals", "2"))
	}
	return n
}

func (b *instanceBuilder) money(name string, v domain.Cents) *node { return b.item(name, v.Decimal()) }

// nilItem ist ein leer übermitteltes Mussfeld (Leitfaden B.14.2.3).
func (b *instanceBuilder) nilItem(name string) *node {
	el, _ := b.tax.Element(name)
	n := &node{name: b.tax.QName(name), attrs: []xml.Attr{attr("xsi:nil", "true"), attr("contextRef", b.contextOf(el))}}
	if el != nil && el.Monetary() {
		n.attrs = append(n.attrs, attr("unitRef", unitEUR))
	}
	return n
}

func (b *instanceBuilder) tuple(name string, children ...*node) *node {
	return &node{name: b.tax.QName(name), children: children}
}

func (b *instanceBuilder) contextOf(el *Element) string {
	if el != nil && el.Period == "instant" {
		return contextInstant
	}
	return contextDuration
}

// instanceBuilder setzt die Instanz zusammen.
type instanceBuilder struct {
	tax         *Taxonomy
	microBilG   bool
	identifier  string
	begin, end  string
	closing     string
	gcd         []*node
	facts       []*node
	tuples      []*node
	usesInstant bool
}

// write schreibt die Instanz als XML.
func (b *instanceBuilder) write() ([]byte, error) {
	root := &node{name: "xbrli:xbrl", attrs: []xml.Attr{
		attr("xmlns:xbrli", nsXBRLI), attr("xmlns:link", nsLink), attr("xmlns:xlink", nsXLink),
		attr("xmlns:xsi", nsXSI), attr("xmlns:iso4217", nsISO4217),
	}}
	prefixes := make([]string, 0, len(b.tax.Namespaces))
	for prefix := range b.tax.Namespaces {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)
	for _, prefix := range prefixes {
		root.attrs = append(root.attrs, attr("xmlns:"+prefix, b.tax.Namespaces[prefix]))
	}
	gaapSchema := b.tax.EntrySchemas.Fiscal
	if b.microBilG {
		gaapSchema = b.tax.EntrySchemas.MicroBilG
	}
	for _, href := range []string{b.tax.EntrySchemas.GCD, gaapSchema} {
		root.children = append(root.children, &node{name: "link:schemaRef",
			attrs: []xml.Attr{attr("xlink:type", "simple"), attr("xlink:href", href)}})
	}
	entity := func() *node {
		return &node{name: "xbrli:entity", children: []*node{{name: "xbrli:identifier",
			attrs: []xml.Attr{attr("scheme", identifierRZF)}, text: b.identifier}}}
	}
	if b.usesInstant {
		root.children = append(root.children, &node{name: "xbrli:context", attrs: []xml.Attr{attr("id", contextInstant)},
			children: []*node{entity(), {name: "xbrli:period", children: []*node{{name: "xbrli:instant", text: b.closing}}}}})
	}
	root.children = append(root.children, &node{name: "xbrli:context", attrs: []xml.Attr{attr("id", contextDuration)},
		children: []*node{entity(), {name: "xbrli:period", children: []*node{
			{name: "xbrli:startDate", text: b.begin}, {name: "xbrli:endDate", text: b.end}}}}})
	root.children = append(root.children, &node{name: "xbrli:unit", attrs: []xml.Attr{attr("id", unitEUR)},
		children: []*node{{name: "xbrli:measure", text: "iso4217:EUR"}}})
	root.children = append(root.children, b.gcd...)
	root.children = append(root.children, b.facts...)
	root.children = append(root.children, b.tuples...)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "\t")
	if err := encodeNode(enc, root); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	buf.WriteString("\n")
	return buf.Bytes(), nil
}

func encodeNode(enc *xml.Encoder, n *node) error {
	start := xml.StartElement{Name: xml.Name{Local: n.name}, Attr: n.attrs}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if n.text != "" {
		if err := enc.EncodeToken(xml.CharData(n.text)); err != nil {
			return err
		}
	}
	for _, child := range n.children {
		if err := encodeNode(enc, child); err != nil {
			return err
		}
	}
	return enc.EncodeToken(xml.EndElement{Name: start.Name})
}

// Envelope bettet die Instanz in den Nutzdatenteil der ELSTER-Übermittlung
// (ebilanz:EBilanz mit Stichtag JJJJMMTT). TransferHeader und
// NutzdatenHeader setzt das übermittelnde Programm mit seiner Hersteller-ID.
func Envelope(instance []byte, closingDate string) ([]byte, error) {
	if len(closingDate) != 10 {
		return nil, fmt.Errorf("der Bilanzstichtag fehlt (erwartet JJJJ-MM-TT)")
	}
	body := bytes.TrimPrefix(bytes.TrimSpace(instance), []byte(strings.TrimSpace(xml.Header)))
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<ebilanz:EBilanz version="000002" xmlns:ebilanz="http://rzf.fin-nrw.de/RMS/EBilanz/2016/XMLSchema">` + "\n")
	buf.WriteString("\t<ebilanz:stichtag>" + strings.ReplaceAll(closingDate, "-", "") + "</ebilanz:stichtag>\n")
	buf.Write(bytes.TrimSpace(body))
	buf.WriteString("\n</ebilanz:EBilanz>\n")
	return buf.Bytes(), nil
}
