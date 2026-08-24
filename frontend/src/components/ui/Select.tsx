
import { Select as Base } from '@base-ui/react/select';
import { Check, ChevronDown } from 'lucide-react';
import { cn } from './cn';
import { CONTROL } from './Input';
import { POPUP, POPUP_ITEM } from './popup';

export interface SelectOption<T> {
  value: T;
  label: string;
  disabled?: boolean;
}

export interface SelectProps<T> {
  items: SelectOption<T>[];
  value?: T;
  defaultValue?: T;
  onValueChange?: (value: T) => void;
  placeholder?: string;
  disabled?: boolean;
  name?: string;
  className?: string;
}

/**
 * Auswahl aus einer festen Liste. Gegenüber dem nativen `select` bringt Base UI
 * Tastaturführung, Typeahead und eine Liste, die sich wie der Rest der
 * Oberfläche gestalten lässt.
 *
 * Für lange Listen wie den SKR04-Kontenrahmen ist die Kontosuche das richtige
 * Bedienelement, nicht diese Auswahl.
 */
export function Select<T extends string | number>({
  items,
  value,
  defaultValue,
  onValueChange,
  placeholder = 'Bitte wählen',
  disabled,
  name,
  className,
}: SelectProps<T>) {
  // Base UI braucht die Zuordnung Wert → Beschriftung, sonst zeigt der Auslöser
  // den rohen Wert. Als Record bleibt unsere Schnittstelle bei einfachen Werten.
  const labels = Object.fromEntries(items.map((item) => [String(item.value), item.label]));

  return (
    <Base.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={(next) => onValueChange?.(next as T)}
      disabled={disabled}
      name={name}
    >
      <Base.Trigger
        className={cn(CONTROL, 'flex items-center justify-between gap-2 text-left', className)}
      >
        {/* Die Render-Funktion bekommt den rohen Wert, nicht die Beschriftung.
            Deshalb wird hier ausdrücklich zugeordnet. */}
        <Base.Value className="truncate">
          {(selected: unknown) =>
            selected === null || selected === undefined || selected === '' ? (
              <span className="text-ink-faint">{placeholder}</span>
            ) : (
              (labels[String(selected)] ?? String(selected))
            )
          }
        </Base.Value>
        <Base.Icon className="shrink-0 text-ink-faint">
          <ChevronDown className="w-4 h-4" strokeWidth={1.5} />
        </Base.Icon>
      </Base.Trigger>

      <Base.Portal>
        <Base.Positioner sideOffset={4} className="z-50">
          <Base.Popup className={cn(POPUP, 'py-1 min-w-[var(--anchor-width)] max-h-72 overflow-auto')}>
            <Base.List>
              {items.map((item) => (
                <Base.Item key={String(item.value)} value={item.value} disabled={item.disabled} className={POPUP_ITEM}>
                  <Base.ItemIndicator className="shrink-0 text-accent-text">
                    <Check className="w-3.5 h-3.5" strokeWidth={1.5} />
                  </Base.ItemIndicator>
                  <Base.ItemText className="truncate">{item.label}</Base.ItemText>
                </Base.Item>
              ))}
            </Base.List>
          </Base.Popup>
        </Base.Positioner>
      </Base.Portal>
    </Base.Root>
  );
}
