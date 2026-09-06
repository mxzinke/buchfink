package accounting

import (
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Zuordnungsvorschlag für Bankumsätze.
//
// „Die Buchung folgt dem Bankumsatz" heißt nicht, dass Buchfink bucht: es heißt,
// dass die Anwenderin den wahrscheinlichsten offenen Posten vorausgewählt
// bekommt und ihn mit einem Klick bestätigt. Deshalb liefert die Bewertung eine
// Punktzahl und Gründe — wer den Vorschlag annimmt, soll sehen, worauf er
// beruht, und wer ihn ablehnt, soll wissen, warum das Programm ihn gemacht hat.

// Die Gewichte der Merkmale. Ihre Reihenfolge ist die Aussage: der genaue
// Betrag und die Rechnungsnummer im Verwendungszweck sind harte Merkmale, die
// Namensähnlichkeit ist ein Indiz, die Datumsnähe allein sagt fast nichts. Jedes
// harte Merkmal für sich schlägt deshalb jedes weiche.
const (
	MatchScoreExactAmount    = 50
	MatchScoreDocumentNumber = 40
	MatchScoreNameSimilar    = 20
	MatchScoreDueNear        = 10
)

// MatchDueWindowDays ist der Abstand, innerhalb dessen die Fälligkeit als nahe
// gilt. Vierzehn Tage, weil das übliche Zahlungsziel dort liegt und ein
// größeres Fenster jeden Posten des Monats einschlösse.
const MatchDueWindowDays = 14

// MatchTransaction ist der Bankumsatz, soweit die Bewertung ihn braucht.
type MatchTransaction struct {
	// Amount trägt das Vorzeichen des Auszugs: positiv für Geldeingang.
	Amount           domain.Cents
	RemittanceInfo   string
	CounterpartyName string
	BookingDate      string
}

// MatchItem ist der offene Posten, soweit die Bewertung ihn braucht.
type MatchItem struct {
	DocumentNumber string
	EntryNumber    string
	ContactName    string
	DueDate        string
	OpenAmount     domain.Cents
}

// ScoreMatch bewertet, wie gut ein offener Posten zu einem Bankumsatz passt.
//
// Die Gründe kommen in derselben Reihenfolge wie die Gewichte, damit der beste
// Grund zuerst dasteht.
func ScoreMatch(tx MatchTransaction, item MatchItem) (int, []string) {
	score := 0
	reasons := make([]string, 0, 4)

	if item.OpenAmount != 0 && tx.Amount.Abs() == item.OpenAmount.Abs() {
		score += MatchScoreExactAmount
		reasons = append(reasons, "Der Betrag stimmt genau überein.")
	}
	if reference := matchedReference(tx.RemittanceInfo, item.DocumentNumber, item.EntryNumber); reference != "" {
		score += MatchScoreDocumentNumber
		reasons = append(reasons, "Der Verwendungszweck nennt "+reference+".")
	}
	if similarNames(tx.CounterpartyName, item.ContactName) {
		score += MatchScoreNameSimilar
		reasons = append(reasons, "Der Zahlungspartner ähnelt dem Namen des Geschäftspartners.")
	}
	if daysApart(item.DueDate, tx.BookingDate) <= MatchDueWindowDays {
		score += MatchScoreDueNear
		reasons = append(reasons, "Die Fälligkeit liegt nahe am Buchungstag.")
	}
	return score, reasons
}

// matchedReference liefert die im Verwendungszweck gefundene Nummer oder "".
func matchedReference(remittance, documentNumber, entryNumber string) string {
	normalized := normalizeReference(remittance)
	if normalized == "" {
		return ""
	}
	for _, candidate := range []string{documentNumber, entryNumber} {
		key := normalizeReference(candidate)
		// Kurze Zeichenfolgen werden nicht gesucht: „7" steht in jedem
		// Verwendungszweck, und ein Treffer darauf wäre keiner.
		if len(key) < 4 {
			continue
		}
		if strings.Contains(normalized, key) {
			return candidate
		}
	}
	return ""
}

// normalizeReference macht aus „RE-2026/0001" das Vergleichsmuster „re20260001".
//
// Bindestriche, Schrägstriche und Leerzeichen fallen weg, weil sie beim
// Übertragen in den Verwendungszweck regelmäßig verloren gehen oder durch andere
// ersetzt werden — die Nummer bleibt dieselbe.
func normalizeReference(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// legalFormTokens sind die Rechtsformzusätze, die für die Namensähnlichkeit
// nichts hergeben: „GmbH" steht in jedem zweiten Namen.
var legalFormTokens = map[string]bool{
	"gmbh": true, "ag": true, "ug": true, "kg": true, "ohg": true, "gbr": true,
	"mbh": true, "co": true, "ltd": true, "se": true, "ev": true, "eg": true,
	"und": true, "der": true, "die": true, "das": true,
}

// similarNames meldet, ob zwei Namen denselben kennzeichnenden Bestandteil
// tragen.
//
// Kein Ähnlichkeitsmaß über Zeichenabstände: der Zahlungspartner auf dem
// Kontoauszug ist selten falsch geschrieben, aber regelmäßig anders abgekürzt
// („Muster Handels GmbH" gegen „Muster Handels-GmbH & Co. KG"). Ein
// gemeinsamer kennzeichnender Bestandteil trifft diesen Fall und erzeugt keine
// Treffer zwischen zwei Namen, die nur beide „GmbH" heißen.
func similarNames(a, b string) bool {
	tokensA := nameTokens(a)
	if len(tokensA) == 0 {
		return false
	}
	tokensB := nameTokens(b)
	for _, t := range tokensA {
		for _, u := range tokensB {
			if t == u {
				return true
			}
		}
	}
	return false
}

// nameTokens zerlegt einen Namen in seine kennzeichnenden Bestandteile.
func nameTokens(name string) []string {
	fields := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 3 || legalFormTokens[f] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// daysApart liefert den Abstand zweier Tage in Tagen, oder eine große Zahl,
// wenn eines der Daten fehlt.
func daysApart(a, b string) int {
	from, err := time.Parse("2006-01-02", a)
	if err != nil {
		return 1 << 20
	}
	to, err := time.Parse("2006-01-02", b)
	if err != nil {
		return 1 << 20
	}
	days := int(to.Sub(from).Hours() / 24)
	if days < 0 {
		return -days
	}
	return days
}

// BankRulePattern bildet das Muster, unter dem eine Zuordnung gelernt wird.
//
// Zahlungspartner und Verwendungszweck zusammen, beide normalisiert und der
// Verwendungszweck auf seine ersten kennzeichnenden Wörter gekürzt: die
// Mietzahlung nennt jeden Monat denselben Empfänger und denselben Zweck, aber
// ein wechselndes Datum oder eine laufende Nummer. Wer das ganze
// Verwendungszweckfeld zum Muster machte, lernte eine Regel, die nie wieder
// passt.
//
// Die Geldrichtung gehört in den Schlüssel und nicht nur an die Regel. Sonst
// träfe die Rückerstattung desselben Partners mit demselben Verwendungszweck
// dieselbe Regel und schriebe sie auf „Geldeingang" um — und die nächste
// Mietzahlung fände ihre eigene Regel nicht mehr. Ein Geldeingang und ein
// Geldausgang sind zwei Vorgänge, auch wenn sie gleich heißen.
func BankRulePattern(counterpartyName, remittanceInfo string, moneyIn bool) string {
	parts := make([]string, 0, 2)
	if partner := BankRulePartnerKey(counterpartyName); partner != "" {
		parts = append(parts, partner)
	}
	if tokens := purposeTokens(remittanceInfo); len(tokens) > 0 {
		parts = append(parts, strings.Join(tokens, " "))
	}
	joined := strings.Join(parts, BankRulePatternSeparator)
	if joined == "" {
		// Ohne kennzeichnenden Bestandteil gibt es nichts zu lernen: eine Regel
		// aus der Richtung allein passte auf jeden Umsatz.
		return ""
	}
	return joined + " (" + BankRuleDirectionLabel(moneyIn) + ")"
}

// BankRuleDirectionLabel benennt die Geldrichtung, wie sie im Muster steht.
//
// Ausgeschrieben und nicht als Zeichen: das Muster ist in den Einstellungen zu
// sehen und dient dort, wo kein eigener Name gelernt wurde, als Beschriftung.
func BankRuleDirectionLabel(moneyIn bool) string {
	if moneyIn {
		return "Geldeingang"
	}
	return "Geldausgang"
}

// periodTokens sind die Wörter, die den Zeitraum eines wiederkehrenden Umsatzes
// benennen und deshalb nicht ins Muster gehören.
//
// „Miete Büro März 2026" und „Miete Büro April 2026" sind derselbe Vorgang. Die
// Ziffernschreibweise („03/2026") fällt schon durch die Zahlenprüfung weg, der
// ausgeschriebene Monat nicht — und gerade er ist bei Miete, Gehalt und
// Abschlägen die Regel. Ohne diese Liste lernte Buchfink jeden Monat eine neue
// Regel, die im Folgemonat nie wieder passte.
var periodTokens = map[string]bool{
	"januar": true, "februar": true, "maerz": true, "märz": true, "april": true,
	"mai": true, "juni": true, "juli": true, "august": true, "september": true,
	"oktober": true, "november": true, "dezember": true,
	"jan": true, "feb": true, "mrz": true, "mär": true, "apr": true,
	"jun": true, "jul": true, "aug": true, "sep": true, "sept": true,
	"okt": true, "nov": true, "dez": true,
	"quartal": true, "monat": true, "jahr": true, "halbjahr": true,
}

// purposeTokens nimmt die ersten drei Wörter des Verwendungszwecks, die den
// Vorgang benennen. Zahlen fallen weg — sie sind Monat, Jahr oder
// Vertragsnummer und wechseln —, und die Zeitraumwörter ebenso.
func purposeTokens(remittance string) []string {
	fields := strings.FieldsFunc(strings.ToLower(remittance), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, 3)
	for _, f := range fields {
		if len(f) < 3 || !hasLetter(f) || periodTokens[f] {
			continue
		}
		out = append(out, f)
		if len(out) == 3 {
			break
		}
	}
	return out
}

// BankRulePatternSeparator trennt den Zahlungspartner vom Verwendungszweck im
// Muster.
const BankRulePatternSeparator = " | "

// BankRulePartnerKey ist der Zahlungspartner-Teil eines Musters.
//
// Sortiert, weil die Bank denselben Empfänger nicht immer gleich schreibt und
// die Reihenfolge der Bestandteile deshalb nichts aussagt.
func BankRulePartnerKey(counterpartyName string) string {
	tokens := nameTokens(counterpartyName)
	if len(tokens) == 0 {
		return ""
	}
	sort.Strings(tokens)
	return strings.Join(tokens, " ")
}

// BankRulePartnerOf liest den Zahlungspartner aus einem gespeicherten Muster.
//
// Er ist der Rückfall der Regelsuche: schreibt derselbe Empfänger den
// Verwendungszweck einmal anders („Miete Büro" statt „Mietzahlung Büro"), passt
// das vollständige Muster nicht mehr, der Partner aber schon. Ein Vorschlag
// allein aus dem Partner ist schwächer und wird deshalb auch schwächer bewertet
// — aber er ist besser als keiner.
//
// Ein Muster ohne Trenner trägt keinen Verwendungszweck; dann ist es selbst der
// Partner.
func BankRulePartnerOf(pattern string) string {
	if idx := strings.Index(pattern, BankRulePatternSeparator); idx >= 0 {
		return pattern[:idx]
	}
	return pattern
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// GroupForAccount sucht die Buchungsgruppe zu einem Konto.
//
// Für die gelernte Regel: gebucht wird gegen ein Konto, angeboten wird eine
// Gruppe — und die Oberfläche spricht in Gruppen. Teilen sich mehrere Gruppen
// ein Konto, gewinnt die erste; sie ist dann die allgemeinere, weil die Liste
// vom Allgemeinen zum Besonderen geordnet ist. Ein leeres Ergebnis heißt: das
// Konto gehört zu keiner Gruppe, dann bleibt es beim Konto.
func GroupForAccount(account string) string {
	if account == "" {
		return ""
	}
	for _, group := range postingGroups {
		if group.Account == account {
			return group.Key
		}
		for _, mapped := range group.RateAccounts {
			if mapped == account {
				return group.Key
			}
		}
	}
	return ""
}

// FindAmountCombination sucht eine Teilmenge, deren Summe den Betrag genau
// trifft, und liefert ihre Indizes.
//
// Die Sammelzahlung ist der Fall, für den es sie gibt: ein Kunde überweist drei
// Rechnungen in einem Betrag, und ohne diese Suche steht der Umsatz als
// unzuordenbar da, obwohl die Posten offen daneben liegen. Gesucht wird mit
// einer Schranke an Posten und Schritten — die Aufgabe ist im Allgemeinen
// aufwendig, und eine Oberfläche, die beim Öffnen eines Kontoauszugs
// nachdenkt, ist keine.
func FindAmountCombination(values []domain.Cents, target domain.Cents, maxItems int) []int {
	if target == 0 || len(values) == 0 {
		return nil
	}
	if maxItems <= 0 {
		maxItems = 4
	}
	budget := 20000
	var chosen []int
	var walk func(start int, remaining domain.Cents, picked []int) bool
	walk = func(start int, remaining domain.Cents, picked []int) bool {
		if budget <= 0 {
			return false
		}
		budget--
		if remaining == 0 && len(picked) > 0 {
			chosen = append([]int(nil), picked...)
			return true
		}
		if remaining < 0 || len(picked) >= maxItems {
			return false
		}
		for i := start; i < len(values); i++ {
			if values[i] <= 0 || values[i] > remaining {
				continue
			}
			if walk(i+1, remaining-values[i], append(picked, i)) {
				return true
			}
		}
		return false
	}
	if walk(0, target, nil) {
		return chosen
	}
	return nil
}
