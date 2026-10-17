/** Aufnahmen des Unternehmens: Stammdaten und Kontakte. */
export default [
  {
    file: 'unternehmen-stammdaten.png',
    go: async ({ page, nav }) => {
      await nav('Unternehmen');
      await page.getByRole('heading', { level: 2, name: 'Stammdaten' }).waitFor();
      await page.getByText('Firmen- oder Inhabername').waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'unternehmen-kontakte.png',
    go: async ({ page, tab }) => {
      await tab('Kontakte');
      await page.getByRole('heading', { name: 'Kunden & Lieferanten' }).waitFor();
      await page.waitForTimeout(400);
    },
  },
];
