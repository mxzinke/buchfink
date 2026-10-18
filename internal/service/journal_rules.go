package service

import (
	"context"

	"github.com/buchfink/buchfink/internal/domain"
)

// AddEntryRules registriert Erweiterungen beim Verdrahten des Dienstes.
// Die Kernregeln und ihre Reihenfolge lassen sich dadurch nicht ersetzen.
func (s *JournalService) AddEntryRules(rules ...domain.EntryRule) {
	for _, rule := range rules {
		if rule.Name == "" || rule.Check == nil || rule.Position > domain.EntryRulesAfterPeriods {
			panic("ungültige Journalregel: " + rule.Name)
		}
		for _, name := range s.EntryRuleNames() {
			if rule.Name == name {
				panic("doppelte Journalregel: " + rule.Name)
			}
		}
		s.ruleExtensions = append(s.ruleExtensions, rule)
	}
}

// AddTaxAccountExceptions registriert Ausnahmen der Kernregel für Steuerkonten.
func (s *JournalService) AddTaxAccountExceptions(exceptions ...domain.TaxAccountException) {
	for _, exception := range exceptions {
		if exception.Name == "" || exception.Allows == nil {
			panic("ungültige Steuerkontenausnahme: " + exception.Name)
		}
		for _, existing := range s.taxAccountExceptions {
			if existing.Name == exception.Name {
				panic("doppelte Steuerkontenausnahme: " + exception.Name)
			}
		}
		s.taxAccountExceptions = append(s.taxAccountExceptions, exception)
	}
}

type entryRule struct {
	name              string
	appliesToReversal bool
	check             func(context.Context, *domain.JournalEntry) error
}

func (s *JournalService) entryRules() []entryRule {
	rules := s.rulesAt(domain.EntryRulesBeforeValidation, -1)
	rules = append(rules, entryRule{"entry", true, func(_ context.Context, e *domain.JournalEntry) error { return e.Validate() }})
	rules = append(rules, s.rulesAt(domain.EntryRulesAfterValidation, -1)...)
	rules = append(rules, entryRule{"accounts", true, s.validateAccounts})
	rules = append(rules, s.rulesAt(domain.EntryRulesAfterAccounts, -1)...)
	rules = append(rules, s.rulesAt(domain.EntryRulesManual, -1)...)
	// Die Feststellung nennt den Weg zurück und geht der Festschreibung vor.
	rules = append(rules, entryRule{"year_not_adopted", true, s.ensureYearNotAdopted},
		entryRule{"period_open", true, s.ensurePeriodOpen})
	return append(rules, s.rulesAt(domain.EntryRulesAfterPeriods, -1)...)
}

func (s *JournalService) rulesAt(position domain.EntryRulePosition, line int) []entryRule {
	var rules []entryRule
	view := journalRuleView{s.settingsRepo, s.receiptRepo, line}
	for _, rule := range s.ruleExtensions {
		if rule.Position == position {
			rules = append(rules, entryRule{rule.Name, rule.AppliesToReversal,
				func(ctx context.Context, e *domain.JournalEntry) error { return rule.Check(ctx, e, view) }})
		}
	}
	return rules
}

func (s *JournalService) checkRulesAt(ctx context.Context, position domain.EntryRulePosition, e *domain.JournalEntry, line int) error {
	return checkEntryRules(ctx, e, s.rulesAt(position, line))
}

// check hält jede Regel, die eine Buchung vor dem Schreiben erfüllen muss.
// Post und ValidatePostable prüfen dieselbe Liste; der erste Fehler entscheidet.
func (s *JournalService) check(ctx context.Context, e *domain.JournalEntry) error {
	return checkEntryRules(ctx, e, s.entryRules())
}

func checkEntryRules(ctx context.Context, e *domain.JournalEntry, rules []entryRule) error {
	for _, rule := range rules {
		if e.Kind == domain.EntryKindReversal && !rule.appliesToReversal {
			continue
		}
		if err := rule.check(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

// EntryRuleNames nennt die Prüfungsreihenfolge. EntryRulesAfterAccount läuft
// innerhalb von accounts nach jeder einzelnen Sachkontozeile.
func (s *JournalService) EntryRuleNames() []string {
	var names []string
	for _, rule := range s.entryRules() {
		names = append(names, rule.name)
		if rule.name == "accounts" {
			for _, lineRule := range s.rulesAt(domain.EntryRulesAfterAccount, -1) {
				names = append(names, lineRule.name)
			}
		}
	}
	return names
}

func (s *JournalService) TaxAccountExceptionNames() []string {
	names := make([]string, 0, len(s.taxAccountExceptions))
	for _, exception := range s.taxAccountExceptions {
		names = append(names, exception.Name)
	}
	return names
}

func (s *JournalService) mayWriteTaxAccount(e *domain.JournalEntry, account string) bool {
	for _, exception := range s.taxAccountExceptions {
		if e.Kind == domain.EntryKindReversal && !exception.AppliesToReversal {
			continue
		}
		if exception.Allows(e, account) {
			return true
		}
	}
	return false
}

type journalRuleView struct {
	settings domain.SettingsRepository
	receipts domain.ReceiptRepository
	line     int
}

func (v journalRuleView) TaxationType(ctx context.Context) string {
	if v.settings == nil {
		return ""
	}
	cfg, err := v.settings.GetCompanySettings(ctx)
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.TaxationType
}

func (v journalRuleView) Receipt(ctx context.Context, id uint) (*domain.Receipt, error) {
	if v.receipts == nil {
		return nil, nil
	}
	return v.receipts.FindByID(ctx, id)
}

func (v journalRuleView) AccountLine() int { return v.line }
