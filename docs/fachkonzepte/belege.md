# Beleg- und Dateiverwaltung

[Fachkonzepte](README.md) · [Beleg- und Buchungsflow](beleg-buchungsflow.md)

## 15. Beleg- & Dateiverwaltung

### Ein Beleg ist mehrere Dateien, nicht eine

Das ist die Konsequenz aus der E-Rechnung, und sie gehört hierher und nicht in ein
Randdokument: eine ZUGFeRD-Rechnung ist ein PDF **mit eingebettetem XML**, eine
XRechnung ist ein XML **ganz ohne** PDF, ein gescannter Papierbeleg ist ein Bild,
und ein Bewirtungsbeleg besteht aus Rechnung plus Eigenbeleg mit Teilnehmern.

Das heutige Modell bildet das nicht ab. Am Journaleintrag hängt genau ein
`DocumentHash` und ein `DocumentPath` – ein Beleg, eine Datei. Für den
Hybridfall müsste man sich entscheiden, welche Hälfte man sichert, und läge in
beide Richtungen falsch: sichert man das PDF, fehlt der Teil, aus dem der
Vorsteuerabzug kommt; sichert man das XML, fehlt das, was der Nutzer ansieht.

**Der Beleg wird deshalb eine eigene Entität zwischen Datei und Buchung:**

| Feld | Inhalt |
|---|---|
| **Belegnummer** | aus dem eigenen Nummernkreis, das Belegfeld für den Prüfer |
| **Richtung** | Eingang / Ausgang |
| **Dateien** | 1..n, jede mit Rolle, Hash, MIME-Typ und Originaldateiname |
| **Beleg-Hash** | über die geordnete Liste der Dateien, siehe unten |
| **Empfangsweg und -zeitpunkt** | bei Eingangsbelegen: wann kam der Beleg herein |
| **Buchungsbezug** | in beide Richtungen |

Die Rolle je Datei ist die tragende Angabe:

| Rolle | Bedeutung |
|---|---|
| **original** | die Datei in der Form, in der sie empfangen wurde – genau eine je Beleg |
| **structured** | der strukturierte Rechnungsdatensatz; bei einem Hybridformat aus dem Original extrahiert und damit abgeleitet, bei einer XRechnung mit dem Original identisch |
| **rendering** | eine von Buchfink erzeugte menschenlesbare Darstellung, wenn das Original keine hat |
| **attachment** | Eigenbeleg, Teilnehmerliste, Lieferschein, Zahlungsnachweis |

Das folgt direkt aus der GoBD, keine Formalie: eingehende Belege sind
in dem Format aufzubewahren, in dem sie empfangen wurden (Rz. 131), und der
strukturierte Datenteil darf nicht durch eine Formatumwandlung verloren gehen
(Rz. 125, Beispiel 10). „Original" und „strukturierter Teil" sind deshalb zwei
Rollen und nicht zwei Namen für dieselbe Datei.

### Was das für die Anzeige heißt

Die Vorschau kann sich nicht mehr darauf verlassen, dass es ein Bild gibt. Drei
Fälle:

| Beleg | Was angezeigt wird |
|---|---|
| Papier, Scan, Foto, reines PDF | die Originaldatei |
| ZUGFeRD (Hybrid) | der PDF-Teil des Originals – **gebucht wird trotzdem aus dem XML** |
| XRechnung (reines XML) | eine von Buchfink erzeugte Darstellung, Rolle `rendering` |
| eigene Ausgangsrechnung | das beim Ausstellen erzeugte hybride PDF |

Der dritte Fall ist neu und nicht optional: eine XRechnung hat schlicht keinen
Bildteil. Ohne eigene Darstellung kann der Nutzer den Beleg nicht prüfen, den er
gerade bucht.

Umgekehrt gilt die Trennung genauso streng: bei einem Hybridbeleg ist der Bildteil
**Anzeige, nie Buchungsquelle**. Weicht er inhaltlich vom XML ab, ist das
potenziell eine zweite Rechnung mit § 14c-Folgen – Buchfink zeigt die Abweichung
an und bewertet sie nicht.

### Ablegen und Buchen sind zwei Schritte

Das ist die Unterscheidung, die man beim Bauen zuerst falsch macht. Eine
XRechnung besteht nur aus XML und hat keinen Bildteil. Sie muss trotzdem
**sofort ablegbar** sein – die GoBD verlangen die Aufbewahrung in der
empfangenen Form (Rz. 131), und die Darstellung kann erst danach erzeugt
werden. Was sie nicht sein darf, ist **buchbar**, bevor diese Darstellung
existiert: niemand soll eine Buchung zu einem Beleg freigeben, den er nicht
ansehen kann.

