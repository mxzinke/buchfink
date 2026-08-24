/**
 * Gemeinsame Klassen für alle Overlays: Dialog, Popover, Dropdown, Tooltip.
 * Sie sind die einzigen Flächen mit Schatten (§6.6).
 *
 * Bewegung nach §7: öffnen 180 ms mit 4 px Versatz nach oben, schließen 120 ms
 * nur Deckkraft. Base UI setzt dafür `data-starting-style` und
 * `data-ending-style`, wir hängen die Zustände als Tailwind-Varianten daran.
 */
export const POPUP =
  'rounded-overlay border border-line bg-surface shadow-popover ' +
  'transition-[opacity,transform] duration-180 ease-quiet ' +
  'data-[starting-style]:opacity-0 data-[starting-style]:-translate-y-1 ' +
  'data-[ending-style]:opacity-0 data-[ending-style]:duration-120';

/** Dunkle Sprechblase für die Tooltip-Stufe (§15.2). */
export const TOOLTIP_POPUP =
  'rounded-control bg-ink px-2.5 py-1.5 text-caption text-paper max-w-[260px] ' +
  'transition-[opacity,transform] duration-180 ease-quiet ' +
  'data-[starting-style]:opacity-0 data-[starting-style]:-translate-y-1 ' +
  'data-[ending-style]:opacity-0 data-[ending-style]:duration-120';

/** Eintrag in Menü, Auswahlliste und Kontosuche. */
export const POPUP_ITEM =
  'flex items-center gap-2 px-3 h-8 text-body text-ink cursor-default select-none outline-none ' +
  'data-[highlighted]:bg-sunken data-[selected]:text-accent-text ' +
  'data-[disabled]:text-ink-faint data-[disabled]:cursor-not-allowed';

/** Abdunklung hinter einem modalen Dialog. */
export const BACKDROP =
  'fixed inset-0 z-50 bg-ink/40 transition-opacity duration-180 ease-quiet ' +
  'data-[starting-style]:opacity-0 data-[ending-style]:opacity-0 data-[ending-style]:duration-120';
