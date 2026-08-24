import { useMemo, useState } from 'react';
import { Combobox as Base } from '@base-ui/react/combobox';
import { Check } from 'lucide-react';
import { cn } from './cn';
import { CONTROL } from './Input';
import { POPUP, POPUP_ITEM } from './popup';

export interface ComboboxOption<T> {
  value: T;
  /** Wird durchsucht und angezeigt, etwa "4400 Erlöse 19 % USt". */
  label: string;
  /** Zweite Zeile, etwa die Kontoart. Wird nicht durchsucht. */
  meta?: string;
  disabled?: boolean;
}

export interface ComboboxProps<T> {
  items: ComboboxOption<T>[];
  value?: T | null;
  onValueChange?: (value: T | null) => void;
  placeholder?: string;
  /** Text, wenn die Suche nichts findet. Ein Satz. */
  emptyText?: string;
  disabled?: boolean;
  name?: string;
  /** Obergrenze der angezeigten Treffer. SKR04 hat rund 1500 Konten. */
  limit?: number;
  className?: string;
}

/**
 * Die Kontosuche aus §8.6: Die Liste öffnet beim Tippen, Enter übernimmt den
 * Treffer. Das ist das Bedienelement für den SKR04-Kontenrahmen, für kurze
 * feste Listen bleibt es bei `Select`.
 *
 * Gefiltert wird hier und nicht in Base UI, damit die Suche auch die zweite
 * Zeile eines Eintrags erfassen kann, wenn wir das später brauchen.
 */
export function Combobox<T extends string | number>({
  items,
  value,
  onValueChange,
  placeholder = 'Suchen …',
  emptyText = 'Kein Treffer.',
  disabled,
  name,
  limit = 100,
  className,
}: ComboboxProps<T>) {
  const [query, setQuery] = useState('');

  const matches = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((item) => item.label.toLowerCase().includes(q));
  }, [items, query]);

  const filtered = matches.slice(0, limit);
  const truncated = matches.length - filtered.length;

  // Zuordnung Wert → Beschriftung, sonst steht der rohe Wert im Feld.
  const labels = useMemo(
    () => Object.fromEntries(items.map((item) => [String(item.value), item.label])),
    [items],
  );

  return (
    <Base.Root
      itemToStringLabel={(item) => labels[String(item)] ?? ''}
      value={value ?? null}
      onValueChange={(next) => onValueChange?.((next as T) ?? null)}
      onInputValueChange={setQuery}
      // Der erste Treffer ist vorausgewählt, damit Enter direkt übernimmt (§8.6).
      autoHighlight
      disabled={disabled}
      name={name}
    >
      <Base.Input className={cn(CONTROL, className)} placeholder={placeholder} />

      <Base.Portal>
        <Base.Positioner sideOffset={4} className="z-50">
          <Base.Popup
            className={cn(POPUP, 'py-1 min-w-[var(--anchor-width)] max-h-72 overflow-auto')}
          >
            <Base.Empty className="px-3 py-2 text-caption text-ink-subtle">{emptyText}</Base.Empty>
            <Base.List>
              {filtered.map((item) => (
                <Base.Item
                  key={String(item.value)}
                  value={item.value}
                  disabled={item.disabled}
                  className={cn(POPUP_ITEM, 'h-auto py-1.5')}
                >
                  <Base.ItemIndicator className="shrink-0 text-accent-text">
                    <Check className="w-3.5 h-3.5" strokeWidth={1.5} />
                  </Base.ItemIndicator>
                  <span className="min-w-0 flex flex-col">
                    <span className="truncate">{item.label}</span>
                    {item.meta && (
                      <span className="text-caption text-ink-subtle truncate">{item.meta}</span>
                    )}
                  </span>
                </Base.Item>
              ))}
            </Base.List>
            {truncated > 0 && (
              <span className="block px-3 py-2 text-caption text-ink-subtle border-t border-line">
                {truncated} weitere Treffer. Suche verfeinern.
              </span>
            )}
          </Base.Popup>
        </Base.Positioner>
      </Base.Portal>
    </Base.Root>
  );
}
