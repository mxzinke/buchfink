import React from 'react';
import type { MonthCloseState, MonthCloseStep } from '../../types';
import type { NavigateFn } from '../../navigation';
import { Button, Notice, Section, SkeletonRows, cn } from '../../components/ui';

/**
 * Die Farbe eines Schritts. Ein gesperrter Prüfbericht hat Befunde, die die
 * Festschreibung verhindern; ein gesperrter späterer Schritt wartet nur auf den
 * davor und bleibt deshalb neutral.
 */
function stepMark(step: MonthCloseStep): string {
  switch (step.state) {
    case 'done':
      return 'bg-positive';
    case 'open':
      return 'bg-attention';
    case 'blocked':
      return step.key === 'check' ? 'bg-negative' : 'bg-ink-faint';
    default:
      return 'bg-ink-faint';
  }
}

const STATE_LABELS: Record<MonthCloseStep['state'], string> = {
  done: 'erledigt',
  open: 'offen',
  blocked: 'wartet',
  not_applicable: 'entfällt',
};

export interface MonthCloseSectionProps {
  /** Der Monat („JJJJ-MM“), dessen Abschluss ansteht. */
  month: string;
  state: MonthCloseState | null;
  /** Warum es für den Monat keinen Stand gibt; das Backend nennt den Grund. */
  error: string;
  onNavigate: NavigateFn;
}

/**
 * Der Monatsabschluss in drei Schritten nebeneinander: Prüfbericht,
 * Festschreiben, Voranmeldung. Gearbeitet wird unter Abschlüsse → Monat.
 */
export const MonthCloseSection: React.FC<MonthCloseSectionProps> = ({
  month,
  state,
  error,
  onNavigate,
}) => {
  // Ein entfallender Schritt, etwa die Voranmeldung in einem Monat ohne
  // Abgabe, steht nicht da und zählt nicht mit.
  const steps = (state?.steps ?? []).filter((step) => step.state !== 'not_applicable');
  const done = steps.filter((step) => step.state === 'done').length;
  const applicable = steps.length;

  let context = 'Prüfbericht, Festschreibung, Voranmeldung';
  if (state) {
    context = state.committed
      ? `${state.label} · festgeschrieben`
      : `${state.label} · ${done} von ${applicable} Schritten erledigt`;
  }

  return (
    <Section
      title="Monatsabschluss"
      context={context}
      helpSummary="Ein Monat ist abgeschlossen, wenn er festgeschrieben und die Voranmeldung bestätigt ist."
      explain={
        <>
          Der Prüfbericht zeigt, was vor der Festschreibung zu klären ist. Danach schreiben Sie den
          Monat fest; ab dann nimmt er keine Buchung mehr auf. Zuletzt geben Sie die Voranmeldung in
          Mein ELSTER ab und tragen das Transferticket ein.
        </>
      }
      action={
        month && (
          <Button
            variant="secondary"
            onClick={() =>
              onNavigate({ area: 'closings', tab: 'month', id: `festschreibung.${month}` })
            }
          >
            Zum Monatsabschluss
          </Button>
        )
      }
    >
      {!state && !error ? (
        <SkeletonRows rows={2} />
      ) : !state ? (
        <Notice text={error} />
      ) : (
        <ol className="grid grid-cols-1 gap-6 md:grid-cols-3">
          {steps.map((step) => (
            <li key={step.key} className="flex min-w-0 items-start gap-3">
              <span
                className={cn('mark-diamond mt-1.5 shrink-0', stepMark(step))}
                aria-hidden="true"
              />
              <span className="min-w-0">
                <span className="block text-body text-ink">{step.title}</span>
                <span className="block text-caption text-ink-subtle">
                  {`Schritt ${step.number} · ${STATE_LABELS[step.state]}`}
                </span>
                <span className="mt-1 block text-caption text-ink-muted">{step.note}</span>
              </span>
            </li>
          ))}
        </ol>
      )}
    </Section>
  );
};
