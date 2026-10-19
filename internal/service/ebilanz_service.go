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
	// Kind ist die Bilanzart; leer heißt Jahresabschluss. Die
	// Eröffnungsbilanz (ebilanz.KindOpening) steht auf den Gründungstag aus
	// den Unternehmensdaten und gehört zum Geschäftsjahr, in das er fällt.
	Kind ebilanz.Kind `json:"kind"`
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

// Generate prüft und erzeugt die E-Bilanz eines Geschäftsjahres oder seine
// Eröffnungsbilanz, ohne sie zu protokollieren. Die Instanz ist leer, solange
// Befunde sie verhindern.
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
	entity, details := fmt.Sprintf("%d", year), fmt.Sprintf("E-Bilanz %d als XBRL-Instanz erzeugt (Taxonomie %s, %s)",
		year, res.Version.Version, res.Report.AccountingStandard)
	if in.Kind == ebilanz.KindOpening {
		entity, details = in.ClosingDate, fmt.Sprintf(
			"Eröffnungsbilanz zum %s als XBRL-Instanz erzeugt (Taxonomie %s, § 5b Abs. 1 Satz 4 EStG)",
			germanDate(in.ClosingDate), res.Version.Version)
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionExport, "EBILANZ", entity, details); err != nil {
		return "", err
	}
	return string(instance), nil
}

// ExportSuSaCSV erzeugt die Summen- und Saldenliste zur E-Bilanz mit
// denselben Konten und Salden wie die Instanz (ebilanz.SuSaCSV).
func (s *EBilanzService) ExportSuSaCSV(ctx context.Context, year int, opts EBilanzOptions) ([]byte, error) {
	in, err := s.input(ctx, year, opts)
	if err != nil {
		return nil, err
	}
	res, err := ebilanz.Generate(in)
	if err != nil {
		return nil, err
	}
	data, err := ebilanz.SuSaCSV(res)
	if err != nil {
		return nil, err
	}
	entity, details := fmt.Sprintf("%d", year),
		fmt.Sprintf("E-Bilanz %d als Summen- und Saldenliste (CSV) erzeugt (Taxonomie %s)", year, res.Version.Version)
	if in.Kind == ebilanz.KindOpening {
		entity, details = in.ClosingDate, fmt.Sprintf(
			"Eröffnungsbilanz zum %s als Summen- und Saldenliste (CSV) erzeugt (Taxonomie %s)",
			germanDate(in.ClosingDate), res.Version.Version)
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionExport, "EBILANZ", entity, details); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *EBilanzService) settings(ctx context.Context) (*domain.CompanySettings, error) {
	settings, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}
	return settings, nil
}

// input beschafft alles, was in die E-Bilanz eines Geschäftsjahres oder
// seine Eröffnungsbilanz eingeht.
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
	switch opts.Kind {
	case "", ebilanz.KindAnnual:
	case ebilanz.KindOpening:
		return s.openingInput(ctx, year, settings, opts)
	default:
		return ebilanz.Input{}, fmt.Errorf("unbekannte Bilanzart %q (erwartet %s oder %s)",
			opts.Kind, ebilanz.KindAnnual, ebilanz.KindOpening)
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

// openingInput beschafft die Eröffnungsbilanz (§ 242 Abs. 1 HGB): Bilanz ohne
// GuV auf den Gründungstag, mit der E-Bilanz zu übermitteln nach § 5b Abs. 1
// Satz 4 EStG. Sie entsteht aus den Buchungen bis zu diesem Tag und gehört zum
// Geschäftsjahr, in das er fällt.
func (s *EBilanzService) openingInput(
	ctx context.Context, year int, settings *domain.CompanySettings, opts EBilanzOptions,
) (ebilanz.Input, error) {
	cutoff := settings.FoundedOn
	if len(cutoff) != 10 {
		return ebilanz.Input{}, fmt.Errorf("in den Unternehmensdaten fehlt das Gründungsdatum; " +
			"die Eröffnungsbilanz steht auf den Tag der Beurkundung")
	}
	stmt, err := s.statementSvc.StatementAt(ctx, cutoff, domain.DepthFull)
	if err != nil {
		return ebilanz.Input{}, err
	}
	if stmt.FiscalYear != year {
		return ebilanz.Input{}, fmt.Errorf("die Eröffnungsbilanz zum %s gehört zum Geschäftsjahr %d",
			germanDate(cutoff), stmt.FiscalYear)
	}
	accounts, err := s.statementSvc.AccountsAt(ctx, cutoff)
	if err != nil {
		return ebilanz.Input{}, err
	}
	if !anyBalance(accounts) {
		return ebilanz.Input{}, fmt.Errorf("zum %s steht keine Buchung in den Büchern; die Eröffnungsbilanz "+
			"entsteht aus den Gründungsbuchungen", germanDate(cutoff))
	}
	fy, err := s.statementSvc.period(ctx, year)
	if err != nil {
		return ebilanz.Input{}, err
	}
	return ebilanz.Input{
		Settings: settings, Statement: stmt, Accounts: accounts, Kind: ebilanz.KindOpening,
		ClosingDate: cutoff, FiscalYearBegin: cutoff, FiscalYearEnd: fy.EndDate, Version: opts.Version,
	}, nil
}

func anyBalance(accounts []domain.Account) bool {
	for _, a := range accounts {
		if a.Balance != 0 {
			return true
		}
	}
	return false
}
