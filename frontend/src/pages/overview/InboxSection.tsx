import React, { useCallback, useEffect, useState } from 'react';
import { FolderOpen, Plus } from 'lucide-react';
import { Api } from '../../services/api';
import type { InboxFile, InboxInfo } from '../../types';
import { formatBytes, formatDateOfTimestamp } from '../../utils/formatters';
import { INBOX_KIND_LABELS, INBOX_PROMPT, dropFeedback, inboxContext, inboxStale } from '../../utils/inbox';
import type { NavigateFn } from '../../navigation';
import { useLiveReload } from '../../live';
import { showDropFeedback } from '../../components/InboxDrop';
import {
  Button,
  Disclosure,
  Notice,
  Section,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Der Eingang in der Übersicht: wie viel dort liegt und wie alt die älteste
 * Datei ist. Abgelegt wird nichts von selbst; Claude legt ab, wenn der Mensch
 * es sagt. Ist die älteste Datei zwei Wochen alt, erinnert die Übersicht
 * daran. Dateien kommen über
 * den Dateimanager, das ganze Fenster (Drag & Drop) oder „Dateien hinzufügen“
 * hinein.
 */
export const InboxSection: React.FC<{ tenantId: string | undefined; onNavigate: NavigateFn }> = ({
  tenantId,
  onNavigate,
}) => {
  const [info, setInfo] = useState<InboxInfo | null>(null);
  const [files, setFiles] = useState<InboxFile[] | null>(null);
  const [filesOpen, setFilesOpen] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState<string>('');

  const load = useCallback(async () => {
    try {
      setInfo(await Api.getInbox());
      setError('');
    } catch (e) {
      setError(`Der Eingang ließ sich nicht lesen. ${message(e)}`);
    }
  }, []);

  const loadFiles = useCallback(async () => {
    try {
      setFiles(await Api.listInbox());
    } catch (e) {
      setFiles([]);
      setError(`Der Eingang ließ sich nicht lesen. ${message(e)}`);
    }
  }, []);

  const reload = useCallback(() => {
    void load();
    if (filesOpen) void loadFiles();
  }, [load, loadFiles, filesOpen]);

  // Ein anderer Mandant hat einen anderen Eingang: die aufgeklappte Liste
  // gehörte zum vorigen.
  useEffect(() => {
    setInfo(null);
    setFiles(null);
    setFilesOpen(false);
    void load();
  }, [load, tenantId]);

  useEffect(() => {
    if (filesOpen) void loadFiles();
  }, [filesOpen, loadFiles]);

  // Was jemand im Dateimanager oder per Scanner-App hineinlegt, meldet kein
  // Ereignis; zurück im Fenster liest die Übersicht den Eingang neu.
  useEffect(() => {
    window.addEventListener('focus', reload);
    return () => window.removeEventListener('focus', reload);
  }, [reload]);

  useLiveReload(reload, ['eingang', 'belege', 'bank']);

  async function openFolder() {
    setBusy('open');
    try {
      await Api.openInbox();
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy('');
    }
  }

  async function addFiles() {
    setBusy('add');
    try {
      const paths = await Api.selectReceiptFiles('Dateien in den Eingang legen');
      if (paths.length === 0) return;
      showDropFeedback(dropFeedback(await Api.addToInbox(paths)));
      reload();
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy('');
    }
  }

  // Den Kontoauszug liest der Mensch hier selbst ein; den ersten eines Kontos
  // darf nur er einlesen, weil erst er den Saldo festlegt, an dem Buchfink die
  // Auszüge des Agenten prüft.
  async function importStatement(file: InboxFile) {
    setBusy(file.path);
    try {
      const count = await Api.importCAMTFile(file.path, '');
      toast.success(`${file.name}: ${count} ${count === 1 ? 'Umsatz' : 'Umsätze'} eingelesen.`);
      reload();
    } catch (e) {
      toast.error(message(e));
    } finally {
      setBusy('');
    }
  }

  const stale = info ? inboxStale(info) : false;

  return (
    <Section
      id="eingang"
      title="Eingang"
      context={info && !error ? inboxContext(info) : 'Belege, E-Rechnungen und Kontoauszüge'}
      helpSummary="Hier sammeln sich Belege, bis Claude sie auf Ihre Anweisung ablegt."
      explain={
        <>
          Legen Sie Belege, E-Rechnungen und Kontoauszüge in den Eingang: im Dateimanager, mit einer
          Scanner-App oder indem Sie die Dateien in dieses Fenster ziehen. Buchfink legt nichts von
          selbst ab. Sagen Sie Claude, wann es so weit ist, etwa „{INBOX_PROMPT}“ Was Claude
          abgelegt hat, verschiebt Buchfink in den Unterordner Übernommen, nach Monaten sortiert. Den
          ersten Kontoauszug eines Bankkontos lesen Sie selbst mit „Einlesen“ ein; erst er legt den Saldo
          fest, an dem Buchfink spätere Auszüge prüft. Den Ordner wählen Sie in den Einstellungen.
        </>
      }
      action={
        <div className="flex flex-wrap justify-end gap-2">
          <Button
            variant="quiet"
            loading={busy === 'add'}
            disabled={!info || busy !== ''}
            icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => void addFiles()}
          >
            Dateien hinzufügen
          </Button>
          <Button
            variant="secondary"
            loading={busy === 'open'}
            disabled={!info || busy !== ''}
            icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => void openFolder()}
          >
            Ordner öffnen
          </Button>
        </div>
      }
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !info ? (
        <SkeletonRows rows={2} />
      ) : (
        <>
          {stale ? (
            <Notice text={`Sagen Sie Claude: „${INBOX_PROMPT}“`} />
          ) : info.count > 0 ? (
            <p className="text-body text-ink-muted">
              {`Claude legt die Dateien ab, wenn Sie es sagen: „${INBOX_PROMPT}“`}
            </p>
          ) : (
            <p className="text-body text-ink-muted">
              Ziehen Sie Belege, E-Rechnungen und Kontoauszüge in dieses Fenster oder legen Sie sie in den
              Ordner.
            </p>
          )}

          {info.count > 0 && (
            <div className="mt-4 border-t border-line">
              <Disclosure
                variant="row"
                title="Dateien im Eingang"
                context="Die älteste zuerst"
                open={filesOpen}
                onOpenChange={setFilesOpen}
              >
                {files === null ? (
                  <SkeletonRows rows={3} />
                ) : (
                  <Table density="kompakt">
                    <Thead>
                      <Tr>
                        <Th>Datei</Th>
                        <Th className="w-28">Art</Th>
                        <Th optional className="w-28">
                          Geändert
                        </Th>
                        <Th numeric optional className="w-24">
                          Größe
                        </Th>
                        <Th className="w-32" aria-label="Beleg" />
                      </Tr>
                    </Thead>
                    <Tbody>
                      {files.map((file) => (
                        <Tr key={file.path}>
                          <Td fill title={file.name}>
                            {file.name}
                          </Td>
                          <Td className="text-ink-muted">{INBOX_KIND_LABELS[file.kind]}</Td>
                          <Td optional className="num text-ink-subtle">
                            {formatDateOfTimestamp(file.modifiedAt)}
                          </Td>
                          <Td numeric optional className="text-ink-subtle">
                            {formatBytes(file.size)}
                          </Td>
                          <Td className="pl-0 text-right">
                            {file.kind === 'kontoauszug' && file.receiptId === undefined && (
                              <Button
                                variant="quiet"
                                size="sm"
                                loading={busy === file.path}
                                disabled={busy !== ''}
                                title="Liest die Umsätze ein und legt den Auszug als Beleg ab."
                                onClick={() => void importStatement(file)}
                              >
                                Einlesen
                              </Button>
                            )}
                            {file.receiptId !== undefined && (
                              <Button
                                variant="quiet"
                                size="sm"
                                title="Derselbe Inhalt liegt schon als Beleg vor."
                                onClick={() =>
                                  onNavigate({
                                    area: 'books',
                                    tab: 'receipts',
                                    id: String(file.receiptId),
                                    year: file.receiptYear,
                                  })
                                }
                              >
                                Schon abgelegt
                              </Button>
                            )}
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                )}
              </Disclosure>
            </div>
          )}

          <p className="mt-3 text-caption text-ink-subtle truncate">
            Ordner <span className="code-num">{info.path}</span>
          </p>
        </>
      )}
    </Section>
  );
};
