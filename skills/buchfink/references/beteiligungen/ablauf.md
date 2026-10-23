# Ablauf: Beteiligungen

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Du buchst Beteiligungsvorgänge auf die Konten, an denen Buchfink die
steuerliche Folge erkennt (§ 8b KStG, Teilfreistellung, Vorabpauschale),
hältst Anlagenkartei und Beteiligungsliste vollständig und prüfst zum
Jahresende die Erklärungszeilen.

Von den Angaben eines Anteils erfasst die Kartei nur die Fondsart. Kennung
(ISIN, WKN, Registernummer), Quote, Kennzeichen „Anteil an einer
Kapitalgesellschaft“, Stückzahl, Notierungswährung, Abgang, Wertänderung und
Währungsbewertung führst du in der Beteiligungsliste und legst sie dem
Menschen vor; vorhandene Werte liest Buchfink.

## Inhalt

1. Vorgang einordnen
2. Ablauf für jeden Vorgang
3. Steuerliche Folgen
4. Entscheidungsregeln
5. Wertungsfragen
6. Nur der Mensch

## 1. Vorgang einordnen

| Vorgang | Konto (SKR04) | Lies |
|---|---|---|
| Tochter, Beteiligung, Aktien oder Genossenschaftsanteile kaufen oder gründen | 0804, 0850, 0910, 1510, 0980 nach Art des Anteils | [erwerb-ausweis.md, Abschnitt 2](erwerb-ausweis.md#2-konto-wählen) |
| ETF oder anderer Investmentfonds | 0900, kurzfristig 1510 | [investmentfonds.md](investmentfonds.md) |
| Dividende einer Kapitalgesellschaft geht ein | 7005, 7006, 7014, 7015, 7103 oder 7104 brutto; 7630, 7633 | [dividenden.md](dividenden.md) |
| Ausschüttung eines Fonds geht ein | 7010 (Anlagevermögen) brutto; 7630, 7633 | [investmentfonds.md](investmentfonds.md) |
| Bank belastet Kapitalertragsteuer auf die Vorabpauschale | 7630, 7633 an Bank | [investmentfonds.md](investmentfonds.md) |
| Wert gesunken oder wieder gestiegen | 7204, 7200, 7201, 7214; 4912, 4913, 4914 | [bewertung-veraeusserung.md](bewertung-veraeusserung.md) |
| Anteile verkauft | 4852 und 4858 oder 6892 und 6898; Fonds 4851 und 4857 oder 6891 und 6897 | [bewertung-veraeusserung.md](bewertung-veraeusserung.md) |
| Darlehen an eine Tochter | 0814 oder 0885; Zinsen 7012 oder 7011 | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Zuzahlung in die Kapitalrücklage einer Tochter | Beteiligungskonto (nachträgliche Anschaffungskosten) | [erwerb-ausweis.md](erwerb-ausweis.md) |
| Tochter zahlt aus dem Einlagekonto zurück | keine Einnahme; Steuerberater | [dividenden.md](dividenden.md) |
| Holding stellt einer Tochter Leistungen in Rechnung oder fragt nach Vorsteuer | – | [holding-umsatzsteuer.md](holding-umsatzsteuer.md) |

## 2. Ablauf für jeden Vorgang

1. **Bestand kennen.** `auswertung` (`art: susa`) für 0800 bis 0999 und
   1500 bis 1530, `anlagen` (`klasse: finanzanlage`). Beteiligungsliste:
   Gesellschaft, Sitz (In- oder Ausland), Rechtsform, Quote zu Beginn des
   Kalenderjahres, Konto, Anschaffungskosten, Buchwert, Kennzeichen,
   Fondsart. Fertig, wenn jeder Saldo einem Eintrag entspricht.
2. **Beleg ablegen** mit `beleg_ablegen`: Notarvertrag, Gründungsurkunde,
   Depotabrechnung, Gewinnverwendungsbeschluss der Tochter,
   Dividendengutschrift, Steuerbescheinigung. Fertig, wenn eine `beleg_id`
   vorliegt oder der Mensch einen Eigenbeleg mit Grund bestätigt hat.
3. **Einordnen** nach Abschnitt 1 und 3, mit der Quote zu Beginn des
   Kalenderjahres. Fertig, wenn Konto, Betrag und steuerliche Folge
   feststehen oder eine Wertungsfrage (Abschnitt 5) benannt ist.
4. **Rückfrage** mit der steuerlichen Folge in einer Zeile, z. B. „95 %
   steuerfrei nach § 8b Abs. 1, 5 KStG; keine Hinzurechnung, Quote 100 %“.
   Fertig, wenn der Mensch Ja gesagt hat.
5. **Buchen.** Ertrag eines Anteils aus der Kartei: `bankumsatz_buchen` mit
   `kapitalertrag`. Ohne Anteil in der Kartei: `aufteilung` wie in
   [dividenden.md, Abschnitt 3](dividenden.md#3-buchung). Ohne Bankumsatz
   (Forderung aus Beschluss, Umbuchung, Einlage): `buchung_erfassen` mit
   `beleg_id`. Fertig, wenn die Antwort die vorgelegten Zeilen zeigt.
6. **Kartei.** `anlagen` nennt Buchungen auf Anlagekonten ohne Anlagegut als
   Aktivierungskandidaten; nach Rückfrage `abschlussbuchung`
   (`art: anlage_aktivieren`, bei einem Fonds mit `fondsart`). Fertig, wenn
   `anlagen` das Anlagegut zeigt und es zu den Salden passt.
7. **Jahresende.** Bewerte jede Finanzanlage
   ([bewertung-veraeusserung.md, Abschnitt 1](bewertung-veraeusserung.md#1-bewertung-zum-stichtag-nach-hgb))
   und prüfe in `erklaerung` (`art: kst`, `art: gewst`) § 8b, Streubesitz,
   § 8 Nr. 5, Teilfreistellung und Vorabpauschale gegen die
   Beteiligungsliste. Offene Zeilen (`kst.gk.8b_streubesitz`, `gewst.8_5`)
   füllst du aus der Liste, Vorschläge bestätigst du, wenn die Quote zum
   Stichtag stimmt (die Quote der Kartei hat kein Datum); beides nach
   Rückfrage mit `erklaerung_eingabe`. Fertig, wenn jede Finanzanlage
   bewertet, keine dieser Zeilen offen ist und jede zur Liste passt.

## 3. Steuerliche Folgen

| Vorgang | Körperschaftsteuer | Gewerbesteuer |
|---|---|---|
| Dividende, Gewinn aus Verkauf, Verlust oder Teilwertabschreibung beim Anteil | [steuerfreistellung.md, Abschnitt 1](steuerfreistellung.md#1-übersicht) | [gewst-holding.md, Abschnitte 1 und 2](gewst-holding.md#2-dividenden-nach-quote) |
| Erträge, Verkauf und Verlust bei Investmentfonds | [investmentfonds.md, Abschnitte 3 und 4](investmentfonds.md#4-teilfreistellung) | [gewst-holding.md, Abschnitt 6](gewst-holding.md#6-investmentfonds) |

## 4. Entscheidungsregeln

- **Zwei Stichtage:** Körperschaftsteuer 10 % mit Rückwirkung des Erwerbs,
  Gewerbesteuer 15 % ohne
  ([gewst-holding.md, Abschnitt 2](gewst-holding.md#2-dividenden-nach-quote)).
- **Fondserträge nie auf §-8b-Konten**
  ([investmentfonds.md, Abschnitte 2 und 3](investmentfonds.md#3-steuerliche-grundregeln)).
- **Kapitalertragsteuer trotz Steuerfreiheit**, getrennt auf 7630 und 7633
  ([steuerfreistellung.md, Abschnitt 8](steuerfreistellung.md#8-konten-und-ihre-wirkung));
  erinnere an die Bescheinigung
  ([dividenden.md, Abschnitte 2 und 4](dividenden.md#4-steuerbescheinigung-und-anrechnung)).
- **Steuerrückstellung:** Die Hinzurechnung nach § 8 Nr. 5 GewStG aus
  `erklaerung` (`art: gewst`) gibst du bei `abschlussbuchung`
  (`art: steuerrueckstellung`) als `hinzurechnungen` mit.

## 5. Wertungsfragen

Format: [SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen).

- Beteiligung oder Wertpapier, Anlage- oder Umlaufvermögen
  ([erwerb-ausweis.md, Abschnitt 1](erwerb-ausweis.md#1-begriffe))?
- Ist eine Wertminderung voraussichtlich dauernd
  ([bewertung-veraeusserung.md, Abschnitte 1 und 2](bewertung-veraeusserung.md#2-teilwert-nach-estg))?
- Dividende vor dem Beschluss der Tochter vereinnahmen
  ([dividenden.md, Abschnitt 1](dividenden.md#1-anspruch-und-zeitpunkt))?
- Ist die Holding Unternehmerin, welcher Anteil der Vorsteuer ist abziehbar
  ([holding-umsatzsteuer.md](holding-umsatzsteuer.md))?
- Ist ein Darlehen an die Tochter fremdüblich
  ([steuerfreistellung.md, Abschnitt 6](steuerfreistellung.md#6-darlehen-an-die-tochter))?
- In welchem Jahr gilt die Vorabpauschale
  ([investmentfonds.md, Abschnitt 5](investmentfonds.md#5-vorabpauschale))?
- Welche Fondsart nennen die Anlagebedingungen
  ([investmentfonds.md, Abschnitt 1](investmentfonds.md#1-was-ein-investmentfonds-ist))?
- Beratungskosten vor dem Kauf als Anschaffungskosten
  ([erwerb-ausweis.md, Abschnitt 3](erwerb-ausweis.md#3-anschaffungskosten))?

## 6. Nur der Mensch

- Als Gesellschafter der Tochter beschließen; Notarverträge schließen
  ([erwerb-ausweis.md, Abschnitt 4](erwerb-ausweis.md#4-erwerb-buchen)).
- Steuerbescheinigungen anfordern; Freistellung vom Steuerabzug beantragen
  ([dividenden.md, Abschnitt 5](dividenden.md#5-sonderfälle)).
- Mitteilung über ausländische Beteiligungen
  ([erwerb-ausweis.md, Abschnitt 7](erwerb-ausweis.md#7-pflichten-die-aus-der-beteiligung-folgen)).
- Einen Steuerberater beauftragen.
