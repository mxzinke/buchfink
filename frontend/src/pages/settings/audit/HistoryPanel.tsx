import React, { useEffect, useState } from 'react';
import { ExternalLink } from 'lucide-react';
import { Api } from '../../../services/api';
import type { ChangelogEntry, MigrationRecord, SchemaMigration } from '../../../types';
import { formatCents, formatDateTime } from '../../../utils/formatters';
import {
  Button,
  Notice,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from '../../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

const MIGRATION_KIND_LABELS: Record<string, string> = {
  restore: 'Aus Sicherung wiederhergestellt',
  open: 'Vorhandene Datei geöffnet',
};

const SubHeading: React.FC<{ title: string; context: string; first?: boolean }> = ({
  title,
  context,
  first = false,
}) => (
  <div className={cn('mb-3', !first && 'mt-8')}>
    <h4 className="text-label text-ink">{title}</h4>
    <p className="text-caption text-ink-subtle mt-0.5">{context}</p>
  </div>
);

/**
 * Die Geschichte des Systems: Schemaläufe, Datenübernahmen und Fassungen. Ein
 * fehlgeschlagener Schemalauf oder eine Übernahme mit gebrochener Kette ist
 * ein Befund und steht deshalb bei der Prüfung.
 */
export const HistoryPanel: React.FC = () => {
  const [changelog, setChangelog] = useState<ChangelogEntry[]>([]);
  const [schema, setSchema] = useState<SchemaMigration[]>([]);
  const [records, setRecords] = useState<MigrationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([Api.getChangeLog(), Api.getSchemaMigrations(), Api.getMigrationRecords()])
      .then(([log, migrations, migrationRecords]) => {
        setChangelog(log);
        setSchema(migrations);
        setRecords(migrationRecords);
      })
      .catch((e) => setError(message(e)))
      .finally(() => setLoading(false));
  }, []);

  async function openReleases() {
    try {
      await Api.openReleasesPage();
    } catch (e) {
      setError(message(e));
    }
  }

  if (loading) return <SkeletonRows rows={6} />;

  return (
    <div>
      {error && <Notice tone="negative" text={error} className="mb-5" />}

      <SubHeading first title="Schemaänderungen" context="Anpassung der Datenbank an jede Programmfassung" />
      {schema.length === 0 ? (
        <p className="text-body text-ink-muted">Noch kein Lauf protokolliert.</p>
      ) : (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-48">Zeitpunkt</Th>
              <Th className="w-24">Fassung</Th>
              <Th numeric className="w-24">Schema</Th>
              <Th className="w-32">Ergebnis</Th>
              <Th>Tabellen</Th>
            </Tr>
          </Thead>
          <Tbody>
            {schema.map((run) => (
              <Tr key={run.id}>
                <Td className="text-ink-subtle num">{formatDateTime(run.runAt)}</Td>
                <Td className="num">{run.appVersion}</Td>
                <Td numeric>
                  {run.fromVersion} → {run.toVersion}
                </Td>
                <Td className={run.result === 'ok' ? 'text-ink-muted' : 'text-negative-text'}>
                  {run.result === 'ok' ? 'Ausgeführt' : 'Fehlgeschlagen'}
                </Td>
                <Td className="whitespace-normal text-ink-muted">{run.tables || run.message}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      <SubHeading title="Datenübernahmen" context="Zählung und Kettenprüfung beim Öffnen oder Wiederherstellen" />
      {records.length === 0 ? (
        <p className="text-body text-ink-muted">Keine Übernahme protokolliert.</p>
      ) : (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-48">Zeitpunkt</Th>
              <Th>Art</Th>
              <Th numeric className="w-24">Buchungen</Th>
              <Th numeric className="w-32">Soll</Th>
              <Th numeric className="w-32">Haben</Th>
              <Th className="w-28">Kette</Th>
            </Tr>
          </Thead>
          <Tbody>
            {records.map((record) => (
              <Tr key={record.id}>
                <Td className="text-ink-subtle num">{formatDateTime(record.runAt)}</Td>
                <Td className="text-ink-muted">{MIGRATION_KIND_LABELS[record.kind] ?? record.kind}</Td>
                <Td numeric>{record.journalEntries}</Td>
                <Td numeric>{formatCents(record.debitTotal)}</Td>
                <Td numeric>{formatCents(record.creditTotal)}</Td>
                <Td className={record.chainValid ? 'text-positive-text' : 'text-negative-text'}>
                  {record.chainValid ? 'Unversehrt' : 'Gebrochen'}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      <div className="mt-8 mb-3 flex items-end justify-between gap-4">
        <div>
          <h4 className="text-label text-ink">Fassungen</h4>
          <p className="text-caption text-ink-subtle mt-0.5">Was sich zwischen den Programmfassungen geändert hat</p>
        </div>
        <Button
          variant="quiet"
          size="sm"
          icon={<ExternalLink className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void openReleases()}
        >
          Fassungen im Netz
        </Button>
      </div>
      {changelog.length === 0 ? (
        <p className="text-body text-ink-muted">Keine Versionshistorie hinterlegt.</p>
      ) : (
        <div className="space-y-6 max-w-3xl">
          {changelog.map((entry) => (
            <div key={`${entry.version}-${entry.date}`}>
              <h5 className="text-label text-ink">
                {entry.version}
                <span className="ml-2 font-normal text-ink-subtle num">{entry.date}</span>
              </h5>
              {entry.summary && <p className="mt-1 text-body text-ink-muted">{entry.summary}</p>}
              <ul className="mt-2 space-y-1">
                {entry.changes.map((change) => (
                  <li key={change} className="flex gap-2.5 text-body text-ink-muted">
                    <span className="mark-diamond bg-line-strong mt-2 shrink-0" />
                    <span>{change}</span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
