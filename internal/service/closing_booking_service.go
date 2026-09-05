package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// ClosingBookingService führt die drei Abschlussbausteine, die keine eigene
// Kartei haben: den Inventurwert der Vorräte, die Umsatzsteuer-Verrechnung und
// die Steuerrückstellung.
//
// Sie stehen zusammen, weil sie dasselbe Muster teilen: sie lesen Kontensalden,
// rechnen daraus einen Betrag, zeigen ihn als Vorschau und buchen ihn erst auf
// Freigabe. Was sie unterscheidet, ist nur die Rechnung dazwischen.
type ClosingBookingService struct {
	inventoryRepo domain.InventoryRepository
	provisionRepo domain.ProvisionRepository
	journalRepo   domain.JournalRepository
	journalSvc    *JournalService
	settingsRepo  domain.SettingsRepository
	auditRepo     domain.AuditRepository
	closingSvc    *ClosingService
	receipts      closingReceiptFiler
	fiscalYear    int
}

// NewClosingBookingService wires die drei Bausteine.
func NewClosingBookingService(
	inventoryRepo domain.InventoryRepository,
	provisionRepo domain.ProvisionRepository,
	journalRepo domain.JournalRepository,
	journalSvc *JournalService,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	closingSvc *ClosingService,
	fiscalYear int,
) *ClosingBookingService {
	return &ClosingBookingService{
		inventoryRepo: inventoryRepo, provisionRepo: provisionRepo, journalRepo: journalRepo,
		journalSvc: journalSvc, settingsRepo: settingsRepo, auditRepo: auditRepo,
		closingSvc: closingSvc, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *ClosingBookingService) SetFiscalYear(year int) { s.fiscalYear = year }

// SetReceiptService gibt dem Dienst den Belegspeicher für die Eigenbelege.
func (s *ClosingBookingService) SetReceiptService(r closingReceiptFiler) { s.receipts = r }

// -------------------------------------------------------------------------
// Vorräte (§ 240 HGB)
// -------------------------------------------------------------------------

// InventoryAccount ist ein Vorratskonto mit Buchwert und erfasstem Inventurwert.
type InventoryAccount struct {
	Account     string `json:"account"`
	AccountName string `json:"accountName"`
	Group       string `json:"group"`
	// ChangeAccount ist das Gegenkonto der Bestandsveränderung.
	ChangeAccount     string       `json:"changeAccount"`
	ChangeAccountName string       `json:"changeAccountName"`
	BookValue         domain.Cents `json:"bookValue"`
	// Counted ist der erfasste Inventurwert; CountedAt der Tag der Aufnahme.
	Counted   domain.Cents `json:"counted"`
	CountedAt string       `json:"countedAt,omitempty"`
	Booked    bool         `json:"booked"`
}

// InventoryOverview ist der Baustein „Vorräte".
type InventoryOverview struct {
	FiscalYear int                `json:"fiscalYear"`
	Cutoff     string             `json:"cutoff"`
	Accounts   []InventoryAccount `json:"accounts"`
	Note       string             `json:"note"`
}

// InventoryAccounts listet die Vorratskonten mit Buchwert und Erfassung.
func (s *ClosingBookingService) InventoryAccounts(ctx context.Context, year int) (*InventoryOverview, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, fy.EndDate)
	if err != nil {
		return nil, err
	}
	counts, err := s.inventoryRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	byAccount := map[string]domain.InventoryCount{}
	for _, c := range counts {
		byAccount[c.Account] = c
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	overview := &InventoryOverview{
		FiscalYear: year, Cutoff: fy.EndDate, Accounts: make([]InventoryAccount, 0),
		Note: "Buchfink bewertet die Vorräte nicht. Verbrauchsfolgeverfahren (§ 256 HGB) und das " +
			"Niederstwertprinzip (§ 253 Abs. 4 HGB) entscheiden sich am einzelnen Gegenstand; der " +
			"erfasste Inventurwert ist deshalb der bewertete Wert und muss eine Abwertung schon " +
			"enthalten.",
	}

	// Konten mit Bestand und Konten mit Erfassung — die Vereinigung, weil ein
	// erstmals aufgenommener Bestand noch keinen Buchwert hat.
	accounts := map[string]bool{}
	for account := range turnovers {
		if accounting.IsInventoryAccount(account) {
			accounts[account] = true
		}
	}
	for account := range byAccount {
		accounts[account] = true
	}

	for account := range accounts {
		turnover := turnovers[account]
		change, err := accounting.InventoryChangeAccount(account)
		if err != nil {
			continue
		}
		row := InventoryAccount{
			Account: account, AccountName: chart.Name(account),
			Group:         accounting.InventoryGroupLabel(account),
			ChangeAccount: change, ChangeAccountName: chart.Name(change),
			BookValue: turnover.Debit - turnover.Credit,
		}
		if count, ok := byAccount[account]; ok {
			row.Counted = count.Amount
			row.CountedAt = count.CountedOn
			row.Booked = count.JournalEntryID != nil
		}
		overview.Accounts = append(overview.Accounts, row)
	}
	sort.Slice(overview.Accounts, func(i, j int) bool {
		return overview.Accounts[i].Account < overview.Accounts[j].Account
	})
	return overview, nil
}

// InventoryRequest ist die Erfassung eines Inventurwertes.
type InventoryRequest struct {
	FiscalYear int          `json:"fiscalYear"`
	Account    string       `json:"account"`
	Amount     domain.Cents `json:"amount"`
	CountedOn  string       `json:"countedOn"`
	Method     string       `json:"method"`
	// ReceiptID verweist auf die Inventurliste im Belegspeicher — Pflicht.
	ReceiptID uint `json:"receiptId"`
}

// InventoryPreview ist die Bestandsveränderung, die aus dem Inventurwert folgt.
type InventoryPreview struct {
	Account       string               `json:"account"`
	AccountName   string               `json:"accountName"`
	ChangeAccount string               `json:"changeAccount"`
	BookValue     domain.Cents         `json:"bookValue"`
	Counted       domain.Cents         `json:"counted"`
	Change        domain.Cents         `json:"change"`
	Lines         []domain.JournalLine `json:"lines"`
	BookingDate   string               `json:"bookingDate"`
	Explanation   string               `json:"explanation"`
}

// PreviewInventory rechnet die Bestandsveränderung.
func (s *ClosingBookingService) PreviewInventory(
	ctx context.Context, req InventoryRequest,
) (*InventoryPreview, error) {
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	change, err := accounting.InventoryChangeAccount(req.Account)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, fy.EndDate)
	if err != nil {
		return nil, err
	}
	turnover := turnovers[req.Account]
	bookValue := turnover.Debit - turnover.Credit
	delta := req.Amount - bookValue

	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}
	preview := &InventoryPreview{
		Account: req.Account, AccountName: chart.Name(req.Account), ChangeAccount: change,
		BookValue: bookValue, Counted: req.Amount, Change: delta, BookingDate: fy.EndDate,
		Lines: make([]domain.JournalLine, 0, 2),
	}
	switch {
	case delta > 0:
		// Bestandserhöhung: der Bestand wächst, der Aufwand des Jahres sinkt
		// bzw. der Ertrag steigt.
		preview.Lines = append(preview.Lines,
			domain.JournalLine{Side: domain.SideDebit, Account: req.Account, Amount: delta, Text: "Bestandserhöhung"},
			domain.JournalLine{Side: domain.SideCredit, Account: change, Amount: delta, Text: "Bestandserhöhung"},
		)
		preview.Explanation = fmt.Sprintf(
			"Der Inventurwert von %s € liegt um %s € über dem Buchwert von %s €. Die Erhöhung wird "+
				"auf %s gebucht.", req.Amount, delta, bookValue, change)
	case delta < 0:
		preview.Lines = append(preview.Lines,
			domain.JournalLine{Side: domain.SideDebit, Account: change, Amount: -delta, Text: "Bestandsminderung"},
			domain.JournalLine{Side: domain.SideCredit, Account: req.Account, Amount: -delta, Text: "Bestandsminderung"},
		)
		preview.Explanation = fmt.Sprintf(
			"Der Inventurwert von %s € liegt um %s € unter dem Buchwert von %s €. Die Minderung wird "+
				"auf %s gebucht.", req.Amount, -delta, bookValue, change)
	default:
		preview.Explanation = fmt.Sprintf(
			"Der Inventurwert stimmt mit dem Buchwert von %s € überein; es ist nichts zu buchen.", bookValue)
	}
	return preview, nil
}

