# G. Bewertung, Anlagen, Fremdwährung

[Anforderungskatalog](README.md) · [Legende](README.md#legende) · [Fachkonzepte](../fachkonzepte/README.md)

### BEW-01 Bewertungsgrundsätze `MUSS`

**Norm:** § 252 HGB

**Bedeutung:** Die sechs Grundsätze des § 252 Abs. 1 HGB steuern jede Bewertung: Bilanzidentität, Fortführung der Unternehmenstätigkeit, Einzelbewertung, Vorsicht mit Realisations- und Imparitätsprinzip, Periodenabgrenzung, Bewertungsstetigkeit. Abweichungen sind nur in begründeten Ausnahmefällen zulässig und im Anhang anzugeben.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Eröffnungsbilanzwerte entsprechen zwingend den Schlussbilanzwerten des Vorjahres, Abweichung technisch ausgeschlossen | ✅ | internal/service/closing_service.go:603-775 stellt die Vortragsvorschau je Konto aus den Schlusssalden des Vorjahres zusammen und rechnet mit :563-568 die Probe (Summe aller Vortragswerte gleich null); :1002-1015 lehnt einen Vortrag ab, dessen Werte nicht aufgehen, statt die Differenz ins neue Jahr zu übernehmen. Die Probe steht als Kennzahl in frontend/src/pages/ClosingPage.tsx:586-595 | – |
| Bewertungsmethoden je Bilanzposition hinterlegt, fortgeschrieben, Änderung begründet mit Anhangshinweis | 🟡 | internal/domain/asset.go:414 führt eine Methode je Anlagegut, nicht je Bilanzposition; eine Methodenänderung ist ein gewöhnliches Feldupdate ohne Begründungspflicht. Seit Welle 5 lässt sich die Angabe nach § 284 Abs. 2 Nr. 1 HGB wenigstens als Anhangtext schreiben und wird ins Folgejahr übernommen (internal/domain/notes_text.go:40-43, internal/service/appropriation_service.go:489-523) — geschrieben von Hand, nicht aus den Daten abgeleitet | 2 |
| Wertansätze je Wirtschaftsgut, Sammelbewertungen gekennzeichnet | 🟡 | internal/domain/asset.go:294-372 bewertet im Anlagevermögen streng einzeln, jede Wertänderung ist eine Bewegung, der Sammelposten ist gekennzeichnet; Sammelbewertungen nach §§ 240 Abs. 3, 4 und 256 HGB betreffen Vorräte und sind außerhalb des Funktionsumfangs | 5 |
| Bericht über alle im Geschäftsjahr geänderten Bewertungsmethoden | ❌ | Kein solcher Bericht in internal/ oder frontend/src | 2 |

**Stand.** Die Bilanzidentität ist seit dem Saldenvortrag nicht nur darstellbar, sondern erzwungen: ein Vortrag, der nicht aufgeht, wird abgelehnt. Die Einzelbewertung im Anlagevermögen ist nachweisbar. Was bleibt, ist die Stetigkeit: ohne geführte Bewertungsmethoden je Bilanzposition gibt es weder Begründungspflicht noch Änderungsbericht. Welle 2.

### BEW-02 Anschaffungs- und Herstellungskosten `MUSS`

**Norm:** § 255 HGB, § 6 Abs. 1 Nr. 1a EStG

**Bedeutung:** § 255 HGB legt fest, welche Bestandteile in die Anschaffungs- und Herstellungskosten einfließen und wo Wahlrechte bestehen. Steuerlich kommt die Regel zu anschaffungsnahen Herstellungskosten hinzu: Instandsetzungs- und Modernisierungsaufwand innerhalb von drei Jahren nach Anschaffung eines Gebäudes gilt als Herstellungskosten, wenn er 15 Prozent der Gebäudeanschaffungskosten ohne Umsatzsteuer übersteigt.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Anschaffungsnebenkosten und nachträgliche Anschaffungskosten nachträglich zuordenbar, ändern die Abschreibungsbasis ab dem Zuordnungszeitpunkt | ✅ | internal/service/asset_service.go:729, internal/accounting/afa.go:317-327; die Basisänderung wirkt nach R 7.4 Abs. 9 EStR zu Beginn des betroffenen Jahres und rechnet die Vergangenheit nicht neu | – |
| Anschaffungspreisminderungen reduzieren die Anschaffungskosten und sind je Wirtschaftsgut dokumentiert | ✅ | internal/domain/asset.go:206, internal/service/asset_service.go:792-799; eigene Bewegungsart mit negativem Betrag und Notiz | – |
| Pflicht- und Wahlbestandteile der Herstellungskosten getrennt erfassbar, Wahlrechtsausübung gespeichert | ❌ | internal/domain/asset.go:404 kennt nur einen Gesamtbetrag; keine Komponentenerfassung, kein Wahlrechtsfeld | 5 |
| Dreijahreszeitraum und 15-Prozent-Grenze für Gebäude überwacht | ✅ | internal/service/asset_welle5c.go:424-491 (`CheckNearAcquisitionCost`) summiert den Instandsetzungs- und Modernisierungsaufwand der drei Jahre nach der Anschaffung eines Gebäudes und hält ihn gegen 15 % der Anschaffungskosten netto; Grenze und Zeitraum stehen datiert in internal/accounting/tax_params.go:79-86, :170-172, Erweiterungen und übliche Erhaltungsarbeiten bleiben nach § 6 Abs. 1 Nr. 1a Satz 2 EStG außen vor (:456-464). Wird der Rahmen gerissen, aktiviert :687-824 (`CapitalizeNearAcquisitionCost`) den gesammelten Aufwand mit Pflichtbegründung als nachträgliche Herstellungskosten und schreibt die Bemessungsgrundlage fort. Gebucht wird die Umbuchung vom Aufwandskonto auf das Gebäude und nicht die Generalumkehr jeder einzelnen Aufwandsbuchung: die Aufwandsbuchungen bleiben mit ihrem Beleg stehen, und die Umbuchung nennt sie | – |
| Handels- und steuerrechtliche Wertansätze parallel geführt, wenn sie abweichen | 🟡 | Für den einen Fall, in dem sie in Buchfink abweichen, stehen beide Werte nebeneinander: internal/domain/asset.go:461-481 führt Satz, Verteilung und Begründung der Sonderabschreibung nach § 7g Abs. 5 EStG am Anlagegut, internal/accounting/afa.go:373-396 rechnet steuerliche AfA und steuerlichen Restbuchwert neben den handelsrechtlichen, internal/service/tax_register_service.go:90-96, :187-205 stellt beide Buchwerte gegenüber. Abweichende Anschaffungs- oder Herstellungskosten selbst kennt der Datensatz nicht — dafür bleibt es bei der Einheitsbilanz | – |

**Stand.** Die Fortschreibung der Bemessungsgrundlage ist klar und normbezogen gelöst, und seit Welle 5 steht neben dem handelsrechtlichen Wertansatz der steuerliche, wo § 7g Abs. 5 EStG beide trennt. Welle 5c ergänzt die anschaffungsnahen Herstellungskosten: der Instandsetzungs- und Modernisierungsaufwand der ersten drei Jahre läuft gegen die 15-Prozent-Grenze, und wird sie gerissen, wandert er mit Begründung als nachträgliche Herstellungskosten auf das Gebäude. Offen bleiben die Bestandteile der Herstellungskosten und das Wahlrecht dazu. Welle 5.

### BEW-03 Anlagenbuchhaltung und Anlagenspiegel `MUSS`

**Norm:** § 253 Abs. 3 HGB, § 284 Abs. 3 HGB, § 5b Abs. 1 EStG

**Bedeutung:** Der Anlagenspiegel ist Pflichtbestandteil des Anhangs; kleine Kapitalgesellschaften sind nach § 288 Abs. 1 Nr. 1 HGB davon befreit. Ab dem Wirtschaftsjahr 2028 sind Anlagenspiegel und zugrunde liegendes Anlagenverzeichnis zusätzlich elektronisch mit der E-Bilanz zu übermitteln.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Je Wirtschaftsgut Bezeichnung, Inventarnummer, Datum, Kosten, Nutzungsdauer, Methode, kumulierte Abschreibungen, Buchwert, Abgangsdatum und -art | ✅ | internal/domain/asset.go:379-538; kumulierte Abschreibung und Buchwert werden aus den Bewegungen abgeleitet (internal/service/asset_service.go:2721) | – |
| Anlagenspiegel automatisch mit allen Spalten und Vorjahresvergleich | 🟡 | internal/service/asset_service.go:2525-2632 erzeugt Anfangsbestand, Zugänge, Abgänge, Umbuchungen, Zuschreibungen, Jahres-AfA und Endbestände je Bewegungskonto; ein vollständiger Vorjahresspiegel fehlt | 2 |
| Handelsrechtlicher und steuerrechtlicher Anlagenspiegel getrennt ausgebbar | ⛔ | Einheitsbilanz: ein Wertansatz. Die einzige zwingende steuerliche Abweichung (§ 7g Abs. 5 EStG) wird über das Wahlrechtsverzeichnis nach BEW-06 abgebildet | – |
| Anlagenspiegel in der Struktur der E-Bilanz-Taxonomie exportierbar | 🟡 | internal/ebilanz/ebilanz.go:112-176 schreibt den Block in die Instanz, aber mit selbst gewählten Elementnamen; der Code verweist selbst darauf, dass die Form gegen die amtliche Taxonomie zu prüfen ist | 2 |
| Anlagen im Bau und geleistete Anzahlungen als eigene Position führbar und umbuchbar | ✅ | internal/accounting/asset_accounts.go:44-48, internal/service/asset_service.go:1631-1730; Konten im Bau schreiben nicht ab, `Transfer` erzeugt paarweise Bewegungen | – |

**Stand.** Der handelsrechtliche Teil ist praktisch vollständig und der stärkste Teil des Moduls. Offen sind Vorjahresspalte und die amtlichen Elementnamen der Taxonomie. Welle 2.

### BEW-04 Abschreibungen `MUSS`

**Norm:** § 253 Abs. 3 HGB, § 7 EStG, AfA-Tabellen der Finanzverwaltung

**Bedeutung:** Handelsrechtlich richtet sich die planmäßige Abschreibung nach der betrieblichen Nutzungsdauer. Steuerlich sind die amtlichen AfA-Tabellen der Maßstab, die AfA-Tabelle AV geht auf das BMF-Schreiben vom 15.12.2000 zurück und wurde nie durch eine Gesamtfassung ersetzt. Die zeitlich befristeten Sonderregeln sind der eigentliche Aufwandstreiber in der Software.

| Regel | Parameter | Norm |
|---|---|---|
| Lineare AfA | Anschaffungskosten geteilt durch Nutzungsdauer | § 7 Abs. 1 EStG |
| Unterjährige Anschaffung | ein Zwölftel je vollem Monat vor dem Anschaffungsmonat | § 7 Abs. 1 S. 4 EStG |
| Degressive AfA, bewegliche Wirtschaftsgüter | Anschaffung 01.07.2025 bis 31.12.2027, höchstens das Dreifache der linearen AfA, höchstens 30 Prozent | § 7 Abs. 2 EStG |
| Elektrofahrzeuge | Anschaffung 07/2025 bis 12/2027, 75 Prozent im Anschaffungsjahr, danach fallende Staffel | § 7 Abs. 2a EStG |
| Gebäude, betrieblich, Bauantrag nach 31.03.1985 | 3 Prozent linear | § 7 Abs. 4 EStG |
| Wohngebäude, Fertigstellung nach 31.12.2022 | 3 Prozent linear | § 7 Abs. 4 EStG |
| Wohngebäude, degressiv | 5 Prozent vom Buchwert, Baubeginn 01.10.2023 bis 30.09.2029 | § 7 Abs. 5a EStG |
| Computerhardware und Software | Nutzungsdauer ein Jahr zulässig | BMF-Schreiben vom 22.02.2022 |
| Sonderabschreibung | bis 40 Prozent, verteilbar auf fünf Jahre, Gewinngrenze 200.000 Euro | § 7g Abs. 5 EStG |
| Investitionsabzugsbetrag | bis 50 Prozent der voraussichtlichen Kosten, Gewinngrenze 200.000 Euro | § 7g Abs. 1 EStG |

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Abschreibungsmethoden je Wirtschaftsgut wählbar | 🟡 | internal/domain/asset.go:98-140 kennt linear, degressiv, Sammelposten, Sofortabzug und seit Welle 5c die Staffel des § 7 Abs. 2a EStG für Elektrofahrzeuge sowie den festen Gebäudesatz des § 7 Abs. 4 EStG; internal/service/asset_welle5c.go:28-74 hält jede Methode an ihr Konto — degressiv nicht auf ein unbewegliches Wirtschaftsgut, der Gebäudesatz nur auf ein Gebäude, die Staffel nur auf ein Fahrzeug im Fenster der Vorschrift. Außerplanmäßige Abschreibung und Sonderabschreibung sind eigene Wege. Die Leistungsabschreibung nach § 7 Abs. 1 S. 6 EStG ist außerhalb des Funktionsumfangs | – |
| Zeitlich befristete Regeln als datierte Regelsätze, Gesetzesänderung ohne Codeänderung | 🟡 | Die Sätze stehen seit Welle 5c in einer Ressource und nicht mehr als Go-Literale: internal/accounting/afa_rules.json führt Wertgrenzen, die Fenster der degressiven Abschreibung, die Staffel des § 7 Abs. 2a EStG (75, dann 10, 5, 5, 3 und 2 Prozent für Anschaffungen vom 01.07.2025 bis 31.12.2027) und die festen Gebäudesätze des § 7 Abs. 4 EStG, jeder Eintrag mit Geltungszeitraum und Fundstelle; internal/accounting/afa_rules.go:87-153 lädt sie beim Start und gibt sie unverändert an Oberfläche und Tests weiter, internal/accounting/afa.go:12-22 verweist für die Werte dorthin. Es fehlt die degressive Abschreibung für Wohngebäude nach § 7 Abs. 5a EStG. Der Investitionsabzugsbetrag nach § 7g Abs. 1 EStG wird außerbilanziell abgezogen und steht in der Ressource als benannte Auslassung. Die Datei ist eingebettet: ein neuer Satz reist mit der nächsten Auslieferung | Politur |
| Wechsel von degressiver zu linearer Abschreibung möglich und im Anlagenstammsatz dokumentiert | 🟡 | internal/accounting/afa.go:590-612 wechselt nach § 7 Abs. 3 EStG automatisch im optimalen Jahr und vermerkt es in der Planzeile, nicht im Stammsatz; wähl- oder verschiebbar ist er weiterhin nicht | Politur |
| Handels- und Steuerbilanz mit unterschiedlichen Nutzungsdauern und Methoden, Differenz auswertbar und in den latenten Steuern | 🟡 | Die Differenz ist seit Welle 5 auswertbar: internal/service/asset_service.go:1052-1066, :1101-1108 führt die Sonderabschreibung als steuerlichen Wert an der Bewegung, statt sie handelsrechtlich zu buchen, internal/accounting/afa.go:373-396 (`TaxAmount`, `TaxClosingBookValue`, `TaxDifference`) rechnet die steuerliche Reihe mit, internal/service/tax_register_service.go:264-352 leitet daraus über. Einheitsbilanz bleibt es trotzdem: internal/domain/asset.go:414-419 führt genau ein Methoden- und ein Nutzungsdauerfeld, und latente Steuern entfallen für die Zielgruppe (§ 274a Nr. 4 HGB, siehe BEW-11) | – |
| Außerplanmäßige Abschreibungen und Zuschreibungen nach § 253 Abs. 5 HGB mit Begründung, Wertaufholungsgebot durch Bericht unterstützt | ✅ | internal/service/asset_service.go:1269-1273 erzwingt die Begründung bei der Abschreibung, :1353-1364 seit Welle 5c auch bei der Zuschreibung — ohne den weggefallenen Grund ist sie von einer willkürlichen Erhöhung des Buchwerts nicht zu unterscheiden —, und :1374-1383 deckelt sie auf die fortgeführten Anschaffungskosten. Den Bericht liefert internal/service/asset_welle5c.go:282-362 (`WriteUpReport`): jedes Anlagegut mit einer außerplanmäßigen Abschreibung, deren Grund weggefallen sein könnte, mit dem höchstmöglichen Zuschreibungsbetrag; :364-394 hält die Bestätigung fest, dass der Grund fortbesteht, und der Abschlussbaustein „Wertaufholung prüfen" stellt die Frage jedes Jahr (internal/domain/closing_step.go:22-25, :81-83) | – |
| AfA-Tabellenwerte als überschreibbare Vorschlagswerte mit Begründungsfeld | 🟡 | internal/accounting/asset_accounts.go:53-72 führt Vorschlagswert, Quelle und das Kennzeichen, dass eine Abweichung zu begründen ist; :83-166 belegt neun der dreiundvierzig Konten, darunter EDV-Software und die sonstige Betriebs- und Geschäftsausstattung mit den zwölf Monaten des BMF-Schreibens vom 22.02.2022. Das Begründungsfeld steht am Anlagegut (internal/domain/asset.go:513-521), und internal/service/asset_welle5c.go:531-575 verlangt es, sobald die Nutzungsdauer vom Vorschlag dieses Schreibens abweicht — bei einem bestehenden Anlagegut nur, wenn die Nutzungsdauer sich ändert. Die übrigen Konten haben weiterhin keinen Vorschlag | Politur |

**Stand.** Die gebaute Mechanik ist von hoher Qualität, und Welle 5c hebt ihren Umfang auf die Tabelle: die Sätze liegen als datierte Ressource neben dem Code, die Staffel des § 7 Abs. 2a EStG und die festen Gebäudesätze des § 7 Abs. 4 EStG rechnen mit, die zwölf Monate für Computerhardware und Software stehen als Vorschlag mit Begründungspflicht bei der Abweichung, und die degressive Abschreibung ist für unbewegliche Wirtschaftsgüter gesperrt — ein Gebäude auf 0240 lässt sich nicht mehr mit 30 Prozent abschreiben. Die Zuschreibung verlangt ihren Grund, und der Wertaufholungsbericht führt jedes Jahr die Anlagegüter vor, deren außerplanmäßige Abschreibung überholt sein könnte. Die Sonderabschreibung des § 7g Abs. 5 EStG läuft seit Welle 5 durch das Verzeichnis und die Überleitung statt durch das Journal. Offen bleiben die degressive Abschreibung für Wohngebäude nach § 7 Abs. 5a EStG, der wähl- und verschiebbare Übergang von der degressiven auf die lineare Abschreibung und die Tabellenwerte für die übrigen Anlagekonten. Alle drei sind Politur.

### BEW-05 Geringwertige Wirtschaftsgüter und Sammelposten `MUSS`

**Norm:** § 6 Abs. 2, Abs. 2a EStG, R 6.13 EStR

**Bedeutung:** Die Wertgrenzen sind seit 2018 unverändert und wurden vom Wachstumschancengesetz entgegen dem Regierungsentwurf nicht angehoben. Sofortabschreibung bis 800 Euro netto, Aufzeichnungspflicht ab 250 Euro netto, Sammelposten für Wirtschaftsgüter von mehr als 250 bis 1.000 Euro netto mit gleichmäßiger Auflösung über fünf Wirtschaftsjahre.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Wertklasse aus dem Nettobetrag erkannt, Vorschlag Sofortabschreibung, Sammelposten oder Aktivierung | ✅ | internal/accounting/afa.go:229-292; `ClassifyAcquisition` liefert Empfehlung, zulässige Alternativen und Begründung mit Paragraf und fragt die selbständige Nutzbarkeit ab | – |
| Alle Wertgrenzen und die Auflösungsdauer parametrisierbar und zeitabhängig versioniert | 🟡 | Die Sätze ab 2010 und ab 2018 einschließlich der Auflösungsdauer stehen seit Welle 5c in internal/accounting/afa_rules.json und werden über internal/accounting/afa_rules.go:87-112 geladen; internal/accounting/afa.go:12-22 verweist für die Werte dorthin, und derselbe Test liest die Datei. Parametrisierbar sind sie ausdrücklich nicht: die Ressource ist eingebettet und gilt als nicht editierbares Stammdatum | Politur |
| Laufendes Verzeichnis für Wirtschaftsgüter über 250 Euro netto mit Datum und Kosten | ✅ | internal/service/asset_service.go:479; auch der Sofortabzug bleibt in der Kartei stehen, das erfüllt § 6 Abs. 2 S. 4 EStG | – |
| Wahlrecht je Wirtschaftsjahr einheitlich ausgeübt, Bericht über Abweichungen | ✅ | internal/service/asset_welle5c.go:181-226 weist beim Speichern eines Zugangs zwischen Aufzeichnungs- und Sammelpostengrenze die zweite Wahl zurück, sobald im selben Wirtschaftsjahr die andere ausgeübt wurde, und nennt § 6 Abs. 2a Satz 5 EStG samt der bereits so behandelten Zugänge; :126-180 (`PoolConsistency`) stellt den Bericht je Wirtschaftsjahr zusammen — beide Gruppen mit ihren Zugängen und der Aussage, ob das Wahlrecht einheitlich ausgeübt ist | – |
| Abgang aus dem Sammelposten mindert diesen nicht | ✅ | internal/service/asset_service.go:2209-2213 lehnt den Abgang unter Verweis auf § 6 Abs. 2a S. 4 EStG ab, die Auflösung läuft weiter | – |

**Stand.** Nahe an erfüllt. Welle 5c schließt die Einheitlichkeit des Wahlrechts: der zweite Weg im selben Wirtschaftsjahr wird beim Speichern abgewiesen, und der Bericht zeigt je Jahr, welche Zugänge in welcher Gruppe stehen. Offen bleibt die vom Kriterium verlangte Parametrisierbarkeit der Grenzen — sie stehen in einer datierten Ressource, editierbar sind sie nicht. Politur.

### BEW-06 Verzeichnis steuerlicher Wahlrechte `MUSS`

**Norm:** § 5 Abs. 1 S. 2 und 3 EStG, § 60 EStDV

**Bedeutung:** Wer ein steuerliches Wahlrecht abweichend vom handelsrechtlichen Wertansatz ausübt, muss die betroffenen Wirtschaftsgüter in ein besonderes, laufend zu führendes Verzeichnis aufnehmen. Ohne das Verzeichnis ist die Wahlrechtsausübung unwirksam.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Verzeichnis je Wirtschaftsgut mit Tag, Kosten, konkreter Vorschrift und vorgenommenen Abschreibungen | ✅ | internal/service/tax_register_service.go:74-96 hält je Wirtschaftsgut Inventarnummer, Tag der Anschaffung, Anschaffungskosten, die Vorschrift („§ 7g Abs. 5 EStG", :136) und die Begründung der Inanspruchnahme, :152-186 je Geschäftsjahr die handelsrechtliche und die steuerliche Abschreibung mit ihrer Differenz; internal/domain/asset.go:461-481 ist die Quelle am Anlagegut | – |
| Eintrag entsteht automatisch, sobald handels- und steuerrechtlicher Wertansatz abweichen | ✅ | internal/service/tax_register_service.go:128-138 nimmt jedes Anlagegut auf, an dem ein Sonderabschreibungssatz steht — das ist genau der Fall, in dem die Wertansätze auseinanderfallen; internal/service/asset_service.go:1052-1066, :1101-1108 führt die Sonderabschreibung seit Welle 5 als steuerlichen Wert an der Bewegung statt als handelsrechtliche Buchung, so dass der Eintrag ohne eigenen Erfassungsschritt entsteht | – |
| Laufend fortgeschrieben, zu jedem Stichtag als Bericht und als Datei ausgebbar | ✅ | internal/service/tax_register_service.go:111-211 stellt das Verzeichnis bis zum Ende jedes Geschäftsjahres zusammen (Bewegungen nach dem Jahr bleiben außen vor, :154-157), :227-253 (`RegisterCSV`) gibt es als CSV je Wirtschaftsgut und Jahr aus; frontend/src/pages/ClosingModulesPage.tsx:3235-3260 zeigt es und lädt die Datei herunter (:3183-3195), internal/wailsbridge/closing_steps_service.go:443-465 ist der Weg dorthin | – |
| Gleiche Aufbewahrungsfrist wie die Bücher, Übermittlung mit der E-Bilanz | 🟡 | Die Wirkung des Verzeichnisses geht mit: internal/ebilanz/ebilanz.go:392-429 (`reconciliationFacts`) schreibt die Überleitung je Position mit handels- und steuerrechtlichem Wert in die Instanz, internal/service/ebilanz_service.go:81-87 hängt sie ein. Zwei Vorbehalte: die Elementnamen des Überleitungsmoduls stehen in internal/ebilanz/taxonomy_6.9.json:469-478 mit `verified: false` und sind vor der Übermittlung gegen die amtliche Taxonomie abzugleichen (JAB-05), und die Aufbewahrung folgt seit Welle 6 der Klasse der Bücher — zehn Jahre für Abschlüsse, Verzeichnisse und Überleitung (internal/accounting/retention.go:176-182) | Politur |

**Stand.** Mit Welle 5 geschlossen, und der Weg dahin war die Umkehrung einer falschen Buchung: die Sonderabschreibung nach § 7g Abs. 5 EStG wird als steuerlicher Wert am Anlagegut geführt, nicht mehr handelsrechtlich gebucht — seit dem BilMoG ist das unzulässig. Daraus entsteht das Verzeichnis von selbst, mit Vorschrift, Kosten, Jahren und Differenz, als Ansicht und als CSV. Offen bleibt, was nicht am Verzeichnis hängt: der Abgleich der Elementnamen des Überleitungsmoduls gegen die amtliche Taxonomie vor der Übermittlung (JAB-05). Die Aufbewahrungsfrist hat Welle 6 nachgeholt; der Abgleich ist Politur.

### BEW-07 Rückstellungen und Abzinsung `MUSS`

**Norm:** §§ 249, 253 Abs. 1 und 2 HGB, § 6 Abs. 1 Nr. 3a EStG, Rückstellungsabzinsungsverordnung

**Bedeutung:** Rückstellungen sind mit dem nach vernünftiger kaufmännischer Beurteilung notwendigen Erfüllungsbetrag anzusetzen. Bei einer Restlaufzeit über einem Jahr ist abzuzinsen: Altersversorgungsverpflichtungen mit dem Zehnjahresdurchschnitt, sonstige Rückstellungen mit dem Siebenjahresdurchschnitt des Marktzinssatzes. Die Deutsche Bundesbank veröffentlicht die Sätze monatlich. Steuerlich gilt abweichend ein fester Satz von 5,5 Prozent.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Rückstellungsarten nach § 249 HGB abbildbar | ✅ | internal/domain/provision.go:16-63 führt die zehn Arten des § 249 HGB als Aufzählung — ungewisse Verbindlichkeit, Drohverlust, unterlassene Instandhaltung, Kulanzgewährleistung, Steuern, Abschluss- und Aufbewahrungskosten, Personal, Pensionen —, internal/accounting/provision.go:20-43 (`ProvisionAccounts`) schlägt je Art Bilanz- und Aufwandskonto vor; internal/service/provision_service.go:370-456 (`BookFormation`, `BookIncrease`, `bookProvisionMovement`) bildet und bucht sie, frontend/src/pages/ClosingModulesPage.tsx:1133-1373 ist die Maske dazu | – |
| Je Rückstellung Erfüllungsbetrag, erwartete Restlaufzeit, Abzinsungssatz und abgezinster Wert | ✅ | internal/domain/provision.go:193-210 führt Erfüllungsbetrag (§ 253 Abs. 1 Satz 2 HGB), erwarteten Erfüllungszeitpunkt, abgezinsten Wert und den verwendeten Satz am Datensatz — der Satz wird mitgeschrieben, damit die Rechnung nachvollziehbar bleibt, wenn die Zinstabelle fortgeschrieben wird; internal/accounting/provision.go:56-138 rechnet Restlaufzeit, Abzinsungspflicht und Barwert, internal/service/provision_service.go:262-301 setzt beides zusammen und zeigt es in der Vorschau (frontend/src/pages/ClosingModulesPage.tsx:1333-1354) | – |
| Abzinsungssätze der Bundesbank pflegbar, mit Monat und Restlaufzeit als Schlüssel | ✅ | internal/domain/provision.go:300-337 führt die Sätze als eigene Tabelle mit Monat, Restlaufzeit und Mittelungsdauer als Schlüssel (sieben Jahre nach § 253 Abs. 2 Satz 1 HGB, zehn für Pensionen nach Satz 2); internal/service/provision_service.go:850-933 pflegt sie einzeln oder als CSV der Bundesbank-Veröffentlichung, :961-978 sucht den Satz des Stichtagsmonats und nennt es, wenn ein älterer genommen wurde. Fehlt der Satz, wird nicht abgezinst und ein Befund erzeugt (:766-829), der im Prüflauf steht (internal/service/check_service.go:107-126). Gepflegt wird in frontend/src/pages/ClosingModulesPage.tsx:1599-1859 | – |
| Handels- und steuerrechtlicher Wertansatz parallel geführt | ✅ | Gebucht wird einer, ausgewiesen werden beide: internal/service/provision_service.go:192-195, :296-300 rechnet zu jeder Rückstellung den steuerlichen Wert mit 5,5 % (§ 6 Abs. 1 Nr. 3a Buchst. e EStG) neben dem handelsrechtlichen Barwert und stellt ihn in die Vorschau (frontend/src/pages/ClosingModulesPage.tsx:1347-1352); internal/service/tax_register_service.go:305-352, :354-392 führt beide Bestände zum Stichtag in der Überleitung zusammen, mit dem Ansatzverbot für Drohverluste (§ 5 Abs. 4a EStG) als eigener Zeile. Für Pensionsrückstellungen unterbleibt der Vergleich ausdrücklich: § 6a EStG verlangt eine Gutachtenrechnung, keine Abzinsung mit 5,5 % (:360-367) | – |
| Auflösung nur bei Wegfall des Grundes, mit Begründung protokolliert | ✅ | internal/service/provision_service.go:477-482 weist eine Auflösung ohne Grund ab und nennt § 249 Abs. 2 Satz 2 HGB, :496-500 lehnt einen Betrag über dem Bestand ab, statt ihn stillschweigend zu kappen; der Grund steht an der Bewegung (internal/domain/provision.go:174-178) und geht in das Änderungsprotokoll und den Eigenbeleg der Buchung (internal/service/provision_service.go:622-661) | – |

**Stand.** Mit Welle 5 vollständig gebaut, und die Abzinsung ist der Teil, an dem sich die Haltung zeigt: die Sätze der Deutschen Bundesbank stehen in einer pflegbaren Tabelle und nicht im Code, weil ein mitgeliefertes Programm einen Monat nach der Auslieferung falsch rechnete — und fehlt der Satz, erzeugt Buchfink einen Befund für den Prüflauf, statt abzuzinsen oder zu raten. Der steuerliche Wert mit 5,5 % läuft neben dem handelsrechtlichen mit und geht in die Überleitung; gebucht wird er nicht. Alle fünf Kriterien sind erfüllt.

### BEW-08 Rechnungsabgrenzung `MUSS`

**Norm:** § 250 HGB, § 252 Abs. 1 Nr. 5 HGB, § 5 Abs. 5 EStG

**Bedeutung:** Ausgaben und Einnahmen vor dem Abschlussstichtag, die Aufwand oder Ertrag für eine bestimmte Zeit danach darstellen, sind abzugrenzen.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Aktive und passive Rechnungsabgrenzungsposten als eigene Bilanzpositionen führbar | ✅ | internal/domain/accrual.go:9-64 unterscheidet aktiven Posten, passiven Posten und Disagio und legt jeder Art ihr Bilanzkonto zu (1900 bzw. 3900, internal/domain/skr04_accounts.go:19-21); internal/domain/accrual.go:151-184 führt den Posten mit Bestand und Auflösungen, internal/service/accrual_service.go:690-736 weist ihn zum Stichtag getrennt nach aktiv und passiv aus | – |
| Abgrenzungen mit Startdatum, Enddatum und Verteilungsschlüssel, automatisch periodisch aufgelöst | ✅ | internal/domain/accrual.go:167-177 führt Beginn, Ende, Stichtag, Konto und Verfahren, :66-128 die beiden Verteilungsschlüssel (Zwölftel oder taggenau) und den Auflösungstakt; internal/accounting/accrual.go:97-141 rechnet den abgegrenzten Anteil, :157-218 den gespeicherten Auflösungsplan. Vorgeschlagen werden die Posten aus den Buchungen, deren Leistung über den Stichtag reicht (internal/service/accrual_service.go:157-267, über `ServiceDateTo`, internal/domain/journal.go:117-118), aufgelöst werden sie beim Saldenvortrag (:561-625, internal/service/closing_service.go:1157-1165) | – |
| Bericht je Stichtag über den Bestand aller Abgrenzungen mit Restlaufzeit | ✅ | internal/service/accrual_service.go:664-736 (`Report`) stellt zu jedem Stichtag jeden gebildeten und noch nicht aufgelösten Posten mit gebildetem Betrag, bereits aufgelöstem Teil, Restbetrag und Restlaufzeit in Kalendertagen zusammen, getrennt nach aktiv und passiv; ein Posten, dessen Bildung storniert wurde, fällt heraus (:697-703). frontend/src/pages/ClosingModulesPage.tsx:1022-1070 zeigt ihn | – |
| Disagio nach § 250 Abs. 3 HGB als eigener Fall abbildbar | ✅ | internal/domain/accrual.go:21-27 führt das Damnum als eigene Art und nicht als Spielart des aktiven Postens — die Verteilung richtet sich nach der Darlehenslaufzeit, und der Aufwand ist Zinsaufwand; internal/service/accrual_service.go:462-481 (`disagioAccount`) zwingt die Auflösung auf das Zinsaufwandskonto, internal/service/accrual_service.go:444-461 baut den Buchungssatz für Bildung und Auflösung | – |

**Stand.** Mit Welle 5 gebaut, und die Vorarbeit hat sich ausgezahlt: der Vorschlag entsteht aus dem Leistungsende, das jede Buchung ohnehin hat, und ist damit keine Schätzung. Der Auflösungsplan wird bei der Bildung gespeichert und nicht bei jedem Aufruf neu gerechnet — was gebucht ist, bleibt dasselbe, auch wenn jemand später das Verteilungsverfahren umstellt —, und der Saldenvortrag bucht ihn im Folgejahr ab. Alle vier Kriterien sind erfüllt.

### BEW-09 Inventar und Vorratsbewertung `MUSS`

**Norm:** §§ 240, 241, 256 HGB

**Bedeutung:** Zum Schluss jedes Geschäftsjahres ist ein Inventar aufzustellen. Die Vereinfachungen des § 241 HGB (Stichprobeninventur, permanente Inventur, verlegte Inventur) sind zulässig, wenn das Verfahren den GoB entspricht. Als Verbrauchsfolgeverfahren nennt § 256 HGB ausschließlich Lifo und Fifo.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Inventar zum Stichtag als Bericht, mit dem Bilanzansatz abgestimmt, Differenzen begründungspflichtig | ⛔ | Kein Lager und kein Vorratsvermögen: das Inventar selbst entsteht außerhalb von Buchfink, ein eigenes führt es nur für das Anlagevermögen. Die Ersatzmaßnahme steht seit Welle 5 und ist der Abgleich mit dem Bilanzansatz: internal/service/closing_booking_service.go:85-153 stellt je Vorratskonto Buchwert und erfassten Inventurwert gegenüber, internal/domain/inventory.go:20-69 hält Wert, Tag und Verfahren der Aufnahme (§ 241 HGB) und verlangt die Inventurliste als Beleg, internal/service/closing_booking_service.go:263-404 prüft, dass es sie gibt und dass sie nicht schon zu einer anderen Aufnahme gehört, und bucht die Differenz als Bestandsveränderung. Bewertet wird nicht — der erfasste Wert ist der bewertete | – |
| Festwertverfahren nach § 240 Abs. 3 HGB mit Erinnerung an die Bestandsaufnahme im Dreijahresrhythmus | ⛔ | Kein Vorratsmodul | – |
| Gruppenbewertung mit gewogenem Durchschnitt nach § 240 Abs. 4 HGB | ⛔ | Kein Vorratsmodul | – |
| Lifo und Fifo je Bewertungsgruppe wählbar und stetig fortgeführt | ⛔ | Kein Vorratsmodul | – |
| Strenges Niederstwertprinzip des § 253 Abs. 4 HGB zum Stichtag angewendet und dokumentiert | ⛔ | Betrifft das Umlaufvermögen; für Anlagegüter ist das gemilderte Prinzip umgesetzt (internal/service/asset_service.go:435) | – |
| Verlegte Inventur mit Aufnahmefenster, Fortschreibung und Rückrechnung | ⛔ | Kein Vorratsmodul | – |

**Stand.** Außerhalb des Funktionsumfangs. Für einen Mandanten mit Warenbestand ist das eine harte Grenze, und sie gehört sichtbar in den Einrichtungsweg und in die Verfahrensdokumentation, damit sie nicht erst beim Abschluss auffällt. Was Welle 5 ergänzt, ist der Anschluss an die Bilanz: der Inventurwert je Vorratskonto wird mit Aufnahmeverfahren und Inventurliste erfasst und als Bestandsveränderung gebucht. Die Aufnahme selbst und ihre Bewertung bleiben draußen.

### BEW-10 Fremdwährung `MUSS*`

**Norm:** § 244 HGB, § 256a HGB

**Bedeutung:** Auf fremde Währung lautende Vermögensgegenstände und Verbindlichkeiten sind zum Devisenkassamittelkurs am Abschlussstichtag umzurechnen. Bei einer Restlaufzeit von einem Jahr oder weniger gelten Höchstwert- und Imparitätsprinzip nicht, unrealisierte Kursgewinne werden also erfolgswirksam erfasst.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Je Fremdwährungsbuchung Originalbetrag, Währung, Kurs, Kursquelle und Kursdatum, Eurobetrag daraus berechnet | ✅ | internal/domain/journal.go:112-125 führt seit Welle 5c den Fremdbetrag je Zeile, :197-200 Währung, Kurs, Quelle und Kurstag im Buchungskopf, und beides geht in die Hash-Kanonisierung ein (internal/accounting/journalhash.go:57, :96-97). Erfasst wird der Betrag, den die Rechnung nennt; den Eurobetrag rechnet internal/service/posting_currency.go:40-115 daraus — je Zeile gerundet, der Rest auf der letzten, damit die Summe der Buchung dem Rechnungsbetrag entspricht, und die Endsumme des Belegs ist die Kontrollsumme dazu (:116-157) | – |
| Kurse aus nachvollziehbarer Quelle bezogen und historisiert gespeichert | ✅ | internal/currency/ecb.go:63-126 holt den EZB-Referenzkurs des Belegtages und gibt bei jedem Fehler einen Fehler zurück — der stille Rückfall auf 1,0 ist fort, und die Adresse des Dienstes ist eine Einstellung (:24-29). internal/service/currency_service.go:87-149 (`RateAt`) nimmt zuerst den gespeicherten Kurs, schreibt jeden geholten mit Quelle und Tag in die Historie (internal/domain/currency.go:25-45), legt den Kurs zusätzlich unter dem Tag seiner Feststellung ab und weist einen älteren ausdrücklich als Näherung aus. Ohne Kurs kommt die Aufforderung, ihn mit seiner Quelle einzutragen (:151-174) | – |
| Stichtagsbewertung aller Fremdwährungsposten, Kursdifferenzen getrennt nach realisiert und unrealisiert | ✅ | internal/service/currency_valuation.go:51-201 bewertet jeden offenen Posten in Fremdwährung zum Stichtagskurs, :466-534 die Bankkonten in Fremdwährung dazu, und internal/service/asset_service.go:2101-2191 die Finanzanlagen wie bisher. Gebucht wird auf 6880 und 4840 (internal/accounting/asset_accounts.go:525-526) und nur der unrealisierte Teil; die realisierte Kursdifferenz entsteht beim Zahlungsausgleich und bleibt davon getrennt. Die Bewertung gilt dem Stichtag und nicht dem Posten: :277-357 (`ReverseInto`) löst sie mit dem Saldenvortrag am ersten Tag des Folgejahres wieder auf, und ein zweiter Lauf holt eine fehlende Auflösung nach. Geführt wird sie vom Abschlussbaustein „Fremdwährungsbewertung" (internal/domain/closing_step.go:27-29, :84-87) | – |
| Sonderregel für Restlaufzeiten bis ein Jahr automatisch über die Fälligkeit | ✅ | internal/service/currency_valuation.go:397-412 (`currencyShortTerm`) misst die Restlaufzeit am Zahlungsziel des offenen Postens und liest einen Posten ohne Zahlungsziel als sofort fällig; :141-155 erfasst den Kursverlust in jedem Fall (§ 252 Abs. 1 Nr. 4 HGB), den Kursgewinn erfolgswirksam nur bei einer Restlaufzeit bis zu einem Jahr (§ 256a Satz 2 HGB) und lässt ihn sonst stehen. Jede Zeile hat den Satz, der sie begründet; an der Finanzanlage gilt dieselbe Regel wie bisher (internal/service/asset_service.go:2026-2040) | – |
| Umrechnungskurs nach § 16 Abs. 6 UStG je Beleg neben dem handelsrechtlichen Kurs | ✅ | internal/domain/currency.go:77-91 führt die amtlichen Durchschnittskurse als eigene Tabelle je Monat und Währung, internal/service/currency_service.go:191-314 pflegt sie einzeln oder als CSV; internal/service/posting_currency.go:69-76 rechnet die Bemessungsgrundlage der Steuerzeile mit dem Durchschnittskurs, während der Aufwand am EZB-Kurs des Belegtages hängt, und :158-217 bucht die Differenz zwischen beiden als Kursaufwand oder Kursertrag auf 6880 und 4840. Fehlt der Monatskurs, bleibt es nach § 16 Abs. 6 Satz 1 UStG beim Tageskurs, und es entsteht keine Differenz | – |

**Stand.** Welle 5c holt die Fremdwährung in das laufende Geschäft. Der Anwender erfasst die Beträge, die auf der Rechnung stehen, Buchfink holt den EZB-Referenzkurs des Belegtages, speichert ihn mit Quelle und Tag in der Historie und rechnet den Eurobetrag daraus; ohne Kurs entsteht keine Buchung, und geraten wird keiner mehr. Für die Umsatzsteuer läuft der amtliche Monatsdurchschnitt daneben, und die Differenz zwischen beiden Kursen steht als Kursaufwand oder Kursertrag auf 6880 und 4840. Zum Stichtag werden offene Posten und Fremdwährungskonten bewertet — der Verlust in jedem Fall, der Gewinn nach der Restlaufzeit —, und der Saldenvortrag löst die Bewertung am ersten Tag des Folgejahres wieder auf. Alle fünf Kriterien sind erfüllt.

### BEW-11 Latente Steuern `MUSS*`

**Norm:** § 274 HGB, § 274a Nr. 4 HGB

**Bedeutung:** Bei Differenzen zwischen handels- und steuerrechtlichen Wertansätzen besteht Passivierungspflicht für Steuermehrbelastungen und ein Aktivierungswahlrecht für Steuerminderbelastungen. Kleine Kapitalgesellschaften sind nach § 274a Nr. 4 HGB befreit. Differenzen aus dem Mindeststeuergesetz bleiben außer Ansatz.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Differenz je Bilanzposition zwischen Handels- und Steuerbilanz, eingeordnet als temporär oder permanent | ⛔ | Buchfink führt eine Einheitsbilanz und richtet sich an kleine Kapitalgesellschaften, die nach § 274a Nr. 4 HGB befreit sind | – |
| Unternehmensindividueller Steuersatz pflegbar, keine Abzinsung | ⛔ | Wie oben | – |
| Verlustvorträge nur bis zur Fünfjahresprognose, Prognose erfassbar und dokumentiert | ⛔ | Wie oben | – |
| Befreiung kleiner Kapitalgesellschaften über die Größenklasse steuerbar | ❌ | Eine Größenklasse nach §§ 267, 267a HGB wird nirgends geführt; sie kommt in Welle 2 und warnt ab mittelgroß (siehe JAB-02) | 2 |
| Anhangangaben zu latenten Steuern aus den Daten abgeleitet | ⛔ | Kein Anhang-Generator für latente Steuern; ab mittelgroß verweist Buchfink an den Steuerberater | – |

**Stand.** Für die Zielgruppe entfällt die Pflicht. Was gebaut werden muss, ist die Größenklasse, die diese Befreiung überhaupt erst belegt und ab mittelgroß warnt. Welle 2.

### BEW-12 Nicht abziehbare Betriebsausgaben `MUSS`

**Norm:** § 4 Abs. 5 und Abs. 7 EStG, § 15 Abs. 1a UStG

**Bedeutung:** Bestimmte Betriebsausgaben sind einzeln und getrennt von den sonstigen Betriebsausgaben aufzuzeichnen. Fehlt die getrennte Aufzeichnung, entfällt der Abzug vollständig, auch wenn die Aufwendung dem Grunde nach abziehbar wäre. Betroffen sind unter anderem Geschenke, Bewirtung, Gästehäuser, Jagd und Fischerei sowie das häusliche Arbeitszimmer. Für Bewirtungen ab dem 1. Januar 2025 gilt das BMF-Schreiben vom 19.11.2025; danach darf die Bewirtungsrechnung digital übermittelt oder nachträglich digitalisiert und der Eigenbeleg digital erstellt werden, wenn die GoBD in der Fassung vom 14.07.2025 eingehalten und die Verfahren in der Verfahrensdokumentation beschrieben sind.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Je Kategorie nach § 4 Abs. 7 EStG ein eigenes Konto oder ein eigener Schlüssel, Vermischung ausgeschlossen | ✅ | internal/accounting/posting_groups.go:228-256 führt jede Kategorie als eigene Buchungsgruppe mit eigenen Konten: Bewirtung auf 6640 und 6644, Geschenke auf 6610 und 6620, das ausschließlich betrieblich nutzbare Geschenk auf 6625 und Gästehaus, Jagd, Fischerei und Yacht auf 6645 ohne Vorsteuerabzug (§ 15 Abs. 1a UStG). Der freie Kontoweg führt seit Welle 5c nicht mehr daran vorbei: :336-360 (`AccountsRequiringGroup`) sammelt diese Konten, internal/service/posting_input_tax.go:246-255 weist sie ab, wenn jemand sie von Hand wählt, und nennt die Gruppe — an ihr hängen die Aufzeichnungspflicht, die Freigrenze je Empfänger und der Vorsteuerausschluss. Das häusliche Arbeitszimmer trifft eine Kapitalgesellschaft nicht | – |
| Geschenke je Empfänger und Wirtschaftsjahr kumuliert, Freigrenze von 50 Euro überwacht | ✅ | internal/service/posting_input_tax.go:342-427 (`resolveGift`) verlangt den Empfänger als Kontakt oder Namen, hält die Aufzeichnung nach § 4 Abs. 7 EStG an der Buchung (internal/domain/gift.go:21-96, in der Hash-Kette wie die Bewirtungsaufzeichnung) und misst die bisherigen Geschenke des Wirtschaftsjahres an der Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG (internal/accounting/tax_params.go:48-55, 50 € für Wirtschaftsjahre ab 2024, gemessen am ersten Tag des Wirtschaftsjahres); :428-448 liest die Summe je Empfänger aus dem Journal. Wird die Grenze gerissen, warnt Buchfink vor der Buchung und bucht das Geschenk auf das nicht abziehbare Konto ohne Vorsteuerabzug (:296-303); die früheren Geschenke desselben Empfängers bleiben stehen, bis internal/service/gift_service.go:380-532 (`RebookGiftsForRecipient`) sie mit Generalumkehr und Neubuchung umbucht | – |
| Bewirtung automatisch in 70 Prozent abziehbar und 30 Prozent nicht abziehbar geteilt, Vorsteuer bleibt vollständig | ✅ | internal/service/posting_service.go:463-482, internal/accounting/tax_params.go:23-27; die Quote steht als datierter Parameter, der nicht abziehbare Rest wird als Differenz gebildet, die Vorsteuerbemessungsgrundlage bleibt der volle Nettobetrag | – |
| Eigenbeleg mit Anlass und Teilnehmern erfasst und mit der Rechnung verknüpft aufbewahrt | ✅ | internal/domain/journal.go:281-320; Ort, Tag, Teilnehmer und Anlass sind Pflicht, die Aufzeichnung steht an der Buchung und ist in die Hash-Kette einbezogen | – |
| Bericht je Kategorie mit abziehbaren und nicht abziehbaren Beträgen des Wirtschaftsjahres | ✅ | internal/service/gift_service.go:196-327 (`NonDeductibleReport`) stellt je Kategorie des § 4 Abs. 5 EStG die abziehbaren und die nicht abziehbaren Beträge des Wirtschaftsjahres zusammen — gezählt über die Konten, damit auch eine Buchung ohne Aufzeichnung erscheint und eine Generalumkehr ihre Buchung wieder herausnimmt —, dazu die Geschenke je Empfänger und die Buchungen, die nach einer Überschreitung umzubuchen sind; internal/accounting/non_deductible.go:12-59 hält die Kategorien mit Vorschrift, Konten und dem Satz, der unter der Zeile steht. Gezeigt wird er auf der Seite „Nebenpflichten" (frontend/src/pages/ObligationsPage.tsx:1323), wo auch die Geschenke je Empfänger und die Umbuchung stehen; die Auswertungsseite verweist dorthin (frontend/src/pages/ReportsPage.tsx:413-440) | – |

**Stand.** Die Bewirtung war das Muster, und Welle 5c zieht die übrigen Kategorien des § 4 Abs. 7 EStG nach: jede hat ihre Buchungsgruppe mit eigenen Konten, das Geschenk verlangt seinen Empfänger, die Freigrenze läuft je Empfänger und Wirtschaftsjahr mit, und wer sie reißt, bekommt die Warnung vor der Buchung und das nicht abziehbare Konto ohne Vorsteuerabzug. Die Konten dieser Kategorien sind von Hand nicht mehr erreichbar; wer eines wählt, bekommt den Weg über die Gruppe genannt. Der Bericht führt je Kategorie und Jahr die abziehbaren und die nicht abziehbaren Beträge, nennt die Empfänger und die Buchungen, die nach einer Überschreitung umzubuchen sind, und die Umbuchung läuft als eigener, sichtbarer Vorgang aus Generalumkehr und Neubuchung. Alle fünf Kriterien sind erfüllt.

### BEW-13 Schuldzinsenabzug `MUSS*`

**Norm:** § 4 Abs. 4a EStG

**Bedeutung:** Übersteigen Entnahmen die Summe aus Gewinn und Einlagen, sind die darauf entfallenden Schuldzinsen typisiert mit sechs Prozent der Überentnahme nicht abziehbar. Die Überentnahmen der Vorjahre wirken kumulativ fort. Für Kapitalgesellschaften ist die Norm über verdeckte Gewinnausschüttungen und Personengesellschaftsbeteiligungen mittelbar relevant.

| Kriterium | Status | Fundstelle / Grund | Welle |
|---|---|---|---|
| Entnahmen und Einlagen je Wirtschaftsjahr getrennt erfasst und über die Betriebszugehörigkeit fortgeschrieben | ⛔ | Buchfink ist für Kapitalgesellschaften gebaut, die keine Entnahmen kennen. Die Rechtsformen mit Entnahmen bleiben wählbar (internal/domain/legalform.go:112-134) und zeigen seit Welle 6 den Hinweis, dass Kapitalkonten, Entnahmen und Einlagen sowie § 4 Abs. 4a EStG in dieser Fassung nicht abgebildet sind: internal/domain/legalform.go:137-157 formuliert ihn, internal/wailsbridge/nachweise_service.go:316-320 gibt ihn zur gespeicherten Rechtsform aus, gezeigt im Einrichtungsassistenten und in den Einstellungen (frontend/src/pages/SettingsPage.tsx:434-441) | – |
| Kumulierte Überentnahme je Stichtag berechnet und als Bericht abrufbar | ⛔ | Wie oben | – |
| Sockelbetrag von 2.050 Euro berücksichtigt | ⛔ | Wie oben | – |
| Zinsen für Investitionsdarlehen gesondert kennzeichenbar und ausgenommen | ⛔ | Wie oben | – |

**Stand.** Außerhalb des Funktionsumfangs, und der Vorbehalt ist mit Welle 6 eingelöst: wer ein Einzelunternehmen oder eine Personenhandelsgesellschaft einrichtet, liest beim Einrichten und in den Einstellungen, dass Kapitalkonten, Entnahmen und der Schuldzinsenabzug nicht abgebildet sind, statt eine Buchführung zu bekommen, die eine ihn treffende Hinzurechnung stillschweigend übergeht.

---
