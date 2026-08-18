# Buchfink – Sicherheitskonzept (Verschlüsselung & Integrität)

Status: Entwurf / Konzept (noch nicht implementiert)
Letzte Aktualisierung: 2026-08-18

Dieses Dokument beschreibt das Zielkonzept für Vertraulichkeit (Verschlüsselung
at-rest) und Integrität/Nachweisbarkeit (Hashkette + Zeitstempel) in Buchfink.
Es ersetzt den bisherigen, faktisch ungenutzten Zertifikats-/Signaturansatz.

## 1. Ausgangslage (Ist-Zustand)

- Es wird pro Mandant ein Ed25519-Keypair + selbstsigniertes X.509-Zertifikat
  erzeugt (`internal/security/cert.go`), aber **nie wieder geladen oder genutzt**.
  Der Key-Pfad wird beim Anlegen sogar verworfen
  (`internal/wailsbridge/app_service.go:237`, `certPath, _, err := ...`).
- **Keine Datenverschlüsselung at-rest:** SQLite wird im Klartext geöffnet
  (`internal/repository/db.go:34`, nur WAL + busy_timeout). Belege liegen als
  Klartextdateien (`ReceiptPath`, `internal/domain/booking.go:30`).
- **Integrität = ungekeyed SHA256-Hashkette** (`internal/accounting/hashchain.go`).
  Kein Geheimnis beteiligt → wer die DB schreiben kann, kann die Kette neu rechnen.
- UI-Label "Sicherheitsschlüssel (Signatur)" (`frontend/src/pages/SettingsPage.tsx:145`)
  ist damit irreführend – es wird nichts signiert.

## 2. Grundsatzentscheidungen

- **Signatur wird gestrichen.** Bei einer lokalen Single-User-App läge der private
  Schlüssel neben den Daten; gegen den einzigen möglichen Manipulator (den Inhaber)
  hebt eine lokale Signatur die Latte kaum. Non-Repudiation entsteht erst durch eine
  **dritte Partei** – diese Rolle übernimmt der RFC-3161-Zeitstempeldienst (Abschnitt 5).
- **Ein Schlüssel für alles.** Genau ein Geheimnis (Nutzer-Passphrase) leitet den
  Datenschlüssel ab. Kein PEM, kein Zertifikat mehr.
- **Hashkette bleibt** – sie wird durch die At-Rest-Verschlüsselung erst *wirksam*
  geschützt (ohne Schlüssel kann niemand die Kette umschreiben).
- **GoBD:** Hashkette + Audit-Log + optionaler Zeitstempel ist ein anerkannter Ansatz.
  Qualifizierte (eIDAS-)Zeitstempel sind **nicht** erforderlich.

## 3. Was entfällt

| Fällt weg | Fundstelle heute |
|---|---|
| Ed25519 Private Key (`buchfink-key.pem`) | `internal/security/cert.go:34,80-104` |
| Ed25519 Public Key (im Cert eingebettet) | `internal/security/cert.go:63` |
| X.509 Zertifikat (`buchfink-cert.pem`) | `internal/security/cert.go:50-77` |
| `CertPath` in Config | `internal/domain/app_config.go:8,18` |
| geplante Signatur-Spalte / `SigningIdentity`-Tabelle | (verworfen) |

`internal/security/cert.go` wird in seiner heutigen Form entfernt und durch die
Schlüsselableitung (Abschnitt 4) ersetzt.

## 4. Vertraulichkeit: Ein-Schlüssel-Modell (Envelope)

```
Passphrase ──Argon2id(salt)──► KEK ──wrap──► DEK (256-bit, zufällig)
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
              verschlüsselt DB           verschlüsselt Belege      schützt damit
              (SQLCipher)                (AES-256-GCM je Datei)    auch die Hashkette
```

- **Ein** Geheimnis (Passphrase), Eingabe beim App-Start (Unlock).
- Auf der Platte nur ein kleines Keyfile: `salt` + `wrapped-DEK` – **kein** PEM/Cert.
- **Passphrase-Wechsel** = nur DEK neu wrappen, keine Neuverschlüsselung der Daten.
- Belege: AES-256-GCM pro Datei unter demselben DEK. `ReceiptHash`
  (`internal/domain/booking.go:29`) bleibt Integritätsnachweis über den Klartext.
- Config `~/.buchfink/config.json`: Modus auf `0600`, enthält keine Geheimnisse.

### Entscheidung: Feld-Verschlüsselung in reinem Go (kein CGo)

Der heutige Treiber `glebarez/sqlite` ist reines Go. Um den Build einfach zu halten
(kein CGo, unkomplizierte Cross-Builds für macOS/Windows/Linux), wird **nicht**
SQLCipher verwendet, sondern **Feld-Verschlüsselung** auf Anwendungsebene per
GORM-Serializer/Hooks mit AES-256-GCM unter dem DEK.

**Bewusst akzeptierter Teilschutz:** Verschlüsselt werden nur Freitext-/PII-Felder.
Felder, die SQL für Filter, Sortierung, Indizes oder Aggregation braucht, bleiben im
Klartext.

