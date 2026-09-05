package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/buildinfo"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/export"
	"github.com/buchfink/buchfink/internal/receiptstore"
)

// ExportOpenItemSource ist der Ausschnitt des Zahlungsflows, den der Export
// braucht: die offenen Posten zu einem Stichtag.
//
// Eine Schnittstelle mit einer Methode statt eines Verweises auf den ganzen
// Dienst — der Export soll keine Zahlung buchen können.
type ExportOpenItemSource interface {
	OpenItemsAt(ctx context.Context, cutoff string) ([]domain.OpenItem, error)
}

// ExportIntegritySource liefert die beiden Prüfläufe, die in das Prüferpaket
// gehören.
type ExportIntegritySource interface {
	VerifyIntegrity(ctx context.Context) (domain.IntegrityCheckResult, error)
	VerifyReceiptFiles(ctx context.Context) (*domain.FileCheckResult, error)
}

// ExportTaxRegisterSource liefert das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG
// als CSV.
//
// Es gehört ins Prüferpaket, weil es dort hingehört: § 5 Abs. 1 Satz 2 EStG
// verlangt ein „besonderes, laufend zu führendes Verzeichnis" für jedes
// steuerliche Wahlrecht, das vom handelsrechtlichen Ansatz abweicht — und ein
// Verzeichnis, das der Prüfer nur auf dem Bildschirm des Mandanten sehen kann,
// ist keines. Wieder eine Schnittstelle mit einer Methode: der Export soll
// lesen und nicht rechnen dürfen.
type ExportTaxRegisterSource interface {
	RegisterCSV(ctx context.Context, year int) (string, error)
}

// ExportService stellt die Datenüberlassung nach § 147 Abs. 6 AO zusammen.
//
// Er liest und schreibt keine Buchführungsdaten. Was er schreibt, sind Dateien
// im gewählten Zielordner und ein Eintrag im Änderungsprotokoll — der Umfang
// einer Überlassung gehört festgehalten, weil später niemand mehr sagen kann,
// was der Prüfer bekommen hat.
type ExportService struct {
	journalRepo    domain.JournalRepository
	accountRepo    domain.AccountRepository
	contactRepo    domain.ContactRepository
	receiptRepo    domain.ReceiptRepository
	assetRepo      domain.AssetRepository
	allocationRepo domain.PaymentAllocationRepository
	auditRepo      domain.AuditRepository
	settingsRepo   domain.SettingsRepository
	commitRepo     domain.FestschreibungRepository
	vatReturnRepo  domain.VatReturnRepository
	checkRunRepo   domain.CheckRunRepository
	fiscalYearRepo domain.FiscalYearRepository

	store    *receiptstore.Store
	dataDir  string
	tenant   string
	openItem ExportOpenItemSource
	checks   ExportIntegritySource
	register ExportTaxRegisterSource

	fiscalYear int
}

// NewExportService verdrahtet den Export.
func NewExportService(
	journalRepo domain.JournalRepository,
	accountRepo domain.AccountRepository,
	contactRepo domain.ContactRepository,
	receiptRepo domain.ReceiptRepository,
	assetRepo domain.AssetRepository,
	allocationRepo domain.PaymentAllocationRepository,
	auditRepo domain.AuditRepository,
	settingsRepo domain.SettingsRepository,
	commitRepo domain.FestschreibungRepository,
	vatReturnRepo domain.VatReturnRepository,
	checkRunRepo domain.CheckRunRepository,
	fiscalYearRepo domain.FiscalYearRepository,
	store *receiptstore.Store,
	dataDir string,
	fiscalYear int,
) *ExportService {
	return &ExportService{
		journalRepo: journalRepo, accountRepo: accountRepo, contactRepo: contactRepo,
		receiptRepo: receiptRepo, assetRepo: assetRepo, allocationRepo: allocationRepo,
		auditRepo: auditRepo, settingsRepo: settingsRepo, commitRepo: commitRepo,
		vatReturnRepo: vatReturnRepo, checkRunRepo: checkRunRepo, fiscalYearRepo: fiscalYearRepo,
		store: store, dataDir: dataDir, fiscalYear: fiscalYear,
	}
}

// SetOpenItemSource hängt den Zahlungsflow an.
func (s *ExportService) SetOpenItemSource(src ExportOpenItemSource) { s.openItem = src }

// SetIntegritySource hängt die beiden Prüfläufe an.
func (s *ExportService) SetIntegritySource(src ExportIntegritySource) { s.checks = src }

// SetTaxRegisterSource hängt das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG an.
func (s *ExportService) SetTaxRegisterSource(src ExportTaxRegisterSource) { s.register = src }

// SetTenantName setzt den Mandantennamen für die Metadaten.
func (s *ExportService) SetTenantName(name string) { s.tenant = name }

