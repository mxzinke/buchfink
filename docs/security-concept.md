# Buchfink – Sicherheitskonzept (Verschlüsselung & Integrität)

Status: umgesetzt (Phasen 0 bis 3, siehe Abschnitt 8)
Letzte Aktualisierung: 2026-08-30

Dieses Dokument beschreibt Vertraulichkeit (Verschlüsselung at-rest) und
Integrität/Nachweisbarkeit (Hashkette + Zeitstempel) in Buchfink. Es war
ursprünglich das Zielkonzept und beschreibt seit der Umsetzung den gebauten
Zustand; es ersetzt den früheren, faktisch ungenutzten
Zertifikats-/Signaturansatz.

## 1. Ausgangslage vor der Umsetzung

Dieser Abschnitt hält fest, wovon das Konzept ausging. Nichts davon gilt heute
noch, die genannten Dateien gibt es nicht mehr.

- Pro Mandant wurde ein Ed25519-Keypair samt selbstsigniertem X.509-Zertifikat
  erzeugt (`internal/security/cert.go`), aber **nie wieder geladen oder genutzt**.
  Der Key-Pfad wurde beim Anlegen sogar verworfen.
- **Keine Datenverschlüsselung at-rest:** SQLite wurde im Klartext geöffnet, nur
  WAL und busy_timeout.
- **Integrität = ungekeyed SHA256-Hashkette.** Kein Geheimnis beteiligt → wer die
  DB schreiben kann, kann die Kette neu rechnen. Das gilt weiterhin für die Kette
  allein; erst die Feldverschlüsselung macht sie wirksam (Abschnitt 2).
- Das UI-Label "Sicherheitsschlüssel (Signatur)" war irreführend, es wurde nichts
  signiert.

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

| Ist entfallen | Fundstelle damals |
|---|---|
| Ed25519 Private Key (`buchfink-key.pem`) | `internal/security/cert.go:34,80-104` |
| Ed25519 Public Key (im Cert eingebettet) | `internal/security/cert.go:63` |
| X.509 Zertifikat (`buchfink-cert.pem`) | `internal/security/cert.go:50-77` |
| `CertPath` in Config | `internal/domain/app_config.go:8,18` |
| geplante Signatur-Spalte / `SigningIdentity`-Tabelle | (verworfen) |

`internal/security/cert.go` wurde entfernt und durch die Schlüsselableitung
(Abschnitt 4) ersetzt.

## 4. Vertraulichkeit: Ein-Schlüssel-Modell (Envelope)

```
OS-Schlüsselbund (Zufallsgeheimnis) ─┐
                                     ├─► KEK ─► DEK (256-bit)  ──► verschlüsselt
Recovery-Schlüssel (Export-Datei) ───┘  (2 unabhängige Slots)      sensible DB-Felder
```

- **Transparent**: ein zufälliges Wrapping-Geheimnis liegt im OS-Schlüsselbund
  (macOS Keychain / Windows Credential Manager / Linux Secret Service), Dienst
  `org.buchfink.app`, Konto = Tenant-ID. Kein Passwort zum Merken.
- Auf der Platte nur ein kleines Keyfile (`buchfink.keyfile.json`, `0600`):
  Slots mit dem gewrappten DEK – **kein** PEM/Cert.
- **Recovery-Slot**: zweiter Slot, entsperrbar mit einem extern gesicherten
  Recovery-Schlüssel (siehe Abschnitt 4a). Beide Slots wrappen denselben DEK.
- **BELEGE WERDEN NICHT VERSCHLÜSSELT.** GoBD verlangt, dass Original-Belege
  unverändert im Originalformat erhalten bleiben. Verschlüsselt wird nur der
  Ablagepfad und Dateiname in der Datenbank (`StoredPath`, `FileName` in
  `internal/domain/receipt.go`); die Beleg-Datei selbst bleibt unangetastet.
  `Receipt.SHA256` je Datei und `ReceiptHash` über die geordnete Dateiliste
  sichern sie. Die Begründung steht auch im Code
  (`internal/receiptstore/store.go:10`).
- Config `~/.buchfink/config.json`: Modus `0600`, enthält keine Geheimnisse.

### 4a. Externe Sicherung: Recovery-Schlüsseldatei (LUKS-Key-Slot-Prinzip)

Da das Wrapping-Geheimnis nur im OS-Schlüsselbund liegt, wäre bei Rechnerverlust
sonst alles verloren. Deshalb ein zweiter, exportierbarer Key-Slot:

- **Export** (`security.ExportTenantRecoveryFile`): erzeugt einen zufälligen
  256-bit Recovery-Key, wrappt den DEK zusätzlich unter ihm (Recovery-Slot im
  Keyfile) und schreibt den Recovery-Key in eine portable JSON-Datei
  (`buchfink-recovery-<tenant>.json`), die extern gelagert wird (USB/Tresor).
- **Restore** (`security.RecoverTenantFromFile`): auf neuem Rechner Datenordner
  wiederherstellen + Recovery-Datei einlesen → DEK entsperrt → **frischer
  Schlüsselbund-Eintrag** wird angelegt (danach wieder transparent).
- **Sicherheitsmodell**: Entschlüsseln erfordert **beides** – Keyfile (im
  Datenbackup) **und** Recovery-Datei. Getrennt gelagert ist keine Hälfte allein
  verwertbar. UI kommuniziert dies im Setup (Schritt 2) und in den Einstellungen.

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
| `FiscalYear`, `EntryNumber`, `BookingDate`, `DocumentDate` | `Description` (Buchungstext), Zeilentext |
| `JournalLine.Account`, `Side` | `StoredPath`, `FileName`, Verwerfungsgrund |
| `Amount`, `TaxBase` (für SQL-Summen) | Gegenpartei-IBAN und Verwendungszweck (Bank-Tx) |
| `PreviousHash`, `EntryHash`, `ReceiptHash` | Kontakt-/Stammdaten-PII, Rechnungs-Freitext, Stornogrund |

