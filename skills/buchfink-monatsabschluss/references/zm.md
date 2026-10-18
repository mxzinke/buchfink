# Zusammenfassende Meldung (ZM)

Rechtsstand: 17. Oktober 2026. Wortlaut von § 18a, § 18b, § 26a UStG am
17.10.2026 gelesen; Übermittlungswege nach den Seiten des Bundeszentralamts
für Steuern (BZSt), abgerufen am 17.10.2026.

## Inhalt

1. Wann die GmbH meldet
2. Meldezeitraum und Frist
3. Inhalt und Abgleich
4. Übermittlung
5. Berichtigung

## 1. Wann die GmbH meldet

- Meldepflichtig sind steuerpflichtige sonstige Leistungen im übrigen
  Gemeinschaftsgebiet nach § 3a Abs. 2 UStG, für die der Empfänger in seinem
  Mitgliedstaat die Steuer schuldet ([§ 18a Abs. 2 UStG][ustg18a]). Das sind die
  Rechnungen an Unternehmer im EU-Ausland mit Hinweis auf die
  Steuerschuldnerschaft des Empfängers (buchfink-rechnungen).
- Innergemeinschaftliche Warenlieferungen sind ebenfalls meldepflichtig
  (§ 18a Abs. 1). Buchfink bildet den Warenverkehr auf der Ausgangsseite nicht
  ab. Verkauft die GmbH doch einen Gegenstand an einen EU-Unternehmer, sag das
  dem Menschen und empfiehl einen Steuerberater.
- In der Voranmeldung erscheinen dieselben Leistungen in Kennziffer 21
  ([§ 18b Satz 1 Nr. 2 UStG][ustg18b]).

## 2. Meldezeitraum und Frist

| Fall | Meldezeitraum | Frist | Fundstelle |
|---|---|---|---|
| nur sonstige Leistungen | Kalendervierteljahr | 25. Tag nach Quartalsende | [§ 18a Abs. 2 Satz 1 UStG][ustg18a] |
| Wahl der monatlichen Meldung (Anzeige an das BZSt) | Kalendermonat | 25. Tag nach Monatsende | § 18a Abs. 3 |

- Die Angaben gehören in den Meldezeitraum, in dem die Leistung ausgeführt
  wurde (§ 18a Abs. 8 Satz 2).
- Die Dauerfristverlängerung gilt nur für Voranmeldungen
  ([§ 46 UStDV][ustdv46]); das BZSt weist darauf ausdrücklich hin.
- Fällt der 25. auf Samstag, Sonntag oder Feiertag: nächster Werktag
  ([§ 108 Abs. 3 AO][ao108]).
- Für die ZM gelten die Vorschriften der AO über Steuererklärungen, ausgenommen
  § 152 AO (§ 18a Abs. 11): kein Verspätungszuschlag. Wer sie nicht, falsch
  oder verspätet abgibt oder nicht rechtzeitig berichtigt, handelt
  ordnungswidrig, wenn er vorsätzlich oder leichtfertig handelt; Geldbuße bis
  5.000 € ([§ 26a Abs. 2 Nr. 5, Abs. 3 UStG][ustg26a]).

Stand: Oktober 2026.

## 3. Inhalt und Abgleich

Je Empfänger: seine USt-IdNr., die Summe der Bemessungsgrundlagen und der
Hinweis auf eine sonstige Leistung (§ 18a Abs. 7 Satz 1 Nr. 3).

Prüfe mit `erklaerung` (`art: zm`, `zeitraum`):

1. Jede Zeile hat eine USt-IdNr., die Buchfink beim BZSt bestätigt hat. Fehlt
   die Bestätigung, meldet Buchfink einen Befund; kläre ihn mit
   buchfink-rechnungen.
2. Summe der Zeilen gleich Summe der Kennziffer 21 aus den Voranmeldungen
   desselben Zeitraums. Buchfink stellt beide gegenüber und speichert
   Differenzen. Eine Differenz bedeutet meist eine Buchung mit falschem
   Steuerfall.
3. Bemessungsgrundlagen in vollen Euro (Buchfink schneidet die Cent ab).

Das BZSt gleicht die ZM mit den Voranmeldungen ab und schreibt bei
Abweichungen an (BZSt, Berichtigung einer ZM).

## 4. Übermittlung

- Die ZM geht elektronisch an das BZSt, entweder über das Formular in
  Mein ELSTER oder über das BZSt-Online-Portal (BOP). Für das BOP ist eine
  zusätzliche Anmeldung zum Fachverfahren ZM nötig. Beide Formulare nehmen eine
  CSV-Datei mit den Meldezeilen an; ein Formular fasst bis zu 1.500
  Meldezeilen (BZSt, Elektronische Abgabe; Stand Oktober 2026).
- Buchfink übermittelt nicht; der Mensch sendet. `fristen` führt die ZM als
  Pflicht `zm` nur für Zeiträume mit Leistungen nach § 3a Abs. 2 UStG an
  Unternehmer im übrigen Gemeinschaftsgebiet.
- Ablauf: Der Mensch exportiert die CSV in der App (ein Werkzeug dafür gibt
  es nicht) → er importiert die Datei im Online-Formular, prüft die Zeilen
  gegen deine Tabelle aus `erklaerung`, sendet → er nennt dir Zeitpunkt und
  Transferticket → er erfasst sie in der App (nur App).
- Ob die CSV von Buchfink dem aktuellen Importformat des Formulars entspricht,
  ist nicht geprüft. Lehnt das Formular sie ab, trägt der Mensch die Zeilen von
  Hand ein; melde die Ablehnung als Fehler von Buchfink.

## 5. Berichtigung

- Erkennt die GmbH, dass eine abgegebene ZM unrichtig oder unvollständig ist,
  berichtigt sie diese innerhalb eines Monats ([§ 18a Abs. 10 UStG][ustg18a]).
- Je betroffenem Meldezeitraum eine eigene berichtigte ZM. Korrekt gemeldete
  Zeilen werden nicht wiederholt. Eine Zeile mit falscher USt-IdNr. wird mit
  Bemessungsgrundlage 0 storniert und mit der richtigen Nummer neu gemeldet
  (BZSt, Berichtigung einer ZM).
- Ist Kennziffer 21 einer Voranmeldung falsch, ist auch die Voranmeldung
  unverzüglich zu berichtigen ([§ 18b Satz 5 UStG][ustg18b]).
- In Buchfink: Die Berichtigung rechnet den Zeitraum vollständig neu und
  verknüpft sie mit der übermittelten Meldung. Die CSV enthält deshalb auch
  unveränderte Zeilen. Leg dem Menschen nur die Zeilen vor, die sich ändern
  oder storniert werden; nur diese übernimmt er in die berichtigte ZM.

[ustg18a]: https://www.gesetze-im-internet.de/ustg_1980/__18a.html
[ustg18b]: https://www.gesetze-im-internet.de/ustg_1980/__18b.html
[ustg26a]: https://www.gesetze-im-internet.de/ustg_1980/__26a.html
[ustdv46]: https://www.gesetze-im-internet.de/ustdv_1980/__46.html
[ao108]: https://www.gesetze-im-internet.de/ao_1977/__108.html

BZSt-Seiten:

- Elektronische Abgabe: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/ElektronischeAbgabe/elektronischeabgabe_zm_node.html
- Berichtigung einer ZM: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/BerichtigungZM/berichtigung_node.html
- Fristen: https://www.bzst.de/DE/Unternehmen/Umsatzsteuer/ZusammenfassendeMeldung/Fristen/fristen_node.html
