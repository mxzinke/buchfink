/** Aufnahmen der Abschlüsse: Monat, Jahr, Erklärungen. */

/** Schaltet das Geschäftsjahr in der Kopfzeile um. */
async function year(page, value) {
  await page.getByRole('button', { name: new RegExp(`^${value}`) }).first().click();
  await page.waitForTimeout(300);
}

/** Rollt den Arbeitsbereich nach oben; ein Ziel davor hat ihn verschoben. */
async function top(page) {
  await page.evaluate(() => document.querySelectorAll('.overflow-y-auto').forEach((el) => el.scrollTo(0, 0)));
  await page.waitForTimeout(150);
}

export default [
  {
    file: 'abschluesse-monat.png',
    // Auf der Projektseite als abschluesse.png (website/README.md).
    site: 'abschluesse.png',
    go: async ({ page, nav }) => {
      await nav('Abschlüsse');
      await page.getByRole('heading', { name: 'Juli 2026' }).waitFor();
      await page.getByText('Weitere Pflichten zu diesem Monat').waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'abschluesse-monat-festschreiben.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Juli festschreiben' }).click();
      await page.getByText('Geprüfte Buchungen').waitFor();
      await page.waitForTimeout(400);
    },
    after: async ({ escape }) => escape(),
  },
  {
    file: 'abschluesse-monat-quartal.png',
    go: async ({ page, open }) => {
      await open('#/closings/month/2026-06');
      await page.getByRole('heading', { name: 'Juni 2026' }).waitFor();
      await page.getByText('Verbleibende Umsatzsteuer-Vorauszahlung').waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'abschluesse-jahr.png',
    go: async ({ page, open }) => {
      await open('#/closings/year');
      await page.getByText('Weg zum Abschluss').waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'abschluesse-jahr-baustein.png',
    go: async ({ page, open }) => {
      await open('#/closings/year/provisions');
      await page.getByText('Rückstellungsspiegel').waitFor();
      await page.waitForTimeout(500);
    },
    after: async ({ escape }) => escape(),
  },
  {
    file: 'abschluesse-jahr-2025.png',
    go: async ({ page, open }) => {
      await open('#/closings/year');
      await year(page, 2025);
      await page.getByRole('heading', { name: 'Jahresabschluss 2025' }).waitFor();
      await page.getByRole('button', { name: 'Offenlegung eintragen' }).waitFor();
      await top(page);
      await page.waitForTimeout(250);
    },
    after: async ({ page }) => year(page, 2026),
  },
  {
    file: 'abschluesse-erklaerungen.png',
    go: async ({ page, open }) => {
      await open('#/closings/declarations');
      await page.getByRole('heading', { name: 'Steuererklärungen 2026' }).waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'abschluesse-kst.png',
    // Auf der Projektseite als erklaerungen.png (website/README.md).
    site: 'erklaerungen.png',
    go: async ({ page, open }) => {
      await open('#/closings/declarations/kst');
      await page.getByRole('heading', { name: 'Zu klären' }).waitFor();
      await page.getByRole('button', { name: /Anlage GK/ }).click();
      await page.waitForTimeout(500);
    },
  },
  {
    file: 'abschluesse-kst-zeile.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: /^Jahresüberschuss \/ Jahresfehlbetrag/ }).click();
      await page.getByText('Herkunft').first().waitFor();
      await page.waitForTimeout(500);
    },
    after: async ({ escape }) => escape(),
  },
  {
    file: 'abschluesse-ust.png',
    go: async ({ page, open }) => {
      await open('#/closings/declarations/ust_jahr');
      await page.getByRole('heading', { name: /Umsatzsteuererklärung 2026/ }).waitFor();
      await page.getByRole('button', { name: /Umsatzsteuererklärung 2025 \(USt 2 A\)/ }).click();
      await page.waitForTimeout(500);
    },
  },
  {
    file: 'abschluesse-ebilanz.png',
    go: async ({ page, open }) => {
      await open('#/closings/declarations/e-bilanz');
      await page.getByRole('heading', { name: 'E-Bilanz 2026' }).waitFor();
      await page.waitForTimeout(500);
    },
  },
];
