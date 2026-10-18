---
name: buchfink-beteiligungen
description: "GmbH oder UG als Anteilseignerin in Buchfink: Beteiligungen, Tochtergesellschaften, Aktien und Investmentfonds erwerben, bewerten und veräußern, Erträge daraus buchen und die außerbilanziellen Korrekturen für Körperschaft- und Gewerbesteuer vormerken; dazu die Umsatzsteuer der Holding. Lade den Skill, wenn die GmbH Anteile kauft, gründet oder verkauft; wenn eine Dividende oder Ausschüttung einer Tochter oder aus einem Depot eingeht; bei Steuerbescheinigung, Kapitalertragsteuer und Solidaritätszuschlag auf Dividenden; bei § 8b KStG, Schachtelprivileg, Streubesitz, Hinzurechnung nach § 8 Nr. 5 oder Kürzung nach § 9 Nr. 2a GewStG; bei Teilwertabschreibung oder Zuschreibung auf Anteile; bei einem Darlehen an eine Tochter; bei ETF, Investmentfonds, Vorabpauschale oder Teilfreistellung; bei Einlagenrückgewähr; wenn eine Holding fragt, ob sie Vorsteuer abziehen darf. Ausschüttungen der eigenen GmbH regelt buchfink-gesellschafter, die Erklärungen buchfink-steuererklaerungen."
license: EUPL-1.2
compatibility: "Benötigt die laufende Buchfink-App mit eingeschaltetem MCP-Server und einem für den Client zugelassenen Mandanten. Setzt den Skill buchfink-grundlagen voraus."
metadata:
  buchfink-rechtsstand: "2026-10-17"
  buchfink-werkzeugstand: "Buchfink 0.3.0"
---

# Buchfink: Beteiligungen

Setzt buchfink-grundlagen voraus; lade ihn, falls noch nicht geschehen.

Hält die GmbH Anteile an anderen Gesellschaften, Aktien oder Fondsanteile,
laufen Handelsbilanz und Steuer auseinander. Dividenden und
Veräußerungsgewinne stehen voll in der GuV, bleiben bei der
Körperschaftsteuer aber zu 95 % außer Ansatz. Fondserträge sind teilweise
steuerfrei, und die Vorabpauschale wird versteuert, ohne dass gebucht wird.
Du buchst die Vorgänge und führst die Liste **Beteiligungsposten**: jede
außerbilanzielle Korrektur mit Konto, Betrag und Fundstelle. Sie geht an
buchfink-steuererklaerungen.

Was Buchfink dafür hat:

- Anlagenkartei für Finanzanlagen: Kennung (ISIN, WKN, Registernummer),
  Beteiligungsquote, Kennzeichen „Anteil an einer Kapitalgesellschaft“ mit
  eigenen Konten für Abgang, Abschreibung und Zuschreibung, Fondsart,
  Stückzahl, Notierungswährung. Diese Felder pflegt nur der Mensch in der App.
- Je Fondsanteil die Vorabpauschale und die Teilfreistellung als Nebenrechnung
  ohne Buchung.
- Keine Rechnung zu § 8b KStG, § 8 Nr. 5 und § 9 Nr. 2a GewStG. Der Vorschlag
  zur Steuerrückstellung setzt Beteiligungserträge voll an.

| Thema | Skill |
|---|---|
| Ausschüttung der eigenen GmbH an ihre Gesellschafter | buchfink-gesellschafter |
| Zinsen der Bank mit Kapitalertragsteuer, Bankdarlehen | buchfink-bank |
| Eingangsrechnung (Notar, Berater) kontieren, Vorsteueranteil | buchfink-belege |
| Rechnung der Holding an eine Tochter | buchfink-rechnungen |
| Abschlussbuchungen, Anlagenspiegel, Größenklasse | buchfink-jahresabschluss |
| Überleitung, Erklärungen, Steuerrückstellung | buchfink-steuererklaerungen |
| Organschaft, Konzernabschluss, Mitunternehmerschaft | buchfink-grundlagen (Grenzen) |

