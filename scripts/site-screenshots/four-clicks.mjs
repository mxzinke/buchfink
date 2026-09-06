#!/usr/bin/env node
/**
 * Das Prüfszenario als Klickweg (GOB-02, Entscheidung 8 der Welle 7).
 *
 * Die Aufgabe: von der Bilanzposition, unter der das Bankkonto steht, zum Beleg
 * der Buchung — in höchstens vier Klicks. Sie ist der Prüfstein für die
 * Belegfunktion: „Keine Buchung ohne Beleg" ist erst dann eine Aussage über die
 * Bedienung, wenn der Weg vom Abschluss zurück zum Papier auch begehbar ist.
 *
 * Gefahren wird die echte Oberfläche aus `frontend/`; nur die Wails-Bridge ist
 * durch `mock-bridge.ts` mit den Beispieldaten ersetzt. Gezählt werden die
 * Klicks ab der Bilanz — der Weg dorthin ist die Navigation und nicht der
 * Vorgang. Der Lauf schlägt fehl, wenn ein Schritt nicht klickbar ist oder die
 * Zahl der Klicks die Grenze überschreitet.
 *
 *     node scripts/site-screenshots/four-clicks.mjs
 *
 * Voraussetzung ist ein `npm install` in `frontend/` und ein Chromium für
 * Playwright (`npx playwright install chromium`).
 */

import { chromium } from 'playwright';
import { ORIGIN, startVite } from './dev-server.mjs';

/** Die Grenze aus dem Prüfszenario. */
const MAX_CLICKS = 4;

/** Der Weg, der gemessen wird — mit den Namen aus den Beispieldaten. */
const ACCOUNT = '1800';
const POSITION = 'Umlaufvermögen';
const ENTRY = 'B-2026-0052';
const RECEIPT = 'BE-2026-0229';

async function main() {
  const vite = await startVite();
  const browser = await chromium.launch({
    args: ['--lang=de-DE'],
    env: { ...process.env, LANG: 'de_DE.UTF-8', LANGUAGE: 'de_DE', LC_ALL: 'de_DE.UTF-8' },
  });

  const steps = [];
  let failure = null;

  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
      locale: 'de-DE',
      timezoneId: 'Europe/Berlin',
      colorScheme: 'light',
      reducedMotion: 'reduce',
    });
    const page = await context.newPage();
    page.on('pageerror', (err) => console.error('  [browser]', err.message));

    /** Ein gezählter Klick. Der Zähler ist der eigentliche Gegenstand des Tests. */
    const click = async (locator, what) => {
      await locator.click();
      steps.push(what);
    };

    await page.goto(ORIGIN, { waitUntil: 'networkidle' });
    await page.waitForSelector('nav', { timeout: 30_000 });

    // Vorlauf: die Bilanz aufschlagen. Diese Klicks zählen nicht mit — das
    // Szenario beginnt bei der Bilanzposition, vor der die Anwenderin sitzt.
    await page.getByRole('button', { name: 'GuV & Bilanz', exact: true }).first().click();
    await page.getByText('Bilanz zum', { exact: false }).first().waitFor({ timeout: 20_000 });

    // 1. Die Position aufklappen — darunter stehen ihre Konten.
    await click(
      page.getByRole('row').filter({ hasText: POSITION }).first(),
      `Bilanzposition „${POSITION}" aufklappen`,
    );
    await page.getByRole('button', { name: ACCOUNT, exact: true }).first().waitFor();

    // 2. Vom Konto ins Kontoblatt.
    await click(
      page.getByRole('button', { name: ACCOUNT, exact: true }).first(),
      `Konto ${ACCOUNT} öffnen`,
    );
    await page.getByText('Kontoblatt', { exact: true }).waitFor({ timeout: 20_000 });

    // 3. Von der Zeile des Kontoblatts auf die Buchung im Journal.
    await click(
      page.getByRole('button', { name: ENTRY, exact: true }).first(),
      `Buchung ${ENTRY} öffnen`,
    );
    await page
      .getByRole('button', { name: `Beleg zur Buchung ${ENTRY} anzeigen` })
      .waitFor({ timeout: 20_000 });

    // 4. Von der Buchung auf ihren Beleg.
    await click(
      page.getByRole('button', { name: `Beleg zur Buchung ${ENTRY} anzeigen` }),
      `Beleg zur Buchung ${ENTRY} öffnen`,
    );
    await page.getByRole('heading', { name: RECEIPT }).waitFor({ timeout: 20_000 });
  } catch (err) {
    failure = err;
  } finally {
    await browser.close();
    vite.kill('SIGTERM');
  }

  steps.forEach((step, index) => console.log(`  ${index + 1}. ${step}`));

  if (failure) {
    console.error(`\nDer Weg bricht nach ${steps.length} Klicks ab: ${failure.message}`);
    process.exitCode = 1;
    return;
  }
  if (steps.length > MAX_CLICKS) {
    console.error(
      `\nDer Weg braucht ${steps.length} Klicks; das Prüfszenario lässt höchstens ${MAX_CLICKS} zu.`,
    );
    process.exitCode = 1;
    return;
  }
  console.log(
    `\nBilanz → Konto → Buchung → Beleg in ${steps.length} Klicks (erlaubt: ${MAX_CLICKS}).`,
  );
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
