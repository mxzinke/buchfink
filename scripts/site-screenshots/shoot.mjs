#!/usr/bin/env node
/**
 * Nimmt die Screenshots der Oberfläche auf.
 *
 * Gefahren wird die echte Oberfläche aus `frontend/`, nur die Wails-Bridge ist
 * durch `mock-bridge.ts` ersetzt. Aufruf aus dem Projektwurzelverzeichnis:
 *
 *     node scripts/site-screenshots/shoot.mjs
 *     SHOTS=buecher,einstellungen node scripts/site-screenshots/shoot.mjs
 *     SCREENSHOT_DIR=/tmp/bilder node scripts/site-screenshots/shoot.mjs
 *
 * Je Bereich der Navigation liegt eine Datei in `shots/`. `SHOTS` wählt
 * Bereiche nach ihrem Dateinamen aus, `SCREENSHOT_DIR` den Zielordner
 * (Vorgabe: website/assets/screenshots). Der Vite-Server wird selbst gestartet
 * und am Ende wieder beendet.
 */

import { copyFile, readFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { chromium } from 'playwright';
import { ORIGIN, ROOT, startVite } from './dev-server.mjs';
import uebersicht from './shots/uebersicht.mjs';
import buecher from './shots/buecher.mjs';
import abschluesse from './shots/abschluesse.mjs';
import unternehmen from './shots/unternehmen.mjs';
import einstellungen from './shots/einstellungen.mjs';
import einrichtung from './shots/einrichtung.mjs';

const HERE = dirname(fileURLToPath(import.meta.url));
const OUT = resolve(ROOT, process.env.SCREENSHOT_DIR ?? 'website/assets/screenshots');

// Die Einrichtung zuletzt: sie lädt die Seite mit eigenen Schaltern neu.
const AREAS = { uebersicht, buecher, abschluesse, unternehmen, einstellungen, einrichtung };
const ONLY = process.env.SHOTS ? process.env.SHOTS.split(',').map((s) => s.trim()) : null;

const VIEWPORT = { width: 1440, height: 900 };
const SCALE = 2;

/** Rendert den Beispielbeleg als Bild, das die Belegvorschau anzeigt. */
async function renderReceipt(context) {
  const html = await readFile(resolve(HERE, 'demo-receipt.html'), 'utf8');
  const page = await context.newPage();
  await page.setViewportSize({ width: 794, height: 1123 });
  await page.goto(`${ORIGIN}/favicon.ico`, { waitUntil: 'commit' }).catch(() => {});
  await page.setContent(html.replace('<head>', `<head><base href="${ORIGIN}/" />`), {
    waitUntil: 'networkidle',
  });
  await page.evaluate(() => document.fonts.ready);
  const buffer = await page.screenshot({ fullPage: true });
  await page.close();
  return `data:image/png;base64,${buffer.toString('base64')}`;
}

/**
 * Was eine Aufnahme braucht: die Seite und die Wege durch die Navigation.
 * `open` setzt das Ziel im Hash, wie es auch ein Verweis in der Anwendung tut.
 */
function helpers(page) {
  return {
    page,
    open: async (hash) => {
      await page.evaluate((target) => {
        window.location.hash = target;
      }, hash);
      await page.waitForTimeout(150);
    },
    nav: (label) => page.locator('nav').getByRole('button', { name: label, exact: true }).click(),
    tab: (label) => page.getByRole('tab', { name: label, exact: true }).first().click(),
    escape: async () => {
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    },
  };
}

async function main() {
  await mkdir(OUT, { recursive: true });

  const vite = await startVite();
  // Die nativen Datumsfelder folgen der Locale des Browserprozesses, nicht
  // der Seitensprache und auch nicht --lang. Ohne LANG steht im Feld
  // 08/10/2026 statt 10.08.2026.
  const browser = await chromium.launch({
    args: ['--lang=de-DE'],
    env: { ...process.env, LANG: 'de_DE.UTF-8', LANGUAGE: 'de_DE', LC_ALL: 'de_DE.UTF-8' },
  });

  let count = 0;
  try {
    const context = await browser.newContext({
      viewport: VIEWPORT,
      deviceScaleFactor: SCALE,
      locale: 'de-DE',
      timezoneId: 'Europe/Berlin',
      colorScheme: 'light',
      reducedMotion: 'reduce',
    });
    // Die Beispieldaten stehen am 25.08.2026. Fälligkeiten und der laufende
    // Monat rechnet die Oberfläche mit der Uhr des Browsers.
    await context.clock.install({ time: new Date('2026-08-25T10:12:00+02:00') });

    const receiptImage = await renderReceipt(context);
    await context.addInitScript((dataUrl) => {
      window.__receiptPreview = dataUrl;
    }, receiptImage);

    const page = await context.newPage();
    page.on('console', (msg) => {
      if (msg.type() === 'error') console.error('  [browser]', msg.text());
    });
    page.on('pageerror', (err) => console.error('  [browser]', err.message));

    await page.goto(ORIGIN, { waitUntil: 'networkidle' });
    await page.evaluate(() => document.fonts.ready);
    await page.waitForSelector('nav', { timeout: 30_000 });
    const ctx = helpers(page);

    for (const [area, shots] of Object.entries(AREAS)) {
      if (ONLY && !ONLY.includes(area)) continue;
      for (const shot of shots) {
        process.stdout.write(`→ ${shot.file}\n`);
        await shot.go(ctx);
        await page.evaluate(() => document.fonts.ready);
        await page.waitForTimeout(250);
        await page.screenshot({ path: resolve(OUT, shot.file) });
        if (shot.site) await copyFile(resolve(OUT, shot.file), resolve(OUT, shot.site));
        count++;
        if (shot.after) await shot.after(ctx);
      }
    }

    console.log(`\nFertig. ${count} Screenshots in ${OUT}`);
  } finally {
    await browser.close();
    vite.kill('SIGTERM');
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
