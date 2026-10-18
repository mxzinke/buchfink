package service

import (
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/taxrules"
)

// newJournalWithTaxRules verwendet dieselben Kataloge wie core/wire.go.
func newJournalWithTaxRules(journal domain.JournalRepository, accounts domain.AccountRepository,
	contacts domain.ContactRepository, audit domain.AuditRepository, settings domain.SettingsRepository,
	tx domain.TxRunner) *JournalService {
	s := NewJournalService(journal, accounts, contacts, audit, settings, tx)
	s.AddEntryRules(taxrules.EntryRules()...)
	s.AddTaxAccountExceptions(taxrules.TaxAccountExceptions()...)
	return s
}
