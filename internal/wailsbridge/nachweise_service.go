package wailsbridge

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/accounting"
	"github.com/buchfink/buchfink/internal/changelog"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/syncdir"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Die Nachweise: Änderungsprotokoll mit Kette, Versionshistorie,
// Migrationsprotokoll, Verfahrensdokumentation, Eröffnungsbilanz des
// Umsteigers und die Altersstruktur der offenen Posten.
//
// Sie stehen in einer eigenen Datei, weil sie eine Sache sind: die Antwort auf
// die Frage, woran ein Prüfer erkennt, dass die Buchführung ordnungsmäßig ist.

// --- Änderungsprotokoll ---------------------------------------------------

// VerifyAuditChain rechnet die Kette des Änderungsprotokolls nach.
func (b *BuchfinkBridge) VerifyAuditChain() (domain.AuditChainResult, error) {
	return readOr(b, func() (domain.AuditChainResult, error) {
		empty := domain.AuditChainResult{IsValid: true, Message: "Kein aktiver Mandant."}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (domain.AuditChainResult, error) {
		return s.Audit.VerifyChain(ctx)
	})
}

// GetAuditLogsFiltered liefert das Änderungsprotokoll mit Vorher/Nachher,
// eingeschränkt durch den Filter. Ein leerer Filter heißt: alles.
func (b *BuchfinkBridge) GetAuditLogsFiltered(limit int, filter domain.AuditFilter) ([]domain.AuditLogEntry, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.AuditLogEntry, error) {
		if limit <= 0 {
			limit = 200
		}
		return s.Audit.GetLogsFiltered(ctx, limit, filter)
	})
}

// GetChangeLog liefert die Versionshistorie des Programms als Zeilen.
//
// Zerlegt und nicht als Markdown: die Oberfläche zeigt eine Tabelle, und ein
// Markdown-Block, den sie selbst zerlegte, ginge beim nächsten Umbruch der
// Datei kaputt. Das Markdown selbst geht unzerlegt ins Prüferpaket
// (internal/service/export_service.go) — dort ist die Datei der Nachweis.
func (b *BuchfinkBridge) GetChangeLog() ([]changelog.Entry, error) {
	return emptyList(changelog.Entries(), nil)
}

// OpenReleasesPage öffnet die Fassungen des Projekts im Browser.
//
// Ohne Adresse als Parameter: eine Bridge-Methode, die jede übergebene Adresse
// öffnet, wäre ein Loch, durch das die Oberfläche den Rechner irgendwohin
// schickt. Es gibt genau ein Ziel, und es steht hier.
//
// Die eingebettete Historie bleibt daneben bestehen (GetChangeLog): dieser Weg
// braucht ein Netz, und die Frage „was hat diese Fassung geändert" stellt sich
// gerade dann, wenn keines da ist.
func (b *BuchfinkBridge) OpenReleasesPage() error {
	app := application.Get()
	if app == nil || app.Browser == nil {
		return fmt.Errorf("der Browser lässt sich von hier aus nicht öffnen")
	}
	return app.Browser.OpenURL(releasesURL)
}

// releasesURL ist die Fassungsübersicht des Projekts.
const releasesURL = "https://github.com/mxzinke/buchfink/releases"

// GetSchemaMigrations liefert das Protokoll der Schemaänderungen.
func (b *BuchfinkBridge) GetSchemaMigrations() ([]domain.SchemaMigration, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.SchemaMigration, error) {
		return s.Repos.Migrations.FindSchemaMigrations(ctx)
	})
}

// GetMigrationRecords liefert das Protokoll der Datenübernahmen.
func (b *BuchfinkBridge) GetMigrationRecords() ([]domain.MigrationRecord, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.MigrationRecord, error) {
		return s.Repos.Migrations.FindMigrationRecords(ctx)
	})
}

// --- Verfahrensdokumentation ----------------------------------------------

// GenerateProcedureDocumentation erzeugt eine neue Fassung und legt sie ab.
func (b *BuchfinkBridge) GenerateProcedureDocumentation() (*service.ProcDocResult, error) {
	year := b.viewYear()
	return operate(b, func(ctx context.Context, t *core.Tenant) (*service.ProcDocResult, error) {
		return t.GenerateProcedureDocumentation(ctx, year)
	})
}

