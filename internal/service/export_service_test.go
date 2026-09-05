package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/ebilanz"
	"github.com/buchfink/buchfink/internal/export"
	"github.com/buchfink/buchfink/internal/receiptstore"
	"github.com/buchfink/buchfink/internal/repository"
)

func (e *testEnv) exports(t *testing.T) *ExportService {
	t.Helper()
	svc := NewExportService(
		e.journalRepo,
		repository.NewAccountRepository(e.db),
		e.contactRepo,
		e.receiptRepo,
		repository.NewAssetRepository(e.db),
		repository.NewPaymentAllocationRepository(e.db),
		repository.NewAuditRepository(e.db),
		repository.NewSettingsRepository(e.db),
		repository.NewFestschreibungRepository(e.db),
		repository.NewVatReturnRepository(e.db),
		repository.NewCheckRunRepository(e.db),
		repository.NewFiscalYearRepository(e.db),
		receiptstore.New(e.dataDir),
		e.dataDir,
		e.fiscalYear,
	)
	svc.SetTenantName("Pfennig Ventures GmbH")
	svc.SetOpenItemSource(e.payments(t))
	return svc
}

// filledBooks legt einen kleinen, aber vollständigen Bestand an: zwei
// Eingangsrechnungen mit Beleg und Personenkonto (eine offen, eine bezahlt),
// eine Handbuchung, ein Anlagegut samt Zugang und Abschreibungslauf und eine
// gespeicherte Voranmeldung. Damit trägt jede Tabelle des Exports etwas —
// eine Tabelle, die nur leer geprüft wird, ist nicht geprüft.
func (e *testEnv) filledBooks(t *testing.T) *filledFixture {
	t.Helper()
	ctx := context.Background()

	vendor := e.vendor(t, "Kabelwerk GmbH", "DE", "DE111111111")
	invoice := e.openPayable(t, vendor.ID, 100000, domain.TaxRateStandard)

	if _, err := e.journal.Post(ctx, simpleEntry("6815", "1800", 4200)); err != nil {
		t.Fatalf("Handbuchung: %v", err)
	}

	// Eine zweite Rechnung, die bezahlt wird: erst dadurch entsteht eine
	// Zahlungszuordnung, und die Einzelpostenliste des Exports hat etwas zu
	// zeigen.
	paid := e.openPayable(t, vendor.ID, 50000, domain.TaxRateStandard)
	if _, err := e.payments(t).Settle(ctx, PaymentRequest{
		PaymentAccount: "1800",
		PaymentDate:    "2026-04-15",
		Description:    "Überweisung Kabelwerk",
		Allocations: []AllocationRequest{
			{OpenItemEntryID: paid.ID, SettledAmount: 59500},
		},
	}); err != nil {
		t.Fatalf("Zahlung buchen: %v", err)
	}

	// Ein Anlagegut mit Zugang und einem gebuchten AfA-Lauf: Stammdaten,
	// Zugangs- und Abschreibungsbewegung.
	assetSvc := e.assets(t)
	assetSvc.SetDocumentStore(e.store)
	asset := e.machine(t, assetSvc)
	if _, err := assetSvc.BookDepreciation(ctx, BookDepreciationRequest{
		FiscalYear: e.fiscalYear, BookingDate: "2026-12-31",
	}); err != nil {
		t.Fatalf("Abschreibungslauf: %v", err)
	}
	if _, err := assetSvc.AttachDocument(ctx, AttachDocumentRequest{
		AssetID: asset.ID, Content: []byte("%PDF-1.4 Kaufvertrag Fräsmaschine"),
		FileName: "kaufvertrag.pdf", Title: "Kaufvertrag",
	}); err != nil {
		t.Fatalf("Anlagendokument ablegen: %v", err)
	}

	// Der Belegeingang entscheidet über den Zeitraum des Vorsteuerabzugs
	// (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 2 UStG). Ohne ihn stünde die Voranmeldung
	// auf lauter Nullen und prüfte nichts.
	receipts, err := e.receiptRepo.FindAll(ctx, e.fiscalYear)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	for i := range receipts {
		e.setReceivedAt(t, receipts[i].ID, "2026-03-01")
	}

	// Und eine gespeicherte Voranmeldung für das erste Quartal.
	vatReturn, err := e.vatReturns(t).Save(ctx, "2026-Q1")
	if err != nil {
		t.Fatalf("Voranmeldung speichern: %v", err)
	}

	return &filledFixture{
		vendor: vendor, openInvoice: invoice, paidInvoice: paid,
		asset: asset, vatReturn: vatReturn,
	}
}

// filledFixture nennt die Stücke des Bestands beim Namen, damit ein Test sie
// wiederfindet, ohne sie aus der Datenbank zu suchen.
type filledFixture struct {
	vendor      *domain.Contact
	openInvoice *domain.JournalEntry
	paidInvoice *domain.JournalEntry
	asset       *domain.FixedAsset
	vatReturn   *domain.VatReturn
}

// --- Z3-Export -------------------------------------------------------------

