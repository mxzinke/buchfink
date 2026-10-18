# § 8b KStG: Beteiligungserträge bei der Körperschaftsteuer

Rechtsstand: 17. Oktober 2026. Wortlaut von KStG, EStG und InvStG am
17.10.2026 auf gesetze-im-internet.de gelesen. Konten aus
`internal/accounting/skr04_2026.json` und `internal/accounting/asset_accounts.go`.
Die Sätze von 5 % und 10 % gelten seit Jahren unverändert; sie ändern sich nur
durch Gesetz. Stand: Oktober 2026.

Die Überleitung zum Einkommen baut buchfink-steuererklaerungen. Hier
berechnest du die Zeile „steuerfreie Beteiligungserträge, 5 % nicht abziehbare
Ausgaben“ und die Hinzurechnungen für Verluste.

## Inhalt

1. Übersicht
2. Dividenden und andere Bezüge
3. Streubesitz unter 10 %
4. Gewinne aus der Veräußerung
5. Verluste und Teilwertabschreibungen
6. Darlehen an die Tochter
7. Was nicht unter § 8b fällt
8. Beteiligungsposten nach Konten
9. Beispiele
10. Erklärung

## 1. Übersicht

| Vorgang | Bleibt außer Ansatz | Gilt als nicht abziehbar | Fundstelle |
|---|---|---|---|
| Dividende, Quote zu Beginn des Kalenderjahres mindestens 10 % | ganze Dividende | 5 % der Dividende | [§ 8b Abs. 1, 4, 5 KStG][kstg8b] |
| Dividende, Quote darunter | nichts | nichts | [§ 8b Abs. 4 Satz 1, 7 KStG][kstg8b] |
| Gewinn aus Verkauf, Auflösung, Kapitalherabsetzung, Wertaufholung eines Anteils | ganzer Gewinn | 5 % des Gewinns | [§ 8b Abs. 2, 3 Satz 1 KStG][kstg8b] |
| Verlust, Teilwertabschreibung, andere Gewinnminderung beim Anteil | – | ganze Minderung | [§ 8b Abs. 3 Satz 3 KStG][kstg8b] |

Tatsächliche Kosten der Beteiligung (Zinsen für den Kaufpreis, Beratung,
Verwaltung) bleiben abziehbar, weil § 3c Abs. 1 EStG nicht anzuwenden ist
([§ 8b Abs. 3 Satz 2, Abs. 5 Satz 2 KStG][kstg8b]). Die 5 % ersetzen sie.

## 2. Dividenden und andere Bezüge

- Außer Ansatz bleiben Bezüge im Sinne des § 20 Abs. 1 Nr. 1, 2, 9 und 10
  Buchst. a EStG ([§ 8b Abs. 1 Satz 1 KStG][kstg8b]): Gewinnanteile,
  verdeckte Gewinnausschüttungen ([§ 20 Abs. 1 Nr. 1 Satz 2 EStG][estg20])
  und Bezüge nach Auflösung.
- Das gilt nur, soweit die Bezüge das Einkommen der leistenden Gesellschaft
  nicht gemindert haben ([§ 8b Abs. 1 Satz 2 KStG][kstg8b]). Bei einer
  verdeckten Gewinnausschüttung der Tochter an die Holding prüft das der
  Steuerberater.
- 5 % der außer Ansatz bleibenden Bezüge gelten als nicht abziehbare
  Betriebsausgaben ([§ 8b Abs. 5 Satz 1 KStG][kstg8b]).
- Eine Einlagenrückgewähr ist kein Bezug
  ([§ 20 Abs. 1 Nr. 1 Satz 3 EStG][estg20]); sie gehört nicht in diese Zeile.

## 3. Streubesitz unter 10 %

- Bezüge sind voll zu berücksichtigen, wenn die Beteiligung zu Beginn des
  Kalenderjahres unmittelbar weniger als 10 % des Grund- oder Stammkapitals
  betragen hat ([§ 8b Abs. 4 Satz 1 KStG][kstg8b]). Die 5 % entfallen dann
  ([§ 8b Abs. 4 Satz 7 KStG][kstg8b]).
- Der Erwerb einer Beteiligung von mindestens 10 % gilt als zu Beginn des
  Kalenderjahres erfolgt ([§ 8b Abs. 4 Satz 6 KStG][kstg8b]). Wertungsfrage:
  Ob ein Zukauf, der erst zusammen mit dem Bestand 10 % erreicht, darunter
  fällt, klärt der Steuerberater.
- Über eine Mitunternehmerschaft gehaltene Anteile werden anteilig
  zugerechnet ([§ 8b Abs. 4 Satz 4, 5 KStG][kstg8b]); Buchfink bildet
  Mitunternehmerschaften nicht ab.
- Die Grenze gilt nur für Bezüge. Veräußerungsgewinne aus Streubesitz bleiben
  nach Abs. 2 außer Ansatz.
