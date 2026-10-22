package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/bank"
	"github.com/buchfink/buchfink/internal/domain"
)

// BankService imports bank statements and books transactions that carry no
// document of their own (fees, interest, transfers between own accounts).
//
// Payments that settle an open item do not run through here — they go through
// the payment matching, which needs the open item and the difference handling.
type BankService struct {
	serviceClock
	bankRepo   domain.BankRepository
	journalSvc *JournalService
	auditRepo  domain.AuditRepository
	receipts   *ReceiptService
	accounts   domain.BankAccountRepository
	settings   domain.SettingsRepository
	txRunner   domain.TxRunner

	// openItems und ruleRepo liefern den Zuordnungsvorschlag (siehe
	// bank_suggest.go). Beide dürfen fehlen: ohne sie importiert und bucht der
	// Dienst wie zuvor, nur ohne Vorschlag.
	openItems LiveOpenItemSource
	ruleRepo  domain.BankRuleRepository
	// creditCards findet die Kartenabrechnung zu einer Abbuchung.
	creditCards CreditCardSource
}

// NewBankService creates the bank import service.
func NewBankService(
	bankRepo domain.BankRepository,
	journalSvc *JournalService,
	auditRepo domain.AuditRepository,
) *BankService {
	return &BankService{bankRepo: bankRepo, journalSvc: journalSvc, auditRepo: auditRepo}
}

// SetReceiptService hängt die Belegablage an.
//
// Ohne sie importiert der Bankimport wie bisher nur die Umsätze. Mit ihr wird
// die CAMT-Datei selbst als Beleg abgelegt — und das ist der eigentliche
// Beleg: die importierte Zeile ist nur, was Buchfink daraus gelesen hat, und
// aufzubewahren ist das empfangene Dokument (§ 147 Abs. 1 Nr. 4 AO, GoBD
// Rz. 130 f.).
func (s *BankService) SetReceiptService(receipts *ReceiptService) { s.receipts = receipts }

// GetTransactions returns imported bank transactions of a fiscal year.
func (s *BankService) GetTransactions(ctx context.Context, fiscalYear int) ([]domain.BankTransaction, error) {
	return s.bankRepo.FindAll(ctx, fiscalYear)
}

// ReadStatementFile liest eine Kontoauszugsdatei.
func ReadStatementFile(path string) ([]byte, string, error) {
	if path == "" {
		return nil, "", fmt.Errorf("kein Pfad zur Kontoauszugsdatei angegeben")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("Kontoauszug lesen: %w", err)
	}
	return data, filepath.Base(path), nil
}

// FileStatement legt einen Kontoauszug als Beleg ab, einmal je Datei, und
// liefert dessen Kennung (nil ohne Belegablage).
func (s *BankService) FileStatement(ctx context.Context, data []byte, filename string) (*uint, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*uint, error) {
		return s.fileStatement(ctx, data, filename)
	})
}

// ImportStatement importiert die Umsätze eines abgelegten Kontoauszugs.
func (s *BankService) ImportStatement(ctx context.Context, data []byte, ledgerAccount string, statementID *uint) (int, error) {
	res, err := s.ImportStatementChecked(ctx, data, ledgerAccount, statementID)
	return res.Inserted, err
}

// StatementImport ist das Ergebnis eines Imports: neu eingelesene Umsätze und
// je Auszug der Datei, ob er geprüft ist (siehe BankTransaction.StatementChecked)
// und, wenn nicht, warum.
type StatementImport struct {
	Inserted  int
	Checked   []bool
	Unchecked []string
}

// ImportStatementChecked ist ImportStatement mit dem Ergebnis der Saldenprüfung.
func (s *BankService) ImportStatementChecked(ctx context.Context, data []byte, ledgerAccount string, statementID *uint) (StatementImport, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (StatementImport, error) {
		return s.importStatements(ctx, bytes.NewReader(data), ledgerAccount, statementID)
	})
}

