package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
	"gorm.io/gorm"
)

type settingsRepositoryGorm struct {
	db *gorm.DB
}

// NewSettingsRepository creates a new GORM-backed SettingsRepository.
func NewSettingsRepository(db *gorm.DB) domain.SettingsRepository {
	return &settingsRepositoryGorm{db: db}
}

func (r *settingsRepositoryGorm) Get(ctx context.Context, key string) (string, error) {
	var item domain.SettingItem
	err := dbFrom(ctx, r.db).Where("key = ?", key).First(&item).Error
	if err != nil {
		return "", err
	}
	return item.Value, nil
}

func (r *settingsRepositoryGorm) Set(ctx context.Context, key string, value string) error {
	item := domain.SettingItem{
		Key:   key,
		Value: value,
		// UTC: der Zeitpunkt wird gespeichert und später verglichen; eine
		// Ortszeit ohne Zone ist in der Nacht der Zeitumstellung mehrdeutig.
		UpdatedAt: time.Now().UTC(),
	}
	return dbFrom(ctx, r.db).Save(&item).Error
}

func (r *settingsRepositoryGorm) GetCompanySettings(ctx context.Context) (*domain.CompanySettings, error) {
	var items []domain.SettingItem
	if err := dbFrom(ctx, r.db).Find(&items).Error; err != nil {
		return nil, err
	}

	settings := &domain.CompanySettings{
		FiscalYearStartMonth: 1,
		Currency:             "EUR",
		SKR:                  "SKR04",
		VatPeriod:            "quarter",
		TaxationType:         "SOLL",
		// Zehn Tage sind die Erfassungsfrist der GoBD Rz. 47; ohne Vorgabe
		// stünde hier null und der Prüflauf meldete jeden Beleg am Tag seines
		// Eingangs als überfällig.
		ReceiptCaptureDays: 10,
		// Tausend Euro sind die Voreinstellung, ab der ein Eingangsbeleg einen
		// Leistungsnachweis haben soll (RECH-08). Sie ist eine Vorgabe des
		// internen Kontrollsystems und keine Rechtspflicht — deshalb
		// einstellbar.
		InvoiceCheckThreshold: 100_000,
		// Die Mahnstufen der Voreinstellung. Sie stehen im Fachbereich und
		// nicht hier: die Reihenfolge und ihre Begründung gehören zusammen.
		DunningLevels: domain.DefaultDunningLevels(),
	}

	// Der Tag, ab dem der fehlende Leistungsnachweis beanstandet wird. Er kann
	// aus zwei Quellen kommen; der ausdrückliche Eintrag geht vor.
	checkSince, thresholdSavedOn := "", ""

	for _, it := range items {
		switch it.Key {
		case "company_name":
			settings.CompanyName = it.Value
		case "legal_form":
			settings.LegalForm = it.Value
		case "fiscal_year":
			y, _ := strconv.Atoi(it.Value)
			settings.FiscalYear = y
		case "fiscal_year_start_month":
			m, _ := strconv.Atoi(it.Value)
			if m >= 1 && m <= 12 {
				settings.FiscalYearStartMonth = m
			}
		case "tax_number":
			settings.TaxNumber = it.Value
		case "vat_id":
			settings.VatID = it.Value
		case "tax_office":
			settings.TaxOffice = it.Value
		case "iban":
			settings.IBAN = it.Value
		case "bic":
			settings.BIC = it.Value
		case "bank_name":
			settings.BankName = it.Value
		case "street":
			settings.Street = it.Value
		case "zip_city":
			settings.ZipCity = it.Value
		case "country":
			settings.Country = it.Value
		case "contact_name":
			settings.ContactName = it.Value
		case "contact_phone":
			settings.ContactPhone = it.Value
		case "contact_email":
			settings.ContactEmail = it.Value
		case "invoice_number_format":
			settings.InvoiceNumberFormat = it.Value
		case "receipt_number_format":
			settings.ReceiptNumberFormat = it.Value
		case "seat":
			settings.Seat = it.Value
		case "register_court":
			settings.RegisterCourt = it.Value
		case "register_number":
			settings.RegisterNumber = it.Value
		case "vat_period":
			settings.VatPeriod = it.Value
		case "taxation_type":
			settings.TaxationType = it.Value
		case "investor_override":
			settings.InvestorOverride = domain.InvestorType(it.Value)
		case "permanent_extension":
			settings.PermanentExtension = it.Value == "true"
		case "special_prepayment":
			v, _ := strconv.ParseInt(it.Value, 10, 64)
			settings.SpecialPrepayment = domain.Cents(v)
		case "receipt_capture_days":
			if d, err := strconv.Atoi(it.Value); err == nil && d > 0 {
				settings.ReceiptCaptureDays = d
			}
		case "commit_grace_days":
			if d, err := strconv.Atoi(it.Value); err == nil && d >= 0 {
				settings.CommitGraceDays = d
			}
		case "invoice_check_threshold":
			if v, err := strconv.ParseInt(it.Value, 10, 64); err == nil && v >= 0 {
				settings.InvoiceCheckThreshold = domain.Cents(v)
			}
			// Der Rückfall für Bestände aus der Zeit vor dem eigenen Schlüssel:
			// der Tag, an dem die Grenze zuletzt gespeichert wurde. Er ist nicht
			// so genau wie der eigene Eintrag, aber besser als kein Datum —
			// ohne eines beanstandete der Prüflauf jeden Altbeleg rückwirkend.
			// Ein vorhandener invoice_check_since sticht ihn (siehe unten).
			thresholdSavedOn = it.UpdatedAt.Format("2006-01-02")
		case "invoice_check_since":
			checkSince = it.Value
		case "dunning_levels":
			// Ein unlesbarer Eintrag bleibt ohne Wirkung: dann gilt die
			// Voreinstellung weiter. Die Mahnstufen ganz fallen zu lassen,
			// hieße den Mahnlauf abzuschalten, weil ein Zeichen fehlt.
			var levels []domain.DunningLevel
			if err := json.Unmarshal([]byte(it.Value), &levels); err == nil && len(levels) > 0 {
				settings.DunningLevels = domain.NormalizeDunningLevels(levels)
			}
		}
	}

	if settings.FiscalYearStartMonth <= 0 || settings.FiscalYearStartMonth > 12 {
		settings.FiscalYearStartMonth = 1
	}
	settings.InvoiceCheckSince = checkSince
	if settings.InvoiceCheckSince == "" {
		settings.InvoiceCheckSince = thresholdSavedOn
	}
	settings.InGruendung = r.inGruendung(ctx)

	return settings, nil
}

