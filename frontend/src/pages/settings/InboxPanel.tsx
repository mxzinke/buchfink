import React, { useCallback, useEffect, useState } from 'react';
import { FolderOpen, Plus } from 'lucide-react';
import { Api } from '../../services/api';
import type { AgentRules, AgentSettings, InboxInfo } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { useLiveReload } from '../../live';
import { Button, Disclosure, FieldValue, Notice, Section, SkeletonRows, toast } from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

type Busy = '' | 'open' | 'inbox' | 'export' | 'read';

/** Eine Zeile mit Bezeichnung, Pfad und den Knöpfen dazu, wie der Sicherungsordner. */
const FolderRow: React.FC<{ label: string; path: string; note?: string; children?: React.ReactNode }> = ({
  label,
  path,
  note,
  children,
}) => (
  <div className="flex flex-wrap items-center gap-x-4 py-1">
    <span className="w-36 text-label text-ink-muted">{label}</span>
    <span className="min-w-0 flex-1">
      <FieldValue className="code-num h-auto min-h-9 truncate">{path || '—'}</FieldValue>
      {note && <span className="block -mt-1.5 pb-1 text-caption text-ink-subtle">{note}</span>}
    </span>
    {children && <span className="flex flex-wrap gap-1">{children}</span>}
  </div>
);

/**
 * Eingang, Exportordner und – unter „Erweitert“ – die weiteren Leseordner.
 * Alle drei stellt nur der Mensch ein. Der Eingang darf auch in einem
 * Cloud-Ordner liegen, damit eine Scanner-App am Telefon dorthin speichern
 * kann; Datenordner und Sicherungen weist das Backend als Lese- oder
 * Eingangsordner ab.
 */
