package service

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// Das Mahnwesen (QUE-05).
//
// Der Dienst schlägt vor, rechnet und schreibt — er bucht nicht. Mahngebühr und
// Verzugszinsen sind erst mit der Zahlung Ertrag: wer sie bei der Mahnung
// einbuchte, wiese eine Forderung aus, die in den meisten Fällen nie fließt, und
// müsste sie später wieder abschreiben. Deshalb steht der Betrag im Schreiben
// und nicht im Journal, und deshalb sagt der Hinweis das auch.

// DunningCategory ist der Zweig im Belegspeicher, unter dem die Schreiben
// liegen (dokumente/mahnungen).
const DunningCategory = "mahnungen"

// DunningPaymentDays ist die Frist, die ein Mahnschreiben voreingestellt setzt.
const DunningPaymentDays = 14

// DunningProposalItem ist ein gemahnter Posten im Vorschlag.
type DunningProposalItem struct {
	EntryID        uint         `json:"entryId"`
	DocumentNumber string       `json:"documentNumber"`
	DocumentDate   string       `json:"documentDate"`
	DueDate        string       `json:"dueDate"`
	OpenAmount     domain.Cents `json:"openAmount"`
	// DaysOverdue ist der Abstand zur Fälligkeit, DefaultFrom der erste Tag des
	// Verzugs (dreißig Tage danach), InterestDays die verzinsten Tage.
	//
	// Verzinst wird der heute offene Betrag über den ganzen Verzugszeitraum.
	// Eine Teilzahlung mitten im Verzug bleibt damit außer Betracht: gerechnet
	// wird auf dem Rest, obwohl bis zur Teilzahlung mehr offen war. Die
	// Vereinfachung geht immer zu Lasten des Gläubigers — die Zinsforderung ist
	// eher zu niedrig als zu hoch —, und der Hinweis unter dem Vorschlag sagt es.
	// Taggenau je Teilzahlung zu rechnen setzte eine Historie des offenen
	// Betrages voraus; sie steht in den Zahlungszuordnungen und wäre der
	// nächste Schritt.
	DaysOverdue   int          `json:"daysOverdue"`
	DefaultFrom   string       `json:"defaultFrom"`
	InterestDays  int          `json:"interestDays"`
	Interest      domain.Cents `json:"interest"`
	Level         int          `json:"level"`
	PreviousLevel int          `json:"previousLevel"`
	// LumpSum ist die Pauschale, die auf diesen Posten entfällt.
	LumpSum domain.Cents `json:"lumpSum"`
	// Note sagt, was an diesem Posten nicht gerechnet werden konnte — etwa ein
	// Basiszinssatz, der für den Zeitraum fehlt. Ein Vorschlag, der die Zinsen
	// still mit null ausweist, führt zu einer zu niedrigen Forderung, und
	// niemand sähe, woran es lag.
	Note string `json:"note,omitempty"`
}

// DunningProposal ist der Mahnvorschlag für einen Kunden.
type DunningProposal struct {
	ContactID   uint   `json:"contactId"`
	ContactName string `json:"contactName"`
	// IsConsumer entscheidet über Zinssatz und Pauschale. Er folgt aus den
	// Stammdaten (Contact.IsPrivate) und ist deshalb kein zweites Feld am
	// Kontakt: ein Geschäftspartner, der kein Unternehmer ist, ist der
	// Verbraucher des § 13 BGB.
	IsConsumer bool `json:"isConsumer"`

	Level      int    `json:"level"`
	LevelLabel string `json:"levelLabel"`
	NoticeDate string `json:"noticeDate"`

	Items []DunningProposalItem `json:"items"`

	Principal domain.Cents `json:"principal"`
	Interest  domain.Cents `json:"interest"`
	Fee       domain.Cents `json:"fee"`
	LumpSum   domain.Cents `json:"lumpSum"`
	Total     domain.Cents `json:"total"`

	Note string `json:"note"`
}

// EnsureLists ersetzt eine nicht belegte Postenliste durch eine leere.
func (p *DunningProposal) EnsureLists() {
	if p.Items == nil {
		p.Items = make([]DunningProposalItem, 0)
	}
}

// DunningRequest ist der Auftrag, ein Mahnschreiben zu erzeugen.
type DunningRequest struct {
	ContactID uint `json:"contactId"`
	// NoticeDate ist das Datum des Schreibens; leer heißt heute.
	NoticeDate string `json:"noticeDate"`
	// PaymentDeadline ist die im Schreiben gesetzte Frist; leer heißt vierzehn
	// Tage nach dem Datum des Schreibens.
	PaymentDeadline string `json:"paymentDeadline"`
}

// DunningService führt den Mahnlauf.
type DunningService struct {
	openItems    LiveOpenItemSource
	contactRepo  domain.ContactRepository
	dunningRepo  domain.DunningRepository
	baseRateRepo domain.BaseRateRepository
	settingsRepo domain.SettingsRepository
	auditRepo    domain.AuditRepository
	store        *receiptstore.Store
	renderer     DocumentRenderer
	// invoiceRepo beantwortet die eine Frage, die das Mahnwesen an die
	// Rechnung stellt: hat sie gegenüber einem Verbraucher auf den Verzug nach
	// dreißig Tagen hingewiesen (§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB)? Ohne die
	// Quelle rechnet der Lauf wie zuvor.
	invoiceRepo domain.InvoiceRepository
	fiscalYear  int
}

