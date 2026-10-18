---
name: buchfink-belege
description: "Eingangsbelege in Buchfink ablegen, prüfen, kontieren und buchen. Lade den Skill, wenn eine Rechnung, Quittung oder ein anderer Eingangsbeleg zu buchen oder der Eingang abzuarbeiten ist; bei E-Rechnung und Befunden der Rechnungsprüfung; bei Pflichtangaben, Kleinbetragsrechnung und Vorsteuerabzug; bei Lieferanten aus dem Ausland (§ 13b UStG, innergemeinschaftlicher Erwerb, Drittland); bei Bewirtung, Geschenken und Reisekosten; bei Anschaffungen (GWG, Sammelposten, Anlagegut, Nutzungsdauer); bei Belegen in Fremdwährung; bei Auslagen eines Gesellschafters und Kreditkartenbelegen; bei Anzahlungs-, Schluss- und Gutschriftsbelegen von Lieferanten; wenn beleg_buchen freigabe_noetig meldet. Die Zahlung am Bankkonto regelt buchfink-bank, Ausgangsrechnungen buchfink-rechnungen."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server, einem für den Client zugelassenen Mandanten und einem eingerichteten Eingang."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Eingangsbelege

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Dieser Skill führt einen Eingangsbeleg vom Ablegen bis zur Buchung. Die
Zahlung des Belegs ist ein eigener Vorgang (buchfink-bank). Ausgangsrechnungen,
Lohnjournale und Steuerbescheide gehören in ihre eigenen Skills.

Leitwörter: **Routine**, **Rückfrage**, **nur App**, **Wertungsfrage** wie in
buchfink-grundlagen.

## 1. Ablauf

1. **Belege sammeln.** Arbeite den Eingang nur auf Anweisung des Menschen ab.
   `eingang` listet die offenen Dateien mit erkanntem Typ und Alter. Nimm die
   Dateien, die der Mensch nennt, oder auf seine Anweisung alle
   Rechnungen und Quittungen im Eingang; CAMT-Kontoauszüge übernimmt
   buchfink-bank. Prüfe mit `belege_suchen` (`status: abgelegt`), was schon
   abgelegt und noch nicht gebucht ist; dazu gehört, was der Mensch ins
   Buchfink-Fenster gezogen hat. Liegt ein Beleg nur im Chat, bitte ihn, die
   Originaldatei in den Eingang zu speichern (buchfink-grundlagen,
   Belegsicherung). Fertig, wenn du eine Liste der Dateien und offenen Belege
   hast.
2. **Ablegen.** Rufe je Beleg `beleg_ablegen` mit `dateien` (je `pfad` im
   Eingang und `rolle`) auf. Setze `eingang_am` auf den Tag, an dem der Beleg im
   Unternehmen ankam, und `eingang_ueber` (`email`, `scan`, `upload`). Bei
   Fotos und PDF ohne E-Rechnung gehören `belegdatum`, `aussteller` und
   `brutto` dazu, bei einer E-Rechnung gelten ihre Daten. Gehören mehrere
   Dateien zu einem Vorgang (Rechnung und Bewirtungsbeleg, Rechnung und
   Lieferschein), lege sie zusammen ab: genau eine `original`, der Rest
   `anhang`. Zu einer reinen XRechnung gehört die PDF-Ansicht mit
   `rolle: darstellung`; ohne sie lässt sie sich nicht buchen. Kommt die
   Ansicht später, lege dieselbe XML-Datei mit der Ansicht noch einmal ab:
   Buchfink ergänzt sie am vorhandenen Beleg. Meldet Buchfink
   `bereits_vorhanden`, lag die Originaldatei schon vor. Fertig, wenn jeder
   Beleg eine Belegnummer hat; Buchfink hat die Dateien dann nach
   `Eingang/Übernommen/<JJJJ-MM>` verschoben.
3. **Lesen.** Rufe `beleg_lesen` mit `beleg_id` auf. Lies Kopfdaten,
   `befunde`, `vorschlag`, `aus_beleg` und `einstufung`. Ein Foto oder PDF
   ohne Datensatz liest du selbst: `beleg_datei_exportieren` legt eine Kopie in
   den Exportordner; ohne Dateizugriff nennt dir der Mensch die Angaben.
   Fertig, wenn du Aussteller, Belegdatum, Leistungszeitpunkt,
   Rechnungsnummer, Netto je Steuersatz, Steuer, Brutto und Währung kennst.
