# Buchfink – Gründung einer Kapitalgesellschaft

Status: umgesetzt
Letzte Aktualisierung: 2026-09-01
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Kontonummern sind gegen `internal/accounting/skr04_2026.json` (DATEV SKR04 2026,
> Art.-Nr. 11175) geprüft. Alle Paragrafenangaben sind am **01.09.2026** gegen den
> Gesetzestext auf gesetze-im-internet.de verifiziert; die Fundstellen stehen in
> [Abschnitt 9](#9-quellen).

## 1. Worum es geht

Eine GmbH entsteht nicht beim Notar und nicht mit der ersten Rechnung, sondern in
zwei Schritten. Beim Notar wird der Gesellschaftsvertrag beurkundet (§ 2 GmbHG),
und von da an existiert die **Vorgesellschaft**. Als juristische Person entsteht
die GmbH erst mit der Eintragung ins Handelsregister (§ 11 Abs. 1 GmbHG).
Dazwischen liegen Wochen bis Monate, und in dieser Zeit ist das Unternehmen
bereits buchführungspflichtig, zahlt Notar- und Gerichtsgebühren, mietet an und
kauft ein.

Zwei Haftungen hängen an dieser Phase, und beide überraschen Gründer
regelmäßig:

**Handelndenhaftung.** „Ist vor der Eintragung im Namen der Gesellschaft
gehandelt worden, so haften die Handelnden persönlich und solidarisch"
(§ 11 Abs. 2 GmbHG). Sie endet mit der Eintragung. Buchfink kann sie nicht
abwenden, aber es kann sagen, wie lange sie noch läuft.

**Unterbilanzhaftung.** Bleibt das Reinvermögen der Gesellschaft am Tag der
Eintragung hinter dem Stammkapital zurück, schulden die Gesellschafter die
Differenz, anteilig nach ihren Geschäftsanteilen. Das ist Richterrecht, kein
Paragraf. Verschärft wird es durch § 248 Abs. 1 Nr. 1 HGB: Aufwendungen für die
Gründung eines Unternehmens dürfen nicht aktiviert werden. Die Notarrechnung ist
also sofort Aufwand und mindert das Reinvermögen in voller Höhe.

Das Rechenbeispiel, an dem die ganze Funktion hängt:

| Vorgang | Wirkung auf das Reinvermögen |
|---|---|
| Stammkapital 25.000 € gezeichnet, 12.500 € eingezahlt | 25.000 € (Bank plus offene Einlageforderung) |
| Notar- und Gerichtskosten 3.000 €, aus der Bank bezahlt | 22.000 € |
| **Unterbilanz am Eintragungstag** | **3.000 €**, davon 1.800 € und 1.200 € bei 60/40 |

Die 3.000 € stehen seit dem Tag der Notarrechnung im Journal. Buchfink hat sie
bisher nur nicht zusammengezählt.

## 2. Für wen der Gründungsweg gilt

Nur für Kapitalgesellschaften: GmbH, UG (haftungsbeschränkt), AG. Bei einer
Personengesellschaft gibt es keine Vorgesellschaft, keine Handelndenhaftung des
§ 11 Abs. 2 GmbHG und keine Unterbilanzhaftung. Der Katalog steht in
`internal/accounting/gruendung.go`; `FoundationRulesFor` ist die einzige Stelle,
an der entschieden wird, ob eine Rechtsform dazugehört. Die Oberfläche fragt
dort, statt Rechtsformnamen zu vergleichen.

Die drei Phasen:

| Phase | Beginn | Ende | Was gilt |
|---|---|---|---|
| Vorgründung | Entschluss | Beurkundung | GbR oder OHG, **nicht** identisch mit der späteren GmbH. Verbindlichkeiten gehen nicht automatisch über. |
| Vorgesellschaft | Beurkundung | Eintragung | Buchführungspflicht, Handelndenhaftung, die Unterbilanz wächst mit jeder Buchung. |
| Eingetragene Gesellschaft | Eintragung | — | Haftungsbeschränkung greift, die Unterbilanz steht fest. |

Buchfink bildet die Phasen zwei und drei ab. Die Vorgründung wird benannt und
nicht verwaltet: vor der Beurkundung gibt es keinen Mandanten und nichts zu
buchen.

## 3. Kapitalaufbringung

| Rechtsform | Mindestkapital | Vor der Anmeldung zu leisten | Fundstelle |
|---|---|---|---|
| GmbH | 25.000 € | Auf jeden Geschäftsanteil ein Viertel des Nennbetrags; zusammen mindestens die Hälfte des **Mindest**stammkapitals, also 12.500 € | § 5 Abs. 1, § 7 Abs. 2 GmbHG |
| UG (haftungsbeschränkt) | 1 € | Das Stammkapital in voller Höhe, Sacheinlagen ausgeschlossen | § 5a Abs. 1 und 2 GmbHG |
| AG | 50.000 € | Bei Bareinlagen mindestens ein Viertel des geringsten Ausgabebetrags | § 7, § 36a Abs. 1 AktG |

Die Untergrenze der GmbH ist der Punkt, an dem eine Umsetzung leicht danebengreift.
§ 7 Abs. 2 Satz 2 GmbHG nennt die Hälfte des Mindeststammkapitals nach § 5 Abs. 1,
nicht die Hälfte des vereinbarten Stammkapitals. Wer eine GmbH mit 100.000 €
gründet, schuldet vor der Anmeldung 25.000 € (ein Viertel je Anteil), nicht
50.000 €. `FoundationRules.RequiredPaidIn` rechnet deshalb beides und nimmt den
höheren Wert.

Sacheinlagen zählen mit ihrem vollen Nennbetrag, weil sie vor der Anmeldung
vollständig zu bewirken sind (§ 7 Abs. 3 GmbHG) und § 7 Abs. 2 Satz 2 sie mit dem
Gesamtnennbetrag in die Rechnung nimmt.

## 4. Die Unterbilanzrechnung

```
Reinvermögen = Aktiva + noch ausstehende Einlagen − Schulden
Unterdeckung = Stammkapital − Reinvermögen        (nie negativ)
gedeckt      = min(Gründungsaufwand laut Satzung, Unterdeckung)
Haftung      = Unterdeckung − gedeckt
je Gesellschafter = Haftung × Geschäftsanteil ÷ Stammkapital
```

Drei Entscheidungen darin sind erklärungsbedürftig.

**Die ausstehende Einlage zählt zum Reinvermögen.** Konto 1298 steht als
Forderung auf der Aktivseite, Konto 2910 als offene Absetzung vom gezeichneten
Kapital auf der Passivseite (§ 272 Abs. 1 Satz 3 HGB). Der Sache nach sind beide
Ansprüche gegen den Gesellschafter, und beide gehören ins Reinvermögen. Ließe man
sie weg, zeigte die Rechnung eine Unterbilanz in Höhe der noch nicht gezahlten
Einlage. Das wäre falsch: Die Einlage wird geschuldet, aber als Einlage, nicht
als Vorbelastungshaftung. Beide Ansprüche stehen nebeneinander.

**Der Stichtag ist der Tag der Eintragung.** Fehlt er noch, rechnet Buchfink auf
heute und schreibt „vorläufig" daneben. Eingefroren wird die Zahl nicht: Die
Buchungen bis zum Eintragungstag hängen in der Hash-Chain, und was sich nicht
mehr ändern kann, muss auch nicht zweimal gespeichert werden. Eine rückdatierte
Buchung könnte das Ergebnis noch verschieben; davor schützt die Festschreibung
des Zeitraums, nicht dieser Dienst.

**Die Satzungsklausel deckt die Unterdeckung, nicht bestimmte Buchungen.**
Buchfink unterscheidet nicht, ob die Unterdeckung aus dem Gründungsaufwand oder
aus einem Anlaufverlust stammt. Diese Zuordnung kann nur der Gründer treffen.
Unterdeckung, gedeckter Anteil und verbleibende Haftung stehen deshalb als drei
Zahlen nebeneinander, statt zu einer verschmolzen zu werden.

Der fehlende Saldenvortrag (siehe [stand-der-umsetzung.md](stand-der-umsetzung.md),
Abschnitt 3) stört hier nicht: Die Gründung liegt im ersten Geschäftsjahr, und
dort sind Bewegung und Bestand dasselbe. Auf ein Folgejahr angewandt wäre die
Rechnung falsch, aber die Vorbelastungshaftung endet mit der Eintragung.

## 5. Buchungen

Zwei Sätze, mehr ist es nicht.

| Schritt | Buchung |
|---|---|
| Zeichnung, am Tag der Beurkundung | SOLL **1298** Ausstehende Einlagen, eingefordert 25.000,00 · HABEN **2900** Gezeichnetes Kapital 25.000,00 |
| Einzahlung je Gesellschafter | SOLL **1800** Bank 7.500,00 · HABEN **1298** Ausstehende Einlagen, eingefordert 7.500,00 |

Gebucht wird über `JournalService.Post`, also denselben Weg wie jede andere
Buchung. Hash-Chain, Belegnummernkreis und Audit-Log greifen unverändert. Die
Quelle ist `EntrySourceOpening`, die bisher definiert war und von keiner Stelle
erzeugt wurde.

Buchfink bucht das volle Stammkapital als eingefordert. Ein nicht eingeforderter
Teil (Konto 2910) ist eine Entscheidung der Gesellschafter und folgt nicht aus den
erfassten Daten; wer ihn braucht, bucht ihn von Hand. Die Unterbilanzrechnung
liest das Konto trotzdem.

Eine Sacheinlage wird nicht automatisch gebucht. Sie gehört auf das Konto des
eingebrachten Gegenstands, und welches das ist, weiß nur der Gründer. Die
Vorschau nennt sie und sagt, warum sie fehlt.

Gebucht wird auf Freigabe, mit Vorschau, wie beim Abschreibungslauf. Buchungen,
die eine Anwendung von sich aus schreibt, sind in einer GoBD-Buchhaltung die
schlechtere Hälfte der Bequemlichkeit.

Der Gründungsaufwand selbst läuft über **6825** Rechts- und Beratungskosten oder
**6827** Abschluss- und Prüfungskosten, als gewöhnlicher Beleg. Ein eigener
Erfassungsweg wäre nur eine zweite Art, dasselbe zu tun.

## 6. Fristen aus der Gründung

| Pflicht | Frist | Fundstelle |
|---|---|---|
| Anmeldung zum Handelsregister | sobald die Mindesteinlage geleistet ist | §§ 7, 8 GmbHG |
| Fragebogen zur steuerlichen Erfassung | einen Monat nach der Gründung | § 138 Abs. 1b und Abs. 4 AO |
| Gewerbeanmeldung bei der Gemeinde | einen Monat | § 14 GewO, § 138 Abs. 1 AO |
| Eröffnungsbilanz auf den Beurkundungstag | im ordnungsmäßigen Geschäftsgang | § 242 Abs. 1 HGB |
| Gesetzliche Rücklage, nur UG | mit dem Jahresabschluss | § 5a Abs. 3 GmbHG |
| Wirtschaftlich Berechtigte melden | unverzüglich nach der Eintragung | § 20 Abs. 1 GwG |
| Ersten Jahresabschluss offenlegen | zwölf Monate nach dem Abschlussstichtag | § 325 Abs. 1a HGB |

Wo das Gesetz „unverzüglich" sagt, steht kein Datum in der Liste, sondern der
Wortlaut. Eine erfundene Tagesfrist wäre bequemer und falsch. Bei der
Eröffnungsbilanz nennt § 242 Abs. 1 HGB ebenfalls keine Frist; der angezeigte
Termin ist als Richtwert gekennzeichnet und stammt aus § 264 Abs. 1 Satz 4 HGB.

Was die Eintragung voraussetzt, erscheint erst nach ihr. Eine Frist, die auf ein
noch nicht eingetretenes Ereignis zeigt, ist keine.

Erledigt wird eine Gründungspflicht mit ihrem Datum in der Datenbank, nicht mit
einem Haken im `localStorage`. Dass der Fragebogen am 12. Oktober übermittelt
wurde, ist eine Tatsache über das Unternehmen. Für die übrigen Steuertermine
bleibt der Haken, was er ist: eine Merkhilfe.

### Der Voranmeldungszeitraum bei Neugründung

Hier stand in der Oberfläche fünf Jahre lang das Falsche. Der
Einrichtungsassistent riet „monatlich gilt bei Neugründung". § 18 Abs. 2 Satz 4
UStG verlangt das auch, aber Satz 6 setzt die Pflicht für die
Besteuerungszeiträume **2021 bis 2026** aus. In dieser Zeit gilt der Regelfall
des Satzes 2, also das Kalendervierteljahr. Ab dem Besteuerungszeitraum 2027 lebt
die monatliche Pflicht wieder auf.

`accounting.RecommendedVatPeriod` beantwortet das aus dem Gründungsjahr,
`VatPeriodReason` liefert die Begründung dazu. Beides steht an einer Stelle, weil
die Regel ein Stichjahr hat: Ein fest getippter Hinweis war seit 2021 falsch und
wäre es ab 2027 wieder.

## 7. Datenmodell

```
Foundation                 höchstens eine je Mandant
├── notarizedOn            Beurkundung, setzt Rumpfgeschäftsjahr und Fristen
├── registeredOn           leer = Vorgesellschaft; die Phase folgt daraus
├── registerCourt/-Number  Amtsgericht, HRB
├── shareCapital           Stammkapital laut Satzung
├── foundationCostCap      Gründungsaufwand laut Satzung
└── shareholders[]         Name (verschlüsselt), Anteil, geleistet, Bar/Sache

FoundationTask             erledigte Pflicht: Schlüssel, Datum, Notiz
```

Die Phase ist abgeleitet, nicht gespeichert. Ein Status, den man unabhängig vom
Datum setzen kann, geht irgendwann mit ihm auseinander, und dann steht in der
Oberfläche etwas anderes als in der Rechnung.

Der Gesellschaftername ist personenbezogen und liegt verschlüsselt, wie jedes
andere Namensfeld (`serializer:encrypted`).

Die Fundstellen im Code:

| Was | Wo |
|---|---|
| Typen und Repository-Schnittstelle | `internal/domain/foundation.go` |
| Regeln je Rechtsform, Fristen, § 18 UStG | `internal/accounting/gruendung.go` |
| Persistenz | `internal/repository/foundation_gorm.go` |
| Unterbilanz, Anmeldungsbefund, Buchungen, Eintragung | `internal/service/foundation_service.go` |
| Bridge | `internal/wailsbridge/app_service.go`, Abschnitt „Gründung" |
| Erfassung | `frontend/src/components/SetupAssistantScreen.tsx` |
| Laufende Begleitung | `frontend/src/components/FoundationSection.tsx`, `frontend/src/pages/DeadlinesPage.tsx` |

## 8. Offene Entscheidungen

**GmbH & Co. KG.** Zwei Gesellschaften, zwei Eintragungen, Kapitalkonten der
Kommanditisten. Die Komplementär-GmbH durchläuft den hier beschriebenen Weg, die
KG einen anderen. Solange die Kapitalkonten der Gesellschafter fehlen (siehe
[stand-der-umsetzung.md](stand-der-umsetzung.md), Abschnitt 4), wäre der zweite
Teil ohnehin nicht darstellbar.

**Bewertung von Sacheinlagen.** Buchfink erfasst, dass eine vorliegt und mit
welchem Nennbetrag. Der Sachgründungsbericht nach § 5 Abs. 4 GmbHG, die
Werthaltigkeitsprüfung und die Differenzhaftung des § 9 GmbHG sind keine
Software-Aufgabe.

**Nicht eingeforderte Einlagen.** Konto 2910 wird gelesen, aber nicht angeboten.
Ob und in welcher Höhe eingefordert wird, entscheiden die Gesellschafter; es aus
dem gesetzlichen Mindestbetrag zu erraten wäre eine Festlegung, die niemand
getroffen hat.

**Die UG-Rücklage nach § 5a Abs. 3 GmbHG.** Sie hängt am Jahresüberschuss und
damit an den Abschlussbuchungen, die es noch nicht gibt. Buchfink führt die
Pflicht als Frist und bucht sie nicht.

**Übermittlung des Fragebogens.** ERiC bleibt out-of-scope (README, „Scope &
Entscheidungen"). Buchfink führt die Frist und den Nachweis; übermittelt wird
über Mein ELSTER.

**Ablage der Gründungsurkunden.** Das Dokumentenmuster gibt es bereits am
Anlagegut (`internal/domain/asset_document.go`). Es auf den Mandanten zu heben
ist ein eigener, sauber abgrenzbarer Schritt.

**Die Zuordnung der Unterdeckung.** Ob eine Unterdeckung aus dem Gründungsaufwand
oder aus einem Anlaufverlust stammt, entscheidet heute der Nutzer im Kopf. Eine
Kennzeichnung am Beleg wäre möglich und ist bewusst nicht gebaut: sie verlangte
eine Angabe bei jeder Buchung der Gründungsphase, und die meisten Gründer haben
gar keine Satzungsklausel.

## 9. Quellen

Geprüft am 01.09.2026 gegen den Wortlaut auf gesetze-im-internet.de.

- § 5 GmbHG – Stammkapital, Geschäftsanteil, Sachgründungsbericht
- § 5a GmbHG – Unternehmergesellschaft: Volleinzahlung, Sacheinlageverbot, gesetzliche Rücklage
- § 7 GmbHG – Anmeldung, Viertelregel, Untergrenze, Sacheinlagen
- § 11 GmbHG – Vorgesellschaft und Handelndenhaftung
- § 7 AktG, § 36a AktG – Grundkapital und Leistung der Einlagen
- § 242 HGB – Pflicht zur Aufstellung der Eröffnungsbilanz
- § 248 Abs. 1 Nr. 1 HGB – Aktivierungsverbot für Gründungsaufwand
- § 272 Abs. 1 HGB – Ausweis von gezeichnetem Kapital und ausstehenden Einlagen
- § 325 HGB – Offenlegung
- § 138 AO – Anzeige der Erwerbstätigkeit, Monatsfrist in Absatz 4
- § 18 Abs. 2 UStG – Voranmeldungszeitraum, Aussetzung 2021 bis 2026 in Satz 6
- § 20 GwG – Mitteilung an das Transparenzregister
- § 14 GewO – Gewerbeanmeldung

Die Unterbilanzhaftung (Vorbelastungshaftung) ist Richterrecht des BGH und steht
in keinem Paragrafen. Sie ist ständige Rechtsprechung; die Rechnung folgt der
herrschenden Auffassung, wonach auf den Tag der Eintragung abzustellen ist und
die Gesellschafter anteilig nach ihren Geschäftsanteilen haften.
