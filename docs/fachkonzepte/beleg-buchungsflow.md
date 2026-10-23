# Belege und Buchungen

[Fachkonzepte](README.md) · [Buchungsbeispiele](buchungsbeispiele.md) ·
[Zahlungswege](zahlungswege.md)

Anforderungen: GOB-01 bis GOB-06, BEL-01 bis BEL-09, UNV-01, UNV-02, UST-01,
UST-05. Kontenrahmen: DATEV SKR04 2026 (`internal/accounting/skr04_2026.json`).

## Leitgedanke

Wer bucht, denkt in Belegen, Rechnungen und Zahlungen. Buchfink übersetzt sie
nach festen Regeln in Soll und Haben. Die Kontierung folgt aus Buchungsgruppe,
Steuerfall, Steuersatz und Zahlungsweg; sie lernt nicht und rät nicht. Jede
Buchung speichert die Fassung dieses Regelwerks (`buildinfo.Program`), damit
eine alte Buchung erklärbar bleibt.

Ein erfasster Beleg wird sofort gebucht. Ist er nicht bezahlt, entsteht ein
offener Posten auf dem Personenkonto. Die Zahlung ist ein eigener
Geschäftsvorfall. Eine gebuchte Buchung ändert niemand; korrigiert wird durch
Storno und neue Buchung.

## Was eine Buchung braucht

| Angabe | Wofür |
|---|---|
| Richtung | Aufwand oder Ertrag |
| Geschäftspartner | Personenkonto, offener Posten, USt-IdNr. |
| Belegdatum | Entstehung der Umsatzsteuer, Vorsteuerabzug |
| Leistungsdatum oder -zeitraum | Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG, Entstehung der Steuer bei § 13b Abs. 1 UStG, Abgrenzung |
| Nettobeträge je Steuersatz | ein Beleg kann 19 % und 7 % enthalten |
| Steuerfall | Vorsteuer- und Umsatzsteuerkonto, Kennzahl der Voranmeldung |
| Buchungsgruppe | Aufwands- oder Erlöskonto |
| Zahlungsweg | offen auf Ziel, bezahlt per Bank, Firmenkreditkarte oder Auslage ([Zahlungswege](zahlungswege.md)) |

Das Buchungsdatum ordnet die Buchung einem Geschäftsjahr zu. Das Jahr folgt
immer aus dem Datum, nie aus einer Eingabe.

## Steuerfälle

Die Liste ist geschlossen (`domain.TaxTreatment`). Jeder Fall ist für Buchung,
Rechnungstext und Voranmeldung durchgängig umgesetzt.

| Steuerfall | Richtung | Steuerzeilen | Kennzahl |
|---|---|---|---|
| Inland 19 % oder 7 % | beide | eine: Vorsteuer 1406/1401 oder Umsatzsteuer 3806/3801 | 81, 86, 66 |
| § 13b Abs. 2 UStG als Empfänger | Eingang | zwei: Vorsteuer 1407/1408, Umsatzsteuer 3837/3835 | 84, 85, 67 |
| § 13b Abs. 1 UStG, sonstige Leistung aus der EU | Eingang | zwei, dieselben Konten | 46, 47, 67 |
| Innergemeinschaftlicher Erwerb | Eingang | zwei: Vorsteuer 1404/1402, Umsatzsteuer 3804/3802 | 89, 93, 61 |
| Leistung mit Steuerschuld des Empfängers | Ausgang | keine; Erlöse auf 4337 | 21 (EU) oder 60 (Inland) |
| Nullsteuersatz § 12 Abs. 3 UStG | beide | keine; Erlöse auf 4290 | 87 |
| steuerfrei, nicht steuerbar | beide | keine | 48, 45 |

Beide Absätze des § 13b buchen auf dieselben Steuerkonten. Der Steuerschlüssel
an der Zeile trennt sie, weil Kennzahl und Entstehung verschieden sind
(`accounting.SKR04TaxResolver`).

Der Nullsteuersatz ist ein eigener Steuerfall und kein Steuersatz von null: der
Umsatz ist steuerpflichtig, der Leistende behält den Vorsteuerabzug. Ein
steuerpflichtiger Inlandsumsatz ohne Satz wird abgewiesen.

Steuerkonten bebucht nur die Steuerautomatik. Eine Handbuchung darauf würde die
Voranmeldung vom Journal trennen; der Buchungskern weist sie ab.