Werkzeuge: `beleg_ablegen`, `bankumsatz_buchen` (Rückfrage),
`buchung_erfassen` (Rückfrage), `abschlussbuchung` (Rückfrage), `anlagen`,
`konten`, `buchungen_suchen`, `auswertung`, `belege_suchen`.

## 1. Vorgang einordnen

| Vorgang | Konto (SKR04) | Lies |
|---|---|---|
| Anteile an einer Tochter kaufen oder gründen (Mehrheit der Stimmrechte) | 0804 Anteile an verbundenen Unternehmen, Kapitalgesellschaften | [erwerb-ausweis.md](references/erwerb-ausweis.md) |
| Beteiligung an einer GmbH oder AG (dauernde Verbindung) | 0850 Beteiligungen an Kapitalgesellschaften | [erwerb-ausweis.md](references/erwerb-ausweis.md) |
| Aktien als Daueranlage ohne Beteiligungsabsicht | 0910 oder 0900 | [erwerb-ausweis.md](references/erwerb-ausweis.md) |
| Aktien oder Fonds zur kurzfristigen Anlage | 1510 Sonstige Wertpapiere | [erwerb-ausweis.md](references/erwerb-ausweis.md) |
| ETF oder anderer Investmentfonds als Daueranlage | 0900 Wertpapiere des Anlagevermögens | [investmentfonds.md](references/investmentfonds.md) |
| Dividende einer Kapitalgesellschaft geht ein | 7005, 7006, 7014, 7015, 7103 oder 7104 brutto; 7630, 7633 | [dividenden.md](references/dividenden.md) |
| Ausschüttung eines Fonds geht ein | 7010 (Anlagevermögen) brutto; 7630, 7633 | [investmentfonds.md](references/investmentfonds.md) |
| Bank belastet Kapitalertragsteuer auf die Vorabpauschale | 7630, 7633 an Bank | [investmentfonds.md](references/investmentfonds.md) |
| Wert gesunken oder wieder gestiegen | 7204, 7200, 7201, 7214; 4912, 4913, 4914 | [bewertung-veraeusserung.md](references/bewertung-veraeusserung.md) |
| Anteile verkauft | 4852 und 4858 oder 6892 und 6898; Fonds 4851 und 4857 oder 6891 und 6897 | [bewertung-veraeusserung.md](references/bewertung-veraeusserung.md) |
| Darlehen an eine Tochter | 0814 oder 0885; Zinsen 7012 oder 7011 | [erwerb-ausweis.md](references/erwerb-ausweis.md) |
| Zuzahlung in die Kapitalrücklage einer Tochter | Beteiligungskonto (nachträgliche Anschaffungskosten) | [erwerb-ausweis.md](references/erwerb-ausweis.md) |
| Tochter zahlt aus dem Einlagekonto zurück | keine Einnahme; Steuerberater | [dividenden.md](references/dividenden.md) |
| Holding stellt einer Tochter Leistungen in Rechnung oder fragt nach Vorsteuer | – | [holding-umsatzsteuer.md](references/holding-umsatzsteuer.md) |

Kontonummern stammen aus `internal/accounting/skr04_2026.json`. Welches
Dividendenkonto zu welchem Anteil gehört, steht in
[dividenden.md](references/dividenden.md).

## 2. Ablauf für jeden Vorgang

1. **Bestand kennen.** `auswertung` (`art: susa`) für 0800 bis 0999 und
   1500 bis 1530, `anlagen` (`klasse: finanzanlage`). Führe eine
   Beteiligungsliste: Gesellschaft, Sitz (Inland oder Ausland), Rechtsform,
   Quote zu Beginn des Kalenderjahres, Konto, Anschaffungskosten, Buchwert,
   Kennzeichen, Fondsart. Fertig, wenn jeder Saldo einem Eintrag der Liste
   entspricht.
2. **Beleg ablegen.** Notarvertrag, Gründungsurkunde, Depotabrechnung,
   Gewinnverwendungsbeschluss der Tochter, Dividendengutschrift,
   Steuerbescheinigung mit `beleg_ablegen`. Fertig, wenn eine `beleg_id`
   vorliegt oder der Mensch einen Eigenbeleg mit Grund bestätigt hat.
