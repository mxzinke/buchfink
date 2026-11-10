package domain

const (
	// SettingTakeoverDate ist der Tag (JJJJ-MM-TT), zu dem das Unternehmen mit
	// bestehenden Büchern aus einem Vorsystem beginnt: der erste Tag eines
	// Geschäftsjahres. Leer heißt: kein Umstieg. Nur der Mensch legt ihn fest;
	// ohne ihn nimmt Buchfink keine Eröffnungsbuchungen aus einem Vorsystem an.
	SettingTakeoverDate = "takeover_date"
	// SettingTakeoverStatement ist die Schlussbilanz des Vorsystems, wie sie zur
	// Eröffnung erfasst wurde (JSON). Die Abstimmung vergleicht die
	// Eröffnungsbuchungen mit ihr.
	SettingTakeoverStatement = "takeover_statement"
)

// CheckRuleTakeoverOpening meldet im Prüflauf eine fehlende oder nicht
// aufgehende Eröffnung eines Umsteigers.
const CheckRuleTakeoverOpening = "takeover_opening"
