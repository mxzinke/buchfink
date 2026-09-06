# Schreibweise

Buchfink erklärt Buchhaltung Menschen, die sie nicht gelernt haben. Das gelingt
nur mit Sätzen, die eine Sache benennen und dann aufhören. Diese Seite sammelt
die Wendungen, die sich im Projekt eingeschlichen haben, und sagt, was
stattdessen dasteht.

Die Regeln gelten für alles, was jemand liest: Oberflächentexte,
Fehlermeldungen, Projektseite, README, Fachdokumente und Kommentare im Quelltext.

## Allzweckverben

Ein Verb, das auf alles passt, sagt nichts. Die linke Spalte ist im Projekt
entstanden, die rechte nennt den Vorgang.

| Statt | Besser |
|---|---|
| Die Rechnung **trägt** den Typcode 386. | Die Rechnung hat den Typcode 386. |
| Jedes Geschenk **trägt** seinen Empfänger. | Zu jedem Geschenk gehört ein Empfänger. |
| Belege **tragen** ein Datum. | Jeder Beleg hat ein Datum. |
| Die Buchung **trägt** die Bearbeiterkennung. | Die Buchung speichert, wer sie erfasst hat. |
| Der Satz **hängt am** Anleger. | Der Satz richtet sich nach dem Anleger. |
| Die Datei **geht hinaus**. | Buchfink schreibt die Datei. |
| Die Einreichung **läuft über** Mein ELSTER. | Die Einreichung geschieht in Mein ELSTER. |
| Die Regel **steht** im Prüflauf. | Der Prüflauf enthält die Regel. |

`tragen` bleibt richtig, wo es aus der Buchhaltung kommt: ein Konto trägt seinen
Saldo im Soll, ein Gesellschafter trägt einen Verlust. `stehen` bleibt richtig
für einen Ort: der Betrag steht in Zeile 12, die Norm steht im Erklärzeichen.

## Satzbau

**Sagen, was gilt, ohne den Gegensatz.** Die Wendungen „nicht X, sondern Y" und
„kein X, sondern Y" zwingen den Leser durch eine falsche Aussage, bevor die
richtige kommt.

> Die Zahlung wird nicht gebucht, sondern nur vorgeschlagen.
> → Buchfink schlägt die Zuordnung vor. Gebucht wird nach der Bestätigung.

> Der Prüfermodus ist kein Bedienelement, sondern eine Sperre.
> → Der Prüfermodus sperrt jeden schreibenden Weg.

Der Gegensatz bleibt, wo er zwei benannte Dinge auseinanderhält, die man
tatsächlich verwechselt. Das ist in der Buchhaltung häufig und dann die
eigentliche Auskunft:

> Die Gutschrift ist keine Rechnung, sondern eine Entgeltminderung nach
> § 17 UStG.

Die Probe: Steht links etwas, das ein Leser wirklich annehmen würde? Dann bleibt
der Satz. Steht links nur eine Verneinung, die niemand erwartet hat, streiche
sie und sage die rechte Seite.

**Handelnden nennen.** Passiv verschweigt, wer etwas tut.

> Die Buchung wird geprüft. → Der Prüflauf prüft die Buchung.

**Ein Gedanke je Satz.** Wer beim Lesen zurückspringen muss, liest einen Satz,
der zwei Sätze sein sollte.

**Verben statt Hauptwörtern.** „Die Durchführung der Prüfung erfolgt" heißt
„Buchfink prüft".

**Füllwörter weg.** „genau die", „eben jener", „durchaus", „bereits", „jeweils"
tragen selten etwas bei. `Genau` bleibt, wo es einen Wert abgrenzt: genau vier
Klicks.

**Satzanfänge abwechseln.** „Wer …, …" ist ein guter Satzbau und wird schlecht,
sobald drei davon hintereinander stehen.

## Was in einer Erklärung steht

Eine Erklärung in der Oberfläche beantwortet eine Frage des Anwenders: Was
passiert, wenn ich das anklicke? Wozu brauche ich das? Was muss ich danach tun?
Sie beschreibt nicht, wie Buchfink es intern löst.

| Statt | Besser |
|---|---|
| Buchfink weist jede schreibende Bedienung ab. | Niemand kann mehr buchen oder etwas ändern. |
| Der Export schreibt CSV, index.xml nach dem Beschreibungsstandard und export.json mit einer Prüfsumme je Datei. | Buchfink legt die Buchführung so ab, dass die Software des Finanzamts sie lesen kann. |
| Die Prüfung spielt die Sicherung in einen Temporärordner zurück und räumt ihn ab. | Die Prüfung sagt Ihnen, ob die Sicherung vollständig ist. An Ihren Daten ändert sich nichts. |
| Der Lauf vergleicht die Datei mit der beim Ablegen gebildeten Prüfsumme. | Der Lauf zeigt, ob jede Datei noch da und unverändert ist. |

Dateinamen, Formate, Datenstrukturen und der Grund für eine Bauweise gehören in
den Quelltext oder in `docs/`, nicht vor den Anwender. Wo der Anwender das
Format wirklich braucht — weil er einen Ordner übergibt oder eine Datei sucht —,
steht es im Dialog hinter „Mehr dazu" und nicht im Popover.

Die Stufen aus §15.2 des Designkonzepts: eine Zeile Kontext in der Ansicht, ein
bis drei Sätze im Erklärzeichen, alles Weitere im Dialog. Wer in drei Sätzen
nicht fertig wird, hat entweder zu viel erklärt oder das Falsche.

## Wörter, die nichts sagen

Diese Wendungen beschreiben ein Gefühl statt einer Sache. Ersetze sie durch das,
was der Leser tun oder wissen soll, oder streiche sie.

- „an ihrer Grenze endet" → sagen, was fehlt
- „ist bestellt", „ist gesetzt" für erledigte Arbeit → sagen, was gebaut wurde
- „das Rückgrat", „der Dreh- und Angelpunkt", „von Grund auf"
- „sauber", „elegant", „modern", „leistungsfähig"
- „nahtlos", „mühelos", „intuitiv"

Ein Satz, der unverändert in der Dokumentation eines anderen Programms stehen
könnte, sagt über Buchfink nichts. Streiche ihn.

## Anrede und Ton

Sie-Form oder anredefrei, nie Du. Keine Ausrufezeichen. Keine Emoji in
Überschriften und Listen. Zahlen und Normen statt Steigerungen: „vier Klicks"
statt „sehr schnell", „§ 288 Abs. 5 BGB" statt „gesetzlich vorgeschrieben".
