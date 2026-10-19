---
name: buchfink-gruendung
description: "Gründung einer GmbH oder UG mit Buchfink begleiten und die Bücher eröffnen: Weg vom Notar bis zur Eintragung, Behördenwege (Finanzamt, Gewerbeamt, Transparenzregister, Berufsgenossenschaft, ELSTER), Kapitalaufbringung mit Mindesteinzahlung, Sacheinlage und Hin- und Herzahlen, Gründungskosten, Vorgründungs- und Vorgesellschaft, Eröffnungsbilanz, Rumpfgeschäftsjahr, Fragebogen zur steuerlichen Erfassung, Voranmeldungszeitraum. Lade den Skill, wenn eine GmbH oder UG gegründet wird oder gerade gegründet wurde; wenn Stammkapital eingezahlt oder gebucht werden soll; wenn Notar-, Gerichts- oder andere Gründungskosten anfallen; wenn Kosten vor der Beurkundung entstanden sind; wenn die Eröffnungsbilanz zu prüfen ist; wenn das erste Geschäftsjahr kürzer als zwölf Monate ist; wenn nach Steuernummer, USt-IdNr., Gewerbeanmeldung oder Transparenzregister gefragt wird oder fristen eine dieser Pflichten meldet. Laufende Belege regelt buchfink-belege, den ersten Jahresabschluss buchfink-jahresabschluss."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem zugelassenen Mandanten. Setzt buchfink-grundlagen voraus. Mandant, Gründungsangaben, Ablage und Eingang legt der Mensch im Einrichtungsassistenten der App an."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Gründung

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Eine GmbH entsteht in zwei Schritten: Mit der Beurkundung des
Gesellschaftsvertrags beginnt die Vorgesellschaft („GmbH i. G.“), mit der
Eintragung ins Handelsregister die GmbH als juristische Person
([§ 2 Abs. 1 GmbHG][gmbhg2], [§ 11 Abs. 1 GmbHG][gmbhg11]). Dieser Skill
führt durch beide Schritte, bucht Zeichnung und Einzahlung des Stammkapitals
und die Gründungskosten und prüft die Eröffnungsbilanz. Behördengänge, Unterschriften
und Zahlungen erledigt der Mensch; du hältst die Liste, die Fristen liefert
`fristen`.

Buchfink führt Bücher ab der Beurkundung. Die Gründungsansicht der App ist
entfallen; geblieben sind die Gründungsangaben im Einrichtungsassistenten und
der Zusatz „i. G.“, den Buchfink bis zur Eintragung an die Firma hängt. Für
Zeichnung und Einzahlung gibt es kein eigenes Werkzeug; du buchst sie nach
Rückfrage mit `buchung_erfassen` oder am Bankumsatz, auch auf die Kapitalkonten.

| Thema | Skill |
|---|---|
| Notar- und andere Gründungsrechnungen prüfen und buchen | buchfink-belege |
| Kontoauszug importieren, Einzahlung zuordnen | buchfink-bank |
| Voranmeldungen, Dauerfristverlängerung | buchfink-monatsabschluss |
| Erster Jahresabschluss, UG-Rücklage | buchfink-jahresabschluss |
| E-Bilanz der Eröffnungsbilanz, erste Steuererklärungen | buchfink-steuererklaerungen |
| Darlehen und Zahlungen zwischen Gesellschafter und GmbH | buchfink-gesellschafter |

Werkzeuge: `status`, `aufgaben`, `fristen`, `eingang`, `beleg_ablegen`,
`buchung_erfassen` (Rückfrage), `bankumsatz_buchen` (Rückfrage), `auswertung`,
`konten`, `buchungen_suchen`.

## 1. Phasen

| Phase | Zeitraum | Rechtslage | In Buchfink |
|---|---|---|---|
| Vorgründungsgesellschaft | Entschluss bis Beurkundung | nicht identisch mit Vorgesellschaft und GmbH; in der Regel nicht körperschaftsteuerpflichtig ([H 1.1 KStH][ksth11]) | kein Mandant; Kosten nur nach Klärung übernehmen ([gruendungskosten.md](references/gruendungskosten.md)) |
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
   ([§ 272 Abs. 1 HGB][hgb272]). Fertig, wenn die Buchung vorliegt.
5. **Einzahlung buchen (Rückfrage).** Je Gesellschafter am Bankumsatz:
   `bankumsatz_buchen` mit einer Zeile `konto: 1298`.
   Ohne importierten Umsatz: `buchung_erfassen` 1800 an 1298 mit dem
   Kontoauszug als Beleg. Prüfe die Mindesteinzahlung nach
   [kapitalaufbringung.md](references/kapitalaufbringung.md). Fertig, wenn der
   Saldo von 1298 dem noch offenen eingeforderten Betrag entspricht.
