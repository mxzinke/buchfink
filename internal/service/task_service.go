package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Aufgabenliste (Architektur 6.1).
//
// Sie beantwortet die Frage, mit der jeder Arbeitstag anfängt: was ist heute zu
// tun. Buchfink kann sie beantworten, weil alle Angaben dazu schon in den Daten
// stehen — sie standen bisher nur auf sieben verschiedenen Seiten, und wer nicht
// wusste, dass es sie gibt, sah sie nie.
//
// Der Dienst rechnet nichts eigenes: er fragt die Dienste, die die jeweilige
// Sache führen, und übersetzt deren Ergebnis in Zeilen mit Titel, Grund, Ziel
// und Kontext. Deshalb sind die Quellen kleine Schnittstellen und keine
// Abhängigkeit auf die Dienste selbst — eine Aufgabenliste, die die halbe
// Anwendung anzieht, ließe sich nicht prüfen.

// LiveOpenItemSource liefert die heute offenen Posten.
//
// Neben OpenItemSource (Stichtagssicht) und nicht statt dessen: die
// Aufgabenliste und der Mahnlauf fragen, was jetzt offen ist, der Abschluss
// fragt, was am Stichtag offen war.
type LiveOpenItemSource interface {
	OpenItems(ctx context.Context) ([]domain.OpenItem, error)
}

// TaskCheckSource rechnet den Prüflauf, ohne ihn zu speichern.
type TaskCheckSource interface {
	Preview(ctx context.Context, req CheckRequest) (*domain.CheckRun, error)
}

// TaskDeadlineSource liefert die Fristen eines Jahres.
type TaskDeadlineSource interface {
	Deadlines(ctx context.Context, year int) ([]domain.Deadline, error)
}

// TaskBankSource liefert die importierten Bankumsätze.
type TaskBankSource interface {
	GetTransactions(ctx context.Context, fiscalYear int) ([]domain.BankTransaction, error)
}

// TaskReceiptSource liefert die abgelegten Belege eines Status.
type TaskReceiptSource interface {
	List(ctx context.Context, status domain.ReceiptStatus) ([]domain.Receipt, error)
}

// TaskAssetDocumentSource liefert die ablaufenden Dokumente am Anlagegut.
type TaskAssetDocumentSource interface {
	ExpiringDocuments(ctx context.Context, until string) ([]ExpiringDocument, error)
}

// TaskBackupSource liefert die letzte erfolgreiche Sicherung und die jüngsten
// Läufe.
//
// Beides, weil beides eine andere Frage beantwortet: die letzte erfolgreiche
// sagt, wie alt der jüngste brauchbare Stand ist, die jüngsten Läufe sagen, ob
// seitdem etwas schiefgegangen ist. Fragte die Aufgabenliste nur nach der
// ersten, fiele ein Fehlschlag erst nach BackupOverdueDays auf — und bis dahin
// glaubt der Anwender, es werde gesichert.
type TaskBackupSource interface {
	LastSuccessful(ctx context.Context) (*domain.BackupRun, error)
	// GetRuns liefert die jüngsten Läufe, der neueste zuerst.
	GetRuns(ctx context.Context, limit int) ([]domain.BackupRun, error)
}

// TaskCarryForwardSource liefert den Stand des Saldenvortrags in ein Jahr.
type TaskCarryForwardSource interface {
	CarryForwardState(ctx context.Context, toYear int) (*CarryForwardPreview, error)
	FiscalYears(ctx context.Context) ([]domain.FiscalYear, error)
}

// TaskAppropriationSource liefert den Ergebnisverwendungsbeschluss eines Jahres.
type TaskAppropriationSource interface {
	Appropriation(ctx context.Context, year int) (*domain.Appropriation, error)
}

// TaskOptions sind die Angaben, die nicht aus einem Dienst kommen.
//
// Der Prüfermodus steht an der Bridge und nicht in einem Dienst — er ist eine
// Eigenschaft der Bedienung, nicht der Buchführung. Er wird deshalb übergeben,
// statt dass der Dienst die Mandantenkonfiguration kennt.
type TaskOptions struct {
	// Today ist der Tag, gegen den gerechnet wird. Leer heißt: heute. Er ist
	// ein Parameter, weil eine Frist, die sich beim Testen nicht setzen lässt,
	// nicht geprüft wird.
	Today          string
	ReadOnlyUntil  string
	ReadOnlyReason string
}

// TaskService stellt die Aufgabenliste zusammen.
type TaskService struct {
	settingsRepo domain.SettingsRepository

	checks        TaskCheckSource
	deadlines     TaskDeadlineSource
	statements    StatementDeadlineSource
	bank          TaskBankSource
	receipts      TaskReceiptSource
	openItems     LiveOpenItemSource
	assetDocs     TaskAssetDocumentSource
	exemptions    ExemptionDeadlineSource
	backups       TaskBackupSource
	carryForward  TaskCarryForwardSource
	appropriation TaskAppropriationSource

	fiscalYear int
}

