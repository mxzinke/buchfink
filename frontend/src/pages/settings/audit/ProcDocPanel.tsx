import React, { useCallback, useEffect, useState } from 'react';
import { FileDown, PenLine, RefreshCw } from 'lucide-react';
import { Api } from '../../../services/api';
import type {
  ComplianceHints,
  OrganisationTexts,
  ProcDocResult,
  ProcedureDocumentation,
} from '../../../types';
import { useWriteLock } from '../../../components/WriteLock';
import {
  formatBytes,
  formatDateOfTimestamp,
  formatDateTime,
  formatShortHash,
} from '../../../utils/formatters';
import {
  Button,
  Dialog,
  Field,
  Notice,
  SkeletonRows,
  Stat,
  StatRow,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  toast,
} from '../../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

interface OrgField {
  key: keyof OrganisationTexts;
  label: string;
  explain: string;
}

/**
 * Die Freitexte der Organisationsanweisung in der Reihenfolge, in der ein
 * Prüfer fragt: wer führt die Bücher, wie kommen Belege herein, wie wird
 * gesichert.
 */
const ORG_FIELDS: OrgField[] = [
  {
    key: 'responsibilities',
    label: 'Verantwortung',
    explain: 'Wer die Bücher führt und dafür einsteht, und woran sich eine Buchung einer Person zuordnen lässt.',
  },
  {
    key: 'approval',
    label: 'Freigabe',
    explain:
      'Wer bucht und wer freigibt. Bucht eine KI-App vor, steht hier, wer ihre Buchungen prüft und festschreibt.',
  },
  {
    key: 'substitution',
    label: 'Vertretung',
    explain: 'Wer bei Krankheit oder Urlaub übernimmt und woher diese Person Zugang zu Daten und Sicherungen bekommt.',
  },
  {
    key: 'receiptFlow',
    label: 'Belegfluss',
    explain: 'Der Weg vom Eingang bis zur Buchung: wo Belege ankommen, wer sie erfasst und was mit dem Papier geschieht.',
  },
  {
    key: 'scanning',
    label: 'Einscannen',
    explain: 'Gerät, Auflösung, Prüfung der Lesbarkeit und ob das Papieroriginal danach vernichtet wird.',
  },
  {
    key: 'backup',
    label: 'Sicherung',
    explain: 'Wohin gesichert wird, wie oft, wer merkt, wenn eine Sicherung ausbleibt, und wann zuletzt zurückgespielt wurde.',
  },
  {
    key: 'notes',
    label: 'Weiteres',
    explain: 'Besonderheiten dieses Unternehmens, die oben keinen Platz haben.',
  },
];

/**
 * Die Verfahrensdokumentation: Buchfink schreibt sie aus den Stammdaten, den
 * Einstellungen und den Angaben zur Organisation. Jede Fassung bleibt über die
 * Aufbewahrungsfrist nachlesbar.
 */
