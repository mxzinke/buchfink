import React from 'react';
import { Button, Dialog, HelpPopover } from './ui';

/**
 * Die Erklärung zur Vorgesellschaft, an einer Stelle.
 *
 * Sie steht an zwei Orten in der Anwendung — als Hinweisstreifen über der
 * Aufgabenliste und am Gründungsabschnitt der Fristenseite — und ist beide Male
 * dieselbe. Zwei Fassungen desselben Sachverhalts laufen auseinander, sobald
 * eine von ihnen berichtigt wird.
 *
 * Die Stufen folgen §15.2: ein Satz auf der Fläche, ein bis drei Sätze im
 * Popover, alles Weitere hinter „Mehr dazu".
 */

/** Ein bis drei Sätze hinter dem Erklärzeichen. */
export const GruendungExplain: React.FC = () => (
  <>
    Zwischen der Beurkundung und der Eintragung besteht die Vorgesellschaft: Sie ist bereits
    buchführungspflichtig, die Haftungsbeschränkung greift aber noch nicht. Bleibt ihr
    Reinvermögen am Tag der Eintragung hinter dem Stammkapital zurück, schulden die
    Gesellschafter die Differenz.
  </>
);

/**
 * Das Erklärzeichen mit dem Sprung in die dritte Stufe. Es steht hinter dem
 * Satz, nicht am rechten Rand (§15.2).
 */
export const GruendungHelpMark: React.FC<{ onMore: () => void; className?: string }> = ({
  onMore,
  className,
}) => (
  <HelpPopover label="Erklärung zur Vorgesellschaft" onMore={onMore} className={className}>
    <GruendungExplain />
  </HelpPopover>
);

/**
 * Die dritte Stufe: die Rechnung mit ihrem Beispiel.
 *
 * Sie steht hier und nicht im Popover, weil sie eine Tabelle braucht — und weil
 * die Unterbilanzhaftung der eine Punkt der Gründung ist, den Gründer
 * regelmäßig übersehen, bis er sie trifft.
 */
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
        Eine Kapitalgesellschaft entsteht in zwei Schritten. Beim Notar wird der
        Gesellschaftsvertrag beurkundet (§ 2 GmbHG) — von da an gibt es die Vorgesellschaft. Als
        juristische Person entsteht die Gesellschaft erst mit der Eintragung ins Handelsregister
        (§ 11 Abs. 1 GmbHG). Dazwischen liegen Wochen bis Monate, und in dieser Zeit wird bereits
        gebucht, gemietet und eingekauft.
      </p>
      <p>
        Bis zur Eintragung führt die Gesellschaft den Zusatz „i. G." und wer in ihrem Namen
        handelt, haftet persönlich und solidarisch (§ 11 Abs. 2 GmbHG). Beides endet mit der
        Eintragung.
      </p>
      <p>
        Die zweite Haftung endet dort nicht, sie wird dort festgestellt: Bleibt das Reinvermögen am
        Tag der Eintragung hinter dem Stammkapital zurück, schulden die Gesellschafter die
        Differenz — anteilig nach ihren Geschäftsanteilen. Verschärft wird das durch § 248 Abs. 1
        Nr. 1 HGB: Aufwendungen für die Gründung dürfen nicht aktiviert werden. Die Notarrechnung
        ist also sofort Aufwand und mindert das Reinvermögen in voller Höhe.
      </p>
      <div className="rounded-control border border-line bg-surface px-4 py-3">
        <p className="text-caption text-ink-subtle mb-2">
          Eine GmbH mit 25.000 € Stammkapital, davon 12.500 € eingezahlt:
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
        Die Unterbilanzhaftung (Vorbelastungshaftung) ist Richterrecht des BGH und steht in keinem
        Paragrafen. Buchfink rechnet nach der herrschenden Auffassung: maßgeblich ist der Tag der
        Eintragung, und die Gesellschafter haften anteilig nach ihren Geschäftsanteilen.
      </p>
    </div>
  </Dialog>
);
