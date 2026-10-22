# Ablauf: Eingangsbelege

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Ein Eingangsbeleg vom Ablegen bis zur Buchung. Zahlung:
[bank/ablauf.md](../bank/ablauf.md); Steuerbescheid:
[steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md).

## Inhalt

1. Ablauf
2. Sonderfälle
3. Stolperfallen

## 1. Ablauf

1. **Sammeln.** Nimm die Dateien aus `eingang`, die der Mensch nennt, und die
   abgelegten, ungebuchten Belege (`belege_suchen`, `status: abgelegt`). Liegt
   ein Beleg nur im Chat, bitte um die Originaldatei im Eingang. Fertig, wenn
   die Liste steht.
2. **Ablegen.** `beleg_ablegen` je Beleg; Dateien eines Vorgangs (Rechnung mit
   Bewirtungsbeleg oder Lieferschein) zusammen, die Rechnung als `original`.
   Schreiben ohne Buchung (Finanzamt, BZSt, Registergericht): `art: brief`;
   Vertrag oder Beschluss für `buchung_erfassen`: `art: sonstiges`. Fertig,
   wenn jeder Beleg eine Belegnummer hat.
3. **Lesen.** `beleg_lesen`. Ein Foto oder PDF ohne Datensatz liest du aus der
   Kopie von `beleg_datei_exportieren`; ohne Dateizugriff nennt der Mensch die
   Angaben. Fertig, wenn Aussteller, Belegdatum, Leistungszeitpunkt,
   Rechnungsnummer, Netto je Steuersatz, Steuer, Brutto und Währung
   feststehen.
