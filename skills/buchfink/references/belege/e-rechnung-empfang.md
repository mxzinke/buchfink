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
6. Fehlerarten
7. Hybride Rechnung
8. Aufbewahrung

## 1. Begriffe

- **E-Rechnung:** in einem strukturierten elektronischen Format ausgestellt,
  übermittelt und empfangen, elektronisch verarbeitbar
  ([§ 14 Abs. 1 Satz 3 UStG][ustg14]). Das Format entspricht der EN 16931 oder
  ist zwischen den Parteien vereinbart und in ein EN-16931-Format extrahierbar
  ([§ 14 Abs. 1 Satz 6 UStG][ustg14]). Beispiele: XRechnung, ZUGFeRD ab Profil
  EN 16931.
- **Sonstige Rechnung:** Papier oder anderes elektronisches Format
  ([§ 14 Abs. 1 Satz 4 UStG][ustg14]). Dazu zählen PDF ohne eingebetteten
  Datensatz, Bilddateien und E-Mails ([BMF vom 15.10.2024, Rn. 7][bmf2024]).
- Alle Pflichtangaben müssen im strukturierten Teil stehen. Ein Verweis auf eine
  Anlage oder einen Link genügt nicht; Ergänzungen wie Stundennachweise dürfen
  als Anhang in der E-Rechnung liegen ([BMF vom 15.10.2025, Rn. 35][bmf2025]).

## 2. Wer eine E-Rechnung schicken muss

- Pflicht für Leistungen zwischen Unternehmern, die beide im Inland ansässig
  sind ([§ 14 Abs. 2 Satz 2 Nr. 1, Satz 3 UStG][ustg14]). Das gilt auch für
  Gutschriften und für Rechnungen über § 13b-Leistungen zwischen Inländern
  ([BMF vom 15.10.2025, Rn. 17][bmf2025]).
- Keine Pflicht und damit kein Prüfpunkt:
  - Lieferant im Ausland.
  - Leistung nach § 4 Nr. 8 bis 29 UStG steuerfrei, etwa Versicherung,
    Kontoführung, Miete ohne Option ([§ 14 Abs. 2 Satz 2 UStG][ustg14]).
  - Rechnung bis 250 € brutto ([§ 33 Satz 4 UStDV][ustdv33]); maßgeblich ist
    der Gesamtbetrag ([BMF vom 15.10.2024, Rn. 23][bmf2024]).
  - Fahrausweis ([§ 34 Abs. 1 Satz 2 UStDV][ustdv34]).
  - Rechnung eines Kleinunternehmers ([§ 34a Satz 4 UStDV][ustdv34a]).
  - Lieferant ist kein Unternehmer.

## 3. Übergangsfristen

Die Übergangsregeln gelten nur für den Aussteller
([§ 27 Abs. 38 UStG][ustg27]). Stand: Oktober 2026.

| Leistung ausgeführt | Sonstige Rechnung zulässig? |
|---|---|
| 2025 und 2026 | ja, Papier immer; anderes elektronisches Format mit Zustimmung des Empfängers (Nr. 1). Rechnung muss bis 31.12.2026 übermittelt sein |
| 2027 | nur wenn der Gesamtumsatz des Ausstellers im Vorjahr höchstens 800.000 € betrug (Nr. 2); EDI-Verfahren mit Zustimmung (Nr. 3). Bis 31.12.2027 |
| ab 2028 | nein |

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
  `beleg_ablegen` ab. Buchfink liest den Datensatz aus und prüft ihn. Zu einer
  reinen XRechnung gehört eine PDF-Ansicht mit `rolle: darstellung`; ohne sie
  lässt sich der Beleg nicht buchen. Die Ansicht kommt vom Lieferanten oder aus
  einem Anzeigeprogramm des Menschen.

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

## 6. Fehlerarten

| Art | Was sie ist | Folge | Fundstelle |
|---|---|---|---|
| Formatfehler | Datei entspricht nicht der Syntax oder lässt keine vollständige Extraktion zu | Die Datei ist eine sonstige Rechnung | [BMF vom 15.10.2025, Rn. 6a][bmf2025] |
| Geschäftsregelfehler | Regeln des Formats verletzt (z. B. fehlende Buyer reference) | umsatzsteuerlich unbeachtlich, wenn keine Pflichtangabe betroffen ist | Rn. 6b, 35a |
| Inhaltsfehler | Pflichtangabe nach §§ 14, 14a UStG fehlt oder ist falsch; auch ohne Validierungsfehler möglich (falscher Steuersatz) | Rechnung nicht ordnungsmäßig | Rn. 35a |

Auf das Ergebnis einer geeigneten Validierung darf sich das Unternehmen bei
kaufmännischer Sorgfalt verlassen; den Validierungsbericht aufzubewahren bietet
sich an ([BMF vom 15.10.2025, Rn. 35a][bmf2025]). Buchfink speichert die
Befunde am Beleg.

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
Form. Einzelheiten stehen in buchfink-grundlagen (Aufbewahrung).

[quellen]: https://github.com/mxzinke/buchfink/blob/main/skills/QUELLEN.md
[ustg14]: https://www.gesetze-im-internet.de/ustg_1980/__14.html
[ustg27]: https://www.gesetze-im-internet.de/ustg_1980/__27.html
[ustdv33]: https://www.gesetze-im-internet.de/ustdv_1980/__33.html
[ustdv34]: https://www.gesetze-im-internet.de/ustdv_1980/__34.html
[ustdv34a]: https://www.gesetze-im-internet.de/ustdv_1980/__34a.html
[bmf2024]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/E-Rechnung/2024-10-15-einfuehrung-e-rechnung.pdf
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/10/2025-10-15-einfuehrung-obligatorische-e-rechnung.pdf