// fileStatement legt den Kontoauszug als Beleg ab, einmal je Datei.
func (s *BankService) fileStatement(ctx context.Context, data []byte, filename string) (*uint, error) {
	var statementID *uint
	if s.receipts != nil {
		// Derselbe Auszug ein zweites Mal: der Belegspeicher würde die Datei
		// nur einmal ablegen, der Beleg selbst entstünde aber erneut — mit
		// neuer Belegnummer und ohne Umsätze, weil der Import Dubletten
		// abweist. Ein Kontoauszug ist ein Beleg, nicht einer je Importlauf.
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		existing, err := s.receipts.FindByOriginalHash(ctx, digest)
		if err != nil {
			return nil, fmt.Errorf("die Belegablage konnte nicht geprüft werden: %w", err)
		}
		if existing != nil {
			id := existing.ID
			statementID = &id
		}
	}
	if s.receipts != nil && statementID == nil {
		// Das Belegdatum des Kontoauszugs ist der Tag des Imports.
		//
		// Der Auszug selbst nennt einen Zeitraum und nicht einen Tag, und der
		// Zeitraum steht erst nach dem Parsen fest — der Beleg entsteht aber
		// davor, damit die empfangene Datei auch dann im Archiv liegt, wenn das
		// Parsen scheitert. Der Importtag ist die Angabe, die zu diesem
		// Zeitpunkt zutrifft; ohne sie wäre der Auszug ein Beleg ohne Datum und
		// damit nicht zuzuordnen (BEL-02).
		today := s.today()
		receipt, err := s.receipts.File(ctx, FileReceiptRequest{
			Direction:    domain.DirectionIncoming,
			Kind:         domain.ReceiptKindStatement,
			ReceivedAt:   today,
			ReceivedVia:  domain.ReceivedViaUpload,
			DocumentDate: today,
			Subject:      fmt.Sprintf("Kontoauszug %s, importiert am %s", filename, today),
			Files:        []NewFile{{Role: domain.ReceiptRoleOriginal, Content: data, FileName: filename}},
		})
		if err != nil {
			return nil, fmt.Errorf("der Kontoauszug konnte nicht abgelegt werden: %w", err)
		}
		id := receipt.ID
		statementID = &id
	}

	return statementID, nil
}

func (s *BankService) importStatements(
	ctx context.Context, r io.Reader, ledgerAccount string, statementID *uint,
) (StatementImport, error) {
	statements, err := bank.ParseCAMT053Statements(r)
	if err != nil {
		return StatementImport{}, err
	}
	var parsed []domain.BankTransaction
	for _, st := range statements {
		parsed = append(parsed, st.Transactions...)
	}

	if err := s.routeTransactions(ctx, parsed, ledgerAccount); err != nil {
		return StatementImport{}, err
	}
	checked, unchecked, err := s.checkStatements(ctx, statements)
	if err != nil {
		return StatementImport{}, err
	}
	channel := string(actor.RecordedChannel(ctx))
	i := 0
	for si, st := range statements {
		for range st.Transactions {
			parsed[i].StatementReceiptID = statementID
			parsed[i].ImportChannel = channel
			parsed[i].StatementChecked = checked[si]
			i++
		}
	}

	inserted, err := s.bankRepo.CreateBatch(ctx, parsed)
	if err != nil {
		return StatementImport{}, err
	}

	archived := "ohne Ablage der Auszugsdatei"
	if statementID != nil {
		archived = fmt.Sprintf("Auszug als Beleg %d abgelegt", *statementID)
	}
	if err := logAudit(ctx, s.auditRepo, domain.AuditActionImport, "BANK_TX", fmt.Sprintf("%d", inserted),
		fmt.Sprintf("%d Umsätze aus CAMT.053 für Konto %s importiert (%s)",
			inserted, ledgerAccount, archived)); err != nil {
		return StatementImport{}, err
	}
	return StatementImport{Inserted: inserted, Checked: checked, Unchecked: unchecked}, nil
}