// inGruendung liest aus der Gründung, ob die Gesellschaft noch Vorgesellschaft
// ist: beurkundet, aber nicht eingetragen.
//
// Der Zustand steht hier und nicht in den Schlüssel-Wert-Zeilen, weil er eine
// Tatsache ist und keine Einstellung — er folgt aus dem Eintragungsdatum. Und er
// wird beim Lesen der Unternehmensdaten gesetzt und nicht von jedem
// Ausgabeweg einzeln erfragt: den Firmennamen brauchen Rechnung, E-Rechnung,
// E-Bilanz, Abschlusskopf, Mahnschreiben und Eigenbeleg, und jedem von ihnen die
// Gründung durchzureichen hieße, sieben Dienste um eine Abhängigkeit zu
// erweitern, die sie nur weitergeben.
//
// Ein Lesefehler heißt „nicht in Gründung": ohne Zusatz auszugeben ist der
// mildere Fehler — er behauptet nichts über den Stand des Registers.
func (r *settingsRepositoryGorm) inGruendung(ctx context.Context) bool {
	// `Find` und nicht `First`: eine fehlende Gründungszeile ist der Regelfall
	// jedes Mandanten, der nicht gerade gegründet hat. `First` meldete sie als
	// „record not found" ins Protokoll — bei jedem Lesen der Unternehmensdaten.
	var found []domain.Foundation
	if err := dbFrom(ctx, r.db).Select("registered_on").Limit(1).Find(&found).Error; err != nil {
		return false
	}
	return len(found) == 1 && found[0].RegisteredOn == ""
}

// numberFormatOrDefault prüft die Systematik des Rechnungsnummernkreises.
//
// Ein leeres Feld heißt „nicht festgelegt" und bekommt die Voreinstellung. Ein
// ausgefülltes, aber untaugliches Format wird abgewiesen und nicht ersetzt:
// wer `RE-{JAHR}` einträgt, hat einen Nummernkreis gemeint, in dem jede
// Rechnung dieselbe Nummer hätte — das stillschweigend durch die Voreinstellung
// zu ersetzen ließe ihn glauben, sein Format sei gespeichert
// (siehe domain.ValidateInvoiceNumberFormat).
func numberFormatOrDefault(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return domain.DefaultInvoiceNumberFormat, nil
	}
	if err := domain.ValidateInvoiceNumberFormat(format); err != nil {
		return "", err
	}
	return format, nil
}

// receiptFormatOrDefault prüft die Systematik des Belegnummernkreises.
func receiptFormatOrDefault(format string) (string, error) {
	if strings.TrimSpace(format) == "" {
		return domain.DefaultReceiptNumberFormat, nil
	}
	if err := domain.ValidateNumberFormat(format); err != nil {
		return "", err
	}
	return format, nil
}