4. **Prüfen**, in dieser Reihenfolge:
   - Beleg dieses Unternehmens? Fremde und private Belege legst du zum
     Verwerfen vor.
   - Dublette? Bei `dublettenverdacht` beide Belege vorlegen. Ohne Datensatz
     suchst du selbst: `belege_suchen` mit dem Aussteller als `text`, dann
     Rechnungsnummer und Brutto vergleichen.
   - Pflichtangaben nach Art der Rechnung
     ([pflichtangaben.md](pflichtangaben.md)). Fehlt eine oder ist der
     Steuerausweis falsch, fordert der Mensch eine Berichtigung an
     ([Abschnitt 5](pflichtangaben.md#5-fehlende-oder-falsche-angaben)).
   - Papier oder PDF eines inländischen Lieferanten statt E-Rechnung, Befunde
     von `beleg_lesen`, abweichender Bildteil:
     [e-rechnung-empfang.md](e-rechnung-empfang.md#5-folgen-für-die-vorsteuer),
     Abschnitte 5 bis 7.
   - Steuerfall: Weist ein inländischer Lieferant 19 % oder 7 % aus, ist es
     Inland, Satz und Netto je Satz aus dem Beleg. Jeden anderen Beleg
     bestimmst du nach
     [steuerfaelle.md](steuerfaelle.md#1-steuerfälle-in-buchfink).
   - Vorsteuer: `eingang_am` ist der echte Eingangstag
     ([vorsteuer.md](vorsteuer.md#2-zeitpunkt)). Eigenbeleg oder Rechnung ohne
     die GmbH als Empfängerin: keine Vorsteuer
     ([vorsteuer.md](vorsteuer.md#1-voraussetzungen)). Teils steuerfreie oder
     unternehmensfremde Verwendung: aufteilen
     ([vorsteuer.md](vorsteuer.md#4-aufteilung)).
   - Buchungsgruppe: Gebühren von Gerichten und Behörden `gebuehren` (6430),
     Kammer- und Verbandsbeiträge `beitraege` (6420); passt die Gruppe nicht
     zum Aussteller, warnt die Vorschau (`gruppe_passt_nicht_zum_aussteller`).
   - Fälligkeit: Datum oder Zahlungsziel auf dem Beleg gehört als
     `faellig_am` in die Buchung, sonst gilt das Zahlungsziel des Kontakts
     ([arbeitsweise.md, Abschnitt 10](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume)).
   - Sonderfall (Abschnitt 2).
   Fertig, wenn je Position Buchungsgruppe, Steuerfall und Satz feststehen,
   bei offener Zahlung die Fälligkeit, oder die offene Frage notiert ist.
5. **Kontakt.** `kontakte_suchen` (zuerst `ust_id`, dann `iban`, dann `text`).
   Fehlt der Lieferant, `kontakt_anlegen` mit USt-IdNr. oder Steuernummer,
   Anschrift und IBAN vom Beleg und `standardgruppe_vorschlag`; daran erkennt
   Buchfink später Routine. Einem Lieferanten ohne Gruppe schlägst du sie mit
   `standardgruppe_vorschlagen` vor. Fertig, wenn du eine `kontakt_id` hast.
6. **Routine buchen.** Bei `einstufung` `direkt`: `beleg_buchen`. Meldet es
   `freigabe_noetig`, kommt der Beleg in die Rückfragen. Fertig, wenn jeder
   Routinebeleg eine Buchungsnummer hat.
7. **Rückfragen.** Lege die übrigen Belege gesammelt vor
   ([SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen)) und buche
   nach dem Ja mit `beleg_buchen_mit_rueckfrage` und den Sonderangaben aus
   Abschnitt 2. Fremde Belege und Dubletten verwirfst du nach dem Ja mit
   `beleg_verwerfen`; ein Brief ist mit der Ablage erledigt. Fertig, wenn jeder
   Beleg gebucht, als Brief abgelegt, verworfen oder mit Grund zurückgestellt
   ist.
8. **Abschließen.** `belege_suchen` (`status: abgelegt`) und `eingang`: Jeder
   verbliebene Beleg und jede liegende Datei hat einen genannten Grund
   (fehlende Angabe, Berichtigung angefordert, Grenze von Buchfink, fremder
   Beleg). Sag dem Menschen kurz, was gebucht ist und was offen bleibt.

## 2. Sonderfälle

Jeder Fall ist Rückfrage; die Sonderangabe gehört in
`beleg_buchen_mit_rueckfrage`. Lies die verlinkte Datei, sobald der Fall
vorliegt.

| Fall | Sonderangabe | Vorher klären | Regeln |
|---|---|---|---|
| Bewirtung aus geschäftlichem Anlass | `bewirtung` | Teilnehmer und Anlass; Rechnung der Gaststätte liegt bei | [bewirtung-geschenke.md](bewirtung-geschenke.md#1-bewirtung) |
| Geschenk an Geschäftspartner | `geschenk` je Position | Empfänger; ob er dieses Jahr schon etwas bekam | [bewirtung-geschenke.md](bewirtung-geschenke.md#2-geschenke) |
| Anschaffung eines Wirtschaftsguts | Gruppe `gwg` oder Anlagekonto als `konto`; danach `abschlussbuchung` mit `art: anlage_aktivieren` | Aufwand oder Anschaffung, Nutzbarkeit, Methode des Jahres | [anlagen-gwg.md](anlagen-gwg.md) |
| Beleg in Fremdwährung | `waehrung`, `fremdbetrag` | Kurs für die Umsatzsteuer | [fremdwaehrung-auslagen.md](fremdwaehrung-auslagen.md#1-beleg-in-fremdwährung) |
| Gesellschafter hat bezahlt | `bezahlung: auslage`, `gesellschafter_kontakt_id` | Wer gezahlt hat; Rechnung an die GmbH | [fremdwaehrung-auslagen.md](fremdwaehrung-auslagen.md#2-auslage-eines-gesellschafters) |
| Mit Firmenkreditkarte bezahlt | `bezahlung: kreditkarte` | Rechnung liegt vor | [bank/kreditkarte-geldtransit.md](../bank/kreditkarte-geldtransit.md#2-kartenbeleg) |
| Bar oder schon vom Bankkonto bezahlt | `bezahlung: bezahlt`, `zahlungskonto` | Konto der Zahlung | – |
| Teils unternehmensfremd oder steuerfrei verwendet | `vorsteueranteil_promille`, `vorsteueranteil_grund` je Position | Maßstab (Wertungsfrage) | [vorsteuer.md](vorsteuer.md#4-aufteilung) |
| Rechnung zu einer Rückstellung | `rueckstellung_id` | Welche Rückstellung | [jahresabschluss/rueckstellungen.md](../jahresabschluss/rueckstellungen.md) |
| Anzahlungsrechnung des Lieferanten | `anzahlung_fuer` | Zahlung ist geleistet | [vorsteuer.md](vorsteuer.md#2-zeitpunkt) |
| Schlussrechnung nach Anzahlungen | `verrechnete_anzahlungen`; ihre Kennungen liefert kein Werkzeug | Welche Anzahlungen abgesetzt sind | [anlagen-gwg.md](anlagen-gwg.md#5-weg-in-buchfink) |
| Beleg über der Nachweisgrenze | `leistungsnachweis`, `leistungsnachweis_am` | Gegen welche Bestellung der Mensch geprüft hat | [pflichtangaben.md](pflichtangaben.md#1-was-eine-rechnung-ist) |
| Blockierender Befund der Rechnungsprüfung | `uebersteuerung_grund` | Grund, den der Mensch trägt | [e-rechnung-empfang.md](e-rechnung-empfang.md#6-fehlerarten-und-befunde) |

## 3. Stolperfallen

- Kartenbeleg, Bestellbestätigung und Lieferschein sind keine Rechnung; ohne
  Rechnung keine Vorsteuer.
- Bei § 13b und innergemeinschaftlichem Erwerb zahlt die GmbH netto
  ([steuerfaelle.md](steuerfaelle.md#2-leistungen-aus-dem-ausland--13b-ustg)).
- Gutschrift oder Preisnachlass eines Lieferanten auf eine gebuchte Rechnung
  ist ein eigener Beleg (Rückfrage); die Vorsteuer mindert sich im Zeitraum der
  Änderung ([vorsteuer.md](vorsteuer.md#7-minderung-des-entgelts)). Ein Storno
  der Ursprungsbuchung wirkt im alten Zeitraum
  ([grundlagen/arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren));
  nimm es nur für eigene Buchungsfehler.