// Der Export muss jede Tabelle erzeugen, und die Zeilenzahlen müssen dem
// entsprechen, was in den Büchern steht. Eine Überlassung, in der eine Tabelle
// fehlt, ist unvollständig — und das fällt beim Prüfer auf und nicht hier.
func TestExportZ3WritesEveryTable(t *testing.T) {
	env := newTestEnv(t)
	fixture := env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	result, err := env.exports(t).ExportZ3(context.Background(), env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	wanted := []string{
		"journal", "bewirtungen", "zahlungszuordnungen", "konten", "salden",
		"kontakte", "offene_posten", "anlagen", "anlagen_bewegungen", "dokumente",
		"belege", "voranmeldungen", "festschreibungen", "pruefläufe",
		"aenderungsprotokoll", "steuerschluessel", "schluesselverzeichnis",
	}
	byName := map[string]export.TableInfo{}
	for _, table := range result.Tables {
		byName[table.Name] = table
	}
	for _, name := range wanted {
		table, ok := byName[name]
		if !ok {
			t.Errorf("die Tabelle %s fehlt im Export", name)
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, table.File)); err != nil {
			t.Errorf("die Datei %s wurde nicht geschrieben: %v", table.File, err)
		}
	}

	entries, err := env.journalRepo.FindAll(context.Background(), env.fiscalYear)
	if err != nil {
		t.Fatalf("Journal lesen: %v", err)
	}
	lines := 0
	for i := range entries {
		lines += len(entries[i].Lines)
	}
	if byName["journal"].Rows != lines {
		t.Errorf("journal.csv hat %d Zeilen, das Journal %d Buchungszeilen", byName["journal"].Rows, lines)
	}
	if byName["kontakte"].Rows != 1 {
		t.Errorf("kontakte.csv hat %d Zeilen, erwartet 1", byName["kontakte"].Rows)
	}
	if byName["offene_posten"].Rows != 1 {
		t.Errorf("offene_posten.csv hat %d Zeilen, erwartet 1 (die unbezahlte Rechnung %s)",
			byName["offene_posten"].Rows, fixture.openInvoice.EntryNumber)
	}
	if byName["belege"].Rows != 2 {
		t.Errorf("belege.csv hat %d Zeilen, erwartet 2 (je eine Datei zu den beiden Eingangsrechnungen)",
			byName["belege"].Rows)
	}
	if byName["zahlungszuordnungen"].Rows != 1 {
		t.Errorf("zahlungszuordnungen.csv hat %d Zeilen, erwartet 1 (die bezahlte Rechnung %s)",
			byName["zahlungszuordnungen"].Rows, fixture.paidInvoice.EntryNumber)
	}
	if byName["dokumente"].Rows != 1 {
		t.Errorf("dokumente.csv hat %d Zeilen, erwartet 1 (der Kaufvertrag zum Anlagegut)",
			byName["dokumente"].Rows)
	}
	if byName["anlagen"].Rows != 1 {
		t.Errorf("anlagen.csv hat %d Zeilen, erwartet 1 (%s)", byName["anlagen"].Rows, fixture.asset.InventoryNumber)
	}
	// Zwei Bewegungen: der Zugang und die Abschreibung des Laufs.
	if byName["anlagen_bewegungen"].Rows != 2 {
		t.Errorf("anlagen_bewegungen.csv hat %d Zeilen, erwartet 2 (Zugang und AfA)",
			byName["anlagen_bewegungen"].Rows)
	}
	// Die Voranmeldung schreibt je belegter Kennziffer eine Zeile; leere
	// Kennziffern bleiben draußen.
	belegte := 0
	for _, line := range fixture.vatReturn.Figures {
		if line.Base != 0 || line.Tax != 0 {
			belegte++
		}
	}
	if belegte == 0 {
		t.Fatal("die gespeicherte Voranmeldung trägt keine belegte Kennziffer — die Fixture prüft dann nichts")
	}
	if byName["voranmeldungen"].Rows != belegte {
		t.Errorf("voranmeldungen.csv hat %d Zeilen, erwartet %d belegte Kennziffern",
			byName["voranmeldungen"].Rows, belegte)
	}
	if byName["steuerschluessel"].Rows != len(accounting.TaxKeyCatalog()) {
		t.Errorf("steuerschluessel.csv hat %d Zeilen, der Katalog %d",
			byName["steuerschluessel"].Rows, len(accounting.TaxKeyCatalog()))
	}
	if byName["schluesselverzeichnis"].Rows == 0 {
		t.Error("das Schlüsselverzeichnis ist leer")
	}

	// Die Bewegungen tragen Vorzeichen: der Zugang erhöht die AHK, die
	// Abschreibung erhöht die kumulierte AfA. Vertauschte Spalten machten aus
	// einem Anlagenspiegel eine Zahlenreihe ohne Aussage.
	movements := readCSV(t, filepath.Join(dir, "anlagen_bewegungen.csv"))
	kind := columnIndex(t, movements[0], "Art")
	cost := columnIndex(t, movements[0], "AHK_Veraenderung")
	depreciation := columnIndex(t, movements[0], "AfA_Veraenderung")
	seen := map[string]bool{}
	for _, row := range movements[1:] {
		seen[row[kind]] = true
		switch row[kind] {
		case string(domain.AssetMovementAcquisition):
			if row[cost] != "12000.00" {
				t.Errorf("der Zugang weist %s € AHK aus, erwartet 12000.00", row[cost])
			}
			if row[depreciation] != "0.00" {
				t.Errorf("der Zugang weist %s € AfA aus, erwartet 0.00", row[depreciation])
			}
		case string(domain.AssetMovementDepreciation):
			if !strings.HasPrefix(row[depreciation], "3000.") {
				t.Errorf("die AfA-Bewegung weist %s € aus, erwartet rund 3000 (12.000 € über 48 Monate)",
					row[depreciation])
			}
			if row[cost] != "0.00" {
				t.Errorf("die AfA-Bewegung verändert die AHK um %s € — sie darf sie nicht anfassen", row[cost])
			}
		}
	}
	if !seen[string(domain.AssetMovementAcquisition)] || !seen[string(domain.AssetMovementDepreciation)] {
		t.Errorf("anlagen_bewegungen.csv enthält nicht Zugang und AfA, sondern %v", seen)
	}

	// Die Voranmeldung führt nur belegte Kennziffern und nennt den Zeitraum.
	returns := readCSV(t, filepath.Join(dir, "voranmeldungen.csv"))
	period := columnIndex(t, returns[0], "Zeitraum")
	code := columnIndex(t, returns[0], "Kennziffer")
	base := columnIndex(t, returns[0], "Bemessungsgrundlage")
	tax := columnIndex(t, returns[0], "Steuer")
	for _, row := range returns[1:] {
		if row[period] != fixture.vatReturn.PeriodKey {
			t.Errorf("voranmeldungen.csv nennt den Zeitraum %q, erwartet %q", row[period], fixture.vatReturn.PeriodKey)
		}
		if row[code] == "" {
			t.Error("eine Zeile der Voranmeldung trägt keine Kennziffer")
		}
		if row[base] == "0.00" && row[tax] == "0.00" {
			t.Errorf("die Kennziffer %s steht mit lauter Nullen in der Datei", row[code])
		}
	}

	// Die Zahlungszuordnung nennt beide Buchungen und den Ausgleichsbetrag.
	allocations := readCSV(t, filepath.Join(dir, "zahlungszuordnungen.csv"))
	item := columnIndex(t, allocations[0], "Posten_Buchungsnummer")
	settled := columnIndex(t, allocations[0], "Ausgleichsbetrag")
	if allocations[1][item] != fixture.paidInvoice.EntryNumber {
		t.Errorf("zahlungszuordnungen.csv nennt den Posten %q, erwartet %q",
			allocations[1][item], fixture.paidInvoice.EntryNumber)
	}
	if allocations[1][settled] != "595.00" {
		t.Errorf("der Ausgleichsbetrag steht mit %s € in der Datei, erwartet 595.00", allocations[1][settled])
	}

	for _, name := range []string{export.IndexFileName, export.DTDFileName, export.FieldDocFileName, export.MetaFileName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s fehlt im Export: %v", name, err)
		}
	}
}

