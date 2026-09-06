package domain

import "sort"

// Die Aufgabenliste (Architektur 6.1).
//
// Sie ist die Startseite und nicht die Kennzahlenübersicht: wer keine
// Buchhalterin ist, weiß nach dem Start nicht, was heute dran ist — und ein
// Bankguthaben beantwortet das nicht. Die Liste sammelt, was Buchfink aus den
// Daten ableiten kann, in drei Gruppen und nennt zu jeder Zeile den Grund.

// TaskGroup ist eine der drei Gruppen der Aufgabenliste.
//
// Drei und nicht mehr: „überfällig" ist eine Frist, die verstrichen ist,
// „offen" ist die tägliche Arbeit ohne Frist, „demnächst" ist die Frist der
// nächsten dreißig Tage. Eine vierte Gruppe („irgendwann") wäre eine Liste, die
// niemand liest.
type TaskGroup string

const (
	// TaskGroupOverdue sind die verstrichenen Fristen.
	TaskGroupOverdue TaskGroup = "overdue"
	// TaskGroupOpen ist die laufende Arbeit ohne Frist.
	TaskGroupOpen TaskGroup = "open"
	// TaskGroupUpcoming sind die Fristen der nächsten dreißig Tage.
	TaskGroupUpcoming TaskGroup = "upcoming"
)

// TaskLookaheadDays ist der Vorlauf der Gruppe „demnächst" (Architektur 6.1).
const TaskLookaheadDays = 30

// Die Schlüssel der Aufgaben. Sie stehen in der Oberfläche als Anker und dürfen
// sich deshalb nicht mehr ändern.
//
// Die Befunde des Prüflaufs haben keinen eigenen Schlüssel je Regel: sie
// stehen unter TaskKeyCheckFindings, um die Regel ergänzt
// („check_findings.ic_supply_evidence_missing"). Damit bekommen die
// ausstehende Bestätigung der USt-IdNr. und der fehlende Belegnachweis ihre
// eigene Zeile, ohne dass die Aufgabenliste die Prüfregeln ein zweites Mal
// aufzählt.
const (
	TaskKeyCheckFindings     = "check_findings"
	TaskKeyBankUnmatched     = "bank_unmatched"
	TaskKeyReceiptsUnbooked  = "receipts_unbooked"
	TaskKeyReceiptsOverdue   = "receipts_overdue"
	TaskKeyOverdueItems      = "overdue_receivables"
	TaskKeyDeadline          = "deadline"
	TaskKeyAssetDocument     = "asset_document_expiring"
	TaskKeyExemption         = "exemption_certificate"
	TaskKeyBackupMissing     = "backup_missing"
	TaskKeyBackupFailed      = "backup_failed"
	TaskKeyReadOnly          = "read_only_active"
	TaskKeyServiceProof      = "service_proof_missing"
	TaskKeyCarryForward      = "carry_forward_difference"
	TaskKeyPriorYearOpen     = "prior_year_not_established"
	TaskKeyAppropriationOpen = "appropriation_open"
)

// TaskTarget ist das Navigationsziel einer Aufgabe.
//
// Seite und Parameter getrennt, weil die Oberfläche daraus ihren Weg baut: eine
// fertige URL aus dem Backend wäre eine Festlegung auf einen Router, den das
// Backend nicht kennt.
type TaskTarget struct {
	Page   string            `json:"page"`
	Params map[string]string `json:"params"`
}

// Task ist eine Zeile der Aufgabenliste.
//
// Titel in Vorgangssprache ohne Paragraphen (Architektur 6.4); die Norm steht
// in Reference und wird von der Oberfläche hinter dem Erklärzeichen gezeigt.
type Task struct {
	Key   string    `json:"key"`
	Group TaskGroup `json:"group"`
	Title string    `json:"title"`
	// Why ist der eine Satz, der sagt, warum die Aufgabe besteht.
	Why string `json:"why"`
	// Reference nennt die Norm — nur für die zweite Erklärstufe.
	Reference string     `json:"reference,omitempty"`
	Target    TaskTarget `json:"target"`
	// Count und Amount sind der Kontext: wie viele Posten, welcher Betrag.
	// Null heißt jeweils: für diese Aufgabe ohne Aussage.
	Count   int    `json:"count"`
	Amount  Cents  `json:"amount"`
	DueDate string `json:"dueDate,omitempty"`
}

// TaskList ist die Aufgabenliste in ihren drei Gruppen.
type TaskList struct {
	// Today ist der Tag, gegen den die Fristen gerechnet wurden. Er steht dabei,
	// weil eine Liste ohne ihren Stichtag am nächsten Morgen etwas anderes
	// behauptet, als sie zeigt.
	Today    string `json:"today"`
	Overdue  []Task `json:"overdue"`
	Open     []Task `json:"open"`
	Upcoming []Task `json:"upcoming"`
}

// Add ordnet eine Aufgabe ihrer Gruppe zu.
func (l *TaskList) Add(task Task) {
	if task.Target.Params == nil {
		task.Target.Params = map[string]string{}
	}
	switch task.Group {
	case TaskGroupOverdue:
		l.Overdue = append(l.Overdue, task)
	case TaskGroupUpcoming:
		l.Upcoming = append(l.Upcoming, task)
	default:
		task.Group = TaskGroupOpen
		l.Open = append(l.Open, task)
	}
}

// Total liefert die Gesamtzahl der Aufgaben.
func (l *TaskList) Total() int { return len(l.Overdue) + len(l.Open) + len(l.Upcoming) }

// Sort bringt jede Gruppe in ihre Reihenfolge: die Frist zuerst, die früheste
// oben; Aufgaben ohne Frist danach, nach Schlüssel.
//
// Stabil und nach dem Schlüssel als zweitem Merkmal, damit zwei Läufe über
// denselben Bestand dieselbe Liste ergeben — eine Aufgabenliste, die bei jedem
// Aufruf die Reihenfolge tauscht, liest sich wie eine geänderte Lage.
func (l *TaskList) Sort() {
	for _, group := range [][]Task{l.Overdue, l.Open, l.Upcoming} {
		tasks := group
		sort.SliceStable(tasks, func(i, j int) bool {
			a, b := tasks[i], tasks[j]
			switch {
			case a.DueDate != "" && b.DueDate == "":
				return true
			case a.DueDate == "" && b.DueDate != "":
				return false
			case a.DueDate != b.DueDate:
				return a.DueDate < b.DueDate
			default:
				return a.Key < b.Key
			}
		})
	}
}

// EnsureLists ersetzt nicht belegte Listen durch leere.
//
// Die Aufgabenliste ist der erste Bildschirm nach dem Start, und der Regelfall
// einer aufgeräumten Buchhaltung ist die leere Gruppe. Ein nil-Slice käme dort
// als `null` an, und `null.map` nähme im Render den ganzen Baum mit.
func (l *TaskList) EnsureLists() {
	if l.Overdue == nil {
		l.Overdue = make([]Task, 0)
	}
	if l.Open == nil {
		l.Open = make([]Task, 0)
	}
	if l.Upcoming == nil {
		l.Upcoming = make([]Task, 0)
	}
	for _, group := range [][]Task{l.Overdue, l.Open, l.Upcoming} {
		for i := range group {
			if group[i].Target.Params == nil {
				group[i].Target.Params = map[string]string{}
			}
		}
	}
}
