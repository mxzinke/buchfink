# MCP-Server

[Dokumentation](../README.md) · [Entwicklung](README.md)

Diese Seite ist die Quelle für die Skills der verbundenen Apps. Ändert sich ein
Werkzeug, ein Fehlercode oder eine Regel, ändert sich diese Seite im selben
Commit. `TestDocsListEveryTool` hält Tabelle und Katalog gleich.

## Aufbau

Der MCP-Server läuft im Prozess der App auf `127.0.0.1:7465` (Streamable HTTP,
eigener OAuth-Autorisierungsserver, Zustimmung im Fenster der App). Claude
Desktop verbindet sich über die stdio-Brücke `cmd/buchfink-mcp`. Jedes Werkzeug
geht durch denselben Rahmen in `internal/mcpserver/tools.go`: Anmeldung lesen,
geöffneten Mandanten mit Lease holen, Zulassung und Rechte für diesen Mandanten
prüfen, den Erfassungsweg `agent:<kurzname>` setzen und über `Tenant.Read` oder
`Tenant.Write` in die Dienste gehen. Werkzeuge rufen nur Dienste auf, nie
Repositories (`TestToolsDoNotTouchRepositories`). Die Regeln für Routine stehen
in `internal/core/policy`, die Sperren für Agenten in den Diensten, nicht im
Werkzeug und nicht im Prompt.

## Freigabestufen

| Stufe | Annotation | Bedeutung |
|---|---|---|
| lesen | `readOnlyHint: true` | ändert nichts; jede zugelassene App mit `buchfink:lesen` |
| direkt | `destructiveHint: false` | schreibt ohne Rückfrage, aber nur, was `internal/core/policy` als Routine einstuft; sonst `freigabe_noetig` |
| rueckfrage | `destructiveHint: true` | der Agent legt dem Menschen den Vorgang im Chat vor und holt sein Ja ein, bevor er aufruft |
| nur App | kein Werkzeug | Festschreibung, Übermittlung und Transferticket, Pflicht als erledigt, Angaben einer Erklärung bestätigen, Aufstellen, Feststellen, Offenlegen, Ergebnisverwendung, Stammdaten samt Bankkonten und steuerlicher Beratung, Darlehensverträge, Standardgruppe bestätigen, Prüfermodus, Apps zulassen und entziehen, Regeln für Apps |

Ein `_meta`-Feld `anthropic/requiresUserInteraction` setzt der Server nicht.
Der Agent fragt laut Skill selbst; eine zweite Abfrage des Clients wäre doppelt.
Die Dienste weisen Vorgänge der App für jeden Erfassungsweg außer `ui` ab
(`requireHuman`, `core.AppOnly`), das Werkzeug meldet dann `nur_in_app`.

Daneben haben einige Konten einen Fachweg, an dem ein Agent nicht vorbeibucht.
Handbuchung, Neubuchung beim Storno und die Aufteilung eines Bankumsatzes
weisen für ihn ab: Kapital, Rücklagen und Ergebnisvortrag (2000 bis 2999),
offene Ausschüttung (3519), einbehaltene Kapitalertragsteuer (3760) und die
Gesellschafterkonten (0960, 1307 bis 1309, 3640 bis 3643). Die Zahlung einer
beschlossenen Ausschüttung über die Bank bleibt möglich, ein Darlehen über
`bankumsatz_buchen` mit `darlehen_id` auch.

## Eingaben und Ausgaben

Schreibende Werkzeuge verlangen `mandant_id` aus `status`. Wer in einem
Vorgang schreibt, nimmt einen `idempotenz_schluessel` (8 bis 100 Zeichen).
Derselbe Schlüssel mit derselben Eingabe liefert das erste Ergebnis mit
`wiederholt: true`, mit anderer Eingabe `schluessel_wiederverwendet`. Pflicht
ist er bei den Werkzeugen ohne natürlichen Schlüssel: `buchung_erfassen`,
`rechnung_ausstellen`, `abschlussbuchung`, `mahnung_vermerken` und
`anlass_erfassen`. Ein Aufruf mit `nur_vorschau` verbraucht den Schlüssel
nicht; derselbe Schlüssel bucht danach. Die Werkzeuge aus mehreren Schritten
(`kontoauszug_importieren`, die drei Exporte) nehmen keinen Schlüssel, sie
schreiben von sich aus nichts doppelt.

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
80 KB.