// inventoryVoucher ist die Rechnung im Eigenbeleg der Bestandsveränderung: der
// Buchwert, der aufgenommene Wert und die Differenz — die drei Zahlen, aus denen
// die Buchung entstanden ist.
type inventoryVoucher struct {
	Account       string       `json:"konto"`
	AccountName   string       `json:"kontobezeichnung"`
	ChangeAccount string       `json:"gegenkonto"`
	BookValue     domain.Cents `json:"buchwert"`
	Counted       domain.Cents `json:"inventurwert"`
	Change        domain.Cents `json:"differenz"`
	CountedOn     string       `json:"aufgenommen_am"`
	Method        string       `json:"verfahren"`
	// CountSheet ist die Belegnummer der Inventurliste. Sie steht im Eigenbeleg,
	// damit die Rechnung auch außerhalb von Buchfink auf ihren Nachweis zeigt.
	CountSheet string `json:"inventurliste,omitempty"`
}

// receiptNumberOf ist die Belegnummer eines Belegs, oder leer.
func receiptNumberOf(receipt *domain.Receipt) string {
	if receipt == nil {
		return ""
	}
	return receipt.ReceiptNumber
}

// BookInventory erfasst den Inventurwert und bucht die Bestandsveränderung.
func (s *ClosingBookingService) BookInventory(
	ctx context.Context, req InventoryRequest,
) (*domain.InventoryCount, error) {
	preview, err := s.PreviewInventory(ctx, req)
	if err != nil {
		return nil, err
	}
	year := req.FiscalYear
	if year == 0 {
		year = s.fiscalYear
	}
	count := &domain.InventoryCount{
		FiscalYear: year, Account: req.Account, Amount: req.Amount,
		BookValue: preview.BookValue, CountedOn: req.CountedOn, Method: req.Method,
	}
	if req.ReceiptID != 0 {
		id := req.ReceiptID
		count.ReceiptID = &id
	}
	if err := count.Validate(); err != nil {
		return nil, err
	}
	// Die Inventurliste muss es geben. Ohne diese Abfrage genügte jede Zahl
	// ungleich null, und die Pflicht des § 240 HGB wäre eine Eingabemaske.
	var sheet *domain.Receipt
	if s.receipts != nil && count.ReceiptID != nil {
		sheet, err = s.receipts.Get(ctx, *count.ReceiptID)
		if err != nil {
			return nil, fmt.Errorf(
				"die Inventurliste zum Beleg %d wurde im Belegspeicher nicht gefunden: %w",
				*count.ReceiptID, err)
		}
		// Eine Inventurliste gehört zu einer Aufnahme. Hängt sie schon an einer
		// anderen, noch stehenden Buchung, ist es die Liste eines anderen
		// Kontos oder Jahres — und die zweite Aufnahme braucht ihre eigene.
		// Nach einem Storno bleibt sie dagegen an der zurückgenommenen Buchung
		// hängen; das ist kein Hindernis, sondern die Spur dorthin.
		if sheet != nil && sheet.JournalEntryID != nil {
			voided, err := entryIsReversed(ctx, s.journalRepo, sheet.JournalEntryID)
			if err != nil {
				return nil, err
			}
			if !voided {
				return nil, fmt.Errorf(
					"die Inventurliste %s gehört bereits zur Buchung %d. Eine zweite Aufnahme "+
						"braucht ihre eigene Liste", sheet.ReceiptNumber, *sheet.JournalEntryID)
			}
		}
	}

	// Ein zweiter Inventurwert für dasselbe Konto und Jahr: ist der erste
	// gebucht, läuft die Korrektur über den Storno seiner Buchung — sonst
	// stünden zwei Aufnahmen nebeneinander, und keine Auswertung könnte sagen,
	// welche gilt. Ist er nicht gebucht (weil er dem Buchwert entsprach), wird
	// er fortgeschrieben.
	existing, err := s.inventoryRepo.FindByYear(ctx, year)
	if err != nil {
		return nil, err
	}
	for i := range existing {
		if existing[i].Account != req.Account {
			continue
		}
		// Ist die Buchung storniert, ist die Sperre aufgehoben: der Weg, den die
		// Meldung nennt, muss auch ans Ziel führen. Der Karteisatz wird dann
		// fortgeschrieben und bekommt die neue Buchung.
		voided, err := entryIsReversed(ctx, s.journalRepo, existing[i].JournalEntryID)
		if err != nil {
			return nil, err
		}
		if existing[i].JournalEntryID != nil && !voided {
			return nil, fmt.Errorf(
				"für das Konto %s ist im Geschäftsjahr %d bereits ein Inventurwert von %s € erfasst "+
					"und gebucht. Eine Korrektur läuft über den Storno dieser Buchung",
				req.Account, year, existing[i].Amount)
		}
		count.ID = existing[i].ID
	}

	if len(preview.Lines) > 0 {
		entry := &domain.JournalEntry{
			BookingDate: preview.BookingDate, DocumentDate: preview.BookingDate,
			ServiceDateFrom: preview.BookingDate, ServiceDateTo: preview.BookingDate,
			Description: fmt.Sprintf("Bestandsveränderung %s zum Inventurwert %s €",
				req.Account, req.Amount),
			Source:             domain.EntrySourceClosing,
			DocumentNumber:     fmt.Sprintf("INV %d-%s", year, req.Account),
			PostingRuleVersion: accounting.PostingRuleVersion,
			Lines:              preview.Lines,
		}
		// Auch die Bestandsveränderung braucht ihren Beleg (GoBD Rz. 61). Der
		// Eigenbeleg trägt die Rechnung — Buchwert, Inventurwert, Differenz —
		// und nennt die Inventurliste, die daneben im Belegspeicher liegt.
		receipt, err := selfIssuedVoucher(ctx, s.receipts, year, closingVoucher{
			Kind: "inventurwert", FiscalYear: year, Date: preview.BookingDate,
			Description: entry.Description, Explanation: preview.Explanation,
			Calculation: inventoryVoucher{
				Account: req.Account, AccountName: preview.AccountName,
				ChangeAccount: preview.ChangeAccount, BookValue: preview.BookValue,
				Counted: req.Amount, Change: preview.Change,
				CountedOn: req.CountedOn, Method: req.Method,
				CountSheet: receiptNumberOf(sheet),
			},
			Lines: preview.Lines,
		})
		if err != nil {
			return nil, err
		}
		reference := entry.DocumentNumber
		attachVoucher(entry, receipt)
		entry.DocumentNumber = reference

		created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
		if err != nil {
			return nil, err
		}
		count.JournalEntryID = &created.ID
		// Die Inventurliste ist der äußere Beleg der Aufnahme (§ 240 HGB). Ohne
		// die Versiegelung bliebe sie als loser, offener Beleg im Belegspeicher
		// liegen — die Buchung wäre da, der Nachweis unverbunden daneben.
		//
		// Hängt sie schon an der stornierten Vorgängerbuchung, bleibt sie dort:
		// eine Versiegelung ist keine Zuordnung, die man verschiebt, und der
		// Weg von der neuen Buchung zur Liste führt über den Eigenbeleg, der
		// ihre Belegnummer trägt.
		if s.receipts != nil && count.ReceiptID != nil && (sheet == nil || sheet.JournalEntryID == nil) {
			if err := s.receipts.Seal(ctx, *count.ReceiptID, created.ID); err != nil {
				return nil, fmt.Errorf(
					"die Buchung %s wurde geschrieben, die Inventurliste aber nicht mit ihr "+
						"verbunden: %w", created.EntryNumber, err)
			}
		}
	}
	if err := s.inventoryRepo.Save(ctx, count); err != nil {
		return nil, fmt.Errorf("der Inventurwert konnte nicht gespeichert werden: %w", err)
	}
	s.audit(ctx, "INVENTORY", year, fmt.Sprintf(
		"Inventurwert %s zum %s: %s € (Buchwert %s €, Veränderung %s €, Verfahren %s)",
		req.Account, req.CountedOn, req.Amount, preview.BookValue, preview.Change, req.Method))
	return count, nil
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Verrechnung
// -------------------------------------------------------------------------

// VatSettlementReference ist die Belegnummer der Verrechnungsbuchung. Über sie
// erkennt der Assistent, ob der Schritt getan ist.
func VatSettlementReference(year int) string { return fmt.Sprintf("USTV %d", year) }

// VatSettlementRow ist ein Steuerkonto mit seinem Jahressaldo.
type VatSettlementRow struct {
	Account     string `json:"account"`
	AccountName string `json:"accountName"`
	// Balance ist der Saldo in Soll-Richtung: Vorsteuer positiv, Umsatzsteuer
	// negativ.
	Balance domain.Cents `json:"balance"`
}

// VatSettlement ist die Vorschau der Jahresverrechnung.
type VatSettlement struct {
	FiscalYear int                `json:"fiscalYear"`
	Cutoff     string             `json:"cutoff"`
	Rows       []VatSettlementRow `json:"rows"`
	InputTax   domain.Cents       `json:"inputTax"`
	OutputTax  domain.Cents       `json:"outputTax"`
	Prepaid    domain.Cents       `json:"prepaid"`
	// Payable ist die Zahllast (auf 3841), Refund die Erstattung (auf 1420).
	// Immer nur eines von beiden.
	Payable     domain.Cents         `json:"payable"`
	Refund      domain.Cents         `json:"refund"`
	Lines       []domain.JournalLine `json:"lines"`
	BookingDate string               `json:"bookingDate"`
	Explanation string               `json:"explanation"`
}

// settlementAccounts sind die Konten, die die Jahresverrechnung anfasst: die
// Steuerautomatik-Konten plus die Vorauszahlungskonten.
func settlementAccounts() []string {
	return []string{
		domain.AccountVorsteuer, domain.AccountVorsteuer7, domain.AccountVorsteuer19,
		domain.AccountVorsteuerIG, domain.AccountVorsteuerIG19,
		domain.AccountVorsteuer13b, domain.AccountVorsteuer13b19,
		domain.AccountUmsatzsteuer, domain.AccountUmsatzsteuer7, domain.AccountUmsatzsteuer19,
		domain.AccountUmsatzsteuerIG, domain.AccountUmsatzsteuerIG19,
		domain.AccountUmsatzsteuer13b, domain.AccountUmsatzsteuer13b19,
		domain.AccountUmsatzsteuer14c,
		domain.AccountUmsatzsteuerVorauszahlungen, domain.AccountSondervorauszahlung,
	}
}

// PreviewVatSettlement rechnet die Jahresverrechnung der Umsatzsteuer.
//
// Am Bilanzstichtag stehen auf den Steuerkonten die Summen des ganzen Jahres:
// die abziehbare Vorsteuer im Soll, die geschuldete Umsatzsteuer im Haben, die
// geleisteten Vorauszahlungen im Soll. Was übrigbleibt, ist die Zahllast oder
// die Erstattung der Jahreserklärung — und genau sie gehört in die Bilanz, nicht
// die drei Salden nebeneinander.
func (s *ClosingBookingService) PreviewVatSettlement(ctx context.Context, year int) (*VatSettlement, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, fy.EndDate)
	if err != nil {
		return nil, err
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	out := &VatSettlement{
		FiscalYear: year, Cutoff: fy.EndDate, BookingDate: fy.EndDate,
		Rows: make([]VatSettlementRow, 0), Lines: make([]domain.JournalLine, 0),
	}
	var total domain.Cents
	for _, account := range settlementAccounts() {
		turnover, ok := turnovers[account]
		if !ok {
			continue
		}
		balance := turnover.Debit - turnover.Credit
		if balance == 0 {
			continue
		}
		out.Rows = append(out.Rows, VatSettlementRow{
			Account: account, AccountName: chart.Name(account), Balance: balance,
		})
		switch account {
		case domain.AccountUmsatzsteuerVorauszahlungen, domain.AccountSondervorauszahlung:
			out.Prepaid += balance
		default:
			if balance > 0 {
				out.InputTax += balance
			} else {
				out.OutputTax += -balance
			}
		}
		total += balance
		// Jedes Konto wird auf null gestellt: die Gegenseite seines Saldos.
		side := domain.SideCredit
		amount := balance
		if balance < 0 {
			side, amount = domain.SideDebit, -balance
		}
		out.Lines = append(out.Lines, domain.JournalLine{
			Side: side, Account: account, Amount: amount,
			Text: fmt.Sprintf("Jahresverrechnung %d", year),
		})
	}

	switch {
	case total < 0:
		out.Payable = -total
		out.Lines = append(out.Lines, domain.JournalLine{
			Side: domain.SideCredit, Account: domain.AccountUmsatzsteuerVorjahr,
			Amount: out.Payable, Text: fmt.Sprintf("Umsatzsteuer-Zahllast %d", year),
		})
		out.Explanation = fmt.Sprintf(
			"Umsatzsteuer %s € abzüglich Vorsteuer %s € und Vorauszahlungen %s € ergibt eine Zahllast "+
				"von %s €. Sie steht als Verbindlichkeit auf %s.",
			out.OutputTax, out.InputTax, out.Prepaid, out.Payable, domain.AccountUmsatzsteuerVorjahr)
	case total > 0:
		out.Refund = total
		out.Lines = append(out.Lines, domain.JournalLine{
			Side: domain.SideDebit, Account: domain.AccountUmsatzsteuerforderung,
			Amount: out.Refund, Text: fmt.Sprintf("Umsatzsteuer-Erstattung %d", year),
		})
		out.Explanation = fmt.Sprintf(
			"Vorsteuer %s € und Vorauszahlungen %s € übersteigen die Umsatzsteuer von %s € um %s €. "+
				"Der Betrag steht als Forderung auf %s.",
			out.InputTax, out.Prepaid, out.OutputTax, out.Refund, domain.AccountUmsatzsteuerforderung)
	default:
		out.Explanation = "Die Steuerkonten gleichen sich aus; es bleibt weder Zahllast noch Erstattung."
	}
	return out, nil
}

// BookVatSettlement bucht die Jahresverrechnung.
func (s *ClosingBookingService) BookVatSettlement(
	ctx context.Context, year int,
) (*domain.JournalEntry, error) {
	settlement, err := s.PreviewVatSettlement(ctx, year)
	if err != nil {
		return nil, err
	}
	if len(settlement.Lines) == 0 {
		return nil, fmt.Errorf(
			"im Geschäftsjahr %d steht auf keinem Steuerkonto ein Saldo; es gibt nichts zu verrechnen",
			settlement.FiscalYear)
	}
	entry := &domain.JournalEntry{
		BookingDate: settlement.BookingDate, DocumentDate: settlement.BookingDate,
		ServiceDateFrom: settlement.BookingDate, ServiceDateTo: settlement.BookingDate,
		Description:        fmt.Sprintf("Umsatzsteuer-Jahresverrechnung %d", settlement.FiscalYear),
		Source:             domain.EntrySourceClosing,
		DocumentNumber:     VatSettlementReference(settlement.FiscalYear),
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              settlement.Lines,
	}
	receipt, err := selfIssuedVoucher(ctx, s.receipts, settlement.FiscalYear, closingVoucher{
		Kind: "umsatzsteuer-verrechnung", FiscalYear: settlement.FiscalYear,
		Date: settlement.BookingDate, Description: entry.Description,
		Explanation: settlement.Explanation, Calculation: settlement, Lines: settlement.Lines,
	})
	if err != nil {
		return nil, err
	}
	// Die Belegnummer der Verrechnung bleibt die Kennung des Schrittes; der
	// Eigenbeleg hängt daneben.
	reference := entry.DocumentNumber
	attachVoucher(entry, receipt)
	entry.DocumentNumber = reference

	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, "VAT_SETTLEMENT", settlement.FiscalYear, fmt.Sprintf(
		"Umsatzsteuer-Jahresverrechnung %d: Zahllast %s €, Erstattung %s €",
		settlement.FiscalYear, settlement.Payable, settlement.Refund))
	return created, nil
}

