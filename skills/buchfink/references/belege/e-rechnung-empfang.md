# E-Rechnung empfangen

Rechtsstand: 17. Oktober 2026. Gesetz am Wortlaut gelesen. BMF-Schreiben vom
15.10.2024 über den Spiegel des LfSt Bayern gelesen, BMF-Schreiben vom
15.10.2025 über einen Kanzleispiegel (bundesfinanzministerium.de war nicht
abrufbar; Abgleich offen, [Quellenverzeichnis][quellen]).

## Inhalt

1. Begriffe
2. Wer eine E-Rechnung schicken muss
3. Übergangsfristen
4. Empfang
5. Folgen für die Vorsteuer
6. Fehlerarten und Befunde
7. Hybride Rechnung
8. Aufbewahrung

## 1. Begriffe

Begriffe, Formate und Pflichtinhalt:
[rechnungen/e-rechnung-ausgang.md, Abschnitte 1, 4 und 5](../rechnungen/e-rechnung-ausgang.md#1-begriffe).
Bilddateien und E-Mails sind sonstige Rechnungen
([BMF vom 15.10.2024, Rn. 7][bmf2024]).

## 2. Wer eine E-Rechnung schicken muss

Die Pflicht des Lieferanten regelt
[rechnungen/e-rechnung-ausgang.md, Abschnitt 2](../rechnungen/e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist).
Keine Pflicht besteht zudem bei Lieferanten im Ausland oder Nichtunternehmern,
bei Fahrausweisen ([§ 34 Abs. 1 Satz 2 UStDV][ustdv34]) und Kleinunternehmern
([§ 34a Satz 4 UStDV][ustdv34a]); für die 250 € zählt der Gesamtbetrag
([BMF vom 15.10.2024, Rn. 23][bmf2024]). Ohne Pflicht ist eine fehlende
E-Rechnung kein Prüfpunkt.

## 3. Übergangsfristen

Die Übergangsregeln des [§ 27 Abs. 38 UStG][ustg27] gelten für den Aussteller;
Tabelle in
[rechnungen/e-rechnung-ausgang.md, Abschnitt 3](../rechnungen/e-rechnung-ausgang.md#3-übergang-bis-ende-2027).
Die Zustimmung zu einem anderen elektronischen Format ist formfrei, auch
konkludent ([BMF vom 15.10.2024, Rn. 63][bmf2024]).

## 4. Empfang

- Seit dem 01.01.2025 muss jedes inländische Unternehmen E-Rechnungen
  empfangen können; ein E-Mail-Postfach genügt
  ([BMF vom 15.10.2024, Rn. 40, 62][bmf2024]; Abschnitt 14.1 Abs. 5 UStAE in
  der Fassung des [BMF-Schreibens vom 15.10.2025][bmf2025]).
- Lege die Datei unverändert ab. Die PDF-Ansicht einer reinen XRechnung kommt
  vom Lieferanten oder aus einem Anzeigeprogramm des Menschen.

## 5. Folgen für die Vorsteuer

Nach [BMF vom 15.10.2024][bmf2024]:

- Besteht die Pflicht ohne Übergangsregel, berechtigt eine sonstige Rechnung
  dem Grunde nach nicht zum Vorsteuerabzug (Rn. 55, 56).
- Eine ersetzende E-Rechnung mit eindeutigem Bezug ist eine Berichtigung und
  wirkt zurück (Rn. 57). Ohne sie ist die Vorsteuer über objektive Nachweise
  möglich, bei inhaltlich richtiger und vollständiger Rechnung regelmäßig
  (Rn. 58).
- Das falsche Format allein wird nicht beanstandet, wenn die GmbH annehmen
  durfte, dass der Lieferant eine Übergangsregel nutzen darf (Rn. 59).

Papier- oder PDF-Rechnung eines inländischen Lieferanten über 250 €:

1. Leistung vor 2027, Rechnung bis 31.12.2026 übermittelt: buchen wie bisher.
   Erst ab 2027 übermittelt: keine Übergangsregel; weiter wie unter 3.
2. Leistung 2027: Frag, ob der Lieferant klein genug für die Übergangsregel ist;
   seinen Vorjahresumsatz kennt Buchfink nicht. Unklar: E-Rechnung anfordern
   lassen, buchen nach Entscheidung (Wertungsfrage).
3. Leistung ab 2028: E-Rechnung anfordern lassen; bis dahin buchen nur nach
   Entscheidung des Menschen.

## 6. Fehlerarten und Befunde

| Klasse aus `beleg_lesen` | Was sie ist | Dein Vorgehen | Fundstelle |
|---|---|---|---|
| Formatfehler | Datei verletzt die Syntax oder lässt keine vollständige Extraktion zu; sie ist dann eine sonstige Rechnung | neuen Datensatz anfordern lassen; ob die sonstige Rechnung genügt, nach Abschnitt 5 | [BMF vom 15.10.2025, Rn. 6a][bmf2025] |
| Geschäftsregelfehler | Regel des Formats verletzt (EN 16931 oder XRechnung, z. B. fehlende Buyer reference); unbeachtlich, wenn keine Pflichtangabe betroffen ist | mit `uebersteuerung_grund` buchen | Rn. 6b, 35a |
| Inhaltsfehler | Pflichtangabe nach §§ 14, 14a UStG fehlt oder ist falsch, auch ohne Validierungsfehler (falscher Steuersatz); kein Vorsteuerabzug ohne Berichtigung | Berichtigung anfordern ([pflichtangaben.md](pflichtangaben.md#5-fehlende-oder-falsche-angaben)); buchen nur nach Entscheidung des Menschen | Rn. 35a |

- `geprueft: false` heißt: kein strukturierter Teil. Prüfe die Pflichtangaben
  selbst an der Datei ([pflichtangaben.md](pflichtangaben.md)).
- Auf eine geeignete Validierung darf sich die GmbH bei kaufmännischer
  Sorgfalt verlassen; einen falschen Steuersatz erkennt sie nicht
  ([BMF vom 15.10.2025, Rn. 35a][bmf2025]).

## 7. Hybride Rechnung

Weicht der Bildteil einer ZUGFeRD-Datei bei Pflichtangaben vom Datensatz ab,
kann er eine weitere Rechnung mit Folgen nach § 14c UStG sein; Vorsteuer gibt
es nur aus dem strukturierten Teil (Abschnitt 14c.1 Abs. 4a UStAE in der
Fassung des [BMF-Schreibens vom 15.10.2025][bmf2025]). Buchfink bucht aus dem
Datensatz und zeigt Abweichungen an; melde jede dem Menschen.

## 8. Aufbewahrung

Bei einer E-Rechnung genügt der strukturierte Teil in seiner ursprünglichen
Form; Einzelheiten in [grundlagen/aufbewahrung.md](../grundlagen/aufbewahrung.md).

[quellen]: https://github.com/mxzinke/buchfink/blob/master/skills/QUELLEN.md
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv34]: https://www.gesetze-im-internet.de/ustdv_1980/__34.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
