# E-Rechnung

[Fachkonzepte](README.md) · [Belege und Buchungen](beleg-buchungsflow.md) ·
[Modulaufbau](../entwicklung/e-rechnung.md)

Anforderungen: RECH-05 bis RECH-07, RECH-10, UST-07, ARC-01, ARC-07.

## Pflichten

Ab dem 1. Januar 2025 ist zwischen inländischen Unternehmern grundsätzlich
per E-Rechnung abzurechnen ([§ 14 Abs. 2 Satz 2 Nr. 1 UStG][ustg14]). Für das
Ausstellen gelten Übergangsregeln bis Ende 2026 und Ende 2027
([§ 27 Abs. 38 UStG][ustg27]), für das Empfangen keine: jeder inländische
Unternehmer muss eine E-Rechnung annehmen können (UStAE 14.1 Abs. 5). Erfasst
sind auch Umsätze mit Steuerschuld des Empfängers. Kleinunternehmer dürfen
sonstige Rechnungen ausstellen, müssen E-Rechnungen aber empfangen.

Der Vorsteuerabzug folgt aus dem strukturierten Teil (UStAE 14c.1 Abs. 4a).
Deshalb bucht Buchfink immer aus dem XML. Das Bild einer empfangenen hybriden
Rechnung gleicht es nicht mit dem XML ab.

## Empfang

- ZUGFeRD, Factur-X und XRechnung in den Syntaxen CII und UBL sowie Peppol BIS
  Billing 3.0 werden erkannt und gelesen, bei hybriden PDFs aus dem
  eingebetteten XML.
- Geprüft wird gegen alle 223 Geschäftsregeln von EN 16931, eine XRechnung
  zusätzlich gegen die deutschen Regeln. Ergebnis, Zeitpunkt und Regelstand
  stehen am Beleg. Bei XRechnung und Peppol BIS Billing 3.0 heißt der
  Prüfumfang „teilweise“: Extension- und Syntaxregeln der XRechnung und die
  Peppol-Regeln (PEPPOL-EN16931-R*) prüft Buchfink nicht. Die Prüfung ändert
  keine Datei und damit nicht den Beleg-Hash.
- Die Profile MINIMUM und BASIC WL enthalten keine vollständige Rechnung und
  gelten nicht als E-Rechnung (UStAE 14.1 Abs. 14).
- Aus dem Datensatz entsteht ein Buchungsvorschlag: Lieferant über USt-IdNr. oder
  Steuernummer, Beträge und Sätze aus dem XML. Den Steuerkategoriecode dreht
  Buchfink in die Sicht des Empfängers: „K“ ist beim Lieferanten eine
  innergemeinschaftliche Lieferung und beim Empfänger ein Erwerb, „AE“ heißt
  Steuerschuld beim Empfänger. Die Buchungsgruppe wählt der Mensch oder der
  Agent; keine Rechnung sagt, welches Aufwandskonto zutrifft.
- Eine Gutschrift bucht Buchfink als Rechnung mit getauschten Seiten
  (`gutschrift_zu`, [Belege und Buchungen](beleg-buchungsflow.md)). Korrektur,
  Anzahlungs- und Abschlagsrechnung erkennt Buchfink am Rechnungstyp und
  schlägt dafür keine Buchung vor.
- Eine reine XRechnung ist ohne PDF buchbar; die App zeigt sie aus dem XML
  (`service.EInvoiceService.View`).
- Mitgeschickte Anlagen sind Belegdateien am Beleg.

Ein eigenes Postfach braucht es nicht; es genügt, die empfangene Datei
abzulegen (UStAE 14.1 Abs. 5).

## Ausstellung

