package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// Die Kopplung des Vorsteuerabzugs an die geprüfte Rechnung (UST-07, RECH-07).
//
// § 15 Abs. 1 Satz 1 Nr. 1 UStG lässt den Abzug nur aus „einer nach den §§ 14,
// 14a ausgestellten Rechnung" zu. Bis hierher hat Buchfink das Ergebnis seiner
// eigenen Rechnungsprüfung nicht gelesen: eine E-Rechnung konnte mit fünf
// Inhaltsfehlern durch den Buchungsweg laufen, und die Vorsteuer wurde gezogen,
// als sei nichts gewesen. Das war die Lücke zwischen zwei Bausteinen, die beide
// für sich funktionierten.
//
// Geprüft wird nur, wo tatsächlich Vorsteuer gezogen wird, und nur beim
// Inlandsumsatz. Beim innergemeinschaftlichen Erwerb und beim Reverse Charge
// hängt der Abzug nicht an einer Rechnung nach §§ 14, 14a UStG (§ 15 Abs. 1
// Satz 1 Nr. 3 und 4 UStG) — eine Sperre dort wäre eine erfundene Voraussetzung.

// InputTaxFinding ist ein blockierender Befund der Rechnungsprüfung.
//
// Er unterscheidet sich von der PostingWarning genau darin: die Warnung zeigt
// und urteilt nicht, der Befund hält die Buchung an. Beides nebeneinander zu
// führen ist Absicht — die E-Rechnungspflicht ist eine Frage, die der Anwender
// mit seinem Lieferanten klärt, eine fehlende Pflichtangabe kostet ihn dagegen
// den Abzug.
type InputTaxFinding struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	// Detail nennt, was fehlt, und die Vorschrift dazu.
	Detail string `json:"detail"`
	// Fixable sagt, ob sich der Befund durch eine Ergänzung der Stammdaten
	// beheben lässt — dann ist die Übersteuerung nicht der erste Weg.
	Fixable bool `json:"fixable"`
}

const (
	findingEInvoiceInvalid   = "input_tax_einvoice_invalid"
	findingIssuerAddress     = "input_tax_issuer_address"
	findingIssuerTaxNumber   = "input_tax_issuer_tax_number"
	findingDocumentDate      = "input_tax_document_date"
	findingSupplierName      = "input_tax_supplier_name"
	findingEInvoiceUnchecked = "input_tax_einvoice_unchecked"
	warningGiftLimitExceeded = "gift_limit_exceeded"
)

// incomingLines ist das Ergebnis der Zeilenbildung eines Eingangsbelegs.
//
// Sie enthält mehr als die Zeilen, weil beim Bilden mehr entsteht als sie: die
// Aufzeichnungen zu den Geschenken, die Warnungen der Freigrenze und die
// Befunde der Rechnungsprüfung. Die Vorschau zeigt alles davon, die Buchung
// wertet die Befunde aus — und beide laufen durch dieselbe Rechnung, damit sie
// nicht auseinandergehen können.
type incomingLines struct {
	lines   []domain.JournalLine
	contact *domain.Contact
	receipt *domain.Receipt
	// fx ist die Umrechnung eines Fremdwährungsbelegs, nil bei einem in Euro.
	fx       *fxContext
	gifts    []domain.GiftRecord
	warnings []PostingWarning
	findings []InputTaxFinding
}

// blockingError hält die Buchung an, solange ein Befund offen und kein Grund
// angegeben ist.
func (b *incomingLines) blockingError(req ReceiptRequest) error {
	if len(b.findings) == 0 {
		return nil
	}
	if strings.TrimSpace(req.OverrideReason) != "" {
		return nil
	}
	parts := make([]string, 0, len(b.findings))
	for _, f := range b.findings {
		parts = append(parts, f.Detail)
	}
	return fmt.Errorf(
		"der Vorsteuerabzug setzt eine Rechnung mit allen Pflichtangaben voraus (§ 15 Abs. 1 Satz 1 "+
			"Nr. 1 i. V. m. §§ 14, 14a UStG). Es fehlt: %s. Ergänze die Angabe, lass die Rechnung "+
			"berichtigen — oder buche mit einem Grund, der festgehalten wird",
		strings.Join(parts, " "))
}