// Die Kontentabelle trägt das Element der HGB-Taxonomie. Ohne es müsste ein
// Prüfer die Zuordnung zur E-Bilanz erraten, obwohl Buchfink sie kennt.
func TestExportedAccountsCarryTheTaxonomyElement(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(context.Background(), env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	accounts, err := repository.NewAccountRepository(env.db).FindAll(context.Background())
	if err != nil {
		t.Fatalf("Kontenrahmen lesen: %v", err)
	}
	byNumber := map[string]domain.Account{}
	for _, a := range accounts {
		byNumber[a.Number] = a
	}

	rows := readCSV(t, filepath.Join(dir, "konten.csv"))
	number := columnIndex(t, rows[0], "Konto")
	element := columnIndex(t, rows[0], "Taxonomie_Element")

	checked := 0
	for _, row := range rows[1:] {
		want := ""
		if key, ok := accounting.StatementKeyForAccount(byNumber[row[number]]); ok {
			if e, hasElement := ebilanz.ElementFor(key); hasElement {
				want = e.Element
			}
		}
		if row[element] != want {
			t.Errorf("Konto %s: Taxonomie_Element %q, erwartet %q", row[number], row[element], want)
		}
		if row[number] == "1800" {
			checked++
			if row[element] == "" {
				t.Error("das Bankkonto 1800 trägt kein Taxonomie-Element")
			}
		}
	}
	if checked != 1 {
		t.Fatalf("das Konto 1800 steht %d-mal in konten.csv — die Stichprobe prüft nichts", checked)
	}
}

// Die Prüfsummen in export.json müssen zu den Dateien passen, die danebenliegen.
// Sonst kann der Empfänger nicht feststellen, ob der Datenträger heil ankam.
func TestExportZ3MetaChecksumsMatchTheFiles(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(context.Background(), env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	var meta export.Result
	raw, err := os.ReadFile(filepath.Join(dir, export.MetaFileName))
	if err != nil {
		t.Fatalf("%s lesen: %v", export.MetaFileName, err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("%s ist unlesbar: %v", export.MetaFileName, err)
	}
	if len(meta.Files) == 0 {
		t.Fatal("export.json führt keine Datei auf")
	}
	for _, file := range meta.Files {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Errorf("%s: %v", file.Path, err)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Errorf("%s: die Prüfsumme in export.json passt nicht zur Datei", file.Path)
		}
		if int64(len(data)) != file.Bytes {
			t.Errorf("%s: die Größe in export.json passt nicht zur Datei", file.Path)
		}
	}
	if meta.ProgramVersion == "" {
		t.Error("export.json nennt die Programmversion nicht")
	}
}

// Der Export wird im Änderungsprotokoll mit seinem Umfang festgehalten. Ohne
// diesen Eintrag ließe sich später nicht sagen, was ein Prüfer bekommen hat.
func TestExportZ3IsLogged(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(context.Background(), env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	logs, err := repository.NewAuditRepository(env.db).FindAll(context.Background(), 0)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	found := false
	for _, entry := range logs {
		if entry.Action == domain.AuditActionExport && entry.EntityType == "DATA_EXPORT" {
			found = true
			if !strings.Contains(entry.Details, "Tabellen") || !strings.Contains(entry.Details, dir) {
				t.Errorf("der Protokolleintrag nennt Umfang oder Ziel nicht: %q", entry.Details)
			}
		}
	}
	if !found {
		t.Error("der Export wurde nicht im Änderungsprotokoll festgehalten")
	}
}

// --- Die Kette von außen nachrechnen ---------------------------------------

// Der eigentliche Beweis der Datenüberlassung: ein Leser, der nur die CSV-
// Dateien hat und die Kanonisierung aus der Feldbeschreibung nachbaut, muss zu
// denselben Hashes kommen wie Buchfink. Kommt er das nicht, ist die
// Unveränderbarkeit außerhalb des Programms nicht überprüfbar — und damit für
// einen Prüfer wertlos.
func TestExportedJournalLetsAnOutsiderRecomputeTheChain(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := env.exports(t).ExportZ3(context.Background(), env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}

	journal := readCSV(t, filepath.Join(dir, "journal.csv"))
	meals := readCSV(t, filepath.Join(dir, "bewirtungen.csv"))
	if len(journal) < 2 {
		t.Fatal("journal.csv enthält keine Daten")
	}

	entries := groupJournalRows(t, journal)
	if len(entries) == 0 {
		t.Fatal("aus journal.csv ließ sich keine Buchung zusammensetzen")
	}

	prev := domain.GenesisHash
	for _, entry := range entries {
		if got := entry.head["Vorgaengerhash"]; got != prev {
			t.Fatalf("Buchung %s: Vorgängerhash %q, erwartet %q",
				entry.head["Buchungsnummer"], got, prev)
		}
		computed := recomputeEntryHash(entry, meals)
		if computed != entry.head["Eigenhash"] {
			t.Fatalf("Buchung %s: nachgerechneter Hash %s, in der Datei steht %s",
				entry.head["Buchungsnummer"], computed, entry.head["Eigenhash"])
		}
		prev = entry.head["Eigenhash"]
	}
}

// --- Archivexport ----------------------------------------------------------

// Der Archivexport legt die Originaldateien unter der Belegnummer ab, und zwar
// unter ihrem Originaldateinamen. Die Prüfsummen müssen stimmen: eine Datei,
// die anders ankommt, als sie gebucht wurde, ist kein Beleg mehr.
func TestExportArchiveCarriesTheReceiptFiles(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "archiv")
	result, err := env.exports(t).ExportArchive(context.Background(), env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Archivexport: %v", err)
	}
	if result.ReceiptFiles != 2 {
		t.Fatalf("erwartet 2 mitgegebene Belegdateien, gezählt %d", result.ReceiptFiles)
	}

	receipts, err := env.receiptRepo.FindAll(context.Background(), env.fiscalYear)
	if err != nil {
		t.Fatalf("Belege lesen: %v", err)
	}
	if len(receipts) == 0 {
		t.Fatal("es liegt kein Beleg vor")
	}

	paths := map[uint]string{}
	for i := range receipts {
		receipt := &receipts[i]
		for j := range receipt.Files {
			file := &receipt.Files[j]
			if file.Role != domain.ReceiptRoleOriginal {
				continue
			}
			// Die Vorgabe: belege/<Belegnummer>/<Originaldateiname>. Kein
			// Rollenpräfix vor dem Original — der Prüfer soll die empfangene
			// Datei unter dem Namen finden, unter dem sie eingegangen ist.
			want := filepath.Join(dir, "belege", export.SafeName(receipt.ReceiptNumber),
				export.SafeName(file.FileName))
			data, err := os.ReadFile(want)
			if err != nil {
				t.Fatalf("die Belegdatei liegt nicht unter %s: %v", want, err)
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != file.SHA256 {
				t.Errorf("%s stimmt nicht mit seiner Prüfsumme überein", want)
			}
			paths[file.ID] = "belege/" + export.SafeName(receipt.ReceiptNumber) + "/" +
				export.SafeName(file.FileName)
		}
	}
	if len(paths) == 0 {
		t.Fatal("kein Beleg trägt eine Originaldatei — der Test prüft dann nichts")
	}

	// Und belege.csv muss den Pfad nennen, sonst findet ihn niemand wieder.
	rows := readCSV(t, filepath.Join(dir, "belege.csv"))
	column := columnIndex(t, rows[0], "Pfad_im_Export")
	role := columnIndex(t, rows[0], "Rolle")
	found := 0
	for _, row := range rows[1:] {
		if row[column] == "" {
			t.Errorf("belege.csv nennt für eine Datei der Rolle %s keinen Pfad im Export", row[role])
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(row[column]))); err != nil {
			t.Errorf("belege.csv nennt den Pfad %q, dort liegt nichts: %v", row[column], err)
		}
		if row[role] == string(domain.ReceiptRoleOriginal) {
			found++
			if strings.Contains(filepath.Base(row[column]), string(domain.ReceiptRoleOriginal)+"-") {
				t.Errorf("das Original liegt als %q — die Vorgabe ist der Originaldateiname ohne Rollenpräfix",
					row[column])
			}
		}
	}
	if found != len(paths) {
		t.Errorf("belege.csv führt %d Originaldateien, im Export liegen %d", found, len(paths))
	}
}

// Zwei Dateien desselben Belegs mit demselben Namen dürfen sich im Archiv nicht
// überschreiben — sonst ginge eine von beiden verloren, ohne dass es auffällt.
func TestArchivePathsStayUniqueWithinAReceipt(t *testing.T) {
	data := &exportData{
		receipts: []domain.Receipt{{
			ReceiptNumber: "RE-2026-0001",
			Files: []domain.ReceiptFile{
				{ID: 1, Position: 1, Role: domain.ReceiptRoleOriginal, FileName: "scan.pdf"},
				{ID: 2, Position: 2, Role: domain.ReceiptRoleOriginal, FileName: "scan.pdf"},
				{ID: 3, Position: 3, Role: domain.ReceiptRoleRendering, FileName: "scan.pdf"},
			},
		}},
	}
	data.planReceiptFilePaths()

	if got := data.receiptFilePaths[1]; got != "belege/RE-2026-0001/scan.pdf" {
		t.Errorf("das erste Original liegt unter %q, erwartet belege/RE-2026-0001/scan.pdf", got)
	}
	if got := data.receiptFilePaths[3]; got != "belege/RE-2026-0001/rendering-scan.pdf" {
		t.Errorf("die erzeugte Darstellung liegt unter %q, erwartet ein Rollenpräfix", got)
	}
	seen := map[string]bool{}
	for id, path := range data.receiptFilePaths {
		if seen[path] {
			t.Errorf("die Datei %d überschreibt eine andere unter %q", id, path)
		}
		seen[path] = true
	}
}

// Die Anlagendokumente liegen im Archiv unter der Inventarnummer, und die
// Tabelle dokumente nennt Pfad und Prüfsumme. Ohne sie stünden die Verträge
// allein in export.json — einer Metadatei über den Datenträger, nicht in der
// Überlassung.
func TestExportArchiveCarriesTheAssetDocuments(t *testing.T) {
	env := newTestEnv(t)
	fixture := env.filledBooks(t)

	dir := filepath.Join(t.TempDir(), "archiv")
	result, err := env.exports(t).ExportArchive(context.Background(), env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Archivexport: %v", err)
	}
	if result.DocumentFiles != 1 {
		t.Fatalf("erwartet 1 mitgegebenes Anlagendokument, gezählt %d", result.DocumentFiles)
	}

	rows := readCSV(t, filepath.Join(dir, "dokumente.csv"))
	if len(rows) != 2 {
		t.Fatalf("dokumente.csv hat %d Zeilen (mit Kopf), erwartet 2", len(rows))
	}
	inventory := columnIndex(t, rows[0], "Inventarnummer")
	name := columnIndex(t, rows[0], "Dateiname")
	sum := columnIndex(t, rows[0], "SHA256")
	path := columnIndex(t, rows[0], "Pfad_im_Export")

	if rows[1][inventory] != fixture.asset.InventoryNumber {
		t.Errorf("dokumente.csv nennt die Inventarnummer %q, erwartet %q",
			rows[1][inventory], fixture.asset.InventoryNumber)
	}
	if rows[1][name] != "kaufvertrag.pdf" {
		t.Errorf("dokumente.csv nennt die Datei %q, erwartet kaufvertrag.pdf", rows[1][name])
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rows[1][path])))
	if err != nil {
		t.Fatalf("das Anlagendokument liegt nicht unter %s: %v", rows[1][path], err)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != rows[1][sum] {
		t.Error("die Prüfsumme in dokumente.csv passt nicht zur mitgegebenen Datei")
	}
}

// Das Prüferpaket trägt den Nachweis der Unversehrtheit. Ohne ihn ist es ein
// Archivexport mit einem anderen Namen.
func TestExportAuditPackageCarriesTheIntegrityReport(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)

	svc := env.exports(t)
	svc.SetIntegritySource(testIntegrity{journal: env.journal, receipts: env.receipts})

	dir := filepath.Join(t.TempDir(), "pruefer")
	result, err := svc.ExportAuditPackage(context.Background(), env.fiscalYear, dir)
	if err != nil {
		t.Fatalf("Prüferpaket: %v", err)
	}

	report, err := os.ReadFile(filepath.Join(dir, "integritaet.txt"))
	if err != nil {
		t.Fatalf("integritaet.txt fehlt: %v", err)
	}
	for _, want := range []string{"Hash-Chain", "Belegdateien", "Keine Unstimmigkeiten"} {
		if !strings.Contains(string(report), want) {
			t.Errorf("der Nachweis nennt %q nicht:\n%s", want, report)
		}
	}

	// Der Nachweis muss den Mandanten nennen, dessen Bücher geprüft wurden. Ein
	// Ordnerpfad an dieser Stelle nennt den falschen Betrieb — und der Nachweis
	// wäre einem anderen Prüferpaket nicht mehr zuzuordnen.
	if !strings.Contains(string(report), "Mandant:  Pfennig Ventures GmbH") {
		t.Errorf("der Nachweis nennt nicht den Mandanten:\n%s", report)
	}
	if strings.Contains(string(report), dir) {
		t.Errorf("im Nachweis steht der Exportordner %s statt des Mandanten:\n%s", dir, report)
	}

	// Die fehlende Verfahrensdokumentation muss als Hinweis erscheinen und nicht
	// stillschweigend fehlen.
	if _, err := os.Stat(filepath.Join(dir, "verfahrensdokumentation.md")); err != nil {
		if len(result.Notes) == 0 {
			t.Error("die fehlende Verfahrensdokumentation wurde nicht vermerkt")
		}
	}
}

