import React, { useEffect, useState } from 'react';
import { Api } from '../../services/api';
import type { CompanySettings, TaxAdvisor, VatPeriodProposal } from '../../types';
import {
  DECLARATIONS,
  FEDERAL_STATES,
  VAT_PERIODS,
  federalStateLabel,
  vatPeriodLabel,
} from '../../utils/companyLabels';
import { formatCents, formatCentsPlain, parseCents } from '../../utils/formatters';
import {
  Checkbox,
  DetailList,
  Field,
  FormGrid,
  Input,
  Notice,
  RadioGroup,
  Select,
} from '../../components/ui';
import { EditPanel, Missing, ProfileBlock, PROFILE_LIST } from './ProfileBlock';
import { taxGaps } from '../../utils/profileGaps';

/** Die Grenzen des Hebesatzes, die der Dienst durchsetzt. */
const TRADE_TAX_MIN = 200;
const TRADE_TAX_MAX = 1000;

type Draft = Pick<
  CompanySettings,
  | 'taxNumber'
  | 'federalState'
  | 'taxOffice'
  | 'vatId'
  | 'vatPeriod'
  | 'permanentExtension'
  | 'specialPrepayment'
  | 'nonProfit'
  | 'inLiquidation'
  | 'fiscalUnity'
> & { taxAdvisor: TaxAdvisor };

function draftOf(s: CompanySettings): Draft {
  return {
    taxNumber: s.taxNumber,
    federalState: s.federalState ?? '',
    taxOffice: s.taxOffice,
    vatId: s.vatId,
    vatPeriod: s.vatPeriod || 'unknown',
    permanentExtension: s.permanentExtension,
    specialPrepayment: s.specialPrepayment,
    nonProfit: Boolean(s.nonProfit),
    inLiquidation: Boolean(s.inLiquidation),
    fiscalUnity: Boolean(s.fiscalUnity),
    taxAdvisor: {
      engaged: Boolean(s.taxAdvisor?.engaged),
      sinceYear: s.taxAdvisor?.sinceYear ?? 0,
      declarations: s.taxAdvisor?.declarations ?? [],
    },
  };
}

function advisorLabel(advisor: TaxAdvisor | undefined): string {
  if (!advisor?.engaged) return 'Ohne Steuerberater';
  const which =
    advisor.declarations.length === 0 || advisor.declarations.length === DECLARATIONS.length
      ? 'alle Erklärungen'
      : DECLARATIONS.filter((d) => advisor.declarations.includes(d.value))
          .map((d) => d.label)
          .join(', ');
  return `Mit Steuerberater${advisor.sinceYear ? ` ab ${advisor.sinceYear}` : ''} für ${which}`;
}

/** Was Buchfink für eine Gesellschaft mit diesem Merkmal nicht rechnet. */
export function exclusionsOf(s: Pick<CompanySettings, 'nonProfit' | 'inLiquidation' | 'fiscalUnity'>): string[] {
  return [
    s.nonProfit && 'gemeinnützig',
    s.inLiquidation && 'in Liquidation',
    s.fiscalUnity && 'Organgesellschaft',
  ].filter(Boolean) as string[];
}

const EXCLUSION_NOTE =
  'Für diese Gesellschaft rechnet Buchfink weder Jahresabschluss noch Steuerrückstellung. Beides gehört zu einer Steuerberatung.';

