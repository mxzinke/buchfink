// Package einvoice implements EN 16931, the European standard for electronic
// invoicing.
//
// # Was es tut
//
// Das Paket liest, prüft und schreibt elektronische Rechnungen. Es kennt beide
// Syntaxen der Norm — UN/CEFACT CII (die Grundlage von ZUGFeRD, Factur-X und
// einer der beiden XRechnung-Ausprägungen) und OASIS UBL (die andere, sowie
// Peppol BIS Billing) — und den hybriden Fall, bei dem der Datensatz in einem
// PDF steckt.
//
//	inv, err := einvoice.ParseAny(data)   // XML in beiden Syntaxen oder hybrides PDF
//	result := einvoice.Validate(inv)      // alle 223 Geschäftsregeln der Norm
//	xml, err := einvoice.RenderCII(inv)   // wieder hinausschreiben
//
// # Wie es gebaut ist
//
// In der Mitte steht das semantische Modell (siehe [Invoice]) mit allen
// Geschäftsbegriffen der Norm. CII und UBL sind zwei Arten, dieses Modell
// aufzuschreiben; die Prüfung läuft auf dem Modell, nicht auf einer Syntax.
//
// Das ist der Aufbau der Norm selbst: ihre Geschäftsregeln stehen in einem
// abstrakten Regelsatz über dem semantischen Modell, und jede Syntax liefert
// nur die Bindungen. Ihm zu folgen hat einen handfesten Ertrag — dieselbe
// Rechnung wird gleich beurteilt, egal in welcher Schreibweise sie ankommt, und
// das ist nachgewiesen und nicht behauptet (siehe die Rundlauftests).
//
// # Was es nicht tut
//
// Es bucht nicht. Aus einer Rechnung eine Buchung zu machen verlangt
// Entscheidungen, die keine Formatfrage sind — allen voran die Perspektive: der
// Steuerkategoriecode steht aus Sicht des Ausstellers im Dokument und muss auf
// der Eingangsseite gedreht werden. Diese Zuordnung gehört in den
// Buchungspfad, nicht hierher.
//
// Es kennt auch keine nationalen Erweiterungen. XRechnung (BR-DE-*) und Peppol
// legen eigene Regeln über EN 16931; die prüft dieses Paket nicht. Ob ein
// Dokument eine XRechnung zu sein behauptet, sagt [Invoice.IsXRechnung] — damit
// ein Aufrufer weiß, dass Bestehen hier nicht die ganze Geschichte ist.
//
// # Umfang der Prüfung
//
// Alle 223 Geschäftsregeln, die EN 16931 für das semantische Modell und seine
// Codelisten definiert. Vier davon — BR-CO-05 bis BR-CO-08 — verlangen, dass
// der Schlüssel eines Grundes und derselbe Grund im Klartext dasselbe bedeuten;
// das ist
// maschinell nicht entscheidbar, und der Referenzprüfer der Norm führt sie
// selbst als `true()`. Buchfink hält es genauso.
//
// [RulesChecked], [RulesUnchecked] und [Rule] machen den Umfang abfragbar
// statt behauptbar. Zwei Tests halten ihn ehrlich: einer schlägt fehl, wenn
// eine Regel zugesagt wird, die es in der Norm nicht gibt, der andere, wenn
// eine zugesagte Regel nirgends gemeldet wird.
//
// # Abhängigkeiten
//
// Das Paket hängt an nichts aus dem übrigen Buchfink. Es bringt seine eigene
// Zahlenschicht mit ([Amount], [Cents]), weil Beträge in einer Rechnung nicht
// alle zweistellig sind: Einzelpreise und Steuersätze tragen legitim mehr
// Nachkommastellen, und ein Satz von 8,375 %, vorher auf zwei Stellen gerundet,
// liegt je Position ein bis zwei Cent daneben.
package einvoice
