package domain

import (
	"context"
	"time"
)

// ExaminerPeriod ist ein Zeitraum des Prüfermodus. Während einer Außenprüfung
// wird der Bestand eingefroren (§ 147 Abs. 6 AO): was der Prüfer sieht, darf
// sich unter seinen Händen nicht ändern.
//
// Der Zeitraum steht in der Mandantendatenbank und nicht in der
// App-Konfiguration: eine Sicherung, ein kopierter Datenordner oder ein
// zweiter Rechner tragen die Sperre mit.
type ExaminerPeriod struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// From ist der Tag des Einschaltens, Until der letzte Tag. Ein Ende ist
	// Pflicht: ein Modus ohne Ende würde vergessen und sperrte irgendwann die
	// laufende Buchführung.
	From   string `gorm:"size:10;not null" json:"from"`
	Until  string `gorm:"size:10;not null;index" json:"until"`
	Reason string `gorm:"type:text;not null" json:"reason"`
	// EndedOn ist der Tag eines vorzeitigen Endes, EndReason sein Grund.
	EndedOn   *string   `gorm:"size:10" json:"endedOn,omitempty"`
	EndReason string    `gorm:"type:text" json:"endReason,omitempty"`
	Actor     string    `gorm:"size:200" json:"actor"`
	Channel   string    `gorm:"size:80" json:"channel"`
	CreatedAt time.Time `json:"createdAt"`
}

// ActiveOn meldet, ob der Zeitraum an diesem Tag gilt. Der letzte Tag zählt
// mit: „bis zum 30.06." heißt einschließlich. Ein abgelaufener Zeitraum endet
// von selbst.
func (p *ExaminerPeriod) ActiveOn(today string) bool {
	return p != nil && p.EndedOn == nil && p.From <= today && today <= p.Until
}

// ExaminerRepository hält die Zeiträume des Prüfermodus.
type ExaminerRepository interface {
	Create(ctx context.Context, p *ExaminerPeriod) error
	Save(ctx context.Context, p *ExaminerPeriod) error
	// Latest liefert den jüngsten nicht vorzeitig beendeten Zeitraum oder nil.
	Latest(ctx context.Context) (*ExaminerPeriod, error)
}
