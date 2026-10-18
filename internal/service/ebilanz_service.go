package service

import (
	"context"
	"fmt"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
)

// AdjustmentSource liefert die Abweichungen der Steuerbilanz je Konto, aus
// denen die Überleitungsrechnung der E-Bilanz entsteht.
type AdjustmentSource interface {
	Adjustments(ctx context.Context, year int) ([]ebilanz.Adjustment, error)
}

// EBilanzService erzeugt die E-Bilanz (§ 5b EStG) aus derselben Gliederung wie
// die Bilanz auf dem Schirm.
type EBilanzService struct {
	statementSvc *StatementService
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	adjustments  AdjustmentSource
}

// NewEBilanzService wires the E-Bilanz export.
func NewEBilanzService(
	statementSvc *StatementService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
) *EBilanzService {
	return &EBilanzService{
		statementSvc: statementSvc,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
	}
}

// SetAdjustmentSource verdrahtet die Abweichungen der Steuerbilanz. Ohne sie
// geht die Bilanz als Einheitsbilanz.
func (s *EBilanzService) SetAdjustmentSource(src AdjustmentSource) { s.adjustments = src }

// EBilanzOptions sind die Wahlrechte der Übermittlung.
type EBilanzOptions struct {
	// Version ist die Taxonomie; leer heißt Regelfassung für das
	// Wirtschaftsjahr.
	Version string `json:"version"`
	// MicroBilG wählt die GuV der Kleinstkapitalgesellschaft (§ 275 Abs. 5 HGB).
	MicroBilG bool `json:"microBilG"`
	// Envelope bettet die Instanz in den Nutzdatenteil der ELSTER-Übermittlung.
	Envelope bool `json:"envelope"`
}

// MappingReport ist der Prüfbericht vor dem Export: wohin jedes Konto geht und
// was die Erzeugung verhindert.
func (s *EBilanzService) MappingReport(ctx context.Context, year int) (*ebilanz.MappingReport, error) {
	res, err := s.Generate(ctx, year, EBilanzOptions{})
	if err != nil {
		return nil, err
	}
	return res.Report, nil
}

// Generate prüft und erzeugt die E-Bilanz eines Geschäftsjahres, ohne sie zu
// protokollieren. Die Instanz ist leer, solange Befunde sie verhindern.
func (s *EBilanzService) Generate(ctx context.Context, year int, opts EBilanzOptions) (*ebilanz.Result, error) {
	in, err := s.input(ctx, year, opts)
	if err != nil {
		return nil, err
	}
	return ebilanz.Generate(in)
}

// ExportXBRL erzeugt die Instanz für ein Geschäftsjahr mit den Vorgaben.
func (s *EBilanzService) ExportXBRL(ctx context.Context, year int) (string, error) {
	return s.ExportXBRLWith(ctx, year, EBilanzOptions{})
}

// ExportXBRLWith erzeugt die Instanz mit den gewählten Wahlrechten.
func (s *EBilanzService) ExportXBRLWith(ctx context.Context, year int, opts EBilanzOptions) (string, error) {
	in, err := s.input(ctx, year, opts)
	if err != nil {
		return "", err
	}
	res, err := ebilanz.Generate(in)
	if err != nil {
		return "", err
	}
	if err := res.Report.BlockingError(); err != nil {
		return "", err
	}
	instance := res.Instance
	if opts.Envelope {
		if instance, err = ebilanz.Envelope(instance, in.FiscalYearEnd); err != nil {
			return "", err
		}
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionExport,
		"EBILANZ",
		fmt.Sprintf("%d", year),
		fmt.Sprintf("E-Bilanz %d als XBRL-Instanz erzeugt (Taxonomie %s, %s)",
			year, res.Version.Version, res.Report.AccountingStandard),
	); err != nil {
		return "", err
	}
	return string(instance), nil
}

// ExportSuSaCSV erzeugt die Summen- und Saldenliste zur E-Bilanz mit
// denselben Konten und Salden wie die Instanz (ebilanz.SuSaCSV).
func (s *EBilanzService) ExportSuSaCSV(ctx context.Context, year int, opts EBilanzOptions) ([]byte, error) {
	res, err := s.Generate(ctx, year, opts)
	if err != nil {
		return nil, err
	}
	data, err := ebilanz.SuSaCSV(res)
	if err != nil {
		return nil, err
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionExport, "EBILANZ", fmt.Sprintf("%d", year),
		fmt.Sprintf("E-Bilanz %d als Summen- und Saldenliste (CSV) erzeugt (Taxonomie %s)", year, res.Version.Version),
	); err != nil {
		return nil, err
	}
	return data, nil
}

