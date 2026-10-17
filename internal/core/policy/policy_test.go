package policy

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/buchfink/buchfink/internal/domain"
)

// routine ist eine E-Rechnung, die alle Regeln erfüllt.
func routine() ReceiptFacts {
	net := map[domain.TaxRate]domain.Cents{domain.TaxRateStandard: 10000}
	return ReceiptFacts{
		Open: true, Displayable: true, Incoming: true, Invoice: true, Structured: true, Checked: true,
		ProposalContactID: 7, ProposalMatchedBy: "ust_id", ProposalTreatment: domain.TaxTreatmentDomestic, ProposalCurrency: "EUR",
		ProposalNet: net, ProposalGross: 11900,
		ContactID: 7, ConfirmedGroup: "software", BookingsWithGroup: 3,
		Treatment: domain.TaxTreatmentDomestic, Currency: "EUR",
		Net: map[domain.TaxRate]domain.Cents{domain.TaxRateStandard: 10000}, Gross: 11900,
		Groups: []string{"software"}, Settlement: "open",
		BookingDate: "2026-05-04", DocumentDate: "2026-05-04", Today: "2026-05-10",
	}
}

func TestReceiptRules(t *testing.T) {
	limits := LimitsFrom(nil)
	if limits.MaxGross != 100000 || limits.MinBookings != 3 {
		t.Fatalf("Voreinstellung: %+v", limits)
	}
	if v := ClassifyReceipt(routine(), limits); !v.Direct() || len(v.Reasons) != 0 {
		t.Fatalf("Routine wird %v: %+v", v.Level, v.Reasons)
	}
	cases := []struct {
		name   string
		change func(*ReceiptFacts)
		reason string
	}{
		{"gebucht", func(f *ReceiptFacts) { f.Open = false }, "beleg_nicht_offen"},
		{"ohne Darstellung", func(f *ReceiptFacts) { f.Displayable = false }, "keine_darstellung"},
		{"Foto", func(f *ReceiptFacts) { f.Structured = false }, "keine_e_rechnung"},
		{"ungeprüft", func(f *ReceiptFacts) { f.Checked = false }, "e_rechnung_ungeprueft"},
		{"Befund", func(f *ReceiptFacts) { f.Findings = 1 }, "e_rechnung_mit_befund"},
		{"kein Vorschlag", func(f *ReceiptFacts) { f.ProposalError = "Gutschrift" }, "kein_vorschlag"},
		{"anderer Aussteller", func(f *ReceiptFacts) { f.ProposalContactID = 8 }, "kontakt_nicht_erkannt"},
		{"nur der Name", func(f *ReceiptFacts) { f.ProposalMatchedBy = "name" }, "kontakt_nicht_erkannt"},
		{"Kopfdaten", func(f *ReceiptFacts) { f.HeaderMismatch = []string{"belegdatum"} }, "kopf_weicht_ab"},
		{"Stammdaten", func(f *ReceiptFacts) { f.MasterDataMismatch = []string{"anschrift"} }, "stammdaten_weichen_ab"},
		{"anderes Konto", func(f *ReceiptFacts) { f.IBANMismatch = true }, "iban_weicht_ab"},
		{"gesperrt", func(f *ReceiptFacts) { f.ContactBlocked = true }, "kontakt_gesperrt"},
		{"nur Vorschlag", func(f *ReceiptFacts) { f.ConfirmedGroup = "" }, "keine_bestaetigte_standardgruppe"},
		{"andere Gruppe", func(f *ReceiptFacts) { f.Groups = []string{"software", "werbung"} }, "gruppe_weicht_ab"},
		{"zwei Buchungen", func(f *ReceiptFacts) { f.BookingsWithGroup = 2 }, "zu_wenige_buchungen"},
		{"Reverse Charge", func(f *ReceiptFacts) { f.Treatment = domain.TaxTreatmentReverseChargeEU }, "steuerfall_nicht_inland"},
		{"Dollar", func(f *ReceiptFacts) { f.Currency = "USD" }, "fremdwaehrung"},
		{"ein Cent", func(f *ReceiptFacts) { f.Gross = 11901 }, "betraege_weichen_ab"},
		{"anderer Satz", func(f *ReceiptFacts) {
			f.Net = map[domain.TaxRate]domain.Cents{domain.TaxRateReduced: 10000}
		}, "betraege_weichen_ab"},
		{"an der Grenze", func(f *ReceiptFacts) {
			f.Gross, f.ProposalGross = 100000, 100000
		}, "ueber_direktgrenze"},
		{"Bewirtung", func(f *ReceiptFacts) { f.Special = []string{"bewirtung"} }, "sonderangaben"},
		{"Auslage", func(f *ReceiptFacts) { f.Settlement = "shareholder_expense" }, "bezahlart"},
		{"Zukunft", func(f *ReceiptFacts) { f.DocumentDate = "2026-06-01" }, "datum_in_zukunft"},
		{"Dublette", func(f *ReceiptFacts) { f.DuplicateOf = "ER-2026-0003" }, "dublettenverdacht"},
	}
	for _, tc := range cases {
		f := routine()
		tc.change(&f)
		v := ClassifyReceipt(f, limits)
		if v.Direct() || !slices.Contains(v.Keys(), tc.reason) {
			t.Errorf("%s: %v %v, erwartet %s", tc.name, v.Level, v.Keys(), tc.reason)
		}
	}
	// Direkt geht nur, was unter der Grenze liegt.
	f := routine()
	f.Gross, f.ProposalGross = 99999, 99999
	if v := ClassifyReceipt(f, limits); !v.Direct() {
		t.Errorf("ein Cent unter der Grenze: %v", v.Keys())
	}
	// Eingestellte Schwellen gelten.
	strict := LimitsFrom(&domain.AgentSettings{DirectMaxGross: 5000, DirectMinBookings: 5})
	if v := ClassifyReceipt(routine(), strict); !slices.Contains(v.Keys(), "ueber_direktgrenze") ||
		!slices.Contains(v.Keys(), "zu_wenige_buchungen") {
		t.Errorf("eingestellte Schwellen: %v", v.Keys())
	}
}

