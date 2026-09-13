# H. Jahresabschluss, E-Bilanz, Offenlegung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### JAB-01 Aufstellung, Gliederung, Fristen `MUSS`

**Norm:** §§ 242 bis 245, 264, 266, 275 HGB

**Bedeutung:** Kapitalgesellschaften erweitern den Jahresabschluss um Anhang und, ab mittelgroß, um einen Lagebericht. Mittelgroße und große Kapitalgesellschaften stellen innerhalb der ersten drei Monate des Folgejahres auf, kleine innerhalb von sechs Monaten.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Bilanz in Kontoform nach § 266 HGB und GuV in Staffelform nach § 275 HGB, mit Vorjahresvergleich | ✅ | internal/accounting/statement.go:64-148 ist die Bilanz in Kontoform, :150-188 die Staffel des § 275 Abs. 2 HGB; :291-358 (`BuildStatement`) verteilt die Salden über die Positionsdaten der Konten (internal/domain/account.go:30-34), bucht ein Konto mit widersprechendem Vorzeichen auf die Gegenposition (:465-497) und weist eine Bilanz ab, die nicht aufgeht (:354-356, :363-378). internal/service/statement_service.go:178-201 lädt das Vorjahr und stellt seine Spalte nach § 265 Abs. 2 HGB daneben, frontend/src/components/StatementView.tsx:152-239 zeigt beides | – |
| Gesamtkosten- und Umsatzkostenverfahren beide wählbar | ⛔ | Buchfink führt nur das Gesamtkostenverfahren; die SKR04-Gliederung GuV.1 bis GuV.16 entspricht ihm | – |
| Gliederungstiefe folgt der Größenklasse | ✅ | internal/domain/statement.go:51-74 begrenzt die Ebenen je Tiefe, internal/accounting/groessenklasse.go:250-271 leitet sie aus der Klasse ab (§ 266 Abs. 1 Sätze 3 und 4 HGB), internal/service/statement_service.go:95-108 wendet sie an, wenn keine gewählt ist; die Merkmale der Größenklasse entstehen dabei immer aus der Vollgliederung (:83-93). Die Wahl steht auch in der Ansicht (frontend/src/components/StatementView.tsx:54-59) | – |
| Pflichtangaben nach § 264 Abs. 1a HGB auf dem Abschluss | ✅ | internal/service/statement_service.go:322-360 (`header`) setzt Firma, Rechtsform, Sitz, Registergericht und Registernummer in den Kopf und benennt jede fehlende Angabe; frontend/src/components/StatementView.tsx:250-282 zeigt sie mit dem Weg in die Einstellungen (frontend/src/pages/SettingsPage.tsx:307-343), internal/service/statement_export.go:212-236 setzt sie in das PDF | – |
| Aufstellungsfrist je Größenklasse überwacht und als Termin angezeigt | ✅ | internal/accounting/groessenklasse.go:287-323 (`StatementDeadlines`) rechnet die Frist aus Stichtag und Klasse, :332-339 nach den ersten drei oder sechs Monaten des folgenden Geschäftsjahres statt nach Stichtag plus N Monaten; internal/service/statement_service.go:367-388 hält fest, was das Geschäftsjahr als aufgestellt vermerkt, frontend/src/pages/DeadlinesPage.tsx:554-567 stellt den Termin in die Fristenliste und frontend/src/components/StatementView.tsx:739-779 an den Abschluss | – |

**Stand.** Bilanz und Gewinn- und Verlustrechnung entstehen im Backend aus einer festen Gliederung, auf die alle 206 HGB-Positionen aus internal/accounting/skr04_2026.json abgebildet sind; Gliederungstiefe, Kopfangaben und Aufstellungsfrist folgen der Größenklasse, die Ausgabe erfolgt als PDF und als CSV. Die Ansicht rechnet nichts mehr nach. Offen bleibt allein das Umsatzkostenverfahren, und dessen Auslassung ist gewollt.

### JAB-02 Größenklassen `MUSS`

**Norm:** §§ 267, 267a HGB

**Bedeutung:** Die Größenklasse steuert Gliederungstiefe, Anhangumfang, Prüfungspflicht und Offenlegungsumfang. Die monetären Schwellenwerte wurden 2024 um 25 Prozent angehoben, die Arbeitnehmerzahlen blieben unverändert. Eine Klasse wechselt erst, wenn mindestens zwei der drei Merkmale an zwei aufeinanderfolgenden Abschlussstichtagen über- oder unterschritten werden.

