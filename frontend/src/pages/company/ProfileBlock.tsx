import React, { useState } from 'react';
import { PenLine } from 'lucide-react';
import { useWriteLock } from '../../components/WriteLock';
import { Button, DetailPanel, Help, Notice, cn } from '../../components/ui';

/** Dieselbe Spaltenbreite in allen Blöcken, damit die Werte untereinander stehen. */
export const PROFILE_LIST = 'grid-cols-[10rem_minmax(0,1fr)]';

export interface ProfileBlockProps {
  /** Anker, über den ein Ziel den Block anspringt. */
  id: string;
  title: string;
  /** Was fehlt; leer heißt vollständig. */
  missing?: string[];
  /** Eine Zeile, wenn nichts fehlt. */
  context?: string;
  helpSummary?: string;
  explain?: React.ReactNode;
  /** Öffnet die Bearbeitung; ohne gibt es keinen Knopf. */
  onEdit?: () => void;
  editLabel?: string;
  first?: boolean;
  children: React.ReactNode;
}

/**
 * Ein Block der Stammdaten: Überschrift, was fehlt, die wichtigsten Angaben.
 * Bearbeitet wird in der Seitenleiste, Block für Block.
 */
export const ProfileBlock: React.FC<ProfileBlockProps> = ({
  id,
  title,
  missing = [],
  context,
  helpSummary,
  explain,
  onEdit,
  editLabel = 'Bearbeiten',
  first = false,
  children,
}) => {
  const writeLock = useWriteLock();
  return (
    <section id={id} className={cn(!first && 'mt-8 pt-8 border-t border-line')}>
      <div className="flex items-start justify-between gap-4 mb-5">
        <div className="min-w-0">
          <h2 aria-label={title} className="flex flex-wrap items-center text-heading text-ink">
            {title}
            {explain && helpSummary && (
              <Help summary={helpSummary} label={`Erklärung zu ${title}`}>
                {explain}
              </Help>
            )}
          </h2>
          {missing.length > 0 ? (
            <p className="text-caption text-attention-text mt-1">Es fehlt: {missing.join(', ')}</p>
          ) : (
            context && <p className="text-caption text-ink-subtle mt-1">{context}</p>
          )}
        </div>
        {onEdit && (
          <Button
            variant="secondary"
            size="sm"
            disabled={writeLock.locked}
            title={writeLock.hint}
            icon={<PenLine className="w-4 h-4" strokeWidth={1.5} />}
            onClick={onEdit}
          >
            {editLabel}
          </Button>
        )}
      </div>
      {children}
    </section>
  );
};

export interface EditPanelProps {
  open: boolean;
  title: string;
  context?: string;
  onClose: () => void;
  /** Speichert; ein Fehler bleibt in der Leiste stehen. */
  onSave: () => Promise<void>;
  saveLabel?: string;
  children: React.ReactNode;
}

/** Die Seitenleiste zum Bearbeiten eines Blocks mit Abbrechen und Speichern. */
export const EditPanel: React.FC<EditPanelProps> = ({
  open,
  title,
  context,
  onClose,
  onSave,
  saveLabel = 'Speichern',
  children,
}) => {
  const writeLock = useWriteLock();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  async function save() {
    setSaving(true);
    setError('');
    try {
      await onSave();
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <DetailPanel
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setError('');
          onClose();
        }
      }}
      title={title}
      context={context}
      width="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void save()}
          >
            {saveLabel}
          </Button>
        </>
      }
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
      >
        {error && <Notice tone="negative" className="mb-5" text={error} />}
        {children}
        <button type="submit" className="hidden" aria-hidden="true" tabIndex={-1} />
      </form>
    </DetailPanel>
  );
};

/** Ein Wert, der fehlt, in der Farbe für Offenes. */
export const Missing: React.FC<{ children?: React.ReactNode }> = ({ children = 'fehlt' }) => (
  <span className="text-attention-text">{children}</span>
);
