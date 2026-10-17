import React, { useEffect, useState } from 'react';
import {
  CheckCircle2,
  FolderOpen,
  HardDriveDownload,
  OctagonAlert,
  RotateCcw,
  Save,
  ShieldCheck,
} from 'lucide-react';
import { Api } from '../services/api';
import type { AppConfig, BackupKind, BackupRun } from '../types';
import { formatBytes, formatDateTime } from '../utils/formatters';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  Input,
  Notice,
  PageHeader,
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
} from '../components/ui';

/**
 * Datensicherung (Welle 9).
 *
 * Die Sicherung läuft täglich und beantwortet „sind meine Daten noch da, wenn
 * die Festplatte ausfällt".
 *
 * Erklärt wird hinter dem Erklärzeichen, nicht auf der Seite (§15).
 */

interface BackupPageProps {
  appConfig: AppConfig | null;
  /** Die Konfiguration hat sich geändert: Banner und Einstellungen ziehen nach. */
  onAppConfigChange: (config: AppConfig) => void;
  /** Nach einer Wiederherstellung: die Mandantenliste neu lesen. */
  onRestored: () => void | Promise<void>;
}

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Warum ein Knopf gerade nicht geht. Ein deaktivierter Knopf ohne Erklärung
 * versteckt seinen Grund (§10.4).
 */
const RUNNING_BACKUP_HINT = 'Die Sicherung läuft gerade.';

const BACKUP_KIND_LABELS: Record<BackupKind, string> = {
  manual: 'Von Hand',
  automatic: 'Automatisch',
  verify: 'Prüfung',
  restore: 'Wiederherstellung',
};

/** Der Anlass eines Laufs, auch wenn das Backend später eine Art dazunimmt. */
function backupKindLabel(kind: BackupKind): string {
  return BACKUP_KIND_LABELS[kind] ?? kind;
}

/**
 * Die dritte Erklärstufe (§15.2). Wer eine Sicherung zurückspielen will, hat
 * gerade keinen guten Tag; das passt nicht in drei Sätze und gehört deshalb
 * hinter „Mehr erfahren" statt in die Ansicht.
 */
const BackupHelpDialog: React.FC<{ open: boolean; onClose: () => void }> = ({ open, onClose }) => (
  <Dialog
    open={open}
    onOpenChange={(next) => !next && onClose()}
    title="Sicherung und Wiederherstellung"
    footer={
      <Button variant="secondary" onClick={onClose}>
        Schließen
      </Button>
    }
  >
    <div className="text-body text-ink-muted space-y-4">
      <p>
        Eine Sicherung ist eine einzelne Datei mit einer Kopie von allem: Buchungen und Belege.
        Sie brauchen sie, wenn die Festplatte ausfällt oder der Rechner verloren geht.
      </p>
      <p>
        Die Sicherung ist nicht verschlüsselt. Jeder, der die Datei hat, kann die Buchführung öffnen.
        Bewahren Sie sie so sorgfältig auf wie die Daten selbst: auf einem anderen Laufwerk, und
        verschlüsselt, wenn sie außer Haus geht.
      </p>
      <p>
        <span className="text-ink">Sicherung prüfen</span> spielt eine vorhandene Datei probeweise
        zurück und sagt Ihnen, ob sie vollständig ist. An Ihren Daten ändert sich dabei nichts. Sich
        auf eine Sicherung verlassen kann nur, wer sie einmal ausprobiert hat.
      </p>
      <p>
        <span className="text-ink">Aus Sicherung wiederherstellen</span> legt den gesicherten Stand
        in einem leeren Ordner neu an. Der vorhandene Mandant bleibt, wie er ist. Danach prüft
        Buchfink den wiederhergestellten Bestand und meldet, ob er unversehrt ist.
      </p>
      <p>
        Aufbewahren müssen Sie die Buchführung zehn Jahre. Eine Sicherung, die nur auf demselben
        Rechner liegt, überbrückt davon keinen einzigen Tag.
      </p>
    </div>
  </Dialog>
);