| Klasse | Bilanzsumme | Umsatzerlöse | Arbeitnehmer |
|---|---|---|---|
| Kleinst (§ 267a HGB) | bis 450.000 Euro | bis 900.000 Euro | bis 10 |
| Klein (§ 267 Abs. 1 HGB) | bis 7.500.000 Euro | bis 15.000.000 Euro | bis 50 |
| Mittelgroß (§ 267 Abs. 2 HGB) | bis 25.000.000 Euro | bis 50.000.000 Euro | bis 250 |
| Groß (§ 267 Abs. 3 HGB) | darüber | darüber | darüber |

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Drei Merkmale je Stichtag berechnet, Größenklasse daraus abgeleitet, Zweijahresregel des § 267 Abs. 4 HGB beachtet | ✅ | internal/accounting/groessenklasse.go:89-118 (`AssessSize`) misst Bilanzsumme, Umsatzerlöse und Arbeitnehmerzahl je Stichtag und benennt die zwei Merkmale, die die Klasse bestimmen; :158-230 (`ClassifySize`, `effectiveClass`) löst die Zweijahresregel auf, einschließlich der Neugründung nach § 267 Abs. 4 Satz 2 HGB, und internal/service/statement_service.go:228-312 beurteilt so viele Vorjahre, wie die Regel braucht. Die Bilanzsumme des § 267 Abs. 4a HGB kommt aus der Gliederung (internal/accounting/statement.go:317-342), die Arbeitnehmerzahl steht am Geschäftsjahr (internal/domain/fiscalyear.go:100-109, internal/service/closing_service.go:1357-1387, frontend/src/pages/ClosingPage.tsx:585-605) | – |
| Schwellenwerte parametrisierbar und zeitabhängig versioniert | ✅ | internal/accounting/groessenklasse.go:39-59 führt zwei datierte Sätze — die Werte vor und nach Art. 79 Abs. 1 EGHGB —, :63-74 (`SizeThresholdsFor`) wählt sie nach dem Beginn des Geschäftsjahres und nicht nach dem Stichtag; der Satz, an dem gemessen wurde, steht in der Beurteilung (internal/domain/sizeclass.go:99-110) und neben den Merkmalen in der Ansicht (frontend/src/components/StatementView.tsx:593-658) | – |
| Klassenwechsel angekündigt, sobald er sich am ersten Stichtag abzeichnet, mit Hinweis auf die Folgen | 🟡 | internal/accounting/groessenklasse.go:192-209 benennt den abweichenden Stichtag im Klartext und sagt, dass die Rechtsfolgen erst am zweiten übereinstimmenden eintreten; frontend/src/components/StatementView.tsx:581-591 und :661-684 zeigen den Satz und die beurteilten Stichtage. Eine Ankündigung, die den Wechsel als bevorstehend kennzeichnet, fehlt: die Tabelle „Folgen der Größenklasse" (:686-737) nennt allein die Folgen der geltenden Klasse | 2 |
| Kapitalmarktorientierte Gesellschaften nach § 264d HGB gelten als groß, Merkmal im Mandanten setzbar | ⛔ | Kapitalmarktorientierung liegt außerhalb des Geltungsbereichs; Buchfink verweist ab mittelgroß ohnehin an den Steuerberater | – |

**Stand.** Die Größenklasse wird aus den drei Merkmalen des § 267 Abs. 1 HGB berechnet, mit datierten Schwellen, der Zweijahresregel und einer Begründung, die jeden beurteilten Stichtag nennt; aus ihr folgen Gliederungstiefe, Anhang-, Lagebericht-, Prüfungs- und Offenlegungspflicht sowie beide Fristen (internal/accounting/groessenklasse.go:240-282). Offen bleibt die Ankündigung des Wechsels, bevor er eintritt. Welle 2.

### JAB-03 Anhang und Lagebericht `MUSS`

**Norm:** §§ 284 bis 289 HGB, §§ 274a, 288 HGB

