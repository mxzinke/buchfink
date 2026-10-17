# MCP-Server

[Dokumentation](../README.md) · [Entwicklung](README.md)

Diese Seite ist die Quelle für die Skills der verbundenen Apps. Ändert sich ein
Werkzeug, ein Fehlercode oder eine Regel, ändert sich diese Seite im selben
Commit; `TestDocsListEveryTool` hält Tabelle und Katalog gleich.

## Aufbau

Der MCP-Server läuft im Prozess der App auf `127.0.0.1:7465` (Streamable HTTP,
eigener OAuth-Autorisierungsserver, Zustimmung im Fenster der App). Claude
Desktop verbindet sich über die stdio-Brücke `cmd/buchfink-mcp`. Jedes Werkzeug
geht durch denselben Rahmen in `internal/mcpserver/tools.go`: Anmeldung lesen,
geöffneten Mandanten mit Lease holen, Zulassung und Rechte für diesen Mandanten
prüfen, den Erfassungsweg `agent:<kurzname>` setzen und über `Tenant.Read` oder
`Tenant.Write` in die Dienste gehen. Werkzeuge rufen nur Dienste auf, nie
Repositories (`TestToolsDoNotTouchRepositories`). Die Regeln für Routine stehen
in `internal/core/policy`, nicht im Werkzeug und nicht im Prompt.

## Freigabestufen

| Stufe | Annotation | Bedeutung |
|---|---|---|
| lesen | `readOnlyHint: true` | ändert nichts; jede zugelassene App mit `buchfink:lesen` |
| direkt | `destructiveHint: false` | schreibt ohne Rückfrage, aber nur, was `internal/core/policy` als Routine einstuft; sonst `freigabe_noetig` |
| rueckfrage | `destructiveHint: true` | der Agent legt dem Menschen den Vorgang im Chat vor und holt sein Ja ein, bevor er aufruft |
| nur App | kein Werkzeug | Festschreibung, Übermittlung und Transferticket, Pflicht als erledigt, Aufstellen, Feststellen, Offenlegen, Ergebnisverwendung, Stammdaten samt Bankkonten und steuerlicher Beratung, Darlehensverträge, Standardgruppe bestätigen, Prüfermodus, Apps zulassen und entziehen, Regeln für Apps |

Ein `_meta`-Feld `anthropic/requiresUserInteraction` setzt der Server nicht:
der Agent fragt laut Skill selbst, eine zweite Abfrage des Clients wäre doppelt.
Die Dienste weisen Vorgänge der App für jeden Erfassungsweg außer `ui` ab
(`requireHuman`, `core.AppOnly`); ein Werkzeug erhält dann `nur_in_app`.

Schreibende Werkzeuge verlangen `mandant_id` aus `status`. Wer in einem
Vorgang schreibt, nimmt einen `idempotenz_schluessel` (8 bis 100 Zeichen):
derselbe Schlüssel mit derselben Eingabe liefert das erste Ergebnis mit
`wiederholt: true`, mit anderer Eingabe `schluessel_wiederverwendet`. Die
Werkzeuge aus mehreren Schritten (`kontoauszug_importieren`, die Exporte)
nehmen keinen Schlüssel; sie schreiben von sich aus nichts doppelt.

Beträge sind Zeichenketten in Euro mit Punkt (`"119.00"`), Datumsangaben
`JJJJ-MM-TT`. Listen kommen seitenweise (`limit` bis 100, `cursor`,
`naechste_seite`, `gesamt`).

## Werkzeuge

