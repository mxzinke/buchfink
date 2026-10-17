import React, { useEffect, useState } from 'react';
import { Archive, FileDown, FolderOpen } from 'lucide-react';
import { Api } from '../../../services/api';
import type { ExportKind, ExportResult } from '../../../types';
import { formatDateTime } from '../../../utils/formatters';
import {
  Button,
  Field,
  FieldRow,
  Help,
  Input,
  Notice,
  Select,
  Stat,
  StatRow,
  toast,
} from '../../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

const RUNNING_HINT = 'Ein anderer Export läuft gerade.';

const EXPORT_LABELS: Record<string, string> = {
  z3: 'Tabellen',
  archive: 'Tabellen mit Belegdateien',
  audit_package: 'Prüferpaket',
  journal: 'Journal eines Zeitraums',
};

type Kind = 'z3' | 'archive' | 'audit_package';

/**
 * Die Datenüberlassung für die Betriebsprüfung: die Buchführung eines Jahres
 * so abgelegt, dass die Software des Finanzamts sie lesen kann. Jeder Export
 * steht im Änderungsprotokoll.
 */
export const DataExportPanel: React.FC<{ year: number; availableYears: number[] }> = ({
  year,
  availableYears,
}) => {
  const [exportYear, setExportYear] = useState(year);
  const [dir, setDir] = useState('');
  const [running, setRunning] = useState<ExportKind | ''>('');
  const [result, setResult] = useState<ExportResult | null>(null);
  const [error, setError] = useState('');

  useEffect(() => setExportYear(year), [year]);

  async function pickDir(): Promise<string> {
    const selected = await Api.selectExportDirectory('Zielordner für die Datenüberlassung wählen');
    if (selected) setDir(selected);
    return selected;
  }

  async function run(kind: Kind) {
    if (running) return;
    try {
      const target = dir || (await pickDir());
      if (!target) return;
      setRunning(kind);
      setResult(null);
      setError('');
      const next =
        kind === 'z3'
          ? await Api.exportZ3(exportYear, target)
          : kind === 'archive'
            ? await Api.exportArchive(exportYear, target)
            : await Api.exportAuditPackage(exportYear, target);
      setResult(next);
      toast.success(`${EXPORT_LABELS[kind]} in ${next.dir} geschrieben.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setRunning('');
    }
  }

  const years = availableYears.length > 0 ? availableYears : [year];

  return (
    <div className="max-w-3xl">
      {error && <Notice tone="negative" className="mb-5" text={error} />}
      <FieldRow>
        <Field label="Geschäftsjahr" className="w-40">
          <Select
            items={years.map((y) => ({ value: y, label: String(y) }))}
            value={exportYear}
            onValueChange={setExportYear}
          />
        </Field>
        <Field label="Zielordner" hint="Am besten ein leerer Ordner" className="flex-1 min-w-64">
          <div className="flex gap-2">
            <Input className="code-num" value={dir} readOnly placeholder="Noch nicht gewählt" />
            <Button
              variant="secondary"
              className="shrink-0"
              icon={<FolderOpen className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => void pickDir()}
            >
              Ordner wählen
            </Button>
          </div>
        </Field>
      </FieldRow>

      <div className="mt-5 flex flex-wrap items-center gap-2">
        <Button
          variant="secondary"
          loading={running === 'audit_package'}
          disabled={running !== '' && running !== 'audit_package'}
          title={running ? RUNNING_HINT : undefined}
          icon={<Archive className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void run('audit_package')}
        >
          Prüferpaket erzeugen
        </Button>
        <Button
          variant="quiet"
          loading={running === 'z3'}
          disabled={running !== '' && running !== 'z3'}
          title={running ? RUNNING_HINT : undefined}
          icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void run('z3')}
        >
          Nur Tabellen
        </Button>
        <Button
          variant="quiet"
          loading={running === 'archive'}
          disabled={running !== '' && running !== 'archive'}
          title={running ? RUNNING_HINT : undefined}
          icon={<Archive className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void run('archive')}
        >
          Mit Belegdateien
        </Button>
        <Help
          summary="Im Zweifel nehmen Sie das Prüferpaket; es enthält die beiden anderen Umfänge."
          label="Erklärung zu den Umfängen"
        >
          Nur Tabellen: Buchungen, Konten, Kontakte, offene Posten, Anlagen, Belegdaten und
          Änderungsprotokoll, mit einer Beschreibung jeder Spalte für die Prüfsoftware (Datenzugriff
          nach § 147 Abs. 6 AO). Mit Belegdateien: zusätzlich jede abgelegte Rechnung als Datei.
          Prüferpaket: dazu der Nachweis, dass nichts nachträglich geändert wurde, das Verzeichnis der
          Wahlrechte und die Verfahrensdokumentation. Welchen Umfang der Prüfer möchte, steht in
          seinem Anschreiben.
        </Help>
      </div>

      {result && (
        <div className="mt-6">
          <StatRow>
            <Stat
              label="Umfang"
              value={EXPORT_LABELS[result.kind] ?? result.kind}
              context={`Geschäftsjahr ${result.fiscalYear}`}
            />
            <Stat
              label="Dateien"
              value={String(result.files.length)}
              context={`${result.receiptFiles} Belegdateien`}
            />
            <Stat
              label="Erzeugt"
              value={formatDateTime(result.createdAt)}
              context={`Buchfink ${result.programVersion}`}
            />
          </StatRow>
          {result.notes.map((note) => (
            <Notice key={note} className="mt-5" text={note} />
          ))}
        </div>
      )}
    </div>
  );
};
