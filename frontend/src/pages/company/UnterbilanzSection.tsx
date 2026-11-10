import React from 'react';
import type { FoundationState } from '../../types';
import { formatCents, formatDate } from '../../utils/formatters';
import { unterbilanzRows } from '../../utils/unterbilanz';
import { DetailList, Help, Table, Tbody, Td, Th, Thead, Tr } from '../../components/ui';

/**
 * Die Unterbilanz am Eintragungstag (FoundationService.Unterbilanz): was die
 * Gesellschafter nachschießen, wenn das Reinvermögen hinter dem Stammkapital
 * zurückbleibt. Vor der Eintragung rechnet Buchfink auf heute.
 */
export const UnterbilanzSection: React.FC<{ foundation: FoundationState | null }> = ({ foundation }) => {
  const u = foundation?.hasFoundation ? foundation.unterbilanz : undefined;
  if (!u) return null;

  return (
    <div className="mt-8">
      <h3 className="text-label text-ink">
        Unterbilanz
        <Help
          summary="Fehlt am Tag der Eintragung Vermögen gegenüber dem Stammkapital, gleichen die Gesellschafter die Differenz aus."
          label="Erklärung zur Unterbilanz"
        >
          Buchfink rechnet das Reinvermögen aus den Buchungen bis zum Tag der Eintragung: Vermögen einschließlich der
          ausstehenden Einlagen, abzüglich der Schulden. Bleibt es hinter dem Stammkapital zurück, haften die
          Gesellschafter nach der Rechtsprechung des BGH anteilig nach ihren Geschäftsanteilen für die Differenz
          (Unterbilanzhaftung, BGH, Urteil vom 9. März 1981, II ZR 54/80). Gründungsaufwand, den die Satzung der
          Gesellschaft auferlegt, deckt die Differenz bis zu seinem Betrag. Vor der Eintragung rechnet Buchfink auf
          heute; die Zahl ist dann vorläufig.
        </Help>
      </h3>
      <p className="text-caption text-ink-subtle mt-0.5">
        {u.isFinal ? `Stand am Eintragungstag ${formatDate(u.asOf)}` : `Vorläufig, Stand ${formatDate(u.asOf)}`}
      </p>
      <DetailList
        className="mt-3"
        items={unterbilanzRows(u).map((row) => ({ label: row.label, value: formatCents(row.cents), numeric: true }))}
      />
      {u.amount > 0 && u.shares.length > 0 && (
        <div className="mt-4">
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Gesellschafter</Th>
                <Th numeric className="w-40">Nennbetrag</Th>
                <Th numeric className="w-40">Haftung</Th>
              </Tr>
            </Thead>
            <Tbody>
              {u.shares.map((s) => (
                <Tr key={s.shareholderId || s.name}>
                  <Td>{s.name}</Td>
                  <Td numeric>{formatCents(s.shareCapital)}</Td>
                  <Td numeric>{formatCents(s.amount)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </div>
      )}
    </div>
  );
};