// --- Einzelexporte ---------------------------------------------------------

func TestExportJournalCSVCoversADateWindow(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.filledBooks(t)

	late := simpleEntry("6815", "1800", 999)
	late.BookingDate = "2026-11-15"
	late.DocumentDate, late.ServiceDateFrom, late.ServiceDateTo = "2026-11-15", "2026-11-15", "2026-11-15"
	if _, err := env.journal.Post(ctx, late); err != nil {
		t.Fatalf("späte Buchung: %v", err)
	}

	path := filepath.Join(t.TempDir(), "journal.csv")
	if _, err := env.exports(t).ExportJournalCSV(ctx, "2026-01-01", "2026-06-30", path); err != nil {
		t.Fatalf("Journalexport: %v", err)
	}

	rows := readCSV(t, path)
	numbers := columnIndex(t, rows[0], "Buchungsnummer")
	for _, row := range rows[1:] {
		if row[numbers] == late.EntryNumber {
			t.Errorf("die Buchung vom 15.11. steht im Export für Januar bis Juni")
		}
	}
	if len(rows) < 2 {
		t.Error("der Zeitraumexport enthält keine Buchung, obwohl im März gebucht wurde")
	}
}

// Das Schlüsselverzeichnis kommt aus derselben Tabelle wie der Z3-Export. Zwei
// Quellen wären zwei Gelegenheiten, einen Code zu vergessen.
func TestExportKeyDirectoryMatchesTheZ3Table(t *testing.T) {
	env := newTestEnv(t)
	env.filledBooks(t)
	svc := env.exports(t)

	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := svc.ExportZ3(context.Background(), env.fiscalYear, dir); err != nil {
		t.Fatalf("Z3-Export: %v", err)
	}
	single := filepath.Join(t.TempDir(), "schluessel.csv")
	if _, err := svc.ExportKeyDirectory(context.Background(), single); err != nil {
		t.Fatalf("Schlüsselverzeichnis: %v", err)
	}

	fromPackage, err := os.ReadFile(filepath.Join(dir, "schluesselverzeichnis.csv"))
	if err != nil {
		t.Fatalf("Verzeichnis aus dem Paket lesen: %v", err)
	}
	standalone, err := os.ReadFile(single)
	if err != nil {
		t.Fatalf("einzelnes Verzeichnis lesen: %v", err)
	}
	if string(fromPackage) != string(standalone) {
		t.Error("das einzeln exportierte Schlüsselverzeichnis weicht von dem im Z3-Export ab")
	}

	// Und jeder Steuerfall, den das System kennt, muss darin stehen.
	content := string(standalone)
	for _, treatment := range domain.AllTaxTreatments() {
		if !strings.Contains(content, string(treatment)) {
			t.Errorf("der Steuerfall %s fehlt im Schlüsselverzeichnis", treatment)
		}
	}
	for _, kind := range domain.AllReceiptKinds() {
		if !strings.Contains(content, string(kind)) {
			t.Errorf("die Belegart %s fehlt im Schlüsselverzeichnis", kind)
		}
	}
}

