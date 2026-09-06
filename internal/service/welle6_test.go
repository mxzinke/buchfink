package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Belegkopfdaten und Originaldatei -------------------------------------

// filePDFReceipt legt einen Beleg mit einer ansehbaren Originaldatei ab.
func filePDFReceipt(t *testing.T, env *testEnv, req FileReceiptRequest) *domain.Receipt {
	t.Helper()
	if req.Direction == "" {
		req.Direction = domain.DirectionIncoming
	}
	if len(req.Files) == 0 {
		req.Files = []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "rechnung.pdf",
			Content: []byte("%PDF-1.4 Rechnung\n"),
		}}
	}
	receipt, err := env.receipts.File(context.Background(), req)
	if err != nil {
		t.Fatalf("Beleg konnte nicht abgelegt werden: %v", err)
	}
	return receipt
}

func TestReceiptGetsRetentionClassOnFiling(t *testing.T) {
	env := newTestEnv(t)

	invoice := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindInvoice, DocumentDate: "2026-03-01",
		IssuerName: "Lieferant GmbH", GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	})
	if invoice.RetentionClass != domain.RetentionClassVouchers {
		t.Errorf("eine Rechnung ist ein Buchungsbeleg, bekommen Klasse %q", invoice.RetentionClass)
	}
	if invoice.RetentionUntil != "2034-12-31" {
		t.Errorf("Fristende %q, erwartet 2034-12-31 (acht Jahre ab Schluss 2026)", invoice.RetentionUntil)
	}

	letter := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindLetter, DocumentDate: "2026-03-01",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "vertrag.pdf",
			Content: []byte("%PDF-1.4 Vertrag\n"),
		}},
	})
	if letter.RetentionClass != domain.RetentionClassLetters {
		t.Errorf("ein Handelsbrief ist keine Rechnung, bekommen Klasse %q", letter.RetentionClass)
	}
	if letter.RetentionUntil != "2032-12-31" {
		t.Errorf("Fristende %q, erwartet 2032-12-31 (sechs Jahre ab Schluss 2026)", letter.RetentionUntil)
	}
}

func TestOriginalFileCannotBeRemoved(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	receipt := filePDFReceipt(t, env, FileReceiptRequest{DocumentDate: "2026-03-01"})
	withAttachment, err := env.receipts.AddFile(ctx, receipt.ID, NewFile{
		Role: domain.ReceiptRoleAttachment, FileName: "zahlungsnachweis.pdf",
		Content: []byte("%PDF-1.4 Nachweis\n"),
	})
	if err != nil {
		t.Fatalf("Anhang konnte nicht ergänzt werden: %v", err)
	}

	var originalID, attachmentID uint
	for _, f := range withAttachment.Files {
		switch f.Role {
		case domain.ReceiptRoleOriginal:
			originalID = f.ID
		case domain.ReceiptRoleAttachment:
			attachmentID = f.ID
		}
	}

	// GoBD Rz. 131: eingehende Dokumente sind in der empfangenen Form
	// aufzubewahren. Ein Beleg, dessen Original sich entfernen lässt, hält
	// nichts fest.
	if _, err := env.receipts.RemoveFile(ctx, withAttachment.ID, originalID); err == nil {
		t.Fatal("die Originaldatei darf sich nicht entfernen lassen")
	} else if !strings.Contains(err.Error(), "empfangenen Form") {
		t.Errorf("die Meldung muss den Grund nennen, bekommen: %v", err)
	}

	// Ein Anhang darf gehen — und Name und Prüfsumme stehen danach im
	// Protokoll.
	removed, err := env.receipts.RemoveFile(ctx, withAttachment.ID, attachmentID)
	if err != nil {
		t.Fatalf("der Anhang muss sich entfernen lassen: %v", err)
	}
	if len(removed.Files) != 1 {
		t.Errorf("nach dem Entfernen bleibt eine Datei, bekommen %d", len(removed.Files))
	}

	entries, _ := repository.NewAuditRepository(env.db).FindFiltered(ctx, 0,
		domain.AuditFilter{EntityType: "RECEIPT"})
	var found bool
	for _, entry := range entries {
		if strings.Contains(entry.Details, "zahlungsnachweis.pdf") && strings.Contains(entry.Details, "SHA256") {
			found = true
		}
	}
	if !found {
		t.Error("das Entfernen einer Datei muss Name und Prüfsumme protokollieren")
	}
}

