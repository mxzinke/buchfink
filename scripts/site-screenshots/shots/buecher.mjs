/** Aufnahmen der Bücher, je Reiter eine. */
export default [
  {
    file: 'buecher-belege.png',
    go: async ({ page, open }) => {
      await open('#/books/receipts');
      await page.getByText('BE-2026-0231').first().waitFor();
      await page.getByText('BE-2026-0231').first().click();
      await page.getByText('Beleg buchen').waitFor();
      await page.locator('img[alt="Beleg"]').waitFor();
      // Die Buchungsgruppe wählen, damit der Buchungssatz aus dem Backend
      // erscheint. Die Gruppe ist ein Suchfeld (Combobox mit Platzhalter).
      const group = page.getByPlaceholder('Gruppe suchen').first();
      await group.click();
      await group.fill('Geringwertig');
      await page.getByRole('option', { name: /Geringwertige Wirtschaftsgüter/ }).click();
      await page.waitForTimeout(700);
    },
  },
  {
    file: 'buecher-bank.png',
    go: async ({ page, tab }) => {
      await tab('Bank');
      await page.getByText('Offene Bankumsätze').waitFor();
      await page.getByText('RE-2026-0119 Wartungspauschale Q3').waitFor();
    },
  },
  {
    file: 'buecher-rechnungen.png',
    go: async ({ page, tab }) => {
      await tab('Rechnungen');
      await page.getByText('RE-2026-0119').first().waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'buecher-journal.png',
    go: async ({ page, tab }) => {
      await tab('Journal');
      await page.getByText('B-2026-0055').first().waitFor();
      await page.waitForTimeout(300);
    },
  },
  {
    file: 'buecher-konten.png',
    go: async ({ page, tab }) => {
      await tab('Konten');
      await page.getByRole('heading', { name: 'Bebuchte Konten' }).waitFor();
    },
  },
  {
    file: 'buecher-anlagen.png',
    go: async ({ page, tab }) => {
      await tab('Anlagen');
      await page.getByRole('heading', { name: 'Anlagevermögen' }).waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'buecher-bilanz.png',
    go: async ({ page, tab }) => {
      await tab('Bilanz & GuV');
      await page.getByText('Bilanz zum', { exact: false }).first().waitFor();
      await page.waitForTimeout(400);
    },
  },
];