func TestBankRules(t *testing.T) {
	base := BankFacts{Open: true, Currency: "EUR", Trusted: true, Requested: 4, Candidates: []BankCandidate{{EntryID: 4, Reference: "RE-2026-0004"}}}
	if v := ClassifyBankMatch(base); !v.Direct() {
		t.Fatalf("exakter Treffer: %v", v.Keys())
	}
	cases := []struct {
		name   string
		change func(*BankFacts)
		reason string
	}{
		{"zugeordnet", func(f *BankFacts) { f.Open = false }, "umsatz_nicht_offen"},
		{"Dollar", func(f *BankFacts) { f.Currency = "USD" }, "fremdwaehrung"},
		{"ungeprüfter Auszug", func(f *BankFacts) { f.Trusted = false }, "auszug_ungeprueft"},
		{"ohne Nummer", func(f *BankFacts) { f.Candidates[0].Reference = "" }, "kein_exakter_treffer"},
		{"kein Betrag", func(f *BankFacts) { f.Candidates = nil }, "kein_exakter_treffer"},
		{"zwei", func(f *BankFacts) {
			f.Candidates = append(f.Candidates, BankCandidate{EntryID: 5, Reference: "RE-2026-0005"})
		}, "mehrere_treffer"},
		{"anderer", func(f *BankFacts) { f.Requested = 9 }, "anderer_posten"},
	}
	for _, tc := range cases {
		f := base
		f.Candidates = append([]BankCandidate{}, base.Candidates...)
		tc.change(&f)
		if v := ClassifyBankMatch(f); v.Direct() || !slices.Contains(v.Keys(), tc.reason) {
			t.Errorf("%s: %v, erwartet %s", tc.name, v.Keys(), tc.reason)
		}
	}
	// Ein zweiter Posten mit gleichem Betrag, aber ohne Nummer, stört nicht.
	f := base
	f.Candidates = []BankCandidate{{EntryID: 4, Reference: "RE-2026-0004"}, {EntryID: 5}}
	if v := ClassifyBankMatch(f); !v.Direct() {
		t.Errorf("gleicher Betrag ohne Nummer: %v", v.Keys())
	}
}

