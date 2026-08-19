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

## 2. Navigation (Reiter)

| Reiter | Nutzer-Sicht | Konten-Sicht |
|---|---|---|
| **Belegsammlung** | Eingangsbelege erfassen (Upload PDF/Bild), Belegdaten + Lieferant | sofortige Buchung auf Aufwand + Verbindlichkeit (falls offen) |
| **Ausgangsrechnungen** | Rechnung erstellen (ZUGFeRD), Kunde + Positionen | sofortige Buchung auf Forderung + Ertrag (falls offen) |
| **Bank & Zahlungen** | Bankumsätze importieren, offene Posten ausgleichen | Zahlungsbuchung auf Bank + OPOS-Ausgleich |
| **Anlagen** | Vermögensgüter verwalten (Fahrzeug, Warenbestand, Finanzanlagen), AfA | Anlagekonto + automatische Abschreibungsbuchung |
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
| **Umlaufvermögen** | Warenbestand (Vorräte) |
| **Finanzanlagen** | Wertpapiere, Beteiligungen, Darlehen (Forderung) |

Je Anlagegut:

- **Erfassen:** Bezeichnung, Anschaffungskosten, Anschaffungsdatum, Nutzungsdauer, AfA-Methode
  (linear/degressiv).
- **Zugangsbuchung:** beim Anlegen automatisch – SOLL Anlagekonto · HABEN Bank/Verbindlichkeit
  (je nach Zahlung).
- **Abschreibung (AfA):** automatisch periodisch (z. B. monatlich/jährlich) – SOLL AfA-Aufwand ·
  HABEN Anlagekonto (kostengleich). Die Software plant die AfA voraus und bucht sie
  automatisch zum Periodenende (freigabepflichtig).
- **Abgang:** Verkauf/Verschrottung – Restbuchwert, evtl. Gewinn/Verlust.

## 5. Kontenzuordnung: Nutzer-Eingabe → Buchung

Der Nutzer wählt **fachlich**, die Software **buchhalterisch**. Aus wenigen Eingaben leitet das
Backend das vollständige SOLL/HABEN-Muster ab:

| Nutzer wählt | Software bucht |
|---|---|
| Eingangsbeleg + „noch offen" | Aufwand + Vorsteuer / Verbindlichkeit |
| Eingangsbeleg + „bar bezahlt" | Aufwand + Vorsteuer / Bank |
| Ausgangsrechnung + „noch offen" | Forderung / Ertrag + USt |
| Ausgangsrechnung + „bar bezahlt" | Bank / Ertrag + USt |
| Zahlung → offener Eingangsbeleg | Verbindlichkeit / Bank |
| Zahlung → offene Ausgangsrechnung | Bank / Forderung |
| Zahlung ohne Beleg → „Rückstellung" | Rückstellung / Bank |
| Zahlung ohne Beleg → „Darlehen" | Darlehen / Bank (Tilgung) bzw. Zinsaufwand / Bank (Zins) |
| Anlagegut anlegen | Anlagekonto / Bank oder Verbindlichkeit |
| AfA (automatisch) | AfA-Aufwand / Anlagekonto |

Der Nutzer muss keine Kontonummern kennen. Er wählt: Belegart, Gegenpartei, „bezahlt/offen",
Betrag, ggf. Steuersatz. Das Backend wählt die konkreten SKR04-Konten, prüft Plausibilität und
warnt bei typischen Verwechslungen (Erstattung als Ertrag, Tilgung als Aufwand, Aktiv/Passiv
vertauscht).

## 6. Sonderfälle

- **Abschlags-/Teilrechnungen:** Rechnungsverbund fasst Abschläge + Schlussrechnung zusammen;
  jeder Abschlag eigener Beleg mit eigener (Teil-)Zahlung.
- **Erstattungen/Gutschriften:** umgekehrter Vorzeichen-Beleg; Umkehrbuchung, GoBD-Storno bleibt
  gewahrt.
- **Darlehen:** Auszahlung SOLL Bank/HABEN Darlehen; Tilgung SOLL Darlehen/HABEN Bank;
  Zinsanteil SOLL Zinsaufwand/HABEN Bank – Software fragt Tilgungsplan ab und splittet.
- **Rückstellungen:** nicht-bar, periodisch – SOLL Aufwand/HABEN Rückstellung; Auflösung über
  Bankbewegung ohne Beleg.
- **Abschreibungen:** automatisch aus Anlagenverwaltung (siehe Abschnitt 4).

## 7. Beleg- & Dateiverwaltung

- **Vorschau:** PDF-Preview für Belege & Rechnungen; Bildvorschau (JPG/PNG/Scan/Foto) für
  Eingangsbelege.
- **Ablage:** sortiert nach Jahr/Art (`belege/<jahr>/eingang/…`, `…/ausgang/…`), Originaldatei
  unverändert (GoBD), Hash-gesichert.

## 8. Offene Designentscheidungen

1. **Rechnungsverbund-Modell:** eigener Gruppierungs-Entity vs. reine Referenz zwischen Belegen.
2. **AfA-Automatik:** monatlich vs. jährlich; Freigabe-Pflicht pro Periode vs. vollautomatisch.
3. **Automatikgrad der Kontenvorschläge:** rein regelbasiert (v1) vs. Lernfunktion aus
   Buchungshistorie (z. B. „dieser Lieferant landet immer auf Konto 4920").
4. **Warenbestand:** echter Bestandskonto-Ansatz mit Bewertung (FIFO/ Durchschnitt) vs.
   periodische Bestandsbuchung.
