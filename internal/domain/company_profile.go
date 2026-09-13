package domain

import (
	"fmt"
	"strings"
	"time"
)

// CompanyShareholder ist ein Gesellschafter mit dem Nennbetrag seiner
// Geschäftsanteile, wie er in der aktuellen Gesellschafterliste steht.
type CompanyShareholder struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	Position int  `gorm:"not null" json:"-"`
	// Name ist personenbezogen und liegt deshalb verschlüsselt.
	Name         string `gorm:"type:text;not null;serializer:encrypted" json:"name"`
	ShareCapital Cents  `gorm:"not null" json:"shareCapital"`
}

// DefaultCountryCode ist das Land, in dem Buchfink bucht.
const DefaultCountryCode = "DE"

// Country ist ein Land der Auswahl in den Stammdaten.
type Country struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// countries sind die Staaten der EU und des EWR, die Schweiz und das
// Vereinigte Königreich, Deutschland zuerst.
var countries = []Country{
	{"DE", "Deutschland"}, {"AT", "Österreich"}, {"CH", "Schweiz"}, {"LI", "Liechtenstein"},
	{"BE", "Belgien"}, {"BG", "Bulgarien"}, {"DK", "Dänemark"}, {"EE", "Estland"},
	{"FI", "Finnland"}, {"FR", "Frankreich"}, {"GR", "Griechenland"}, {"IE", "Irland"},
	{"IS", "Island"}, {"IT", "Italien"}, {"HR", "Kroatien"}, {"LV", "Lettland"},
	{"LT", "Litauen"}, {"LU", "Luxemburg"}, {"MT", "Malta"}, {"NL", "Niederlande"},
	{"NO", "Norwegen"}, {"PL", "Polen"}, {"PT", "Portugal"}, {"RO", "Rumänien"},
	{"SE", "Schweden"}, {"SK", "Slowakei"}, {"SI", "Slowenien"}, {"ES", "Spanien"},
	{"CZ", "Tschechien"}, {"HU", "Ungarn"}, {"GB", "Vereinigtes Königreich"}, {"CY", "Zypern"},
}

// Countries returns the selectable countries.
func Countries() []Country {
	out := make([]Country, len(countries))
	copy(out, countries)
	return out
}

// CountryName returns the German name of a country code, or the code itself.
func CountryName(code string) string {
	for _, c := range countries {
		if c.Code == code {
			return c.Name
		}
	}
	return code
}

// CountryCodeFromName liest eine frei eingetragene Landesangabe als Code. Leer
// heißt Deutschland, Unbekanntes ergibt "".
func CountryCodeFromName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return DefaultCountryCode
	}
	for _, c := range countries {
		if strings.EqualFold(c.Code, trimmed) || strings.EqualFold(c.Name, trimmed) {
			return c.Code
		}
	}
	switch strings.ToLower(trimmed) {
	case "germany", "bundesrepublik deutschland", "brd", "d":
		return "DE"
	case "austria", "a":
		return "AT"
	case "switzerland", "ch-schweiz":
		return "CH"
	}
	return ""
}

// ResolvedCountryCode is the country code with Germany as the default.
func (s *CompanySettings) ResolvedCountryCode() string {
	if code := strings.ToUpper(strings.TrimSpace(s.CountryCode)); code != "" {
		return code
	}
	return DefaultCountryCode
}

// PostalLine returns "80331 München".
func (s *CompanySettings) PostalLine() string {
	return strings.TrimSpace(strings.TrimSpace(s.PostalCode) + " " + strings.TrimSpace(s.City))
}