func TestDuplicateContact(t *testing.T) {
	existing := []domain.Contact{
		{ID: 1, Type: domain.ContactTypeVendor, Name: "Netzwerk GmbH", PostalCode: "20095", VatID: "DE555666777"},
		{ID: 2, Type: domain.ContactTypeVendor, Name: "Papier AG", IBAN: "DE02 1203 0000 0000 2020 51"},
		{ID: 3, Type: domain.ContactTypeCustomer, Name: "Kunde KG", PostalCode: "80331"},
	}
	cases := []struct {
		c     domain.Contact
		id    uint
		match string
	}{
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Anders", VatID: "de 555666777"}, 1, "ust_id"},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Anders", IBAN: "DE02120300000000202051"}, 2, "iban"},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "netzwerk gmbh", PostalCode: "20095"}, 1, "name_plz"},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Netzwerk GmbH", PostalCode: "10115"}, 0, ""},
		{domain.Contact{Type: domain.ContactTypeVendor, Name: "Kunde KG", PostalCode: "80331"}, 0, ""},
	}
	for _, tc := range cases {
		found, match := DuplicateContact(existing, tc.c)
		id := uint(0)
		if found != nil {
			id = found.ID
		}
		if id != tc.id || match != tc.match {
			t.Errorf("%+v: %d %q, erwartet %d %q", tc.c, id, match, tc.id, tc.match)
		}
	}
}

func TestReadFileStaysInsideTheReceiptFolders(t *testing.T) {
	base := t.TempDir()
	data := filepath.Join(base, "daten")
	inbox := filepath.Join(data, InboxDir)
	downloads := filepath.Join(base, "Downloads")
	outside := filepath.Join(base, "privat")
	for _, d := range []string{inbox, downloads, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string, content string) string {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	pdf := "%PDF-1.7\n..."
	write(filepath.Join(data, "buchfink.sqlite"), "SQLite format 3")
	f := Folders{Receipt: []string{inbox, downloads}, DataDir: data}

	if got, err := ReadFile(write(filepath.Join(inbox, "rechnung.pdf"), pdf), f); err != nil || got.Kind != KindPDF {
		t.Fatalf("Eingang: %v %+v", err, got)
	}
	if _, err := ReadFile(write(filepath.Join(downloads, "e-rechnung.xml"), "\xef\xbb\xbf <?xml version=\"1.0\"?><x/>"), f); err != nil {
		t.Errorf("XML im Downloads-Ordner: %v", err)
	}

	secret := write(filepath.Join(outside, "ausweis.pdf"), pdf)
	link := filepath.Join(inbox, "rechnung-link.pdf")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}
	dbLink := filepath.Join(inbox, "db.xml")
	if err := os.Symlink(filepath.Join(data, "buchfink.sqlite"), dbLink); err != nil {
		t.Fatal(err)
	}
	refused := []struct {
		name, path string
		notAllowed bool
	}{
		{"außerhalb", secret, true},
		{"Symlink hinaus", link, true},
		{"Punkte hinaus", inbox + "/../../privat/ausweis.pdf", true},
		{"relativ", "Belegeingang/rechnung.pdf", true},
		{"Datenbank über Symlink", dbLink, true},
		{"Datenordner", filepath.Join(data, "buchfink.sqlite"), true},
		{"versteckt", write(filepath.Join(inbox, ".rechnung.pdf"), pdf), true},
		{"falsche Endung", write(filepath.Join(inbox, "skript.sh"), "#!/bin/sh"), false},
		{"falscher Inhalt", write(filepath.Join(inbox, "fake.pdf"), "MZ\x90"), false},
		{"leer", write(filepath.Join(inbox, "leer.pdf"), ""), false},
		{"fehlt", filepath.Join(inbox, "nichtda.pdf"), false},
		{"fehlt außerhalb", filepath.Join(outside, "nichtda.pdf"), true},
		{"Ordner selbst", downloads, true},
		{"Netzwerkpfad", "//server/freigabe/rechnung.pdf", true},
		{"Netzwerkpfad Windows", `\\server\freigabe\rechnung.pdf`, true},
		{"Hardlink hinaus", hardLink(t, secret, filepath.Join(inbox, "hart.pdf")), true},
		{"Hardlink in den Datenordner", hardLink(t, write(filepath.Join(data, "intern.pdf"), pdf),
			filepath.Join(inbox, "hart-daten.pdf")), true},
	}
	for _, tc := range refused {
		_, err := ReadFile(tc.path, f)
		var fe *FileError
		if !errors.As(err, &fe) {
			t.Errorf("%s: erwartet Abweisung, bekam %v", tc.name, err)
			continue
		}
		if fe.NotAllowed != tc.notAllowed {
			t.Errorf("%s: NotAllowed=%v (%v)", tc.name, fe.NotAllowed, err)
		}
	}
	// Nur XML, wo ein Kontoauszug erwartet wird.
	if _, err := ReadFile(filepath.Join(inbox, "rechnung.pdf"), f, KindXML); err == nil {
		t.Error("ein PDF ist kein Kontoauszug")
	}
	// Eine leere Liste sperrt alles, auch den Eingang.
	if _, err := ReadFile(filepath.Join(inbox, "rechnung.pdf"), FoldersFrom(&domain.AgentSettings{ReceiptFolders: []string{}}, data)); err == nil {
		t.Error("ohne Belegordner darf nichts gelesen werden")
	}
}

