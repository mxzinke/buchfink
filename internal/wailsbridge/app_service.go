package wailsbridge

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/core/policy"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/einvoice"
	"github.com/buchfink/buchfink/internal/einvoice/xrechnung"
	"github.com/buchfink/buchfink/internal/einvoice/zugferd"
	"github.com/buchfink/buchfink/internal/mcpserver"
	"github.com/buchfink/buchfink/internal/repository"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// BuchfinkBridge ist die Oberfläche des Kerns für das Wails-Frontend. Mandant,
// Geschäftsjahr und Dienste leben im Kern (internal/core); die Bridge reicht
// durch und setzt den Erfassungsweg `ui` (session.go).
type BuchfinkBridge struct {
	host *core.Host
	// mcp ist der Server für verbundene Apps (mcp_service.go); nil, wenn er
	// sich nicht einrichten ließ, dann steht der Grund in mcpErr.
	mcp    *mcpserver.Server
	mcpErr string
}

func NewBuchfinkBridge() (*BuchfinkBridge, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	baseDir := filepath.Join(homeDir, ".buchfink")
	changes := newChangeBatcher(changeInterval, emitToWindow)
	host, err := core.NewHost(context.Background(), core.HostOptions{
		Config:       repository.NewAppConfigRepository(baseDir),
		BaseDir:      baseDir,
		WarmRenderer: true,
		Jobs:         true,
		OnChange:     changes.add,
	})
	if err != nil {
		return nil, err
	}
	b := &BuchfinkBridge{host: host}
	newMCPServer(b)
	return b, nil
}

// -------------------------------------------------------------
// MULTI-TENANT MANAGEMENT (MANDANTENVERWALTUNG)
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetTenants() []domain.TenantConfig {
	return b.host.Config().Tenants
}

func (b *BuchfinkBridge) GetActiveTenant() (*domain.TenantConfig, error) {
	return b.host.ActiveTenant(), nil
}

func (b *BuchfinkBridge) SwitchTenant(tenantID string) error {
	_, err := b.host.Switch(uiContext(), tenantID)
	return err
}

// createTenant legt einen Mandanten an. Die Oberfläche tut das nur über die
// Einrichtung (SetupApplication); gebunden ist die Methode deshalb nicht.
func (b *BuchfinkBridge) createTenant(
	name string,
	dataDir string,
	settings domain.CompanySettings,
) (*domain.TenantConfig, error) {
	return b.host.CreateTenant(uiContext(), name, dataDir, settings)
}

// DeleteTenant entfernt einen Mandanten aus der Konfiguration. Die Daten auf
// der Platte bleiben liegen, samt einem laufenden Prüfermodus in ihrer
// Datenbank.
func (b *BuchfinkBridge) DeleteTenant(tenantID string) error {
	return b.host.RemoveTenant(uiContext(), tenantID)
}

// -------------------------------------------------------------
// SETUP & ONBOARDING ASSISTANT (FIRST LAUNCH)
// -------------------------------------------------------------

// GetAppConfig liefert den Zustand, aus dem die Oberfläche liest. Die
// Spiegelfelder des aktiven Mandanten sind auf den heutigen Tag gerechnet: ein
// abgelaufener Prüfermodus gilt nicht mehr, auch wenn seit dem Ablauf nichts
// gespeichert wurde.
func (b *BuchfinkBridge) GetAppConfig() domain.AppConfig {
	return b.host.Config()
}

// SetupApplication richtet einen Mandanten ein: Datenordner (Ablage),
// Eingang und Unternehmensdaten. Ein leerer Eingang ist <Datenordner>/Eingang;
// ein gewählter darf keinen Datenordner und keine Sicherung berühren. Der
// Eingang wird angelegt.
func (b *BuchfinkBridge) SetupApplication(
	dataDir string,
	inboxDir string,
	settings domain.CompanySettings,
) error {
	name := settings.CompanyName
	if name == "" {
		name = "Hauptmandant"
	}
	inboxDir = domain.CleanFolder(inboxDir)
	if inboxDir != "" {
		dir, err := core.CheckTenantDirectory(dataDir)
		if err != nil {
			return err
		}
		if err := policy.CheckFolders(inboxDir, nil, dir, policy.ProtectedDirs(b.host.Config(), dir)); err != nil {
			return err
		}
	}
	_, err := b.host.CreateTenant(uiContext(), name, dataDir, settings, core.WithInbox(inboxDir))
	return err
}

