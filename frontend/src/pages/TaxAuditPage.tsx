import React, { useCallback, useEffect, useState } from 'react';
import {
  Archive,
  BookMarked,
  FileDown,
  FileText,
  FolderOpen,
  Lock,
  RefreshCw,
  Unlock,
} from 'lucide-react';
import {
  AppConfig,
  ComplianceHints,
  ExportKind,
  ExportResult,
  KeyDirectoryEntry,
  OrganisationTexts,
  ProcDocResult,
  ProcedureDocumentation,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import {
  formatBytes,
  formatDate,
  formatDateOfTimestamp,
  formatDateTime,
  formatShortHash,
} from '../utils/formatters';
import {
  Button,
  Dialog,
  EmptyState,
  Field,
  FieldRow,
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
  toast,
} from '../components/ui';

/**
 * Betriebsprüfung (Welle 9).
 *
 * Alles, was gebraucht wird, wenn der Prüferbrief auf dem Tisch liegt, und
 * nichts sonst: die Anwendung schreibgeschützt schalten, die Daten des
 * geforderten Jahres herausgeben und die Verfahrensdokumentation mitgeben.
 *
 * Was der Prüfer daneben *ansehen* will — Journal, Konten, Bilanz, das
 * Änderungsprotokoll —, steht dort, wo es hingehört, und wird hier nicht ein
 * zweites Mal aufgebaut: eine zweite, kürzere Fassung derselben Auskunft ist
 * die, der man am ehesten glaubt, vollständig zu sein.
 *
 * Erklärt wird hinter dem Erklärzeichen, nicht auf der Seite (§15).
 */

interface TaxAuditPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile — der Vorschlag für den Export. */
  year: number;
  availableYears: number[];
  appConfig: AppConfig | null;
  /** Die Konfiguration hat sich geändert: Banner und Einstellungen ziehen nach. */
  onAppConfigChange: (config: AppConfig) => void;
}

/**
 * Warum ein Knopf gerade nicht geht. Ein deaktivierter Knopf ohne Erklärung
 * versteckt seinen Grund (§10.4).
 */
const RUNNING_EXPORT_HINT = 'Ein anderer Export läuft gerade.';

const EXPORT_LABELS: Record<string, string> = {
  z3: 'Tabellen (Z3)',
  archive: 'Archiv mit Belegdateien',
  audit_package: 'Prüferpaket',
  journal: 'Journal eines Zeitraums',
  key_directory: 'Schlüsselverzeichnis',
};

/**
 * Die zwei Hälften dieser Seite.
 *
 * Die eine wird bedient, wenn der Prüferbrief kommt: schreibgeschützt schalten
 * und die Daten des geforderten Jahres herausgeben. Die andere ist ein
 * Formular, das einmal ausgefüllt und danach selten angefasst wird — und das
 * unter der Datenüberlassung als Anhängsel stand, obwohl es die einzige Stelle
 * der Anwendung ist, an der das Unternehmen etwas über sich selbst aufschreibt.
 */
type TaxAuditTab = 'ueberlassung' | 'verfahren';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Die dritte Erklärstufe (§15.2). Wer alle paar Jahre einen Prüferbrief auf dem
 * Tisch hat, muss wissen, welchen der drei Umfänge er nehmen soll; das passt
 * nicht in drei Sätze und gehört deshalb hinter „Mehr dazu" statt in die
 * Ansicht.
 */