// NewDunningService wires the Mahnlauf.
func NewDunningService(
	openItems LiveOpenItemSource,
	contactRepo domain.ContactRepository,
	dunningRepo domain.DunningRepository,
	baseRateRepo domain.BaseRateRepository,
	settingsRepo domain.SettingsRepository,
	auditRepo domain.AuditRepository,
	store *receiptstore.Store,
	fiscalYear int,
) *DunningService {
	return &DunningService{
		openItems: openItems, contactRepo: contactRepo, dunningRepo: dunningRepo,
		baseRateRepo: baseRateRepo, settingsRepo: settingsRepo, auditRepo: auditRepo,
		store: store, fiscalYear: fiscalYear,
	}
}

// SetRenderer hängt den Dokumentensetzer an. Ohne ihn entsteht das Schreiben
// ohne PDF; der Vorgang bleibt trotzdem verzeichnet.
func (s *DunningService) SetRenderer(r DocumentRenderer) { s.renderer = r }

// SetInvoiceSource hängt die Rechnungen an: sie haben das Kennzeichen, ob der
// Verzugshinweis an den Verbraucher gedruckt wurde.
func (s *DunningService) SetInvoiceSource(r domain.InvoiceRepository) { s.invoiceRepo = r }

// consumerNoticePrinted meldet, ob die Rechnung hinter einem offenen Posten den
// Verzugshinweis aufwies.
//
// consumerNoticePrinted beantwortet die Frage nur für die Rechnungen, die
// Buchfink selbst ausgestellt hat: dort steht das Kennzeichen am Datensatz,
// und eine Bestandsrechnung aus der Zeit vor dem Hinweis hat es zu Recht
// nicht. Ein von Hand gebuchter Posten — die Rechnung wurde außerhalb geschrieben — ist
// kein „ohne Hinweis": Buchfink kennt das Dokument nicht und darf über seinen
// Text nichts behaupten. Für ihn bleibt es beim Verzug nach dreißig Tagen, und
// der Hinweis unter dem Vorschlag sagt, worauf er beruht.
func (s *DunningService) consumerNoticePrinted(ctx context.Context, documentNumber string) bool {
	if s.invoiceRepo == nil || strings.TrimSpace(documentNumber) == "" {
		return true
	}
	inv, err := s.invoiceRepo.FindByNumber(ctx, documentNumber)
	if err != nil || inv == nil {
		return true
	}
	return inv.ConsumerNoticePrinted
}

// SetFiscalYear updates the active fiscal year.
func (s *DunningService) SetFiscalYear(year int) { s.fiscalYear = year }