func TestSaveHeaderRecomputesTheReceiptHash(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Ein Papierscan kommt ohne Kopfdaten herein: die Ablage darf daran nicht
	// scheitern (GoBD Rz. 131), das Buchen schon.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{})
	if receipt.HasHeader() {
		t.Fatal("ein Beleg ohne Kopfdaten darf keine tragen")
	}
	if err := receipt.ValidateBookable(); err == nil {
		t.Error("ohne Kopfdaten darf nicht gebucht werden")
	}
	hashBefore := receipt.ReceiptHash

	updated, err := env.receipts.SaveHeader(ctx, receipt.ID, domain.ReceiptHeader{
		DocumentDate: "2026-03-01", IssuerName: "Lieferant GmbH",
		GrossAmount: 11900, TaxAmount: 1900, Currency: "EUR",
	})
	if err != nil {
		t.Fatalf("die Kopfdaten konnten nicht nachgetragen werden: %v", err)
	}
	if updated.ReceiptHash == hashBefore {
		t.Error("die Kopfdaten stehen im Beleg-Hash — er muss sich mit ihnen ändern")
	}
	if err := updated.ValidateBookable(); err != nil {
		t.Errorf("mit vollständigen Kopfdaten ist der Beleg buchbar: %v", err)
	}
	if updated.RetentionUntil == "" {
		t.Error("mit dem Belegdatum steht auch die Aufbewahrungsfrist fest")
	}
}

// --- Stornieren und neu buchen --------------------------------------------

func TestCorrectEntryLinksBothDirections(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("die ursprüngliche Buchung ist fehlgeschlagen: %v", err)
	}

	replacement := simpleEntry("6820", "1800", 12000)
	replacement.Description = "Bürobedarf, richtiges Konto"

	result, err := env.journal.CorrectEntry(ctx, original.ID, "falsches Aufwandskonto", replacement)
	if err != nil {
		t.Fatalf("die Korrektur ist fehlgeschlagen: %v", err)
	}

	if result.Reversal.ReversalOfID == nil || *result.Reversal.ReversalOfID != original.ID {
		t.Error("die Generalumkehr muss auf die ursprüngliche Buchung verweisen")
	}
	if result.Replacement.CorrectsEntryID == nil || *result.Replacement.CorrectsEntryID != original.ID {
		t.Error("die Neubuchung muss auf die ersetzte Buchung verweisen (BEL-09)")
	}

	// Die Gegenrichtung: von der stornierten Buchung zur Neubuchung. Ohne sie
	// bliebe an der alten Buchung die Frage offen, ob je richtig gebucht wurde.
	back, err := env.journal.CorrectionOf(ctx, original.ID)
	if err != nil {
		t.Fatalf("die Gegenrichtung ließ sich nicht lesen: %v", err)
	}
	if back == nil || back.ID != result.Replacement.ID {
		t.Errorf("von der ersetzten Buchung muss die Neubuchung erreichbar sein, bekommen %v", back)
	}

	// Die Kette bleibt heil: der Verweis steht außerhalb der kanonischen Form.
	check, err := env.journal.VerifyIntegrity(ctx)
	if err != nil || !check.IsValid {
		t.Errorf("die Korrektur darf die Hash-Kette nicht brechen: %v / %s", err, check.Message)
	}
}

func TestCorrectEntryDoesNotReverseWhenTheNewBookingIsInvalid(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	original, err := env.journal.Post(ctx, simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("die ursprüngliche Buchung ist fehlgeschlagen: %v", err)
	}

	broken := simpleEntry("6820", "1800", 12000)
	broken.Lines[1].Amount = 11000 // nicht ausgeglichen

	if _, err := env.journal.CorrectEntry(ctx, original.ID, "Grund", broken); err == nil {
		t.Fatal("eine nicht buchbare Neubuchung darf den Storno nicht auslösen")
	}

	// Die ursprüngliche Buchung steht unverändert da: eine halb ausgeführte
	// Korrektur wäre schlimmer als keine.
	if reversal, _ := env.journalRepo.FindReversalOf(ctx, original.ID); reversal != nil {
		t.Error("es darf keine Generalumkehr geschrieben worden sein")
	}
}