**Bedeutung:** Der Anhang erläutert Bilanzierungs- und Bewertungsmethoden und enthält den Anlagenspiegel sowie die Pflichtangaben des § 285 HGB. Kleinstkapitalgesellschaften können den Anhang weglassen, wenn sie bestimmte Angaben unter der Bilanz machen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Aus den Buchungsdaten mindestens Anlagenspiegel, Restlaufzeitengliederung, Haftungsverhältnisse, sonstige finanzielle Verpflichtungen, Beteiligungsliste, latente Steuern, Ergebnisverwendungsvorschlag | 🟡 | Aus den Daten kommen drei: internal/service/asset_service.go (`Anlagenspiegel`) den Anlagenspiegel aus der Kartei, internal/service/statement_service.go:399-450 (`maturities`) die Restlaufzeitengliederung nach § 268 Abs. 4 und 5 HGB aus den offenen Posten zum Abschlussstichtag (frontend/src/components/StatementView.tsx:524-564, internal/service/statement_export.go:180-198) und seit Welle 5 den Rückstellungsspiegel nach § 285 Nr. 12 HGB (internal/accounting/provision.go:172-217, internal/service/statement_service.go:174-201). Der Ergebnisverwendungsvorschlag steht als beschlossene Verwendung daneben (internal/domain/appropriation.go:22-58). Haftungsverhältnisse, sonstige finanzielle Verpflichtungen und Beteiligungsliste sind Freitextabschnitte und keine Auswertung (internal/domain/notes_text.go:39-63) — sie stehen in keiner Buchung; latente Steuern entfallen für die Zielgruppe (§ 274a Nr. 4 HGB, siehe BEW-11) | 5 |
| Freitextangaben erfassbar und über den Jahreswechsel als Vorlage fortgeschrieben | ✅ | internal/domain/notes_text.go:17-99 führt sieben Abschnitte des Anhangs — Methoden, Organbezüge, Nachtragsbericht, finanzielle Verpflichtungen, Haftungsverhältnisse, Beteiligungen, Ergebnisverwendungsvorschlag — je mit Vorschrift und Erklärtext, internal/service/appropriation_service.go:445-481 liest und schreibt sie, :489-523 (`CopyNotesInto`) übernimmt sie beim Anlegen des neuen Geschäftsjahres als Vorlage, ohne eine bereits geschriebene Fassung zu überschreiben (internal/service/closing_service.go:196-206). Geschrieben wird in frontend/src/pages/ClosingModulesPage.tsx:3100-3147, gezeigt in frontend/src/pages/ReportsPage.tsx:78-116 | – |
| Anhangumfang folgt automatisch der Größenklasse | 🟡 | internal/accounting/groessenklasse.go:250-282 entscheidet je Klasse über Anhang und Lagebericht und nennt die Norm dazu (§ 264 Abs. 1 Sätze 1 und 5, § 289 HGB), frontend/src/components/StatementView.tsx:686-737 zeigt beides. Seit Welle 5 entsteht ein Anhang (internal/service/statement_service.go:174-201, frontend/src/pages/ReportsPage.tsx:66-245), sein Umfang folgt der Größenklasse aber nicht: `notesFor` stellt dieselben Abschnitte zusammen, gleich welche Klasse gilt, und die Befreiungen der §§ 274a, 288 HGB blenden nichts aus | 5 |
| Für Kleinstkapitalgesellschaften Angaben unter der Bilanz statt eines Anhangs | 🟡 | internal/accounting/groessenklasse.go:251-259 nimmt der Kleinstgesellschaft den Anhang unter der Bedingung des § 264 Abs. 1 Satz 5 HGB, und die Angaben unter der Bilanz stehen als eigener Abschnitt in Ansicht und PDF (frontend/src/pages/ReportsPage.tsx:279, frontend/src/components/StatementView.tsx:122-132, internal/service/statement_export.go:180-198). Sie enthalten bislang nur die Restlaufzeiten; die Haftungsverhältnisse nach § 268 Abs. 7 HGB lassen sich seit Welle 5 als Anhangtext erfassen (internal/domain/notes_text.go:53-55), stehen damit aber im Anhang und nicht unter der Bilanz, und die Vorschüsse an Organmitglieder fehlen | 5 |

**Stand.** Das Muster wiederholt sich jetzt dreimal: der Anlagenspiegel als Auswertung über die Kartei, die Restlaufzeiten als Auswertung über die offenen Posten (internal/domain/payment.go:77), der Rückstellungsspiegel als Auswertung über die Rückstellungskartei — keine davon eine zweite Buchung. Mit Welle 5 gibt es den Anhang als Sache: sieben Freitextabschnitte mit ihrer Vorschrift, ins Folgejahr übernommen, dazu Rückstellungsspiegel und Überleitung zur Steuerbilanz. Offen bleibt, dass sein Umfang der Größenklasse nicht folgt und dass Haftungsverhältnisse und Beteiligungen Freitext bleiben. Welle 5.

### JAB-04 Feststellung und Unterzeichnung `MUSS`

**Norm:** § 245 HGB, § 42a GmbHG, §§ 172, 173 AktG

