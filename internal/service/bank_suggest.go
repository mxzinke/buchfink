package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Der Zuordnungsvorschlag zum Bankumsatz.
//
// Das Versprechen der Anwendung ist „die Buchung folgt dem Bankumsatz". Bisher
// hieß das: die Anwenderin sucht den passenden offenen Posten in einer Liste.
// Jetzt sucht Buchfink ihn und legt ihn vor — mit Punktzahl und Begründung, und
// ohne zu buchen. Der Klick, der bucht, bleibt beim Menschen.

// SuggestionKind sagt, welcher Art ein Vorschlag ist.
type SuggestionKind string

const (
	// SuggestionOpenItem ist ein einzelner offener Posten.
	SuggestionOpenItem SuggestionKind = "open_item"
	// SuggestionCollective ist die Sammelzahlung: mehrere Posten desselben
	// Partners, deren Summe den Betrag trifft.
	SuggestionCollective SuggestionKind = "collective"
	// SuggestionRule ist die gelernte Buchungsgruppe eines wiederkehrenden
	// Umsatzes ohne offenen Posten.
	SuggestionRule SuggestionKind = "rule"
)

// BankSuggestion ist ein Vorschlag zu einem Bankumsatz.
type BankSuggestion struct {
	Kind  SuggestionKind `json:"kind"`
	Score int            `json:"score"`
	// Reasons nennt die Merkmale, auf denen der Vorschlag beruht — in der
	// Reihenfolge ihres Gewichts.
	Reasons []string `json:"reasons"`
	Label   string   `json:"label"`

	ContactID   uint         `json:"contactId,omitempty"`
	ContactName string       `json:"contactName,omitempty"`
	Amount      domain.Cents `json:"amount"`
	// Items sind die vorgeschlagenen offenen Posten (bei der Sammelzahlung
	// mehrere).
	Items []domain.OpenItem `json:"items"`

	// Die gelernte Regel: Gegenkonto und Buchungsgruppe.
	CounterAccount string `json:"counterAccount,omitempty"`
	PostingGroup   string `json:"postingGroup,omitempty"`
	RuleID         uint   `json:"ruleId,omitempty"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (s *BankSuggestion) EnsureLists() {
	if s.Reasons == nil {
		s.Reasons = make([]string, 0)
	}
	if s.Items == nil {
		s.Items = make([]domain.OpenItem, 0)
	}
}

// BankSuggestions ist die Vorschlagsliste zu einem Bankumsatz, die beste zuerst.
type BankSuggestions struct {
	BankTxID    uint             `json:"bankTxId"`
	Amount      domain.Cents     `json:"amount"`
	Suggestions []BankSuggestion `json:"suggestions"`
	// Note steht dabei, wenn es nichts vorzuschlagen gab. Ein leerer Kasten
	// ohne Erklärung sieht aus wie ein Fehler.
	Note string `json:"note,omitempty"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (s *BankSuggestions) EnsureLists() {
	if s.Suggestions == nil {
		s.Suggestions = make([]BankSuggestion, 0)
	}
	for i := range s.Suggestions {
		s.Suggestions[i].EnsureLists()
	}
}

// SetOpenItemSource hängt die offene-Posten-Liste an, aus der die Vorschläge
// kommen. Ohne sie schlägt der Dienst nur gelernte Regeln vor.
func (s *BankService) SetOpenItemSource(src LiveOpenItemSource) { s.openItems = src }

// SetRuleRepo hängt die gelernten Zuordnungen an.
func (s *BankService) SetRuleRepo(repo domain.BankRuleRepository) { s.ruleRepo = repo }

// MaxCollectiveItems ist die Zahl der Posten, die eine Sammelzahlung höchstens
// zusammenfassen darf. Über acht Posten wird die Suche teuer und der Vorschlag
// unglaubwürdig: was zufällig zusammenpasst, ist keine Zahlung.
const MaxCollectiveItems = 8

// Suggest liefert die wahrscheinlichsten Zuordnungen zu einem Bankumsatz.
func (s *BankService) Suggest(ctx context.Context, bankTxID uint) (*BankSuggestions, error) {
	tx, err := s.bankRepo.FindByID(ctx, bankTxID)
	if err != nil {
		return nil, fmt.Errorf("Bankumsatz %d wurde nicht gefunden: %w", bankTxID, err)
	}

	out := &BankSuggestions{BankTxID: tx.ID, Amount: tx.Amount}
	out.EnsureLists()

	match := accounting.MatchTransaction{
		Amount:           tx.Amount,
		RemittanceInfo:   tx.RemittanceInfo,
		CounterpartyName: tx.CounterpartyName,
		BookingDate:      tx.BookingDate,
	}

	items := s.candidates(ctx, tx)
	for i := range items {
		item := items[i]
		score, reasons := accounting.ScoreMatch(match, accounting.MatchItem{
			DocumentNumber: item.DocumentNumber,
			EntryNumber:    item.EntryNumber,
			ContactName:    item.ContactName,
			DueDate:        item.DueDate,
			OpenAmount:     item.OpenAmount,
		})
		// Ein Vorschlag ohne ein einziges Merkmal ist keiner: er wäre nur die
		// Liste aller offenen Posten in anderer Reihenfolge.
		if score == 0 {
			continue
		}
		suggestion := BankSuggestion{
			Kind: SuggestionOpenItem, Score: score, Reasons: reasons,
			ContactID: item.ContactID, ContactName: item.ContactName,
			Amount: item.OpenAmount,
			Items:  []domain.OpenItem{item},
			Label: fmt.Sprintf("%s – %s über %s €",
				item.ContactName, documentLabel(item), item.OpenAmount),
		}
		out.Suggestions = append(out.Suggestions, suggestion)
	}

	out.Suggestions = append(out.Suggestions, s.collectiveSuggestions(match, items)...)

	if rule := s.ruleSuggestion(ctx, tx); rule != nil {
		out.Suggestions = append(out.Suggestions, *rule)
	}

	// Der beste zuerst; bei gleicher Punktzahl der ältere Posten, damit die
	// Reihenfolge nicht zwischen zwei Aufrufen springt.
	sort.SliceStable(out.Suggestions, func(i, j int) bool {
		a, b := out.Suggestions[i], out.Suggestions[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.Label < b.Label
	})

	if len(out.Suggestions) == 0 {
		out.Note = "Zu diesem Umsatz passt kein offener Posten und keine gelernte Regel. " +
			"Er ist von Hand zuzuordnen — Buchfink lernt die Zuordnung dann für das nächste Mal."
	}
	out.EnsureLists()
	return out, nil
}

// candidates sind die offenen Posten der Seite, die zum Vorzeichen des Umsatzes
// passt: ein Geldeingang gleicht eine Forderung aus, ein Ausgang eine
// Verbindlichkeit.
func (s *BankService) candidates(ctx context.Context, tx *domain.BankTransaction) []domain.OpenItem {
	if s.openItems == nil {
		return nil
	}
	items, err := s.openItems.OpenItems(ctx)
	if err != nil {
		return nil
	}
	wanted := domain.ContactTypeVendor
	if tx.Amount > 0 {
		wanted = domain.ContactTypeCustomer
	}
	out := make([]domain.OpenItem, 0, len(items))
	for i := range items {
		if items[i].ContactType == wanted && items[i].OpenAmount != 0 {
			out = append(out, items[i])
		}
	}
	return out
}

// collectiveSuggestions sucht je Geschäftspartner eine Kombination von Posten,
// deren Summe den Betrag genau trifft.
//
// Nur je Partner: ein Betrag, der zufällig zur Summe der Rechnungen dreier
// verschiedener Kunden passt, ist keine Sammelzahlung, sondern ein Zufall.
func (s *BankService) collectiveSuggestions(
	match accounting.MatchTransaction, items []domain.OpenItem,
) []BankSuggestion {
	byContact := map[uint][]domain.OpenItem{}
	order := make([]uint, 0, 4)
	for _, item := range items {
		if _, seen := byContact[item.ContactID]; !seen {
			order = append(order, item.ContactID)
		}
		byContact[item.ContactID] = append(byContact[item.ContactID], item)
	}

	target := match.Amount.Abs()
	out := make([]BankSuggestion, 0, 2)
	for _, contactID := range order {
		group := byContact[contactID]
		if len(group) < 2 {
			continue
		}
		amounts := make([]domain.Cents, len(group))
		for i := range group {
			amounts[i] = group[i].OpenAmount.Abs()
		}
		picked := accounting.FindAmountCombination(amounts, target, MaxCollectiveItems)
		if len(picked) < 2 {
			continue
		}
		chosen := make([]domain.OpenItem, 0, len(picked))
		var sum domain.Cents
		for _, idx := range picked {
			chosen = append(chosen, group[idx])
			sum += group[idx].OpenAmount
		}
		// Die Sammelzahlung erbt das Gewicht des genauen Betrages: sie trifft
		// ihn, nur eben über mehrere Posten. Die Namensähnlichkeit kommt dazu,
		// wenn der Zahlungspartner passt.
		score := accounting.MatchScoreExactAmount
		reasons := []string{fmt.Sprintf(
			"Die Summe von %d offenen Posten dieses Partners ergibt genau den Betrag.", len(chosen))}
		if _, nameReasons := accounting.ScoreMatch(match, accounting.MatchItem{
			ContactName: group[0].ContactName,
		}); len(nameReasons) > 0 {
			score += accounting.MatchScoreNameSimilar
			reasons = append(reasons, nameReasons...)
		}
		out = append(out, BankSuggestion{
			Kind: SuggestionCollective, Score: score, Reasons: reasons,
			ContactID: contactID, ContactName: group[0].ContactName,
			Amount: sum, Items: chosen,
			Label: fmt.Sprintf("Sammelzahlung %s über %s €", group[0].ContactName, sum),
		})
	}
	return out
}

// ruleSuggestion liefert die gelernte Buchungsgruppe zu einem wiederkehrenden
// Umsatz.
func (s *BankService) ruleSuggestion(ctx context.Context, tx *domain.BankTransaction) *BankSuggestion {
	if s.ruleRepo == nil {
		return nil
	}
	pattern := accounting.BankRulePattern(tx.CounterpartyName, tx.RemittanceInfo, tx.Amount > 0)
	if pattern == "" {
		return nil
	}
	rule, err := s.ruleRepo.FindByPattern(ctx, pattern)
	if err != nil {
		return nil
	}
	// Das Muster trägt die Geldrichtung seit Welle 7 selbst; die Prüfung bleibt
	// für die Regeln, die davor gelernt wurden.
	if rule != nil && rule.MoneyIn != (tx.Amount > 0) {
		rule = nil
	}
	// Der volle Treffer zuerst, sonst der Rückfall über den Zahlungspartner.
	score := accounting.MatchScoreNameSimilar + accounting.MatchScoreDueNear
	reason := ""
	if rule == nil {
		rule = s.partnerRule(ctx, tx.CounterpartyName, tx.Amount > 0)
		if rule == nil {
			return nil
		}
		// Schwächer bewertet: der Partner allein sagt weniger als Partner und
		// Verwendungszweck zusammen. Er bleibt aber über null und damit
		// sichtbar — ein Vorschlag, den die Anwenderin verwirft, kostet einen
		// Blick; ein fehlender kostet die Suche im Kontenplan.
		score = accounting.MatchScoreNameSimilar
		reason = fmt.Sprintf(
			"Umsätze desselben Zahlungspartners wurden zuletzt gegen Konto %s gebucht (%d-mal bestätigt).",
			rule.CounterAccount, rule.Hits)
	}
	if reason == "" {
		reason = fmt.Sprintf(
			"Ein gleichartiger Umsatz wurde zuletzt gegen Konto %s gebucht (%d-mal bestätigt).",
			rule.CounterAccount, rule.Hits)
	}
	label := rule.Label
	if label == "" {
		label = pattern
	}
	// Unterhalb des genauen Betrages und der Rechnungsnummer: ein offener
	// Posten, der passt, ist die bessere Antwort als eine Gewohnheit.
	return &BankSuggestion{
		Kind: SuggestionRule, Score: score,
		Reasons:        []string{reason},
		Label:          fmt.Sprintf("Wiederkehrend: %s", label),
		Amount:         tx.Amount.Abs(),
		CounterAccount: rule.CounterAccount,
		PostingGroup:   rule.PostingGroup,
		RuleID:         rule.ID,
	}
}

// partnerRule sucht eine gelernte Regel desselben Zahlungspartners in derselben
// Geldrichtung.
//
// Der Rückfall greift, wenn der Verwendungszweck sich ändert, ohne dass der
// Vorgang ein anderer wird. Gewählt wird die am häufigsten bestätigte Regel:
// sie ist die Gewohnheit und nicht der Einzelfall. Ein Partner ohne
// kennzeichnenden Bestandteil (leerer Namensteil) fällt heraus — sonst passte
// jede Regel ohne Partner auf jeden Umsatz ohne Partner.
func (s *BankService) partnerRule(
	ctx context.Context, counterpartyName string, moneyIn bool,
) *domain.BankRule {
	partner := accounting.BankRulePartnerKey(counterpartyName)
	if partner == "" {
		return nil
	}
	rules, err := s.ruleRepo.FindAll(ctx)
	if err != nil {
		return nil
	}
	var best *domain.BankRule
	for i := range rules {
		rule := &rules[i]
		if rule.MoneyIn != moneyIn || accounting.BankRulePartnerOf(rule.Pattern) != partner {
			continue
		}
		if best == nil || rule.Hits > best.Hits {
			best = rule
		}
	}
	return best
}

// documentLabel benennt den Beleg eines offenen Postens.
func documentLabel(item domain.OpenItem) string {
	if item.DocumentNumber != "" {
		return "Beleg " + item.DocumentNumber
	}
	if item.EntryNumber != "" {
		return "Buchung " + item.EntryNumber
	}
	return "offener Posten"
}

// Rules liefert die gelernten Zuordnungen für die Einstellungen.
func (s *BankService) Rules(ctx context.Context) ([]domain.BankRule, error) {
	if s.ruleRepo == nil {
		return make([]domain.BankRule, 0), nil
	}
	return s.ruleRepo.FindAll(ctx)
}

// DeleteRule entfernt eine gelernte Zuordnung.
//
// Löschen und nicht abschalten: eine Regel ist keine Aufzeichnung eines
// Geschäftsvorfalls, sondern eine Gewohnheit des Programms. Was sie einmal
// vorgeschlagen hat, steht als Buchung im Journal und bleibt dort.
func (s *BankService) DeleteRule(ctx context.Context, id uint) error {
	if s.ruleRepo == nil {
		return fmt.Errorf("die gelernten Bankregeln sind nicht eingerichtet")
	}
	if err := s.ruleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("die Regel konnte nicht entfernt werden: %w", err)
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "BANK_RULE", fmt.Sprintf("%d", id),
			"Gelernte Zuordnung eines wiederkehrenden Bankumsatzes gelöscht")
	}
	return nil
}