// NewTaskService baut die Aufgabenliste über den Einstellungen auf; die Quellen
// werden einzeln angehängt.
//
// Einzeln, weil jede für sich fehlen darf: ein Mandant ohne Anlagenkartei hat
// keine ablaufenden Dokumente, und eine Aufgabenliste, die deshalb gar nicht
// entsteht, wäre der schlechteste Umgang damit.
func NewTaskService(settingsRepo domain.SettingsRepository, fiscalYear int) *TaskService {
	return &TaskService{settingsRepo: settingsRepo, fiscalYear: fiscalYear}
}

// SetFiscalYear updates the year the list is built for.
func (s *TaskService) SetFiscalYear(year int) { s.fiscalYear = year }

func (s *TaskService) SetCheckSource(src TaskCheckSource)             { s.checks = src }
func (s *TaskService) SetDeadlineSource(src TaskDeadlineSource)       { s.deadlines = src }
func (s *TaskService) SetStatementSource(src StatementDeadlineSource) { s.statements = src }
func (s *TaskService) SetBankSource(src TaskBankSource)               { s.bank = src }
func (s *TaskService) SetReceiptSource(src TaskReceiptSource)         { s.receipts = src }
func (s *TaskService) SetOpenItemSource(src LiveOpenItemSource)       { s.openItems = src }
func (s *TaskService) SetAssetDocumentSource(src TaskAssetDocumentSource) {
	s.assetDocs = src
}
func (s *TaskService) SetExemptionSource(src ExemptionDeadlineSource) { s.exemptions = src }
func (s *TaskService) SetBackupSource(src TaskBackupSource)           { s.backups = src }
func (s *TaskService) SetCarryForwardSource(src TaskCarryForwardSource) {
	s.carryForward = src
}
func (s *TaskService) SetAppropriationSource(src TaskAppropriationSource) {
	s.appropriation = src
}

// BackupOverdueDays ist der Abstand, nach dem eine fehlende Sicherung zur
// Aufgabe wird. Buchfink sichert täglich (Architektur 6.6); nach drei Tagen ohne
// Sicherung ist etwas nicht in Ordnung, und zwar so, dass es auffallen muss,
// bevor die Platte es entscheidet.
const BackupOverdueDays = 3

// Tasks stellt die Aufgabenliste zusammen.
//
// Fällt eine Quelle mit einem Fehler aus, fehlt ihre Zeile — die Liste entsteht
// trotzdem. Das ist Absicht: der erste Bildschirm nach dem Start darf nicht
// daran scheitern, dass ein einzelner Dienst nicht antwortet.
func (s *TaskService) Tasks(ctx context.Context, opts TaskOptions) (*domain.TaskList, error) {
	today := opts.Today
	if today == "" {
		today = todayLocal()
	}
	horizon := addDays(today, domain.TaskLookaheadDays)

	list := &domain.TaskList{Today: today}
	list.EnsureLists()

	cfg, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}

	s.addReadOnly(list, opts)
	// Die Fristen zuerst: die Zeile „Januar 2026 festschreiben" entsteht dort,
	// und die Befunde des Prüflaufs zur Festschreibung desselben Monats sind
	// dann dieselbe Arbeit ein zweites Mal (siehe addCheckFindings).
	s.addDeadlines(ctx, list, today, horizon)
	s.addCheckFindings(ctx, list, today)
	s.addBank(ctx, list)
	s.addReceipts(ctx, list, today, cfg)
	s.addOverdueReceivables(ctx, list, today)
	s.addAssetDocuments(ctx, list, today, horizon)
	s.addExemptions(ctx, list, today)
	s.addBackup(ctx, list, today)
	s.addYearEnd(ctx, list, today)

	list.Sort()
	list.EnsureLists()
	return list, nil
}

func (s *TaskService) settings(ctx context.Context) (*domain.CompanySettings, error) {
	if s.settingsRepo == nil {
		return &domain.CompanySettings{ReceiptCaptureDays: 10}, nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Unternehmensdaten konnten nicht gelesen werden: %w", err)
	}
	return cfg, nil
}

// addReadOnly meldet den laufenden Prüfermodus.
//
// Als Aufgabe und nicht nur als Banner: der Modus schaltet die Buchführung
// schreibgeschützt, und wer nicht weiß, warum das Buchen abgewiesen wird, sucht
// den Fehler im Programm.
func (s *TaskService) addReadOnly(list *domain.TaskList, opts TaskOptions) {
	if opts.ReadOnlyUntil == "" {
		return
	}
	why := fmt.Sprintf("Bis zum %s nimmt die Buchführung keine Änderung auf.", opts.ReadOnlyUntil)
	if opts.ReadOnlyReason != "" {
		why += " Grund: " + opts.ReadOnlyReason + "."
	}
	list.Add(domain.Task{
		Key: domain.TaskKeyReadOnly, Group: domain.TaskGroupOpen,
		Title:     "Prüfermodus ist eingeschaltet",
		Why:       why + " Der Zugriff des Prüfers wird protokolliert (§ 147 Abs. 6 AO).",
		Reference: "§ 147 Abs. 6 AO",
		DueDate:   opts.ReadOnlyUntil,
		Target:    domain.TaskTarget{Page: "taxaudit"},
	})
}