// -------------------------------------------------------------------------
// Steuerrückstellung
// -------------------------------------------------------------------------

// TaxProvisionPreview ist der Vorschlag für die Steuerrückstellung.
type TaxProvisionPreview struct {
	FiscalYear int    `json:"fiscalYear"`
	Cutoff     string `json:"cutoff"`

	accounting.TaxProvisionResult
	Input accounting.TaxProvisionInput `json:"input"`

	Lines       []domain.JournalLine `json:"lines"`
	Explanation string               `json:"explanation"`
	// Warning sagt ausdrücklich, dass die Rechnung eine Schätzung ist.
	Warning string `json:"warning"`
}

// PreviewTaxProvision rechnet Körperschaftsteuer, Solidaritätszuschlag und
// Gewerbesteuer auf das Ergebnis des Jahres.
func (s *ClosingBookingService) PreviewTaxProvision(
	ctx context.Context, year int,
) (*TaxProvisionPreview, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	fy, err := s.closingSvc.PeriodOf(ctx, year)
	if err != nil {
		return nil, err
	}
	turnovers, err := s.journalRepo.AccountTurnoversUntil(ctx, year, fy.EndDate)
	if err != nil {
		return nil, err
	}
	chart, err := s.journalSvc.Chart(ctx)
	if err != nil {
		return nil, err
	}

	nonDeductibleAccounts := map[string]bool{}
	for _, dir := range []domain.Direction{domain.DirectionIncoming, domain.DirectionOutgoing} {
		for _, group := range accounting.PostingGroups(dir) {
			if group.NonDeductibleAccount != "" {
				nonDeductibleAccounts[group.NonDeductibleAccount] = true
			}
		}
	}

	prepaidCorporate, prepaidTrade, err := s.taxPrepayments(ctx, year)
	if err != nil {
		return nil, err
	}

	var profit, nonDeductible domain.Cents
	for account, turnover := range turnovers {
		acc, ok := chart.Lookup(account)
		if !ok {
			continue
		}
		balance := turnover.Debit - turnover.Credit
		if isIncomeTaxAccount(account) {
			// Die Ertragsteuern selbst gehören nicht in ihre eigene
			// Bemessungsgrundlage — weder als Aufwand noch als Vorauszahlung.
			// Was auf ihnen steht, ist beides zugleich; getrennt wird das über
			// die Herkunft der Buchung, nicht über den Saldo.
			continue
		}
		switch acc.Type {
		case domain.AccountTypeRevenue:
			profit += -balance
		case domain.AccountTypeExpense:
			profit += -balance
		}
		if nonDeductibleAccounts[account] {
			nonDeductible += balance
		}
	}

	input := accounting.TaxProvisionInput{
		ProfitBeforeTax: profit, NonDeductible: nonDeductible,
		TradeTaxRatePercent: s.tradeTaxRate(ctx),
		PrepaidCorporate:    prepaidCorporate, PrepaidTrade: prepaidTrade,
		Date: fy.EndDate,
	}
	result, err := accounting.CalculateTaxProvision(input)
	if err != nil {
		return nil, err
	}

	preview := &TaxProvisionPreview{
		FiscalYear: year, Cutoff: fy.EndDate, TaxProvisionResult: *result, Input: input,
		Lines: make([]domain.JournalLine, 0, 4),
		Warning: "Das ist eine Schätzung und keine Steuererklärung. Verlustvorträge (§ 10d EStG, " +
			"§ 10a GewStG), gewerbesteuerliche Hinzurechnungen und Kürzungen (§§ 8 und 9 GewStG) und " +
			"verdeckte Gewinnausschüttungen kennt Buchfink nicht. Der Betrag ist änderbar.",
	}
	preview.Lines = taxProvisionLines(result)
	preview.Explanation = fmt.Sprintf(
		"Ergebnis vor Steuern %s € zuzüglich nicht abziehbarer Betriebsausgaben %s € ergibt %s €. "+
			"Darauf %s: Körperschaftsteuer %s €, Solidaritätszuschlag %s €, Gewerbesteuer %s € "+
			"(Gewerbeertrag %s €, Messbetrag %s €). Abzüglich Vorauszahlungen bleiben %s € "+
			"Körperschaftsteuer- und %s € Gewerbesteuerrückstellung.",
		profit, nonDeductible, result.TaxableIncome, result.RatesUsed,
		result.CorporateTax, result.Solidarity, result.TradeTax,
		result.TradeIncome, result.TradeBase, result.IncomeProvision, result.TradeProvision)
	return preview, nil
}