- Das Feld „Beteiligungsquote“ der Anlagenkartei hält einen Wert ohne Datum.
  Halte die Quote zu Beginn jedes Kalenderjahres in der Beteiligungsliste fest.

## 4. Gewinne aus der Veräußerung

- Gewinne aus der Veräußerung eines Anteils an einer Körperschaft, deren
  Leistungen beim Empfänger Einnahmen nach § 20 Abs. 1 Nr. 1, 2, 9, 10 Buchst. a
  EStG sind, bleiben außer Ansatz ([§ 8b Abs. 2 Satz 1 KStG][kstg8b]).
- Veräußerungsgewinn ist der Preis nach Abzug der Veräußerungskosten, soweit
  er den steuerlichen Buchwert übersteigt ([§ 8b Abs. 2 Satz 2 KStG][kstg8b]).
  Hat Buchfink Veräußerungskosten (Notar, Berater) als Aufwand gebucht,
  gehören sie in die Rechnung des Gewinns und werden mit ihm herausgenommen.
- Ebenso: Gewinne aus Auflösung, Herabsetzung des Nennkapitals und dem
  Wertansatz nach § 6 Abs. 1 Nr. 2 Satz 3 EStG (Wertaufholung)
  ([§ 8b Abs. 2 Satz 3 KStG][kstg8b]); die verdeckte Einlage eines Anteils
  gilt als Veräußerung (Satz 6).
- Ausnahme: Soweit der Anteil früher steuerwirksam auf den niedrigeren
  Teilwert abgeschrieben und das nicht wieder ausgeglichen wurde, bleibt der
  Gewinn steuerpflichtig ([§ 8b Abs. 2 Satz 4 KStG][kstg8b]). Frag nach der
  Geschichte des Anteils, wenn er vor der Buchführung in Buchfink erworben
  wurde.
- 5 % des Gewinns gelten als nicht abziehbare Betriebsausgaben
  ([§ 8b Abs. 3 Satz 1 KStG][kstg8b]).

## 5. Verluste und Teilwertabschreibungen

- Gewinnminderungen im Zusammenhang mit dem Anteil sind nicht zu
  berücksichtigen ([§ 8b Abs. 3 Satz 3 KStG][kstg8b]): Teilwertabschreibung,
  Verlust beim Verkauf, Verlust bei Auflösung. Rechne sie hinzu.
- Eine spätere Wertaufholung bleibt nach Abs. 2 Satz 3 außer Ansatz, mit 5 %
  nicht abziehbar (Abschnitt 4).

## 6. Darlehen an die Tochter

- Gewinnminderungen aus einer Darlehensforderung oder aus der Inanspruchnahme
  einer Sicherheit für ein Darlehen sind ebenfalls nicht zu berücksichtigen,
  wenn der Darlehensgeber zu mehr als einem Viertel am Kapital der
  Darlehensnehmerin beteiligt ist oder war
  ([§ 8b Abs. 3 Satz 4 KStG][kstg8b]). Das gilt auch für Forderungen aus
  Rechtshandlungen, die einem Darlehen wirtschaftlich vergleichbar sind
  (Satz 8), etwa lange stehengelassene Forderungen aus Lieferungen.
- Währungskursverluste zählen nicht dazu (Satz 6).
- Ausnahme: Die GmbH weist nach, dass ein fremder Dritter das Darlehen bei
  sonst gleichen Umständen gewährt oder nicht zurückgefordert hätte; dabei
  zählen nur die eigenen Sicherungsmittel der Tochter (Satz 7).
  Wertungsfrage, Steuerberater.
- Eine spätere Wertaufholung auf die Darlehensforderung bleibt außer Ansatz,
  soweit die Abschreibung nach Satz 3 unberücksichtigt blieb (Satz 9). Hier
  gibt es keinen 5-%-Zuschlag; Abs. 3 Satz 1 nennt nur Gewinne nach Abs. 2.

## 7. Was nicht unter § 8b fällt

- Erträge aus Investmentfonds, auch aus Aktien-ETF
  ([§ 16 Abs. 3 InvStG][invstg16]); dort gilt die Teilfreistellung (Referenz
  zu Investmentfonds, Liste in SKILL.md).
- Zinsen, auch aus Darlehen an die Tochter.
- Gewinnanteile aus Personengesellschaften; § 8b Abs. 6 KStG regelt
  Mitunternehmerschaften, die Buchfink nicht abbildet.
- Anteile im Handelsbestand von Kredit-, Wertpapier- und
  Finanzdienstleistungsinstituten und im Umlaufvermögen von
  Finanzunternehmen, an denen solche Institute zu mehr als 50 % beteiligt
  sind ([§ 8b Abs. 7 KStG][kstg8b]); Anteile bei Lebens- und
  Krankenversicherern ([§ 8b Abs. 8 KStG][kstg8b]). Trifft das zu: Grenze,
  Steuerberater.

## 8. Beteiligungsposten nach Konten

Lies die Salden mit `auswertung` (`art: susa`) und prüfe jeden Betrag am
Kontoblatt (`art: kontoblatt`), bevor du ihn übernimmst.