**Bedeutung:** Der Jahresabschluss ist vom Kaufmann unter Angabe des Datums zu unterzeichnen. Bei der GmbH stellen die Gesellschafter fest, bei der AG in der Regel Vorstand und Aufsichtsrat. Erst der festgestellte Abschluss ist offenlegungsfähig.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Status Entwurf, aufgestellt, festgestellt, offengelegt, jeder Wechsel mit Datum, Person und Beschlussbezug protokolliert | ✅ | internal/domain/fiscalyear.go:18-72 führt die vier Stände (der Entwurf heißt "Offen"), :92-98 Datum je Schritt und den Beschlussbezug, :184-195 lässt keinen Stand ohne sein Datum zu; internal/service/closing_service.go:360-433 (`SetFiscalYearStatus`) schaltet nur einen Schritt weiter und protokolliert jeden Wechsel mit Datum und Beschluss. Die Person entfällt im Einzelplatzbetrieb | – |
| Ab dem Status festgestellt keine Änderungen an den zugrunde liegenden Buchungen | ✅ | internal/service/journal_service.go:489-505 (`ensureYearNotAdopted`) weist jede Buchung in ein festgestelltes Jahr ab, geprüft auf dem einzigen Schreibweg (:115-117) und in der Vorprüfung mehrteiliger Vorgänge (:146-170); die Feststellung setzt ihrerseits die Jahres-Festschreibung voraus (internal/service/closing_service.go:394-405) | – |
| Dokumentierte Rücksetzung des Status bei Änderung | ✅ | internal/service/closing_service.go:443-474 (`ReopenFiscalYear`) setzt auf "Aufgestellt" zurück, verlangt einen Grund und schreibt ihn samt Ausgangsstand ins Protokoll; die Festschreibungen bleiben unberührt | – |
| Feststellungsbeschluss als Dokument mit dem Abschluss verknüpfbar | ❌ | internal/receiptstore/store.go speichert Eingangsbelege, Ausgangsrechnungen und Anlagendokumente; seit Welle 5 lässt sich auch der Gesellschafterbeschluss über die Ergebnisverwendung ablegen und versiegeln (internal/service/appropriation_service.go:388-416, :409-416 versiegelt ihn mit der Buchung), das ist aber ein anderer Beschluss. Für den Feststellungsbeschluss gibt es keinen Platz am Geschäftsjahr — internal/domain/fiscalyear.go:92-98 hält nur Datum und Beschlussbezug als Text | 1 |
| Unterzeichneter Abschluss als unveränderliches Dokument archiviert | ❌ | Es entsteht kein Abschlussdokument | 1 |

**Stand.** Der Abschlussstand ist als Kette aus vier Ständen geführt, mit Datum, Beschlussbezug und einer Rücksetzung, die nur mit Grund geht; ab der Feststellung nimmt das Jahr keine Buchung mehr an. Was fehlt, sind die Papiere dazu: der Feststellungsbeschluss als Dokument und der unterzeichnete Abschluss als archiviertes Dokument. Welle 1.

### JAB-05 E-Bilanz `MUSS` / `TERMIN`

**Norm:** § 5b EStG, § 60 EStDV, BMF-Schreiben zu den Taxonomien