export const ProcDocPanel: React.FC = () => {
  const lock = useWriteLock();
  const [documents, setDocuments] = useState<ProcedureDocumentation[]>([]);
  const [texts, setTexts] = useState<OrganisationTexts | null>(null);
  const [defaults, setDefaults] = useState<OrganisationTexts | null>(null);
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  const [result, setResult] = useState<ProcDocResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [downloading, setDownloading] = useState<number | null>(null);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [editing, setEditing] = useState(false);
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

  /** Ein Abschnitt steht noch im Muster, wenn er Wort für Wort das Muster ist. */
  function isTemplate(key: keyof OrganisationTexts): boolean {
    if (!texts || !defaults) return false;
    const value = texts[key].trim();
    if (value === '') return defaults[key].trim() !== '';
    return value === defaults[key].trim();
  }

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

  async function download(doc: ProcedureDocumentation, wantPdf: boolean) {
    setDownloading(doc.id);
    setError('');
    try {
      const path = await Api.saveProcedureDocumentationAs(doc.id, wantPdf);
      if (path) toast.success(`Fassung ${doc.version} gespeichert: ${path}`);
    } catch (e) {
      setError(message(e));
    } finally {
      setDownloading(null);
    }
  }

  if (loading) return <SkeletonRows rows={4} />;

  const latest = documents[0];
  const templateCount = ORG_FIELDS.filter((field) => isTemplate(field.key)).length;
  const taxCaseHints = hints?.taxCaseHints ?? [];

  return (
    <div>
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      {hints?.cloudWarning && <Notice text={hints.cloudWarning} className="mb-5" />}
      {result?.pdfNote && <Notice text={result.pdfNote} className="mb-5" />}

      <StatRow>
        <Stat
          label="Letzte Fassung"
          value={latest?.version ?? '—'}
          context={latest ? formatDateOfTimestamp(latest.createdAt) : 'noch keine erzeugt'}
        />
        <Stat
          label="Angaben zur Organisation"
          value={`${ORG_FIELDS.length - templateCount} von ${ORG_FIELDS.length}`}
          context={templateCount === 0 ? 'alle beschrieben' : `${templateCount} noch im Muster`}
        />
        <Stat
          label="Programmfassung"
          value={latest?.appVersion ?? '—'}
          context={latest ? `Regelwerk ${latest.ruleVersion}` : 'aus der letzten Fassung'}
        />
      </StatRow>

      <div className="mt-5 flex flex-wrap gap-2">
        <Button
          variant="secondary"
          size="sm"
          loading={busy}
          disabled={lock.locked}
          title={lock.hint}
          icon={<RefreshCw className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void generate()}
        >
          {latest ? 'Neue Fassung erzeugen' : 'Fassung erzeugen'}
        </Button>
        {latest && (
          <Button
            variant="secondary"
            size="sm"
            loading={downloading === latest.id}
            icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => void download(latest, Boolean(latest.pdfStoredPath))}
          >
            Herunterladen
          </Button>
        )}
        <Button
          variant="quiet"
          size="sm"
          icon={<PenLine className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => setEditing(true)}
        >
          Angaben zur Organisation
        </Button>
        {documents.length > 1 && (
          <Button variant="quiet" size="sm" onClick={() => setVersionsOpen(true)}>
            Frühere Fassungen
          </Button>
        )}
      </div>

      {taxCaseHints.length > 0 && (
        <div className="mt-8">
          <h4 className="text-label text-ink">Grenzen des Funktionsumfangs</h4>
          <p className="text-caption text-ink-subtle mt-0.5 mb-3">Stehen in jeder Fassung</p>
          <ul className="flex flex-col gap-1.5 max-w-3xl">
            {taxCaseHints.map((hint) => (
              <li key={hint} className="flex gap-2.5 text-body text-ink-muted">
                <span className="mark-diamond bg-line-strong mt-2 shrink-0" />
                <span>{hint}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {editing && texts && (
        <OrganisationDialog
          texts={texts}
          isTemplate={isTemplate}
          onClose={() => setEditing(false)}
          onSaved={(saved) => {
            setTexts(saved);
            setEditing(false);
          }}
        />
      )}

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
          Zu jedem Geschäftsjahr gilt die Fassung, die damals galt.
        </p>
        <Table density="kompakt">
          <Thead sticky>
            <Tr>
              <Th className="w-28">Fassung</Th>
              <Th className="w-48">Erzeugt</Th>
              <Th numeric className="w-20">Jahr</Th>
              <Th numeric className="w-24">Größe</Th>
              <Th className="w-36">Prüfsumme</Th>
              <Th className="w-40" aria-label="Herausgeben" />
            </Tr>
          </Thead>
          <Tbody>
            {documents.map((doc) => (
              <Tr key={doc.id}>
                <Td className="num">{doc.version}</Td>
                <Td className="text-ink-subtle num">{formatDateTime(doc.createdAt)}</Td>
                <Td numeric>{doc.fiscalYear}</Td>
                <Td numeric>{formatBytes(doc.size)}</Td>
                <Td className="code-num text-ink-subtle">{formatShortHash(doc.sha256)}</Td>
                <Td className="pl-0">
                  <span className="flex items-center gap-1">
                    {doc.pdfStoredPath && (
                      <Button variant="quiet" size="sm" loading={downloading === doc.id} onClick={() => void download(doc, true)}>
                        PDF
                      </Button>
                    )}
                    <Button variant="quiet" size="sm" onClick={() => void download(doc, false)}>
                      Markdown
                    </Button>
                  </span>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Dialog>
    </div>
  );
};

const OrganisationDialog: React.FC<{
  texts: OrganisationTexts;
  isTemplate: (key: keyof OrganisationTexts) => boolean;
  onClose: () => void;
  onSaved: (texts: OrganisationTexts) => void;
}> = ({ texts, isTemplate, onClose, onSaved }) => {
  const lock = useWriteLock();
  const [draft, setDraft] = useState<OrganisationTexts>(texts);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const dirty = JSON.stringify(draft) !== JSON.stringify(texts);

  async function save() {
    setSaving(true);
    setError('');
    try {
      await Api.saveOrganisationTexts(draft);
      toast.success('Angaben zur Organisation gespeichert. Sie stehen in der nächsten Fassung.');
      onSaved(draft);
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title="Angaben zur Organisation"
      width="max-w-2xl"
      dirty={dirty}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={lock.locked || !dirty}
            title={lock.hint ?? (dirty ? undefined : 'Es ist nichts geändert.')}
            onClick={() => void save()}
          >
            Speichern
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-5">
        {ORG_FIELDS.map((field) => (
          <Field
            key={field.key}
            label={field.label}
            hint={isTemplate(field.key) ? 'Muster von Buchfink' : 'Von Ihnen beschrieben'}
            helpSummary="Beschreiben Sie, wie dieser Teil der Buchhaltung bei Ihnen organisiert ist."
            explain={field.explain}
            optional={field.key === 'notes'}
          >
            <Textarea
              rows={3}
              value={draft[field.key]}
              disabled={lock.locked}
              onChange={(e) => setDraft({ ...draft, [field.key]: e.target.value })}
            />
          </Field>
        ))}
        {error && <Notice tone="negative" text={error} />}
      </div>
    </Dialog>
  );
};
