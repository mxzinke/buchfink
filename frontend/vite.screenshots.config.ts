import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { fileURLToPath } from 'node:url';

/**
 * Vite-Konfiguration für den Screenshot-Lauf der Projektseite
 * (`scripts/site-screenshots/shoot.mjs`).
 *
 * Gegenüber `vite.config.ts` ändern sich zwei Dinge: Das Wails-Plugin
 * entfällt, weil es keine Desktop-Laufzeit gibt, und `services/bridge.ts`
 * wird durch Beispieldaten ersetzt. Alles andere — Seiten, Bausteine, Tokens,
 * Formatierer — läuft unverändert, damit die Bilder die Oberfläche zeigen und
 * keinen Nachbau.
 */

const here = (path: string) => fileURLToPath(new URL(path, import.meta.url));

export default defineConfig({
  server: { host: '127.0.0.1', port: 9246, strictPort: true },
  resolve: {
    alias: [{ find: /^\.\/bridge$/, replacement: here('../scripts/site-screenshots/mock-bridge.ts') }],
  },
  plugins: [tailwindcss(), react()],
});
