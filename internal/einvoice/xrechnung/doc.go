// Package xrechnung implements the German CIUS on top of EN 16931.
//
// XRechnung is not a separate format. It is a "Core Invoice Usage
// Specification": the same semantic model, the same syntaxes, with rules laid
// on top that make optional fields mandatory, restrict code lists and forbid
// combinations the standard would allow. Public authorities in Germany require
// it, and since 2025 every business receiving invoices has to be able to read
// one.
//
// The package is therefore a layer, not a parallel implementation:
//
//	inv, err := einvoice.ParseAny(data)
//	result := einvoice.ValidateWith(inv, xrechnung.Ruleset())
//
// The core rules run once, this layer adds its own, and no document can be
// judged differently by the two.
//
// # Was nicht geprüft wird
//
// Die **XRechnung-Extension** (BR-DEX-02, -03, -09 bis -15) betrifft
// Geschäftsbegriffe, die es in EN 16931 nicht gibt: Unterpositionen
// (BG-DEX-01) und Zahlungen durch Dritte (BG-DEX-09). Sie stehen nicht im
// semantischen Modell und werden hier nicht geprüft. [UncheckedRules] nennt sie
// einzeln.
//
// Die Regeln für die **Beschaffung sauberer Fahrzeuge** (BR-DE-CVD-*) gehören
// zu einem Vergabeszenario, das sich dem Dokument nicht ansehen lässt — der
// Auftraggeber weiß, dass er nach der Clean Vehicles Directive beschafft, die
// Rechnung sagt es nicht. Sie laufen deshalb nur auf Verlangen, über
// [WithCleanVehicles].
//
// Erzeugt gegen: itplr-kosit/xrechnung-schematron (Apache-2.0).
package xrechnung