Die Prüfungen sind also zwei:

| Prüfung | Wann | Was |
|---|---|---|
| **Struktur** | beim Ablegen | genau ein Original, höchstens ein strukturierter Teil, höchstens eine Darstellung, jede Datei mit Prüfsumme; abgeleitete Dateien als solche gekennzeichnet |
| **Buchbarkeit** | beim Buchen | zusätzlich: der Beleg ist anzeigbar |

Wer beides in eine Prüfung legt, kann eine XRechnung gar nicht erst annehmen –
und verstößt damit gegen genau die Norm, die er einhalten will.

### Ablage und Hash

- **Ablage** nach Jahr und Richtung (`belege/<jahr>/eingang/…`, `…/ausgang/…`),
  jede Datei unverändert.
- **Je Datei** ein SHA256 über den Dateiinhalt.
- **Der Dateiname auf der Platte ist die Prüfsumme**, nicht die Belegnummer.
  Das ist kein Detail: stünde die Belegnummer im Pfad, müsste sie feststehen,
  bevor die erste Datei geschrieben wird – also *vor* der Transaktion, die den
  Beleg einfügt. Eine fehlgeschlagene Ablage risse dann eine Lücke in den
  Nummernkreis. Inhaltsadressierte Ablage entkoppelt beides, und identische
  Dateien liegen nebenbei nur einmal auf der Platte. Die Originalendung bleibt
  angehängt (`<prüfsumme>.pdf`) – wer den Belegordner außerhalb von Buchfink
  öffnet, fände sonst ein Verzeichnis typloser Blobs.
- **Geschrieben wird in eine Temporärdatei, dann `fsync` und atomares Umbenennen.**
  Eine mitten im Schreiben abgebrochene Ablage hinterlässt damit keine halbe
  Belegdatei, sondern gar keine.
- **Je Beleg** ein Beleg-Hash über die *geordnete* Liste aus Rolle,
  Originaldateiname und Datei-Hash – längenpräfigiert wie bei der Buchung, damit
  kein Dateiname eine Feldgrenze vortäuschen kann. Damit ändert jede zusätzliche,
  entfernte oder umsortierte Datei den Beleg-Hash.
- **In der Buchung** steht der Beleg-Hash, nicht der Pfad. Ein verschobener
  Datenordner darf die Kette nicht brechen.

Daraus folgt eine Regel, die zum bestehenden Umgang mit gebuchten Daten passt:
**mit der Buchung ist der Beleg versiegelt.** Nachträglich eine Datei anzuhängen
würde den Beleg-Hash und damit die Kette brechen. Was später dazukommt – eine
Mahnung, ein Zahlungsnachweis, eine korrigierte Rechnung – ist ein eigener Beleg,
der auf denselben Geschäftsvorfall zeigt. Für eine inhaltliche Korrektur gilt
weiter Storno plus Neuerfassung.

Das Versiegeln gehört **hinter** den Journalschreibvorgang. Scheitert die
Buchung, muss der Beleg offen bleiben und korrigiert erneut gebucht werden
können; die andere Reihenfolge hinterließe einen unveränderlichen Beleg ohne
Buchung.

### Hinweis auf die E-Rechnungspflicht

Stellt ein inländischer Lieferant, der Unternehmer ist, eine gewöhnliche PDF- oder
Papierrechnung, ist das zweierlei: ein Risiko für den eigenen Vorsteuerabzug und
ein Anlass, den Lieferanten auf eine Pflicht hinzuweisen, die für ihn läuft.
Buchfink zeigt beides an der Buchungsvorschau an und **blockiert nie** – was daraus
folgt, ist eine Rechtsfrage.

Der Text hängt am Belegdatum. Bis zum 31.12.2026 ist die sonstige Rechnung nach
§ 27 Abs. 38 Nr. 1 UStG noch zulässig, also ein reiner Hinweis. Für 2027 hängt es am
Gesamtumsatz des **Ausstellers** im Vorjahr (§ 27 Abs. 38 Nr. 2 UStG) – den Buchfink
nicht kennt, was der Hinweis sagt, statt eine Bewertung zu behaupten. Ab 2028 gibt es
keine Übergangsregelung mehr.

