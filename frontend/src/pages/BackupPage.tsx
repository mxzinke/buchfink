import React, { useCallback, useEffect, useState } from 'react';
import {
  CheckCircle2,
  FolderOpen,
  HardDriveDownload,
  OctagonAlert,
  Pause,
  Play,
  RotateCcw,
  Save,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { RETENTION_CLASS_LABELS } from '../types';
import type {
  AppConfig,
  BackupKind,
  BackupRun,
  RetentionHold,
  RetentionHoldReason,
  RetentionOverview,
  RetentionYear,
} from '../types';
import {
  formatBytes,
  formatDate,
  formatDateOfTimestamp,
  formatDateTime,
} from '../utils/formatters';
import {
  Button,
  ConfirmDialog,
  Dialog,
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
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Datensicherung (Welle 9).
 *
 * Eine Absicht, zwei Zeiträume: die Kopie für morgen und die Frist über zehn
 * Jahre. Die Sicherung läuft täglich und beantwortet „sind meine Daten noch
 * da, wenn die Festplatte ausfällt"; die Aufbewahrung beantwortet „wie lange
 * muss ich sie halten und wann darf ich sie loswerden". Beides ist die Sorge um
 * denselben Bestand und stand bis hierher auf zwei Seiten, die einander nicht
 * kannten.
 *
 * Als zwei Reiter und nicht untereinander: die eine Hälfte wird täglich
 * angesehen, die andere einmal im Jahr. Wer nachsehen will, ob die letzte
 * Sicherung durchgelaufen ist, soll dafür nicht an Löschfristen vorbeiscrollen.
 *
 * Erklärt wird hinter dem Erklärzeichen, nicht auf der Seite (§15).
 */

interface BackupPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile — der Vorschlag der Fristenansicht. */
  year: number;
  appConfig: AppConfig | null;
  /** Die Konfiguration hat sich geändert: Banner und Einstellungen ziehen nach. */
  onAppConfigChange: (config: AppConfig) => void;
  /** Nach einer Wiederherstellung: die Mandantenliste neu lesen. */
  onRestored: () => void | Promise<void>;
}

type BackupTab = 'sicherung' | 'aufbewahrung';

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
 * Ein Befund in Worten: der Stand einer Frist.
 *
 * Bewusst kein StatusBadge. Dessen Vokabular (§11.3) ist abschließend und
 * beschreibt Zustände von Buchungen, Rechnungen und Perioden; eine
 * Aufbewahrungsfrist hat keinen davon. Ein Abzeichen mit fremder Beschriftung
 * gäbe ihr die Bedeutung des fremden Zustands mit: Bernstein „Offen" heißt
 * „noch zu erledigen" — das trifft auf eine laufende Frist nicht zu.
 */
const VERDICT_TONE = {
  neutral: 'text-ink-muted',
  attention: 'text-attention-text',
} as const;

const Verdict: React.FC<{ text: string; tone?: keyof typeof VERDICT_TONE }> = ({
  text,
  tone = 'neutral',
}) => <span className={`text-body whitespace-nowrap ${VERDICT_TONE[tone]}`}>{text}</span>;

/**
 * Die dritte Erklärstufe (§15.2). Wer eine Sicherung zurückspielen will, hat
 * gerade keinen guten Tag; das passt nicht in drei Sätze und gehört deshalb
 * hinter „Mehr dazu" statt in die Ansicht.
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
        Eine Sicherung ist eine einzelne Datei mit einer Kopie von allem: Buchungen, Belege,
        Dokumente und der Schlüssel, mit dem sich die Daten öffnen lassen. Sie brauchen sie, wenn
        die Festplatte ausfällt oder der Rechner verloren geht.
      </p>
      <p>
        Weil der Schlüssel mit darin liegt, kann jeder die Buchführung öffnen, der die Datei hat.
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
  year,
  appConfig,
  onAppConfigChange,
  onRestored,
}) => {
  // Der Reiter ist Zustand dieser Seite und kein Navigationsziel: die Aufgaben,
  // die hierher führen, meinen alle die Sicherung. Ein Parameter, den keine
  // Stelle setzt, wäre nur ein Versprechen.
  const [tab, setTab] = useState<BackupTab>('sicherung');

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
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      {/* Jeder Reiter hat seine eigene erste Handlung (§10.4). Die Aufbewahrung
          hat keine: dort wird gelesen, und das Aussetzen einer Frist steht am
          Abschnitt, den es betrifft. */}
      <PageHeader
        title="Datensicherung"
        context="Kopien anlegen und Aufbewahrungsfristen im Blick behalten"
        action={
          tab === 'sicherung' ? (
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
          ) : undefined
        }
      />

      {loadError && <Notice tone="negative" className="mt-6" text={loadError} />}

      <Tabs<BackupTab>
        value={tab}
        onValueChange={setTab}
        className="mt-6"
        items={[
          { value: 'sicherung', label: 'Sicherung' },
          { value: 'aufbewahrung', label: 'Aufbewahrung' },
        ]}
      >
        <TabPanel value="sicherung">
          <Section
            title="Sicherung"
            divider={false}
            context={
              lastBackupAt
                ? `Zuletzt gesichert am ${formatDateTime(lastBackupAt)}`
                : 'Noch keine Sicherung gelaufen'
            }
            explain={
              <>
                Eine Sicherung ist eine Kopie von allem in einer Datei: Buchungen, Belege und der
                Schlüssel dazu. Sie läuft beim Beenden von selbst, höchstens einmal am Tag. Weil der
                Schlüssel mit darin liegt, gehört sie so gut verwahrt wie die Daten selbst.
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
        </TabPanel>

        {/* Die Fristen rechnen über alle Jahre und werden einmal im Jahr
            gebraucht: sie laden erst, wenn der Reiter offen ist. */}
        <TabPanel value="aufbewahrung">
          {tab === 'aufbewahrung' && <RetentionPanel year={year} />}
        </TabPanel>
      </Tabs>

      <BackupHelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
    </div>
  );
};

