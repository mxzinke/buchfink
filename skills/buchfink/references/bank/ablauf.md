# Ablauf: Bank

Rechtsstand: 17. Oktober 2026. Werkzeugstand: Buchfink 0.3.0.

Jeder Bankumsatz endet zugeordnet (offener Posten ausgeglichen), gebucht
(gegen ein Konto) oder ignoriert mit Grund. Ein Umsatz mit unklarem Zweck
bleibt offen, bis der Mensch ihn klärt.

## Inhalt

1. Ablauf
2. Welcher Umsatz wohin
3. Regeln, die immer gelten
4. Was du den Menschen fragst

## 1. Ablauf

1. **Importieren.** Auf Anweisung des Menschen jede CAMT-Datei aus `eingang`
   mit `kontoauszug_importieren`. Den ersten Auszug eines Kontos liest der
   Mensch in der App ein (Übersicht, Eingang, „Einlesen“ oder Bücher › Bank).
   Ein unbekanntes Konto schlägst du mit `stammdaten_vorschlagen`
   (`bankkonten`) vor; eine Ablehnung des Imports gibst du weiter. Fertig, wenn
   `neu` zu den Erwartungen passt und du jeden ungeprüften Auszug mit dem Grund
   aus `hinweise` genannt hast.
2. **Sichten.** `bankumsaetze`. Fertig, wenn jeder Umsatz einer Zeile in
   Abschnitt 2 zugeordnet oder als unklar notiert ist.
3. **Belege zuerst.** Fehlt zu einer Zahlung die gebuchte Rechnung, buche erst
   den Beleg nach [belege/ablauf.md](../belege/ablauf.md) mit Bezahlung
   `offen`; fehlt der Beleg ganz, frag danach.
4. **Routine zuordnen.** Vorschläge mit `einstufung` `direkt` ordnest du mit
   `bankumsatz_zuordnen` zu; bei `freigabe_noetig` kommt der Umsatz in die
   Rückfragen. Fertig, wenn jeder Routineumsatz zugeordnet ist.