// SetFiscalYear setzt das Geschäftsjahr.
func (s *ExportService) SetFiscalYear(year int) { s.fiscalYear = year }

// --- Die drei Umfänge ------------------------------------------------------

// ExportZ3 schreibt die reine Datenüberlassung: die Tabellen, die
// Beschreibungsdatei mit ihrer Grammatik und die Feldbeschreibung.
func (s *ExportService) ExportZ3(ctx context.Context, year int, targetDir string) (*export.Result, error) {
	return s.exportPackage(ctx, year, targetDir, export.KindZ3)
}

// ExportArchive schreibt die Datenüberlassung samt den Originaldateien.
//
// Ein Prüfer, der die Zahlen bekommt, aber nicht die Belege, kann die Zahlen
// nicht prüfen: die Belegfunktion verlangt, dass jede Buchung auf ein Dokument
// zurückführt (GoBD Rz. 61).
func (s *ExportService) ExportArchive(ctx context.Context, year int, targetDir string) (*export.Result, error) {
	return s.exportPackage(ctx, year, targetDir, export.KindArchive)
}

// ExportAuditPackage schreibt das Prüferpaket: das Archiv, den Nachweis der
// Unversehrtheit und die Verfahrensdokumentation.
func (s *ExportService) ExportAuditPackage(ctx context.Context, year int, targetDir string) (*export.Result, error) {
	return s.exportPackage(ctx, year, targetDir, export.KindAuditPackage)
}

func (s *ExportService) exportPackage(
	ctx context.Context, year int, targetDir string, kind export.Kind,
) (*export.Result, error) {
	if year <= 0 {
		year = s.fiscalYear
	}
	data, err := s.collect(ctx, year)
	if err != nil {
		return nil, err
	}

	// Beim Archiv trägt jede Belegdatei ihren Pfad im Export. Er muss vor dem
	// Bau der Tabelle feststehen, weil er in belege.csv als Spalte steht: eine
	// Datei, die beiliegt, aber in der Tabelle nicht auffindbar ist, hilft
	// niemandem.
	withFiles := kind == export.KindArchive || kind == export.KindAuditPackage
	if withFiles {
		data.planReceiptFilePaths()
		data.planDocumentFilePaths()
	}

	dataset, err := s.buildDataset(data)
	if err != nil {
		return nil, err
	}

	builder, err := export.NewBuilder(targetDir, kind, dataset)
	if err != nil {
		return nil, err
	}
	if err := builder.WriteDataset(dataset); err != nil {
		return nil, err
	}

	if withFiles {
		if err := s.copyReceiptFiles(builder, data); err != nil {
			return nil, err
		}
		if err := s.copyAssetDocuments(builder, data); err != nil {
			return nil, err
		}
	}

	if kind == export.KindAuditPackage {
		if err := s.writeIntegrityReport(ctx, builder); err != nil {
			return nil, err
		}
		if err := s.writeTaxElectionRegister(ctx, builder, year); err != nil {
			return nil, err
		}
		s.copyProcessDocumentation(builder)
	}

	result, err := builder.Finish()
	if err != nil {
		return nil, err
	}

	s.logExport(ctx, result)
	return result, nil
}

// ExportJournalCSV schreibt das Journal eines beliebigen Zeitraums als einzelne
// Datei und liefert ihren Pfad.
//
// Derselbe Tabellenaufbau wie im Z3-Export: ein Prüfer, der zuerst einen Monat
// und später das Jahr anfordert, bekommt zweimal dieselben Spalten.
func (s *ExportService) ExportJournalCSV(ctx context.Context, from, to, targetPath string) (string, error) {
	if targetPath == "" {
		return "", fmt.Errorf("kein Ziel für den Journalexport angegeben")
	}
	entries, err := s.journalRepo.FindByBookingDateRange(ctx, 0, from, to)
	if err != nil {
		return "", fmt.Errorf("die Buchungen konnten nicht gelesen werden: %w", err)
	}

	data, err := s.baseData(ctx)
	if err != nil {
		return "", err
	}
	data.entries = entries
	data.from, data.to = from, to
	data.indexEntries()
	if err := s.loadCommitsForEntries(ctx, data); err != nil {
		return "", err
	}

	table, err := journalTable(data)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, export.RenderCSV(table), 0o600); err != nil {
		return "", fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(targetPath), err)
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "JOURNAL", fmt.Sprintf("%s..%s", from, to),
			fmt.Sprintf("Journalexport %s bis %s: %d Buchungen, %d Zeilen (Buchfink %s)",
				from, to, len(entries), len(table.Rows), buildinfo.Version))
	}
	return targetPath, nil
}