export const BackupPage: React.FC<BackupPageProps> = ({
  appConfig,
  onAppConfigChange,
  onRestored,
}) => {
  const [backupBusy, setBackupBusy] = useState<'' | 'create' | 'verify' | 'restore' | 'dir'>('');
  const [backupRuns, setBackupRuns] = useState<BackupRun[]>([]);
  const [loading, setLoading] = useState(true);
  // Ein Fehler aus dem Backend gehört als Hinweisfläche über die Aktionen des
  // Abschnitts und nicht in einen Toast (§10.4): die Sicherung läuft lange, und
  // wer daneben weiterliest, hätte den Toast nach vier Sekunden verpasst —
  // samt Pfad und Grund, an denen sich der Fehler beheben lässt.
  const [loadError, setLoadError] = useState('');
  const [backupError, setBackupError] = useState('');
  const [helpOpen, setHelpOpen] = useState(false);

  useEffect(() => {
    void loadRuns();
  }, []);

  async function loadRuns() {
    setLoading(true);
    setLoadError('');
    try {
      setBackupRuns(await Api.getBackupRuns());
    } catch (e) {
      setLoadError(message(e));
    } finally {
      setLoading(false);
    }
  }

  async function pickBackupDir() {
    setBackupBusy('dir');
    setBackupError('');
    try {
      const selected = await Api.selectBackupDir(
        'Ordner für die Sicherung wählen (am besten ein anderes Laufwerk)',
      );
      if (!selected) return;
      onAppConfigChange(await Api.setBackupDir(selected));
      toast.success('Sicherungsordner gesetzt.');
    } catch (e) {
      setBackupError(message(e));
    } finally {
      setBackupBusy('');
    }
  }

  async function createBackup() {
    setBackupBusy('create');
    setBackupError('');
    try {
      const run = await Api.createBackup();
      setBackupRuns(await Api.getBackupRuns());
      toast.success(`Sicherung geschrieben: ${run.fileCount} Dateien, ${formatBytes(run.bytes)}.`);
    } catch (e) {
      setBackupError(message(e));
    } finally {
      setBackupBusy('');
    }
  }

  async function verifyBackup() {
    setBackupBusy('verify');
    setBackupError('');
    try {
      const path = await Api.selectBackupFile('Sicherung zum Prüfen auswählen');
      if (!path) return;
      const run = await Api.verifyBackup(path);
      setBackupRuns(await Api.getBackupRuns());
      if (run.success) toast.success(run.message || 'Die Sicherung ist zurückspielbar.');
      else setBackupError(run.message || 'Die Sicherung ist nicht zurückspielbar.');
    } catch (e) {
      setBackupError(message(e));
    } finally {
      setBackupBusy('');
    }
  }

  async function restoreBackup() {
    setBackupBusy('restore');
    setBackupError('');
    try {
      const path = await Api.selectBackupFile('Sicherung zum Wiederherstellen auswählen');
      if (!path) return;
      const target = await Api.selectDirectoryDialog('Leeren Zielordner für die Daten wählen');
      if (!target) return;
      const tenant = await Api.restoreFromBackup(path, target);
      await onRestored();
      toast.success(`${tenant.name} wiederhergestellt und geprüft.`);
    } catch (e) {
      setBackupError(message(e));
    } finally {
      setBackupBusy('');
    }
  }

  const backupDir = appConfig?.backupDir ?? '';
  const lastBackupAt = appConfig?.lastBackupAt ?? '';

  return (
    <div>
      <PageHeader
        title="Datensicherung"
        context="Kopien anlegen und prüfen"
        action={
          <Button
            variant="primary"
            loading={backupBusy === 'create'}
            disabled={backupBusy !== '' && backupBusy !== 'create'}
            title={backupBusy ? RUNNING_BACKUP_HINT : undefined}
            onClick={() => void createBackup()}
            icon={<Save className="w-4 h-4" strokeWidth={1.5} />}
          >
            Jetzt sichern
          </Button>
        }
      />

      {loadError && <Notice tone="negative" className="mt-6" text={loadError} />}

      <div className="mt-6">
        <Section helpSummary="Eine Sicherung enthält Ihre Buchhaltung und Unterlagen zur Wiederherstellung."
          title="Sicherung"
          divider={false}
          context={
            lastBackupAt
              ? `Zuletzt gesichert am ${formatDateTime(lastBackupAt)}`
              : 'Noch keine Sicherung gelaufen'
          }
          explain={
            <>
              Eine Sicherung ist eine Kopie von allem in einer Datei: Buchungen und Belege. Buchfink
              sichert beim Start, wenn die letzte Sicherung mindestens einen Tag zurückliegt. Beim
              Beenden sichert es außerdem Änderungen seit der letzten Sicherung. Die Sicherung ist
              nicht verschlüsselt und gehört so gut verwahrt wie die Daten selbst.
            </>
          }
          onMore={() => setHelpOpen(true)}
        >
          {backupError && <Notice tone="negative" className="mb-5" text={backupError} />}

          {!backupDir && (
            <Notice
              className="mb-5"
              text="Ohne einen Sicherungsordner läuft keine Sicherung."
            />
          )}

          <Field
            label="Sicherungsordner"
            hint="Am besten ein anderes Laufwerk"
            className="max-w-3xl"
          >
            <div className="flex gap-2">
              <Input className="code-num" value={backupDir} readOnly placeholder="Nicht eingerichtet" />
              <Button
                variant="secondary"
                loading={backupBusy === 'dir'}
                onClick={() => void pickBackupDir()}
                icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
                className="shrink-0"
              >
                Ordner wählen
              </Button>
            </div>
          </Field>

          {/* „Jetzt sichern" steht als Primäraktion in der Kopfzeile und deshalb
              nicht noch einmal hier. */}
          <div className="flex flex-wrap gap-2 mt-5">
            <Button
              variant="secondary"
              loading={backupBusy === 'verify'}
              disabled={backupBusy !== '' && backupBusy !== 'verify'}
              title={backupBusy ? RUNNING_BACKUP_HINT : undefined}
              onClick={() => void verifyBackup()}
              icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
            >
              Sicherung prüfen
            </Button>
            <Button
              variant="secondary"
              loading={backupBusy === 'restore'}
              disabled={backupBusy !== '' && backupBusy !== 'restore'}
              title={backupBusy ? RUNNING_BACKUP_HINT : undefined}
              onClick={() => void restoreBackup()}
              icon={<RotateCcw className="w-4 h-4" strokeWidth={1.5} />}
            >
              Aus Sicherung wiederherstellen
            </Button>
          </div>
        </Section>

        <Section title="Letzte Läufe" context="Sicherung, Prüfung und Wiederherstellung">
          {loading ? (
            <SkeletonRows rows={4} />
          ) : backupRuns.length === 0 ? (
            <EmptyState
              icon={<HardDriveDownload className="w-6 h-6" strokeWidth={1.5} />}
              title="Noch kein Lauf"
              description="Die erste Sicherung entsteht mit „Jetzt sichern“ oder beim Beenden."
            />
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-44">Zeitpunkt</Th>
                  <Th className="w-36">Anlass</Th>
                  <Th>Ziel</Th>
                  <Th numeric className="w-28">Dateien</Th>
                  <Th numeric className="w-28">Größe</Th>
                  <Th className="w-40">Ergebnis</Th>
                </Tr>
              </Thead>
              <Tbody>
                {backupRuns.map((run) => (
                  <Tr key={run.id}>
                    <Td className="text-ink-subtle num">{formatDateTime(run.startedAt)}</Td>
                    <Td className="text-ink-muted">{backupKindLabel(run.kind)}</Td>
                    <Td code title={run.target}>
                      {run.target || '—'}
                    </Td>
                    <Td numeric>{run.fileCount}</Td>
                    <Td numeric>{formatBytes(run.bytes)}</Td>
                    <Td
                      className={cn(
                        'text-caption',
                        run.success ? 'text-positive-text' : 'text-negative-text',
                      )}
                      title={run.message}
                    >
                      <span className="flex items-center gap-1.5">
                        {run.success ? (
                          <CheckCircle2 className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                        ) : (
                          <OctagonAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                        )}
                        {run.success ? 'Ohne Beanstandung' : 'Fehlgeschlagen'}
                      </span>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </Section>
      </div>

      <BackupHelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
    </div>
  );
};

export default BackupPage;
