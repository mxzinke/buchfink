# Hilfetexte

[Designkonzept](README.md) · [Dokumentation](../README.md)

## 15. Text

Buchfink erklärt, ohne zuzutexten. Wer täglich damit arbeitet, scrollt am
Erklärsatz beim zwanzigsten Mal vorbei, statt ihn noch zu lesen. Text, der
immer sichtbar ist, obwohl man ihn selten braucht, ist deshalb Lärm, kein
Service.

Die Regel: **Eine Arbeitsansicht enthält keinen Fließtext.** Was länger als ein
Satz ist, gehört in den Detaildialog hinter „Mehr erfahren“.

### 15.1 Textbudget

| Ort | Erlaubt |
|---|---|
| Seitenkopf | Titel plus eine Zeile Kontext, höchstens 60 Zeichen |
| Feldhilfe unter dem Feld | höchstens sechs Wörter, sonst Erklärzeichen |
| Fehlermeldung | zwei Sätze: Ursache, nächster Schritt |
| Leerzustand | Überschrift, ein Satz, eine Aktion |
| Hinweisstreifen | ein Satz |
| Tabellenzelle | kein Erklärtext, nie |

Alles darüber gehört in eine Erklärung nach [Abschnitt 15.2](#152-tooltip-und-detaildialog). Wer beim Schreiben merkt, dass
ein Absatz nötig wäre, hat entweder die Oberfläche zu erklärungsbedürftig gebaut
oder schreibt gerade Dokumentation an der falschen Stelle.

### 15.2 Tooltip und Detaildialog

Die Zielgruppe hat keine Ausbildung in Buchhaltung, Steuerrecht oder Technik.
Die Arbeitsansicht nennt die Aufgabe und die nötigen Angaben. Ein Fehler bleibt
sichtbar und erklärt, wie er behoben werden kann.

| Ort | Inhalt | Auslöser |
|---|---|---|
| Fragezeichen | Ein kurzer Satz in einfachen Worten, ohne Normen oder Links | Hover, Tastaturfokus oder Klick |
| Detaildialog | Verständliche Hintergründe, Beispiele und verlinkte Rechtsgrundlagen | Dezenter Button „Mehr erfahren“ neben dem Fragezeichen |

Jedes Fragezeichen verwendet `Help`. Der Tooltip enthält keine interaktiven
Elemente. Der getrennte Button öffnet einen Dialog; Escape schließt ihn und
gibt den Fokus zurück. Das funktioniert auch innerhalb eines Eingabedialogs.
Eine begonnene Eingabe bleibt dabei erhalten.

`summary` ist der Kurztext. `children` enthält die Details; `onMore` öffnet bei
Bedarf einen eigenen Dialog mit Tabellen oder Beispielen. `Field`, `Section`
und `PageHeader` reichen `helpSummary` und `explain` an dieselbe Hilfe weiter.
Das Zeichen steht hinter der Beschriftung. Zusammengehörige Inhalte haben ein
Fragezeichen, das Layout darf bei langen Beschriftungen umbrechen.

`LegalText.tsx` verlinkt erkannte Gesetzesverweise in Dialogen mit Gesetze im
Internet, GoBD mit dem amtlichen Handbuch und DSGVO-Verweise mit EUR-Lex.
Bei mehreren Paragraphen führt der Link zum zuerst genannten Paragraphen;
weitere können über das Inhaltsverzeichnis der Quelle erreicht werden.
Rechtsprechung und andere Fachquellen werden ausdrücklich verlinkt. Ein Link
belegt nur den zugehörigen Sachverhalt, keine allgemeine Gesetzeskonformität.

### 15.3 Wortwahl

Die [Schreibweise](schreibweise.md) regelt Anrede, Satzbau und verständliche
Erklärungen. Für Oberflächentexte gilt zusätzlich:

- Fachbegriffe bleiben in den Labels stehen. Die Hilfe aus
  [Abschnitt 15.2](#152-tooltip-und-detaildialog) erklärt sie.
- Fehlermeldungen nennen Ursache und nächsten Schritt, etwa:
  "Die Buchung ist nicht ausgeglichen. Soll (1.190,00 €) und Haben (1.000,00 €)
  müssen übereinstimmen."
- Buttons benennen die Handlung, etwa "Buchung festschreiben".
- Bestätigungen benennen die Folge, siehe
  [Abschnitt 8.2](interaktion.md#82-rückgängig-statt-rückfrage).
- Ein Text soll ergänzen, was die Ansicht noch nicht zeigt. Eine Tabelle braucht
  keinen Satz, der ihre Spalten aufzählt.
