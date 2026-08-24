/**
 * Die Schale (§16): dunkler Grund für Vollbild-Schirme vor dem Arbeitsbereich —
 * Start, Einrichtung, Wiederherstellung. Dort gibt es kein Papier, also drehen
 * sich die Rollen um: Die Primäraktion trägt die hellste Fläche, nicht die
 * dunkelste.
 *
 * Es sind Klassenbündel und keine Komponenten, weil auf diesen Schirmen jeweils
 * nur ein oder zwei Bedienelemente stehen und ein eigener Bausteinsatz für die
 * Schale mehr kosten würde, als er trägt.
 */

/** Erhöhte Fläche auf der Schale. Trägt Rand und Fläche zusammen (§6.1). */
export const SHELL_PANEL = 'rounded-overlay border border-shell-line bg-shell/90 backdrop-blur-xl';

const SHELL_BUTTON_BASE =
  'relative inline-flex items-center justify-center gap-2 h-9 px-4 rounded-control ' +
  'text-label font-semibold whitespace-nowrap select-none ' +
  'transition-colors duration-120 ease-quiet disabled:cursor-not-allowed';

export const SHELL_BUTTON = {
  primary: `${SHELL_BUTTON_BASE} bg-paper text-shell-deep hover:bg-white disabled:bg-shell-raised disabled:text-shell-text-muted`,
  secondary: `${SHELL_BUTTON_BASE} border border-shell-line text-shell-text hover:bg-shell-raised disabled:text-shell-text-muted`,
  quiet: `${SHELL_BUTTON_BASE} text-shell-text-muted hover:bg-shell-raised hover:text-shell-text disabled:text-shell-text-muted`,
};

/** Eingabefeld auf der Schale. Gleiche Geometrie wie CONTROL, dunkle Rollen. */
export const SHELL_CONTROL =
  'h-9 w-full px-3 rounded-control border border-shell-line bg-shell-deep text-body text-shell-text ' +
  'placeholder:text-shell-text-muted transition-colors duration-120 ease-quiet ' +
  'focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent/30 ' +
  'disabled:opacity-60 disabled:cursor-not-allowed';