// Proposals liefert die Mahnvorschläge je Kunde.
func (s *DunningService) Proposals(ctx context.Context, today string) ([]DunningProposal, error) {
	if today == "" {
		today = todayLocal()
	}
	proposals := make([]DunningProposal, 0)
	if s.openItems == nil {
		return proposals, nil
	}

	items, err := s.openItems.OpenItems(ctx)
	if err != nil {
		return nil, err
	}
	levels, err := s.levels(ctx)
	if err != nil {
		return nil, err
	}
	rates, err := s.rates(ctx)
	if err != nil {
		return nil, err
	}
	reached := map[uint]int{}
	lastNotice := map[uint]string{}
	lumpSumCharged := map[uint]bool{}
	if s.dunningRepo != nil {
		if reached, err = s.dunningRepo.LevelByOpenItem(ctx); err != nil {
			return nil, err
		}
		if lastNotice, err = s.dunningRepo.LastNoticeByOpenItem(ctx); err != nil {
			return nil, err
		}
		if lumpSumCharged, err = s.dunningRepo.LumpSumChargedByOpenItem(ctx); err != nil {
			return nil, err
		}
	}

	byContact := map[uint]*DunningProposal{}
	order := make([]uint, 0, 8)
	// notes sammelt, was je Kunde nicht gerechnet werden konnte.
	notes := map[uint][]string{}

	for i := range items {
		item := &items[i]
		if item.ContactType != domain.ContactTypeCustomer || item.OpenAmount <= 0 {
			continue
		}
		if item.DueDate == "" || item.DueDate >= today {
			continue
		}
		overdue := daysOverdueBetween(item.DueDate, today)
		previous := reached[item.EntryID]
		sinceLast := -1
		if last := lastNotice[item.EntryID]; last != "" {
			sinceLast = daysOverdueBetween(last, today)
		}
		level, ok := nextDunningLevel(levels, previous, overdue, sinceLast)
		if !ok {
			continue
		}

		proposal := byContact[item.ContactID]
		if proposal == nil {
			// Ein Kunde, dessen Stammdaten sich nicht lesen lassen, fällt nicht
			// aus dem Mahnlauf: die Forderung besteht, und ein stillschweigend
			// übersprungener Kunde ist eine vergessene Forderung. Gerechnet wird
			// dann mit dem Verbrauchersatz — dem niedrigeren, ohne Pauschale —,
			// weil eine zu hohe Forderung der teurere Fehler ist. Der Grund steht
			// im Vorschlag.
			isConsumer := true
			contact, err := s.contact(ctx, item.ContactID)
			if err == nil {
				isConsumer = contact.IsConsumer()
			} else {
				notes[item.ContactID] = append(notes[item.ContactID], fmt.Sprintf(
					"Die Stammdaten dieses Kunden ließen sich nicht lesen (%v); gerechnet ist "+
						"vorsichtshalber mit fünf Prozentpunkten und ohne Pauschale. Prüfe den "+
						"Kontakt, bevor das Schreiben herausgeht.", err))
			}
			proposal = &DunningProposal{
				ContactID: item.ContactID, ContactName: item.ContactName,
				IsConsumer: isConsumer, NoticeDate: today,
			}
			proposal.EnsureLists()
			byContact[item.ContactID] = proposal
			order = append(order, item.ContactID)
		}

		row := DunningProposalItem{
			EntryID: item.EntryID, DocumentNumber: item.DocumentNumber,
			DocumentDate: item.DocumentDate, DueDate: item.DueDate,
			OpenAmount: item.OpenAmount, DaysOverdue: overdue,
			Level: level.Level, PreviousLevel: previous,
		}
		// Gegenüber einem Verbraucher tritt der Verzug nach dreißig Tagen nur
		// ein, wenn die Rechnung darauf hingewiesen hat (§ 286 Abs. 3 Satz 1
		// Halbsatz 2 BGB). Der Hinweis steht an der Rechnung und nicht in einer
		// Einstellung: für eine Rechnung ohne diesen Hinweis macht keine
		// spätere Änderung ihn nachträglich wahr.
		noticePrinted := true
		if proposal.IsConsumer {
			noticePrinted = s.consumerNoticePrinted(ctx, item.DocumentNumber)
		}
		if from, err := accounting.DefaultInterestStartFor(
			item.DueDate, proposal.IsConsumer, noticePrinted,
		); err != nil {
			// Ohne Verzugsbeginn keine Zinsen — und keine Pauschale, denn beide
			// setzen den Verzug voraus. Gesagt wird es trotzdem.
			row.Note = fmt.Sprintf(
				"Zu %s ist der Verzugsbeginn nicht bestimmbar: %v", documentOf(item), err)
		} else {
			row.DefaultFrom = from
			// Der Verzug richtet sich nach dem Kalender, nicht nach dem Gelingen der Zinsrechnung:
			// „from" ist der erste Tag, für den Zinsen laufen, also auch der
			// Tag, an dem der Verzug eingetreten ist (§ 286 Abs. 3 BGB).
			inDefault := from <= today
			interest, err := accounting.DefaultInterest(
				item.OpenAmount, from, today, proposal.IsConsumer, rates)
			if err != nil {
				// Der häufigste Grund ist ein fehlender Basiszinssatz für ein
				// Halbjahr. Er steht in den Einstellungen und ist dort
				// nachzutragen; verschwiegen führte er zu einer Forderung ohne
				// Zinsen, die niemand als unvollständig erkennt.
				row.Note = fmt.Sprintf(
					"Zu %s sind keine Verzugszinsen gerechnet: %v. Trage den Basiszinssatz in den "+
						"Einstellungen nach.", documentOf(item), err)
			} else {
				row.Interest = interest.Amount
				row.InterestDays = interest.Days
			}
			// Die Pauschale des § 288 Abs. 5 BGB fällt je Forderung einmal an
			// und nur gegenüber einem Schuldner, der kein Verbraucher ist.
			//
			// Angesetzt wird sie, sobald der Verzug eingetreten ist — und nicht
			// mit der ersten Mahnstufe: die voreingestellte Zahlungserinnerung
			// geht sieben Tage nach Fälligkeit heraus, der Verzug tritt erst
			// nach dreißig Tagen ein (§ 286 Abs. 3 BGB). Wer die Pauschale an
			// das erste Schreiben knüpfte, setzte sie im Regelfall nie an. Dass
			// sie sich im nächsten Schreiben nicht wiederholt, sichert der
			// Blick in die schon ergangenen Schreiben und nicht die Stufe.
			//
			// Sie richtet sich auch nicht nach den gerechneten Zinstagen: ein
			// fehlender Basiszinssatz lässt die Zinsen ausfallen, den Verzug
			// aber nicht. Wer die Pauschale daran knüpfte, ließe sie
			// stillschweigend wegfallen — und der Hinweis nennte nur die
			// fehlenden Zinsen.
			if !proposal.IsConsumer && inDefault && !lumpSumCharged[item.EntryID] {
				row.LumpSum = accounting.DefaultInterestLumpSum
				if err != nil {
					row.Note += " Die Pauschale von 40 € ist trotzdem angesetzt: sie richtet sich nach " +
						"dem Verzug und nicht nach der Zinsrechnung."
				}
			}
		}

		proposal.Items = append(proposal.Items, row)
		proposal.Principal += row.OpenAmount
		proposal.Interest += row.Interest
		proposal.LumpSum += row.LumpSum
		if level.Level > proposal.Level {
			proposal.Level = level.Level
			proposal.LevelLabel = level.Label
			proposal.Fee = level.Fee
		}
	}

	for _, contactID := range order {
		proposal := byContact[contactID]
		sort.SliceStable(proposal.Items, func(i, j int) bool {
			return proposal.Items[i].DueDate < proposal.Items[j].DueDate
		})
		proposal.Total = proposal.Principal + proposal.Interest + proposal.Fee + proposal.LumpSum
		proposal.Note = dunningNote(proposal, notes[contactID])
		proposal.EnsureLists()
		proposals = append(proposals, *proposal)
	}
	// Die höchste Stufe zuerst, dann der größte Betrag: was am längsten offen
	// ist, steht oben.
	sort.SliceStable(proposals, func(i, j int) bool {
		if proposals[i].Level != proposals[j].Level {
			return proposals[i].Level > proposals[j].Level
		}
		return proposals[i].Total > proposals[j].Total
	})
	return proposals, nil
}

