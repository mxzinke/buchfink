# Buchfink – Anforderung: Beleg- & Buchungsflow

Status: Entwurf / Anforderungsdokument (noch nicht implementiert)
Letzte Aktualisierung: 2026-08-19

## 1. Leitgedanke

Ein Abstraktionslayer, der dem Nutzer verständlich ist: er denkt in **Belegen** und **Rechnungen**,
die Software übersetzt über Backend-Logik + saubere Auswahl in die korrekten **SOLL/HABEN-Buchungen**
auf die richtigen Konten. Beide Seiten werden sauber getrennt, aber deterministisch verbunden.

**Kein „vorbereiten, nicht buchen".** Jeder erfasste Beleg/jede Rechnung wird **sofort gebucht**.
Ob dabei Geld fließt, entscheidet die Kontenseite: ist noch nicht bezahlt, entsteht eine
**Verbindlichkeit** (Eingang) bzw. **Forderung** (Ausgang). Die Zahlung ist ein späterer,
separater Geschäftsvorfall.

**Scope-Grenze: Steuern & Auswertungen sind außen vor.** Steuerliche Auswertungen
(USt-Voranmeldung, USt-Erklärung, ZM) und der Jahresabschluss (Saldenvortrag, GuV-Abrechnung,
Bilanzierung) sind in diesem Konzept bewusst **nicht über den Beleg-/Zahlungsflow** abgedeckt.
Sie erfordern eigene, separate Eingabemasken und Auswertungslogik, die nicht Teil des
Abstraktionslayers für die laufenden Geschäftsvorfälle sind. Die hier beschriebenen Vorgänge
dienen rein der Erfassung der laufenden Buchungen; steuer- und abschlussbezogene Buchungen
werden später separat ergänzt.

## 2. Navigation (Reiter)

| Reiter | Nutzer-Sicht | Konten-Sicht |
|---|---|---|
| **Belegsammlung** | Eingangsbelege erfassen (Upload PDF/Bild), Belegdaten + Lieferant | sofortige Buchung auf Aufwand + Verbindlichkeit (falls offen) |
| **Ausgangsrechnungen** | Rechnung erstellen (ZUGFeRD), Kunde + Positionen | sofortige Buchung auf Forderung + Ertrag (falls offen) |
| **Bank & Zahlungen** | Bankumsätze importieren, offene Posten ausgleichen | Zahlungsbuchung auf Bank + OPOS-Ausgleich |
| **Anlagen** | Vermögensgüter verwalten (Fahrzeug, Maschine, Finanzanlagen), AfA | Anlagekonto + Abschreibungsbuchung (vor Festschreibung) |
| **Journal** | gebuchte Geschäftsvorfälle ansehen | SOLL/HABEN, Belegnr., Hash-Chain, unveränderbar |

## 3. Der Flow (überarbeitet)

### 3.1 Eingangsbeleg / Ausgangsrechnung erfassen → sofort buchen

```
Beleg erfassen ──► sofort buchen ──► bezahlt? ──► nein: OPOS offen
                                    │
                                    ja: Zahlung zuordnen ──► OPOS ausgleichen
```

**Immer sofort buchen** – zwei Fälle:

| Fall | Buchung (sofort) |
|---|---|
| **Eingangsbeleg, noch nicht bezahlt** | SOLL Aufwand (+ Vorsteuer) · HABEN Verbindlichkeit a. LuL |
| **Eingangsbeleg, bar bezahlt** | SOLL Aufwand (+ Vorsteuer) · HABEN Bank |
| **Ausgangsrechnung, noch nicht bezahlt** | SOLL Forderung a. LuL · HABEN Ertrag (+ USt) |
| **Ausgangsrechnung, bar bezahlt** | SOLL Bank · HABEN Ertrag (+ USt) |

Ist zum Erfassungszeitpunkt schon eine Zahlung bekannt, wird eine Buchung gemacht (bar). Ist
noch nicht bezahlt, entsteht OPOS. Die Zahlung folgt später (3.2).

### 3.2 Zahlung zuordnen → OPOS ausgleichen