3. **Einordnen** nach Abschnitt 1 und 3. Bestimme die Quote zu Beginn des
   Kalenderjahres. Fertig, wenn Konto, Betrag und steuerliche Folge feststehen
   oder eine Wertungsfrage (Abschnitt 5) benannt ist.
4. **Rückfrage** im Format aus buchfink-grundlagen. Nenne zusätzlich in einer
   Zeile die steuerliche Folge, z. B. „95 % steuerfrei nach § 8b Abs. 1, 5
   KStG; keine Hinzurechnung, Quote 100 %“. Fertig, wenn der Mensch Ja gesagt
   hat.
5. **Buchen.** Mit Bankumsatz: `bankumsatz_buchen`, Zeilen `konto`: Ertrag
   brutto mit positivem `betrag`, Kapitalertragsteuer und
   Solidaritätszuschlag als Abzüge mit negativem `betrag` (wie bei Zinsen in
   buchfink-bank). Ohne
   Bankumsatz (Forderung aus Beschluss, Umbuchung, Einlage):
   `buchung_erfassen` mit `beleg_id`. Fertig, wenn die Antwort die vorgelegten
   Zeilen zeigt.
6. **Kartei.** Buchungen auf Anlagekonten ohne Anlagegut nennt `anlagen` als
   Aktivierungskandidaten. Das Anlagegut legst du nach Rückfrage mit
   `abschlussbuchung` an (`art: anlage_aktivieren`, `klasse: finanzanlage`,
   `methode: keine`), oder der Mensch legt es in der App an. Kennung, Quote,
   Kennzeichen, Fondsart und Stückzahl setzt nur der Mensch in der
   Anlagenkartei; ebenso Abgang, außerplanmäßige Abschreibung, Zuschreibung
   und Vorabpauschale. Fertig, wenn `anlagen` das Anlagegut zeigt und es zu
   den Salden aus `auswertung` passt.
7. **Beteiligungsposten vormerken.** Je Vorgang: Datum, Konto, Betrag,
   Korrektur Körperschaftsteuer, Korrektur Gewerbesteuer, Fundstelle,
   `beleg_id`. Fertig, wenn der Posten in der Liste steht und der Mensch sie
   kennt.

## 3. Steuerliche Folgen auf einen Blick

| Vorgang | Körperschaftsteuer | Gewerbesteuer |
|---|---|---|
| Dividende, Quote zu Beginn des Kalenderjahres unter 10 % | voll steuerpflichtig, kein 5-%-Zuschlag ([§ 8b Abs. 4 Satz 1, 7 KStG][kstg8b]) | voll steuerpflichtig |
| Dividende, Quote mindestens 10 %, zu Beginn des Erhebungszeitraums unter 15 % | außer Ansatz, 5 % nicht abziehbar ([§ 8b Abs. 1, 5 KStG][kstg8b]) | Hinzurechnung der freien 95 % ([§ 8 Nr. 5 GewStG][gewstg8]) |
| Dividende einer inländischen Kapitalgesellschaft, Quote zu Beginn des Erhebungszeitraums mindestens 15 % | außer Ansatz, 5 % nicht abziehbar | keine weitere Korrektur; die 5 % bleiben ([§ 9 Nr. 2a Satz 4 GewStG][gewstg9]) |
| Gewinn aus dem Verkauf eines Anteils, jede Quote | außer Ansatz, 5 % des Gewinns nicht abziehbar ([§ 8b Abs. 2, 3 Satz 1 KStG][kstg8b]) | folgt der Körperschaftsteuer ([§ 7 Satz 1 GewStG][gewstg7]) |
| Verlust, Teilwertabschreibung auf einen Anteil | nicht abziehbar ([§ 8b Abs. 3 Satz 3 KStG][kstg8b]) | folgt der Körperschaftsteuer |
| Erträge aus Investmentfonds | kein § 8b ([§ 16 Abs. 3 InvStG][invstg16]); Teilfreistellung, für die GmbH bei Aktienfonds 80 % ([§ 20 Abs. 1 Satz 3 InvStG][invstg20]) | Teilfreistellung nur zur Hälfte ([§ 20 Abs. 5 InvStG][invstg20]) |