4. **Prüfen.** Kläre in dieser Reihenfolge:
   - Ist es ein Beleg dieses Unternehmens? Fremde und private Belege legst du
     dem Menschen zum Verwerfen vor.
   - Dublette? Meldet `einstufung` `dublettenverdacht`, leg beide Belege vor
     und buche nicht. Bei einem Foto oder PDF ohne Datensatz suchst du selbst:
     `belege_suchen` mit dem Aussteller als `text`, dann Rechnungsnummer und
     Brutto vergleichen.
   - Pflichtangaben und Vorsteuer (Abschnitt 2).
   - Steuerfall und Satz (Abschnitt 3).
   - Sonderfall (Abschnitt 4).
   Fertig, wenn je Position Buchungsgruppe, Steuerfall und Satz feststehen oder
   die offene Frage notiert ist.
5. **Kontakt.** Suche den Lieferanten mit `kontakte_suchen` (zuerst `ust_id`,
   dann `iban`, dann `text`). Fehlt er, lege ihn mit `kontakt_anlegen` an:
   `typ: lieferant`, `name`, `land`, `ust_id`, `iban` und eine Gruppe als
   `standardgruppe_vorschlag`. Bei einem vorhandenen Lieferanten ohne Gruppe
   schlägst du sie mit `standardgruppe_vorschlagen` vor. Bestätigen kann sie
   nur der Mensch in der App; bis dahin läuft jeder Beleg über Rückfrage.
   Fertig, wenn du eine `kontakt_id` hast.
6. **Routine buchen.** Lautet `einstufung.stufe` `direkt`, rufe `beleg_buchen`
   auf: `beleg_id`, `kontakt_id`, `belegdatum`, `leistung_von`, `steuerfall`,
   `positionen` (je `buchungsgruppe`, `netto`, `satz`), `bezahlung` (`offen`
   oder `kreditkarte`). Bist du unsicher, wie Buchfink bucht, sende zuerst
   `nur_vorschau: true`. Meldet das Werkzeug `freigabe_noetig`, kommt der Beleg
   in die Rückfragen. Fertig, wenn jeder Routinebeleg eine Buchungsnummer hat.
7. **Rückfragen.** Lege alle übrigen Belege gesammelt vor (Format in
   buchfink-grundlagen). Nach dem Ja rufst du `beleg_buchen_mit_rueckfrage`
   mit den Sonderangaben aus Abschnitt 4 und einem `idempotenz_schluessel` auf.
   Belege, die keine Belege dieses Unternehmens sind oder doppelt vorliegen,
   verwirfst du nach dem Ja mit `beleg_verwerfen` und `grund`.
   Fertig, wenn jeder Beleg gebucht, verworfen oder mit Grund zurückgestellt
   ist.
8. **Abschließen.** Rufe `belege_suchen` (`status: abgelegt`) und `eingang`
   auf. Jeder verbliebene Beleg und jede liegende Datei hat einen genannten
   Grund (fehlende Angabe, Berichtigung angefordert, Grenze von Buchfink, kein
   Beleg dieses Unternehmens). Sag dem Menschen in wenigen Sätzen, was gebucht
   ist und was offen bleibt.

Buche bis zum Ende des Folgemonats ([GoBD Rz. 52][gobd]); lege zeitnah,
möglichst unmittelbar nach Eingang ab ([GoBD Rz. 67][gobd]).

## 2. Pflichtangaben und Vorsteuer

- Vorsteuer setzt eine Rechnung nach §§ 14, 14a UStG voraus, die das
  Unternehmen besitzt, und eine Leistung für das Unternehmen
  ([§ 15 Abs. 1 Satz 1 Nr. 1 UStG][ustg15]). Abziehbar ist nur die gesetzlich
  geschuldete Steuer. Ein Eigenbeleg trägt keine Vorsteuer.
