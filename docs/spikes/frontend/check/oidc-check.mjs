// Usage: node oidc-check.mjs <web-url>: full login (code + PKCE) against oidc-mock, then loads the list with the token.
import { chromium } from 'playwright';
const b = await chromium.launch();
const p = await (await b.newContext()).newPage();
p.on('pageerror', (e) => console.log('pageerror', String(e).slice(0, 200)));
await p.goto(process.argv[2]);
await p.getByRole('button', { name: 'Sign in' }).click();
await p.waitForURL(/\/$/); await p.getByTestId('user').waitFor({ timeout: 8000 });
console.log('user:', await p.getByTestId('user').innerText());
await p.waitForTimeout(800);
console.log('switches loaded:', await p.locator('[role=switch]').count());
const stored = await p.evaluate(() => JSON.stringify({ ...localStorage, ...sessionStorage }));
console.log('access token persisted in browser storage:', stored.includes('spike-token'));
await b.close();
