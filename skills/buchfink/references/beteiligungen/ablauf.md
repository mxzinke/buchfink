# Ablauf: Beteiligungen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Hält die GmbH Anteile an anderen Gesellschaften, Aktien oder Fondsanteile,
laufen Handelsbilanz und Steuer auseinander. Dividenden und
Veräußerungsgewinne stehen voll in der GuV, bleiben bei der
Körperschaftsteuer aber zu 95 % außer Ansatz. Fondserträge sind teilweise
steuerfrei, und die Vorabpauschale wird versteuert, ohne dass gebucht wird.
Du buchst die Vorgänge auf die Konten, an denen Buchfink die steuerliche Folge
erkennt, hältst die Anlagenkartei vollständig und prüfst zum Jahresende die
Zeilen der Erklärungen.

Was Buchfink dafür hat:

- Anlagenkartei für Finanzanlagen: Kennung (ISIN, WKN, Registernummer),
  Beteiligungsquote (ohne Stichtag), Kennzeichen „Anteil an einer
  Kapitalgesellschaft“ mit eigenen Konten für Abgang, Abschreibung und
  Zuschreibung, Fondsart, Stückzahl, Notierungswährung. Diese Felder und die
  Bewegungen eines Anlageguts (Abgang, Wertänderung, Ertrag, Vorabpauschale,
  Währungsbewertung) bildet Buchfink noch nicht ab, weder als Werkzeug noch in
  der App; vorhandene Werte liest es. Leg dem Menschen Sachverhalt und Zahlen
  vor; ein Steuerberater kann die Werte für die Erklärung bestätigen.
- Die Rechnung zu § 8b KStG, § 8 Nr. 5 und § 9 Nr. 2a GewStG, Teilfreistellung
  und Vorabpauschale in `erklaerung` (`art: kst`, `art: gewst`), aus den
  §-8b-Konten und der Kartei. Sind Bezüge keinem Anteil mit Quote zugeordnet,
  bleiben die Zeilen zu Streubesitz und § 8 Nr. 5 offen.