Wenn eine Bankbewegung eingeht, kann sie einem **offenen Beleg** zugeordnet werden:

- **Eine Zahlung → ein Beleg:** klassischer Ausgleich. SOLL Verbindlichkeit · HABEN Bank
  (bzw. SOLL Bank · HABEN Forderung). OPOS schließt.
- **Mehrere Zahlungen → ein Beleg:** Teilzahlungen, Raten, Abschlagszahlungen. Jede Zahlung ist
  eine eigene Buchung; der Beleg bleibt offen, bis die Summe der Zahlungen den Belegbetrag
  erreicht. Restbetrag bleibt als offener Posten stehen.
- **Eine Zahlung → mehrere Belege:** Sammelüberweisung/-abbuchung. Betrag wird aufgeteilt und
  mehreren OPOS zugeordnet.

Der Belegstatus ergibt sich aus dem Saldo: `bezahlt` (vollständig), `teilbezahlt`
(Saldo > 0), `offen` (keine Zahlung).

### 3.3 Bankbewegung ohne Beleg → direkt buchen

Nicht jeder Bankumsatz hat einen Beleg. Beispiele: Zinsen, Gebühren, Privatentnahmen,
Rückstellungsauflösungen. Diese werden in der Bank-Ansicht direkt auf das passende Konto
gebucht – inkl. Rückstellungen, Abschreibungen, Darlehensbewegungen. Die Software bietet
kontextabhängig die passenden Buchungsmuster an.

## 4. Anlagenverwaltung (neuer Reiter)

| Anlageart | Beispiele |
|---|---|
| **Sachanlagen** | Fahrzeug, Maschine, Büroausstattung, IT-Hardware |
| **Finanzanlagen** | Wertpapiere, Beteiligungen, Darlehen (Forderung) |

*Warenbestand/Umlaufvermögen ist out of scope für v1.*

Je Anlagegut:

- **Erfassen:** Bezeichnung, Anschaffungskosten, Anschaffungsdatum, Nutzungsdauer, AfA-Methode
  (linear/degressiv).
- **Zugangsbuchung:** beim Anlegen automatisch – SOLL Anlagekonto · HABEN Bank/Verbindlichkeit
  (je nach Zahlung).
- **Abschreibung (AfA):** periodisch – SOLL AfA-Aufwand · HABEN Anlagekonto (kostengleich).
  Gekoppelt an den **Festschreibungszeitraum** (siehe 4.1).
- **Abgang:** Verkauf/Verschrottung – Restbuchwert, evtl. Gewinn/Verlust.

### 4.1 AfA-Kopplung an Festschreibung

AfA und Rückstellungen sind typische **Abschlussbuchungen zum Bilanzstichtag** (§ 253 Abs. 3 HGB,
§ 7 EStG für AfA; § 249 HGB für Rückstellungen). Sie werden am Jahresende gebucht, i. d. R. im
Q1 des Folgejahres, sobald alle Belege des abgelaufenen Jahres vollständig vorliegen. Bei
Anschaffung während des Jahres wird die AfA zeitanteilig (monatsgenau) berechnet.

Die AfA wird **nicht** vollautomatisch im Hintergrund gebucht. Stattdessen ist die
**jährliche Festschreibung** (Jahresabschluss) der Triggerpunkt:

1. Vor der **jährlichen** Festschreibung prüft die Software, ob für alle Anlagegüter die fälligen
   AfA-Buchungen des Jahres noch ausstehen.
2. Fehlende AfA-Buchungen werden angezeigt und können **mit einem Klick erzeugt** werden
   (Vorschau der Buchungssätze, dann Freigabe).
3. Erst wenn alle AfA-Buchungen vorhanden sind (oder bewusst übersprungen wurden), kann das
   Jahr festgeschrieben werden.

Bei **monatlichen/quartalsweisen** Festschreibungen (USt-Zeitraum) ist die AfA nicht fällig und
wird nicht geprüft – sie ist eine Jahresendbuchung, keine laufende Geschäftsvorfall-Buchung.
Gleiches gilt für Rückstellungen: sie werden zum Bilanzstichtag gebildet, nicht monatlich.