## Werkzeuge

| Werkzeug | Stufe | Zweck |
|---|---|---|
| `status` | lesen | Mandant mit USt-Zeitraum, Dauerfrist, festgeschrieben bis, letzter Sicherung; Geschäftsjahre, Prüfermodus, Belegordner, Exportordner, Direktregeln |
| `aufgaben` | lesen | Arbeitsliste mit dem Werkzeug für den nächsten Schritt |
| `fristen` | lesen | Pflichtenkalender mit Status, Kanal, Zuständigkeit, `anker` und Verweis auf `pflichten.md#<anker>` |
| `konten` | lesen | Kontensuche im SKR04, Buchungsgruppen, Steuerfälle |
| `kontakte_suchen` | lesen | Kunden und Lieferanten mit Saldo und Standardgruppe |
| `belege_suchen` | lesen | Belege nach Status, Art, Richtung, Datum, Text |
| `beleg_lesen` | lesen | Kopfdaten, Dateien, Befunde, Vorschlag aus der E-Rechnung samt bestätigter Standardgruppe, Einstufung |
| `buchungen_suchen` | lesen | Journal nach Zeitraum, Konto, Kontakt, Beleg, Betrag, Erfassungsweg |
| `buchung_lesen` | lesen | eine Buchung mit allen Zeilen und Storno-Bezügen |
| `offene_posten` | lesen | Forderungen und Verbindlichkeiten, Altersstruktur, Verzugszinsen, Mahnstand |
| `bankumsaetze` | lesen | Bankumsätze mit den besten Zuordnungsvorschlägen, Einstufung und Prüfstand des Auszugs |
| `auswertung` | lesen | Summen und Salden, Kontoblatt eines Sachkontos, Bilanz, GuV, USt-Übersicht, Prüfbericht |
| `anlagen` | lesen | Anlagenverzeichnis, fällige AfA, Aktivierungskandidaten |
| `abschluss_status` | lesen | Schritte des Jahresabschlusses mit nächster Aktion |
| `erklaerung` | lesen | UStVA, ZM, USt-, KSt- und GewSt-Erklärung formularnah mit Herkunft |
| `rechnung_vorschau` | lesen | Ausgangsrechnung durchrechnen ohne Ausstellen |
| `beleg_ablegen` | direkt | Dateien aus einem Belegordner ablegen, E-Rechnung auslesen und prüfen, fehlende Dateien an einem offenen Beleg ergänzen |
| `beleg_datei_exportieren` | direkt | Kopie einer Belegdatei in den Exportordner, zum Lesen oder Versenden |
| `kontakt_anlegen` | direkt | Kontakt anlegen nach Dublettenprüfung |
| `kontoauszug_importieren` | direkt | CAMT.053 ablegen, Salden prüfen, Umsätze einlesen |
| `beleg_buchen` | direkt | Eingangsrechnung buchen, nur Routine |
| `bankumsatz_zuordnen` | direkt | Bankumsatz mit genau einem offenen Posten ausgleichen, nur Routine |
| `erklaerung_exportieren` | direkt | Jahreserklärung als JSON in den Exportordner |
| `ebilanz_exportieren` | direkt | E-Bilanz prüfen und als XBRL in den Exportordner |
| `pflicht_vorbereitet` | direkt | Pflicht als vorbereitet melden, nie als erledigt |
| `standardgruppe_vorschlagen` | direkt | Standard-Buchungsgruppe eines Lieferanten vorschlagen |
| `beleg_buchen_mit_rueckfrage` | rueckfrage | Eingangsbeleg in allen Nicht-Routinefällen, auch Auslage; ergänzt fehlende Kopfdaten und berichtigt `eingang_am` |
| `bankumsatz_buchen` | rueckfrage | Aufteilung mit Skonto, Gebühren, Darlehen; Kreditkartenabrechnung; Ignorieren mit Grund |
| `buchung_erfassen` | rueckfrage | Handbuchung mit Beleg oder Eigenbeleg, auch in der Abschlussperiode; ein Kontakt an den Personenkontozeilen macht sie zum offenen Posten |
| `buchung_stornieren` | rueckfrage | Generalumkehr mit Grund, auf Wunsch mit Neubuchung; `datum` nur beim Saldenvortrag |
| `lohnjournal_buchen` | rueckfrage | Monatssummen des Lohnbüros |
| `rechnung_ausstellen` | rueckfrage | Rechnung, Abschlagsrechnung, Schlussrechnung; PDF und E-Rechnung im Exportordner |
| `mahnung_vermerken` | rueckfrage | Zugang einer Mahnung festhalten; daraus folgt der Verzug |
| `abschlussbuchung` | rueckfrage | AfA-Lauf, Abgrenzung, Rückstellung, USt-Verrechnung, Steuerrückstellung, UG-Rücklage, Wertberichtigung, unfertige Leistungen, Fremdwährung, Anlage aktivieren |
| `erklaerung_eingabe` | rueckfrage | Angabe zu einer Zeile einer Jahreserklärung; was der Mensch gesetzt oder bestätigt hat, ändert nur er |
| `beleg_verwerfen` | rueckfrage | abgelegten Beleg mit Grund verwerfen; er bleibt im Archiv |
| `anlass_erfassen` | rueckfrage | Gesellschafterwechsel, Änderung der wirtschaftlich Berechtigten, Vorauszahlungsbescheid KSt oder GewSt |