// addCheckFindings macht aus den Befunden des Prüflaufs eine Zeile je Regel.
//
// Je Regel und nicht je Befund: fünfzig Belege ohne Buchung sind eine Aufgabe
// und nicht fünfzig. Die Regel hat den Schlüssel, damit die Oberfläche zur
// zugehörigen Stelle springen kann.
func (s *TaskService) addCheckFindings(ctx context.Context, list *domain.TaskList, today string) {
	if s.checks == nil {
		return
	}
	run, err := s.checks.Preview(ctx, CheckRequest{CutoffDate: today})
	if err != nil || run == nil {
		return
	}
	counts := map[string]int{}
	blocking := map[string]bool{}
	order := make([]string, 0, 8)
	for _, finding := range run.Findings {
		if _, seen := counts[finding.Rule]; !seen {
			order = append(order, finding.Rule)
		}
		counts[finding.Rule]++
		if finding.Severity == domain.CheckBlocking {
			blocking[finding.Rule] = true
		}
	}
	for _, rule := range order {
		// Einige Regeln haben eine eigene, genauere Zeile aus der Quelle selbst
		// (Bankumsätze, Belege ohne Buchung, fehlender Leistungsnachweis). Sie
		// hier ein zweites Mal zu zeigen, hieße dieselbe Arbeit zweimal
		// aufzuschreiben.
		if rule == domain.CheckRuleBankUnmatched || rule == domain.CheckRuleReceiptUnbooked ||
			rule == domain.CheckRuleReceiptOverdue || rule == domain.CheckRuleServiceProofMissing {
			continue
		}
		// Die Festschreibung eines Monats steht schon als Fristzeile in der
		// Liste („Januar 2026 festschreiben", aus commitDeadlines). Die beiden
		// Prüfregeln dazu meinen dieselbe Arbeit; als eigene Zeile stünde sie
		// zweimal da, einmal mit Monat und einmal ohne.
		if rule == domain.CheckRulePeriodNotCommitted || rule == domain.CheckRuleCommitOverdue {
			if hasCommitDeadline(list) {
				continue
			}
		}
		group := domain.TaskGroupOpen
		if blocking[rule] {
			group = domain.TaskGroupOverdue
		}
		title, why, reference := checkRuleTask(rule)
		list.Add(domain.Task{
			Key: domain.TaskKeyCheckFindings + "." + rule, Group: group,
			Title: title, Why: why, Reference: reference,
			Count:  counts[rule],
			Target: domain.TaskTarget{Page: "audit", Params: map[string]string{"rule": rule}},
		})
	}
}

// hasCommitDeadline meldet, ob die Fristzeile einer Monatsfestschreibung schon
// in der Liste steht.
func hasCommitDeadline(list *domain.TaskList) bool {
	prefix := domain.TaskKeyDeadline + "." + DeadlineKeyCommit + "."
	for _, group := range [][]domain.Task{list.Overdue, list.Open, list.Upcoming} {
		for _, task := range group {
			if strings.HasPrefix(task.Key, prefix) {
				return true
			}
		}
	}
	return false
}

// checkRuleTask übersetzt eine Prüfregel in Vorgangssprache.
//
// Der Titel nennt keinen Paragraphen (Architektur 6.4); die Norm steht in der
// Fundstelle und erscheint erst in der zweiten Erklärstufe.
func checkRuleTask(rule string) (title, why, reference string) {
	switch rule {
	case domain.CheckRuleEntryWithoutReceipt:
		return "Buchungen ohne Beleg klären",
			"Zu jeder Buchung gehört ein Beleg; ohne ihn ist der Geschäftsvorfall nicht nachvollziehbar.",
			"§ 146 Abs. 1 AO, GoBD Rz. 61"
	case domain.CheckRuleInterimBalance:
		return "Interimskonten ausgleichen",
			"Ein Verrechnungskonto mit Saldo bedeutet: ein Vorgang ist noch nicht zu Ende gebucht.",
			"GoBD Rz. 46"
	case domain.CheckRuleDuplicateReceipt:
		return "Mögliche Doppelbelege ansehen",
			"Zweimal derselbe Beleg heißt zweimal Vorsteuer — und einmal zu viel gezahlt.",
			"§ 15 UStG"
	case domain.CheckRuleDuplicatePayment:
		return "Mögliche Doppelzahlungen ansehen",
			"Derselbe Betrag an denselben Partner am selben Tag ist selten Absicht.",
			""
	case domain.CheckRuleNumberGap:
		return "Lücken im Nummernkreis begründen",
			"Eine fehlende Rechnungsnummer muss erklärt sein, sonst sieht sie aus wie ein unterdrückter Umsatz.",
			"§ 14 Abs. 4 Nr. 4 UStG, GoBD Rz. 36"
	case domain.CheckRuleAccountUnmapped:
		return "Konten der Bilanz zuordnen",
			"Ein bebuchtes Konto ohne Bilanzposition taucht im Abschluss nicht auf.",
			"§§ 266, 275 HGB"
	case domain.CheckRuleDepreciationMissing:
		return "Abschreibung nachholen",
			"Ein Anlagegut ohne Abschreibung des Jahres steht mit einem zu hohen Wert in der Bilanz.",
			"§ 253 Abs. 3 HGB, § 7 EStG"
	case domain.CheckRuleVatReturnMissing:
		return "Voranmeldung nachholen",
			"Für einen Zeitraum ohne Anmeldung setzt das Finanzamt die Steuer selbst fest.",
			"§ 18 Abs. 1 UStG"
	case domain.CheckRuleCommitOverdue:
		return "Zeitraum festschreiben",
			"Buchungen sind bis zum Ablauf des Folgemonats unveränderbar zu stellen.",
			"GoBD Rz. 107"
	case domain.CheckRulePeriodNotCommitted:
		return "Monat festschreiben",
			"Zwei Monate nach dem Ende des Zeitraums ist auch die Voranmeldung mit " +
				"Dauerfristverlängerung abgegeben — bis dahin sollten die Buchungen unveränderbar " +
				"gestellt sein.",
			"GoBD Rz. 107, § 146 Abs. 4 AO"
	case domain.CheckRuleProvisionDiscount:
		return "Abzinsung der Rückstellungen prüfen",
			"Ohne den Satz des Stichtagsmonats ist die Rückstellung nicht bewertet.",
			"§ 253 Abs. 2 HGB"
	case domain.CheckRuleClosingStepSkipped:
		return "Übersprungene Abschlussschritte begründen",
			"Ein übergangener Baustein des Abschlusses ist eine Aussage, und sie gehört in den Bericht.",
			""
	case domain.CheckRuleSizeClassChange:
		return "Größenklassenwechsel vorbereiten",
			"Prüferbestellung, Gliederungstiefe und Aufstellungsfrist ändern sich, sobald der zweite " +
				"Stichtag dieselbe Klasse ergibt — das lässt sich nicht in dem Monat vorbereiten, in dem es gilt.",
			"§ 267 Abs. 4 Satz 1 HGB"
	case domain.CheckRuleICSupplyEvidenceMissing:
		return "Belegnachweis der Auslandslieferungen führen",
			"Ohne den Nachweis der Warenbewegung ist die steuerfreie Lieferung ins EU-Ausland steuerpflichtig.",
			"§§ 17a ff. UStDV"
	case domain.CheckRuleICSupplyUnconfirmed:
		return "Bestätigung der USt-IdNr. nachholen",
			"Die gültige Nummer des Abnehmers ist Voraussetzung der Steuerbefreiung, nicht eine Formalie.",
			"§ 6a Abs. 1 Satz 1 Nr. 4 UStG, § 18e UStG"
	default:
		return "Prüfbefunde ansehen",
			"Der Prüflauf hat etwas gefunden, das vor der Festschreibung geklärt sein sollte.",
			"GoBD Rz. 100 ff."
	}
}

