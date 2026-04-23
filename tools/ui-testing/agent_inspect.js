const { chromium } = require('@playwright/test');
const fs = require('fs');

async function run() {
  // Parse command line arguments
  const args = {};
  for (let i = 2; i < process.argv.length; i++) {
    const arg = process.argv[i];
    if (arg.startsWith('--')) {
      if (arg.includes('=')) {
        const parts = arg.split('=');
        args[parts[0]] = parts.slice(1).join('=');
      } else {
        args[arg] = process.argv[i + 1];
        i++;
      }
    }
  }

  const route = args['--route'] || '/';
  const outPrefix = args['--out-prefix'] || '/tmp/agent_inspect';
  // Use the baseURL from the playwright config
  const baseURL = 'http://localhost:34115';

  console.log(`Starting inspection...`);
  console.log(`Target: ${baseURL}${route}`);

  const browser = await chromium.launch();
  const page = await browser.newPage({
    viewport: { width: 1280, height: 800 }
  });

  try {
    // 1. Navigation with networkidle wait
    await page.goto(`${baseURL}${route}`, { waitUntil: 'networkidle' });

    // 2. Optional Interactions
    if (args['--fill']) {
      const fillArg = args['--fill'];
      const colonIndex = fillArg.indexOf(':');
      if (colonIndex !== -1) {
        const selector = fillArg.substring(0, colonIndex);
        const text = fillArg.substring(colonIndex + 1);
        console.log(`Filling selector "${selector}" with "${text}"`);
        await page.fill(selector, text);
      }
    }
    
    if (args['--click']) {
      console.log(`Clicking selector "${args['--click']}"`);
      await page.click(args['--click']);
    }
    
    if (args['--wait']) {
      const waitTime = parseInt(args['--wait'], 10);
      console.log(`Waiting for ${waitTime}ms...`);
      await page.waitForTimeout(waitTime);
    }

    // 3. Captures (Vision + Structure + Semantics)
    const pngPath = `${outPrefix}.png`;
    await page.screenshot({ path: pngPath, fullPage: true });
    
    const htmlPath = `${outPrefix}.html`;
    const html = await page.content();
    fs.writeFileSync(htmlPath, html);
    
    const a11yPath = `${outPrefix}_a11y.json`;
    try {
      const client = await page.context().newCDPSession(page);
      const { nodes } = await client.send('Accessibility.getFullAXTree');
      fs.writeFileSync(a11yPath, JSON.stringify(nodes, null, 2));
    } catch (e) {
      console.warn("Could not get accessibility tree:", e);
      fs.writeFileSync(a11yPath, JSON.stringify({ error: "Not available" }));
    }

    console.log(`Success! Generated artifacts:`);
    console.log(`  - Screenshot:     ${pngPath}`);
    console.log(`  - HTML DOM:       ${htmlPath}`);
    console.log(`  - Accessibility:  ${a11yPath}`);
  } catch (error) {
    console.error('Error during inspection:', error);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

run();
