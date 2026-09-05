package accounting

import (
	"fmt"
	"sort"
	"strings"
)

// Der Belegnachweis der innergemeinschaftlichen Lieferung (§§ 17a bis 17c UStDV).
//
// Die Steuerbefreiung des § 4 Nr. 1 Buchst. b i. V. m. § 6a UStG steht und fällt
// mit dem Nachweis, dass der Gegenstand tatsächlich in das übrige
// Gemeinschaftsgebiet gelangt ist. § 17a UStDV übernimmt dafür die
// Gelangensvermutung des Art. 45a MwStVO: liegen bestimmte, einander nicht
// widersprechende Belege von voneinander unabhängigen Dritten vor, wird das
// Gelangen vermutet. § 17b UStDV lässt daneben den einfacheren Weg über das
// Rechnungsdoppel und die Gelangensbestätigung.
//
// Buchfink bewertet die Belege und entscheidet nicht über die Steuerbefreiung.
// Der Unterschied ist wichtig: die Vermutung ist widerlegbar, und ob ein
// Frachtbrief einem anderen Beleg widerspricht, sieht ein Mensch und keine
// Software. Was hier entsteht, ist die Auskunft „die Voraussetzungen der
// Vermutung liegen vor" oder „es fehlt dieses".

// EvidenceKind ist die Art eines Nachweisbelegs.
type EvidenceKind string

const (
	// --- Gruppe a des Art. 45a Abs. 3 MwStVO: Beförderungsbelege -----------
	EvidenceCMR             EvidenceKind = "cmr_frachtbrief"
	EvidenceBillOfLading    EvidenceKind = "konnossement"
	EvidenceAirFreight      EvidenceKind = "luftfrachtrechnung"
	EvidenceForwarderInvoic EvidenceKind = "spediteurbescheinigung"

	// --- Gruppe b des Art. 45a Abs. 3 MwStVO: sonstige Belege --------------
	EvidenceInsurance       EvidenceKind = "versicherungspolice"
	EvidenceBankRecord      EvidenceKind = "bankbeleg"
	EvidencePublicAuthority EvidenceKind = "behoerdliche_bestaetigung"
	EvidenceWarehouse       EvidenceKind = "lagerbescheinigung"

	// --- Belege außerhalb der beiden Gruppen -------------------------------

	// EvidenceArrival ist die Gelangensbestätigung des Abnehmers (§ 17b Abs. 2
	// Nr. 2 UStDV). Im Abholfall verlangt Art. 45a Abs. 1 Buchst. b MwStVO sie
	// zusätzlich zu den Belegen; für sich allein trägt sie den Nachweis nur
	// zusammen mit dem Rechnungsdoppel nach § 17b UStDV.
	EvidenceArrival EvidenceKind = "gelangensbestaetigung"
	// EvidenceInvoiceCopy ist das Doppel der Rechnung (§ 17b Abs. 1 UStDV).
	EvidenceInvoiceCopy EvidenceKind = "rechnungsdoppel"
	// EvidenceTracking ist ein Sendungsverfolgungsprotokoll. Es steht in keiner
	// der beiden Gruppen des Art. 45a MwStVO und trägt die Vermutung deshalb
	// nicht — als ergänzender Beleg im Rahmen der freien Beweiswürdigung ist es
	// trotzdem etwas wert, und deshalb ist es ablegbar.
	EvidenceTracking EvidenceKind = "tracking_protokoll"
	EvidenceOther    EvidenceKind = "sonstiges"
)

// EvidenceGroup ordnet eine Belegart in die Systematik des Art. 45a MwStVO ein.
type EvidenceGroup string

const (
	EvidenceGroupTransport EvidenceGroup = "a" // Beförderungsbelege
	EvidenceGroupOther     EvidenceGroup = "b" // sonstige Belege
	EvidenceGroupNone      EvidenceGroup = ""  // trägt die Vermutung nicht
)

// EvidenceKindInfo beschreibt eine Belegart für die Oberfläche und für das
// Schlüsselverzeichnis der Datenüberlassung.
type EvidenceKindInfo struct {
	Kind  EvidenceKind  `json:"kind"`
	Label string        `json:"label"`
	Group EvidenceGroup `json:"group"`
	Hint  string        `json:"hint,omitempty"`
}

