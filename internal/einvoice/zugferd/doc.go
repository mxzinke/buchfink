// Package zugferd implements the ZUGFeRD and Factur-X profiles over EN 16931.
//
// ZUGFeRD is not a separate format either. It is a set of profiles — subsets of
// the standard, from MINIMUM up to EXTENDED — plus a delivery form: the
// structured data travels inside a PDF/A-3, so the same file is both a readable
// invoice and a machine-readable record.
//
//	profile := zugferd.ProfileOf(inv)          // was das Dokument angibt
//	needed  := zugferd.MinimumProfileFor(inv)  // was sein Inhalt verlangt
//	result  := einvoice.ValidateWith(inv, zugferd.Ruleset())
//
// # Warum das Profil zählt
//
// Zwei der Profile tragen keine vollständige Rechnung. MINIMUM hat weder
// Positionen noch Steueraufschlüsselung, BASIC WL keine Positionen; nach
// UStAE 14.1 Abs. 14 Satz 4 sind sie deshalb keine E-Rechnung im Sinne des
// Gesetzes. Wer daraus Vorsteuer zieht, zieht sie aus einem Dokument, das
// rechtlich keine Rechnung ist. Sie sind als Begleitsatz zu einer Papier- oder
// PDF-Rechnung gedacht, und nur so dürfen sie behandelt werden.
//
// # Was hier geprüft wird
//
// Ob der Inhalt eines Dokuments zu dem Profil passt, das es angibt. Ein Beleg,
// der MINIMUM nennt und Positionen mitschickt, widerspricht sich selbst — und
// ein Empfänger, der dem Profil glaubt, übersieht die Hälfte.
//
// Die Regelkennungen dafür (ZF-PROFIL-*) sind Buchfinks eigene. Das
// Validierungsartefakt drückt dieselben Beschränkungen als
// XML-Kardinalitätsregeln aus (FX-SCH-A-*, bis zu 943 je Profil); die gehören
// zur CII-Syntax und lassen sich am semantischen Modell nicht nachbilden. Die
// Profiltabelle unten ist maschinell aus eben diesen Regeln abgeleitet.
//
// Abgeleitet aus: ZUGFeRD/mustangproject, validator/schematron/ZF_250
// (Apache-2.0).
package zugferd
