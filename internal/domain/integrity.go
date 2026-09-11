package domain

// HashChainService defines the contract for computing and validating GoBD hash chains.
type HashChainService interface {
	CalculateHash(entry *JournalEntry, prevHash string) string
	VerifyChain(entries []JournalEntry) IntegrityCheckResult
}

// EBilanzExporter defines the contract for generating E-Bilanz XBRL instance files.
// The implementation uses an unverified taxonomy; export does not imply validity.
type EBilanzExporter interface {
	ExportXBRL(settings *CompanySettings, accounts []Account, summary *FinancialSummary) (string, error)
	// Supplementary and special balance sheets are not supported.
}
