import React, { useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Api } from '../../services/api';
import type { ContributionKind, Foundation, FoundationRules } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatCentsPlain, parseCents } from '../../utils/formatters';
import { Button, Dialog, Field, FormGrid, Input, Notice, Select, cn, toast } from '../../components/ui';

interface Row {
  key: number;
  name: string;
  share: string;
  paid: string;
  kind: ContributionKind;
}

function rowsOf(foundation: Foundation | undefined, fallback: { name: string; shareCapital: number }[]): Row[] {
  if (foundation?.shareholders?.length) {
    return foundation.shareholders.map((s, i) => ({
      key: i,
      name: s.name,
      share: formatCentsPlain(s.shareCapital),
      paid: s.paidIn ? formatCentsPlain(s.paidIn) : '',
      kind: s.kind,
    }));
  }
  if (fallback.length) {
    return fallback.map((s, i) => ({
      key: i,
      name: s.name,
      share: s.shareCapital ? formatCentsPlain(s.shareCapital) : '',
      paid: '',
      kind: 'cash',
    }));
  }
  return [{ key: 0, name: '', share: '', paid: '', kind: 'cash' }];
}

/**
 * Die Gründung laut Gesellschaftsvertrag: Beurkundung, Stammkapital, wer welchen
 * Anteil übernommen und was eingezahlt hat. Daraus rechnet Buchfink die
 * Mindesteinzahlung vor der Anmeldung, die Eröffnungsbilanz und die
 * Vorbelastung bis zur Eintragung.
 */