func TestPostStampsVersionAndActor(t *testing.T) {
	env := newTestEnv(t)

	entry, err := env.journal.Post(context.Background(), simpleEntry("6815", "1800", 10000))
	if err != nil {
		t.Fatalf("die Buchung ist fehlgeschlagen: %v", err)
	}
	if entry.AppVersion == "" {
		t.Error("jede Buchung trägt die Programmfassung (UNV-06)")
	}
	if entry.Actor == "" {
		t.Error("jede Buchung trägt die Bearbeiterkennung (UNV-04)")
	}
	if entry.PostingRuleVersion == "" {
		t.Error("auch eine von Hand erfasste Buchung trägt den Regelstand")
	}
}

// --- Eröffnungsbilanz des Umsteigers --------------------------------------

func TestOpeningBalanceBooksAccountsAndOpenItems(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	customer := env.customer(t, "Kunde GmbH", "DE", "")
	vendor := env.vendor(t, "Lieferant GmbH", "DE", "")
	// Die Schlussbilanz des Altsystems ist keine Rechnung: sie hat keinen
	// leistenden Unternehmer und keinen einzelnen Betrag. Ihre Kopfdaten sind
	// deshalb Belegdatum und Betreff (siehe Receipt.ValidateHeader).
	closing := filePDFReceipt(t, env, FileReceiptRequest{
		Kind: domain.ReceiptKindOther, DocumentDate: "2025-12-31",
		Subject: "Schlussbilanz zum 31.12.2025 aus dem Vorgängerprogramm",
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal, FileName: "schlussbilanz-altsystem.pdf",
			Content: []byte("%PDF-1.4 Schlussbilanz\n"),
		}},
	})

	req := OpeningBalanceRequest{
		FiscalYear:   2026,
		Date:         "2026-01-01",
		ReceiptID:    &closing.ID,
		LegacySystem: "Vorgängerprogramm",
		// Zwei Bestandskonten der Aktivseite. Die Sammelkonten der Bilanz —
		// 1200 Forderungen, 3300 Verbindlichkeiten — stehen bewusst nicht hier:
		// sie werden nicht direkt bebucht, ihre Werte kommen aus den
		// Personenkonten weiter unten.
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 500000, LegacyRef: "ALT-1800"},
			{Account: "1600", Side: domain.SideDebit, Amount: 20000, LegacyRef: "ALT-1600"},
		},
		Receivables: []OpenOpeningItem{{
			ContactID: customer.ID, Amount: 119000,
			DocumentNumber: "RE-2025-0099", DocumentDate: "2025-12-01",
			DueDate: "2026-01-15", LegacyRef: "ALT-RE-99",
		}},
		Payables: []OpenOpeningItem{{
			ContactID: vendor.ID, Amount: 59500,
			DocumentNumber: "ER-2025-0042", DocumentDate: "2025-12-10",
			LegacyRef: "ALT-ER-42",
		}},
	}

	preview, err := env.journal.PreviewOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die Vorschau ist fehlgeschlagen: %v", err)
	}
	if len(preview.Entries) != 3 {
		t.Errorf("%d Buchungen in der Vorschau, erwartet 3 (Sachkonten, Forderung, Verbindlichkeit)",
			len(preview.Entries))
	}
	if !preview.Balanced {
		t.Errorf("die Eröffnungsbilanz muss ausgeglichen sein: Soll %s, Haben %s",
			preview.DebitTotal, preview.CreditTotal)
	}

	booked, err := env.journal.BookOpeningBalance(ctx, req)
	if err != nil {
		t.Fatalf("die Eröffnungsbilanz ließ sich nicht buchen: %v", err)
	}

	var receivable *domain.JournalEntry
	for _, entry := range booked.Entries {
		if entry.Source != domain.EntrySourceOpening {
			t.Errorf("Buchung %s trägt die Quelle %q, erwartet %q",
				entry.EntryNumber, entry.Source, domain.EntrySourceOpening)
		}
		if entry.LegacyRef == "" {
			t.Errorf("Buchung %s trägt keine Herkunftskennung aus dem Altsystem", entry.EntryNumber)
		}
		if entry.ContactID != nil && *entry.ContactID == customer.ID {
			receivable = entry
		}
	}
	if receivable == nil {
		t.Fatal("die Forderung des Umsteigers wurde nicht als eigene Buchung erfasst")
	}
	if receivable.LegacyRef != "ALT-RE-99" {
		t.Errorf("Herkunftskennung %q, erwartet ALT-RE-99", receivable.LegacyRef)
	}

	// Je offener Posten eine Buchung auf dem Personenkonto gegen 9008/9009 —
	// eine Sammelbuchung ließe sich später nicht ausgleichen.
	var onLedger, onCarryforward bool
	for _, line := range receivable.Lines {
		if line.Account == customer.LedgerAccount && line.Side == domain.SideDebit {
			onLedger = true
		}
		if line.Account == domain.AccountSaldenvortraegeDebitoren {
			onCarryforward = true
		}
	}
	if !onLedger || !onCarryforward {
		t.Errorf("die Forderung muss auf dem Personenkonto gegen %s stehen: %+v",
			domain.AccountSaldenvortraegeDebitoren, receivable.Lines)
	}

	check, err := env.journal.VerifyIntegrity(ctx)
	if err != nil || !check.IsValid {
		t.Errorf("die Eröffnungsbilanz darf die Kette nicht brechen: %v / %s", err, check.Message)
	}
}