// -------------------------------------------------------------------------
// Aufbewahrung: Fristen, Aussetzung, Löschung
// -------------------------------------------------------------------------

const HOLD_REASONS: { value: RetentionHoldReason; label: string }[] = [
  { value: 'audit', label: 'Außenprüfung' },
  { value: 'appeal', label: 'Rechtsbehelfsverfahren' },
  { value: 'other', label: 'Sonstiger Grund' },
];

function holdReasonLabel(reason: RetentionHoldReason): string {
  return HOLD_REASONS.find((entry) => entry.value === reason)?.label ?? 'Sonstiger Grund';
}

const RetentionPanel: React.FC<{ year: number }> = ({ year }) => {
  const lock = useWriteLock();
  const [overview, setOverview] = useState<RetentionOverview | null>(null);
  const [holds, setHolds] = useState<RetentionHold[]>([]);
  const [expired, setExpired] = useState<RetentionYear[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  // Die Aussetzung, die Aufhebung und die Löschung führen je einen eigenen
  // Dialog: sie fragen Verschiedenes, und ein gemeinsamer Dialog mit drei
  // Zuständen wäre an jeder Stelle halb falsch.
  //
  // Jeder Dialog führt seinen eigenen Fehlerzustand. Der Fehler der Seite
  // stünde hinter dem Backdrop des offenen Dialogs, und der Dialog bliebe
  // ohne Meldung stehen — ein Klick, der sichtbar nichts bewirkt (§10.4).
  const [holdYear, setHoldYear] = useState<number | null>(null);
  const [holdReason, setHoldReason] = useState<RetentionHoldReason>('audit');
  const [holdText, setHoldText] = useState('');
  const [holdError, setHoldError] = useState('');
  const [releasing, setReleasing] = useState<RetentionHold | null>(null);
  const [releaseText, setReleaseText] = useState('');
  const [releaseError, setReleaseError] = useState('');
  const [deleting, setDeleting] = useState<RetentionYear | null>(null);
  const [confirmation, setConfirmation] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleteError, setDeleteError] = useState('');
  // Das Löschprotokoll bleibt stehen, bis es weggeklickt wird. Es ist ein
  // mehrzeiliges Ergebnis mit Zählungen und dem Pfad des Archivs — also etwas,
  // das weiterverwendet wird, und kein Vollzug, den man ohnehin sieht (§10.4).
  // Als Toast wäre der Archivpfad nach vier Sekunden nur noch im
  // Änderungsprotokoll zu finden.
  const [deleteReport, setDeleteReport] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [view, holdList, expiredYears] = await Promise.all([
        Api.getRetentionOverview(0),
        Api.getRetentionHolds(),
        Api.getExpiredObjects(),
      ]);
      setOverview(view);
      setHolds(holdList);
      setExpired(expiredYears);
    } catch (e) {
      setOverview(null);
      setHolds([]);
      setExpired([]);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const years = overview?.years ?? [];
  const activeHolds = holds.filter((hold) => !hold.releasedAt);
  const suggestedYear = years.some((row) => row.fiscalYear === year)
    ? year
    : (years[0]?.fiscalYear ?? year);

  async function saveHold() {
    if (holdYear === null) return;
    if (holdReason === 'other' && holdText.trim() === '') {
      setHoldError('Ein sonstiger Grund ist zu beschreiben — sonst lässt sich später nicht beurteilen, ob er noch gilt.');
      return;
    }
    setBusy(true);
    setHoldError('');
    try {
      await Api.setRetentionHold(holdYear, holdReason, holdText.trim());
      setHoldYear(null);
      setHoldText('');
      await load();
    } catch (e) {
      setHoldError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function release() {
    if (!releasing) return;
    if (releaseText.trim() === '') {
      setReleaseError('Zur Aufhebung gehört ein Grund; er bleibt im Protokoll stehen.');
      return;
    }
    setBusy(true);
    setReleaseError('');
    try {
      await Api.releaseRetentionHold(releasing.id, releaseText.trim());
      setReleasing(null);
      setReleaseText('');
      await load();
    } catch (e) {
      setReleaseError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function archiveAndDelete() {
    if (!deleting) return;
    setBusy(true);
    setDeleteError('');
    try {
      const result = await Api.archiveAndDeleteFiscalYear(deleting.fiscalYear, confirmation.trim());
      const deletedYear = deleting.fiscalYear;
      setDeleting(null);
      setConfirmation('');
      await load();
      setDeleteReport(result.message);
      // Der Toast sagt nur, dass es geschehen ist; was geschehen ist, steht in
      // der Hinweisfläche darunter.
      toast.success(`Geschäftsjahr ${deletedYear} archiviert und gelöscht.`);
    } catch (e) {
      // Der Dialog bleibt mit seiner Eingabe stehen: die Ablehnung des
      // Backends — noch laufende Frist, gescheiterter Archivexport — nennt
      // einen Grund, der hier zu lesen und nicht wegzuklicken ist.
      setDeleteError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      <StatRow>
        <Stat label="Geschäftsjahre" value={String(years.length)} context="mit Daten" />
        {/* Die abgelaufenen Objekte stehen als Jahreszahlen dabei und nicht
            nur als Anzahl: „3 abgelaufen" verlangt eine zweite Frage, „2013,
            2014, 2015" beantwortet sie gleich mit. */}
        <Stat
          label="Frist abgelaufen"
          value={String(expired.length)}
          context={
            // Bei null abgelaufenen Jahren beschreibt „löschbar nach Archiv"
            // nichts — es gibt nichts zu löschen.
            expired.length > 0 ? expired.map((row) => row.fiscalYear).join(', ') : 'keines'
          }
        />
        <Stat
          label={
            <>
              Fristen ausgesetzt
              <HelpPopover label="Erklärung zur ausgesetzten Frist">
                Die Aufbewahrungsfrist läuft nicht ab, solange die Unterlagen für eine noch offene Festsetzung von Bedeutung sind (§ 147 Abs. 3 Satz 5 AO).
              </HelpPopover>
            </>
          }
          value={String(activeHolds.length)}
          context="laufende Aussetzungen"
        />
        <Stat
          label="Stichtag"
          value={formatDate(overview?.today ?? '')}
          context="Grundlage der Fristen"
        />
      </StatRow>

      {/* Der Umstellungszeitpunkt gehört auf die Fristenseite (Entscheidung 13):
          die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO ist eine
          Aufbewahrungsfrist und keine Frage der Versionshistorie. Wer hier
          liest, wann gelöscht werden darf, muss sie sehen. Gepflegt wird der
          Zeitpunkt in den Einstellungen, wo die übrigen Unternehmensangaben
          stehen. */}
      {overview?.systemChangeNote && (
        <Notice className="mt-6" text={overview.systemChangeNote} />
      )}

      {/* Das Löschprotokoll: Zählungen und Archivpfad bleiben stehen, bis sie
          zur Kenntnis genommen sind. */}
      {deleteReport && (
        <Notice
          className="mt-6"
          text={deleteReport}
          action={
            <Button variant="quiet" size="sm" onClick={() => setDeleteReport(null)}>
              Verstanden
            </Button>
          }
        />
      )}

      <Section
        title="Fristen je Geschäftsjahr"
        context="Es gilt die längste Frist des Jahres, weil das Löschen das ganze Jahr trifft"
        explain={
          <>
            Die Frist beginnt mit dem Schluss des Kalenderjahres, in dem die letzte Eintragung
            gemacht oder der Beleg entstanden ist (§ 257 Abs. 5 HGB, § 147 Abs. 4 AO). Handelsbücher
            und Abschlüsse sind zehn Jahre aufzubewahren, Buchungsbelege und Rechnungen seit 2025
            acht, Handelsbriefe sechs. Gelöscht werden darf ab dem Tag nach dem Fristende.
          </>
        }
        action={
          <Button
            variant="secondary"
            disabled={lock.locked || years.length === 0}
            // Ein gesperrter Knopf ohne Erklärung versteckt seinen Grund
            // (§10.4): der Prüfermodus ist der eine, ein Mandant ohne
            // Geschäftsjahr mit Daten der andere.
            title={
              lock.hint ??
              (years.length === 0
                ? 'Es gibt noch kein Geschäftsjahr mit Daten, dessen Frist ausgesetzt werden könnte.'
                : undefined)
            }
            onClick={() => {
              setHoldYear(suggestedYear);
              setHoldReason('audit');
              setHoldText('');
              setHoldError('');
            }}
            icon={<Pause className="w-4 h-4" strokeWidth={1.5} />}
          >
            Frist aussetzen
          </Button>
        }
      >
        {years.length === 0 ? (
          <EmptyState
            title="Noch kein Geschäftsjahr mit Daten"
            description="Die Fristen entstehen mit der ersten Buchung."
          />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-24" numeric>
                  Jahr
                </Th>
                <Th numeric className="w-28">
                  Buchungen
                </Th>
                <Th numeric className="w-24">
                  Belege
                </Th>
                <Th className="w-72">Fristen</Th>
                <Th className="w-36">Löschbar ab</Th>
                <Th className="w-40">Stand</Th>
                <Th className="w-44" />
              </Tr>
            </Thead>
            <Tbody>
              {years.map((row) => (
                <Tr key={row.fiscalYear}>
                  <Td numeric className="num">
                    {row.fiscalYear}
                  </Td>
                  <Td numeric className="num">
                    {row.counts.journalEntries}
                  </Td>
                  <Td numeric className="num">
                    {row.counts.receipts}
                  </Td>
                  <Td className="whitespace-normal text-ink-muted">
                    {row.classes.map((entry) => (
                      <span key={entry.class} className="block text-caption">
                        {RETENTION_CLASS_LABELS[entry.class]} · {entry.years} Jahre · bis{' '}
                        {formatDate(entry.retentionEnd)}
                      </span>
                    ))}
                  </Td>
                  <Td className="num text-ink-muted">{formatDate(row.earliestDeletion)}</Td>
                  <Td>
                    {row.hold ? (
                      <Verdict text="Frist ausgesetzt" tone="attention" />
                    ) : row.deletable ? (
                      <Verdict text="Frist abgelaufen" />
                    ) : (
                      <Verdict text="Aufzubewahren" />
                    )}
                  </Td>
                  <Td>
                    <Button
                      variant="quiet"
                      size="sm"
                      disabled={lock.locked || !row.deletable}
                      title={
                        lock.hint ??
                        (row.deletable
                          ? undefined
                          : row.hold
                            ? `Die Frist ist seit dem ${formatDateOfTimestamp(row.hold.setAt)} ausgesetzt (${holdReasonLabel(row.hold.reason)}).`
                            : `Löschbar ab ${formatDate(row.earliestDeletion)}.`)
                      }
                      onClick={() => {
                        setDeleting(row);
                        setConfirmation('');
                        setDeleteError('');
                      }}
                      icon={<Trash2 className="w-3.5 h-3.5" strokeWidth={1.5} />}
                    >
                      Archivieren & löschen
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section title="Ausgesetzte Fristen" context="Gesetzte und aufgehobene Aussetzungen">
        {holds.length === 0 ? (
          <EmptyState
            title="Keine Frist ausgesetzt"
            description="Eine Aussetzung hält die Frist an, solange ein Verfahren läuft."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th numeric className="w-20">
                  Jahr
                </Th>
                <Th className="w-44">Grund</Th>
                <Th>Erläuterung</Th>
                <Th className="w-52">Gesetzt</Th>
                <Th className="w-52">Aufgehoben</Th>
                <Th className="w-28" />
              </Tr>
            </Thead>
            <Tbody>
              {holds.map((hold) => (
                // Eine aufgehobene Aussetzung bleibt stehen: dass eine Frist
                // einmal ausgesetzt war, gehört zur Geschichte der Daten.
                <Tr key={hold.id}>
                  <Td numeric className="num">
                    {hold.fiscalYear}
                  </Td>
                  <Td className="text-ink-muted">{holdReasonLabel(hold.reason)}</Td>
                  <Td className="whitespace-normal">
                    {hold.description || '—'}
                    {hold.affectedNote && (
                      <span className="block text-caption text-ink-subtle">
                        {hold.affectedNote}
                      </span>
                    )}
                  </Td>
                  <Td className="text-ink-subtle num">
                    {formatDateTime(hold.setAt)}
                    <span className="block text-caption">{hold.setBy}</span>
                  </Td>
                  <Td className="text-ink-subtle num">
                    {hold.releasedAt ? (
                      <>
                        {formatDateTime(hold.releasedAt)}
                        <span className="block text-caption">{hold.releasedBy}</span>
                      </>
                    ) : (
                      '—'
                    )}
                  </Td>
                  <Td>
                    {!hold.releasedAt && (
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={lock.locked}
                        title={lock.hint}
                        onClick={() => {
                          setReleasing(hold);
                          setReleaseText('');
                          setReleaseError('');
                        }}
                        icon={<Play className="w-3.5 h-3.5" strokeWidth={1.5} />}
                      >
                        Aufheben
                      </Button>
                    )}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Löschkonzept"
        context="Datenkategorie, Frist und Rechtsgrundlage"
        explain={
          <>
            Das Konzept steht im Programmcode und nicht in einem Dokument daneben: es ist zugleich
            der Abschnitt der Verfahrensdokumentation und die Regel, nach der die Löschfunktion
            entscheidet. Zwei Fassungen desselben Konzepts laufen auseinander.
          </>
        }
      >
        {(overview?.concept ?? []).length === 0 ? (
          <EmptyState title="Kein Löschkonzept hinterlegt" />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-80">Datenkategorie</Th>
                <Th className="w-56">Klasse</Th>
                <Th numeric className="w-20">
                  Jahre
                </Th>
                <Th>Rechtsgrundlage</Th>
              </Tr>
            </Thead>
            <Tbody>
              {(overview?.concept ?? []).map((row) => (
                <Tr key={row.category}>
                  <Td className="whitespace-normal">{row.category}</Td>
                  <Td className="text-ink-muted">{RETENTION_CLASS_LABELS[row.class]}</Td>
                  <Td numeric className="num">
                    {row.years}
                  </Td>
                  <Td className="whitespace-normal text-ink-subtle">{row.legalBasis}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {/* Frist aussetzen */}
      <Dialog
        open={holdYear !== null}
        onOpenChange={(open) => !open && setHoldYear(null)}
        title="Aufbewahrungsfrist aussetzen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setHoldYear(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" loading={busy} onClick={() => void saveHold()}>
              Frist aussetzen
            </Button>
          </>
        }
      >
        {holdError && <Notice tone="negative" text={holdError} className="mb-4" />}

        <FieldRow>
          <Field label="Geschäftsjahr">
            <Select<number>
              items={years.map((row) => ({ value: row.fiscalYear, label: String(row.fiscalYear) }))}
              value={holdYear ?? suggestedYear}
              onValueChange={(value) => setHoldYear(value)}
              aria-label="Geschäftsjahr"
            />
          </Field>
          <Field label="Grund" explain="Die Frist läuft nicht ab, solange der Grund besteht.">
            <Select<RetentionHoldReason>
              items={HOLD_REASONS}
              value={holdReason}
              onValueChange={(value) => setHoldReason(value)}
              aria-label="Grund"
            />
          </Field>
        </FieldRow>
        <Field
          label="Erläuterung"
          className="mt-4"
          optional={holdReason !== 'other'}
          hint={holdReason === 'other' ? 'Bei sonstigem Grund Pflicht' : undefined}
        >
          <Textarea
            rows={3}
            value={holdText}
            onChange={(e) => setHoldText(e.target.value)}
          />
        </Field>
      </Dialog>

      {/* Aussetzung aufheben */}
      <Dialog
        open={releasing !== null}
        onOpenChange={(open) => !open && setReleasing(null)}
        title="Aussetzung aufheben"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setReleasing(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" loading={busy} onClick={() => void release()}>
              Aufheben
            </Button>
          </>
        }
      >
        {releaseError && <Notice tone="negative" text={releaseError} className="mb-4" />}

        <Field label="Grund der Aufhebung" hint="Bleibt im Protokoll stehen">
          <Textarea
            rows={3}
            value={releaseText}
            onChange={(e) => setReleaseText(e.target.value)}
          />
        </Field>
      </Dialog>

      {/* Archivieren und löschen: erst die ausgeschriebene Jahreszahl, dann
          die Rückfrage. Der Vorgang lässt sich nicht rückgängig machen.

          Während die Rückfrage steht, weicht dieser Dialog: zwei Overlays
          übereinander verdecken einander, und die Rückfrage ist ein
          AlertDialog, der sich bewusst nicht durch einen Klick daneben
          schließen lässt (§8.2). Wird sie abgebrochen, steht dieser Dialog
          samt Eingabe wieder da, weil `deleting` gesetzt bleibt. */}
      <Dialog
        open={deleting !== null && !confirmDelete}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={`Geschäftsjahr ${deleting?.fiscalYear ?? ''} archivieren und löschen`}
        width="max-w-xl"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleting(null)}>
              Abbrechen
            </Button>
            <Button
              variant="danger"
              loading={busy}
              disabled={confirmation.trim() !== String(deleting?.fiscalYear ?? '')}
              title={
                confirmation.trim() === String(deleting?.fiscalYear ?? '')
                  ? undefined
                  : 'Die Jahreszahl als Bestätigung eingeben.'
              }
              onClick={() => setConfirmDelete(true)}
            >
              Archivieren und löschen
            </Button>
          </>
        }
      >
        {/* Die Ablehnung des Backends steht über der Eingabe, die sie
            betrifft, und über der Ankündigung des Vorgangs. */}
        {deleteError && <Notice tone="negative" text={deleteError} className="mb-4" />}

        <Notice
          tone="negative"
          text={`${deleting?.counts.journalEntries ?? 0} Buchungen, ${deleting?.counts.receipts ?? 0} Belege und ${deleting?.counts.invoices ?? 0} Rechnungen dieses Jahres werden nach dem Archivexport gelöscht.`}
        />
        <Field
          label="Bestätigung"
          className="mt-4"
          hint="Die Jahreszahl eingeben"
          explain="Ein Klick allein ist bei einem unumkehrbaren Vorgang keine Bestätigung."
        >
          <Input
            value={confirmation}
            onChange={(e) => setConfirmation(e.target.value)}
            placeholder={String(deleting?.fiscalYear ?? '')}
          />
        </Field>
      </Dialog>

      {/* Die zweite Rückfrage vor einem unumkehrbaren Schritt ist ein
          ConfirmDialog und kein Dialog: er lässt sich nur über eine der beiden
          Antworten schließen, nicht durch einen Klick daneben (§8.2). Der Text
          nennt die Folge, nicht die Aktion. */}
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Löschung wirklich ausführen?"
        description={`Buchfink erstellt zuerst den Archivexport des Geschäftsjahres ${deleting?.fiscalYear ?? ''} und löscht danach seine Daten. Der Vorgang lässt sich nicht rückgängig machen; das Löschprotokoll bleibt.`}
        confirmLabel="Endgültig löschen"
        destructive
        onConfirm={() => void archiveAndDelete()}
      />
    </>
  );
};

export default BackupPage;