| Werkzeug | Stufe | Zweck |
|---|---|---|
| `status` | lesen | Mandant, Geschäftsjahre, Prüfermodus, Belegordner, Exportordner, Direktregeln |
| `aufgaben` | lesen | Arbeitsliste mit dem Werkzeug für den nächsten Schritt |
| `fristen` | lesen | Pflichtenkalender mit Status, Kanal, Zuständigkeit und Verweis auf `pflichten.md#<anker>` |
| `konten` | lesen | Kontensuche im SKR04, Buchungsgruppen, Steuerfälle |
| `kontakte_suchen` | lesen | Kunden und Lieferanten mit Saldo und Standardgruppe |
| `belege_suchen` | lesen | Belege nach Status, Richtung, Datum, Text |
| `beleg_lesen` | lesen | Kopfdaten, Dateien, Befunde, Vorschlag aus der E-Rechnung, Einstufung |
| `buchungen_suchen` | lesen | Journal nach Zeitraum, Konto, Kontakt, Beleg, Betrag, Erfassungsweg |
| `buchung_lesen` | lesen | eine Buchung mit allen Zeilen und Storno-Bezügen |
| `offene_posten` | lesen | Forderungen und Verbindlichkeiten, Altersstruktur, Verzugszinsen, Mahnstand |
| `bankumsaetze` | lesen | Bankumsätze mit Zuordnungsvorschlägen und Einstufung |
| `auswertung` | lesen | Summen und Salden, Kontoblatt, Bilanz, GuV, USt-Übersicht, Prüfbericht |
| `anlagen` | lesen | Anlagenverzeichnis, fällige AfA, Aktivierungskandidaten |
| `abschluss_status` | lesen | Schritte des Jahresabschlusses mit nächster Aktion |
| `erklaerung` | lesen | UStVA, ZM, USt-, KSt- und GewSt-Erklärung formularnah mit Herkunft |
| `rechnung_vorschau` | lesen | Ausgangsrechnung durchrechnen ohne Ausstellen |
| `beleg_ablegen` | direkt | Dateien aus einem Belegordner ablegen, E-Rechnung auslesen und prüfen |
| `kontakt_anlegen` | direkt | Kontakt anlegen nach Dublettenprüfung |
| `kontoauszug_importieren` | direkt | CAMT.053 ablegen und Umsätze einlesen |
| `beleg_buchen` | direkt | Eingangsrechnung buchen, nur Routine |
| `bankumsatz_zuordnen` | direkt | Bankumsatz mit genau einem offenen Posten ausgleichen, nur Routine |
| `erklaerung_exportieren` | direkt | Jahreserklärung als JSON in den Exportordner |
| `ebilanz_exportieren` | direkt | E-Bilanz prüfen und als XBRL in den Exportordner |
| `pflicht_vorbereitet` | direkt | Pflicht als vorbereitet melden, nie als erledigt |
| `standardgruppe_vorschlagen` | direkt | Standard-Buchungsgruppe eines Lieferanten vorschlagen |
| `beleg_buchen_mit_rueckfrage` | rueckfrage | Eingangsbeleg in allen Nicht-Routinefällen, auch Auslage |
| `bankumsatz_buchen` | rueckfrage | Aufteilung mit Skonto, Gebühren, Darlehen; Kreditkartenabrechnung; Ignorieren mit Grund |
| `buchung_erfassen` | rueckfrage | Handbuchung mit Beleg oder Eigenbeleg, auch in der Abschlussperiode; ein Kontakt an den Personenkontozeilen macht sie zum offenen Posten |
| `buchung_stornieren` | rueckfrage | Generalumkehr mit Grund, auf Wunsch mit Neubuchung |
| `lohnjournal_buchen` | rueckfrage | Monatssummen des Lohnbüros |
| `rechnung_ausstellen` | rueckfrage | Rechnung, Abschlagsrechnung, Schlussrechnung |
| `mahnung_vermerken` | rueckfrage | Zugang einer Mahnung festhalten; daraus folgt der Verzug |
| `abschlussbuchung` | rueckfrage | AfA-Lauf, Abgrenzung, Rückstellung, USt-Verrechnung, Steuerrückstellung, UG-Rücklage, Wertberichtigung, unfertige Leistungen, Fremdwährung, Anlage aktivieren |
| `erklaerung_eingabe` | rueckfrage | Angabe zu einer Zeile einer Jahreserklärung; was der Mensch gesetzt oder bestätigt hat, ändert nur er |
| `beleg_verwerfen` | rueckfrage | abgelegten Beleg mit Grund verwerfen |
| `anlass_erfassen` | rueckfrage | Gesellschafterwechsel oder Vorauszahlungsbescheid erfassen |