// documentOf benennt den Posten in einem Hinweis.
func documentOf(item *domain.OpenItem) string {
	if number := strings.TrimSpace(item.DocumentNumber); number != "" {
		return "Beleg " + number
	}
	if number := strings.TrimSpace(item.EntryNumber); number != "" {
		return "Buchung " + number
	}
	return "diesem Posten"
}

// dunningNote ist der Satz, der unter dem Vorschlag steht.
//
// Er sagt auch, was beim Rechnen schiefging: ein Vorschlag, dessen Zinsen an
// einem fehlenden Basiszinssatz gescheitert sind, sieht sonst aus wie einer ohne
// Verzug — und die Forderung ginge zu niedrig heraus.
func dunningNote(p *DunningProposal, extra []string) string {
	parts := []string{
		"Mahngebühr und Verzugszinsen werden nicht gebucht: sie sind erst mit der Zahlung Ertrag.",
	}
	if p.IsConsumer {
		parts = append(parts,
			"Der Kunde ist Verbraucher: Verzugszinsen fünf Prozentpunkte über dem Basiszinssatz, "+
				"keine Pauschale (§ 288 Abs. 1 und 5 BGB).")
		// Der Verzug ohne Mahnung tritt gegenüber einem Verbraucher nur ein,
		// wenn die Rechnung auf diese Folge hingewiesen hat (§ 286 Abs. 3 Satz 1
		// Halbsatz 2 BGB). Für die eigenen Rechnungen weiß Buchfink es und
		// rechnet danach; für einen von Hand gebuchten Posten kennt es das
		// Dokument nicht. Verschwiegen forderte das Schreiben in diesem Fall
		// Zinsen, die noch gar nicht laufen.
		parts = append(parts,
			"Gegenüber einem Verbraucher setzen die Zinsen voraus, dass die Rechnung auf den "+
				"Verzugseintritt dreißig Tage nach Fälligkeit hingewiesen hat "+
				"(§ 286 Abs. 3 Satz 1 Halbsatz 2 BGB) — fehlt der Hinweis, beginnt der Verzug "+
				"erst mit der Mahnung. Bei den in Buchfink ausgestellten Rechnungen ist das "+
				"berücksichtigt; bei einem von Hand gebuchten Posten prüfe den Text der Rechnung.")
	} else {
		parts = append(parts,
			"Der Kunde ist kein Verbraucher: Verzugszinsen neun Prozentpunkte über dem Basiszinssatz, "+
				"Pauschale 40 Euro je Forderung (§ 288 Abs. 2 und 5 BGB).")
	}
	// Die Vereinfachung bei Teilzahlungen gehört unter den Vorschlag und nicht
	// nur in den Quelltext: wer die Zinsen nachrechnet, kommt sonst auf einen
	// höheren Betrag und hält die Rechnung für falsch.
	for _, item := range p.Items {
		if item.InterestDays > 0 {
			parts = append(parts,
				"Die Zinsen sind auf den heute offenen Betrag gerechnet; eine Teilzahlung während "+
					"des Verzugs bleibt außer Betracht — die Forderung ist dann eher zu niedrig.")
			break
		}
	}
	seen := map[string]bool{}
	for _, item := range p.Items {
		if item.Note == "" || seen[item.Note] {
			continue
		}
		seen[item.Note] = true
		parts = append(parts, item.Note)
	}
	for _, note := range extra {
		if note == "" || seen[note] {
			continue
		}
		seen[note] = true
		parts = append(parts, note)
	}
	return strings.Join(parts, " ")
}

// nextDunningLevel liefert die Stufe, die ein Posten mit diesem Lauf erreicht.
//
// Zwei Bedingungen, und beide sind nötig. nextDunningLevel hält die Folge
// ein: aus der Zahlungserinnerung wird die erste Mahnung und aus ihr die
// zweite — auch dann, wenn der Posten schon so lange offen ist, dass die
// dritte Stufe fällig wäre. Wer eine Stufe überspringt, mahnt einen Kunden,
// der noch keine Erinnerung bekommen hat, mit einer zweiten Mahnung. Und es
// wahrt den Abstand zwischen zwei Schreiben: er ist der Abstand, den die Stufenfolge vorsieht
// (21 nach 7 Tagen heißt vierzehn Tage dazwischen). Ohne ihn stünde ein lange
// offener Posten am selben Tag dreimal im Vorschlag und wäre nach drei Klicks
// bei der zweiten Mahnung — ohne dass der Kunde je Zeit zu zahlen hatte.
//
// daysSinceLastNotice ist −1, wenn zu dem Posten noch kein Schreiben ergangen
// ist.
func nextDunningLevel(
	levels []domain.DunningLevel, previous, daysOverdue, daysSinceLastNotice int,
) (domain.DunningLevel, bool) {
	var previousLevel domain.DunningLevel
	for _, level := range levels {
		if level.Level == previous {
			previousLevel = level
		}
		if level.Level != previous+1 {
			continue
		}
		if daysOverdue < level.DaysAfterDue {
			return domain.DunningLevel{}, false
		}
		if previous > 0 && daysSinceLastNotice >= 0 {
			gap := level.DaysAfterDue - previousLevel.DaysAfterDue
			if gap > 0 && daysSinceLastNotice < gap {
				return domain.DunningLevel{}, false
			}
		}
		return level, true
	}
	return domain.DunningLevel{}, false
}