// AddressLines returns the address as printed: street, addition, postal line
// and — outside Germany — the country.
func (s *CompanySettings) AddressLines() []string {
	lines := []string{}
	for _, line := range []string{s.Street, s.AddressAddition, s.PostalLine()} {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	if code := s.ResolvedCountryCode(); code != DefaultCountryCode {
		lines = append(lines, CountryName(code))
	}
	return lines
}

// HasCompleteAddress reports whether street, postal code and city are known —
// the parts § 14 Abs. 4 Nr. 1 UStG asks for.
func (s *CompanySettings) HasCompleteAddress() bool {
	return strings.TrimSpace(s.Street) != "" &&
		strings.TrimSpace(s.PostalCode) != "" &&
		strings.TrimSpace(s.City) != ""
}

// SplitLegacyPostalLine zerlegt die frühere Zeile „PLZ und Ort".
//
// Erkannt werden eine führende Postleitzahl aus Ziffern, auch mit
// Länderkennung („D-80331 München"), und die niederländische Form
// („1012 AB Amsterdam"). Was sich nicht sicher trennen lässt, steht danach
// vollständig im Ort und bleibt zur Prüfung sichtbar.
func SplitLegacyPostalLine(line string) (postalCode, city string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", strings.TrimSpace(line)
	}
	code := fields[0]
	if i := strings.IndexByte(code, '-'); i > 0 && i <= 2 && isLetters(code[:i]) {
		code = code[i+1:]
	}
	if !isDigitString(code) {
		return "", strings.Join(fields, " ")
	}
	rest := fields[1:]
	if len(code) == 4 && len(rest) > 1 && len(rest[0]) == 2 && isLetters(rest[0]) && strings.ToUpper(rest[0]) == rest[0] {
		return code + " " + rest[0], strings.Join(rest[1:], " ")
	}
	return code, strings.Join(rest, " ")
}

// ValidateProfile prüft Anschrift, Gründungsdaten und Gesellschafterliste.
func (s *CompanySettings) ValidateProfile() error {
	code := strings.ToUpper(strings.TrimSpace(s.CountryCode))
	if code != "" && (len(code) != 2 || !isLetters(code)) {
		return fmt.Errorf("das Land ist als zweistelliger Ländercode anzugeben, etwa DE")
	}
	postalCode := strings.TrimSpace(s.PostalCode)
	if postalCode != "" && s.ResolvedCountryCode() == DefaultCountryCode && (len(postalCode) != 5 || !isDigitString(postalCode)) {
		return fmt.Errorf("eine deutsche Postleitzahl hat fünf Ziffern, eingetragen ist %q", postalCode)
	}
	for _, date := range []struct{ label, value string }{
		{"Gründungsdatum", s.FoundedOn},
		{"Tag der Eintragung", s.RegisteredOn},
	} {
		if date.value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", date.value); err != nil {
			return fmt.Errorf("%s: das Datum ist unvollständig (erwartet JJJJ-MM-TT)", date.label)
		}
	}
	if s.FoundedOn != "" && s.RegisteredOn != "" && s.RegisteredOn < s.FoundedOn {
		return fmt.Errorf("die Eintragung (%s) kann nicht vor der Gründung (%s) liegen",
			GermanDate(s.RegisteredOn), GermanDate(s.FoundedOn))
	}
	if s.ShareCapital < 0 {
		return fmt.Errorf("das Stammkapital kann nicht negativ sein")
	}
	var subscribed Cents
	for i, sh := range s.Shareholders {
		if strings.TrimSpace(sh.Name) == "" {
			return fmt.Errorf("der %d. Gesellschafter hat keinen Namen", i+1)
		}
		if sh.ShareCapital <= 0 {
			return fmt.Errorf("%s: der Nennbetrag der Geschäftsanteile muss größer als null sein", sh.Name)
		}
		subscribed += sh.ShareCapital
	}
	if len(s.Shareholders) > 0 && s.ShareCapital > 0 && subscribed != s.ShareCapital {
		return fmt.Errorf(
			"die Geschäftsanteile der Gesellschafter ergeben %s €, das Stammkapital beträgt %s €. "+
				"Beides muss übereinstimmen (§ 5 Abs. 3 Satz 2 GmbHG)",
			subscribed, s.ShareCapital)
	}
	return nil
}

func isDigitString(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isLetters(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}
