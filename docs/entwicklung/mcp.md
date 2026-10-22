# MCP-Server

[Dokumentation](../README.md) · [Entwicklung](README.md)

Diese Seite ist die Quelle für die Skills der verbundenen Apps. Ändert sich ein
Werkzeug, ein Fehlercode oder eine Regel, ändert sich diese Seite im selben
Commit. `TestDocsListEveryTool` hält Werkzeugtabelle und App-Vorbehalte mit
`internal/ops` gleich.

## Aufbau

Der MCP-Server läuft im Prozess der App auf `127.0.0.1:7465` (Streamable HTTP,
eigener OAuth-Autorisierungsserver, Zustimmung im Fenster der App). Claude
Desktop verbindet sich über die stdio-Brücke `cmd/buchfink-mcp`. Jedes Werkzeug
geht durch denselben Rahmen in `internal/mcpserver/tools.go`: Anmeldung lesen,
geöffneten Mandanten mit Lease holen, Zulassung und Rechte für diesen Mandanten
prüfen, den Erfassungsweg `agent:<kurzname>` setzen und über `Tenant.ReadAgent`
oder `Tenant.WriteAgent` in die Dienste gehen. Diese Einstiege verwenden
`Read` und `Write` mit `core.AgentServices`, der auf die benötigten Methoden
begrenzten Dienstsicht. Dieselbe Sicht verwendet `core/policy`.
`TestAgentServicesMatchAdapterUse` hält den Adapter frei von vollständigen
Diensttypen und prüft, dass jede Fassadenmethode genutzt wird.
Werkzeuge rufen nur Dienste auf, nie
Repositories (`TestToolsDoNotTouchRepositories`). Die Regeln für Routine stehen
in `internal/core/policy`, die Vorbehalte der App im Kern und in den Diensten,
nicht im Werkzeug und nicht im Prompt.

## Freigabestufen

[`internal/ops`](../../internal/ops) ist die deklarative Quelle der Stufen.
Bei der Registrierung übernimmt der MCP-Rahmen `Spec.Tier` aus dem Katalog
und bildet daraus die Annotationen. Unbekannte und `AppOnly`-Vorgänge lassen
sich nicht als Werkzeug registrieren. Golden-Tests sichern Werkzeugliste,
Schemas, Annotationen und Stufen.