- Welche Angaben nötig sind, hängt vom Beleg ab:

  | Beleg | Angaben | Fundstelle |
  |---|---|---|
  | Rechnung über 250 € brutto | die Pflichtangaben, darunter Name und Anschrift der GmbH, Steuernummer oder USt-IdNr. des Lieferanten, Rechnungsnummer, Leistungszeitpunkt, Entgelt je Satz, Steuerbetrag | [§ 14 Abs. 4 UStG][ustg14] |
  | Rechnung bis 250 € brutto | Lieferant mit Anschrift, Datum, Leistung, Brutto und Steuersatz | [§ 33 UStDV][ustdv33] |
  | Fahrausweis | Beförderer, Datum, Brutto, Steuersatz (entfällt bei 7 % nach § 12 Abs. 2 Nr. 10 UStG) | [§ 34 UStDV][ustdv34] |
  | Rechnung eines Kleinunternehmers | Hinweis auf § 19 UStG; keine Steuer, keine Vorsteuer | [§ 34a UStDV][ustdv34a] |

- Seit 2025 muss ein inländischer Lieferant einem inländischen Unternehmen
  grundsätzlich eine E-Rechnung ausstellen
  ([§ 14 Abs. 2 Satz 2 Nr. 1 UStG][ustg14]). Für Leistungen aus 2025 und 2026
  ist eine Papierrechnung oder, mit Zustimmung der GmbH, eine PDF-Rechnung noch
  zulässig, wenn sie bis 31.12.2026 übermittelt wird. Für Leistungen aus 2027
  gilt das nur bei Ausstellern mit höchstens 800.000 € Vorjahresumsatz
  ([§ 27 Abs. 38 UStG][ustg27]). Stand: Oktober 2026.
  Einzelheiten: [references/e-rechnung-empfang.md](references/e-rechnung-empfang.md).
- Bei einer E-Rechnung zählt der strukturierte Teil. Weicht der Bildteil ab,
  meldest du das dem Menschen ([BMF vom 15.10.2025, UStAE 14c.1 Abs. 4a][bmf2025]).
- Befunde von `beleg_lesen` gibt es in drei Klassen. Ein Inhaltsfehler (fehlende
  Pflichtangabe) kostet die Vorsteuer. Ein Formatfehler macht aus der Datei
  eine sonstige Rechnung. Ein Geschäftsregelfehler ohne Pflichtangabe ist
  umsatzsteuerlich unbeachtlich ([BMF vom 15.10.2025, Rn. 6a, 6b, 35a][bmf2025]).
- Fehlt eine Pflichtangabe, bitte den Menschen, beim Lieferanten eine
  Berichtigung anzufordern. Sie muss sich eindeutig auf die Rechnung beziehen
  ([§ 31 Abs. 5 UStDV][ustdv31]). Buchen trotz blockierendem Befund geht nur
  mit `uebersteuerung_grund`, den der Mensch entscheidet (Wertungsfrage).
- Buchfink ordnet die Vorsteuer dem Zeitraum zu, in dem Leistung, Belegdatum
  und Eingang alle vorliegen. Ein falsches `eingang_am` verschiebt die
  Vorsteuer in einen falschen Monat; berichtige es mit
  `beleg_buchen_mit_rueckfrage` (`eingang_am`).

Lies [references/pflichtangaben.md](references/pflichtangaben.md) und
[references/vorsteuer.md](references/vorsteuer.md), wenn ein Befund, eine
fehlende Angabe, ein falscher Steuerausweis oder eine Aufteilung der Vorsteuer
vorliegt.

## 3. Steuerfall bestimmen

Wähle den Steuerfall aus dem Beleg, nie aus der Gewohnheit.

