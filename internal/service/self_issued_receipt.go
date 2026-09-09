package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
)

// Der Eigenbeleg (BEL-01 K1 und K2, GOB-05 K1).
//
// „Keine Buchung ohne Beleg" ist keine Formel, sondern die Grundregel des
// § 146 Abs. 1 AO und der GoBD Rz. 61. Sie ist im Alltag genau dort schwer
// einzuhalten, wo es keinen Fremdbeleg gibt: das Trinkgeld, der verlorene
// Parkschein, die Privateinlage, die Entnahme aus der Kasse. Ohne einen Weg,
// den Vorgang selbst zu belegen, entsteht die Buchung trotzdem — nur eben ohne
// Beleg, und das ist der Mangel, den eine Betriebsprüfung zuerst findet.
//
// Buchfink erzeugt den Eigenbeleg deshalb selbst: aus Aussteller (dem eigenen
// Unternehmen), Datum, Betrag, Grund und der erfassenden Person entsteht ein
// PDF, das als Original im Belegspeicher liegt. Es ist kein Formular zum
// Ausdrucken und Unterschreiben, sondern der Beleg — mit dem Satz, dass kein
// Fremdbeleg vorliegt, weil genau das die Aussage ist, die ein Eigenbeleg
// trifft.

// SelfIssuedReceiptRequest ist die Eingabe für einen Eigenbeleg.
type SelfIssuedReceiptRequest struct {
	// DocumentDate ist der Tag des Vorgangs, nicht der Tag der Erfassung. Ohne
	// ihn lässt sich die zeitgerechte Erfassung nicht beurteilen.
	DocumentDate string `json:"documentDate"`
	// GrossAmount ist der Betrag des Vorgangs, brutto.
	GrossAmount domain.Cents `json:"grossAmount"`
	TaxAmount   domain.Cents `json:"taxAmount"`
	Currency    string       `json:"currency,omitempty"`
	// Reason ist der Grund: was belegt dieser Beleg, und warum gibt es keinen
	// fremden. Pflicht — er ist der ganze Inhalt des Dokuments.
	Reason string `json:"reason"`
	// Direction ist die Richtung des Vorgangs; leer heißt Eingang.
	Direction domain.Direction `json:"direction,omitempty"`
	// FiscalYear überschreibt das aktive Geschäftsjahr; 0 heißt: das aktive.
	FiscalYear int `json:"fiscalYear,omitempty"`
}

// selfIssuedNotice ist der Satz, der einen Eigenbeleg zum Eigenbeleg macht.
//
// Er steht als Konstante, weil er die rechtlich erhebliche Aussage des
// Dokuments ist und nicht seine Gestaltung: ein Eigenbeleg ohne ihn sähe aus
// wie eine Quittung, die jemand selbst geschrieben hat, ohne zu sagen, dass sie
// keine ist.
const selfIssuedNotice = "Zu diesem Geschäftsvorfall liegt kein Fremdbeleg vor. " +
	"Dieser Beleg wurde vom Unternehmen selbst ausgestellt (Eigenbeleg)."

// SetRenderer hängt den Satzweg an. Ohne ihn entsteht kein Eigenbeleg — und das
// wird gesagt statt eine leere Datei abzulegen.
func (s *ReceiptService) SetRenderer(r DocumentRenderer) { s.renderer = r }

// SetSettingsSource hängt die Unternehmensangaben an. Der Aussteller eines
// Eigenbelegs ist das eigene Unternehmen, und es steht dort.
func (s *ReceiptService) SetSettingsSource(repo domain.SettingsRepository) { s.settingsRepo = repo }

