# J. Querschnitt

[Anforderungskatalog](README.md) · [Legende](README.md#legende) ·
[Datensicherheit](../nutzung/datensicherheit.md)

## QUE-01 Löschpflicht und Aufbewahrungspflicht

MUSS · Art. 5 Abs. 1 lit. e, Art. 17 Abs. 3 lit. b DSGVO, § 257 HGB, § 147 AO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Löschverlangen mit Normverweis zurückgestellt statt abgelehnt | erfüllt | `accounting.BlockedContactAnswer` formuliert die Antwort |
| Betroffene Datensätze bis Fristablauf gesperrt | erfüllt | `service.ContactService.BlockContact`; mit einem gesperrten Partner beginnt kein neues Geschäft, offene Posten lassen sich ausgleichen |
| Nach Fristablauf löschen oder Entscheidung vorlegen, Löschung protokolliert | nicht im Umfang | Buchfink löscht keine Buchführungsdaten; das Löschen nach Fristablauf geschieht außerhalb |
| Löschkonzept mit Frist und Rechtsgrundlage je Datenkategorie | erfüllt | `accounting.DeletionConcept`, Teil der Verfahrensdokumentation |

## QUE-02 Technische Maßnahmen und Auftragsverarbeitung

MUSS · Art. 28, Art. 32 DSGVO

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Übertragung und Speicherung verschlüsselt, Verfahren dokumentiert | teilweise | Netzdienste über HTTPS; der MCP-Server lauscht nur auf 127.0.0.1. Gespeichert wird unverschlüsselt; die Verfahrensdokumentation verweist auf die Festplattenverschlüsselung des Betriebssystems |
| Zugriffe auf personenbezogene Daten protokolliert | erfüllt | Schreibvorgänge, Exporte und Datenüberlassungen im Änderungsprotokoll, mit Erfassungsweg |
| Auftragsverarbeitungsvertrag | nicht im Umfang | Buchfink verarbeitet nichts im Auftrag. Wer einen KI-Agenten verbindet, gibt Daten an dessen Anbieter; den Vertrag schließt das Unternehmen mit ihm ([KI-Anbieter](../nutzung/datensicherheit.md#ki-anbieter)) |
| Verzeichnis von Verarbeitungstätigkeiten | erfüllt | Abschnitt der Verfahrensdokumentation (Art. 30 DSGVO) |

## QUE-03 Mandantentrennung

MUSS · § 146 Abs. 1 AO, GoBD Rz 36 ff.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Buchungen wirken nur in einem Mandanten | erfüllt | eine Datenbank je Mandant, genau ein Mandant geöffnet (`core.Host`) |
| Auswertungen und Exporte mandantenscharf | erfüllt | jede Auswertung liest über den geöffneten Mandanten |
| Berechtigungen je Mandant | teilweise | Verbundene Apps erhalten Rechte je Mandant (`oauth.Store.ActiveGrant`); Personenrechte gibt es nicht |
| Nummernkreise je Mandant unabhängig | erfüllt | `domain.NumberRange` in der Mandantendatenbank |

## QUE-04 Zeit und Zeitstempel

MUSS · § 146 Abs. 1 AO, GoBD Rz 45 ff., 107 ff.

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Zeitstempel in UTC gespeichert, mit Zeitzone ausgegeben | erfüllt | [Architektur, Regel 5](../entwicklung/architektur.md#2-aufbau) |
| Systemzeit gegen eine Quelle geprüft, Abweichung protokolliert | erfüllt | Vergleich mit der Zeit des Zeitstempeldienstes bei der Festschreibung (`service.TimeDriftNote`) |
| Erfassungszeitpunkt nicht setzbar, Beleg- und Buchungsdatum getrennt | erfüllt | `service.JournalService.Post` setzt ihn selbst |
| Zeitstempel in Protokollen geschützt wie Buchungen | erfüllt | Zeitpunkt in der Hash-Kette des Protokolls (`accounting.AuditChain`) |

## QUE-05 Zahlungsziele und Verzug

SOLL · §§ 271a, 286, 288 BGB

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Basiszinssatz als datierte Reihe | teilweise | `accounting.DefaultBaseRates` im Programm; neue Sätze kommen mit einem Programmupdate |
| Verzugszinsen taggenau, Aufschlag nach Verbraucher und Unternehmer | erfüllt | `accounting.DefaultInterest`, `service.OverdueService.Claims` |
| Pauschale von 40 Euro bei Unternehmern | erfüllt | `accounting.DefaultInterestLumpSum` |
| Hinweis bei Zahlungszielen über 60 Tage | erfüllt | `domain.PaymentTermNotice` |
| Mahnstufen, Gebühren, archivierte Mahnungen | nicht im Umfang | Kein Mahnwesen; der Agent verfolgt Forderungen anhand der überfälligen Posten |

## QUE-06 Auskunft und Rechenschaft

SOLL · § 259 BGB, §§ 42a, 51a GmbHG

| Kriterium | Stand | Fundstelle oder Grund |
|---|---|---|
| Lesender Zugang für Gesellschafter, Umfang einstellbar, protokolliert | teilweise | Prüfermodus wie PRF-01; der Umfang ist nicht je Person einstellbar |
| Zusammenstellung der Einnahmen und Ausgaben mit Belegverweisen | erfüllt | `service.ExportService.ExportJournalCSV` mit Belegnummer und Beleg-Hash |
| Jahresabschluss in der Frist des § 42a GmbHG | teilweise | Pflichten `aufstellung` und `feststellung` im Pflichtenkalender; die Vorlage an die Gesellschafter geschieht außerhalb |