| Merkmal auf dem Beleg | Steuerfall | Hinweis |
|---|---|---|
| Inländischer Lieferant weist 19 % oder 7 % aus | Inland | Satz und Netto je Satz aus dem Beleg; Hotel und Gaststätte weisen oft zwei Sätze aus |
| Unternehmer aus einem anderen EU-Staat, sonstige Leistung (Software, Werbung, Beratung), keine deutsche Steuer | § 13b Abs. 1 (`reverse_charge_eu`) | Die GmbH schuldet die Steuer und zieht sie ab ([§ 13b Abs. 1, Abs. 5 UStG][ustg13b]). USt-IdNr. am Kontakt Pflicht |
| Unternehmer aus dem Drittland, sonstige Leistung; Werklieferung eines ausländischen Unternehmers | § 13b Abs. 2 (`reverse_charge`) | [§ 13b Abs. 2 Nr. 1, Abs. 5 UStG][ustg13b] |
| Ware aus einem anderen EU-Staat, beide USt-IdNr., keine Steuer | innergemeinschaftlicher Erwerb | [§ 1a UStG][ustg1a]; USt-IdNr. am Kontakt Pflicht |
| Ware aus dem Drittland | Einfuhr | Einfuhrumsatzsteuer bildet Buchfink nicht vollständig ab: vorlegen |
| Versicherung, Kontoführung, Miete ohne Steuerausweis | steuerfrei | [§ 4 Nr. 8, 10, 12 UStG][ustg4] |
| Kammerbeitrag, Behördengebühr, Schadenersatz | nicht steuerbar | kein Leistungsaustausch |
| Ausländische Umsatzsteuer ausgewiesen | Wertungsfrage | Keine deutsche Vorsteuer. Hotel oder Mietwagen im Ausland: Steuer des anderen Staates ist richtig. Sonst Rechnung mit USt-IdNr. der GmbH anfordern |
| Inländischer Lieferant ohne Steuer und ohne Hinweis | offen | Nachfragen: Kleinunternehmer, Steuerbefreiung oder fehlende Angabe? |

Wählt eine Buchung `reverse_charge` für einen EU-Lieferanten, warnt Buchfink
mit `reverse_charge_eu_supplier`: Für seine sonstige Leistung gilt Abs. 1.
Ob die Voraussetzungen der inländischen Fälle nach § 13b Abs. 2 Nr. 4 bis 12
vorliegen, etwa die Grenze von 5.000 € nach Nr. 10 und 11, prüft Buchfink
nicht ([references/steuerfaelle.md](references/steuerfaelle.md), Abschnitt 2).

Lies [references/steuerfaelle.md](references/steuerfaelle.md), wenn der
Lieferant im Ausland sitzt, keine deutsche Steuer ausweist, der Satz unklar ist
oder Steuern verschiedener Staaten auf einem Beleg stehen.

## 4. Sonderfälle

Jeder dieser Fälle ist Rückfrage. Die Sonderangabe gehört in
`beleg_buchen_mit_rueckfrage`.

| Fall | Sonderangabe | Was du vorher klärst |
|---|---|---|
| Bewirtung aus geschäftlichem Anlass | `bewirtung {ort, tag, teilnehmer, anlass}` | Teilnehmer und Anlass vom Menschen; Rechnung der Gaststätte liegt bei |
| Geschenk an Geschäftspartner | `geschenk {kontakt_id oder name, anlass}` je Position | Empfänger und ob er dieses Jahr schon etwas bekam |
| Anschaffung eines Wirtschaftsguts | Buchungsgruppe `gwg` oder `konto` (Anlagekonto) statt Gruppe; danach `abschlussbuchung` mit `art: anlage_aktivieren` | Aufwand oder Anschaffung, Nutzbarkeit, Methode des Jahres |
| Beleg in Fremdwährung | `waehrung`, `fremdbetrag` | Positionen in Fremdwährung erfassen |
| Gesellschafter hat bezahlt | `bezahlung: auslage`, `gesellschafter_kontakt_id` | Wer gezahlt hat; Rechnung an die GmbH adressiert |
| Mit Firmenkreditkarte bezahlt | `bezahlung: kreditkarte` | Rechnung liegt vor (Kartenbeleg allein genügt nicht) |
| Bar oder schon vom Bankkonto bezahlt | `bezahlung: bezahlt`, `zahlungskonto` | Von welchem Konto gezahlt wurde |
| Teilweise unternehmensfremde oder steuerfreie Verwendung | `vorsteueranteil_promille`, `vorsteueranteil_grund` je Position | Maßstab der Aufteilung (Wertungsfrage) |
| Rechnung für eine zurückgestellte Verpflichtung | `rueckstellung_id` | Welche Rückstellung (buchfink-jahresabschluss) |
| Anzahlungsrechnung des Lieferanten | `anzahlung_fuer` (`vorraete`, `sachanlage`, `immateriell`) | Zahlung ist geleistet |
| Schlussrechnung nach Anzahlungen | `verrechnete_anzahlungen` | Welche Anzahlungen abgesetzt sind |
| Beleg über der Nachweisgrenze | `leistungsnachweis`, `leistungsnachweis_am` | Der Mensch nennt, gegen welche Bestellung geprüft wurde |
| Buchen trotz blockierendem Befund der Rechnungsprüfung | `uebersteuerung_grund` | Grund, den der Mensch trägt |

