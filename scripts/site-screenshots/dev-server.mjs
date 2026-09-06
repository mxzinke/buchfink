/**
 * Startet die Oberfläche mit den Beispieldaten.
 *
 * Gefahren wird die echte Anwendung aus `frontend/`, nur die Wails-Bridge ist
 * durch `mock-bridge.ts` ersetzt (siehe `frontend/vite.screenshots.config.ts`).
 * Zwei Werkzeuge brauchen denselben Server — die Screenshots (`shoot.mjs`) und
 * der Klickweg des Prüfszenarios (`four-clicks.mjs`) —, und zwei Fassungen
 * desselben Startvorgangs laufen früher oder später auseinander.
 */

import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const HERE = dirname(fileURLToPath(import.meta.url));

/** Das Wurzelverzeichnis des Projekts. */
export const ROOT = resolve(HERE, '../..');

/** Die Adresse, unter der die Oberfläche antwortet. */
export const ORIGIN = 'http://127.0.0.1:9246';

export const sleep = (ms) => new Promise((ok) => setTimeout(ok, ms));

/**
 * Startet den Vite-Server und wartet, bis er antwortet. Der Aufrufer beendet
 * ihn mit `proc.kill('SIGTERM')`.
 */
export async function startVite() {
  // Direkt die Binärdatei statt über npx: npx bleibt als Elternprozess
  // stehen, und ein SIGTERM an npx lässt den Server samt Port zurück.
  const proc = spawn(
    resolve(ROOT, 'frontend/node_modules/.bin/vite'),
    ['--config', 'vite.screenshots.config.ts', '--clearScreen', 'false'],
    { cwd: resolve(ROOT, 'frontend'), stdio: ['ignore', 'pipe', 'pipe'] },
  );
  proc.stderr.on('data', (chunk) => process.stderr.write(chunk));

  await new Promise((ok, fail) => {
    const timer = setTimeout(() => fail(new Error('Vite ist nicht gestartet.')), 90_000);
    proc.stdout.on('data', (chunk) => {
      const text = String(chunk);
      process.stdout.write(text);
      if (text.includes('ready in') || text.includes('Local:')) {
        clearTimeout(timer);
        ok();
      }
    });
    proc.on('exit', (code) => fail(new Error(`Vite beendet mit Code ${code}.`)));
  });

  // Der erste Request löst das Bündeln aus; danach antwortet der Server schnell.
  for (let i = 0; i < 40; i++) {
    try {
      const res = await fetch(ORIGIN);
      if (res.ok) break;
    } catch {
      /* noch nicht bereit */
    }
    await sleep(500);
  }
  return proc;
}
