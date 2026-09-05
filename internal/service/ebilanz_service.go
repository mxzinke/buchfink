package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
)

// AnlagenspiegelSource liefert die Entwicklung des Anlagevermögens.
//
// Der Export kennt die Anlagenbuchhaltung darüber und nicht weiter: er braucht
// eine Auswertung, keine Kartei.
type AnlagenspiegelSource interface {
	Anlagenspiegel(ctx context.Context) (*domain.Anlagenspiegel, error)
}

// EBilanzService erzeugt die XBRL-Instanz der E-Bilanz (§ 5b EStG).
//
// Sie entsteht aus derselben Gliederung wie die Bilanz auf dem Schirm. Bis
// hierher war sie ein zweiter Weg zu denselben Zahlen — mit einer eigenen
// Kontentabelle, die neunzig Konten kannte und alle übrigen still auf einer
// Sammelposition ablegte. Zwei Wege zu einer Bilanz sind einer zu viel.
type EBilanzService struct {
	statementSvc *StatementService
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	assets       AnlagenspiegelSource
	fiscalYear   int
}

// NewEBilanzService wires the E-Bilanz export.
func NewEBilanzService(
	statementSvc *StatementService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *EBilanzService {
	return &EBilanzService{
		statementSvc: statementSvc,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
		fiscalYear:   fiscalYear,
	}
}

// SetAnlagenspiegelSource verdrahtet den Anlagenspiegel in den Export. Ohne sie
// entsteht die Instanz wie sonst, nur ohne den Nachweis zum Anlagevermögen.
func (s *EBilanzService) SetAnlagenspiegelSource(src AnlagenspiegelSource) { s.assets = src }

// SetFiscalYear updates the active fiscal year.
func (s *EBilanzService) SetFiscalYear(year int) { s.fiscalYear = year }

// MappingReport ist der Zuordnungsbericht vor dem Export: welches Konto unter
// welcher Gliederungsposition und welchem Taxonomie-Element erscheint, und was
// die Erzeugung verhindert.
func (s *EBilanzService) MappingReport(ctx context.Context, year int) (*ebilanz.MappingReport, error) {
	in, err := s.input(ctx, year)
	if err != nil {
		return nil, err
	}
	return ebilanz.BuildMappingReport(in.FiscalYear, in.Statement, in.Accounts)
}

// ExportXBRL erzeugt die Instanz für ein Geschäftsjahr.
func (s *EBilanzService) ExportXBRL(ctx context.Context, year int) (string, error) {
	in, err := s.input(ctx, year)
	if err != nil {
		return "", err
	}

	// Der Anlagenspiegel ist Bestandteil des Anhangs (§ 284 Abs. 3 HGB) und im
	// Kontennachweis das, was den ausgewiesenen Buchwert erklärt. Scheitert seine
	// Auswertung, entsteht die Instanz ohne ihn — eine E-Bilanz an einer nicht
	// rechenbaren Kartei scheitern zu lassen hülfe niemandem.
	if s.assets != nil {
		in.Anlagenspiegel, _ = s.assets.Anlagenspiegel(ctx)
	}

	xbrl, _, err := ebilanz.GenerateEBilanzXBRL(in)
	if err != nil {
		return "", err
	}

	_ = s.auditRepo.Log(
		ctx,
		domain.AuditActionExport,
		"EBILANZ",
		fmt.Sprintf("%d", in.FiscalYear),
		fmt.Sprintf("E-Bilanz %d als XBRL-Instanz erzeugt", in.FiscalYear),
	)
	return xbrl, nil
}

// input beschafft alles, was in die Instanz eingeht.
func (s *EBilanzService) input(ctx context.Context, year int) (ebilanz.InstanceInput, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	if s.statementSvc == nil {
		return ebilanz.InstanceInput{}, fmt.Errorf("die Gliederung ist nicht verfügbar")
	}

	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return ebilanz.InstanceInput{}, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}

	// Immer die volle Gliederung: die E-Bilanz kennt die Erleichterungen des
	// § 266 Abs. 1 HGB nicht, sie will jede Position.
	stmt, err := s.statementSvc.Statement(ctx, year, domain.DepthFull)
	if err != nil {
		return ebilanz.InstanceInput{}, err
	}
	accounts, err := s.statementSvc.Accounts(ctx, year)
	if err != nil {
		return ebilanz.InstanceInput{}, err
	}

	in := ebilanz.InstanceInput{
		Settings: settings, Statement: stmt, Accounts: accounts, FiscalYear: year,
		StartDate: fmt.Sprintf("%d-01-01", year), EndDate: fmt.Sprintf("%d-12-31", year),
	}
	if fy, err := s.statementSvc.period(ctx, year); err == nil {
		in.StartDate, in.EndDate = fy.StartDate, fy.EndDate
	}
	if stmt.HasPrior {
		in.PriorStartDate = fmt.Sprintf("%d-01-01", year-1)
		in.PriorEndDate = fmt.Sprintf("%d-12-31", year-1)
		if fy, err := s.statementSvc.period(ctx, year-1); err == nil {
			in.PriorStartDate, in.PriorEndDate = fy.StartDate, fy.EndDate
		}
	}
	return in, nil
}
