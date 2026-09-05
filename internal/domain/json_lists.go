package domain

// Leere Listen statt `null` in den Antworten, die an die Oberfläche gehen.
//
// Ein nicht belegter Go-Slice wird in JSON zu `null`. Die Masken lesen die
// Antworten ohne Umweg — `receipt.files.map`, `accrual.releases.length` —, und
// `null.map` wirft im Render einen TypeError, der ohne ErrorBoundary den ganzen
// Baum mitnimmt. Betroffen ist jeweils der Randfall, den niemand von Hand
// ausprobiert: der Beleg ohne Datei, die Abgrenzung ohne Auflösungsplan, die
// Gründung ohne erfassten Gesellschafter.
//
// Die Listen entstehen an zwei Stellen leer: GORM belegt eine Verknüpfung ohne
// Kindsätze nicht, und ein `append` auf einen nicht belegten Slice läuft nie.
// Deshalb sitzen die Zusicherungen an den Typen selbst und werden dort
// aufgerufen, wo gelesen wird.

// EnsureLists ersetzt die nicht belegten Listen eines Belegs durch leere.
func (r *Receipt) EnsureLists() {
	if r == nil {
		return
	}
	if r.Files == nil {
		r.Files = make([]ReceiptFile, 0)
	}
}

// EnsureLists ersetzt den nicht belegten Auflösungsplan durch einen leeren.
func (a *Accrual) EnsureLists() {
	if a == nil {
		return
	}
	if a.Releases == nil {
		a.Releases = make([]AccrualRelease, 0)
	}
}

// EnsureLists ersetzt die nicht belegten Bewegungen einer Rückstellung durch
// eine leere Liste.
func (p *Provision) EnsureLists() {
	if p == nil {
		return
	}
	if p.Movements == nil {
		p.Movements = make([]ProvisionMovement, 0)
	}
}

// EnsureLists ersetzt die nicht belegte Gesellschafterliste durch eine leere.
func (f *Foundation) EnsureLists() {
	if f == nil {
		return
	}
	if f.Shareholders == nil {
		f.Shareholders = make([]Shareholder, 0)
	}
}

// EnsureLists ersetzt die nicht belegte Abschlagsliste eines Verbunds durch
// eine leere.
func (g *InvoiceGroup) EnsureLists() {
	if g == nil {
		return
	}
	if g.Advances == nil {
		g.Advances = make([]AdvanceItem, 0)
	}
}

// EnsureLists ersetzt die nicht belegte Fristenliste des Abschlusses durch eine
// leere. Ohne Bilanzstichtag gibt es keine Frist — und trotzdem eine Liste.
func (s *FinancialStatement) EnsureLists() {
	if s == nil {
		return
	}
	if s.Deadlines == nil {
		s.Deadlines = make([]Deadline, 0)
	}
}

// EnsureLists ersetzt die nicht belegten Umsätze je Steuersatz durch eine leere
// Liste. Ein Zeitraum ohne steuerpflichtigen Umsatz ist der Regelfall im
// ersten Monat einer Gesellschaft.
func (s *VatSummary) EnsureLists() {
	if s == nil {
		return
	}
	if s.TaxableRevenue == nil {
		s.TaxableRevenue = make([]VatFigure, 0)
	}
}