// ExportKeyDirectory schreibt das Schlüsselverzeichnis als einzelne Datei.
//
// Es ist dieselbe Tabelle wie im Z3-Export — eine Quelle, zwei Wege hinaus.
func (s *ExportService) ExportKeyDirectory(ctx context.Context, targetPath string) (string, error) {
	if targetPath == "" {
		return "", fmt.Errorf("kein Ziel für das Schlüsselverzeichnis angegeben")
	}
	table, err := keyDirectoryTable()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, export.RenderCSV(table), 0o600); err != nil {
		return "", fmt.Errorf("%s konnte nicht geschrieben werden: %w", filepath.Base(targetPath), err)
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "KEY_DIRECTORY", "",
			fmt.Sprintf("Schlüsselverzeichnis exportiert: %d Einträge (Buchfink %s)",
				len(table.Rows), buildinfo.Version))
	}
	return targetPath, nil
}

// KeyDirectory liefert das Schlüsselverzeichnis als Tabelle, ohne es zu
// schreiben. Die Oberfläche zeigt es damit an.
func (s *ExportService) KeyDirectory() (export.Table, error) { return keyDirectoryTable() }

// --- Datenbeschaffung ------------------------------------------------------

// balanceRow sind die Verkehrszahlen eines Kontos, getrennt nach Vortrag und
// laufenden Buchungen.
type balanceRow struct {
	opening       domain.Cents
	debit, credit domain.Cents
	count         int
}

// exportData hält alles, was die Tabellen brauchen, in einem Zug gelesen.
//
// Ein Zug, nicht dreizehn: die Tabellen verweisen aufeinander — die Belegzeile
// nennt die Buchungsnummer, die Zahlungszuordnung beide Buchungen —, und
// Auswertungen aus verschiedenen Lesezeitpunkten könnten einen Stand zeigen,
// den es so nie gab.
type exportData struct {
	year     int
	from, to string

	entries     []domain.JournalEntry
	entryNames  map[uint]string
	accounts    []domain.Account
	chart       *accounting.Chart
	ledgerNames map[string]string
	contacts    []domain.Contact
	receipts    []domain.Receipt
	assets      []domain.FixedAsset
	documents   []domain.AssetDocument
	auditLog    []domain.AuditLogEntry
	vatReturns  []domain.VatReturn
	commits     []domain.Festschreibung
	checkRuns   []domain.CheckRun
	openItems   []domain.OpenItem
	allocations []domain.PaymentAllocation
	balances    map[string]balanceRow

	// receiptFilePaths ist der Pfad je Belegdatei innerhalb des Exports,
	// documentFilePaths derselbe je Anlagendokument. Beide leer, solange keine
	// Dateien beiliegen.
	receiptFilePaths  map[uint]string
	documentFilePaths map[uint]string

	tenantName string
	// supplierLocation ist der Sitz des Betriebs. Der Beschreibungsstandard
	// erwartet ihn im Element Location der index.xml.
	supplierLocation string
	programVersion   string
	createdAt        string
}

func (d *exportData) accountName(number string) string {
	if name, ok := d.ledgerNames[number]; ok {
		return name
	}
	if d.chart != nil {
		return d.chart.Name(number)
	}
	return number
}

func (d *exportData) entryNumber(id uint) string { return d.entryNames[id] }

// inventoryNumber liefert die Inventarnummer eines Wirtschaftsguts.
func (d *exportData) inventoryNumber(assetID uint) string {
	for i := range d.assets {
		if d.assets[i].ID == assetID {
			return d.assets[i].InventoryNumber
		}
	}
	return ""
}

// committedOn liefert den Tag, an dem der Zeitraum der Buchung festgeschrieben
// wurde — die früheste Festschreibung desselben Geschäftsjahres, deren Stichtag
// die Buchung noch einschließt.
func (d *exportData) committedOn(e *domain.JournalEntry) string {
	best := ""
	bestCutoff := ""
	for i := range d.commits {
		c := &d.commits[i]
		if c.FiscalYear != e.FiscalYear || c.CutoffDate < e.BookingDate {
			continue
		}
		if bestCutoff == "" || c.CutoffDate < bestCutoff {
			bestCutoff = c.CutoffDate
			best = c.CreatedAt.Format("2006-01-02")
		}
	}
	return best
}

func (d *exportData) indexEntries() {
	d.entryNames = make(map[uint]string, len(d.entries))
	for i := range d.entries {
		d.entryNames[d.entries[i].ID] = d.entries[i].EntryNumber
	}
}