Kein Hinweis erscheint, wo keine Pflicht besteht: bei einem strukturierten Teil am
Beleg, bei ausländischen Lieferanten, bei Kleinunternehmern (§ 34a UStDV), bei
Privatpersonen, unterhalb der Kleinbetragsgrenze des § 33 UStDV und bei steuerfreien
Umsätzen. Der letzte Punkt ist eine **Näherung**: Buchfink weiß, dass ein Umsatz als
steuerfrei behandelt wurde, nicht welche Nummer des § 4 UStG greift – nur Nr. 8 bis 29
nehmen die Pflicht heraus.

Ob ein Geschäftspartner Unternehmer und ob er Kleinunternehmer ist, steht in den
Kontaktstammdaten. An einem Hinweis zum Vorsteuerabzug darf nicht hängen, ob jemand
einen Firmennamen eingetippt hat.

### Die Oberfläche rechnet nicht

Netto, Steuer und Brutto einer Erfassungsmaske kommen aus einer Buchungsvorschau
des Backends, die exakt dieselben Zeilen erzeugt wie die spätere Buchung – sie
schreibt sie nur nicht. Die Rechnungsmaske hatte die Steuerermittlung samt
Rundung je Steuersatzgruppe selbst nachgebaut, mit dem Kommentar „genau wie im
Backend". Das ist eine zweite Wahrheit, die auseinanderläuft, sobald ein
Steuerfall dazukommt – und zwar die, die niemand testet.

Ebenso wird der Steuerfall nicht in der Maske hergeleitet: welche Stammdaten er
verlangt, sagt das Backend über `TaxTreatmentInfo`, und welchen Fall eine
Buchungsgruppe vorschlägt, sagt die Gruppe selbst.

### Keine Buchung ohne Beleg

Der Belegbezug ist verbindlich, nicht optional: eine Eingangsbuchung ohne
abgelegten Beleg wird abgewiesen. Liegt kein Dokument des Lieferanten vor, gehört
ein Eigenbeleg abgelegt. Ein optionaler Verweis hätte das freihändig getippte
Belegfeld wieder eingeführt, das dieses Modell gerade ersetzt.

Die Belegnummer kommt damit aus dem Beleg, nicht mehr aus der Eingabe.
Beim Ausgangsbeleg ist sie die Rechnungsnummer, die die Rechnung ohnehin
schon aus ihrem Nummernkreis gezogen hat – zwei Nummern für dasselbe Dokument
wären eine zu viel.

### Die eigene Ausgangsrechnung

Beim Ausstellen entsteht ein hybrides PDF/A-3b mit dem ZUGFeRD-XML als
zugeordneter Datei, und daraus ein Ausgangsbeleg: das PDF als `original`, das XML
als `structured` und als abgeleitet gekennzeichnet, weil es aus demselben Vorgang
stammt. Die Belegnummer ist die Rechnungsnummer – zwei Nummern für dasselbe
Dokument wären eine zu viel.

Gerendert wird mit Typst, das als WebAssembly im Prozess läuft: kein externes
Programm, kein CGO, nichts mitzuinstallieren. Typst erzeugt PDF/A-3b und hängt die
Datei in einem Schritt an, samt der XMP-Kennzeichnung – das erspart eine
Nachbearbeitung, in der das Factur-X-Extension-Schema von Hand zu schreiben wäre.
Die Beziehung ist `alternative`: PDF und XML sind zwei Darstellungen derselben
Rechnung, und für die Profile BASIC und EN 16931 ist alles andere in Deutschland
nicht rechtsgültig. Typst besteht dabei selbst darauf, dass eine eingebettete
Datei Dateityp und Beschreibung hat – genau die Prüfung, die ZUGFeRD braucht.

GoBD Rz. 76 Abs. 2 erlaubte, auf das archivierte PDF ganz zu verzichten, solange
sich jederzeit ein inhaltlich identisches Mehrstück erzeugen lässt. Es wird
trotzdem abgelegt: ein gespeichertes Dokument ist für den Nutzer greifbarer als
eine Rendering-Zusage.

### Migration

`DocumentHash` und `DocumentPath` am Journaleintrag sind durch einen Verweis auf
den Beleg plus den Beleg-Hash ersetzt; `DocumentNumber` bleibt, es ist das
Belegfeld für den DATEV-Export. Da noch keine produktiven Daten existieren, war
das ein Schnitt und keine Datenmigration.

Der Beleg-Hash tritt an die Stelle des bisherigen Datei-Hashes in der
Kanonisierung der Buchung – die Kette deckt damit unverändert **ein** Feld ab,
nur zeigt es jetzt auf die ganze Dateiliste statt auf eine einzelne Datei. Die
Belegtabelle selbst hat keine eigene Kette: sie hängt über diesen einen Wert an
der des Journals.