Eine Auslage eines Gesellschafters bucht `beleg_buchen_mit_rueckfrage` mit
`bezahlung: auslage`; ein eigenes Werkzeug gibt es dafür nicht.

`fristen` nimmt als Art und im Schlüssel auch den Anker aus `pflichten.md` an.
Bei vier Arten weicht er ab: `festschreibung` hat den Anker
`monatsfestschreibung`, `abschlussbuchungen` `bilanzstichtag`, `e-bilanz`
`ebilanz` und `steuerliche-erfassung` `fragebogen-steuerliche-erfassung`.

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
| `nur_in_app` | der Vorgang gehört dem Menschen in der App, auch ein Konto mit Fachweg |
| `freigabe_noetig` | kein Routinefall; `gruende` nennt die Regeln, `werkzeug` das Werkzeug mit Rückfrage |
| `pfad_nicht_erlaubt` | Pfad liegt außerhalb der Belegordner, in einem gesperrten Ordner oder hat weitere harte Verweise |
| `datei_ungueltig` | Datei fehlt im Belegordner, Typ, Größe oder Inhalt passt nicht |
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
das Konto nur aus einem Auszug des Menschen oder einem geprüften. Der erste
Auszug, den ein Agent einliest, ist deshalb ungeprüft, bis der Mensch einmal
einen Auszug in der App eingelesen hat. Auszüge in zeitlicher Reihenfolge
einlesen.

Die Standardgruppe schlägt ein Agent nur vor (`standardgruppe_vorschlagen`,
`kontakt_anlegen`). Bestätigen kann sie nur der Mensch
(`ContactService.ConfirmDefaultPostingGroup`, Bridge
`ConfirmDefaultPostingGroup`). `SaveContact` ändert die Felder nicht. Die
Schwellen und Ordner stellt der Mensch in der App ein
(`SettingsService.SaveAgentSettings`, Bridge `GetAgentRules` und
`SaveAgentRules`; `receiptFolders: null` heißt Voreinstellung, eine leere
Liste sperrt alle Belegordner).

Bekannte Lücke: die Oberfläche für diese drei Bridge-Methoden fehlt noch. Bis
sie da ist, gibt es keine bestätigte Standardgruppe, und `beleg_buchen` bucht
nichts direkt; `beleg_buchen_mit_rueckfrage` geht immer. Die Hinweise der
Werkzeuge sagen das so.

Restrisiko: Buchfink prüft nicht, woher eine Datei im Belegordner kommt. Wer
dorthin schreiben kann (ein Agent mit Dateizugriff wie Claude Code oder
Cowork, ein Download), kann eine E-Rechnung im Namen eines Lieferanten
erzeugen. USt-IdNr., Stammdaten, IBAN und Dublette fangen die plumpe
Fälschung ab, nicht die genaue. Die Direktgrenze begrenzt den Schaden.

## Belegordner und Exportordner

