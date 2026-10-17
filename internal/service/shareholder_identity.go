package service

import (
	"strings"
	"unicode"

	"github.com/buchfink/buchfink/internal/domain"
)

// companyRole sagt, in welcher Eigenschaft ein Kontakt der GmbH
// gegenübersteht. Maßgeblich sind Gesellschafterliste und Geschäftsführung in
// den Unternehmensdaten; der Kontakt wird über seinen Namen zugeordnet.
type companyRole struct {
	shareholder bool
	director    bool
	// stakePermille ist der Anteil am Stammkapital in Promille, null wenn das
	// Stammkapital fehlt.
	stakePermille int
	// listedName ist der Name, unter dem er in der Liste steht.
	listedName string
}

// companyRoleOf ordnet einen Kontakt der Gesellschafterliste und der
// Geschäftsführung zu.
func companyRoleOf(cfg *domain.CompanySettings, c *domain.Contact) companyRole {
	var role companyRole
	if cfg == nil || c == nil {
		return role
	}
	names := []string{c.Name, c.Company, c.LegalName()}
	var capital domain.Cents
	for _, sh := range cfg.Shareholders {
		capital += sh.ShareCapital
	}
	if cfg.ShareCapital > 0 {
		capital = cfg.ShareCapital
	}
	for _, sh := range cfg.Shareholders {
		if !anyNameMatches(names, sh.Name) {
			continue
		}
		role.shareholder = true
		role.listedName = sh.Name
		if capital > 0 {
			role.stakePermille += int(int64(sh.ShareCapital) * 1000 / int64(capital))
		}
	}
	for _, director := range splitDirectors(cfg.ManagingDirectors) {
		if anyNameMatches(names, director) {
			role.director = true
			if role.listedName == "" {
				role.listedName = director
			}
		}
	}
	return role
}

// splitDirectors trennt die Geschäftsführung in einzelne Namen.
func splitDirectors(value string) []string {
	value = strings.NewReplacer("\n", ",", ";", ",", " und ", ",", "&", ",").Replace(value)
	out := make([]string, 0, 2)
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func anyNameMatches(candidates []string, listed string) bool {
	want := nameTokens(listed)
	if len(want) == 0 {
		return false
	}
	for _, candidate := range candidates {
		if namesMatch(nameTokens(candidate), want) {
			return true
		}
	}
	return false
}

// namesMatch vergleicht zwei Namen wortweise. Gleich sind sie, wenn sie
// übereinstimmen oder der kürzere aus mindestens zwei Wörtern im längeren
// steht, etwa „Mara Beispiel" in „Dr. Mara Beispiel".
func namesMatch(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	if strings.Join(a, " ") == strings.Join(b, " ") {
		return true
	}
	short, long := a, b
	if len(short) > len(long) {
		short, long = long, short
	}
	if len(short) < 2 {
		return false
	}
	for i := 0; i+len(short) <= len(long); i++ {
		if strings.Join(long[i:i+len(short)], " ") == strings.Join(short, " ") {
			return true
		}
	}
	return false
}

func nameTokens(name string) []string {
	return strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
