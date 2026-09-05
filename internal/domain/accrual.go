package domain

import (
	"context"
	"fmt"
	"time"
)

// AccrualKind unterscheidet die drei Fälle des § 250 HGB.
type AccrualKind string

const (
	// AccrualActive ist der aktive Rechnungsabgrenzungsposten (§ 250 Abs. 1
	// HGB): eine Ausgabe vor dem Stichtag, die Aufwand für eine bestimmte Zeit
	// danach ist. Die Versicherungsprämie für das kommende Jahr ist der
	// Regelfall.
	AccrualActive AccrualKind = "active"
	// AccrualPassive ist der passive Posten (§ 250 Abs. 2 HGB): eine Einnahme
	// vor dem Stichtag, die Ertrag für eine Zeit danach ist.
	AccrualPassive AccrualKind = "passive"
	// AccrualDisagio ist das Wahlrecht des § 250 Abs. 3 HGB: der Unterschied
	// zwischen Erfüllungs- und Ausgabebetrag einer Verbindlichkeit darf aktiviert
	// und über die Laufzeit verteilt werden. Es ist eine eigene Art und keine
	// Spielart des aktiven Postens, weil die Verteilung an der Laufzeit des
	// Darlehens hängt und der Aufwand Zinsaufwand ist, nicht der ursprüngliche
	// Aufwand.
	AccrualDisagio AccrualKind = "disagio"
)

// Label benennt die Art für die Oberfläche.
func (k AccrualKind) Label() string {
	switch k {
	case AccrualActive:
		return "Ausgabe für spätere Jahre (aktive Rechnungsabgrenzung)"
	case AccrualPassive:
		return "Einnahme für spätere Jahre (passive Rechnungsabgrenzung)"
	case AccrualDisagio:
		return "Damnum/Disagio aus einem Darlehen"
	}
	return string(k)
}

// Valid meldet, ob die Art eine der bekannten ist.
func (k AccrualKind) Valid() bool {
	switch k {
	case AccrualActive, AccrualPassive, AccrualDisagio:
		return true
	}
	return false
}

// AllAccrualKinds liefert die Arten in fester Reihenfolge.
func AllAccrualKinds() []AccrualKind {
	return []AccrualKind{AccrualActive, AccrualPassive, AccrualDisagio}
}

// BalanceAccount ist das Bilanzkonto der Art: 1900 für den aktiven Posten und
// das Disagio, 3900 für den passiven.
func (k AccrualKind) BalanceAccount() string {
	if k == AccrualPassive {
		return AccountPassiveRAP
	}
	return AccountAktiveRAP
}

// AccrualMethod ist das Verteilungsverfahren.
//
// Es ist eine Einstellung für den ganzen Mandanten und keine Angabe je Posten:
// § 252 Abs. 1 Nr. 6 HGB verlangt, dass die Bewertungsmethoden beibehalten
// werden. Zwei Abgrenzungen desselben Jahres nach verschiedenen Verfahren zu
// verteilen wäre genau das nicht.
type AccrualMethod string

const (
	// AccrualMonthly verteilt nach Zwölfteln. Der angefangene Monat zählt voll;
	// das ist die in der Praxis übliche Vereinfachung und für den Regelfall
	// eines Vertrags, der zum Monatsersten beginnt, exakt.
	AccrualMonthly AccrualMethod = "monthly"
	// AccrualDaily verteilt taggenau nach der Zahl der Kalendertage.
	AccrualDaily AccrualMethod = "daily"
)

// Valid meldet, ob das Verfahren eines der bekannten ist.
func (m AccrualMethod) Valid() bool {
	return m == AccrualMonthly || m == AccrualDaily
}

// Label benennt das Verfahren für die Oberfläche.
func (m AccrualMethod) Label() string {
	if m == AccrualDaily {
		return "taggenau"
	}
	return "monatsgenau (Zwölftel)"
}

// AccrualReleaseSchedule ist der Takt, in dem der abgegrenzte Betrag im
// Folgejahr wieder aufgelöst wird.
//
// Er ist etwas anderes als die Methode: die Methode entscheidet, *wie* der
// Anteil gerechnet wird (nach Zwölfteln oder nach Tagen), der Takt, in *wie
// vielen Buchungen* er zurückkommt. Beides zusammenzufassen ginge nicht — wer
// taggenau rechnet, will deswegen noch keine 365 Auflösungsbuchungen.
type AccrualReleaseSchedule string

