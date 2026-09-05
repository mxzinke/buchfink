package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/domain"
)

// SupplyEvidenceService führt den Belegnachweis der innergemeinschaftlichen
// Lieferung (§§ 17a bis 17c UStDV).
//
// Die Steuerbefreiung des § 4 Nr. 1 Buchst. b UStG hat zwei Seiten: die
// materiellen Voraussetzungen des § 6a UStG — auf ihnen sitzt die
// Bestätigungsabfrage der USt-IdNr. — und den Nachweis, dass der Gegenstand
// tatsächlich in das übrige Gemeinschaftsgebiet gelangt ist. Der Nachweis ist
// der, der in der Praxis fehlt: die Rechnung ist an dem Tag geschrieben, an dem
// geliefert wird, und der Frachtbrief kommt eine Woche später. Wer ihn dann
// nirgends ablegen kann, hat ihn zur Prüfung nicht.
type SupplyEvidenceService struct {
	repo        domain.SupplyEvidenceRepository
	invoiceRepo domain.InvoiceRepository
	auditRepo   domain.AuditRepository
	// receipts ist der Belegspeicher. Ohne ihn nimmt der Nachweis nur einen
	// bereits abgelegten Beleg entgegen; mit ihm auch die Datei selbst.
	receipts   evidenceFiler
	fiscalYear int
}

// evidenceFiler ist der Ausschnitt der Belegverwaltung, den der Belegnachweis
// braucht: eine Datei ablegen und daraus einen Beleg machen.
type evidenceFiler interface {
	File(ctx context.Context, req FileReceiptRequest) (*domain.Receipt, error)
}

// SetReceiptService koppelt den Belegspeicher an den Nachweis.
//
// Der Frachtbrief kommt eine Woche nach der Rechnung, als PDF im Anhang einer
// Mail. Wer ihn nirgends ablegen kann, hat ihn zur Prüfung nicht — deshalb nimmt
// der Nachweis die Datei entgegen und legt sie im Belegspeicher ab, statt auf
// einen Beleg zu warten, den jemand vorher von Hand anlegt.
func (s *SupplyEvidenceService) SetReceiptService(f evidenceFiler) { s.receipts = f }

// NewSupplyEvidenceService wires den Belegnachweis.
func NewSupplyEvidenceService(
	repo domain.SupplyEvidenceRepository,
	invoiceRepo domain.InvoiceRepository,
	auditRepo domain.AuditRepository,
	fiscalYear int,
) *SupplyEvidenceService {
	return &SupplyEvidenceService{
		repo: repo, invoiceRepo: invoiceRepo, auditRepo: auditRepo, fiscalYear: fiscalYear,
	}
}

// SetFiscalYear schaltet das aktive Geschäftsjahr um.
func (s *SupplyEvidenceService) SetFiscalYear(year int) { s.fiscalYear = year }

// SupplyEvidenceRequest legt einen Nachweisbeleg ab.
type SupplyEvidenceRequest struct {
	InvoiceID   uint   `json:"invoiceId"`
	Kind        string `json:"kind"`
	Issuer      string `json:"issuer"`
	Independent bool   `json:"independent"`
	Date        string `json:"date"`
	// ReceiptID verweist auf den Beleg, unter dem die Datei abgelegt ist.
	ReceiptID uint   `json:"receiptId,omitempty"`
	Note      string `json:"note,omitempty"`
	// Transport sagt, wer befördert hat, und entscheidet damit, ob die
	// Gelangensbestätigung zusätzlich nötig ist (Art. 45a Abs. 1 Buchst. b
	// MwStVO).
	//
	// Er wird an der Rechnung festgehalten und nicht nur mitgegeben: die Angabe
	// beschreibt die Lieferung und nicht den einzelnen Beleg. Bliebe sie
	// flüchtig, bewertete der Jahresbericht jeden Abholfall als Regelfall — und
	// meldete eine Lieferung ohne Gelangensbestätigung als nachgewiesen. Leer
	// heißt: keine Änderung an dem, was an der Rechnung steht.
	Transport string `json:"transport,omitempty"`

	// FilePath ist die Datei des Nachweises auf der Platte. Ist sie angegeben,
	// legt Buchfink sie im Belegspeicher ab und verknüpft den entstandenen Beleg
	// — ReceiptID braucht dann niemand vorher zu besorgen.
	FilePath string `json:"filePath,omitempty"`
}

