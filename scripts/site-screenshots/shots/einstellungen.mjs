/**
 * Aufnahmen der Einstellungen: Apps verbinden, eine App, Eingang & Ordner mit
 * den Regeln für Buchungen ohne Rückfrage, der Port, Prüfung & GoBD.
 */
export default [
  {
    file: 'einstellungen.png',
    // Auf der Projektseite als einstellungen-apps.png (website/README.md).
    site: 'einstellungen-apps.png',
    go: async ({ page, nav }) => {
      await nav('Einstellungen');
      await page.getByText('Erreichbar unter', { exact: false }).waitFor();
      await page.getByText('claude mcp add', { exact: false }).waitFor();
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
    // Eingang und Exportordner, unter „Erweitert“ die weiteren Leseordner,
    // darunter die Regeln für Buchungen ohne Rückfrage.
    file: 'einstellungen-eingang.png',
    go: async ({ page, open }) => {
      await open('#/settings/inbox');
      await page.getByRole('heading', { level: 2, name: 'Eingang & Ordner' }).waitFor();
      await page.locator('#einstellungen-inbox').getByRole('button', { name: /^Erweitert/ }).click();
      await page.getByText('/Users/anna/Dokumente/Verträge').waitFor();
      await page.getByRole('heading', { level: 2, name: 'Buchen ohne Rückfrage' }).waitFor();
      await page.waitForTimeout(700);
      await page.locator('#einstellungen-inbox').evaluate((section) => section.scrollIntoView({ block: 'start' }));
      await page.waitForTimeout(200);
    },
  },
  {
    // Der Port des Zugangs für KI-Apps.
    file: 'einstellungen-port.png',
    go: async ({ page, open }) => {
      await open('#/settings/apps');
      await page.getByRole('button', { name: 'Port ändern' }).click();
      await page.getByRole('dialog', { name: 'Port für KI-Apps' }).waitFor();
      await page.waitForTimeout(400);
    },
    after: ({ escape }) => escape(),
  },
  {
    // Der Zustandsanzeiger im Fuß der Navigation öffnet „Prüfung & GoBD";
    // die Zeile Integrität geht auf.
    file: 'einstellungen-pruefung.png',
    go: async ({ page }) => {
      await page
        .getByRole('button', { name: /Daten unverändert|Integrität verletzt/ })
        .first()
        .click();
      await page.getByText('Systemhistorie', { exact: true }).waitFor();
      await page.getByRole('button', { name: /^Integrität/ }).click();
      await page.getByText('Festgeschriebene Zeiträume').waitFor();
      await page.waitForTimeout(600);
    },
  },
];
