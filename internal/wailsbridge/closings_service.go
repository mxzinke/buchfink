package wailsbridge

import (
	"context"
	"fmt"
	"strconv"

	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/declaration"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/service"
)

// Die Abschlüsse: Pflichtenkalender, Jahreserklärungen und E-Bilanz.
//
// Bestätigen, Entfallen, Übermitteln und das Bestätigen eines Vorschlags sind
// dem Menschen in der App vorbehalten (§ 146 Abs. 4 AO, § 42a GmbHG); der Kern
// weist sie für jeden anderen Erfassungsweg ab.

// GetObligations liefert die Pflichten eines Geschäftsjahres mit Stand,
// Fälligkeit, Nachweis und Ziel in der Oberfläche, die früheste zuerst.
// Bestätigen und Entfallen stehen in overview_service.go.
func (b *BuchfinkBridge) GetObligations(year int) ([]ObligationView, error) {
	year = b.yearOr(year)
	return readList(b, func(ctx context.Context, s *core.Services) ([]ObligationView, error) {
		list, err := s.Obligations.List(ctx, service.ObligationQuery{FiscalYear: year})
		if err != nil {
			return nil, err
		}
		out := make([]ObligationView, 0, len(list))
		for i := range list {
			if list[i].Hints == nil {
				list[i].Hints = []string{}
			}
			out = append(out, *obligationView(&list[i]))
		}
		return out, nil
	})
}

// DeclarationView ist eine Jahreserklärung, wie die Oberfläche sie zeigt:
// zuerst die Übersicht mit Ergebnis und offenen Zeilen, darunter die Vordrucke
// mit allen Zeilen und ihrer Herkunft.
type DeclarationView struct {
	Overview    declaration.Overview     `json:"overview"`
	Declaration *declaration.Declaration `json:"declaration"`
}

// GetDeclaration rechnet eine Jahreserklärung (ust_jahr, kst, gewst) aus dem
// heutigen Stand der Bücher.
func (b *BuchfinkBridge) GetDeclaration(kind string, year int) (*DeclarationView, error) {
	year = b.yearOr(year)
	return read(b, func(ctx context.Context, s *core.Services) (*DeclarationView, error) {
		d, err := s.Declarations.Build(ctx, declaration.Kind(kind), year)
		if err != nil {
			return nil, err
		}
		return &DeclarationView{Overview: d.Overview(), Declaration: d}, nil
	})
}

// SetDeclarationInput legt eine Angabe zu einer Zeile ab, etwa einen
// Verlustvortrag aus dem Bescheid. In der App gesetzt gilt sie sofort.
func (b *BuchfinkBridge) SetDeclarationInput(req service.InputRequest) (*declaration.Input, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*declaration.Input, error) {
		return s.Declarations.SetInput(ctx, req)
	})
}

// ConfirmDeclarationInput bestätigt den Vorschlag eines Agenten zu einer
// Zeile, die dem Menschen vorbehalten ist.
func (b *BuchfinkBridge) ConfirmDeclarationInput(kind string, year int, key string) (*declaration.Input, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*declaration.Input, error) {
		return s.Declarations.ConfirmInput(ctx, declaration.Kind(kind), year, key)
	}, core.AppOnly())
}

// DeleteDeclarationInput nimmt eine Angabe zurück; die Zeile rechnet danach
// wieder aus den Büchern oder bleibt offen.
func (b *BuchfinkBridge) DeleteDeclarationInput(kind string, year int, key string) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.Declarations.DeleteInput(ctx, declaration.Kind(kind), year, key)
	})
}

// SubmitDeclaration hält die Übermittlung in Mein ELSTER mit Datum,
// Transferticket und dem Zahlen-Hash des übermittelten Stands fest.
func (b *BuchfinkBridge) SubmitDeclaration(req service.SubmissionRequest) (*domain.DeclarationRecord, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.DeclarationRecord, error) {
		return s.Declarations.MarkSubmitted(ctx, req)
	}, core.AppOnly())
}