const (
	// AccrualReleaseYearly löst je Geschäftsjahr einmal auf, am ersten Tag. Für
	// Bilanz und GuV des Jahres genügt das, und es hält die Zahl der
	// Abschlussbuchungen klein.
	AccrualReleaseYearly AccrualReleaseSchedule = "yearly"
	// AccrualReleaseMonthly löst monatlich auf, jeweils am Monatsersten. Wer
	// unterjährig auswertet — BWA, Zwischenabschluss —, braucht das: sonst
	// trägt der Januar den ganzen Vorjahresaufwand, und jeder Monatsvergleich
	// ist verzerrt.
	AccrualReleaseMonthly AccrualReleaseSchedule = "monthly"
)

// Valid meldet, ob der Takt einer der bekannten ist.
func (s AccrualReleaseSchedule) Valid() bool {
	return s == AccrualReleaseYearly || s == AccrualReleaseMonthly
}

// Label benennt den Takt für die Oberfläche.
func (s AccrualReleaseSchedule) Label() string {
	if s == AccrualReleaseMonthly {
		return "monatlich"
	}
	return "einmal je Geschäftsjahr"
}

// SettingAccrualMethod und SettingAccrualThreshold sind die Schlüssel der
// beiden Einstellungen, die die Abgrenzung steuert.
const (
	SettingAccrualMethod    = "accrual_method"
	SettingAccrualThreshold = "accrual_threshold"
	// SettingAccrualRelease ist der Auflösungstakt (yearly oder monthly).
	SettingAccrualRelease = "accrual_release"
	// SettingTradeTaxRate ist der Gewerbesteuer-Hebesatz der Gemeinde in
	// Prozent (400 = 400 %).
	SettingTradeTaxRate = "trade_tax_rate"
)

// DefaultAccrualThreshold ist die Vorschlagsschwelle von 800 Euro.
//
// Sie stammt aus § 5 Abs. 5 Satz 2 EStG, der die Bildung eines aktiven Postens
// bei geringwertigen Beträgen zum Wahlrecht macht — und sie ist ausdrücklich
// nur eine Schwelle für den Vorschlag. Das Handelsrecht kennt keine Grenze:
// § 250 Abs. 1 HGB verlangt die Abgrenzung ohne Rücksicht auf die Höhe.
const DefaultAccrualThreshold Cents = 80000

// Accrual ist ein Rechnungsabgrenzungsposten mit seinem Auflösungsplan.
type Accrual struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// FiscalYear ist das Jahr, in dem der Posten gebildet wurde.
	FiscalYear int         `gorm:"index;not null" json:"fiscalYear"`
	Kind       AccrualKind `gorm:"size:20;not null;index" json:"kind"`

	// SourceEntryID verweist auf die Buchung, aus der der Posten entstanden ist
	// — die Rechnung, die den ganzen Betrag getragen hat. Optional: eine
	// Abgrenzung darf auch von Hand entstehen.
	SourceEntryID *uint  `gorm:"index" json:"sourceEntryId,omitempty"`
	Text          string `gorm:"size:255;not null;serializer:encrypted" json:"text"`

	// TotalAmount ist der Gesamtbetrag der Leistung, DeferredAmount der Teil,
	// der nach dem Stichtag liegt und damit abgegrenzt wird. Beide werden
	// gespeichert, weil der Bericht die Rechnung zeigen muss und nicht nur ihr
	// Ergebnis.
	TotalAmount    Cents  `gorm:"not null" json:"totalAmount"`
	DeferredAmount Cents  `gorm:"not null" json:"deferredAmount"`
	StartDate      string `gorm:"size:10;not null" json:"startDate"`
	EndDate        string `gorm:"size:10;not null" json:"endDate"`
	// CutoffDate ist der Bilanzstichtag, zu dem abgegrenzt wurde.
	CutoffDate string `gorm:"size:10;not null" json:"cutoffDate"`

	// Account ist das Aufwands- oder Ertragskonto, das der Posten entlastet und
	// im Folgejahr wieder belastet. Beim Disagio ist es das Zinsaufwandskonto.
	Account string        `gorm:"size:10;not null" json:"account"`
	Method  AccrualMethod `gorm:"size:10;not null;default:'monthly'" json:"method"`

	// FormationEntryID ist die Bildungsbuchung. Leer heißt: nur vorgeschlagen.
	FormationEntryID *uint `gorm:"index" json:"formationEntryId,omitempty"`

	Releases  []AccrualRelease `gorm:"foreignKey:AccrualID;constraint:OnDelete:CASCADE" json:"releases"`
	CreatedAt time.Time        `json:"createdAt"`
}