// taxPrepayments sind die geleisteten Steuervorauszahlungen des Jahres.
//
// Gelesen wird die Herkunft der Buchung und nicht der Kontensaldo. Auf 7600 ff.
// steht am Jahresende beides: die Vorauszahlungen, die das Finanzamt eingezogen
// hat, und der Aufwand aus der Steuerrückstellung, die dieser Baustein selbst
// gebucht hat. Über den Saldo gerechnet mindert die Rückstellung sich selbst —
// und ein zweiter Lauf käme auf null, ohne dass jemand es merkt.
func (s *ClosingBookingService) taxPrepayments(
	ctx context.Context, year int,
) (corporate, trade domain.Cents, err error) {
	entries, err := s.journalRepo.FindAll(ctx, year)
	if err != nil {
		return 0, 0, fmt.Errorf(
			"die Buchungen des Geschäftsjahres %d konnten nicht gelesen werden: %w", year, err)
	}
	for i := range entries {
		entry := &entries[i]
		if entry.Source == domain.EntrySourceClosing {
			continue
		}
		for _, line := range entry.Lines {
			if !isIncomeTaxAccount(line.Account) {
				continue
			}
			amount := line.Amount
			if line.Side == domain.SideCredit {
				amount = -amount
			}
			if line.Account == domain.AccountGewerbesteuer {
				trade += amount
			} else {
				corporate += amount
			}
		}
	}
	return corporate, trade, nil
}

