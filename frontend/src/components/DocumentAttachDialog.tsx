import React, { useEffect, useRef, useState } from 'react';
import { FileText, X } from 'lucide-react';
import { Api } from '../services/api';
import type { CompanyDocument, DocumentFileInfo, DocumentKindOption } from '../types';
import { formatDate } from '../utils/formatters';
import { Button, Dialog, Field, FileDrop, Input, Select, Textarea, toast } from './ui';

interface SelectedDocument extends DocumentFileInfo {
  path?: string;
  browserFile?: File;
}

export interface DocumentAttachDialogProps {
  open: boolean;
  dutyKey: string;
  dutyTitle: string;
  documents?: CompanyDocument[];
  onOpenDocument?: (doc: CompanyDocument) => void;
  readOnly?: boolean;
  onOpenChange: (open: boolean) => void;
  onAttached: () => void;
}

export const DocumentAttachDialog: React.FC<DocumentAttachDialogProps> = ({
  open, dutyKey, dutyTitle, documents = [], onOpenDocument, readOnly = false, onOpenChange, onAttached,
}) => {
  const [kinds, setKinds] = useState<DocumentKindOption[]>([]);
  const [kind, setKind] = useState('');
  const [title, setTitle] = useState('');
  const [documentDate, setDocumentDate] = useState('');
  const [dateSuggested, setDateSuggested] = useState(false);
  const [note, setNote] = useState('');
  const [file, setFile] = useState<SelectedDocument | null>(null);
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [reading, setReading] = useState(false);
  const selectionVersion = useRef(0);

  useEffect(() => {
    if (!open) return;
    let active = true;
    setReading(false);
    setFile(null); setTitle(''); setDocumentDate(''); setNote(''); setDateSuggested(false);
    setKind(''); setLoadError('');
    Api.getDocumentKinds().then((list) => {
      if (!active) return;
      setKinds(list); setKind(matchingKind(dutyKey, list));
    }).catch(() => { if (active) setLoadError('Dokumentarten konnten nicht geladen werden. Bitte den Dialog erneut öffnen.'); });
    return () => { active = false; selectionVersion.current++; };
  }, [open, dutyKey]);

  function applySelection(selected: SelectedDocument) {
    if (!selected.size || selected.size > 20 * 1024 * 1024) { toast.error('Bitte eine Datei mit bis zu 20 MB auswählen. Leere Dateien können nicht abgelegt werden.'); return; }
    setFile(selected);
    setTitle(selected.name.replace(/\.[^.]+$/, ''));
    const modified = new Date(selected.lastModified);
    const suggested = selected.lastModified > 0 && !Number.isNaN(modified.getTime());
    setDocumentDate(suggested ? modified.toLocaleDateString('en-CA') : '');
    setDateSuggested(suggested);
  }

  function choose(files: File[]) {
    if (files.length !== 1) { toast.error('Bitte jeweils eine Datei ablegen.'); return; }
    selectionVersion.current++;
    setReading(false);
    const selected = files[0];
    applySelection({ name: selected.name, size: selected.size, lastModified: selected.lastModified, browserFile: selected });
  }

  async function choosePaths(paths: string[]) {
    if (paths.length !== 1) { toast.error('Bitte jeweils eine Datei ablegen.'); return; }
    const version = ++selectionVersion.current;
    setReading(true);
    try {
      const info = await Api.getDocumentFileInfo(paths[0]);
      if (version === selectionVersion.current) applySelection({ ...info, path: paths[0] });
    } catch (e) {
      if (version === selectionVersion.current) toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      if (version === selectionVersion.current) setReading(false);
    }
  }

  async function attach() {
    if (!file || busy || readOnly) return;
    setBusy(true);
    try {
      const contentBase64 = file.browserFile ? await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result).split(',')[1]);
        reader.onerror = () => reject(new Error('Die Datei konnte nicht gelesen werden.'));
        reader.readAsDataURL(file.browserFile!);
      }) : undefined;
      const doc = await Api.attachDocument({ kind, title: title.trim(), documentDate, note: note.trim(), dutyKey, fileName: file.name, path: file.path, contentBase64 });
      toast.success(`${doc.fileName} liegt in den Unterlagen.`);
      onAttached();
    } catch (e) { toast.error(e instanceof Error ? e.message : String(e)); }
    finally { setBusy(false); }
  }

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!busy) onOpenChange(next); }}
      title={dutyTitle ? `Nachweise: ${dutyTitle}` : 'Unterlage ablegen'} width="max-w-2xl" dirty={file !== null && !busy}
      footer={<>
        <Button variant="secondary" disabled={busy} onClick={() => onOpenChange(false)}>Schließen</Button>
        {!readOnly && <Button variant="primary" loading={busy} disabled={!file || !kind} onClick={() => void attach()}>Nachweis ablegen</Button>}
      </>}>
      <div className="space-y-6">
        {documents.length > 0 && <section aria-label="Abgelegte Nachweise">
          <p className="text-label text-ink-muted mb-2">Bereits abgelegt</p>
          <ul className="divide-y divide-line">
            {documents.map((doc) => <li key={doc.id} className="flex items-center gap-3 py-3">
              <FileText className="h-5 w-5 shrink-0 text-ink-subtle" />
              <div className="min-w-0 flex-1">
                <button type="button" onClick={() => onOpenDocument?.(doc)} className="text-body text-accent-text hover:underline break-words text-left">{doc.title || doc.fileName}</button>
                <p className="text-caption text-ink-subtle break-words">{doc.fileName}{doc.documentDate ? ` · ${formatDate(doc.documentDate)}` : ''}</p>
                {doc.note && <p className="text-caption text-ink-muted break-words">{doc.note}</p>}
              </div>
            </li>)}
          </ul>
        </section>}
        {!readOnly && <>
          {file ? <div className="flex items-center gap-3 rounded-control border border-line px-4 py-3">
            <FileText className="h-6 w-6 shrink-0 text-accent-text" />
            <div className="min-w-0 flex-1"><p className="text-body break-words">{file.name}</p><p className="text-caption text-ink-subtle">{Math.max(1, Math.round(file.size / 1024))} KB · zum Ablegen bereit</p></div>
            <Button variant="quiet" iconOnly disabled={busy} aria-label="Dateiauswahl entfernen" onClick={() => { setFile(null); setTitle(''); setNote(''); setDocumentDate(''); }}><X className="h-4 w-4" /></Button>
          </div> : <FileDrop multiple={false} disabled={busy || reading} onFiles={choose} onPaths={(paths) => void choosePaths(paths)} hint="Nachweis hierher ziehen oder eine Datei auswählen · bis 20 MB" />}
          {reading && <p role="status" className="text-caption text-ink-muted">Datei wird gelesen …</p>}
          {file && <div className="space-y-4">
            <Field label="Bezeichnung"><Input value={title} disabled={busy} onChange={(e) => setTitle(e.target.value)} placeholder={file.name} /></Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Dokumentart"><Select value={kind} disabled={busy} onValueChange={setKind} items={kinds.map((k) => ({ value: k.value, label: k.label }))} /></Field>
              <Field label="Dokumentdatum" optional hint={dateSuggested ? 'Aus Dateiänderungsdatum vorgeschlagen. Bitte prüfen.' : undefined}>
                <Input type="date" value={documentDate} disabled={busy} onChange={(e) => { setDocumentDate(e.target.value); setDateSuggested(false); }} />
              </Field>
            </div>
            <Field label="Notiz" optional><Textarea aria-label="Notiz (optional)" value={note} disabled={busy} onChange={(e) => setNote(e.target.value)} rows={2} placeholder="Bei Bedarf ergänzen" /></Field>
          </div>}
          {loadError && <p role="alert" className="text-body text-negative-text">{loadError}</p>}
        </>}
        {readOnly && documents.length === 0 && <p className="text-body text-ink-muted">Noch keine Nachweise abgelegt.</p>}
      </div>
    </Dialog>
  );
};

function matchingKind(dutyKey: string, kinds: DocumentKindOption[]): string {
  const byDuty: Record<string, string> = {
    handelsregister: 'handelsregister', fragebogen: 'steuerliche_erfassung', gewerbeanmeldung: 'gewerbeanmeldung',
    transparenzregister: 'transparenzregister', eroeffnungsbilanz: 'eroeffnungsbilanz',
    ust_id: 'behoerde', ihk: 'behoerde', rundfunkbeitrag: 'behoerde', unfallversicherung: 'behoerde', betriebsnummer: 'behoerde',
  };
  const wanted = byDuty[dutyKey] || 'sonstiges';
  return kinds.some((k) => k.value === wanted) ? wanted : kinds[0]?.value || '';
}
