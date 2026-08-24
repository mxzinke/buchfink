import React from 'react';
import { Loader2 } from 'lucide-react';
import { cn } from './cn';

export type ButtonVariant = 'primary' | 'secondary' | 'quiet' | 'danger';
export type ButtonSize = 'md' | 'sm';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Icon vor der Beschriftung. 14 px bei `sm`, 16 px bei `md`. */
  icon?: React.ReactNode;
  /**
   * Zeigt den Ladezustand. Der Button behält Position und Breite, die
   * Beschriftung bleibt stehen und wird nur unsichtbar (§8.4).
   */
  loading?: boolean;
  /** Quadratisch, nur Icon. Braucht immer `title` und `aria-label`. */
  iconOnly?: boolean;
}

const BASE =
  'relative inline-flex items-center justify-center gap-2 rounded-control ' +
  'text-label font-semibold whitespace-nowrap select-none ' +
  'transition-colors duration-120 ease-quiet ' +
  'disabled:cursor-not-allowed';

const VARIANTS: Record<ButtonVariant, string> = {
  // Genau eine Primäraktion pro Ansicht. Tinte, nicht Himmelblau (§3.4).
  primary: 'bg-ink text-paper hover:bg-ink-muted disabled:bg-sunken disabled:text-ink-faint',
  secondary:
    'border border-control-border bg-surface text-ink-muted ' +
    'hover:bg-sunken hover:text-ink disabled:bg-transparent disabled:text-ink-faint disabled:border-line',
  quiet:
    'text-ink-subtle hover:bg-sunken hover:text-ink disabled:text-ink-faint disabled:hover:bg-transparent',
  danger: 'bg-negative-text text-white hover:brightness-95 disabled:bg-sunken disabled:text-ink-faint',
};

const SIZES: Record<ButtonSize, string> = {
  md: 'h-9 px-4',
  sm: 'h-8 px-2.5',
};

const ICON_ONLY: Record<ButtonSize, string> = {
  md: 'h-9 w-9 px-0',
  sm: 'h-8 w-8 px-0',
};

/**
 * `forwardRef` ist Pflicht: Base UI hängt den Button über `render` an seine
 * Auslöser (Menü, Popover, Dialog schließen) und braucht dafür die Ref auf das
 * echte Element. Ohne sie findet die Positionierung keinen Anker.
 */
export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = 'secondary',
    size = 'md',
    icon,
    loading = false,
    iconOnly = false,
    disabled,
    className,
    children,
    ...props
  },
  ref,
) {
  return (
    <button
      ref={ref}
      type="button"
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(BASE, VARIANTS[variant], iconOnly ? ICON_ONLY[size] : SIZES[size], className)}
      {...props}
    >
      <span className={cn('inline-flex items-center gap-2', loading && 'invisible')}>
        {icon}
        {children}
      </span>
      {loading && <Loader2 className="absolute w-4 h-4 animate-spin" aria-hidden="true" />}
    </button>
  );
});
