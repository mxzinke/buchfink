# Umsatzsteuer-Verrechnung und Steuerrückstellung

Rechtsstand: 17. Oktober 2026. Wortlaut von § 249 HGB, § 4 Abs. 5b und § 36
EStG, § 10 KStG am 17.10.2026 gelesen.

Die Steuern rechnet Buchfink in den Erklärungen
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

Was Buchfink bucht: Zum Stichtag werden die Salden aller Vorsteuer- und
Umsatzsteuerkonten sowie 3820 Umsatzsteuer-Vorauszahlungen und 3830
Sondervorauszahlung gegeneinander verrechnet. Ein Überhang der Steuer geht auf
3841 Umsatzsteuerverbindlichkeiten Vorjahr, ein Überhang zugunsten der GmbH auf
1420. Die Buchung trägt den Verweis `USTV <Jahr>` und zählt nicht in die
Voranmeldung.

Ablauf:

1. Die letzte Voranmeldung des Jahres ist geprüft
   ([monatsabschluss/ablauf.md](../monatsabschluss/ablauf.md)).
2. Vorab-Prüfung: `auswertung` (`art: susa`) für die Steuerkonten, 3820 und
   3830. Nach der Verrechnung sollte auf 3841 etwa die
   noch nicht gezahlte Vorauszahlung des letzten Zeitraums stehen. Weicht der
   Betrag ab, such nach unbezahlten, doppelt gebuchten oder falsch
   zugeordneten Vorauszahlungen auf 3820.
3. `abschlussbuchung` mit `art: ust_verrechnung`: erst `nur_vorschau: true`,
   Betrag und Erläuterung vorlegen, nach dem Ja buchen. `buchung_erfassen`
   nimmt Steuerkonten nicht an.
4. Prüfe danach: Vorsteuer- und Umsatzsteuerkonten null.

Folgejahr: Die Zahlung der letzten Voranmeldung des alten Jahres und eine
Abschlusszahlung aus der Jahreserklärung gehören auf 3841, eine Erstattung auf
1420 ([bank/steuerzahlungen.md](../bank/steuerzahlungen.md#2-konten-je-steuerart)).

## 2. Vorsteuerberichtigung nach § 15a UStG

Führt Buchfink Wirtschaftsgüter im Verzeichnis nach § 15a UStG (Regeln:
[belege/vorsteuer.md](../belege/vorsteuer.md)), ist vor der letzten
Voranmeldung des Jahres der
Verwendungsanteil zu prüfen. Ihn zu erfassen und die Berichtigung zu buchen
(Kennziffer 64) bildet Buchfink noch nicht ab; sag das dem Menschen.

## 3. Steuerrückstellung: Betrag

- Noch nicht festgesetzte Steuern für das abgelaufene Jahr sind ungewisse
  Verbindlichkeiten ([§ 249 Abs. 1 Satz 1 HGB][hgb249]) und stehen unter den
  Steuerrückstellungen.
- Die Jahressteuer liefern die Erklärungen (`erklaerung`, `art: kst` und
  `art: gewst`, `zeitraum: <Jahr>`;
  [steuererklaerungen/kst.md](../steuererklaerungen/kst.md),
  [steuererklaerungen/gewst.md](../steuererklaerungen/gewst.md)).
- Rückstellung = Körperschaftsteuer + Solidaritätszuschlag + Gewerbesteuer des
  Jahres − Vorauszahlungen für dieses Jahr − anrechenbare Kapitalertragsteuer
  und darauf entfallender Solidaritätszuschlag, nur mit Steuerbescheinigung
  ([§ 36 Abs. 2 Nr. 2 EStG][estg36],
  [bank/zinsen-kapest.md](../bank/zinsen-kapest.md)).
- Körperschaftsteuer, Solidaritätszuschlag und Gewerbesteuer mindern das
  Einkommen nicht ([§ 10 Nr. 2 KStG][kstg10], [§ 4 Abs. 5b EStG][estg4]). Die
  Rückstellung ändert deshalb die Bemessungsgrundlage nicht; ein Rechengang
  genügt.

Prüfe vor dem Buchen:

1. Auf 7600, 7608 und 7610 stehen nur Vorauszahlungen für dieses Jahr
   (`auswertung`, `art: kontoblatt`). Nachzahlungen für Vorjahre gehören auf
   die Konten für Vorjahre
   ([bank/steuerzahlungen.md](../bank/steuerzahlungen.md#3-nachzahlung-und-erstattung-für-vorjahre));
   schlag die Umbuchung vor.
2. Der Hebesatz der Gemeinde stimmt mit dem Gewerbesteuerbescheid überein.
3. Ein Verlustvortrag aus Vorjahren ist berücksichtigt.

Der Vorschlag zur Steuerrückstellung nimmt die Vorauszahlungen von 7600, 7608
und 7610, den Hebesatz aus den Einstellungen und Verlustvorträge,
Hinzurechnungen und Kürzungen nur als Eingabe. Nimm diese Werte aus den
Erklärungen.

## 4. Steuerrückstellung: Buchung

`abschlussbuchung` mit `art: steuerrueckstellung`. Gib die Zahlen aus den
Erklärungen mit: `hebesatz`, `verlustvortrag_kst`,
`verlustvortrag_gewst`, `korrekturen` (etwa eine verdeckte
Gewinnausschüttung), `hinzurechnungen`, `kuerzungen`; weichen die Beträge ab,
`kst_betrag` und `gewst_betrag` mit Rechenweg in `grund`. Vergleiche die
Vorschau mit den Erklärungen, bevor du buchst. Die Buchung:

| Steuer | Nachzahlung erwartet | Vorauszahlungen zu hoch |
|---|---|---|
| Körperschaftsteuer und Solidaritätszuschlag | Soll 7600 und 7608, Haben 3040 Körperschaftsteuerrückstellung | Soll 1450 Körperschaftsteuerrückforderung, Haben 7600 und 7608 |
| Gewerbesteuer | Soll 7610, Haben 3035 Gewerbesteuerrückstellung nach § 4 Abs. 5b EStG | Soll 1435 Forderungen aus Gewerbesteuerüberzahlungen, Haben 7610 |

Buchfink selbst bucht Körperschaftsteuer und Solidaritätszuschlag zusammen auf
3040. Halte dich daran, damit das Kontoblatt eine Rückstellung je Steuer zeigt.

## 5. Folgejahr

Kommt der Bescheid, legt der Mensch ihn ab; den Abgleich beschreibt
[steuererklaerungen/bescheide.md](../steuererklaerungen/bescheide.md).
Zahlung, Verbrauch der Rückstellung und die Differenz auf den Konten für
Vorjahre: [bank/steuerzahlungen.md](../bank/steuerzahlungen.md#3-nachzahlung-und-erstattung-für-vorjahre).

[hgb249]: https://www.gesetze-im-internet.de/hgb/__249.html
[estg4]: https://www.gesetze-im-internet.de/estg/__4.html
[estg36]: https://www.gesetze-im-internet.de/estg/__36.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