// addDeadlines übernimmt die Fristen: verstrichene nach oben, die der nächsten
// dreißig Tage nach unten. Was weiter weg ist, steht auf der Fristenseite.
//
// Gefragt werden zwei Jahre. Die Fristen eines Jahres werden aus seinen
// Zeiträumen gebildet, fällig sind einige davon erst im Folgejahr: die
// Voranmeldung für Dezember ist am 10. Januar fällig, die Festschreibung des
// Dezembers Ende Januar. Wer im Januar nur das neue Geschäftsjahr abfragte,
// sähe im Monat der meisten Fristen die wenigsten. Aus dem Vorjahr kommt
// deshalb mit, was jetzt noch aussteht — nicht sein ganzer Rückstand.
func (s *TaskService) addDeadlines(ctx context.Context, list *domain.TaskList, today, horizon string) {
	if s.deadlines == nil {
		return
	}
	deadlines, err := s.deadlines.Deadlines(ctx, s.fiscalYear)
	if err != nil {
		return
	}
	if previous, err := s.deadlines.Deadlines(ctx, s.fiscalYear-1); err == nil {
		// Die Grenze nach unten ist derselbe Vorlauf, mit dem die Liste nach
		// oben blickt: eine Frist, die länger als dreißig Tage verstrichen ist,
		// gehört auf die Fristenseite und nicht auf den ersten Bildschirm.
		earliest := addDays(today, -domain.TaskLookaheadDays)
		for _, deadline := range previous {
			if deadline.DueDate >= earliest {
				deadlines = append(deadlines, deadline)
			}
		}
	}
	seen := map[string]bool{}
	for _, deadline := range deadlines {
		if deadline.IsDone || deadline.DueDate == "" {
			continue
		}
		// Die Festschreibung eines Monats ist schon vor ihrer Frist Arbeit
		// (UNV-02 K2): ab dem 10. des Folgemonats ist die Voranmeldung
		// abgegeben, und von da an gibt es keinen Grund mehr, den Monat offen
		// zu lassen. Sie steht deshalb ab diesem Tag als offene Aufgabe in der
		// Liste — auch dann, wenn ihre Frist mit Dauerfristverlängerung weiter
		// als der Vorlauf entfernt liegt.
		openFrom := commitOpenFrom(deadline.Key)
		visible := openFrom != "" && today >= openFrom
		if deadline.DueDate > horizon && !visible {
			continue
		}
		// Zwei Jahre können denselben Termin führen (ein Jahresschlüssel steht
		// im Schlüssel, ein Stichtagsschlüssel nicht). Zweimal dieselbe Zeile
		// wäre zweimal dieselbe Arbeit.
		if seen[deadline.Key] {
			continue
		}
		seen[deadline.Key] = true
		group := domain.TaskGroupUpcoming
		if visible {
			group = domain.TaskGroupOpen
		}
		if deadline.DueDate < today {
			group = domain.TaskGroupOverdue
		}
		list.Add(domain.Task{
			Key: domain.TaskKeyDeadline + "." + deadline.Key, Group: group,
			Title:     deadline.Title,
			Why:       deadline.Description,
			Reference: deadline.Reference,
			DueDate:   deadline.DueDate,
			Target: domain.TaskTarget{
				Page: "deadlines", Params: map[string]string{"key": deadline.Key},
			},
		})
	}
}

