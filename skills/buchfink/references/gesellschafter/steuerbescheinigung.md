# Steuerbescheinigung für Gesellschafter

Rechtsstand: 17. Oktober 2026. Verwaltungsauffassung:
[BMF-Schreiben vom 16.05.2025][bmf2025], IV C 1 - S 2401/00008/014/051.

## Inhalt

1. Pflicht
2. Form
3. Ablauf

## 1. Pflicht

- Die GmbH stellt dem Gesellschafter auf Verlangen eine Bescheinigung nach
  amtlichem Muster aus ([§ 45a Abs. 2 Satz 1 Nr. 1 EStG][estg45a]), auch ohne
  einbehaltene Steuer ([BMF 16.05.2025, Einleitung][bmf2025]) und für eine
  vGA ([BMF 16.05.2025, Rn. 48][bmf2025]).
- Ausgestellt wird erst nach der Zahlung, weil der Zahlungstag zu
  bescheinigen ist ([BMF 16.05.2025, Rn. 47][bmf2025]).
- Bei einer Einlagenrückgewähr bescheinigt die GmbH die Minderung des
  Einlagekontos mit Name, Anschrift, Betrag und Zahlungstag
  ([§ 27 Abs. 3 KStG][kstg27]).

## 2. Form

- Eine GmbH nutzt **Muster II**, verbindlich in Inhalt, Aufbau und
  Reihenfolge ([BMF 16.05.2025, Rn. 1, 43][bmf2025]); für Zuflüsse ab
  01.01.2026 die Muster vom 16.05.2025, davor die vom 23.05.2022
  ([BMF 16.05.2025, Rn. 71][bmf2025]).
- Elektronisch erlaubt, auf Anforderung auf Papier
  ([§ 45a Abs. 2 Satz 2 EStG][estg45a]); ohne Unterschrift gültig, wenn
  maschinell erstellt und der Aussteller erkennbar ist
  ([§ 45a Abs. 2 Satz 3 EStG][estg45a]).
- Eine fehlerhafte Bescheinigung ersetzt die GmbH unverzüglich durch eine als
  berichtigt gekennzeichnete und meldet das dem Finanzamt
  ([§ 45a Abs. 5 EStG][estg45a]).

## 3. Ablauf

1. Prüfe, dass Auszahlung und Abführung gebucht sind (3519 und 3760 auf null,
   `auswertung` mit `art: kontoblatt`).
2. Stelle je Gesellschafter zusammen: Name, Anschrift, Zahlungstag,
   Kapitalertrag brutto, Kapitalertragsteuer, Solidaritätszuschlag,
   Kirchensteuer, Finanzamt und Steuernummer der GmbH, gegebenenfalls Betrag
   aus dem Einlagekonto.
3. Buchfink erzeugt keine Steuerbescheinigung. Der Mensch überträgt die Daten
   in Muster II oder lässt sie vom Steuerberater ausstellen.
4. Lege die ausgestellte Bescheinigung mit `beleg_ablegen` ab.

[estg45a]: https://www.gesetze-im-internet.de/estg/__45a.html
[kstg27]: https://www.gesetze-im-internet.de/kstg_1977/__27.html
[bmf2025]: https://www.steuerschroeder.de/blog/wp-content/uploads/2025/05/2025-05-16-kapitalertragSt-steuerbescheinigung.pdf
