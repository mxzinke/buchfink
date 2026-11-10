# Umfang und Grenzen

[Dokumentation](../README.md) · [Erste Schritte](README.md) ·
[Umsetzungsstand](../projekt/umsetzungsstand.md)

## Für wen Buchfink gebaut ist

- GmbH und UG (haftungsbeschränkt) mit Sitz in Deutschland, doppelte
  Buchführung nach HGB im Kontenrahmen SKR04.
- Dienstleistung, Software, Agentur, Projektgeschäft und Holding.
- Kleinst- und kleine Kapitalgesellschaften. Buchfink berechnet die
  Größenklasse und meldet im Prüflauf, wenn die Gesellschaft an zwei Stichtagen
  in eine andere Klasse fällt.
- Mit und ohne Steuerberater. In den Unternehmensdaten steht, ob ein Berater
  beauftragt ist, ab welchem Jahr und für welche Erklärungen. Danach richten
  sich die Abgabefristen ([§ 149 Abs. 2 und 3 AO][ao149]) und wer eine
  Erklärung erstellt.
- Geschäftsjahr gleich Kalenderjahr oder abweichend, mit Rumpfjahr bei Gründung
  oder Umstellung.

## Was Buchfink abdeckt

| Bereich | Umfang |
|---|---|
| Belege | Ein Eingang je Unternehmen für Belege, E-Rechnungen und Kontoauszüge, auch per Drag & Drop ins Fenster. Ablage im Original, Kopfdaten, Eigenbeleg. E-Rechnungen (ZUGFeRD, Factur-X, XRechnung in CII und UBL) liest und prüft Buchfink. |
| Zahlungswege | Bankkonten mit Import von CAMT.053, Firmenkreditkarte, Auslage durch Gesellschafter oder Geschäftsführer, Gesellschafterdarlehen. Ein Bankumsatz lässt sich auf mehrere Gegenkonten aufteilen, etwa eine Dividende mit Kapitalertragsteuer oder eine Darlehensrate mit Zins. |
| Ausgangsrechnungen | Stornorechnung und Berichtigung mit Vorschau in App und Werkzeug; ZUGFeRD und XRechnung im CII-Format, überfällige Posten mit Verzugszinsen. |
| Lohn | Das Lohnjournal eines externen Lohnbüros als Monatssumme buchen und die Lohnkonten abstimmen. |
| Umsatzsteuer | Voranmeldung mit Rückverfolgung bis zur Buchung, Dauerfristverlängerung, Sondervorauszahlung, Zusammenfassende Meldung für sonstige Leistungen, Mindest-Istversteuerung bei Zahlung vor der Leistung, Verzeichnis nach § 15a UStG, Bestätigung von USt-IdNrn. |
| Anlagen | Verkauf, Verschrottung, Fertigstellung, Wertminderung, Zuschreibung und Fremdwährungs-Finanzanlagen in App und Werkzeug; Storno stellt die Kartei wieder her. Beschreibende Stammdaten mit Grund und Änderungsprotokoll in der App pflegen; Stückzahlen nur vor Buchung und Folgebewegungen in offener Periode. Anlagenkartei, lineare und degressive Abschreibung, geringwertige Wirtschaftsgüter, Sammelposten, Sonderabschreibung nach § 7g EStG. |
| Jahresabschluss | Anhangtexte je Geschäftsjahr in der App, Entwürfe durch Agenten nach Rückfrage, Freigabe beim Aufstellen. Abschlussperiode zwischen Dezember- und Jahresfestschreibung, Bausteine von der Abschreibung über Wertberichtigung, unfertige Leistungen, Abgrenzung und Rückstellungen bis zur Steuerrückstellung, Bilanz und GuV nach §§ 266, 275 HGB, gesetzliche Rücklage der UG, Ergebnisverwendung und Saldenvortrag. |
| Erklärungen | Körperschaftsteuer, Gewerbesteuer und Umsatzsteuer-Jahreserklärung formularnah mit der Herkunft jedes Werts, als Eingabehilfe für Mein ELSTER und als JSON. E-Bilanz als XBRL nach der amtlichen Taxonomie 6.9 oder 6.10, ersatzweise als Summen- und Saldenliste für eBilanz+. |
| Pflichten | Pflichtenkalender mit Voranmeldungen, Vorauszahlungen, Festschreibung, Abschluss, Erklärungen, Offenlegung und Registerpflichten. Fristen verschieben sich nach Wochenende und Feiertagen des Bundeslands ([§ 108 Abs. 3 AO][ao108]). |
| Nachweise | Hash-Ketten, Festschreibung mit Zeitstempel, Änderungsprotokoll, Prüfermodus, Datenüberlassung für die Betriebsprüfung, Verfahrensdokumentation. |

Welche Teile gebaut und geprüft sind, steht im
[Umsetzungsstand](../projekt/umsetzungsstand.md).

## Forderungen ausbuchen

Uneinbringliche Forderungen lassen sich unter Bücher › Konten › Offene Posten
und mit `forderung_ausbuchen` nach Rückfrage ganz oder teilweise ausbuchen.
Grund, Nachweis und Eintrittsdatum sind Pflicht. Die Vorschau zeigt die
Steuerkorrektur und die Auflösung einer zugeordneten Einzelwertberichtigung.
Ein geführter Ablauf für spätere Zahlung auf ausgebuchte Forderungen fehlt.
Einzelwertberichtigungen ohne Forderungszuordnung sperren die Ausbuchung;
die Zuordnung muss fachlich geklärt werden.

