const { chromium } = require('@playwright/test');
const fs = require('fs');

function parseArgs(argv) {
  const args = {};

  for (let i = 2; i < argv.length; i++) {
    const arg = argv[i];
    if (!arg.startsWith('--')) {
      continue;
    }

    if (arg.includes('=')) {
      const parts = arg.split('=');
      args[parts[0]] = parts.slice(1).join('=');
      continue;
    }

    const next = argv[i + 1];
    if (next && !next.startsWith('--')) {
      args[arg] = next;
      i++;
      continue;
    }

    args[arg] = 'true';
  }

  return args;
}

function shouldCaptureScreenshot(args) {
  return args['--screenshot'] === 'true';
}

function buildSuccessMessage({ stdout, pngPath, htmlPath, a11yPath, captureScreenshot }) {
  const lines = [
    'Success! Generated artifacts:',
  ];

  if (captureScreenshot) {
    lines.push(`  - Screenshot:     ${pngPath}`);
  }

  lines.push(
    `  - HTML DOM:       ${htmlPath}`,
    `  - Accessibility:  ${a11yPath}`,
  );

  return `${lines.join('\n')}\n${stdout ? `\n${stdout}` : ''}`.trim();
}

async function run() {
  // Parse command line arguments
  const args = parseArgs(process.argv);

  const route = args['--route'] || '/';
  const outPrefix = args['--out-prefix'] || '/tmp/agent_inspect';
  const captureScreenshot = shouldCaptureScreenshot(args);
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

    let pngPath = null;
    if (captureScreenshot) {
      pngPath = `${outPrefix}.png`;
      await page.screenshot({ path: pngPath, fullPage: true });
    }

    console.log(buildSuccessMessage({
      stdout: '',
      pngPath,
      htmlPath,
      a11yPath,
      captureScreenshot,
    }));
  } catch (error) {
    console.error('Error during inspection:', error);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

if (require.main === module) {
  run();
}

module.exports = {
  parseArgs,
  shouldCaptureScreenshot,
  buildSuccessMessage,
  run,
};
