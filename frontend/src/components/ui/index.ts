/**
 * Die Bausteine des Design-Konzepts. Spezifikation: docs/design-konzept.md.
 *
 * Verhalten kommt von Base UI (`@base-ui/react`): Fokusfang, Positionierung mit
 * Kollisionsprüfung, Tastaturführung, ARIA. Gestalt kommt von hier, in
 * Tailwind-Klassen aus den Tokens in index.css.
 *
 * Rein darstellende Bausteine ohne Verhalten bauen wir selbst, weil es dort
 * nichts falsch zu machen gibt: Button, Section, PageHeader, StatRow, Table,
 * StatusBadge, EmptyState.
 *
 * Bewusst gibt es keine `Card`. Wo eine Fläche nötig ist (§6.2), steht sie an
 * genau dieser Stelle im Code und nicht als Baustein, der sich unbemerkt
 * vermehrt.
 */
export { cn } from './cn';
export { BACKDROP, POPUP, POPUP_ITEM, TOOLTIP_POPUP } from './popup';

export { Button, type ButtonProps, type ButtonSize, type ButtonVariant } from './Button';
export { Combobox, type ComboboxOption, type ComboboxProps } from './Combobox';
export { ConfirmDialog, Dialog, type ConfirmDialogProps, type DialogProps } from './Dialog';
export { EmptyState, type EmptyStateProps } from './EmptyState';
export {
  Notice,
  Progress,
  Skeleton,
  SkeletonRows,
  toast,
  type NoticeProps,
  type ProgressProps,
  type SkeletonRowsProps,
} from './Feedback';
export { FileDrop, type FileDropProps } from './FileDrop';
export { Field, FieldRow, FieldValue, type FieldProps } from './Field';
export {
  HelpPopover,
  HelpTooltip,
  InfoPopover,
  type HelpPopoverProps,
  type HelpTooltipProps,
  type InfoPopoverProps,
} from './Help';
export {
  AmountInput,
  CONTROL,
  Input,
  SearchInput,
  Textarea,
  type AmountInputProps,
  type InputProps,
  type SearchInputProps,
  type TextareaProps,
} from './Input';
export {
  Menu,
  MenuCheckItem,
  MenuGroup,
  MenuItem,
  MenuSeparator,
  type MenuProps,
} from './Menu';
export { PageHeader, Section, type PageHeaderProps, type SectionProps } from './Section';
export { SHELL_BUTTON, SHELL_CONTROL, SHELL_PANEL } from './shell';
export { Select, type SelectOption, type SelectProps } from './Select';
export { Stat, StatRow, type StatProps } from './StatRow';
export { StatusBadge, type Status, type StatusBadgeProps } from './StatusBadge';
export {
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  type RowVariant,
  type TableProps,
  type TdProps,
  type TheadProps,
  type ThProps,
  type TrProps,
} from './Table';
export { Separator, TabPanel, Tabs, type SeparatorProps, type TabItem, type TabsProps } from './Tabs';
export {
  Checkbox,
  RadioGroup,
  Switch,
  type CheckboxProps,
  type RadioGroupProps,
  type RadioOption,
  type SwitchProps,
} from './Toggle';