## Was Buchfink nicht abdeckt

| Thema | Was gilt |
|---|---|
| Rechtsform | Nur GmbH und UG. Keine Einzelunternehmen, Personengesellschaften oder AG, keine Einnahmen-Überschuss-Rechnung, kein Konzernabschluss. |
| Besondere Gesellschaften | Für gemeinnützige Gesellschaften, Gesellschaften in Liquidation und Organgesellschaften rechnet Buchfink weder Abschluss noch Steuerrückstellung. Das gehört zu einem Steuerberater. |
| Besteuerungsart | Nur Sollversteuerung. Ist die Istversteuerung eingetragen, weist Buchfink neue Buchungen ab. |
| Steuerfälle | Unterstützt sind Inland mit 19 % und 7 %, Nullsteuersatz nach § 12 Abs. 3 UStG, steuerfrei, nicht steuerbar, § 13b UStG als Empfänger, innergemeinschaftlicher Erwerb und sonstige Leistungen mit Steuerschuld des Empfängers. Nicht unterstützt: eigene Kleinunternehmerregelung, OSS und IOSS, Differenzbesteuerung, Reiseleistungen. |
| Warenverkehr | Keine innergemeinschaftliche Lieferung, keine Ausfuhr, keine Vorräte und keine Inventur. |
| Genossenschaftsanteile | Erträge aus Anteilen an einer eingetragenen Genossenschaft, etwa einer Volksbank, bucht Buchfink nicht von selbst und schlägt kein Ertragskonto vor. Die Mitgliedschaft gilt nicht als Beteiligung ([§ 271 Abs. 1 Satz 5 HGB][hgb271]); die Buchung legt der Agent Ihnen zur Rückfrage vor, im Zweifel klären Sie sie mit dem Steuerberater. |
| Kasse | Kein Kassenbuch. Jede Zahlung läuft über ein Bankkonto, die Firmenkreditkarte oder eine Auslage. |
| Lohnabrechnung | Buchfink rechnet keine Löhne ab. Lohnsteuer-Anmeldung und Beitragsnachweise kommen vom Lohnbüro. |
| Übermittlung | Buchfink übermittelt nichts an Finanzamt oder Register. Voranmeldung, Zusammenfassende Meldung und Jahreserklärungen übertragen Sie selbst, etwa in Mein ELSTER. Die E-Bilanz übermitteln Sie über einen E-Bilanz-Dienst wie ebilanzplus.de oder ebilanzonline.de, weil Mein ELSTER kein Formular dafür hat ([E-Bilanz übermitteln](README.md#e-bilanz-übermitteln)). Die Offenlegung geschieht im Unternehmensregister. |
| Automatischer Import | Buchfink beobachtet keinen Ordner und holt keine Belege aus Mailpostfach oder Portalen. Was im Eingang liegt, legt der Agent ab, wenn Sie es sagen. |
| Verschlüsselung | Buchfink verschlüsselt nichts, siehe [Datensicherheit](datensicherheit.md). |
| Mehrere Bearbeiter | Keine Benutzerverwaltung. Buchfink läuft an einem Rechner, ein Mandant hat einen Schreiber. |

Buchfink ersetzt keine steuerliche Beratung. Für Gestaltungsfragen, Bescheide
und alles außerhalb der Tabelle oben ist ein Steuerberater der richtige Weg.

Die ZM-Ansicht erkennt Änderungen an bestätigten Meldungen, zeigt alte und neue
Beträge und legt Berichtigungen mit Kenntnisdatum und Monatsfrist an. Erstmeldung
und Berichtigung lassen sich als BOP-CSV exportieren; die Übermittlung bestätigt
weiterhin der Mensch. Ohne Kenntnisdatum bleibt die datierte Berichtigungsfrist
offen. Die Erzeugung aus Buchungen umfasst weiterhin sonstige Leistungen.


[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[hgb271]: https://www.gesetze-im-internet.de/hgb/__271.html

### Übertragung von Anlagen an Gesellschafter

Für unentgeltliche oder verbilligte Übertragungen von Anlagen an Gesellschafter
besteht kein eigener Erfassungsweg. Eine mögliche verdeckte Gewinnausschüttung
([§ 8 Abs. 3 Satz 2 KStG](https://www.gesetze-im-internet.de/kstg_1977/__8.html))
und eine unentgeltliche Wertabgabe
([§ 3 Abs. 1b UStG](https://www.gesetze-im-internet.de/ustg_1980/__3.html))
müssen im Einzelfall beurteilt werden. Ein gewöhnlicher Verkauf an einen
Gesellschafter wird mit Rechnung erfasst. Eine Abgangsart „Entnahme“ gibt es nicht.

## Rückstellungen fortführen

App und `abschlussbuchung` bieten Zuführung, Auflösung und Aufzinsung mit
Vorschau. `abschluss_status` liefert Bestände und Spiegel über die Jahre.
Verbrauch ist beim Buchen einer Eingangsrechnung mit Rückstellungszuordnung möglich.

Grenzen: Die automatische Aufzinsung verwendet den gespeicherten Satz.
Neubewertung bei geändertem Marktzins, gebrochene Laufzeiten und steuerliche
Ausnahmen sind gesondert zu prüfen. Einzelheiten und Quellen:
[Rückstellungen fortführen](../fachkonzepte/jahresabschluss.md#rückstellungen-fortführen).