So wird verhindert, dass ein Jahr versehentlich ohne AfA gesperrt wird – ohne dass der Nutzer
von unerwarteten automatischen Buchungen überrascht wird.

## 5. Eröffnungsbilanz & Stammkapital

Die Gründung einer Kapitalgesellschaft (GmbH/UG) erfordert besondere Buchungssätze, die die
Software geführt anbietet. Rechtsgrundlage: § 272 Abs. 1 HGB (offener Abzug nicht eingeforderter
Einlagen vom gezeichneten Kapital) und § 19 Abs. 2 GmbHG (Einfordern nur per
Gesellschafterbeschluss).

### 5.1 Eröffnungsbilanzkonto (EBK 9000)

Das EBK (SKR04 Konto 9000, Saldovortragskonten) ist das Hilfskonto, über das alle
Anfangsbestände eröffnet werden. Es muss nach allen Eröffnungsbuchungen den Saldo null aufweisen.

### 5.2 Stammkapital anlegen (Volleinzahlung)

Beispiel: 25.000 € Stammkapital, voll eingezahlt.

| # | Buchungssatz | SKR04 |
|---|---|---|
| 1 | EBK an Gezeichnetes Kapital | 9000 an 2900 · 25.000 € |
| 2 | Bank an EBK | 1800 an 9000 · 25.000 € |

### 5.3 Teileinzahlung (z. B. nur die Hälfte)

Beispiel: 25.000 € Stammkapital, 12.500 € eingezahlt. Der Rest ist **nicht eingeforderte
ausstehende Einlage** (§ 272 Abs. 1 Satz 3 HGB: offen vom gezeichneten Kapital abzusetzen).

| # | Buchungssatz | SKR04 |
|---|---|---|
| 1 | EBK an Gezeichnetes Kapital (voll) | 9000 an 2900 · 25.000 € |
| 2 | Bank an EBK (eingezahlt) | 1800 an 9000 · 12.500 € |
| 3 | Ausstehende Einlagen (nicht eingefordert) an EBK | 2910 an 9000 · 12.500 € |

Bilanzausweis: Gezeichnetes Kapital 25.000 €, abzüglich nicht eingeforderte Einlagen 12.500 €,
= eingefordertes Kapital 12.500 €.

### 5.4 Einfordern ausstehender Einlagen (Gesellschafterbeschluss)

Wird der noch ausstehende Betrag per Gesellschafterbeschluss eingefordert, wandelt er sich von
einer „nicht eingeforderten" (Passivausweis, Kapitalkorrektur) in eine **eingeforderte Forderung**
gegen den Gesellschafter (Aktivseite) um.

| # | Buchungssatz | SKR04 |
|---|---|---|
| 1 | Ausstehende Einlagen, eingefordert (Forderung) an Ausstehende Einlagen, nicht eingefordert | 1298 an 2910 · 12.500 € |

Der Betrag wechselt von Konto 2910 (Passiva, Abzug vom Eigenkapital) auf Konto 1298 (Aktiva,
Forderung). Das gezeichnete Kapital (2900) bleibt unberührt.

### 5.5 Einzahlung nach Einfordern

Zahlt der Gesellschafter nach dem Beschluss den eingeforderten Betrag, wird die Forderung
ausgebucht:

| # | Buchungssatz | SKR04 |
|---|---|---|
| 1 | Bank an Ausstehende Einlagen, eingefordert | 1800 an 1298 · 12.500 € |

### 5.6 Zusammenführung der Konten

| Bedeutung | SKR04 | Bilanzseite |
|---|---|---|
| Gezeichnetes Kapital (Stammkapital) | 2900 | Passiva (Eigenkapital) |
| Ausstehende Einlagen, **nicht** eingefordert | 2910 | Passiva (offener Abzug vom Kapital) |
| Ausstehende Einlagen, eingefordert (Forderung) | 1298 | Aktiva (Forderungen) |
| Eröffnungsbilanzkonto / Saldenvortrag | 9000 | Hilfskonto (Saldo null) |

