package invoice

import (
	"fmt"
	"strings"

	"github.com/buchfink/buchfink/internal/domain"
)

// Der Abgleich des Hybridformats (RECH-06 K4).
//
// Eine ZUGFeRD-Rechnung ist ein PDF mit einem XML darin, und die beiden müssen
// dasselbe sagen. Weichen sie ab, ist das nach Abschn. 14c.1 UStAE nicht ein
// Schönheitsfehler, sondern möglicherweise ein zweiter Steuerausweis: der
// Empfänger bucht nach dem XML, der Mensch liest das PDF, und die Rechnung, die
// beide vor sich haben, ist nicht dieselbe. Maßgeblich ist der strukturierte
// Teil (§ 14 Abs. 1 Satz 6 UStG) — das PDF ist die Darstellung.
//
// Buchfink erzeugt beide aus demselben Datensatz, und trotzdem wird geprüft.
// Genau darum: die Prüfung ist der Nachweis, dass die Erzeugung getan hat, was
// sie sollte. Ein Fehler in der Vorlage, ein verlorenes Feld im Anhängen des
// XML, eine Bibliothek, die das PDF neu schreibt — jeder davon fiele sonst erst
// dem Empfänger auf. Geprüft wird vor der Ablage: eine Rechnung, die den
// Abgleich nicht besteht, darf nicht in den Belegspeicher.

// VerifyEmbeddedRecord liest den Rechnungsdatensatz aus einem erzeugten
// Dokument zurück und vergleicht ihn mit der Rechnung, aus der er entstanden
// ist.
//
// data ist die PDF-Datei oder das XML selbst — der Leser nimmt beides. Der
// Vergleich umfasst Rechnungsnummer, Rechnungsdatum, Netto, Steuer, Brutto und
// die Zahl der Positionen: die Angaben, aus denen gebucht und gezahlt wird.
func VerifyEmbeddedRecord(data []byte, inv *domain.Invoice) error {
	if inv == nil {
		return fmt.Errorf("ohne Rechnung ist nichts zu vergleichen")
	}
	if len(data) == 0 {
		return fmt.Errorf(
			"die erzeugte Rechnung %s hat keinen Rechnungsdatensatz. Ein Hybridformat ohne "+
				"eingebettetes XML ist keine E-Rechnung (§ 14 Abs. 1 Satz 3 UStG)", inv.InvoiceNumber)
	}
	read, err := NewReader().Read(data)
	if err != nil {
		return fmt.Errorf(
			"der eingebettete Rechnungsdatensatz der Rechnung %s ist nicht lesbar: %w",
			inv.InvoiceNumber, err)
	}

	var problems []string
	compare := func(label, want, got string) {
		if strings.TrimSpace(want) != strings.TrimSpace(got) {
			problems = append(problems, fmt.Sprintf("%s: Datensatz %q, Rechnung %q", label, got, want))
		}
	}
	compareAmount := func(label string, want, got domain.Cents) {
		if want != got {
			problems = append(problems, fmt.Sprintf("%s: Datensatz %s €, Rechnung %s €", label, got, want))
		}
	}

	compare("Rechnungsnummer", inv.InvoiceNumber, read.Number)
	compare("Rechnungsdatum", inv.Date, read.IssueDate)
	compareAmount("Nettobetrag", inv.NetAmount, read.NetAmount)
	compareAmount("Steuerbetrag", inv.TaxAmount, read.TaxAmount)
	compareAmount("Bruttobetrag", inv.GrossAmount, read.GrossAmount)
	if len(read.Positions) != len(inv.Items) {
		problems = append(problems, fmt.Sprintf(
			"Positionszahl: Datensatz %d, Rechnung %d", len(read.Positions), len(inv.Items)))
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf(
		"der eingebettete Rechnungsdatensatz weicht von der Rechnung %s ab: %s. Die Rechnung wurde "+
			"deshalb nicht abgelegt — ein Dokument, dessen lesbarer und dessen strukturierter Teil "+
			"verschiedene Beträge nennen, führt beim Empfänger zu einer anderen Buchung als beim "+
			"Aussteller (§ 14 Abs. 1 Satz 6 UStG, Abschn. 14c.1 UStAE)",
		inv.InvoiceNumber, strings.Join(problems, "; "))
}