Heute tragen 31 Felder das Tag `serializer:encrypted`; die aktuelle Liste findet
sich mit `grep -rn 'serializer:encrypted' internal/domain`.

Konsequenz: Beträge bleiben lesbar (nötig für `CalculateAccountSums`/`CalculateTypeSums`).
Wer stärkeren Schutz will, müsste später auf SQLCipher (CGo) oder App-seitige
Aggregation umstellen.

## 5. Integrität & Nachweis: Hashkette + RFC-3161-Zeitstempel

- **Hashkette** unverändert (`internal/accounting/journalhash.go`):
  Erzeugung in `JournalService.Post` (`internal/service/journal_service.go:83`),
  Prüfung in `JournalService.VerifyIntegrity` (`journal_service.go:218`).
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

## 6. Konfiguration & UI

- `TenantConfig` führt weder `CertPath` noch `KeyPath`. Das Keyfile liegt im
  Datenordner des Mandanten, das Wrapping-Geheimnis im Schlüsselbund
  (`internal/domain/app_config.go`).
- Eine Passphrase gibt es nicht: das Entsperren ist transparent über den
  Schlüsselbund. Zu merken ist nichts, zu sichern ist die Recovery-Datei.
- `frontend/src/pages/SettingsPage.tsx:398` zeigt unter "Speicherort und
  Schlüssel" den Datenordner, den Schlüsselbund-Eintrag samt Fundort je
  Betriebssystem und den Export des Recovery-Schlüssels.
- `frontend/src/components/RecoveryScreen.tsx` liest eine Recovery-Datei wieder
  ein, wenn der Schlüsselbund den Eintrag nicht mehr hat.

## 7. Migration (Entwicklungsphase)

- App ist noch in Entwicklung → **kein Nachverschlüsseln/Nachstempeln von Altdaten**
  nötig. Ab Aktivierung wird vorwärts verschlüsselt/gestempelt.
- Neue Spalten für Zeitstempel-Token nullable; Bestandsbuchungen ohne Token gelten
  als "vor Aktivierung" und werden gemeldet, nicht als Fehler gewertet.

## 8. Phasenplan & Umsetzungsstand

- **Phase 0 – Aufräumen ✅ ERLEDIGT:** `cert.go`/Ed25519/Cert-Erzeugung + `CertPath`
  entfernt, Bindings regeneriert, UI-Labels korrigiert.
- **Phase 1 – Ein-Key at-rest ✅ ERLEDIGT:** Argon2id + Envelope (DEK,
  `internal/security/crypto.go`), Feld-Verschlüsselung per GORM-Serializer
  (`internal/repository/encryption.go`), transparenter OS-Schlüsselbund-Unlock
  (`keyring.go`). Belege NICHT verschlüsselt (GoBD). Recovery-Schlüsseldatei
  (`recovery.go`) + Recovery-Screen (`RecoveryScreen.tsx`).
  - Offen (Follow-up): "Schlüssel/Passphrase ändern"-UI (Backend
    `ChangeTenantPassphrase` existiert bereits).
- **Phase 2 – RFC-3161-Zeitstempel ✅ ERLEDIGT:** `internal/timestamp/tsa.go`
  (DigiCert Default, FreeTSA-fähig), Offline-Verifikation. Nur der SHA-256-Hash
  wird gesendet, Token self-contained.
- **Phase 3 – Festschreibung ✅ ERLEDIGT (fachlicher Anker):** Statt eines manuellen
  „Zeitstempel"-Buttons ist der Zeitstempel jetzt ein **stiller Nebeneffekt der
  GoBD-Festschreibung**:
  - Modell `domain.Festschreibung` (Periode, Stichtag, Chain-Head, eingebetteter
    Zeitstempel), Repo, Migration.
  - **Enforcement** (`JournalService.ensurePeriodOpen`,
    `internal/service/journal_service.go:372`): keine rückdatierten Neubuchungen
    in einen festgeschriebenen Zeitraum (`BookingDate <= LatestCutoff`); Storno
    bleibt erlaubt (auf heute datiert). Ein automatischer Test dafür fehlt
    bisher.
  - **Stiller Zeitstempel**: bei `CommitPeriod` best-effort über DigiCert; offline →
    Festschreibung steht trotzdem, Status `pending`, Nachholen beim Start
    (`retryPendingTimestamps`).
  - **UI**: Festschreiben unter „Fristen & Termine" (an den USt-Zeitraum gekoppelt,
    monatlich/quartalsweise), Übersicht + Offline-Prüfung auf der Audit-Seite.
  - Buchungen sind ohnehin append-only (kein Update/Delete), Festschreibung ergänzt
    die Sperre gegen rückdatierte Neuanlage.
  - Offen (Optional): konfigurierbare TSA-URL in den Einstellungen.

## 9. Getroffene Entscheidungen

1. **DB-Krypto:** Feld-Verschlüsselung in reinem Go (kein CGo/SQLCipher), Teilschutz
   akzeptiert (Abschnitt 4).
2. **Default-TSA:** DigiCert (`http://timestamp.digicert.com`), FreeTSA als
   konfigurierbare Alternative (Abschnitt 5).
