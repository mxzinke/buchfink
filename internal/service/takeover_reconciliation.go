package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/domain"
)

// TakeoverAccountRow stellt je Sachkonto den Bestand der Schlussbilanz des
// Vorsystems dem gebuchten Eröffnungsbestand gegenüber. Beträge sind Salden:
// Soll positiv, Haben negativ. Die Sammelkonten zeigen als gebuchten Bestand
// die Summe der übernommenen Personenkonten.
type TakeoverAccountRow struct {
	Account     string       `json:"account"`
	AccountName string       `json:"accountName"`
	Statement   domain.Cents `json:"statement"`
	Booked      domain.Cents `json:"booked"`
	Difference  domain.Cents `json:"difference"`
}

// TakeoverSubLedgerRow stellt ein Nebenbuch seinem Sachkonto gegenüber: die
// offenen Posten dem Sammelkonto der Schlussbilanz, die Anlagen eines
// Anlagekontos dessen gebuchtem Eröffnungsbestand.
type TakeoverSubLedgerRow struct {
	// Kind ist "receivables", "payables" oder "assets".
	Kind        string `json:"kind"`
	Account     string `json:"account"`
	AccountName string `json:"accountName"`
	// Count ist die Zahl der offenen Posten oder Anlagen.
	Count int `json:"count"`
	// SubLedger ist die Summe des Nebenbuchs, Ledger der Bestand des Sachkontos,
	// beide als Betrag ohne Vorzeichen.
	SubLedger  domain.Cents `json:"subLedger"`
	Ledger     domain.Cents `json:"ledger"`
	Difference domain.Cents `json:"difference"`
}

