# Ziele und nächste Schritte

Stand: 11. September 2026. Diese Seite beschreibt geplante Arbeit.
Was bereits implementiert ist, steht im [Umsetzungsstand](stand-der-umsetzung.md).

Buchfink soll kleinen Unternehmen, Gründern und Holdings ermöglichen, ihre
Buchhaltung frei und selbstständig zu erledigen. Dazu entwickeln wir gemeinsam
eine gesetzeskonforme Buchhaltungssoftware, die auch Menschen ohne fachliche
Ausbildung bedienen können. Der Quellcode ist öffentlich; Beiträge sind auch
als Rückmeldung, verständliche Erklärung oder fachliche Prüfung willkommen.

## Einfache Jahresabschlüsse selbst erstellen

Langfristig soll Buchfink durch einen einfachen Jahresabschluss bis zu den
nötigen Einreichungen führen. Viele kleine Unternehmen beauftragen damit heute
eine Steuerkanzlei. Buchfink soll ihnen ermöglichen, diese Arbeit selbst zu
erledigen und steuerliche Beratung gezielt für Gestaltung und konkrete Fragen
in Anspruch zu nehmen.

Dafür stehen insbesondere diese Arbeiten aus:

- Die E-Bilanz-Zuordnung gegen die amtliche Taxonomie prüfen und die erzeugten
  Dateien mit einem geeigneten Übermittlungsweg erproben.
- Unterzeichneten Abschluss, Feststellungsbeschluss und Einreichungsnachweise
  in den Abschlussablauf einbinden.
- Offenlegung und gegebenenfalls Hinterlegung für die unterstützten
  Unternehmensgrößen abbilden.
- Den Umfang von Anhang und weiteren Abschlussangaben an die Größenklasse
  anpassen. Die vorhandene Berechnung der Klasse allein reicht dafür nicht.
- Vollständige Geschäftsjahre mit Menschen aus der Zielgruppe und fachkundigen
  Prüfern erproben. Ein automatischer Test belegt keine rechtliche Vollständigkeit.

## Verständliche Bedienung

Fragezeichen erklären einen Begriff oder die nächste Handlung in einem kurzen
Tooltip. Über „Mehr erfahren“ lassen sich verständliche Details mit verlinkten
Rechtsgrundlagen öffnen. Rückmeldungen aus der Nutzung sollen zeigen, welche
Begriffe und Schritte weiterhin unklar sind.

## Lokale Anbindung an Chatbots

Geplant ist ein lokaler MCP-Server, damit sich Buchfink mit dem Chatbot der
Wahl verbinden lässt. Eine solche Schnittstelle ist noch nicht implementiert.
Funktionsumfang, Zugriffsrechte und Freigaben für Änderungen sind noch zu
entwerfen. Die lokale Schnittstelle allein sagt nicht aus, wo ein angeschlossener
Chatbot die von ihm abgerufenen Daten verarbeitet.

## Einfach installieren

Fertige, signierte Installationsdateien sollen die Einrichtung ohne
Entwicklungswerkzeuge ermöglichen. Aktuell wird die Vorschau aus dem Quellcode
gebaut. Es gibt noch keinen verbindlichen Termin für Version 1 oder MCP.

Die fachlichen Einzelanforderungen stehen im
[Anforderungskatalog](anforderungskatalog.md). Ein DATEV-Export bleibt ein
[offener Entwurf](anforderung-datev-export.md), ohne Umsetzungszusage.