// DunningRunRequest ist der Auftrag eines Mahnlaufs über mehrere Kunden.
type DunningRunRequest struct {
	// ContactIDs sind die Kunden, für die ein Schreiben entstehen soll — die
	// Auswahl aus der Vorschlagsliste. Leer heißt: nichts tun. Ausdrücklich
	// nicht „alle": ein Mahnlauf, der auf einen Klick jeden Kunden anschreibt,
	// wird einmal versehentlich ausgelöst und ist dann nicht mehr einzufangen.
	ContactIDs      []uint `json:"contactIds"`
	NoticeDate      string `json:"noticeDate"`
	PaymentDeadline string `json:"paymentDeadline"`
}

// CreateMany erzeugt die Mahnschreiben eines Laufs.
//
// Kunde für Kunde und nicht in einer Transaktion: jedes Schreiben ist ein
// eigener Vorgang, und wenn das dritte scheitert, sind die ersten beiden
// trotzdem herausgegangen. Der Fehler wird zurückgegeben, die erzeugten
// Schreiben ebenso — verschwiegen wird keines von beiden.
func (s *DunningService) CreateMany(
	ctx context.Context, req DunningRunRequest,
) ([]domain.DunningNotice, error) {
	notices := make([]domain.DunningNotice, 0, len(req.ContactIDs))
	if len(req.ContactIDs) == 0 {
		return notices, fmt.Errorf("es ist kein Kunde ausgewählt, für den ein Mahnschreiben entstehen soll")
	}
	var failed []string
	for _, contactID := range req.ContactIDs {
		notice, err := s.Create(ctx, DunningRequest{
			ContactID: contactID, NoticeDate: req.NoticeDate,
			PaymentDeadline: req.PaymentDeadline,
		})
		if err != nil {
			failed = append(failed, fmt.Sprintf("Kunde %d: %v", contactID, err))
			continue
		}
		notices = append(notices, *notice)
	}
	if len(failed) > 0 {
		return notices, fmt.Errorf("%d von %d Schreiben sind nicht entstanden — %s",
			len(failed), len(req.ContactIDs), strings.Join(failed, "; "))
	}
	return notices, nil
}

// Create erzeugt das Mahnschreiben eines Kunden und legt es ab.
func (s *DunningService) Create(ctx context.Context, req DunningRequest) (*domain.DunningNotice, error) {
	if req.ContactID == 0 {
		return nil, fmt.Errorf("zu einem Mahnschreiben gehört der Kunde")
	}
	noticeDate := req.NoticeDate
	if noticeDate == "" {
		noticeDate = todayLocal()
	}
	deadline := req.PaymentDeadline
	if deadline == "" {
		deadline = addDays(noticeDate, DunningPaymentDays)
	}

	proposals, err := s.Proposals(ctx, noticeDate)
	if err != nil {
		return nil, err
	}
	var proposal *DunningProposal
	for i := range proposals {
		if proposals[i].ContactID == req.ContactID {
			proposal = &proposals[i]
			break
		}
	}
	if proposal == nil {
		return nil, fmt.Errorf(
			"für diesen Kunden gibt es zum %s nichts zu mahnen: kein Posten hat die nächste Mahnstufe erreicht",
			noticeDate)
	}
	if s.dunningRepo == nil {
		return nil, fmt.Errorf("die Ablage der Mahnschreiben ist nicht eingerichtet")
	}

	notice := &domain.DunningNotice{
		FiscalYear: s.fiscalYear, ContactID: proposal.ContactID,
		ContactName: proposal.ContactName, IsConsumer: proposal.IsConsumer,
		Level: proposal.Level, LevelLabel: proposal.LevelLabel,
		NoticeDate: noticeDate, DueDate: deadline,
		PrincipalAmount: proposal.Principal, InterestAmount: proposal.Interest,
		FeeAmount: proposal.Fee, LumpSumAmount: proposal.LumpSum,
		TotalAmount: proposal.Total,
	}
	for _, item := range proposal.Items {
		notice.Items = append(notice.Items, domain.DunningNoticeItem{
			OpenItemEntryID: item.EntryID, DocumentNumber: item.DocumentNumber,
			DocumentDate: item.DocumentDate, DueDate: item.DueDate,
			OpenAmount: item.OpenAmount, DefaultFrom: item.DefaultFrom,
			InterestDays: item.InterestDays, InterestAmount: item.Interest,
			LumpSumAmount: item.LumpSum, Level: item.Level,
		})
	}
	notice.EnsureLists()

	note, removeDocument := s.renderDocument(ctx, notice, proposal)
	notice.DocumentNote = note

	if err := s.dunningRepo.Create(ctx, notice); err != nil {
		// Das PDF liegt schon im Belegspeicher, der Datensatz ist nicht
		// entstanden: ohne das Aufräumen bliebe eine Datei unter
		// dokumente/mahnungen/ liegen, zu der es kein Schreiben, keinen Empfänger
		// und keinen Protokolleintrag gibt — die Art Fundstück, die im
		// Prüfermodus niemand mehr erklären kann.
		removeDocument()
		return nil, fmt.Errorf("das Mahnschreiben konnte nicht abgelegt werden: %w", err)
	}

	if s.auditRepo != nil {
		details := fmt.Sprintf(
			"%s an %s über %d Posten: %s € Hauptforderung, %s € Zinsen, %s € Gebühr, %s € Pauschale",
			notice.LevelLabel, notice.ContactName, len(notice.Items),
			notice.PrincipalAmount, notice.InterestAmount, notice.FeeAmount, notice.LumpSumAmount)
		if notice.DocumentPath != "" {
			details += fmt.Sprintf("; abgelegt als %s (SHA256 %s)", notice.DocumentName, notice.DocumentSHA256)
		} else if notice.DocumentNote != "" {
			details += "; " + notice.DocumentNote
		}
		_ = s.auditRepo.Log(ctx, domain.AuditActionCreate, "DUNNING_NOTICE",
			fmt.Sprintf("%d", notice.ID), details)
	}
	return notice, nil
}

