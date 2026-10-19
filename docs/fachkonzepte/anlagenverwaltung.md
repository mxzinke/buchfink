# Anlagen

[Fachkonzepte](README.md) · [Jahresabschluss](jahresabschluss.md)

Anforderungen: BEW-02 bis BEW-06, BEW-10, JAB-05. Code: `internal/domain/asset.go`,
`internal/accounting/afa.go`, `internal/accounting/asset_accounts.go`,
`service.AssetService`. Die Oberfläche (Bücher → Anlagen,
`frontend/src/pages/books/AssetsTab.tsx`) zeigt Verzeichnis,
Abschreibungsplan und Anlagenspiegel nur an; erfasst, abgeschrieben und
abgegangen wird über einen verbundenen Agenten.

## Sofortabzug, Sammelposten oder Aktivierung

Vor der Abschreibung steht die Einordnung:

| Fall | Voraussetzung | Konten |
|---|---|---|
| Sofortabzug | selbständig nutzbar, Anschaffungskosten bis 800 € ([§ 6 Abs. 2 EStG][estg6]) | **0670** GWG, **6260** Sofortabschreibung |
| Sammelposten | über 250 € bis 1.000 €, aufgelöst über fünf Jahre (§ 6 Abs. 2a EStG) | **0675** Sammelposten, **6264** Abschreibung |
| Aktivierung | alles darüber und alles nicht selbständig Nutzbare | Anlagekonto, **6220** oder **6222** |

„Selbständig nutzbar“ ist die eigentliche Hürde: ein Monitor ohne Rechner ist
kein GWG. Das Sammelposten-Wahlrecht gilt einheitlich für alle
Wirtschaftsgüter eines Jahres; je Wirtschaftsjahr entsteht genau ein
Sammelposten. Die Wertgrenzen stehen datiert im Code
(`accounting.AfAParametersFor`) und binden die Eingabe, auch für ältere Jahre.

## Zugang

Anschaffungskosten sind Preis plus Nebenkosten minus Minderungen
([§ 255 Abs. 1 HGB][hgb255]). Ein Skonto auf eine Anlagenrechnung mindert
deshalb die Anschaffungskosten: der Zahlungsflow erkennt die Rechnung als
Zugang und bucht das Skonto auf das Anlagekonto. Die Steuerkorrektur nach
§ 17 UStG bleibt unverändert.

Nachträgliche Herstellungskosten erhöhen den Buchwert und verteilen sich ab
ihrem Jahr auf die Restnutzungsdauer. Erhaltungsaufwand geht sofort in die GuV
und wird nur mit dem Anlagegut verknüpft. Die Abgrenzung ist eine Einschätzung;
Buchfink fragt sie ab.

Anzahlungen und Anlagen im Bau stehen auf **0700**. Die Fertigstellung bucht um,
setzt das Datum der Betriebsbereitschaft und startet damit die Abschreibung.

## Abschreibung

- Monatsgenau ab dem Anschaffungsmonat.
- Linear oder degressiv in den Zeitfenstern des [§ 7 Abs. 2 EStG][estg7], die
  datiert im Code stehen. Den Wechsel von degressiv auf linear rechnet Buchfink
  selbst und zeigt ihn als eigene Zeile im Plan.
- Sonderabschreibung nach [§ 7g Abs. 5 EStG][estg7g] neben der linearen oder
  degressiven AfA, ohne Zeitanteil im Anschaffungsjahr, auf eigenem Konto
  (**6241**, für Fahrzeuge **6242**). Nach dem Begünstigungszeitraum verteilt
  Buchfink den Restwert auf die Restnutzungsdauer (§ 7a Abs. 9 EStG). Gewinngrenze
  und betriebliche Nutzung fragt es ab. Der steuerliche Wert läuft neben dem
  handelsrechtlichen und erscheint in der Überleitung der E-Bilanz.
- Nutzungsdauern schlägt Buchfink für eindeutige Fälle aus den AfA-Tabellen vor;
  jeder Vorschlag ist überschreibbar.
- Außerplanmäßige Abschreibung bei dauernder Wertminderung (§ 253 Abs. 3 HGB)
  auf **6230**, nur mit Grund. Fällt der Grund weg, ist zuzuschreiben
  (§ 253 Abs. 5 HGB); der Jahresabschluss fragt das in einem eigenen Baustein.

