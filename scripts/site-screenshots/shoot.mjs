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

import { readFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { chromium } from 'playwright';
import { ORIGIN, ROOT, sleep, startVite } from './dev-server.mjs';

const HERE = dirname(fileURLToPath(import.meta.url));
const OUT = resolve(ROOT, 'website/assets/screenshots');

const VIEWPORT = { width: 1440, height: 900 };
const SCALE = 2;

// -------------------------------------------------------------------------

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
    // Vor dem Arbeitsbereich steht die Mandantenwahl, und sie hat keine
    // Navigation daneben: erst der geöffnete Mandant bringt die Seitenleiste.
    await page.getByRole('button', { name: /öffnen$/i }).first().click();
    await page.waitForSelector('nav', { timeout: 30_000 });

    const nav = (label) => page.getByRole('button', { name: label, exact: true }).first();

    const shots = [
      {
        // Die Startseite: die Aufgabenliste mit den Kennzahlen darunter. Die
        // frühere Seite „Übersicht" ist darin aufgegangen (Welle 7).
        file: 'aufgaben.png',
        go: async () => {
          await nav('Aufgaben').click();
          await page.getByRole('heading', { name: 'Aufgaben', exact: true }).waitFor();
          await page.getByText('Zuletzt erfasst').waitFor();
          await page.getByText('B-2026-0055').waitFor();
        },
      },
      {
        // Der Monatsabschluss in drei Schritten, geöffnet aus der Aufgabenliste.
        file: 'monatsabschluss.png',
        go: async () => {
          await page.getByRole('button', { name: 'Monat öffnen' }).click();
          await page.getByRole('dialog').waitFor();
          await page.getByText('Prüfbericht', { exact: true }).first().waitFor();
          await page.getByText('Festschreiben', { exact: true }).first().waitFor();
          await page.waitForTimeout(400);
        },
        after: async () => {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(300);
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
          // Seit Welle 7 wählt der Dialog den besten Vorschlag vor; ein blinder
          // Klick nähme den Haken wieder heraus. Erst warten, bis der Vorschlag
          // übernommen ist, dann nur setzen, wenn der Haken fehlt.
          await page.getByText('Übernommen', { exact: true }).waitFor();
          const match = page.getByRole('checkbox').first();
          if (!(await match.isChecked())) await match.click();
          await page.getByText('Zuordnung passt zum Kontoauszug').waitFor();
        },
        after: async () => {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(300);
        },
      },
      {
        // Das Mahnwesen wohnt als Reiter auf derselben Seite wie der Abgleich:
        // beides handelt von Zahlungen.
        file: 'mahnwesen.png',
        go: async () => {
          await page.getByRole('tab', { name: 'Mahnwesen' }).click();
          await page.getByText('Mahnvorschläge').waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'rechnungen.png',
        go: async () => {
          await nav('Rechnungen').click();
          await page.getByText('RE-2026-0119').first().waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'anzahlungen.png',
        go: async () => {
          await nav('Anzahlungen').click();
          await page.getByText('Rechnungsverbünde').waitFor();
          await page.getByText('Lagerleitstand Billstraße, Ausbaustufe 1').first().waitFor();
          await page.waitForTimeout(400);
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
          // Die Gruppe ist ein Suchfeld (Combobox mit Platzhalter), keine Liste.
          const group = page.getByPlaceholder('Gruppe suchen').first();
          await group.click();
          await group.fill('Geringwertig');
          await page.getByRole('option', { name: /Geringwertige Wirtschaftsgüter/ }).click();
          await page.waitForTimeout(700);
        },
      },
      {
        // Die Kopfdaten des Belegs (BEL-02): ohne sie nimmt das Backend die
        // Buchung nicht an.
        file: 'belege-kopfdaten.png',
        go: async () => {
          await page.getByRole('button', { name: 'Kopfdaten', exact: true }).click();
          await page.getByRole('dialog').waitFor();
          await page.getByText('Kopfdaten zu Beleg BE-2026-0231').waitFor();
          await page.waitForTimeout(400);
        },
        after: async () => {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(300);
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
        // Die Umsatzsteuer ist seit Welle 5c eine eigene Seite und kein Reiter
        // der Auswertungen mehr.
        file: 'umsatzsteuer.png',
        go: async () => {
          await nav('Umsatzsteuer').click();
          await page.getByText('Kennziffern des Vordrucks USt 1 A').waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'fristen.png',
        go: async () => {
          await nav('Steuerfristen').click();
          await page.getByText('Zeiträume festschreiben').waitFor();
          await page.getByText('Umsatzsteuer-Voranmeldung 3. Quartal 2026').first().waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'nebenpflichten.png',
        go: async () => {
          await nav('Nebenpflichten').click();
          await page.getByText('Wirtschaftsgüter im Verzeichnis').waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'guv.png',
        go: async () => {
          await nav('GuV & Bilanz').click();
          await page.getByText('Gewinn- und Verlustrechnung').waitFor();
          await page.getByText('Jahresergebnis', { exact: true }).first().waitFor();
        },
      },
      {
        file: 'jahresabschluss.png',
        go: async () => {
          await nav('Jahresabschluss').click();
          await page.getByText('Abschlussbausteine').waitFor();
          await page.getByText('Weg zum Abschluss').waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        file: 'ebilanz.png',
        go: async () => {
          await nav('E-Bilanz').click();
          await page.getByText('Zuordnung der Konten').waitFor();
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
      {
        // Die Prüfübersicht hat seit Welle 9 keinen Navigationseintrag mehr:
        // sie hängt am Zustandsanzeiger in der Fußzeile der Navigation und
        // wird über ihn geöffnet.
        file: 'sicherheit.png',
        go: async () => {
          await page.getByRole('button', { name: /Daten unverändert|Integrität verletzt/ })
            .first()
            .click();
          await page.getByText('Zustand der Kette').waitFor();
          await page.waitForTimeout(400);
        },
      },
      {
        // Das Änderungsprotokoll steht auf derselben Seite im zweiten Reiter.
        file: 'nachweise.png',
        go: async () => {
          await page.getByRole('tab', { name: 'Änderungsprotokoll' }).click();
          await page.getByRole('heading', { name: 'Änderungsprotokoll', exact: true }).waitFor();
          await page.waitForTimeout(600);
        },
      },
      {
        file: 'datenzugriff.png',
        go: async () => {
          await nav('Betriebsprüfung').click();
          await page.getByText('Datenüberlassung').first().waitFor();
          await page.waitForTimeout(400);
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