func TestOpeningBalanceNeedsTheClosingStatementAsReceipt(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.journal.PreviewOpeningBalance(context.Background(), OpeningBalanceRequest{
		FiscalYear: 2026,
		Accounts: []OpeningBalanceLine{
			{Account: "1800", Side: domain.SideDebit, Amount: 100000},
		},
	})
	if err == nil {
		t.Fatal("eine Eröffnungsbilanz ohne Beleg wäre eine Behauptung über fremde Zahlen (§ 146 Abs. 1 AO)")
	}
	if !strings.Contains(err.Error(), "Schlussbilanz") {
		t.Errorf("die Meldung muss sagen, was fehlt: %v", err)
	}
}

// --- Zeitabgleich mit dem Zeitstempeldienst -------------------------------

func TestTimeDriftNoteOnlyBeyondTolerance(t *testing.T) {
	trusted := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	if note := TimeDriftNote(trusted.Add(2*time.Minute), trusted, "TSA"); note != "" {
		t.Errorf("zwei Minuten liegen in der Toleranz, bekommen: %q", note)
	}
	if note := TimeDriftNote(trusted.Add(-2*time.Minute), trusted, "TSA"); note != "" {
		t.Errorf("zwei Minuten Rückstand liegen in der Toleranz, bekommen: %q", note)
	}

	ahead := TimeDriftNote(trusted.Add(90*time.Minute), trusted, "DigiCert")
	if ahead == "" {
		t.Fatal("90 Minuten Abweichung müssen gemeldet werden")
	}
	if !strings.Contains(ahead, "geht vor") || !strings.Contains(ahead, "DigiCert") {
		t.Errorf("die Meldung muss Richtung und Dienst nennen: %q", ahead)
	}

	behind := TimeDriftNote(trusted.Add(-3*24*time.Hour), trusted, "DigiCert")
	if !strings.Contains(behind, "geht nach") || !strings.Contains(behind, "3 Tage") {
		t.Errorf("die Meldung muss Richtung und Größe nennen: %q", behind)
	}
}

// --- Aufbewahrung ---------------------------------------------------------

func newRetentionEnv(t *testing.T) (*testEnv, *RetentionService) {
	t.Helper()
	env := newTestEnv(t)
	svc := NewRetentionService(
		repository.NewRetentionRepository(env.db),
		env.journalRepo,
		repository.NewSettingsRepository(env.db),
		repository.NewAuditRepository(env.db),
		env.store,
	)
	return env, svc
}

