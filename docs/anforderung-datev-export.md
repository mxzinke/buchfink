# Buchfink – DATEV-Export

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-22
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> **Dieses Dokument ist als einziges nicht gegen eine Primärquelle geprüft.** Ein
> Versuch am 22.08.2026 ist gescheitert: die DATEV-Formatbeschreibung liegt hinter
> einem JavaScript-Portal, das ohne Browser nur eine leere Hülle ausliefert
> (`developer.datev.de/de/file-format/…` und die Wissensplattform, Dok. 1036228,
> liefern beide keinen Text). Alle Feldnamen, Formatkategorien und Längen unten
> stammen daher aus **Sekundärquellen** und sind mit **[unverifiziert]** markiert.
> Sie sind vor der Umsetzung gegen die offizielle Beschreibung zu prüfen – siehe
> [Abschnitt 8](#8-quellen).
>
> Der fachliche Kern des Dokuments – die Architekturentscheidung in Abschnitt 2 –
> hängt an keinem dieser Details.

## 1. Wozu

Der Steuerberater arbeitet mit DATEV. Solange Buchfink seine Daten nicht in dessen
Format liefert, bleibt die Mandantenbuchhaltung eine Insel: entweder tippt jemand ab,
oder das Unternehmen bucht doch wieder beim Berater. Der Export ist damit weniger ein
Feature als die Bedingung dafür, dass die Software im Alltag benutzbar ist.

Anders als die übrigen offenen Punkte braucht dieser hier **keine buchhalterische
Entscheidung**: das Format ist von DATEV vorgegeben. Es gibt genau eine
Architekturfrage, und die steht im nächsten Abschnitt.

## 2. Die eigentliche Frage: n Zeilen gegen Konto und Gegenkonto

Buchfink führt eine Buchung als Kopf mit beliebig vielen Zeilen. Ein Eingangsbeleg mit
Vorsteuer hat drei, ein Reverse-Charge-Vorgang vier:

```
SOLL  5909  Fremdleistungen § 13b     1.000,00
SOLL  1407  Vorsteuer § 13b 19 %        190,00
HABEN 3837  Umsatzsteuer § 13b 19 %     190,00
HABEN 70001 Kreditor                  1.000,00
```

Der DATEV-Buchungsstapel kennt dagegen pro Datensatz genau **ein Konto und ein
Gegenkonto**. Die Steuer entsteht dort nicht als eigene Zeile, sondern wird über den
**BU-Schlüssel** (Buchungsschlüssel) von der DATEV-Steuerautomatik selbst errechnet.
Dieselbe Buchung ist dort ein einziger Datensatz: Konto 5909, Gegenkonto 70001, Umsatz
1.000,00, BU-Schlüssel für § 13b.

Damit stehen zwei Wege offen, und sie schließen sich aus.

### Weg A: über die Steuerautomatik

Die Steuerzeilen werden beim Export **weggelassen** und durch den passenden BU-Schlüssel
ersetzt. DATEV rechnet die Steuer neu.

*Dafür:* Das ist die Form, die ein Steuerberater erwartet. Die Buchungen sehen aus wie
selbst erfasst, die Steuerkonten stimmen mit DATEVs eigener Logik überein, und der
Stapel lässt sich ohne Nacharbeit verarbeiten.

*Dagegen:* Buchfink gibt die Kontrolle über einen Betrag ab, den es selbst schon
berechnet hat. Rundet DATEV in einem Fall anders, weicht der importierte Stapel um
Cent-Beträge von der eigenen Buchhaltung ab – und zwar unbemerkt, weil niemand beide
Seiten nebeneinander legt. Außerdem braucht es eine vollständige Abbildung der eigenen
Steuerfälle auf BU-Schlüssel, die bei § 13b und innergemeinschaftlichem Erwerb nicht
trivial ist.

### Weg B: Steuerzeilen explizit exportieren

Jede Buchungszeile wird ein eigener DATEV-Datensatz, ohne BU-Schlüssel.

*Dafür:* Die Beträge sind exakt die gebuchten. Keine Neuberechnung, keine Abweichung.

*Dagegen:* Der Stapel sieht für den Berater ungewohnt aus, die Steuerautomatik ist
umgangen, und DATEVs eigene USt-Auswertungen greifen unter Umständen nicht. Wie eine
mehrzeilige Buchung als Folge von Konto/Gegenkonto-Paaren dargestellt wird, ohne die
Klammer um den Geschäftsvorfall zu verlieren, ist ebenfalls zu klären.

### Empfehlung

**Weg A**, mit einer Absicherung: Buchfink exportiert über die Steuerautomatik und legt
dem Export ein Protokoll bei, das je Buchung den selbst berechneten Steuerbetrag nennt.
Weicht der Import ab, ist die Differenz auffindbar statt unsichtbar. Zusätzlich ein
Test, der für jeden unterstützten Steuerfall prüft, dass die eigene Berechnung und die
BU-Schlüssel-Logik zum selben Betrag führen.

Diese Entscheidung ist der Grund, warum der Export ein eigenes Dokument bekommt: sie
lässt sich später nur schwer zurücknehmen.

## 3. Was exportiert wird

| Datei | Inhalt | Formatkategorie **[unverifiziert]** |
|---|---|---|
| **Buchungsstapel** | die Buchungen eines Zeitraums | 21 |
| **Debitoren/Kreditoren** | Personenkonten-Stammdaten | 16 |
| **Sachkontenbeschriftungen** | abweichende Kontobezeichnungen | 20 |

Alle drei als CSV mit Semikolon, Windows-1252, einer Kopfzeile mit Metadaten, einer
Zeile mit Spaltenüberschriften und danach den Daten. **[unverifiziert]**

Die Kopfzeile beginnt mit dem Kennzeichen `EXTF` (Export aus einem Fremdprogramm)
und trägt Versionsnummer, Formatkategorie, Formatversion, Berater- und
Mandantennummer, Wirtschaftsjahresbeginn, Sachkontenlänge und Buchungszeitraum.
**[unverifiziert]**

### Buchungsstapel: die tragenden Felder

Alle Zeilen dieser Tabelle sind **[unverifiziert]**.

| Feld | Quelle in Buchfink | Anmerkung |
|---|---|---|
| Umsatz | Betrag der Buchung | **ohne Vorzeichen**, Komma als Dezimaltrennzeichen |
| Soll/Haben-Kennzeichen | Seite der Zeile | S oder H |
| Konto | Konto der Hauptzeile | |
| Gegenkonto | Gegenkonto | ohne BU-Schlüssel-Präfix |
| BU-Schlüssel | aus dem Steuerfall abgeleitet | siehe Abschnitt 2 |
| Belegdatum | Belegdatum | Format TTMM – **ohne Jahr**, das steht im Kopf |
| Belegfeld 1 | Belegnummer | |
| Buchungstext | Buchungstext | Längenbegrenzung beachten |
| Generalumkehr | `Kind == reversal` | dafür wurde die Generalumkehr eingeführt |
| Festschreibung | Festschreibungsstand | |

Zwei Fallen fallen hier schon auf: **das Belegdatum trägt kein Jahr**, es kommt aus dem
Wirtschaftsjahr im Dateikopf – ein Export über eine Jahresgrenze hinweg geht also nicht
in einer Datei. Und **der Umsatz hat kein Vorzeichen**; die Richtung steckt allein im
Soll/Haben-Kennzeichen. Eine Generalumkehr mit negativem Betrag muss deshalb über das
GU-Kennzeichen abgebildet werden und nicht über ein Minus im Betrag.

Beide Punkte sind nicht verifiziert, aber sie sind der Grund, warum der Export
überhaupt Anforderungen an den Buchungskern stellt – wenn eines davon anders ist,
ändert sich die Abbildung, nicht das Datenmodell. Der Kern trägt beide Varianten:
das Belegdatum steht vollständig im Journal, und der Storno-Kind steht am Kopf.

### Personenkonten

Der Export der Debitoren und Kreditoren ist der Grund, warum echte Nummernkreise
gebraucht wurden. Er transportiert Kontonummer, Name, Anschrift, USt-IdNr.,
Zahlungsbedingungen und Bankverbindung – alles bereits vorhanden.

Zu klären ist die **Sachkontenlänge**: DATEV leitet die Länge der Personenkonten aus
der Sachkontenlänge ab. Bei vierstelligen Sachkonten sind Personenkonten fünfstellig,
was zu den Bereichen 10000–69999 und 70000–99999 passt. Der Wert gehört in den
Dateikopf und muss mit den Einstellungen des Beraters übereinstimmen, sonst weist DATEV
den Stapel ab. **[unverifiziert]**

## 4. Was der Export braucht und schon hat

| Voraussetzung | Stand |
|---|---|
| Personenkonten in DATEV-Nummernkreisen | vorhanden |
| Generalumkehr-Kennzeichen statt Seitentausch | vorhanden |
| Steuerfall je Buchung gespeichert | vorhanden (Steuerschlüssel an der Zeile) |
| Belegnummer und Belegdatum getrennt | vorhanden |
| Festschreibungsstand je Periode | vorhanden |
| Beraternummer, Mandantennummer, WJ-Beginn | **fehlt** – gehört in die Stammdaten |
| Abbildung Steuerfall → BU-Schlüssel | **fehlt** |

Die beiden Lücken sind klein. Beraternummer und Mandantennummer bekommt der Nutzer vom
Steuerberater; ohne sie akzeptiert DATEV den Import nicht.

## 5. Rückweg

Ein Import aus DATEV ist **nicht** vorgesehen. Buchungen, die außerhalb entstanden
sind, hätten keine Hash-Chain und keinen Beleg – sie würden die Unveränderbarkeitskette
unterbrechen, die den Rest des Systems trägt. Wer beim Berater bucht, bucht dort; ein
Rückimport wäre kein Datenaustausch, sondern eine zweite Buchhaltung.

## 6. Offene Entscheidungen

- **Weg A oder B** (Abschnitt 2) – die einzige wirklich offene Frage.
- **Exportumfang:** ganzer Zeitraum oder nur festgeschriebene Perioden? *Vorschlag: nur
  festgeschriebene, damit der Berater keine noch veränderlichen Buchungen bekommt.*
- **Belegdateien:** DATEV kann Belegbilder mit dem Stapel verknüpfen. In v1 mitliefern
  oder auf die Buchungen beschränken?
- **Wiederholte Exporte:** wie wird verhindert, dass derselbe Stapel zweimal importiert
  wird? DATEV erkennt Dubletten nicht zuverlässig.

## 7. Abhängigkeiten

- **Stammdaten** brauchen Berater- und Mandantennummer sowie die Sachkontenlänge.
- Die **Steuerlogik** braucht eine Abbildung ihrer Steuerfälle auf BU-Schlüssel.
- Der Export sollte **nach** Anlagen und Anzahlungen kommen, damit deren Buchungen von
  Anfang an mit abgedeckt sind – blockiert ist er dadurch aber nicht.

## 8. Quellen

**Primärquelle, noch nicht ausgewertet.** Verbindlich ist allein die
DATEV-Formatbeschreibung:

- DATEV-Entwicklerportal, „DATEV-Format – Buchungsstapel":
  <https://developer.datev.de/de/file-format/details/datev-format/format-description/booking-batch>
- DATEV-Wissensplattform, Dokument **1036228** („DATEV-Format"):
  <https://wissensplattform.apps.datev.de/help/document/1036228>

Beide Seiten wurden am 22.08.2026 abgerufen und liefern ohne JavaScript-Browser
keinen Text (die erste antwortet mit 200 und 648 Byte leerer Hülle). Eine
Verifikation im Rahmen dieser Konzeptarbeit war damit nicht möglich.

**Was tatsächlich zu tun ist**, bevor implementiert wird: die Formatbeschreibung
als PDF aus dem DATEV-Portal ziehen (dafür genügt ein normaler Browser, kein
DATEV-Zugang) und die Tabellen in Abschnitt 3 Feld für Feld dagegen abgleichen.
Erst danach dürfen die **[unverifiziert]**-Marken entfallen.

**Sekundärquellen**, aus denen die Angaben in Abschnitt 3 stammen – brauchbar für
das Grobbild, nicht als Implementierungsgrundlage:

| Quelle | Link |
|---|---|
| `datev-types-rs` (Rust-Implementierung des Formats) | <https://github.com/JensWalter/datev-types-rs> |
| „DATEV EXTF-Format erklärt" | <https://smartkontoauszug.de/blog/datev-extf-format-erklaert> |
| „DATEV Buchungsstapel EXTF — Format, Export & Auswertung" | <https://auditplan.io/datev-buchungsstapel-extf> |
| DATEV-Community, Thread zur Schnittstellenbeschreibung | <https://www.datev-community.de/t5/Betriebliches-Rechnungswesen/DATEV-Format-Schnittstellenbeschreibung/td-p/24100> |

Sie stimmen untereinander darin überein, dass die Kopfzeile mit `EXTF` beginnt und
die Formatkategorie 21 den Buchungsstapel bezeichnet. Zur Formatversion nennen sie
unterschiedliche Stände – ein weiterer Grund, die Primärquelle abzuwarten.

**Nicht zu verwechseln:** die BU-Schlüssel sind kein Teil der Formatbeschreibung,
sondern hängen am Kontenrahmen und am Mandanten. Die Abbildung aus Abschnitt 4
(„Steuerfall → BU-Schlüssel") braucht also eine eigene Quelle und muss gegen die
Einstellungen des jeweiligen Beraters geprüft werden.
