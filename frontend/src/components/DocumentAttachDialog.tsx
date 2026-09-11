import React, { useEffect, useState } from 'react';
import { Api } from '../services/api';
import type { DocumentKindOption } from '../types';
import { Button, Dialog, Field, FormGrid, Input, Select, Textarea, toast } from './ui';

/**
 * Eine Unterlage in die Ablage des Unternehmens legen.
 *
 * Sie liegt dort dauerhaft und nicht am Vorgang: der Registerauszug bleibt
 * auffindbar, wenn die Gründung längst vorbei ist. Verknüpft wird er trotzdem
 * mit seinem Schritt — dann steht er dort als Nachweis, ohne ihm zu gehören.
 *
 * Die Ablage ist freiwillig. Ein Schritt gilt mit seinem Datum als erledigt und
 * nicht erst mit einer Datei; wer seine Papiere anderswo führt, soll das dürfen.
 */

export interface DocumentAttachDialogProps {
  open: boolean;
  /** Leer, wenn die Unterlage zu keiner Gründungspflicht gehört. */
  dutyKey: string;
  dutyTitle: string;
  onOpenChange: (open: boolean) => void;
  onAttached: () => void;
}

export const DocumentAttachDialog: React.FC<DocumentAttachDialogProps> = ({
  open,
  dutyKey,
  dutyTitle,
  onOpenChange,
  onAttached,
}) => {
  const [kinds, setKinds] = useState<DocumentKindOption[]>([]);
  const [kind, setKind] = useState('');
  const [title, setTitle] = useState('');
  const [documentDate, setDocumentDate] = useState('');
  const [note, setNote] = useState('');
  const [path, setPath] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    Api.getDocumentKinds()
      .then((list) => {
        setKinds(list);
        // Die Art, die zum Schritt passt, ist vorbelegt: der Registerauszug
        // gehört zur Anmeldung, der Gewerbeschein zur Gewerbeanmeldung. Die
        // Schlüssel stimmen an der Stelle überein, wo es einen passenden gibt.
        setKind((current) => current || matchingKind(dutyKey, list));
      })
      .catch(() => setKinds([]));
  }, [dutyKey]);

  // Bei jedem Öffnen von vorn: eine Maske, die die vorige Eingabe zeigt, ist
  // ein Angebot, dasselbe zweimal abzulegen.
  useEffect(() => {
    if (!open) return;
    setTitle('');
    setDocumentDate('');
    setNote('');
    setPath('');
  }, [open]);

  async function choose() {
    try {
      const files = await Api.selectDocuments();
      if (files.length > 0) setPath(files[0]);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  async function attach() {
    setBusy(true);
    try {
      const doc = await Api.attachDocument({
        kind,
        title: title.trim(),
        documentDate,
        note: note.trim(),
        dutyKey,
        path,
      });
      toast.success(`${doc.fileName} liegt in den Unterlagen.`);
      onAttached();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={dutyTitle ? `Nachweis zu „${dutyTitle}“` : 'Unterlage ablegen'}
      width="max-w-xl"
      dirty={path !== '' || title !== ''}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={path === '' || kind === ''}
            onClick={() => void attach()}
          >
            Ablegen
          </Button>
        </>
      }
    >
      <FormGrid>
        <Field helpSummary="Die Originaldatei bleibt unverändert erhalten."
          label="Datei"
          explain="Die Datei wird unter ihrer Prüfsumme abgelegt und bleibt unverändert."
        >
          <div className="flex items-center gap-2">
            <Input
              value={path}
              readOnly
              placeholder="Keine Datei gewählt"
              className="flex-1"
            />
            <Button variant="secondary" onClick={() => void choose()}>
              Auswählen
            </Button>
          </div>
        </Field>

        <Field label="Art">
          <Select
            value={kind}
            onValueChange={setKind}
            items={kinds.map((k) => ({ value: k.value, label: k.label }))}
          />
        </Field>

        <Field label="Bezeichnung" hint="Leer heißt: der Dateiname">
          <Input value={title} onChange={(e) => setTitle(e.target.value)} />
        </Field>

        <Field helpSummary="Dieses Datum hilft zu bestimmen, wie lange Sie die Unterlage aufbewahren müssen." label="Datum des Dokuments" explain="Aus ihm folgt der Beginn der Aufbewahrungsfrist.">
          <Input
            type="date"
            value={documentDate}
            onChange={(e) => setDocumentDate(e.target.value)}
          />
        </Field>

        <Field label="Notiz">
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} />
        </Field>
      </FormGrid>
    </Dialog>
  );
};

/**
 * Die Dokumentart, die zu einem Gründungsschritt passt.
 *
 * Die Zuordnung steht hier und nicht im Backend: sie ist eine Vorbelegung der
 * Maske und keine Regel — wer den Registerauszug als „Sonstiges" ablegen will,
 * darf das.
 */
function matchingKind(dutyKey: string, kinds: DocumentKindOption[]): string {
  const byDuty: Record<string, string> = {
    handelsregister: 'handelsregister',
    fragebogen: 'steuerliche_erfassung',
    gewerbeanmeldung: 'gewerbeanmeldung',
    transparenzregister: 'transparenzregister',
    eroeffnungsbilanz: 'eroeffnungsbilanz',
  };
  const wanted = byDuty[dutyKey];
  if (wanted && kinds.some((k) => k.value === wanted)) return wanted;
  return kinds.length > 0 ? kinds[kinds.length - 1].value : '';
}
