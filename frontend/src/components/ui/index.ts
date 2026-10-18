/**
 * Die Bausteine des Design-Konzepts. Spezifikation: docs/gestaltung/komponenten.md.
 *
 * Verhalten kommt von Base UI (`@base-ui/react`): Fokusfang, Positionierung mit
 * Kollisionsprüfung, Tastaturführung, ARIA. Gestalt kommt von hier, in
 * Tailwind-Klassen aus den Tokens in index.css.
 *
 * Rein darstellende Bausteine ohne Verhalten bauen wir selbst, weil es dort
 * nichts falsch zu machen gibt: Button, Section, PageHeader, PageFrame,
 * StatRow, Table, StatusBadge, EmptyState.
 *
 * Bewusst gibt es keine `Card`. Wo eine Fläche nötig ist (§6.2), steht sie an
 * dieser Stelle im Code und nicht als Baustein, der sich unbemerkt
 * vermehrt.
 */
export { cn } from './cn';
export { BACKDROP, POPUP, POPUP_ITEM } from './popup';

export {
  Button,
  DisabledReason,
  type ButtonProps,
  type ButtonSize,
  type ButtonVariant,
  type DisabledReasonProps,
} from './Button';
export { Combobox, type ComboboxOption, type ComboboxProps } from './Combobox';
export { ConfirmDialog, Dialog, type ConfirmDialogProps, type DialogProps } from './Dialog';
export {
  DetailList,
  DetailPanel,
  type DetailListProps,
  type DetailPanelProps,
} from './DetailPanel';
export { Disclosure, type DisclosureProps } from './Disclosure';
export { EmptyState, type EmptyStateProps } from './EmptyState';
export {
  Notice,
  Progress,
  SkeletonRows,
  toast,
  type NoticeProps,
  type ProgressProps,
  type SkeletonRowsProps,
} from './Feedback';
export {
  Field,
  FieldRow,
  FieldValue,
  FormGrid,
  type FieldProps,
  type FormGridProps,
} from './Field';
export { Help, type HelpProps } from './Help';
export {
  CONTROL,
  Input,
  SearchInput,
  Textarea,
  type InputProps,
  type SearchInputProps,
  type TextareaProps,
} from './Input';
export {
  Menu,
  MenuGroup,
  MenuItem,
  MenuSeparator,
  type MenuProps,
} from './Menu';
export {
  PAGE_CONTAINER,
  PageFrame,
  useInFrame,
  type FrameTab,
  type PageFrameProps,
} from './PageFrame';
export { PageHeader, Section, type PageHeaderProps, type SectionProps } from './Section';
export { SHELL_BUTTON, SHELL_CONTROL, SHELL_PANEL } from './shell';
export { Select, type SelectOption, type SelectProps } from './Select';
export { Stat, StatRow, type StatProps } from './StatRow';
export { StatusBadge, type Status, type StatusBadgeProps } from './StatusBadge';
export {
  RowButton,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  type RowButtonProps,
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
