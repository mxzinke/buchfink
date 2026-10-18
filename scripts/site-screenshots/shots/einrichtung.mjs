/**
 * Aufnahmen vor und über dem Arbeitsbereich: die Einrichtung in zwei Schritten
 * und die Zustimmung, wenn sich eine App anmeldet. Die Beispieldaten schalten
 * beides über den lokalen Speicher (`mock:setup`, `mock:consent`).
 */

async function withFlag(page, flags) {
  await page.evaluate((pairs) => {
    window.localStorage.removeItem('mock:setup');
    window.localStorage.removeItem('mock:consent');
    for (const [key, value] of pairs) window.localStorage.setItem(`mock:${key}`, value);
    window.location.hash = '#/overview';
  }, flags);
  await page.reload({ waitUntil: 'networkidle' });
  await page.evaluate(() => document.fonts.ready);
}

export default [
  {
    file: 'einrichtung-unternehmen.png',
    go: async ({ page }) => {
      await withFlag(page, [['setup', '1']]);
      await page.getByText('Ihr Unternehmen').waitFor();
      await page.getByLabel('Firma', { exact: true }).fill('Nordlicht Systeme GmbH');
      await page.getByLabel('Tag der Beurkundung', { exact: true }).fill('2026-05-04');
      await page.getByLabel('Bundesland').selectOption('HH');
      await page.waitForTimeout(600);
    },
  },
  {
    // Die Ablage in iCloud Drive: ein ruhiger Hinweis, gesperrt wird nichts.
    file: 'einrichtung-ablage.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Ordner ändern' }).click();
      await page
        .getByLabel('Ordner für Buchungsdaten und Belege')
        .fill('/Users/anna/Library/Mobile Documents/com~apple~CloudDocs/Buchfink/nordlicht');
      const warning = page.getByText('Die Datenbank verträgt keine Synchronisation', { exact: false });
      await warning.waitFor();
      await warning.evaluate((element) => element.scrollIntoView({ block: 'center' }));
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'einrichtung-verbinden.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Weiter' }).click();
      await page.getByText('Claude verbinden').waitFor();
      await page.getByText('claude mcp add', { exact: false }).waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    // Claude Code meldet sich an; Zulassen ist nach zwei Sekunden klickbar.
    file: 'zustimmung.png',
    go: async ({ page }) => {
      await withFlag(page, [['consent', '1']]);
      await page.getByRole('dialog', { name: 'Eine App möchte auf Buchfink zugreifen' }).waitFor();
      await page.waitForTimeout(2600);
    },
    after: async ({ page }) => {
      await page.getByRole('button', { name: 'Ablehnen' }).click();
      await withFlag(page, []);
    },
  },
];