Die Software bietet diese Vorgänge als geführten Workflow „Gründung / Eröffnungsbilanz" an, der
den Nutzer durch Kapitalhöhe, Einzahlungsstand und ggf. späteres Einfordern führt – mit
korrekter Unterscheidung der Konten 2910 vs. 1298 und korrektem Bilanzausweis.

### 5.7 Export der Eröffnungsbilanz

Nach Abschluss des Wizards wird die Eröffnungsbilanz als **PDF herunterladbar** sein. Das
entspricht dem bestehenden Ansatz für Ausgangsrechnungen, die ebenfalls via Typst zu PDF
gerendert werden (`internal/service/invoice_service.go`). Die Eröffnungsbilanz ist das formelle
Ergebnis des Gründungsvorgangs und sollte als Dokument vorliegen – zur Vorlage beim
Notar/Handelsregister und zur eigenen Ablage.

## 6. Kontenzuordnung: fachliche Gruppierung → deterministisches Mapping

Die Kontenzuordnung ist **deterministisch** – keine Lernfunktion. Der Nutzer wählt auf der
Nutzerseite eine **vereinfachte, fachliche Gruppierung**, das Backend mappt deterministisch auf
die konkreten SKR04-Konten. Eine Auswahl kann dabei mehrere Konten gleichzeitig belegen (z. B.
Erlöskonto + USt-Passivkonto).

### 6.1 Prinzip

```
Nutzer wählt "Gruppe"          Backend mappt deterministisch
─────────────────────          ──────────────────────────────
z. B. "Erlöse 19 %"      ──►   8400 Erlöse 19% USt (Ertrag)
                             + 3804 USt 19% (Passivkonto)
                             + 1200 Bank ODER 1400 Forderung (je nach bezahlt/offen)
```

Der Nutzer denkt in **Gruppen** („Büromaterial", „Miete", „Erlöse 19 %", „Fahrzeug"), nicht in
Kontonummern. Jede Gruppe ist ein vordefinierter Satz aus einem Hauptkonto (Aufwand/Ertrag)
plus ggf. Steuermapping plus Zahlungsgegenkonto.

### 6.2 Mapping-Tabelle (Beispiele)

| Nutzer wählt (Gruppe) | Zahlung | Software bucht (SOLL · HABEN) |
|---|---|---|
| „Büromaterial" + offen | nein | Aufwand + Vorsteuer · Verbindlichkeit |
| „Büromaterial" + bar | ja | Aufwand + Vorsteuer · Bank |
| „Erlöse 19 %" + offen | nein | Forderung · Erlös + USt |
| „Erlöse 19 %" + bar | ja | Bank · Erlös + USt |
| Zahlung → offener Eingangsbeleg | — | Verbindlichkeit · Bank |
| Zahlung → offene Ausgangsrechnung | — | Bank · Forderung |
| Zahlung ohne Beleg → „Rückstellung" | — | Rückstellung · Bank |
| Zahlung ohne Beleg → „Darlehen" | — | Darlehen · Bank (Tilgung) bzw. Zinsaufwand · Bank (Zins) |
| Anlagegut anlegen | je nach Zahlung | Anlagekonto · Bank oder Verbindlichkeit |
| AfA (vor Festschreibung) | — | AfA-Aufwand · Anlagekonto |
| Eröffnungsbilanz / Stammkapital | — | EBK · Gezeichnetes Kapital, Bank, Ausstehende Einlagen (siehe Abschnitt 5) |

### 6.3 Plausibilitätsprüfung

Das Backend prüft deterministisch und warnt bei typischen Verwechslungen: Erstattung als Ertrag,
Tilgung als Aufwand, Aktiv/Passiv vertauscht. Diese Prüfungen sind fest hinterlegt, nicht
heuristisch.

## 7. Durchgespielte Geschäftsvorfälle

Um die Nutzer-Struktur zu validieren, hier die typischen Vorfälle Ende-zu-Ende. Jeder Vorfall
zeigt: was der Nutzer tut (Nutzerseite) und was gebucht wird (Kontenseite). Ziel ist, dass
jeder Vorfall durch die gleiche, einfache Nutzer-Interaktion abgedeckt ist.

