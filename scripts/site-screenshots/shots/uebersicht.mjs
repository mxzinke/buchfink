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
];
