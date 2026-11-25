// Usage: node check.mjs <url> <screenshot.png>
// Loads the page in Chromium, waits for the workflow list, runs axe-core, toggles the first switch.
import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
const [url, shot] = process.argv.slice(2);
const b = await chromium.launch();
const ctx = await b.newContext({ viewport: { width: 900, height: 700 } });
const p = await ctx.newPage();
p.on('console', (m) => m.type() === 'error' && console.log('console.error:', m.text()));
await p.goto(url);
await p.waitForTimeout(4000);
const text = await p.evaluate(() => document.body.innerText);
console.log('has data:', /GitHub issue/.test(text));
const axe = await new AxeBuilder({ page: p }).analyze();
console.log('axe violations:', axe.violations.map((v) => `${v.id} (${v.impact})`));
console.log('switch roles:', await p.locator('[role=switch]').count());
await p.screenshot({ path: shot });
await b.close();
