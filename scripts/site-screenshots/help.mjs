#!/usr/bin/env node
import assert from 'node:assert/strict';
import { chromium } from 'playwright';
import { ORIGIN, startVite } from './dev-server.mjs';

const vite = await startVite();
let browser;
try {
  browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'de-DE' });
  context.setDefaultTimeout(10000);
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  // Popover und Detaildialog tragen denselben Namen; der Dialog hat eine Überschrift.
  const popover = (name) => page.getByRole('dialog', { name, exact: true }).and(page.locator('[aria-label]'));
  const details = (name) => page.getByRole('dialog', { name, exact: true }).and(page.locator('[aria-labelledby]'));
  await page.goto(`${ORIGIN}/#/closings/declarations/e-bilanz`);
  // Die Ansicht scrollt zur E-Bilanz; erst danach steht das Zeichen still.
  const mark = page.getByRole('button', { name: 'Erklärung zur Übermittlung der E-Bilanz', exact: true });
  await mark.waitFor();
  await page.waitForTimeout(800);
  await mark.scrollIntoViewIfNeeded();
  await mark.hover();

  // Das Popover trägt den Kurztext ohne Norm und als einziges Bedienelement
  // „Mehr erfahren“. Der Detaildialog ist noch zu.
  const tip = popover('Erklärung zur Übermittlung der E-Bilanz');
  await tip.waitFor();
  assert.match(await tip.innerText(), /kein Formular/);
  assert.doesNotMatch(await tip.innerText(), /§|XBRL/);
  assert.equal(await tip.locator('a').count(), 0);
  assert.equal(await tip.getByRole('button').count(), 1);
  assert.equal(await page.getByRole('dialog').count(), 1);
  await page.mouse.move(0, 0);
  await tip.waitFor({ state: 'hidden' });

  // Der Tastaturfokus öffnet es ebenso und lässt den Fokus am Zeichen.
  await mark.focus();
  await page.keyboard.press('Shift+Tab');
  await page.keyboard.press('Escape');
  await page.keyboard.press('Tab');
  await tip.waitFor();
  assert.equal(await mark.evaluate(el => document.activeElement === el), true);
  await page.keyboard.press('Escape');
  await tip.waitFor({ state: 'hidden' });

  // Enter führt zu „Mehr erfahren“, ein zweites Enter öffnet die Details.
  await page.keyboard.press('Shift+Tab');
  await page.keyboard.press('Tab');
  await tip.waitFor();
  await page.keyboard.press('Enter');
  const more = page.getByRole('button', { name: 'Mehr erfahren: Erklärung zur Übermittlung der E-Bilanz', exact: true });
  assert.equal(await more.evaluate(el => document.activeElement === el), true);
  await page.keyboard.press('Enter');
  const dialog = details('Erklärung zur Übermittlung der E-Bilanz');
  await dialog.waitFor();
  const source = dialog.getByRole('link', { name: '§ 5b EStG', exact: true });
  assert.equal(await source.getAttribute('href'), 'https://www.gesetze-im-internet.de/estg/__5b.html');
  // Der Browser öffnet Quellen separat; die Buchhaltung bleibt geöffnet.
  await context.route('https://www.gesetze-im-internet.de/**', route => route.fulfill({ body: 'Quelle' }));
  const popupPromise = context.waitForEvent('page');
  await source.click();
  const popup = await popupPromise;
  await popup.close();
  await page.keyboard.press('Escape');
  await dialog.waitFor({ state: 'hidden' });
  // Der Fokus kehrt an das Fragezeichen zurück; „Mehr erfahren“ steht nicht mehr.
  assert.equal(await mark.evaluate(el => document.activeElement === el), true);

  // Auch ein Eingabedialog darf eine Hilfe öffnen, ohne die Eingabe zu verlieren.
  // Hier der Nachweis zu einer Pflicht aus der Übersicht, geöffnet aus ihrer
  // Seitenleiste.
  await page.goto(`${ORIGIN}/#/overview/lohnsteuer-anmeldung.2026-08`);
  await page.getByRole('button', { name: 'Als erledigt bestätigen', exact: true }).click();
  const parent = page.getByRole('dialog', { name: 'Als erledigt bestätigen', exact: true });
  await parent.waitFor();
  const ticket = parent.getByRole('textbox').first();
  await ticket.fill('et-4711');
  await parent.getByRole('button', { name: 'Erklärung zu Nachweis', exact: true }).click();
  await page.getByRole('button', { name: 'Mehr erfahren: Erklärung zu Nachweis', exact: true }).click();
  const nested = details('Erklärung zu Nachweis');
  await nested.waitFor();
  await page.keyboard.press('Escape');
  await nested.waitFor({ state: 'hidden' });
  assert.equal(await parent.isVisible(), true);
  assert.equal(await ticket.inputValue(), 'et-4711');
  // Die begonnene Eingabe geht nur nach Rückfrage verloren.
  await page.keyboard.press('Escape');
  await page.getByRole('button', { name: 'Verwerfen', exact: true }).click();
  await parent.waitFor({ state: 'hidden' });
  assert.deepEqual(errors, []);
  console.log('Hilfe: Popover bei Hover, Fokus und Enter, Escape, Dialog, Quellenlink, Fokusrückgabe und verschachtelter Dialog geprüft.');
} finally {
  await browser?.close();
  vite.kill('SIGTERM');
}
