import React, { useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import type { CompanySettings, FoundationRules } from '../types';
import { formatCents, formatCentsPlain, parseCents } from '../utils/formatters';
import { Button, Field, FormGrid, Input, Section, cn } from './ui';

interface ShareholderDraft {
  key: number;
  name: string;
  amountText: string;
}

interface CompanyCapitalSectionProps {
  settings: CompanySettings;
  rules: FoundationRules;
  patch: (next: Partial<CompanySettings>) => void;
}

/**
 * Urkunde, Stamm- oder Grundkapital und Gesellschafterliste einer
 * Kapitalgesellschaft, jeweils im aktuellen Stand.
 */
export const CompanyCapitalSection: React.FC<CompanyCapitalSectionProps> = ({
  settings,
  rules,
  patch,
}) => {
  const isAG = rules.legalForm === 'AG';
  const capitalLabel = isAG ? 'Grundkapital' : 'Stammkapital';
  const [capitalText, setCapitalText] = useState(
    settings.shareCapital ? formatCentsPlain(settings.shareCapital) : '',
  );
  const [nextKey, setNextKey] = useState((settings.shareholders ?? []).length);
  const [rows, setRows] = useState<ShareholderDraft[]>(() =>
    (settings.shareholders ?? []).map((sh, index) => ({
      key: index,
      name: sh.name,
      amountText: formatCentsPlain(sh.shareCapital),
    })),
  );

  function update(next: ShareholderDraft[]) {
    setRows(next);
    patch({
      shareholders: next.map((row) => ({
        name: row.name,
        shareCapital: parseCents(row.amountText) ?? 0,
      })),
    });
  }

  const subscribed = rows.reduce((sum, row) => sum + (parseCents(row.amountText) ?? 0), 0);
  const capitalMismatch = rows.length > 0 && settings.shareCapital > 0 && subscribed !== settings.shareCapital;
  const belowMinimum = settings.shareCapital > 0 && settings.shareCapital < rules.minShareCapital;

  return (
    <Section
      helpSummary="Halten Sie Urkunde, Kapital und Gesellschafter Ihrer Gesellschaft fest."
      title={isAG ? 'Satzung und Grundkapital' : 'Gesellschaftsvertrag und Gesellschafter'}
      explain={
        <>
          Maßgeblich ist der aktuelle Stand laut Handelsregister und Gesellschafterliste (§ 40
          GmbHG). Die Nennbeträge der Geschäftsanteile müssen zusammen das Stammkapital ergeben (§ 5
          Abs. 3 Satz 2 GmbHG). Wird die Gründung mit Buchfink begleitet, übernimmt Buchfink die
          Angaben von dort, solange Sie sie hier nicht abweichend pflegen.
        </>
      }
    >
      <FormGrid>
        <Field label="Notarin oder Notar" optional>
          <Input value={settings.notary} onChange={(e) => patch({ notary: e.target.value })} />
        </Field>
        <Field label="Urkundenrollennummer" optional>
          <Input
            className="code-num"
            placeholder="UR-Nr. 123/2026"
            value={settings.deedNumber}
            onChange={(e) => patch({ deedNumber: e.target.value })}
          />
        </Field>
        <Field
          label={capitalLabel}
          hint={`mindestens ${formatCents(rules.minShareCapital)}`}
          error={belowMinimum ? `Für ${rules.legalForm} sind mindestens ${formatCents(rules.minShareCapital)} vorgeschrieben.` : undefined}
        >
          <Input
            align="right"
            placeholder="25.000,00"
            value={capitalText}
            onChange={(e) => setCapitalText(e.target.value)}
            onBlur={() => {
              const cents = parseCents(capitalText);
              patch({ shareCapital: cents ?? 0 });
              setCapitalText(cents ? formatCentsPlain(cents) : '');
            }}
          />
        </Field>
      </FormGrid>

      <div className="mt-6">
        <p className="text-label text-ink-muted mb-2">{isAG ? 'Aktionäre' : 'Gesellschafter'}</p>
        {rows.length > 0 && (
          <ul className="flex flex-col gap-2">
            {rows.map((row, index) => (
              <li key={row.key} className="flex items-end gap-3">
                <Field label="Name" className="flex-1">
                  <Input
                    value={row.name}
                    onChange={(e) =>
                      update(rows.map((r, i) => (i === index ? { ...r, name: e.target.value } : r)))
                    }
                  />
                </Field>
                <Field label={isAG ? 'Anteil am Grundkapital' : 'Nennbetrag der Anteile'} className="w-48">
                  <Input
                    align="right"
                    value={row.amountText}
                    onChange={(e) =>
                      update(rows.map((r, i) => (i === index ? { ...r, amountText: e.target.value } : r)))
                    }
                    onBlur={() => {
                      const cents = parseCents(row.amountText);
                      update(
                        rows.map((r, i) =>
                          i === index ? { ...r, amountText: cents ? formatCentsPlain(cents) : '' } : r,
                        ),
                      );
                    }}
                  />
                </Field>
                <Button
                  variant="quiet"
                  iconOnly
                  title="Gesellschafter entfernen"
                  aria-label={`${row.name || `Gesellschafter ${index + 1}`} entfernen`}
                  onClick={() => update(rows.filter((_, i) => i !== index))}
                >
                  <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                </Button>
              </li>
            ))}
          </ul>
        )}
        <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-2">
          <Button
            variant="quiet"
            size="sm"
            className="-ml-2.5"
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => {
              update([...rows, { key: nextKey, name: '', amountText: '' }]);
              setNextKey(nextKey + 1);
            }}
          >
            {isAG ? 'Aktionär hinzufügen' : 'Gesellschafter hinzufügen'}
          </Button>
          {rows.length > 0 && (
            <span className={cn('text-body num', capitalMismatch ? 'text-negative-text' : 'text-ink-subtle')}>
              {formatCents(subscribed)} von {formatCents(settings.shareCapital)} übernommen
            </span>
          )}
        </div>
      </div>
    </Section>
  );
};
