import { clsx, type ClassValue } from 'clsx';
import { extendTailwindMerge } from 'tailwind-merge';

/**
 * Klassen zusammenführen, letzte Angabe gewinnt.
 *
 * tailwind-merge kennt die Theme-Schlüssel aus index.css nicht. Ohne die
 * Erweiterung unten passieren zwei Dinge:
 *
 *   cn('text-body', 'text-ink')          → 'text-ink'
 *     Schriftgröße und Farbe landen in derselben Gruppe und löschen sich.
 *   cn('rounded-control', 'rounded-card') → beide bleiben stehen
 *     Die eigenen Radien werden nicht als Konflikt erkannt.
 *
 * Die Listen müssen mit dem @theme-Block in index.css übereinstimmen.
 */
const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      'font-size': [{ text: ['display', 'heading', 'body', 'label', 'caption', 'overline'] }],
      rounded: [{ rounded: ['control', 'card', 'overlay'] }],
      shadow: [{ shadow: ['popover', 'dialog'] }],
      ease: [{ ease: ['quiet'] }],
    },
  },
});

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}
