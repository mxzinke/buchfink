import { chromium } from '../site-screenshots/node_modules/playwright/index.mjs';
import { mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const out = resolve(process.env.BUCHFINK_BUGHUNT_VIEWS || '.cache/bughunt-2026-09-12/evidence/views');
const company = process.env.BUCHFINK_BUGHUNT_COMPANY || 'Fadenwerk';
const tenant = () => page.getByRole('button').filter({ hasText: company }).filter({ hasText: /öffnen/i }).first();
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1280, height: 800 }, locale: 'de-DE', timezoneId: 'Europe/Berlin' });
const errors = [];
page.on('pageerror', error => errors.push(error.message));
const result = [];
try {
  await page.goto('http://127.0.0.1:9250');
  await tenant().click();
  const pages = ['Aufgaben', 'Bank & Zahlungen', 'Belege', 'Rechnungen', 'Anzahlungen', 'Journal', 'Anlagevermögen', 'Kontakte', 'Kontenübersicht', 'GuV & Bilanz', 'Jahresabschluss', 'Abschlussbausteine', 'Umsatzsteuer', 'Nebenpflichten', 'Steuerfristen', 'E-Bilanz', 'Unterlagen', 'Datensicherung', 'Betriebsprüfung', 'Einstellungen'];
  for (const [index, name] of pages.entries()) {
    const before = errors.length;
    await page.locator('aside').getByRole('button', { name, exact: true }).click();
    await page.waitForTimeout(900);
    const text = await page.locator('main').innerText({ timeout: 3000 }).catch(() => '');
    const file = `${String(index + 1).padStart(2, '0')}`;
    await page.screenshot({ path: `${out}/${file}.png` });
    await writeFile(`${out}/${file}.txt`, text);
    result.push({ page: name, rendered: text.length > 0, errors: errors.slice(before), textLength: text.length });
    console.log(JSON.stringify(result.at(-1)));
    if (!text) {
      await page.reload();
      await tenant().click();
    }
  }
  await writeFile(`${out}/results.json`, JSON.stringify(result, null, 2));
} finally {
  await browser.close();
}
if (result.some(r => !r.rendered || r.errors.length)) process.exitCode = 1;