// --- Hilfen ----------------------------------------------------------------

type testIntegrity struct {
	journal  *JournalService
	receipts *ReceiptService
}

func (c testIntegrity) VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error) {
	return c.journal.VerifyIntegrity(ctx)
}

func (c testIntegrity) VerifyReceiptFiles(ctx context.Context) (*domain.FileCheckResult, error) {
	return c.receipts.VerifyReceiptFiles(ctx)
}

// readCSV liest eine exportierte Datei nach RFC 4180 mit Semikolon.
func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s lesen: %v", filepath.Base(path), err)
	}

	var rows [][]string
	var row []string
	var field strings.Builder
	inQuotes := false
	text := string(data)
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case inQuotes && c == '"' && i+1 < len(text) && text[i+1] == '"':
			field.WriteByte('"')
			i++
		case c == '"':
			inQuotes = !inQuotes
		case !inQuotes && c == ';':
			row = append(row, field.String())
			field.Reset()
		case !inQuotes && c == '\r' && i+1 < len(text) && text[i+1] == '\n':
			row = append(row, field.String())
			field.Reset()
			rows = append(rows, row)
			row = nil
			i++
		default:
			field.WriteByte(c)
		}
	}
	return rows
}

func columnIndex(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, h := range header {
		if h == name {
			return i
		}
	}
	t.Fatalf("die Spalte %s fehlt in der Datei", name)
	return -1
}