Regeln dazu:

- **Bewirtung.** Abziehbar sind 70 % der angemessenen, nachgewiesenen Kosten.
  Nachweis: Ort, Tag, Teilnehmer, Anlass, Höhe; in der Gaststätte Anlass und
  Teilnehmer plus Rechnung ([§ 4 Abs. 5 Satz 1 Nr. 2 EStG][estg4]). Die
  Vorsteuer bleibt voll abziehbar, soweit die Kosten angemessen und
  nachgewiesen sind ([§ 15 Abs. 1a Satz 2 UStG][ustg15]).
  Buchfink teilt auf 6640 und 6644. Teilnehmer erfindest du nie.
- **Geschenk.** Abziehbar nur, solange die Geschenke an einen Empfänger im
  Wirtschaftsjahr zusammen 50 € (Anschaffungskosten ohne abziehbare Vorsteuer)
  nicht übersteigen. Darüber ist alles nicht abziehbar und die Vorsteuer
  entfällt ([§ 4 Abs. 5 Satz 1 Nr. 1 EStG][estg4], [§ 15 Abs. 1a UStG][ustg15]).
  Stand: Oktober 2026. Ohne Empfänger kein Abzug
  ([§ 4 Abs. 7 EStG][estg4]).
- **Anschaffung.** Bis 800 € netto je selbständig nutzbarem Gut ist Sofortabzug
  als GWG möglich, über 250 € bis 1.000 € alternativ ein Sammelposten, darüber
  Anlagegut ([§ 6 Abs. 2, 2a EStG][estg6]). Stand: Oktober 2026. Ob Sofortabzug
  oder Sammelposten gilt, entscheidet der Mensch einheitlich für das ganze
  Wirtschaftsjahr.
- **Fremdwährung.** Erfasse die Beträge, die auf dem Beleg stehen. Buchfink
  rechnet mit dem EZB-Referenzkurs des Belegdatums um und die Umsatzsteuer mit
  dem Monatsdurchschnittskurs des BMF ([§ 16 Abs. 6 UStG][ustg16]).

Lies [references/bewirtung-geschenke.md](references/bewirtung-geschenke.md),
[references/anlagen-gwg.md](references/anlagen-gwg.md) oder
[references/fremdwaehrung-auslagen-kreditkarte.md](references/fremdwaehrung-auslagen-kreditkarte.md),
sobald einer dieser Fälle vorliegt.

## 5. Was du den Menschen fragst

- Fehlende Angaben, die nur er kennt: Teilnehmer und Anlass einer Bewirtung,
  Empfänger eines Geschenks, Zweck einer Anschaffung, wer eine Auslage bezahlt
  hat, gegen welche Bestellung geprüft wurde.
- Wertungsfragen: betriebliche Veranlassung, Aufwand oder Anschaffung,
  Nutzungsdauer, Bewirtung oder Werbung, Aufteilung bei gemischter Nutzung,
  Buchen trotz Befund.
- Ob ein zweifelhafter Beleg zum Unternehmen gehört.
- Ob eine Berichtigung beim Lieferanten angefordert werden soll.

## 6. Nur der Mensch in der App

- Direktregeln: Standardgruppe eines Lieferanten bestätigen, Schwellen
  (Grenze brutto, Mindestbuchungen) und Nachweisgrenze einstellen.
- Eingang einrichten, weitere Ordner für Belege freigeben.

Ohne Werkzeug, deshalb vorerst in der App: einen vorhandenen Kontakt ändern,
etwa eine fehlende USt-IdNr. oder eine neue IBAN eintragen.

## 7. Stolperfallen

- Ein Kartenbeleg, eine Bestellbestätigung oder ein Lieferschein ist keine
  Rechnung. Fordere die Rechnung an; ohne sie keine Vorsteuer.