| Thema | Regel in |
|---|---|
| Ausschüttung der eigenen GmbH an ihre Gesellschafter | [gesellschafter/ausschuettung.md](../gesellschafter/ausschuettung.md) |
| Zinsen der Bank mit Kapitalertragsteuer, Bankdarlehen | [bank/zinsen-kapest.md](../bank/zinsen-kapest.md), [bank/darlehen.md](../bank/darlehen.md) |
| Eingangsrechnung (Notar, Berater) kontieren, Vorsteueranteil | [belege/ablauf.md](../belege/ablauf.md), [belege/vorsteuer.md](../belege/vorsteuer.md) |
| Rechnung der Holding an eine Tochter | [rechnungen/ablauf.md](../rechnungen/ablauf.md) |
| Abschlussbuchungen, Anlagenspiegel, Größenklasse | [jahresabschluss/ablauf.md](../jahresabschluss/ablauf.md), [jahresabschluss/groessenklassen-anhang.md](../jahresabschluss/groessenklassen-anhang.md) |
| Überleitung, Erklärungen, Steuerrückstellung | [steuererklaerungen/kst.md](../steuererklaerungen/kst.md), [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md), [jahresabschluss/steuern-im-abschluss.md](../jahresabschluss/steuern-im-abschluss.md) |
| Organschaft, Konzernabschluss, Mitunternehmerschaft | [grundlagen/grenzen.md](../grundlagen/grenzen.md#1-was-buchfink-nicht-abbildet) |

Werkzeuge: `beleg_ablegen`, `bankumsatz_buchen` (Rückfrage),
`buchung_erfassen` (Rückfrage), `abschlussbuchung` (Rückfrage), `anlagen`,
`konten`, `buchungen_suchen`, `auswertung`, `belege_suchen`.

## Inhalt

1. Vorgang einordnen
2. Ablauf für jeden Vorgang
3. Steuerliche Folgen
4. Entscheidungsregeln
5. Wertungsfragen
6. Nur der Mensch
7. Stolperfallen

## 1. Vorgang einordnen

| Vorgang | Konto (SKR04) | Lies |
|---|---|---|
| Anteile an einer Tochter kaufen oder gründen (Mehrheit der Stimmrechte) | 0804 Anteile an verbundenen Unternehmen, Kapitalgesellschaften | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Beteiligung an einer GmbH oder AG (dauernde Verbindung) | 0850 Beteiligungen an Kapitalgesellschaften | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Aktien als Daueranlage ohne Beteiligungsabsicht | 0910 oder 0900 | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Aktien oder Fonds zur kurzfristigen Anlage | 1510 Sonstige Wertpapiere | [erwerb-ausweis.md](erwerb-ausweis.md) |
| ETF oder anderer Investmentfonds als Daueranlage | 0900 Wertpapiere des Anlagevermögens | [investmentfonds.md](investmentfonds.md) |
| Dividende einer Kapitalgesellschaft geht ein | 7005, 7006, 7014, 7015, 7103 oder 7104 brutto; 7630, 7633 | [dividenden.md](dividenden.md) |
| Ausschüttung eines Fonds geht ein | 7010 (Anlagevermögen) brutto; 7630, 7633 | [investmentfonds.md](investmentfonds.md) |
| Bank belastet Kapitalertragsteuer auf die Vorabpauschale | 7630, 7633 an Bank | [investmentfonds.md](investmentfonds.md) |
| Wert gesunken oder wieder gestiegen | 7204, 7200, 7201, 7214; 4912, 4913, 4914 | [bewertung-veraeusserung.md](bewertung-veraeusserung.md) |
| Anteile verkauft | 4852 und 4858 oder 6892 und 6898; Fonds 4851 und 4857 oder 6891 und 6897 | [bewertung-veraeusserung.md](bewertung-veraeusserung.md) |
| Darlehen an eine Tochter | 0814 oder 0885; Zinsen 7012 oder 7011 | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Zuzahlung in die Kapitalrücklage einer Tochter | Beteiligungskonto (nachträgliche Anschaffungskosten) | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Tochter zahlt aus dem Einlagekonto zurück | keine Einnahme; Steuerberater | [dividenden.md](dividenden.md) |
| Holding stellt einer Tochter Leistungen in Rechnung oder fragt nach Vorsteuer | – | [holding-umsatzsteuer.md](holding-umsatzsteuer.md) |

Kontonummern nach dem SKR04 in Buchfink. Welches Dividendenkonto zu welchem
Anteil gehört, steht in [dividenden.md](dividenden.md).

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
4. **Rückfrage** im Format aus
   [SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen). Nenne
   zusätzlich in einer Zeile die steuerliche Folge, z. B. „95 % steuerfrei
   nach § 8b Abs. 1, 5 KStG; keine Hinzurechnung, Quote 100 %“. Fertig, wenn
   der Mensch Ja gesagt hat.
5. **Buchen.** Ertrag eines Anteils aus der Kartei mit Bankumsatz:
   `bankumsatz_buchen` mit `kapitalertrag` (`anlage_id`, `brutto`,
   `kapitalertragsteuer`, `solidaritaetszuschlag`); Buchfink wählt das
   Ertragskonto aus dem Anlagekonto und verknüpft den Ertrag mit dem Anteil
   ([dividenden.md, Abschnitt 3](dividenden.md#3-buchung)). Ohne Anteil in der
   Kartei: `aufteilung` mit Zeilen `konto`, Ertrag brutto mit positivem
   `betrag`, Kapitalertragsteuer und Solidaritätszuschlag als Abzüge mit
   negativem `betrag` (wie bei Zinsen in
   [bank/zinsen-kapest.md](../bank/zinsen-kapest.md)). Ohne
   Bankumsatz (Forderung aus Beschluss, Umbuchung, Einlage):
   `buchung_erfassen` mit `beleg_id`. Fertig, wenn die Antwort die vorgelegten
   Zeilen zeigt.
6. **Kartei.** Buchungen auf Anlagekonten ohne Anlagegut nennt `anlagen` als
   Aktivierungskandidaten. Das Anlagegut legst du nach Rückfrage mit
   `abschlussbuchung` an (`art: anlage_aktivieren`, `klasse: finanzanlage`,
   `methode: keine`). Die Felder, die Buchfink noch nicht abbildet (oben),
   hältst du in der Beteiligungsliste fest. Fertig, wenn `anlagen` das
   Anlagegut zeigt und es zu den Salden aus `auswertung` passt.
7. **Steuerzeilen prüfen.** Zum Jahresende `erklaerung` (`art: kst` und
   `art: gewst`): Bezüge und Gewinne nach § 8b, Streubesitz, § 8 Nr. 5,
   Teilfreistellung, Vorabpauschale. Gleiche jede Zeile mit der
   Beteiligungsliste ab. Offene Zeilen (`kst.gk.8b_streubesitz`, `gewst.8_5`)
   füllst du aus der Liste, Vorschläge bestätigst du, wenn die Quote zum
   Stichtag stimmt; beides nach Rückfrage mit `erklaerung_eingabe`. Fertig,
   wenn keine dieser Zeilen offen ist und jede zur Liste passt.

## 3. Steuerliche Folgen

| Vorgang | Körperschaftsteuer | Gewerbesteuer |
|---|---|---|
| Dividende, Gewinn aus Verkauf, Verlust oder Teilwertabschreibung beim Anteil | [steuerfreistellung.md, Abschnitt 1](steuerfreistellung.md#1-übersicht) | [gewst-holding.md, Abschnitte 1 und 2](gewst-holding.md#2-dividenden-nach-quote) |
| Erträge, Verkauf und Verlust bei Investmentfonds | [investmentfonds.md, Abschnitte 3 und 4](investmentfonds.md#4-teilfreistellung) | [gewst-holding.md, Abschnitt 6](gewst-holding.md#6-investmentfonds) |

Die Grenzen (10 % für Streubesitz, 15 % für die Kürzung) und die Stichtage
der Quote stehen in [gewst-holding.md, Abschnitt 2](gewst-holding.md#2-dividenden-nach-quote).

## 4. Entscheidungsregeln

- **Zwei Stichtage.** § 8b Abs. 4 KStG und § 9 Nr. 2a GewStG messen die Quote
  an verschiedenen Tagen; ein Erwerb im Laufe des Jahres wirkt bei beiden
  Steuern verschieden
  ([gewst-holding.md, Abschnitt 2](gewst-holding.md#2-dividenden-nach-quote)).
- **Dividendenkonto nach Anteil.** Buche Dividenden auf die §-8b-Konten nach
  der Tabelle in [dividenden.md, Abschnitt 3](dividenden.md#3-buchung);
  Buchfink rechnet aus ihnen. Steht die Dividende eines Anteils aus der Kartei
  auf einem anderen Konto, meldet Buchfink `bezuege_ohne_8b_konto`.
- **Fondsanteile sind keine Kapitalgesellschaftsanteile.** Setze bei einem
  Investmentfonds nie das Kennzeichen „Anteil an einer Kapitalgesellschaft“
  und buche seine Erträge nie auf §-8b-Konten
  ([investmentfonds.md, Abschnitte 2 und 3](investmentfonds.md#3-steuerliche-grundregeln)).
- **Kapitalertragsteuer trotz Steuerfreiheit.** Abzug, Anrechnung und die
  nötige Steuerbescheinigung:
  [dividenden.md, Abschnitte 2 und 4](dividenden.md#4-steuerbescheinigung-und-anrechnung).
  Erinnere an die Bescheinigung. Kapitalertragsteuer und
  Solidaritätszuschlag stehen getrennt auf 7630 und 7633; Buchfink rechnet
  sie in der Überleitung hinzu
  ([steuerfreistellung.md, Abschnitt 8](steuerfreistellung.md#8-konten-und-ihre-wirkung)).
- **Bewertung zum Stichtag.** Prüfe jede Finanzanlage zum Jahresende nach
  [bewertung-veraeusserung.md, Abschnitt 1](bewertung-veraeusserung.md#1-bewertung-zum-stichtag-nach-hgb).
- **Rückstellungsvorschlag prüfen.** Der Vorschlag behandelt Bezüge auf 7005,
  7006, 7015 und 7104 als steuerfrei nach § 8b Abs. 1 KStG, auf 7014 und 7103
  als steuerpflichtig, und sagt das im Hinweis. Die Hinzurechnung nach § 8
  Nr. 5 GewStG nimmt er nur als Eingabe (`hinzurechnungen`); den Betrag
  liefert die Gewerbesteuererklärung
  ([steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)).

## 5. Wertungsfragen

Format: [SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen).

- Beteiligung oder Wertpapier, Anlage- oder Umlaufvermögen
  ([erwerb-ausweis.md, Abschnitt 1](erwerb-ausweis.md#1-begriffe))?
- Ist eine Wertminderung voraussichtlich dauernd
  ([bewertung-veraeusserung.md, Abschnitte 1 und 2](bewertung-veraeusserung.md#2-teilwert-nach-estg))?
- Darf eine Dividende vor dem Beschluss der Tochter vereinnahmt werden
  (phasengleich; [dividenden.md, Abschnitt 1](dividenden.md#1-anspruch-und-zeitpunkt))?
- Ist die Holding Unternehmerin, und welcher Anteil der Vorsteuer ist
  abziehbar ([holding-umsatzsteuer.md](holding-umsatzsteuer.md))?
- Ist ein Darlehen an die Tochter fremdüblich
  ([steuerfreistellung.md, Abschnitt 6](steuerfreistellung.md#6-darlehen-an-die-tochter))?
- In welchem Jahr setzt die GmbH die Vorabpauschale an
  ([investmentfonds.md, Abschnitt 5](investmentfonds.md#5-vorabpauschale))?
- Welche Fondsart nennen die Anlagebedingungen
  ([investmentfonds.md, Abschnitt 1](investmentfonds.md#1-was-ein-investmentfonds-ist))?
- Gehören Beratungskosten vor dem Kauf zu den Anschaffungskosten
  ([erwerb-ausweis.md, Abschnitt 3](erwerb-ausweis.md#3-anschaffungskosten))?

## 6. Nur der Mensch

- Als Gesellschafter der Tochter beschließen; Notarverträge schließen
  ([erwerb-ausweis.md, Abschnitt 4](erwerb-ausweis.md#4-erwerb-buchen)).
- Steuerbescheinigungen anfordern; eine Freistellung vom Steuerabzug beim
  Finanzamt beantragen ([dividenden.md, Abschnitt 5](dividenden.md#5-sonderfälle)).
- Mitteilung über ausländische Beteiligungen mit der
  Körperschaftsteuererklärung
  ([erwerb-ausweis.md, Abschnitt 7](erwerb-ausweis.md#7-pflichten-die-aus-der-beteiligung-folgen)).
- Einen Steuerberater beauftragen.

## 7. Stolperfallen

- Die Streubesitzgrenze von 10 % gilt nur für Dividenden, nicht für
  Veräußerungsgewinne
  ([steuerfreistellung.md, Abschnitt 3](steuerfreistellung.md#3-streubesitz-unter-10-)).
- Den Basiszins für die Vorabpauschale bestimmt jedes Jahr das BMF neu.
  Buchfink kennt ihn für 2024 bis 2026, rechnet die Vorabpauschale aber weder
  in der App noch über ein Werkzeug; den Betrag nennt die Bescheinigung der
  Bank ([investmentfonds.md, Abschnitt 5](investmentfonds.md#5-vorabpauschale)).
- Die Bank behält auf Fondserträge Kapitalertragsteuer nach der
  Teilfreistellung für Privatanleger ein
  ([investmentfonds.md, Abschnitt 7](investmentfonds.md#7-kapitalertragsteuer-und-bescheinigung)).
- Auf Gewinne aus dem Verkauf von Aktien und Fondsanteilen behält eine
  inländische Bank bei der GmbH keine Kapitalertragsteuer ein
  ([bewertung-veraeusserung.md, Abschnitt 7](bewertung-veraeusserung.md#7-verkauf-steuer)).
- Wann eine Beteiligung vermutet wird:
  [erwerb-ausweis.md, Abschnitt 1](erwerb-ausweis.md#1-begriffe).
- Eine reine Beteiligungsholding ist keine Kleinstkapitalgesellschaft; eine
  Führungsholding kann es sein
  ([erwerb-ausweis.md, Abschnitt 7](erwerb-ausweis.md#7-pflichten-die-aus-der-beteiligung-folgen)).
- Verlustuntergang bei der Tochter nach einem Anteilswechsel: Weise beim Kauf
  oder Verkauf darauf hin und empfiehl einen Steuerberater
  ([erwerb-ausweis.md, Abschnitt 7](erwerb-ausweis.md#7-pflichten-die-aus-der-beteiligung-folgen)).
- Die Quote der Kartei trägt keinen Stichtag; prüfe sie je Steuer an der
  Gesellschafterliste
  ([gewst-holding.md, Abschnitt 8](gewst-holding.md#8-angaben-für-die-erklärung)).
- Fondserträge in GewSt 1 A Zeile 48:
  [gewst-holding.md, Abschnitt 6](gewst-holding.md#6-investmentfonds).