Belege kommen nur als Pfad. `policy.ReadFile` prüft den Pfad zuerst ohne
Zugriff auf das Dateisystem: er muss bereinigt in einem Belegordner liegen,
UNC-, Geräte- und Netzwerkpfade weist es ab. Vor diesem Treffer antwortet es
einheitlich mit `pfad_nicht_erlaubt` und ohne Systemmeldung; ob eine Datei
außerhalb existiert, erfährt der Agent nicht. Danach löst es Symlinks und `..`
auf und prüft erneut, öffnet ohne Symlink im letzten Glied, weist Dateien mit
mehr als einem harten Verweis ab und prüft nach dem Öffnen, dass die Datei
noch an derselben Stelle liegt. Gelesen wird einmal: PDF, PNG, JPEG, TIFF,
WebP oder XML bis 50 MB, Inhalt passend zur Endung, keine versteckten
Dateien. Gesperrt sind der eigene Datenordner bis auf seinen Eingang, die
Datenordner aller anderen Mandanten und alle Sicherungsordner, auch wenn ein
Belegordner sie umschließt. `SaveAgentRules` nimmt einen Belegordner nicht an,
der einen dieser Ordner enthält oder darin liegt.

Voreinstellung der Belegordner ist allein `<Datenordner>/Belegeingang`.
Downloads ist es nicht, obwohl dort Anhänge und Portal-Downloads landen. Jede
Webseite und jeder Mailanhang schreibt dorthin, und eine abgelegte Datei
lässt sich nicht zurücknehmen: auch ein verworfener Beleg bleibt zehn Jahre im
Archiv, in jeder Sicherung und im Prüferexport. Ein Agent ohne eigenen
Dateizugriff (Claude Desktop) bekommt eine Rechnung, wenn der Mensch sie in
den Eingang legt; wer Downloads will, nimmt den Ordner in der App auf. Die
Beschreibung von `beleg_ablegen` verlangt, nur Dateien abzulegen, die der
Mensch als Beleg benannt hat.

`eingang_am` ist bei `beleg_ablegen` Pflicht: der Tag des Eingangs bestimmt
den Voranmeldungszeitraum der Vorsteuer (§ 15 Abs. 1 UStG). Liegt er in einem
späteren Zeitraum als das Belegdatum, nennt ein Hinweis beide Zeiträume;
berichtigen lässt er sich mit `beleg_buchen_mit_rueckfrage`.

Exporte schreiben nur in den Exportordner, voreingestellt
`<Datenordner>/Export`. Dorthin legt `beleg_datei_exportieren` die Kopie einer
Belegdatei und `rechnung_ausstellen` PDF und E-Rechnung zum Versand.

## Ein Werkzeug ergänzen

1. Dienst zuerst: die fachliche Regel und jede Sperre für Agenten
   (`requireHuman`, Fachweg-Konten) gehören in `internal/service`, nicht ins
   Werkzeug.
2. Ein- und Ausgabe als Go-Structs mit `json`- und `jsonschema`-Tags. Beträge
   als `Betrag`, Daten als `Datum`, Gruppen und Steuerfälle als
   `Buchungsgruppe`, `Steuerfall`, `Satz`. Ein Auswahlfeld bekommt einen
   eigenen Typ in `auswahl.go` mit Werteliste und Übersetzung in den Kern
   (`pick`, `label`). Listen in der Ausgabe nie `nil`.
3. Registrieren mit `AddRead`, `AddWrite`, `AddWritePrepared` (Vorlauf ohne
   Sperre, etwa ein Kurs aus dem Netz) oder `AddFlow` (mehrere Vorgänge) und der
   Stufe in `Spec.Tier`. Ohne natürlichen Schlüssel `KeyedWriteArgs` statt
   `WriteArgs`; eine Eingabe mit `nur_vorschau` implementiert `preview()`.
   Die Beschreibung sagt, wann das Werkzeug passt, wann nicht und welches
   danach kommt.
4. Ein direktes Werkzeug fragt `internal/core/policy` und antwortet sonst mit
   `errApproval`.
5. Tests über den Harness (`harness_test.go`): Erfolg mit Prüfung der Ausgabe
   gegen ihr Schema, Rechte, falscher Mandant, Prüfermodus, Idempotenz, bei
   direkt die Regelverletzung. Die Matrix in `catalog_test.go` braucht eine
   gültige Mindesteingabe des Werkzeugs.
6. Diese Tabelle ergänzen.
