package service

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/buchfink/buchfink/internal/bank"
	"github.com/buchfink/buchfink/internal/domain"
)

func (s *BankService) SetAccountRegistry(registry domain.BankAccountRepository, settings domain.SettingsRepository, tx domain.TxRunner) {
	s.accounts, s.settings, s.txRunner = registry, settings, tx
}
func (s *BankService) Accounts(ctx context.Context) ([]domain.BankAccount, error) {
	if s.accounts == nil {
		return []domain.BankAccount{}, nil
	}
	accounts, err := s.accounts.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	known := map[string]int{}
	for i, account := range accounts {
		known[account.IBAN] = i
	}
	history, err := s.bankRepo.FindAll(ctx, 0)
	if err != nil {
		return nil, err
	}
	for _, tx := range history {
		iban := normalizedIBAN(tx.AccountIBAN)
		if iban == "" || tx.LedgerAccount == "" {
			continue
		}
		if i, exists := known[iban]; exists {
			if accounts[i].LedgerAccount != tx.LedgerAccount {
				return nil, fmt.Errorf("IBAN %s wurde bisher mehreren Buchungskonten zugeordnet; bitte die historischen Importe prüfen", iban)
			}
			continue
		}
		known[iban] = len(accounts)
		accounts = append(accounts, domain.BankAccount{IBAN: iban, Name: "Bankkonto " + tx.LedgerAccount, LedgerAccount: tx.LedgerAccount, Currency: tx.Currency})
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].LedgerAccount < accounts[j].LedgerAccount })
	return accounts, nil
}

type BankImportPreview struct {
	Accounts     []domain.BankAccount `json:"accounts"`
	Transactions int                  `json:"transactions"`
}

func (s *BankService) PreviewFile(ctx context.Context, path string) (*BankImportPreview, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	parsed, err := bank.ParseCAMT053(file)
	if err != nil {
		return nil, err
	}
	known, err := s.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	out := &BankImportPreview{Accounts: []domain.BankAccount{}, Transactions: len(parsed)}
	seen := map[string]bool{}
	for _, tx := range parsed {
		if seen[tx.AccountIBAN] {
			continue
		}
		seen[tx.AccountIBAN] = true
		account := domain.BankAccount{IBAN: tx.AccountIBAN, Currency: tx.Currency}
		for _, k := range known {
			if k.IBAN == account.IBAN {
				account = k
				break
			}
		}
		out.Accounts = append(out.Accounts, account)
	}
	return out, nil
}
func normalizedIBAN(iban string) string {
	return strings.ToUpper(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, iban))
}
func validBankIBAN(iban string) bool {
	if len(iban) < 15 || len(iban) > 34 || iban[0] < 'A' || iban[0] > 'Z' || iban[1] < 'A' || iban[1] > 'Z' || iban[2] < '0' || iban[2] > '9' || iban[3] < '0' || iban[3] > '9' {
		return false
	}
	n := 0
	for _, c := range iban[4:] + iban[:4] {
		switch {
		case c >= '0' && c <= '9':
			n = (n*10 + int(c-'0')) % 97
		case c >= 'A' && c <= 'Z':
			n = (n*100 + int(c-'A') + 10) % 97
		default:
			return false
		}
	}
	return n == 1
}

