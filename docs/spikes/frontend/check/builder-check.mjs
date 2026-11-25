// Usage: node builder-check.mjs <url>: drags a connection in the React Flow builder, checks axe and keyboard focus.
import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';
const b = await chromium.launch();
const p = await (await b.newContext({ viewport: { width: 1000, height: 800 } })).newPage();
await p.goto(process.argv[2]); await p.waitForTimeout(2000);
const before = await p.getByTestId('edge-count').innerText();
const src = p.locator('.react-flow__node[data-id="cond"] .react-flow__handle.source');
const dst = p.locator('.react-flow__node[data-id="mail"] .react-flow__handle.target');
const s = await src.boundingBox(), d = await dst.boundingBox();
await p.mouse.move(s.x + s.width / 2, s.y + s.height / 2); await p.mouse.down();
await p.mouse.move(d.x + d.width / 2, d.y + d.height / 2, { steps: 10 }); await p.mouse.up();
console.log('before:', before, '| after:', await p.getByTestId('edge-count').innerText());
const axe = await new AxeBuilder({ page: p }).analyze();
console.log('axe violations:', axe.violations.map((v) => `${v.id} (${v.impact})`));
const nodeFocusable = await p.locator('.react-flow__node[data-id="cond"]').getAttribute('tabindex');
console.log('node tabindex:', nodeFocusable, '| node role:', await p.locator('.react-flow__node[data-id="cond"]').getAttribute('role'));
await p.screenshot({ path: '/tmp/builder.png' });
await b.close();
