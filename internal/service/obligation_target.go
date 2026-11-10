package service

import "github.com/buchfink/buchfink/internal/domain"

// ObligationTarget ist die Stelle der Oberfläche, an der eine Pflicht
// vorbereitet oder bestätigt wird. Die Aufgabenliste und die Übersicht führen
// dieselbe Pflicht an dieselbe Stelle.
func ObligationTarget(o domain.Obligation) domain.TaskTarget { return obligationTarget(o) }