// ExportOpeningXBRL erzeugt die Instanz der Eröffnungsbilanz (§ 5b Abs. 1
// EStG, § 242 Abs. 1 HGB): Bilanz ohne GuV auf den Tag, an dem das
// Handelsgewerbe beginnt.
func (s *EBilanzService) ExportOpeningXBRL(ctx context.Context, cutoff string) (string, error) {
	if len(cutoff) != 10 {
		return "", fmt.Errorf("der Stichtag fehlt oder ist unvollständig (erwartet JJJJ-MM-TT)")
	}
	if s.statementSvc == nil {
		return "", fmt.Errorf("die Gliederung ist nicht verfügbar")
	}
	settings, err := s.settings(ctx)
	if err != nil {
		return "", err
	}
	stmt, err := s.statementSvc.StatementAt(ctx, cutoff, domain.DepthFull)
	if err != nil {
		return "", err
	}
	accounts, err := s.statementSvc.AccountsAt(ctx, cutoff)
	if err != nil {
		return "", err
	}
	fy, err := s.statementSvc.period(ctx, stmt.FiscalYear)
	if err != nil {
		return "", err
	}
	res, err := ebilanz.Generate(ebilanz.Input{
		Settings: settings, Statement: stmt, Accounts: accounts, Kind: ebilanz.KindOpening,
		ClosingDate: cutoff, FiscalYearBegin: cutoff, FiscalYearEnd: fy.EndDate,
	})
	if err != nil {
		return "", err
	}
	if err := res.Report.BlockingError(); err != nil {
		return "", err
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionExport,
		"EBILANZ",
		cutoff,
		fmt.Sprintf("Eröffnungsbilanz auf den %s als XBRL-Instanz erzeugt (§ 5b Abs. 1 EStG)", cutoff),
	); err != nil {
		return "", err
	}
	return string(res.Instance), nil
}

func (s *EBilanzService) settings(ctx context.Context) (*domain.CompanySettings, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}
	return settings, nil
}

// input beschafft alles, was in die E-Bilanz eines Geschäftsjahres eingeht.
func (s *EBilanzService) input(ctx context.Context, year int, opts EBilanzOptions) (ebilanz.Input, error) {
	if err := requireYear(year); err != nil {
		return ebilanz.Input{}, err
	}
	if s.statementSvc == nil {
		return ebilanz.Input{}, fmt.Errorf("die Gliederung ist nicht verfügbar")
	}
	settings, err := s.settings(ctx)
	if err != nil {
		return ebilanz.Input{}, err
	}
	stmt, err := s.statementSvc.Statement(ctx, year, domain.DepthFull)
	if err != nil {
		return ebilanz.Input{}, err
	}
	accounts, err := s.statementSvc.Accounts(ctx, year)
	if err != nil {
		return ebilanz.Input{}, err
	}
	fy, err := s.statementSvc.period(ctx, year)
	if err != nil {
		return ebilanz.Input{}, err
	}
	in := ebilanz.Input{
		Settings: settings, Statement: stmt, Accounts: accounts, Kind: ebilanz.KindAnnual,
		FiscalYearBegin: fy.StartDate, FiscalYearEnd: fy.EndDate,
		Version: opts.Version, MicroBilG: opts.MicroBilG,
	}
	if sizeClass, err := s.statementSvc.SizeClassFor(ctx, year); err == nil && sizeClass != nil {
		in.SizeClass = sizeClass.Class
	}
	if s.adjustments != nil {
		if in.Adjustments, err = s.adjustments.Adjustments(ctx, year); err != nil {
			return ebilanz.Input{}, fmt.Errorf("die Abweichungen der Steuerbilanz ließen sich nicht ermitteln: %w", err)
		}
	}

	// Vorjahr: Daten für die Stammdaten, Eigenkapital für den
	// Betriebsvermögensvergleich. Im Gründungsjahr ist das Vorjahresvermögen
	// null; ohne Vorjahr in den Büchern bleibt es unbekannt.
	switch {
	case settings.FoundedOn != "" && settings.FoundedOn >= fy.StartDate:
		zero := domain.Cents(0)
		in.PriorEquity, in.PriorPaidIn = &zero, &zero
	case stmt.HasPrior:
		if prior, err := s.statementSvc.period(ctx, year-1); err == nil {
			in.PriorFiscalYearBegin, in.PriorFiscalYearEnd = prior.StartDate, prior.EndDate
		}
		var equity, paidIn domain.Cents
		if line := stmt.Line("passiva.A"); line != nil {
			equity = line.PriorAmount
		}
		for _, key := range []string{"passiva.A.I", "passiva.A.II"} {
			if line := stmt.Line(key); line != nil {
				paidIn += line.PriorAmount
			}
		}
		// Steuerlich zählt das Eigenkapital nach Überleitung.
		for _, adj := range in.Adjustments {
			equity += adj.Prior
		}
		in.PriorEquity, in.PriorPaidIn = &equity, &paidIn
	}
	return in, nil
}