Stand: Oktober 2026. Die Grenzen und Sätze ändern sich nur durch Gesetz.
Berechnung, Beispiele und Ausnahmen:
[steuerfreistellung.md](references/steuerfreistellung.md),
[gewst-holding.md](references/gewst-holding.md),
[investmentfonds.md](references/investmentfonds.md).

## 4. Entscheidungsregeln

- **Zwei Stichtage.** Für § 8b Abs. 4 KStG zählt die unmittelbare Quote zu
  Beginn des Kalenderjahres; ein Erwerb von mindestens 10 % gilt als zu
  Beginn erfolgt ([§ 8b Abs. 4 Satz 1, 6 KStG][kstg8b]). Für § 9 Nr. 2a GewStG
  zählt die Quote zu Beginn des Erhebungszeitraums ohne diese Rückbeziehung.
  Wer im Juni 100 % kauft, bekommt die Dividende desselben Jahres bei der
  Körperschaftsteuer zu 95 % frei und muss sie bei der Gewerbesteuer
  hinzurechnen.
- **Dividendenkonto nach Anteil.** Buche Dividenden auf die
  §-8b-Konten (7005, 7006, 7014, 7015, 7103, 7104). Das SKR04 kennzeichnet sie
  für Körperschaft- und Gewerbesteuer; die Überleitung findet sie dort.
- **Fondsanteile sind keine Kapitalgesellschaftsanteile.** Setze bei einem
  Investmentfonds nie das Kennzeichen „Anteil an einer Kapitalgesellschaft“
  und buche seine Erträge nie auf §-8b-Konten ([§ 16 Abs. 3 InvStG][invstg16]).
- **Kapitalertragsteuer trotz Steuerfreiheit.** Die Tochter oder die Bank
  behält 25 % Kapitalertragsteuer und 5,5 % Solidaritätszuschlag darauf ein,
  ungeachtet § 8b KStG ([§ 43 Abs. 1 Satz 3][estg43],
  [§ 43a Abs. 1 Satz 1 Nr. 1 EStG][estg43a], [§ 3 Abs. 1 Nr. 5][solzg3],
  [§ 4 Satz 1 SolZG][solzg4]). Stand: Oktober 2026. Sie wird angerechnet, auch
  soweit die Bezüge außer Ansatz bleiben, aber nur mit Steuerbescheinigung
  ([§ 36 Abs. 2 Nr. 2 Buchst. b, Satz 2 EStG][estg36] über
  [§ 31 Abs. 1 KStG][kstg31]). Erinnere an die Bescheinigung.
- **Getrennte Steuerkonten.** Kapitalertragsteuer auf 7630, Solidaritätszuschlag
  auf 7633. Beide mindern den Jahresüberschuss, sind aber nicht abziehbar
  ([§ 10 Nr. 2 KStG][kstg10]); vermerke sie als Hinzurechnung.
- **Bewertung zum Stichtag.** Prüfe jede Finanzanlage zum Jahresende auf
  Wertminderung und frühere Abschreibungen auf Wertaufholung
  ([§ 253 Abs. 3 Satz 5, 6, Abs. 5 HGB][hgb253]).
- **Rückstellungsvorschlag prüfen.** Bei steuerfreien Beteiligungserträgen ist
  der Vorschlag von Buchfink zu hoch, bei nicht abziehbaren Verlusten und
  Abschreibungen zu niedrig. Übergib die Beteiligungsposten an
  buchfink-steuererklaerungen, bevor der Mensch die Rückstellung bestätigt.

## 5. Wertungsfragen

Leite jede mit „Wertungsfrage:“ ein, nenne Grundlage, Optionen und Folgen.

- Beteiligung oder Wertpapier, Anlage- oder Umlaufvermögen: Soll der Anteil
  dauernd dem Geschäftsbetrieb dienen
  ([§ 247 Abs. 2][hgb247], [§ 271 Abs. 1 HGB][hgb271])?
- Ist eine Wertminderung voraussichtlich dauernd
  ([§ 253 Abs. 3 Satz 5, 6 HGB][hgb253], [§ 6 Abs. 1 Nr. 2 Satz 2 EStG][estg6])?
