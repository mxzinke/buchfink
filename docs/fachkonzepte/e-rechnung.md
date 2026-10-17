# E-Rechnung

[Fachkonzepte](README.md) · [Belege und Buchungen](beleg-buchungsflow.md) ·
[Modulaufbau](../entwicklung/e-rechnung.md)

Anforderungen: RECH-05 bis RECH-07, RECH-10, UST-07, ARC-01, ARC-07.

## Pflichten

Seit dem 1. Januar 2025 ist zwischen inländischen Unternehmern grundsätzlich
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

- ZUGFeRD, Factur-X und XRechnung in den Syntaxen CII und UBL werden erkannt
  und gelesen, bei hybriden PDFs aus dem eingebetteten XML.
- Geprüft wird gegen alle 223 Geschäftsregeln von EN 16931, eine XRechnung
  zusätzlich gegen die deutschen Regeln. Ergebnis, Zeitpunkt und Regelstand
  stehen am Beleg. Die Prüfung ändert keine Datei und damit nicht den
  Beleg-Hash.
- Die Profile MINIMUM und BASIC WL enthalten keine vollständige Rechnung und
  gelten nicht als E-Rechnung (UStAE 14.1 Abs. 14).
- Aus dem Datensatz entsteht ein Buchungsvorschlag: Lieferant über USt-IdNr. oder
  Steuernummer, Beträge und Sätze aus dem XML. Den Steuerkategoriecode dreht
  Buchfink in die Sicht des Empfängers: „K“ ist beim Lieferanten eine
  innergemeinschaftliche Lieferung und beim Empfänger ein Erwerb, „AE“ heißt
  Steuerschuld beim Empfänger. Die Buchungsgruppe wählt der Mensch oder der
  Agent; keine Rechnung sagt, welches Aufwandskonto zutrifft.
- Gutschrift, Korrektur, Anzahlungs- und Abschlagsrechnung erkennt Buchfink am
  Rechnungstyp und schlägt dafür keine Buchung vor.
- Mitgeschickte Anlagen landen als Belegdateien am Beleg.

Ein eigenes Postfach braucht es nicht; es genügt, die empfangene Datei
abzulegen (UStAE 14.1 Abs. 5).

## Ausstellung

Ausgangsrechnungen erzeugt Buchfink als ZUGFeRD im Profil EN 16931 (PDF/A-3b mit
eingebettetem XML) oder als XRechnung im CII-Format. Gesetzt wird mit Typst als
WebAssembly im Prozess. Eine Rechnung, die gegen die Regeln verstößt, entsteht
gar nicht erst; eine XRechnung verlangt die Leitweg-ID des Empfängers. Vor der
Ablage liest Buchfink den Datensatz aus dem erzeugten Dokument zurück und
vergleicht ihn mit der Rechnung (`invoice/hybrid_check.go`), weil PDF und XML
dasselbe sagen müssen (UStAE 14c.1). Das hybride PDF wird als Ausgangsbeleg
abgelegt, obwohl GoBD Rz. 76 den Verzicht erlaubt: ein gespeichertes Dokument
ist greifbarer als die Zusage, es jederzeit neu zu erzeugen.

## Aufbewahrung

Bei E-Rechnungen genügt der strukturierte Teil; er darf durch keine
Formatumwandlung verloren gehen (GoBD Rz. 119, 125, 131). Buchfink bewahrt das
Original in der empfangenen Form und deckt es über den Beleg-Hash in der
Hash-Kette.

## Hinweis bei Papierrechnungen

Stellt ein inländischer Unternehmer eine Rechnung ohne strukturierten Teil,
zeigt die Buchungsvorschau einen Hinweis, der sich nach dem Belegdatum und den
Übergangsregeln richtet (`service/einvoice_notice.go`). Kein Hinweis erscheint
bei ausländischen Lieferanten, Kleinunternehmern, Privatpersonen, Kleinbeträgen
nach § 33 UStDV und steuerfreien Umsätzen. Blockiert wird nie: ob der
Vorsteuerabzug trägt, ist eine Rechtsfrage, und den Vorjahresumsatz des
Ausstellers kennt Buchfink nicht.

[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
