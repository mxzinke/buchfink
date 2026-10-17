/** Aufnahmen der Bücher: je Reiter die Liste, dazu Seitenleisten und Notfallmasken. */

/** Schließt Dialog und Seitenleiste wieder, damit die nächste Aufnahme frei beginnt. */
const closeAll = async ({ escape }) => {
  await escape();
  await escape();
};

export default [
  {
    file: 'buecher-belege.png',
    go: async ({ page, open }) => {
      await open('#/books/receipts');
      await page.getByText('BE-2026-0231').first().waitFor();
      await page.waitForTimeout(300);
    },
  },
  {
    file: 'buecher-beleg.png',
    go: async ({ page, open }) => {
      await open('#/books/receipts/1');
      await page.getByRole('heading', { name: 'BE-2026-0231' }).waitFor();
      await page.locator('img[alt="Beleg"]').waitFor();
      await page.waitForTimeout(300);
    },
  },
  {
    file: 'buecher-beleg-buchen.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Beleg buchen' }).click();
      const group = page.getByPlaceholder('Gruppe suchen').first();
      await group.waitFor();
      await group.click();
      await group.fill('Geringwertig');
      await page.getByRole('option', { name: /Geringwertige Wirtschaftsgüter/ }).click();
      await page.getByText('Netto 2.716,00 €', { exact: false }).waitFor();
      await page.waitForTimeout(300);
    },
    after: async ({ page, escape }) => {
      await escape();
      // Die begonnene Eingabe fragt nach, ob sie verworfen werden soll.
      const discard = page.getByRole('button', { name: 'Verwerfen' });
      if (await discard.isVisible().catch(() => false)) await discard.click();
      await page.waitForTimeout(200);
      await escape();
    },
  },
  {
    file: 'buecher-bank.png',
    go: async ({ page, open }) => {
      await open('#/books/bank/1');
      await page.getByText('Vorschläge').waitFor();
      await page.getByText('Betrag stimmt genau', { exact: false }).first().waitFor();
      await page.waitForTimeout(300);
    },
    after: closeAll,
  },
  {
    file: 'buecher-rechnungen.png',
    go: async ({ page, open }) => {
      await open('#/books/invoices');
      await page.getByText('RE-2026-0119').first().waitFor();
      await page.waitForTimeout(300);
    },
  },
  {
    file: 'buecher-rechnung-schreiben.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Rechnung schreiben' }).click();
      const customer = page.getByPlaceholder('Kunde suchen');
      await customer.waitFor();
      await customer.click();
      await customer.fill('Nordwind');
      await page.getByRole('option', { name: /Nordwind/ }).click();
      await page.getByLabel('Beschreibung Position 1').fill('Wartung Kundenportal, September 2026');
      await page.getByLabel('Einzelpreis Position 1').fill('2.300,00');
      await page.getByText('Brutto 2.737,00 €', { exact: false }).waitFor();
      await page.waitForTimeout(300);
    },
    after: async ({ page, escape }) => {
      await escape();
      const discard = page.getByRole('button', { name: 'Verwerfen' });
      if (await discard.isVisible().catch(() => false)) await discard.click();
      await page.waitForTimeout(200);
    },
  },
  {
    file: 'buecher-journal.png',
    go: async ({ page, open }) => {
      await open('#/books/journal/B-2026-0052');
      await page.getByRole('heading', { name: 'B-2026-0052' }).waitFor();
      await page.waitForTimeout(300);
    },
    after: closeAll,
  },
  {
    file: 'buecher-konten.png',
    go: async ({ page, open }) => {
      await open('#/books/accounts/1800');
      await page.getByText('Kontoblatt', { exact: true }).waitFor();
      await page.waitForTimeout(300);
    },
    after: closeAll,
  },
  {
    file: 'buecher-anlagen.png',
    go: async ({ page, open }) => {
      await open('#/books/assets/2');
      await page.getByText('Abschreibungsplan', { exact: true }).waitFor();
      await page.waitForTimeout(300);
    },
    after: closeAll,
  },
  {
    file: 'buecher-bilanz.png',
    go: async ({ page, open }) => {
      await open('#/books/statements');
      await page.getByText('Bilanz zum', { exact: false }).first().waitFor();
      await page.waitForTimeout(400);
    },
  },
];