// SupplyEvidenceView ist der Nachweisstand einer Rechnung.
type SupplyEvidenceView struct {
	InvoiceID     uint                          `json:"invoiceId"`
	InvoiceNumber string                        `json:"invoiceNumber"`
	Date          string                        `json:"date"`
	ContactName   string                        `json:"contactName"`
	Transport     accounting.TransportKind      `json:"transport"`
	Items         []domain.SupplyEvidence       `json:"items"`
	Status        accounting.EvidenceStatus     `json:"status"`
	Kinds         []accounting.EvidenceKindInfo `json:"kinds"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (v *SupplyEvidenceView) EnsureLists() {
	if v.Items == nil {
		v.Items = make([]domain.SupplyEvidence, 0)
	}
	if v.Kinds == nil {
		v.Kinds = make([]accounting.EvidenceKindInfo, 0)
	}
	if v.Status.Missing == nil {
		v.Status.Missing = make([]string, 0)
	}
}

// Add legt einen Nachweisbeleg ab.
func (s *SupplyEvidenceService) Add(ctx context.Context, req SupplyEvidenceRequest) (*SupplyEvidenceView, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("der Belegnachweis ist nicht eingerichtet")
	}
	if _, ok := accounting.EvidenceKindGroup(accounting.EvidenceKind(req.Kind)); !ok {
		return nil, fmt.Errorf(
			"%q ist keine bekannte Belegart. Möglich sind die Arten des Art. 45a MwStVO und die "+
				"Bausteine des § 17b UStDV", req.Kind)
	}
	invoice, err := s.invoiceRepo.FindByID(ctx, req.InvoiceID)
	if err != nil {
		return nil, fmt.Errorf("die Rechnung %d wurde nicht gefunden: %w", req.InvoiceID, err)
	}
	if err := s.saveTransport(ctx, invoice, req.Transport); err != nil {
		return nil, err
	}

	receiptID := req.ReceiptID
	if strings.TrimSpace(req.FilePath) != "" {
		filed, err := s.fileEvidence(ctx, invoice, req)
		if err != nil {
			return nil, err
		}
		receiptID = filed.ID
	}

	evidence := &domain.SupplyEvidence{
		InvoiceID:   invoice.ID,
		Kind:        req.Kind,
		Issuer:      strings.TrimSpace(req.Issuer),
		Independent: req.Independent,
		Date:        req.Date,
		Note:        req.Note,
	}
	if receiptID != 0 {
		id := receiptID
		evidence.ReceiptID = &id
	}
	if err := evidence.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, evidence); err != nil {
		return nil, fmt.Errorf("der Nachweisbeleg ließ sich nicht speichern: %w", err)
	}
	s.audit(ctx, invoice.ID, fmt.Sprintf(
		"Nachweisbeleg zu %s abgelegt: %s von %s vom %s",
		invoice.InvoiceNumber, accounting.EvidenceKindLabel(accounting.EvidenceKind(req.Kind)),
		evidence.Issuer, evidence.Date))
	return s.View(ctx, invoice.ID, accounting.TransportKind(req.Transport))
}

// fileEvidence legt die Nachweisdatei im Belegspeicher ab.
//
// Als eigener Beleg der Art „Sonstiges" und nicht als weitere Datei an der
// Ausgangsrechnung: die Rechnung ist mit dem Ausstellen versiegelt, und was
// danach kommt — ein Frachtbrief, eine Gelangensbestätigung — ist ein eigenes
// Dokument zu demselben Geschäftsvorfall. Es wird deshalb auch nicht gebucht.
func (s *SupplyEvidenceService) fileEvidence(
	ctx context.Context, invoice *domain.Invoice, req SupplyEvidenceRequest,
) (*domain.Receipt, error) {
	if s.receipts == nil {
		return nil, fmt.Errorf(
			"der Belegspeicher ist nicht eingerichtet; lege die Datei zuerst als Beleg ab und gib " +
				"seine Nummer an")
	}
	received := req.Date
	if received == "" {
		received = invoice.Date
	}
	receipt, err := s.receipts.File(ctx, FileReceiptRequest{
		Direction:  domain.DirectionIncoming,
		FiscalYear: invoice.FiscalYear,
		Kind:       domain.ReceiptKindOther,
		ReceivedAt: received,
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal,
			Path: strings.TrimSpace(req.FilePath),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf(
			"die Nachweisdatei ließ sich nicht im Belegspeicher ablegen: %w", err)
	}
	return receipt, nil
}

// saveTransport hält die Beförderungsart an der Rechnung fest.
func (s *SupplyEvidenceService) saveTransport(
	ctx context.Context, invoice *domain.Invoice, transport string,
) error {
	kind := strings.TrimSpace(transport)
	if kind == "" || kind == invoice.TransportKind {
		return nil
	}
	if kind != string(accounting.TransportBySupplier) && kind != string(accounting.TransportByCustomer) {
		return fmt.Errorf(
			"%q ist keine bekannte Beförderungsart. Möglich sind „supplier\" (Beförderung durch den "+
				"Lieferer) und „customer\" (Abholfall)", kind)
	}
	if err := s.invoiceRepo.UpdateTransportKind(ctx, invoice.ID, kind); err != nil {
		return fmt.Errorf("die Beförderungsart ließ sich nicht speichern: %w", err)
	}
	invoice.TransportKind = kind
	return nil
}

// SetTransport hält die Beförderungsart an der Rechnung fest.
//
// Sie musste bisher an einem Nachweisbeleg mitreisen: die Auswahl in der Maske
// änderte nur die Bewertung der Ansicht, und wer sie umstellte und die Ansicht
// verließ, fand nach dem Neuladen wieder den alten Wert. Ein Feld, das wie eine
// gespeicherte Einstellung aussieht und keine ist, ist schlimmer als keines —
// beim Abholfall hängt die Gelangensbestätigung daran.
func (s *SupplyEvidenceService) SetTransport(
	ctx context.Context, invoiceID uint, transport string,
) (*SupplyEvidenceView, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("der Belegnachweis ist nicht eingerichtet")
	}
	invoice, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("die Rechnung %d wurde nicht gefunden: %w", invoiceID, err)
	}
	if err := s.saveTransport(ctx, invoice, transport); err != nil {
		return nil, err
	}
	s.audit(ctx, invoiceID, fmt.Sprintf("Beförderungsart auf %q gesetzt", invoice.TransportKind))
	return s.View(ctx, invoiceID, accounting.TransportKind(invoice.TransportKind))
}

// Remove nimmt einen Nachweisbeleg zurück.
func (s *SupplyEvidenceService) Remove(ctx context.Context, invoiceID, evidenceID uint, transport string) (*SupplyEvidenceView, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("der Belegnachweis ist nicht eingerichtet")
	}
	// Gelöscht wird nur, was zu dieser Rechnung gehört. Die Maske schickt zwei
	// Kennungen, und eine falsche Kombination — ein stehengebliebenes Formular,
	// eine zweite offene Ansicht — nähme sonst den Nachweis einer anderen
	// Rechnung mit, ohne dass es jemand merkte.
	items, err := s.repo.FindByInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	belongs := false
	for i := range items {
		if items[i].ID == evidenceID {
			belongs = true
			break
		}
	}
	if !belongs {
		return nil, fmt.Errorf(
			"der Nachweisbeleg %d gehört nicht zur Rechnung %d", evidenceID, invoiceID)
	}
	if err := s.repo.Delete(ctx, evidenceID); err != nil {
		return nil, err
	}
	s.audit(ctx, invoiceID, fmt.Sprintf("Nachweisbeleg %d entfernt", evidenceID))
	return s.View(ctx, invoiceID, accounting.TransportKind(transport))
}

// View liefert den Nachweisstand einer Rechnung.
func (s *SupplyEvidenceService) View(
	ctx context.Context, invoiceID uint, transport accounting.TransportKind,
) (*SupplyEvidenceView, error) {
	invoice, err := s.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("die Rechnung %d wurde nicht gefunden: %w", invoiceID, err)
	}
	items := make([]domain.SupplyEvidence, 0)
	if s.repo != nil {
		items, err = s.repo.FindByInvoice(ctx, invoiceID)
		if err != nil {
			return nil, err
		}
	}
	// Ohne ausdrückliche Angabe gilt, was an der Rechnung steht; steht dort
	// nichts, der Regelfall der Beförderung durch den Lieferer.
	if transport == "" {
		transport = accounting.TransportKind(invoice.TransportKind)
	}
	if transport == "" {
		transport = accounting.TransportBySupplier
	}
	out := &SupplyEvidenceView{
		InvoiceID:     invoice.ID,
		InvoiceNumber: invoice.InvoiceNumber,
		Date:          invoice.Date,
		ContactName:   invoice.ContactName,
		Transport:     transport,
		Items:         items,
		Status:        accounting.AssessSupplyEvidence(transport, evidenceItems(items)),
		Kinds:         accounting.EvidenceKinds(),
	}
	out.EnsureLists()
	return out, nil
}

// evidenceItems übersetzt die gespeicherten Belege in die Form, die die
// Bewertung liest.
func evidenceItems(items []domain.SupplyEvidence) []accounting.EvidenceItem {
	out := make([]accounting.EvidenceItem, 0, len(items))
	for _, it := range items {
		out = append(out, accounting.EvidenceItem{
			Kind:        accounting.EvidenceKind(it.Kind),
			Issuer:      it.Issuer,
			Independent: it.Independent,
		})
	}
	return out
}

// SupplyEvidenceReportRow ist eine steuerfreie Lieferung im Bericht.
type SupplyEvidenceReportRow struct {
	InvoiceID     uint                      `json:"invoiceId"`
	InvoiceNumber string                    `json:"invoiceNumber"`
	Date          string                    `json:"date"`
	ContactName   string                    `json:"contactName"`
	NetAmount     domain.Cents              `json:"netAmount"`
	EvidenceCount int                       `json:"evidenceCount"`
	Transport     accounting.TransportKind  `json:"transport"`
	Status        accounting.EvidenceStatus `json:"status"`
}

// SupplyEvidenceReport ist der Bericht eines Geschäftsjahres.
type SupplyEvidenceReport struct {
	FiscalYear int                       `json:"fiscalYear"`
	Rows       []SupplyEvidenceReportRow `json:"rows"`
	// Incomplete zählt die Lieferungen ohne vollständigen Nachweis.
	Incomplete int    `json:"incomplete"`
	Note       string `json:"note"`
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
func (r *SupplyEvidenceReport) EnsureLists() {
	if r.Rows == nil {
		r.Rows = make([]SupplyEvidenceReportRow, 0)
	}
}

// Report führt die steuerfreien innergemeinschaftlichen Lieferungen eines Jahres
// mit ihrem Nachweisstand auf.
//
// Der Fristhinweis gehört dazu: der Nachweis ist bis zur Abgabe der
// Voranmeldung des Zeitraums zu führen, in dem die Lieferung ausgeführt wurde.
// Wer ihn später beibringt, kann die Befreiung verlieren — und dann steht eine
// Rechnung ohne Steuerausweis da, deren Steuer trotzdem geschuldet wird.
func (s *SupplyEvidenceService) Report(ctx context.Context, year int) (*SupplyEvidenceReport, error) {
	if year == 0 {
		year = s.fiscalYear
	}
	out := &SupplyEvidenceReport{FiscalYear: year}
	out.EnsureLists()

	invoices, err := s.invoiceRepo.FindAll(ctx, year)
	if err != nil {
		return nil, err
	}
	relevant := make([]domain.Invoice, 0, 4)
	ids := make([]uint, 0, 4)
	for i := range invoices {
		if invoices[i].TaxTreatment != domain.TaxTreatmentIntraCommunitySupply {
			continue
		}
		if invoices[i].Status == domain.InvoiceStatusCancelled {
			continue
		}
		relevant = append(relevant, invoices[i])
		ids = append(ids, invoices[i].ID)
	}
	if len(relevant) == 0 {
		out.Note = fmt.Sprintf(
			"Im Geschäftsjahr %d ist keine steuerfreie innergemeinschaftliche Lieferung ausgestellt "+
				"worden.", year)
		return out, nil
	}

	byInvoice := map[uint][]domain.SupplyEvidence{}
	if s.repo != nil {
		byInvoice, err = s.repo.FindByInvoices(ctx, ids)
		if err != nil {
			return nil, err
		}
	}

	for i := range relevant {
		inv := &relevant[i]
		items := byInvoice[inv.ID]
		// Die Beförderungsart steht an der Rechnung. Sie entscheidet darüber, ob
		// zur Vermutung des § 17a UStDV die Gelangensbestätigung hinzukommen muss
		// — und ein Abholfall, den der Bericht als Regelfall bewertete, erschiene
		// als nachgewiesen, obwohl der Nachweis fehlt. Ist nichts vermerkt, gilt
		// der Regelfall: die strengere Prüfung ohne Anhaltspunkt anzuwenden hieße,
		// jede Lieferung als unvollständig zu melden.
		transport := accounting.TransportKind(inv.TransportKind)
		if transport == "" {
			transport = accounting.TransportBySupplier
		}
		row := SupplyEvidenceReportRow{
			InvoiceID: inv.ID, InvoiceNumber: inv.InvoiceNumber, Date: inv.Date,
			ContactName: inv.ContactName, NetAmount: inv.NetAmount,
			EvidenceCount: len(items),
			Transport:     transport,
			Status:        accounting.AssessSupplyEvidence(transport, evidenceItems(items)),
		}
		if !row.Status.Fulfilled {
			out.Incomplete++
		}
		out.Rows = append(out.Rows, row)
	}
	sort.SliceStable(out.Rows, func(i, j int) bool {
		if out.Rows[i].Status.Fulfilled != out.Rows[j].Status.Fulfilled {
			return !out.Rows[i].Status.Fulfilled
		}
		return out.Rows[i].Date < out.Rows[j].Date
	})

	out.Note = fmt.Sprintf(
		"%d von %d steuerfreien innergemeinschaftlichen Lieferungen des Jahres %d haben noch keinen "+
			"vollständigen Belegnachweis. Er ist bis zur Abgabe der Voranmeldung des Zeitraums zu "+
			"führen, in dem die Lieferung ausgeführt wurde (§ 17a Abs. 1 UStDV); fehlt er, droht die "+
			"Steuerpflicht der Lieferung.",
		out.Incomplete, len(out.Rows), year)
	return out, nil
}

func (s *SupplyEvidenceService) audit(ctx context.Context, invoiceID uint, details string) {
	if s.auditRepo == nil {
		return
	}
	_ = s.auditRepo.Log(ctx, domain.AuditActionUpdate, "SUPPLY_EVIDENCE",
		fmt.Sprintf("%d", invoiceID), details)
}
