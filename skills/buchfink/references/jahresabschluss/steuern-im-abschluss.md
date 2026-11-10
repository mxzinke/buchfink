# Umsatzsteuer-Verrechnung und Steuerrückstellung

Rechtsstand: 17. Oktober 2026.

Die Jahressteuern rechnet Buchfink in den Erklärungen
([steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
[steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)); Zahlungen
buchst du nach [bank/steuerzahlungen.md](../bank/steuerzahlungen.md).

## Inhalt

1. Umsatzsteuer-Verrechnung
2. Vorsteuerberichtigung nach § 15a UStG
3. Steuerrückstellung: Betrag
4. Steuerrückstellung: Buchung
5. Folgejahr

## 1. Umsatzsteuer-Verrechnung

`abschlussbuchung` mit `art: ust_verrechnung` bucht zum Stichtag die Salden
aller Vorsteuer- und Umsatzsteuerkonten mit 3820 Umsatzsteuer-Vorauszahlungen
und 3830 Sondervorauszahlung auf 3841 Umsatzsteuerverbindlichkeiten Vorjahr,
bei einem Überhang zugunsten der GmbH auf 1420. Sie zählt nicht in die
Voranmeldung.

1. Die letzte Voranmeldung des Jahres ist geprüft
   ([monatsabschluss/ablauf.md](../monatsabschluss/ablauf.md)).
2. Mit `auswertung` (`art: susa`) vorab prüfen: Nach der Verrechnung steht
   auf 3841 etwa die unbezahlte Vorauszahlung des letzten Zeitraums; sonst such
   unbezahlte, doppelte oder falsch zugeordnete Vorauszahlungen auf 3820.
3. Verrechnung buchen. Fertig, wenn Vorsteuer- und Umsatzsteuerkonten null
   zeigen.

Konten für die Zahlungen im Folgejahr:
[bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart).

## 2. Vorsteuerberichtigung nach § 15a UStG

Für Wirtschaftsgüter im Verzeichnis nach § 15a UStG
([belege/vorsteuer.md](../belege/vorsteuer.md)) prüfst du vor der letzten
Voranmeldung den Verwendungsanteil. Ihn zu erfassen und die Berichtigung zu
buchen (Kennziffer 64) bildet Buchfink nicht ab; sag das dem Menschen.

## 3. Steuerrückstellung: Betrag

- Noch nicht festgesetzte Steuern des abgelaufenen Jahres sind ungewisse
  Verbindlichkeiten ([§ 249 Abs. 1 Satz 1 HGB][hgb249]).
- Rückstellung = Körperschaftsteuer + Solidaritätszuschlag + Gewerbesteuer des
  Jahres (`erklaerung`, `art: kst` und `art: gewst`) − Vorauszahlungen für
  dieses Jahr − anrechenbare Kapitalertragsteuer mit Solidaritätszuschlag
  ([Anrechnung](../steuererklaerungen/kst.md#9-steuer-rundung-anrechnung-vorauszahlungen)).
- Diese Steuern mindern das Einkommen nicht
  ([kst.md, Abschnitt 2](../steuererklaerungen/kst.md#2-überleitung-zum-zu-versteuernden-einkommen));
  ein Rechengang genügt.

Prüfe vor dem Buchen:

1. Auf 7600, 7608 und 7610 stehen nur Vorauszahlungen für dieses Jahr
   (`auswertung`, `art: kontoblatt`). Nachzahlungen für Vorjahre gehören auf
   die Konten für Vorjahre
   ([bank/steuerzahlungen.md](../bank/steuerzahlungen.md#3-nachzahlung-und-erstattung-für-vorjahre));
   schlag die Umbuchung vor.
2. Der Hebesatz stimmt mit dem Gewerbesteuerbescheid überein.
3. Ein Verlustvortrag aus Vorjahren ist berücksichtigt.

## 4. Steuerrückstellung: Buchung

`abschlussbuchung` mit `art: steuerrueckstellung` nimmt die Vorauszahlungen
von 7600, 7608 und 7610; Hebesatz, Verlustvorträge, Korrekturen (etwa eine
verdeckte Gewinnausschüttung), Hinzurechnungen und Kürzungen gibst du aus den
Erklärungen mit. Vergleiche die Vorschau mit den Erklärungen; weichen sie ab,
nenne `kst_betrag` und `gewst_betrag` mit Rechenweg in `grund`. Konten:
Körperschaftsteuer mit Solidaritätszuschlag 3040, Gewerbesteuer 3035,
Überzahlung 1450 bzw. 1435.

## 5. Folgejahr

Bescheid: [steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md).
Zahlung, Verbrauch der Rückstellung und Differenz:
[bank/steuerzahlungen.md](../bank/steuerzahlungen.md#3-nachzahlung-und-erstattung-für-vorjahre).

[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