**Bedeutung:** Bilanz und Gewinn- und Verlustrechnung sind nach amtlich vorgeschriebenem Datensatz elektronisch zu übermitteln. Der Umfang wächst: unverdichtete Kontennachweise mit Kontensalden für Wirtschaftsjahre ab 2025, Anlagenspiegel und Anlagenverzeichnis für Wirtschaftsjahre ab 2028. Für das Wirtschaftsjahr 2026 gilt die Taxonomie 6.9 (BMF-Schreiben vom 10.06.2025). Die Taxonomie 6.10 wurde mit BMF-Schreiben vom 08.06.2026 veröffentlicht (Taxonomien vom 01.04.2026) und ist für Wirtschaftsjahre verpflichtend, die nach dem 31.12.2026 beginnen; ihre Verwendung bereits für das Wirtschaftsjahr 2026 wird nicht beanstandet, die Übermittlung in Echtfällen ist ab Mai 2027 vorgesehen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| XBRL-Datensatz nach der gültigen Taxonomie, übermittelt oder zur Übermittlung durch Dritte bereitgestellt | 🟡 | internal/ebilanz/ebilanz.go:106-168 erzeugt die Instanz mit `encoding/xml` aus derselben Gliederung wie die Bilanz; :253-289 schreibt jede Position von Bilanz und GuV mit Vorjahreskontext und die Bilanzsumme dazu, Aufwendungen positiv. Geprüft ist kein Elementname: internal/ebilanz/taxonomy_6.9.json ist durchgehend mit `verified: false` markiert, und der Vorbehalt steht in der Datei selbst (:170-177). Buchfink bindet ERiC nicht ein, die Datei wird zur Übermittlung über Mein ELSTER oder den Steuerberater bereitgestellt | 2 |
| Taxonomieversion als austauschbare Ressource hinterlegt, Versionswechsel ohne Codeänderung | ✅ | internal/ebilanz/taxonomy.go:10-11 bindet internal/ebilanz/taxonomy_6.9.json als Ressource ein, :48-66 liest Version, Datum, Namensräume und Elemente daraus; internal/ebilanz/ebilanz.go:127-129 schreibt die Namensräume der Ressource in die Instanz und :172-177 ihre Version in den Vorbehalt — der Wechsel tauscht die Datei, nicht den Übersetzer | – |
| Jedes Konto einer Taxonomieposition zugeordnet, nicht zugeordnete Konten mit Saldo blockieren und werden benannt | ✅ | internal/ebilanz/mapping.go:52-107 führt jedes Konto mit Saldo über seine Gliederungsposition auf ein Taxonomie-Element, :111-128 (`BlockingError`) nennt die ungeklärten mit Nummer, Name und Grund, internal/ebilanz/ebilanz.go:118-124 bricht ab, bevor eine Datei entsteht; internal/ebilanz/taxonomy.go:68-80 kennt keinen Auffangwert mehr, `bs.other` entfällt. frontend/src/pages/EBilanzPage.tsx:116-170 zeigt die Liste vor dem Export | – |
| Unverdichtete Kontennachweise mit Kontensalden erzeugt und mitübermittelt | 🟡 | internal/ebilanz/ebilanz.go:291-309 schreibt je Konto mit Saldo Nummer, Bezeichnung, Gliederungsposition, Taxonomie-Element und Saldo unverdichtet in die Instanz (:150); die Hüllelemente `accountAuditProof` und die Felder darunter stehen in keiner Taxonomie-Ressource, sie sind wie bisher frei gebildet | 2 |
| Ab Wirtschaftsjahr 2028 Anlagenspiegel und Anlagenverzeichnis mitübermittelt | 🟡 | internal/ebilanz/ebilanz.go:320-374 schreibt den Anlagenspiegel je Konto, je Klasse des § 266 Abs. 2 A HGB und in Summe, mit Anschaffungskosten, Abschreibungen und Buchwerten, und hängt an jede Zeile die Taxonomieposition ihres Kontos; das Anlagenverzeichnis geht nicht mit | 2 |
| Auffangpositionen nur ersatzweise verwendet, Verwendung erscheint in einem Bericht | ✅ | internal/accounting/statement.go:667-690 zählt jede als Auffang gekennzeichnete Position der Gliederung (:88, :100, :103, :120, :132, :142, :161, :171) mit Kontenzahl und Betrag aus, internal/ebilanz/mapping.go:44-45, :66-68 nimmt die Zählung in den Zuordnungsbericht, frontend/src/pages/EBilanzPage.tsx:137-141, :221-250 zeigt sie vor dem Export | – |
| Übermittlungsprotokoll revisionssicher gespeichert | ❌ | internal/service/ebilanz_service.go:86-92 protokolliert weiterhin nur, dass eine Datei erzeugt wurde. Das Muster steht seit Welle 3 an der Voranmeldung (internal/domain/vatreturn.go:107-146, :205-222: Entität mit Übermittlungsdatum, Transferticket und Status), für die E-Bilanz gibt es kein solches Objekt | 3 |

**Stand.** Die Reihenfolge hat sich bewährt: seit die Gliederung aus JAB-01 steht, zeigt das Mapping auf Positionen statt auf Konten. Die Instanz enthält Bilanz und Gewinn- und Verlustrechnung mit Vorjahreskontext, den unverdichteten Kontennachweis und den Anlagenspiegel, und ein Konto ohne Zuordnung blockiert den Export namentlich, statt still auf einer Sammelposition zu verschwinden. Was bleibt, ist der Abgleich der Elementnamen gegen die amtliche Taxonomie 6.9 und das Protokoll der Übermittlung. Welle 2.

### JAB-06 Maßgeblichkeit und Überleitung `MUSS`

**Norm:** § 5 Abs. 1 EStG, § 60 Abs. 2 EStDV

