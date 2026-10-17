/** Aufnahmen der Abschlüsse: Monat, Jahr, Erklärungen. */
export default [
  {
    file: 'abschluesse-monat.png',
    go: async ({ page, nav }) => {
      await nav('Abschlüsse');
      await page.getByRole('heading', { name: 'Zeiträume festschreiben' }).waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'abschluesse-jahr.png',
    go: async ({ page, tab }) => {
      await tab('Jahr');
      await page.getByText('Weg zum Abschluss').waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    file: 'abschluesse-erklaerungen.png',
    go: async ({ page, tab }) => {
      await tab('Erklärungen');
      await page.getByText('Zuordnung der Konten').waitFor();
      await page.waitForTimeout(400);
    },
  },
];
