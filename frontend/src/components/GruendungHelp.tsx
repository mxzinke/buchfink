import React from 'react';
import { Button, Dialog, Help } from './ui';
import { SourceLink } from './ui/LegalText';

/** Gemeinsame Gründungserklärung für Aufgaben, Fristen und Gründungsabschnitt. */
export const GruendungExplain: React.FC = () => (
  <>
    Zwischen Beurkundung und Eintragung besteht die Vorgesellschaft. Wer vor der
    Eintragung für eine GmbH oder UG handelt, kann persönlich haften. Die Grundlage
    nennt{' '}
    <SourceLink href="https://www.gesetze-im-internet.de/gmbhg/__11.html">§ 11 GmbHG</SourceLink>.
    Reicht das Vermögen bei der Eintragung nicht zur Deckung des Stammkapitals,
    können weitere Zahlungspflichten entstehen.
  </>
);

/** Kurzer Tooltip und Verweis auf den gemeinsamen Gründungsdialog. */
export const GruendungHelpMark: React.FC<{ onMore: () => void; className?: string }> = ({
  onMore,
  className,
}) => (
  <Help summary="Bis zur Eintragung können für die Gründer persönliche Haftungsrisiken bestehen." label="Erklärung zur Vorgesellschaft" onMore={onMore} className={className}>
    <GruendungExplain />
  </Help>
);

/** Erläutert die Gründung mit Beispiel und Rechtsquellen. */
export const GruendungHelpDialog: React.FC<{ open: boolean; onClose: () => void }> = ({
  open,
  onClose,
}) => (
  <Dialog
    open={open}
    onOpenChange={(next) => !next && onClose()}
    title="Vorgesellschaft und Unterbilanzhaftung"
    footer={
      <Button variant="secondary" onClick={onClose}>
        Schließen
      </Button>
    }
  >
    <div className="text-body text-ink-muted space-y-4">
      <p>
        Bei GmbH und UG beginnt die Gründung mit der notariellen Beurkundung des
        Gesellschaftsvertrags (§ 2 GmbHG). Bis zur Eintragung ins Handelsregister
        besteht eine Vorgesellschaft. Die GmbH als solche entsteht erst durch die
        Eintragung (§ 11 Abs. 1 GmbHG).
      </p>
      <p>
        Wer vor der Eintragung im Namen der Gesellschaft handelt, kann persönlich
        haften (§ 11 Abs. 2 GmbHG). Für eine AG gelten eigene Vorschriften,
        insbesondere §§ 29 und 41 AktG.
      </p>
      <p>
        Reicht das Vermögen nach Abzug der Schulden am Eintragungstag nicht aus,
        um das Stammkapital zu decken, kann eine zusätzliche Zahlungspflicht der
        Gesellschafter entstehen. Das heißt Unterbilanzhaftung.
        Gründungskosten sind Aufwand und dürfen nicht als Vermögen angesetzt werden
        (§ 248 Abs. 1 Nr. 1 HGB).
      </p>
      <div className="rounded-control border border-line bg-surface px-4 py-3">
        <p className="text-caption text-ink-subtle mb-2">
          Vereinfachtes Beispiel ohne satzungsmäßig übernommenen Gründungsaufwand:
        </p>
        <dl className="space-y-1.5">
          <div className="flex justify-between gap-4">
            <dt>Bank und offene Einlageforderung</dt>
            <dd className="num shrink-0">25.000,00 €</dd>
          </div>
          <div className="flex justify-between gap-4">
            <dt>Notar- und Gerichtskosten, aus der Bank bezahlt</dt>
            <dd className="num shrink-0">− 3.000,00 €</dd>
          </div>
          <div className="flex justify-between gap-4 pt-1.5 border-t border-line font-semibold text-ink">
            <dt>Unterbilanz am Eintragungstag</dt>
            <dd className="num shrink-0">3.000,00 €</dd>
          </div>
        </dl>
      </div>
      <p>
        Die noch ausstehende Einlage zählt dabei zum Reinvermögen: Sie wird geschuldet, aber als
        Einlage und nicht als Vorbelastungshaftung. Beide Ansprüche stehen nebeneinander.
      </p>
      <p>
        Ein Gründungsaufwand, den der Gesellschaftsvertrag der Gesellschaft betragsmäßig auferlegt,
        deckt die Unterdeckung, soweit er reicht. Ohne eine solche Klausel schlägt der gesamte
        Gründungsaufwand durch.
      </p>
      <p className="text-caption text-ink-subtle">
        Grundlage der Unterbilanzhaftung ist die Rechtsprechung. Siehe etwa{' '}
        <SourceLink href="https://juris.bundesgerichtshof.de/cgi-bin/rechtsprechung/document.py?Art=en&amp;Blank=1.pdf&amp;Datum=2006-1-16&amp;Gericht=bgh&amp;anz=6&amp;nr=35702&amp;pos=3">
          BGH, Urteil vom 16. Januar 2006, II ZR 65/04
        </SourceLink>. Die Berechnung in Buchfink verteilt eine verbleibende Unterdeckung
        nach den Geschäftsanteilen; die konkrete Haftung hängt vom Einzelfall ab.
      </p>
    </div>
  </Dialog>
);
