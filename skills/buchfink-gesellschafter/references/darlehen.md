# Darlehen zwischen GmbH und Gesellschafter

Rechtsstand: 17. Oktober 2026. Bankdarlehen regelt buchfink-bank.

## Inhalt

1. Vertrag
2. Gesellschafter leiht der GmbH
3. GmbH leiht dem Gesellschafter
4. Zinsen und Steuern
5. Insolvenz, Nachrang, Rangrücktritt
6. Buchungen

## 1. Vertrag

- Ein Darlehen verpflichtet zur Auszahlung, zur Zinszahlung und zur
  Rückzahlung bei Fälligkeit; ohne bestimmte Laufzeit gilt eine Kündigungsfrist
  von drei Monaten ([§ 488 BGB][bgb488]).
- Mit einem beherrschenden Gesellschafter braucht es eine klare, im Voraus
  geschlossene und tatsächlich durchgeführte Vereinbarung, sonst droht eine vGA
  ([R 8.5 Abs. 2 KStR 2022][kstr85]).
- Ein-Personen-GmbH mit dem Gesellschafter als alleinigem Geschäftsführer:
  Befreiung von § 181 BGB und Niederschrift über den Vertrag
  ([§ 35 Abs. 3 GmbHG][gmbhg35]).
- Frag nach dem Vertrag, lege ihn mit `beleg_ablegen` ab und erfasse ihn nach
  dem Ja mit `darlehen_erfassen`, bevor du den ersten Zahlungsfluss buchst. Fehlt er, empfiehl dem Menschen, ihn schriftlich
  nachzuholen; die Wirkung „im Voraus“ heilt ein späterer Vertrag nicht.
  Wertungsfrage, Steuerberater.

## 2. Gesellschafter leiht der GmbH

- Konto nach vereinbarter Restlaufzeit: 3641 bis 1 Jahr, 3642 1 bis 5 Jahre,
  3643 über 5 Jahre (Verbindlichkeiten gegenüber GmbH-Gesellschaftern). Der
  Betrag mit Restlaufzeit bis zu einem Jahr ist zu vermerken
  ([§ 268 Abs. 5 Satz 1 HGB][hgb268]).
- Zins: Wertungsfrage. Ein Zins über dem, was ein fremder Darlehensgeber
  erhielte, ist im übersteigenden Teil vGA-verdächtig
  ([§ 8 Abs. 3 Satz 2 KStG][kstg8]). Ein niedriger Zins oder ein zinsloses
  Darlehen belastet die GmbH nicht.
- Die Rückzahlung eines Gesellschafterdarlehens fällt nicht unter das
  Auszahlungsverbot ([§ 30 Abs. 1 Satz 3 GmbHG][gmbhg30]). Insolvenzrisiken in
  Abschnitt 5.

## 3. GmbH leiht dem Gesellschafter

- Zulässig nur, wenn der Rückzahlungsanspruch vollwertig ist
  ([§ 30 Abs. 1 Satz 2 GmbHG][gmbhg30]). Wertungsfrage: Bonität des
  Gesellschafters, Sicherheit. Geschäftsführer, Prokuristen und
  Generalbevollmächtigte dürfen keinen Kredit aus dem Vermögen erhalten, das das
  Stammkapital deckt ([§ 43a GmbHG][gmbhg43a]).
- Zinslos oder zu niedrig verzinst: verhinderte Vermögensmehrung, vGA-Verdacht
  ([R 8.5 Abs. 1 KStR 2022][kstr85]).
- Konto: Soll das Darlehen dauernd dem Geschäftsbetrieb dienen, gehört es ins
  Anlagevermögen ([§ 247 Abs. 2 HGB][hgb247]): 0960 Ausleihungen an
  Gesellschafter. Sonst 1308 (Restlaufzeit bis 1 Jahr) oder 1309 (über 1 Jahr).
  Forderungen mit Restlaufzeit über einem Jahr sind zu vermerken
  ([§ 268 Abs. 4 Satz 1 HGB][hgb268]). Die Einordnung ist eine Wertungsfrage.
- Zinsen an die GmbH: Ertrag 7110 Sonstige Zinserträge.

## 4. Zinsen und Steuern

- Zinsaufwand der GmbH: 7316 Zinsen für Gesellschafterdarlehen; bei einem
  Gesellschafter mit mehr als 25 % Beteiligung führt der SKR04 das Konto 7317
  (Kontenbezeichnung). Frag nach der Beteiligungsquote.
- Kapitalertragsteuer: Auf Zinsen eines gewöhnlichen Darlehens behält die GmbH
  keine Kapitalertragsteuer ein; § 43 Abs. 1 Satz 1 Nr. 7 EStG erfasst nur
  Zinsen aus registrierten Anleihen und Zinsen, die Kreditinstitute schulden
  ([§ 43 Abs. 1 Satz 1 Nr. 7 EStG][estg43]). Gewinnabhängige Zinsen
  (partiarisches Darlehen) oder eine stille Beteiligung unterliegen dem Abzug
  ([§ 43 Abs. 1 Satz 1 Nr. 3, § 20 Abs. 1 Nr. 4 EStG][estg43]): Grenze,
  Steuerberater.