// LoadExistingDatabase öffnet eine vorhandene Buchfink-Datei als Mandanten.
// Protokolliert wird sie als „geöffnet" und nicht als „übernommen": es kommen
// keine fremden Daten herein.
func (b *BuchfinkBridge) LoadExistingDatabase(dbFilePath string) error {
	_, err := b.host.RegisterTenant(uiContext(), dbFilePath, "", domain.MigrationKindOpen)
	return err
}

// SelectDirectoryDialog opens a native OS folder picker.
func (b *BuchfinkBridge) SelectDirectoryDialog(title string) (string, error) {
	if title == "" {
		title = "Buchfink Datenordner auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle(title).
		PromptForSingleSelection()
}

// SelectDatabaseFileDialog opens a native OS file picker for SQLite database files.
func (b *BuchfinkBridge) SelectDatabaseFileDialog(title string) (string, error) {
	if title == "" {
		title = "Buchfink Buchhaltungsdatei auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return "", nil
	}
	return app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Buchfink Buchhaltungsdateien (*.sqlite, *.db)", "*.sqlite;*.db").
		SetTitle(title).
		PromptForSingleSelection()
}

// -------------------------------------------------------------
// DYNAMIC FISCAL YEARS & FILTERING
// -------------------------------------------------------------

// GetAvailableFiscalYears lists the fiscal years the user can switch to: the
// union of the recorded fiscal years and those that only appear in the journal.
//
// Beide Quellen, weil beide unvollständig sind. Ein neu angelegtes Geschäftsjahr
// hat noch keine Buchung, und eine gewachsene Datenbank hat Buchungen in Jahren,
// zu denen die Entität erst nachträglich entsteht.
func (b *BuchfinkBridge) GetAvailableFiscalYears() []int {
	years, err := read(b, func(ctx context.Context, s *core.Services) ([]int, error) {
		years := s.Accounting.GetAvailableFiscalYears(ctx)
		recorded, err := s.Closing.FiscalYears(ctx)
		if err != nil {
			return years, nil
		}
		known := make(map[int]bool, len(years))
		for _, y := range years {
			known[y] = true
		}
		for _, fy := range recorded {
			if fy.Year > 0 && !known[fy.Year] {
				known[fy.Year] = true
				years = append(years, fy.Year)
			}
		}
		// Das angesehene Jahr steht immer zur Wahl, auch ohne Buchung.
		if view := b.viewYear(); !known[view] {
			years = append(years, view)
		}
		sort.Ints(years)
		return years, nil
	})
	if err != nil {
		return repository.DiscoverAvailableFiscalYears(b.host.Config().DataDir)
	}
	if years == nil {
		return []int{}
	}
	return years
}

// GetFiscalYear ist das Geschäftsjahr der Ansicht.
func (b *BuchfinkBridge) GetFiscalYear() int {
	return b.viewYear()
}

// SetFiscalYear stellt die Ansicht auf ein Geschäftsjahr um und merkt es sich
// für den nächsten Start. Das Jahr ist ein Filter der Oberfläche; die Dienste
// bekommen es mit jedem Aufruf.
func (b *BuchfinkBridge) SetFiscalYear(year int) error {
	if year < 1900 || year > 9999 {
		return fmt.Errorf("%d ist kein Geschäftsjahr", year)
	}
	if _, release, err := b.host.Acquire(context.Background()); err != nil {
		return err
	} else {
		release()
	}
	if err := b.host.SetLastFiscalYear(year); err != nil {
		return fmt.Errorf("das Geschäftsjahr ließ sich nicht merken: %w", err)
	}
	return nil
}

// -------------------------------------------------------------
// COMPANY SETTINGS & STAMMDATEN
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetCompanySettings() (*domain.CompanySettings, error) {
	return readOr(b, func() (*domain.CompanySettings, error) {
		return &domain.CompanySettings{
			FiscalYear: time.Now().Year(), FiscalYearStartMonth: 1, Currency: "EUR",
			SKR: "SKR04", TaxationType: "SOLL", CountryCode: domain.DefaultCountryCode,
			Shareholders: []domain.CompanyShareholder{},
		}, nil
	}, func(ctx context.Context, s *core.Services) (*domain.CompanySettings, error) {
		return s.Settings.GetCompanySettings(ctx)
	})
}

// UpdateCompanySettings ändert die Stammdaten des Unternehmens; der Firmenname
// geht mit in die Mandantenliste. Nur in der App.
func (b *BuchfinkBridge) UpdateCompanySettings(settings domain.CompanySettings) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		if err := s.Settings.UpdateCompanySettings(ctx, &settings); err != nil {
			return err
		}
		if settings.CompanyName == "" {
			return nil
		}
		return b.host.RenameTenant(s.Tenant.ID, settings.CompanyName)
	}, core.AppOnly())
}

