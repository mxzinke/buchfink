# Steuerbescheinigung für Gesellschafter

Rechtsstand: 17. Oktober 2026. Verwaltungsauffassung: BMF-Schreiben vom
16.05.2025, IV C 1 - S 2401/00008/014/051, „Kapitalertragsteuer; Ausstellung
von Steuerbescheinigungen nach § 45a Absatz 2 und 3 EStG“. Gelesen über einen
[nichtamtlichen Spiegel][bmf2025]; das BMF-Original war nicht abrufbar.

## Pflicht

- Die GmbH als Schuldnerin der Kapitalerträge stellt dem Gesellschafter auf
  Verlangen eine Bescheinigung nach amtlichem Muster aus
  ([§ 45a Abs. 2 Satz 1 Nr. 1 EStG][estg45a]). Die Pflicht besteht auch, wenn
  keine Steuer einbehalten wurde ([BMF 16.05.2025, Einleitung][bmf2025]).
- Eine Frist nennt das Gesetz nicht; ausgestellt wird erst nach der Zahlung,
  weil der Zahlungstag zu bescheinigen ist ([BMF 16.05.2025, Rn. 47][bmf2025]).
- Sie gilt auch für eine verdeckte Gewinnausschüttung
  ([BMF 16.05.2025, Rn. 48][bmf2025]). Bei einer Einlagenrückgewähr bescheinigt
  die GmbH die Minderung des Einlagekontos nach
  [§ 27 Abs. 3 KStG][kstg27] (Name, Anschrift, Betrag, Zahlungstag).

## Form

- Eine GmbH nutzt **Muster II** ([BMF 16.05.2025, Rn. 43][bmf2025]). Inhalt,
  Aufbau und Reihenfolge des Musters sind verbindlich
  ([BMF 16.05.2025, Rn. 1][bmf2025]).
- Für Kapitalerträge, die ab dem 01.01.2026 zufließen, gelten die Muster des
  Schreibens vom 16.05.2025; für Zuflüsse davor die Muster vom 23.05.2022
  ([BMF 16.05.2025, Rn. 71][bmf2025]).
- Elektronische Übermittlung ist erlaubt; auf Anforderung auf Papier
  ([§ 45a Abs. 2 Satz 2 EStG][estg45a]). Ohne Unterschrift gültig, wenn
  maschinell erstellt und der Aussteller erkennbar ist
  ([§ 45a Abs. 2 Satz 3 EStG][estg45a]).
- Eine fehlerhafte Bescheinigung ist unverzüglich durch eine als berichtigt
  gekennzeichnete zu ersetzen; die GmbH meldet das dem Finanzamt
  ([§ 45a Abs. 5 EStG][estg45a]).

## Ablauf

1. Prüfe, dass Auszahlung und Abführung gebucht sind (3519 und 3760 auf null,
   `auswertung` mit `art: kontoblatt`).
2. Stelle je Gesellschafter zusammen: Name, Anschrift, Zahlungstag,
   Kapitalertrag brutto, Kapitalertragsteuer, Solidaritätszuschlag,
   Kirchensteuer, Finanzamt und Steuernummer der GmbH, gegebenenfalls Betrag
   aus dem Einlagekonto.
3. Buchfink erzeugt keine Steuerbescheinigung. Der Mensch überträgt die Daten
   in Muster II (Anlage zum BMF-Schreiben) oder lässt sie vom Steuerberater
   ausstellen.
4. Lege die ausgestellte Bescheinigung als Beleg ab (`beleg_ablegen`).

[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/05/2025-05-16-kapitalertragSt-steuerbescheinigung.pdf