| Stufe | Annotation | Bedeutung |
|---|---|---|
| lesen | `readOnlyHint: true` | ändert nichts; jede zugelassene App mit `buchfink:lesen` |
| direkt | `destructiveHint: false` | schreibt ohne Rückfrage, aber nur, was `internal/core/policy` als Routine einstuft; sonst `freigabe_noetig` |
| rueckfrage | `destructiveHint: true` | der Agent legt dem Menschen den Vorgang im Chat vor und holt sein Ja ein, bevor er aufruft |
| nur App | kein Werkzeug | die [vorbehaltenen Vorgänge](#vorgänge-nur-in-der-app) mit ihrer Durchsetzungsart |

Ein `_meta`-Feld `anthropic/requiresUserInteraction` setzt der Server nicht.
Der Agent fragt laut Skill selbst; eine zweite Abfrage des Clients wäre doppelt.
Kern und Dienste behalten ihre Schutzprüfungen (`core.AppOnly`,
`requireApp`, `service.requireHuman`). Die kanalgeschützten Vorgänge weisen
jeden Erfassungsweg außer `ui` ab; ein Aufruf über ein Werkzeug meldet dann
`nur_in_app`. Tests führen jede Schutzstelle aus dem Katalog über zwei
Agentenkanäle, `system` sowie fehlende und ungültige Erfassungswege aus.
Dienste werden dabei ohne die zusätzliche `core.AppOnly`-Option aufgerufen.

Zulassen und Entziehen verbundener Apps sind strukturell geschützt: nur die
Bridge ruft die privaten Verwaltungs-APIs des Servers auf. Diese APIs prüfen
keinen Erfassungsweg; die Bridge übergibt `context.Background()`. Ein Test
begrenzt alle Methodenverweise bis zum OAuth-Speicher auf diese Aufrufkette.
Weder ein MCP-Werkzeug noch ein HTTP-Handler darf sie erreichen. Der
OAuth-Endpunkt zum Widerrufen eines Tokens entzieht keine App-Zulassung.

Was sich per Storno korrigieren lässt, darf ein Agent mit Rückfrage, auf jedem
Konto des Kontenrahmens:

| Vorgang | Werkzeug |
|---|---|
| Kapital, Rücklagen, Ausschüttung, Gesellschafterkonten, Gründungsbuchungen (Zeichnung, Einforderung) | `buchung_erfassen`, Zahlungen über `bankumsatz_buchen` |
| Darlehensvertrag mit einem Gesellschafter anlegen oder berichtigen, seine Zinsen buchen | `darlehen_erfassen` |
| Ergebnisverwendung nach der Feststellung durch den Menschen | `ergebnisverwendung_buchen` |
| Organisationstexte der Verfahrensdokumentation | `verfahrensdoku_text` |

Die Gründung selbst mit Gesellschaftern, Anteilen und Eintragung bleibt
Stammdatum des Menschen (`FoundationService.Save`, `Register`). Ein Agent
schlägt sie mit `stammdaten_vorschlagen` vor; geschrieben wird erst, wenn der
Mensch den Vorschlag in der App übernimmt.

Eine Zeile auf einem Konto, das Beträge je Gesellschafter führt (0960, 1307
bis 1309, 3640 bis 3643 und die übrigen aus `shareholderAccounts`), braucht den
Gesellschafter als Kontakt, in der Handbuchung, in der Neubuchung beim Storno
und in der Aufteilung eines Bankumsatzes. Ohne ihn fehlte sie im Stand des
Darlehens und in der Zinsrechnung.

### Vorgänge nur in der App

Der Katalog unterscheidet den vorbehaltenen Vorgang von seiner ganzen
Einstiegsmethode: `SetInput` darf zum Beispiel Vorschläge annehmen, aber keine
bestätigte Angabe des Menschen überschreiben. Beim Anlegen eines Kontakts
entfernt `keepPostingGroup` eine unzulässige Bestätigung; der Kontakt selbst
wird angelegt.

| Vorgang | Stufe | Bedeutung | Durchsetzung |
|---|---|---|---|
| `abschlusseinstellungen_aendern` | nur App | Abschluss-Einstellungen ändern | Erfassungsweg ui im Kern oder Dienst |
| `abschlussstatus_setzen` | nur App | Jahresabschluss aufstellen, feststellen und offenlegen | Erfassungsweg ui im Kern oder Dienst |
| `agentregeln_aendern` | nur App | Regeln für direktes Buchen sowie Export- und Leseordner ändern | Erfassungsweg ui im Kern oder Dienst |
| `apps_entziehen` | nur App | Apps den Zugriff entziehen | strukturell: private API der Bridge |
| `apps_zulassen` | nur App | Apps zulassen | strukturell: private API der Bridge |
| `bankkonten_einrichten` | nur App | Bankkonten einrichten | Erfassungsweg ui im Kern oder Dienst |
| `erster_kontoauszug` | nur App | Ersten Kontoauszug eines Kontos einlesen | Erfassungsweg ui im Kern oder Dienst |
| `datenbank_anmelden` | nur App | Bestehende Datenbank als Mandant anmelden | Erfassungsweg ui im Kern oder Dienst |
| `eingang_waehlen` | nur App | Eingang wählen | Erfassungsweg ui im Kern oder Dienst |
| `eintragung_erfassen` | nur App | Eintragung ins Handelsregister erfassen | Erfassungsweg ui im Kern oder Dienst |
| `erklaerung_angabe_bestaetigen` | nur App | Vorbehaltene Angabe einer Erklärung bestätigen | Erfassungsweg ui im Kern oder Dienst |
| `erklaerung_bestaetigte_angabe_aendern` | nur App | Bestätigte Angabe des Menschen ändern | Erfassungsweg ui im Kern oder Dienst |
| `erklaerung_bestaetigte_angabe_loeschen` | nur App | Bestätigte vorbehaltene Angabe zurücknehmen | Erfassungsweg ui im Kern oder Dienst |
| `erklaerung_uebermittlung_bestaetigen` | nur App | Übermittlung einer Jahreserklärung mit Transferticket bestätigen | Erfassungsweg ui im Kern oder Dienst |
| `festschreiben` | nur App | Festschreibung | Erfassungsweg ui im Kern oder Dienst |
| `feststellung_zuruecknehmen` | nur App | Feststellung zurücknehmen | Erfassungsweg ui im Kern oder Dienst |
| `gruendung_erfassen` | nur App | Gründung mit Gesellschaftern erfassen | Erfassungsweg ui im Kern oder Dienst |
| `mandant_anlegen` | nur App | Mandant mit Ablage anlegen | Erfassungsweg ui im Kern oder Dienst |
| `mandant_entfernen` | nur App | Mandant aus der Liste entfernen | Erfassungsweg ui im Kern oder Dienst |
| `mandant_wechseln` | nur App | Mandant wechseln | Erfassungsweg ui im Kern oder Dienst |
| `mcp_port_aendern` | nur App | Port des MCP-Servers ändern | Erfassungsweg ui im Kern oder Dienst |
| `pflicht_bestaetigen` | nur App | Pflicht als erledigt bestätigen | Erfassungsweg ui im Kern oder Dienst |
| `pflicht_entfallen` | nur App | Pflicht als entfallen kennzeichnen | Erfassungsweg ui im Kern oder Dienst |
| `pflicht_zuruecknehmen` | nur App | Bestätigung einer Pflicht zurücknehmen | Erfassungsweg ui im Kern oder Dienst |
| `pruefermodus_aendern` | nur App | Prüfermodus ein- und ausschalten | Erfassungsweg ui im Kern oder Dienst |
| `rechnungskonto_waehlen` | nur App | Bankkonto für Rechnungen wählen | Erfassungsweg ui im Kern oder Dienst |
| `sicherung_wiederherstellen` | nur App | Sicherung wiederherstellen | Erfassungsweg ui im Kern oder Dienst |
| `sicherungsordner_waehlen` | nur App | Sicherungsordner wählen | Erfassungsweg ui im Kern oder Dienst |
| `stammdatenvorschlag_entscheiden` | nur App | Vorschlag eines Agenten für die Stammdaten übernehmen oder ablehnen | Erfassungsweg ui im Kern oder Dienst |
| `standardgruppe_bestaetigen` | nur App | Standardgruppe bestätigen, auch beim Anlegen eines Kontakts | Erfassungsweg ui im Kern oder Dienst |
| `unternehmensdaten_aendern` | nur App | Unternehmensstammdaten einschließlich Gesellschaftern und steuerlicher Beratung ändern | Erfassungsweg ui im Kern oder Dienst |
| `ustva_uebermittlung_bestaetigen` | nur App | Übermittlung der UStVA mit Transferticket bestätigen | Erfassungsweg ui im Kern oder Dienst |
| `zm_uebermittlung_bestaetigen` | nur App | Übermittlung der ZM mit Transferticket bestätigen | Erfassungsweg ui im Kern oder Dienst |

## Eingaben und Ausgaben

Schreibende Werkzeuge verlangen `mandant_id` aus `status`. Wer in einem
Vorgang schreibt, nimmt einen `idempotenz_schluessel` (8 bis 100 Zeichen).
Derselbe Schlüssel mit derselben Eingabe liefert das erste Ergebnis mit
`wiederholt: true`, mit anderer Eingabe `schluessel_wiederverwendet`. Pflicht
ist er bei den Werkzeugen ohne natürlichen Schlüssel: `buchung_erfassen`,
`rechnung_ausstellen`, `abschlussbuchung`, `mahnung_vermerken`,
`anlass_erfassen`, `darlehen_erfassen` und `stammdaten_vorschlagen`. Ein Aufruf mit `nur_vorschau`
verbraucht den Schlüssel nicht; derselbe Schlüssel bucht danach. Die Werkzeuge
aus mehreren Schritten (`kontoauszug_importieren`, die drei Exporte) nehmen
keinen Schlüssel, sie schreiben von sich aus nichts doppelt.

Beträge sind Zeichenketten in Euro mit Punkt (`"119.00"`), Datumsangaben
`JJJJ-MM-TT`. Jedes Auswahlfeld hat eine Werteliste im Schema; ein falscher
Wert scheitert an der Schemaprüfung des SDK. Die Werte sind deutsch, bis auf
die Steuerfälle: sie tragen die Schlüssel des Kerns (`domestic`,
`reverse_charge_eu` …), weil Belege, Buchungen und Prüfberichte sie so führen.
`konten` mit `art=steuerfaelle` erklärt sie.

Lange Listen kommen seitenweise (`limit` bis 100, `cursor`, `naechste_seite`,
`gesamt`). Ganz kommen `aufgaben`, `abschluss_status`, `erklaerung` und die
Aktivierungskandidaten in `anlagen`. `bankumsaetze` nennt je Umsatz die besten
drei Vorschläge und `vorschlaege_gesamt`; alle gibt es mit `bankumsatz_id`.

Der Katalog veröffentlicht keine Ausgabeschemas. Er liegt in jedem Gespräch im
Kontext des Agenten, und die Ausgaben machten mehr als die Hälfte davon aus.
Die Antworten sind trotzdem festgelegt: die Tests prüfen jede gegen das Schema
aus `Toolbox.OutputSchema`, `TestCatalogStaysSmall` hält den Katalog unter
84 KB.

## Werkzeuge

| Werkzeug | Stufe | Zweck |
|---|---|---|
| `status` | lesen | Mandant mit USt-Zeitraum, Dauerfrist, festgeschrieben bis, letzter Sicherung; Geschäftsjahre, Prüfermodus, Eingang mit Zahl der Dateien, Leseordner, Exportordner, Direktregeln, offene und in den letzten 14 Tagen entschiedene Vorschläge für die Stammdaten |
| `aufgaben` | lesen | Arbeitsliste mit dem Werkzeug für den nächsten Schritt, darunter „N Dateien im Eingang“ |
| `eingang` | lesen | Dateien im Eingang ohne `Übernommen/`, die älteste zuerst, mit Art und `bereits_abgelegt`; seitenweise |
| `fristen` | lesen | Pflichtenkalender mit Status, Kanal, Zuständigkeit, `anker` und Verweis auf `buchfink/references/grundlagen/pflichten.md#<anker>` |
| `konten` | lesen | Kontensuche im SKR04, Buchungsgruppen, Steuerfälle |
| `kontakte_suchen` | lesen | Kunden und Lieferanten mit Saldo und Standardgruppe |
| `belege_suchen` | lesen | Belege nach Status, Art, Richtung, Datum, Text |
| `beleg_lesen` | lesen | Kopfdaten, Dateien, Befunde, Vorschlag aus der E-Rechnung samt bestätigter Standardgruppe, Einstufung |
| `buchungen_suchen` | lesen | Journal nach Zeitraum, Konto, Kontakt, Beleg, Betrag, Erfassungsweg |
| `buchung_lesen` | lesen | eine Buchung mit allen Zeilen und Storno-Bezügen |
| `offene_posten` | lesen | Forderungen und Verbindlichkeiten, Altersstruktur, Verzugszinsen, Mahnstand |
| `bankumsaetze` | lesen | Bankumsätze mit den besten Zuordnungsvorschlägen, Einstufung und Prüfstand des Auszugs |
| `auswertung` | lesen | Summen und Salden, Kontoblatt eines Sachkontos, Bilanz, GuV, USt-Übersicht, Prüfbericht, Gesellschafterdarlehen mit Stand und Befunden, Unterbilanz der Gründung (`FoundationService.GetState`) |
| `anlagen` | lesen | Anlagenverzeichnis, fällige AfA, Aktivierungskandidaten |
| `abschluss_status` | lesen | Schritte des Jahresabschlusses mit nächster Aktion |
| `erklaerung` | lesen | UStVA, ZM, USt-, KSt- und GewSt-Erklärung formularnah mit Herkunft |
| `rechnung_vorschau` | lesen | Ausgangsrechnung durchrechnen ohne Ausstellen |
| `beleg_ablegen` | direkt | Dateien aus Eingang oder Leseordner ablegen, E-Rechnung auslesen und prüfen, fehlende Dateien an einem offenen Beleg ergänzen; `art: brief` legt ein Schreiben ohne Buchung ab, `art: sonstiges` einen Vertrag oder Beschluss für `buchung_erfassen`; danach geht die Datei aus dem Eingang nach `Übernommen/` |
| `beleg_datei_exportieren` | direkt | Kopie einer Belegdatei in den Exportordner, zum Lesen oder Versenden |
| `kontakt_anlegen` | direkt | Kontakt anlegen nach Dublettenprüfung; mit `leitweg_id` bekommt ein Kunde XRechnung |
| `kontoauszug_importieren` | direkt | CAMT.053 ablegen, Salden prüfen, Umsätze einlesen; danach geht die Datei aus dem Eingang nach `Übernommen/` |
| `beleg_buchen` | direkt | Eingangsrechnung buchen, nur Routine |
| `bankumsatz_zuordnen` | direkt | Bankumsatz mit genau einem offenen Posten ausgleichen, nur Routine |
| `erklaerung_exportieren` | direkt | Jahreserklärung als JSON in den Exportordner |
| `ebilanz_exportieren` | direkt | E-Bilanz prüfen und in den Exportordner: `format` `xbrl` (Vorgabe) oder `susa_csv`; bei abweichender Steuerbilanz nennt `ueberleitung`, was die Liste nicht enthält; `bilanzart: eroeffnungsbilanz` schreibt die Eröffnungsbilanz, `jahr` ist dann das Geschäftsjahr der Gründung |
| `pflicht_vorbereitet` | direkt | Pflicht als vorbereitet melden, nie als erledigt |
| `stammdaten_vorschlagen` | direkt | Unternehmensdaten, Gründung oder Bankkonten vorschlagen; ändert nichts, der Mensch übernimmt nach der Vorschau in der App |
| `standardgruppe_vorschlagen` | direkt | Standard-Buchungsgruppe eines Lieferanten vorschlagen |
| `beleg_buchen_mit_rueckfrage` | rueckfrage | Eingangsbeleg in allen Nicht-Routinefällen, auch Auslage; ergänzt fehlende Kopfdaten und berichtigt `eingang_am` |
| `bankumsatz_buchen` | rueckfrage | Aufteilung mit Skonto, Gebühren, Darlehen; Kreditkartenabrechnung; Kapitalertrag einer Finanzanlage, brutto mit Kapitalertragsteuer (7630) und Solidaritätszuschlag (7633), mit dem Anlagegut verknüpft (`AssetService.BookAssetIncome`); Ignorieren mit Grund |
| `buchung_erfassen` | rueckfrage | Handbuchung mit Beleg oder Eigenbeleg, auch in der Abschlussperiode und für die Gründung (Zeichnung, Einforderung); ein Kontakt an den Personenkontozeilen macht sie zum offenen Posten, an Gesellschafterkonten ist er Pflicht; Darlehenszinsen nicht hier, sondern mit `darlehen_erfassen` |
| `buchung_stornieren` | rueckfrage | Generalumkehr mit Grund, auf Wunsch mit Neubuchung; `datum` nur beim Saldenvortrag |
| `lohnjournal_buchen` | rueckfrage | Monatssummen des Lohnbüros |
| `rechnung_ausstellen` | rueckfrage | Rechnung, Abschlagsrechnung, Schlussrechnung; PDF und E-Rechnung im Exportordner |
| `mahnung_vermerken` | rueckfrage | Zugang einer Mahnung festhalten; daraus folgt der Verzug |
| `abschlussbuchung` | rueckfrage | AfA-Lauf, Abgrenzung, Rückstellung, USt-Verrechnung, Steuerrückstellung, UG-Rücklage, Wertberichtigung, unfertige Leistungen, Fremdwährung, Anlage aktivieren (bei einem Investmentanteil mit Fondsart), Vorabpauschale eines Investmentanteils (ohne Buchung am Anlagegut festgehalten) |
| `erklaerung_eingabe` | rueckfrage | Angabe zu einer Zeile einer Jahreserklärung; was der Mensch gesetzt oder bestätigt hat, ändert nur er |
| `beleg_verwerfen` | rueckfrage | abgelegten Beleg mit Grund verwerfen; er bleibt im Archiv |
| `anlass_erfassen` | rueckfrage | Gesellschafterwechsel, Änderung der wirtschaftlich Berechtigten, Vorauszahlungsbescheid KSt oder GewSt |
| `darlehen_erfassen` | rueckfrage | Darlehensvertrag mit einem Gesellschafter anlegen oder mit Grund berichtigen, Vertrag als `beleg_id`; beim Berichtigen bleibt, was fehlt, und sind Auszahlungen gebucht, bleiben Konto, Gesellschafter und Richtung; `art: zinsen` bucht die Zinsen bis `zinsen_bis` mit Eigenbeleg der Zinsrechnung und vermerkt die Zinsperiode; `nur_vorschau` zeigt Konten, Stand zum heutigen Tag (`stichtag`) und Befunde zum Fremdvergleich |
| `ergebnisverwendung_buchen` | rueckfrage | Gesellschafterbeschluss über das Ergebnis im Folgejahr buchen: Rücklagen, Ausschüttung mit Kapitalertragsteuer, Vortrag; erst nach der Feststellung durch den Menschen |
| `verfahrensdoku_text` | rueckfrage | Organisationstexte der Verfahrensdokumentation lesen und ändern; ohne `texte` nur lesen, dafür genügt `buchfink:lesen`, auch im Prüfermodus |

Eine Auslage eines Gesellschafters bucht `beleg_buchen_mit_rueckfrage` mit
`bezahlung: auslage`; ein eigenes Werkzeug gibt es dafür nicht.

`fristen` nimmt als Art und im Schlüssel auch den Anker aus `pflichten.md` an.
Bei fünf Arten weicht er ab: `festschreibung` hat den Anker
`monatsfestschreibung`, `abschlussbuchungen` `bilanzstichtag`, `e-bilanz`
`ebilanz`, `e-bilanz-eroeffnung` `ebilanz-eroeffnung` und
`steuerliche-erfassung` `fragebogen-steuerliche-erfassung`.

Als erledigt gilt eine Vorauszahlung erst mit einer Zahlung, die vom Bankkonto
kommt oder die der Mensch gebucht hat. Eine Handbuchung eines Agenten belegt
keine.

## Fehlercodes

Ein Fehler kommt als `isError` mit `{"fehler": {code, nachricht,
naechster_schritt, werkzeug, gruende, wiederholbar}}`. `werkzeug` nennt das
Werkzeug, mit dem es weitergeht, oder `nur_app`. Schemaverstöße (falscher
Auswahlwert, Muster, fehlendes Pflichtfeld) meldet das SDK selbst mit einem
Text, der mit `validating` beginnt.

| Code | Bedeutung |
|---|---|
| `kein_mandant_offen` | in der App ist kein Mandant geöffnet |
| `mandant_nicht_freigegeben` | die App ist für den geöffneten Mandanten nicht zugelassen |
| `mandant_gewechselt` | `mandant_id` ist nicht der geöffnete Mandant; nichts geschrieben |
| `berechtigung_fehlt` | Scope `buchfink:lesen` oder `buchfink:buchen` fehlt |
| `pruefermodus_aktiv` | Prüfermodus, nur lesen; Exporte laufen weiter |
| `nur_in_app` | der Vorgang gehört dem Menschen in der App |
| `freigabe_noetig` | kein Routinefall; `gruende` nennt die Regeln, `werkzeug` das Werkzeug mit Rückfrage |
| `pfad_nicht_erlaubt` | Pfad liegt außerhalb von Eingang und Leseordnern, in einem gesperrten Ordner oder hat weitere harte Verweise |
| `datei_ungueltig` | Datei fehlt, Typ, Größe oder Inhalt passt nicht |
| `schluessel_wiederverwendet` | Idempotenzschlüssel mit anderer Eingabe |
| `beschaeftigt` | Schreibsperre länger als 30 Sekunden belegt; wiederholen |
| `ungueltige_eingabe` | Feld fehlt oder ist falsch, auch ein fehlender oder zu kurzer Idempotenzschlüssel |
| `nicht_gefunden` | Kennung unbekannt; die Nachricht nennt das Feld, `werkzeug` das Such-Werkzeug |
| `fachliche_regel` | der Dienst lehnt ab; die Nachricht nennt den Grund, `wiederholbar` sagt, ob ein späterer Versuch hilft (fehlendes Netz) |
| `intern` | Programmfehler |

Die stdio-Brücke für Claude Desktop antwortet im selben Format mit eigenen
Codes, bevor ein Aufruf den Server erreicht: `app_nicht_geoeffnet`,
`anmeldung_erforderlich` und `anmeldung_abgelehnt`
([packaging/claude-desktop](../../packaging/claude-desktop/README.md)).

## Regeln für direkt

`beleg_buchen` bucht nur, wenn alles zutrifft (`policy.ClassifyReceipt`).
Sonst nennt `gruende` jeden verletzten Schlüssel. Die Vorschau
(`nur_vorschau`) nennt sie auch.

| Schlüssel | Regel |
|---|---|
| `beleg_nicht_offen`, `keine_eingangsrechnung` | abgelegte Eingangsrechnung |
| `keine_darstellung` | der Beleg hat eine ansehbare Datei; eine reine XRechnung bekommt sie mit `beleg_ablegen` und `rolle: darstellung` |
| `keine_e_rechnung`, `e_rechnung_ungeprueft`, `e_rechnung_mit_befund`, `kein_vorschlag` | E-Rechnung mit strukturiertem Teil, geprüft, ohne Befund; Fotos und PDF ohne Datensatz nie |
| `kontakt_nicht_erkannt`, `kontakt_gesperrt` | der Aussteller ist über USt-IdNr. oder Steuernummer dieser Kontakt, nicht nur dem Namen nach; nicht gesperrt |
| `stammdaten_weichen_ab` | USt-IdNr., Steuernummer und Anschrift der E-Rechnung widersprechen den Stammdaten nicht |
| `iban_weicht_ab` | hat der Kontakt eine IBAN, verlangt die E-Rechnung keine Zahlung auf ein anderes Konto |
| `kopf_weicht_ab` | Belegdatum, Brutto, Aussteller und Betreff am Beleg sind die der E-Rechnung |
| `keine_bestaetigte_standardgruppe`, `gruppe_weicht_ab` | der Mensch hat die Standardgruppe in der App bestätigt; alle Positionen nutzen sie |
| `zu_wenige_buchungen` | mindestens so viele frühere, nicht stornierte Belegbuchungen des Lieferanten auf den Konten der Gruppe wie eingestellt (Vorgabe 3) |
| `betraege_weichen_ab` | Netto je Satz und Brutto centgenau wie in der E-Rechnung |
| `steuerfall_nicht_inland`, `fremdwaehrung` | Inland, Euro |
| `ueber_direktgrenze` | Brutto unter der Direktgrenze (Vorgabe 1.000 €; 1.000,00 € selbst ist Rückfrage) |
| `sonderangaben` | keine Bewirtung, kein Geschenk, kein Vorsteueranteil, keine Rückstellung, keine Anzahlung, keine Gruppe mit Sonderregel, kein Konto statt Gruppe, keine Übersteuerung, kein Befund zur Vorsteuer (`befund_vorsteuer`) |
| `bezahlart` | offen oder Firmenkreditkarte |
| `datum_in_zukunft` | Beleg- und Buchungsdatum nicht in der Zukunft |
| `dublettenverdacht` | kein anderer Beleg desselben Ausstellers mit derselben Rechnungsnummer (aus der E-Rechnung, in allen Geschäftsjahren) oder demselben Betrag am selben Tag |
| `vorschau` | nur in der `einstufung` von `beleg_lesen`: die naheliegende Buchung ließ sich nicht durchrechnen; der Text nennt den Grund |

Die Dublettenprüfung liest Rechnungsnummer und Steuerkennung des Ausstellers
aus dem strukturierten Teil; Buchfink speichert beide beim Auslesen am Beleg.
Kopfdaten im Aufruf von `beleg_ablegen` gelten bei einer E-Rechnung nicht.

`bankumsatz_zuordnen` bucht nur, wenn genau ein offener Posten in der Richtung
des Umsatzes centgenau den Betrag hat und seine Beleg-, Buchungs- oder
Rechnungsnummer im Verwendungszweck steht, und nur, wenn der Auszug geprüft
ist (`policy.ClassifyBankMatch`; Gründe `umsatz_nicht_offen`,
`fremdwaehrung`, `auszug_ungeprueft`, `kein_exakter_treffer`,
`mehrere_treffer`, `anderer_posten`). Geprüft ist ein Auszug, den der Mensch
in der App eingelesen hat, oder einer, dessen Salden stimmen: Anfangs- und
Endsaldo stehen darin, die Umsätze ergeben ihre Differenz, und der
Anfangssaldo ist der gemerkte Endsaldo des Kontos. Den Endsaldo merkt sich
das Konto nur aus einem Auszug des Menschen oder einem geprüften. Den ersten
Auszug eines Kontos liest deshalb der Mensch ein; `kontoauszug_importieren`
lehnt ihn mit `nur_app` ab, bevor er abgelegt ist
(`BankService.requireHumanAnchor`). Warum ein späterer Auszug ungeprüft bleibt,
nennt das Werkzeug in `hinweise` (`BankService.checkStatements`). Auszüge in
zeitlicher Reihenfolge einlesen.

Die Standardgruppe schlägt ein Agent nur vor (`standardgruppe_vorschlagen`,
`kontakt_anlegen`). Bestätigen kann sie nur der Mensch
(`ContactService.ConfirmDefaultPostingGroup`, Bridge
`ConfirmDefaultPostingGroup`), in der App unter Unternehmen › Kontakte in der
Seitenleiste des Lieferanten. `SaveContact` ändert die Felder nicht. Bis zur
Bestätigung bucht `beleg_buchen` diesen Lieferanten nicht direkt;
`beleg_buchen_mit_rueckfrage` geht immer.

Unternehmensdaten, Gründung und Bankkonten schlägt ein Agent ebenfalls nur vor
(`stammdaten_vorschlagen`, `MasterDataProposalService.Propose`); der Vorschlag
ändert nichts. Die App zeigt ihn als Vorschau „bisher → neu“ gegen den
heutigen Stand, in einem Dialog und unter Unternehmen › Stammdaten. Übernehmen
oder ablehnen kann nur der Mensch (`Accept`, `Reject`, Bridge
`AcceptMasterDataProposal`, `RejectMasterDataProposal`). Übernommen wird nur
die Vorschau, die er gesehen hat: die Bridge gibt ihren Fingerabdruck mit, und
hat sich der Stand inzwischen geändert, weist `Accept` ab. Geschrieben wird in
einer Transaktion über dieselben Dienstmethoden wie aus den Formularen
(`FoundationService.Save`, `BankService.ConfigureAccounts` und
`SetInvoiceAccount`, `SettingsService.UpdateCompanySettings`), mit ihren
Prüfungen. Rechtsform, Geschäftsjahr, Nummernkreise und Prüfregeln kann ein
Agent nicht vorschlagen. Die Entscheidung, bei einer Ablehnung mit Grund, steht
in `status` unter `stammdaten_vorschlaege`.

Betragsgrenze und Mindestzahl früherer Buchungen stellt der Mensch unter
Einstellungen › „Buchen ohne Rückfrage“ ein, Eingang, Exportordner und weitere
Leseordner unter Einstellungen › „Eingang & Ordner“
(`SettingsService.SaveAgentSettings`, Bridge `GetAgentRules`,
`SaveAgentRules` und `SetInboxFolder`).

Restrisiko: Buchfink prüft nicht, woher eine Datei im Eingang oder einem
Leseordner kommt. Wer dorthin schreiben kann (ein Agent mit Dateizugriff wie
Claude Code oder Cowork, eine Scanner-App, ein Download), kann eine
E-Rechnung im Namen eines Lieferanten erzeugen. USt-IdNr., Stammdaten, IBAN
und Dublette fangen die plumpe Fälschung ab, nicht die genaue. Die
Direktgrenze begrenzt den Schaden.

## Eingang, Leseordner und Exportordner

Jeder Mandant hat genau einen Eingang (`inbox.folder`, voreingestellt
`<Datenordner>/Eingang`) für Belege, E-Rechnungen und Kontoauszüge. Der Mensch
wählt ihn in der Einrichtung oder in den Einstellungen (Bridge
`SetupApplication`, `SetInboxFolder`); er darf auch in einem Sync-Ordner
liegen, in den eine Scanner-App am Telefon schreibt. Im Datenordner ist er die
einzige erlaubte Stelle. Daneben gibt es weitere Leseordner
(`agent.extra_read_folders`, voreingestellt keiner). Eingang und Leseordner
dürfen weder einen Datenordner noch eine Sicherung enthalten oder in einem
liegen, und ein Leseordner überschneidet sich nicht mit dem Eingang
(`policy.CheckFolders`). Geprüft und gespeichert wird derselbe Pfad
(`domain.CleanFolder`); Leerzeichen am Ende gehören zum Namen. Ein
Sicherungsordner darf nicht im Eingang oder einem Leseordner liegen
(`Host.SetBackupDir`). Beides stellt nur der Mensch in der App ein.

Buchfink legt nichts von selbst ab und beobachtet den Ordner nicht. Der Agent
listet den Eingang mit `eingang` und legt ab, wenn der Mensch es sagt. Die
Liste lässt gesperrte Ordner im Eingang aus, etwa eine Sicherung oder den
Datenordner eines anderen Mandanten. Ist der Eingang ein Symlink, liest
Buchfink den Ordner, auf den er zeigt; liegt der in einem gesperrten Ordner,
weist Buchfink den Eingang ab. Ist der Eingang nicht lesbar, etwa weil das
System der App den Zugriff verweigert, nennt `status` den Grund unter
`eingang.fehler`, und `aufgaben` führt ihn als Aufgabe für den Menschen; die
übrigen Werkzeuge arbeiten weiter.

Nach einer gelungenen Ablage mit `beleg_ablegen` oder
`kontoauszug_importieren` verschiebt Buchfink die Datei außerhalb der
Schreibsperre nach
`<Eingang>/Übernommen/<JJJJ-MM>/` (Monat des Ablagetags, bei gleichem Namen
„Name (2).pdf“), ebenso eine exakte Dublette, deren Inhalt schon im Archiv
liegt. Verschoben wird nur innerhalb des Eingangs: steht an der Stelle von
`Übernommen/` oder des Monatsordners ein Symlink oder eine Datei, bleibt die
Datei liegen, und ein Hinweis nennt den Grund. Eine vorhandene Datei
überschreibt Buchfink nie, auch nicht bei gleichzeitigen Ablagen. Die Antwort
nennt den neuen Pfad unter `verschoben`; ein zweites
Ablegen derselben Datei, etwa mit der Darstellung einer reinen XRechnung,
nimmt ihn; wer den alten Pfad nennt, bekommt den neuen in der Meldung von
`datei_ungueltig`. Scheitert das Verschieben, steht ein Hinweis in der Antwort, und
die Ablage gilt. Dateien in Leseordnern bleiben liegen. Die App verschiebt
genauso, wenn ein Beleg aus Datei oder ein Kontoauszug aus dem Eingang kommt.

Belege kommen nur als Pfad. `policy.ReadFile` prüft den Pfad zuerst ohne
Zugriff auf das Dateisystem: er muss bereinigt im Eingang oder einem
Leseordner liegen, UNC-, Geräte- und Netzwerkpfade weist es ab. Vor diesem
Treffer antwortet es einheitlich mit `pfad_nicht_erlaubt` und ohne
Systemmeldung; ob eine Datei außerhalb existiert, erfährt der Agent nicht.
Danach löst es Symlinks und `..` auf und prüft erneut, öffnet ohne Symlink im
letzten Glied, weist Dateien mit mehr als einem harten Verweis ab und prüft
nach dem Öffnen, dass die Datei noch an derselben Stelle liegt. Gelesen wird
einmal: PDF, PNG, JPEG, TIFF, WebP oder XML bis 50 MB, Inhalt passend zur
Endung, keine versteckten Dateien. Gesperrt sind der eigene Datenordner bis
auf den voreingestellten Eingang darin, die Datenordner aller anderen
Mandanten, auch wenn ein Leseordner sie umschließt oder der Eingang
inzwischen dorthin zeigt, und alle Sicherungsordner bis auf einen, der den
Eingang umschließt.

Voreingestellt liest ein Agent nur aus dem Eingang. Den Download-Ordner des
Systems nimmt Buchfink nicht von sich aus auf: jede Webseite und jeder
Mailanhang schreibt dorthin, und eine abgelegte Datei lässt sich nicht
zurücknehmen. Auch ein verworfener Beleg bleibt zehn Jahre im Archiv, in jeder
Sicherung und im Prüferexport. Wer den Download-Ordner trotzdem freigeben will,
nimmt ihn in der App als Leseordner auf. Ein Agent ohne eigenen Dateizugriff (Claude Desktop) bekommt
eine Rechnung, wenn der Mensch sie in den Eingang legt oder ins Fenster von
Buchfink zieht (`AddToInbox`). Die Beschreibung von `beleg_ablegen` verlangt,
nur Dateien abzulegen, die der Mensch als Beleg benannt hat.

`eingang_am` ist bei `beleg_ablegen` Pflicht: der Tag des Eingangs bestimmt
den Voranmeldungszeitraum der Vorsteuer (§ 15 Abs. 1 UStG). Liegt er in einem
späteren Zeitraum als das Belegdatum, nennt ein Hinweis beide Zeiträume;
berichtigen lässt er sich mit `beleg_buchen_mit_rueckfrage`.

Exporte schreiben nur in den Exportordner, voreingestellt
`<Datenordner>/Export`. Dorthin legt `beleg_datei_exportieren` die Kopie einer
Belegdatei und `rechnung_ausstellen` PDF und E-Rechnung zum Versand.
`ebilanz_exportieren` schreibt die XBRL-Instanz oder mit `format: susa_csv`
die Summen- und Saldenliste `ebilanz-<Jahr>-susa.csv` (UTF-8 mit BOM,
Semikolon, Dezimalkomma, CRLF; Spalten `Konto;Bezeichnung;Saldo;Taxonomie`,
Saldo im Soll positiv). `Taxonomie` ist die Position, unter der das Konto im
Kontennachweis der Instanz steht; Konten und Salden sind dieselben, Konten
ohne Saldo entfallen. Weicht die Steuerbilanz ab, hat die Liste die Salden
der Handelsbilanz; `ueberleitung` und `hinweis` nennen die Abweichungen, die
der Mensch im Dienst nachträgt. Mit `bilanzart: eroeffnungsbilanz` heißen die
Dateien `e-bilanz-eroeffnung-<Jahr>-<Hash>.xml` und
`ebilanz-eroeffnung-<Jahr>-susa.csv`; die Bilanzart steht im Dienst
(`EBilanzOptions.Kind`), der das Gründungsdatum der Unternehmensdaten als
Stichtag nimmt und jedes andere Jahr als das der Gründung abweist.

## Live-Aktualisierung

Was ein Agent schreibt, sieht der Mensch in der App ohne Neuladen. Nach jedem
gelungenen `Tenant.Write`, gleich auf welchem Erfassungsweg, und nach einer
Änderung im Eingang meldet der Kern einen `core.Change` mit Mandant,
Erfassungsweg und grober Art (`buchungen`, `belege`, `bank`, `stammdaten`,
`abschluss`, `eingang`); die Art liest er aus den Protokolleinträgen des
Vorgangs. Der Kern kennt dafür nur den Beobachter `OnChange` in den
Host-Optionen. Die Bridge bündelt die Meldungen je Mandant und sendet höchstens
alle 250 ms das Wails-Ereignis `buchfink:geaendert`. Die Oberfläche lädt die
offene Ansicht nach, wenn der Mandant der aktive ist (`useLiveReload`).
Werkzeuge und Skills müssen den Menschen deshalb nicht bitten, die Ansicht neu
zu laden.

## Ein Werkzeug ergänzen

1. Dienst zuerst: die fachliche Regel und jeder Vorbehalt der App
   (`requireHuman`) gehören in `internal/service`. Den Vorgang in
   `internal/ops` mit `Define(name, tier)` einordnen. Für `AppOnly` die
   Schutzstellen und Bridge-Methoden angeben und die Ausführungsproben in
   `TestAgentsAreRefusedForHumanOnlyOperations` ergänzen.
   Benötigte Dienstmethoden in `core.AgentServices` aufnehmen; die Fassade
   gibt weder vollständige Dienste noch Repositories heraus.
2. Ein- und Ausgabe als Go-Structs mit `json`- und `jsonschema`-Tags. Beträge
   als `Betrag`, Daten als `Datum`, Gruppen und Steuerfälle als
   `Buchungsgruppe`, `Steuerfall`, `Satz`. Ein Auswahlfeld bekommt einen
   eigenen Typ in `auswahl.go` mit Werteliste und Übersetzung in den Kern
   (`pick`, `label`). Listen in der Ausgabe nie `nil`.
3. Mit `AddRead`, `AddWrite`, `AddWritePrepared` (Vorlauf ohne
   Sperre, etwa ein Kurs aus dem Netz), `AddWriteThen` (Nachlauf nach dem
   Vorgang, etwa das Aufräumen des Eingangs) oder `AddFlow` (mehrere Vorgänge)
   registrieren. Der Werkzeugname bezeichnet den Katalogvorgang; nur bei einem
   abweichenden Namen wird `Spec.Operation` gesetzt. `Spec.Tier` kommt aus dem
   Katalog. Ohne natürlichen Schlüssel `KeyedWriteArgs`
   statt `WriteArgs`; eine Eingabe mit `nur_vorschau` implementiert
   `preview()`.
   Die Beschreibung sagt, wann das Werkzeug passt, wann nicht und welches
   danach kommt.
4. Ein direktes Werkzeug fragt `internal/core/policy` und antwortet sonst mit
   `errApproval`.
5. Tests über den Harness (`harness_test.go`): Erfolg mit Prüfung der Ausgabe
   gegen ihr Schema, Rechte, falscher Mandant, Prüfermodus, Idempotenz, bei
   direkt die Regelverletzung. Die Matrix in `catalog_test.go` braucht eine
   gültige Mindesteingabe des Werkzeugs.
6. Diese Tabelle ergänzen.
