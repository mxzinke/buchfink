import React from 'react';
import { cn } from './cn';

/**
 * Die Datentabelle ist die eine Stelle im Arbeitsbereich, die eine eigene
 * Fläche bekommt (§6.2, Fall 2): eigene Spalten, eigenes seitliches Scrollen,
 * eine Kopfzeile, die stehen bleibt.
 *
 * Die Überschrift des Abschnitts steht über der Fläche, nicht darin. Sonst
 * entsteht wieder eine Karte mit Kopfzeile.
 *
 * Die Fläche ist ein Container: Wird sie schmaler als 56rem, treten Spalten
 * mit `optional` zurück. Kennung, Betrag und Stand sind nie optional, sie
 * sind das, was die Liste zuerst zeigen soll.
 */
export interface TableProps extends React.TableHTMLAttributes<HTMLTableElement> {
  density?: 'komfortabel' | 'kompakt';
  className?: string;
}

const DENSITY = {
  komfortabel: '[&_tbody_td]:h-10',
  kompakt: '[&_tbody_td]:h-8',
};

export const Table: React.FC<TableProps> = ({
  density = 'komfortabel',
  className,
  children,
  ...props
}) => (
  <div className="@container rounded-card border border-line bg-surface overflow-hidden">
    <div className="overflow-x-auto">
      <table
        className={cn('w-full border-collapse text-body text-ink', DENSITY[density], className)}
        {...props}
      >
        {children}
      </table>
    </div>
  </div>
);

export interface TheadProps extends React.HTMLAttributes<HTMLTableSectionElement> {
  /** Bleibt beim Scrollen stehen. Braucht `bg-surface`, damit die Zeilen
   *  nicht durchscheinen. */
  sticky?: boolean;
}

export const Thead: React.FC<TheadProps> = ({ sticky = false, className, children, ...props }) => (
  <thead
    className={cn(sticky && '[&_th]:sticky [&_th]:top-0 [&_th]:bg-surface [&_th]:z-10', className)}
    {...props}
  >
    {children}
  </thead>
);

/** Keine Zebrastreifen. Haarlinien und der Hover reichen. */
export const Tbody: React.FC<React.HTMLAttributes<HTMLTableSectionElement>> = ({
  className,
  children,
  ...props
}) => (
  <tbody
    className={cn(
      '[&>tr:hover>td]:bg-sunken [&>tr:last-child>td]:border-b-0',
      // Die Transition muss an der Zelle sitzen, nicht am tbody. Sonst
      // wechselt der Hover-Hintergrund ohne Übergang.
      '[&>tr>td]:transition-colors [&>tr>td]:duration-120 [&>tr>td]:ease-quiet',
      className,
    )}
    {...props}
  >
    {children}
  </tbody>
);

export type RowVariant = 'default' | 'selected' | 'storno' | 'sum';

export interface TrProps extends React.HTMLAttributes<HTMLTableRowElement> {
  variant?: RowVariant;
}

const ROW: Record<RowVariant, string> = {
  default: '',
  selected: '[&>td]:bg-accent-soft',
  // Tönung, Markierung und Badge. Nie durchgestrichen, der Betrag bleibt
  // lesbar (§11.2). Die Markierung ist eine Pille im Pseudo-Element, keine
  // einseitige Border: Die wuerde am Eckenradius der Flaeche krumm laufen.
  storno:
    '[&>td]:bg-negative-soft/60 [&>td:first-child]:relative ' +
    "[&>td:first-child]:before:content-[''] [&>td:first-child]:before:absolute " +
    '[&>td:first-child]:before:left-1.5 [&>td:first-child]:before:top-1/2 ' +
    '[&>td:first-child]:before:h-4 [&>td:first-child]:before:w-0.5 ' +
    '[&>td:first-child]:before:-translate-y-1/2 [&>td:first-child]:before:rounded-full ' +
    '[&>td:first-child]:before:bg-negative',
  // Buchhalterische Doppellinie. Sie genügt allein, ohne Füllung.
  sum: '[&>td]:rule-total [&>td]:font-semibold [&>td]:border-b-0',
};

export const Tr: React.FC<TrProps> = ({ variant = 'default', className, children, ...props }) => (
  <tr className={cn(ROW[variant], className)} {...props}>
    {children}
  </tr>
);

interface CellCommon {
  /** Rechtsbündig mit tabellarischen Ziffern. Pflicht für jede Zahlenspalte. */
  numeric?: boolean;
  /** Spalte, die bei schmaler Tabelle ausgeblendet wird. Kopf und Zellen gleich setzen. */
  optional?: boolean;
}

/** Unter 56rem Tabellenbreite, also bei einem Fenster unter rund 1200 px. */
const OPTIONAL = '@max-4xl:hidden';

export interface ThProps extends React.ThHTMLAttributes<HTMLTableCellElement>, CellCommon {}

export const Th: React.FC<ThProps> = ({ numeric, optional, className, children, ...props }) => (
  <th
    scope="col"
    className={cn(
      'h-[34px] px-4 border-b border-line-strong text-label font-medium text-ink-subtle whitespace-nowrap',
      numeric ? 'text-right' : 'text-left',
      optional && OPTIONAL,
      className,
    )}
    {...props}
  >
    {children}
  </th>
);

export interface TdProps extends React.TdHTMLAttributes<HTMLTableCellElement>, CellCommon {
  /** Beleg- und Kontonummern: tabellarisch mit durchgestrichener Null. */
  code?: boolean;
  /**
   * Die Textspalte der Zeile: nimmt den freien Platz und kürzt mit Auslassung,
   * statt die Tabelle zu verbreitern. Eine Spalte je Tabelle.
   */
  fill?: boolean;
}

export const Td: React.FC<TdProps> = ({ numeric, code, optional, fill, className, children, ...props }) => (
  <td
    className={cn(
      'px-4 border-b border-line align-middle whitespace-nowrap',
      numeric && 'text-right num',
      code && 'code-num text-caption text-ink-muted',
      optional && OPTIONAL,
      fill && 'w-full max-w-0 truncate',
      className,
    )}
    {...props}
  >
    {children}
  </td>
);

export interface RowButtonProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onClick'> {
  /** Öffnet, was auch der Klick auf die Zeile öffnet. */
  onOpen: () => void;
}

/**
 * Die Kennung einer Zeile, die eine Seitenleiste öffnet. Als Knopf erreicht
 * sie die Tastatur; der Klick auf die übrige Zeile ist die Abkürzung für die
 * Maus. Steht in der ersten Zelle.
 */
export const RowButton: React.FC<RowButtonProps> = ({ onOpen, className, children, ...props }) => (
  <button
    type="button"
    className={cn('max-w-full truncate text-left underline-offset-2 hover:underline', className)}
    onClick={(event) => {
      event.stopPropagation();
      onOpen();
    }}
    {...props}
  >
    {children}
  </button>
);
