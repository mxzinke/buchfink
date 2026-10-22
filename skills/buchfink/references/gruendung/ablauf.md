# Ablauf: Gründung

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Buchfink führt Bücher ab der Beurkundung.

## Inhalt

1. Phasen
2. Ablauf
3. Entscheidungsregeln
4. Wertungsfragen
5. Nur der Mensch

## 1. Phasen

| Phase | Zeitraum | Rechtslage | In Buchfink |
|---|---|---|---|
| Vorgründungsgesellschaft | Entschluss bis Beurkundung | eigene Gesellschaft; ihre Verträge gehen nicht von selbst auf die GmbH über ([gruendungskosten.md, Abschnitt 4](gruendungskosten.md#4-kosten-vor-der-beurkundung)) | kein Mandant |
| Vorgesellschaft, GmbH i. G. | Beurkundung ([§ 2 Abs. 1 GmbHG][gmbhg2]) bis Eintragung | Handelnde haften persönlich ([§ 11 Abs. 2 GmbHG][gmbhg11]); körperschaftsteuerpflichtig ([H 1.1 KStH][ksth11]) | erstes Geschäftsjahr beginnt |
| GmbH | ab Eintragung ([§ 11 Abs. 1 GmbHG][gmbhg11]) | es haftet nur das Gesellschaftsvermögen ([§ 13 Abs. 1, 2 GmbHG][gmbhg13]) | Registerdaten übernommen |

## 2. Ablauf

1. **Lage klären.** Frag nach Rechtsform, Stammkapital, Gesellschaftern mit
   Nennbetrag und Einlageart (Geld oder Sache), Musterprotokoll, Klausel zum
   Gründungsaufwand, Beurkundungstag, Eintragung, Geschäftsjahr und
   Geschäftskonto. Fertig, wenn jede Angabe feststeht oder als offen benannt
   ist.
2. **Mandant anlegen lassen (nur App).** Der Mensch legt den Mandanten im
   Einrichtungsassistenten an und trägt Hebesatz und steuerliche Beratung in
   den Stammdaten ein; daraus rechnet Buchfink die Fristen. Fertig, wenn
   `status` den Mandanten zeigt und das erste Geschäftsjahr am
   Beurkundungstag beginnt
   ([rumpfjahr.md, Abschnitt 2](rumpfjahr.md#2-in-buchfink)).
3. **Unterlagen ablegen und vorschlagen.** Lege mit `beleg_ablegen` ab:
   Gesellschaftsvertrag oder Musterprotokoll, Gesellschafterliste und
   Beschlüsse als `art: sonstiges`, Schreiben ohne Buchung als `art: brief`,
   Notar- und Gerichtsrechnungen als Rechnung. Schlag dann mit
   `stammdaten_vorschlagen` und dem Gesellschaftsvertrag als `beleg_id`
   `gruendung`, `unternehmen` und das Geschäftskonto als `bankkonten` vor.
   Fertig, wenn jede Unterlage eine `beleg_id` hat und `status` den Vorschlag
   als `uebernommen` zeigt.
4. **Zeichnung buchen (Rückfrage).** `buchung_erfassen` zum Beurkundungstag
   mit dem Gesellschaftsvertrag als Beleg, Konten nach
   [kapitalaufbringung.md, Abschnitt 3](kapitalaufbringung.md#3-buchungen).
   Fertig, wenn die Buchung vorliegt.
5. **Einzahlung buchen (Rückfrage).** Je Gesellschafter `bankumsatz_buchen`
   mit einer Zeile `konto: 1298` ([bank/ablauf.md](../bank/ablauf.md)); ohne
   importierten Umsatz `buchung_erfassen` 1800 an 1298 mit dem Kontoauszug
   als Beleg. Prüfe die Mindesteinzahlung nach
   [kapitalaufbringung.md, Abschnitt 2](kapitalaufbringung.md#2-was-vor-der-anmeldung-eingezahlt-sein-muss).
   Fertig, wenn der Saldo von 1298 dem eingeforderten, noch nicht gezahlten
   Betrag entspricht.
6. **Gründungskosten buchen.** Kläre zuerst, ob die GmbH die Kosten nach dem
   Vertrag trägt
   ([gruendungskosten.md, Abschnitt 2](gruendungskosten.md#2-wer-die-gründungskosten-trägt)),
   dann buche die Rechnungen nach [belege/ablauf.md](../belege/ablauf.md).
   Fertig, wenn jede Gründungsrechnung gebucht oder als Kosten eines
   Gesellschafters geklärt ist.
7. **Behördenwege nachhalten.** Begleite die Pflichten aus `fristen`; die
   übrigen Wege stehen in
   [gruendungsweg.md, Abschnitt 4](gruendungsweg.md#4-behördenwege-mit-fristen),
   die Angaben zum Fragebogen in
   [steuerliche-erfassung.md](steuerliche-erfassung.md). Treib den Fragebogen
   früh voran: Ohne Steuernummer fehlt jeder Ausgangsrechnung eine
   Pflichtangabe
   ([pflichten.md#fragebogen-steuerliche-erfassung](../grundlagen/pflichten.md#fragebogen-steuerliche-erfassung)).
   Fertig, wenn jede Pflicht erledigt ist oder einen Termin hat.
8. **Eröffnungsbilanz prüfen** nach
   [eroeffnungsbilanz.md, Abschnitt 4](eroeffnungsbilanz.md#4-prüfen).
9. **Eintragung nachhalten.** Schlag Eintragungstag, Registergericht und
   Registernummer aus der Eintragungsnachricht mit `stammdaten_vorschlagen`
   vor; der Mensch übernimmt sie in der App. Prüfe danach die Unterbilanz
   ([kapitalaufbringung.md, Abschnitt 6](kapitalaufbringung.md#6-unterbilanz-am-eintragungstag)).
   Fertig, wenn die Eintragung übernommen und die Unterbilanz geprüft ist.
10. **Erstes Geschäftsjahr planen.** Nenne dem Menschen nach
    [rumpfjahr.md](rumpfjahr.md) das Ende des Rumpfgeschäftsjahres und die
    Fristen des ersten Abschlusses und der ersten Erklärungen. Fertig, wenn
    `fristen` diese Pflichten mit Fälligkeit zeigt.

## 3. Entscheidungsregeln

- Die Einlage zählt nur, wenn sie vom Gesellschafter auf ein Konto der
  Gesellschaft geht und endgültig zur freien Verfügung der Geschäftsführer
  steht ([§ 7 Abs. 3 GmbHG][gmbhg7], [§ 8 Abs. 2 Satz 1 GmbHG][gmbhg8]).
- Zahlungen an einen Gesellschafter und Käufe von ihm kurz nach der
  Einzahlung legst du vor, bevor du buchst
  ([kapitalaufbringung.md, Abschnitt 5](kapitalaufbringung.md#5-verdeckte-sacheinlage-und-hin--und-herzahlen)).
- Eine Sacheinlage buchst du nur, wenn Gegenstand und Nennbetrag im Vertrag
  stehen und der Mensch den Wert mit Beleg vorlegt
  ([kapitalaufbringung.md, Abschnitt 4](kapitalaufbringung.md#4-sacheinlage)).
- Schließt jemand vor der Eintragung größere Verträge im Namen der GmbH,
  erinnere an die Handelndenhaftung ([§ 11 Abs. 2 GmbHG][gmbhg11]).

## 4. Wertungsfragen

Format: [SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen).

- Kosten vor der Beurkundung
  ([gruendungskosten.md, Abschnitt 4](gruendungskosten.md#4-kosten-vor-der-beurkundung)).
- Gründungskosten über der Klausel oder ohne Klausel
  ([gruendungskosten.md, Abschnitt 2](gruendungskosten.md#2-wer-die-gründungskosten-trägt)).
- Bewertung einer Sacheinlage
  ([kapitalaufbringung.md, Abschnitt 4](kapitalaufbringung.md#4-sacheinlage)).
- Gewerbeanzeige bei rein freiberuflichem Gegenstand
  ([pflichten.md#gewerbeanmeldung](../grundlagen/pflichten.md#gewerbeanmeldung)).
- Vorsteuer aus Gründungsrechnungen
  ([gruendungskosten.md, Abschnitt 3](gruendungskosten.md#3-buchen)).
- Beginn des Handelsgewerbes für die Eröffnungsbilanz
  ([eroeffnungsbilanz.md, Abschnitt 2](eroeffnungsbilanz.md#2-stichtag-in-buchfink)).
- Voranmeldungszeitraum im Jahr nach einer Gründung 2026
  ([steuerliche-erfassung.md, Abschnitt 3](steuerliche-erfassung.md#3-voranmeldungszeitraum)).
- Unterdeckung bei der Eintragung aus Gründungsaufwand oder Anlaufverlust
  ([kapitalaufbringung.md, Abschnitt 6](kapitalaufbringung.md#6-unterbilanz-am-eintragungstag)).

## 5. Nur der Mensch

- Beurkundung, Unterschriften, Anmeldung zum Handelsregister über den Notar.
- Geschäftskonto eröffnen, Einlage einzahlen.
- Fragebogen zur steuerlichen Erfassung, Gewerbeanzeige, Mitteilung an das
  Transparenzregister, Meldung an die Berufsgenossenschaft,
  ELSTER-Registrierung.
- Mandant, Ablage und Eingang anlegen; deine Vorschläge zu Unternehmensdaten,
  Gründung, Registerdaten und Bankkonten übernehmen.

[gmbhg2]: https://www.gesetze-im-internet.de/gmbhg/__2.html
[gmbhg7]: https://www.gesetze-im-internet.de/gmbhg/__7.html
[gmbhg8]: https://www.gesetze-im-internet.de/gmbhg/__8.html
[gmbhg11]: https://www.gesetze-im-internet.de/gmbhg/__11.html
[gmbhg13]: https://www.gesetze-im-internet.de/gmbhg/__13.html
[ksth11]: https://www.steuerschroeder.de/steuergesetze/ksth/1.1
