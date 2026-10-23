package domain

import (
	"context"
	"time"
)

// MasterDataProposal ist der Vorschlag eines Agenten, Unternehmensdaten,
// Gründung oder Bankkonten zu ändern. Er ändert nichts: übernehmen kann ihn
// nur der Mensch in der App, und nur so, wie die Vorschau ihn gezeigt hat.
type MasterDataProposal struct {
	ID     uint                     `gorm:"primaryKey" json:"id"`
	Status MasterDataProposalStatus `gorm:"size:12;not null;index" json:"status"`
	// Reason sagt, woher die Angaben stammen, etwa „Schreiben des Finanzamts
	// vom 02.09.2026".
	Reason    string `gorm:"size:1000;not null" json:"reason"`
	ReceiptID *uint  `json:"receiptId,omitempty"`
	// Changes ist MasterDataChanges als JSON.
	Changes   string    `gorm:"type:text;not null" json:"-"`
	Channel   string    `gorm:"size:60" json:"channel"`
	Actor     string    `gorm:"size:200" json:"actor"`
	CreatedAt time.Time `json:"createdAt"`
	// DecidedAt, DecidedBy und DecisionNote halten die Entscheidung des
	// Menschen fest; die Notiz sagt dem Agenten, warum abgelehnt wurde.
	DecidedAt    *time.Time `json:"decidedAt,omitempty"`
	DecidedBy    string     `gorm:"size:200" json:"decidedBy,omitempty"`
	DecisionNote string     `gorm:"size:1000" json:"decisionNote,omitempty"`
}

type MasterDataProposalStatus string

const (
	ProposalOpen     MasterDataProposalStatus = "offen"
	ProposalAccepted MasterDataProposalStatus = "uebernommen"
	ProposalRejected MasterDataProposalStatus = "abgelehnt"
)

// MasterDataChanges sind die vorgeschlagenen Änderungen. Was fehlt, bleibt,
// wie es ist.
type MasterDataChanges struct {
	Company    *CompanyChanges     `json:"company,omitempty"`
	Foundation *FoundationProposal `json:"foundation,omitempty"`
	// BankAccounts legt Konten an oder ändert sie, erkannt an der IBAN.
	// IsInvoiceAccount macht ein Konto zum Konto für Rechnungen.
	BankAccounts []BankAccount `json:"bankAccounts,omitempty"`
}

// Empty meldet einen Vorschlag ohne Inhalt.
func (c MasterDataChanges) Empty() bool {
	return (c.Company == nil || c.Company.Empty()) && c.Foundation == nil && len(c.BankAccounts) == 0
}

// CompanyChanges sind die Unternehmensdaten, die ein Agent vorschlagen kann;
// nil heißt: unverändert. Rechtsform, Geschäftsjahr, Nummernkreise und
// Prüfregeln stehen nicht darin, sie ändert nur der Mensch selbst.
type CompanyChanges struct {
	CompanyName           *string               `json:"companyName,omitempty"`
	ManagingDirectors     *string               `json:"managingDirectors,omitempty"`
	SupervisoryBoardChair *string               `json:"supervisoryBoardChair,omitempty"`
	Street                *string               `json:"street,omitempty"`
	HouseNumber           *string               `json:"houseNumber,omitempty"`
	AddressAddition       *string               `json:"addressAddition,omitempty"`
	PostalCode            *string               `json:"postalCode,omitempty"`
	City                  *string               `json:"city,omitempty"`
	CountryCode           *string               `json:"countryCode,omitempty"`
	ContactName           *string               `json:"contactName,omitempty"`
	ContactPhone          *string               `json:"contactPhone,omitempty"`
	ContactEmail          *string               `json:"contactEmail,omitempty"`
	Notary                *string               `json:"notary,omitempty"`
	DeedNumber            *string               `json:"deedNumber,omitempty"`
	Seat                  *string               `json:"seat,omitempty"`
	RegisterCourt         *string               `json:"registerCourt,omitempty"`
	RegisterNumber        *string               `json:"registerNumber,omitempty"`
	RegisteredOn          *string               `json:"registeredOn,omitempty"`
	TaxNumber             *string               `json:"taxNumber,omitempty"`
	TaxOffice             *string               `json:"taxOffice,omitempty"`
	FederalState          *string               `json:"federalState,omitempty"`
	VatID                 *string               `json:"vatId,omitempty"`
	VatPeriod             *string               `json:"vatPeriod,omitempty"`
	TaxationType          *string               `json:"taxationType,omitempty"`
	PermanentExtension    *bool                 `json:"permanentExtension,omitempty"`
	SpecialPrepayment     *Cents                `json:"specialPrepayment,omitempty"`
	TaxAdvisor            *TaxAdvisor           `json:"taxAdvisor,omitempty"`
	ShareCapital          *Cents                `json:"shareCapital,omitempty"`
	Shareholders          *[]CompanyShareholder `json:"shareholders,omitempty"`
}

