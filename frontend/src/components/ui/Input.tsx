import React from 'react';
import { Input as Base } from '@base-ui/react/input';
import { NumberField } from '@base-ui/react/number-field';
import { Minus, Plus, Search } from 'lucide-react';
import { cn } from './cn';

/**
 * Weißes Feld auf Papiergrund. Das ist die Bedienbarkeits-Aussage, es braucht
 * keine Fläche darum herum (§6.1). Der ungültige Zustand kommt über
 * `data-invalid` aus dem umgebenden Field.
 */
export const CONTROL =
  'h-9 w-full px-3 rounded-control border border-control-border bg-surface text-body text-ink ' +
  'placeholder:text-ink-faint transition-colors duration-120 ease-quiet ' +
  'focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent/25 ' +
  'data-[invalid]:border-negative data-[invalid]:ring-2 data-[invalid]:ring-negative/20 ' +
  'data-[disabled]:bg-sunken data-[disabled]:text-ink-faint data-[disabled]:cursor-not-allowed ' +
  'disabled:bg-sunken disabled:text-ink-faint disabled:cursor-not-allowed';

export interface InputProps extends React.ComponentProps<typeof Base> {
  /** `right` setzt zusätzlich tabellarische Ziffern. Für alle Beträge (§4.1). */
  align?: 'left' | 'right';
}

export const Input: React.FC<InputProps> = ({ align = 'left', className, ...props }) => (
  <Base className={cn(CONTROL, align === 'right' && 'text-right num', className)} {...props} />
);

export interface AmountInputProps
  extends Omit<React.ComponentProps<typeof NumberField.Root>, 'format'> {
  /** Ohne Währung, etwa für Mengen oder Prozentsätze. */
  currency?: string | null;
  className?: string;
}

/**
 * Betragsfeld. Deckt §8.6 ab: Pfeiltasten erhöhen um 1,00 €, mit Shift um
 * 100,00 €. Die Formatierung ist de-DE, dieselbe Schreibweise wie in
 * `utils/formatters.ts`.
 */
export const AmountInput: React.FC<AmountInputProps> = ({
  currency = 'EUR',
  className,
  ...props
}) => (
  <NumberField.Root
    step={1}
    largeStep={100}
    smallStep={0.01}
    locale="de-DE"
    format={
      currency
        ? { style: 'currency', currency, minimumFractionDigits: 2 }
        : { minimumFractionDigits: 2, maximumFractionDigits: 2 }
    }
    className={cn('min-w-0', className)}
    {...props}
  >
    <NumberField.Group className="relative flex">
      <NumberField.Decrement
        className="absolute left-0 top-0 h-9 w-7 grid place-items-center text-ink-faint hover:text-ink-muted transition-colors duration-120 ease-quiet"
        aria-label="Betrag verringern"
      >
        <Minus className="w-3.5 h-3.5" strokeWidth={1.5} />
      </NumberField.Decrement>
      <NumberField.Input className={cn(CONTROL, 'px-7 text-right num')} />
      <NumberField.Increment
        className="absolute right-0 top-0 h-9 w-7 grid place-items-center text-ink-faint hover:text-ink-muted transition-colors duration-120 ease-quiet"
        aria-label="Betrag erhöhen"
      >
        <Plus className="w-3.5 h-3.5" strokeWidth={1.5} />
      </NumberField.Increment>
    </NumberField.Group>
  </NumberField.Root>
);

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

export const Textarea: React.FC<TextareaProps> = ({ className, ...props }) => (
  <textarea className={cn(CONTROL, 'h-auto min-h-20 py-2 leading-5', className)} {...props} />
);

export interface SearchInputProps extends Omit<InputProps, 'type'> {}

/** Suchfeld mit Lupe. Der Platzhalter sagt, wonach gesucht wird. */
export const SearchInput: React.FC<SearchInputProps> = ({ className, ...props }) => (
  <div className="relative min-w-0 flex-1">
    <Search
      className="pointer-events-none absolute left-3 top-1/2 w-4 h-4 -translate-y-1/2 text-ink-faint"
      strokeWidth={1.5}
    />
    <Input type="search" className={cn('pl-9', className)} {...props} />
  </div>
);