// -------------------------------------------------------------
// ACCOUNTING & JOURNAL
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetAccounts() ([]domain.Account, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Account, error) {
		return s.Accounting.GetAccounts(ctx, b.viewYear())
	})
}

func (b *BuchfinkBridge) GetAccountLedger(accountNumber string) (*domain.AccountLedger, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.AccountLedger, error) {
		return s.Accounting.GetAccountLedger(ctx, b.viewYear(), accountNumber)
	})
}

func (b *BuchfinkBridge) GetSuSaOverview() (*domain.SuSaOverview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*domain.SuSaOverview, error) {
		return s.Accounting.GetSuSaOverview(ctx, b.viewYear())
	})
}

// GetPostingGroups returns the fachliche Gruppen the user picks from instead of
// account numbers. "incoming" for Eingangsbelege, "outgoing" for Erlöse, empty
// for both.
func (b *BuchfinkBridge) GetPostingGroups(direction string) []accounting.PostingGroup {
	return accounting.PostingGroups(domain.Direction(direction))
}

// GetTaxTreatments returns the Steuerfälle valid for a direction, with the hints
// the UI shows next to them.
func (b *BuchfinkBridge) GetTaxTreatments(direction string) []domain.TaxTreatmentInfo {
	return domain.TaxTreatments(domain.Direction(direction))
}

// GetPaymentAccounts lists the accounts that can settle a document immediately.
func (b *BuchfinkBridge) GetPaymentAccounts() ([]domain.Account, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Account, error) {
		all, err := s.Accounting.GetAccounts(ctx, b.viewYear())
		if err != nil {
			return nil, err
		}
		liquid := map[string]bool{}
		for _, a := range domain.LiquidAccounts() {
			liquid[a] = true
		}
		result := make([]domain.Account, 0, len(all))
		for _, a := range all {
			if liquid[a.Number] {
				result = append(result, a)
			}
		}
		return result, nil
	})
}

func (b *BuchfinkBridge) GetJournalEntries() ([]domain.JournalEntry, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.JournalEntry, error) {
		return s.Accounting.GetEntries(ctx, b.viewYear())
	})
}

func (b *BuchfinkBridge) GetAllJournalEntries() ([]domain.JournalEntry, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.JournalEntry, error) {
		return s.Accounting.GetAllEntries(ctx)
	})
}

// PostIncomingReceipt books an Eingangsbeleg from the fachliche Gruppe, the
// Steuerfall and the payment state.
func (b *BuchfinkBridge) PostIncomingReceipt(req service.ReceiptRequest) (*domain.JournalEntry, error) {
	return prepared(b, func(ctx context.Context, t *core.Tenant) error {
		if err := t.PrefetchRate(ctx, req.Currency, req.DocumentDate); err != nil {
			return fmt.Errorf("für den Beleg in %s vom %s: %w", req.Currency, req.DocumentDate, err)
		}
		return nil
	}, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Posting.PostIncomingReceipt(ctx, req)
	})
}

