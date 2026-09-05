package domain

import (
	"context"
	"fmt"
	"time"
)

// FiscalYearStatus ist der Stand des Jahresabschlusses.
//
// Die vier Werte sind keine Abstufungen derselben Sache, sondern verschiedene
// Vorgänge mit verschiedenen Beteiligten: die Aufstellung obliegt der
// Geschäftsführung (§ 242, § 264 Abs. 1 HGB), die Feststellung den
// Gesellschaftern (§ 42a Abs. 2 GmbHG), die Offenlegung geschieht gegenüber
// dem Bundesanzeiger (§ 325 HGB). Sie in einem Feld „abgeschlossen ja/nein"
// zusammenzufassen hieße, den Beschluss der Gesellschafter zu unterschlagen –
// und genau an ihm hängt, ab wann nicht mehr gebucht werden darf.
type FiscalYearStatus string

const (
	// FiscalYearOpen ist das laufende Jahr: es darf gebucht werden.
	FiscalYearOpen FiscalYearStatus = "open"
	// FiscalYearPrepared ist aufgestellt: die Geschäftsführung hat den
	// Abschluss erstellt, die Gesellschafter haben ihn noch nicht beschlossen.
	FiscalYearPrepared FiscalYearStatus = "prepared"
	// FiscalYearAdopted ist festgestellt: ab hier nimmt das Jahr keine Buchung
	// mehr auf, weil der Abschluss beschlossen ist.
	FiscalYearAdopted FiscalYearStatus = "adopted"
	// FiscalYearDisclosed ist offengelegt.
	FiscalYearDisclosed FiscalYearStatus = "disclosed"
)

// rank ordnet die Stände in ihrer zeitlichen Folge. Damit lässt sich ein
// übersprungener Schritt erkennen — festgestellt werden kann nur, was
// aufgestellt wurde.
func (s FiscalYearStatus) rank() int {
	switch s {
	case FiscalYearOpen:
		return 0
	case FiscalYearPrepared:
		return 1
	case FiscalYearAdopted:
		return 2
	case FiscalYearDisclosed:
		return 3
	}
	return -1
}

// Valid meldet, ob der Stand einer der bekannten vier ist.
func (s FiscalYearStatus) Valid() bool { return s.rank() >= 0 }

// Label benennt den Stand für die Oberfläche.
func (s FiscalYearStatus) Label() string {
	switch s {
	case FiscalYearOpen:
		return "Offen"
	case FiscalYearPrepared:
		return "Aufgestellt"
	case FiscalYearAdopted:
		return "Festgestellt"
	case FiscalYearDisclosed:
		return "Offengelegt"
	}
	return string(s)
}

// AllFiscalYearStatuses liefert die Stände in der Reihenfolge, in der sie
// durchlaufen werden.
func AllFiscalYearStatuses() []FiscalYearStatus {
	return []FiscalYearStatus{FiscalYearOpen, FiscalYearPrepared, FiscalYearAdopted, FiscalYearDisclosed}
}

