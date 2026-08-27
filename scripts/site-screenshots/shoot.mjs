#!/usr/bin/env node
/**
 * Nimmt die Screenshots für die Projektseite auf.
 *
 * Gefahren wird die echte Oberfläche aus `frontend/`, nur die Wails-Bridge ist
 * durch `mock-bridge.ts` ersetzt. Aufruf aus dem Projektwurzelverzeichnis:
 *
 *     node scripts/site-screenshots/shoot.mjs
 *
 * Voraussetzung ist ein `npm install` in `frontend/`. Der Vite-Server wird
 * selbst gestartet und am Ende wieder beendet.
 */

import { spawn } from 'node:child_process';
import { readFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { chromium } from 'playwright';

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(HERE, '../..');
const OUT = resolve(ROOT, 'website/assets/screenshots');
const ORIGIN = 'http://127.0.0.1:9246';

const VIEWPORT = { width: 1440, height: 900 };
const SCALE = 2;

// -------------------------------------------------------------------------

async function startVite() {
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

const sleep = (ms) => new Promise((ok) => setTimeout(ok, ms));

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

// -------------------------------------------------------------------------

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

  try {
    const context = await browser.newContext({
      viewport: VIEWPORT,
      deviceScaleFactor: SCALE,
      locale: 'de-DE',
      timezoneId: 'Europe/Berlin',
      colorScheme: 'light',
      reducedMotion: 'reduce',
    });

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

    const nav = (label) => page.getByRole('button', { name: label, exact: true }).first();

    const shots = [
      {
        file: 'uebersicht.png',
        go: async () => {
          await nav('Übersicht').click();
          await page.getByText('Buchhaltungsübersicht').waitFor();
          await page.getByText('B-2026-0055').waitFor();
        },
      },
      {
        file: 'bank.png',
        go: async () => {
          await nav('Bank & Zahlungen').click();
          await page.getByText('Offene Bankumsätze').waitFor();
          await page.getByText('RE-2026-0119 Wartungspauschale Q3').waitFor();
        },
      },
      {
        file: 'bank-zuordnung.png',
        go: async () => {
          await page.getByText('RE-2026-0119 Wartungspauschale Q3').click();
          await page.getByRole('dialog').waitFor();
          await page.getByText('Nordwind Handels GmbH').first().waitFor();
          await page.getByRole('checkbox').first().click();
          await page.getByText('Zuordnung passt zum Kontoauszug').waitFor();
        },
        after: async () => {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(300);
        },
      },
      {
        file: 'belege.png',
        go: async () => {
          await nav('Belege').click();
          await page.getByText('BE-2026-0231').first().waitFor();
          await page.getByText('BE-2026-0231').first().click();
          await page.getByText('Beleg buchen').waitFor();
          await page.locator('img[alt="Beleg"]').waitFor();
          // Die Buchungsgruppe wählen, damit der Buchungssatz aus dem Backend
          // erscheint — genau der Schritt, den die Rechnung nicht vorgibt.
          await page.getByRole('combobox').filter({ hasText: /wählen|Buchungsgruppe/i }).first()
            .click()
            .catch(async () => {
              await page.locator('[role="combobox"]').nth(2).click();
            });
          await page.getByRole('option', { name: /Geringwertige Wirtschaftsgüter/ }).click();
          await page.waitForTimeout(700);
        },
      },
      {
        file: 'journal.png',
        go: async () => {
          await nav('Journal').click();
          await page.getByText('B-2026-0055').waitFor();
          await page.getByRole('button', { name: 'Buchungssatz anzeigen' }).nth(7).click();
          await page.waitForTimeout(300);
        },
      },
      {
        file: 'guv.png',
        go: async () => {
          await nav('GuV & Bilanz').click();
          await page.getByText('Gewinn- und Verlustrechnung').waitFor();
          await page.getByText('Vorläufiges Jahresergebnis').waitFor();
        },
      },
      {
        file: 'umsatzsteuer.png',
        go: async () => {
          await page.getByRole('tab', { name: 'Umsatzsteuer' }).click();
          await page.getByText('Kennziffern der Voranmeldung').waitFor();
        },
      },
      {
        file: 'protokoll.png',
        go: async () => {
          await nav('Sicherheit & Protokoll').click();
          await page.getByText('Zustand der Kette').waitFor();
        },
      },
      {
        file: 'ebilanz.png',
        go: async () => {
          await nav('E-Bilanz').click();
          await page.getByText('Zuordnung der Standardkonten').waitFor();
          await page.getByRole('button', { name: 'Rohdaten anzeigen' }).click();
          await page.getByText('Rohdaten', { exact: true }).waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'konten.png',
        go: async () => {
          await nav('Kontenübersicht').click();
          await page.getByText('Bebuchte Konten').waitFor();
        },
      },
    ];

    for (const shot of shots) {
      process.stdout.write(`→ ${shot.file}\n`);
      await shot.go();
      await page.evaluate(() => document.fonts.ready);
      await page.waitForTimeout(250);
      await page.screenshot({ path: resolve(OUT, shot.file) });
      if (shot.after) await shot.after();
    }

    console.log(`\nFertig. ${shots.length} Screenshots in website/assets/screenshots/`);
  } finally {
    await browser.close();
    vite.kill('SIGTERM');
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
