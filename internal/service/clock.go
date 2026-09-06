package service

import "time"

// todayLocal ist der Kalendertag, mit dem Buchfink ein leeres Datumsfeld
// vorbelegt — in Ortszeit und bewusst nicht in UTC.
//
// Welle 6 hat alle Zeitstempel in Persistenzpfaden auf UTC gestellt (QUE-04):
// wann etwas geschehen ist, wird verglichen, sortiert und über
// Rechnergrenzen hinweg gelesen, und dafür ist eine Ortszeit ohne Zonenangabe
// wertlos. Ein Datumsfeld ist etwas anderes als ein Zeitstempel. Buchungstag,
// Belegdatum, Zahlungstag, Belegeingang und der Tag einer Erledigung sind
// Kalendertage einer Buchführung, die an einem Ort geführt wird; § 146 Abs. 1
// AO beurteilt die zeitgerechte Erfassung nach genau diesen Tagen, und die
// Fristen des Steuerrechts laufen nach deutschem Kalender.
//
// In UTC gerechnet trüge ein Beleg, den jemand in Deutschland um 00:30 Uhr
// ablegt, den Vortag; im Sommer wäre die Grenze 02:00 Uhr. Der Prüflauf
// verglich ihn dann mit einem Stichtag, der einen Tag danebenliegt, und eine
// Zahlung vom Monatsersten fiele in den Vormonat und damit in einen bereits
// festgeschriebenen Zeitraum. Die Ortszeit ist hier also nicht die
// unaufgeräumte, sondern die richtige Antwort — sie steht hier an einer
// Stelle, damit sie eine Entscheidung bleibt und nicht als Versehen aussieht.
func todayLocal() string { return time.Now().Format("2006-01-02") }
