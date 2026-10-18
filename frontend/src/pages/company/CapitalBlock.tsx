import React, { useCallback, useEffect, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Api } from '../../services/api';
import type {
  CompanySettings,
  Deadline,
  FoundationRules,
  FoundationState,
  ShareholderLoanView,
} from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatCentsPlain, formatDate, parseCents } from '../../utils/formatters';
import {
  Button,
  DetailList,
  DetailPanel,
  Field,
  FormGrid,
  Input,
  Notice,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from '../../components/ui';
import { EditPanel, Missing, ProfileBlock, PROFILE_LIST } from './ProfileBlock';
import { capitalGaps } from '../../utils/profileGaps';
import { COMPANY_OBLIGATIONS, ObligationList, obligationsOf } from './ObligationList';
import { FoundationDialog } from './FoundationDialog';
import { LoanDialog } from './LoanDialog';

interface ShareRow {
  key: number;
  name: string;
  amount: string;
}

type Draft = { shareCapital: string; notary: string; deedNumber: string; rows: ShareRow[] };

function draftOf(s: CompanySettings): Draft {
  return {
    shareCapital: s.shareCapital ? formatCentsPlain(s.shareCapital) : '',
    notary: s.notary,
    deedNumber: s.deedNumber,
    rows: (s.shareholders ?? []).map((sh, i) => ({
      key: i,
      name: sh.name,
      amount: sh.shareCapital ? formatCentsPlain(sh.shareCapital) : '',
    })),
  };
}

function rateLabel(bp: number): string {
  return `${(bp / 100).toFixed(2).replace('.', ',')} %`;
}

function percentOf(part: number, whole: number): string {
  if (!whole) return '—';
  return `${((part / whole) * 100).toFixed(1).replace('.', ',')} %`;
}

/**
 * Gesellschafter und Kapital: die Gesellschafterliste, die Gründung laut
 * Gesellschaftsvertrag und die Darlehensverträge mit Gesellschaftern.
 */
export const CapitalBlock: React.FC<{
  settings: CompanySettings;
  foundation: FoundationState | null;
  rules: FoundationRules | undefined;
  deadlines: Deadline[];
  /** Das früheste Geschäftsjahr der Bücher; liegt die Gründung darin, gehört sie hierher. */
  firstYear: number;
  onSave: (patch: Partial<CompanySettings>) => Promise<void>;
  onReload: () => void;
}> = ({ settings, foundation, rules, deadlines, firstYear, onSave, onReload }) => {
  const writeLock = useWriteLock();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<Draft>(() => draftOf(settings));
  const [loans, setLoans] = useState<ShareholderLoanView[]>([]);
  const [loanDialog, setLoanDialog] = useState<{ loan?: ShareholderLoanView } | null>(null);
  const [selectedLoan, setSelectedLoan] = useState<ShareholderLoanView | null>(null);
  const [foundationOpen, setFoundationOpen] = useState(false);

  const loadLoans = useCallback(() => {
    Api.getShareholderLoans()
      .then(setLoans)
      .catch(() => setLoans([]));
  }, []);

  useEffect(() => loadLoans(), [loadLoans]);

  const shareholders = settings.shareholders ?? [];
  const capital = settings.shareCapital;
  const subscribed = shareholders.reduce((sum, s) => sum + s.shareCapital, 0);
  const hasFoundation = Boolean(foundation?.hasFoundation && foundation.foundation);
  const missing = capitalGaps(settings, { hasFoundation, firstYear });
  const foundationMissing = missing.includes('Gründung');

  const draftSubscribed = draft.rows.reduce((sum, r) => sum + (parseCents(r.amount) ?? 0), 0);
  const draftCapital = parseCents(draft.shareCapital) ?? 0;

  function save() {
    return onSave({
      shareCapital: draftCapital,
      notary: draft.notary,
      deedNumber: draft.deedNumber,
      shareholders: draft.rows
        .filter((r) => r.name.trim() !== '')
        .map((r) => ({ name: r.name.trim(), shareCapital: parseCents(r.amount) ?? 0 })),
    });
  }

  const f = foundation?.foundation;
  const paidIn = f?.shareholders.reduce((sum, s) => sum + s.paidIn, 0) ?? 0;

  return (
    <ProfileBlock
      id="gesellschafter"
      title="Gesellschafter & Kapital"
      missing={missing}
      context={
        capital
          ? `${formatCents(capital)} Stammkapital · ${shareholders.length} ${shareholders.length === 1 ? 'Gesellschafter' : 'Gesellschafter'}`
          : undefined
      }
      helpSummary="Maßgeblich ist der Stand laut Handelsregister und Gesellschafterliste."
      explain="Die Nennbeträge der Geschäftsanteile ergeben zusammen das Stammkapital (§ 5 Abs. 3 Satz 2 GmbHG). Buchfink erkennt Gesellschafter in Buchungen und Darlehen an ihrem Namen in dieser Liste."
      onEdit={() => {
        setDraft(draftOf(settings));
        setEditing(true);
      }}
    >
      {shareholders.length === 0 ? (
        <p className="text-body text-ink-muted">
          <Missing>Noch keine Gesellschafter eingetragen.</Missing>
        </p>
      ) : (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Gesellschafter</Th>
              <Th numeric className="w-40">Nennbetrag</Th>
              <Th numeric className="w-28">Anteil</Th>
            </Tr>
          </Thead>
          <Tbody>
            {shareholders.map((s) => (
              <Tr key={s.name}>
                <Td>{s.name}</Td>
                <Td numeric>{formatCents(s.shareCapital)}</Td>
                <Td numeric className="text-ink-muted">
                  {percentOf(s.shareCapital, capital || subscribed)}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
      {capital > 0 && subscribed !== capital && shareholders.length > 0 && (
        <p className="mt-2 text-caption text-negative-text">
          Die Anteile ergeben {formatCents(subscribed)}, das Stammkapital beträgt {formatCents(capital)}.
        </p>
      )}

      <DetailList
        className={`mt-5 ${PROFILE_LIST}`}
        items={[
          {
            label: 'Urkunde',
            value: [settings.notary, settings.deedNumber].filter(Boolean).join(' · ') || '—',
          },
          ...(hasFoundation && f
            ? [
                {
                  label: 'Gründung',
                  value: (
                    <span className="flex flex-wrap items-center gap-x-3">
                      <span className="num">
                        Beurkundet am {formatDate(f.notarizedOn)} · {formatCents(paidIn)} von{' '}
                        {formatCents(f.shareCapital)} eingezahlt
                      </span>
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={writeLock.locked || !rules}
                        title={writeLock.hint}
                        onClick={() => setFoundationOpen(true)}
                      >
                        Bearbeiten
                      </Button>
                    </span>
                  ),
                },
              ]
            : []),
        ]}
      />
      {foundation?.anmeldung && !foundation.anmeldung.isSatisfied && foundation.stage === 'vorgesellschaft' && (
        <Notice className="mt-4" text={foundation.anmeldung.findings[0] ?? 'Die Mindesteinzahlung vor der Anmeldung ist nicht erreicht.'} />
      )}
      {foundationMissing && rules && (
        <Notice
          className="mt-5"
          text="Die Gesellschaft wurde in diesen Büchern gegründet. Erfassen Sie Kapital, Anteile und Einzahlungen laut Gesellschaftsvertrag."
          action={
            <Button
              variant="secondary"
              size="sm"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setFoundationOpen(true)}
            >
              Gründung erfassen
            </Button>
          }
        />
      )}

      <div className="mt-8 mb-3 flex items-end justify-between gap-4">
        <div>
          <h3 className="text-label text-ink">Darlehensverträge</h3>
          <p className="text-caption text-ink-subtle mt-0.5">
            {loans.length === 0 ? 'Keine Darlehen mit Gesellschaftern' : 'Stand heute laut Buchungen'}
          </p>
        </div>
        <Button
          variant="quiet"
          size="sm"
          disabled={writeLock.locked || shareholders.length === 0}
          title={writeLock.hint ?? (shareholders.length === 0 ? 'Zuerst die Gesellschafter eintragen.' : undefined)}
          icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => setLoanDialog({})}
        >
          Darlehensvertrag erfassen
        </Button>
      </div>
      {loans.length > 0 && (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Gesellschafter</Th>
              <Th>Richtung</Th>
              <Th numeric className="w-32">Betrag</Th>
              <Th numeric className="w-24">Zins</Th>
              <Th className="w-28">Fällig</Th>
              <Th numeric className="w-32">Stand</Th>
            </Tr>
          </Thead>
          <Tbody>
            {loans.map((view) => (
              <Tr key={view.loan.id}>
                <Td>
                  <button
                    type="button"
                    className="text-left text-ink underline-offset-2 hover:underline"
                    onClick={() => setSelectedLoan(view)}
                  >
                    {view.contactName}
                  </button>
                  {view.findings.length > 0 && (
                    <span className="ml-2 text-caption text-attention-text">
                      {view.findings.length} {view.findings.length === 1 ? 'Befund' : 'Befunde'}
                    </span>
                  )}
                </Td>
                <Td className="text-ink-muted">
                  {view.loan.direction === 'to_company' ? 'an die GmbH' : 'an den Gesellschafter'}
                </Td>
                <Td numeric>{formatCents(view.loan.principal)}</Td>
                <Td numeric>{rateLabel(view.loan.interestRateBp)}</Td>
                <Td className="text-ink-subtle num">
                  {view.loan.maturityDate ? formatDate(view.loan.maturityDate) : 'unbefristet'}
                </Td>
                <Td numeric>{formatCents(view.outstanding)}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      <ObligationList
        deadlines={obligationsOf(deadlines, COMPANY_OBLIGATIONS.gesellschafter)}
        onConfirmed={onReload}
      />

      <EditPanel open={editing} title="Gesellschafter & Kapital" onClose={() => setEditing(false)} onSave={save}>
        <FormGrid>
          <Field
            label="Stammkapital"
            hint={rules ? `Mindestens ${formatCents(rules.minShareCapital)}` : undefined}
          >
            <Input
              align="right"
              value={draft.shareCapital}
              placeholder="25.000,00"
              onChange={(e) => setDraft({ ...draft, shareCapital: e.target.value })}
              onBlur={() => {
                const cents = parseCents(draft.shareCapital);
                if (cents) setDraft({ ...draft, shareCapital: formatCentsPlain(cents) });
              }}
            />
          </Field>
          <div />
          <Field label="Notarin oder Notar" optional>
            <Input value={draft.notary} onChange={(e) => setDraft({ ...draft, notary: e.target.value })} />
          </Field>
          <Field label="Urkundenrollennummer" optional>
            <Input
              className="code-num"
              value={draft.deedNumber}
              placeholder="UR-Nr. 123/2026"
              onChange={(e) => setDraft({ ...draft, deedNumber: e.target.value })}
            />
          </Field>
        </FormGrid>

        <h3 className="text-label text-ink mt-8 mb-2">Gesellschafterliste</h3>
        <ul className="flex flex-col gap-2">
          {draft.rows.map((row, index) => (
            <li key={row.key} className="grid grid-cols-[minmax(0,1fr)_10rem_2.25rem] gap-2 items-center">
              <Input
                aria-label={`Name des ${index + 1}. Gesellschafters`}
                value={row.name}
                placeholder="Name wie in der Gesellschafterliste"
                onChange={(e) =>
                  setDraft({ ...draft, rows: draft.rows.map((r) => (r.key === row.key ? { ...r, name: e.target.value } : r)) })
                }
              />
              <Input
                aria-label="Nennbetrag"
                align="right"
                value={row.amount}
                placeholder="12.500,00"
                onChange={(e) =>
                  setDraft({ ...draft, rows: draft.rows.map((r) => (r.key === row.key ? { ...r, amount: e.target.value } : r)) })
                }
              />
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                title="Gesellschafter entfernen"
                aria-label={`${row.name || `Gesellschafter ${index + 1}`} entfernen`}
                onClick={() => setDraft({ ...draft, rows: draft.rows.filter((r) => r.key !== row.key) })}
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
              setDraft({
                ...draft,
                rows: [...draft.rows, { key: Math.max(0, ...draft.rows.map((r) => r.key)) + 1, name: '', amount: '' }],
              })
            }
          >
            Gesellschafter hinzufügen
          </Button>
          {draft.rows.length > 0 && (
            <span
              className={cn(
                'text-caption num',
                draftCapital > 0 && draftSubscribed !== draftCapital ? 'text-negative-text' : 'text-ink-subtle',
              )}
            >
              {formatCents(draftSubscribed)} von {formatCents(draftCapital)} übernommen
            </span>
          )}
        </div>
      </EditPanel>

      <DetailPanel
        open={selectedLoan !== null}
        onOpenChange={(open) => !open && setSelectedLoan(null)}
        title={selectedLoan ? `Darlehen ${selectedLoan.contactName}` : ''}
        context={
          selectedLoan
            ? `${formatCents(selectedLoan.loan.principal)} zu ${rateLabel(selectedLoan.loan.interestRateBp)} ab ${formatDate(selectedLoan.loan.startDate)}`
            : undefined
        }
        footer={
          selectedLoan && (
            <Button
              variant="secondary"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => {
                setLoanDialog({ loan: selectedLoan });
                setSelectedLoan(null);
              }}
            >
              Berichtigen
            </Button>
          )
        }
      >
        {selectedLoan && (
          <>
            <DetailList
              items={[
                { label: 'Stand heute', value: formatCents(selectedLoan.outstanding), numeric: true },
                {
                  label: 'Ausgezahlt ab',
                  value: selectedLoan.firstMovement ? formatDate(selectedLoan.firstMovement) : 'noch nicht gebucht',
                },
                {
                  label: 'Zinsen gebucht bis',
                  value: selectedLoan.interestBookedThrough ? formatDate(selectedLoan.interestBookedThrough) : '—',
                },
                { label: 'Darlehenskonto', value: <span className="code-num">{selectedLoan.loan.account}</span> },
                { label: 'Zinskonto', value: <span className="code-num">{selectedLoan.loan.interestAccount}</span> },
                { label: 'Vertrag vom', value: selectedLoan.loan.contractDate ? formatDate(selectedLoan.loan.contractDate) : '—' },
                { label: 'Sicherheiten', value: selectedLoan.loan.collateral || '—' },
                { label: 'Rangrücktritt', value: selectedLoan.loan.subordinated ? 'vereinbart' : 'nein' },
              ]}
            />
            {selectedLoan.findings.length > 0 && (
              <div className="mt-6 flex flex-col gap-2">
                {selectedLoan.findings.map((finding) => (
                  <Notice key={`${finding.rule}-${finding.message}`} text={finding.message} />
                ))}
              </div>
            )}
          </>
        )}
      </DetailPanel>

      {loanDialog && (
        <LoanDialog
          loan={loanDialog.loan}
          shareholders={shareholders}
          onClose={() => setLoanDialog(null)}
          onSaved={() => {
            setLoanDialog(null);
            loadLoans();
          }}
        />
      )}

      {foundationOpen && rules && (
        <FoundationDialog
          foundation={f}
          rules={rules}
          defaults={{ notarizedOn: settings.foundedOn, shareCapital: capital, shareholders }}
          onClose={() => setFoundationOpen(false)}
          onSaved={() => {
            setFoundationOpen(false);
            onReload();
          }}
        />
      )}
    </ProfileBlock>
  );
};
