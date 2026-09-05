import React, { useEffect, useState } from 'react';
import {
  Archive,
  CheckCircle2,
  FileDown,
  FolderOpen,
  HardDriveDownload,
  Lock,
  OctagonAlert,
  RotateCcw,
  Save,
  ShieldCheck,
  Unlock,
} from 'lucide-react';
import {
  AppConfig,
  BackupKind,
  BackupRun,
  ExportKind,
  ExportResult,
  FileCheckResult,
  KeyDirectoryEntry,
} from '../types';
import { Api } from '../services/api';
import { formatBytes, formatDate, formatDateTime } from '../utils/formatters';
import {
  Button,
  EmptyState,
  Field,
  FieldRow,
  HelpPopover,
  Input,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
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
 * Datenzugriff und Sicherung.
 *
 * Drei Pflichten stehen hier nebeneinander, weil sie derselben Frage dienen —
 * ob die Bücher einer Prüfung standhalten: die Datenüberlassung nach § 147
 * Abs. 6 AO, der Nachweis, dass Buchungen und Belegdateien unverändert sind
 * (GoBD Rz. 110), und die Sicherung, ohne die die zehnjährige
 * Aufbewahrungsfrist eine Behauptung bleibt (§ 147 Abs. 1 AO).
 *
 * Erklärt wird hinter dem Erklärzeichen, nicht auf der Seite (§15).
 */

interface DataAccessPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile — der Vorschlag für den Export. */
  year: number;
  availableYears: number[];
  appConfig: AppConfig | null;
  /** Die Konfiguration hat sich geändert: Banner und Einstellungen ziehen nach. */
  onAppConfigChange: (config: AppConfig) => void;
  /** Nach einer Wiederherstellung: die Mandantenliste neu lesen. */
  onRestored: () => void | Promise<void>;
}

/**
 * Warum ein Knopf gerade nicht geht. Ein deaktivierter Knopf ohne Erklärung
 * versteckt seinen Grund (§10.4).
 */
const RUNNING_EXPORT_HINT = 'Ein anderer Export läuft gerade.';
const RUNNING_BACKUP_HINT = 'Ein anderer Lauf der Sicherung ist gerade unterwegs.';

const EXPORT_LABELS: Record<string, string> = {
  z3: 'Tabellen (Z3)',
  archive: 'Archiv mit Belegdateien',
  audit_package: 'Prüferpaket',
  journal: 'Journal eines Zeitraums',
  key_directory: 'Schlüsselverzeichnis',
};

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

/** Belegdatei oder Anlagendokument — beide tragen dieselbe Prüfsumme. */
function issueKindLabel(kind: string): string {
  return kind === 'document' ? 'Anlagendokument' : 'Belegdatei';
}

function issueReasonLabel(reason: string): string {
  return reason === 'missing' ? 'Fehlt' : 'Beschädigt';
}

