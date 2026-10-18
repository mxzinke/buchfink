# Pflichten und Erklärungen

[Fachkonzepte](README.md) · [Jahresabschluss](jahresabschluss.md)

Anforderungen: UST-03, UST-04, JAB-01, JAB-04, JAB-05, JAB-07.

Buchfink weiß, welche Pflichten eine GmbH oder UG hat, wann sie fällig sind und
ob sie erledigt sind. Es bereitet Voranmeldungen, Jahreserklärungen und
E-Bilanz vor. Übermitteln und bestätigen tut der Mensch.

## Pflichtenkalender

Der Katalog (`service/obligation_catalog.go`) führt 21 Pflichtarten, von der
Umsatzsteuer-Voranmeldung über Vorauszahlungen, Festschreibung, Aufstellung,
Feststellung, Jahreserklärungen und E-Bilanz bis zu Offenlegung,
Transparenzregister, Gesellschafterliste, Gründungsanzeigen und
Kapitalertragsteuer. Jede Art hat:

| Merkmal | Inhalt |
|---|---|
| Rechtsgrundlage und Rhythmus | etwa § 18 UStG, monatlich oder vierteljährlich |
| Weg | Mein ELSTER, BZSt, Unternehmensregister, Transparenzregister, Notar, Gewerbeamt, Lohnbüro oder intern |
| Zuständigkeit | der Agent bereitet vor, der Mensch handelt selbst, oder der Steuerberater erstellt |
| Nachweis | aus den Daten, durch Bestätigung des Menschen oder beides |
| Skill-Verweis | Anker in `buchfink-grundlagen/references/pflichten.md` |

Die Regeln (`service/obligation_rules.go`) rechnen daraus die Pflichten je
Geschäftsjahr und Zeitraum mit ihrer Frist. Fristen, die auf Wochenende oder
Feiertag fallen, verschieben sich auf den nächsten Werktag
([§ 108 Abs. 3 AO][ao108]); die Feiertage richten sich nach dem Bundesland des
Sitzes (`internal/calendar`). Die Dauerfristverlängerung verschiebt
Voranmeldung und Monatsfestschreibung um einen Monat, nicht aber die
Zusammenfassende Meldung.

**Steuerliche Beratung.** In den Unternehmensdaten steht, ob ein Steuerberater
beauftragt ist, ab welchem Jahr und für welche Erklärungen
(`domain.TaxAdvisor`). Ohne Berater gilt für die Jahreserklärungen die Frist des
[§ 149 Abs. 2 AO][ao149] (sieben Monate nach Ablauf des Kalenderjahres), mit
Berater die des § 149 Abs. 3 AO (Ende Februar des zweiten Folgejahres), und die
Pflicht trägt den Hinweis auf die Vorabanforderung nach Abs. 4. Erklärungen, die
der Berater erstellt, stehen in seiner Zuständigkeit.

**Stand.** Eine Pflicht ist offen, vorbereitet, erledigt, überfällig oder
entfällt.

- Aus den Daten erledigt: Voranmeldung und Erklärungen mit Transferticket,
  Festschreibung, Abschlussstand aufgestellt, festgestellt oder offengelegt,
  Vorauszahlungen mit einer Buchung im Zeitfenster um den Termin.
- Vom Menschen in der App bestätigt, mit Nachweis wie Transferticket,
  Aktenzeichen oder Eingangsbestätigung. Ebenso nur in der App: eine Pflicht als
  entfallen kennzeichnen oder eine Bestätigung zurücknehmen.
- Der Agent meldet höchstens „vorbereitet“.

## Jahreserklärungen

`internal/declaration` stellt Körperschaftsteuer (KSt 1 mit Anlagen),
Gewerbesteuer (GewSt 1 A) und Umsatzsteuer (USt 2 A) formularnah zusammen. Das
Paket kennt weder Datenbank noch Dienste; `service.DeclarationService` rechnet
die Erklärung bei jedem Abruf aus dem aktuellen Stand der Bücher.

**Zeilenkatalog.** Zeilen, Kennzahlen und Bezeichnungen stehen je Vordruckjahr
in `internal/declaration/catalog/*.json`, jede Zeile mit Prüfvermerk und
Quelle. Die Rechnung hängt am Schlüssel der Zeile, nicht an ihrer Nummer. Fehlt
der Katalog eines Jahres, nimmt Buchfink den nächstälteren und meldet, dass die
Zeilen am Vordruck des Jahres zu suchen sind.

**Herkunft jedes Werts.** Jede Zeile nennt, woher ihr Wert kommt: Konten,
Buchungen, Rechnung, Stammdaten, Voranmeldungen, Anlagenkartei, Abschluss oder
Eingabe. Und wer sie verantwortet:

| Zuständig | Bedeutung |
|---|---|
| Bücher | Buchfink rechnet den Wert, ohne Wertungsfrage |
| Agent mit Rückfrage | der Agent schlägt vor und setzt den Wert nach Zustimmung im Chat |
| Mensch | nur der Mensch, etwa ein Wert aus einem Bescheid; setzt ihn ein Agent, bleibt er Vorschlag, bis der Mensch ihn in der App bestätigt |

Eine bestätigte Angabe des Menschen ändert nur er. Jede Eingabe steht mit
Quelle (Bescheid, Beschluss, Beleg, Mensch, Bücher) im Änderungsprotokoll.

**Abstimmung.** Die USt 2 A wird gegen die Summe der Voranmeldungen
abgestimmt. Befunde haben die Stufen Hinweis, Warnung und blockiert; für
gemeinnützige, liquidierende und Organgesellschaften blockiert Buchfink die
Erklärung.

**Export und Übermittlung.** Der Export ist eine JSON-Datei nach dem Schema
`buchfink.erklaerung/1` mit Zahlen-Hash. Übertragen wird in Mein ELSTER. Die
Übermittlung erfasst der Mensch in der App mit Datum und Transferticket;
Buchfink speichert dazu den Stand der Zahlen, der übermittelt wurde. Sind die
Bücher seit dem Export geändert, steht das im Protokoll.

## E-Bilanz

`internal/ebilanz` erzeugt die XBRL-Instanz nach [§ 5b EStG][estg5b] aus
derselben Gliederung wie die Bilanz, auf Wunsch zusätzlich eine Summen- und
Saldenliste mit derselben Zuordnung.

- **Taxonomie.** Eingebettet sind Auszüge der HGB-Taxonomie 6.9 und 6.10
  (`internal/ebilanz/taxonomy/`). Ohne Vorgabe nimmt Buchfink die
  Regelfassung des Wirtschaftsjahres; eine gewählte Fassung muss für das
  Wirtschaftsjahr zulässig sein (`ebilanz.ChooseVersion`).
- **Zuordnung.** Jedes Konto bekommt eine Position der Taxonomie. Hat es keine
  eigene Regel, nimmt Buchfink die Auffangposition seiner Gliederungsposition
  und zeigt das an. Ohne Auffangposition blockiert der Export.
- **Inhalt.** Bilanz, GuV (wahlweise in der Form der Kleinstgesellschaft),
  Kontennachweis je Konto, Überleitung bei steuerlichen Abweichungen,
  Ergebnisverwendung. Auch eine Eröffnungsbilanz lässt sich erzeugen.
- **Übermittlung.** Mein ELSTER hat kein Formular für die E-Bilanz. Der
  Mensch lädt die XBRL-Datei bei einem E-Bilanz-Dienst wie ebilanzplus.de oder
  ebilanzonline.de hoch und übermittelt dort. Wahlweise bettet Buchfink die
  Instanz in den Nutzdatenteil einer ELSTER-Übermittlung ein. Die Übermittlung
  bestätigt der Mensch im Pflichtenkalender mit dem Transferticket.
- **Saldenliste als Ausweichweg.** Nimmt der Dienst die XBRL-Datei nicht an,
  liest der Mensch bei eBilanz+ die Summen- und Saldenliste ein (Import
  „Summen- und Saldenliste“, Taxonomie-Layout oder Zuordnung nach SKR04).
  `ebilanz.SuSaCSV` schreibt je Konto mit Saldo eine Zeile
  `Konto;Bezeichnung;Saldo;Taxonomie`: Saldo im Soll positiv, im Haben
  negativ, mit Dezimalkomma; `Taxonomie` ist die Position, unter der das Konto
  im Kontennachweis der Instanz steht. Konten und Salden sind dieselben wie in
  der Instanz, blockierende Befunde verhindern beides. Die Liste füllt dort nur
  Bilanz und GuV; Stammdaten, Anhang und weitere Berichtsteile ergänzt der
  Mensch im Dienst. Weicht die Steuerbilanz ab (Sonderabschreibung nach § 7g
  Abs. 5 EStG, Abzinsung der Rückstellungen, Drohverlustrückstellung nach § 5
  Abs. 4a EStG), ist die Instanz eine Handelsbilanz mit Überleitungsrechnung
  (§ 5b Abs. 1 Satz 2 EStG), und die Liste hat die handelsrechtlichen Salden.
  Die Überleitung überträgt sie nicht: `MappingReport.Transfers` nennt die
  Abweichungen mit Position und Betrag, die App und `ebilanz_exportieren`
  (`ueberleitung`) zeigen sie, und der Mensch trägt sie im Dienst nach.

Die Instanzen lassen sich mit `task test:ebilanz-xsd` gegen die amtlichen
Schemata prüfen. Ob ein E-Bilanz-Dienst die Datei oder die Saldenliste
annimmt und die Finanzverwaltung sie akzeptiert, ist nicht belegt.

[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html
[ao149]: https://www.gesetze-im-internet.de/ao_1977/__149.html
[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
