import React, { useCallback, useEffect, useState } from 'react';
import { Api } from '../../services/api';
import type { AgentRules } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { useLiveReload } from '../../live';
import { formatCents, formatCentsPlain, parseCents } from '../../utils/formatters';
import { Button, Field, FormGrid, Input, Notice, Section, SkeletonRows, toast } from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

interface Draft {
  maxGross: string;
  minBookings: string;
}

function draftOf(rules: AgentRules): Draft {
  return {
    maxGross: rules.stored.directMaxGross > 0 ? formatCentsPlain(rules.stored.directMaxGross) : '',
    minBookings: rules.stored.directMinBookings > 0 ? String(rules.stored.directMinBookings) : '',
  };
}

/**
 * Wann Claude eine E-Rechnung ohne Rückfrage bucht: unter der Betragsgrenze
 * und bei einem Lieferanten mit bestätigter Standardgruppe, der schon so oft
 * mit ihr gebucht wurde. Leere Felder heißen Voreinstellung; die zeigt das
 * Feld als Platzhalter. Nur der Mensch ändert die Regeln.
 */
export const AgentRulesPanel: React.FC<{ id?: string }> = ({ id }) => {
  const writeLock = useWriteLock();
  const [rules, setRules] = useState<AgentRules | null>(null);
  const [draft, setDraft] = useState<Draft>({ maxGross: '', minBookings: '' });
  const [errors, setErrors] = useState<Partial<Record<keyof Draft, string>>>({});
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async (resetDraft: boolean) => {
    try {
      const next = await Api.getAgentRules();
      setRules(next);
      if (resetDraft) setDraft(draftOf(next));
      setError('');
    } catch (e) {
      setError(`Die Regeln ließen sich nicht lesen. ${message(e)}`);
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  // Eine begonnene Eingabe bleibt stehen; nur der gespeicherte Stand kommt neu.
  useLiveReload(() => load(false), ['stammdaten']);

  async function save() {
    if (!rules) return;
    const next: Partial<Record<keyof Draft, string>> = {};
    const gross = draft.maxGross.trim() === '' ? 0 : parseCents(draft.maxGross);
    if (gross === null || gross < 0) next.maxGross = 'Ein Betrag wie 1.000,00.';
    const bookings = draft.minBookings.trim() === '' ? 0 : Number(draft.minBookings);
    if (!Number.isInteger(bookings) || bookings < 0) next.minBookings = 'Eine ganze Zahl wie 3.';
    setErrors(next);
    if (Object.keys(next).length > 0) return;
    setSaving(true);
    setError('');
    try {
      const fresh = await Api.getAgentRules();
      const saved = await Api.saveAgentRules({
        ...fresh.stored,
        directMaxGross: gross ?? 0,
        directMinBookings: bookings,
      });
      setRules(saved);
      setDraft(draftOf(saved));
      toast.success('Regeln gespeichert.');
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(false);
    }
  }

  const dirty = rules ? JSON.stringify(draft) !== JSON.stringify(draftOf(rules)) : false;

  return (
    <Section
      id={id}
      title="Buchen ohne Rückfrage"
      context={
        rules
          ? `Unter ${formatCents(rules.directMaxGross)} brutto, ab ${rules.directMinBookings} früheren Buchungen`
          : 'Wann Claude E-Rechnungen direkt bucht'
      }
      helpSummary="Unter diesen Grenzen bucht Claude E-Rechnungen bekannter Lieferanten ohne Rückfrage."
      explain={
        <>
          Claude bucht eine E-Rechnung nur dann ohne Rückfrage, wenn der Lieferant eine Standardgruppe
          hat, die Sie bestätigt haben, schon so oft mit ihr gebucht wurde und der Bruttobetrag unter
          der Grenze liegt. Fotos und PDF ohne E-Rechnung legt Claude Ihnen immer vor. Eine falsche
          Buchung hebt ein Storno auf, bis der Monat festgeschrieben ist. Ein leeres Feld steht für die
          Voreinstellung.
        </>
      }
    >
      {error && <Notice tone="negative" className="mb-5" text={error} />}
      {!rules ? (
        !error && <SkeletonRows rows={2} />
      ) : (
        <>
          <FormGrid>
            <Field label="Betragsgrenze" hint="Brutto je E-Rechnung" error={errors.maxGross}>
              <Input
                align="right"
                inputMode="decimal"
                placeholder={rules.stored.directMaxGross > 0 ? '' : formatCentsPlain(rules.directMaxGross)}
                value={draft.maxGross}
                disabled={writeLock.locked}
                onChange={(e) => setDraft((prev) => ({ ...prev, maxGross: e.target.value }))}
                onBlur={() => {
                  const cents = parseCents(draft.maxGross);
                  if (cents !== null && cents > 0) setDraft((prev) => ({ ...prev, maxGross: formatCentsPlain(cents) }));
                }}
              />
            </Field>
            <Field label="Frühere Buchungen" hint="Mit der bestätigten Standardgruppe" error={errors.minBookings}>
              <Input
                type="number"
                min={1}
                align="right"
                placeholder={rules.stored.directMinBookings > 0 ? '' : String(rules.directMinBookings)}
                value={draft.minBookings}
                disabled={writeLock.locked}
                onChange={(e) => setDraft((prev) => ({ ...prev, minBookings: e.target.value }))}
              />
            </Field>
          </FormGrid>
          <div className="mt-5">
            <Button
              variant="secondary"
              loading={saving}
              disabled={writeLock.locked || !dirty}
              title={writeLock.hint}
              onClick={() => void save()}
            >
              Regeln speichern
            </Button>
          </div>
        </>
      )}
    </Section>
  );
};
