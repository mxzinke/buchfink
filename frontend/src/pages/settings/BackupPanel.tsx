import React, { useEffect, useState } from 'react';
import { CheckCircle2, FolderOpen, OctagonAlert, RotateCcw, Save, ShieldCheck } from 'lucide-react';
import { Api } from '../../services/api';
import type { AppConfig, BackupKind, BackupRun } from '../../types';
import { formatBytes, formatDateTime } from '../../utils/formatters';
import {
  Button,
  Disclosure,
  FieldValue,
  Notice,
  Section,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Warum ein Knopf gerade nicht geht (§10.4). */
const RUNNING_HINT = 'Ein anderer Lauf der Sicherung ist gerade aktiv.';

const KIND_LABELS: Record<BackupKind, string> = {
  manual: 'Von Hand',
  automatic: 'Automatisch',
  verify: 'Prüfung',
  restore: 'Wiederherstellung',
};

type Busy = '' | 'create' | 'verify' | 'restore' | 'dir';

interface BackupPanelProps {
  id?: string;
  appConfig: AppConfig;
  /** Die Konfiguration hat sich geändert, etwa der Sicherungsordner. */
  onAppConfigChange: (config: AppConfig) => void;
  /** Nach einer Wiederherstellung: die Mandantenliste neu lesen. */
  onRestored: () => void | Promise<void>;
}

/**
 * Die Sicherung: wohin, wann zuletzt, jetzt sichern. Prüfen, Wiederherstellen
 * und die Liste der Läufe stehen eingeklappt darunter.
 */
export const BackupPanel: React.FC<BackupPanelProps> = ({
  id,
  appConfig,
  onAppConfigChange,
  onRestored,
}) => {
  const [busy, setBusy] = useState<Busy>('');
  const [runs, setRuns] = useState<BackupRun[] | null>(null);
  // Ein Fehler bleibt am Abschnitt stehen: die Sicherung läuft lange, und ein
  // Toast wäre vorbei, bevor jemand den Pfad darin gelesen hat (§10.4).
  const [error, setError] = useState('');

  useEffect(() => {
    Api.getBackupRuns()
      .then(setRuns)
      .catch((e) => {
        setRuns([]);
        setError(message(e));
      });
  }, []);

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

  const pickDir = () =>
    run('dir', async () => {
      const selected = await Api.selectBackupDir('Ordner für die Sicherung wählen, am besten auf einem anderen Laufwerk');
      if (!selected) return;
      onAppConfigChange(await Api.setBackupDir(selected));
      toast.success('Sicherungsordner gesetzt.');
    });

  const create = () =>
    run('create', async () => {
      const result = await Api.createBackup();
      setRuns(await Api.getBackupRuns());
      onAppConfigChange(await Api.getAppConfig());
      toast.success(`Gesichert: ${result.fileCount} Dateien, ${formatBytes(result.bytes)}.`);
    });

  const verify = () =>
    run('verify', async () => {
      const path = await Api.selectBackupFile('Sicherung zum Prüfen auswählen');
      if (!path) return;
      const result = await Api.verifyBackup(path);
      setRuns(await Api.getBackupRuns());
      if (result.success) toast.success(result.message || 'Die Sicherung ist vollständig.');
      else setError(result.message || 'Die Sicherung ist nicht vollständig.');
    });

  const restore = () =>
    run('restore', async () => {
      const path = await Api.selectBackupFile('Sicherung zum Wiederherstellen auswählen');
      if (!path) return;
      const target = await Api.selectDirectoryDialog('Leeren Zielordner für die Daten wählen');
      if (!target) return;
      const tenant = await Api.restoreFromBackup(path, target);
      await onRestored();
      toast.success(`${tenant.name} wiederhergestellt und geprüft.`);
    });

  const backupDir = appConfig.backupDir ?? '';
  const lastBackupAt = appConfig.lastBackupAt ?? '';
  const latest = runs?.[0];

  return (
    <Section
      id={id}
      title="Sicherung"
      context={
        lastBackupAt ? `Zuletzt gesichert am ${formatDateTime(lastBackupAt)}` : 'Noch keine Sicherung gelaufen'
      }
      helpSummary="Eine Sicherung ist eine Kopie der Buchführung mit allen Belegen in einer Datei."
      explain={
        <>
          Buchfink sichert beim Start, wenn die letzte Sicherung mindestens einen Tag zurückliegt, und
          beim Beenden, wenn sich etwas geändert hat. Die Datei ist nicht verschlüsselt: Bewahren Sie
          sie so sorgfältig auf wie die Daten selbst, auf einem anderen Laufwerk. Die Buchführung ist
          zehn Jahre aufzubewahren (§ 147 Abs. 3 AO).
        </>
      }
      action={
        <Button
          variant="secondary"
          loading={busy === 'create'}
          disabled={!backupDir || (busy !== '' && busy !== 'create')}
          title={!backupDir ? 'Zuerst einen Sicherungsordner wählen.' : busy ? RUNNING_HINT : undefined}
          icon={<Save className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void create()}
        >
          Jetzt sichern
        </Button>
      }
    >
      {error && <Notice tone="negative" className="mb-5" text={error} />}
      {!backupDir && (
        <Notice
          className="mb-5"
          text="Ohne Sicherungsordner sichert Buchfink nicht."
          action={
            <Button variant="secondary" size="sm" loading={busy === 'dir'} onClick={() => void pickDir()}>
              Ordner wählen
            </Button>
          }
        />
      )}

      {backupDir && (
        <div className="flex flex-wrap items-center gap-x-4 max-w-[900px]">
          <span className="text-label text-ink-muted w-36">Sicherungsordner</span>
          <FieldValue className="code-num flex-1 truncate" >{backupDir}</FieldValue>
          <Button
            variant="quiet"
            size="sm"
            loading={busy === 'dir'}
            icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => void pickDir()}
          >
            Ändern
          </Button>
        </div>
      )}

      <div className="mt-4 max-w-[900px] border-t border-line">
        <Disclosure
          variant="row"
          title="Prüfen und wiederherstellen"
          context="Eine Sicherung probeweise zurückspielen oder als neuen Mandanten anlegen"
        >
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              loading={busy === 'verify'}
              disabled={busy !== '' && busy !== 'verify'}
              title={busy ? RUNNING_HINT : 'Ihre Daten ändern sich dabei nicht.'}
              icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => void verify()}
            >
              Sicherung prüfen
            </Button>
            <Button
              variant="secondary"
              loading={busy === 'restore'}
              disabled={busy !== '' && busy !== 'restore'}
              title={busy ? RUNNING_HINT : 'Der Bestand kommt in einen leeren Ordner; der geöffnete Mandant bleibt.'}
              icon={<RotateCcw className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => void restore()}
            >
              Aus Sicherung wiederherstellen
            </Button>
          </div>
        </Disclosure>
        <Disclosure
          variant="row"
          title="Letzte Läufe"
          context={
            runs === null
              ? 'Wird gelesen'
              : latest
                ? `${runs.length} ${runs.length === 1 ? 'Lauf' : 'Läufe'} · zuletzt ${latest.success ? 'ohne Beanstandung' : 'fehlgeschlagen'}`
                : 'Noch kein Lauf'
          }
        >
          {runs === null ? (
            <SkeletonRows rows={3} />
          ) : runs.length === 0 ? (
            <p className="text-body text-ink-muted">Die erste Sicherung entsteht mit „Jetzt sichern“ oder beim Beenden.</p>
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-48">Zeitpunkt</Th>
                  <Th className="w-36">Anlass</Th>
                  <Th numeric className="w-24">Dateien</Th>
                  <Th numeric className="w-24">Größe</Th>
                  <Th>Ergebnis</Th>
                </Tr>
              </Thead>
              <Tbody>
                {runs.slice(0, 10).map((r) => (
                  <Tr key={r.id}>
                    <Td className="text-ink-subtle num">{formatDateTime(r.startedAt)}</Td>
                    <Td className="text-ink-muted">{KIND_LABELS[r.kind] ?? r.kind}</Td>
                    <Td numeric>{r.fileCount}</Td>
                    <Td numeric>{formatBytes(r.bytes)}</Td>
                    <Td
                      className={cn('text-caption', r.success ? 'text-positive-text' : 'text-negative-text')}
                      title={r.message || r.target}
                    >
                      <span className="flex items-center gap-1.5">
                        {r.success ? (
                          <CheckCircle2 className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                        ) : (
                          <OctagonAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                        )}
                        {r.success ? 'Ohne Beanstandung' : 'Fehlgeschlagen'}
                      </span>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </Disclosure>
      </div>
    </Section>
  );
};