// planReceiptFilePaths legt fest, wohin jede Belegdatei im Export kommt:
// belege/<Belegnummer>/<Originaldateiname>.
//
// Das Original behält seinen Namen. Es ist die Datei, auf die es ankommt — die
// empfangene Rechnung liegt im Archiv als „rechnung.pdf" und nicht als
// „original-rechnung.pdf", damit der Prüfer sie unter dem Namen findet, unter
// dem sie eingegangen ist (GoBD Rz. 131). Rendering, strukturierter Datensatz
// und Anlagen tragen ihre Rolle vorn, weil sie sonst gleich hießen wie das
// Original, aus dem sie entstanden sind.
func (d *exportData) planReceiptFilePaths() {
	d.receiptFilePaths = make(map[uint]string)
	for i := range d.receipts {
		r := &d.receipts[i]
		dir := "belege/" + export.SafeName(r.ReceiptNumber)
		files := append([]domain.ReceiptFile(nil), r.Files...)
		sort.SliceStable(files, func(a, b int) bool { return files[a].Position < files[b].Position })

		taken := map[string]bool{}
		for _, f := range files {
			name := export.SafeName(f.FileName)
			if f.Role != domain.ReceiptRoleOriginal {
				name = export.SafeName(fmt.Sprintf("%s-%s", f.Role, f.FileName))
			}
			// Zwei Originale desselben Belegs dürfen denselben Namen tragen —
			// zwei Scans „scan.pdf" etwa. Die Position entscheidet dann, und
			// sie ist innerhalb eines Belegs eindeutig.
			if taken[name] {
				name = fmt.Sprintf("%d-%s", f.Position, name)
			}
			taken[name] = true
			d.receiptFilePaths[f.ID] = dir + "/" + name
		}
	}
}

// planDocumentFilePaths legt fest, wohin jedes Anlagendokument kommt:
// dokumente/<Inventarnummer>/<Dateiname>.
func (d *exportData) planDocumentFilePaths() {
	d.documentFilePaths = make(map[uint]string)
	taken := map[string]bool{}
	for i := range d.documents {
		doc := &d.documents[i]
		dir := "dokumente/" + export.SafeName(d.inventoryNumber(doc.AssetID))
		name := export.SafeName(doc.FileName)
		if taken[dir+"/"+name] {
			name = fmt.Sprintf("%d-%s", doc.ID, name)
		}
		taken[dir+"/"+name] = true
		d.documentFilePaths[doc.ID] = dir + "/" + name
	}
}

// baseData liest, was jeder Export braucht, unabhängig vom Geschäftsjahr.
func (s *ExportService) baseData(ctx context.Context) (*exportData, error) {
	accounts, err := s.accountRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("der Kontenrahmen konnte nicht gelesen werden: %w", err)
	}
	contacts, err := s.contactRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("die Geschäftspartner konnten nicht gelesen werden: %w", err)
	}

	ledgerNames := make(map[string]string, len(contacts))
	for _, c := range contacts {
		ledgerNames[c.LedgerAccount] = c.Name
	}

	name := s.tenant
	location := ""
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil {
			if name == "" {
				name = cfg.CompanyName
			}
			location = companyLocation(cfg)
		}
	}

	return &exportData{
		accounts:          accounts,
		chart:             accounting.NewChart(accounts),
		ledgerNames:       ledgerNames,
		contacts:          contacts,
		entryNames:        map[uint]string{},
		receiptFilePaths:  map[uint]string{},
		documentFilePaths: map[uint]string{},
		balances:          map[string]balanceRow{},
		tenantName:        name,
		supplierLocation:  location,
		programVersion:    buildinfo.Version,
		createdAt:         time.Now().Format(time.RFC3339),
	}, nil
}