Die Abschreibung ist eine Abschlussbuchung zum Bilanzstichtag. Fehlt sie für
ein Anlagegut, blockiert der Prüflauf die Jahresfestschreibung.

## Abgang

Beim Abgang wird die Abschreibung bis zum Abgangsmonat nachgeholt, der
Restbuchwert ausgebucht und der Erlös mit Umsatzsteuer erfasst. Der SKR04
wählt das Erlöskonto nach dem Ergebnis: **4845** bei Buchgewinn, **6885** bei
Buchverlust. Buchfink rechnet das Ergebnis deshalb, bevor es das Konto wählt.

Finanzanlagen haben eigene Regeln: keine planmäßige Abschreibung, gemildertes
Niederstwertprinzip, Teilabgang nach Stückzahl und eine Tilgung von
Ausleihungen ohne Erlös und ohne Steuerfall.

## Laufende Erträge

Dividenden, Ausschüttungen und Zinsen einer Finanzanlage bucht Buchfink brutto
auf das Ertragskonto, das aus dem Anlagekonto folgt
(`accounting.AssetIncomeAccount`): Anteile an Kapitalgesellschaften auf die
Konten des § 8b KStG, Investmentanteile auf 7010. Kapitalertragsteuer und
Solidaritätszuschlag stehen getrennt auf **7630** und **7633**, weil die
Steuerrückstellung sie getrennt anrechnet. Der Ertrag hängt als Bewegung am
Anlagegut (`AssetService.BookAssetIncome`); daraus liest die
Körperschaftsteuererklärung die Bezüge je Anteil und die Fondserträge. Der
Agent bucht ihn am Bankumsatz (`bankumsatz_buchen` mit `kapitalertrag`).

## Fremdwährung und Investmentanteile

Anlagegüter in Fremdwährung bewertet Buchfink zum Stichtag nach
[§ 256a HGB][hgb256a], nach oben durch die Anschaffungskosten gedeckelt.
Gebucht wird über außerplanmäßige Abschreibung oder Zuschreibung.

Für Investmentanteile rechnet Buchfink Teilfreistellung (§ 20 InvStG) und
Vorabpauschale (§ 18 InvStG) als Nebenrechnung. Die Vorabpauschale wird nicht
gebucht, aber festgehalten, weil sie beim Abgang abzuziehen ist. Die
Teilfreistellung folgt für GmbH und UG aus der Stellung als Körperschaft; für
die Ausnahmen des § 20 Abs. 1 Sätze 4 und 5 InvStG lässt sich das abweichend
festlegen. Den Basiszins ([§ 18 Abs. 4 InvStG][invstg18]) kennt Buchfink für
die Jahre, deren BMF-Schreiben gelesen wurde (`accounting.InvestmentBaseRateFor`:
2024 bis 2026); eine abweichende Eingabe weist es ab. Für andere Jahre gibt der
Mensch ihn mit Quelle ein. Als zugeflossen gilt die Vorabpauschale am ersten
Werktag des Folgejahres, Samstag ausgenommen, wie das BMF für 2026 den
4. Januar 2027 nennt. Der Agent rechnet und hält sie mit `abschlussbuchung`
(`art: vorabpauschale`, `AssetService.Vorabpauschale`) fest; die Fondsart
nimmt die Aktivierung auf (`anlage_aktivieren` mit `fondsart`). Die
Körperschaftsteuererklärung setzt sie im Geschäftsjahr des fiktiven Zuflusses
an.

## Anlagenspiegel

Der Anlagenspiegel zeigt je Position Anschaffungskosten, Zugänge, Abgänge,
Umbuchungen, kumulierte Abschreibungen und Buchwerte. Er setzt eine Kartei
voraus, die über Geschäftsjahre läuft, während das Journal je Jahr organisiert
ist. Kleine Kapitalgesellschaften sind von ihm im Anhang befreit
([§ 288 Abs. 1 Nr. 1 HGB][hgb288]).

[estg6]: https://www.gesetze-im-internet.de/estg/__6.html
[estg7]: https://www.gesetze-im-internet.de/estg/__7.html
[estg7g]: https://www.gesetze-im-internet.de/estg/__7g.html
[hgb255]: https://www.gesetze-im-internet.de/hgb/__255.html
[hgb256a]: https://www.gesetze-im-internet.de/hgb/__256a.html
[hgb288]: https://www.gesetze-im-internet.de/hgb/__288.html
[invstg18]: https://www.gesetze-im-internet.de/invstg_2018/__18.html
