import puppeteer from 'puppeteer';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const rootDir = path.resolve(__dirname, '..');
const outDir = path.join(rootDir, 'docs', 'images');
const framesDir = path.join(outDir, 'frames');

if (!fs.existsSync(framesDir)) {
  fs.mkdirSync(framesDir, { recursive: true });
}

async function run() {
  console.log('Launching headless browser via Puppeteer...');
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage'],
  });

  const page = await browser.newPage();

  // Desktop Viewport
  await page.setViewport({
    width: 1920,
    height: 1080,
    deviceScaleFactor: 2,
  });

  console.log('Navigating to http://localhost:8085/ ...');
  await page.goto('http://localhost:8085/', { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('span.font-bold');
  await new Promise(r => setTimeout(r, 1200)); // Allow animation to settle

  // 1. Dashboard Desktop Screenshot
  const dashboardPath = path.join(outDir, 'tokenprune_dashboard.png');
  await page.screenshot({ path: dashboardPath });
  console.log(`Saved: ${dashboardPath}`);

  // 2. Sessions Tab Screenshot
  console.log('Navigating to Sessions tab...');
  const sessionsBtn = await page.waitForSelector('nav button:nth-child(2)');
  await sessionsBtn.click();
  await new Promise(r => setTimeout(r, 800));

  const sessionsPath = path.join(outDir, 'tokenprune_sessions.png');
  await page.screenshot({ path: sessionsPath });
  console.log(`Saved: ${sessionsPath}`);

  // 3. Sandbox Tab Screenshot
  console.log('Navigating to Sandbox tab...');
  const sandboxBtn = await page.waitForSelector('nav button:nth-child(3)');
  await sandboxBtn.click();
  await new Promise(r => setTimeout(r, 1200)); // wait for initial prune calculation

  const sandboxPath = path.join(outDir, 'tokenprune_sandbox.png');
  await page.screenshot({ path: sandboxPath });
  console.log(`Saved: ${sandboxPath}`);

  // 4. Mobile Responsive Screenshot (iPhone 14 / modern mobile 390x844)
  console.log('Switching to mobile viewport 390x844...');
  await page.setViewport({
    width: 390,
    height: 844,
    deviceScaleFactor: 2,
    isMobile: true,
    hasTouch: true,
  });

  // Navigate back to Dashboard on mobile
  const mobileDashBtn = await page.waitForSelector('nav button:nth-child(1)');
  await mobileDashBtn.click();
  await new Promise(r => setTimeout(r, 1000));

  const mobilePath = path.join(outDir, 'tokenprune_mobile.png');
  await page.screenshot({ path: mobilePath });
  console.log(`Saved: ${mobilePath}`);

  // 5. Recording Animation Frames for Demo GIF
  console.log('Capturing interaction sequence for animated demo GIF...');
  await page.setViewport({
    width: 1440,
    height: 900,
    deviceScaleFactor: 1.5,
  });

  let frameCount = 0;
  const snap = async (count = 1, delay = 100) => {
    for (let i = 0; i < count; i++) {
      frameCount++;
      const frameNum = String(frameCount).padStart(3, '0');
      await page.screenshot({ path: path.join(framesDir, `frame_${frameNum}.png`) });
      if (delay > 0) await new Promise(r => setTimeout(r, delay));
    }
  };

  // Phase 1: Dashboard overview
  await page.goto('http://localhost:8085/', { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('span.font-bold');
  await new Promise(r => setTimeout(r, 1000));
  await snap(8, 120);

  // Hover over chart
  try {
    const chart = await page.$('svg.overflow-visible');
    if (chart) {
      const box = await chart.boundingBox();
      if (box) {
        await page.mouse.move(box.x + box.width * 0.4, box.y + box.height * 0.5);
        await snap(4, 150);
        await page.mouse.move(box.x + box.width * 0.75, box.y + box.height * 0.4);
        await snap(4, 150);
      }
    }
  } catch (e) {
    console.log('Hover skip:', e.message);
  }

  // Phase 2: Click to Sandbox
  const sbBtn = await page.waitForSelector('nav button:nth-child(3)');
  await sbBtn.click();
  await new Promise(r => setTimeout(r, 600));
  await snap(6, 120);

  // Switch Scenario preset
  const select = await page.$('select');
  if (select) {
    await select.select('error-preserve');
    await new Promise(r => setTimeout(r, 500));
    await snap(4, 150);

    // Click Run Prune
    const runBtn = await page.$('button.bg-brand-500');
    if (runBtn) {
      await runBtn.click();
      await new Promise(r => setTimeout(r, 600));
      await snap(8, 150);
    }
  }

  // Phase 3: Switch to Sessions
  const sBtn = await page.waitForSelector('nav button:nth-child(2)');
  await sBtn.click();
  await new Promise(r => setTimeout(r, 600));
  await snap(6, 120);

  // Click first session to open modal
  const firstRow = await page.$('tbody tr:first-child');
  if (firstRow) {
    await firstRow.click();
    await new Promise(r => setTimeout(r, 700));
    await snap(10, 150);

    // Switch to Side-by-Side in modal
    const sideBtns = await page.$$('button');
    for (const b of sideBtns) {
      const txt = await page.evaluate(el => el.textContent, b);
      if (txt && txt.includes('Side-by-Side')) {
        await b.click();
        await new Promise(r => setTimeout(r, 500));
        await snap(8, 150);
        break;
      }
    }
  }

  console.log(`Captured ${frameCount} total frames!`);
  await browser.close();
}

run().catch(err => {
  console.error('Capture error:', err);
  process.exit(1);
});