// collect liest alles, was eine vollständige Überlassung eines Geschäftsjahres
// braucht.
func (s *ExportService) collect(ctx context.Context, year int) (*exportData, error) {
	d, err := s.baseData(ctx)
	if err != nil {
		return nil, err
	}
	d.year = year
	d.from, d.to = s.fiscalYearBounds(ctx, year)

	if d.entries, err = s.journalRepo.FindAll(ctx, year); err != nil {
		return nil, fmt.Errorf("die Buchungen konnten nicht gelesen werden: %w", err)
	}
	d.indexEntries()
	d.balances = computeBalances(d.entries)

	if d.receipts, err = s.receiptRepo.FindAll(ctx, year); err != nil {
		return nil, fmt.Errorf("die Belege konnten nicht gelesen werden: %w", err)
	}
	if s.assetRepo != nil {
		if d.assets, err = s.assetRepo.FindAll(ctx); err != nil {
			return nil, fmt.Errorf("die Anlagenkartei konnte nicht gelesen werden: %w", err)
		}
		for i := range d.assets {
			movements, err := s.assetRepo.FindMovements(ctx, d.assets[i].ID)
			if err != nil {
				return nil, fmt.Errorf("die Anlagenbewegungen konnten nicht gelesen werden: %w", err)
			}
			d.assets[i].Movements = movements
		}
		// Am Stück und nicht je Anlagegut: die Kartei sonst n-mal zu lesen
		// wäre nicht nur langsam, sondern gäbe auch n Lesezeitpunkte.
		if d.documents, err = s.assetRepo.FindAllDocuments(ctx); err != nil {
			return nil, fmt.Errorf("die Anlagendokumente konnten nicht gelesen werden: %w", err)
		}
	}
	if s.allocationRepo != nil {
		all, err := s.allocationRepo.FindAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("die Zahlungszuordnungen konnten nicht gelesen werden: %w", err)
		}
		if err := s.selectAllocations(ctx, d, all); err != nil {
			return nil, err
		}
	}
	if s.auditRepo != nil {
		// Kein Limit: das Änderungsprotokoll ist der Nachweis, und ein
		// Nachweis, der bei zweihundert Zeilen aufhört, ist keiner.
		if d.auditLog, err = s.auditRepo.FindAll(ctx, 0); err != nil {
			return nil, fmt.Errorf("das Änderungsprotokoll konnte nicht gelesen werden: %w", err)
		}
		sort.SliceStable(d.auditLog, func(i, j int) bool { return d.auditLog[i].ID < d.auditLog[j].ID })
	}
	if s.vatReturnRepo != nil {
		if d.vatReturns, err = s.vatReturnRepo.FindByFiscalYear(ctx, year); err != nil {
			return nil, fmt.Errorf("die Voranmeldungen konnten nicht gelesen werden: %w", err)
		}
	}
	if s.commitRepo != nil {
		if d.commits, err = s.commitRepo.FindByFiscalYear(ctx, year); err != nil {
			return nil, fmt.Errorf("die Festschreibungen konnten nicht gelesen werden: %w", err)
		}
	}
	if s.checkRunRepo != nil {
		if d.checkRuns, err = s.checkRunRepo.FindByFiscalYear(ctx, year); err != nil {
			return nil, fmt.Errorf("die Prüfläufe konnten nicht gelesen werden: %w", err)
		}
	}
	if s.openItem != nil {
		if d.openItems, err = s.openItem.OpenItemsAt(ctx, d.to); err != nil {
			return nil, fmt.Errorf("die offenen Posten konnten nicht ermittelt werden: %w", err)
		}
	}
	return d, nil
}

// selectAllocations nimmt die Zuordnungen auf, die das Exportjahr berühren, und
// besorgt die Buchungsnummern der Gegenseite.
//
// Zwei Gründe für beides. Erstens: eine Überlassung für 2026 darf nicht die
// Zahlungen von 2024 mitliefern — sie ist auf das angeforderte Jahr begrenzt,
// und ein Prüfer, der mehr bekommt, als er verlangt hat, bekommt Daten, für die
// er keine Anforderung hat. Zweitens: eine Zuordnung hat naturgemäß zwei
// Hälften in verschiedenen Jahren — die Dezemberrechnung wird im Januar bezahlt
// (§ 252 Abs. 1 Nr. 5 HGB). Die Gegenseite liegt dann außerhalb des Exportjahres
// und ihre Buchungsnummer nicht im Verzeichnis; ohne Nachladen bliebe die Spalte
// leer und die Zahlung wäre nicht mehr aufzulösen.
func (s *ExportService) selectAllocations(
	ctx context.Context, d *exportData, all []domain.PaymentAllocation,
) error {
	d.allocations = make([]domain.PaymentAllocation, 0, len(all))
	missing := make([]uint, 0)
	note := func(id uint) {
		if id == 0 {
			return
		}
		if _, known := d.entryNames[id]; !known {
			missing = append(missing, id)
		}
	}
	for _, a := range all {
		_, payment := d.entryNames[a.PaymentEntryID]
		_, item := d.entryNames[a.OpenItemEntryID]
		if !payment && !item {
			continue
		}
		d.allocations = append(d.allocations, a)
		note(a.PaymentEntryID)
		note(a.OpenItemEntryID)
	}

	// Nur die Nummer wird nachgeladen und nicht die Buchung: die Gegenseite
	// gehört in die Überlassung eines anderen Jahres, hier steht sie allein als
	// Verweis.
	for _, id := range missing {
		if _, done := d.entryNames[id]; done {
			continue
		}
		entry, err := s.journalRepo.FindByID(ctx, id)
		if err != nil || entry == nil {
			// Eine gelöschte oder unlesbare Gegenbuchung macht die Zuordnung
			// nicht wertlos: Kennung und Betrag stehen weiter in der Zeile.
			continue
		}
		d.entryNames[id] = entry.EntryNumber
	}
	return nil
}

