package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

func (s *PeriodReader) closingCalendar() fiscalCalendar {
	calendar := s.fiscalCalendar
	calendar.limitMonth = true
	return calendar
}

// FiscalYear liefert das Geschäftsjahr, ohne es anzulegen.
//
// Die Vorschau darf nichts erzeugen: wer den Abschlussstand von 2026 ansieht,
// fragt damit auch nach dem Vortrag ins Jahr 2027 — und fände es danach in der
// Jahresauswahl wieder, ohne es angelegt zu haben.
func (s *PeriodReader) FiscalYear(ctx context.Context, year int) (*domain.FiscalYear, error) {
	fy, err := s.fiscalYearRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	if fy != nil {
		return fy, nil
	}
	derived := s.derive(ctx, year)
	if err := derived.Validate(); err != nil {
		return nil, err
	}
	return derived, nil
}

// derive baut den Zeitraum eines Geschäftsjahres aus den Unternehmensdaten.
func (s *PeriodReader) derive(ctx context.Context, year int) *domain.FiscalYear {
	startMonth := s.closingCalendar().fiscalYearStartMonth(ctx)
	start := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	startDate := start.Format("2006-01-02")
	endDate := lastDayOfTwelveMonths(startDate)

	// Das Gründungsjahr beginnt mit der Beurkundung, nicht am Kalenderanfang:
	// vorher gab es die Gesellschaft nicht, und ein Jahr, das vor ihrer
	// Entstehung beginnt, hätte einen Zeitraum ohne Bücher.
	if founded := s.foundingDate(ctx); founded > startDate && founded <= endDate {
		startDate = founded
	}
	return domain.NewFiscalYear(year, startDate, endDate)
}

// foundingDate ist der Tag der Beurkundung: das Gründungsdatum der
// Unternehmensdaten. Die Einrichtung setzt es bei einer Gründung als Starttag
// der Bücher, die Erfassung der Gründung schreibt es fort.
func (s *PeriodReader) foundingDate(ctx context.Context) string {
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil && len(cfg.FoundedOn) == 10 {
			return cfg.FoundedOn
		}
	}
	return ""
}

func (s *PeriodService) yearOf(ctx context.Context, year int) (*domain.FiscalYear, error) {
	fy, err := s.fiscalYearRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	if fy != nil {
		return fy, nil
	}
	derived := s.derive(ctx, year)
	if err := derived.Validate(); err != nil {
		return nil, err
	}
	if err := s.fiscalYearRepo.Save(ctx, derived); err != nil {
		return nil, fmt.Errorf("das Geschäftsjahr %d konnte nicht angelegt werden: %w", year, err)
	}
	// Auch ein beim Buchen angelegtes Geschäftsjahr wird protokolliert.
	if err := s.audit(ctx, domain.AuditActionCreate, year, fmt.Sprintf(
		"Geschäftsjahr %d angelegt (%s bis %s%s)", year, derived.StartDate, derived.EndDate, shortSuffix(derived))); err != nil {
		return nil, err
	}
	if err := s.prefillPriorYearRevenue(ctx, derived); err != nil {
		return nil, err
	}
	return derived, nil
}

// carryForwardDate bestimmt das Buchungsdatum des Vortrags.
//
// Der erste Tag des neuen Jahres, außer er ist schon festgeschrieben. Das kommt
// beim Korrekturvortrag vor: bis dahin ist im neuen Jahr weitergebucht und
// vielleicht schon ein Monat festgeschrieben worden. Dann rückt der Vortrag auf
// den ersten offenen Tag — die Alternative wäre, die Festschreibung zu brechen.
func (s *PeriodReader) carryForwardDate(ctx context.Context, fy *domain.FiscalYear) (string, bool, error) {
	date := fy.StartDate
	if s.festschreibungRepo == nil {
		return date, false, nil
	}
	cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, fy.Year)
	if err != nil {
		return "", false, fmt.Errorf("der Festschreibungsstand konnte nicht geprüft werden: %w", err)
	}
	if cutoff == "" || cutoff < date {
		return date, false, nil
	}
	return nextDay(cutoff), true, nil
}