// commitOpenFrom ist der Tag, ab dem die Festschreibung eines Monats als offene
// Aufgabe geführt wird: der 10. des Folgemonats (UNV-02 K2).
//
// Der 10. und nicht der Monatserste: bis zum 10. ist die Voranmeldung des
// Monats abzugeben (§ 18 Abs. 1 Satz 1 UStG), und bis dahin wird an dem Monat
// noch gebucht. Für jede andere Frist ergibt sich nichts — die Funktion liefert
// dann den leeren Tag, und die Frist bleibt eine Frist.
func commitOpenFrom(key string) string {
	rest, ok := strings.CutPrefix(key, DeadlineKeyCommit+".")
	if !ok {
		return ""
	}
	p, err := accounting.ParseVatPeriodKey(rest)
	if err != nil {
		return ""
	}
	last, err := time.Parse("2006-01-02", p.To)
	if err != nil {
		return ""
	}
	// Der letzte Tag des Zeitraums plus ein Tag ist der Monatserste des
	// Folgemonats; plus neun weitere ist sein zehnter.
	return last.AddDate(0, 0, 10).Format("2006-01-02")
}

// addBank meldet die Bankumsätze ohne Zuordnung.
func (s *TaskService) addBank(ctx context.Context, list *domain.TaskList) {
	if s.bank == nil {
		return
	}
	transactions, err := s.bank.GetTransactions(ctx, s.fiscalYear)
	if err != nil {
		return
	}
	count := 0
	var amount domain.Cents
	for i := range transactions {
		if transactions[i].MatchStatus != domain.MatchStatusUnmatched {
			continue
		}
		count++
		amount += transactions[i].Amount.Abs()
	}
	if count == 0 {
		return
	}
	list.Add(domain.Task{
		Key: domain.TaskKeyBankUnmatched, Group: domain.TaskGroupOpen,
		Title:     "Bankumsätze zuordnen",
		Why:       "Ein Umsatz ohne Zuordnung ist ein Geschäftsvorfall, der noch nicht gebucht ist.",
		Reference: "§ 146 Abs. 1 AO",
		Count:     count, Amount: amount,
		Target: domain.TaskTarget{Page: "bank"},
	})
}

// addReceipts meldet die abgelegten Belege ohne Buchung — und die, die länger
// liegen als die Erfassungsfrist.
func (s *TaskService) addReceipts(
	ctx context.Context, list *domain.TaskList, today string, cfg *domain.CompanySettings,
) {
	if s.receipts == nil {
		return
	}
	// Alle Belege des Jahres und nicht nur die abgelegten: der
	// Leistungsnachweis fehlt auch an einem gebuchten Beleg, und gerade dort
	// fällt er sonst niemandem mehr auf.
	receipts, err := s.receipts.List(ctx, "")
	if err != nil {
		return
	}
	limit := cfg.ReceiptCaptureDays
	if limit <= 0 {
		limit = 10
	}
	deadline := addDays(today, -limit)

	open, late := 0, 0
	var openAmount, lateAmount domain.Cents
	oldest := ""
	// Der Prüfvermerk gehört zum Beleg und nicht zum Prüflauf: er ist eine Angabe,
	// die jemand machen muss, und keine Regel, die etwas findet.
	missingProof := 0
	var proofAmount domain.Cents
	// Der Beleg, den die Aufgabe aufschlägt: der älteste ohne Vermerk.
	//
	// Ein Filter täte es nicht — die Belegliste kennt „zu buchen" und „zu
	// klären", und beide zeigen den gebuchten Beleg gerade nicht, an dem der
	// Vermerk am häufigsten fehlt. Ein Zustandsfilter führte deshalb auf eine
	// Liste ohne den gemeinten Beleg. Sind es mehrere, führt der Weg zum
	// ältesten; die Aufgabe bleibt stehen, bis alle einen Vermerk haben.
	var proofReceipt *domain.Receipt
	proofReference := ""

	for i := range receipts {
		receipt := &receipts[i]
		if receipt.Status == domain.ReceiptStatusDiscarded {
			continue
		}
		if s.needsServiceProof(receipt, cfg) {
			missingProof++
			proofAmount += receipt.GrossAmount
			when := receipt.ReceivedAt
			if when == "" {
				when = receipt.DocumentDate
			}
			if proofReceipt == nil || when < proofReference ||
				(when == proofReference && receipt.ID < proofReceipt.ID) {
				proofReceipt = receipt
				proofReference = when
			}
		}
		if receipt.JournalEntryID != nil || receipt.Status == domain.ReceiptStatusSealed {
			continue
		}
		open++
		openAmount += receipt.GrossAmount
		reference := receipt.ReceivedAt
		if reference == "" {
			reference = receipt.DocumentDate
		}
		if reference != "" && reference <= deadline {
			late++
			lateAmount += receipt.GrossAmount
			if oldest == "" || reference < oldest {
				oldest = reference
			}
		}
	}

	if open > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyReceiptsUnbooked, Group: domain.TaskGroupOpen,
			Title:     "Belege buchen",
			Why:       "Ein abgelegter Beleg ohne Buchung ist ein Geschäftsvorfall, den die Buchführung noch nicht kennt.",
			Reference: "§ 146 Abs. 1 AO",
			Count:     open, Amount: openAmount,
			Target: domain.TaskTarget{Page: "receipts", Params: map[string]string{"status": "filed"}},
		})
	}
	if late > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyReceiptsOverdue, Group: domain.TaskGroupOverdue,
			Title: fmt.Sprintf("Belege buchen, die länger als %d Tage liegen", limit),
			Why: fmt.Sprintf(
				"Unbare Geschäftsvorfälle sind innerhalb von %d Tagen zu erfassen; der älteste liegt seit dem %s.",
				limit, oldest),
			Reference: "GoBD Rz. 47",
			Count:     late, Amount: lateAmount, DueDate: oldest,
			Target: domain.TaskTarget{Page: "receipts", Params: map[string]string{"status": "filed"}},
		})
	}
	if missingProof > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyServiceProof, Group: domain.TaskGroupOpen,
			Title: "Leistungsnachweis zu größeren Eingangsbelegen erfassen",
			Why: fmt.Sprintf(
				"Ab %s € verlangt die eigene Vorgabe den Vermerk, gegen welche Bestellung geprüft wurde; ohne ihn belegt nichts, dass die Leistung erbracht wurde.",
				cfg.InvoiceCheckThreshold),
			Reference: "§ 15 UStG",
			Count:     missingProof, Amount: proofAmount,
			Target: domain.TaskTarget{
				Page:   "receipts",
				Params: map[string]string{"receiptId": fmt.Sprintf("%d", proofReceipt.ID)},
			},
		})
	}
}