// loadCommitsForEntries liest die Festschreibungen aller Geschäftsjahre, in die
// die gelesenen Buchungen fallen.
//
// Der Zeitraumexport geht über Jahresgrenzen — „Dezember bis Januar" ist eine
// gewöhnliche Anforderung. Ohne die Festschreibungen dieser Jahre bliebe die
// Spalte Festgeschrieben_am leer, und die Überlassung sagte nichts darüber, ab
// wann die Buchung unveränderbar war (GoBD Rz. 107).
func (s *ExportService) loadCommitsForEntries(ctx context.Context, d *exportData) error {
	if s.commitRepo == nil {
		return nil
	}
	seen := make(map[int]bool, 2)
	years := make([]int, 0, 2)
	for i := range d.entries {
		year := d.entries[i].FiscalYear
		if year == 0 || seen[year] {
			continue
		}
		seen[year] = true
		years = append(years, year)
	}
	sort.Ints(years)

	d.commits = make([]domain.Festschreibung, 0, len(years))
	for _, year := range years {
		commits, err := s.commitRepo.FindByFiscalYear(ctx, year)
		if err != nil {
			return fmt.Errorf("die Festschreibungen konnten nicht gelesen werden: %w", err)
		}
		d.commits = append(d.commits, commits...)
	}
	return nil
}

// computeBalances trennt Vortrag und laufende Buchungen je Konto.
func computeBalances(entries []domain.JournalEntry) map[string]balanceRow {
	out := map[string]balanceRow{}
	for i := range entries {
		e := &entries[i]
		opening := e.Source == domain.EntrySourceOpening
		for _, l := range e.Lines {
			row := out[l.Account]
			row.count++
			switch {
			case opening && l.Side == domain.SideDebit:
				row.opening += l.Amount
			case opening:
				row.opening -= l.Amount
			case l.Side == domain.SideDebit:
				row.debit += l.Amount
			default:
				row.credit += l.Amount
			}
			out[l.Account] = row
		}
	}
	return out
}

// fiscalYearBounds liefert Anfang und Ende des Geschäftsjahres.
func (s *ExportService) fiscalYearBounds(ctx context.Context, year int) (string, string) {
	if s.fiscalYearRepo != nil {
		if fy, err := s.fiscalYearRepo.FindByYear(ctx, year); err == nil && fy != nil &&
			fy.StartDate != "" && fy.EndDate != "" {
			return fy.StartDate, fy.EndDate
		}
	}
	startMonth := 1
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil && cfg.FiscalYearStartMonth > 0 {
			startMonth = cfg.FiscalYearStartMonth
		}
	}
	start := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// buildDataset baut alle Tabellen.
func (s *ExportService) buildDataset(d *exportData) (*export.Dataset, error) {
	builders := []func(*exportData) (export.Table, error){
		journalTable, entertainmentTable, allocationsTable,
		accountsTable, balancesTable, contactsTable, openItemsTable,
		assetsTable, assetMovementsTable,
		receiptsTable, documentsTable, vatReturnsTable, commitsTable, checkRunsTable, auditLogTable,
	}

	dataset := &export.Dataset{
		TenantName:       d.tenantName,
		SupplierLocation: d.supplierLocation,
		FiscalYear:       d.year,
		From:             d.from,
		To:               d.to,
		CreatedAt:        d.createdAt,
		ProgramVersion:   d.programVersion,
		Tables:           make([]export.Table, 0, len(builders)+2),
	}
	for _, build := range builders {
		t, err := build(d)
		if err != nil {
			return nil, err
		}
		dataset.Tables = append(dataset.Tables, t)
	}

	taxKeys, err := taxKeysTable()
	if err != nil {
		return nil, err
	}
	keys, err := keyDirectoryTable()
	if err != nil {
		return nil, err
	}
	dataset.Tables = append(dataset.Tables, taxKeys, keys)
	return dataset, nil
}

// --- Dateien ---------------------------------------------------------------

// copyReceiptFiles legt die Originaldateien unter belege/<Belegnummer>/ ab.
//
// Eine fehlende oder beschädigte Datei bricht den Export nicht ab: ein Prüfer,
// der neunundneunzig von hundert Belegen bekommt, ist besser gestellt als
// einer, der wegen des hundertsten gar nichts bekommt. Der Ausfall steht als
// Hinweis im Ergebnis und in export.json.
func (s *ExportService) copyReceiptFiles(b *export.Builder, d *exportData) error {
	if s.store == nil {
		b.Note("Die Belegdateien konnten nicht beigelegt werden: kein Belegspeicher eingerichtet.")
		return nil
	}
	for i := range d.receipts {
		r := &d.receipts[i]
		for j := range r.Files {
			f := &r.Files[j]
			rel, ok := d.receiptFilePaths[f.ID]
			if !ok {
				continue
			}
			source := filepath.Join(s.dataDir, filepath.FromSlash(f.StoredPath))
			sum, err := b.CopyFile(rel, source)
			if err != nil {
				b.Note("Beleg %s, Datei %s: %v", r.ReceiptNumber, f.FileName, err)
				continue
			}
			if sum != f.SHA256 {
				b.Note("Beleg %s, Datei %s: die Prüfsumme der Datei weicht von der abgelegten ab (erwartet %s, gefunden %s).",
					r.ReceiptNumber, f.FileName, f.SHA256, sum)
			}
			b.CountReceiptFile()
		}
	}
	return nil
}

