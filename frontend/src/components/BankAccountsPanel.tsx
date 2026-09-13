import React, { useState } from 'react';
import { Landmark, MoreHorizontal, Plus } from 'lucide-react';
import type { Account, BankAccount } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from './WriteLock';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  FormGrid,
  Input,
  Menu,
  MenuItem,
  Notice,
  Section,
  Select,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from './ui';

/** Die Buchungskonten, denen ein Bankkonto zugeordnet werden kann. */
const BANK_LEDGER_ACCOUNTS = ['1800', '1810', '1820', '1830', '1840', '1850'];

/** „DE89370400440532013000" in Viererblöcken. */
function formatIban(iban: string): string {
  return iban.replace(/(.{4})/g, '$1 ').trim();
}

interface BankAccountsPanelProps {
  accounts: BankAccount[];
  paymentAccounts: Account[];
  onChanged: () => Promise<void>;
}

/**
 * Die Bankkonten des Unternehmens und das Konto, dessen Verbindung auf
 * Rechnungen steht. Konten entstehen beim Einlesen eines Kontoauszugs oder hier
 * von Hand.
 */
export const BankAccountsPanel: React.FC<BankAccountsPanelProps> = ({
  accounts,
  paymentAccounts,
  onChanged,
}) => {
  // Die Bankkonten gelten über alle Geschäftsjahre; ein abgeschlossenes Jahr
  // sperrt sie nicht, der Prüfermodus schon.
  const writeLock = useWriteLock();
  const [editing, setEditing] = useState<{ draft: BankAccount; isNew: boolean } | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const occupied = new Set(accounts.map((a) => a.ledgerAccount));
  const ledgerItems = paymentAccounts
    .filter((item) => BANK_LEDGER_ACCOUNTS.includes(item.number))
    .map((item) => ({ value: item.number, label: `${item.number} · ${item.name}` }));

  function openNew() {
    setError('');
    setEditing({
      isNew: true,
      draft: {
        iban: '',
        name: '',
        bic: '',
        bankName: '',
        currency: 'EUR',
        ledgerAccount: BANK_LEDGER_ACCOUNTS.find((n) => !occupied.has(n)) ?? '',
      },
    });
  }

  async function save() {
    if (!editing) return;
    setSaving(true);
    setError('');
    try {
      await Api.configureBankAccounts([editing.draft]);
      toast.success(editing.isNew ? 'Bankkonto angelegt.' : 'Bankkonto gespeichert.');
      setEditing(null);
      await onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  async function chooseForInvoices(account: BankAccount) {
    try {
      await Api.setInvoiceBankAccount(account.iban);
      toast.success(`${account.name} steht ab jetzt auf Ihren Rechnungen.`);
      await onChanged();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  const patch = (next: Partial<BankAccount>) =>
    setEditing((prev) => prev && { ...prev, draft: { ...prev.draft, ...next } });

  return (
    <Section
      helpSummary="Legen Sie fest, welches Bankkonto auf Ihren Rechnungen steht."
      title="Bankkonten"
      context={accounts.length ? `${accounts.length} ${accounts.length === 1 ? 'Konto' : 'Konten'}` : undefined}
      divider={false}
      className="mt-8"
      explain={
        <>
          Buchfink legt ein Bankkonto an, sobald Sie einen Kontoauszug einlesen, und erkennt es
          später an der IBAN wieder. Das Konto für Rechnungen liefert IBAN, BIC und Kreditinstitut
          für Rechnungen, E-Rechnungen und Mahnschreiben. Ohne ein solches Konto enthalten Ihre
          Rechnungen keine Zahlungsverbindung.
        </>
      }
      action={
        <Button
          variant="secondary"
          size="sm"
          icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={openNew}
        >
          Bankkonto anlegen
        </Button>
      }
    >
      {accounts.length === 0 ? (
        <EmptyState
          icon={<Landmark className="w-6 h-6" strokeWidth={1.5} />}
          title="Noch keine Bankkonten"
          description="Lesen Sie einen Kontoauszug ein oder legen Sie ein Konto von Hand an."
        />
      ) : (
        <Table>
          <Thead>
            <Tr>
              <Th>Name</Th>
              <Th>IBAN</Th>
              <Th>Kreditinstitut</Th>
              <Th className="w-28">Buchungskonto</Th>
              <Th className="w-36">Rechnungen</Th>
              <Th className="w-12" aria-label="Aktionen" />
            </Tr>
          </Thead>
          <Tbody>
            {accounts.map((account) => (
              <Tr key={account.iban}>
                <Td>{account.name}</Td>
                <Td className="code-num">{formatIban(account.iban)}</Td>
                <Td className="text-ink-subtle">
                  {[account.bankName, account.bic].filter(Boolean).join(' · ') || '—'}
                </Td>
                <Td className="code-num">{account.ledgerAccount}</Td>
                <Td className="text-ink-subtle">
                  {account.isInvoiceAccount ? 'Konto für Rechnungen' : '—'}
                </Td>
                <Td className="pl-0">
                  <span className="flex items-center justify-end">
                    <Menu
                      trigger={
                        <Button
                          variant="quiet"
                          size="sm"
                          iconOnly
                          title="Aktionen zu diesem Konto"
                          aria-label={`Aktionen zu ${account.name}`}
                        >
                          <MoreHorizontal className="w-4 h-4" strokeWidth={1.5} />
                        </Button>
                      }
                    >
                      <MenuItem
                        disabled={writeLock.locked || account.isInvoiceAccount}
                        title={writeLock.hint}
                        onClick={() => void chooseForInvoices(account)}
                      >
                        Für Rechnungen verwenden
                      </MenuItem>
                      <MenuItem
                        disabled={writeLock.locked}
                        title={writeLock.hint}
                        onClick={() => {
                          setError('');
                          setEditing({ isNew: false, draft: { ...account } });
                        }}
                      >
                        Bearbeiten
                      </MenuItem>
                    </Menu>
                  </span>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      {editing && (
        <Dialog
          open
          onOpenChange={(open) => {
            if (!open && !saving) setEditing(null);
          }}
          title={editing.isNew ? 'Bankkonto anlegen' : 'Bankkonto bearbeiten'}
          footer={
            <>
              <Button variant="secondary" disabled={saving} onClick={() => setEditing(null)}>
                Abbrechen
              </Button>
              <Button
                variant="primary"
                loading={saving}
                disabled={
                  writeLock.locked ||
                  !editing.draft.name.trim() ||
                  !editing.draft.iban.trim() ||
                  !editing.draft.ledgerAccount
                }
                onClick={() => void save()}
              >
                Speichern
              </Button>
            </>
          }
        >
          <div className="flex flex-col gap-5">
            <FormGrid>
              <Field label="Name des Kontos" hint="Zum Beispiel Geschäftskonto" className="md:col-span-2">
                <Input value={editing.draft.name} onChange={(e) => patch({ name: e.target.value })} />
              </Field>
              <Field
                label="IBAN"
                hint={editing.isNew ? undefined : 'nach dem Anlegen fest'}
                className="md:col-span-2"
              >
                <Input
                  className="code-num"
                  value={editing.draft.iban}
                  disabled={!editing.isNew}
                  onChange={(e) => patch({ iban: e.target.value })}
                />
              </Field>
              <Field label="BIC" optional>
                <Input
                  className="code-num"
                  value={editing.draft.bic ?? ''}
                  onChange={(e) => patch({ bic: e.target.value })}
                />
              </Field>
              <Field label="Kreditinstitut" optional>
                <Input
                  value={editing.draft.bankName ?? ''}
                  onChange={(e) => patch({ bankName: e.target.value })}
                />
              </Field>
              <Field
                label="Zuordnung in der Buchhaltung"
                hint={editing.isNew ? 'Für jedes Bankkonto ein eigenes' : 'nach dem Anlegen fest'}
                className="md:col-span-2"
              >
                <Select
                  value={editing.draft.ledgerAccount}
                  disabled={!editing.isNew}
                  items={ledgerItems}
                  onValueChange={(value) => patch({ ledgerAccount: value })}
                />
              </Field>
            </FormGrid>
            {error && <Notice tone="negative" text={error} />}
          </div>
        </Dialog>
      )}
    </Section>
  );
};