const ExportHelpDialog: React.FC<{ open: boolean; onClose: () => void }> = ({ open, onClose }) => (
  <Dialog
    open={open}
    onOpenChange={(next) => !next && onClose()}
    title="Daten für die Betriebsprüfung"
    footer={
      <Button variant="secondary" onClick={onClose}>
        Schließen
      </Button>
    }
  >
    <div className="text-body text-ink-muted space-y-4">
      <p>
        Bei einer Außenprüfung darf das Finanzamt die Buchführung eines Jahres auf einem Datenträger
        verlangen. Der Prüfer liest sie mit seiner eigenen Software. Sie wählen Jahr und Zielordner,
        Buchfink legt alles Nötige dort ab, und Sie übergeben den Ordner.
      </p>
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th className="w-48">Umfang</Th>
            <Th>Was dazukommt</Th>
          </Tr>
        </Thead>
        <Tbody>
          <Tr>
            <Td>Nur Tabellen</Td>
            <Td className="whitespace-normal">
              Buchungen, Konten, Salden, Kontakte, offene Posten, Anlagen, Belegdaten und
              Änderungsprotokoll als Tabellen, dazu eine Erklärung jeder Spalte für die
              Prüfsoftware.
            </Td>
          </Tr>
          <Tr>
            <Td>Mit Belegdateien</Td>
            <Td className="whitespace-normal">
              Jede abgelegte Rechnung und jedes Dokument als eigene Datei.
            </Td>
          </Tr>
          <Tr>
            <Td>Prüferpaket</Td>
            <Td className="whitespace-normal">
              Der Nachweis, dass nichts nachträglich geändert wurde, das Verzeichnis der
              steuerlichen Wahlrechte und die Verfahrensdokumentation.
            </Td>
          </Tr>
        </Tbody>
      </Table>
      <p>
        Welchen Umfang der Prüfer möchte, steht in seinem Anschreiben. Im Zweifel nehmen Sie das
        Prüferpaket, es enthält die beiden anderen.
      </p>
      <p>
        Jeder Export wird im Änderungsprotokoll vermerkt. Damit lässt sich später belegen, was der
        Prüfer wann bekommen hat.
      </p>
    </div>
  </Dialog>
);

/**
 * Das Schlüsselverzeichnis, aufgeschlagen statt ausgebreitet.
 *
 * Es ist eine Tabelle, die kein Anwender je liest: Sie erklärt dem Prüfer die
 * Kürzel in den überlassenen Tabellen und liegt jedem Export ohnehin bei.
 * Gebraucht wird sie einzeln nur, wenn ein Prüfer sie nachfordert.
 */