// taxProvisionLines baut den Buchungssatz: die Steuerarten laufen auf eigene
// Aufwandskonten und eigene Rückstellungskonten, weil § 4 Abs. 5b EStG die
// Gewerbesteuer vom Abzug ausnimmt und die Überleitung sie deshalb getrennt
// braucht.
func taxProvisionLines(result *accounting.TaxProvisionResult) []domain.JournalLine {
	lines := make([]domain.JournalLine, 0, 6)
	if result.IncomeProvision > 0 {
		corporate := result.CorporateTax
		solidarity := result.Solidarity
		// Die Vorauszahlungen mindern zuerst die Körperschaftsteuer; der
		// Solidaritätszuschlag folgt ihr.
		if surplus := corporate + solidarity - result.IncomeProvision; surplus > 0 {
			if corporate >= surplus {
				corporate -= surplus
			} else {
				solidarity -= surplus - corporate
				corporate = 0
			}
		}
		if corporate > 0 {
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: domain.AccountKoerperschaftsteuer,
				Amount: corporate, Text: "Körperschaftsteuer",
			})
		}
		if solidarity > 0 {
			lines = append(lines, domain.JournalLine{
				Side: domain.SideDebit, Account: domain.AccountSolidaritaetszuschlag,
				Amount: solidarity, Text: "Solidaritätszuschlag",
			})
		}
		lines = append(lines, domain.JournalLine{
			Side: domain.SideCredit, Account: domain.AccountRueckstellungKoerperschaft,
			Amount: result.IncomeProvision, Text: "Steuerrückstellung",
		})
	}
	if result.TradeProvision > 0 {
		lines = append(lines,
			domain.JournalLine{
				Side: domain.SideDebit, Account: domain.AccountGewerbesteuer,
				Amount: result.TradeProvision, Text: "Gewerbesteuer",
			},
			domain.JournalLine{
				Side: domain.SideCredit, Account: domain.AccountRueckstellungGewerbesteuer,
				Amount: result.TradeProvision, Text: "Gewerbesteuerrückstellung",
			},
		)
	}
	return lines
}