// CreateSelfIssued erzeugt einen Eigenbeleg mit PDF und Kopfdaten.
func (s *ReceiptService) CreateSelfIssued(
	ctx context.Context, req SelfIssuedReceiptRequest,
) (*domain.Receipt, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, fmt.Errorf(
			"ein Eigenbeleg braucht seinen Grund. Er ist der einzige Nachweis des Vorgangs: was " +
				"aufgewendet oder vereinnahmt wurde und warum es dazu keinen fremden Beleg gibt")
	}
	if len(req.DocumentDate) != 10 {
		return nil, fmt.Errorf("ein Eigenbeleg braucht das Datum des Vorgangs (JJJJ-MM-TT)")
	}
	if req.GrossAmount == 0 {
		return nil, fmt.Errorf("ein Eigenbeleg braucht einen Betrag")
	}
	if s.renderer == nil {
		return nil, fmt.Errorf(
			"der PDF-Satz ist nicht eingerichtet; ohne ihn entsteht kein Eigenbeleg")
	}

	issuer := s.issuerName(ctx)
	created := actor.Actor()
	number := s.nextSelfIssuedLabel(ctx)

	markdown := selfIssuedMarkdown(req, issuer, created, number, reason)
	// Der Ident macht zwei Läufe desselben Belegs byteweise vergleichbar; er
	// darf deshalb nicht die Uhrzeit enthalten.
	pdf, err := s.renderer.RenderDocumentPDF(ctx,
		procdoc.Typst(markdown, "Eigenbeleg", parseDayOrNow(req.DocumentDate)),
		"eigenbeleg-"+req.DocumentDate+"-"+created)
	if err != nil {
		return nil, fmt.Errorf("der Eigenbeleg konnte nicht gesetzt werden: %w", err)
	}

	direction := req.Direction
	if direction == "" {
		direction = domain.DirectionIncoming
	}
	currency := req.Currency
	if currency == "" {
		currency = "EUR"
	}

	return s.File(ctx, FileReceiptRequest{
		Direction:    direction,
		FiscalYear:   req.FiscalYear,
		Kind:         domain.ReceiptKindSelfIssued,
		ReceivedVia:  domain.ReceivedViaSelfIssued,
		ReceivedAt:   todayLocal(),
		DocumentDate: req.DocumentDate,
		IssuerName:   issuer,
		GrossAmount:  req.GrossAmount,
		TaxAmount:    req.TaxAmount,
		Currency:     currency,
		Subject:      reason,
		Files: []NewFile{{
			Role: domain.ReceiptRoleOriginal,
			// Nicht abgeleitet: das PDF *ist* der empfangene Beleg. Es aus einer
			// anderen Datei entstanden zu nennen wäre falsch — es gibt keine.
			FileName: "eigenbeleg.pdf",
			Content:  pdf,
		}},
	})
}

// DropRolledBackFiles entfernt die Dateien eines Belegs, dessen Anlage
// zurückgerollt wurde.
//
// Die Datenbank rollt zurück, der Belegspeicher nicht: das PDF des Eigenbelegs
// liegt auf der Platte, bevor die Transaktion überhaupt endet. Scheitert die
// Buchung danach, bliebe eine Datei ohne Beleg und ohne Buchung zurück — und
// der Integritätslauf fände sie als verwaiste Datei.
//
// Gelöscht wird nur, was niemand sonst braucht: der Speicher legt gleiche
// Inhalte einmal ab, deshalb wird vorher gefragt, ob ein anderer Beleg dieselbe
// Prüfsumme führt. Ein Löschen ohne diese Frage nähme einem bestehenden Beleg
// sein Original.
func (s *ReceiptService) DropRolledBackFiles(ctx context.Context, receipt *domain.Receipt) {
	if receipt == nil || s.store == nil {
		return
	}
	for _, f := range receipt.Files {
		if f.StoredPath == "" {
			continue
		}
		if s.receiptRepo != nil {
			other, err := s.receiptRepo.FindByOriginalHash(ctx, f.SHA256)
			if err != nil || other != nil {
				continue
			}
		}
		_ = s.store.Delete(f.StoredPath)
	}
}