// needsServiceProof meldet, ob ein Eingangsbeleg über der Grenze noch keinen
// Leistungsnachweis hat.
//
// Dieselbe Regel, die den Beleg nicht buchen lässt (siehe service_proof.go):
// die Aufgabenliste führt genau die Belege, an denen das Buchen scheitern würde.
func (s *TaskService) needsServiceProof(receipt *domain.Receipt, cfg *domain.CompanySettings) bool {
	return receiptNeedsServiceProof(receipt, cfg.InvoiceCheckThreshold)
}

// addOverdueReceivables meldet die überfälligen Forderungen.
func (s *TaskService) addOverdueReceivables(ctx context.Context, list *domain.TaskList, today string) {
	if s.openItems == nil {
		return
	}
	items, err := s.openItems.OpenItems(ctx)
	if err != nil {
		return
	}
	count := 0
	var amount domain.Cents
	oldest := ""
	for i := range items {
		item := &items[i]
		if item.ContactType != domain.ContactTypeCustomer || !item.IsOverdue(today) {
			continue
		}
		count++
		amount += item.OpenAmount
		if oldest == "" || item.DueDate < oldest {
			oldest = item.DueDate
		}
	}
	if count == 0 {
		return
	}
	list.Add(domain.Task{
		Key: domain.TaskKeyOverdueItems, Group: domain.TaskGroupOverdue,
		Title:     "Überfällige Kundenrechnungen mahnen",
		Why:       "Nach Ablauf von dreißig Tagen ab Fälligkeit kommt der Kunde auch ohne Mahnung in Verzug; ab dann laufen Verzugszinsen.",
		Reference: "§§ 286, 288 BGB",
		Count:     count, Amount: amount, DueDate: oldest,
		Target: domain.TaskTarget{Page: "bank", Params: map[string]string{"view": "dunning"}},
	})
}

// addAssetDocuments meldet die ablaufenden Dokumente am Anlagegut.
func (s *TaskService) addAssetDocuments(ctx context.Context, list *domain.TaskList, today, horizon string) {
	if s.assetDocs == nil {
		return
	}
	documents, err := s.assetDocs.ExpiringDocuments(ctx, horizon)
	if err != nil || len(documents) == 0 {
		return
	}
	expired, upcoming := 0, 0
	first := ""
	for _, document := range documents {
		if document.ValidUntil < today {
			expired++
		} else {
			upcoming++
		}
		if first == "" || document.ValidUntil < first {
			first = document.ValidUntil
		}
	}
	if expired > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyAssetDocument + ".expired", Group: domain.TaskGroupOverdue,
			Title:   "Abgelaufene Unterlagen am Anlagegut erneuern",
			Why:     "Eine abgelaufene Police oder ein fälliges Darlehen am Anlagegut ist ein Vorgang, der nicht warten kann.",
			Count:   expired,
			DueDate: first,
			Target:  domain.TaskTarget{Page: "assets"},
		})
	}
	if upcoming > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyAssetDocument, Group: domain.TaskGroupUpcoming,
			Title:   "Unterlagen am Anlagegut laufen aus",
			Why:     "Die Frist an einem hinterlegten Dokument endet in den nächsten dreißig Tagen.",
			Count:   upcoming,
			DueDate: first,
			Target:  domain.TaskTarget{Page: "assets"},
		})
	}
}