// exportedEntry ist eine aus journal.csv zusammengesetzte Buchung.
type exportedEntry struct {
	head  map[string]string
	lines []map[string]string
}

// groupJournalRows setzt die Zeilen der Datei wieder zu Buchungen zusammen.
func groupJournalRows(t *testing.T, rows [][]string) []exportedEntry {
	t.Helper()
	header := rows[0]
	index := map[string]int{}
	for i, name := range header {
		index[name] = i
	}

	var out []exportedEntry
	order := []string{}
	byID := map[string]*exportedEntry{}
	for _, row := range rows[1:] {
		values := map[string]string{}
		for name, i := range index {
			values[name] = row[i]
		}
		id := values["Buchung_ID"]
		entry, ok := byID[id]
		if !ok {
			entry = &exportedEntry{head: values}
			byID[id] = entry
			order = append(order, id)
		}
		entry.lines = append(entry.lines, values)
	}
	for _, id := range order {
		entry := byID[id]
		sort.SliceStable(entry.lines, func(i, j int) bool {
			a, _ := strconv.Atoi(entry.lines[i]["Zeilennummer"])
			b, _ := strconv.Atoi(entry.lines[j]["Zeilennummer"])
			return a < b
		})
		out = append(out, *entry)
	}
	return out
}

// recomputeEntryHash baut die kanonische Form allein aus den exportierten
// Spalten nach — nach dem Verfahren, das feldbeschreibung.md beschreibt, und
// ohne eine Zeile des Buchfink-Codes zu benutzen.
func recomputeEntryHash(entry exportedEntry, meals [][]string) string {
	var b strings.Builder
	put := func(name, value string) {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(len(value)))
		b.WriteByte(':')
		b.WriteString(value)
		b.WriteByte('\n')
	}
	h := entry.head

	put("prev", h["Vorgaengerhash"])
	put("number", h["Buchungsnummer"])
	put("fy", h["Geschaeftsjahr"])
	put("booking_date", h["Buchungsdatum"])
	put("document_date", h["Belegdatum"])
	put("service_from", h["Leistungsbeginn"])
	put("service_to", h["Leistungsende"])
	put("value_date", h["Valuta"])
	put("description", h["Buchungstext"])
	put("source", h["Quelle"])
	put("document_number", h["Belegnummer"])
	put("receipt_hash", h["Beleg_SHA256"])
	put("tax_treatment", h["Steuerfall"])
	put("contact", h["Kontakt_ID"])
	put("bank_tx", h["Bankumsatz_ID"])
	put("kind", h["Buchungsart"])
	put("reversal_of", h["Storno_von_ID"])
	put("reversal_reason", h["Storno_Grund"])
	put("currency", h["Waehrung"])
	put("rate_micros", h["Kurs_Millionstel"])
	put("rate_source", h["Kursquelle"])
	put("rate_date", h["Kursdatum"])
	put("rule_version", h["Regelversion"])
	put("created_at", h["Erfassungszeitpunkt_UTC"])

	put("lines", strconv.Itoa(len(entry.lines)))
	for _, line := range entry.lines {
		put("line_pos", line["Zeilennummer"])
		put("line_side", line["Seite"])
		put("line_account", line["Konto"])
		put("line_amount", line["Betrag_Cent"])
		put("line_contact", line["Zeile_Kontakt_ID"])
		put("line_tax_key", line["Steuerschluessel"])
		put("line_tax_base", line["Bemessungsgrundlage_Cent"])
		put("line_text", line["Zeilentext"])
	}

	if meal := findMeal(meals, h["Buchung_ID"]); meal != nil {
		put("entertainment", "1")
		put("entertainment_place", meal["Ort"])
		put("entertainment_day", meal["Tag"])
		put("entertainment_participants", meal["Teilnehmer"])
		put("entertainment_occasion", meal["Anlass"])
	} else {
		put("entertainment", "0")
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func findMeal(rows [][]string, entryID string) map[string]string {
	if len(rows) < 2 {
		return nil
	}
	index := map[string]int{}
	for i, name := range rows[0] {
		index[name] = i
	}
	for _, row := range rows[1:] {
		if row[index["Buchung_ID"]] != entryID {
			continue
		}
		values := map[string]string{}
		for name, i := range index {
			values[name] = row[i]
		}
		return values
	}
	return nil
}

// --- Jahresgrenzen ---------------------------------------------------------

// Eine Zahlung überschreitet die Jahresgrenze: die Dezemberrechnung wird im
// Januar bezahlt (§ 252 Abs. 1 Nr. 5 HGB). Die Überlassung eines Jahres darf
// deshalb weder die Zuordnungen fremder Jahre mitliefern noch die Buchungsnummer
// der Gegenseite verschweigen — sonst ist die Zahlung nicht mehr aufzulösen,
// und das ist genau das, wofür die Einzelpostenliste da ist (GoBD Rz. 36).
func TestExportedAllocationsStayWithinTheYearAndNameBothSides(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	fx := env.filledBooks(t)

	// Eine zweite Rechnung von 2026, bezahlt im Januar 2027.
	crossYear := env.openPayable(t, fx.vendor.ID, 20000, domain.TaxRateStandard)
	payment, err := env.payments(t).Settle(ctx, PaymentRequest{
		PaymentAccount: "1800",
		PaymentDate:    "2027-01-15",
		Description:    "Überweisung im Folgejahr",
		Allocations:    []AllocationRequest{{OpenItemEntryID: crossYear.ID, SettledAmount: 23800}},
	})
	if err != nil {
		t.Fatalf("jahresübergreifende Zahlung buchen: %v", err)
	}
	if payment.FiscalYear != 2027 {
		t.Fatalf("die Zahlung liegt im Geschäftsjahr %d statt in 2027", payment.FiscalYear)
	}

	svc := env.exports(t)

	// Der Export von 2026 kennt den Posten und muss die Nummer der Zahlung aus
	// 2027 nachladen.
	rows2026 := allocationRows(t, svc, 2026)
	row, ok := rows2026[strconv.FormatUint(uint64(crossYear.ID), 10)]
	if !ok {
		t.Fatalf("die Zuordnung zum Posten aus 2026 fehlt in der Überlassung 2026: %v", rows2026)
	}
	if row["Zahlung_Buchungsnummer"] != payment.EntryNumber {
		t.Errorf("die Nummer der Zahlung aus 2027 fehlt in der Überlassung 2026: %q statt %q",
			row["Zahlung_Buchungsnummer"], payment.EntryNumber)
	}
	if row["Posten_Buchungsnummer"] != crossYear.EntryNumber {
		t.Errorf("die Nummer des Postens fehlt: %q statt %q",
			row["Posten_Buchungsnummer"], crossYear.EntryNumber)
	}

	// Der Export von 2027 kennt die Zahlung und muss die Nummer des Postens aus
	// 2026 nachladen — aber nicht die Zahlung, die 2026 vollständig abgewickelt
	// wurde.
	rows2027 := allocationRows(t, svc, 2027)
	row, ok = rows2027[strconv.FormatUint(uint64(crossYear.ID), 10)]
	if !ok {
		t.Fatalf("die Zuordnung zur Zahlung aus 2027 fehlt in der Überlassung 2027: %v", rows2027)
	}
	if row["Posten_Buchungsnummer"] != crossYear.EntryNumber {
		t.Errorf("die Nummer des Postens aus 2026 fehlt in der Überlassung 2027: %q statt %q",
			row["Posten_Buchungsnummer"], crossYear.EntryNumber)
	}
	if _, found := rows2027[strconv.FormatUint(uint64(fx.paidInvoice.ID), 10)]; found {
		t.Error("die Überlassung für 2027 enthält eine Zuordnung, deren beide Hälften in 2026 liegen")
	}
	if _, found := rows2026[strconv.FormatUint(uint64(fx.paidInvoice.ID), 10)]; !found {
		t.Error("die Zuordnung des Jahres 2026 fehlt in ihrer eigenen Überlassung")
	}
}

// allocationRows exportiert ein Jahr und liefert zahlungszuordnungen.csv, nach
// der Kennung der Postenbuchung geschlüsselt.
func allocationRows(t *testing.T, svc *ExportService, year int) map[string]map[string]string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "z3")
	if _, err := svc.ExportZ3(context.Background(), year, dir); err != nil {
		t.Fatalf("Z3-Export %d: %v", year, err)
	}
	rows := readCSV(t, filepath.Join(dir, "zahlungszuordnungen.csv"))
	header := rows[0]
	out := make(map[string]map[string]string, len(rows)-1)
	for _, row := range rows[1:] {
		fields := make(map[string]string, len(header))
		for i, name := range header {
			if i < len(row) {
				fields[name] = row[i]
			}
		}
		out[fields["Posten_Buchung_ID"]] = fields
	}
	return out
}

