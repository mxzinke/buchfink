# Buchfink – Konzept: Belegworkflow & geführte Verbuchung

Status: Entwurf / Konzept (noch nicht implementiert)
Letzte Aktualisierung: 2026-08-19

## 1. Ausgangslage & Ziel

Belege und Rechnungen sind heute nur Anhänge an eine bereits fertige Buchung. Es fehlt ein
eigener Lebenszyklus: ein Beleg existiert unabhängig davon, ob schon gebucht wurde. Damit ist
auch kein sauberer „Beleg → offen → gebucht"-Prozess möglich.

**Ziel:** Ein einheitlicher, geführter Prozess, der jeden Geschäftsvorfall gleich führt – egal ob
Eingangsbeleg, Ausgangsrechnung oder buchungsinterner Vorgang. Die Software leitet den Nutzer,
vermeidet Konten-Verwechslungen und trennt sauber: *Beleg erfassen* → *Buchung vorbereiten* →
*Zahlung zuordnen* → *Verbuchen*.

## 2. Zwei Perspektiven

| Perspektive | Anliegen |
|---|---|
| **Nutzer** | Geschäftsvorfall schnell & immer gleich anlegen: Beleg → Zahlung (oder nicht) → Verbuchung (SOLL/HABEN via Konten) |
| **Kontenwelt** | Korrekte Zuordnung auf Ertrags-, Aufwands-, Aktiv- und Passivkonten; Verwechslungen vermeiden |

Die Brücke ist die **Buchungsstelle**: eine vorbereitete, noch nicht ausgeführte Buchungsvorlage
mit Kontenvorschlag. Sie übersetzt die Nutzer-Perspektive in die Konten-Perspektive, bevor
gebucht wird.

## 3. Navigation (Reiter)

| Reiter | Inhalt |
|---|---|
| **Eingangsbelege** | Lieferantenbelege: erfassen, Buchungsstelle vorbereiten, Status offen/gebucht |
| **Ausgangsrechnungen** | Rechnungen an Kunden: erstellen, ZUGFeRD, Status offen/gebucht |
| **Bank & Zahlungen** | CAMT-Import, offene Bankumsätze, Zuordnung zu offenen Belegen/Rechnungen |
| **Journal** | Alle gebuchten Geschäftsvorfälle (SOLL/HABEN, Belegnr., Hash-Chain) |

Bestehende Reiter (Kontakte, Konten, GuV & Bilanz, Fristen, E-Bilanz, Sicherheit) bleiben bestehen.
„Eingangsbelege" und „Ausgangsrechnungen" sind der Einstiegspunkt für neue Vorfälle; „Bank &
Zahlungen" ist der Zuordnungsort; „Journal" ist das Resultat.

## 4. Entitäten & Begriffe

- **Beleg** – das Dokument (Eingangsbeleg vom Lieferanten ODER Ausgangsrechnung vom Nutzer
  erstellt). First-class-Objekt mit eigenem Lebenszyklus, *unabhängig vom Buchungsstatus*.
- **Geschäftsvorfall** – der buchhalterische Vorgang, der aus einem Beleg entsteht
  (z. B. „Büromaterial gekauft", „Abschlag gestellt").
- **Buchungsstelle** – vorbereitete, noch nicht gebuchte Soll/Haben-Vorlage mit Kontenvorschlag.
- **Buchungssatz** – die ausgeführte, hash-gekettte Buchung (unveränderbar, Storno-only).

## 5. Der einheitliche Flow (Lebenszyklus)

```
Erfassen ──► Buchungsstelle vorbereiten ──► Zahlung zuordnen? ──► Verbuchen
 (1)              (2)                       (3)                    (4)
```

1. **Erfassen** – Beleg hochladen (PDF/Bild) oder Rechnung erstellen. Metadaten: Datum,
   Gegenpartei, Betrag, Belegnummer.
2. **Buchungsstelle vorbereiten** – SOLL/HABEN-Konten wählen (mit Vorschlag), Betrag & Steuer
   prüfen. **Noch nicht buchen.**
3. **Zahlung zuordnen** (optional) – passenden Bankumsatz matchen. Ohne Zuordnung → Status
   **offen** (später aus der Bank-Ansicht buchbar).
4. **Verbuchen** – wenn Beleg korrekt erfasst UND Zahlung zugeordnet (oder bei nicht-bar bewusst
   freigegeben): der Geschäftsvorfall wird auf die Konten gebucht. Buchungssatz entsteht,
   Hash-Chain wird fortgeschrieben.

Status eines Geschäftsvorfalls:

| Status | Bedeutung |
|---|---|
| `entwurf` | erfasst, Buchungsstelle noch offen |
| `bereit` | Buchungsstelle vorbereitet, wartet auf Zahlung |
| `offen` | bereit, ohne Zahlungszuordnung (wartet) |
| `gebucht` | Buchungssatz erzeugt |
| `storniert` | korrigiert (GoBD-Storno) |

## 6. Zwei Buchungspfade

Nicht jeder Geschäftsvorfall hat eine Zahlung.

| Pfad | Trigger | Beispiele |
|---|---|---|
| **A – mit Zahlung** | Bankumsatz zugeordnet + Beleg korrekt | Eingangsrechnung, Ausgangsrechnung, Erstattung, Darlehensauszahlung/-tilgung, Zins-/Miet-/Provisionseingang |
| **B – ohne Zahlung** | Bewusste Freigabe (kein Bankumsatz nötig) | Abschreibung (AfA), Rückstellung, periodische Buchung, Eigenbeleg |