var evidenceKinds = []EvidenceKindInfo{
	{EvidenceCMR, "CMR-Frachtbrief (unterzeichnet)", EvidenceGroupTransport,
		"Der unterzeichnete CMR-Frachtbrief ist der Regelfall des Beförderungsbelegs."},
	{EvidenceBillOfLading, "Konnossement", EvidenceGroupTransport, ""},
	{EvidenceAirFreight, "Luftfrachtrechnung", EvidenceGroupTransport, ""},
	{EvidenceForwarderInvoic, "Spediteurrechnung oder -bescheinigung", EvidenceGroupTransport, ""},
	{EvidenceInsurance, "Versicherungspolice für den Transport", EvidenceGroupOther, ""},
	{EvidenceBankRecord, "Bankbeleg über die Bezahlung des Transports", EvidenceGroupOther, ""},
	{EvidencePublicAuthority, "Bestätigung einer öffentlichen Stelle über die Ankunft", EvidenceGroupOther, ""},
	{EvidenceWarehouse, "Lagerbescheinigung des Lagerinhabers", EvidenceGroupOther, ""},
	{EvidenceArrival, "Gelangensbestätigung des Abnehmers", EvidenceGroupNone,
		"Im Abholfall zusätzlich erforderlich; zusammen mit dem Rechnungsdoppel trägt sie den " +
			"Nachweis nach § 17b UStDV auch allein."},
	{EvidenceInvoiceCopy, "Doppel der Rechnung", EvidenceGroupNone,
		"Der zweite Baustein des Nachweises nach § 17b Abs. 1 UStDV."},
	{EvidenceTracking, "Sendungsverfolgungsprotokoll", EvidenceGroupNone,
		"Steht in keiner der Gruppen des Art. 45a MwStVO und trägt die Vermutung nicht."},
	{EvidenceOther, "Sonstiger Beleg", EvidenceGroupNone, ""},
}

// EvidenceKinds liefert die Belegarten in fester Reihenfolge.
func EvidenceKinds() []EvidenceKindInfo {
	out := make([]EvidenceKindInfo, len(evidenceKinds))
	copy(out, evidenceKinds)
	return out
}

// EvidenceKindGroup liefert die Gruppe einer Belegart und ob sie bekannt ist.
func EvidenceKindGroup(kind EvidenceKind) (EvidenceGroup, bool) {
	for _, k := range evidenceKinds {
		if k.Kind == kind {
			return k.Group, true
		}
	}
	return EvidenceGroupNone, false
}

// EvidenceKindLabel liefert den Klartext einer Belegart.
func EvidenceKindLabel(kind EvidenceKind) string {
	for _, k := range evidenceKinds {
		if k.Kind == kind {
			return k.Label
		}
	}
	return string(kind)
}

// TransportKind sagt, wer den Gegenstand befördert hat.
//
// Die Unterscheidung ist die Weiche des Art. 45a Abs. 1 MwStVO: befördert der
// Lieferer, genügen die Belege; holt der Abnehmer ab, kommt seine schriftliche
// Bestätigung hinzu, dass der Gegenstand angekommen ist. Ohne sie weiß niemand,
// wohin er gefahren ist.
type TransportKind string

const (
	// TransportBySupplier: Beförderung oder Versendung durch den Lieferer oder
	// auf seine Rechnung durch einen Dritten (Art. 45a Abs. 1 Buchst. a MwStVO).
	TransportBySupplier TransportKind = "supplier"
	// TransportByCustomer: Abholfall — der Erwerber befördert oder versendet
	// (Art. 45a Abs. 1 Buchst. b MwStVO).
	TransportByCustomer TransportKind = "customer"
)

// EvidenceItem ist ein abgelegter Nachweisbeleg, so weit die Bewertung ihn
// braucht.
type EvidenceItem struct {
	Kind EvidenceKind
	// Issuer ist der Aussteller. Er entscheidet über die Unabhängigkeit: zwei
	// Belege desselben Spediteurs sind ein Beleg mit zwei Blättern.
	Issuer string
	// Independent sagt, dass der Aussteller weder der Lieferer noch der Erwerber
	// ist. Art. 45a Abs. 1 MwStVO verlangt „von zwei verschiedenen Parteien, die
	// voneinander sowie vom Verkäufer und vom Erwerber unabhängig sind".
	Independent bool
}

// EvidenceStatus ist die Bewertung des Belegnachweises einer Lieferung.
type EvidenceStatus struct {
	Fulfilled bool `json:"fulfilled"`
	// Basis nennt die Vorschrift, auf die sich das Ergebnis stützt.
	Basis string `json:"basis,omitempty"`
	// Reason ist ein Satz in Klartext: warum der Nachweis trägt oder was fehlt.
	Reason string `json:"reason"`
	// Missing sind die Bausteine, die noch fehlen — leer, wenn nichts fehlt.
	Missing []string `json:"missing"`
	// GroupACount und GroupBCount sind die gezählten unabhängigen Belege.
	GroupACount int `json:"groupACount"`
	GroupBCount int `json:"groupBCount"`
}