// TakeoverReconciliation ist die Abstimmung der Eröffnung mit der Schlussbilanz
// des Vorsystems.
type TakeoverReconciliation struct {
	TakeoverState
	// ReceiptID ist der Beleg mit der Schlussbilanz des Vorsystems.
	ReceiptID    uint   `json:"receiptId,omitempty"`
	LegacySystem string `json:"legacySystem,omitempty"`

	Accounts   []TakeoverAccountRow   `json:"accounts"`
	SubLedgers []TakeoverSubLedgerRow `json:"subLedgers"`

	// Aktiva und Passiva der Schlussbilanz und der Eröffnungsbuchungen.
	StatementAssets      domain.Cents `json:"statementAssets"`
	StatementLiabilities domain.Cents `json:"statementLiabilities"`
	BookedAssets         domain.Cents `json:"bookedAssets"`
	BookedLiabilities    domain.Cents `json:"bookedLiabilities"`
	// CarryForwardBalance ist der Saldo der Vortragskonten 9000, 9008 und 9009
	// aus den Eröffnungsbuchungen; null, wenn die Eröffnung aufgeht.
	CarryForwardBalance domain.Cents `json:"carryForwardBalance"`

	// Differences nennt jede Abweichung in einem Satz. Reconciled ist wahr, wenn
	// die Eröffnung gebucht ist und keine Abweichung besteht.
	Differences []string `json:"differences"`
	Reconciled  bool     `json:"reconciled"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *TakeoverReconciliation) EnsureLists() {
	if r.Accounts == nil {
		r.Accounts = make([]TakeoverAccountRow, 0)
	}
	if r.SubLedgers == nil {
		r.SubLedgers = make([]TakeoverSubLedgerRow, 0)
	}
	if r.Differences == nil {
		r.Differences = make([]string, 0)
	}
}

// Reconciliation stellt die Eröffnungsbuchungen der Schlussbilanz des
// Vorsystems gegenüber: je Sachkonto und in Summe, Aktiva gegen Passiva, und
// die Nebenbücher der offenen Posten und Anlagen gegen ihre Sachkonten.
func (s *TakeoverService) Reconciliation(ctx context.Context) (*TakeoverReconciliation, error) {
	state, err := s.State(ctx)
	if err != nil {
		return nil, err
	}
	rec := &TakeoverReconciliation{TakeoverState: *state}
	rec.EnsureLists()
	if state.Date == "" {
		return rec, nil
	}
	entries, err := s.openingEntries(ctx, state)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		rec.Differences = append(rec.Differences, fmt.Sprintf(
			"Zum %s ist keine Eröffnung aus der Schlussbilanz des Vorsystems gebucht.",
			domain.GermanDate(state.Date)))
		return rec, nil
	}
	statement, err := s.statement(ctx)
	if err != nil {
		return nil, err
	}
	chart, err := s.journal.Chart(ctx)
	if err != nil {
		return nil, err
	}

	// Gebuchter Stand: Sachkonten für sich, Personenkonten unter ihrem
	// Sammelkonto.
	booked := map[string]domain.Cents{}
	itemCount := map[string]int{}
	for i := range entries {
		for _, line := range entries[i].Lines {
			amount := line.Amount
			if line.Side == domain.SideCredit {
				amount = -amount
			}
			if domain.IsCarryForwardAccount(line.Account) {
				rec.CarryForwardBalance += amount
				continue
			}
			account := line.Account
			if kind, ok := domain.LedgerAccountKind(line.Account); ok {
				account = collectiveAccountOf(kind)
				itemCount[account]++
			}
			booked[account] += amount
			if amount > 0 {
				rec.BookedAssets += amount
			} else {
				rec.BookedLiabilities -= amount
			}
		}
	}
	stated := map[string]domain.Cents{}
	if statement != nil {
		rec.ReceiptID, rec.LegacySystem = statement.ReceiptID, statement.LegacySystem
		for _, line := range statement.Accounts {
			if line.Side == domain.SideCredit {
				stated[line.Account] -= line.Amount
				rec.StatementLiabilities += line.Amount
			} else {
				stated[line.Account] += line.Amount
				rec.StatementAssets += line.Amount
			}
		}
	} else {
		rec.Differences = append(rec.Differences,
			"Zu den Eröffnungsbuchungen ist keine Schlussbilanz des Vorsystems erfasst.")
	}

	accounts := make([]string, 0, len(booked)+len(stated))
	for account := range booked {
		accounts = append(accounts, account)
	}
	for account := range stated {
		if _, ok := booked[account]; !ok {
			accounts = append(accounts, account)
		}
	}
	sort.Strings(accounts)
	for _, account := range accounts {
		row := TakeoverAccountRow{
			Account: account, AccountName: chart.Name(account),
			Statement: stated[account], Booked: booked[account],
			Difference: stated[account] - booked[account],
		}
		rec.Accounts = append(rec.Accounts, row)
		if statement != nil && row.Difference != 0 {
			rec.Differences = append(rec.Differences, fmt.Sprintf(
				"Konto %s %s: Schlussbilanz %s, gebucht %s, Differenz %s €.",
				account, row.AccountName, signedBalance(row.Statement), signedBalance(row.Booked), row.Difference.Abs()))
		}
	}

	if statement != nil && rec.StatementAssets != rec.StatementLiabilities {
		rec.Differences = append(rec.Differences, fmt.Sprintf(
			"Aktiva (%s €) und Passiva (%s €) der Schlussbilanz unterscheiden sich um %s €.",
			rec.StatementAssets, rec.StatementLiabilities, (rec.StatementAssets-rec.StatementLiabilities).Abs()))
	}
	if rec.CarryForwardBalance != 0 {
		rec.Differences = append(rec.Differences, fmt.Sprintf(
			"Die Eröffnungsbuchungen gehen nicht auf: Aktiva %s €, Passiva %s €, die Vortragskonten 9000, "+
				"9008 und 9009 stehen auf %s €.",
			rec.BookedAssets, rec.BookedLiabilities, rec.CarryForwardBalance.Abs()))
	}

	for _, sub := range []struct{ kind, account, label string }{
		{"receivables", domain.AccountForderungenLuL, "Forderungen"},
		{"payables", domain.AccountVerbindlichkeitenLuL, "Verbindlichkeiten"},
	} {
		if itemCount[sub.account] == 0 && stated[sub.account] == 0 {
			continue
		}
		row := TakeoverSubLedgerRow{
			Kind: sub.kind, Account: sub.account, AccountName: chart.Name(sub.account),
			Count:     itemCount[sub.account],
			SubLedger: booked[sub.account].Abs(), Ledger: stated[sub.account].Abs(),
		}
		row.Difference = row.SubLedger - row.Ledger
		rec.SubLedgers = append(rec.SubLedgers, row)
	}
	assetRows, err := s.assetSubLedger(ctx, state, booked, chart.Name)
	if err != nil {
		return nil, err
	}
	for _, row := range assetRows {
		rec.SubLedgers = append(rec.SubLedgers, row)
		if row.Difference != 0 {
			rec.Differences = append(rec.Differences, fmt.Sprintf(
				"Die Anlagen auf Konto %s %s haben zusammen einen Buchwert von %s €, gebucht sind %s € "+
					"(Differenz %s €).",
				row.Account, row.AccountName, row.SubLedger, row.Ledger, row.Difference.Abs()))
		}
	}
	rec.Reconciled = len(rec.Differences) == 0
	return rec, nil
}

// assetSubLedger vergleicht je Anlagekonto die Buchwerte der übernommenen
// Anlagen zum Umstieg mit dem gebuchten Eröffnungsbestand.
func (s *TakeoverService) assetSubLedger(
	ctx context.Context, state *TakeoverState, booked map[string]domain.Cents, name func(string) string,
) ([]TakeoverSubLedgerRow, error) {
	assets, err := s.takenOverAssets(ctx)
	if err != nil {
		return nil, err
	}
	rows := map[string]*TakeoverSubLedgerRow{}
	rowFor := func(account string) *TakeoverSubLedgerRow {
		row, ok := rows[account]
		if !ok {
			row = &TakeoverSubLedgerRow{Kind: "assets", Account: account, AccountName: name(account)}
			rows[account] = row
		}
		return row
	}
	for i := range assets {
		cost, carried := takeoverValues(&assets[i], state.FiscalYear)
		row := rowFor(assets[i].Account)
		row.Count++
		row.SubLedger += cost - carried
	}
	for account, amount := range booked {
		if isFixedAssetAccount(account) {
			rowFor(account).Ledger = amount
		}
	}
	out := make([]TakeoverSubLedgerRow, 0, len(rows))
	for _, row := range rows {
		row.Difference = row.SubLedger - row.Ledger
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Account < out[j].Account })
	return out, nil
}

// Findings meldet dem Prüflauf eine fehlende oder nicht aufgehende Eröffnung
// eines Umsteigers. Der Befund sperrt die Festschreibung: ein Monat, dessen
// Anfangsbestände nicht mit der Schlussbilanz des Vorsystems übereinstimmen,
// schriebe den Fehler fest.
func (s *TakeoverService) Findings(ctx context.Context, cutoff string) ([]domain.CheckFinding, error) {
	rec, err := s.Reconciliation(ctx)
	if err != nil {
		return nil, err
	}
	if rec.Date == "" || cutoff < rec.Date {
		return nil, nil
	}
	out := make([]domain.CheckFinding, 0, len(rec.Differences))
	for _, difference := range rec.Differences {
		out = append(out, domain.CheckFinding{
			Rule: domain.CheckRuleTakeoverOpening, Severity: domain.CheckBlocking,
			ObjectType: "TAKEOVER", ObjectID: rec.Date,
			ObjectName: "Eröffnung zum " + domain.GermanDate(rec.Date),
			Message:    difference,
			Reference:  "§ 252 Abs. 1 Nr. 1 HGB",
		})
	}
	return out, nil
}

func collectiveAccountOf(kind domain.ContactType) string {
	if kind == domain.ContactTypeVendor {
		return domain.AccountVerbindlichkeitenLuL
	}
	return domain.AccountForderungenLuL
}

// signedBalance nennt einen Saldo mit seiner Seite.
func signedBalance(amount domain.Cents) string {
	if amount < 0 {
		return amount.Abs().String() + " € im Haben"
	}
	return amount.String() + " € im Soll"
}