// ConfigureAccounts keeps every IBAN on one ledger account. Reassigning booked
// history requires an explicit accounting correction, not a settings edit.
func (s *BankService) ConfigureAccounts(ctx context.Context, accounts []domain.BankAccount) error {
	if s.accounts == nil || s.txRunner == nil {
		return fmt.Errorf("Bankkontoeinrichtung ist nicht verfügbar")
	}
	return s.txRunner.RunInTx(ctx, func(ctx context.Context) error {
		known, err := s.accounts.FindAll(ctx)
		if err != nil {
			return err
		}
		byIBAN, byLedger := map[string]domain.BankAccount{}, map[string]string{}
		for _, k := range known {
			byIBAN[k.IBAN] = k
			byLedger[k.LedgerAccount] = k.IBAN
		}
		historical, err := s.bankRepo.FindAll(ctx, 0)
		if err != nil {
			return err
		}
		for _, tx := range historical {
			if tx.LedgerAccount != "" && tx.AccountIBAN != "" {
				if previous := byLedger[tx.LedgerAccount]; previous != "" && previous != tx.AccountIBAN {
					return fmt.Errorf("auf Konto %s wurden bereits Umsätze verschiedener IBANs importiert. Bitte die historischen Zuordnungen vor weiteren Importen prüfen", tx.LedgerAccount)
				}
				byLedger[tx.LedgerAccount] = tx.AccountIBAN
				if previous, ok := byIBAN[tx.AccountIBAN]; ok && previous.LedgerAccount != tx.LedgerAccount {
					return fmt.Errorf("IBAN %s ist in den bisherigen Umsätzen mehreren Buchungskonten zugeordnet; bitte die Zuordnung prüfen", tx.AccountIBAN)
				}
				byIBAN[tx.AccountIBAN] = domain.BankAccount{IBAN: tx.AccountIBAN, LedgerAccount: tx.LedgerAccount, Currency: tx.Currency}
			}
		}
		seen := map[string]bool{}
		for _, input := range accounts {
			input.IBAN = normalizedIBAN(input.IBAN)
			input.Name = strings.TrimSpace(input.Name)
			if !validBankIBAN(input.IBAN) {
				return fmt.Errorf("ungültige IBAN: %s", input.IBAN)
			}
			if input.Name == "" {
				return fmt.Errorf("bitte dem Bankkonto %s einen Namen geben", input.IBAN)
			}
			if seen[input.IBAN] {
				return fmt.Errorf("IBAN %s wurde mehrfach angegeben", input.IBAN)
			}
			seen[input.IBAN] = true
			if input.Currency == "" {
				input.Currency = "EUR"
			}
			if input.Currency != "EUR" {
				return fmt.Errorf("der Bankimport unterstützt derzeit Euro-Konten. Ein Konto in %s benötigt eine gesonderte Währungsumrechnung", input.Currency)
			}
			if input.LedgerAccount != "1800" && input.LedgerAccount != "1810" && input.LedgerAccount != "1820" && input.LedgerAccount != "1830" && input.LedgerAccount != "1840" && input.LedgerAccount != "1850" {
				return fmt.Errorf("bitte für das Bankkonto eines der Konten 1800 bis 1850 wählen")
			}
			if old, ok := byIBAN[input.IBAN]; ok && old.LedgerAccount != input.LedgerAccount {
				return fmt.Errorf("IBAN %s gehört bereits zu Konto %s", input.IBAN, old.LedgerAccount)
			}
			if other := byLedger[input.LedgerAccount]; other != "" && other != input.IBAN {
				return fmt.Errorf("Konto %s gehört bereits zu IBAN %s; bitte ein eigenes Konto wählen", input.LedgerAccount, other)
			}
			if err := s.accounts.Save(ctx, &input); err != nil {
				return err
			}
			byIBAN[input.IBAN], byLedger[input.LedgerAccount] = input, input.IBAN
		}
		// The first account supplies invoice payment details until explicitly changed
		// in company settings. Additional imports never replace that choice.
		if s.settings != nil && len(accounts) > 0 {
			settings, err := s.settings.GetCompanySettings(ctx)
			if err != nil {
				return err
			}
			if settings != nil && strings.TrimSpace(settings.IBAN) == "" {
				settings.IBAN = normalizedIBAN(accounts[0].IBAN)
				settings.BankName = strings.TrimSpace(accounts[0].Name)
				if err := s.settings.UpdateCompanySettings(ctx, settings); err != nil {
					return err
				}
			}
		}
		if s.auditRepo != nil {
			return s.auditRepo.Log(ctx, domain.AuditActionUpdate, "BANK_ACCOUNT", "", fmt.Sprintf("%d Bankkonten eingerichtet oder bestätigt", len(accounts)))
		}
		return nil
	})
}
func (s *BankService) routeTransactions(ctx context.Context, parsed []domain.BankTransaction, ledger string) error {
	startMonth := 1
	if s.settings != nil {
		settings, err := s.settings.GetCompanySettings(ctx)
		if err != nil {
			return err
		}
		if settings != nil && settings.FiscalYearStartMonth > 0 {
			startMonth = settings.FiscalYearStartMonth
		}
	}
	known, err := s.Accounts(ctx)
	if err != nil {
		return err
	}
	routes := map[string]domain.BankAccount{}
	for _, a := range known {
		routes[a.IBAN] = a
	}
	distinct := map[string]bool{}
	for _, tx := range parsed {
		distinct[tx.AccountIBAN] = true
	}
	if s.accounts != nil && ledger != "" && len(distinct) > 1 {
		return fmt.Errorf("dieser Auszug enthält mehrere Bankkonten. Bitte die Konten im Importdialog einzeln zuordnen")
	}
	if s.accounts != nil && len(parsed) > 0 && ledger != "" {
		iban := parsed[0].AccountIBAN
		if existing, ok := routes[iban]; ok && existing.LedgerAccount != ledger {
			return fmt.Errorf("dieser Auszug gehört zu Konto %s, nicht zu %s", existing.LedgerAccount, ledger)
		}
		if _, ok := routes[iban]; !ok {
			account := domain.BankAccount{IBAN: iban, Name: "Geschäftskonto", LedgerAccount: ledger, Currency: parsed[0].Currency}
			if err := s.ConfigureAccounts(ctx, []domain.BankAccount{account}); err != nil {
				return err
			}
			routes[iban] = account
		}
	}
	missing := []string{}
	for i := range parsed {
		tx := &parsed[i]
		if tx.Currency != "EUR" {
			return fmt.Errorf("Umsätze in %s können ohne Währungsumrechnung nicht als Euro gebucht werden", tx.Currency)
		}
		tx.FiscalYear = domain.GetFiscalYearForDate(tx.BookingDate, startMonth)
		if s.accounts == nil {
			if len(distinct) > 1 {
				return fmt.Errorf("mehrere Bankkonten benötigen eine getrennte Kontozuordnung")
			}
			tx.LedgerAccount = ledger
			if ledger == "" {
				tx.LedgerAccount = domain.AccountBank
			}
			continue
		}
		if account, ok := routes[tx.AccountIBAN]; ok {
			tx.LedgerAccount = account.LedgerAccount
		} else {
			missing = append(missing, tx.AccountIBAN)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("bitte das Bankkonto %s zuerst im Importdialog einrichten", missing[0])
	}
	return nil
}