export const InboxPanel: React.FC<{ id?: string }> = ({ id }) => {
  const writeLock = useWriteLock();
  const [info, setInfo] = useState<InboxInfo | null>(null);
  const [rules, setRules] = useState<AgentRules | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState<Busy>('');

  const load = useCallback(async () => {
    try {
      const [nextInfo, nextRules] = await Promise.all([Api.getInbox(), Api.getAgentRules()]);
      setInfo(nextInfo);
      setRules(nextRules);
      setError('');
    } catch (e) {
      setError(`Die Ordner ließen sich nicht lesen. ${message(e)}`);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useLiveReload(load, ['stammdaten', 'eingang']);

  async function run(kind: Busy, action: () => Promise<void>) {
    setBusy(kind);
    setError('');
    try {
      await action();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy('');
    }
  }

  const setInbox = (path: string) =>
    run('inbox', async () => {
      setInfo(await Api.setInboxFolder(path));
      setRules(await Api.getAgentRules());
      toast.success(path ? 'Eingang geändert.' : 'Der Eingang liegt wieder in der Ablage.');
    });

  const pickInbox = () =>
    run('inbox', async () => {
      const selected = await Api.selectDirectoryDialog('Eingang für Belege, E-Rechnungen und Kontoauszüge wählen');
      if (!selected) return;
      setInfo(await Api.setInboxFolder(selected));
      setRules(await Api.getAgentRules());
      toast.success('Eingang geändert.');
    });

  /** Speichert die Ordner auf dem gespeicherten Stand der übrigen Regeln. */
  const saveRules = (kind: Busy, patch: Partial<AgentSettings>, done: string) =>
    run(kind, async () => {
      const fresh = await Api.getAgentRules();
      setRules(await Api.saveAgentRules({ ...fresh.stored, ...patch }));
      setInfo(await Api.getInbox());
      toast.success(done);
    });

  const pickExport = async () => {
    const selected = await Api.selectDirectoryDialog('Ordner für Exporte von Claude wählen').catch((e) => {
      setError(message(e));
      return '';
    });
    if (selected) await saveRules('export', { exportFolder: selected }, 'Exportordner geändert.');
  };

  const addReadFolder = async () => {
    const selected = await Api.selectDirectoryDialog('Weiteren Leseordner für Claude wählen').catch((e) => {
      setError(message(e));
      return '';
    });
    if (!selected || !rules) return;
    const folders = rules.stored.extraReadFolders;
    if (folders.includes(selected)) return;
    await saveRules('read', { extraReadFolders: [...folders, selected] }, 'Leseordner hinzugefügt.');
  };

  const removeReadFolder = (path: string) =>
    rules &&
    saveRules(
      'read',
      { extraReadFolders: rules.stored.extraReadFolders.filter((folder) => folder !== path) },
      'Leseordner entfernt.',
    );

  const locked = writeLock.locked || busy !== '';
  const readFolders = rules?.extraReadFolders ?? [];

  return (
    <Section
      id={id}
      title="Eingang und Ordner"
      context="Wo Belege ankommen und wohin Claude exportiert"
      helpSummary="Belege im Eingang legt Claude ab, wenn Sie es sagen."
      explain={
        <>
          In den Eingang kommen Belege, E-Rechnungen und Kontoauszüge, ob aus dem Dateimanager, einer
          Scanner-App oder per Drag & Drop ins Fenster. Claude legt sie ab, wenn Sie es sagen; danach
          verschiebt Buchfink sie in den Unterordner Übernommen. Der Eingang darf in einem Cloud-Ordner
          liegen. In den Exportordner schreibt Claude Auswertungen und Exporte. Weitere Leseordner,
          etwa für Verträge, darf Claude lesen; dort verschiebt Buchfink nichts.
        </>
      }
    >
      {error && <Notice tone="negative" className="mb-5" text={error} />}
      {!info || !rules ? (
        !error && <SkeletonRows rows={3} />
      ) : (
        <>
          <FolderRow label="Eingang" path={info.path} note={info.default ? 'In der Ablage' : undefined}>
            <Button
              variant="quiet"
              size="sm"
              loading={busy === 'open'}
              disabled={busy !== ''}
              onClick={() =>
                void run('open', async () => {
                  await Api.openInbox();
                })
              }
            >
              Ordner öffnen
            </Button>
            <Button
              variant="quiet"
              size="sm"
              loading={busy === 'inbox'}
              disabled={locked}
              title={writeLock.hint}
              icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => void pickInbox()}
            >
              Anderen Ordner wählen
            </Button>
            {!info.default && (
              <Button
                variant="quiet"
                size="sm"
                disabled={locked}
                title={writeLock.hint}
                onClick={() => void setInbox('')}
              >
                Zurück in die Ablage
              </Button>
            )}
          </FolderRow>

          <FolderRow
            label="Exportordner"
            path={rules.exportFolder}
            note={rules.stored.exportFolder ? undefined : 'In der Ablage'}
          >
            <Button
              variant="quiet"
              size="sm"
              loading={busy === 'export'}
              disabled={locked}
              title={writeLock.hint}
              icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => void pickExport()}
            >
              Anderen Ordner wählen
            </Button>
            {rules.stored.exportFolder && (
              <Button
                variant="quiet"
                size="sm"
                disabled={locked}
                title={writeLock.hint}
                onClick={() => void saveRules('export', { exportFolder: '' }, 'Der Exportordner liegt wieder in der Ablage.')}
              >
                Zurück in die Ablage
              </Button>
            )}
          </FolderRow>

          <div className="mt-4 border-t border-line">
            <Disclosure
              variant="row"
              title="Erweitert"
              context={
                readFolders.length === 0
                  ? 'Weitere Leseordner für Claude · keiner'
                  : `Weitere Leseordner für Claude · ${readFolders.length}`
              }
            >
              <p className="mb-3 text-body text-ink-muted">
                Claude darf dort Dateien lesen, etwa Verträge. Buchfink verschiebt dort nichts.
              </p>
              {readFolders.length > 0 && (
                <ul className="mb-3 flex flex-col">
                  {readFolders.map((folder) => (
                    <li key={folder} className="flex items-center gap-3 border-t border-line py-1.5 first:border-t-0">
                      <span className="min-w-0 flex-1 truncate code-num text-body text-ink" title={folder}>
                        {folder}
                      </span>
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={locked}
                        title={writeLock.hint}
                        onClick={() => void removeReadFolder(folder)}
                      >
                        Entfernen
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
              <Button
                variant="secondary"
                size="sm"
                loading={busy === 'read'}
                disabled={locked}
                title={writeLock.hint}
                icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
                onClick={() => void addReadFolder()}
              >
                Leseordner hinzufügen
              </Button>
            </Disclosure>
          </div>
        </>
      )}
    </Section>
  );
};
