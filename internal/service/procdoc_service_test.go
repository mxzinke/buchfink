package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/procdoc"
	"github.com/buchfink/buchfink/internal/repository"
)

// --- Verfahrensdokumentation als PDF --------------------------------------

// Entscheidung 7 verlangt beide Formen: Markdown und PDF über Typst. Geprüft
// wird mit dem echten Setzer und nicht mit einer Attrappe — eine Attrappe
// bewiese nur, dass eine Zeichenkette weitergereicht wird, während die Frage
// ist, ob sich der übersetzte Text überhaupt setzen lässt.
func TestProcedureDocumentationIsAlsoTypesetAsPDF(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	svc.SetRenderer(sharedRenderer())
	ctx := context.Background()

	result, err := svc.Generate(ctx, 2026, procDocEnv(env), time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}
	if result.PDFNote != "" {
		t.Fatalf("der Satz ist gescheitert: %s", result.PDFNote)
	}
	doc := result.Document
	if doc.PDFStoredPath == "" || doc.PDFSHA256 == "" || doc.PDFSize == 0 {
		t.Fatalf("das PDF ist nicht abgelegt: %+v", doc)
	}
	if !strings.HasSuffix(doc.PDFFileName, ".pdf") ||
		strings.TrimSuffix(doc.FileName, ".md") != strings.TrimSuffix(doc.PDFFileName, ".pdf") {
		t.Errorf("die beiden Fassungen heißen verschieden: %q / %q", doc.FileName, doc.PDFFileName)
	}

	// Beide Formen liegen nebeneinander im Belegspeicher und sind über ihre
	// Prüfsumme wiederzufinden.
	content, err := os.ReadFile(filepath.Join(env.dataDir, doc.PDFStoredPath))
	if err != nil {
		t.Fatalf("das abgelegte PDF ließ sich nicht lesen: %v", err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF")) {
		t.Errorf("die abgelegte Datei ist kein PDF: % x", content[:8])
	}
	if int64(len(content)) != doc.PDFSize {
		t.Errorf("Größe %d, festgehalten %d", len(content), doc.PDFSize)
	}

	stored, err := svc.Documentations(ctx)
	if err != nil || len(stored) != 1 {
		t.Fatalf("die Fassung wurde nicht festgehalten: %v (%d)", err, len(stored))
	}
	if stored[0].PDFStoredPath != doc.PDFStoredPath {
		t.Errorf("der Verweis auf das PDF fehlt in der Historie: %+v", stored[0])
	}
}

// Ohne Setzer bleibt es beim Markdown — die Fassung gilt trotzdem, und der
// Grund steht im Ergebnis. Eine Verfahrensdokumentation an einer Formfrage
// scheitern zu lassen wäre die schlechtere Antwort.
func TestProcedureDocumentationSurvivesAFailedTypesetting(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	svc.SetRenderer(failingRenderer{})
	ctx := context.Background()

	result, err := svc.Generate(ctx, 2026, procDocEnv(env), time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ein Satzfehler darf die Fassung nicht verhindern: %v", err)
	}
	if result.Document.SHA256 == "" {
		t.Error("das Markdown muss trotzdem abgelegt sein")
	}
	if result.PDFNote == "" {
		t.Error("der Grund für das fehlende PDF muss im Ergebnis stehen")
	}
	if result.Document.PDFStoredPath != "" {
		t.Error("ohne Satz gibt es keinen PDF-Pfad")
	}
}

type failingRenderer struct{}

func (failingRenderer) RenderDocumentPDF(ctx context.Context, template, ident string) ([]byte, error) {
	return nil, fmt.Errorf("der Setzer steht nicht zur Verfügung")
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
	)
	return svc
}

// procDocEnv ist die Umgebung, mit der die Tests eine Fassung erzeugen.
func procDocEnv(env *testEnv) ProcDocEnvironment {
	return ProcDocEnvironment{
		DataDir:      env.dataDir,
		BackupDir:    "/media/sicherung",
		BackupRhythm: "täglich",
	}
}

func TestProcedureDocumentationCoversAllParts(t *testing.T) {
	env := newTestEnv(t)
	svc := newProcDocService(t, env)
	ctx := context.Background()

	result, err := svc.Generate(ctx, 2026, procDocEnv(env), time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
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
		"v0.1",                           // Eintrag der Historie
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

	first, err := svc.Generate(ctx, 2026, procDocEnv(env), day)
	if err != nil {
		t.Fatalf("erste Fassung: %v", err)
	}
	second, err := svc.Generate(ctx, 2026, procDocEnv(env), day.Add(2*time.Hour))
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
		t.Errorf("das Nachher muss den neuen Text enthalten: %q", entries[0].After)
	}

	result, err := svc.Generate(ctx, 2026, procDocEnv(env), time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("die Verfahrensdokumentation ließ sich nicht erzeugen: %v", err)
	}
	if !strings.Contains(result.Markdown, "Der Steuerberater scannt monatlich.") {
		t.Error("die unternehmensindividuellen Texte gehören in die erzeugte Fassung")
	}
}

// Die Verfahrensdokumentation nennt die tatsächlich eingestellte Systematik der
// Belegnummern (BEL-02 K4).
func TestProcDocShowsTheConfiguredReceiptNumberFormat(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	settingsRepo := repository.NewSettingsRepository(env.db)
	cfg, err := settingsRepo.GetCompanySettings(ctx)
	if err != nil {
		t.Fatalf("Unternehmensdaten: %v", err)
	}
	cfg.ReceiptNumberFormat = "BEL-{JAHR}-{NR:5}"
	if err := settingsRepo.UpdateCompanySettings(ctx, cfg); err != nil {
		t.Fatalf("Belegnummernformat setzen: %v", err)
	}

	result, err := newProcDocService(t, env).
		Generate(ctx, 2026, procDocEnv(env), time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Verfahrensdokumentation: %v", err)
	}
	if !strings.Contains(result.Markdown, "BEL-{JAHR}-{NR:5}") {
		t.Error("die Nummernkreistabelle nennt die eingestellte Systematik der Belegnummern nicht")
	}
	if strings.Contains(result.Markdown, "ER-{JAHR}-{NR:4}") {
		t.Error("die Verfahrensdokumentation beschreibt eine Systematik, die das Programm nicht " +
			"vergibt")
	}
}