// AssessSupplyEvidence bewertet die Belege einer innergemeinschaftlichen
// Lieferung.
//
// Gezählt werden nur Belege unabhängiger Aussteller, und je Aussteller nur
// einer: die Vermutung verlangt zwei *voneinander unabhängige* Parteien, und
// zwei Frachtbriefe derselben Spedition sind keine zwei Parteien. Ohne diese
// Zählung wäre die Prüfung eine Zählung von Dateien und keine des Nachweises.
func AssessSupplyEvidence(transport TransportKind, items []EvidenceItem) EvidenceStatus {
	status := EvidenceStatus{Missing: make([]string, 0, 3)}

	hasArrival := false
	hasInvoiceCopy := false
	// Je Gruppe wird ein Aussteller nur einmal gezählt.
	issuersA := map[string]bool{}
	issuersB := map[string]bool{}
	for _, it := range items {
		switch it.Kind {
		case EvidenceArrival:
			hasArrival = true
		case EvidenceInvoiceCopy:
			hasInvoiceCopy = true
		}
		if !it.Independent {
			continue
		}
		issuer := strings.ToLower(strings.TrimSpace(it.Issuer))
		if issuer == "" {
			// Ohne Aussteller lässt sich die Unabhängigkeit nicht prüfen. Der
			// Beleg zählt deshalb nicht — er ist nicht wertlos, aber er trägt die
			// Vermutung nicht.
			continue
		}
		group, _ := EvidenceKindGroup(it.Kind)
		switch group {
		case EvidenceGroupTransport:
			issuersA[issuer] = true
		case EvidenceGroupOther:
			issuersB[issuer] = true
		}
	}
	// Ein Aussteller, der einen Beleg aus a und einen aus b geliefert hat, ist
	// eine Partei und keine zwei. Er zählt dort, wo er als Erstes gebraucht wird.
	for issuer := range issuersA {
		delete(issuersB, issuer)
	}
	status.GroupACount = len(issuersA)
	status.GroupBCount = len(issuersB)

	// § 17b UStDV: Rechnungsdoppel und Gelangensbestätigung tragen den Nachweis
	// ohne die Vermutung des § 17a UStDV. Der Weg steht neben ihr und nicht
	// hinter ihr — er wird zuerst geprüft, weil er der einfachere ist.
	if hasInvoiceCopy && hasArrival {
		status.Fulfilled = true
		status.Basis = "§ 17b Abs. 1 und Abs. 2 Nr. 2 UStDV"
		status.Reason = "Das Doppel der Rechnung und die Gelangensbestätigung des Abnehmers liegen vor. " +
			"Damit ist der Belegnachweis geführt."
		return status
	}

	presumption := status.GroupACount >= 2 || (status.GroupACount >= 1 && status.GroupBCount >= 1)
	needsArrival := transport == TransportByCustomer

	switch {
	case presumption && (!needsArrival || hasArrival):
		status.Fulfilled = true
		status.Basis = "§ 17a Abs. 1 UStDV i. V. m. Art. 45a MwStVO"
		status.Reason = evidencePresumptionReason(status, needsArrival)
		return status
	case presumption:
		status.Missing = append(status.Missing, "Gelangensbestätigung des Abnehmers")
		status.Reason = "Die Belege reichen für die Vermutung aus, aber der Abnehmer hat den Gegenstand " +
			"selbst befördert. Dann verlangt Art. 45a Abs. 1 Buchst. b MwStVO zusätzlich seine " +
			"schriftliche Bestätigung, dass der Gegenstand im Bestimmungsland angekommen ist."
		return status
	}

	switch {
	case status.GroupACount == 0:
		status.Missing = append(status.Missing,
			"ein Beförderungsbeleg (CMR-Frachtbrief, Konnossement, Luftfrachtrechnung oder Spediteurrechnung)")
	case status.GroupACount == 1 && status.GroupBCount == 0:
		status.Missing = append(status.Missing,
			"ein zweiter Beförderungsbeleg oder ein Beleg der Gruppe b (Versicherungspolice, Bankbeleg, "+
				"behördliche Bestätigung, Lagerbescheinigung) — von einem anderen, unabhängigen Aussteller")
	}
	if needsArrival && !hasArrival {
		status.Missing = append(status.Missing, "Gelangensbestätigung des Abnehmers")
	}
	if !hasInvoiceCopy || !hasArrival {
		status.Missing = append(status.Missing,
			"alternativ: das Doppel der Rechnung zusammen mit der Gelangensbestätigung (§ 17b UStDV)")
	}
	sort.Strings(status.Missing)

	status.Reason = fmt.Sprintf(
		"Der Belegnachweis ist nicht geführt. Vorliegen %d unabhängige Beförderungsbelege und %d "+
			"unabhängige Belege der Gruppe b; die Vermutung des § 17a Abs. 1 UStDV setzt zwei "+
			"Beförderungsbelege oder einen Beförderungsbeleg und einen Beleg der Gruppe b voraus, "+
			"jeweils von voneinander unabhängigen Ausstellern.",
		status.GroupACount, status.GroupBCount)
	return status
}

func evidencePresumptionReason(status EvidenceStatus, needsArrival bool) string {
	base := ""
	if status.GroupACount >= 2 {
		base = fmt.Sprintf("Es liegen %d Beförderungsbelege von voneinander unabhängigen Ausstellern vor.",
			status.GroupACount)
	} else {
		base = "Es liegen ein Beförderungsbeleg und ein Beleg der Gruppe b von voneinander unabhängigen " +
			"Ausstellern vor."
	}
	if needsArrival {
		base += " Für den Abholfall kommt die Gelangensbestätigung des Abnehmers hinzu."
	}
	return base + " Das Gelangen in das übrige Gemeinschaftsgebiet wird damit vermutet."
}