// ExportDeclarationJSON schreibt die Erklärung als JSON in einen gewählten
// Ordner und hält den Export fest. Leer heißt: der Dialog wurde abgebrochen.
func (b *BuchfinkBridge) ExportDeclarationJSON(kind string, year int) (*core.DeclarationExport, error) {
	year = b.yearOr(year)
	dir, err := b.SelectExportDirectoryDialog("Ordner für die Erklärung wählen")
	if err != nil || dir == "" {
		return nil, err
	}
	return operate(b, func(ctx context.Context, t *core.Tenant) (*core.DeclarationExport, error) {
		return t.ExportDeclaration(ctx, declaration.Kind(kind), year, dir)
	})
}

// SaveEBilanzAs erzeugt die E-Bilanz eines Geschäftsjahres und schreibt sie an
// einen gewählten Ort: als XBRL-Instanz (format xbrl oder leer) oder als
// Summen- und Saldenliste mit Taxonomie-Position je Konto (susa_csv), etwa für
// eBilanz+. Leer heißt: der Dialog wurde abgebrochen.
func (b *BuchfinkBridge) SaveEBilanzAs(year int, format string) (string, error) {
	year = b.yearOr(year)
	name, label := fmt.Sprintf("e-bilanz-%d.xbrl", year), "XBRL-Instanz"
	switch format {
	case "", "xbrl":
	case "susa_csv":
		name, label = fmt.Sprintf("ebilanz-%d-susa.csv", year), "Summen- und Saldenliste (CSV)"
	default:
		return "", fmt.Errorf("unbekanntes Format %q; xbrl oder susa_csv", format)
	}
	return b.saveEBilanz(year, service.EBilanzOptions{}, format == "susa_csv",
		"E-Bilanz speichern", name, fmt.Sprintf("E-Bilanz %d als %s", year, label))
}

// SaveOpeningEBilanzAs erzeugt die E-Bilanz der Eröffnungsbilanz (§ 5b Abs. 1
// Satz 4 EStG) als XBRL-Instanz und schreibt sie an einen gewählten Ort. year
// ist das Geschäftsjahr, in das die Gründung fällt. Leer heißt: der Dialog
// wurde abgebrochen.
func (b *BuchfinkBridge) SaveOpeningEBilanzAs(year int) (string, error) {
	year = b.yearOr(year)
	return b.saveEBilanz(year, service.EBilanzOptions{Kind: ebilanz.KindOpening}, false,
		"Eröffnungsbilanz als E-Bilanz speichern", fmt.Sprintf("e-bilanz-eroeffnung-%d.xbrl", year),
		fmt.Sprintf("E-Bilanz der Eröffnungsbilanz %d als XBRL-Instanz", year))
}

// saveEBilanz erzeugt vor dem Dialog: blockierende Befunde melden sich, bevor
// jemand einen Speicherort sucht.
func (b *BuchfinkBridge) saveEBilanz(
	year int, opts service.EBilanzOptions, susa bool, title, name, what string,
) (string, error) {
	data, err := read(b, func(ctx context.Context, s *core.Services) ([]byte, error) {
		res, err := s.EBilanz.Generate(ctx, year, opts)
		if err != nil {
			return nil, err
		}
		if err := res.Report.BlockingError(); err != nil {
			return nil, err
		}
		if susa {
			return ebilanz.SuSaCSV(res)
		}
		return res.Instance, nil
	})
	if err != nil {
		return "", err
	}
	path, err := b.saveFileDialog(title, name)
	if err != nil || path == "" {
		return "", err
	}
	if err := b.handOut(path, data, "EBILANZ", strconv.Itoa(year),
		fmt.Sprintf("%s nach %s herausgegeben", what, path)); err != nil {
		return "", err
	}
	return path, nil
}
