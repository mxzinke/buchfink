import React from 'react';
import { Bot } from 'lucide-react';
import type { AgentBooking, AgentBookings, ConnectedApp } from '../../types';
import { formatCents, formatDate } from '../../utils/formatters';
import { appName, onlyClaude } from '../../utils/overview';
import type { NavigateFn } from '../../navigation';
import {
  Button,
  EmptyState,
  Section,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from '../../components/ui';

/** Der Betrag einer Buchung: die Summe ihrer Sollzeilen. */
function grossOf(entry: AgentBooking): number {
  return entry.lines.filter((line) => line.side === 'S').reduce((sum, line) => sum + line.amount, 0);
}

export interface AgentBookingsSectionProps {
  bookings: AgentBookings | null;
  apps: ConnectedApp[];
  onNavigate: NavigateFn;
}

/**
 * Was verbundene Apps seit der letzten Festschreibung gebucht haben. Bis zur
 * Festschreibung sieht der Mensch hier nach; eine falsche Buchung hebt ein
 * Storno auf. Eine Zeile führt zur Buchung im Journal.
 */
export const AgentBookingsSection: React.FC<AgentBookingsSectionProps> = ({
  bookings,
  apps,
  onNavigate,
}) => {
  const entries = bookings?.entries ?? [];
  const channels = [...new Set(entries.map((entry) => entry.channel ?? ''))];
  const title = onlyClaude(channels)
    ? 'Von Claude gebucht seit der letzten Festschreibung'
    : 'Von Apps gebucht seit der letzten Festschreibung';
  // Eine Spalte für die App erst, wenn mehr als eine gebucht hat. Sonst steht
  // sie einmal in der Kontextzeile.
  const severalApps = channels.length > 1;
  const numbers = new Map(entries.map((entry) => [entry.id, entry.entryNumber]));

  let context = 'Buchungen, die noch nicht festgeschrieben sind';
  if (bookings) {
    const count = bookings.total === 1 ? '1 Buchung' : `${bookings.total} Buchungen`;
    const by = !severalApps && channels[0] ? ` von ${appName(channels[0], apps)}` : '';
    const since = bookings.lastCommit
      ? ` · festgeschrieben bis ${formatDate(bookings.lastCommit.cutoffDate)}`
      : ' · noch keine Festschreibung';
    context = `${count}${by}${since}`;
  }

  return (
    <Section
      title={title}
      context={context}
      helpSummary="Prüfen Sie diese Buchungen, bevor Sie den Monat festschreiben."
      explain={
        <>
          Hier stehen die Buchungen, die eine verbundene App seit der letzten Festschreibung
          angelegt hat. Ist eine falsch, hebt ein Storno sie auf und eine neue Buchung tritt an ihre
          Stelle. Bitten Sie Claude darum. Nach der Festschreibung bleibt die Buchung stehen; die
          Korrektur folgt dann im laufenden Monat.
        </>
      }
      action={
        bookings && bookings.total > entries.length ? (
          <Button variant="quiet" onClick={() => onNavigate({ area: 'books', tab: 'journal' })}>
            {`Alle ${bookings.total} im Journal`}
          </Button>
        ) : undefined
      }
    >
      {!bookings ? (
        <SkeletonRows rows={4} />
      ) : entries.length === 0 ? (
        <EmptyState
          icon={<Bot className="w-6 h-6" strokeWidth={1.5} />}
          title="Keine neuen Buchungen von Apps"
          description={
            apps.length > 0
              ? 'Seit der letzten Festschreibung hat keine verbundene App gebucht.'
              : 'Sobald Sie Claude verbinden, stehen seine Buchungen hier.'
          }
          action={
            apps.length === 0 ? (
              <Button variant="secondary" onClick={() => onNavigate({ area: 'settings', tab: 'apps' })}>
                Claude verbinden
              </Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-32">Buchung</Th>
                <Th className="w-28">Datum</Th>
                <Th>Buchungstext</Th>
                {severalApps && <Th className="w-36">App</Th>}
                <Th numeric className="w-32">
                  Betrag
                </Th>
                <Th className="w-44" aria-label="Storno" />
              </Tr>
            </Thead>
            <Tbody>
              {entries.map((entry) => {
                const isReversal = entry.kind === 'reversal';
                const original = entry.reversalOfId ? numbers.get(entry.reversalOfId) : undefined;
                return (
                  <Tr
                    key={entry.id}
                    variant={isReversal || entry.reversed ? 'storno' : 'default'}
                    className="cursor-pointer"
                    onClick={() =>
                      onNavigate({ area: 'books', tab: 'journal', id: entry.entryNumber })
                    }
                  >
                    <Td code>
                      <button
                        type="button"
                        className="underline-offset-2 hover:underline"
                        onClick={(event) => {
                          event.stopPropagation();
                          onNavigate({ area: 'books', tab: 'journal', id: entry.entryNumber });
                        }}
                      >
                        {entry.entryNumber}
                      </button>
                    </Td>
                    <Td className="num text-ink-subtle">{formatDate(entry.bookingDate)}</Td>
                    <Td className="max-w-[24rem] truncate" title={entry.description}>
                      {entry.description}
                    </Td>
                    {severalApps && (
                      <Td className="text-caption text-ink-subtle">{appName(entry.channel, apps)}</Td>
                    )}
                    <Td numeric>{formatCents(grossOf(entry), entry.currency)}</Td>
                    <Td>
                      {isReversal ? (
                        <StatusBadge
                          status="storniert"
                          reference={original ? `Storno zu ${original}` : 'Storno'}
                        />
                      ) : entry.reversed ? (
                        <StatusBadge status="storniert" />
                      ) : null}
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
          <p className="mt-3 text-caption text-ink-subtle">
            Eine falsche Buchung hebt ein Storno auf. Bitten Sie Claude darum.
          </p>
        </>
      )}
    </Section>
  );
};