// postInYear schreibt eine Buchung in ein bestimmtes Geschäftsjahr, an der
// Festschreibungs- und Abschlussprüfung vorbei: die Fristen sollen geprüft
// werden, nicht der Buchungsweg.
func postInYear(t *testing.T, env *testEnv, year int) {
	t.Helper()
	date := time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	entry := &domain.JournalEntry{
		FiscalYear: year, BookingDate: date, DocumentDate: date,
		ServiceDateFrom: date, ServiceDateTo: date,
		Description: "Altbuchung", Source: domain.EntrySourceManual,
		Kind: domain.EntryKindNormal, Currency: "EUR", ExchangeRateMicros: 1_000_000,
		Lines: []domain.JournalLine{
			{Position: 1, Side: domain.SideDebit, Account: "6815", Amount: 10000},
			{Position: 2, Side: domain.SideCredit, Account: "1800", Amount: 10000},
		},
	}
	if err := env.journalRepo.Append(context.Background(), entry, accounting.NewHashChain().CalculateHash); err != nil {
		t.Fatalf("Buchung für %d konnte nicht angelegt werden: %v", year, err)
	}
}

func TestRetentionOverviewNamesTheLongestPeriodPerYear(t *testing.T) {
	env, svc := newRetentionEnv(t)
	postInYear(t, env, 2020)

	overview, err := svc.Overview(context.Background(), "2026-06-30")
	if err != nil {
		t.Fatalf("die Fristenübersicht ist fehlgeschlagen: %v", err)
	}
	if len(overview.Years) != 1 {
		t.Fatalf("%d Jahre in der Übersicht, erwartet 1", len(overview.Years))
	}
	year := overview.Years[0]

	// Das Löschen trifft das ganze Jahr, also gilt die längste Frist: zehn
	// Jahre für die Handelsbücher.
	if year.EarliestDeletion != "2031-01-01" {
		t.Errorf("frühestes Löschdatum %q, erwartet 2031-01-01", year.EarliestDeletion)
	}
	if year.Expired || year.Deletable {
		t.Error("2020 ist am 30.6.2026 noch aufzubewahren")
	}
	if len(year.Classes) != 3 {
		t.Errorf("%d Fristenklassen, erwartet 3", len(year.Classes))
	}
	if len(overview.Concept) == 0 {
		t.Error("zur Fristenübersicht gehört das Löschkonzept")
	}
}

func TestDeletionIsRefusedBeforeTheDeadlineAndUnderHold(t *testing.T) {
	env, svc := newRetentionEnv(t)
	ctx := context.Background()
	postInYear(t, env, 2020)

	// Vor Fristablauf: gesperrt, und die Meldung nennt das Datum.
	err := svc.EnsureDeletable(ctx, 2020, "2026-06-30")
	if err == nil {
		t.Fatal("vor Fristablauf darf nicht gelöscht werden")
	}
	if !strings.Contains(err.Error(), "2031-01-01") {
		t.Errorf("die Meldung muss das Datum nennen: %v", err)
	}

	// Nach Fristablauf: erlaubt.
	if err := svc.EnsureDeletable(ctx, 2020, "2031-06-30"); err != nil {
		t.Fatalf("nach Fristablauf muss gelöscht werden dürfen: %v", err)
	}

	// Mit Aussetzung: gesperrt, auch nach Fristablauf (§ 147 Abs. 3 Satz 5 AO).
	if _, err := svc.SetHold(ctx, 2020, domain.RetentionHoldAudit, "Außenprüfung 2020-2022"); err != nil {
		t.Fatalf("die Aussetzung konnte nicht eingetragen werden: %v", err)
	}
	err = svc.EnsureDeletable(ctx, 2020, "2031-06-30")
	if err == nil {
		t.Fatal("eine ausgesetzte Frist läuft nicht ab")
	}
	if !strings.Contains(err.Error(), "147") {
		t.Errorf("die Meldung muss die Norm nennen: %v", err)
	}
}

