import React from 'react';
import { Checkbox as BaseCheckbox } from '@base-ui/react/checkbox';
import { Radio as BaseRadio } from '@base-ui/react/radio';
import { RadioGroup as BaseRadioGroup } from '@base-ui/react/radio-group';
import { Switch as BaseSwitch } from '@base-ui/react/switch';
import { Check, Minus } from 'lucide-react';
import { cn } from './cn';

/** Gemeinsamer Rahmen für Kästchen und Schalter. */
const BOX =
  'grid place-items-center w-4 h-4 shrink-0 rounded-[4px] border border-control-border bg-surface ' +
  'transition-colors duration-120 ease-quiet ' +
  'data-[checked]:bg-accent data-[checked]:border-accent data-[checked]:text-white ' +
  'data-[indeterminate]:bg-accent data-[indeterminate]:border-accent data-[indeterminate]:text-white ' +
  'data-[disabled]:bg-sunken data-[disabled]:border-line data-[disabled]:cursor-not-allowed';

const ROW = 'inline-flex items-start gap-2.5 text-body text-ink cursor-default';

export interface CheckboxProps extends React.ComponentProps<typeof BaseCheckbox.Root> {
  label: React.ReactNode;
  /** Ein kurzer Zusatz unter der Beschriftung. */
  hint?: string;
}

export const Checkbox: React.FC<CheckboxProps> = ({ label, hint, className, ...props }) => (
  <label className={cn(ROW, className)}>
    <BaseCheckbox.Root className={cn(BOX, 'mt-0.5')} {...props}>
      <BaseCheckbox.Indicator className="grid place-items-center data-[unchecked]:hidden">
        {props.indeterminate ? (
          <Minus className="w-3 h-3" strokeWidth={2} />
        ) : (
          <Check className="w-3 h-3" strokeWidth={2} />
        )}
      </BaseCheckbox.Indicator>
    </BaseCheckbox.Root>
    <span className="min-w-0">
      <span className="block">{label}</span>
      {hint && <span className="block text-caption text-ink-subtle">{hint}</span>}
    </span>
  </label>
);

export interface RadioOption<T> {
  value: T;
  label: React.ReactNode;
  hint?: string;
  disabled?: boolean;
}

export interface RadioGroupProps<T> {
  options: RadioOption<T>[];
  value?: T;
  defaultValue?: T;
  onValueChange?: (value: T) => void;
  name?: string;
  disabled?: boolean;
  /** Nebeneinander statt untereinander. Nur bei zwei kurzen Optionen. */
  inline?: boolean;
  className?: string;
}

export function RadioGroup<T extends string | number>({
  options,
  value,
  defaultValue,
  onValueChange,
  name,
  disabled,
  inline = false,
  className,
}: RadioGroupProps<T>) {
  return (
    <BaseRadioGroup
      value={value}
      defaultValue={defaultValue}
      onValueChange={(next) => onValueChange?.(next as T)}
      name={name}
      disabled={disabled}
      className={cn('flex gap-x-6 gap-y-2', inline ? 'flex-row flex-wrap' : 'flex-col', className)}
    >
      {options.map((option) => (
        <label key={String(option.value)} className={ROW}>
          <BaseRadio.Root
            value={option.value}
            disabled={option.disabled}
            className={cn(BOX, 'rounded-full mt-0.5')}
          >
            <BaseRadio.Indicator className="w-1.5 h-1.5 rounded-full bg-white data-[unchecked]:hidden" />
          </BaseRadio.Root>
          <span className="min-w-0">
            <span className="block">{option.label}</span>
            {option.hint && <span className="block text-caption text-ink-subtle">{option.hint}</span>}
          </span>
        </label>
      ))}
    </BaseRadioGroup>
  );
}

export interface SwitchProps extends React.ComponentProps<typeof BaseSwitch.Root> {
  label?: React.ReactNode;
}

/**
 * Nur für Einstellungen, die sofort wirken. Alles, was erst beim Speichern
 * greift, ist ein Kästchen (§8.1).
 */
export const Switch: React.FC<SwitchProps> = ({ label, className, ...props }) => {
  const control = (
    <BaseSwitch.Root
      className={cn(
        'relative h-5 w-9 shrink-0 rounded-full border border-control-border bg-sunken',
        'transition-colors duration-120 ease-quiet',
        'data-[checked]:bg-accent data-[checked]:border-accent',
        'data-[disabled]:cursor-not-allowed data-[disabled]:opacity-60',
        !label && className,
      )}
      {...props}
    >
      <BaseSwitch.Thumb
        className="block h-3.5 w-3.5 rounded-full bg-surface shadow-popover
                   transition-transform duration-120 ease-quiet translate-x-0.5
                   data-[checked]:translate-x-[18px]"
      />
    </BaseSwitch.Root>
  );

  if (!label) return control;

  return (
    <label className={cn('inline-flex items-center gap-2.5 text-body text-ink', className)}>
      {control}
      <span>{label}</span>
    </label>
  );
};