// learnRule merkt sich eine bestätigte Zuordnung.
//
// Sie läuft nach dem Buchen und darf deshalb nicht scheitern lassen, was schon
// gebucht ist: eine Gewohnheit, die nicht angelegt werden konnte, ist ein
// verlorener Vorschlag und kein verlorener Geschäftsvorfall.
func (s *BankService) learnRule(ctx context.Context, tx *domain.BankTransaction, counterAccount string) {
	if s.ruleRepo == nil || counterAccount == "" {
		return
	}
	pattern := accounting.BankRulePattern(tx.CounterpartyName, tx.RemittanceInfo, tx.Amount > 0)
	if pattern == "" {
		return
	}
	label := tx.CounterpartyName
	if label == "" {
		label = tx.RemittanceInfo
	}
	rule := &domain.BankRule{
		Pattern:        pattern,
		Label:          label,
		CounterAccount: counterAccount,
		PostingGroup:   accounting.GroupForAccount(counterAccount),
		MoneyIn:        tx.Amount > 0,
		LastUsedAt:     tx.BookingDate,
	}
	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return
	}
	// Auch das Lernen kommt ins Protokoll — wie das Löschen.
	//
	// Die Regel bucht nichts; sie ist eine Gewohnheit des Programms. Aber sie
	// ist in den Einstellungen sichtbar, und eine sichtbare Gewohnheit, deren
	// Entstehung nirgends steht, lässt sich im Prüfermodus nicht erklären.
	// Fehler bleiben unbeachtet: ein Protokolleintrag darf eine Buchung, die
	// schon geschrieben ist, nicht nachträglich zu Fall bringen.
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "BANK_RULE", fmt.Sprintf("%d", rule.ID),
			fmt.Sprintf(
				"Gelernte Zuordnung aus Bankumsatz %d: Muster %q → Gegenkonto %s (%s), %s, %d Treffer",
				tx.ID, rule.Pattern, rule.CounterAccount, rule.PostingGroup,
				moneyDirectionLabel(rule.MoneyIn), rule.Hits))
	}
}

// moneyDirectionLabel benennt die Geldrichtung einer Regel.
//
// Sie gehört in den Protokolleintrag, weil dieselbe Zeichenfolge im
// Verwendungszweck einmal eine Einnahme und einmal eine Ausgabe sein kann —
// „Miete" ist beim Vermieter Ertrag und beim Mieter Aufwand.
func moneyDirectionLabel(moneyIn bool) string {
	if moneyIn {
		return "Geldeingang"
	}
	return "Geldausgang"
}