### 7.1 Lieferantenrechnung (Dienstleistung, auf Ziel)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Beleg erfassen | Belegsammlung: PDF hochladen, Lieferant wählen, Gruppe „Dienstleistungen/Fremdarbeiten", Betrag netto 1.000 €, 19 % USt, „offen" | SOLL 4400 Fremdleistungen 1.000 € + SOLL 1576 abzuf. VSt 190 € · HABEN 1600 Verb. a. LuL 1.190 € |
| 2. Zahlung eingehend | Bank & Zahlungen: Umsatz aus CAMT, offenem Beleg zuordnen | SOLL 1600 Verb. a. LuL 1.190 € · HABEN 1800 Bank 1.190 € |

OPOS: Beleg ist offen bis Zahlung zugeordnet. Status wechselt von `offen` → `bezahlt`.

### 7.2 Ausgangsrechnung (Dienstleistung, auf Ziel)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Rechnung erstellen | Ausgangsrechnungen: Kunde wählen, Position „Beratungsleistung", Gruppe „Erlöse 19 %", 2.000 € netto, 19 % USt, „offen" | SOLL 1200 Ford. a. LuL 2.380 € · HABEN 8400 Erlöse 19 % 2.000 € + HABEN 3804 USt 19 % 380 € |
| 2. Zahlung eingehend | Bank & Zahlungen: Umsatz offenem Beleg zuordnen | SOLL 1800 Bank 2.380 € · HABEN 1200 Ford. a. LuL 2.380 € |

### 7.3 Barzahlung (direkt, ohne OPOS)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Beleg erfassen | Belegsammlung: Quittung hochladen, Gruppe „Büromaterial", 50 € netto, 19 % USt, „bar bezahlt" | SOLL 4930 Büromaterial 50 € + SOLL 1576 abzuf. VSt 9,50 € · HABEN 1800 Bank 59,50 € |

Kein OPOS, Beleg sofort `bezahlt`. Eine Buchung, kein Zwei-Schritt.

### 7.4 Rückstellung am Jahresende (nicht-bar, Abschlussbuchung)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Rückstellung bilden | Journal/Belegsammlung: „Rückstellung", Gruppe z. B. „Rückstellung Gewährleistung", 5.000 €, „nicht-bar / ohne Beleg" | SOLL 4780 Gewährleistungsaufwand 5.000 € · HABEN 0970 Rückstellungen für Gewährleistungen 5.000 € |

Kein Bankumsatz, keine Zahlungszuordnung. Die Buchungsstelle ist „nicht-bar" – direkte Freigabe.
Typische Abschlussbuchung zum Bilanzstichtag (§ 249 HGB), i. d. R. im Q1 des Folgejahres
gebucht.

### 7.5 Rückstellungsauflösung (mit späterer Zahlung)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Inanspruchnahme | Bank & Zahlungen: Umsatz ohne Beleg → „Rückstellungsauflösung" wählen | SOLL 0970 Rückstellung 5.000 € · HABEN 1800 Bank 5.000 € |

Rückstellung wird durch Zahlung aufgebraucht. Übersteigt die Rückstellung die tatsächliche
Zahlung, wird der Rest gewinnmindernd aufgelöst (SOLL Rückstellung · HABEN Ertrag).

### 7.6 Abschreibung (AfA, aus Anlagenverwaltung, Jahresendbuchung)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Anlage erfassen | Anlagen: „Fahrzeug", 30.000 €, Nutzungsdauer 6 Jahre, linear | SOLL 0820 Fahrzeug 30.000 € · HABEN 1800 Bank 30.000 € (Zugang) |
| 2. AfA am Jahresende | Anlagen: vor jährlicher Festschreibung „AfA erzeugen", 5.000 € p. a. | SOLL 6310 AfA auf Sachanlagen 5.000 € · HABEN 0820 Fahrzeug 5.000 € |

Restbuchwert nach Jahr 1: 25.000 €. Das System plant die AfA vor und zeigt den verbleibenden
Buchwert. AfA ist eine Abschlussbuchung zum Bilanzstichtag (§ 253 Abs. 3 HGB, § 7 EStG), zeitanteilig
bei Anschaffung während des Jahres. Sie wird nur vor der jährlichen Festschreibung geprüft (siehe 4.1).