// checkStatements prüft jeden Auszug: beide Salden, Umsätze gleich ihrer
// Differenz, Anfangssaldo gleich dem gemerkten Endsaldo des Kontos. Den
// Endsaldo merkt sich das Konto aus einem geprüften Auszug oder aus einem,
// den der Mensch eingelesen hat; ein ungeprüfter Auszug eines Agenten
// verschiebt den Anschluss nicht.
func (s *BankService) checkStatements(ctx context.Context, statements []bank.Statement) ([]bool, []string, error) {
	checked := make([]bool, len(statements))
	unchecked := make([]string, len(statements))
	if s.accounts == nil {
		return checked, unchecked, nil
	}
	known, err := s.accounts.FindAll(ctx)
	if err != nil {
		return nil, nil, err
	}
	accounts := map[string]domain.BankAccount{}
	for _, a := range known {
		accounts[a.IBAN] = a
	}
	human := !actor.RecordedChannel(ctx).IsAgent()
	for i, st := range statements {
		account, ok := accounts[st.IBAN]
		if !ok {
			unchecked[i] = fmt.Sprintf("Das Konto %s ist nicht angelegt.", st.IBAN)
			continue
		}
		if !st.Consistent() {
			unchecked[i] = "Der Auszug nennt keinen Anfangs- und Endsaldo, oder die Umsätze ergeben nicht ihre Differenz."
			continue
		}
		checked[i] = account.ClosingDate != "" && account.ClosingBalance == *st.Opening &&
			st.OpeningDate >= account.ClosingDate
		if !checked[i] {
			unchecked[i] = statementGap(account, st)
		}
		if (!checked[i] && !human) || st.ClosingDate < account.ClosingDate {
			continue
		}
		account.ClosingBalance, account.ClosingDate = *st.Closing, st.ClosingDate
		if err := s.accounts.Save(ctx, &account); err != nil {
			return nil, nil, err
		}
		accounts[st.IBAN] = account
	}
	return checked, unchecked, nil
}

// statementGap sagt, warum ein Auszug mit stimmigen Salden nicht an den
// zuletzt geprüften anschließt.
func statementGap(account domain.BankAccount, st bank.Statement) string {
	switch {
	case account.ClosingDate == "":
		return fmt.Sprintf("Für Konto %s gibt es noch keinen geprüften Saldo.", account.LedgerAccount)
	case st.OpeningDate < account.ClosingDate:
		return fmt.Sprintf("Der Auszug beginnt am %s, vor dem Ende des zuletzt geprüften Auszugs am %s.",
			domain.GermanDate(st.OpeningDate), domain.GermanDate(account.ClosingDate))
	default:
		return fmt.Sprintf("Der Anfangssaldo %s € schließt nicht an den Endsaldo %s € des zuletzt geprüften "+
			"Auszugs vom %s an.", st.Opening.String(), account.ClosingBalance.String(),
			domain.GermanDate(account.ClosingDate))
	}
}

// BookDirect books a bank transaction that has no document behind it against a
// single counter account.
//
// The bank side is taken from the statement, so the direction cannot be entered
// wrongly: money in is a debit on the liquid account, money out a credit.
//
// Buchung, Vermerk am Umsatz, gelernte Regel und Protokoll stehen in einer
// Transaktion: ein gebuchter Umsatz, der weiter als offen gilt, würde beim
// nächsten Versuch ein zweites Mal gebucht.
func (s *BankService) BookDirect(ctx context.Context, bankTxID uint, counterAccount, description string) (*domain.JournalEntry, error) {
	return atomically(ctx, s.journalSvc, func(ctx context.Context) (*domain.JournalEntry, error) {
		return s.bookDirect(ctx, bankTxID, counterAccount, description)
	})
}

