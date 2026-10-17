package ebilanz

import (
	"fmt"
	"strings"
)

// versionRule ist der Anwendungszeitraum einer Fassung.
//
// Technisch nimmt ERiC eine Fassung für Wirtschaftsjahre an, die nach
// AcceptedAfter beginnen und vor EndsBefore enden (Leitfaden B.18). Regelfall
// ist sie für Wirtschaftsjahre, die nach RegularAfter beginnen; für das
// Wirtschaftsjahr davor beanstandet die Verwaltung sie nicht (BMF-Schreiben
// vom 10.06.2025 zu 6.9 und vom 08.06.2026 zu 6.10).
type versionRule struct {
	Version       string
	AcceptedAfter string
	RegularAfter  string
	EndsBefore    string
}

var versionRules = []versionRule{
	{Version: "6.9", AcceptedAfter: "2024-12-31", RegularAfter: "2025-12-31", EndsBefore: "2027-12-31"},
	{Version: "6.10", AcceptedAfter: "2025-12-31", RegularAfter: "2026-12-31", EndsBefore: "2028-12-31"},
}

// VersionChoice ist die gewählte Fassung mit den zulässigen Alternativen.
type VersionChoice struct {
	Version string `json:"version"`
	// Regular sagt, ob die Fassung für dieses Wirtschaftsjahr die
	// Regelfassung ist und nicht nur zulässig.
	Regular bool     `json:"regular"`
	Allowed []string `json:"allowed"`
	Reason  string   `json:"reason"`
}

// ChooseVersion wählt die Taxonomie nach dem Wirtschaftsjahr. Ohne Wunsch ist
// es die Regelfassung, sonst die neueste zulässige. Eine gewünschte Fassung
// muss zulässig sein.
//
// Eine Eröffnungsbilanz geht mit einer Fassung, mit der auch die Schlussbilanz
// des ersten Wirtschaftsjahres übermittelt werden kann; maßgeblich sind Beginn
// und Ende des Wirtschaftsjahres (Leitfaden B.18).
func ChooseVersion(fiscalYearBegin, fiscalYearEnd, requested string) (VersionChoice, error) {
	if len(fiscalYearBegin) != 10 || len(fiscalYearEnd) != 10 {
		return VersionChoice{}, fmt.Errorf("Beginn und Ende des Wirtschaftsjahres fehlen (erwartet JJJJ-MM-TT)")
	}
	var choice VersionChoice
	for _, r := range versionRules {
		if fiscalYearBegin > r.AcceptedAfter && fiscalYearEnd < r.EndsBefore {
			choice.Allowed = append(choice.Allowed, r.Version)
		}
	}
	if len(choice.Allowed) == 0 {
		return VersionChoice{}, fmt.Errorf(
			"für das Wirtschaftsjahr %s bis %s ist keine der eingebundenen Taxonomien (%s) zulässig "+
				"(Leitfaden B.18); dafür braucht es eine ältere oder neuere Fassung",
			fiscalYearBegin, fiscalYearEnd, strings.Join(Versions(), ", "))
	}
	// Ohne Regelfassung unter den zulässigen bleibt die älteste zulässige: das
	// Wirtschaftsjahr 2025 verlangt regulär 6.8, nimmt aber 6.9 an.
	choice.Version = choice.Allowed[0]
	for _, v := range choice.Allowed {
		if regularFor(v, fiscalYearBegin) {
			choice.Version, choice.Regular = v, true
		}
	}
	if requested != "" {
		if !contains(choice.Allowed, requested) {
			return VersionChoice{}, fmt.Errorf(
				"die Taxonomie %s ist für das Wirtschaftsjahr %s bis %s nicht zulässig; zulässig: %s",
				requested, fiscalYearBegin, fiscalYearEnd, strings.Join(choice.Allowed, ", "))
		}
		choice.Version = requested
		choice.Regular = regularFor(requested, fiscalYearBegin)
	}
	if choice.Regular {
		choice.Reason = fmt.Sprintf("Taxonomie %s ist die Regelfassung für Wirtschaftsjahre, die nach dem %s beginnen.",
			choice.Version, germanDate(ruleOf(choice.Version).RegularAfter))
	} else {
		choice.Reason = fmt.Sprintf("Taxonomie %s ist für dieses Wirtschaftsjahr zulässig; ältere Fassungen bindet Buchfink nicht ein.",
			choice.Version)
	}
	return choice, nil
}

func ruleOf(version string) versionRule {
	for _, r := range versionRules {
		if r.Version == version {
			return r
		}
	}
	return versionRule{}
}

func regularFor(version, fiscalYearBegin string) bool {
	r := ruleOf(version)
	if fiscalYearBegin <= r.RegularAfter {
		return false
	}
	for _, other := range versionRules {
		if other.RegularAfter > r.RegularAfter && fiscalYearBegin > other.RegularAfter {
			return false
		}
	}
	return true
}

// germanDate macht aus 2025-12-31 den 31.12.2025.
func germanDate(iso string) string {
	if len(iso) != 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
}