Eine Auslage eines Gesellschafters bucht `beleg_buchen_mit_rueckfrage` mit
`bezahlung: auslage`; ein eigenes Werkzeug gibt es dafür nicht.

## Fehlercodes

Ein Fehler kommt als `isError` mit `{"fehler": {code, nachricht,
naechster_schritt, werkzeug, gruende, wiederholbar}}`. Schemaverstöße meldet
das SDK selbst.

| Code | Bedeutung |
|---|---|
| `kein_mandant_offen` | in der App ist kein Mandant geöffnet |
| `mandant_nicht_freigegeben` | die App ist für den geöffneten Mandanten nicht zugelassen |
| `mandant_gewechselt` | `mandant_id` ist nicht der geöffnete Mandant; nichts geschrieben |
| `berechtigung_fehlt` | Scope `buchfink:lesen` oder `buchfink:buchen` fehlt |
| `pruefermodus_aktiv` | Prüfermodus, nur lesen; Exporte laufen weiter |
| `nur_in_app` | der Vorgang gehört dem Menschen in der App |
| `freigabe_noetig` | kein Routinefall; `gruende` nennt die Regeln, `werkzeug` das Werkzeug mit Rückfrage |
| `pfad_nicht_erlaubt` | Pfad liegt außerhalb der Belegordner oder im Datenordner |
| `datei_ungueltig` | Typ, Größe oder Inhalt passt nicht |
| `schluessel_wiederverwendet` | Idempotenzschlüssel mit anderer Eingabe |
| `beschaeftigt` | Schreibsperre länger als 30 Sekunden belegt; wiederholen |
| `ungueltige_eingabe` | Feld fehlt oder ist falsch, auch ein zu kurzer Idempotenzschlüssel |
| `nicht_gefunden` | Kennung unbekannt |
| `fachliche_regel` | der Dienst lehnt ab; die Nachricht nennt den Grund |
| `intern` | Programmfehler |

## Regeln für direkt

`beleg_buchen` bucht nur, wenn alles zutrifft (`policy.ClassifyReceipt`).
Sonst nennt `gruende` jeden verletzten Schlüssel:

| Schlüssel | Regel |
|---|---|
| `beleg_nicht_offen`, `keine_eingangsrechnung` | abgelegte Eingangsrechnung |
| `keine_e_rechnung`, `e_rechnung_ungeprueft`, `e_rechnung_mit_befund`, `kein_vorschlag` | E-Rechnung mit strukturiertem Teil, geprüft, ohne Befund; Fotos und PDF ohne Datensatz nie |
| `kontakt_nicht_erkannt`, `kontakt_gesperrt` | der Aussteller der E-Rechnung ist dieser Kontakt, nicht gesperrt |
| `keine_bestaetigte_standardgruppe`, `gruppe_weicht_ab` | der Mensch hat die Standardgruppe in der App bestätigt; alle Positionen nutzen sie |
| `zu_wenige_buchungen` | mindestens so viele frühere, nicht stornierte Belegbuchungen des Lieferanten auf den Konten der Gruppe wie eingestellt (Vorgabe 3) |
| `betraege_weichen_ab` | Netto je Satz und Brutto centgenau wie in der E-Rechnung |
| `steuerfall_nicht_inland`, `fremdwaehrung` | Inland, Euro |
| `ueber_direktgrenze` | Brutto bis zur Direktgrenze (Vorgabe 1.000 €) |
| `sonderangaben` | keine Bewirtung, kein Geschenk, kein Vorsteueranteil, keine Rückstellung, keine Anzahlung, keine Gruppe mit Sonderregel, kein Konto statt Gruppe, keine Übersteuerung |
| `bezahlart` | offen oder Firmenkreditkarte |
| `datum_in_zukunft` | Beleg- und Buchungsdatum nicht in der Zukunft |
| `dublettenverdacht` | kein anderer Beleg desselben Ausstellers mit gleicher Rechnungsnummer oder gleichem Betrag am selben Tag |