// addExemptions meldet die ablaufenden Freistellungsbescheinigungen.
func (s *TaskService) addExemptions(ctx context.Context, list *domain.TaskList, today string) {
	if s.exemptions == nil {
		return
	}
	warnings, err := s.exemptions.ExemptionCertificateWarnings(ctx, today)
	if err != nil || len(warnings) == 0 {
		return
	}
	expired, expiring := 0, 0
	first := ""
	for _, warning := range warnings {
		if warning.State == "expired" {
			expired++
		} else {
			expiring++
		}
		if first == "" || warning.ValidUntil < first {
			first = warning.ValidUntil
		}
	}
	if expired > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyExemption + ".expired", Group: domain.TaskGroupOverdue,
			Title:     "Abgelaufene Freistellungsbescheinigung anfordern",
			Why:       "Ohne gültige Bescheinigung sind bei einer Bauleistung 15 % der Gegenleistung einzubehalten und abzuführen.",
			Reference: "§§ 48, 48b EStG",
			Count:     expired, DueDate: first,
			Target: domain.TaskTarget{Page: "contacts"},
		})
	}
	if expiring > 0 {
		list.Add(domain.Task{
			Key: domain.TaskKeyExemption, Group: domain.TaskGroupUpcoming,
			Title:     "Freistellungsbescheinigung läuft aus",
			Why:       "Die Bescheinigung eines Geschäftspartners endet in den nächsten dreißig Tagen.",
			Reference: "§ 48b EStG",
			Count:     expiring, DueDate: first,
			Target: domain.TaskTarget{Page: "contacts"},
		})
	}
}

// addBackup meldet die fehlende oder fehlgeschlagene Sicherung.
func (s *TaskService) addBackup(ctx context.Context, list *domain.TaskList, today string) {
	if s.backups == nil {
		return
	}
	s.addFailedBackup(ctx, list)
	last, err := s.backups.LastSuccessful(ctx)
	if err != nil {
		return
	}
	if last == nil {
		list.Add(domain.Task{
			Key: domain.TaskKeyBackupMissing, Group: domain.TaskGroupOverdue,
			Title:     "Sicherung einrichten",
			Why:       "Es gibt keine erfolgreiche Sicherung. Aufzeichnungen müssen über die Aufbewahrungsfrist lesbar bleiben.",
			Reference: "§ 147 Abs. 2 AO, GoBD Rz. 103 ff.",
			Target:    domain.TaskTarget{Page: "backup"},
		})
		return
	}
	lastDay := last.StartedAt.Format("2006-01-02")
	if lastDay >= addDays(today, -BackupOverdueDays) {
		return
	}
	list.Add(domain.Task{
		Key: domain.TaskKeyBackupMissing, Group: domain.TaskGroupOverdue,
		Title: "Sicherung nachholen",
		Why: fmt.Sprintf(
			"Die letzte erfolgreiche Sicherung stammt vom %s. Buchfink sichert täglich; bleibt sie aus, stimmt etwas am Zielordner nicht.",
			lastDay),
		Reference: "§ 147 Abs. 2 AO",
		DueDate:   lastDay,
		Target:    domain.TaskTarget{Page: "backup"},
	})
}

// addFailedBackup meldet den fehlgeschlagenen jüngsten Lauf.
//
// Gefragt wird nur der jüngste: ein Fehlschlag vor drei Wochen, dem seitdem
// zwanzig gelungene Läufe gefolgt sind, ist erledigt und keine Aufgabe. Ist
// dagegen der letzte Versuch gescheitert, ist die Sicherung heute kaputt — auch
// wenn vorgestern noch eine gelang und die Frist deshalb noch nicht abgelaufen
// ist.
func (s *TaskService) addFailedBackup(ctx context.Context, list *domain.TaskList) {
	runs, err := s.backups.GetRuns(ctx, 1)
	if err != nil || len(runs) == 0 {
		return
	}
	last := runs[0]
	if last.Success {
		return
	}
	// Die Wiederherstellung selbst ist ein Vorgang des Anwenders und keine
	// laufende Sicherung; ihr Fehlschlag steht in ihrem eigenen Bericht.
	if last.Kind == domain.BackupKindRestore {
		return
	}
	what := "Die Sicherung"
	if last.Kind == domain.BackupKindVerify {
		what = "Der Wiederherstellungstest der Sicherung"
	}
	why := fmt.Sprintf("%s vom %s ist fehlgeschlagen.", what, last.StartedAt.Format("2006-01-02"))
	if last.Message != "" {
		why += " Grund: " + last.Message
	}
	list.Add(domain.Task{
		Key: domain.TaskKeyBackupFailed, Group: domain.TaskGroupOverdue,
		Title: "Sicherung ist fehlgeschlagen",
		Why: why + " Aufzeichnungen müssen über die Aufbewahrungsfrist lesbar bleiben; " +
			"ein Zielordner, der nicht mehr beschreibbar ist, fällt sonst erst auf, wenn die Sicherung gebraucht wird.",
		Reference: "§ 147 Abs. 2 AO, GoBD Rz. 103 ff.",
		DueDate:   last.StartedAt.Format("2006-01-02"),
		Target:    domain.TaskTarget{Page: "backup"},
	})
}