- Darf eine Dividende vor dem Beschluss der Tochter vereinnahmt werden
  (phasengleich)? Steuerlich nach dem BFH nur in seltenen Ausnahmen
  ([dividenden.md](references/dividenden.md)).
- Ist die Holding Unternehmerin, und welcher Anteil der Vorsteuer ist
  abziehbar ([§ 2 Abs. 1][ustg2], [§ 15 Abs. 4 UStG][ustg15])?
- Ist ein Darlehen an die Tochter fremdüblich
  ([§ 8b Abs. 3 Satz 7 KStG][kstg8b])?
- In welchem Jahr setzt die GmbH die Vorabpauschale an
  ([investmentfonds.md](references/investmentfonds.md))?
- Welche Fondsart nennen die Anlagebedingungen ([§ 2 Abs. 6, 7, 9 InvStG][invstg2])?
- Gehören Beratungskosten vor dem Kauf zu den Anschaffungskosten
  ([§ 255 Abs. 1 HGB][hgb255])?

## 6. Nur der Mensch

- Anlagegut anlegen und ändern: Kennzeichen, Quote, Fondsart, Stückzahl.
- Vorabpauschale berechnen und festhalten, Abgang, außerplanmäßige
  Abschreibung, Zuschreibung, Währungsbewertung in der Anlagenkartei.
- Als Gesellschafter der Tochter beschließen; Notarverträge schließen
  ([§ 15 Abs. 3, 4 GmbHG][gmbhg15]).
- Steuerbescheinigungen anfordern; eine Freistellung vom Steuerabzug beim
  Finanzamt beantragen ([§ 44a Abs. 5 EStG][estg44a]).
- Mitteilung über ausländische Beteiligungen mit der
  Körperschaftsteuererklärung ([§ 138 Abs. 2 Satz 1 Nr. 3, Abs. 5 AO][ao138]).
- Einen Steuerberater beauftragen.

## 7. Stolperfallen

- Die Streubesitzgrenze von 10 % gilt nur für Dividenden, nicht für
  Veräußerungsgewinne ([§ 8b Abs. 4 Satz 1 KStG][kstg8b] nennt nur „Bezüge“).
- „Ertrag buchen“ in der Anlagenkartei der App wählt 7000, 7010 oder 7011 und
  bucht Kapitalertragsteuer und Solidaritätszuschlag zusammen auf 7630. Für
  Überleitung und Anrechnung brauchst du die Aufteilung; nimm den Weg über
  das Werkzeug oder trenne die Beträge in der Liste.
- Buchfink nennt als Zuflusstag der Vorabpauschale 2026 den 02.01.2027, das
  BMF den 04.01.2027 ([investmentfonds.md](references/investmentfonds.md)).
- Den Basiszins liefert Buchfink nicht mit: 2,53 % für 2025, 3,20 % für 2026
  (BMF vom 10.01.2025 und 13.01.2026). Stand: Oktober 2026; jedes Jahr neu.
- Die Bank behält auf Fondserträge Kapitalertragsteuer nach der
  Teilfreistellung für Privatanleger ein; die 80 % für die GmbH wirken erst in
  der Veranlagung ([§ 43a Abs. 2 Satz 1 EStG][estg43a]).
- Auf Gewinne aus dem Verkauf von Aktien und Fondsanteilen behält eine
  inländische Bank bei der GmbH keine Kapitalertragsteuer ein
  ([§ 43 Abs. 2 Satz 3 Nr. 1 EStG][estg43]).
- Eine Beteiligung wird erst vermutet, wenn die Anteile ein Fünftel des
  Nennkapitals überschreiten ([§ 271 Abs. 1 Satz 3 HGB][hgb271]); der
  Hinweistext der App sagt „ab einem Fünftel“.
- Eine Holding, deren einziger Zweck Erwerb, Verwaltung und Verwertung von
  Beteiligungen ist, ohne in die Verwaltung der Töchter einzugreifen, ist keine
  Kleinstkapitalgesellschaft ([§ 267a Abs. 3 Nr. 3 HGB][hgb267a]). Eine
  Führungsholding fällt nicht darunter; buchfink-jahresabschluss.