export const FoundationDialog: React.FC<{
  foundation?: Foundation;
  rules: FoundationRules;
  /** Gründungsdatum der Stammdaten; Kapital und Gesellschafter als Vorbelegung ohne erfasste Gründung. */
  defaults: { foundedOn: string; shareCapital: number; shareholders: { name: string; shareCapital: number }[] };
  onClose: () => void;
  onSaved: () => void;
}> = ({ foundation, rules, defaults, onClose, onSaved }) => {
  const writeLock = useWriteLock();
  const [notarizedOn, setNotarizedOn] = useState(defaults.foundedOn);
  const [capital, setCapital] = useState(() => {
    const cents = foundation?.shareCapital ?? defaults.shareCapital;
    return cents ? formatCentsPlain(cents) : '';
  });
  const [costCap, setCostCap] = useState(foundation?.foundationCostCap ? formatCentsPlain(foundation.foundationCostCap) : '');
  const [rows, setRows] = useState<Row[]>(() => rowsOf(foundation, defaults.shareholders));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const capitalCents = parseCents(capital) ?? 0;
  const subscribed = rows.reduce((sum, r) => sum + (parseCents(r.share) ?? 0), 0);
  const paid = rows.reduce((sum, r) => sum + (parseCents(r.paid) ?? 0), 0);
  const mismatch = capitalCents > 0 && subscribed !== capitalCents;
  const update = (key: number, next: Partial<Row>) =>
    setRows((prev) => prev.map((r) => (r.key === key ? { ...r, ...next } : r)));

  async function save() {
    setBusy(true);
    setError('');
    try {
      await Api.saveFoundation(notarizedOn, {
        ...(foundation ?? {}),
        registeredOn: foundation?.registeredOn ?? '',
        registerCourt: foundation?.registerCourt ?? '',
        registerNumber: foundation?.registerNumber ?? '',
        shareCapital: capitalCents,
        foundationCostCap: parseCents(costCap) ?? 0,
        shareholders: rows.map((r) => ({
          id: 0,
          foundationId: foundation?.id ?? 0,
          name: r.name.trim(),
          shareCapital: parseCents(r.share) ?? 0,
          paidIn: parseCents(r.paid) ?? 0,
          kind: rules.cashOnly ? 'cash' : r.kind,
        })),
      });
      toast.success('Gründung gespeichert.');
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={foundation ? 'Gründung bearbeiten' : 'Gründung erfassen'}
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void save()}
          >
            Gründung speichern
          </Button>
        </>
      }
    >
      <FormGrid>
        <Field label="Beurkundet am" hint="Beginn der Bücher">
          <Input type="date" value={notarizedOn} onChange={(e) => setNotarizedOn(e.target.value)} />
        </Field>
        <Field label="Stammkapital laut Satzung" hint={`Mindestens ${formatCents(rules.minShareCapital)}`}>
          <Input
            align="right"
            value={capital}
            placeholder="25.000,00"
            onChange={(e) => setCapital(e.target.value)}
            onBlur={() => {
              const cents = parseCents(capital);
              if (cents) setCapital(formatCentsPlain(cents));
            }}
          />
        </Field>
        <Field
          label="Gründungsaufwand laut Satzung"
          optional
          helpSummary="Was die Satzung der Gesellschaft an Gründungskosten auferlegt."
          explain="Bis zu diesem Betrag trägt die Gesellschaft die Gründungskosten; er mindert die Vorbelastung der Gesellschafter."
        >
          <Input align="right" value={costCap} placeholder="2.500,00" onChange={(e) => setCostCap(e.target.value)} />
        </Field>
      </FormGrid>

      <div className="mt-6">
        <div className="grid grid-cols-[minmax(0,1fr)_8rem_8rem_7rem_2.25rem] gap-2 text-label text-ink-muted pb-1">
          <span>Gesellschafter</span>
          <span className="text-right">Anteil</span>
          <span className="text-right">Eingezahlt</span>
          <span>{rules.cashOnly ? '' : 'Einlage'}</span>
          <span />
        </div>
        <ul className="flex flex-col gap-2">
          {rows.map((row, index) => (
            <li key={row.key} className="grid grid-cols-[minmax(0,1fr)_8rem_8rem_7rem_2.25rem] gap-2 items-center">
              <Input
                aria-label={`Name des ${index + 1}. Gesellschafters`}
                value={row.name}
                placeholder="Anna Bauer"
                onChange={(e) => update(row.key, { name: e.target.value })}
              />
              <Input
                aria-label="Übernommener Anteil"
                align="right"
                value={row.share}
                placeholder="12.500,00"
                onChange={(e) => update(row.key, { share: e.target.value })}
              />
              <Input
                aria-label="Eingezahlt"
                align="right"
                value={row.paid}
                placeholder="0,00"
                onChange={(e) => update(row.key, { paid: e.target.value })}
              />
              {rules.cashOnly ? (
                <span />
              ) : (
                <Select
                  aria-label="Art der Einlage"
                  items={[
                    { value: 'cash', label: 'Bar' },
                    { value: 'kind', label: 'Sache' },
                  ]}
                  value={row.kind}
                  onValueChange={(kind) => update(row.key, { kind: kind as ContributionKind })}
                />
              )}
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                disabled={rows.length === 1}
                title="Gesellschafter entfernen"
                aria-label={`${row.name || `Gesellschafter ${index + 1}`} entfernen`}
                onClick={() => setRows((prev) => prev.filter((r) => r.key !== row.key))}
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </Button>
            </li>
          ))}
        </ul>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <Button
            variant="quiet"
            size="sm"
            className="-ml-2.5"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() =>
              setRows((prev) => [
                ...prev,
                { key: Math.max(0, ...prev.map((r) => r.key)) + 1, name: '', share: '', paid: '', kind: 'cash' },
              ])
            }
          >
            Gesellschafter hinzufügen
          </Button>
          <span className={cn('text-caption num', mismatch ? 'text-negative-text' : 'text-ink-subtle')}>
            {formatCents(subscribed)} übernommen · {formatCents(paid)} eingezahlt
          </span>
        </div>
        {mismatch && (
          <p className="mt-2 text-caption text-negative-text">
            Die Anteile ergeben {formatCents(subscribed)}, das Stammkapital beträgt {formatCents(capitalCents)}.
          </p>
        )}
      </div>
      {error && <Notice tone="negative" className="mt-5" text={error} />}
    </Dialog>
  );
};