5. **Rückfragen.** Lege die übrigen Umsätze mit vorgeschlagener Aufteilung
   gesammelt vor ([SKILL.md, Abschnitt 3](../../SKILL.md#3-den-menschen-fragen))
   und buche nach dem Ja mit `bankumsatz_buchen`
   ([zuordnung.md](zuordnung.md#3-aufteilung-schreiben)). Fertig, wenn jeder
   Umsatz zugeordnet, gebucht, ignoriert oder als offene Frage beim Menschen
   ist.
6. **Abstimmen.** Je Bankkonto: Saldo laut `auswertung` (`art: kontoblatt`,
   `konto: <Bankkonto>`, `bis: <Ende des Auszugs>`) plus offene Umsätze bis
   dahin gleich Endsaldo des Auszugs; Geldtransit 1460 auf null oder ein
   erklärter Übertrag
   ([kreditkarte-geldtransit.md](kreditkarte-geldtransit.md#7-übertrag-zwischen-eigenen-konten)).
   Fertig, wenn jede Differenz null oder erklärt ist und der Mensch das
   Ergebnis kennt. Erst dann ist der Monat bereit für den Prüflauf
   ([monatsabschluss/ablauf.md](../monatsabschluss/ablauf.md)).

## 2. Welcher Umsatz wohin

| Umsatz | Werkzeug und Aufteilung | Mehr in |
|---|---|---|
| Vorschlag mit `einstufung` `direkt` | `bankumsatz_zuordnen` | [zuordnung.md](zuordnung.md#2-routine) |
| Vorschlag `art: regel` (Gebühren, Übertrag) | `konto` und `betrag` aus dem Vorschlag | [zuordnung.md](zuordnung.md#2-routine) |
| Lastschrift der Kreditkartenabrechnung | `kreditkartenabrechnung` | [kreditkarte-geldtransit.md](kreditkarte-geldtransit.md#3-monatsabrechnung-ausgleichen) |
| Zahlung mit Skonto | `buchung_id`, `ausgleich`, `differenz {art: skonto}` | [skonto-differenzen.md](skonto-differenzen.md#1-skonto) |
| Teilzahlung, Sammelüberweisung, Überzahlung | je Rechnung `buchung_id` mit dem gezahlten Betrag als `ausgleich` | [zuordnung.md](zuordnung.md#4-teilzahlung-sammelzahlung-überzahlung) |
| Bankgebühr zur Zahlung belastet oder abgezogen | `buchung_id`, voller Rechnungsbetrag als `ausgleich`, `differenz {art: gebuehr}` | [skonto-differenzen.md](skonto-differenzen.md#2-gebühr) |
| Rechnung in Fremdwährung bezahlt | `buchung_id`, `differenz {art: waehrung}` | [skonto-differenzen.md](skonto-differenzen.md#4-kursdifferenz) |
| Rücklastschrift | Personenkonto des Kunden, Gebühr als eigene Zeile | [zuordnung.md](zuordnung.md#6-rücklastschrift-und-rücküberweisung) |
| Kontoführung, Überweisungsentgelt | `konto: 6855` | [skonto-differenzen.md](skonto-differenzen.md#2-gebühr) |
| Zinsgutschrift | Zinsertrag brutto (7110), Kapitalertragsteuer (7630), Solidaritätszuschlag (7633) | [zinsen-kapest.md](zinsen-kapest.md#3-buchung) |
| Dividende oder Ausschüttung einer Finanzanlage | `kapitalertrag` | [beteiligungen/dividenden.md](../beteiligungen/dividenden.md) |
| Bankdarlehen: Auszahlung, Rate | Darlehenskonto (3150 bis 3170), Zinsen (7310, 7320) | [darlehen.md](darlehen.md#5-bankdarlehen-auszahlung-und-rate) |
| Gesellschafterdarlehen: Auszahlung, Tilgung | `darlehen_id` aus `darlehen_erfassen` | [darlehen.md](darlehen.md#12-buchungen-eines-gesellschafterdarlehens) |
| Sonstige Zahlung an oder von Gesellschaftern, Ausschüttung | nach Art des Vorgangs | [gesellschafter/ablauf.md](../gesellschafter/ablauf.md) |
| Löhne, Lohnsteuer, Sozialversicherung | Konten aus dem Lohnjournal | [lohn/lohnjournal.md](../lohn/lohnjournal.md) |
| Zahlung an oder vom Finanzamt oder der Gemeinde | Konto nach Steuerart | [steuerzahlungen.md](steuerzahlungen.md#2-konten-je-steuerart) |
| Übertrag zwischen eigenen Konten | Geldtransit 1460 auf beiden Seiten | [kreditkarte-geldtransit.md](kreditkarte-geldtransit.md#7-übertrag-zwischen-eigenen-konten) |
| Zweck unklar | offen lassen, den Menschen fragen | [zuordnung.md](zuordnung.md#7-umsatz-ohne-beleg-oder-verwendungszweck) |

## 3. Regeln, die immer gelten

- **Beleg.** Für Gebühren, Zinsen und Überträge genügt der Kontoauszug
  ([zuordnung.md](zuordnung.md#7-umsatz-ohne-beleg-oder-verwendungszweck)),
  für Steuern Anmeldung oder Bescheid
  ([steuerzahlungen.md](steuerzahlungen.md#1-erkennen-und-belegen)), für
  Darlehen Vertrag und Tilgungsplan
  ([darlehen.md](darlehen.md#1-vertrag-und-belege)).
- **Betrag.** Der Betrag kommt aus dem Auszug und wird nie passend gemacht;
  jede Differenz bekommt eine eigene, benannte Zeile
  ([grundlagen/gobd.md](../grundlagen/gobd.md#4-richtigkeit)).
- **Getrennte Zeilen.** Tilgung, Zinsen, Gebühren, Steuern und steuerliche
  Nebenleistungen je auf ihr Konto
  ([darlehen.md](darlehen.md#3-zinsen-umsatzsteuer-gewerbesteuer-stichtag),
  [steuerzahlungen.md](steuerzahlungen.md#4-säumniszuschläge-zinsen-verspätungszuschläge)).
- **Kapitalertragsteuer:** Erinnere den Menschen an die Steuerbescheinigung
  ([zinsen-kapest.md](zinsen-kapest.md#4-anrechnung-und-steuerbescheinigung)).
- **Ignorieren** nur, wenn der Vorgang schon vollständig gebucht ist
  ([zuordnung.md](zuordnung.md#8-ignorieren)).

## 4. Was du den Menschen fragst

Zweck eines Umsatzes ohne Beleg oder Kontakt und ob er einem Gesellschafter
zuzurechnen ist; Art einer Differenz (Skonto, Gebühr, Rundung, Kürzung,
Teilzahlung); Erstattung oder Verrechnung einer Über- oder Doppelzahlung.
Wertungsfragen: kleine Differenzen ausbuchen, Zuordnung ohne
Verwendungszweck, Umsatz ohne Beleg, Disagio, Erstattungszinsen.