// GetProcedureDocumentations liefert die abgelegten Fassungen.
func (b *BuchfinkBridge) GetProcedureDocumentations() ([]domain.ProcedureDocumentation, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.ProcedureDocumentation, error) {
		return s.ProcDoc.Documentations(ctx)
	})
}

// SaveProcedureDocumentationAs schreibt eine abgelegte Fassung an einen vom
// Anwender gewählten Ort. `wantPDF` wählt das gesetzte PDF statt des Markdowns.
//
// Die Fassung bleibt dabei im Belegspeicher: dies ist eine Kopie hinaus und
// kein Umzug. Gebraucht wird sie, wenn ein Prüfer die Verfahrensdokumentation
// einzeln anfordert — im Prüferpaket liegt sie ohnehin bei, und der Weg über
// den Datenordner setzte voraus, dass jemand den Pfad einer Zeile in der
// Oberfläche abtippt.
//
// Der leere Rückgabewert heißt: der Speichern-Dialog wurde abgebrochen. Das ist
// keine Fehlermeldung wert.
func (b *BuchfinkBridge) SaveProcedureDocumentationAs(id uint, wantPDF bool) (string, error) {
	// Gelesen wird vor dem Dialog: er ist modal und steht offen, solange der
	// Anwender überlegt.
	file, err := read(b, func(ctx context.Context, s *core.Services) (*service.ProcDocFile, error) {
		return s.ProcDoc.File(ctx, id, wantPDF)
	})
	if err != nil {
		return "", err
	}

	path, err := b.saveFileDialog("Verfahrensdokumentation speichern", file.FileName)
	if err != nil || path == "" {
		return "", err
	}
	if err := b.handOut(path, file.Data, "PROCEDURE_DOC", strconv.FormatUint(uint64(id), 10),
		fmt.Sprintf("Verfahrensdokumentation %s nach %s herausgegeben", file.FileName, path)); err != nil {
		return "", err
	}
	return path, nil
}

// GetOrganisationTexts liefert die unternehmensindividuellen Freitexte, mit
// Mustern dort, wo noch nichts erfasst ist.
func (b *BuchfinkBridge) GetOrganisationTexts() (domain.OrganisationTexts, error) {
	return readOr(b, func() (domain.OrganisationTexts, error) {
		return domain.DefaultOrganisationTexts(), nil
	}, func(ctx context.Context, s *core.Services) (domain.OrganisationTexts, error) {
		return s.ProcDoc.OrganisationTexts(ctx), nil
	})
}

// GetOrganisationTextDefaults liefert die Muster, mit denen die Freitexte
// vorbelegt sind.
//
// Getrennt von den Texten selbst, weil GetOrganisationTexts das eine nicht vom
// anderen unterscheidet: es liefert den erfassten Text und dort, wo keiner
// steht, das Muster. Die Oberfläche kann daran nicht erkennen, welcher
// Abschnitt beschrieben und welcher nur vorbelegt ist — und genau das muss sie
// sagen können. Eine Verfahrensdokumentation mit unverändertem Muster behauptet
// eine Beschreibung, die niemand geprüft hat.
func (b *BuchfinkBridge) GetOrganisationTextDefaults() (domain.OrganisationTexts, error) {
	return domain.DefaultOrganisationTexts(), nil
}

// SaveOrganisationTexts schreibt die Freitexte der Organisationsanweisung.
func (b *BuchfinkBridge) SaveOrganisationTexts(texts domain.OrganisationTexts) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		return s.ProcDoc.SaveOrganisationTexts(ctx, texts)
	})
}

// SetSystemChangeDate hält den Umstellungszeitpunkt der Datenübernahme fest.
//
// An ihm hängt die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO: so lange ist das
// Altsystem für den Datenzugriff verfügbar zu halten.
func (b *BuchfinkBridge) SetSystemChangeDate(date string) error {
	return b.change(func(ctx context.Context, s *core.Services) error {
		trimmed := strings.TrimSpace(date)
		if trimmed != "" {
			if _, err := time.Parse("2006-01-02", trimmed); err != nil {
				return fmt.Errorf("der Umstellungszeitpunkt wird als Datum erwartet (JJJJ-MM-TT)")
			}
		}
		return s.Settings.SetValue(ctx, domain.SettingSystemChangeDate, trimmed,
			"Umstellungszeitpunkt der Datenübernahme aus einem Altsystem gesetzt")
	})
}

