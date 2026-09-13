import React, { useEffect, useState } from 'react';
import { Api } from '../services/api';
import type { FiscalYearCandidate, FiscalYearCandidates } from '../types';
import { formatDate } from '../utils/formatters';
import { Button, Dialog, Notice, RadioGroup, SkeletonRows } from './ui';

export type FiscalYearDirection = 'next' | 'previous';

export interface FiscalYearDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (year: number) => void;
}

/**
 * Ob sich das kommende oder das vergangene Geschäftsjahr anlegen lässt, legt
 * das Backend fest. Der Dialog zeigt die Gründe an, aus denen das nicht geht.
 */
export const FiscalYearDialog: React.FC<FiscalYearDialogProps> = ({ open, onOpenChange, onCreate }) => {
  const [candidates, setCandidates] = useState<FiscalYearCandidates | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [direction, setDirection] = useState<FiscalYearDirection>('next');

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setCandidates(null);
    setError(null);
    Api.getFiscalYearCandidates()
      .then((c) => {
        if (cancelled) return;
        setCandidates(c);
        setDirection(c.next.blocked && !c.previous.blocked ? 'previous' : 'next');
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [open]);

  const target = candidates?.[direction];
  const creatable = !!target && !target.blocked;
  const period = (c: FiscalYearCandidate) => `${formatDate(c.startDate)} bis ${formatDate(c.endDate)}`;

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Geschäftsjahr anlegen"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            {creatable ? 'Abbrechen' : 'Schließen'}
          </Button>
          {creatable && (
            <Button
              variant="primary"
              onClick={() => {
                onOpenChange(false);
                onCreate(target.year);
              }}
            >
              Geschäftsjahr {target.year} anlegen
            </Button>
          )}
        </>
      }
    >
      {error && <Notice tone="negative" text={error} />}
      {!error && !candidates && <SkeletonRows />}
      {candidates && (
        <>
          <RadioGroup<FiscalYearDirection>
            value={direction}
            onValueChange={setDirection}
            options={[
              {
                value: 'next',
                label: `Kommendes Geschäftsjahr ${candidates.next.year}`,
                hint: candidates.next.blocked ?? period(candidates.next),
                disabled: !!candidates.next.blocked,
              },
              {
                value: 'previous',
                label: `Vergangenes Geschäftsjahr ${candidates.previous.year}`,
                hint: candidates.previous.blocked ?? period(candidates.previous),
                disabled: !!candidates.previous.blocked,
              },
            ]}
          />

          {creatable ? (
            <>
              <p className="text-body text-ink-muted mt-5">
                {direction === 'next'
                  ? `Buchfink legt das Geschäftsjahr ${target.year} an und schaltet die Ansicht darauf um.`
                  : `Buchfink legt das Geschäftsjahr ${target.year} vor dem bisher ersten Jahr an und schaltet die Ansicht darauf um. Der Saldenvortrag danach ist neu zu buchen.`}
              </p>
              <Notice className="mt-4" text="Ein angelegtes Geschäftsjahr lässt sich nicht wieder entfernen." />
            </>
          ) : (
            <Notice className="mt-5" text="Derzeit lässt sich kein Geschäftsjahr anlegen. Die Gründe stehen bei den beiden Jahren." />
          )}
        </>
      )}
    </Dialog>
  );
};