// addYearEnd meldet die drei Aufgaben am Jahreswechsel: die Vortragsdifferenz,
// den nicht festgestellten Vorjahresabschluss und die offene Ergebnisverwendung.
func (s *TaskService) addYearEnd(ctx context.Context, list *domain.TaskList, today string) {
	priorYear := s.fiscalYear - 1

	if s.carryForward != nil {
		if preview, err := s.carryForward.CarryForwardState(ctx, s.fiscalYear); err == nil && preview != nil {
			switch {
			case !preview.IsBalanced:
				list.Add(domain.Task{
					Key: domain.TaskKeyCarryForward + ".unbalanced", Group: domain.TaskGroupOverdue,
					Title: "Differenz im Saldenvortrag klären",
					Why: fmt.Sprintf(
						"Die Vortragswerte aus %d gehen um %s € nicht auf. Ein Vortrag trüge den Fehler ins neue Jahr.",
						priorYear, preview.BalanceDifference),
					Reference: "§ 252 Abs. 1 Nr. 1 HGB",
					Amount:    preview.BalanceDifference,
					Target:    domain.TaskTarget{Page: "closing"},
				})
			case !preview.AlreadyCarried && len(preview.Rows) > 0:
				list.Add(domain.Task{
					Key: domain.TaskKeyCarryForward, Group: domain.TaskGroupOpen,
					Title:     "Saldenvortrag ins neue Jahr buchen",
					Why:       "Die Schlussbilanzwerte des Vorjahres sind als Eröffnungsbilanzwerte zu übernehmen (Bilanzidentität).",
					Reference: "§ 252 Abs. 1 Nr. 1 HGB",
					Count:     len(preview.Rows),
					Target:    domain.TaskTarget{Page: "closing"},
				})
			case preview.NeedsCorrection:
				list.Add(domain.Task{
					Key: domain.TaskKeyCarryForward + ".correction", Group: domain.TaskGroupOpen,
					Title:     "Saldenvortrag berichtigen",
					Why:       "Im Vorjahr wurde nach dem Vortrag noch gebucht; die vorgetragenen Werte sind überholt.",
					Reference: "§ 252 Abs. 1 Nr. 1 HGB",
					Target:    domain.TaskTarget{Page: "closing"},
				})
			}
		}
	}

	priorState := s.priorYearState(ctx, priorYear)
	if priorState == nil {
		return
	}

	if priorState.Status != domain.FiscalYearAdopted && priorState.Status != domain.FiscalYearDisclosed {
		if due := s.preparationDeadline(ctx, priorYear); due != "" && due < today {
			list.Add(domain.Task{
				Key: domain.TaskKeyPriorYearOpen, Group: domain.TaskGroupOverdue,
				Title: fmt.Sprintf("Jahresabschluss %d feststellen", priorYear),
				Why: fmt.Sprintf(
					"Die Aufstellungsfrist ist am %s abgelaufen, und der Abschluss ist noch nicht festgestellt. Bis dahin steht das Ergebnis nicht fest.",
					due),
				Reference: "§ 264 Abs. 1 HGB, § 42a Abs. 2 GmbHG",
				DueDate:   due,
				Target:    domain.TaskTarget{Page: "closing", Params: map[string]string{"year": fmt.Sprintf("%d", priorYear)}},
			})
		}
		return
	}

	// Erst wenn der Abschluss festgestellt ist, kann über das Ergebnis
	// beschlossen werden — vorher gibt es nichts zu verwenden.
	if s.appropriation == nil {
		return
	}
	decision, err := s.appropriation.Appropriation(ctx, priorYear)
	if err != nil || decision != nil {
		return
	}
	list.Add(domain.Task{
		Key: domain.TaskKeyAppropriationOpen, Group: domain.TaskGroupOpen,
		Title:     fmt.Sprintf("Ergebnisverwendung %d beschließen", priorYear),
		Why:       "Der festgestellte Abschluss braucht den Beschluss der Gesellschafter darüber, was mit dem Ergebnis geschieht.",
		Reference: "§ 29 GmbHG, § 42a Abs. 2 GmbHG",
		Target:    domain.TaskTarget{Page: "closingmodules", Params: map[string]string{"year": fmt.Sprintf("%d", priorYear)}},
	})
}

// priorYearState liefert das Vorjahr aus der Geschäftsjahresliste, oder nil.
func (s *TaskService) priorYearState(ctx context.Context, priorYear int) *domain.FiscalYear {
	if s.carryForward == nil {
		return nil
	}
	years, err := s.carryForward.FiscalYears(ctx)
	if err != nil {
		return nil
	}
	for i := range years {
		if years[i].Year == priorYear {
			return &years[i]
		}
	}
	return nil
}

// preparationDeadline liefert die Aufstellungsfrist eines Jahres.
func (s *TaskService) preparationDeadline(ctx context.Context, year int) string {
	if s.statements == nil {
		return ""
	}
	deadlines, err := s.statements.Deadlines(ctx, year)
	if err != nil {
		return ""
	}
	for _, deadline := range deadlines {
		if deadline.Key == "abschluss.aufstellung" {
			return deadline.DueDate
		}
	}
	return ""
}