`bankumsatz_zuordnen` bucht nur, wenn genau ein offener Posten in der Richtung
des Umsatzes centgenau den Betrag hat und seine Beleg-, Buchungs- oder
Rechnungsnummer im Verwendungszweck steht (`policy.ClassifyBankMatch`;
Gründe `umsatz_nicht_offen`, `fremdwaehrung`, `kein_exakter_treffer`,
`mehrere_treffer`, `anderer_posten`).

Die Standardgruppe schlägt ein Agent nur vor (`standardgruppe_vorschlagen`,
`kontakt_anlegen`); bestätigen kann sie nur der Mensch
(`ContactService.ConfirmDefaultPostingGroup`, Bridge
`ConfirmDefaultPostingGroup`). `SaveContact` ändert die Felder nicht. Die
Schwellen und Ordner stellt der Mensch in der App ein
(`SettingsService.SaveAgentSettings`, Bridge `GetAgentRules` und
`SaveAgentRules`; `receiptFolders: null` heißt Voreinstellung, eine leere Liste
sperrt alle Belegordner).

## Belegordner und Exportordner

Belege kommen nur als Pfad. `policy.ReadFile` löst Symlinks und `..` auf,
prüft danach den Ordner und liest die Datei einmal: PDF, PNG, JPEG, TIFF, WebP
oder XML bis 50 MB, Inhalt passend zur Endung, keine versteckten Dateien. Der
Datenordner des Mandanten ist gesperrt bis auf seinen Eingang, der
Sicherungsordner ganz.

Voreinstellung der Belegordner ist `<Datenordner>/Belegeingang` und der
Downloads-Ordner des Nutzers. Downloads, weil dort Anhänge und Portal-Downloads
landen; ein Agent ohne eigenen Dateizugriff wie Claude Desktop käme sonst nur
an eine Rechnung, wenn der Mensch sie vorher verschiebt. Der Preis: ein Agent
kann eine private Datei aus Downloads als Beleg ablegen. Sie bleibt lokal, und
`beleg_ablegen` ist sichtbar im Protokoll. Wer das nicht will, nimmt Downloads
in der App aus der Liste. Exporte schreiben nur in den Exportordner,
voreingestellt `<Datenordner>/Export`.

## Ein Werkzeug ergänzen

1. Dienst zuerst: die fachliche Regel und jede Sperre für Agenten
   (`requireHuman`) gehören in `internal/service`, nicht ins Werkzeug.
2. Ein- und Ausgabe als Go-Structs mit `json`- und `jsonschema`-Tags. Beträge
   als `Betrag`, Daten als `Datum`, Gruppen und Steuerfälle als
   `Buchungsgruppe`, `Steuerfall`, `Satz`; das Schema zeigt dann Muster und
   gültige Werte. Listen in der Ausgabe nie `nil`.
3. Registrieren mit `AddRead`, `AddWrite`, `AddWritePrepared` (Vorlauf ohne
   Sperre, etwa ein Kurs aus dem Netz) oder `AddFlow` (mehrere Vorgänge) und der
   Stufe in `Spec.Tier`. Die Beschreibung sagt, wann das Werkzeug passt, wann
   nicht und welches danach kommt.
4. Ein direktes Werkzeug fragt `internal/core/policy` und antwortet sonst mit
   `errApproval`.
5. Tests über den Harness (`harness_test.go`): Erfolg mit Schemaprüfung der
   Ausgabe, Rechte, falscher Mandant, Prüfermodus, Idempotenz, bei direkt die
   Regelverletzung. Die Matrix in `catalog_test.go` braucht eine gültige
   Mindesteingabe des Werkzeugs.
6. Diese Tabelle ergänzen.
