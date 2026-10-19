# Ablauf: Gründung

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Eine GmbH entsteht in zwei Schritten: Mit der Beurkundung des
Gesellschaftsvertrags beginnt die Vorgesellschaft („GmbH i. G.“), mit der
Eintragung ins Handelsregister die GmbH als juristische Person
([§ 2 Abs. 1 GmbHG][gmbhg2], [§ 11 Abs. 1 GmbHG][gmbhg11]). Dieser Ablauf
führt durch beide Schritte, bucht Zeichnung und Einzahlung des Stammkapitals
und die Gründungskosten und prüft die Eröffnungsbilanz. Behördengänge,
Unterschriften und Zahlungen erledigt der Mensch; du hältst die Liste, die
Fristen liefert `fristen`.

Buchfink führt Bücher ab der Beurkundung. Eine eigene Gründungsansicht hat die
App nicht: Mandant, Gründungsangaben, Ablage und Eingang erfasst der Mensch im
Einrichtungsassistenten, und bis zur Eintragung hängt Buchfink den Zusatz
„i. G.“ an die Firma. Für Zeichnung und Einzahlung gibt es kein eigenes
Werkzeug; du buchst sie nach Rückfrage mit `buchung_erfassen` oder am
Bankumsatz, auch auf die Kapitalkonten.

| Thema | Regel in |
|---|---|
| Notar- und andere Gründungsrechnungen prüfen und buchen | [belege/ablauf.md](../belege/ablauf.md) |
| Kontoauszug importieren, Einzahlung zuordnen | [bank/ablauf.md](../bank/ablauf.md) |
| Voranmeldungen, Dauerfristverlängerung | [monatsabschluss/ustva.md](../monatsabschluss/ustva.md), [monatsabschluss/dauerfristverlaengerung.md](../monatsabschluss/dauerfristverlaengerung.md) |
| Erster Jahresabschluss, UG-Rücklage | [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md), [jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md) |
| E-Bilanz der Eröffnungsbilanz, erste Steuererklärungen | [steuererklaerungen/ebilanz.md](../steuererklaerungen/ebilanz.md), [steuererklaerungen/ablauf.md](../steuererklaerungen/ablauf.md) |
| Darlehen und Zahlungen zwischen Gesellschafter und GmbH | [bank/darlehen.md](../bank/darlehen.md), [gesellschafter/verrechnungskonto.md](../gesellschafter/verrechnungskonto.md) |

Werkzeuge: `status`, `aufgaben`, `fristen`, `eingang`, `beleg_ablegen`,
`buchung_erfassen` (Rückfrage), `bankumsatz_buchen` (Rückfrage), `auswertung`,
`konten`, `buchungen_suchen`.

## Inhalt

1. Phasen
2. Ablauf
3. Entscheidungsregeln
4. Wertungsfragen
5. Nur der Mensch
6. Stolperfallen

## 1. Phasen