func (b *BuchfinkBridge) VerifyIntegrity() (domain.IntegrityCheckResult, error) {
	return readOr(b, func() (domain.IntegrityCheckResult, error) {
		// Auch der Leerlauf gibt belegte Listen zurück: die Ansicht liest
		// `breaks.length`, und `null.length` nähme den ganzen Baum mit.
		empty := domain.IntegrityCheckResult{IsValid: true, Message: "Bereit"}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (domain.IntegrityCheckResult, error) {
		return s.Journal.VerifyIntegrity(ctx)
	})
}

// -------------------------------------------------------------
// BANK IMPORT & RECONCILIATION
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetBankTransactions() ([]domain.BankTransaction, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.BankTransaction, error) {
		return s.Bank.GetTransactions(ctx, b.viewYear())
	})
}

// BookBankTransactionDirect books a statement line that has no document behind
// it (fees, interest, transfers between own accounts).
func (b *BuchfinkBridge) BookBankTransactionDirect(bankTxID uint, counterAccount, description string) (*domain.JournalEntry, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Bank.BookDirect(ctx, bankTxID, counterAccount, description)
	})
}

// -------------------------------------------------------------
// CONTACTS & INVOICES
// -------------------------------------------------------------

func (b *BuchfinkBridge) GetContacts() ([]domain.Contact, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Contact, error) {
		return s.Contacts.GetContacts(ctx, b.viewYear())
	})
}

func (b *BuchfinkBridge) SaveContact(c domain.Contact) (*domain.Contact, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Contact, error) {
		if err := s.Contacts.SaveContact(ctx, &c); err != nil {
			return nil, err
		}
		return &c, nil
	})
}

func (b *BuchfinkBridge) GetInvoices() ([]domain.Invoice, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Invoice, error) {
		return s.Invoices.GetInvoices(ctx, b.viewYear())
	})
}

// IssueInvoice assigns the consecutive invoice number and books the receivable
// in one step — an invoice never exists on paper without being in the journal.
func (b *BuchfinkBridge) IssueInvoice(inv domain.Invoice) (*domain.Invoice, error) {
	return prepared(b, func(ctx context.Context, t *core.Tenant) error {
		return t.PrepareInvoice(ctx, &inv)
	}, func(ctx context.Context, s *core.Services) (*domain.Invoice, error) {
		if err := s.Invoices.Issue(ctx, &inv); err != nil {
			return nil, err
		}
		return &inv, nil
	})
}

// -------------------------------------------------------------
// BELEGE
// -------------------------------------------------------------

// ReceiptFileInput is one file on its way into a Beleg.
//
// Files travel as paths, not as content: the native dialog and drag & drop both
// hand over paths, and a multi-megabyte scan has no business crossing the IPC
// boundary base64-encoded.
type ReceiptFileInput struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

// SelectReceiptFilesDialog opens the native picker for Belegdateien.
func (b *BuchfinkBridge) SelectReceiptFilesDialog(title string) ([]string, error) {
	if title == "" {
		title = "Belegdateien auswählen"
	}
	app := application.Get()
	if app == nil || app.Dialog == nil {
		return []string{}, nil
	}
	// Der Abbruch des Dialogs liefert nil; als `null` in der Oberfläche wäre
	// daraus ein TypeError beim Lesen der Länge geworden.
	return emptyList(app.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Belege (PDF, Bild, XML)", "*.pdf;*.png;*.jpg;*.jpeg;*.tif;*.tiff;*.webp;*.xml").
		SetTitle(title).
		PromptForMultipleSelection())
}

// FileIncomingReceipt files an incoming Beleg away without booking it. The two
// are separate steps: an XRechnung has to be storable the moment it arrives, and
// only becomes bookable once a rendering exists. Dateien aus dem Eingang gehen
// danach nach Übernommen/<JJJJ-MM>.
func (b *BuchfinkBridge) FileIncomingReceipt(receivedAt, receivedVia string, files []ReceiptFileInput) (*domain.Receipt, error) {
	receipt, err := write(b, func(ctx context.Context, s *core.Services) (*domain.Receipt, error) {
		return s.Receipts.File(ctx, service.FileReceiptRequest{
			Direction:   domain.DirectionIncoming,
			ReceivedAt:  receivedAt,
			ReceivedVia: receivedVia,
			Files:       toNewFiles(files),
		})
	})
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	b.settleInbox(paths)
	return receipt, nil
}