// renderDocument setzt das Schreiben als PDF und legt es im Belegspeicher ab.
//
// Der erste Rückgabewert ist der Grund, aus dem kein PDF entstanden ist; leer
// heißt: es ist entstanden. Ein Fehler nimmt dem Mahnlauf nicht das Ergebnis —
// die Forderung ist gemahnt, sobald das Schreiben herausgeht, und die Beträge
// stehen im Datensatz.
//
// Der zweite Rückgabewert räumt die abgelegte Datei wieder weg. Er wird
// gebraucht, wenn der Datensatz danach nicht entsteht: die Ablage geht dem
// Schreiben voraus, weil Pfad und Prüfsumme in den Datensatz gehören. Entfernt
// wird nur eine Datei, die dieser Lauf selbst geschrieben hat — der Speicher
// legt gleiche Inhalte nur einmal ab, und eine wiederverwendete Datei gehört
// einem anderen Schreiben.
func (s *DunningService) renderDocument(
	ctx context.Context, notice *domain.DunningNotice, proposal *DunningProposal,
) (string, func()) {
	nothingToRemove := func() {}
	if s.renderer == nil {
		return "Der Dokumentensetzer ist nicht verfügbar; das Schreiben liegt nur als Datensatz vor.",
			nothingToRemove
	}
	if s.store == nil {
		return "Ohne Belegspeicher lässt sich das Schreiben nicht ablegen.", nothingToRemove
	}
	// Die vollständigen Unternehmensdaten und nicht nur der Name: ein
	// Mahnschreiben, das um Zahlung bittet, muss sagen, wohin gezahlt werden
	// soll. Die Bankverbindung steht in den Einstellungen — sie hier nicht zu
	// lesen hieße, den Empfänger nach der Kontonummer suchen zu lassen.
	var cfg *domain.CompanySettings
	if s.settingsRepo != nil {
		if settings, err := s.settingsRepo.GetCompanySettings(ctx); err == nil {
			cfg = settings
		}
	}
	markdown := dunningMarkdown(notice, proposal, cfg)
	title := fmt.Sprintf("%s an %s vom %s", notice.LevelLabel, notice.ContactName, notice.NoticeDate)
	issued, err := time.Parse("2006-01-02", notice.NoticeDate)
	if err != nil {
		issued = time.Now()
	}
	pdf, err := s.renderer.RenderDocumentPDF(ctx, procdoc.Typst(markdown, title, issued), title)
	if err != nil {
		return fmt.Sprintf("Das PDF konnte nicht gesetzt werden: %v.", err), nothingToRemove
	}
	name := fmt.Sprintf("mahnung-%s-stufe-%d-%s.pdf",
		notice.NoticeDate, notice.Level, sanitizeFileName(notice.ContactName))
	stored, err := s.store.PutDocument(DunningCategory, name, bytes.NewReader(pdf))
	if err != nil {
		return fmt.Sprintf("Das PDF konnte nicht abgelegt werden: %v.", err), nothingToRemove
	}
	notice.DocumentName = name
	notice.DocumentPath = stored.RelPath
	notice.DocumentSHA256 = stored.SHA256
	if stored.Deduplicated {
		return "", nothingToRemove
	}
	relPath := stored.RelPath
	return "", func() { _ = s.store.Delete(relPath) }
}