func TestArchiveAndDeleteDemandsConfirmationAndArchive(t *testing.T) {
	env, svc := newRetentionEnv(t)
	ctx := context.Background()
	postInYear(t, env, 2010)

	base := DeleteRequest{FiscalYear: 2010, Today: "2026-06-30", ArchivePath: "/tmp/archiv/2010"}

	wrong := base
	wrong.Confirmation = "ja"
	if _, err := svc.ArchiveAndDelete(ctx, wrong); err == nil {
		t.Error("ein unumkehrbarer Vorgang braucht eine ausgeschriebene Bestätigung")
	}

	noArchive := base
	noArchive.Confirmation = "2010"
	noArchive.ArchivePath = ""
	if _, err := svc.ArchiveAndDelete(ctx, noArchive); err == nil {
		t.Error("ohne Archivexport darf nicht gelöscht werden")
	}

	good := base
	good.Confirmation = "2010"
	result, err := svc.ArchiveAndDelete(ctx, good)
	if err != nil {
		t.Fatalf("nach Fristablauf muss die Löschung durchgehen: %v", err)
	}
	if result.Deleted.JournalEntries != 1 {
		t.Errorf("%d gelöschte Buchungen, erwartet 1", result.Deleted.JournalEntries)
	}

	remaining, _ := env.journalRepo.FindAll(ctx, 2010)
	if len(remaining) != 0 {
		t.Errorf("das Geschäftsjahr trägt noch %d Buchungen", len(remaining))
	}

	// Was von dem Jahr bleibt, ist der Protokolleintrag.
	entries, _ := repository.NewAuditRepository(env.db).FindFiltered(ctx, 0,
		domain.AuditFilter{EntityType: "FISCAL_YEAR_DELETED"})
	if len(entries) != 1 {
		t.Fatalf("%d Protokolleinträge zur Löschung, erwartet 1", len(entries))
	}
	if !strings.Contains(entries[0].Details, "Archiv") {
		t.Errorf("der Protokolleintrag muss das Archiv nennen: %q", entries[0].Details)
	}
}

func TestBlockedContactStaysOutOfSelections(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	keep := env.customer(t, "Bleibt GmbH", "DE", "")
	block := env.customer(t, "Löschverlangen GmbH", "DE", "")

	if _, _, err := env.contacts.BlockContact(ctx, block.ID, ""); err == nil {
		t.Error("zum Sperren gehört ein Grund")
	}

	blocked, answer, err := env.contacts.BlockContact(ctx, block.ID, "Löschverlangen nach Art. 17 DSGVO")
	if err != nil {
		t.Fatalf("das Sperren ist fehlgeschlagen: %v", err)
	}
	if !blocked.Blocked || blocked.BlockedAt == "" {
		t.Error("der gesperrte Kontakt muss die Sperre mit Datum tragen")
	}
	if !strings.Contains(answer, "Art. 17 Abs. 3 Buchst. b DSGVO") {
		t.Errorf("die Antwort an die betroffene Person muss die Norm nennen: %q", answer)
	}

	selectable, err := env.contacts.SelectableContacts(ctx)
	if err != nil {
		t.Fatalf("die Auswahl ließ sich nicht lesen: %v", err)
	}
	for _, c := range selectable {
		if c.ID == block.ID {
			t.Error("ein gesperrter Kontakt darf in keiner Auswahl mehr stehen")
		}
	}
	var keptFound bool
	for _, c := range selectable {
		if c.ID == keep.ID {
			keptFound = true
		}
	}
	if !keptFound {
		t.Error("die übrigen Kontakte bleiben wählbar")
	}

	// In der Kontaktliste bleibt er sichtbar: die Buchungen, in denen er steht,
	// sind aufzubewahren.
	all, _ := env.contacts.GetContacts(ctx)
	var stillThere bool
	for _, c := range all {
		if c.ID == block.ID {
			stillThere = true
		}
	}
	if !stillThere {
		t.Error("ein gesperrter Kontakt wird nicht gelöscht — er bleibt in Buchungen und Exporten sichtbar")
	}
}

// --- Verfahrensdokumentation ----------------------------------------------

func newProcDocService(t *testing.T, env *testEnv) *ProcDocService {
	t.Helper()
	svc := NewProcDocService(
		repository.NewSettingsRepository(env.db),
		env.numberRepo,
		repository.NewProcedureDocumentationRepository(env.db),
		repository.NewMigrationRepository(env.db),
		repository.NewAuditRepository(env.db),
		env.store,
		2026,
	)
	svc.SetEnvironment(ProcDocEnvironment{
		DataDir:      env.dataDir,
		BackupDir:    "/media/sicherung",
		BackupRhythm: "täglich",
	})
	return svc
}