// inputTaxFindings prüft die Voraussetzungen des Vorsteuerabzugs.
func (s *PostingService) inputTaxFindings(
	ctx context.Context,
	req ReceiptRequest,
	contact *domain.Contact,
	receipt *domain.Receipt,
	lines []domain.JournalLine,
) ([]InputTaxFinding, error) {
	out := make([]InputTaxFinding, 0, 2)
	if !deductsDomesticInputTax(lines) {
		return out, nil
	}

	// (a) Die E-Rechnung: Buchfink hat sie selbst geprüft und das Ergebnis am
	// Beleg festgehalten. Es zu haben und nicht zu lesen wäre die schlechteste
	// aller Möglichkeiten.
	if _, ok := receipt.FileByRole(domain.ReceiptRoleStructured); ok {
		switch {
		case receipt.ValidationErrors > 0:
			out = append(out, InputTaxFinding{
				Code:  findingEInvoiceInvalid,
				Title: "Die E-Rechnung hat Fehler",
				Detail: fmt.Sprintf(
					"Der Rechnungsdatensatz des Belegs %s hat %d Fehler aus der Prüfung nach EN 16931. "+
						"Solange sie bestehen, ist nicht belegt, dass die Rechnung alle Pflichtangaben "+
						"der §§ 14, 14a UStG hat.",
					receipt.ReceiptNumber, receipt.ValidationErrors),
			})
		case receipt.ValidatedAt == "":
			// Ein ungeprüfter Datensatz ist kein geprüfter. Ihn durchzulassen,
			// weil kein Fehler gemeldet wurde, hieße Schweigen als Zustimmung zu
			// lesen — es wurde nur nie gefragt.
			out = append(out, InputTaxFinding{
				Code:    findingEInvoiceUnchecked,
				Title:   "Der Rechnungsdatensatz wurde nicht geprüft",
				Fixable: true,
				Detail: fmt.Sprintf(
					"die Prüfung des Rechnungsdatensatzes von Beleg %s nach EN 16931. Ohne sie ist "+
						"nicht festgestellt, ob die Rechnung die Pflichtangaben der §§ 14, 14a UStG "+
						"hat — lies den Beleg erneut ein oder buche mit einem Grund.",
					receipt.ReceiptNumber),
			})
		default:
			// Ein geprüfter Datensatz ohne Fehler hat die Pflichtangaben in
			// Feldern; die Stammdatenprüfung darunter wäre dann eine zweite,
			// gröbere Prüfung derselben Frage.
			return out, nil
		}
	}

	// (b) Die sonstige Rechnung. Was Buchfink von ihr weiß, steht in den
	// Kopfdaten des Belegs und in den Stammdaten des Ausstellers.
	// Dieselbe Prüfung, die die Klärungsliste als Inhaltsfehler zeigt (siehe
	// receipt_content_checks.go). Zwei Fassungen davon gingen auseinander.
	for _, d := range issuerMasterDataDefects(contact) {
		out = append(out, InputTaxFinding{
			Code: d.Code, Title: d.Title, Detail: d.Detail, Fixable: d.Fixable,
		})
	}
	if len(req.DocumentDate) != 10 {
		out = append(out, InputTaxFinding{
			Code:   findingDocumentDate,
			Title:  "Das Rechnungsdatum fehlt",
			Detail: "das Ausstellungsdatum der Rechnung (§ 14 Abs. 4 Nr. 3 UStG).",
		})
	}
	_ = ctx
	return out, nil
}

// deductsDomesticInputTax meldet, ob die Buchung Vorsteuer aus einer Rechnung
// eines anderen Unternehmers zieht.
//
// Erkennbar am Steuerschlüssel und nicht am Steuerfall: der Schlüssel ist das,
// was am Ende in der Kennziffer 66 landet, und genau für diese Kennziffer
// verlangt § 15 Abs. 1 Satz 1 Nr. 1 UStG die Rechnung.
func deductsDomesticInputTax(lines []domain.JournalLine) bool {
	for _, l := range lines {
		if l.TaxKey == "VST19" || l.TaxKey == "VST7" {
			return true
		}
	}
	return false
}

// -------------------------------------------------------------------------
// Vorsteuerschlüssel und Geschenke
// -------------------------------------------------------------------------

// resolvedPosition ist eine Belegposition, nachdem Gruppe, Konto, abziehbarer
// Vorsteueranteil und die Aufzeichnung zum Geschenk feststehen.
type resolvedPosition struct {
	position ReceiptPosition
	group    accounting.PostingGroup
	hasGroup bool
	// account ist das erzwungene Konto — der freie Kontoweg der Maske oder das
	// nicht abziehbare Konto eines Geschenks über der Freigrenze.
	account string
	// permille ist der abziehbare Anteil der Vorsteuer, 1000 für den Regelfall.
	permille int64
	note     string
	gift     *domain.GiftRecord
	warnings []PostingWarning
}