**Mindest-Istversteuerung.** Wird das Entgelt vor der Leistung vereinnahmt,
entsteht die Steuer mit Ablauf des Voranmeldungszeitraums der Zahlung
(§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG). Ein im Januar bezahltes Jahresabo
gehört deshalb in die Voranmeldung für Januar, nicht in die des Leistungsendes.
Buchfink leitet das aus Rechnung und Zahlung ab (`accounting/vat_prepayment.go`).

## Konten

Jeder Geschäftspartner hat ein eigenes Personenkonto: Debitoren 10000 bis
69999, Kreditoren 70000 bis 99999. 1200 und 3300 sind Bilanzpositionen und
keine Buchungsziele; der Kern weist Zeilen darauf ab. So gibt es für
Forderungen und Verbindlichkeiten nur eine Wahrheit.

Gesperrt sind außerdem Bereichskonten wie `4400-4409`, die Kontenklasse 8 und
Konten, die zur Rechtsform nicht passen: eine GmbH bucht keine Privatentnahme
(`accounting/legal_form_accounts.go`).

## Der Beleg

Ein Beleg ist eine eigene Entität mit einer oder mehreren Dateien. Jede Datei
hat eine Rolle:

| Rolle | Bedeutung |
|---|---|
| `original` | die Datei, wie sie empfangen wurde; genau eine je Beleg |
| `structured` | der Rechnungsdatensatz, bei ZUGFeRD aus dem PDF gelesen, bei XRechnung gleich dem Original |
| `rendering` | eine Darstellung, die Buchfink erzeugt, wenn das Original keine hat |
| `attachment` | Anlage, etwa ein Stundenzettel aus der E-Rechnung |

Eingehende Belege bleiben in der empfangenen Form ([GoBD Rz. 131][gobd]). Die
Originaldatei lässt sich nach dem Ablegen nicht mehr entfernen.

**Ablegen und Buchen sind zwei Schritte.** Eine XRechnung ist sofort ablegbar,
buchbar erst mit einer Darstellung und vollständigen Kopfdaten, etwa
Belegdatum, Aussteller und Betrag. Bei einer E-Rechnung kommen die Kopfdaten aus dem
Datensatz; gebucht wird aus dem strukturierten Teil, nie aus dem Bild.

**Eingang.** Vor der Ablage liegen Belege, E-Rechnungen und Kontoauszüge im
Eingang des Mandanten (`internal/inbox`). Dort ist eine Datei noch kein Beleg:
sie steht in keiner Sicherung, und der Mensch kann sie löschen. Abgelegt wird
auf seine Anweisung, durch den Agenten oder in der App. Danach verschiebt
Buchfink die Datei in `Übernommen/<JJJJ-MM>/` des Eingangs; maßgeblich bleibt
die Kopie im Archiv.

**Ablage.** Dateien liegen unter `belege/<Jahr>/eingang|ausgang/`, benannt nach
ihrer SHA-256-Prüfsumme mit der Originalendung. Geschrieben wird in eine
Temporärdatei, die nach dem Schreiben umbenannt wird; ein Abbruch hinterlässt
keine halbe Datei.

**Beleg-Hash.** Über die geordnete Dateiliste (Rolle, Name, Prüfsumme) und die
Kopfdaten bildet Buchfink einen Hash (`accounting.ReceiptHash`), der in die
Buchung eingeht. Mit der Buchung ist der Beleg versiegelt. Was später dazukommt,
etwa ein Zahlungsnachweis, ist ein eigener Beleg. Ein Beleg, der einen Datensatz
statt einer Buchung belegt, etwa der Vertrag eines Gesellschafterdarlehens, ist
diesem zugeordnet (`assigned`): er braucht keine Buchung, seine Dateien sind
ebenso eingefroren. Ob ein Beleg noch gebucht werden muss, sagt
`Receipt.NeedsBooking`.

**Keine Buchung ohne Beleg.** Eine Buchung verweist auf einen abgelegten Beleg.
Fehlt ein Fremdbeleg, entsteht ein Eigenbeleg mit Aussteller, Datum, Betrag,
Grund und Bearbeiter als PDF (`ReceiptService.CreateSelfIssued`). Folgebuchungen
wie AfA, Zahlung, Saldenvortrag und Abschlussbuchungen haben ihren Nachweis in
der Buchung, auf die sie sich beziehen.