- Wechseln bei einer Tochter innerhalb von fünf Jahren mehr als 50 % der
  Anteile, gehen ihre nicht genutzten Verluste unter
  ([§ 8c Abs. 1 Satz 1 KStG][kstg8c]). Beim Kauf oder Verkauf Steuerberater.
- Hinzurechnung nach § 8 Nr. 5 und Kürzung nach § 9 Nr. 2a GewStG kennt die
  Schätzung von Buchfink nicht. Übergib die Beträge je Beteiligung an
  buchfink-steuererklaerungen; der Mensch trägt sie in Mein ELSTER ein
  ([gewst-holding.md](references/gewst-holding.md)).
- Fondserträge: In GewSt 1 A Zeile 48 steht die volle Teilfreistellung; die
  Hälfte nach § 20 Abs. 5 InvStG rechnet das Finanzamt
  ([gewst-holding.md](references/gewst-holding.md)).

## Referenzen

- [references/erwerb-ausweis.md](references/erwerb-ausweis.md): Lies beim
  Kauf, bei der Gründung einer Tochter, bei Einlagen in eine Tochter, bei
  einem Darlehen an sie und wenn Konto oder Bilanzposten unklar sind.
- [references/dividenden.md](references/dividenden.md): Lies, wenn eine
  Dividende beschlossen wird oder eingeht, eine Steuerbescheinigung fehlt
  oder eine Einlagenrückgewähr bescheinigt wird.
- [references/steuerfreistellung.md](references/steuerfreistellung.md): Lies,
  wenn du Beteiligungsposten für die Körperschaftsteuer berechnest.
- [references/gewst-holding.md](references/gewst-holding.md): Lies, wenn du
  Hinzurechnung oder Kürzung für Beteiligungserträge bestimmst.
- [references/bewertung-veraeusserung.md](references/bewertung-veraeusserung.md):
  Lies zum Jahresende, bei Kursverlust, Wertaufholung und Verkauf.
- [references/investmentfonds.md](references/investmentfonds.md): Lies bei
  ETF, Fondsausschüttung, Vorabpauschale und Fondsverkauf.
- [references/holding-umsatzsteuer.md](references/holding-umsatzsteuer.md):
  Lies, wenn die GmbH vor allem Beteiligungen hält, Leistungen an Töchter
  abrechnet oder Vorsteuer aus Kosten der Beteiligung abziehen will.

[ao138]: https://www.gesetze-im-internet.de/ao_1977/__138.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[estg43]: https://www.gesetze-im-internet.de/estg/__43.html
[estg43a]: https://www.gesetze-im-internet.de/estg/__43a.html
[estg44a]: https://www.gesetze-im-internet.de/estg/__44a.html
[gewstg7]: https://www.gesetze-im-internet.de/gewstg/__7.html
[gewstg8]: https://www.gesetze-im-internet.de/gewstg/__8.html
[gewstg9]: https://www.gesetze-im-internet.de/gewstg/__9.html
[gmbhg15]: https://www.gesetze-im-internet.de/gmbhg/__15.html
[hgb247]: https://www.gesetze-im-internet.de/hgb/__247.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[hgb267a]: https://www.gesetze-im-internet.de/hgb/__267a.html
[hgb271]: https://www.gesetze-im-internet.de/hgb/__271.html
[invstg2]: https://www.gesetze-im-internet.de/invstg_2018/__2.html
[invstg16]: https://www.gesetze-im-internet.de/invstg_2018/__16.html
[invstg20]: https://www.gesetze-im-internet.de/invstg_2018/__20.html
[kstg8b]: https://www.gesetze-im-internet.de/kstg_1977/__8b.html
[kstg8c]: https://www.gesetze-im-internet.de/kstg_1977/__8c.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
[kstg31]: https://www.gesetze-im-internet.de/kstg_1977/__31.html
[solzg3]: https://www.gesetze-im-internet.de/solzg_1995/__3.html
[solzg4]: https://www.gesetze-im-internet.de/solzg_1995/__4.html
[ustg2]: https://www.gesetze-im-internet.de/ustg_1980/__2.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
