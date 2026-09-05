import React from 'react';
import { Field as Base } from '@base-ui/react/field';
import { cn } from './cn';
import { HelpPopover, HelpTooltip } from './Help';

/**
 * Base UI verdrahtet Label, Beschreibung, Fehler und Bedienelement
 * untereinander: id, aria-describedby und aria-invalid entstehen von selbst.
 * Hier kommt nur die Gestalt dazu.
 */
export interface FieldProps {
  label: string;
  /** Höchstens sechs Wörter. Was länger ist, gehört in `help` (§15.1). */
  hint?: string;
  /** Ersetzt den Hinweis, solange er steht. */
  error?: string;
  /** Ein Satz hinter dem Erklärzeichen (§15.2). */
  help?: string;
  /**
   * Bis drei Sätze hinter dem Erklärzeichen — die zweite Stufe der Erklärung
   * (§15.2). Für Text, dessen Länge die Ansicht nicht in der Hand hat, etwa
   * eine Auskunft aus dem Backend. Ein Tooltip trüge ihn nicht.
   */
  explain?: React.ReactNode;
  /** Gekennzeichnet wird das Seltenere: optional, nicht Pflicht. */
  optional?: boolean;
  disabled?: boolean;
  name?: string;
  className?: string;
  children: React.ReactNode;
}

export const Field: React.FC<FieldProps> = ({
  label,
  hint,
  error,
  help,
  explain,
  optional = false,
  disabled,
  name,
  className,
  children,
}) => (
  <Base.Root
    name={name}
    disabled={disabled}
    invalid={Boolean(error)}
    className={cn('flex flex-col gap-1 min-w-0', className)}
  >
    <div className="flex items-center">
      <Base.Label className="text-label text-ink-muted">
        {label}
        {optional && <span className="text-ink-subtle font-normal"> · optional</span>}
      </Base.Label>
      {help && <HelpTooltip content={help} label={`Erklärung zu ${label}`} />}
      {explain && <HelpPopover label={`Erklärung zu ${label}`}>{explain}</HelpPopover>}
    </div>

    {children}

    {error ? (
      // `match` erzwingt die Anzeige. Ohne das Prop richtet sich Base UI nach
      // dem Validity-State des Controls, und ein fachlicher Fehler aus dem
      // Backend taucht dort nie auf.
      <Base.Error match className="text-caption text-negative-text">
        {error}
      </Base.Error>
    ) : hint ? (
      <Base.Description className="text-caption text-ink-subtle">{hint}</Base.Description>
    ) : null}
  </Base.Root>
);

/** Mehrere Felder nebeneinander, ohne dass jede Seite ein eigenes Raster baut. */
export const FieldRow: React.FC<{ className?: string; children: React.ReactNode }> = ({
  className,
  children,
}) => <div className={cn('flex flex-wrap items-start gap-4', className)}>{children}</div>;
