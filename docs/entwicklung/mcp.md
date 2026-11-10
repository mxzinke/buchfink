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

`openWorldHint` ist nur bei `beleg_buchen_mit_rueckfrage` gesetzt, das
Wechselkurse aus dem Netz holt. Den Titel trägt das Werkzeug selbst, nicht noch
einmal die Annotation. Ein `_meta`-Feld `anthropic/requiresUserInteraction`
setzt der Server nicht.
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
aus `Toolbox.OutputSchema`, `TestCatalogStaysSmall` begrenzt die Größe des
Katalogs. Was für alle schreibenden Werkzeuge gilt (`mandant_id`,
`idempotenz_schluessel`), steht einmal in den Anweisungen des Servers und in
den Schemas nur kurz.

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
| `beleg_ablegen` | direkt | Dateien aus Eingang oder Leseordner ablegen, E-Rechnung auslesen und prüfen, bei einer E-Rechnung Vorschlag und Einstufung wie `beleg_lesen`, fehlende Dateien an einem offenen Beleg ergänzen; `art: brief` legt ein Schreiben ohne Buchung ab, `art: sonstiges` einen Vertrag oder Beschluss für `buchung_erfassen`; danach geht die Datei aus dem Eingang nach `Übernommen/` |
| `beleg_datei_exportieren` | direkt | Kopie einer Belegdatei in den Exportordner, zum Lesen oder Versenden |
| `kontakt_anlegen` | direkt | Kontakt anlegen nach Dublettenprüfung; mit `leitweg_id` bekommt ein Kunde XRechnung |
| `kontoauszug_importieren` | direkt | CAMT.053 ablegen, Salden prüfen, Umsätze einlesen; danach geht die Datei aus dem Eingang nach `Übernommen/` |
| `beleg_buchen` | direkt | Eingangsrechnung buchen, nur Routine; `faellig_am` übernimmt die Fälligkeit vom Beleg, ohne sie meldet die Vorschau `faelligkeit_aus_zahlungsziel` |
| `bankumsatz_zuordnen` | direkt | Bankumsatz mit genau einem offenen Posten ausgleichen, nur Routine |
| `erklaerung_exportieren` | direkt | Jahreserklärung als JSON in den Exportordner |
| `ebilanz_exportieren` | direkt | E-Bilanz prüfen und in den Exportordner: `format` `xbrl` (Vorgabe) oder `susa_csv`; bei abweichender Steuerbilanz nennt `ueberleitung`, was die Liste nicht enthält; `bilanzart: eroeffnungsbilanz` schreibt die Eröffnungsbilanz, `jahr` ist dann das Geschäftsjahr der Gründung |
| `pflicht_vorbereitet` | direkt | Pflicht als vorbereitet melden, nie als erledigt |
| `stammdaten_vorschlagen` | direkt | Unternehmensdaten, Gründung oder Bankkonten vorschlagen; ändert nichts, der Mensch übernimmt nach der Vorschau in der App |
| `standardgruppe_vorschlagen` | direkt | Standard-Buchungsgruppe eines Lieferanten vorschlagen |
| `beleg_buchen_mit_rueckfrage` | rueckfrage | Eingangsbeleg in allen Nicht-Routinefällen, auch Auslage; ergänzt fehlende Kopfdaten und berichtigt `eingang_am`; `faellig_am` wie bei `beleg_buchen`; `brutto` statt `netto` bei Kleinbetragsrechnung und Fahrausweis; `nicht_steuerbar` je Position; `gutschrift_zu` bucht die Gutschrift eines Lieferanten mit getauschten Seiten; `belastet` bucht einen bezahlten Fremdwährungsbeleg mit dem belasteten Euro-Betrag |
| `bankumsatz_buchen` | rueckfrage | Aufteilung mit Skonto, Gebühren, Darlehen; ein ganz ausgeglichener Fremdwährungsposten bekommt seine Kursdifferenz ohne Angabe (bis 10 %), eine Gutschrift wird mit Posten der Gegenrichtung verrechnet; Kreditkartenabrechnung mit Kurs- und Rundungsdifferenzen, eine Abweichung im Rahmen der Fremdwährungsbelege gilt ohne Angabe als Kursdifferenz; Kapitalertrag einer Finanzanlage, brutto mit Kapitalertragsteuer (7630) und Solidaritätszuschlag (7633), mit dem Anlagegut verknüpft (`AssetService.BookAssetIncome`); `beleg_id` versiegelt bei einer Aufteilung den abgelegten Beleg zum Grund der Zahlung, etwa einen Vertrag, mit der Buchung; Ignorieren mit Grund |
| `buchung_erfassen` | rueckfrage | Handbuchung mit Beleg oder Eigenbeleg, auch in der Abschlussperiode und für die Gründung (Zeichnung, Einforderung); ein Kontakt an den Personenkontozeilen macht sie zum offenen Posten, an Gesellschafterkonten ist er Pflicht; Darlehenszinsen nicht hier, sondern mit `darlehen_erfassen` |
| `buchung_stornieren` | rueckfrage | Generalumkehr mit Grund, auf Wunsch mit Neubuchung; `datum` nur beim Saldenvortrag |
| `lohnjournal_buchen` | rueckfrage | Monatssummen des Lohnbüros |
| `rechnung_ausstellen` | rueckfrage | Rechnung, Abschlagsrechnung, Schlussrechnung; PDF und E-Rechnung im Exportordner; gebucht zum Ende des Leistungszeitraums (§ 252 Abs. 1 Nr. 4 HGB), bei festgeschriebenem Leistungstag zum Rechnungsdatum |
| `mahnung_vermerken` | rueckfrage | Zugang einer Mahnung festhalten; daraus folgt der Verzug |
| `abschlussbuchung` | rueckfrage | AfA-Lauf, Abgrenzung, Rückstellung, USt-Verrechnung, Steuerrückstellung, UG-Rücklage, Wertberichtigung, unfertige Leistungen, Fremdwährung, Anlage aktivieren (`anlage_aktivieren`, bei einem Investmentanteil mit Fondsart), Vorabpauschale eines Investmentanteils (ohne Buchung am Anlagegut festgehalten) |
| `erklaerung_eingabe` | rueckfrage | Angabe zu einer Zeile einer Jahreserklärung; was der Mensch gesetzt oder bestätigt hat, ändert nur er |
| `beleg_verwerfen` | rueckfrage | abgelegten Beleg mit Grund verwerfen; er bleibt im Archiv |
| `anlass_erfassen` | rueckfrage | Gesellschafterwechsel, Änderung der wirtschaftlich Berechtigten, Vorauszahlungsbescheid KSt oder GewSt |
| `darlehen_erfassen` | rueckfrage | Darlehensvertrag mit einem Gesellschafter anlegen oder mit Grund berichtigen, Vertrag als `beleg_id`; beim Berichtigen bleibt, was fehlt, und sind Auszahlungen gebucht, bleiben Konto, Gesellschafter und Richtung; `art: zinsen` bucht die Zinsen bis `zinsen_bis` mit Eigenbeleg der Zinsrechnung und vermerkt die Zinsperiode; `vermerk_beleg_id` ordnet einen abgelegten Vermerk zur Zinsentscheidung zu, der den Fremdvergleich ohne Vergleichszins belegt; `nur_vorschau` zeigt Konten, Stand zum heutigen Tag (`stichtag`) und Befunde zum Fremdvergleich |
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
naechster_schritt, werkzeug, argumente, gruende, wiederholbar}}`. `werkzeug`
nennt das Werkzeug, mit dem es weitergeht, oder `nur_app`. `argumente` steht
da, wenn dessen Eingabe anders aussieht als die des gescheiterten Aufrufs, etwa
`bankumsatz_buchen` nach `bankumsatz_zuordnen`; die Tests prüfen jeden solchen
Folgeaufruf und jede Aufgabe aus `aufgaben` gegen das Eingabeschema. Schemaverstöße (falscher
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
| `keine_darstellung` | der Beleg lässt sich zeigen: als PDF- oder Bilddatei oder als Ansicht seines Rechnungsdatensatzes; eine reine XRechnung braucht kein PDF |
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

Standardgruppen und Stammdaten kann der Agent vorschlagen. Der Mensch
bestätigt sie in der App. Die Einstellungen für Routinebuchungen und Ordner
ändert ebenfalls nur der Mensch.

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

Nach der Ablage verschiebt Buchfink Dateien aus dem Eingang nach
`Übernommen/<JJJJ-MM>/`. Dateien aus Leseordnern bleiben liegen. Scheitert
das Verschieben, bleibt die Ablage bestehen und die Antwort nennt den Grund.

`policy.ReadFile` begrenzt den Zugriff auf zugelassene Ordner und prüft Pfad,
Dateityp und Größe. Ein Agent ohne eigenen Dateizugriff kann Dateien verarbeiten,
die der Mensch in den Eingang legt oder in die App zieht.

`eingang_am` ist bei `beleg_ablegen` Pflicht: der Tag des Eingangs bestimmt
den Voranmeldungszeitraum der Vorsteuer (§ 15 Abs. 1 UStG). Liegt er in einem
späteren Zeitraum als das Belegdatum, nennt ein Hinweis beide Zeiträume;
berichtigen lässt er sich mit `beleg_buchen_mit_rueckfrage`.

Exporte schreiben in den Exportordner, voreingestellt `<Datenordner>/Export`.
Die Übermittlung der E-Bilanz beschreibt die [Anleitung](../nutzung/README.md#e-bilanz-übermitteln).

## Live-Aktualisierung

Änderungen eines Agenten erscheinen ohne Neuladen in der App. Der Kern meldet
`core.Change`; die Bridge bündelt die Meldungen und die Oberfläche lädt die
betroffene Ansicht nach (`useLiveReload`).

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
