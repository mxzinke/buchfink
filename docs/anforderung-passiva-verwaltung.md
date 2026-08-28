# Buchfink – Passiva verwalten: Darlehen, Rückstellungen, Sonderposten

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-28
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)
Vorbild: [Anlagenverwaltung](anforderung-anlagenverwaltung.md) – umgesetzt, dieselbe Mechanik, andere Bilanzseite

> **Ausgangslage.** Die Anlagenkartei steht: `internal/domain/asset.go`,
> `internal/accounting/afa.go`, `internal/service/asset_service.go`,
> `frontend/src/pages/AssetsPage.tsx`. Damit ist auf der Aktivseite auch der
> Darlehensfall abgedeckt, den dieses Dokument als offen beschrieb – eine
> Ausleihung ist dort eine Finanzanlage, ihre Tilgung ein eigener Abgangsgrund.
> Offen ist die Passivseite. Abschnitt 10 beschreibt, wie sie an die vorhandene
> Kartei anschließt, ohne sie umzubauen.

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind am **28.08.2026** gegen den
> Gesetzestext auf gesetze-im-internet.de verifiziert; die Fundstellen stehen in
> [Abschnitt 15](#15-quellen).

## 1. Warum das dieselbe Maschine ist wie das Anlagenverzeichnis

Das Anlagenverzeichnis ist kein Aktivthema. Es ist der erste umgesetzte Fall eines
Musters, das auf der Passivseite dreimal wiederkehrt:

> Ein Bestand entsteht einmal, verändert sich planmäßig über Jahre und verschwindet
> irgendwann – und keine dieser Veränderungen hat einen Beleg.

Ein Anlagegut wird angeschafft, jährlich abgeschrieben, geht ab. Ein Darlehen wird
ausgezahlt, monatlich getilgt und verzinst, ist am Ende getilgt. Eine Rückstellung wird
gebildet, fortgeschrieben, verbraucht oder aufgelöst. Ein Sonderposten für einen
Zuschuss wird eingestellt und über die Nutzungsdauer aufgelöst.

Vier Sachverhalte, ein Ablauf: **Stammsatz anlegen, Plan rechnen, Bewegungen zur
Freigabe vorschlagen, Journalbuchung erzeugen, Rückverweis behalten.**

Für die Anlagenkartei ist dieser Ablauf gebaut und im Betrieb – mit der Bewegung als
Kern (`domain.AssetMovement` mit `JournalEntryID`, `Account` und `FiscalYear`), der
reinen Rechnung in `internal/accounting/afa.go` und dem Dienst, der beides zu Buchungen
zusammenführt. Die Frage lautet deshalb nicht mehr, ob das Datenmodell für beide
Bilanzseiten geschnitten wird – dafür ist es zu spät und die Kartei zu gut. Sie lautet:
**was davon wird geteilt, was gespiegelt, und was bleibt getrennt.**

Die Antwort steht in Abschnitt 10 und in einem Satz hier: geteilt wird der Zahlungsplan,
gespiegelt wird die Struktur, getrennt bleiben die Karteien.

## 2. Bestandsaufnahme: was auf der Passivseite liegt

Gegliedert nach § 266 Abs. 3 HGB, mit den Konten aus dem mitgelieferten SKR04 und der
Frage, woher der Saldo heute käme.

| § 266 Abs. 3 | Inhalt | SKR04 | Entsteht heute durch |
|---|---|---|---|
| **A. I–V** Eigenkapital | Gezeichnetes Kapital, Kapitalrücklage, Gewinnrücklagen, Vorträge, Jahresergebnis | **2900**, **2920**, **2930**–**2968**, **2970**/**2978** | nichts – nur manuelle Journalbuchung |
| **A.** Privatkonten (Einzelunternehmen, Vollhafter) | Entnahmen, Einlagen | **2100** ff. | Bank-Direktbuchung, wenn der Nutzer das Konto kennt |
| **A.** Kapitalanteile PersHG | Vollhafter, Kommanditisten | **2000**/**2010**, **2050**/**2060** | manuell |
| — Sonderposten (DATEV führt sie zwischen EK und Rückstellungen) | Rücklagen § 6b EStG, Sonderabschreibungen, Zuschüsse, Investitionszulagen | **2980**–**2999** | manuell |
| **B. 1** Pensionsrückstellungen | Direktzusagen, pensionsähnliche Verpflichtungen | **3000**–**3015** | manuell, Wert kommt vom Gutachter |
| **B. 2** Steuerrückstellungen | KSt, GewSt, latente Steuern | **3020**, **3035**, **3040**, **3060**, **3065** | manuell |
| **B. 3** Sonstige Rückstellungen | Gewährleistung, Urlaub, Abschlusskosten, Aufbewahrung, drohende Verluste | **3070**–**3099** | manuell |
| **C. 1** Anleihen | konvertibel / nicht konvertibel | **3100** ff. | manuell |
| **C. 2** Kreditinstitute | Darlehen, Kontokorrent, Teilzahlungsverträge | **3150**/**3151**/**3160**/**3170**, **3180** ff. | Bank-Direktbuchung, Rate nur manuell splittbar |
| **C. 3** Erhaltene Anzahlungen | versteuerte Anzahlungen, RLZ-Gliederung | **3250**, **3260**, **3272**, **3280** ff. | eigenes Dokument, [Anzahlungen](anforderung-anzahlungen.md) |
| **C. 4** Verbindlichkeiten aus LL | Kreditoren | **3300** + Personenkonten | **Belegflow ✅** |
| **C. 5** Wechsel | gezogene und eigene Wechsel | Bereich 3350–3399, im SKR04 reserviert | out of scope |
| **C. 6/7** verbundene Unternehmen, Beteiligungen | Lieferung/Leistung und Finanzierung | **3400** ff., **3450** ff. | Belegflow, aber ohne gesonderten Ausweis |
| **C. 8** sonstige Verbindlichkeiten – Gesellschafter | GmbH-Gesellschafter, phG, Kommanditisten, stille Gesellschafter | **3640**–**3643**, **3645**–**3648**, **3650**–**3653**, **3520** ff., **3530** ff. | manuell |
| **C. 8** sonstige Verbindlichkeiten – Personal | Lohn- und Kirchensteuer, soziale Sicherheit | **3730**, **3740** | manuell |
| **C. 8** sonstige Verbindlichkeiten – Umsatzsteuer | USt, USt-Vorauszahlungen | **3806**, **3801**, **3820** | **Steuerautomatik ✅** |
| **C. 8** sonstige Verbindlichkeiten – übrige | erhaltene Kautionen, offene Ausschüttungen, einbehaltene KapESt, übrige | **3550**–**3557**, **3519**, **3760**, **3500**/**3501** | Bank-Direktbuchung |
| **C. 8** Privatkonten Teilhafter (Fremdkapital) | Kommanditisten | **2500** ff. | manuell |
| **D.** Passive Rechnungsabgrenzung | PRAP | **3900** | eigenes Dokument, [Rechnungsabgrenzung](anforderung-rechnungsabgrenzung.md) |
| **E.** Passive latente Steuern | | **3065**, **3950** | manuell |

Von zwanzig Zeilen sind **zwei** automatisiert: Verbindlichkeiten aus Lieferungen und
Leistungen über den Belegflow und die Umsatzsteuer über die Steuerautomatik. Zwei
weitere haben ein eigenes Anforderungsdokument. Der Rest der Passivseite ist heute
Handarbeit im Journal.

## 3. Was davon nicht über Belege geht – und warum

Nicht alles Nicht-Automatisierte ist derselbe Fall. Es sind fünf Gruppen, und sie
brauchen fünf verschiedene Auslöser:

| Gruppe | Beispiele | Was fehlt |
|---|---|---|
| **1. Finanzierungen mit Zahlungsplan** | Bankdarlehen, Gesellschafterdarlehen, Anleihe, Teilzahlungsvertrag, Kaution | Es gibt einen Vertrag statt eines Belegs, und monatlich Geld ohne jede Rechnung. |
| **2. Bewertungsvorgänge zum Stichtag** | Rückstellungen, Abzinsung, latente Steuern, Restlaufzeit-Umgliederung | Kein Zahlungsvorgang, keine externe Unterlage – nur eine Beurteilung, die dokumentiert werden muss. |
| **3. Ergebnisverwendung** | Vortrag, Einstellung in Rücklagen, Ausschüttung | Einmal jährlich, aus einem Gesellschafterbeschluss, mit steuerlichen Nebenpflichten. |
| **4. Gesellschafterbewegungen** | Einlage, Entnahme, Kapitalerhöhung | Bankumsatz vorhanden, aber die Kontenwahl ist rechtsformabhängig und für den Nutzer nicht erratbar. |
| **5. Fremdabgerechnetes** | Lohnsteuer und Sozialversicherung aus der Lohnabrechnung | Die Zahlen kommen aus einem anderen System; es gibt einen Beleg, aber keinen, den der Belegflow versteht. |

Nur Gruppe 1 hat regelmäßig einen Bankumsatz, an den man andocken kann. Gruppe 2 und 3
hängen am Stichtag, Gruppe 4 an einem Vorgang, Gruppe 5 an einem Import.

**Die Konsequenz für die Oberfläche:** ein einziges „Passiva verwalten" gibt es nicht.
Es gibt ein Verzeichnis (Gruppe 1), einen Abschlussassistenten (Gruppen 2 und 3),
Buchungsgruppen für die Bank (Gruppe 4) und einen Import (Gruppe 5). Was sie verbindet,
ist der darunterliegende Mechanismus aus Abschnitt 4 – nicht die Maske.

## 4. Das gemeinsame Muster: Verzeichnis = Stammsatz + Plan + Bewegungen

Ein Verzeichniseintrag besteht aus drei Teilen, unabhängig davon, ob er ein Anlagegut,
ein Darlehen oder eine Rückstellung beschreibt:

| Teil | Inhalt | Anlagegut (umgesetzt) | Darlehen (offen) |
|---|---|---|---|
| **Stammsatz** | Bezeichnung, Bestandskonto, Kontakt, Beginn, Ausgangsbetrag, abgeleiteter Status | `domain.FixedAsset` | `domain.Liability` |
| **Plan** | Zeitreihe mit Fälligkeit, Betrag, Art und Kontierung | `accounting.BuildAfASchedule`, gerechnet statt gespeichert | `accounting.BuildTilgungsplan`, zusätzlich gespeichert – die Rate wird gegen einen Bankumsatz zugeordnet, die AfA nicht |
| **Bewegungen** | Ist-Buchungen mit Verweis auf den Journaleintrag | `domain.AssetMovement` mit `JournalEntryID` | `domain.LiabilityMovement`, gleiche Felder |

Der eine echte Unterschied steht in der mittleren Zeile: Ein Abschreibungsplan wird bei
Bedarf neu gerechnet, weil ihn niemand von außen anfasst. Ein Tilgungsplan muss
gespeichert werden, weil ein Bankumsatz auf **eine bestimmte Rate** trifft und weil
Zinsanpassung und Sondertilgung ihn ändern, ohne die bereits gebuchten Raten anzurühren.

Drei Arten, wie aus einer geplanten eine gebuchte Bewegung wird:

| Modus | Auslöser | Fälle |
|---|---|---|
| **zahlungsgetrieben** | Bankumsatz trifft ein und wird gegen den Plan zugeordnet | Darlehensrate, Leasingrate, Kaution, Anzahlungsrückzahlung |
| **stichtagsgetrieben** | Festschreibung bzw. Jahresabschluss | AfA, RAP-Auflösung, Zinsabgrenzung, Restlaufzeit-Umgliederung, Abzinsung, Sonderposten-Auflösung |
| **ereignisgetrieben** | Nutzer erfasst einen Vorgang | Rückstellung bilden oder verbrauchen, Einlage, Entnahme, Ergebnisverwendung |

Der stichtagsgetriebene Modus ist genau der, den die Anlagenverwaltung
([Abschnitt 6](anforderung-anlagenverwaltung.md#6-kopplung-an-die-festschreibung)) und
die Rechnungsabgrenzung
([Abschnitt 3](anforderung-rechnungsabgrenzung.md#3-was-buchfink-daraus-machen-kann))
bereits beschreiben: vor der Jahressperre prüfen, mit Vorschau anzeigen, auf Freigabe
buchen. Beide beschreiben denselben Ablauf für unterschiedliche Sachverhalte. Er gehört
einmal gebaut und dreimal benutzt.

## 5. Der Plan ist der Beleg

Das ist der Kern der Antwort auf „ohne manuell zu buchen".

Buchfink erzeugt Buchungen heute aus zwei Quellen: aus einem Beleg (Abschnitt 10 des
Buchungsflows) und aus der Zuordnung eines Bankumsatzes zu einem offenen Posten
(`PaymentService.OpenItems`, `internal/service/payment_service.go:113`). Offene Posten
entstehen dort ausschließlich aus Journaleinträgen mit Personenkonto und Kontakt –
für eine Darlehensrate gibt es beides nicht. Übrig bleibt die Direktbuchung
(`BankService.BookDirect`, `internal/service/bank_service.go:70`), und die kann
konstruktionsbedingt nur **zwei Zeilen**: Bank gegen ein Gegenkonto. Eine Rate aus
Tilgung, Zins und Gebühr passt da nicht hinein. Genau deshalb ist die Rate heute
Handarbeit.

Die Lösung ist keine dritte Buchungsmaske, sondern eine **zweite Quelle für erwartete
Zahlungen**:

1. Der Tilgungsplan erzeugt je Fälligkeit eine **erwartete Zahlung** mit fertiger
   Kontierung – Betrag, Datum, Soll- und Habenzeilen.
2. Der Bankimport bietet sie in derselben Zuordnungsansicht an wie einen offenen Posten
   (`frontend/src/pages/BankImportPage.tsx:360`, Modus `open_item` – „Offenen Posten
   ausgleichen").
3. Zuordnen heißt buchen: der Plan liefert die Zeilen, der Kontoauszug den Betrag und
   das Datum. Der Nutzer bestätigt, er kontiert nicht.
4. Weicht der Betrag ab – variable Zinsen, Sondertilgung, Gebühr –, wird die Differenz
   nach denselben Regeln behandelt wie eine Zahlungsdifferenz beim offenen Posten:
   Vorschlag, Begründung, Freigabe.

Der Plan übernimmt damit exakt die Rolle, die sonst der Beleg spielt: Er sagt, was das
Geld war. **Das ist die einzige Stelle, an der die Passivseite in den bestehenden Flow
eingebunden werden muss** – alles andere sind Auswertungen und Abschlussvorgänge.

Nebeneffekt: Die Zwei-Zeilen-Grenze der Direktbuchung fällt für alle geplanten
Vorgänge, ohne dass `BookDirect` angefasst wird. Die Mehrzeilenbuchung entsteht aus dem
Plan, nicht aus einer Eingabe.

## 6. Darlehen und Finanzierungen

### 6.1 Anlegen

Erfasst wird der Vertrag, nicht die Buchung: Bezeichnung, Kreditgeber (Kontakt),
Auszahlungsbetrag, Nennbetrag, Auszahlungsdatum, Zinssatz, Zinsbindung, Laufzeit,
Tilgungsart (Annuität, Rate, endfällig), Rhythmus, Sicherheiten, Sondertilgungsrecht.

Aus dem Kreditgeber-Typ folgt das Bestandskonto – der Nutzer wählt es nicht:

| Kreditgeber | Konto | Bilanzposten |
|---|---|---|
| Kreditinstitut | **3150** / **3151** / **3160** / **3170** nach Restlaufzeit | § 266 Abs. 3 C.2 |
| Teilzahlungsvertrag mit Kreditinstitut | **3180** / **3181** / **3190** / **3200** | § 266 Abs. 3 C.2 |
| Anleihe | **3100** ff. | § 266 Abs. 3 C.1 |
| sonstiger Dritter (Privatdarlehen) | **3560** / **3561** / **3564** / **3567** | § 266 Abs. 3 C.8 |
| partiarisches Darlehen | **3540** / **3541** / **3544** / **3547** | § 266 Abs. 3 C.8 |
| GmbH-Gesellschafter | **3640** / **3641** / **3642** / **3643** | § 266 Abs. 3 C.8 |
| persönlich haftender Gesellschafter | **3645**–**3648**, bei PersHG zusätzlich **2020** | C.8 bzw. § 264c Abs. 2 |
| Kommanditist | **3650**–**3653**, bei PersHG zusätzlich **2070** | C.8 bzw. § 264c Abs. 2 |
| typisch stiller Gesellschafter | **3520**–**3527** | § 266 Abs. 3 C.8 |
| atypisch stiller Gesellschafter | **3530**–**3537** | § 266 Abs. 3 C.8 |

### 6.2 Tilgungsplan

Gerechnet wird beim Anlegen, nicht beim Buchen. Annuität, Rate oder endfällig, monatlich
bis jährlich, auf ganze Cent, mit Ausgleich der Rundungsdifferenz in der Schlussrate.

Der Plan ist ein **Vorschlagswerk, kein Dogma**: Zinsanpassung, Sondertilgung und
Stundung müssen ihn nachträglich ändern können, ohne bereits gebuchte Bewegungen
anzutasten. Ein neu gerechneter Plan ersetzt nur die Zukunft.

### 6.3 Buchungen

| Vorgang | Buchung |
|---|---|
| Auszahlung | SOLL **1800** Bank · HABEN **3160** Verb. ggü. Kreditinstituten |
| Rate 1.000 € (800 Tilgung, 200 Zins) | SOLL **3160** 800,00 + SOLL **7320** Zinsaufwendungen für langfristige Verbindlichkeiten 200,00 · HABEN **1800** Bank 1.000,00 |
| Rate mit Bearbeitungsgebühr | zusätzlich SOLL **6855** Nebenkosten des Geldverkehrs |
| Kontokorrentzinsen | SOLL **7318** Zinsen auf Kontokorrentkonten · HABEN **1800** |
| Zinsen an Gesellschafter | SOLL **7316** Zinsen für Gesellschafterdarlehen, bei Beteiligung über 25 % **7317** |

Zinsaufwand ist kein Umsatz: Kreditgewährung ist nach § 4 Nr. 8 Buchst. a UStG
steuerfrei. Es gibt keine Vorsteuer und keinen Steuerfall zu wählen.

### 6.4 Restlaufzeit ist kein Stammdatum, sondern ein Stichtagswert

§ 268 Abs. 5 Satz 1 HGB verlangt den Vermerk der Beträge mit einer Restlaufzeit bis zu
einem Jahr und über einem Jahr **bei jedem gesondert ausgewiesenen Posten**. Der SKR04
löst das über getrennte Konten – 3151 (bis 1 Jahr), 3160 (1 bis 5 Jahre), 3170 (über
5 Jahre).

Damit wandert ein Darlehen im Zeitverlauf von Konto zu Konto, und zwar zu jedem
Stichtag neu. Von Hand macht das niemand richtig. Aus dem Tilgungsplan dagegen ist es
eine Rechnung: Wie viel des Restsaldos wird in den nächsten zwölf Monaten fällig, wie
viel danach? Die Umgliederung ist eine stichtagsgetriebene Bewegung wie die AfA:

| Buchung zum 31.12. |
|---|
| SOLL **3160** (1 bis 5 Jahre) · HABEN **3151** (bis 1 Jahr) – in Höhe der in 2027 fälligen Tilgungen |

**Der Anhang braucht dieselbe Rechnung ein Jahr weiter:** § 285 Nr. 1 HGB verlangt den
Gesamtbetrag der Verbindlichkeiten mit einer Restlaufzeit von mehr als fünf Jahren und
den Gesamtbetrag der durch Pfandrechte oder ähnliche Rechte gesicherten
Verbindlichkeiten unter Angabe von Art und Form der Sicherheiten. Beides fällt aus dem
Verzeichnis ab, wenn die Sicherheit im Stammsatz steht – und ist sonst nicht
rekonstruierbar.

### 6.5 Disagio

Wird ein Darlehen unter Nennwert ausgezahlt, ist die Verbindlichkeit mit dem
Erfüllungsbetrag anzusetzen (§ 253 Abs. 1 Satz 2 HGB). Für den Unterschiedsbetrag
gewährt § 250 Abs. 3 Satz 1 HGB ein **Wahlrecht**: Er darf in den aktiven
Rechnungsabgrenzungsposten aufgenommen werden – Konto **1940** Damnum/Disagio, sonst
sofort Aufwand.

| Auszahlung 50.000 € Nennwert, 48.500 € Zufluss, Disagio aktiviert |
|---|
| SOLL **1800** Bank 48.500,00 + SOLL **1940** Damnum/Disagio 1.500,00 · HABEN **3160** 50.000,00 |

Die Verteilung über die Laufzeit ist wieder ein Plan – derselbe Mechanismus wie die
RAP-Auflösung, nur mit der Darlehenslaufzeit als Zeitraum. Das Wahlrecht gehört an den
Vertrag, nicht in die Stammdaten: Es kann je Darlehen anders ausgeübt werden.

### 6.6 Zinsabgrenzung

Zinsen für Dezember, die im Januar abgebucht werden, sind Aufwand des alten Jahres. Der
Plan kennt die Fälligkeiten und den Zinsanteil, kann die Abgrenzung also selbst
errechnen – SOLL **7320** · HABEN **3500** Sonstige Verbindlichkeiten – und im Folgejahr
auflösen. Ohne Verzeichnis fällt dieser Posten schlicht aus; er hat keinen Beleg, der an
ihn erinnert.

### 6.7 Gesellschafterdarlehen brauchen eine Zusatzangabe

Für die GmbH verlangt § 42 Abs. 3 GmbHG, Ausleihungen, Forderungen und
Verbindlichkeiten gegenüber Gesellschaftern jeweils gesondert auszuweisen oder im Anhang
anzugeben; werden sie unter anderen Posten ausgewiesen, muss diese Eigenschaft vermerkt
werden. § 264c Abs. 1 HGB sagt dasselbe für Personenhandelsgesellschaften.

Praktisch heißt das: **Das Merkmal „Gesellschafter" gehört an den Kontakt**, nicht an
die Buchung. Steht es dort, kann das Verzeichnis das richtige Konto wählen, und die
Auswertung kann die Angabe erzeugen. Steht es nicht dort, ist die Angabe später nicht
mehr herstellbar, ohne jeden Kreditgeber einzeln zu beurteilen.

Bei Personenhandelsgesellschaften kommt hinzu, dass das Gesellschafter-Darlehen nach
Kontenklasse 2 gehört (**2020** Vollhafter, **2070** Kommanditisten) und damit
rechtsformabhängig ein anderes Konto trägt als bei der GmbH. Die Rechtsform steht in den
Stammdaten – die Ableitung gehört ins Backend, nicht in die Auswahl des Nutzers.

### 6.8 Der Spiegelfall ist bereits gebaut

Ein Darlehen, das das Unternehmen **vergibt**, liegt in der Anlagenkartei: eine
Ausleihung ist eine Finanzanlage, und die Kartei kennt sie.

| Fall | Konto | Stand |
|---|---|---|
| dauerhaft überlassen | **0940** Darlehen, **0930** übrige sonstige Ausleihungen | im Katalog (`internal/accounting/asset_accounts.go`) |
| an verbundene Unternehmen / Beteiligungen | **0810** / **0880** | im Katalog |
| an Gesellschafter | **0960**–**0963** | **fehlt im Katalog** – gerade der Fall, den § 42 Abs. 3 GmbHG gesondert ausgewiesen sehen will |
| Tilgung | `DisposalRepayment` – Geld gegen Ausleihung, kein Erlös, kein Buchgewinn | umgesetzt |
| Teiltilgung | Teilabgang mit Stückzahl bzw. Betrag | umgesetzt |
| Fälligkeit | `MaturityDate` am Anlagegut, für § 256a Satz 2 HGB | umgesetzt |
| Zinsertrag | **7011** Erträge aus Ausleihungen, sonst **7100** | umgesetzt (`AssetIncomeAccount`) |

Zwei Lücken bleiben auf dieser Seite, und beide gehören zu diesem Dokument, nicht zur
Anlagenverwaltung:

1. **Die Ausleihung hat keinen Zahlungsplan.** Die Tilgung wird gebucht, wenn sie
   kommt; dass sie kommen *wird*, weiß niemand. Genau das leistet der Plan aus
   Abschnitt 5 – und er leistet es für beide Seiten. Deshalb hängt er nicht an der
   Verbindlichkeit, sondern steht neben beiden Karteien.
2. **Die Ausleihungen an Gesellschafter fehlen im Kontenkatalog.** Vier Zeilen in
   `assetAccounts`, mehr ist es nicht – aber ohne sie muss der Nutzer für genau den
   Fall, der eine Anhangangabe auslöst, auf **0930** ausweichen.

## 7. Rückstellungen

Rückstellungen sind der Gegenentwurf zum Darlehen: kein Zahlungsplan, dafür eine
Beurteilung, die dokumentiert werden muss.

Zu bilden sind sie nach § 249 Abs. 1 HGB für ungewisse Verbindlichkeiten und drohende
Verluste aus schwebenden Geschäften, für in den ersten drei Monaten des Folgejahres
nachgeholte Instandhaltung, für im Folgejahr nachgeholte Abraumbeseitigung und für
Gewährleistungen ohne rechtliche Verpflichtung. Aufgelöst werden dürfen sie nach § 249
Abs. 2 Satz 2 HGB nur, **soweit der Grund hierfür entfallen ist** – nicht, weil das
Ergebnis es gerade braucht.

| Vorgang | Buchung |
|---|---|
| Bildung Gewährleistung | SOLL **6790** Aufwand für Gewährleistung · HABEN **3090** Rückstellungen für Gewährleistungen |
| Bildung Abschlusskosten | SOLL **6827** Abschluss- und Prüfungskosten · HABEN **3095** |
| Verbrauch (Rechnung kommt) | SOLL **3090** · HABEN Kreditorenkonto |
| Auflösung (Grund entfallen) | SOLL **3090** · HABEN **4930** Erträge aus der Auflösung von Rückstellungen |
| Aufzinsung langfristiger Rückstellungen | SOLL **7362** Zinsaufwendungen aus der Abzinsung von Rückstellungen · HABEN Rückstellungskonto |

Rückstellungen mit einer Restlaufzeit von mehr als einem Jahr sind nach § 253 Abs. 2
Satz 1 HGB abzuzinsen – mit dem laufzeitentsprechenden durchschnittlichen Marktzinssatz
der vergangenen sieben Geschäftsjahre, bei Altersversorgungsverpflichtungen der
vergangenen zehn. Die Zinssätze veröffentlicht die Bundesbank monatlich; sie gehören in
die Stammdaten je Geschäftsjahr, wie die AfA-Wertgrenzen auch. Für
Altersversorgungsverpflichtungen erlaubt Satz 2 pauschal den Satz bei angenommener
Restlaufzeit von 15 Jahren.

Was das Verzeichnis dafür führen muss: Sachverhalt, Grund der Bildung, Betrag, erwarteter
Fälligkeitszeitpunkt, Bewertungsmethode, Historie der Zuführungen und Verbräuche. **Der
Grund ist der wichtigste Teil** – ohne ihn ist ein Jahr später weder die Auflösung noch
die Beibehaltung begründbar, und die Betriebsprüfung fragt genau danach.

Pensionsrückstellungen (**3000**–**3015**) sind ein Sonderfall: Der Wert kommt aus einem
versicherungsmathematischen Gutachten. Buchfink kann ihn erfassen und fortschreiben,
aber nicht errechnen. Das ist kein Mangel, sondern die richtige Grenze.

## 8. Eigenkapital, Ergebnisverwendung, Privatkonten

Diese Vorgänge kommen einmal jährlich und hängen an einem Beschluss, nicht an einem
Zahlungsvorgang. Sie gehören in einen **Jahresabschluss-Assistenten**, nicht in ein
Verzeichnis:

| Vorgang | Buchung |
|---|---|
| Ergebnisvortrag | Jahresergebnis auf **2970** Gewinnvortrag bzw. **2978** Verlustvortrag |
| Einstellung in Gewinnrücklagen | SOLL **7765** / **7780** Einstellungen in die gesetzliche bzw. andere Gewinnrücklagen · HABEN **2930** / **2960** |
| Entnahme aus Rücklagen | SOLL **2930** / **2950** · HABEN **7735** / **7745** |
| Beschlossene Ausschüttung | SOLL Gewinnvortrag · HABEN **3519** Verbindlichkeiten gegenüber Gesellschaftern für offene Ausschüttungen |
| Kapitalertragsteuer auf die Ausschüttung | HABEN **3760** Verbindlichkeiten aus Einbehaltungen (KapESt, SolZ) |
| Vorabausschüttung | **7790** |

Die einbehaltene Kapitalertragsteuer ist der Punkt, an dem aus einem Beschluss eine
Frist wird: Sie ist anzumelden und abzuführen. Buchfink hat mit der Fristenansicht
(`frontend/src/pages/DeadlinesPage.tsx`) bereits den Ort dafür.

**Privatentnahmen und -einlagen** (Einzelunternehmen, Personengesellschaften) sind der
einzige Fall dieser Gruppe mit Bankumsatz. Sie brauchen kein Verzeichnis, sondern zwei
Buchungsgruppen im Katalog (`internal/accounting/posting_groups.go:85`), die auf **2100**
ff. bzw. bei Kommanditisten auf **2500** ff. zeigen – rechtsformabhängig abgeleitet, wie
in 6.7. Heute muss der Nutzer diese Kontonummern kennen; das ist die eigentliche Hürde,
nicht die Buchung.

## 9. Sonderposten und Zuschüsse – die Naht zum Anlagenverzeichnis

Ein Investitionszuschuss trifft beide Seiten gleichzeitig. Für die handelsrechtliche
Behandlung gibt es zwei Wege, und sie schließen einander aus:

| Weg | Wirkung | Konten |
|---|---|---|
| Minderung der Anschaffungskosten | AfA-Bemessungsgrundlage sinkt | Anlagekonto |
| Passivierung als Sonderposten | Sonderposten wird parallel zur AfA ertragswirksam aufgelöst | **2998** Sonderposten für Zuschüsse Dritter, **2999** Investitionszulagen |

Der zweite Weg ist genau der Fall, für den sich das Verzeichnis lohnt: Der Sonderposten
bekommt einen Auflösungsplan mit derselben Laufzeit wie die AfA des Anlageguts, und
beide müssen aneinander gekoppelt bleiben – wird das Anlagegut vorzeitig verkauft, ist
der Restsonderposten in einem Zug aufzulösen. Dasselbe Muster gilt für die steuerlichen
Rücklagen: **2990** (Sonderabschreibungen), Einstellungen über **6922** / **6924** /
**6927**, Auflösung über **4927** / **4928** / **4935** / **4937**.

**Daraus folgt die Form des Sonderpostens:** ein eigener Stammsatz mit einem Verweis
auf das Anlagegut, dessen Abschreibung er begleitet – nicht ein weiteres Feld an
`FixedAsset`. Der Verweis zeigt von der Passivseite auf die Aktivseite und nicht
umgekehrt, weil der Sonderposten ohne das Anlagegut sinnlos ist, das Anlagegut ohne ihn
aber vollständig.

## 10. Wie die Umsetzung anzugehen ist

### 10.1 Die Vorentscheidung: spiegeln, nicht nachträglich generalisieren

Die naheliegende Idee wäre, aus `FixedAsset` und `AssetMovement` ein allgemeines
Verzeichnis herauszuziehen, das beide Bilanzseiten trägt. Sie ist falsch, und zwar aus
drei Gründen:

1. **Der Umbau fasst laufenden, getesteten Code an, ohne dass ein Nutzer etwas davon
   hat.** Die Kartei ist rund 9.000 Zeilen mit eigenen Tests; jede Woche, die in ihre
   Verallgemeinerung geht, bringt kein Darlehen näher.
2. **Die beiden Sachverhalte sind sich ähnlicher, als sie gleich sind.** Ein Anlagegut
   nutzt sich ab, eine Verbindlichkeit nicht. Ein Anlagegut hat eine Nutzungsdauer,
   eine Verbindlichkeit eine Restlaufzeit – die eine ist geschätzt und ändert die AfA,
   die andere ist vertraglich und ändert den Bilanzausweis. Der Anlagenspiegel hat
   Spalten, die der Verbindlichkeitenspiegel nicht kennt, und umgekehrt. Ein
   gemeinsamer Typ müsste alles davon als „optional" führen und wäre für beide Seiten
   schlechter als zwei klare.
3. **Doppelter Code, den man später zusammenlegt, ist billiger als eine Abstraktion,
   die zwei ungleiche Sachverhalte in eine Form zwingt.** Umgekehrt gilt das nicht.

Gespiegelt wird deshalb die **Struktur**, nicht der Typ: gleiche Dateilage, gleiche
Namensmuster, gleiche Art von Nahtstelle. Wer die Anlagenkartei gelesen hat, findet
sich in der Verbindlichkeitenkartei sofort zurecht.

| Anlagen (vorhanden) | Verbindlichkeiten (neu) | Inhalt |
|---|---|---|
| `internal/domain/asset.go` | `internal/domain/liability.go` | Stammsatz, Bewegung, Repository-Schnittstelle |
| `internal/repository/asset_gorm.go` | `internal/repository/liability_gorm.go` | Persistenz, Eintrag in `AutoMigrate` |
| `internal/accounting/afa.go` | `internal/accounting/tilgung.go` | die reine Rechnung, ohne Datenbank und ohne Buchung |
| `internal/accounting/asset_accounts.go` | `internal/accounting/liability_accounts.go` | Kontenkatalog und Kontenableitung |
| `internal/service/asset_service.go` | `internal/service/liability_service.go` | Kartei, Vorschau, Buchung, Auswertung |
| `internal/wailsbridge/asset_service.go` | `internal/wailsbridge/liability_service.go` | Endpunkte mit `nil`-Wache |
| `frontend/src/pages/AssetsPage.tsx` | `frontend/src/pages/LiabilitiesPage.tsx` | Oberfläche |

Geteilt wird genau ein Stück – der **Zahlungsplan** aus Stufe 2. Er ist der einzige
Baustein, für den es zwei echte Nutzer gibt: die Rate eines Darlehens und die
erwartete Tilgung einer Ausleihung (6.8). Ein Baustein mit einem Nutzer wäre
Spekulation; dieser hat zwei, bevor er geschrieben ist.

### 10.2 Was ausdrücklich nicht getan wird

- **`FixedAsset` nicht um Verbindlichkeiten erweitern.** Ein Darlehen wäre dort ein
  Anlagegut mit negativem Vorzeichen, ohne Abschreibungsplan, das nicht in den
  Anlagenspiegel darf. Jede Abfrage der Kartei müsste es danach ausschließen.
- **`BookDirect` nicht auf n Zeilen erweitern.** Die Zwei-Zeilen-Grenze ist dort
  richtig: eine Buchung ohne Beleg soll klein bleiben. Die Mehrzeilenbuchung kommt aus
  dem Plan.
- **Keine neue Buchungsmaske.** Wo der Nutzer Soll und Haben tippt, hat die Software
  ihre Arbeit nicht getan.
- **Kein zweiter Belegkreis für Verträge.** Kreditverträge gehen in dieselbe
  Dokumentenablage wie die Papiere am Anlagegut (`internal/receiptstore`, Zweig
  `dokumente/`).

### 10.3 Stufe 1 – Die Kartei für Verbindlichkeiten

Das kleinste, das für sich Sinn ergibt: Darlehen anlegen, Plan sehen, Auszahlung buchen.

**Modell** (`internal/domain/liability.go`), gebaut wie `asset.go`:

- `Liability`: Bezeichnung, Vertragsnummer, Art (Bankdarlehen, Anleihe, Privatdarlehen,
  Gesellschafterdarlehen, stille Beteiligung, Kaution), Bestandskonto, Kontakt,
  Auszahlungsdatum, Nennbetrag, Auszahlungsbetrag, Zinssatz in Basispunkten, Zinsbindung,
  Laufzeit, Tilgungsart, Rhythmus, Sicherheit (Art und Form – § 285 Nr. 1 HGB),
  Disagio-Behandlung, Status.
- `LiabilityMovement` mit denselben Feldern, die sich an `AssetMovement` bewährt haben:
  `Date`, `FiscalYear`, `Account` (das Konto *dieser* Bewegung, nicht das aktuelle des
  Stammsatzes – sonst verschiebt eine Restlaufzeit-Umgliederung rückwirkend die
  Vorjahre des Spiegels), `JournalEntryID`, `Note`.
- Bewegungsarten, bewusst geschlossen wie `AssetMovementKind`: `disbursement`,
  `repayment`, `interest`, `fee`, `accrual` (Zinsabgrenzung), `reclassification`
  (Restlaufzeit), `disagio_release`, `waiver` (Erlass), `final`.
- Ein `Status`, der wie beim Anlagegut abgeleitet und nie gespeichert wird – inklusive
  `unbooked` für den **Bestandsfall**: ein Darlehen, das schon läuft, wird mit
  Restsaldo erfasst, ohne eine Auszahlungsbuchung zu erfinden. Das ist der Normalfall
  bei der Einführung und darf nicht der Sonderfall im Code sein.

**Rechnung** (`internal/accounting/tilgung.go`): eine reine Funktion
`BuildTilgungsplan(plan) ([]TilgungsRate, error)` nach dem Vorbild von
`BuildAfASchedule` – Annuität, Rate, endfällig; monatlich bis jährlich; auf ganze Cent
mit Ausgleich in der Schlussrate. Kein Datenbankzugriff, keine Buchung, damit sie so
testbar ist wie `afa_test.go` es vormacht.

**Kontenableitung** (`internal/accounting/liability_accounts.go`): Katalog nach dem
Muster von `assetAccounts`, mit der Ableitung Art × Rechtsform × Restlaufzeit → Konto
(Tabelle in 6.1) und den Zinskonten. Die Rechtsform steht seit
`domain.LegalFormCatalog` als auswertbares Feld bereit – die Ableitung für
Gesellschafterdarlehen (2020/2070 gegenüber 3640 ff.) kann sie direkt lesen. Ein Test
nach dem Vorbild von `TestAssetAccountsExistInSKR04` hält den Katalog am SKR04 fest.

**Dienst** (`internal/service/liability_service.go`): `List`, `Get`, `Save`,
`PreviewSchedule`, `BookDisbursement`, `Delete`. Gebucht wird über `JournalService.Post`
wie überall, mit Rückverweis in beide Richtungen.

**Oberfläche**: `LiabilitiesPage.tsx`, Sidebar-Eintrag „Finanzierungen" in der Gruppe
„Buchhaltung" neben „Anlagevermögen" (`frontend/src/components/Sidebar.tsx:70`).
Verdrahtet wird der Dienst in `internal/wailsbridge/app_service.go:227`, wo auch die
Anlagenkartei entsteht.

### 10.4 Stufe 2 – Der Zahlungsplan und die erwartete Zahlung

Erst hier entsteht der Nutzen, um den es geht: die Rate ohne Handarbeit.

- `internal/domain/schedule.go`: `ScheduledPayment` mit Fälligkeit, Betrag,
  Buchungszeilen und einem Besitzer aus zwei Feldern (`OwnerKind` = `liability` oder
  `asset`, `OwnerID`). Der zweite Wert ist kein Vorrat für die Zukunft, sondern der
  Fall aus 6.8, der heute schon existiert.
- Nahtstelle im Zahlungsflow nach dem Vorbild von `AssetRegister`
  (`internal/service/payment_service.go:53`) – zwei Methoden, optional verdrahtet, ohne
  sie bucht alles wie bisher:

  ```go
  type ScheduleRegister interface {
      DuePayments(ctx context.Context, until string) ([]ScheduledPayment, error)
      SettleScheduled(ctx context.Context, req ScheduledSettlement) (*domain.JournalEntry, error)
  }
  ```

- Im Bankimport erscheinen fällige Raten in derselben Liste wie die offenen Posten
  (`frontend/src/pages/BankImportPage.tsx:360`). Betragsabweichungen laufen durch
  dieselbe Differenzbehandlung wie eine Zahlungsdifferenz.

Prüfstein für diese Stufe: Eine Rate über 1.000 € wird zugeordnet und erzeugt eine
dreizeilige Buchung (Tilgung, Zins, Bank), ohne dass der Nutzer ein Konto gesehen hat.

### 10.5 Stufe 3 – Der Stichtagslauf

Die Bewegungen, die kein Bankumsatz auslöst. Vorbild ist der Abschreibungslauf mit
`PendingDepreciation` und der Sperre in `ensureDepreciationBooked`
(`internal/wailsbridge/festschreibung_service.go`):

- `PendingLiabilityBookings(ctx, fiscalYear)` liefert, was zum Stichtag fehlt:
  Zinsabgrenzung, Restlaufzeit-Umgliederung, Disagio-Auflösung.
- Der Festschreibungs-Hook prüft künftig beide Karteien. Er heißt dann nicht mehr
  `ensureDepreciationBooked`, sondern trägt den Namen des Vorgangs, den er absichert –
  der Jahresabschluss, nicht die Abschreibung.
- Die Restlaufzeit-Umgliederung ist die eine Stelle, an der die Kartei ihr eigenes
  Bestandskonto ändert. Das Muster dafür steht in `AssetService.Transfer`: eine
  Bewegung ab dem alten Konto, eine auf das neue, jede mit ihrem eigenen `Account`.

### 10.6 Stufe 4 – Rückstellungen

Eigene, kleinere Kartei ohne Zahlungsplan: Bildung, Zuführung, Verbrauch, Auflösung,
Abzinsung. Der Grund der Bildung ist Pflichtfeld, wie die Begründung der
außerplanmäßigen Abschreibung es schon ist. Sie hängt am Stichtagslauf aus Stufe 3, an
sonst nichts – und lässt sich deshalb unabhängig von den Stufen 1 und 2 bauen, wenn
das Darlehen warten muss.

### 10.7 Stufe 5 – Auswertungen und E-Bilanz

Verbindlichkeitenspiegel als Auswertung über die Bewegungen, angeboten über eine
Schnittstelle mit einer Methode – genau wie `AnlagenspiegelSource`
(`internal/service/ebilanz_service.go:15`). Dazu die Restlaufzeitangabe nach
§ 268 Abs. 5 Satz 1 HGB und die Zuordnung der Passivkonten in `skr04ToXBRL`.

### 10.8 Stufe 6 – Eigenkapital, Privatkonten, Lohn

Unabhängig von allem Vorherigen und einzeln lieferbar: die zwei Buchungsgruppen für
Privatentnahme und -einlage (`internal/accounting/posting_groups.go`), der
Ergebnisverwendungs-Assistent aus Abschnitt 8 und der Import der Lohnabrechnung.

### 10.9 Reihenfolge und Prüfsteine

| Stufe | Ergebnis für den Nutzer | Prüfstein |
|---|---|---|
| 1 | Darlehen anlegen, Plan sehen, Auszahlung buchen | Bestandsdarlehen ohne Auszahlungsbuchung erfassbar |
| 2 | Rate per Zuordnung, mehrzeilig, ohne Kontenwahl | dreizeilige Buchung aus einem Klick |
| 3 | Jahresabschluss vollständig | Festschreibung verweigert das Jahr mit offener Umgliederung |
| 4 | Rückstellungen geführt statt getippt | Auflösung nur mit Grund (§ 249 Abs. 2 Satz 2 HGB) |
| 5 | Spiegel, Restlaufzeit, E-Bilanz | Verbindlichkeitenspiegel summiert sich auf den Bilanzausweis |
| 6 | Eigenkapital und Privatkonten | Entnahme ohne Kontonummer buchbar |

Jede Stufe ist für sich lieferbar. Wer nach Stufe 2 aufhört, hat den Fall gelöst, der
heute die meiste Handarbeit macht.

## 11. Was die Auswertungen daraus brauchen

- **Restlaufzeitvermerk** nach § 268 Abs. 5 Satz 1 HGB je Bilanzposten – fällt aus den
  Plänen ab, sobald sie existieren.
- **Verbindlichkeitenspiegel** als Gegenstück zum Anlagenspiegel: Stand am Jahresanfang,
  Zugänge, Tilgungen, Stand am Jahresende, gegliedert nach Restlaufzeit. Keine Buchung,
  eine Auswertung – aber eine, die eine jahresübergreifende Kartei voraussetzt.
- **Anhangangaben** nach § 285 Nr. 1 HGB (über fünf Jahre, gesicherte Verbindlichkeiten
  mit Art und Form der Sicherheit) und die gesonderte Angabe für Gesellschafter nach
  § 42 Abs. 3 GmbHG bzw. § 264c Abs. 1 HGB.
- **Bilanzgliederung nach Positionen:** `ReportsPage.tsx:104` gruppiert heute flach nach
  Aktiva und Passiva über `type` und `kontenklasse`. Die Felder `hgbCode` und
  `positionId` stehen an jedem Konto (`internal/domain/account.go:30`), werden aber nicht
  ausgewertet. Ohne sie ist ein Restlaufzeitvermerk nicht darstellbar.
- **E-Bilanz:** die Mapping-Tabelle `skr04ToXBRL` (`internal/ebilanz/ebilanz.go`) führt
  inzwischen 72 Zuordnungen – aber nur sieben davon auf der Passivseite: 2000, 2100,
  2900, 3300, 3801, 3806 und 3820. Mit dem Anlagenkatalog ist die Aktivseite
  nachgezogen worden, die Passivseite steht noch da, wo sie war: alles Übrige fällt auf
  `de-gaap-ci:bs.other`. Ein Verzeichnis, dessen Ergebnis in der Sammelposition landet,
  hilft beim Abschluss nicht. Auch die drei Eigenkapitalzuordnungen sind zu prüfen:
  **2900** Gezeichnetes Kapital zeigt auf `bs.eqLiab.equity.retainedEarn`
  (Gewinnrücklagen), **2000** Festkapital auf `subscribedCap`.

## 12. Befunde am mitgelieferten Kontenrahmen

Beim Durchzählen der Passivseite in `skr04_2026.json` sind vier Dinge aufgefallen, die
vor der Umsetzung geklärt sein sollten – sie betreffen die Metadaten, nicht die Konten:

1. **Die Buchstaben in `hgb_code` folgen nicht § 266 Abs. 3 HGB.** Der Katalog führt
   Rückstellungen als `Passiva.C`, Verbindlichkeiten als `Passiva.D`, den
   Rechnungsabgrenzungsposten als `Passiva.E` und passive latente Steuern als
   `Passiva.F`. Im Gesetz sind das B, C, D und E; die Verschiebung entsteht durch den
   DATEV-Block „Sonderposten" als `Passiva.B`. Die Nummerierung innerhalb der Gruppen
   stimmt (`Passiva.D.2` = Kreditinstitute = § 266 Abs. 3 C.2). Wer aus `hgb_code` eine
   Bilanzgliederung baut, braucht eine Übersetzungstabelle.
2. **Konto 2900 „Gezeichnetes Kapital" trägt die Kategorie „Sonderposten mit
   Rücklageanteil"**, während die Sonderposten **2980**–**2999** unter „Eigenkapital /
   Jahresüberschuss" (`Passiva.A.V`) stehen. Die Gruppenüberschriften sind gegenüber den
   Konten um einen Block verschoben. `category` und `subcategory` taugen damit nicht als
   Grundlage für den Bilanzausweis; `position_id` ist die verlässlichere Angabe.
3. **Die Zinsaufwandskonten hängen an Positionen mit falschem Namen:** 7300, 7310, 7316
   und 7320 verweisen auf `guv.guv_12.abschreibungen_auf_finanzanlagen_und_auf` bzw.
   `..._und_wer`. Der HGB-Code stimmt, der aus dem PDF übernommene Positionsname nicht.
   In einer nach Positionen gegliederten GuV stünde der Zinsaufwand unter
   „Abschreibungen auf Finanzanlagen".
4. **Konto 3950** trägt die Position „Passive latente Steuern" (`Passiva.F`), heißt aber
   „Abgrenzung unterjährig pauschal gebuchter Abschreibungen"; die eigentlichen passiven
   latenten Steuern liegen auf **3065** unter `Passiva.C.2`.

## 13. Offene Entscheidungen

- **Der Sonderposten braucht ein Zuhause.** Er gehört fachlich zur Passivseite, hängt
  aber am Anlagegut, dessen Abschreibung er begleitet. Vorschlag: eigener Stammsatz in
  der Verbindlichkeitenkartei mit Verweis auf das Anlagegut, weil sein Auflösungsplan
  dessen Nutzungsdauer folgt – nicht als weiteres Feld an `FixedAsset`.
- **Wann wird zusammengelegt?** Nach Stufe 4 stehen zwei Karteien nebeneinander. Dann,
  und erst dann, lohnt der Blick, was wirklich doppelt ist – der Verdacht liegt auf der
  Spiegel-Auswertung und der Ableitung des Status, nicht auf den Stammsätzen.
- **Wie weit trägt der Plan?** Bei variablem Zins ist er ab der ersten Anpassung falsch.
  Vorschlag: Plan bis zur nächsten Zinsbindung rechnen, danach neu.
- **Sondertilgung:** Plan neu rechnen oder Restlaufzeit verkürzen? Beides ist üblich; die
  Wahl gehört an den Vorgang, nicht in die Stammdaten.
- **Disagio-Wahlrecht** (§ 250 Abs. 3 HGB): je Darlehen oder einmal je Mandant?
  Vorschlag: je Darlehen, vorbelegt aus den Stammdaten.
- **Abzinsungssätze** nach § 253 Abs. 2 HGB: mitliefern und pflegen, oder vom Nutzer
  eintragen lassen? Sie ändern sich monatlich.
- **Pensionsrückstellungen:** nur erfassen und fortschreiben, oder in v1 ganz
  zurückstellen?
- **Erwartete Zahlungen im Bankimport:** in dieselbe Liste wie die offenen Posten, oder
  in einen eigenen Abschnitt? Fachlich sind es verschiedene Dinge, für den Nutzer ist es
  dieselbe Frage: „Was war dieser Umsatz?"
- **§ 288 Abs. 1 HGB** enthält Erleichterungen für kleine Kapitalgesellschaften bei den
  Anhangangaben. Welche Nummern des § 285 HGB davon erfasst sind, ist hier **nicht
  verifiziert** und vor der Umsetzung der Anhangangaben aus 6.4 zu prüfen.

## 14. Abhängigkeiten

- Die **Anlagenkartei** ist das Vorbild und bleibt unangetastet. Berührt wird sie an
  zwei Stellen: der Zahlungsplan aus Stufe 2 nimmt eine Ausleihung als Besitzer auf,
  und der Kontenkatalog bekommt die Ausleihungen an Gesellschafter (6.8).
- Der **Festschreibungs-Workflow** prüft heute die AfA (`ensureDepreciationBooked` in
  `internal/wailsbridge/festschreibung_service.go`). Er wird zum Prüfschritt für alle
  stichtagsgetriebenen Bewegungen – auch die Rechnungsabgrenzung gehört später dorthin.
- Der **Kontakt** braucht die Merkmale „Gesellschafter", „verbundenes Unternehmen" und
  „Beteiligungsverhältnis" – ohne sie sind weder Kontenwahl noch Anhangangabe möglich.
  Das ist die einzige Modelländerung außerhalb der neuen Dateien.
- Die **Stammdaten** tragen die Rechtsform inzwischen als Katalogwert
  (`domain.LegalFormCatalog`); die Ableitung der Gesellschafterkonten kann sie lesen,
  statt Text zu raten.
- Die **Dokumentenablage** (`internal/receiptstore`, Zweig `dokumente/`) nimmt
  Kreditverträge auf, ohne dass dafür etwas Neues entsteht.
- Die **E-Bilanz** braucht das erweiterte Kontenmapping aus Abschnitt 11 und einen
  Verbindlichkeitenspiegel nach dem Muster von `AnlagenspiegelSource`.
- Der **DATEV-Export** ist nicht betroffen: Er sieht fertige Journalbuchungen.

## 15. Quellen

Stand der Prüfung: 28.08.2026, Volltexte über gesetze-im-internet.de.

| Aussage im Dokument | Fundstelle | Link |
|---|---|---|
| Gliederung der Passivseite: A. Eigenkapital, B. Rückstellungen, C. Verbindlichkeiten (1–8), D. Rechnungsabgrenzungsposten, E. Passive latente Steuern | § 266 Abs. 3 HGB | [hgb/__266.html](https://www.gesetze-im-internet.de/hgb/__266.html) |
| Vermerk der Verbindlichkeiten mit Restlaufzeit bis zu einem Jahr und über einem Jahr bei jedem gesondert ausgewiesenen Posten | § 268 Abs. 5 Satz 1 HGB | [hgb/__268.html](https://www.gesetze-im-internet.de/hgb/__268.html) |
| Rückstellungen für ungewisse Verbindlichkeiten, drohende Verluste, unterlassene Instandhaltung (drei Monate), Abraumbeseitigung, Gewährleistung ohne rechtliche Verpflichtung | § 249 Abs. 1 HGB | [hgb/__249.html](https://www.gesetze-im-internet.de/hgb/__249.html) |
| Auflösung nur, soweit der Grund entfallen ist | § 249 Abs. 2 Satz 2 HGB | dito |
| Verbindlichkeiten zum Erfüllungsbetrag, Rückstellungen zum notwendigen Erfüllungsbetrag | § 253 Abs. 1 Satz 2 HGB | [hgb/__253.html](https://www.gesetze-im-internet.de/hgb/__253.html) |
| Abzinsung von Rückstellungen mit Restlaufzeit über einem Jahr; sieben Geschäftsjahre, bei Altersversorgungsverpflichtungen zehn | § 253 Abs. 2 Satz 1 HGB | dito |
| Pauschalierung mit angenommener Restlaufzeit von 15 Jahren für Altersversorgungsverpflichtungen | § 253 Abs. 2 Satz 2 HGB | dito |
| Disagio-Wahlrecht: Unterschiedsbetrag zwischen Erfüllungs- und Ausgabebetrag darf als aktiver RAP angesetzt werden | § 250 Abs. 3 Satz 1 HGB | [hgb/__250.html](https://www.gesetze-im-internet.de/hgb/__250.html) |
| Anhang: Gesamtbetrag der Verbindlichkeiten mit Restlaufzeit über fünf Jahren; Gesamtbetrag der gesicherten Verbindlichkeiten mit Art und Form der Sicherheiten | § 285 Nr. 1 HGB | [hgb/__285.html](https://www.gesetze-im-internet.de/hgb/__285.html) |
| Ausleihungen, Forderungen und Verbindlichkeiten gegenüber Gesellschaftern gesondert ausweisen oder im Anhang angeben (GmbH) | § 42 Abs. 3 GmbHG | [gmbhg/__42.html](https://www.gesetze-im-internet.de/gmbhg/__42.html) |
| Dasselbe für Personenhandelsgesellschaften; Gliederung des Eigenkapitals nach Kapitalanteilen | § 264c Abs. 1 und 2 HGB | [hgb/__264c.html](https://www.gesetze-im-internet.de/hgb/__264c.html) |
| Erhaltene Anzahlungen gesondert unter den Verbindlichkeiten, soweit nicht offen von den Vorräten abgesetzt | § 268 Abs. 5 Satz 2 HGB | [hgb/__268.html](https://www.gesetze-im-internet.de/hgb/__268.html) |
| Kreditgewährung ist steuerfrei – „die Gewährung und die Vermittlung von Krediten" | § 4 Nr. 8 Buchst. a UStG | [ustg_1980/__4.html](https://www.gesetze-im-internet.de/ustg_1980/__4.html) |

**Nicht verifiziert und vor der Umsetzung zu prüfen:** der genaue Umfang der
Erleichterungen für kleine Kapitalgesellschaften nach § 288 Abs. 1 HGB, insbesondere ob
die Angabe nach § 285 Nr. 1 HGB davon erfasst ist. Die Norm verweist auf eine
Aufzählung einzelner Nummern des § 285 HGB, die hier nicht im Volltext geprüft werden
konnte.

**Keine Gesetzesfrage, sondern Konvention:** die Zuordnung der Restlaufzeiten zu
getrennten SKR04-Konten (3151 / 3160 / 3170). § 268 Abs. 5 HGB verlangt den Vermerk,
nicht bestimmte Konten. DATEV löst ihn über die Kontenstruktur; eine Auswertung, die die
Restlaufzeit rechnerisch aus dem Plan bestimmt, wäre gleichwertig – aber nur, solange
beide Wege dasselbe Ergebnis liefern.