// dunningMarkdown schreibt den Text des Mahnschreibens.
//
// Als Markdown und nicht als Typst: derselbe Weg, den die Verfahrens-
// dokumentation geht (procdoc.Typst), und derselbe Vorteil — der Text lässt
// sich lesen und prüfen, ohne den Setzer zu starten.
func dunningMarkdown(
	notice *domain.DunningNotice, proposal *DunningProposal, cfg *domain.CompanySettings,
) string {
	var b strings.Builder
	if cfg != nil && cfg.FirmName() != "" {
		fmt.Fprintf(&b, "%s\n\n", cfg.FirmName())
	}
	fmt.Fprintf(&b, "# %s\n\n", notice.LevelLabel)
	fmt.Fprintf(&b, "%s\n\n", notice.ContactName)
	fmt.Fprintf(&b, "Datum: %s\n\n", germanDate(notice.NoticeDate))

	b.WriteString("Sehr geehrte Damen und Herren,\n\n")
	if notice.Level <= 1 {
		b.WriteString("die folgenden Rechnungen sind noch offen. Vermutlich ist die Zahlung nur übersehen worden.\n\n")
	} else {
		b.WriteString("trotz unserer bisherigen Schreiben sind die folgenden Rechnungen weiterhin offen.\n\n")
	}

	b.WriteString("| Beleg | Datum | Fällig | Betrag | Verzug seit | Tage | Zinsen |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, item := range notice.Items {
		defaultFrom := "—"
		if item.DefaultFrom != "" && item.InterestDays > 0 {
			defaultFrom = germanDate(item.DefaultFrom)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s € | %s | %d | %s € |\n",
			orDash(item.DocumentNumber), germanDate(item.DocumentDate), germanDate(item.DueDate),
			item.OpenAmount, defaultFrom, item.InterestDays, item.InterestAmount)
	}
	b.WriteString("\n")

	fmt.Fprintf(&b, "Hauptforderung: %s €\n\n", notice.PrincipalAmount)
	if notice.InterestAmount != 0 {
		fmt.Fprintf(&b, "Verzugszinsen: %s €\n\n", notice.InterestAmount)
	}
	if notice.FeeAmount != 0 {
		fmt.Fprintf(&b, "Mahngebühr: %s €\n\n", notice.FeeAmount)
	}
	if notice.LumpSumAmount != 0 {
		fmt.Fprintf(&b, "Pauschale nach § 288 Abs. 5 BGB: %s €\n\n", notice.LumpSumAmount)
	}
	fmt.Fprintf(&b, "**Gesamtbetrag: %s €**\n\n", notice.TotalAmount)

	// Der Verweis auf das „unten genannte Konto" steht nur da, wo unten
	// tatsächlich eines steht. Ohne hinterlegte IBAN bliebe es eine Zusage, die
	// das Schreiben nicht einlöst — dann wird schlicht um Ausgleich gebeten.
	bank := bankDetailsBlock(cfg, notice)
	if bank != "" {
		fmt.Fprintf(&b,
			"Wir bitten um Ausgleich bis zum %s auf das unten genannte Konto.\n\n",
			germanDate(notice.DueDate))
	} else {
		fmt.Fprintf(&b, "Wir bitten um Ausgleich bis zum %s.\n\n", germanDate(notice.DueDate))
	}

	if notice.InterestAmount != 0 {
		rate := "neun"
		if notice.IsConsumer {
			rate = "fünf"
		}
		fmt.Fprintf(&b,
			"Die Verzugszinsen sind taggenau mit %s Prozentpunkten über dem Basiszinssatz "+
				"des jeweiligen Halbjahres gerechnet (§ 288 BGB, § 247 BGB).\n\n", rate)
	}
	if bank != "" {
		b.WriteString(bank)
	}
	if proposal != nil && len(proposal.Items) > 0 {
		b.WriteString("Sollte sich Ihre Zahlung mit diesem Schreiben gekreuzt haben, betrachten Sie es bitte als gegenstandslos.\n")
	}
	return b.String()
}

// bankDetailsBlock ist die Zahlungsangabe unter dem Schreiben; leer, solange
// keine IBAN hinterlegt ist.
//
// Der Block richtet sich nach der IBAN: ohne sie ist mit Bankname und BIC allein nichts
// überwiesen. Der Verwendungszweck nennt die gemahnten Belege — er ist es, der
// die eingehende Zahlung den Posten wieder zuordnet, und ohne ihn landet sie im
// Bankimport als unbekannter Eingang.
func bankDetailsBlock(cfg *domain.CompanySettings, notice *domain.DunningNotice) string {
	if cfg == nil || strings.TrimSpace(cfg.IBAN) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("**Bankverbindung**\n\n")
	if holder := cfg.FirmName(); holder != "" {
		fmt.Fprintf(&b, "Kontoinhaber: %s\n\n", holder)
	}
	if bank := strings.TrimSpace(cfg.BankName); bank != "" {
		fmt.Fprintf(&b, "Bank: %s\n\n", bank)
	}
	fmt.Fprintf(&b, "IBAN: %s\n\n", strings.TrimSpace(cfg.IBAN))
	if bic := strings.TrimSpace(cfg.BIC); bic != "" {
		fmt.Fprintf(&b, "BIC: %s\n\n", bic)
	}
	if purpose := dunningPaymentPurpose(notice); purpose != "" {
		fmt.Fprintf(&b, "Verwendungszweck: %s\n\n", purpose)
	}
	return b.String()
}

// dunningPaymentPurpose nennt die gemahnten Belege als Verwendungszweck.
//
// Höchstens fünf: ein Verwendungszweck, der über mehrere Zeilen läuft, wird beim
// Überweisen abgeschnitten. Was darüber hinausgeht, steht ohnehin in der Tabelle
// des Schreibens.
func dunningPaymentPurpose(notice *domain.DunningNotice) string {
	if notice == nil {
		return ""
	}
	numbers := make([]string, 0, len(notice.Items))
	for _, item := range notice.Items {
		if number := strings.TrimSpace(item.DocumentNumber); number != "" {
			numbers = append(numbers, number)
		}
		if len(numbers) == 5 {
			break
		}
	}
	if len(numbers) == 0 {
		return ""
	}
	purpose := strings.Join(numbers, ", ")
	if len(numbers) < len(notice.Items) {
		purpose += " u. a."
	}
	return purpose
}

// Notices liefert die Mahnschreiben eines Kunden; 0 heißt: alle.
func (s *DunningService) Notices(ctx context.Context, contactID uint) ([]domain.DunningNotice, error) {
	if s.dunningRepo == nil {
		return make([]domain.DunningNotice, 0), nil
	}
	return s.dunningRepo.FindByContact(ctx, contactID)
}