| Bleibt Klartext (SQL-relevant) | Wird verschlüsselt (Freitext/PII) |
|---|---|
| `FiscalYear`, `BookingNumber`, `Date`, `ValueDate` | `Description` (Buchungstext) |
| `DebitAccount`, `CreditAccount` | `ReceiptNumber`, `ReceiptPath` |
| `Amount`, `TaxAmount` (für SQL-Summen) | Gegenpartei-Name/-IBAN (Bank-Tx) |
| `PreviousHash`, `EntryHash`, `ReceiptHash` | Kontakt-/Stammdaten-PII, Rechnungs-Freitext |

Konsequenz: Beträge bleiben lesbar (nötig für `CalculateAccountSums`/`CalculateTypeSums`).
Wer stärkeren Schutz will, müsste später auf SQLCipher (CGo) oder App-seitige
Aggregation umstellen.

## 5. Integrität & Nachweis: Hashkette + RFC-3161-Zeitstempel

- **Hashkette** unverändert (`internal/accounting/hashchain.go`):
  Erzeugung in `AccountingService.CreateBooking`
  (`internal/service/accounting_service.go:391`), Prüfung in `VerifyIntegrity`
  (`accounting_service.go:497`).
- **RFC-3161-Zeitstempel** als zusätzlicher, unabhängiger Nachweis ("dieser Zustand
  existierte nachweislich zu Zeitpunkt T", durch eine dritte Partei bestätigt):
  - Es wird **nur der SHA256-Hash** an die TSA gesendet, **nie Buchungsdaten** –
    datenschutzfreundlich, kein Inhalt verlässt den Rechner.
  - **Nicht jede Buchung**, sondern der **Chain-Head bei Jahresabschluss/Export**
    – ein Stempel sichert die gesamte davorliegende Kette mit ab.
  - **Optional & ausfallsicher:** kein Netz → Buchung trotzdem möglich, Stempel wird
    nachgeholt; nie ein harter Blocker.
  - **Token mitspeichern:** vollständiger `.tsr`-Token **+ TSA-Zertifikatskette** in
    der DB → später **offline** verifizierbar, auch wenn die TSA verschwindet.
  - **Bibliothek:** `github.com/digitorus/timestamp` (reines Go, kein CGo).
  - **Default-TSA:** `http://timestamp.digicert.com` (zuverlässig, keine Rate-Limits);
    konfigurierbar, Alternative `https://freetsa.org/tsr`.
  - Kostenlos; qualifizierte eIDAS-Zeitstempel (kostenpflichtig, z. B. D-Trust) sind
    nicht nötig.
  - Andockpunkt Interface: `HashChainService`, TODO bei `internal/domain/integrity.go:18`.

## 6. Auswirkungen auf Konfiguration & UI

- `TenantConfig`: `CertPath`/`KeyPath` entfallen; stattdessen Keyfile bei den Daten.
- Frage "Key-Pfad in Settings nicht änderbar" löst sich auf: statt Pfad auf eine
  Key-Datei nun **Passphrase eingeben/ändern**; Keyfile lebt beim Datenspeicher.
- `frontend/src/pages/SettingsPage.tsx`: Read-only-Zertifikatsanzeige → "Passphrase
  ändern" + "Datenspeicher wählen".
- Setup-Wizard `frontend/src/components/SetupAssistantScreen.tsx`: Schritt
  "Sicherheitsschlüssel (Signatur)" → "Passwort / Verschlüsselung des Datenspeichers".

## 7. Migration (Entwicklungsphase)

- App ist noch in Entwicklung → **kein Nachverschlüsseln/Nachstempeln von Altdaten**
  nötig. Ab Aktivierung wird vorwärts verschlüsselt/gestempelt.
- Neue Spalten für Zeitstempel-Token nullable; Bestandsbuchungen ohne Token gelten
  als "vor Aktivierung" und werden gemeldet, nicht als Fehler gewertet.

## 8. Phasenplan

- **Phase 0 – Aufräumen (gering):** `cert.go`/Ed25519/Cert-Erzeugung + `CertPath`
  entfernen, UI-Labels korrigieren.
- **Phase 1 – Ein-Key at-rest (reines Go):** Argon2id + Envelope (DEK),
  Feld-Verschlüsselung per GORM-Serializer (AES-256-GCM), Beleg-Verschlüsselung,
  Passphrase-Unlock beim Start, "Passphrase ändern".
- **Phase 2 – RFC-3161-Zeitstempel (mittel):** DigiCert als Default-TSA
  (konfigurierbar, FreeTSA als Alternative), ausgelöst bei Abschluss/Export,
  Token + Cert bei den Daten, Offline-Verifikation.

## 9. Getroffene Entscheidungen

1. **DB-Krypto:** Feld-Verschlüsselung in reinem Go (kein CGo/SQLCipher), Teilschutz
   akzeptiert (Abschnitt 4).
2. **Default-TSA:** DigiCert (`http://timestamp.digicert.com`), FreeTSA als
   konfigurierbare Alternative (Abschnitt 5).