**Bedeutung:** Die Steuerbilanz leitet sich aus der Handelsbilanz ab. Weichen Ansätze ab, ist entweder eine eigene Steuerbilanz einzureichen oder die Handelsbilanz durch Zusätze und Anmerkungen anzupassen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Handels- und Steuerbilanz aus demselben Buchungsstamm, Abweichungen als eigene Wertansätze je Position, keine Parallelbuchhaltung | ✅ | internal/service/tax_register_service.go:264-352 (`Reconcile`) stellt je Position den handelsrechtlichen und den steuerlichen Wertansatz nebeneinander — Anlagevermögen aus § 7g Abs. 5 EStG, Rückstellungen aus der abweichenden Abzinsung, Drohverlustrückstellungen aus dem Ansatzverbot des § 5 Abs. 4a EStG —, alles aus demselben Buchungsstamm und ohne eine einzige Buchung: die steuerlichen Werte werden gerechnet, nicht geführt (internal/domain/statement.go:287-310, internal/service/tax_register_service.go:354-392). Mehr Abweichungen kennt Buchfink nicht; entstünden sie, blieben sie außen vor | – |
| Überleitungsrechnung von der Handels- zur Steuerbilanz mit Rechtsgrundlage je Abweichung | ✅ | Jede Zeile hat ihre Vorschrift und ihre Erläuterung (internal/service/tax_register_service.go:292-350: § 7g Abs. 5 und § 5 Abs. 1 Satz 2 EStG, § 253 Abs. 2 HGB gegen § 6 Abs. 1 Nr. 3a Buchst. e EStG, § 249 Abs. 1 HGB gegen § 5 Abs. 4a EStG), dazu die Wirkung auf das Eigenkapital; sie steht im Anhang (internal/service/statement_service.go:195-198, frontend/src/pages/ReportsPage.tsx:177-242) und unter den Abschlussbausteinen (frontend/src/pages/ClosingModulesPage.tsx:3322-3385) | – |
| Wahlweise eine vollständige Steuerbilanz ausgebbar | ⛔ | Einheitsbilanz: eine zweite Bilanz würde jede Erfassungsmaske verdoppeln | – |
| Überleitung in der E-Bilanz-Struktur übermittelbar | 🟡 | internal/ebilanz/ebilanz.go:392-429 (`reconciliationFacts`) schreibt je Position handelsrechtlichen Wert, steuerlichen Wert, Differenz und Rechtsgrundlage als eigenen Block in die Instanz, internal/service/ebilanz_service.go:81-87 hängt ihn ein, sobald die Überleitung Zeilen hat. Die Elementnamen des Überleitungsmoduls sind nach der Systematik der Taxonomie gebildet und stehen in internal/ebilanz/taxonomy_6.9.json:469-478 mit `verified: false`; vor der Übermittlung sind sie gegen die amtliche Fassung abzugleichen (JAB-05) | 2 |

**Stand.** Mit Welle 5 gibt es die Überleitung, und sie ist bewusst eine Rechnung neben der Bilanz und keine zweite Buchführung: die beiden Stellen, an denen Handels- und Steuerbilanz in Buchfink zwingend auseinanderfallen — die Sonderabschreibung nach § 7g Abs. 5 EStG und die Abzinsung der Rückstellungen mit 5,5 % samt Ansatzverbot für Drohverluste —, stehen mit Wertansatz, Differenz und Rechtsgrundlage im Anhang und gehen als eigener Block in die E-Bilanz. Einzelne Unterschiede laufen daneben weiterhin über getrennte Konten, etwa die nicht abziehbaren Betriebsausgaben (internal/service/posting_service.go:430). Eine vollständige Steuerbilanz bleibt außerhalb des Funktionsumfangs, der Abgleich der Elementnamen offen. Welle 2.

### JAB-07 Offenlegung `MUSS`

**Norm:** §§ 325 bis 329, 335 HGB

**Bedeutung:** Seit dem Geschäftsjahr 2022 gehen die Unterlagen an das Unternehmensregister, nicht mehr an den Bundesanzeiger. Die Frist beträgt zwölf Monate nach dem Abschlussstichtag, bei kapitalmarktorientierten Gesellschaften vier Monate. Kleine Gesellschaften legen nur Bilanz und Anhang ohne GuV-bezogene Angaben offen, Kleinstgesellschaften können statt der Offenlegung die dauerhafte Hinterlegung der Bilanz wählen. Bei Verstoß setzt das Bundesamt für Justiz ein Ordnungsgeld ab 2.500 Euro fest.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Offenzulegender Umfang automatisch aus der Größenklasse | 🟡 | internal/accounting/groessenklasse.go:250-282 setzt je Klasse den Umfang und die Norm dazu (§ 325 Abs. 1, § 326 Abs. 1 und Abs. 2 HGB), :309-321 nennt ihn im Termin, frontend/src/components/StatementView.tsx:730-734 zeigt ihn. Ein auf diesen Umfang beschränkter Datensatz entsteht nicht: internal/service/statement_export.go:168-198 gibt Bilanz, Gewinn- und Verlustrechnung und die Angaben unter der Bilanz immer vollständig aus | 2 |
| Datensatz im Format der Einreichungsplattform erzeugt, übermittelbar oder exportierbar | ❌ | Kein Export für das Unternehmensregister | 2 |
| Zwölfmonatsfrist je Geschäftsjahr überwacht, mit Vorwarnung | ✅ | internal/accounting/groessenklasse.go:309-322 setzt den Termin für jedes Geschäftsjahr auf zwölf Monate nach dem Abschlussstichtag (§ 325 Abs. 1a Satz 1 HGB), internal/service/statement_service.go:367-388 trägt eine bereits erfolgte Offenlegung aus dem Geschäftsjahr ein, frontend/src/pages/DeadlinesPage.tsx:554-567 stellt ihn in die Fristenliste, die überfällige und in den nächsten 30 Tagen fällige Termine heraushebt (:656, :768-845) | – |
| Hinterlegungsvariante für Kleinstgesellschaften wählbar | ❌ | internal/accounting/gruendung.go:300-302 nennt § 326 Abs. 2 HGB im Beschreibungstext, nicht als Wahl | 2 |
| Einreichungsnachweis mit dem Abschluss archiviert | ❌ | Nicht vorhanden | 2 |

