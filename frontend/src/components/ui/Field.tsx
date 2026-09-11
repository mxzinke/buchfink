import React from 'react';
import { Field as Base } from '@base-ui/react/field';
import { cn } from './cn';
import { Help } from './Help';

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
  /** Ausführliche Erklärung im Dialog hinter „Mehr erfahren“. */
  explain?: React.ReactNode;
  /** Ein kurzer Satz für den Tooltip. */
  helpSummary?: string;
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
  explain,
  helpSummary,
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
      {explain && <Help summary={helpSummary!} label={`Erklärung zu ${label}`}>{explain}</Help>}
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

/**
 * Eine Auskunft an der Stelle eines Bedienelements.
 *
 * Für Werte, die zum Feld gehören, aber nicht gewählt werden: das Format einer
 * Rechnung steht am Empfänger, der Kontenrahmen am Mandanten. Früher stand
 * dort ein gesperrtes Eingabefeld — das ist kein Bedienelement, sondern Text
 * in Deaktiviert-Optik, und §3.1 hält `ink-faint` von Fließtext fern. Die Höhe
 * ist die eines Bedienelements, damit die Zeile neben den Feldern nicht
 * verspringt.
 */
export const FieldValue: React.FC<{ className?: string; children: React.ReactNode }> = ({
  className,
  children,
}) => (
  <p className={cn('flex h-9 items-center text-body text-ink min-w-0', className)}>{children}</p>
);

/** Mehrere Felder nebeneinander, ohne dass jede Seite ein eigenes Raster baut. */
export const FieldRow: React.FC<{ className?: string; children: React.ReactNode }> = ({
  className,
  children,
}) => <div className={cn('flex flex-wrap items-start gap-4', className)}>{children}</div>;

export interface FormGridProps {
  /**
   * Eine Spalte für Werte, die die Breite brauchen — Pfade, lange Freitexte.
   * Sonst zwei; ein einzelnes Feld steht dann in der linken Spalte und ist so
   * breit wie die Felder darüber.
   */
  cols?: 1 | 2;
  className?: string;
  children: React.ReactNode;
}

/**
 * Das Raster eines Formulars.
 *
 * Es steht hier und nicht als Klassenkette in jeder Ansicht: Sonst hat jeder
 * Abschnitt sein eigenes Raster, und die Abschnitte einer Seite laufen
 * auseinander — mal zwei Spalten über die volle Breite, mal eine Spalte mit
 * `max-w-2xl`, mal ein einzelnes Feld mit `max-w-sm`. Nebeneinander gestellt
 * sieht das aus, als wäre jede Zeile für sich entstanden.
 *
 * Ein Feld über beide Spalten bekommt `className="md:col-span-2"`.
 */
export const FormGrid: React.FC<FormGridProps> = ({ cols = 2, className, children }) => (
  <div
    className={cn('grid grid-cols-1 gap-x-6 gap-y-5', cols === 2 && 'md:grid-cols-2', className)}
  >
    {children}
  </div>
);