// resolvePosition entscheidet, auf welches Konto eine Position geht und wie viel
// Vorsteuer sie hat.
func (s *PostingService) resolvePosition(
	ctx context.Context, req ReceiptRequest, p ReceiptPosition, giftTotals map[string]domain.Cents,
) (resolvedPosition, error) {
	out := resolvedPosition{position: p, permille: 1000, warnings: make([]PostingWarning, 0, 1)}

	if p.InputTaxShare != 0 {
		if p.InputTaxShare < 0 || p.InputTaxShare > 1000 {
			return out, fmt.Errorf("der abziehbare Vorsteueranteil liegt zwischen 0 und 100 %%")
		}
		if p.InputTaxShare < 1000 && strings.TrimSpace(p.InputTaxShareReason) == "" {
			return out, fmt.Errorf(
				"zu einem geteilten Vorsteuerabzug gehört sein Maßstab. § 15 Abs. 4 Satz 2 UStG lässt " +
					"die Aufteilung nach einer sachgerechten Schätzung zu — halte fest, worauf sie " +
					"beruht, etwa „Kfz zu 60 %% betrieblich genutzt, Fahrtenbuch 2026\"")
		}
		// Der geteilte Abzug gilt auch beim § 13b-Umsatz und beim
		// innergemeinschaftlichen Erwerb (UST-07 K2). Dort entstehen zwei
		// Steuerzeilen aus derselben Bemessungsgrundlage, und der Schlüssel
		// wirkt nur auf die Vorsteuerzeile — die geschuldete Steuer bleibt voll
		// (siehe taxLinesForShare). Vorher wies Buchfink den Fall ab; der
		// betrieblich zu 60 % genutzte Wagen aus dem EU-Ausland war damit nicht
		// zu buchen, obwohl das Gesetz beide Seiten klar trennt.
		out.permille = int64(p.InputTaxShare)
		if out.permille < 1000 {
			out.note = fmt.Sprintf("Vorsteuer zu %s abziehbar: %s",
				accounting.PermilleLabel(out.permille), strings.TrimSpace(p.InputTaxShareReason))
		}
	}

	if p.Account != "" {
		if label, blocked := accounting.AccountsRequiringGroup()[p.Account]; blocked {
			return out, fmt.Errorf(
				"das Konto %s gehört zur Gruppe %q. Diese Konten sind nur über ihre Buchungsgruppe "+
					"erreichbar: an ihnen hängen die Aufzeichnungspflicht des § 4 Abs. 7 EStG, die "+
					"Freigrenze je Empfänger und der Ausschluss der Vorsteuer — von Hand gebucht wäre "+
					"davon nichts geprüft", p.Account, label)
		}
		out.account = p.Account
		return out, nil
	}
	if p.PostingGroup == "" {
		return out, fmt.Errorf("weder eine Buchungsgruppe noch ein Konto angegeben")
	}
	group, err := accounting.LookupPostingGroup(p.PostingGroup)
	if err != nil {
		return out, err
	}
	if group.Direction != domain.DirectionIncoming {
		return out, fmt.Errorf("%q passt nicht zur Belegrichtung", group.Label)
	}
	out.group, out.hasGroup = group, true

	// § 15 Abs. 1a UStG: für die Aufwendungen des § 4 Abs. 5 Satz 1 Nr. 1 bis 4
	// und 7 EStG gibt es keinen Vorsteuerabzug. Die Steuer gehört dann zum
	// Aufwand.
	if group.InputTaxExcluded {
		out.permille = 0
		out.note = appendText(out.note,
			"Kein Vorsteuerabzug (§ 15 Abs. 1a UStG); die Umsatzsteuer gehört zum Aufwand")
	}

	if group.RecipientRequired || group.Limit == accounting.LimitGiftPerRecipient {
		gift, warning, err := s.resolveGift(ctx, req, p, group, giftTotals)
		if err != nil {
			return out, err
		}
		out.gift = gift
		if warning != nil {
			out.warnings = append(out.warnings, *warning)
		}
		if gift != nil && gift.NonDeductible {
			out.account = group.NonDeductibleAccount
			out.permille = 0
			out.hasGroup = false
			out.note = appendText(out.note, fmt.Sprintf(
				"Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG für %s überschritten: nicht abziehbar, "+
					"und ohne Vorsteuerabzug (§ 15 Abs. 1a UStG)", gift.RecipientName))
		}
	}
	return out, nil
}