### 7.7 Darlehen (Aufnahme + Tilgung + Zins)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Auszahlung | Bank & Zahlungen: Umsatz → „Darlehensaufnahme", 50.000 € | SOLL 1800 Bank 50.000 € · HABEN 0630 Darlehen 50.000 € |
| 2. Tilgung + Zins | Bank & Zahlungen: Umsatz → „Darlehensrate", Betrag 1.000 €, Software fragt Tilgungsplan ab: 800 € Tilgung, 200 € Zins | SOLL 0630 Darlehen 800 € + SOLL 2120 Zinsaufwand 200 € · HABEN 1800 Bank 1.000 € |

Die Software splittet die Rate automatisch in Tilgung und Zinsanteil auf Basis des hinterlegten
Tilgungsplans.

### 7.8 Erstattung / Gutschrift (Lieferant)

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. Gutschrift erfassen | Belegsammlung: „Gutschrift", Gruppe „Dienstleistungen/Fremdarbeiten", 1.000 € netto, 19 % USt, Bezug zur Ursprungsrechnung | SOLL 1600 Verb. a. LuL 1.190 € · HABEN 4400 Fremdleistungen 1.000 € + HABEN 1576 abzuf. VSt 190 € (Rückbuchung der Vorsteuer) |
| 2. Ausgleich | Bank & Zahlungen: Umsatz der Gutschrift zuordnen | SOLL 1600 Verb. a. LuL 1.190 € · HABEN 1800 Bank 1.190 € |

Gutschrift ist die Spiegelbild zur Ursprungsrechnung: Aufwand und Vorsteuer werden retour
gebucht. Bezugsnummer zur Ursprungsrechnung sichert die Nachvollziehbarkeit.

### 7.9 Jahresabschluss (Saldenvortrag & Abschlussbuchungen)

*Hinweis: Der Jahresabschluss ist in diesem Konzept nur skizziert – er fällt in den
Steuer-/Auswertungs-Scope, der außen vor bleibt (siehe Abschnitt 1). Die konkreten
Abschlussbuchungen müssen über separate Eingabemasken erfolgen und werden später ergänzt.*

| Schritt | Nutzerseite | Kontenseite (SKR04) |
|---|---|---|
| 1. AfA prüfen | Anlagen: vor Festschreibung fehlende AfA erzeugen | siehe 7.6 |
| 2. Rückstellungen prüfen | Journal: offene Rückstellungen prüfen/bilden | siehe 7.4 |
| 3. Festschreiben | Fristen & Termine: Periode (Jahr) festschreiben | Hash-Chain-Head wird mit RFC-3161-Zeitstempel fixiert |
| 4. Saldenvortrag | separate Abschluss-Maske: EBK 9000 nimmt alle Salden des Vorjahres auf | SOLL Aktiva (Bestandskonten) · HABEN 9000 EBK; SOLL 9000 EBK · HABEN Passiva (Bestandskonten) |
| 5. Erfolgskonten abschließen | separate Abschluss-Maske: Aufwands- und Ertragskonten werden auf das Gewinn- und Verlustkonto (GuV, 9890) gebucht und mit null abgeschlossen | SOLL 9890 GuV · HABEN Ertragskonten; SOLL Aufwandskonten · HABEN 9890 GuV |
| 6. Eigenkapital via Saldenvortrag: | SOLL 9890 GuV · HABEN 9000 EBK (Gewinn) bzw. SOLL 9000 EBK · HABEN 9890 GuV (Verlust) | Saldo des GuV-Kontos wird über das EBK ins Eigenkapital (2900/2920) vorgetragen |

Die Abschlussbuchungen (Saldenvortrag, GuV-Abrechnung, Eigenkapitalvortrag) werden nicht über den
Beleg-/Zahlungsflow erfasst, sondern über separate Abschluss-Masken. Buchfink stellt die Salden
bereit; die eigentliche Abschlussbuchung ist ein separater, später zu ergänzender Bereich.