func (r *settingsRepositoryGorm) UpdateCompanySettings(ctx context.Context, s *domain.CompanySettings) error {
	vatPeriod := s.VatPeriod
	if vatPeriod == "" {
		vatPeriod = "quarter"
	}
	taxationType := s.TaxationType
	if taxationType == "" {
		taxationType = "SOLL"
	}
	startMonth := s.FiscalYearStartMonth
	if startMonth <= 0 || startMonth > 12 {
		startMonth = 1
	}
	captureDays := s.ReceiptCaptureDays
	if captureDays <= 0 {
		captureDays = 10
	}
	graceDays := s.CommitGraceDays
	if graceDays < 0 {
		graceDays = 0
	}
	numberFormat, err := numberFormatOrDefault(s.InvoiceNumberFormat)
	if err != nil {
		return err
	}
	// Der Belegnummernkreis wird nach derselben Regel geprüft wie der
	// Rechnungsnummernkreis: ein untaugliches Format wird abgewiesen und nicht
	// stillschweigend durch die Voreinstellung ersetzt.
	receiptFormat, err := receiptFormatOrDefault(s.ReceiptNumberFormat)
	if err != nil {
		return err
	}
	// Eine Null im Feld ist ein leeres Feld und keine Abschaltung: wer keinen
	// Leistungsnachweis verlangen will, setzt die Grenze hoch. Andernfalls
	// schaltete ein Formular, das das Feld nicht kennt, die Regel stumm ab.
	//
	// Und deshalb wird die Grenze dann gar nicht geschrieben: ein gepflegter
	// Wert fiele sonst auf die Voreinstellung zurück — ein Formular, das ein
	// Feld nicht kennt, darf es nicht ändern.
	// Dasselbe gilt für die Mahnstufen.
	writeCheckThreshold := s.InvoiceCheckThreshold > 0
	writeLevels := len(s.DunningLevels) > 0
	levels, err := json.Marshal(domain.NormalizeDunningLevels(s.DunningLevels))
	if err != nil {
		return fmt.Errorf("die Mahnstufen ließen sich nicht speichern: %w", err)
	}

	kv := map[string]string{
		"company_name":            s.CompanyName,
		"legal_form":              s.LegalForm,
		"fiscal_year":             fmt.Sprintf("%d", s.FiscalYear),
		"fiscal_year_start_month": fmt.Sprintf("%d", startMonth),
		"tax_number":              s.TaxNumber,
		"vat_id":                  s.VatID,
		"tax_office":              s.TaxOffice,
		"iban":                    s.IBAN,
		"bic":                     s.BIC,
		"bank_name":               s.BankName,
		"street":                  s.Street,
		"zip_city":                s.ZipCity,
		"country":                 s.Country,
		"contact_name":            s.ContactName,
		"contact_phone":           s.ContactPhone,
		"contact_email":           s.ContactEmail,
		"invoice_number_format":   numberFormat,
		"receipt_number_format":   receiptFormat,
		"seat":                    s.Seat,
		"register_court":          s.RegisterCourt,
		"register_number":         s.RegisterNumber,
		"vat_period":              vatPeriod,
		"taxation_type":           taxationType,
		// Leer bleibt leer: die Anlegerstellung folgt dann aus der Rechtsform.
		// Ein hier eingesetzter Vorgabewert wäre eine Festlegung, die niemand
		// getroffen hat — und die die Ableitung stumm überschriebe.
		"investor_override": string(s.InvestorOverride),
		// Die Dauerfristverlängerung verschiebt jede Fälligkeit um einen Monat;
		// die Sondervorauszahlung wird im letzten Zeitraum des Jahres
		// angerechnet. Beide gehören zusammen und stehen deshalb nebeneinander.
		"permanent_extension":  strconv.FormatBool(s.PermanentExtension),
		"special_prepayment":   strconv.FormatInt(int64(s.SpecialPrepayment), 10),
		"receipt_capture_days": strconv.Itoa(captureDays),
		"commit_grace_days":    strconv.Itoa(graceDays),
	}
	// Der Prüfvermerk am Eingangsbeleg und die Mahnstufen: beides sind
	// Festlegungen des internen Kontrollsystems, die der Anwender trifft — und
	// die nur ändert, wer sie mitschickt.
	if writeCheckThreshold {
		kv["invoice_check_threshold"] = strconv.FormatInt(int64(s.InvoiceCheckThreshold), 10)
		// Der Tag, ab dem der Prüflauf den fehlenden Leistungsnachweis
		// beanstandet. Er wird einmal gesetzt und danach nicht mehr verschoben:
		// eine Festlegung des Kontrollsystems gilt ab dem Tag, an dem sie
		// getroffen wurde, und jede spätere Änderung der Grenze macht die
		// Belege davor nicht nachträglich mangelhaft. Ein ausdrücklich
		// mitgeschicktes Datum sticht — wer rückwirkend prüfen will, trägt es
		// ein.
		since := strings.TrimSpace(s.InvoiceCheckSince)
		if since == "" {
			// Ein fehlender Eintrag ist kein Fehler: er ist der Normalfall beim
			// ersten Speichern der Grenze, und genau dann wird der Tag gesetzt.
			var item domain.SettingItem
			if err := dbFrom(ctx, r.db).Where("key = ?", "invoice_check_since").
				First(&item).Error; err == nil {
				since = strings.TrimSpace(item.Value)
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if since == "" {
			since = time.Now().Format("2006-01-02")
		}
		kv["invoice_check_since"] = since
	}
	if writeLevels {
		kv["dunning_levels"] = string(levels)
	}

	for k, v := range kv {
		item := domain.SettingItem{
			Key:       k,
			Value:     v,
			UpdatedAt: time.Now().UTC(),
		}
		if err := dbFrom(ctx, r.db).Save(&item).Error; err != nil {
			return err
		}
	}

	return nil
}
