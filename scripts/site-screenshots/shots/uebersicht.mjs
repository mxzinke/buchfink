/** Aufnahmen der Übersicht. */
export default [
  {
    file: 'uebersicht.png',
    go: async ({ page, nav }) => {
      await nav('Übersicht');
      await page.getByRole('heading', { level: 1, name: 'Übersicht' }).waitFor();
      await page.getByText('Zuletzt erfasst').waitFor();
      await page.getByText('B-2026-0055').first().waitFor();
    },
  },
  {
    // Der Monatsabschluss in drei Schritten, geöffnet aus der Übersicht.
    file: 'uebersicht-monatsabschluss.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Monat öffnen' }).click();
      await page.getByRole('dialog').waitFor();
      await page.getByText('Prüfbericht', { exact: true }).first().waitFor();
      await page.waitForTimeout(400);
    },
    after: ({ escape }) => escape(),
  },
];