### 7.10 Zusammenfassung: einheitliche Nutzer-Aktionen

Alle obigen Vorfälle werden durch max. drei Nutzer-Aktionen abgedeckt:

1. **Etwas erfassen** (Beleg/Rechnung/Rückstellung/Anlage) → sofortige Buchung
2. **Zahlung zuordnen** (falls vorhanden) → OPOS-Ausgleich
3. **Freigeben** (für nicht-bar oder periodische Vorfälle wie AfA/Rückstellung/Abschluss)

Die Kontenwahl ergibt sich deterministisch aus der gewählten fachlichen Gruppe + „bezahlt/offen".
Der Nutzer wählt nie direkt eine SOLL/HABEN-Kombination.

## 8. Beleg- & Dateiverwaltung

- **Vorschau:** PDF-Preview für Belege & Rechnungen; Bildvorschau (JPG/PNG/Scan/Foto) für
  Eingangsbelege.
- **Ablage:** sortiert nach Jahr/Art (`belege/<jahr>/eingang/…`, `…/ausgang/…`), Originaldatei
  unverändert (GoBD), Hash-gesichert.

## 9. Offene Designentscheidungen

### Entschieden

- **Rechnungsverbund-Modell:** eigener Gruppierungs-Entity (nicht nur Referenz). Begründung: für
  die Nutzer-Abstraktionsschicht ist ein eigener Verbund-Entity nötig, der Abschläge und
  Schlussrechnung zusammenfasst und den Gesamtfortschritt (geleistet/offen) sauber darstellt.
- **AfA-Automatik:** keine Vollautomatik. AfA wird an den Festschreibungszeitraum gekoppelt –
  vor dem Sperren einer Periode werden fehlende AfA-Buchungen angezeigt und per Freigabe erzeugt
  (siehe 4.1).
- **AfA-Methoden:** linear, degressiv und Sonderabschreibungen. Linear als Standard; degressiv
  nach dem Investitionssofortprogramm 2025 (3-faches der linearen, max. 30 %, für Anschaffungen
  01.07.2025–31.12.2027); zusätzlich Sonderabschreibung § 7g Abs. 5 EStG (bis zu 40 % für KMU,
  5-Jahres-Begünstigungszeitraum). Übergang degressiv → linear zulässig.
- **Kontenvorschläge:** deterministisch, keine Lernfunktion. Der Nutzer wählt fachliche Gruppen
  (z. B. „Büromaterial", „Erlöse 19 %"); das Backend mappt fest auf SKR04-Konten (inkl. Steuer-
  und Zahlungsgegenkonten). Siehe Abschnitt 6.
- **Gruppen-Katalog:** sehr vollständig. Das Projekt enthält bereits das vollständige SKR04-2026-
  Mapping (`internal/accounting/skr04_2026.json`, 1.855 Konten, davon 1.602 aktiv, 286 Erlöskonten,
  545 Aufwandskonten – jeweils mit category, subcategory, account_type und HGB-Position). Die
  Erlöskonten/Aufwandskonten aus diesem Mapping bilden direkt die Grundlage für den
  fachlichen Gruppenkatalog.
- **Eröffnungsbilanz-Workflow:** geführter Wizard. Nur verfügbar, solange noch keine Buchungen
  vorhanden sind. Optional direkt in den Setup-Workflow integrierbar (Option „neugegründetes
  Unternehmen" führt durch Stammkapital, Einzahlungsstand und Eröffnungsbilanz).
- **Warenbestand:** out of scope für v1.
- **Steuern & Auswertungen außen vor:** Steuerliche Auswertungen (USt-Voranmeldung,
  USt-Erklärung, ZM) und der Jahresabschluss (Saldenvortrag, GuV-Abrechnung, Bilanzierung) sind
  nicht Teil des Beleg-/Zahlungsflows. Sie benötigen eigene Eingabemasken und separate Logik und
  werden später ergänzt. Dieses Konzept deckt nur die Erfassung laufender Geschäftsvorfälle.

### Noch offen

*(aktuell keine offenen Entscheidungen)*