| Konto (SKR04) | Inhalt | Korrektur Körperschaftsteuer |
|---|---|---|
| 7005, 7006, 7014, 7015, 7103, 7104 | Dividenden | − Betrag und + 5 % ([§ 8b Abs. 1, 5][kstg8b]); bei Quote unter 10 %: keine Korrektur |
| 4852 Erlöse, 4858 Restbuchwert | Verkauf eines Anteils aus dem Anlagevermögen mit Buchgewinn | − (4852 − 4858 − gebuchte Veräußerungskosten), + 5 % davon ([§ 8b Abs. 2, 3 Satz 1][kstg8b]) |
| 4906 Erträge, 4904 Buchwert | Verkauf aus dem Umlaufvermögen mit Buchgewinn | wie oben |
| 6892 Erlöse, 6898 Restbuchwert | Verkauf mit Buchverlust | + (6898 − 6892 + Veräußerungskosten) ([§ 8b Abs. 3 Satz 3][kstg8b]) |
| 7204, 7214 | Teilwertabschreibung auf Anteile | + Betrag ([§ 8b Abs. 3 Satz 3][kstg8b]) |
| 4913, 4914, 4916 | Zuschreibungen | bei einem Anteil − Betrag und + 5 % ([§ 8b Abs. 2 Satz 3][kstg8b]); bei einer Darlehensforderung − Betrag ohne Zuschlag (Abs. 3 Satz 9) |
| 7200, 7201 auf Ausleihungen an die Tochter | Teilwertabschreibung Darlehen | + Betrag, wenn Abs. 3 Satz 4 greift und Satz 7 nicht |
| 7630, 7633 | Kapitalertragsteuer, Solidaritätszuschlag | + Betrag ([§ 10 Nr. 2 KStG][kstg10]); Anrechnung getrennt |

Buchfink bucht die Zuschreibung eines als „Anteil an einer
Kapitalgesellschaft“ gekennzeichneten Anlageguts auf 4913, dessen SKR04-Name
§ 8b Abs. 3 Satz 9 nennt. Entscheide nach dem Gegenstand, nicht nach dem
Kontonamen.

## 9. Beispiele

Rechnerische Beispiele ohne Beratung.

**A. Dividende einer 100-%-Tochter** von 100.000,00 €, gebucht auf 7006,
Kapitalertragsteuer 25.000,00 € auf 7630, Solidaritätszuschlag 1.375,00 € auf
7633:

| Posten | Betrag |
|---|---|
| außer Ansatz nach § 8b Abs. 1 | − 100.000,00 € |
| nicht abziehbar nach § 8b Abs. 5 | + 5.000,00 € |
| Kapitalertragsteuer und Solidaritätszuschlag, § 10 Nr. 2 | + 26.375,00 € |
| Wirkung auf das Einkommen gegenüber dem Jahresüberschuss | − 68.625,00 € |

Anzurechnen sind 25.000,00 € Kapitalertragsteuer und 1.375,00 €
Solidaritätszuschlag.

**B. Verkauf einer Beteiligung:** Erlös 200.000,00 € (4852), Restbuchwert
50.000,00 € (4858), Notar und Berater 5.000,00 € als Aufwand gebucht.
Veräußerungsgewinn 145.000,00 €. Korrektur: − 145.000,00 € und + 7.250,00 €
(5 %).

**C. Streubesitz:** Dividende 1.000,00 € aus Aktien mit 2 % Quote, gebucht auf
7014. Keine Korrektur nach § 8b; nur Kapitalertragsteuer und
Solidaritätszuschlag rechnest du hinzu.

## 10. Erklärung

Die Körperschaftsteuererklärung trägt der Mensch in Mein ELSTER ein
(buchfink-steuererklaerungen). Die
[ELSTER-Anleitung zur Körperschaftsteuer 2025][elsterkst] nennt für die Anlage GK den Abschnitt „Anwendung des § 8b KStG und
vergleichbarer Vorschriften (Zeilen 150 bis 195)“; zu Zeile 156 weist sie auf die
Beteiligungshöhe zu Beginn des Kalenderjahres hin und schließt
REIT-Ausschüttungen aus. Die einzelnen Zeilen je Posten sind nicht belegt.
Übergib die Posten getrennt nach Rechtsgrundlage
(Abs. 1, Abs. 2, Abs. 3 Satz 1, Abs. 3 Satz 3, Abs. 4, Abs. 5) und mit
Bruttobeträgen; der Mensch folgt der Ausfüllhilfe des Formulars.

[elsterkst]: https://www.elster.de/eportal/helpGlobal?themaGlobal=help_kst_2025
[estg20]: https://www.gesetze-im-internet.de/estg/__20.html
[invstg16]: https://www.gesetze-im-internet.de/invstg_2018/__16.html
[kstg8b]: https://www.gesetze-im-internet.de/kstg_1977/__8b.html
[kstg10]: https://www.gesetze-im-internet.de/kstg_1977/__10.html