// Auch der Zeitraumexport muss sagen, ab wann eine Buchung festgeschrieben und
// damit unveränderbar war (GoBD Rz. 107). Die Spalte blieb leer, weil der
// Zeitraumexport die Festschreibungen gar nicht las — und ein Prüfer, der einen
// Monat anfordert, bekam die Auskunft „nirgends festgeschrieben".
func TestExportJournalCSVCarriesTheCommitmentDate(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.filledBooks(t)

	festRepo := repository.NewFestschreibungRepository(env.db)
	if err := festRepo.Create(ctx, &domain.Festschreibung{
		FiscalYear: 2026, PeriodType: "quarter", PeriodLabel: "Q1 2026",
		CutoffDate: "2026-03-31", ChainHead: domain.GenesisHash,
	}); err != nil {
		t.Fatalf("Festschreibung: %v", err)
	}

	path := filepath.Join(t.TempDir(), "journal.csv")
	if _, err := env.exports(t).ExportJournalCSV(ctx, "2026-01-01", "2026-06-30", path); err != nil {
		t.Fatalf("Journalexport: %v", err)
	}

	rows := readCSV(t, path)
	committed := columnIndex(t, rows[0], "Festgeschrieben_am")
	dates := columnIndex(t, rows[0], "Buchungsdatum")
	filled := 0
	for _, row := range rows[1:] {
		if row[dates] > "2026-03-31" {
			continue
		}
		if row[committed] == "" {
			t.Errorf("die Buchung vom %s ist festgeschrieben, die Spalte Festgeschrieben_am ist aber leer", row[dates])
			break
		}
		filled++
	}
	if filled == 0 {
		t.Error("keine Buchung des ersten Quartals im Zeitraumexport — der Test prüft nichts")
	}
}