**Hinweis auf die E-Rechnungspflicht.** Kommt von einem inländischen
Unternehmer eine Rechnung ohne Datensatz, zeigt Buchfink einen Hinweis und
blockiert nicht. Der Text richtet sich nach dem Belegdatum und den Übergangsregeln
des [§ 27 Abs. 38 UStG][ustg27].

## Zahlung zuordnen

Eine Zuordnung ist ein eigener Datensatz. Eine Zahlung kann einen Beleg
ausgleichen, mehrere (Sammelzahlung) oder einen Teil (Rate). Der Status
`offen`, `teilbezahlt`, `bezahlt` folgt aus dem Saldo. Bei einem Bankumsatz
muss die Summe der Zuordnungen dem Kontoauszug entsprechen; ein vertipptes
Skonto wird zur Fehlermeldung.

Offene Posten kennen keinen Jahreswechsel. Die Rechnung vom Dezember wird im
Januar bezahlt; die Liste zeigt die Posten aller Jahre, und eine stornierte
Zahlung zählt nicht als Ausgleich.

| Differenz | Behandlung |
|---|---|
| Skonto | mindert Entgelt und Steuer nach § 17 UStG; bei § 13b und Erwerb werden beide Steuerzeilen berichtigt |
| Bankgebühr | Aufwand auf 6855; beim Zahlungsausgang zusätzlich zum Rechnungsbetrag abgebucht, beim Zahlungseingang vom Betrag abgezogen |
| Kursdifferenz | realisierter Gewinn oder Verlust beim Ausgleich |
| Zahlungsausfall | Ausbuchung mit Berichtigung der Umsatzsteuer (`service.PaymentService.WriteOffOpenItem`); weder App noch MCP bieten sie an |

Ein Bankumsatz ohne Beleg, etwa Zinsen oder eine Steuerlastschrift, wird gegen
ein Konto gebucht oder auf mehrere aufgeteilt. Der Import schlägt dafür kein
Konto aus dem Verwendungszweck vor.

## Unveränderbarkeit

- **Hash-Kette.** Jede Buchung enthält den SHA-256-Hash der vorigen. Gehasht
  wird eine kanonische, längenpräfigierte Form aller buchungsrelevanten Felder,
  einschließlich Erfassungsweg, Bearbeiter, Programmfassung und Steuerfall.
  Der Dateipfad des Belegs gehört nicht dazu.
- **Nummern.** Buchungen, Belege und Rechnungen haben je einen Zähler je
  Geschäftsjahr, vergeben in derselben Transaktion wie der Datensatz.
- **Generalumkehr.** Ein Storno bucht dieselben Konten auf denselben Seiten mit
  negativem Betrag, datiert auf den Tag der Korrektur. Die Verkehrszahlen gehen
  auf null zurück. Eine Buchung wird höchstens einmal storniert.
- **Festschreibung.** Ein festgeschriebener Zeitraum nimmt keine Buchung mehr
  an. Bis zur Jahresfestschreibung bleibt die Abschlussperiode zum
  Bilanzstichtag offen ([Jahresabschluss](jahresabschluss.md)).
- **Exakte Beträge.** Ganzzahlige Cent, gerundet einmal je Steuersatzgruppe.

## Nicht abziehbare Betriebsausgaben

| Fall | Konten |
|---|---|
| Bewirtung: 70 % abziehbar, Vorsteuer voll ([§ 4 Abs. 5 Satz 1 Nr. 2 EStG][estg4], § 15 Abs. 1a Satz 2 UStG) | **6640** abziehbar, **6644** nicht abziehbar |
| Geschenke über 50 € je Empfänger und Jahr: weder Aufwand noch Vorsteuer | **6610** abziehbar, **6620** nicht abziehbar |

Eine Bewirtung erzeugt zwei Aufwandszeilen; der nicht abziehbare Teil ist die
Differenz, damit nicht zweimal gerundet wird. Die Grenzen stehen datiert in
`accounting/tax_params.go`. Ort, Tag, Teilnehmer und Anlass der Bewirtung stehen
an der Buchung und damit unter der Hash-Kette.

[gobd]: https://amtliche-handbuecher.bundesfinanzministerium.de/ao/2025/Anhaenge/BMF-Schreiben-und-gleichlautende-Laendererlasse/Anhang-33/anhang-33.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