/** Steuern: Nummern, Finanzamt, Umsatzsteuer, Hebesatz, Beratung und Ausschlüsse. */
export const TaxBlock: React.FC<{
  settings: CompanySettings;
  tradeTaxRatePercent: number;
  year: number;
  onSave: (patch: Partial<CompanySettings>, tradeTaxRatePercent: number) => Promise<void>;
}> = ({ settings, tradeTaxRatePercent, year, onSave }) => {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<Draft>(() => draftOf(settings));
  const [rateText, setRateText] = useState(String(tradeTaxRatePercent));
  const [rateError, setRateError] = useState('');
  const [prepaymentText, setPrepaymentText] = useState('');
  const [elster, setElster] = useState<{ value: string; error: string }>({ value: '', error: '' });
  const [proposal, setProposal] = useState<VatPeriodProposal | null>(null);

  const patch = (next: Partial<Draft>) => setDraft((prev) => ({ ...prev, ...next }));

  // Die Steuernummer im Bundesschema schon beim Eintippen; erst mit Land
  // lässt sie sich umrechnen.
  useEffect(() => {
    if (!editing) return;
    let active = true;
    const timer = window.setTimeout(() => {
      Api.formatElsterTaxNumber(draft.taxNumber, draft.federalState ?? '')
        .then((value) => active && setElster({ value, error: '' }))
        .catch((e) => active && setElster({ value: '', error: e instanceof Error ? e.message : String(e) }));
    }, 250);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [editing, draft.taxNumber, draft.federalState]);

  useEffect(() => {
    if (!editing) return;
    Api.getVatPeriodProposal(year)
      .then(setProposal)
      .catch(() => setProposal(null));
  }, [editing, year]);

  function open() {
    setDraft(draftOf(settings));
    setRateText(String(tradeTaxRatePercent));
    setRateError('');
    setPrepaymentText(settings.specialPrepayment ? formatCentsPlain(settings.specialPrepayment) : '');
    setEditing(true);
  }

  async function save() {
    const rate = Math.trunc(Number(rateText));
    if (!Number.isFinite(rate) || rate < TRADE_TAX_MIN || rate > TRADE_TAX_MAX) {
      const message = `Der Hebesatz liegt zwischen ${TRADE_TAX_MIN} % und ${TRADE_TAX_MAX} %.`;
      setRateError(message);
      throw new Error(message);
    }
    const prepayment = parseCents(prepaymentText);
    await onSave(
      {
        ...draft,
        specialPrepayment: draft.permanentExtension && prepayment ? prepayment : 0,
      },
      rate,
    );
  }

  const exclusions = exclusionsOf(settings);
  const draftExclusions = exclusionsOf(draft);
  const missing = taxGaps(settings);

  return (
    <ProfileBlock
      id="steuern"
      title="Steuern"
      missing={missing}
      context="Bestimmt Fristen, Voranmeldungen und Erklärungen"
      helpSummary="Aus diesen Angaben rechnet Buchfink Fristen, Voranmeldungen und Erklärungen."
      explain="Das Bundesland bestimmt die Feiertage am Fristende (§ 108 Abs. 3 AO) und das Schema der Steuernummer. Mit Steuerberater verlängert sich die Frist der Jahreserklärungen (§ 149 Abs. 3 AO)."
      onEdit={open}
    >
      {exclusions.length > 0 && (
        <Notice tone="negative" className="mb-5" text={`Die Gesellschaft ist als ${exclusions.join(', ')} gekennzeichnet. ${EXCLUSION_NOTE}`} />
      )}
      <DetailList
        className={PROFILE_LIST}
        items={[
          {
            label: 'Steuernummer',
            value: settings.taxNumber ? (
              <span>
                <span className="code-num">{settings.taxNumber}</span>
                {settings.taxNumberElster && (
                  <span className="ml-3 text-caption text-ink-subtle">
                    ELSTER <span className="code-num">{settings.taxNumberElster}</span>
                  </span>
                )}
              </span>
            ) : (
              <Missing />
            ),
          },
          { label: 'Bundesland', value: federalStateLabel(settings.federalState) || <Missing /> },
          { label: 'Finanzamt', value: settings.taxOffice || '—' },
          { label: 'USt-IdNr.', value: settings.vatId ? <span className="code-num">{settings.vatId}</span> : '—' },
          {
            label: 'Umsatzsteuer',
            value:
              !settings.vatPeriod || settings.vatPeriod === 'unknown' ? (
                <Missing>Zeitraum noch nicht geklärt</Missing>
              ) : (
                `${vatPeriodLabel(settings.vatPeriod)}${settings.permanentExtension ? ' · mit Dauerfristverlängerung' : ''}`
              ),
          },
          { label: 'Hebesatz', value: `${tradeTaxRatePercent} %`, numeric: true },
          { label: 'Beratung', value: advisorLabel(settings.taxAdvisor) },
        ]}
      />

      <EditPanel open={editing} title="Steuern" onClose={() => setEditing(false)} onSave={save}>
        <FormGrid>
          <Field
            label="Steuernummer"
            hint={elster.error ? undefined : elster.value ? `ELSTER: ${elster.value}` : 'Wie auf dem Bescheid'}
            error={elster.error || undefined}
          >
            <Input className="code-num" value={draft.taxNumber} onChange={(e) => patch({ taxNumber: e.target.value })} />
          </Field>
          <Field label="Bundesland" hint="Sitz der Gesellschaft">
            <Select
              items={FEDERAL_STATES}
              value={draft.federalState ?? ''}
              placeholder="Bitte wählen"
              onValueChange={(code) => patch({ federalState: String(code) })}
            />
          </Field>
          <Field label="Finanzamt" optional>
            <Input value={draft.taxOffice} onChange={(e) => patch({ taxOffice: e.target.value })} />
          </Field>
          <Field label="USt-IdNr." optional>
            <Input className="code-num" value={draft.vatId} onChange={(e) => patch({ vatId: e.target.value })} />
          </Field>
          <Field
            label="Umsatzsteuer-Voranmeldung"
            hint={
              proposal?.changes
                ? `Vorschlag aus ${proposal.basedOnYear}: ${vatPeriodLabel(proposal.proposed)}`
                : undefined
            }
            helpSummary="Den Zeitraum legt das Finanzamt fest; er steht im Schreiben zur steuerlichen Erfassung."
            explain="Der Zeitraum richtet sich nach der Umsatzsteuer des Vorjahres, für Gründungen gelten Sonderregeln (§ 18 Abs. 2 UStG). Sobald die Steuer des Vorjahres feststeht, schlägt Buchfink einen Wechsel vor."
          >
            <Select
              items={VAT_PERIODS}
              value={draft.vatPeriod}
              onValueChange={(period) => patch({ vatPeriod: String(period) })}
            />
          </Field>
          <Field
            label="Gewerbesteuer-Hebesatz"
            hint="Prozent der Gemeinde"
            error={rateError || undefined}
          >
            <Input
              type="number"
              min={TRADE_TAX_MIN}
              max={TRADE_TAX_MAX}
              align="right"
              value={rateText}
              onChange={(e) => {
                setRateText(e.target.value);
                setRateError('');
              }}
            />
          </Field>
          <div className="md:col-span-2 flex flex-col gap-3">
            <Checkbox
              checked={draft.permanentExtension}
              onCheckedChange={(next) => patch({ permanentExtension: Boolean(next) })}
              label="Dauerfristverlängerung genehmigt"
              hint="Jede Voranmeldung wird einen Monat später fällig"
            />
            {draft.permanentExtension && draft.vatPeriod === 'month' && (
              <Field label="Angemeldete Sondervorauszahlung" hint="Ein Elftel der Vorjahressteuer" className="max-w-xs">
                <Input
                  align="right"
                  placeholder="0,00"
                  value={prepaymentText}
                  onChange={(e) => setPrepaymentText(e.target.value)}
                  onBlur={() => {
                    const cents = parseCents(prepaymentText);
                    setPrepaymentText(cents ? formatCentsPlain(cents) : '');
                  }}
                />
              </Field>
            )}
          </div>
        </FormGrid>

        <h3 className="text-label text-ink mt-8 mb-3">Steuerliche Beratung</h3>
        <RadioGroup
          inline
          options={[
            { value: 'ohne', label: 'Ohne Steuerberater' },
            { value: 'mit', label: 'Mit Steuerberater' },
          ]}
          value={draft.taxAdvisor.engaged ? 'mit' : 'ohne'}
          onValueChange={(next) => patch({ taxAdvisor: { ...draft.taxAdvisor, engaged: next === 'mit' } })}
        />
        {draft.taxAdvisor.engaged && (
          <div className="mt-4 flex flex-col gap-4">
            <Field label="Ab Geschäftsjahr" hint="Leer heißt: alle Jahre" className="max-w-xs">
              <Input
                type="number"
                align="right"
                value={draft.taxAdvisor.sinceYear ? String(draft.taxAdvisor.sinceYear) : ''}
                onChange={(e) =>
                  patch({ taxAdvisor: { ...draft.taxAdvisor, sinceYear: Math.trunc(Number(e.target.value)) || 0 } })
                }
              />
            </Field>
            <div className="flex flex-col gap-2">
              <span className="text-label text-ink-muted">Übernimmt</span>
              {DECLARATIONS.map((declaration) => {
                const list = draft.taxAdvisor.declarations;
                const all = list.length === 0;
                return (
                  <Checkbox
                    key={declaration.value}
                    checked={all || list.includes(declaration.value)}
                    onCheckedChange={(checked) => {
                      const current = all ? DECLARATIONS.map((d) => d.value) : list;
                      const next = checked
                        ? Array.from(new Set([...current, declaration.value]))
                        : current.filter((d) => d !== declaration.value);
                      patch({
                        taxAdvisor: {
                          ...draft.taxAdvisor,
                          declarations: next.length === DECLARATIONS.length ? [] : next,
                        },
                      });
                    }}
                    label={declaration.label}
                  />
                );
              })}
            </div>
          </div>
        )}

        <h3 className="text-label text-ink mt-8 mb-3">Was Buchfink nicht abdeckt</h3>
        <div className="flex flex-col gap-2">
          <Checkbox
            checked={draft.nonProfit}
            onCheckedChange={(next) => patch({ nonProfit: Boolean(next) })}
            label="Die Gesellschaft ist gemeinnützig"
          />
          <Checkbox
            checked={draft.inLiquidation}
            onCheckedChange={(next) => patch({ inLiquidation: Boolean(next) })}
            label="Die Gesellschaft ist in Liquidation"
          />
          <Checkbox
            checked={draft.fiscalUnity}
            onCheckedChange={(next) => patch({ fiscalUnity: Boolean(next) })}
            label="Die Gesellschaft ist Organgesellschaft einer Organschaft"
          />
        </div>
        {draftExclusions.length > 0 && <Notice tone="negative" className="mt-4" text={EXCLUSION_NOTE} />}
        {settings.specialPrepayment > 0 && !draft.permanentExtension && (
          <p className="mt-4 text-caption text-ink-subtle">
            Ohne Dauerfristverlängerung entfällt die angemeldete Sondervorauszahlung von{' '}
            {formatCents(settings.specialPrepayment)}.
          </p>
        )}
      </EditPanel>
    </ProfileBlock>
  );
};
