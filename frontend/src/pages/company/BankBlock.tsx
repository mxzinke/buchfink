import React, { useCallback, useEffect, useState } from 'react';
import { PenLine } from 'lucide-react';
import { Api } from '../../services/api';
import type { Account, BankAccount, CompanySettings } from '../../types';
import { BankAccountsPanel } from '../../components/BankAccountsPanel';
import { useWriteLock } from '../../components/WriteLock';
import { Button, DetailList, Field, FormGrid, Input, SkeletonRows, toast } from '../../components/ui';
import { EditPanel, Missing, PROFILE_LIST } from './ProfileBlock';

type Draft = Pick<
  CompanySettings,
  'contactName' | 'contactPhone' | 'contactEmail' | 'invoiceNumberFormat' | 'receiptNumberFormat'
>;

function draftOf(s: CompanySettings): Draft {
  return {
    contactName: s.contactName,
    contactPhone: s.contactPhone,
    contactEmail: s.contactEmail,
    invoiceNumberFormat: s.invoiceNumberFormat,
    receiptNumberFormat: s.receiptNumberFormat,
  };
}

/**
 * Bankkonten und was auf Rechnungen steht: das Konto für Zahlungen,
 * Ansprechpartner für Rückfragen und die Systematik der Nummern.
 */
export const BankBlock: React.FC<{
  settings: CompanySettings;
  onSave: (patch: Partial<CompanySettings>) => Promise<void>;
  onChanged: () => void;
}> = ({ settings, onSave, onChanged }) => {
  const writeLock = useWriteLock();
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<Draft>(() => draftOf(settings));
  const patch = (next: Partial<Draft>) => setDraft((prev) => ({ ...prev, ...next }));

  const load = useCallback(async () => {
    try {
      const [banks, payment] = await Promise.all([Api.getBankAccounts(), Api.getPaymentAccounts()]);
      setAccounts(banks);
      setPaymentAccounts(payment);
    } catch (e) {
      setAccounts([]);
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (accounts === null) {
    return (
      <section id="bankkonten" className="mt-8 pt-8 border-t border-line">
        <SkeletonRows rows={3} />
      </section>
    );
  }

  const contact = [settings.contactName, settings.contactPhone, settings.contactEmail].filter(Boolean).join(' · ');

  return (
    <BankAccountsPanel
      id="bankkonten"
      title="Bankkonten & Rechnungen"
      divider
      accounts={accounts}
      paymentAccounts={paymentAccounts}
      onChanged={async () => {
        await load();
        onChanged();
      }}
    >
      <div className="mt-6 flex items-start justify-between gap-4">
        <DetailList
          className={`flex-1 ${PROFILE_LIST}`}
          items={[
            {
              label: 'Konto auf Rechnungen',
              value: settings.iban ? (
                <span className="code-num">{settings.iban.replace(/(.{4})/g, '$1 ').trim()}</span>
              ) : (
                <Missing>Kein Konto festgelegt</Missing>
              ),
            },
            { label: 'Ansprechpartner', value: contact || <Missing>Für XRechnungen nötig</Missing> },
            {
              label: 'Rechnungsnummern',
              value: <span className="code-num">{settings.invoiceNumberFormat || 'RE-{JAHR}-{NR:4}'}</span>,
            },
            {
              label: 'Belegnummern',
              value: <span className="code-num">{settings.receiptNumberFormat || 'ER-{JAHR}-{NR:4}'}</span>,
            },
          ]}
        />
        <Button
          variant="secondary"
          size="sm"
          disabled={writeLock.locked}
          title={writeLock.hint}
          icon={<PenLine className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => {
            setDraft(draftOf(settings));
            setEditing(true);
          }}
        >
          Rechnungsangaben
        </Button>
      </div>

      <EditPanel open={editing} title="Angaben auf Rechnungen" onClose={() => setEditing(false)} onSave={() => onSave(draft)}>
        <FormGrid>
          <Field label="Ansprechpartner" hint="Bei XRechnung Pflicht">
            <Input value={draft.contactName} onChange={(e) => patch({ contactName: e.target.value })} />
          </Field>
          <Field label="Telefon" hint="Bei XRechnung Pflicht">
            <Input value={draft.contactPhone} onChange={(e) => patch({ contactPhone: e.target.value })} />
          </Field>
          <Field label="E-Mail für Rückfragen" hint="Bei XRechnung Pflicht" className="md:col-span-2">
            <Input type="email" value={draft.contactEmail} onChange={(e) => patch({ contactEmail: e.target.value })} />
          </Field>
          <Field
            label="Rechnungsnummern"
            hint="{JAHR} und {NR:4}"
            helpSummary="Wie Jahr und laufende Nummer in der Rechnungsnummer stehen."
            explain="{JAHR} steht für das Geschäftsjahr, {NR:4} für den Zähler mit vier Stellen. Ohne {NR} hätte jede Rechnung dieselbe Nummer; ein solches Format weist Buchfink zurück (§ 14 Abs. 4 Nr. 4 UStG)."
          >
            <Input
              className="code-num"
              placeholder="RE-{JAHR}-{NR:4}"
              value={draft.invoiceNumberFormat}
              onChange={(e) => patch({ invoiceNumberFormat: e.target.value })}
            />
          </Field>
          <Field label="Belegnummern" hint="{JAHR} und {NR:4}">
            <Input
              className="code-num"
              placeholder="ER-{JAHR}-{NR:4}"
              value={draft.receiptNumberFormat}
              onChange={(e) => patch({ receiptNumberFormat: e.target.value })}
            />
          </Field>
        </FormGrid>
      </EditPanel>
    </BankAccountsPanel>
  );
};