const KeyDirectoryDialog: React.FC<{
  open: boolean;
  onClose: () => void;
  entries: KeyDirectoryEntry[];
  onSave: () => void;
}> = ({ open, onClose, entries, onSave }) => (
  <Dialog
    open={open}
    onOpenChange={(next) => !next && onClose()}
    title="Schlüsselverzeichnis"
    width="max-w-4xl"
    footer={
      <>
        <Button variant="secondary" onClick={onClose}>
          Schließen
        </Button>
        <Button
          variant="primary"
          onClick={onSave}
          icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
        >
          Als CSV speichern
        </Button>
      </>
    }
  >
    <p className="text-body text-ink-muted mb-5">
      In den überlassenen Tabellen stehen Kürzel wie „13b_2_1“. Diese Liste sagt dem Prüfer, was
      jedes davon bedeutet. Sie liegt jedem Export bei; einzeln brauchen Sie sie nur, wenn der
      Prüfer sie nachfordert.
    </p>
    {entries.length === 0 ? (
      <SkeletonRows rows={6} />
    ) : (
      <Table density="kompakt">
        <Thead sticky>
          <Tr>
            <Th className="w-48">Gruppe</Th>
            <Th className="w-40">Schlüssel</Th>
            <Th className="w-56">Klartext</Th>
            <Th>Bedeutung</Th>
          </Tr>
        </Thead>
        <Tbody>
          {entries.map((entry) => (
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
  </Dialog>
);

export const TaxAuditPage: React.FC<TaxAuditPageProps> = ({
  year,
  availableYears,
  appConfig,
  onAppConfigChange,
}) => {
  const [exportYear, setExportYear] = useState<number>(year);
  const [exportDir, setExportDir] = useState('');
  const [runningExport, setRunningExport] = useState<ExportKind | ''>('');
  const [exportResult, setExportResult] = useState<ExportResult | null>(null);
  const [keyDirectory, setKeyDirectory] = useState<KeyDirectoryEntry[]>([]);

  // Ein Fehler aus dem Backend gehört als Hinweisfläche über die Aktionen des
  // Abschnitts und nicht in einen Toast (§10.4): der Export läuft lange, und
  // wer daneben weiterliest, hätte den Toast nach vier Sekunden verpasst —
  // samt Pfad und Grund, an denen sich der Fehler beheben lässt. Je Abschnitt
  // einer, damit der Streifen bei der Aktion steht, die ihn ausgelöst hat.
  const [exportError, setExportError] = useState('');

  const [readOnlyUntil, setReadOnlyUntil] = useState('');
  const [readOnlyReason, setReadOnlyReason] = useState('');
  const [readOnlyError, setReadOnlyError] = useState('');
  const [endReason, setEndReason] = useState('');
  const [switchingMode, setSwitchingMode] = useState(false);

  const [helpOpen, setHelpOpen] = useState(false);
  const [keyDirOpen, setKeyDirOpen] = useState(false);

  // Der Reiter ist Zustand dieser Seite und kein Navigationsziel: jeder Weg
  // hierher meint die Herausgabe.
  const [tab, setTab] = useState<TaxAuditTab>('ueberlassung');

  // Das Schlüsselverzeichnis wird erst geladen, wenn jemand es aufschlägt. Es
  // wird nur bei einer Datenüberlassung gebraucht, und jedem Export liegt es
  // ohnehin bei.
  useEffect(() => {
    if (!keyDirOpen || keyDirectory.length > 0) return;
    void Api.getKeyDirectory()
      .then(setKeyDirectory)
      .catch((e: unknown) => setExportError(message(e)));
  }, [keyDirOpen]);

  // Das Jahr der Kopfzeile ist der Vorschlag; eine eigene Wahl bleibt stehen,
  // solange die Seite offen ist.
  useEffect(() => {
    setExportYear(year);
  }, [year]);

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
      setExportError('');
      const result =
        kind === 'z3'
          ? await Api.exportZ3(exportYear, target)
          : kind === 'archive'
            ? await Api.exportArchive(exportYear, target)
            : await Api.exportAuditPackage(exportYear, target);
      setExportResult(result);
      toast.success(`${EXPORT_LABELS[kind]} in ${result.dir} geschrieben.`);
    } catch (e) {
      setExportError(message(e));
    } finally {
      setRunningExport('');
    }
  }

  async function saveKeyDirectory() {
    setExportError('');
    try {
      const path = await Api.exportKeyDirectory();
      if (path) toast.success(`Schlüsselverzeichnis gespeichert: ${path}`);
    } catch (e) {
      setExportError(message(e));
    }
  }

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
      setReadOnlyError(message(e));
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
      setReadOnlyError(message(e));
    } finally {
      setSwitchingMode(false);
    }
  }

  const readOnly = appConfig?.readOnly ?? false;
  const years = availableYears.length > 0 ? availableYears : [year];

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      {/* Jeder Reiter hat seine eigene erste Handlung (§10.4): das Prüferpaket
          herausgeben oder die Verfahrensdokumentation neu erzeugen. Die zweite
          steht im Reiter selbst, weil sie von den Texten darunter abhängt. */}
      <PageHeader
        title="Betriebsprüfung"
        context="Die Buchführung schreibgeschützt schalten und dem Prüfer übergeben"
        action={
          tab === 'ueberlassung' ? (
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
          ) : undefined
        }
      />

      <Tabs<TaxAuditTab>
        value={tab}
        onValueChange={setTab}
        className="mt-6"
        items={[
          { value: 'ueberlassung', label: 'Datenüberlassung' },
          { value: 'verfahren', label: 'Verfahrensdokumentation' },
        ]}
      >
        <TabPanel value="ueberlassung">
          {/* Der Streifen „Prüfermodus bis …" steht in App.tsx über jeder
              Ansicht: gesperrt ist die Anwendung und nicht diese Seite. Hier
              steht nur, was zum Schalten nötig ist. */}
          <Section
            title="Prüfermodus"
            divider={false}
            context={
              readOnly
                ? `Aktiv bis ${formatDate(appConfig?.readOnlyUntil ?? '')} · Grund: ${appConfig?.readOnlyReason || '—'}`
                : 'Aus. Sie können buchen und ändern.'
            }
            explain={
              <>
                Solange der Prüfermodus läuft, kann niemand mehr buchen oder etwas ändern. Auswerten,
                drucken und exportieren geht weiter. Er endet an dem Tag, den Sie angeben, von selbst.
              </>
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
            context="Die Daten eines Jahres für den Betriebsprüfer"
            explain={
              <>
                Wählen Sie Jahr und Zielordner, und Buchfink legt die Buchführung dort so ab, dass die
                Software des Finanzamts sie lesen kann. Wie viel dabei herauskommt, hängt vom Umfang ab.
              </>
            }
            onMore={() => setHelpOpen(true)}
          >
            {exportError && <Notice tone="negative" className="mb-5" text={exportError} />}

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
              {/* Das Schlüsselverzeichnis liegt jedem Export bei. Einzeln steht es
                  deshalb hier und nicht als eigener Abschnitt: gebraucht wird es
                  nur, wenn ein Prüfer es nachfordert. */}
              <Button
                variant="quiet"
                onClick={() => setKeyDirOpen(true)}
                icon={<BookMarked className="w-4 h-4" strokeWidth={1.5} />}
              >
                Schlüsselverzeichnis
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
        </TabPanel>

        {/* Die Verfahrensdokumentation lädt für sich: sie holt Fassungen,
            Freitexte, Muster und Hinweise, und wer ein Prüferpaket erzeugen
            will, braucht davon nichts. */}
        <TabPanel value="verfahren">{tab === 'verfahren' && <ProcDocPanel />}</TabPanel>
      </Tabs>

      <ExportHelpDialog open={helpOpen} onClose={() => setHelpOpen(false)} />
      <KeyDirectoryDialog
        open={keyDirOpen}
        onClose={() => setKeyDirOpen(false)}
        entries={keyDirectory}
        onSave={() => void saveKeyDirectory()}
      />
    </div>
  );
};
// -------------------------------------------------------------------------
// Verfahrensdokumentation
// -------------------------------------------------------------------------

interface OrgField {
  key: keyof OrganisationTexts;
  label: string;
  /** Höchstens sechs Wörter (§15.1); die Erklärung steht dahinter. */
  hint: string;
  /** Wonach der Abschnitt gefragt ist, in ein bis drei Sätzen (§15.2). */
  explain: string;
}

/**
 * Die Freitexte der Organisationsanweisung, nach Fragen gruppiert.
 *
 * Sieben Textfelder nebeneinander sind kein Formular, sondern eine Liste, die
 * man von oben nach unten abarbeitet, ohne zu wissen, wonach gefragt ist. Die
 * Gruppen sind die drei Fragen, die ein Prüfer an die Organisation stellt — wer
 * arbeitet damit, wie kommen die Belege herein, wie sind die Daten gesichert —,
 * und was in keine davon passt, steht darunter.
 *
 * Die Reihenfolge innerhalb einer Gruppe folgt dem Weg des Belegs und nicht der
 * Reihenfolge im Datensatz.
 */
const ORG_GROUPS: { title: string; context: string; fields: OrgField[] }[] = [
  {
    title: 'Wer die Buchführung führt',
    context: 'Verantwortung, Freigabe und Vertretung',
    fields: [
      {
        key: 'responsibilities',
        label: 'Verantwortung',
        hint: 'Wer führt die Buchführung',
        explain:
          'Wer die Bücher führt und wer dafür einsteht. Dazu gehört, woran sich eine einzelne Buchung später einer Person zuordnen lässt.',
      },
      {
        key: 'approval',
        label: 'Freigabe',
        hint: 'Wer bucht, wer schreibt fest',
        explain:
          'Wer bucht und wer freigibt. Liegt beides in einer Hand, gehört hierher, was das ausgleicht — etwa der Prüflauf vor der Festschreibung und der Abgleich mit dem Kontoauszug.',
      },
      {
        key: 'substitution',
        label: 'Vertretung',
        hint: 'Im Verhinderungsfall',
        explain:
          'Wer übernimmt bei Krankheit oder Urlaub, und woher diese Person den Zugang zu den Daten und zum Wiederherstellungsschlüssel bekommt.',
      },
    ],
  },
  {
    title: 'Wie Belege ins Haus kommen',
    context: 'Eingang, Erfassung und Einscannen',
    fields: [
      {
        key: 'receiptFlow',
        label: 'Belegfluss',
        hint: 'Eingang, Erfassung, Prüfung',
        explain:
          'Der Weg vom Eingang bis zur Buchung: wo Belege ankommen, wer sie erfasst, in welcher Frist gebucht wird und was mit dem Papier geschieht.',
      },
      {
        key: 'scanning',
        label: 'Einscannen',
        hint: 'Gerät, Auflösung, Ablage',
        explain:
          'Gerät, Auflösung und Farbe, die Prüfung der Lesbarkeit — und ob das Papieroriginal danach vernichtet wird. Ersetzendes Scannen verlangt eine eigene Beschreibung.',
      },
    ],
  },
  {
    title: 'Wie die Daten gesichert werden',
    context: 'Rhythmus, Ziel und Überwachung',
    fields: [
      {
        key: 'backup',
        label: 'Sicherung',
        hint: 'Wer überwacht, wo liegen Kopien',
        explain:
          'Wohin gesichert wird, wie oft, wer merkt, wenn eine Sicherung ausbleibt, und wann zuletzt eine probeweise zurückgespielt wurde.',
      },
    ],
  },
  {
    title: 'Weiteres',
    context: 'Was in keinen der Abschnitte darüber passt',
    fields: [
      {
        key: 'notes',
        label: 'Weiteres',
        hint: 'Alles Übrige',
        explain:
          'Besonderheiten dieses Unternehmens, die zur Buchführung gehören und oben keinen Platz haben. Bleibt das Feld leer, fehlt in der Fassung nichts.',
      },
    ],
  },
];

const ORG_FIELD_COUNT = ORG_GROUPS.reduce((count, group) => count + group.fields.length, 0);

/**
 * Die Verfahrensdokumentation als das, was sie für den Anwender ist: ein
 * Formular, das einmal ausgefüllt wird, und ein Dokument, das daraus entsteht.
 *
 * Die Liste der früheren Fassungen mit Prüfsumme, Größe und Ablagepfad steht
 * hinter „Frühere Fassungen": zu jedem Geschäftsjahr gilt die Fassung, die
 * damals galt (GoBD Rz. 151), gebraucht wird die ältere aber nur, wenn ein
 * Prüfer nach dem Jahr fragt — als Tabelle auf der Seite war sie das Größte an
 * einem Abschnitt, an dem sonst zwei Knöpfe stehen.
 */
const ProcDocPanel: React.FC = () => {
  const lock = useWriteLock();
  const [documents, setDocuments] = useState<ProcedureDocumentation[]>([]);
  const [texts, setTexts] = useState<OrganisationTexts | null>(null);
  // Die Muster daneben: nur im Vergleich mit ihnen lässt sich sagen, welcher
  // Abschnitt beschrieben und welcher bloß vorbelegt ist. Die Texte allein
  // sehen in beiden Fällen gleich aus.
  const [defaults, setDefaults] = useState<OrganisationTexts | null>(null);
  // Der gespeicherte Stand. An ihm hängt, ob eine neue Fassung gerade Sinn
  // ergibt: erzeugt wird aus dem, was gespeichert ist, und nicht aus dem, was
  // im Feld steht.
  const [saved, setSaved] = useState<OrganisationTexts | null>(null);
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  const [result, setResult] = useState<ProcDocResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [saving, setSaving] = useState(false);
  const [downloading, setDownloading] = useState<number | null>(null);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [docs, orgTexts, orgDefaults, complianceHints] = await Promise.all([
        Api.getProcedureDocumentations(),
        Api.getOrganisationTexts(),
        Api.getOrganisationTextDefaults(),
        Api.getComplianceHints(),
      ]);
      setDocuments(docs);
      setTexts(orgTexts);
      setSaved(orgTexts);
      setDefaults(orgDefaults);
      setHints(complianceHints);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const latest = documents[0];
  const taxCaseHints = hints?.taxCaseHints ?? [];

  /** Ein Abschnitt steht noch im Muster, wenn er Wort für Wort das Muster ist. */
  function isTemplate(key: keyof OrganisationTexts): boolean {
    if (!texts || !defaults) return false;
    const value = texts[key].trim();
    // Ein leeres „Weiteres" ist kein unbeantworteter Abschnitt: das Muster
    // dafür ist selbst leer.
    if (value === '') return defaults[key].trim() !== '';
    return value === defaults[key].trim();
  }

  const templateCount = ORG_GROUPS.flatMap((group) => group.fields).filter((field) =>
    isTemplate(field.key),
  ).length;
  const described = ORG_FIELD_COUNT - templateCount;
  // Ungespeicherte Eingaben gingen in eine neue Fassung nicht ein: erzeugt wird
  // aus dem gespeicherten Stand. Statt das stillschweigend hinzunehmen, sperrt
  // es den Knopf und nennt den Grund (§10.4).
  const dirty = texts !== null && saved !== null && JSON.stringify(texts) !== JSON.stringify(saved);

  async function generate() {
    setBusy(true);
    setError('');
    try {
      const generated = await Api.generateProcedureDocumentation();
      setResult(generated);
      await load();
      toast.success(generated.message || `Fassung ${generated.document.version} erzeugt.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  /**
   * Eine Fassung herausgeben. Das PDF, wo es entstanden ist — der Prüfer
   * bekommt ein Dokument und keine Textdatei —, sonst das Markdown.
   */
  async function download(doc: ProcedureDocumentation, wantPdf: boolean) {
    setDownloading(doc.id);
    setError('');
    try {
      const path = await Api.saveProcedureDocumentationAs(doc.id, wantPdf);
      // Ein abgebrochener Speichern-Dialog ist keine Meldung wert.
      if (path) toast.success(`Fassung ${doc.version} gespeichert: ${path}`);
    } catch (e) {
      setError(message(e));
    } finally {
      setDownloading(null);
    }
  }

  async function saveTexts() {
    if (!texts) return;
    setSaving(true);
    setError('');
    try {
      await Api.saveOrganisationTexts(texts);
      setSaved(texts);
      toast.success('Organisationsanweisung gespeichert.');
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}
      {hints?.cloudWarning && <Notice text={hints.cloudWarning} className="mb-6" />}
      {hints?.legalFormNote && <Notice text={hints.legalFormNote} className="mb-6" />}

      <StatRow>
        <Stat
          label="Letzte Fassung"
          value={latest?.version ?? '—'}
          context={latest ? formatDateOfTimestamp(latest.createdAt) : 'noch keine erzeugt'}
        />
        <Stat
          label="Ihre Angaben"
          value={`${described} von ${ORG_FIELD_COUNT}`}
          // Kein Rosé bei unveränderten Mustern: sie sind für ein
          // Ein-Personen-Unternehmen oft die Wahrheit, und ein roter Wert
          // behauptete einen Fehler, wo nur eine Frage offen ist.
          context={templateCount === 0 ? 'alle beschrieben' : `${templateCount} noch im Muster`}
        />
        <Stat
          label="Programmfassung"
          value={latest?.appVersion ?? '—'}
          context={latest ? `Regelwerk ${latest.ruleVersion}` : 'aus der letzten Fassung'}
        />
        <Stat label="Speicherort" value="Lokaler Datenordner" context={hints?.dataDir ?? '—'} />
      </StatRow>

      <Section
        title="Aktuelle Fassung"
        context={
          latest
            ? `Fassung ${latest.version} vom ${formatDateOfTimestamp(latest.createdAt)}`
            : 'Noch keine Fassung erzeugt'
        }
        explain={
          <>
            Die Verfahrensdokumentation beschreibt dem Prüfer, wie Ihre Buchführung zustande kommt.
            Buchfink schreibt sie aus Ihren Angaben, den Stammdaten und dem Stand, der zu dem
            Zeitpunkt galt. Jede Fassung bleibt über die Aufbewahrungsfrist nachlesbar und liegt
            jedem Prüferpaket bei; einzeln brauchen Sie sie, wenn der Prüfer nur nach ihr fragt.
          </>
        }
        action={
          documents.length > 0 && (
            <div className="flex items-center gap-2">
              <Button variant="quiet" onClick={() => setVersionsOpen(true)}>
                Frühere Fassungen
              </Button>
              <Button
                variant="secondary"
                loading={downloading === latest.id}
                onClick={() => void download(latest, Boolean(latest.pdfStoredPath))}
                icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
              >
                Herunterladen
              </Button>
              {/* Die eine Primäraktion dieses Reiters (§10.4): aus den Angaben
                  darunter wird das Dokument. */}
              <Button
                variant="primary"
                loading={busy}
                disabled={lock.locked || dirty}
                title={
                  lock.hint ??
                  (dirty
                    ? 'Die geänderten Angaben sind noch nicht gespeichert; die Fassung entstünde ohne sie.'
                    : undefined)
                }
                onClick={() => void generate()}
                icon={<RefreshCw className="w-4 h-4" strokeWidth={1.5} />}
              >
                Neue Fassung erzeugen
              </Button>
            </div>
          )
        }
      >
        {result?.pdfNote && <Notice text={result.pdfNote} className="mb-5" />}

        {documents.length === 0 ? (
          <EmptyState
            icon={<FileText className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch keine Fassung erzeugt"
            description="Sie entsteht aus Ihren Angaben darunter, den Stammdaten und den heutigen Einstellungen."
            action={
              <Button
                variant="primary"
                loading={busy}
                disabled={lock.locked || dirty}
                title={
                  lock.hint ??
                  (dirty
                    ? 'Die geänderten Angaben sind noch nicht gespeichert; die Fassung entstünde ohne sie.'
                    : undefined)
                }
                onClick={() => void generate()}
              >
                Fassung erzeugen
              </Button>
            }
          />
        ) : (
          <p className="text-body text-ink-muted">
            {documents.length === 1
              ? 'Eine Fassung liegt im Datenordner und wandert mit jeder Sicherung mit.'
              : `${documents.length} Fassungen liegen im Datenordner und wandern mit jeder Sicherung mit.`}{' '}
            {latest.pdfStoredPath
              ? 'Herausgegeben wird das PDF; das Markdown daneben steht unter „Frühere Fassungen“.'
              : 'Zu dieser Fassung ist kein PDF entstanden; herausgegeben wird das Markdown.'}
          </p>
        )}
      </Section>

      <Section
        title="Ihre Angaben zur Organisation"
        context={
          templateCount === 0
            ? `${ORG_FIELD_COUNT} Abschnitte, alle beschrieben`
            : `${described} von ${ORG_FIELD_COUNT} Abschnitten beschrieben, ${templateCount} im Muster`
        }
        explain={
          <>
            Wer scannt, wer prüft, wer freigibt und wie vertreten wird: das weiß nur das
            Unternehmen, und deshalb schreibt Buchfink es nicht. Die Muster beschreiben den
            Einzelplatzbetrieb — für ein Ein-Personen-Unternehmen ist das oft die Wahrheit, aber
            gelesen haben sollten Sie sie. Ihre Texte stehen als eigener Abschnitt in jeder neuen
            Fassung.
          </>
        }
        action={
          <Button
            variant="secondary"
            loading={saving}
            disabled={lock.locked || !texts || !dirty}
            title={lock.hint ?? (dirty ? undefined : 'Es steht nichts Ungespeichertes im Formular.')}
            onClick={() => void saveTexts()}
          >
            Texte speichern
          </Button>
        }
      >
        {texts && (
          <div className="space-y-8 max-w-3xl">
            {ORG_GROUPS.map((group) => (
              <div key={group.title}>
                {/* Die Gruppenüberschrift ist keine eigene Section: der
                    Speichern-Knopf gehört über alle Gruppen, und vier
                    Abschnitte mit Haarlinie hätten vier Stellen, an denen man
                    ihn sucht. */}
                <h3 className="text-label text-ink">{group.title}</h3>
                <p className="text-caption text-ink-subtle mt-0.5 mb-4">{group.context}</p>
                <div className="space-y-5">
                  {group.fields.map((field) => (
                    <Field
                      key={field.key}
                      label={field.label}
                      // Der Hinweis sagt den Stand, nicht die Frage: wonach
                      // gefragt ist, steht hinter dem Erklärzeichen, und ein
                      // Feld, das noch im Muster steht, muss das sagen können.
                      hint={isTemplate(field.key) ? 'Muster von Buchfink' : 'Von Ihnen beschrieben'}
                      explain={field.explain}
                      optional={field.key === 'notes'}
                    >
                      <Textarea
                        rows={4}
                        value={texts[field.key]}
                        disabled={lock.locked}
                        onChange={(e) => setTexts({ ...texts, [field.key]: e.target.value })}
                      />
                    </Field>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </Section>

      <Section
        title="Grenzen des Funktionsumfangs"
        context="Steuerfälle, die Buchfink nicht abbildet"
        explain={
          <>
            Sie stehen in jeder Fassung: eine Verfahrensdokumentation, die verschweigt, was das
            Programm nicht kann, beschreibt ein anderes Programm.
          </>
        }
      >
        {taxCaseHints.length === 0 ? (
          <EmptyState title="Keine Einschränkung hinterlegt" />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Hinweis</Th>
              </Tr>
            </Thead>
            <Tbody>
              {taxCaseHints.map((hint) => (
                <Tr key={hint}>
                  <Td className="whitespace-normal text-ink-muted">{hint}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Dialog
        open={versionsOpen}
        onOpenChange={(open) => !open && setVersionsOpen(false)}
        title="Fassungen der Verfahrensdokumentation"
        width="max-w-4xl"
        footer={
          <Button variant="secondary" onClick={() => setVersionsOpen(false)}>
            Schließen
          </Button>
        }
      >
        <p className="text-body text-ink-muted mb-5">
          Zu jedem Geschäftsjahr gilt die Fassung, die damals galt. Die Prüfsumme steht daneben,
          damit sich eine herausgegebene Datei später der Fassung zuordnen lässt, deren Erzeugung im
          Änderungsprotokoll steht.
        </p>
        <Table density="kompakt">
          <Thead sticky>
            <Tr>
              <Th className="w-36">Fassung</Th>
              <Th className="w-44">Erzeugt</Th>
              <Th numeric className="w-20">
                Jahr
              </Th>
              <Th className="w-24">Programm</Th>
              <Th numeric className="w-24">
                Größe
              </Th>
              <Th className="w-36">Prüfsumme</Th>
              <Th className="w-44" aria-label="Herausgeben" />
            </Tr>
          </Thead>
          <Tbody>
            {documents.map((doc) => (
              <Tr key={doc.id}>
                <Td className="num">{doc.version}</Td>
                <Td className="text-ink-subtle num">
                  {formatDateTime(doc.createdAt)}
                  <span className="block text-caption">{doc.actor}</span>
                </Td>
                <Td numeric className="num">
                  {doc.fiscalYear}
                </Td>
                <Td className="num text-ink-muted">{doc.appVersion}</Td>
                <Td numeric className="num">
                  {formatBytes(doc.size)}
                </Td>
                <Td className="code-num text-ink-subtle">{formatShortHash(doc.sha256)}</Td>
                <Td className="pl-0">
                  <span className="flex items-center gap-1">
                    {doc.pdfStoredPath && (
                      <Button
                        variant="quiet"
                        size="sm"
                        loading={downloading === doc.id}
                        onClick={() => void download(doc, true)}
                      >
                        PDF
                      </Button>
                    )}
                    <Button
                      variant="quiet"
                      size="sm"
                      loading={downloading === doc.id && !doc.pdfStoredPath}
                      onClick={() => void download(doc, false)}
                    >
                      Markdown
                    </Button>
                  </span>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Dialog>
    </>
  );
};

export default TaxAuditPage;
