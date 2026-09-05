package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/buchfink/buchfink/internal/bank"
	"github.com/buchfink/buchfink/internal/domain"
)

// BankService imports bank statements and books transactions that carry no
// document of their own (fees, interest, transfers between own accounts).
//
// Payments that settle an open item do not run through here — they go through
// the payment matching, which needs the open item and the difference handling.
type BankService struct {
	bankRepo   domain.BankRepository
	journalSvc *JournalService
	auditRepo  domain.AuditRepository
	receipts   *ReceiptService
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

// ImportCAMT053 parses an ISO 20022 CAMT.053 statement and stores its lines.
func (s *BankService) ImportCAMT053(ctx context.Context, r io.Reader, ledgerAccount string) (int, error) {
	return s.importCAMT053(ctx, r, ledgerAccount, nil)
}

// ImportCAMT053File legt die Datei zuerst als Beleg ab und importiert dann die
// Umsätze daraus.
//
// Die Reihenfolge ist der Punkt: erst der Beleg, dann die Buchungsgrundlage.
// Scheitert das Parsen, liegt die empfangene Datei trotzdem im Archiv — und
// genau das verlangt die Belegsicherung. Der Auszug bekommt die Belegart
// „Kontoauszug" und wird deshalb vom Prüflauf nicht als ungebucht gemeldet.
func (s *BankService) ImportCAMT053File(ctx context.Context, path, ledgerAccount string) (int, error) {
	if path == "" {
		return 0, fmt.Errorf("kein Pfad zur Kontoauszugsdatei angegeben")
	}

	var statementID *uint
	if s.receipts != nil {
		// Derselbe Auszug ein zweites Mal: der Belegspeicher würde die Datei
		// nur einmal ablegen, der Beleg selbst entstünde aber erneut — mit
		// neuer Belegnummer und ohne Umsätze, weil der Import Dubletten
		// abweist. Ein Kontoauszug ist ein Beleg, nicht einer je Importlauf.
		digest, err := fileSHA256(path)
		if err != nil {
			return 0, fmt.Errorf("die Datei %s konnte nicht gelesen werden: %w", filepath.Base(path), err)
		}
		existing, err := s.receipts.FindByOriginalHash(ctx, digest)
		if err != nil {
			return 0, fmt.Errorf("die Belegablage konnte nicht geprüft werden: %w", err)
		}
		if existing != nil {
			id := existing.ID
			statementID = &id
		}
	}
	if s.receipts != nil && statementID == nil {
		receipt, err := s.receipts.File(ctx, FileReceiptRequest{
			Direction:   domain.DirectionIncoming,
			Kind:        domain.ReceiptKindStatement,
			ReceivedAt:  time.Now().Format("2006-01-02"),
			ReceivedVia: domain.ReceivedViaUpload,
			Files:       []NewFile{{Role: domain.ReceiptRoleOriginal, Path: path}},
		})
		if err != nil {
			return 0, fmt.Errorf("der Kontoauszug konnte nicht abgelegt werden: %w", err)
		}
		id := receipt.ID
		statementID = &id
	}

	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("die Datei %s konnte nicht gelesen werden: %w", filepath.Base(path), err)
	}
	defer file.Close()

	return s.importCAMT053(ctx, file, ledgerAccount, statementID)
}

func (s *BankService) importCAMT053(
	ctx context.Context, r io.Reader, ledgerAccount string, statementID *uint,
) (int, error) {
	if ledgerAccount == "" {
		ledgerAccount = domain.AccountBank
	}

	parsed, err := bank.ParseCAMT053(r)
	if err != nil {
		return 0, err
	}

	for i := range parsed {
		parsed[i].FiscalYear = domain.GetFiscalYearForDate(parsed[i].BookingDate, 1)
		parsed[i].LedgerAccount = ledgerAccount
		parsed[i].StatementReceiptID = statementID
	}

	inserted, err := s.bankRepo.CreateBatch(ctx, parsed)
	if err != nil {
		return 0, err
	}

	if s.auditRepo != nil {
		archived := "ohne Ablage der Auszugsdatei"
		if statementID != nil {
			archived = fmt.Sprintf("Auszug als Beleg %d abgelegt", *statementID)
		}
		_ = s.auditRepo.Log(ctx, domain.AuditActionImport, "BANK_TX", fmt.Sprintf("%d", inserted),
			fmt.Sprintf("%d Umsätze aus CAMT.053 für Konto %s importiert (%s)",
				inserted, ledgerAccount, archived))
	}
	return inserted, nil
}

// BookDirect books a bank transaction that has no document behind it against a
// single counter account.
//
// The bank side is taken from the statement, so the direction cannot be entered
// wrongly: money in is a debit on the liquid account, money out a credit.
func (s *BankService) BookDirect(ctx context.Context, bankTxID uint, counterAccount, description string) (*domain.JournalEntry, error) {
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

	return created, nil
}

// Ignore marks a bank transaction as deliberately not booked.
func (s *BankService) Ignore(ctx context.Context, bankTxID uint) error {
	return s.bankRepo.SetMatchStatus(ctx, bankTxID, domain.MatchStatusIgnored)
}

// fileSHA256 liest eine Datei und liefert ihre Prüfsumme als Hex.
func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