// issuerName liefert den Aussteller: das eigene Unternehmen.
func (s *ReceiptService) issuerName(ctx context.Context) string {
	if s.settingsRepo == nil {
		return "Eigenes Unternehmen"
	}
	cfg, err := s.settingsRepo.GetCompanySettings(ctx)
	if err != nil || cfg == nil || cfg.FirmName() == "" {
		return "Eigenes Unternehmen"
	}
	// Die Firma, unter der das Unternehmen auftritt: der Eigenbeleg nennt seinen
	// Aussteller so, wie ihn ein Dritter auf einer Rechnung läse.
	return cfg.FirmName()
}

// nextSelfIssuedLabel nennt die Belegnummer, die der Beleg bekommen wird.
//
// Sie steht im PDF, weil ein Beleg ohne seine Nummer im Ordner nicht zu finden
// ist. Gelesen wird sie ohne den Zähler zu verbrauchen (Peek); vergeben wird sie
// in der Transaktion des Ablegens. Weichen die beiden auseinander — zwei
// Eigenbelege im selben Augenblick —, steht im PDF eine Nummer daneben, und das
// ist der kleinere Fehler: eine verbrauchte Nummer ohne Beleg wäre eine Lücke im
// Nummernkreis, die erklärt werden müsste.
func (s *ReceiptService) nextSelfIssuedLabel(ctx context.Context) string {
	if s.numberRepo == nil {
		return ""
	}
	seq, err := s.numberRepo.Peek(ctx, domain.NumberRangeReceipt, s.fiscalYear)
	if err != nil || seq <= 0 {
		return ""
	}
	format := domain.DefaultReceiptNumberFormat
	if s.settingsRepo != nil {
		if cfg, err := s.settingsRepo.GetCompanySettings(ctx); err == nil && cfg != nil {
			format = receiptNumberFormatOf(cfg)
		}
	}
	return domain.FormatReceiptNumberWith(format, s.fiscalYear, seq)
}

// SetNumberRepo hängt den Nummernkreis an, damit die Belegnummer schon im PDF
// steht. Optional: ohne ihn entsteht der Eigenbeleg ohne Nummer im Dokument.
func (s *ReceiptService) SetNumberRepo(repo domain.NumberRangeRepository) { s.numberRepo = repo }

// selfIssuedMarkdown ist der Text des Eigenbelegs.
//
// Als Markdown und nicht unmittelbar als Typst: derselbe Weg, den die
// Verfahrensdokumentation und das Mahnschreiben gehen. Der Satz ist dann eine
// Übersetzung dieses Textes und keine zweite Quelle, die auseinanderlaufen kann.
func selfIssuedMarkdown(req SelfIssuedReceiptRequest, issuer, createdBy, number, reason string) string {
	var b strings.Builder
	b.WriteString("# Eigenbeleg\n\n")
	if number != "" {
		fmt.Fprintf(&b, "**Belegnummer:** %s\n\n", number)
	}
	b.WriteString("| Angabe | Inhalt |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| Aussteller | %s |\n", issuer)
	fmt.Fprintf(&b, "| Datum des Vorgangs | %s |\n", germanDate(req.DocumentDate))
	fmt.Fprintf(&b, "| Betrag (brutto) | %s %s |\n", req.GrossAmount, currencyOrEuro(req.Currency))
	if req.TaxAmount != 0 {
		fmt.Fprintf(&b, "| davon Umsatzsteuer | %s %s |\n", req.TaxAmount, currencyOrEuro(req.Currency))
	}
	fmt.Fprintf(&b, "| Erfasst von | %s |\n", createdBy)
	fmt.Fprintf(&b, "| Erfasst am | %s |\n", germanDate(todayLocal()))
	b.WriteString("\n## Grund\n\n")
	b.WriteString(reason + "\n\n")
	b.WriteString("## Hinweis\n\n")
	b.WriteString(selfIssuedNotice + "\n\n")
	b.WriteString(
		"Der Eigenbeleg ist ein Buchungsbeleg im Sinne des § 147 Abs. 1 Nr. 4 AO und wird " +
			"entsprechend aufbewahrt.\n")
	return b.String()
}

// currencyOrEuro liefert die Währung oder den Euro.
func currencyOrEuro(code string) string {
	if strings.TrimSpace(code) == "" {
		return "EUR"
	}
	return code
}

