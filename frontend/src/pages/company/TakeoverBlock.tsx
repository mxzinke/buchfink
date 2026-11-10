import React, { useState } from 'react';
import { Plus } from 'lucide-react';
import { Api } from '../../services/api';
import type { Cents, FoundationState, TakeoverReconciliation } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { formatCents, formatCentsPlain, formatDate } from '../../utils/formatters';
import {
  Button,
  DetailList,
  Disclosure,
  Field,
  Input,
  Notice,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from '../../components/ui';
import { EditPanel, Missing, PROFILE_LIST, ProfileBlock } from './ProfileBlock';
import { TakeoverDialog } from './TakeoverDialog';

const SUB_LEDGER: Record<TakeoverReconciliation['subLedgers'][number]['kind'], string> = {
  receivables: 'Offene Forderungen',
  payables: 'Offene Verbindlichkeiten',
  assets: 'Anlagen',
};

/** Ein Saldo mit seiner Seite: Soll positiv, Haben negativ. */
function balance(amount: Cents): string {
  if (amount === 0) return formatCentsPlain(0);
  return `${formatCentsPlain(Math.abs(amount))} ${amount < 0 ? 'H' : 'S'}`;
}

/**
 * Der Umstieg aus einem Vorsystem: der Tag, zu dem die Bücher mit übernommenen
 * Beständen beginnen, und die Abstimmung der Eröffnung mit der Schlussbilanz
 * des Vorsystems. Den Tag legt nur der Mensch fest; die Eröffnung bucht er
 * hier oder Claude nach Rückfrage.
 */
export const TakeoverBlock: React.FC<{
  takeover: TakeoverReconciliation | null;
  foundation: FoundationState | null;
  onReload: () => void;
}> = ({ takeover, foundation, onReload }) => {
  const writeLock = useWriteLock();
  const [editing, setEditing] = useState(false);
  const [date, setDate] = useState('');
  const [capturing, setCapturing] = useState(false);

  // Eine in Buchfink gegründete Gesellschaft übernimmt keine Bestände.
  if (!takeover || (!takeover.date && foundation?.hasFoundation)) return null;

  const set = takeover.date !== '';
  const state = !takeover.booked ? (
    <Missing>nicht gebucht</Missing>
  ) : takeover.reconciled ? (
    <StatusBadge status="gebucht" />
  ) : (
    <StatusBadge status="fehlerhaft" />
  );

  return (
    <ProfileBlock
      id="umstieg"
      title="Umstieg"
      context={set ? undefined : 'Kein Umstieg festgelegt.'}
      helpSummary="Für Unternehmen, die ihre Bücher aus einem anderen Programm mitbringen."
      explain={
        <>
          Der Umstieg legt den Tag fest, an dem die Bücher in Buchfink mit den Beständen des Vorsystems beginnen: den
          ersten Tag eines Geschäftsjahres. Die Eröffnung übernimmt die Schlussbilanz des Vorjahres, jede offene
          Rechnung einzeln und jedes Anlagegut mit Anschaffungskosten und bisheriger Abschreibung. Die Wertansätze der
          Eröffnung müssen mit denen der Schlussbilanz übereinstimmen (§ 252 Abs. 1 Nr. 1 HGB); die Abstimmung zeigt
          jede Differenz. Die Daten des Vorsystems bleiben dort aufzubewahren.
        </>
      }
      onEdit={
        takeover.booked
          ? undefined
          : () => {
              setDate(takeover.date);
              setEditing(true);
            }
      }
      editLabel={set ? 'Bearbeiten' : 'Umstieg festlegen'}
    >
      {set && (
        <div className="flex items-start justify-between gap-4">
          <DetailList
            className={`flex-1 ${PROFILE_LIST}`}
            items={[
              { label: 'Beginn der Bücher', value: formatDate(takeover.date) },
              { label: 'Eröffnung', value: state },
              ...(takeover.booked
                ? [
                    { label: 'Aktiva', value: formatCents(takeover.bookedAssets), numeric: true },
                    { label: 'Passiva', value: formatCents(takeover.bookedLiabilities), numeric: true },
                  ]
                : []),
            ]}
          />
          {!takeover.booked && (
            <Button
              variant="secondary"
              size="sm"
              disabled={writeLock.locked}
              title={writeLock.hint}
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => setCapturing(true)}
            >
              Eröffnung erfassen
            </Button>
          )}
        </div>
      )}

      {takeover.booked &&
        takeover.differences.map((difference) => (
          <Notice key={difference} tone="negative" className="mt-4" text={difference} />
        ))}

      {takeover.subLedgers.length > 0 && (
        <div className="mt-6">
          <h3 className="text-label text-ink mb-2">Nebenbücher</h3>
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Nebenbuch</Th>
                <Th>Sachkonto</Th>
                <Th numeric className="w-20">Anzahl</Th>
                <Th numeric className="w-36">Nebenbuch</Th>
                <Th numeric className="w-36">Sachkonto</Th>
                <Th numeric className="w-32">Differenz</Th>
              </Tr>
            </Thead>
            <Tbody>
              {takeover.subLedgers.map((row) => (
                <Tr key={`${row.kind}-${row.account}`}>
                  <Td>{SUB_LEDGER[row.kind]}</Td>
                  <Td>
                    <span className="code-num">{row.account}</span> {row.accountName}
                  </Td>
                  <Td numeric>{row.count}</Td>
                  <Td numeric>{formatCents(row.subLedger)}</Td>
                  <Td numeric>{formatCents(row.ledger)}</Td>
                  <Td numeric>{formatCents(row.difference)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </div>
      )}

      {takeover.accounts.length > 0 && (
        <Disclosure
          title="Abstimmung je Konto"
          context={`${takeover.accounts.length} Konten · Schlussbilanz ${formatCents(takeover.statementAssets)}`}
        >
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-20">Konto</Th>
                <Th>Bezeichnung</Th>
                <Th numeric className="w-40">Schlussbilanz</Th>
                <Th numeric className="w-40">Gebucht</Th>
                <Th numeric className="w-32">Differenz</Th>
              </Tr>
            </Thead>
            <Tbody>
              {takeover.accounts.map((row) => (
                <Tr key={row.account}>
                  <Td className="code-num">{row.account}</Td>
                  <Td>{row.accountName}</Td>
                  <Td numeric>{balance(row.statement)}</Td>
                  <Td numeric>{balance(row.booked)}</Td>
                  <Td numeric>{formatCents(Math.abs(row.difference))}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Disclosure>
      )}

      <EditPanel
        open={editing}
        title="Umstieg festlegen"
        onClose={() => setEditing(false)}
        onSave={async () => {
          await Api.setTakeoverDate(date);
          onReload();
        }}
      >
        <Field
          label="Bücher beginnen am"
          hint="Erster Tag eines Geschäftsjahres"
          helpSummary="Ab diesem Tag führt Buchfink die Bücher mit den Beständen des Vorsystems."
          explain="Leer heißt: kein Umstieg. Ein Umstieg im laufenden Geschäftsjahr ist nicht vorgesehen; buchen Sie das angebrochene Jahr im Vorsystem zu Ende und steigen Sie zum nächsten Geschäftsjahr um."
        >
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
      </EditPanel>

      {capturing && (
        <TakeoverDialog
          date={takeover.date}
          onClose={() => setCapturing(false)}
          onBooked={() => {
            setCapturing(false);
            onReload();
          }}
        />
      )}
    </ProfileBlock>
  );
};