func hardLink(t *testing.T, target, link string) string {
	t.Helper()
	if err := os.Link(target, link); err != nil {
		t.Fatal(err)
	}
	return link
}

// Zwei Mandanten unter einem gemeinsamen Ordner: der Belegordner des einen
// umschließt den Datenordner des anderen. Gelesen wird trotzdem nur aus dem
// eigenen Eingang und aus Ordnern, die keinem Mandanten gehören.
func TestReceiptFolderNeverReachesAnotherTenant(t *testing.T) {
	parent := t.TempDir()
	own := filepath.Join(parent, "Pfennig GmbH")
	other := filepath.Join(parent, "Fremd GmbH")
	backup := filepath.Join(parent, "Sicherung")
	for _, d := range []string{filepath.Join(own, InboxDir), filepath.Join(other, InboxDir), backup, filepath.Join(parent, "Scans")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	xml := []byte("<?xml version=\"1.0\"?><x/>")
	for _, p := range []string{filepath.Join(other, InboxDir, "geheim.xml"), filepath.Join(other, "beleg.xml"),
		filepath.Join(backup, "sicherung.xml"), filepath.Join(own, InboxDir, "eigen.xml"), filepath.Join(parent, "Scans", "scan.xml")} {
		if err := os.WriteFile(p, xml, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := domain.AppConfig{BackupDir: backup, Tenants: []domain.TenantConfig{
		{ID: "a", DataDir: own, BackupDir: backup}, {ID: "b", DataDir: other},
	}}
	f := FoldersFrom(&domain.AgentSettings{ReceiptFolders: []string{parent, filepath.Join(own, InboxDir)}}, own,
		ProtectedDirs(cfg, own)...)
	for _, p := range []string{filepath.Join(other, InboxDir, "geheim.xml"), filepath.Join(other, "beleg.xml"),
		filepath.Join(backup, "sicherung.xml")} {
		var fe *FileError
		if _, err := ReadFile(p, f); !errors.As(err, &fe) || !fe.NotAllowed {
			t.Errorf("%s: %v", p, err)
		}
	}
	for _, p := range []string{filepath.Join(own, InboxDir, "eigen.xml"), filepath.Join(parent, "Scans", "scan.xml")} {
		if _, err := ReadFile(p, f); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}

	// Einstellen lässt sich ein solcher Ordner gar nicht erst.
	protected := ProtectedDirs(cfg, own)
	for _, folder := range []string{parent, other, filepath.Join(other, InboxDir), backup, own} {
		if err := CheckReceiptFolders([]string{folder}, own, protected); err == nil {
			t.Errorf("Belegordner %s angenommen", folder)
		}
	}
	if err := CheckReceiptFolders([]string{filepath.Join(own, InboxDir), filepath.Join(parent, "Scans")}, own, protected); err != nil {
		t.Errorf("eigener Eingang und fremder Ordner: %v", err)
	}
}

func TestDefaultFolders(t *testing.T) {
	data := t.TempDir()
	f := FoldersFrom(nil, data)
	if len(f.Receipt) != 1 || f.Receipt[0] != filepath.Join(data, InboxDir) {
		t.Errorf("Voreinstellung Belegordner ist allein der Eingang: %v", f.Receipt)
	}
	if f.Export != filepath.Join(data, ExportDir) {
		t.Errorf("Voreinstellung Exportordner: %s", f.Export)
	}
	path, err := f.ExportPath("e-bilanz-2026.xbrl")
	if err != nil || filepath.Dir(path) != f.Export {
		t.Fatalf("Exportpfad: %s %v", path, err)
	}
	if _, err := f.ExportPath("../hinaus.xml"); err == nil {
		t.Error("ein Dateiname mit Pfad darf nicht hinaus")
	}
	custom := FoldersFrom(&domain.AgentSettings{ExportFolder: filepath.Join(data, "fehlt")}, data)
	if _, err := custom.ExportPath("x.json"); err == nil {
		t.Error("ein eingestellter Exportordner wird nicht angelegt")
	}
}
