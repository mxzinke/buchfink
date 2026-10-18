# Umsatzsteuer der Holding

Rechtsstand: 17. Oktober 2026. Wortlaut des UStG am 17.10.2026 auf
gesetze-im-internet.de gelesen. Umsatzsteuer-Anwendungserlass (UStAE)
Abschnitte 2.3 und 15.22 in der Fassung „VZ 2010–2026 (aktuell)“ über die
NWB-Datenbank gelesen; der amtliche UStAE war nicht abrufbar.

Ob eine GmbH, die Beteiligungen hält, Unternehmerin ist und Vorsteuer abziehen
darf, ist eine **Wertungsfrage**. Du nennst Grundlage, Optionen und Folgen;
der Mensch entscheidet, bei Zweifeln mit Steuerberater. Buchfink geht von
einer Unternehmerin aus, die Umsatzsteuer anmeldet (buchfink-grundlagen,
Grenzen).

## Inhalt

1. Unternehmerin oder nicht
2. Drei Formen der Holding
3. Vorsteuer aus Kosten der Beteiligung
4. Leistungen an Tochtergesellschaften
5. Darlehen an Tochtergesellschaften
6. Verkauf einer Beteiligung
7. Organschaft
8. Rückfrage an den Menschen

## 1. Unternehmerin oder nicht

- Unternehmer ist, wer eine gewerbliche oder berufliche Tätigkeit
  selbstständig ausübt; gewerblich ist jede nachhaltige Tätigkeit zur
  Erzielung von Einnahmen ([§ 2 Abs. 1 UStG][ustg2]).
- Das bloße Erwerben, Halten und Veräußern von Beteiligungen ist keine
  unternehmerische Tätigkeit; Dividenden sind kein Entgelt für eine Leistung
  ([UStAE 2.3 Abs. 2 Satz 1 bis 3][ustae23]).
- Unternehmerisch wird es, wenn die Beteiligung gewerbsmäßig gehandelt wird,
  der eigenen unternehmerischen Tätigkeit dient (etwa Einkaufs- oder
  Absatzkonditionen sichert) oder die Holding durch entgeltliche Leistungen in
  die Verwaltung der Tochter eingreift, etwa mit administrativen,
  finanziellen, kaufmännischen oder technischen Dienstleistungen
  ([UStAE 2.3 Abs. 3 Satz 5 ff.][ustae23]).
- Eine Beteiligung gehört nur dann zum Unternehmen, wenn ein erkennbarer und
  objektiver wirtschaftlicher Zusammenhang mit der unternehmerischen
  Tätigkeit besteht ([UStAE 2.3 Abs. 4 Satz 1 bis 3][ustae23]).

## 2. Drei Formen der Holding

| Form | Merkmal | Folge | Fundstelle |
|---|---|---|---|
| Finanzholding | hält und verwaltet nur Beteiligungen, erbringt keine Leistungen gegen Entgelt | keine Unternehmerin, kein Vorsteuerabzug | [UStAE 2.3 Abs. 3 Satz 2][ustae23] |
| Führungs- oder Funktionsholding | greift im Sinne einheitlicher Leitung aktiv in das Tagesgeschäft der Töchter ein, durch entgeltliche Leistungen | Unternehmerin; Beteiligungen gehören zum Unternehmen | [UStAE 2.3 Abs. 3 Satz 3, 5 ff.][ustae23] |
| gemischte Holding | leitet einige Töchter, hält andere nur | unternehmerischer und nichtunternehmerischer Bereich | [UStAE 2.3 Abs. 3 Satz 4][ustae23] |

Eine GmbH mit eigenem operativem Geschäft (Dienstleistung, Software,
Agentur), die nebenbei Beteiligungen hält, ist Unternehmerin. Die
Beteiligungen gehören grundsätzlich nicht zum Unternehmen, außer es besteht
der Zusammenhang aus Abschnitt 1 ([UStAE 2.3 Abs. 2 Satz 5, 6][ustae23]).

Eine reine Finanzholding ist keine Unternehmerin. Buchfink bildet diesen Fall
nicht ab: Seine Voranmeldung setzt eine Unternehmerin voraus. Leg das als
Grenze vor; ob überhaupt Voranmeldungen abzugeben sind, klärt der Mensch mit
dem Steuerberater.

## 3. Vorsteuer aus Kosten der Beteiligung

- Gehört die Beteiligung zum Unternehmen, ist die Vorsteuer aus Kosten in
  ihrem Zusammenhang unter den allgemeinen Voraussetzungen abziehbar
  ([UStAE 15.22 Abs. 1 Satz 1, 2][ustae1522]; [§ 15 Abs. 1 UStG][ustg15]).
- Hält die Holding daneben Beteiligungen im nichtunternehmerischen Bereich,
  wird die Vorsteuer aus gemeinsamen Kosten aufgeteilt: Verwaltung,
  allgemeine Beratung, Steuerberatung ([UStAE 15.22 Abs. 1 Satz 3, 4][ustae1522]).
  Maßstab ist, in welche Ausgangsumsätze die Kosten eingehen.
- Hängt eine Eingangsleistung direkt mit den Umsätzen einer Tochter zusammen
  und nicht mit eigenen Umsätzen der Holding, ist die Vorsteuer bei der
  Holding nicht abziehbar ([UStAE 15.22 Abs. 1 Satz 6][ustae1522] mit BFH vom
  15.02.2023, XI R 24/22).