func TestProcedureDocumentationCoversAllParts(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	ctx := context.Background()

	result, err := svc.Generate(ctx, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}

	for _, heading := range []string{
		procdoc.HeadingGeneral, procdoc.HeadingUser,
		procdoc.HeadingTechnical, procdoc.HeadingOperations, procdoc.HeadingControls,
	} {
		if !strings.Contains(result.Markdown, heading) {
			t.Errorf("der Abschnitt %q fehlt", heading)
		}
	}

	// Die Mandantendaten, die Nummernkreise, der Regelstand, die
	// Programmfassung und die Prüfregeln kommen aus dem System und nicht aus
	// einer Vorlage — das ist der Sinn der Erzeugung.
	for _, needle := range []string{
		"Pfennig Ventures GmbH",          // Stammdaten
		"Hauptstraße 1",                  // Anschrift
		"SKR04",                          // Kontenrahmen
		"Buchungsnummer",                 // Nummernkreis
		"{JAHR}",                         // Systematik des Nummernkreises
		"entry_without_receipt",          // Prüfregel des IKS
		"§ 257 Abs. 1 Nr. 1, Abs. 4 HGB", // Aufbewahrung
		"Ersetzendes Scannen",            // Anwenderdokumentation
		"Generalumkehr",                  // Storno-Prinzip
		"RFC 3161",                       // Zeitstempel
		"Änderungshistorie",              // CHANGELOG
		"Welle 6",                        // Eintrag der Historie
	} {
		if !strings.Contains(result.Markdown, needle) {
			t.Errorf("die Verfahrensdokumentation nennt %q nicht", needle)
		}
	}

	if result.Document.Version != "2026-09-05-1" {
		t.Errorf("Fassung %q, erwartet 2026-09-05-1", result.Document.Version)
	}
	if result.Document.AppVersion == "" || result.Document.RuleVersion == "" {
		t.Error("zur Fassung gehören Programmfassung und Regelstand")
	}
	if result.Document.SHA256 == "" || result.Document.FileName == "" {
		t.Error("die Fassung muss im Belegspeicher abgelegt sein")
	}
	if !strings.Contains(result.Document.FileName, "Verfahrensdokumentation") {
		t.Errorf("Dateiname %q, erwartet den Bezug zur Verfahrensdokumentation", result.Document.FileName)
	}
}

func TestProcedureDocumentationVersionsPerDay(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	ctx := context.Background()
	day := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)

	first, err := svc.Generate(ctx, day)
	if err != nil {
		t.Fatalf("erste Fassung: %v", err)
	}
	second, err := svc.Generate(ctx, day.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("zweite Fassung: %v", err)
	}
	if first.Document.Version == second.Document.Version {
		t.Errorf("zwei Fassungen an einem Tag dürfen nicht gleich heißen: %q", first.Document.Version)
	}
	if second.Document.Version != "2026-09-05-2" {
		t.Errorf("Fassung %q, erwartet 2026-09-05-2", second.Document.Version)
	}

	// Die Historie bleibt: GoBD Rz. 151 verlangt zu jedem Geschäftsjahr die
	// Fassung, die damals gegolten hat.
	all, err := svc.Documentations(ctx)
	if err != nil {
		t.Fatalf("die Fassungen ließen sich nicht lesen: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("%d Fassungen, erwartet 2", len(all))
	}
}

func TestOrganisationTextsComeWithSamplesAndAreLogged(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	ctx := context.Background()

	// Muster statt leerer Felder: eine Verfahrensdokumentation mit leeren
	// Abschnitten behauptet Vollständigkeit, die sie nicht hat.
	texts := svc.OrganisationTexts(ctx)
	if texts.Scanning == "" || texts.Responsibilities == "" {
		t.Fatal("die Freitextfelder müssen mit Mustern vorbelegt sein")
	}

	texts.Scanning = "Der Steuerberater scannt monatlich."
	if err := svc.SaveOrganisationTexts(ctx, texts); err != nil {
		t.Fatalf("die Organisationsanweisung ließ sich nicht speichern: %v", err)
	}

	reread := svc.OrganisationTexts(ctx)
	if reread.Scanning != "Der Steuerberater scannt monatlich." {
		t.Errorf("der gespeicherte Text kam nicht zurück: %q", reread.Scanning)
	}

	entries, _ := repository.NewAuditRepository(env.db).FindFiltered(ctx, 0,
		domain.AuditFilter{EntityType: "SETTINGS", EntityID: "ORGANISATION"})
	if len(entries) != 1 {
		t.Fatalf("%d Protokolleinträge, erwartet 1", len(entries))
	}
	if !strings.Contains(entries[0].After, "Der Steuerberater scannt monatlich.") {
		t.Errorf("das Nachher muss den neuen Text tragen: %q", entries[0].After)
	}

	result, err := svc.Generate(ctx, time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}
	if !strings.Contains(result.Markdown, "Der Steuerberater scannt monatlich.") {
		t.Error("die unternehmensindividuellen Texte gehören in die erzeugte Fassung")
	}
}