Pfad B wird an der Buchungsstelle als „nicht-bar" markiert und kann direkt freigegeben werden –
ohne die Bank-Ansicht. So bleibt der Flow für alle Vorfälle gleich; nur der Zahlungsschritt
entfällt bzw. wird durch „Freigeben" ersetzt.

## 7. Sonderfälle & Belegarten

### Abschlags-/Teilrechnungen
Ein **Rechnungsverbund** fasst mehrere Belege zusammen: Abschlag 1, Abschlag 2, …,
Schlussrechnung. Jeder Abschlag ist ein eigener Beleg mit eigener (Teil-)Zahlung. Die
Schlussrechnung fasst zusammen und gleicht ab. So bleiben Teilbeträge und offene Reste sauber
nachvollziehbar.

### Erstattungen / Gutschriften
Eigener Belegtyp mit umgekehrtem Vorzeichen. Bei Ausgangsrechnungen: Storno/Gutschrift. Bei
Eingangsbelegen: Lieferantengutschrift. Verbuchung als Umkehrbuchung; das GoBD-Storno-Prinzip
bleibt gewahrt.

### Darlehen
- Auszahlung: SOLL Bank, HABEN Darlehen (Passivkonto)
- Tilgung: SOLL Darlehen, HABEN Bank
- Zinsanteil: SOLL Zinsaufwand, HABEN Bank (anteilig)

Die Software fragt Tilgungsplan/Anteile ab und splittet automatisch in Tilgung und Zins.

### Rückstellungen
Buchungsstelle „nicht-bar": SOLL Aufwand, HABEN Rückstellung. Periodisch, ohne Zahlung. Späterer
Verbrauch/Auflösung ist ein eigener Geschäftsvorfall.

### Verbindlichkeiten (offene Posten Lieferanten)
Ein Eingangsbeleg erzeugt eine Verbindlichkeit aus Lieferung & Leistung; sie bleibt als offener
Posten (OPOS), bis die Zahlung zugeordnet ist. Bei Ausgangsrechnungen entsprechend die Forderung
beim Kunden.

### Abschreibungen
Periodisch, „nicht-bar": SOLL AfA-Aufwand, HABEN Anlagevermögen (kostengleich). Die Buchungsstelle
fragt Anlagegut + AfA-Methode ab.

## 8. Geführte Verbuchung (Verwechslungen vermeiden)

Aus Belegart + Gegenpartei + Betrag leitet die Software die **Buchungsstelle** (SOLL/HABEN-Muster)
ab und schlägt konkrete Konten vor:

| Geschäftsvorfall | SOLL | HABEN |
|---|---|---|
| Eingangsrechnung (bar) | Aufwand | Bank |
| Eingangsrechnung (Ziel) | Aufwand | Verbindlichkeit |
| Ausgangsrechnung (bar) | Bank | Ertrag |
| Ausgangsrechnung (Ziel) | Forderung | Ertrag |
| Zahlungseingang (Ziel) | Bank | Forderung |
| Zahlungsausgang (Ziel) | Verbindlichkeit | Bank |
| Abschreibung | AfA-Aufwand | Anlagevermögen |
| Rückstellung | Aufwand | Rückstellung |
| Darlehensaufnahme | Bank | Darlehen |
| Zinsertrag | Bank | Zinsertrag |

Der Nutzer bestätigt oder korrigiert. Plausibilitätsprüfungen warnen vor typischen Fehlern
(z. B. Erstattung als Ertrag, Tilgung als Aufwand, Aktiv-/Passivkonto vertauscht).

## 9. Belege & Dateiverwaltung

- **Vorschau:** PDF-Preview für Eingangsbelege & Ausgangsrechnungen; Bildvorschau
  (JPG/PNG/Scan/Foto) für Belege.
- **Ablage:** sortiert nach Jahr/Art, z. B. `belege/<jahr>/eingang/…` bzw. `…/ausgang/…`.
  Originaldatei bleibt unverändert (GoBD), Hash-gesichert.
- Der Hash sichert die Dateiintegrität; der Pfad ist ein (verschlüsseltes) Metadatum in der DB.

## 10. Offene Designentscheidungen

1. **OPOS-Timing:** Verbindlichkeit/Forderung schon bei Beleg-Erfassung verbuchen (echtes OPOS,
   zwei Buchungssätze: Rechnung + später Zahlung) oder erst bei Zahlungsmatch in einer Buchung?
   Empfehlung: echte OPOS-Zwei-Schritt-Buchung, damit offene Posten sichtbar bleiben. Das passt
   dann so zusammen: „Buchungsstelle vorbereiten" = Rechnungsbuchung bereitstellen (OPOS entsteht
   erst bei Verbuchung), „Zahlung zuordnen" = zweite Buchung (Ausgleich).
2. **Rechnungsverbund-Modell:** eigener Gruppierungs-Entity oder reine Referenz zwischen Belegen?
3. **Automatikgrad der Buchungsvorschläge:** rein regelbasiert (v1) vs. Lernfunktion aus
   bereits gebuchten Vorfällen.