// Der Vorsteuerausschluss des § 15 Abs. 1a UStG ist seit Welle 8 auch beim
// § 13b-Umsatz und beim innergemeinschaftlichen Erwerb buchbar: die geschuldete
// Steuer entsteht auf die volle Bemessungsgrundlage, die Vorsteuerzeile
// entfällt (siehe taxLinesForShare). Die Vorschrift nimmt den Abzug und nicht
// die Steuerschuld — vorher wies Buchfink den Fall ab, weil es beides nur
// gemeinsam rechnen konnte.

// resolveGift baut die Aufzeichnung zu einem Geschenk und entscheidet über die
// Freigrenze.
//
// Buchfink bucht das aktuelle Geschenk auf das nicht abziehbare Konto, sobald es
// die Grenze reißt, und warnt vorher. Die früheren Geschenke desselben
// Empfängers bleiben stehen, wo sie sind — sie rückwirkend umzubuchen wäre ein
// Eingriff in gebuchte Vorgänge, den niemand angeordnet hat. Der Bericht führt
// sie als umzubuchen auf, und die Umbuchung ist ein eigener, sichtbarer Vorgang.
func (s *PostingService) resolveGift(
	ctx context.Context, req ReceiptRequest, p ReceiptPosition, group accounting.PostingGroup,
	giftTotals map[string]domain.Cents,
) (*domain.GiftRecord, *PostingWarning, error) {
	input := p.Gift
	if input == nil || (input.ContactID == 0 && strings.TrimSpace(input.Name) == "") {
		return nil, nil, fmt.Errorf(
			"zu %q gehört der Empfänger. § 4 Abs. 7 EStG lässt den Abzug nur zu, wenn die Aufwendung "+
				"einzeln und getrennt aufgezeichnet ist — und ohne Empfänger ließe sich die Freigrenze "+
				"je Empfänger und Wirtschaftsjahr nicht führen", group.Label)
	}

	record := &domain.GiftRecord{
		Date:          req.DocumentDate,
		RecipientName: strings.TrimSpace(input.Name),
		Occasion:      strings.TrimSpace(input.Occasion),
		NetAmount:     p.Net,
		Account:       group.Account,
	}
	if input.ContactID != 0 {
		id := input.ContactID
		record.RecipientContactID = &id
		// Der Name kommt aus der Kartei und nicht aus einer Kennung. „Kontakt 17"
		// ist keine Aufzeichnung im Sinne des § 4 Abs. 7 EStG: sie muss den
		// Empfänger benennen, und eine Kennung benennt eine Zeile in einer
		// Tabelle. Ein unbekannter Kontakt wird zurückgewiesen — die Freigrenze
		// je Empfänger ließe sich sonst über einen Empfänger führen, den es nicht
		// gibt.
		recipient, err := s.contactRepo.FindByID(ctx, id)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"der Empfänger des Geschenks (Kontakt %d) wurde nicht gefunden: %w", id, err)
		}
		if name := contactCompanyName(recipient); name != "" {
			record.RecipientName = name
		}
	}
	if req.BookingDate != "" {
		record.FiscalYear = domain.GetFiscalYearForDate(req.BookingDate, s.fiscalYearStartMonth())
	}
	if err := record.Validate(); err != nil {
		return nil, nil, err
	}

	if group.Limit != accounting.LimitGiftPerRecipient {
		return record, nil, nil
	}
	// Die Freigrenze gilt für Wirtschaftsjahre, die nach einem Stichtag
	// beginnen, und wird deshalb am ersten Tag des Wirtschaftsjahres gemessen —
	// so wie im Bericht. Am Belegdatum gemessen rechnete ein abweichendes
	// Wirtschaftsjahr 2023/24 mit zwei verschiedenen Grenzen: 35 € vor dem
	// Jahreswechsel und 50 € danach, im selben Wirtschaftsjahr.
	limitDate := req.DocumentDate
	if record.FiscalYear > 0 {
		limitDate = fmt.Sprintf("%04d-%02d-01", record.FiscalYear, s.fiscalYearStartMonth())
	}
	params, err := accounting.TaxParametersFor(limitDate)
	if err != nil {
		return nil, nil, err
	}
	before := giftTotals[record.RecipientKey()]
	after := before + p.Net
	if after <= params.GiftDeductibleLimit {
		return record, nil, nil
	}

	record.NonDeductible = true
	record.Account = group.NonDeductibleAccount
	warning := &PostingWarning{
		Code:     warningGiftLimitExceeded,
		Severity: "warning",
		Title:    "Die Freigrenze für Geschenke ist überschritten",
		Detail: fmt.Sprintf(
			"An %s sind in diesem Wirtschaftsjahr bereits %s € netto verschenkt worden; mit diesem "+
				"Geschenk sind es %s €. Die Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG liegt bei "+
				"%s € — und sie ist eine Freigrenze: mit ihrer Überschreitung sind sämtliche Geschenke "+
				"an diesen Empfänger nicht abziehbar, und mit ihnen entfällt nach § 15 Abs. 1a UStG "+
				"auch der Vorsteuerabzug. Dieses Geschenk wird deshalb auf %s ohne Vorsteuer gebucht; "+
				"die früheren stehen im Bericht über die nicht abziehbaren Betriebsausgaben als "+
				"umzubuchen.",
			record.RecipientName, before, after, params.GiftDeductibleLimit, group.NonDeductibleAccount),
	}
	return record, warning, nil
}

