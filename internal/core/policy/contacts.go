package policy

import (
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// DuplicateContact sucht vor dem Anlegen einen vorhandenen Kontakt desselben
// Typs: gleiche USt-IdNr., gleiche IBAN oder gleicher Name mit gleicher
// Postleitzahl. Match nennt das Merkmal ("ust_id", "iban", "name_plz").
func DuplicateContact(existing []domain.Contact, c domain.Contact) (found *domain.Contact, match string) {
	vat, iban := compact(c.VatID), compact(c.IBAN)
	names := map[string]bool{normalizeName(c.Name): true, normalizeName(c.Company): true}
	delete(names, "")
	plz := compact(c.PostalCode)
	for i := range existing {
		e := &existing[i]
		if e.Type != c.Type {
			continue
		}
		switch {
		case vat != "" && compact(e.VatID) == vat:
			return e, "ust_id"
		case iban != "" && compact(e.IBAN) == iban:
			return e, "iban"
		case (names[normalizeName(e.Name)] || names[normalizeName(e.Company)]) && compact(e.PostalCode) == plz:
			return e, "name_plz"
		}
	}
	return nil, ""
}

func compact(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), ""))
}