// Empty meldet, dass kein Feld vorgeschlagen ist.
func (c CompanyChanges) Empty() bool { return c == (CompanyChanges{}) }

// ApplyTo legt die vorgeschlagenen Felder über s.
func (c CompanyChanges) ApplyTo(s *CompanySettings) {
	setString := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	setString(&s.CompanyName, c.CompanyName)
	setString(&s.ManagingDirectors, c.ManagingDirectors)
	setString(&s.SupervisoryBoardChair, c.SupervisoryBoardChair)
	setString(&s.Street, c.Street)
	setString(&s.HouseNumber, c.HouseNumber)
	setString(&s.AddressAddition, c.AddressAddition)
	setString(&s.PostalCode, c.PostalCode)
	setString(&s.City, c.City)
	setString(&s.CountryCode, c.CountryCode)
	setString(&s.ContactName, c.ContactName)
	setString(&s.ContactPhone, c.ContactPhone)
	setString(&s.ContactEmail, c.ContactEmail)
	setString(&s.Notary, c.Notary)
	setString(&s.DeedNumber, c.DeedNumber)
	setString(&s.Seat, c.Seat)
	setString(&s.RegisterCourt, c.RegisterCourt)
	setString(&s.RegisterNumber, c.RegisterNumber)
	setString(&s.RegisteredOn, c.RegisteredOn)
	setString(&s.TaxNumber, c.TaxNumber)
	setString(&s.TaxOffice, c.TaxOffice)
	setString(&s.FederalState, c.FederalState)
	setString(&s.VatID, c.VatID)
	setString(&s.VatPeriod, c.VatPeriod)
	setString(&s.TaxationType, c.TaxationType)
	if c.PermanentExtension != nil {
		s.PermanentExtension = *c.PermanentExtension
	}
	if c.SpecialPrepayment != nil {
		s.SpecialPrepayment = *c.SpecialPrepayment
	}
	if c.TaxAdvisor != nil {
		s.TaxAdvisor = *c.TaxAdvisor
	}
	if c.ShareCapital != nil {
		s.ShareCapital = *c.ShareCapital
	}
	if c.Shareholders != nil {
		s.Shareholders = append([]CompanyShareholder{}, *c.Shareholders...)
	}
}

// FoundationProposal ist die Gründung laut Gesellschaftsvertrag, wie sie die
// Erfassung der Gründung in der App aufnimmt.
type FoundationProposal struct {
	NotarizedOn       string                          `json:"notarizedOn"`
	ShareCapital      Cents                           `json:"shareCapital"`
	FoundationCostCap Cents                           `json:"foundationCostCap"`
	Shareholders      []FoundationShareholderProposal `json:"shareholders"`
}

type FoundationShareholderProposal struct {
	Name         string           `json:"name"`
	ShareCapital Cents            `json:"shareCapital"`
	PaidIn       Cents            `json:"paidIn"`
	Uncalled     Cents            `json:"uncalled"`
	Kind         ContributionKind `json:"kind"`
}

// MasterDataProposalRepository speichert die Vorschläge.
type MasterDataProposalRepository interface {
	Create(ctx context.Context, p *MasterDataProposal) error
	Find(ctx context.Context, id uint) (*MasterDataProposal, error)
	// FindByStatus liefert die Vorschläge eines Stands, den ältesten zuerst.
	FindByStatus(ctx context.Context, status MasterDataProposalStatus) ([]MasterDataProposal, error)
	// FindRecent liefert die jüngsten Vorschläge, den neuesten zuerst.
	FindRecent(ctx context.Context, limit int) ([]MasterDataProposal, error)
	Save(ctx context.Context, p *MasterDataProposal) error
}