| Phase | Zeitraum | Rechtslage | In Buchfink |
|---|---|---|---|
| Vorgründungsgesellschaft | Entschluss bis Beurkundung | eigene Gesellschaft; ihre Verträge gehen nicht von selbst auf die GmbH über ([gruendungskosten.md, Abschnitt 4](gruendungskosten.md#4-kosten-vor-der-beurkundung)) | kein Mandant; Kosten nur nach Klärung übernehmen |
| Vorgesellschaft, GmbH i. G. | Beurkundung bis Eintragung | GmbH besteht als solche noch nicht; wer in ihrem Namen handelt, haftet persönlich ([§ 11 GmbHG][gmbhg11]); körperschaftsteuerpflichtig ([H 1.1 KStH][ksth11]) | erstes Geschäftsjahr beginnt; Zeichnung, Einzahlung, Gründungskosten |
| GmbH | ab Eintragung | juristische Person; für Schulden haftet nur das Gesellschaftsvermögen ([§ 13 Abs. 1, 2 GmbHG][gmbhg13]) | Registerdaten in der App |

## 2. Ablauf

1. **Lage klären.** Frag nach: Rechtsform (GmbH oder UG), Stammkapital,
   Gesellschafter mit Nennbetrag und Einlageart (Geld oder Sache),
   Musterprotokoll ja oder nein, Klausel zum Gründungsaufwand,
   Beurkundungstag, Eintragung schon erfolgt, Geschäftsjahr, Geschäftskonto.
   Fertig, wenn alle Angaben feststehen oder als offen benannt sind.
2. **Mandant anlegen lassen (nur App).** Der Mensch legt den Mandanten im
   Einrichtungsassistenten an und erfasst dort Beurkundung, Stammkapital,
   Gesellschafter, Gründungsaufwand laut Satzung, Beginn des Geschäftsjahres,
   Bundesland, Voranmeldungszeitraum, Hebesatz und ob ein Steuerberater
   beauftragt ist (steuerliche Beratung). Er wählt dort auch Ablage und
   Eingang. Davon rechnet Buchfink die Fristen.
   Prüfe danach mit `status`, dass das erste Geschäftsjahr am Beurkundungstag
   beginnt. Fertig, wenn `status` den Mandanten mit Rechtsform und
   Geschäftsjahr zeigt.
3. **Unterlagen ablegen.** Der Mensch legt Gesellschaftsvertrag oder
   Musterprotokoll, Gesellschafterliste, Bestätigung der Kontoeröffnung, Notar-
   und Gerichtsrechnungen in den Eingang; du übernimmst sie mit
   `beleg_ablegen`. Fertig, wenn jede Unterlage eine `beleg_id` hat.
4. **Zeichnung buchen (Rückfrage).** `buchung_erfassen` am Beurkundungstag mit
   dem Gesellschaftsvertrag als Beleg: 1298 Ausstehende Einlagen, eingefordert
   an 2900 Gezeichnetes Kapital über das Stammkapital. Ist ein Teil nach dem
   Vertrag nicht eingefordert, buche ihn über 2910 statt 1298
   ([kapitalaufbringung.md, Abschnitt 3](kapitalaufbringung.md#3-buchungen)).
   Fertig, wenn die Buchung vorliegt.
5. **Einzahlung buchen (Rückfrage).** Je Gesellschafter am Bankumsatz:
   `bankumsatz_buchen` mit einer Zeile `konto: 1298`.
   Ohne importierten Umsatz: `buchung_erfassen` 1800 an 1298 mit dem
   Kontoauszug als Beleg. Prüfe die Mindesteinzahlung nach
   [kapitalaufbringung.md](kapitalaufbringung.md). Fertig, wenn der
   Saldo von 1298 dem noch offenen eingeforderten Betrag entspricht.
6. **Gründungskosten buchen.** Rechnungen nach
   [belege/ablauf.md](../belege/ablauf.md), Buchungsgruppen nach
   [gruendungskosten.md, Abschnitt 3](gruendungskosten.md#3-buchen). Prüfe
   vorher, ob die GmbH die Kosten nach dem Vertrag tragen darf
   ([gruendungskosten.md, Abschnitt 2](gruendungskosten.md#2-wer-die-gründungskosten-trägt)).
   Fertig, wenn jede Gründungsrechnung gebucht oder als Kosten eines
   Gesellschafters geklärt ist.
7. **Behördenwege begleiten.** `fristen` führt ab der Beurkundung die
   Pflichten [fragebogen-steuerliche-erfassung](../grundlagen/pflichten.md#fragebogen-steuerliche-erfassung)
   und [gewerbeanmeldung](../grundlagen/pflichten.md#gewerbeanmeldung), ab der Eintragung
   [transparenzregister](../grundlagen/pflichten.md#transparenzregister); Frist, Weg und Nachweis
   stehen dort. Die übrigen Wege (Berufsgenossenschaft, Gemeinde, IHK) und die
   Angaben zum Fragebogen: [gruendungsweg.md](gruendungsweg.md),
   [steuerliche-erfassung.md](steuerliche-erfassung.md). Fertig,
   wenn jede Pflicht in `fristen` erledigt ist oder einen Termin hat und die
   übrigen Wege ein Datum haben.
8. **Eröffnungsbilanz prüfen.** Salden der Gründungskonten zum
   Beurkundungstag mit `auswertung` (`art: kontoblatt`, `konto`,
   `bis: <Beurkundungstag>`); stehen nur Gründungsbuchungen im Jahr, genügt
   `auswertung` (`art: bilanz`). Fertig, wenn Aktiva und Passiva
   übereinstimmen und der Mensch sie gesehen hat.
   [eroeffnungsbilanz.md](eroeffnungsbilanz.md)
9. **Eintragung nachhalten (nur App).** Der Mensch trägt Datum, Registergericht
   und Nummer in den Unternehmensdaten ein. Prüfe danach die Unterbilanz
   ([kapitalaufbringung.md](kapitalaufbringung.md), Abschnitt 6) und
   mit `fristen`, dass die Mitteilung an das Transparenzregister ansteht.
   Fertig, wenn die Eintragung in der App erfasst und die Unterbilanz geprüft
   ist.
10. **Erstes Geschäftsjahr planen.** Lies
    [rumpfjahr.md](rumpfjahr.md) und nenne dem Menschen Ende des
    Rumpfjahres, Fristen des ersten Abschlusses und der ersten Erklärungen.
    Fertig, wenn `fristen` diese Pflichten mit Fälligkeit zeigt.

## 3. Entscheidungsregeln

- Die Einlage zählt nur, wenn sie endgültig zur freien Verfügung der
  Geschäftsführer steht ([§ 7 Abs. 3 GmbHG][gmbhg7], [§ 8 Abs. 2 Satz 1 GmbHG][gmbhg8]). Sie
  geht vom Gesellschafter auf ein Konto der Gesellschaft.
- Buche keine Zahlung an einen Gesellschafter, die wirtschaftlich die Einlage
  zurückführt, und keinen Kauf von einem Gesellschafter aus der Einlage, ohne
  Rückfrage. Warum:
  [kapitalaufbringung.md, Abschnitt 5](kapitalaufbringung.md#5-verdeckte-sacheinlage-und-hin--und-herzahlen).
- Eine Sacheinlage buchst du nur, wenn Gegenstand und Nennbetrag im Vertrag
  stehen; den Wert legt der Mensch mit Beleg vor. Regeln und UG:
  [kapitalaufbringung.md, Abschnitte 2 und 4](kapitalaufbringung.md#4-sacheinlage).
- Kosten vor der Beurkundung buchst du nicht ohne Klärung in die GmbH
  ([gruendungskosten.md, Abschnitt 4](gruendungskosten.md#4-kosten-vor-der-beurkundung)).
- Unternehmensstammdaten, Registerdaten und Bankkonten ändert nur der Mensch
  in der App ([SKILL.md, Abschnitt 4](../../SKILL.md#4-nur-der-mensch-in-der-app)).

## 4. Wertungsfragen

Format: [SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen).

- Wer trägt Kosten aus der Zeit vor der Beurkundung
  ([gruendungskosten.md, Abschnitt 4](gruendungskosten.md#4-kosten-vor-der-beurkundung))?
- Trägt die GmbH Gründungskosten über die Klausel im Vertrag hinaus, oder ohne
  Klausel ([gruendungskosten.md, Abschnitt 2](gruendungskosten.md#2-wer-die-gründungskosten-trägt))?
  Folge: möglicherweise verdeckte Gewinnausschüttung
  ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md)).
- Ist eine Sacheinlage richtig bewertet
  ([kapitalaufbringung.md, Abschnitt 4](kapitalaufbringung.md#4-sacheinlage))?
- Ist bei einem rein freiberuflichen Unternehmensgegenstand ein Gewerbe
  anzumelden ([pflichten.md#gewerbeanmeldung](../grundlagen/pflichten.md#gewerbeanmeldung))?
- Steht der GmbH die Vorsteuer aus Gründungsrechnungen zu
  ([gruendungskosten.md, Abschnitt 3](gruendungskosten.md#3-buchen))?
- Wann genau beginnt das Handelsgewerbe für die Eröffnungsbilanz
  ([eroeffnungsbilanz.md, Abschnitt 2](eroeffnungsbilanz.md#2-stichtag-in-buchfink))?
- Welcher Voranmeldungszeitraum gilt im Jahr nach einer Gründung 2026
  ([steuerliche-erfassung.md, Abschnitt 3](steuerliche-erfassung.md#3-voranmeldungszeitraum))?

## 5. Nur der Mensch

- Beurkundung, Unterschriften, Anmeldung zum Handelsregister über den Notar.
- Geschäftskonto eröffnen, Einlage einzahlen.
- Fragebogen zur steuerlichen Erfassung, Gewerbeanzeige, Mitteilung an das
  Transparenzregister, Meldung an die Berufsgenossenschaft, ELSTER-Registrierung.
- Mandant, Unternehmensstammdaten, Bankkonten, Registerdaten, Ablage und
  Eingang in der App.
- Entscheidung über Sacheinlage, Gründungskosten und Vorgründungskosten.

## 6. Stolperfallen

- Bis zur Eintragung haften die Handelnden persönlich
  ([§ 11 Abs. 2 GmbHG][gmbhg11]). Erinnere daran, wenn vor der Eintragung
  größere Verträge geschlossen werden.
- Unterbilanz am Eintragungstag:
  [kapitalaufbringung.md, Abschnitt 6](kapitalaufbringung.md#6-unterbilanz-am-eintragungstag).
- Gründungsaufwand ist Aufwand des ersten Jahres, das Musterprotokoll begrenzt
  ihn: [gruendungskosten.md](gruendungskosten.md).
- Treib den Fragebogen zur steuerlichen Erfassung früh voran: Ohne
  Steuernummer fehlt jeder Ausgangsrechnung eine Pflichtangabe
  ([pflichten.md#fragebogen-steuerliche-erfassung](../grundlagen/pflichten.md#fragebogen-steuerliche-erfassung)).
- Voranmeldungszeitraum nach einer Gründung:
  [steuerliche-erfassung.md, Abschnitt 3](steuerliche-erfassung.md#3-voranmeldungszeitraum).
- Die UG stellt jedes Jahr einen Teil des Jahresüberschusses in die
  gesetzliche Rücklage
  ([jahresabschluss/ergebnisverwendung-ug.md](../jahresabschluss/ergebnisverwendung-ug.md)).
- Auch die Eröffnungsbilanz ist als E-Bilanz zu übermitteln
  ([eroeffnungsbilanz.md, Abschnitt 5](eroeffnungsbilanz.md#5-unterzeichnen-aufbewahren-übermitteln)).

[gmbhg2]: https://www.gesetze-im-internet.de/gmbhg/__2.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg8]: https://www.gesetze-im-internet.de/gmbhg/__8.html
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg13]: https://www.gesetze-im-internet.de/gmbhg/__13.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