// BaseRates liefert die Basiszinssätze: die hinterlegten, überschrieben von den
// gepflegten.
func (s *DunningService) BaseRates(ctx context.Context) ([]domain.BaseRate, error) {
	stored := make([]domain.BaseRate, 0)
	if s.baseRateRepo != nil {
		var err error
		if stored, err = s.baseRateRepo.FindAll(ctx); err != nil {
			return nil, err
		}
	}
	merged := accounting.MergeBaseRates(toRatePeriods(stored))
	out := make([]domain.BaseRate, 0, len(merged))
	for _, rate := range merged {
		out = append(out, domain.BaseRate{
			ValidFrom: rate.ValidFrom, BasisPoints: rate.BasisPoints,
			Source: rate.Source, Provisional: rate.Provisional,
		})
	}
	return out, nil
}

// SaveBaseRate trägt einen bekanntgegebenen Basiszinssatz nach.
//
// basisPoints ist der Satz in Hundertsteln eines Prozentpunktes: 152 sind
// 1,52 %. Ein nachgetragener Satz gilt als bekanntgegeben und nicht als
// fortgeschrieben — er kommt aus dem Bundesanzeiger und nicht aus einer
// Schätzung.
func (s *DunningService) SaveBaseRate(
	ctx context.Context, validFrom string, basisPoints int,
) ([]domain.BaseRate, error) {
	if s.baseRateRepo == nil {
		return nil, fmt.Errorf("die Basiszinstabelle ist nicht eingerichtet")
	}
	if _, err := time.Parse("2006-01-02", validFrom); err != nil {
		return nil, fmt.Errorf("%q ist kein gültiger Stichtag (erwartet JJJJ-MM-TT)", validFrom)
	}
	// Der Basiszinssatz hat sich seit seiner Einführung zwischen −0,88 % und
	// 3,62 % bewegt. Eine Schranke hält den Tippfehler ab, der aus 1,52 %
	// hundertfünfzigmal so viel Zins macht.
	if basisPoints < -1000 || basisPoints > 2000 {
		return nil, fmt.Errorf(
			"%d Hundertstel Prozentpunkte sind kein plausibler Basiszinssatz (erwartet zwischen -1000 und 2000)",
			basisPoints)
	}
	rate := &domain.BaseRate{
		ValidFrom: validFrom, BasisPoints: basisPoints,
		Source: "Bekanntgabe der Deutschen Bundesbank, nachgetragen",
	}
	if err := s.baseRateRepo.Save(ctx, rate); err != nil {
		return nil, fmt.Errorf("der Basiszinssatz konnte nicht gespeichert werden: %w", err)
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "BASE_RATE", validFrom,
			fmt.Sprintf("Basiszinssatz ab %s auf %s gesetzt", validFrom, rate.Percent()))
	}
	return s.BaseRates(ctx)
}

// levels liefert die eingestellten Mahnstufen.
func (s *DunningService) levels(ctx context.Context) ([]domain.DunningLevel, error) {
	if s.settingsRepo == nil {
		return domain.DefaultDunningLevels(), nil
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Mahnstufen konnten nicht gelesen werden: %w", err)
	}
	return domain.NormalizeDunningLevels(cfg.DunningLevels), nil
}

// rates liefert die Basiszinssätze in der Form der Zinsrechnung.
func (s *DunningService) rates(ctx context.Context) ([]accounting.BaseRatePeriod, error) {
	stored := make([]domain.BaseRate, 0)
	if s.baseRateRepo != nil {
		var err error
		if stored, err = s.baseRateRepo.FindAll(ctx); err != nil {
			return nil, err
		}
	}
	return accounting.MergeBaseRates(toRatePeriods(stored)), nil
}

func toRatePeriods(rates []domain.BaseRate) []accounting.BaseRatePeriod {
	out := make([]accounting.BaseRatePeriod, 0, len(rates))
	for _, rate := range rates {
		out = append(out, accounting.BaseRatePeriod{
			ValidFrom: rate.ValidFrom, BasisPoints: rate.BasisPoints,
			Source: rate.Source, Provisional: rate.Provisional,
		})
	}
	return out
}

func (s *DunningService) contact(ctx context.Context, id uint) (*domain.Contact, error) {
	if s.contactRepo == nil {
		return &domain.Contact{ID: id}, nil
	}
	return s.contactRepo.FindByID(ctx, id)
}

// daysOverdueBetween zählt die Tage zwischen zwei ISO-Daten; ein unlesbares
// Datum ergibt null.
func daysOverdueBetween(from, to string) int {
	a, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0
	}
	b, err := time.Parse("2006-01-02", to)
	if err != nil {
		return 0
	}
	return int(b.Sub(a).Hours() / 24)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// sanitizeFileName macht aus einem Namen einen Dateinamensbestandteil.
func sanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == 'ä':
			b.WriteString("ae")
		case r == 'ö':
			b.WriteString("oe")
		case r == 'ü':
			b.WriteString("ue")
		case r == 'ß':
			b.WriteString("ss")
		default:
			b.WriteRune('-')
		}
	}
	trimmed := strings.Trim(b.String(), "-")
	for strings.Contains(trimmed, "--") {
		trimmed = strings.ReplaceAll(trimmed, "--", "-")
	}
	if trimmed == "" {
		return "kunde"
	}
	return trimmed
}
