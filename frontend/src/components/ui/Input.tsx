import React from 'react';
import { Input as Base } from '@base-ui/react/input';
import { Search } from 'lucide-react';
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