// TaxProvisionRequest bucht die Steuerrückstellung, wahlweise mit geänderten
// Beträgen: der Steuerberater kennt Verlustvorträge und Hinzurechnungen.
type TaxProvisionRequest struct {
	FiscalYear      int          `json:"fiscalYear"`
	IncomeProvision domain.Cents `json:"incomeProvision"`
	TradeProvision  domain.Cents `json:"tradeProvision"`
	Reason          string       `json:"reason"`
}

// BookTaxProvision bildet die Steuerrückstellungen und bucht sie.
func (s *ClosingBookingService) BookTaxProvision(
	ctx context.Context, req TaxProvisionRequest,
) ([]domain.Provision, error) {
	preview, err := s.PreviewTaxProvision(ctx, req.FiscalYear)
	if err != nil {
		return nil, err
	}
	// Zweimal gebucht wäre die Steuerrückstellung doppelt gebildet: die erste
	// Buchung steht, und eine zweite käme oben drauf. Wer den Betrag ändern
	// will, storniert die erste Buchung — das ist der Weg, den § 239 Abs. 3 HGB
	// für alles Gebuchte vorsieht. Und genau deshalb zählt hier nur, was nach
	// dem Storno übrig bleibt: liveProvisions streicht die Bewegungen, deren
	// Buchung zurückgenommen wurde, und mit ihnen die Rückstellung, von der
	// nichts mehr übrig ist. Sonst wäre der Rat, zu stornieren, ein Rat in eine
	// Sackgasse — der Lauf bliebe für immer gesperrt.
	stored, err := s.provisionRepo.FindByYear(ctx, preview.FiscalYear)
	if err != nil {
		return nil, err
	}
	booked, err := liveProvisions(ctx, s.journalRepo, stored)
	if err != nil {
		return nil, err
	}
	for i := range booked {
		if !booked[i].Kind.IsTax() {
			continue
		}
		return nil, fmt.Errorf(
			"für das Geschäftsjahr %d besteht bereits eine Steuerrückstellung (%q über %s €). "+
				"Storniere zuerst ihre Buchung; ein zweiter Lauf bildete sie ein zweites Mal",
			preview.FiscalYear, booked[i].Text, booked[i].SettlementAmount)
	}
	result := preview.TaxProvisionResult
	if req.IncomeProvision > 0 || req.TradeProvision > 0 {
		// Geänderte Beträge: die Aufteilung auf Körperschaftsteuer und
		// Solidaritätszuschlag bleibt im Verhältnis der Rechnung, weil sie
		// gesetzlich feststeht (5,5 % auf die Körperschaftsteuer).
		result.IncomeProvision = req.IncomeProvision
		result.TradeProvision = req.TradeProvision
		if total := result.CorporateTax + result.Solidarity; total > 0 {
			result.CorporateTax = domain.MulRound(req.IncomeProvision, int64(result.CorporateTax), int64(total))
			result.Solidarity = req.IncomeProvision - result.CorporateTax
		}
		result.TradeTax = req.TradeProvision
	}
	lines := taxProvisionLines(&result)
	if len(lines) == 0 {
		return nil, fmt.Errorf(
			"für das Geschäftsjahr %d ergibt sich keine Steuerrückstellung: das Ergebnis von %s € "+
				"trägt keine oder die Vorauszahlungen decken sie bereits",
			preview.FiscalYear, preview.Input.ProfitBeforeTax)
	}

	reason := req.Reason
	if reason == "" {
		reason = preview.Explanation
	}
	entry := &domain.JournalEntry{
		BookingDate: preview.Cutoff, DocumentDate: preview.Cutoff,
		ServiceDateFrom: preview.Cutoff, ServiceDateTo: preview.Cutoff,
		Description:        fmt.Sprintf("Steuerrückstellung %d", preview.FiscalYear),
		Source:             domain.EntrySourceClosing,
		PostingRuleVersion: accounting.PostingRuleVersion,
		Lines:              lines,
	}
	receipt, err := selfIssuedVoucher(ctx, s.receipts, preview.FiscalYear, closingVoucher{
		Kind: "steuerrueckstellung", FiscalYear: preview.FiscalYear, Date: preview.Cutoff,
		Description: entry.Description, Explanation: preview.Explanation,
		Calculation: result, Lines: lines,
	})
	if err != nil {
		return nil, err
	}
	attachVoucher(entry, receipt)

	created, err := postWithVoucher(ctx, s.journalSvc, s.receipts, entry, receipt)
	if err != nil {
		return nil, err
	}

	provisions := make([]domain.Provision, 0, 2)
	for _, part := range []struct {
		kind   domain.ProvisionKind
		amount domain.Cents
		text   string
	}{
		{domain.ProvisionTaxIncome, result.IncomeProvision, "Körperschaftsteuer und Solidaritätszuschlag"},
		{domain.ProvisionTaxTrade, result.TradeProvision, "Gewerbesteuer"},
	} {
		if part.amount <= 0 {
			continue
		}
		balance, expense := accounting.ProvisionAccounts(part.kind)
		provision := &domain.Provision{
			FiscalYear: preview.FiscalYear, Kind: part.kind,
			Text:             fmt.Sprintf("%s %d", part.text, preview.FiscalYear),
			SettlementAmount: part.amount, DiscountedAmount: part.amount,
			// Steuern des abgelaufenen Jahres werden mit der Veranlagung fällig;
			// binnen eines Jahres wird nicht abgezinst (§ 253 Abs. 2 Satz 1 HGB).
			ExpectedDate:   preview.Cutoff,
			BalanceAccount: balance, ExpenseAccount: expense, Reason: reason,
		}
		if err := s.provisionRepo.Save(ctx, provision); err != nil {
			return provisions, fmt.Errorf(
				"die Buchung %s wurde geschrieben, die Rückstellung aber nicht gespeichert: %w",
				created.EntryNumber, err)
		}
		movement := &domain.ProvisionMovement{
			ProvisionID: provision.ID, Kind: domain.ProvisionFormation, Date: preview.Cutoff,
			FiscalYear: preview.FiscalYear, Amount: part.amount, Reason: reason,
			JournalEntryID: &created.ID,
		}
		if err := s.provisionRepo.AddMovement(ctx, movement); err != nil {
			return provisions, err
		}
		provision.Movements = append(provision.Movements, *movement)
		provisions = append(provisions, *provision)
	}
	s.audit(ctx, "TAX_PROVISION", preview.FiscalYear, fmt.Sprintf(
		"Steuerrückstellung %d: Körperschaftsteuer und Soli %s €, Gewerbesteuer %s €",
		preview.FiscalYear, result.IncomeProvision, result.TradeProvision))
	return provisions, nil
}

// isIncomeTaxAccount meldet, ob ein Konto die Steuern vom Einkommen und Ertrag
// trägt (§ 275 Abs. 2 Nr. 14 HGB). Der SKR04 führt sie im Bereich 7600 bis 7649.
func isIncomeTaxAccount(account string) bool {
	n, err := strconv.Atoi(account)
	if err != nil {
		return false
	}
	return n >= 7600 && n <= 7649
}

// tradeTaxRate liest den Hebesatz der Gemeinde aus den Einstellungen.
func (s *ClosingBookingService) tradeTaxRate(ctx context.Context) int64 {
	const defaultRate = 400
	if s.settingsRepo == nil {
		return defaultRate
	}
	value, err := s.settingsRepo.Get(ctx, domain.SettingTradeTaxRate)
	if err != nil || strings.TrimSpace(value) == "" {
		return defaultRate
	}
	rate, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || rate <= 0 {
		return defaultRate
	}
	return rate
}

func (s *ClosingBookingService) audit(ctx context.Context, entity string, year int, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionCreate, entity, fmt.Sprintf("%d", year), details)
}
