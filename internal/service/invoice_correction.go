package service

import (
	"context"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Storno und Berichtigung einer Ausgangsrechnung.
//
// Eine ausgestellte Rechnung wird nicht geändert. GoBD Rz. 58 lässt einen
// erfassten Geschäftsvorfall nicht mehr veränderbar sein, und § 14 Abs. 4
// Nr. 4 UStG lässt keine zweite Rechnung unter derselben Nummer zu. Die
// Korrektur ist deshalb ein eigenes Dokument mit eigener Nummer, das auf die
// Ursprungsrechnung verweist (BG-3) — und die Ursprungsrechnung bleibt
// unverändert im Archiv.
//
// Das Wort „Gutschrift" kommt hier nicht vor, und das ist keine
// Geschmacksfrage: eine Gutschrift im Sinne des § 14 Abs. 2 Satz 2 UStG ist die
// Abrechnung durch den Leistungsempfänger. Wer sein Storno so überschreibt,
// weist nach herrschender Auffassung der Finanzverwaltung eine Steuer nach
// § 14c Abs. 2 UStG aus.

// RegenerateDocument holt ein fehlendes Rechnungsdokument nach.
//
// Der Fall, für den es da ist: Nummer und Buchung stehen, das Erzeugen des PDF
// ist gescheitert. Die Rechnung steht dann auf „Dokument fehlt", und ohne diesen
// Weg wäre die Nummer verloren — der Anwender müsste stornieren und neu
// ausstellen, für einen Fehler, der mit der Rechnung nichts zu tun hatte.
func (s *InvoiceService) RegenerateDocument(ctx context.Context, invoiceID uint) (*domain.Invoice, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv.ReceiptID != nil {
		return nil, fmt.Errorf(
			"zu Rechnung %s liegt bereits ein Dokument vor. Ein zweites trüge dieselbe Nummer und wäre ein zweiter Beleg zu einem Vorgang",
			inv.InvoiceNumber)
	}
	if !inv.Status.IsIssued() {
		return nil, fmt.Errorf("Rechnung %s ist nicht ausgestellt", inv.InvoiceNumber)
	}

	contact, err := s.contactForInvoice(ctx, inv)
	if err != nil {
		return nil, err
	}
	if err := s.attachDocument(ctx, inv, contact); err != nil {
		return nil, err
	}
	s.log(ctx, domain.AuditActionUpdate, inv,
		fmt.Sprintf("Das Dokument zu Rechnung %s wurde nachträglich erzeugt", inv.InvoiceNumber))
	return inv, nil
}

// CancelWithDocument storniert eine Rechnung und stellt die Stornorechnung aus.
//
// Drei Dinge geschehen: die Ursprungsbuchung wird durch Generalumkehr
// zurückgenommen, es entsteht ein Stornodokument mit eigener Nummer und dem
// Bezug auf die Ursprungsrechnung, und die Ursprungsrechnung wird als storniert
// gekennzeichnet — mit Verweis auf das Dokument, das sie storniert.
//
// Das Stornodokument trägt den Tag der Korrektur. Die Buchung folgt ihm nicht:
// die Generalumkehr übernimmt Beleg- und Leistungsdatum der Ursprungsbuchung,
// und accounting.VatPeriodFor ordnet sie damit dem Voranmeldungszeitraum der
// Ursprungsrechnung zu — nicht dem Korrekturmonat.
//
// Das ist die Entscheidung aus Welle 3 und keine Nachlässigkeit. Ein Storno
// nimmt einen Umsatz zurück, den es nie gab: die Rechnung war von Anfang an
// falsch, und die Steuer ist im Ursprungszeitraum nicht entstanden (bei einer
// zu Unrecht ausgewiesenen Steuer § 14c UStG, dessen Berichtigung ohnehin die
// Zustimmung des Finanzamts braucht). § 17 Abs. 1 Satz 8 UStG greift dort, wo
// sich eine wirksam entstandene Bemessungsgrundlage *ändert* — Skonto,
// Rückzahlung, Uneinbringlichkeit; diese Fälle bucht Buchfink deshalb mit
// eigenem Datum (siehe PostAdvanceRefund und WriteOffOpenItem) und nicht als
// Generalumkehr. Ist der Ursprungszeitraum bereits übermittelt, gehört dazu
// eine berichtigte Voranmeldung (§ 153 AO); ein eigener Nachtragsweg dafür
// steht in vat_service.go bewusst noch nicht.
func (s *InvoiceService) CancelWithDocument(ctx context.Context, invoiceID uint, reason string) (*domain.Invoice, error) {
	original, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if err := ensureNotCancellation(original); err != nil {
		return nil, err
	}
	if original.Status == domain.InvoiceStatusCancelled {
		return nil, fmt.Errorf("Rechnung %s ist bereits storniert", original.InvoiceNumber)
	}
	if !original.Status.IsIssued() {
		return nil, fmt.Errorf("Rechnung %s ist nicht ausgestellt und kann nicht storniert werden", original.InvoiceNumber)
	}
	if reason == "" {
		return nil, fmt.Errorf("zu jedem Storno gehört eine Begründung (GoBD Rz. 58)")
	}
	if err := s.ensureAdvanceCancellable(ctx, original); err != nil {
		return nil, err
	}
	if err := s.ensureFinalCancellable(ctx, original); err != nil {
		return nil, err
	}

	contact, err := s.contactForInvoice(ctx, original)
	if err != nil {
		return nil, err
	}

	storno := buildStorno(original)
	// Nummer, Stornodokument, Generalumkehr und der Status der
	// Ursprungsrechnung entstehen in einer Transaktion.
	//
	// Vorher lag die Generalumkehr dahinter: scheiterte sie — eine
	// festgeschriebene Periode, ein festgestelltes Jahr —, blieb ein
	// nummeriertes Stornodokument ohne Buchung zurück, die Ursprungsrechnung
	// stand weiter auf „ausgestellt", und der nächste Versuch verbrauchte eine
	// zweite Nummer. Dieselbe Klammer wie beim Ausstellen (siehe
	// InvoiceService.Issue): entweder alle drei oder keines.
	err = s.issueCorrection(ctx, storno, contact, func(ctx context.Context) error {
		if original.JournalEntryID != nil {
			entry, err := s.postingSvc.journalSvc.Reverse(ctx, *original.JournalEntryID,
				fmt.Sprintf("Storno %s zu Rechnung %s: %s", storno.InvoiceNumber, original.InvoiceNumber, reason))
			if err != nil {
				return fmt.Errorf("die Buchung der Rechnung %s ließ sich nicht zurücknehmen: %w",
					original.InvoiceNumber, err)
			}
			storno.JournalEntryID = &entry.ID
			if err := s.invoiceRepo.Save(ctx, storno); err != nil {
				return err
			}
		}

		original.Status = domain.InvoiceStatusCancelled
		original.CancelledByInvoiceID = &storno.ID
		if err := s.invoiceRepo.Save(ctx, original); err != nil {
			return err
		}
		// Ein stornierter Abschlag fällt aus der Verrechnung der Schlussrechnung
		// heraus: mit ihm entfällt die Rechnung im Sinne des § 14 Abs. 5 Satz 2
		// UStG und damit der Grund für die Absetzung.
		if err := s.markAdvanceCancelled(ctx, original.ID); err != nil {
			return err
		}
		// Die Generalumkehr nimmt mit der Schlussrechnung auch die Auflösung
		// der Anzahlungen zurück: 3272 und 3806 stehen danach wieder im Haben.
		// Der Verbund muss das mitmachen, sonst gälten die Anzahlungen weiter
		// als verrechnet und wären nie wieder absetzbar.
		return s.reopenFinalGroup(ctx, original)
	})
	if err != nil {
		original.Status = domain.InvoiceStatusIssued
		original.CancelledByInvoiceID = nil
		return nil, fmt.Errorf("die Rechnung %s konnte nicht storniert werden: %w", original.InvoiceNumber, err)
	}

	s.log(ctx, domain.AuditActionUpdate, original, fmt.Sprintf(
		"Rechnung %s storniert mit %s: %s", original.InvoiceNumber, storno.InvoiceNumber, reason))
	return storno, nil
}

// ensureNotCancellation weist das Storno einer Stornorechnung zurück.
//
// Ein Storno des Stornos negierte die schon negierten Beträge: das zweite
// Dokument trüge wieder die Beträge der Ursprungsrechnung, und die
// Generalumkehr der Generalumkehr stellte die Forderung im Journal wieder her —
// ohne dass irgendein Dokument dem Empfänger sagt, dass die stornierte Rechnung
// wieder gelten soll. Entsteht sie doch, ist sie eine neue Rechnung mit eigener
// Nummer und eigener Leistung, keine Rücknahme einer Rücknahme.
//
// Dieselbe Sperre trägt CorrectInvoice: die Berichtigung storniert zuerst und
// kommt hier vorbei. Berichtigt wird die Ursprungsrechnung, nicht ihr Storno.
func ensureNotCancellation(inv *domain.Invoice) error {
	if inv.ResolvedKind() != domain.InvoiceKindCancellation {
		return nil
	}
	if inv.CorrectsInvoiceNumber != "" {
		return fmt.Errorf(
			"%s ist die Stornorechnung zu %s und lässt sich weder stornieren noch berichtigen; "+
				"zu berichtigen ist die Rechnung selbst",
			inv.InvoiceNumber, inv.CorrectsInvoiceNumber)
	}
	return fmt.Errorf(
		"%s ist eine Stornorechnung und lässt sich weder stornieren noch berichtigen",
		inv.InvoiceNumber)
}

// ensureAdvanceCancellable weist das Storno einer vereinnahmten
// Abschlagsrechnung zurück.
//
// Der Grund ist die Steuer. Sie ist mit der Vereinnahmung entstanden
// (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG) und bleibt geschuldet, solange das
// Geld beim Leistenden liegt: die Berichtigung nach § 17 Abs. 2 Nr. 2 UStG
// setzt die Rückzahlung des Entgelts voraus (Abschn. 17.1 Abs. 7 UStAE). Ein
// Storno, das den Abschlag bloß aus der Verrechnung nähme, ließe die
// Vereinnahmungsbuchung stehen und die Schlussrechnung die volle Steuer noch
// einmal ausweisen — der doppelte Ausweis des § 14c Abs. 1 UStG, und dazu eine
// Forderung über einen Betrag, der längst bezahlt ist.
func (s *InvoiceService) ensureAdvanceCancellable(ctx context.Context, inv *domain.Invoice) error {
	if s.groupRepo == nil || inv.ResolvedKind() != domain.InvoiceKindAdvance {
		return nil
	}
	advance, err := s.groupRepo.FindAdvanceByInvoice(ctx, inv.ID)
	if err != nil || advance == nil {
		return err
	}
	if advance.SettledInFinal {
		return fmt.Errorf(
			"die Abschlagsrechnung %s ist in der Schlussrechnung verrechnet und kann nicht mehr storniert "+
				"werden; zu berichtigen ist dann die Schlussrechnung", advance.InvoiceNumber)
	}
	if advance.Settled() {
		return fmt.Errorf(
			"die Abschlagsrechnung %s ist am %s vereinnahmt. Die darauf entstandene Umsatzsteuer wird erst "+
				"mit der Rückzahlung des Betrags berichtigt (§ 17 Abs. 2 Nr. 2 UStG). Buche zuerst die "+
				"Rückzahlung, danach lässt sich die Rechnung stornieren",
			advance.InvoiceNumber, domain.GermanDate(advance.SettledAt))
	}
	return nil
}

// CorrectInvoice storniert eine Rechnung und stellt zugleich die berichtigte
// aus.
//
// Zwei Dokumente, nicht eines: die Ursprungsrechnung wird storniert, und die
// neue Rechnung trägt den vollständigen richtigen Inhalt mit Bezug auf die
// berichtigte. Eine „Korrekturrechnung über die Differenz" wäre die andere
// Möglichkeit — sie ist zulässig, aber sie lässt den Empfänger zwei Dokumente
// zusammenrechnen, und in der Praxis rechnet er falsch.
func (s *InvoiceService) CorrectInvoice(ctx context.Context, invoiceID uint, reason string, replacement *domain.Invoice) (*domain.Invoice, error) {
	original, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	// Die Erreichbarkeit des Verbunds wird vor dem Storno geprüft und nicht
	// danach: ohne ihn ließe sich die berichtigte Abschlagsrechnung nicht
	// ausstellen, und der Anwender stünde mit einer stornierten Rechnung und
	// ohne Ersatz da.
	if replacement != nil && original.ResolvedKind() == domain.InvoiceKindAdvance {
		if _, err := s.replacementGroupOf(ctx, original); err != nil {
			return nil, err
		}
	}
	storno, err := s.CancelWithDocument(ctx, invoiceID, reason)
	if err != nil {
		return nil, err
	}
	if replacement == nil {
		return storno, nil
	}

	replacement.ID = 0
	replacement.InvoiceNumber = ""
	replacement.JournalEntryID = nil
	replacement.ReceiptID = nil
	replacement.Status = domain.InvoiceStatusDraft
	replacement.Kind = domain.InvoiceKindCorrection
	// Die berichtigte Abschlagsrechnung bleibt eine Abschlagsrechnung.
	//
	// Mit dem Typcode 384 liefe sie über den gewöhnlichen Weg: gebucht beim
	// Ausstellen, SOLL Debitor an HABEN 4400/3806. Damit entstünde die
	// Umsatzsteuer vor der Vereinnahmung, gegen § 13 Abs. 1 Nr. 1 Buchst. a
	// Satz 4 UStG — und das Ersatzdokument stünde außerhalb des Verbunds: ohne
	// offenen Posten der Quelle „Abschlag", ohne Anrechnung auf den
	// Gesamtbetrag und ohne Absetzung in der Schlussrechnung.
	if original.ResolvedKind() == domain.InvoiceKindAdvance {
		replacement.Kind = domain.InvoiceKindAdvance
	}
	// Das Format folgt der Ursprungsrechnung und nicht dem heutigen Stand des
	// Kontakts: die Berichtigung tritt an die Stelle des berichtigten Dokuments,
	// und ein zwischenzeitlich umgestelltes Empfängerprofil machte aus ihr ein
	// Dokument, das der Empfänger in dieser Form nie bekommen hat.
	if replacement.EInvoiceProfile == "" {
		replacement.EInvoiceProfile = original.EInvoiceProfile
	}
	replacement.CorrectsInvoiceID = &original.ID
	replacement.CorrectsInvoiceNumber = original.InvoiceNumber
	replacement.CorrectsInvoiceDate = original.Date
	if replacement.ContactID == 0 {
		replacement.ContactID = original.ContactID
	}
	if replacement.Date == "" {
		replacement.Date = time.Now().Format("2006-01-02")
	}

	// Die berichtigte Abschlagsrechnung entsteht im Verbund und ohne Buchung —
	// die Begründung steht an issueAdvanceReplacement.
	if original.ResolvedKind() == domain.InvoiceKindAdvance {
		if err := s.issueAdvanceReplacement(ctx, replacement, original); err != nil {
			return nil, fmt.Errorf(
				"die Abschlagsrechnung %s ist storniert, die berichtigte ließ sich aber nicht ausstellen: %w",
				original.InvoiceNumber, err)
		}
		return replacement, nil
	}

	// Die Berichtigung einer Schlussrechnung ist keine gewöhnliche Ausstellung.
	//
	// Sie muss die vereinnahmten Anzahlungen wieder absetzen — dieselbe Pflicht
	// aus § 14 Abs. 5 Satz 2 UStG, die für die stornierte galt. Liefe sie über
	// den gewöhnlichen Weg, wies sie den vollen Erlös und die volle Steuer aus,
	// ohne BT-113 und ohne die Auflösung von 3272/3806: doppelter Steuerausweis
	// (§ 14c Abs. 1 UStG) und eine Forderung über einen längst gezahlten
	// Betrag.
	if original.ResolvedKind() == domain.InvoiceKindFinal {
		if err := s.issueFinalReplacement(ctx, replacement, original); err != nil {
			return nil, fmt.Errorf(
				"die Schlussrechnung %s ist storniert, die berichtigte ließ sich aber nicht ausstellen: %w",
				original.InvoiceNumber, err)
		}
		return replacement, nil
	}

	if err := s.Issue(ctx, replacement); err != nil {
		return nil, fmt.Errorf(
			"die Ursprungsrechnung %s ist storniert, die berichtigte Rechnung ließ sich aber nicht ausstellen: %w",
			original.InvoiceNumber, err)
	}
	return replacement, nil
}

// buildStorno macht aus einer Rechnung ihre Stornorechnung.
//
// Negiert werden die Mengen und damit alle Beträge. Ein Storno mit den
// ursprünglichen Beträgen und einem Hinweis „bitte nicht beachten" ist kein
// Storno: das System des Empfängers bucht, was die Zahlen sagen.
func buildStorno(original *domain.Invoice) *domain.Invoice {
	storno := *original
	storno.ID = 0
	storno.InvoiceNumber = ""
	storno.JournalEntryID = nil
	storno.ReceiptID = nil
	storno.ZUGFeRDXML = ""
	storno.CancelledByInvoiceID = nil
	storno.SentAt, storno.SentVia, storno.SentNote = "", "", ""
	storno.Status = domain.InvoiceStatusDraft
	storno.Kind = domain.InvoiceKindCancellation
	storno.CorrectsInvoiceID = &original.ID
	storno.CorrectsInvoiceNumber = original.InvoiceNumber
	storno.CorrectsInvoiceDate = original.Date
	// Das Stornodokument trägt den Tag der Korrektur (§ 17 Abs. 1 Satz 8 UStG),
	// nicht den der Ursprungsrechnung.
	storno.Date = time.Now().Format("2006-01-02")
	storno.DueDate = storno.Date

	storno.Items = make([]domain.InvoiceItem, len(original.Items))
	copy(storno.Items, original.Items)
	for i := range storno.Items {
		storno.Items[i].ID = 0
		storno.Items[i].InvoiceID = 0
	}
	// Die Bezüge auf vorausgegangene Rechnungen (BG-3) werden kopiert und nicht
	// übernommen: mit ihren Schlüsseln zöge das Speichern des Stornos die Zeilen
	// der Ursprungsrechnung zu sich herüber, und die stornierte Schlussrechnung
	// stünde ohne die Abschläge da, die sie abgesetzt hat.
	storno.PrecedingRefs = make([]domain.InvoiceReference, len(original.PrecedingRefs))
	copy(storno.PrecedingRefs, original.PrecedingRefs)
	for i := range storno.PrecedingRefs {
		storno.PrecedingRefs[i].ID = 0
		storno.PrecedingRefs[i].InvoiceID = 0
	}
	storno.Negate()
	return &storno
}

// issueCorrection stellt ein Korrekturdokument aus: Nummer, Datensatz und —
// über book — die Buchung, die dazugehört.
//
// Bei einem Storno ist das die Generalumkehr der Ursprungsbuchung und keine
// neue Forderung; sie läuft deshalb als Rückruf innerhalb derselben
// Transaktion, in der die Nummer vergeben wird. Das Dokument entsteht danach:
// es liegt als Datei auf der Platte, und eine Datei kann keine
// Datenbanktransaktion zurücknehmen.
func (s *InvoiceService) issueCorrection(
	ctx context.Context, doc *domain.Invoice, contact *domain.Contact, book func(context.Context) error,
) error {
	doc.FiscalYear = domain.GetFiscalYearForDate(doc.Date, s.fiscalYearStartMonth(ctx))
	for i := range doc.Items {
		doc.Items[i].Position = i + 1
	}
	doc.Recalculate()

	format := s.numberFormat(ctx)
	var allocated int64
	err := s.runInTx(ctx, func(ctx context.Context) error {
		seq, err := s.numberRepo.Allocate(ctx, domain.NumberRangeInvoice, doc.FiscalYear)
		if err != nil {
			return fmt.Errorf("Rechnungsnummer konnte nicht vergeben werden: %w", err)
		}
		allocated = seq
		doc.InvoiceNumber = domain.FormatInvoiceNumberWith(format, doc.FiscalYear, seq)
		doc.Status = domain.InvoiceStatusPendingDocument
		if err := s.invoiceRepo.Save(ctx, doc); err != nil {
			return err
		}
		if book == nil {
			return nil
		}
		return book(ctx)
	})
	if err != nil {
		s.recordNumberGap(ctx, doc, allocated, err)
		resetAfterRollback(doc, allocated)
		return err
	}
	if err := s.attachDocument(ctx, doc, contact); err != nil {
		return err
	}
	s.logIssued(ctx, doc)
	return nil
}

// MarkSent vermerkt, dass die Rechnung den Empfänger erreicht hat.
//
// Buchfink versendet nicht selbst. Der Vermerk ist trotzdem mehr als eine
// Notiz: § 14 Abs. 1 Satz 1 UStG kennt die Rechnung als Abrechnung *über* eine
// Leistung gegenüber dem Empfänger, und wer im Streitfall den Zugang belegen
// muss, braucht festgehalten, wann und wie sie hinausgegangen ist.
func (s *InvoiceService) MarkSent(ctx context.Context, invoiceID uint, date string, via domain.InvoiceSentVia, note string) (*domain.Invoice, error) {
	inv, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if !inv.Status.IsIssued() {
		return nil, fmt.Errorf("Rechnung %s ist nicht ausgestellt und kann nicht versendet worden sein", inv.InvoiceNumber)
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	switch via {
	case domain.InvoiceSentViaEmail, domain.InvoiceSentViaPortal, domain.InvoiceSentViaPost, domain.InvoiceSentViaOther:
	default:
		return nil, fmt.Errorf("unbekannter Versandweg %q", via)
	}

	inv.SentAt, inv.SentVia, inv.SentNote = date, via, note
	if err := s.invoiceRepo.Save(ctx, inv); err != nil {
		return nil, err
	}
	s.log(ctx, domain.AuditActionUpdate, inv, fmt.Sprintf(
		"Rechnung %s am %s per %s versendet%s", inv.InvoiceNumber, domain.GermanDate(date), via, noteSuffix(note)))
	return inv, nil
}

func noteSuffix(note string) string {
	if note == "" {
		return ""
	}
	return ": " + note
}

// NumberGaps liefert den Lückenbericht des Rechnungsnummernkreises.
//
// Verglichen werden Zählerstand und vergebene Nummern. Der Zähler ist die
// Instanz für die Frage, wie viele Nummern ausgegeben wurden — eine gelöschte
// Zeile ändert ihn nicht, und genau das macht ihn zum Maßstab.
func (s *InvoiceService) NumberGaps(ctx context.Context, fiscalYear int) (*domain.NumberGapReport, error) {
	if fiscalYear == 0 {
		return nil, fmt.Errorf("ohne Geschäftsjahr lässt sich kein Lückenbericht erstellen")
	}
	next, err := s.numberRepo.Peek(ctx, domain.NumberRangeInvoice, fiscalYear)
	if err != nil {
		return nil, err
	}
	numbers, err := s.invoiceRepo.FindNumbers(ctx, fiscalYear)
	if err != nil {
		return nil, err
	}
	recorded := []domain.NumberGap{}
	if s.gapRepo != nil {
		recorded, err = s.gapRepo.FindByYear(ctx, domain.NumberRangeInvoice, fiscalYear)
		if err != nil {
			return nil, err
		}
	}
	report := domain.BuildNumberGapReport(fiscalYear, next, numbers, recorded, s.numberFormat(ctx))
	return &report, nil
}

// RecordNumberGapReason dokumentiert den Grund einer Lücke.
//
// Die Betriebsprüfung fragt nach fehlenden Nummern, und eine Antwort im Kopf
// des Geschäftsführers ist keine. Der Vermerk gehört an die Nummer, mit
// Zeitpunkt.
func (s *InvoiceService) RecordNumberGapReason(ctx context.Context, fiscalYear int, sequence int64, reason domain.NumberGapReason, detail string) error {
	if s.gapRepo == nil {
		return fmt.Errorf("der Lückenbericht ist nicht eingerichtet")
	}
	switch reason {
	case domain.NumberGapAborted, domain.NumberGapTest, domain.NumberGapCancelled, domain.NumberGapUnknown:
	default:
		return fmt.Errorf("unbekannter Lückengrund %q", reason)
	}
	number := domain.FormatInvoiceNumberWith(s.numberFormat(ctx), fiscalYear, sequence)
	if err := s.gapRepo.Record(ctx, &domain.NumberGap{
		Key:        domain.NumberRangeInvoice,
		FiscalYear: fiscalYear,
		Sequence:   sequence,
		Number:     number,
		Reason:     reason,
		Detail:     detail,
		RecordedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return err
	}
	// Der Grund gehört auch ins Protokoll, und nicht nur in die Tabelle.
	//
	// Er ist die Angabe, nach der die Betriebsprüfung fragt, und er wird
	// nachträglich erfasst — wer wann welche Begründung eingetragen hat, ist
	// dann selbst eine Angabe (GoBD Rz. 58). Jede andere fachliche Entscheidung
	// dieser Welle steht im Protokoll; diese stünde als einzige nur in einer
	// Zeile, die sich später anders lesen ließe.
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "NUMBER_GAP", number, fmt.Sprintf(
			"Lücke %s begründet: %s%s", number, reason.Label(), noteSuffix(detail)))
	}
	return nil
}

// contactForInvoice lädt den Empfänger einer Rechnung. Eine
// Kleinbetragsrechnung ohne erfassten Kunden hat keinen, und das ist kein
// Fehler (§ 33 UStDV).
func (s *InvoiceService) contactForInvoice(ctx context.Context, inv *domain.Invoice) (*domain.Contact, error) {
	if inv.ContactID == 0 {
		return nil, nil
	}
	contact, err := s.contactRepo.FindByID(ctx, inv.ContactID)
	if err != nil {
		return nil, fmt.Errorf("Rechnungsempfänger konnte nicht geladen werden: %w", err)
	}
	return contact, nil
}

func (s *InvoiceService) log(ctx context.Context, action domain.AuditAction, inv *domain.Invoice, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, action, "INVOICE", fmt.Sprintf("%d", inv.ID), details)
}