// GetReceipts lists the Belege of the active fiscal year. An empty status
// returns all of them.
func (b *BuchfinkBridge) GetReceipts(status string) ([]domain.Receipt, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Receipt, error) {
		return s.Receipts.List(ctx, b.viewYear(), domain.ReceiptStatus(status))
	})
}

// ReceiptPreview is what the frontend shows for a Beleg.
type ReceiptPreview struct {
	// DataURL carries the displayable file inline so the frontend can render it
	// without a second transport.
	DataURL  string `json:"dataUrl"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	// Intact is false when the file on disk no longer matches the checksum it
	// was filed under. It is shown rather than hidden.
	Intact bool `json:"intact"`
}

// maxPreviewBytes caps what is inlined into the IPC response. Larger files are
// reported rather than silently truncated.
const maxPreviewBytes = 32 << 20

// GetReceiptPreview returns the file the user is shown for a Beleg — the
// original for a PDF or an image, the generated rendering for an XRechnung.
//
// On a hybrid Beleg this is the image part, which is display only: booking reads
// the structured part. A difference between the two is potentially a second
// invoice with § 14c consequences and is shown, not judged.
func (b *BuchfinkBridge) GetReceiptPreview(receiptID uint) (*ReceiptPreview, error) {
	content, err := read(b, func(ctx context.Context, s *core.Services) (*service.FileContent, error) {
		return s.Receipts.DisplayContent(ctx, receiptID)
	})
	if err != nil {
		return nil, err
	}
	if len(content.Data) > maxPreviewBytes {
		return nil, fmt.Errorf("die Belegdatei %s ist zu groß für die Vorschau (%d MB)", content.FileName, len(content.Data)>>20)
	}
	return &ReceiptPreview{
		DataURL:  "data:" + content.MimeType + ";base64," + base64.StdEncoding.EncodeToString(content.Data),
		FileName: content.FileName,
		MimeType: content.MimeType,
		Intact:   content.Intact,
	}, nil
}

// GetInvoiceDocument returns the issued invoice as a data URL for display and
// download. It is the same hybrid PDF that was archived when the invoice was
// issued, not a fresh rendering — what the customer received is what is shown.
func (b *BuchfinkBridge) GetInvoiceDocument(invoiceID uint) (*ReceiptPreview, error) {
	content, err := read(b, func(ctx context.Context, s *core.Services) (*service.FileContent, error) {
		inv, err := s.Repos.Invoices.FindByID(ctx, invoiceID)
		if err != nil {
			return nil, err
		}
		if inv.ReceiptID == nil {
			return nil, fmt.Errorf("zu Rechnung %s liegt kein Belegdokument vor", inv.InvoiceNumber)
		}
		return s.Receipts.DisplayContent(ctx, *inv.ReceiptID)
	})
	if err != nil {
		return nil, err
	}
	if len(content.Data) > maxPreviewBytes {
		return nil, fmt.Errorf("das Rechnungsdokument ist zu groß für die Anzeige")
	}
	return &ReceiptPreview{
		DataURL:  "data:" + content.MimeType + ";base64," + base64.StdEncoding.EncodeToString(content.Data),
		FileName: content.FileName,
		MimeType: content.MimeType,
		Intact:   content.Intact,
	}, nil
}

// ProposeFromEInvoice turns the structured part of a Beleg into a prefilled
// booking. The fachliche Gruppe stays open — no invoice knows which expense
// account a supply belongs on.
func (b *BuchfinkBridge) ProposeFromEInvoice(receiptID uint) (*service.EInvoiceProposal, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.EInvoiceProposal, error) {
		return s.EInvoice.Propose(ctx, receiptID)
	})
}

// GetEInvoiceRules lists the rules Buchfink checks. It is deliberately exposed:
// "validated" without the list of what was checked tells a user nothing they
// can act on.
//
// The list covers EN 16931 completely and the German CIUS as far as its rules
// can be decided on the semantic model; xrechnung.UncheckedRules names the
// rest, with the reason for each.
func (b *BuchfinkBridge) GetEInvoiceRules() []string {
	rules := einvoice.RulesChecked()
	rules = append(rules, xrechnung.CheckedRules()...)
	rules = append(rules, zugferd.CheckedRules()...)
	sort.Strings(rules)
	return rules
}

// PreviewIncomingReceipt computes the booking without writing it, so the form can
// show the Buchungssatz instead of re-deriving it.
func (b *BuchfinkBridge) PreviewIncomingReceipt(req service.ReceiptRequest) (*service.PostingPreview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.PostingPreview, error) {
		return s.Posting.PreviewIncomingReceipt(ctx, req)
	})
}

// PreviewOutgoingInvoice computes what an invoice would book, without issuing it.
func (b *BuchfinkBridge) PreviewOutgoingInvoice(inv domain.Invoice) (*service.PostingPreview, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.PostingPreview, error) {
		return s.Invoices.Preview(ctx, &inv)
	})
}

func toNewFiles(files []ReceiptFileInput) []service.NewFile {
	out := make([]service.NewFile, 0, len(files))
	for _, f := range files {
		role := domain.ReceiptFileRole(f.Role)
		if role == "" {
			role = domain.ReceiptRoleOriginal
		}
		out = append(out, service.NewFile{
			Path: f.Path,
			Role: role,
			// A rendering is the only role the user can pick that is by
			// definition produced rather than received.
			Derived: role == domain.ReceiptRoleRendering,
		})
	}
	return out
}

// -------------------------------------------------------------
// OFFENE POSTEN & ZAHLUNGSZUORDNUNG
// -------------------------------------------------------------

// SettlePayment books a payment against one or more open items, including
// Skonto, bank fees and rounding differences.
func (b *BuchfinkBridge) SettlePayment(req service.PaymentRequest) (*domain.JournalEntry, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.JournalEntry, error) {
		return s.Payments.Settle(ctx, req)
	})
}

// -------------------------------------------------------------
// GRÜNDUNG (VORGESELLSCHAFT BIS EINTRAGUNG)
// -------------------------------------------------------------

// GetFoundationState returns everything the Gründungsansicht needs: die Regeln
// der Rechtsform, die erfasste Gründung, den Anmeldungsbefund, die Unterbilanz
// und die Fristen.
//
// Ein Aufruf, nicht fünf: die Teile hängen voneinander ab, und fünf Aufrufe
// könnten einen Stand zeigen, den es so nie gab.
func (b *BuchfinkBridge) GetFoundationState() (*service.FoundationState, error) {
	return read(b, func(ctx context.Context, s *core.Services) (*service.FoundationState, error) {
		return s.Foundation.GetState(ctx)
	})
}

// SaveFoundation records or updates the Gründung of the active tenant.
func (b *BuchfinkBridge) SaveFoundation(f domain.Foundation) (*domain.Foundation, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Foundation, error) {
		return s.Foundation.Save(ctx, &f)
	}, core.AppOnly())
}

// GetFoundationRules returns the Kapitalaufbringungsregeln of every legal form
// the Gründungsweg covers. Der Einrichtungsassistent verzweigt daran, statt
// Rechtsformnamen zu vergleichen.
func (b *BuchfinkBridge) GetFoundationRules() []accounting.FoundationRules {
	return accounting.FoundationLegalForms()
}

// GetCountries lists the countries selectable for the company address.
func (b *BuchfinkBridge) GetCountries() []domain.Country {
	return domain.Countries()
}

// RegisterCompany records the entry in the Handelsregister and ends the
// Vorgesellschaft. Ab hier steht die Unterbilanz auf diesen Tag fest.
func (b *BuchfinkBridge) RegisterCompany(date, court, number string) (*domain.Foundation, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*domain.Foundation, error) {
		return s.Foundation.Register(ctx, date, court, number)
	}, core.AppOnly())
}