// parseDayOrNow liefert den Tag des Vorgangs als Zeitpunkt für den Satz.
func parseDayOrNow(iso string) time.Time {
	day, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return time.Now()
	}
	return day
}

// RetentionOverride verlängert die Aufbewahrungsfrist eines Belegs (ARC-01 K2).
//
// Nur nach oben: die gesetzliche Frist ist die Untergrenze; eine kürzere
// wäre ein Verstoß gegen § 257 HGB und § 147 AO.
// Nach oben steht sie frei und ist oft geboten — ein Beleg, der zu einem
// laufenden Verfahren gehört, oder ein Vertrag, der über seine Belegfrist hinaus
// wirkt.
//
// Mit Grund und mit Vorher/Nachher im Protokoll: eine geänderte Frist ist eine
// Aussage über die Aufbewahrung, und wer sie später liest, muss wissen, worauf
// sie beruht und was vorher galt (GoBD Rz. 34).
func (s *ReceiptService) OverrideRetention(
	ctx context.Context, receiptID uint, class domain.RetentionClass, reason string,
) (*domain.Receipt, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("zur Verlängerung einer Aufbewahrungsfrist gehört ihr Grund")
	}
	receipt, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	origin := retentionOriginYear(receipt)
	current := accounting.RetentionFor(domain.RetentionKindOf(receipt.Kind), origin)
	target := accounting.RetentionForClass(class, origin)
	if target.Years == 0 {
		return nil, fmt.Errorf("die Aufbewahrungsklasse %q gibt es nicht", class)
	}
	// Verglichen werden die Enden und nicht die Klassen: die Klasse ist ein
	// Name, das Ende ist die Frist. Ein überschriebener Beleg, dessen Frist
	// bereits verlängert wurde, wird an seinem heutigen Ende gemessen — sonst
	// ließe sich eine Verlängerung über den Umweg einer zweiten Verlängerung
	// wieder zurücknehmen.
	effective := current.RetentionEnd
	if receipt.RetentionUntil > effective {
		effective = receipt.RetentionUntil
	}
	if target.RetentionEnd <= effective {
		return nil, fmt.Errorf(
			"die Aufbewahrungsfrist lässt sich nur verlängern. Für Beleg %s gilt sie bis zum %s; "+
				"„%s\" endete am %s. Die gesetzliche Frist ist die Untergrenze (§ 257 Abs. 4 HGB, "+
				"§ 147 Abs. 3 AO)",
			receipt.ReceiptNumber, effective, class.Label(), target.RetentionEnd)
	}

	at := time.Now().UTC().Format(time.RFC3339)
	if err := s.receiptRepo.SaveRetention(
		ctx, receiptID, class, target.RetentionEnd, reason, at); err != nil {
		return nil, err
	}
	updated, err := s.Get(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.LogChange(ctx, domain.AuditActionUpdate, "RECEIPT",
			fmt.Sprintf("%d", receiptID),
			fmt.Sprintf(
				"Aufbewahrungsfrist von Beleg %s von %s (%s) auf %s (%s) verlängert: %s",
				receipt.ReceiptNumber, effective, receipt.RetentionClass.Label(),
				target.RetentionEnd, class.Label(), reason),
			receipt, updated)
	}
	return updated, nil
}

// retentionOriginYear ist das Jahr, aus dem die Frist eines Belegs gerechnet
// wird: sein Belegjahr, ersatzweise sein Geschäftsjahr.
//
// Eine Funktion und keine zweite Abschrift der Regel: applyRetention entscheidet
// beim Ablegen genauso, und zwei Fassungen davon ergäben für denselben Beleg
// zwei verschiedene Fristen.
func retentionOriginYear(receipt *domain.Receipt) int {
	if len(receipt.DocumentDate) >= 4 {
		if year, err := strconv.Atoi(receipt.DocumentDate[:4]); err == nil && year > 1900 {
			return year
		}
	}
	return receipt.FiscalYear
}