6. **Gründungskosten buchen.** Rechnungen nach buchfink-belege: Notar mit der
   Buchungsgruppe `beratung` (6825), Gerichtskosten mit `beitraege` (6420,
   nicht steuerbar). Prüfe vorher, ob die GmbH die Kosten nach dem Vertrag
   tragen darf ([gruendungskosten.md](references/gruendungskosten.md)).
   Fertig, wenn jede Gründungsrechnung gebucht oder als Kosten eines
   Gesellschafters geklärt ist.
7. **Behördenwege begleiten.** `fristen` führt ab der Beurkundung die
   Pflichten [fragebogen-steuerliche-erfassung](../buchfink-grundlagen/references/pflichten.md#fragebogen-steuerliche-erfassung)
   und [gewerbeanmeldung](../buchfink-grundlagen/references/pflichten.md#gewerbeanmeldung), ab der Eintragung
   [transparenzregister](../buchfink-grundlagen/references/pflichten.md#transparenzregister); Frist, Weg und Nachweis
   stehen dort. Die übrigen Wege (Berufsgenossenschaft, Gemeinde, IHK) und die
   Angaben zum Fragebogen: [gruendungsweg.md](references/gruendungsweg.md),
   [steuerliche-erfassung.md](references/steuerliche-erfassung.md). Fertig,
   wenn jede Pflicht in `fristen` erledigt ist oder einen Termin hat und die
   übrigen Wege ein Datum haben.
8. **Eröffnungsbilanz prüfen.** Salden der Gründungskonten zum
   Beurkundungstag mit `auswertung` (`art: kontoblatt`, `konto`,
   `bis: <Beurkundungstag>`); stehen nur Gründungsbuchungen im Jahr, genügt
   `auswertung` (`art: bilanz`). Fertig, wenn Aktiva und Passiva
   übereinstimmen und der Mensch sie gesehen hat.
   [eroeffnungsbilanz.md](references/eroeffnungsbilanz.md)
9. **Eintragung nachhalten (nur App).** Der Mensch trägt Datum, Registergericht
   und Nummer in den Unternehmensdaten ein. Prüfe danach die Unterbilanz
   ([kapitalaufbringung.md](references/kapitalaufbringung.md), Abschnitt 6) und
   mit `fristen`, dass die Mitteilung an das Transparenzregister ansteht.
   Fertig, wenn die Eintragung in der App erfasst und die Unterbilanz geprüft
   ist.
10. **Erstes Geschäftsjahr planen.** Lies
    [rumpfjahr.md](references/rumpfjahr.md) und nenne dem Menschen Ende des
    Rumpfjahres, Fristen des ersten Abschlusses und der ersten Erklärungen.
    Fertig, wenn `fristen` diese Pflichten mit Fälligkeit zeigt.

## 3. Entscheidungsregeln

- Die Einlage zählt nur, wenn sie endgültig zur freien Verfügung der
  Geschäftsführer steht ([§ 7 Abs. 3 GmbHG][gmbhg7], [§ 8 Abs. 2 Satz 1 GmbHG][gmbhg8]). Sie
  geht vom Gesellschafter auf ein Konto der Gesellschaft.
- Buche keine Zahlung an einen Gesellschafter, die wirtschaftlich die Einlage
  zurückführt, und keinen Kauf von einem Gesellschafter aus der Einlage, ohne
  Rückfrage. Beides kann die Einlagepflicht offenlassen
  ([§ 19 Abs. 4, 5 GmbHG][gmbhg19]).
- Eine Sacheinlage buchst du nur, wenn Gegenstand und Nennbetrag im Vertrag
  stehen ([§ 5 Abs. 4 GmbHG][gmbhg5]); den Wert legt der Mensch mit Beleg vor.
  Bei der UG ist sie ausgeschlossen ([§ 5a Abs. 2 Satz 2 GmbHG][gmbhg5a]).
- Kosten vor der Beurkundung buchst du nicht ohne Klärung in die GmbH.
- Unternehmensstammdaten, Registerdaten und Bankkonten ändert nur der Mensch
  in der App (buchfink-grundlagen).

## 4. Wertungsfragen

Leite jede mit „Wertungsfrage:“ ein, nenne Grundlage und Optionen.

- Wer trägt Kosten aus der Zeit vor der Beurkundung?
- Trägt die GmbH Gründungskosten über die Klausel im Vertrag hinaus, oder ohne
  Klausel? Folge: möglicherweise verdeckte Gewinnausschüttung
  (buchfink-gesellschafter).
- Ist eine Sacheinlage richtig bewertet ([§ 9 Abs. 1 GmbHG][gmbhg9])?
- Ist bei einem rein freiberuflichen Unternehmensgegenstand ein Gewerbe
  anzumelden?
- Steht der GmbH die Vorsteuer aus Gründungsrechnungen zu?
- Wann genau beginnt das Handelsgewerbe für die Eröffnungsbilanz?
- Welcher Voranmeldungszeitraum gilt im Jahr nach einer Gründung 2026?

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
- Reicht das Vermögen am Eintragungstag nicht an das Stammkapital heran,
  haften die Gesellschafter für die Differenz (Unterbilanzhaftung,
  [kapitalaufbringung.md](references/kapitalaufbringung.md)).
- Gründungsaufwand darf nicht aktiviert werden
  ([§ 248 Abs. 1 Nr. 1 HGB][hgb248]); er ist Aufwand des ersten Jahres.
- Das Musterprotokoll begrenzt die Gründungskosten der Gesellschaft auf 300 €,
  höchstens auf das Stammkapital ([Anlage 1 GmbHG][gmbhganl1]).
- Ohne Steuernummer oder USt-IdNr. fehlt eine Pflichtangabe jeder Rechnung
  ([§ 14 Abs. 4 Satz 1 Nr. 2 UStG][ustg14]). Treib den Fragebogen deshalb früh
  voran.
- Wer seine Tätigkeit ab 2027 aufnimmt, gibt die Voranmeldung im Gründungsjahr
  und im Folgejahr monatlich ab ([§ 18 Abs. 2 Satz 4, 6 UStG][ustg18]); für
  2021 bis 2026 gilt eine Sonderregel
  ([steuerliche-erfassung.md](references/steuerliche-erfassung.md)).
- Die UG muss jedes Jahr ein Viertel des Jahresüberschusses in die gesetzliche
  Rücklage stellen ([§ 5a Abs. 3 GmbHG][gmbhg5a]; buchfink-jahresabschluss).
- Auch die Eröffnungsbilanz ist als E-Bilanz zu übermitteln
  ([§ 5b Abs. 1 Satz 4 EStG][estg5b]; buchfink-steuererklaerungen).

## Referenzen

- [references/gruendungsweg.md](references/gruendungsweg.md): Lies vor dem
  Notartermin und für die Liste der Behördenwege mit Fristen.
- [references/kapitalaufbringung.md](references/kapitalaufbringung.md): Lies
  vor jeder Buchung von Zeichnung, Einzahlung oder Sacheinlage und vor der
  Eintragung (Unterbilanz).
- [references/gruendungskosten.md](references/gruendungskosten.md): Lies bei
  jeder Rechnung aus der Gründungsphase und bei Kosten vor der Beurkundung.
- [references/eroeffnungsbilanz.md](references/eroeffnungsbilanz.md): Lies zur
  Prüfung der Eröffnungsbilanz und wenn eine bestehende GmbH zu Buchfink wechselt.
- [references/steuerliche-erfassung.md](references/steuerliche-erfassung.md):
  Lies vor dem Fragebogen zur steuerlichen Erfassung und bei Fragen zu
  Voranmeldungszeitraum, Ist-Versteuerung, Dauerfristverlängerung und USt-IdNr.
- [references/rumpfjahr.md](references/rumpfjahr.md): Lies, wenn das erste
  Geschäftsjahr kürzer als zwölf Monate ist, und für die Fristen des ersten
  Abschlusses.

[estg5b]: https://www.gesetze-im-internet.de/estg/__5b.html
[gmbhg2]: https://www.gesetze-im-internet.de/gmbhg/__2.html
[gmbhg5]: https://www.gesetze-im-internet.de/gmbhg/__5.html
[gmbhg5a]: https://www.gesetze-im-internet.de/gmbhg/__5a.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg8]: https://www.gesetze-im-internet.de/gmbhg/__8.html
[gmbhg9]: https://www.gesetze-im-internet.de/gmbhg/__9.html
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg13]: https://www.gesetze-im-internet.de/gmbhg/__13.html
[gmbhg19]: https://www.gesetze-im-internet.de/gmbhg/__19.html
[gmbhganl1]: https://www.gesetze-im-internet.de/normengrafiken/bgbl1_2021/j3364_0010.pdf
[hgb248]: https://www.gesetze-im-internet.de/hgb/__248.html
[hgb272]: https://www.gesetze-im-internet.de/hgb/__272.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg18]: https://www.gesetze-im-internet.de/ustg_1980/__18.html