**Stand.** Die Frist ist aus den Gründungspflichten in die jährliche Terminliste gewandert, und der offenzulegende Umfang folgt der Größenklasse als benannte Rechtsfolge. Der Datensatz für das Unternehmensregister, die Wahl der Hinterlegung nach § 326 Abs. 2 HGB und der Einreichungsnachweis fehlen. Welle 2.

### JAB-08 Prüfungsfähigkeit `MUSS*`

**Norm:** §§ 316 bis 324a HGB

**Bedeutung:** Mittelgroße und große Kapitalgesellschaften sind prüfungspflichtig. Ohne Prüfung kann der Jahresabschluss nicht festgestellt werden. Die Software muss dem Prüfer arbeitsfähige Daten liefern, nicht nur Berichte.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Prüferzugang mit ausschließlich lesenden Rechten, zeitlich befristbar | ✅ | Statt Benutzerkonten, die es im Einzelplatzbetrieb nicht gibt, schaltet der Prüfermodus die ganze Anwendung schreibgeschützt: internal/wailsbridge/readonly.go:227-260 verlangt Ablaufdatum und Grund und protokolliert beides, :210-220 weist jede schreibende Bridge-Methode ab, :37-189 führt die zulässigen Methoden als abschließende Liste — ein Test ordnet jede exportierte Bridge-Methode einer der beiden Seiten zu (internal/wailsbridge/readonly_test.go) | – |
| Alle Bewegungsdaten eines Geschäftsjahres maschinell auswertbar exportierbar, einschließlich Journal, Konten, Salden, Belegverweisen und Änderungsprotokoll | ✅ | internal/service/export_service.go:697-720 stellt achtzehn Tabellen zusammen (internal/service/export_tables.go:21-40): Journal mit Hashes, Konten mit Gliederungs- und Taxonomieposition, Salden, Kontakte, offene Posten, Anlagen und Bewegungen, Belege, Dokumente, Voranmeldungen, Festschreibungen, Prüfläufe, Zahlungszuordnungen, Bewirtungen, Änderungsprotokoll, Steuerschlüssel, Schlüsselverzeichnis und seit Welle 7 den Prüfpfad je Beleg (internal/service/export_tables.go:1030-1102); das Prüferpaket legt Belegdateien, Integritätsnachweis und Verfahrensdokumentation dazu (:155-215) | – |
| Summen- und Saldenlisten zu jedem beliebigen Stichtag reproduzierbar, auch rückwirkend | ✅ | internal/service/accounting_service.go:326-330 und internal/repository/journal_gorm.go:274-293 summieren bis zum Stichtag; dieselbe Grenze gilt für das Kontoblatt über Jahresgrenzen hinweg (internal/service/accounting_service.go:196-200, internal/repository/journal_gorm.go:167) und für die offenen Posten (internal/service/payment_service.go:176-189). Die Listen entstehen aus den Buchungen und nicht aus einem fortgeschriebenen Saldo, sind also beliebig oft rückwirkend herstellbar | – |
| Saldenbestätigungslauf für Debitoren und Kreditoren | ❌ | internal/service/payment_service.go:109-127 liefert die operative Liste, :185-198 dieselbe zum Stichtag; Welle 7 hat den Weg vom offenen Posten zum Anschreiben für das Mahnwesen gebaut (internal/service/dunning_service.go:146-318, :471-551), der Saldenbestätigungslauf mit Anschreiben je Geschäftspartner und festgehaltener Rückmeldung fehlt weiterhin | Politur |
| Prüfungsvermerk und Prüfungsbericht als Dokument mit dem Abschluss verknüpfbar | ❌ | Kein Abschlussobjekt, keine Verknüpfung (siehe JAB-04) | 1 |

**Stand.** Der Prüfer bekommt seit Welle 4, was er zum Arbeiten braucht: einen befristeten Nur-Lese-Zugang, die Bewegungsdaten des Jahres als Datenüberlassung und jede Auswertung auf den Stichtag, nach dem er fragt — die Stichtagsvariante der Kontenumsätze deckt SuSa, Kontoblatt und OP-Liste zugleich ab. Mit Welle 7 kommt der Prüfpfad je Beleg als achtzehnte Tabelle dazu. Offen bleiben die beiden Punkte, die von der Abschlussprüfung selbst abhängen: der Saldenbestätigungslauf, für den der Weg vom offenen Posten zum Anschreiben inzwischen im Mahnwesen steht, und der Prüfungsvermerk am Abschlussobjekt aus Welle 1. Politur.

