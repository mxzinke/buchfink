# Buchfink – Passiva verwalten: Darlehen, Rückstellungen, Sonderposten

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-28
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)
Schwesterdokument: [Anlagenverwaltung](anforderung-anlagenverwaltung.md) – dieselbe Mechanik, andere Bilanzseite

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind am **28.08.2026** gegen den
> Gesetzestext auf gesetze-im-internet.de verifiziert; die Fundstellen stehen in
> [Abschnitt 15](#15-quellen).

## 1. Warum das dieselbe Maschine ist wie das Anlagenverzeichnis

Das Anlagenverzeichnis ist kein Aktivthema. Es ist der erste Fall eines Musters, das
auf der Passivseite mindestens dreimal wieder auftaucht:

> Ein Bestand entsteht einmal, verändert sich planmäßig über Jahre und verschwindet
> irgendwann – und keine dieser Veränderungen hat einen Beleg.

Ein Anlagegut wird angeschafft, jährlich abgeschrieben, geht ab. Ein Darlehen wird
ausgezahlt, monatlich getilgt und verzinst, ist am Ende getilgt. Eine Rückstellung wird
gebildet, fortgeschrieben, verbraucht oder aufgelöst. Ein Sonderposten für einen
Zuschuss wird eingestellt und über die Nutzungsdauer aufgelöst.

Vier Sachverhalte, ein Ablauf: **Stammsatz anlegen, Plan rechnen, Bewegungen zur
Freigabe vorschlagen, Journalbuchung erzeugen, Rückverweis behalten.** Wer das
Anlagenverzeichnis als Spezialfall baut, baut es dreimal. Wer es als Verzeichnis mit
Plan baut, bekommt die Passivseite fast geschenkt.

Deshalb gehört die Entscheidung **vor** die Umsetzung der Anlagenverwaltung, nicht
danach: Es geht nicht darum, was die Anlagenverwaltung zusätzlich können muss, sondern
darum, dass ihr Datenmodell nicht nur für Anlagegüter geschnitten wird.

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
| **C. 5** Wechsel | gezogene und eigene Wechsel | **3350**–**3399** | out of scope |
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

| Teil | Inhalt | Beispiel Darlehen | Beispiel Anlagegut |
|---|---|---|---|
| **Stammsatz** | Bezeichnung, Bestandskonto, Kontakt, Beginn, Ausgangsbetrag, Status | 3160, Hausbank, 50.000 €, 01.03.2026 | 0520, 34.000 €, 12.09.2026 |
| **Plan** | Zeitreihe geplanter Bewegungen mit Fälligkeit, Betrag, Art, Soll- und Habenkonto | 60 Raten à 1.000 € (Tilgung + Zins) | 72 Monats-AfA |
| **Bewegungen** | Ist-Buchungen mit Verweis auf Journaleintrag und Bankumsatz | gebuchte Raten | gebuchte AfA |

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
(`PaymentService.OpenItems`, `internal/service/payment_service.go:86`). Offene Posten
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

### 6.8 Der Spiegelfall: Darlehen, die das Unternehmen vergibt

Dasselbe Objekt mit umgekehrtem Vorzeichen. Der Plan ist identisch, nur Soll und Haben
tauschen, und die Kontenableitung greift auf die Aktivseite:

| Fall | Konto |
|---|---|
| dauerhaft überlassen (Finanzanlage) | **0940** Darlehen, **0930** übrige sonstige Ausleihungen, **0970** nahe stehende Personen |
| an Gesellschafter | **0960**–**0963** Ausleihungen an Gesellschafter |
| an verbundene Unternehmen / Beteiligungen | **0810** ff. / **0880** ff. |
| kurzfristig (Umlaufvermögen) | **1360** / **1361** / **1365** Darlehen |
| Zinsertrag | **7011** Erträge aus Ausleihungen des Finanzanlagevermögens, sonst **7100** |

Ein Verzeichnis, das nur Verbindlichkeiten kann, müsste für diesen Fall ein zweites Mal
gebaut werden. Ein Verzeichnis mit Bilanzseite am Stammsatz kann beides.

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

**Daraus folgt eine Anforderung an das Datenmodell der Anlagenverwaltung:** Ein
Verzeichniseintrag muss auf einen anderen verweisen können. Wird das jetzt nicht
vorgesehen, ist die Kopplung Zuschuss ↔ Anlagegut später ein Umbau.

## 10. Wo das im Code hingehört

Die Reihenfolge ist wichtiger als der Umfang: Der gemeinsame Kern zuerst, die
Sachverhalte danach.

| Schicht | Datei | Inhalt |
|---|---|---|
| Domäne | `internal/domain/register.go` (neu) | `RegisterItem` (Stammsatz, Bilanzseite, Bestandskonto, Kontakt, Status), `PlannedMovement`, `RegisterMovement` mit `JournalEntryID` und `BankTxID`, Repository-Interfaces – analog zu `domain/receipt.go` |
| Persistenz | `internal/repository/register_gorm.go` (neu), Eintrag in `AutoMigrate` (`internal/repository/db.go:71`) | jahresübergreifend geführt, nicht je Geschäftsjahr – dieselbe Anforderung wie bei der Anlagenkartei |
| Kontenableitung | `internal/accounting/register_accounts.go` (neu) | Kreditgebertyp × Rechtsform × Restlaufzeit → Konto. Bewusst getrennt von `posting_groups.go`: dort entscheidet der Steuerfall, hier die Vertragsart |
| Plan | `internal/service/register_service.go` (neu) | Tilgungs-, Auflösungs- und Abgrenzungspläne rechnen; Vorschau erzeugen; nach Freigabe über `JournalService.Post` buchen |
| Zahlungszuordnung | `internal/service/payment_service.go` erweitern | zweite Quelle neben `OpenItems`: erwartete Zahlungen aus Plänen, gleiche Struktur, gleiche Differenzbehandlung |
| Abschluss | Festschreibungs-Workflow | ein Prüfschritt für alle stichtagsgetriebenen Bewegungen: AfA, RAP, Zinsabgrenzung, Restlaufzeit-Umgliederung, Sonderposten, Rückstellungsbewertung |
| Bridge | `internal/wailsbridge/app_service.go` | CRUD, Planvorschau, Freigabe – wie `PreviewIncomingReceipt` / `PostIncomingReceipt` |
| Oberfläche | `frontend/src/pages/` + `Sidebar.tsx:62` | eigene Gruppe „Verzeichnisse" mit Anlagen, Darlehen, Rückstellungen; im Bankimport erscheinen fällige Raten in der bestehenden Zuordnungsansicht |

Zwei Dinge sind bewusst **nicht** in dieser Liste: eine neue Buchungsmaske und eine
Erweiterung von `BookDirect`. Beides wäre der falsche Weg – Buchungen entstehen aus dem
Plan, nicht aus einer Eingabe.

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
- **E-Bilanz:** die Mapping-Tabelle `skr04ToXBRL` (`internal/ebilanz/ebilanz.go:12`)
  enthält kein einziges Darlehens-, Rückstellungs- oder Eigenkapitalkonto außer 2000,
  2100 und 2900. Alles andere fällt auf `de-gaap-ci:bs.other`
  (`internal/ebilanz/ebilanz.go:66`). Ein Verzeichnis, dessen Ergebnis in der Sammelposition
  landet, hilft beim Abschluss nicht. Auch die drei vorhandenen Zuordnungen sind zu
  prüfen: **2900** Gezeichnetes Kapital zeigt dort auf `bs.eqLiab.equity.retainedEarn`
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

- **Ein Verzeichnis oder drei?** Vorschlag: ein Datenmodell mit `kind`-Feld, drei
  Ansichten. Der Alternativweg – getrennte Tabellen je Sachverhalt – ist am Anfang
  schneller und spätestens beim Sonderposten teuer.
- **Namensgebung im Code:** `RegisterItem` / `PlannedMovement` oder fachlich deutsch?
  Der Bestand ist englisch benannt mit deutschen Kommentaren und Meldungen; das sollte so
  bleiben.
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

- Die **Anlagenverwaltung** sollte auf demselben Kern aufsetzen. Wird sie vorher als
  Einzellösung gebaut, entsteht der Umbau später doppelt – siehe Abschnitt 9 zur
  Kopplung Zuschuss ↔ Anlagegut.
- Der **Festschreibungs-Workflow** bekommt einen gemeinsamen Prüfschritt für alle
  stichtagsgetriebenen Bewegungen; AfA und Rechnungsabgrenzung beschreiben ihn bereits
  je für sich.
- Der **Kontakt** braucht die Merkmale „Gesellschafter", „verbundenes Unternehmen" und
  „Beteiligungsverhältnis" – ohne sie sind weder Kontenwahl noch Anhangangabe möglich.
- Die **Stammdaten** brauchen die Rechtsform als auswertbares Feld, nicht nur als Text.
- Die **E-Bilanz** braucht das erweiterte Kontenmapping aus Abschnitt 11.
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
