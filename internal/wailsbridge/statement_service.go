package wailsbridge

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
)

// Bilanz und Gewinn- und Verlustrechnung.
//
// Bis hierher entstanden beide im Frontend durch Filtern nach Kontenklasse.
// Jetzt liefert die Bridge die fertige Gliederung nach §§ 266, 275 HGB — mit
// Vorjahresspalte, Größenklasse, Fristen und Zuordnungsbericht. Die Ansicht
// rechnet nichts nach.

// GetStatement stellt den Jahresabschluss eines Geschäftsjahres auf.
//
// depth darf leer bleiben; dann gilt die Gliederungstiefe der Größenklasse
// (§ 266 Abs. 1 Sätze 3 und 4 HGB).
func (b *BuchfinkBridge) GetStatement(year int, depth string) (*domain.FinancialStatement, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.statementSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	// Leer heißt hier „die Tiefe der Größenklasse" und nicht „volle
	// Gliederung": den Unterschied kennt der Dienst, nicht die Umwandlung.
	parsed := domain.StatementDepth("")
	if depth != "" {
		var err error
		if parsed, err = domain.ParseStatementDepth(depth); err != nil {
			return nil, err
		}
	}
	return b.statementSvc.Build(context.Background(), year, parsed)
}

// GetSizeClass beurteilt die Größenklasse eines Geschäftsjahres nach den
// §§ 267, 267a HGB samt Begründung und Folgen.
func (b *BuchfinkBridge) GetSizeClass(year int) (*accounting.SizeClass, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.statementSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.statementSvc.SizeClassFor(context.Background(), year)
}

// GetStatementDeadlines liefert die Termine für Aufstellung und Offenlegung.
func (b *BuchfinkBridge) GetStatementDeadlines(year int) ([]domain.Deadline, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.statementSvc == nil {
		return []domain.Deadline{}, nil
	}
	return b.statementSvc.Deadlines(context.Background(), year)
}

// ExportStatementPDF liefert das Dokument als Base64 — wie der Rechnungs- und
// der E-Bilanz-Export auch, damit das Frontend es unverändert speichern kann.
//
// Die Gliederungstiefe ist nicht wählbar: ausgegeben wird die Tiefe, in der der
// Abschluss offenzulegen ist. Eine Datei, die tiefer gliedert als der
// offengelegte Abschluss, wäre ein zweites Dokument mit demselben Anspruch.
func (b *BuchfinkBridge) ExportStatementPDF(year int) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.statementSvc == nil {
		return "", fmt.Errorf("kein aktiver Mandant")
	}
	pdf, err := b.statementSvc.ExportPDF(context.Background(), year, "")
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pdf), nil
}

// ExportStatementCSV liefert die Gliederung als CSV-Text (UTF-8, Semikolon).
func (b *BuchfinkBridge) ExportStatementCSV(year int) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.statementSvc == nil {
		return "", fmt.Errorf("kein aktiver Mandant")
	}
	return b.statementSvc.ExportCSV(context.Background(), year, "")
}

// GetEBilanzMappingReport zeigt vor dem Export, welches Konto unter welcher
// Gliederungsposition und welchem Taxonomie-Element erscheint — und was die
// Erzeugung verhindert.
func (b *BuchfinkBridge) GetEBilanzMappingReport(year int) (*ebilanz.MappingReport, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.ebilanzSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.ebilanzSvc.MappingReport(context.Background(), year)
}

// SetPriorYearRevenue hält den Gesamtumsatz des Vorjahres fest. An ihm hängt
// die Übergangsfrist des § 27 Abs. 38 Nr. 2 UStG: bis 800.000 € darf 2027 noch
// eine sonstige Rechnung ohne strukturierten Datensatz ausgestellt werden.
func (b *BuchfinkBridge) SetPriorYearRevenue(year int, amount domain.Cents) (*domain.FiscalYear, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.SetPriorYearRevenue(context.Background(), year, amount)
}

// SetAverageEmployees hält die durchschnittliche Arbeitnehmerzahl eines
// Geschäftsjahres fest. Sie ist das dritte Merkmal des § 267 Abs. 1 HGB und
// lässt sich aus der Buchführung nicht ableiten.
func (b *BuchfinkBridge) SetAverageEmployees(year, count int) (*domain.FiscalYear, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ensureWritable(); err != nil {
		return nil, err
	}
	if b.closingSvc == nil {
		return nil, fmt.Errorf("kein aktiver Mandant")
	}
	return b.closingSvc.SetAverageEmployees(context.Background(), year, count)
}