- In Buchfink buchst du solche Rechnungen nach buchfink-belege mit
  `beleg_buchen_mit_rueckfrage` und je Position `vorsteueranteil_promille`
  (abziehbarer Anteil in Promille, etwa 400 für 40 %; ohne Angabe voll) samt
  `vorsteueranteil_grund`. Der Grund nennt Maßstab und Rechnung, z. B.
  „Beratung Kauf Beteiligung X; 40 % den eigenen Umsätzen zuzuordnen“. Der
  Wert 0 heißt im Werkzeug „voll abziehbar“; ist gar keine Vorsteuer
  abziehbar, leg den Fall dem Menschen vor, bevor du buchst.
- Wertungsfrage: Maßstab der Aufteilung
  ([§ 15 Abs. 4 Satz 1 bis 3 UStG][ustg15]).

## 4. Leistungen an Tochtergesellschaften

- Leitungs-, Buchhaltungs-, IT- oder Personalleistungen gegen Entgelt machen
  die Holding zur Unternehmerin (Abschnitt 1). Sie stellt dafür Rechnungen mit
  Umsatzsteuer aus (buchfink-rechnungen).
- Halte die Leistungen in einem schriftlichen Vertrag fest und lege ihn als
  Beleg ab. Die Eingriffe in die Verwaltung der Tochter müssen durch
  unternehmerische Leistungen erfolgen ([UStAE 2.3 Abs. 3][ustae23]); die
  Führungsrolle allein reicht nicht.
- Wertungsfrage: Ist das Entgelt angemessen? Ein zu hohes oder zu niedriges
  Entgelt zwischen verbundenen Gesellschaften kann ertragsteuerlich eine
  verdeckte Gewinnausschüttung oder Einlage sein (buchfink-gesellschafter).

## 5. Darlehen an Tochtergesellschaften

- Die Gewährung von Krediten ist steuerfrei
  ([§ 4 Nr. 8 Buchst. a UStG][ustg4]). Vorsteuer aus Kosten, die diesen
  Umsätzen dienen, ist ausgeschlossen
  ([§ 15 Abs. 2 Satz 1 Nr. 1 UStG][ustg15]); bei gemeinsamen Kosten wird
  aufgeteilt ([§ 15 Abs. 4 UStG][ustg15]).
- Auf die Steuerfreiheit kann verzichtet werden, wenn der Kredit an einen
  anderen Unternehmer für dessen Unternehmen geht
  ([§ 9 Abs. 1 UStG][ustg9]). Buchfink bildet den Verzicht nicht ab
  (buchfink-grundlagen, Grenzen).
- Zinsen sind steuerfreie Umsätze. Führt der Mensch sie über die
  Anlagenkartei, setzt Buchfink den Steuerfall „steuerfrei“. Wie sie in der
  Voranmeldung erscheinen, prüfst du mit `erklaerung`
  (buchfink-monatsabschluss).

## 6. Verkauf einer Beteiligung

- Das bloße Veräußern ist keine unternehmerische Tätigkeit, außer die
  Beteiligung gehört zum Unternehmen ([UStAE 15.22 Abs. 2 Satz 1, 2][ustae1522]).
- Vorsteuer aus Kosten des Verkaufs (Makler, Berater, Notar) ist nicht
  abziehbar: bei einer Beteiligung außerhalb des Unternehmens wegen des nicht
  steuerbaren Verkaufs, bei einer Beteiligung im Unternehmen wegen der
  Steuerfreiheit nach [§ 4 Nr. 8 Buchst. e, f UStG][ustg4]
  ([UStAE 15.22 Abs. 2 Satz 3 bis 5][ustae1522]). Ausnahme: Verkauf an einen
  Unternehmer für dessen Unternehmen mit Verzicht auf die Steuerfreiheit
  ([§ 9 Abs. 1 UStG][ustg9]); Buchfink bildet den Verzicht nicht ab,
  Steuerberater.

## 7. Organschaft

Ist eine Tochter finanziell, wirtschaftlich und organisatorisch in das
Unternehmen der Holding eingegliedert, ist sie umsatzsteuerlich nicht
selbstständig ([§ 2 Abs. 2 Nr. 2 Satz 1 UStG][ustg2]). Die im Inland
gelegenen Unternehmensteile werden als ein Unternehmen behandelt; Leistungen
zwischen ihnen sind Innenleistungen (Satz 2, 3). Buchfink bildet die
Organschaft nicht ab. Bei einer
Führungsholding mit Mehrheitsbeteiligung: Weise auf die Möglichkeit hin und
empfiehl den Steuerberater.

## 8. Rückfrage an den Menschen

```
[Nr.] Wertungsfrage: Vorsteuer der Holding
Sachverhalt: <Was die GmbH tut: eigenes Geschäft, Leistungen an Töchter, nur Halten>.
Grundlage: § 2 Abs. 1, § 15 UStG; UStAE 2.3 Abs. 2–4, 15.22.
Optionen: (a) Beteiligung im Unternehmen, Vorsteuer voll
          (b) gemischte Holding, Vorsteuer anteilig nach <Maßstab>
          (c) Beteiligung außerhalb des Unternehmens, keine Vorsteuer
Folge: <Betrag der Vorsteuer je Option, Wirkung auf die Voranmeldung>.
Empfehlung: Steuerberater bei <Grund>.
```

[ustae23]: https://datenbank.nwb.de/Dokument/378652_2___3/
[ustae1522]: https://datenbank.nwb.de/Dokument/378652_15___22/
[ustg2]: https://www.gesetze-im-internet.de/ustg_1980/__2.html
[ustg4]: https://www.gesetze-im-internet.de/ustg_1980/__4.html
[ustg9]: https://www.gesetze-im-internet.de/ustg_1980/__9.html
[ustg15]: https://www.gesetze-im-internet.de/ustg_1980/__15.html
