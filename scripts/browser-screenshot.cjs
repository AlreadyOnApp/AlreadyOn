// Manual, read-only UI inspection against a developer's running instance.
const { chromium } = require('../web/node_modules/playwright');
(async () => {
  const args = process.argv.slice(2);
  const allowUntrustedLocalTLS = args.includes('--allow-untrusted-local-tls');
  const output = args.find((arg) => !arg.startsWith('--')) || '/tmp/alreadyon-guide.png';
  const url = new URL(process.env.ALREADYON_SCREENSHOT_URL || 'https://localhost:5173');
  const isLoopback = (hostname) => ['localhost', '127.0.0.1', '[::1]'].includes(hostname);
  // This exception belongs to this inspection context, never the server or host.
  // Validate before launching a browser or transmitting local fixture credentials.
  if (allowUntrustedLocalTLS && (url.protocol !== 'https:' || !isLoopback(url.hostname) || url.username || url.password)) {
    throw new Error('--allow-untrusted-local-tls requires an HTTPS loopback URL without embedded credentials.');
  }
  const browser = await chromium.launch();
  try {
    const context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
      ignoreHTTPSErrors: allowUntrustedLocalTLS,
      // Prevent service workers from bypassing the loopback request restriction.
      serviceWorkers: allowUntrustedLocalTLS ? 'block' : 'allow',
    });
    if (allowUntrustedLocalTLS) {
      console.log('Local inspection: HTTPS remains enabled; certificate verification is skipped in this context only. External requests are blocked.');
      // ignoreHTTPSErrors is context-wide. Block external requests, including
      // remote fonts, so normal inspection stays within the local preview.
      await context.route('**/*', (route) => {
        const requestURL = new URL(route.request().url());
        return isLoopback(requestURL.hostname) ? route.continue() : route.abort();
      });
    }
    const page = await context.newPage();
    await page.goto(url.href);
    const dialog = page.getByRole('dialog');
    await dialog.waitFor();
    if ((await dialog.getByRole('heading').textContent()).includes('Create')) {
      throw new Error('Create your local account in the browser first.');
    }
    const username = process.env.ALREADYON_SCREENSHOT_USER;
    const password = process.env.ALREADYON_SCREENSHOT_PASSWORD;
    if (!username || !password) throw new Error('Set the screenshot username/password environment variables.');
    await dialog.getByLabel('Username').fill(username);
    await dialog.getByLabel('Password', { exact: true }).fill(password);
    await dialog.getByRole('button', { name: /^sign in$/i }).click();
    await dialog.waitFor({ state: 'hidden' });
    await page.locator('.guide-page').waitFor();
    await page.getByText('Loading guide…', { exact: true }).waitFor({ state: 'hidden' });
    await page.locator('.guide-row').first().waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path: output, fullPage: true });
    // Do not leave an extra signed-in browser session behind.
    const logout = await page.request.post(new URL('/api/auth/logout', page.url()).href);
    if (!logout.ok()) throw new Error('Screenshot saved, but signing out failed.');
    console.log('Guide screenshot saved.');
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e.message);
  process.exitCode = 1;
});