// --- Hinweise -------------------------------------------------------------

// ComplianceHints sind die Hinweise, die zur Einrichtung und zu den
// Einstellungen gehören.
type ComplianceHints struct {
	// CloudWarning ist belegt, wenn ein Dienst den Datenordner synchronisiert
	// (syncdir, dieselbe Erkennung wie in der Einrichtung).
	CloudWarning string `json:"cloudWarning"`
	// DataDir ist der Speicherort, damit die Oberfläche ihn nennen kann.
	DataDir string `json:"dataDir"`
	// TaxCaseHints nennt die Steuerfälle außerhalb des Funktionsumfangs.
	TaxCaseHints []string `json:"taxCaseHints"`
	// SystemChangeDate und SystemChangeNote sind der Umstellungszeitpunkt und
	// die Fünfjahresfrist daraus.
	SystemChangeDate string `json:"systemChangeDate"`
	SystemChangeNote string `json:"systemChangeNote"`
}

// GetComplianceHints liefert die Hinweise zu Speicherort und Steuerfällen.
func (b *BuchfinkBridge) GetComplianceHints() (*ComplianceHints, error) {
	dataDir := b.host.Config().DataDir
	hints := &ComplianceHints{
		DataDir:      dataDir,
		CloudWarning: syncdir.Warning(syncdir.Detect(dataDir)),
		TaxCaseHints: domain.TaxCaseHints(),
	}
	date, err := read(b, func(ctx context.Context, s *core.Services) (string, error) {
		return s.Repos.Settings.Get(ctx, domain.SettingSystemChangeDate)
	})
	if err == nil {
		hints.SystemChangeDate = date
	}
	if hints.SystemChangeDate != "" {
		hints.SystemChangeNote = accounting.SystemChangeNote(hints.SystemChangeDate, "")
	}
	return hints, nil
}

// --- Kontakte -------------------------------------------------------------

// BlockContactResult ist das Ergebnis einer Sperre nach einem Löschverlangen.
type BlockContactResult struct {
	Contact *domain.Contact `json:"contact"`
	// Answer ist der Antworttext an die betroffene Person, mit den Normen, auf
	// die sich die Ablehnung des Löschverlangens stützt.
	Answer string `json:"answer"`
}

// BlockContact sperrt einen Geschäftspartner nach einem Löschverlangen.
func (b *BuchfinkBridge) BlockContact(id uint, reason string) (*BlockContactResult, error) {
	return write(b, func(ctx context.Context, s *core.Services) (*BlockContactResult, error) {
		contact, answer, err := s.Contacts.BlockContact(ctx, id, reason)
		if err != nil {
			return nil, err
		}
		return &BlockContactResult{Contact: contact, Answer: answer}, nil
	})
}

// GetSelectableContacts liefert die Geschäftspartner, die sich noch auswählen
// lassen — gesperrte fehlen.
func (b *BuchfinkBridge) GetSelectableContacts() ([]domain.Contact, error) {
	return readList(b, func(ctx context.Context, s *core.Services) ([]domain.Contact, error) {
		return s.Contacts.SelectableContacts(ctx, b.viewYear())
	})
}

// GetOpenItemsAging liefert Altersstruktur und Restlaufzeiten der offenen
// Posten zum Stichtag. Leerer Stichtag heißt: heute.
func (b *BuchfinkBridge) GetOpenItemsAging(cutoff string) (*domain.OpenItemsAging, error) {
	return readOr(b, func() (*domain.OpenItemsAging, error) {
		empty := &domain.OpenItemsAging{Cutoff: cutoff}
		empty.EnsureLists()
		return empty, nil
	}, func(ctx context.Context, s *core.Services) (*domain.OpenItemsAging, error) {
		if cutoff == "" {
			cutoff = time.Now().UTC().Format("2006-01-02")
		}
		items, err := s.Payments.OpenItemsAt(ctx, cutoff)
		if err != nil {
			return nil, err
		}
		aging := accounting.AgeOpenItems(items, cutoff)
		// Die Zusicherung steht hier und nicht nur bei der leeren Antwort: eine
		// Liste, die an einer Stelle `null` sein darf, ist keine Zusicherung.
		aging.EnsureLists()
		return &aging, nil
	})
}