func (s *BankService) bookDirect(ctx context.Context, bankTxID uint, counterAccount, description string) (*domain.JournalEntry, error) {
	tx, err := s.bankRepo.FindByID(ctx, bankTxID)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", bankTxID, err)
	}
	if tx.MatchStatus == domain.MatchStatusMatched {
		return nil, fmt.Errorf("Bankumsatz %d ist bereits zugeordnet", bankTxID)
	}
	if counterAccount == "" {
		return nil, fmt.Errorf("Gegenkonto fehlt")
	}
	if tx.Amount == 0 {
		return nil, fmt.Errorf("Bankumsatz %d hat den Betrag null", bankTxID)
	}

	if description == "" {
		description = fmt.Sprintf("%s – %s", tx.CounterpartyName, tx.RemittanceInfo)
	}

	amount := tx.Amount.Abs()
	bankSide, counterSide := domain.SideDebit, domain.SideCredit
	if tx.Amount < 0 {
		bankSide, counterSide = domain.SideCredit, domain.SideDebit
	}

	entry := &domain.JournalEntry{
		FiscalYear:      tx.FiscalYear,
		BookingDate:     tx.BookingDate,
		DocumentDate:    tx.BookingDate,
		ServiceDateFrom: tx.BookingDate,
		ServiceDateTo:   tx.BookingDate,
		ValueDate:       tx.ValueDate,
		Description:     description,
		Source:          domain.EntrySourcePayment,
		BankTxID:        &tx.ID,
		Currency:        tx.Currency,
		Lines: []domain.JournalLine{
			{Position: 1, Side: bankSide, Account: tx.LedgerAccount, Amount: amount},
			{Position: 2, Side: counterSide, Account: counterAccount, Amount: amount},
		},
	}

	created, err := s.journalSvc.Post(ctx, entry)
	if err != nil {
		return nil, err
	}

	if err := s.bankRepo.SetMatchStatus(ctx, bankTxID, domain.MatchStatusMatched); err != nil {
		return nil, fmt.Errorf("Bankumsatz konnte nicht als zugeordnet markiert werden: %w", err)
	}

	// Gelernt wird die bestätigte Zuordnung, und bestätigt ist sie mit der
	// Buchung.
	if err := s.learnRule(ctx, tx, counterAccount); err != nil {
		return nil, err
	}
	return created, nil
}

// Transaction liefert einen Bankumsatz.
func (s *BankService) Transaction(ctx context.Context, bankTxID uint) (*domain.BankTransaction, error) {
	tx, err := s.bankRepo.FindByID(ctx, bankTxID)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", bankTxID, err)
	}
	return tx, nil
}

// IgnoreBecause markiert einen Bankumsatz als nicht zu buchen und hält den
// Grund im Protokoll fest.
func (s *BankService) IgnoreBecause(ctx context.Context, bankTxID uint, reason string) error {
	reason = strings.TrimSpace(reason)
	return s.journalSvc.InTx(ctx, func(ctx context.Context) error {
		tx, err := s.bankRepo.FindByID(ctx, bankTxID)
		if err != nil {
			return fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", bankTxID, err)
		}
		if tx.MatchStatus == domain.MatchStatusMatched {
			return fmt.Errorf("Bankumsatz %d ist bereits gebucht und lässt sich nicht mehr ignorieren", bankTxID)
		}
		if err := s.bankRepo.SetMatchStatus(ctx, bankTxID, domain.MatchStatusIgnored); err != nil {
			return err
		}
		details := fmt.Sprintf("Bankumsatz vom %s über %s € (%s) als nicht zu buchen markiert",
			tx.BookingDate, tx.Amount, tx.CounterpartyName)
		if reason != "" {
			details += ": " + reason
		}
		return logAudit(ctx, s.auditRepo, domain.AuditActionUpdate, "BANK_TX", fmt.Sprintf("%d", bankTxID), details)
	})
}