// copyAssetDocuments legt Verträge, Gutachten und Zulassungen unter
// dokumente/<Inventarnummer>/ ab.
func (s *ExportService) copyAssetDocuments(b *export.Builder, d *exportData) error {
	if s.store == nil {
		return nil
	}
	for i := range d.documents {
		doc := &d.documents[i]
		rel, ok := d.documentFilePaths[doc.ID]
		if !ok {
			continue
		}
		inventory := d.inventoryNumber(doc.AssetID)
		source := filepath.Join(s.dataDir, filepath.FromSlash(doc.StoredPath))
		sum, err := b.CopyFile(rel, source)
		if err != nil {
			b.Note("Anlagegut %s, Dokument %s: %v", inventory, doc.FileName, err)
			continue
		}
		if sum != doc.SHA256 {
			b.Note("Anlagegut %s, Dokument %s: die Prüfsumme weicht von der abgelegten ab (erwartet %s, gefunden %s).",
				inventory, doc.FileName, doc.SHA256, sum)
		}
		b.CountDocumentFile()
	}
	return nil
}

// writeIntegrityReport legt den Nachweis der Unversehrtheit bei.
//
// Er gehört ins Prüferpaket und nicht nur auf den Bildschirm: die Aussage
// „die Kette ist ungebrochen" ist Teil dessen, was überlassen wird.
func (s *ExportService) writeIntegrityReport(ctx context.Context, b *export.Builder) error {
	if s.checks == nil {
		b.Note("Der Integritätsnachweis fehlt: die Prüfung war nicht verfügbar.")
		return nil
	}

	chain, err := s.checks.VerifyIntegrity(ctx)
	if err != nil {
		return fmt.Errorf("die Integritätsprüfung ist fehlgeschlagen: %w", err)
	}
	files, err := s.checks.VerifyReceiptFiles(ctx)
	if err != nil {
		return fmt.Errorf("der Belegprüflauf ist fehlgeschlagen: %w", err)
	}

	report := fmt.Sprintf(`Nachweis der Unversehrtheit
===========================

Mandant:  %s
Geprüft:  %s
Programm: Buchfink %s

Hash-Chain über alle Geschäftsjahre
-----------------------------------
Geprüfte Geschäftsjahre: %v
Geprüfte Buchungen:      %d
Ergebnis:                %s
%s

Belegdateien
------------
Geprüfte Dateien: %d
Unversehrt:       %d
Beschädigt:       %d
Fehlend:          %d
Ergebnis:         %s
%s
Das Verfahren, mit dem sich die Hash-Chain aus den Dateien dieses Exports
nachrechnen lässt, steht in feldbeschreibung.md.
`,
		s.tenantLabel(ctx), chain.CheckedAt, buildinfo.Version,
		chain.FiscalYears, chain.TotalEntries, chain.Message,
		formatBreaks(chain.Breaks),
		files.Checked, files.Intact, files.Damaged, files.Missing, files.Message,
		formatFileIssues(files.Issues),
	)
	return b.WriteFile("integritaet.txt", []byte(report))
}

// taxRegisterFileName ist der Name des Verzeichnisses im Prüferpaket.
const taxRegisterFileName = "verzeichnis-5-1-2-estg.csv"

// writeTaxElectionRegister legt das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG bei.
//
// Fehlt die Quelle, wird das vermerkt und nicht verschwiegen — wie bei der
// Verfahrensdokumentation: ein Paket, das ohne Hinweis ohne das Verzeichnis
// ankommt, sieht vollständig aus.
func (s *ExportService) writeTaxElectionRegister(
	ctx context.Context, b *export.Builder, year int,
) error {
	if s.register == nil {
		b.Note("Das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG fehlt: die Quelle war nicht verfügbar.")
		return nil
	}
	content, err := s.register.RegisterCSV(ctx, year)
	if err != nil {
		return fmt.Errorf("das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG konnte nicht gestellt werden: %w", err)
	}
	return b.WriteFile(taxRegisterFileName, []byte(content))
}

// tenantLabel ist der Name des Mandanten für den Nachweis.
//
// Der Name und nicht der Exportordner: der Nachweis sagt aus, wessen Bücher
// geprüft wurden, und ein Pfad nennt allenfalls, wo das Ergebnis landet. Fehlt
// der Name an der Bridge, kommt er aus den Unternehmensdaten — dieselbe Quelle,
// aus der ihn auch die Beschreibungsdatei nimmt.
func (s *ExportService) tenantLabel(ctx context.Context) string {
	if strings.TrimSpace(s.tenant) != "" {
		return s.tenant
	}
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil &&
			strings.TrimSpace(cfg.CompanyName) != "" {
			return cfg.CompanyName
		}
	}
	return "(Mandantenname nicht hinterlegt)"
}