### JAB-09 Jahreswechsel und Saldenvortrag `MUSS`

**Norm:** § 252 Abs. 1 Nr. 1 HGB, § 242 Abs. 1 HGB

**Bedeutung:** Die Eröffnungsbilanz des Geschäftsjahres muss mit der Schlussbilanz des Vorjahres übereinstimmen. Der Vortrag darf keine Werte verändern.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Saldenvortrag als eigener Buchungsvorgang mit eigenem Belegverweis, im Journal sichtbar | ✅ | internal/service/closing_service.go:1107-1175 (`buildEntry`) bucht die Bestandskonten gegen 9000, die Personenkonten gegen 9008 und 9009 (internal/domain/skr04_accounts.go:27-29), als Buchung der Herkunft Eröffnung mit dem Belegverweis "SV JJJJ" (:1176); geschrieben wird über denselben `Post`-Weg wie jede andere Buchung und steht damit im Journal | – |
| Vortrag wiederholbar ohne doppelte Werte, ein erneuter Lauf ersetzt den vorherigen nachvollziehbar | ✅ | internal/service/closing_service.go:995-1100 nimmt die bestehenden Vortragsbuchungen per Generalumkehr auf das Vortragsdatum zurück (internal/service/journal_service.go:203-215, `ReverseOn`) und bucht neu; ein Lauf ohne Änderung wird abgewiesen (:1019-1022), ein nicht mehr zurücknehmbarer Altvortrag ebenfalls, statt seine Werte zu verdoppeln (:1026-1034) | – |
| Ändert sich das Vorjahr nach dem Vortrag, wird die Differenz gemeldet und ein korrigierender Vortrag angeboten | ✅ | internal/service/closing_service.go:708-738 stellt je Konto Schlusssaldo, vorgetragenen Wert und Differenz gegenüber und setzt daraus `NeedsCorrection` (:764); die Vorschau zeigt die Zeilen, der Korrekturvortrag steht als Aktion daneben (frontend/src/pages/ClosingPage.tsx:516-670) | – |
| Personenkonten mit offenen Posten vorgetragen, nicht nur mit Saldo | ✅ | internal/service/closing_service.go:897-942 (`openItemsAt`) sammelt die zum Bilanzstichtag offenen Posten je Geschäftspartner, internal/repository/journal_gorm.go:134-141 liefert dafür die Stichtagssicht einschließlich später stornierter Rechnungen; :1133-1148 schreibt je Posten eine eigene Zeile mit Belegverweis und Kontakt. Der Vortrag selbst zählt nicht noch einmal als offener Posten (internal/service/payment_service.go:141-149) | – |
| Vorjahresergebnis auf das Ergebnisvortragskonto gebucht, gesteuert durch den Ergebnisverwendungsbeschluss | ✅ | internal/service/closing_service.go:654-663 bringt das Jahresergebnis mit dem Vortrag zunächst unverwendet auf 2970 oder 2978 (internal/domain/skr04_accounts.go:69-77) — nachträglich verteilen dürfte es der Vortrag nicht, § 252 Abs. 1 Nr. 1 HGB verbietet die Änderung der Eröffnungsbilanz. Verteilt wird es durch den Beschluss: internal/domain/appropriation.go:22-86 hält Datum, Wortlaut, Belegverweis und die vier Beträge, internal/service/appropriation_service.go:93-223 rechnet Rücklagen, Ausschüttung samt Kapitalertragsteuer und Vortrag auf neue Rechnung und erzwingt die Pflichtrücklage der UG nach § 5a Abs. 3 GmbHG (:232-301, :332-338), :323-424 bucht ihn im Folgejahr mit dem Gesellschafterbeschluss als Beleg | – |

**Stand.** Der Jahreslauf ist nicht mehr blockiert: der Vortrag bringt Bestandskonten, offene Posten und Jahresergebnis ins Folgejahr, ist wiederholbar und wird abgelehnt, wenn er nicht aufgeht. Seit Welle 5 ist die Ergebnisverwendung beschlossen und nicht mehr unterstellt: der Vortrag legt das Ergebnis unverwendet ab, der Beschluss verteilt es im Folgejahr an seinem eigenen Datum. Ein Rand bleibt: `AccountLedger.OpeningBalance` steht im Kontoblatt weiterhin hart auf 0 (internal/service/accounting_service.go:230), der Vortrag erscheint dort als erste Buchung des Jahres statt als Anfangsbestand. Welle 1.

---