- Hinweis für den Gesellschafter: Zinsen an einen zu mindestens 10 %
  beteiligten Gesellschafter fallen nicht unter den gesonderten Steuertarif
  für Kapitalerträge ([§ 32d Abs. 2 Nr. 1 Buchst. b EStG][estg32d]). Seine
  Einkommensteuer bildet Buchfink nicht ab.
- Zinsen für Schulden werden bei der Gewerbesteuer teilweise hinzugerechnet;
  das regelt buchfink-steuererklaerungen.

## 5. Insolvenz, Nachrang, Rangrücktritt

- Forderungen auf Rückgewähr eines Gesellschafterdarlehens sind im
  Insolvenzverfahren nachrangig ([§ 39 Abs. 1 Satz 1 Nr. 5 InsO][inso39]).
  Ausgenommen ist ein nicht geschäftsführender Gesellschafter mit höchstens
  10 % am Haftkapital ([§ 39 Abs. 5 InsO][inso39]).
- Anfechtbar sind eine Rückzahlung im letzten Jahr vor dem Insolvenzantrag und
  eine Sicherung in den letzten zehn Jahren
  ([§ 135 Abs. 1 InsO][inso135]). Weise vor jeder Tilgung auf diese Frist hin,
  wenn die Zahlen auf eine Krise deuten (buchfink-grundlagen, Warnzeichen).
- Ein vereinbarter Rangrücktritt hinter alle Forderungen nach § 39 Abs. 1
  Nr. 1 bis 5 InsO nimmt das Darlehen aus der Überschuldungsprüfung
  ([§ 19 Abs. 2 Satz 2 InsO][inso19]).
- Steuerlich: Ist die Verbindlichkeit nur aus künftigen Gewinnen oder Einnahmen
  zu erfüllen, darf sie in der Steuerbilanz erst angesetzt werden, wenn diese
  anfallen ([§ 5 Abs. 2a EStG][estg5]); die Ausbuchung kann einen Gewinn
  auslösen. Die Formulierung eines Rangrücktritts gehört zu Anwalt und
  Steuerberater. Buchfink führt eine Einheitsbilanz und bildet abweichende
  Steuerbilanzwerte hier nicht ab.

## 6. Buchungen

| Vorgang | Soll | Haben | Werkzeug |
|---|---|---|---|
| Gesellschafter zahlt Darlehen aus | Bank | 3641 bis 3643 | `bankumsatz_buchen`, Zeile `darlehen_id` |
| Zinsen fällig, noch nicht gezahlt | 7316 oder 7317 | 3640 | `darlehen_erfassen` mit `art: zinsen`, Eigenbeleg mit der Zinsrechnung |
| Zinszahlung | 3640 | Bank | `bankumsatz_buchen`, Zeile `konto` |
| Tilgung | 3641 bis 3643 | Bank | `bankumsatz_buchen`, Zeile `darlehen_id` |
| GmbH zahlt Darlehen an Gesellschafter aus | 0960, 1308 oder 1309 | Bank | `bankumsatz_buchen`, Zeile `darlehen_id` |
| Gesellschafter tilgt | Bank | 0960, 1308 oder 1309 | `bankumsatz_buchen`, Zeile `darlehen_id` |

Das Konto des Darlehens legt `darlehen_erfassen` aus dem Vertrag fest. Alle
Darlehensbuchungen sind Rückfrage. Auszahlung und Tilgung sind kein Umsatz
(`not_taxable` bei `buchung_erfassen`). Zinsen, die die GmbH für einen Kredit erhält, sind
steuerfrei ([§ 4 Nr. 8 Buchst. a UStG][ustg4], `exempt`); Zinsen an einen
privaten Gesellschafter tragen keine Umsatzsteuer. Zinsen zum Jahresende
abgrenzen regelt buchfink-jahresabschluss.

[bgb488]: https://www.gesetze-im-internet.de/bgb/__488.html
[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg35]: https://www.gesetze-im-internet.de/gmbhg/__35.html
[gmbhg43a]: https://www.gesetze-im-internet.de/gmbhg/__43a.html
[hgb247]: https://www.gesetze-im-internet.de/hgb/__247.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[estg5]: https://www.gesetze-im-internet.de/estg/__5.html
[estg32d]: https://www.gesetze-im-internet.de/estg/__32d.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[inso19]: https://www.gesetze-im-internet.de/inso/__19.html
[inso39]: https://www.gesetze-im-internet.de/inso/__39.html
[inso135]: https://www.gesetze-im-internet.de/inso/__135.html
[kstr85]: https://www.steuerschroeder.de/steuergesetze/kstr/8.5