// FiscalYear ist das Geschäftsjahr als eigene Entität.
//
// Bisher war es nur eine Zahl an jeder Buchung, abgeleitet aus dem
// Buchungsdatum. Damit ließ sich nicht sagen, wann ein Jahr anfängt (ein
// Rumpfgeschäftsjahr beginnt mit der Beurkundung, nicht am 1. Januar), wann es
// endet und ob sein Abschluss festgestellt ist. Alles drei sind Tatsachen über
// das Jahr, nicht über eine Buchung darin.
type FiscalYear struct {
	// Year ist der Schlüssel und identisch mit JournalEntry.FiscalYear.
	Year int `gorm:"primaryKey;autoIncrement:false" json:"year"`

	StartDate string `gorm:"size:10;not null" json:"startDate"` // YYYY-MM-DD
	EndDate   string `gorm:"size:10;not null" json:"endDate"`   // YYYY-MM-DD
	// IsShort kennzeichnet das Rumpfgeschäftsjahr (§ 8b EStDV): kürzer als
	// zwölf Monate, weil das Unternehmen im Lauf des Jahres entstanden ist oder
	// das Geschäftsjahr umgestellt wurde.
	IsShort bool `gorm:"default:false" json:"isShort"`

	Status      FiscalYearStatus `gorm:"size:20;not null;default:'open'" json:"status"`
	PreparedOn  string           `gorm:"size:10" json:"preparedOn,omitempty"`
	AdoptedOn   string           `gorm:"size:10" json:"adoptedOn,omitempty"`
	DisclosedOn string           `gorm:"size:10" json:"disclosedOn,omitempty"`
	// AdoptionNote hält den Beschlussbezug fest: welcher
	// Gesellschafterbeschluss den Abschluss festgestellt hat.
	AdoptionNote string `gorm:"size:500" json:"adoptionNote,omitempty"`

	// AverageEmployees ist die durchschnittliche Zahl der Arbeitnehmer des
	// Geschäftsjahres — das dritte Merkmal des § 267 Abs. 1 HGB neben
	// Bilanzsumme und Umsatzerlösen.
	//
	// Es lässt sich aus der Buchführung nicht ableiten: § 267 Abs. 5 HGB
	// bestimmt den Durchschnitt aus vier Quartalsstichtagen, und wer an ihnen
	// beschäftigt war, steht in keinem Konto. Deshalb ist es eine Angabe und
	// keine Rechnung; null heißt „noch nicht erfasst" und führt dazu, dass
	// dieses Merkmal für die kleinste Klasse spricht.
	AverageEmployees int `gorm:"default:0" json:"averageEmployees"`

	// PriorYearRevenue ist der Gesamtumsatz des vorangegangenen Kalenderjahres.
	//
	// Er entscheidet über die Übergangsfrist des § 27 Abs. 38 Nr. 2 UStG: bis
	// 800.000 € darf im Jahr 2027 noch eine sonstige Rechnung ausgestellt
	// werden. Vorbelegt aus der Gewinn- und Verlustrechnung des Vorjahres,
	// überschreibbar — der Gesamtumsatz des § 19 Abs. 3 UStG ist nicht dasselbe
	// wie die Umsatzerlöse des § 275 HGB, und die Differenz kennt nur der
	// Steuerpflichtige. Null heißt „nicht erfasst".
	PriorYearRevenue Cents `gorm:"default:0" json:"priorYearRevenue"`

	// CarriedForwardAt ist der Zeitpunkt des letzten Saldenvortrags in dieses
	// Jahr. Nicht ob, sondern wann: ein Vortrag kann durch spätere Buchungen im
	// Vorjahr überholt werden, und dann ist der Zeitpunkt die einzige Spur, an
	// der sich das ablesen lässt.
	CarriedForwardAt *time.Time `json:"carriedForwardAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// NewFiscalYear baut ein Geschäftsjahr für einen Zeitraum und leitet daraus ab,
// ob es ein Rumpfgeschäftsjahr ist.
func NewFiscalYear(year int, start, end string) *FiscalYear {
	fy := &FiscalYear{
		Year:      year,
		StartDate: start,
		EndDate:   end,
		Status:    FiscalYearOpen,
	}
	fy.IsShort = fy.isShorterThanFullYear()
	return fy
}

// isShorterThanFullYear meldet, ob der Zeitraum keine vollen zwölf Monate
// umfasst. Ein volles Jahr endet am Tag vor dem Jahrestag seines Beginns.
func (f *FiscalYear) isShorterThanFullYear() bool {
	start, end, err := f.period()
	if err != nil {
		return false
	}
	return end.Before(start.AddDate(0, 12, 0).AddDate(0, 0, -1))
}

func (f *FiscalYear) period() (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", f.StartDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("Beginn des Geschäftsjahres %d ist kein Datum (erwartet JJJJ-MM-TT): %q", f.Year, f.StartDate)
	}
	end, err := time.Parse("2006-01-02", f.EndDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("Ende des Geschäftsjahres %d ist kein Datum (erwartet JJJJ-MM-TT): %q", f.Year, f.EndDate)
	}
	return start, end, nil
}

// Contains meldet, ob ein Buchungsdatum in dieses Geschäftsjahr fällt.
func (f *FiscalYear) Contains(date string) bool {
	return date >= f.StartDate && date <= f.EndDate
}

// IsAdopted meldet, ob der Abschluss festgestellt ist. Die Offenlegung setzt
// die Feststellung voraus und hebt sie nicht auf, zählt hier also mit.
func (f *FiscalYear) IsAdopted() bool {
	return f.Status == FiscalYearAdopted || f.Status == FiscalYearDisclosed
}

// Validate prüft die Invarianten des Geschäftsjahres.
//
// Die harte Grenze ist die Länge: § 240 Abs. 2 Satz 2 HGB lässt für den
// Zeitraum eines Geschäftsjahres höchstens zwölf Monate zu. Ein längerer
// Zeitraum wäre kein zulässiges Geschäftsjahr, und die Buchungen darin ließen
// sich keinem Abschluss zuordnen.
func (f *FiscalYear) Validate() error {
	if f.Year <= 0 {
		return fmt.Errorf("das Geschäftsjahr braucht eine Jahreszahl")
	}
	start, end, err := f.period()
	if err != nil {
		return err
	}
	if end.Before(start) {
		return fmt.Errorf(
			"das Geschäftsjahr %d endet am %s und damit vor seinem Beginn am %s",
			f.Year, f.EndDate, f.StartDate)
	}
	// Zwölf Monate ab Beginn ist der erste Tag, der nicht mehr dazugehört.
	if !end.Before(start.AddDate(0, 12, 0)) {
		return fmt.Errorf(
			"das Geschäftsjahr %d umfasst mit %s bis %s mehr als zwölf Monate; "+
				"nach § 240 Abs. 2 Satz 2 HGB darf ein Geschäftsjahr zwölf Monate nicht überschreiten",
			f.Year, f.StartDate, f.EndDate)
	}
	if !f.Status.Valid() {
		return fmt.Errorf("unbekannter Abschlussstand %q", f.Status)
	}
	// Zu jedem erreichten Schritt gehört sein Datum. Ein Stand ohne Datum wäre
	// eine Behauptung ohne Beleg — und § 42a Abs. 2 GmbHG knüpft Fristen genau
	// an diese Tage.
	if f.Status.rank() >= FiscalYearPrepared.rank() && f.PreparedOn == "" {
		return fmt.Errorf("zum Stand %q fehlt das Datum der Aufstellung", f.Status.Label())
	}
	if f.Status.rank() >= FiscalYearAdopted.rank() && f.AdoptedOn == "" {
		return fmt.Errorf("zum Stand %q fehlt das Datum der Feststellung", f.Status.Label())
	}
	if f.Status == FiscalYearDisclosed && f.DisclosedOn == "" {
		return fmt.Errorf("zum Stand %q fehlt das Datum der Offenlegung", f.Status.Label())
	}
	return nil
}

// FiscalYearRepository persistiert die Geschäftsjahre.
type FiscalYearRepository interface {
	FindAll(ctx context.Context) ([]FiscalYear, error)
	// FindByYear liefert nil, wenn das Jahr noch keine Entität hat. Das ist der
	// Normalfall in einer gewachsenen Datenbank und kein Fehler.
	FindByYear(ctx context.Context, year int) (*FiscalYear, error)
	Save(ctx context.Context, fy *FiscalYear) error
}