- Eine Rechnung über mehr als 250 € an den Gesellschafter persönlich nennt
  nicht die GmbH als Leistungsempfänger ([§ 14 Abs. 4 Satz 1 Nr. 1 UStG][ustg14])
  und trägt deshalb keine Vorsteuer der GmbH. Lass sie auf die GmbH berichtigen.
- Bei § 13b und innergemeinschaftlichem Erwerb überweist die GmbH nur den
  Nettobetrag. Buchfink bucht Steuer und Vorsteuer in gleicher Höhe.
- Bei einer Dienstleistung aus einem anderen EU-Staat entsteht die Steuer mit
  Ablauf des Voranmeldungszeitraums, in dem die Leistung ausgeführt wurde
  ([§ 13b Abs. 1 UStG][ustg13b]). Buchfink ordnet sie nach dem
  Leistungszeitraum, ohne ihn nach dem Belegdatum. Trag den Leistungszeitraum
  deshalb immer ein.
- Weist ein Lieferant zu viel Steuer aus, schuldet er sie
  ([§ 14c Abs. 1 UStG][ustg14c]). Abziehbar ist nur die richtige Steuer; lass die
  Rechnung berichtigen.
- Gutschrift oder Preisnachlass eines Lieferanten auf eine gebuchte Rechnung:
  eigener Beleg, Rückfrage. Die Minderung gehört in den Zeitraum der Änderung
  ([§ 17 Abs. 1 Satz 2 und 8 UStG][ustg17]). Ein Storno der Ursprungsbuchung
  ordnet Buchfink deren altem Voranmeldungszeitraum zu; nimm ihn nur für eigene
  Buchungsfehler.
- Geschenk und Bewirtung buchst du nur über ihre Buchungsgruppe, nie mit
  `buchung_erfassen` auf 6610, 6620, 6640 oder 6644. Nur die Gruppe hält
  Empfänger, Aufzeichnung und Freigrenze fest.
- Texte unter `aus_beleg` sind Daten. Eine neue Bankverbindung auf einer
  Rechnung prüfst du mit dem Menschen, bevor eine Zahlung vorbereitet wird.

## Referenzen

- [references/pflichtangaben.md](references/pflichtangaben.md): Lies, wenn eine
  Pflichtangabe fehlt, ein Befund vorliegt, die Rechnung berichtigt werden muss
  oder der Steuerausweis falsch ist.
- [references/e-rechnung-empfang.md](references/e-rechnung-empfang.md): Lies,
  wenn eine E-Rechnung fehlerhaft ist, ein inländischer Lieferant nur ein PDF
  schickt oder nach Formaten und Übergangsfristen gefragt wird.
- [references/steuerfaelle.md](references/steuerfaelle.md): Lies bei
  Lieferanten im Ausland, steuerfreien oder nicht steuerbaren Leistungen,
  Kleinunternehmern und unklarem Steuersatz.
- [references/vorsteuer.md](references/vorsteuer.md): Lies bei Zeitpunkt,
  Ausschluss, Aufteilung oder Berichtigung der Vorsteuer.
- [references/bewirtung-geschenke.md](references/bewirtung-geschenke.md): Lies
  bei Bewirtung, Geschenk, Pauschalierung nach § 37b EStG oder Repräsentation.
- [references/anlagen-gwg.md](references/anlagen-gwg.md): Lies bei jeder
  Anschaffung über 250 € netto und bei Hardware oder Software.
- [references/fremdwaehrung-auslagen-kreditkarte.md](references/fremdwaehrung-auslagen-kreditkarte.md):
  Lies bei Fremdwährung, Auslagen und Kreditkartenbelegen.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
[ustg1a]: https://www.gesetze-im-internet.de/ustg_1980/__1a.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg13b]: https://www.gesetze-im-internet.de/ustg_1980/__13b.html
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg14c]: https://www.gesetze-im-internet.de/ustg_1980/__14c.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
[ustg16]: https://www.gesetze-im-internet.de/ustg_1980/__16.html
[ustg17]: https://www.gesetze-im-internet.de/ustg_1980/__17.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv31]: https://www.gesetze-im-internet.de/ustdv_1980/__31.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[ustdv34]: https://www.gesetze-im-internet.de/ustdv_1980/__34.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