// AccrualRelease ist eine geplante oder gebuchte Auflösung.
//
// Der Plan steht vollständig in der Datenbank und wird nicht bei jedem Aufruf
// neu gerechnet: die Auflösung des Folgejahres wird beim Saldenvortrag gebucht,
// und was gebucht ist, muss auch dann noch dasselbe sein, wenn jemand später
// das Verteilungsverfahren umstellt.
type AccrualRelease struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	AccrualID  uint   `gorm:"index;not null" json:"accrualId"`
	FiscalYear int    `gorm:"index;not null" json:"fiscalYear"`
	Date       string `gorm:"size:10;not null" json:"date"`
	Amount     Cents  `gorm:"not null" json:"amount"`

	JournalEntryID *uint `gorm:"index" json:"journalEntryId,omitempty"`
}

// IsBooked meldet, ob der Posten gebildet wurde.
func (a *Accrual) IsBooked() bool { return a.FormationEntryID != nil }

// ReleasedAmount ist die Summe der bereits gebuchten Auflösungen.
func (a *Accrual) ReleasedAmount() Cents {
	var total Cents
	for _, r := range a.Releases {
		if r.JournalEntryID != nil {
			total += r.Amount
		}
	}
	return total
}

// Remaining ist der Restbetrag des Postens am Bilanzstichtag: was gebildet und
// noch nicht aufgelöst wurde.
func (a *Accrual) Remaining() Cents { return a.DeferredAmount - a.ReleasedAmount() }

// Validate prüft die Invarianten eines Abgrenzungspostens.
func (a *Accrual) Validate() error {
	if !a.Kind.Valid() {
		return fmt.Errorf("unbekannte Art der Abgrenzung %q", a.Kind)
	}
	if a.Text == "" {
		return fmt.Errorf("zur Abgrenzung gehört ein Text, der sagt, worum es geht")
	}
	if a.TotalAmount <= 0 {
		return fmt.Errorf("der Gesamtbetrag der Abgrenzung muss größer als null sein")
	}
	if a.DeferredAmount <= 0 {
		return fmt.Errorf("es ist kein Betrag abzugrenzen: der Zeitraum liegt vollständig im Geschäftsjahr")
	}
	if a.DeferredAmount > a.TotalAmount {
		return fmt.Errorf(
			"der abgegrenzte Betrag %s € ist größer als der Gesamtbetrag %s €",
			a.DeferredAmount, a.TotalAmount)
	}
	if a.StartDate == "" || a.EndDate == "" {
		return fmt.Errorf("die Abgrenzung braucht Beginn und Ende des Zeitraums")
	}
	if a.EndDate < a.StartDate {
		return fmt.Errorf("der Zeitraum endet am %s und damit vor seinem Beginn am %s", a.EndDate, a.StartDate)
	}
	if a.Account == "" {
		return fmt.Errorf("die Abgrenzung braucht das Aufwands- oder Ertragskonto, das sie berührt")
	}
	if !a.Method.Valid() {
		return fmt.Errorf("unbekanntes Verteilungsverfahren %q", a.Method)
	}
	return nil
}

// AccrualRepository persistiert die Abgrenzungsposten.
type AccrualRepository interface {
	FindAll(ctx context.Context) ([]Accrual, error)
	FindByYear(ctx context.Context, fiscalYear int) ([]Accrual, error)
	FindByID(ctx context.Context, id uint) (*Accrual, error)
	Save(ctx context.Context, accrual *Accrual) error
	// SaveRelease schreibt eine einzelne Auflösung fort. Getrennt vom Posten,
	// weil eine Auflösung neben ihrer Buchung entsteht und ein vollständiges
	// Save die übrigen Auflösungen ersetzen würde.
	SaveRelease(ctx context.Context, release *AccrualRelease) error
	Delete(ctx context.Context, id uint) error
}
