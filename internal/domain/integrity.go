// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

package domain

// HashChainService defines the contract for computing and validating GoBD hash chains.
type HashChainService interface {
	CalculateHash(entry *JournalEntry, prevHash string) string
	VerifyChain(entries []JournalEntry) IntegrityCheckResult
}

// EBilanzExporter defines the contract for generating official E-Bilanz XBRL instance files.
type EBilanzExporter interface {
	ExportXBRL(settings *CompanySettings, accounts []Account, summary *FinancialSummary) (string, error)
	// TODO: Add support for XBRL Taxonomie 6.8 & Ergänzungsbilanzen / Sonderbilanzen
}
