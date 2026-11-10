# Ablauf: Eingangsbelege

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Ein Eingangsbeleg vom Ablegen bis zur Buchung. Zahlung:
[bank/ablauf.md](../bank/ablauf.md); Steuerbescheid:
[steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md).

## Inhalt

1. Ablauf
2. Sonderfälle

## 1. Ablauf

1. **Sammeln.** Nimm die Dateien aus `eingang`, die der Mensch nennt, und die
   abgelegten, ungebuchten Belege (`belege_suchen`, `status: abgelegt`). Liegt
   ein Beleg nur im Chat, bitte um die Originaldatei im Eingang. Fertig, wenn
   die Liste steht.
2. **Ablegen.** `beleg_ablegen` je Beleg; Dateien eines Vorgangs (Rechnung mit
   Bewirtungsbeleg oder Lieferschein) zusammen. Schreiben ohne Buchung
   (Finanzamt, BZSt, Registergericht): `art: brief`; Vertrag oder Beschluss:
   `art: sonstiges` (Darlehensvertrag siehe
   [gesellschafter/ablauf.md](../gesellschafter/ablauf.md#4-darlehen)). Fertig,
   wenn jeder Beleg eine Belegnummer hat.
3. **Lesen.** `beleg_lesen`; ein Foto oder PDF liest du aus der Kopie von
   `beleg_datei_exportieren`, ohne Dateizugriff nennt der Mensch die Angaben.
   Fertig, wenn Aussteller, Belegdatum, Leistungszeitpunkt, Rechnungsnummer,
   Netto je Steuersatz, Steuer, Brutto und Währung feststehen. Eine
   Kleinbetragsrechnung oder ein Fahrausweis nennt nur Brutto je Satz
   ([§ 35 UStDV][ustdv35]): `brutto` statt `netto`.
4. **Prüfen**, in dieser Reihenfolge:
   - Beleg dieses Unternehmens? Fremde und private Belege legst du zum
     Verwerfen vor.
   - Rechnung? Kartenbeleg, Bestellbestätigung und Lieferschein sind keine;
     ohne Rechnung keine Vorsteuer.
   - Dublette? Bei `dublettenverdacht` beide Belege vorlegen. Ohne Datensatz
     suchst du selbst: `belege_suchen` mit dem Aussteller als `text`, dann
     Rechnungsnummer und Brutto vergleichen.
   - Pflichtangaben nach Art der Rechnung
     ([pflichtangaben.md](pflichtangaben.md)); bei fehlender oder falscher
     Angabe fordert der Mensch eine Berichtigung an
     ([Abschnitt 5](pflichtangaben.md#5-fehlende-oder-falsche-angaben)).
   - Papier oder PDF eines inländischen Lieferanten statt E-Rechnung, Befunde
     von `beleg_lesen`, abweichender Bildteil:
     [e-rechnung-empfang.md](e-rechnung-empfang.md#5-folgen-für-die-vorsteuer),
     Abschnitte 5 bis 7.
   - Steuerfall: Weist ein inländischer Lieferant 19 % oder 7 % aus, ist es
     Inland, Satz und Netto je Satz aus dem Beleg. Ein Posten ohne Steuer
     darauf (Trinkgeld, Kurtaxe für die Gemeinde,
     [§ 10 Abs. 1 Satz 4 UStG][ustg10]) ist `nicht_steuerbar` mit Grund und
     ohne Vorsteuer ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]). Jeden anderen
     Beleg bestimmst du nach
     [steuerfaelle.md](steuerfaelle.md#1-steuerfälle-in-buchfink); bei § 13b
     und innergemeinschaftlichem Erwerb zahlt die GmbH netto.
   - Vorsteuer: `eingang_am` ist der echte Eingangstag
     ([vorsteuer.md](vorsteuer.md#2-zeitpunkt)). Eigenbeleg oder Rechnung ohne
     die GmbH als Empfängerin: keine Vorsteuer
     ([vorsteuer.md](vorsteuer.md#1-voraussetzungen)). Teils steuerfreie oder
     unternehmensfremde Verwendung: aufteilen
     ([vorsteuer.md](vorsteuer.md#4-aufteilung)).
   - Buchungsgruppe: Gebühren von Gerichten und Behörden `gebuehren`, Kammer-
     und Verbandsbeiträge `beitraege`.
   - Fälligkeit: Datum oder Zahlungsziel auf dem Beleg als `faellig_am`
     ([arbeitsweise.md, Abschnitt 10](../grundlagen/arbeitsweise.md#10-geschäftsjahr-und-zeiträume)).
   - Sonderfall (Abschnitt 2).
   Fertig, wenn je Position Buchungsgruppe, Steuerfall und Satz feststehen,
   bei offener Zahlung die Fälligkeit, oder die offene Frage notiert ist.
5. **Kontakt.** `kontakte_suchen` (zuerst `ust_id`, dann `iban`, dann `text`).
   Fehlt der Lieferant, `kontakt_anlegen` mit USt-IdNr. oder Steuernummer,
   Anschrift und IBAN vom Beleg; einem Lieferanten ohne Gruppe schlägst du sie
   vor. Fertig, wenn du eine `kontakt_id` hast.
6. **Routine buchen.** Bei `einstufung` `direkt`: `beleg_buchen`. Fertig, wenn
   jeder Routinebeleg eine Buchungsnummer hat.
7. **Rückfragen.** Lege die übrigen Belege gesammelt vor
   ([SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen)) und buche
   nach dem Ja mit `beleg_buchen_mit_rueckfrage` und den Sonderangaben aus
   Abschnitt 2. Fremde Belege und Dubletten verwirfst du nach dem Ja mit
   `beleg_verwerfen`; ein Brief ist mit der Ablage erledigt. Fertig, wenn jeder
   Beleg gebucht, als Brief abgelegt, einem Darlehen zugeordnet, verworfen
   oder mit Grund zurückgestellt ist.
8. **Abschließen.** `belege_suchen` (`status: abgelegt`) und `eingang`: Jeder
   verbliebene Beleg und jede liegende Datei hat einen genannten Grund
   (fehlende Angabe, Berichtigung angefordert, Grenze von Buchfink, fremder
   Beleg). Sag dem Menschen kurz, was gebucht ist und was offen bleibt.

## 2. Sonderfälle

Jeder Fall ist Rückfrage mit `beleg_buchen_mit_rueckfrage`. Lies die
verlinkte Datei, sobald der Fall vorliegt.

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
| Gutschrift des Lieferanten (Rechnungstyp 381 oder Preisnachlass) | `gutschrift_zu` (Rechnungsnummer) | Welche Rechnung sie mindert; Erstattung oder Abzug von der nächsten Zahlung. Kein Storno der Ursprungsbuchung ([arbeitsweise.md](../grundlagen/arbeitsweise.md#11-eine-buchung-korrigieren)); den nur für eigene Fehler | [vorsteuer.md](vorsteuer.md#7-minderung-des-entgelts) |
| Anzahlungsrechnung des Lieferanten | `anzahlung_fuer` | Zahlung ist geleistet | [vorsteuer.md](vorsteuer.md#2-zeitpunkt) |
| Schlussrechnung nach Anzahlungen | `verrechnete_anzahlungen`; ihre Kennungen liefert kein Werkzeug | Welche Anzahlungen abgesetzt sind | [anlagen-gwg.md](anlagen-gwg.md#5-weg-in-buchfink) |
| Beleg über der Nachweisgrenze | `leistungsnachweis`, `leistungsnachweis_am` | Gegen welche Bestellung der Mensch geprüft hat | [pflichtangaben.md](pflichtangaben.md#1-was-eine-rechnung-ist) |
| Blockierender Befund der Rechnungsprüfung | `uebersteuerung_grund` | Grund, den der Mensch trägt | [e-rechnung-empfang.md](e-rechnung-empfang.md#6-fehlerarten-und-befunde) |

[ustdv35]: https://www.gesetze-im-internet.de/ustdv_1980/__35.html
[ustg10]: https://www.gesetze-im-internet.de/ustg_1980/__10.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
