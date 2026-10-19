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

- Was eine E-Rechnung und was eine sonstige Rechnung ist, welche Formate
  genügen und was im strukturierten Teil stehen muss:
  [rechnungen/e-rechnung-ausgang.md, Abschnitte 1, 4 und 5](../rechnungen/e-rechnung-ausgang.md#1-begriffe).
- Auf der Eingangsseite zählen zur sonstigen Rechnung auch Bilddateien und
  E-Mails ([BMF vom 15.10.2024, Rn. 7][bmf2024]).

## 2. Wer eine E-Rechnung schicken muss

- Wann ein Lieferant eine E-Rechnung ausstellen muss und wann nicht, steht in
  [rechnungen/e-rechnung-ausgang.md, Abschnitt 2](../rechnungen/e-rechnung-ausgang.md#2-wann-eine-e-rechnung-pflicht-ist).
  Wo keine Pflicht besteht, ist das Fehlen einer E-Rechnung kein Prüfpunkt.
- Auf der Eingangsseite kommen hinzu: Lieferant im Ausland oder kein
  Unternehmer; Fahrausweis ([§ 34 Abs. 1 Satz 2 UStDV][ustdv34]); Rechnung
  eines Kleinunternehmers ([§ 34a Satz 4 UStDV][ustdv34a]). Für die Grenze von
  250 € zählt der Gesamtbetrag ([BMF vom 15.10.2024, Rn. 23][bmf2024]).

## 3. Übergangsfristen

Die Übergangsregeln des [§ 27 Abs. 38 UStG][ustg27] gelten nur für den
Aussteller; die Tabelle steht in
[rechnungen/e-rechnung-ausgang.md, Abschnitt 3](../rechnungen/e-rechnung-ausgang.md#3-übergang-bis-ende-2027).

- Die Zustimmung zu einem anderen elektronischen Format braucht keine Form und
  kann konkludent erfolgen ([BMF vom 15.10.2024, Rn. 63][bmf2024]).
- Den Vorjahresumsatz eines Lieferanten kennt Buchfink nicht. Buchfink zeigt
  für 2027 deshalb nur die Bedingung an. Du fragst den Menschen, ob er etwas
  über die Größe des Lieferanten weiß.

## 4. Empfang

- Jedes inländische Unternehmen muss E-Rechnungen seit dem 01.01.2025 empfangen
  können; dafür gibt es keine Übergangsregel. Ein E-Mail-Postfach genügt
  ([BMF vom 15.10.2024, Rn. 40, 62][bmf2024]; Abschnitt 14.1 Abs. 5 UStAE in der
  Fassung des [BMF-Schreibens vom 15.10.2025][bmf2025]).
- Wer eine E-Rechnung nicht annimmt, hat keinen Anspruch auf eine andere Form
  ([BMF vom 15.10.2024, Rn. 42][bmf2024]).
- In Buchfink: Lege die empfangene Datei unverändert aus dem Eingang mit
  `beleg_ablegen` ab; Buchfink liest den Datensatz aus und prüft ihn. Eine
  reine XRechnung braucht eine PDF-Ansicht
  ([ablauf.md](ablauf.md#1-ablauf), Schritt 2). Die Ansicht kommt vom
  Lieferanten oder aus einem Anzeigeprogramm des Menschen.

## 5. Folgen für die Vorsteuer

- Besteht die Pflicht zur E-Rechnung und keine Übergangsregel, ist eine
  sonstige Rechnung keine ordnungsmäßige Rechnung. Sie berechtigt dem Grunde
  nach nicht zum Vorsteuerabzug ([BMF vom 15.10.2024, Rn. 55, 56][bmf2024]).
- Ersatz durch eine E-Rechnung mit eindeutigem Bezug auf die ursprüngliche
  Rechnung ist eine Berichtigung; sie wirkt auf die Ausstellung der sonstigen
  Rechnung zurück ([BMF vom 15.10.2024, Rn. 57][bmf2024]).
- Ohne Berichtigung kann die Vorsteuer unter strengem Maßstab über objektive
  Nachweise möglich sein; bei einer inhaltlich richtigen und vollständigen
  sonstigen Rechnung regelmäßig ([BMF vom 15.10.2024, Rn. 58][bmf2024]).
- Das falsche Format allein wird nicht beanstandet, wenn die GmbH davon
  ausgehen konnte, dass der Lieferant eine Übergangsregel nutzen durfte.
  Bekanntes über Größe und Vorjahresumsatz des Lieferanten ist zu
  berücksichtigen ([BMF vom 15.10.2024, Rn. 59][bmf2024]).

Dein Vorgehen bei einer Papier- oder PDF-Rechnung eines inländischen
Lieferanten über 250 €:

1. Leistung vor 2027, Rechnung bis 31.12.2026 übermittelt: buchen wie bisher;
   Buchfink zeigt einen Hinweis. Leistung vor 2027, Rechnung erst ab 2027
   übermittelt: keine Übergangsregel (Nr. 1 endet, Nr. 2 und 3 gelten nur für
   Umsätze ab 2027); weiter wie unter 3.
2. Leistung 2027: Frag den Menschen, ob der Lieferant klein genug für die
   Übergangsregel ist. Unklar: E-Rechnung anfordern lassen; buchen nach
   Entscheidung (Wertungsfrage).
3. Leistung ab 2028: E-Rechnung anfordern lassen. Bis sie da ist, ist die
   Vorsteuer gefährdet; buchen nur nach Entscheidung des Menschen.

## 6. Fehlerarten und Befunde

`beleg_lesen` liefert `befunde {geprueft, gesamt, blockierend, liste}`, je
Befund `klasse`, `regel`, `schwere`, `text`, `norm` und `blockiert`. Die Klassen
folgen der Verwaltung:

| Klasse | Was sie ist | Folge | Dein Vorgehen | Fundstelle |
|---|---|---|---|---|
| Formatfehler | Datei entspricht nicht der Syntax oder lässt keine vollständige Extraktion zu | Die Datei ist eine sonstige Rechnung | Lieferanten um neuen Datensatz bitten lassen. Ob die sonstige Rechnung nach der Übergangsregel genügt, prüfst du nach Abschnitt 3 | [BMF vom 15.10.2025, Rn. 6a][bmf2025] |
| Geschäftsregelfehler | Regeln des Formats verletzt (EN 16931 oder XRechnung, z. B. fehlende Buyer reference) | umsatzsteuerlich unbeachtlich, wenn keine Pflichtangabe betroffen ist | mit Grund buchen (`uebersteuerung_grund`) | Rn. 6b, 35a |
| Inhaltsfehler | Pflichtangabe nach §§ 14, 14a UStG fehlt oder ist falsch; auch ohne Validierungsfehler möglich (falscher Steuersatz) | Rechnung nicht ordnungsmäßig; kein Vorsteuerabzug ohne Berichtigung | Berichtigung anfordern ([pflichtangaben.md](pflichtangaben.md#5-fehlende-oder-falsche-angaben)); buchen nur nach Entscheidung des Menschen | Rn. 35a |

- `geprueft: false` heißt: Es gab keinen strukturierten Teil zu prüfen. Das ist
  kein „ohne Befund“. Prüfe die Pflichtangaben dann selbst an der Datei
  ([pflichtangaben.md](pflichtangaben.md)).
- Eine Validierung ersetzt nicht die Prüfung auf Vollständigkeit und
  Richtigkeit, etwa eines falschen Steuersatzes. Auf das Ergebnis einer
  geeigneten Validierung darf sich das Unternehmen bei kaufmännischer Sorgfalt
  verlassen; den Validierungsbericht aufzubewahren bietet sich an
  ([BMF vom 15.10.2025, Rn. 35a][bmf2025]). Buchfink speichert die Befunde am
  Beleg.
- Buchen trotz blockierendem Befund geht nur mit `uebersteuerung_grund`; den
  Grund entscheidet der Mensch (Wertungsfrage).

## 7. Hybride Rechnung

- Eine ZUGFeRD-Datei enthält Bildteil und Datensatz. Ist der Bildteil inhaltlich
  gleich, ist er ein Mehrstück. Weicht er bei Pflichtangaben ab, kann er eine
  weitere Rechnung mit Folgen nach § 14c UStG sein. Vorsteuer gibt es nur aus
  dem strukturierten Teil (Abschnitt 14c.1 Abs. 4a UStAE in der Fassung des
  [BMF-Schreibens vom 15.10.2025][bmf2025]).
- Buchfink bucht aus dem Datensatz und zeigt Abweichungen an. Melde jede
  Abweichung dem Menschen; bewerten tut er sie.

## 8. Aufbewahrung

Bei einer E-Rechnung genügt der strukturierte Teil in seiner ursprünglichen
Form. Einzelheiten stehen in
[grundlagen/aufbewahrung.md](../grundlagen/aufbewahrung.md).

[quellen]: https://github.com/mxzinke/buchfink/blob/master/skills/QUELLEN.md
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv34]: https://www.gesetze-im-internet.de/ustdv_1980/__34.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
