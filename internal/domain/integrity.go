package domain

// IntegrityCheckResult reports the status of the cryptographic hash chain.
type IntegrityCheckResult struct {
	IsValid          bool   `json:"isValid"`
	TotalEntries     int    `json:"totalEntries"`
	CheckedEntries   int    `json:"checkedEntries"`
	FirstBrokenID    *uint  `json:"firstBrokenId,omitempty"`
	Message          string `json:"message"`
	LastVerifiedHash string `json:"lastVerifiedHash"`
	CheckedAt        string `json:"checkedAt"`
}

// HashChainService defines the contract for computing and validating GoBD hash chains.
type HashChainService interface {
	CalculateHash(entry *BookingEntry, prevHash string) string
	VerifyChain(entries []BookingEntry) IntegrityCheckResult
	// TODO: Add support for external timestamping authority (RFC 3161 TSP)
}

// EBilanzExporter defines the contract for generating official E-Bilanz XBRL instance files.
type EBilanzExporter interface {
	ExportXBRL(settings *CompanySettings, accounts []Account, summary *FinancialSummary) (string, error)
	// TODO: Add support for XBRL Taxonomie 6.8 & Ergänzungsbilanzen / Sonderbilanzen
}