Ausgangsrechnungen erzeugt Buchfink als ZUGFeRD im Profil EN 16931 (PDF/A-3b mit
eingebettetem XML) oder als XRechnung im CII-Format. Gesetzt wird mit Typst als
WebAssembly im Prozess. Eine Rechnung, die gegen die Regeln verstößt, entsteht
gar nicht erst; eine XRechnung verlangt die Leitweg-ID des Empfängers. Vor der
Ablage liest Buchfink den Datensatz aus dem erzeugten Dokument zurück und
vergleicht ihn mit der Rechnung (`invoice.VerifyEmbeddedRecord`), weil PDF und XML
dasselbe sagen müssen (UStAE 14c.1). Das hybride PDF wird als Ausgangsbeleg
abgelegt, obwohl GoBD Rz. 76 den Verzicht erlaubt: ein gespeichertes Dokument
ist greifbarer als die Zusage, es jederzeit neu zu erzeugen.

## Storno und Berichtigung

Eine Stornorechnung hat eine eigene Nummer und verweist mit Nummer und Datum
auf die Ursprungsrechnung (BG-3, BT-25/BT-26). Im XML ist sie eine Credit Note
mit Typcode 381 und positiven Beträgen; der lesbare Teil heißt Stornorechnung
und zeigt die negativen Beträge. Die Buchung kehrt die Rechnung einschließlich
ihrer Differenzbuchungen um. Erhaltene Zahlungen bleiben als Kundenguthaben in
den offenen Posten und können verrechnet oder erstattet werden.

Die Berichtigung von Euro-Rechnungen hat Typcode 384 und den vollständigen
richtigen Inhalt. [§ 31 Abs. 5 UStDV][ustdv31] verlangt nur die fehlenden oder
unrichtigen Angaben und einen eindeutigen Bezug. Der vollständige Inhalt und die
eigene Nummer sind eine Produktentscheidung. Eine reine Textberichtigung bucht
nichts; sonst entsteht eine Differenzbuchung am Änderungstag
([§ 17 Abs. 1 UStG][ustg17]). Beide Wege lassen festgeschriebene Perioden unberührt.
Eine bereits ersetzte oder stornierte Rechnung lässt sich nicht erneut bearbeiten.
Bei einem anderen Kunden wird storniert und neu ausgestellt. Eine aktive
Forderungsausbuchung muss zuerst storniert werden, damit die Steuer nicht zweimal
vermindert wird.

Vereinnahmte Abschläge lassen sich erst nach ihrer Rückzahlung stornieren;
verrechnete Abschläge erst nach der Stornierung der Schlussrechnung. Der Storno
der Schlussrechnung öffnet den Verbund wieder und nimmt die Absetzung der
Anzahlungen zurück. Die Berichtigung der Schlussrechnung bucht die Differenz
und behält die Absetzung der Anzahlungen. Unbezahlte Abschläge werden in einer Transaktion
storniert und durch einen berichtigten Abschlag im selben Verbund ersetzt;
es entsteht keine Buchung. Beide Wege haben eine Vorschau.

Die Regeln für 381/384 prüft `internal/einvoice`, insbesondere die Typcodeliste
und BR-DE-26 in `xrechnung/ruleset.go`. Die Vorzeichen einer Credit Note werden
an der Dokumentgrenze übersetzt und beim Rücklesen wieder verglichen.

[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html

## Aufbewahrung

Bei E-Rechnungen genügt der strukturierte Teil; er darf durch keine
Formatumwandlung verloren gehen (GoBD Rz. 119, 125, 131). Buchfink bewahrt das
Original in der empfangenen Form und deckt es über den Beleg-Hash in der
Hash-Kette.

## Hinweis bei Papierrechnungen

Stellt ein inländischer Unternehmer eine Rechnung ohne strukturierten Teil,
zeigt die Buchungsvorschau einen Hinweis, der sich nach dem Belegdatum und den
Übergangsregeln richtet (`service.eInvoiceNotice`). Kein Hinweis erscheint
bei ausländischen Lieferanten, Kleinunternehmern, Privatpersonen, Kleinbeträgen
nach § 33 UStDV und steuerfreien Umsätzen. Blockiert wird nie: ob der
Vorsteuerabzug trägt, ist eine Rechtsfrage, und den Vorjahresumsatz des
Ausstellers kennt Buchfink nicht.

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