func formatBreaks(breaks []domain.IntegrityBreak) string {
	if len(breaks) == 0 {
		return "Keine Unstimmigkeiten.\n"
	}
	out := ""
	for _, br := range breaks {
		// Die drei Marken sind auf dieselbe Spalte gesetzt: erwarteter und
		// tatsächlicher Hash stehen untereinander, sonst ist die abweichende
		// Stelle im Vergleich nicht zu finden.
		out += fmt.Sprintf(
			"\nBuchung %s (Geschäftsjahr %d, ID %d)\n  Grund:       %s\n  Erwartet:    %s\n  Tatsächlich: %s\n  %s\n",
			br.EntryNumber, br.FiscalYear, br.EntryID, br.Reason, br.ExpectedHash, br.ActualHash, br.Message)
	}
	return out
}

func formatFileIssues(issues []domain.FileCheckIssue) string {
	if len(issues) == 0 {
		return "Keine Beanstandungen.\n"
	}
	out := ""
	for _, i := range issues {
		out += fmt.Sprintf("\n%s %s (%s): %s\n", i.ReceiptNumber, i.FileName, i.Reason, i.Message)
	}
	return out
}

// processDocFileName ist der Name, unter dem die Verfahrensdokumentation im
// Datenordner des Mandanten liegt.
//
// Nur dort und nicht unter docs/: in der gepackten Anwendung gibt es keinen
// Projektordner, und der Pfad relativ zum Arbeitsverzeichnis griff allein im
// Entwicklungs-Checkout — wo er dann eine Projektdatei in ein Prüferpaket
// kopierte, die den Betrieb des Mandanten gar nicht beschreibt.
//
// Die Spezifikation nennt docs/verfahrensdokumentation.md. Die Abweichung ist
// gewollt und gehört in die Planung von Welle 6: die Verfahrensdokumentation
// muss dort in den Datenordner des Mandanten geschrieben (oder als eingebettete
// Vorlage ausgeliefert und beim Anlegen des Mandanten dorthin kopiert) werden,
// sonst liegt sie keinem Prüferpaket bei.
const processDocFileName = "verfahrensdokumentation.md"

// copyProcessDocumentation legt die Verfahrensdokumentation bei, sofern sie
// vorliegt.
//
// Fehlt sie, wird das vermerkt und nicht verschwiegen: die GoBD verlangen sie
// (Rz. 151 ff.), und ein Prüferpaket, das ohne Hinweis ohne sie ankommt, sieht
// vollständig aus.
func (s *ExportService) copyProcessDocumentation(b *export.Builder) {
	if s.dataDir != "" {
		path := filepath.Join(s.dataDir, processDocFileName)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			if _, err := b.CopyFile(processDocFileName, path); err == nil {
				return
			}
		}
	}
	b.Note("Die Verfahrensdokumentation liegt nicht vor und konnte deshalb nicht beigelegt werden (GoBD Rz. 151 ff.). Erwartet wird sie als %s im Datenordner.", processDocFileName)
}

// companyLocation macht aus den Unternehmensdaten den Sitz, wie ihn die
// Beschreibungsdatei im Element Location erwartet.
func companyLocation(cfg *domain.CompanySettings) string {
	parts := make([]string, 0, 3)
	for _, part := range []string{cfg.Street, cfg.ZipCity, cfg.Country} {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, strings.TrimSpace(part))
		}
	}
	return strings.Join(parts, ", ")
}

// logExport hält den Umfang der Überlassung im Änderungsprotokoll fest.
func (s *ExportService) logExport(ctx context.Context, r *export.Result) {
	if s.auditRepo == nil {
		return
	}
	rows := 0
	for _, t := range r.Tables {
		rows += t.Rows
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionExport, "DATA_EXPORT", fmt.Sprintf("GJ_%d", r.FiscalYear),
		fmt.Sprintf("%s nach %s: %d Tabellen mit %d Datensätzen, %d Belegdateien, %d Anlagendokumente (Buchfink %s)",
			exportKindLabel(r.Kind), r.Dir, len(r.Tables), rows, r.ReceiptFiles, r.DocumentFiles, r.ProgramVersion))
}

func exportKindLabel(kind export.Kind) string {
	switch kind {
	case export.KindArchive:
		return "Archivexport"
	case export.KindAuditPackage:
		return "Prüferpaket"
	default:
		return "Datenüberlassung Z3"
	}
}
