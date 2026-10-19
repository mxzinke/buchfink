# Unfertige Leistungen

Rechtsstand: 17. Oktober 2026. Wortlaut von §§ 252, 253, 255, 266, 268 HGB,
§ 6 Abs. 1 Nr. 1b EStG und §§ 640, 641 BGB am 17.10.2026 gelesen. Abschläge,
Anzahlungen und Schlussrechnung regelt buchfink-rechnungen.

## Inhalt

1. Wann der Posten entsteht
2. Bewertung
3. Verlustprojekte
4. Erhaltene Anzahlungen
5. Buchen
6. Daten beschaffen

## 1. Wann der Posten entsteht

- Gewinne sind nur zu berücksichtigen, wenn sie am Stichtag realisiert sind
  ([§ 252 Abs. 1 Nr. 4 HGB][hgb252]). Bei einem Werkvertrag ist die Vergütung
  mit der Abnahme zu entrichten, bei Teilabnahme je Teil
  ([§ 641 Abs. 1 BGB][bgb641], [§ 640 BGB][bgb640]).
- Ein Projekt, das am Stichtag begonnen, aber weder abgenommen noch als
  Teilleistung abgerechnet ist, hat noch keinen Erlös. Die bisher angefallenen
  Kosten werden als unfertige Leistung aktiviert (Bilanzposten „unfertige
  Erzeugnisse, unfertige Leistungen“, [§ 266 Abs. 2 B. I. 2. HGB][hgb266]).
- Bei Dienstverträgen mit laufender Abrechnung (Stunden, Monatspauschale)
  entsteht in der Regel kein Bestand. Noch nicht abgerechnete Stunden des alten
  Jahres sind Erlös des alten Jahres (SKILL.md, Schritt 5).

Wertungsfrage: ob ein Projekt am Stichtag abgenommen, eine Teilleistung
abgerechnet oder ein Dienstvertrag ist. Frag nach Vertrag, Abnahmeprotokoll und
Abrechnungsweise.

## 2. Bewertung

Zu Herstellungskosten ([§ 255 Abs. 2 HGB][hgb255]), höchstens zum niedrigeren
beizulegenden Wert (Abschnitt 3):

| Bestandteil | Behandlung |
|---|---|
| Einzelkosten: Fremdleistungen und Material für das Projekt, Löhne der Beschäftigten für ihre Projektstunden | Pflicht (Satz 2) |
| angemessene Teile der Gemeinkosten, die durch die Leistung veranlasst sind, einschließlich anteiliger Abschreibung | Pflicht (Satz 2) |
| allgemeine Verwaltung, soziale Einrichtungen, freiwillige Sozialleistungen, Altersversorgung | Wahlrecht (Satz 3); steuerlich ebenso, einheitlich mit dem handelsrechtlichen Abschluss ([§ 6 Abs. 1 Nr. 1b EStG][estg6]) |
| Vertriebskosten, Forschung | verboten (Satz 4) |
| Fremdkapitalzinsen | nicht, außer für die Herstellung finanziert ([§ 255 Abs. 3 HGB][hgb255]) |
| Gewinnanteil | nie |

- Einmal gewählt, gilt die Methode in den Folgejahren weiter
  ([§ 252 Abs. 1 Nr. 6 HGB][hgb252]).
- Wertungsfragen: Höhe und Schlüssel der Gemeinkosten; ob die Arbeit des
  Geschäftsführers mit einem Anteil seines Gehalts eingeht, wenn er selbst im
  Projekt arbeitet. Leg Rechenweg und Annahmen offen.

## 3. Verlustprojekte

- Übersteigen die bisherigen Kosten zuzüglich der noch zu erwartenden Kosten
  den Erlös, ist der Bestand auf den niedrigeren Wert abzuschreiben
  ([§ 253 Abs. 4 Satz 2 HGB][hgb253]).
- Ist der erwartete Verlust größer als der aktivierte Bestand, ist für den
  Rest eine Rückstellung für drohende Verluste zu bilden
  ([§ 249 Abs. 1 Satz 1 HGB][hgb249]); steuerlich ist sie nicht zulässig
  (Liste Abschlussposten; SKILL.md, Schritt 6).

## 4. Erhaltene Anzahlungen

Erhaltene Anzahlungen stehen unter den Verbindlichkeiten oder werden vom Posten
Vorräte offen abgesetzt ([§ 268 Abs. 5 Satz 2 HGB][hgb268]). Buchfink führt sie
auf eigenen Konten (buchfink-rechnungen). Verrechne sie nicht mit dem Bestand
der unfertigen Leistung.

## 5. Buchen

- Erhöhung: Soll 1080 Unfertige Leistungen (Bestand), Haben 4815
  Bestandsveränderungen unfertige Leistungen.
- Verminderung oder Auflösung: umgekehrt.
- Zu jedem Stichtag passt du den Bestand je Projekt auf den neuen Wert an; die
  Differenz geht über 4815. Ist das Projekt im Folgejahr abgerechnet, sinkt der
  Bestand dieses Projekts auf null.
- Werkzeug: `abschlussbuchung` mit `art: unfertige_leistungen` und
  `projekte` (je `name`, `kontakt_id`, `einzelkosten`, `gemeinkosten`,
  `auftragswert`, `restkosten`, `anzahlungen`, `notiz`); `grund` und `notiz`
  nennen Vertrag, Stand am Stichtag, Kostenaufstellung, Gemeinkostensatz und
  Begründung des Werts.

## 6. Daten beschaffen

Buchfink führt keine Projektkostenrechnung. Frag den Menschen nach:

- Liste der Projekte, die am Stichtag liefen, mit Vertrag und Abnahmestand;
- Stunden je Beschäftigtem und Projekt (Zeiterfassung);
- Fremdleistungen je Projekt: `buchungen_suchen` mit Kontakt oder Text der
  Subunternehmer;
- erwartete Restkosten bei Projekten mit Verlustrisiko.

Fertig, wenn jedes laufende Projekt bewertet oder vom Menschen als unwesentlich
festgehalten ist.

[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
[hgb252]: https://www.gesetze-im-internet.de/hgb/__252.html
[hgb253]: https://www.gesetze-im-internet.de/hgb/__253.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[hgb266]: https://www.gesetze-im-internet.de/hgb/__266.html
[hgb268]: https://www.gesetze-im-internet.de/hgb/__268.html
[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[bgb640]: https://www.gesetze-im-internet.de/bgb/__640.html
[bgb641]: https://www.gesetze-im-internet.de/bgb/__641.html
