# Darlehen: Bank und Gesellschafter

Rechtsstand: 17. Oktober 2026. Wortlaut von HGB, EStG, KStG, GewStG und UStG am
17.10.2026 gelesen. Werkzeugstand: Buchfink 0.3.0.

Darlehen an Töchter:
[beteiligungen/erwerb-ausweis.md](../beteiligungen/erwerb-ausweis.md).
Gesellschafterdarlehen im Arbeitsablauf:
[gesellschafter/ablauf.md](../gesellschafter/ablauf.md#4-darlehen).

## Inhalt

1. Vertrag und Belege
2. Ansatz und Ausweis
3. Zinsen: Umsatzsteuer, Gewerbesteuer, Stichtag
4. Bankdarlehen: Konten
5. Bankdarlehen: Auszahlung und Rate
6. Disagio und Bearbeitungsentgelt
7. Kontokorrent
8. Gesellschafter leiht der GmbH
9. GmbH leiht dem Gesellschafter
10. Zinsen an und von Gesellschaftern
11. Insolvenz, Nachrang, Rangrücktritt
12. Buchungen eines Gesellschafterdarlehens

## 1. Vertrag und Belege

- Lege den Darlehensvertrag (beim Bankdarlehen mit Tilgungsplan) ab, bevor die
  Auszahlung gebucht wird; er ist Ursprungsbeleg aller Folgebuchungen
  ([GoBD Rz. 81][gobd]). Ebenso die jährliche Zins- und Saldenbestätigung der
  Bank.
- Mit einem beherrschenden Gesellschafter braucht es eine klare, im Voraus
  geschlossene und durchgeführte Vereinbarung, sonst droht eine vGA
  ([gesellschafter/vga-einlage.md](../gesellschafter/vga-einlage.md#1-was-eine-vga-ist);
  Ein-Personen-GmbH: [gesellschafter/ablauf.md](../gesellschafter/ablauf.md#3-grundregeln)).
  Fehlt der Vertrag, empfiehl, ihn schriftlich nachzuholen; für die
  Vergangenheit heilt das nicht. Wertungsfrage, Steuerberater.

## 2. Ansatz und Ausweis

Darlehen stehen mit dem Erfüllungsbetrag in der Bilanz, dem
Rückzahlungsbetrag ([§ 253 Abs. 1 Satz 2 HGB][hgb253]); je Darlehen ein Konto
der passenden Restlaufzeit (Abschnitte 4, 8, 9). Restlaufzeitvermerk und
Umgliederung zum Stichtag:
[jahresabschluss/forderungen.md](../jahresabschluss/forderungen.md).

## 3. Zinsen: Umsatzsteuer, Gewerbesteuer, Stichtag

- Kreditzinsen sind steuerfrei ([§ 4 Nr. 8 Buchst. a UStG][ustg4], `exempt`),
  ohne Vorsteuer; Zinsen an einen privaten Gesellschafter ohne Umsatzsteuer.
  Auszahlung und Tilgung sind kein Umsatz (`not_taxable` bei
  `buchung_erfassen`).
- **Gewerbesteuer:** Ein Viertel der Entgelte für Schulden und weiterer
  Finanzierungsanteile über 200.000 € wird hinzugerechnet
  ([§ 8 Nr. 1 GewStG][gewstg8]; Stand: Oktober 2026; Rechnung in
  [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)). Buche
  Zinsen deshalb getrennt von Gebühren.
- **Zinsschranke** ab 3 Mio. € Zinsaufwand:
  [darlehen-sonderfaelle.md](darlehen-sonderfaelle.md#4-zinsschranke).
- **Stichtag:** Zinsen für das alte Jahr, die erst im neuen belastet werden,
  grenzt [jahresabschluss/abgrenzung.md](../jahresabschluss/abgrenzung.md) ab.

## 4. Bankdarlehen: Konten

- Verbindlichkeiten gegenüber Kreditinstituten 3150; nach Restlaufzeit 3151
  (bis 1 Jahr), 3160 (1 bis 5 Jahre), 3170 (über 5 Jahre).
- Zinsen 7310 (kurzfristig), 7320 (langfristig), 7318 (Kontokorrent).
- Disagio 1940, Abschreibung darauf 7323.

Bei mehreren Bankdarlehen frag, wie sie getrennt werden (Text, Konto).

## 5. Bankdarlehen: Auszahlung und Rate

- **Auszahlung:** `bankumsatz_buchen`, Zeile `konto` (Darlehenskonto) über
  den Nennbetrag; je Einbehalt (Disagio, Entgelt) eine Zeile mit negativem
  `betrag` (Abschnitt 6).
- **Rate:** nach Tilgungsplan oder Auszug in Tilgung (Darlehenskonto) und
  Zinsen (7320 bzw. 7310). Fehlt die Aufteilung, frag nach dem Tilgungsplan;
  selbst rechnest du nur auf Bitte und mit Bestätigung des Menschen.

## 6. Disagio und Bearbeitungsentgelt

Disagio aktivieren (1940) und über die Laufzeit abschreiben; Bearbeitungsentgelte
sind Wertungsfrage. Herleitung und Quellen:
[darlehen-sonderfaelle.md](darlehen-sonderfaelle.md#1-disagio-und-bearbeitungsentgelt).

## 7. Kontokorrent

Ein Sollsaldo (Dispositionskredit) bleibt auf dem Bankkonto, Zinsen auf 7318.
Einen dauerhaft negativen Saldo meldest du als Warnzeichen
([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest)).

## 8. Gesellschafter leiht der GmbH

- Konto nach vereinbarter Restlaufzeit: 3641 bis 1 Jahr, 3642 1 bis 5 Jahre,
  3643 über 5 Jahre.
- Ein Zins über dem Fremdvergleich ist im übersteigenden Teil vGA-verdächtig
  ([§ 8 Abs. 3 Satz 2 KStG][kstg8]); niedriger Zins oder zinslos belastet die
  GmbH nicht. Wertungsfrage.
- Die Rückzahlung fällt nicht unter das Auszahlungsverbot
  ([§ 30 Abs. 1 Satz 3 GmbHG][gmbhg30]); Insolvenz in Abschnitt 11.

## 9. GmbH leiht dem Gesellschafter

- Zulässig nur bei vollwertigem Rückzahlungsanspruch
  ([§ 30 Abs. 1 Satz 2 GmbHG][gmbhg30]; Bonität, Sicherheit: Wertungsfrage).
  Geschäftsführer, Prokuristen und Generalbevollmächtigte erhalten keinen
  Kredit aus stammkapitalgebundenem Vermögen ([§ 43a GmbHG][gmbhg43a]).
- Zinslos oder zu niedrig verzinst: vGA-Verdacht
  ([R 8.5 Abs. 1 KStR 2022][kstr85]).
- Konto: dient das Darlehen dauernd dem Geschäftsbetrieb, 0960 Ausleihungen an
  Gesellschafter ([§ 247 Abs. 2 HGB][hgb247]); sonst 1308 (bis 1 Jahr) oder
  1309 (über 1 Jahr). Wertungsfrage.

## 10. Zinsen an und von Gesellschaftern

- Zinsaufwand: 7316 Zinsen für Gesellschafterdarlehen, bei mehr als 25 %
  Beteiligung 7317; frag nach der Quote. Zinsertrag: 7110.
- Kapitalertragsteuer: keine auf Zinsen eines gewöhnlichen Darlehens; partiarisches
  Darlehen und stille Beteiligung sind Grenze, Steuerberater
  ([darlehen-sonderfaelle.md](darlehen-sonderfaelle.md#3-kapitalertragsteuer-auf-gesellschafterzinsen)).

## 11. Insolvenz, Nachrang, Rangrücktritt

Gesellschafterdarlehen sind in der Insolvenz nachrangig, Rückzahlungen
anfechtbar. Deuten die Zahlen auf eine Krise, weise vor jeder Tilgung darauf
hin ([grundlagen/grenzen.md](../grundlagen/grenzen.md#2-warnzeichen-die-du-sofort-meldest)).
Fristen und Rangrücktritt: [darlehen-sonderfaelle.md](darlehen-sonderfaelle.md#2-insolvenz-nachrang-rangrücktritt).

## 12. Buchungen eines Gesellschafterdarlehens

| Vorgang | Soll | Haben | Werkzeug |
|---|---|---|---|
| Gesellschafter zahlt Darlehen aus | Bank | 3641 bis 3643 | `bankumsatz_buchen`, Zeile `darlehen_id` |
| Zinsen fällig, noch nicht gezahlt | 7316 oder 7317 | 3640 | `darlehen_erfassen` mit `art: zinsen`, Eigenbeleg mit der Zinsrechnung |
| Zinszahlung | 3640 | Bank | `bankumsatz_buchen`, Zeile `konto` |
| Tilgung | 3641 bis 3643 | Bank | `bankumsatz_buchen`, Zeile `darlehen_id` |
| GmbH zahlt Darlehen an Gesellschafter aus | 0960, 1308 oder 1309 | Bank | `bankumsatz_buchen`, Zeile `darlehen_id` |
| Gesellschafter tilgt | Bank | 0960, 1308 oder 1309 | `bankumsatz_buchen`, Zeile `darlehen_id` |

Das Konto des Darlehens legt `darlehen_erfassen` aus dem Vertrag fest.

[gobd]: https://www.lfst.bayern.de/fileadmin/RESSOURCEN/INFORMATIONEN/Steuerinfos/Weitere_Themen/Aussenpruefung/2019-11-28-GoBD-1.pdf
[gmbhg30]: https://www.gesetze-im-internet.de/gmbhg/__30.html
[gmbhg43a]: https://www.gesetze-im-internet.de/gmbhg/__43a.html
[hgb247]: https://www.gesetze-im-internet.de/hgb/__247.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[kstg8]: https://www.gesetze-im-internet.de/kstg_1977/__8.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[kstr85]: https://ksth.bundesfinanzministerium.de/ksth/2026/A-Koerperschaftsteuergesetz/II-Einkommen/1-Allgemeine-Vorschriften/Paragraf-8/inhalt.html