// giftTotals liest die bisherigen Geschenke des Wirtschaftsjahres je Empfänger.
func (s *PostingService) giftTotals(
	ctx context.Context, req ReceiptRequest,
) (map[string]domain.Cents, error) {
	out := map[string]domain.Cents{}
	if s.gifts == nil || !requestHasGifts(req) {
		return out, nil
	}
	year := domain.GetFiscalYearForDate(req.BookingDate, s.fiscalYearStartMonth())
	records, err := s.gifts.GiftsInYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("die bisherigen Geschenke des Jahres ließen sich nicht lesen: %w", err)
	}
	for i := range records {
		out[records[i].RecipientKey()] += records[i].NetAmount
	}
	return out, nil
}

// requestHasGifts meldet, ob überhaupt eine Position ein Geschenk ist. Ohne sie
// wird die Kartei nicht gelesen — der Regelfall soll nicht die Kosten des
// Sonderfalls tragen.
func requestHasGifts(req ReceiptRequest) bool {
	for _, p := range req.Positions {
		if p.Gift != nil {
			return true
		}
		if p.PostingGroup == "" {
			continue
		}
		if group, err := accounting.LookupPostingGroup(p.PostingGroup); err == nil {
			if group.Limit == accounting.LimitGiftPerRecipient || group.RecipientRequired {
				return true
			}
		}
	}
	return false
}

// giftRegister ist der Ausschnitt der Geschenkkartei, den der Belegweg braucht.
type giftRegister interface {
	GiftsInYear(ctx context.Context, fiscalYear int) ([]domain.GiftRecord, error)
}

// SetGiftRegister koppelt die Geschenkkartei an den Belegweg. Ohne sie kennt er
// keine Freigrenze und bucht jedes Geschenk als abziehbar — deshalb wird sie
// gesetzt, wo es eine gibt.
func (s *PostingService) SetGiftRegister(g giftRegister) { s.gifts = g }

// SetFiscalYearStartMonth sagt dem Belegweg, wann das Geschäftsjahr beginnt. Die
// Freigrenze läuft je Wirtschaftsjahr, und bei einem abweichenden Geschäftsjahr
// ist das nicht das Kalenderjahr.
func (s *PostingService) SetFiscalYearStartMonth(month int) { s.fiscalStartMonth = month }

// SetFiscalYearStartReader gibt dem Belegweg den Beginn des Geschäftsjahres als
// Frage statt als Zahl.
//
// Der Unterschied ist der zwischen einem Wert von damals und einem von jetzt:
// die Einstellung wurde bisher einmal beim Einrichten des Mandanten gelesen, und
// wer den Beginn des Geschäftsjahres danach änderte, bekam bis zum nächsten
// Programmstart die Freigrenze des alten Wirtschaftsjahres.
func (s *PostingService) SetFiscalYearStartReader(read func() int) { s.fiscalStartReader = read }

func (s *PostingService) fiscalYearStartMonth() int {
	month := s.fiscalStartMonth
	if s.fiscalStartReader != nil {
		month = s.fiscalStartReader()
	}
	if month <= 0 || month > 12 {
		return 1
	}
	return month
}