func (s *PeriodService) alignFoundingYear(ctx context.Context) error {
	founded := s.foundingDate(ctx)
	if founded == "" {
		// Ohne Gründungsdatum ist nichts anzugleichen.
		return nil
	}

	year := domain.GetFiscalYearForDate(founded, s.closingCalendar().fiscalYearStartMonth(ctx))
	fy, err := s.fiscalYearRepo.FindByYear(ctx, year)
	if err != nil {
		return err
	}
	if fy == nil {
		// Noch kein Geschäftsjahr angelegt: dann entsteht es später über
		// `derive`, und das kennt die Beurkundung bereits.
		return nil
	}
	// Das Gründungsjahr beginnt mit der Beurkundung, frühestens mit seinem
	// regulären Beginn (`derive`). Ein berichtigtes Gründungsdatum zieht den
	// Beginn deshalb in beide Richtungen nach.
	start := s.derive(ctx, year).StartDate
	if fy.StartDate == start || founded > fy.EndDate {
		return nil
	}

	if fy.IsAdopted() {
		return fmt.Errorf(
			"das Geschäftsjahr %d ist %s. Sein Beginn ist Teil des festgestellten Abschlusses und "+
				"lässt sich nicht mehr auf den Beurkundungstag setzen",
			year, fy.Status.Label())
	}
	if s.festschreibungRepo != nil {
		cutoff, err := s.festschreibungRepo.LatestCutoff(ctx, year)
		if err != nil {
			return fmt.Errorf("der Festschreibungsstand des Geschäftsjahres %d konnte nicht gelesen werden: %w", year, err)
		}
		if cutoff != "" {
			return fmt.Errorf(
				"das Geschäftsjahr %d ist bis zum %s festgeschrieben. Ein festgeschriebener Zeitraum "+
					"behält seinen Beginn",
				year, germanDate(cutoff))
		}
	}
	// Eine Buchung vor dem neuen Beginn wäre nach der Angleichung außerhalb
	// ihres Geschäftsjahres — sie gehörte dann zu keinem Zeitraum mehr.
	before, err := s.journalRepo.FindByBookingDateRange(ctx, year, "", previousDay(start))
	if err != nil {
		return fmt.Errorf("die Buchungen des Geschäftsjahres %d konnten nicht gelesen werden: %w", year, err)
	}
	if len(before) > 0 {
		return fmt.Errorf(
			"im Geschäftsjahr %d stehen %d Buchungen vor dem Beurkundungstag %s. Sie lägen nach der "+
				"Umstellung außerhalb ihres Geschäftsjahres — prüfen Sie, ob das Beurkundungsdatum stimmt",
			year, len(before), germanDate(start))
	}

	previous := fy.StartDate
	snapshot := *fy
	aligned := domain.NewFiscalYear(year, start, fy.EndDate)
	if err := aligned.Validate(); err != nil {
		return err
	}
	// Nur der Zeitraum wandert. Abschlussstand, Arbeitnehmerzahl, Vorjahresumsatz
	// und der Zeitpunkt des Saldenvortrags gehören dem Jahr und nicht seinem
	// Beginn.
	fy.StartDate = aligned.StartDate
	fy.IsShort = aligned.IsShort
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return err
	}
	if err := s.logChange(ctx, domain.AuditActionUpdate, year, fmt.Sprintf(
		"Beginn des Geschäftsjahres %d von %s auf den %s gesetzt (Gründungsdatum %s)%s",
		year, germanDate(previous), germanDate(start), germanDate(founded), shortSuffix(fy)), &snapshot, fy); err != nil {
		return err
	}
	return nil
}

// PeriodOf liefert den Zeitraum für Abschlussbausteine.
func (s *PeriodReader) PeriodOf(ctx context.Context, year int) (*domain.FiscalYear, error) {
	return s.FiscalYear(ctx, year)
}

// FiscalYears liefert die erfassten Geschäftsjahre.
func (s *PeriodReader) FiscalYears(ctx context.Context) ([]domain.FiscalYear, error) {
	return s.fiscalYearRepo.FindAll(ctx)
}

// EarliestFiscalYear liefert das erste Jahr aus Jahresverwaltung und Journal.
func (s *PeriodReader) EarliestFiscalYear(ctx context.Context) (int, error) {
	years, err := s.fiscalYearRepo.FindAll(ctx)
	if err != nil {
		return 0, err
	}
	earliest := 0
	for _, fy := range years {
		if earliest == 0 || fy.Year < earliest {
			earliest = fy.Year
		}
	}
	journalYears, err := s.journalRepo.GetAvailableFiscalYears(ctx)
	if err != nil {
		return earliest, nil
	}
	for _, y := range journalYears {
		if y > 0 && (earliest == 0 || y < earliest) {
			earliest = y
		}
	}
	return earliest, nil
}

// prefillPriorYearRevenue belegt den Vorjahresumsatz eines neu angelegten
// Geschäftsjahres aus der GuV des Vorjahres vor.
//
// Nur wo er noch nicht erfasst ist, und nur wo sich das Vorjahr überhaupt
// auswerten lässt: eine Bilanz, die nicht aufgeht, gibt keine Umsatzerlöse her,
// und das ist kein Grund, das neue Jahr nicht anzulegen. Die Vorbelegung ist
// ein Vorschlag — ohne sie bliebe der Wert null, und null heißt „nicht
// erfasst": die Übergangsregel des § 27 Abs. 38 Nr. 2 UStG käme dann nie zum
// Zug.
func (s *PeriodService) prefillPriorYearRevenue(ctx context.Context, fy *domain.FiscalYear) error {
	if s.revenue == nil || fy == nil || fy.PriorYearRevenue != 0 {
		return nil
	}
	revenue, err := s.revenue.RevenueOf(ctx, fy.Year-1)
	if err != nil || revenue <= 0 {
		return nil
	}
	fy.PriorYearRevenue = revenue
	if err := s.fiscalYearRepo.Save(ctx, fy); err != nil {
		return fmt.Errorf("der Vorjahresumsatz ließ sich nicht vorbelegen: %w", err)
	}
	return s.audit(ctx, domain.AuditActionUpdate, fy.Year, fmt.Sprintf(
		"Vorjahresumsatz des Geschäftsjahres %d mit %s € aus der Gewinn- und Verlustrechnung des Jahres %d "+
			"vorbelegt (§ 27 Abs. 38 Nr. 2 UStG)", fy.Year, revenue, fy.Year-1))
}
