#!/usr/bin/env node
/**
 * Öffnet ein Ziel der Oberfläche mit den Beispieldaten und legt ein Bild ab.
 * Gedacht zum Nachsehen während der Arbeit an einer Ansicht:
 *
 *     node scripts/site-screenshots/peek.mjs '#/books/journal' /tmp/journal.png
 *
 * Fehler des Browsers erscheinen auf der Konsole.
 */

import { chromium } from 'playwright';
import { ORIGIN, startVite } from './dev-server.mjs';

const [hash = '#/overview', out = 'peek.png'] = process.argv.slice(2);

const vite = await startVite();
const browser = await chromium.launch({
  args: ['--lang=de-DE'],
  env: { ...process.env, LANG: 'de_DE.UTF-8', LANGUAGE: 'de_DE', LC_ALL: 'de_DE.UTF-8' },
});
try {
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    locale: 'de-DE',
    timezoneId: 'Europe/Berlin',
    reducedMotion: 'reduce',
  });
  const page = await context.newPage();
  page.on('console', (msg) => {
    if (msg.type() === 'error' || msg.type() === 'warning') console.error(`  [${msg.type()}]`, msg.text());
  });
  page.on('pageerror', (err) => console.error('  [pageerror]', err.message));
  await page.goto(`${ORIGIN}/${hash}`, { waitUntil: 'networkidle' });
  await page.waitForSelector('nav', { timeout: 30_000 });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: out, fullPage: process.env.FULL === '1' });
  console.log(`Bild: ${out}`);
} finally {
  await browser.close();
  vite.kill('SIGTERM');
}
