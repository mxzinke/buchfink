/** Aufnahmen der Einstellungen: Apps, Seitenleiste einer App, Prüfung & GoBD. */
export default [
  {
    file: 'einstellungen.png',
    go: async ({ page, nav }) => {
      await nav('Einstellungen');
      await page.getByText('Erreichbar unter', { exact: false }).waitFor();
      await page.getByRole('button', { name: 'Claude Code', exact: true }).waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    // Die Einzelheiten einer App in der Seitenleiste.
    file: 'einstellungen-app.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Claude Code', exact: true }).click();
      await page.getByRole('dialog', { name: 'Claude Code' }).waitFor();
      await page.waitForTimeout(400);
    },
    after: ({ escape }) => escape(),
  },
  {
    // Der Zustandsanzeiger im Fuß der Navigation öffnet „Prüfung & GoBD".
    file: 'einstellungen-pruefung.png',
    go: async ({ page }) => {
      await page
        .getByRole('button', { name: /Daten unverändert|Integrität verletzt/ })
        .first()
        .click();
      await page.getByText('Zustand der Kette').waitFor();
      await page.waitForTimeout(400);
    },
  },
];