// --- Kopfdaten aus der E-Rechnung -----------------------------------------

func TestEInvoicePrefillsTheReceiptHeader(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Ein Beleg ohne Kopfdaten — so kommt eine E-Rechnung herein, bevor ihr
	// strukturierter Teil gelesen ist.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{})

	read := &domain.IncomingInvoice{
		Number:      "RE-2026-4711",
		IssueDate:   "2026-03-14",
		Currency:    "EUR",
		Supplier:    domain.InvoiceParty{Name: "Bürobedarf Schmidt e. K."},
		NetAmount:   10000,
		TaxAmount:   1900,
		GrossAmount: 11900,
	}
	if err := env.einvoices().prefillHeader(ctx, receipt.ID, read); err != nil {
		t.Fatalf("die Kopfdaten ließen sich nicht übernehmen: %v", err)
	}

	filled, err := env.receipts.Get(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("der Beleg ließ sich nicht lesen: %v", err)
	}
	if filled.DocumentDate != "2026-03-14" {
		t.Errorf("Belegdatum %q, erwartet das Rechnungsdatum aus dem Datensatz", filled.DocumentDate)
	}
	if filled.IssuerName != "Bürobedarf Schmidt e. K." {
		t.Errorf("Aussteller %q, erwartet den Lieferanten aus dem Datensatz", filled.IssuerName)
	}
	if filled.GrossAmount != 11900 || filled.TaxAmount != 1900 {
		t.Errorf("Beträge %s / %s, erwartet 119,00 / 19,00", filled.GrossAmount, filled.TaxAmount)
	}
	if filled.Subject != "RE-2026-4711" {
		t.Errorf("Betreff %q, erwartet die Rechnungsnummer", filled.Subject)
	}
	if err := filled.ValidateBookable(); err != nil {
		t.Errorf("mit den übernommenen Kopfdaten ist der Beleg buchbar: %v", err)
	}
}

func TestEInvoicePrefillDoesNotOverwriteWhatWasEntered(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Was jemand erfasst hat, bleibt stehen: die Übernahme ergänzt, sie
	// überschreibt nicht. Sonst verlöre eine von Hand berichtigte Angabe beim
	// nächsten Lesen des strukturierten Teils ihre Wirkung.
	receipt := filePDFReceipt(t, env, FileReceiptRequest{
		DocumentDate: "2026-01-02", IssuerName: "Von Hand erfasst GmbH", GrossAmount: 5000,
	})

	read := &domain.IncomingInvoice{
		Number: "RE-9", IssueDate: "2026-03-14", Currency: "EUR",
		Supplier:    domain.InvoiceParty{Name: "Aus dem Datensatz GmbH"},
		GrossAmount: 11900, TaxAmount: 1900,
	}
	if err := env.einvoices().prefillHeader(ctx, receipt.ID, read); err != nil {
		t.Fatalf("die Übernahme ist fehlgeschlagen: %v", err)
	}

	filled, _ := env.receipts.Get(ctx, receipt.ID)
	if filled.DocumentDate != "2026-01-02" || filled.IssuerName != "Von Hand erfasst GmbH" {
		t.Errorf("die erfassten Angaben wurden überschrieben: %q / %q",
			filled.DocumentDate, filled.IssuerName)
	}
	if filled.GrossAmount != 5000 {
		t.Errorf("der erfasste Betrag wurde überschrieben: %s", filled.GrossAmount)
	}
	// Was leer war, wird ergänzt.
	if filled.TaxAmount != 1900 {
		t.Errorf("der leere Steuerbetrag muss ergänzt werden, bekommen %s", filled.TaxAmount)
	}
}
