/** Aufnahmen des Unternehmens: Stammdaten, ein Block in Bearbeitung, Kontakte. */
export default [
  {
    file: 'unternehmen-stammdaten.png',
    go: async ({ page, nav }) => {
      await nav('Unternehmen');
      await page.getByRole('heading', { level: 2, name: 'Firma & Register' }).waitFor();
      await page.getByText('ELSTER', { exact: false }).first().waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    // Der Block Steuern in der Seitenleiste.
    file: 'unternehmen-steuern.png',
    // Auf der Projektseite als unternehmen.png (website/README.md).
    site: 'unternehmen.png',
    go: async ({ page }) => {
      await page.locator('#steuern').getByRole('button', { name: 'Bearbeiten' }).click();
      await page.getByRole('dialog', { name: 'Steuern' }).waitFor();
      await page.getByText('ELSTER: 2227012345678').waitFor();
      await page.waitForTimeout(400);
    },
    after: ({ escape }) => escape(),
  },
  {
    // Gesellschafter, Darlehen und die Pflichten zum Unternehmen.
    file: 'unternehmen-gesellschafter.png',
    go: async ({ page, open }) => {
      await open('#/company/profile/gesellschafter');
      await page.getByText('Darlehensverträge').waitFor();
      await page.getByText('Transparenzregister', { exact: false }).first().waitFor();
      await page.waitForTimeout(800);
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
  {
    // Ein Kontakt in der Seitenleiste, aufgeschlagen über das Ziel.
    file: 'unternehmen-kontakt.png',
    go: async ({ page, open }) => {
      await open('#/company/contacts/1');
      await page.getByRole('dialog', { name: 'Nordwind Handels GmbH' }).waitFor();
      await page.getByText('Bestätigt am', { exact: false }).waitFor();
      await page.waitForTimeout(400);
    },
    after: ({ escape }) => escape(),
  },
];
