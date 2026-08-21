# Buchfink – DATEV-Export

Status: Anforderung, noch nicht implementiert
Letzte Aktualisierung: 2026-08-21
Voraussetzung: [Beleg- & Buchungsflow](anforderung-beleg-buchungsflow.md)

> Die Feldnamen und Formatnummern in diesem Dokument sind **[zu verifizieren]** gegen
> die offizielle DATEV-Formatdokumentation, bevor implementiert wird. Der fachliche
> Kern – Abschnitt 2 – hängt nicht daran.

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

| Datei | Inhalt | Formatkategorie [zu verifizieren] |
|---|---|---|
| **Buchungsstapel** | die Buchungen eines Zeitraums | 21 |
| **Debitoren/Kreditoren** | Personenkonten-Stammdaten | 16 |
| **Sachkontenbeschriftungen** | abweichende Kontobezeichnungen | 20 |

Alle drei als CSV mit Semikolon, Windows-1252, einer Kopfzeile mit Metadaten, einer
Zeile mit Spaltenüberschriften und danach den Daten. [zu verifizieren]

### Buchungsstapel: die tragenden Felder

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

### Personenkonten

Der Export der Debitoren und Kreditoren ist der Grund, warum echte Nummernkreise
gebraucht wurden. Er transportiert Kontonummer, Name, Anschrift, USt-IdNr.,
Zahlungsbedingungen und Bankverbindung – alles bereits vorhanden.

Zu klären ist die **Sachkontenlänge**: DATEV leitet die Länge der Personenkonten aus
der Sachkontenlänge ab. Bei vierstelligen Sachkonten sind Personenkonten fünfstellig,
was zu den Bereichen 10000–69999 und 70000–99999 passt. Der Wert gehört in den
Dateikopf und muss mit den Einstellungen des Beraters übereinstimmen, sonst weist DATEV
den Stapel ab.

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