export const DataAccessPage: React.FC<DataAccessPageProps> = ({
  year,
  availableYears,
  appConfig,
  onAppConfigChange,
  onRestored,
}) => {
  const [loading, setLoading] = useState(true);
  const [backupRuns, setBackupRuns] = useState<BackupRun[]>([]);
  const [keyDirectory, setKeyDirectory] = useState<KeyDirectoryEntry[]>([]);

  const [exportYear, setExportYear] = useState<number>(year);
  const [exportDir, setExportDir] = useState('');
  const [runningExport, setRunningExport] = useState<ExportKind | ''>('');
  const [exportResult, setExportResult] = useState<ExportResult | null>(null);

  const [fileCheck, setFileCheck] = useState<FileCheckResult | null>(null);
  const [checkingFiles, setCheckingFiles] = useState(false);

  const [backupBusy, setBackupBusy] = useState<'' | 'create' | 'verify' | 'restore' | 'dir'>('');

  const [readOnlyUntil, setReadOnlyUntil] = useState('');
  const [readOnlyReason, setReadOnlyReason] = useState('');
  const [readOnlyError, setReadOnlyError] = useState('');
  const [endReason, setEndReason] = useState('');
  const [switchingMode, setSwitchingMode] = useState(false);

  useEffect(() => {
    void loadData();
  }, []);

  // Das Jahr der Kopfzeile ist der Vorschlag; eine eigene Wahl bleibt stehen,
  // solange die Seite offen ist.
  useEffect(() => {
    setExportYear(year);
  }, [year]);

  async function loadData() {
    setLoading(true);
    try {
      const [runs, keys] = await Promise.all([Api.getBackupRuns(), Api.getKeyDirectory()]);
      setBackupRuns(runs);
      setKeyDirectory(keys);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  function report(e: unknown) {
    toast.error(e instanceof Error ? e.message : String(e));
  }

  // --- Datenüberlassung ---------------------------------------------------

  async function pickExportDir(): Promise<string> {
    const selected = await Api.selectExportDirectory('Zielordner für die Datenüberlassung wählen');
    if (selected) setExportDir(selected);
    return selected;
  }

  async function runExport(kind: 'z3' | 'archive' | 'audit_package') {
    if (runningExport) return;
    try {
      const target = exportDir || (await pickExportDir());
      // Ein abgebrochener Ordnerdialog ist keine Fehlermeldung wert.
      if (!target) return;

      setRunningExport(kind);
      setExportResult(null);
      const result =
        kind === 'z3'
          ? await Api.exportZ3(exportYear, target)
          : kind === 'archive'
            ? await Api.exportArchive(exportYear, target)
            : await Api.exportAuditPackage(exportYear, target);
      setExportResult(result);
      toast.success(`${EXPORT_LABELS[kind]} in ${result.dir} geschrieben.`);
    } catch (e) {
      report(e);
    } finally {
      setRunningExport('');
    }
  }

  async function saveKeyDirectory() {
    try {
      const path = await Api.exportKeyDirectory();
      if (path) toast.success(`Schlüsselverzeichnis gespeichert: ${path}`);
    } catch (e) {
      report(e);
    }
  }

  // --- Belegprüflauf ------------------------------------------------------

  async function checkFiles() {
    setCheckingFiles(true);
    try {
      const result = await Api.verifyReceiptFiles();
      setFileCheck(result);
      if (result.isValid) toast.success(`${result.checked} Dateien unverändert.`);
    } catch (e) {
      report(e);
    } finally {
      setCheckingFiles(false);
    }
  }

  // --- Sicherung ----------------------------------------------------------

  async function pickBackupDir() {
    setBackupBusy('dir');
    try {
      const selected = await Api.selectBackupDir(
        'Ordner für die Sicherung wählen (am besten ein anderes Laufwerk)',
      );
      if (!selected) return;
      onAppConfigChange(await Api.setBackupDir(selected));
      toast.success('Sicherungsordner gesetzt.');
    } catch (e) {
      report(e);
    } finally {
      setBackupBusy('');
    }
  }

  async function createBackup() {
    setBackupBusy('create');
    try {
      const run = await Api.createBackup();
      setBackupRuns(await Api.getBackupRuns());
      toast.success(`Sicherung geschrieben: ${run.fileCount} Dateien, ${formatBytes(run.bytes)}.`);
    } catch (e) {
      report(e);
    } finally {
      setBackupBusy('');
    }
  }

  async function verifyBackup() {
    setBackupBusy('verify');
    try {
      const path = await Api.selectBackupFile('Sicherung zum Prüfen auswählen');
      if (!path) return;
      const run = await Api.verifyBackup(path);
      setBackupRuns(await Api.getBackupRuns());
      if (run.success) toast.success(run.message || 'Die Sicherung ist zurückspielbar.');
      else toast.error(run.message || 'Die Sicherung ist nicht zurückspielbar.');
    } catch (e) {
      report(e);
    } finally {
      setBackupBusy('');
    }
  }

  async function restoreBackup() {
    setBackupBusy('restore');
    try {
      const path = await Api.selectBackupFile('Sicherung zum Wiederherstellen auswählen');
      if (!path) return;
      const target = await Api.selectDirectoryDialog('Leeren Zielordner für die Daten wählen');
      if (!target) return;
      const tenant = await Api.restoreFromBackup(path, target);
      await onRestored();
      toast.success(`${tenant.name} wiederhergestellt und geprüft.`);
    } catch (e) {
      report(e);
    } finally {
      setBackupBusy('');
    }
  }

  // --- Prüfermodus --------------------------------------------------------

  async function enableReadOnly() {
    setReadOnlyError('');
    setSwitchingMode(true);
    try {
      onAppConfigChange(await Api.enableReadOnly(readOnlyUntil, readOnlyReason));
      setReadOnlyUntil('');
      setReadOnlyReason('');
      toast.success('Prüfermodus eingeschaltet.');
    } catch (e) {
      // Der Fehler bleibt am Formular stehen, bis er behoben ist (§8.3).
      setReadOnlyError(e instanceof Error ? e.message : String(e));
    } finally {
      setSwitchingMode(false);
    }
  }

  async function disableReadOnly() {
    setReadOnlyError('');
    setSwitchingMode(true);
    try {
      onAppConfigChange(await Api.disableReadOnly(endReason));
      setEndReason('');
      toast.success('Prüfermodus beendet.');
    } catch (e) {
      setReadOnlyError(e instanceof Error ? e.message : String(e));
    } finally {
      setSwitchingMode(false);
    }
  }

  const readOnly = appConfig?.readOnly ?? false;
  const backupDir = appConfig?.backupDir ?? '';
  const lastBackupAt = appConfig?.lastBackupAt ?? '';
  const years = availableYears.length > 0 ? availableYears : [year];

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Datenzugriff & Sicherung"
        context="Datenüberlassung, Prüfläufe, Sicherung und Prüfermodus"
        action={
          <Button
            variant="primary"
            loading={runningExport === 'audit_package'}
            disabled={runningExport !== '' && runningExport !== 'audit_package'}
            title={runningExport ? RUNNING_EXPORT_HINT : undefined}
            onClick={() => void runExport('audit_package')}
            icon={<Archive className="w-4 h-4" strokeWidth={1.5} />}
          >
            Prüferpaket erzeugen
          </Button>
        }
      />

      {/* Der Streifen „Prüfermodus bis …" steht in App.tsx über jeder Ansicht:
          gesperrt ist die Anwendung und nicht diese Seite. Hier steht nur, was
          zum Schalten nötig ist. */}
      <Section
        title="Prüfermodus"
        divider={false}
        className="mt-8"
        context={
          readOnly
            ? `Aktiv bis ${formatDate(appConfig?.readOnlyUntil ?? '')} · Grund: ${appConfig?.readOnlyReason || '—'}`
            : 'Aus — die Buchführung nimmt Änderungen auf'
        }
        action={
          <HelpPopover label="Erklärung zum Prüfermodus">
            Im Prüfermodus weist Buchfink jede schreibende Bedienung ab, Auswertung und Export
            bleiben möglich. Er gilt bis zu dem Tag, den Sie angeben, und endet dann von selbst.
            Ein- und Ausschalten stehen mit Grund im Änderungsprotokoll.
          </HelpPopover>
        }
      >
        {readOnlyError && <Notice tone="negative" className="mb-4" text={readOnlyError} />}

        {readOnly ? (
          <FieldRow className="max-w-3xl">
            <Field
              label="Grund für das Beenden"
              hint="Steht im Änderungsprotokoll"
              className="flex-1 min-w-64"
            >
              <Input
                value={endReason}
                onChange={(e) => setEndReason(e.target.value)}
                placeholder="Prüfung abgeschlossen"
              />
            </Field>
            <Button
              variant="secondary"
              loading={switchingMode}
              onClick={() => void disableReadOnly()}
              icon={<Unlock className="w-4 h-4" strokeWidth={1.5} />}
              className="mt-6"
            >
              Prüfermodus beenden
            </Button>
          </FieldRow>
        ) : (
          <FieldRow className="max-w-3xl">
            <Field label="Gilt bis einschließlich" className="w-52">
              <Input
                type="date"
                value={readOnlyUntil}
                onChange={(e) => setReadOnlyUntil(e.target.value)}
              />
            </Field>
            <Field label="Grund" hint="Etwa: Außenprüfung 2022 bis 2024" className="flex-1 min-w-64">
              <Input
                value={readOnlyReason}
                onChange={(e) => setReadOnlyReason(e.target.value)}
                placeholder="Außenprüfung"
              />
            </Field>
            <Button
              variant="secondary"
              loading={switchingMode}
              onClick={() => void enableReadOnly()}
              icon={<Lock className="w-4 h-4" strokeWidth={1.5} />}
              className="mt-6"
            >
              Prüfermodus einschalten
            </Button>
          </FieldRow>
        )}
      </Section>

      <Section
        title="Datenüberlassung"
        context="Tabellen, Belegdateien und Beschreibung für die Prüfsoftware"
        action={
          <HelpPopover label="Erklärung zur Datenüberlassung">
            Der Export schreibt die Daten eines Geschäftsjahres als CSV in den gewählten Ordner,
            dazu index.xml nach dem Beschreibungsstandard, die Feldbeschreibung jeder Spalte und
            export.json mit einer Prüfsumme je Datei. Das Archiv nimmt die Belegdateien mit, das
            Prüferpaket zusätzlich den Integritätsnachweis.
          </HelpPopover>
        }
      >
        <FieldRow className="max-w-3xl">
          <Field label="Geschäftsjahr" className="w-44">
            <Select
              items={years.map((y) => ({ value: y, label: String(y) }))}
              value={exportYear}
              onValueChange={setExportYear}
            />
          </Field>
          <Field label="Zielordner" hint="Am besten ein leerer Ordner" className="flex-1 min-w-64">
            <div className="flex gap-2">
              <Input className="code-num" value={exportDir} readOnly placeholder="Noch nicht gewählt" />
              <Button
                variant="secondary"
                onClick={() => void pickExportDir()}
                icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
                className="shrink-0"
              >
                Ordner wählen
              </Button>
            </div>
          </Field>
        </FieldRow>

        <div className="flex flex-wrap gap-2 mt-5">
          <Button
            variant="secondary"
            loading={runningExport === 'z3'}
            disabled={runningExport !== '' && runningExport !== 'z3'}
            title={runningExport ? RUNNING_EXPORT_HINT : undefined}
            onClick={() => void runExport('z3')}
            icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
          >
            Nur Tabellen (Z3)
          </Button>
          <Button
            variant="secondary"
            loading={runningExport === 'archive'}
            disabled={runningExport !== '' && runningExport !== 'archive'}
            title={runningExport ? RUNNING_EXPORT_HINT : undefined}
            onClick={() => void runExport('archive')}
            icon={<Archive className="w-4 h-4" strokeWidth={1.5} />}
          >
            Mit Belegdateien
          </Button>
        </div>

        {exportResult && (
          <div className="mt-6">
            <StatRow>
              <Stat
                label="Umfang"
                value={EXPORT_LABELS[exportResult.kind] ?? exportResult.kind}
                context={`Geschäftsjahr ${exportResult.fiscalYear}`}
              />
              <Stat
                label="Dateien"
                value={String(exportResult.files.length)}
                context={`${exportResult.receiptFiles} Belegdateien · ${exportResult.documentFiles} Dokumente`}
              />
              <Stat
                label="Erzeugt"
                value={formatDateTime(exportResult.createdAt)}
                context={`Buchfink ${exportResult.programVersion} · ${exportResult.standardVersion}`}
              />
            </StatRow>

            {exportResult.notes.map((note) => (
              <Notice key={note} className="mt-5" text={note} />
            ))}

            <h3 className="text-label text-ink-muted mt-8 mb-2">Tabellen im Zielordner</h3>
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-56">Tabelle</Th>
                  <Th>Datei</Th>
                  <Th numeric className="w-32">Datensätze</Th>
                </Tr>
              </Thead>
              <Tbody>
                {exportResult.tables.map((table) => (
                  <Tr key={table.file}>
                    <Td>{table.name}</Td>
                    <Td code>{table.file}</Td>
                    <Td numeric>{table.rows.toLocaleString('de-DE')}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </div>
        )}
      </Section>

      <Section
        title="Belegprüflauf"
        context="Jede Belegdatei und jedes Anlagendokument gegen seine Prüfsumme"
        action={
          <Button
            variant="secondary"
            loading={checkingFiles}
            onClick={() => void checkFiles()}
            icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
          >
            Dateien prüfen
          </Button>
        }
      >
        {!fileCheck ? (
          <EmptyState
            title="Noch nicht geprüft"
            description="Der Lauf vergleicht jede abgelegte Datei mit der Prüfsumme, die beim Ablegen gebildet wurde."
          />
        ) : (
          <>
            <StatRow>
              <Stat
                label="Geprüfte Dateien"
                value={String(fileCheck.checked)}
                context={`Zuletzt ${formatDateTime(fileCheck.checkedAt)}`}
              />
              <Stat
                label="Unversehrt"
                value={String(fileCheck.intact)}
                tone={fileCheck.isValid ? 'positive' : 'neutral'}
              />
              <Stat
                label="Beschädigt oder fehlend"
                value={String(fileCheck.damaged + fileCheck.missing)}
                context={`${fileCheck.damaged} beschädigt · ${fileCheck.missing} fehlend`}
                tone={fileCheck.isValid ? 'neutral' : 'negative'}
              />
            </StatRow>

            {fileCheck.issues.length > 0 && (
              <div className="mt-6">
                <Table density="kompakt">
                  <Thead>
                    <Tr>
                      <Th className="w-36">Art</Th>
                      <Th className="w-32">Beleg</Th>
                      <Th>Datei</Th>
                      <Th className="w-32">Befund</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {fileCheck.issues.map((issue) => (
                      <Tr key={`${issue.path}-${issue.fileName}`}>
                        <Td className="text-ink-muted">{issueKindLabel(issue.kind)}</Td>
                        <Td code>{issue.receiptNumber || '—'}</Td>
                        <Td className="text-ink-muted">{issue.fileName}</Td>
                        <Td className="text-negative-text">
                          <span className="flex items-center gap-1.5">
                            <OctagonAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                            {issueReasonLabel(issue.reason)}
                          </span>
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </div>
            )}
          </>
        )}
      </Section>

      <Section
        title="Sicherung"
        context={
          lastBackupAt
            ? `Zuletzt gesichert am ${formatDateTime(lastBackupAt)}`
            : 'Noch keine Sicherung gelaufen'
        }
        action={
          <HelpPopover label="Erklärung zur Sicherung">
            Die Sicherung schreibt Datenbank, Schlüsseldatei, Belege und Dokumente in eine
            ZIP-Datei und läuft zusätzlich beim Beenden, wenn die letzte älter als ein Tag ist. Die
            Prüfung spielt eine vorhandene Sicherung versuchsweise in einen Temporärordner zurück
            und räumt ihn wieder ab. Bewahren Sie Sicherung und Wiederherstellungsdatei getrennt
            auf — eine Sicherung, die den Schlüssel danebenlegt, schützt vor nichts.
          </HelpPopover>
        }
      >
        {!backupDir && (
          <Notice
            className="mb-5"
            text="Es ist kein Sicherungsordner eingerichtet — bis dahin sichert Buchfink nichts."
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

        <div className="flex flex-wrap gap-2 mt-5">
          <Button
            variant="secondary"
            loading={backupBusy === 'create'}
            disabled={backupBusy !== '' && backupBusy !== 'create'}
            title={backupBusy ? RUNNING_BACKUP_HINT : undefined}
            onClick={() => void createBackup()}
            icon={<Save className="w-4 h-4" strokeWidth={1.5} />}
          >
            Jetzt sichern
          </Button>
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

        <h3 className="text-label text-ink-muted mt-8 mb-2">Letzte Läufe</h3>
        {loading ? (
          <SkeletonRows rows={4} />
        ) : backupRuns.length === 0 ? (
          <EmptyState
            icon={<HardDriveDownload className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch kein Lauf"
            description="Der erste Lauf entsteht mit „Jetzt sichern“ oder beim Beenden der Anwendung."
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

      <Section
        title="Schlüsselverzeichnis"
        context="Jeder Code, den die Daten tragen, im Klartext"
        action={
          <Button
            variant="secondary"
            onClick={() => void saveKeyDirectory()}
            icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
          >
            Als CSV speichern
          </Button>
        }
      >
        {loading ? (
          <SkeletonRows rows={6} />
        ) : keyDirectory.length === 0 ? (
          <EmptyState title="Kein Verzeichnis verfügbar" />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-56">Gruppe</Th>
                <Th className="w-44">Schlüssel</Th>
                <Th className="w-64">Klartext</Th>
                <Th>Bedeutung</Th>
              </Tr>
            </Thead>
            <Tbody>
              {keyDirectory.map((entry) => (
                <Tr key={`${entry.category}-${entry.key}`}>
                  <Td className="text-ink-subtle">{entry.category}</Td>
                  <Td code>{entry.key}</Td>
                  <Td>{entry.label}</Td>
                  <Td className="whitespace-normal text-ink-muted">{entry.description}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>
    </div>
  );
};
